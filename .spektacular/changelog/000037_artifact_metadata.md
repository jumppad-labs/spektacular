---
created_date: "2026-07-29"
status: completed
closed_date: "2026-07-29"
---

# Feature: 000037_artifact_metadata — Per-artifact lifecycle metadata and cross-kind discovery

## What was built

Every document Spektacular writes as workflow output — specifications, plan documents (plan.md, context.md, research.md, test-plan.md), and per-change changelog entries — now begins with a `---` fenced YAML frontmatter block carrying three fields: `created_date` (stamped once, when the workflow first writes the document), `status` (one of `in-progress`, `completed`, `superseded`, `archived`), and `closed_date` (stamped when the status first transitions out of `in-progress`). Each workflow attaches the block on first write and flips its own artifacts to `completed` at its terminal step; the metadata never drifts because the CLI's shared write handler routes every commit through a single Go package (`internal/metadata`) that owns the schema.

Two query surfaces let users and agents find artifacts by that metadata without opening the files:

- The existing `spec file list`, `plan file list`, and `changelog file list` commands gained five combinable flags — `--status`, `--created-after`, `--created-before`, `--closed-after`, `--closed-before` — with AND intersection semantics and inclusive date boundaries. The list output shape moved from bare strings to structured entries carrying `name`, `path`, and the metadata fields.
- A new top-level `spektacular artifacts list` command scans all three configured store subtrees at once and returns a single JSON envelope where each entry is tagged with a `kind` discriminant (`spec`, `plan.plan`, `plan.context`, `plan.research`, `plan.test-plan`, `changelog`). A `--kind` flag accepts a comma-separated list to narrow the scan.

A metadata-only `<kind> file set-status <path> --status <s>` subcommand shipped alongside so workflows and humans can flip a document to `superseded` or `archived` (or any status) without a body rewrite.

Existing pre-shipping artifacts stay byte-identical to what they were before this feature landed — no backfill sweep is performed. They surface in unfiltered listings but drop out of any query that sets a metadata filter (they have no frontmatter to match), which matches the "no backfill" acceptance criterion in the spec.

## Why it matters

Before this feature, anyone triaging accumulated workflow output had to inspect git history or read document bodies to work out how old an artifact was or whether its workflow had ever completed. Downstream automation had no signal at all — no way for a future auto-archive sweep, a stale-plan report, or an oncall who needs to know "is this spec still active?" to answer that question without a per-file investigation. Agents doing archaeological reads on historical specs and plans (a pattern established by `000036_spec_plan_historical_artifacts`) now have machine-readable evidence of an artifact's age and lifecycle, and can weight information accordingly rather than treating a two-year-old superseded plan as equally authoritative to yesterday's completed one.

The cross-kind `spektacular artifacts list` closes a specific gap that the per-kind commands alone could not: a single query that answers "what is in flight right now?" or "what completed in the last week?" across every artifact class, without a scripted union in the caller.

## Deviations from the plan

Two implementation-time discoveries reshaped the delivery slightly, both flagged in the plan's inline `## Changelog`:

- **Write-handler idempotency (Phase 1.3 hotfix during Phase 1.4)**: the initial write handler merged over the raw source bytes, so any `<kind> file write --from <src>` on a source that already carried a frontmatter block accumulated another block on top. This surfaced immediately during this workflow's own `plan file write` calls — plan.md accumulated three stacked blocks in under a minute. Fixed by adding `stripLeadingFrontmatterBlocks` in `cmd/storefile.go` (loops `metadata.Split` until no leading block remains) and a `TestStoreFileWrite_IdempotentUnderRepeatedWrites` regression guard.
- **Aggregator scan path (Phase 2.2 hotfix during testing)**: the first draft of `spektacular artifacts list` passed absolute paths to `store.FileStore.List`, which internally re-joins with the store root and produced a nonsense path that `os.ReadDir` reported as "not found" — silently returning an empty envelope in every real scenario. Caught by the Phase 2.2 integration tests and fixed by passing store-relative directories (e.g. `cfg.Spec.Config.Directory`) instead. The lesson: `store.Store.List/Read/Write` always take store-relative paths; `st.Root()` is for on-disk assertions, never for feeding back into store calls.

Neither deviation changed the plan's scope or the spec's requirements; both landed inside the phases where they were discovered.

## Public surface added

- `spec file write --status <s>`, `plan file write --status <s>`, `changelog file write --status <s>` — optional status flag on the existing write command.
- `spec file set-status <path> --status <s>`, `plan file set-status <path> --status <s>`, `changelog file set-status <path> --status <s>` — new subcommand.
- `spec file list [--status] [--created-{after,before}] [--closed-{after,before}]` — plus the two sibling commands.
- `spektacular artifacts list [--kind] [--status] [--created-{after,before}] [--closed-{after,before}]` — new top-level command.
- List output shape changed from `{"files": ["a.md", ...]}` to `{"files": [{"name": ..., "path": ..., "created_date"?: ..., "status"?: ..., "closed_date"?: ...}, ...]}`.

## Package added

- `internal/metadata` — owns the frontmatter schema (`Metadata`, `Status` and its four constants), YAML round-trip through `MarshalYAML`/`UnmarshalYAML` with day precision, and the four helpers `Split`, `Render`, `Merge`, `Close`.
