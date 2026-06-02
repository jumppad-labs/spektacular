package workflow

import (
	"path/filepath"
	"testing"

	"github.com/jumppad-labs/spektacular/internal/store"
	"github.com/stretchr/testify/require"
)

var testSteps = []StepConfig{
	{Name: "one", Src: []string{"new"}, Dst: "one"},
	{Name: "two", Src: []string{"one"}, Dst: "two"},
	{Name: "three", Src: []string{"two"}, Dst: "three"},
}

func TestNew(t *testing.T) {
	sp := filepath.Join(t.TempDir(), "state.json")
	wf := New(testSteps, sp, Config{}, nil, nil)

	require.Equal(t, "new", wf.Current())
	require.False(t, wf.IsComplete())
}

func TestNextAdvancesThroughAllSteps(t *testing.T) {
	sp := filepath.Join(t.TempDir(), "state.json")
	wf := New(testSteps, sp, Config{}, nil, nil)

	err := wf.Next() // new → one
	require.NoError(t, err)
	require.Equal(t, "one", wf.Current())

	err = wf.Next() // one → two
	require.NoError(t, err)
	require.Equal(t, "two", wf.Current())

	err = wf.Next() // two → three
	require.NoError(t, err)
	require.Equal(t, "three", wf.Current())

	err = wf.Next() // three → done
	require.NoError(t, err)
	require.True(t, wf.IsComplete())
}

func TestNextOnCompleteErrors(t *testing.T) {
	sp := filepath.Join(t.TempDir(), "state.json")
	wf := New(testSteps, sp, Config{}, nil, nil)

	for i := 0; i <= len(testSteps); i++ {
		err := wf.Next()
		require.NoError(t, err)
	}

	err := wf.Next()
	require.Error(t, err)
}

func TestGotoForward(t *testing.T) {
	sp := filepath.Join(t.TempDir(), "state.json")
	wf := New(testSteps, sp, Config{}, nil, nil)

	wf.Next() // → one

	err := wf.Goto("two")
	require.NoError(t, err)
	require.Equal(t, "two", wf.Current())
}

// captureWriter is an in-test ResultWriter that records each WriteResult call,
// letting the re-render test assert the step's instruction was re-emitted.
type captureWriter struct {
	calls int
	last  any
}

func (c *captureWriter) WriteResult(v any) error {
	c.calls++
	c.last = v
	return nil
}

func TestGotoSameStepReRenders(t *testing.T) {
	// invoked counts how many times the "one" callback fires; the closure also
	// writes a sentinel result so we can confirm the instruction is re-emitted.
	// Returning "" means the callback does NOT advance the workflow.
	invoked := 0
	steps := []StepConfig{
		{
			Name: "one",
			Src:  []string{"new"},
			Dst:  "one",
			Callback: func(data Data, out ResultWriter, st store.Store, cfg Config) (string, error) {
				invoked++
				_ = out.WriteResult("one-instruction")
				return "", nil
			},
		},
		{Name: "two", Src: []string{"one"}, Dst: "two"},
		{Name: "three", Src: []string{"two"}, Dst: "three"},
	}

	sp := filepath.Join(t.TempDir(), "state.json")
	out := &captureWriter{}
	wf := New(steps, sp, Config{}, nil, out)

	require.NoError(t, wf.Next()) // new → one (fires "one" callback once)

	baselineInvoked := invoked
	baselineWrites := out.calls

	err := wf.Goto("one") // same as current step → re-render, no transition
	require.NoError(t, err)

	// The callback ran again, re-emitting the instruction, without advancing.
	require.Equal(t, baselineInvoked+1, invoked)
	require.Equal(t, baselineWrites+1, out.calls)
	require.Equal(t, "one-instruction", out.last)

	// Current step and completed-steps list are unchanged by the re-render.
	require.Equal(t, "one", wf.Current())
	require.Equal(t, []string{}, wf.State().CompletedSteps)
}

func TestGotoInvalidStepFails(t *testing.T) {
	sp := filepath.Join(t.TempDir(), "state.json")
	wf := New(testSteps, sp, Config{}, nil, nil)

	err := wf.Goto("nonexistent")
	require.Error(t, err)
}

func TestAutoSaveOnTransition(t *testing.T) {
	sp := filepath.Join(t.TempDir(), "state.json")
	wf := New(testSteps, sp, Config{}, nil, nil)

	wf.Next() // → one
	wf.Next() // → two

	// Rebuild from persisted state (auto-saved by enter_state).
	loaded := New(testSteps, sp, Config{}, nil, nil)
	require.Equal(t, "two", loaded.Current())
	require.Equal(t, []string{"one"}, loaded.State().CompletedSteps)
}

