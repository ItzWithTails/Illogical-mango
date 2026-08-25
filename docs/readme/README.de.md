<h1 align="center">Illogical-mango</h1>

<p align="center">
  <b>Eine vollständige Desktop-Shell für MangoWM, gebaut auf Quickshell</b>
</p>

<p align="center">
  <sub>
    <a href="../../README.md">English</a> · <a href="README.ru.md">Русский</a> · <a href="README.es.md">Español</a> · <a href="README.zh.md">中文</a> · <a href="README.ja.md">日本語</a> · <a href="README.pt.md">Português</a> · <a href="README.fr.md">Français</a> · <a href="README.de.md">Deutsch</a> · <a href="README.ko.md">한국어</a> · <a href="README.hi.md">हिन्दी</a> · <a href="README.ar.md">العربية</a> · <a href="README.it.md">Italiano</a>
  </sub>
</p>

---

## Dieser Port wurde von einer KI geschrieben. Komplett. Diese README auch, zu etwa 90 %

Das Projekt ist ein Spaß. Hier hat sich niemand Mühe gegeben.

Der Port auf MangoWM - `services/MangoService.qml`,
`services/deferred/MangoKeybinds.qml`, der Umbau der Compositor-Erkennung in
`services/CompositorService.qml`, die Änderungen am Installer und an doctor - wurde von
Anfang bis Ende über Claude geschrieben.
Nicht "mit Hilfe von". Von ihr geschrieben.

Das steht ganz oben, damit du es nicht später erfährst — aus einem Diff oder aus einem Bug.
Es ist keine Leistung und wird auch nicht als solche verkauft. Im Grunde habe ich den Port
für mich selbst und aus Spaß gemacht. Rechne damit.

Die Shell unter der Port-Schicht ist snowarchs iNiR, geschrieben (hoffentlich) von einem
Menschen.

---

## Was das hier ist

Ehrlich gesagt: Wer schon einmal selbst einen nackten Wayland-Compositor aufgesetzt hat,
dem muss man nicht erklären, wozu der eine Shell braucht. Aber ich bin verpflichtet zu
erklären, wie es funktioniert.

```
deine Anwendungen
   ↓
Illogical-mango   Leiste, Dock, Sidebars, Übersicht, Benachrichtigungen, Einstellungen, Sperrbildschirm
   ↓
Quickshell        QML-Laufzeitumgebung für Wayland-Shells
   ↓
MangoWM           Fenster und Rendering
   ↓
Wayland → GPU
```

**Was es von anderen Quickshell-Konfigurationen unterscheidet:**

- **Zwei vollständige Panel-Familien in einer Installation.** Material ii (schwebende
  Leiste, Sidebars, Dock) und Waffle (Taskleiste unten, Startmenü, Info-Center). Das sind
  keine Themes über denselben Widgets — es sind getrennte Panel-Bäume mit eigenen
  Token-Systemen, zur Laufzeit umschaltbar mit
  <kbd>Super</kbd>+<kbd>Shift</kbd>+<kbd>W</kbd>.
- **Systemweites Theming, nicht nur Shell-Theming.** Ein Hintergrundbild erzeugt eine
  Material-You-Palette, die nach GTK3/4, Qt, zehn Terminal- und TUI-Werkzeuge, Firefox,
  Discord, Spicetify, Steam und SDDM geschrieben wird.
- **Konfigurierbar ohne Code zu bearbeiten.** Alles ist eine GUI-Einstellung über einer
  einzigen `config.json`. QML musst du nie anfassen, um Aussehen oder Verhalten zu ändern.
- **Ein echter Installations- und Update-Pfad.** `./setup` kümmert sich um Abhängigkeiten
  und Systemkonfiguration; `inir update` holt, führt Schema-Migrationen aus, bewahrt deine
  Änderungen und kann zurückrollen.

