import qs
import qs.modules.common
import qs.modules.common.widgets
import qs.services
import qs.modules.common.functions
import QtQuick
import QtQuick.Layouts

RippleButton {
    id: root
    property string query

    implicitHeight: 30
    leftPadding: 6
    rightPadding: 10
    buttonRadius: Appearance.angelEverywhere ? Appearance.angel.roundingSmall
        : Appearance.ilmangoEverywhere ? Appearance.ilmango.roundingSmall : Appearance.rounding.verysmall
    colBackground: Appearance.angelEverywhere ? Appearance.angel.colGlassCard
        : Appearance.ilmangoEverywhere ? Appearance.ilmango.colLayer2 
        : Appearance.auroraEverywhere ? "transparent" : Appearance.colors.colSurfaceContainerHighest
    colBackgroundHover: Appearance.angelEverywhere ? Appearance.angel.colGlassCardHover
        : Appearance.ilmangoEverywhere ? Appearance.ilmango.colLayer2Hover 
        : Appearance.auroraEverywhere ? Appearance.aurora.colSubSurfaceHover : Appearance.colors.colSurfaceContainerHighestHover
    colRipple: Appearance.angelEverywhere ? Appearance.angel.colGlassCardActive
        : Appearance.ilmangoEverywhere ? Appearance.ilmango.colLayer2Active 
        : Appearance.auroraEverywhere ? Appearance.aurora.colSubSurfaceActive : Appearance.colors.colSurfaceContainerHighestActive

    PointingHandInteraction {}
    onClicked: {
        let url = (Config.options?.search?.engineBaseUrl ?? "https://www.google.com/search?q=") + root.query;
        for (let site of (Config?.options?.search.excludedSites ?? [])) {
            url += ` -site:${site}`;
        }
        Qt.openUrlExternally(url);
        GlobalStates.sidebarLeftOpen = false;
    }

    contentItem: Item {
        anchors.centerIn: parent
        implicitWidth: rowLayout.implicitWidth
        implicitHeight: rowLayout.implicitHeight
        RowLayout {
            id: rowLayout
            anchors.centerIn: parent
            spacing: 5
            MaterialSymbol {
                text: "search"
                iconSize: 20
                color: Appearance.ilmangoEverywhere ? Appearance.ilmango.colText : Appearance.colors.colOnSurface
            }
            StyledText {
                id: text
                horizontalAlignment: Text.AlignHCenter
                text: root.query
                color: Appearance.ilmangoEverywhere ? Appearance.ilmango.colText : Appearance.colors.colOnSurface
            }
        }
    }
}
