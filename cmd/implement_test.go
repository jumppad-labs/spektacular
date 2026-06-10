package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jumppad-labs/spektacular/internal/workflow"
	"github.com/stretchr/testify/require"
)

// writeFixturePlan creates a minimal plan.md file inside the fake data dir so
// the implement workflow's plan-exists precondition passes.
func writeFixturePlan(t *testing.T, dataDir, name string) string {
	t.Helper()
	planDir := filepath.Join(dataDir, "plans", name)
	require.NoError(t, os.MkdirAll(planDir, 0o755))
	planPath := filepath.Join(planDir, "plan.md")
	body := `# Plan: ` + name + `

## Overview

fixture

## Milestones & Phases

#### - [ ] Phase 1.1: First

#### - [ ] Phase 1.2: Second

#### - [x] Phase 1.3: Already done
`
	require.NoError(t, os.WriteFile(planPath, []byte(body), 0o644))
	return planPath
}

// writeInProgressState marshals a workflow.State to .spektacular/state.json so a
// `new` command's resume prologue (resumeOrClear) sees an in-progress workflow.
func writeInProgressState(t *testing.T, dataDir string, st workflow.State) {
	t.Helper()
	require.NoError(t, os.MkdirAll(dataDir, 0o755))
	b, err := json.MarshalIndent(st, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "state.json"), b, 0o644))
}

// setupImplementCmd resets rootCmd state for a clean test invocation.
func setupImplementCmd(t *testing.T) (*bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})
	return stdout, stderr
}

func TestImplementNew_RejectsMissingPlan(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".spektacular"), 0o755))

	setupImplementCmd(t)
	rootCmd.SetArgs([]string{"implement", "new", "--data", `{"name":"nosuch"}`})

	err := rootCmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "plan file not found")
}

func TestImplementNew_RejectsInvalidName(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".spektacular"), 0o755))

	setupImplementCmd(t)
	rootCmd.SetArgs([]string{"implement", "new", "--data", `{"name":"Invalid Name"}`})

	err := rootCmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "name must match")
}

func TestImplementNew_SucceedsWithExistingPlan(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	dataDir := filepath.Join(dir, ".spektacular")
	require.NoError(t, os.MkdirAll(dataDir, 0o755))
	writeFixturePlan(t, dataDir, "fixture")

	stdout, _ := setupImplementCmd(t)
	rootCmd.SetArgs([]string{"implement", "new", "--data", `{"name":"fixture"}`})

	require.NoError(t, rootCmd.Execute())

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	require.Equal(t, "read_plan", result["step"])
	require.Equal(t, "fixture", result["plan_name"])
	require.Contains(t, result["plan_path"], "plans/fixture/plan.md")
	require.NotEmpty(t, result["instruction"])
}

func TestImplementGoto_RequiresActiveWorkflow(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".spektacular"), 0o755))

	setupImplementCmd(t)
	rootCmd.SetArgs([]string{"implement", "goto", "--data", `{"step":"analyze"}`})

	err := rootCmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "no active implement workflow")
}

func TestImplementGoto_AdvancesThroughStep(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	dataDir := filepath.Join(dir, ".spektacular")
	require.NoError(t, os.MkdirAll(dataDir, 0o755))
	writeFixturePlan(t, dataDir, "fixture")

	// Start the workflow — state file is written with name=fixture.
	setupImplementCmd(t)
	rootCmd.SetArgs([]string{"implement", "new", "--data", `{"name":"fixture"}`})
	require.NoError(t, rootCmd.Execute())

	// Now goto analyze — should produce an analyze instruction.
	stdout, _ := setupImplementCmd(t)
	rootCmd.SetArgs([]string{"implement", "goto", "--data", `{"step":"analyze"}`})
	require.NoError(t, rootCmd.Execute())

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	require.Equal(t, "analyze", result["step"])
}

