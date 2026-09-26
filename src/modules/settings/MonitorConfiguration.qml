pragma ComponentBehavior: Bound

import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import qs.services
import qs.modules.common
import qs.modules.common.widgets

ColumnLayout {
    id: root
    Layout.fillWidth: true
    spacing: Appearance.sizes.spacingMedium

    property var draftOutputs: []
    property bool dirty: false
    property bool initialized: false
    readonly property var enabledOutputs: draftOutputs.filter(output => output.enabled)
    readonly property string displayMode: root.detectDisplayMode()

    function cloneOutputs(outputs): var { return JSON.parse(JSON.stringify(outputs ?? [])) }
    function loadOutputs(): void {
        root.draftOutputs = root.cloneOutputs(MonitorConfig.outputs)
        root.dirty = false
        root.initialized = true
    }
    function outputIndex(name): int { return root.draftOutputs.findIndex(output => output.name === name) }
    function updateOutput(name, values): void {
        const list = root.cloneOutputs(root.draftOutputs)
        const index = list.findIndex(output => output.name === name)
        if (index < 0) return
        Object.assign(list[index], values)
        root.draftOutputs = list
        root.dirty = true
    }
    function logicalWidth(output): real {
        return (Number(output.transform ?? 0) % 2 === 1 ? output.height : output.width)
            / Math.max(0.5, Number(output.scale ?? 1))
    }
    function logicalHeight(output): real {
        return (Number(output.transform ?? 0) % 2 === 1 ? output.width : output.height)
            / Math.max(0.5, Number(output.scale ?? 1))
    }
    function modeKey(mode): string {
        return mode.width + "x" + mode.height + "@" + Number(mode.refresh).toFixed(2)
    }
    function currentModeKey(output): string {
        return output.width + "x" + output.height + "@" + Number(output.refresh).toFixed(2)
    }
    function modeOptions(output): var {
        return (output?.modes ?? []).map(mode => ({
            value: root.modeKey(mode),
            displayName: mode.width + " × " + mode.height + "  ·  "
                + Number(mode.refresh).toFixed(mode.refresh % 1 === 0 ? 0 : 2) + " Hz"
                + (mode.preferred ? "  (" + Translation.tr("recommended") + ")" : "")
        }))
    }
    function selectMode(name, key): void {
        const index = root.outputIndex(name)
        if (index < 0) return
        const mode = (root.draftOutputs[index].modes ?? []).find(item => root.modeKey(item) === key)
        if (mode) root.updateOutput(name, { width: mode.width, height: mode.height, refresh: mode.refresh })
    }
    function detectDisplayMode(): string {
        if (!root.initialized || root.enabledOutputs.length === 0) return "custom"
        if (root.enabledOutputs.length === root.draftOutputs.length) return "extend"
        if (root.enabledOutputs.length === 1) return root.enabledOutputs[0].internal ? "internal" : "external"
        return "custom"
    }
    function chooseDisplayMode(mode): void {
        const internal = root.draftOutputs.find(output => output.internal)
        const external = root.draftOutputs.find(output => !output.internal)
        const list = root.cloneOutputs(root.draftOutputs)
        for (let i = 0; i < list.length; i++) {
            if (mode === "extend") list[i].enabled = true
            else if (mode === "internal") list[i].enabled = internal && list[i].name === internal.name
            else if (mode === "external") list[i].enabled = external && list[i].name === external.name
            if (list[i].enabled) { list[i].x = 0; list[i].y = 0 }
        }
        root.draftOutputs = list
        root.dirty = true
        if (mode === "extend") root.arrange("horizontal")
    }
    function arrange(direction): void {
        const list = root.cloneOutputs(root.draftOutputs)
        let cursor = 0
        for (let i = 0; i < list.length; i++) {
            if (!list[i].enabled) continue
            if (direction === "horizontal") {
                list[i].x = Math.round(cursor); list[i].y = 0
                cursor += root.logicalWidth(list[i])
            } else {
                list[i].x = 0; list[i].y = Math.round(cursor)
                cursor += root.logicalHeight(list[i])
            }
        }
        root.draftOutputs = list
        root.dirty = true
    }
    function minX(): real { return root.enabledOutputs.length ? Math.min(...root.enabledOutputs.map(output => output.x)) : 0 }
    function minY(): real { return root.enabledOutputs.length ? Math.min(...root.enabledOutputs.map(output => output.y)) : 0 }
    function layoutWidth(): real {
        if (!root.enabledOutputs.length) return 1
        return Math.max(1, Math.max(...root.enabledOutputs.map(output => output.x + root.logicalWidth(output))) - root.minX())
    }
    function layoutHeight(): real {
        if (!root.enabledOutputs.length) return 1
        return Math.max(1, Math.max(...root.enabledOutputs.map(output => output.y + root.logicalHeight(output))) - root.minY())
    }

    Connections {
        target: MonitorConfig
        function onOutputsChanged() { if (!root.dirty || !root.initialized) root.loadOutputs() }
        function onApplied(success) { if (success) root.dirty = false }
    }
    Component.onCompleted: {
        if (MonitorConfig.outputs.length > 0) root.loadOutputs()
        else MonitorConfig.refresh()
    }

    NoticeBox {
        visible: !MonitorConfig.supported
        Layout.fillWidth: true
        materialIcon: "info"
        text: Translation.tr("Display configuration is currently available in MangoWM sessions.")
    }
    RowLayout {
        visible: MonitorConfig.supported
        Layout.fillWidth: true
        StyledText {
            Layout.fillWidth: true
            text: Translation.tr("Display mode")
            font.weight: Font.Medium
            color: Appearance.colors.colOnLayer1
        }
        RippleButtonWithIcon {
            materialIcon: "refresh"
            mainText: Translation.tr("Refresh")
            enabled: !MonitorConfig.loading && !MonitorConfig.applying
            onClicked: MonitorConfig.refresh()
        }
    }
    ConfigSelectionArray {
        visible: MonitorConfig.supported
        enableSettingsSearch: false
        currentValue: root.displayMode
        options: [
            { value: "extend", displayName: Translation.tr("Extend"), icon: "desktop_windows" },
            { value: "internal", displayName: Translation.tr("Built-in only"), icon: "laptop" },
            { value: "external", displayName: Translation.tr("External only"), icon: "monitor" }
        ]
        onSelected: value => root.chooseDisplayMode(value)
    }
    NoticeBox {
        visible: MonitorConfig.error.length > 0
        Layout.fillWidth: true
        materialIcon: "error"
        text: MonitorConfig.error
    }

    Rectangle {
        id: arrangementCanvas
        visible: MonitorConfig.supported && root.draftOutputs.length > 0
        Layout.fillWidth: true
        implicitHeight: 260
        radius: Appearance.rounding.small
        color: Appearance.colors.colLayer1
        border.width: 1
        border.color: SettingsMaterialPreset.groupBorderColor
        clip: true
        readonly property real inset: 22
        readonly property real scaleFactor: Math.min(
            Math.max(0.02, (width - inset * 2) / root.layoutWidth()),
            Math.max(0.02, (height - inset * 2) / root.layoutHeight()), 0.22)
        readonly property real contentWidth: root.layoutWidth() * scaleFactor
        readonly property real contentHeight: root.layoutHeight() * scaleFactor
        readonly property real originX: (width - contentWidth) / 2 - root.minX() * scaleFactor
        readonly property real originY: (height - contentHeight) / 2 - root.minY() * scaleFactor

        Repeater {
            model: root.enabledOutputs
            Rectangle {
                id: monitorTile
                required property var modelData
                property real dragStartX: 0
                property real dragStartY: 0
                property real pendingX: Number(modelData.x)
                property real pendingY: Number(modelData.y)
                x: arrangementCanvas.originX + (dragHandler.active ? pendingX : modelData.x) * arrangementCanvas.scaleFactor
                y: arrangementCanvas.originY + (dragHandler.active ? pendingY : modelData.y) * arrangementCanvas.scaleFactor
                width: Math.max(90, root.logicalWidth(modelData) * arrangementCanvas.scaleFactor)
                height: Math.max(58, root.logicalHeight(modelData) * arrangementCanvas.scaleFactor)
                radius: Appearance.rounding.small
                color: dragHandler.active ? Appearance.colors.colPrimaryContainer : Appearance.colors.colSecondaryContainer
                border.width: 2
                border.color: dragHandler.active ? Appearance.colors.colPrimary : Appearance.colors.colSecondary
                ColumnLayout {
                    anchors.centerIn: parent
                    width: parent.width - Appearance.sizes.spacingMedium
                    spacing: 1
                    MaterialSymbol {
                        Layout.alignment: Qt.AlignHCenter
                        text: modelData.internal ? "laptop" : "monitor"
                        iconSize: Appearance.font.pixelSize.larger
                        color: Appearance.colors.colOnSecondaryContainer
                    }
                    StyledText {
                        Layout.fillWidth: true
                        horizontalAlignment: Text.AlignHCenter
                        text: modelData.name
                        font.weight: Font.Medium
                        elide: Text.ElideRight
                        color: Appearance.colors.colOnSecondaryContainer
                    }
                    StyledText {
                        Layout.fillWidth: true
                        horizontalAlignment: Text.AlignHCenter
                        text: Math.round(dragHandler.active ? monitorTile.pendingX : modelData.x)
                            + ", " + Math.round(dragHandler.active ? monitorTile.pendingY : modelData.y)
                        font.pixelSize: Appearance.font.pixelSize.smaller
                        color: Appearance.colors.colOnSecondaryContainer
                    }
                }
                DragHandler {
                    id: dragHandler
                    target: null
                    cursorShape: Qt.ClosedHandCursor
                    onActiveChanged: {
                        if (active) {
                            monitorTile.dragStartX = Number(monitorTile.modelData.x)
                            monitorTile.dragStartY = Number(monitorTile.modelData.y)
                            monitorTile.pendingX = monitorTile.dragStartX
                            monitorTile.pendingY = monitorTile.dragStartY
                        } else if (monitorTile.pendingX !== monitorTile.dragStartX
                                || monitorTile.pendingY !== monitorTile.dragStartY) {
                            root.updateOutput(monitorTile.modelData.name,
                                { x: monitorTile.pendingX, y: monitorTile.pendingY })
                        }
                    }
                    onTranslationChanged: if (active) {
                        const grid = 10
                        const nextX = Math.round((monitorTile.dragStartX + translation.x / arrangementCanvas.scaleFactor) / grid) * grid
                        const nextY = Math.round((monitorTile.dragStartY + translation.y / arrangementCanvas.scaleFactor) / grid) * grid
                        monitorTile.pendingX = nextX
                        monitorTile.pendingY = nextY
                    }
                }
            }
        }
    }

    RowLayout {
        visible: MonitorConfig.supported && root.draftOutputs.length > 1
        Layout.fillWidth: true
        spacing: Appearance.sizes.spacingSmall
        StyledText {
            Layout.fillWidth: true
            text: Translation.tr("Drag displays above, or use a preset")
            color: Appearance.colors.colSubtext
            font.pixelSize: Appearance.font.pixelSize.smaller
        }
        RippleButtonWithIcon { materialIcon: "view_column"; mainText: Translation.tr("Side by side"); onClicked: root.arrange("horizontal") }
        RippleButtonWithIcon { materialIcon: "view_agenda"; mainText: Translation.tr("Stacked"); onClicked: root.arrange("vertical") }
    }

    Repeater {
        model: root.draftOutputs
        Rectangle {
            id: outputCard
            required property var modelData
            Layout.fillWidth: true
            implicitHeight: outputLayout.implicitHeight + Appearance.sizes.spacingLarge * 2
            radius: Appearance.rounding.small
            color: Appearance.colors.colLayer1
            border.width: 1
            border.color: modelData.enabled ? Appearance.colors.colPrimary : SettingsMaterialPreset.groupBorderColor
            ColumnLayout {
                id: outputLayout
                anchors.fill: parent
                anchors.margins: Appearance.sizes.spacingLarge
                spacing: Appearance.sizes.spacingSmall
                RowLayout {
                    Layout.fillWidth: true
                    MaterialSymbol { text: outputCard.modelData.internal ? "laptop" : "monitor"; color: Appearance.colors.colPrimary }
                    StyledText {
                        Layout.fillWidth: true
                        text: outputCard.modelData.name
                        font.weight: Font.Medium
                        color: Appearance.colors.colOnLayer1
                    }
                    SettingsSwitch {
                        implicitWidth: 90
                        enableSettingsSearch: false
                        autoToggle: false
                        text: outputCard.modelData.enabled ? Translation.tr("Enabled") : Translation.tr("Disabled")
                        checked: outputCard.modelData.enabled
                        onToggledByUser: checked => root.updateOutput(outputCard.modelData.name, { enabled: checked })
                    }
                }
                GridLayout {
                    Layout.fillWidth: true
                    columns: 2
                    columnSpacing: Appearance.sizes.spacingMedium
                    rowSpacing: Appearance.sizes.spacingSmall
                    StyledText { text: Translation.tr("Resolution and refresh rate"); color: Appearance.colors.colSubtext }
                    StyledComboBox {
                        Layout.fillWidth: true
                        enabled: outputCard.modelData.enabled
                        model: root.modeOptions(outputCard.modelData)
                        textRole: "displayName"; valueRole: "value"
                        currentIndex: {
                            const options = root.modeOptions(outputCard.modelData)
                            return Math.max(0, options.findIndex(option => option.value === root.currentModeKey(outputCard.modelData)))
                        }
                        onActivated: index => {
                            const options = root.modeOptions(outputCard.modelData)
                            if (index >= 0 && index < options.length) root.selectMode(outputCard.modelData.name, options[index].value)
                        }
                    }
                    StyledText { text: Translation.tr("Scale"); color: Appearance.colors.colSubtext }
                    StyledComboBox {
                        Layout.fillWidth: true
                        enabled: outputCard.modelData.enabled
                        model: [
                            { value: 1, displayName: "100%" }, { value: 1.25, displayName: "125%" },
                            { value: 1.5, displayName: "150%" }, { value: 1.75, displayName: "175%" },
                            { value: 2, displayName: "200%" }
                        ]
                        textRole: "displayName"; valueRole: "value"
                        currentIndex: Math.max(0, model.findIndex(option => option.value === Number(outputCard.modelData.scale)))
                        onActivated: index => root.updateOutput(outputCard.modelData.name, { scale: model[index].value })
                    }
                    StyledText { text: Translation.tr("Orientation"); color: Appearance.colors.colSubtext }
                    StyledComboBox {
                        Layout.fillWidth: true
                        enabled: outputCard.modelData.enabled
                        model: [
                            { value: 0, displayName: Translation.tr("Landscape") },
                            { value: 1, displayName: Translation.tr("Portrait left") },
                            { value: 2, displayName: Translation.tr("Landscape flipped") },
                            { value: 3, displayName: Translation.tr("Portrait right") }
                        ]
                        textRole: "displayName"; valueRole: "value"
                        currentIndex: Math.max(0, model.findIndex(option => option.value === Number(outputCard.modelData.transform)))
                        onActivated: index => root.updateOutput(outputCard.modelData.name, { transform: model[index].value })
                    }
                    StyledText { text: Translation.tr("Exact position (X / Y)"); color: Appearance.colors.colSubtext }
                    RowLayout {
                        Layout.fillWidth: true
                        StyledSpinBox {
                            Layout.fillWidth: true; enabled: outputCard.modelData.enabled
                            from: -20000; to: 20000; value: Number(outputCard.modelData.x)
                            onValueModified: root.updateOutput(outputCard.modelData.name, { x: value })
                        }
                        StyledSpinBox {
                            Layout.fillWidth: true; enabled: outputCard.modelData.enabled
                            from: -20000; to: 20000; value: Number(outputCard.modelData.y)
                            onValueModified: root.updateOutput(outputCard.modelData.name, { y: value })
                        }
                    }
                }
            }
        }
    }

    RowLayout {
        visible: MonitorConfig.supported
        Layout.fillWidth: true
        spacing: Appearance.sizes.spacingSmall
        StyledText {
            Layout.fillWidth: true
            text: MonitorConfig.message.length > 0 ? Translation.tr("Saved to Mango config") : ""
            color: Appearance.colors.colPrimary
        }
        RippleButtonWithIcon {
            materialIcon: "undo"; mainText: Translation.tr("Discard")
            enabled: root.dirty && !MonitorConfig.applying
            onClicked: root.loadOutputs()
        }
        RippleButtonWithIcon {
            materialIcon: MonitorConfig.applying ? "progress_activity" : "check"
            mainText: MonitorConfig.applying ? Translation.tr("Applying…") : Translation.tr("Apply")
            enabled: root.dirty && root.enabledOutputs.length > 0 && !MonitorConfig.applying
            colBackground: Appearance.colors.colPrimaryContainer
            onClicked: MonitorConfig.apply(root.draftOutputs)
        }
    }
}
