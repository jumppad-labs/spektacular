# Plan: spec-workflow-output-changelog

<!-- Metadata -->
<!-- Created: 2026-07-08T11:10:41Z -->
<!-- Commit: 5a9cbbf -->
<!-- Branch: main -->
<!-- Repository: git@github.com:jumppad-labs/spektacular.git -->

## Overview

This plan gives Spektacular a durable, self-contained changelog record for every feature the implement workflow completes. Today, once a feature ships, the only records of what happened are a terse repo-level release note and a developer-facing, plan-store-internal audit log — neither is dense or self-contained enough to hand to a later documentation or announcement pass without re-reading the spec, plan, and conversation. This plan adds a new changelog store (one Markdown file per feature, retrievable by name) and a new step at the end of the implement workflow that automatically authors a record into it, grounded in what was actually built rather than what was originally planned.

## Conventions

No project conventions apply to this feature. The project's always-applied knowledge (`go run . knowledge always-applied`) contains only the `conventions` and `glossary` category README placeholders — no actual convention entries have been recorded in the knowledge base yet.

## Architecture & Design Decisions

This feature adds a new store kind, `changelog`, alongside the existing `spec` and `plan` stores, and a new mandatory step in the implement workflow's FSM that populates it. The changelog store follows the exact shape already established by `plan`/`spec`: a configured directory (`internal/config.ChangelogConfig`/`FileChangelogConfig`, cloned from `PlanConfig`/`FilePlanConfig`, default `.spektacular/changelog`), a `changelog file read/write/list` command family built from the existing generic `newStoreFileCmd` factory (`cmd/storefile.go`), and one flat Markdown file per feature named after the plan's slug (`.spektacular/changelog/<name>.md`). No changes are needed to the `Store` interface or `FileStore` implementation (`internal/store/store.go`) — the store layer has no notion of "kind" today; a third root directory slots in with zero store-layer code (research.md § Chosen approach). Retrieval by name is therefore free: `changelog file read <name>.md` is the entire lookup mechanism, matching every other store in this codebase (filename-as-identifier, no manifest or index — research.md § Alternatives 3–4 explain why a growing single file or an index/manifest were rejected).

The record itself is produced by a new implement-workflow step, `update_feature_changelog`, inserted into the FSM between the existing `test_plan` and `finished` steps (`internal/steps/implement/steps.go`). It runs unconditionally, once, as part of every implement run — the spec's framing ("without requiring the user to manually write it") reads as always-on, and gating it behind a config flag would add a skip-path through the FSM for a requirement nothing in the spec asks for. The new step is modeled directly on `test_plan`'s existing shape: its template instructs the agent to read the feature's spec and the plan's already-accumulated inline `## Changelog` section live via `plan file read`, author a self-contained Markdown record grounded in what was actually built, stage it at `.spektacular/tmp/`, and commit it via `changelog file write <name>.md --from ...` before advancing. Because the plan's inline changelog already records per-phase deviations from the original plan (its `**Deviations**` field), the new record's "actual outcome, not the original plan" requirement is satisfied by construction — the source material the agent reads already privileges what happened over what was intended, rather than requiring new machinery to detect drift.

Two smaller, structural changes follow from wiring the new step in: `workflow.Config` (`internal/workflow/workflow.go`) gains a `ChangelogDir` field alongside its existing `SpecDir`/`PlanDir`, sourced from the new config section the same way `PlanDir` is sourced today (`cmd/implement.go`); and the implement package's `PathStrategy` (`internal/steps/implement/strategy.go`) gains a `changelogDir` field and a computed `changelog_path` template variable, plus a `ChangelogFilePath` helper mirroring `PlanFilePath`/`ContextFilePath`. It also gains a `specDir` field and a `spec_path` template variable — a variable the implement workflow does not have today — reusing the plan workflow's already-proven "plan slug equals spec slug" convention (`internal/steps/plan/strategy.go`) rather than inventing a new lookup. The terminal `finished` step's template gains one additional summary bullet pointing at the new record's location, consistent with how it already reports the repo `CHANGELOG.md` location.

