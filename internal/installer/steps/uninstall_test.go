package steps

import (
	"strings"
	"testing"
)

func TestStripMangoHookRemovesOnlyOurLines(t *testing.T) {
	config := strings.Join([]string{
		"# the user's own window management",
		"tagrule=isfloating:1",
		"bind=SUPER,Return,spawn,foot",
		"",
		"# Added by Illogical-mango: shell keybinds and autostart.",
		"source-optional=/home/u/.config/mango/ilmango.conf",
	}, "\n")

	cleaned, removed := stripMangoHook(config)

	if removed != 2 {
		t.Errorf("removed %d lines, want the comment and the source line", removed)
	}
	for _, want := range []string{"tagrule=isfloating:1", "bind=SUPER,Return,spawn,foot", "# the user's own window management"} {
		if !strings.Contains(cleaned, want) {
			t.Errorf("stripping removed the user's own line %q", want)
		}
	}
	if strings.Contains(cleaned, "source-optional") {
		t.Error("our source line survived")
	}
}

func TestStripMangoHookLeavesForeignSourceLines(t *testing.T) {
	// Another tool's include must not be mistaken for ours.
	config := "source-optional=/home/u/.config/mango/someone-else.conf\n"

	cleaned, removed := stripMangoHook(config)

	if removed != 0 {
		t.Errorf("removed %d lines, want none", removed)
	}
	if cleaned != config {
		t.Errorf("config was altered: %q", cleaned)
	}
}

func TestStripMangoHookIsIdempotent(t *testing.T) {
	config := "bind=SUPER,q,killclient\n"

	cleaned, removed := stripMangoHook(config)

	if removed != 0 || cleaned != config {
		t.Errorf("a config we never hooked was changed: %q (%d removed)", cleaned, removed)
	}
}