**Herkunft.** [end-4s illogical-impulse](https://github.com/end-4/dots-hyprland)
(Hyprland-Dotfiles) → [snowarchs iNiR](https://github.com/snowarch/iNiR) (für niri neu
geschrieben) → das hier, portiert auf MangoWM. CLI, Konfigurationspfade und Innereien
heißen weiterhin `inir`: Ein Umbenennen würde jeden Update-Pfad zerlegen, also blieb der
Name.
Warum nicht direkt end-4 forken? Die Logik ist einfach - ein Projekt, das schon einmal
portiert wurde, lässt sich leichter erneut portieren.
Als Analogie: Nimm Void Linux. Installiere systemd darauf, und es läuft problemlos.
Nimm Arch Linux und reiß systemd heraus, und du musst fast die gesamte Paketbasis ändern.


## Compositor

Gebaut für [MangoWM](https://github.com/DreamMaoMao/mango) und nur darauf getestet.

Die Shell spricht mit mango über dessen IPC-Socket unter `$MANGO_INSTANCE_SIGNATURE`, das
bei jeder Änderung einen vollständigen Schnappschuss der Sitzung schickt. mango ist
dwm-artig — Tags statt einer Liste von Arbeitsflächen —, also bildet `MangoService` Paare
aus `(Monitor, Tag-Index)` auf dasselbe Arbeitsflächenmodell ab, das Leiste, Dock,
Übersicht und Arbeitsflächenstreifen ohnehin erwarten, und diese Module laufen unverändert.

Die Konfiguration ist bewusst nicht destruktiv. mango liest genau eine Datei
(`~/.config/mango/config.conf`) und führt nichts zusammen, deshalb überschreibt der
Installer nie deine Compositor-Konfiguration. Er legt Tastenkürzel und Autostart der Shell
in `~/.config/mango/inir.conf` ab und hängt eine einzige `source-optional=`-Zeile an, die
darauf zeigt, ohne deine Fensterverwaltung anzurühren. Der Autostart ist eine Zeile
`exec-once=inir run --daemon` in dieser Datei, keine systemd-Unit.

> [!NOTE]
> **Code für niri und Hyprland liegt noch im Baum.** `NiriService.qml`,
> `HyprlandData.qml` und die Zweige `isNiri` / `isHyprland` haben von upstream überlebt und
> kompilieren weiterhin. Sie sind geerbt, nicht unterstützt: Hier wird nichts gegen diese
> Compositor getestet und nichts für sie gepflegt. Wenn du niri willst, nimm
> [das originale iNiR](https://github.com/snowarch/iNiR).

---

## Screenshots

Beide Panel-Familien, unverändert von upstream übernommen.

<details open>
<summary><b>Material ii</b>: schwebende Leiste, Sidebars, Material-Design-Ästhetik</summary>

| | |
|:---:|:---:|
| ![](https://github.com/user-attachments/assets/1fe258bc-8aec-4fd9-8574-d9d7472c3cc8) | ![](https://github.com/user-attachments/assets/3ce2055b-648c-45a1-9d09-705c1b4a03b7) |
| ![](https://github.com/user-attachments/assets/ea2311dc-769e-44dc-a46d-37cf8807d2cc) | ![](https://github.com/user-attachments/assets/ba866063-b26a-47cb-83c8-d77bd033bf8b) |
| ![](https://github.com/user-attachments/assets/88e76566-061b-4f8c-a9a8-53c157950138) | |

</details>

<details>
<summary><b>Waffle</b>: Taskleiste unten, Info-Center, Windows-11-Vibe</summary>

| | |
|:---:|:---:|
| ![](https://github.com/user-attachments/assets/5c5996e7-90eb-4789-9921-0d5fe5283fa3) | ![](https://github.com/user-attachments/assets/fadf9562-751e-4138-a3a1-b87b31114d44) |

</details>

---

> [!WARNING]
> Die Standardkonfiguration zielt auf halbwegs moderne Hardware. Auf schwachen Maschinen
> schalte Effekte ab, wirf Panels raus, die du nicht brauchst, und mach den visuellen Stil
> flacher — all das geht über die Einstellungen oder `config.json`.

## Funktionen

**Zwei Panel-Familien**, im laufenden Betrieb umschaltbar mit <kbd>Super</kbd>+<kbd>Shift</kbd>+<kbd>W</kbd>:

- **Material ii** — schwebende Leiste, Sidebars, Dock und 8 visuelle Stile (Material,
  Cards, Aurora, iNiR, Angel, Regalia, ZZZ, Cookie Shapes)
- **Waffle** — Taskleiste im Windows-11-Stil, Startmenü, Info-Center,
  Benachrichtigungszentrale

**Automatisches Theming.** Wähle ein Hintergrundbild, und das ganze System zieht nach: Die
Material-You-Farben der Shell werden nach GTK3/4, Qt, Terminals, Firefox, Discord,
Spicetify, Steam und SDDM verteilt. Mit den Presets Regalia, Gruvbox, Catppuccin und Rosé
Pine, oder bau dir dein eigenes.

<details>
<summary><b>Vollständige Funktionsliste</b></summary>

### Theming und Aussehen

- **8 visuelle Stile**: Material (deckend), Cards, Aurora (Glas-Unschärfe), iNiR (TUI-inspiriert), Angel (Neobrutalismus), Regalia (schwarzes Ingenieurs-Chassis, warme Elfenbeintinte, zurückhaltende Champagner-Beschläge), ZZZ (Plakatplatten), Cookie Shapes (animiertes Formen-Morphing)
- **Dynamische Farben aus dem Hintergrundbild** via Material You, systemweit verteilt
- **10 Terminals und TUI-Werkzeuge automatisch gethemt**: foot, kitty, alacritty, ghostty, wezterm, starship, fuzzel, btop, lazygit, yazi
- **App-Theming**: GTK3/4, Qt (über plasma-integration und darkly), Firefox (MaterialFox), Discord/Vesktop (System24), Zed, Spicetify, Steam, SDDM
- **Theme-Presets**: Regalia, Regalia Ivory, Gruvbox, Catppuccin, Rosé Pine und eigene
- **Video-Hintergründe**: mp4/webm/gif mit optionaler Unschärfe, oder eingefrorenes erstes Bild für die Leistung
- **Desktop-Widgets**: Uhr (mehrere Stile), Wetter, Mediensteuerung auf der Hintergrundebene

### Leiste

- **6 Leistenstile**: classic, islands, scenic, frame, Material-3-Kapseln und pill
- **Pill-Leiste**: eine morphende Insel in der Mitte, die sich beim Überfahren zu Arbeitsflächen, Starter, Mixer, Medien, Kalender und Bildschirmaufnahme öffnet
- **Modulares Layout** mit Drag-Editor in den Einstellungen: jedes Modul kann überallhin
- **Vertikale Leiste** für Layouts am Bildschirmrand

### Sidebars und Widgets (Material ii)

Linke Sidebar (App-Schublade):
- **AI Chat**: Live-Modellkataloge über Ollama, LM Studio, OpenRouter, Gemini, Groq, Mistral, Cerebras, Anthropic, OpenAI und OpenCode
- **YT Music**: cookieloser InnerTube-Player mit Suche, Warteschlange, Radio und synchronisiertem Text
- **Wallhaven-Browser**: Hintergründe direkt suchen und anwenden
- **Anime-Tracker**: AniList-Integration mit Terminansicht
- **Übersetzer**: über Gemini oder translate-shell
- **Ziehbare Widgets**: Krypto, Medienplayer, Schnellnotizen, Statusringe, Wochenkalender

Rechte Sidebar:
- **Kalender** mit Termin-Integration
- **Benachrichtigungszentrale**
- **Schnellschalter**: WLAN, Bluetooth, Nachtlicht, Nicht stören, Energieprofile, WARP VPN, EasyEffects
- **Lautstärkemixer** mit Steuerung pro Anwendung
- **Bluetooth- und WLAN-Geräteverwaltung**
- **Pomodoro-Timer**, **Aufgabenliste**, **Taschenrechner**, **Notizblock**
- **Systemmonitor**: CPU, RAM, Temperatur

### Werkzeuge

- **Arbeitsflächen-Übersicht**: App-Suche und Taschenrechner, auf mangos Tag-Modell abgebildet
- **Dashboard**: konfigurierbares Overlay in drei Spalten mit Agenda, Benachrichtigungen, Aufgaben, Notizen, Medien und Wetter
- **Arbeitsflächenstreifen am Rand**: Schiene beim Überfahren mit Live-Vorschauen und Umsortieren per Ziehen
- **Fensterwechsler**: animiertes Alt-Tab über alle Arbeitsflächen, optional
- **Zwischenablage-Verwaltung**: Verlauf mit Suche und Bildvorschau
- **Bereichswerkzeuge**: Screenshots, Bildschirmaufnahme, OCR, Rückwärtssuche für Bilder
- **Spickzettel**: Tastenkürzel-Anzeige, aus deiner mango-Konfiguration gezogen
- **Mediensteuerung**: vollwertiger MPRIS-Player mit mehreren Layout-Presets
- **Bildschirmanzeige**: OSD für Lautstärke, Helligkeit und Medien
- **Songerkennung**: Shazam-artig über SongRec
- **Spracheingabe**: lokales whisper.cpp, sofern installiert, oder ein angebundenes Backend von Groq, Gemini oder OpenAI

### System

- **GUI-Einstellungen**: alles konfigurierbar, ohne Dateien anzufassen
- **GameMode**: schaltet Effekte für Vollbild-Apps automatisch ab
- **Automatische Updates**: `inir update` mit Rollback, Migrationen und Erhalt deiner Änderungen
- **Sperrbildschirm** und **Sitzungsbildschirm** (Abmelden/Neustart/Herunterfahren/Standby)
- **Polkit-Agent**, **Bildschirmtastatur**, **Autostart-Verwaltung** gestützt auf die `exec-once`-Zeile in der mango-Konfiguration
- **Kira**: optionales Pixelart-Maskottchen, das an den Bildschirmrändern umherläuft und auf dein Tun reagiert. Standardmäßig aus; das ~32 MiB große Art-Paket wird separat unter `./setup` › Extras geladen
- **15 Sprachen** mit automatischer Erkennung
- **Nachtlicht**: nach Zeitplan oder manuell
- **Wetter**: Open-Meteo, unterstützt GPS, manuelle Koordinaten oder Städtenamen
- **Akkuverwaltung**: konfigurierbare Schwellen, Auto-Standby bei kritischem Stand
- **Eigene Ereignisklänge** mit Gesamtlautstärke und je einer Audiodatei pro Ereignis

</details>

---

## Schnellstart (der Installer wird künftig ein anderer sein)

```bash
git clone https://github.com/ItzWithTails/Illogical-mango.git
cd Illogical-mango
./setup install       # interaktiv, fragt vor jedem Schritt
./setup install -y    # automatisch, ohne Rückfragen
```

Der Installer kümmert sich um Abhängigkeiten, Systemkonfiguration und Theming. Er schreibt
die Tastenkürzel der Shell nach `~/.config/mango/inir.conf` und hängt sie an deine
bestehende mango-Konfiguration, ohne deine Fensterverwaltung anzurühren. Starte mango neu
oder führe `mmsg dispatch reload_config` aus.

```bash
inir run                        # Shell starten
inir settings                   # Einstellungs-GUI öffnen
inir logs                       # Logs ansehen
inir doctor                     # Selbstdiagnose und Reparatur
inir update                     # pull + Migrationen + Neustart
```

Weitere Einstiegspunkte:

```bash
./setup                 # TUI-Menü, du wählst aus
./setup install --skip-mango    # die mango-Konfiguration gar nicht anfassen
sudo make install       # systemweit statt im Home
./setup rollback        # letztes Update rückgängig machen
```

**Distributionen.** Arch ist das Hauptziel und am besten getestet. Für Debian und Fedora
gibt es natürlich einen Port... auf eigene Gefahr, getestet wurde darauf nicht.

---

## Tastenkürzel

Werden aus `defaults/mango/config.conf` installiert:

| Taste | Aktion |
|-----|--------|
| <kbd>Super</kbd> + <kbd>Space</kbd> | Übersicht: App-Suche, Navigation über Tags |
| <kbd>Super</kbd> + <kbd>V</kbd> | Zwischenablage-Verlauf |
| <kbd>Super</kbd> + <kbd>P</kbd> | Linke Sidebar |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>N</kbd> | Rechte Sidebar |
| <kbd>Super</kbd> + <kbd>D</kbd> | Dashboard |
| <kbd>Super</kbd> + <kbd>,</kbd> | Einstellungen |
| <kbd>Super</kbd> + <kbd>/</kbd> | Spickzettel |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>W</kbd> | Panel-Familie wechseln |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>S</kbd> | Bereich als Screenshot |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>X</kbd> | OCR eines Bereichs |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>R</kbd> | Bereich aufnehmen |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>C</kbd> | Rückwärtssuche für einen Bereich |
| <kbd>Super</kbd> + <kbd>L</kbd> | Sperren |
| <kbd>Ctrl</kbd> + <kbd>Alt</kbd> + <kbd>Delete</kbd> | Sitzungsbildschirm |

Die Kürzel zur Fensterverwaltung gehören dir — die Shell definiert sie nicht. Vollständige
Referenz: [Tastenkürzel](../KEYBINDS.md).

---

## Hintergrundbilder

15 Hintergrundbilder sind dabei. Mehr gibt es bei [iNiR-Walls](https://github.com/snowarch/iNiR-Walls),
einer Sammlung, die gut mit der Material-You-Pipeline funktioniert.

---

## Dokumentation (für niri, nicht für mango)

| Seite | Worum es geht |
|---|---|
| [Installation](../INSTALL.md) | Wie man es zum Laufen bringt |
| [Setup](../SETUP.md) | Updates, Migrationen, Rollback |
| [Tastenkürzel](../KEYBINDS.md) | Alle Kombinationen |
| [IPC](../IPC.md) | Ziele, die du auf eine Taste legen oder aus einem Skript aufrufen kannst |
| [Pakete](../PACKAGES.md) | Jede Abhängigkeit und warum sie da ist |
| [Einschränkungen](../LIMITATIONS.md) | Was bekannt kaputt ist und wie man es umgeht |
| [Compositor](../COMPOSITORS.md) | Wie die Compositor-Anbindung funktioniert |
| [Architektur](../../ARCHITECTURE.md) | Wie der Code aufgebaut ist |

Der Großteil von `docs/` stammt von upstream und beschreibt stellenweise noch niri. Wo die
Dokumentation und diese README sich darüber uneinig sind, welcher Compositor unterstützt
wird, hat diese README recht.

---

## Fehlersuche

```bash
inir logs                       # aktuelle Laufzeit-Logs
inir restart                    # aktive Laufzeit neu starten
inir repair                     # doctor + Neustart + gefilterte Log-Prüfung
inir doctor                     # Selbstdiagnose und Reparatur typischer Probleme
./setup rollback                # letztes Update rückgängig machen
claude "hilf mir bitte"         # falls du keine Lust hast, selbst zu suchen. na komm, die 20 $ muss er ja irgendwie abarbeiten
```

Schau in [Einschränkungen](../LIMITATIONS.md), um was zu lachen zu haben.

---

## Mitmachen

Siehe [CONTRIBUTING.md](../../CONTRIBUTING.md) — Entwicklungsumgebung, Code-Muster und
Regeln für Pull Requests.


---

## Dank

- [**snowarch**](https://github.com/snowarch/iNiR): iNiR, die hier portierte Shell
- [**end-4**](https://github.com/end-4/dots-hyprland): illogical-impulse, woraus iNiR geforkt wurde
- [**Gakuseei**](https://github.com/Gakuseei): [Ricelin](https://github.com/Gakuseei/Ricelin), woher die Pill-Leiste und der Washi- und Flame-Look stammen
- [**Quickshell**](https://quickshell.outfoxxed.me/): das Framework, auf dem das läuft
- [**MangoWM**](https://github.com/DreamMaoMao/mango): der Compositor, für den das gebaut ist
- **Claude** (Anthropic): hat den MangoWM-Port geschrieben, wie ganz oben gesagt

GPL-3.0, wie end-4s Dotfiles. Upstream-Copyright (C) 2025-2026 snowarch.
