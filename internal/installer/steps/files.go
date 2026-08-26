package steps

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"ilmango/internal/installer"
)

// payloadDirs are the repository directories that make up the shell itself.
// The list mirrors sdata/runtime-payload-dirs.txt, which the packaging
// Makefile also reads.
var payloadDirs = []string{
	"modules",
	"services",
	"scripts",
	"assets",
	"translations",
	"defaults",
	"dots",
	"sdata",
}

// payloadFiles are the loose repository files the shell needs at runtime.
var payloadFiles = []string{
	"VERSION",
	"CHANGELOG.md",
}

// excludedNames are development artefacts that must never reach a user's
// machine: agent directives, editor state and maintainer tooling. The
// packaging Makefile strips the same set.
var excludedNames = map[string]bool{
	"AGENTS.md": true, "CLAUDE.md": true, "CODEX.md": true, "PI.md": true,
	"codemap.md": true, ".mcp.json": true, "opencode.json": true,
	"skills-lock.json": true,
	".agents":          true, ".claude": true, ".codex": true, ".factory": true,
	".opencode": true, ".codebase-memory": true, ".impeccable": true,
	".pi-subagents": true,
}

type filesStep struct{ base }

func newFilesStep() installer.Step {
	return filesStep{base{
		id:     "files",
		title:  "Install config files",
		detail: "The shell itself, plus dotfiles for Mango, the terminal and theming.",
		applies: func(c installer.Config) bool {
			return c.Effective(installer.OptConfigFiles)
		},
	}}
}

func (s filesStep) Run(ctx context.Context, env *installer.Env) error {
	if err := s.installShell(ctx, env); err != nil {
		return err
	}
	if err := s.installDefaults(ctx, env); err != nil {
		return err
	}
	return s.installDotfiles(ctx, env)
}

// installShell copies the QML shell and its payload into the Quickshell config
// directory, where Quickshell will look for it.
func (s filesStep) installShell(ctx context.Context, env *installer.Env) error {
	target := shellDir()
	env.Detail("shell → " + target)

	if err := env.Home.EnsureDir(target, 0o755); err != nil {
		return err
	}

	qml, err := filepath.Glob(filepath.Join(env.Repo.Root, "*.qml"))
	if err != nil {
		return fmt.Errorf("locating QML files: %w", err)
	}
	if len(qml) == 0 {
		return fmt.Errorf("no QML files in %s — is this an Illogical-mango checkout?", env.Repo.Root)
	}

	for _, src := range append(qml, repoPaths(env, "qmldir")...) {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := env.Home.CopyFile(src, filepath.Join(target, filepath.Base(src))); err != nil {
			return err
		}
	}

	for _, name := range payloadFiles {
		src := filepath.Join(env.Repo.Root, name)
		if _, err := os.Stat(src); err != nil {
			continue // optional
		}
		if err := env.Home.CopyFile(src, filepath.Join(target, name)); err != nil {
			return err
		}
	}

	for _, dir := range payloadDirs {
		if err := ctx.Err(); err != nil {
			return err
		}
		src := filepath.Join(env.Repo.Root, dir)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		env.Detail("shell → " + dir)
		if err := env.Home.CopyTreeExcept(src, filepath.Join(target, dir), excluded); err != nil {
			return err
		}
	}
	return nil
}

// defaultDirs are trees under defaults/ that land in the user's config home.
// defaults/ is the single source of truth for these: dots/ used to carry a
// hand-maintained mirror that silently drifted (old GTK colour tokens, a
// pre-modular niri config, an outdated UI font), so it was removed.
var defaultDirs = map[string]string{
	"matugen": "matugen",
	"fuzzel":  "fuzzel",
	"gtk-3.0": "gtk-3.0",
	"gtk-4.0": "gtk-4.0",
}

// defaultFiles are single files under defaults/ that land in the config home.
// The KDE pair is only useful when Dolphin is actually installed.
var defaultFiles = []struct {
	src, dst    string
	requiresCmd string
}{
	{src: "kde/kdeglobals", dst: "kdeglobals"},
	{src: "kde/dolphinrc", dst: "dolphinrc", requiresCmd: "dolphin"},
	{src: "kde/kservicemenurc", dst: "kservicemenurc", requiresCmd: "dolphin"},
}

// installDefaults applies defaults/ to the user's config home, mirroring what
// sdata/subcmd-install/3.files.sh does for the shell-script install path.
func (s filesStep) installDefaults(ctx context.Context, env *installer.Env) error {
	defaults := filepath.Join(env.Repo.Root, "defaults")
	if _, err := os.Stat(defaults); err != nil {
		return nil // optional payload
	}
	cfg := configHome()

	for src, dst := range defaultDirs {
		if err := ctx.Err(); err != nil {
			return err
		}
		from := filepath.Join(defaults, src)
		if _, err := os.Stat(from); err != nil {
			continue
		}
		env.Detail("defaults → ~/.config/" + dst)
		if err := env.Home.CopyTreeExcept(from, filepath.Join(cfg, dst), excluded); err != nil {
			return err
		}
	}

	for _, f := range defaultFiles {
		if err := ctx.Err(); err != nil {
			return err
		}
		if f.requiresCmd != "" {
			if _, err := exec.LookPath(f.requiresCmd); err != nil {
				continue
			}
		}
		from := filepath.Join(defaults, f.src)
		if _, err := os.Stat(from); err != nil {
			continue
		}
		env.Detail("defaults → ~/.config/" + f.dst)
		if err := env.Home.CopyFile(from, filepath.Join(cfg, f.dst)); err != nil {
			return err
		}
	}

	// The compositor config is not handled here: this fork targets Mango, and
	// mangoStep hooks defaults/mango/config.conf in on its own.
	return nil
}

// installDotfiles mirrors dots/ into the home directory. The layout under
// dots/ is already the layout it lands in, so this stays a straight copy.
func (s filesStep) installDotfiles(ctx context.Context, env *installer.Env) error {
	dots := filepath.Join(env.Repo.Root, "dots")
	entries, err := os.ReadDir(dots)
	if err != nil {
		return fmt.Errorf("reading dotfiles: %w", err)
	}

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		// dots/sddm is a system theme, installed separately and only with
		// explicit consent; it does not belong in the home directory.
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		env.Detail("dotfiles → ~/" + entry.Name())
		src := filepath.Join(dots, entry.Name())
		if err := env.Home.CopyTreeExcept(src, filepath.Join(home(), entry.Name()), excluded); err != nil {
			return err
		}
	}

	if env.Backup != nil && env.Backup.Count() > 0 {
		env.NoteApplied(fmt.Sprintf("%d existing files were replaced. The originals are in %s.",
			env.Backup.Count(), env.Backup.Path()))
	}
	return nil
}

// excluded reports whether an entry is a development artefact that must never
// reach a user's machine.
func excluded(name string) bool { return excludedNames[name] }

// repoPaths returns the named repository files that actually exist.
func repoPaths(env *installer.Env, names ...string) []string {
	var out []string
	for _, name := range names {
		path := filepath.Join(env.Repo.Root, name)
		if _, err := os.Stat(path); err == nil {
			out = append(out, path)
		}
	}
	return out
}
