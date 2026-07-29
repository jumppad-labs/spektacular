---
created_date: "2026-07-28"
status: in-progress
---

# Plan: 000037_artifact_metadata

<!-- Metadata -->
<!-- Created: 2026-07-17T12:20:00Z -->
<!-- Commit: d509822fcc51b17171bb5b204d4d4184558337a0 -->
<!-- Branch: f-artifacts -->
<!-- Repository: git@github.com:jumppad-labs/spektacular.git -->

## Overview

This plan delivers per-document lifecycle metadata to every artifact Spektacular writes as workflow output — specifications, plan-directory siblings, changelog entries, and test plans — as a small YAML frontmatter block carrying creation date, status, and closed date. Workflows attach and update this metadata as they own the documents through their lifecycle, and the CLI grows filter flags plus a new cross-kind `spektacular artifacts list` command so users and agents can find artifacts by status and date range without opening the files themselves. This closes the discovery gap that today forces anyone triaging accumulated workflow output to inspect git history or read document bodies to infer age and lifecycle.

## Conventions

The project's `conventions` knowledge category currently contains only the category README stub (no concrete convention entries yet), so no project-declared conventions apply directly to this feature. Two standing repo-level house rules from `AGENTS.md` do bear on it, however:

- **Route all knowledge/memory writes through `spek-knowledge`, not the per-user memory store** — applies because the discovery step surfaced a possible durable learning (the `planDocStillScaffold` byte-compare gotcha); if that learning is worth persisting, it must be proposed to the user and written through `go run . knowledge write`, not saved to the machine-wide memory store.
- **Managed sections in `AGENTS.md` (Memory & Context, Spec-Worthy Discussion Recognition) are `spektacular init`-owned and hand edits will not survive re-init** — applies because this feature does not add a new AGENTS.md section; if that ever changes during implementation, the addition must be delivered through `templates/agents/` + `internal/agent/` installer + wire-in, not by hand-editing AGENTS.md.

## Architecture & Design Decisions

Every workflow document Spektacular writes gains a YAML frontmatter block at the top of the file — `---` fenced, containing `created_date`, `status`, and (once transitioned) `closed_date`. The block is machine-readable via `gopkg.in/yaml.v3` (already a project dependency for config and skill-doc parsing), invisible to a reader who only cares about the body, and produced only on new writes so pre-existing documents remain byte-identical to what they were before this feature ships. The four artifact classes covered are (1) spec files under `.spektacular/specs/`, (2) the four plan-directory siblings (`plan.md`, `context.md`, `research.md`, `test-plan.md`) under `.spektacular/plans/<id>_<name>/`, and (3) changelog entries under `.spektacular/changelog/`. YAML frontmatter is chosen because it is the spec's stated natural fit, has an established precedent in the tree (skill docs already use it — see `internal/agent/agent_test.go:226-250`), and needs no new dependencies. See `research.md#alternatives-considered-and-rejected` for the two rejected alternatives (per-writer implementations, agent-authored frontmatter in scratch files).

A new `internal/metadata` package is the single owner of frontmatter parse, render, merge, and enum validation, and it is called by every write site: the direct `st.Write()` inside `internal/steps/spec/steps.go:75-79`, the three plan write callbacks at `internal/steps/plan/steps.go:245-275`, the implement workflow's test-plan and changelog steps, and — most importantly — the shared `newStoreFileCmd` write handler at `cmd/storefile.go:79-98` that all agent-driven `<kind> file write --from <src>` traffic flows through. Centralising the schema in one Go package is a direct consequence of the "machine-readable" and "structured data" acceptance criteria: five current write sites with per-site formatting would drift under any future edit, and the write path is where the invariant (preserve `created_date`, enforce the enum on `status`, stamp `closed_date` iff transitioning to a closed status) has to hold. The store interface at `internal/store/store.go:37-55` stays deliberately byte-oriented — metadata sits above the substrate, not inside it, which matches the spec's "storage substrate delegated per file writer" direction and keeps the door open for non-file substrates without leaking markdown-specific concerns down into `Store.Read/Write`.

Update happens on two surfaces, per the user's interview answers. First, **implicit on write**: `<kind> file write` reads the store's current content for that path, extracts any existing frontmatter, preserves `created_date`, applies status/closed_date changes from CLI flags (or defaults on first write), and re-emits frontmatter + body. This is the primary path — the vast majority of writes flow through it without any workflow needing to think about metadata. Second, **a metadata-only subcommand** `<kind> file set-status <path> --status <s>` mutates the frontmatter block without touching the body. This is the tool the owning workflow's terminal step uses to flip `in-progress` → `completed` (and stamp `closed_date`) at the moment its FSM enters `finished` — a status-only transition where re-writing the body would be pointless. The same subcommand is what a future workflow (or a user by hand) would use to move a doc into `superseded` or `archived`; in v1 the enum accepts all four values but only `in-progress` and `completed` are workflow-writable, so the terminal-close case rides the same code path as the future `superseded`/`archived` moves.

Listing and filtering is delivered on two surfaces mirroring the update pair. Per-kind: `spec file list`, `plan file list`, `changelog file list` each grow typed cobra flags — `--status`, `--created-after`, `--created-before`, `--closed-after`, `--closed-before` — combinable in a single query. Cross-kind: a new top-level `spektacular artifacts list` (mounted at `cmd/root.go:214-228` alongside `spec`/`plan`/`changelog`) returns hits from all four artifact classes tagged with a `kind` discriminant field in the JSON envelope, accepting the same filter flags. Both surfaces sit above `store.Store.List` and post-filter by loading each candidate's frontmatter through `internal/metadata`; the search index in `internal/store/search.go` is not touched, and no new metadata-typed fields are added to `Hit` or `DirEntry`, preserving the substrate-agnostic layering. Existing pre-shipping documents surface in `list` output but are automatically excluded whenever any metadata filter is set (they have no frontmatter to match), which is the correct behaviour under the spec's "no backfill" requirement.

Two implementation-critical details fall out of the design and are load-bearing on it. First, `planDocStillScaffold` at `internal/steps/plan/steps.go:215-225` byte-compares stored plan siblings against the rendered scaffold; the frontmatter that this feature prepends on write would break that check unless the comparison is reworked to ignore leading frontmatter (or to compare against a scaffold that itself carries the current frontmatter). Second, malformed hand-edited frontmatter in a store file must produce an actionable error on the next write, not a silent re-stamp of `created_date` — otherwise the "implicit on write" mechanism the user picked would violate its own invariant (created_date preservation) whenever a human touched the file directly. Both are addressed in the component design that follows.

## Component Breakdown

- **`internal/metadata` (new package)** — Owns the artifact metadata schema, its YAML frontmatter serialization, and its enforcement. Exposes a small API: split a raw byte slice into `(frontmatter, body)`; parse frontmatter into a typed struct with enum validation; render a struct + body back to the fenced-frontmatter form; and a merge helper that takes an existing store-side blob plus caller-supplied field updates (status change, closed-date stamp) and preserves the invariants (`created_date` never re-stamped after first write; `closed_date` set only on transition into a closed state; enum enforced on `status`). This is the *single* place the schema lives; every other component that touches metadata calls into it. Reused by both the CLI write path and the in-Go write callbacks.

- **Store interface (`internal/store`)** — Unchanged, deliberately. The store stays byte-oriented (`Read`, `Write`, `List`, `Search`, `Exists`, `Delete`); it does not learn what metadata is. This preserves the spec's "storage substrate delegated per file writer" direction and lets a future non-file substrate slot in without renegotiating what a Store is.

- **`newStoreFileCmd` — shared per-kind CLI factory (`cmd/`)** — The single choke point every agent-driven write flows through. This component gains four responsibilities: (1) on `write`, wrap the caller-supplied body through `internal/metadata`'s merge helper before calling `st.Write`, so implicit-on-write metadata handling is uniform across `spec file write`, `plan file write`, and `changelog file write`; (2) on `list`, load each candidate's frontmatter via `internal/metadata` and apply the new filter flags before emitting the results envelope; (3) a new `set-status` subcommand that mutates only the frontmatter block and re-writes the file, using the same helper; (4) `read` stays byte-verbatim — it does not strip frontmatter, so metadata is trivially co-transported with the artifact through its normal access channel.