func TestStepStatus(t *testing.T) {
	sp := filepath.Join(t.TempDir(), "state.json")
	wf := New(testSteps, sp, Config{}, nil, nil)

	wf.Next() // → one
	wf.Next() // → two

	infos := wf.StepStatus()
	require.Len(t, infos, 3)
	require.Equal(t, "completed", infos[0].Status)
	require.Equal(t, "current", infos[1].Status)
	require.Equal(t, "pending", infos[2].Status)
}

func TestGotoBackwardFails(t *testing.T) {
	sp := filepath.Join(t.TempDir(), "state.json")
	wf := New(testSteps, sp, Config{}, nil, nil)

	wf.Next() // → one
	wf.Next() // → two
	wf.Next() // → three

	err := wf.Goto("one")
	require.Error(t, err)
}

func TestNextStepName(t *testing.T) {
	sp := filepath.Join(t.TempDir(), "state.json")
	wf := New(testSteps, sp, Config{}, nil, nil)

	wf.Next() // → one
	require.Equal(t, "two", wf.NextStepName())

	wf.Next() // → two
	require.Equal(t, "three", wf.NextStepName())

	wf.Next() // → three
	require.Equal(t, "", wf.NextStepName())
}

func TestCallbackGotoAdvances(t *testing.T) {
	steps := []StepConfig{
		{
			Name: "init",
			Src:  []string{"new"},
			Dst:  "init",
			Callback: func(data Data, out ResultWriter, st store.Store, cfg Config) (string, error) {
				return "real", nil
			},
		},
		{Name: "real", Src: []string{"init"}, Dst: "real"},
		{Name: "final", Src: []string{"real"}, Dst: "final"},
	}
	sp := filepath.Join(t.TempDir(), "state.json")
	wf := New(steps, sp, Config{}, nil, nil)

	err := wf.Next() // fires "init", callback returns "real", so advances to "real"
	require.NoError(t, err)
	require.Equal(t, "real", wf.Current())
}

// TestMultiSourceTransition verifies that workflow.StepConfig.Src accepts a
// multi-element source slice and that the FSM can fire the transition from any
// of the listed sources. This is a prerequisite for the implement workflow's
// analyze step, which must be reachable from both read_plan and update_changelog.
func TestMultiSourceTransition(t *testing.T) {
	steps := []StepConfig{
		{Name: "one", Src: []string{"start"}, Dst: "one"},
		{Name: "two", Src: []string{"one", "three"}, Dst: "two"},
		{Name: "three", Src: []string{"two"}, Dst: "three"},
	}

	sp := filepath.Join(t.TempDir(), "state.json")
	wf := New(steps, sp, Config{}, nil, nil)

	require.NoError(t, wf.Goto("one"))
	require.Equal(t, "one", wf.Current())

	require.NoError(t, wf.Goto("two"))
	require.Equal(t, "two", wf.Current())

	require.NoError(t, wf.Goto("three"))
	require.Equal(t, "three", wf.Current())

	// Loop back: three → two via the multi-source edge.
	require.NoError(t, wf.Goto("two"))
	require.Equal(t, "two", wf.Current())
}

func TestCompletedStepsTracked(t *testing.T) {
	sp := filepath.Join(t.TempDir(), "state.json")
	wf := New(testSteps, sp, Config{}, nil, nil)

	wf.Next() // → one
	wf.Next() // → two
	require.Equal(t, []string{"one"}, wf.State().CompletedSteps)

	wf.Next() // → three (terminal)
	// The terminal step is marked completed by commitTerminal after the
	// event fires, since no further transition will mark it later.
	require.Equal(t, []string{"one", "two", "three"}, wf.State().CompletedSteps)
}

func TestKindPersistedOnNew(t *testing.T) {
	sp := filepath.Join(t.TempDir(), "state.json")
	wf := New(testSteps, sp, Config{Kind: "spec"}, nil, nil)

	// Trigger a save so the state file is written to disk.
	require.NoError(t, wf.Next()) // new → one

	// Read the persisted state back from disk and assert the recorded kind.
	loaded, err := loadState(sp)
	require.NoError(t, err)
	require.Equal(t, "spec", loaded.Kind)
}

func TestInProgress(t *testing.T) {
	tests := []struct {
		name        string
		currentStep string
		expected    bool
	}{
		{name: "empty step is not in progress", currentStep: "", expected: false},
		{name: "finished step is not in progress", currentStep: "finished", expected: false},
		{name: "unfinished step is in progress", currentStep: "one", expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &State{CurrentStep: tt.currentStep}
			require.Equal(t, tt.expected, s.InProgress())
		})
	}
}
