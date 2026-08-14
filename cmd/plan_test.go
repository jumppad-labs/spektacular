package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jumppad-labs/spektacular/internal/config"
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
	writeSpecCommandConfig(t, dir, "")

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

// repoRoster projects each configured repo's identity plus its own
// descriptive metadata (sourced the same way `repo list` sources it: from
// the repo's own repo.yaml when it is present on disk, absent otherwise)
// into the shape the plan workflow's templates render.

// A materialized repo with its own repo.yaml carrying descriptive metadata
// produces a roster entry with the full description/role/tags/deployment,
// tags joined into a comma-separated string.
func TestRepoRoster_MaterializedRepoReportsOwnMetadata(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "vendor", "docs")
	require.NoError(t, writeDirRepoFootprint(t, target, config.RepoConfig{
		Description: "the documentation repo",
		Role:        "documentation",
		Tags:        []string{"docs", "markdown"},
		Deployment:  "static site on the CDN",
	}))

	cfg := config.NewDefault()
	cfg.Name = "testproj"
	cfg.Repos = []config.RepoEntry{
		{Name: "docs", Local: target},
	}

	roster := repoRoster(cfg, root)
	require.Len(t, roster, 1)
	entry := roster[0]
	require.Equal(t, "docs", entry["name"])
	require.Equal(t, "the documentation repo", entry["description"])
	require.Equal(t, "documentation", entry["role"])
	require.Equal(t, "docs, markdown", entry["tags"])
	require.Equal(t, "static site on the CDN", entry["deployment"])
}

// An unmaterialized, address-only repo produces a roster entry with the name
// only — descriptive fields are all empty strings, and no error occurs.
func TestRepoRoster_UnmaterializedRepoReportsNameOnly(t *testing.T) {
	root := t.TempDir()
	cfg := config.NewDefault()
	cfg.Name = "testproj"
	cfg.Repos = []config.RepoEntry{
		{Name: "remote-only", Address: "https://example.invalid/remote-only.git"},
	}

	roster := repoRoster(cfg, root)
	require.Len(t, roster, 1)
	entry := roster[0]
	require.Equal(t, "remote-only", entry["name"])
	require.Equal(t, "", entry["description"])
	require.Equal(t, "", entry["role"])
	require.Equal(t, "", entry["tags"])
	require.Equal(t, "", entry["deployment"])
}

// writeDirRepoFootprint creates dir and a .spektacular/repo.yaml inside it
// carrying the given descriptive metadata, mirroring a materialized repo's
// on-disk footprint.
func writeDirRepoFootprint(t *testing.T, dir string, meta config.RepoConfig) error {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".spektacular"), 0o755); err != nil {
		return err
	}
	repoCfg := config.NewDefaultRepoConfig()
	repoCfg.Description = meta.Description
	repoCfg.Role = meta.Role
	repoCfg.Tags = meta.Tags
	repoCfg.Deployment = meta.Deployment
	return repoCfg.ToYAMLFile(filepath.Join(dir, ".spektacular", config.RepoConfigFileName))
}
