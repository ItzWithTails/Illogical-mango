package steps

import (
	"context"
	"fmt"
	"path/filepath"

	"ilmango/internal/installer"
)

// ManifestPath is where the record of an installation lives. An uninstall
// reads it back, so its location is part of the contract between the two.
func ManifestPath() string {
	return filepath.Join(stateHome(), "ilmango", "installed-files.json")
}

type manifestStep struct{ base }

func newManifestStep() installer.Step {
	return manifestStep{base{
		id:     "manifest",
		title:  "Record what was installed",
		detail: "Write the file manifest that a later uninstall reads back.",
		applies: func(c installer.Config) bool {
			return c.Effective(installer.OptConfigFiles)
		},
	}}
}

func (s manifestStep) Run(_ context.Context, env *installer.Env) error {
	if env.Manifest == nil {
		return fmt.Errorf("no manifest was collected during this run")
	}

	if err := env.Manifest.Save(env.Home, ManifestPath()); err != nil {
		return err
	}

	env.Detail(fmt.Sprintf("%d paths", env.Manifest.Len()))
	return nil
}
