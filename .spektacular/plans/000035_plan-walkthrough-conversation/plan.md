# Plan: 000035_plan-walkthrough-conversation

<!-- Metadata -->
<!-- Created: 2026-07-10T09:05:57Z -->
<!-- Commit: ca60a2b11d2cf6424f2f294b7725813ff41db72d -->
<!-- Branch: f-conversational -->
<!-- Repository: git@github.com:jumppad-labs/spektacular.git -->

## Overview

Today, once the plan workflow finishes generating `plan.md`, `context.md`, and `research.md`, the assistant simply reports the documents are ready — the user is left to read dense documents alone to catch anything they'd want changed. This plan adds a conversational walkthrough option: after a plan finishes, the assistant offers the user a choice between reading the documents directly (unchanged, today's path) or having the assistant present the plan the way a colleague would explain it aloud — the chosen approach and reasoning, how the work breaks into stages, and what was deliberately left out — in a small number of natural beats rather than a document recitation. The user can interrupt at any point with a question or a change request, which the assistant applies immediately to the same plan document the rest of the workflow reads and writes, confirms, and then resumes from. The walkthrough only ends once the user explicitly agrees the plan is right. This lets users fully understand and shape a plan through conversation, rather than needing to read documents alone to catch something they'd want changed.

## Conventions

No project conventions apply to this feature.

The always-applied knowledge loaded during discovery contained only the category definitions for conventions and glossary (`.spektacular/knowledge/conventions/README.md` and `glossary/README.md`), not actual project-specific conventions. This feature touches only a workflow-step template (`templates/steps/plan/18-finished.md`) and a Claude-Code skill file (`.claude/skills/spek-plan/SKILL.md`) as prose edits — no coding standards, naming schemes, or required code patterns from the knowledge base bear on either surface.

## Architecture & Design Decisions

