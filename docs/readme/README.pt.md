<h1 align="center">Illogical-mango</h1>

<p align="center">
  <b>Um shell de desktop completo para o MangoWM, construído sobre o Quickshell</b>
</p>

<p align="center">
  <sub>
    <a href="../../README.md">English</a> · <a href="README.ru.md">Русский</a> · <a href="README.es.md">Español</a> · <a href="README.zh.md">中文</a> · <a href="README.ja.md">日本語</a> · <a href="README.pt.md">Português</a> · <a href="README.fr.md">Français</a> · <a href="README.de.md">Deutsch</a> · <a href="README.ko.md">한국어</a> · <a href="README.hi.md">हिन्दी</a> · <a href="README.ar.md">العربية</a> · <a href="README.it.md">Italiano</a>
  </sub>
</p>

---

## Este port foi escrito por uma IA. Totalmente. Este README também, uns 90 %

O projeto é uma brincadeira. Ninguém se esforçou aqui.

O port para o MangoWM - `services/MangoService.qml`,
`services/deferred/MangoKeybinds.qml`, a reescrita da detecção de compositor em
`services/CompositorService.qml`, as mudanças no instalador e no doctor - foi escrito do
início ao fim através do Claude.
Não "com a ajuda de". Escrito por ele.

Isto está dito logo no topo para você não descobrir depois, num diff ou num bug. Não é um
mérito e não é apresentado como tal. No fundo, fiz este port para mim e por diversão. Leve
isso em conta.

O shell por baixo da camada do port é o iNiR do snowarch, escrito (espero) por um humano.

---

## O que é isto

Sinceramente, se você já instalou sozinho um compositor Wayland cru, ninguém precisa
explicar por que ele precisa de um shell. Mas sou obrigado a explicar como funciona.

```
seus aplicativos
   ↓
Illogical-mango   barra, dock, barras laterais, visão geral, notificações, ajustes, tela de bloqueio
   ↓
Quickshell        runtime QML para shells Wayland
   ↓
MangoWM           janelas e renderização
   ↓
Wayland → GPU
```

**O que o diferencia de outras configurações do Quickshell:**

- **Duas famílias de painéis completas numa única instalação.** Material ii (barra
  flutuante, barras laterais, dock) e Waffle (barra de tarefas embaixo, menu iniciar,
  central de ações). Não são temas por cima dos mesmos widgets — são árvores de painéis
  separadas, com seus próprios sistemas de tokens, trocadas em tempo real com
  <kbd>Super</kbd>+<kbd>Shift</kbd>+<kbd>W</kbd>.
- **Tematização do sistema inteiro, não só do shell.** Um papel de parede gera uma paleta
  Material You que é escrita em GTK3/4, Qt, dez terminais e ferramentas TUI, Firefox,
  Discord, Spicetify, Steam e SDDM.
- **Configurável sem mexer no código.** Tudo é um ajuste da interface por cima de um único
  `config.json`. Nunca é preciso tocar em QML para mudar a aparência ou o comportamento.
- **Um caminho de instalação e atualização de verdade.** O `./setup` cuida das dependências
  e da configuração do sistema; o `ilmango update` faz pull, roda as migrações de esquema,
  preserva suas mudanças e sabe reverter.

