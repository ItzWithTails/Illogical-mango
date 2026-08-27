package steps

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"ilmango/internal/installer"

	"ilmango/internal/fsx"
)

func TestShippedMangoConfigsUseOnlyKeysMangoParses(t *testing.T) {
	// The env= lines this project shipped for years were never parsed. Nothing
	// failed; the settings simply did not happen. A directive that silently
	// does nothing is worth a test.
	known := map[string]bool{
		"bind": true, "bindl": true, "bindr": true, "bindp": true, "bindc": true,
		"binds": true, "mousebind": true, "axisbind": true, "switchbind": true,
		"gesturebind": true, "source": true, "source-optional": true,
		"exec": true, "exec-once": true,
		"cursor_theme": true, "cursor_size": true,
	}

	dir := filepath.Join("..", "..", "..", "..", "src", "defaults", "mango")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("templates not readable from here: %v", err)
	}

	for _, entry := range entries {
		body, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatalf("reading %s: %v", entry.Name(), err)
		}
		for n, line := range strings.Split(string(body), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			key, _, found := strings.Cut(line, "=")
			if !found {
				t.Errorf("%s:%d is not a key=value line: %q", entry.Name(), n+1, line)
				continue
			}
			if !known[key] {
				t.Errorf("%s:%d uses %q, which mango does not parse", entry.Name(), n+1, key)
			}
		}
	}
}

func TestMangoHookGoesFirstSoItsBindsWin(t *testing.T) {
	// mango stops at the first binding that matches, so an include appended at
	// the end loses every collision with the lines above it — including
	// Super+M, which the stock defaults bind to quitting the compositor.
	existing := strings.Join([]string{
		"# the user's own config",
		"bind=SUPER,m,quit",
		"bind=SUPER,r,reload_config",
	}, "\n")

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	main := filepath.Join(home, ".config", "mango", "config.conf")
	if err := os.MkdirAll(filepath.Dir(main), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(main, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	env := &installer.Env{Home: fsx.Tree{Mode: fsx.ModeApply}, Config: installer.NewConfig()}
	if err := (mangoStep{}).ensureSourceLine(env, main, "/home/u/.config/mango/ilmango.conf"); err != nil {
		t.Fatalf("ensureSourceLine() error = %v", err)
	}

	body, err := os.ReadFile(main)
	if err != nil {
		t.Fatal(err)
	}
	sourceAt := strings.Index(string(body), "source-optional=")
	quitAt := strings.Index(string(body), "bind=SUPER,m,quit")
	if sourceAt < 0 {
		t.Fatalf("the include was not written:\n%s", body)
	}
	if sourceAt > quitAt {
		t.Fatalf("the include is below the binds it must override:\n%s", body)
	}
	if !strings.Contains(string(body), "# the user's own config") {
		t.Error("the user's own config did not survive")
	}
}

func TestMangoHookIsWrittenOnlyOnce(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	main := filepath.Join(home, ".config", "mango", "config.conf")
	if err := os.MkdirAll(filepath.Dir(main), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(main, []byte("bind=SUPER,m,quit\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	env := &installer.Env{Home: fsx.Tree{Mode: fsx.ModeApply}, Config: installer.NewConfig()}
	for i := 0; i < 3; i++ {
		if err := (mangoStep{}).ensureSourceLine(env, main, "/home/u/.config/mango/ilmango.conf"); err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
	}

	body, _ := os.ReadFile(main)
	if n := strings.Count(string(body), "source-optional="); n != 1 {
		t.Fatalf("the include appears %d times after three runs", n)
	}
}

// bindKey identifies a binding the way mango's matcher does.
type bindKey struct{ kind, mods, key string }

func parseBinds(t *testing.T, name string) map[bindKey]string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "src", "defaults", "mango", name))
	if err != nil {
		t.Skipf("template not readable from here: %v", err)
	}

	out := map[bindKey]string{}
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "bind") && !strings.HasPrefix(line, "axisbind") {
			continue
		}
		kind, rest, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		parts := strings.Split(rest, ",")
		if len(parts) < 3 {
			continue
		}
		mods := strings.Split(strings.ToLower(parts[0]), "+")
		sort.Strings(mods)

		// mango compares keysyms through xkb_keysym_to_lower, so SUPER,L and
		// SUPER,l are one binding, not two.
		k := bindKey{kind: strings.TrimSuffix(kind, "l"), mods: strings.Join(mods, "+"),
			key: strings.ToLower(strings.TrimSpace(parts[1]))}
		out[k] = strings.Join(parts[2:], ",")
	}
	return out
}

