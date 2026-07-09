# Context: 000034_spec-plan-implement-reconciliation

## Current State Analysis

The `implement` workflow's step sequence today (`internal/steps/implement/steps.go:16-31`) is: `new → read_plan → analyze → implement → test → verify → update_plan → update_changelog → update_repo_changelog → test_plan → update_feature_changelog → finished`. Nothing in this sequence ever reads the originating specification except `update_feature_changelog` (`templates/steps/implement/10-update_feature_changelog.md:5-14`), and even there only to source the changelog record's "why it matters" framing — never to check off or reconcile against the spec's own Requirements/Acceptance Criteria checkboxes. The `plan` workflow's `verification` step (`internal/steps/plan/steps.go:44`, `templates/steps/plan/14-verification.md`) checks structural completeness of the assembled plan documents only, never content-vs-spec fidelity.

`read_plan` (`internal/steps/implement/steps.go:19`, template `templates/steps/implement/01-read_plan.md`) is already the implement workflow's validation-and-drift gate — explicitly documented as "nothing else runs until it passes" (line 3). It performs: full plan read (Step 1), structural validation of required `## ` sections and phase-checkbox shape (Step 2), a drift check against the working tree with a STOP-and-three-option prompt (Step 3, lines 40-59), and changelog-mode detection (Step 4, lines 61-66).

`internal/steps/implement/strategy.go:47-63` (`PathVars`) already computes `spec_path` (line 61, derived as `{specDir}/{instanceName}.md`) and `plan_name` (line 58) unconditionally for every implement step — no plumbing changes needed for any new or extended step to read the spec.

`cmd/file.go:8-11` already registers `spec file read/write/list` on `specCmd` via the shared `newStoreFileCmd` builder — the same generic command-building mechanism backing `plan file` and `changelog file`. `spec file write <name>.md --from <source-path>` is available today with the established stage-to-`.spektacular/tmp/`-then-commit contract.

No Go code anywhere in this codebase parses markdown checkboxes (`- [ ]`/`- [x]`) today — every existing example of judgment against markdown content (e.g. `update_changelog`'s phase-completion check, `10-update_feature_changelog.md`'s deviation summarization) is agent-driven prose in a template, not deterministic Go parsing.

The most directly relevant prior work is plan `000033_spec-workflow-output-changelog` (`.spektacular/plans/000033_spec-workflow-output-changelog/`), which added the `changelog` document store and the `update_feature_changelog`/`finished` step pair to the implement FSM — the same category of change (new step(s) near the tail of `implement`, extending the terminal report) that this plan makes. Its `10-update_feature_changelog.md` template is the closest existing structural sibling to this plan's new `reconcile_spec` step.

## Per-Phase Technical Notes

### Phase 1.1: Extend read_plan with spec coverage check

