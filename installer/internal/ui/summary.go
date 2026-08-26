package ui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"ilmango/internal/installer"
	"ilmango/internal/ui/theme"
	"ilmango/internal/ui/widget"
)

// summaryScreen closes the flow: what happened, and what to do next.
type summaryScreen struct {
	session *Session
}

func newSummaryScreen(session *Session) Screen {
	return &summaryScreen{session: session}
}

func (s *summaryScreen) Init() tea.Cmd { return nil }

func (s *summaryScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "enter", "q", "esc":
			return s, quit(false)
		}
	}
	return s, nil
}

func (s *summaryScreen) Chrome() Chrome {
	heading, explanation := "Done", "Illogical-mango is installed."
	switch {
	case !s.session.Outcome.Complete && s.session.Removing():
		heading, explanation = "Uninstall failed", "Nothing further was attempted after the failure."
	case !s.session.Outcome.Complete:
		heading, explanation = "Installation failed", "Nothing further was attempted after the failure."
	case s.session.Config.DryRun:
		heading, explanation = "Dry run complete", "Every step was walked through. Nothing on this machine was changed."
	case s.session.Removing():
		heading, explanation = "Removed", "Illogical-mango is uninstalled."
	}
	return Chrome{
		Heading:     heading,
		Explanation: explanation,
		Meta:        stageMeta(s.session, StageSummary),
		Keys:        []widget.Key{{Keys: "enter", Help: "close"}},
	}
}

func (s *summaryScreen) View() string {
	t := s.session.Theme
	width := s.session.ContentWidth()
	outcome := s.session.Outcome

	var sections []string

	if outcome.Complete {
		headline := fmt.Sprintf("Version %s installed in %s.", s.session.Repo.Version, formatElapsed(outcome.Duration))
		if s.session.Removing() {
			headline = fmt.Sprintf("Removed in %s.", formatElapsed(outcome.Duration))
		}
		if s.session.Config.DryRun {
			headline = fmt.Sprintf("Walked %s of plan without touching anything.", formatElapsed(outcome.Duration))
		}
		sections = append(sections,
			t.Success.Render(theme.GlyphDone+" ")+t.Body.Render(headline),
			"",
			t.Muted.Render("Next"),
		)
		for _, step := range s.nextSteps() {
			sections = append(sections, "  "+t.Faint.Render(theme.GlyphBullet)+" "+widget.Wrap(t, width-4, t.Body.Render(step)))
		}
	} else {
		sections = append(sections,
			t.Error.Render(theme.GlyphFail+" ")+t.Body.Render(failureHeadline(outcome)),
			"",
			widget.Wrap(t, width, t.Muted.Render(errorText(outcome))),
			"",
			t.Muted.Render("Next"),
			"  "+t.Faint.Render(theme.GlyphBullet)+" "+t.Body.Render("Read the transcript, fix the cause, then run the installer again — it is safe to repeat."),
			"  "+t.Faint.Render(theme.GlyphBullet)+" "+t.Body.Render("Or run ./setup doctor for a diagnostic pass."),
		)
	}

	if len(outcome.Notes) > 0 {
		sections = append(sections, "", t.Muted.Render("Read this"))
		for _, note := range outcome.Notes {
			sections = append(sections, "  "+t.Warning.Render(theme.GlyphWarn)+" "+
				widget.Wrap(t, width-4, t.Body.Render(note)))
		}
	}

	if outcome.LogPath != "" {
		sections = append(sections, "", t.Muted.Render("Transcript"), "  "+t.Faint.Render(outcome.LogPath))
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// nextSteps are tailored to what the run actually did.
func (s *summaryScreen) nextSteps() []string {
	if s.session.Removing() {
		return []string{
			"Log out of Mango; the shell is no longer installed.",
			"Packages were left in place — remove any you no longer want yourself.",
		}
	}

	steps := []string{
		"Log out and select Mango at your display manager, then log back in.",
		"Run ilmango run to start the shell in the current session.",
	}
	if s.session.Config.Effective(installer.OptSystemSetup) {
		steps = append(steps, "Group membership changes only take effect after a full logout.")
	}
	if !s.session.Config.Effective(installer.OptDependencies) {
		steps = append(steps, "Dependencies were skipped — make sure quickshell and niri are installed.")
	}
	return steps
}

func failureHeadline(o Outcome) string {
	if o.Failed != "" {
		return fmt.Sprintf("%s failed after %s.", o.Failed, formatElapsed(o.Duration))
	}
	return fmt.Sprintf("The run stopped after %s.", formatElapsed(o.Duration))
}

func errorText(o Outcome) string {
	if o.Err == nil {
		return "No error was reported, which usually means the run was cancelled."
	}
	return o.Err.Error()
}
