package plan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadKnowledge_EmptyDir_ReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	knowledgeDir := filepath.Join(dir, ".spektacular", "knowledge")
	require.NoError(t, os.MkdirAll(knowledgeDir, 0755))

	result := LoadKnowledge(dir)
	require.Empty(t, result)
}

func TestLoadKnowledge_LoadsMarkdownFiles(t *testing.T) {
	dir := t.TempDir()
	knowledgeDir := filepath.Join(dir, ".spektacular", "knowledge")
	require.NoError(t, os.MkdirAll(knowledgeDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(knowledgeDir, "arch.md"), []byte("# Architecture"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(knowledgeDir, "notes.md"), []byte("# Notes"), 0644))

	result := LoadKnowledge(dir)
	require.Len(t, result, 2)
	require.Equal(t, "# Architecture", result["arch.md"])
}

func TestLoadKnowledge_IgnoresNonMarkdown(t *testing.T) {
	dir := t.TempDir()
	knowledgeDir := filepath.Join(dir, ".spektacular", "knowledge")
	require.NoError(t, os.MkdirAll(knowledgeDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(knowledgeDir, "file.txt"), []byte("ignored"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(knowledgeDir, "file.md"), []byte("included"), 0644))

	result := LoadKnowledge(dir)
	require.Len(t, result, 1)
	_, hasMarkdown := result["file.md"]
	require.True(t, hasMarkdown)
}

func TestLoadKnowledge_MissingDir_ReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	result := LoadKnowledge(dir)
	require.Empty(t, result)
}

func TestWritePlanOutput_CreatesFileAndDir(t *testing.T) {
	dir := t.TempDir()
	planDir := filepath.Join(dir, ".spektacular", "plans", "my-spec")

	err := WritePlanOutput(planDir, "plan content here")
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(planDir, "plan.md"))
	require.NoError(t, err)
	require.Equal(t, "plan content here", string(data))
}

func TestLoadAgentPrompt_ReturnsContent(t *testing.T) {
	content := LoadAgentPrompt()
	require.NotEmpty(t, content)
}
