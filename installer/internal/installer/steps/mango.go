package steps

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ilmango/internal/installer"
	"ilmango/internal/run"
)

// mango reads exactly one configuration file and merges nothing, so writing
// our own would throw away whatever window management the user already has.
//
// Instead the shell's keybinds and autostart go into a file of their own and
// are pulled in with a single source-optional line, which mango ignores if
// that file ever disappears. Nothing the user wrote is touched.
const (
	mangoSourceMarker = "mango/ilmango.conf"
	mangoSourceNote   = "\n# Added by Illogical-mango: shell keybinds and autostart.\nsource-optional=%s\n"
)

type mangoStep struct{ base }

func newMangoStep() installer.Step {
	return mangoStep{base{
		id:     "mango",
		title:  "Hook into the mango config",
		detail: "Add the shell's keybinds without touching your window management.",
		applies: func(c installer.Config) bool {
			return c.Effective(installer.OptMango)
		},
	}}
}

func (s mangoStep) Run(ctx context.Context, env *installer.Env) error {
	source := filepath.Join(env.Repo.Payload, "defaults", "mango", "config.conf")
	if _, err := os.Stat(source); err != nil {
		env.Log("no defaults/mango/config.conf in this checkout; nothing to hook in")
		return nil
	}
	if !run.Exists("mango") {
		env.Note("mango is not installed, so its config was left alone. Install mangowm and run the installer again to add the shell's keybinds.")
		return nil
	}

	dir := filepath.Join(configHome(), "mango")
	ourConfig := filepath.Join(dir, "ilmango.conf")
	mainConfig := filepath.Join(dir, "config.conf")

	if err := env.Home.EnsureDir(dir, 0o755); err != nil {
		return err
	}
	if err := s.writeKeybinds(env, source, ourConfig); err != nil {
		return err
	}
	env.Detail("keybinds → " + ourConfig)

	// The main config belongs to the user even when we seed it: it holds their
	// window management. Every write to it goes through an unrecorded tree, so
	// it never lands in the manifest and an uninstall can never delete it.
	if err := s.ensureMainConfig(env, mainConfig); err != nil {
		return err
	}
	if err := s.ensureSourceLine(env, mainConfig, ourConfig); err != nil {
		return err
	}

	s.reload(ctx, env)
	return nil
}

// ensureMainConfig makes sure there is a user config to append to.
//
// mango falls back to the system file when the user has none, so the first
// install seeds a copy: appending to /etc would edit a file the package
// manager owns.
func (s mangoStep) ensureMainConfig(env *installer.Env, mainConfig string) error {
	if env.Home.Exists(mainConfig) {
		return nil
	}

	const systemConfig = "/etc/mango/config.conf"
	if data, err := os.ReadFile(systemConfig); err == nil {
		env.Log("seeding " + mainConfig + " from " + systemConfig)
		return env.Home.Unrecorded().WriteFile(mainConfig, data, 0o644)
	}

	env.Log("creating an empty " + mainConfig)
	return env.Home.Unrecorded().WriteFile(mainConfig, nil, 0o644)
}

// ensureSourceLine appends the include exactly once.
func (s mangoStep) ensureSourceLine(env *installer.Env, mainConfig, ourConfig string) error {
	current, err := env.Home.ReadFile(mainConfig)
	if err != nil {
		return fmt.Errorf("reading %s: %w", mainConfig, err)
	}

	if strings.Contains(string(current), mangoSourceMarker) {
		env.Log("mango already sources the Illogical-mango config")
		return nil
	}

	updated := append(current, []byte(fmt.Sprintf(mangoSourceNote, ourConfig))...)
	if err := env.Home.Unrecorded().WriteFile(mainConfig, updated, 0o644); err != nil {
		return err
	}

	env.Log("mango now sources " + ourConfig)
	env.NoteApplied("Your mango config now has one added line sourcing ~/.config/mango/ilmango.conf. Your window management was not touched.")
	return nil
}

// reload asks a running mango to pick the change up, so the keybinds work
// without logging out. A failure here costs nothing: the next start reads it.
func (s mangoStep) reload(ctx context.Context, env *installer.Env) {
	if !run.Exists("mmsg") || os.Getenv("MANGO_INSTANCE_SIGNATURE") == "" {
		return
	}
	if err := env.Runner.Run(ctx, run.Command{Name: "mmsg", Args: []string{"dispatch", "reload_config"}}); err != nil {
		env.Log("could not reload mango: " + err.Error())
	}
}

// writeKeybinds assembles the file mango sources.
//
// The shell's own bindings are always written: without them the panels have no
// keys and the install is decorative. The conventional desktop set is appended
// only on request, because that half defines window management and can only do
// so by overriding whatever the user bound first.
func (s mangoStep) writeKeybinds(env *installer.Env, source, dest string) error {
	body, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("reading %s: %w", source, err)
	}

	if env.Config.Choice(installer.OptKeybinds) != installer.KeybindsFull {
		return env.Home.WriteFile(dest, body, 0o644)
	}

	extra := filepath.Join(filepath.Dir(source), "keybinds-full.conf")
	more, err := os.ReadFile(extra)
	if err != nil {
		env.Note("The conventional keybinds were requested but " + extra + " is missing from this checkout, so only the shell's own keys were written.")
		return env.Home.WriteFile(dest, body, 0o644)
	}

	env.Detail("including the conventional desktop keybinds")
	combined := append(append(body, '\n'), more...)
	return env.Home.WriteFile(dest, combined, 0o644)
}
