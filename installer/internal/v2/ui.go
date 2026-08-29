package v2

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"ilmango/internal/pkg"
	"ilmango/internal/run"
	"ilmango/internal/system"
)

type uiStage int

const (
	stageWelcome uiStage = iota
	stageConfigure
	stageReview
	stageProgress
	stageDone
)

type UI struct {
	Config Config
	Plan   *Plan
	Result Result

	stage      uiStage
	cursor     int
	offset     int
	width      int
	height     int
	editing    bool
	buildErr   error
	aborted    bool
	cancelling bool

	cancel context.CancelFunc
	events []Event
	stream <-chan tea.Msg
	runner *run.Runner

	accent lipgloss.Style
	muted  lipgloss.Style
	warn   lipgloss.Style
	bad    lipgloss.Style
	good   lipgloss.Style
}

type runEventMsg struct{ Event Event }
type runResultMsg struct{ Result Result }
type authResultMsg struct{ Err error }

func NewUI(cfg Config, runner *run.Runner) *UI {
	plain := cfg.NoColor || os.Getenv("NO_COLOR") != ""
	style := func(color string) lipgloss.Style {
		if plain {
			return lipgloss.NewStyle()
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	}
	return &UI{Config: cfg, runner: runner, width: 80, height: 24,
		accent: style("212").Bold(true), muted: style("245"), warn: style("214"),
		bad: style("196"), good: style("82")}
}

func (u *UI) Init() tea.Cmd { return nil }
func (u *UI) Aborted() bool { return u.aborted }

func (u *UI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		u.width, u.height = msg.Width, msg.Height
	case runEventMsg:
		u.events = append(u.events, msg.Event)
		if len(u.events) > 500 {
			u.events = append([]Event{}, u.events[len(u.events)-500:]...)
		}
		return u, waitStream(u.stream)
	case runResultMsg:
		u.Result = msg.Result
		u.stage = stageDone
		u.stream = nil
		if u.aborted && errors.Is(msg.Result.Err, context.Canceled) {
			return u, tea.Quit
		}
		return u, nil
	case authResultMsg:
		if msg.Err != nil {
			u.buildErr = fmt.Errorf("authentication failed: %w", msg.Err)
			return u, nil
		}
		return u, u.startRun()
	case tea.KeyPressMsg:
		key := msg.String()
		if key == "ctrl+c" {
			if u.stage == stageProgress && u.cancel != nil {
				u.cancel()
				u.cancelling = true
				u.aborted = true
				return u, nil
			}
			u.aborted = true
			return u, tea.Quit
		}
		if u.editing {
			return u.updateEditor(key)
		}
		return u.updateKey(key)
	}
	return u, nil
}

func (u *UI) updateEditor(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "enter", "esc":
		u.editing = false
		if strings.TrimSpace(u.Config.KeyboardLayout) == "" {
			u.Config.KeyboardLayout = "system"
		}
	case "backspace":
		if u.Config.KeyboardLayout != "" {
			_, size := utf8.DecodeLastRuneInString(u.Config.KeyboardLayout)
			u.Config.KeyboardLayout = u.Config.KeyboardLayout[:len(u.Config.KeyboardLayout)-size]
		}
	default:
		if len(key) == 1 || key == "," || key == "-" || key == "_" {
			u.Config.KeyboardLayout += key
		}
	}
	return u, nil
}

