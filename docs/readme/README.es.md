<h1 align="center">Illogical-mango</h1>

<p align="center">
  <b>Un shell de escritorio completo para MangoWM, construido sobre Quickshell</b>
</p>

<p align="center">
  <sub>
    <a href="../../README.md">English</a> · <a href="README.ru.md">Русский</a> · <a href="README.es.md">Español</a> · <a href="README.zh.md">中文</a> · <a href="README.ja.md">日本語</a> · <a href="README.pt.md">Português</a> · <a href="README.fr.md">Français</a> · <a href="README.de.md">Deutsch</a> · <a href="README.ko.md">한국어</a> · <a href="README.hi.md">हिन्दी</a> · <a href="README.ar.md">العربية</a> · <a href="README.it.md">Italiano</a>
  </sub>
</p>

---

## Este port lo escribió una IA. Del todo. Este README también, un 90%

El proyecto es una broma. Nadie se esforzó aquí.

El port a MangoWM - `services/MangoService.qml`, `services/deferred/MangoKeybinds.qml`,
la reescritura de la detección de compositor en `services/CompositorService.qml`, los
cambios en el instalador y en doctor - fue escrito de principio a fin a través de Claude.
No "con ayuda de". Escrito por ella.

Esto se dice arriba del todo para que no te enteres después, en un diff o en un bug. No es
un mérito ni se presenta como tal. En el fondo hice este port para mí y por diversión.
Tenlo en cuenta.

El shell que hay debajo de la capa del port es el iNiR de snowarch, escrito (espero) por
un humano.

---

## Qué es esto

La verdad, si alguna vez has instalado tú mismo un compositor Wayland pelado, no hace
falta explicarte por qué necesita un shell. Pero estoy obligado a explicar cómo funciona.

```
tus aplicaciones
   ↓
Illogical-mango   barra, dock, paneles laterales, vista general, notificaciones, ajustes, bloqueo
   ↓
Quickshell        entorno QML para shells de Wayland
   ↓
MangoWM           ventanas y renderizado
   ↓
Wayland → GPU
```

**Qué lo diferencia de otras configuraciones de Quickshell:**

- **Dos familias de paneles completas en una sola instalación.** Material ii (barra
  flotante, paneles laterales, dock) y Waffle (barra de tareas inferior, menú de inicio,
  centro de acciones). No son temas sobre los mismos widgets — son árboles de paneles
  distintos con sus propios sistemas de tokens, y se cambian en caliente con
  <kbd>Super</kbd>+<kbd>Shift</kbd>+<kbd>W</kbd>.
- **Tematización de todo el sistema, no solo del shell.** Un fondo de pantalla genera una
  paleta Material You que se escribe en GTK3/4, Qt, diez terminales y utilidades TUI,
  Firefox, Discord, Spicetify, Steam y SDDM.
- **Se configura sin tocar código.** Todo es un ajuste de la GUI sobre un único
  `config.json`. Nunca hace falta tocar QML para cambiar su aspecto o comportamiento.
- **Un camino de instalación y actualización de verdad.** `./install` se encarga de las
  dependencias y la configuración del sistema; `ilmango update` hace pull, ejecuta las
  migraciones de esquema, conserva tus cambios y sabe revertir.

