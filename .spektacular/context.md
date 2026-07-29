# Implement: 000038_artifact_metadata — working context

## Workflow

- Implement workflow started against plan `000038_artifact_metadata` on 2026-07-28.
- Plan store: `.spektacular/plans/000038_artifact_metadata/{plan.md,context.md,research.md}`.
- Source spec: `.spektacular/specs/000038_artifact_metadata.md`.
- Read all three plan documents via `go run . plan file read` (never with `Read` tool).

## Read-plan step outcomes

- **Structural validation:** plan.md has all 10 required top-level sections (Overview, Architecture & Design Decisions, Component Breakdown, Data Structures & Interfaces, Implementation Detail, Dependencies, Testing Approach, Milestones & Phases, Open Questions, Out of Scope). Seven `#### - [ ]` phases across two milestones, each with a `context.md#phase-*` link that resolves.
- **Drift check: NO DRIFT.** Every file path, symbol, template path, and CLI command referenced in the plan exists at its stated location. `internal/metadata/` correctly does NOT exist yet (it's a new package). No existing `spektacular artifacts` command wired.
- **Spec coverage:** every Requirement and Acceptance Criterion in the spec is covered somewhere under `## Milestones & Phases` in the plan. No descoped items.
- **Changelog mode: FIRST-PHASE INVOCATION.** No `## Changelog` section in plan.md. Pick up at Phase 1.1.

## Open Question resolutions (from drift check)

**Open Question #1 — consumers of `{"files": [...]}` list shape (relevant for Phase 2.1):**

- **Only one producer:** `cmd/storefile.go:159` — `return output.Write(cmd.OutOrStdout(), map[string]any{"files": names}, "")`.
- **Only one code test consumer:** `cmd/changelog_file_test.go:54` — unmarshals into `struct { Files []string }`. Will need update to accept the new struct-per-entry shape.
- **Template/skill references** (call the command, don't parse the shape): `templates/skills/workflows/spek-{new,plan,implement}/SKILL.md`, `templates/steps/plan/02-discovery.md`, `templates/skill_list_command_test.go`.

**Open Question #2 — in-Go `st.Write` call sites (relevant for Phase 1.5):**

Production writers via `git grep -n 'st\.Write' internal/steps/ cmd/`:

1. `cmd/storefile.go:97` — the shared `newStoreFileCmd` write subcommand (covered by Phase 1.3).
2. `internal/steps/spec/steps.go:79` — spec workflow `new()` scaffold write (Phase 1.5 routes through `metadata.Merge`).

The implement and plan workflows do NOT call `st.Write` directly from step callbacks — they emit template instructions telling the agent to call `plan file write` / `changelog file write` via the CLI, which flows through the storefile.go write handler. The finished-step close in Phase 1.5 therefore needs to go through the CLI (or through the shared metadata helper called from Go with the store handle the callback already has) — no additional in-Go `st.Write` sites to worry about.

## Key decisions carried from plan authorship

1. **Owner-workflow closes.** Spec workflow → spec.md at `finished`. Plan workflow → plan.md/context.md/research.md at `finished`. Implement workflow → test-plan.md + changelog entry at `finished`. Downstream body mutations do NOT re-open metadata (spec 000034 checkbox ticking pattern).
2. **Two update surfaces:** implicit on `<kind> file write` + explicit metadata-only `<kind> file set-status`.
3. **Enum accepts all four values day one:** `in-progress`, `completed`, `superseded`, `archived`. Workflows only write `in-progress` and `completed`; `superseded`/`archived` reachable only via `set-status`.
4. **Store interface stays byte-oriented.** Metadata sits above the substrate. No new fields on `DirEntry` / `Hit`.
5. **YAML dates are `YYYY-MM-DD`** (day precision, no time-of-day). Custom `MarshalYAML`/`UnmarshalYAML` on a date wrapper enforces this.
6. **No backfill.** Pre-shipping bare docs stay byte-identical; excluded from any filtered list query.

## Load-bearing gotchas

- **`planDocStillScaffold` byte-compare (Phase 1.2)** at `internal/steps/plan/steps.go:215-225`. MUST land before Phase 1.3 or plan workflow regresses.
- **Malformed hand-edited frontmatter** must produce actionable error, never silent re-stamp of `created_date`.
- **List output shape change in Phase 2.1** — update `cmd/changelog_file_test.go:54` and check the two template test files.

## Phase execution order

Milestone 1: 1.1 → 1.2 → 1.3 → 1.4 → 1.5.
Milestone 2: 2.1 → 2.2.

Phase 1.5 is the fan-out candidate (three workflow packages, same pattern). Others are single-agent.

## Current phase

**Phase 1.1: Author the `internal/metadata` package** — code written and compiling.

Design decisions made during implementation:

- Chose `Metadata.MarshalYAML/UnmarshalYAML` with an unexported `yamlShape` (string-typed date fields) rather than a public `Date` wrapper. Public API keeps `time.Time` fields as the plan's Data Structures section specifies; day precision enforced at marshal boundary.
- Enum validation lives on `UnmarshalYAML` AND on `Merge` (via `validateStatus`). Both paths reject non-enum values with an actionable error.
- `Split` returns `(nil, raw, nil)` for bare artifacts (no leading `---\n`); wrapped `malformed frontmatter:` error for unterminated block, non-newline after closing `---`, or yaml.Unmarshal failure.
- `Render` writes `---\n<yaml>---\n\n<body>`. Note yaml.Marshal already emits a trailing newline on its own output, so the sequence is `---\n` + yaml + `---\n\n` + body.
- `Merge` `ClosedDate` semantics: preserves existing ClosedDate if already set (idempotent for closed→same-closed and closed→different-closed); stamps today only on in-progress→closed transition; clears on closed→in-progress. First-write with `opts.Status = closed` stamps ClosedDate=today.

Files created:

- `internal/metadata/metadata.go` — `Metadata`, `Status`, four consts, MarshalYAML/UnmarshalYAML, private `validateStatus`/`isClosed`.
- `internal/metadata/frontmatter.go` — `Split`, `Render`.
- `internal/metadata/merge.go` — `Merge`, `UpdateOptions`.

## Autonomous mode

User authorized "run all remaining phases without asking" on 2026-07-28 after Phase 1.1 committed. Do NOT ask between phases; loop analyze → implement → test → verify → update_plan → update_changelog → analyze until all seven phases are ticked, then advance to `update_repo_changelog`.

## Session state

- Phase 1.1 complete and committed (plan ticked, changelog entry added). Advance next: loop back to `analyze` for Phase 1.2.

Note on YAML quoting: `yaml.v3` quotes date-shaped string fields (e.g. `created_date: "2026-07-01"`) — tests assert the presence of the value without pinning quote style.
