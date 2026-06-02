package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/jumppad-labs/spektacular/internal/steps/spec"
	"github.com/jumppad-labs/spektacular/internal/workflow"
	"github.com/stretchr/testify/require"
)

type specCommandResult struct {
	Step        string `json:"step"`
	SpecPath    string `json:"spec_path"`
	SpecName    string `json:"spec_name"`
	Instruction string `json:"instruction"`
}

func writeSpecCommandConfig(t *testing.T, dir, body string) {
	t.Helper()
	dataDir := filepath.Join(dir, ".spektacular")
	require.NoError(t, os.MkdirAll(dataDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "config.yaml"), []byte(body), 0o644))
}

func writeSpecCommandFile(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, ".spektacular", "specs", name+".md")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("existing"), 0o644))
}

func resetSpecCommandFlags(t *testing.T) {
	t.Helper()
	reset := func() {
		require.NoError(t, specCmd.PersistentFlags().Set("schema", "false"))
		require.NoError(t, specCmd.PersistentFlags().Set("dry-run", "false"))
		require.NoError(t, specNewCmd.Flags().Set("data", ""))
		require.NoError(t, specNewCmd.Flags().Set("stdin", ""))
		require.NoError(t, specNewCmd.Flags().Set("file", ""))
		require.NoError(t, specNewCmd.Flags().Set("force", "false"))
	}
	reset()
	t.Cleanup(reset)
}

func setSpecIdentifierNow(t *testing.T, now time.Time) {
	t.Helper()
	original := specIdentifierNow
	specIdentifierNow = func() time.Time { return now }
	t.Cleanup(func() {
		specIdentifierNow = original
	})
}

func runSpecNewForTest(t *testing.T, args ...string) (specCommandResult, error) {
	t.Helper()
	resetSpecCommandFlags(t)
	stdout, _ := setupImplementCmd(t)
	rootCmd.SetArgs(append([]string{"spec", "new"}, args...))

	err := rootCmd.Execute()
	if err != nil {
		return specCommandResult{}, err
	}

	var result specCommandResult
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	return result, nil
}

func runSpecNewSchemaForTest(t *testing.T) commandSchema {
	t.Helper()
	resetSpecCommandFlags(t)
	stdout, _ := setupImplementCmd(t)
	rootCmd.SetArgs([]string{"spec", "new", "--schema"})

	require.NoError(t, rootCmd.Execute())

	var schema commandSchema
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &schema))
	return schema
}

func TestSpecNewSchemaDocumentsNameAndOptionalID(t *testing.T) {
	schema := runSpecNewSchemaForTest(t)

	require.Contains(t, schema.Input.Properties, "name")
	require.Contains(t, schema.Input.Properties, "id")
	require.Equal(t, []string{"name"}, schema.Input.Required)
	require.Equal(t, spec.MaxIdentifierPartLength, schema.Input.Properties["name"].MaxLen)
	require.Equal(t, spec.MaxIdentifierPartLength, schema.Input.Properties["id"].MaxLen)
}

func TestSpecNew_DefaultUsesTimestampPrefix(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	result, err := runSpecNewForTest(t, "--data", `{"name":"Billing.Export"}`)
	require.NoError(t, err)

	require.Equal(t, "overview", result.Step)
	require.Regexp(t, regexp.MustCompile(`^\d{14}-billing-export$`), result.SpecName)
	require.Equal(t, filepath.Join(dir, ".spektacular", "specs", result.SpecName+".md"), result.SpecPath)
	require.FileExists(t, result.SpecPath)
}

func TestSpecNew_TimestampCollisionBumpsSeconds(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	setSpecIdentifierNow(t, time.Date(2026, time.May, 9, 1, 2, 3, 0, time.UTC))
	writeSpecCommandFile(t, dir, "20260509010203-billing-export")

	result, err := runSpecNewForTest(t, "--data", `{"name":"billing-export"}`)
	require.NoError(t, err)

	require.Equal(t, "20260509010204-billing-export", result.SpecName)
	require.FileExists(t, filepath.Join(dir, ".spektacular", "specs", "20260509010203-billing-export.md"))
	require.FileExists(t, result.SpecPath)
}

func TestSpecNew_ExplicitIDOverridesGeneratedMethod(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeSpecCommandConfig(t, dir, "spec:\n  id_method: counter\n")

	result, err := runSpecNewForTest(t, "--data", `{"name":"Billing Export","id":"EXT.User@123"}`)
	require.NoError(t, err)

	require.Equal(t, "ext-user-123-billing-export", result.SpecName)
	require.FileExists(t, result.SpecPath)
}

func TestSpecNew_ExternalModeWithIDCreatesSpec(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeSpecCommandConfig(t, dir, "spec:\n  id_method: external\n")

	result, err := runSpecNewForTest(t, "--data", `{"name":"Billing Export","id":"EXT.User@123"}`)
	require.NoError(t, err)

	require.Equal(t, "ext-user-123-billing-export", result.SpecName)
	require.FileExists(t, result.SpecPath)
}

