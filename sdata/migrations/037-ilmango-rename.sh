#!/usr/bin/env bash
# Migration: iNiR -> Illogical-mango rename
#
# The project, its CLI, its config namespace and its global style were all
# called "inir". They are now "ilmango" (brand name: Illogical-mango).
# This moves every installed artifact to the new name and leaves compatibility
# symlinks behind, so user keybinds and scripts that still point at the old
# paths keep working.

MIGRATION_ID="037-ilmango-rename"
MIGRATION_TITLE="Rename iNiR to Illogical-mango"
MIGRATION_DESCRIPTION="Moves config, shell payload, launcher, services, desktop entries and
  the compositor keybind file from the 'inir' name to 'ilmango'. Renames the
  'inir' global style to 'ilmango' in your config. Old paths stay reachable
  through symlinks."
MIGRATION_TARGET_FILE="~/.config/ilmango + installed artifacts"
MIGRATION_REQUIRED=true

_037_xdg_config() { printf '%s' "${XDG_CONFIG_HOME:-$HOME/.config}"; }
_037_xdg_cache() { printf '%s' "${XDG_CACHE_HOME:-$HOME/.cache}"; }
_037_xdg_data() { printf '%s' "${XDG_DATA_HOME:-$HOME/.local/share}"; }
_037_bin_dir() { printf '%s' "${XDG_BIN_HOME:-$HOME/.local/bin}"; }

# Config file, wherever it currently lives.
_037_config_file() {
  local cfg
  cfg="$(_037_xdg_config)"
  local candidate
  for candidate in "$cfg/ilmango/config.json" "$cfg/inir/config.json" "$cfg/illogical-impulse/config.json"; do
    [[ -f "$candidate" ]] && { printf '%s' "$candidate"; return; }
  done
  printf '%s' "$cfg/ilmango/config.json"
}

# Move $1 to $2 and leave $1 behind as a symlink to $2.
# No-op when the source is missing or already a symlink.
_037_move_with_link() {
  local src="$1" dst="$2"
  [[ -e "$src" ]] || return 0
  [[ -L "$src" ]] && return 0
  [[ -e "$dst" ]] && return 0
  mkdir -p "$(dirname "$dst")"
  mv "$src" "$dst" || return 1
  ln -s "$dst" "$src" 2>/dev/null || true
}

# Plain rename, no compatibility link (for generated files nothing references
# by absolute path).
_037_move() {
  local src="$1" dst="$2"
  [[ -e "$src" ]] || return 0
  [[ -e "$dst" ]] && return 0
  mkdir -p "$(dirname "$dst")"
  mv "$src" "$dst" || return 1
}

migration_check() {
  local cfg cache data bin
  cfg="$(_037_xdg_config)"; cache="$(_037_xdg_cache)"
  data="$(_037_xdg_data)"; bin="$(_037_bin_dir)"

  # Any real (non-symlink) artifact still carrying the old name.
  local p
  for p in \
    "$cfg/inir" \
    "$cfg/quickshell/inir" \
    "$cache/inir" \
    "$cache/quickshell/inir" \
    "$bin/inir" \
    "$bin/inir_super_overview_daemon.py" \
    "$cfg/systemd/user/inir.service" \
    "$cfg/systemd/user/inir-super-overview.service" \
    "$data/applications/inir.desktop" \
    "$data/applications/inir-settings.desktop" \
    "$data/icons/hicolor/scalable/apps/inir.svg" \
    "$cfg/mango/inir.conf" \
    "$cfg/fish/conf.d/inir-env.fish" \
    "$cfg/vesktop/themes/inir-midnight.theme.css" \
    "$cfg/spicetify/Themes/Inir" \
    "$cfg/zed/themes/iNiR-alt.json"
  do
    [[ -e "$p" && ! -L "$p" ]] && return 0
  done

  # Old env var name still in a shell profile or compositor config.
  local f
  for f in "$HOME/.bashrc" "$HOME/.zshrc" "$cfg/mango/config.conf" "$cfg/niri/config.kdl"; do
    [[ -f "$f" ]] && grep -q "INIR_\|# iNiR\|/inir\b" "$f" 2>/dev/null && return 0
  done

  # Global style still set to the old name.
  local conf
  conf="$(_037_config_file)"
  if [[ -f "$conf" ]] && command -v jq >/dev/null 2>&1; then
    jq -e '(.appearance.globalStyle? == "inir")
           or (.appearance.globalStyleCornerStyles? | type == "object" and has("inir"))' \
      "$conf" >/dev/null 2>&1 && return 0
  fi

  return 1
}

