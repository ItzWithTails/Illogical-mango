package steps

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ilmango/internal/fsx"
	"ilmango/internal/installer"
	"ilmango/internal/pkg"
	"ilmango/internal/run"
)

// ErrNotInstalled is returned when no manifest exists to uninstall from.
var ErrNotInstalled = errors.New("no installation record found")

// stopServicesStep shuts down what the installation started, before the files
// defining those services are removed.
type stopServicesStep struct{ base }

func newStopServicesStep() installer.Step {
	return stopServicesStep{base{
		id:     "stop-services",
		title:  "Stop Illogical-mango services",
		detail: "Disable the user services this installation enabled.",
	}}
}

func (s stopServicesStep) Run(ctx context.Context, env *installer.Env) error {
	if !run.Exists("systemctl") {
		env.Log("systemd is not in use here; nothing to stop")
		return nil
	}

	for _, service := range append([]string{"ilmango"}, userServices...) {
		cmd := run.Command{Name: "systemctl", Args: []string{"--user", "disable", "--now", service}}
		if err := env.Runner.Run(ctx, cmd); err != nil {
			// A unit that was never enabled is the normal case, not a failure.
			env.Log("skipped " + service + ": " + err.Error())
		}
	}

	if err := env.Runner.Run(ctx, run.Command{Name: "systemctl", Args: []string{"--user", "daemon-reload"}}); err != nil {
		env.Log("could not reload the user systemd manager: " + err.Error())
	}
	return nil
}

// removeFilesStep deletes exactly what the installation wrote.
//
// It removes a file only while its content still matches what was installed.
// Anything the user has edited since is left in place and reported: losing
// someone's customisations is a far worse outcome than leaving a file behind.
type removeFilesStep struct{ base }

func newRemoveFilesStep() installer.Step {
	return removeFilesStep{base{
		id:     "remove-files",
		title:  "Remove installed files",
		detail: "Delete the files this installation created, keeping your edits.",
	}}
}

func (s removeFilesStep) Run(ctx context.Context, env *installer.Env) error {
	manifest := env.Manifest
	if manifest == nil || manifest.Len() == 0 {
		return fmt.Errorf("%w at %s", ErrNotInstalled, ManifestPath())
	}

	entries := manifest.RemovalOrder()
	env.Detail(fmt.Sprintf("%d paths", len(entries)))

	var removed, kept, absent int
	var modified []string
	dirs := map[string]bool{}

	for i, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if i%200 == 0 {
			env.Detail(fmt.Sprintf("%d of %d", i, len(entries)))
		}

		outcome, err := env.Home.Remove(entry.Path, entry.Sum)
		if err != nil {
			return err
		}

		switch outcome {
		case fsx.Removed:
			removed++
			dirs[filepath.Dir(entry.Path)] = true
		case fsx.Modified:
			kept++
			modified = append(modified, entry.Path)
		default:
			absent++
		}
	}

	// Tidy the directory skeleton, but never climb past the home directory.
	for dir := range dirs {
		env.Home.PruneEmptyDirs(dir, home())
	}

	env.Log(fmt.Sprintf("removed %d, kept %d, already gone %d", removed, kept, absent))
	s.reportKept(env, modified)
	return nil
}

// reportKept tells the user precisely which of their files survived and why.
func (s removeFilesStep) reportKept(env *installer.Env, modified []string) {
	if len(modified) == 0 {
		return
	}

	const listed = 5
	shown := modified
	suffix := ""
	if len(shown) > listed {
		shown, suffix = shown[:listed], fmt.Sprintf(" and %d more", len(modified)-listed)
	}

	env.Note(fmt.Sprintf(
		"%d files were changed after installation and were kept: %s%s. Delete them yourself if you no longer want them.",
		len(modified), strings.Join(shown, ", "), suffix))
}

// reportPackagesStep lists the packages an installation pulled in.
//
// It never removes them. Dependencies are shared: another program may well
// need the same Qt libraries, and an uninstaller that takes 400 packages with
// it can break a working system in ways that are tedious to undo.
type reportPackagesStep struct{ base }

func newReportPackagesStep() installer.Step {
	return reportPackagesStep{base{
		id:     "report-packages",
		title:  "Report installed packages",
		detail: "List what was installed as a dependency, without removing any.",
	}}
}

// ReadOnly marks this step as an inspection.
func (reportPackagesStep) ReadOnly() bool { return true }

