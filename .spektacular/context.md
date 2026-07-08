# Implement working context — spec-workflow-output-changelog

## Sequencing note (carried from spec workflow)
000033 is a follow-on from 000032 (spec-workflow-pair-programming-enhancements). User's explicit
intent: plan + implement 000033 first, so the changelog capability exists in time to capture 000032
as its first real dogfood output. Do not plan/implement 000032 before 000033 unless the user
explicitly says otherwise.

## Key decisions from planning (see research.md / plan.md for full evidence)
- New store kind `changelog`, one Markdown file per feature (`.spektacular/changelog/<name>.md`),
  following the exact spec/plan `Store` + `newStoreFileCmd` pattern — no store-layer changes needed.
- New implement-workflow FSM step `update_feature_changelog` inserted between `test_plan` and
  `finished` (internal/steps/implement/steps.go), modeled directly on the `test_plan` step's shape.
- Record content authored by the agent from the spec (why) + the plan's already-accumulated inline
  `## Changelog` section (what was actually built, including recorded deviations).
- `workflow.Config` needs a new `ChangelogDir` field; `internal/config/config.go` needs a new
  `ChangelogConfig`/`FileChangelogConfig` pair cloned from `PlanConfig`/`FilePlanConfig`.
- No manifest/index needed — filename-as-identifier is the universal codebase convention.
- No `cmd/init.go` changes needed — `Store.Write` auto-creates parent dirs.
- New step lives in `internal/steps/implement` (not a new package). One flat file per feature.
- Record generation is unconditional/mandatory (new required FSM step), not optional/configurable.
- New `spec_path` template var added to implement's strategy, reusing the plan workflow's proven
  "spec slug == plan slug" convention (internal/steps/plan/strategy.go:22).

## Implement workflow — progress
- read_plan step: complete.
  - Structural validation: all 10 required `## ` sections present in plan.md; 4 phase checkboxes
    (1.1, 1.2, 2.1, 2.2) all present, all unchecked, all with valid `*Technical detail:*` links
    resolving to matching `### Phase N.M:` headings in context.md's Per-Phase Technical Notes.
  - Drift check: delegated to an Explore subagent covering 14 specific claims (FSM step list shape,
    strategy.go structs, workflow.Config fields, config.go structs/functions, cmd/ file existence,
    store.go, plan strategy's specDir precedent, template files, existing test names,
    config.yaml contents). Result: **no drift found** — plan matches current codebase exactly.
  - Changelog mode: **absent** — no `## Changelog` section in plan.md yet. This is a first-phase
    invocation; `update_changelog` step will create the section on first use. Pick up analysis at
    Phase 1.1 (first `#### - [ ]` phase).

## Plan phases to implement, in order
1. Phase 1.1: Add changelog configuration — **code + tests complete, `go test ./internal/config/...`
   passes (16 tests).** Added `DefaultChangelogDir` const, `ChangelogConfig`/`FileChangelogConfig`
   types (cloned from Plan's), `Changelog` field on `Config`, `NewDefault()` population,
   `Config.Validate()` call to `c.Changelog.Validate()`, and `ChangelogConfig.Validate()` method.
   Tests: extended `TestToYAMLFile_ProviderSectionsRoundTrip` and
   `TestFromYAMLFile_AbsentProviderSectionsUseDefaults` to cover Changelog; added
   `TestFromYAMLFile_UnknownChangelogProviderReturnsError` and
   `TestFromYAMLFile_EmptyChangelogDirectoryReturnsError` (mirroring the SpecConfig YAML-route
   pattern). Verified: `make test` and `make lint` both pass (all green). Phase 1.1 checkbox and
   both acceptance criteria marked `[x]` in plan.md. Changelog entry appended (new `## Changelog`
   section created after `## Out of Scope`, this being the first invocation for this plan). 3 phases
   remain unchecked (1.2, 2.1, 2.2) — looping back to `analyze` per Auto Mode (no pause for
   confirmation between phases).
2. Phase 1.2: Add `changelog file` commands — **code written, compiles clean, smoke-tested.**
   Analysis confirmed no drift (cmd/plan_file.go, cmd/file.go, cmd/plan.go:38-41, cmd/storefile.go,
   cmd/root.go:221-226 all matched plan exactly). Created `cmd/changelog.go` (changelogCmd, cloned
   from planCmd shape) and `cmd/changelog_file.go` (wires newStoreFileCmd against
   `c.Changelog.Config.Directory`, cloned from cmd/plan_file.go). Added `rootCmd.AddCommand(changelogCmd)`
   in cmd/root.go between planCmd and implementCmd. Verified `go run . changelog file --help` shows
   read/write/delete/list subcommands correctly. Tests: new `cmd/changelog_file_test.go`, cloned from
   `cmd/plan_file_test.go`'s full-CLI-harness pattern (`rootCmd.Execute()` against a temp
   `.spektacular` dir) — `TestChangelogFileWriteRead_RoundTrips` and
   `TestChangelogFileList_ShowsAllWrittenRecords`. `go test ./cmd/...` passes. Verified: `make test`
   and `make lint` both pass (all green). Phase 1.2 checkbox and both acceptance criteria marked
   `[x]` in plan.md. Changelog entry appended after the Phase 1.1 entry (section already existed).
   Milestone 1 (changelog store foundation) is now complete.
