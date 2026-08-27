package ui

import (
	"strings"
	"testing"

	"ilmango/internal/installer"
	"ilmango/internal/system"
	"ilmango/internal/ui/theme"
)

func newPackagesFixture(t *testing.T) *packagesScreen {
	t.Helper()
	session := &Session{
		Width: 90, Height: 30,
		Config: installer.NewConfig(),
		Theme:  theme.New(),
		Distro: system.Distro{Name: "Arch Linux", Family: "arch"},
	}
	return newPackagesScreen(session).(*packagesScreen)
}

func TestPackagePickerExcludesAndRestores(t *testing.T) {
	s := newPackagesFixture(t)
	name := s.rows[s.cursor].name
	if name == "" {
		t.Fatal("the cursor started on a group heading; it must start on a package")
	}

	s.toggle()
	if !s.session.Config.PackageSkipped(name) {
		t.Fatalf("%s was not excluded", name)
	}
	s.toggle()
	if s.session.Config.PackageSkipped(name) {
		t.Fatalf("%s was not put back", name)
	}
}

func TestPackagePickerNeverRestsOnAHeading(t *testing.T) {
	// Headings are not selectable; space on one would silently do nothing.
	s := newPackagesFixture(t)
	for i := 0; i < len(s.rows)+5; i++ {
		if s.rows[s.cursor].name == "" {
			t.Fatalf("the cursor landed on the %q heading after %d moves", s.rows[s.cursor].group, i)
		}
		s.move(1)
	}
}

func TestPackagePickerActsOnOneGroupAtATime(t *testing.T) {
	s := newPackagesFixture(t)
	group := s.rows[s.cursor].group

	s.setGroup(true)

	var elsewhere int
	for _, row := range s.rows {
		if row.name == "" {
			continue
		}
		skipped := s.session.Config.PackageSkipped(row.name)
		if row.group == group && !skipped {
			t.Errorf("%s in %s was left included", row.name, group)
		}
		if row.group != group && skipped {
			elsewhere++
		}
	}
	if elsewhere > 0 {
		t.Errorf("%d packages outside %s were excluded too", elsewhere, group)
	}
}

func TestPackagePickerWindowKeepsTheCursorVisible(t *testing.T) {
	// The catalogue is far longer than any terminal. Whatever the cursor is
	// on has to be on screen, or the selection is invisible.
	s := newPackagesFixture(t)
	for i := 0; i < len(s.rows); i++ {
		first, last := s.window()
		if s.cursor < first || s.cursor > last {
			t.Fatalf("cursor at %d is outside the drawn window %d..%d", s.cursor, first, last)
		}
		s.move(1)
	}
}

func TestPackagePickerShowsExcludedCriticalPackages(t *testing.T) {
	s := newPackagesFixture(t)
	s.session.Config.SkipPackage("mangowm", true)

	// Put the cursor near mangowm so it falls inside the drawn window.
	for i, row := range s.rows {
		if row.name == "mangowm" {
			s.cursor = i
			break
		}
	}

	if body := s.View(); !strings.Contains(body, "the shell needs this") {
		t.Fatalf("excluding a critical package is not flagged:\n%s", body)
	}
}

func TestPackagePickerSurvivesADistributionWithNoList(t *testing.T) {
	// NixOS and anything unrecognised have no package list. The screen still
	// has to accept every key it advertises rather than panicking on an empty
	// catalogue.
	session := &Session{
		Width: 90, Height: 30,
		Config: installer.NewConfig(),
		Theme:  theme.New(),
		Distro: system.Distro{Name: "NixOS", Family: "nixos"},
	}
	s := newPackagesScreen(session).(*packagesScreen)

	s.move(1)
	s.toggle()
	s.setGroup(true)
	if body := s.View(); !strings.Contains(body, "No package list") {
		t.Fatalf("an empty catalogue rendered as %q", body)
	}
}
