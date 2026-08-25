<h1 align="center">Illogical-mango</h1>

<p align="center">
  <b>Un shell de bureau complet pour MangoWM, bâti sur Quickshell</b>
</p>

<p align="center">
  <sub>
    <a href="../../README.md">English</a> · <a href="README.ru.md">Русский</a> · <a href="README.es.md">Español</a> · <a href="README.zh.md">中文</a> · <a href="README.ja.md">日本語</a> · <a href="README.pt.md">Português</a> · <a href="README.fr.md">Français</a> · <a href="README.de.md">Deutsch</a> · <a href="README.ko.md">한국어</a> · <a href="README.hi.md">हिन्दी</a> · <a href="README.ar.md">العربية</a> · <a href="README.it.md">Italiano</a>
  </sub>
</p>

---

## Ce portage a été écrit par une IA. Entièrement. Ce README aussi, à 90 %

Le projet est une blague. Personne ne s'est appliqué ici.

Le portage vers MangoWM - `services/MangoService.qml`,
`services/deferred/MangoKeybinds.qml`, la refonte de la détection du compositeur dans
`services/CompositorService.qml`, les changements dans l'installeur et dans doctor - a été
écrit du début à la fin via Claude.
Pas "avec l'aide de". Écrit par elle.

C'est dit tout en haut pour que tu ne l'apprennes pas plus tard, dans un diff ou dans un
bug. Ce n'est pas un exploit et ce n'est pas présenté comme tel. Au fond, j'ai fait ce
portage pour moi et pour rigoler. Prends-le en compte.

