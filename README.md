<h1 align="center">Illogical-mango</h1>

<p align="center">
  <b>A complete desktop shell for MangoWM, built on Quickshell</b>
</p>

<p align="center">
  <sub>
    <a href="README.md">English</a> · <a href="docs/readme/README.ru.md">Русский</a> · <a href="docs/readme/README.es.md">Español</a> · <a href="docs/readme/README.zh.md">中文</a> · <a href="docs/readme/README.ja.md">日本語</a> · <a href="docs/readme/README.pt.md">Português</a> · <a href="docs/readme/README.fr.md">Français</a> · <a href="docs/readme/README.de.md">Deutsch</a> · <a href="docs/readme/README.ko.md">한국어</a> · <a href="docs/readme/README.hi.md">हिन्दी</a> · <a href="docs/readme/README.ar.md">العربية</a> · <a href="docs/readme/README.it.md">Italiano</a>
  </sub>
</p>

---

## This port was written by an AI. Completely. This README too, about 90% of it

The project is a joke. Nobody was trying hard here.

The MangoWM port - `services/MangoService.qml`, `services/deferred/MangoKeybinds.qml`,
the compositor detection rework in `services/CompositorService.qml`, the installer and
doctor changes - was written from start to finish through Claude.
Not "with the help of". Written by it.

This is stated at the very top so you don't find it out later, from a diff or from a bug.
It is not an achievement and isn't presented as one. Basically I made this port for
myself, for fun. Factor that in.

The shell under the port layer is snowarch's iNiR, written (I hope) by a human.

---

## What this is

Honestly, if you have ever set up a bare Wayland compositor yourself, nobody needs to
explain to you why it needs a shell. But I am obliged to explain how it works.

```
your applications
   ↓
Illogical-mango   bar, dock, sidebars, overview, notifications, settings, lock screen
   ↓
Quickshell        QML runtime for Wayland shells
   ↓
MangoWM           windows and rendering
   ↓
Wayland → GPU
```

**What distinguishes it from other Quickshell configs:**

- **Two complete panel families in one install.** Material ii (floating bar, sidebars,
  dock) and Waffle (bottom taskbar, start menu, action center). These are not themes over
  the same widgets — they are separate panel trees with their own token systems, swapped
  at runtime with <kbd>Super</kbd>+<kbd>Shift</kbd>+<kbd>W</kbd>.
- **System-wide theming, not just shell theming.** One wallpaper drives a Material You
  palette that is written out to GTK3/4, Qt, ten terminal and TUI tools, Firefox,
  Discord, Spicetify, Steam and SDDM.
- **Configurable without editing code.** Everything is a GUI setting on top of a single
  `config.json`. You never need to touch QML to change how it looks or behaves.
- **A real install and upgrade path.** `./setup` takes care of dependencies and system
  config; `ilmango update` pulls, runs schema migrations, preserves your changes and can
  roll back.

