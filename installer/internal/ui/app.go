package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"ilmango/internal/ui/theme"
	"ilmango/internal/ui/widget"
)

// App is the root Bubble Tea model. It owns the frame, the terminal size and
// the global keys, and delegates everything else to the active screen.
type App struct {
	session *Session
	stage   Stage
	screen  Screen

	quitting bool
	// aborted records that the user left before the run finished, so main can
	// exit with a non-zero status.
	aborted bool
}

var _ tea.Model = (*App)(nil)

// NewApp builds the root model for a session.
func NewApp(session *Session) *App {
	app := &App{session: session, stage: StageWelcome}
	app.screen = newScreen(app.stage, session)
	return app
}

// Aborted reports whether the user quit before the installation finished.
func (a *App) Aborted() bool { return a.aborted }

// Outcome exposes the installation result for the caller's exit status.
func (a *App) Outcome() Outcome { return a.session.Outcome }

func (a *App) Init() tea.Cmd {
	return a.screen.Init()
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.session.Width = msg.Width
		a.session.Height = msg.Height
		// Fall through: screens with viewports need the size too.

	case tea.KeyPressMsg:
		// Ctrl+C always works, even mid-install; the running plan is cancelled
		// before the program exits so no child process is left behind.
		if msg.String() == "ctrl+c" {
			return a, a.abort()
		}

	case navigateMsg:
		a.stage = msg.to
		a.screen = newScreen(a.stage, a.session)
		return a, a.screen.Init()

	case quitMsg:
		a.quitting = true
		a.aborted = msg.aborted
		return a, tea.Quit
	}

	screen, cmd := a.screen.Update(msg)
	a.screen = screen
	return a, cmd
}

func (a *App) View() tea.View {
	view := tea.NewView("")
	// The installer takes the whole screen: a long package transcript should
	// not push the user's shell history away.
	view.AltScreen = true

	if a.quitting {
		return view
	}

	t := a.session.Theme
	width := a.session.ContentWidth()
	chrome := a.screen.Chrome()

	sections := []string{
		widget.Header(t, width, "Illogical-mango", chrome.Meta),
		"",
		widget.Title(t, chrome.Heading, chrome.Explanation),
		"",
		a.screen.View(),
	}

	body := lipgloss.JoinVertical(lipgloss.Left, sections...)
	if footer := widget.Footer(t, width, chrome.Keys); footer != "" {
		body = lipgloss.JoinVertical(lipgloss.Left, body, "", footer)
	}

	view.Content = t.Screen.Render(body)
	return view
}

// abort stops any running plan before quitting, so a bash phase is killed
// rather than left orphaned when the user interrupts.
func (a *App) abort() tea.Cmd {
	if a.session.cancelRun != nil {
		a.session.cancelRun()
		a.session.cancelRun = nil
	}
	a.quitting = true
	a.aborted = !a.session.Outcome.Complete
	return tea.Quit
}

// quitMsg ends the program from within a screen.
type quitMsg struct{ aborted bool }

// quit returns a command that ends the program.
func quit(aborted bool) tea.Cmd {
	return func() tea.Msg { return quitMsg{aborted: aborted} }
}

// stageMeta renders the "step N of M" indicator shown in the header.
func stageMeta(session *Session, stage Stage) string {
	labels := []string{"System", "Options", "Review", "Install", "Done"}
	if int(stage) >= len(labels) {
		return ""
	}

	parts := make([]string, 0, len(labels))
	for i, label := range labels {
		switch {
		case Stage(i) == stage:
			parts = append(parts, session.Theme.Accent.Render(label))
		case Stage(i) < stage:
			parts = append(parts, session.Theme.Muted.Render(label))
		default:
			parts = append(parts, session.Theme.Faint.Render(label))
		}
	}
	return strings.Join(parts, session.Theme.Faint.Render(" "+theme.GlyphBullet+" "))
}

// repoMeta renders the version identity shown on the first screens.
func repoMeta(session *Session) string {
	meta := "v" + session.Repo.Version
	if session.Repo.Commit != "" && session.Repo.Commit != "unknown" {
		meta += fmt.Sprintf(" · %s", session.Repo.Commit)
	}
	return meta
}
