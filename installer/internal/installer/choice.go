package installer

import "sort"

// Choice is an installation setting with several named values, as opposed to
// the on/off Options. It exists so a decision like "which AUR helper" is data
// in a catalogue rather than a special case threaded through the code.
type Choice struct {
	ID          OptionID
	Group       OptionGroup
	Title       string
	Description string

	// Values are offered in order; the first is what a cycle starts from.
	Values []ChoiceValue
	// Default is the value used when the user expresses no preference.
	Default string
	// Requires names an option that must be on for this choice to matter.
	Requires OptionID
}

// ChoiceValue is one selectable value.
type ChoiceValue struct {
	Value  string
	Label  string
	Detail string
	// Warning is shown on the review screen when this value is the one
	// selected. It is per value, not per choice, because what deserves a
	// warning is usually one of the answers rather than the question.
	Warning string
}

// AUR helper choice values. They are referenced by the package layer, so the
// strings are a contract rather than free text.
const (
	AURAuto               = "auto"
	AURParu               = "paru"
	AURYay                = "yay"
	AURNone               = "none"
	OptAURHelper OptionID = "aur-helper"
)

// Keybind choice values.
const (
	KeybindsFull           = "full"
	KeybindsShell          = "shell"
	OptKeybinds   OptionID = "keybinds"
)

// System upgrade choice values.
const (
	UpgradeFull               = "full"
	UpgradeSkip               = "skip"
	OptSystemUpgrade OptionID = "system-upgrade"
)

// choiceCatalog is the ordered set of choices. Extend it here; the picker and
// the CLI both render whatever it contains.
var choiceCatalog = []Choice{
	{
		ID:          OptAURHelper,
		Group:       GroupDependencies,
		Title:       "AUR helper",
		Description: "Several dependencies live only in the AUR, and pacman alone cannot build them.",
		Default:     AURAuto,
		Requires:    OptDependencies,
		Values: []ChoiceValue{
			{Value: AURAuto, Label: "automatic", Detail: "use paru, then yay, then plain pacman"},
			{Value: AURParu, Label: "paru", Detail: "require paru"},
			{Value: AURYay, Label: "yay", Detail: "require yay"},
			{Value: AURNone, Label: "none", Detail: "pacman only; AUR packages will be reported as missing"},
		},
	},
	{
		ID:          OptKeybinds,
		Group:       GroupBehaviour,
		Title:       "Keybinds",
		Description: "The shell's own panels are always bound. This decides whether the conventional desktop keys come with them — a terminal on Super+T, files on Super+E, workspaces on Super+scroll, and the window management to go with them.",
		Default:     KeybindsFull,
		Requires:    OptMango,
		Values: []ChoiceValue{
			{
				Value:   KeybindsFull,
				Label:   "full",
				Detail:  "the conventional set, window management included",
				Warning: "The conventional keybinds define window management, so where they collide with bindings you already have, they win. Choose keybinds=shell to keep only the shell's own panel keys.",
			},
			{Value: KeybindsShell, Label: "shell", Detail: "only the shell's panels, media and capture keys"},
		},
	},
	{
		ID:          OptSystemUpgrade,
		Group:       GroupDependencies,
		Title:       "System upgrade",
		Description: "Arch does not support installing new packages against freshly synced databases without also upgrading: the new package pulls libraries the installed ones do not expect. Skipping the upgrade is safe, but the databases stay as they are, so a machine left alone for a long time may not find some packages.",
		Default:     UpgradeFull,
		Requires:    OptDependencies,
		Values: []ChoiceValue{
			{
				Value:   UpgradeFull,
				Label:   "full",
				Detail:  "upgrade the system first, the way the distribution expects",
				Warning: "Your whole system will be upgraded before anything is installed, not just the packages this shell needs.",
			},
			{Value: UpgradeSkip, Label: "skip", Detail: "install against the databases already on the machine"},
		},
	},
}

// Choices returns the choice catalogue in presentation order.
func Choices() []Choice {
	out := make([]Choice, len(choiceCatalog))
	copy(out, choiceCatalog)
	return out
}

// LookupChoice finds a choice descriptor by ID.
func LookupChoice(id OptionID) (Choice, bool) {
	for _, c := range choiceCatalog {
		if c.ID == id {
			return c, true
		}
	}
	return Choice{}, false
}

// Valid reports whether value is one this choice offers.
func (c Choice) Valid(value string) bool {
	for _, v := range c.Values {
		if v.Value == value {
			return true
		}
	}
	return false
}

// Label returns the human-readable name of a value.
func (c Choice) Label(value string) string {
	for _, v := range c.Values {
		if v.Value == value {
			return v.Label
		}
	}
	return value
}

// Warning returns the review-screen warning attached to a value, if any.
func (c Choice) Warning(value string) string {
	for _, v := range c.Values {
		if v.Value == value {
			return v.Warning
		}
	}
	return ""
}

// Detail returns the explanation attached to a value.
func (c Choice) Detail(value string) string {
	for _, v := range c.Values {
		if v.Value == value {
			return v.Detail
		}
	}
	return ""
}

// next returns the value after the given one, wrapping around.
func (c Choice) next(current string, delta int) string {
	if len(c.Values) == 0 {
		return current
	}
	index := 0
	for i, v := range c.Values {
		if v.Value == current {
			index = i
		}
	}
	index = (index + delta + len(c.Values)) % len(c.Values)
	return c.Values[index].Value
}

// ChoiceNames lists every choice ID, sorted, for help text.
func ChoiceNames() []string {
	out := make([]string, 0, len(choiceCatalog))
	for _, c := range choiceCatalog {
		out = append(out, string(c.ID))
	}
	sort.Strings(out)
	return out
}
