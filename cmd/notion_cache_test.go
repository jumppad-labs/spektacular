package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jumppad-labs/spektacular/internal/artifact"
	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/stretchr/testify/require"
)

func writeNotionCacheConfig(t *testing.T, dir string) {
	t.Helper()
	cfg := config.NewDefault()
	cfg.Spec.IDMethod = config.SpecIDMethodExternal
	cfg.Artifacts.Backend = config.ArtifactBackendNotion
	cfg.Artifacts.Notion.SpecsDataSource = "collection://specs"
	cfg.Artifacts.Notion.PlansDataSource = "collection://plans"
	dataDir := filepath.Join(dir, ".spektacular")
	require.NoError(t, os.MkdirAll(dataDir, 0o755))
	require.NoError(t, cfg.ToYAMLFile(filepath.Join(dataDir, "config.yaml")))
}

func cacheRemote(version string) artifact.RemoteMetadata {
	return artifact.RemoteMetadata{
		NotionURL:     "https://notion.so/spec",
		PageID:        "page-123",
		DataSourceURL: "collection://specs",
		ExternalID:    "SPEC-1",
		RemoteVersion: version,
		Title:         "Spec 1",
	}
}

func cacheData(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	require.NoError(t, err)
	return string(raw)
}

func runNotionCacheForTest(t *testing.T, args ...string) (map[string]any, error) {
	t.Helper()
	resetNotionCommandFlags(t)
	stdout, _ := setupImplementCmd(t)
	rootCmd.SetArgs(append([]string{"notion", "cache"}, args...))

	err := rootCmd.Execute()
	if err != nil {
		return nil, err
	}
	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	return result, nil
}

func TestNotionCachePullPrepareAndCommitPush(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeNotionCacheConfig(t, dir)

	pull, err := runNotionCacheForTest(t, "pull", "--data", cacheData(t, map[string]any{
		"kind":    "spec",
		"name":    "spec-1",
		"content": "baseline",
		"remote":  cacheRemote("v1"),
	}))
	require.NoError(t, err)
	require.Equal(t, "pulled", pull["status"])
	require.FileExists(t, filepath.Join(dir, ".spektacular", "cache", "notion", "specs", "spec-1.md"))

	require.NoError(t, os.WriteFile(filepath.Join(dir, ".spektacular", "cache", "notion", "specs", "spec-1.md"), []byte("local edit"), 0o644))
	ready, err := runNotionCacheForTest(t, "prepare-push", "--data", cacheData(t, map[string]any{
		"kind":           "spec",
		"name":           "spec-1",
		"remote_version": "v1",
	}))
	require.NoError(t, err)
	require.Equal(t, "ready", ready["status"])
	require.Equal(t, "local edit", ready["local_content"])

	committed, err := runNotionCacheForTest(t, "commit-push", "--data", cacheData(t, map[string]any{
		"kind":   "spec",
		"name":   "spec-1",
		"remote": cacheRemote("v2"),
	}))
	require.NoError(t, err)
	require.Equal(t, "committed", committed["status"])
	entry := committed["entry"].(map[string]any)
	require.Equal(t, "v2", entry["remote_version"])
}

func TestNotionCachePreparePushReturnsMergeRequestByExternalID(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeNotionCacheConfig(t, dir)
	_, err := runNotionCacheForTest(t, "pull", "--data", cacheData(t, map[string]any{
		"kind":    "spec",
		"name":    "spec-1",
		"content": "baseline",
		"remote":  cacheRemote("v1"),
	}))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".spektacular", "cache", "notion", "specs", "spec-1.md"), []byte("local edit"), 0o644))

	result, err := runNotionCacheForTest(t, "prepare-push", "--data", cacheData(t, map[string]any{
		"kind":           "spec",
		"external_id":    "SPEC-1",
		"remote_version": "v2",
		"remote_content": "remote edit",
	}))
	require.NoError(t, err)
	require.Equal(t, "merge_required", result["status"])
	merge := result["merge_request"].(map[string]any)
	require.Equal(t, "baseline", merge["baseline_content"])
	require.Equal(t, "local edit", merge["local_content"])
	require.Equal(t, "remote edit", merge["remote_content"])
	require.Equal(t, "Merge baseline, local, and remote content; then run resolve-merge before retrying prepare-push.", merge["required_action"])
}

func TestNotionCacheStatusCanLocateByURLAndPageID(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeNotionCacheConfig(t, dir)
	_, err := runNotionCacheForTest(t, "pull", "--data", cacheData(t, map[string]any{
		"kind":    "spec",
		"name":    "spec-1",
		"content": "baseline",
		"remote":  cacheRemote("v1"),
	}))
	require.NoError(t, err)

	byURL, err := runNotionCacheForTest(t, "status", "--data", `{"notion_url":"https://notion.so/spec"}`)
	require.NoError(t, err)
	require.Equal(t, "status", byURL["status"])
	entry := byURL["entry"].(map[string]any)
	require.Equal(t, "spec-1", entry["name"])

	byPageID, err := runNotionCacheForTest(t, "status", "--data", `{"page_id":"page-123"}`)
	require.NoError(t, err)
	require.Equal(t, "status", byPageID["status"])
}

func TestNotionCacheResolveMergeAllowsPrepareRetry(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeNotionCacheConfig(t, dir)
	_, err := runNotionCacheForTest(t, "pull", "--data", cacheData(t, map[string]any{
		"kind":    "spec",
		"name":    "spec-1",
		"content": "baseline",
		"remote":  cacheRemote("v1"),
	}))
	require.NoError(t, err)

	resolved, err := runNotionCacheForTest(t, "resolve-merge", "--data", cacheData(t, map[string]any{
		"kind":             "spec",
		"name":             "spec-1",
		"remote":           cacheRemote("v2"),
		"remote_content":   "remote edit",
		"resolved_content": "merged content",
	}))
	require.NoError(t, err)
	require.Equal(t, "resolved", resolved["status"])

	ready, err := runNotionCacheForTest(t, "prepare-push", "--data", cacheData(t, map[string]any{
		"kind":           "spec",
		"name":           "spec-1",
		"remote_version": "v2",
	}))
	require.NoError(t, err)
	require.Equal(t, "ready", ready["status"])
	require.Equal(t, "merged content", ready["local_content"])
}

func TestNotionCacheErrorsForMissingManifestAndMalformedRemote(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeNotionCacheConfig(t, dir)

	_, err := runNotionCacheForTest(t, "prepare-push", "--data", `{"kind":"spec","name":"missing"}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")

	_, err = runNotionCacheForTest(t, "pull", "--data", cacheData(t, map[string]any{
		"kind":    "spec",
		"name":    "spec-1",
		"content": "baseline",
		"remote": map[string]string{
			"page_id": "page-123",
		},
	}))
	require.Error(t, err)
	require.Contains(t, err.Error(), "remote.notion_url")
}
