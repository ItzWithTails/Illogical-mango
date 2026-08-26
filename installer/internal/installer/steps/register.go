package steps

import "ilmango/internal/installer"

// Registration order is execution order. Adding a step means writing it here
// and, if it needs a new toggle, adding the option to installer/options.go —
// the UI renders both catalogues without further change.
func init() {
	installer.Register(installer.OpInstall,
		newConflictsStep(),
		newPackagesStep(),
		newSystemStep(),
		newFilesStep(),
		newLauncherStep(),
		newStateDirsStep(),
		newPythonStep(),
		newDesktopStep(),
		newServiceStep(),
		newDesktopSettingsStep(),
		newMangoStep(),
		newVersionStep(),
		// The manifest is written last so it captures every earlier write,
		// version.json included.
		newManifestStep(),
		// Last of all: everything the shell reads must already be in place.
		newStartStep(),
	)

	// Removal runs in the reverse order of installation: services stop before
	// the files that define them go, and the record of the install is the last
	// thing to be discarded.
	installer.Register(installer.OpUninstall,
		newStopServicesStep(),
		newUnhookMangoStep(),
		newRemoveFilesStep(),
		newRemoveVenvStep(),
		newReportPackagesStep(),
		newForgetStep(),
	)
}