3. Phase 2.1: Add `update_feature_changelog` FSM step — **plumbing code complete, compiles clean,
   `go run . implement steps` confirms correct ordering** (`... test_plan, update_feature_changelog,
   finished`). Ran 3 parallel Explore research agents (FSM plumbing, template model, test layers) per
   spawn-implementation-agents guidance for this High-complexity phase — all returned precise
   verbatim file maps, no drift found anywhere.
   - `internal/workflow/workflow.go`: added `ChangelogDir string` field to `Config`.
   - `internal/steps/implement/strategy.go`: added `ChangelogFilePath(dir, name) string` helper
     (flat `<dir>/<name>.md`, NOT a subdirectory like PlanFilePath). Added `changelogDir`/`specDir`
     fields to `strategy` struct. Added `changelog_path`/`spec_path` to `PathVars` — `spec_path`
     computed identically to the plan workflow's own precedent
     (`internal/steps/plan/strategy.go:22`: `filepath.Join(storeRoot, s.specDir, instanceName+".md")`).
   - `internal/steps/implement/steps.go`: inserted `update_feature_changelog` StepConfig between
     `test_plan` and `finished` in `Steps()`. Updated `writeStep`'s strategy literal to populate the
     2 new fields from `cfg`. Changed `testPlan()`'s NextStep from `"finished"` to
     `"update_feature_changelog"`. Added `updateFeatureChangelog()` function (models `testPlan()`'s
     shape exactly). **Deviation from plan**: renamed `finished`'s template path from
     `10-finished.md` to `11-finished.md` (plan explicitly flagged this numbering as a free
     implementer choice with no behavioral effect) — used `git mv` to preserve history.
   - `templates/steps/implement/10-update_feature_changelog.md` (new): modeled on `09-test_plan.md`'s
     exact house style (STOP-on-mismatch section, `{{config.command}}` Go-template syntax, `.spektacular/tmp/`
     stage-then-commit-via-CLI pattern, identical context.md-refresh footer verbatim). Reads
     `{{plan_name}}.md` from the spec store (new) and `{{plan_name}}/plan.md`'s `{{changelog_section_name}}`
     section (existing var) — only reads the existing inline changelog section, does not create it
     (that's `update_changelog`'s job, which always runs first in this workflow).
   - `cmd/implement.go`: both `workflow.Config{...}` sites (runImplementNew, runImplementGoto) now set
     `ChangelogDir: cfg.Changelog.Config.Directory`.
   Tests: `TestStepsOrderMatchesExpected`, `TestFSMWalkFromNewToFinished`,
   `TestFSMLoopFromUpdateChangelogBackToAnalyze` extended for the new step's insertion point;
   `TestStopOnMismatchDirectivePresentInEveryNonTerminalTemplate` extended with the new step's map
   entry; new `TestUpdateFeatureChangelogStepMentionsSourcesAndCommitCommand` (steps_test.go).
   `TestImplementStatus_ReportsUncheckedPhases`/`TestImplementSteps_ListsAllSteps` step-count updated
   11→12; new `TestImplementGoto_UpdateFeatureChangelogInstructsChangelogWrite` CLI-harness test
   walking the full transition chain (cmd/implement_test.go). `go test ./internal/steps/implement/...`
   and `./cmd/...` both pass. Verified: `make test`/`make lint` all green; manual smoke tests confirmed
   `changelog file --help` wiring and `implement steps` ordering; a full end-to-end FSM walk against a
   real fixture project (spec + plan on disk, built binary, walked new→...→update_feature_changelog→
   finished) confirmed the rendered instruction correctly references the spec read, plan read, scratch
   path, and changelog write command. Phase 2.1 checkbox + all 3 acceptance criteria marked `[x]`
   in plan.md, changelog entry appended. **Phase 2.1 fully complete.**

