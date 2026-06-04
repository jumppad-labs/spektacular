# Research: 000025_convention-aware-planning

## Alternatives considered and rejected

- **Reuse `knowledge list` + `knowledge read` instead of a new `conventions` command.**
  Rejected: the workflow skills forbid reading knowledge entries with the raw `Read` tool and steer all access through dedicated CLI verbs; a single fan-out command that returns every convention *body* in one call is what the discovery step needs (one call, full digest), and the spec explicitly sketches `knowledge conventions`. `List` returns only `{scope,path}` (`internal/knowledge/set.go:111-123`), not bodies, so it would force N follow-up `read` calls.

- **Filter `conventions/` out of search at the `knowledge.Set` layer (post-filter hits by path prefix).**
  Rejected: the spec requires the exclusion be applied *identically in both* the ripgrep path and the native-walk fallback so the two stay equivalent (`internal/store/search_test.go:79-97` asserts equivalence). A post-filter in the Set layer would technically work but the spec is explicit the exclusion belongs in `FileStore.Search` itself, and a post-filter would still walk/scan the conventions files wastefully. Inject in both `searchRipgrep` (a `--glob=!conventions/**` arg) and `searchNative` (a `filepath.SkipDir` on a `conventions` dir).

- **Keep conventions as a single flat `conventions.md` file (no directory).**
  Rejected by spec requirement: "A user must be able to add or change a single always-apply convention as an independent item, without editing unrelated conventions." One-rule-per-file under `conventions/` is the authoring-ergonomics requirement. Files are concatenated when read.

- **Give the implement workflow new knowledge access so it can re-read conventions.**
  Rejected: explicit non-goal. Knowledge is a planning-time input only; the plan documents are the distilled output implement consumes. No change to implement steps.

- **Auto-decide convention relevance (no human review).**
  Rejected: explicit non-goal. Relevance is *proposed* during planning and confirmed by the user.

- **Migrate an existing flat `conventions.md` into `conventions/` on init.**
  Rejected by user decision: the project is still fresh and accepts breaking changes, so there is no existing flat conventions file worth preserving. Carrying migrate-and-guard ordering in `init.go` (move before the seed write at `:85-92`) would add complexity with no real beneficiary. This waives the spec's "existing projects must keep working" constraint for this project; init simply stops writing the flat file and seeds into the directory instead.

## Chosen approach — evidence

- **New `knowledge conventions` read-only subcommand**, fanning across scopes like `list`/`search`, returning every file body under `conventions/` per scope, scope-tagged. Wire following the exact cobra pattern at `cmd/knowledge.go:45-49` (command var), `:78-81` (output schema), `:191-201` (runner — `sources` is the closest analogue: no `--data`), `:251` (register in `AddCommand`). Backing `Set` method modelled on `List` (`internal/knowledge/set.go:111-123`) but reads each file via `src.store.Read` and returns bodies. Output envelope `{"conventions":[{scope,path,content}]}` (new struct, since `Entry` at `:30-33` has no content field).
  - **Must tolerate a scope with no `conventions/` dir**: `listFiles(st,"conventions")` calls `Store.List("conventions")` which returns `ErrNotFound` (`internal/store/store.go:124-141`) when absent. The method must treat not-found as "zero conventions for this scope", not an error — otherwise a fresh project with no conventions dir breaks every plan run.

- **Exclude `conventions/` from `FileStore.Search`** in both paths:
  - Ripgrep: add `"--glob=!conventions/**"` to the args slice at `internal/store/search.go:66` (between `--ignore-case` and `query`).
  - Native fallback: in the `filepath.WalkDir` callback (`internal/store/search.go:112-118`), before the `d.IsDir()` skip, return `filepath.SkipDir` when `d.IsDir() && d.Name()=="conventions"`.
  - Equivalence guarded by `TestSearch_RipgrepAndFallbackEquivalent` (`internal/store/search_test.go:79-97`); fallback forced via `FileStore.forceFallback` field (`internal/store/store.go:54-61`). Extend fixture `writeSearchFixture` (`:22-33`) with a `conventions/<f>.md` containing the needle, and assert no hit has a `conventions/` path-prefix in both paths.

- **Discovery step rewrite** (`templates/steps/plan/02-discovery.md`): replace the broad "search for anything about this area" pass (line 7) with (a) a full convention load via `{{config.command}} knowledge conventions` and (b) surface-targeted `knowledge search <surface>` calls keyed on the design surfaces the feature introduces.