**Lineage.** [end-4's illogical-impulse](https://github.com/end-4/dots-hyprland) (Hyprland
dots) → [snowarch's iNiR](https://github.com/snowarch/iNiR) (rewritten for niri) → this,
ported to MangoWM. The CLI, the config paths and the internals are called `ilmango`.
Installs from the iNiR era are carried over by migration 037, which leaves symlinks at the
old paths so existing keybinds and scripts keep working.
Why not fork end-4 directly? The logic is simple - a project that has been ported once is
easier to port again.
As an analogy, take Void Linux. Install systemd on it and it will run just fine.
Take Arch Linux and tear systemd out, and you will have to change almost the entire package base.


## Compositor

Built for [MangoWM](https://github.com/DreamMaoMao/mango) and tested only on it.

The shell talks to mango over its IPC socket at `$MANGO_INSTANCE_SIGNATURE`, which sends a
full session snapshot on every change. mango is dwm-style — tags, not a workspace list —
so `MangoService` maps `(monitor, tag index)` pairs onto the same workspace model the bar,
dock, overview and workspace strip already expect, and those modules work unchanged.

Configuration is deliberately non-destructive. mango reads exactly one file
(`~/.config/mango/config.conf`) and merges nothing, so the installer never overwrites your
compositor config. It puts the shell's keybinds and autostart into
`~/.config/mango/ilmango.conf` and appends a single `source-optional=` line pointing at it,
without touching your window management. Autostart is an `exec-once=ilmango run --daemon`
line in that file, not a systemd unit.

> [!NOTE]
> **niri and Hyprland code is still in the tree.** `NiriService.qml`, `HyprlandData.qml`
> and the `isNiri` / `isHyprland` branches survive from upstream and still compile. They
> are inherited, not supported: nothing here is tested against those compositors and
> nothing is maintained for them. If you want niri, take
> [the original iNiR](https://github.com/snowarch/iNiR).

---

## Screenshots

Both panel families, carried over from upstream unchanged.

<details open>
<summary><b>Material ii</b>: floating bar, sidebars, Material Design aesthetic</summary>

| | |
|:---:|:---:|
| ![](https://github.com/user-attachments/assets/1fe258bc-8aec-4fd9-8574-d9d7472c3cc8) | ![](https://github.com/user-attachments/assets/3ce2055b-648c-45a1-9d09-705c1b4a03b7) |
| ![](https://github.com/user-attachments/assets/ea2311dc-769e-44dc-a46d-37cf8807d2cc) | ![](https://github.com/user-attachments/assets/ba866063-b26a-47cb-83c8-d77bd033bf8b) |
| ![](https://github.com/user-attachments/assets/88e76566-061b-4f8c-a9a8-53c157950138) | |

</details>

<details>
<summary><b>Waffle</b>: bottom taskbar, action center, Windows 11 vibes</summary>

| | |
|:---:|:---:|
| ![](https://github.com/user-attachments/assets/5c5996e7-90eb-4789-9921-0d5fe5283fa3) | ![](https://github.com/user-attachments/assets/fadf9562-751e-4138-a3a1-b87b31114d44) |

</details>

---

> [!WARNING]
> The default configuration targets reasonably modern hardware. On low-spec machines,
> disable effects, drop panels you don't need and flatten the visual style — all of that
> is done from Settings or through `config.json`.

## Features

**Two panel families**, switchable on the fly with <kbd>Super</kbd>+<kbd>Shift</kbd>+<kbd>W</kbd>:

- **Material ii** — floating bar, sidebars, dock and 8 visual styles (Material, Cards,
  Aurora, Illogical-mango, Angel, Regalia, ZZZ, Cookie Shapes)
- **Waffle** — Windows 11-style taskbar, start menu, action center, notification center

**Automatic theming.** Pick a wallpaper and the whole system follows: Material You colors
for the shell are propagated to GTK3/4, Qt, terminals, Firefox, Discord, Spicetify, Steam
and SDDM. Ships with Regalia, Gruvbox, Catppuccin and Rosé Pine presets, or build your own.

<details>
<summary><b>Full feature list</b></summary>

### Theming and appearance

- **8 visual styles**: Material (solid), Cards, Aurora (glass blur), Illogical-mango (TUI-inspired), Angel (neo-brutalism), Regalia (black engineered chassis, warm ivory ink, restrained champagne hardware), ZZZ (poster plates), Cookie Shapes (animated shape morphing)
- **Dynamic wallpaper colors** via Material You, propagated system-wide
- **10 terminal and TUI tools auto-themed**: foot, kitty, alacritty, ghostty, wezterm, starship, fuzzel, btop, lazygit, yazi
- **App theming**: GTK3/4, Qt (via plasma-integration and darkly), Firefox (MaterialFox), Discord/Vesktop (System24), Zed, Spicetify, Steam, SDDM
- **Theme presets**: Regalia, Regalia Ivory, Gruvbox, Catppuccin, Rosé Pine, and custom
- **Video wallpapers**: mp4/webm/gif with optional blur, or a frozen first frame for performance
- **Desktop widgets**: clock (multiple styles), weather, media controls on the wallpaper layer

### Bar

- **6 bar styles**: classic, islands, scenic, frame, Material 3 capsules, and pill
- **Pill bar**: a morphing centre island that opens on hover into workspaces, launcher, mixer, media, calendar and a screen recorder
- **Modular layout** with a drag editor in Settings, so any module can go anywhere
- **Vertical bar** for screen-edge layouts

### Sidebars and widgets (Material ii)

Left sidebar (app drawer):
- **AI Chat**: live model catalogs across Ollama, LM Studio, OpenRouter, Gemini, Groq, Mistral, Cerebras, Anthropic, OpenAI and OpenCode
- **YT Music**: cookie-less InnerTube player with search, queue, radio and synced lyrics
- **Wallhaven browser**: search and apply wallpapers directly
- **Anime tracker**: AniList integration with schedule view
- **Translator**: via Gemini or translate-shell
- **Draggable widgets**: crypto, media player, quick notes, status rings, weekly calendar

Right sidebar:
- **Calendar** with event integration
- **Notification center**
- **Quick toggles**: WiFi, Bluetooth, night light, DND, power profiles, WARP VPN, EasyEffects
- **Volume mixer** with per-app control
- **Bluetooth and WiFi** device management
- **Pomodoro timer**, **todo list**, **calculator**, **notepad**
- **System monitor**: CPU, RAM, temperature

### Tools

- **Workspace overview**: app search and calculator, mapped onto mango's tag model
- **Dashboard hub**: configurable three-column overlay with agenda, notifications, todo, notes, media and weather
- **Workspace edge strip**: hover rail with live workspace previews and drag-to-reorder
- **Window switcher**: animated Alt-Tab across all workspaces, opt-in
- **Clipboard manager**: history with search and image preview
- **Region tools**: screenshots, screen recording, OCR, reverse image search
- **Cheatsheet**: keybind viewer pulled from your mango config
- **Media controls**: full MPRIS player with multiple layout presets
- **On-screen display**: volume, brightness, and media OSD
- **Song recognition**: Shazam-style identification via SongRec
- **Voice input**: local whisper.cpp when installed, or a connected Groq, Gemini or OpenAI backend

### System

- **GUI settings**: configure everything without touching files
- **GameMode**: auto-disables effects for fullscreen apps
- **Auto-updates**: `ilmango update` with rollback, migrations, and user change preservation
- **Lock screen** and **session screen** (logout/reboot/shutdown/suspend)
- **Polkit agent**, **on-screen keyboard**, **autostart manager** backed by the `exec-once` line in the mango config
- **Kira**: opt-in pixel-art mascot who wanders the screen edges and reacts to what you do. Off by default; the ~32 MiB art pack is a separate download under `./setup` › Extras
- **15 languages** with auto-detection
- **Night light**: scheduled or manual
- **Weather**: Open-Meteo, supports GPS, manual coordinates, or city name
- **Battery management**: configurable thresholds, auto-suspend on critical
- **Custom event sounds** with a master volume and per-event audio files

</details>

---

## Quick Start (the installer will be different in the future)

```bash
git clone https://github.com/ItzWithTails/illogical-mango.git
cd Illogical-mango
./setup install       # interactive, asks before each step
./setup install -y    # automatic, no questions asked
```

The installer takes care of dependencies, system config and theming. It writes the shell's
keybinds to `~/.config/mango/ilmango.conf` and hooks them into your existing mango config
without touching your window management. Restart mango or run
`mmsg dispatch reload_config`.

```bash
ilmango run                        # launch the shell
ilmango settings                   # open settings GUI
ilmango logs                       # check the logs
ilmango doctor                     # auto-diagnose and fix
ilmango update                     # pull + migrate + restart
```

Other entry points:

```bash
./setup                 # TUI menu, pick what you want
./setup install --skip-mango    # don't touch the mango config at all
sudo make install       # system-wide instead of your home
./setup rollback        # undo the last update
```

**Distros.** Arch is the primary target and the best tested. Debian and Fedora do have a
port, of course... at your own risk, there has been no testing on them.

---

## Keybinds

Installed from `defaults/mango/config.conf`:

| Key | Action |
|-----|--------|
| <kbd>Super</kbd> + <kbd>Space</kbd> | Overview: app search, tag navigation |
| <kbd>Super</kbd> + <kbd>V</kbd> | Clipboard history |
| <kbd>Super</kbd> + <kbd>P</kbd> | Left sidebar |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>N</kbd> | Right sidebar |
| <kbd>Super</kbd> + <kbd>D</kbd> | Dashboard |
| <kbd>Super</kbd> + <kbd>,</kbd> | Settings |
| <kbd>Super</kbd> + <kbd>/</kbd> | Cheatsheet |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>W</kbd> | Switch panel family |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>S</kbd> | Screenshot a region |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>X</kbd> | OCR a region |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>R</kbd> | Record a region |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>C</kbd> | Reverse image search a region |
| <kbd>Super</kbd> + <kbd>L</kbd> | Lock |
| <kbd>Ctrl</kbd> + <kbd>Alt</kbd> + <kbd>Delete</kbd> | Session screen |

Window management binds are yours — the shell does not define them. Full reference:
[Keybinds](docs/KEYBINDS.md).

---

## Wallpapers

15 wallpapers ship bundled. For more, see [iNiR-Walls](https://github.com/snowarch/iNiR-Walls),
a collection that works well with the Material You pipeline.

---

## Documentation (for niri, not mango)

| Page | What's in it |
|---|---|
| [Install](docs/INSTALL.md) | Getting it running |
| [Setup](docs/SETUP.md) | Updates, migrations, rollback |
| [Keybinds](docs/KEYBINDS.md) | Every shortcut |
| [IPC](docs/IPC.md) | Targets you can bind to a key or call from a script |
| [Packages](docs/PACKAGES.md) | Every dependency and why it's there |
| [Limitations](docs/LIMITATIONS.md) | What's known broken, and workarounds |
| [Compositors](docs/COMPOSITORS.md) | How the compositor integration works |
| [Architecture](ARCHITECTURE.md) | How the code is put together |

Most of `docs/` was inherited from upstream and still describes niri in places. Where the
docs and this README disagree about which compositor is supported, this README is correct.

---

## Troubleshooting

```bash
ilmango logs                       # recent runtime logs
ilmango restart                    # restart the active runtime
ilmango repair                     # doctor + restart + filtered log check
ilmango doctor                     # auto-diagnose and fix common problems
./setup rollback                # undo the last update
claude "help me please"         # if you'd rather not work it out yourself. come on, it has to earn its $20 somehow
```

Have a look at [Limitations](docs/LIMITATIONS.md) for a laugh.

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) — development setup, code patterns, and pull
request guidelines.


---

## Credits

- [**snowarch**](https://github.com/snowarch/iNiR): iNiR, the shell that is ported here
- [**end-4**](https://github.com/end-4/dots-hyprland): illogical-impulse, which iNiR forked from
- [**Gakuseei**](https://github.com/Gakuseei): [Ricelin](https://github.com/Gakuseei/Ricelin), where the pill bar and the washi and flame look come from
- [**Quickshell**](https://quickshell.outfoxxed.me/): the framework this runs on
- [**MangoWM**](https://github.com/DreamMaoMao/mango): the compositor this is built for
- **Claude** (Anthropic): wrote the MangoWM port, as stated at the very top

GPL-3.0, same as end-4's dots. Upstream copyright (C) 2025-2026 snowarch.
