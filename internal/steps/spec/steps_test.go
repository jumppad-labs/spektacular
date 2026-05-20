package spec

import (
	"path/filepath"
	"testing"

	"github.com/jumppad-labs/spektacular/internal/store"
	"github.com/jumppad-labs/spektacular/internal/workflow"
	"github.com/stretchr/testify/require"
)

type testData struct {
	values map[string]any
}

func (d *testData) Get(key string) (any, bool) {
	v, ok := d.values[key]
	return v, ok
}

func (d *testData) Set(key string, value any) {
	d.values[key] = value
}

type captureWriter struct {
	result Result
}

func (c *captureWriter) WriteResult(v any) error {
	c.result = v.(Result)
	return nil
}

func renderStep(t *testing.T, cb workflow.StepCallback) string {
	t.Helper()
	data := &testData{values: map[string]any{"name": "test"}}
	writer := &captureWriter{}
	st := store.NewFileStore(t.TempDir(), "project")
	_, err := cb(data, writer, st, workflow.Config{Command: "spektacular"})
	require.NoError(t, err)
	return writer.result.Instruction
}

func TestStepsOrderMatchesExpected(t *testing.T) {
	expected := []string{
		"new",
		"overview",
		"requirements",
		"acceptance_criteria",
		"constraints",
		"technical_approach",
		"success_metrics",
		"non_goals",
		"verification",
		"finished",
	}
	got := Steps()
	require.Len(t, got, len(expected))
	for i, step := range got {
		require.Equal(t, expected[i], step.Name, "step %d name mismatch", i)
	}
}

func TestFSMWalkFromNewToFinished(t *testing.T) {
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")
	st := store.NewFileStore(tmp, "project")
	writer := &captureWriter{}

	wf := workflow.New(Steps(), statePath, workflow.Config{Command: "spektacular", DryRun: true}, st, writer)
	wf.SetData("name", "test")
	wf.SetData("spec_template", "spec content")

	require.Equal(t, "start", wf.Current())

	expectedStates := []string{
		"overview",
		"requirements",
		"acceptance_criteria",
		"constraints",
		"technical_approach",
		"success_metrics",
		"non_goals",
		"verification",
		"finished",
	}

	for _, want := range expectedStates {
		require.NoError(t, wf.Next(), "transition to %s failed", want)
		require.Equal(t, want, wf.Current(), "expected state %s after transition", want)
	}
}

func TestOverviewStepRendersInstruction(t *testing.T) {
	out := renderStep(t, overview())
	require.NotEmpty(t, out)
}

func TestVerificationStepPassesSpecTemplate(t *testing.T) {
	// Verification must render with a spec_template extra var populated from
	// the scaffold so the template can embed the scaffold body.
	tmp := t.TempDir()
	data := &testData{values: map[string]any{"name": "test"}}
	writer := &captureWriter{}
	st := store.NewFileStore(tmp, "project")

	_, err := verification()(data, writer, st, workflow.Config{Command: "spektacular"})
	require.NoError(t, err)
	require.NotEmpty(t, writer.result.Instruction)
}

func TestNewStepWritesScaffold(t *testing.T) {
	tmp := t.TempDir()
	data := &testData{values: map[string]any{"name": "fixture"}}
	writer := &captureWriter{}
	st := store.NewFileStore(tmp, "project")

	next, err := new()(data, writer, st, workflow.Config{Command: "spektacular", SpecDir: "specs"})
	require.NoError(t, err)
	require.Equal(t, "overview", next)
	require.True(t, st.Exists(SpecFilePath("specs", "fixture")))
}

// TestSpecFilePath_UsesConfiguredDirectory asserts the path helper roots the
// spec file under the given directory argument (Phase 2.2, criterion 1).
func TestSpecFilePath_UsesConfiguredDirectory(t *testing.T) {
	require.Equal(t, "my-specs/x.md", SpecFilePath("my-specs", "x"))
}

// TestNewStep_WritesUnderConfiguredSpecDir runs the new callback with a
// non-default SpecDir and asserts the spec file lands under that directory
// (Phase 2.2, criterion 1).
func TestNewStep_WritesUnderConfiguredSpecDir(t *testing.T) {
	tmp := t.TempDir()
	data := &testData{values: map[string]any{"name": "fixture"}}
	writer := &captureWriter{}
	st := store.NewFileStore(tmp, "project")

	_, err := new()(data, writer, st, workflow.Config{Command: "spektacular", SpecDir: "my-specs"})
	require.NoError(t, err)
	require.True(t, st.Exists(SpecFilePath("my-specs", "fixture")), "spec must land under my-specs")
	require.False(t, st.Exists(SpecFilePath("specs", "fixture")), "spec must not land under default specs")
}
