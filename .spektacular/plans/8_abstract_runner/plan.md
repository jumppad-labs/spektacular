# Abstract Runner - Implementation Plan

## Overview
- **Specification**: `.spektacular/specs/8_abstract_runner.md`
- **Complexity**: Medium
- **Estimated Effort**: ~3-4 hours
- **Dependencies**: None (internal refactoring)

## Current State Analysis

### What Exists Now
The entire system is hardcoded to Claude CLI. The `internal/runner` package directly spawns a `claude` subprocess, parses its stream-JSON output, and exposes Claude-specific types (`ClaudeEvent`, `RunClaude`, `RunOptions`). Both the `plan` and `implement` packages, plus the TUI, call `runner.RunClaude()` directly.

### What's Missing
- **No common interface** — there is no abstraction that defines what a "runner" does
- **No pluggability** — adding another tool (e.g., Bob, OpenAI, Aider) would require forking all call sites
- **Claude-specific naming** — types like `ClaudeEvent` and functions like `RunClaude` leak implementation details

### Key Constraints
- The current event-streaming model (channels of events + error channel) is a good pattern and should be preserved in the interface
- The `Question` type and `DetectQuestions()` are transport-agnostic and should remain in the shared runner package
- The `Workflow` struct in the TUI already provides a partial abstraction — it should work seamlessly with the new interface
- Configuration must support selecting which runner to use

### Integration Points
- `internal/plan/plan.go:114` — calls `runner.RunClaude()`
- `internal/implement/implement.go:95` — calls `runner.RunClaude()`
- `internal/tui/tui.go:132` — calls `runner.RunClaude()` via `startAgentCmd`
- `internal/tui/tui.go:147` — calls `runner.RunClaude()` via `resumeAgentCmd`
- `internal/tui/tui.go:687` — calls `runner.RunClaude()` via `implementStartCmd`
- `internal/config/config.go:57-62` — `AgentConfig` is Claude-specific

## Implementation Strategy

The approach is a **two-phase refactoring**:

1. **Phase 1**: Define the runner interface and shared types in `internal/runner/`, then extract the Claude implementation into `internal/runner/claude/`
2. **Phase 2**: Update all call sites (`plan`, `implement`, `tui`) to use the interface, and add a factory function for runner creation from config

This keeps the refactoring contained and ensures all existing tests continue to pass at each step.

## Phase 1: Define Interface & Extract Claude Runner

### 1.1 — Define the Runner interface and rename shared types

**File**: `internal/runner/runner.go`

The current `runner.go` mixes three concerns: shared types, shared utilities, and Claude-specific execution. We'll keep the shared types and interface here, and move the Claude implementation out.

**Current shared types to keep in `internal/runner/runner.go`**:
- `Event` (renamed from `ClaudeEvent`) — the generic event type
- `Question` — structured question type
- `RunOptions` — options for running any agent
- `DetectQuestions()` — question detection (transport-agnostic)
- `BuildPrompt()` / `BuildPromptWithHeader()` — prompt assembly (transport-agnostic)

**New interface to add**:

```go
// Runner is the interface that all agent backends must implement.
// It spawns an agent subprocess (or API call) and returns a channel of events.
type Runner interface {
    // Run starts the agent with the given options and returns a channel of
    // events and an error channel. The caller must drain both channels;
    // the event channel is closed when the agent finishes.
    Run(opts RunOptions) (<-chan Event, <-chan error)
}
```

**Proposed new `internal/runner/runner.go`**:

