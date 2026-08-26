<h1 align="center">Illogical-mango</h1>

<p align="center">
  <b>Quickshell の上に構築された、MangoWM 向けの完全なデスクトップシェル</b>
</p>

<p align="center">
  <sub>
    <a href="../../README.md">English</a> · <a href="README.ru.md">Русский</a> · <a href="README.es.md">Español</a> · <a href="README.zh.md">中文</a> · <a href="README.ja.md">日本語</a> · <a href="README.pt.md">Português</a> · <a href="README.fr.md">Français</a> · <a href="README.de.md">Deutsch</a> · <a href="README.ko.md">한국어</a> · <a href="README.hi.md">हिन्दी</a> · <a href="README.ar.md">العربية</a> · <a href="README.it.md">Italiano</a>
  </sub>
</p>

---

## この移植は AI が書きました。丸ごと全部。この README も 9 割方そう

このプロジェクトはネタです。誰も本気で作っていません。

MangoWM への移植 - `services/MangoService.qml`、
`services/deferred/MangoKeybinds.qml`、`services/CompositorService.qml` におけるコンポジタ
検出の作り直し、インストーラと doctor への変更 - は、最初から最後まで Claude を通して
書かれました。
「手伝ってもらった」ではありません。あれが書いたのです。

これを一番上に書いてあるのは、後になって diff やバグから気づく、ということがないように
するためです。手柄でもないし、手柄として出しているわけでもありません。要するにこの移植は
自分のために、面白半分で作ったものです。そのつもりで見てください。

移植レイヤーの下にあるシェルは snowarch の iNiR で、（たぶん）人間が書いたものです。

---

## これは何か

正直なところ、素の Wayland コンポジタを自分で立てたことがある人になら、なぜシェルが要るのかを
説明する必要はありません。とはいえ、仕組みの説明はしておく義務があります。

```
あなたのアプリケーション
   ↓
Illogical-mango   バー、ドック、サイドバー、概要、通知、設定、ロック画面
   ↓
Quickshell        Wayland シェル向けの QML ランタイム
   ↓
MangoWM           ウィンドウと描画
   ↓
Wayland → GPU
```

**ほかの Quickshell 設定との違い:**

- **1 回のインストールに完全なパネルファミリーが 2 つ。** Material ii（フローティングバー、
  サイドバー、ドック）と Waffle（下部タスクバー、スタートメニュー、アクションセンター）。
  同じウィジェットに被せたテーマではなく、それぞれ独自のトークン体系を持つ別々のパネルツリー
  で、<kbd>Super</kbd>+<kbd>Shift</kbd>+<kbd>W</kbd> で実行中に切り替わります。
- **シェルだけでなくシステム全体のテーマ化。** 壁紙 1 枚から Material You のパレットが作られ、
  GTK3/4、Qt、10 個のターミナルおよび TUI ツール、Firefox、Discord、Spicetify、Steam、SDDM に
  書き出されます。
- **コードを編集せずに設定できる。** すべては単一の `config.json` の上に載った GUI の設定項目
  です。見た目や挙動を変えるのに QML を触る必要は一度もありません。
- **まともなインストールとアップグレードの導線。** `./setup` が依存関係とシステム設定を面倒
  みて、`ilmango update` が pull し、スキーマのマイグレーションを走らせ、あなたの変更を保ったまま、
  ロールバックもできます。

