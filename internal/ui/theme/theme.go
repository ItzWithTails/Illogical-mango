// Package theme centralises the installer's visual language.
//
// The design follows the same rule as the shell installer it replaces:
// whitespace over borders, colour over chrome. Every style used anywhere in
// the UI is defined here, so a palette change is a one-file change.
package theme

import (
	"encoding/json"
	"image/color"
	"os"
	"path/filepath"

	"charm.land/lipgloss/v2"
)

// Palette is the semantic colour set. Screens name roles, never raw colours.
type Palette struct {
	Accent    color.Color
	AccentDim color.Color
	Success   color.Color
	Warning   color.Color
	Error     color.Color
	Info      color.Color
	Text      color.Color
	Muted     color.Color
	Faint     color.Color
	Surface   color.Color
}

// defaultPalette mirrors the ANSI-256 values in sdata/lib/tui.sh, so the Go
// installer and the shell installer look like the same product.
var defaultPalette = Palette{
	Accent:    lipgloss.Color("212"),
	AccentDim: lipgloss.Color("99"),
	Success:   lipgloss.Color("82"),
	Warning:   lipgloss.Color("214"),
	Error:     lipgloss.Color("196"),
	Info:      lipgloss.Color("39"),
	Text:      lipgloss.Color("252"),
	Muted:     lipgloss.Color("245"),
	Faint:     lipgloss.Color("240"),
	Surface:   lipgloss.Color("236"),
}

// Theme is a resolved palette plus the styles derived from it.
type Theme struct {
	Palette Palette

	Title    lipgloss.Style
	Subtitle lipgloss.Style
	Body     lipgloss.Style
	Muted    lipgloss.Style
	Faint    lipgloss.Style
	Accent   lipgloss.Style
	Success  lipgloss.Style
	Warning  lipgloss.Style
	Error    lipgloss.Style
	Info     lipgloss.Style
	Badge    lipgloss.Style
	Selected lipgloss.Style
	KeyHint  lipgloss.Style
	KeyCap   lipgloss.Style
	Rule     lipgloss.Style
	LogLine  lipgloss.Style
	Screen   lipgloss.Style
}

// Glyphs are the status markers. They are plain Unicode rather than Nerd Font
// codepoints: the installer runs before any font is installed.
const (
	GlyphDone    = "✓"
	GlyphFail    = "✗"
	GlyphWarn    = "⚠"
	GlyphSkip    = "–"
	GlyphPending = "○"
	GlyphCursor  = "❯"
	GlyphOn      = "◉"
	GlyphOff     = "○"
	GlyphBullet  = "•"
)

// New builds a theme, preferring the user's generated Material You colours and
// falling back to the built-in palette when they are not present yet.
func New() Theme {
	return FromPalette(loadGeneratedPalette())
}

// FromPalette derives every style from p.
func FromPalette(p Palette) Theme {
	base := lipgloss.NewStyle()
	return Theme{
		Palette:  p,
		Title:    base.Foreground(p.Accent).Bold(true),
		Subtitle: base.Foreground(p.AccentDim).Bold(true),
		Body:     base.Foreground(p.Text),
		Muted:    base.Foreground(p.Muted),
		Faint:    base.Foreground(p.Faint),
		Accent:   base.Foreground(p.Accent),
		Success:  base.Foreground(p.Success),
		Warning:  base.Foreground(p.Warning),
		Error:    base.Foreground(p.Error),
		Info:     base.Foreground(p.Info),
		Badge:    base.Foreground(p.Surface).Background(p.Accent).Bold(true).Padding(0, 1),
		Selected: base.Foreground(p.Accent).Bold(true),
		KeyHint:  base.Foreground(p.Muted),
		KeyCap:   base.Foreground(p.Text).Bold(true),
		Rule:     base.Foreground(p.Faint),
		LogLine:  base.Foreground(p.Muted),
		Screen:   base.Padding(1, 3),
	}
}

// generatedColors is the subset of the quickshell palette the installer uses.
type generatedColors struct {
	Primary   string `json:"primary"`
	Secondary string `json:"secondary"`
	Error     string `json:"error"`
	OnSurface string `json:"on_surface"`
	Outline   string `json:"outline"`
	Surface   string `json:"surface_container"`
}

// loadGeneratedPalette reads the colours Illogical-mango generates from the current
// wallpaper. Any problem — missing file, partial JSON, empty field — falls back
// to the corresponding default, so the installer never fails over cosmetics.
func loadGeneratedPalette() Palette {
	p := defaultPalette

	stateHome := os.Getenv("XDG_STATE_HOME")
	if stateHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return p
		}
		stateHome = filepath.Join(home, ".local", "state")
	}

	data, err := os.ReadFile(filepath.Join(stateHome, "quickshell", "user", "generated", "colors.json"))
	if err != nil {
		return p
	}

	var c generatedColors
	if err := json.Unmarshal(data, &c); err != nil {
		return p
	}

	apply(&p.Accent, c.Primary)
	apply(&p.AccentDim, c.Secondary)
	apply(&p.Info, c.Secondary)
	apply(&p.Error, c.Error)
	apply(&p.Text, c.OnSurface)
	apply(&p.Muted, c.Outline)
	apply(&p.Surface, c.Surface)
	return p
}

// apply overwrites a colour only when the candidate looks usable.
func apply(target *color.Color, candidate string) {
	if candidate == "" {
		return
	}
	if candidate[0] != '#' {
		candidate = "#" + candidate
	}
	if len(candidate) != 7 && len(candidate) != 4 {
		return
	}
	*target = lipgloss.Color(candidate)
}