- **Per-kind command factories (`cmd/file.go`, `cmd/plan_file.go`, `cmd/changelog_file.go`)** — Continue to be thin factories that construct `newStoreFileCmd` per kind. They pass through the new subcommand and flag surface the shared factory now exposes; each also declares its `--status` flag's help text with the same enum, and each opts into the same filter-flag set on `list`.

- **`internal/steps/spec/steps.go` — spec workflow**  — The `new` step's direct `st.Write` for the scaffold is routed through `internal/metadata` so the first-ever spec write carries `created_date=today` and `status=in-progress`. The `finished` step gains a metadata-only status flip (via the shared helper — not via forking a CLI) that sets `status=completed` and stamps `closed_date=today`. No other spec steps change.

- **`internal/steps/plan/steps.go` — plan workflow** — Two touch points. First, `planDocStillScaffold` (the byte-compare that verifies the agent hasn't half-written a plan sibling) is reworked to strip a leading frontmatter block before comparing to the rendered scaffold; without this, the scaffold check would false-fail after this feature ships. Second, the `finished` step gains the same metadata-only close as the spec workflow, applied to all three plan-store siblings (`plan.md`, `context.md`, `research.md`). The `writePlan/writeContext/writeResearch` steps do not change directly — they call the shared write path in `newStoreFileCmd`, which already grew implicit metadata handling.

- **`internal/steps/implement/steps.go` — implement workflow** — Adds the metadata-only close at `finished` for the two artifacts this workflow owns: the test-plan file (`plan file write .../test-plan.md`) and the changelog entry (`changelog file write .../<id>_<slug>.md`). Neither the `test_plan` nor the `update_feature_changelog` step template needs a change — the frontmatter is injected by the shared CLI write path they already invoke, and the terminal close is done from the workflow's Go callback.

- **`spektacular artifacts` — new root command (`cmd/`)** — The cross-kind aggregator. One `list` subcommand that scans the four artifact classes' stores (specs directory, plans directory recursively for the four sibling doc types, changelog directory), loads each candidate's frontmatter through `internal/metadata`, applies the same filter-flag set as the per-kind `list`, and emits a JSON envelope tagged per-hit with a `kind` discriminant so callers can attribute a result. Mounted at root alongside `spec`/`plan`/`changelog`/`implement`. This is the only *new* root-level command this feature adds; everything else grows through the existing per-kind hierarchy.

- **`internal/output/writer.go`** — Unchanged. The existing success/error envelope already accommodates the new payload shapes (`{"error": false, "files": [...]}` for per-kind filtered lists; `{"error": false, "artifacts": [...]}` for the cross-kind aggregator). This is called out to make its non-participation explicit.

## Data Structures & Interfaces

The feature introduces one artifact-facing schema, one Go type mirroring it, and a small helper API in `internal/metadata`. The store, workflow-state, and CLI-output envelopes stay as they are.

**Artifact frontmatter (YAML)** — the on-disk contract. Fenced by `---` at the top of a markdown file, followed by a blank line, then the existing body:

```yaml
---
created_date: 2026-07-17
status: in-progress
closed_date: 2026-07-20   # present only when status is completed | superseded | archived
---
```

Three fields, no more. `created_date` is stamped once at the first-ever write and preserved thereafter. `status` is one of `in-progress`, `completed`, `superseded`, `archived`. `closed_date` is absent while `status: in-progress` and present (stamped at the transition, then preserved) for the other three values. Dates are YAML native `YYYY-MM-DD`. Documents that existed before this feature ships have no frontmatter block at all; downstream consumers accept a missing block and treat the document as unmetadata'd (no status match under any filter).

**`internal/metadata.Metadata` (Go)** — the in-memory mirror:

```go
type Status string

const (
    StatusInProgress Status = "in-progress"
    StatusCompleted  Status = "completed"
    StatusSuperseded Status = "superseded"
    StatusArchived   Status = "archived"
)

type Metadata struct {
    CreatedDate time.Time `yaml:"created_date"`
    Status      Status    `yaml:"status"`
    ClosedDate  time.Time `yaml:"closed_date,omitempty"`
}
```

Times are marshalled and unmarshalled as `YYYY-MM-DD` strings (never with a time-of-day component); the field is round-tripped through a small `yaml.Marshaler` / `yaml.Unmarshaler` so that day-precision is enforced at the type boundary rather than at every call site.

**`internal/metadata` package API** — the contract every write site programs against:

```go
// Split a raw artifact blob into its frontmatter block (or nil if absent) and its body.
func Split(raw []byte) (fm *Metadata, body []byte, err error)

// Render metadata + body back into a single blob with a fenced frontmatter block.
func Render(fm Metadata, body []byte) ([]byte, error)

// Merge caller-supplied field updates into the store's existing state for a path.
// - If existing has no metadata: stamps CreatedDate=today, Status=StatusInProgress (or override from opts).
// - If existing has metadata: preserves CreatedDate; applies Status/ClosedDate opts.
// - Enforces the enum on Status.
// - Stamps ClosedDate exactly when Status transitions in-progress -> closed value.
// - Returns an actionable error on malformed existing frontmatter (no silent re-stamp).
func Merge(existing []byte, newBody []byte, opts UpdateOptions) (out []byte, err error)

// UpdateOptions is what a caller passes to Merge; every field is optional and nil-checked.
type UpdateOptions struct {
    Status *Status  // nil = no change (or "in-progress" on first write)
    Today  time.Time // caller-injected clock; tests set this, prod passes time.Now().UTC() truncated to day
}
```

**`newStoreFileCmd` — CLI surface (`cmd/storefile.go`)**. Two new subcommands and one new flag set on the existing `list` subcommand. Signatures — one per surface, mirrored across all three kinds (`spec`/`plan`/`changelog`):

- `<kind> file write <path> --from <src> [--status <s>]` — same command as today, plus an optional `--status` that flows through `UpdateOptions.Status`. Body comes from `--from`; frontmatter is merged in by the CLI, not by the caller.
- `<kind> file set-status <path> --status <s>` — new. Mutates only the frontmatter, preserves the body, re-writes.
- `<kind> file list [path] [--status <s>] [--created-after <YYYY-MM-DD>] [--created-before <YYYY-MM-DD>] [--closed-after <YYYY-MM-DD>] [--closed-before <YYYY-MM-DD>]` — same command as today, plus a five-flag filter set. Combining flags intersects (AND semantics, not union), matching AC "combined filters intersect correctly."

**Cross-kind aggregator — CLI surface (`cmd/artifacts.go`, new)**:

- `spektacular artifacts list [--kind <k>] [--status <s>] [--created-after …] [--created-before …] [--closed-after …] [--closed-before …]` — scans specs, plan-directory siblings, and the changelog directory; loads each candidate's frontmatter; applies filters; returns one JSON envelope with a `kind`-tagged array.

**JSON output envelopes** — extensions of the existing shape produced by `internal/output.Write`. Per-kind filtered `list` returns:

```json
{
  "error": false,
  "files": [
    {
      "name": "000024_feature.md",
      "path": "specs/000024_feature.md",
      "created_date": "2026-01-15",
      "status": "completed",
      "closed_date": "2026-01-20"
    }
  ]
}
```

Cross-kind `artifacts list` returns:

```json
{
  "error": false,
  "artifacts": [
    {
      "kind": "spec",
      "name": "000024_feature.md",
      "path": "specs/000024_feature.md",
      "created_date": "2026-01-15",
      "status": "completed",
      "closed_date": "2026-01-20"
    },
    {
      "kind": "plan.plan",
      "name": "plan.md",
      "path": "plans/000024_feature/plan.md",
      "created_date": "2026-01-16",
      "status": "in-progress"
    }
  ]
}
```

`kind` values are one of: `spec`, `plan.plan`, `plan.context`, `plan.research`, `plan.test-plan`, `changelog`. The `plan.*` prefix distinguishes the four sibling doc types within a plan directory. Documents without frontmatter (pre-shipping artifacts) appear as `list` entries with only `name`/`path`/`kind` fields — the date and status fields are simply omitted — and are excluded from any query that sets a metadata filter.

**Store, workflow state, and existing knowledge/search hits** — no changes. `store.Store`'s interface (`Read/Write/List/Search/Exists/Delete/Root`) stays byte-oriented; `store.DirEntry` and `store.Hit` are not extended. `workflow.State` is unchanged — workflow lifecycle is separate from artifact lifecycle, and coupling them here would blur two distinct concerns. `internal/store/search.go`'s free-form keyword search remains as-is; the filter/list surface is deliberately a separate code path, not a modification of the search index, to keep the substrate substrate-agnostic.

## Implementation Detail

The plan introduces one new package (`internal/metadata`) and grows the shared CLI factory that already backs `spec file`, `plan file`, and `changelog file`. It follows the codebase's existing shape rather than introducing new architectural patterns — everything sits where a reader would already expect to find it, which keeps the surface area of the change small and locally auditable.

**A new module boundary — `internal/metadata` — separates the schema from every write site.** This is the one meaningful piece of new architecture the feature adds. The package exposes a narrow API (split, render, merge, and an `UpdateOptions` type — see § Data Structures & Interfaces) and owns the enum, the date-format contract, and the two invariants that the "implicit on write" mechanism depends on: `created_date` is never re-stamped after first write, and `closed_date` is stamped exactly once, on the transition from `in-progress` to any closed state. Every artifact write in the codebase — the direct `st.Write` inside the spec workflow's `new` step, the three plan-store writes triggered by the plan workflow's write phase, the two implement-workflow writes (test-plan and changelog entry), and the shared CLI `write` handler — flows through this package's `Merge`. Enforcing invariants in one Go type rather than in five step callbacks or in prompt text is a deliberate choice; it is the one place a future edit could break the schema, so it is the one place that grows the strongest tests.

**Existing patterns are followed everywhere else.** The store interface stays byte-oriented — no metadata-typed fields on `DirEntry` or `Hit` — mirroring the substrate-agnostic layering already in place; metadata sits *above* the store, not inside it. The CLI subcommand hierarchy stays as it is: `spec file` / `plan file` / `changelog file` continue to be per-kind factories that construct `newStoreFileCmd`; the new `set-status` subcommand and the new filter flags attach at the shared factory rather than being reimplemented per kind, matching how `write`/`read`/`list`/`delete` already work. The workflow FSM in `internal/workflow` and its per-step callbacks remain untouched; the only change to any workflow package is a small Go-side call into the metadata helper from each workflow's `finished` step callback, using the same `store.Store` handle those callbacks already receive as their third parameter.

**One deliberate exception to "no new commands at root": the cross-kind aggregator.** `spektacular artifacts list` is the only new root-level command. It is factored the same way as `knowledge search` — a single subcommand under a fresh top-level noun that reads from multiple stores. This is the cleanest way to satisfy the spec's "applicable across all covered artifact classes" requirement without smearing cross-kind logic through the per-kind commands, and it keeps the per-kind list commands from having to grow a `--kind` flag they would only tolerate for cross-cutting queries. Registration happens at `root.go`'s existing `AddCommand` block, in the same style as every other top-level command.

**Code-structure UX for a future reader.** A developer opening a stored artifact for the first time sees a small YAML block at the top of the file, blank line, then the familiar body — the shape they already know from skill docs. A developer opening the write path in `cmd/storefile.go` sees exactly one new call to `metadata.Merge` and one new flag; a developer opening a workflow's `finished` callback sees exactly one new call into `internal/metadata` to flip status. A developer looking for the schema goes to `internal/metadata` and reads the whole contract on one screen. Filter-flag help text under each per-kind `list` is templated from the same source of truth as the aggregator's flag set, so `spec file list --help` and `artifacts list --help` describe the same values in the same words.

**One code-shape change to an existing helper.** The plan workflow's byte-compare against a rendered scaffold (used to verify the agent hasn't half-written a plan sibling document to the store) is narrowed to compare bodies rather than raw blobs — it now strips a leading frontmatter block from the store side before the comparison. This keeps the scaffold's own text as the canonical "still a scaffold" signal; it is the only existing helper whose semantics are altered by this feature. Every other read path in the codebase — including `<kind> file read` — is untouched and continues to return the artifact byte-verbatim (frontmatter and body together), because the spec requires that metadata is co-transported with the artifact through its normal access channel.