**File changes**:
- `templates/steps/implement/01-read_plan.md` — insert a new "Step 3.5" (or renumber existing Step 3/4 as needed) between the existing "Step 3: Drift check against the working tree" (lines 40-59) and "Step 4: Changelog mode detection" (lines 61-66). New content: read the spec via `{{config.command}} spec file read {{plan_name}}.md` (spec_path/plan_name already available — `internal/steps/implement/strategy.go:52,58`), enumerate every `- [ ]`/`- [x]` line under the spec's `## Requirements` and `## Acceptance Criteria` headings, and for each confirm it is referenced (by paraphrase or explicit mention) somewhere under the plan's `## Milestones & Phases` phase summaries or acceptance criteria. Model the STOP-and-prompt UX on the existing Step 3 mismatch prompt (lines 53-59) but with exactly two options: (1) fix the plan first (update `{{plan_path}}` to add the missing coverage, then restart this step), (2) accept the gap as descoped (write the marker from Phase 1.2, then continue). Before presenting the prompt, check whether the gap is already recorded as accepted (Phase 1.2's marker) and skip prompting for any gap already marked.
- `internal/steps/implement/steps_test.go` — add a test asserting the rendered `01-read_plan.md` instruction text mentions the spec-coverage check and both resolution options, following the existing pattern of `TestReadPlanTemplateDirectsStructuralValidation` (line 173) and `TestReadPlanTemplateDirectsDriftCheck` (line 194).
- No Go callback changes needed in `internal/steps/implement/steps.go` — `readPlan()` (lines 67-71) already just calls `writeStep` with the template path; the new check is entirely template content, consistent with how the existing drift-check and structural-validation checks in the same template require no Go-side logic beyond rendering.

**Complexity**: Medium — no new FSM/Go logic, but the template content is substantial (a full new gate section with a two-option branch) and must integrate cleanly with the existing Step 3/Step 4 numbering without disrupting the drift-check or changelog-mode-detection content around it.

**Token estimate**: ~15k tokens (mostly template authoring + a couple of targeted test additions; no wide codebase search needed since discovery already pinned every relevant file).

**Agent strategy**: Single agent, sequential execution. The template edit and its accompanying test are tightly coupled (the test asserts on the template's exact rendered wording) and small enough that parallelizing would add coordination overhead for no benefit.

### Phase 1.2: Durable descope marker

**File changes**:
- `templates/steps/implement/01-read_plan.md` — as part of the same Step 3.5 content from Phase 1.1, specify the marker format: a `**Descoped requirements**:` list appended near the end of the plan's `## Milestones & Phases` section (or created if absent), one bullet per accepted gap: `- <requirement/acceptance-criterion short title> — descoped: <one-line reason>`. Written the same way every other plan.md edit in this codebase is written — staged to `.spektacular/tmp/`, committed via `{{config.command}} plan file write {{plan_name}}/plan.md --from ...`, scratch file removed — mirroring the exact staging pattern already used in `templates/steps/implement/07-update_changelog.md:37-42`.
- No new Go code, no new store, no new file. This is pure template-prose specifying a markdown convention the agent authors and reads back on every `read_plan` invocation.

**Complexity**: Low — this is a formatting/convention specification within the same template edit as Phase 1.1, not a separate mechanism.

**Token estimate**: ~5k tokens (folded into the same template-authoring pass as Phase 1.1; listed separately here only because it has its own acceptance criteria in plan.md).

**Agent strategy**: Single agent, sequential execution, combined with Phase 1.1 in practice (same file, same editing pass) — kept as a separate phase in plan.md because it has independently testable acceptance criteria (gap not re-flagged on resume), not because it needs separate implementation effort.

### Phase 2.1: Add reconcile_spec step

**File changes**:
- `internal/steps/implement/steps.go:28` — insert a new `StepConfig` entry: `{Name: "reconcile_spec", Src: []string{"update_feature_changelog"}, Dst: "reconcile_spec", Callback: reconcileSpec()}` immediately after the `update_feature_changelog` entry (currently line 28) and before the `finished` entry (currently line 29).
- `internal/steps/implement/steps.go:29` — change `finished`'s `Src` from `[]string{"update_feature_changelog"}` to `[]string{"reconcile_spec"}`.
- `internal/steps/implement/steps.go` — add a `reconcileSpec()` callback function (new, placed after `updateFeatureChangelog()` at line 136-140 and before `finished()` at line 142-146), following the exact one-liner shape of every other step callback: `return "", writeStep("reconcile_spec", "finished", "steps/implement/11-reconcile_spec.md", data, out, st, cfg, nil)`.
- `templates/steps/implement/11-finished.md` → renamed to `templates/steps/implement/12-finished.md` via `git mv` (mirrors the `10-finished.md`→`11-finished.md` rename precedent from the 000033 changelog feature).
- `templates/steps/implement/11-reconcile_spec.md` — new template file, structured to mirror `templates/steps/implement/10-update_feature_changelog.md`'s four-part shape: Step 1 gather source material (`{{config.command}} spec file read {{plan_name}}.md` + `{{config.command}} plan file read {{plan_name}}/plan.md`, reusing the `{{changelog_section_name}}` phase-by-phase record exactly as `10-update_feature_changelog.md:5-14` already does); Step 2 judge each spec Requirements/Acceptance-Criteria checkbox against the phase entries' "What was done"/"Deviations" content and decide satisfied/unsatisfied+reason; Step 3 stage the updated spec (checkboxes flipped to `- [x]` where satisfied) to `.spektacular/tmp/spec_reconcile.md` and commit via `{{config.command}} spec file write {{plan_name}}.md --from .spektacular/tmp/spec_reconcile.md`, then `rm` the scratch file — mirroring `10-update_feature_changelog.md:28-34`; a STOP-on-mismatch section for a missing spec or empty/missing changelog section, mirroring `10-update_feature_changelog.md:38-40`; an Advance section calling `{{config.command}} implement goto --data '{"step":"finished"}'`; the standard trailing context.md-refresh reminder.
- `internal/steps/implement/steps_test.go` — add tests: (a) FSM wiring assertion that `reconcile_spec` is reachable only from `update_feature_changelog` and `finished` is reachable only from `reconcile_spec`, mirroring how the existing multi-source `analyze` transition is likely asserted; (b) rendered-template content assertion for `11-reconcile_spec.md` mirroring `TestUpdateChangelogStepSpecifiesEntryFields` (line 237) in style — asserts the instruction text directs reading spec + plan changelog and committing via `spec file write`.

**Complexity**: Medium — one new FSM entry (mechanical, low risk) plus one substantial new template file modeled closely on an existing sibling (`10-update_feature_changelog.md`), plus a file rename. The judgment logic itself (what counts as "satisfied") is specified in template prose, not code, keeping the Go-side change small.

**Token estimate**: ~20k tokens (new template authored from a close existing model, FSM wiring change, rename, and 2 new/extended tests).

**Agent strategy**: Single agent, sequential execution. The FSM wiring change, the rename, and the new template are all tightly coupled (the rename must happen before/alongside the FSM change since `finished`'s template path changes too) and best done as one coherent edit rather than split across parallel agents.

### Phase 2.2: Extend finished report

**File changes**:
- `templates/steps/implement/12-finished.md` (post-rename; content currently at `11-finished.md`) — extend the "Summary" section (currently lines 5-10) with a new bullet: "The specification's Requirements and Acceptance Criteria have been reconciled against the completed work; see `{{spec_path}}` for the updated checkbox state." Extend the "What to do next" section (currently lines 12-19) with a new instruction: "Report the specification's completion status: read `{{spec_path}}` via `{{config.command}} spec file read {{plan_name}}.md`, and tell the user which Requirements/Acceptance-Criteria items are now checked, and for any still unchecked, the reason recorded during `reconcile_spec` (deferred, descoped, or not attempted)." `spec_path`/`plan_name` are already available in every implement step's template variables (`internal/steps/implement/strategy.go:52,58`) — no new variable plumbing needed.
- `internal/steps/implement/steps_test.go` — extend or add a test asserting the renamed `12-finished.md`'s rendered instruction text directs reporting spec completion status, alongside its existing phase/changelog summary content.
- No Go callback changes needed beyond the rename already covered in Phase 2.1 — `finished()` (lines 142-146) already just calls `writeStep`; this phase is template-content-only.

**Complexity**: Low — small, additive template edit to an already-simple terminal step; no new Go logic, no new variables.

**Token estimate**: ~8k tokens.

**Agent strategy**: Single agent, sequential execution.

## Testing Strategy

Per-phase, testing is entirely unit-level Go tests in `internal/steps/implement/steps_test.go`, following the codebase's existing string-assertion-on-rendered-template pattern:

- **Phase 1.1 & 1.2**: one test asserting `01-read_plan.md`'s rendered instruction text mentions the spec-coverage check, both resolution options (update plan / accept as descoped), and the descoped-marker format — modeled on `TestReadPlanTemplateDirectsStructuralValidation` / `TestReadPlanTemplateDirectsDriftCheck`.
- **Phase 2.1**: an FSM-wiring test asserting `reconcile_spec` is reachable only from `update_feature_changelog` and `finished` only from `reconcile_spec`; a rendered-template test for `11-reconcile_spec.md` asserting it directs reading spec + plan changelog and committing via `spec file write`, modeled on `TestUpdateChangelogStepSpecifiesEntryFields`.
- **Phase 2.2**: a rendered-template test for the renamed `12-finished.md` asserting it directs reporting spec completion status alongside the existing changelog/phase summary.

No integration tests drive a full mocked `implement` run end-to-end (see plan.md § Testing Approach for why — the checkbox-satisfaction judgment is agent-driven, not deterministic, so a scripted integration test would hard-code the judgment being tested). The three success metrics that require observing real judgment quality are captured as manual checks in the implementation test plan (produced by the existing `test_plan` step once the code exists), not as automated tests.

## Project References

- `.spektacular/specs/000034_spec-plan-implement-reconciliation.md` — this feature's own spec; source of truth for requirements, constraints, acceptance criteria, and non-goals.
- `.spektacular/plans/000033_spec-workflow-output-changelog/plan.md` — the changelog feature; direct structural model for this plan's `reconcile_spec` step and the `finished` renumbering.
- `.spektacular/specs/000032_spec-workflow-pair-programming-enhancements.md` — the incident spec that motivated this feature; its plan silently dropped Milestone 1 and narrowed Milestone 2 without detection, and its checkboxes remain unchecked despite the linked implement run reporting `finished`.
- `research.md` (this plan's own research document) — full alternatives-considered, evidence, and rehydration cues for this feature.

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|-------------|----------------|
| Low | ~10k | Single agent, sequential |
| Medium | ~25k | 2-3 parallel agents |
| High | ~50k+ | Parallel analysis, sequential integration |

All four phases in this plan are Low or Medium complexity (see Per-Phase Technical Notes above) and are executed by a single agent sequentially within each phase — no phase in this plan requires the High tier's parallel-analysis strategy, since every insertion point and template model was already pinned down during discovery.

## Migration Notes

None. This plan adds new step content and one new FSM step; it does not change the shape or meaning of any existing plan or spec document already in the store. Plans and specs created before this feature ships are unaffected until the next time `implement` is run against them, at which point the new `read_plan` coverage check and `reconcile_spec` step apply going forward — consistent with the spec's Non-Goal that this feature is not retroactive.

## Performance Considerations

None beyond the existing cost profile of running the `implement` workflow. The new coverage check and reconciliation step each add one additional spec-file read (and, for `reconcile_spec`, one additional spec-file write) per `implement` run — negligible relative to the codebase research and code-writing work the workflow already performs per phase.
