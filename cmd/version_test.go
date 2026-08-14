package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jumppad-labs/spektacular/internal/output"
	"github.com/stretchr/testify/require"
)

// This file covers Phase 2.1 (the `version check` command): the pure
// classification helper, and the command's observable behavior through
// runRootCmd — the same wrapper Execute() uses in production.
//
// Every expected version string below is a hand-maintained oracle: the dev
// default the binary compiles with is the literal "0.1.0" (cmd/root.go's
// `version` var), asserted as that literal and never derived from the
// `version` var or versionString() at runtime.

// resetVersionCheckFlags clears the schema flag on `version check` between
// runs, mirroring resetSpecCommandFlags: cobra flag values persist on the
// package-level command, so a --schema run would otherwise leak into every
// later invocation in the same test binary.
func resetVersionCheckFlags(t *testing.T) {
	t.Helper()
	reset := func() {
		require.NoError(t, versionCheckCmd.Flags().Set("schema", "false"))
	}
	reset()
	t.Cleanup(reset)
}

// writeVersionCheckFile seeds dir/.spektacular/version with content.
func writeVersionCheckFile(t *testing.T, dir, content string) string {
	t.Helper()
	dataDir := filepath.Join(dir, ".spektacular")
	require.NoError(t, os.MkdirAll(dataDir, 0o755))
	path := filepath.Join(dataDir, "version")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

// runVersionCheckJSON runs `version check` and decodes stdout into a generic
// map so tests can assert both present and absent keys.
func runVersionCheckJSON(t *testing.T) (m map[string]any, code int) {
	t.Helper()
	resetVersionCheckFlags(t)
	stdout, stderr, code := runRootCmd(t, "version", "check")
	require.Empty(t, stderr)
	require.NoError(t, json.Unmarshal([]byte(stdout), &m))
	return m, code
}

// TestClassifyVersion pins the pure classification decision with hand-written
// expected values for every recorded-content shape the file can hold.
func TestClassifyVersion(t *testing.T) {
	tests := []struct {
		name          string
		recorded      string
		current       string
		wantStatus    string
		wantInstalled string
	}{
		{name: "exact match", recorded: "0.1.0", current: "0.1.0", wantStatus: "match", wantInstalled: "0.1.0"},
		{name: "match with trailing newline", recorded: "0.1.0\n", current: "0.1.0", wantStatus: "match", wantInstalled: "0.1.0"},
		{name: "match with surrounding spaces", recorded: "  0.1.0  ", current: "0.1.0", wantStatus: "match", wantInstalled: "0.1.0"},
		{name: "differing versions", recorded: "9.9.9", current: "0.1.0", wantStatus: "mismatch", wantInstalled: "9.9.9"},
		{name: "empty string", recorded: "", current: "0.1.0", wantStatus: "missing", wantInstalled: ""},
		{name: "whitespace-only string", recorded: " \n\t ", current: "0.1.0", wantStatus: "missing", wantInstalled: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, installed := classifyVersion(tt.recorded, tt.current)
			require.Equal(t, tt.wantStatus, status)
			require.Equal(t, tt.wantInstalled, installed)
		})
	}
}

// TestVersionCheck_Match: a version file recording the running binary's own
// version reports "match", both version fields, and — per the spec — no
// action text at all (the key must be absent from the JSON, not just empty).
func TestVersionCheck_Match(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeVersionCheckFile(t, dir, "0.1.0\n")

	m, code := runVersionCheckJSON(t)
	require.Equal(t, 0, code)
	require.Equal(t, false, m["error"])
	require.Equal(t, "match", m["status"])
	require.Equal(t, "0.1.0", m["installed_version"])
	require.Equal(t, "0.1.0", m["current_version"])
	require.NotContains(t, m, "action", "matching versions must carry no action text")
}

