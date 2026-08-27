package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"ilmango/internal/pkg"
	"ilmango/internal/ui/theme"
	"ilmango/internal/ui/widget"
)

// packagesScreen picks individual packages out of the install.
//
// The options screen decides in groups, which is the right granularity for
// almost everyone. This is the escape hatch for the rest: someone who wants
// the shell but not its media player should not have to choose between taking
// the whole audio group and taking none of it.
//
// It only ever excludes packages this installer would install. What those
// packages pull in themselves belongs to the package manager, and pretending
// otherwise would produce a list the installer cannot honour.
type packagesScreen struct {
	session *Session
	rows    []packageRow
	cursor  int
}

// packageRow is one line: either a group heading or a package under it.
type packageRow struct {
	group    pkg.Group
	name     string
	critical bool
}

func newPackagesScreen(session *Session) Screen {
	s := &packagesScreen{session: session}
	s.build()
	return s
}

// build flattens the catalogue into the rows the screen shows.
func (s *packagesScreen) build() {
	family := string(s.session.Distro.Family)
	s.rows = nil
	for _, group := range pkg.AllGroups() {
		names := pkg.Packages(family, group)
		if len(names) == 0 {
			continue
		}
		s.rows = append(s.rows, packageRow{group: group})
		for _, name := range names {
			s.rows = append(s.rows, packageRow{group: group, name: name, critical: pkg.IsCritical(name)})
		}
	}
	if s.cursor >= len(s.rows) {
		s.cursor = 0
	}
	// Never rest on a heading: the cursor exists to point at something that
	// can be toggled.
	if len(s.rows) > 0 && s.rows[s.cursor].name == "" {
		s.move(1)
	}
}

func (s *packagesScreen) Init() tea.Cmd { return nil }

func (s *packagesScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return s, nil
	}

	switch key.String() {
	case "q":
		return s, quit(true)
	case "esc":
		return s, navigate(StageOptions)
	case "enter":
		return s, navigate(StageReview)
	case "up", "k":
		s.move(-1)
	case "down", "j":
		s.move(1)
	case "space", "x":
		s.toggle()
	case "a":
		s.setGroup(false)
	case "n":
		s.setGroup(true)
	case "r":
		for _, name := range s.session.Config.SkippedPackages() {
			s.session.Config.SkipPackage(name, false)
		}
	}
	return s, nil
}

// move steps the cursor to the next selectable row, skipping headings.
func (s *packagesScreen) move(delta int) {
	if len(s.rows) == 0 {
		return
	}
	for i := 0; i < len(s.rows); i++ {
		s.cursor = (s.cursor + delta + len(s.rows)) % len(s.rows)
		if s.rows[s.cursor].name != "" {
			return
		}
	}
}

func (s *packagesScreen) toggle() {
	if len(s.rows) == 0 {
		return
	}
	row := s.rows[s.cursor]
	if row.name == "" {
		return
	}
	s.session.Config.SkipPackage(row.name, !s.session.Config.PackageSkipped(row.name))
}

// setGroup includes or excludes every package in the group under the cursor.
func (s *packagesScreen) setGroup(skip bool) {
	if len(s.rows) == 0 {
		return
	}
	group := s.rows[s.cursor].group
	for _, row := range s.rows {
		if row.name != "" && row.group == group {
			s.session.Config.SkipPackage(row.name, skip)
		}
	}
}

func (s *packagesScreen) Chrome() Chrome {
	skipped := len(s.session.Config.SkippedPackages())
	explanation := "Everything is included by default. What a package pulls in itself is the package manager's business, not this list's."
	if skipped > 0 {
		explanation = fmt.Sprintf("%d excluded. %s", skipped, explanation)
	}

	return Chrome{
		Heading:     "Which packages",
		Explanation: explanation,
		Meta:        stageMeta(s.session, StagePackages),
		Keys: []widget.Key{
			{Keys: "↑↓", Help: "move"},
			{Keys: "space", Help: "include/exclude"},
			{Keys: "a/n", Help: "group all/none"},
			{Keys: "r", Help: "reset"},
			{Keys: "enter", Help: "review"},
		},
	}
}

func (s *packagesScreen) View() string {
	t := s.session.Theme
	lines := make([]string, 0, len(s.rows))

	first, last := s.window()
	for i, row := range s.rows {
		if i < first || i > last {
			continue
		}
		if row.name == "" {
			if i > 0 {
				lines = append(lines, "")
			}
			lines = append(lines, t.Muted.Render(string(row.group)))
			continue
		}

		cursor := "  "
		if i == s.cursor {
			cursor = t.Accent.Render(theme.GlyphCursor) + " "
		}

		on := !s.session.Config.PackageSkipped(row.name)
		glyph, style := theme.GlyphOff, t.Faint
		if on {
			glyph, style = theme.GlyphOn, t.Accent
		}

		name := t.Body.Render(row.name)
		switch {
		case i == s.cursor:
			name = t.Selected.Render(row.name)
		case !on:
			name = t.Muted.Render(row.name)
		}

		line := "  " + cursor + style.Render(glyph) + " " + name
		if row.critical && !on {
			// Excluding one of these is allowed and worth flagging: it is the
			// user's machine, but the shell will show it.
			line += " " + t.Warning.Render(theme.GlyphWarn+" the shell needs this")
		}
		lines = append(lines, line)
	}

	if len(lines) == 0 {
		return t.Muted.Render("No package list for this distribution; dependencies are installed by hand.")
	}
	if first > 0 {
		lines = append([]string{t.Faint.Render(fmt.Sprintf("  ↑ %d more", first))}, lines...)
	}
	if last < len(s.rows)-1 {
		lines = append(lines, t.Faint.Render(fmt.Sprintf("  ↓ %d more", len(s.rows)-1-last)))
	}
	return strings.Join(lines, "\n")
}

// window returns the row range to draw, keeping the cursor inside it.
//
// The catalogue runs to well over a hundred rows, which no terminal shows at
// once. Centring on the cursor rather than paging keeps the surrounding
// context visible while moving through a group.
func (s *packagesScreen) window() (first, last int) {
	if len(s.rows) == 0 {
		return 0, -1
	}
	height := s.session.ContentHeight() - 2 // the two "N more" markers
	if height >= len(s.rows) {
		return 0, len(s.rows) - 1
	}

	first = s.cursor - height/2
	if first < 0 {
		first = 0
	}
	last = first + height - 1
	if last > len(s.rows)-1 {
		last = len(s.rows) - 1
		first = last - height + 1
	}
	return first, last
}
