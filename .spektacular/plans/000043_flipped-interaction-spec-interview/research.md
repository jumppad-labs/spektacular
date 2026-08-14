---
created_date: "2026-08-14"
status: completed
closed_date: "2026-08-14"
---

# Research: 000043_flipped-interaction-spec-interview

## Alternatives considered and rejected

- **Making the interview a sub-conversation inside the existing `new` step,
  rather than its own FSM step.** Rejected: the `00-new.md` template
  (spektacular:templates/steps/spec/00-new.md) already has one well-defined
  job — create the scaffold and capture free-form context.md before
  advancing — and is explicitly exempted from the generic
  refresh-context directive for that reason
  (spektacular:templates/context_directive_test.go:35-37,
  `exemptFromContextDirective`). Folding the adaptive interview loop into
  it would overload one step with two different interaction models (a
  single "write down what was said" instruction vs. a multi-turn
  question/stop-condition loop) and would make the interview
  un-resumable as its own state (a session interrupted mid-interview
  would resume at `new`, which the FSM already treats as
  instantaneous/no-user-interaction). A dedicated step gets its own
  `current_step` value, so resume semantics fall out for free, exactly as
  000040's `walkthrough` step did for the plan workflow (see Prior plans
  below).
- **Not persisting interview findings to a working file, keeping them
  only in `context.md`.** Rejected: `context.md` is explicitly scoped
  in every existing template as "cross-cutting working context only... not
  a copy of content already captured elsewhere"
  (spektacular:templates/steps/spec/01-overview.md:28, the same marker
  string `context_directive_test.go` asserts on). The interview's findings
  are exactly the kind of durable, structured content the existing
  per-section pattern (`.spektacular/work/{{spec_name}}/*.md`) already
  exists for. Reusing that convention rather than inventing a second
  persistence channel keeps one working-file contract instead of two.
- **Applying the Flipped Interaction pattern to the plan workflow's
  discovery/architecture steps too.** Out of scope per the spec's
  Non-Goals; also would touch `internal/steps/plan`, a package explicitly
  out of scope for this plan.
- **A fresh per-section reconfirmation gate for a section amended as a
  side effect of a rejection elsewhere.** Rejected per spec constraint —
  relies on the existing verification-step fresh-eyes review/triage loop
  (spektacular:templates/steps/spec/08-verification.md steps 3-4) as the
  single end-of-workflow safety net, exactly as 000040 relied on its
  `walkthrough` step for the analogous plan-workflow decision.

## Chosen approach — evidence

**One new FSM step, `interview`, inserted between `new` and `overview`.**
Mechanically identical in shape to every existing step entry
(spektacular:internal/steps/spec/steps.go:25-38) and to how 000040 inserted
`walkthrough` between `write_research` and `finished` in the plan workflow's
step table (evidence: plan 000040_plan-draft-then-walkthrough §Data
Structures & Interfaces, `{Name: "walkthrough", Src: []string{"write_research"}, ...}`).
No FSM/engine changes needed — `workflow.New`/`Next`/`Goto` are generic over
the step table (spektacular:internal/workflow, confirmed via
spektacular:internal/steps/spec/steps.go using the same `workflow.StepConfig`,
`workflow.StepCallback` types the plan workflow uses).

**The `new` step's Dst/next_step changes from `overview` to `interview`,
and `interview`'s Dst/next_step is `overview`.** Every downstream step
(`overview` through `verification`) is otherwise unchanged in the FSM
table — satisfies the spec's "step count and order... changes what happens
within existing steps, not the sequence of steps itself" constraint,
because the *existing* steps keep their order; only one step is added at
the front, which is the same additive shape 000040 used.

