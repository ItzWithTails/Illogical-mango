package v2

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestRussianUIAndCustomKeyboardEntry(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Language = "ru"
	u := NewUI(cfg, nil)
	if got := u.welcomeView(); !strings.Contains(got, "Установка") {
		t.Fatalf("Russian welcome was not selected:\n%s", got)
	}
	u.stage = stageConfigure
	u.cursor = 4
	u.updateKey("e")
	for _, key := range []string{"u", "s", ",", "d", "e", "enter"} {
		u.updateEditor(key)
	}
	if u.Config.KeyboardLayout != "us,de" || u.editing {
		t.Fatalf("custom layout = %q, editing = %v", u.Config.KeyboardLayout, u.editing)
	}
}

func TestReviewViewportFitsSmallTerminal(t *testing.T) {
	u := NewUI(DefaultConfig(), nil)
	u.height = 12
	u.Plan = &Plan{Config: u.Config}
	for i := 0; i < 100; i++ {
		u.Plan.Actions = append(u.Plan.Actions, Action{Kind: Write, Path: strings.Repeat("x", i%10+1)})
	}
	view := u.reviewView()
	if lines := strings.Count(view, "\n") + 1; lines > 8 {
		t.Fatalf("small-terminal review rendered %d lines:\n%s", lines, view)
	}
	if !strings.Contains(view, "/") {
		t.Fatal("scroll position is not visible")
	}
	u.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	if u.height != 10 || u.width != 40 {
		t.Fatal("window resize was ignored")
	}
}

func TestEquivalentCommandIncludesOperationAndQuotesPaths(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Operation = Update
	cfg.Home = "/home/test user"
	cfg.Repo = "/tmp/source tree"
	cfg.Packages = false
	got := cfg.CommandLine()
	for _, want := range []string{"ilmango-installer-v2 update", "--home '/home/test user'", "--repo '/tmp/source tree'", "--no-packages", "--yes"} {
		if !strings.Contains(got, want) {
			t.Errorf("command %q lacks %q", got, want)
		}
	}
}