```go
// Package runner defines the Runner interface and shared types for agent backends.
package runner

import (
    "encoding/json"
    "fmt"
    "regexp"
    "strings"

    "github.com/jumppad-labs/spektacular/internal/config"
)

var questionPattern = regexp.MustCompile(`<!--QUESTION:([\s\S]*?)-->`)

// Runner is the interface that all agent backends must implement.
type Runner interface {
    // Run starts the agent with the given options and returns a channel of
    // events and an error channel. The caller must drain both channels;
    // the event channel is closed when the agent finishes.
    Run(opts RunOptions) (<-chan Event, <-chan error)
}

// Event is a single parsed event from an agent's output stream.
type Event struct {
    Type string
    Data map[string]any
}

// SessionID returns the session_id field if present.
func (e Event) SessionID() string {
    v, _ := e.Data["session_id"].(string)
    return v
}

// IsResult reports whether this is a terminal result event.
func (e Event) IsResult() bool { return e.Type == "result" }

// IsError reports whether this is an error result.
func (e Event) IsError() bool {
    if !e.IsResult() {
        return false
    }
    v, _ := e.Data["is_error"].(bool)
    return v
}

// ResultText returns the result text from a result event, or empty string.
func (e Event) ResultText() string {
    if !e.IsResult() {
        return ""
    }
    v, _ := e.Data["result"].(string)
    return v
}

// TextContent extracts concatenated text blocks from an assistant event.
func (e Event) TextContent() string {
    if e.Type != "assistant" {
        return ""
    }
    msg, _ := e.Data["message"].(map[string]any)
    content, _ := msg["content"].([]any)
    var texts []string
    for _, item := range content {
        block, _ := item.(map[string]any)
        if block["type"] == "text" {
            if t, ok := block["text"].(string); ok {
                texts = append(texts, t)
            }
        }
    }
    return strings.Join(texts, "\n")
}

// ToolUses extracts tool_use blocks from an assistant event.
func (e Event) ToolUses() []map[string]any {
    if e.Type != "assistant" {
        return nil
    }
    msg, _ := e.Data["message"].(map[string]any)
    content, _ := msg["content"].([]any)
    var tools []map[string]any
    for _, item := range content {
        block, _ := item.(map[string]any)
        if block["type"] == "tool_use" {
            tools = append(tools, block)
        }
    }
    return tools
}

// Question is a structured question detected in agent output.
type Question struct {
    Question string
    Header   string
    Options  []map[string]any
}

// detectQuestions finds <!--QUESTION:{...}--> markers in text and returns parsed questions.
func detectQuestions(text string) []Question {
    // ... (unchanged)
}

// DetectQuestions is the exported wrapper used by other packages.
func DetectQuestions(text string) []Question { return detectQuestions(text) }

// RunOptions holds parameters for running an agent.
type RunOptions struct {
    Prompt       string
    SystemPrompt string        // passed to the agent for specialization
    Config       config.Config
    SessionID    string
    CWD          string
    Command      string        // used only for debug log filename
}

// BuildPrompt assembles the user prompt: knowledge hint + spec content.
func BuildPrompt(specContent string) string {
    return BuildPromptWithHeader(specContent, "Specification to Plan")
}

// BuildPromptWithHeader assembles the user prompt with a custom content section header.
func BuildPromptWithHeader(content string, header string) string {
    var b strings.Builder
    b.WriteString("Additional project knowledge, architectural context, and past learnings can be found in `.spektacular/knowledge/`. Use your available tools to explore this directory as needed.\n\n")
    fmt.Fprintf(&b, "---\n\n# %s\n\n%s", header, content)
    return b.String()
}
```

**Key decisions**:
- `ClaudeEvent` → `Event` (generic name since all runners produce the same event shape)
- The `Event` type stays in the shared package because **all runners must produce events in this format** — it's the contract between runners and consumers
- `RunClaude()` is removed from this file (moved to the claude sub-package)
- `RunOptions` stays here as the shared options type

### 1.2 — Create the Claude runner sub-package

**New file**: `internal/runner/claude/claude.go`

