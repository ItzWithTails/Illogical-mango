package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"ilmango/internal/installer"
	"ilmango/internal/ui/widget"
)

// eventBuffer is generous: package managers emit output in bursts, and a full
// channel would block the runner goroutine mid-install.
const eventBuffer = 256

// logTail is how many recent output lines stay on screen. The full transcript
// goes to the log file.
const logTail = 12

// Outcome is the result of a run, handed to the summary screen and to main.
type Outcome struct {
	Complete bool
	Err      error
	Duration time.Duration
	LogPath  string
	Failed   string // title of the step that failed, if any
	// Notes are the things steps decided the user must read afterwards.
	Notes []string
}

// progressScreen runs the plan and renders it live.
type progressScreen struct {
	session *Session
	plan    installer.Plan

	events  chan installer.Event
	cancel  context.CancelFunc
	journal *journal

	rows    []widget.StepRow
	log     []string
	spinner spinner.Model

	started  time.Time
	elapsed  time.Duration
	finished bool
}

// eventMsg wraps a runner event for the Bubble Tea loop.
type eventMsg struct{ event installer.Event }

// tickMsg drives the elapsed-time display.
type tickMsg time.Time

func newProgressScreen(session *Session) Screen {
	plan := installer.BuildPlan(session.Config, session.Steps)

	rows := make([]widget.StepRow, len(plan.Steps))
	for i, planned := range plan.Steps {
		state := installer.StatePending
		if planned.Skipped {
			state = installer.StateSkipped
		}
		rows[i] = widget.StepRow{Title: planned.Step.Title(), State: state}
	}

	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	sp.Style = session.Theme.Accent

	return &progressScreen{
		session: session,
		plan:    plan,
		events:  make(chan installer.Event, eventBuffer),
		rows:    rows,
		spinner: sp,
		started: time.Now(),
		journal: openJournal(),
	}
}

func (s *progressScreen) Init() tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	// The root model calls this when the user quits mid-run, so the bash child
	// is killed rather than orphaned.
	s.session.cancelRun = cancel

	// A long package install can outlast sudo's timestamp; refresh it in the
	// background so a step does not fail halfway through for want of a
	// credential the user already gave.
	if s.session.Env.Runner != nil {
		go s.session.Env.Runner.KeepAlive(ctx)
	}

	go func() {
		defer close(s.events)
		runner := installer.Runner{}
		_ = runner.Run(ctx, s.plan, s.session.Env, func(e installer.Event) { s.events <- e })
	}()

	return tea.Batch(s.spinner.Tick, tick(), waitForEvent(s.events))
}

// waitForEvent blocks on the event channel and delivers one event to the loop.
// A closed channel yields nil, which Bubble Tea discards.
func waitForEvent(ch <-chan installer.Event) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-ch
		if !ok {
			return nil
		}
		return eventMsg{event: event}
	}
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (s *progressScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case eventMsg:
		cmd := s.apply(msg.event)
		return s, tea.Batch(cmd, waitForEvent(s.events))

	case spinner.TickMsg:
		if s.finished {
			return s, nil
		}
		var cmd tea.Cmd
		s.spinner, cmd = s.spinner.Update(msg)
		return s, cmd

	case tickMsg:
		if s.finished {
			return s, nil
		}
		s.elapsed = time.Since(s.started)
		return s, tick()

	case tea.KeyPressMsg:
		// Installation is deliberately hard to leave by accident: only Ctrl+C,
		// handled by the root model, interrupts a run in progress.
		return s, nil
	}
	return s, nil
}

