---
created_date: "2026-07-31"
status: completed
closed_date: "2026-07-31"
---

# Plan: 000040_plan-draft-then-walkthrough

<!-- Metadata -->
<!-- Created: 2026-07-31T06:48:11Z -->
<!-- Commit: 834adad -->
<!-- Branch: f-improve-plan -->
<!-- Repository: git@github.com:jumppad-labs/spektacular.git -->

## Overview

The plan workflow stops asking the user to approve every section as it is gathered and instead drafts the entire plan autonomously, recording every judgement call it makes — including the choice of design direction — as an assumption with rationale and rejected alternatives, interrupting only for genuinely blocking questions. When the draft is complete, a new mandatory walkthrough step reviews the plan and its assumptions with the user section by section, applies requested changes immediately, and only marks the plan approved on an explicit affirmative sign-off. Anyone creating a plan gets a colleague-style propose-then-review experience instead of roughly ten serial approval gates, without any change to the documents the implement workflow consumes.

## Conventions

No project conventions apply to this feature. The always-applied knowledge loaded during discovery (`go run . knowledge always-applied`) returned only the conventions and glossary category README placeholders — the knowledge base holds no populated project-specific conventions — and targeted knowledge searches for the surfaces this feature touches (plan workflow, walkthrough, resume/working files) surfaced no binding entries.

## Architecture & Design Decisions

The plan workflow's interaction model changes from "confirm each section as it is gathered" to "draft everything autonomously, then review once, guided, at the end" — implemented almost entirely as instruction-prose edits to the existing step templates, plus one new FSM step. The ten confirmation gates in the gathering templates (`templates/steps/plan/02-discovery.md` through `12-out_of_scope.md`) are removed: each step now drafts its section from the spec and research, saves it to its working file, records any judgement call it made in a durable assumption log, and advances immediately. A step interrupts the user only for a genuinely blocking decision — mutually exclusive directions the agent cannot responsibly choose between, or information only the user holds — using the same "proceed without interruption unless blocked, else STOP and present options" conditional shape the implement workflow's `read_plan` gate already uses. The architecture step, today's strongest gate, becomes the clearest example of the new model: the agent picks the best-grounded direction itself and records it, with rationale and rejected alternatives, as a first-class assumption for review.

Assumptions get a durable record in the same spirit as the existing per-section working files: a new git-tracked working file, `.spektacular/work/<plan_name>/assumptions.md`, appended to by every drafting step as calls are made (decision, rationale, rejected alternatives). Because working files are entirely template-driven — no Go code reads or writes `.spektacular/work/` — this needs only prose additions to the gathering steps, and it survives session interruption exactly the way the section files do. At the assemble step, the accumulated assumptions are folded into `research.md`, so the committed documents carry the full record and the walkthrough can present it from the store rather than from scratch files.

The review itself moves out of the terminal `finished` step into a new `walkthrough` FSM step inserted between `write_research` and `finished` in `internal/steps/plan/steps.go`. The step's template lifts the walkthrough prose that already exists in `18-finished.md` (narrative beats, interrupt-and-apply-changes-immediately via the existing `plan file write` path, explicit affirmative sign-off) and promotes it from an optional offer to the mandatory primary review, adding a beat that walks the recorded assumptions. This placement is what makes two spec requirements fall out of existing machinery rather than new code: a session interrupted mid-walkthrough is resumable because the workflow state still sits on a non-terminal step (`InProgress()` in `internal/workflow/state.go` only treats `finished` as done, and a same-step `goto` re-renders the walkthrough instruction), and "no sign-off, no approval" holds because the shared `state.json` blocks `implement new` while the plan workflow is in progress, and the plan documents keep their default `status: in-progress` frontmatter until the post-sign-off `finished` step stamps them `completed` — the only place that ever does. No new status field, flag, or approval artifact is introduced, which keeps the final documents byte-compatible for the downstream implement workflow.

This direction beats the two rejected shapes — keeping the mandatory walkthrough inside the terminal `finished` step, and overlaying "don't ask" behavior from an agent file while leaving the templates' confirmation prose intact — because both fail spec requirements the chosen design satisfies structurally: the terminal-step variant cannot resume an interrupted review and cannot distinguish a reviewed plan from an unreviewed one, and the overlay variant leaves contradictory instructions on the one surface every agent reads. See `research.md#alternatives-considered-and-rejected` for the full evidence trail. The cost is a known, precedented one: a step-slice edit plus template renumber (the implement workflow's `reconcile_spec` insertion followed the identical recipe) and mechanical updates to the step-order, work-file, and context-directive test suites.

