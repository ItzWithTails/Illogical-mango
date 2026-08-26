<h1 align="center">Illogical-mango</h1>

<p align="center">
  <b>Una shell desktop completa per MangoWM, costruita su Quickshell</b>
</p>

<p align="center">
  <sub>
    <a href="../../README.md">English</a> · <a href="README.ru.md">Русский</a> · <a href="README.es.md">Español</a> · <a href="README.zh.md">中文</a> · <a href="README.ja.md">日本語</a> · <a href="README.pt.md">Português</a> · <a href="README.fr.md">Français</a> · <a href="README.de.md">Deutsch</a> · <a href="README.ko.md">한국어</a> · <a href="README.hi.md">हिन्दी</a> · <a href="README.ar.md">العربية</a> · <a href="README.it.md">Italiano</a>
  </sub>
</p>

---

## Questo port l'ha scritto un'IA. Del tutto. Anche questo README, al 90 %

Il progetto è una burla. Qui nessuno si è impegnato.

Il port su MangoWM - `services/MangoService.qml`,
`services/deferred/MangoKeybinds.qml`, la riscrittura del rilevamento del compositor in
`services/CompositorService.qml`, le modifiche all'installer e a doctor - è stato scritto
dall'inizio alla fine tramite Claude.
Non "con l'aiuto di". Scritto da lei.

Lo dico in cima così non lo scopri dopo, da un diff o da un bug. Non è un merito e non
viene presentato come tale. In sostanza il port l'ho fatto per me e per divertimento.
Tienine conto.

La shell sotto lo strato del port è l'iNiR di snowarch, scritta (spero) da un umano.

---

## Che cos'è

Sinceramente, se hai già installato da solo un compositor Wayland nudo e crudo, non c'è
bisogno di spiegarti perché gli serva una shell. Ma sono tenuto a spiegare come funziona.

```
le tue applicazioni
   ↓
Illogical-mango   barra, dock, barre laterali, panoramica, notifiche, impostazioni, blocco schermo
   ↓
Quickshell        runtime QML per shell Wayland
   ↓
MangoWM           finestre e rendering
   ↓
Wayland → GPU
```

**Cosa lo distingue dalle altre configurazioni Quickshell:**

- **Due famiglie di pannelli complete in una sola installazione.** Material ii (barra
  fluttuante, barre laterali, dock) e Waffle (barra delle applicazioni in basso, menu
  start, centro notifiche e azioni). Non sono temi sopra gli stessi widget — sono alberi di
  pannelli separati con i propri sistemi di token, scambiabili a caldo con
  <kbd>Super</kbd>+<kbd>Shift</kbd>+<kbd>W</kbd>.
- **Tematizzazione dell'intero sistema, non solo della shell.** Uno sfondo genera una
  palette Material You che viene scritta in GTK3/4, Qt, dieci terminali e strumenti TUI,
  Firefox, Discord, Spicetify, Steam e SDDM.
- **Si configura senza toccare il codice.** Tutto è un'impostazione della GUI sopra un
  unico `config.json`. Non serve mai mettere mano al QML per cambiare aspetto o
  comportamento.
- **Un vero percorso di installazione e aggiornamento.** `./setup` si occupa delle
  dipendenze e della configurazione di sistema; `ilmango update` fa pull, esegue le
  migrazioni dello schema, conserva le tue modifiche e sa tornare indietro.

