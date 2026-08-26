package ui

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"ilmango/internal/installer"
	"ilmango/internal/run"
	"ilmango/internal/system"
	"ilmango/internal/ui/theme"
	"ilmango/internal/ui/widget"
)

// reviewScreen is the last stop before anything is changed. It states what
// will run, calls out the choices that reach outside the config directory, and
// prints the equivalent shell invocation so the run stays reproducible.
type reviewScreen struct {
	session *Session
	plan    installer.Plan

	// authErr records a failed privilege prompt, so the user is told why
	// nothing happened when they pressed enter.
	authErr error
}

func newReviewScreen(session *Session) Screen {
	return &reviewScreen{
		session: session,
		plan:    installer.BuildPlan(session.Config, session.Steps),
	}
}

func (s *reviewScreen) Init() tea.Cmd { return nil }

func (s *reviewScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	if failed, ok := msg.(authFailedMsg); ok {
		s.authErr = failed.err
		return s, nil
	}

	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return s, nil
	}

	switch key.String() {
	case "q":
		return s, quit(true)
	case "esc", "left", "h":
		return s, navigate(StageOptions)
	case "enter":
		if s.plan.Mutating() == 0 {
			return s, nil
		}
		return s, s.begin()
	}
	return s, nil
}

// begin obtains a privilege credential, if the plan needs one, before moving
// to the progress screen.
//
// The credential is taken here, with the interface stepped aside, because once
// installation starts the TUI owns the terminal: a password prompt raised then
// would be painted over and the install would wait forever on input the user
// cannot see.
func (s *reviewScreen) begin() tea.Cmd {
	runner := s.session.Env.Runner

	switch {
	case s.session.Config.DryRun:
		// A dry run executes nothing, so it needs no credential.
	case !s.session.Config.NeedsPrivileges():
	case !runner.NeedsPrivileges():
		// Nothing to escalate with; the steps will report this themselves.
	case runner.HasPrivileges(context.Background()):
		// A valid credential is already cached.
	default:
		return s.authenticate(runner)
	}
	return navigate(StageProgress)
}

// authenticate hands the terminal back so the escalation tool can prompt.
func (s *reviewScreen) authenticate(runner *run.Runner) tea.Cmd {
	name, args, ok := runner.AcquireCommand()
	if !ok {
		return navigate(StageProgress)
	}

	cmd := exec.Command(name, args...)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return authFailedMsg{err: err}
		}
		return navigateMsg{to: StageProgress}
	})
}

// authFailedMsg reports that the user could not or would not authenticate.
type authFailedMsg struct{ err error }

func (s *reviewScreen) Chrome() Chrome {
	keys := []widget.Key{
		{Keys: "enter", Help: "install"},
		{Keys: "esc", Help: "back"},
		{Keys: "q", Help: "quit"},
	}

	explanation := "Nothing has been changed yet. This is what will happen."
	switch {
	case s.session.Config.DryRun:
		explanation = "Dry run: every step will be announced, none will be executed."
	case s.session.Removing():
		explanation = "Files you have edited since installing will be left alone."
	}

	keys[0] = widget.Key{Keys: "enter", Help: "install"}
	if s.session.Removing() {
		keys[0] = widget.Key{Keys: "enter", Help: "uninstall"}
	}

	if s.plan.Mutating() == 0 {
		keys = keys[1:]
	}

	return Chrome{
		Heading:     "Review",
		Explanation: explanation,
		Meta:        stageMeta(s.session, StageReview),
		Keys:        keys,
	}
}

func (s *reviewScreen) View() string {
	t := s.session.Theme
	width := s.session.ContentWidth()

	sections := []string{
		t.Muted.Render(fmt.Sprintf("Plan  %s", planSummary(s.plan))),
		s.planList(),
	}

	if warnings := s.warnings(); len(warnings) > 0 {
		sections = append(sections, "", t.Muted.Render("Worth knowing"))
		for _, w := range warnings {
			sections = append(sections, "  "+t.Warning.Render(theme.GlyphWarn)+" "+
				widget.Wrap(t, width-4, t.Body.Render(w)))
		}
	}

	if !s.session.Removing() {
		sections = append(sections,
			"",
			t.Muted.Render("Equivalent command"),
			"  "+t.Faint.Render(s.session.Config.CommandLine()),
		)
	}

	if s.plan.Mutating() == 0 {
		sections = append(sections, "",
			t.Error.Render(theme.GlyphFail+" ")+t.Body.Render("Every stage is switched off — there is nothing to install."))
	}

	if s.session.Removing() && s.session.Env.Manifest != nil {
		sections = append(sections, "",
			t.Muted.Render("Recorded installation"),
			"  "+t.Faint.Render(fmt.Sprintf("v%s, %d files, installed %s",
				s.session.Env.Manifest.Release,
				s.session.Env.Manifest.Len(),
				s.session.Env.Manifest.InstalledAt)))
	}

	if s.authErr != nil {
		sections = append(sections, "",
			t.Error.Render(theme.GlyphFail+" ")+widget.Wrap(t, width-2,
				t.Body.Render("Authentication failed, so nothing was installed. Press enter to try again.")))
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// planList renders the resolved plan, keeping skipped steps visible so the
// shape of a full installation stays legible.
func (s *reviewScreen) planList() string {
	rows := make([]widget.StepRow, 0, len(s.plan.Steps))
	for _, planned := range s.plan.Steps {
		state := installer.StatePending
		if planned.Skipped {
			state = installer.StateSkipped
		}
		rows = append(rows, widget.StepRow{Title: planned.Step.Title(), State: state})
	}
	return widget.StepList(s.session.Theme, rows)
}

// warnings collects everything the user should read before committing.
func (s *reviewScreen) warnings() []string {
	var out []string

	for _, o := range installer.Options() {
		if o.Risky && s.session.Config.Effective(o.ID) {
			out = append(out, riskNote(o))
		}
	}

	for _, c := range installer.Choices() {
		if c.Requires != "" && !s.session.Config.Effective(c.Requires) {
			continue
		}
		if note := c.Warning(s.session.Config.Choice(c.ID)); note != "" {
			out = append(out, note)
		}
	}

	if s.session.Config.Effective(installer.OptConfigFiles) && !s.session.Config.Effective(installer.OptBackup) {
		out = append(out, "Backups are off: existing configs will be overwritten with no copy kept.")
	}

	switch s.session.Distro.Support() {
	case system.SupportExperimental:
		out = append(out, fmt.Sprintf("%s support is experimental — dependencies fall back to the generic installer and may need manual work.", s.session.Distro.Name))
	case system.SupportManual:
		out = append(out, "NixOS needs its dependencies declared in your system configuration; this run will not install them.")
	}

	return out
}

func riskNote(o installer.Option) string {
	switch o.ID {
	case installer.OptSystemSetup:
		return "System configuration adds your user to groups and enables services — it will ask for your password."
	case installer.OptDefaultShell:
		return "Your login shell will be changed to Fish via chsh."
	default:
		return o.Title + " changes state outside your config directory."
	}
}

func planSummary(plan installer.Plan) string {
	skipped := len(plan.Steps) - plan.Active()
	if skipped == 0 {
		return fmt.Sprintf("%d steps", plan.Active())
	}
	return fmt.Sprintf("%d steps, %s skipped", plan.Active(), strings.TrimSpace(fmt.Sprint(skipped)))
}
