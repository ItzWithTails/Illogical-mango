<h1 align="center">Illogical-mango</h1>

<p align="center">
  <b>Quickshell 위에 만든 MangoWM용 완전한 데스크톱 셸</b>
</p>

<p align="center">
  <sub>
    <a href="../../README.md">English</a> · <a href="README.ru.md">Русский</a> · <a href="README.es.md">Español</a> · <a href="README.zh.md">中文</a> · <a href="README.ja.md">日本語</a> · <a href="README.pt.md">Português</a> · <a href="README.fr.md">Français</a> · <a href="README.de.md">Deutsch</a> · <a href="README.ko.md">한국어</a> · <a href="README.hi.md">हिन्दी</a> · <a href="README.ar.md">العربية</a> · <a href="README.it.md">Italiano</a>
  </sub>
</p>

---

## 이 포팅은 AI가 썼습니다. 전부 다. 이 README도 90%쯤은

이 프로젝트는 장난입니다. 아무도 여기에 공을 들이지 않았습니다.

MangoWM 포팅 - `services/MangoService.qml`,
`services/deferred/MangoKeybinds.qml`, `services/CompositorService.qml`의 컴포지터 감지
재작성, 설치 스크립트와 doctor의 변경 - 은 처음부터 끝까지 Claude를 통해 작성됐습니다.
"도움을 받아"가 아닙니다. 그쪽이 쓴 겁니다.

이 이야기를 맨 위에 적어 두는 이유는, 나중에 diff나 버그에서 알게 되는 일이 없도록 하기
위해서입니다. 자랑거리도 아니고 자랑거리로 내세우는 것도 아닙니다. 사실상 이 포팅은 제가
저 쓰려고, 재미로 만든 겁니다. 그 점을 감안하세요.

포팅 계층 아래에 있는 셸은 snowarch의 iNiR이고, (아마도) 사람이 쓴 것입니다.

---

## 이게 뭔가

솔직히 맨몸의 Wayland 컴포지터를 직접 올려 본 적이 있다면, 거기에 왜 셸이 필요한지는 굳이
설명할 필요가 없습니다. 그래도 어떻게 굴러가는지는 설명할 의무가 있으니 적습니다.

```
당신의 애플리케이션
   ↓
Illogical-mango   바, 독, 사이드바, 개요, 알림, 설정, 잠금 화면
   ↓
Quickshell        Wayland 셸을 위한 QML 런타임
   ↓
MangoWM           창과 렌더링
   ↓
Wayland → GPU
```

**다른 Quickshell 설정들과 다른 점:**

- **한 번 설치에 완전한 패널 패밀리가 둘.** Material ii(떠 있는 바, 사이드바, 독)와
  Waffle(하단 작업 표시줄, 시작 메뉴, 액션 센터). 같은 위젯 위에 씌운 테마가 아니라, 각자
  자기 토큰 체계를 가진 별개의 패널 트리이고 <kbd>Super</kbd>+<kbd>Shift</kbd>+<kbd>W</kbd>
  로 실행 중에 교체됩니다.
- **셸만이 아니라 시스템 전체의 테마화.** 배경화면 하나로 Material You 팔레트가 만들어져
  GTK3/4, Qt, 터미널·TUI 도구 열 개, Firefox, Discord, Spicetify, Steam, SDDM에 기록됩니다.
- **코드를 고치지 않고 설정.** 모든 것이 단일 `config.json` 위에 얹힌 GUI 설정 항목입니다.
  겉모습이나 동작을 바꾸려고 QML을 건드릴 일은 없습니다.
- **제대로 된 설치·업그레이드 경로.** `./install`이 의존성과 시스템 설정을 맡고,
  `ilmango update`가 pull하고 스키마 마이그레이션을 돌리고 당신의 변경을 보존하며 롤백도 됩니다.

