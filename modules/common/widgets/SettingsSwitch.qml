import qs.modules.common
import qs.modules.common.widgets

ConfigSwitch {
    colBackground: SettingsMaterialPreset.groupColor
    colBackgroundHover: Appearance.angelEverywhere ? Appearance.angel.colGlassCardHover
        : Appearance.ilmangoEverywhere ? Appearance.ilmango.colLayer2Hover
        : Appearance.auroraEverywhere ? Appearance.aurora.colElevatedSurfaceHover
        : Appearance.colors.colLayer2Hover
    colRipple: Appearance.angelEverywhere ? Appearance.angel.colGlassCardActive
        : Appearance.ilmangoEverywhere ? Appearance.ilmango.colLayer2Active
        : Appearance.auroraEverywhere ? Appearance.aurora.colSubSurfaceActive
        : Appearance.colors.colLayer2Active
}
