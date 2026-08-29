import qs
import qs.services
import qs.modules.common
import qs.modules.common.widgets
import qs.modules.common.functions
import qs.modules.sidebarLeft.anime
import QtQml
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import Qt5Compat.GraphicalEffects
import Quickshell
import Quickshell.Io
import Quickshell.Hyprland

Button {
    id: root
    property var imageData
    property var fallbackTags: []
    property var rowHeight
    property bool aspectCrop: false
    property bool lazyTagFetch: false
    property bool manualDownload: false
    property string previewDownloadPath
    property string downloadPath
    property string nsfwPath
    readonly property string _fileUrl: imageData?.file_url ?? imageData?.sample_url ?? imageData?.preview_url ?? ""
    readonly property var _previewCandidates: {
        const result = []
        const candidates = [root.imageData?.preview_url, root.imageData?.sample_url, root.imageData?.file_url]
        for (const candidate of candidates) {
            const url = String(candidate ?? "")
            if (url.length > 0 && !result.includes(url))
                result.push(url)
        }
        return result
    }
    readonly property bool _requiresSecureDns: root._previewCandidates.some(url =>
        /^https?:\/\/([^/]+\.)?konachan\.com(?:\/|$)/i.test(String(url)))
    readonly property real _aspectRatio: {
        const value = Number(root.imageData?.aspect_ratio ?? 1)
        return Number.isFinite(value) && value > 0 ? value : 1
    }
    property string fileName: {
        if (root._fileUrl.length > 0) {
            const cleanUrl = root._fileUrl.split("?")[0]
            const slashIndex = cleanUrl.lastIndexOf("/")
            const candidate = decodeURIComponent(cleanUrl.substring(slashIndex + 1))
            if (candidate.length > 0)
                return candidate
        }
        const fallbackId = String(root.imageData?.id ?? "preview")
        const fallbackExt = String(root.imageData?.file_ext ?? "jpg")
        return fallbackId + "." + fallbackExt
    }
    readonly property string previewFileName: {
        const key = String(root.imageData?.md5 ?? root.imageData?.id ?? Qt.md5(root._fileUrl))
            .replace(/[^A-Za-z0-9._-]/g, "_")
        const previewUrl = String(root.imageData?.preview_url ?? "").split("?")[0]
        const extensionMatch = previewUrl.match(/\.([A-Za-z0-9]{2,5})$/)
        const extension = extensionMatch ? extensionMatch[1].toLowerCase() : "jpg"
        return key + "." + extension
    }
    readonly property string previewFilePath: `${root.previewDownloadPath}/${root.previewFileName}`
    property bool _previewFallbackStarted: false
    property int maxTagStringLineLength: 50
    property real imageRadius: Appearance.rounding.small
    property bool showBackground: true  // When false, no background rectangle behind image

    // Allow consumers (e.g. Wallhaven) to opt-out of hover tooltips
    property bool enableTooltip: true
    property bool buttonHovered: false

    // Wallhaven tags are expensive (detail endpoint). Fetch them only when the user shows intent.
    property bool tagsRequested: false

    function shellQuote(value): string {
        return "'" + String(value ?? "").replace(/'/g, "'\"'\"'") + "'"
    }

    Timer {
        id: tagFetchTimer
        interval: 450
        repeat: false
        onTriggered: {
            if (!root.lazyTagFetch)
                return
            if (!root.imageData || !root.imageData.id)
                return
            if (root.imageData.tags && root.imageData.tags.length > 0)
                return
            root.tagsRequested = true
            Wallhaven.ensureWallpaperTags(root.imageData.id)
        }
    }

    readonly property string _tagText: {
        if (root.imageData && root.imageData.tags && root.imageData.tags.length > 0)
            return root.imageData.tags
        if (root.fallbackTags && root.fallbackTags.length > 0)
            return root.fallbackTags
        return ""
    }

    hoverEnabled: true

    onHoveredChanged: {
        if (!root.lazyTagFetch)
            return
        if (root.hovered) {
            // Only start the timer if tags are not already present.
            if (!(root.imageData && root.imageData.tags && root.imageData.tags.length > 0)) {
                tagFetchTimer.restart()
            }
        } else {
            tagFetchTimer.stop()
        }
    }
    
    Process {
        id: downloadProcess
        property var candidates: []
        property int candidateIndex: 0
        property bool secureDnsPass: false
        property string lastError: ""

        running: false
        stdout: StdioCollector {}
        stderr: StdioCollector {
            id: downloadErrorCollector
            onStreamFinished: downloadProcess.lastError = text.trim()
        }

        function startDownload(skipSystemDns: bool): void {
            if (running || root._previewCandidates.length === 0)
                return

            root._previewFallbackStarted = true
            candidates = [...root._previewCandidates]
            candidateIndex = 0
            secureDnsPass = skipSystemDns
            lastError = ""
            runNext()
        }

        function runNext(): void {
            if (candidateIndex >= candidates.length) {
                if (!secureDnsPass) {
                    // Browsers commonly use Secure DNS while Qt follows the
                    // system resolver. Retry once through DoH for filtered or
                    // missing image-CDN records such as assets.yande.re.
                    secureDnsPass = true
                    candidateIndex = 0
                } else {
                    console.warn("[BooruImage] Could not cache preview:", lastError)
                    return
                }
            }

            const args = [
                "curl", "--fail", "--location", "--silent", "--show-error",
                "--remove-on-error", "--connect-timeout", "8", "--max-time", "30",
                "--create-dirs", "--user-agent", String(Booru.defaultUserAgent),
                "--output", root.previewFilePath
            ]
            if (secureDnsPass)
                args.push("--doh-url", "https://1.1.1.1/dns-query")
            args.push(candidates[candidateIndex])
            candidateIndex += 1
            command = args
            running = true
        }

        onExited: (exitCode, exitStatus) => {
            if (exitCode === 0) {
                imageObject.source = root.previewFilePath
                return
            }
            Qt.callLater(() => downloadProcess.runNext())
        }
    }

    Component.onCompleted: {
        if (root.manualDownload)
            downloadProcess.startDownload(root._requiresSecureDns)
    }

    StyledToolTip {
        // Scrolling moves the pointer across many thumbnails. Require real
        // hover intent for Wallhaven instead of flashing every available tag.
        delay: root.lazyTagFetch ? 750 : 16
        extraVisibleCondition: root.enableTooltip && root.imageData && root._tagText.length > 0
        alternativeVisibleCondition: root.enableTooltip && (root.buttonHovered || root.hovered)
        text: `${StringUtils.wordWrap(root._tagText, root.maxTagStringLineLength)}`
    }

    padding: 0
    implicitWidth: root.rowHeight * root._aspectRatio
    implicitHeight: root.rowHeight

    background: Rectangle {
        implicitWidth: root.rowHeight * root._aspectRatio
        implicitHeight: root.rowHeight
        radius: imageRadius
        color: root.showBackground ? (Appearance.angelEverywhere ? Appearance.angel.colGlassCard : Appearance.ilmangoEverywhere ? Appearance.ilmango.colLayer2 : (Appearance.auroraEverywhere ? Appearance.aurora.colElevatedSurface : Appearance.colors.colLayer2)) : "transparent"
    }

    contentItem: Item {
        anchors.fill: parent

        StyledImage {
            id: imageObject
            anchors.fill: parent
            width: root.rowHeight * root._aspectRatio
            height: root.rowHeight
            fillMode: root.aspectCrop ? Image.PreserveAspectCrop : Image.PreserveAspectFit
            source: root.manualDownload ? "" : (root.imageData?.preview_url ?? "")
            sourceSize.width: root.rowHeight * root._aspectRatio
            sourceSize.height: root.rowHeight

            onStatusChanged: {
                if (status === Image.Error && !root._previewFallbackStarted)
                    downloadProcess.startDownload(true)
            }

            layer.enabled: true
            layer.effect: OpacityMask {
                maskSource: Rectangle {
                    width: root.rowHeight * root._aspectRatio
                    height: root.rowHeight
                    radius: imageRadius
                }
            }
        }

        MouseArea {
            anchors.fill: parent
            acceptedButtons: Qt.RightButton
            hoverEnabled: true
            propagateComposedEvents: true
            onEntered: root.buttonHovered = true
            onExited: root.buttonHovered = false
            onWheel: wheel => {
                if (contextMenu.visible) {
                    contextMenu.close()
                }
                wheel.accepted = false
            }
            onPressed: mouse => {
                if (mouse.button !== Qt.RightButton)
                    return

                contextMenu.x = mouse.x
                contextMenu.y = mouse.y
                contextMenu.popup()
                mouse.accepted = true
            }
        }

        RippleButton {
            id: menuButton
            z: 2
            anchors.top: parent.top
            anchors.right: parent.right
            property real buttonSize: 24
            anchors.margins: 6
            implicitHeight: buttonSize
            implicitWidth: buttonSize
            visible: root.hovered || root.buttonHovered || contextMenu.visible

            buttonRadius: buttonSize / 2
            rippleEnabled: false
            colBackground: ColorUtils.transparentize(Appearance.colors.colLayer1, 0.38)
            colBackgroundHover: colBackground
            colRipple: colBackground

            contentItem: MaterialSymbol {
                horizontalAlignment: Text.AlignHCenter
                iconSize: Appearance.font.pixelSize.normal
                color: Appearance.colors.colOnSurface
                text: "more_vert"
            }

            onClicked: {
                contextMenu.x = menuButton.x + menuButton.width
                contextMenu.y = menuButton.y + menuButton.height
                contextMenu.popup()
            }
        }

        Menu {
            id: contextMenu
            z: 3
            closePolicy: Popup.CloseOnEscape | Popup.CloseOnPressOutside

            MenuItem {
                enabled: root._fileUrl.length > 0
                text: Translation.tr("Open file link")
                onTriggered: Qt.openUrlExternally(root._fileUrl)
            }
            MenuItem {
                visible: String(root.imageData?.source ?? "").length > 0
                text: Translation.tr("Go to source (%1)").arg(
                    StringUtils.getDomain(root.imageData?.source ?? ""))
                onTriggered: Qt.openUrlExternally(root.imageData.source)
            }
            MenuSeparator {}
            MenuItem {
                enabled: root._fileUrl.length > 0
                text: Translation.tr("Download")
                onTriggered: {
                    const targetPath = root.imageData.is_nsfw ? root.nsfwPath : root.downloadPath
                    const localPath = `${targetPath}/${root.fileName}`
                    const curlOptions = root._requiresSecureDns
                        ? "--doh-url https://1.1.1.1/dns-query " : ""
                    Quickshell.execDetached(["/usr/bin/bash", "-c",
                        `mkdir -p ${root.shellQuote(targetPath)} && curl -fL ${curlOptions}${root.shellQuote(root._fileUrl)} -o ${root.shellQuote(localPath)} && notify-send ${root.shellQuote(Translation.tr("Download complete"))} ${root.shellQuote(localPath)} -a Shell`])
                    if (Config.options?.sidebar?.openFolderOnDownload ?? false)
                        ShellExec.execDetachedArgs(["xdg-open", targetPath], "Open image")
                }
            }
            MenuItem {
                enabled: root._fileUrl.length > 0
                text: Translation.tr("Set as wallpaper")
                onTriggered: {
                    const targetPath = root.imageData.is_nsfw ? root.nsfwPath : root.downloadPath
                    const localPath = `${targetPath}/${root.fileName}`
                    const mode = Appearance.m3colors.darkmode ? "dark" : "light"
                    const curlOptions = root._requiresSecureDns
                        ? "--doh-url https://1.1.1.1/dns-query " : ""
                    Quickshell.execDetached(["/usr/bin/bash", "-c",
                        `mkdir -p ${root.shellQuote(targetPath)} && curl -fLsS ${curlOptions}${root.shellQuote(root._fileUrl)} -o ${root.shellQuote(localPath)} && ${root.shellQuote(Directories.wallpaperSwitchScriptPath)} --image ${root.shellQuote(localPath)} --mode ${root.shellQuote(mode)}`])
                }
            }
        }
    }
}