**系譜。** [end-4 の illogical-impulse](https://github.com/end-4/dots-hyprland)（Hyprland の
dotfiles）→ [snowarch の iNiR](https://github.com/snowarch/iNiR)（niri 向けに書き直し）→ これ、
MangoWM に移植したもの。CLI も設定パスも内部も `ilmango` という名前です。iNiR 時代のインストールは
マイグレーション 037 が引き継ぎ、旧パスにシンボリックリンクを残すので、既存のキーバインドや
スクリプトはそのまま動きます。
アップグレードの経路が軒並み壊れるので、名前はそのままにしました。
なぜ end-4 を直接フォークしなかったのか？理屈は単純です - 一度移植された経験のあるプロジェクト
は、もう一度移植するのが楽なのです。
たとえるなら Void Linux です。あれに systemd を入れれば、問題なく動きます。
Arch Linux から systemd をもぎ取ったら、パッケージ基盤をほぼ丸ごと入れ替えるはめになります。


## コンポジタ

[MangoWM](https://github.com/DreamMaoMao/mango) 向けに作られ、テストもそれだけです。

シェルは `$MANGO_INSTANCE_SIGNATURE` にある IPC ソケット経由で mango と話します。ここには変化の
たびにセッションの完全なスナップショットが流れてきます。mango は dwm 系で、ワークスペースの一覧
ではなくタグを使います。そこで `MangoService` が `(モニタ, タグ番号)` の組を、バー・ドック・概要・
ワークスペースストリップがもともと想定しているワークスペースモデルへ写像するので、これらのモジュール
は手を入れずにそのまま動きます。

設定はあえて非破壊的にしてあります。mango はファイルをちょうど 1 つ
（`~/.config/mango/config.conf`）だけ読み、マージは一切しません。ですからインストーラがあなたの
コンポジタ設定を上書きすることはありません。シェルのキーバインドと自動起動は
`~/.config/mango/ilmango.conf` に置き、そこを指す `source-optional=` を 1 行追記するだけで、
ウィンドウ管理には触れません。自動起動はそのファイル内の `exec-once=ilmango run --daemon` の行で
あって、systemd ユニットではありません。

> [!NOTE]
> **niri と Hyprland のコードはまだツリーに残っています。** `NiriService.qml`、
> `HyprlandData.qml`、`isNiri` / `isHyprland` の分岐は上流から残ったもので、今もコンパイルは
> 通ります。ただし引き継いだだけで、サポート対象ではありません。ここでは、それらのコンポジタで
> テストされているものは何もなく、それら向けに保守されているものもありません。niri が使いたい
> なら [本家の iNiR](https://github.com/snowarch/iNiR) を使ってください。

---

## スクリーンショット

パネルファミリーは両方とも、上流からそのまま持ってきたものです。

<details open>
<summary><b>Material ii</b>: フローティングバー、サイドバー、Material Design 的な佇まい</summary>

| | |
|:---:|:---:|
| ![](https://github.com/user-attachments/assets/1fe258bc-8aec-4fd9-8574-d9d7472c3cc8) | ![](https://github.com/user-attachments/assets/3ce2055b-648c-45a1-9d09-705c1b4a03b7) |
| ![](https://github.com/user-attachments/assets/ea2311dc-769e-44dc-a46d-37cf8807d2cc) | ![](https://github.com/user-attachments/assets/ba866063-b26a-47cb-83c8-d77bd033bf8b) |
| ![](https://github.com/user-attachments/assets/88e76566-061b-4f8c-a9a8-53c157950138) | |

</details>

<details>
<summary><b>Waffle</b>: 下部タスクバー、アクションセンター、Windows 11 な雰囲気</summary>

| | |
|:---:|:---:|
| ![](https://github.com/user-attachments/assets/5c5996e7-90eb-4789-9921-0d5fe5283fa3) | ![](https://github.com/user-attachments/assets/fadf9562-751e-4138-a3a1-b87b31114d44) |

</details>

---

> [!WARNING]
> 既定の設定はそこそこ新しいハードウェアを前提にしています。非力なマシンでは、エフェクトを切り、
> 使わないパネルを外し、見た目のスタイルを平坦にしてください。どれも設定画面か `config.json`
> からできます。

## 機能

**パネルファミリーが 2 つ**、<kbd>Super</kbd>+<kbd>Shift</kbd>+<kbd>W</kbd> で随時切り替え:

- **Material ii** — フローティングバー、サイドバー、ドック、そして 8 種の視覚スタイル
  （Material、Cards、Aurora、Illogical-mango、Angel、Regalia、ZZZ、Cookie Shapes）
- **Waffle** — Windows 11 風のタスクバー、スタートメニュー、アクションセンター、通知センター

**自動テーマ化。** 壁紙を選べばシステム全体がそれに追従します。シェルの Material You カラーが
GTK3/4、Qt、ターミナル、Firefox、Discord、Spicetify、Steam、SDDM へ伝わります。Regalia、Gruvbox、
Catppuccin、Rosé Pine のプリセット同梱、自作もできます。

<details>
<summary><b>機能の一覧</b></summary>

### テーマと外観

- **8 種の視覚スタイル**: Material（ベタ塗り）、Cards、Aurora（すりガラス）、Illogical-mango（TUI 風）、Angel（ネオブルータリズム）、Regalia（黒い工学的シャーシ、温かみのあるアイボリーの文字、抑えたシャンパン色の金具）、ZZZ（ポスター調の板）、Cookie Shapes（形状がアニメーションで変形）
- **壁紙からの動的な配色**を Material You 経由でシステム全体へ
- **10 個のターミナル・TUI ツールを自動テーマ化**: foot、kitty、alacritty、ghostty、wezterm、starship、fuzzel、btop、lazygit、yazi
- **アプリのテーマ化**: GTK3/4、Qt（plasma-integration と darkly 経由）、Firefox（MaterialFox）、Discord/Vesktop（System24）、Zed、Spicetify、Steam、SDDM
- **テーマプリセット**: Regalia、Regalia Ivory、Gruvbox、Catppuccin、Rosé Pine、および自作
- **動画壁紙**: mp4/webm/gif、任意でぼかし。性能重視なら最初のフレームで静止も可
- **デスクトップウィジェット**: 時計（複数スタイル）、天気、壁紙レイヤー上のメディア操作

### バー

- **6 種のバースタイル**: classic、islands、scenic、frame、Material 3 のカプセル、そして pill
- **pill バー**: 中央で形を変える島。ホバーするとワークスペース、ランチャー、ミキサー、メディア、カレンダー、画面録画へ展開
- **モジュール式レイアウト**、設定内のドラッグエディタでどのモジュールもどこへでも
- **縦置きバー**、画面端に寄せたい人向け

### サイドバーとウィジェット（Material ii）

左サイドバー（アプリドロワー）:
- **AI Chat**: Ollama、LM Studio、OpenRouter、Gemini、Groq、Mistral、Cerebras、Anthropic、OpenAI、OpenCode のモデル一覧をライブ取得
- **YT Music**: cookie 不要の InnerTube プレイヤー。検索、キュー、ラジオ、同期歌詞つき
- **Wallhaven ブラウザ**: 壁紙をその場で検索して適用
- **アニメトラッカー**: AniList 連携、放送スケジュール表示つき
- **翻訳**: Gemini または translate-shell 経由
- **ドラッグできるウィジェット**: 暗号資産、メディアプレイヤー、クイックメモ、ステータスリング、週間カレンダー

右サイドバー:
- **カレンダー**（予定連携つき）
- **通知センター**
- **クイックトグル**: WiFi、Bluetooth、ナイトライト、通知オフ、電源プロファイル、WARP VPN、EasyEffects
- **音量ミキサー**（アプリごとに調整）
- **Bluetooth と WiFi** のデバイス管理
- **ポモドーロタイマー**、**ToDo リスト**、**電卓**、**メモ帳**
- **システムモニタ**: CPU、メモリ、温度

### ツール

- **ワークスペース概要**: アプリ検索と電卓を、mango のタグモデルの上に載せたもの
- **ダッシュボード**: 予定、通知、ToDo、メモ、メディア、天気を並べた 3 カラムのオーバーレイ（構成可変）
- **画面端のワークスペースストリップ**: ホバーで出るレール。ライブプレビューとドラッグ並べ替えつき
- **ウィンドウスイッチャー**: 全ワークスペースを横断するアニメーション付き Alt-Tab、任意で有効化
- **クリップボードマネージャ**: 検索と画像プレビューつきの履歴
- **範囲ツール**: スクリーンショット、画面録画、OCR、画像の逆検索
- **チートシート**: あなたの mango 設定から読み取ったキーバインド一覧
- **メディア操作**: 複数のレイアウトプリセットを備えた本格的な MPRIS プレイヤー
- **オンスクリーン表示**: 音量、明るさ、メディアの OSD
- **曲の識別**: Shazam 的な認識を SongRec 経由で
- **音声入力**: インストール済みならローカルの whisper.cpp、または接続済みの Groq、Gemini、OpenAI バックエンド

### システム

- **GUI 設定**: ファイルを触らずに何でも設定できる
- **GameMode**: 全画面アプリのときエフェクトを自動で無効化
- **自動更新**: `ilmango update`。ロールバック、マイグレーション、ユーザ変更の保持つき
- **ロック画面**と**セッション画面**（ログアウト／再起動／シャットダウン／サスペンド）
- **polkit エージェント**、**スクリーンキーボード**、**自動起動マネージャ**（mango 設定の `exec-once` の行が土台）
- **Kira**: 任意で有効にするドット絵のマスコット。画面の端をうろつき、あなたの操作に反応します。既定では無効。約 32 MiB の素材パックは `./setup` › Extras から別途ダウンロード
- **15 言語**、自動判別つき
- **ナイトライト**: スケジュールまたは手動
- **天気**: Open-Meteo。GPS、手入力の座標、都市名に対応
- **バッテリー管理**: しきい値を設定でき、危険域で自動サスペンド
- **イベント音の差し替え**: マスター音量つき、イベントごとに音声ファイルを指定

</details>

---

## クイックスタート（インストーラは将来別のものになります）

```bash
git clone https://github.com/ItzWithTails/illogical-mango.git
cd Illogical-mango
./setup install       # 対話式。各ステップの前に確認します
./setup install -y    # 全自動。何も訊きません
```

インストーラが依存関係、システム設定、テーマ化を引き受けます。シェルのキーバインドを
`~/.config/mango/ilmango.conf` に書き、既存の mango 設定へ繋ぎ込みますが、ウィンドウ管理には
触れません。あとは mango を再起動するか、`mmsg dispatch reload_config` を実行してください。

```bash
ilmango run                        # シェルを起動
ilmango settings                   # 設定 GUI を開く
ilmango logs                       # ログを見る
ilmango doctor                     # 自動診断と修復
ilmango update                     # pull + マイグレーション + 再起動
```

ほかの入り口:

```bash
./setup                 # TUI メニュー。必要なものを選ぶ
./setup install --skip-mango    # mango の設定には一切触らない
sudo make install       # ホームではなくシステム全体へ
./setup rollback        # 直前の更新を取り消す
```

**ディストリビューション。** Arch が主対象で、いちばんテストされています。Debian と Fedora にも
もちろん移植はあります……自己責任で。あちらでのテストはしていません。

---

## キーバインド

`defaults/mango/config.conf` からインストールされます:

| キー | 動作 |
|-----|--------|
| <kbd>Super</kbd> + <kbd>Space</kbd> | 概要: アプリ検索、タグ間の移動 |
| <kbd>Super</kbd> + <kbd>V</kbd> | クリップボード履歴 |
| <kbd>Super</kbd> + <kbd>P</kbd> | 左サイドバー |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>N</kbd> | 右サイドバー |
| <kbd>Super</kbd> + <kbd>D</kbd> | ダッシュボード |
| <kbd>Super</kbd> + <kbd>,</kbd> | 設定 |
| <kbd>Super</kbd> + <kbd>/</kbd> | チートシート |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>W</kbd> | パネルファミリーを切り替え |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>S</kbd> | 範囲をスクリーンショット |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>X</kbd> | 範囲を OCR |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>R</kbd> | 範囲を録画 |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>C</kbd> | 範囲を逆画像検索 |
| <kbd>Super</kbd> + <kbd>L</kbd> | ロック |
| <kbd>Ctrl</kbd> + <kbd>Alt</kbd> + <kbd>Delete</kbd> | セッション画面 |

ウィンドウ管理のキーはあなたのものです。シェルは定義しません。完全な一覧:
[キーバインド](../KEYBINDS.md)。

---

## 壁紙

壁紙が 15 枚同梱されています。もっと欲しければ [iNiR-Walls](https://github.com/snowarch/iNiR-Walls)
を見てください。Material You のパイプラインと相性のよいコレクションです。

---

## ドキュメント（niri 向けで、mango 向けではありません）

| ページ | 内容 |
|---|---|
| [インストール](../INSTALL.md) | 動かすまで |
| [Setup](../SETUP.md) | 更新、マイグレーション、ロールバック |
| [キーバインド](../KEYBINDS.md) | すべてのショートカット |
| [IPC](../IPC.md) | キーに割り当てたりスクリプトから呼べるターゲット |
| [パッケージ](../PACKAGES.md) | 依存関係と、それが必要な理由 |
| [制限事項](../LIMITATIONS.md) | 壊れているとわかっているものと回避策 |
| [コンポジタ](../COMPOSITORS.md) | コンポジタ連携の仕組み |
| [アーキテクチャ](../../ARCHITECTURE.md) | コードの組み立て方 |

`docs/` の大部分は上流から引き継いだもので、ところどころ今も niri の説明になっています。
ドキュメントとこの README で「どのコンポジタに対応しているか」が食い違う場合は、この README が
正しいです。

---

## トラブルシューティング

```bash
ilmango logs                       # 直近のランタイムログ
ilmango restart                    # 動作中のランタイムを再起動
ilmango repair                     # doctor + 再起動 + ログの絞り込み確認
ilmango doctor                     # よくある問題の自動診断と修復
./setup rollback                # 直前の更新を取り消す
claude "助けてください"          # 自分で調べる気がないとき。まあ、20 ドル分は働いてもらわないと
```

[制限事項](../LIMITATIONS.md) は笑うために覗いてみてください。

---

## コントリビュート

[CONTRIBUTING.md](../../CONTRIBUTING.md) を参照 — 開発環境の準備、コードの書き方、プルリクエスト
の規約。


---

## クレジット

- [**snowarch**](https://github.com/snowarch/iNiR): iNiR。ここで移植されているシェル
- [**end-4**](https://github.com/end-4/dots-hyprland): illogical-impulse。iNiR のフォーク元
- [**Gakuseei**](https://github.com/Gakuseei): [Ricelin](https://github.com/Gakuseei/Ricelin)。pill バーと washi・flame の見た目の出どころ
- [**Quickshell**](https://quickshell.outfoxxed.me/): これが動いているフレームワーク
- [**MangoWM**](https://github.com/DreamMaoMao/mango): 対象としているコンポジタ
- **Claude**（Anthropic）: MangoWM 移植を書いた張本人。一番上に書いたとおり

GPL-3.0、end-4 の dotfiles と同じです。上流の著作権 (C) 2025-2026 snowarch。
