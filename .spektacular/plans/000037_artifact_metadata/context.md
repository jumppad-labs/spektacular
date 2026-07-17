# Context: 000037_artifact_metadata

## Current State Analysis

Spektacular today produces four artifact classes as workflow output and none of them carry any lifecycle metadata:

- **Specifications** at `.spektacular/specs/<id>_<name>.md`. Written by `internal/steps/spec/steps.go:66-93` (`new()` step, direct `st.Write` of a rendered scaffold) and mutated by agents during the workflow via `spec file write` (`cmd/storefile.go:71-99`). No frontmatter today. See `templates/scaffold/spec.md` for the initial body shape.
- **Plan-directory sibling documents** at `.spektacular/plans/<id>_<name>/{plan.md,context.md,research.md,test-plan.md}`. `plan.md`, `context.md`, and `research.md` are written via the plan workflow's write phase (`internal/steps/plan/steps.go:245-275`), with the agent commit going through `plan file write` (`cmd/storefile.go:79-98`). `test-plan.md` is written by the implement workflow's `test_plan` step (`internal/steps/implement/steps.go`) via `plan file write .../test-plan.md`.
- **Changelog entries** at `.spektacular/changelog/<id>_<name>.md`. Written by the implement workflow's `update_feature_changelog` step (`internal/steps/implement/steps.go`) via `changelog file write`.

The Store interface at `internal/store/store.go:37-55` is deliberately byte-oriented — `Read`, `Write`, `List`, `Search`, `Exists`, `Delete`, `Root` — and this feature does not extend it. `store.DirEntry` (`internal/store/store.go:17-20`) carries only `{Name, IsDir}`; `store.Hit` (`internal/store/store.go:25-33`) carries text-search fields but no metadata. YAML is already a project dependency: `internal/config/config.go:176,293` uses `gopkg.in/yaml.v3` for config marshalling, and `internal/agent/agent_test.go:226-250` parses `---`-fenced frontmatter on skill docs — the shape this feature adopts.

The three top-level workflows share the `finished` terminal step convention. `internal/workflow/state.go:14-27` defines `State{Kind, CurrentStep, CompletedSteps, CreatedAt, UpdatedAt, Data}` with `InProgress()` (line 26) returning `CurrentStep != "" && CurrentStep != "finished"`. This is the signal a workflow's terminal-step callback uses (indirectly) to know that a lifecycle transition is happening — this feature does not add fields to `State`, it just calls into the metadata helper from each workflow's `finished` callback.

The shared CLI factory `newStoreFileCmd` at `cmd/storefile.go:71-165` is the choke point every agent-driven write flows through, and every per-kind `<kind> file` command tree is built from it — see `cmd/file.go:8-12` (spec), `cmd/plan_file.go:8-14` (plan), and `cmd/changelog_file.go` (changelog). It also owns the ID-prefix validation at lines 27-41 that this feature's new `set-status` subcommand reuses. The `list` subcommand at lines 138-161 currently returns `{"files": ["name1.md", …]}` — a bare-string array; Phase 2.1 changes this to a struct-per-entry.

There are no prior plans in `.spektacular/plans/` on the topic of artifact metadata, frontmatter, or artifact lifecycle. Two shipped specs are contextually relevant but not gating: `000034_spec-plan-implement-reconciliation` established the precedent of the implement workflow ticking checkboxes in a shipped spec (a *body* mutation, not a metadata one), and `000036_spec_plan_historical_artifacts` established the archaeology / owning-workflow distinction that this plan takes as foundational.

## Per-Phase Technical Notes

### Phase 1.1: Author the `internal/metadata` package

**File changes**:

