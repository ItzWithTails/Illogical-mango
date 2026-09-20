import QtQuick
import QtQuick.Layouts
import qs.services
import qs.modules.common
import qs.modules.common.functions
import qs.modules.waffle.looks
import qs.modules.waffle.bar
import Quickshell

AppButton {
    id: root

    required property var appEntry
    property var tasksParent: null  // Reference to Tasks for closing other menus
    readonly property bool isSeparator: appEntry.appId === "SEPARATOR"
    readonly property var desktopEntry: AppSearch.lookupDesktopEntry(appEntry.appId)
    property bool active: root.appEntry.toplevels.some(t => t.activated)
    readonly property int visibleWindowCount: CompositorService.isMango
        ? root.appEntry.toplevels.filter(t => t?.isMinimized !== true).length
        : root.appEntry.toplevels.length
    property bool hasWindows: root.visibleWindowCount > 0

    // Focused window index for smart indicator (Niri)
    property int focusedWindowIndex: {
        if (!root.active || appEntry.toplevels.length <= 1) return 0;
        const focusedToplevel = appEntry.toplevels.find(t => t.activated === true);
        if (!focusedToplevel) return 0;
        
        if (CompositorService.isNiri && focusedToplevel.niriWindowId) {
            const niriWindows = NiriService.windows;
            const windowPositions = [];
            for (let i = 0; i < appEntry.toplevels.length; i++) {
                const tl = appEntry.toplevels[i];
                let col = 999999;
                if (tl.niriWindowId) {
                    const niriWin = niriWindows.find(w => w.id === tl.niriWindowId);
                    if (niriWin?.layout?.pos_in_scrolling_layout) {
                        col = niriWin.layout.pos_in_scrolling_layout[0];
                    }
                }
                windowPositions.push({ idx: i, col: col, activated: tl.activated });
            }
            windowPositions.sort((a, b) => a.col - b.col);
            for (let i = 0; i < windowPositions.length; i++) {
                if (windowPositions[i].activated) return i;
            }
        }
        
        for (let i = 0; i < appEntry.toplevels.length; i++) {
            if (appEntry.toplevels[i].activated) return i;
        }
        return 0;
    }

    signal hoverPreviewRequested()
    signal hoverPreviewDismissed()

    multiple: appEntry.toplevels.length > 1
    checked: active
    iconName: AppSearch.guessIcon(appEntry.appId)
    tryCustomIcon: false
    
    onHoverTimedOut: {
        root.hoverPreviewRequested()
    }

    // Niri emulates minimize with a hidden workspace; mango exposes the real
    // minimized state on each client.
    readonly property var mangoMinimizedToplevels: CompositorService.isMango
        ? root.appEntry.toplevels.filter(t => t?.isMinimized === true)
        : []
    readonly property int minimizedCount: CompositorService.isMango
        ? root.mangoMinimizedToplevels.length
        : MinimizedWindows.countMinimizedForApp(appEntry.appId)
    readonly property bool hasMinimized: minimizedCount > 0

    function niriWindowIds(): list<var> {
        const ids = []
        for (const toplevel of root.appEntry.toplevels ?? []) {
            const id = Number(toplevel?.niriWindowId ?? -1)
            if (id > 0)
                ids.push(id)
        }
        return ids
    }

    function mangoWindowIds(includeMinimized = true): list<var> {
        const ids = []
        for (const toplevel of root.appEntry.toplevels ?? []) {
            if (!includeMinimized && toplevel?.isMinimized === true)
                continue
            const id = Number(toplevel?.mangoWindowId ?? -1)
            if (id > 0)
                ids.push(id)
        }
        return ids
    }

    function fluentIconForDesktopAction(iconName, actionName): string {
        const icon = String(iconName ?? "").toLowerCase();
        const name = String(actionName ?? "").toLowerCase();

        if (name.includes("new") && (name.includes("window") || name.includes("instance") || name.includes("tab"))) {
            return "add";
        }
        if (name.includes("open") || name.includes("launch")) {
            return "arrow-enter-left";
        }
        if (name.includes("private") || name.includes("incognito")) {
            return "shield";
        }
        if (name.includes("settings") || name.includes("preferences") || icon.includes("settings")) {
            return "settings";
        }
        if (name.includes("quit") || name.includes("exit") || name.includes("close")) {
            return "dismiss";
        }

        // Never return arbitrary desktop-action icons: that would fall back to non-Fluent
        // system icons if we don't have a matching Fluent asset.
        if (icon.includes("settings") || icon.includes("preferences")) return "settings";
        if (icon.includes("new") || icon.includes("add")) return "add";
        if (icon.includes("open") || icon.includes("launch")) return "arrow-enter-left";
        if (icon.includes("close") || icon.includes("quit") || icon.includes("exit")) return "dismiss";

        return "app-generic";
    }

    // Track if this app was active (focused) - use the toplevel activated state
    // which is more reliable than checking NiriService during click
    readonly property bool wasActive: root.active

    onClicked: {
        root.hoverTimer.stop()
        
        const isAppFocused = root.wasActive;

        if (CompositorService.isMango) {
            const focused = root.appEntry.toplevels.find(t => t.activated === true)

            // KDE/Windows-style task button: clicking the focused app hides
            // it, while clicking its dimmed task restores it.
            if (isAppFocused && focused?.mangoWindowId) {
                MangoService.minimizeWindow(focused.mangoWindowId)
                return
            }

            if (root.hasMinimized) {
                const minimized = root.mangoMinimizedToplevels[root.mangoMinimizedToplevels.length - 1]
                if (minimized?.mangoWindowId) {
                    MangoService.restoreMinimizedWindow(minimized.mangoWindowId)
                    return
                }
            }

            const windowIds = root.mangoWindowIds(false)
            if (windowIds.length > 0) {
                MangoService.focusWindow(windowIds[0])
                return
            }
        } else if (CompositorService.isNiri) {
            const windowIds = root.niriWindowIds()

            // Case 1: App is focused -> minimize the exact active window.
            if (isAppFocused && windowIds.length > 0) {
                const focused = root.appEntry.toplevels.find(t => t.activated)
                MinimizedWindows.minimize(
                    focused?.niriWindowId ?? windowIds[0])
                return
            }

            // Case 2: App has minimized windows -> restore the latest.
            if (root.hasMinimized) {
                MinimizedWindows.restoreLatestForApp(root.appEntry.appId)
                return
            }

            // Case 3: App has visible windows but is not focused.
            if (windowIds.length > 0) {
                NiriService.focusWindow(windowIds[0])
                return
            }
        } else if (root.appEntry.toplevels.length > 0) {
            root.appEntry.toplevels[0]?.activate()
            return
        }
        
        // Case 4: App not running -> launch it
        if (root.desktopEntry) {
            AppSearch.launchEntry(root.desktopEntry)
        }
    }

    middleClickAction: () => {
        if (root.desktopEntry) {
            AppSearch.launchEntry(root.desktopEntry)
        }
    }

    altAction: () => {
        root.hoverPreviewDismissed()
        root.hoverTimer.stop()
        // Close other context menus first
        if (tasksParent) tasksParent.closeAllContextMenus()
        contextMenu.active = true;
    }
    
    Connections {
        target: root.tasksParent
        enabled: root.tasksParent !== null
        function onCloseAllContextMenus() {
            if (contextMenu.active && contextMenu.item) {
                contextMenu.item.close();
            }
        }
    }

    // Smart indicator: W11 style pills showing window count and which is focused
    Row {
        id: indicatorRow
        visible: root.hasWindows || root.hasMinimized
        anchors {
            horizontalCenter: root.background.horizontalCenter
            bottom: root.background.bottom
            bottomMargin: 1
        }
        spacing: 2

        // Active windows
        Repeater {
            model: Math.min(root.visibleWindowCount, 5)
            delegate: Rectangle {
                required property int index
                property bool isFocused: root.active && index === root.focusedWindowIndex

                implicitWidth: isFocused ? 16 : 4
                implicitHeight: 3
                radius: height / 2
                color: isFocused ? Looks.colors.accent : Looks.colors.accentUnfocused

                Behavior on implicitWidth {
                    animation: NumberAnimation { duration: Looks.transition.enabled ? Looks.transition.duration.panel : 0; easing.type: Easing.BezierSpline; easing.bezierCurve: Looks.transition.easing.bezierCurve.decelerate }
                }
                Behavior on color {
                    animation: ColorAnimation { duration: Looks.transition.enabled ? 70 : 0; easing.type: Easing.BezierSpline; easing.bezierCurve: Looks.transition.easing.bezierCurve.standard }
                }
            }
        }
        
        // Minimized windows (dimmed indicator)
        Repeater {
            model: Math.min(root.minimizedCount, 3)
            delegate: Rectangle {
                implicitWidth: 4
                implicitHeight: 3
                radius: height / 2
                color: Looks.colors.fg1
                opacity: 0.5
            }
        }

        Behavior on opacity {
            animation: NumberAnimation { duration: Looks.transition.enabled ? Looks.transition.duration.normal : 0; easing.type: Easing.BezierSpline; easing.bezierCurve: Looks.transition.easing.bezierCurve.standard }
        }
    }

    // Refined has exactly one hover surface: the icon-bearing TaskPreview,
    // whose header row already renders the app icon and name, so a pinned app
    // with no windows still gets a label. The plain tooltip used to cover that
    // case via `!hasWindows`, but the two surfaces raced whenever a window
    // opened or closed under the pointer and could show at once, overlapping
    // and outliving each other. Classic keeps the tooltip.
    BarToolTip {
        barExtraVisibleCondition: root.shouldShowTooltip && !root.hasWindows
        text: desktopEntry ? desktopEntry.name : appEntry.appId
    }

    BarMenu {
        id: contextMenu
        anchorHovered: root.hovered
        noSmoothClosing: false // On the real thing this is always smooth
        closeOnHoverLostDelay: 500  // Slower close to give time to click

        model: [
            ...((root.desktopEntry?.actions.length > 0) ? root.desktopEntry.actions.map(action =>({
                iconName: root.fluentIconForDesktopAction(action.icon, action.name),
                text: action.name,
                action: () => {
                    AppSearch.launchDesktopAction(root.desktopEntry, action)
                }
            })).concat({ type: "separator" }) : []),
            {
                iconName: root.iconName,
                text: root.desktopEntry ? root.desktopEntry.name : StringUtils.toTitleCase(appEntry.appId),
                monochromeIcon: false,
                action: () => {
                    if (root.desktopEntry) {
                        AppSearch.launchEntry(root.desktopEntry)
                    }
                }
            },
            {
                iconName: root.appEntry.pinned ? "pin-off" : "pin",
                text: root.appEntry.pinned ? Translation.tr("Unpin from taskbar") : Translation.tr("Pin to taskbar"),
                action: () => {
                    TaskbarApps.togglePin(root.appEntry.appId);
                }
            },
            ...((CompositorService.isNiri || CompositorService.isMango)
                    && root.appEntry.toplevels.length > root.minimizedCount ? [
                {
                    iconName: "caret-down",
                    text: Translation.tr("Minimize"),
                    action: () => {
                        if (CompositorService.isMango) {
                            for (const id of root.mangoWindowIds(false))
                                MangoService.minimizeWindow(id)
                        } else {
                            for (const id of root.niriWindowIds())
                                MinimizedWindows.minimize(id)
                        }
                    }
                }
            ] : []),
            ...(root.hasMinimized ? [
                {
                    iconName: "arrow-up",
                    text: Translation.tr("Show"),
                    action: () => {
                        if (CompositorService.isMango) {
                            const minimized = root.mangoMinimizedToplevels[root.mangoMinimizedToplevels.length - 1]
                            if (minimized?.mangoWindowId)
                                MangoService.restoreMinimizedWindow(minimized.mangoWindowId)
                        } else {
                            MinimizedWindows.restoreLatestForApp(root.appEntry.appId)
                        }
                    }
                }
            ] : []),
            ...(root.appEntry.toplevels.length > 0 ? [
                {
                    iconName: "dismiss",
                    text: root.multiple ? Translation.tr("Close all windows") : Translation.tr("Close window"),
                    action: () => {
                        for (let toplevel of root.appEntry.toplevels) {
                            if (CompositorService.isNiri
                                    && toplevel?.niriWindowId) {
                                NiriService.closeWindow(toplevel.niriWindowId)
                            } else {
                                toplevel?.close()
                            }
                        }
                    }
                }
            ] : []),
        ]
    }
}
