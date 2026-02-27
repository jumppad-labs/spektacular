# Adhoc JSON Protocol - Implementation Plan

## Overview
- **Specification**: `.spektacular/plans/5_adhoc_json_protocol/prompt.md`
- **Complexity**: Complex (0.7)
- **Dependencies**: `internal/runner`, `internal/config`, `github.com/spf13/cobra`
- **New Packages**: `internal/protocol`, `internal/adhoc`

## Current State Analysis

### What Exists
- **CLI framework** (`cmd/root.go`): Cobra-based with 4 commands (init, new, plan, run)
- **Runner** (`internal/runner/runner.go`): Claude CLI subprocess spawner with stream-JSON parsing, session management, question detection via `<!--QUESTION:-->` markers
- **Config** (`internal/config/config.go`): YAML-based config with `OutputConfig.Format` field (currently only `"markdown"`)
- **TUI** (`internal/tui/`): Bubble Tea interactive UI with TTY detection pattern in `cmd/plan.go:40`
- **Plan loop** (`internal/plan/plan.go:101-148`): Event-driven loop with question/answer cycling via `runner.RunClaude()` — reference pattern for adhoc execution

### What's Missing
- No `adhoc` command
- No JSONL protocol types or I/O
- No stdin reading for inbound events
- No provider-agnostic event normalization
- No `config set/get` subcommands
- No run lifecycle management (terminal event guarantee)

### Key Constraints
- Protocol frames on stdout only (stderr reserved for diagnostics)
- Exactly one terminal event per run
- Must not break existing CLI UX when `--output` is omitted
- Provider/tool agnostic protocol design
- Unknown fields ignored, unknown event types ignored with warning

## Implementation Strategy

**Approach**: Build the JSONL protocol layer as a clean abstraction in `internal/protocol/`, then create an `internal/adhoc/` package that bridges the existing `runner.RunClaude()` event stream into protocol events. The CLI command wires it together with output mode selection.

**Phasing**:
1. Protocol types and I/O (foundation)
2. Adhoc execution engine with event bridging
3. CLI command with `--output` flag
4. Config extension for persisted output mode
5. Tests at each layer

---

## Phase 1: Protocol Types

### New File: `internal/protocol/types.go`

Define the message envelope and all event types as concrete Go structs with JSON tags.

```go
package protocol

import "time"

const Version = "1"

// Envelope is the common wrapper for all protocol messages.
type Envelope struct {
    V       string    `json:"v"`
    ID      string    `json:"id"`
    TS      time.Time `json:"ts"`
    Type    string    `json:"type"`
    RunID   string    `json:"run_id"`
    Payload any       `json:"payload"`
}

// --- Inbound event payloads (stdin) ---

// RunStartPayload is the payload for "run.start" events.
type RunStartPayload struct {
    Prompt  string            `json:"prompt"`
    Config  map[string]any    `json:"config,omitempty"`
    Context map[string]string `json:"context,omitempty"`
}

// RunInputPayload is the payload for "run.input" events.
type RunInputPayload struct {
    QuestionID string `json:"question_id"`
    Value      string `json:"value"`
}

// RunCancelPayload is the payload for "run.cancel" events.
type RunCancelPayload struct {
    Reason string `json:"reason,omitempty"`
}

// --- Outbound event payloads (stdout) ---

// RunStartedPayload is the payload for "run.started" events.
type RunStartedPayload struct {
    Provider string         `json:"provider"`
    Model    string         `json:"model,omitempty"`
    Config   map[string]any `json:"config,omitempty"`
}

// RunProgressPayload is the payload for "run.progress" events.
type RunProgressPayload struct {
    Kind    string `json:"kind"` // "text", "tool_use", "tool_result", "status"
    Content string `json:"content,omitempty"`
    Tool    string `json:"tool,omitempty"`
}

// QuestionOption represents a single selectable option.
type QuestionOption struct {
    Label       string `json:"label"`
    Description string `json:"description,omitempty"`
}

// RunQuestionPayload is the payload for "run.question" events.
type RunQuestionPayload struct {
    QuestionID string           `json:"question_id"`
    Kind       string           `json:"kind"` // "confirm", "select", "text", "secret", "file_path"
    Text       string           `json:"text"`
    Options    []QuestionOption `json:"options,omitempty"`
    Default    string           `json:"default,omitempty"`
    Required   bool             `json:"required"`
    TimeoutS   int              `json:"timeout_s,omitempty"`
}

// RunArtifactPayload is the payload for "run.artifact" events.
type RunArtifactPayload struct {
    Name    string `json:"name"`
    Path    string `json:"path,omitempty"`
    Kind    string `json:"kind"` // "file", "plan", "diff", etc.
    Content string `json:"content,omitempty"`
}

// RunCompletedPayload is the payload for "run.completed" events.
type RunCompletedPayload struct {
    Summary   string   `json:"summary,omitempty"`
    Artifacts []string `json:"artifacts,omitempty"`
}

// RunFailedPayload is the payload for "run.failed" events.
type RunFailedPayload struct {
    Code    string `json:"code"` // "agent_error", "unsupported_version", "internal_error", "protocol_error"
    Message string `json:"message"`
}

// RunCancelledPayload is the payload for "run.cancelled" events.
type RunCancelledPayload struct {
    Reason string `json:"reason,omitempty"`
}
```

