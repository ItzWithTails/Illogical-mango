package steps

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"ilmango/internal/installer"
	"ilmango/internal/run"
)

// serviceStep installs the systemd user unit.
//
// Under mango, autostart comes from the exec-once line in the compositor
// config rather than from systemd, so this unit exists for the manual
// `ilmango service restart` path and for compositors that do use it. Optional
// accordingly.
type serviceStep struct{ base }

func newServiceStep() installer.Step {
	return serviceStep{base{
		id:     "user-service",
		title:  "Install the user service",
		detail: "A systemd user unit for restarting the shell on demand.",
		applies: func(c installer.Config) bool {
			return c.Effective(installer.OptConfigFiles)
		},
	}}
}

// Optional marks the unit as a convenience.
func (serviceStep) Optional() bool { return true }

func (s serviceStep) Run(ctx context.Context, env *installer.Env) error {
	source := filepath.Join(env.Repo.Payload, "assets", "systemd", "ilmango.service")
	data, err := os.ReadFile(source)
	if err != nil {
		env.Log("no service unit in this checkout; skipping")
		return nil
	}
	if !run.Exists("systemctl") {
		env.Log("systemd is not in use here; skipping the unit")
		return nil
	}

	// The shipped unit assumes a system-wide launcher; point it at ours.
	unit := strings.ReplaceAll(string(data), "/usr/local/bin/ilmango", LauncherPath())
	unit = strings.ReplaceAll(unit, "/usr/bin/ilmango", LauncherPath())

	target := filepath.Join(configHome(), "systemd", "user", "ilmango.service")
	if err := env.Home.EnsureDir(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if err := env.Home.WriteFile(target, []byte(unit), 0o644); err != nil {
		return err
	}

	env.Detail(target)
	return env.Runner.Run(ctx, run.Command{Name: "systemctl", Args: []string{"--user", "daemon-reload"}})
}

// cursorTheme is the pointer the shell's look assumes. It has to be published
// twice, to two audiences that do not talk to each other: gsettings for GTK
// applications, and the XCursor default theme for the compositor that draws the
// pointer itself.
const (
	cursorTheme = "capitaine-cursors-light"
	cursorSize  = "24"
)

// desktopSettingsStep applies the GTK theme, cursor and font that the shell's
// look assumes, for applications that read them from gsettings.
//
// Optional: without it GTK apps simply keep their previous appearance.
type desktopSettingsStep struct{ base }

func newDesktopSettingsStep() installer.Step {
	return desktopSettingsStep{base{
		id:     "desktop-settings",
		title:  "Apply desktop settings",
		detail: "GTK theme, cursor and interface font for non-Qt applications.",
		applies: func(c installer.Config) bool {
			return c.Effective(installer.OptConfigFiles) && c.Effective(installer.OptFonts)
		},
	}}
}

// Optional marks appearance defaults as survivable.
func (desktopSettingsStep) Optional() bool { return true }

// interfaceSettings are applied in order; each is independent.
var interfaceSettings = [][2]string{
	{"color-scheme", "prefer-dark"},
	{"gtk-theme", "adw-gtk3-dark"},
	{"icon-theme", "Papirus-Dark"},
	{"cursor-theme", cursorTheme},
	{"cursor-size", cursorSize},
	{"font-name", "Rubik 11"},
}

func (s desktopSettingsStep) Run(ctx context.Context, env *installer.Env) error {
	if err := s.writeCursorDefault(env); err != nil {
		return err
	}

	if !run.Exists("gsettings") {
		env.Log("gsettings is not available; GTK applications keep their current look")
		return nil
	}

	for _, setting := range interfaceSettings {
		cmd := run.Command{
			Name: "gsettings",
			Args: []string{"set", "org.gnome.desktop.interface", setting[0], setting[1]},
		}
		if err := env.Runner.Run(ctx, cmd); err != nil {
			// A schema this desktop does not have is not worth failing over.
			env.Log("could not set " + setting[0] + ": " + err.Error())
		}
	}
	return nil
}

// writeCursorDefault points the XCursor "default" theme at the shell's cursor.
//
// gsettings is not enough on its own. It reaches GTK applications, but the
// pointer you actually move is drawn by the compositor, and a wlroots
// compositor never reads gsettings: it resolves the theme named by
// XCURSOR_THEME, or failing that the one called "default". Left alone, that
// resolves to the system default — on Arch, Adwaita, whose arrow is filled
// black. On this shell's dark background a black pointer reads as a hollow
// outline, which looks like a rendering fault rather than a theme.
func (s desktopSettingsStep) writeCursorDefault(env *installer.Env) error {
	dir := filepath.Join(dataHome(), "icons", "default")
	if err := env.Home.EnsureDir(dir, 0o755); err != nil {
		return err
	}

	// Inherits, rather than a copy of the theme: the pointer follows whatever
	// capitaine-cursors-light ships, and one line is something a user can read
	// and undo.
	body := "[Icon Theme]\nName=default\nComment=Set by Illogical-mango\nInherits=" + cursorTheme + "\n"
	return env.Home.WriteFile(filepath.Join(dir, "index.theme"), []byte(body), 0o644)
}

// stateDirsStep creates the directories the shell writes its generated theme
// and runtime state into. Quickshell expects them to exist before first start.
type stateDirsStep struct{ base }

func newStateDirsStep() installer.Step {
	return stateDirsStep{base{
		id:     "state-dirs",
		title:  "Prepare state directories",
		detail: "Create the directories the shell generates its theme into.",
		applies: func(c installer.Config) bool {
			return c.Effective(installer.OptConfigFiles)
		},
	}}
}

func (s stateDirsStep) Run(_ context.Context, env *installer.Env) error {
	base := filepath.Join(stateHome(), "quickshell", "user", "generated")

	for _, dir := range []string{base, filepath.Join(base, "wallpaper"), filepath.Join(base, "terminal")} {
		if err := env.Home.EnsureDir(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}