**Origem.** [illogical-impulse do end-4](https://github.com/end-4/dots-hyprland) (dotfiles
do Hyprland) → [iNiR do snowarch](https://github.com/snowarch/iNiR) (reescrito para o niri)
→ isto, portado para o MangoWM. A CLI, os caminhos de configuração e as entranhas ainda se
chamam `ilmango`. Instalações da época do iNiR são migradas pela migração 037, que deixa
links simbólicos nos caminhos antigos para que atalhos e scripts existentes continuem
funcionando.
Por que não forkar o end-4 direto? A lógica é simples - um projeto que já foi portado uma
vez é mais fácil de portar de novo.
Como analogia, pegue o Void Linux. Instale systemd nele e ele vai rodar numa boa.
Pegue o Arch Linux e arranque o systemd, e você terá de mudar quase toda a base de pacotes.


## Compositor

Feito para o [MangoWM](https://github.com/DreamMaoMao/mango) e testado apenas nele.

O shell fala com o mango pelo socket IPC em `$MANGO_INSTANCE_SIGNATURE`, que envia um
instantâneo completo da sessão a cada mudança. O mango é no estilo dwm — tags, não uma
lista de áreas de trabalho —, então o `MangoService` mapeia pares `(monitor, índice da
tag)` no mesmo modelo de áreas de trabalho que a barra, o dock, a visão geral e a faixa de
áreas de trabalho já esperam, e esses módulos funcionam sem alteração.

A configuração é deliberadamente não destrutiva. O mango lê exatamente um arquivo
(`~/.config/mango/config.conf`) e não mescla nada, então o instalador nunca sobrescreve a
sua configuração do compositor. Ele coloca os atalhos e a inicialização automática do shell
em `~/.config/mango/ilmango.conf` e acrescenta uma única linha `source-optional=` apontando
para lá, sem tocar na sua gestão de janelas. A inicialização automática é uma linha
`exec-once=ilmango run --daemon` nesse arquivo, não uma unidade do systemd.

> [!NOTE]
> **O código do niri e do Hyprland ainda está na árvore.** `NiriService.qml`,
> `HyprlandData.qml` e os ramos `isNiri` / `isHyprland` sobreviveram do upstream e ainda
> compilam. São herdados, não suportados: aqui nada é testado nesses compositores e nada é
> mantido para eles. Se você quer niri, pegue
> [o iNiR original](https://github.com/snowarch/iNiR).

---

## Capturas de tela

Ambas as famílias de painéis, trazidas do upstream sem alterações.

<details open>
<summary><b>Material ii</b>: barra flutuante, barras laterais, estética Material Design</summary>

| | |
|:---:|:---:|
| ![](https://github.com/user-attachments/assets/1fe258bc-8aec-4fd9-8574-d9d7472c3cc8) | ![](https://github.com/user-attachments/assets/3ce2055b-648c-45a1-9d09-705c1b4a03b7) |
| ![](https://github.com/user-attachments/assets/ea2311dc-769e-44dc-a46d-37cf8807d2cc) | ![](https://github.com/user-attachments/assets/ba866063-b26a-47cb-83c8-d77bd033bf8b) |
| ![](https://github.com/user-attachments/assets/88e76566-061b-4f8c-a9a8-53c157950138) | |

</details>

<details>
<summary><b>Waffle</b>: barra de tarefas embaixo, central de ações, vibe Windows 11</summary>

| | |
|:---:|:---:|
| ![](https://github.com/user-attachments/assets/5c5996e7-90eb-4789-9921-0d5fe5283fa3) | ![](https://github.com/user-attachments/assets/fadf9562-751e-4138-a3a1-b87b31114d44) |

</details>

---

> [!WARNING]
> A configuração padrão mira hardware razoavelmente moderno. Em máquinas fracas, desligue
> os efeitos, tire os painéis que não usa e achate o estilo visual — tudo isso se faz pelos
> ajustes ou pelo `config.json`.

## Recursos

**Duas famílias de painéis**, trocadas em tempo real com <kbd>Super</kbd>+<kbd>Shift</kbd>+<kbd>W</kbd>:

- **Material ii** — barra flutuante, barras laterais, dock e 8 estilos visuais (Material,
  Cards, Aurora, Illogical-mango, Angel, Regalia, ZZZ, Cookie Shapes)
- **Waffle** — barra de tarefas no estilo Windows 11, menu iniciar, central de ações,
  central de notificações

**Tematização automática.** Escolha um papel de parede e o sistema inteiro acompanha: as
cores Material You do shell são propagadas para GTK3/4, Qt, terminais, Firefox, Discord,
Spicetify, Steam e SDDM. Vem com as predefinições Regalia, Gruvbox, Catppuccin e Rosé Pine,
ou monte a sua.

<details>
<summary><b>Lista completa de recursos</b></summary>

### Tematização e aparência

- **8 estilos visuais**: Material (sólido), Cards, Aurora (desfoque de vidro), Illogical-mango (inspirado em TUI), Angel (neobrutalismo), Regalia (chassi preto de engenharia, tinta marfim quente, ferragens champanhe contidas), ZZZ (placas de cartaz), Cookie Shapes (morfose animada de formas)
- **Cores dinâmicas do papel de parede** via Material You, propagadas por todo o sistema
- **10 terminais e ferramentas TUI tematizados automaticamente**: foot, kitty, alacritty, ghostty, wezterm, starship, fuzzel, btop, lazygit, yazi
- **Tematização de apps**: GTK3/4, Qt (via plasma-integration e darkly), Firefox (MaterialFox), Discord/Vesktop (System24), Zed, Spicetify, Steam, SDDM
- **Predefinições de tema**: Regalia, Regalia Ivory, Gruvbox, Catppuccin, Rosé Pine e personalizadas
- **Papéis de parede em vídeo**: mp4/webm/gif com desfoque opcional, ou primeiro quadro congelado por desempenho
- **Widgets na área de trabalho**: relógio (vários estilos), previsão do tempo, controles de mídia na camada do papel de parede

### Barra

- **6 estilos de barra**: classic, islands, scenic, frame, cápsulas Material 3 e pill
- **Barra pill**: uma ilha central que se transforma e abre ao passar o mouse em áreas de trabalho, lançador, mixer, mídia, calendário e gravador de tela
- **Layout modular** com editor de arrastar nos ajustes: qualquer módulo vai para onde você quiser
- **Barra vertical** para layouts na borda da tela

### Barras laterais e widgets (Material ii)

Barra esquerda (gaveta de aplicativos):
- **AI Chat**: catálogos de modelos ao vivo em Ollama, LM Studio, OpenRouter, Gemini, Groq, Mistral, Cerebras, Anthropic, OpenAI e OpenCode
- **YT Music**: player InnerTube sem cookies com busca, fila, rádio e letras sincronizadas
- **Navegador do Wallhaven**: busque e aplique papéis de parede direto
- **Rastreador de animes**: integração com AniList e visão de programação
- **Tradutor**: via Gemini ou translate-shell
- **Widgets arrastáveis**: cripto, player de mídia, notas rápidas, anéis de status, calendário semanal

Barra direita:
- **Calendário** com integração de eventos
- **Central de notificações**
- **Alternadores rápidos**: WiFi, Bluetooth, luz noturna, não perturbe, perfis de energia, WARP VPN, EasyEffects
- **Mixer de volume** com controle por aplicativo
- **Gerenciamento de dispositivos** Bluetooth e WiFi
- **Timer Pomodoro**, **lista de tarefas**, **calculadora**, **bloco de notas**
- **Monitor do sistema**: CPU, RAM, temperatura

### Ferramentas

- **Visão geral das áreas de trabalho**: busca de aplicativos e calculadora, apoiadas no modelo de tags do mango
- **Painel**: sobreposição configurável em três colunas com agenda, notificações, tarefas, notas, mídia e clima
- **Faixa de áreas de trabalho na borda**: trilho ao passar o mouse com prévias ao vivo e reordenação por arrasto
- **Alternador de janelas**: Alt-Tab animado por todas as áreas de trabalho, opcional
- **Gerenciador da área de transferência**: histórico com busca e prévia de imagens
- **Ferramentas de região**: capturas, gravação de tela, OCR, busca reversa de imagens
- **Cola**: visualizador de atalhos extraído da sua configuração do mango
- **Controles de mídia**: player MPRIS completo com várias predefinições de layout
- **Indicação na tela**: OSD de volume, brilho e mídia
- **Reconhecimento de músicas**: estilo Shazam, via SongRec
- **Entrada por voz**: whisper.cpp local, se instalado, ou um backend conectado do Groq, Gemini ou OpenAI

### Sistema

- **Ajustes em GUI**: configura tudo sem mexer em arquivos
- **GameMode**: desliga os efeitos automaticamente em apps em tela cheia
- **Atualizações automáticas**: `ilmango update` com reversão, migrações e preservação das suas mudanças
- **Tela de bloqueio** e **tela de sessão** (sair/reiniciar/desligar/suspender)
- **Agente polkit**, **teclado na tela**, **gerenciador de inicialização automática** apoiado na linha `exec-once` da configuração do mango
- **Kira**: mascote opcional em pixel art que perambula pelas bordas da tela e reage ao que você faz. Desligada por padrão; o pacote de arte de ~32 MiB é baixado à parte em `./setup` › Extras
- **15 idiomas** com detecção automática
- **Luz noturna**: agendada ou manual
- **Clima**: Open-Meteo, aceita GPS, coordenadas manuais ou nome da cidade
- **Gerenciamento de bateria**: limiares configuráveis, suspensão automática em nível crítico
- **Sons de eventos personalizados** com volume geral e um arquivo de áudio por evento

</details>

---

## Início rápido (o instalador será outro no futuro)

```bash
git clone https://github.com/ItzWithTails/illogical-mango.git
cd Illogical-mango
./setup install       # interativo, pergunta antes de cada passo
./setup install -y    # automático, sem perguntas
```

O instalador cuida das dependências, da configuração do sistema e da tematização. Ele
escreve os atalhos do shell em `~/.config/mango/ilmango.conf` e os engata na sua configuração
existente do mango sem tocar na gestão de janelas. Reinicie o mango ou rode
`mmsg dispatch reload_config`.

```bash
ilmango run                        # iniciar o shell
ilmango settings                   # abrir a GUI de ajustes
ilmango logs                       # ver os logs
ilmango doctor                     # autodiagnóstico e conserto
ilmango update                     # pull + migrações + reinício
```

Outros pontos de entrada:

```bash
./setup                 # menu TUI, você escolhe o que quer
./setup install --skip-mango    # não tocar em nada na configuração do mango
sudo make install       # instalação de sistema em vez da sua home
./setup rollback        # desfazer a última atualização
```

**Distribuições.** O Arch é o alvo principal e o mais testado. Para Debian e Fedora existe
port, claro... por sua conta e risco, não houve teste algum neles.

---

## Atalhos

Instalados a partir de `defaults/mango/config.conf`:

| Tecla | Ação |
|-----|--------|
| <kbd>Super</kbd> + <kbd>Space</kbd> | Visão geral: busca de aplicativos, navegação por tags |
| <kbd>Super</kbd> + <kbd>V</kbd> | Histórico da área de transferência |
| <kbd>Super</kbd> + <kbd>P</kbd> | Barra lateral esquerda |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>N</kbd> | Barra lateral direita |
| <kbd>Super</kbd> + <kbd>D</kbd> | Painel |
| <kbd>Super</kbd> + <kbd>,</kbd> | Ajustes |
| <kbd>Super</kbd> + <kbd>/</kbd> | Cola |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>W</kbd> | Trocar de família de painéis |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>S</kbd> | Captura de uma região |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>X</kbd> | OCR de uma região |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>R</kbd> | Gravar uma região |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>C</kbd> | Busca reversa de uma região |
| <kbd>Super</kbd> + <kbd>L</kbd> | Bloquear |
| <kbd>Ctrl</kbd> + <kbd>Alt</kbd> + <kbd>Delete</kbd> | Tela de sessão |

Os atalhos de gestão de janelas são seus — o shell não os define. Referência completa:
[Atalhos](../KEYBINDS.md).

---

## Papéis de parede

Vêm 15 papéis de parede junto. Para mais, veja [iNiR-Walls](https://github.com/snowarch/iNiR-Walls),
uma coleção que funciona bem com o pipeline do Material You.

---

## Documentação (para niri, não para mango)

| Página | Sobre o que é |
|---|---|
| [Instalação](../INSTALL.md) | Como colocar para rodar |
| [Setup](../SETUP.md) | Atualizações, migrações, reversão |
| [Atalhos](../KEYBINDS.md) | Todas as combinações |
| [IPC](../IPC.md) | Alvos que você pode ligar a uma tecla ou chamar de um script |
| [Pacotes](../PACKAGES.md) | Cada dependência e por que ela está lá |
| [Limitações](../LIMITATIONS.md) | O que se sabe quebrado, e contornos |
| [Compositores](../COMPOSITORS.md) | Como funciona a integração com o compositor |
| [Arquitetura](../../ARCHITECTURE.md) | Como o código está montado |

A maior parte de `docs/` foi herdada do upstream e em alguns pontos ainda descreve o niri.
Onde a documentação e este README discordarem sobre qual compositor é suportado, quem está
certo é este README.

---

## Solução de problemas

```bash
ilmango logs                       # logs recentes do runtime
ilmango restart                    # reiniciar o runtime ativo
ilmango repair                     # doctor + reinício + checagem filtrada dos logs
ilmango doctor                     # autodiagnóstico e conserto de problemas comuns
./setup rollback                # desfazer a última atualização
claude "me ajuda por favor"     # se você não quiser se virar sozinho. vai, ele tem que trabalhar pelos 20 US$ dele
```

Dê uma olhada nas [Limitações](../LIMITATIONS.md) para rir um pouco.

---

## Contribuindo

Veja [CONTRIBUTING.md](../../CONTRIBUTING.md) — ambiente de desenvolvimento, padrões de
código e regras para pull requests.


---

## Créditos

- [**snowarch**](https://github.com/snowarch/iNiR): iNiR, o shell que é portado aqui
- [**end-4**](https://github.com/end-4/dots-hyprland): illogical-impulse, de onde o iNiR foi forkado
- [**Gakuseei**](https://github.com/Gakuseei): [Ricelin](https://github.com/Gakuseei/Ricelin), de onde vêm a barra pill e o visual washi e flame
- [**Quickshell**](https://quickshell.outfoxxed.me/): o framework em que isto roda
- [**MangoWM**](https://github.com/DreamMaoMao/mango): o compositor para o qual isto foi feito
- **Claude** (Anthropic): escreveu o port para o MangoWM, como dito lá em cima

GPL-3.0, igual aos dotfiles do end-4. Copyright upstream (C) 2025-2026 snowarch.