**Rationale**: Concrete structs over interfaces — the protocol is well-defined. The `Envelope.Payload` field uses `any` for flexibility during JSON marshal, but typed payloads are used throughout the code.

### New File: `internal/protocol/ids.go`

```go
package protocol

import (
    "crypto/rand"
    "fmt"
)

// NewMessageID generates a unique message ID with "msg_" prefix.
func NewMessageID() string {
    b := make([]byte, 8)
    _, _ = rand.Read(b)
    return fmt.Sprintf("msg_%x", b)
}

// NewRunID generates a unique run ID with "run_" prefix.
func NewRunID() string {
    b := make([]byte, 8)
    _, _ = rand.Read(b)
    return fmt.Sprintf("run_%x", b)
}
```

### Success Criteria — Phase 1
- [ ] All protocol types compile and have correct JSON tags
- [ ] `go vet ./internal/protocol/...` passes
- [ ] Types documented with godoc comments

---

## Phase 2: Protocol I/O

### New File: `internal/protocol/writer.go`

JSONL writer that emits one envelope per line to an `io.Writer`. Mutex-protected for goroutine safety.

```go
package protocol

import (
    "encoding/json"
    "io"
    "sync"
    "time"
)

// Writer emits JSONL protocol frames to an output stream.
type Writer struct {
    w     io.Writer
    mu    sync.Mutex
    runID string
}

// NewWriter creates a Writer for the given run.
func NewWriter(w io.Writer, runID string) *Writer {
    return &Writer{w: w, runID: runID}
}

// Emit writes a single protocol envelope as one JSON line.
func (pw *Writer) Emit(eventType string, payload any) error {
    env := Envelope{
        V:       Version,
        ID:      NewMessageID(),
        TS:      time.Now().UTC(),
        Type:    eventType,
        RunID:   pw.runID,
        Payload: payload,
    }
    pw.mu.Lock()
    defer pw.mu.Unlock()
    data, err := json.Marshal(env)
    if err != nil {
        return err
    }
    data = append(data, '\n')
    _, err = pw.w.Write(data)
    return err
}
```

### New File: `internal/protocol/reader.go`

JSONL reader that parses inbound envelopes from an `io.Reader` (stdin).

```go
package protocol

import (
    "bufio"
    "encoding/json"
    "fmt"
    "io"
)

// Reader parses JSONL protocol frames from an input stream.
type Reader struct {
    scanner *bufio.Scanner
}

// NewReader creates a Reader.
func NewReader(r io.Reader) *Reader {
    s := bufio.NewScanner(r)
    s.Buffer(make([]byte, 1024*1024), 1024*1024)
    return &Reader{scanner: s}
}

// Read returns the next envelope from the stream.
// Returns io.EOF when the stream is exhausted.
func (pr *Reader) Read() (Envelope, error) {
    if !pr.scanner.Scan() {
        if err := pr.scanner.Err(); err != nil {
            return Envelope{}, err
        }
        return Envelope{}, io.EOF
    }
    line := pr.scanner.Text()
    if line == "" {
        return pr.Read() // skip blank lines
    }
    var env Envelope
    if err := json.Unmarshal([]byte(line), &env); err != nil {
        return Envelope{}, fmt.Errorf("invalid JSON frame: %w", err)
    }
    return env, nil
}

// ReadPayload unmarshals the raw payload field into a typed struct.
func ReadPayload[T any](env Envelope) (T, error) {
    var result T
    raw, err := json.Marshal(env.Payload)
    if err != nil {
        return result, err
    }
    err = json.Unmarshal(raw, &result)
    return result, err
}
```

