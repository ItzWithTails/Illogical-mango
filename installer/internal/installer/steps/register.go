package steps

import "ilmango/internal/installer"

// Registration order is execution order. Adding a step means writing it here
// and, if it needs a new toggle, adding the option to installer/options.go —
// the UI renders both catalogues without further change.
func init() {
	// An update is an install that first moves the source forward, so it
	// reuses the install list rather than growing a parallel one that would
	// quietly drift out of step with it.
	install := []installer.Step{
		newConflictsStep(),
		newPackagesStep(),
		newSystemStep(),
		newFilesStep(),
		newLauncherStep(),
		newPathStep(),
		newStateDirsStep(),
		newPythonStep(),
		newDesktopStep(),
		newServiceStep(),
		newDesktopSettingsStep(),
		newMangoStep(),
		// The extras run once the shell is on disk — the mascot pack unpacks
		// into it — and before the manifest, so what they install is recorded.
		newWallpapersStep(),
		newIconThemeStep(),
		newMascotStep(),
		newSDDMThemeStep(),
		newVersionStep(),
		// The manifest is written last so it captures every earlier write,
		// version.json included.
		newManifestStep(),
		// Last of all: everything the shell reads must already be in place.
		newStartStep(),
	}

	installer.Register(installer.OpInstall, install...)
	installer.Register(installer.OpUpdate, append([]installer.Step{newPullStep()}, install...)...)
	installer.Register(installer.OpRollback, newRollbackStep())
	installer.Register(installer.OpChanges, newChangesStep())

	// Removal runs in the reverse order of installation: services stop before
	// the files that define them go, and the record of the install is the last
	// thing to be discarded.
	installer.Register(installer.OpUninstall,
		newStopServicesStep(),
		newUnhookMangoStep(),
		newUnhookPathStep(),
		newRemoveFilesStep(),
		newRemoveVenvStep(),
		newReportPackagesStep(),
		newReportExtrasStep(),
		newForgetStep(),
	)
}
