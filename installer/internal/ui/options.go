package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"ilmango/internal/installer"
	"ilmango/internal/ui/widget"
)

// entry is one selectable line: either an on/off option or a multi-value
// choice. Holding both in one list keeps the cursor, the grouping and the
// rendering identical for both kinds, so adding either to a catalogue is
// enough to make it appear here.
type entry struct {
	option *installer.Option
	choice *installer.Choice
}

func (e entry) group() installer.OptionGroup {
	if e.option != nil {
		return e.option.Group
	}
	return e.choice.Group
}

// optionsScreen lets the user tune what gets installed.
type optionsScreen struct {
	session *Session
	entries []entry
	cursor  int
}

func newOptionsScreen(session *Session) Screen {
	return &optionsScreen{session: session, entries: buildEntries()}
}

// buildEntries flattens both catalogues, keeping each group's options ahead of
// its choices so related settings stay together.
func buildEntries() []entry {
	options := installer.Options()
	choices := installer.Choices()

	var out []entry
	for _, group := range installer.Groups() {
		for i := range options {
			if options[i].Group == group {
				out = append(out, entry{option: &options[i]})
			}
		}
		for i := range choices {
			if choices[i].Group == group {
				out = append(out, entry{choice: &choices[i]})
			}
		}
	}
	return out
}

func (s *optionsScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return s, nil
	}

	switch key.String() {
	case "q":
		return s, quit(true)
	case "esc":
		return s, navigate(StageWelcome)
	case "up", "k":
		s.move(-1)
	case "down", "j":
		s.move(1)
	case "space", "x":
		s.act(1)
	case "left", "h":
		s.act(-1)
	case "right", "l":
		s.act(1)
	case "a":
		s.setAll(true)
	case "n":
		s.setAll(false)
	case "r":
		s.session.Config = installer.NewConfig()
	case "p":
		// The group toggles above cover almost everyone; this is for the rest.
		return s, navigate(StagePackages)
	case "enter":
		return s, navigate(StageReview)
	}
	return s, nil
}

func (s *optionsScreen) Init() tea.Cmd { return nil }

func (s *optionsScreen) move(delta int) {
	if len(s.entries) == 0 {
		return
	}
	s.cursor = (s.cursor + delta + len(s.entries)) % len(s.entries)
}

// act changes the entry under the cursor: a toggle flips, a choice cycles.
func (s *optionsScreen) act(delta int) {
	if len(s.entries) == 0 {
		return
	}

	current := s.entries[s.cursor]
	if current.option != nil {
		// Toggling a parent leaves its children's own values alone: they are
		// merely inert, and switching the parent back on restores the choice.
		s.session.Config.Toggle(current.option.ID)
		return
	}
	s.session.Config.CycleChoice(current.choice.ID, delta)
}

func (s *optionsScreen) setAll(on bool) {
	for _, e := range s.entries {
		if e.option == nil {
			continue
		}
		// Never switch on a risky option in bulk; those stay opt-in.
		if on && e.option.Risky && !e.option.Default {
			continue
		}
		s.session.Config.Set(e.option.ID, on)
	}
}

func (s *optionsScreen) Chrome() Chrome {
	return Chrome{
		Heading:     "What to install",
		Explanation: "Everything is on by default. Turn off what you already manage yourself.",
		Meta:        stageMeta(s.session, StageOptions),
		Keys: []widget.Key{
			{Keys: "↑↓", Help: "move"},
			{Keys: "space", Help: "change"},
			{Keys: "←→", Help: "cycle"},
			{Keys: "a/n", Help: "all/none"},
			{Keys: "p", Help: "packages"},
			{Keys: "enter", Help: "review"},
		},
	}
}

func (s *optionsScreen) View() string {
	t := s.session.Theme
	cfg := s.session.Config

	var sections []string
	for _, group := range installer.Groups() {
		var rendered []string

		var toggles []widget.OptionRow
		var choices []widget.ChoiceRow

		for i, e := range s.entries {
			if e.group() != group {
				continue
			}

			if o := e.option; o != nil {
				toggles = append(toggles, widget.OptionRow{
					Title:       o.Title,
					Description: o.Description,
					On:          cfg.Get(o.ID),
					Inert:       o.Requires != "" && !cfg.Get(o.Requires),
					Risky:       o.Risky,
					Cursor:      i == s.cursor,
				})
				continue
			}

			c := e.choice
			value := cfg.Choice(c.ID)
			choices = append(choices, widget.ChoiceRow{
				Title:  c.Title,
				Value:  c.Label(value),
				Detail: c.Detail(value),
				Inert:  c.Requires != "" && !cfg.Get(c.Requires),
				Cursor: i == s.cursor,
			})
		}

		if len(toggles) > 0 {
			rendered = append(rendered, widget.OptionList(t, toggles))
		}
		if len(choices) > 0 {
			rendered = append(rendered, widget.ChoiceList(t, choices))
		}
		if len(rendered) == 0 {
			continue
		}

		sections = append(sections, t.Muted.Render(string(group)))
		sections = append(sections, rendered...)
		sections = append(sections, "")
	}

	return strings.TrimRight(lipgloss.JoinVertical(lipgloss.Left, sections...), "\n")
}
