package installer

import (
	"fmt"
	"sort"
	"strings"
)

// Config is the state of every installation option.
//
// It says what to do, never where: the checkout being installed from lives on
// Env.Repo, so a configuration can be rendered, compared and replayed without
// dragging a filesystem path along.
//
// The zero value is not usable; construct one with NewConfig.
type Config struct {
	// DryRun walks every step and its decisions without letting any of them
	// modify the machine.
	DryRun bool

	values  map[OptionID]bool
	choices map[OptionID]string
	// skipped names individual packages the user does not want. It holds
	// exclusions rather than inclusions so that a package added to the
	// catalogue later is installed by default, the way every other new
	// dependency has always been.
	skipped map[string]bool
}

// NewConfig returns a configuration with every option at its catalog default.
func NewConfig() Config {
	values := make(map[OptionID]bool, len(catalog))
	for _, o := range catalog {
		values[o.ID] = o.Default
	}
	choices := make(map[OptionID]string, len(choiceCatalog))
	for _, c := range choiceCatalog {
		choices[c.ID] = c.Default
	}
	return Config{values: values, choices: choices, skipped: map[string]bool{}}
}

// SkipPackage excludes a package from the install, or puts it back.
func (c *Config) SkipPackage(name string, skip bool) {
	if c.skipped == nil {
		c.skipped = map[string]bool{}
	}
	if skip {
		c.skipped[name] = true
		return
	}
	delete(c.skipped, name)
}

// PackageSkipped reports whether a package has been excluded.
func (c Config) PackageSkipped(name string) bool { return c.skipped[name] }

// SkippedPackages lists the exclusions, sorted.
func (c Config) SkippedPackages() []string {
	out := make([]string, 0, len(c.skipped))
	for name := range c.skipped {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// KeepPackages returns names with the excluded ones removed.
func (c Config) KeepPackages(names []string) []string {
	if len(c.skipped) == 0 {
		return names
	}
	out := make([]string, 0, len(names))
	for _, name := range names {
		if !c.skipped[name] {
			out = append(out, name)
		}
	}
	return out
}

// Choice returns the selected value of a choice.
func (c Config) Choice(id OptionID) string { return c.choices[id] }

// SetChoice selects a value, rejecting one the choice does not offer.
func (c *Config) SetChoice(id OptionID, value string) error {
	choice, ok := LookupChoice(id)
	if !ok {
		return fmt.Errorf("unknown choice %q (known: %s)", id, strings.Join(ChoiceNames(), ", "))
	}
	if !choice.Valid(value) {
		var offered []string
		for _, v := range choice.Values {
			offered = append(offered, v.Value)
		}
		return fmt.Errorf("%q is not a value for %s (offered: %s)", value, id, strings.Join(offered, ", "))
	}

	c.choices[id] = value
	return nil
}

// CycleChoice moves to the next or previous value and returns it.
func (c *Config) CycleChoice(id OptionID, delta int) string {
	choice, ok := LookupChoice(id)
	if !ok {
		return ""
	}
	c.choices[id] = choice.next(c.choices[id], delta)
	return c.choices[id]
}

// Get reports whether the option is switched on, ignoring its parent. Use
// Effective when you care whether the option will actually take effect.
func (c Config) Get(id OptionID) bool { return c.values[id] }

// Effective reports whether the option is on and its parent chain permits it.
func (c Config) Effective(id OptionID) bool {
	o, ok := LookupOption(id)
	if !ok {
		return false
	}
	return o.enabledIn(c)
}

// Set switches an option on or off.
func (c *Config) Set(id OptionID, on bool) { c.values[id] = on }

// Toggle flips an option and returns its new state.
func (c *Config) Toggle(id OptionID) bool {
	c.values[id] = !c.values[id]
	return c.values[id]
}

// NeedsPrivileges reports whether the configuration includes work that has to
// run as root. The UI uses this to obtain a credential before it takes over
// the terminal, rather than letting sudo prompt where nobody can see it.
func (c Config) NeedsPrivileges() bool {
	return c.Effective(OptDependencies) || c.Effective(OptSystemSetup)
}

// CommandLine renders the invocation that reproduces this configuration. It
// is shown on the review screen so a choice made by clicking around can be
// repeated in a script or a bug report.
func (c Config) CommandLine() string {
	args := []string{"ilmango-installer", "-y"}

	var enabled, disabled []string
	for _, o := range catalog {
		if c.values[o.ID] == o.Default {
			continue
		}
		if c.values[o.ID] {
			enabled = append(enabled, string(o.ID))
		} else {
			disabled = append(disabled, string(o.ID))
		}
	}

	if len(enabled) > 0 {
		args = append(args, "--enable", strings.Join(enabled, ","))
	}
	if len(disabled) > 0 {
		args = append(args, "--disable", strings.Join(disabled, ","))
	}
	for _, choice := range choiceCatalog {
		if value := c.choices[choice.ID]; value != choice.Default {
			args = append(args, "--set", string(choice.ID)+"="+value)
		}
	}
	if skipped := c.SkippedPackages(); len(skipped) > 0 {
		args = append(args, "--without", strings.Join(skipped, ","))
	}
	if c.DryRun {
		args = append(args, "--dry-run")
	}
	return strings.Join(args, " ")
}

// Apply switches the named options on or off, as accepted by --enable and
// --disable. Unknown names are reported rather than silently ignored.
func (c *Config) Apply(names []string, on bool) error {
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := LookupOption(OptionID(name)); !ok {
			return fmt.Errorf("unknown option %q (known: %s)", name, strings.Join(OptionNames(), ", "))
		}
		c.values[OptionID(name)] = on
	}
	return nil
}

// OptionNames lists every option ID, sorted, for help text and error messages.
func OptionNames() []string {
	out := make([]string, 0, len(catalog))
	for _, o := range catalog {
		out = append(out, string(o.ID))
	}
	sort.Strings(out)
	return out
}