func (u *UI) updateKey(key string) (tea.Model, tea.Cmd) {
	switch u.stage {
	case stageWelcome:
		switch key {
		case "q", "esc":
			u.aborted = true
			return u, tea.Quit
		case "enter":
			if u.Config.Operation == Install || u.Config.Operation == Update {
				u.stage = stageConfigure
			} else {
				u.preparePlan()
			}
		}
	case stageConfigure:
		switch key {
		case "q":
			u.aborted = true
			return u, tea.Quit
		case "esc":
			u.stage = stageWelcome
		case "up", "k":
			u.cursor = (u.cursor + 5) % 6
		case "down", "j":
			u.cursor = (u.cursor + 1) % 6
		case "left", "h":
			u.changeConfig(-1)
		case "right", "l", "space":
			u.changeConfig(1)
		case "e":
			if u.cursor == 4 {
				u.Config.KeyboardLayout = ""
				u.editing = true
			}
		case "enter":
			u.preparePlan()
		}
	case stageReview:
		switch key {
		case "q":
			u.aborted = true
			return u, tea.Quit
		case "esc":
			if u.Config.Operation == Install || u.Config.Operation == Update {
				u.stage = stageConfigure
			} else {
				u.stage = stageWelcome
			}
		case "up", "k", "pgup":
			if u.offset > 0 {
				u.offset--
			}
		case "down", "j", "pgdown":
			u.offset++
		case "enter":
			if u.Plan != nil && u.buildErr == nil {
				if u.Config.Operation == Status {
					u.Result = (Engine{}).Run(context.Background(), u.Plan)
					u.stage = stageDone
					return u, nil
				}
				return u, u.authorize()
			}
		}
	case stageProgress:
		// Deliberately ignore q/esc while mutating. Ctrl+C cancels the context,
		// waits for rollback, and is the only interrupt path.
	case stageDone:
		if key == "enter" || key == "q" || key == "esc" {
			return u, tea.Quit
		}
	}
	return u, nil
}

func (u *UI) changeConfig(delta int) {
	switch u.cursor {
	case 0:
		values := []Preset{Minimal, Recommended, Full}
		u.Config.Preset = values[cycleIndex(values, u.Config.Preset, delta)]
		if u.Config.Preset == Minimal {
			u.Config.MangoHook = false
		}
	case 1:
		if u.Config.ConflictPolicy == Preserve {
			u.Config.ConflictPolicy = Replace
		} else {
			u.Config.ConflictPolicy = Preserve
		}
	case 2:
		u.Config.Packages = !u.Config.Packages
	case 3:
		u.Config.MangoHook = !u.Config.MangoHook
	case 4:
		if !u.Config.MangoHook {
			return
		}
		layouts := []string{"system", "us", "de", "us,de"}
		u.Config.KeyboardLayout = layouts[cycleIndex(layouts, u.Config.KeyboardLayout, delta)]
	case 5:
		if system.DetectDistro().Family != system.FamilyArch {
			return
		}
		u.Config.SystemUpgrade = !u.Config.SystemUpgrade
		if u.Config.SystemUpgrade {
			u.Config.Packages = true
		}
	}
}

func cycleIndex[T comparable](values []T, current T, delta int) int {
	idx := 0
	for i, v := range values {
		if v == current {
			idx = i
		}
	}
	return (idx + delta + len(values)) % len(values)
}

func (u *UI) preparePlan() {
	u.Plan, u.buildErr = PreparePlan(u.Config)
	u.stage, u.offset = stageReview, 0
}

func (u *UI) startRun() tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	u.cancel = cancel
	stream := make(chan tea.Msg, 64)
	u.stream = stream
	u.stage = stageProgress
	plan := u.Plan
	go func() {
		result := (Engine{Packages: ArchPackages{Runner: u.runner}, Emit: func(ev Event) {
			// Thousands of tiny payload files can be written in seconds. Painting
			// every one is slower than the filesystem work and turns progress into
			// visual noise, so retain early context and regular milestones.
			if ev.Step == "packages" || ev.Step == "done" || ev.Done < 5 || ev.Done%25 == 0 || ev.Done+1 >= ev.Total {
				stream <- runEventMsg{ev}
			}
		}}).Run(ctx, plan)
		stream <- runResultMsg{result}
		close(stream)
	}()
	return waitStream(stream)
}

func (u *UI) authorize() tea.Cmd {
	if u.Config.DryRun || len(u.Plan.Impact.Packages) == 0 || u.runner == nil {
		return u.startRun()
	}
	if system.DetectDistro().Family != system.FamilyArch {
		return u.startRun()
	}
	manager, err := pkg.FindManager(string(system.FamilyArch))
	if err != nil || !manager.Privileged || !u.runner.NeedsPrivileges() || u.runner.HasPrivileges(context.Background()) {
		return u.startRun()
	}
	name, args, ok := u.runner.AcquireCommand()
	if !ok {
		return u.startRun()
	}
	cmd := exec.Command(name, args...)
	return tea.ExecProcess(cmd, func(err error) tea.Msg { return authResultMsg{Err: err} })
}

func waitStream(stream <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-stream
		if !ok {
			return runResultMsg{Result: Result{Err: fmt.Errorf("installer event stream closed unexpectedly")}}
		}
		return msg
	}
}

