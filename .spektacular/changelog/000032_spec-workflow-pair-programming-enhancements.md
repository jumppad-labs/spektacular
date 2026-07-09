# Feature Changelog: 000032_spec-workflow-pair-programming-enhancements

## What Was Built

The spec workflow now captures conversation context for resumption. When you start a spec via `spec new`, the workflow clears `.spektacular/context.md` and returns an instruction telling the agent to write detailed discussion context to that file. The instruction specifies capturing:

- The problem identified and why it needs solving
- All requirements and constraints discussed
- Alternatives considered and why they were rejected
- The user's exact phrasing for key requirements

If no meaningful context exists (e.g., the user simply said "create a spec for X" without elaboration), the agent skips the write, leaving context.md empty. This context persists across sessions, so if the workflow is interrupted and later resumed, the agent can read context.md to recover the discussion details without re-asking the user.

**Implementation**: Modified `internal/steps/spec/steps.go` to add a new `new()` step callback that clears context.md after creating the spec scaffold, then uses `writeStep()` to render a new template (`templates/steps/spec/00-new.md`) containing the instruction. Added 5 comprehensive tests in `internal/steps/spec/steps_test.go` verifying all acceptance criteria.

## Why It Matters

This addresses a gap where interrupted spec workflows lost all conversation context. Previously, if a session ended mid-workflow (crash, reboot, closed terminal), resuming meant starting from scratch with no memory of what had been discussed. Now the conversation context is explicitly captured in a git-tracked file that survives interruptions, enabling true resumption without forcing users to re-explain their requirements.

This is Phase 2.1 of a larger feature (spec 000032) that will eventually enable proactive spec-creation offers during open-ended discussions. Phase 2.1 delivers the context-persistence mechanism; future phases will add the threshold configuration and AGENTS.md instruction delivery for recognizing spec-worthy discussions.

## Deviations from Plan

None. Implementation followed the plan exactly. All acceptance criteria were met:

- ✅ `new()` step clears `.spektacular/context.md` after creating spec scaffold
- ✅ `new` step returns instruction (not silent) via writeStep
- ✅ Instruction specifies detailed format: problem, requirements, constraints, alternatives, exact phrasing
- ✅ Instruction includes caveat to skip if no meaningful context
- ✅ Resume preserves context.md (new step doesn't run on goto)
- ✅ Existing workflows unchanged

**Discovery**: The new() step now uses writeStep() instead of returning "overview" directly, which changes the FSM behavior slightly - the workflow stays at "new" state after the first transition and returns an instruction. Tests were updated to reflect this. The relative path `.spektacular/context.md` resolves correctly against the current working directory (project root) when running `go run . spec new`.