```go
// Package claude implements the Runner interface for the Claude CLI agent.
package claude

import (
    "bufio"
    "encoding/json"
    "fmt"
    "io"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "time"

    "github.com/jumppad-labs/spektacular/internal/config"
    "github.com/jumppad-labs/spektacular/internal/runner"
)

// Claude implements runner.Runner by spawning the Claude CLI subprocess.
type Claude struct{}

// New returns a new Claude runner.
func New() *Claude { return &Claude{} }

// Run spawns the claude subprocess and returns a channel of events and an error channel.
func (c *Claude) Run(opts runner.RunOptions) (<-chan runner.Event, <-chan error) {
    events := make(chan runner.Event, 64)
    errc := make(chan error, 1)

    go func() {
        defer close(events)
        if err := run(opts, events); err != nil {
            errc <- err
        }
        close(errc)
    }()

    return events, errc
}

func run(opts runner.RunOptions, events chan<- runner.Event) error {
    cfg := opts.Config
    cmd := []string{cfg.Agent.Command, "-p", opts.Prompt}
    if opts.SystemPrompt != "" {
        cmd = append(cmd, "--system-prompt", opts.SystemPrompt)
    }
    cmd = append(cmd, cfg.Agent.Args...)

    if len(cfg.Agent.AllowedTools) > 0 {
        cmd = append(cmd, "--allowedTools", strings.Join(cfg.Agent.AllowedTools, ","))
    }
    if cfg.Agent.DangerouslySkipPermissions {
        cmd = append(cmd, "--dangerously-skip-permissions")
    }
    if opts.SessionID != "" {
        cmd = append(cmd, "--resume", opts.SessionID)
    }

    cwd := opts.CWD
    if cwd == "" {
        var err error
        cwd, err = os.Getwd()
        if err != nil {
            return fmt.Errorf("getting working directory: %w", err)
        }
    }

    proc := exec.Command(cmd[0], cmd[1:]...) //nolint:gosec
    proc.Dir = cwd
    proc.Stderr = io.Discard

    stdout, err := proc.StdoutPipe()
    if err != nil {
        return fmt.Errorf("creating stdout pipe: %w", err)
    }
    if err := proc.Start(); err != nil {
        return fmt.Errorf("starting claude process: %w", err)
    }

    var debugLog *os.File
    if cfg.Debug.Enabled {
        debugLog = openDebugLog(cfg, opts.Command, cwd)
        if debugLog != nil {
            defer debugLog.Close()
        }
    }

    scanner := bufio.NewScanner(stdout)
    scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1 MiB lines
    for scanner.Scan() {
        line := scanner.Text()
        if line == "" {
            continue
        }
        if debugLog != nil {
            fmt.Fprintln(debugLog, line)
        }
        var data map[string]any
        if err := json.Unmarshal([]byte(line), &data); err != nil {
            continue
        }
        eventType, _ := data["type"].(string)
        events <- runner.Event{Type: eventType, Data: data}
    }

    if err := proc.Wait(); err != nil {
        return fmt.Errorf("claude process exited with error: %w", err)
    }
    return nil
}

func openDebugLog(cfg config.Config, command, cwd string) *os.File {
    logDir := filepath.Join(cwd, cfg.Debug.LogDir)
    if err := os.MkdirAll(logDir, 0755); err != nil {
        return nil
    }
    ts := time.Now().Format("2006-01-02_150405")
    filename := fmt.Sprintf("%s_%s_%s.log", ts, cfg.Agent.Command, command)
    f, err := os.Create(filepath.Join(logDir, filename))
    if err != nil {
        return nil
    }
    return f
}
```

### 1.3 — Add backward-compatible aliases (temporary)

To keep the refactoring safe, add type aliases and a wrapper function in `internal/runner/runner.go` so that existing call sites don't break during the transition:

```go
// ClaudeEvent is a deprecated alias for Event.
// TODO: Remove once all call sites are updated.
type ClaudeEvent = Event

// RunClaude is a backward-compatible wrapper that creates a Claude runner and calls Run.
// Deprecated: Use claude.New().Run(opts) directly, or obtain a Runner from NewRunner().
func RunClaude(opts RunOptions) (<-chan Event, <-chan error) {
    r := defaultRunner()
    return r.Run(opts)
}
```

This ensures **zero breakage** during the transition. Call sites can be updated incrementally.

### 1.4 — Add a factory function for creating runners from config

**Add to `internal/runner/runner.go`**:

```go
// NewRunner returns a Runner based on the agent command in the config.
// Currently only "claude" is supported; future runners (bob, openai, etc.)
// will be added here.
func NewRunner(cfg config.Config) (Runner, error) {
    switch cfg.Agent.Command {
    case "claude":
        return claude.New(), nil
    default:
        return nil, fmt.Errorf("unsupported runner: %q", cfg.Agent.Command)
    }
}
```

**Note**: This creates a circular import (`runner` → `runner/claude` → `runner`). To resolve this, the factory lives in a separate file or we use the existing config to instantiate. The cleaner approach:

- The `runner` package defines the interface
- The `runner/claude` package implements it
- A top-level `runner.NewRunner()` factory imports `runner/claude`

This is fine because `runner/claude` imports `runner` (for types), and `runner` imports `runner/claude` (for construction). **This is a circular import.**

**Resolution**: Move the factory function to a separate package or make it a function that each call site handles. The simplest approach is:

**Option A (Recommended)**: Keep the factory in `runner` but use a registration pattern:

```go
// internal/runner/registry.go

var registry = map[string]func() Runner{}

// Register adds a runner constructor for a given command name.
func Register(name string, constructor func() Runner) {
    registry[name] = constructor
}

// NewRunner returns a Runner for the agent command in the config.
func NewRunner(cfg config.Config) (Runner, error) {
    constructor, ok := registry[cfg.Agent.Command]
    if !ok {
        return nil, fmt.Errorf("unsupported runner: %q (available: %v)", cfg.Agent.Command, registeredNames())
    }
    return constructor(), nil
}

func registeredNames() []string {
    names := make([]string, 0, len(registry))
    for k := range registry {
        names = append(names, k)
    }
    return names
}
```

