---
created_date: "2026-07-31"
status: completed
closed_date: "2026-07-31"
---

# Research: 000040_plan-draft-then-walkthrough

## Alternatives considered and rejected

- **Keep the walkthrough inside the terminal `finished` step (prose-only change, no FSM edit).** Extend `templates/steps/plan/18-finished.md` to make the walkthrough mandatory and strip the confirmation gates from steps 02–12. Rejected because `finished` is terminal: `internal/workflow/state.go:25` defines `InProgress()` as `current_step != "" && current_step != "finished"`, so the moment the workflow enters `finished`, (a) an interruption mid-walkthrough is unresumable — `resumeOrClear` (`cmd/resume.go:123`) treats the state as stale and deletes it — violating the spec's "interrupted drafting can resume" requirement for the walkthrough segment; and (b) `finished()` (`internal/steps/plan/steps.go:287-324`) has already stamped all three documents `status: completed` via `metadata.Close`, so an un-signed-off plan is indistinguishable from an approved one, violating "no sign-off, no approval".

- **Agent-file overlay in the spec-trigger style (templates untouched).** Model on `templates/agents/spec-trigger.md:31`, which overlays "answer from the conversation instead of asking cold" onto unchanged spec steps. Rejected: the overlay pattern still requires per-step confirmation ("The user's confirmation or correction is still the final word"), which is exactly what this feature removes; leaving "Get the user's explicit agreement" prose in the templates while an overlay says "don't ask" creates contradictory instructions on the surface every non-Claude agent reads; and the spec's Non-Goals rule out keeping the old flow as an alternative, so there is nothing for the untouched templates to serve.

- **A new post-`finished` review step.** Already considered and rejected by prior plan 000035 (`plan file read 000035_plan-walkthrough-conversation/plan.md`, § Out of Scope): a step after `finished` breaks the shared `InProgress()` terminal-step check for all three workflow kinds. Inserting a step *before* `finished` (the chosen approach) has none of that cost — steps before `finished` are the FSM's normal shape.

- **Gate implement readiness with a new Go-side approval flag or new frontmatter field.** Rejected as unnecessary: the shared `.spektacular/state.json` already blocks `implement new` while a plan workflow is in progress (`cmd/resume.go:123-151` reports `cross_kind_workflow_in_progress`), and the existing `status` frontmatter (`internal/metadata/metadata.go:22-30`) already distinguishes `in-progress` from `completed` — `finished()` is the only place `StatusCompleted` is stamped. Moving sign-off before `finished` makes both existing mechanisms express approval with no new field, satisfying the "documents unchanged for implement" constraint.

- **Record assumptions only in `.spektacular/context.md` (the sidecar).** Rejected: the sidecar holds cross-cutting learnings and user answers, is overwritten freely, and is explicitly "not a copy of content already captured elsewhere" (`templates/context_directive_test.go:15`); assumptions need a first-class, per-plan durable record that the assemble step can fold into the final documents. The working-file mechanism (`.spektacular/work/<name>/`, git-tracked, agent-`Write`-tool territory, zero Go involvement) is the established home for exactly this.

## Chosen approach — evidence

The chosen direction (confirmed in the architecture step): strip the per-section confirmation gates from the gathering templates, record judgement calls in a new `assumptions.md` working file, fold them into the plan documents at assemble, and insert a new `walkthrough` FSM step between `write_research` and `finished` that owns the mandatory guided review and explicit sign-off.