The walkthrough is delivered entirely as instruction prose appended to the plan workflow's existing terminal step, with a small consistency edit to the driving skill file — no Go code, no new FSM state, no new CLI command. `templates/steps/plan/18-finished.md`'s success branch (`{{^plan_incomplete}}`, lines 17-27) already renders once all three plan documents are confirmed committed to the store; its closing instruction ("Inform the user that the plan workflow is finished...", line 26) is extended with the full offer-and-walkthrough behavior: offer the user a choice between reading the documents directly (unchanged, existing behavior) or a conversational walkthrough; if accepted, present the plan as a small number of natural beats — approach & reasoning, stage/phase breakdown, scope boundaries (what's deliberately excluded) — pausing between beats rather than delivering one monologue; handle any mid-walkthrough question or change request inline before continuing; and close only once the user gives explicit affirmative agreement to a direct question, never assuming agreement from silence or a topic change. `.claude/skills/spek-plan/SKILL.md` gets one small addition near its existing "report completion" framing (lines 20-22) so the skill's own words don't imply the agent should stop short of presenting the offer it just received in the `finished` instruction.

This beats adding a new terminal FSM step (e.g. `discuss_plan`) because `internal/workflow/state.go:25-27`'s `InProgress()` hardcodes the literal string `"finished"` as the terminal step for all three workflow kinds (spec, plan, implement) — a step appended after `finished` would make a plan sitting in that new state look indistinguishable from one genuinely stuck mid-workflow to the resume-detection logic every workflow kind shares, requiring a kind-aware rework of shared code disproportionate to what is otherwise a purely conversational feature. It beats a skill-only instruction (with no template change) because `SKILL.md` is a Claude-Code-specific artifact; the template is the one surface every agent driving this CLI's JSON protocol reads, so putting the offer only in the skill file would make it invisible outside Claude Code specifically.

Mid-walkthrough changes are applied through the exact same mechanism the workflow itself already uses to commit the plan: stage the edited section under `.spektacular/tmp/`, run `{{config.command}} plan file write <name>/plan.md --from <scratch>` (the same standalone, FSM-independent command already documented in `SKILL.md`'s "Reading and writing plan files" section and used by the `finished` template's own recovery path), confirm the update to the user, remove the scratch file, and resume the walkthrough re-summarizing anything that changed as a result. There is no second write path to design or maintain — this satisfies the spec's constraint that walkthrough edits land in "the same plan document the rest of the workflow reads and writes," by construction rather than by added enforcement.

This design was independently validated by a sibling feature: spec 000032 (spec-workflow-pair-programming-enhancements)'s own Non-Goals section names this exact gap ("Adding a user-acceptance walkthrough to the plan workflow's verification step — a separately identified gap, not addressed here") and that plan's own architecture — offering something conversationally, honoring accept/defer/decline, and folding all of that into existing instruction-prose surfaces with zero FSM or workflow-state changes — is the direct structural precedent this plan follows (see [research.md § Chosen approach — evidence, Evidence 3](./research.md#chosen-approach--evidence)). The offer-once behavior needs no persisted state either: `finished()` (`internal/steps/plan/steps.go:277-299`) fires exactly once per plan run — nothing transitions back into it — so there is no code path where a re-offer could occur that would need guarding against (see [research.md § Alternatives considered and rejected](./research.md#alternatives-considered-and-rejected)).

## Component Breakdown

- **`finished` step instruction (changed)** — The plan workflow's terminal step template. Gains the entire behavioral contract for this feature: offering the user a choice between reading the plan directly or a conversational walkthrough, and — if accepted — describing how to structure the walkthrough (approach & reasoning, stage breakdown, scope boundaries), how to handle mid-walkthrough questions and change requests, and how to close only on explicit user agreement. Owns the full instruction surface; no other component carries any of this behavior.

- **Plan document write path (unchanged, reused)** — The existing mechanism for committing plan documents to the store. The walkthrough instruction directs the agent to reuse this same path for any mid-walkthrough edit, rather than introducing a separate edit mechanism. Its responsibility (validating and persisting a plan document) does not change; it simply gets invoked again, on demand, during the walkthrough instead of only during the workflow's write steps.

- **`spek-plan` skill instructions (changed)** — The skill file that drives an agent through the plan workflow's step-by-step loop. Gains a small addition near its existing "workflow finished, report completion" guidance, so the skill's framing acknowledges that the `finished` step's instruction may now include an offer to walk through the plan, and that presenting that offer is part of finishing the workflow, not something to skip past. Its existing responsibilities (looping through steps, reading/writing plan documents, resume handling) are unchanged.

- **Plan workflow FSM (unchanged, referenced)** — The state machine driving the plan workflow's steps. This feature adds no new step, event, or transition to it; the terminal `finished` state and its single-fire behavior are unchanged and relied upon as-is.

## Data Structures & Interfaces

No new data structures or interfaces are introduced. This feature adds no Go types, no new `workflow.Data` keys, and no new template variables — the `finished` step's existing render pipeline already supplies every variable the extended instruction needs (`plan_dir`, `plan_path`, `context_path`, `research_path`, `plan_name`, `config.command`), and the walkthrough's behavior (offer, beats, mid-walkthrough edits, closing agreement) is expressed entirely as instruction prose interpreted by the agent, not as a data contract between components.

The one existing contract this feature relies on, unchanged, is the `plan file write <name>/<doc>.md --from <source-path>` CLI command already used elsewhere in the workflow — mid-walkthrough edits invoke it exactly as-is, with no new flags, arguments, or output shape.

## Implementation Detail

This plan introduces no new code-shape pattern — it extends the codebase's existing, already-proven split between what the CLI's state machine owns (step sequencing, document validation, file writes) and what agent-facing instruction prose owns (judgment calls, conversational behavior, natural-language interpretation of "ask the user," "offer a choice," "wait for agreement"). That split is not new to this feature — it is the same split the memory-redirect instruction (spec 000023) and the spec-trigger-recognition instruction (spec 000032) already rely on. This plan's only genuinely new content is more prose in that same category, landing in an existing template rather than a new one.

**Where new prose is added.** `templates/steps/plan/18-finished.md`'s success branch gains additional instruction text after its current closing line. The new text is structured as: (1) the offer itself — read as a simple two-way choice, framed once, not repeated if declined; (2) if accepted, guidance on pacing the walkthrough as a small number of natural beats with brief pauses between them (approach & reasoning, then stage/phase breakdown, then scope boundaries) rather than either a rigid per-phase sign-off gate or one uninterrupted monologue — deliberately loose guidance, not a numbered protocol, since the goal is a natural conversational cadence a real colleague would use, not a scripted sequence; (3) how to handle an interruption at any point — respond to the question or change directly before returning to the beat in progress; (4) how to handle a change request specifically — apply it immediately (edit the working copy, commit via the existing write path, confirm to the user), then resume, re-summarizing anything that changed as a consequence, rather than deferring all edits to the end; (5) the closing gate — ask directly whether the plan is right and the user is ready to proceed, and treat only an explicit affirmative as agreement, never silence or a change of subject.

**Code-structure UX.** A developer reading `templates/steps/plan/18-finished.md` after this change sees one more block of prose appended to the existing success-branch text — same file, same template variables, same rendering path, no new conditionals or mustache sections beyond what's already there (`{{#plan_incomplete}}`/`{{^plan_incomplete}}`). A developer reading `.claude/skills/spek-plan/SKILL.md` sees one additional sentence or short paragraph near the existing "workflow is finished, report completion" guidance, acknowledging that finishing now includes presenting whatever offer the `finished` instruction contains. Neither file gains new structure (headings, sections, template blocks) — both are edits within their existing shape. No Go file changes at all; `go build`/`go vet`/existing Go tests are unaffected because no `.go` file is touched.

**Pattern precedent being followed, not introduced.** The "propose an offer, honor accept/defer-equivalent/decline, apply the accepted action inline, confirm before continuing" shape mirrors the propose-then-confirm pattern already established for the `spek-knowledge` skill's writes (spec 000022) and reused by spec 000032's spec-trigger-recognition instruction. This plan does not invent a new interaction pattern for agents to learn — it applies an existing one to a new moment in the plan workflow's lifecycle.

## Dependencies

- **`templates/steps/plan/18-finished.md` (existing, changed)** — The template being extended. No structural change to its rendering path; the plan adds prose within its existing `{{^plan_incomplete}}` branch.

- **`.claude/skills/spek-plan/SKILL.md` (existing, changed)** — The Claude-Code skill that drives the plan workflow loop. Gains a small consistency addition; no change to its existing loop/resume/file-access instructions.

- **`internal/steps/plan` package (existing, unchanged)** — Provides the `finished()` callback and the FSM step list. This plan depends on it staying exactly as-is: no new step, no new `Extra` key, no signature change. Confirmed during discovery that the existing render pipeline (`writeStep`/`stepkit.WriteStepResult`) already supplies every template variable the extended instruction needs.

- **`internal/workflow` package (existing, unchanged)** — Provides the FSM and `InProgress()`'s resume-detection logic. This plan depends on it staying untouched — the chosen design deliberately avoids the alternative that would have required changing `state.go`'s hardcoded `"finished"` terminal-step check.

- **`plan file write` CLI command (existing, unchanged, reused)** — Standalone command already used elsewhere in the workflow to commit plan documents to the store. Mid-walkthrough edits reuse it exactly as-is; no new flag, argument, or behavior is required of it.

- **Spec 000032 (spec-workflow-pair-programming-enhancements) — planning precedent only, no code dependency.** Already shipped; its plan/research documents were used as the structural template for this plan's own architecture decision (instruction-prose-only, zero FSM changes) and its Non-Goals explicitly named this feature as a deferred, separate gap. Nothing from its implementation is imported or required by this plan — the dependency is purely on the pattern it validated, not on any code it introduced.

No external libraries, no new Go dependencies, and no prior plan must land before this one starts — all referenced prior work (specs 000020, 000022, 000023, 000032) is already shipped, and this plan touches no code path any of them modified.

## Testing Approach

**No automated tests are added by this plan.** The feature is entirely a change to instruction prose in two Markdown files (`templates/steps/plan/18-finished.md` and `.claude/skills/spek-plan/SKILL.md`) — no Go code, no new function, no new branch, no new data contract exists for a unit or integration test to exercise. This mirrors the precedent set by spec 000032's structurally identical instruction-prose feature, which likewise added zero automated tests and relied entirely on manual verification for its behavioral guarantees, for the same reason: the thing being verified is whether an agent, reading prose, behaves as instructed — not a deterministic code path.

**Manual smoke tests are the sole verification mechanism.** After this plan's phases land, verify by actually running the plan workflow end to end (`plan new` → `plan goto` through every step → `finished`) and confirming, in a live session:

- The `finished` step's response, once all three documents are committed, includes an offer between reading the documents directly and a conversational walkthrough — and that this offer appears exactly once (declining it and re-reaching the same terminal state, e.g. via a resumed session re-rendering `finished`, does not need to suppress a second offer per this plan's scoping, but the first, natural pass through `finished` must show it).
- Accepting the offer produces a walkthrough that explicitly covers three distinct things before reaching a close — the reasoning behind the chosen approach, the ordered stage/phase breakdown, and what was deliberately left out — as distinct conversational points, not the plan document's section text recited verbatim.
- Interrupting the walkthrough with a question gets answered directly in the very next response, before the walkthrough continues to unrelated material.
- Requesting a change during the walkthrough results in the plan document on disk being updated (verifiable via `plan file read <name>/plan.md`) and the response confirming that update, before the walkthrough resumes.
- The walkthrough does not conclude or imply the plan is settled until the user gives an explicit affirmative answer to a direct closing question — a topic change, silence, or an ambiguous reply must not be treated as agreement.
- Declining the initial offer (choosing to read the documents directly instead) completes the workflow exactly as it does today, with no missing step and no degraded output.

