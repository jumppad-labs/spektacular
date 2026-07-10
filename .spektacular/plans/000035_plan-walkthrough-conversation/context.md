# Context: 000035_plan-walkthrough-conversation

## Current State Analysis

Today, the plan workflow's `finished` step (`internal/steps/plan/steps.go:277-299`, rendering `templates/steps/plan/18-finished.md`) is the FSM's terminal step (`internal/steps/plan/steps.go:48` — `{Name: "finished", Src: []string{"write_research"}, Dst: "finished", Callback: finished()}`, nothing transitions out of it). Once all three plan documents (`plan.md`, `context.md`, `research.md`) are confirmed committed to the store, its success branch (`{{^plan_incomplete}}`, lines 17-27 of the template) renders a short message listing the three document paths and a closing instruction: "Inform the user that the plan workflow is finished and the three documents are ready for review." (line 26). Nothing today offers the user an alternative to reading the documents themselves.

The plan workflow is driven, for Claude Code specifically, by `.claude/skills/spek-plan/SKILL.md`, whose current framing (lines 20-22) ends at "Keep looping... until a returned instruction tells you the workflow is *finished*. Only then should you report completion to the user." — with no acknowledgment that "report completion" might, after this feature ships, also mean presenting a walkthrough offer.

`internal/workflow/state.go:25-27`'s `InProgress()` hardcodes the literal string `"finished"` as the terminal step for all three workflow kinds (spec, plan, implement): `return s.CurrentStep != "" && s.CurrentStep != "finished"`. This is the key constraint that rules out adding any new FSM step after `finished` — doing so would make `InProgress()` report `true` for a plan sitting in a new terminal state, indistinguishable from a plan genuinely stuck mid-workflow, breaking the resume-detection contract shared across all three workflow kinds.

