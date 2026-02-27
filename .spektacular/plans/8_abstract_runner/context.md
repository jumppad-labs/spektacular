# Abstract Runner - Context

## Quick Summary
Extract a `Runner` interface from the monolithic Claude runner, move the Claude implementation into its own sub-package, and update all call sites to use the interface via a registry-based factory.

## Key Files & Locations

### Primary Implementation (to modify/create)
- `internal/runner/runner.go` — Interface definition + shared types (modify)
- `internal/runner/registry.go` — Runner registration & factory (create)
- `internal/runner/claude/claude.go` — Claude runner implementation (create)
- `internal/runner/claude/claude_test.go` — Interface compliance test (create)

### Call Sites (to update)
- `internal/plan/plan.go:114-121` — `runner.RunClaude()` → `r.Run()`
- `internal/implement/implement.go:95-102` — `runner.RunClaude()` → `r.Run()`
- `internal/tui/tui.go:132-141` — `startAgentCmd` uses `runner.RunClaude()`
- `internal/tui/tui.go:145-155` — `resumeAgentCmd` uses `runner.RunClaude()`
- `internal/tui/tui.go:674-696` — `implementStartCmd` uses `runner.RunClaude()`

### Configuration
- `internal/config/config.go:57-62` — `AgentConfig` (no changes needed)
- `.spektacular/config.yaml` — Runtime config (no changes needed)

### Tests
- `internal/runner/runner_test.go` — Update `ClaudeEvent` → `Event`
- `internal/plan/plan_test.go` — No changes needed (tests file I/O)
- `internal/implement/implement_test.go` — No changes needed (tests file I/O)

### Registration Import
- `cmd/root.go` — Add `_ "github.com/jumppad-labs/spektacular/internal/runner/claude"`

## Dependencies

### Code Dependencies
- `internal/runner` — defines the interface (no new deps)
- `internal/runner/claude` — imports `internal/runner` and `internal/config`
- `internal/plan` — imports `internal/runner` (unchanged)
- `internal/implement` — imports `internal/runner` (unchanged)
- `internal/tui` — imports `internal/runner` (unchanged)

### External Dependencies
- None new — all existing Go modules are sufficient

### Database Changes
- None

## Environment Requirements

### Configuration Variables
- None new — existing `agent.command: claude` maps to the registry

### Migration Scripts
- None

### Feature Flags
- None

## Integration Points

### API Endpoints
- None affected

### Event Stream Contract
All runners must produce events in this format:
```
Event{Type: "system"|"assistant"|"result", Data: map[string]any{...}}
```

- `system` events must contain `session_id`
- `assistant` events must contain `message.content` array with `text` and `tool_use` blocks
- `result` events must contain `result` (string) and optionally `is_error` (bool)

### Adding a New Runner (Future)
1. Create `internal/runner/<name>/<name>.go`
2. Implement `runner.Runner` interface
3. Call `runner.Register("<name>", func() runner.Runner { return New() })` in `init()`
4. Add blank import in `cmd/root.go`: `_ "github.com/jumppad-labs/spektacular/internal/runner/<name>"`
5. User sets `agent.command: <name>` in config.yaml

## Build & Verification Commands
```bash
make test     # All tests pass
make build    # Binary compiles
make lint     # No issues
```