- **The ten confirmation gates to remove are all template prose, no Go.** Enumerated exactly: `03-architecture.md:16-21,42,46`, `02-discovery.md:43`, `04-components.md:15,19`, `05-data_structures.md:15,19`, `06-implementation_detail.md:27,31`, `07-dependencies.md:15,19`, `08-testing_approach.md:36,40`, `09-milestones.md:15,20`, `10-phases.md:35,46`, `11-open_questions.md:7,16,29`, `12-out_of_scope.md:18,22`. Each also uses the phrase "the **agreed** X content" in its save-to-working-file paragraph, which becomes "drafted". Steps 01, 13–17 already run without user gates.
- **The walkthrough behavior to promote already exists in full.** `templates/steps/plan/18-finished.md:18-36`: offer (`:26`), three-beat narrative pacing (`:28-32`), interrupt-and-apply-immediately via stage→`plan file write`→`rm` (`:34`), explicit-affirmative closing gate (`:36`). The new `walkthrough` step template lifts this shape, drops the "offer, once" framing (review becomes the mandatory primary mechanism), and adds a beat that walks the recorded assumptions.
- **Inserting a step is a known three-edit change.** Step order lives solely in the Go slice `internal/steps/plan/steps.go:31-53`; each callback is a one-liner calling `writeStep(stepName, nextStep, templatePath, …)` (`steps.go:67`, e.g. `:88-283`). Precedent: plan 000034 inserted `reconcile_spec` before `finished` in the implement workflow the same way (slice entry + retargeted `Src` + renumbered template).
- **Resume for the walkthrough segment comes free from the FSM.** `Goto` to the current step re-renders without transitioning (`internal/workflow/workflow.go:184-187`, `renderStep` `:243`; `TestGotoSameStepReRenders` `internal/workflow/workflow_test.go:87`), and `State.InProgress()` (`state.go:25`) keeps a workflow sitting on `walkthrough` resumable.
- **Approval gating comes free from existing mechanisms.** While state.json sits on `walkthrough`, `implement new` is refused (`cmd/resume.go:123-151`); plan documents committed by steps 15–17 carry default `status: in-progress` (`internal/metadata/merge.go:52`) until `finished()` closes them `completed` (`internal/steps/plan/steps.go:287-324`) — which now happens only after sign-off.
- **The assumptions record slots into the existing working-file + assemble machinery.** Working files are entirely template-driven — no Go reads `.spektacular/work/` (grep-confirmed by the architecture agent) — so `assumptions.md` needs only prose additions: each gathering step appends to it, and the assemble mapping table (`13-assemble.md:19-39`) folds it into `research.md`. `research.md` already has mandatory `## Open assumptions` and `## Alternatives considered and rejected` sections (`templates/scaffold/research.md`, checked by `14-verification.md:13,46`).
- **The blocking-question shape already exists in the implement workflow.** `templates/steps/implement/01-read_plan.md:73-79`: "If every remaining spec item has coverage … **proceed to Step 4 without interruption**", else STOP and present numbered options — the exact "interrupt only when genuinely blocked" conditional the drafting steps need.
- **`{{config.command}} plan file set-status` exists** (`cmd/storefile.go:265`) if any step needs a frontmatter-only status rewrite, though the chosen design doesn't require it.

## Files examined

- `internal/steps/plan/steps.go:31-53` — 19-step FSM slice `new→…→assemble→verification→write_plan→write_context→write_research→finished`; `:67` `writeStep` helper; `:164-185` `assemble()` injects scaffold extras; `:221-283` `planDocStillScaffold`/`docWarning`; `:287-324` `finished()` sets `plan_incomplete` or `metadata.Close(StatusCompleted)`.
- `internal/workflow/workflow.go:73-130` — FSM engine; callbacks registered `before_<event>` so a failing step vetoes the transition (`:123`); `:184-187` same-step Goto re-renders (resume); `:268` `commitTerminal`.
- `internal/workflow/state.go:14-25` — persisted state shape; `InProgress()` = `current_step != "" && != "finished"`; single `.spektacular/state.json` for all kinds (`cmd/root.go:209`).
- `cmd/resume.go:87-153` — `detectInProgress`, `resumeOrClear` (new-command prologue: force/stale/same-kind/cross-kind), `guardKind`.
- `internal/stepkit/stepkit.go:55-121` — render pipeline; var merge order standard→strategy→Extra; standard vars `step`/`title`/`next_step`/`config.command`.
- `internal/steps/plan/strategy.go:18` — template vars `plan_path`, `context_path`, `research_path`, `plan_dir`, `plan_name`, `spec_path`.
- `internal/metadata/metadata.go:22-41`, `merge.go:52`, `close.go:16` — frontmatter schema (`created_date`/`status`/`closed_date`), default `status: in-progress` on first write, `Close` stamps `completed`.
- `cmd/storefile.go:110-265` — shared `plan file` read/write/list/set-status; stage-from-`--from` contract; ID-prefix validation.
- `cmd/implement.go:119-122` — implement's only Go gate is `os.Stat` on plan.md; the real gate is `templates/steps/implement/01-read_plan.md` prose.
- `templates/steps/plan/01…18` — per-step instructions; ten user-confirmation gates (see Chosen approach); non-terminal steps all carry the context-directive footer; `17-write_research.md:22` performs `rm -rf .spektacular/work/{{plan_name}}`.
- `templates/steps/plan/18-finished.md` — `plan_incomplete` recovery branch (`:3-16`) and full walkthrough prose (`:18-36`).
- `templates/steps/plan/13-assemble.md:19-39` — working-file→scaffold-heading mapping table; STOP if a working file is missing.
- `templates/steps/plan/14-verification.md:13,46` — required-section checks incl. research.md's 7 sections.
- `templates/scaffold/{plan,context,research}.md` — document scaffolds (plan.md 11 sections, context.md 7, research.md 7).
- `templates/agents/spec-trigger.md:31` — the "answer from conversation, don't ask cold" overlay pattern (rejected as the vehicle here).
- `templates/agents/draft-presentation.md` — standing rule: drafts as plain text first, confirmation dialogs only after.
- `templates/steps/implement/01-read_plan.md:53-88` — STOP-and-pick-an-option gates; descoped-requirements marker + idempotent re-read (`:71`) — the pattern for not re-flagging recorded decisions.
- `templates/skills/workflows/spek-plan/SKILL.md` — the loop contract; working-files exception (`:26-38`); resume report handling (`:42-66`).
- `internal/steps/plan/steps_test.go` — `TestStepsOrderMatchesExpected` (`:102`), `TestFSMWalkFromNewToFinished` (`:131`), template-content assertions incl. `TestArchitectureStepContainsOptionsAndAgreementBeat` (`:55`), `finished()` close/skip tests (`:203-343`).
- `templates/work_files_test.go:23-131` — gathering steps 02–12 must reference the work dir; 01 must not; only `spec/08-verification.md` and `plan/17-write_research.md` may clean up; implement steps must not reference work files.
- `templates/context_directive_test.go:15-96` — every non-terminal step template must contain the marker "not a copy of content already captured elsewhere" exactly once; `*-finished.md` zero times; floor of 29 directive-bearing templates.
- `templates/skill_resume_test.go:22-54`, `templates/skill_list_command_test.go:15` — skills must be resume-aware and direct agents to CLI list commands.
- `tests/harbor/plan-workflow/` — E2E suite: `instruction.md` success criteria; `tests/test_plan_workflow.py` asserts canonical step order (`:599`), every rendered `next_step` valid (`:894`), docs committed only via `plan file write` (`:660`); Makefile target `harbor-test-plan` (`Makefile:73-78`).