This direction was chosen over folding the record-writing logic into the existing `finished` step (which today is a pure, file-I/O-free reporting step, and conflating "author a durable artifact" into it would blur that step's single responsibility) and over making the step config-gated/optional (no requirement in the spec calls for opt-out, and it would be strictly more code — a config branch plus conditional FSM wiring — for a case nobody asked for). See `research.md#alternatives-considered-and-rejected` for the full evidence behind each rejected option, including why a single growing changelog file and a manifest/index were also rejected in favor of the flat, one-file-per-feature, filename-as-identifier convention already used everywhere else in this codebase.

## Component Breakdown

- **Changelog store configuration** — a new configuration section, parallel to the existing spec and plan sections, that names the changelog store's provider and its directory. Owns the single setting (directory location, defaulting to `.spektacular/changelog`) that every other component depends on to know where records live. Validated the same way the spec and plan sections are validated today.

- **Changelog file commands** — a new `changelog file` command group (`read`, `write`, `delete`, `list`), built from the same generic store-file-command machinery that already powers `spec file` and `plan file`. Owns all direct interaction with changelog records on disk: reading a record by feature name, writing/committing a staged record, and listing existing records. This is the only component that talks to the changelog store directly — every other component that needs to read or write a record goes through it (via the CLI, exactly as the implement workflow already does for plan documents).

- **Changelog path variables** — an extension to the implement workflow's existing path-computation logic, which already derives `plan_path`, `context_path`, and `research_path` for use in step templates. Gains two more computed paths (the changelog record's location, and the feature's spec location) and the directory values it needs to compute them, so step templates can reference both without hardcoding them.

- **`update_feature_changelog` step** — a new step in the implement workflow's sequence, positioned between the existing `test_plan` step and the terminal `finished` step. Owns producing the changelog record: it instructs the assistant to gather source material (the feature's spec and the plan's already-recorded per-phase changelog entries), author a self-contained record in prose, and commit it through the changelog file commands. This is the only component that decides *when* a record gets written and *what* goes into it; it relies on the changelog file commands for the actual write and on the changelog path variables for where to write it.

- **`finished` step (updated)** — the implement workflow's existing terminal step, gaining one additional line in its summary so it reports the new record's location to the user alongside the other artifacts it already reports (the repo-level CHANGELOG.md entry, the completed phases). No new responsibility beyond surfacing what `update_feature_changelog` produced.

- **Implement workflow sequence (updated)** — the existing ordered list of implement workflow steps, updated so the new `update_feature_changelog` step sits between `test_plan` and `finished` instead of `test_plan` transitioning directly to `finished`. This is a one-step insertion into an existing, unchanged sequencing mechanism — no new sequencing concept is introduced.

## Data Structures & Interfaces

**`ChangelogConfig` / `FileChangelogConfig`** — new configuration types, structurally identical to the existing `PlanConfig`/`FilePlanConfig` pair:

```go
type ChangelogConfig struct {
    Provider string             `yaml:"provider"`
    Config   FileChangelogConfig `yaml:"config"`
}

type FileChangelogConfig struct {
    Directory string `yaml:"directory"`
}
```

A new `Changelog ChangelogConfig` field is added to the top-level `Config` struct, defaulted in the same construction path as `Spec`/`Plan` (provider `"file"`, directory `.spektacular/changelog`), and validated by a new `ChangelogConfig.Validate()` method following the same shape as `PlanConfig.Validate()`.

**`workflow.Config` (extended)** — the existing runtime config passed into every step callback gains one new field:

```go
type Config struct {
    Command string
    Kind    string
    DryRun  bool
    SpecDir string
    PlanDir string
    ChangelogDir string // new
}
```

No other field changes; `ChangelogDir` is populated from `Config.Changelog.Config.Directory` at the same call sites that populate `PlanDir` today.

**Changelog path helper and template variables** — the implement workflow's path-computation logic gains a new helper function alongside its existing `PlanFilePath`/`ContextFilePath`/`ResearchFilePath` functions:

```go
func ChangelogFilePath(dir, name string) string // returns "<dir>/<name>.md"
```

and its `PathVars` (already produces `plan_path`, `context_path`, `research_path`, `plan_dir`, `plan_name`) gains two more computed entries: `changelog_path`, built from `ChangelogFilePath`, and `spec_path`, built the same way the plan workflow's own strategy already builds it. The strategy type computing these gains `changelogDir` and `specDir` fields alongside its existing `planDir` field, sourced from `workflow.Config.ChangelogDir` and `workflow.Config.SpecDir` respectively.

**No new interfaces.** The changelog store reuses the existing `Store` interface (`Read`/`Write`/`Delete`/`List`/`Exists`/`Search`) and its sole implementation unchanged — a changelog store is just another `FileStore` instance rooted at a different directory, exactly like the plan and spec stores today. No new Go interface is introduced anywhere in this plan.

**No new FSM data contract.** The new `update_feature_changelog` step follows the same callback signature every implement step already uses (`func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error)`) and writes nothing into `workflow.Data` — consistent with every other implement step, none of which persist structured fields there today. The record's content is authored by the agent as prose and committed as a file, not assembled from typed data.

**Changelog record file** — not a Go type; a Markdown document at `.spektacular/changelog/<name>.md`, one per feature, structured as free-form prose with a small number of expected headings (what was built, why it matters, what it enables, notable deviations from the plan) so both a human and an agent reading it later can extract the same facts. Its exact heading structure is a template-authoring decision, not a data-structure decision, and is left to the implementation phase.

## Implementation Detail

No new architectural pattern is introduced by this plan — it is a same-shape extension of two patterns already established in the codebase. First, the "configured store kind" pattern (a config section naming a directory, a generic file-command factory bound to that directory, a `Store` instance rooted there) is cloned a third time, having already been proven twice for specs and plans. A developer reading the change will find the changelog store's config, command wiring, and path helpers structured identically to the plan store's — same field names, same function shapes, same command verbs — so no new mental model is required to understand it, only recognition of a pattern already present twice.

Second, the "FSM step that authors and commits a store artifact" pattern is reused as-is for the new `update_feature_changelog` step, following the shape already established by `test_plan`: a step callback that renders a template with no file I/O of its own, a template that instructs the agent to gather source material, author prose, stage it to a scratch path, and commit it through a store-file command. No new step-callback shape, no new template-authoring convention, and no new agent-facing instruction pattern (STOP-on-mismatch guards, the "refresh context.md before advancing" footer) is introduced — the new step's template is written against the same conventions every other implement-step template already follows.

The one piece of genuine judgment this plan introduces — not a code pattern, but a content decision — is what the changelog record's internal structure looks like: which headings it uses, how it balances "dense enough for an agent to extract facts" against "readable prose for a human," and how explicitly it calls out plan-vs-actual divergence. This is authored as part of the new step's template (the instructions the agent follows when writing a record), not as a rigid schema enforced by code, consistent with how the existing inline plan changelog and repo `CHANGELOG.md` also define their shape through template prose rather than a parser or schema. A developer or future template-editor extending the record's structure later edits one template file, the same way editing the repo-changelog format today means editing `08-update_repo_changelog.md`.

Existing patterns being followed, not replaced: the store abstraction (`Store` interface, `FileStore`), the generic store-file command factory, the `PathStrategy` interface for computing template variables, the FSM step-insertion mechanism (`Src`/`Dst`/`Callback` triples), and the stage-to-scratch-then-commit write convention. Nothing in this plan touches or generalizes any of these abstractions further than adding one more instance of each — there is no refactor, no interface change, and no package split.

## Dependencies

- **`internal/config`** — provides the `Config` struct, `NewDefault()`, `Validate()`, and the `SpecConfig`/`PlanConfig` pattern being cloned for `ChangelogConfig`. Needs changes: a new config section, default value, and validation method, added in the same shape as the existing spec/plan sections.

- **`internal/store`** — provides the `Store` interface and `FileStore` implementation. No changes needed; the changelog store is a new `FileStore` instance rooted at a new directory, using the existing interface unmodified.

- **`internal/workflow`** — provides the FSM engine, `Config`, `Data`, `StepConfig`, and `ResultWriter` that every workflow (spec, plan, implement) is built on. Needs a small change: one new field (`ChangelogDir`) added to `workflow.Config`. No change to the FSM engine itself, the `Data` store, or step-callback signature.

- **`internal/steps/implement`** — the package this plan's core work lands in: the new `update_feature_changelog` step, the updated step sequence, the extended `PathStrategy`, and the updated `finished` step summary. This is the primary package being changed, not merely depended on.

- **`internal/stepkit`** — provides `WriteStepResult` and the shared step-rendering pipeline every implement step already calls through `writeStep`. No changes needed; the new step reuses it exactly as every existing step does.

- **`cmd`** — provides command registration (`cmd/root.go`), the generic store-file command factory (`cmd/storefile.go`), and the existing `plan file`/`spec file` wiring examples (`cmd/plan_file.go`, `cmd/file.go`) the new `changelog file` commands are cloned from. Needs a new command file and one new line in `cmd/root.go` to register the `changelog` root command.

- **`templates`** — provides the mustache templates rendered at each implement step. Needs one new template for `update_feature_changelog` and a one-line addition to the existing `10-finished.md` template.

- **Prior plan `000015_implementation`** — established the implement workflow's step sequence and both existing changelog mechanisms (inline plan audit log, repo release note) this plan builds alongside. Already landed; no changes needed, referenced only as precedent.

- **Prior plan `000021_file-writing`** — established the stage-to-`.spektacular/tmp/`-then-`--from`-flag write convention this plan's new step reuses. Already landed; no changes needed.

- **Prior plan `000014_spektacular_store`** — established the `Store` interface this plan depends on unchanged. Already landed; no changes needed.

- **No external library dependencies.** No new third-party packages are introduced; this plan only extends existing internal packages and the standard library facilities they already use (`os`, `path/filepath`, `gopkg.in/yaml.v3` — all already imported by `internal/config`).

- **No planning-order dependency on spec `000032`.** Despite `000033` being sequenced to plan/implement before `000032` (so `000032` becomes the changelog's first dogfood subject), this plan has no technical dependency on `000032` — it only needs `000032`'s spec to exist as a completed feature to later exercise the new changelog step against, which is a validation step for the *next* piece of work, not a dependency of this plan's implementation.

## Testing Approach

Testing follows the two layers already established for the implement workflow: package-level structural/template tests (`internal/steps/implement/steps_test.go`) and the CLI-level harness in `cmd/implement_test.go` (`setupImplementCmd`, which drives real `rootCmd.Execute()` calls — `implement new`/`implement goto` — against a temp `.spektacular` directory seeded with a fixture plan, and asserts on the actual JSON result each step produces). This plan extends both layers rather than introducing a new testing approach; no test runs a real LLM agent in either layer — the CLI harness exercises the Go workflow/FSM/template-rendering machinery end-to-end, not agent behavior.

**FSM structure** (unit tests, most coverage here): assert the updated step list contains `update_feature_changelog` positioned between `test_plan` and `finished` (extending the existing `TestStepsOrderMatchesExpected`-style test), assert the FSM transition guards it correctly (`test_plan → update_feature_changelog → finished`, and that `finished` is no longer directly reachable from `test_plan`), and extend the existing full-walk test (`TestFSMWalkFromNewToFinished`-style) to walk through the new step. These are the load-bearing assertions for correctness: get the FSM wiring wrong and the workflow either skips the new step entirely or dead-ends before `finished`, both of which are easy to introduce accidentally when splicing a step into an existing linear chain.

**Template content** (unit tests, string/content assertions): assert the new step's template contains the expected directives — that it instructs reading the spec and the plan's inline changelog, that it names the correct scratch path and the correct `changelog file write` command, that it carries a STOP-on-mismatch guard (matching the existing `TestStopOnMismatchDirectivePresentInEveryNonTerminalTemplate` pattern, which should be extended to cover the new template automatically if it iterates all non-terminal templates, or a new explicit case added otherwise), and that the updated `10-finished.md` template's summary mentions the changelog record's location. These guarantee the instructions an agent receives are structurally complete, the same guarantee the existing template tests provide for every other step — they do not and cannot verify that an agent, given those instructions, writes a *good* record.

**Config and path plumbing** (unit tests): assert `ChangelogConfig.Validate()` rejects an unsupported provider and an empty directory (mirroring `PlanConfig.Validate()`'s existing test coverage), assert `NewDefault()` populates the changelog section with the expected default directory, and assert the new `changelog_path` template variable and `ChangelogFilePath` helper compute the expected path for a given name and directory (mirroring `TestPlanFilePaths_UseConfiguredDirectory`).

**CLI command wiring** (unit or thin integration tests, whichever matches how `plan file`/`spec file` are already tested): assert `changelog file write`/`read`/`list` round-trip against a temp directory, following the existing test pattern for the generic store-file command factory — this is largely free coverage since the factory itself is untouched and only a new binding is added.

**CLI-level implement harness (`cmd/implement_test.go`)** — this is where the new step's wiring gets its most realistic check, and is the layer the user specifically flagged as needing changelog coverage. Extend the existing `setupImplementCmd` harness with a test that walks a fixture plan through `implement new` and successive `implement goto` calls all the way to `update_feature_changelog` and `finished` (mirroring `TestImplementGoto_AdvancesThroughStep`'s shape), asserting: the step sequence actually reaches `update_feature_changelog` (not skipped, not dead-ending early), the rendered instruction at that step names the correct changelog path and the correct `changelog file write` command, and the `finished` step's rendered instruction mentions the changelog record's location. This test exercises the real FSM transition, the real `PathVars` computation, and the real template rendering together — it is the closest thing this codebase has to an integration test for a step, and confirms the plumbing decided in Architecture/Data Structures actually holds together end-to-end at the CLI boundary, not just in isolated package-level assertions.

**Deliberate gap:** no automated test exercises an agent actually authoring a changelog record's prose content, because the project has no such harness for any existing step (the `update_changelog`, `update_repo_changelog`, and `test_plan` steps are tested the same structural/CLI-harness way, not by running an agent). Whether a *produced record* is actually good — dense, self-contained, useful for downstream doc/blog writing — is not something a unit or CLI-harness test can assert; it depends on the quality of an LLM's output against a template's instructions, which is why both of the spec's success metrics are classified as manual below.

**Success metrics classification:**

- *"The changelog record for 000032 (once implemented) is sufficient on its own to write user-facing documentation or a blog post about it, without needing to re-read the spec, plan, or conversation."* — **Manual, captured in the implementation test plan.** This requires a real record to exist (produced by actually running the implement workflow against a real feature) and a human or independent agent judgment on whether that record is sufficient, which cannot be expressed as a behavioural assertion.

- *"Users rarely need to supplement a changelog record with outside context when writing docs or announcements from it."* — **Manual, captured in the implementation test plan.** This is an aggregate, subjective, usage-pattern metric ("rarely") that can only be assessed by observing real usage over time, not verified at implementation time by any single test.

## Milestones & Phases

### Milestone 1: A changelog store exists and can be read, written, and listed

**What changes**: Spektacular gains a new, configured location for changelog records (`.spektacular/changelog` by default) and a `changelog file read/write/list` command family that works exactly like the existing `plan file` and `spec file` commands. After this milestone, a changelog record can be manually authored and stored — read back by name, listed alongside others — but nothing yet produces one automatically. This is independently valuable and independently verifiable on its own: it proves the storage and retrieval mechanics work before any workflow depends on them.

#### - [x] Phase 1.1: Add changelog configuration

Introduce a new configuration section for the changelog store, following the exact shape already used for specs and plans: a provider name and a directory, defaulted automatically so a freshly initialised project gets a sensible location without any manual setup.

*Technical detail:* [context.md#phase-11](./context.md#phase-11-add-changelog-configuration)

**Acceptance criteria**:
- [x] A freshly generated `config.yaml` (via `spektacular init`) contains a `changelog` section with a default directory, alongside the existing `spec` and `plan` sections.
- [x] An invalid changelog configuration (unsupported provider, empty directory) is rejected with a clear error, the same way an invalid plan configuration is today.

#### - [x] Phase 1.2: Add `changelog file` commands

Add a `changelog file read/write/list` command family, giving users and the assistant the ability to read, write, and list changelog records directly — the same commands that already exist for spec and plan documents, applied to the new store.

*Technical detail:* [context.md#phase-12](./context.md#phase-12-add-changelog-file-commands)

**Acceptance criteria**:
- [x] Writing a changelog record by name and reading it back returns the same content.
- [x] Listing changelog records shows every record that has been written, by name.

### Milestone 2: Finishing an implementation automatically produces a changelog record

**What changes**: When the implement workflow completes for a feature, a changelog record for that feature now exists automatically — the assistant is instructed to gather what was actually built (from the feature's spec and its accumulated implementation history) and write a self-contained record, without the user writing anything by hand. The terminal summary the user sees at the end of an implement run now also tells them where the new record lives. After this milestone, the full loop the spec asks for is in place: implement a feature, and its changelog record exists afterward automatically, ready for a later documentation or announcement pass.

#### - [x] Phase 2.1: Add the `update_feature_changelog` step to the implement workflow

Insert a new step into the implement workflow's sequence, positioned right before the workflow reports completion. This step instructs the assistant to gather what was actually built for the feature — its spec and the record of what happened phase by phase, including anything that turned out differently than planned — and write it as a self-contained changelog record.

*Technical detail:* [context.md#phase-21](./context.md#phase-21-add-the-update_feature_changelog-step-to-the-implement-workflow)

**Acceptance criteria**:
- [x] Running the implement workflow to completion against a real plan produces a new changelog record named after that plan, without the user writing it by hand.
- [x] The record is understandable on its own — what was built, why it matters, and what it enables — without needing the original spec, plan, or conversation open alongside it.
- [x] Where the implementation diverged from the original plan, the record reflects what actually happened, not what was originally planned.

#### - [x] Phase 2.2: Report the changelog record's location when the workflow finishes

Update the workflow's final completion summary so it tells the user where the new changelog record lives, alongside the other artifacts it already reports (the repo-level release note, the completed phases).

*Technical detail:* [context.md#phase-22](./context.md#phase-22-report-the-changelog-records-location-when-the-workflow-finishes)

**Acceptance criteria**:
- [x] The summary a user sees at the end of a completed implement workflow mentions the changelog record's location.

No third milestone is needed — the spec's two remaining acceptance-criteria concerns (records accumulating rather than overwriting, and a specific feature's record being retrievable by name) fall out of Milestone 1's storage design (filename-as-identifier, one file per feature) with no additional work in Milestone 2.

## Open Questions

None. Every design decision this plan depends on was resolved during discovery, architecture, and phase drafting by reading the actual codebase — the FSM insertion point, the config clone pattern, the storage shape, and the spec-slug-equals-plan-slug convention (verified directly at `internal/steps/plan/strategy.go:22`, not assumed) are all confirmed facts, not open uncertainties.

The one loose end noted during phase drafting — the exact numeric prefix for the new template file (`10-update_feature_changelog.md` vs. renumbering `10-finished.md` to `11-`) — is a cosmetic file-naming choice with no behavioral effect, since templates are referenced by their explicit path in each step's `writeStep` call, not by directory sort order. It does not meet the bar for this section (it can be decided in five seconds during implementation, not discovered) and is left as an implementer's free choice rather than escalated here.

## Out of Scope

- **Building the downstream doc-writing or blog-writing process itself.** This plan produces the changelog record as an input artifact only; a separate future process that consumes records to generate docs or announcements is not part of this work (spec `000033` Non-Goals).

- **Retroactively generating changelog records for features implemented before this capability existed.** The changelog starts capturing from when this plan ships; prior implement-workflow runs (including everything already in `.spektacular/plans/`) will not get a backfilled record (spec `000033` Non-Goals).

- **A viewer, browsable UI, or search interface dedicated to the changelog.** Retrieval by name via `changelog file read <name>.md` and listing via `changelog file list` satisfy the spec's retrieval requirement; a human-browsable interface is not built here (spec `000033` Non-Goals). The changelog directory is incidentally covered by the existing `Store.Search` full-text search once it exists, but building or exposing a dedicated search UI for it is not part of this plan.

- **Automatic redaction, retention policies, or lifecycle management of changelog records.** Records are written and kept indefinitely with no automated pruning, expiry, or sensitive-data handling (spec `000033` Non-Goals).

- **A manifest or index file for the changelog store.** Rejected as an alternative during discovery (research.md § Alternatives 3–4) — retrieval relies entirely on filename-as-identifier, matching every other store in this codebase. If a future need for cross-record querying beyond full-text search arises, it is a separate future plan, not part of this one.

- **Folding changelog-record generation into the existing `finished` step, or making it config-gated/optional.** Both were presented as alternatives during the architecture step and explicitly not chosen (see this plan's Architecture & Design Decisions) — the record is produced by its own dedicated, unconditional FSM step instead.

- **Making the implement workflow's `spec_path` template variable configurable or resolvable by any means other than the existing plan-slug-equals-spec-slug convention.** This plan reuses the convention already proven by the plan workflow (`internal/steps/plan/strategy.go:22`) rather than introducing a lookup mechanism for cases where a plan's name might differ from its source spec's filename — no such case exists anywhere in this codebase today, and handling it is out of scope.

- **Extending or modifying the two existing changelog mechanisms** — the plan's inline `## Changelog` audit-log section and the repo-root `CHANGELOG.md` release notes. Both are explicitly preserved unchanged, per the spec's constraint that this is "a distinct, additional, retrospective artifact," not a replacement or repurposing of either (spec `000033` Constraints).

- **Sequencing or implementing spec `000032`** (spec-workflow-pair-programming-enhancements). It is the intended first real subject to exercise this changelog against once this plan ships, but planning or implementing it is separate work, tracked under its own spec.

## Changelog

### FINAL SUMMARY

This plan gave Spektacular a durable, self-contained changelog record for every feature the implement workflow completes. It added a new `changelog` store (config section, `changelog file read/write/list` commands, one flat Markdown file per feature under `.spektacular/changelog` by default) alongside the existing spec and plan stores, then wired a new `update_feature_changelog` FSM step into the implement workflow — positioned right before the terminal `finished` step — that instructs the assistant to gather the feature's spec and its accumulated inline changelog history and author a self-contained record grounded in what was actually built, including any deviations from the original plan. The workflow's final summary now also reports the new record's location. All four phases shipped exactly as planned, with no scope changes and no unresolved open questions.

**Total phases**: 4/4 completed

**Notable deviations from the plan**: None functionally. One cosmetic choice was made where the plan explicitly left it open: the `finished` step's template file was renamed from `10-finished.md` to `11-finished.md` (via `git mv`) to keep templates numbered sequentially once `10-update_feature_changelog.md` took that slot — the plan noted this numbering has no behavioral effect since templates are referenced by explicit path, not directory sort order.

### 2026-07-08 — Phase 1.1: Add changelog configuration

**What was done**: Added a `changelog` configuration section to Spektacular's config, following the exact shape already used for specs and plans (a provider name plus a directory setting). The section defaults automatically to `.spektacular/changelog`, so a freshly initialised project gets a sensible location with no manual setup, and an invalid configuration (unsupported provider, empty directory) is rejected with a clear error just like the existing plan configuration.

**Deviations**: None.

**Files changed**:
- `internal/config/config.go`
- `internal/config/config_test.go`

**Discoveries**: None — this phase was a direct, mechanical clone of the existing `PlanConfig`/`FilePlanConfig` pattern, exactly as the plan anticipated. gofmt auto-aligned the new struct's field tags after the initial edit; no functional impact.

### 2026-07-08 — Phase 1.2: Add `changelog file` commands

**What was done**: Added a `changelog file read/write/list` command family, giving users and the assistant the ability to read, write, and list changelog records directly, mirroring the existing `plan file` and `spec file` commands applied to the new changelog store.

**Deviations**: None.

**Files changed**:
- `cmd/changelog.go`
- `cmd/changelog_file.go`
- `cmd/root.go`
- `cmd/changelog_file_test.go`

**Discoveries**: None — this phase was a direct, mechanical clone of the existing `plan file` wiring pattern (`cmd/plan_file.go`, `cmd/plan.go`), reusing `newStoreFileCmd` unchanged, exactly as the plan anticipated.

### 2026-07-08 — Phase 2.1: Add the `update_feature_changelog` step to the implement workflow

**What was done**: Inserted a new `update_feature_changelog` FSM step into the implement workflow, positioned between `test_plan` and the terminal `finished` step. The step instructs the assistant to read the feature's spec and the plan's accumulated inline changelog section (which already captures actual-vs-planned divergence phase by phase), then author and commit a self-contained changelog record to the new changelog store. Added `ChangelogDir` to `workflow.Config`, `changelog_path`/`spec_path` template variables to the implement workflow's path strategy (the latter reusing the plan workflow's proven spec-slug-equals-plan-slug convention), and a new template modeled directly on the existing `test_plan` step's shape.

**Deviations**: Renamed the `finished` step's template file from `10-finished.md` to `11-finished.md` (via `git mv`, preserving history) to keep templates numbered sequentially now that `10-update_feature_changelog.md` occupies that slot. The plan explicitly flagged this numbering as a free implementer choice with no behavioral effect (templates are referenced by explicit path, not directory sort order), so this is not a functional deviation.

**Files changed**:
- `internal/workflow/workflow.go`
- `internal/steps/implement/strategy.go`
- `internal/steps/implement/steps.go`
- `internal/steps/implement/steps_test.go`
- `templates/steps/implement/10-update_feature_changelog.md`
- `templates/steps/implement/11-finished.md` (renamed from `10-finished.md`)
- `cmd/implement.go`
- `cmd/implement_test.go`

**Discoveries**: Three parallel Explore research agents (FSM plumbing, template model, test conventions) returned precise verbatim file maps before any code was written, confirming zero drift from the plan across every referenced file:line. This made the implementation itself mechanical — every edit matched what research predicted. An end-to-end smoke test (real spec + plan on disk, built binary, walked the full FSM from `new` through `finished`) confirmed the rendered `update_feature_changelog` instruction correctly references the spec read, plan read, scratch path, and changelog write commands.

### 2026-07-08 — Phase 2.2: Report the changelog record's location when the workflow finishes

**What was done**: Added two bullet points to the implement workflow's terminal `finished` step template (`templates/steps/implement/11-finished.md`), referencing the `{{changelog_path}}` template variable added in Phase 2.1, so the summary a user sees at the end of a completed implement run now tells them where the new changelog record lives — alongside the other artifacts it already reports (completed phases, the repo-level release note).

**Deviations**: None.

**Files changed**:
- `templates/steps/implement/11-finished.md`
- `internal/steps/implement/steps_test.go`

**Discoveries**: None — this phase was strictly sequential after Phase 2.1 and depended only on the already-existing `changelog_path` variable. Verified the resolved changelog path (`.../test.md`) is genuinely distinct from the resolved plan path (`.../test/plan.md`) in the test fixture, so the new assertion has no false-positive risk.