**Interview persists findings to `.spektacular/work/{{spec_name}}/interview.md`**,
following the exact convention every section step already uses
(spektacular:templates/steps/spec/01-overview.md:20,
`.spektacular/work/{{spec_name}}/overview.md`, and enforced by
spektacular:templates/work_files_test.go:23-36,
`TestSpecPlanGatheringStepsReferenceWorkDir`). This file is *not* one of
the eight scaffold sections — it is read by the section-drafting steps
that follow it as their primary source material, and is included in the
verification step's cleanup (`rm -rf .spektacular/work/{{spec_name}}`,
spektacular:templates/steps/spec/08-verification.md:113, asserted by
spektacular:templates/work_files_test.go:89-117
`TestAssemblyStepsCleanupWorkDir`) since it is scratch, not a store
document.

**Section steps change from "ask scripted questions" to "draft from
interview.md (+ prior corrections), present, and confirm."** Current
scripted-question wording, to be replaced, e.g.:
spektacular:templates/steps/spec/01-overview.md:7-11 ("Ask the user to
describe this feature in 2-3 sentences: • What is being built? • What
problem does it solve? • Who benefits?"), and
spektacular:templates/steps/spec/02-requirements.md:3 ("Ask the user to
list the specific, testable behaviours..."). Both templates already
contain everything else needed unchanged: the WHAT-not-HOW test
(requirements.md:20-28), the stakeholder-readability rule (overview.md:14),
the working-file write instruction (both templates' penultimate
paragraph), and the context-refresh directive
(both templates' final paragraph, the marker
`context_directive_test.go` checks). Only the *question-asking* paragraph
at the top of each template needs to become *draft-and-present* wording —
this is a pure prose edit per step, no new mechanism, mirroring exactly how
000040 rewrote the plan workflow's gathering-step "closing beat" without
touching the surrounding rules (plan 000040 §Implementation Detail,
"De-gating is a rewrite of each gathering step's closing beat, not a
restructure").

**Rejection handling and cross-section amendment are new instruction
prose on the section-step templates, not new Go code or new state.**
There is no existing per-step "reject and ask why" mechanism in the spec
workflow — the closest existing precedent is the verification step's
step 3/4 triage loop (spektacular:templates/steps/spec/08-verification.md:75-93),
which already establishes the pattern this plan generalizes: "for every
finding you keep, propose it to the user and wait for confirmation," and
"update the owning section's working file... then re-assemble... from the
working files," i.e. amend a different working file than the one
currently being discussed, using only file tools, with no FSM
involvement. Generalizing this to every section step (not just
verification) is additive prose, following the same
`.spektacular/work/{{spec_name}}/<section>.md` addressing scheme already
in place for all eight sections.

**Cross-repo awareness reuses `repoRoster` and the `repoRosterExtra`
plumbing pattern unchanged, wired into spec the same way the plan
workflow already wires it — full mechanism now confirmed end-to-end.**
`repoRoster(cfg, root) []map[string]any` (spektacular:cmd/plan.go:155-177)
already projects each registered repo's name plus its own descriptive
metadata (description, role, tags-as-comma-joined-string, deployment)
sourced from `set.DescriptiveMetadata` — exactly the (name, role,
description) triple the spec's Technical Approach asks the interview to
reuse, tolerating a `repo.New` error by falling back to identity-only
entries. Two call sites set it into workflow `Data`, on *every*
invocation so a resumed session sees a freshly refreshed roster (comment
at spektacular:internal/steps/plan/steps.go:95-99): `runPlanNew`
(cmd/plan.go:133) and `runPlanGoto` (cmd/plan.go:234), both
`wf.SetData("repos", repoRoster(cfg, root))` immediately after
constructing `wf`. A small pull-back helper,
`repoRosterExtra(data workflow.Data) map[string]any`
(spektacular:internal/steps/plan/steps.go:100-106), reads the `"repos"`
key back out of `Data` and wraps it as `{"repos": repos}` for use as a
step's `Extra` map; exactly two plan steps consume it —
`discovery()` and `architecture()`
(internal/steps/plan/steps.go:108-118) — each passing
`repoRosterExtra(data)` as `writeStep`'s final argument instead of `nil`.
Template-side, `templates/steps/plan/02-discovery.md:15-24` renders a
mustache `{{#repos}}...{{/repos}}` block (with a `{{^repos}}` empty-
registry fallback sentence) listing each repo's name/description/role/
tags/deployment inline.

`cmd/spec.go`'s `runSpecNew` and `runSpecGoto` build an equivalent
`wfCfg`/`steps`/`wf` (spektacular:cmd/spec.go:149-244, 246-316) but
neither calls `wf.SetData("repos", ...)` today, and
`internal/steps/spec/steps.go` has no `repoRosterExtra` equivalent. Three
mechanical additions mirror the plan-workflow pattern exactly: (1) call
`wf.SetData("repos", repoRoster(cfg, root))` in both `runSpecNew` and
`runSpecGoto`, immediately after constructing `wf`, matching the plan
call sites line-for-line; (2) add a `repoRosterExtra`-equivalent (or
reuse a shared one — see below) in `internal/steps/spec/steps.go`, passed
as the new `interview()` step's `Extra` argument; (3) add the same
`{{#repos}}...{{/repos}}` mustache block to the new interview template.
**DRY consideration** (per this project's own "prefer DRY refactors over
accepting duplication" feedback convention): `repoRoster` living in
`cmd/plan.go` is package-`cmd`-reachable from `cmd/spec.go` without an
import change, so the smallest correct change calls it as-is from
`runSpecNew`/`runSpecGoto` rather than duplicating its body — but since
both call sites would then be functionally identical uses of a
plan-workflow-named symbol from spec-workflow code, the architecture step
should decide whether to leave `repoRoster` where it is (cheapest,
slightly misnamed provenance) or relocate/rename it to a workflow-neutral
location (e.g. a small shared `cmd` helper file, not a new package,
since it is a thin `cmd`-layer projection over `internal/repo` and
`internal/config` types already used by both commands). The
`repoRosterExtra` pull-back helper is 6 lines
(internal/steps/plan/steps.go:100-106) with no plan-specific logic in it
either — same DRY question applies, same recommendation (relocate to a
shared location both `internal/steps/plan` and `internal/steps/spec` can
call, e.g. `internal/stepkit`, rather than duplicating it verbatim into
`internal/steps/spec/steps.go`).

## Files examined

- spektacular:internal/steps/spec/steps.go:1-202 — full current step table
  (`new → overview → requirements → acceptance_criteria → constraints →
  technical_approach → success_metrics → non_goals → verification →
  finished`), each step a one-line `writeStep(...)` call; `new()` and
  `finished()` and `verification()` carry extra logic (scaffold creation,
  metadata close, spec_template extra var) but every other step is a pure
  one-liner — the shape a new `interview()` step callback should match.
  `writeStep` (steps.go:52-64) delegates to `stepkit.WriteStepResult`
  (internal/stepkit/stepkit.go:55-94), which merges `{step, title,
  next_step, config.command}` with the strategy's `PathVars` and the
  callback's `Extra` map (Extra wins on conflict), renders the mustache
  template, and writes a `Result` via `out.WriteResult`. Confirms `Extra`
  is exactly the channel the interview's repo-roster data (and, if
  needed, any other per-step injected data) flows through — no new
  plumbing concept required.
- spektacular:internal/steps/spec/strategy.go:1-26 — `PathStrategy` impl
  supplying `spec_path`/`spec_name` template vars to every step; no
  changes needed, `interview` step reuses it via `writeStep`.
- spektacular:internal/steps/spec/steps_test.go:56-107 — the two tests
  that hard-code the step list and must gain `"interview"` between
  `"new"` and `"overview"`: `TestStepsOrderMatchesExpected` (line 57-68)
  and `TestFSMWalkFromNewToFinished` (line 90-101, expectedStates slice).
- spektacular:templates/steps/spec/00-new.md:1-25 — `new` step template;
  its `next` frontmatter (line 3) and final `spec goto` call (line 20)
  both currently target `overview` and must retarget to `interview`.
- spektacular:templates/steps/spec/01-overview.md,
  02-requirements.md — read in full; both share the same three-part shape
  (mechanism-boundary preamble → scripted-question block → working-file
  write instruction → context-directive footer) that recurs across all
  seven gathering templates (01 through 07). Only the scripted-question
  block changes per step.
- spektacular:templates/steps/spec/08-verification.md:1-125 — full
  assembly + fresh-eyes-review + triage + commit + cleanup flow. Step 3/4
  (lines 75-93) is the direct precedent for "ask why before changing,
  update the owning working file, don't require re-confirmation of
  already-passed sections" — the interview/rejection-repair design
  generalizes this existing pattern rather than inventing a new one. Line
  9-16 lists the working-file-to-heading mapping that must gain no new
  entry for `interview.md` (it is source material, not an assembled
  section) but does gain a note that section drafts must additionally
  read it.
- spektacular:templates/steps/spec/09-finished.md:1-19 — terminal step;
  unaffected, no interview-related change needed.
- spektacular:cmd/spec.go:149-244 — `runSpecNew`; builds `wfCfg`
  (line 229) and calls `wf.SetData("name", ...)` (line 238) with no
  `"repos"` key set — the one-line gap `repoRoster` wiring fills.
- spektacular:cmd/plan.go:120-172 (repoRoster full definition ~145-172) —
  existing `repoRoster(cfg, root) []map[string]any` helper and its
  `wf.SetData("repos", repoRoster(cfg, root))` call site (line 133) in
  `runPlanNew` — the exact precedent to mirror in `runSpecNew`.
- spektacular:templates/context_directive_test.go:1-97 — enforces every
  non-terminal step template across all three workflows carries the
  refresh-context-directive marker exactly once, with the `00-new.md`
  exemption pattern (lines 35-37) for steps that have their own
  purpose-built context instruction. The new `interview` step must decide
  which camp it falls in (see Open assumptions).
- spektacular:templates/work_files_test.go:1-141 — enforces (a) every
  spec gathering step (01-07) references
  `.spektacular/work/{{spec_name}}/` (lines 23-36), (b) only the assembly
  step (08) contains the `rm -rf .spektacular/work/{{spec_name}}` cleanup
  (lines 89-117), (c) the assembly step's template text names every
  section working file it reads back (lines 119-140). All three
  assertions need a corresponding update/addition for the new interview
  step and file.
- spektacular:internal/workflow (package, referenced not fully read) —
  generic FSM engine (`looplab/fsm`-backed per
  spektacular:architecture/workflow-steps.md); confirmed step-table-driven
  and requires no change for an additive step insertion, per both direct
  reading of `steps.go`'s use of `workflow.StepConfig`/`StepCallback` and
  000040's precedent of doing the identical kind of insertion with zero
  engine changes.
- spektacular:architecture/workflow-steps.md (knowledge base, read in
  full) — canonical doc for adding a step: edit `Steps()`, write the
  callback, add the template. Directly followed as the recipe for the new
  `interview` step.
- spektacular:architecture/working-with-files-from-steps.md (knowledge
  base, read in full) — confirms all step file I/O goes through
  `store.Store`, never raw `os` calls, except the one documented exception
  already used by `new()` for `context.md` (relative-path `os.WriteFile`,
  spektacular:internal/steps/spec/steps.go:92-95) — working files under
  `.spektacular/work/` are written by the *agent's own Write tool*, not by
  Go step code, so this constraint does not apply to the interview's
  working file at all (consistent with how every existing section working
  file is written).
- spektacular:tests/harbor/spec-workflow/tests/test_spec_workflow.py:1-553
  — full E2E test suite. `EXPECTED_STEP_ORDER` (lines 20-31) hard-codes
  the ten-step list and needs `"interview"` inserted after `"new"`. Every
  `Test<Step>Step` class (`TestNewStep`, `TestOverviewStep`, ...) follows
  an identical three-test shape (`test_step_completed`, `test_tool_called`,
  `test_section_has_content`) that a new `TestInterviewStep` class should
  mirror, though "section has content" does not map cleanly (interview
  produces no scaffold section) — see Open assumptions.
- spektacular:tests/harbor/spec-workflow/instruction.md:1-50 — the
  harbor task brief handed to the agent under test; describes the
  workflow only in terms of "the skill will guide you through the full
  spec workflow" — no step names hard-coded, so no edit needed here
  beyond possibly noting the new interview phase for realism.
- spektacular:cmd/spec_test.go — command-handler-level tests for `spec
  new`/`spec goto`/`spec status`/`spec steps` (CLI plumbing, flags,
  schema output). A separate concern from the `internal/steps/spec`
  callback unit tests; likely needs a new/extended test if
  `wf.SetData("repos", ...)` is added to `runSpecNew`/`runSpecGoto`,
  mirroring whatever equivalent assertion exists for `cmd/plan_test.go`'s
  `repoRoster` wiring (not yet read; check before writing this plan's
  testing-approach section).
- spektacular:internal/steps/plan/steps_test.go (~lines 155-177,
  ~590-664) — contains a `repoRoster` test-fixture reimplementation and
  the `TestDiscovery...RendersRegisteredRepos` /
  `TestDiscoveryAndArchitectureStepsRenderEmptyRegistryFallback` /
  `TestDiscoveryStepDirectsRepoScopedResearch` tests — the direct
  template for whatever equivalent tests the new spec `interview` step
  needs for its own repo-roster rendering and empty-registry fallback.
- spektacular:tests/harbor/spec-workflow/solution/solve.sh:1-86 — the
  scripted non-agent "golden" solution that calls `spec goto` for each of
  the seven scripted `for STEP in ...` names (line 15) then hand-writes
  the full spec body and commits it via `spec file write`. This loop must
  gain `interview` (either inside the `for` loop after `new`, or as an
  explicit extra `spec goto interview` call before the loop) since the
  script drives the FSM directly rather than through the agent/skill.
- spektacular-website:src/pages/how-it-works.mdx — read in full by the
  docs research agent. Stage 1 body
  (`Fragment slot="body"` at lines 225-239) is the authoritative "what
  happens when you create a spec" prose and needs rewriting to describe
  the interview phase happening first. `PipelineNode` subtitle at line
  221 also names the `New` step and could gain an interview hint.
- spektacular-website:src/pages/index.mdx — read in full by the docs
  research agent. `FeaturesGrid`/`FeatureCard` (lines 45-78) is the site's
  existing "differentiator" pattern (six feature cards); adding a seventh
  card (or reframing the first) is the mechanism to satisfy "featured, not
  buried."
- spektacular-website:src/components/sections/FeaturesGrid.astro,
  src/components/FeatureCard.astro — read by the docs research agent;
  `icon`/`title` props with a slot body, no new component needed.
- spektacular-website:src/components/sections/PipelineStage.astro,
  Step.astro, SpecFormat.astro — read by the docs research agent;
  `SpecFormat`'s `slot="example"` Fragment holding a fenced ```markdown
  block (how-it-works.mdx:117-152) is the closest existing pattern for a
  worked mock-exchange example, reusable as-is for both the interview
  example and the cross-repo example (no new component required).
- spektacular-website:README.md — read by the docs research agent;
  confirmed this is the Astro-site build README (npm scripts only), not a
  product-feature surface — ruled out as a target for the "featured, not
  buried" requirement in favor of the homepage/how-it-works.

## External references

- White, J. et al., "A Prompt Pattern Catalog to Enhance Prompt
  Engineering with ChatGPT," arXiv:2302.11382 — source of the Flipped
  Interaction pattern this feature implements; the paper's own structure
  (stated goal, adaptive questioning, explicit stopping condition) is the
  shape the interview step's instructions should follow, per the spec's
  Technical Approach. Already summarized in detail in the spec-workflow
  context notes carried over from spec creation
  (`.spektacular/context.md` history for this plan, Overview step).

## Prior plans / specs consulted

- 000040_plan-draft-then-walkthrough (plan, read in full via `plan file
  read`) — closest structural precedent in this exact codebase: same kind
  of change (interaction-model rewrite via one new FSM step +
  template-prose edits, no engine changes), same package family
  (`internal/steps/plan` there, `internal/steps/spec` here), same
  enforcement mechanism (template-content tests), same working-file
  convention it extended (`.spektacular/work/<name>/*.md`, there adding
  `assumptions.md`; here adding `interview.md`), same E2E harness pattern
  (`tests/harbor/plan-workflow` updated for the new step, mirrored here by
  `tests/harbor/spec-workflow`). The milestone/phase structure of 000040
  (Milestone 1: new step + wiring + tests; Milestone 2: behavior change +
  verification) is a directly reusable shape for this plan's own
  milestones.
- 000042_repo-self-describing-metadata (plan, read in full via `plan file
  read`) — establishes `repoRoster`/`DescriptiveMetadata` as the current
  source of truth for a repo's own (description, role, tags, deployment),
  confirming there is no separate mechanism to build for cross-repo
  awareness — the existing helper is reused verbatim, only its call site
  changes (added to `cmd/spec.go`, matching its existing call site in
  `cmd/plan.go`).

## Open assumptions

- **Whether `interview` needs its own exemption in
  `context_directive_test.go`, like `00-new.md`.** Resolved during phases
  drafting (see this plan's Drafting assumptions): the interview step is
  **not** exempt and carries the standard directive, since its job
  (extracting structured findings into `interview.md`) is additive to,
  not a replacement for, `context.md`'s cross-cutting decisions/learnings
  refresh — unlike `00-new.md`, whose exemption exists because its job
  already duplicates the generic directive's purpose. This supersedes
  this research note's original "likely exempt" guess, made before the
  phases step's more considered pass.
- **Exact working-file name and shape for interview findings.** Assumed
  `interview.md` (singular, one file), holding the interview's synthesized
  understanding, not a raw transcript — matching the "capturing the
  interview's findings" phrasing in the spec's Technical Approach
  (findings, not transcript) and the existing convention that working
  files hold agreed *content*, not process. Not verified against any
  written precedent since no analogous "distilled conversation into one
  file" example exists in the spec workflow today (context.md is the
  closest, but is explicitly cross-cutting/decisions-only, not a single
  section's source material) — architecture step should confirm this
  shape explicitly as a first-class decision.
- **Whether the harbor `solve.sh` script (a scripted, non-agentic golden
  path) needs a real synthesized interview.md or can pass through a
  trivial one.** Assumed: trivial is acceptable, since `solve.sh` already
  bypasses conversational nuance entirely (it hand-writes the whole spec
  body directly) — it only needs to call `spec goto interview` (or
  equivalent) to keep the FSM walk valid, mirroring how it already treats
  every other step as a bare transition with no real content produced
  until the final `spec file write`.
- **Whether cross-section amendment requires any test assertion beyond
  "more than one working file changed after a single rejection
  conversation."** Assumed the testing approach step will define this as
  a manual/behavioral check (per the spec's own acceptance criteria
  language, which is behavioral/qualitative), not a Go unit test, since
  the actual amendment logic lives entirely in step-template prose with
  no parseable state to assert on structurally — consistent with how
  000040 treated its own qualitative success metrics as "Manual —
  captured in the implementation test plan."

## Drafting assumptions

### Chosen direction: mirror 000040's recipe, one new `interview` FSM step (architecture)
- **Decision**: Insert one new FSM step named `interview` between `new` and `overview` in `internal/steps/spec/steps.go`'s `Steps()`. All existing steps keep their relative order; only `new`'s destination/next_step retargets.
- **Rationale**: This is the exact recipe 000040_plan-draft-then-walkthrough already used and validated in this codebase for an interaction-model rewrite (new FSM step + template-prose edits, no engine changes). It satisfies the spec's "step count/order... changes what happens within existing steps, not the sequence itself" constraint, since the sequence of existing steps is preserved and only one step is added at the front.
- **Rejected**: Folding the interview into the existing `new` step (rejected — `new` already has one well-defined job and is explicitly exempted from the generic context-directive test for that reason; a dedicated step is also what makes the interview independently resumable, matching why 000040's `walkthrough` is its own step).

### Interview working-file name and shape (architecture)
- **Decision**: `.spektacular/work/{{spec_name}}/interview.md`, holding synthesized findings (not a raw transcript), following the exact convention every section step's working file already uses.
- **Rationale**: Matches the spec's Technical Approach wording ("capturing the interview's findings"), and reuses one working-file convention rather than inventing a second persistence channel alongside the existing `context.md`/section-file split.
- **Rejected**: Storing interview findings only in `context.md` (rejected — `context.md` is explicitly scoped as cross-cutting decisions/learnings only, not primary section source material, per the marker text every template's context-directive footer already asserts).

### Relocate `repoRoster`/`repoRosterExtra` rather than reach across packages (architecture)
- **Decision**: Move `repoRoster(cfg, root)` from `cmd/plan.go` into `internal/repo` (e.g. `repo.Roster(cfg, root, git)`), and move `repoRosterExtra` from `internal/steps/plan/steps.go` into `internal/stepkit`. Both `cmd/plan.go` and the new `cmd/spec.go` wiring call the relocated shared symbols; plan-workflow behavior is unchanged.
- **Rationale**: `internal/repo` already owns `Set`/`DescriptiveMetadata` and already imports `internal/config`, making it a natural fit with zero new dependencies; `internal/stepkit` already hosts the shared step-callback helpers (`GetString`, `RenderTemplate`, `WriteStepResult`) that both `internal/steps/plan` and `internal/steps/spec` import today. This satisfies the project's DRY-refactor preference (feedback memory: prefer DRY refactors over accepting duplication) without a larger redesign — it is a two-function move plus updated call sites.
- **Rejected**: Calling `repoRoster` directly from `cmd/spec.go` as it sits today in `cmd/plan.go` (rejected — works mechanically since both are package `cmd`, but leaves a function reasoned about as plan-specific serving a second, unrelated workflow, which reads as duplication-in-waiting the moment a third workflow needs the same roster).

### Rejection-repair and cross-section amendment as shared prose on every gathering step, not a new mechanism (architecture)
- **Decision**: Generalize the existing verification-step step-3/4 triage-and-fix loop (already: propose fix, wait for confirmation, update the owning working file, no fresh reconfirmation gate) into shared closing-beat prose added to the interview step and all seven section steps, rather than building a distinct rejection-handling mechanism.
- **Rationale**: The exact behavior the spec asks for (ask why before changing, allow edits to land in a different section's working file, rely on the end-of-workflow walkthrough as the safety net) already exists once in this codebase, in verification's triage loop. Reusing its shape everywhere is a prose generalization, not new design.
- **Rejected**: A fresh per-section reconfirmation gate for any earlier section amended as a side effect (rejected per the spec's own explicit constraint, and because it would reintroduce the serial-approval-gate proliferation 000040 deliberately moved away from on the plan-workflow side).

### Interview template numbering: 00b-interview.md rather than renumbering (phases)
- **Decision**: Name the new template `templates/steps/spec/00b-interview.md`, avoiding a renumber of `01-overview.md` through `09-finished.md`.
- **Rationale**: 000040 accepted a renumber for its own single insertion (retargeting the terminal template), but that was a single file at the very end of the sequence; renumbering eight files here (01 through 09, all shifting by one) is a larger blast radius for a purely cosmetic ordering concern, since the actual step order is enforced by the FSM's Src/Dst wiring and tests, not by filename sort order.
- **Rejected**: A full renumber to `01-interview.md`/`02-overview.md`/etc. — flagged explicitly in phases_context.md as the fallback if the codebase's actual template-numbering convention turns out to require strict sequential integers (not confirmed either way during discovery); implementer should check for any code or test that parses/sorts template filenames by numeric prefix before committing to `00b`.

### Context-directive exemption default: interview step is NOT exempt (phases)
- **Decision**: Default to giving the interview step the standard context-directive footer (like every section step), not the `00-new.md`-style exemption.
- **Rationale**: `00-new.md`'s exemption exists because its job (capture free-form conversation context) already duplicates what the generic directive asks for. The interview step's job is different — extract structured findings into `interview.md` — which doesn't inherently cover refreshing `context.md`'s cross-cutting decisions/learnings, so the generic directive is still additive value, not redundant.
- **Rejected**: Exempting the interview step by default — rejected as the less-safe default; an implementer who disagrees can move it to the exemption list with a one-line test change, whereas silently missing the directive would be a harder-to-notice gap.

### Cross-section-amendment E2E scenario ownership: instruction.md gains it, solve.sh's role deferred (phases)
- **Decision**: Extend `instruction.md` with a scripted rejection-and-correction beat; leave `solve.sh`'s exact treatment (mirror the scenario vs. an explicit skip note) as an implementation-time call.
- **Rationale**: `instruction.md` drives the actual agent under test, which is what needs to demonstrate the real behavior; `solve.sh`'s role in the harbor suite (golden-path baseline vs. something else) wasn't independently confirmed during discovery, so committing to its exact treatment now risks guessing wrong about a mechanism not fully verified.
- **Rejected**: Deciding solve.sh's treatment now without confirming its actual role — rejected in favor of flagging it explicitly for implementation-time confirmation, consistent with "no reasonable default" being the bar for stopping, and this genuinely has more than one reasonable default pending that one fact.

## Rehydration cues

- Re-read spektacular:internal/steps/spec/steps.go in full (202 lines) —
  the entire current step table and every callback signature; any new
  `interview()` callback must match this file's existing shape exactly.
- Re-read spektacular:templates/steps/spec/00-new.md,
  01-overview.md, 02-requirements.md, 08-verification.md,
  09-finished.md — the five templates most load-bearing for this
  feature's design (entry point, two representative gathering steps, the
  rejection-repair precedent, the terminal step).
- Re-read spektacular:templates/context_directive_test.go and
  spektacular:templates/work_files_test.go in full — both are short
  (<150 lines each) and encode every mechanical constraint the new step
  and template must satisfy; treat them as executable specification.
- Re-read spektacular:cmd/spec.go:149-244 (`runSpecNew`) and
  spektacular:cmd/plan.go's `repoRoster` function and its call site in
  `runPlanNew` — the one-line wiring change and its exact precedent.
- Re-read spektacular:tests/harbor/spec-workflow/tests/test_spec_workflow.py
  in full (553 lines) before writing the testing-approach section — every
  existing `Test<X>Step` class is a template for the new interview test
  class and for any rejection-repair/cross-section-amendment assertions.
- Re-read plan 000040_plan-draft-then-walkthrough via
  `go run . plan file read 000040_plan-draft-then-walkthrough/plan.md` —
  the single closest structural precedent for milestone/phase shaping;
  re-use its two-milestone pattern (step insertion + wiring first,
  behavior change second) as a starting skeleton.
- Run `go run . repo list` again if repo roster wiring is touched, to
  confirm the live roster shape (name/description/role/tags/deployment)
  matches what `repoRoster` still produces at implementation time.
- docs repo research already captured in full in this research.md's
  Files examined section above — no need to re-dispatch a docs research
  agent; re-read `how-it-works.mdx` Stage 1 body and `index.mdx`
  FeaturesGrid directly if the content outline needs re-verification
  against live file line numbers (they may have shifted).
