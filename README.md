<p align="center">
  <img src="https://github.com/user-attachments/assets/da6beb4a-ccee-40ba-a372-5eea77b595f8" alt="iNiR" width="800">
</p>

<h1 align="center">iNiR</h1>

<p align="center">
  <b>A complete desktop shell for scrolling and tiling Wayland compositors, built on Quickshell</b>
</p>

<p align="center">
  <a href="https://github.com/snowarch/inir/releases"><img src="https://img.shields.io/badge/version-2.29.2-blue?style=flat-square" alt="Version"></a>
  <a href="https://github.com/snowarch/inir/stargazers"><img src="https://img.shields.io/github/stars/snowarch/inir?style=flat-square" alt="Stars"></a>
  <a href="https://discord.gg/pAPTfAhZUJ"><img src="https://img.shields.io/badge/Discord-join-5865F2?style=flat-square&logo=discord&logoColor=white" alt="Discord"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-GPL--3.0-green?style=flat-square" alt="License"></a>
</p>

<p align="center">
  <a href="https://github.com/snowarch/inir/wiki/INSTALL">Install</a> &bull;
  <a href="https://github.com/snowarch/inir/wiki/KEYBINDS">Keybinds</a> &bull;
  <a href="https://github.com/snowarch/inir/wiki/IPC">IPC Reference</a> &bull;
  <a href="https://discord.gg/pAPTfAhZUJ">Discord</a> &bull;
  <a href="CONTRIBUTING.md">Contributing</a>
</p>

<p align="center">
  <sub>
    <a href="README.md">English</a> · <a href="docs/readme/README.es.md">Español</a> · <a href="docs/readme/README.ru.md">Русский</a> · <a href="docs/readme/README.zh.md">中文</a> · <a href="docs/readme/README.ja.md">日本語</a> · <a href="docs/readme/README.pt.md">Português</a> · <a href="docs/readme/README.fr.md">Français</a> · <a href="docs/readme/README.de.md">Deutsch</a> · <a href="docs/readme/README.ko.md">한국어</a> · <a href="docs/readme/README.hi.md">हिन्दी</a> · <a href="docs/readme/README.ar.md">العربية</a> · <a href="docs/readme/README.it.md">Italiano</a>
  </sub>
</p>

---

## What iNiR is

A Wayland compositor draws windows. It does not give you a bar, a dock, notifications,
a launcher, a lock screen, or a settings app. iNiR is that entire layer, as one
integrated program.

It is not a theme and not a dotfiles bundle you copy into place. It is a shell: a single
Quickshell process that owns every piece of desktop UI outside your application windows,
plus a Bash/Python/Go toolchain that installs it, keeps it updated, and pushes its color
palette out to the rest of your system.

**The stack:**

```
your applications
   ↓
iNiR              bar, dock, sidebars, overview, notifications, settings, lock screen
   ↓
Quickshell        QML runtime for Wayland shells
   ↓
compositor        niri · mango · Hyprland — windows and rendering
   ↓
Wayland → GPU
```

**What distinguishes it from other Quickshell configs:**

- **Two complete panel families in one install.** Material ii (floating bar, sidebars,
  dock) and Waffle (bottom taskbar, start menu, action center). They are not themes over
  the same widgets — they are separate panel trees with their own token systems, swapped
  at runtime with <kbd>Super</kbd>+<kbd>Shift</kbd>+<kbd>W</kbd>.
- **System-wide theming, not just shell theming.** One wallpaper drives a Material You
  palette that is written out to GTK3/4, Qt, ten terminal and TUI tools, Firefox,
  Discord, Spicetify, Steam, and SDDM.
- **Configurable without editing code.** Everything is a GUI setting backed by a single
  `config.json`. You never have to touch QML to change how it looks or behaves.
- **A real install and upgrade path.** `./setup` handles dependencies and system config;
  `inir update` pulls, runs schema migrations, preserves your changes, and can roll back.

