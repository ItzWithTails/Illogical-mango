<h1 align="center">Illogical-mango</h1>

<p align="center">
  <b>基于 Quickshell 构建的完整 MangoWM 桌面外壳</b>
</p>

<p align="center">
  <sub>
    <a href="../../README.md">English</a> · <a href="README.ru.md">Русский</a> · <a href="README.es.md">Español</a> · <a href="README.zh.md">中文</a> · <a href="README.ja.md">日本語</a> · <a href="README.pt.md">Português</a> · <a href="README.fr.md">Français</a> · <a href="README.de.md">Deutsch</a> · <a href="README.ko.md">한국어</a> · <a href="README.hi.md">हिन्दी</a> · <a href="README.ar.md">العربية</a> · <a href="README.it.md">Italiano</a>
  </sub>
</p>

---

## 这个移植是 AI 写的。完完全全。这份 README 也有九成是

这个项目就是图一乐。没人在这上面下过功夫。

移植到 MangoWM 的部分 - `services/MangoService.qml`、
`services/deferred/MangoKeybinds.qml`、`services/CompositorService.qml` 里合成器检测的重写、
安装器和 doctor 的改动 - 从头到尾都是通过 Claude 写的。
不是"在它协助下"。是它写的。

这话放在最顶上，免得你以后才发现——从一个 diff 里，或者从一个 bug 里。这不是什么功绩，也没
打算当成功绩来讲。说白了这个移植是我给自己做的，图个乐。你自己掂量。

移植层底下的那个外壳，是 snowarch 的 iNiR，（但愿）是人写的。

---

## 这是什么

说实话，如果你自己装过一个光秃秃的 Wayland 合成器，就不用别人告诉你它为什么需要一个外壳了。
不过我还是有义务讲讲它是怎么运作的。

```
你的应用程序
   ↓
Illogical-mango   状态栏、程序坞、侧边栏、总览、通知、设置、锁屏
   ↓
Quickshell        面向 Wayland 外壳的 QML 运行时
   ↓
MangoWM           窗口与渲染
   ↓
Wayland → GPU
```

**与其他 Quickshell 配置的区别：**

- **一次安装里有两套完整的面板家族。** Material ii（悬浮状态栏、侧边栏、程序坞）和 Waffle
  （底部任务栏、开始菜单、操作中心）。它们不是套在同一批控件上的主题——而是各自独立的面板树，
  各有各的 token 体系，用 <kbd>Super</kbd>+<kbd>Shift</kbd>+<kbd>W</kbd> 在运行时切换。
- **给整个系统上色，不只是外壳。** 一张壁纸生成一套 Material You 配色，写入 GTK3/4、Qt、十个
  终端与 TUI 工具、Firefox、Discord、Spicetify、Steam 和 SDDM。
- **不改代码就能配置。** 所有东西都是图形界面里的设置项，底下只有一个 `config.json`。想改外观
  或行为，永远不需要碰 QML。
- **一条像样的安装与升级路径。** `./install` 负责依赖和系统配置；`ilmango update` 拉取更新、执行
  schema 迁移、保留你的修改，并且能回滚。