- `internal/metadata/metadata.go` (new) — declares `Metadata`, `Status`, and the four `Status*` constants (mirroring `internal/agent/agent.go` style); implements a `dateOnly` YAML wrapper around `time.Time` so `created_date` / `closed_date` marshal as `YYYY-MM-DD` (using `MarshalYAML`/`UnmarshalYAML` on the wrapper). Uses `gopkg.in/yaml.v3` — same import as `internal/config/config.go:176`.
- `internal/metadata/frontmatter.go` (new) — `Split(raw []byte) (*Metadata, []byte, error)` scans for a leading `---\n…\n---\n` block (mirror of the shape parsed in `internal/agent/agent_test.go:226-250`); `Render(m Metadata, body []byte) ([]byte, error)` writes the fenced block then a single blank line then the body. `Split` returns `(nil, raw, nil)` when there is no leading block (bare artifact — not an error). `Split` returns a wrapped `fmt.Errorf("malformed frontmatter: %w", …)` when a `---` opener is present but the block is unterminated or the YAML fails to unmarshal.
- `internal/metadata/merge.go` (new) — `Merge(existing, newBody []byte, opts UpdateOptions) ([]byte, error)` calls `Split(existing)`, computes the resulting `Metadata` (preserving `CreatedDate`, applying `opts.Status`, stamping `ClosedDate` on transition), calls `Render`, returns the result. First-write (no existing metadata) stamps `CreatedDate=opts.Today`, `Status=StatusInProgress` unless overridden. `opts.Today` defaults to `time.Now().UTC()` truncated to day when zero-valued so production callers do not have to inject the clock (tests do).
- `internal/metadata/metadata_test.go` (new) — table-driven unit tests covering: split with/without frontmatter; render round-trip; merge fresh (asserts first-write stamps); merge preserving `created_date`; merge transitioning `in-progress` → `completed` (asserts single `closed_date` stamp); merge idempotent (transitioning `completed` → `completed` leaves file unchanged); enum rejection; malformed-frontmatter error propagation.

**Complexity**: Low

**Token estimate**: ~4k tokens

**Agent strategy**: Single agent, sequential execution. This is a small, self-contained package with no other-file coupling; parallelising costs more coordination than it saves.

### Phase 1.2: Make the plan scaffold check frontmatter-tolerant

**File changes**:

- `internal/steps/plan/steps.go:215-225` — the `planDocStillScaffold(name, docName, scaffoldPath string, st store.Store, cfg workflow.Config)` helper (or whatever its exact signature is at land time). Change: after `st.Read(...)`, call `metadata.Split(raw)` and compare `body` (the returned body slice) to the rendered scaffold rather than comparing `raw` to it. This makes the check answer "is the body still the scaffold?" which is the semantics it was always trying to capture.
- `internal/steps/plan/steps_test.go` (or the equivalent existing test file for plan step helpers) — add two cases: (a) a bare scaffold in the store returns "still scaffold"; (b) the same scaffold body with a well-formed frontmatter block prepended also returns "still scaffold"; (c) a materially different body returns "not scaffold" regardless of frontmatter presence.

**Complexity**: Low

**Token estimate**: ~2k tokens

**Agent strategy**: Single agent, sequential execution. One helper, one test file.

### Phase 1.3: Attach metadata on every `<kind> file write`

**File changes**:

- `cmd/storefile.go:71-99` — the `newStoreFileCmd` factory's `write` subcommand. Change: before calling `st.Write(...)`, read the existing store content (if any) via `st.Read(...)`, and call `metadata.Merge(existing, newBody, opts)` to compute the bytes to write. Add a `--status` flag on the `write` subcommand, parsed into an `UpdateOptions.Status`. `existing = nil` when `st.Read` returns `store.ErrNotFound`; that path is a first write and `Merge` stamps `CreatedDate=today`, `Status=StatusInProgress` (or the flag override).
- `cmd/storefile_test.go` (or wherever the current `newStoreFileCmd` write tests live) — extend to assert: (a) fresh write attaches frontmatter with `created_date=today, status=in-progress`; (b) subsequent write preserves `created_date`; (c) `--status completed` transitions and stamps `closed_date=today`; (d) writing a file that has no prior frontmatter attaches one on the next write. Each assertion runs against all three kinds via a table.

**Complexity**: Medium

**Token estimate**: ~7k tokens (the write handler has existing validation logic — `validateIDPrefix` at `cmd/storefile.go:27-41`, error paths — that the new code must interleave cleanly with).

**Agent strategy**: Single agent, sequential execution. The write handler is the choke point and needs its logic ordered carefully (validate ID → read existing → merge → write); parallelising the code and tests here risks the tests going stale.

### Phase 1.4: Add a metadata-only `<kind> file set-status` subcommand

**File changes**:

- `cmd/storefile.go:71-165` — the `newStoreFileCmd` factory. Add a new `set-status <path>` subcommand under it that takes `--status <s>` as a required flag, reuses `validateIDPrefix` on the path (same as `write`), reads the current file, calls `metadata.Merge(existing, existingBody, UpdateOptions{Status: &parsed})` — note: body reused from the existing file so no body change — and writes the result. Returns the standard success envelope with `{"path": path, "status": <new>, "closed_date": <if any>}`.
- `cmd/storefile_test.go` — new test cases (parameterised over the three kinds) covering: (a) `set-status completed` on an in-progress artifact stamps `closed_date` and leaves body byte-identical; (b) `set-status` with an out-of-enum value errors and does not modify the file; (c) `set-status` on a bare artifact attaches a fresh frontmatter block; (d) `set-status` twice with the same value is idempotent (second call leaves the file byte-identical to after the first).

