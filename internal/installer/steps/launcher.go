package steps

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ilmango/internal/installer"
)

// binHome is where a user's own executables belong.
func binHome() string {
	if dir := os.Getenv("XDG_BIN_HOME"); dir != "" {
		return dir
	}
	return filepath.Join(home(), ".local", "bin")
}

// LauncherPath is the ilmango command's home. Desktop entries point at it, so it
// is shared rather than recomputed.
func LauncherPath() string { return filepath.Join(binHome(), "ilmango") }

// launcherStep puts the ilmango command on the user's PATH.
//
// This is not a convenience: every entry point the documentation offers —
// ilmango run, ilmango settings, ilmango doctor, ilmango update — is this one script. An
// installation without it leaves the user with no way to start what they just
// installed, so the step is required rather than optional.
type launcherStep struct{ base }

func newLauncherStep() installer.Step {
	return launcherStep{base{
		id:     "launcher",
		title:  "Install the ilmango command",
		detail: "Put the ilmango launcher on your PATH.",
		applies: func(c installer.Config) bool {
			return c.Effective(installer.OptConfigFiles)
		},
	}}
}

func (s launcherStep) Run(_ context.Context, env *installer.Env) error {
	source := filepath.Join(env.Repo.Root, "scripts", "ilmango")
	if _, err := os.Stat(source); err != nil {
		return fmt.Errorf("the launcher is missing from this checkout: %w", err)
	}

	target := LauncherPath()
	if err := env.Home.EnsureDir(binHome(), 0o755); err != nil {
		return err
	}
	if err := env.Home.CopyFile(source, target); err != nil {
		return err
	}
	if err := env.Home.Chmod(target, 0o755); err != nil {
		return err
	}

	env.Detail(target)
	s.warnIfOffPath(env, target)
	return nil
}

// warnIfOffPath tells the user when the command exists but they cannot run it,
// which is otherwise a baffling failure the moment they follow the README.
func (s launcherStep) warnIfOffPath(env *installer.Env, target string) {
	dir := filepath.Dir(target)
	for _, entry := range filepath.SplitList(os.Getenv("PATH")) {
		if entry == dir {
			return
		}
	}

	env.Note(fmt.Sprintf(
		"%s is not on your PATH, so the ilmango command will not be found. Add it with: fish_add_path %s  (or the equivalent for your shell).",
		dir, dir))
}

// desktopStep adds the application entries and the icon, so Illogical-mango appears in
// launchers and menus.
//
// It is optional: a missing desktop database or icon cache costs a menu entry,
// not a working shell.
type desktopStep struct{ base }

func newDesktopStep() installer.Step {
	return desktopStep{base{
		id:     "desktop-entries",
		title:  "Add application entries",
		detail: "Desktop entries and the icon, so Illogical-mango shows up in launchers.",
		applies: func(c installer.Config) bool {
			return c.Effective(installer.OptConfigFiles)
		},
	}}
}

// Optional marks menu integration as survivable.
func (desktopStep) Optional() bool { return true }

// desktopEntries maps each source entry to the command its Exec line must run.
var desktopEntries = map[string]string{
	"ilmango.desktop":          "service restart",
	"ilmango-settings.desktop": "settings",
}

func (s desktopStep) Run(_ context.Context, env *installer.Env) error {
	appsDir := filepath.Join(dataHome(), "applications")
	if err := env.Home.EnsureDir(appsDir, 0o755); err != nil {
		return err
	}

	for name, command := range desktopEntries {
		source := filepath.Join(env.Repo.Root, "assets", "applications", name)
		data, err := os.ReadFile(source)
		if err != nil {
			env.Log("no " + name + " in this checkout; skipping")
			continue
		}

		// The shipped Exec assumes a system-wide install; point it at the
		// launcher this run actually placed.
		rewritten := rewriteExec(string(data), LauncherPath()+" "+command)
		if err := env.Home.WriteFile(filepath.Join(appsDir, name), []byte(rewritten), 0o644); err != nil {
			return err
		}
	}

	return s.installIcon(env)
}

func (s desktopStep) installIcon(env *installer.Env) error {
	source := filepath.Join(env.Repo.Root, "assets", "icons", "desktop-symbolic.svg")
	if _, err := os.Stat(source); err != nil {
		return nil
	}

	target := filepath.Join(dataHome(), "icons", "hicolor", "scalable", "apps", "ilmango.svg")
	if err := env.Home.EnsureDir(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return env.Home.CopyFile(source, target)
}

// rewriteExec replaces the Exec line of a desktop entry.
func rewriteExec(content, command string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "Exec=") {
			lines[i] = "Exec=" + command
		}
	}
	return strings.Join(lines, "\n")
}
