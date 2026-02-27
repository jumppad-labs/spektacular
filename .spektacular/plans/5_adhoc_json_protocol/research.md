# Adhoc JSON Protocol - Research Notes

## Specification Analysis

### Original Requirements
- Add `spektacular adhoc --output <cli|json>` output mode selection
- Support optional persisted default output mode via config
- In JSON mode, use JSONL over stdin/stdout for bidirectional messaging
- Define common message envelope with versioning and correlation IDs
- Define standardized inbound types: `run.start`, `run.input`, `run.cancel`
- Define standardized outbound types: `run.started`, `run.progress`, `run.question`, `run.artifact`, `run.completed`, `run.failed`, `run.cancelled`
- Define tool-agnostic question schema for interactive prompts
- Ensure exactly one terminal event per run
- Define compatibility behavior for unknown fields/types and version mismatch
- Define safe handling for secrets and diagnostics

### Implicit Requirements
- The JSON mode must handle concurrent stdin reading (for `run.input` and `run.cancel`) while the agent is executing
- Session management must be bridged (Claude's session IDs are internal to the protocol, not exposed)
- The protocol must normalize Claude's `<!--QUESTION:-->` marker format into the generic question schema
- CLI mode must still function when `--output` is omitted (backward compatibility)

### Constraints Identified
- stdout is exclusively for protocol frames (no diagnostic output in JSON mode)
- stderr for diagnostics only
- JSONL format: one JSON object per line, UTF-8
- No multi-run multiplexing (single run lifecycle per process)
- Provider-agnostic protocol surface (no Claude-specific fields in the protocol)

## Research Process

### Sub-agents Spawned
1. **Codebase Explorer** — Full project structure inventory
2. **CLI Pattern Analyzer** — Cobra command patterns, flag handling, TTY detection
3. **Internal Package Researcher** — Deep dive into all internal packages, types, interfaces

### Files Examined

| File | Purpose | Key Findings |
|------|---------|--------------|
| `main.go` | Entry point | Delegates to `cmd.Execute()` |
| `cmd/root.go` | Command registration | Cobra-based, 4 commands registered in `init()` |
| `cmd/init.go` | Init command | Flag pattern: `BoolVar` in `init()`, working dir via `os.Getwd()` |
| `cmd/new.go` | New command | Flag pattern: `StringVar`, positional args via `cobra.ExactArgs(1)` |
| `cmd/plan.go` | Plan command | **Critical pattern**: TTY detection at line 40, config loading at 28-37, dual mode (TUI vs streaming) |
| `cmd/run.go` | Run command | Stub only, not implemented |
| `internal/config/config.go` | Configuration | `OutputConfig` at line 44 has `Format` and `IncludeMetadata`. `AgentConfig` at line 56 has command/args/tools |
| `internal/runner/runner.go` | Claude subprocess | `ClaudeEvent` type at line 22, `RunClaude()` at line 148, `DetectQuestions()` at line 123, `BuildPrompt()` at line 126 |
| `internal/plan/plan.go` | Plan orchestration | `RunPlan()` at line 74 — event loop with question/answer cycling. Key reference for adhoc execution pattern |
| `internal/tui/tui.go` | Bubble Tea UI | `agentEventMsg` at line 26, model state at line 39, `handleAgentEvent()` at line 261 |
| `internal/defaults/defaults.go` | Embedded files | `ReadFile()` and `MustReadFile()` for bundled assets |
| `internal/project/init.go` | Project init | Creates `.spektacular/` directory tree |
| `internal/spec/spec.go` | Spec creation | Template-based spec file generation |
| `Makefile` | Build system | `go build`, `go test ./...`, cross-compilation targets |

### Patterns Discovered

#### 1. Command Registration Pattern (`cmd/root.go:25-30`)
All commands are registered in `root.go`'s `init()` function. New commands follow the same pattern.

#### 2. Config Loading Pattern (`cmd/plan.go:28-37`)
```go
configPath := filepath.Join(cwd, ".spektacular", "config.yaml")
var cfg config.Config
if _, err := os.Stat(configPath); err == nil {
    cfg, err = config.FromYAMLFile(configPath)
} else {
    cfg = config.NewDefault()
}
```
This exact pattern should be reused in `cmd/adhoc.go`.

#### 3. Event Processing Pattern (`internal/plan/plan.go:101-148`)
The plan generation loop is the closest existing pattern to what the adhoc engine needs:
- Spawn `runner.RunClaude()` → get event channel
- Iterate events, accumulate text and questions
- If questions found, get answer and re-spawn with `--resume` session ID
- On result event, write output and return

#### 4. Question Detection (`internal/runner/runner.go:97-123`)
Questions are embedded in Claude's text output as HTML comments:
```
<!--QUESTION:{"questions":[{"question":"...","header":"...","options":[...]}]}-->
```
The `runner.Question` struct has `Question`, `Header`, and `Options []map[string]any` fields.

#### 5. Session Resumption (`internal/runner/runner.go:174-176`)
When a session ID is provided, the runner adds `--resume <sessionID>` to the claude CLI command. This enables question/answer cycling.

#### 6. TTY Detection (`cmd/plan.go:40`)
```go
if term.IsTerminal(int(os.Stdout.Fd())) { ... }
```
Used to decide between interactive TUI and streaming output. The adhoc command uses `--output` flag instead.

## Key Findings

### Architecture Insights
- The codebase follows a clean layered architecture: `cmd/` → `internal/` (packages don't cross-depend)
- The runner package is the only component that interacts with the Claude CLI subprocess
- Event processing is channel-based: `RunClaude()` returns `<-chan ClaudeEvent` and `<-chan error`
- There are no interfaces for provider abstraction — the runner directly spawns the `claude` command

### Existing Implementations
- `plan.RunPlan()` is the closest analog to what `adhoc.Engine.Run()` needs to do
- The TUI's `handleAgentEvent()` shows how to process different event types (text, tool_use, result)
- Debug logging pattern: conditional `openDebugLog()` based on `cfg.Debug.Enabled`

### Reusable Components
- `runner.RunClaude()` — direct reuse for spawning the Claude subprocess
- `runner.DetectQuestions()` — direct reuse for question detection in text
- `runner.ClaudeEvent` methods — `TextContent()`, `ToolUses()`, `IsResult()`, `IsError()`, `ResultText()`, `SessionID()`
- `config.FromYAMLFile()` / `config.NewDefault()` — config loading

### Testing Infrastructure
- All tests use `testing` + `github.com/stretchr/testify` (require package)
- `t.TempDir()` for filesystem tests
- `t.Setenv()` for environment variable mocking
- No mock framework used — tests create real config/files
- Test files are co-located with source: `*_test.go`

## Design Decisions

### Decision: Concrete protocol types vs interface-based
**Choice**: Concrete structs with JSON tags
**Rationale**: The protocol is well-specified with fixed event types. Interfaces would add unnecessary abstraction. `Envelope.Payload` uses `any` for marshal flexibility, but all code paths use typed payloads.

### Decision: Separate `internal/protocol/` package
**Choice**: New package rather than extending `runner`
**Rationale**: The protocol is tool-agnostic by design. Keeping it separate from the Claude-specific runner enforces this boundary. Other providers can reuse the protocol package without importing Claude internals.

### Decision: Mutex-protected terminal events
**Choice**: `sync.Mutex` + `done` flag on terminal event helpers
**Rationale**: The spec requires exactly one terminal event per run. With concurrent stdin listening (for `run.cancel`) and agent execution, a race between `emitFailed()` and `emitCancelled()` is possible. The mutex ensures only the first terminal event wins.

### Decision: Generic `ReadPayload[T]` function
**Choice**: Go generics for payload deserialization
**Rationale**: Avoids repetitive unmarshal boilerplate for each payload type. The caller specifies the expected type, and the function handles the `any` → typed conversion via JSON round-trip.

### Decision: Engine owns the run ID
**Choice**: Run ID generated in `adhoc.New()`, not from `run.start` envelope
**Rationale**: The engine creates the run context. The inbound `run.start` may carry a run_id for correlation, but the engine's own run_id is used for all outbound events, ensuring consistency.

### Decision: CLI mode uses runner directly
**Choice**: CLI mode bypasses the protocol engine entirely
**Rationale**: In CLI mode, there's no JSONL protocol — just text streaming. Wrapping it in the engine would add unnecessary complexity. The `runAdhocCLI()` helper calls `runner.RunClaude()` directly, matching the existing plan command's non-TTY path.

## Open Questions (Resolved)

All questions resolved through specification analysis and codebase research. No outstanding blockers.

## Code Examples & Patterns

### Existing Event Loop Pattern (reference for adhoc engine)
```go
// File: internal/plan/plan.go:101-148
for {
    var questionsFound []runner.Question
    var finalResult string

    events, errc := runner.RunClaude(runner.RunOptions{
        Prompt:    currentPrompt,
        Config:    cfg,
        SessionID: sessionID,
        CWD:       projectPath,
        Command:   "plan",
    })

    for event := range events {
        if id := event.SessionID(); id != "" {
            sessionID = id
        }
        if text := event.TextContent(); text != "" {
            if onText != nil {
                onText(text)
            }
            questionsFound = append(questionsFound, runner.DetectQuestions(text)...)
        }
        if event.IsResult() {
            if event.IsError() {
                return "", fmt.Errorf("agent error: %s", event.ResultText())
            }
            finalResult = event.ResultText()
        }
    }

    if err := <-errc; err != nil {
        return "", fmt.Errorf("runner error: %w", err)
    }

    if len(questionsFound) > 0 && onQuestion != nil {
        answer := onQuestion(questionsFound)
        currentPrompt = answer
        continue
    }

    // ... write result
}
```

### Existing Config Loading Pattern
```go
// File: cmd/plan.go:28-37
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
```

### Existing Question Structure
```go
// File: internal/runner/runner.go:91-95
type Question struct {
    Question string
    Header   string
    Options  []map[string]any
}
```

Normalized to protocol format:
```go
// Protocol question with explicit kinds and structured options
RunQuestionPayload{
    QuestionID: "q_1",
    Kind:       "select",
    Text:       q.Question,
    Options:    []QuestionOption{{Label: "...", Description: "..."}},
    Required:   true,
}
```
