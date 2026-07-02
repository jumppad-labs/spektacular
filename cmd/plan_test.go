package cmd

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/jumppad-labs/spektacular/internal/output"
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

	stdout, _, code := runRootCmd(t, "plan", "new", "--data", `{"name":"myplan"}`)
	require.Equal(t, 1, code)

	var er output.ErrorResponse
	require.NoError(t, json.Unmarshal([]byte(stdout), &er))
	require.True(t, er.IsError)
	require.Equal(t, "cross_kind_workflow_in_progress", er.Code)
	require.Equal(t, "000024_resume", er.Resource)
	require.NotNil(t, er.State)
	require.Equal(t, "overview", er.State.Current)
	require.Contains(t, er.Message, "spec", "message must name the in-progress kind")
}
