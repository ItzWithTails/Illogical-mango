import qs.modules.common
import qs.modules.common.widgets
import qs.modules.common.functions
import qs.services
import QtQuick
import Quickshell
import Quickshell.Services.SystemTray
import Quickshell.Widgets
import Qt5Compat.GraphicalEffects

MouseArea {
    id: root
    required property SystemTrayItem item
    property var trayParent: null  // Reference to SysTray for closing other menus
    property bool targetMenuOpen: false
    readonly property bool monochromeIcon: Config.options?.bar?.tray?.monochromeIcons ?? false
    readonly property string iconSource: TrayService.getSafeIcon(root.item)
    readonly property color themeIconColor: Appearance.angelEverywhere ? Appearance.angel.colText
        : Appearance.regaliaEverywhere ? Appearance.regalia.onColor
        : Appearance.ilmangoEverywhere ? Appearance.ilmango.colText
        : Appearance.colors.colOnLayer0

    signal menuOpened(qsWindow: var)
    signal menuClosed(qsWindow: var)

    hoverEnabled: true
    acceptedButtons: Qt.LeftButton | Qt.MiddleButton | Qt.RightButton
    implicitWidth: 18
    implicitHeight: 18
    onPressed: (event) => {
        switch (event.button) {
        case Qt.LeftButton: {
            // Smart toggle: click to show, click again to minimize
            // Falls back to normal activate() if not handled
            if (!TrayService.smartToggle(item)) {
                item.activate();
            }
            break;
        }
        case Qt.MiddleButton:
            // Middle click: try secondary activate (useful for some apps)
            item.secondaryActivate();
            break;
        case Qt.RightButton:
            if (item.hasMenu) {
                // Close other tray menus first
                if (trayParent) trayParent.closeAllTrayMenus();
                menu.open();
            }
            break;
        }
        event.accepted = true;
    }
    onEntered: {
        if (!item) return;
        const tooltipTitle = item.tooltipTitle ?? "";
        const title = item.title ?? "";
        const tooltipDescription = item.tooltipDescription ?? "";
        
        tooltip.text = tooltipTitle.length > 0 ? tooltipTitle
                : (title.length > 0 ? title : "");
        if (tooltip.text.length === 0) return;
        if (tooltipDescription.length > 0) tooltip.text += " • " + tooltipDescription;
    }

    // Listen for close signal from parent tray
    Connections {
        target: root.trayParent
        enabled: root.trayParent !== null
        function onCloseAllTrayMenus() {
            if (menu.visible)
                menu.close();
        }
    }

    // Let Quickshell render the DBus menu through the platform menu backend.
    // Telegram and Throne populate their menu asynchronously; the old custom
    // PopupWindow opened at its initial 1x1 size and remained a blurred blob.
    QsMenuAnchor {
        id: menu
        menu: root.item.menu
        anchor {
            item: root
            edges: (Config.options?.bar?.vertical ?? false)
                ? ((Config.options?.bar?.bottom ?? false) ? Edges.Left : Edges.Right)
                : ((Config.options?.bar?.bottom ?? false) ? Edges.Top : Edges.Bottom)
            gravity: (Config.options?.bar?.vertical ?? false)
                ? ((Config.options?.bar?.bottom ?? false) ? Edges.Left : Edges.Right)
                : ((Config.options?.bar?.bottom ?? false) ? Edges.Top : Edges.Bottom)
            adjustment: PopupAdjustment.FlipX | PopupAdjustment.FlipY
                | PopupAdjustment.SlideX | PopupAdjustment.SlideY
        }
        onClosed: root.menuClosed(null)
    }

    IconImage {
        id: trayIcon
        visible: !root.monochromeIcon
        source: root.iconSource
        anchors.centerIn: parent
        width: parent.width
        height: parent.height
    }

    Loader {
        active: root.monochromeIcon
        anchors.centerIn: parent
        width: root.width
        height: root.height
        sourceComponent: Item {
            IconImage {
                id: tintedIcon
                visible: false
                anchors.fill: parent
                source: root.iconSource
            }
            Colorize {
                anchors.fill: tintedIcon
                source: tintedIcon
                hue: root.themeIconColor.hslHue >= 0 ? root.themeIconColor.hslHue : 0
                saturation: root.themeIconColor.hslSaturation
                lightness: (root.themeIconColor.hslLightness - 0.5) * 0.35
            }
        }
    }

    MaterialSymbol {
        anchors.centerIn: parent
        visible: root.iconSource.length === 0 || trayIcon.status === Image.Error
        text: "apps"
        iconSize: 17
        color: Appearance.angelEverywhere ? Appearance.angel.colText
            : Appearance.ilmangoEverywhere ? Appearance.ilmango.colText
            : Appearance.colors.colOnLayer0
    }

    PopupToolTip {
        id: tooltip
        extraVisibleCondition: root.containsMouse
        alternativeVisibleCondition: extraVisibleCondition
        anchorEdges: (Config.options?.bar?.vertical ?? false)
            ? ((Config.options?.bar?.bottom ?? false) ? Edges.Left : Edges.Right)
            : ((Config.options?.bar?.bottom ?? false) ? Edges.Top : Edges.Bottom)
    }

}
