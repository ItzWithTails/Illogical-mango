import qs.modules.common
import qs.modules.common.widgets
import QtQuick
import QtQuick.Controls.Material
import QtQuick.Controls

/**
 * Material 3 styled TextField (filled style)
 * https://m3.material.io/components/text-fields/overview
 */
TextField {
    id: root
    Material.theme: Material.System
    Material.accent: Appearance.regaliaEverywhere ? "transparent" : Appearance.colors.colPrimary
    Material.primary: Appearance.colors.colPrimary
    Material.background: Appearance.regaliaEverywhere ? "transparent" : Appearance.colors.colLayer1
    Material.foreground: Appearance.regaliaEverywhere ? Appearance.regalia.onColor : Appearance.colors.colOnSurface
    Material.containerStyle: Appearance.regaliaEverywhere ? Material.Filled : Material.Outlined
    renderType: Text.NativeRendering

    RegaliaControlFace {
        anchors.fill: parent
        z: -0.5
        visible: Appearance.regaliaEverywhere
        fillColor: root.activeFocus
            ? Appearance.regalia.controlPlateHover
            : Appearance.regalia.controlPlate
        radius: Appearance.regalia.roundSmall
        hovered: root.hovered && !root.activeFocus
    }

    // Settings search integration
    property bool enableSettingsSearch: true
    property int settingsSearchOptionId: -1

    function _findSettingsContext() {
        var page = null;
        var sectionTitle = "";
        var groupTitle = "";
        var p = root.parent;
        while (p) {
            if (!page && p.hasOwnProperty("settingsPageIndex")) {
                page = p;
            }
            if (p.hasOwnProperty("title")) {
                if (!sectionTitle && p.hasOwnProperty("icon")) {
                    sectionTitle = p.title;
                } else if (!groupTitle && !p.hasOwnProperty("icon")) {
                    groupTitle = p.title;
                }
            }
            p = p.parent;
        }
        return { page: page, sectionTitle: sectionTitle, groupTitle: groupTitle };
    }

    function focusFromSettingsSearch() {
        var p = root.parent;
        while (p) {
            if (p.hasOwnProperty("expanded") && p.hasOwnProperty("collapsible")) {
                p.expanded = true;
                break;
            }
            p = p.parent;
        }
        root.forceActiveFocus();
    }

    Component.onCompleted: {
        if (!enableSettingsSearch)
            return;
        if (typeof SettingsSearchRegistry === "undefined")
            return;

        var ctx = _findSettingsContext();
        var page = ctx.page;
        if (!page || page.settingsPageIndex === undefined || page.settingsPageIndex < 0)
            return;
        var sectionTitle = ctx.sectionTitle;
        var label = root.placeholderText || root.text || ctx.groupTitle || sectionTitle;

        settingsSearchOptionId = SettingsSearchRegistry.registerOption({
            control: root,
            pageIndex: page && page.settingsPageIndex !== undefined ? page.settingsPageIndex : -1,
            pageName: page && page.settingsPageName ? page.settingsPageName : "",
            section: sectionTitle,
            label: label,
            description: "",
            keywords: []
        });
    }

    Component.onDestruction: {
        if (typeof SettingsSearchRegistry !== "undefined") {
            SettingsSearchRegistry.unregisterControl(root);
        }
    }

    selectedTextColor: Appearance.regaliaEverywhere ? Appearance.regalia.primaryPlateInk : Appearance.colors.colOnSecondaryContainer
    selectionColor: Appearance.regaliaEverywhere ? Appearance.regalia.primaryPlate : Appearance.colors.colSecondaryContainer
    placeholderTextColor: Appearance.regaliaEverywhere ? Appearance.regalia.onMuted : Appearance.colors.colOutline
    clip: true

    font {
        family: Appearance.font.family.main
        pixelSize: Appearance?.font.pixelSize.small ?? 15
        hintingPreference: Font.PreferFullHinting
        variableAxes: Appearance.font.variableAxes.main
    }
    wrapMode: TextEdit.Wrap

    MouseArea {
        anchors.fill: parent
        acceptedButtons: Qt.NoButton
        hoverEnabled: true
        cursorShape: Qt.IBeamCursor
    }

    TextInputContextMenu {
        target: root
    }
}