func TestShippedBindsDoNotCollideWithEachOther(t *testing.T) {
	// A collision inside our own files is never intentional, and mango
	// resolves it silently by taking the first: the second binding simply does
	// nothing, which is indistinguishable from a broken key.
	base := parseBinds(t, "config.conf")
	full := parseBinds(t, "keybinds-full.conf")

	for key, action := range full {
		if other, clash := base[key]; clash {
			t.Errorf("%s+%s is bound twice in our own configs: %q and %q",
				key.mods, key.key, other, action)
		}
	}
}

func TestMangoConfigStartsTheClipboardWatchers(t *testing.T) {
	// Nothing records the clipboard unless something is asked to. The niri
	// configuration has started these two watchers for as long as it has
	// existed; the mango one never did, so the clipboard overlay opened onto
	// an empty database and every refresh failed.
	body, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "src", "defaults", "mango", "config.conf"))
	if err != nil {
		t.Skipf("template not readable from here: %v", err)
	}

	for _, want := range []string{
		"exec-once=wl-paste --type text --watch",
		"exec-once=wl-paste --type image --watch cliphist store",
	} {
		if !strings.Contains(string(body), want) {
			t.Errorf("the mango config does not start %q", want)
		}
	}
}

func TestRejectedKeysAreCorrectedWithTheirValuesKept(t *testing.T) {
	// mango answers each of these with "Unknown keyword" and carries on, so
	// the setting never applies. The value is the user's; only the name is
	// wrong.
	config := strings.Join([]string{
		"# Trackpad",
		"tap_to_click=1",
		"disable_while_typing=1",
		"left_handed=0",
		"  middle_button_emulation=0",
	}, "\n")

	fixed, corrected := repairRejectedKeys(config)

	if len(corrected) != 3 {
		t.Fatalf("corrected %d keys, want 3: %v", len(corrected), corrected)
	}
	for _, want := range []string{
		"trackpad_disable_while_typing=1",
		"trackpad_left_handed=0",
		"  trackpad_middle_button_emulation=0",
	} {
		if !strings.Contains(fixed, want) {
			t.Errorf("missing %q in:\n%s", want, fixed)
		}
	}
	if !strings.Contains(fixed, "tap_to_click=1") {
		t.Error("a key the parser accepts was rewritten")
	}
}

func TestRejectedKeysInsideADeviceRuleAreLeftAlone(t *testing.T) {
	// The same words are legitimate inside devicerule=, which addresses one
	// device rather than every trackpad. Rewriting there would break a rule
	// that works.
	config := "devicerule=name:foo,disable_while_typing:1,left_handed:0\n"

	fixed, corrected := repairRejectedKeys(config)

	if len(corrected) != 0 {
		t.Fatalf("a device rule was rewritten: %v", corrected)
	}
	if fixed != config {
		t.Fatalf("the device rule changed:\n%s", fixed)
	}
}

func TestRepairingRejectedKeysIsIdempotent(t *testing.T) {
	config := "disable_while_typing=1\n"

	once, first := repairRejectedKeys(config)
	twice, second := repairRejectedKeys(once)

	if len(first) != 1 {
		t.Fatalf("first pass corrected %v, want one key", first)
	}
	if len(second) != 0 {
		t.Fatalf("a second pass corrected %v; there was nothing left to do", second)
	}
	if once != twice {
		t.Fatalf("a second pass changed the file again:\n%s", twice)
	}
}

func TestConfigNamesTheLauncherPortablyRatherThanTrustingPATH(t *testing.T) {
	// mango spawns from its own environment. A session started by a display
	// manager or a tty login has never sourced a profile, so "ilmango" resolves
	// to nothing: the keybinds do nothing and the shell never starts.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))

	config := strings.Join([]string{
		"exec-once=ilmango run --daemon",
		"bind=SUPER,t,spawn,ilmango terminal",
		"bindl=NONE,XF86AudioMute,spawn,ilmango audio mute",
		"bind=SUPER,f,togglefullscreen,",
	}, "\n")

	got := string(resolveLauncher([]byte(config)))

	// $HOME, not the expanded path: mango runs spawn through wordexp() and
	// exec-once through sh -c, so both forms work, and this one is readable and
	// not tied to one user's home.
	if strings.Contains(got, home) {
		t.Errorf("the config hardcodes a home directory:\n%s", got)
	}
	for _, want := range []string{
		"exec-once=$HOME/.local/bin/ilmango run --daemon",
		"spawn,$HOME/.local/bin/ilmango terminal",
		"spawn,$HOME/.local/bin/ilmango audio mute",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "spawn,ilmango ") || strings.Contains(got, "exec-once=ilmango ") {
		t.Errorf("a bare launcher name survived:\n%s", got)
	}
	// Compositor functions are not commands.
	if !strings.Contains(got, "bind=SUPER,f,togglefullscreen,") {
		t.Error("a compositor function was rewritten")
	}
}