func (u *UI) View() tea.View {
	view := tea.NewView("")
	view.AltScreen = true
	if u.width <= 0 {
		u.width = 80
	}
	body := u.render()
	view.Content = lipgloss.NewStyle().Padding(1, 3).Width(max(20, min(u.width-6, 100))).Render(body)
	return view
}

func (u *UI) render() string {
	header := u.accent.Render("Illogical-mango · installer v2")
	meta := u.muted.Render(strings.ToUpper(string(u.Config.Operation)))
	var body string
	switch u.stage {
	case stageWelcome:
		body = u.welcomeView()
	case stageConfigure:
		body = u.configureView()
	case stageReview:
		body = u.reviewView()
	case stageProgress:
		body = u.progressView()
	case stageDone:
		body = u.doneView()
	}
	return header + "  " + meta + "\n\n" + body
}

func (u *UI) welcomeView() string {
	ru := u.ru()
	title := tr(ru, "Install with evidence, not guesses", "Установка с проверяемыми последствиями")
	lines := []string{u.accent.Render(title), "",
		tr(ru, "This installer first computes every file it will create, replace or remove.", "Сначала установщик вычислит каждый создаваемый, заменяемый и удаляемый файл."),
		tr(ru, "User edits are preserved by default. A write-ahead journal rolls back interrupted runs.", "Пользовательские правки по умолчанию сохраняются. Журнал откатит прерванную операцию."), "",
		u.muted.Render(tr(ru, "No display-manager changes. No hidden system upgrade. No package removal on uninstall.", "Без правки display manager, скрытого обновления системы и удаления пакетов при uninstall.")), "",
		tr(ru, "enter  continue    q  quit", "enter  продолжить    q  выйти")}
	return strings.Join(lines, "\n")
}

func (u *UI) configureView() string {
	ru := u.ru()
	rows := [][2]string{
		{tr(ru, "Preset", "Профиль"), string(u.Config.Preset)},
		{tr(ru, "Existing files", "Существующие файлы"), string(u.Config.ConflictPolicy)},
		{tr(ru, "Dependencies", "Зависимости"), onOff(ru, u.Config.Packages)},
		{tr(ru, "Mango integration", "Интеграция Mango"), onOff(ru, u.Config.MangoHook)},
		{tr(ru, "Keyboard layout", "Раскладка"), u.Config.KeyboardLayout},
		{tr(ru, "Full system upgrade", "Полное обновление системы"), onOff(ru, u.Config.SystemUpgrade)},
	}
	if !u.Config.MangoHook {
		rows[4][1] += " " + tr(ru, "(inactive)", "(не используется)")
	}
	if system.DetectDistro().Family != system.FamilyArch {
		rows[5][1] = tr(ru, "unavailable on this distro", "недоступно в этом дистрибутиве")
	}
	lines := []string{u.accent.Render(tr(ru, "Choose scope", "Выберите объём")),
		tr(ru, "Recommended avoids unrelated desktop dotfiles. Full is deliberately broad.", "Recommended не трогает посторонние desktop-dotfiles. Full намеренно широкий."), ""}
	for i, row := range rows {
		cursor := "  "
		if i == u.cursor {
			cursor = u.accent.Render("› ")
		}
		value := row[1]
		if i == 4 && u.editing {
			value += "█"
		}
		lines = append(lines, fmt.Sprintf("%s%-24s %s", cursor, row[0], u.accent.Render(value)))
	}
	lines = append(lines, "", tr(ru, "↑↓ move  ←→ change  e edit layout  enter review  esc back", "↑↓ выбор  ←→ изменить  e ввести раскладку  enter проверить  esc назад"))
	return strings.Join(lines, "\n")
}

