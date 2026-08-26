package steps

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"ilmango/internal/installer"
	"ilmango/internal/run"
)

// venvPath is where the shell's Python helpers live. The path is part of the
// contract with the shell, which reads it from ILMANGO_VENV.
func venvPath() string {
	return filepath.Join(stateHome(), "quickshell", ".venv")
}

// pythonStep provisions the virtual environment the shell's Python helpers
// need — the YT Music client, gesture handling and the image tooling.
//
// Optional: without it those individual features stop working, while the shell
// itself starts and runs perfectly well.
type pythonStep struct{ base }

func newPythonStep() installer.Step {
	return pythonStep{base{
		id:     "python-env",
		title:  "Set up the Python environment",
		detail: "Create the virtualenv the shell's Python helpers run in.",
		applies: func(c installer.Config) bool {
			return c.Effective(installer.OptConfigFiles)
		},
	}}
}

// Optional marks the environment as a per-feature dependency.
func (pythonStep) Optional() bool { return true }

func (s pythonStep) Run(ctx context.Context, env *installer.Env) error {
	requirements := filepath.Join(env.Repo.Root, "sdata", "uv", "requirements.txt")
	if _, err := os.Stat(requirements); err != nil {
		env.Log("no requirements.txt in this checkout; nothing to install")
		return nil
	}
	if !run.Exists("uv") {
		env.Note("uv is not installed, so the Python environment was not created. Install uv and run the installer again to restore YT Music and gesture support.")
		return nil
	}

	dir := venvPath()
	if err := s.ensureVenv(ctx, env, dir); err != nil {
		return err
	}

	env.Detail("installing Python packages")
	return env.Runner.Run(ctx, run.Command{
		Name: "uv",
		Args: []string{"pip", "install", "-r", requirements},
		// uv installs into the environment VIRTUAL_ENV names, which avoids
		// having to source an activation script from a non-interactive shell.
		Env: []string{"VIRTUAL_ENV=" + dir},
	})
}

// ensureVenv creates the environment, replacing one left broken by a Python
// upgrade — a stale venv is the most common cause of the helpers silently
// failing long after installation.
func (s pythonStep) ensureVenv(ctx context.Context, env *installer.Env, dir string) error {
	python := filepath.Join(dir, "bin", "python")

	if env.Home.Exists(python) {
		if _, err := env.Runner.Output(ctx, run.Command{Name: python, Args: []string{"--version"}}); err == nil {
			env.Log("existing Python environment is healthy")
			return nil
		}
		env.Log("existing Python environment is broken; recreating it")
		if err := env.Home.RemoveAll(dir); err != nil {
			return err
		}
	}

	if err := env.Home.EnsureDir(filepath.Dir(dir), 0o755); err != nil {
		return err
	}

	env.Detail("creating the virtualenv")
	// Pin the interpreter the shell was tested against, but do not insist:
	// a distribution that ships only a newer Python should still work.
	pinned := run.Command{Name: "uv", Args: []string{"venv", "--prompt", "ii-venv", dir, "-p", "3.12"}}
	if err := env.Runner.Run(ctx, pinned); err == nil {
		return nil
	}

	env.Log("Python 3.12 is unavailable; using the system interpreter")
	fallback := run.Command{Name: "uv", Args: []string{"venv", "--prompt", "ii-venv", dir}}
	if err := env.Runner.Run(ctx, fallback); err != nil {
		return fmt.Errorf("creating the Python environment: %w", err)
	}
	return nil
}