func TestSpecNew_ExternalModeRequiresIDWithoutSideEffects(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	dataDir := filepath.Join(dir, ".spektacular")
	writeSpecCommandConfig(t, dir, "spec:\n  id_method: external\n")

	_, err := runSpecNewForTest(t, "--data", `{"name":"Billing Export"}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "id is required")
	require.NoDirExists(t, filepath.Join(dataDir, "specs"))
	require.NoFileExists(t, filepath.Join(dataDir, "state.json"))
}

func TestSpecNew_CounterModeUsesNextValueFromStore(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeSpecCommandConfig(t, dir, "spec:\n  id_method: counter\n")
	writeSpecCommandFile(t, dir, "000007_old-feature")

	result, err := runSpecNewForTest(t, "--data", `{"name":"billing-export"}`)
	require.NoError(t, err)

	require.Equal(t, "000008_billing-export", result.SpecName)
	require.FileExists(t, result.SpecPath)
}

func TestSpecNew_CounterModeCollisionBumpsValue(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeSpecCommandConfig(t, dir, "spec:\n  id_method: counter\n")
	writeSpecCommandFile(t, dir, "000007_old-feature")
	writeSpecCommandFile(t, dir, "000008_billing-export")

	result, err := runSpecNewForTest(t, "--data", `{"name":"billing-export"}`)
	require.NoError(t, err)

	require.Equal(t, "000009_billing-export", result.SpecName)
	require.FileExists(t, result.SpecPath)
}

func TestSpecNew_DryRunReportsCanonicalNameWithoutWrites(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	dataDir := filepath.Join(dir, ".spektacular")
	writeSpecCommandConfig(t, dir, "spec:\n  id_method: counter\n")
	writeSpecCommandFile(t, dir, "000007_old-feature")

	result, err := runSpecNewForTest(t, "--dry-run", "--data", `{"name":"billing-export"}`)
	require.NoError(t, err)

	require.Equal(t, "000008_billing-export", result.SpecName)
	require.Equal(t, filepath.Join(dataDir, "specs", "000008_billing-export.md"), result.SpecPath)
	require.NoFileExists(t, result.SpecPath)
	require.NoFileExists(t, filepath.Join(dataDir, "state.json"))
}

func TestSpecNew_ValidationFailuresLeaveNoSpecOrState(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{
			name: "untrimmed name",
			data: `{"name":" billing"}`,
			want: "leading or trailing whitespace",
		},
		{
			name: "path separator id",
			data: `{"name":"billing","id":"bad/id"}`,
			want: "path separators",
		},
		{
			name: "untrimmed id",
			data: `{"name":"billing","id":"ext "}`,
			want: "leading or trailing whitespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			dataDir := filepath.Join(dir, ".spektacular")

			_, err := runSpecNewForTest(t, "--data", tt.data)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.want)
			require.NoDirExists(t, filepath.Join(dataDir, "specs"))
			require.NoFileExists(t, filepath.Join(dataDir, "state.json"))
		})
	}
}

func TestSpecNew_RejectsUnknownConfiguredIDMethod(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	dataDir := filepath.Join(dir, ".spektacular")
	require.NoError(t, os.MkdirAll(dataDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "config.yaml"), []byte("spec:\n  id_method: unsupported\n"), 0o644))

	setupImplementCmd(t)
	rootCmd.SetArgs([]string{"spec", "new", "--data", `{"name":"fixture"}`})

	err := rootCmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "spec.id_method")
	require.NoFileExists(t, filepath.Join(dataDir, "specs", "fixture.md"))
	require.NoFileExists(t, filepath.Join(dataDir, "state.json"))
}

// fixedResumeTime is a deterministic timestamp for seeded in-progress state so
// byte-for-byte oracles never depend on time.Now().
var fixedResumeTime = time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)

func TestSpecNew_InProgressReturnsResumeReportAndPreservesState(t *testing.T) {
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

	before, err := os.ReadFile(filepath.Join(dataDir, "state.json"))
	require.NoError(t, err)

	resetSpecCommandFlags(t)
	stdout, _ := setupImplementCmd(t)
	rootCmd.SetArgs([]string{"spec", "new", "--data", `{"name":"whatever"}`})
	require.NoError(t, rootCmd.Execute())

	var r ResumeReport
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &r))
	require.True(t, r.Resumable)
	require.Equal(t, "spec", r.Kind)
	require.Equal(t, "000024_resume", r.Name)
	require.Equal(t, "overview", r.CurrentStep)
	require.NotEmpty(t, r.Instruction)

	after, err := os.ReadFile(filepath.Join(dataDir, "state.json"))
	require.NoError(t, err)
	require.Equal(t, before, after)

	require.NoDirExists(t, filepath.Join(dataDir, "specs"))
}

func TestSpecNew_ForceStartsFreshOverInProgress(t *testing.T) {
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

	resetSpecCommandFlags(t)
	stdout, _ := setupImplementCmd(t)
	rootCmd.SetArgs([]string{"spec", "new", "--force", "--data", `{"name":"billing"}`})
	require.NoError(t, rootCmd.Execute())

	var result specCommandResult
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	require.Equal(t, "overview", result.Step)
	require.FileExists(t, result.SpecPath)
}

func TestSpecNew_CleanDirHasNoResumeReport(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	resetSpecCommandFlags(t)
	stdout, _ := setupImplementCmd(t)
	rootCmd.SetArgs([]string{"spec", "new", "--data", `{"name":"billing"}`})
	require.NoError(t, rootCmd.Execute())

	var r ResumeReport
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &r))
	require.False(t, r.Resumable)
}

func TestSpecNew_KindlessInProgressStateErrorsWithoutClobber(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	dataDir := filepath.Join(dir, ".spektacular")

	writeInProgressState(t, dataDir, workflow.State{
		CurrentStep:    "overview",
		CompletedSteps: []string{"new"},
		CreatedAt:      fixedResumeTime,
		UpdatedAt:      fixedResumeTime,
		Data:           map[string]any{"name": "legacy"},
	})

	before, err := os.ReadFile(filepath.Join(dataDir, "state.json"))
	require.NoError(t, err)

	resetSpecCommandFlags(t)
	setupImplementCmd(t)
	rootCmd.SetArgs([]string{"spec", "new", "--data", `{"name":"whatever"}`})
	err = rootCmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "kind marker")

	after, err := os.ReadFile(filepath.Join(dataDir, "state.json"))
	require.NoError(t, err)
	require.Equal(t, before, after)
}
