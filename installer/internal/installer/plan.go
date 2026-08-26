package installer

import (
	"fmt"
	"sync"
)

// StepState is the lifecycle position of a step within a run.
type StepState int

const (
	StatePending StepState = iota
	StateRunning
	StateDone
	StateFailed
	StateSkipped
	// StateWarned is an optional step that failed. The run carried on.
	StateWarned
)

func (s StepState) String() string {
	switch s {
	case StateRunning:
		return "running"
	case StateDone:
		return "done"
	case StateFailed:
		return "failed"
	case StateSkipped:
		return "skipped"
	case StateWarned:
		return "warned"
	default:
		return "pending"
	}
}

// Terminal reports whether no further transition is expected.
func (s StepState) Terminal() bool {
	return s == StateDone || s == StateFailed || s == StateSkipped || s == StateWarned
}

// PlannedStep pairs a step with the reason it is or is not going to run.
type PlannedStep struct {
	Step    Step
	Skipped bool
}

// Plan is an ordered, immutable set of steps resolved against a Config.
type Plan struct {
	Config Config
	Steps  []PlannedStep
}

// BuildPlan resolves the registered step catalog against cfg. Steps that do
// not apply are retained as skipped entries so the UI can show the full shape
// of an installation rather than a mysteriously short list.
func BuildPlan(cfg Config, steps []Step) Plan {
	planned := make([]PlannedStep, 0, len(steps))
	for _, s := range steps {
		planned = append(planned, PlannedStep{Step: s, Skipped: !s.AppliesTo(cfg)})
	}
	return Plan{Config: cfg, Steps: planned}
}

// ReadOnly is implemented by steps that only inspect the machine. The plan
// uses it to tell "nothing will change" apart from "nothing will run": a run
// consisting only of checks is not an installation.
//
// It is an optional interface, so a step that does not implement it is treated
// as changing something — the safe assumption.
type ReadOnly interface {
	ReadOnly() bool
}

// Optional is implemented by steps whose failure must not abort the run.
//
// Desktop integration is the motivating case: a missing icon cache or an
// unavailable gsettings schema should cost the user that one convenience, not
// the entire installation they just waited through.
//
// It is an optional interface; a step that does not implement it is required.
type Optional interface {
	Optional() bool
}

// isOptional reports whether a step's failure should be survivable.
func isOptional(s Step) bool {
	o, ok := s.(Optional)
	return ok && o.Optional()
}

// isReadOnly reports whether a step only inspects the machine.
func isReadOnly(s Step) bool {
	ro, ok := s.(ReadOnly)
	return ok && ro.ReadOnly()
}

// Mutating returns the number of steps that will run and change something.
func (p Plan) Mutating() int {
	n := 0
	for _, s := range p.Steps {
		if !s.Skipped && !isReadOnly(s.Step) {
			n++
		}
	}
	return n
}

// Active returns the number of steps that will actually run.
func (p Plan) Active() int {
	n := 0
	for _, s := range p.Steps {
		if !s.Skipped {
			n++
		}
	}
	return n
}

// Operation is what a plan is for. Each has its own step catalogue, so
// installing and removing share the runner, the events and the interface
// without sharing a single conditional.
type Operation string

const (
	OpInstall   Operation = "install"
	OpUninstall Operation = "uninstall"
)

// registry holds the process-wide step catalogues. Steps register themselves
// from package init, which keeps adding one to a single file.
var registry struct {
	mu    sync.RWMutex
	steps map[Operation][]Step
}

// Register adds steps to an operation's catalogue. It panics on a duplicate ID
// within one operation, since that is a programming error that would otherwise
// surface as a confusing plan.
func Register(op Operation, steps ...Step) {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	if registry.steps == nil {
		registry.steps = map[Operation][]Step{}
	}
	for _, s := range steps {
		for _, existing := range registry.steps[op] {
			if existing.ID() == s.ID() {
				panic(fmt.Sprintf("installer: duplicate %s step ID %q", op, s.ID()))
			}
		}
		registry.steps[op] = append(registry.steps[op], s)
	}
}

// RegisteredSteps returns an operation's catalogue in registration order.
func RegisteredSteps(op Operation) []Step {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	out := make([]Step, len(registry.steps[op]))
	copy(out, registry.steps[op])
	return out
}