## External references

- `github.com/looplab/fsm` — the FSM library; its `before_` vs `after_` callback semantics are why step callbacks can veto transitions (`workflow.go:115-123`). Mattered for confirming a new step is safe to insert.
- `github.com/cbroglie/mustache` — template engine; section flags (`{{#plan_incomplete}}`) are the only conditional mechanism available to step templates.

## Prior plans / specs consulted

- `plan file read 000035_plan-walkthrough-conversation/plan.md` — introduced the current `18-finished.md` walkthrough; documents the rejected post-`finished` FSM step (InProgress() hardcodes `"finished"`) and established the "instruction prose owns judgment, Go owns sequencing" split this plan continues.
- `plan file read 000034_spec-plan-implement-reconciliation/plan.md` — precedent for inserting a step before `finished` (implement's `reconcile_spec`): slice entry + retargeted `Src` + template renumber; also the descoped-marker pattern for durable, idempotent recorded decisions inside a store document.
- Spec 000040 (this feature's spec) — source of truth for scope; notable constraints: plan workflow only, final document shape/names/location unchanged, same agent-and-CLI mechanism.
- Commit `1237193` ("Update to add conversational plan walk through") — the sole prior interaction-model template change; zero Go edits. Commit `834adad` only added spec 000040 — no template work exists yet; implementation is entirely ahead.

## Open assumptions

- **Adding a `## Drafting assumptions`-style section to research.md (or folding assumptions into its existing `Open assumptions`/`Alternatives` sections) does not break the implement workflow**, whose structural gate (`01-read_plan.md`) checks plan.md's `##` sections, not research.md's. If implementation finds implement-side checks on research.md's section list, STOP and ask.
- **The cross-kind state.json block is an acceptable implementation of "implement does not treat the plan as ready"** — it blocks `implement new` while the plan workflow sits on `walkthrough`, but a user running `implement new --force` can still discard the in-progress review. If a harder gate is required, STOP and ask.
- **`.spektacular/work/<name>/` cleanup can stay in `17-write_research.md`** because assumptions are folded into research.md at assemble and the walkthrough reads committed documents via `plan file read` — it does not need the working files. If walkthrough-time revisions turn out to need the owning working files, cleanup must move to post-sign-off instead.
- **The harbor E2E suite's step-order and prompt-count assertions can be updated mechanically** (new canonical order with `walkthrough`; instruction.md success criteria) without restructuring the suite.

## Rehydration cues

- `go run . knowledge always-applied` — currently returns only the conventions/glossary category READMEs; no populated conventions bind this plan.
- `go run . spec file read 000040_plan-draft-then-walkthrough.md` — the spec being planned.
- `go run . plan file read 000035_plan-walkthrough-conversation/plan.md` and `…000034_spec-plan-implement-reconciliation/plan.md` — the two structural precedents.
- Re-read: `internal/steps/plan/steps.go`, `internal/workflow/state.go:25`, `cmd/resume.go:87-153`, `templates/steps/plan/18-finished.md`, `templates/steps/plan/13-assemble.md`, `templates/work_files_test.go`, `templates/context_directive_test.go`.
- Sibling working files in `.spektacular/work/000040_plan-draft-then-walkthrough/` hold each drafted section; `.spektacular/context.md` holds cross-cutting decisions and user answers.