Le shell sous la couche de portage, c'est l'iNiR de snowarch, écrit (j'espère) par un
humain.

---

## Ce que c'est

Honnêtement, si tu as déjà installé toi-même un compositeur Wayland nu, personne n'a
besoin de t'expliquer pourquoi il lui faut un shell. Mais je suis obligé d'expliquer
comment ça marche.

```
tes applications
   ↓
Illogical-mango   barre, dock, panneaux latéraux, vue d'ensemble, notifications, réglages, écran de verrouillage
   ↓
Quickshell        moteur QML pour shells Wayland
   ↓
MangoWM           fenêtres et rendu
   ↓
Wayland → GPU
```

**Ce qui le distingue des autres configurations Quickshell :**

- **Deux familles de panneaux complètes dans une seule installation.** Material ii (barre
  flottante, panneaux latéraux, dock) et Waffle (barre des tâches en bas, menu démarrer,
  centre d'actions). Ce ne sont pas des thèmes posés sur les mêmes widgets — ce sont des
  arbres de panneaux distincts avec leurs propres systèmes de jetons, échangeables à chaud
  avec <kbd>Super</kbd>+<kbd>Shift</kbd>+<kbd>W</kbd>.
- **Thématisation de tout le système, pas seulement du shell.** Un fond d'écran génère une
  palette Material You écrite vers GTK3/4, Qt, dix terminaux et outils TUI, Firefox,
  Discord, Spicetify, Steam et SDDM.
- **Configurable sans toucher au code.** Tout est un réglage de l'interface au-dessus d'un
  unique `config.json`. Jamais besoin de toucher au QML pour changer l'apparence ou le
  comportement.
- **Un vrai chemin d'installation et de mise à jour.** `./setup` s'occupe des dépendances
  et de la configuration système ; `inir update` fait un pull, exécute les migrations de
  schéma, préserve tes modifications et sait revenir en arrière.

**Origine.** [illogical-impulse de end-4](https://github.com/end-4/dots-hyprland) (dotfiles
Hyprland) → [iNiR de snowarch](https://github.com/snowarch/iNiR) (réécrit pour niri) →
ceci, porté sur MangoWM. La CLI, les chemins de configuration et les entrailles s'appellent
toujours `inir` : les renommer casserait tous les chemins de mise à jour, donc le nom est
resté.
Pourquoi ne pas forker end-4 directement ? La logique est simple - un projet déjà porté une
fois se reporte plus facilement.
Par analogie, prends Void Linux. Installe systemd dessus et il tournera très bien.
Prends Arch Linux et arrache systemd, et il faudra changer presque toute la base de paquets.


## Compositeur

Fait pour [MangoWM](https://github.com/DreamMaoMao/mango) et testé uniquement dessus.

Le shell parle à mango via sa socket IPC sur `$MANGO_INSTANCE_SIGNATURE`, qui envoie un
instantané complet de la session à chaque changement. mango est de style dwm — des
étiquettes, pas une liste d'espaces de travail —, donc `MangoService` fait correspondre les
paires `(moniteur, index d'étiquette)` au même modèle d'espaces de travail qu'attendent
déjà la barre, le dock, la vue d'ensemble et la bande d'espaces de travail, et ces modules
fonctionnent sans modification.

La configuration est délibérément non destructive. mango lit exactement un fichier
(`~/.config/mango/config.conf`) et ne fusionne rien, donc l'installeur n'écrase jamais ta
configuration de compositeur. Il place les raccourcis et le démarrage automatique du shell
dans `~/.config/mango/inir.conf` et ajoute une seule ligne `source-optional=` qui pointe
dessus, sans toucher à ta gestion de fenêtres. Le démarrage automatique est une ligne
`exec-once=inir run --daemon` dans ce fichier, pas une unité systemd.

> [!NOTE]
> **Le code niri et Hyprland est toujours dans l'arbre.** `NiriService.qml`,
> `HyprlandData.qml` et les branches `isNiri` / `isHyprland` ont survécu depuis l'amont et
> compilent encore. Ils sont hérités, pas supportés : rien ici n'est testé contre ces
> compositeurs et rien n'est maintenu pour eux. Si tu veux niri, prends
> [l'iNiR original](https://github.com/snowarch/iNiR).

---

## Captures d'écran

Les deux familles de panneaux, reprises de l'amont sans modification.

<details open>
<summary><b>Material ii</b> : barre flottante, panneaux latéraux, esthétique Material Design</summary>

| | |
|:---:|:---:|
| ![](https://github.com/user-attachments/assets/1fe258bc-8aec-4fd9-8574-d9d7472c3cc8) | ![](https://github.com/user-attachments/assets/3ce2055b-648c-45a1-9d09-705c1b4a03b7) |
| ![](https://github.com/user-attachments/assets/ea2311dc-769e-44dc-a46d-37cf8807d2cc) | ![](https://github.com/user-attachments/assets/ba866063-b26a-47cb-83c8-d77bd033bf8b) |
| ![](https://github.com/user-attachments/assets/88e76566-061b-4f8c-a9a8-53c157950138) | |

</details>

<details>
<summary><b>Waffle</b> : barre des tâches en bas, centre d'actions, ambiance Windows 11</summary>

| | |
|:---:|:---:|
| ![](https://github.com/user-attachments/assets/5c5996e7-90eb-4789-9921-0d5fe5283fa3) | ![](https://github.com/user-attachments/assets/fadf9562-751e-4138-a3a1-b87b31114d44) |

</details>

---

> [!WARNING]
> La configuration par défaut vise du matériel raisonnablement moderne. Sur des machines
> faibles, désactive les effets, retire les panneaux dont tu n'as pas besoin et aplatis le
> style visuel — tout cela se fait depuis les réglages ou via `config.json`.

## Fonctionnalités

**Deux familles de panneaux**, échangeables à chaud avec <kbd>Super</kbd>+<kbd>Shift</kbd>+<kbd>W</kbd> :

- **Material ii** — barre flottante, panneaux latéraux, dock et 8 styles visuels (Material,
  Cards, Aurora, iNiR, Angel, Regalia, ZZZ, Cookie Shapes)
- **Waffle** — barre des tâches façon Windows 11, menu démarrer, centre d'actions, centre
  de notifications

**Thématisation automatique.** Tu choisis un fond d'écran et tout le système suit : les
couleurs Material You du shell sont propagées vers GTK3/4, Qt, les terminaux, Firefox,
Discord, Spicetify, Steam et SDDM. Livré avec les préréglages Regalia, Gruvbox, Catppuccin
et Rosé Pine, ou fabrique le tien.

<details>
<summary><b>Liste complète des fonctionnalités</b></summary>

### Thématisation et apparence

- **8 styles visuels** : Material (plein), Cards, Aurora (flou de verre), iNiR (inspiré TUI), Angel (néo-brutalisme), Regalia (châssis noir d'ingénierie, encre ivoire chaude, quincaillerie champagne discrète), ZZZ (plaques d'affiche), Cookie Shapes (morphing animé des formes)
- **Couleurs dynamiques issues du fond d'écran** via Material You, propagées à tout le système
- **10 terminaux et outils TUI thématisés automatiquement** : foot, kitty, alacritty, ghostty, wezterm, starship, fuzzel, btop, lazygit, yazi
- **Thématisation des applications** : GTK3/4, Qt (via plasma-integration et darkly), Firefox (MaterialFox), Discord/Vesktop (System24), Zed, Spicetify, Steam, SDDM
- **Préréglages de thèmes** : Regalia, Regalia Ivory, Gruvbox, Catppuccin, Rosé Pine et personnalisés
- **Fonds d'écran vidéo** : mp4/webm/gif avec flou optionnel, ou première image figée pour les performances
- **Widgets de bureau** : horloge (plusieurs styles), météo, contrôles média sur la couche du fond d'écran

### Barre

- **6 styles de barre** : classic, islands, scenic, frame, capsules Material 3 et pill
- **Barre pill** : un îlot central qui se métamorphose et s'ouvre au survol sur les espaces de travail, le lanceur, le mixeur, les médias, le calendrier et un enregistreur d'écran
- **Disposition modulaire** avec un éditeur par glisser-déposer dans les réglages : n'importe quel module va n'importe où
- **Barre verticale** pour les dispositions le long du bord de l'écran

### Panneaux latéraux et widgets (Material ii)

Panneau gauche (tiroir d'applications) :
- **AI Chat** : catalogues de modèles en direct sur Ollama, LM Studio, OpenRouter, Gemini, Groq, Mistral, Cerebras, Anthropic, OpenAI et OpenCode
- **YT Music** : lecteur InnerTube sans cookies avec recherche, file d'attente, radio et paroles synchronisées
- **Navigateur Wallhaven** : cherche et applique des fonds d'écran directement
- **Suivi d'animes** : intégration AniList avec vue calendrier
- **Traducteur** : via Gemini ou translate-shell
- **Widgets déplaçables** : crypto, lecteur média, notes rapides, anneaux d'état, calendrier hebdomadaire

Panneau droit :
- **Calendrier** avec intégration des évènements
- **Centre de notifications**
- **Bascules rapides** : WiFi, Bluetooth, lumière nocturne, ne pas déranger, profils d'énergie, WARP VPN, EasyEffects
- **Mixeur de volume** avec contrôle par application
- **Gestion des appareils** Bluetooth et WiFi
- **Minuteur Pomodoro**, **liste de tâches**, **calculatrice**, **bloc-notes**
- **Moniteur système** : CPU, RAM, température

### Outils

- **Vue d'ensemble des espaces de travail** : recherche d'applications et calculatrice, posées sur le modèle d'étiquettes de mango
- **Tableau de bord** : superposition configurable à trois colonnes avec agenda, notifications, tâches, notes, médias et météo
- **Bande d'espaces de travail au bord** : rail au survol avec aperçus en direct et réordonnancement par glisser
- **Sélecteur de fenêtres** : Alt-Tab animé sur tous les espaces de travail, optionnel
- **Gestionnaire de presse-papiers** : historique avec recherche et aperçu des images
- **Outils de zone** : captures, enregistrement d'écran, OCR, recherche d'image inversée
- **Antisèche** : visionneuse de raccourcis tirée de ta configuration mango
- **Contrôles média** : lecteur MPRIS complet avec plusieurs dispositions
- **Affichage à l'écran** : OSD de volume, luminosité et médias
- **Reconnaissance de chansons** : façon Shazam, via SongRec
- **Saisie vocale** : whisper.cpp local s'il est installé, ou un backend connecté Groq, Gemini ou OpenAI

### Système

- **Réglages graphiques** : tout se configure sans toucher aux fichiers
- **GameMode** : désactive automatiquement les effets pour les applications en plein écran
- **Mises à jour automatiques** : `inir update` avec retour arrière, migrations et préservation de tes modifications
- **Écran de verrouillage** et **écran de session** (déconnexion/redémarrage/extinction/veille)
- **Agent polkit**, **clavier virtuel**, **gestionnaire de démarrage automatique** adossé à la ligne `exec-once` de la configuration mango
- **Kira** : mascotte pixel art optionnelle qui erre sur les bords de l'écran et réagit à ce que tu fais. Désactivée par défaut ; le pack d'illustrations d'environ 32 Mio se télécharge à part dans `./setup` › Extras
- **15 langues** avec détection automatique
- **Lumière nocturne** : programmée ou manuelle
- **Météo** : Open-Meteo, gère le GPS, des coordonnées manuelles ou un nom de ville
- **Gestion de la batterie** : seuils configurables, mise en veille automatique au niveau critique
- **Sons d'évènements personnalisés** avec un volume général et un fichier audio par évènement

</details>

---

## Démarrage rapide (l'installeur sera différent à l'avenir)

```bash
git clone https://github.com/ItzWithTails/Illogical-mango.git
cd Illogical-mango
./setup install       # interactif, demande avant chaque étape
./setup install -y    # automatique, sans questions
```

L'installeur s'occupe des dépendances, de la configuration système et de la thématisation.
Il écrit les raccourcis du shell dans `~/.config/mango/inir.conf` et les branche à ta
configuration mango existante sans toucher à ta gestion de fenêtres. Redémarre mango ou
lance `mmsg dispatch reload_config`.

```bash
inir run                        # lancer le shell
inir settings                   # ouvrir l'interface de réglages
inir logs                       # consulter les logs
inir doctor                     # autodiagnostic et réparation
inir update                     # pull + migrations + redémarrage
```

Autres points d'entrée :

```bash
./setup                 # menu TUI, tu choisis ce que tu veux
./setup install --skip-mango    # ne pas toucher du tout à la configuration mango
sudo make install       # installation système au lieu du home
./setup rollback        # annuler la dernière mise à jour
```

**Distributions.** Arch est la cible principale et la mieux testée. Pour Debian et Fedora
il y a un portage, bien sûr... à tes risques et périls, aucun test n'a été fait dessus.

---

## Raccourcis

Installés depuis `defaults/mango/config.conf` :

| Touche | Action |
|-----|--------|
| <kbd>Super</kbd> + <kbd>Space</kbd> | Vue d'ensemble : recherche d'applications, navigation par étiquettes |
| <kbd>Super</kbd> + <kbd>V</kbd> | Historique du presse-papiers |
| <kbd>Super</kbd> + <kbd>P</kbd> | Panneau gauche |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>N</kbd> | Panneau droit |
| <kbd>Super</kbd> + <kbd>D</kbd> | Tableau de bord |
| <kbd>Super</kbd> + <kbd>,</kbd> | Réglages |
| <kbd>Super</kbd> + <kbd>/</kbd> | Antisèche |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>W</kbd> | Changer de famille de panneaux |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>S</kbd> | Capture d'une zone |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>X</kbd> | OCR d'une zone |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>R</kbd> | Enregistrer une zone |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>C</kbd> | Recherche inversée d'une zone |
| <kbd>Super</kbd> + <kbd>L</kbd> | Verrouiller |
| <kbd>Ctrl</kbd> + <kbd>Alt</kbd> + <kbd>Delete</kbd> | Écran de session |

Les raccourcis de gestion de fenêtres sont les tiens — le shell ne les définit pas.
Référence complète : [Raccourcis](../KEYBINDS.md).

---

## Fonds d'écran

15 fonds d'écran sont fournis. Pour d'autres, vois [iNiR-Walls](https://github.com/snowarch/iNiR-Walls),
une collection qui fonctionne bien avec le pipeline Material You.

---

## Documentation (pour niri, pas pour mango)

| Page | Contenu |
|---|---|
| [Installation](../INSTALL.md) | Comment le faire tourner |
| [Setup](../SETUP.md) | Mises à jour, migrations, retour arrière |
| [Raccourcis](../KEYBINDS.md) | Toutes les combinaisons |
| [IPC](../IPC.md) | Cibles à associer à une touche ou à appeler depuis un script |
| [Paquets](../PACKAGES.md) | Chaque dépendance et pourquoi elle est là |
| [Limitations](../LIMITATIONS.md) | Ce qui est cassé et comment contourner |
| [Compositeurs](../COMPOSITORS.md) | Comment fonctionne l'intégration au compositeur |
| [Architecture](../../ARCHITECTURE.md) | Comment le code est assemblé |

L'essentiel de `docs/` est hérité de l'amont et décrit encore niri par endroits. Là où la
documentation et ce README divergent sur le compositeur supporté, c'est ce README qui a
raison.

---

## Dépannage

```bash
inir logs                       # logs récents du runtime
inir restart                    # redémarrer le runtime actif
inir repair                     # doctor + redémarrage + vérification filtrée des logs
inir doctor                     # autodiagnostic et réparation des problèmes courants
./setup rollback                # annuler la dernière mise à jour
claude "aide-moi s'il te plaît" # si tu n'as pas envie de chercher toi-même. allez, il faut bien qu'il gagne ses 20 $
```

Jette un œil aux [Limitations](../LIMITATIONS.md) pour rigoler.

---

## Contribuer

Voir [CONTRIBUTING.md](../../CONTRIBUTING.md) — environnement de développement, motifs de
code et règles pour les pull requests.


---

## Remerciements

- [**snowarch**](https://github.com/snowarch/iNiR) : iNiR, le shell porté ici
- [**end-4**](https://github.com/end-4/dots-hyprland) : illogical-impulse, dont iNiR est un fork
- [**Gakuseei**](https://github.com/Gakuseei) : [Ricelin](https://github.com/Gakuseei/Ricelin), d'où viennent la barre pill et le look washi et flame
- [**Quickshell**](https://quickshell.outfoxxed.me/) : le framework sur lequel tout ça tourne
- [**MangoWM**](https://github.com/DreamMaoMao/mango) : le compositeur visé
- **Claude** (Anthropic) : a écrit le portage MangoWM, comme dit tout en haut

GPL-3.0, comme les dotfiles de end-4. Copyright amont (C) 2025-2026 snowarch.
