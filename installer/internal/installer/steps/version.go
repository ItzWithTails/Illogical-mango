package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"ilmango/internal/installer"
)

// versionRecord is the contract the update operation and the shell's own
// version panel read. The duplicated snake_case and camelCase keys are part of
// that contract, not an oversight: both spellings are in use across the tree.
type versionRecord struct {
	Version        string `json:"version"`
	Commit         string `json:"commit"`
	InstalledAt    string `json:"installed_at"`
	InstalledAtAlt string `json:"installedAt"`
	Source         string `json:"source"`
	RepoPath       string `json:"repo_path"`
	RepoPathAlt    string `json:"repoPath"`
	InstallMode    string `json:"install_mode"`
	InstallModeAlt string `json:"installMode"`
}

type versionStep struct{ base }

func newVersionStep() installer.Step {
	return versionStep{base{
		id:     "version",
		title:  "Record the installed version",
		detail: "Write version.json so updates and diagnostics know this build.",
		// Only claim a version is installed when something actually was.
		// Recording one after a run that installed nothing would mislead
		// every tool that reads version.json.
		applies: func(c installer.Config) bool {
			return c.Effective(installer.OptConfigFiles) || c.Effective(installer.OptDependencies)
		},
	}}
}

func (s versionStep) Run(_ context.Context, env *installer.Env) error {
	now := time.Now().Format(time.RFC3339)
	record := versionRecord{
		Version:        env.Repo.Version,
		Commit:         env.Repo.Commit,
		InstalledAt:    now,
		InstalledAtAlt: now,
		Source:         "installer",
		RepoPath:       env.Repo.Root,
		RepoPathAlt:    env.Repo.Root,
		InstallMode:    "repo-managed",
		InstallModeAlt: "repo-managed",
	}

	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding version record: %w", err)
	}

	path := filepath.Join(configHome(), "ilmango", "version.json")
	if err := env.Home.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return err
	}

	env.Detail("v" + env.Repo.Version)
	return nil
}
