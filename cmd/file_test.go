package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSpecFileWrite_ResolvesConfiguredDirectory asserts `spec file write` lands
// the file under the configured (non-default) spec directory rather than the
// .spektacular data directory.
func TestSpecFileWrite_ResolvesConfiguredDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeSpecCommandConfig(t, dir, "spec:\n  config:\n    directory: docs/specs\n")

	setupImplementCmd(t)
	rootCmd.SetIn(strings.NewReader("spec body"))
	t.Cleanup(func() { rootCmd.SetIn(nil) })
	rootCmd.SetArgs([]string{"spec", "file", "write", "feature.md"})

	require.NoError(t, rootCmd.Execute())

	content, err := os.ReadFile(filepath.Join(dir, "docs", "specs", "feature.md"))
	require.NoError(t, err)
	require.Equal(t, "spec body", string(content))
}

// TestSpecFileRead_ResolvesConfiguredDirectory asserts `spec file read` reads
// from the configured (non-default) spec directory.
func TestSpecFileRead_ResolvesConfiguredDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeSpecCommandConfig(t, dir, "spec:\n  config:\n    directory: docs/specs\n")

	specPath := filepath.Join(dir, "docs", "specs", "feature.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(specPath), 0o755))
	require.NoError(t, os.WriteFile(specPath, []byte("stored body"), 0o644))

	stdout, _ := setupImplementCmd(t)
	rootCmd.SetArgs([]string{"spec", "file", "read", "feature.md"})

	require.NoError(t, rootCmd.Execute())
	require.Equal(t, "stored body", stdout.String())
}