Then in `internal/runner/claude/claude.go`, add an `init()`:

```go
func init() {
    runner.Register("claude", func() runner.Runner { return New() })
}
```

And ensure `runner/claude` is imported (blank import) in `main.go` or `cmd/root.go`:

```go
import _ "github.com/jumppad-labs/spektacular/internal/runner/claude"
```

**Option B (Simpler, no registration)**: Put the factory in the `cmd` package or wherever runners are constructed, avoiding circular imports entirely. The call sites would do:

```go
import "github.com/jumppad-labs/spektacular/internal/runner/claude"

r := claude.New()
events, errc := r.Run(opts)
```

**Recommendation**: Use **Option A** (registry pattern). It's more extensible and keeps the runner creation logic in one place. It follows Go conventions (similar to `database/sql` driver registration, `image` format registration, etc.).

## Phase 2: Update All Call Sites

### 2.1 — Update `internal/plan/plan.go`

**Change**: Replace `runner.RunClaude()` with interface-based call.

```go
// RunPlan executes the full plan-generation loop for specPath.
func RunPlan(
    specPath, projectPath string,
    cfg config.Config,
    onText func(string),
    onQuestion func([]runner.Question) string,
) (string, error) {
    r, err := runner.NewRunner(cfg)
    if err != nil {
        return "", fmt.Errorf("creating runner: %w", err)
    }

    // ... (rest unchanged, just replace runner.RunClaude(opts) with r.Run(opts))

    for {
        // ...
        events, errc := r.Run(runner.RunOptions{
            Prompt:       currentPrompt,
            SystemPrompt: agentPrompt,
            Config:       cfg,
            SessionID:    sessionID,
            CWD:          projectPath,
            Command:      "plan",
        })
        // ... (event handling unchanged)
    }
}
```

### 2.2 — Update `internal/implement/implement.go`

**Change**: Same pattern as plan — replace `runner.RunClaude()` with `r.Run()`.

```go
func RunImplement(
    planDir, projectPath string,
    cfg config.Config,
    onText func(string),
    onQuestion func([]runner.Question) string,
) (string, error) {
    r, err := runner.NewRunner(cfg)
    if err != nil {
        return "", fmt.Errorf("creating runner: %w", err)
    }

    // ...
    for {
        events, errc := r.Run(runner.RunOptions{
            // ... same options
        })
        // ... (unchanged event handling)
    }
}
```

### 2.3 — Update `internal/tui/tui.go`

The TUI is more complex because `runner.RunClaude()` is called inside closures that produce `tea.Cmd` values. The runner instance needs to be accessible within these closures.

**Approach**: Pass the runner through the `Workflow.Start` function or store it in the model. Since `Workflow.Start` already receives `config.Config`, the cleanest approach is to create the runner inside the `Start` closures:

**Updated `startAgentCmd`**:
```go
func startAgentCmd(specPath, projectPath string, cfg config.Config, sessionID string) tea.Cmd {
    return func() tea.Msg {
        r, err := runner.NewRunner(cfg)
        if err != nil {
            return agentErrMsg{err: fmt.Errorf("creating runner: %w", err)}
        }
        // ... rest unchanged, just use r.Run(opts) instead of runner.RunClaude(opts)
    }
}
```

**Updated `resumeAgentCmd`**:
```go
func resumeAgentCmd(cfg config.Config, sessionID, projectPath, answer string) tea.Cmd {
    return func() tea.Msg {
        r, err := runner.NewRunner(cfg)
        if err != nil {
            return agentErrMsg{err: fmt.Errorf("creating runner: %w", err)}
        }
        events, errc := r.Run(runner.RunOptions{
            Prompt:    answer,
            Config:    cfg,
            SessionID: sessionID,
            CWD:       projectPath,
            Command:   "plan",
        })
        return readNext(events, errc)
    }
}
```

**Updated `implementStartCmd`** — same pattern.

### 2.4 — Update type references across all files

All references to `runner.ClaudeEvent` must be updated to `runner.Event`:
- `internal/tui/tui.go:27-31` — `agentEventMsg` fields
- `internal/tui/tui.go:159` — `waitForEvent` parameter types

