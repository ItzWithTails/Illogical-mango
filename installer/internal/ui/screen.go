// Package ui implements the installer's terminal interface.
//
// The flow is a sequence of Screens sharing one Session. A screen owns its own
// state and keybindings and asks to move on by returning a navigation command;
// the root model owns the frame, the window size and the global keys. Adding a
// screen means adding a Stage constant, a constructor, and an entry in
// newScreen — nothing else changes.
package ui

import (
	tea "charm.land/bubbletea/v2"

	"ilmango/internal/installer"
	"ilmango/internal/system"
	"ilmango/internal/ui/theme"
	"ilmango/internal/ui/widget"
)

// Stage identifies a screen in the installer flow.
type Stage int

const (
	StageWelcome Stage = iota
	StageOptions
	StagePackages
	StageReview
	StageProgress
	StageSummary
)

// Screen is one step of the flow.
type Screen interface {
	// Init returns the command to run when the screen becomes active.
	Init() tea.Cmd
	// Update handles a message and returns the screen to display next.
	Update(msg tea.Msg) (Screen, tea.Cmd)
	// View renders the screen body, without the surrounding frame.
	View() string
	// Chrome describes what the frame draws around this screen.
	Chrome() Chrome
}

// Chrome is the frame content a screen requests.
type Chrome struct {
	Heading     string
	Explanation string
	Meta        string
	Keys        []widget.Key
}

// Session is the state every screen shares. Screens hold a pointer, so a
// choice made on one screen is visible to the next.
type Session struct {
	// Operation is what this run does. It selects the step catalogue and the
	// wording throughout, so installing and removing share one interface.
	Operation installer.Operation

	Theme  theme.Theme
	Repo   system.Repo
	Distro system.Distro
	Config installer.Config
	Steps  []installer.Step

	// Env is what steps are allowed to touch: the command runner, the
	// destination filesystem and the backup set. It is built once, by main, so
	// that a dry run or a redirected root applies to the whole session.
	Env *installer.Env

	// Checks is filled in by the welcome screen once preflight completes.
	Checks []system.CheckResult

	// Outcome is filled in by the progress screen.
	Outcome Outcome

	// Width and Height track the terminal, minus the frame's own padding.
	Width  int
	Height int

	// cancelRun is set while a plan is running, so the root model can stop the
	// runner — and its bash child — when the user quits mid-install.
	cancelRun func()
}

// ContentWidth is the width available to a screen body.
func (s *Session) ContentWidth() int {
	w := s.Width - 6 // the frame pads three columns either side
	if w < 20 {
		return 20
	}
	if w > 96 { // long measures are hard to read; cap the text column
		return 96
	}
	return w
}

// ContentHeight is the number of body lines a screen may render.
//
// The frame spends a fixed number of rows on the heading, the explanation and
// the key hints; what is left is the screen's. A screen with more to show than
// this has to scroll, because the frame will not.
func (s *Session) ContentHeight() int {
	h := s.Height - 12
	if h < 6 {
		return 6
	}
	return h
}

// navigateMsg asks the root model to switch screens.
type navigateMsg struct{ to Stage }

// navigate returns a command that moves the flow to the given stage.
func navigate(to Stage) tea.Cmd {
	return func() tea.Msg { return navigateMsg{to: to} }
}

// Removing reports whether this run is an uninstall.
func (s *Session) Removing() bool { return s.Operation == installer.OpUninstall }

// Verb names the operation for use in a sentence.
func (s *Session) Verb() string {
	if s.Removing() {
		return "Uninstall"
	}
	return "Install"
}

// newScreen constructs the screen for a stage.
func newScreen(stage Stage, session *Session) Screen {
	switch stage {
	case StageOptions:
		// Removal has nothing to configure: the manifest already decided what
		// it will touch, so the picker would offer choices with no effect.
		if session.Removing() {
			return newReviewScreen(session)
		}
		return newOptionsScreen(session)
	case StagePackages:
		return newPackagesScreen(session)
	case StageReview:
		return newReviewScreen(session)
	case StageProgress:
		return newProgressScreen(session)
	case StageSummary:
		return newSummaryScreen(session)
	default:
		return newWelcomeScreen(session)
	}
}
