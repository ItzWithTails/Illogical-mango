pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import Quickshell.Io
import Quickshell.Hyprland
import qs.services

/**
 * Provides access to some Hyprland data not available in Quickshell.Hyprland.
 */
Singleton {
    id: root

    // ---------- mango compat shim ----------
    // A lot of consumers read HyprlandData.* unconditionally (no compositor
    // check at all: Overview, Lock, Session, RegionSelection...). Rather than
    // teach every one of those about mango, we translate MangoService's tag
    // model into hyprctl's shape here, once.
    //
    // Workspace id == the bare 1-based tag index. Hyprland ids are globally
    // unique, mango's tags genuinely are not (each monitor has its own 1..N),
    // so on a multi-head setup two tags can share an id. Encoding the monitor
    // into the id instead (monitorIndex*100 + tag) was worse: it produced ids
    // like 101..109 that break the "ids start at 1" arithmetic every consumer
    // does (workspace grouping, displayed numbers). Consumers that must
    // disambiguate filter on the window's `monitor` field as well.
    readonly property var _mangoMonitorIndex: {
        const idx = ({})
        if (!CompositorService.isMango) return idx
        const names = Object.keys(MangoService.outputs).sort()
        for (let i = 0; i < names.length; i++) idx[names[i]] = i
        return idx
    }

    function _mangoWorkspaceId(monitorName, tagIndex) {
        return tagIndex
    }

    readonly property var _mangoWindowList: {
        if (!CompositorService.isMango) return []
        return MangoService.windows.map(w => {
            const tagIdx = (w.tags && w.tags.length > 0) ? w.tags[0] : 0
            const appId = w.appid || w.app_id || ""
            return {
                address: "mango:" + w.id,
                class: appId,
                initialClass: appId,
                title: w.title || "",
                workspace: { id: root._mangoWorkspaceId(w.monitor, tagIdx), name: String(tagIdx) },
                monitor: root._mangoMonitorIndex[w.monitor] ?? 0,
                floating: !!w.is_floating,
                fullscreen: !!w.is_fullscreen,
                size: [w.width, w.height],
                at: [w.x, w.y],
                pid: w.pid,
                xwayland: !!w.is_xwayland
            }
        })
    }

    readonly property var _mangoMonitors: {
        if (!CompositorService.isMango) return []
        const names = Object.keys(MangoService.outputs).sort()
        return names.map((name, idx) => {
            const mon = MangoService.outputs[name]
            const activeTag = (mon.active_tags && mon.active_tags.length > 0) ? mon.active_tags[0] : 1
            return {
                id: idx,
                name,
                activeWorkspace: { id: root._mangoWorkspaceId(name, activeTag), name: String(activeTag) },
                x: mon.x, y: mon.y, width: mon.width, height: mon.height, scale: mon.scale,
                focused: !!mon.active,
                // mango's IPC exposes neither layer-shell exclusive zones nor
                // output transform, so report the neutral values consumers
                // (Overview geometry) expect rather than leaving them undefined.
                reserved: [0, 0, 0, 0],
                transform: 0
            }
        })
    }

    readonly property var _mangoWorkspaces: {
        if (!CompositorService.isMango) return []
        return MangoService.allWorkspaces.map(ws => ({
            id: root._mangoWorkspaceId(ws.output, ws.idx),
            name: String(ws.idx),
            monitor: ws.output,
            windows: ws.client_count
        }))
    }

    readonly property var _mangoActiveWorkspace: {
        if (!CompositorService.isMango) return null
        const ws = MangoService.workspaces[MangoService.focusedWorkspaceId]
        if (!ws) return null
        return { id: root._mangoWorkspaceId(ws.output, ws.idx), name: String(ws.idx), monitor: ws.output }
    }
    // ---------- end mango compat shim ----------

    property var windowList: CompositorService.isMango ? _mangoWindowList : []
    property var addresses: CompositorService.isMango ? _mangoWindowList.map(w => w.address) : []
    property var windowByAddress: {
        if (!CompositorService.isMango) return ({})
        const m = {}
        for (const w of _mangoWindowList) m[w.address] = w
        return m
    }
    property var workspaces: CompositorService.isMango ? _mangoWorkspaces : []
    property var workspaceIds: CompositorService.isMango ? _mangoWorkspaces.map(w => w.id) : []
    property var workspaceById: {
        if (!CompositorService.isMango) return ({})
        const m = {}
        for (const w of _mangoWorkspaces) m[w.id] = w
        return m
    }
    property var activeWorkspace: CompositorService.isMango ? _mangoActiveWorkspace : null
    property var monitors: CompositorService.isMango ? _mangoMonitors : []
    property var layers: ({}) // no layer-shell introspection over mango's IPC yet

    function updateWindowList() {
        if (!CompositorService.isHyprland)
            return;
        getClients.running = true;
    }

    function updateLayers() {
        if (!CompositorService.isHyprland)
            return;
        getLayers.running = true;
    }

    function updateMonitors() {
        if (!CompositorService.isHyprland)
            return;
        getMonitors.running = true;
    }

    function updateWorkspaces() {
        if (!CompositorService.isHyprland)
            return;
        getWorkspaces.running = true;
        getActiveWorkspace.running = true;
    }

    function updateAll() {
        if (!CompositorService.isHyprland)
            return;
        updateWindowList();
        updateMonitors();
        updateLayers();
        updateWorkspaces();
    }

    // Resolve a Wayland toplevel handle to the key used in `windowByAddress`.
    // Hyprland stamps its own address onto the handle; mango has no such
    // identity over IPC, so MangoService matches on app id + title instead.
    function addressForToplevel(toplevel) {
        if (CompositorService.isMango) {
            const win = MangoService.findWindowForToplevel(toplevel)
            return win ? ("mango:" + win.id) : ""
        }
        return `0x${toplevel?.HyprlandToplevel?.address}`
    }

    function biggestWindowForWorkspace(workspaceId) {
        const windowsInThisWorkspace = HyprlandData.windowList.filter(w => w.workspace.id == workspaceId);
        return windowsInThisWorkspace.reduce((maxWin, win) => {
            const maxArea = (maxWin?.size?.[0] ?? 0) * (maxWin?.size?.[1] ?? 0);
            const winArea = (win?.size?.[0] ?? 0) * (win?.size?.[1] ?? 0);
            return winArea > maxArea ? win : maxWin;
        }, null);
    }

    Component.onCompleted: {
        if (!CompositorService.isHyprland)
            return;
        updateAll();
    }

    Connections {
        target: Hyprland
        enabled: CompositorService.isHyprland

        function onRawEvent(event) {
            // console.log("Hyprland raw event:", event.name);
            updateAll()
        }
    }

    Process {
        id: getClients
        command: ["/usr/bin/hyprctl", "clients", "-j"]
        stdout: StdioCollector {
            id: clientsCollector
            onStreamFinished: {
                try {
                    root.windowList = JSON.parse(clientsCollector.text)
                } catch (e) {
                    console.log("[HyprlandData] Failed to parse clients JSON:", e);
                    root.windowList = [];
                }
                let tempWinByAddress = {};
                for (var i = 0; i < root.windowList.length; ++i) {
                    var win = root.windowList[i];
                    tempWinByAddress[win.address] = win;
                }
                root.windowByAddress = tempWinByAddress;
                root.addresses = root.windowList.map(win => win.address);
            }
        }
    }

    Process {
        id: getMonitors
        command: ["/usr/bin/hyprctl", "monitors", "-j"]
        stdout: StdioCollector {
            id: monitorsCollector
            onStreamFinished: {
                try {
                    root.monitors = JSON.parse(monitorsCollector.text);
                } catch (e) {
                    console.log("[HyprlandData] Failed to parse monitors JSON:", e);
                    root.monitors = [];
                }
            }
        }
    }

    Process {
        id: getLayers
        command: ["/usr/bin/hyprctl", "layers", "-j"]
        stdout: StdioCollector {
            id: layersCollector
            onStreamFinished: {
                try {
                    root.layers = JSON.parse(layersCollector.text);
                } catch (e) {
                    console.log("[HyprlandData] Failed to parse layers JSON:", e);
                    root.layers = {};
                }
            }
        }
    }

    Process {
        id: getWorkspaces
        command: ["/usr/bin/hyprctl", "workspaces", "-j"]
        stdout: StdioCollector {
            id: workspacesCollector
            onStreamFinished: {
                try {
                    root.workspaces = JSON.parse(workspacesCollector.text);
                } catch (e) {
                    console.log("[HyprlandData] Failed to parse workspaces JSON:", e);
                    root.workspaces = [];
                }
                let tempWorkspaceById = {};
                for (var i = 0; i < root.workspaces.length; ++i) {
                    var ws = root.workspaces[i];
                    tempWorkspaceById[ws.id] = ws;
                }
                root.workspaceById = tempWorkspaceById;
                root.workspaceIds = root.workspaces.map(ws => ws.id);
            }
        }
    }

    Process {
        id: getActiveWorkspace
        command: ["/usr/bin/hyprctl", "activeworkspace", "-j"]
        stdout: StdioCollector {
            id: activeWorkspaceCollector
            onStreamFinished: {
                try {
                    root.activeWorkspace = JSON.parse(activeWorkspaceCollector.text);
                } catch (e) {
                    console.log("[HyprlandData] Failed to parse active workspace JSON:", e);
                    root.activeWorkspace = null;
                }
            }
        }
    }
}
