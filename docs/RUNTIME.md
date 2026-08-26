# Runtime and Boot Pipeline

What happens between "user logs in" and "shell is on screen", step by step.

## The full sequence

```
User logs in
  |
Display manager starts Niri (or Hyprland)
  |
Compositor reaches graphical-session.target
  |
systemd starts ilmango.service (wants link from compositor)
  |
ExecStart calls: /usr/bin/ilmango run --session
  |
ilmango script (bash):
  - Validates QS/Qt ABI compatibility
  - Sets QT_SCALE_FACTOR=1
  - Suppresses noisy Qt log categories
  - Bridges niri env vars to systemd session
  - Launches: qs -c ilmango
  |
Quickshell loads shell.qml
  |
ShellRoot initialization:
  1. Force-instantiate Idle and PowerProfilePersistence
  2. Load FirstRunExperience (checks if first run)
  3. Load ConflictKiller (kills conflicting trays/notification daemons)
  4. Wait for Config.ready
  |
Config.ready fires:
  1. Apply current theme (ThemeService.applyCurrentTheme)
  2. Initialize icon theme
  3. Migrate enabledPanels if needed
  4. Start shell entry timer (200ms delay for animation)
  5. Schedule deferred init (500ms for non-critical services)
  |
Panel loading:
  - Selects ShellIiPanels or ShellWafflePanels based on panelFamily
  - Each PanelLoader activates when its conditions are met
  - Immediate panels load first (bar, background, OSD)
  - Deferred panels load after GlobalStates.deferredPanelsReady
  |
Shell entry animation completes
  |
Deferred services load (GameMode, Weather, etc.)
  |
Shell fully operational
```

## Service wiring

The systemd service is the key piece. It does not use `systemctl enable` in the traditional sense because there's no `[Install]` section. Instead, Illogical-mango creates a wants link from your compositor's service:

**Niri**: `~/.config/systemd/user/niri.service.wants/ilmango.service`

**Hyprland**: `~/.config/systemd/user/wayland-wm@Hyprland.service.wants/ilmango.service`

This means Illogical-mango starts when your compositor starts and stops when it stops. It will never accidentally start under KDE or GNOME.

Managing the link:

```bash
ilmango service enable    # create the wants link
ilmango service disable   # remove it
ilmango service status    # check current state
```

## The ilmango launcher

`scripts/ilmango` is a 3600+ line bash script that wraps Quickshell. It's not the same as running `qs -c ilmango` directly:

| | `ilmango run` | `qs -c ilmango` |
|---|---|---|
| Environment setup | Sets QT_SCALE_FACTOR, suppresses warnings, bridges niri env | Raw environment |
| Output | Backgrounded, logs to journal | Foreground, direct stdout |
| Crash recovery | systemd restarts on failure (max 3 in 30s) | None |
| ABI check | Validates Quickshell/Qt compatibility | None |
| Orphan cleanup | ExecStopPost cleans stale runtime | None |

For development and debugging, `qs -c ilmango` (direct mode) is usually better because you get stdout immediately. For daily use, the systemd service handles everything.

## Environment variables

The launcher sets these before starting Quickshell:

| Variable | Value | Why |
|----------|-------|-----|
| `QT_SCALE_FACTOR` | `1` | Shell handles its own scaling in QML |
| `QT_SCALE_FACTOR_ROUNDING_POLICY` | `RoundPreferFloor` | Prevents blurry rendering with fractional compositor scaling (1.25 etc) |
| `QT_LOGGING_RULES` | (long list) | Suppress known-harmless Qt/QML warnings |

The launcher also unsets inherited DPI variables that would cause blur: `QT_WAYLAND_FORCE_DPI`, `QT_FONT_DPI`, `QT_AUTO_SCREEN_SCALE_FACTOR`, `QT_SCREEN_SCALE_FACTORS`, `GDK_SCALE`, `GDK_DPI_SCALE`.

The `--session` flag (used by systemd) also runs `ensure_systemd_graphical_env` in the background, which bridges critical Niri environment variables to the systemd user session. Without this, apps launched from the shell wouldn't get `WAYLAND_DISPLAY`, `NIRI_SOCKET`, or `ELECTRON_OZONE_PLATFORM_HINT`.

## Config loading

`Config.qml` uses Quickshell's `FileView` to read the user's JSON config file. The loading sequence:

1. FileView reads `~/.config/illogical-impulse/config.json`
2. JsonAdapter parses the content
3. Schema properties bind to parsed values (with fallbacks)
4. `Config.ready` becomes true
5. Everything that was waiting on config starts loading

If the config file doesn't exist (fresh install), Config creates it from `defaults/config.json`.

Hot-reload: if you edit config.json externally, FileView detects the change and re-parses within 50ms.

## Panel loading

Each panel is wrapped in a `PanelLoader`:

```qml
PanelLoader {
    identifier: "iiBar"
    extraCondition: !(Config.options?.bar?.vertical ?? false)
    component: Bar {}
}
```

A panel loads when all three conditions are true:

1. `Config.ready` is true
2. The identifier exists in `Config.options.enabledPanels`
3. `extraCondition` evaluates to true

Panels are split into immediate (load at first frame) and deferred (load after `deferredPanelsReady`):

**Immediate**: bar, background, notification popup, OSD. These need to be visible right away.

**Deferred**: sidebars, overview, clipboard, lock screen, cheatsheet. These load after the shell is already on screen and responsive.

## Crash recovery

The systemd service has:

- `Restart=on-failure` with `StartLimitBurst=3` and `StartLimitIntervalSec=30`
- If Illogical-mango crashes, systemd restarts it (up to 3 times in 30 seconds)
- `ExecStopPost` runs `ilmango cleanup-orphans` to clear stale Quickshell runtime entries
- Exit code 143 (SIGTERM) is treated as success, not failure

## Deferred initialization

Non-critical services load 500ms after the first frame to reduce boot contention:

- GameMode (fullscreen detection)
- WindowPreviewService (alt-tab previews)
- Weather (API polling)
- VoiceSearch (Gemini transcription)
- FontSyncService (GTK/KDE font sync)
- Hyprsunset (night light)

This keeps the initial frame fast. The bar and background appear immediately, everything else fills in shortly after.

## Debugging startup

If the shell won't start:

```bash
# Direct stdout (bypass systemd)
qs -c ilmango

# Verbose internal logging
qs -v -c ilmango

# Extra verbose
qs -vv -c ilmango

# Debug-level service logging
QS_DEBUG=1 qs -c ilmango
```

Check `ilmango logs` for recent journal output, or `ilmango doctor` for automated diagnostics.

## Performance diagnostics

`ilmango doctor --perf` prints a read-only snapshot of the running shell. It reports
process memory and threads, the observed Qt Quick RHI and render loop, DRM render
nodes, Qt Multimedia decoder ownership, open video and GIF files, mapped Niri
layer surfaces, child processes, and the `ilmango.service` cgroup.

```bash
ilmango doctor --perf
```

Use it before changing graphics environment variables or blaming one feature for
the complete service cgroup. The snapshot distinguishes Quickshell from helpers
and applications, and flags duplicate media files opened by more than one
renderer.