### Success Criteria — Phase 2
- [ ] Writer emits valid JSONL (one JSON object per line, newline terminated)
- [ ] Reader parses JSONL from stdin-like input
- [ ] Reader handles blank lines gracefully
- [ ] `ReadPayload` generic function correctly deserializes typed payloads
- [ ] Writer is goroutine-safe (mutex-protected)

---

## Phase 3: Adhoc Execution Engine

### New File: `internal/adhoc/adhoc.go`

The core execution engine that bridges the existing runner to the JSONL protocol.

**Key responsibilities:**
1. Read `run.start` from stdin, validate version
2. Emit `run.started`
3. Spawn Claude runner, bridge events to protocol
4. Normalize `runner.Question` → `run.question` schema
5. Listen for `run.input` (answers) and `run.cancel` on stdin concurrently
6. Guarantee exactly one terminal event via mutex-protected helpers

```go
package adhoc

import (
    "context"
    "fmt"
    "io"
    "sync"

    "github.com/nicholasjackson/spektacular/internal/config"
    "github.com/nicholasjackson/spektacular/internal/protocol"
    "github.com/nicholasjackson/spektacular/internal/runner"
)

// Engine manages adhoc execution with the JSONL protocol.
type Engine struct {
    cfg    config.Config
    cwd    string
    writer *protocol.Writer
    reader *protocol.Reader
    cancel context.CancelFunc
    mu     sync.Mutex
    done   bool
}

// New creates an Engine.
func New(cfg config.Config, cwd string, stdin io.Reader, stdout io.Writer) *Engine {
    runID := protocol.NewRunID()
    return &Engine{
        cfg:    cfg,
        cwd:    cwd,
        writer: protocol.NewWriter(stdout, runID),
        reader: protocol.NewReader(stdin),
    }
}
```

**Run method lifecycle:**

```go
// Run executes the JSONL protocol lifecycle:
// 1. Wait for run.start on stdin
// 2. Validate version
// 3. Emit run.started
// 4. Execute agent, emitting run.progress and run.question events
// 5. Emit exactly one terminal event (run.completed, run.failed, or run.cancelled)
func (e *Engine) Run() error {
    env, err := e.reader.Read()
    if err != nil {
        return e.emitFailed("internal_error", fmt.Sprintf("reading stdin: %s", err))
    }
    if env.V != protocol.Version {
        return e.emitFailed("unsupported_version",
            fmt.Sprintf("expected protocol version %s, got %s", protocol.Version, env.V))
    }
    if env.Type != "run.start" {
        return e.emitFailed("protocol_error",
            fmt.Sprintf("expected run.start, got %s", env.Type))
    }
    startPayload, err := protocol.ReadPayload[protocol.RunStartPayload](env)
    if err != nil {
        return e.emitFailed("protocol_error",
            fmt.Sprintf("invalid run.start payload: %s", err))
    }

    e.writer.Emit("run.started", protocol.RunStartedPayload{
        Provider: "claude",
        Model:    e.cfg.Models.Default,
    })

    ctx, cancel := context.WithCancel(context.Background())
    e.cancel = cancel
    defer cancel()

    inputCh := make(chan protocol.RunInputPayload, 8)
    go e.listenStdin(ctx, inputCh)

    return e.executeAgent(ctx, startPayload.Prompt, inputCh)
}
```

**Event bridging — translates runner.ClaudeEvent → protocol events:**