// TestVersionCheck_Mismatch: a recorded version differing from the binary's
// reports "mismatch" with a non-empty re-run instruction, still exit 0 —
// staleness is a report, not a failure. The command must also be strictly
// read-only: the version file's bytes and the .spektacular listing are
// identical before and after the invocation.
func TestVersionCheck_Mismatch(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	path := writeVersionCheckFile(t, dir, "9.9.9\n")

	before, err := os.ReadFile(path)
	require.NoError(t, err)
	entriesBefore, err := os.ReadDir(filepath.Join(dir, ".spektacular"))
	require.NoError(t, err)

	m, code := runVersionCheckJSON(t)
	require.Equal(t, 0, code)
	require.Equal(t, false, m["error"])
	require.Equal(t, "mismatch", m["status"])
	require.Equal(t, "9.9.9", m["installed_version"])
	require.Equal(t, "0.1.0", m["current_version"])
	action, ok := m["action"].(string)
	require.True(t, ok, "mismatch must carry an action string")
	require.NotEmpty(t, action)
	require.Contains(t, action, "re-run")

	// Read-only guarantee.
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, before, after, "version check must never rewrite the version file")
	entriesAfter, err := os.ReadDir(filepath.Join(dir, ".spektacular"))
	require.NoError(t, err)
	namesOf := func(entries []os.DirEntry) []string {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		return names
	}
	require.Equal(t, namesOf(entriesBefore), namesOf(entriesAfter), "version check must not create or remove files in .spektacular")
}

// Criterion 4: the stale-version remediation composes the correct init
// command from the recorded config — the configured command plus the recorded
// agent — after a real init has established both.
func TestVersionCheck_MismatchComposesInitCommandFromConfig(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	resetInitFlags(t)

	rootCmd.SetArgs([]string{"init", "claude"})
	require.NoError(t, rootCmd.Execute())
	writeVersionCheckFile(t, dir, "9.9.9\n")

	m, code := runVersionCheckJSON(t)
	require.Equal(t, 0, code)
	require.Equal(t, "mismatch", m["status"])
	action, ok := m["action"].(string)
	require.True(t, ok, "mismatch must carry an action string")
	require.Contains(t, action, "`spektacular init claude`",
		"the remediation must compose the configured command with the recorded agent")
}

// TestVersionCheck_Missing: no .spektacular directory (and hence no version
// file) is the pre-versioning install state — "missing", no
// installed_version key, and a re-run instruction, exit 0.
func TestVersionCheck_Missing(t *testing.T) {
	t.Chdir(t.TempDir())

	m, code := runVersionCheckJSON(t)
	require.Equal(t, 0, code)
	require.Equal(t, false, m["error"])
	require.Equal(t, "missing", m["status"])
	require.NotContains(t, m, "installed_version", "missing state has nothing installed to report")
	require.Equal(t, "0.1.0", m["current_version"])
	action, ok := m["action"].(string)
	require.True(t, ok, "missing must carry an action string")
	require.NotEmpty(t, action)
	require.Contains(t, action, "re-run")
}

// TestVersionCheck_WhitespaceOnlyFile: a version file that exists but holds
// only whitespace classifies as "missing", not a mismatch, exit 0.
func TestVersionCheck_WhitespaceOnlyFile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeVersionCheckFile(t, dir, " \n\t\n")

	m, code := runVersionCheckJSON(t)
	require.Equal(t, 0, code)
	require.Equal(t, false, m["error"])
	require.Equal(t, "missing", m["status"])
	require.NotContains(t, m, "installed_version")
}

// TestVersionCheck_UnreadableFileIsGenuineFault: a version file that exists
// but cannot be read (here: it is a directory, so os.ReadFile fails with a
// non-IsNotExist error) is a genuine fault — exit 1 with the error envelope
// and the dedicated version_file_unreadable code, never a silent "missing".
func TestVersionCheck_UnreadableFileIsGenuineFault(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".spektacular", "version"), 0o755))

	resetVersionCheckFlags(t)
	stdout, stderr, code := runRootCmd(t, "version", "check")
	require.Equal(t, 1, code)
	require.Empty(t, stderr)

	var er output.ErrorResponse
	require.NoError(t, json.Unmarshal([]byte(stdout), &er))
	require.True(t, er.IsError)
	require.Equal(t, "version_file_unreadable", er.Code)
}

// TestVersionCheck_Schema: --schema short-circuits before any filesystem
// access and prints the output contract, including the status enum.
func TestVersionCheck_Schema(t *testing.T) {
	t.Chdir(t.TempDir())
	resetVersionCheckFlags(t)

	stdout, stderr, code := runRootCmd(t, "version", "check", "--schema")
	require.Equal(t, 0, code)
	require.Empty(t, stderr)
	require.Contains(t, stdout, `"status"`)
	require.Contains(t, stdout, "match")
	require.Contains(t, stdout, "mismatch")
	require.Contains(t, stdout, "missing")
}
