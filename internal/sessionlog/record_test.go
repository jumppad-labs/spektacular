package sessionlog

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// readLines returns the non-empty lines of the file at path, split on '\n'.
func readLines(t *testing.T, path string) []string {
	t.Helper()

	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	require.NoError(t, scanner.Err())
	return lines
}

// TestSessionID covers the pure derivation logic in isolation: a nil
// snapshot, a snapshot with both fields empty (the zero value a fresh
// StateSnapshot literal would have), and a normal populated Kind/Name pair.
func TestSessionID(t *testing.T) {
	tests := []struct {
		name  string
		state *StateSnapshot
		want  string
	}{
		{
			name:  "nil state maps to the no-active-workflow sentinel",
			state: nil,
			want:  "no-active-workflow",
		},
		{
			name:  "both Kind and Name empty maps to the no-active-workflow sentinel",
			state: &StateSnapshot{Kind: "", Name: ""},
			want:  "no-active-workflow",
		},
		{
			name:  "populated Kind and Name join with a colon",
			state: &StateSnapshot{Kind: "spec", Name: "000001_billing"},
			want:  "spec:000001_billing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, SessionID(tt.state))
		})
	}
}

func TestRecordMultipleCallsProduceOrderedIncreasingSeq(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "session.log")

	Record(logPath, Event{Command: []string{"spek", "plan"}, ExitCode: 0})
	Record(logPath, Event{Command: []string{"spek", "implement"}, ExitCode: 0})
	Record(logPath, Event{Command: []string{"spek", "status"}, ExitCode: 1})

	lines := readLines(t, logPath)
	require.Len(t, lines, 3)

	wantCommands := [][]string{
		{"spek", "plan"},
		{"spek", "implement"},
		{"spek", "status"},
	}

	for i, line := range lines {
		var ev Event
		require.NoError(t, json.Unmarshal([]byte(line), &ev), "line %d must be independently parseable JSON", i)
		require.Equal(t, int64(i), ev.Seq, "line %d must carry seq %d", i, i)
		require.Equal(t, wantCommands[i], ev.Command, "line %d must preserve call order", i)
	}
}

func TestRecordRoundTripsAllFields(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "session.log")

	ev := Event{
		Timestamp:   time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		SessionID:   "implement:billing",
		Command:     []string{"spek", "implement", "--phase", "1.1"},
		DurationMS:  4210,
		ExitCode:    0,
		Response:    `{"step":"done"}`,
		StateBefore: &StateSnapshot{Kind: "implement", Name: "billing", CurrentStep: "analyze", CompletedSteps: []string{"read_plan"}},
		StateAfter:  &StateSnapshot{Kind: "implement", Name: "billing", CurrentStep: "implement", CompletedSteps: []string{"read_plan", "analyze"}},
		Advanced:    true,
	}

	Record(logPath, ev)

	lines := readLines(t, logPath)
	require.Len(t, lines, 1)

	// Hand-written oracle, independent of the Event struct being tested.
	want := `{
		"seq": 0,
		"timestamp": "2024-01-15T10:30:00Z",
		"session_id": "implement:billing",
		"command": ["spek", "implement", "--phase", "1.1"],
		"duration_ms": 4210,
		"exit_code": 0,
		"response": "{\"step\":\"done\"}",
		"state_before": {"kind": "implement", "name": "billing", "current_step": "analyze", "completed_steps": ["read_plan"]},
		"state_after": {"kind": "implement", "name": "billing", "current_step": "implement", "completed_steps": ["read_plan", "analyze"]},
		"advanced": true
	}`
	require.JSONEq(t, want, lines[0])
}

func TestRecordMissingFileFirstCallResultsInSeqZero(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "does-not-exist-yet", "session.log")

	_, err := os.Stat(logPath)
	require.True(t, os.IsNotExist(err), "precondition: log file must not exist yet")

	Record(logPath, Event{Command: []string{"spek", "spec"}})

	lines := readLines(t, logPath)
	require.Len(t, lines, 1)

	var ev Event
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &ev))
	require.Equal(t, int64(0), ev.Seq)
}

// TestRecordMkdirFailureDoesNotPanic covers the case where a plain file
// already occupies the path component that Record needs to MkdirAll as a
// directory, so directory creation fails outright.
func TestRecordMkdirFailureDoesNotPanic(t *testing.T) {
	dir := t.TempDir()

	// "blocker" exists as a regular file, so MkdirAll(".../blocker") for the
	// log's parent directory must fail.
	blockerPath := filepath.Join(dir, "blocker")
	require.NoError(t, os.WriteFile(blockerPath, []byte("not a directory"), 0644))

	logPath := filepath.Join(blockerPath, "session.log")

	require.NotPanics(t, func() {
		Record(logPath, Event{Command: []string{"spek", "plan"}})
	})

	// The blocker file must be untouched, and no log file should have been
	// created anywhere.
	data, err := os.ReadFile(blockerPath)
	require.NoError(t, err)
	require.Equal(t, "not a directory", string(data))

	// logPath can't exist: its parent ("blocker") is a plain file, so even
	// stat-ing it fails (ENOTDIR rather than ENOENT, hence not IsNotExist).
	_, err = os.Stat(logPath)
	require.Error(t, err)

	// A subsequent call against a valid path must still work normally,
	// proving the earlier failure left no corrupted state behind.
	okPath := filepath.Join(dir, "session.log")
	Record(okPath, Event{Command: []string{"spek", "status"}})
	lines := readLines(t, okPath)
	require.Len(t, lines, 1)
}

