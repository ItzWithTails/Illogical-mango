package installer

// OptionID uniquely identifies a user-tunable installation choice.
//
// Option IDs are part of the CLI contract: they are accepted by --enable and
// --disable, so renaming one is a breaking change.
type OptionID string

// Installation options. Adding a new one requires appending a descriptor to
// the catalog in options.go — no UI code needs to change.
const (
	OptDependencies  OptionID = "deps"
	OptSystemSetup   OptionID = "setups"
	OptConfigFiles   OptionID = "files"
	OptAudio         OptionID = "audio"
	OptToolkit       OptionID = "toolkit"
	OptScreenCapture OptionID = "screencapture"
	OptFonts         OptionID = "fonts"
	OptBackup        OptionID = "backup"
	OptDefaultShell  OptionID = "default-shell"
	OptMango         OptionID = "mango"
	OptStartNow      OptionID = "start"
)

// OptionGroup labels a related run of options in the picker.
type OptionGroup string

const (
	GroupStages       OptionGroup = "Installation stages"
	GroupDependencies OptionGroup = "Dependency groups"
	GroupBehaviour    OptionGroup = "Behaviour"
)

// Option describes a single installation choice: how it is presented, how it
// defaults, and how it maps onto the shell installer's environment contract.
type Option struct {
	ID          OptionID
	Group       OptionGroup
	Title       string
	Description string

	// Default is the value used when the user expresses no preference.
	Default bool

	// Requires names an option that must be enabled for this one to have any
	// effect. Dependent options are shown as inert while their parent is off.
	Requires OptionID

	// Risky marks choices that change state outside the config directory and
	// therefore deserve visual emphasis on the review screen.
	Risky bool
}

// Enabled reports whether opt is on in cfg, accounting for its parent option.
func (o Option) enabledIn(c Config) bool {
	if o.Requires != "" && !c.values[o.Requires] {
		return false
	}
	return c.values[o.ID]
}
