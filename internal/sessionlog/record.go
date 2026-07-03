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