## Component Breakdown

- **Gathering step instructions (changed)** — the plan workflow's section-drafting steps, from discovery through out-of-scope. Each loses its "present for review and wait until the user is happy" gate and instead drafts its section autonomously from the spec and research, saves it to its working file, and advances. Each gains two shared behaviors: recording any judgement call it makes in the assumption log, and interrupting the user only when a decision is genuinely blocking (no reasonable default), phrased as a proceed-unless-blocked conditional. The architecture step changes most: instead of presenting options and waiting for agreement, it selects the best-grounded direction itself and records the choice, rationale, and rejected alternatives as a first-class assumption.

- **Assumption log (new, working-file convention)** — a durable, git-tracked record of every judgement call made during drafting: the decision taken, its rationale, and the alternatives rejected. Appended to by each gathering step as calls are made, read back on session resume alongside the section working files, folded into the research document at assembly, and walked through entry by entry during review. Owned and interpreted entirely by step instructions; no other component parses it.

- **Assemble step (changed)** — the existing step that maps working files onto the three document scaffolds. Gains one mapping: folding the accumulated assumption log into the research document, so the committed store documents carry the full assumption record and the walkthrough can present it from the store.

- **`walkthrough` step (new)** — a new workflow step between the last document-write step and the terminal step, owning the entire guided review: walking the user through the committed plan section by section with the recorded assumptions raised as a first-class beat, inviting comments as it goes, applying requested changes immediately through the existing document write path and re-summarizing what changed, and closing only on an explicit affirmative answer to a direct sign-off question. Silence or ambiguity keeps the step open; an interrupted session resumes here with the review intact. Promotes the review shape that previously lived, as an optional offer, in the terminal step.

- **Terminal `finished` step (changed)** — loses the walkthrough offer it carries today and becomes the post-sign-off closer: it stamps the plan documents completed (as it already does) and reports the workflow done. Reached only via the walkthrough's sign-off, which is what makes "no sign-off, no approval" structural. Its existing incomplete-documents recovery branch is unchanged.

- **Plan workflow step table (changed)** — the ordered step definitions gain the new walkthrough step and retarget the terminal step to follow it. A mechanical, precedented insertion; the workflow engine itself is untouched.

- **`spek-plan` skill instructions (changed)** — the agent-facing driver of the step loop. Reframed so the loop explicitly runs without stopping to check in between drafting steps, describes the assumption log as part of the working-file contract, covers the new walkthrough step in the loop and in resume guidance, and drops the now-stale framing of the walkthrough as an optional offer at the end.

- **Plan document write path (unchanged, reused)** — the existing store-write command remains the sole way plan documents are committed, and mid-walkthrough revisions reuse it exactly as the current terminal-step walkthrough does. No new write mechanism.

- **Resume machinery (unchanged, relied upon)** — the existing state persistence, in-progress detection, and same-step re-render are what make an interrupted draft or walkthrough resumable and what block the implement workflow from starting while review is open. Relied on as-is; no changes.

## Data Structures & Interfaces

No new Go types, interfaces, template variables, or serialization formats are introduced. The feature reuses every existing contract unchanged and adds one lightweight markdown convention.

**FSM step registration (changed contract instance, existing type)** — the plan workflow's step table gains one entry of the existing `workflow.StepConfig` shape, and the terminal entry's source is retargeted:

```go
{Name: "walkthrough", Src: []string{"write_research"}, Dst: "walkthrough", Callback: walkthrough()},
{Name: "finished",    Src: []string{"walkthrough"},    Dst: "finished",    Callback: finished()},
```

`walkthrough()` follows the same one-line `writeStep` callback shape as every sibling step. No change to `StepConfig`, the FSM engine, `Data`, `Extra`, or the render pipeline.

**Assumption log entry (new markdown convention, not a schema)** — `.spektacular/work/<plan_name>/assumptions.md` holds one entry per judgement call, in a fixed lightweight shape the step instructions define and only agents interpret:

```markdown
### <short decision title> (<step name>)
- **Decision**: what was chosen
- **Rationale**: why this was the reasonable default
- **Rejected**: alternatives considered and why not
```

No Go code parses this file; it is a working-file convention in the same category as the existing per-section files.

