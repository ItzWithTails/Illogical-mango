package steps

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
	mangoSourceNote   = "# Added by Illogical-mango: shell keybinds and autostart.\n" +
		"# It goes first because mango stops at the first bind that matches a\n" +
		"# combination (see keybinding() in mango.c), so an include at the end\n" +
		"# loses every collision with the lines above it.\n" +
		"source-optional=%s\n\n"
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
	if err := s.repairRejectedKeys(env, mainConfig); err != nil {
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

// ensureSourceLine puts the include at the top of the config, exactly once.
//
// The top, not the bottom. mango walks its bindings in order and stops at the
// first one whose modifiers and key match, so whichever include is read first
// wins every conflict. Appended, the shell's own keybinds silently lose to the
// defaults a fresh config is seeded with — Super+M among them, which in those
// defaults quits the compositor.
//
// An include that is already present but sitting below the binds is moved.
// Checking only for its presence was not enough: a config written by an older
// version keeps the bad position for ever, because every later run sees the
// line, considers the job done, and leaves it where it is. That is a
// reinstall that changes nothing and explains nothing.
func (s mangoStep) ensureSourceLine(env *installer.Env, mainConfig, ourConfig string) error {
	current, err := env.Home.ReadFile(mainConfig)
	if err != nil {
		return fmt.Errorf("reading %s: %w", mainConfig, err)
	}

	updated, action := placeSourceLine(string(current), fmt.Sprintf(mangoSourceNote, ourConfig))
	if action == sourceLineUnchanged {
		env.Log("mango already sources the Illogical-mango config, first")
		return nil
	}

	if err := env.Home.Unrecorded().WriteFile(mainConfig, []byte(updated), 0o644); err != nil {
		return err
	}

	if action == sourceLineMoved {
		env.Log("moved the include above the binds it has to override")
		env.NoteApplied("The line sourcing ~/.config/mango/ilmango.conf was below your keybinds, so the shell's own keys lost every collision with them — Super+M reached the default underneath and quit the compositor. It has been moved to the top of the file; nothing else changed.")
		return nil
	}

	env.Log("mango now sources " + ourConfig)
	env.NoteApplied("Your mango config now starts with one added line sourcing ~/.config/mango/ilmango.conf. Nothing else in it was changed.")
	return nil
}

// What placeSourceLine did.
type sourceLineAction int

const (
	sourceLineUnchanged sourceLineAction = iota
	sourceLineAdded
	sourceLineMoved
)

// placeSourceLine returns the config with our include first.
//
// The comment block above the line travels with it, so moving does not leave
// an explanation stranded in the middle of the file.
func placeSourceLine(content, note string) (string, sourceLineAction) {
	lines := strings.Split(content, "\n")

	at := -1
	firstBind := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if at < 0 && strings.Contains(trimmed, mangoSourceMarker) && strings.HasPrefix(trimmed, "source") {
			at = i
		}
		if firstBind < 0 && strings.HasPrefix(trimmed, "bind") {
			firstBind = i
		}
	}

	switch {
	case at < 0:
		return note + content, sourceLineAdded
	case firstBind < 0 || at < firstBind:
		// Already ahead of everything it needs to beat.
		return content, sourceLineUnchanged
	}

	// Take the line and the comment block immediately above it.
	from := at
	for from > 0 {
		above := strings.TrimSpace(lines[from-1])
		if above == "" || !strings.HasPrefix(above, "#") {
			break
		}
		from--
	}
	rest := append(append([]string{}, lines[:from]...), lines[at+1:]...)
	return note + strings.Join(rest, "\n"), sourceLineMoved
}

// reload asks a running mango to pick the change up, so the keybinds work
// without logging out.
//
// This is not a nicety. A config written and never applied is the exact shape
// of "the keybinds died again": the file on disk is right, the compositor is
// still running on what it read at startup, and nothing says so.
func (s mangoStep) reload(ctx context.Context, env *installer.Env) {
	if !run.Exists("mmsg") {
		return
	}

	signature := mangoSignature()
	if signature == "" {
		// No mango running for this user, so there is nothing to tell.
		return
	}

	cmd := run.Command{
		Name: "mmsg", Args: []string{"dispatch", "reload_config"},
		Env: []string{"MANGO_INSTANCE_SIGNATURE=" + signature},
	}
	if err := env.Runner.Run(ctx, cmd); err != nil {
		env.Log("could not reload mango: " + err.Error())
		env.Note("mango is running but would not reload, so the keybinds take effect at your next login.")
		return
	}
	env.Detail("reloaded the running mango")
}

