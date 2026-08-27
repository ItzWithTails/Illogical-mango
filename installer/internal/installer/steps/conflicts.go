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

	s.reportPackageConflicts(ctx, env, manager)

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

// reportPackageConflicts names dependencies that cannot be installed over what
// is already there.
//
// pacman discovers this at the end of the transaction, which on Arch can be an
// hour of compiling away, and then refuses the whole thing. Saying it here
// costs a few queries and lets someone decide before waiting.
//
// It only reports. Removing a package to make room is not something an
// installer should decide on a working machine — the replacement might be the
// one the user wanted.
func (s conflictsStep) reportPackageConflicts(ctx context.Context, env *installer.Env, manager pkg.Manager) {
	if !env.Config.Effective(installer.OptDependencies) {
		return
	}

	family := string(env.Distro.Family)
	wanted := env.Config.KeepPackages(pkg.Packages(family, groupsFor(env.Config)...))
	if len(wanted) == 0 {
		return
	}

	conflicts := manager.ConflictsWith(ctx, env.Runner, wanted)
	if len(conflicts) == 0 {
		return
	}

	var lines []string
	for _, c := range conflicts {
		env.Log("conflict: " + c.Wanted + " cannot be installed over " + c.Installed)
		lines = append(lines, c.Wanted+" over "+c.Installed)
	}
	env.Note(fmt.Sprintf(
		"These packages cannot be installed while what is already there stays: %s. "+
			"The install will fail on them unless you remove the installed one yourself, "+
			"or leave the new one out with --without.",
		strings.Join(lines, ", ")))
}
