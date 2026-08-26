package installer

import (
	"strings"
	"testing"
)

func TestNewConfigUsesCatalogDefaults(t *testing.T) {
	cfg := NewConfig()

	for _, o := range Options() {
		if got := cfg.Get(o.ID); got != o.Default {
			t.Errorf("option %s: got %v, want catalog default %v", o.ID, got, o.Default)
		}
	}
}

func TestEffectiveRespectsParentOption(t *testing.T) {
	cfg := NewConfig()

	if !cfg.Effective(OptAudio) {
		t.Fatal("audio should be effective while dependencies are on")
	}

	cfg.Set(OptDependencies, false)

	if cfg.Effective(OptAudio) {
		t.Error("audio must not be effective once dependencies are off")
	}
	if !cfg.Get(OptAudio) {
		t.Error("the user's own audio choice must be remembered, not cleared")
	}
}

func TestCommandLineOnlyReportsDeviations(t *testing.T) {
	cfg := NewConfig()

	if got := cfg.CommandLine(); got != "ilmango-installer -y" {
		t.Errorf("default config rendered %q, want no extra flags", got)
	}

	cfg.Set(OptAudio, false)
	cfg.Set(OptDefaultShell, true)

	got := cfg.CommandLine()
	for _, want := range []string{"--enable default-shell", "--disable audio"} {
		if !strings.Contains(got, want) {
			t.Errorf("CommandLine() = %q, missing %q", got, want)
		}
	}
}

func TestCommandLineRoundTrips(t *testing.T) {
	original := NewConfig()
	original.Set(OptFonts, false)
	original.Set(OptDefaultShell, true)

	// Re-applying the rendered flags must reproduce the same configuration.
	replayed := NewConfig()
	if err := replayed.Apply([]string{"default-shell"}, true); err != nil {
		t.Fatal(err)
	}
	if err := replayed.Apply([]string{"fonts"}, false); err != nil {
		t.Fatal(err)
	}

	if original.CommandLine() != replayed.CommandLine() {
		t.Errorf("round trip diverged:\n  %s\n  %s", original.CommandLine(), replayed.CommandLine())
	}
}

func TestApplyRejectsUnknownOption(t *testing.T) {
	cfg := NewConfig()

	if err := cfg.Apply([]string{"nonsense"}, true); err == nil {
		t.Fatal("expected an error for an unknown option name")
	}
	if err := cfg.Apply([]string{"audio", ""}, false); err != nil {
		t.Fatalf("Apply with a blank entry: %v", err)
	}
	if cfg.Get(OptAudio) {
		t.Error("audio should have been switched off")
	}
}

func TestChoiceDefaultsAndSelection(t *testing.T) {
	cfg := NewConfig()

	if got := cfg.Choice(OptAURHelper); got != AURAuto {
		t.Errorf("default aur-helper = %q, want %q", got, AURAuto)
	}
	if err := cfg.SetChoice(OptAURHelper, AURParu); err != nil {
		t.Fatalf("SetChoice: %v", err)
	}
	if got := cfg.Choice(OptAURHelper); got != AURParu {
		t.Errorf("aur-helper = %q, want %q", got, AURParu)
	}
}

func TestSetChoiceRejectsUnknownValue(t *testing.T) {
	cfg := NewConfig()

	if err := cfg.SetChoice(OptAURHelper, "pikaur"); err == nil {
		t.Error("a value the choice does not offer must be refused")
	}
	if err := cfg.SetChoice("nonsense", "x"); err == nil {
		t.Error("an unknown choice must be refused")
	}
	if got := cfg.Choice(OptAURHelper); got != AURAuto {
		t.Errorf("a rejected selection changed the value to %q", got)
	}
}

func TestCycleChoiceWrapsBothWays(t *testing.T) {
	cfg := NewConfig()
	choice, _ := LookupChoice(OptAURHelper)

	// Forward through every value returns to where it started.
	for range choice.Values {
		cfg.CycleChoice(OptAURHelper, 1)
	}
	if got := cfg.Choice(OptAURHelper); got != AURAuto {
		t.Errorf("a full cycle ended on %q, want the starting value", got)
	}

	if got := cfg.CycleChoice(OptAURHelper, -1); got != choice.Values[len(choice.Values)-1].Value {
		t.Errorf("cycling backwards from the first value gave %q", got)
	}
}

func TestCommandLineIncludesChangedChoices(t *testing.T) {
	cfg := NewConfig()

	if strings.Contains(cfg.CommandLine(), "--set") {
		t.Error("a default choice should not be rendered")
	}

	if err := cfg.SetChoice(OptAURHelper, AURNone); err != nil {
		t.Fatal(err)
	}
	if got := cfg.CommandLine(); !strings.Contains(got, "--set aur-helper=none") {
		t.Errorf("CommandLine() = %q, missing the chosen helper", got)
	}
}
