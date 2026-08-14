---
created_date: "2026-08-14"
status: completed
closed_date: "2026-08-14"
---

# Context: 000043_flipped-interaction-spec-interview

## Current State Analysis

The spec workflow's step table lives at `internal/steps/spec/steps.go:25-38` (`Steps()`), currently ten steps: `new`, `overview`, `requirements`, `acceptance_criteria`, `constraints`, `technical_approach`, `success_metrics`, `non_goals`, `verification`, `finished`. Every gathering-step callback (`overview()` through `nonGoals()`, steps.go:101-141) is a one-line `writeStep(...)` call; only `new()`, `verification()`, and `finished()` carry additional logic. `writeStep` (steps.go:52-64) delegates to `stepkit.WriteStepResult` (`internal/stepkit/stepkit.go:55-94`), which merges `{step, title, next_step, config.command}` with the strategy's `PathVars` (`internal/steps/spec/strategy.go:15-22`, supplying `spec_path`/`spec_name`) and the callback's `Extra` map, renders the mustache template, and writes a `Result`.

Every section-gathering template (`templates/steps/spec/01-overview.md` through `07-non_goals.md`) shares an identical three-part shape: an opening "Ask the user: [scripted question]" block, section-specific format/altitude rules, then a byte-identical closing pair: a working-file write instruction (`.spektacular/work/{{spec_name}}/<section>.md`) and the context-directive footer (marker text asserted by `templates/context_directive_test.go:17`). This working-file convention already exists for spec, confirmed by `templates/work_files_test.go:23-36` (`TestSpecPlanGatheringStepsReferenceWorkDir`), which currently checks all seven section templates. `templates/steps/spec/08-verification.md` is the assembly-and-review step: it reads every section's working file, stages the assembled spec, spawns an independent fresh-eyes reviewer, triages findings with the user (steps 3-4, lines 75-93, the direct precedent for this plan's rejection-repair generalization), applies confirmed fixes to owning working files, commits via `spec file write`, then removes the working directory (`rm -rf .spektacular/work/{{spec_name}}`, the only two templates in the codebase permitted to contain this cleanup per `TestAssemblyStepsCleanupWorkDir`, work_files_test.go:89-117).

Cross-repo awareness already exists, but only for the plan workflow. `repoRoster(cfg, root) []map[string]any` (`cmd/plan.go:155-177`) projects each registered repo's name plus its own descriptive metadata (description, role, tags, deployment) from `set.DescriptiveMetadata` (`internal/repo/set.go:112-130`). `runPlanNew` (`cmd/plan.go:133`) and `runPlanGoto` (`cmd/plan.go:234`) both call `wf.SetData("repos", repoRoster(cfg, root))` on every invocation. `repoRosterExtra(data workflow.Data) map[string]any` (`internal/steps/plan/steps.go:100-106`) pulls it back out of workflow `Data`; `discovery()` and `architecture()` (steps.go:108-118) pass `repoRosterExtra(data)` as their `Extra` argument. The rendered template (`templates/steps/plan/02-discovery.md:15-24`) shows a `{{#repos}}...{{/repos}}` block with a `{{^repos}}` empty-registry fallback. `cmd/spec.go`'s `runSpecNew`/`runSpecGoto` (spec.go:149-244, 246-316) have no equivalent call today.

The docs repo (spektacular-website) has two directly relevant pages already read in full: `src/pages/how-it-works.mdx`'s Stage 1 ("Specification") body (`Fragment slot="body"`, lines 225-239) is the authoritative "what happens when you create a spec" prose today, and needs rewriting to describe the interview phase happening first. `src/pages/index.mdx`'s `FeaturesGrid` (lines 45-78) is the site's existing "differentiator" pattern, six `FeatureCard` entries today. No dedicated "mock conversation transcript" component exists anywhere in `src/components/`; the closest reusable pattern for a worked example is `SpecFormat`'s `slot="example"` Fragment holding a fenced markdown block (how-it-works.mdx:117-152).

