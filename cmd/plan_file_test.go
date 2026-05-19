package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPlanFileWrite_ResolvesConfiguredDirectory asserts `plan file write` lands
// the document under the configured (non-default) plan directory.
func TestPlanFileWrite_ResolvesConfiguredDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeSpecCommandConfig(t, dir, "plan:\n  config:\n    directory: docs/plans\n")

	setupImplementCmd(t)
	rootCmd.SetIn(strings.NewReader("plan body"))
	t.Cleanup(func() { rootCmd.SetIn(nil) })
	rootCmd.SetArgs([]string{"plan", "file", "write", "feature/plan.md"})

	require.NoError(t, rootCmd.Execute())

	content, err := os.ReadFile(filepath.Join(dir, "docs", "plans", "feature", "plan.md"))
	require.NoError(t, err)
	require.Equal(t, "plan body", string(content))
}

// TestPlanFileRead_ResolvesConfiguredDirectory asserts `plan file read` reads
// from the configured (non-default) plan directory.
func TestPlanFileRead_ResolvesConfiguredDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeSpecCommandConfig(t, dir, "plan:\n  config:\n    directory: docs/plans\n")

	planPath := filepath.Join(dir, "docs", "plans", "feature", "context.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(planPath), 0o755))
	require.NoError(t, os.WriteFile(planPath, []byte("stored context"), 0o644))

	stdout, _ := setupImplementCmd(t)
	rootCmd.SetArgs([]string{"plan", "file", "read", "feature/context.md"})

	require.NoError(t, rootCmd.Execute())
	require.Equal(t, "stored context", stdout.String())
}
