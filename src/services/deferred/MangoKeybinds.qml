pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import Quickshell.Io
import qs.services

/**
 * Keybinds for the mango compositor.
 *
 * Parses mango's own config format directly — no helper script needed, the
 * grammar is one bind per line:
 *
 *     bind=<mods>,<key>,<func>[,arg...]
 *
 * Mods are joined with '+' and are case-insensitive ("SUPER+SHIFT", "alt",
 * "none"). mango resolves its config as ~/.config/mango/config.conf first and
 * falls back to /etc/mango/config.conf (see parse_config.h: load_config), so we
 * follow the same order. Note it is `mango`, NOT `mangowm`, which is a
 * different directory that mango never reads.
 *
 * Read-only: unlike NiriKeybinds there is no write-back path, so the cheatsheet
 * shows binds but the settings editor cannot rewrite them.
 */
Singleton {
    id: root

    readonly property string userConfigPath: `${Quickshell.env("HOME")}/.config/mango/config.conf`
    readonly property string systemConfigPath: "/etc/mango/config.conf"

    property string configPath: ""
    property bool loaded: false
    property string errorMessage: ""

    property var keybinds: ({ children: [] })

    // Includes are collected while parsing and drained one at a time through
    // includeView. The buckets accumulate across every file, so the cheatsheet
    // shows one list rather than one per config fragment.
    property var _buckets: ({})
    property int _count: 0
    property var _pending: []
    property var _visited: ({})

    function reload(): void {
        root._buckets = ({})
        root._count = 0
        root._pending = []
        root._visited = ({})
        userConfigView.reload()
    }

    // Try the user config first; fall back to the system one when it is absent,
    // mirroring mango's own resolution order.
    FileView {
        id: userConfigView
        path: root.userConfigPath
        onLoaded: {
            root.configPath = root.userConfigPath
            root._collect(text(), root.userConfigPath)
            root._drain()
        }
        onLoadFailed: {
            systemConfigView.path = root.systemConfigPath
            systemConfigView.reload()
        }
    }

    FileView {
        id: systemConfigView
        // The path is set only when the user config is missing. Binding it here
        // loaded both, and since a user config starts life as a copy of this
        // one, every default bind was listed twice.
        onLoaded: {
            root.configPath = root.systemConfigPath
            root._collect(text(), root.systemConfigPath)
            root._drain()
        }
        onLoadFailed: {
            root.loaded = false
            root.errorMessage = "No mango config found at " + root.userConfigPath + " or " + root.systemConfigPath
            root.keybinds = ({ children: [] })
        }
    }

    // One view reused for every include, in turn: the list is short and the
    // order does not matter once the buckets are merged.
    FileView {
        id: includeView
        onLoaded: {
            root._collect(text(), path)
            root._drain()
        }
        // source-optional names a file that need not exist; a missing include
        // is the normal case, not an error.
        onLoadFailed: root._drain()
    }

    Component.onCompleted: if (CompositorService.isMango) root.reload()

    // ── Presentation helpers ─────────────────────────────────────────────

    readonly property var _modNames: ({
        "super": "Super", "ctrl": "Ctrl", "control": "Ctrl",
        "alt": "Alt", "shift": "Shift", "logo": "Super", "mod": "Super"
    })

    function _prettyMods(modString) {
        if (!modString) return []
        const raw = modString.trim().toLowerCase()
        if (raw === "" || raw === "none") return []
        return raw.split("+")
            .map(m => m.trim())
            .filter(m => m.length > 0 && m !== "none")
            .map(m => root._modNames[m] ?? (m.charAt(0).toUpperCase() + m.slice(1)))
    }

    function _prettyKey(key) {
        if (!key) return ""
        const k = key.trim()
        // Single letters read better capitalised; named keys (Left, Return,
        // XF86AudioRaiseVolume) already carry their own casing.
        return k.length === 1 ? k.toUpperCase() : k
    }

    // Human-readable description per mango dispatch function. Args are appended
    // where they carry meaning (spawn's command, tag numbers, directions).
    function _describe(func, args) {
        const a = args ?? []
        const arg0 = a[0] ?? ""
        switch (func) {
        case "spawn":
        case "spawn_shell":
            return a.join(",")
        case "spawn_on_empty":       return `${arg0} (on empty tag)`
        case "view":                 return `Switch to tag ${arg0}`
        case "tag":                  return `Move window to tag ${arg0}`
        case "tagsilent":            return `Move window to tag ${arg0} (stay)`
        case "toggletag":            return `Toggle tag ${arg0} on window`
        case "viewtoleft":           return "Previous tag"
        case "viewtoright":          return "Next tag"
        case "viewtoleft_have_client":  return "Previous non-empty tag"
        case "viewtoright_have_client": return "Next non-empty tag"
        case "tagtoleft":            return "Move window to previous tag"
        case "tagtoright":           return "Move window to next tag"
        case "focusstack":           return `Focus ${arg0 || "next"} window`
        case "focusdir":             return `Focus window ${arg0}`
        case "focusmon":             return `Focus monitor ${arg0}`
        case "focuslast":            return "Focus last window"
        case "focusid":              return "Focus window by id"
        case "exchange_client":      return `Swap window ${arg0}`
        case "exchange_stack_client": return `Swap window in stack (${arg0})`
        case "tagmon":               return `Move window to monitor ${arg0}`
        case "killclient":           return "Close window"
        case "quit":                 return "Quit mango"
        case "reload_config":        return "Reload config"
        case "load_config_file":     return `Load config ${arg0}`
        case "togglefloating":       return "Toggle floating"
        case "togglefullscreen":     return "Toggle fullscreen"
        case "togglefakefullscreen": return "Toggle fake fullscreen"
        case "togglemaximizescreen": return "Toggle maximize"
        case "toggleoverview":       return "Toggle overview"
        case "toggleglobal":         return "Toggle global (show on all tags)"
        case "toggleoverlay":        return "Toggle overlay"
        case "togglejump":           return "Jump to window"
        case "minimized":            return "Minimize window"
        case "restore_minimized":    return "Restore minimized window"
        case "toggle_scratchpad":    return "Toggle scratchpad"
        case "toggle_render_border": return "Toggle window borders"
        case "togglegaps":           return "Toggle gaps"
        case "incgaps":              return `${Number(arg0) >= 0 ? "Increase" : "Decrease"} gaps`
        case "setlayout":            return `Set layout: ${arg0}`
        case "switch_layout":        return "Next layout"
        case "switch_keyboard_layout": return "Switch keyboard layout"
        case "setkeymode":           return `Key mode: ${arg0}`
        case "incnmaster":           return "Change master count"
        case "setmfact":             return "Resize master area"
        case "zoom":                 return "Promote to master"
        case "centerwin":            return "Center window"
        case "movewin":              return "Move window"
        case "resizewin":            return "Resize window"
        case "moveresize":           return "Move/resize with pointer"
        case "set_proportion":       return `Set proportion ${arg0}`
        case "switch_proportion_preset": return "Cycle proportion preset"
        case "scroller_stack":       return `Scroller stack ${arg0}`
        case "dwindle_toggle_split_direction": return "Toggle split direction"
        case "groupjoin":            return `Group with window ${arg0}`
        case "groupleave":           return "Leave group"
        case "groupfocus":           return `Focus ${arg0} in group`
        case "togglehdr":            return "Toggle HDR"
        case "chvt":                 return `Switch to VT ${arg0}`
        case "toggle_trackpad_enable": return "Toggle trackpad"
        case "setoption":            return `Set ${arg0}`
        case "create_virtual_output": return "Create virtual output"
        case "destroy_all_virtual_output": return "Destroy virtual outputs"
        default:
            return a.length > 0 ? `${func} (${a.join(", ")})` : func
        }
    }

    // Category buckets, checked in order — first match wins.
    function _categoryFor(func) {
        if (func === "spawn" || func === "spawn_shell" || func === "spawn_on_empty")
            return "Applications"
        if (func.startsWith("view") || func.startsWith("tag") || func === "toggletag")
            return "Tags"
        if (func.startsWith("focus") || func.startsWith("exchange") || func.startsWith("group"))
            return "Focus & Windows"
        if (func.startsWith("toggle") || func === "minimized" || func === "restore_minimized")
            return "Window State"
        if (func.includes("layout") || func.includes("gaps") || func.includes("proportion")
                || func.includes("master") || func === "zoom" || func === "setmfact"
                || func.includes("scroller") || func.includes("dwindle")
                || func === "movewin" || func === "resizewin" || func === "moveresize"
                || func === "centerwin")
            return "Layout"
        return "System"
    }

    readonly property var _categoryOrder: [
        "Applications", "Tags", "Focus & Windows", "Window State", "Layout", "System"
    ]

    // ── Parser ───────────────────────────────────────────────────────────

    // _resolve turns a path as written in a config into an absolute one.
    // mango resolves a bare name against ~/.config/mango; see parse_config.h.
    function _resolve(raw) {
        const home = Quickshell.env("HOME")
        let path = raw.trim().replace(/^["']|["']$/g, "")
        if (path.length === 0)
            return ""
        if (path.startsWith("~/"))
            path = home + path.slice(1)
        else if (path.startsWith("$HOME/"))
            path = home + path.slice("$HOME".length)
        else if (!path.startsWith("/"))
            path = `${home}/.config/mango/${path}`
        return path
    }

    // _collect reads one config file into the shared buckets and queues any
    // file it includes.
    function _collect(content, path) {
        // FileView can report the same file more than once — a reload and a
        // change notification both arrive as onLoaded — and counting it twice
        // shows every bind twice in the cheatsheet.
        if (!content || root._visited[path])
            return
        root._visited[path] = true

        for (const rawLine of content.split("\n")) {
            const line = rawLine.trim()
            if (line.length === 0 || line.startsWith("#"))
                continue

            const include = line.match(/^source(-optional)?=(.*)$/)
            if (include) {
                const target = root._resolve(include[2])
                // A config that includes itself, directly or in a cycle, would
                // otherwise load for ever.
                if (target.length > 0 && !root._visited[target] && root._pending.indexOf(target) === -1)
                    root._pending.push(target)
                continue
            }

            // bind, bindl, bindr and the rest are the same grammar; the letters
            // after "bind" are flags such as "fires while locked".
            const bind = line.match(/^bind[slrpc]*=(.*)$/)
            if (!bind)
                continue

            const parts = bind[1].split(",")
            if (parts.length < 3)
                continue

            const mods = root._prettyMods(parts[0])
            const key = root._prettyKey(parts[1])
            const func = parts[2].trim()
            const args = parts.slice(3).map(s => s.trim())
            if (key.length === 0 || func.length === 0)
                continue

            const category = root._categoryFor(func)
            if (!root._buckets[category])
                root._buckets[category] = []
            root._buckets[category].push({
                mods: mods,
                key: key,
                comment: root._describe(func, args)
            })
            root._count++
        }
    }

    // _drain loads the next queued include, or publishes the result when the
    // queue is empty.
    function _drain() {
        if (root._pending.length > 0) {
            const next = root._pending.shift()
            includeView.path = next
            includeView.reload()
            return
        }
        root._publish()
    }

    function _publish() {
        const children = []
        for (const name of root._categoryOrder) {
            if (root._buckets[name] && root._buckets[name].length > 0)
                children.push({ name: name, children: [{ keybinds: root._buckets[name] }] })
        }
        // Any category the order list does not mention (defensive — _categoryFor
        // only ever returns names from that list today).
        for (const name in root._buckets) {
            if (root._categoryOrder.indexOf(name) === -1)
                children.push({ name: name, children: [{ keybinds: root._buckets[name] }] })
        }

        root.keybinds = ({ children: children })
        root.loaded = root._count > 0
        console.info("MangoKeybinds: " + root._count + " binds from " + Object.keys(root._visited).length + " file(s)")
        root.errorMessage = root._count > 0 ? "" : "No binds found in " + root.configPath
    }
}