**Load-bearing assertions the manual tests guarantee:**
- The offer appears exactly once per completed plan and does not block the existing "read the documents yourself" path.
- A walkthrough, once accepted, is a narrative covering approach/reasoning, stages, and exclusions — not a read-aloud of plan.md's headings.
- Interruptions and change requests get a direct, immediate response, and applied changes land in the same plan document the rest of the workflow reads and writes (`plan.md`, verified via `plan file read`) — never a separate copy or transient state.
- The walkthrough's end is gated on an explicit affirmative, never inferred.

### Success metrics

The specification's Success Metrics section states: "No success metrics have been defined for this feature." There are no metrics to carry forward into a verification mapping. The specification's Acceptance Criteria (five items, covering exactly-once offer, narrative structure, direct interruption handling, immediate change application with confirmation, and explicit-agreement closing) serve as the de facto verification targets in place of success metrics, and are each addressed by the manual smoke tests above.

## Milestones & Phases

### Milestone 1: After a plan finishes generating, the assistant offers — and can conduct — a conversational walkthrough of it

**What changes**: Once a plan workflow finishes generating all three documents (plan.md, context.md, research.md), the assistant's next message no longer simply reports the documents are ready — it also offers the user a choice: read the documents directly, or have the assistant walk through the plan conversationally. This offer appears exactly once; declining it changes nothing about today's behavior. Accepting it starts a narrative presentation, structured as a small number of natural conversational beats rather than a document read-aloud — covering why the chosen approach was picked, how the work breaks into stages, and what was deliberately left out — with brief pauses between beats rather than one uninterrupted monologue. At any point during the walkthrough, the user can interrupt with a question or a request to change something; the assistant responds directly before continuing. A requested change is applied to the plan document immediately (the same document the rest of the workflow reads and writes, not a separate copy), confirmed to the user, and the walkthrough resumes, re-summarizing anything the change affected. The walkthrough only concludes once the user explicitly confirms the plan is right and they're ready to proceed — never inferred from silence or a change of subject.

