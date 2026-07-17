# Plan: 000037_artifact_metadata — working context

## Workflow

- Discovery step complete. Research written to `.spektacular/work/000037_artifact_metadata/research.md`.
- Spec path: `.spektacular/specs/000037_artifact_metadata.md`.
- Plan output path: `.spektacular/plans/000037_artifact_metadata/plan.md`.

## Spec at a glance

Attach machine-readable per-document metadata (created_date, status ∈ {in-progress, completed, superseded, archived}, closed_date on transition) to every workflow-produced document — specs, plan-directory siblings, changelog entries, test plans. YAML frontmatter is the natural fit for markdown artifacts. No backfill of pre-existing docs. Filtering artifacts by status + creation-date range + closed-date range must be combinable in one query, applicable per-kind and across all classes.

## Decisions (user interview + judgment calls)

1. **Close semantics — owner-workflow closes.** The workflow that writes a document is the one that stamps its `completed` status and `closed_date` at its terminal step. Spec workflow → spec.md. Plan workflow → plan.md/context.md/research.md. Implement workflow → test-plan.md + its changelog entry. Later mutations by a downstream workflow (e.g. implement ticking checkboxes in a shipped spec) do not re-open metadata.

2. **Update mechanism — two surfaces.**
   - **Implicit on `<kind> file write`.** Every `spec/plan/changelog file write` reads existing store frontmatter if present (preserving `created_date`), enforces the enum on `status`, and stamps `closed_date` iff transitioning to a closed status. First-write path (no existing file) stamps `created_date = today`, `status = in-progress`. Also applies to the direct `st.Write()` inside `internal/steps/spec/steps.go new()` and any other in-Go write site.
   - **Metadata-only subcommand** — `<kind> file set-status <path> --status <s>` — mutates only the frontmatter block (no body rewrite). Covers the workflow terminal-step "flip to completed" case and later manual `superseded` / `archived` moves.

3. **List / filter surface — per-kind flags + cross-kind aggregator.**
   - Extend `spec/plan/changelog file list` with typed cobra flags: `--status`, `--created-after`, `--created-before`, `--closed-after`, `--closed-before` (RFC3339 date-only, i.e. `YYYY-MM-DD`). Combinable in one query.
   - Add a new top-level `spektacular artifacts list` that returns hits from all four artifact classes tagged with a `kind` discriminant; accepts the same filter flags.

4. **Superseded / archived** — schema-only in v1; enum accepts all four values day one. Workflows only write `in-progress` and `completed`. `superseded` and `archived` are only reachable via the metadata-only subcommand from step 2.

## Judgment calls I made without asking

- **Schema fields core only** — `created_date` (YAML date), `status` (enum), `closed_date` (YAML date, absent while `in-progress`). No `kind` field embedded in frontmatter (the containing store already tells us the kind; embedding it would duplicate + risk drift).
- **YAML native date format `YYYY-MM-DD`.** Matches spec wording ("date"), renders cleanly through `yaml.v3`.
- **No backfill.** Existing bare docs stay bare. `file read` returns them unchanged. `file list --status X` naturally excludes bare docs (they have no metadata to match) — correct behaviour per spec AC.
- **`internal/metadata/` new package** as the shared writer helper. Called by every write site (both Go callbacks and the CLI write path).
- **Store interface stays byte-oriented.** Metadata lives above the substrate — parsed on `list`, injected on `write`. Preserves the spec's "substrate delegated per file writer" direction.

## Open assumptions worth watching (STOP if any wrong)

- Four artifact classes are exhaustive for "workflow output" (specs / plan-dir siblings [plan.md, context.md, research.md, test-plan.md] / changelog entries). No other doc types produced by any workflow today.
- `planDocStillScaffold()` at `internal/steps/plan/steps.go:215-225` — its byte-compare against the scaffold must be reworked to ignore leading frontmatter, otherwise adding frontmatter on write will break plan verification.
- `spektacular artifacts` is a fresh root verb — confirm no clash at architecture step.
- Malformed hand-edited frontmatter produces an actionable error rather than silently re-stamping `created_date`. Required for the user's "implicit on write" choice to keep created_date stable.

## Key files (architecture step will fold these into components)

- `internal/store/store.go` — byte-oriented Store interface + FileStore.
- `internal/workflow/state.go` — `State.InProgress()` signal for owner-workflow close.
- `internal/steps/spec/steps.go` — spec workflow steps (`new`, `finished`, verification).
- `internal/steps/plan/steps.go` — plan workflow steps + `planDocStillScaffold` byte-compare.
- `internal/steps/implement/steps.go` — implement steps (test_plan, update_feature_changelog, finished).
- `cmd/storefile.go` — shared `newStoreFileCmd` — the choke point for `<kind> file write/read/list/delete`; also where `set-status` and filter flags attach.
- `cmd/file.go`, `cmd/plan_file.go`, `cmd/changelog_file.go` — per-kind command factories.
- `cmd/root.go` — mount point for a new `artifactsCmd`.
- `internal/output/writer.go` — CLI envelope for filtered / cross-kind results.
- `internal/config/config.go` — existing `gopkg.in/yaml.v3` import.

## Sections drafted so far (working files under `.spektacular/work/000037_artifact_metadata/`)

- `architecture.md` — four-paragraph body covering substrate (YAML frontmatter + `internal/metadata`), two-surface update, two-surface list/filter, two load-bearing gotchas (`planDocStillScaffold` byte-compare; malformed-frontmatter error).
- `conventions.md` — project conventions category is currently README-only; two `AGENTS.md`-level house rules noted.
- `components.md` — eight components: `internal/metadata` (new), Store (unchanged, deliberately), `newStoreFileCmd` (grows two subcommands + filter flags), per-kind factories, spec/plan/implement step packages (small terminal-close changes + `planDocStillScaffold` fix), new `spektacular artifacts` root command, output writer (unchanged).
- `data_structures.md` — YAML frontmatter contract (three fields), `Metadata` Go type + Status enum, `internal/metadata` API (Split/Render/Merge + UpdateOptions), CLI subcommand signatures, filter-flag set, JSON output envelopes (`files` for per-kind, `artifacts` with `kind` discriminant for cross-kind).
- `implementation_detail.md` — the one new module boundary (`internal/metadata`), existing patterns everywhere else, the one exception (`spektacular artifacts` new root command), code-shape UX for readers, one existing helper altered (`planDocStillScaffold`).
- `dependencies.md` — `gopkg.in/yaml.v3` already available; internal deps consumed unchanged; new package + new `cmd/artifacts.go`; two prior specs (000034, 000036) as context, no blocking predecessors.
- `testing_approach.md` — three tiers (unit on `internal/metadata`, integration on CLI write/list + cross-kind, workflow tests on terminal-step close); five success metrics mapped, three behavioural + two manual (in test plan).
- `milestones.md` — two milestones (documents carry metadata; users can list-filter).
- `phases_plan.md` and `phases_context.md` — five phases in M1 (metadata pkg; scaffold-check fix; implicit write; set-status subcommand; wire three workflows), two phases in M2 (per-kind filter flags; cross-kind aggregator).

## All three store documents committed

- `plans/000037_artifact_metadata/plan.md`, `context.md`, and `research.md` are in the plan store.
- Working files under `.spektacular/work/000037_artifact_metadata/` removed after commit.
- Scratch files under `.spektacular/tmp/` removed after commit.

## Session state

- Ready to advance: `go run . plan goto --data '{"step":"finished"}'`.
