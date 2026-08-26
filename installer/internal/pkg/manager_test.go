package pkg

import (
	"context"
	"strings"
	"testing"

	"ilmango/internal/run"
)

func TestBatchBudgetScalesWithTheWorkAndIsCapped(t *testing.T) {
	small := BatchBudget(5)
	large := BatchBudget(120)
	huge := BatchBudget(5000)

	if !(small < large) {
		t.Fatalf("BatchBudget(5) = %s is not smaller than BatchBudget(120) = %s", small, large)
	}
	if small < batchOverhead {
		t.Fatalf("BatchBudget(5) = %s dips below the fixed overhead %s", small, batchOverhead)
	}
	if huge != batchCeiling {
		t.Fatalf("BatchBudget(5000) = %s, want the ceiling %s", huge, batchCeiling)
	}
}

func TestArchManagersNeverSyncWithoutUpgrading(t *testing.T) {
	// "pacman -Sy" followed by an install is the partial upgrade Arch documents
	// as unsupported, and it is an easy thing to reintroduce by copying a line
	// in the table above.
	for _, m := range managers["arch"] {
		for _, arg := range m.RefreshArgs {
			if strings.HasPrefix(arg, "-Sy") {
				t.Fatalf("%s refreshes with %q; on Arch a database sync must come with an upgrade", m.Name, arg)
			}
		}
		if !m.CanUpgrade() {
			t.Fatalf("%s offers no upgrade step, so it can never refresh its databases", m.Name)
		}
	}
}

func TestRefreshDoesNothingWhenTheUpgradeIsDeclined(t *testing.T) {
	// Declining the upgrade must not quietly fall back to the unsafe sync.
	var logged []string
	r := run.Runner{Log: func(line string) { logged = append(logged, line) }}

	for _, m := range managers["arch"] {
		if err := m.Refresh(context.Background(), &r, false); err != nil {
			t.Fatalf("%s: Refresh(upgrade=false) error = %v", m.Name, err)
		}
	}
	if len(logged) != 0 {
		t.Fatalf("declining the upgrade still planned %v", logged)
	}
}