**Origine.** [illogical-impulse di end-4](https://github.com/end-4/dots-hyprland) (dotfile
per Hyprland) → [iNiR di snowarch](https://github.com/snowarch/iNiR) (riscritto per niri) →
questo, portato su MangoWM. La CLI, i percorsi di configurazione e le viscere si chiamano
`ilmango`. Le installazioni dell'era iNiR vengono riprese dalla migrazione 037, che lascia
symlink sui vecchi percorsi in modo che scorciatoie e script esistenti continuino a funzionare.
Il nome è
rimasto.
Perché non forkare direttamente end-4? La logica è semplice - un progetto già portato una
volta si porta più facilmente.
Come analogia, prendi Void Linux. Installaci systemd e girerà tranquillamente.
Prendi Arch Linux e stacca systemd, e dovrai cambiare quasi tutta la base dei pacchetti.


## Compositor

Fatto per [MangoWM](https://github.com/DreamMaoMao/mango) e testato solo su quello.

La shell parla con mango attraverso il suo socket IPC su `$MANGO_INSTANCE_SIGNATURE`, che a
ogni cambiamento invia un'istantanea completa della sessione. mango è in stile dwm — tag,
non un elenco di spazi di lavoro —, quindi `MangoService` mappa le coppie `(monitor, indice
del tag)` sullo stesso modello di spazi di lavoro che barra, dock, panoramica e striscia
degli spazi di lavoro già si aspettano, e quei moduli funzionano senza modifiche.

La configurazione è deliberatamente non distruttiva. mango legge esattamente un file
(`~/.config/mango/config.conf`) e non unisce nulla, quindi l'installer non sovrascrive mai
la tua configurazione del compositor. Mette le scorciatoie e l'avvio automatico della shell
in `~/.config/mango/ilmango.conf` e aggiunge una sola riga `source-optional=` che punta lì,
senza toccare la tua gestione delle finestre. L'avvio automatico è una riga
`exec-once=ilmango run --daemon` in quel file, non un'unità systemd.

> [!NOTE]
> **Il codice di niri e Hyprland è ancora nell'albero.** `NiriService.qml`,
> `HyprlandData.qml` e i rami `isNiri` / `isHyprland` sopravvivono da upstream e compilano
> ancora. Sono ereditati, non supportati: qui non si testa nulla su quei compositor e non
> si mantiene nulla per loro. Se vuoi niri, prendi
> [l'iNiR originale](https://github.com/snowarch/iNiR).

---

## Screenshot

Entrambe le famiglie di pannelli, riprese da upstream senza modifiche.

<details open>
<summary><b>Material ii</b>: barra fluttuante, barre laterali, estetica Material Design</summary>

| | |
|:---:|:---:|
| ![](https://github.com/user-attachments/assets/1fe258bc-8aec-4fd9-8574-d9d7472c3cc8) | ![](https://github.com/user-attachments/assets/3ce2055b-648c-45a1-9d09-705c1b4a03b7) |
| ![](https://github.com/user-attachments/assets/ea2311dc-769e-44dc-a46d-37cf8807d2cc) | ![](https://github.com/user-attachments/assets/ba866063-b26a-47cb-83c8-d77bd033bf8b) |
| ![](https://github.com/user-attachments/assets/88e76566-061b-4f8c-a9a8-53c157950138) | |

</details>

<details>
<summary><b>Waffle</b>: barra delle applicazioni in basso, centro azioni, vibe Windows 11</summary>

| | |
|:---:|:---:|
| ![](https://github.com/user-attachments/assets/5c5996e7-90eb-4789-9921-0d5fe5283fa3) | ![](https://github.com/user-attachments/assets/fadf9562-751e-4138-a3a1-b87b31114d44) |

</details>

---

> [!WARNING]
> La configurazione predefinita punta a hardware ragionevolmente moderno. Su macchine
> deboli disattiva gli effetti, togli i pannelli che non usi e appiattisci lo stile visivo
> — tutto questo si fa dalle impostazioni o tramite `config.json`.

## Funzionalità

**Due famiglie di pannelli**, scambiabili a caldo con <kbd>Super</kbd>+<kbd>Shift</kbd>+<kbd>W</kbd>:

- **Material ii** — barra fluttuante, barre laterali, dock e 8 stili visivi (Material,
  Cards, Aurora, Illogical-mango, Angel, Regalia, ZZZ, Cookie Shapes)
- **Waffle** — barra delle applicazioni in stile Windows 11, menu start, centro azioni,
  centro notifiche

**Tematizzazione automatica.** Scegli uno sfondo e tutto il sistema lo segue: i colori
Material You della shell vengono propagati a GTK3/4, Qt, terminali, Firefox, Discord,
Spicetify, Steam e SDDM. Include i preset Regalia, Gruvbox, Catppuccin e Rosé Pine, oppure
costruisci il tuo.

<details>
<summary><b>Elenco completo delle funzionalità</b></summary>

### Tematizzazione e aspetto

- **8 stili visivi**: Material (pieno), Cards, Aurora (sfocatura vetro), Illogical-mango (ispirato alle TUI), Angel (neobrutalismo), Regalia (telaio nero da ingegneria, inchiostro avorio caldo, ferramenta champagne sobria), ZZZ (lastre da manifesto), Cookie Shapes (morphing animato delle forme)
- **Colori dinamici dallo sfondo** via Material You, propagati a tutto il sistema
- **10 terminali e strumenti TUI tematizzati in automatico**: foot, kitty, alacritty, ghostty, wezterm, starship, fuzzel, btop, lazygit, yazi
- **Tematizzazione delle app**: GTK3/4, Qt (tramite plasma-integration e darkly), Firefox (MaterialFox), Discord/Vesktop (System24), Zed, Spicetify, Steam, SDDM
- **Preset di temi**: Regalia, Regalia Ivory, Gruvbox, Catppuccin, Rosé Pine e personalizzati
- **Sfondi video**: mp4/webm/gif con sfocatura opzionale, o primo fotogramma congelato per le prestazioni
- **Widget sulla scrivania**: orologio (più stili), meteo, controlli multimediali sul livello dello sfondo

### Barra

- **6 stili di barra**: classic, islands, scenic, frame, capsule Material 3 e pill
- **Barra pill**: un'isola centrale che si trasforma e al passaggio del mouse si apre su spazi di lavoro, avviatore, mixer, media, calendario e registratore dello schermo
- **Disposizione modulare** con editor a trascinamento nelle impostazioni: ogni modulo va dove vuoi
- **Barra verticale** per le disposizioni lungo il bordo dello schermo

### Barre laterali e widget (Material ii)

Barra sinistra (cassetto delle applicazioni):
- **AI Chat**: cataloghi di modelli in tempo reale su Ollama, LM Studio, OpenRouter, Gemini, Groq, Mistral, Cerebras, Anthropic, OpenAI e OpenCode
- **YT Music**: lettore InnerTube senza cookie con ricerca, coda, radio e testi sincronizzati
- **Browser Wallhaven**: cerca e applica sfondi direttamente
- **Tracker anime**: integrazione con AniList e vista del palinsesto
- **Traduttore**: tramite Gemini o translate-shell
- **Widget trascinabili**: cripto, lettore multimediale, note rapide, anelli di stato, calendario settimanale

Barra destra:
- **Calendario** con integrazione degli eventi
- **Centro notifiche**
- **Interruttori rapidi**: WiFi, Bluetooth, luce notturna, non disturbare, profili di alimentazione, WARP VPN, EasyEffects
- **Mixer del volume** con controllo per applicazione
- **Gestione dispositivi** Bluetooth e WiFi
- **Timer Pomodoro**, **lista di cose da fare**, **calcolatrice**, **blocco note**
- **Monitor di sistema**: CPU, RAM, temperatura

### Strumenti

- **Panoramica degli spazi di lavoro**: ricerca app e calcolatrice, appoggiate sul modello a tag di mango
- **Dashboard**: overlay configurabile a tre colonne con agenda, notifiche, impegni, note, media e meteo
- **Striscia degli spazi di lavoro sul bordo**: guida al passaggio del mouse con anteprime in tempo reale e riordino per trascinamento
- **Selettore finestre**: Alt-Tab animato su tutti gli spazi di lavoro, opzionale
- **Gestore degli appunti**: cronologia con ricerca e anteprima delle immagini
- **Strumenti per aree**: schermate, registrazione dello schermo, OCR, ricerca inversa di immagini
- **Bigino**: visualizzatore delle scorciatoie estratte dalla tua configurazione di mango
- **Controlli multimediali**: lettore MPRIS completo con vari preset di disposizione
- **Indicatori a schermo**: OSD di volume, luminosità e media
- **Riconoscimento dei brani**: alla Shazam, tramite SongRec
- **Input vocale**: whisper.cpp locale se installato, o un backend collegato di Groq, Gemini o OpenAI

### Sistema

- **Impostazioni grafiche**: si configura tutto senza toccare i file
- **GameMode**: disattiva automaticamente gli effetti per le app a schermo intero
- **Aggiornamenti automatici**: `ilmango update` con ripristino, migrazioni e conservazione delle tue modifiche
- **Schermata di blocco** e **schermata di sessione** (disconnessione/riavvio/spegnimento/sospensione)
- **Agente polkit**, **tastiera a schermo**, **gestore dell'avvio automatico** basato sulla riga `exec-once` nella configurazione di mango
- **Kira**: mascotte opzionale in pixel art che gironzola sui bordi dello schermo e reagisce a quello che fai. Disattivata di default; il pacchetto di illustrazioni da ~32 MiB si scarica a parte da `./setup` › Extras
- **15 lingue** con rilevamento automatico
- **Luce notturna**: programmata o manuale
- **Meteo**: Open-Meteo, supporta GPS, coordinate manuali o nome della città
- **Gestione della batteria**: soglie configurabili, sospensione automatica al livello critico
- **Suoni degli eventi personalizzati** con volume generale e un file audio per evento

</details>

---

## Avvio rapido (in futuro l'installer sarà un altro)

```bash
git clone https://github.com/ItzWithTails/illogical-mango.git
cd Illogical-mango
./setup install       # interattivo, chiede prima di ogni passo
./setup install -y    # automatico, senza domande
```

L'installer si occupa di dipendenze, configurazione di sistema e tematizzazione. Scrive le
scorciatoie della shell in `~/.config/mango/ilmango.conf` e le aggancia alla tua
configurazione di mango esistente senza toccare la gestione delle finestre. Riavvia mango
oppure esegui `mmsg dispatch reload_config`.

```bash
ilmango run                        # avviare la shell
ilmango settings                   # aprire la GUI delle impostazioni
ilmango logs                       # guardare i log
ilmango doctor                     # autodiagnosi e riparazione
ilmango update                     # pull + migrazioni + riavvio
```

Altri punti di ingresso:

```bash
./setup                 # menu TUI, scegli tu cosa vuoi
./setup install --skip-mango    # non toccare affatto la configurazione di mango
sudo make install       # installazione di sistema invece che nella home
./setup rollback        # annullare l'ultimo aggiornamento
```

**Distribuzioni.** Arch è l'obiettivo principale e il più testato. Per Debian e Fedora il
port c'è, certo... a tuo rischio e pericolo, non è stato fatto alcun test su di esse.

---

## Scorciatoie

Installate da `defaults/mango/config.conf`:

| Tasto | Azione |
|-----|--------|
| <kbd>Super</kbd> + <kbd>Space</kbd> | Panoramica: ricerca app, navigazione tra i tag |
| <kbd>Super</kbd> + <kbd>V</kbd> | Cronologia degli appunti |
| <kbd>Super</kbd> + <kbd>P</kbd> | Barra laterale sinistra |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>N</kbd> | Barra laterale destra |
| <kbd>Super</kbd> + <kbd>D</kbd> | Dashboard |
| <kbd>Super</kbd> + <kbd>,</kbd> | Impostazioni |
| <kbd>Super</kbd> + <kbd>/</kbd> | Bigino |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>W</kbd> | Cambiare famiglia di pannelli |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>S</kbd> | Schermata di un'area |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>X</kbd> | OCR di un'area |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>R</kbd> | Registrare un'area |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>C</kbd> | Ricerca inversa di un'area |
| <kbd>Super</kbd> + <kbd>L</kbd> | Blocco |
| <kbd>Ctrl</kbd> + <kbd>Alt</kbd> + <kbd>Delete</kbd> | Schermata di sessione |

Le scorciatoie per la gestione delle finestre sono tue — la shell non le definisce.
Riferimento completo: [Scorciatoie](../KEYBINDS.md).

---

## Sfondi

Sono inclusi 15 sfondi. Per averne altri, vedi [iNiR-Walls](https://github.com/snowarch/iNiR-Walls),
una raccolta che funziona bene con la pipeline Material You.

---

## Documentazione (per niri, non per mango)

| Pagina | Di cosa parla |
|---|---|
| [Installazione](../INSTALL.md) | Come metterlo in funzione |
| [Setup](../SETUP.md) | Aggiornamenti, migrazioni, ripristino |
| [Scorciatoie](../KEYBINDS.md) | Tutte le combinazioni |
| [IPC](../IPC.md) | Obiettivi da associare a un tasto o da chiamare da uno script |
| [Pacchetti](../PACKAGES.md) | Ogni dipendenza e perché c'è |
| [Limitazioni](../LIMITATIONS.md) | Cosa è rotto e come aggirarlo |
| [Compositor](../COMPOSITORS.md) | Come funziona l'integrazione con il compositor |
| [Architettura](../../ARCHITECTURE.md) | Com'è messo insieme il codice |

Gran parte di `docs/` è ereditata da upstream e in alcuni punti descrive ancora niri. Dove
la documentazione e questo README non concordano su quale compositor sia supportato, ha
ragione questo README.

---

## Risoluzione dei problemi

```bash
ilmango logs                       # log recenti del runtime
ilmango restart                    # riavviare il runtime attivo
ilmango repair                     # doctor + riavvio + controllo filtrato dei log
ilmango doctor                     # autodiagnosi e riparazione dei problemi tipici
./setup rollback                # annullare l'ultimo aggiornamento
claude "aiutami per favore"     # se non hai voglia di scervellarti. dai, i suoi 20 $ deve pur guadagnarseli
```

Dai un'occhiata alle [Limitazioni](../LIMITATIONS.md), giusto per farti due risate.

---

## Contribuire

Vedi [CONTRIBUTING.md](../../CONTRIBUTING.md) — ambiente di sviluppo, schemi di codice e
regole per le pull request.


---

## Ringraziamenti

- [**snowarch**](https://github.com/snowarch/iNiR): iNiR, la shell qui portata
- [**end-4**](https://github.com/end-4/dots-hyprland): illogical-impulse, da cui è stato forkato iNiR
- [**Gakuseei**](https://github.com/Gakuseei): [Ricelin](https://github.com/Gakuseei/Ricelin), da cui vengono la barra pill e il look washi e flame
- [**Quickshell**](https://quickshell.outfoxxed.me/): il framework su cui gira il tutto
- [**MangoWM**](https://github.com/DreamMaoMao/mango): il compositor per cui è fatto
- **Claude** (Anthropic): ha scritto il port su MangoWM, come detto in cima

GPL-3.0, come i dotfile di end-4. Copyright upstream (C) 2025-2026 snowarch.
