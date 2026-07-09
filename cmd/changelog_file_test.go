package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChangelogFileWriteRead_RoundTrips(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeSpecCommandConfig(t, dir, "changelog:\n  config:\n    directory: docs/changelog\n")

	srcPath := filepath.Join(t.TempDir(), "source.md")
	require.NoError(t, os.WriteFile(srcPath, []byte("changelog body"), 0o644))

	setupImplementCmd(t)
	rootCmd.SetArgs([]string{"changelog", "file", "write", "20260709000000-release-notes.md", "--from", srcPath})

	require.NoError(t, rootCmd.Execute())

	stdout, _ := setupImplementCmd(t)
	rootCmd.SetArgs([]string{"changelog", "file", "read", "20260709000000-release-notes.md"})

	require.NoError(t, rootCmd.Execute())
	require.Equal(t, "changelog body", stdout.String())
}

func TestChangelogFileList_ShowsAllWrittenRecords(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeSpecCommandConfig(t, dir, "changelog:\n  config:\n    directory: docs/changelog\n")

	srcPath := filepath.Join(t.TempDir(), "source.md")
	require.NoError(t, os.WriteFile(srcPath, []byte("entry body"), 0o644))

	setupImplementCmd(t)
	rootCmd.SetArgs([]string{"changelog", "file", "write", "20260709000000-alpha.md", "--from", srcPath})
	require.NoError(t, rootCmd.Execute())

	setupImplementCmd(t)
	rootCmd.SetArgs([]string{"changelog", "file", "write", "20260709000001-beta.md", "--from", srcPath})
	require.NoError(t, rootCmd.Execute())

	stdout, _ := setupImplementCmd(t)
	rootCmd.SetArgs([]string{"changelog", "file", "list"})

	require.NoError(t, rootCmd.Execute())

	var result struct {
		Files []string `json:"files"`
	}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	require.ElementsMatch(t, []string{"20260709000000-alpha.md", "20260709000001-beta.md"}, result.Files)
}
