# Plan: 000034_spec-plan-implement-reconciliation

<!-- Metadata -->
<!-- Created: 2026-07-09T06:25:25Z -->
<!-- Commit: 5b502a8 -->
<!-- Branch: f-conversational -->
<!-- Repository: git@github.com:jumppad-labs/spektacular.git -->

## Overview

Today, nothing checks a plan against the specification it was created from, or the completed work against that specification once implementation finishes — a plan can silently drift from the spec's actual goal (as happened with spec 000032's Milestone 2), and a spec's own Requirements and Acceptance Criteria checkboxes remain unchecked forever even after the linked work is done. This plan closes that gap by adding two guardrails to the existing `implement` workflow: a pre-implementation check that confirms the plan covers every spec requirement and acceptance criterion before implementation starts, and a post-implementation step that checks off satisfied spec items and folds the specification's completion status into the workflow's existing final report. Users get confidence that what gets built matches what was asked for, and any gap is surfaced at the moment it matters instead of being discovered later by manual review.

## Conventions

No project conventions apply to this feature. `go run . knowledge always-applied` returned only the conventions and glossary category README placeholders (no populated entries exist in the knowledge base yet), and `go run . knowledge search` for step/template-related terms during discovery surfaced nothing binding.

## Architecture & Design Decisions

This feature adds two guardrails to the existing `plan` and `implement` finite-state-machine workflows, without introducing any new standalone workflow, command, or FSM engine mechanism. Both guardrails are implemented as extensions to the existing step machinery documented in `internal/workflow/`, `internal/stepkit/`, and the per-workflow `steps.go`/`strategy.go` pairs — no new abstractions are needed because the FSM already supports everything both guardrails require: conditional template-driven branching via multi-source edges, per-step arbitrary Go-computed template variables via `Extra`, and store-backed document read/write via `spec file`/`plan file` CLI commands.

**Pre-implementation coverage check.** This lands solely in the `implement` workflow's existing `read_plan` step (`internal/steps/implement/steps.go:19`, template `templates/steps/implement/01-read_plan.md`), which is already documented as "the validation and drift gate for the implement workflow — nothing else runs until it passes." The step is extended with a new check: read the spec via `spec file read {{plan_name}}.md` (already available with zero new plumbing, since `spec_path`/`plan_name` are already computed unconditionally by `implement`'s `strategy.PathVars`, `internal/steps/implement/strategy.go:47-63`), enumerate every `- [ ]`/`- [x]` checkbox under the spec's `## Requirements` and `## Acceptance Criteria` headings, and confirm each has corresponding coverage somewhere in the plan's `## Milestones & Phases` section. This deliberately does not also land in the `plan` workflow's `verification` step — a plan-creation-time-only check would miss the exact failure mode that motivated this spec: a plan silently hand-edited after the plan workflow already finished (as happened to spec 000032's Milestone 2). Checking once, at the moment implementation is about to start against whatever the plan currently says, covers both a freshly-generated plan and a hand-edited one with a single check, which is simpler than maintaining two copies of the same comparison logic in two workflows for a single-check guarantee. A gap found here is not a hard failure: the step's existing STOP-and-multi-option-prompt convention (already used for structural/drift mismatches at `01-read_plan.md:53-59`) is extended with a two-option variant — update the plan to add the missing coverage, or explicitly accept the gap as an intentional descope — consistent with the spec's non-goal ruling out any auto-fix or hard block.

**Recording an accepted gap so it is not re-flagged.** When the user chooses to accept a gap as descoped, that decision is written directly into the plan document itself (a short note near the relevant requirement, or a dedicated "Descoped" list appended to the plan) rather than into `state.json`'s `data` field. This follows the codebase's established pattern that plans, specs, and changelogs — not opaque workflow state — are the durable, human-visible record of decisions; `state.json`'s `data` field is used elsewhere in the codebase only for transient per-run values (e.g. the plan/spec name), not standing decisions that must survive a workflow restart or a fresh `implement new` run against the same plan later. On each `read_plan` invocation, the coverage check treats a requirement already marked descoped in the plan as resolved and does not re-prompt for it.

