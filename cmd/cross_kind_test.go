package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jumppad-labs/spektacular/internal/output"
	"github.com/jumppad-labs/spektacular/internal/steps/spec"
	"github.com/jumppad-labs/spektacular/internal/workflow"
	"github.com/stretchr/testify/require"
)

// resetSpecGotoFlags clears the goto --data flag so it does not leak across
// tests sharing the package-global cobra commands.
func resetSpecGotoFlags(t *testing.T) {
	t.Helper()
	reset := func() { require.NoError(t, specGotoCmd.Flags().Set("data", "")) }
	reset()
	t.Cleanup(reset)
}

// TestSpecNew_CrossKindReturnsMismatchError asserts that running `spec new`
// while a *plan* workflow is in progress does not resume the plan as a spec:
// it fails with the shared cross_kind_workflow_in_progress error (naming both
// the in-progress kind and the requested kind) and leaves the plan's state
// untouched.
func TestSpecNew_CrossKindReturnsMismatchError(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	dataDir := filepath.Join(dir, ".spektacular")
	writeSpecCommandConfig(t, dir, "")

	writeInProgressState(t, dataDir, workflow.State{
		Kind:           "plan",
		CurrentStep:    "discovery",
		CompletedSteps: []string{"new", "overview"},
		CreatedAt:      fixedResumeTime,
		UpdatedAt:      fixedResumeTime,
		Data:           map[string]any{"name": "000024_resume"},
	})

	before, err := os.ReadFile(filepath.Join(dataDir, "state.json"))
	require.NoError(t, err)

	resetSpecCommandFlags(t)
	stdout, _, code := runRootCmd(t, "spec", "new", "--data", `{"name":"whatever"}`)
	require.Equal(t, 1, code)

	var er output.ErrorResponse
	require.NoError(t, json.Unmarshal([]byte(stdout), &er))
	require.True(t, er.IsError)
	require.Equal(t, "cross_kind_workflow_in_progress", er.Code, "must be reported as a cross-kind mismatch, not a same-kind resume")
	require.Equal(t, "000024_resume", er.Resource)
	require.NotNil(t, er.State)
	require.Equal(t, "discovery", er.State.Current)
	require.Contains(t, er.Message, "plan", "message must name the in-progress kind")
	require.Contains(t, er.Message, "spec", "message must name the requested kind")
	require.Contains(t, er.NextAction, "plan", "instruction must name the in-progress kind")
	require.Contains(t, er.NextAction, "spec new --force", "instruction must offer overwriting with the requested kind")

	after, err := os.ReadFile(filepath.Join(dataDir, "state.json"))
	require.NoError(t, err)
	require.Equal(t, before, after, "a cross-kind new must not mutate the in-progress state")
}

// TestSpecGoto_CrossKindRefusesAndPreservesState asserts the CLI back door is
// closed: `spec goto` against an in-progress plan refuses (mismatch report)
// instead of applying spec steps to the plan's state, and does not advance it.
func TestSpecGoto_CrossKindRefusesAndPreservesState(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	dataDir := filepath.Join(dir, ".spektacular")
	writeSpecCommandConfig(t, dir, "")

	writeInProgressState(t, dataDir, workflow.State{
		Kind:           "plan",
		CurrentStep:    "discovery",
		CompletedSteps: []string{"new", "overview"},
		CreatedAt:      fixedResumeTime,
		UpdatedAt:      fixedResumeTime,
		Data:           map[string]any{"name": "000024_resume"},
	})

	before, err := os.ReadFile(filepath.Join(dataDir, "state.json"))
	require.NoError(t, err)

	resetSpecGotoFlags(t)
	stdout, _, code := runRootCmd(t, "spec", "goto", "--data", `{"step":"overview"}`)
	require.Equal(t, 1, code)

	var er output.ErrorResponse
	require.NoError(t, json.Unmarshal([]byte(stdout), &er))
	require.True(t, er.IsError)
	require.Equal(t, "cross_kind_workflow_in_progress", er.Code)
	require.Equal(t, "000024_resume", er.Resource)
	require.NotNil(t, er.State)
	require.Equal(t, "discovery", er.State.Current)
	require.Contains(t, er.Message, "plan")
	require.Contains(t, er.Message, "spec")

	after, err := os.ReadFile(filepath.Join(dataDir, "state.json"))
	require.NoError(t, err)
	require.Equal(t, before, after, "a refused cross-kind goto must not advance the plan's state")
}

