// Package steps holds the concrete installation steps.
//
// Every step is native Go. Nothing here sources or executes the shell
// installer under sdata/: package managers and systemd are driven through
// explicit, logged argument lists, and files are written through fsx, so a
// dry run is genuinely inert and a test run can be pointed at a scratch home.
package steps

import (
	"context"
	"fmt"
	"strings"

	"ilmango/internal/installer"
	"ilmango/internal/pkg"
)

// conflicting are shells that ship their own Quickshell runtime or configs and
// cannot coexist with Illogical-mango.
//
// This step only ever reports them. Removing a package is the user's decision:
// it is not reversible from inside an installer, and a conflict detected here
// is not worth uninstalling something they may still be relying on.
var conflicting = []string{
	"ags",
	"bms-shell-bin",
	"cachyos-niri-noctalia",
	"caelestia-shell",
	"caelestia-shell-git",
	"dms-shell",
	"dms-shell-git",
	"noctalia-qs",
	"noctalia-qs-git",
	"noctalia-shell",
}

type conflictsStep struct{ base }

func newConflictsStep() installer.Step {
	return conflictsStep{base{
		id:     "conflicts",
		title:  "Check for conflicts",
		detail: "Look for other Quickshell desktops that cannot coexist with Illogical-mango.",
	}}
}

// ReadOnly marks this step as an inspection: it reports conflicts and never
// removes anything, so a plan containing only this step changes nothing.
func (conflictsStep) ReadOnly() bool { return true }

func (s conflictsStep) Run(ctx context.Context, env *installer.Env) error {
	manager, err := pkg.FindManager(string(env.Distro.Family))
	if err != nil {
		env.Log("no supported package manager; skipping the conflict scan")
		return nil
	}

	var found []string
	for _, name := range conflicting {
		if err := ctx.Err(); err != nil {
			return err
		}
		if manager.IsInstalled(ctx, env.Runner, name) {
			found = append(found, name)
		}
	}

	if len(found) == 0 {
		env.Log("no conflicting desktop shells installed")
		return nil
	}

	for _, name := range found {
		env.Log("conflict: " + name + " is installed")
	}
	env.Note(fmt.Sprintf(
		"Conflicting shells are installed: %s. Illogical-mango will not work correctly until they are removed — remove them yourself with your package manager when you are ready.",
		strings.Join(found, ", ")))
	return nil
}