// mangoSignature finds the socket of a mango running for this user.
//
// The environment variable is set for anything mango spawned, which covers a
// terminal inside the session — but not an installer run over ssh, from
// another VT, or by a service. Those are exactly the runs that would otherwise
// write a config and leave the compositor on the old one, so the socket is
// looked for directly when the variable is absent.
func mangoSignature() string {
	if signature := os.Getenv("MANGO_INSTANCE_SIGNATURE"); signature != "" {
		return signature
	}

	matches, err := filepath.Glob(filepath.Join(runtimeDir(), "mango-*.sock"))
	if err != nil || len(matches) != 1 {
		// None means nothing to reload; more than one means more than one
		// session, and reloading an arbitrary one is worse than reloading none.
		return ""
	}
	return matches[0]
}

// runtimeDir is where compositors put their sockets.
func runtimeDir() string {
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		return dir
	}
	return filepath.Join("/run/user", strconv.Itoa(os.Getuid()))
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
		return env.Home.WriteFile(dest, resolveLauncher(body), 0o644)
	}

	extra := filepath.Join(filepath.Dir(source), "keybinds-full.conf")
	more, err := os.ReadFile(extra)
	if err != nil {
		env.Note("The conventional keybinds were requested but " + extra + " is missing from this checkout, so only the shell's own keys were written.")
		return env.Home.WriteFile(dest, resolveLauncher(body), 0o644)
	}

	env.Detail("including the conventional desktop keybinds")
	combined := append(append(body, '\n'), more...)
	return env.Home.WriteFile(dest, resolveLauncher(combined), 0o644)
}

// resolveLauncher writes the launcher's location into the config instead of
// trusting the compositor's PATH.
//
// mango spawns from its own environment, and a session started by a display
// manager or a tty login has never sourced a shell profile: its PATH is the
// system one, without ~/.local/bin. A bind that says "ilmango" therefore finds
// nothing and fails silently, and the exec-once that starts the shell fails
// the same way — the shell never appears and none of its keys work, which is
// indistinguishable from the shell being broken.
//
// The location is written with $HOME rather than expanded, so the file stays
// readable and is not tied to one user's home. That is safe because mango
// expands both forms: spawn runs its argument through wordexp(), and exec-once
// through sh -c.
func resolveLauncher(config []byte) []byte {
	launcher := LauncherPath()
	if rest, under := strings.CutPrefix(launcher, home()+string(filepath.Separator)); under {
		launcher = "$HOME/" + rest
	}

	body := string(config)
	// The two places a config names the launcher: a keybind spawns it, and
	// autostart runs it.
	body = strings.ReplaceAll(body, "spawn,ilmango ", "spawn,"+launcher+" ")
	body = strings.ReplaceAll(body, "exec-once=ilmango ", "exec-once="+launcher+" ")
	return []byte(body)
}

// mangoRenamedKeys are settings mango's own example config writes under names
// its parser does not accept at the top level.
//
// Each of these is a field of a device rule — valid inside a devicerule= line,
// where the trackpad and the mouse are addressed separately — used in the
// example as though it were a global setting. mango answers each one with
// "Unknown keyword" and carries on, so the setting silently never applies.
//
// The names on the right are what the parser actually reads. The trackpad
// variants are the right ones because that is the section of the example these
// lines sit in.
var mangoRenamedKeys = map[string]string{
	"disable_while_typing":    "trackpad_disable_while_typing",
	"left_handed":             "trackpad_left_handed",
	"middle_button_emulation": "trackpad_middle_button_emulation",
}

// repairRejectedKeys corrects those names in the user's config.
//
// The installer seeds that config from mango's example, so it is the reason
// these lines are there at all. Correcting them is repairing a defect this
// installer introduced by copying, not redefining anything the user chose: the
// value on the right of the "=" is carried across untouched, and a line mango
// rejects outright has no behaviour to preserve.
func (s mangoStep) repairRejectedKeys(env *installer.Env, mainConfig string) error {
	current, err := env.Home.ReadFile(mainConfig)
	if err != nil {
		return fmt.Errorf("reading %s: %w", mainConfig, err)
	}

	fixed, corrected := repairRejectedKeys(string(current))
	if len(corrected) == 0 {
		return nil
	}

	if err := env.Home.Unrecorded().WriteFile(mainConfig, []byte(fixed), 0o644); err != nil {
		return err
	}

	env.Detail(fmt.Sprintf("corrected %d rejected settings", len(corrected)))
	env.NoteApplied(fmt.Sprintf(
		"%s used names mango does not accept, so those settings never applied: %s. They now use the names its parser reads; the values are unchanged.",
		mainConfig, strings.Join(corrected, ", ")))
	return nil
}

// repairRejectedKeys rewrites the assignments, and only those.
//
// The same words are legitimate inside a devicerule= line, which addresses one
// device rather than every trackpad, so only a line that begins with the name
// is touched.
func repairRejectedKeys(content string) (string, []string) {
	lines := strings.Split(content, "\n")
	var corrected []string

	for i, line := range lines {
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		replacement, renamed := mangoRenamedKeys[key]
		if !renamed {
			continue
		}
		lines[i] = indent + replacement + "=" + value
		corrected = append(corrected, key+" → "+replacement)
	}
	return strings.Join(lines, "\n"), corrected
}