**Origen.** [illogical-impulse de end-4](https://github.com/end-4/dots-hyprland) (dotfiles
de Hyprland) → [iNiR de snowarch](https://github.com/snowarch/iNiR) (reescrito para niri)
→ esto, portado a MangoWM. La CLI, las rutas de configuración y las tripas se siguen
llaman `ilmango`. Las instalaciones de la época de iNiR las migra la migración 037, que deja
enlaces simbólicos en las rutas antiguas para que los atajos y scripts existentes sigan
funcionando. El nombre
se quedó.
¿Por qué no forkear directamente a end-4? La lógica es simple - un proyecto que ya se ha
portado una vez es más fácil de portar otra.
Como analogía, toma Void Linux. Instálale systemd y funcionará sin problemas.
Toma Arch Linux y arráncale systemd, y tendrás que cambiar casi toda la base de paquetes.


## Compositor

Hecho para [MangoWM](https://github.com/DreamMaoMao/mango) y probado solo en él.

El shell habla con mango por su socket IPC en `$MANGO_INSTANCE_SIGNATURE`, que envía una
instantánea completa de la sesión en cada cambio. mango es de estilo dwm — etiquetas, no
una lista de escritorios —, así que `MangoService` mapea pares `(monitor, índice de
etiqueta)` sobre el mismo modelo de escritorios que ya esperan la barra, el dock, la vista
general y la tira de escritorios, y esos módulos funcionan sin cambios.

La configuración es deliberadamente no destructiva. mango lee exactamente un archivo
(`~/.config/mango/config.conf`) y no fusiona nada, así que el instalador nunca sobrescribe
tu configuración del compositor. Deja los atajos y el autoarranque del shell en
`~/.config/mango/ilmango.conf` y añade una sola línea `source-optional=` que apunta ahí, sin
tocar tu gestión de ventanas. El autoarranque es una línea `exec-once=ilmango run --daemon`
en ese archivo, no una unidad de systemd.

> [!NOTE]
> **El código de niri y Hyprland sigue en el árbol.** `NiriService.qml`,
> `HyprlandData.qml` y las ramas `isNiri` / `isHyprland` sobreviven de upstream y todavía
> compilan. Son heredadas, no soportadas: aquí no se prueba nada contra esos compositores
> ni se mantiene nada para ellos. Si quieres niri, coge
> [el iNiR original](https://github.com/snowarch/iNiR).

---

## Capturas

Ambas familias de paneles, traídas de upstream sin cambios.

<details open>
<summary><b>Material ii</b>: barra flotante, paneles laterales, estética Material Design</summary>

| | |
|:---:|:---:|
| ![](https://github.com/user-attachments/assets/1fe258bc-8aec-4fd9-8574-d9d7472c3cc8) | ![](https://github.com/user-attachments/assets/3ce2055b-648c-45a1-9d09-705c1b4a03b7) |
| ![](https://github.com/user-attachments/assets/ea2311dc-769e-44dc-a46d-37cf8807d2cc) | ![](https://github.com/user-attachments/assets/ba866063-b26a-47cb-83c8-d77bd033bf8b) |
| ![](https://github.com/user-attachments/assets/88e76566-061b-4f8c-a9a8-53c157950138) | |

</details>

<details>
<summary><b>Waffle</b>: barra de tareas inferior, centro de acciones, rollo Windows 11</summary>

| | |
|:---:|:---:|
| ![](https://github.com/user-attachments/assets/5c5996e7-90eb-4789-9921-0d5fe5283fa3) | ![](https://github.com/user-attachments/assets/fadf9562-751e-4138-a3a1-b87b31114d44) |

</details>

---

> [!WARNING]
> La configuración por defecto apunta a hardware razonablemente moderno. En máquinas
> flojas, desactiva los efectos, quita los paneles que no uses y aplana el estilo visual —
> todo eso se hace desde los ajustes o mediante `config.json`.

## Características

**Dos familias de paneles**, se cambian en caliente con <kbd>Super</kbd>+<kbd>Shift</kbd>+<kbd>W</kbd>:

- **Material ii** — barra flotante, paneles laterales, dock y 8 estilos visuales
  (Material, Cards, Aurora, Illogical-mango, Angel, Regalia, ZZZ, Cookie Shapes)
- **Waffle** — barra de tareas al estilo Windows 11, menú de inicio, centro de acciones,
  centro de notificaciones

**Tematización automática.** Eliges un fondo y todo el sistema le sigue: los colores
Material You del shell se propagan a GTK3/4, Qt, terminales, Firefox, Discord, Spicetify,
Steam y SDDM. Incluye los presets Regalia, Gruvbox, Catppuccin y Rosé Pine, o crea el tuyo.

<details>
<summary><b>Lista completa de características</b></summary>

### Tematización y aspecto

- **8 estilos visuales**: Material (sólido), Cards, Aurora (desenfoque de cristal), Illogical-mango (inspirado en TUI), Angel (neobrutalismo), Regalia (chasis negro de ingeniería, tinta marfil cálida, herrajes champán contenidos), ZZZ (planchas de cartel), Cookie Shapes (morfeo animado de formas)
- **Colores dinámicos del fondo de pantalla** vía Material You, propagados a todo el sistema
- **10 terminales y utilidades TUI tematizadas automáticamente**: foot, kitty, alacritty, ghostty, wezterm, starship, fuzzel, btop, lazygit, yazi
- **Tematización de apps**: GTK3/4, Qt (vía plasma-integration y darkly), Firefox (MaterialFox), Discord/Vesktop (System24), Zed, Spicetify, Steam, SDDM
- **Presets de temas**: Regalia, Regalia Ivory, Gruvbox, Catppuccin, Rosé Pine y personalizados
- **Fondos de vídeo**: mp4/webm/gif con desenfoque opcional, o primer fotograma congelado por rendimiento
- **Widgets de escritorio**: reloj (varios estilos), tiempo, controles multimedia sobre la capa del fondo

### Barra

- **6 estilos de barra**: classic, islands, scenic, frame, cápsulas Material 3 y pill
- **Barra pill**: una isla central que muta y se abre al pasar el ratón en escritorios, lanzador, mezclador, multimedia, calendario y grabador de pantalla
- **Diseño modular** con un editor de arrastrar en los ajustes: cualquier módulo va donde quieras
- **Barra vertical** para quien prefiere el borde de la pantalla

### Paneles laterales y widgets (Material ii)

Panel izquierdo (cajón de aplicaciones):
- **AI Chat**: catálogos de modelos en vivo de Ollama, LM Studio, OpenRouter, Gemini, Groq, Mistral, Cerebras, Anthropic, OpenAI y OpenCode
- **YT Music**: reproductor InnerTube sin cookies con búsqueda, cola, radio y letras sincronizadas
- **Navegador de Wallhaven**: busca y aplica fondos directamente
- **Seguimiento de anime**: integración con AniList y vista de calendario
- **Traductor**: vía Gemini o translate-shell
- **Widgets arrastrables**: cripto, reproductor, notas rápidas, anillos de estado, calendario semanal

Panel derecho:
- **Calendario** con integración de eventos
- **Centro de notificaciones**
- **Interruptores rápidos**: WiFi, Bluetooth, luz nocturna, no molestar, perfiles de energía, WARP VPN, EasyEffects
- **Mezclador de volumen** con control por aplicación
- **Gestión de dispositivos** Bluetooth y WiFi
- **Temporizador Pomodoro**, **lista de tareas**, **calculadora**, **bloc de notas**
- **Monitor del sistema**: CPU, RAM, temperatura

### Herramientas

- **Vista general de escritorios**: búsqueda de apps y calculadora, apoyadas en el modelo de etiquetas de mango
- **Panel de control**: superposición configurable a tres columnas con agenda, notificaciones, tareas, notas, multimedia y tiempo
- **Tira de escritorios en el borde**: raíl al pasar el ratón con vistas previas en vivo y reordenado por arrastre
- **Cambiador de ventanas**: Alt-Tab animado por todos los escritorios, opcional
- **Gestor del portapapeles**: historial con búsqueda y vista previa de imágenes
- **Herramientas de región**: capturas, grabación de pantalla, OCR, búsqueda inversa de imágenes
- **Chuleta**: visor de atajos sacado de tu configuración de mango
- **Controles multimedia**: reproductor MPRIS completo con varios diseños
- **Indicadores en pantalla**: OSD de volumen, brillo y multimedia
- **Reconocimiento de canciones**: al estilo Shazam, vía SongRec
- **Entrada por voz**: whisper.cpp local si está instalado, o un backend conectado de Groq, Gemini u OpenAI

### Sistema

- **Ajustes con GUI**: se configura todo sin tocar archivos
- **GameMode**: desactiva los efectos automáticamente en apps a pantalla completa
- **Actualizaciones automáticas**: `ilmango update` con reversión, migraciones y conservación de tus cambios
- **Pantalla de bloqueo** y **pantalla de sesión** (cerrar sesión/reiniciar/apagar/suspender)
- **Agente polkit**, **teclado en pantalla**, **gestor de autoarranque** apoyado en la línea `exec-once` de la configuración de mango
- **Kira**: mascota opcional en pixel art que deambula por los bordes de la pantalla y reacciona a lo que haces. Desactivada por defecto; el pack de arte de ~32 MiB se descarga aparte en `./install` › Extras
- **15 idiomas** con detección automática
- **Luz nocturna**: programada o manual
- **Tiempo**: Open-Meteo, admite GPS, coordenadas manuales o nombre de ciudad
- **Gestión de batería**: umbrales configurables, suspensión automática en nivel crítico
- **Sonidos de eventos propios** con volumen general y un archivo por evento

</details>

---

## Inicio rápido (el instalador será otro en el futuro)

```bash
git clone https://github.com/ItzWithTails/illogical-mango.git
cd Illogical-mango
./install       # interactivo, pregunta antes de cada paso
./install -y    # automático, sin preguntas
```

El instalador se encarga de las dependencias, la configuración del sistema y la
tematización. Escribe los atajos del shell en `~/.config/mango/ilmango.conf` y los engancha a
tu configuración de mango existente sin tocar tu gestión de ventanas. Reinicia mango o
ejecuta `mmsg dispatch reload_config`.

```bash
ilmango run                        # arrancar el shell
ilmango settings                   # abrir la GUI de ajustes
ilmango logs                       # mirar los logs
ilmango doctor                     # autodiagnóstico y arreglo
ilmango update                     # pull + migraciones + reinicio
```

Otros puntos de entrada:

```bash
./install                 # menú TUI, eliges lo que quieras
./install --disable mango    # no tocar en absoluto la configuración de mango
sudo make install       # instalación de sistema en vez de en tu home
./install --rollback        # deshacer la última actualización
```

**Distribuciones.** Arch es el objetivo principal y el mejor probado. Para Debian y Fedora
hay port, claro... bajo tu propia responsabilidad, no se ha probado en ellas.

---

## Atajos

Se instalan desde `defaults/mango/config.conf`:

| Tecla | Acción |
|-----|--------|
| <kbd>Super</kbd> + <kbd>Space</kbd> | Vista general: búsqueda de apps, navegación por etiquetas |
| <kbd>Super</kbd> + <kbd>V</kbd> | Historial del portapapeles |
| <kbd>Super</kbd> + <kbd>P</kbd> | Panel izquierdo |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>N</kbd> | Panel derecho |
| <kbd>Super</kbd> + <kbd>D</kbd> | Panel de control |
| <kbd>Super</kbd> + <kbd>,</kbd> | Ajustes |
| <kbd>Super</kbd> + <kbd>/</kbd> | Chuleta |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>W</kbd> | Cambiar familia de paneles |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>S</kbd> | Captura de una región |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>X</kbd> | OCR de una región |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>R</kbd> | Grabar una región |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>C</kbd> | Búsqueda inversa de una región |
| <kbd>Super</kbd> + <kbd>L</kbd> | Bloquear |
| <kbd>Ctrl</kbd> + <kbd>Alt</kbd> + <kbd>Delete</kbd> | Pantalla de sesión |

Los atajos de gestión de ventanas son tuyos: el shell no los define. Referencia completa:
[Atajos](../KEYBINDS.md).

---

## Fondos de pantalla

Vienen 15 fondos incluidos. Para más, mira [iNiR-Walls](https://github.com/snowarch/iNiR-Walls),
una colección que funciona bien con el pipeline de Material You.

---

## Documentación (para niri, no para mango)

| Página | De qué va |
|---|---|
| [Instalación](../INSTALL.md) | Cómo ponerlo en marcha |
| [Setup](../SETUP.md) | Actualizaciones, migraciones, reversión |
| [Atajos](../KEYBINDS.md) | Todas las combinaciones |
| [IPC](../IPC.md) | Objetivos que puedes asignar a una tecla o llamar desde un script |
| [Paquetes](../PACKAGES.md) | Cada dependencia y por qué está ahí |
| [Limitaciones](../LIMITATIONS.md) | Qué está roto y cómo sortearlo |
| [Compositores](../COMPOSITORS.md) | Cómo funciona la integración con el compositor |
| [Arquitectura](../../ARCHITECTURE.md) | Cómo está montado el código |

La mayor parte de `docs/` viene heredada de upstream y en algunos sitios todavía describe
niri. Donde la documentación y este README no coincidan sobre qué compositor está
soportado, manda este README.

---

## Solución de problemas

```bash
ilmango logs                       # logs recientes del runtime
ilmango restart                    # reiniciar el runtime activo
ilmango repair                     # doctor + reinicio + revisión filtrada de logs
ilmango doctor                     # autodiagnóstico y arreglo de problemas típicos
./install --rollback                # deshacer la última actualización
claude "ayúdame por favor"      # si no te apetece pelearte tú. venga, que se gane sus 20 $
```

Échale un ojo a [Limitaciones](../LIMITATIONS.md) para reírte un rato.

---

## Contribuir

Mira [CONTRIBUTING.md](../../CONTRIBUTING.md) — entorno de desarrollo, patrones de código
y normas para los PR.


---

## Créditos

- [**snowarch**](https://github.com/snowarch/iNiR): iNiR, el shell que aquí se porta
- [**end-4**](https://github.com/end-4/dots-hyprland): illogical-impulse, del que se forkeó iNiR
- [**Gakuseei**](https://github.com/Gakuseei): [Ricelin](https://github.com/Gakuseei/Ricelin), de donde vienen la barra pill y el look washi y flame
- [**Quickshell**](https://quickshell.outfoxxed.me/): el framework sobre el que corre esto
- [**MangoWM**](https://github.com/DreamMaoMao/mango): el compositor para el que está hecho
- **Claude** (Anthropic): escribió el port a MangoWM, como se dice arriba del todo

GPL-3.0, igual que los dotfiles de end-4. Copyright upstream (C) 2025-2026 snowarch.