func TestMangoSignatureIsFoundWithoutTheEnvironment(t *testing.T) {
	// The variable is set for anything mango spawned. An installer run over
	// ssh or from another VT has no such parent — and those are exactly the
	// runs that would otherwise leave the compositor on the old config.
	runtime := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", runtime)
	t.Setenv("MANGO_INSTANCE_SIGNATURE", "")

	if got := mangoSignature(); got != "" {
		t.Fatalf("with no socket present, mangoSignature() = %q, want empty", got)
	}

	socket := filepath.Join(runtime, "mango-1234.sock")
	if err := os.WriteFile(socket, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if got := mangoSignature(); got != socket {
		t.Fatalf("mangoSignature() = %q, want %q", got, socket)
	}

	// Two sessions: reloading an arbitrary one is worse than reloading none.
	if err := os.WriteFile(filepath.Join(runtime, "mango-5678.sock"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if got := mangoSignature(); got != "" {
		t.Fatalf("with two sockets, mangoSignature() = %q, want empty", got)
	}
}

func TestMangoSignaturePrefersTheEnvironment(t *testing.T) {
	runtime := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", runtime)
	t.Setenv("MANGO_INSTANCE_SIGNATURE", "/run/user/1000/mango-99.sock")
	if err := os.WriteFile(filepath.Join(runtime, "mango-1234.sock"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	if got := mangoSignature(); got != "/run/user/1000/mango-99.sock" {
		t.Fatalf("mangoSignature() = %q, want the session's own socket", got)
	}
}

func TestAnIncludeBelowTheBindsIsMovedUp(t *testing.T) {
	// The case a real machine hit: an older version appended the include, and
	// every later run saw the line, considered the job done, and left the
	// shell's keys losing all sixteen collisions — Super+M among them, which
	// reached the default underneath and quit the compositor.
	note := "# Added by Illogical-mango: shell keybinds and autostart.\nsource-optional=/home/u/.config/mango/ilmango.conf\n\n"
	config := strings.Join([]string{
		"# the user's own config",
		"bind=SUPER,m,quit",
		"bind=SUPER,r,reload_config",
		"",
		"# Added by Illogical-mango: shell keybinds and autostart.",
		"source-optional=/home/u/.config/mango/ilmango.conf",
	}, "\n")

	moved, action := placeSourceLine(config, note)

	if action != sourceLineMoved {
		t.Fatalf("action = %v, want the include to be moved", action)
	}
	sourceAt := strings.Index(moved, "source-optional=")
	quitAt := strings.Index(moved, "bind=SUPER,m,quit")
	if sourceAt > quitAt {
		t.Fatalf("the include is still below the binds:\n%s", moved)
	}
	if n := strings.Count(moved, "source-optional="); n != 1 {
		t.Fatalf("the include appears %d times after the move:\n%s", n, moved)
	}
	// The comment travels with it rather than being stranded mid-file.
	if n := strings.Count(moved, "# Added by Illogical-mango"); n != 1 {
		t.Fatalf("the comment appears %d times:\n%s", n, moved)
	}
	if !strings.Contains(moved, "# the user's own config") {
		t.Error("the user's own config did not survive the move")
	}
}

func TestAnIncludeAlreadyFirstIsLeftAlone(t *testing.T) {
	note := "# Added by Illogical-mango: shell keybinds and autostart.\nsource-optional=/home/u/.config/mango/ilmango.conf\n\n"
	config := note + "bind=SUPER,m,quit\n"

	unchanged, action := placeSourceLine(config, note)

	if action != sourceLineUnchanged {
		t.Fatalf("action = %v, want unchanged", action)
	}
	if unchanged != config {
		t.Fatalf("a config that was already correct was rewritten:\n%s", unchanged)
	}
}

func TestAnIncludeWithNoBindsBelowItIsLeftAlone(t *testing.T) {
	// Nothing to lose to means nothing to fix; rewriting would be churn.
	note := "# Added by Illogical-mango: shell keybinds and autostart.\nsource-optional=/home/u/.config/mango/ilmango.conf\n\n"
	config := "cursor_size=24\nsource-optional=/home/u/.config/mango/ilmango.conf\n"

	unchanged, action := placeSourceLine(config, note)

	if action != sourceLineUnchanged || unchanged != config {
		t.Fatalf("action = %v; a config with no binds was rewritten:\n%s", action, unchanged)
	}
}
