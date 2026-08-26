package steps

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"ilmango/internal/fsx"
	"ilmango/internal/installer"
	"ilmango/internal/run"
)

// The maintenance operations that survived the old shell installer.
//
// That script carried a dozen subcommands. Most of them reported on state the
// installer now records itself, and its migration engine spent 22 of its 37
// migrations rewriting niri configuration, which this fork does not support.
// What genuinely had no replacement was moving a checkout forward, putting a
// replaced file back, and answering "what have I changed?" — so those three
// are here and the rest is gone.

// pullStep brings the checkout forward before an update reinstalls from it.
type pullStep struct{ base }

func newPullStep() installer.Step {
	return pullStep{base{
		id:     "pull",
		title:  "Update the checkout",
		detail: "Fast-forward the repository this shell is installed from.",
	}}
}

func (s pullStep) Run(ctx context.Context, env *installer.Env) error {
	if !run.Exists("git") {
		env.Note("git is not installed, so the checkout was not updated; the files already here were reinstalled.")
		return nil
	}
	if !dirExists(filepath.Join(env.Repo.Root, ".git")) {
		env.Note(env.Repo.Root + " is not a git checkout, so there was nothing to pull; the files already here were reinstalled.")
		return nil
	}

	// --ff-only, never a merge. An update that produces conflict markers in
	// someone's checkout is worse than an update that declines to run.
	env.Detail("git pull --ff-only in " + env.Repo.Root)
	if err := env.Runner.Run(ctx, run.Command{
		Name: "git", Args: []string{"-C", env.Repo.Root, "pull", "--ff-only"},
	}); err != nil {
		return fmt.Errorf("the checkout could not be fast-forwarded — commit or stash your changes first: %w", err)
	}
	return nil
}

// rollbackStep restores the files a previous run replaced.
type rollbackStep struct{ base }

func newRollbackStep() installer.Step {
	return rollbackStep{base{
		id:     "rollback",
		title:  "Restore the last backup",
		detail: "Put back the files the most recent run replaced.",
	}}
}

func (s rollbackStep) Run(_ context.Context, env *installer.Env) error {
	backup, err := latestBackup()
	if err != nil {
		return err
	}
	if backup == "" {
		env.Note("There is no backup to restore. Backups are only taken when a run actually replaces something.")
		return nil
	}
	env.Detail("restoring from " + backup)

	restored := 0
	err = filepath.WalkDir(backup, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		// The backup mirrors the layout of what it replaced, so the path
		// inside it is the destination.
		rel, relErr := filepath.Rel(backup, path)
		if relErr != nil {
			return relErr
		}
		dest := string(filepath.Separator) + rel

		// Restoring is unrecorded: these files predate the install, so putting
		// them back must not make a later uninstall think it owns them.
		if err := env.Home.Unrecorded().CopyFile(path, dest); err != nil {
			return err
		}
		restored++
		return nil
	})
	if err != nil {
		return err
	}

	env.Detail(fmt.Sprintf("%d restored", restored))
	env.NoteApplied(fmt.Sprintf("%s restored from %s. The backup itself was left in place.",
		plural(restored, "file was", "files were"), backup))
	return nil
}

// latestBackup returns the newest backup directory, or "" if there is none.
func latestBackup() (string, error) {
	base := filepath.Join(stateHome(), "ilmango")
	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("looking for backups in %s: %w", base, err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "backup-") {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		return "", nil
	}
	// The names are timestamps in a sortable format, so the last one is the
	// most recent — no need to stat anything.
	sort.Strings(names)
	return filepath.Join(base, names[len(names)-1]), nil
}

// changesStep reports which installed files the user has since edited.
//
// This is what the manifest's checksums were for: an installed file whose
// content no longer matches what was written is one the user changed, and
// those are exactly the files an uninstall refuses to delete.
type changesStep struct{ base }

func newChangesStep() installer.Step {
	return changesStep{base{
		id:     "changes",
		title:  "Report local changes",
		detail: "List installed files whose contents no longer match the install.",
	}}
}

// ReadOnly marks this step as an inspection.
func (changesStep) ReadOnly() bool { return true }

func (s changesStep) Run(_ context.Context, env *installer.Env) error {
	if env.Manifest == nil || len(env.Manifest.Entries) == 0 {
		env.Note("There is no installation record to compare against. Install first, then this can tell you what you have changed.")
		return nil
	}

	var changed, missing []string
	for _, entry := range env.Manifest.Entries {
		if entry.Sum == "" {
			continue // symlinks and directories carry no checksum
		}
		sum, err := fsx.SumFile(entry.Path)
		switch {
		case os.IsNotExist(err):
			missing = append(missing, entry.Path)
		case err != nil:
			continue
		case sum != entry.Sum:
			changed = append(changed, entry.Path)
		}
	}

	switch {
	case len(changed) == 0 && len(missing) == 0:
		env.Note(fmt.Sprintf("All %d installed files are untouched.", len(env.Manifest.Entries)))
	default:
		env.Detail(fmt.Sprintf("%d changed, %d missing, out of %d", len(changed), len(missing), len(env.Manifest.Entries)))
	}

	if len(changed) > 0 {
		sort.Strings(changed)
		env.Note(fmt.Sprintf("You have edited %s; an uninstall will leave %s alone:\n  %s",
			plural(len(changed), "installed file", "installed files"),
			pronoun(len(changed)), strings.Join(changed, "\n  ")))
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		env.Note(fmt.Sprintf("%s gone. Reinstalling puts %s back.",
			plural(len(missing), "installed file is", "installed files are"), pronoun(len(missing))))
	}
	return nil
}

// plural renders a count with the wording that matches it.
func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

// pronoun matches the count too, so a sentence about one file does not talk
// about "them".
func pronoun(n int) string {
	if n == 1 {
		return "it"
	}
	return "them"
}
