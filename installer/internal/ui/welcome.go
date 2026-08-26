package ui

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"ilmango/internal/system"
	"ilmango/internal/ui/theme"
	"ilmango/internal/ui/widget"
)

// welcomeScreen introduces the installer and reports whether this machine is
// ready. Preflight runs off the UI goroutine so a slow check cannot freeze the
// interface.
type welcomeScreen struct {
	session *Session
	spinner spinner.Model
	done    bool
}

// checksDoneMsg carries the finished preflight suite back to the UI.
type checksDoneMsg struct{ results []system.CheckResult }

func newWelcomeScreen(session *Session) Screen {
	sp := spinner.New(spinner.WithSpinner(spinner.Dot))
	sp.Style = session.Theme.Accent
	return &welcomeScreen{session: session, spinner: sp}
}

func (s *welcomeScreen) Init() tea.Cmd {
	return tea.Batch(s.spinner.Tick, runPreflight(s.session.Repo))
}

// runPreflight executes the checks in the background.
func runPreflight(repo system.Repo) tea.Cmd {
	return func() tea.Msg {
		return checksDoneMsg{results: system.RunChecks(context.Background(), repo)}
	}
}

func (s *welcomeScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case checksDoneMsg:
		s.session.Checks = msg.results
		s.done = true
		return s, nil

	case spinner.TickMsg:
		if s.done {
			return s, nil
		}
		var cmd tea.Cmd
		s.spinner, cmd = s.spinner.Update(msg)
		return s, cmd

	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "esc":
			return s, quit(true)
		case "enter":
			if s.blocked() {
				return s, nil
			}
			if s.session.Removing() {
				return s, navigate(StageReview)
			}
			return s, navigate(StageOptions)
		}
	}
	return s, nil
}

// blocked reports whether preflight found something disqualifying.
func (s *welcomeScreen) blocked() bool {
	return !s.done || system.Blocking(s.session.Checks)
}

func (s *welcomeScreen) Chrome() Chrome {
	keys := []widget.Key{{Keys: "enter", Help: "continue"}, {Keys: "q", Help: "quit"}}
	if s.blocked() && s.done {
		keys = []widget.Key{{Keys: "q", Help: "quit"}}
	}
	heading, explanation := "Install Illogical-mango", "A Quickshell desktop for Mango. Let's check this machine first."
	if s.session.Removing() {
		heading = "Uninstall Illogical-mango"
		explanation = "This removes what the installer put here, and keeps anything you edited."
	}

	return Chrome{
		Heading:     heading,
		Explanation: explanation,
		Meta:        repoMeta(s.session),
		Keys:        keys,
	}
}

func (s *welcomeScreen) View() string {
	t := s.session.Theme
	width := s.session.ContentWidth()

	sections := []string{
		s.systemCard(),
		"",
		t.Muted.Render("Preflight"),
	}

	if s.done {
		sections = append(sections, widget.CheckList(t, s.session.Checks))
	} else {
		sections = append(sections, "  "+s.spinner.View()+t.Faint.Render(" checking this machine…"))
	}

	if s.done {
		sections = append(sections, "", s.verdict(width))
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// systemCard renders the identified host as a compact label column.
func (s *welcomeScreen) systemCard() string {
	t := s.session.Theme
	d := s.session.Distro

	rows := [][2]string{
		{"Distribution", d.Name},
		{"Family", string(d.Family) + t.Faint.Render("  ("+d.Support().String()+")")},
		{"Source", fmt.Sprintf("%s  %s", s.session.Repo.Root, branchNote(t, s.session.Repo.Branch))},
	}

	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		label := t.Faint.Render(pad(row[0], 14))
		lines = append(lines, "  "+label+t.Body.Render(row[1]))
	}
	return strings.Join(lines, "\n")
}

// verdict states plainly whether the user can proceed.
func (s *welcomeScreen) verdict(width int) string {
	t := s.session.Theme

	if system.Blocking(s.session.Checks) {
		return widget.Wrap(t, width, t.Error.Render(theme.GlyphFail+" ")+
			t.Body.Render("This machine is not ready. Resolve the failures above and run the installer again."))
	}

	if s.session.Distro.Support() == system.SupportManual {
		return widget.Wrap(t, width, t.Warning.Render(theme.GlyphWarn+" ")+
			t.Body.Render("NixOS installs declaratively. Continue only if you already have the dependencies; the config files will still be placed."))
	}

	if warned(s.session.Checks) {
		return widget.Wrap(t, width, t.Warning.Render(theme.GlyphWarn+" ")+
			t.Body.Render("Some checks came back with warnings. Installation can proceed."))
	}

	if s.session.Removing() {
		return t.Success.Render(theme.GlyphDone+" ") + t.Body.Render("Ready to uninstall.")
	}
	return t.Success.Render(theme.GlyphDone+" ") + t.Body.Render("Ready to install.")
}

func warned(results []system.CheckResult) bool {
	for _, r := range results {
		if r.Status == system.CheckWarn {
			return true
		}
	}
	return false
}

func branchNote(t theme.Theme, branch string) string {
	if branch == "" || branch == "HEAD" {
		return ""
	}
	return t.Faint.Render("on " + branch)
}

// pad right-aligns a label column without pulling in a table dependency.
func pad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
