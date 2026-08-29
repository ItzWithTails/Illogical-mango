pragma ComponentBehavior: Bound
import QtQuick
import Quickshell
import Quickshell.Services.SystemTray
import Quickshell.Widgets
import Qt5Compat.GraphicalEffects
import qs.services
import qs.modules.barM3
import qs.modules.common
import qs.modules.common.widgets
import qs.modules.common.functions

MouseArea {
    id: root
    required property SystemTrayItem item
    property bool targetMenuOpen: false
    readonly property bool tintIcon: Config.options?.bar?.m3?.tray?.monochromeIcons ?? true
    readonly property string iconSource: TrayService.getSafeIcon(root.item)

    signal menuOpened(qsWindow: var)
    signal menuClosed()

    hoverEnabled: true
    acceptedButtons: Qt.LeftButton | Qt.RightButton
    implicitWidth: 20
    implicitHeight: 20
    onPressed: (event) => {
        switch (event.button) {
        case Qt.LeftButton:
            item.activate();
            break;
        case Qt.RightButton:
            if (item.hasMenu)
                if (menu.active && menu.item && typeof menu.item.close === "function")
                    menu.item.close();
                else
                    menu.open();
            break;
        }
        event.accepted = true;
    }
    onEntered: {
        tooltip.text = TrayService.getTooltipForItem(root.item);
    }

    Loader {
        id: menu
        function open() {
            menu.active = true;
        }
        active: false
        sourceComponent: SysTrayMenu {
            Component.onCompleted: this.open();
            trayItemMenuHandle: root.item.menu
            trayItemId: root.item.id
            anchor {
                window: root.QsWindow.window
                item: root
                gravity: Config.options.bar.vertical
                    ? (Config.options.bar.bottom ? Edges.Left : Edges.Right)
                    : (Config.options.bar.bottom ? Edges.Top : Edges.Bottom)
                edges: Config.options.bar.vertical
                    ? (Config.options.bar.bottom ? Edges.Left : Edges.Right)
                    : (Config.options.bar.bottom ? Edges.Top : Edges.Bottom)
                adjustment: Config.options.bar.vertical
                    ? PopupAdjustment.SlideY
                    : PopupAdjustment.SlideX
            }
            onMenuOpened: (window) => root.menuOpened(window);
            onMenuClosed: {
                root.menuClosed();
                menu.active = false;
            }
        }
    }

    IconImage {
        id: trayIcon
        visible: !root.tintIcon
        source: root.iconSource
        anchors.centerIn: parent
        width: parent.width
        height: parent.height
    }

    Loader {
        active: root.tintIcon
        anchors.fill: trayIcon
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
                readonly property color themeColor: M3Palette.pillInk("sysTray")
                hue: themeColor.hslHue >= 0 ? themeColor.hslHue : 0
                saturation: themeColor.hslSaturation
                lightness: (themeColor.hslLightness - 0.5) * 0.35
            }
        }
    }

    MaterialSymbol {
        anchors.centerIn: parent
        visible: root.iconSource.length === 0 || trayIcon.status === Image.Error
        text: "apps"
        iconSize: 18
        color: M3Palette.pillInk("sysTray")
    }

    M3ToolTip {
        id: tooltip
        extraVisibleCondition: root.containsMouse
        alternativeVisibleCondition: extraVisibleCondition
        anchorEdges: (!Config.options.bar.bottom && !Config.options.bar.vertical) ? Edges.Bottom : Edges.Top
    }

}
