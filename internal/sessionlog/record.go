// Package sessionlog appends a JSON Lines record of each completed CLI
// invocation to a local debug log, for reconstructing an agent session
// after the fact. It knows nothing about workflows, commands, or
// configuration — it only turns one already-assembled Event into one line
// on disk.
package sessionlog

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Event is one JSON Lines record, written once per completed CLI invocation.
type Event struct {
	Seq         int64          `json:"seq"`
	Timestamp   time.Time      `json:"timestamp"`
	SessionID   string         `json:"session_id"`
	Command     []string       `json:"command"`
	DurationMS  int64          `json:"duration_ms"`
	ExitCode    int            `json:"exit_code"`
	Response    string         `json:"response"`
	StateBefore *StateSnapshot `json:"state_before"`
	StateAfter  *StateSnapshot `json:"state_after"`
	Advanced    bool           `json:"advanced"`
}

// noActiveWorkflowSessionID is the fixed session id used when a command runs
// with no workflow state on disk yet (e.g. an ad hoc knowledge search before
// any spec/plan/implement workflow has ever run in the project).
const noActiveWorkflowSessionID = "no-active-workflow"

// SessionID derives the session a command belongs to directly from the
// workflow state store's current (Kind, Name) pair — a pure function, not a
// stored value, so no session registry, lock, or marker file exists
// anywhere in this design. The same state always produces the same id.
func SessionID(state *StateSnapshot) string {
	if state == nil || (state.Kind == "" && state.Name == "") {
		return noActiveWorkflowSessionID
	}
	return state.Kind + ":" + state.Name
}

// StateSnapshot is a small, read-only view of the workflow engine's
// persisted state, taken immediately before and immediately after a
// command runs. A nil *StateSnapshot means no workflow state existed on
// disk yet at that point.
type StateSnapshot struct {
	Kind           string   `json:"kind"`
	Name           string   `json:"name"`
	CurrentStep    string   `json:"current_step"`
	CompletedSteps []string `json:"completed_steps"`
}

// logFileTimestampFormat stamps each minted filename with a fixed-width,
// zero-padded moment so filenames for the same session sort chronologically
// as plain strings — the most recently started one always sorts last.
// Nanosecond precision (rather than just to the second) keeps two rapid
// restarts of the same named workflow — e.g. `new --force` run again
// moments later — from minting the same filename and silently colliding.
const logFileTimestampFormat = "20060102T150405.000000000Z"

// LogFilePath resolves which file a session's record belongs in. When start
// is true — a `<kind> new` command just began a genuinely fresh workflow —
// it always mints a brand-new file stamped with now, agent, and sessionID,
// even if a file for the same sessionID already exists (e.g. a `--force`
// restart of the same named workflow): each start is its own session, never
// appended to a predecessor's log. Otherwise it reuses the most recently
// started file already on disk for sessionID — what a resumed workflow, or
// any other command sharing that session, should keep appending to — or
// mints a fresh one if none exists yet (the session's first command ever
// recorded, or one that predates this scheme). LogFilePath only resolves
// the path; it never creates the file or its parent directory itself.
func LogFilePath(dir, agent, sessionID string, start bool, now time.Time) string {
	if !start {
		if existing := latestLogFile(dir, sessionID); existing != "" {
			return filepath.Join(dir, existing)
		}
	}
	return filepath.Join(dir, newLogFileName(agent, sessionID, now))
}

// newLogFileName mints a filename stamped with when the session started,
// which agent the project is configured for, and the session id (workflow
// kind + name, or the no-active-workflow sentinel), so a plain directory
// listing identifies every session at a glance without opening a file.
func newLogFileName(agent, sessionID string, now time.Time) string {
	if agent == "" {
		agent = "unknown"
	}
	return now.UTC().Format(logFileTimestampFormat) + "_" + agent + "_" + sanitizeSessionID(sessionID) + ".jsonl"
}

// sessionIDFileSuffix is the filename suffix shared by every log file that
// belongs to sessionID, regardless of which agent or moment started it.
func sessionIDFileSuffix(sessionID string) string {
	return "_" + sanitizeSessionID(sessionID) + ".jsonl"
}

// sanitizeSessionID replaces characters a session id may contain (the ":"
// joining kind and name) that are unsafe or awkward in a filename.
func sanitizeSessionID(sessionID string) string {
	return strings.NewReplacer(":", "_", "/", "_", "\\", "_").Replace(sessionID)
}

// latestLogFile returns the filename (not full path) of the most recently
// started log file belonging to sessionID in dir, or "" if none exists or
// dir can't be read. Filenames sort chronologically as plain strings
// because logFileTimestampFormat is fixed-width and zero-padded.
func latestLogFile(dir, sessionID string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	suffix := sessionIDFileSuffix(sessionID)
	var latest string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if name := e.Name(); strings.HasSuffix(name, suffix) && name > latest {
			latest = name
		}
	}
	return latest
}

// Record appends ev as one JSON line to the file at logPath, creating the
// file and its parent directory if needed. It never returns an error and
// never panics outward — any failure while building or writing the entry is
// contained and dropped inside this function, so a recording failure can
// never surface as a command failure.
func Record(logPath string, ev Event) {
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return
	}

	seq, err := countLines(logPath)
	if err != nil {
		return
	}
	ev.Seq = seq

	data, err := json.Marshal(ev)
	if err != nil {
		return
	}

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	data = append(data, '\n')
	f.Write(data)
}

// countLines returns the number of newline-terminated lines already present
// at path, used to derive each new Event's strictly increasing Seq across
// process boundaries. A missing file counts as zero existing lines.
func countLines(path string) (int64, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return int64(bytes.Count(data, []byte("\n"))), nil
}
