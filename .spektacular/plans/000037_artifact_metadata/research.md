# Research: 000037_artifact_metadata

## Alternatives considered and rejected

### Update mechanism

- **Rejected: dedicated close command only, no shared writer helper.** Each write site (`internal/steps/spec/steps.go:75-79`, `cmd/storefile.go:79-98`, plan `writePlan`/`writeContext`/`writeResearch` at `internal/steps/plan/steps.go:245-275`, `test_plan` step in `internal/steps/implement/steps.go`, `update_feature_changelog` step in the same file) would each re-implement frontmatter formatting. Rejected because there are five+ write sites already and any drift between them produces docs whose metadata parses differently — exactly the class of bug the spec's "machine-readable" requirement is trying to prevent.
- **Rejected: agent-authored frontmatter in scratch files.** The agent stages `.spektacular/tmp/plan_template.md` etc. before `plan file write --from` (`templates/steps/plan/15-write_plan.md:9`). We could have the agent write the frontmatter into the scratch file. Rejected because it moves invariant enforcement (schema, closed_date semantics, created_date preservation) into prompt text, where it will drift the moment step templates get edited. Frontmatter is a machine concern; keep it in Go.
- **Chosen (from user): implicit-on-write in the `<kind> file write` path + a metadata-only subcommand for status-only transitions.** The write path always renders frontmatter (preserving `created_date`, enforcing enum on `status`, stamping `closed_date` iff transitioning to a closed status); the metadata-only subcommand covers status flips that don't need a body rewrite (terminal `completed` at workflow close, later `superseded`/`archived` moves).

### List / filter surface

- **Rejected: per-kind flags only.** Meets the spec's per-artifact requirement, but leaves cross-kind queries as a user-side merge job. Spec explicitly says filters must be "applicable across all covered artifact classes" — a single query, not a scripted union.
- **Rejected: cross-kind aggregator only.** Removes convenient per-kind exploration. Also increases the smallest useful surface — every filter has to be authored against the multi-kind envelope even when the user only wants one kind.
- **Chosen (from user): both.** Extend `spec file list` / `plan file list` / `changelog file list` with typed cobra flags for `--status` / `--created-{after,before}` / `--closed-{after,before}`; add a new `spektacular artifacts list` aggregator that returns hits from all four artifact classes tagged with a `kind` discriminant.

### Close semantics

- **Rejected: "completed" = implement workflow finished** (i.e., plan is completed only after the work described in it has shipped). Would leave plans that were written but never implemented stuck in `in_progress` indefinitely, and it conflates two lifecycles (writing done vs. shipping done). Also inconsistent with the spec wording "workflow closed successfully" — the workflow that owns writing.
- **Rejected: two-phase status (in-progress → planned → implemented → …).** Expands the enum beyond the four values the spec named; other artifacts (specs, changelog) don't need the middle state.
- **Chosen (from user): the workflow that writes a document is the one that closes it.** Spec workflow ships → spec.md becomes `completed`. Plan workflow ships → plan.md/context.md/research.md become `completed`. Implement workflow ships → test-plan.md and its changelog entry become `completed`.

### Superseded / archived

- **Rejected: two-value enum in v1** (in-progress + completed only). Would force a schema migration when superseded/archived land, and would fail the acceptance criterion "status field, whose value is one of: in-progress, completed, superseded, archived" if that criterion is read strictly at ship time.
- **Rejected: ship manual `supersede` / `archive` CLI commands now.** Non-goal on auto-archive suggests these are follow-up product decisions; not required by any AC in this spec.
- **Chosen: enum accepts all four values from day one; only in-progress and completed are workflow-writable in v1. Superseded and archived are only reachable through the metadata-only subcommand** (which the user confirmed should exist "in addition to file save writing metadata … a sub command that allows metadata update without writing the file"). Validators enforce enum membership so a hand-edited file with a typoed status is caught.

## Chosen approach — evidence