**Existing contracts relied on, unchanged**: the `plan file read/write <name>/<doc>.md [--from]` CLI surface (walkthrough revisions and reads), the persisted workflow state shape and its `InProgress()` semantics (resume and implement-blocking), the document frontmatter schema and its `status` lifecycle (`in-progress` until `finished` closes to `completed`), and the mustache template variables already supplied to every plan step (`plan_path`, `context_path`, `research_path`, `plan_dir`, `plan_name`, `spec_path`, `config.command`). The final plan documents keep their exact shape, names, and location.

## Implementation Detail

This feature extends the codebase's established split — the Go layer owns step sequencing, state persistence, and document validation; instruction prose owns judgement, conversation, and content — without introducing any new mechanism on either side. Everything user-visible about the change (autonomous drafting, assumption recording, the guided review) is expressed as template prose; the only Go change is one mechanical step insertion of the same shape as every existing step.

**De-gating is a rewrite of each gathering step's closing beat, not a restructure.** Each gathering template today ends with a present-for-review gate and a "once the user is happy, advance" instruction. Under the new model that closing beat becomes: save the drafted section, append any judgement calls to the assumption log, refresh the working context, advance. The body of each step — what the section must contain, its rules, its working file — is untouched, so a developer diffing the templates sees the interaction model change and nothing else. Wording that presumes agreement ("the agreed content") becomes drafting language. The architecture step keeps its options-analysis discipline but redirects its output: options are still enumerated and weighed against research, and the choice is recorded as the leading assumption-log entry instead of being put to the user.

**Blocking questions follow the proceed-unless-blocked conditional the implement workflow already uses.** Rather than inventing a new protocol, the drafting steps adopt the established STOP-shape: proceed without interruption by default; when a genuinely blocking decision arises — no reasonable default, or information only the user holds — stop, present the options in one block, and do not advance past the dependent point until answered. A shared, compact statement of this rule (and of assumption recording) is repeated in each gathering step in the same way the context-refresh directive already is: identical prose, per-template, enforceable by the template test suite.

**The walkthrough step is a promotion, not an invention.** Its template lifts the existing terminal-step walkthrough prose — narrative beats, interrupt-to-comment, apply-changes-immediately through the store write path, explicit affirmative closing gate — and reframes it from optional offer to mandatory review, adding one beat that walks the assumption log entry by entry and inviting challenge on each. Revisions during review edit the committed documents directly via the existing staged-write cycle, exactly as today's walkthrough does. The terminal step correspondingly shrinks: the offer prose moves out, the incomplete-documents recovery branch and the completed-status stamping stay.

