package steps

import (
	"context"
	"os"
	"strings"

	"ilmango/internal/installer"
	"ilmango/internal/run"
)

// startStep launches the shell as soon as the installation finishes.
//
// This is deliberately opt-in. The compositor config the installer writes
// already carries `exec-once=ilmango run --daemon`, so the shell comes up by
// itself at the next login; starting it here only helps someone installing
// from a session that is already running, and doing it unasked would surprise
// anyone who expected an installer to install and stop there.
type startStep struct{ base }

func newStartStep() installer.Step {
	return startStep{base{
		id:     "start-shell",
		title:  "Start the shell",
		detail: "Launch Illogical-mango in the session that is already running.",
		applies: func(c installer.Config) bool {
			return c.Effective(installer.OptStartNow)
		},
	}}
}

// Optional keeps a failed launch from marking a good installation as broken.
func (startStep) Optional() bool { return true }

func (s startStep) Run(ctx context.Context, env *installer.Env) error {
	if os.Getenv("WAYLAND_DISPLAY") == "" {
		env.Note("There is no Wayland session here, so the shell was not started. It will come up on its own when you log into the compositor.")
		return nil
	}
	if s.alreadyRunning(ctx, env) {
		env.Log("the shell is already running; leaving it alone")
		return nil
	}

	launcher := LauncherPath()
	if !env.Home.Exists(launcher) {
		env.Log("the launcher is not installed; nothing to start")
		return nil
	}

	env.Detail("starting")
	if err := env.Runner.Start(ctx, run.Command{Name: launcher, Args: []string{"run", "--daemon"}}); err != nil {
		return err
	}

	env.NoteApplied("The shell was started. If anything looks wrong, check it with: ilmango logs")
	return nil
}

// alreadyRunning avoids a second instance fighting the first over the same
// Wayland surfaces.
func (s startStep) alreadyRunning(ctx context.Context, env *installer.Env) bool {
	out, err := env.Runner.Output(ctx, run.Command{Name: "pgrep", Args: []string{"-af", "quickshell"}})
	if err != nil {
		return false
	}
	return strings.Contains(out, "ilmango")
}
