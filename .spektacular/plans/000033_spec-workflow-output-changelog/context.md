# Context: spec-workflow-output-changelog

## Current State Analysis

The implement workflow's FSM step list is defined at `internal/steps/implement/steps.go:16-30` as eleven steps ending in `test_plan` (`steps.go:27`) → `finished` (`steps.go:28`), a terminal state with no `goto` emitted (`internal/steps/implement/steps_test.go:296-299`, `TestFinishedStepEmitsNoGoto`). No implement-step callback ever writes to `workflow.Data` — the only key ever set is `"name"`, populated once by `cmd/implement.go:141` in `runImplementNew`. All richer per-step template variables (`plan_path`, `context_path`, `research_path`, `plan_dir`, `plan_name`, `changelog_section_name`) are computed fresh on every render by `internal/steps/implement/strategy.go:37-49` (`PathVars`), not stored in `Data`.

Two changelog-adjacent mechanisms already exist, both designed in prior plan `000015_implementation` and neither overlapping with this plan's new artifact:

1. An inline, developer-facing, phase-by-phase audit log under a `## Changelog` heading inside `plan.md` itself, written by the `update_changelog` step (`internal/steps/implement/steps.go:108-112`, template `templates/steps/implement/07-update_changelog.md`), with a fixed entry shape (`07-update_changelog.md:17-29`): **What was done**, **Deviations**, **Files changed**, **Discoveries**.
2. A terse, user-facing repo-root `CHANGELOG.md` release note, written once per plan by `update_repo_changelog` (`internal/steps/implement/steps.go:114-118`, template `templates/steps/implement/08-update_repo_changelog.md`), explicitly excluding implementation detail (`08-update_repo_changelog.md:14`).

