package project

import (
	"os"
	"path/filepath"
	"testing"

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
}

func TestInit_CreatesGitignore(t *testing.T) {
	dir := t.TempDir()
	err := Init(dir, false)
	require.NoError(t, err)

	gitignorePath := filepath.Join(dir, ".spektacular", ".gitignore")
	_, err = os.Stat(gitignorePath)
	require.NoError(t, err)
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
