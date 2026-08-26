package steps

import (
	"strings"
	"testing"
)

func TestStripPathEntryRemovesOnlyOurLines(t *testing.T) {
	profile := strings.Join([]string{
		"# the user's own profile",
		"export EDITOR=nvim",
		"export PATH=\"$HOME/bin:$PATH\"",
		"",
		pathMarker,
		"export PATH=\"/home/u/.local/bin:$PATH\"",
		"",
		"echo hello",
	}, "\n")

	cleaned, removed := stripPathEntry(profile)

	if removed != 2 {
		t.Errorf("removed %d lines, want the marker and the export", removed)
	}
	for _, want := range []string{"export EDITOR=nvim", "export PATH=\"$HOME/bin:$PATH\"", "echo hello"} {
		if !strings.Contains(cleaned, want) {
			t.Errorf("stripping removed the user's own line %q", want)
		}
	}
	if strings.Contains(cleaned, "/home/u/.local/bin") {
		t.Error("our export survived")
	}
}

func TestStripPathEntryLeavesAnEditedLineAlone(t *testing.T) {
	// If the line under our marker is no longer an export, someone changed it
	// and it is theirs to keep.
	profile := pathMarker + "\nsource ~/my-own-setup.sh\n"

	cleaned, removed := stripPathEntry(profile)

	if removed != 1 {
		t.Errorf("removed %d lines, want only the marker", removed)
	}
	if !strings.Contains(cleaned, "my-own-setup.sh") {
		t.Error("an edited line was removed along with the marker")
	}
}

func TestProfileTargetMatchesTheShell(t *testing.T) {
	// zsh reads .zprofile at login; .zshrc may never be read by a graphical
	// session, which is exactly when the launcher has to be findable.
	if got := profileFor("zsh"); !strings.HasSuffix(got, ".zprofile") {
		t.Errorf("profileFor(zsh) = %s, want .zprofile", got)
	}
	for _, shell := range []string{"bash", "sh", "dash", ""} {
		if got := profileFor(shell); !strings.HasSuffix(got, ".profile") {
			t.Errorf("profileFor(%q) = %s, want .profile", shell, got)
		}
	}
}