**Complexity**: Low

**Token estimate**: ~5k tokens

**Agent strategy**: Single agent, sequential execution. Same choke point as Phase 1.3; layering on top of the same factory keeps changes local.

### Phase 1.5: Wire each workflow to attach on first write and close on finish

**File changes**:

- `internal/steps/spec/steps.go:66-93` — `new()` step callback. Change: replace the plain `st.Write(SpecFilePath(cfg.SpecDir, name), []byte(rendered))` with a call that goes through `metadata.Merge([]byte(""), []byte(rendered), UpdateOptions{Status: &StatusInProgress})` (or the equivalent helper that a small `metadata.WriteThrough(st, path, body, opts)` convenience wrapper would provide — decide during implementation).
- `internal/steps/spec/steps.go:150-168` — `finished()` step callback. Change: at the end of the callback's existing verification read-back, call `metadata.Merge(existing, existingBody, UpdateOptions{Status: &StatusCompleted})` and `st.Write` the result. Body is the existing body — no body change.
- `internal/steps/spec/steps_test.go` — new cases exercising the full workflow to `finished` against a temp store and asserting the frontmatter carries `status: completed` with a stamped `closed_date`; and a mid-flight case asserting `status: in-progress`.
- `internal/steps/plan/steps.go:277-299` — `finished()` step callback. Change: for each of the three plan-store siblings (`plan.md`, `context.md`, `research.md`), call `metadata.Merge` with `Status: &StatusCompleted` and re-write. Guard against a missing sibling by treating a `store.ErrNotFound` as an error the workflow surfaces to the agent (should not happen if the write steps ran).
- `internal/steps/plan/steps_test.go` — new case running the plan workflow to `finished` and asserting all three siblings carry `status: completed` with matching `closed_date`s; a mid-flight case asserting `in-progress` on the siblings after they were written but before `finished`.
- `internal/steps/implement/steps.go` — `finished()` step callback (or the last step of the FSM equivalent). Change: close two artifacts — the plan-store `test-plan.md` (path `<plan_name>/test-plan.md`) and the changelog entry (path `<id>_<slug>.md` under `changelog/`). Same `metadata.Merge` + `st.Write` pattern.
- `internal/steps/implement/steps_test.go` — new case running implement to `finished` and asserting both artifacts carry `status: completed` with a stamped `closed_date`; a mid-flight case asserting `in-progress` after the artifacts are written but before `finished`.

**Complexity**: Medium

**Token estimate**: ~14k tokens (six file changes across three workflow packages, three test suites, cross-file consistency).

**Agent strategy**: 2-3 parallel agents — one per workflow package (spec / plan / implement). Each workflow is an independent change: same pattern, different callback and test files. Merge sequentially at the end. This is a natural fan-out; the shared dependency (`internal/metadata`) is already stable after Phase 1.1.

### Phase 2.1: Add filter flags to per-kind `<kind> file list`

**File changes**:

- `cmd/storefile.go:138-165` — the `list` subcommand's cobra definition. Add five typed flags: `--status` (string, one of the four enum values or empty), `--created-after` / `--created-before` (`YYYY-MM-DD` string, parsed via `time.Parse("2006-01-02", …)`), `--closed-after` / `--closed-before` (same shape). Change the handler: after `st.List(path)`, load each candidate's raw bytes via `st.Read`, call `metadata.Split(raw)` (ignoring the `nil, raw, nil` case as a bare artifact), and match against each supplied filter. Return only matches. Extend the emitted per-file entry from a bare string to a struct: `{name, path, created_date?, status?, closed_date?}` where the metadata fields are present iff the artifact carries frontmatter. Bare artifacts appear when no filter is set; are excluded when any filter is set.
- `cmd/storefile_test.go` — new cases (per kind) covering: (a) `--status in-progress` returns only matching artifacts; (b) `--created-after X --created-before Y` returns only artifacts in the inclusive range; (c) `--closed-after X --closed-before Y` similarly; (d) combined filters intersect; (e) unfiltered `list` returns the same set as before plus per-entry metadata fields where present; (f) bare artifacts appear in unfiltered lists but not in filtered ones.
- **Backwards-compatibility check**: the output shape changes from `{"files": ["a.md", "b.md"]}` to `{"files": [{"name": "a.md", "path": "…"}, …]}`. Existing consumers of unfiltered `<kind> file list` output must be updated in the same phase. Grep `git grep -n '"files"' cmd/ internal/ tests/` and `git grep -n 'file list' templates/` to enumerate consumers before implementation begins.