```go
func (e *Engine) executeAgent(ctx context.Context, prompt string, inputCh <-chan protocol.RunInputPayload) error {
    events, errc := runner.RunClaude(runner.RunOptions{
        Prompt:  prompt,
        Config:  e.cfg,
        CWD:     e.cwd,
        Command: "adhoc",
    })

    questionCounter := 0
    sessionID := ""

    for {
        select {
        case <-ctx.Done():
            return e.emitCancelled("user requested cancellation")

        case event, ok := <-events:
            if !ok {
                if err := <-errc; err != nil {
                    return e.emitFailed("agent_error", err.Error())
                }
                return e.emitFailed("agent_error", "agent completed without result")
            }

            if id := event.SessionID(); id != "" {
                sessionID = id
            }

            // Bridge text → run.progress
            if text := event.TextContent(); text != "" {
                e.writer.Emit("run.progress", protocol.RunProgressPayload{
                    Kind:    "text",
                    Content: text,
                })

                // Detect and normalize questions
                if questions := runner.DetectQuestions(text); len(questions) > 0 {
                    for _, q := range questions {
                        questionCounter++
                        qID := fmt.Sprintf("q_%d", questionCounter)
                        options := make([]protocol.QuestionOption, len(q.Options))
                        for i, opt := range q.Options {
                            label, _ := opt["label"].(string)
                            desc, _ := opt["description"].(string)
                            options[i] = protocol.QuestionOption{
                                Label: label, Description: desc,
                            }
                        }
                        e.writer.Emit("run.question", protocol.RunQuestionPayload{
                            QuestionID: qID,
                            Kind:       "select",
                            Text:       q.Question,
                            Options:    options,
                            Required:   true,
                        })
                    }
                    // Wait for input answer
                    select {
                    case input := <-inputCh:
                        events, errc = runner.RunClaude(runner.RunOptions{
                            Prompt:    input.Value,
                            Config:    e.cfg,
                            SessionID: sessionID,
                            CWD:       e.cwd,
                            Command:   "adhoc",
                        })
                    case <-ctx.Done():
                        return e.emitCancelled("cancelled while waiting for input")
                    }
                }
            }

            // Bridge tool uses → run.progress
            for _, tool := range event.ToolUses() {
                name, _ := tool["name"].(string)
                e.writer.Emit("run.progress", protocol.RunProgressPayload{
                    Kind: "tool_use",
                    Tool: name,
                })
            }

            // Bridge result → terminal event
            if event.IsResult() {
                if event.IsError() {
                    return e.emitFailed("agent_error", event.ResultText())
                }
                return e.emitCompleted(event.ResultText())
            }
        }
    }
}
```

**Stdin listener for `run.input` and `run.cancel`:**

```go
func (e *Engine) listenStdin(ctx context.Context, inputCh chan<- protocol.RunInputPayload) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
        }
        env, err := e.reader.Read()
        if err != nil {
            return
        }
        switch env.Type {
        case "run.input":
            payload, err := protocol.ReadPayload[protocol.RunInputPayload](env)
            if err != nil {
                continue
            }
            inputCh <- payload
        case "run.cancel":
            if e.cancel != nil {
                e.cancel()
            }
            return
        }
        // Unknown types: silently ignore (spec compatibility rule)
    }
}
```

**Terminal event helpers — mutex-protected, emit-once guarantee:**

```go
func (e *Engine) emitCompleted(summary string) error {
    e.mu.Lock()
    defer e.mu.Unlock()
    if e.done { return nil }
    e.done = true
    return e.writer.Emit("run.completed", protocol.RunCompletedPayload{Summary: summary})
}

func (e *Engine) emitFailed(code, message string) error {
    e.mu.Lock()
    defer e.mu.Unlock()
    if e.done { return nil }
    e.done = true
    return e.writer.Emit("run.failed", protocol.RunFailedPayload{Code: code, Message: message})
}

func (e *Engine) emitCancelled(reason string) error {
    e.mu.Lock()
    defer e.mu.Unlock()
    if e.done { return nil }
    e.done = true
    return e.writer.Emit("run.cancelled", protocol.RunCancelledPayload{Reason: reason})
}
```

### Success Criteria — Phase 3
- [ ] Engine reads `run.start` from stdin, emits `run.started`
- [ ] Runner events bridge to `run.progress` events
- [ ] Questions from Claude normalize to `run.question` schema
- [ ] `run.input` answers resume the agent session
- [ ] `run.cancel` triggers clean cancellation
- [ ] Exactly one terminal event emitted per run
- [ ] Version mismatch emits `run.failed` with code `unsupported_version`

