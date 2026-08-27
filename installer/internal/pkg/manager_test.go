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

func TestProgressCountsPackagesAndIgnoresHooks(t *testing.T) {
	// pacman numbers its post-transaction hooks "(1/2)" but prints packages
	// as a bare verb. Counting the numbers would report hook progress as
	// package progress; counting names we asked for cannot.
	var seen []Progress
	watch := newProgressWatcher([]string{"bc", "jq", "ripgrep"}, func(p Progress) { seen = append(seen, p) })

	for _, line := range []string{
		":: Processing package changes...",
		"installing bc...",
		"reinstalling jq...",
		"installing libfoo...", // a dependency, not something we asked for
		":: Running post-transaction hooks...",
		"(1/2) Arming ConditionNeedsUpdate...",
		"(2/2) Updating the info directory file...",
		"upgrading ripgrep...",
	} {
		watch(line)
	}

	if len(seen) != 3 {
		t.Fatalf("reported %d packages, want 3: %+v", len(seen), seen)
	}
	for i, want := range []string{"bc", "jq", "ripgrep"} {
		if seen[i].Name != want {
			t.Errorf("report %d named %q, want %q", i, seen[i].Name, want)
		}
		if seen[i].Done != i+1 || seen[i].Total != 3 {
			t.Errorf("report %d counted %d/%d, want %d/3", i, seen[i].Done, seen[i].Total, i+1)
		}
	}
	if seen[2].Action != "upgrading" {
		t.Errorf("ripgrep was reported as %q, want the verb pacman used", seen[2].Action)
	}
}

func TestProgressCountsEachPackageOnce(t *testing.T) {
	// The retry pass runs the manager again, and a package installed in the
	// failed batch must not be counted a second time.
	var seen []Progress
	watch := newProgressWatcher([]string{"bc"}, func(p Progress) { seen = append(seen, p) })

	watch("installing bc...")
	watch("installing bc...")

	if len(seen) != 1 {
		t.Fatalf("reported %d times, want 1: %+v", len(seen), seen)
	}
}

func TestProgressReportsBuildsAndDownloadsWithoutCountingThem(t *testing.T) {
	// The AUR phase can run for a quarter of an hour before the first package
	// is installed. Saying nothing for that long is indistinguishable from
	// having hung, which is exactly what it looked like.
	var seen []Progress
	watch := newProgressWatcher([]string{"ttf-oxanium", "cava"}, func(p Progress) { seen = append(seen, p) })

	for _, line := range []string{
		"==> Making package: 38c3-styles 2-2 (Thu Aug 27 12:07:02 2026)",
		"  -> Downloading 38c3-styleguide-full-v2.zip...",
		"installing ttf-oxanium...",
	} {
		watch(strings.TrimSpace(line))
	}

	if len(seen) != 3 {
		t.Fatalf("reported %d events, want 3: %+v", len(seen), seen)
	}
	if seen[0].Action != "building" || seen[0].Name != "38c3-styles" {
		t.Errorf("first event = %+v, want building 38c3-styles", seen[0])
	}
	if seen[1].Action != "downloading" || seen[1].Name != "38c3-styleguide-full-v2.zip" {
		t.Errorf("second event = %+v, want the source being fetched", seen[1])
	}

	// A build is not a package from the list: one can pull in several, so
	// counting them would make the total a lie.
	if seen[0].Counted || seen[1].Counted {
		t.Error("a build or download was counted as an installed package")
	}
	if seen[0].Done != 0 || seen[1].Done != 0 {
		t.Errorf("the count moved during the build phase: %d, %d", seen[0].Done, seen[1].Done)
	}
	if !seen[2].Counted || seen[2].Done != 1 || seen[2].Total != 2 {
		t.Errorf("the installed package was not counted: %+v", seen[2])
	}
}