iNiR began as a fork of [end-4's illogical-impulse](https://github.com/end-4/dots-hyprland)
Hyprland dots and was rewritten around niri's scrolling workspace model.

## Compositor support

| Compositor | Status | Integration |
|---|---|---|
| [niri](https://github.com/YaLTeR/niri) | **Primary.** Developed and tested against. | Full IPC over `$NIRI_SOCKET`: workspaces, windows, outputs, keyboard layouts. iNiR manages niri's config as modular KDL files under `~/.config/niri/config.d/`, editing them surgically and never clobbering your overrides. |
| [mango](https://github.com/DreamMaoMao/mango) | **Supported.** | Full IPC over `$MANGO_INSTANCE_SIGNATURE`. dwm-style tags are mapped onto the same workspace model the rest of the shell uses, so panels, dock and overview work unchanged. |
| [Hyprland](https://hyprland.org/) | **Legacy.** Inherited from the fork, kept building, not actively tested. | Quickshell's built-in Hyprland module plus `hyprctl`. |

The running compositor is detected at startup from its environment variable; nothing
needs to be configured by hand.

---

## Screenshots

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
> disable effects, drop panels you don't use, and flatten the visual style — all from
> Settings or `config.json`.

## Features

**Two panel families**, switchable on the fly with <kbd>Super</kbd>+<kbd>Shift</kbd>+<kbd>W</kbd>:

- **Material ii** — floating bar, sidebars, dock, and 8 visual styles (Material, Cards,
  Aurora, iNiR, Angel, Regalia, ZZZ, Cookie Shapes)
- **Waffle** — Windows 11-inspired taskbar, start menu, action center, notification center

**Automatic theming.** Pick a wallpaper and the whole system follows: Material You colors
for the shell, propagated to GTK3/4, Qt, terminals, Firefox, Discord, Spicetify, Steam and
SDDM. Ships with Regalia, Gruvbox, Catppuccin and Rosé Pine presets, or build your own.

<details>
<summary><b>Full feature list</b></summary>

### Theming and appearance

- **8 visual styles**: Material (solid), Cards, Aurora (glass blur), iNiR (TUI-inspired), Angel (neo-brutalism), Regalia (black engineered chassis, warm ivory ink, restrained champagne hardware), ZZZ (poster plates), Cookie Shapes (animated shape morphing)
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

- **Workspace overview**: adapted for niri's scrolling model, with app search and calculator
- **Dashboard hub**: configurable three-column overlay with agenda, notifications, todo, notes, media and weather
- **Workspace edge strip**: hover rail with live workspace previews and drag-to-reorder
- **Window switcher**: animated Alt-Tab across all workspaces, opt-in since niri ships its own
- **Clipboard manager**: history with search and image preview
- **Region tools**: screenshots, screen recording, OCR, reverse image search
- **Cheatsheet**: keybind viewer pulled from your compositor config
- **Media controls**: full MPRIS player with multiple layout presets
- **On-screen display**: volume, brightness, and media OSD
- **Song recognition**: Shazam-style identification via SongRec
- **Voice input**: local whisper.cpp when installed, or a connected Groq, Gemini or OpenAI backend

### System

- **GUI settings**: configure everything without touching files
- **GameMode**: auto-disables effects for fullscreen apps
- **Auto-updates**: `inir update` with rollback, migrations, and user change preservation
- **Lock screen** and **session screen** (logout/reboot/shutdown/suspend)
- **Polkit agent**, **on-screen keyboard**, **autostart manager** backed by the compositor's own startup file
- **Kira**: opt-in pixel-art mascot who wanders the screen edges and reacts to what you do. Off by default; the ~32 MiB art pack is a separate download under `./setup` › Extras
- **15 languages** with auto-detection
- **Night light**: scheduled or manual
- **Weather**: Open-Meteo, supports GPS, manual coordinates, or city name
- **Battery management**: configurable thresholds, auto-suspend on critical
- **Custom event sounds** with a master volume and per-event audio files
- **Shell update checker**: notifies when new versions are available

</details>

---

## Quick Start

```bash
git clone https://github.com/snowarch/inir.git
cd inir
./setup install       # interactive, asks before each step
./setup install -y    # automatic, no questions asked
```

The installer handles dependencies, system config and theming. After install, run
`inir run` to start the shell, or log out and back in.

```bash
inir run                        # launch the shell
inir settings                   # open settings GUI
inir logs                       # check runtime logs
inir doctor                     # auto-diagnose and fix
inir update                     # pull + migrate + restart
```

Other entry points:

```bash
./setup                 # TUI menu, pick what you want
sudo make install       # system-wide instead of your home
./setup rollback        # undo the last update
```

**Distros.** Arch is the primary target and the best tested. Debian and Fedora have
automated dependency installers; anything else falls back to a guided generic path that
installs what it can and tells you the rest — the
[package list](https://github.com/snowarch/inir/wiki/PACKAGES) covers every dependency.
NixOS is supported experimentally through the flake in this repo, which exposes a package
plus NixOS and Home Manager modules; see [NixOS](docs/NIXOS.md).

---

## Keybinds

| Key | Action |
|-----|--------|
| <kbd>Super</kbd> + <kbd>Space</kbd> | Overview: search apps, navigate workspaces |
| <kbd>Super</kbd> + <kbd>V</kbd> | Clipboard history |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>S</kbd> | Screenshot a region |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>X</kbd> | OCR a region |
| <kbd>Super</kbd> + <kbd>,</kbd> | Settings |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>W</kbd> | Switch panel family |
| <kbd>Super</kbd> + <kbd>/</kbd> | Cheatsheet |

Full list: [Keybinds](https://github.com/snowarch/inir/wiki/KEYBINDS)

---

## Wallpapers

15 wallpapers ship bundled. For more, see [iNiR-Walls](https://github.com/snowarch/iNiR-Walls),
a curated collection chosen to work well with the Material You pipeline.

---

## Documentation

Everything user-facing lives in the [Wiki](https://github.com/snowarch/inir/wiki).

| Page | What's in it |
|---|---|
| [Install](https://github.com/snowarch/inir/wiki/INSTALL) | Getting it running |
| [Setup](https://github.com/snowarch/inir/wiki/SETUP) | Updates, migrations, rollback |
| [Keybinds](https://github.com/snowarch/inir/wiki/KEYBINDS) | Every shortcut |
| [IPC](https://github.com/snowarch/inir/wiki/IPC) | Targets you can bind or script |
| [Packages](https://github.com/snowarch/inir/wiki/PACKAGES) | Every dependency and why it's there |
| [Limitations](https://github.com/snowarch/inir/wiki/LIMITATIONS) | What's known broken, and workarounds |
| [Compositors](docs/COMPOSITORS.md) | How niri, mango and Hyprland are integrated |
| [Architecture](ARCHITECTURE.md) | How the code is put together |

---

## Troubleshooting

```bash
inir logs                       # check recent runtime logs
inir restart                    # restart the active runtime
inir repair                     # doctor + restart + filtered log check
inir doctor                     # auto-diagnose and fix common problems
./setup rollback                # undo the last update
```

Check [Limitations](https://github.com/snowarch/inir/wiki/LIMITATIONS) before opening an
issue. Discord is usually faster for questions.

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, code patterns, and pull
request guidelines.

---

## Credits

- [**end-4**](https://github.com/end-4/dots-hyprland): illogical-impulse, the Hyprland dots iNiR forked from
- [**Gakuseei**](https://github.com/Gakuseei): [Ricelin](https://github.com/Gakuseei/Ricelin), where the pill bar and the washi and flame look come from
- [**Quickshell**](https://quickshell.outfoxxed.me/): the framework this runs on
- [**niri**](https://github.com/YaLTeR/niri) and [**mango**](https://github.com/DreamMaoMao/mango): the compositors it targets

GPL-3.0, same as end-4's dots. Copyright (C) 2025-2026 snowarch.

---

<p align="center">
  <a href="https://github.com/snowarch/inir/graphs/contributors">Contributors</a> &bull;
  <a href="CHANGELOG.md">Changelog</a> &bull;
  <a href="LICENSE">GPL-3.0 License</a>
</p>
