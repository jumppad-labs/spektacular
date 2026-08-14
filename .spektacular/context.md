# Implementation Context: 000043_flipped-interaction-spec-interview

## Workflow state

- First-phase invocation: plan.md has no `## Changelog` section yet. Pick up at Phase 1.1.
- Plan document frontmatter says `status: completed`/`closed_date: 2026-08-14`, but this is stale/incorrect — every phase checkbox in `## Milestones & Phases` is unchecked and nothing from this plan exists in the codebase (confirmed: no `interview` step in `internal/steps/spec/steps.go`, no `templates/steps/spec/00b-interview.md`). Treat the plan as not-yet-implemented; disregard the frontmatter status.

## Drift check findings (accepted, not fixed in plan doc)

User chose "proceed with mismatches noted" rather than editing the plan. Two real gaps in Phase 1.2, both in `cmd/spec.go`:

1. `runSpecNew` is actually at line 149, not the plan's claimed ~229-238.
2. `internal/repo` is NOT imported in `cmd/spec.go`, and there is no `repoGit`/`GitRunner` value in scope there at all (it's local to `cmd/plan.go` only). When implementing Phase 1.2's `wf.SetData("repos", repo.Roster(cfg, root, repoGit))` calls in `cmd/spec.go`, will need to add the `internal/repo` import and construct/wire an equivalent `GitRunner` value into scope in both `runSpecNew` and `runSpecGoto`.

All other drift-check findings across Phases 1.1, 1.3, 2.1, 2.2, 2.3, 3.1, 3.2 were MATCH or trivial line-number drift (off by a few lines) — not blocking, no action needed beyond re-reading files fresh when editing rather than trusting exact plan line numbers.

## Spec coverage

Verified during Step 3.5: every requirement and acceptance criterion in the spec (000043_flipped-interaction-spec-interview.md) has corresponding coverage in plan.md's Milestones & Phases. No descoped items.

## Phase 1.1 analysis (confirmed before implementing)

- `internal/steps/spec/steps.go` `Steps()` (lines 25-38) confirmed exact: 10 steps, `new` at Src `["start"]`, `overview` at Src `["new"]`. Plan of action: insert `{Name: "interview", Src: []string{"new"}, Dst: "interview", Callback: interview()}` between them; change `overview`'s Src to `["interview"]`.
- `new()` (steps.go:68-99) calls `writeStep("new", "overview", "steps/spec/00-new.md", ...)` twice (dry-run branch line 71, and real branch line 97) — both need their `"overview"` nextStep arg changed to `"interview"`.
- `overview()` (steps.go:101-105) confirmed exact one-line `writeStep(...)` shape — `interview()` will follow the identical shape.
- `templates/steps/spec/00-new.md` confirmed: `next: overview` at line 3, goto call `{{command}} spec goto --data '{"step":"overview"}'` at line 20 — both need to become `interview`.
- No test enforces sequential filename numbering (confirmed via grep across `internal/`, `templates/*.go`) — templates are referenced by explicit path in `steps.go` and by explicit lists in `templates/work_files_test.go` / `templates/context_directive_test.go`. Safe to add `templates/steps/spec/00b-interview.md` without renumbering 01-09.
- `templates/context_directive_test.go`'s `TestContextDirectivePresent` walks the directory via `fs.WalkDir` and asserts the context-directive marker on every non-terminal template not in `exemptFromContextDirective` (currently only `00-new.md`). `00b-interview.md` will be auto-covered by the walk as long as it carries the standard directive footer — no test file needs editing for this, and the `>= 30` floor check won't break.
- `templates/work_files_test.go`'s `TestSpecPlanGatheringStepsReferenceWorkDir` (line ~23-36) has an explicit `specGathering` slice listing `01-overview.md` through `07-non_goals.md` — per Phase 1.3 of the plan, this needs `"steps/spec/00b-interview.md"` added to it explicitly (unlike the context-directive test, this one is not a directory walk).

## Phase 1.1 implementation (done)

- `internal/steps/spec/steps.go`: inserted `interview` step in `Steps()` between `new` and `overview`; changed `overview`'s Src to `["interview"]`; changed both of `new()`'s `writeStep(...)` calls (dry-run branch and real branch) to target `"interview"` instead of `"overview"`; added `interview()` function following `overview()`'s exact one-line shape, pointing at new template `steps/spec/00b-interview.md`.
- `templates/steps/spec/00-new.md`: `next: overview` → `next: interview` in frontmatter; goto call and surrounding prose updated to reference `interview`.
- New `templates/steps/spec/00b-interview.md`: Flipped Interaction structure (stated goal, adaptive questions, explicit stopping condition, arXiv:2302.11382 citation), writes synthesized findings (not a transcript) to `.spektacular/work/{{spec_name}}/interview.md`, standard context-directive footer. Deliberately does NOT yet include repo-roster/`{{#repos}}` content — that's Phase 1.2's job per context.md; `interview()`'s Extra arg stays `nil` until then.
- Verified: `go build ./...` clean. `go test ./templates/...` passes (confirms `TestContextDirectivePresent`'s directory walk picked up the new template correctly with exactly one directive marker). `go test ./internal/...` has exactly 2 expected failures — `TestStepsOrderMatchesExpected` and `TestFSMWalkFromNewToFinished` in `internal/steps/spec/steps_test.go` — both asserting the OLD 10-step order without `interview`. These are Phase 1.3's job to fix (per context.md's Phase 1.3 notes: add `"interview"` to both tests' expected slices). No other test regressed.
- Confirmed no test enforces filename sequencing — `00b-interview.md` naming is safe, no renumbering needed.

## Phase 1.1 test step (done — no new test written, verified via sub-agent)

Delegated to a sub-agent per `follow-test-patterns`. Findings, verified concretely:
- FSM-order tests (`TestStepsOrderMatchesExpected`, `TestFSMWalkFromNewToFinished` in `internal/steps/spec/steps_test.go`) intentionally left failing — updating them is explicitly Phase 1.3's job. Not touched.
- `interview()` is a trivial one-line `writeStep(...)` wrapper identical in shape to 6 of 7 sibling step callbacks that have NO dedicated unit test (only `overview()` has one, a leftover from early development never replicated for siblings). No consistent per-callback testing convention exists to follow, so no `TestInterviewStepRendersInstruction` was invented.
- `templates/work_files_test.go`'s `TestSpecPlanGatheringStepsReferenceWorkDir` correctly does NOT need `00b-interview.md` added — that template isn't a numbered section-drafting step (it writes `interview.md`, not a `## <Section>` spec section), so it's out of scope for that specific test regardless of phasing.
- `templates/context_directive_test.go`'s `TestContextDirectivePresent` already covers `00b-interview.md` automatically via its `fs.WalkDir` walk — confirmed PASS.
- No test file written or modified in this step.

## Phase 1.1 verify step (done — found and fixed a plan gap, user-approved)

`make test` initially surfaced 4 failures, not the 2 anticipated by context.md's Phase 1.3 notes. The 2 extra were in `cmd/root_test.go`: `TestSessionLog_AdvancedTrueWhenStateChanges` (subtest "subsequent real step advance") and `TestSessionLog_SameSessionIDAcrossFoundingAndResumedCalls`, both of which hardcode `spec new` → `spec goto --data '{"step":"overview"}'` as a directly-reachable transition (using "overview" incidentally, as a stand-in for "any next reachable step", not testing overview-specific behavior). Verified via sub-agent (stash + test against clean main + pop) that this was a genuine regression from the Phase 1.1 change, not pre-existing.

**This is a real gap in the plan**: context.md's Phase 1.3 notes list `internal/steps/spec/steps_test.go`, `tests/harbor/spec-workflow/tests/test_spec_workflow.py`, and `tests/harbor/spec-workflow/solution/solve.sh` as needing FSM-order updates, but did NOT list `cmd/root_test.go`, which has the same class of hardcoded-step-order fragility. User chose to fix now rather than defer to Phase 1.3.

**Fix applied** in `cmd/root_test.go`:
- `TestSessionLog_AdvancedTrueWhenStateChanges` (~line 750-776): inserted an extra `goto --data '{"step":"interview"}'` call between `spec new` and the existing `goto overview` call, preserving the test's walk-through-real-steps intent rather than just renaming. This shifted the event count from 3→4 and the asserted event index from `events[2]`→`events[3]` (the test asserts properties of the overview→requirements transition, which is now the 4th event, not the 3rd).
- `TestSessionLog_SameSessionIDAcrossFoundingAndResumedCalls` (~line 850-854): simple rename, `"overview"` → `"interview"`, since this test only asserts session-ID equality across calls, not the specific step name — no event-count or index shift needed.
- Verified: `go build ./...` clean, `go test ./cmd/...` passes in full, `go test ./...` (whole repo) now has exactly the 2 originally-expected failures in `internal/steps/spec` and nothing else.

**Remaining note for Phase 1.3**: when that phase runs, be aware `cmd/root_test.go` already got its fix here — do not re-flag it as needing a change, and do not duplicate the `interview` goto insertion.

## Phase 1.1 update_plan step (done)

Checked off Phase 1.1's heading and 3 of 4 acceptance criteria in plan.md. Left the 4th ("step-order and full-workflow-walk unit tests pass with the new step in place") unchecked with an inline note — deliberately deferred to Phase 1.3 per user decision, since fixing those two tests is explicitly that phase's job and they're currently failing by design. Verified the "resumes with findings intact" criterion concretely via sub-agent before checking it: confirmed via `internal/workflow/workflow.go`'s generic `Goto`/`renderStep` (name-agnostic, proven by `TestGotoSameStepReRenders` in `workflow_test.go:87` using arbitrary step names) plus `templates/steps/resume.md`'s directory-glob read-back of `.spektacular/work/{{name}}/*` working files (not a fixed filename list) — both pre-existing, generic mechanisms needed no interview-specific code.

## Phase 1.1 update_changelog step (done)

First `## Changelog` section created in plan.md (appended after `## Out of Scope`), with Phase 1.1's entry. No knowledge-capture offer made — the two candidate discoveries (no filename-numbering test enforcement; `TestContextDirectivePresent`'s auto-covering directory walk) were judged too narrow/plan-specific to clear the durable-and-non-obvious bar.

**User has authorized running through all remaining 7 phases without pausing to ask between each one** (asked once after Phase 1.1, user chose "continue through all phases"). Do not re-ask before each subsequent phase's update_changelog step — loop straight through via `analyze` unless a STOP-on-mismatch condition fires.

## Phase 1.2 implementation (done)

Relocated the repo-roster mechanism and wired it into the spec workflow's interview step:
- New `internal/repo/roster.go`: `Roster(cfg config.Config, root string, git GitRunner) []map[string]any` — verbatim body of the old `cmd/plan.go` `repoRoster`, now taking `git` as an explicit param instead of closing over the package-level `repoGit` var (since `cmd/spec.go` needed its own call site too).
- `cmd/plan.go`: removed `repoRoster`, both call sites (`runPlanNew`, `runPlanGoto`) now call `repo.Roster(cfg, root, repoGit)`. Removed now-unused `strings`/`internal/config` imports.
- `internal/stepkit/stepkit.go`: added `RepoRosterExtra(data workflow.Data) map[string]any`, verbatim body of the old `internal/steps/plan/steps.go` `repoRosterExtra`, placed after `RenderTemplate`.
- `internal/steps/plan/steps.go`: removed `repoRosterExtra`; `discovery()`/`architecture()` now call `stepkit.RepoRosterExtra(data)`.
- `internal/steps/spec/steps.go`: `interview()`'s Extra arg changed from `nil` to `stepkit.RepoRosterExtra(data)`.
- `cmd/spec.go`: added `internal/repo` import; added `wf.SetData("repos", repo.Roster(cfg, root, repoGit))` in both `runSpecNew` (after `wf := workflow.New(...)`, before the `extraData` loop) and `runSpecGoto` (same placement pattern as `cmd/plan.go`'s `runPlanGoto`). This resolved the drift-check gap accepted at read_plan (missing import + no repoGit in scope) — build succeeded immediately once `repoGit` (a package-level var already in `package cmd` via `cmd/repo.go`) was referenced, no additional wiring needed since it's same-package.
- `templates/steps/spec/00b-interview.md`: added a `{{#repos}}...{{/repos}}`/`{{^repos}}` block (adapted from `templates/steps/plan/02-discovery.md`) plus prose instructing the interview to ask a cross-repo question — shaped by each other repo's own role/description, not generic — when the feature reads as focused on one repo and more than one repo is registered.

**Test relocation** (required to keep the codebase compiling after moving `repoRoster`, not new test authoring): `cmd/plan_test.go`'s two direct-unit tests of the old `repoRoster` (`TestRepoRoster_MaterializedRepoReportsOwnMetadata`, `TestRepoRoster_UnmaterializedRepoReportsNameOnly`) moved to new `internal/repo/roster_test.go` as `TestRoster_MaterializedRepoReportsOwnMetadata`/`TestRoster_UnmaterializedRepoReportsNameOnly`, reusing `set_test.go`'s existing `writeRepoConfig` helper instead of the removed `cmd`-package-local `writeDirRepoFootprint`. Removed now-unused `os`/`internal/config` imports from `cmd/plan_test.go`.

**Bug caught and fixed during relocation**: the relocated test initially failed — `writeRepoConfig(t, target, config.RepoConfig{...})` (a bare struct literal) fails `RepoConfigFromYAMLFile`'s `cfg.Validate()` because `Knowledge`/`Changelog` sub-configs are zero-value instead of defaulted. Fixed by constructing via `config.NewDefaultRepoConfig()` first and overriding only the descriptive fields, matching the original `cmd/plan_test.go` test's construction pattern (`writeDirRepoFootprint` also called `NewDefaultRepoConfig()` internally) which I'd initially dropped when simplifying to use `writeRepoConfig`. Confirmed fixed and passing before moving on.

Verified: `go build ./...` clean, `go test ./...` has exactly the 2 expected `internal/steps/spec` failures (deferred to Phase 1.3), nothing else broke — plan-workflow's own regression tests (`TestDiscoveryAndArchitectureStepsRenderRepoRoster`, `TestDiscoveryAndArchitectureStepsRenderEmptyRegistryFallback`, etc.) pass unmodified, confirming the relocation was behavior-neutral for the plan workflow.

## Phase 1.2 test step (done)

Sub-agent added `renderStepWithData` helper to `internal/steps/spec/steps_test.go` (mirroring `internal/steps/plan/steps_test.go`'s equivalent), plus 3 new tests: `TestInterviewStepRendersRepoRoster`, `TestInterviewStepRendersEmptyRegistryFallback` (table test, 2 cases: empty roster, absent key), `TestInterviewStepDirectsCrossRepoQuestion` (asserts stable substrings from the actual template prose). All pass. Regression check for criterion 3 (plan workflow's repo-roster rendering unchanged) confirmed via `go test ./internal/steps/plan/... ./internal/repo/... ./internal/stepkit/...` — all pass. Verified directly (not just trusting sub-agent report): `go build ./...` clean, full `go test ./...` has only the 2 expected `internal/steps/spec` failures.

## Phase 1.2 verify step (done — clean)

`go build ./...` pass, `make test` fails only the 2 expected `internal/steps/spec` tests (nothing else), `make lint` (`go vet`) pass. No new gaps found this time (unlike Phase 1.1's cmd/root_test.go surprise).

## Phase 1.2 update_plan step (done)

Checked off Phase 1.2's heading and all 3 acceptance criteria in plan.md — all genuinely verified.

## Phase 1.2 update_changelog step (done)

Appended Phase 1.2's changelog entry under the existing `## Changelog` section. Offered knowledge capture for the RepoConfig-construction gotcha discovered during test relocation; user accepted. Saved via spek-knowledge to project scope, `gotchas/repoconfig-must-start-from-default.md`.

6 phases remain (1.3, 2.1-2.3, 3.1-3.2). User already authorized running through all phases without re-asking (see earlier note) — proceeding straight to Phase 1.3 via `analyze`.

## Phase 1.3 implementation (done)

- `internal/steps/spec/steps_test.go`: added `"interview"` to `TestStepsOrderMatchesExpected`'s expected slice and `TestFSMWalkFromNewToFinished`'s expectedStates slice, both between `"new"` and `"overview"`. Both tests (and the whole suite) now pass — the 2 previously-deferred failures from Phase 1.1 are resolved.
- `tests/harbor/spec-workflow/tests/test_spec_workflow.py`: added `"interview"` to `EXPECTED_STEP_ORDER`; added new `TestInterviewStep` class (mirrors `TestNewStep`'s shape: `test_step_completed`, `test_tool_called`) after `TestNewStep`; updated `TestOverviewStep.test_tool_called` (stale comment/assertion said "reached automatically after spec new" — now correctly asserts `"spec goto interview"` was called, since `overview` is no longer auto-reached from `new`).
- `tests/harbor/spec-workflow/solution/solve.sh`: added explicit `spektacular spec goto --data '{"step":"interview"}'` call before the step loop; also added `overview` to the loop's step list (it was previously omitted from the loop because it was auto-reached from `new` — no longer true now that `interview` sits between them).
- `templates/skills/workflows/spek-new/SKILL.md`: updated the stale "(the `overview` step)" reference to "(the `interview` step)"; added a new "# The interview step" section covering what it does, repo-roster awareness + cross-repo question, the stopping condition, the working-file convention, and mid-interview resume behavior.
- `templates/work_files_test.go`: added `"steps/spec/00b-interview.md"` to `TestSpecPlanGatheringStepsReferenceWorkDir`'s `specGathering` slice, per the plan's explicit instruction (the assertion is a substring check for `.spektacular/work/{{spec_name}}/`, which the interview template already satisfies).

**Deliberately not touched**: `.claude/skills/spek-new/SKILL.md` and `.bob/skills/spek-new/SKILL.md` — these are git-tracked but *generated* installed copies of the template (rendered with `{{command}}` → `go run .`), refreshed only by an explicit user-initiated `go run . init <agent>` re-run, per AGENTS.md's own documented convention ("Hand edits will not survive the next init"). Confirmed no Makefile target regenerates them and no prior plan's changelog shows a precedent for force-regenerating them mid-implementation. Editing only the source template is correct.

**cmd/root_test.go**: NOT touched in this phase — already fixed during Phase 1.1's verify step (the two `TestSessionLog_*` tests), which context.md's original Phase 1.3 notes hadn't anticipated but which the user chose to fix immediately rather than deferring. No duplicate work needed here.

Verified: `go build ./...` clean, `go test ./...` fully green (zero failures anywhere, first phase where that's true), `go vet ./...` clean.

## Phase 1.3 test step (done)

No new production code in this phase (the changes ARE test/doc updates), so delegated a verification-only pass instead of new test authoring. All 3 acceptance criteria confirmed PASS via direct file inspection. `go build ./...` clean, `go test ./...` FULLY green (zero failures anywhere — first phase where that's true). Python (`py_compile`) and bash (`bash -n`) syntax checks both clean on the harbor test/solve files.

## Phase 1.3 verify step (done — all green)

`go build`, `make test`, `make lint` all clean, zero failures.

## Phase 1.3 update_plan step (done) — Milestone 1 complete

Checked off Phase 1.3's heading and all 3 criteria. Milestone 1 (interview mechanics) is fully done: Phases 1.1, 1.2, 1.3 all checked, full test suite green.

## Phase 1.3 update_changelog step (done)

Appended Phase 1.3's changelog entry. No knowledge-capture offer — the one discovery (installed skill copies lag until next init) just restates AGENTS.md's own already-documented convention, not new durable knowledge.

5 phases remain (2.1-2.3, 3.1-3.2). Continuing per prior authorization.

## Phase 2.1 implementation (done)

Replaced each of the 7 section templates' opening "Ask the user: [scripted question]" block with drafting-from-interview instructions (draft from `.spektacular/work/{{spec_name}}/interview.md` + any already-confirmed prior-section working files, present the draft, ask for confirmation or correction), leaving every other existing rule (format rules, altitude boundaries, stakeholder-readability, blackbox-level for acceptance criteria, non-duplication rules, working-file write instruction, context-directive footer) completely untouched:
- `01-overview.md`: draft 2-3 sentence overview from interview findings.
- `02-requirements.md`: draft testable behaviours from interview findings.
- `03-acceptance_criteria.md`: draft pass/fail conditions per requirement (from requirements.md + interview.md).
- `04-constraints.md`: draft hard constraints from interview findings; also updated the "before accepting no constraints, check common sources" paragraph to check against interview findings first, only asking the user directly for gaps, and to draft-then-confirm an empty section rather than blindly "capture their response."
- `05-technical_approach.md`: draft technical direction from interview findings; updated "Capture their response, if blank note nothing decided" to "draft the section as saying so plainly" (same treatment).
- `06-success_metrics.md`: draft success metrics from interview findings; same empty-section treatment.
- `07-non_goals.md`: adapted the section's PRE-EXISTING "propose inferred non-goals, get explicit confirmation" pattern (this section already had a draft-like flow before this phase) to draft from interview findings as well as prior spec sections; kept the "drafting is not deciding" / explicit-confirmation-required framing intact since it maps directly onto the new model.

Did NOT touch `templates/steps/spec/08-verification.md` (Phase 2.2's job) or `00b-interview.md` (already correct from Phase 1.1/1.2).

Verified: `go build ./...` clean, `go test ./...` fully green including both critical contract tests (`TestContextDirectivePresent`, `TestSpecPlanGatheringStepsReferenceWorkDir`). Confirmed via grep the harbor Python suite asserts nothing about the old "Ask the user"/scripted wording, so no harbor test needed updating for this phase specifically (Phase 2.3 owns the new cross-section-amendment scenario).

## Phase 2.1 test step (done)

New `templates/section_drafting_test.go`: `TestSectionStepsDraftFromInterviewAndConfirm` (iterates all 7 templates, asserts interview.md reference + exact draft-confirm sentence present, asserts 6 old scripted-question fragments absent) and `TestInterviewAndVerificationStepsUntouched` (guards 00b-interview.md/08-verification.md don't carry the new marker — scope boundary check). Verified directly: build clean, new tests pass, full suite green.

## Phase 2.1 verify step (done — all green)

## Phase 2.1 update_plan step (done)

## Phase 2.1 update_changelog step (done). 4 phases remain (2.2, 2.3, 3.1, 3.2).

## Phase 2.2 implementation + test (done — wrote both directly, no delegation needed)

Added identical "**If the user rejects this draft.**" paragraph to all 8 gathering-step templates (00b-interview.md, 01-overview.md through 07-non_goals.md), positioned after the goto command and before the context-directive footer's `---` separator. Exact wording per context.md's Phase 2.2 spec, verbatim across all 8. Verified via grep: exactly 1 occurrence in each of the 8, 0 in 08-verification.md.

New `templates/rejection_repair_directive_test.go`: `TestRejectionRepairPresent` (iterates all 8 templates via a fixed list — not a directory walk, since 08-verification.md and other non-gathering templates must NOT carry this marker — asserts marker present exactly once) and `TestRejectionRepairAbsentFromVerification` (guards 08-verification.md doesn't carry it, scope boundary check mirroring Phase 2.1's `TestInterviewAndVerificationStepsUntouched` pattern).

Verified: build clean, new tests pass, full suite green (including Phase 2.1's tests — confirmed the two rejection/draft markers are distinct strings and don't interfere with each other).

## Phase 2.2 test step (done)

Re-checked Phase 2.2's 4 acceptance criteria against plan.md. Criterion 3 ("edits to more than one section, or none, not bound to exactly one change") wasn't fully covered by the initial paragraph wording, so strengthened all 8 templates with one additional sentence: "The follow-up conversation may surface edits to more than one section, or conclude that nothing needs to change after all — do not assume the fix is exactly one edit to exactly the section under review." Verified via grep: exactly 1 occurrence in each of the 8 files. Existing `rejectionRepairMarker` test string unaffected (it's a substring that's still present unchanged). Full suite re-verified green after the strengthening.

All 4 criteria now covered: (1) ask-why-first — original paragraph; (2) cross-section editing, no new confirmation gate — original paragraph; (3) unbounded edit count — new sentence; (4) template unit tests — `templates/rejection_repair_directive_test.go` written during implementation.

## Phase 2.2 verify + update_plan steps (done — all green, all 4 criteria checked)

## Phase 2.2 update_changelog step (done). 3 phases remain (2.3, 3.1, 3.2).

## Phase 2.3 implementation (done) — resolved the solve.sh Open Question

**Open Question resolved**: `tests/harbor/spec-workflow/tests/test.sh` just runs `pytest test_spec_workflow.py` — it never invokes `solution/solve.sh`. No `task.toml` or other harbor config references `solve.sh` either. `solve.sh` is a Harbor-framework convention: a standalone "known-good solution" the framework can run independently to sanity-check the task setup, entirely separate from the live-agent-driven pytest suite. Conclusion: `solve.sh` does NOT need to demonstrate the cross-section-amendment scenario (a scripted non-interactive CLI walkthrough has no "rejection" to script against) — noted this explicitly as a comment in solve.sh rather than leaving it silently unaddressed. This resolves the plan's stated Open Question in favor of the "note as agent-only" fallback it already named.

**Scenario designed**: Added a "A rejection that reveals a missing requirement" section to `instruction.md` — scripts the test-driving agent to reject the Constraints draft, explain (when asked why) that token revocation must be supported (a Requirements-level behavior, not a Constraints-level boundary), and expect the agent to fold this into the already-confirmed Requirements working file without re-opening/re-confirming Requirements in the moment.

**Assertions added**: New `TestCrossSectionAmendment` class in `test_spec_workflow.py` (after `TestConstraintsStep`): `test_requirements_mentions_revocation` (checks final spec's Requirements section for "revoc", via the persisted assembled spec — NOT the working files, which are deleted by `08-verification.md`'s cleanup step before the harness ever gets to inspect them) and `test_requirements_not_reconfirmed_after_constraints` (asserts at most one `spec goto requirements` call in the transcript, proving no fresh confirmation gate reopened for the amended section).

**Files changed**: `tests/harbor/spec-workflow/instruction.md`, `tests/harbor/spec-workflow/tests/test_spec_workflow.py`, `tests/harbor/spec-workflow/solution/solve.sh` (comment only, functionally unchanged).

Verified: `go build`/`go test ./...` unaffected (pure harbor-suite work, no Go code touched) and fully green; Python syntax (`py_compile`) and bash syntax (`bash -n` on both solve.sh and test.sh) all clean.

## Phase 2.3 test step (done — no delegation needed, test authoring already done during implementation)

Re-verified all 3 criteria against plan.md: (1) adaptive-interview-opens — already covered by Phase 1.3's `TestInterviewStep` + updated `TestOverviewStep.test_tool_called`; (2) cross-section-amendment observable — new `TestCrossSectionAmendment` (2 assertions); (3) per-section content checks unaffected — confirmed I didn't touch any existing `test_section_has_content`/`test_content_is_relevant` assertions across the other step classes. Python syntax re-confirmed clean.

## Phase 2.3 verify + update_plan steps (done) — Milestone 2 complete

Milestones 1 and 2 fully done (Phases 1.1-2.3). Only Milestone 3 remains: docs site work in the SEPARATE spektacular-website repo (local path `../spektacular-website`, confirmed via `go run . repo list` earlier by a research sub-agent).

## Phase 2.3 update_changelog step (done). Knowledge capture: harbor working-file-cleanup-timing gotcha saved (project scope, gotchas/harbor-working-files-deleted-before-inspection.md).

2 phases remain: 3.1, 3.2 — BOTH entirely in the separate `docs` repo (spektacular-website), confirmed materialized locally at /home/nicj/code/github.com/jumppad-labs/spektacular-website via `go run . repo list`.

## Phase 3.1 implementation (done — in the docs repo, /home/nicj/code/github.com/jumppad-labs/spektacular-website)

**Deviation from context.md's Phase 3.1 notes**: `PipelineStage.astro` only supports a default slot and `slot="body"` — it does NOT support `slot="example"` (that's `SpecFormat.astro`-only, confirmed by reading the component source). Rather than extend the component (out of scope for a docs-content plan), embedded both example exchanges as plain fenced code blocks directly within the existing `Fragment slot="body"`, matching the site's existing convention of fenced blocks inside body prose.

- `src/pages/how-it-works.mdx` Stage 1 "Specification" body (`Fragment slot="body"`): fully rewritten per plan.md's Content outline — opening line naming Flipped Interaction with prompt-engineering-research attribution, first example exchange (verbatim from the plan's illustrative copy), stopping-condition sentence, cross-repo worked example (verbatim from the plan, linking to `/repo-configuration/`, confirmed that page exists), transition sentence covering draft-and-confirm + rejection-repair, closing spec-worthy-discussion paragraph preserved unchanged.
- `PipelineNode` sub-text for the "New" step (line ~221) changed from "structured markdown spec template" to "adaptive interview, then drafted sections" (the optional cosmetic tweak, taken).
- Quick-start Step 3 (lines ~60-76): added one sentence noting the interview happens first, adaptive questions, before section drafting/confirmation.

Verified: `npx astro check` — 0 errors, 0 warnings (3 pre-existing unrelated hints in files I didn't touch: `main.js`/`Shell.astro` `document.execCommand` deprecation notices). `npm run build` succeeds, `how-it-works.mdx` builds to `/how-it-works/index.html` without error, all 12 pages build.

## Phase 3.1 test step (done — no Go tests apply, docs repo verified via its own toolchain)

Re-confirmed all 3 acceptance criteria via grep: "Flipped Interaction" present, "prompt-engineering research" attribution present, cross-repo example present. Matches plan.md's own Testing Strategy note that Phase 3.1/3.2 are verified by npm run build / npx astro check, not this repo's Go suite.

## Phase 3.1 verify + update_plan steps (done — all criteria checked)

## Phase 3.1 update_changelog step (done, used `docs: ` prefix on file path). 1 phase remains: 3.2 (the LAST phase).

## Phase 3.2 implementation + verify (done — in the docs repo)

Added a new `FeatureCard` to `src/pages/index.mdx`'s `FeaturesGrid`, positioned right after "Spec-driven workflow" (chose add-new over reframe, since the existing card is about the overall pipeline, not specifically the interview): icon 💬, title "Flipped Interaction interview", body copy taken verbatim from the plan's content outline illustrative text. `FeatureCard.astro` needed no changes (existing icon/title props sufficient).

Verified: `npx astro check` 0 errors/0 warnings (same 3 pre-existing unrelated hints), `npm run build` succeeds, all 12 pages build including index.mdx. Confirmed via grep the exact title text appears once.

No Go tests apply (docs-only phase, same as 3.1) — docs repo's own toolchain is Phase 3.2's test/verify step per plan.md's Testing Strategy.

## Phase 3.2 update_plan step (done — both criteria checked). ALL 8 phases now complete.

## All 8 phases complete, changelog entries for all 8 written to plan.md's inline ## Changelog section. Confirmed 0 unchecked phases remain via grep.

## update_repo_changelog step (done)

Prepended `### FINAL SUMMARY` block to plan.md's `## Changelog` section (above all 8 per-phase entries) per the update-changelog skill's own lifecycle instructions — the FSM step itself didn't explicitly call for this, but it's the skill's documented final-invocation convention, so did it before writing repo-level changelogs.

Updated BOTH affected repos' root CHANGELOG.md (work spanned two repos): `/home/nicj/code/github.com/jumppad-labs/spektacular/CHANGELOG.md` (CLI/behavior-focused summary) and `/home/nicj/code/github.com/jumppad-labs/spektacular-website/CHANGELOG.md` (docs-page-focused summary, matching that repo's own changelog convention of describing what the pages now say). Both prepended above existing top entry, `## 000043_flipped-interaction-spec-interview` header per plan slug.

## test_plan step (done)

Wrote `test-plan.md` with 3 manual verification procedures, matching plan.md's Testing Approach's 3 "Manual — captured in the implementation test plan" success metrics: (1) vague-starting-point spec quality — real fresh-project session procedure; (2) corrections-decrease-over-time trend — informal longitudinal tracking across next 5-10 real sessions; (3) docs positioning as genuine differentiator — cold-read comprehension check by a second person. Each grounded in the actual implementation (real commands, real file paths, real repo). Confirmed written via plan file read.

## update_feature_changelog step (done)

Wrote central changelog record (`changelog file write 000043_flipped-interaction-spec-interview.md`) grounded in the spec's Overview + plan's `## Changelog` phase entries. Wrote derived entry for the `docs` repo (the only affected registered repo besides the colocated one), via `--repo docs`, with the required "Derived from project..." reference line — confirmed the CLI auto-stamped provenance frontmatter (project/spec/plan) correctly.

## reconcile_spec step (done)

Judged all 26 spec checkboxes (13 Requirements + 13 Acceptance Criteria) against the plan's Changelog phase entries. Checked 23, left 3 unchecked deliberately — all three are genuinely emergent/conversational-quality criteria with no direct e2e proof, conservatively left unchecked per the step's own "if in doubt, leave unchecked" instruction:
- "Interview questions vary with the answers given" — no test proves two different sessions produce different questions.
- "Interview ends without exhausting every possible question" — same category, qualitative/emergent.
- "A single-repo-focused feature prompts a cross-repo question" — the harbor e2e suite's scenario tests cross-section-amendment, not this specific cross-repo-question trigger condition; no direct proof exists.

These three exactly match the plan's own "Deliberate gaps" note in Testing Approach ("not unit-testable prose judgement... covered by... manual review of the actual first real usage") and are exactly the kind of thing the test-plan.md manual procedures (#1 vague-starting-point session, #2 corrections-trend) are meant to observe over real usage. Confirmed final write: 23 checked.

Next: finished step — the terminal step of the implement workflow.
