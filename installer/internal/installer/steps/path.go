package steps

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ilmango/internal/installer"
)

// pathMarker identifies the line this installer adds to a shell profile, so it
// can be recognised later and removed without disturbing anything around it.
const pathMarker = "# Added by Illogical-mango: put the ilmango command on PATH."

// pathStep makes the ilmango command reachable.
//
// Installing a command the user cannot then run is the most easily avoided
// failure in this whole program, and for most of this project's life the
// installer merely warned about it. The warning was correct and useless: it
// told someone their shell was broken and left them to fix it.
//
// What it writes depends on the shell, because there is no one file that all
// of them read. fish gets a file of its own under conf.d, which is the
// idiomatic place and which uninstalling can simply delete. bash and zsh get a
// single guarded line appended to the profile they read at login, in the same
// style as the mango hook: one line, marked, and removable.
type pathStep struct{ base }

func newPathStep() installer.Step {
	return pathStep{base{
		id:     "path",
		title:  "Put ilmango on your PATH",
		detail: "Add the launcher's directory to the shell profile, if it is missing.",
		applies: func(c installer.Config) bool {
			return c.Effective(installer.OptPathEntry)
		},
	}}
}

// Optional marks this as survivable: an unwritable profile costs convenience.
func (pathStep) Optional() bool { return true }

func (s pathStep) Run(_ context.Context, env *installer.Env) error {
	dir := binHome()
	if onPath(dir) {
		env.Detail(dir + " is already on PATH")
		return nil
	}

	shell := loginShell()
	env.Detail("login shell looks like " + shell)

	switch shell {
	case "fish":
		return s.writeFishSnippet(env, dir)
	default:
		return s.appendProfileLine(env, shell, dir)
	}
}

// writeFishSnippet gives fish a file of its own. fish sources every .fish in
// conf.d, so nothing existing has to be edited.
func (s pathStep) writeFishSnippet(env *installer.Env, dir string) error {
	target := filepath.Join(configHome(), "fish", "conf.d", "ilmango-path.fish")
	if err := env.Home.EnsureDir(filepath.Dir(target), 0o755); err != nil {
		return err
	}

	body := pathMarker + "\nfish_add_path " + dir + "\n"
	if err := env.Home.WriteFile(target, []byte(body), 0o644); err != nil {
		return err
	}
	env.Detail(target)
	env.NoteApplied("The ilmango command is on your PATH from your next shell. In this one: fish_add_path " + dir)
	return nil
}

// appendProfileLine adds one guarded export to the profile a login shell reads.
func (s pathStep) appendProfileLine(env *installer.Env, shell, dir string) error {
	target := profileFor(shell)
	if err := env.Home.EnsureDir(filepath.Dir(target), 0o755); err != nil {
		return err
	}

	existing, err := env.Home.ReadFile(target)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading %s: %w", target, err)
	}
	if strings.Contains(string(existing), pathMarker) {
		env.Detail(target + " already has the entry")
		return nil
	}

	// The profile is the user's file, not ours: it is written unrecorded so
	// uninstalling strips our line rather than deleting the whole profile.
	body := string(existing)
	if body != "" && !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	body += "\n" + pathMarker + "\nexport PATH=\"" + dir + ":$PATH\"\n"

	if err := env.Home.Unrecorded().WriteFile(target, []byte(body), 0o644); err != nil {
		return err
	}
	env.Detail(target)
	env.NoteApplied("One line was added to " + target + " so the ilmango command is found. In this shell: export PATH=\"" + dir + ":$PATH\"")
	return nil
}

// onPath reports whether dir is already in PATH.
func onPath(dir string) bool {
	for _, entry := range filepath.SplitList(os.Getenv("PATH")) {
		if entry == dir {
			return true
		}
	}
	return false
}

// loginShell names the user's shell, falling back to bash. Only the basename
// matters: what is wanted is which family of profile files to write.
func loginShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return "bash"
	}
	return filepath.Base(shell)
}

// profileFor picks the file a login shell of that family actually reads.
func profileFor(shell string) string {
	if shell == "zsh" {
		// .zprofile, not .zshrc: a graphical session started from a login
		// shell reads the former and may never read the latter.
		return filepath.Join(home(), ".zprofile")
	}
	return filepath.Join(home(), ".profile")
}

// stripPathEntry removes the line this installer added, and the marker above
// it, leaving everything else in the profile untouched.
func stripPathEntry(content string) (string, int) {
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	removed := 0

	for i := 0; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != pathMarker {
			out = append(out, lines[i])
			continue
		}
		removed++
		// The export always follows the marker; anything else means someone
		// edited it, and then it is theirs to keep.
		if i+1 < len(lines) && strings.Contains(lines[i+1], "PATH") {
			i++
			removed++
		}
	}
	return strings.Join(out, "\n"), removed
}
