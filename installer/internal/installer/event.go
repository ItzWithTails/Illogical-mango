package installer

import "time"

// Event is a progress notification emitted while a plan runs. Consumers switch
// on the concrete type; the sealed interface keeps the set closed.
type Event interface{ isEvent() }

// RunStarted is emitted once, before the first step.
type RunStarted struct {
	Total int // number of steps that will actually run
}

// StepStarted is emitted when a step begins.
type StepStarted struct {
	Index int
	Step  Step
}

// StepSkipped is emitted for steps the configuration excluded.
type StepSkipped struct {
	Index int
	Step  Step
}

// StepOutput carries one line of a step's output.
type StepOutput struct {
	Index int
	Line  string
}

// StepDetail updates the short status shown beside a step's title.
type StepDetail struct {
	Index int
	Text  string
}

// StepFinished is emitted when a step ends, successfully or not.
type StepFinished struct {
	Index int
	Step  Step
	Err   error
	// Optional says the step failed but the run continued, because losing it
	// costs a convenience rather than the installation.
	Optional bool
	Duration time.Duration
}

// RunFinished is emitted once, after the last step. Err is the first failure.
type RunFinished struct {
	Err      error
	Duration time.Duration
}

func (RunStarted) isEvent()   {}
func (StepStarted) isEvent()  {}
func (StepSkipped) isEvent()  {}
func (StepOutput) isEvent()   {}
func (StepDetail) isEvent()   {}
func (StepFinished) isEvent() {}
func (RunFinished) isEvent()  {}