**No new dependency, no new configuration knob, no new workflow FSM state.** The feature ships purely within the codebase's existing packaging (`internal/metadata` sits next to `internal/store` and `internal/workflow`), uses YAML through the `gopkg.in/yaml.v3` module that is already a project dependency, and adds no fields to `state.json`. This is intentional: the spec's Non-Goals rule out user-facing tooling for viewing or editing metadata, and adding a config knob (e.g. "which artifacts get metadata") would be a foot-gun the spec's "every workflow document" requirement explicitly forbids.

## Dependencies

**External libraries**

- **`gopkg.in/yaml.v3`** — YAML marshalling and unmarshalling for the frontmatter block. Already a project dependency (used in `internal/config` for config-file handling and referenced by the skill-doc frontmatter validation in tests). No version change, no `go.mod` update.

**Internal packages this work builds on**

- **`internal/store`** — provides the byte-oriented `Store` interface (`Read`, `Write`, `List`, `Search`, etc.) that every write site uses. Consumed unchanged; the metadata layer sits above it. No modification to the interface or the file-backed implementation.
- **`internal/workflow`** — provides `Workflow`, `State`, and the `finished` terminal-step convention. Consumed unchanged; the state model already carries the signal (`State.InProgress()` returning `false` at the `finished` step) that the metadata close hooks read from indirectly via the callback's `data`/`store` handles. No new fields on `State`.
- **`internal/output`** — provides the JSON envelope helper used by every CLI command. Consumed unchanged; the two new payload shapes (filtered `list` and cross-kind `artifacts list`) plug into the existing envelope.
- **`internal/config`** — provides the runtime configuration used to resolve store paths and command names. Consumed unchanged; the new subcommands read config the same way as their siblings.
- **`internal/steps/spec`, `internal/steps/plan`, `internal/steps/implement`** — the three workflow-step packages that own the write callbacks the new metadata helper is called from. Each one gets a small change to its `finished`-step callback (and the plan package additionally changes `planDocStillScaffold`), but no new interfaces are introduced between the workflow FSM and these callbacks.

**New internal package this work introduces**

- **`internal/metadata`** — new package that owns the frontmatter schema, YAML serialization, enum validation, and the merge helper used by every write site. Justified by the number of current write sites (five) that would otherwise re-implement frontmatter formatting and drift; centralising in one Go type is the load-bearing structural decision this feature makes. No other component depends on it landing separately — it is authored inside this same plan.

**CLI-layer additions**

- **`cmd/artifacts.go`** — new file registering the top-level `spektacular artifacts list` command. Depends on `internal/metadata` and `internal/store`. No dependency on anything landing outside this plan.
- **Existing `cmd/storefile.go` and per-kind factories (`cmd/file.go`, `cmd/plan_file.go`, `cmd/changelog_file.go`)** — modified in place to add `set-status`, filter flags on `list`, and the `--status` flag on `write`. No new dependencies; internal wiring only.

**Upstream specifications and prior plans**

- **Spec `000037_artifact_metadata`** — the source of truth for this plan. Its Requirements, Acceptance Criteria, and Success Metrics dictate every plan decision.
- **Spec `000034_spec-plan-implement-reconciliation`** (already shipped) — established the precedent that the implement workflow mutates a shipped spec's body (ticking checkboxes). This plan does **not** re-open that spec's metadata when the implement workflow ticks its boxes; the spec is closed by its *own* workflow finishing, and post-close body mutations do not reset lifecycle state. Called out here so the interaction between the two features is explicit and does not have to be rediscovered later.
- **Spec `000036_spec_plan_historical_artifacts`** (already shipped) — established the archaeology / owning-workflow distinction that this plan takes as foundational. No changes to that spec's shipped surface are required.
- **No prior plan** on artifact metadata, frontmatter, or lifecycle status exists in the plans store — this is a greenfield plan.