**Post-implementation reconciliation.** This is a new FSM step, `reconcile_spec`, inserted into the `implement` workflow between the existing `update_feature_changelog` and `finished` steps (`Src: []string{"update_feature_changelog"}` for the new step; `finished`'s `Src` is retargeted from `["update_feature_changelog"]` to `["reconcile_spec"]`). It is deliberately a separate step rather than folded into `update_feature_changelog`, even though both steps read the same source material (the spec and the plan's accumulated `## Changelog` phase-by-phase record). `update_feature_changelog` already has a fully scoped job — author a self-contained changelog-store record — with its own STOP-on-mismatch condition (spec or changelog section missing). Mixing in a second, semantically different task (judge each spec checkbox against the implementation record, then write back to the spec store) would conflate two different failure classes and two different artifacts being written in one step's instructions, making the step harder to reason about and its STOP conditions harder to specify precisely. `reconcile_spec`'s template follows the same structural shape as `update_feature_changelog`'s (`templates/steps/implement/10-update_feature_changelog.md`): gather source material (re-read the spec and plan's changelog section fresh via the CLI, per the codebase's established habit of never carrying document state across steps), judge each checkbox, stage an updated spec to `.spektacular/tmp/` and commit it via the already-existing `spec file write` command (`cmd/file.go:8-11`, the same generic mechanism backing `plan file`/`changelog file` — no new CLI surface needed), and a STOP-on-mismatch section for a missing spec or empty changelog record. The `finished` step's terminal report is then extended to read the just-rewritten spec file and report its checkbox state — satisfied items and, for anything left unchecked, the stated reason (deferred, descoped, or not attempted) — alongside the existing changelog summary. No new data-threading between `reconcile_spec` and `finished` is required: the spec file itself, freshly rewritten, is the record `finished` reads back, mirroring how `finished` already re-reads `plan_path`/`changelog_path` rather than receiving them as passed-through state.

Both checkbox judgments (pre-implementation coverage matching and post-implementation satisfaction) are agent-driven prose instructions in the step templates, not mechanical Go-side parsing. No Go code in this codebase parses markdown checkboxes today, and the judgment involved — does this phase's implementation record actually satisfy this specific acceptance criterion's wording — is inherently semantic, consistent with how `update_feature_changelog` already delegates comparable judgment (summarizing deviations from plan) to the agent rather than to deterministic code.

See `research.md#alternatives-considered-and-rejected` for the rejected alternatives (a standalone reconciliation workflow, a Go-level hard-fail gate, a new Go-level branching primitive, running the pre-check in two places, and folding the post-check into the `finished` Go callback) and their citations.

## Component Breakdown

- **`read_plan` step (extended)** — the implement workflow's existing validation-and-drift gate. Gains a new responsibility: reading the originating spec and confirming every Requirements/Acceptance-Criteria checkbox has corresponding coverage in the plan's milestones and phases. Owns presenting a coverage gap to the user and capturing their resolution (update the plan, or accept the gap as an intentional descope). No new component is created — this is an extension of an existing gate's checklist, alongside its current structural-validation and drift-check responsibilities.

- **Descoped-requirement marker (new, lightweight convention within the plan document)** — a small addition to the plan document's own content, not a new file or store. When the user accepts a coverage gap as descoped, that decision is recorded directly in the plan so a later re-run of the coverage check recognizes it as already resolved and does not re-prompt. Owned and interpreted entirely by the `read_plan` step's instructions; no other component needs to know about it.

- **`reconcile_spec` step (new)** — a new step in the implement workflow, positioned after the feature changelog is written and before the workflow's final report. Owns comparing the completed implementation (via the plan's accumulated phase-by-phase changelog record) against the spec's Requirements and Acceptance Criteria, judging which are genuinely satisfied, and committing an updated spec document with satisfied items checked off. This is the single place that writes back to the spec store during implementation — no other step touches the spec file.

- **`finished` step (extended)** — the implement workflow's terminal reporting step. Gains a new responsibility: reading the spec document (as just updated by `reconcile_spec`) and folding its completion status — satisfied items, and any unsatisfied item together with why (deferred, descoped, or not attempted) — into the same completion report that already summarizes the changelog and phases completed. No new report or output location is introduced; this is additional content in the existing report.

- **Spec store (existing, reused)** — the existing spec read/write/list CLI surface. No changes to this component; `reconcile_spec` and the extended `read_plan` step are new *consumers* of its existing read and write operations, not something that needs new capability added to it.

- **Plan store (existing, reused)** — the existing plan read/write CLI surface, including the plan's accumulated changelog section. No changes to this component; it is read by both the extended `read_plan` step (for coverage matching) and the new `reconcile_spec` step (as the implementation record used to judge satisfaction).

## Data Structures & Interfaces

