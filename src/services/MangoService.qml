pragma Singleton

pragma ComponentBehavior: Bound

import QtCore
import QtQuick
import Quickshell
import Quickshell.Io
import Quickshell.Wayland
import qs.modules.common
import qs.services

// mango compositor backend, mirrors the public surface of NiriService where
// the two window models overlap. mango is dwm-style: N (1-9) tags per
// monitor, a client can sit on multiple tags at once, and every `watch`
// stream sends a FULL snapshot on each change (verified against mango
// 0.16.1's ipc.h — ipc_notify_all_clients/all_monitors rebuild the entire
// array every time), so there is no incremental-diff bookkeeping to do here.
//
// Workspace identity: mango tags are (monitor, 1-based index) pairs, not
// globally unique ids like niri's. We synthesize "<monitor>::<index>" as the
// stable id used everywhere NiriService would use niri's numeric ws id.
Singleton {
    id: root

    readonly property string socketPath: Quickshell.env("MANGO_INSTANCE_SIGNATURE")

    property var workspaces: ({})        // synthId -> workspace-like object
    property var allWorkspaces: []
    property int focusedWorkspaceIndex: 0
    property string focusedWorkspaceId: ""
    property var currentOutputWorkspaces: []
    property string currentOutput: ""

    property var outputs: ({})           // monitor name -> monitor json
    property var windows: []             // client json, enriched with workspace_id/is_focused
    property var mruWindowIds: []
    property var activeWindow: null

    property string currentKeyboardLayout: ""
    // mango's IPC only ever reports the *current* layout name, never the
    // full configured list, so we fall back to what we configured in
    // /etc/mango/config.conf (xkb_rules_layout=us,ru) to give
    // hasMultipleKeyboardLayouts/switchLayout something to work with.
    property var keyboardLayoutNames: ["English (US)", "Russian"]
    property int currentKeyboardLayoutIndex: 0
    readonly property bool hasMultipleKeyboardLayouts: keyboardLayoutNames.length > 1

    signal windowUrgentChanged
    signal windowOrderChanged

    // ---------- watch sockets (one persistent connection per resource) ----------

    DankSocket {
        id: clientsSocket
        path: root.socketPath
        connected: CompositorService.isMango

        onConnectionStateChanged: if (connected) send("watch all-clients")

        parser: SplitParser {
            onRead: line => {
                try {
                    const data = JSON.parse(line)
                    if (data.clients) root._applyClients(data.clients)
                } catch (e) {
                    console.warn("MangoService: failed to parse all-clients:", line, e)
                }
            }
        }
    }

    DankSocket {
        id: monitorsSocket
        path: root.socketPath
        connected: CompositorService.isMango

        onConnectionStateChanged: if (connected) send("watch all-monitors")

        parser: SplitParser {
            onRead: line => {
                try {
                    const data = JSON.parse(line)
                    if (data.monitors) root._applyMonitors(data.monitors)
                } catch (e) {
                    console.warn("MangoService: failed to parse all-monitors:", line, e)
                }
            }
        }
    }

    DankSocket {
        id: keyboardLayoutSocket
        path: root.socketPath
        connected: CompositorService.isMango

        onConnectionStateChanged: if (connected) send("watch keyboardlayout")

        parser: SplitParser {
            onRead: line => {
                try {
                    const data = JSON.parse(line)
                    if (data.layout !== undefined) root._applyKeyboardLayout(data.layout)
                } catch (e) {
                    console.warn("MangoService: failed to parse keyboardlayout:", line, e)
                }
            }
        }
    }

    // ---------- one-shot dispatch/get, via mmsg (already on $PATH under mango) ----------

    Process {
        id: dispatchProcess
        property var _queue: []
        property bool _running: false

        stdout: StdioCollector {}
        stderr: StdioCollector {
            onStreamFinished: if (text.length > 0) console.warn("MangoService: mmsg:", text)
        }

        onExited: {
            _running = false
            if (_queue.length > 0)
                _runNext()
        }

        function _runNext() {
            const args = _queue.shift()
            _running = true
            command = args
            running = true
        }

        function enqueue(args) {
            _queue.push(args)
            if (!_running)
                _runNext()
        }
    }

    function dispatch(funcAndArgs, clientId) {
        if (!CompositorService.isMango)
            return false
        const args = ["mmsg", "dispatch", funcAndArgs]
        if (clientId !== undefined && clientId !== null)
            args.push("client," + clientId)
        dispatchProcess.enqueue(args)
        return true
    }

    // ---------- snapshot application ----------

    function _workspaceId(monitorName, tagIndex) {
        return monitorName + "::" + tagIndex
    }

    // Public alias — Workspaces.qml et al need to resolve a bar slot number
    // (1-9, per current output) to the same synthetic id used in `workspaces`.
    function workspaceIdFor(monitorName, tagIndex) {
        return root._workspaceId(monitorName, tagIndex)
    }

    function _applyMonitors(monitors) {
        const nextOutputs = {}
        const nextWorkspaces = {}
        let nextCurrentOutput = ""

        for (const mon of monitors) {
            nextOutputs[mon.name] = mon
            if (mon.active)
                nextCurrentOutput = mon.name

            const activeTags = mon.active_tags || []
            for (const tag of (mon.tags || [])) {
                const id = root._workspaceId(mon.name, tag.index)
                nextWorkspaces[id] = {
                    id,
                    idx: tag.index,
                    output: mon.name,
                    is_active: activeTags.includes(tag.index),
                    // "focused" here means: this is the active tag on the
                    // monitor that currently has compositor input focus.
                    is_focused: mon.active && activeTags.includes(tag.index),
                    is_urgent: tag.is_urgent,
                    client_count: tag.client_count,
                    layout: tag.layout
                }
            }
        }

        outputs = nextOutputs
        root.workspaces = nextWorkspaces
        allWorkspaces = Object.values(nextWorkspaces).sort((a, b) => {
            if (a.output !== b.output) return a.output < b.output ? -1 : 1
            return a.idx - b.idx
        })

        currentOutput = nextCurrentOutput
        focusedWorkspaceIndex = allWorkspaces.findIndex(w => w.is_focused)
        if (focusedWorkspaceIndex < 0) focusedWorkspaceIndex = 0
        focusedWorkspaceId = allWorkspaces[focusedWorkspaceIndex]?.id ?? ""

        updateCurrentOutputWorkspaces()
    }

    function _applyClients(clients) {
        const previousById = new Map()
        for (const w of root.windows) previousById.set(w.id, w)

        // mango keeps a per-tag focus memory: `is_focused` stays true on clients
        // that are no longer on screen, and more than one client can carry it at
        // once (one per tag). Only a client that is BOTH focused and visible is
        // actually the focused window — matching what `get focusing-client`
        // reports, which returns "no focused client" on an empty tag.
        const nextWindows = clients.map(c => {
            const primaryTag = (c.tags && c.tags.length > 0) ? c.tags[0] : null
            return Object.assign({}, c, {
                app_id: c.appid,
                workspace_id: primaryTag !== null ? root._workspaceId(c.monitor, primaryTag) : null,
                is_focused: !!c.is_focused && !!c.is_visible
            })
        })

        const orderChanged = root._windowOrderDiffers(root.windows, nextWindows)

        // MRU: move newly-focused window to the front.
        const focused = nextWindows.find(w => w.is_focused)
        if (focused) {
            const rest = mruWindowIds.filter(id => id !== focused.id)
            rest.unshift(focused.id)
            mruWindowIds = rest
        }
        const liveIds = new Set(nextWindows.map(w => w.id))
        if (mruWindowIds.some(id => !liveIds.has(id)))
            mruWindowIds = mruWindowIds.filter(id => liveIds.has(id))

        root.windows = nextWindows
        activeWindow = focused ?? null

        if (orderChanged)
            windowOrderChanged()

        const anyNewUrgent = nextWindows.some(w => {
            const prev = previousById.get(w.id)
            return w.is_urgent && (!prev || !prev.is_urgent)
        })
        if (anyNewUrgent)
            windowUrgentChanged()
    }

    function _windowOrderDiffers(previousWindows, nextWindows) {
        if (previousWindows.length !== nextWindows.length)
            return true
        const previousById = new Map()
        for (const w of previousWindows) previousById.set(w.id, w)
        for (const w of nextWindows) {
            const prev = previousById.get(w.id)
            if (!prev) return true
            if (prev.workspace_id !== w.workspace_id || !!prev.is_floating !== !!w.is_floating)
                return true
        }
        return false
    }

    function _applyKeyboardLayout(name) {
        currentKeyboardLayout = name
        const idx = keyboardLayoutNames.findIndex(n => n === name)
        currentKeyboardLayoutIndex = idx >= 0 ? idx : 0
    }

    function updateCurrentOutputWorkspaces() {
        if (!currentOutput) {
            currentOutputWorkspaces = allWorkspaces
            return
        }
        currentOutputWorkspaces = allWorkspaces.filter(w => w.output === currentOutput)
    }

    // ========== ACTIONS (mirror the NiriService names Illogical-mango call sites use) ==========

    function switchToWorkspace(tagIndex) {
        return dispatch("view," + tagIndex + ",0")
    }

    function switchToWorkspaceById(workspaceId) {
        const ws = root.workspaces[workspaceId]
        if (!ws) return false
        return dispatch("view," + ws.idx + ",0")
    }

    function focusWindow(windowId) {
        return dispatch("focusid", windowId)
    }

    function moveWindowToWorkspace(windowId, tagIndex, focus) {
        dispatch("tag," + tagIndex + ",0", windowId)
        if (focus)
            root.switchToWorkspace(tagIndex)
        return true
    }

    function closeWindow(windowId) {
        return dispatch("killclient,0", windowId)
    }

    function powerOffMonitors() {
        console.warn("MangoService: mango has no DPMS dispatch action; wire an idle daemon that speaks wlr-output-power-management-v1 instead")
        return false
    }

    function powerOnMonitors() {
        console.warn("MangoService: mango has no DPMS dispatch action; wire an idle daemon that speaks wlr-output-power-management-v1 instead")
        return false
    }

    function quit() {
        return dispatch("quit")
    }

    // ========== WORKSPACE NAVIGATION ==========
    function focusWorkspaceUp() { return dispatch("viewtoleft,0") }
    function focusWorkspaceDown() { return dispatch("viewtoright,0") }

    // ========== WINDOW FOCUS ==========
    function focusColumnLeft() { return dispatch("focusdir,left") }
    function focusColumnRight() { return dispatch("focusdir,right") }

    // ========== KEYBOARD ==========
    // mango's switch_keyboard_layout takes a ONE-based layout number, and
    // treats 0 (or anything out of range) as "advance to the next layout".
    // Passing a zero-based index here silently selects the wrong layout: 0
    // cycles instead of picking the first, and 1 picks the first instead of the
    // second.
    function switchLayout() {
        // Let mango do the cycling: it reads the live xkb state, so this stays
        // correct even when our layout-name list is out of date.
        return dispatch("switch_keyboard_layout,0")
    }
    function switchLayoutPrevious() {
        const count = keyboardLayoutNames.length
        if (count < 2)
            return false
        const prevZeroBased = (currentKeyboardLayoutIndex - 1 + count) % count
        return dispatch("switch_keyboard_layout," + (prevZeroBased + 1))
    }
    function getCurrentKeyboardLayoutName() {
        return currentKeyboardLayout
    }

    function getCurrentOutputWorkspaceNumbers() {
        return currentOutputWorkspaces.map(w => w.idx)
    }

    function getCurrentWorkspaceNumber() {
        if (focusedWorkspaceIndex >= 0 && focusedWorkspaceIndex < allWorkspaces.length)
            return allWorkspaces[focusedWorkspaceIndex].idx
        return 1
    }

    // ========== TOPLEVEL <-> CLIENT MATCHING (mirrors NiriService) ==========

    function matchToplevelToWindow(toplevel, mangoWindow) {
        if (toplevel.appId !== mangoWindow.app_id)
            return 0
        let score = 1
        if (mangoWindow.title && toplevel.title) {
            if (toplevel.title === mangoWindow.title) score = 3
            else if (toplevel.title.includes(mangoWindow.title) || mangoWindow.title.includes(toplevel.title)) score = 2
        }
        return score
    }

    function enrichToplevel(toplevel, mangoWindow) {
        const windowId = mangoWindow.id
        const enriched = {
            "appId": toplevel.appId,
            "title": toplevel.title,
            "activated": !!mangoWindow.is_focused,
            "mangoWindowId": windowId,
            "mangoWorkspaceId": mangoWindow.workspace_id,
            "_sourceKey": `mango:${windowId}`,
            "_sourceToplevel": toplevel,
            "activate": function () { return MangoService.focusWindow(windowId) },
            "close": function () { return toplevel.close ? toplevel.close() : false }
        }
        for (let prop in toplevel) {
            if (!(prop in enriched)) enriched[prop] = toplevel[prop]
        }
        return enriched
    }

    // Best-effort map from a Wayland foreign-toplevel handle back to the mango
    // client it represents. mango's IPC exposes no handle identity, so this
    // matches on app id + title, same as sortToplevels().
    function findWindowForToplevel(toplevel) {
        if (!toplevel || !CompositorService.isMango)
            return null
        let best = null
        let bestScore = 0
        for (const win of root.windows) {
            const score = root.matchToplevelToWindow(toplevel, win)
            if (score > bestScore) {
                bestScore = score
                best = win
                if (score === 3) break
            }
        }
        return best
    }

    function sortToplevels(toplevels) {
        if (!toplevels || !CompositorService.isMango)
            return toplevels ? [...toplevels] : []
        if (windows.length === 0)
            return []

        const usedToplevels = new Set()
        const enrichedToplevels = []

        for (const win of root.windows) {
            let bestMatch = null
            let bestScore = -1
            for (const toplevel of toplevels) {
                if (usedToplevels.has(toplevel)) continue
                const score = matchToplevelToWindow(toplevel, win)
                if (score > bestScore) {
                    bestScore = score
                    bestMatch = toplevel
                    if (score === 3) break
                }
            }
            if (!bestMatch || bestScore <= 0) continue
            usedToplevels.add(bestMatch)
            enrichedToplevels.push(enrichToplevel(bestMatch, win))
        }
        return enrichedToplevels
    }

    function filterCurrentWorkspace(toplevels, screenName) {
        let currentTagIndex = null
        for (const ws of allWorkspaces) {
            if (ws.output === screenName && ws.is_active) {
                currentTagIndex = ws.idx
                break
            }
        }
        if (currentTagIndex === null)
            return toplevels

        const wsWindows = root.windows.filter(w => {
            return w.monitor === screenName && Array.isArray(w.tags) && w.tags.includes(currentTagIndex)
        })

        const usedToplevels = new Set()
        const result = []
        for (const win of wsWindows) {
            let bestMatch = null
            let bestScore = -1
            for (const toplevel of toplevels) {
                if (usedToplevels.has(toplevel)) continue
                const score = matchToplevelToWindow(toplevel, win)
                if (score > bestScore) {
                    bestScore = score
                    bestMatch = toplevel
                    if (score === 3) break
                }
            }
            if (!bestMatch || bestScore <= 0) continue
            usedToplevels.add(bestMatch)
            result.push(enrichToplevel(bestMatch, win))
        }
        return result
    }
}