func (s reportPackagesStep) Run(ctx context.Context, env *installer.Env) error {
	family := string(env.Distro.Family)
	if !pkg.KnownFamily(family) {
		return nil
	}

	manager, err := pkg.FindManager(family)
	if err != nil {
		return nil
	}

	installed, err := manager.InstalledSet(ctx, env.Runner)
	if err != nil {
		return nil
	}

	var present []string
	for _, name := range pkg.Packages(family, pkg.GroupCore, pkg.GroupQuickshell,
		pkg.GroupAudio, pkg.GroupToolkit, pkg.GroupScreenCapture, pkg.GroupFonts) {
		if installed[name] {
			present = append(present, name)
		}
	}
	if len(present) == 0 {
		return nil
	}

	// The list belongs in the transcript, not in a note nobody can copy from.
	env.Log("packages installed for Illogical-mango that are still present:")
	env.Log(strings.Join(present, " "))
	env.Note(fmt.Sprintf(
		"%d packages were left installed, because other software may need them. The transcript lists them if you want to review the list yourself.",
		len(present)))
	return nil
}

// forgetStep discards the installation's own bookkeeping, last of all.
type forgetStep struct{ base }

func newForgetStep() installer.Step {
	return forgetStep{base{
		id:     "forget",
		title:  "Discard the installation record",
		detail: "Remove version.json and the file manifest.",
	}}
}

func (s forgetStep) Run(_ context.Context, env *installer.Env) error {
	for _, path := range []string{
		filepath.Join(configHome(), "ilmango", "version.json"),
		ManifestPath(),
	} {
		if _, err := env.Home.Remove(path, ""); err != nil {
			return err
		}
	}

	env.Home.PruneEmptyDirs(filepath.Join(configHome(), "ilmango"), home())
	return nil
}

// unhookMangoStep removes the single line the install added to the user's
// compositor config, and nothing else.
//
// The config itself is never deleted: the installer only ever appended to it,
// and everything around that line is the user's own window management.
type unhookMangoStep struct{ base }

func newUnhookMangoStep() installer.Step {
	return unhookMangoStep{base{
		id:     "unhook-mango",
		title:  "Unhook from the mango config",
		detail: "Remove the one line the installer added, leaving the rest alone.",
	}}
}

// Optional keeps a hand-edited compositor config from blocking the removal.
func (unhookMangoStep) Optional() bool { return true }

func (s unhookMangoStep) Run(_ context.Context, env *installer.Env) error {
	mainConfig := filepath.Join(configHome(), "mango", "config.conf")
	if !env.Home.Exists(mainConfig) {
		return nil
	}

	current, err := env.Home.ReadFile(mainConfig)
	if err != nil {
		return fmt.Errorf("reading %s: %w", mainConfig, err)
	}
	if !strings.Contains(string(current), mangoSourceMarker) {
		return nil
	}

	cleaned, removed := stripMangoHook(string(current))
	if removed == 0 {
		return nil
	}

	if err := env.Home.Unrecorded().WriteFile(mainConfig, []byte(cleaned), 0o644); err != nil {
		return err
	}

	env.Log(fmt.Sprintf("removed %d lines from %s", removed, mainConfig))
	env.Detail("unhooked")
	return nil
}

// stripMangoHook drops the source line and the comment the installer wrote
// above it, matching on exactly what was added rather than on anything near it.
func stripMangoHook(content string) (string, int) {
	lines := strings.Split(content, "\n")
	kept := make([]string, 0, len(lines))

	removed := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		isOurs := strings.HasPrefix(trimmed, "source-optional=") && strings.Contains(trimmed, mangoSourceMarker)
		isOurComment := trimmed == "# Added by Illogical-mango: shell keybinds and autostart."

		if isOurs || isOurComment {
			removed++
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n"), removed
}

// removeVenvStep discards the Python environment.
//
// It is handled separately because uv creates it outside the filesystem
// gateway, so it never appears in the manifest that drives the rest of the
// removal. Nothing the user authored lives there — it is regenerated from
// requirements.txt — so it is removed wholesale.
type removeVenvStep struct{ base }

func newRemoveVenvStep() installer.Step {
	return removeVenvStep{base{
		id:     "remove-venv",
		title:  "Remove the Python environment",
		detail: "Delete the virtualenv the shell's Python helpers used.",
	}}
}

// Optional keeps a stubborn virtualenv from blocking the rest of a removal.
func (removeVenvStep) Optional() bool { return true }

func (s removeVenvStep) Run(_ context.Context, env *installer.Env) error {
	dir := venvPath()
	if !env.Home.Exists(dir) {
		env.Log("no Python environment to remove")
		return nil
	}

	env.Detail(dir)
	return env.Home.RemoveAll(dir)
}

// FindManifest loads the installation record, reporting a clear error when
// there is nothing installed to remove.
func FindManifest() (*installer.Manifest, error) { return FindManifestUnder("") }

// FindManifestUnder loads the record from beneath a redirected root, which is
// how a test installation is examined without touching the real home.
func FindManifestUnder(root string) (*installer.Manifest, error) {
	path := ManifestPath()
	if root != "" {
		path = filepath.Join(root, path)
	}

	manifest, err := installer.LoadManifest(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: expected %s", ErrNotInstalled, path)
		}
		return nil, err
	}
	return manifest, nil
}
