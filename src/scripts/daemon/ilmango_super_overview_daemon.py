#!/usr/bin/env python3

import asyncio
import os
import shutil
import subprocess
import time

from evdev import InputDevice, categorize, ecodes, list_devices

SUPER_CODES = {ecodes.KEY_LEFTMETA, ecodes.KEY_RIGHTMETA}

POINTER_BUTTON_CODES = {
    ecodes.BTN_LEFT,
    ecodes.BTN_RIGHT,
    ecodes.BTN_MIDDLE,
    ecodes.BTN_SIDE,
    ecodes.BTN_EXTRA,
    ecodes.BTN_FORWARD,
    ecodes.BTN_BACK,
}

# Debounce window (seconds) to coalesce multiple duplicate events from
# several devices (physical + virtual keyboards).
DEBOUNCE_SEC = 0.25
last_toggle_time = 0.0

super_down_global = False
interaction_since_super_down = False
tap_handled = False

# Cache of ilmango's environment so we don't hit /proc on every tap.
ILMANGO_ENV_CACHE = {}
ILMANGO_ENV_PID = None


def _find_ilmango_pid():
    """Locate the PID of the running Illogical-mango quickshell process by inspecting /proc.

    Matches both legacy ``qs -c ilmango`` invocations and the current
    path-based ``qs -p <path>`` / ``qs -n -p <path>`` form.
    """
    proc_root = "/proc"
    for entry in os.listdir(proc_root):
        if not entry.isdigit():
            continue
        pid = int(entry)
        cmdline_path = f"{proc_root}/{entry}/cmdline"
        try:
            with open(cmdline_path, "rb") as f:
                raw = f.read().decode("utf-8", errors="ignore")
        except FileNotFoundError:
            continue
        if not raw:
            continue
        args = [a for a in raw.split("\0") if a]
        if len(args) < 2:
            continue
        exe = os.path.basename(args[0])
        if exe != "qs":
            continue
        # Legacy: qs -c ilmango
        if len(args) >= 3 and args[1] == "-c" and args[2] == "ilmango":
            return pid
        # Path-based: qs ... -p <path>/shell.qml  or  qs ... -p <path>
        # where <path> ends with /ilmango or contains /ilmango/
        for i, arg in enumerate(args[1:], 1):
            if arg == "-p" and i + 1 < len(args):
                p = args[i + 1]
                if p.rstrip("/").endswith("/ilmango") or "/ilmango/" in p:
                    return pid
                break
    return None


def get_ilmango_env():
    """Get relevant environment variables from the running Illogical-mango quickshell
    session to reuse them when calling IPC.

    Caches the environment while the PID stays the same to reduce
    perceived latency for Super taps.
    """
    global ILMANGO_ENV_CACHE, ILMANGO_ENV_PID
    try:
        pid = _find_ilmango_pid()
        if pid is None:
            print("[ilmango-super-daemon] ilmango not running, cannot import env", flush=True)
            ILMANGO_ENV_CACHE = {}
            ILMANGO_ENV_PID = None
            return {}

        if ILMANGO_ENV_PID == pid and ILMANGO_ENV_CACHE:
            return ILMANGO_ENV_CACHE

        print(f"[ilmango-super-daemon] Found ilmango pid={pid}", flush=True)
        environ_path = f"/proc/{pid}/environ"
        with open(environ_path, "rb") as f:
            raw = f.read().decode("utf-8", errors="ignore")
        env_vars = {}
        for entry in raw.split("\0"):
            if not entry or "=" not in entry:
                continue
            k, v = entry.split("=", 1)
            # Only keep what matters for Wayland / Qt
            if k in (
                "WAYLAND_DISPLAY",
                "XDG_RUNTIME_DIR",
                "QT_QPA_PLATFORM",
                "NIRI_SOCKET",
            ):
                env_vars[k] = v
        ILMANGO_ENV_CACHE = env_vars
        ILMANGO_ENV_PID = pid
        print(f"[ilmango-super-daemon] Imported env from ilmango: {env_vars}", flush=True)
        return ILMANGO_ENV_CACHE
    except Exception as e:
        print(f"[ilmango-super-daemon] Error reading ilmango env: {e}", flush=True)
        return {}


def find_keyboard_devices():
    keyboards = []
    pointers = []
    for path in list_devices():
        try:
            dev = InputDevice(path)
            caps = dev.capabilities().get(ecodes.EV_KEY, [])
        except Exception as e:
            print(f"[ilmango-super-daemon] Error inspecting {path}: {e}", flush=True)
            continue

        name = (dev.name or "").lower()

        # Ignore clearly virtual devices (ydotoold, etc.) to avoid echo.
        if "ydotool" in name or "virtual" in name:
            continue

        has_super = any(code in SUPER_CODES for code in caps)
        has_pointer_button = any(code in POINTER_BUTTON_CODES for code in caps)

        if has_super:
            print(
                f"[ilmango-super-daemon] Using keyboard device {path} ({dev.name}), has_super={has_super}",
                flush=True,
            )
            keyboards.append(path)

        if has_pointer_button:
            print(
                f"[ilmango-super-daemon] Using pointer device {path} ({dev.name}), has_pointer_button={has_pointer_button}",
                flush=True,
            )
            pointers.append(path)

    if not keyboards:
        print("[ilmango-super-daemon] No suitable keyboard devices found", flush=True)

    return keyboards, pointers