Since we're using a type alias (`ClaudeEvent = Event`) in Phase 1, these can be updated incrementally. But it's cleaner to do them all at once.

### 2.5 — Add blank import for Claude runner registration

**File**: `cmd/root.go`

```go
import (
    // Register the Claude runner so it's available via runner.NewRunner().
    _ "github.com/jumppad-labs/spektacular/internal/runner/claude"
)
```

### 2.6 — Remove deprecated aliases

Once all call sites are updated, remove from `internal/runner/runner.go`:
- The `ClaudeEvent` type alias
- The `RunClaude()` wrapper function
- The `runClaude()` internal function
- The `openDebugLog()` function (moved to claude package)

## Testing Strategy

### Unit Tests

**`internal/runner/runner_test.go`** (update existing):
- Rename `TestClaudeEvent_*` → `TestEvent_*`
- All existing test logic is unchanged (just the type name changes)
- Add test for `NewRunner()` factory:
  - Returns claude runner when `cfg.Agent.Command == "claude"`
  - Returns error for unknown command

**`internal/runner/claude/claude_test.go`** (new):
- Test that `Claude` implements `runner.Runner` interface (compile-time check):
  ```go
  var _ runner.Runner = (*Claude)(nil)
  ```
- Test `New()` returns non-nil

**Existing tests** (`plan_test.go`, `implement_test.go`, `tui_test.go`):
- Should pass without changes since they test file I/O and type properties, not the runner invocation directly

### Integration Verification
- `make test` — all existing tests pass
- `make build` — binary compiles cleanly
- `make lint` — no vet issues

## File Change Summary

| File | Action | Description |
|------|--------|-------------|
| `internal/runner/runner.go` | **Modify** | Add `Runner` interface, rename `ClaudeEvent` → `Event`, remove Claude-specific code, add backward compat aliases |
| `internal/runner/registry.go` | **Create** | Runner registration and factory |
| `internal/runner/claude/claude.go` | **Create** | Claude runner implementation (extracted from runner.go) |
| `internal/runner/claude/claude_test.go` | **Create** | Interface compliance test |
| `internal/runner/runner_test.go` | **Modify** | Rename `ClaudeEvent` → `Event` in tests, add `NewRunner` tests |
| `internal/plan/plan.go` | **Modify** | Use `runner.NewRunner()` instead of `runner.RunClaude()` |
| `internal/implement/implement.go` | **Modify** | Use `runner.NewRunner()` instead of `runner.RunClaude()` |
| `internal/tui/tui.go` | **Modify** | Use `runner.NewRunner()` instead of `runner.RunClaude()`, update type refs |
| `cmd/root.go` | **Modify** | Add blank import for claude runner registration |

## Success Criteria

### Automated Verification
- [ ] `make test` — all tests pass (existing + new)
- [ ] `make build` — compiles without errors
- [ ] `make lint` — no linting issues
- [ ] `var _ runner.Runner = (*Claude)(nil)` compiles (interface compliance)

### Manual Verification
- [ ] `spektacular plan <spec>` works identically to before
- [ ] `spektacular implement <plan>` works identically to before
- [ ] TUI mode works (both plan and implement)
- [ ] Non-TTY streaming mode works
- [ ] Debug logging still works

### Design Verification
- [ ] Adding a new runner requires only: (1) new sub-package, (2) `runner.Register()` call, (3) blank import
- [ ] No Claude-specific types leak outside `internal/runner/claude/`
- [ ] The `Runner` interface is minimal and sufficient

## Migration & Rollout
- **No data migration** — this is a pure code refactoring
- **No configuration changes required** — existing `config.yaml` files work as-is since `agent.command: claude` already maps to the right runner
- **Backward compatibility** — type aliases ensure a smooth transition
- **Rollback** — simple git revert since no data/schema changes

## References
- `internal/runner/runner.go:1-249` — current monolithic runner
- `internal/plan/plan.go:114-121` — plan's RunClaude call site
- `internal/implement/implement.go:95-102` — implement's RunClaude call site
- `internal/tui/tui.go:132-141` — TUI's startAgentCmd call site
- `internal/tui/tui.go:145-155` — TUI's resumeAgentCmd call site
- `internal/tui/tui.go:674-696` — TUI's implementStartCmd call site
- `internal/config/config.go:57-62` — AgentConfig structure
- Go stdlib `database/sql` — registration pattern precedent
- Go stdlib `image` — format registration pattern precedent
