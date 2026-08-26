# NixOS

> Experimental. The Arch installer is still the primary supported path.

Illogical-mango provides a flake with:

| Output | Purpose |
|---|---|
| `packages.<system>.default` | Packaged Illogical-mango runtime and `ilmango` launcher |
| `nixosModules.ilmango` | NixOS module for system package + user service |
| `homeModules.ilmango` | Home Manager module for user package + user service |

The module does not run `./setup install` or `./setup update`. Nix owns the installed files, and Illogical-mango runs from the package store path.

The package and modules are ordinary Nix expressions under `nix/`. Flakes are only one entrypoint, so traditional Nix configurations can import them directly. Both entrypoints use the same `package.nix`, NixOS module, and Home Manager module rather than maintaining separate implementations.

## Without flakes

Point `ilmangoSrc` at a local checkout or a source pinned with your preferred Nix fetcher:

```nix
{ pkgs, ... }:
let
  ilmangoSrc = /path/to/ilmango;
in
{
  imports = [
    (import (ilmangoSrc + "/nix/nixos-module.nix"))
  ];

  programs.ilmango = {
    enable = true;
    package = pkgs.callPackage (ilmangoSrc + "/nix/package.nix") { inherit pkgs; };
    service.compositor = "niri";
  };
}
```

For Home Manager, import `nix/home-module.nix` instead. The package expression accepts the consumer's `pkgs` set explicitly, so traditional configurations can choose or pin nixpkgs without converting the project to a flake. Both modules use that same package expression by default unless `programs.ilmango.package` is overridden.

## With niri-flake

Add both flakes:

```nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    niri.url = "github:sodiboo/niri-flake";
    ilmango.url = "github:ItzWithTails/illogical-mango";
  };
}
```

Then import both modules in your NixOS configuration:

```nix
{ config, inputs, ... }: {
  imports = [
    inputs.niri.nixosModules.niri
    inputs.ilmango.nixosModules.ilmango
  ];

  programs.niri.enable = true;

  programs.ilmango = {
    enable = true;
    service.compositor = "niri";
    extraPackages = [ config.programs.niri.package ];
  };
}
```

`programs.ilmango.service.compositor = "niri"` creates the user unit wiring under `niri.service.wants/ilmango.service`. It does not wire Illogical-mango to `graphical-session.target`, so it will not auto-start under KDE, GNOME, or other desktop sessions.

`extraPackages = [ config.programs.niri.package ];` puts the same `niri` client binary used by your compositor on Illogical-mango's runtime `PATH`, so features that call `niri msg` use the matching package.

For useful default shortcuts, merge Illogical-mango actions into `programs.niri.settings.binds`:

```nix
{
  programs.niri.settings.binds = {
    "Mod+Space" = {
      repeat = false;
      action.spawn = [ "ilmango" "overview" "toggle" ];
    };

    "Mod+V".action.spawn = [ "ilmango" "clipboard" "toggle" ];
    "Mod+Comma".action.spawn = [ "ilmango" "settings" ];
    "Mod+Slash".action.spawn = [ "ilmango" "cheatsheet" "toggle" ];
    "Mod+Shift+W".action.spawn = [ "ilmango" "panelFamily" "cycle" ];

    "Mod+Alt+L" = {
      allow-when-locked = true;
      action.spawn = [ "ilmango" "lock" "activate" ];
    };

    "Mod+Shift+S".action.spawn = [ "ilmango" "region" "screenshot" ];
    "Mod+Shift+X".action.spawn = [ "ilmango" "region" "ocr" ];
    "Mod+Shift+A".action.spawn = [ "ilmango" "region" "search" ];
  };
}
```

## Home Manager

If you manage your user session with Home Manager, import the Home Manager module instead:

```nix
{ inputs, ... }: {
  imports = [
    inputs.ilmango.homeModules.ilmango
  ];

  programs.ilmango = {
    enable = true;
    service.compositor = "niri";
  };
}
```

The Home Manager module can also expose the packaged runtime at:

```text
~/.config/quickshell/ilmango
```

That symlink keeps tools that expect the traditional config path working, but it is opt-in because it will conflict with an existing repo checkout at the same path. Enable it with:

```nix
programs.ilmango.configSymlink.enable = true;
```

## Hyprland

Hyprland users can wire the service to the UWSM unit:

```nix
programs.ilmango.service.compositor = "hyprland";
```

This creates `wayland-wm@Hyprland.service.wants/ilmango.service`.

## Manual service wiring

To create the service but avoid auto-start wiring:

```nix
programs.ilmango.service.compositor = null;
```

Then start it manually:

```bash
systemctl --user start ilmango.service
```

## Notes

- Use `ilmango logs --full` for runtime errors.
- The packaged `ilmango` launcher wraps Quickshell and runtime tools in `PATH`.
- User preferences still live in Illogical-mango's normal config/state files; the packaged QML source itself is immutable.
- `ilmango update` is not the right update path for a Nix install. Update through your flake inputs and rebuild.