- **Frontmatter (YAML `---` fenced) as the substrate.** Precedent already in the tree at `internal/agent/agent_test.go:226-250` (`validateSkillFrontmatter`) parses `---\n<yaml>\n---\n<body>` on skill docs. `gopkg.in/yaml.v3` is already imported (`internal/config/config.go:176,293`) — no new dependency.
- **New `internal/metadata` package as the shared writer helper.** Justified by the five current write sites: `internal/steps/spec/steps.go:75-79` (spec scaffold), `cmd/storefile.go:79-98` (all agent-driven `<kind> file write --from` calls), `internal/steps/plan/steps.go:245-275` (three plan siblings), `internal/steps/implement/steps.go` (test_plan + update_feature_changelog steps). All of them eventually call `store.Store.Write(path, content)` (`internal/store/store.go:102-111`). Centralising frontmatter render/parse/merge in one Go package and calling it from each site avoids reimplementation drift.
- **Storage substrate delegated per file writer.** Store interface (`internal/store/store.go:37-55`) is deliberately byte-oriented (`Read/Write/List/Search`) — extending `Hit`/`DirEntry` with typed metadata fields would leak markdown-specific concerns into the substrate. Keeping the store dumb + layering metadata parsing above (in a filter step for `list` and a helper for `write`) matches the spec's "delegated to each file writer" direction and leaves the door open for non-file substrates.
- **Existing YAML native date type covers the "date" wording** in the spec. `YYYY-MM-DD` renders cleanly through `yaml.v3` and matches spec language ("creation date", "closing date").
- **Owner-workflow close aligned with `state.go`'s existing `InProgress` predicate** (`internal/workflow/state.go:26`: `CurrentStep != "" && CurrentStep != "finished"`) — the workflow already knows when it is "still open" vs "closed successfully"; the metadata write on `finished` is a direct read of that same signal.
- **ID prefix validation in `cmd/storefile.go:27-41`** already enforces that plan/changelog writes reuse the spec ID. Any new `<kind> file set-status <path>` subcommand goes through the same `newStoreFileCmd` machinery so it inherits validation for free.

## Files examined

