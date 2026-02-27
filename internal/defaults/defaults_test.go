package defaults

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadFile_PlannerPrompt(t *testing.T) {
	data, err := ReadFile("agents/planner.md")
	require.NoError(t, err)
	require.NotEmpty(t, data)
}

func TestReadFile_ExecutorPrompt(t *testing.T) {
	data, err := ReadFile("agents/executor.md")
	require.NoError(t, err)
	require.NotEmpty(t, data)
}

func TestReadFile_SpecTemplate(t *testing.T) {
	data, err := ReadFile("spec-template.md")
	require.NoError(t, err)
	require.Contains(t, string(data), "{title}")
}

func TestReadFile_Conventions(t *testing.T) {
	data, err := ReadFile("conventions.md")
	require.NoError(t, err)
	require.NotEmpty(t, data)
}

func TestReadFile_Gitignore(t *testing.T) {
	data, err := ReadFile(".gitignore")
	require.NoError(t, err)
	require.NotEmpty(t, data)
}

func TestReadFile_NotFound_ReturnsError(t *testing.T) {
	_, err := ReadFile("nonexistent.md")
	require.Error(t, err)
}
