package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

// TestSpecNew_CrossKindReturnsMismatchReport asserts that running `spec new`
// while a *plan* workflow is in progress does not resume the plan as a spec:
// it returns a cross-kind mismatch ResumeReport (RequestedKind=spec, Kind=plan)
// and leaves the plan's state untouched.
func TestSpecNew_CrossKindReturnsMismatchReport(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	dataDir := filepath.Join(dir, ".spektacular")

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
	stdout, _ := setupImplementCmd(t)
	rootCmd.SetArgs([]string{"spec", "new", "--data", `{"name":"whatever"}`})
	require.NoError(t, rootCmd.Execute())

	var r ResumeReport
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &r))
	require.True(t, r.Resumable)
	require.Equal(t, "plan", r.Kind, "report must carry the in-progress workflow's kind")
	require.Equal(t, "spec", r.RequestedKind, "report must carry the kind the user tried to run")
	require.Equal(t, "discovery", r.CurrentStep)
	require.Contains(t, r.Instruction, "plan", "instruction must name the in-progress kind")
	require.Contains(t, r.Instruction, "spec new --force", "instruction must offer overwriting with the requested kind")

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
	stdout, _ := setupImplementCmd(t)
	rootCmd.SetArgs([]string{"spec", "goto", "--data", `{"step":"overview"}`})
	require.NoError(t, rootCmd.Execute())

	var r ResumeReport
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &r))
	require.True(t, r.Resumable)
	require.Equal(t, "plan", r.Kind)
	require.Equal(t, "spec", r.RequestedKind)

	after, err := os.ReadFile(filepath.Join(dataDir, "state.json"))
	require.NoError(t, err)
	require.Equal(t, before, after, "a refused cross-kind goto must not advance the plan's state")
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