**A developer reading the changed code sees familiar shapes everywhere.** The step table gains one entry identical in form to its neighbors; the new template is numbered into the existing sequence (renumbering the terminal template, as the implement workflow's reconcile-spec insertion did); the assumption log is one more working file in an existing directory contract; and the skill file's loop framing is reinforced rather than reshaped. Existing patterns followed: working files as durable per-section state, staged-write document commits, prose-driven conditional gates, template-content tests as the enforcement mechanism. New pattern introduced: only the assumption log's entry convention — deliberately a markdown convention interpreted by agents, not a parsed format.

## Dependencies

- **`internal/steps/plan` (existing, changed)** — the plan workflow's step table and callbacks; gains the `walkthrough` step entry and callback, retargets `finished`'s source. The only Go package modified.
- **`internal/workflow` (existing, unchanged, relied upon)** — provides the FSM engine, state persistence, `InProgress()` resume semantics, and same-step re-render; the design depends on all of these staying exactly as-is.
- **`internal/stepkit` (existing, unchanged)** — the shared render pipeline; the new step's template uses only variables it already supplies.
- **`internal/metadata` (existing, unchanged)** — frontmatter lifecycle; `finished`'s existing close-to-completed behavior is what expresses approval after sign-off.
- **`cmd` — `plan file` / resume guards (existing, unchanged, reused)** — the store write path used for mid-walkthrough revisions, and the cross-kind in-progress guard that keeps `implement new` blocked while review is open.
- **`templates/steps/plan/` (existing, changed)** — gathering templates 02–12 de-gated and given assumption-log/blocking-question prose; assemble step gains the assumption-fold mapping; a new walkthrough template is numbered in; the terminal template is renumbered and slimmed.
- **`templates/skills/workflows/spek-plan/SKILL.md` (existing, changed)** — loop framing, working-file contract, and resume guidance updated for autonomous drafting and the new step. Installed copies (e.g. `.claude/skills/spek-plan/`) refresh via `init` as usual.
- **Template and step test suites (existing, changed)** — `internal/steps/plan/steps_test.go` (step order, walk, template-content assertions), `templates/context_directive_test.go` (new non-terminal step must carry the directive), `templates/work_files_test.go` (assumption log joins the working-file contract), `tests/harbor/plan-workflow/` (canonical step order, success criteria, prompt-count expectations).
- **External libraries: none added or changed** — `looplab/fsm` and `cbroglie/mustache` are existing dependencies used as-is.
- **Prior plans/specs: none must land first** — the two structural precedents (000034's step insertion, 000035's walkthrough prose) are already shipped; this plan builds on the code they left behind, not on pending work.

## Testing Approach

The feature splits into a deterministic layer (FSM wiring, template content) covered by Go unit tests, and an agent-behavior layer (does drafting actually run without prompts, does the review actually revise) covered by the existing end-to-end harness plus manual checks — mirroring how every prior interaction-model change in this codebase was verified.

**Unit tests (Go, extending the existing step and template suites).** The step-table tests are updated for the new canonical order: the walkthrough step is reachable only from the last document-write step, the terminal step only from the walkthrough, and the full walk from new to finished passes through it. Template-content assertions — the codebase's established way of testing prose-driven behavior — are updated and extended: gathering steps no longer contain their wait-for-approval gate phrasing and do contain the assumption-recording and proceed-unless-blocked instructions; the walkthrough template contains the assumptions beat, the apply-changes-immediately store-write instruction, and the explicit-affirmative closing gate; the terminal template no longer carries the offer. The existing template contract suites are extended rather than duplicated: the context-directive test covers the new non-terminal step, and the working-files test admits the assumption log into the working-file contract (gathering steps reference it; only the established cleanup points remove the work directory).

**End-to-end (harbor plan-workflow suite, updated).** The suite's canonical step order, success criteria, and instruction are updated for the new flow. Its most load-bearing new assertions: a full run on a spec with no blocking decisions reaches the walkthrough with zero confirmation questions during drafting; the assumption log exists and is populated with decision/rationale/alternatives entries; the committed documents remain in-progress until sign-off and are completed after it.

**Load-bearing guarantees, in plain language:** drafting cannot silently reintroduce approval gates (template assertions fail); the walkthrough cannot be skipped on the way to finished (FSM wiring); an unreviewed plan is never marked completed (existing finished-step close tests plus the new wiring); and an interrupted session resumes on the walkthrough step (existing same-step re-render tests, relied on rather than re-proven — per project convention, no redundant second assertion for an already-covered behavior).

**Deliberate gaps:** no unit tests attempt to verify conversational quality (whether the walkthrough is a good narrative, whether an assumption's rationale is persuasive) — prose judgement is not unit-testable and is covered by the E2E run and manual review. No new tests for the store write path or resume machinery, which are unchanged and already covered.

**Success metrics from the spec, each made verifiable:**

- *Interruptions between starting a plan and the walkthrough drop to only genuinely blocking questions — zero for a typical plan* — **Behavioural test**: the end-to-end run on a blocking-decision-free spec asserts the workflow reaches the walkthrough with no user-directed question during drafting.
- *Plans are still meaningfully reviewed — comments produce visible plan revisions rather than a rubber stamp* — **Manual — captured in the implementation test plan** (requires a live session with a human commenting and inspecting the resulting document diff).
- *The user no longer reports being asked for approval section-by-section* — **Manual — captured in the implementation test plan** (user-experience observation over real use; the behavioural zero-prompt assertion is the automated proxy).
- *No regression in downstream quality — plans remain implementable without more clarification round-trips* — **Manual — captured in the implementation test plan** (observed across subsequent implement runs; not assertable at plan time).

## Milestones & Phases

### Milestone 1: Plan review becomes a dedicated, sign-off-gated workflow step

**What changes**: The guided walkthrough stops being an optional offer at the end of the plan workflow and becomes its own mandatory step, sitting between the documents being written and the workflow finishing. A user completing a plan is always walked through it section by section, can interrupt with comments that revise the saved documents immediately, and must give an explicit affirmative answer to a direct closing question before the plan is treated as done — silence or ambiguity keeps the review open. Because the workflow now sits on the review step until sign-off, an interrupted review can be resumed in a later session, the plan documents stay marked in-progress until sign-off, and the implement workflow cannot start against a plan whose review never closed. Drafting behavior is unchanged in this milestone; only the review and approval mechanics move.

**Validation point**: The step-table and template unit tests pass with the new canonical order; a live plan run reaches the walkthrough step, resumes on it after an interruption, refuses to finish without an explicit affirmative, and only then marks the documents completed.


#### - [x] Phase 1.1: Insert the walkthrough step into the plan workflow

Add a new walkthrough step between the last document-write step and the terminal step, and move the guided-review behavior into it. The new step's instructions carry the full review contract — section-by-section walkthrough of the committed documents, immediate application of requested changes through the store write path, and an explicit affirmative sign-off as the only way to advance to finished. The terminal step is slimmed to what remains its job: stamping the documents completed and reporting the workflow done. Step-order and wiring tests are updated to the new canonical sequence.

*Technical detail:* [context.md#phase-11](./context.md#phase-11-insert-the-walkthrough-step-into-the-plan-workflow)

**Acceptance criteria**:

- [x] The plan workflow's step sequence includes a walkthrough step immediately before finished, and finished can only be reached from it
- [x] The walkthrough instruction directs a section-by-section review of the committed documents, applies requested changes immediately via the existing document write command, and concludes only on an explicit affirmative answer to a direct closing question
- [x] The terminal step no longer offers or conducts a walkthrough; it stamps the documents completed and reports completion, with its incomplete-documents recovery branch intact
- [x] A plan interrupted during review resumes on the walkthrough step with the committed documents intact, and the documents remain marked in-progress until sign-off

#### - [x] Phase 1.2: Align the driving skill and end-to-end harness with the new step

Update the agent-facing skill instructions so the step loop covers the walkthrough (including resume guidance), and update the end-to-end harness's canonical step order and success criteria so the suite validates the new sequence. This keeps every surface that teaches or checks the workflow's shape consistent with Phase 1.1.

*Technical detail:* [context.md#phase-12](./context.md#phase-12-align-the-driving-skill-and-end-to-end-harness-with-the-new-step)

**Acceptance criteria**:

- [x] The skill instructions describe the walkthrough as a mandatory step in the loop, including how a resumed session continues an open review
- [x] The end-to-end harness expects the new canonical step order and asserts the workflow reaches finished only after the walkthrough
- [x] The template contract tests account for the new non-terminal step and the renumbered terminal template


### Milestone 2: Drafting runs autonomously, with judgement calls recorded and reviewed as assumptions

**What changes**: Producing the draft no longer interrupts the user section by section. The workflow drafts every section itself, making reasonable judgement calls as it goes — including choosing the design direction — and recording each one durably with its rationale and rejected alternatives. Only a genuinely blocking decision, one with no reasonable default, stops the drafting to ask. The recorded assumptions survive session interruption alongside the drafted sections, are folded into the research document, and are raised entry by entry as a first-class beat of the Milestone 1 walkthrough so every call can be challenged before sign-off. For a typical spec, the user is not interrupted once between starting the plan and beginning the review.

**Validation point**: The updated end-to-end harness run on a spec with no blocking decisions reaches the walkthrough with zero confirmation questions during drafting and a populated assumption log; template unit tests confirm the approval-gate phrasing is gone from the gathering steps and the assumptions beat is present in the walkthrough.

#### - [x] Phase 2.1: Make the gathering steps draft autonomously and record assumptions

Rewrite the closing beat of each section-gathering step: instead of presenting the section and waiting for approval, each step drafts its section, appends any judgement calls to a durable assumption log with rationale and rejected alternatives, and advances. Every gathering step gains the same compact rule for blocking questions — proceed without interruption by default; stop and ask only when a decision has no reasonable default or needs information only the user holds. The architecture step switches from asking the user to choose a direction to choosing the best-grounded direction itself and recording it as the leading assumption. Template-content tests are updated to enforce the new phrasing.

*Technical detail:* [context.md#phase-21](./context.md#phase-21-make-the-gathering-steps-draft-autonomously-and-record-assumptions)

**Acceptance criteria**:

- [x] No gathering step asks the user to approve its section before advancing; the wait-for-approval phrasing is gone from all of them
- [x] Each gathering step instructs recording judgement calls in the assumption log with decision, rationale, and rejected alternatives, and the log survives session interruption alongside the section working files
- [x] Each gathering step carries the proceed-unless-blocked rule, and the architecture step records its chosen direction as an assumption instead of asking
- [x] Template tests fail if a wait-for-approval gate or the old agreement phrasing is reintroduced in a gathering step

#### - [x] Phase 2.2: Fold assumptions into the research document and raise them in the walkthrough

Give the recorded assumptions their permanent home and their review moment: the assemble step folds the assumption log into the research document, the verification step checks the resulting section is present and filled, and the walkthrough gains a first-class beat that walks the assumptions entry by entry — decision, rationale, rejected alternatives — inviting challenge on each before sign-off. The skill instructions are updated to describe the assumption log as part of the working-file contract and to reinforce that drafting runs without stopping between steps.

*Technical detail:* [context.md#phase-22](./context.md#phase-22-fold-assumptions-into-the-research-document-and-raise-them-in-the-walkthrough)

**Acceptance criteria**:

- [x] After assembly, the research document contains the drafting assumptions with decision, rationale, and rejected alternatives for each
- [x] The walkthrough presents the recorded assumptions as a distinct beat of the review, not merely as a file the user could read
- [x] The skill instructions cover the assumption log in the working-file contract and direct the agent to draft straight through without per-section check-ins

#### - [x] Phase 2.3: Verify the autonomous flow end to end

Update the end-to-end harness to exercise and enforce the new interaction model: a full run against a spec with no blocking decisions must reach the walkthrough with zero confirmation questions during drafting, produce a populated assumption log, and leave the documents in-progress until sign-off. This is the automated proof of the spec's headline metric.

*Technical detail:* [context.md#phase-23](./context.md#phase-23-verify-the-autonomous-flow-end-to-end)

**Acceptance criteria**:

- [x] An end-to-end run on a blocking-decision-free spec reaches the walkthrough without any user-directed confirmation question during drafting
- [x] The run produces an assumption log populated with decision, rationale, and rejected-alternative entries, folded into the research document
- [x] The harness fails if the documents are marked completed before sign-off or if a drafting step stops to ask for approval

## Open Questions

Two genuinely implementation-time uncertainties remain; everything else was resolved during planning.

- **Whether the harbor harness can reliably detect "the agent asked a confirmation question" in a drafting transcript.** The zero-prompt assertion (Phase 2.3) depends on distinguishing a user-directed confirmation question from ordinary narration in the recorded agent transcript — only writing the detection against real transcripts will show whether a robust signal exists (e.g. structured question tool calls vs. prose heuristics). If no reliable signal can be found, STOP and ask the user whether to narrow the assertion (e.g. assert uninterrupted step progression only) or accept a weaker proxy.

- **Whether any implement-workflow surface hard-codes research.md's current seven-section list.** Discovery found implement's structural gate checks plan.md's sections, not research.md's, so adding `## Drafting assumptions` is believed safe — but this is recorded as an open assumption, and the truth only fully surfaces when the implement workflow is run against a plan produced under the new scaffold. If an implement-side check rejects the eight-section research.md, STOP and ask before changing anything on the implement side (that workflow is out of this spec's scope).

## Out of Scope

- **Changing the spec workflow's interaction model** — the spec workflow keeps its per-section confirmation flow untouched (spec § Constraints). Applying draft-then-walkthrough there is a possible future spec.
- **Keeping the old per-section confirmation flow as a configurable alternative** — the new model replaces it outright; no config toggle, no fallback (spec § Non-Goals).
- **Changing what content the plan gathers** — the set of plan sections and their meaning stay the same; only when the user is consulted changes. The one additive exception, explicitly in scope, is the `## Drafting assumptions` section in research.md (spec § Non-Goals, § Technical Approach).
- **Applying the model to the implement workflow's interaction points** — implement's STOP gates and per-phase pauses are unchanged (spec § Non-Goals). If the eight-section research.md surfaces an implement-side incompatibility, that is raised, not silently fixed (see § Open Questions).
- **A tunable threshold for what counts as a blocking question** — no config knob akin to `spec_trigger_threshold`; the drafting agent uses its judgement (spec § Non-Goals).
- **A hard, Go-enforced approval gate for `implement new`** — approval is expressed through the existing in-progress state and document status; a user can still deliberately bypass an open review with `--force`, and hardening that is left as a possible follow-up if it proves a problem in practice (recorded as an open assumption in research.md).
- **The knowledge-write propose-then-confirm rule** — `go run . knowledge write` still requires explicit user confirmation even during autonomous drafting; that standing rule belongs to the knowledge skill, not the plan workflow's interaction model.
- **Changing the shape, names, or location of the final plan documents** — plan.md, context.md, and research.md keep their paths and structure so the implement workflow consumes them unchanged (spec § Constraints).

## Changelog

### 2026-07-31 — Phase 1.1: Insert the walkthrough step into the plan workflow

**What was done**: Added a `walkthrough` FSM step between `write_research` and `finished` in the plan workflow, with a new mandatory-review template (`18-walkthrough.md`) lifted from the old finished-step walkthrough offer. The terminal template was renumbered to `19-finished.md` and slimmed to its post-sign-off job (stamp completed, report done, recovery branch intact). Step-order, FSM-walk, wiring, and template-content tests were extended accordingly.

**Deviations**: None.

**Files changed**:
- `internal/steps/plan/steps.go`
- `internal/steps/plan/steps_test.go`
- `templates/steps/plan/18-walkthrough.md` (new)
- `templates/steps/plan/19-finished.md` (renamed from `18-finished.md`, success branch slimmed)

**Discoveries**: `17-write_research.md`'s closing goto renders `{{next_step}}` from the callback's next-step argument, so retargeting `writeResearch()` in Go automatically pointed it at `walkthrough` with no template edit. The `templates/templates.go` embed is `all:*`, so the new template needed no registration. The context-directive floor (29) and the installed `.claude/skills` copies are deliberately untouched — both are Phase 1.2 work.

### 2026-07-31 — Phase 1.2: Align the driving skill and end-to-end harness with the new step

**What was done**: Updated the spek-plan skill so its loop names the walkthrough, states that the workflow ends with a mandatory sign-off review, and tells a resumed session on `walkthrough` to read the committed documents (the working files are gone by then). Updated the harbor harness's instruction and canonical step order for the new step, added a transcript-level assertion that the finished instruction renders only after a walkthrough instruction, and bumped the context-directive floor to 30. Refreshed the installed skill copy via `go run . init claude`.

**Deviations**: None.

**Files changed**:
- `templates/skills/workflows/spek-plan/SKILL.md`
- `.claude/skills/spek-plan/SKILL.md` (regenerated)
- `templates/context_directive_test.go`
- `tests/harbor/plan-workflow/instruction.md`
- `tests/harbor/plan-workflow/tests/test_plan_workflow.py`

**Discoveries**: `EXPECTED_SKILLS_PER_STEP` in the harbor suite needs no walkthrough entry — the walkthrough template references no skills. The harbor run has no human reviewer, so instruction.md now tells the harness agent it acts as the reviewing user and gives the affirmative sign-off itself; the harness cannot exercise a genuinely withheld sign-off.

### 2026-07-31 — Phase 2.1: Make the gathering steps draft autonomously and record assumptions

**What was done**: Removed the per-section approval gates from all eleven gathering templates (02-discovery through 12-out_of_scope): "present for review" and "once the user is happy/agreed" phrasing is gone, and "the agreed content" became "the drafted content". Every gathering step gained an identical shared block instructing it to append judgement calls to `.spektacular/work/<plan_name>/assumptions.md` (decision, rationale, rejected alternatives) and to proceed without interruption unless a decision has no reasonable default. The architecture step now picks the best-grounded direction itself and records it as the leading assumption-log entry instead of asking the user to choose.

**Deviations**: None. The knowledge-write propose-then-confirm gate in 02-discovery.md was deliberately preserved (out of scope per the spec).

**Files changed**:
- `templates/steps/plan/02-discovery.md` … `12-out_of_scope.md` (all eleven gathering templates)
- `internal/steps/plan/steps_test.go` (gate test rewritten as `TestArchitectureStepWeighsOptionsAndRecordsAssumption`; new table-driven `TestGatheringStepsProceedWithoutApprovalGates`)
- `templates/work_files_test.go` (gathering steps must also reference `assumptions.md`; spec steps exempt)

**Discoveries**: The word "agreement" appears in no rendered gathering template after the edit (only in 18-walkthrough.md), so the architecture test can ban it wholesale. The shared block is byte-identical across all eleven templates (verified by hash) — keep it that way when editing; the table-driven test asserts its two anchor phrases per template. 13-assemble.md deliberately does not map assumptions.md yet — that is Phase 2.2's job, and `TestAssemblyStepsMapSectionFiles` must gain that entry then.

### 2026-07-31 — Phase 2.2: Fold assumptions into the research document and raise them in the walkthrough

**What was done**: Gave the assumption log its permanent home and review moment. The research.md scaffold gained a `## Drafting assumptions` section (position 7 of 8), the assemble step maps `assumptions.md` into it (a missing log is not a STOP — an explicit "No drafting assumptions were recorded." line is written instead), the verification step requires the section, and the walkthrough gained a fourth beat that reads the section from the committed research.md and walks the entries one by one, inviting challenge — a challenged assumption is routed through the existing apply-immediately change path. The spek-plan skill now names `assumptions.md` in the working-file contract and states that drafting runs straight through without per-section check-ins.

**Deviations**: None.

**Files changed**:
- `templates/scaffold/research.md`
- `templates/steps/plan/13-assemble.md`
- `templates/steps/plan/14-verification.md`
- `templates/steps/plan/18-walkthrough.md`
- `templates/skills/workflows/spek-plan/SKILL.md` (+ regenerated `.claude/skills/spek-plan/SKILL.md`)
- `internal/steps/plan/scaffold_test.go`, `internal/steps/plan/steps_test.go`, `templates/work_files_test.go`

**Discoveries**: 02-discovery.md's "all seven sections" phrasing refers to the discovery step's own research working file, which still maps onto seven research.md sections — `## Drafting assumptions` is filled from `assumptions.md` at assemble, so that template needed no change. The implement workflow's structural gate checks plan.md's sections only, so the eight-section research.md is safe for existing tooling (the plan's open question about implement-side coupling stays open until an implement run consumes a new-format plan).

### 2026-07-31 — Phase 2.3: Verify the autonomous flow end to end

**What was done**: Updated the harbor E2E harness to exercise and enforce the new interaction model, and ran it to a fully green 93/93 result. New assertions: zero `AskUserQuestion` calls in any pre-walkthrough step window (the extractor whitelist now captures that tool), a populated `## Drafting assumptions` section in the committed research.md (decision/rationale/rejected per entry, or the explicit empty-fallback line), and a status-lifecycle pair — documents observed `in-progress` via `plan file read` results inside the walkthrough window and `completed` on all three artefacts after finished (new hand-rolled frontmatter parser). instruction.md gained draft-straight-through guidance, walkthrough sign-off guidance (the harness agent acts as the reviewing user), and matching success criteria; solve.sh gained the missing `assemble` and new `walkthrough` gotos.

**Deviations**: The plan's open question about question-detection resolved without narrowing: `AskUserQuestion` tool calls are a reliable transcript signal (prose-question detection was deliberately not attempted — assistant text is unconstrained and the automated agent self-answers). Beyond the planned scope, four pre-existing harness/product drifts surfaced by the runs were fixed: (1) `task.toml` agent timeout raised 900→1800s — the old budget predated the walkthrough step and timed out mid-assemble; (2) the seeded spec was renamed `20260101000000-user-auth.md` (Dockerfile/instruction/solve.sh aligned) because the store's `missing_id_prefix` validation rejects un-prefixed names under `spec.id_method: timestamp`; (3) the `CONVENTIONS_READ_COMMAND` oracle updated `knowledge conventions` → `knowledge always-applied` to match the current discovery template; (4) the `<description>` scaffold-leftover literal false-positived on legitimate JSON-envelope prose and was replaced with the distinctive `### Option A: <name>`. One product-template hardening rode along: `18-walkthrough.md` now mandates reading all three committed documents at walkthrough start (was "as you need them"), which grounds resumed reviews in committed state and makes the in-progress observation deterministic.

**Files changed**:
- `tests/harbor/plan-workflow/tests/test_plan_workflow.py`
- `tests/harbor/plan-workflow/instruction.md`
- `tests/harbor/plan-workflow/solution/solve.sh`
- `tests/harbor/plan-workflow/task.toml`
- `tests/harbor/plan-workflow/environment/Dockerfile` (+ spec renamed to `environment/20260101000000-user-auth.md`)
- `templates/steps/plan/18-walkthrough.md`

**Discoveries**: The harbor verifier is name-agnostic (it derives the plan name from `state.json` `data.name`), so the spec rename needed no test edits. Run evidence worth keeping: the in-container agent drove all drafting steps with zero confirmation questions in every run, and when its first `plan file write user-auth/...` was rejected it improvised a timestamp prefix — the store's `next_action` error guidance works, but a seeded spec must carry a valid ID prefix up front. Full suite: 93 passed (job 2026-07-31__10-24-54's successor run).