## Per-Phase Technical Notes

### Phase 1.1: Insert the interview step into the spec workflow

**File changes:**
- `internal/steps/spec/steps.go:25-38` (`Steps()`) — insert a new entry `{Name: "interview", Src: []string{"new"}, Dst: "interview", Callback: interview()}` between the `new` and `overview` entries; change the `new` entry's implicit next-step target (the `writeStep("new", "overview", ...)` call inside `new()`, steps.go:71,97) from `"overview"` to `"interview"`; change `overview()`'s Src expectation to originate from `interview` (steps.go:28's `Src: []string{"new"}` becomes `Src: []string{"interview"}`).
- `internal/steps/spec/steps.go` — add a new `interview()` function following the exact one-line shape of `overview()` (steps.go:101-105): `return "", writeStep("interview", "overview", "steps/spec/00b-interview.md", data, out, st, cfg, nil)` (the `nil` Extra becomes `stepkit.RepoRosterExtra(data)` once Phase 1.2 lands).
- `templates/steps/spec/00-new.md` — frontmatter `next: overview` (line 3) becomes `next: interview`; the closing `spec goto --data '{"step":"overview"}'` call (line 20) becomes `{"step":"interview"}`.
- New template file `templates/steps/spec/00b-interview.md` (numbering note below). Content: the Flipped Interaction structure (stated goal, adaptive questions, explicit stopping condition per the spec's Technical Approach and the arXiv:2302.11382 citation), the "not enumerating every requirement, building a picture sufficient to draft a first pass" framing, an instruction to write synthesized findings (not a transcript) to `.spektacular/work/{{spec_name}}/interview.md` using the agent's own `Write` tool, and the standard context-directive footer.

**Numbering assumption**: `00b-interview.md` avoids renumbering `01-overview.md` through `09-finished.md`, since step order is enforced by FSM Src/Dst wiring and tests, not filename sort order. Flagged as an Open Question (plan.md § Open Questions) because this was not exhaustively verified during discovery; if the implementer finds a dependency on strict sequential filename numbering, fall back to a full renumber (`01-interview.md`/`02-overview.md`/etc.), mirroring how 000040 renumbered its own terminal template on insertion.

**Complexity**: Medium (touches the FSM step table, a retargeted step, and a new template that must get the interaction-model prose right, the prose is the hard part, not the wiring).

**Token estimate**: ~15k tokens.

**Agent strategy**: Single agent, sequential execution, the step-table change and its template are tightly coupled and best done by one agent that can cross-check the rendered instruction against the FSM wiring.

### Phase 1.2: Give the interview access to the project's repo roster and a cross-repo question

**File changes:**
- `cmd/plan.go:145-172` — remove the unexported `repoRoster` function from this file; its body moves verbatim (only signature/receiver-package changes) to a new exported function.
- `internal/repo/set.go` (or a new sibling file, e.g. `internal/repo/roster.go`) — add `func Roster(cfg config.Config, root string, git GitRunner) []map[string]any`, body identical to the removed `cmd/plan.go` version, using `Set.DescriptiveMetadata` (set.go:112-130) exactly as before.
- `cmd/plan.go:133` (`runPlanNew`) and `cmd/plan.go:234` (`runPlanGoto`) — update call sites from `repoRoster(cfg, root)` to `repo.Roster(cfg, root, repoGit)` (the existing `repoGit` `GitRunner` value already in scope at both call sites).
- `internal/steps/plan/steps.go:100-106` — remove the unexported `repoRosterExtra` function; its body moves verbatim to a new exported function `stepkit.RepoRosterExtra` in `internal/stepkit/stepkit.go` (adjacent to `GetString`/`RenderTemplate`, stepkit.go:110-127).
- `internal/steps/plan/steps.go:108-118` (`discovery()`, `architecture()`) — update both calls from `repoRosterExtra(data)` to `stepkit.RepoRosterExtra(data)`.
- `cmd/spec.go:229-238` (`runSpecNew`) — add `wf.SetData("repos", repo.Roster(cfg, root, repoGit))` immediately after constructing `wf` (line 232), matching `cmd/plan.go:133`'s placement exactly; requires importing `internal/repo` in `cmd/spec.go` (confirm at implementation time whether it's already imported) and having `root`/`repoGit` in scope (root already resolved at spec.go:173-176).
- `cmd/spec.go:246-316` (`runSpecGoto`) — same `wf.SetData(...)` addition, mirroring `cmd/plan.go:234`.
- `internal/steps/spec/steps.go` — `interview()`'s `nil` Extra argument (from Phase 1.1) becomes `stepkit.RepoRosterExtra(data)`.
- `templates/steps/spec/00b-interview.md` — add the `{{#repos}}...{{/repos}}` / `{{^repos}}` mustache block, copied from `templates/steps/plan/02-discovery.md:15-24`'s existing rendering, adapted for the interview's own surrounding prose (asking whether there's cross-cutting impact, not what the changes are, that's plan-workflow territory per the spec's own scoping).

**Complexity**: Medium (two symbol relocations plus two new call sites; low risk since both relocated functions are copied verbatim, but requires careful confirmation that no plan-workflow test regresses).

**Token estimate**: ~12k tokens.

**Agent strategy**: Single agent, sequential execution, relocate first, confirm plan-workflow tests still pass unmodified, then wire the spec side.

### Phase 1.3: Align the driving skill and end-to-end harness with the new step

**File changes:**
- `templates/skills/workflows/spek-new/SKILL.md` (path to confirm at implementation time) — add coverage of the interview step to the loop description: what it does, its stopping condition, its repo-roster awareness, and how a resumed session continues an open interview.
- `internal/steps/spec/steps_test.go:56-74` (`TestStepsOrderMatchesExpected`) — add `"interview"` to the `expected` slice between `"new"` and `"overview"`.
- `internal/steps/spec/steps_test.go:76-107` (`TestFSMWalkFromNewToFinished`) — add `"interview"` to `expectedStates` in the same position.
- `tests/harbor/spec-workflow/tests/test_spec_workflow.py:20-31` (`EXPECTED_STEP_ORDER`) — add `"interview"` after `"new"`.
- `tests/harbor/spec-workflow/tests/test_spec_workflow.py` — add a `TestInterviewStep` class after `TestNewStep` (line ~309-321): `test_step_completed`, `test_tool_called`, and an adapted content check reusing the `TestOverviewStep.test_content_is_relevant` style since the interview produces no scaffold section itself.
- `tests/harbor/spec-workflow/solution/solve.sh:15-17` — add `interview` as an explicit prior `spektacular spec goto --data '{"step":"interview"}'` call before the existing `for STEP in ...` loop.
- `templates/context_directive_test.go:35-37` (`exemptFromContextDirective`) — default decision: do **not** add the interview template here; it carries the standard directive (see Assumptions below for rationale).
- `templates/work_files_test.go:23-36` (`TestSpecPlanGatheringStepsReferenceWorkDir`) — add `"steps/spec/00b-interview.md"` to the `specGathering` slice.

**Complexity**: Low (mechanical test-list updates plus one skill-file prose addition; the context-directive exemption question has a stated default so it does not block progress).

**Token estimate**: ~8k tokens.

**Agent strategy**: Single agent, sequential execution.

### Phase 2.1: Make section steps draft from the interview and present for confirmation

**File changes:**
- `templates/steps/spec/01-overview.md:7-14` — replace the scripted-question block with drafting instructions: read `.spektacular/work/{{spec_name}}/interview.md` (and this section's own working file if one already exists from a prior pass), draft the Overview applying the existing stakeholder-readability rule (line 14), present the draft, and ask the user to confirm or correct it. Lines 3-6 and 16-28 unchanged.
- `templates/steps/spec/02-requirements.md:3` — same treatment; lines 5-30 unchanged.
- `templates/steps/spec/03-acceptance_criteria.md`, `04-constraints.md`, `05-technical_approach.md`, `06-success_metrics.md`, `07-non_goals.md` — identical treatment, each template's opening line replaced, every other line unchanged. Re-read each file at implementation time before editing (exact line numbers beyond 01/02 not individually re-verified during discovery).
- `templates/work_files_test.go` — no change expected to `TestSpecPlanGatheringStepsReferenceWorkDir`'s assertions; the seven templates still reference `.spektacular/work/{{spec_name}}/` exactly as before.

**Complexity**: Medium (seven templates, each requiring careful prose that preserves every existing rule while replacing only the elicitation mechanism).

**Token estimate**: ~20k tokens.

**Agent strategy**: 2-3 parallel agents for independent changes — split by group (e.g. 01-03, 04-05, 06-07), then a single sequential consistency pass before Phase 2.2.

### Phase 2.2: Add rejection-repair and cross-section amendment to every gathering step

**File changes:**
- `templates/steps/spec/00b-interview.md`, `01-overview.md` through `07-non_goals.md` — add a shared closing-beat paragraph, positioned after the working-file-write instruction and before the context-directive footer, generalizing `templates/steps/spec/08-verification.md:75-93`: "If the user indicates this draft is wrong, ask a follow-up question to understand why before changing anything, the issue may reveal a broader need you didn't surface, or may be a genuine miss on your part, and the follow-up conversation determines which. Apply any resulting changes directly to the working file(s) they belong to, which may include a different section's working file than the one under review; a section amended this way does not need a fresh confirmation step now, the end-of-workflow verification step is where everything, including this change, gets reviewed together." Identical wording across all eight templates so a single test string can assert its presence everywhere.
- New test: a `TestRejectionRepairPresent`-style function (location TBD, likely `templates/rejection_repair_directive_test.go` following `context_directive_test.go`'s naming) walking `steps/spec/00b-interview.md` through `steps/spec/07-non_goals.md`, asserting each contains the shared marker substring exactly once.

**Complexity**: Medium (eight templates need the identical paragraph at the same relative position; low design risk since wording is fully specified, but requires careful placement to avoid disrupting the existing context-directive footer's own marker-count assertions).

**Token estimate**: ~15k tokens.

**Agent strategy**: 2-3 parallel agents, same rationale as Phase 2.1.

### Phase 2.3: Verify the new interaction model end to end

**File changes:**
- `tests/harbor/spec-workflow/tests/test_spec_workflow.py` — extend `TestWorkflow` or add a new class (e.g. `TestCrossSectionAmendment`) inspecting the transcript for a scripted rejection scenario and asserting the resulting spec content in an earlier section reflects the correction, using the existing `parse_sections`/`load_spec` helpers (test_spec_workflow.py:57-89).
- `tests/harbor/spec-workflow/instruction.md` — extend the task brief with a scripted rejection beat: after an early section (e.g. Requirements) is drafted and confirmed, the agent under test rejects a later section's draft (e.g. Constraints) in a way that should surface a needed addition to Requirements, and the harness checks Requirements' final content for that addition.
- `tests/harbor/spec-workflow/solution/solve.sh` — genuinely undecided at plan time: needs either an equivalent scripted rejection-and-correction sequence, or an explicit note that this scenario is agent-only. Deferred to implementation (see Open Questions, plan.md).

**Complexity**: Medium (designing a realistic, assertable cross-section-amendment scenario is the hard part; assertion mechanics reuse existing helpers).

**Token estimate**: ~15k tokens.

**Agent strategy**: Single agent, sequential execution — scenario design, instruction wording, and assertion all need to agree with each other.

### Phase 3.1: Document the Flipped Interaction pattern and the interview phase on the docs site

**File changes:**
- `docs:src/pages/how-it-works.mdx:217-241` (`PipelineStage` "Stage 1: Specification", `Fragment slot="body"` at lines 225-239) — rewrite per plan.md's Content outline: pattern-naming opening line, interview example exchange (new `Fragment slot="example"` block following the existing pattern at how-it-works.mdx:117-152), stopping-condition sentence, cross-repo worked example, transition sentence.
- `docs:src/pages/how-it-works.mdx:221` (`PipelineNode` `sub` text) — optional cosmetic revision hinting at the interview; implementer's call.
- `docs:src/pages/how-it-works.mdx:60-76` (Quick-start step 3) — one-line addition noting the interview happens first.

**Complexity**: Medium (new prose content following an established component pattern; no new components, but needs to read well and match the site's existing voice).

**Token estimate**: ~15k tokens.

**Agent strategy**: Single agent, sequential execution.

### Phase 3.2: Feature the capability on the homepage

**File changes:**
- `docs:src/pages/index.mdx:45-78` (`FeaturesGrid`) — add one new `FeatureCard` (or reframe the existing "Spec-driven workflow" card) per plan.md's Content outline, using the existing `icon`/`title` props and slot body pattern (`docs:src/components/FeatureCard.astro:1-16`).

**Complexity**: Low (one small, well-precedented addition to an existing repeatable component).

**Token estimate**: ~5k tokens.

**Agent strategy**: Single agent, sequential execution.

## Testing Strategy

See plan.md § Testing Approach for the overall strategy. Per-phase test obligations: Phase 1.1/1.3 add/update `internal/steps/spec/steps_test.go` FSM-order tests and harbor `EXPECTED_STEP_ORDER`; Phase 1.2 relies on `internal/steps/plan/steps_test.go`'s existing repo-roster tests as an unmodified regression guard for the relocation; Phase 1.3 extends `templates/context_directive_test.go` and `templates/work_files_test.go`; Phase 2.1 has no new test file (existing `work_files_test.go` assertions already cover the touched lines); Phase 2.2 adds a new template-content test asserting the shared rejection-repair marker across all eight steps; Phase 2.3 is where the harbor suite gains its cross-section-amendment behavioral proof; Phase 3.1/3.2 are verified by the docs repo's own `npm run build` / `npx astro check` per the MDX authoring convention's verification table, not by this repo's Go test suite.

## Project References

- **spektacular** (this repo, role: tool) — carries every Go-side change: `internal/steps/spec`, `internal/repo`, `internal/stepkit`, `cmd/spec.go`, `cmd/plan.go`, `internal/steps/plan`, `templates/steps/spec/`, `templates/skills/workflows/spek-new/`, and the Go/template/harbor test suites. All of Milestones 1 and 2 land here.
- **docs** (spektacular-website, role: documentation, local path `../spektacular-website` per `go run . repo list`) — carries Milestone 3 in full: `src/pages/how-it-works.mdx` and `src/pages/index.mdx`. No other file in this repo is touched by this plan.
- Prior plans consulted for precedent, both in this repo's own plan store: 000040_plan-draft-then-walkthrough (structural precedent for the step-insertion-plus-template-rewrite recipe) and 000042_repo-self-describing-metadata (established the `repoRoster`/`DescriptiveMetadata` mechanism this plan relocates and reuses).

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|-------------|----------------|
| Low | ~10k | Single agent, sequential |
| Medium | ~25k | 2-3 parallel agents |
| High | ~50k+ | Parallel analysis, sequential integration |

## Migration Notes

None. No data migration, no config schema change, no breaking change to any existing CLI flag or output shape. The repo-roster relocation (Phase 1.2) is an internal refactor with no external-facing effect; existing projects' `.spektacular/config.yaml` and `repo.yaml` files are read exactly as before.

## Performance Considerations

None beyond what already applies to every spec-workflow step: one additional template render and one additional `spec goto` round-trip per spec-creation session (the new interview step), negligible relative to the conversational nature of the workflow. No new I/O pattern, no new network call, no change to the CLI's startup or command-dispatch cost.
