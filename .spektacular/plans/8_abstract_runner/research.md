# Abstract Runner - Research Notes

## Specification Analysis

### Original Requirements
1. Define a common interface for all runners to implement
2. Abstract the current Claude implementation into a separate runner that implements the common interface

### Implicit Requirements
- Backward compatibility during transition (no big-bang rewrite)
- The interface must support the existing event-streaming pattern (channels)
- Configuration-driven runner selection (via `agent.command` in config)
- Future extensibility for other tools (Bob, OpenAI, Aider, etc.)
- No behavioral changes to end users

### Constraints Identified
- Go's type system requires careful handling of circular imports between `runner` (interface) and `runner/claude` (implementation)
- The TUI uses channels in closures (tea.Cmd), which complicates dependency injection
- No existing mock/DI infrastructure — the codebase prefers concrete types and function parameters
- The `Event` type is tightly coupled to Claude's stream-JSON format, but this format is generic enough to serve as a common contract

## Research Process

### Sub-agents Spawned
1. **Codebase Explorer** — Mapped all files, packages, and dependencies
2. **Claude Runner Analyzer** — Found all references to Claude execution
3. **Interface Pattern Researcher** — Discovered existing abstraction patterns
4. **Learnings Search** — Checked for institutional knowledge

### Files Examined

| File | Lines | Key Findings |
|------|-------|-------------|
| `internal/runner/runner.go` | 249 | Monolithic: shared types + Claude subprocess + debug logging |
| `internal/runner/runner_test.go` | 137 | Tests Event properties and question detection — no subprocess tests |
| `internal/plan/plan.go` | 167 | Calls `runner.RunClaude()` at line 114, event loop at 123-139 |
| `internal/plan/plan_test.go` | 75 | Tests file I/O only, no runner interaction |
| `internal/implement/implement.go` | 137 | Calls `runner.RunClaude()` at line 95, identical event loop |
| `internal/implement/implement_test.go` | 73 | Tests file I/O only, no runner interaction |
| `internal/tui/tui.go` | 717 | Three call sites for `runner.RunClaude()`, uses `runner.ClaudeEvent` |
| `internal/tui/tui_test.go` | — | Exists but minimal |
| `internal/config/config.go` | 161 | `AgentConfig.Command` already supports configuration of agent name |
| `cmd/root.go` | 31 | Registers all commands, good place for blank imports |
| `cmd/plan.go` | 66 | Creates config, delegates to plan.RunPlan or tui.RunPlanTUI |
| `cmd/implement.go` | 67 | Creates config, delegates to implement.RunImplement or tui.RunImplementTUI |

### Patterns Discovered

**1. Existing Workflow Abstraction (tui.go:40-53)**
```go
type Workflow struct {
    StatusLabel string
    Start       func(cfg config.Config, sessionID string) tea.Cmd
    OnResult    func(resultText string) (string, error)
}
```
This is a partial strategy pattern — it abstracts the workflow (plan vs implement) but not the runner backend. Our Runner interface complements this by abstracting the other axis.

**2. Callback-based DI Pattern (plan.go:82-87)**
```go
func RunPlan(
    specPath, projectPath string,
    cfg config.Config,
    onText func(string),
    onQuestion func([]runner.Question) string,
) (string, error)
```
The codebase prefers function parameters for dependency injection rather than interfaces. However, for the runner abstraction, an interface is more appropriate because runners have state and multiple methods could be needed in the future.

**3. Config-Driven Agent Selection (config.go:100-105)**
```go
Agent: AgentConfig{
    Command:      "claude",
    Args:         []string{"--output-format", "stream-json", "--verbose"},
    AllowedTools: []string{"Task", "Bash", ...},
}
```
The `Command` field already serves as a runner identifier. This maps naturally to a registry key.

**4. Event Streaming Pattern (runner.go:149-164)**
```go
func RunClaude(opts RunOptions) (<-chan ClaudeEvent, <-chan error) {
    events := make(chan ClaudeEvent, 64)
    errc := make(chan error, 1)
    go func() {
        defer close(events)
        if err := runClaude(opts, events); err != nil {
            errc <- err
        }
        close(errc)
    }()
    return events, errc
}
```
This channel-based pattern is idiomatic Go and works well as an interface contract.

