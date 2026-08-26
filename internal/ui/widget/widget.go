// Package widget holds the installer's reusable presentation pieces: the
// chrome that frames every screen, and the lists that render plan and
// preflight state. They are pure functions of their inputs — no Bubble Tea
// model state lives here.
package widget

import (
	"strings"

	"charm.land/lipgloss/v2"

	"ilmango/internal/ui/theme"
)

// Key is one keybinding advertised in the footer.
type Key struct {
	Keys string
	Help string
}

// Header renders the product mark, a right-aligned meta line, and a rule.
func Header(t theme.Theme, width int, brand, meta string) string {
	left := t.Title.Render(brand)
	right := t.Faint.Render(meta)

	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		left+strings.Repeat(" ", gap)+right,
		t.Rule.Render(strings.Repeat("─", max(width, 1))),
	)
}

// Title renders a screen's heading and its one-line explanation.
func Title(t theme.Theme, heading, explanation string) string {
	out := t.Subtitle.Render(heading)
	if explanation != "" {
		out += "\n" + t.Muted.Render(explanation)
	}
	return out
}

// Footer renders the keybinding hints, separated by a rule.
func Footer(t theme.Theme, width int, keys []Key) string {
	if len(keys) == 0 {
		return ""
	}

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, t.KeyCap.Render(k.Keys)+" "+t.KeyHint.Render(k.Help))
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		t.Rule.Render(strings.Repeat("─", max(width, 1))),
		strings.Join(parts, t.Faint.Render("   ")),
	)
}

// Progress renders a slim determinate bar. It is drawn by hand rather than
// with bubbles/progress so it inherits the theme's accent directly.
func Progress(t theme.Theme, width, done, total int) string {
	if total <= 0 || width <= 0 {
		return ""
	}

	filled := width * done / total
	filled = clamp(filled, 0, width)

	return t.Accent.Render(strings.Repeat("━", filled)) +
		t.Faint.Render(strings.Repeat("━", width-filled))
}

// Wrap re-flows text to width, preserving existing paragraph breaks.
func Wrap(t theme.Theme, width int, text string) string {
	if width <= 0 {
		return text
	}
	return lipgloss.NewStyle().Width(width).Render(text)
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
