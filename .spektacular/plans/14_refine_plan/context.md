# Refine Plan — Context

## Key Files and Purpose

| File | Purpose |
|---|---|
| `internal/tui/tui.go` | BubbleTea TUI — model, Update loop, View, all UI state management. Primary file for changes. |
| `internal/steps/plan.go` | Constructs the `tui.Workflow` for the plan command. Configures steps and callbacks. |
| `internal/plan/plan.go` | Plan-specific helpers: `LoadKnowledge()`, `PreparePlanDir()`, `WritePlanOutput()`, `LoadAgentPrompt()`. |
| `internal/runner/runner.go` | `Runner` interface, `Event` type, `RunOptions`, `DetectFinished()`, `DetectQuestions()`, prompt builders. |
| `internal/runner/claude/claude.go` | Claude CLI runner implementation. Spawns `claude` subprocess, streams JSONL events. |
| `cmd/plan.go` | Cobra command for `spektacular plan`. TTY path uses TUI, non-TTY streams directly. |
| `internal/tui/tui_test.go` | Existing TUI unit tests. Pattern: direct struct construction, `testify/require`. |
| `internal/tui/textarea_test.go` | Existing textarea input tests. Shows how to test `handleTextareaInput()`. |

## Important Types and Interfaces

### `tui.Workflow`
Multi-step agent pipeline for the TUI. Key fields: `Steps []WorkflowStep`, `OnDone func() (string, error)`, `Preamble string`. **Will add**: `Feedback *FeedbackConfig`.

### `tui.WorkflowStep`
One step in a workflow. `BuildRunOptions` creates the `runner.RunOptions` at step start.

### `tui.model`
BubbleTea model. Manages all UI state: viewport, questions, textarea, status text, step progress. **Will add**: `feedbackMode bool`, `feedbackRound int`.

### `runner.RunOptions`
Parameters for running an agent: `Prompts`, `Config`, `SessionID`, `CWD`, `LogFile`, `Model`.

### `runner.Event`
Parsed event from agent output. Methods: `SessionID()`, `TextContent()`, `IsResult()`, `IsError()`, `ToolUses()`.

## Key Control Flow

### Current Plan Workflow
```
cmd/plan.go → steps.PlanWorkflow() → tui.RunAgentTUI()
  → model.Init() → startCurrentStep()
    → step.BuildRunOptions() → runner.Run() → events channel
  → model.Update loop:
    → agentEventMsg → handleAgentEvent()
      → text with FINISHED? → advanceStep()
      → text with QUESTION? → show question panel, wait for answer
      → result event? → advanceStep()
    → advanceStep():
      → more steps? → start next step
      → all done? → OnDone() → set m.done = true
```

### New Feedback Flow (after this feature)
```
  → advanceStep() (all steps done):
    → Feedback configured? → enterFeedbackMode()
      → show PromptMessage / PostUpdateMessage
      → activate textarea
    → No Feedback? → completeWorkflow() (existing behavior)
  → handleTextareaInput() (feedbackMode=true):
    → ctrl+d with content → handleFeedbackSubmit()
      → build prompt via Feedback.BuildPrompt()
      → resumeAgentCmd() → agent processes feedback
        → may ask QUESTION → existing handling
        → FINISHED → advanceStep() → enterFeedbackMode() (next round)
    → ctrl+d empty / esc → completeWorkflow()
```

## Agent Session Resumption

The Claude CLI supports session resumption via `--resume SESSION_ID`. The TUI tracks `sessionID` on the model and passes it through `resumeAgentCmd()`. This means:

1. After initial plan generation, the agent has full context of what it generated
2. When we resume with feedback, the agent knows what plan it wrote
3. The agent can use its tools (Read, Write, Edit) to re-read and modify plan files
4. Multiple feedback rounds all share the same session — context accumulates

## Testing Patterns

- Use `testify/require` (not `assert`)
- Construct models directly: `m := model{feedbackMode: true, ...}`
- Use `initialModel(wf, path, cfg)` helper for default setup
- Test tea.Cmd return values by checking model state changes (not executing commands)
- See `textarea_test.go` for how to test `handleTextareaInput()` with `tea.KeyMsg`