## Key Findings

### Architecture Insights
- The system has a clean separation between **orchestration** (plan/implement) and **execution** (runner) — the refactoring boundary is clear
- The `Event` type is actually transport-agnostic — it's just `{Type string, Data map[string]any}` which any runner can produce
- The TUI and non-TUI paths both consume events identically, so the interface only needs to abstract the event source

### Existing Implementations
- No other runner implementations exist
- The `run` command in `cmd/run.go` is a TODO placeholder
- The `.spektacular/knowledge/architecture/initial-idea.md` mentions plans for Claude Code, Aider, and Cursor adapters

### Reusable Components
- `Event` type and all its methods
- `Question` type and `DetectQuestions()`
- `BuildPrompt()` and `BuildPromptWithHeader()`
- `RunOptions` struct
- The event loop pattern in plan.go and implement.go (identical, could be extracted later)

### Testing Infrastructure
- Uses `github.com/stretchr/testify/require` for assertions
- No mock libraries — tests use concrete types with test data
- Tests are focused on pure functions and file I/O
- No subprocess-level tests exist for the runner

## Design Decisions

### Decision 1: Registry Pattern vs Direct Imports
- **Choice**: Registry pattern (`runner.Register()` + `runner.NewRunner()`)
- **Options Considered**:
  - Direct imports in each call site (`claude.New()`)
  - Factory function with switch statement
  - Registry with `init()` registration
- **Rationale**: The registry pattern avoids circular imports, is idiomatic Go (used by `database/sql`, `image`), and makes it trivial to add new runners
- **Trade-offs**: Slightly more indirection; requires blank import in main/root

### Decision 2: Type Rename Strategy
- **Choice**: Rename `ClaudeEvent` → `Event` with temporary type alias
- **Options Considered**:
  - Keep `ClaudeEvent` name (confusing for non-Claude runners)
  - Big-bang rename (risky)
  - Type alias for backward compat (chosen)
- **Rationale**: The alias lets us update files incrementally while keeping everything compiling
- **Trade-offs**: Temporary code debt (aliases to remove later)

### Decision 3: Single Interface Method
- **Choice**: `Runner` interface has one method: `Run(opts RunOptions) (<-chan Event, <-chan error)`
- **Options Considered**:
  - Separate `Start()`, `Stop()`, `Resume()` methods
  - `Run()` + `Health()` methods
  - Single `Run()` method (chosen)
- **Rationale**: The current codebase treats each invocation as stateless (session ID is passed via RunOptions). A single method keeps the interface minimal and easy to implement.
- **Trade-offs**: May need to expand the interface later if runners need lifecycle management

### Decision 4: Event Format as Universal Contract
- **Choice**: All runners must produce `Event{Type, Data}` in the same format
- **Options Considered**:
  - Runner-specific event types with adapters
  - Universal event type (chosen)
  - Interface-based events with method extraction
- **Rationale**: The existing Event type is already generic (`map[string]any`). Non-Claude runners would emit events with the same structure (system/assistant/result types with appropriate data)
- **Trade-offs**: Other runners will need to adapt their output to this format, but this is appropriate — it means consumers don't need to know which runner produced the event

## Open Questions (All Resolved)

All questions were resolved during research:

1. **Q**: Should the runner factory use a registry or a switch statement?
   **A**: Registry — more extensible, idiomatic Go
   **Impact**: Added `registry.go` file to the plan

2. **Q**: Where should the blank import go?
   **A**: `cmd/root.go` — it's the central initialization point
   **Impact**: Single line change to root.go

3. **Q**: Should `RunOptions` include runner-specific fields?
   **A**: No — keep it generic. Runner-specific config can live in `config.Agent.Args` or future runner-specific config sections
   **Impact**: No config changes needed

4. **Q**: Should the event loop be extracted into a shared function?
   **A**: Not in this PR — it's a separate refactoring concern. The plan/implement event loops are nearly identical but extracting them adds complexity without being required by the spec
   **Impact**: Deferred to future work
