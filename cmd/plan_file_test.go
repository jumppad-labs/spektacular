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

	srcPath := filepath.Join(t.TempDir(), "source.md")
	require.NoError(t, os.WriteFile(srcPath, []byte("plan body"), 0o644))

	setupImplementCmd(t)
	rootCmd.SetArgs([]string{"plan", "file", "write", "feature/plan.md", "--from", srcPath})

	require.NoError(t, rootCmd.Execute())

	content, err := os.ReadFile(filepath.Join(dir, "docs", "plans", "feature", "plan.md"))
	require.NoError(t, err)
	require.Equal(t, "plan body", string(content))
}

// TestPlanFileWrite_PreservesProblematicCharacters asserts that the bytes
// written to the destination are byte-identical to the source bytes, even when
// the source contains shell-sensitive characters and embedded newlines.
func TestPlanFileWrite_PreservesProblematicCharacters(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeSpecCommandConfig(t, dir, "plan:\n  config:\n    directory: docs/plans\n")

	body := []byte("line with `backticks` and $dollar and 'single' and \"double\" quotes\nsecond line\n")
	srcPath := filepath.Join(t.TempDir(), "source.md")
	require.NoError(t, os.WriteFile(srcPath, body, 0o644))

	setupImplementCmd(t)
	rootCmd.SetArgs([]string{"plan", "file", "write", "feature/plan.md", "--from", srcPath})

	require.NoError(t, rootCmd.Execute())

	dstPath := filepath.Join(dir, "docs", "plans", "feature", "plan.md")
	content, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	require.Equal(t, body, content)
}

// TestPlanFileWrite_MissingSourceErrors asserts that pointing `--from` at a
// non-existent path returns an error referencing the offending path and does
// not create the destination file.
func TestPlanFileWrite_MissingSourceErrors(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeSpecCommandConfig(t, dir, "plan:\n  config:\n    directory: docs/plans\n")

	srcPath := filepath.Join(t.TempDir(), "missing.md")

	setupImplementCmd(t)
	rootCmd.SetArgs([]string{"plan", "file", "write", "feature/plan.md", "--from", srcPath})

	err := rootCmd.Execute()
	require.Error(t, err)
	require.ErrorContains(t, err, srcPath)
	require.NoFileExists(t, filepath.Join(dir, "docs", "plans", "feature", "plan.md"))
}

// TestPlanFileWrite_PreservesSourceFile asserts that a successful write leaves
// the source file's bytes unchanged.
func TestPlanFileWrite_PreservesSourceFile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeSpecCommandConfig(t, dir, "plan:\n  config:\n    directory: docs/plans\n")

	body := []byte("original source bytes")
	srcPath := filepath.Join(t.TempDir(), "source.md")
	require.NoError(t, os.WriteFile(srcPath, body, 0o644))

	setupImplementCmd(t)
	rootCmd.SetArgs([]string{"plan", "file", "write", "feature/plan.md", "--from", srcPath})

	require.NoError(t, rootCmd.Execute())

	after, err := os.ReadFile(srcPath)
	require.NoError(t, err)
	require.Equal(t, body, after)
}

// TestPlanFileWrite_PipedStdinWithoutFromFails asserts that omitting `--from`
// fails the command even when stdin has data piped in, and that the
// destination file is not created.
func TestPlanFileWrite_PipedStdinWithoutFromFails(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeSpecCommandConfig(t, dir, "plan:\n  config:\n    directory: docs/plans\n")

	setupImplementCmd(t)
	rootCmd.SetIn(strings.NewReader("ignored"))
	t.Cleanup(func() { rootCmd.SetIn(nil) })
	rootCmd.SetArgs([]string{"plan", "file", "write", "feature/plan.md"})

	err := rootCmd.Execute()
	require.Error(t, err)
	require.NoFileExists(t, filepath.Join(dir, "docs", "plans", "feature", "plan.md"))
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