**계보.** [end-4의 illogical-impulse](https://github.com/end-4/dots-hyprland)(Hyprland
닷파일) → [snowarch의 iNiR](https://github.com/snowarch/iNiR)(niri용으로 재작성) → 이것,
MangoWM으로 포팅한 것. CLI도 설정 경로도 내부도 `ilmango`라는 이름입니다. iNiR 시절 설치는
마이그레이션 037이 옮겨 주며, 옛 경로에 심볼릭 링크를 남기므로 기존 키바인드와 스크립트는
그대로 동작합니다.
업그레이드 경로가 전부 깨지기 때문에 그대로 뒀습니다.
왜 end-4를 바로 포크하지 않았느냐? 논리는 간단합니다 - 한 번 옮겨 본 프로젝트는 다시 옮기기가
더 쉽습니다.
비유하자면 Void Linux를 생각해 보세요. 거기에 systemd를 깔면 멀쩡히 돌아갑니다.
Arch Linux에서 systemd를 뜯어내면 패키지 기반을 거의 통째로 갈아야 합니다.


## 컴포지터

[MangoWM](https://github.com/DreamMaoMao/mango)을 위해 만들었고, 테스트도 그것 하나에서만
했습니다.

셸은 `$MANGO_INSTANCE_SIGNATURE`에 있는 IPC 소켓으로 mango와 이야기합니다. 이 소켓은 변화가
있을 때마다 세션 전체 스냅숏을 보내줍니다. mango는 dwm 계열이라 작업 공간 목록이 아니라
태그를 씁니다. 그래서 `MangoService`가 `(모니터, 태그 번호)` 쌍을, 바·독·개요·작업 공간
스트립이 이미 기대하고 있는 그 작업 공간 모델로 매핑하고, 그 모듈들은 손대지 않아도 그대로
동작합니다.

설정은 의도적으로 비파괴적입니다. mango는 정확히 파일 하나
(`~/.config/mango/config.conf`)만 읽고 병합은 전혀 하지 않으므로, 설치 스크립트가 당신의
컴포지터 설정을 덮어쓰는 일은 없습니다. 셸의 단축키와 자동 시작은
`~/.config/mango/ilmango.conf`에 넣고, 거기를 가리키는 `source-optional=` 한 줄만 덧붙입니다.
당신의 창 관리에는 손대지 않습니다. 자동 시작은 그 파일 안의
`exec-once=ilmango run --daemon` 한 줄이지 systemd 유닛이 아닙니다.

> [!NOTE]
> **niri와 Hyprland 코드는 아직 트리에 남아 있습니다.** `NiriService.qml`,
> `HyprlandData.qml` 그리고 `isNiri` / `isHyprland` 분기는 업스트림에서 살아남았고 지금도
> 컴파일됩니다. 물려받았을 뿐 지원 대상은 아닙니다. 여기서는 그 컴포지터들에 대해 아무것도
> 테스트하지 않고, 그들을 위해 아무것도 유지보수하지 않습니다. niri를 원한다면
> [원본 iNiR](https://github.com/snowarch/iNiR)을 쓰세요.

---

## 스크린샷

두 패널 패밀리 모두, 업스트림에서 그대로 가져온 것입니다.

<details open>
<summary><b>Material ii</b>: 떠 있는 바, 사이드바, Material Design 감성</summary>

| | |
|:---:|:---:|
| ![](https://github.com/user-attachments/assets/1fe258bc-8aec-4fd9-8574-d9d7472c3cc8) | ![](https://github.com/user-attachments/assets/3ce2055b-648c-45a1-9d09-705c1b4a03b7) |
| ![](https://github.com/user-attachments/assets/ea2311dc-769e-44dc-a46d-37cf8807d2cc) | ![](https://github.com/user-attachments/assets/ba866063-b26a-47cb-83c8-d77bd033bf8b) |
| ![](https://github.com/user-attachments/assets/88e76566-061b-4f8c-a9a8-53c157950138) | |

</details>

<details>
<summary><b>Waffle</b>: 하단 작업 표시줄, 액션 센터, Windows 11 느낌</summary>

| | |
|:---:|:---:|
| ![](https://github.com/user-attachments/assets/5c5996e7-90eb-4789-9921-0d5fe5283fa3) | ![](https://github.com/user-attachments/assets/fadf9562-751e-4138-a3a1-b87b31114d44) |

</details>

---

> [!WARNING]
> 기본 설정은 어느 정도 최신인 하드웨어를 전제로 합니다. 사양이 낮은 기기에서는 효과를 끄고,
> 안 쓰는 패널을 빼고, 시각 스타일을 납작하게 만드세요. 전부 설정 화면이나 `config.json`에서
> 할 수 있습니다.

## 기능

**패널 패밀리 두 가지**, <kbd>Super</kbd>+<kbd>Shift</kbd>+<kbd>W</kbd>로 즉시 전환:

- **Material ii** — 떠 있는 바, 사이드바, 독, 그리고 8가지 시각 스타일(Material, Cards,
  Aurora, Illogical-mango, Angel, Regalia, ZZZ, Cookie Shapes)
- **Waffle** — Windows 11 스타일 작업 표시줄, 시작 메뉴, 액션 센터, 알림 센터

**자동 테마화.** 배경화면을 고르면 시스템 전체가 따라옵니다. 셸의 Material You 색이 GTK3/4,
Qt, 터미널, Firefox, Discord, Spicetify, Steam, SDDM으로 퍼집니다. Regalia, Gruvbox,
Catppuccin, Rosé Pine 프리셋이 들어 있고, 직접 만들 수도 있습니다.

<details>
<summary><b>전체 기능 목록</b></summary>

### 테마와 외관

- **8가지 시각 스타일**: Material(불투명), Cards, Aurora(유리 블러), Illogical-mango(TUI 풍), Angel(네오브루탈리즘), Regalia(검은 공학용 섀시, 따뜻한 아이보리 잉크, 절제된 샴페인 금속), ZZZ(포스터 판), Cookie Shapes(형태가 애니메이션으로 변형)
- **배경화면에서 뽑은 동적 색상**을 Material You로 시스템 전체에 전파
- **터미널·TUI 도구 10종 자동 테마화**: foot, kitty, alacritty, ghostty, wezterm, starship, fuzzel, btop, lazygit, yazi
- **앱 테마화**: GTK3/4, Qt(plasma-integration과 darkly 경유), Firefox(MaterialFox), Discord/Vesktop(System24), Zed, Spicetify, Steam, SDDM
- **테마 프리셋**: Regalia, Regalia Ivory, Gruvbox, Catppuccin, Rosé Pine 및 사용자 정의
- **동영상 배경화면**: mp4/webm/gif, 블러 선택 가능. 성능을 위해 첫 프레임 정지도 가능
- **데스크톱 위젯**: 시계(여러 스타일), 날씨, 배경화면 레이어 위의 미디어 조작

### 바

- **6가지 바 스타일**: classic, islands, scenic, frame, Material 3 캡슐, 그리고 pill
- **pill 바**: 가운데에서 형태가 변하는 섬. 마우스를 올리면 작업 공간, 실행기, 믹서, 미디어, 달력, 화면 녹화로 펼쳐집니다
- **모듈식 배치**, 설정에 드래그 편집기가 있어 어떤 모듈이든 어디로든
- **세로 바**, 화면 가장자리를 쓰고 싶은 사람들을 위해

### 사이드바와 위젯 (Material ii)

왼쪽 사이드바(앱 서랍):
- **AI Chat**: Ollama, LM Studio, OpenRouter, Gemini, Groq, Mistral, Cerebras, Anthropic, OpenAI, OpenCode의 실시간 모델 목록
- **YT Music**: 쿠키 없는 InnerTube 플레이어. 검색, 재생 목록, 라디오, 동기화 가사
- **Wallhaven 브라우저**: 배경화면을 바로 찾아서 적용
- **애니 트래커**: AniList 연동과 방영 일정 보기
- **번역기**: Gemini 또는 translate-shell 경유
- **끌 수 있는 위젯**: 코인, 미디어 플레이어, 빠른 메모, 상태 링, 주간 달력

오른쪽 사이드바:
- **달력**(일정 연동)
- **알림 센터**
- **빠른 토글**: WiFi, 블루투스, 야간 조명, 방해 금지, 전원 프로필, WARP VPN, EasyEffects
- **볼륨 믹서**(앱별 조절)
- **블루투스와 WiFi** 기기 관리
- **뽀모도로 타이머**, **할 일 목록**, **계산기**, **메모장**
- **시스템 모니터**: CPU, RAM, 온도

### 도구

- **작업 공간 개요**: 앱 검색과 계산기를 mango의 태그 모델 위에 얹은 것
- **대시보드**: 일정, 알림, 할 일, 메모, 미디어, 날씨를 담은 구성 가능한 3열 오버레이
- **화면 가장자리 작업 공간 스트립**: 마우스를 올리면 나오는 레일. 실시간 미리보기와 드래그 재정렬
- **창 전환기**: 모든 작업 공간을 넘나드는 애니메이션 Alt-Tab, 선택적으로 켜기
- **클립보드 관리자**: 검색과 이미지 미리보기가 있는 기록
- **영역 도구**: 스크린샷, 화면 녹화, OCR, 이미지 역검색
- **치트시트**: 당신의 mango 설정에서 뽑아온 단축키 보기
- **미디어 조작**: 여러 배치 프리셋을 갖춘 완전한 MPRIS 플레이어
- **화면 표시**: 음량, 밝기, 미디어 OSD
- **곡 인식**: Shazam 방식, SongRec 경유
- **음성 입력**: 설치돼 있으면 로컬 whisper.cpp, 아니면 연결된 Groq, Gemini, OpenAI 백엔드

### 시스템

- **GUI 설정**: 파일을 건드리지 않고 전부 설정
- **GameMode**: 전체 화면 앱에서 효과를 자동으로 끔
- **자동 업데이트**: `ilmango update`. 롤백, 마이그레이션, 사용자 변경 보존 포함
- **잠금 화면**과 **세션 화면**(로그아웃/재부팅/종료/절전)
- **polkit 에이전트**, **화면 키보드**, **자동 시작 관리자**(mango 설정의 `exec-once` 줄이 바탕)
- **Kira**: 선택 사항인 픽셀 아트 마스코트. 화면 가장자리를 돌아다니며 당신의 행동에 반응합니다. 기본은 꺼짐이고, 약 32 MiB짜리 아트 팩은 `./install` › Extras에서 따로 내려받습니다
- **15개 언어**, 자동 감지
- **야간 조명**: 예약 또는 수동
- **날씨**: Open-Meteo. GPS, 직접 입력한 좌표, 도시 이름 지원
- **배터리 관리**: 임계값 설정 가능, 위험 수준에서 자동 절전
- **이벤트 소리 사용자 지정**: 마스터 볼륨과 이벤트별 오디오 파일

</details>

---

## 빠른 시작 (설치 스크립트는 앞으로 다른 것이 됩니다)

```bash
git clone https://github.com/ItzWithTails/illogical-mango.git
cd Illogical-mango
./install       # 대화식. 각 단계 전에 물어봅니다
./install -y    # 자동. 아무것도 묻지 않습니다
```

설치 스크립트가 의존성, 시스템 설정, 테마화를 맡습니다. 셸의 단축키를
`~/.config/mango/ilmango.conf`에 쓰고 기존 mango 설정에 걸어 주되, 창 관리에는 손대지 않습니다.
그다음 mango를 재시작하거나 `mmsg dispatch reload_config`를 실행하세요.

```bash
ilmango run                        # 셸 실행
ilmango settings                   # 설정 GUI 열기
ilmango logs                       # 로그 보기
ilmango doctor                     # 자동 진단과 수리
ilmango update                     # pull + 마이그레이션 + 재시작
```

다른 진입점:

```bash
./install                 # TUI 메뉴. 원하는 것을 고르기
./install --disable mango    # mango 설정은 아예 건드리지 않기
sudo make install       # 홈이 아니라 시스템 전역에 설치
./install --rollback        # 마지막 업데이트 되돌리기
```

**배포판.** Arch가 주 대상이고 가장 잘 테스트돼 있습니다. Debian과 Fedora에도 물론 포팅이
있습니다… 위험은 본인 몫이고, 그쪽에서는 테스트한 적이 없습니다.

---

## 단축키

`defaults/mango/config.conf`에서 설치됩니다:

| 키 | 동작 |
|-----|--------|
| <kbd>Super</kbd> + <kbd>Space</kbd> | 개요: 앱 검색, 태그 이동 |
| <kbd>Super</kbd> + <kbd>V</kbd> | 클립보드 기록 |
| <kbd>Super</kbd> + <kbd>P</kbd> | 왼쪽 사이드바 |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>N</kbd> | 오른쪽 사이드바 |
| <kbd>Super</kbd> + <kbd>D</kbd> | 대시보드 |
| <kbd>Super</kbd> + <kbd>,</kbd> | 설정 |
| <kbd>Super</kbd> + <kbd>/</kbd> | 치트시트 |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>W</kbd> | 패널 패밀리 전환 |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>S</kbd> | 영역 스크린샷 |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>X</kbd> | 영역 OCR |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>R</kbd> | 영역 녹화 |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>C</kbd> | 영역 이미지 역검색 |
| <kbd>Super</kbd> + <kbd>L</kbd> | 잠금 |
| <kbd>Ctrl</kbd> + <kbd>Alt</kbd> + <kbd>Delete</kbd> | 세션 화면 |

창 관리 단축키는 당신 것입니다. 셸은 그것을 정의하지 않습니다. 전체 참조:
[단축키](../KEYBINDS.md).

---

## 배경화면

배경화면 15장이 함께 들어 있습니다. 더 필요하면 [iNiR-Walls](https://github.com/snowarch/iNiR-Walls)
를 보세요. Material You 파이프라인과 잘 맞는 모음입니다.

---

## 문서 (mango가 아니라 niri용입니다)

| 페이지 | 내용 |
|---|---|
| [설치](../INSTALL.md) | 돌아가게 만들기까지 |
| [Setup](../SETUP.md) | 업데이트, 마이그레이션, 롤백 |
| [단축키](../KEYBINDS.md) | 모든 조합 |
| [IPC](../IPC.md) | 키에 걸거나 스크립트에서 호출할 수 있는 대상 |
| [패키지](../PACKAGES.md) | 모든 의존성과 그것이 있는 이유 |
| [제한 사항](../LIMITATIONS.md) | 깨진 것으로 알려진 것과 우회법 |
| [컴포지터](../COMPOSITORS.md) | 컴포지터 연동이 어떻게 되어 있는지 |
| [아키텍처](../../ARCHITECTURE.md) | 코드가 어떻게 짜여 있는지 |

`docs/` 대부분은 업스트림에서 물려받은 것이라 군데군데 아직 niri를 설명합니다. 어떤 컴포지터를
지원하는지에 대해 문서와 이 README가 어긋나면, 이 README가 맞습니다.

---

## 문제 해결

```bash
ilmango logs                       # 최근 런타임 로그
ilmango restart                    # 실행 중인 런타임 재시작
ilmango repair                     # doctor + 재시작 + 걸러낸 로그 확인
ilmango doctor                     # 흔한 문제 자동 진단과 수리
./install --rollback                # 마지막 업데이트 되돌리기
claude "도와줘"                  # 직접 파헤치기 싫을 때. 20달러 값은 해야지
```

[제한 사항](../LIMITATIONS.md)은 웃자고 한번 보세요.

---

## 기여

[CONTRIBUTING.md](../../CONTRIBUTING.md) 참고 — 개발 환경 준비, 코드 패턴, 풀 리퀘스트 규칙.


---

## 크레딧

- [**snowarch**](https://github.com/snowarch/iNiR): iNiR, 여기서 포팅한 그 셸
- [**end-4**](https://github.com/end-4/dots-hyprland): illogical-impulse, iNiR이 여기서 포크됨
- [**Gakuseei**](https://github.com/Gakuseei): [Ricelin](https://github.com/Gakuseei/Ricelin), pill 바와 washi·flame 느낌의 출처
- [**Quickshell**](https://quickshell.outfoxxed.me/): 이것이 돌아가는 프레임워크
- [**MangoWM**](https://github.com/DreamMaoMao/mango): 대상으로 삼은 컴포지터
- **Claude**(Anthropic): MangoWM 포팅을 쓴 장본인. 맨 위에 적은 그대로

GPL-3.0, end-4의 닷파일과 동일합니다. 업스트림 저작권 (C) 2025-2026 snowarch.