#### - [x] Phase 1.1: Add the walkthrough offer and conversational instructions to the `finished` step

Extend the plan workflow's terminal step so that, once all three plan documents are confirmed committed to the store, the assistant's instruction includes offering the user a choice between reading the documents directly or having the assistant walk through the plan conversationally. If accepted, the instruction guides the assistant to present the plan as a small number of natural conversational beats — approach and reasoning, stage breakdown, and scope boundaries — pausing between them rather than delivering an uninterrupted monologue, to respond directly to any interruption or change request before continuing, to apply an accepted change to the plan document immediately (through the same mechanism the workflow already uses to write it) and confirm the update before resuming, and to close the walkthrough only once the user gives explicit affirmative agreement. Declining the initial offer changes nothing about the workflow's existing behavior.

*Technical detail:* [context.md#phase-11](./context.md#phase-11-add-the-walkthrough-offer-and-conversational-instructions-to-the-finished-step)

**Acceptance criteria**:

- [x] After a plan finishes generating, the assistant's response offers a choice between reading the plan directly and a conversational walkthrough, and this offer does not repeat if declined
- [x] Declining the offer completes the plan workflow exactly as it does today, with no missing step
- [x] Accepting the offer produces a walkthrough that explicitly addresses the chosen approach and reasoning, the ordered stage breakdown, and what was deliberately excluded, as distinct points rather than a recitation of plan.md's section text
- [x] A question or change request raised at any point during the walkthrough receives a direct response in the assistant's next message, before the walkthrough continues to unrelated material
- [x] A requested change is reflected in the plan document on disk, with a confirmation message, before the walkthrough resumes
- [x] The walkthrough does not end or imply the plan is settled until the user gives an explicit affirmative response to a direct closing question

#### - [x] Phase 1.2: Align the `spek-plan` skill's completion guidance with the new offer

Update the skill instructions that drive an agent through the plan workflow so that its existing "the workflow is finished, report completion" guidance also acknowledges that finishing may include presenting the walkthrough offer introduced in Phase 1.1 — ensuring an agent reading the skill's framing doesn't report completion and stop before actually presenting that offer to the user.

*Technical detail:* [context.md#phase-12](./context.md#phase-12-align-the-spek-plan-skills-completion-guidance-with-the-new-offer)

**Acceptance criteria**:

- [x] The skill's completion guidance reflects that presenting the walkthrough offer (when the `finished` instruction includes one) is part of finishing the workflow, not a separate optional step that can be skipped
- [x] No other part of the skill's existing loop, resume, or file-access guidance changes

## Open Questions

No open questions remain. Every design decision needed to implement this feature was resolved during planning:

- **Integration point**: Resolved to `templates/steps/plan/18-finished.md`'s success branch (line 26), not a new FSM step and not a skill-only instruction — confirmed directly against the codebase (`internal/workflow/state.go:25-27`, `internal/steps/plan/steps.go:28-48, 277-299`), not left for implementation to discover.
- **Mid-walkthrough change mechanism**: Resolved to reusing the existing `plan file write <name>/plan.md --from <scratch>` command as-is — no new write path to design.
- **Offer-once enforcement**: Resolved to needing no persisted state — the `finished` step fires exactly once per plan run, so there's no code path where a re-offer could occur.
- **Milestone/phase structure**: Resolved to one milestone, two sequential phases (finished-step instruction content; SKILL.md consistency edit).
- **Conventions**: Resolved — none apply (knowledge base has no populated project-specific conventions yet).
- **Testing**: Resolved — no automated tests (no code path exists to test), manual smoke tests covering the specification's acceptance criteria instead (the specification defines no Success Metrics).

Because the entire feature is a prose edit to two already-read, already-understood Markdown files with no new code, no new data flow, and no ambiguity about what the instruction text should say, there is no genuine unknown left that can only be resolved once implementation begins. If the implementer discovers that either file's actual current content has drifted from what research.md/this plan describes (e.g. `18-finished.md` or `SKILL.md` has been edited by other work landing first), STOP and re-read the current file before proceeding, rather than applying the phase's instructions against a stale assumption of the file's shape.

## Out of Scope

- **Reciting the plan document's sections or content verbatim** — the walkthrough is a narrative explanation, not a read-aloud of `plan.md`'s headings or text (spec § Non-Goals).

- **Deferring all requested changes to the end of the walkthrough** — changes are handled inline, immediately, as they come up, not batched and applied at the close (spec § Non-Goals).

- **Cross-session or cross-plan memory of prior walkthroughs** — whether the user accepted or declined a walkthrough for a previous plan has no bearing on whether the offer appears for a future plan; the offer-once behavior applies only within a single plan's completion (spec § Non-Goals).

- **A new terminal FSM step for the walkthrough (e.g. `discuss_plan`)** — considered and rejected during the architecture step in favor of extending the existing `finished` step's instruction; adding a new step would require making `internal/workflow/state.go`'s `InProgress()` kind-aware instead of comparing against the literal string `"finished"`, a shared-code change affecting spec/implement resume detection too, disproportionate to a purely conversational feature (research.md § Alternatives considered and rejected).

- **A skill-only instruction with no change to the CLI's rendered template** — considered and rejected because `.claude/skills/spek-plan/SKILL.md` is Claude-Code-specific; putting the entire behavior there would make it invisible to any other agent surface driving this CLI's JSON protocol (research.md § Alternatives considered and rejected).

- **A persisted `workflow.Data` flag to track "already offered"** — considered and rejected as unnecessary: the `finished` step fires exactly once per plan run, so there is no code path where a re-offer could occur that would need guarding against (research.md § Alternatives considered and rejected).

- **Changing anything about the "read the documents directly" path** — that existing behavior (lines 20-25 of `templates/steps/plan/18-finished.md`) is preserved exactly as-is; this plan only adds an additional offered path alongside it, per the spec's constraint that reading the plan directly "must remain a fully valid path with no missing step."

- **Extending this walkthrough pattern to the spec or implement workflows' own terminal steps** — this plan scopes the walkthrough offer to the plan workflow's `finished` step only, per the spec's Overview ("After Spektacular finishes generating an implementation plan..."). Applying a similar pattern elsewhere would be a separate, future plan.

## Changelog

### 2026-07-10 — Phase 1.1: Add the walkthrough offer and conversational instructions to the `finished` step

**What was done**: Extended `templates/steps/plan/18-finished.md`'s success branch (`{{^plan_incomplete}}`) with the full walkthrough instruction: an offer (made once, not repeated on decline) between reading the documents directly or a conversational walkthrough; if accepted, guidance to present the plan as three paced beats (approach & reasoning, stage breakdown, scope boundaries) rather than a monologue or a document recitation; direct handling of any mid-walkthrough question or change request; inline application of accepted changes via the existing `plan file write <name>/plan.md --from <scratch>` command, with confirmation before resuming; and a closing gate requiring explicit affirmative agreement before the walkthrough concludes.

**Deviations**: None. The implementation matches context.md's Phase 1.1 technical notes exactly — same file, same insertion point (after the former line 26), same five-part structure, no new template variables or mustache sections.

**Files changed**:
- `templates/steps/plan/18-finished.md`

**Discoveries**: None beyond what research.md already established. The existing `internal/steps/plan/steps_test.go` FSM-order tests don't assert on this template's rendered prose, so the change is free of any test-content coupling to update.

### 2026-07-10 — Phase 1.2: Align the `spek-plan` skill's completion guidance with the new offer

**What was done**: Added one sentence to `.claude/skills/spek-plan/SKILL.md`'s existing "report completion" line (formerly line 20) clarifying that reporting completion includes presenting whatever offer the `finished` instruction contains — e.g. the Phase 1.1 walkthrough offer — and conducting the walkthrough if the user accepts, rather than summarizing the instruction and stopping short.

**Deviations**: None.

**Files changed**:
- `.claude/skills/spek-plan/SKILL.md`

**Discoveries**: None. No other part of the skill file's loop/resume/file-access guidance needed to change, as anticipated.
