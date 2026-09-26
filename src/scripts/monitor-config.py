#!/usr/bin/env python3
"""Read and persist MangoWM output configuration for the settings UI.

The helper deliberately owns only exact-name monitor rules.  On the first
write it removes older exact-name rules for the outputs being saved, then
places the authoritative rules in a marker-delimited block at the end of
Mango's config.  Catch-all and make/model rules remain untouched.
"""

from __future__ import annotations

import json
import math
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tempfile
from typing import Any


BEGIN = "# >>> ilmango-managed-monitors >>>"
END = "# <<< ilmango-managed-monitors <<<"
CONFIG = Path(os.environ.get("XDG_CONFIG_HOME", Path.home() / ".config")) / "mango" / "config.conf"


def run(command: list[str]) -> subprocess.CompletedProcess[str]:
    env = os.environ.copy()
    if not env.get("MANGO_INSTANCE_SIGNATURE"):
        runtime = Path(env.get("XDG_RUNTIME_DIR", f"/run/user/{os.getuid()}"))
        sockets = sorted(runtime.glob("mango-*.sock"), key=lambda p: p.stat().st_mtime, reverse=True)
        if sockets:
            env["MANGO_INSTANCE_SIGNATURE"] = str(sockets[0])
    return subprocess.run(command, text=True, capture_output=True, env=env, check=False)


def parse_modetest() -> dict[str, dict[str, Any]]:
    result = run(["modetest", "-c"])
    if result.returncode != 0:
        return {}

    connectors: dict[str, dict[str, Any]] = {}
    current: dict[str, Any] | None = None
    connector_re = re.compile(r"^\s*\d+\s+\d+\s+(connected|disconnected)\s+(\S+)\s+")
    mode_re = re.compile(r"^\s*#\d+\s+(\d+)x(\d+)\s+([0-9.]+)\b(.*)$")
    for line in result.stdout.splitlines():
        connector = connector_re.match(line)
        if connector:
            state, name = connector.groups()
            current = {"name": name, "connected": state == "connected", "modes": [], "vrrCapable": False}
            connectors[name] = current
            continue
        if current is None:
            continue
        mode = mode_re.match(line)
        if mode:
            width, height, refresh, tail = mode.groups()
            entry = {
                "width": int(width),
                "height": int(height),
                "refresh": round(float(refresh), 2),
                "preferred": "preferred" in tail,
            }
            key = (entry["width"], entry["height"], entry["refresh"])
            if not any((m["width"], m["height"], m["refresh"]) == key for m in current["modes"]):
                current["modes"].append(entry)
        elif "vrr_capable:" in line:
            # The actual value is printed several lines later; retaining this
            # hook makes status useful even on modetest versions that print it
            # inline.  Live Mango state still reports whether VRR is enabled.
            current["vrrCapable"] = current["vrrCapable"] or line.rstrip().endswith("1")
    return {name: item for name, item in connectors.items() if item["connected"]}


def parse_live() -> dict[str, dict[str, Any]]:
    result = run(["mmsg", "get", "all-monitors"])
    if result.returncode != 0:
        return {}
    try:
        return {item["name"]: item for item in json.loads(result.stdout).get("monitors", [])}
    except (json.JSONDecodeError, KeyError, TypeError):
        return {}


def parse_rule(line: str) -> dict[str, str] | None:
    stripped = line.strip()
    if not stripped.startswith("monitorrule="):
        return None
    values: dict[str, str] = {}
    for field in stripped.split("=", 1)[1].split(","):
        if ":" not in field:
            continue
        key, value = field.split(":", 1)
        values[key.strip()] = value.strip()
    return values


def exact_rule_name(rule: dict[str, str] | None) -> str | None:
    if not rule:
        return None
    name = rule.get("name", "")
    if name.startswith("^") and name.endswith("$"):
        return re.sub(r"\\(.)", r"\1", name[1:-1])
    if re.fullmatch(r"[A-Za-z0-9_.-]+", name):
        return name
    return None


def configured_rules() -> dict[str, dict[str, str]]:
    rules: dict[str, dict[str, str]] = {}
    try:
        lines = CONFIG.read_text(encoding="utf-8").splitlines()
    except OSError:
        return rules
    for line in lines:
        parsed = parse_rule(line)
        name = exact_rule_name(parsed)
        if name:
            rules[name] = parsed or {}
    return rules


def number(rule: dict[str, str], key: str, fallback: float) -> float:
    try:
        value = float(rule.get(key, fallback))
        return value if math.isfinite(value) else fallback
    except (TypeError, ValueError):
        return fallback


def status() -> dict[str, Any]:
    connectors = parse_modetest()
    live = parse_live()
    rules = configured_rules()
    outputs: list[dict[str, Any]] = []

    for name, connector in connectors.items():
        current = live.get(name, {})
        rule = rules.get(name, {})
        modes = connector["modes"]
        preferred = next((m for m in modes if m["preferred"]), modes[0] if modes else None)
        width = int(number(rule, "width", current.get("width", preferred["width"] if preferred else 0)))
        height = int(number(rule, "height", current.get("height", preferred["height"] if preferred else 0)))
        refresh = number(rule, "refresh", 0.0)
        if refresh <= 0:
            matching = [m for m in modes if m["width"] == width and m["height"] == height]
            refresh = matching[0]["refresh"] if matching else (preferred["refresh"] if preferred else 60.0)
        enabled = rule.get("disable", "0") not in ("1", "true", "yes")
        if name not in live and "disable" not in rule:
            enabled = False
        outputs.append({
            "name": name,
            "enabled": enabled,
            "x": int(number(rule, "x", current.get("x", 0))),
            "y": int(number(rule, "y", current.get("y", 0))),
            "width": width,
            "height": height,
            "refresh": round(refresh, 2),
            "scale": round(number(rule, "scale", current.get("scale", 1.0)), 2),
            "transform": int(number(rule, "rr", 0)),
            "vrr": rule.get("vrr", "1" if current.get("is_vrr") else "0") == "1",
            "vrrCapable": bool(connector.get("vrrCapable", False) or current.get("is_vrr")),
            "internal": name.startswith(("eDP-", "LVDS-", "DSI-")),
            "modes": modes,
        })

    outputs.sort(key=lambda item: (not item["internal"], item["x"], item["name"]))
    return {"ok": True, "outputs": outputs, "config": str(CONFIG)}


