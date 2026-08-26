import QtQuick
import qs.modules.common

Rectangle {
    id: contentItem
    anchors.fill: parent
    // Overlay no tiene blur de wallpaper, usar colores sólidos en aurora
    color: Appearance.angelEverywhere ? Appearance.angel.colGlassPanel
         : Appearance.ilmangoEverywhere ? Appearance.ilmango.colLayer1
         : Appearance.auroraEverywhere ? Appearance.colors.colLayer2Base
         : Appearance.colors.colSurfaceContainer
}
