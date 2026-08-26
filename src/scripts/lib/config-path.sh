#!/usr/bin/env bash

# Library file: intended to be sourced by other scripts.
# Do not set shell options here; callers own execution mode.

# Canonical Illogical-mango config directory is ~/.config/ilmango.
# Installs from the iNiR era used ~/.config/inir; older ones ~/.config/illogical-impulse.
#
# Compatibility policy:
# - If a legacy dir exists as a real directory, keep using it.
# - If a legacy dir is a symlink and the new dir exists, use the new dir.
# - If only one exists, use that one.
# - If none exists, default to new dir.

XDG_CONFIG_HOME="${XDG_CONFIG_HOME:-$HOME/.config}"

ILMANGO_CONFIG_DIR_NEW="${XDG_CONFIG_HOME}/ilmango"
ILMANGO_CONFIG_DIR_PREV="${XDG_CONFIG_HOME}/inir"
ILMANGO_CONFIG_DIR_OLD="${XDG_CONFIG_HOME}/illogical-impulse"

ilmango_config_dir() {
    # Legacy symlink -> new (post-migration steady state)
    if [[ -L "$ILMANGO_CONFIG_DIR_OLD" && -d "$ILMANGO_CONFIG_DIR_NEW" ]]; then
        printf '%s\n' "$ILMANGO_CONFIG_DIR_NEW"
        return
    fi

    # Legacy real directory wins for backwards compatibility.
    if [[ -d "$ILMANGO_CONFIG_DIR_OLD" && ! -L "$ILMANGO_CONFIG_DIR_OLD" ]]; then
        printf '%s\n' "$ILMANGO_CONFIG_DIR_OLD"
        return
    fi

    # iNiR-era directory, before migration 037 moved it.
    if [[ -d "$ILMANGO_CONFIG_DIR_PREV" && ! -L "$ILMANGO_CONFIG_DIR_PREV" ]]; then
        printf '%s\n' "$ILMANGO_CONFIG_DIR_PREV"
        return
    fi

    # New path when no legacy directory is present.
    if [[ -d "$ILMANGO_CONFIG_DIR_NEW" ]]; then
        printf '%s\n' "$ILMANGO_CONFIG_DIR_NEW"
        return
    fi

    # Fresh install default.
    printf '%s\n' "$ILMANGO_CONFIG_DIR_NEW"
}

ilmango_config_file() {
    printf '%s/config.json\n' "$(ilmango_config_dir)"
}

ilmango_version_file() {
    printf '%s/version.json\n' "$(ilmango_config_dir)"
}

ilmango_installed_marker_file() {
    printf '%s/installed_true\n' "$(ilmango_config_dir)"
}

ilmango_migrations_state_file() {
    printf '%s/migrations.json\n' "$(ilmango_config_dir)"
}
