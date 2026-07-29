package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jumppad-labs/spektacular/internal/metadata"
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
	// The stored file now carries a frontmatter block; read returns the raw
	// bytes, so strip the block before asserting on body preservation.
	meta, body, err := metadata.Split(stdout.Bytes())
	require.NoError(t, err)
	require.NotNil(t, meta, "write must produce a frontmatter block")
	require.Equal(t, metadata.StatusInProgress, meta.Status)
	require.Equal(t, "changelog body", string(body))
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
		Files []struct {
			Name string `json:"name"`
		} `json:"files"`
	}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	names := make([]string, len(result.Files))
	for i, f := range result.Files {
		names[i] = f.Name
	}
	require.ElementsMatch(t, []string{"20260709000000-alpha.md", "20260709000001-beta.md"}, names)
}
