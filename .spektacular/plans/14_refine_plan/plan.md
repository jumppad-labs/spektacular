# Refine Plan — Implementation Plan

## Overview
- **Spec**: 14_refine_plan
- **Complexity**: Low
- **Dependencies**: `charmbracelet/bubbletea`, existing TUI workflow system, existing runner/session resume infrastructure

## Current State

The plan workflow (`spektacular plan <spec>`) ran a single agent step that generated plan files, then completed. There was no feedback loop — once the plan was generated, the workflow ended.

**What already exists that we leveraged:**
- **Session resumption**: `resumeAgentCmd()` in `internal/tui/tui.go` resumes the Claude session with a user message. The agent retains full context.
- **QUESTION marker**: The agent can emit `<!--QUESTION:...-->` to collect user input. The TUI shows a question panel, collects the answer, and resumes the session.
- **FINISHED marker**: `<!-- FINISHED -->` drives step transitions via `advanceStep()`.

## Implementation

### Core mechanism: named steps + `<!-- GOTO: step name -->`

Each `WorkflowStep` now has a `Name` field. When an agent emits `<!-- GOTO: step name -->`, the TUI jumps to the step with that name — setting `currentStep` to that index and calling `startCurrentStep()` with the existing `sessionID`. This means the agent resumes in the same conversation with full context.

This replaces the original `<!-- LOOP -->` approach and is more general: any step can navigate to any named step.

### Changes

**`internal/runner/runner.go`**
- Added `gotoPattern = regexp.MustCompile(`<!--\s*GOTO:\s*([\w][\w\s-]*?)\s*-->`)`
- Added `DetectGoto(text string) (string, bool)` — returns target step name if marker is present
- Updated `StripMarkers()` to also strip `<!-- GOTO:... -->` markers

**`internal/tui/tui.go`**
- Added `Name string` field to `WorkflowStep`
- Added `gotoStep(name string) (tea.Model, tea.Cmd)` — finds step by name, sets `currentStep`, clears question state, preserves `sessionID`, calls `startCurrentStep()`
- In `handleAgentEvent`: detect `<!-- GOTO: ... -->` via `DetectGoto` and call `gotoStep(target)` alongside existing `<!-- FINISHED -->` handling

**`internal/steps/plan.go`**
- Added `feedbackStep(planDir string)` — a new step (name: `"feedback"`) added after `planStep` in `PlanWorkflow`
- Added `buildFeedbackPrompt(planDir string)` — instructs the agent to re-read plan files, ask for feedback via `<!--QUESTION:...-->`, incorporate feedback and emit `<!-- GOTO: feedback -->` for another round, or emit `<!-- FINISHED -->` when the user submits empty input
- Added `Name: "plan"` to `planStep`

**`internal/steps/new.go`** — added `Name` to all 7 spec steps: `"overview"`, `"requirements"`, `"acceptance-criteria"`, `"constraints"`, `"technical-approach"`, `"success-metrics"`, `"non-goals"`

**`internal/steps/implement.go`** — added `Name: "implement"` to `implementStep`

### Plan feedback flow

```
planStep runs → FINISHED → advanceStep() → feedbackStep starts
  → agent re-reads plan files
  → agent asks: "Do you have any feedback?" via <!--QUESTION:...-->
  → user submits feedback
    → non-empty: agent incorporates feedback → <!-- GOTO: feedback -->
      → TUI jumps back to feedbackStep (same session, agent retains context)
      → repeat...
    → empty: agent outputs <!-- FINISHED --> → workflow completes
```

## Not In Scope

- **Non-TTY feedback loop**: The non-TTY path in `cmd/plan.go` does not get feedback support. Interactive feedback requires a terminal.
- **Implement command feedback**: Only the `plan` command gets a feedback step. The implement workflow could add one by including a named feedback step.

## Files Changed

- `internal/runner/runner.go` — `DetectGoto`, `gotoPattern`, `StripMarkers`
- `internal/tui/tui.go` — `WorkflowStep.Name`, `gotoStep()`
- `internal/steps/plan.go` — `feedbackStep`, `buildFeedbackPrompt`, named steps
- `internal/steps/new.go` — named steps
- `internal/steps/implement.go` — named step
