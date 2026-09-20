pragma Singleton

import qs.modules.common
import QtQuick
import Quickshell
import Quickshell.Wayland
import qs.services

Singleton {
    id: root

    readonly property bool sortingEnabled:
        (Config.options?.panelFamily ?? "ii") === "waffle"
    property int _identityRulesRevision: 0

    Connections {
        target: Config.options?.windows
        function onAppIdentityRulesChanged() {
            root._identityRulesRevision++
        }
    }

    function syncSortingDemand(): void {
        CompositorService.setSortingConsumer("waffleTaskbar",
            root.sortingEnabled)
    }

    onSortingEnabledChanged: syncSortingDemand()
    Component.onCompleted: syncSortingDemand()
    Component.onDestruction:
        CompositorService.setSortingConsumer("waffleTaskbar", false)

    function togglePin(appId) {
        const pinned = Config.options?.dock?.pinnedApps ?? []
        const exists = pinned.indexOf(appId) !== -1
        const next = exists ? pinned.filter(id => id !== appId) : pinned.concat([appId])
        Config.setNestedValue(["dock", "pinnedApps"], next)
    }

    function sourceToplevels(): list<var> {
        const sorted = CompositorService.sortedToplevels ?? [];
        return CompositorService.isNiri
            ? sorted
            : (sorted.length > 0
                ? sorted
                : (ToplevelManager.toplevels?.values ?? []));
    }

    function buildApps(sourceToplevels): list<var> {
        const identityRulesRevision = root._identityRulesRevision;
        var map = new Map();

        // Pinned apps
        const pinnedApps = Config.options?.dock?.pinnedApps ?? [];
        for (const appId of pinnedApps) {
            // Skip pinned apps with no desktop entry installed
            if (!AppSearch.lookupDesktopEntry(appId))
                continue;
            if (!map.has(appId.toLowerCase())) map.set(appId.toLowerCase(), ({
                pinned: true,
                toplevels: []
            }));
        }

        // Separator
        if (pinnedApps.length > 0) {
            map.set("SEPARATOR", { pinned: false, toplevels: [] });
        }

        // Ignored apps
        const ignoredRegexStrings = Config.options?.dock?.ignoredAppRegexes ?? [];
        const systemIgnored = [
            "^$", "^portal$", "^x-run-dialog$", "^kdialog$",
            "^org.freedesktop.impl.portal.*"
        ];
        const ignoredRegexes = ignoredRegexStrings.concat(systemIgnored)
            .map(pattern => new RegExp(pattern, "i"));

        // Open windows
        for (const toplevel of sourceToplevels) {
            const appId = AppSearch.resolveWindowIdentity(toplevel);
            if (appId.length === 0 || ignoredRegexes.some(re => re.test(appId)))
                continue;
            const lowerAppId = appId.toLowerCase();
            if (!map.has(lowerAppId)) map.set(lowerAppId, ({
                pinned: false,
                toplevels: []
            }));
            map.get(lowerAppId).toplevels.push(toplevel);
        }

        var values = [];

        for (const [key, value] of map) {
            values.push({
                appId: key,
                toplevels: value.toplevels,
                pinned: value.pinned
            });
        }

        return values;
    }

    // Pinned launchers remain shared, but running windows are scoped to the
    // active workspace of the bar's own output.
    function appsForScreen(screen): list<var> {
        const filtered = CompositorService.filterCurrentWorkspace(
            root.sourceToplevels(), screen);
        return root.buildApps(filtered);
    }

    property list<var> apps: root.buildApps(root.sourceToplevels())

}