def validate(payload: dict[str, Any]) -> list[dict[str, Any]]:
    available = parse_modetest()
    submitted = payload.get("outputs")
    if not isinstance(submitted, list) or not submitted:
        raise ValueError("No connected outputs were supplied")

    cleaned: list[dict[str, Any]] = []
    for raw in submitted:
        name = str(raw.get("name", ""))
        if name not in available:
            raise ValueError(f"Output {name!r} is no longer connected")
        enabled = bool(raw.get("enabled", True))
        width, height = int(raw.get("width", 0)), int(raw.get("height", 0))
        refresh = round(float(raw.get("refresh", 0)), 2)
        supported = any(
            mode["width"] == width and mode["height"] == height
            and abs(mode["refresh"] - refresh) < 0.06
            for mode in available[name]["modes"]
        )
        if enabled and not supported:
            raise ValueError(f"Unsupported mode for {name}: {width}x{height}@{refresh}")
        scale = round(float(raw.get("scale", 1)), 2)
        transform = int(raw.get("transform", 0))
        x, y = int(raw.get("x", 0)), int(raw.get("y", 0))
        if not 0.5 <= scale <= 4.0:
            raise ValueError(f"Invalid scale for {name}")
        if transform not in range(8):
            raise ValueError(f"Invalid transform for {name}")
        cleaned.append({
            "name": name, "enabled": enabled, "width": width, "height": height,
            "refresh": refresh, "scale": scale, "transform": transform,
            "x": x, "y": y, "vrr": bool(raw.get("vrr", False)),
        })

    enabled = [item for item in cleaned if item["enabled"]]
    if not enabled:
        raise ValueError("At least one output must stay enabled")

    # Mango/XWayland requires non-negative global coordinates.  Preserve the
    # shape the user drew while moving the whole layout to the positive origin.
    min_x = min(item["x"] for item in enabled)
    min_y = min(item["y"] for item in enabled)
    for item in enabled:
        item["x"] -= min_x
        item["y"] -= min_y
    return cleaned


def format_number(value: float) -> str:
    return f"{value:.2f}".rstrip("0").rstrip(".")


def save(payload: dict[str, Any]) -> dict[str, Any]:
    outputs = validate(payload)
    CONFIG.parent.mkdir(parents=True, exist_ok=True)
    original = CONFIG.read_text(encoding="utf-8") if CONFIG.exists() else ""
    names = {item["name"] for item in outputs}

    kept: list[str] = []
    in_managed = False
    for line in original.splitlines():
        if line.strip() == BEGIN:
            in_managed = True
            continue
        if line.strip() == END:
            in_managed = False
            continue
        if in_managed:
            continue
        if exact_rule_name(parse_rule(line)) in names:
            continue
        kept.append(line)

    while kept and not kept[-1].strip():
        kept.pop()
    managed = [
        "",
        BEGIN,
        "# Generated by Settings > Monitors. Exact output rules here override older entries.",
    ]
    for item in outputs:
        escaped = re.escape(item["name"])
        fields = [f"name:^{escaped}$"]
        if not item["enabled"]:
            fields.append("disable:1")
        else:
            fields.extend([
                f"width:{item['width']}", f"height:{item['height']}",
                f"refresh:{format_number(item['refresh'])}",
                f"x:{item['x']}", f"y:{item['y']}",
                f"scale:{format_number(item['scale'])}",
                f"rr:{item['transform']}", f"vrr:{1 if item['vrr'] else 0}",
            ])
        managed.append("monitorrule=" + ",".join(fields))
    managed.append(END)
    content = "\n".join(kept + managed) + "\n"

    backup = CONFIG.with_suffix(CONFIG.suffix + ".bak-before-monitor-gui")
    if CONFIG.exists() and not backup.exists():
        shutil.copy2(CONFIG, backup)
    fd, temporary = tempfile.mkstemp(prefix=".config.conf.", dir=CONFIG.parent, text=True)
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as handle:
            handle.write(content)
            handle.flush()
            os.fsync(handle.fileno())
        os.replace(temporary, CONFIG)
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)

    reload_result = run(["mmsg", "dispatch", "reload_config"])
    if reload_result.returncode != 0:
        raise RuntimeError(reload_result.stderr.strip() or "Mango could not reload its config")
    return {"ok": True, "message": "Monitor configuration applied", "outputs": outputs}


def main() -> int:
    try:
        if len(sys.argv) < 2 or sys.argv[1] == "status":
            result = status()
        elif sys.argv[1] == "apply":
            if len(sys.argv) != 3:
                raise ValueError("apply expects one JSON argument")
            result = save(json.loads(sys.argv[2]))
        else:
            raise ValueError(f"Unknown command: {sys.argv[1]}")
        print(json.dumps(result, ensure_ascii=False, separators=(",", ":")))
        return 0
    except Exception as error:  # Keep the QML side simple and deterministic.
        print(json.dumps({"ok": False, "error": str(error)}, ensure_ascii=False, separators=(",", ":")))
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