No new Go types or interfaces are introduced. This feature is expressed entirely through the existing `workflow.StepConfig` contract and markdown template content — the FSM engine, `Data`/`Extra` mechanisms, and `PathStrategy` interface are all reused unchanged.

**FSM registration change** (`internal/steps/implement/steps.go`): one new `workflow.StepConfig` entry is inserted into the `implement` workflow's step slice:

```go
{Name: "reconcile_spec", Src: []string{"update_feature_changelog"}, Dst: "reconcile_spec", Callback: reconcileSpec()},
```

and the existing terminal entry's `Src` is retargeted:

```go
{Name: "finished", Src: []string{"reconcile_spec"}, Dst: "finished", Callback: finished()},
```

`reconcileSpec()` follows the same one-liner shape as every other step callback in this file (e.g. `updateFeatureChangelog()`): it calls the package's `writeStep` helper with the new template path, no custom logic beyond that — matching the pattern used by every step in both `steps.go` files.

**Template variable contract** — no new variables need to be added to `strategy.PathVars` for either workflow. `spec_path` and `plan_name` are already computed unconditionally by `implement`'s `strategy.PathVars` (used by the extended `read_plan` and `finished` steps, and by the new `reconcile_spec` step); `changelog_section_name` is already available for reading the plan's accumulated implementation record.

**Descoped-requirement marker** — not a new type or schema, but a documented markdown convention within the plan document's existing free-form content, interpreted only by the `read_plan` step's own instructions (no other step or component parses it). Shape: a short list appended near the plan's `## Milestones & Phases` section (or a small inline note against the relevant requirement) recording which spec requirement/acceptance-criterion was accepted as descoped and why. The exact heading/format is decided during this plan's Implementation Detail step; it deliberately does not require a new `## ` top-level section in the plan-store scaffold, since it's an annotation, not a required document section.

**Spec checkbox format (read, not defined by this feature)** — the existing spec document convention, already authored by the `spec` workflow (`templates/steps/spec/02-requirements.md`, `03-acceptance_criteria.md`): `- [ ] **Bold title**` followed by an indented description paragraph, under `## Requirements` and `## Acceptance Criteria` headings. This feature's new steps read and rewrite these lines (flipping `[ ]` to `[x]`) but do not change or extend this format.

## Implementation Detail

This plan follows the codebase's existing step-authoring pattern exactly rather than introducing a new one: every step is a template-driven instruction to the agent, with the Go layer contributing only mechanical scaffolding (state transition, path variables) and never the judgment itself. The two guardrails fit into that pattern as extensions of existing gate/report steps plus one new step of the same shape as its neighbors — there is no new code-structure idiom for a developer to learn.

**Gate extension, not gate duplication.** `read_plan`'s existing structural-validation and drift-check checks are already presented to the agent as a numbered sequence of gate conditions that must all pass before the step advances. The spec-coverage check is added as a further numbered condition in that same sequence, reusing the step's existing "STOP and report, offer the user a choice" idiom rather than inventing a second, parallel gating mechanism. A developer reading `01-read_plan.md` after this change sees one more check in a list they already understand the shape of, not a structurally different kind of validation bolted alongside it.

**Branch-by-template-prose, not new Go control flow.** Both guardrails need conditional behavior (a fully-covered plan proceeds without interruption; a gap prompts the user and waits for a choice; an already-descoped gap is recognized and skipped). Consistent with every existing branch in this codebase (e.g. the phase-loop-or-advance choice in `update_changelog`), this is expressed entirely as prose in the step's template — "if X, do A; if Y, do B" — followed by the agent issuing the corresponding `goto` call itself. No new Go-level conditional, flag, or branching primitive is introduced anywhere in `internal/workflow/` or `internal/steps/`.

**One new step, shaped like its closest sibling.** `reconcile_spec` is structured to closely mirror `update_feature_changelog`'s existing four-part shape (gather source material → do the judgment/authoring work → stage-then-commit the artifact via the store CLI → STOP-on-mismatch), because that is the established shape for "read spec + plan changelog, produce something, write it back through a store command" steps in this codebase. A developer who already understands `update_feature_changelog` will recognize `reconcile_spec`'s structure immediately; the only genuinely new content is what gets judged (checkbox satisfaction) and where it gets written (the spec store instead of the changelog store).

