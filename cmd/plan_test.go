package cmd

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/jumppad-labs/spektacular/internal/workflow"
	"github.com/stretchr/testify/require"
)

// TestPlanNew_InProgressSpecReportedCrossKind proves `plan new` runs the shared
// resume prologue and surfaces the stored kind (spec) rather than its own kind
// (plan), so a cross-kind in-progress workflow is reported faithfully.
func TestPlanNew_InProgressSpecReportedCrossKind(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	dataDir := filepath.Join(dir, ".spektacular")

	writeInProgressState(t, dataDir, workflow.State{
		Kind:           "spec",
		CurrentStep:    "overview",
		CompletedSteps: []string{"new"},
		CreatedAt:      fixedResumeTime,
		UpdatedAt:      fixedResumeTime,
		Data:           map[string]any{"name": "000024_resume"},
	})

	stdout, _ := setupImplementCmd(t)
	rootCmd.SetArgs([]string{"plan", "new", "--data", `{"name":"myplan"}`})
	require.NoError(t, rootCmd.Execute())

	var r ResumeReport
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &r))
	require.True(t, r.Resumable)
	require.Equal(t, "spec", r.Kind)
	require.Equal(t, "000024_resume", r.Name)
	require.Equal(t, "overview", r.CurrentStep)
}