**No blocking predecessors**

- Nothing in this plan is gated on a spec or plan that has not yet shipped. Every referenced prior spec is in `completed` state (workflow-closed) at the time this plan is being authored.

## Testing Approach

Tests fall into three tiers matching where the code's invariants live: table-driven unit tests on `internal/metadata` (the schema owner), integration tests on the CLI write and list paths (where the invariants meet real files), and workflow tests on the three workflows' terminal-step behaviour (where the close-transition semantics are enforced). The bulk of the coverage weight goes on `internal/metadata` because it is the one place a future edit could silently break the schema, and on the write path in `cmd/storefile.go` because it is the choke point every agent-driven write flows through.

**Unit tests — `internal/metadata`.** Table-driven tests covering: split of a raw blob into (frontmatter, body) with and without a leading `---` block; render of a schema struct + body back to the fenced form; merge of caller updates into an existing store blob with `created_date` preservation asserted directly; merge on a first-ever write asserting the default stamp (`created_date=today`, `status=in-progress`); merge on a status transition asserting `closed_date` is stamped exactly once and preserved thereafter; merge on malformed hand-edited frontmatter asserting an actionable error (not a silent re-stamp of `created_date`); enum validation on `status` rejecting anything outside the four values. The clock is injected through `UpdateOptions.Today` so tests are deterministic.

**Integration tests — CLI write and list.** These sit alongside the existing `cmd/storefile_test.go` (or wherever the current `newStoreFileCmd` tests live) and exercise the metadata behaviour against a temp store:

- `<kind> file write` on a fresh path attaches frontmatter with `created_date=today, status=in-progress`.
- `<kind> file write` on a path that already has frontmatter preserves `created_date`.
- `<kind> file write --status completed` on an in-progress artifact transitions and stamps `closed_date=today`.
- `<kind> file write` on a bare (pre-shipping) artifact treats it as a first write and stamps metadata.
- `<kind> file read` on any artifact returns the full byte-verbatim content including frontmatter (metadata co-transported through the normal access channel).
- `<kind> file set-status <path> --status archived` mutates only the frontmatter block and leaves the body byte-identical.
- `<kind> file list --status <s>` returns only artifacts with matching status; bare artifacts are excluded from any filtered list.
- `<kind> file list --created-after X --created-before Y` returns artifacts whose `created_date` falls in the interval, inclusive at the boundaries.
- `<kind> file list --status s --created-after X` intersects filters (AND semantics), not unions them.

Each of the three per-kind factories (`spec`, `plan`, `changelog`) gets the same behaviour, exercised via a table-driven test rather than three copy-pasted suites.

**Integration tests — cross-kind aggregator.** A separate suite for `spektacular artifacts list`: emit two of each artifact class into a temp store, assert that unfiltered listing returns all of them with correct `kind` discriminants; assert that per-filter and combined filters intersect correctly across kinds; assert that `--kind plan.context` narrows results to just that class.

**Workflow tests — terminal-step close.** The three workflow packages already have step-callback tests; each grows a case that runs the workflow to `finished` against a temp store and asserts (via the metadata helper) that the artifact's frontmatter shows `status=completed` and a `closed_date` at the terminal step, and shows `status=in-progress` and no `closed_date` in an intermediate state. For the plan workflow, an additional case asserts that `planDocStillScaffold` continues to return the correct verdict after this feature ships — for both a bare (no-frontmatter) scaffold and a frontmatter-prepended scaffold.

**Regression coverage — no unintended surface changes.** Existing behavioural tests for `<kind> file read`, `<kind> file delete`, and `<kind> file list` without filters continue to pass byte-for-byte; the metadata layer must not change any un-flagged output shape. Existing tests for the store interface's byte-for-byte round-trip remain untouched — the store does not participate in metadata handling.

**No new end-to-end tests.** The existing `tests/` directory covers end-to-end workflow runs; those tests transitively exercise the metadata write path through the normal workflow steps and are expected to pass without new assertions.

**Deliberate gap: no property-based tests.** The metadata schema has three fields with narrow value spaces (two dates and one four-value enum); the table-driven unit tests exhaustively cover the interesting transitions. Property-based tests would not surface a class of bug the table tests miss here.

### Verifiability of each Success Metric from the spec

Every metric in the spec's Success Metrics section is carried through here — this is the plan's handoff to the implementer, and the implement workflow reads the plan, not the spec.

- **Agents reading historical documents can extract and act on the metadata.** → **Behavioural test.** Integration test on `<kind> file read`: read a stored artifact after a full workflow lifecycle and assert that the returned bytes contain a well-formed YAML frontmatter block parseable by the `internal/metadata` package (the same helper an agent would use to consume it).
- **Users can perform archive triage on accumulated artifacts using only the metadata, without inspecting git history or reading document bodies.** → **Behavioural test.** Integration test on the cross-kind aggregator: seed a store with a mix of statuses and ages, run `spektacular artifacts list --status archived` (and `--created-before X`), assert the returned envelope is sufficient to identify the artifacts (name, path, dates, status) without any body content.
- **When someone opens a historical document written after this feature ships, they can determine its age and lifecycle status directly from the document's metadata, without any secondary lookup.** → **Behavioural test.** Integration test on `<kind> file read`: the returned bytes contain both fields (`created_date`, `status`, `closed_date` when applicable) at the top of the file, no other lookup required.
- **Every new document produced by any Spektacular workflow (spec, plan, implement, changelog) carries metadata, verified by spot-checking recently-written artifacts after the feature ships.** → **Manual — captured in the implementation test plan.** The workflow-level assertion is covered automatically by the workflow tests above, but the "spot-check after ship" phrasing is deliberately a manual step against real workflow output; the implement workflow's test-plan artifact will carry the concrete spot-check procedure.
- **The metadata and search capabilities provide sufficient signal to enable future downstream features (e.g., an automatic archive action that mutates stale artifacts, or reports and dashboards built on the search results) without requiring changes to the metadata schema itself.** → **Manual — captured in the implementation test plan.** This is a design-longevity claim, not a behavioural assertion; there is no test that can prove no future feature will need a new field. The implement workflow's test-plan artifact will carry a manual review step against the shipped schema.

## Milestones & Phases

### Milestone 1: Every new workflow-produced document carries lifecycle metadata

**What changes.** After this milestone lands, any specification, plan-directory sibling (`plan.md`, `context.md`, `research.md`, `test-plan.md`), or changelog entry that Spektacular writes as part of a normal workflow gains a small YAML frontmatter block at the top of the file: the date the document was first written, its current lifecycle status (`in-progress` while the workflow that owns it is still open, `completed` once that workflow reaches its terminal step), and — once transitioned to a closed status — the date of that transition. Documents in the store before this milestone ships are untouched. A user who runs any workflow after this milestone and then opens the resulting file sees the block; a user who opens an older, pre-shipping document sees the same content it had before. The `<kind> file read` command continues to return whatever is stored, so metadata is co-transported with the artifact through the same channel a caller already uses. A new `<kind> file set-status <path> --status <s>` subcommand ships alongside so a user can flip a document to `superseded` or `archived` without rewriting the body, closing out the transitions the workflows themselves do not initiate.

**Validation point.** Run each of the three top-level workflows (spec, plan, implement) end-to-end against a temp store; open the resulting artifacts and confirm each carries frontmatter with a `created_date` matching the run date, `status: completed`, and a `closed_date` matching the run date. Additionally, run a workflow to an intermediate step (short of `finished`) and confirm the same artifacts show `status: in-progress` and no `closed_date`. Existing tests for spec/plan/changelog/implement continue to pass — this milestone is additive, not destructive.

#### - [x] Phase 1.1: Author the `internal/metadata` package

Stand up the single Go module that owns the artifact-metadata schema, its YAML on-disk shape, and its invariants. This is the foundation every subsequent phase calls into: it exposes helpers to split a stored artifact into `(frontmatter, body)`, render a struct + body back to the fenced form, and merge caller-supplied field updates into an existing store blob while preserving `created_date`, enforcing the four-value status enum, and stamping `closed_date` on transition. All logic is table-tested against representative inputs — fresh writes, subsequent updates, malformed hand-edited frontmatter, and each status transition.