- **`## Conventions` section in plan.md**: relevant-only, each with a one-line rationale, plus an explicit "none apply" state. Canonical section list lives ONLY in `templates/steps/plan/13-verification.md:62-73` (no Go enum); scaffold comments in `templates/scaffold/plan.md`. Authoring location (dedicated step vs folded) is the open design decision for the architecture step — it determines whether `EXPECTED_STEP_ORDER` changes.

- **Init scaffold (no migration)** (`internal/project/init.go`): add `filepath.Join(spektacularDir,"knowledge","conventions")` to the `dirs` slice (`:35-45`). The current flat seed writes `templates/.spektacular/conventions.md` → `.spektacular/knowledge/conventions.md` (`:85-92`); change it to write the embedded starter into `knowledge/conventions/` and stop writing the flat file. No migration of an existing flat `conventions.md` (user decision: project is fresh, breaking change accepted). Init always runs `force=true` (`cmd/init.go:32`) and `os.MkdirAll` is idempotent, so re-running init is safe.

- **E2E harbor extension** (`tests/harbor/plan-workflow/`): seed a distinctive JWT-relevant convention under `conventions/` via the `environment/Dockerfile` (mirror the spec COPY at lines 17-18); extend the hand-maintained oracles in `tests/test_plan_workflow.py` — add a conventions-read assertion for the discovery window (mirror `EXPECTED_SKILLS_PER_STEP` `:79-94` + `resolve_step_windows` `:359-394`), add `"conventions"` to `EXPECTED_PLAN_SECTIONS` (`:135-146`), update `EXPECTED_STEP_ORDER` (`:54-73`) only if a dedicated step is added, and add a content assertion that the section reflects the seeded convention. Oracles are hand-maintained (docstring `:1-32` forbids runtime derivation) — update in the same commit as the template/state-machine change.

## Files examined

- `internal/knowledge/set.go:25-33` — `Set` (sources slice), `Entry{Scope,Path}` (no content field — need a richer struct for conventions bodies).
- `internal/knowledge/set.go:77-87` — `Search` fan-out pattern to mirror.
- `internal/knowledge/set.go:90-96` — `Read(scope,path)` → `src.store.Read`; reuse to fetch convention bodies.
- `internal/knowledge/set.go:111-123` — `List` fan-out; the template for a new `Conventions()` method.
- `internal/knowledge/set.go:147-169` — `listFiles` recursive walk via `Store.List`; reuse with start dir `"conventions"`; note ErrNotFound propagation.
- `cmd/knowledge.go:45-49,78-81,191-201,244-252` — `sources` subcommand is the no-`--data` analogue to copy for `conventions`; `init()` registration.
- `internal/store/store.go:16-19` — `DirEntry{Name,IsDir}`. `:32-50` — `Store` interface. `:54-61` — `forceFallback` test field. `:124-141` — `FileStore.List` returns `ErrNotFound` for missing dir.
- `internal/store/search.go:31-41` — path selection (`forceFallback`, then `exec.LookPath("rg")`). `:65-104` — `searchRipgrep`, arg slice at `:66`. `:108-140` — `searchNative` walk, dir-skip at `:116-118`.
- `internal/store/search_test.go:22-33` — `writeSearchFixture`. `:79-97` — `TestSearch_RipgrepAndFallbackEquivalent`, forces fallback via `fbStore.forceFallback=true`.
- `internal/steps/plan/steps.go:27-49` — `Steps()` ordered list (18 steps), each `{Name,Src,Dst,Callback}`. `:63-75` — `writeStep` helper. `:78-94` — callback pattern. Adding/reordering a step touches this + the per-step `nextStep` chaining.
- `internal/steps/plan/steps_test.go` — `TestStepsOrderMatchesExpected` mirrors the step order; update if a step is added.
- `templates/steps/plan/02-discovery.md:7` — the broad knowledge-search line to replace; `:9` test-convention note.
- `templates/steps/plan/13-verification.md:62-73` — canonical plan.md section list (sole source of truth, no Go enum). `:75-93` — context.md/research.md section lists. `:19-29` — section→working-file mapping.
- `templates/scaffold/plan.md` — per-section comment blocks; add a Conventions block.
- `templates/templates.go:1-8` — `//go:embed all:*`; new template files need no registration.
- `internal/stepkit/stepkit.go:119-127` — `RenderTemplate` reads embedded FS by path.
- `internal/project/init.go:14-105` — init flow; `:35-45` dirs slice; `:85-92` conventions.md write (always overwrites); `:94-102` README seeding.
- `cmd/init.go:14-51` — always calls `project.Init(cwd,true)`; config.yaml written only if absent.
- `internal/config/config.go:34-36,118-131,219-238` — `DefaultKnowledgeLocation`, default project source, `WithDefaults`. No version/schema field — no existing migration hook.
- `templates/.spektacular/conventions.md` — current flat seed content (generic PEP8/testing placeholders).
- `tests/harbor/plan-workflow/environment/Dockerfile:17-18` — seeds `user-auth.md` spec; pattern for seeding a conventions file.
- `tests/harbor/plan-workflow/environment/user-auth.md` — JWT auth spec (the relevance anchor for the seeded convention).
- `tests/harbor/plan-workflow/tests/test_plan_workflow.py:1-32` (docstring — oracles hand-maintained), `:54-73` (`EXPECTED_STEP_ORDER`), `:79-94` (`EXPECTED_SKILLS_PER_STEP`), `:98` (`EXPECTED_SPAWN_STEPS`), `:135-146` (`EXPECTED_PLAN_SECTIONS`), `:204-225` (`parse_sections`), `:245-270` (`extract_tool_calls`), `:329-394` (`find_plan_cli_calls`, `resolve_step_windows`), `:719-734` (section assertions). `MIN_SECTION_LENGTH=100`.

