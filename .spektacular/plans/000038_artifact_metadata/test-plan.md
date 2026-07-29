---
created_date: "2026-07-29"
status: completed
closed_date: "2026-07-29"
---

# Test Plan: 000038_artifact_metadata

The automated test suite (unit + integration + workflow tests across `internal/metadata`, `cmd/`, and `internal/steps/{spec,plan,implement}`) covers three of the five success metrics from the spec. The remaining two require manual verification against real workflow output — an artifact-carries-metadata spot check and a design-longevity review that no unit test can prove.

## Manual metric 1 — Every new workflow document carries metadata (spot check)

**What to measure**: After this feature ships, every artifact produced by any Spektacular workflow (spec, plan, implement) carries a `---` fenced YAML frontmatter block at the top of the file with `created_date`, `status`, and (when closed) `closed_date` fields.

**How**:

1. In a clean Spektacular-initialized repository, run each workflow end-to-end against real inputs:
   - `go run . spec new --data '{"name":"probe-feature"}'` and drive it through every step to `finished` via the `spek-new` skill.
   - `go run . plan new --data '{"name":"probe-feature"}'` and drive it through every step to `finished` via `spek-plan`.
   - `go run . implement new --data '{"name":"probe-feature"}'` and drive at least one phase through to `finished` via `spek-implement`.
2. For each artifact produced, run `go run . spec file read probe-feature.md | head -6` (and equivalents for `plan file read probe-feature/{plan,context,research,test-plan}.md` and `changelog file read probe-feature.md`).
3. Also run the aggregator: `go run . artifacts list --kind spec,plan.plan,plan.context,plan.research,plan.test-plan,changelog`.

**Expected result**:

- Every artifact begins with a `---\n` line, followed by YAML containing at minimum `created_date: <YYYY-MM-DD>` and `status: <in-progress|completed>`, then a `---\n` closing line, then the artifact's body.
- After each workflow reaches `finished`, its owned artifacts show `status: completed` and a `closed_date: <YYYY-MM-DD>` matching the finish date.
- `spektacular artifacts list` returns entries for every produced artifact, each with a non-empty `kind`, `name`, `path`, `created_date`, and `status` field (and `closed_date` where applicable).

**Pass condition**: Zero probe artifacts are missing frontmatter; zero closed workflows leave their artifacts in `in-progress`.

**Who / when**: Release owner, immediately post-ship against staging or a fresh scratch repo, and once again 24 hours later against real workflow output produced by real users, to catch any workflow path the internal tests did not exercise.

## Manual metric 2 — Schema is sufficient for future downstream features

**What to measure**: The three-field schema (`created_date`, `status`, `closed_date`) exposed via frontmatter and via `spektacular artifacts list` is sufficient signal to build (a) an automatic archive sweep that mutates stale artifacts, and (b) reports or dashboards derived from list output — without any schema addition or CLI shape change.

**How**:

1. Design (on paper, no code) two hypothetical future features:
   - **Auto-archive**: a command that scans for artifacts whose `status: completed` was stamped more than N days ago and calls `<kind> file set-status --status archived` on each.
   - **Dashboard**: a report that groups artifacts by `status` and buckets `created_date`s into monthly cohorts, e.g. "5 in-progress plans older than 30 days, 2 completed specs from Q1 2026".
2. For each design, walk through:
   - Can the feature be implemented using only the three existing fields on `Metadata` (no `Modified`, no `Priority`, no `Owner`)?
   - Can the query be expressed with the existing five filter flags (`--status`, `--created-{after,before}`, `--closed-{after,before}`)?
   - Would either feature require a new CLI subcommand shape (beyond the existing `<kind> file set-status`, `spektacular artifacts list`), or does the existing surface suffice for its actions?

**Expected result**:

- Both designs are implementable using only the shipped schema and CLI surface.
- If either design requires a new schema field or a new CLI verb to work at all, the schema is insufficient — flag the specific gap so a follow-up spec can address it before either hypothetical is built for real.

**Pass condition**: Both designs are demonstrably implementable against the shipped surface with no schema/CLI changes required. A single "yes, this works" from the reviewer is the pass signal.

**Who / when**: Feature owner (or a peer with equivalent context), performed once as the last review gate before this plan is marked closed. Repeat only when a concrete future feature is being scoped and the reviewer wants to re-confirm the schema still covers it.