`plan file write <name>/<doc>.md --from <source-path>` (backed by `cmd/plan_file.go`) is a standalone CLI command, not gated by FSM state — it is already used mid-workflow (e.g. the `finished` template's own `plan_incomplete` recovery path, lines 7-9) and is the sole supported mechanism for committing any of the three plan documents. This is the exact mechanism mid-walkthrough edits reuse.

A directly relevant precedent already exists in this codebase: spec 000032 (spec-workflow-pair-programming-enhancements) solved a structurally similar problem — an agent proactively offering something during a conversation, honoring accept/defer/decline, with the offer/response loop as pure agent behavior rather than CLI state — entirely through instruction prose in an existing file (`AGENTS.md`), with zero FSM or workflow-state changes. That spec's own Non-Goals section explicitly named this feature's exact scope ("Adding a user-acceptance walkthrough to the plan workflow's verification step") as a separately identified gap, deferred to its own spec — this one.

## Per-Phase Technical Notes

### Phase 1.1: Add the walkthrough offer and conversational instructions to the `finished` step

**File changes**:
- `templates/steps/plan/18-finished.md:26` — Replace the single closing sentence ("Inform the user that the plan workflow is finished and the three documents are ready for review.") within the `{{^plan_incomplete}}` branch (lines 17-27) with expanded instruction text covering:
  1. The offer: present a two-way choice — read the documents directly (today's behavior, lines 20-25, unchanged) vs. a conversational walkthrough — framed once, not repeated if declined.
  2. If accepted, pacing guidance: a small number of natural beats (approach & reasoning → stage/phase breakdown → scope boundaries/exclusions) with brief pauses between them — not a rigid per-phase sign-off gate, not one uninterrupted monologue.
  3. Interruption handling: respond directly to any question or change request raised at any point, before returning to the beat in progress.
  4. Change handling: apply an accepted change immediately — stage the edited section under `.spektacular/tmp/`, run `{{config.command}} plan file write {{plan_name}}/plan.md --from <scratch>` (same command already shown at line 7 of this same template for the `plan_incomplete` recovery path), confirm the update to the user, remove the scratch file, then resume the walkthrough, re-summarizing anything the change affected.
  5. Closing gate: ask directly whether the plan is right and the user is ready to proceed; treat only an explicit affirmative as agreement — not silence, not a topic change, not an ambiguous reply.
  No mustache variables beyond the ones already rendered (`plan_dir`, `plan_path`, `context_path`, `research_path`, `plan_name`, `config.command`) are needed — confirmed during discovery that `finished()` (`internal/steps/plan/steps.go:277-299`) already supplies all of these via the existing `writeStep`/`stepkit.WriteStepResult` pipeline (`steps.go:64-76`).
- No other file in `internal/steps/plan` or `internal/workflow` changes — `finished()`'s Go logic, `Steps()`'s FSM wiring (`steps.go:28-48`), and `state.go`'s `InProgress()` (`state.go:25-27`) are all unchanged, by design (see plan.md § Architecture and research.md § Alternatives considered and rejected).

**Complexity**: Low

**Token estimate**: ~2k tokens (prose-only edit to one existing template file, ~10-15 new lines of instruction text)

**Agent strategy**: Single agent, sequential execution.

### Phase 1.2: Align the `spek-plan` skill's completion guidance with the new offer

**File changes**:
- `.claude/skills/spek-plan/SKILL.md:20-22` — Add one or two sentences near the existing "Keep looping... until a returned instruction tells you the workflow is *finished*. Only then should you report completion to the user." guidance, noting that the `finished` instruction may now include an offer to walk through the plan, and that presenting that offer (and, if accepted, conducting the walkthrough per Phase 1.1's instructions) is part of finishing the workflow — not something to summarize past or skip. No change to any other section of the skill file (the loop mechanics, the plan-file read/write rules, the working-files section, or the resume flow all stay exactly as they are).

**Complexity**: Low

**Token estimate**: ~1k tokens (one to two sentences added to an existing skill file)

**Agent strategy**: Single agent, sequential execution. This phase should run after Phase 1.1 so the skill's added sentence can reference the actual offer behavior just written, rather than being drafted against a guess of what Phase 1.1 will say.

## Testing Strategy

No automated tests are added by either phase — both are prose edits to Markdown files (a workflow-step template and a Claude-Code skill file), with no Go code, function, or data contract for a unit or integration test to exercise. This mirrors spec 000032's identical instruction-prose feature, which added zero automated tests for the same reason.

Verification for both phases is a single combined manual smoke test, run after both phases land: execute the plan workflow end-to-end (`plan new` → `plan goto` through every step → `finished`) in a live agent session and confirm each of the acceptance criteria listed under Phase 1.1 and Phase 1.2 in plan.md. This single pass covers both phases' changes together since Phase 1.2's skill-level change only affects how the agent driving the workflow behaves once it receives Phase 1.1's extended `finished` instruction — the two are not independently testable in isolation from each other's effect on real agent behavior.

## Project References

- Spec 000020 (context): established `.spektacular/context.md` as the git-tracked cross-cutting working-context sidecar; unaffected by this plan.
- Spec 000022 (spek-knowledge skill): established the propose-then-confirm interaction pattern this plan's offer/accept/apply-change flow follows.
- Spec 000023 (context-over-memory): established the managed-instruction-section pattern (`installMemoryContextSection`) that spec 000032 cloned; referenced here only as background for why "instruction prose as the entire behavioral surface" is an established pattern in this codebase, not itself a dependency of this plan's file changes.
- Spec/plan 000032 (spec-workflow-pair-programming-enhancements): direct precedent. Its Non-Goals named this exact gap; its plan.md and research.md are the structural template this plan's Architecture, Testing Approach, and Milestones sections followed.
- Spec/plan 000034 (spec-plan-implement-reconciliation): read for context during discovery — a related but distinct plan-workflow enhancement (spec-coverage verification, not conversational walkthrough). No overlap; both extend the plan workflow's tail end but address different concerns and can coexist in `18-finished.md`/adjacent steps without conflict.

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|--------------|----------------|
| Low | ~2k | Single agent, sequential |

Both phases in this plan are Low complexity and use single-agent sequential execution; no Medium/High-tier work exists in this plan.

## Migration Notes

No migration required. The feature is additive and backward-compatible: existing behavior (reading the plan documents directly) is fully preserved, and the new offer/walkthrough is available to any user of the plan workflow the moment these two file changes land — no re-running `spektacular init`, no config change, no data migration.

## Performance Considerations

No performance impact. Both changes are static prose additions to Markdown files already read/rendered by the existing workflow and skill mechanisms; no additional file reads, network calls, or computation are introduced.
