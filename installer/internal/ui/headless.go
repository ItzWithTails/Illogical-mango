package ui

import (
	"context"
	"fmt"
	"io"
	"strings"

	"ilmango/internal/installer"
	"ilmango/internal/system"
	"ilmango/internal/ui/theme"
)

// Headless runs a plan without taking over the terminal. It exists so the same
// step catalog serves CI, scripted installs and the `--yes` flag, rather than
// those paths drifting into a separate implementation.
type Headless struct {
	Out     io.Writer
	Theme   theme.Theme
	Verbose bool // stream every line of phase output rather than step summaries
}

// Run executes the plan and returns the first error.
func (h Headless) Run(ctx context.Context, session *Session) error {
	plan := installer.BuildPlan(session.Config, session.Steps)
	j := openJournal()
	defer j.close()

	h.printf("%s\n", h.Theme.Title.Render("Illogical-mango "+session.Repo.Version+"  "+strings.ToLower(session.Verb())))
	h.printf("%s\n\n", h.Theme.Faint.Render(fmt.Sprintf("%s · %d steps", session.Distro.Name, plan.Active())))

	err := installer.Runner{}.Run(ctx, plan, session.Env, func(event installer.Event) {
		h.report(j, event)
	})

	session.Outcome = Outcome{
		Complete: err == nil,
		Err:      err,
		LogPath:  j.path(),
		Notes:    session.Env.Notes,
	}

	if err != nil {
		h.printf("\n%s %s\n", h.Theme.Error.Render(theme.GlyphFail), err)
		if path := j.path(); path != "" {
			h.printf("%s\n", h.Theme.Faint.Render("transcript: "+path))
		}
		return err
	}

	completion := "Installation complete."
	switch {
	case session.Config.DryRun:
		completion = "Dry run complete — nothing was changed."
	case session.Removing():
		completion = "Uninstall complete."
	case session.Operation == installer.OpUpdate:
		completion = "Update complete."
	case session.Operation == installer.OpRollback:
		completion = "Rollback complete."
	case session.Operation == installer.OpChanges:
		completion = "Nothing was changed; this was a report."
	}
	h.printf("\n%s %s\n", h.Theme.Success.Render(theme.GlyphDone), completion)
	for _, note := range session.Env.Notes {
		h.printf("%s %s\n", h.Theme.Warning.Render(theme.GlyphWarn), note)
	}
	return nil
}

// report renders one event as a line of plain output.
func (h Headless) report(j *journal, event installer.Event) {
	switch e := event.(type) {
	case installer.StepStarted:
		j.section(e.Step.Title())
		h.printf("%s %s\n", h.Theme.Accent.Render(theme.GlyphCursor), e.Step.Title())

	case installer.StepSkipped:
		h.printf("%s %s\n", h.Theme.Faint.Render(theme.GlyphSkip), h.Theme.Faint.Render(e.Step.Title()))

	case installer.StepDetail:
		// A step's inline status is where progress lives — which package of
		// how many, which directory is being copied. In the interface it
		// overwrites one line; here each is worth its own, because a headless
		// run is read afterwards as much as watched.
		j.line(e.Text)
		h.printf("  %s\n", h.Theme.Faint.Render(e.Text))

	case installer.StepOutput:
		j.line(e.Line)
		if h.Verbose {
			h.printf("  %s\n", h.Theme.LogLine.Render(e.Line))
		}

	case installer.StepFinished:
		if e.Err != nil {
			j.line("error: " + e.Err.Error())
			marker := h.Theme.Error.Render(theme.GlyphFail)
			if e.Optional {
				marker = h.Theme.Warning.Render(theme.GlyphWarn)
			}
			h.printf("  %s %s\n", marker, e.Err)
			return
		}
		h.printf("  %s %s\n", h.Theme.Success.Render(theme.GlyphDone), h.Theme.Faint.Render(formatElapsed(e.Duration)))

	case installer.RunFinished:
		_ = e
	}
}

func (h Headless) printf(format string, args ...any) {
	fmt.Fprintf(h.Out, format, args...)
}

// PrintChecks writes preflight results for the headless path.
func (h Headless) PrintChecks(results []system.CheckResult) {
	for _, r := range results {
		marker := h.Theme.Success.Render(theme.GlyphDone)
		switch r.Status {
		case system.CheckWarn:
			marker = h.Theme.Warning.Render(theme.GlyphWarn)
		case system.CheckFail:
			marker = h.Theme.Error.Render(theme.GlyphFail)
		}
		h.printf("%s %s %s\n", marker, r.Title, h.Theme.Faint.Render(r.Detail))
	}
	h.printf("\n")
}
