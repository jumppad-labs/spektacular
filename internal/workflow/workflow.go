package workflow

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/store"
	"github.com/looplab/fsm"
)

// Config holds runtime configuration for a workflow. It is not persisted.
type Config struct {
	Command string
	DryRun  bool
	Project config.Config
}

// ResultWriter is implemented by the output writer and passed into step callbacks.
type ResultWriter interface {
	WriteResult(v any) error
}

// StepCallback is the function signature for step callbacks.
// Steps receive the data store, output writer, project store, and workflow config.
// The returned string is the name of the next step to advance to. If empty, the
// workflow stays at the current step. The transition is deferred — it executes
// after the current FSM event completes.
type StepCallback func(data Data, out ResultWriter, st store.Store, cfg Config) (string, error)

// StepConfig defines a single step in a workflow.
// Name is the event name (and step identifier).
// Src lists valid source states. Dst is the destination state.
type StepConfig struct {
	Name     string
	Src      []string
	Dst      string
	Callback StepCallback
}

// Workflow is a linear state machine with persistence.
type Workflow struct {
	FSM         *fsm.FSM
	steps       []StepConfig
	state       *State
	data        *mapData
	statePath   string
	cfg         Config
	store       store.Store
	out         ResultWriter
	pendingGoto string
}

// New builds a workflow FSM from step configs.
// If a state file exists at statePath, its CurrentStep is used as the initial
// FSM state. Otherwise the initial state defaults to steps[0].Src[0].
// An implicit "done" transition is appended after the last step.
// State is automatically persisted on every transition unless cfg.DryRun is true.
// st is the project store passed to every step callback; it may be nil for
// workflows that do not perform storage operations.
func New(steps []StepConfig, statePath string, cfg Config, st store.Store, out ResultWriter) *Workflow {
	events := make([]fsm.EventDesc, 0, len(steps)+1)
	callbacks := make(fsm.Callbacks)

	// Load existing state or create new.
	var state *State
	var initialState string
	if s, err := loadState(statePath); err == nil {
		state = s
		initialState = s.CurrentStep
	} else {
		initialState = steps[0].Src[0]
		now := time.Now().UTC()
		state = &State{
			CurrentStep:    initialState,
			CompletedSteps: []string{},
			CreatedAt:      now,
			UpdatedAt:      now,
		}
	}

	w := &Workflow{
		steps:     steps,
		state:     state,
		data:      newMapData(state.Data),
		statePath: statePath,
		cfg:       cfg,
		store:     st,
		out:       out,
	}
	// Keep state.Data pointing at the same underlying map so saves include step writes.
	w.state.Data = w.data.base

	for _, s := range steps {
		events = append(events, fsm.EventDesc{
			Name: s.Name,
			Src:  s.Src,
			Dst:  s.Dst,
		})
		if s.Callback != nil {
			step := s // capture
			callbacks["after_"+s.Name] = func(_ context.Context, e *fsm.Event) {
				nextStep, err := step.Callback(w.data, w.out, w.store, w.cfg)
				if err != nil {
					e.Cancel(err)
					return
				}
				if nextStep != "" {
					w.pendingGoto = nextStep
				}
			}
		}
	}

	// Implicit final transition to "done".
	if len(steps) > 0 {
		last := steps[len(steps)-1]
		events = append(events, fsm.EventDesc{
			Name: "done",
			Src:  []string{last.Dst},
			Dst:  "done",
		})
	}

	// Auto-update and persist state on every transition.
	callbacks["enter_state"] = func(_ context.Context, e *fsm.Event) {
		if w.validStep(e.Src) {
			w.state.markCompleted(e.Src)
		}
		w.state.CurrentStep = e.Dst
		w.state.UpdatedAt = time.Now().UTC()
		if !cfg.DryRun {
			_ = saveState(w.statePath, w.state)
		}
	}

	w.FSM = fsm.NewFSM(initialState, events, callbacks)
	return w
}

// Next fires the first available transition.
// If the step callback returns a next step name, Next delegates to Goto to
// advance the workflow further.
func (w *Workflow) Next() error {
	transitions := w.FSM.AvailableTransitions()
	if len(transitions) == 0 {
		return fmt.Errorf("workflow is already complete")
	}
	w.pendingGoto = ""
	if err := w.FSM.Event(context.Background(), transitions[0]); err != nil {
		return err
	}
	if w.pendingGoto != "" {
		return w.Goto(w.pendingGoto)
	}
	w.commitTerminal()
	return nil
}

// Goto jumps to a named step by firing the corresponding FSM event.
// The step's Src list must include the current state; otherwise the FSM errors.
// If the step callback returns a next step name, Goto calls itself recursively.
func (w *Workflow) Goto(name string) error {
	if w.Current() == name {
		return nil
	}

	w.pendingGoto = ""
	if err := w.FSM.Event(context.Background(), name); err != nil {
		return err
	}
	if w.pendingGoto != "" {
		return w.Goto(w.pendingGoto)
	}
	w.commitTerminal()
	return nil
}

// commitTerminal marks the terminal step as completed and persists state
// when the workflow has landed on the last step. enter_state only marks
// src on each transition, so without this the terminal step would never be
// recorded in completed_steps (nothing ever transitions away from it).
func (w *Workflow) commitTerminal() {
	if len(w.steps) == 0 {
		return
	}
	last := w.steps[len(w.steps)-1].Dst
	if w.Current() != last {
		return
	}
	if slices.Contains(w.state.CompletedSteps, last) {
		return
	}
	w.state.markCompleted(last)
	w.state.UpdatedAt = time.Now().UTC()
	if !w.cfg.DryRun {
		_ = saveState(w.statePath, w.state)
	}
}

// SetData stores a value in the persistent data store.
func (w *Workflow) SetData(key string, value any) {
	w.data.Set(key, value)
}

// GetData retrieves a value from the data store.
func (w *Workflow) GetData(key string) (any, bool) {
	return w.data.Get(key)
}

// FirstStep returns the name of the first step.
func (w *Workflow) FirstStep() string {
	if len(w.steps) > 0 {
		return w.steps[0].Name
	}
	return ""
}

// Current returns the current state name.
func (w *Workflow) Current() string {
	return w.FSM.Current()
}

// State returns the persisted state.
func (w *Workflow) State() *State {
	return w.state
}

// IsComplete returns true if the workflow has reached "done".
func (w *Workflow) IsComplete() bool {
	return w.Current() == "done"
}

// StepNames returns the ordered step names from the config.
func (w *Workflow) StepNames() []string {
	names := make([]string, len(w.steps))
	for i, s := range w.steps {
		names[i] = s.Name
	}
	return names
}

// NextStepName returns the name of the next step, or "" if at the end.
func (w *Workflow) NextStepName() string {
	cur := w.Current()
	for i, s := range w.steps {
		if s.Dst == cur && i+1 < len(w.steps) {
			return w.steps[i+1].Name
		}
	}
	return ""
}

// StepStatus returns per-step status info.
func (w *Workflow) StepStatus() []StepInfo {
	infos := make([]StepInfo, len(w.steps))
	for i, s := range w.steps {
		status := "pending"
		if slices.Contains(w.state.CompletedSteps, s.Name) {
			status = "completed"
		}
		if s.Dst == w.Current() {
			status = "current"
		}
		infos[i] = StepInfo{Name: s.Name, Status: status}
	}
	return infos
}

// StepInfo is a step with its current status.
type StepInfo struct {
	Name   string
	Status string // "pending", "current", "completed"
}

func (w *Workflow) validStep(name string) bool {
	for _, s := range w.steps {
		if s.Name == name {
			return true
		}
	}
	return false
}
