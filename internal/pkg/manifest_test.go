package pkg

import (
	"sort"
	"testing"
)

func TestPackagesDeduplicatesAcrossGroups(t *testing.T) {
	// Fonts and Quickshell both pull in kvantum on Arch.
	got := Packages("arch", GroupQuickshell, GroupFonts)

	seen := map[string]int{}
	for _, name := range got {
		seen[name]++
	}
	for name, count := range seen {
		if count > 1 {
			t.Errorf("%s appears %d times, want once", name, count)
		}
	}
	if !sort.StringsAreSorted(got) {
		t.Error("package list should be sorted for a stable transcript")
	}
}

func TestPackagesUnknownFamilyIsEmpty(t *testing.T) {
	if got := Packages("plan9", GroupCore); len(got) != 0 {
		t.Errorf("unknown family returned %d packages, want none", len(got))
	}
}

func TestEveryFamilyCoversEveryGroup(t *testing.T) {
	groups := []Group{GroupCore, GroupQuickshell, GroupAudio, GroupToolkit, GroupScreenCapture, GroupFonts}

	for _, family := range Families() {
		for _, group := range groups {
			if len(Packages(family, group)) == 0 {
				t.Errorf("family %s has no packages for group %s", family, group)
			}
		}
	}
}

func TestKnownFamily(t *testing.T) {
	if !KnownFamily("arch") {
		t.Error("arch should be a known family")
	}
	if KnownFamily("plan9") {
		t.Error("plan9 should not be a known family")
	}
}

func TestCriticalPackagesExistInSomeFamily(t *testing.T) {
	// A critical name that no family installs is a typo, and the installer
	// would never report it as missing.
	for _, name := range parsed.Critical {
		var found bool
		for family := range parsed.Families {
			for _, group := range parsed.Families[family] {
				for _, candidate := range group {
					if candidate == name {
						found = true
					}
				}
			}
		}
		if !found {
			t.Errorf("critical package %q is not installed by any family", name)
		}
	}
}

func TestSplitCriticalPartitions(t *testing.T) {
	critical, optional := SplitCritical([]string{"quickshell", "songrec", "mangowm"})

	if len(critical) != 2 {
		t.Errorf("critical = %v, want quickshell and mangowm", critical)
	}
	if len(optional) != 1 || optional[0] != "songrec" {
		t.Errorf("optional = %v, want [songrec]", optional)
	}
}

func TestArchInstallsMangoNotNiri(t *testing.T) {
	// This project targets MangoWM; niri is inherited from upstream and the
	// README states it is unsupported. Installing it would be wrong.
	core := Packages("arch", GroupCore)

	var hasMango, hasNiri bool
	for _, name := range core {
		switch name {
		case "mangowm":
			hasMango = true
		case "niri":
			hasNiri = true
		}
	}

	if !hasMango {
		t.Error("the arch core group must install the mango compositor")
	}
	if hasNiri {
		t.Error("niri is unsupported and must not be installed")
	}
}

func TestFindManagerPreferringRejectsUnknownHelper(t *testing.T) {
	if _, err := FindManagerPreferring("arch", "pikaur"); err == nil {
		t.Error("a helper this installer does not drive must be refused")
	}
}

func TestNoneExcludesAURHelpers(t *testing.T) {
	// "none" must never yield a helper, whatever happens to be installed.
	m, err := FindManagerPreferring("arch", "none")
	if err != nil {
		t.Skipf("no package manager on this machine: %v", err)
	}
	if m.IsAURHelper() {
		t.Errorf("chose %s despite the AUR being ruled out", m.Name)
	}
}

func TestIsAURHelper(t *testing.T) {
	for _, m := range managers["arch"] {
		want := m.Name != "pacman"
		if got := m.IsAURHelper(); got != want {
			t.Errorf("%s: IsAURHelper() = %v, want %v", m.Name, got, want)
		}
	}
}