## External references

- ripgrep `--glob` semantics — `!`-prefixed pattern excludes; gitignore-style matching. Why it mattered: confirms `--glob=!conventions/**` is the correct exclusion form and matches at any depth.
- Go `filepath.WalkDir` / `fs.SkipDir` — returning `filepath.SkipDir` for a directory entry skips its whole subtree. Why it mattered: the native-fallback exclusion mechanism that must mirror the ripgrep glob.

## Prior plans / specs consulted

- `go run . plan file list` / `spec file list` not separately mined — the knowledge base holds only architecture notes (bob/claude output specs, initial-idea) with nothing on this feature. `knowledge search conventions|planning` returned only generic architecture docs, no prior design.
- Spec `000025_convention-aware-planning.md` is the source of truth; `.spektacular/context.md` already holds the settled design decisions (two tiers, conventions/ dir, search exclusion, plan §Conventions, implement reads plan only).

## Open assumptions

- **`knowledge conventions` returns bodies, not just paths.** Spec says "returns every convention body, scope-tagged." If the team instead wants paths-only (with the agent reading each), STOP and confirm — it changes the command's output schema and the discovery instruction.
- **A scope with no `conventions/` directory yields zero conventions (not an error).** Required for fresh projects; assumed acceptable. If `Store.List` not-found should instead surface as an error anywhere, revisit.
- **No migration; init writes conventions only into `conventions/`.** Decided with the user: the project is fresh and accepts the breaking change, so init stops writing the flat `conventions.md` and seeds into the directory. This assumes no environment relies on a flat `conventions.md` still existing; since conventions live only under `conventions/`, the search exclusion (directory-targeted) and the full read by `knowledge conventions` stay consistent. If an external project ever needs migration, it is separate follow-up work.
- **The harbor verifier captures `knowledge conventions` calls** the same way it captures other Bash calls (`extract_tool_calls` records all Bash tool_use). Assumed; the conventions-read oracle is a Bash-substring check within the discovery window.

## Rehydration cues

- Re-read this file + `.spektacular/context.md` (settled decisions + user answers).
- Re-read spec: `go run . spec file read 000025_convention-aware-planning.md` (or it's under `.spektacular/specs/`).
- Knowledge subsystem: `internal/knowledge/set.go`, `cmd/knowledge.go`, `internal/store/store.go`, `internal/store/search.go` (+ `search_test.go`).
- Plan workflow: `internal/steps/plan/steps.go` (+ `steps_test.go`), `templates/steps/plan/02-discovery.md` and `13-verification.md`, `templates/scaffold/plan.md`.
- Init scaffold (no migration): `internal/project/init.go`, `cmd/init.go`, `internal/config/config.go`, `templates/.spektacular/conventions.md`.
- E2E: `tests/harbor/plan-workflow/` (`environment/Dockerfile`, `tests/test_plan_workflow.py`).
- Skill `go run . skill spawn-planning-agents` rebuilds the parallel-research approach.
