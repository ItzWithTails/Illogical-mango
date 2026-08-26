package installer

// catalog is the ordered set of installation options. It is the single source
// of truth for the picker UI, the CLI flags, and the environment handed to the
// shell phases — extend it here and every consumer follows.
var catalog = []Option{
	{
		ID:          OptDependencies,
		Group:       GroupStages,
		Title:       "Dependencies",
		Description: "Packages, Quickshell runtime and compositor requirements.",
		Default:     true,
	},
	{
		ID:          OptSystemSetup,
		Group:       GroupStages,
		Title:       "System configuration",
		Description: "User groups, systemd services and device permissions.",
		Default:     true,
		Risky:       true,
	},
	{
		ID:          OptConfigFiles,
		Group:       GroupStages,
		Title:       "Config files",
		Description: "Shell config, wallpapers, theming inputs and dotfiles.",
		Default:     true,
	},

	{
		ID:          OptWallpapers,
		Group:       GroupExtras,
		Title:       "Wallpaper pack",
		Description: "About 148 wallpapers, roughly 630 MiB. Existing files of the same name are kept.",
		Default:     false,
	},
	{
		ID:          OptIconTheme,
		Group:       GroupExtras,
		Title:       "Monochrome icon theme",
		Description: "YAMIS by dirn-typo, about 23 MiB. Installed alongside your icons; nothing is switched over.",
		Default:     false,
	},
	{
		ID:          OptMascot,
		Group:       GroupExtras,
		Title:       "Mascot art pack",
		Description: "354 poses and animations for Kira, about 32 MiB. The mascot itself stays off until you enable it.",
		Default:     false,
		Requires:    OptConfigFiles,
	},
	{
		ID:          OptSDDMTheme,
		Group:       GroupExtras,
		Title:       "Login screen theme",
		Description: "Applies the ii-pixel theme to SDDM. Needs root and edits the display manager's configuration.",
		Default:     false,
		Risky:       true,
	},

	{
		ID:          OptAudio,
		Group:       GroupDependencies,
		Title:       "Audio stack",
		Description: "PipeWire, WirePlumber and the cava visualiser.",
		Default:     true,
		Requires:    OptDependencies,
	},
	{
		ID:          OptToolkit,
		Group:       GroupDependencies,
		Title:       "Input toolkit",
		Description: "ydotool and backlight helpers for keybinds and brightness.",
		Default:     true,
		Requires:    OptDependencies,
	},
	{
		ID:          OptScreenCapture,
		Group:       GroupDependencies,
		Title:       "Screen capture",
		Description: "Screenshot, region-select and screen recording tools.",
		Default:     true,
		Requires:    OptDependencies,
	},
	{
		ID:          OptFonts,
		Group:       GroupDependencies,
		Title:       "Fonts and theming",
		Description: "Material Symbols, Rubik, and the Qt/GTK theme bridges.",
		Default:     true,
		Requires:    OptDependencies,
	},

	{
		ID:          OptMango,
		Group:       GroupStages,
		Title:       "Hook into the mango config",
		Description: "Add one source line for the shell's keybinds. Never rewrites your own config.",
		Default:     true,
		Requires:    OptConfigFiles,
	},
	{
		ID:          OptPathEntry,
		Group:       GroupBehaviour,
		Title:       "Put ilmango on your PATH",
		Description: "Adds one line to your shell profile if the launcher's directory is missing from PATH. Without it the command installs but cannot be run.",
		Default:     true,
		Risky:       true,
	},
	{
		ID:          OptBackup,
		Group:       GroupBehaviour,
		Title:       "Back up existing configs",
		Description: "Snapshot anything this installer is about to overwrite.",
		Default:     true,
		Requires:    OptConfigFiles,
	},
	{
		ID:          OptStartNow,
		Group:       GroupBehaviour,
		Title:       "Start the shell when finished",
		Description: "Useful when installing from a session already running. Otherwise it starts at next login.",
		Default:     false,
		Requires:    OptConfigFiles,
	},
	{
		ID:          OptDefaultShell,
		Group:       GroupBehaviour,
		Title:       "Make Fish the login shell",
		Description: "Runs chsh for the current user. Off unless you ask for it.",
		Default:     false,
		Requires:    OptSystemSetup,
		Risky:       true,
	},
}

// Options returns the installation option catalog in presentation order.
func Options() []Option {
	out := make([]Option, len(catalog))
	copy(out, catalog)
	return out
}

// LookupOption finds an option descriptor by ID.
func LookupOption(id OptionID) (Option, bool) {
	for _, o := range catalog {
		if o.ID == id {
			return o, true
		}
	}
	return Option{}, false
}

// Groups returns the option groups in presentation order, deduplicated.
func Groups() []OptionGroup {
	var out []OptionGroup
	seen := map[OptionGroup]bool{}
	for _, o := range catalog {
		if !seen[o.Group] {
			seen[o.Group] = true
			out = append(out, o.Group)
		}
	}
	return out
}
