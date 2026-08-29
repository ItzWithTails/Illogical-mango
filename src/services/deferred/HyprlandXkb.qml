pragma Singleton

import QtQuick
import Quickshell
import Quickshell.Io
import Quickshell.Hyprland
import qs.modules.common
import qs.services

/**
 * Exposes the active compositor XKB layout name and code for indicators.
 *
 * The name is historical: the service started as a Hyprland-only adapter, but
 * the bar and lock screen use it as their compositor-neutral layout source.
 */
Singleton {
    id: root
    // You can read these
    property list<string> layoutCodes: []
    property var cachedLayoutCodes: ({})
    property string currentLayoutName: ""
    property string currentLayoutCode: ""
    // For the service
    property var baseLayoutFilePath: "/usr/share/X11/xkb/rules/base.lst"
    property bool needsLayoutRefresh: false

    // Compositors expose a human-readable name; turn it into the short bar code.
    onCurrentLayoutNameChanged: root.updateLayoutCode()
    function updateLayoutCode() {
        if (!currentLayoutName || currentLayoutName.length === 0) {
            root.currentLayoutCode = "";
            return;
        }
        if (cachedLayoutCodes.hasOwnProperty(currentLayoutName)) {
            root.currentLayoutCode = cachedLayoutCodes[currentLayoutName];
        } else {
            // Do not leave the previous layout on screen while base.lst is
            // being read. This fallback also keeps the indicator useful for
            // custom XKB descriptions that base.lst does not know about.
            root.currentLayoutCode = root.fallbackLayoutCode(currentLayoutName);
            getLayoutProc.running = true;
        }
    }

    function fallbackLayoutCode(name) {
        const text = String(name ?? "").trim();
        const parenthesized = text.match(/\(([A-Za-z0-9_-]{2,8})\)\s*$/);
        if (parenthesized)
            return parenthesized[1].toLowerCase();
        if (/^[A-Za-z0-9_-]{2,8}$/.test(text))
            return text.toLowerCase();

        const aliases = {
            "english": "us", "russian": "ru", "german": "de",
            "french": "fr", "spanish": "es", "italian": "it",
            "ukrainian": "ua", "polish": "pl", "portuguese": "pt"
        };
        const firstWord = text.split(/[\s(]/)[0].toLowerCase();
        return aliases[firstWord] ?? firstWord.slice(0, 4);
    }

    function syncFromCompositor() {
        if (CompositorService.isNiri && typeof NiriService !== "undefined") {
            root.layoutCodes = NiriService.keyboardLayoutNames || [];
            root.currentLayoutName = NiriService.getCurrentKeyboardLayoutName();
            return;
        }

        if (CompositorService.isMango && typeof MangoService !== "undefined") {
            root.layoutCodes = MangoService.keyboardLayoutNames || [];
            root.currentLayoutName = MangoService.getCurrentKeyboardLayoutName();
        }
    }

    // Get the layout code from the base.lst file by grabbing the line with the current layout name
    Process {
        id: getLayoutProc
        command: ["cat", root.baseLayoutFilePath]

        stdout: StdioCollector {
            id: layoutCollector

            onStreamFinished: {
                const lines = layoutCollector.text.split("\n");
                const targetDescription = root.currentLayoutName;
                lines.find(line => {
                    // Skip comment lines and empty lines
                    if (!line.trim() || line.trim().startsWith('!'))
                        return false;

                    // Match layout: (whitespace + ) key + whitespace + description
                    const matchLayout = line.match(/^\s*(\S+)\s+(.+)$/);
                    if (matchLayout && matchLayout[2] === targetDescription) {
                        root.cachedLayoutCodes[matchLayout[2]] = matchLayout[1];
                        root.currentLayoutCode = matchLayout[1];
                        return true;
                    }

                    // Match variant: (whitespace + ) variant + whitespace + key + whitespace + description
                    const matchVariant = line.match(/^\s*(\S+)\s+(\S+)\s+(.+)$/);
                    if (matchVariant && matchVariant[3] === targetDescription) {
                        const complexLayout = matchVariant[2] + matchVariant[1];
                        root.cachedLayoutCodes[matchVariant[3]] = complexLayout;
                        root.currentLayoutCode = complexLayout;
                        return true;
                    }
                    
                    return false;
                });
                // console.log("[HyprlandXkb] Found line:", foundLine);
                // console.log("[HyprlandXkb] Layout:", root.currentLayoutName, "| Code:", root.currentLayoutCode);
                // console.log("[HyprlandXkb] Cached layout codes:", JSON.stringify(root.cachedLayoutCodes, null, 2));
            }
        }
    }

    // Find out available layouts and current active layout. Should only be necessary on init
    Process {
        id: fetchLayoutsProc
        running: CompositorService.isHyprland
        command: ["hyprctl", "-j", "devices"]

        stdout: StdioCollector {
            id: devicesCollector
            onStreamFinished: {
                try {
                    const parsedOutput = JSON.parse(devicesCollector.text);
                    const hyprlandKeyboard = parsedOutput["keyboards"].find(kb => kb.main === true);
                    root.layoutCodes = hyprlandKeyboard["layout"].split(",");
                    root.currentLayoutName = hyprlandKeyboard["active_keymap"];
                } catch (e) {
                    console.log("[HyprlandXkb] Failed to parse devices JSON:", e);
                    root.layoutCodes = [];
                    root.currentLayoutName = "";
                }
            }
        }
    }

    // Update the layout name when it changes
    Connections {
        target: Hyprland
        enabled: CompositorService.isHyprland
        function onRawEvent(event) {
            if (event.name === "activelayout") {
                if (root.needsLayoutRefresh) {
                    root.needsLayoutRefresh = false;
                    fetchLayoutsProc.running = true;
                }

                // If there's only one layout, the updated layout is always the same
                if (root.layoutCodes.length <= 1) return;

                // Update when layout might have changed
                const dataString = event.data;
                root.currentLayoutName = dataString.substring(dataString.indexOf(",") + 1);

                // Update layout for on-screen keyboard (osk)
                Config.setNestedValue(["osk", "layout"], root.currentLayoutName.split(" (")[0])
            } else if (event.name == "configreloaded") {
                // Mark layout code list to be updated when config is reloaded
                root.needsLayoutRefresh = true;
            }
        }
    }

    Connections {
        target: NiriService
        enabled: CompositorService.isNiri
        function onKeyboardLayoutNamesChanged() { root.syncFromCompositor(); }
        function onCurrentKeyboardLayoutIndexChanged() { root.syncFromCompositor(); }
    }

    Connections {
        target: MangoService
        enabled: CompositorService.isMango
        function onKeyboardLayoutNamesChanged() { root.syncFromCompositor(); }
        function onCurrentKeyboardLayoutChanged() { root.syncFromCompositor(); }
    }

    Component.onCompleted: {
        if (CompositorService.isNiri || CompositorService.isMango)
            root.syncFromCompositor();
    }
}
