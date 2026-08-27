package steps

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ilmango/internal/installer"

	"ilmango/internal/fsx"
)

func TestStripPathEntryRemovesOnlyOurLines(t *testing.T) {
	profile := strings.Join([]string{
		"# the user's own profile",
		"export EDITOR=nvim",
		"export PATH=\"$HOME/bin:$PATH\"",
		"",
		pathMarker,
		"export PATH=\"/home/u/.local/bin:$PATH\"",
		"",
		"echo hello",
	}, "\n")

	cleaned, removed := stripPathEntry(profile)

	if removed != 2 {
		t.Errorf("removed %d lines, want the marker and the export", removed)
	}
	for _, want := range []string{"export EDITOR=nvim", "export PATH=\"$HOME/bin:$PATH\"", "echo hello"} {
		if !strings.Contains(cleaned, want) {
			t.Errorf("stripping removed the user's own line %q", want)
		}
	}
	if strings.Contains(cleaned, "/home/u/.local/bin") {
		t.Error("our export survived")
	}
}

func TestStripPathEntryLeavesAnEditedLineAlone(t *testing.T) {
	// If the line under our marker is no longer an export, someone changed it
	// and it is theirs to keep.
	profile := pathMarker + "\nsource ~/my-own-setup.sh\n"

	cleaned, removed := stripPathEntry(profile)

	if removed != 1 {
		t.Errorf("removed %d lines, want only the marker", removed)
	}
	if !strings.Contains(cleaned, "my-own-setup.sh") {
		t.Error("an edited line was removed along with the marker")
	}
}

func TestProfileTargetsCoverLoginAndInteractiveShells(t *testing.T) {
	// bash reads .bash_profile at login and .bashrc in every terminal opened
	// afterwards. Writing only the first leaves the command missing from every
	// terminal until the next login.
	home := t.TempDir()
	t.Setenv("HOME", home)

	targets := profileTargets("bash")
	if len(targets) != 2 {
		t.Fatalf("profileTargets(bash) = %v, want a login file and an rc file", targets)
	}
	if want := filepath.Join(home, ".profile"); targets[0] != want {
		t.Errorf("with no bash files present the login target is %q, want %q", targets[0], want)
	}
	if want := filepath.Join(home, ".bashrc"); targets[1] != want {
		t.Errorf("the interactive target is %q, want %q", targets[1], want)
	}

	// bash takes the first of .bash_profile, .bash_login and .profile that
	// exists. Arch ships a .bash_profile that sources only .bashrc.
	bashProfile := filepath.Join(home, ".bash_profile")
	if err := os.WriteFile(bashProfile, []byte("[[ -f ~/.bashrc ]] && . ~/.bashrc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := profileTargets("bash")[0]; got != bashProfile {
		t.Fatalf("login target = %q, want %q — the file bash will read", got, bashProfile)
	}
}

func TestProfileTargetsForZsh(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	targets := profileTargets("zsh")
	if len(targets) != 2 || !strings.HasSuffix(targets[0], ".zprofile") || !strings.HasSuffix(targets[1], ".zshrc") {
		t.Fatalf("profileTargets(zsh) = %v, want .zprofile then .zshrc", targets)
	}
}

func TestPathLineGuardsAgainstBeingAppliedTwice(t *testing.T) {
	// A login shell reads the profile and then sources the rc file, so an
	// unguarded export would put the directory on PATH twice.
	line := guardedPathLine("/home/u/.local/bin")

	for _, want := range []string{`case ":$PATH:" in`, `*":/home/u/.local/bin:"*) ;;`, `export PATH="/home/u/.local/bin:$PATH"`} {
		if !strings.Contains(line, want) {
			t.Errorf("the line is missing %q:\n%s", want, line)
		}
	}
	if strings.Contains(line, "\n") {
		t.Errorf("the line spans more than one line, which the uninstaller strips by pairs:\n%s", line)
	}
}

func TestEveryInstalledShellIsCovered(t *testing.T) {
	// $SHELL names the login shell, which on a normal Arch machine is bash
	// while the terminal this project ships opens fish. The shell someone
	// types into is chosen by their terminal, so guessing from $SHELL leaves
	// the command missing from the one they actually use.
	shells := installedShells()
	if len(shells) == 0 {
		t.Fatal("installedShells() returned nothing; there is always a fallback")
	}

	// bash exists everywhere this runs.
	var sawBash bool
	for _, s := range shells {
		if s == "bash" {
			sawBash = true
		}
	}
	if !sawBash {
		t.Errorf("installedShells() = %v, want bash among them", shells)
	}
}

func TestFishSnippetDoesNotOutliveItsFile(t *testing.T) {
	// fish_add_path records a universal variable that would survive both the
	// file being deleted and the shell being uninstalled. Setting PATH for the
	// shell keeps the change tied to the file the uninstaller removes.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	env := &installer.Env{Home: fsx.Tree{Mode: fsx.ModeApply}}
	target, err := (pathStep{}).writeFishSnippet(env, "/home/u/.local/bin")
	if err != nil {
		t.Fatalf("writeFishSnippet() error = %v", err)
	}

	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "fish_add_path") {
		t.Errorf("the snippet uses fish_add_path, which outlives the file:\n%s", body)
	}
	for _, want := range []string{"if not contains /home/u/.local/bin $PATH", "set -gx PATH /home/u/.local/bin $PATH"} {
		if !strings.Contains(string(body), want) {
			t.Errorf("missing %q in:\n%s", want, body)
		}
	}
}