**No new persistence or serialization layer.** The descoped-requirement marker is plain markdown content inside the plan document, written and read the same way every other part of the plan document already is (via `plan file read`/`plan file write`, staged through `.spektacular/tmp/`). It does not introduce a new file, a new store, or a new parsing layer — it is additional prose in a document the codebase already treats as free-form Markdown that only the agent interprets.

**Renumbering, not restructuring.** Inserting `reconcile_spec` before `finished` requires renumbering `templates/steps/implement/11-finished.md` to `12-finished.md`, exactly mirroring the precedent set when `update_feature_changelog` was added ahead of `finished` in the prior changelog feature (a `git mv`, not a content restructuring). This is a mechanical consequence of the numeric-prefix template-naming convention already in use, not a new pattern.

## Dependencies

- **`internal/workflow`** — the generic FSM engine (`StepConfig`, `StepCallback`, `Data`, state persistence). No changes needed; this plan only adds entries to the `[]workflow.StepConfig` slice returned by `implement.Steps()`, using the existing multi-source-edge and `Extra` mechanisms as-is.
- **`internal/stepkit`** — shared step machinery (`WriteStepResult`, `RenderTemplate`, `PathStrategy`). No changes needed; the new and extended steps use the existing `writeStep` wrapper pattern already used by every other step in `internal/steps/implement/steps.go`.
- **`internal/steps/implement` (package)** — the package being changed. Depends on its own `strategy.go` for `spec_path`/`plan_name`/`changelog_section_name`, all already present; no changes needed there.
- **`cmd` — `spec file` / `plan file` CLI commands** — both already exist (`cmd/file.go`, shared `newStoreFileCmd` builder) with read/write/list operations and the established stage-to-`.spektacular/tmp/`-then-commit contract. This plan is a pure consumer of both; no changes needed to either command.
- **Prior plan/spec on which this plan's approach is modeled**: `.spektacular/plans/000033_spec-workflow-output-changelog/` (the changelog feature). Its `update_feature_changelog`/`finished` step pair is the direct structural template for this plan's new `reconcile_spec` step and the `finished` renumbering. It has already landed — no dependency ordering issue, just a pattern to follow.
- **This feature's own spec**, `.spektacular/specs/000034_spec-plan-implement-reconciliation.md` — the plan is being written against it; no other spec must land first.
- **No new external (third-party) library dependencies.** `github.com/looplab/fsm` (FSM engine) and `github.com/cbroglie/mustache` (template rendering) are existing dependencies already used throughout `internal/workflow` and `internal/stepkit`; this plan does not add, upgrade, or otherwise touch either.
- **No dependency on the `spec` or `plan` workflow packages changing.** The pre-implementation coverage check lives entirely in `implement`'s `read_plan` step (per the locked architecture decision); the `plan` workflow package (`internal/steps/plan/`) is not touched by this plan at all.

## Testing Approach

This feature's core logic — FSM step registration, path/variable plumbing, and the terminal-report renumbering — is deterministic Go code and is covered by unit tests following this codebase's existing pattern (`internal/steps/implement/steps_test.go` already tests step wiring and template-instruction content via string assertions on rendered output, e.g. `TestReadPlanTemplateDirectsStructuralValidation`, `TestUpdateChangelogStepSpecifiesEntryFields`). The judgment-laden parts of both guardrails (deciding whether a plan's phases cover a spec requirement; deciding whether a phase's changelog entry satisfies an acceptance criterion) are agent-driven prose instructions, not code, and are not unit-testable — they are covered by asserting the *instructions themselves* direct the correct behavior, the same way existing STOP-on-mismatch and branching instructions are tested today.

**Unit tests (Go, `internal/steps/implement/steps_test.go`)**:
- FSM wiring: `reconcile_spec` is reachable only from `update_feature_changelog`; `finished` is reachable only from `reconcile_spec` (not directly from `update_feature_changelog` anymore) — a structural assertion on `Steps()`'s returned `[]workflow.StepConfig`, mirroring how existing tests assert `analyze`'s multi-source `Src`.
- Template rendering: the extended `read_plan` template's rendered instruction text mentions the spec-coverage check and both resolution options (update plan / accept as descoped) — string-assertion style, matching `TestReadPlanTemplateDirectsStructuralValidation`.
- Template rendering: the new `reconcile_spec` template's rendered instruction text directs reading the spec and plan changelog, judging checkbox satisfaction, and committing via `spec file write` — mirroring `TestUpdateChangelogStepSpecifiesEntryFields`.
- Template rendering: the extended `finished` template's rendered instruction text directs reporting spec completion status alongside the existing changelog summary.
- Path variables: `spec_path` continues to resolve correctly for both the extended `read_plan` and new `reconcile_spec` steps (regression coverage — this plumbing already exists and must not be broken by the new step's addition).

**Deliberate gap**: no integration test drives a full mocked `implement` run end-to-end through `reconcile_spec` with a fabricated spec/plan pair and asserts the spec file's checkboxes actually flip. This is because the checkbox-satisfaction judgment is agent-driven, not deterministic — a scripted integration test would have to hard-code what "satisfied" means for a fabricated example, which tests the test fixture more than the feature. The existing codebase does not integration-test other agent-judgment steps (e.g. `update_changelog`'s deviation summarization) this way either, for the same reason; unit tests on wiring and instruction content are the established ceiling for this class of step.