func TestImplementStatus_ReportsUncheckedPhases(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	dataDir := filepath.Join(dir, ".spektacular")
	require.NoError(t, os.MkdirAll(dataDir, 0o755))
	writeFixturePlan(t, dataDir, "fixture")

	setupImplementCmd(t)
	rootCmd.SetArgs([]string{"implement", "new", "--data", `{"name":"fixture"}`})
	require.NoError(t, rootCmd.Execute())

	stdout, _ := setupImplementCmd(t)
	rootCmd.SetArgs([]string{"implement", "status"})
	require.NoError(t, rootCmd.Execute())

	var status map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &status))
	// Fixture has 2 unchecked phases (1.1, 1.2) and 1 checked (1.3).
	require.EqualValues(t, 2, status["unchecked_phases"])
	require.Equal(t, "fixture", status["plan_name"])
	require.EqualValues(t, 11, status["total_steps"])
}

func TestImplementSteps_ListsAllSteps(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".spektacular"), 0o755))

	stdout, _ := setupImplementCmd(t)
	rootCmd.SetArgs([]string{"implement", "steps"})
	require.NoError(t, rootCmd.Execute())

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	steps := result["steps"].([]any)
	require.Len(t, steps, 11)
	expected := []string{
		"new",
		"read_plan",
		"analyze",
		"implement",
		"test",
		"verify",
		"update_plan",
		"update_changelog",
		"update_repo_changelog",
		"test_plan",
		"finished",
	}
	for i, want := range expected {
		require.Equal(t, want, steps[i])
	}
}

// TestImplementNew_InProgressReturnsResumeReport proves `implement new` runs the
// shared resume prologue and emits a ResumeReport for an in-progress implement
// workflow instead of starting fresh.
func TestImplementNew_InProgressReturnsResumeReport(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	dataDir := filepath.Join(dir, ".spektacular")
	writeFixturePlan(t, dataDir, "fixture")

	writeInProgressState(t, dataDir, workflow.State{
		Kind:           "implement",
		CurrentStep:    "analyze",
		CompletedSteps: []string{"new", "read_plan"},
		CreatedAt:      fixedResumeTime,
		UpdatedAt:      fixedResumeTime,
		Data:           map[string]any{"name": "fixture"},
	})

	stdout, _ := setupImplementCmd(t)
	rootCmd.SetArgs([]string{"implement", "new", "--data", `{"name":"fixture"}`})
	require.NoError(t, rootCmd.Execute())

	var r ResumeReport
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &r))
	require.True(t, r.Resumable)
	require.Equal(t, "implement", r.Kind)
	require.Equal(t, "fixture", r.Name)
	require.Equal(t, "analyze", r.CurrentStep)
}

func TestImplementNew_SchemaOutput(t *testing.T) {
	stdout, _ := setupImplementCmd(t)
	rootCmd.SetArgs([]string{"implement", "new", "--schema"})
	require.NoError(t, rootCmd.Execute())
	require.Contains(t, stdout.String(), `"name"`)
	require.Contains(t, stdout.String(), `"plan_path"`)
}

func TestImplementGoto_SchemaOutput(t *testing.T) {
	stdout, _ := setupImplementCmd(t)
	rootCmd.SetArgs([]string{"implement", "goto", "--schema"})
	require.NoError(t, rootCmd.Execute())
	require.Contains(t, stdout.String(), `"step"`)
	// The enum should list all ten step names.
	require.Contains(t, stdout.String(), "read_plan")
	require.Contains(t, stdout.String(), "update_repo_changelog")
}

func TestImplementStatus_SchemaOutput(t *testing.T) {
	stdout, _ := setupImplementCmd(t)
	rootCmd.SetArgs([]string{"implement", "status", "--schema"})
	require.NoError(t, rootCmd.Execute())
	require.Contains(t, stdout.String(), `"unchecked_phases"`)
}

func TestImplementSteps_SchemaOutput(t *testing.T) {
	stdout, _ := setupImplementCmd(t)
	rootCmd.SetArgs([]string{"implement", "steps", "--schema"})
	require.NoError(t, rootCmd.Execute())
	require.Contains(t, stdout.String(), `"steps"`)
}
