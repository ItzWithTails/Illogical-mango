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

	// Two mechanisms, because there are two ways a session starts. A login
	// shell reads a profile; a session started by a display manager or a user
	// service does not, and reads environment.d instead. Writing both is
	// cheaper than guessing which one this machine will use.
	if err := s.writeSystemdEnvironment(env, dir); err != nil {
		return err
	}

	shell := loginShell()
	env.Detail("login shell looks like " + shell)

	// Every shell that is installed, not the one $SHELL names.
	//
	// $SHELL is what the login shell was, which on a normal Arch machine is
	// bash — while the terminal this project ships opens fish. The shell
	// someone types into is chosen by their terminal, not by /etc/passwd, so
	// guessing from either gets it wrong. Writing for each shell present costs
	// a marked line in a file the uninstaller removes, and guesses nothing.
	var written []string
	for _, shell := range installedShells() {
		if shell == "fish" {
			target, err := s.writeFishSnippet(env, dir)
			if err != nil {
				return err
			}
			if target != "" {
				written = append(written, target)
			}
			continue
		}
		for _, target := range profileTargets(shell) {
			added, err := s.appendProfileLine(env, target, dir)
			if err != nil {
				return err
			}
			if added {
				written = append(written, target)
			}
		}
	}
	if len(written) == 0 {
		return nil
	}

	env.NoteApplied(fmt.Sprintf(
		"The ilmango command was added to %s. New terminals have it at once; the compositor keybinds that run it need a full login, because a running session cannot be given a new PATH.",
		strings.Join(written, ", ")))
	return nil
}

// installedShells lists the shells worth writing for, in a stable order.
func installedShells() []string {
	var out []string
	for _, shell := range []string{"bash", "zsh", "fish"} {
		if run.Exists(shell) {
			out = append(out, shell)
		}
	}
	if len(out) == 0 {
		// Nothing recognisable: write the POSIX profile and hope.
		return []string{"bash"}
	}
	return out
}

// writeSystemdEnvironment covers sessions that never run a login shell.
//
// systemd's user manager reads every .conf here and applies it to the units it
// starts, which is how a session launched by a display manager or by uwsm gets
// its environment. It is ignored by a plain tty login, which is why the shell
// profile is written as well.
func (s pathStep) writeSystemdEnvironment(env *installer.Env, dir string) error {
	target := filepath.Join(configHome(), "environment.d", "10-ilmango-path.conf")
	if err := env.Home.EnsureDir(filepath.Dir(target), 0o755); err != nil {
		return err
	}

	body := "# " + strings.TrimPrefix(pathMarker, "# ") + "\nPATH=" + dir + ":$PATH\n"
	if err := env.Home.WriteFile(target, []byte(body), 0o644); err != nil {
		return err
	}
	env.Detail(target)
	return nil
}

// writeFishSnippet gives fish a file of its own. fish sources every .fish in
// conf.d, so nothing existing has to be edited.
//
// It sets PATH for the shell rather than calling fish_add_path, which records
// a universal variable that would outlive the file and survive an uninstall.
func (s pathStep) writeFishSnippet(env *installer.Env, dir string) (string, error) {
	target := filepath.Join(configHome(), "fish", "conf.d", "ilmango-path.fish")
	if err := env.Home.EnsureDir(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}

	body := pathMarker + "\n" +
		"if not contains " + dir + " $PATH\n" +
		"    set -gx PATH " + dir + " $PATH\n" +
		"end\n"
	if err := env.Home.WriteFile(target, []byte(body), 0o644); err != nil {
		return "", err
	}
	env.Detail(target)
	return target, nil
}

// appendProfileLine adds the entry to one file, once, and says whether it did.
//
// The line guards itself rather than prepending blindly: a login shell reads
// the profile and then sources the rc file, so an unguarded export would put
// the directory on PATH twice.
func (s pathStep) appendProfileLine(env *installer.Env, target, dir string) (bool, error) {
	if err := env.Home.EnsureDir(filepath.Dir(target), 0o755); err != nil {
		return false, err
	}

	existing, err := env.Home.ReadFile(target)
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("reading %s: %w", target, err)
	}
	if strings.Contains(string(existing), pathMarker) {
		env.Detail(target + " already has the entry")
		return false, nil
	}

	// The file is the user's, not ours: it is written unrecorded so
	// uninstalling strips our line rather than deleting the whole profile.
	body := string(existing)
	if body != "" && !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	body += "\n" + pathMarker + "\n" + guardedPathLine(dir) + "\n"

	if err := env.Home.Unrecorded().WriteFile(target, []byte(body), 0o644); err != nil {
		return false, err
	}
	env.Detail(target)
	return true, nil
}

// guardedPathLine is the POSIX form for "prepend this once".
func guardedPathLine(dir string) string {
	return fmt.Sprintf(`case ":$PATH:" in *":%s:"*) ;; *) export PATH="%s:$PATH" ;; esac`, dir, dir)
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

// profileTargets lists the files a shell of that family reads, login first.
//
// For the login file bash takes the first of .bash_profile, .bash_login and
// .profile which exists — not all of them, and not whichever we would prefer.
// Arch ships a .bash_profile that sources only .bashrc, so a line added to
// .profile on such a machine is never read at all.
//
// The rc file matters just as much and for the opposite reason: every terminal
// opened inside a session is a non-login interactive shell, and it reads only
// that.
func profileTargets(shell string) []string {
	if shell == "zsh" {
		// .zprofile, not .zshrc, for the login half: a graphical session
		// started from a login shell reads the former and may never read the
		// latter.
		return []string{filepath.Join(home(), ".zprofile"), filepath.Join(home(), ".zshrc")}
	}
	return []string{bashLoginFile(), filepath.Join(home(), ".bashrc")}
}

// bashLoginFile is the one bash will actually read at login.
func bashLoginFile() string {
	for _, name := range []string{".bash_profile", ".bash_login"} {
		candidate := filepath.Join(home(), name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
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