- `.spektacular/specs/000037_artifact_metadata.md` — the spec being planned; core fields (created, status enum, closed_date), no backfill, YAML frontmatter is "natural fit", substrate delegated per writer, filters combinable across classes.
- `.spektacular/specs/000034_spec-plan-implement-reconciliation.md` — established precedent for the implement workflow mutating spec content (checkbox ticks); relevant because it's another cross-workflow write path but does not itself change spec metadata.
- `.spektacular/specs/000036_spec_plan_historical_artifacts.md` — established the archaeology / owning-workflow distinction. Confirms that the workflow that owns a doc is the natural owner for lifecycle transitions.
- `internal/store/store.go:1-152` — Store interface (Read/Write/List/Search/Exists/Delete/Root), `DirEntry{Name, IsDir}` (no metadata), `Hit{Scope, Path, Title, Excerpts, Score, Category, Checksum}` (no metadata), `FileStore` implementation. Byte-oriented; substrate-agnostic.
- `internal/workflow/state.go:14-27` — `State{Kind, CurrentStep, CompletedSteps, CreatedAt, UpdatedAt, Data}`, `InProgress()` (line 26). Terminal state hardcoded to `"finished"`. This is the signal for owner-workflow close.
- `internal/steps/spec/steps.go:66-93` — `new()` step: renders scaffold via `stepkit.RenderTemplate("scaffold/spec.md", …)` and calls `st.Write(SpecFilePath(cfg.SpecDir, name), …)`. First-write site for specs. Metadata injection needed here.
- `internal/steps/spec/steps.go:150-168` — `finished()` step; reads spec back to verify, no metadata mutation today. Close-transition site.
- `internal/steps/plan/steps.go:161-182` — `assemble()` renders three scaffolds and stages them via the `Extra` map; agent writes them to `.spektacular/tmp/` via own tool.
- `internal/steps/plan/steps.go:245-275` — `writePlan()/writeContext()/writeResearch()` verify staged docs are still scaffolds, then the agent commits via `plan file write`. Frontmatter must be injected before or during the `store.Write()` call, not in the scratch file.
- `internal/steps/plan/steps.go:215-225` — `planDocStillScaffold()` reads back from store to verify commit; note: this reads current content — frontmatter injection must not break this check (either the check ignores frontmatter or it re-renders scaffold with frontmatter).
- `internal/steps/plan/steps.go:277-299` — plan `finished` step. Close-transition site for the three plan-store siblings.
- `internal/steps/implement/steps.go` — test_plan and update_feature_changelog callbacks; write paths for test-plan.md and changelog entries. `finished` step is the close-transition site for those two.
- `cmd/storefile.go:71-165` — the shared `newStoreFileCmd` used by `spec file`, `plan file`, `changelog file`. Wraps subcommands `{write, read, delete, list}`. `list` handler at lines 138-161 returns `{"files": [name, …]}`. `write` at 79-98 currently does nothing about frontmatter. This is the single point where implicit-on-write frontmatter handling lands.
- `cmd/storefile.go:27-41` — `validateIDPrefix` — reuse for the new `set-status` subcommand.
- `cmd/file.go:8-12`, `cmd/plan_file.go:8-14`, `cmd/changelog_file.go` — three thin factories that wire the shared `newStoreFileCmd` per kind. Where to attach the new `set-status` subcommand and the filter flags.
- `cmd/spec.go:71-402`, `cmd/plan.go:38-305`, `cmd/changelog.go` — top-level per-kind cobra command hierarchy; where a cross-kind aggregator (`spektacular artifacts list`) would attach at root instead.
- `cmd/root.go:214-228` — root command registration; the mount point for a new `artifactsCmd`.
- `internal/output/writer.go:45-52,92-161` — CLI success/error envelope. Success auto-injects `"error": false`. Filtered list output goes through this. Cross-kind aggregator returns `{"error": false, "artifacts": [...]}` following the same shape.
- `internal/config/config.go:176,293` — existing `gopkg.in/yaml.v3` usage. Confirms the dependency is available.
- `internal/agent/agent_test.go:226-250` — existing YAML frontmatter parser precedent (skill docs). Not directly reusable (test-scoped) but confirms the `---\n...\n---\n<body>` shape.
- `templates/scaffold/spec.md` — pure markdown, no frontmatter today. Metadata is prepended by the write path, not baked into the scaffold; scaffold stays intent-only.
- `templates/steps/plan/13-assemble.md:19-37,59-67` — plan assembly instructions to the agent; the working files listed there feed into the scratch templates. No template change needed to add frontmatter — it happens on write, not in the assembly instruction.
- `templates/steps/plan/15-write_plan.md:9` (and 16/17 write_context/write_research templates) — `plan file write` invocation; unchanged, but the CLI it invokes learns to handle frontmatter.
- `templates/steps/implement/09-test_plan.md:28-40` — `plan file write .../test-plan.md --from …` invocation; same story.
- `templates/steps/implement/10-update_feature_changelog.md` — `changelog file write` invocation; same story.
- `internal/store/search.go:1-50` — free-form keyword search. Not used for the list/filter path; kept as-is. Filter path reads each candidate file's frontmatter after `st.List()` and matches against flag values.

## External references

- `gopkg.in/yaml.v3` — already a dependency; docs at https://pkg.go.dev/gopkg.in/yaml.v3. Marshals `time.Time` values with the `time` YAML tag; a `!!timestamp` or `!!str`-shaped date field renders as `YYYY-MM-DD` when the value has zero time-of-day and is written via a `yaml:",omitempty"` friendly struct. Relevant for the schema struct.
- YAML 1.2 spec (schema resolution rules) — https://yaml.org/spec/1.2.2/#10214-timestamp. Confirms `YYYY-MM-DD` is a natively recognised timestamp.
- Common markdown frontmatter convention (Jekyll/Hugo lineage) — `---` fenced block at file top, blank line, body. No formal RFC but the de-facto shape agents and humans both recognise. Reinforces the spec's "YAML frontmatter is the natural fit" direction.

## Prior plans / specs consulted