Neither is self-contained or dense enough for a downstream doc/blog process: the inline changelog lives inside the plan store (a planning-internal document per the spec's constraint) and assumes the reader already has the rest of `plan.md`; the repo changelog is deliberately 2-4 sentences with no implementation detail. This plan adds a third, additional artifact rather than repurposing either.

The store layer (`internal/store/store.go:37-55`, `Store` interface; sole implementation `FileStore`, `store.go:59-151`) has no notion of "kind" — `spec`/`plan` are both just a `FileStore` rooted at the whole project root, with a configured sub-directory joined onto store-relative paths by each call site. `Write` auto-creates parent directories (`store.go:102-111`), so no `cmd/init.go` scaffolding is needed for a new store root. `cmd/storefile.go:17,39-124` (`newStoreFileCmd`) is a single generic factory parametrized by a `storeDirFunc` closure, already used identically by `cmd/plan_file.go:8-12` and `cmd/file.go:7-11`. There is no manifest, index, or database anywhere in the codebase (confirmed by grep across `internal/store`, `internal/steps/plan`, `internal/steps/spec`) — every existing retrieval mechanism is filename-as-identifier.

`internal/config/config.go` defines `SpecConfig`/`FileSpecConfig` (`config.go:47-56`) and `PlanConfig`/`FilePlanConfig` (`config.go:60-68`) as near-identical two-level structs, both validated from `Config.Validate()` (`config.go:155-166`) and defaulted in `NewDefault()` (`config.go:99-133`). `.spektacular/config.yaml` currently has no `changelog` key.

`workflow.Config` (`internal/workflow/workflow.go:17-27`) carries exactly `Command`, `Kind`, `DryRun`, `SpecDir`, `PlanDir` — confirmed by direct read, no spare field exists for a third directory.

The plan workflow's own `strategy` (`internal/steps/plan/strategy.go:9-32`) already computes a `spec_path` template variable from `instanceName` (`strategy.go:22`: `specPath := filepath.Join(storeRoot, s.specDir, instanceName+".md")`) — i.e. it already assumes and relies on "plan slug equals spec filename stem" as a live, working convention. The implement workflow's own strategy (`internal/steps/implement/strategy.go`) has no equivalent `spec_path` variable today; this plan adds one, cloning the plan workflow's exact approach rather than inventing a new lookup.

The CLI-level test harness in `cmd/implement_test.go` (`setupImplementCmd`, `implement_test.go:51-63`) drives real `rootCmd.Execute()` calls (`implement new`/`implement goto`) against a temp `.spektacular` directory seeded by `writeFixturePlan` (`implement_test.go:17-38`, which writes a minimal `plan.md` with a `## Milestones & Phases` section and mixed checked/unchecked phase checkboxes) and asserts on the JSON result. `TestImplementGoto_AdvancesThroughStep` (`implement_test.go:124-144`) is the existing pattern for walking multiple steps in sequence and is the direct model for Phase 2.1/2.2's new harness test.

## Per-Phase Technical Notes

### Phase 1.1: Add changelog configuration

**File changes**:
- `internal/config/config.go:14-37` — add `DefaultChangelogDir = ".spektacular/changelog"` alongside the existing `DefaultSpecDir`/`DefaultPlanDir` constants (`config.go:28,31`).
- `internal/config/config.go:58-68` — add `ChangelogConfig`/`FileChangelogConfig` types, cloned field-for-field from `PlanConfig`/`FilePlanConfig` (`config.go:60-68`).
- `internal/config/config.go:89-96` — add `Changelog ChangelogConfig \`yaml:"changelog"\`` field to the top-level `Config` struct, alongside `Spec`/`Plan` (`config.go:93-94`).
- `internal/config/config.go:99-133` (`NewDefault()`) — add changelog population mirroring the `Plan: PlanConfig{...}` block (`config.go:112-117`): `Provider: ProviderFile`, `Config: FileChangelogConfig{Directory: DefaultChangelogDir}`.
- `internal/config/config.go:154-166` (`Config.Validate()`) — add `if err := c.Changelog.Validate(); err != nil { return err }`, following the existing `c.Plan.Validate()` call (`config.go:159-161`).
- `internal/config/config.go:185-195` — add `func (c ChangelogConfig) Validate() error`, cloned from `PlanConfig.Validate()` (`config.go:187-195`): reject unsupported provider, reject empty directory.
- `internal/config/config_test.go` — add tests mirroring existing `PlanConfig` test cases: `NewDefault()` populates the changelog section, `Validate()` rejects bad provider/empty directory.

**Complexity**: Low
**Token estimate**: ~8k tokens
**Agent strategy**: Low — single agent, sequential execution. This is a mechanical clone of an existing, well-understood pattern (`PlanConfig` → `ChangelogConfig`) touching one file plus its test file.

### Phase 1.2: Add `changelog file` commands

**File changes**:
- `cmd/changelog.go` (new file) — define `changelogCmd`, cloned from `planCmd` (`cmd/plan.go:38-41`): `Use: "changelog"`, a short description, no `RunE` (parent command only, subcommands attach via `AddCommand`).
- `cmd/changelog_file.go` (new file) — wire the generic factory, cloned from `cmd/plan_file.go:1-12`:
  ```go
  package cmd

  import "github.com/jumppad-labs/spektacular/internal/config"

  func init() {
      changelogCmd.AddCommand(newStoreFileCmd(
          "Read and write files in the changelog store",
          func(c config.Config) string { return c.Changelog.Config.Directory },
      ))
  }
  ```
- `cmd/root.go:221-226` — add `rootCmd.AddCommand(changelogCmd)` alongside the existing `AddCommand` calls for `specCmd`/`planCmd`/`implementCmd`.
- No changes needed to `cmd/storefile.go` (`newStoreFileCmd`, `storefile.go:39-124`) or `internal/store/store.go` — both are reused unchanged.
- Test file `cmd/changelog_file_test.go` (new) — mirror whatever test pattern covers `plan file`/`spec file` round-tripping today; clone its shape for `changelog file write/read/list`.

**Complexity**: Low
**Token estimate**: ~6k tokens
**Agent strategy**: Low — single agent, sequential execution. Two new small files plus a one-line `root.go` addition, directly cloned from an existing, working pattern with zero new logic.

### Phase 2.1: Add the `update_feature_changelog` step to the implement workflow

**File changes**:
- `internal/workflow/workflow.go:17-27` — add `ChangelogDir string` field to `Config` struct, alongside `SpecDir`/`PlanDir` (`workflow.go:25-26`).
- `cmd/implement.go:137` and `cmd/implement.go:204` — both `workflow.Config{...}` construction sites gain `ChangelogDir: cfg.Changelog.Config.Directory`, alongside the existing `SpecDir`/`PlanDir` fields.
- `internal/steps/implement/strategy.go:13-27` — add `ChangelogFilePath(dir, name string) string { return dir + "/" + name + ".md" }`, alongside `PlanFilePath`/`ContextFilePath`/`ResearchFilePath`. This is a flat `<dir>/<name>.md` shape (no subdirectory), unlike `PlanFilePath`'s `<dir>/<name>/plan.md` — per the confirmed "one flat file per feature" storage decision.
- `internal/steps/implement/strategy.go:29-33` (`strategy` struct) — add `changelogDir string` and `specDir string` fields alongside `planDir`. `specDir` mirrors `internal/steps/plan/strategy.go:13` (the plan workflow's own `strategy` struct already carries a `specDir` for exactly this purpose — computing `spec_path` from the instance name, confirmed at `internal/steps/plan/strategy.go:22`); the implement workflow currently has no `spec_path` template variable at all, so this also newly exposes one there, reusing the plan workflow's proven "spec slug == plan slug" convention rather than inventing a new lookup.
- `internal/steps/implement/strategy.go:37-49` (`PathVars`) — add `changelogPath := filepath.Join(storeRoot, ChangelogFilePath(s.changelogDir, instanceName))` and `specPath := filepath.Join(storeRoot, s.specDir, instanceName+".md")` (cloned from `internal/steps/plan/strategy.go:22`), then add `"changelog_path": changelogPath` and `"spec_path": specPath` to the returned map.
- `internal/steps/implement/steps.go:44-56` (`writeStep`) — the `strategy{planDir: cfg.PlanDir}` construction (`steps.go:50`) becomes `strategy{planDir: cfg.PlanDir, changelogDir: cfg.ChangelogDir, specDir: cfg.SpecDir}` (`cfg.SpecDir` already exists on `workflow.Config`, per `internal/workflow/workflow.go:25` — only `ChangelogDir` is new).
- `internal/steps/implement/steps.go:16-30` (`Steps()`) — change `{Name: "finished", Src: []string{"test_plan"}, ...}` (`steps.go:28`) to `{Name: "finished", Src: []string{"update_feature_changelog"}, Dst: "finished", Callback: finished()}` and insert `{Name: "update_feature_changelog", Src: []string{"test_plan"}, Dst: "update_feature_changelog", Callback: updateFeatureChangelog()}` immediately before it.
- `internal/steps/implement/steps.go:120-129` (`testPlan()`) — change its `writeStep` call's `NextStep` argument from `"finished"` to `"update_feature_changelog"` (`steps.go:127`, currently `writeStep("test_plan", "finished", ...)`).
- `internal/steps/implement/steps.go` (new function, placed after `testPlan()` and before `finished()`) — add `updateFeatureChangelog()`, cloned from `testPlan()`'s shape (`steps.go:125-129`):
  ```go
  func updateFeatureChangelog() workflow.StepCallback {
      return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
          return "", writeStep("update_feature_changelog", "finished", "steps/implement/10-update_feature_changelog.md", data, out, st, cfg, nil)
      }
  }
  ```
  Template numbering (`10-update_feature_changelog.md` vs. renumbering `10-finished.md` to `11-`) is a free implementer choice with no behavioral effect, since templates are referenced by explicit path in `writeStep`, not directory sort order.
- `templates/steps/implement/10-update_feature_changelog.md` (new file) — modeled directly on `templates/steps/implement/09-test_plan.md`: Step 1 reads the feature's spec (`{{config.command}} spec file read {{plan_name}}.md`, using the new `spec_path`-backing convention added to the strategy above) and the plan's accumulated `{{changelog_section_name}}` section (`{{config.command}} plan file read {{plan_name}}/plan.md`); Step 2 has the agent author a self-contained record covering what was built, why it matters, what it enables, and any deviations from the plan (surfaced via the already-recorded **Deviations** fields); Step 3 stages to `.spektacular/tmp/changelog_record.md`, commits via `{{config.command}} changelog file write {{plan_name}}.md --from .spektacular/tmp/changelog_record.md`, removes the scratch file; includes a STOP-on-mismatch guard (e.g. if the spec file cannot be found under that name) per the convention in every other non-terminal template (`templates/steps/implement/08-update_repo_changelog.md:24-26`); ends with the standard Advance block and the context.md-refresh footer every template carries.
- `internal/steps/implement/result.go` — no change expected; `Result{Step, PlanPath, PlanName, Instruction}` already carries what the new step needs to report.
- `internal/steps/implement/steps_test.go:45-63` (`TestStepsOrderMatchesExpected`) — update the `expected` slice to insert `"update_feature_changelog"` between `"test_plan"` and `"finished"`.
- `internal/steps/implement/steps_test.go:77-114` (`TestFSMWalkFromNewToFinished`) — add `wf.Goto("update_feature_changelog")` and its assertion between the existing `wf.Goto("test_plan")` and `wf.Goto("finished")` calls (`steps_test.go:108-113`).
- `internal/steps/implement/steps_test.go:296-299` (`TestFinishedStepEmitsNoGoto`) — unaffected in intent (still asserts `finished` is terminal), but confirm it still passes with the new `Src`.
- New test in `internal/steps/implement/steps_test.go` — add template-content assertions for `10-update_feature_changelog.md`, mirroring existing tests like `TestUpdateChangelogStepSpecifiesEntryFields` (`steps_test.go:232-243`): assert the template mentions reading the spec, reading the plan's changelog section, the correct scratch path, and the correct `changelog file write` command.
- New CLI-harness test in `cmd/implement_test.go`, following `TestImplementGoto_AdvancesThroughStep`'s shape (`implement_test.go:124-144`): extend `writeFixturePlan` (`implement_test.go:17-38`) if needed so the fixture plan has a matching fixture spec available, walk a fixture plan through `implement new` then successive `implement goto` calls up to `update_feature_changelog`, and assert the returned `instruction` references the expected changelog path/command. This is the harness-level "changelog checks" the user asked for explicitly.

**Complexity**: High
**Token estimate**: ~45k tokens
**Agent strategy**: High — this phase touches the FSM sequencing (steps.go), the path-computation strategy (strategy.go), the workflow config plumbing (workflow.go, implement.go), a new template, and three layers of tests (package-level structural, package-level template-content, CLI-harness). Recommended approach: one agent does the Go plumbing (workflow.go → strategy.go → steps.go, in that dependency order, since each layer depends on the field/function the previous one adds), a second agent (can run in parallel once the plumbing agent has defined the new step name and template path) authors the new template modeled on `09-test_plan.md`, then a final integration pass adds and runs all three test layers together once both are in place — sequential integration after parallel analysis, per the High-complexity pattern (`go run . skill spawn-implementation-agents`).

### Phase 2.2: Report the changelog record's location when the workflow finishes

**File changes**:
- `templates/steps/implement/10-finished.md:1-19` (file path likely renumbers per Phase 2.1's note) — add one bullet to the "### Summary" list (after the existing repo-`CHANGELOG.md` bullet, `10-finished.md:9`): `- A changelog record for this feature has been written to {{changelog_path}}.` — reusing the `changelog_path` template variable added in Phase 2.1.
- `internal/steps/implement/steps_test.go` — add or extend a template-content test asserting `10-finished.md` (or its renumbered filename) mentions `{{changelog_path}}` in its summary, following the style of existing template-content tests (e.g. `TestReadPlanStepMentionsChangelog`, `steps_test.go:161-167`).
- `cmd/implement_test.go` — extend the CLI-harness test added in Phase 2.1 (or add a small follow-on assertion) to walk one step further to `finished` and assert the rendered instruction mentions the changelog record's location, completing the harness-level walk described in the Testing Approach.

**Complexity**: Low
**Token estimate**: ~5k tokens
**Agent strategy**: Low — single agent, sequential execution. A one-line template addition plus two small test extensions, entirely dependent on Phase 2.1 having already landed (`changelog_path` must exist as a template variable first).

## Project References

- **Spec `000033_spec-workflow-output-changelog.md`** (`.spektacular/specs/000033_spec-workflow-output-changelog.md`) — the spec this plan implements. Its Constraints section (line 34) is the source of the "must not replace or repurpose plan.md/research.md" rule driving the "additional artifact, not a fold-in" architecture decision.
- **Plan `000015_implementation`** (`.spektacular/plans/000015_implementation/plan.md`) — designed both existing changelog mechanisms (inline `plan.md` audit log, repo-root `CHANGELOG.md`) this plan builds alongside without modifying. Read in full during discovery; see research.md § Prior plans / specs consulted.
- **Plan `000021_file-writing`** (`.spektacular/plans/000021_file-writing/plan.md`) — established the stage-to-`.spektacular/tmp/`-then-`--from`-flag write convention every store-writing step, including this plan's new step, must follow.
- **Plan `000014_spektacular_store`** (`.spektacular/plans/000014_spektacular_store/plan.md`) — original introduction of the `Store` interface this plan depends on unchanged.
- **Spec `000032_spec-workflow-pair-programming-enhancements.md`** (`.spektacular/specs/000032_spec-workflow-pair-programming-enhancements.md`, spec only, not yet planned) — the intended first real feature to be captured by the changelog this plan builds, once this plan ships and 000032 is subsequently planned and implemented. Its own Non-Goals section explicitly defers "a durable changelog/context artifact" to this plan (000033).
- **Knowledge doc `architecture/workflow-steps.md`** (`.spektacular/knowledge/architecture/workflow-steps.md`) — general step-authoring architecture reference; conceptually accurate but has a stale package-path reference (`internal/spec` vs. the actual `internal/steps/spec`) not corrected as part of this plan.

## Testing Strategy

Two existing test layers are extended, no new layer introduced:

- **Package-level (`internal/steps/implement/steps_test.go`)**: FSM structural assertions (step order, transition guards, full-walk) and template-content assertions (string/directive checks) for the new `update_feature_changelog` step and the updated `finished` step. Also extended for `internal/config/config_test.go` (new `ChangelogConfig` validation/defaults) and `internal/steps/implement` strategy tests (`changelog_path`/`spec_path` computation, mirroring `TestPlanFilePaths_UseConfiguredDirectory`).
- **CLI-level (`cmd/implement_test.go`, `cmd/changelog_file_test.go`)**: the `setupImplementCmd` harness extended to walk a fixture plan through `update_feature_changelog` and `finished`, asserting on rendered instruction content; a new `changelog_file_test.go` for `changelog file read/write/list` round-tripping, mirroring existing plan/spec file command tests.

No agent-driven integration test is added or expected — consistent with every other implement step in this codebase, none of which are tested by actually running an LLM agent. See plan.md § Testing Approach for the full rationale and the spec success-metric classification.

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|-------------|----------------|
| Low | ~10k | Single agent, sequential |
| Medium | ~25k | 2-3 parallel agents |
| High | ~50k+ | Parallel analysis, sequential integration |

Phase 1.1 and 1.2: Low (~8k, ~6k). Phase 2.1: High (~45k) — the only phase warranting parallel analysis before sequential integration, per its technical notes above. Phase 2.2: Low (~5k), strictly sequential after 2.1.

## Migration Notes

None. No existing data is migrated or reshaped — the changelog store is entirely new and starts empty; the two existing changelog mechanisms (inline plan audit log, repo `CHANGELOG.md`) are untouched.

## Performance Considerations

None identified. All new I/O is file reads/writes through the existing `Store`/`FileStore` abstraction, at the same frequency (once per implement-workflow completion) as the existing `update_repo_changelog` and `test_plan` steps already perform. No new hot path, no new concurrency, no new external call.