func (u *UI) reviewView() string {
	ru := u.ru()
	if u.buildErr != nil {
		return u.bad.Render(tr(ru, "Cannot build a safe plan: ", "Невозможно построить безопасный план: ")+u.buildErr.Error()) +
			"\n\n" + tr(ru, "esc  back", "esc  назад")
	}
	p := u.Plan
	lines := []string{u.accent.Render(tr(ru, "Review actual impact", "Проверьте реальные последствия")),
		fmt.Sprintf(tr(ru, "Create %d · replace %d · remove stale %d", "Создать %d · заменить %d · удалить устаревших %d"),
			p.Impact.Create, p.Impact.Replace, p.Impact.RemoveStale),
		fmt.Sprintf(tr(ru, "Keep modified %d · unchanged %d", "Сохранить изменённых %d · без изменений %d"),
			p.Impact.KeepModified, p.Impact.Unchanged)}
	if p.Config.Operation == Rollback {
		lines[1] = fmt.Sprintf(tr(ru, "Restore previous %d · remove newly-created %d",
			"Восстановить прежних %d · удалить созданных последней операцией %d"),
			p.Impact.RestorePrevious, p.Impact.RemoveCreated)
		lines = lines[:2]
	}
	lines = append(lines, wrapPlain(tr(ru, "Equivalent: ", "Эквивалентная команда: ")+p.Config.CommandLine(), u.contentWidth())...)
	if len(p.Impact.Packages) > 0 {
		lines = append(lines, wrapPlain(fmt.Sprintf(tr(ru, "Packages (%d): %s", "Пакеты (%d): %s"),
			len(p.Impact.Packages), strings.Join(p.Impact.Packages, ", ")), u.contentWidth())...)
	}
	for _, detail := range p.Impact.Details {
		lines = append(lines, wrapPlain(detail, u.contentWidth())...)
	}
	for _, warning := range p.Impact.Warnings {
		warning = localizeWarning(ru, warning)
		wrapped := wrapPlain(warning, max(10, u.contentWidth()-2))
		for i, line := range wrapped {
			prefix := "  "
			if i == 0 {
				prefix = "⚠ "
			}
			lines = append(lines, u.warn.Render(prefix+line))
		}
	}
	lines = append(lines, "", u.muted.Render(tr(ru, "Filesystem actions", "Операции с файлами")))
	for _, action := range p.Actions {
		lines = append(lines, fmt.Sprintf("%-7s %s", action.Kind,
			truncateMiddle(action.Path, max(8, u.contentWidth()-8))))
	}
	if len(p.Actions) == 0 && p.Config.Operation != Rollback {
		lines = append(lines, u.muted.Render(tr(ru, "Nothing needs changing.", "Изменения не требуются.")))
	}
	footer := tr(ru, "enter apply  ↑↓ scroll  esc back  q quit", "enter применить  ↑↓ прокрутка  esc назад  q выйти")
	return u.viewport(lines, footer)
}

func (u *UI) progressView() string {
	ru := u.ru()
	lines := []string{u.accent.Render(tr(ru, "Applying transaction", "Применение транзакции")),
		tr(ru, "Ctrl+C requests cancellation; rollback completes before exit.", "Ctrl+C отменяет операцию; перед выходом будет завершён откат."), ""}
	if u.cancelling {
		lines = append(lines, u.warn.Render(tr(ru, "Cancelling — waiting for rollback to finish…", "Отмена — ожидаем завершения отката…")), "")
	}
	for _, ev := range u.events {
		progress := ""
		if ev.Total > 0 {
			progress = fmt.Sprintf(" %d/%d", ev.Done, ev.Total)
		}
		prefix := fmt.Sprintf("%-9s%s  ", ev.Step, progress)
		lines = append(lines, prefix+truncateMiddle(ev.Detail, max(8, u.contentWidth()-len(prefix))))
	}
	return u.viewport(lines, tr(ru, "Working…", "Выполняется…"))
}

func (u *UI) doneView() string {
	ru := u.ru()
	if u.Result.Err != nil {
		body := u.bad.Render("✗ "+tr(ru, "Operation failed", "Операция завершилась ошибкой")) + "\n\n" + u.Result.Err.Error()
		if u.Result.LogPath != "" {
			body += "\n\n" + tr(ru, "Transcript: ", "Журнал: ") + u.Result.LogPath
		}
		return body + "\n\n" + tr(ru, "enter  close", "enter  закрыть")
	}
	body := u.good.Render("✓ "+tr(ru, "Operation complete", "Операция завершена")) + "\n\n" +
		fmt.Sprintf(tr(ru, "%d paths changed; %d modified paths kept; %s elapsed.",
			"Изменено путей: %d; сохранено изменённых: %d; время: %s."),
			u.Result.Changed, u.Result.Kept, u.Result.Duration.Round(time.Millisecond))
	if u.Result.LogPath != "" {
		body += "\n" + tr(ru, "Transcript: ", "Журнал: ") + u.Result.LogPath
	}
	return body + "\n\n" + tr(ru, "enter  close", "enter  закрыть")
}