async def monitor_device(path):
    global \
        super_down_global, \
        interaction_since_super_down, \
        last_toggle_time, \
        tap_handled
    super_down = False
    chord = False

    try:
        dev = InputDevice(path)
        async for event in dev.async_read_loop():
            if event.type != ecodes.EV_KEY:
                continue

            key_event = categorize(event)
            code = key_event.scancode
            value = key_event.keystate  # 1=down, 2=hold, 0=up

            if code in SUPER_CODES:
                if value == key_event.key_down:
                    super_down = True
                    chord = False
                    super_down_global = True
                    interaction_since_super_down = False
                    tap_handled = False
                elif value == key_event.key_up:
                    if (
                        super_down
                        and not chord
                        and not interaction_since_super_down
                        and not tap_handled
                    ):
                        # Tap of Super with no other keys or clicks: toggle ilmango overview
                        # with a global debounce so multiple devices don't double-trigger.
                        now = time.monotonic()
                        if now - last_toggle_time >= DEBOUNCE_SEC:
                            last_toggle_time = now
                            tap_handled = True
                            print(
                                "[ilmango-super-daemon] Super tap detected, toggling ilmango overview",
                                flush=True,
                            )
                            try:
                                ilmango_env = get_ilmango_env()
                                if not ilmango_env:
                                    print(
                                        "[ilmango-super-daemon] No ilmango env available, skipping toggle",
                                        flush=True,
                                    )
                                    super_down = False
                                    super_down_global = False
                                    interaction_since_super_down = False
                                    continue

                                env = os.environ.copy()
                                env.update(ilmango_env)

                                # Resolve the ilmango launcher for the IPC call
                                ilmango_bin = os.environ.get(
                                    "ILMANGO_LAUNCHER_PATH",
                                    shutil.which("ilmango") or "ilmango",
                                )
                                subprocess.Popen(
                                    [ilmango_bin, "overview", "toggle"],
                                    env=env,
                                    stdout=subprocess.DEVNULL,
                                    stderr=subprocess.DEVNULL,
                                )
                            except Exception as e:
                                print(
                                    f"[ilmango-super-daemon] Error running toggle command: {e}",
                                    flush=True,
                                )
                    super_down = False
                    chord = False
                    super_down_global = False
                    interaction_since_super_down = False
                continue

            # Any other key while Super is down marks this as a chord.
            if super_down and value == key_event.key_down:
                chord = True
            if super_down_global and value == key_event.key_down:
                interaction_since_super_down = True
    except asyncio.CancelledError:
        return
    except OSError:
        # Device vanished (unplug, suspend/resume). Returning lets the supervisor
        # in main() re-adopt it on the next rescan instead of killing the daemon.
        return


async def monitor_pointer_device(path):
    global interaction_since_super_down

    try:
        dev = InputDevice(path)
        async for event in dev.async_read_loop():
            if event.type != ecodes.EV_KEY:
                continue

            key_event = categorize(event)
            code = key_event.scancode
            value = key_event.keystate

            if code in POINTER_BUTTON_CODES and value == key_event.key_down:
                if super_down_global:
                    interaction_since_super_down = True
    except asyncio.CancelledError:
        return
    except OSError:
        return


async def main():
    # Retry keyboard detection until we have at least one device with Super,
    # so the service still works if it starts before the session is fully up.
    keyboard_paths = []
    while not keyboard_paths:
        keyboard_paths, _ = find_keyboard_devices()
        if keyboard_paths:
            break
        print(
            "[ilmango-super-daemon] No keyboards with Super yet, retrying in 5s",
            flush=True,
        )
        await asyncio.sleep(5)

    # Supervise: a monitor returns when its device disappears, so rescan on a
    # tick and (re)adopt anything unmonitored. Without this, unplugging a
    # keyboard left the daemon alive but deaf, and one plugged in after startup
    # was never watched at all.
    tasks = {}
    while True:
        keyboard_paths, pointer_paths = find_keyboard_devices()
        for path, monitor in [
            *((p, monitor_device) for p in keyboard_paths),
            *((p, monitor_pointer_device) for p in pointer_paths),
        ]:
            task = tasks.get(path)
            if task is None or task.done():
                tasks[path] = asyncio.create_task(monitor(path))
        await asyncio.sleep(5)


if __name__ == "__main__":
    asyncio.run(main())