**Complexity**: Medium

**Token estimate**: ~8k tokens

**Agent strategy**: Single agent, sequential execution. The output-shape change is the load-bearing risk; a single agent tracking every consumer in one pass is safer than parallel work that might miss one.

### Phase 2.2: Ship a cross-kind `spektacular artifacts list` aggregator

**File changes**:

- `cmd/artifacts.go` (new) — declares `artifactsCmd` (root of the new command tree) and `artifactsListCmd`. The `list` handler iterates: (a) `st.List("specs")` and emits each `.md` entry tagged `kind=spec`; (b) `st.List("plans")` — for each subdirectory, `st.List` inside and emit each of the four sibling doc types with `kind=plan.plan`/`plan.context`/`plan.research`/`plan.test-plan` (matched by filename); (c) `st.List("changelog")` and emit each `.md` entry tagged `kind=changelog`. For each candidate, load metadata via `metadata.Split` and apply the same five filter flags as Phase 2.1. Supports a `--kind` flag that accepts a comma-separated list narrowing which classes are scanned. Emits `{"error": false, "artifacts": [...]}`.
- `cmd/root.go:214-228` — register `artifactsCmd` on the root command in the same style as `spec`, `plan`, `changelog`, `implement`, `knowledge`.
- `cmd/artifacts_test.go` (new) — seed a temp store with a mix of artifacts across statuses and dates; exercise unfiltered list, per-filter list, combined-filter list, and `--kind` narrowing. Assert output-shape invariants (envelope shape, per-entry field presence) hold across empty and populated stores.

**Complexity**: Medium

**Token estimate**: ~9k tokens

**Agent strategy**: Single agent, sequential execution. The scan-and-filter logic is straightforward but needs to be threaded consistently across the four `kind` values; one agent keeps the mapping local.

## Testing Strategy

Testing sits at three tiers that map onto the phase structure above:

- **Phase 1.1 (unit)** — `internal/metadata` gets the strongest coverage in the plan because it is the one place a future edit could silently break the schema. Table-driven tests exhaustively cover the interesting transitions (fresh write, subsequent update, `in-progress` → each closed value, malformed input, enum violation) with a caller-injected clock for determinism.
- **Phases 1.2, 1.3, 1.4, 2.1 (integration)** — CLI-level tests against a temp store, exercised through the cobra command surface rather than by direct package call so the tests double as documentation of the CLI contract. Each phase's tests cover both the new behaviour it introduces and the specific regressions its change could cause (Phase 1.3 asserts `<kind> file write` behaviour across three kinds; Phase 2.1 additionally asserts unfiltered `list` shape is preserved for existing consumers).
- **Phase 1.5 (workflow)** — end-to-end workflow runs against a temp store, asserting that each of the three top-level workflows produces artifacts with the expected metadata at each of the two lifecycle points (in-progress mid-run, completed post-finish). Uses `2-3 parallel agents` per the phase strategy — one per workflow — because each workflow is independently testable.
- **Phase 2.2 (integration)** — cross-kind aggregator gets its own suite because the scan-and-filter behaviour is orthogonal to the per-kind list tests; sharing one test file would obscure whether a failure lives in the aggregator or in the per-kind path.

