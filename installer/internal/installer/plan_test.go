package installer

import (
	"context"
	"errors"
	"testing"
)

// fakeStep is a Step whose behaviour a test dictates.
type fakeStep struct {
	id      string
	applies bool
	err     error
	ran     bool
}

func (f *fakeStep) ID() string            { return f.id }
func (f *fakeStep) Title() string         { return f.id }
func (f *fakeStep) Description() string   { return "" }
func (f *fakeStep) AppliesTo(Config) bool { return f.applies }
func (f *fakeStep) Run(context.Context, *Env) error {
	f.ran = true
	return f.err
}

func TestBuildPlanKeepsSkippedStepsVisible(t *testing.T) {
	steps := []Step{
		&fakeStep{id: "a", applies: true},
		&fakeStep{id: "b", applies: false},
	}

	plan := BuildPlan(NewConfig(), steps)

	if len(plan.Steps) != 2 {
		t.Fatalf("plan has %d steps, want both retained", len(plan.Steps))
	}
	if plan.Active() != 1 {
		t.Errorf("Active() = %d, want 1", plan.Active())
	}
	if !plan.Steps[1].Skipped {
		t.Error("step b should be marked skipped")
	}
}

func TestRunnerStopsAtFirstFailure(t *testing.T) {
	failing := &fakeStep{id: "boom", applies: true, err: errors.New("boom")}
	later := &fakeStep{id: "later", applies: true}
	plan := BuildPlan(NewConfig(), []Step{failing, later})

	err := Runner{}.Run(context.Background(), plan, &Env{}, func(Event) {})

	if err == nil {
		t.Fatal("expected the runner to surface the step error")
	}
	if later.ran {
		t.Error("no step should run after a failure unless ContinueOnError is set")
	}
}

func TestRunnerContinueOnError(t *testing.T) {
	failing := &fakeStep{id: "boom", applies: true, err: errors.New("boom")}
	later := &fakeStep{id: "later", applies: true}
	plan := BuildPlan(NewConfig(), []Step{failing, later})

	err := Runner{ContinueOnError: true}.Run(context.Background(), plan, &Env{}, func(Event) {})

	if err == nil {
		t.Fatal("the first error should still be returned")
	}
	if !later.ran {
		t.Error("ContinueOnError should have let the later step run")
	}
}

func TestRunnerEmitsLifecycleEvents(t *testing.T) {
	plan := BuildPlan(NewConfig(), []Step{
		&fakeStep{id: "a", applies: true},
		&fakeStep{id: "b", applies: false},
	})

	var started, skipped, finished, runFinished int
	_ = Runner{}.Run(context.Background(), plan, &Env{}, func(e Event) {
		switch e.(type) {
		case StepStarted:
			started++
		case StepSkipped:
			skipped++
		case StepFinished:
			finished++
		case RunFinished:
			runFinished++
		}
	})

	if started != 1 || skipped != 1 || finished != 1 || runFinished != 1 {
		t.Errorf("events: started=%d skipped=%d finished=%d runFinished=%d",
			started, skipped, finished, runFinished)
	}
}

func TestRunnerRecoversFromPanic(t *testing.T) {
	plan := BuildPlan(NewConfig(), []Step{panickyStep{}})

	err := Runner{}.Run(context.Background(), plan, &Env{}, func(Event) {})

	if err == nil {
		t.Fatal("a panicking step must be reported as an error, not crash the program")
	}
}

type panickyStep struct{}

func (panickyStep) ID() string            { return "panic" }
func (panickyStep) Title() string         { return "panic" }
func (panickyStep) Description() string   { return "" }
func (panickyStep) AppliesTo(Config) bool { return true }
func (panickyStep) Run(context.Context, *Env) error {
	panic("kaboom")
}
