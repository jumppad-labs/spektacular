package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/stretchr/testify/require"
)

func TestInit_CreatesDirectoryStructure(t *testing.T) {
	dir := t.TempDir()

	err := Init(dir, false)
	require.NoError(t, err)

	expectedDirs := []string{
		".spektacular",
		".spektacular/plans",
		".spektacular/specs",
		".spektacular/knowledge",
		".spektacular/knowledge/learnings",
		".spektacular/knowledge/architecture",
		".spektacular/knowledge/gotchas",
	}
	for _, d := range expectedDirs {
		info, err := os.Stat(filepath.Join(dir, d))
		require.NoError(t, err, "expected dir %s to exist", d)
		require.True(t, info.IsDir(), "%s should be a directory", d)
	}
}

func TestInit_CreatesConfigFile(t *testing.T) {
	dir := t.TempDir()
	err := Init(dir, false)
	require.NoError(t, err)

	configPath := filepath.Join(dir, ".spektacular", "config.yaml")
	_, err = os.Stat(configPath)
	require.NoError(t, err, "config.yaml should exist")

	cfg, err := config.FromYAMLFile(configPath)
	require.NoError(t, err)
	require.Equal(t, "timestamp", cfg.Spec.IDMethod)
	require.Equal(t, 0, cfg.Spec.Counter)
	require.Equal(t, "local", cfg.Artifacts.Backend)
	require.Equal(t, "cache/notion", cfg.Artifacts.CacheDir)
}

func TestInitWithOptions_NotionModeOmitsLocalArtifactDirsAndCreatesCache(t *testing.T) {
	dir := t.TempDir()
	cfg := config.NewDefault()
	cfg.Spec.IDMethod = config.SpecIDMethodExternal
	cfg.Artifacts.Backend = config.ArtifactBackendNotion
	cfg.Artifacts.Notion.BasePageURL = "https://notion.example/base"
	cfg.Artifacts.Notion.SpecsDataSource = "collection://specs"
	cfg.Artifacts.Notion.PlansDataSource = "collection://plans"

	err := InitWithOptions(dir, InitOptions{
		Config:               cfg,
		CreateNotionCacheDir: true,
	})
	require.NoError(t, err)

	require.NoDirExists(t, filepath.Join(dir, ".spektacular", "plans"))
	require.NoDirExists(t, filepath.Join(dir, ".spektacular", "specs"))
	require.DirExists(t, filepath.Join(dir, ".spektacular", "cache", "notion"))
	require.DirExists(t, filepath.Join(dir, ".spektacular", "knowledge"))

	loaded, err := config.FromYAMLFile(filepath.Join(dir, ".spektacular", "config.yaml"))
	require.NoError(t, err)
	require.Equal(t, config.ArtifactBackendNotion, loaded.Artifacts.Backend)
	require.Equal(t, config.SpecIDMethodExternal, loaded.Spec.IDMethod)
	require.Equal(t, "collection://specs", loaded.Artifacts.Notion.SpecsDataSource)
	require.Equal(t, "collection://plans", loaded.Artifacts.Notion.PlansDataSource)
	require.Equal(t, "Spec ID", loaded.Artifacts.Notion.SpecIDProperty)
	require.Equal(t, "Plan ID", loaded.Artifacts.Notion.PlanIDProperty)
}

func TestInit_CreatesGitignore(t *testing.T) {
	dir := t.TempDir()
	err := Init(dir, false)
	require.NoError(t, err)

	gitignorePath := filepath.Join(dir, ".spektacular", ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	require.NoError(t, err)
	require.Contains(t, string(data), "cache/notion/")
}

func TestInit_CreatesConventionsMd(t *testing.T) {
	dir := t.TempDir()
	err := Init(dir, false)
	require.NoError(t, err)

	conventionsPath := filepath.Join(dir, ".spektacular", "knowledge", "conventions.md")
	_, err = os.Stat(conventionsPath)
	require.NoError(t, err)
}

func TestInit_CreatesKnowledgeREADMEs(t *testing.T) {
	dir := t.TempDir()
	err := Init(dir, false)
	require.NoError(t, err)

	for _, sub := range []string{"learnings", "architecture", "gotchas"} {
		readmePath := filepath.Join(dir, ".spektacular", "knowledge", sub, "README.md")
		data, err := os.ReadFile(readmePath)
		require.NoError(t, err, "README.md should exist in %s", sub)
		require.Contains(t, string(data), sub)
	}
}

func TestInit_AlreadyExists_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	err := Init(dir, false)
	require.NoError(t, err)

	err = Init(dir, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")
}

func TestInit_Force_OverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	err := Init(dir, false)
	require.NoError(t, err)

	err = Init(dir, true)
	require.NoError(t, err)
}
