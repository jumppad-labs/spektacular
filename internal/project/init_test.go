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
}

func TestInit_CreatesGitignore(t *testing.T) {
	dir := t.TempDir()
	err := Init(dir, false)
	require.NoError(t, err)

	gitignorePath := filepath.Join(dir, ".spektacular", ".gitignore")
	_, err = os.Stat(gitignorePath)
	require.NoError(t, err)
}

func TestInit_SeedsConventionsDirectory(t *testing.T) {
	dir := t.TempDir()
	err := Init(dir, false)
	require.NoError(t, err)

	conventionsDir := filepath.Join(dir, ".spektacular", "knowledge", "conventions")
	info, err := os.Stat(conventionsDir)
	require.NoError(t, err, "conventions directory should exist")
	require.True(t, info.IsDir(), "conventions should be a directory")

	starterPath := filepath.Join(conventionsDir, "conventions.md")
	data, err := os.ReadFile(starterPath)
	require.NoError(t, err, "starter conventions.md should exist")
	require.NotEmpty(t, data, "starter conventions.md should not be empty")

	flatPath := filepath.Join(dir, ".spektacular", "knowledge", "conventions.md")
	_, err = os.Stat(flatPath)
	require.True(t, os.IsNotExist(err), "old flat conventions.md should not exist")
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

// TestInit_DefaultConfig_CreatesSpecsAndPlansDirs asserts that with default
// config Init still creates the conventional specs/plans directories
// (Phase 2.2, criterion 3).
func TestInit_DefaultConfig_CreatesSpecsAndPlansDirs(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, Init(dir, false))

	for _, d := range []string{".spektacular/specs", ".spektacular/plans"} {
		info, err := os.Stat(filepath.Join(dir, d))
		require.NoError(t, err, "expected default dir %s", d)
		require.True(t, info.IsDir())
	}
}

// TestInit_NonDefaultConfig_CreatesConfiguredDirs writes a config.yaml with
// non-default spec/plan directories into a temp project, runs Init, and asserts
// the configured directories are created on disk (Phase 2.2, criterion 3).
// Spec and plan directories are project-root-relative paths.
func TestInit_NonDefaultConfig_CreatesConfiguredDirs(t *testing.T) {
	dir := t.TempDir()
	spektacularDir := filepath.Join(dir, ".spektacular")
	require.NoError(t, os.MkdirAll(spektacularDir, 0755))

	cfg := config.NewDefault()
	cfg.Spec.Config.Directory = "my-specs"
	cfg.Plan.Config.Directory = "my-plans"
	require.NoError(t, cfg.ToYAMLFile(filepath.Join(spektacularDir, "config.yaml")))

	// Force is required because .spektacular already exists.
	require.NoError(t, Init(dir, true))

	for _, d := range []string{"my-specs", "my-plans"} {
		info, err := os.Stat(filepath.Join(dir, d))
		require.NoError(t, err, "expected configured dir %s", d)
		require.True(t, info.IsDir(), "%s should be a directory", d)
	}
	// The default directories must NOT be created.
	_, err := os.Stat(filepath.Join(dir, config.DefaultSpecDir))
	require.True(t, os.IsNotExist(err), "default specs dir should not be created")
	_, err = os.Stat(filepath.Join(dir, config.DefaultPlanDir))
	require.True(t, os.IsNotExist(err), "default plans dir should not be created")
}

// TestInit_CreatesProjectKnowledgeSourceOnly asserts that Init creates the
// directory for the configured project knowledge source but leaves team and
// global sources alone — those are shared and expected to exist independently.
func TestInit_CreatesProjectKnowledgeSourceOnly(t *testing.T) {
	dir := t.TempDir()
	spektacularDir := filepath.Join(dir, ".spektacular")
	require.NoError(t, os.MkdirAll(spektacularDir, 0755))

	cfg := config.NewDefault()
	cfg.Knowledge = config.KnowledgeConfig{
		Sources: []config.SourceConfig{
			{
				Scope:    "project",
				Provider: config.ProviderFile,
				Config:   config.FileKnowledgeConfig{Location: ".spektacular/team-notes"},
			},
			{
				Scope:    "team",
				Provider: config.ProviderFile,
				Config:   config.FileKnowledgeConfig{Location: "shared/team-kb"},
			},
		},
	}
	require.NoError(t, cfg.ToYAMLFile(filepath.Join(spektacularDir, "config.yaml")))

	// Force is required because .spektacular already exists.
	require.NoError(t, Init(dir, true))

	// The project source's configured directory is created.
	info, err := os.Stat(filepath.Join(dir, ".spektacular", "team-notes"))
	require.NoError(t, err, "project knowledge source directory should be created")
	require.True(t, info.IsDir())

	// The team source's directory is NOT created by init.
	_, err = os.Stat(filepath.Join(dir, "shared", "team-kb"))
	require.True(t, os.IsNotExist(err), "team knowledge source dir should not be created by init")
}

// TestInit_DefaultConfig_CreatesProjectKnowledgeDir asserts that with no
// knowledge config the synthesised default project source directory exists.
func TestInit_DefaultConfig_CreatesProjectKnowledgeDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, Init(dir, false))

	info, err := os.Stat(filepath.Join(dir, ".spektacular", "knowledge"))
	require.NoError(t, err, "default project knowledge directory should exist")
	require.True(t, info.IsDir())
}
