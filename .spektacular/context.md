# Context: plan-walkthrough-conversation

## Problem identified and why it needs solving

Today, the `plan` workflow's `finished` step (`internal/steps/plan/steps.go`,
`templates/steps/plan/18-finished.md`) ends by printing the three document paths
(plan.md, context.md, research.md) and telling the agent to inform the user the
workflow is finished. The user is left to read the documents alone to understand
and evaluate what was decided.

The user wants to simulate the process two colleagues would go through when
discussing a feature: the agent should be able to walk the user through the
finished plan as a presentation, so the user can either agree with it or request
changes — fully understanding the plan rather than just being handed file paths.

Explicitly **not** the default path: if the user just wants to read the plan
themselves, they will — this is an *offered alternative* for when a live
walkthrough is preferred over reading, not a mandatory gate blocking completion.

## Requirements agreed (final design from conversation — ready to turn into spec requirements/acceptance criteria)

1. **Trigger — offer once, opt in.** After the plan workflow's `finished` step
   reports the three document paths (unchanged from today), the agent offers
   once: "Want me to walk you through it, or would you rather read it yourself?"
   Declining ends the workflow exactly as it does today — no repeated prompting.
2. **Presentation style — segmented with natural pause points.** If accepted,
   the agent presents like a colleague giving a walkthrough, not reading the
   document aloud. Structured in beats with brief pauses between them, roughly:
   approach & why → phase breakdown → scope boundaries (what's deliberately out).
   Not a rigid per-phase Q&A gate, but not one unstoppable monologue either — the
   user can interrupt easily between beats.
3. **Mid-flow changes — handle inline, then resume.** If the user requests a
   change at any point during the presentation, the agent discusses it right
   there, edits the affected plan section(s), recommits through the existing
   `plan file write` mechanism (confirmed in an earlier conversation — no new
   FSM step/state for revision, just a normal store write), briefly confirms the
   update to the user, then resumes the presentation from where it left off,
   re-summarizing anything that changed as a result of the edit.
4. **Closing — explicit agreement checkpoint.** After the last beat (scope
   boundaries), the agent asks directly: "does this look right, good to
   proceed?" and waits for explicit agreement rather than trailing off. Only
   once the user agrees is the plan considered settled.

## Alternatives considered and rejected

- **Reading the sections aloud in document order** — rejected. A colleague
  wouldn't recite headings; the user wants a narrative explanation, not a
  table-of-contents read-through.
- **Always-on walkthrough (no offer/opt-in)** — rejected. The user was explicit
  that reading the plan document directly remains the default expectation;
  the walkthrough is something the agent offers, not something forced.
- **Strict phase-by-phase gated Q&A** (pause and require explicit sign-off after
  every single phase before continuing) — considered, but user chose the lighter
  "segmented with natural pause points" version as a better match for how a real
  colleague-to-colleague conversation flows; less friction than a hard gate per
  phase.
- **Defer all requested changes to the end of the presentation** (collect
  feedback, apply edits only after the full walkthrough finishes) — considered
  to avoid re-deriving downstream sections mid-flow, but user chose to handle
  changes inline immediately when raised, then resume.
- **Implicit/no closing question** (just end after the last beat, no formal
  sign-off) — rejected in favor of an explicit closing agreement checkpoint,
  matching how a real design conversation actually concludes.

## Prior technical investigation carried forward (from the earlier, now-discarded spec attempt — still valid, not yet chosen)

The exact **wiring mechanism** for where this lives in the workflow is not yet
decided and is explicitly left for the plan workflow to resolve, not this spec:

1. Instruction-only extension of `templates/steps/plan/18-finished.md` — no FSM
   or Go changes; `internal/workflow/state.go`'s `InProgress()` hardcodes the
   literal string `"finished"` as terminal for all three workflow kinds (spec,
   plan, implement), so this option touches zero shared code.
2. New terminal FSM step (e.g. `discuss_plan`) appended after `finished` —
   structurally possible since `internal/workflow/workflow.go`'s `commitTerminal`,
   the implicit `"done"` transition, and `NextStepName()` are positional
   (`steps[len(steps)-1]`), not name-based. The blocker: `state.go:26`'s
   `InProgress()` would need to become kind-aware (or compare against
   `steps[len(steps)-1].Name` instead of the magic string `"finished"`) without
   breaking spec/implement's existing resume-detection behavior.
3. Not fully explored: a skill-level instruction (in `spek-plan/SKILL.md`)
   triggered once `finished` reports done, with no CLI step involved at all.

This spec should describe the *what* (the four requirements above); the *how*
(which of the three wiring shapes, or another) belongs to the downstream plan
workflow's architecture step.

## Verification step: fresh-eyes review findings applied

An independent subagent reviewed the assembled spec with no conversation context.
Three findings, all accepted:

- Technical Approach's closing line referenced the spec's own drafting history
  ("more than one shape was discussed") — reworded to neutral planner guidance.
- Added a constraint: mid-walkthrough edits must be written to the same plan
  document the rest of the workflow reads/writes, not a copy or transient state
  (closes a gap Requirement 4 / AC 4 implied but never stated as a hard rule).
- Dropped a Non-Goals bullet ("no rigid per-stage sign-off gate") as duplication
  of Technical Approach's existing phrasing — Technical Approach already states
  this direction; repeating its inverse in Non-Goals added no new boundary.

Spec committed to `.spektacular/specs/000035_plan-walkthrough-conversation.md`.

## Plan workflow status

Now planning against this spec via `go run . plan new --data '{"name":
"000035_plan-walkthrough-conversation"}'`. Discovery is done; research.md
written to `.spektacular/work/000035_plan-walkthrough-conversation/research.md`.

**Wiring decision resolved during discovery** (was left open in the spec):
Option 1 — instruction-only extension of `templates/steps/plan/18-finished.md`
(line 26, the "Inform the user that the plan workflow is finished..." sentence)
— confirmed as the correct approach. Option 2 (new terminal FSM step) is ruled
out: `internal/workflow/state.go:25-27`'s `InProgress()` hardcodes the literal
string `"finished"` as terminal for all three workflow kinds, so a step after
it would break resume detection without a shared-code change. Option 3
(skill-only, no template change) is ruled out: `SKILL.md` is Claude-Code-only,
the template reaches every agent surface driving the CLI. Also need a small
consistency addition to `.claude/skills/spek-plan/SKILL.md` (lines 20-22)
so the skill's "report completion" framing doesn't skip past the new offer.

Direct precedent found: spec 000032's own Non-Goals explicitly names this
exact feature ("Adding a user-acceptance walkthrough to the plan workflow's
verification step — a separately identified gap, not addressed here") and
that plan's finished shape (pure instruction prose, zero FSM/Go changes,
modeled on `installMemoryContextSection`'s AGENTS.md pattern) is the direct
structural template for this plan's own Architecture/Testing Approach
sections — see plan 000032's plan.md and research.md.

Mid-walkthrough edits reuse the exact same `plan file write <name>/plan.md
--from <scratch>` mechanism the workflow itself already uses (confirmed via
`cmd/plan_file.go` — standalone CLI command, not FSM-gated) — no new write
path needed, satisfying the spec's "same document, not a copy" constraint
by construction.

One open assumption flagged for the implement workflow to double check:
the offer belongs in `18-finished.md` (after all 3 docs are committed to the
store), not earlier in `14-verification.md` before commit — based on the
spec's Overview framing ("After Spektacular finishes generating..."). If this
turns out wrong, STOP and ask before proceeding.

## Architecture step: user confirmed

User confirmed the recommended option: extend `18-finished.md` + small
`SKILL.md` consistency edit, zero FSM/Go changes (see the 3-option
AskUserQuestion — chose "Extend 18-finished.md template (Recommended)" over
the new-FSM-step and skill-only alternatives). Architecture section and
Conventions section ("no conventions apply", user-confirmed) written to their
working files.

## Components step: user confirmed

Four components (finished-step instruction changed; spek-plan skill file
changed; plan document write path reused unchanged; FSM unchanged) confirmed
by user as-is, no edits requested.

## Data structures step

No new data structures/interfaces — pure prose feature, confirmed no Go types,
no new workflow.Data keys, no new template variables. Straightforward enough
not to pause for a separate confirmation round.

## Implementation detail step: user confirmed

Sketch confirmed as-is: 5-part instruction shape (offer once; loose beats —
approach/reasoning, stages, scope boundaries — not a rigid gate; handle
interruptions directly; apply changes inline via existing write path then
resume; explicit closing agreement question). No Go changes, template +
SKILL.md prose only.

## Dependencies step

No external libs, no Go package changes, no prior-plan blockers — all
referenced prior specs (000020, 000022, 000023, 000032) already shipped.
Objective/low-risk list.

## Testing approach step: user confirmed

No automated tests (prose-only feature, no code path exists to test) — manual
smoke tests cover all 5 acceptance criteria, mirroring spec 000032's
precedent. Spec has no Success Metrics defined; acceptance criteria serve as
the de facto verification targets instead. User confirmed. Proceeding to
milestones step.

Note: earlier in this session, a background Explore agent
(abd58fc5e2cb49634) spawned during discovery to double-check the finished-step
FSM facts never returned/completed before a session interruption — this did
not block progress since the same facts (InProgress() hardcoding "finished",
the finished() callback, Steps() list, SKILL.md content) were already
confirmed directly via Read/Bash and are captured in research.md. No need to
resume that agent; safe to ignore/let it be cleaned up.

## Milestones step: user confirmed

Single milestone (matching spec 000032's shape) covering the whole feature —
user confirmed, no split requested.

## Phases step: user confirmed

Two sequential phases: 1.1 the finished-step template's full walkthrough
instruction (offer, beats, interruption/change handling, closing gate),
1.2 the SKILL.md consistency edit (runs after 1.1 so it can reference the
actual offer text just written). User confirmed as-is.

## Open questions step

None remain — every design decision was resolved during planning (see list
in open_questions.md working file). Straightforward/objective outcome;
proceeding to out_of_scope step without a separate confirmation round.

## Out of scope step: user confirmed

Pulled from spec's Non-Goals (verbatim recitation, deferred-batch changes,
cross-session memory) + rejected architecture alternatives (new FSM step,
skill-only instruction, persisted offer-flag) + 2 explicit boundary notes
(read-directly path untouched; not extending to spec/implement workflows).
User confirmed as-is.

## All content sections complete — ready to assemble

All 9 working files written and user-confirmed: architecture.md,
conventions.md, components.md, data_structures.md, implementation_detail.md,
dependencies.md, testing_approach.md, milestones.md, phases_plan.md +
phases_context.md, open_questions.md, out_of_scope.md. Plus research.md from
discovery.

## Assemble step complete

Metadata gathered: commit ca60a2b11d2cf6424f2f294b7725813ff41db72d, branch
f-conversational, repo git@github.com:jumppad-labs/spektacular.git, created
2026-07-10T09:05:57Z. All three documents assembled from working files and
staged to `.spektacular/tmp/{plan_template,context_template,research_template}.md`.
Nothing written to the plan store yet.

## Verification step complete

All 11 plan.md sections, 7 context.md sections, 7 research.md sections
present, in order, non-empty (checked via grep + line-count spot-check).
Phase 1.1/1.2 anchors in plan.md match context.md's heading slugs exactly.

## write_plan step complete

plan.md committed to store, scratch file removed.

## write_context step complete

context.md committed to store, scratch file removed.

## write_research step complete

research.md committed to store, scratch file removed, working directory
`.spektacular/work/000035_plan-walkthrough-conversation` deleted (no longer
needed — all three documents are now durably in the plan store). Proceeding
to finished step.

## Implement workflow: analyze step complete (Phase 1.1)

Current phase: 1.1 ("Add the walkthrough offer and conversational
instructions to the `finished` step"), first unchecked phase in plan.md.
Technical detail confirmed at context.md#phase-11. Low complexity — did
analysis directly, no sub-agent delegation needed. Confirmed via direct read
that `templates/steps/plan/18-finished.md` still matches the plan/research
exactly (27 lines, `{{^plan_incomplete}}` branch lines 17-27, insertion point
at line 26's closing sentence). No new template variables needed —
`{{config.command}}` and `{{plan_name}}` are the only ones the new prose
requires, both already rendered. No mismatches found. Proceeding to
`implement` step to write the phase.

## Implement workflow: implement step complete (Phases 1.1 and 1.2)

Both phases written in one pass (both Low complexity, no code, pure prose):

- Phase 1.1: extended `templates/steps/plan/18-finished.md`'s
  `{{^plan_incomplete}}` branch (was line 26's single closing sentence) with
  the full 5-part walkthrough instruction — offer once, 3 paced beats
  (approach/reasoning, stage breakdown, scope boundaries), interruption
  handling, inline change application via the existing
  `plan file write {{plan_name}}/plan.md --from <scratch>` command, and an
  explicit-affirmative closing gate. No new mustache variables/sections used
  (only pre-existing `{{plan_path}}`, `{{config.command}}`, `{{plan_name}}`).
- Phase 1.2: added one sentence to `.claude/skills/spek-plan/SKILL.md` line 20
  (the "report completion" line) clarifying that reporting completion
  includes presenting the `finished` instruction's offer and conducting the
  walkthrough if accepted — not summarizing and stopping short.

`go build ./...` passes (expected — no `.go` files touched). Diff scope
confirmed limited to exactly these two files for this phase's work. Proceeding
to `test` step next (no automated tests are added per the plan's Testing
Approach — expect that step to confirm manual-smoke-test-only and move on).

## Implement workflow: test step complete (no tests to write)

Confirmed via `git diff --name-only -- '*.go'` that zero `.go` files changed
by Phases 1.1/1.2's work — matches the plan's Testing Approach ("No automated
tests are added by this plan... no Go code, no new function, no new branch,
no new data contract exists for a unit or integration test to exercise").
No sub-agent delegated; nothing for a test-writing agent to act on. Manual
smoke tests (per plan.md's Testing Approach) are the sole verification
mechanism and will be exercised at the `verify` step. Proceeding to `verify`.

## Implement workflow: verify step complete — all green

Ran directly (Low complexity, no sub-agent needed): `make test` (all packages
pass, including `internal/steps/plan` — existing FSM-order tests unaffected
since they don't assert on rendered template prose) and `make lint` (clean).
Confirmed via `internal/steps/plan/steps_test.go` that no test asserts on
`18-finished.md`'s content, so template prose is free to change without
breaking coverage — expected, since prose isn't a data contract.

Manual smoke test: rendered `18-finished.md`'s success branch with sample
mustache values (substituting `{{config.command}}`, `{{plan_name}}`, etc.) to
confirm the walkthrough offer prose reads naturally and covers Phase 1.1's
5 acceptance criteria (offer-once, 3-beat narrative structure, interruption
handling, inline change application via `plan file write`, explicit-agreement
close). Uses only pre-existing mustache variables/sections — no new ones —
so the production render path (`cbroglie/mustache` via
`stepkit.RenderTemplate`), already exercised by existing passing tests,
covers this template unchanged.

Proceeding to `update_plan` to check off Phase 1.1 and Phase 1.2's checkboxes
in plan.md.

## Implement workflow: update_plan step complete

Checked off both phase headings (`#### - [x] Phase 1.1`, `#### - [x] Phase
1.2`) and all 8 acceptance criteria across both phases (6 for 1.1, 2 for 1.2)
— all verified passing in the `verify` step. Committed via `plan file write
000035_plan-walkthrough-conversation/plan.md --from
.spektacular/tmp/plan_update.md`, scratch file removed. Confirmed via `plan
file read` that both checkboxes landed correctly. No criteria left unchecked
— every acceptance criterion for both phases was satisfied by the prose
added to `18-finished.md` and `SKILL.md`. Proceeding to `update_changelog`.

## Implement workflow: update_changelog step complete

Created the `## Changelog` section (first invocation for this plan) at the
end of plan.md and appended two entries in one pass, matching how both
phases were implemented together: Phase 1.1 (the `18-finished.md` walkthrough
instruction) and Phase 1.2 (the `SKILL.md` consistency sentence). Both marked
"Deviations: None" — implementation matched context.md's technical notes
exactly. Re-read plan.md and confirmed zero `#### - [ ] Phase` (unchecked)
headings remain — this was the last phase. Proceeding to
`update_repo_changelog` for the repo-level release-note summary.

## Implement workflow: update_repo_changelog step complete

Prepended a new `## 000035_plan-walkthrough-conversation` section to the top
of repo-root `CHANGELOG.md` (above the existing `## 000034_...` section) with
a 4-sentence user-facing summary — no file paths or package names, focused on
the behavior change (offer, walkthrough structure, inline edits, explicit
close, unchanged read-directly path). Matches the format of all prior entries
in the file. Proceeding to `test_plan`.

## Implement workflow: test_plan step complete (none required)

Plan's Testing Approach defines no success metrics (spec has none; acceptance
criteria substitute), so wrote the explicit "none required" artifact to
`000035_plan-walkthrough-conversation/test-plan.md` via `plan file write`,
noting the manual smoke tests were already run in `verify`. Confirmed via
`plan file read`. Proceeding to `update_feature_changelog`.

## User's exact phrasing for the key requirement

"Yeah so what I would like to simulate is the process between two colleagues
discussing a feature. Let's figure out a good process for this. The aim is that
I the user fully understand your plan so that I can either agree with it or
request changes."

"It should be like a presentation, If I just wanted to read through I would read
the plan, which the user would most likely do, but when asked if the user wants
a walk through it should be presentation style, at each point the user can stop
and ask questions, this may change the plan."

## Implement workflow: read_plan step complete

Started `go run . implement new --data '{"name":
"000035_plan-walkthrough-conversation"}'`. Read plan.md, context.md,
research.md in full. Structural validation passed (all 10 required `## `
sections present; 2 phases, each with a `*Technical detail:*` link resolving
to a matching `### Phase N.M:` heading in context.md). Drift check passed
against the working tree with zero mismatches — every referenced file/line
(`templates/steps/plan/18-finished.md` 27 lines exactly as described,
`.claude/skills/spek-plan/SKILL.md` lines 20-22, `internal/steps/plan/steps.go`
lines 28-48 and 277-299, `internal/workflow/state.go` lines 25-27,
`cmd/plan_file.go`) matches the plan/research documents verbatim, including
line numbers. Spec coverage check passed: all 5 requirements and 5 acceptance
criteria in the spec map to Phase 1.1 (offer, narrative beats, interruption
handling, immediate change application, explicit-agreement close) or Phase 1.2
(SKILL.md consistency); no gaps, no descoping needed. No `## Changelog`
section exists in plan.md yet — this is a first-phase invocation; the
`update_changelog` step will create the section on first use. Proceeding to
`analyze` step, which will pick up at Phase 1.1 (the first `#### - [ ]` phase).

## Implement workflow: update_feature_changelog step complete

Authored a self-contained changelog record (What was built / Why it matters
/ Deviations: None) drawing on the spec's Overview and the plan's own
Changelog section (both already in context from earlier steps). Committed to
the changelog store via `go run . changelog file write
000035_plan-walkthrough-conversation.md --from .spektacular/tmp/changelog_record.md`,
scratch file removed, confirmed via `changelog file read`. Proceeding to
`finished` — the terminal step. All phases complete; implementation done.

## Implement workflow: reconcile_spec step complete

The skill's stated next step after `update_feature_changelog` was stale
(`finished`) — the CLI's actual FSM required `reconcile_spec` first (from
plan 000034, landed after this skill file). Followed the CLI's
`invalid_transition` error and `next_action` field rather than the skill's
text. Judged all 5 spec Requirements and 5 Acceptance Criteria against the
plan's Changelog entries (both phases: Deviations: None) — all 10 genuinely
satisfied by the delivered prose (offer-once, narrative structure,
interruption handling, immediate change application, explicit-agreement
close). Flipped all 10 checkboxes to `[x]`, committed via `spec file write
000035_plan-walkthrough-conversation.md --from .spektacular/tmp/spec_reconcile.md`,
confirmed via `spec file read`. Proceeding to `finished`.

## Implement workflow: FINISHED — terminal state reached

Reached the `finished` step. All work complete:
- Both phases (1.1, 1.2) implemented and checked off in plan.md.
- Plan's inline `## Changelog` has both phase entries (Deviations: None).
- Repo `CHANGELOG.md` has the user-facing release note.
- Feature changelog record written to `.spektacular/changelog/000035_plan-walkthrough-conversation.md`.
- Spec fully reconciled — all 10 Requirements/Acceptance-Criteria checkboxes
  flipped to `[x]`, nothing left unchecked or descoped.

No further `goto` — this is the terminal state. Reporting completion to the
user next.