migration_preview() {
  local cfg
  cfg="$(_037_xdg_config)"
  echo -e "${STY_YELLOW}~ rename installed artifacts:${STY_RST}"
  echo "  ${cfg}/inir                  -> ${cfg}/ilmango"
  echo "  ${cfg}/quickshell/inir       -> ${cfg}/quickshell/ilmango"
  echo "  ~/.local/bin/inir            -> ~/.local/bin/ilmango"
  echo "  inir.service                 -> ilmango.service"
  echo "  inir.desktop                 -> ilmango.desktop"
  echo "  ${cfg}/mango/inir.conf       -> ${cfg}/mango/ilmango.conf"
  echo ""
  echo -e "${STY_RED}- export INIR_VENV=...${STY_RST}"
  echo -e "${STY_GREEN}+ export ILMANGO_VENV=...${STY_RST}"
  echo ""
  echo -e "${STY_YELLOW}~ appearance.globalStyle: \"inir\" -> \"ilmango\"${STY_RST}"
  echo ""
  echo "Old paths stay reachable — each move leaves a symlink behind."
}

migration_diff() {
  local cfg conf
  cfg="$(_037_xdg_config)"
  echo "Artifacts still using the old name:"
  local p found=0
  for p in "$cfg/inir" "$cfg/quickshell/inir" "$cfg/mango/inir.conf" \
           "${XDG_BIN_HOME:-$HOME/.local/bin}/inir" \
           "$cfg/systemd/user/inir.service"; do
    if [[ -e "$p" && ! -L "$p" ]]; then echo "  $p"; found=1; fi
  done
  [[ $found -eq 0 ]] && echo "  (none)"

  conf="$(_037_config_file)"
  if [[ -f "$conf" ]] && command -v jq >/dev/null 2>&1; then
    echo "Global style:"
    jq -r '.appearance.globalStyle // "(unset)"' "$conf" 2>/dev/null | sed 's/^/  /'
  fi
}

