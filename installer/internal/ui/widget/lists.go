package widget

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"ilmango/internal/installer"
	"ilmango/internal/system"
	"ilmango/internal/ui/theme"
)

// StepRow is the display state of one step in a running plan.
type StepRow struct {
	Title    string
	State    installer.StepState
	Detail   string
	Duration time.Duration
	Spinner  string // frame to draw while the step is running
}

// StepList renders the plan as a status column.
func StepList(t theme.Theme, rows []StepRow) string {
	lines := make([]string, 0, len(rows))
	for _, r := range rows {
		marker, style := stepMarker(t, r)

		title := t.Body.Render(r.Title)
		switch r.State {
		case installer.StatePending:
			title = t.Faint.Render(r.Title)
		case installer.StateSkipped:
			title = t.Faint.Render(r.Title)
		case installer.StateRunning:
			title = t.Body.Bold(true).Render(r.Title)
		}

		line := "  " + style.Render(marker) + " " + title
		if detail := stepDetail(r); detail != "" {
			line += t.Faint.Render("  " + detail)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func stepMarker(t theme.Theme, r StepRow) (string, lipgloss.Style) {
	switch r.State {
	case installer.StateRunning:
		if r.Spinner != "" {
			return r.Spinner, t.Accent
		}
		return theme.GlyphCursor, t.Accent
	case installer.StateDone:
		return theme.GlyphDone, t.Success
	case installer.StateFailed:
		return theme.GlyphFail, t.Error
	case installer.StateWarned:
		return theme.GlyphWarn, t.Warning
	case installer.StateSkipped:
		return theme.GlyphSkip, t.Faint
	default:
		return theme.GlyphPending, t.Faint
	}
}

// stepDetail picks the most useful trailing annotation for a row.
func stepDetail(r StepRow) string {
	switch r.State {
	case installer.StateWarned:
		return "skipped after an error"
	case installer.StateSkipped:
		return "skipped"
	case installer.StateDone:
		if r.Duration >= time.Second {
			return formatDuration(r.Duration)
		}
		return ""
	case installer.StateRunning:
		return r.Detail
	default:
		return ""
	}
}

// CheckList renders preflight results.
func CheckList(t theme.Theme, results []system.CheckResult) string {
	if len(results) == 0 {
		return t.Faint.Render("  checking…")
	}

	lines := make([]string, 0, len(results))
	for _, r := range results {
		var marker string
		var style lipgloss.Style
		switch r.Status {
		case system.CheckPass:
			marker, style = theme.GlyphDone, t.Success
		case system.CheckWarn:
			marker, style = theme.GlyphWarn, t.Warning
		default:
			marker, style = theme.GlyphFail, t.Error
		}

		line := "  " + style.Render(marker) + " " + t.Body.Render(r.Title)
		if r.Detail != "" {
			line += t.Faint.Render("  " + r.Detail)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

// OptionRow is the display state of one togglable option.
type OptionRow struct {
	Title       string
	Description string
	On          bool
	// Inert marks an option whose parent is switched off: it is shown, but it
	// cannot take effect, so it is drawn dimmed and unselectable.
	Inert  bool
	Risky  bool
	Cursor bool
}

// OptionList renders a group of options as a toggle column.
func OptionList(t theme.Theme, rows []OptionRow) string {
	lines := make([]string, 0, len(rows)*2)
	for _, r := range rows {
		cursor := "  "
		if r.Cursor {
			cursor = t.Accent.Render(theme.GlyphCursor) + " "
		}

		glyph := theme.GlyphOff
		glyphStyle := t.Faint
		if r.On {
			glyph = theme.GlyphOn
			glyphStyle = t.Accent
		}

		title := t.Body.Render(r.Title)
		switch {
		case r.Inert:
			glyphStyle, title = t.Faint, t.Faint.Render(r.Title)
		case r.Cursor:
			title = t.Selected.Render(r.Title)
		case !r.On:
			title = t.Muted.Render(r.Title)
		}

		line := cursor + glyphStyle.Render(glyph) + " " + title
		if r.Risky && r.On && !r.Inert {
			line += " " + t.Warning.Render(theme.GlyphWarn)
		}
		lines = append(lines, line)

		if r.Cursor && r.Description != "" {
			lines = append(lines, "    "+t.Faint.Render(r.Description))
		}
	}
	return strings.Join(lines, "\n")
}

// ChoiceRow is the display state of a multi-value setting.
type ChoiceRow struct {
	Title       string
	Description string
	Value       string
	Detail      string
	Inert       bool
	Cursor      bool
}

// ChoiceList renders settings that cycle through values rather than toggle.
func ChoiceList(t theme.Theme, rows []ChoiceRow) string {
	lines := make([]string, 0, len(rows)*2)
	for _, r := range rows {
		cursor := "  "
		if r.Cursor {
			cursor = t.Accent.Render(theme.GlyphCursor) + " "
		}

		title := t.Body.Render(r.Title)
		value := t.Accent.Render(r.Value)
		switch {
		case r.Inert:
			title, value = t.Faint.Render(r.Title), t.Faint.Render(r.Value)
		case r.Cursor:
			title = t.Selected.Render(r.Title)
		}

		// The arrows say the value cycles, which a bare label would not.
		lines = append(lines, cursor+"  "+title+t.Faint.Render("  ‹ ")+value+t.Faint.Render(" ›"))

		if r.Cursor && r.Detail != "" {
			lines = append(lines, "    "+t.Faint.Render(r.Detail))
		}
	}
	return strings.Join(lines, "\n")
}

func formatDuration(d time.Duration) string {
	switch {
	case d >= time.Minute:
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	case d >= time.Second:
		return fmt.Sprintf("%.1fs", d.Seconds())
	default:
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
}