// apply folds a runner event into the screen's state.
func (s *progressScreen) apply(event installer.Event) tea.Cmd {
	switch e := event.(type) {
	case installer.StepStarted:
		s.setState(e.Index, installer.StateRunning)
		s.journal.section(e.Step.Title())

	case installer.StepOutput:
		s.appendLog(e.Line)
		s.journal.line(e.Line)

	case installer.StepDetail:
		if e.Index < len(s.rows) {
			s.rows[e.Index].Detail = e.Text
		}

	case installer.StepFinished:
		if e.Index < len(s.rows) {
			s.rows[e.Index].Duration = e.Duration
			s.rows[e.Index].Detail = ""
		}
		if e.Err != nil {
			state := installer.StateFailed
			if e.Optional {
				state = installer.StateWarned
			}
			s.setState(e.Index, state)
			s.appendLog("error: " + e.Err.Error())
			s.journal.line("error: " + e.Err.Error())
		} else {
			s.setState(e.Index, installer.StateDone)
		}

	case installer.RunFinished:
		s.finished = true
		s.elapsed = e.Duration
		s.session.Outcome = Outcome{
			Complete: e.Err == nil,
			Err:      e.Err,
			Duration: e.Duration,
			LogPath:  s.journal.path(),
			Failed:   s.failedStep(),
			Notes:    s.session.Env.Notes,
		}
		s.session.cancelRun = nil
		s.journal.close()
		if s.cancel != nil {
			s.cancel()
		}
		return navigate(StageSummary)
	}
	return nil
}

func (s *progressScreen) setState(index int, state installer.StepState) {
	if index >= 0 && index < len(s.rows) {
		s.rows[index].State = state
	}
}

func (s *progressScreen) failedStep() string {
	for _, r := range s.rows {
		if r.State == installer.StateFailed {
			return r.Title
		}
	}
	return ""
}

// appendLog keeps only the visible tail in memory; the journal holds the rest.
func (s *progressScreen) appendLog(line string) {
	s.log = append(s.log, line)
	if len(s.log) > logTail {
		s.log = s.log[len(s.log)-logTail:]
	}
}

func (s *progressScreen) Chrome() Chrome {
	heading, explanation := "Installing", "Leave this running. Package installation can take several minutes."
	if s.session.Removing() {
		heading, explanation = "Uninstalling", "Removing the files this installation created."
	}

	return Chrome{
		Heading:     heading,
		Explanation: explanation,
		Meta:        stageMeta(s.session, StageProgress),
		Keys:        []widget.Key{{Keys: "ctrl+c", Help: "cancel"}},
	}
}

func (s *progressScreen) View() string {
	t := s.session.Theme
	width := s.session.ContentWidth()

	for i := range s.rows {
		if s.rows[i].State == installer.StateRunning {
			s.rows[i].Spinner = s.spinner.View()
		} else {
			s.rows[i].Spinner = ""
		}
	}

	done, total := s.counts()
	header := fmt.Sprintf("%d of %d  %s", done, total, formatElapsed(s.elapsed))

	sections := []string{
		widget.Progress(t, width, done, total),
		t.Faint.Render(header),
		"",
		widget.StepList(t, s.rows),
	}

	if len(s.log) > 0 {
		sections = append(sections, "", t.Muted.Render("Output"), s.logView(width))
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// logView renders the tail, truncated to the content width so a long package
// line cannot wrap and shove the layout around.
func (s *progressScreen) logView(width int) string {
	t := s.session.Theme
	lines := make([]string, 0, len(s.log))
	for _, line := range s.log {
		lines = append(lines, "  "+t.LogLine.Render(truncate(line, width-2)))
	}
	return strings.Join(lines, "\n")
}

func (s *progressScreen) counts() (done, total int) {
	for _, r := range s.rows {
		if r.State == installer.StateSkipped {
			continue
		}
		total++
		if r.State == installer.StateDone || r.State == installer.StateFailed {
			done++
		}
	}
	return done, total
}

func truncate(s string, width int) string {
	if width <= 1 || len(s) <= width {
		return s
	}
	return s[:width-1] + "…"
}

func formatElapsed(d time.Duration) string {
	if d < time.Second {
		return "0s"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm %02ds", int(d.Minutes()), int(d.Seconds())%60)
}