*Technical detail:* [context.md#phase-11](./context.md#phase-11-author-the-internalmetadata-package)

**Acceptance criteria**:
- [x] A new `internal/metadata` package exists with `Split`, `Render`, `Merge`, `UpdateOptions`, `Metadata`, and `Status` symbols publicly available to the rest of the codebase.
- [x] `Merge` on an input that already has metadata never re-stamps `created_date`.
- [x] `Merge` on an input transitioning from `in-progress` to any closed status stamps `closed_date` exactly once.
- [x] `Merge` rejects a status value outside the four-value enum with an actionable error.
- [x] `Merge` on an input with malformed frontmatter returns an actionable error rather than silently discarding it.
- [x] Unit tests cover fresh-write, subsequent-update, malformed-input, and enum-violation cases, and pass under a caller-injected clock.

#### - [x] Phase 1.2: Make the plan scaffold check frontmatter-tolerant

Update the plan workflow's byte-comparison that verifies an agent has not half-written a plan-store sibling. Once the write path in Phase 1.3 begins attaching frontmatter to plan documents, the current byte-for-byte comparison against the rendered scaffold would false-fail on every plan run. This phase narrows the comparison to compare bodies only — a leading frontmatter block on the store side is stripped before the compare — so the check continues to answer the question it was asked without changing its outward behaviour. This must land before Phase 1.3 so the plan workflow does not regress between phases.

*Technical detail:* [context.md#phase-12](./context.md#phase-12-make-the-plan-scaffold-check-frontmatter-tolerant)

**Acceptance criteria**:
- [x] The plan workflow's scaffold-verification check treats a store artifact with a leading frontmatter block the same as one without, comparing bodies rather than raw blobs.
- [x] A unit test asserts the check returns the same verdict for a bare rendered scaffold and for the same rendered scaffold with a well-formed frontmatter block prepended.
- [x] A unit test asserts the check still fails on a body that materially differs from the rendered scaffold, regardless of whether frontmatter is present.

#### - [x] Phase 1.3: Attach metadata on every `<kind> file write`

Wire the shared CLI write handler that backs `spec file write`, `plan file write`, and `changelog file write` to route its content through the metadata package before touching the store. On a first-ever write the handler stamps `created_date=today`, `status=in-progress` (or a caller-supplied status), and no `closed_date`. On a subsequent write it preserves `created_date` and reflects any status change (stamping `closed_date` iff the transition is into a closed state). A new `--status` flag on the write command carries the caller's chosen status; without it, status is unchanged on updates and defaults to `in-progress` on first writes. Existing tests for the write command continue to pass byte-for-byte in the no-frontmatter/no-flag case only when they are exercised against pre-shipping artifacts; new tests cover the metadata attachment behaviour across the three kinds.

*Technical detail:* [context.md#phase-13](./context.md#phase-13-attach-metadata-on-every-kind-file-write)

**Acceptance criteria**:
- [x] Writing a new artifact through `spec file write`, `plan file write`, or `changelog file write` produces a stored file whose top begins with a YAML frontmatter block containing `created_date` set to the current date and `status: in-progress`.
- [x] Writing an existing artifact through the same commands preserves the `created_date` it already had.
- [x] Writing with `--status completed` on an artifact currently `in-progress` transitions the status and stamps `closed_date` set to the current date.
- [x] Writing an existing artifact that has no prior frontmatter is treated as a first write — the artifact gains a metadata block on its next write, and no error is raised.
- [x] Integration tests exercise the write behaviour for all three per-kind write commands.

#### - [x] Phase 1.4: Add a metadata-only `<kind> file set-status` subcommand

Ship a new subcommand — mirrored under `spec file`, `plan file`, and `changelog file` — that mutates only the frontmatter block of a stored artifact. This is the surface a workflow's terminal step uses (from Phase 1.5) to flip its own artifacts to `completed` without a body rewrite, and the surface a user or a future workflow uses to move a document into `superseded` or `archived` without touching content. The subcommand enforces the same enum validation as the write path and stamps `closed_date` at the transition.

*Technical detail:* [context.md#phase-14](./context.md#phase-14-add-a-metadata-only-kind-file-set-status-subcommand)

**Acceptance criteria**:
- [x] `spec file set-status <path> --status <s>`, `plan file set-status <path> --status <s>`, and `changelog file set-status <path> --status <s>` each mutate only the frontmatter block of the target artifact and leave the body byte-identical.
- [x] Running `set-status` with a value outside the four-value enum returns an actionable error and does not modify the file.
- [x] Running `set-status` on an artifact with no prior frontmatter attaches a block with `created_date=today` and the requested status, and (if the status is a closed value) `closed_date=today`.
- [x] Running `set-status` twice with the same value is idempotent — the second call leaves the file unchanged including its `closed_date`.
- [x] Integration tests cover each of the three per-kind `set-status` variants.

#### - [x] Phase 1.5: Wire each workflow to attach on first write and close on finish

Route the three workflows' write paths and terminal steps through the metadata helper so each workflow owns the lifecycle of the artifacts it produces. In the spec workflow, the scaffold write done by the workflow itself gains metadata (`created_date=today`, `status=in-progress`), and the terminal `finished` step flips the spec to `completed` and stamps `closed_date`. The plan and implement workflows write through the CLI already covered by Phase 1.3, so they need only close their own artifacts at the terminal step — plan closes its three sibling documents, implement closes the test-plan artifact and the changelog entry. After this phase, running any workflow to completion produces artifacts that carry `status: completed` at rest, and no other workflow's artifacts are touched.

*Technical detail:* [context.md#phase-15](./context.md#phase-15-wire-each-workflow-to-attach-on-first-write-and-close-on-finish)

**Acceptance criteria**:
- [x] Running the spec workflow to `finished` produces a spec file with `status: completed` and a `closed_date` matching the run date, whose `created_date` matches the run's start date.
- [x] Running the plan workflow to `finished` produces `plan.md`, `context.md`, and `research.md` in the plan directory with `status: completed` and matching `closed_date`s.
- [x] Running the implement workflow to `finished` produces a `test-plan.md` and a changelog entry with `status: completed` and matching `closed_date`s.
- [x] Running any workflow to an intermediate step short of `finished` produces artifacts with `status: in-progress` and no `closed_date`.
- [x] A workflow's terminal step does not touch any artifact produced by a different workflow.

### Milestone 2: Users and agents can list artifacts filtered by metadata

**What changes.** After this milestone lands, every per-kind `file list` command (`spec file list`, `plan file list`, `changelog file list`) accepts a common set of filter flags — filter by status, filter by creation-date range, filter by closed-date range — combinable in a single query. A new top-level `spektacular artifacts list` command applies the same filters across all four artifact classes at once and returns a single result set tagged with the kind of each match. Existing (unfiltered) invocations of `file list` return the same output shape they did before. Documents that pre-dated Milestone 1 (and therefore carry no metadata) still show up in an unfiltered `file list`, but are naturally excluded from any query that sets a metadata filter, matching the spec's "no backfill" requirement.

**Validation point.** Seed a temp store with a mix of artifacts across statuses and dates; run per-kind filtered queries and the cross-kind aggregator and confirm each returns exactly the set the filters describe, that intersecting filters (e.g. status *and* a date range) return the intersection rather than the union, and that pre-metadata bare documents are present in unfiltered lists but absent from filtered ones. Additionally, invoke each `file list` command with no flags and confirm the pre-existing output shape is unchanged.

#### - [x] Phase 2.1: Add filter flags to per-kind `<kind> file list`

Extend the shared `list` subcommand — used by `spec file list`, `plan file list`, and `changelog file list` — with five typed cobra flags: `--status`, `--created-after`, `--created-before`, `--closed-after`, `--closed-before`. When any filter flag is set, the command reads each candidate artifact's frontmatter through the metadata package and returns only artifacts whose metadata matches all supplied filters (intersecting, not unioning). Artifacts with no frontmatter (pre-shipping documents) are still returned when no filter is set, but naturally drop out of any filtered query. The output shape gains the metadata fields per entry so the caller does not have to re-read each file to see status.

*Technical detail:* [context.md#phase-21](./context.md#phase-21-add-filter-flags-to-per-kind-kind-file-list)

**Acceptance criteria**:
- [x] `spec file list --status in-progress`, `plan file list --status in-progress`, and `changelog file list --status in-progress` each return only artifacts whose stored status matches.
- [x] Filtering by a creation-date range with `--created-after` and `--created-before` returns only artifacts whose `created_date` falls inclusively within the range.
- [x] Filtering by a closed-date range with `--closed-after` and `--closed-before` returns only artifacts whose `closed_date` falls inclusively within the range.
- [x] Combining filter flags in one query returns the intersection — an artifact must satisfy every specified filter to appear.
- [x] An unfiltered `<kind> file list` invocation returns the same set of artifacts as before this phase, with metadata fields added per entry where present.
- [x] Bare (pre-shipping) artifacts appear in unfiltered lists but do not appear in any list that sets a metadata filter.

#### - [x] Phase 2.2: Ship a cross-kind `spektacular artifacts list` aggregator

Register a new top-level command that lists artifacts across all four covered classes at once. It scans the specs directory, every plan directory (surfacing each of the four sibling documents as its own entry), and the changelog directory; loads each candidate's frontmatter; applies the same five-flag filter set as Phase 2.1; and returns one JSON envelope tagged per-hit with a `kind` discriminant so the caller can attribute results. An additional `--kind` flag narrows the scan to one or more classes at the caller's discretion.

*Technical detail:* [context.md#phase-22](./context.md#phase-22-ship-a-cross-kind-spektacular-artifacts-list-aggregator)

**Acceptance criteria**:
- [x] `spektacular artifacts list` with no filters returns one entry per artifact across specs, plan-directory siblings, and changelog entries, each tagged with its `kind`.
- [x] Each entry in the result carries `kind`, `name`, `path`, and (where present) `created_date`, `status`, and `closed_date`.
- [x] Passing any of the five filter flags applies the same intersection semantics as Phase 2.1, across all classes in a single query.
- [x] Passing `--kind plan.context` (or any other class discriminant) narrows the result to only that class.
- [x] The command produces the same output shape whether invoked from an empty store or a fully populated one — the envelope's structure does not depend on how many hits it contains.

## Open Questions

Two genuinely implementation-time uncertainties remain. The rest of the plan resolves every decision.

- **Which existing consumers of `<kind> file list` output depend on the current bare-string shape.** Phase 2.1 changes the entry shape from `"a.md"` to `{"name": "a.md", "path": …, "created_date": …, "status": …}`, and updating those consumers is inside the phase. The implementer must run `git grep -n '"files"'` across `cmd/`, `internal/`, `tests/`, and `templates/`, plus any workflow-step JSON parser that reads the list envelope, before writing the shape change — the plan enumerates the surface but the *exact* set of consumer sites is only knowable at implementation time against the tree at that moment. If any consumer would meaningfully break beyond a mechanical field-access update, STOP and ask the user whether to add a `--legacy-list` compatibility flag rather than change the default shape.

- **Whether any other in-Go write site of a store artifact exists outside the three already identified.** The plan covers the spec workflow's `new` step (a direct `st.Write`) and every workflow's terminal-step close (via `metadata.Merge` + `st.Write`), and it covers every agent-driven write via the shared CLI `write` handler. Discovery searched for `st.Write` calls under `internal/steps/`, but a future step-callback added between plan authoring and implementation could add a fourth in-Go writer that would silently produce metadata-less artifacts. The implementer must run `git grep -n 'st\.Write' internal/steps/ cmd/` at the start of Phase 1.5 and confirm the list matches the plan's expectations; if a new site exists, STOP and ask the user whether to route it through `metadata.Merge` in the same phase or park it as follow-up work.

## Out of Scope

Carried over from the spec's Non-Goals:

- **Automatic archival — no workflow, cron job, or command in this plan moves an artifact to `archived` based on age or any other rule.** The plan ships the schema value and the `<kind> file set-status <path> --status archived` surface that a future feature would call, but the decision to archive is a user (or future workflow) action, not something this plan performs. Tracked as: not yet specified.
- **User-facing tooling or UI for viewing or editing metadata.** This plan exposes metadata via the same CLI channels that already exist (`<kind> file read`, `<kind> file list`, `spektacular artifacts list`); it does not add a dashboard, TUI, or web view. Tracked as: not yet specified.
- **Metadata on Spektacular's internal state and configuration files.** `.spektacular/state.json`, `.spektacular/config.yaml`, workflow scratch files under `.spektacular/tmp/`, and the working files under `.spektacular/work/` are all out of scope. Only artifact documents produced *as output* of a workflow are covered.
- **Metadata on transient scratch files.** The staged templates under `.spektacular/tmp/` used during plan assembly do not get frontmatter — they are deleted at workflow completion and never reach the store.

Chosen-design exclusions surfaced during the architecture step:

- **Backfilling metadata onto pre-shipping artifacts.** Every existing spec, plan-directory sibling, and changelog entry stays byte-identical to what it was before this feature ships. Downstream consumers accept that bare artifacts have no metadata and are excluded from filtered lists. A future backfill sweep, if ever wanted, would be its own spec. Tracked as: not yet specified.
- **Auto-detecting supersede relationships.** The plan supports the `superseded` status value in the schema and via `set-status`, but it does not attempt to *detect* that a new spec supersedes an older one (e.g. by title similarity, cross-reference, or workflow-time flag). Every supersede transition in v1 is an explicit `set-status` call. Tracked as: not yet specified.
- **Extending `store.Search()` or `store.Hit` with metadata-typed fields.** The filter/list path is a deliberately separate code layer above the byte-oriented store; the search index is not touched. Any future full-text-plus-metadata query surface (e.g. "search body content but also filter by status") would be its own plan. Tracked as: not yet specified.
- **Adding new fields to `state.json` for artifact lifecycle.** Workflow state and artifact lifecycle are two distinct concepts and are deliberately kept in separate substrates: workflow FSM in `state.json`, artifact lifecycle in the artifact's own frontmatter.
- **A configuration knob for which artifacts get metadata.** Every workflow-produced document gets metadata, unconditionally. A per-artifact opt-out would violate the spec's "every workflow document" requirement and would be a foot-gun.

Deliberately deferred from a strict reading of the spec:

- **The manual `supersede` and `archive` CLI shortcuts.** The user's answer to the plan-phase interview steered v1 toward "schema-only" for these two states, reachable via the metadata-only `set-status` subcommand rather than dedicated verbs. If a future feature needs `spektacular <kind> supersede <old> --by <new>` (which also mutates the newer artifact to reference the older one), it is a follow-up spec.

## Changelog

### 2026-07-28 — Phase 1.1: Author the `internal/metadata` package

**What was done**: Introduced a new Go package `internal/metadata` that owns the artifact-frontmatter schema, its YAML on-disk shape, and its invariants. Exposes `Split`, `Render`, `Merge`, `Metadata`, `Status`, and `UpdateOptions`; internal helpers `validateStatus` and `isClosed` enforce the four-value enum and the closed-status classification. Every write site downstream will route through `Merge` so the schema stays consistent across sites.

**Deviations**: None. The plan's Data Structures section shows `Metadata` with `time.Time` fields; the implementation keeps that public shape and enforces `YYYY-MM-DD` day precision by attaching `MarshalYAML`/`UnmarshalYAML` to `Metadata` with an unexported `yamlShape` intermediate (rather than a public `Date` wrapper type). Same on-disk contract, cleaner public API.

**Files changed**:
- `internal/metadata/metadata.go` (new) — `Status`, four consts, `Metadata`, `MarshalYAML`/`UnmarshalYAML`, `validateStatus`, `isClosed`.
- `internal/metadata/frontmatter.go` (new) — `Split`, `Render`.
- `internal/metadata/merge.go` (new) — `Merge`, `UpdateOptions`.
- `internal/metadata/metadata_test.go` (new) — table-driven tests for all three functions plus the enum boundary and the injected-clock assertion.

**Discoveries**: `yaml.v3` quotes date-shaped strings when they are typed as `string` in a marshaled struct (e.g. `created_date: "2026-07-01"`). This is a schema-safe choice — the parser round-trips them as strings rather than reinterpreting them as YAML `!!timestamp` values, which is exactly what we want for day precision. Downstream tests should assert on the *value* rather than pin the quoting style. Also: `Metadata.UnmarshalYAML` enforces the enum at parse time, so `Split` on frontmatter with a bogus status returns a wrapped `malformed frontmatter:` error rather than a valid `Metadata` with a bad status — dual enforcement (Split + Merge) is intentional so agents cannot slip a bad status through either surface.

### 2026-07-28 — Phase 1.2: Make the plan scaffold check frontmatter-tolerant

**What was done**: Narrowed `planDocStillScaffold` in `internal/steps/plan/steps.go` so it strips a leading YAML frontmatter block from the stored bytes before comparing to the rendered scaffold. When `metadata.Split` returns a non-nil `*Metadata`, the returned body slice is compared instead of the raw store bytes; otherwise the raw bytes are used unchanged. This preserves the pre-existing verdict semantics on bare artifacts and prevents a false-fail once Phase 1.3 begins attaching frontmatter on write.

**Deviations**: None. Change is exactly what the phase context called for — one helper, one comparison, one new import.

**Files changed**:
- `internal/steps/plan/steps.go` — imported `internal/metadata`; `planDocStillScaffold` now strips frontmatter via `metadata.Split` before comparison.
- `internal/steps/plan/planstill_test.go` (new) — table-driven coverage across all three `planDocs` entries: bare scaffold, scaffold-with-frontmatter, filled body without frontmatter, filled body with frontmatter, missing file.

**Discoveries**: `metadata.Split` swallowing its error and falling through to the raw-bytes branch is a deliberate design choice — a malformed frontmatter block that cannot parse should not silently make the check say "not scaffold"; the check answers the narrower question "did the agent commit real content?" so treating unparseable bytes as raw content and letting the downstream write path surface the parse error is the correct layering. If a future phase changes this contract, it should surface the malformed-frontmatter error to the caller.

### 2026-07-28 — Phase 1.3: Attach metadata on every `<kind> file write`

**What was done**: Routed the shared `newStoreFileCmd` write handler through `metadata.Merge` before writing to the store, so `spec file write`, `plan file write`, and `changelog file write` all attach (or update) frontmatter automatically. Added a `--status` flag on the write command, plumbed through a new `metadataOptsForStatus` helper that enum-validates the value and returns an `UpdateOptions` payload. First-writes stamp `created_date=today, status=in-progress`; subsequent writes preserve `created_date`; `--status completed` on an in-progress artifact stamps `closed_date=today`. Bare pre-shipping artifacts are treated as first-writes on their next `<kind> file write`.

**Deviations**: None from plan scope. Adjusted three pre-existing byte-identity assertions in `cmd/file_test.go`, `cmd/plan_file_test.go`, and `cmd/changelog_file_test.go` to strip frontmatter via `metadata.Split` before comparing bodies — the intent of those tests (body preservation) still holds, only the shape assertion moved above the metadata layer.

**Files changed**:
- `cmd/storefile.go` — imported `internal/metadata`; added `metadataOptsForStatus` helper; wired write handler to read existing store content, call `metadata.Merge`, and write the merged result; added `--status` flag on `write`.
- `cmd/file_test.go` — updated `TestSpecFileWrite_ResolvesConfiguredDirectory` and `TestSpecFileWrite_PreservesProblematicCharacters` to strip frontmatter before body assertion.
- `cmd/plan_file_test.go` — same treatment for `TestPlanFileWrite_ResolvesConfiguredDirectory`, `TestPlanFileWrite_PreservesProblematicCharacters`, `TestPlanFileWrite_AcceptsCounterIDPrefix`.
- `cmd/changelog_file_test.go` — updated `TestChangelogFileWriteRead_RoundTrips` to strip frontmatter from the read output.
- `cmd/storefile_metadata_test.go` (new) — table-driven Phase 1.3 acceptance tests across all three kinds: fresh-write stamps, existing-artifact preservation, status-flag transition, bare-artifact upgrade, fresh-write-with-closed-status, invalid-status rejection.

**Discoveries**: Cobra flag state on `--status` is *package-global* through the closure-captured `statusFlag` string inside `newStoreFileCmd`. A test that sets `--status` and does not reset it leaks the value into the next `rootCmd.Execute()` call in the same process — hurt one test until the sub-agent added a `resetWriteStatusFlag` helper with `t.Cleanup` to walk `rootCmd.Find([]string{kind, "file", "write"})` and clear the flag. Production is unaffected (each real invocation is a fresh process), but every new test that touches `--status` must reset the flag afterwards. Also: `metadata.Merge` bubbles up as an `output.NewError("metadata_merge_failed", …)` so agents see the actionable error envelope rather than a bare Go error string.

### 2026-07-29 — Phase 1.4: Add a metadata-only `<kind> file set-status` subcommand

**What was done**: Added a `set-status <path>` subcommand to `newStoreFileCmd` so `spec file`, `plan file`, and `changelog file` all expose the same metadata-only transition surface. The handler reuses `validateIDPrefix` (gated by the same `requireID` flag as write), reads the current file, splits out its body via `metadata.Split`, then calls `metadata.Merge(existing, body, opts)` with a required `--status` flag value. On success it emits `{"error": false, "path": "<path>", "status": "<new>", "closed_date": "<if any>"}`. Missing files return `not_found`; a missing `--status` returns `missing_status` (belt-and-braces alongside cobra's `MarkFlagRequired`); an invalid enum value returns `invalid_status`.

**Deviations**: None to the plan. During this phase's own workflow run I discovered a genuine Phase 1.3 bug — repeated `plan file write --from <src>` on a source that already carries a frontmatter block accumulates blocks (the write handler was merging over the raw source bytes, so each round added a fresh block on top of the source's existing one). Fixed by adding `stripLeadingFrontmatterBlocks` in `cmd/storefile.go` which loops `metadata.Split` until no leading block remains, then passes the result as `newBody` to `metadata.Merge`. Idempotency of `<kind> file write` under repeated invocations is now enforced by `TestStoreFileWrite_IdempotentUnderRepeatedWrites`. Also cleaned up the three accumulated frontmatter blocks that this bug had left at the top of this very plan.md.

**Files changed**:
- `cmd/storefile.go` — added `set-status` subcommand and a `stripLeadingFrontmatterBlocks` helper; write handler now strips source-side frontmatter before Merge to guarantee idempotency.
- `cmd/storefile_metadata_test.go` — extended with six new set-status acceptance tests (all three kinds via table-driven scaffold) plus `TestStoreFileWrite_IdempotentUnderRepeatedWrites` regression guard; added a `resetSetStatusFlag` helper and a `seedArtifactWithMetadata` helper.

**Discoveries**: The set-status flag has the same cobra package-global state hazard as the write flag; `resetSetStatusFlag` mirrors `resetWriteStatusFlag` and additionally clears the flag's `Changed` bit so `MarkFlagRequired`'s "was set" gate re-arms between tests. The Phase 1.3 idempotency bug surfaced immediately when the implement workflow's own `plan file write` calls started stacking frontmatter blocks on plan.md — the workflow effectively dogfooded its own change into a regression signal in under a minute. Worth remembering for any future feature that adds a file-mutating CLI whose output feeds back into its own input.

### 2026-07-29 — Phase 1.5: Wire each workflow to attach on first write and close on finish

**What was done**: Each of the three top-level workflows now owns the lifecycle of the artifacts it produces. Spec workflow's `new()` routes its scaffold write through `metadata.Merge` so the very first spec write carries `created_date=today, status=in-progress`; its `finished()` calls `metadata.Close(...)` on the spec file when the scaffold check confirms it was written. Plan workflow's `finished()` iterates `planDocs` and closes plan.md / context.md / research.md — gated by the same scaffold check that already surfaced `plan_incomplete=true` warnings. Implement workflow's `finished()` closes both artifacts it owns: `<PlanDir>/<name>/test-plan.md` and `<ChangelogDir>/<name>.md`. A shared `metadata.Close(store, path, status)` helper in the new `internal/metadata/close.go` wraps the read → split → merge → write pattern; it accepts a minimal `StoreReadWriter` interface so `internal/metadata` stays dependency-free from `internal/store`. Spec's `specStillScaffold` also gained the frontmatter-tolerant compare that `planDocStillScaffold` grew in Phase 1.2.

**Deviations**: None. Three parallel agents split the work (one per workflow) plus a single-agent test-writing pass — matches the plan's suggested "2-3 parallel agents" fan-out and shares the choke-point (`metadata.Close`) I authored inline before dispatch.

**Files changed**:
- `internal/metadata/close.go` (new) — `Close`, `StoreReadWriter` interface.
- `internal/metadata/close_test.go` (new) — four unit tests for `Close` (transition, bare-artifact upgrade, idempotency, read-error propagation) against an in-file `fakeStore`.
- `internal/steps/spec/steps.go` — `new()` merges scaffold through metadata; `finished()` closes the spec; `specStillScaffold` strips leading frontmatter before compare.
- `internal/steps/spec/steps_test.go` — four new callback-level tests.
- `internal/steps/plan/steps.go` — `finished()` closes all three plan siblings on non-incomplete runs; no other callback changed.
- `internal/steps/plan/steps_test.go` — three new tests covering close-all, incomplete-gate, no-mutation-when-incomplete.
- `internal/steps/implement/steps.go` — `finished()` closes test-plan.md and the changelog entry; treats `store.ErrNotFound` as no-op so a skipped test_plan step doesn't crash the terminal transition.
- `internal/steps/implement/steps_test.go` — two new tests for close-both and skip-missing.

**Discoveries**: The plan workflow's incomplete-doc gate was already tracking scaffold-vs-committed for the `plan_incomplete` warning, so it was the natural signal to reuse for "should we close?" — same predicate, opposite verdict, no new state. This means a plan whose author committed some but not all siblings will *not* have its written siblings closed at finished — that avoids half-completing the metadata lifecycle when the doc set itself is incomplete. `metadata.Close`'s minimal `StoreReadWriter` interface lets tests use a two-line in-memory fake instead of instantiating a `FileStore` on a temp dir; keeps the unit tests genuinely unit-scoped and independent of any store implementation.

### 2026-07-29 — Phase 2.1: Add filter flags to per-kind `<kind> file list`

**What was done**: Grew the shared `<kind> file list` command with five typed cobra flags — `--status`, `--created-after`, `--created-before`, `--closed-after`, `--closed-before` — combinable in a single query with AND semantics. When any filter is active, each candidate is read, its frontmatter split via `metadata.Split`, and only artifacts satisfying every predicate are returned. Bare (no-frontmatter) artifacts drop out of any filtered query but still surface in unfiltered listings. The output shape moved from bare strings (`["a.md","b.md"]`) to structured entries carrying `name`, `path`, and (when metadata is present) `created_date`, `status`, `closed_date`. Directory entries expose only `name` and `path`. Extracted the filter parse/match logic into a new `cmd/artifactfilter.go` so Phase 2.2's cross-kind aggregator can reuse the exact same predicate without duplication.

**Deviations**: None. Output shape change was accompanied by an update to the only in-tree consumer (`cmd/changelog_file_test.go`'s round-trip test) — its assertion was rewritten to accept the new struct-per-entry shape by name-projecting into a `[]string` for the ElementsMatch comparison.

**Files changed**:
- `cmd/storefile.go` — added five list flags and rewrote the `list` handler to read+filter+emit-structured; delegates filter parsing/matching to `parseListFilter`.
- `cmd/artifactfilter.go` (new) — `artifactFilter` struct, `active`, `matches`, `parseListFilter`. Owns date-boundary inclusivity semantics and enum validation. Intentionally private to `cmd` since it is glue for the CLI surface, not a domain type.
- `cmd/artifactfilter_test.go` (new) — 11 unit tests for the filter helper (zero-value inactive, per-filter parse, boundary inclusivity, closed-date filters reject zero closed_date, combined intersection, invalid-status and invalid-date error codes).
- `cmd/storefile_list_filter_test.go` (new) — 7 integration tests across spec/plan/changelog via a `listFilterFixtures` table plus a plan-specific top-level directory-entry test; introduces a `resetListFilterFlags` helper for the same cobra-global-state hazard the write/set-status tests already contend with.
- `cmd/changelog_file_test.go` — updated the round-trip test to parse the new struct-per-entry envelope and project a `[]string` of names for the ElementsMatch comparison.

**Discoveries**: The plan directory's top-level `list` returns directory entries (each plan is a sub-folder), not files — the new list handler correctly emits `{name, path}` for those without attempting a frontmatter read. This means `plan file list` at the top level shows plans by folder name, while `plan file list <plan_name>` shows the four sibling docs inside a plan with their metadata. Also: date filters use `time.Time.Before`/`After`, which are strict (not inclusive) — the boundary-inclusive semantics required by the acceptance criteria fall out naturally because a date equals another date is neither Before nor After, so an equal date matches automatically.

### 2026-07-29 — Phase 2.2: Ship a cross-kind `spektacular artifacts list` aggregator

**What was done**: Added a new top-level `spektacular artifacts list` command that scans all three configured store subtrees (specs, plans/<plan>/sibling-docs, changelog) and returns a single JSON envelope `{"artifacts": [...]}` where each entry carries a `kind` discriminant — one of `spec`, `plan.plan`, `plan.context`, `plan.research`, `plan.test-plan`, `changelog` — plus `name`, `path`, and (when metadata is present) `created_date`, `status`, `closed_date`. Reuses the exact `parseListFilter`/`artifactFilter` from Phase 2.1 so the five filter flags carry identical semantics across the per-kind and cross-kind surfaces. A `--kind` flag accepts a comma-separated list of discriminants; invalid values return `invalid_kind`. Missing configured directories are treated as empty (defence-in-depth for fresh workspaces).

**Deviations**: The initial implementation passed an absolute path to `store.FileStore.List`, but that helper internally re-joins its argument with the store root — the aggregator was silently returning empty results in every real scenario. Caught by the Phase 2.2 acceptance tests immediately and fixed inside this phase: the three `append*Artifacts` helpers now pass the store-relative directory (e.g. `cfg.Spec.Config.Directory`, i.e. `"docs/specs"`) to `st.List`. `scanArtifact`'s pre-existing `TrimPrefix(storePath, st.Root()+…)` becomes a no-op on the now-relative path, which is what we want because that relative form is also the desired `path` field.

**Files changed**:
- `cmd/artifacts.go` (new) — `artifactsCmd`, its `list` subcommand, three `append*Artifacts` helpers, `scanArtifact` helper, `parseKindFlag` helper, kind constants, `planSiblingKinds` map.
- `cmd/root.go` — registered `artifactsCmd` under the root command.
- `cmd/artifacts_test.go` (new) — 12 tests: unfiltered/kind/status/date/combined coverage across all classes, bare-artifact handling, empty-store shape, missing-directory tolerance, plus three `parseKindFlag` unit tests.

**Discoveries**: `store.FileStore.List` accepts a store-relative path and internally joins with `root` — passing an absolute path double-joins into a nonsense location that `os.ReadDir` reports as ErrNotExist, which then trips the "empty directory" fallback silently. This affected the aggregator's whole-tree scan (it returned empty in real usage) and was invisible without a proper end-to-end integration test. The write handler in `cmd/storefile.go` gets away with `filepath.Join(storeDir, args[0])` because `storeDir` there is the store-relative dir (`dir(cfg)`), not absolute — the aggregator's mistake was passing `filepath.Join(st.Root(), cfg.X.Config.Directory)` at all. The pattern to remember: `store.Store.List` and `store.Store.Read`/`Write` always take store-relative paths; `st.Root()` is for building on-disk assertions (e.g. `os.ReadFile`), not for feeding back into store calls.