**来历。** [end-4 的 illogical-impulse](https://github.com/end-4/dots-hyprland)（Hyprland
dotfiles）→ [snowarch 的 iNiR](https://github.com/snowarch/iNiR)（为 niri 重写）→ 这个，移植到
MangoWM。CLI、配置路径和内部实现都叫 `ilmango`。iNiR 时代的安装由迁移 037 接管，它会在旧路径上留下
软链接，因此已有的快捷键和脚本仍然可用。
为什么不直接 fork end-4？逻辑很简单 - 一个已经被移植过一次的项目，再移植一次要容易得多。
打个比方，拿 Void Linux 来说。给它装上 systemd，它照样跑得好好的。
拿 Arch Linux 把 systemd 抽掉，那你几乎得把整个软件包基础全换一遍。


## 合成器

为 [MangoWM](https://github.com/DreamMaoMao/mango) 而做，也只在它上面测试过。

外壳通过 `$MANGO_INSTANCE_SIGNATURE` 上的 IPC 套接字与 mango 通信，每次变化它都会发来一份完整
的会话快照。mango 是 dwm 风格的——用的是 tag，不是工作区列表——所以 `MangoService` 把
`(显示器, tag 序号)` 这样的组合映射到状态栏、程序坞、总览和工作区边条本来就期望的那套工作区模型
上，这些模块不用改就能用。

配置刻意做成非破坏性的。mango 只读取一个文件（`~/.config/mango/config.conf`），并且不做任何合并，
所以安装器绝不会覆盖你的合成器配置。它把外壳的快捷键和自启动放进
`~/.config/mango/ilmango.conf`，再追加一行指向它的 `source-optional=`，完全不碰你的窗口管理。自启动
是那个文件里的一行 `exec-once=ilmango run --daemon`，不是 systemd 单元。

> [!NOTE]
> **niri 和 Hyprland 的代码仍留在代码树里。** `NiriService.qml`、`HyprlandData.qml` 以及
> `isNiri` / `isHyprland` 这些分支从上游沿袭下来，目前仍能编译。它们只是被继承下来的，并不受
> 支持：这里没有任何东西在那些合成器上测试过，也没有任何东西为它们维护。想要 niri 的话，去用
> [原版 iNiR](https://github.com/snowarch/iNiR)。

---

## 截图

两套面板家族，都是从上游原样搬过来的。

<details open>
<summary><b>Material ii</b>：悬浮状态栏、侧边栏、Material Design 风格</summary>

| | |
|:---:|:---:|
| ![](https://github.com/user-attachments/assets/1fe258bc-8aec-4fd9-8574-d9d7472c3cc8) | ![](https://github.com/user-attachments/assets/3ce2055b-648c-45a1-9d09-705c1b4a03b7) |
| ![](https://github.com/user-attachments/assets/ea2311dc-769e-44dc-a46d-37cf8807d2cc) | ![](https://github.com/user-attachments/assets/ba866063-b26a-47cb-83c8-d77bd033bf8b) |
| ![](https://github.com/user-attachments/assets/88e76566-061b-4f8c-a9a8-53c157950138) | |

</details>

<details>
<summary><b>Waffle</b>：底部任务栏、操作中心、Windows 11 的味道</summary>

| | |
|:---:|:---:|
| ![](https://github.com/user-attachments/assets/5c5996e7-90eb-4789-9921-0d5fe5283fa3) | ![](https://github.com/user-attachments/assets/fadf9562-751e-4138-a3a1-b87b31114d44) |

</details>

---

> [!WARNING]
> 默认配置是按还算现代的硬件来的。机器弱的话，把特效关掉、把用不上的面板去掉、把视觉风格调平——
> 这些都能在设置里或者通过 `config.json` 完成。

## 功能

**两套面板家族**，用 <kbd>Super</kbd>+<kbd>Shift</kbd>+<kbd>W</kbd> 随时切换：

- **Material ii** — 悬浮状态栏、侧边栏、程序坞，以及 8 种视觉风格（Material、Cards、Aurora、
  Illogical-mango、Angel、Regalia、ZZZ、Cookie Shapes）
- **Waffle** — Windows 11 风格的任务栏、开始菜单、操作中心、通知中心

**自动配色。** 选一张壁纸，整个系统跟着走：外壳的 Material You 配色会传播到 GTK3/4、Qt、终端、
Firefox、Discord、Spicetify、Steam 和 SDDM。自带 Regalia、Gruvbox、Catppuccin 和 Rosé Pine 预设，
也可以自己做一套。

<details>
<summary><b>完整功能列表</b></summary>

### 配色与外观

- **8 种视觉风格**：Material（实色）、Cards、Aurora（毛玻璃）、Illogical-mango（TUI 风）、Angel（新野兽派）、Regalia（黑色工程机身、暖象牙色字迹、克制的香槟色五金）、ZZZ（海报色块）、Cookie Shapes（形状动态变形）
- **从壁纸取动态配色**，经 Material You 传播到全系统
- **10 个终端与 TUI 工具自动配色**：foot、kitty、alacritty、ghostty、wezterm、starship、fuzzel、btop、lazygit、yazi
- **应用配色**：GTK3/4、Qt（经 plasma-integration 和 darkly）、Firefox（MaterialFox）、Discord/Vesktop（System24）、Zed、Spicetify、Steam、SDDM
- **主题预设**：Regalia、Regalia Ivory、Gruvbox、Catppuccin、Rosé Pine，以及自定义
- **视频壁纸**：mp4/webm/gif，可选模糊，或为了性能只定格第一帧
- **桌面小组件**：时钟（多种样式）、天气、壁纸层上的媒体控制

### 状态栏

- **6 种状态栏样式**：classic、islands、scenic、frame、Material 3 胶囊，以及 pill
- **pill 状态栏**：中央一块会变形的岛，悬停即展开为工作区、启动器、混音器、媒体、日历和录屏
- **模块化布局**，设置里有拖拽编辑器，任何模块都能放到任何位置
- **竖向状态栏**，给喜欢贴屏幕边的人

### 侧边栏与小组件（Material ii）

左侧边栏（应用抽屉）：
- **AI Chat**：Ollama、LM Studio、OpenRouter、Gemini、Groq、Mistral、Cerebras、Anthropic、OpenAI 和 OpenCode 的实时模型目录
- **YT Music**：无 cookie 的 InnerTube 播放器，带搜索、队列、电台和同步歌词
- **Wallhaven 浏览器**：直接搜索并应用壁纸
- **番剧追踪**：AniList 集成，带放送表视图
- **翻译器**：经 Gemini 或 translate-shell
- **可拖拽小组件**：加密货币、媒体播放器、速记、状态环、周历

右侧边栏：
- **日历**，带日程集成
- **通知中心**
- **快捷开关**：WiFi、蓝牙、夜灯、勿扰、电源配置、WARP VPN、EasyEffects
- **音量混音器**，支持按应用调节
- **蓝牙与 WiFi** 设备管理
- **番茄钟**、**待办清单**、**计算器**、**记事本**
- **系统监视**：CPU、内存、温度

### 工具

- **工作区总览**：应用搜索和计算器，架在 mango 的 tag 模型之上
- **仪表盘**：可配置的三栏浮层，含日程、通知、待办、笔记、媒体和天气
- **屏幕边缘工作区条**：悬停出现的导轨，带实时预览和拖拽排序
- **窗口切换器**：跨所有工作区的动画 Alt-Tab，需手动开启
- **剪贴板管理器**：带搜索和图片预览的历史记录
- **区域工具**：截图、录屏、OCR、以图搜图
- **速查表**：从你的 mango 配置里读出快捷键并展示
- **媒体控制**：完整的 MPRIS 播放器，多种布局预设
- **屏幕提示**：音量、亮度和媒体的 OSD
- **听歌识曲**：Shazam 式识别，经 SongRec
- **语音输入**：装了就用本地 whisper.cpp，或者接 Groq、Gemini、OpenAI 后端

### 系统

- **图形化设置**：什么都能配，不用动文件
- **GameMode**：全屏应用时自动关掉特效
- **自动更新**：`ilmango update`，带回滚、迁移，并保留你的修改
- **锁屏**与**会话界面**（注销/重启/关机/挂起）
- **polkit 代理**、**屏幕键盘**、**自启动管理器**，底层是 mango 配置里的 `exec-once` 那行
- **Kira**：可选的像素画吉祥物，在屏幕边缘游荡，会对你的操作有反应。默认关闭；约 32 MiB 的素材包在 `./install` › Extras 里单独下载
- **15 种语言**，自动识别
- **夜灯**：定时或手动
- **天气**：Open-Meteo，支持 GPS、手动坐标或城市名
- **电池管理**：阈值可配，电量危急时自动挂起
- **自定义事件音效**，有总音量，每个事件可单独指定音频文件

</details>

---

## 快速上手（安装器以后会换一个）

```bash
git clone https://github.com/ItzWithTails/illogical-mango.git
cd Illogical-mango
./install       # 交互式，每一步都会问
./install -y    # 全自动，不问任何问题
```

安装器负责依赖、系统配置和配色。它把外壳的快捷键写进 `~/.config/mango/ilmango.conf`，并挂到你现有的
mango 配置上，不碰你的窗口管理。之后重启 mango，或者执行 `mmsg dispatch reload_config`。

```bash
ilmango run                        # 启动外壳
ilmango settings                   # 打开设置界面
ilmango logs                       # 看日志
ilmango doctor                     # 自动诊断并修复
ilmango update                     # 拉取 + 迁移 + 重启
```

其他入口：

```bash
./install                 # TUI 菜单，想装什么自己挑
./install --disable mango    # 完全不碰 mango 配置
sudo make install       # 装到系统里，而不是你的家目录
./install --rollback        # 撤销上一次更新
```

**发行版。** Arch 是主要目标，测试得也最充分。Debian 和 Fedora 当然也有移植……后果自负，那上面
没测过。

---

## 快捷键

由 `defaults/mango/config.conf` 安装：

| 按键 | 动作 |
|-----|--------|
| <kbd>Super</kbd> + <kbd>Space</kbd> | 总览：搜索应用、按 tag 导航 |
| <kbd>Super</kbd> + <kbd>V</kbd> | 剪贴板历史 |
| <kbd>Super</kbd> + <kbd>P</kbd> | 左侧边栏 |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>N</kbd> | 右侧边栏 |
| <kbd>Super</kbd> + <kbd>D</kbd> | 仪表盘 |
| <kbd>Super</kbd> + <kbd>,</kbd> | 设置 |
| <kbd>Super</kbd> + <kbd>/</kbd> | 速查表 |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>W</kbd> | 切换面板家族 |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>S</kbd> | 区域截图 |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>X</kbd> | 区域 OCR |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>R</kbd> | 区域录制 |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>C</kbd> | 区域以图搜图 |
| <kbd>Super</kbd> + <kbd>L</kbd> | 锁屏 |
| <kbd>Ctrl</kbd> + <kbd>Alt</kbd> + <kbd>Delete</kbd> | 会话界面 |

窗口管理的快捷键归你自己管——外壳不会去定义它们。完整参考：[快捷键](../KEYBINDS.md)。

---

## 壁纸

自带 15 张壁纸。想要更多，看 [iNiR-Walls](https://github.com/snowarch/iNiR-Walls)，这套合集配
Material You 的流程效果不错。

---

## 文档（是给 niri 的，不是 mango）

| 页面 | 内容 |
|---|---|
| [安装](../INSTALL.md) | 怎么把它跑起来 |
| [Setup](../SETUP.md) | 更新、迁移、回滚 |
| [快捷键](../KEYBINDS.md) | 所有组合键 |
| [IPC](../IPC.md) | 可以绑到按键或从脚本调用的目标 |
| [软件包](../PACKAGES.md) | 每个依赖，以及它为什么在那儿 |
| [限制](../LIMITATIONS.md) | 已知坏掉的东西和绕开的办法 |
| [合成器](../COMPOSITORS.md) | 合成器集成是怎么做的 |
| [架构](../../ARCHITECTURE.md) | 代码是怎么搭起来的 |

`docs/` 里大部分内容沿袭自上游，有些地方讲的还是 niri。文档和这份 README 在"支持哪个合成器"上
对不上的地方，以这份 README 为准。

---

## 排查问题

```bash
ilmango logs                       # 最近的运行时日志
ilmango restart                    # 重启当前运行时
ilmango repair                     # doctor + 重启 + 过滤后的日志检查
ilmango doctor                     # 自动诊断并修复常见问题
./install --rollback                # 撤销上一次更新
claude "帮帮我"                 # 如果你不想自己折腾。来吧，那 20 刀总得让它挣回来
```

去 [限制](../LIMITATIONS.md) 那儿看看，乐一乐。

---

## 参与贡献

见 [CONTRIBUTING.md](../../CONTRIBUTING.md) — 开发环境搭建、代码写法，以及 PR 规范。


---

## 鸣谢

- [**snowarch**](https://github.com/snowarch/iNiR)：iNiR，这里移植的就是它
- [**end-4**](https://github.com/end-4/dots-hyprland)：illogical-impulse，iNiR 从它 fork 而来
- [**Gakuseei**](https://github.com/Gakuseei)：[Ricelin](https://github.com/Gakuseei/Ricelin)，pill 状态栏以及 washi 和 flame 的观感出自这里
- [**Quickshell**](https://quickshell.outfoxxed.me/)：这东西运行所依赖的框架
- [**MangoWM**](https://github.com/DreamMaoMao/mango)：它所面向的合成器
- **Claude**（Anthropic）：写了 MangoWM 移植，就是最顶上说的那件事

GPL-3.0，跟 end-4 的 dotfiles 一样。上游版权 (C) 2025-2026 snowarch。
