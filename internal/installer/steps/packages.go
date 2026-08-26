package steps

import (
	"context"
	"fmt"
	"strings"

	"ilmango/internal/installer"
	"ilmango/internal/pkg"
)

type packagesStep struct{ base }

func newPackagesStep() installer.Step {
	return packagesStep{base{
		id:     "packages",
		title:  "Install dependencies",
		detail: "Quickshell, the Mango compositor and the tools the shell drives.",
		applies: func(c installer.Config) bool {
			return c.Effective(installer.OptDependencies)
		},
	}}
}

// groupsFor maps installation options onto package groups. Core and Quickshell
// are unconditional: without them there is nothing to install.
func groupsFor(cfg installer.Config) []pkg.Group {
	groups := []pkg.Group{pkg.GroupCore, pkg.GroupQuickshell}
	for opt, group := range map[installer.OptionID]pkg.Group{
		installer.OptAudio:         pkg.GroupAudio,
		installer.OptToolkit:       pkg.GroupToolkit,
		installer.OptScreenCapture: pkg.GroupScreenCapture,
		installer.OptFonts:         pkg.GroupFonts,
	} {
		if cfg.Effective(opt) {
			groups = append(groups, group)
		}
	}
	return groups
}

func (s packagesStep) Run(ctx context.Context, env *installer.Env) error {
	family := string(env.Distro.Family)

	if !pkg.KnownFamily(family) {
		env.Note(fmt.Sprintf(
			"No package list exists for %s, so dependencies were not installed. Install them by hand — docs/MANUAL_INSTALL.md lists what Illogical-mango needs.",
			env.Distro.Name))
		env.Log("unsupported family " + family + "; skipping dependency installation")
		return nil
	}

	manager, err := pkg.FindManagerPreferring(family, env.Config.Choice(installer.OptAURHelper))
	if err != nil {
		return err
	}
	env.Log("using " + manager.Name)

	if family == "arch" && !manager.IsAURHelper() {
		env.Note("No AUR helper is in use, so packages that live only in the AUR — the variable fonts among them — cannot be installed. Install paru or yay and run this again if the shell looks wrong.")
	}

	packages := pkg.Packages(family, groupsFor(env.Config)...)
	env.Detail(fmt.Sprintf("%d packages", len(packages)))

	upgrade := env.Config.Choice(installer.OptSystemUpgrade) == installer.UpgradeFull
	switch {
	case upgrade && manager.CanUpgrade():
		env.Detail("upgrading the system")
		if err := manager.Refresh(ctx, env.Runner, true); err != nil {
			return fmt.Errorf("upgrading the system: %w", err)
		}
	case manager.CanUpgrade():
		// The databases are deliberately left alone: see Manager.Refresh.
		env.Note("The system was not upgraded, so packages are installed from the databases already on this machine. If any of them cannot be found, upgrade the system and run this again.")
	default:
		env.Detail("refreshing package databases")
		if err := manager.Refresh(ctx, env.Runner, false); err != nil {
			return fmt.Errorf("refreshing package databases: %w", err)
		}
	}

	env.Detail(fmt.Sprintf("installing %d packages", len(packages)))
	skipped, err := manager.Install(ctx, env.Runner, packages)
	if err != nil {
		return err
	}

	s.reportFailures(env, skipped)
	return nil
}

// reportFailures tells the user about packages that would not install,
// separating the ones that break the shell from the ones that cost a feature.
func (s packagesStep) reportFailures(env *installer.Env, failed []string) {
	if len(failed) == 0 {
		return
	}

	critical, optional := pkg.SplitCritical(failed)

	if len(critical) > 0 {
		env.Note(fmt.Sprintf(
			"These packages are required and could not be installed: %s. Illogical-mango will not start correctly until they are present.",
			strings.Join(critical, ", ")))
	}
	if len(optional) > 0 {
		env.Note(fmt.Sprintf(
			"%d optional packages could not be installed: %s. Each one costs a feature, not the whole shell.",
			len(optional), strings.Join(optional, ", ")))
	}
}