Two Success Metrics are marked as **Manual — captured in the implementation test plan** rather than as automated behavioural tests: "every new document produced by any workflow carries metadata, verified by spot-checking recently-written artifacts after the feature ships" (the spec's phrasing is deliberately post-ship; the workflow tests cover it internally but the spot-check itself is manual), and "the metadata and search capabilities provide sufficient signal to enable future downstream features without requiring changes to the metadata schema itself" (a design-longevity claim that cannot be asserted by any automated test). The implement workflow's `test_plan` step produces the concrete procedures for both.

## Project References

External and internal references used to build this plan. The full evidence trail lives in `research.md § Files examined` and `research.md § External references`; this section is a quick jump list.

**Codebase touchpoints** (in the order the phases hit them):

- `internal/metadata/` — new package (does not exist yet).
- `internal/steps/plan/steps.go:215-225` — `planDocStillScaffold` helper; Phase 1.2 changes its byte-compare.
- `cmd/storefile.go:71-165` — shared `newStoreFileCmd` factory; Phases 1.3, 1.4, 2.1 all attach here.
- `cmd/storefile.go:27-41` — `validateIDPrefix`, reused by the new `set-status` subcommand.
- `cmd/file.go:8-12`, `cmd/plan_file.go:8-14`, `cmd/changelog_file.go` — thin per-kind factories.
- `internal/steps/spec/steps.go:66-93` — spec `new()` step, direct `st.Write` — Phase 1.5 routes through `metadata.Merge`.
- `internal/steps/spec/steps.go:150-168` — spec `finished()` step — Phase 1.5 close.
- `internal/steps/plan/steps.go:277-299` — plan `finished()` step — Phase 1.5 close for three siblings.
- `internal/steps/implement/steps.go` — test_plan + update_feature_changelog + finished — Phase 1.5 close.
- `cmd/root.go:214-228` — root command registration; Phase 2.2 mounts `artifactsCmd` here.

**Read-only reference points** (understanding, not modifying):

- `internal/store/store.go:1-152` — Store interface + FileStore. Byte-oriented; not modified.
- `internal/workflow/state.go:14-27` — `State` struct + `InProgress()`. Not modified.
- `internal/output/writer.go:45-52,92-161` — JSON envelope. Not modified.
- `internal/config/config.go:176,293` — existing `gopkg.in/yaml.v3` usage; confirms the dependency.
- `internal/agent/agent_test.go:226-250` — existing YAML frontmatter parser precedent (skill docs); the shape reference.
- `templates/scaffold/spec.md` — spec scaffold body; unchanged.
- `templates/steps/plan/13-assemble.md`, `templates/steps/plan/15-write_plan.md` (+16, +17) — plan assembly and write templates; unchanged.
- `templates/steps/implement/09-test_plan.md`, `templates/steps/implement/10-update_feature_changelog.md` — implement write templates; unchanged.

**Prior specs consulted**:

- `.spektacular/specs/000037_artifact_metadata.md` — the source of truth for this plan.
- `.spektacular/specs/000034_spec-plan-implement-reconciliation.md` — established the implement-workflow-mutates-shipped-spec pattern (checkbox ticks).
- `.spektacular/specs/000036_spec_plan_historical_artifacts.md` — established the archaeology / owning-workflow distinction.
- `.spektacular/specs/000033_spec-workflow-output-changelog.md` — established the one-file-per-feature changelog shape.

**External references**:

- `gopkg.in/yaml.v3` — https://pkg.go.dev/gopkg.in/yaml.v3 (project dependency).
- YAML 1.2 timestamp schema — https://yaml.org/spec/1.2.2/#10214-timestamp (confirms `YYYY-MM-DD` is natively recognised).

**Prior plans consulted**: none — no prior plan on artifact metadata, frontmatter, or lifecycle status exists in the plans store.

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|-------------|----------------|
| Low | ~10k | Single agent, sequential |
| Medium | ~25k | 2-3 parallel agents |
| High | ~50k+ | Parallel analysis, sequential integration |

Applied to this plan's phases:

- **Low tier** — Phases 1.1 (~4k), 1.2 (~2k), 1.4 (~5k). Single-agent execution; the changes are self-contained and small.
- **Medium tier** — Phases 1.3 (~7k), 1.5 (~14k), 2.1 (~8k), 2.2 (~9k). Phase 1.5 is the one clear parallel-fan-out candidate (three workflow packages, same pattern each); the other three medium-tier phases are single-agent because their changes concentrate on a single choke point (`cmd/storefile.go` for 1.3 and 2.1; a new file plus one wire-in for 2.2).
- **High tier** — none in this plan. The single largest phase (1.5) tops out around 14k because the change per file is small; the phase count is what makes it feel large, not the per-site complexity.

## Migration Notes

No data migration is required. Every workflow-produced document that pre-dates this feature stays byte-identical on disk; every workflow-produced document written after this feature ships gains frontmatter automatically. The one code-level "migration" to be aware of is the `list` output-shape change in Phase 2.1 — see that phase's file changes for the mechanical work required to update in-tree consumers.

## Performance Considerations

Two paths need lightweight consideration but no active optimisation:

- **`<kind> file list` with a filter set** now reads every candidate artifact's bytes to load frontmatter, whereas the pre-existing unfiltered list only enumerated directory entries. For the current corpus (~40 specs, ~35 plan directories with 3–4 siblings each, a similar number of changelog entries — order of hundreds of files) this is imperceptible. If corpus size grows by orders of magnitude, a metadata cache (e.g. a sidecar index rebuilt on write) would be the natural next step, but building it now is premature.
- **`spektacular artifacts list`** does the same read work across the four artifact classes at once. Same reasoning: fine at current scale; if it becomes a problem the fix is a cross-store index, not a change to the per-hit read cost.

Both concerns are called out here rather than parked as open questions because the fix — if a fix ever becomes necessary — is additive and does not affect the schema or the CLI surface this plan ships. No latency budget is set; no benchmark is added.