### Success Metrics → verification mapping

- **"A gap like 000032 is caught by the pre-implementation check before implementation proceeds"** — Manual, captured in the implementation test plan. This requires a real plan with a genuine coverage gap run through the actual `implement` workflow with an agent driving it, observing that the gap is surfaced and blocks silent progression — not something a scripted test can assert without itself embedding the judgment being tested.
- **"After an implement workflow completes, a user can tell which specification requirements and acceptance criteria are satisfied and which are not without opening the specification file themselves"** — Behavioural test (unit-level) for the mechanical half: assert the `finished` template's rendered instruction text explicitly directs reporting spec status in the completion report (i.e., the report *will* contain this content structurally). Manual, captured in the implementation test plan for the judgment half: that the reported status is actually correct for a real implementation run.
- **"Specification files for completed features accurately reflect what was built, rather than remaining entirely unchecked despite the linked work being finished"** — Behavioural test (unit-level) for the mechanical half: assert `reconcile_spec`'s template directs committing the updated spec via `spec file write` (i.e., the spec *will* be written back to, not left untouched). Manual, captured in the implementation test plan for the judgment half: that specific checkboxes end up correctly checked/unchecked for a real implementation run.

## Milestones & Phases

### Milestone 1: Implementation stops and asks before proceeding on a plan with spec coverage gaps

**What changes**: When `implement` starts against a plan, it now checks that every requirement and acceptance criterion in the originating specification has corresponding coverage somewhere in the plan's milestones and phases. If something is missing — whether the plan never covered it or someone edited the plan afterward and dropped it — the user is shown exactly what's missing and asked to either update the plan or explicitly accept the gap as descoped, before any implementation work begins. A plan that already covers everything proceeds exactly as it does today, with no extra interruption. Once a gap has been accepted as descoped, re-running or resuming `implement` against that plan does not ask about it again.

#### - [ ] Phase 1.1: Extend the implement workflow's read_plan gate with a spec coverage check

The implement workflow's existing validation step — the one that already checks the plan's structure and looks for drift against the codebase before any implementation work starts — gains a new check: it reads the specification the plan was created from and confirms every requirement and acceptance criterion has corresponding coverage in the plan's milestones and phases. If something is missing, the workflow stops and asks the user to either update the plan or explicitly accept the gap as an intentional descope, the same way it already stops and asks about other kinds of mismatches. A plan that already covers everything is unaffected — implementation proceeds exactly as before.

