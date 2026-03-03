# Refine Plan — Research Notes

## Design Decision: Feedback Loop Location

### Option A: New Workflow Step (Rejected)
Add a special "repeatable" step to the workflow that runs after the plan step. This would require changes to the step advancement logic and how BuildRunOptions works (it would need the user's feedback as input).

**Rejected because**: Steps are designed to be independent units with their own prompts. A feedback step would need dynamic input (the user's feedback text) and would create a new session, losing the planning context.

### Option B: Post-Completion TUI State (Chosen)
Add a feedback mode to the TUI model that activates after all steps complete. The mode uses the existing textarea for input and resumes the agent session with a feedback prompt.

**Chosen because**:
- Reuses existing infrastructure (textarea, resumeAgentCmd, question handling)
- Preserves agent session context (same session ID)
- Clean separation: workflow steps handle planning, feedback mode handles refinement
- Agent clarification questions work automatically via existing QUESTION mechanism

### Option C: External Loop (Rejected)
Handle the feedback loop outside the TUI, in `cmd/plan.go`. Re-run the TUI in a loop.

**Rejected because**: Would lose the session context between iterations. Each TUI run creates a new session. The agent would not remember what plan it generated.

## Design Decision: Feedback Prompt Strategy

### Option A: Include Plan Content in Prompt (Rejected)
Read plan files from disk and include their content in the feedback prompt sent to the agent.

**Rejected because**: The agent already has context from the planning session. Including full plan content wastes tokens and may exceed context limits for large plans. The agent can read files itself.

### Option B: Instruct Agent to Re-Read (Chosen)
Tell the agent to re-read plan files from disk before acting. The agent uses its Read tool to get current file contents.

**Chosen because**:
- Handles direct file edits naturally (user edits files → agent reads latest version)
- Token-efficient (only the feedback text is in the prompt)
- Agent has full tool access to read, write, and edit files
- Aligns with the spec requirement: "agent must always re-read the plan files from disk"

## Edge Cases Considered

### 1. Agent asks clarification during feedback
- Agent outputs `<!--QUESTION:...-->` → TUI handles natively
- User answers → session resumes → agent modifies plan → FINISHED
- Then `advanceStep()` → `enterFeedbackMode()` for next round
- **No special handling needed** — existing question flow works

### 2. User provides no feedback
- Empty textarea → `handleFeedbackSubmit("")` → `completeWorkflow()`
- Alternatively, user presses Esc → `completeWorkflow()`
- **No plan modification occurs**

### 3. User edits plan files between rounds
- Agent is instructed to re-read from disk every round
- The feedback prompt says: "The user may have edited them directly"
- Agent reads current file contents, not cached versions

### 4. Agent fails during feedback processing
- Error in runner → `agentErrMsg` → error displayed, `m.done = true`
- Same error handling as during initial planning

### 5. Many feedback rounds
- `currentStep` keeps incrementing past `len(workflow.Steps)` — no issue
- `feedbackRound` increments each round — used for message selection only
- Session context accumulates — agent remembers all previous feedback

### 6. Q key behavior
- During initial planning: q quits (existing)
- During feedback textarea: q types the letter (textarea has focus)
- When feedback textarea is dismissed: q should not quit (user may want to re-enter)
- After completeWorkflow(): q quits (done state)