---

## Phase 4: CLI Command

### New File: `cmd/adhoc.go`

```go
package cmd

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/nicholasjackson/spektacular/internal/adhoc"
    "github.com/nicholasjackson/spektacular/internal/config"
    "github.com/nicholasjackson/spektacular/internal/runner"
    "github.com/spf13/cobra"
)

var adhocOutput string

var adhocCmd = &cobra.Command{
    Use:   "adhoc [prompt]",
    Short: "Run an adhoc task with optional JSON protocol output",
    Long: `Run a one-off task through Spektacular.

In CLI mode (default), provide a prompt as an argument:
  spektacular adhoc "fix the login bug"

In JSON mode, use JSONL protocol over stdin/stdout:
  spektacular adhoc --output json`,
    Args: cobra.MaximumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        cwd, err := os.Getwd()
        if err != nil {
            return fmt.Errorf("getting working directory: %w", err)
        }

        configPath := filepath.Join(cwd, ".spektacular", "config.yaml")
        var cfg config.Config
        if _, err := os.Stat(configPath); err == nil {
            cfg, err = config.FromYAMLFile(configPath)
            if err != nil {
                return fmt.Errorf("loading config: %w", err)
            }
        } else {
            cfg = config.NewDefault()
        }

        outputMode := adhocOutput
        if outputMode == "" {
            outputMode = cfg.Output.AdhocOutput
        }
        if outputMode == "" {
            outputMode = "cli"
        }

        switch outputMode {
        case "json":
            engine := adhoc.New(cfg, cwd, os.Stdin, os.Stdout)
            if err := engine.Run(); err != nil {
                os.Exit(1)
            }
            return nil
        case "cli":
            if len(args) == 0 {
                return fmt.Errorf("prompt argument required in CLI mode")
            }
            return runAdhocCLI(cfg, cwd, args[0])
        default:
            return fmt.Errorf("unknown output mode: %s (use 'cli' or 'json')", outputMode)
        }
    },
}

func init() {
    adhocCmd.Flags().StringVar(&adhocOutput, "output", "",
        "Output mode: 'cli' for terminal, 'json' for JSONL protocol")
}

func runAdhocCLI(cfg config.Config, cwd, prompt string) error {
    events, errc := runner.RunClaude(runner.RunOptions{
        Prompt:  prompt,
        Config:  cfg,
        CWD:     cwd,
        Command: "adhoc",
    })
    for event := range events {
        if text := event.TextContent(); text != "" {
            fmt.Print(text)
        }
        if event.IsResult() {
            if event.IsError() {
                return fmt.Errorf("agent error: %s", event.ResultText())
            }
            fmt.Println(event.ResultText())
        }
    }
    if err := <-errc; err != nil {
        return fmt.Errorf("runner error: %w", err)
    }
    return nil
}
```

### Modified File: `cmd/root.go:25-30`

Add `adhocCmd` to command registration:

```go
func init() {
    rootCmd.AddCommand(initCmd)
    rootCmd.AddCommand(newCmd)
    rootCmd.AddCommand(planCmd)
    rootCmd.AddCommand(runCmd)
    rootCmd.AddCommand(adhocCmd)
}
```

### Success Criteria — Phase 4
- [ ] `spektacular adhoc "fix bug"` works in CLI mode (text output to stdout)
- [ ] `spektacular adhoc --output json` reads JSONL from stdin, emits JSONL on stdout
- [ ] `--output` flag overrides config default
- [ ] Missing prompt in CLI mode returns clear error
- [ ] Unknown output mode returns clear error

---

## Phase 5: Config Extension

### Modified File: `internal/config/config.go:44-48`

Add `AdhocOutput` to `OutputConfig`:

```go
type OutputConfig struct {
    Format          string `yaml:"format"`
    IncludeMetadata bool   `yaml:"include_metadata"`
    AdhocOutput     string `yaml:"adhoc_output"`
}
```

### Modified File: `internal/config/config.go:96-99`

Update `NewDefault()` to set default:

```go
Output: OutputConfig{
    Format:          "markdown",
    IncludeMetadata: true,
    AdhocOutput:     "cli",
},
```