*Technical detail:* [context.md#phase-11](./context.md#phase-11-extend-read_plan-with-spec-coverage-check)

**Acceptance criteria**:
- [ ] Running `implement` against a plan whose milestones/phases do not cover one or more spec requirements stops before implementation begins and shows the user which requirements are missing.
- [ ] Running `implement` against a plan that covers every spec requirement proceeds without any extra interruption.
- [ ] The user is offered a choice between updating the plan and accepting the gap as descoped, and the workflow does not decide unilaterally.

#### - [ ] Phase 1.2: Make an accepted gap durable so it isn't re-flagged

When the user accepts a coverage gap as an intentional descope, that decision is recorded directly in the plan document itself, so that resuming or re-running `implement` against the same plan later recognizes the gap as already resolved instead of asking again.

*Technical detail:* [context.md#phase-12](./context.md#phase-12-durable-descope-marker)

**Acceptance criteria**:
- [ ] After a gap is accepted as descoped, resuming the implement workflow (or restarting it against the same plan) does not re-prompt about that same gap.
- [ ] The accepted-gap decision is visible in the plan document itself, not hidden in workflow state that a human reading the plan would never see.

### Milestone 2: Completion reports tell the user what the specification says is done

**What changes**: When `implement` finishes, the specification's own Requirements and Acceptance Criteria checkboxes are checked off for everything the completed work genuinely satisfies, and the specification file is updated to reflect that. The final report the user sees — the same report that already summarizes the changelog and completed phases — now also states plainly which specification items are satisfied and, for anything still open, why (deferred, descoped, or not yet done). Nobody has to separately open the specification file to find this out.

#### - [ ] Phase 2.1: Add the reconcile_spec step and wire it into the implement workflow

A new step is added to the implement workflow, running after the feature's changelog record is written and before the workflow's final report. This step reads the specification and the plan's accumulated phase-by-phase implementation record, judges which specification requirements and acceptance criteria are genuinely satisfied by what was built, and updates the specification file to check off the ones that are.

*Technical detail:* [context.md#phase-21](./context.md#phase-21-add-reconcile_spec-step)

**Acceptance criteria**:
- [ ] After `implement` completes, specification checklist items that the completed work satisfies are marked checked in the specification file.
- [ ] No specification checklist item is marked checked unless the completed work, as recorded in the plan's implementation history, actually satisfies it.
- [ ] The step correctly handles a specification that cannot be found or a plan with no recorded implementation history, by stopping and asking the user how to proceed rather than guessing.

#### - [ ] Phase 2.2: Report specification completion status in the final summary

The implement workflow's final report — the same report that already summarizes the changelog and the phases completed — is extended to also state which specification requirements and acceptance criteria are satisfied, and for anything still open, why (deferred, descoped, or not yet done). The user does not need to separately open the specification file to learn this.

*Technical detail:* [context.md#phase-22](./context.md#phase-22-extend-finished-report)

**Acceptance criteria**:
- [ ] The final report presented at the end of a completed implement run names both the specification items that are satisfied and any that remain unsatisfied, together with a reason for each unsatisfied item.
- [ ] This information appears in the same report as the existing changelog summary, without the user needing to open a separate file.

## Open Questions

None. Every design decision in this plan was resolved during discovery and the architecture step — the exact FSM insertion points, the existing template patterns to model the new content on, the `spec file`/`plan file` CLI commands to reuse, and the descoped-marker convention were all confirmed by direct inspection of the codebase (see `research.md`) rather than left uncertain. There is no aspect of this work that depends on information only discoverable by starting implementation.

## Out of Scope

- **Retroactively auditing or fixing already-completed specifications** (e.g. going back to reconcile spec 000032 itself) — per the spec's Non-Goals. This feature applies going forward only; a follow-up task to manually reconcile 000032 would need to be tracked separately if the user wants it done.
- **Enforcing coverage checks on the spec workflow itself** (e.g. checking a specification against whatever upstream discussion produced it) — per the spec's Non-Goals. This plan touches only the `implement` workflow's `read_plan` and terminal steps; the `spec` workflow package (`internal/steps/spec/`) is not modified.
- **Automatically fixing or writing plan content to close a coverage gap** — per the spec's Non-Goals. The pre-implementation check surfaces gaps and lets the user choose; it never generates or edits plan content on the user's behalf.
- **Hard-blocking or failing the workflow on a coverage gap** — per the spec's Non-Goals. A gap is surfaced and the user is asked; the workflow does not refuse to proceed or error out.
- **Adding the coverage check to the `plan` workflow's `verification` step** — decided during the architecture step (see `research.md`/architecture section above). The check lives solely in `implement`'s `read_plan` step, which catches both freshly-created and hand-edited plans at the single moment implementation is about to start; a second, duplicate check at plan-creation time was considered and rejected as unnecessary complexity for the same guarantee.
- **Folding `reconcile_spec` into the existing `update_feature_changelog` step** — decided during the architecture step. A dedicated new step was chosen instead, to keep the changelog-record-authoring and spec-checkbox-judgment failure modes and artifacts cleanly separated. Revisiting this merge is not part of this plan.
- **A new Go-level branching or state-passing primitive for the "update plan vs. accept gap as descoped" choice, or for threading the coverage-check outcome between steps** — the existing multi-source-FSM-edge-plus-template-prose pattern and plan-document-as-durable-record approach are used throughout instead; no new mechanism is introduced in `internal/workflow/`.
- **Fixing the `.spektacular/context.md`-not-cleared-on-`plan new`/`implement new` gap** noticed mid-planning-session — the user explicitly deferred this to a follow-up after this plan lands; it is unrelated to spec 000034's scope and is not addressed by this plan.