// TestLogFilePath_StartMintsFreshTimestampedFile asserts that when start is
// true, LogFilePath always mints a brand-new filename stamped with the
// given moment, the agent, and the session id — even when a file for the
// same session already exists (e.g. a `--force` restart of the same named
// workflow) — rather than reusing it.
func TestLogFilePath_StartMintsFreshTimestampedFile(t *testing.T) {
	dir := t.TempDir()
	earlier := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	later := earlier.Add(time.Hour)

	first := LogFilePath(dir, "claude", "plan:billing", true, earlier)
	require.NoError(t, os.WriteFile(first, []byte(`{}`+"\n"), 0644))

	second := LogFilePath(dir, "claude", "plan:billing", true, later)

	require.NotEqual(t, first, second, "a second start must mint a distinct file, not reuse the first")
	require.Contains(t, filepath.Base(first), "claude")
	require.Contains(t, filepath.Base(first), "plan_billing")
	require.Contains(t, filepath.Base(second), "plan_billing")
}

// TestLogFilePath_ResumeAppendsToLatestMatchingFile asserts that when start
// is false, LogFilePath finds and reuses the most recently started file for
// the same session id — the one a resumed workflow (or any other command
// sharing that session) should keep appending to — even when an older,
// unrelated file for the same session also exists on disk.
func TestLogFilePath_ResumeAppendsToLatestMatchingFile(t *testing.T) {
	dir := t.TempDir()
	older := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	newer := time.Date(2024, 1, 16, 9, 0, 0, 0, time.UTC)

	oldSession := LogFilePath(dir, "claude", "plan:billing", true, older)
	require.NoError(t, os.WriteFile(oldSession, []byte(`{}`+"\n"), 0644))
	newSession := LogFilePath(dir, "claude", "plan:billing", true, newer)
	require.NoError(t, os.WriteFile(newSession, []byte(`{}`+"\n"), 0644))

	resumed := LogFilePath(dir, "claude", "plan:billing", false, newer.Add(time.Minute))

	require.Equal(t, newSession, resumed, "resume must append to the most recently started session file, not an older one")
}

// TestLogFilePath_ResumeWithNoExistingFileMintsOne asserts a non-start call
// still mints a fresh file when no matching one exists yet — covering an ad
// hoc command that runs before any workflow's `new` has ever been recorded,
// and any session that predates this file-per-session scheme.
func TestLogFilePath_ResumeWithNoExistingFileMintsOne(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	got := LogFilePath(dir, "claude", "no-active-workflow", false, now)

	require.Contains(t, filepath.Base(got), "no-active-workflow")
	_, err := os.Stat(got)
	require.True(t, os.IsNotExist(err), "LogFilePath only resolves the path; it must not create the file itself")
}

// TestLogFilePath_DistinctSessionsNeverShareAFile asserts that two
// different session ids started at the exact same moment resolve to
// different files, so unrelated workflows are never interleaved into one
// log purely because they happened to start in the same second.
func TestLogFilePath_DistinctSessionsNeverShareAFile(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	billing := LogFilePath(dir, "claude", "plan:billing", true, now)
	widget := LogFilePath(dir, "claude", "plan:widget", true, now)

	require.NotEqual(t, billing, widget)
}

// TestRecordReadOnlyDirectoryDoesNotPanic covers the case where the log's
// parent directory already exists but is not writable, so file creation
// (rather than directory creation) is what fails.
func TestRecordReadOnlyDirectoryDoesNotPanic(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: permission bits do not block writes")
	}

	dir := t.TempDir()
	roDir := filepath.Join(dir, "readonly")
	require.NoError(t, os.Mkdir(roDir, 0755))
	require.NoError(t, os.Chmod(roDir, 0555))
	t.Cleanup(func() {
		// Restore write permission so t.TempDir()'s cleanup can remove it.
		require.NoError(t, os.Chmod(roDir, 0755))
	})

	logPath := filepath.Join(roDir, "session.log")

	require.NotPanics(t, func() {
		Record(logPath, Event{Command: []string{"spek", "plan"}})
	})

	_, err := os.Stat(logPath)
	require.True(t, os.IsNotExist(err))
}