func (u *UI) viewport(lines []string, footer string) string {
	available := max(4, u.height-12)
	maxOffset := max(0, len(lines)-available)
	if u.offset > maxOffset {
		u.offset = maxOffset
	}
	end := min(len(lines), u.offset+available)
	visible := lines[u.offset:end]
	if maxOffset > 0 {
		footer += fmt.Sprintf("  [%d–%d/%d]", u.offset+1, end, len(lines))
	}
	return strings.Join(visible, "\n") + "\n\n" + u.muted.Render(footer)
}

func (u *UI) contentWidth() int { return max(20, min(u.width-8, 98)) }

func truncateMiddle(value string, width int) string {
	runes := []rune(value)
	if len(runes) <= width || width < 5 {
		return value
	}
	left := (width - 1) / 2
	right := width - left - 1
	return string(runes[:left]) + "…" + string(runes[len(runes)-right:])
}

func wrapPlain(value string, width int) []string {
	if width < 10 {
		return []string{truncateMiddle(value, width)}
	}
	words := strings.Fields(value)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	line := words[0]
	for _, word := range words[1:] {
		if utf8.RuneCountInString(line)+1+utf8.RuneCountInString(word) <= width {
			line += " " + word
			continue
		}
		lines = append(lines, truncateMiddle(line, width))
		line = word
	}
	return append(lines, truncateMiddle(line, width))
}

func localizeWarning(ru bool, warning string) string {
	if !ru {
		return warning
	}
	exact := map[string]string{
		"Filesystem sandbox active: no host commands or package tools will run.":                                                                     "Активен файловый sandbox: команды хоста и package manager запускаться не будут.",
		"Replace policy is active: conflicting files will be overwritten after their exact previous state is journalled.":                            "Активна политика replace: конфликтующие файлы будут заменены после записи их точного исходного состояния в журнал.",
		"Package-manager changes are outside the filesystem transaction and are never automatically removed; the exact package list is shown below.": "Изменения package manager не входят в файловую транзакцию и автоматически не удаляются; точный список пакетов показан ниже.",
		"You explicitly enabled a full Arch system upgrade. This can change packages unrelated to Illogical-mango.":                                  "Вы явно включили полное обновление Arch. Оно может изменить пакеты, не связанные с Illogical-mango.",
		"A legacy v1 installation record will be migrated transactionally.":                                                                          "Старый manifest v1 будет перенесён транзакционно.",
		"Recovered and rolled back an interrupted previous transaction before computing this plan.":                                                  "Перед расчётом плана восстановлена и отменена прерванная предыдущая транзакция.",
	}
	if translated, ok := exact[warning]; ok {
		return translated
	}
	for prefix, translated := range map[string]string{
		"Kept pre-existing file: ":                              "Сохранён существовавший до установки файл: ",
		"Kept identical pre-existing file unowned: ":            "Сохранён без принятия во владение идентичный существующий файл: ",
		"Kept locally modified installed file: ":                "Сохранён локально изменённый установленный файл: ",
		"Kept stale but modified path: ":                        "Сохранён устаревший, но изменённый путь: ",
		"Will keep modified path: ":                             "Будет сохранён изменённый путь: ",
		"Automatic packages are intentionally unavailable for ": "Автоматические пакеты намеренно недоступны для ",
		"MangoWM was not found on PATH. ":                       "MangoWM не найден в PATH. ",
	} {
		if rest, ok := strings.CutPrefix(warning, prefix); ok {
			return translated + rest
		}
	}
	return warning
}

func (u *UI) ru() bool {
	lang := u.Config.Language
	if lang == "auto" {
		lang = strings.ToLower(os.Getenv("LANG"))
	}
	return strings.HasPrefix(lang, "ru")
}

func tr(ru bool, en, russian string) string {
	if ru {
		return russian
	}
	return en
}
func onOff(ru, on bool) string {
	if on {
		return tr(ru, "on", "вкл")
	}
	return tr(ru, "off", "выкл")
}
