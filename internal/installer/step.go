package installer

import (
	"context"

	"ilmango/internal/fsx"
	"ilmango/internal/run"
	"ilmango/internal/system"
)

// Step is one unit of installation work.
//
// A step never calls os/exec or writes files directly: it goes through the
// Runner and Tree on its Env, both of which honour dry-run and can be pointed
// at a temporary root. That is what makes the installer testable without
// modifying the machine it is running on.
type Step interface {
	// ID is stable and unique; it appears in logs and diagnostics.
	ID() string
	// Title is the one-line label shown in the step list.
	Title() string
	// Description explains what the step touches, in one sentence.
	Description() string
	// AppliesTo reports whether the step should run under this configuration.
	AppliesTo(Config) bool
	// Run performs the work, honouring ctx cancellation.
	Run(ctx context.Context, env *Env) error
}

// Env is everything a step is allowed to touch.
type Env struct {
	Config Config
	Repo   system.Repo
	Distro system.Distro

	// Runner executes external commands.
	Runner *run.Runner
	// Home is the user's home directory as a writable tree.
	Home fsx.Tree
	// Backup collects whatever the run replaces. It may be nil.
	Backup *fsx.Backup

	// Manifest records what an installation writes, and is what an uninstall
	// reads back. It may be nil during operations that need neither.
	Manifest *Manifest

	// Reporter streams progress to the UI.
	Reporter Reporter

	// Notes accumulates messages worth showing on the summary screen —
	// packages that were unavailable, actions the user still has to take.
	Notes []string
}

// Note records a message for the summary. Steps use it instead of logging
// something important into an output stream the user may never scroll back to.
func (e *Env) Note(text string) { e.Notes = append(e.Notes, text) }

// NoteApplied records a message only when the run actually changed something.
// It is for notes that describe a completed action — telling a user to log out
// for group changes that a dry run never made would be a lie.
func (e *Env) NoteApplied(text string) {
	if !e.Config.DryRun {
		e.Note(text)
	}
}

// Log forwards a line of output, tolerating a nil reporter.
func (e *Env) Log(line string) {
	if e.Reporter != nil {
		e.Reporter.Log(line)
	}
}

// Detail updates the step's inline status.
func (e *Env) Detail(text string) {
	if e.Reporter != nil {
		e.Reporter.Detail(text)
	}
}

// Reporter receives streamed progress from a running step.
type Reporter interface {
	Log(line string)
	Detail(text string)
}
