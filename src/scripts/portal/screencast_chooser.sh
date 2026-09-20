#!/usr/bin/env bash

# KDE-style source chooser for xdg-desktop-portal-wlr. The portal sends valid
# screencast targets on stdin and expects the selected line back on stdout.

set -u

dialog_bin="${KDIALOG_BIN:-/usr/bin/kdialog}"
declare -a menu_items=()
first_source=""

while IFS= read -r source || [[ -n "$source" ]]; do
    [[ -n "$source" ]] || continue
    [[ -n "$first_source" ]] || first_source="$source"

    case "$source" in
        "Monitor: "*) label="🖥  ${source#Monitor: }" ;;
        "Window: "*)  label="▣  ${source#Window: }" ;;
        *)             label="$source" ;;
    esac

    menu_items+=("$source" "$label")
done

((${#menu_items[@]} > 0)) || exit 0
[[ -x "$dialog_bin" ]] || exit 127

"$dialog_bin" \
    --title "Screen Sharing" \
    --icon video-display \
    --geometry 720x420 \
    --default "$first_source" \
    --menu "Choose a monitor or window to share" \
    "${menu_items[@]}"
