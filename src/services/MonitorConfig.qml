pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import Quickshell.Io
import qs.services

Singleton {
    id: root

    property var outputs: []
    property bool loading: false
    property bool applying: false
    property string error: ""
    property string message: ""
    readonly property bool supported: CompositorService.isMango

    signal applied(bool success)

    function helperCommand(action, payload): var {
        const command = ["python3", Quickshell.shellPath("scripts/monitor-config.py"), action]
        if (payload !== undefined)
            command.push(JSON.stringify(payload))
        return command
    }

    function refresh(): void {
        if (!root.supported || statusProcess.running)
            return
        root.loading = true
        root.error = ""
        statusProcess.command = root.helperCommand("status")
        statusProcess.running = true
    }

    function apply(outputList): void {
        if (!root.supported || applyProcess.running)
            return
        root.applying = true
        root.error = ""
        root.message = ""
        applyProcess.command = root.helperCommand("apply", { outputs: outputList })
        applyProcess.running = true
    }

    function parseResult(raw, exitCode): var {
        try {
            const parsed = JSON.parse(String(raw ?? "").trim())
            if (!parsed.ok)
                root.error = String(parsed.error ?? "Unknown monitor configuration error")
            return parsed
        } catch (e) {
            root.error = String(raw ?? "").trim() || ("Monitor helper exited with code " + exitCode)
            return { ok: false }
        }
    }

    Process {
        id: statusProcess
        stdout: StdioCollector { id: statusOutput }
        stderr: StdioCollector { id: statusError }
        onExited: exitCode => {
            root.loading = false
            const result = root.parseResult(statusOutput.text, exitCode)
            if (result.ok)
                root.outputs = result.outputs ?? []
            else if (!root.error)
                root.error = statusError.text.trim()
        }
    }

    Process {
        id: applyProcess
        stdout: StdioCollector { id: applyOutput }
        stderr: StdioCollector { id: applyError }
        onExited: exitCode => {
            root.applying = false
            const result = root.parseResult(applyOutput.text, exitCode)
            if (result.ok) {
                root.message = String(result.message ?? "")
                root.applied(true)
                refreshDelay.restart()
            } else {
                if (!root.error)
                    root.error = applyError.text.trim()
                root.applied(false)
            }
        }
    }

    Timer {
        id: refreshDelay
        interval: 700
        repeat: false
        onTriggered: root.refresh()
    }

    Component.onCompleted: root.refresh()
}
