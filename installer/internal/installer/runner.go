package installer

import (
	"context"
	"fmt"
	"time"
)

// Runner executes a Plan and streams Events describing its progress.
type Runner struct {
	// ContinueOnError keeps the run going after a step fails, which is useful
	// for a diagnostic pass. The default is to stop at the first failure.
	ContinueOnError bool
}

// Run executes the plan against env, delivering every event to emit. Emission
// is synchronous, so emit must not block for long; the TUI hands events to a
// buffered channel.
//
// Run returns the first step error, or ctx.Err() if the run was cancelled.
func (r Runner) Run(ctx context.Context, plan Plan, env *Env, emit func(Event)) error {
	started := time.Now()
	emit(RunStarted{Total: plan.Active()})

	var firstErr error
	for i, planned := range plan.Steps {
		if planned.Skipped {
			emit(StepSkipped{Index: i, Step: planned.Step})
			continue
		}

		if err := ctx.Err(); err != nil {
			firstErr = coalesce(firstErr, err)
			break
		}

		emit(StepStarted{Index: i, Step: planned.Step})
		stepStart := time.Now()
		env.Reporter = newEmitReporter(i, emit)
		// Command output belongs to whichever step is running, so the runner
		// re-points the command log for each one.
		if env.Runner != nil {
			env.Runner.Log = env.Reporter.Log
		}
		err := runStep(ctx, planned.Step, env)
		optional := err != nil && isOptional(planned.Step) && ctx.Err() == nil

		emit(StepFinished{
			Index:    i,
			Step:     planned.Step,
			Err:      err,
			Optional: optional,
			Duration: time.Since(stepStart),
		})

		switch {
		case err == nil:
		case optional:
			// The step is a convenience: say what was lost and carry on.
			env.Note(fmt.Sprintf("%s did not complete: %v", planned.Step.Title(), err))
		default:
			firstErr = coalesce(firstErr, err)
			if !r.ContinueOnError {
				return finish(emit, started, firstErr)
			}
		}
	}

	return finish(emit, started, firstErr)
}

// finish emits the closing event exactly once, whichever way the run ended.
func finish(emit func(Event), started time.Time, err error) error {
	emit(RunFinished{Err: err, Duration: time.Since(started)})
	return err
}

// runStep isolates a step so a panic in one cannot take down the TUI.
func runStep(ctx context.Context, step Step, env *Env) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("step %s panicked: %v", step.ID(), rec)
		}
	}()
	return step.Run(ctx, env)
}

func coalesce(existing, candidate error) error {
	if existing != nil {
		return existing
	}
	return candidate
}

// emitReporter adapts the event stream to the Reporter a step is handed.
type emitReporter struct {
	index int
	emit  func(Event)
}

func newEmitReporter(index int, emit func(Event)) Reporter {
	return emitReporter{index: index, emit: emit}
}

func (r emitReporter) Log(line string)    { r.emit(StepOutput{Index: r.index, Line: line}) }
func (r emitReporter) Detail(text string) { r.emit(StepDetail{Index: r.index, Text: text}) }
