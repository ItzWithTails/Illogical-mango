pragma ComponentBehavior: Bound

import qs.modules.common
import qs.modules.common.functions
import qs.services
import QtQuick
import Quickshell.Services.Notifications

RippleButton {
    id: button
    property string buttonText: ""
    property var urgency: NotificationUrgency.Normal
    readonly property bool critical: button.urgency === undefined || button.urgency === null
        ? false
        : button.urgency === NotificationUrgency.Critical
            || String(button.urgency).toLowerCase() === "critical"

    implicitHeight: Appearance.regaliaEverywhere ? Appearance.regalia.controlHeight : 34
    leftPadding: 15
    rightPadding: 15
    buttonRadius: Appearance.regaliaEverywhere ? Appearance.regalia.controlRadius
        : Appearance.zzzEverywhere ? Appearance.zzz.controlRadius
        : Appearance.angelEverywhere ? Appearance.angel.roundingSmall
        : Appearance.ilmangoEverywhere ? Appearance.ilmango.roundingSmall : Appearance.rounding.small
    colBackground: Appearance.regaliaEverywhere
        ? (button.critical ? Appearance.regalia.signalPlate : Appearance.regalia.controlPlate)
        : Appearance.zzzEverywhere
        ? (button.critical ? Appearance.zzz.secondary : Appearance.zzz.paperAlt)
        : button.critical
        ? Appearance.colors.colSecondaryContainer
        : Appearance.angelEverywhere ? Appearance.angel.colGlassCard
        : Appearance.ilmangoEverywhere ? Appearance.ilmango.colLayer3
        : Appearance.auroraEverywhere ? "transparent"
        : Appearance.colors.colLayer4
    colBackgroundHover: Appearance.regaliaEverywhere
        ? (button.critical ? Appearance.regalia.signalPlateHover : Appearance.regalia.controlPlateHover)
        : Appearance.zzzEverywhere
        ? (button.critical
            ? Appearance.colors.colSecondaryContainerHover
            : Appearance.colors.colLayer2Hover)
        : button.critical
        ? Appearance.colors.colSecondaryContainerHover
        : Appearance.angelEverywhere ? Appearance.angel.colGlassCardHover
        : Appearance.ilmangoEverywhere ? Appearance.ilmango.colLayer3Hover
        : Appearance.auroraEverywhere ? Appearance.aurora.colSubSurface
        : Appearance.colors.colLayer4Hover
    colRipple: Appearance.regaliaEverywhere
        ? (button.critical ? Appearance.regalia.signalPlateHover : Appearance.regalia.controlPlateActive)
        : Appearance.zzzEverywhere
        ? (button.critical
            ? Appearance.colors.colSecondaryContainerActive
            : Appearance.colors.colLayer2Active)
        : button.critical
        ? Appearance.colors.colSecondaryContainerActive
        : Appearance.angelEverywhere ? Appearance.angel.colGlassCardActive
        : Appearance.ilmangoEverywhere ? Appearance.ilmango.colLayer3Active
        : Appearance.auroraEverywhere ? Appearance.aurora.colSubSurfaceActive
        : Appearance.colors.colLayer4Active

    contentItem: StyledText {
        horizontalAlignment: Text.AlignHCenter
        text: buttonText
        color: Appearance.regaliaEverywhere
            ? (button.critical ? Appearance.regalia.signalPlateInk : Appearance.regalia.onColor)
            : Appearance.zzzEverywhere
            ? (button.critical ? Appearance.zzz.onSecondary : Appearance.zzz.ink)
            : button.critical
                ? Appearance.colors.colOnSecondaryContainer : Appearance.colors.colOnLayer3

        Behavior on color {
            enabled: Appearance.animationsEnabled
            animation: ColorAnimation { duration: Appearance.animation.elementMoveFast.duration; easing.type: Appearance.animation.elementMoveFast.type; easing.bezierCurve: Appearance.animation.elementMoveFast.bezierCurve }
        }
    }
}