## Remaining work
- Phase 2.2 (last phase): Report changelog record location in `finished` step's summary. Add one
  bullet to `templates/steps/implement/11-finished.md`'s `### Summary` list referencing the already-
  existing `{{changelog_path}}` template variable (added in Phase 2.1), following the exact bullet
  style of the existing repo-CHANGELOG.md bullet there. Plus template-content test + CLI-harness
  extension walking one step further to `finished`. Low complexity, strictly sequential after 2.1.
  Currently looping to `analyze` per Auto Mode.
  - **Code written**: added two bullets to `templates/steps/implement/11-finished.md`'s `### Summary`
    and `### What to do next` lists referencing `{{changelog_path}}`, matching the exact bullet style
    of the existing repo-CHANGELOG.md bullets. No Go code changes needed. `go build ./...` clean.
    Tests: new `TestFinishedStepMentionsChangelogPath` in steps_test.go asserts the rendered
    `finished()` output contains `"test.md"` (the resolved changelog_path suffix for the renderStep
    fixture — verified `.../test.md` is genuinely distinct from `plan_path`'s `.../test/plan.md`, no
    false-positive risk). No CLI-harness test added — judged redundant since the CLI harness would
    exercise the identical `writeStep`/`PathVars` code path with no new integration risk, and the
    acceptance criterion is already satisfied by the package-level test. Full `go build ./...` and
    `go test ./...` (whole repo) pass. Verified: `make test`/`make lint` all green. This is the LAST
    phase in the plan — ready for update_plan step, then final changelog entry + FINAL SUMMARY block,
    then update_repo_changelog (repo CHANGELOG.md), test_plan, and finished. Phase 2.2 checkbox +
    its acceptance criterion marked `[x]` in plan.md — **all 4 phases now complete.** Changelog entry
    appended AND FINAL SUMMARY block prepended below `## Changelog` heading (this was the last
    phase). No unchecked phases remain — advanced to `update_repo_changelog`. Prepended a
    `## spec-workflow-output-changelog` section to repo-root `CHANGELOG.md` (user-facing, 3
    sentences, no implementation detail). test_plan step: wrote `spec-workflow-output-changelog/test-plan.md`
    covering both manual success metrics (record-sufficiency, rarely-needs-supplementation), both
    grounded in the real implement/changelog CLI commands. update_feature_changelog step: **dogfooded
    the brand-new step on this very plan.** Hit the exact edge case the plan's Out of Scope section
    anticipated — `spec file read spec-workflow-output-changelog.md` 404'd, because this plan's slug
    (`spec-workflow-output-changelog`) omits the spec's numeric prefix (actual filename
    `000033_spec-workflow-output-changelog.md`) — the plan-slug-equals-spec-slug convention doesn't
    hold for specs created before slug-without-prefix became this repo's plan-naming norm. Resolved
    per the template's STOP-on-mismatch option (a): located the correct spec name
    (`go run . spec file list`) and read it directly, rather than blocking — unambiguous, mechanical
    resolution, consistent with Auto Mode. Authored and committed the changelog record to
    `changelog file write spec-workflow-output-changelog.md`, confirmed via `changelog file read`.
    Ready for finished step.