- **Spec `000034_spec-plan-implement-reconciliation`** — established the precedent that the implement workflow ticks checkboxes in a shipped spec. That precedent is *body* mutation of a shipped doc, not metadata mutation; it doesn't touch this feature's design directly but confirms cross-workflow mutation of shipped documents is already an accepted pattern in the codebase.
- **Spec `000036_spec_plan_historical_artifacts`** — established the archaeology / owning-workflow distinction. Confirms our decision that the owning workflow closes its own docs. Also flags a light constraint: this feature must not accidentally require historical (metadata-less) docs to be read en masse during discovery.
- **Spec `000033_spec-workflow-output-changelog`** — established that a changelog entry is one `.md` per feature under `.spektacular/changelog/`. Confirms shape assumed in the plan.
- **No prior *plan* directly consulted** (nothing about metadata, frontmatter, or artifact status has been planned before).

## Open assumptions

If any of these turn out wrong, the implement workflow must STOP and ask:

- **The four artifact classes are exhaustive** for what the spec calls "Spektacular workflow output": (a) specs in `.spektacular/specs/*.md`, (b) plan-directory siblings (`plan.md`, `context.md`, `research.md`, `test-plan.md`) under `.spektacular/plans/<id>_<name>/`, (c) changelog entries under `.spektacular/changelog/*.md`. Anything else (e.g. `.spektacular/knowledge/` content, `.spektacular/state.json`) is out of scope per the spec's Non-Goals.
- **`created_date` at first-ever write of a file is the correct value**, not the workflow's `CreatedAt`. If a spec is re-created after being deleted, `created_date` is *the new file's* birth, not the original workflow's. This matches the spec's "the date the document was first written."
- **Preserving `created_date` on subsequent writes** is done by parsing existing frontmatter at the store path *before* writing. If the parse fails (malformed frontmatter that a human hand-edited into an invalid state), the write returns an actionable error rather than silently stamping a new date. The user picked the "implicit on write" mechanism specifically to avoid drift; catching malformed frontmatter is required for that guarantee.
- **YAML `!!timestamp` (or a date string emitted as such) is the correct field type** for `created_date` and `closed_date`. Not a Unix epoch, not RFC3339 with time-of-day. Spec says "date" not "timestamp"; workflow completion is per-day granularity in practice.
- **`planDocStillScaffold` at `internal/steps/plan/steps.go:215-225`** compares against the rendered scaffold body. Adding frontmatter on the store-side does not break this check as long as the check strips or ignores a leading frontmatter block. This is the only known place a store read is byte-compared against a template; if any similar comparisons exist in `internal/steps/spec/` or `internal/steps/implement/`, they need the same treatment.
- **Cross-kind aggregator command name `spektacular artifacts list`** doesn't clash with an existing verb. `artifacts` is a fresh top-level noun (unlike `spec`/`plan`/`changelog` which are per-kind); confirming during architecture step.
- **`test-plan.md` is the only additional sibling** the implement workflow writes into a plan directory. If future implement steps grow more siblings (or if the current codebase already has one I missed), the metadata coverage list needs to grow with them.
- **Filter flag defaults are "no filter"** (empty string). An empty `--status` means "any status" (including bare docs with no metadata). A `--status in-progress` filter excludes bare docs (they have no metadata to match), and per the spec's AC that is correct behaviour ("returns the set of artifacts whose current status matches").

## Rehydration cues

If this file is being read cold in a future session:

- `go run . spec file read 000037_artifact_metadata.md` — the source spec.
- `go run . plan file read 000037_artifact_metadata/plan.md` — this feature's own plan (may not be committed yet).
- Read `.spektacular/context.md` for the running decision log and the summary of the user interview.
- Read `.spektacular/work/000037_artifact_metadata/*.md` for per-section drafts written en route.
- `cat internal/store/store.go` for the byte-oriented store interface (metadata layer sits above this).
- `cat cmd/storefile.go` for the `newStoreFileCmd` factory used by `spec file`, `plan file`, `changelog file` — where write/read/list/delete/set-status subcommands all attach.
- `cat internal/workflow/state.go` for `State.InProgress()` — the signal a workflow uses to know it's still open.
- `ls .spektacular/plans/000035_plan-walkthrough-conversation/` (or any recent plan dir) — the exhaustive list of plan-directory siblings (plan.md, context.md, research.md, test-plan.md).
- `go run . knowledge search "metadata"` and `go run . knowledge search "frontmatter"` — knowledge base is thin here; expect low signal.