migration_apply() {
  local cfg cache data bin
  cfg="$(_037_xdg_config)"; cache="$(_037_xdg_cache)"
  data="$(_037_xdg_data)"; bin="$(_037_bin_dir)"

  # --- Config directory -----------------------------------------------------
  # Keep ~/.config/inir as a symlink: migrations already in flight may still
  # hold the old path, and user scripts reference it.
  _037_move_with_link "$cfg/inir" "$cfg/ilmango"

  # Re-point the illogical-impulse compatibility link at the new canonical dir.
  if [[ -L "$cfg/illogical-impulse" && -d "$cfg/ilmango" ]]; then
    local target
    target="$(readlink "$cfg/illogical-impulse" 2>/dev/null || true)"
    if [[ "$target" != "$cfg/ilmango" ]]; then
      rm -f "$cfg/illogical-impulse"
      ln -s "$cfg/ilmango" "$cfg/illogical-impulse"
    fi
  fi

  # --- Shell payload, cache -------------------------------------------------
  _037_move_with_link "$cfg/quickshell/inir" "$cfg/quickshell/ilmango"
  _037_move "$cache/inir" "$cache/ilmango"
  _037_move "$cache/quickshell/inir" "$cache/quickshell/ilmango"

  # --- Launcher and daemon --------------------------------------------------
  _037_move_with_link "$bin/inir" "$bin/ilmango"
  _037_move "$bin/inir_super_overview_daemon.py" "$bin/ilmango_super_overview_daemon.py"

  # --- systemd user units ---------------------------------------------------
  local unit_dir="$cfg/systemd/user"
  local had_service=0
  if [[ -e "$unit_dir/inir.service" && ! -L "$unit_dir/inir.service" ]]; then
    had_service=1
    systemctl --user stop inir.service >/dev/null 2>&1 || true
    systemctl --user disable inir.service >/dev/null 2>&1 || true
  fi
  _037_move "$unit_dir/inir.service" "$unit_dir/ilmango.service"
  _037_move "$unit_dir/inir.service.d" "$unit_dir/ilmango.service.d"
  _037_move "$unit_dir/inir-super-overview.service" "$unit_dir/ilmango-super-overview.service"

  # Rewrite unit contents and any wants/ symlinks pointing at the old names.
  local u
  for u in "$unit_dir/ilmango.service" "$unit_dir/ilmango-super-overview.service" \
           "$unit_dir/ilmango.service.d"/*.conf; do
    [[ -f "$u" ]] || continue
    sed -i 's/\bINIR_/ILMANGO_/g; s|/inir\b|/ilmango|g; s|\binir |ilmango |g; s|inir\.service|ilmango.service|g' "$u"
  done

  local wants
  for wants in "$unit_dir"/*.wants; do
    [[ -d "$wants" ]] || continue
    if [[ -L "$wants/inir.service" ]]; then
      rm -f "$wants/inir.service"
      ln -sf "$unit_dir/ilmango.service" "$wants/ilmango.service"
    fi
    if [[ -L "$wants/inir-super-overview.service" ]]; then
      rm -f "$wants/inir-super-overview.service"
      ln -sf "$unit_dir/ilmango-super-overview.service" "$wants/ilmango-super-overview.service"
    fi
  done

  if command -v systemctl >/dev/null 2>&1; then
    systemctl --user daemon-reload >/dev/null 2>&1 || true
    [[ $had_service -eq 1 ]] && systemctl --user enable ilmango.service >/dev/null 2>&1 || true
  fi

  # --- Desktop entries and icon --------------------------------------------
  _037_move "$data/applications/inir.desktop" "$data/applications/ilmango.desktop"
  _037_move "$data/applications/inir-settings.desktop" "$data/applications/ilmango-settings.desktop"
  _037_move "$data/icons/hicolor/scalable/apps/inir.svg" "$data/icons/hicolor/scalable/apps/ilmango.svg"
  local d
  for d in "$data/applications/ilmango.desktop" "$data/applications/ilmango-settings.desktop"; do
    [[ -f "$d" ]] || continue
    sed -i 's|/inir\b|/ilmango|g; s|^Exec=inir\b|Exec=ilmango|; s|^Icon=inir$|Icon=ilmango|; s|\biNiR\b|Illogical-mango|g' "$d"
  done

  # --- Compositor keybinds --------------------------------------------------
  # The keybind file is ours and generated, so every `inir` in it is the old
  # command name. The user's own config.conf is touched as little as possible:
  # only our source-optional line, our env vars, and launcher invocations.
  _037_move "$cfg/mango/inir.conf" "$cfg/mango/ilmango.conf"
  if [[ -f "$cfg/mango/ilmango.conf" ]]; then
    sed -i -E 's%\binir\b%ilmango%g; s%\bINIR_%ILMANGO_%g' "$cfg/mango/ilmango.conf"
  fi
  if [[ -f "$cfg/mango/config.conf" ]]; then
    sed -i -E 's%mango/inir\.conf%mango/ilmango.conf%g;
               s%\bINIR_%ILMANGO_%g;
               s%/quickshell/inir\b%/quickshell/ilmango%g;
               s%\binir( +(run|settings|lock|logs|doctor|update|restart|repair|ipc|service|overview)\b)%ilmango\1%g' \
      "$cfg/mango/config.conf"
  fi

  # --- Niri config (inherited setups) --------------------------------------
  local nf
  for nf in "$cfg/niri/config.kdl" "$cfg/niri/config.d"/*.kdl; do
    [[ -f "$nf" ]] || continue
    sed -i -E 's%\bINIR_%ILMANGO_%g;
               s%/quickshell/inir\b%/quickshell/ilmango%g;
               s%"inir"%"ilmango"%g;
               s%"inir( )%"ilmango\1%g' "$nf"
  done

  # --- Shell profiles -------------------------------------------------------
  local rc
  for rc in "$HOME/.bashrc" "$HOME/.zshrc"; do
    [[ -f "$rc" ]] || continue
    # The block markers were "# iNiR environment" ... "# end iNiR".
    sed -i 's|^# iNiR environment$|# Illogical-mango environment|; s|^# end iNiR$|# end Illogical-mango|; s|\bINIR_VENV\b|ILMANGO_VENV|g' "$rc"
  done

  _037_move "$cfg/fish/conf.d/inir-env.fish" "$cfg/fish/conf.d/ilmango-env.fish"
  if [[ -f "$cfg/fish/conf.d/ilmango-env.fish" ]]; then
    sed -i 's|\bINIR_VENV\b|ILMANGO_VENV|g; s|^# iNiR environment|# Illogical-mango environment|' \
      "$cfg/fish/conf.d/ilmango-env.fish"
  fi

  # --- Generated themes -----------------------------------------------------
  _037_move "$cfg/vesktop/themes/inir-midnight.theme.css" "$cfg/vesktop/themes/ilmango-midnight.theme.css"
  _037_move "$cfg/Vesktop/themes/inir-midnight.theme.css" "$cfg/Vesktop/themes/ilmango-midnight.theme.css"
  _037_move "$cfg/spicetify/Themes/Inir" "$cfg/spicetify/Themes/Ilmango"
  _037_move "$cfg/zed/themes/iNiR-alt.json" "$cfg/zed/themes/Illogical-mango-alt.json"

  # --- Global style in user config -----------------------------------------
  local conf
  conf="$(_037_config_file)"
  if [[ -f "$conf" ]] && command -v jq >/dev/null 2>&1; then
    local tmp="${conf}.migration-tmp"
    if jq '
      (if .appearance.globalStyle? == "inir" then .appearance.globalStyle = "ilmango" else . end)
      | (if (.appearance.globalStyleCornerStyles? | type) == "object"
             and (.appearance.globalStyleCornerStyles | has("inir"))
         then .appearance.globalStyleCornerStyles
                |= (. + {ilmango: .inir} | del(.inir))
         else . end)
    ' "$conf" > "$tmp" 2>/dev/null; then
      mv "$tmp" "$conf"
      echo "  Global style renamed to ilmango where it was set to inir"
    else
      rm -f "$tmp"
      echo "  Could not rewrite $conf — leaving the global style untouched"
    fi
  fi

  echo "  Renamed installed artifacts from inir to ilmango"
}