### Success Criteria — Phase 5
- [ ] Config YAML supports `output.adhoc_output: json`
- [ ] `--output` flag overrides config value
- [ ] Default remains `cli` when nothing is configured

---

## Phase 6: Comprehensive Tests

### Test File: `internal/protocol/types_test.go`
- JSON marshal/unmarshal round-trip for all envelope + payload types
- Version constant correctness
- ID generation uniqueness

### Test File: `internal/protocol/writer_test.go`
- Single event writes valid JSONL line
- Multiple events write separate lines
- Concurrent writes don't corrupt output
- All required fields present

### Test File: `internal/protocol/reader_test.go`
- Parse valid JSONL frames
- Skip blank lines
- Handle malformed JSON gracefully
- EOF handling
- `ReadPayload` generic deserialization

### Test File: `internal/adhoc/adhoc_test.go`
- Happy path: `run.start` -> `run.started` -> `run.completed`
- Version mismatch -> `run.failed` with `unsupported_version`
- Invalid first event -> `run.failed` with `protocol_error`
- Question flow: `run.question` -> `run.input` -> resume
- Cancellation: `run.cancel` -> `run.cancelled`
- Exactly one terminal event guarantee
- Error from runner -> `run.failed`

### Test File: `cmd/adhoc_test.go`
- CLI mode with prompt argument
- CLI mode without prompt (error)
- Flag parsing for `--output`
- Unknown output mode (error)

### Success Criteria — Phase 6
- [ ] `go test ./internal/protocol/...` passes
- [ ] `go test ./internal/adhoc/...` passes
- [ ] `go test ./cmd/...` passes
- [ ] `go vet ./...` clean

---

## Exit Codes

| Scenario | Exit Code |
|----------|-----------|
| `run.completed` | 0 |
| `run.cancelled` | 0 |
| `run.failed` (any code) | 1 |
| CLI mode success | 0 |
| CLI mode error | 1 |

---

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/protocol/types.go` | Message envelope and all event payload types |
| `internal/protocol/ids.go` | Message ID and Run ID generation |
| `internal/protocol/writer.go` | JSONL stdout writer |
| `internal/protocol/reader.go` | JSONL stdin reader |
| `internal/protocol/types_test.go` | Protocol type tests |
| `internal/protocol/writer_test.go` | Writer tests |
| `internal/protocol/reader_test.go` | Reader tests |
| `internal/adhoc/adhoc.go` | Adhoc execution engine with event bridging |
| `internal/adhoc/adhoc_test.go` | Engine tests |
| `cmd/adhoc.go` | CLI command definition + CLI mode helper |
| `cmd/adhoc_test.go` | Command tests |

### Modified Files
| File | Change |
|------|--------|
| `cmd/root.go:29` | Add `rootCmd.AddCommand(adhocCmd)` |
| `internal/config/config.go:45-48` | Add `AdhocOutput` field to `OutputConfig` |
| `internal/config/config.go:97` | Add default `AdhocOutput: "cli"` |

---

## Verification Commands

### Automated
```bash
go build ./...
go vet ./...
go test ./internal/protocol/... -v
go test ./internal/adhoc/... -v
go test ./cmd/... -v
go test ./... -count=1
```

### Manual Integration Test
```bash
# CLI mode
./spektacular adhoc "list the files in the current directory"

# JSON mode — happy path
echo '{"v":"1","id":"msg_1","ts":"2026-02-27T00:00:00Z","type":"run.start","run_id":"run_1","payload":{"prompt":"list files"}}' \
  | ./spektacular adhoc --output json

# JSON mode — version mismatch
echo '{"v":"99","id":"msg_1","ts":"2026-02-27T00:00:00Z","type":"run.start","run_id":"run_1","payload":{"prompt":"test"}}' \
  | ./spektacular adhoc --output json
# Expected: run.failed with code "unsupported_version"
```

---

## References
- Existing runner: `internal/runner/runner.go:146-229`
- Existing plan loop: `internal/plan/plan.go:101-148`
- TTY detection pattern: `cmd/plan.go:40`
- Config structure: `internal/config/config.go:64-72`
- Command registration: `cmd/root.go:25-30`
- Question detection: `internal/runner/runner.go:97-123`
- Question format: `runner.Question` struct at `internal/runner/runner.go:91-95`
