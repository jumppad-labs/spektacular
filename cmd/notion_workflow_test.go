package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jumppad-labs/spektacular/internal/artifact"
	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/store"
	"github.com/stretchr/testify/require"
)

func remoteForWorkflow(kind, id, version string) artifact.RemoteMetadata {
	return artifact.RemoteMetadata{
		NotionURL:     "https://notion.so/" + kind + "/" + id,
		PageID:        kind + "-page-" + id,
		DataSourceURL: "collection://" + kind,
		ExternalID:    id,
		RemoteVersion: version,
		Title:         id,
	}
}

func TestSpecNew_NotionUsesRemoteSpecIDAndCreatesCacheEntry(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeNotionCacheConfig(t, dir)

	result, err := runSpecNewForTest(t, "--data", cacheData(t, map[string]any{
		"name":   "Billing",
		"remote": remoteForWorkflow("specs", "SPEC-1", "v1"),
	}))
	require.NoError(t, err)

	require.Equal(t, "spec-1-billing", result.SpecName)
	require.Contains(t, result.SpecPath, filepath.Join(".spektacular", "cache", "notion", "specs", "spec-1-billing.md"))
	require.FileExists(t, filepath.Join(dir, ".spektacular", "cache", "notion", "specs", "spec-1-billing.md"))

	manifest, err := artifact.LoadManifest(store.NewFileStore(filepath.Join(dir, ".spektacular")), mustLoadConfig(t, dir))
	require.NoError(t, err)
	entry := manifest.Entries[artifact.Key(artifact.KindSpec, "spec-1-billing")]
	require.Equal(t, "SPEC-1", entry.ExternalID)
	require.Equal(t, "v1", entry.RemoteVersion)
}

func TestPlanWorkflow_NotionWritesPlanContextResearchCacheEntries(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeNotionCacheConfig(t, dir)
	name := "plan-1-billing"

	runPlanCommand(t, "new", "--data", cacheData(t, map[string]any{"name": name}))
	for _, step := range []string{
		"discovery",
		"architecture",
		"components",
		"data_structures",
		"implementation_detail",
		"dependencies",
		"testing_approach",
		"milestones",
		"phases",
		"open_questions",
		"out_of_scope",
		"verification",
	} {
		runPlanCommand(t, "goto", "--data", cacheData(t, map[string]any{"step": step}))
	}
	runPlanCommand(t, "goto", "--data", cacheData(t, map[string]any{
		"step":          "write_plan",
		"plan_template": "# Plan",
		"remote":        remoteForWorkflow("plans", "PLAN-1", "v1"),
	}))
	runPlanCommand(t, "goto", "--data", cacheData(t, map[string]any{
		"step":             "write_context",
		"context_template": "# Context",
		"context_remote":   remoteForWorkflow("plans", "CTX-1", "v1"),
	}))
	runPlanCommand(t, "goto", "--data", cacheData(t, map[string]any{
		"step":              "write_research",
		"research_template": "# Research",
		"research_remote":   remoteForWorkflow("plans", "RSH-1", "v1"),
	}))

	require.FileExists(t, filepath.Join(dir, ".spektacular", "cache", "notion", "plans", name, "plan.md"))
	require.FileExists(t, filepath.Join(dir, ".spektacular", "cache", "notion", "plans", name, "context.md"))
	require.FileExists(t, filepath.Join(dir, ".spektacular", "cache", "notion", "plans", name, "research.md"))
	require.NoFileExists(t, filepath.Join(dir, ".spektacular", "plans", name, "plan.md"))

	manifest, err := artifact.LoadManifest(store.NewFileStore(filepath.Join(dir, ".spektacular")), mustLoadConfig(t, dir))
	require.NoError(t, err)
	require.Equal(t, "PLAN-1", manifest.Entries[artifact.Key(artifact.KindPlan, name)].ExternalID)
	require.Equal(t, "CTX-1", manifest.Entries[artifact.Key(artifact.KindContext, name)].ExternalID)
	require.Equal(t, "RSH-1", manifest.Entries[artifact.Key(artifact.KindResearch, name)].ExternalID)
}

func TestImplementNew_NotionRequiresCachedPlan(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeNotionCacheConfig(t, dir)
	resetImplementCommandFlags(t)
	setupImplementCmd(t)
	rootCmd.SetArgs([]string{"implement", "new", "--data", `{"name":"missing"}`})
	err := rootCmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "plan file not found")

	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".spektacular", "cache", "notion", "plans", "cached"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".spektacular", "cache", "notion", "plans", "cached", "plan.md"), []byte("# Plan"), 0o644))
	resetImplementCommandFlags(t)
	stdout, _ := setupImplementCmd(t)
	rootCmd.SetArgs([]string{"implement", "new", "--data", `{"name":"cached"}`})
	require.NoError(t, rootCmd.Execute())

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	require.Equal(t, "read_plan", result["step"])
	require.Contains(t, result["plan_path"], filepath.Join(".spektacular", "cache", "notion", "plans", "cached", "plan.md"))
}

func resetImplementCommandFlags(t *testing.T) {
	t.Helper()
	reset := func() {
		require.NoError(t, implementCmd.PersistentFlags().Set("schema", "false"))
		require.NoError(t, implementCmd.PersistentFlags().Set("dry-run", "false"))
		require.NoError(t, implementNewCmd.Flags().Set("data", ""))
		require.NoError(t, implementGotoCmd.Flags().Set("data", ""))
	}
	reset()
	t.Cleanup(reset)
}

func runPlanCommand(t *testing.T, args ...string) map[string]any {
	t.Helper()
	resetPlanCommandFlags(t)
	stdout, _ := setupImplementCmd(t)
	rootCmd.SetArgs(append([]string{"plan"}, args...))
	require.NoError(t, rootCmd.Execute())
	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	return result
}

func resetPlanCommandFlags(t *testing.T) {
	t.Helper()
	reset := func() {
		require.NoError(t, planCmd.PersistentFlags().Set("schema", "false"))
		require.NoError(t, planCmd.PersistentFlags().Set("dry-run", "false"))
		require.NoError(t, planNewCmd.Flags().Set("data", ""))
		require.NoError(t, planNewCmd.Flags().Set("stdin", ""))
		require.NoError(t, planNewCmd.Flags().Set("file", ""))
		require.NoError(t, planGotoCmd.Flags().Set("data", ""))
		require.NoError(t, planGotoCmd.Flags().Set("stdin", ""))
		require.NoError(t, planGotoCmd.Flags().Set("file", ""))
	}
	reset()
	t.Cleanup(reset)
}

func mustLoadConfig(t *testing.T, dir string) config.Config {
	t.Helper()
	cfg, err := config.FromYAMLFile(filepath.Join(dir, ".spektacular", "config.yaml"))
	require.NoError(t, err)
	return cfg
}