// TestImplementStatus_AfterOtherKindFinishedReportsNoActiveWorkflow
// reproduces a real incident: a plan workflow ran to completion, and before
// any `implement new` was ever called, `implement status` was run. Because
// guardKind previously only guarded against a different-kind workflow still
// in progress (detectInProgress treats "finished" as "nothing to see"), the
// finished plan's raw state.json slipped through unguarded and was loaded
// into an implement-shaped FSM with no kind check of its own — completed
// steps from the plan (20 of them) bled into the implement status report
// against implement's own 13-step list, yielding a nonsensical
// "progress": "20/13". This must instead be a clean, correctly-coded
// "no active workflow" error naming what to run next.
func TestImplementStatus_AfterOtherKindFinishedReportsNoActiveWorkflow(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	dataDir := filepath.Join(dir, ".spektacular")
	writeSpecCommandConfig(t, dir, "")

	writeInProgressState(t, dataDir, workflow.State{
		Kind:        "plan",
		CurrentStep: "finished",
		CompletedSteps: []string{
			"new", "overview", "discovery", "architecture", "components",
			"data_structures", "implementation_detail", "dependencies",
			"testing_approach", "milestones", "phases", "open_questions",
			"out_of_scope", "assemble", "verification", "write_plan",
			"write_context", "write_research", "walkthrough", "finished",
		},
		CreatedAt: fixedResumeTime,
		UpdatedAt: fixedResumeTime,
		Data:      map[string]any{"name": "000045_config-file-migration"},
	})

	stdout, _, code := runRootCmd(t, "implement", "status")
	require.Equal(t, 1, code)

	var er output.ErrorResponse
	require.NoError(t, json.Unmarshal([]byte(stdout), &er))
	require.True(t, er.IsError)
	require.Equal(t, "no_active_workflow", er.Code)
	require.Contains(t, er.Message, "plan", "message must name the finished kind")
	require.Contains(t, er.Message, "000045_config-file-migration", "message must name the finished workflow's instance")
	require.Contains(t, er.NextAction, "implement new")

	// The bug's fingerprint: no field on the response may carry the plan's
	// step vocabulary or its impossible 20-completed-of-13-total progress.
	require.NotContains(t, stdout, "architecture")
	require.NotContains(t, stdout, "20/13")
}

// TestImplementGoto_AfterOtherKindFinishedReportsNoActiveWorkflow is the
// `goto` counterpart: the same stale, finished cross-kind state must not be
// silently loaded into an implement FSM to attempt a transition against.
func TestImplementGoto_AfterOtherKindFinishedReportsNoActiveWorkflow(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	dataDir := filepath.Join(dir, ".spektacular")
	writeSpecCommandConfig(t, dir, "")

	writeInProgressState(t, dataDir, workflow.State{
		Kind:           "plan",
		CurrentStep:    "finished",
		CompletedSteps: []string{"new"},
		CreatedAt:      fixedResumeTime,
		UpdatedAt:      fixedResumeTime,
		Data:           map[string]any{"name": "000045_config-file-migration"},
	})

	resetImplementCommandFlags(t)
	stdout, _, code := runRootCmd(t, "implement", "goto", "--data", `{"step":"analyze"}`)
	require.Equal(t, 1, code)

	var er output.ErrorResponse
	require.NoError(t, json.Unmarshal([]byte(stdout), &er))
	require.Equal(t, "no_active_workflow", er.Code)
	require.Contains(t, er.NextAction, "implement new")
}

// TestSpecStatus_AfterSameKindFinishedStillReportsStatus asserts guardKind's
// fix is scoped to cross-kind mismatches only: a finished workflow of the
// *same* kind (the normal, expected case right after a workflow completes)
// must still report its real status, not be treated as "no active
// workflow" — that would regress a working, legitimate case.
func TestSpecStatus_AfterSameKindFinishedStillReportsStatus(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	dataDir := filepath.Join(dir, ".spektacular")
	writeSpecCommandConfig(t, dir, "")

	writeInProgressState(t, dataDir, workflow.State{
		Kind:           "spec",
		CurrentStep:    "finished",
		CompletedSteps: []string{"new", "interview", "overview", "finished"},
		CreatedAt:      fixedResumeTime,
		UpdatedAt:      fixedResumeTime,
		Data:           map[string]any{"name": "000024_resume"},
	})

	stdout, _, code := runRootCmd(t, "spec", "status")
	require.Equal(t, 0, code)

	var st spec.StatusResult
	require.NoError(t, json.Unmarshal([]byte(stdout), &st))
	require.Equal(t, "000024_resume", st.SpecName)
	require.Equal(t, "finished", st.CurrentStep)
}

// TestMismatchInstruction_RendersBothPathsAcrossKinds asserts the cross-kind
// resume template names the in-progress kind, the requested kind, and both
// follow-up commands (continue the other kind, or overwrite with --force).
func TestMismatchInstruction_RendersBothPathsAcrossKinds(t *testing.T) {
	out, err := mismatchInstruction("spektacular", "plan", "spec", "000024_resume", "discovery")
	require.NoError(t, err)

	require.NotContains(t, out, "{{", "template must be fully rendered")
	require.Contains(t, out, "plan")
	require.Contains(t, out, "spec")
	require.Contains(t, out, "000024_resume")
	require.Contains(t, out, "discovery")
	// Continue the in-progress (plan) workflow with its own skill.
	require.Contains(t, out, `spektacular plan goto --data '{"step":"discovery"}'`)
	// Or overwrite and start the requested (spec) workflow.
	require.Contains(t, out, "spektacular spec new --force")
	// Must not steer the agent to resume the plan as a spec.
	require.NotContains(t, out, "spec goto", "must not suggest resuming the other kind as a spec")
	require.True(t, strings.Contains(out, "in progress"), "must state a workflow is in progress")
}
