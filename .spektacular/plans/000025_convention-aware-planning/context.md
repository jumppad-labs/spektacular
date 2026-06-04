# Context: 000025_convention-aware-planning

## Current State Analysis

Knowledge today is a multi-source layer (`internal/knowledge/set.go`) over a `Store` abstraction (`internal/store/store.go`), with a single project scope configured by default (`internal/config/config.go:34-36,118-131,219-238`). The `Set` fans `Search`, `List`, `Read`, `Write` across scopes and scope-tags results (`set.go:77-123`); `Entry` carries only `{Scope, Path}` (`set.go:30-33`) — no body — and recursion lives in `listFiles` (`set.go:147-169`) atop the one-level `Store.List` (`store.go:124-141`, returns `store.ErrNotFound` for a missing dir).

Conventions currently live as a single flat file. Init writes the embedded `templates/.spektacular/conventions.md` to `.spektacular/knowledge/conventions.md` unconditionally (`internal/project/init.go:85-92`), creating `knowledge/{learnings,architecture,gotchas}` (`init.go:35-45`) but no `conventions/` directory. Init always runs with `force=true` (`cmd/init.go:32`); config.yaml is written only when absent (`init.go:69-74`). There is no version/schema field and no existing migration hook.

Search has two paths kept equivalent by test: ripgrep (`internal/store/search.go:65-104`, args at `:66`) and a native `filepath.WalkDir` fallback (`:108-140`, dir-skip at `:116-118`), selected at `:31-41` via `forceFallback` (test field, `store.go:54-61`) then `exec.LookPath("rg")`. Equivalence is asserted by `TestSearch_RipgrepAndFallbackEquivalent` (`search_test.go:79-97`) over `writeSearchFixture` (`:22-33`).

The plan workflow is an 18-step state machine (`internal/steps/plan/steps.go:27-49`, order mirrored by `steps_test.go`); steps render embedded templates (`templates/templates.go` `//go:embed all:*`, resolved by `internal/stepkit/stepkit.go:119-127`). The discovery step's broad knowledge-search instruction is `templates/steps/plan/02-discovery.md:7`. The canonical plan.md section list has no Go enum — it lives only in `templates/steps/plan/13-verification.md:62-73`, with the section→working-file mapping at `:19-29` and scaffold comments in `templates/scaffold/plan.md`. The harbor verifier `tests/harbor/plan-workflow/tests/test_plan_workflow.py` holds the independent, hand-maintained oracles (`EXPECTED_STEP_ORDER` `:54-73`, `EXPECTED_SKILLS_PER_STEP` `:79-94`, `EXPECTED_PLAN_SECTIONS` `:135-146`) and forbids runtime derivation (docstring `:1-32`).

## Per-Phase Technical Notes

### Phase 1.1: Convention reader and `knowledge conventions` command

**File changes:**
- `internal/knowledge/set.go` — add a `Convention` struct alongside `Entry` (near `set.go:30-33`) carrying `Scope`, `Path`, `Content` (`Entry` has no content field, so a new type is required). Add a `Conventions() ([]Convention, error)` method after `List()` (`set.go:123`), modelled on `List` (`set.go:111-123`): iterate `s.sources`, call `listFiles(src.store, "conventions")` (`set.go:147-169`), and for each file `src.store.Read(path)` the body. Absorb the missing-directory case: `listFiles` → `Store.List("conventions")` returns `store.ErrNotFound` when absent (`internal/store/store.go:124-141`); wrap the per-scope call so `errors.Is(err, store.ErrNotFound)` yields zero conventions for that scope instead of erroring (add `errors` import). Confirm the exact exported error name in `internal/store/store.go` before coding.
- `cmd/knowledge.go` — add `knowledgeConventionsCmd` cobra var following the argument-free `knowledgeSourcesCmd` (`cmd/knowledge.go:45-49`); add `knowledgeConventionsOutputSchema` with a single `conventions` array property mirroring `knowledgeSourcesOutputSchema` (`:78-81`); add `runKnowledgeConventions` modelled on `runKnowledgeSources` (`:191-201`) — schema-flag short-circuit, `newKnowledgeSet()`, call `set.Conventions()`, nil-guard to `[]knowledge.Convention{}`, emit `{"conventions": ...}` via `output.New(...).WriteResult`; register in `init()`'s `AddCommand` (`:251`).

**Complexity**: Medium
**Token estimate**: ~40k
**Agent strategy**: Single agent, sequential — the `Set` method and the CLI wiring are tightly coupled (the command depends on the method's signature). Add a focused unit test for `Conventions()` covering multi-scope concatenation, full-body return, and the missing-directory→empty case in the same pass.

### Phase 1.2: Exclude conventions from topic search

**File changes:**
- `internal/store/search.go` — ripgrep path: insert `"--glob=!conventions/**"` into the `exec.Command` args slice at `search.go:66`, between `"--ignore-case"` and `query`. Native fallback: in the `filepath.WalkDir` callback (`search.go:108-140`), before the existing `if d.IsDir()` skip at `:116-118`, add `if d.IsDir() && d.Name() == "conventions" { return filepath.SkipDir }`.
- `internal/store/search_test.go` — extend `writeSearchFixture` (`:22-33`) to also write a `conventions/<f>.md` file containing the search needle. In `TestSearch_RipgrepAndFallbackEquivalent` (`:79-97`) the existing `require.ElementsMatch` already proves both paths agree; add a single assertion (applied to the shared result set) that no returned hit's `Path` is within `conventions/`. Do not duplicate the assertion per path — the equivalence check already guarantees parity (avoid redundant assertions).

**Complexity**: Low
**Token estimate**: ~25k
**Agent strategy**: Single agent, sequential. Verify the `rg` glob form excludes at any depth and that the `t.TempDir`-based fixture writes through the production `FileStore.Write` path (it does — `:25-32`).

### Phase 1.3: Convention directory scaffolding

**File changes:**
- `internal/project/init.go` — add `filepath.Join(spektacularDir, "knowledge", "conventions")` to the `dirs` slice (`init.go:35-45`). Replace the current unconditional flat write (`init.go:85-92`, which reads embedded `.spektacular/conventions.md` and writes `knowledge/conventions.md`) with a write of the embedded starter into `knowledge/conventions/` (e.g. `conventions/conventions.md`). Stop writing the flat `knowledge/conventions.md`. No migration: there is no pre-existing flat file to preserve (project is fresh, breaking change accepted).
- `templates/.spektacular/conventions.md` — keep as the embedded starter body, now written into the directory as an individual starter file (change target path only; content can stay). Keep one starter file for minimal change.
- `internal/project/init_test.go` — add a test that renders init into a `t.TempDir` via the production `project.Init` path (never touching the repo's own `.spektacular`): assert the `conventions/` directory exists and contains the seeded starter, and assert init no longer writes a flat `conventions.md`.

**Complexity**: Low
**Token estimate**: ~20k
**Agent strategy**: Single agent, sequential. Init runs with `force=true` always (`cmd/init.go:32`) and `os.MkdirAll` is idempotent, so re-running init is safe; no migration ordering to reason about.

### Phase 2.1: Discovery loads conventions and targets topic searches

**File changes:**
- `templates/steps/plan/02-discovery.md` — replace the broad knowledge-search instruction at `02-discovery.md:7` ("Search the configured knowledge sources for anything already written about this area … `knowledge search <query>`") with two directives: (a) load all conventions in full via `{{config.command}} knowledge conventions`, and (b) run surface-targeted `{{config.command}} knowledge search <surface>` calls keyed on the design surfaces the feature introduces. Keep the existing test-convention guidance (`:9`) and the rest of the step intact. Use the `{{config.command}}` placeholder, never the rendered command.

**Complexity**: Low
**Token estimate**: ~20k
**Agent strategy**: Single agent, sequential. Pure template edit; no Go or state-machine change. Depends on Phase 1.1 (the command must exist).

### Phase 2.2: Plans carry a Conventions section

**File changes:**
- `templates/steps/plan/03-architecture.md` — add an instruction that, once the design shape is locked, the agent selects from the conventions loaded at discovery the subset relevant to the design, annotates each with a one-line rationale (and cites them inline where they drive a choice), and writes that content (with the explicit "none apply" fallback) to a new working file `.spektacular/work/{{plan_name}}/conventions.md` using its own `Write` tool. Relevance is proposed and user-confirmed, not auto-decided.
- `templates/steps/plan/13-verification.md` — add `## Conventions` to the canonical plan.md section list (`13-verification.md:62-73`) and add a `conventions.md` → `## Conventions` entry to the section→working-file mapping (`:19-29`). Decide placement (recommend directly after `## Overview`).
- `templates/scaffold/plan.md` — add a `## Conventions` comment block describing the section's purpose and the "none apply" state, positioned to match the verification list ordering.

**Complexity**: Medium
**Token estimate**: ~35k
**Agent strategy**: Single agent, sequential — the three template files must agree on section name, ordering, and the working-file name. No `internal/steps/plan/steps.go` or `steps_test.go` change (folded approach; no new step). Depends on Phase 2.1.

### Phase 3.1: End-to-end coverage of convention-aware planning

**File changes:**
- `tests/harbor/plan-workflow/environment/Dockerfile` — add a step (mirroring the spec COPY at `Dockerfile:17-18`) that creates `/app/.spektacular/knowledge/conventions/` and copies in a distinctive convention file relevant to the seeded `user-auth.md` JWT spec (e.g. an auth/middleware-structure rule). Add the convention source file under `environment/`.
- `tests/harbor/plan-workflow/tests/test_plan_workflow.py` — (a) add `"conventions"` to `EXPECTED_PLAN_SECTIONS` (`:135-146`); (b) add a new hand-maintained conventions-read oracle: assert that within the discovery step window (`resolve_step_windows` `:359-394`; window keyed on `plan goto` transitions `:329-356`) a Bash call's command contains `knowledge conventions` — model the detection on `_bash_command`/`find_plan_cli_calls` (`:329-356`) and the per-step-window parametrization used for `EXPECTED_SKILLS_PER_STEP` (`:79-94`); (c) add a content assertion that the parsed `conventions` section (`parse_sections` `:204-225`) contains a distinctive token from the seeded convention. Update the module docstring's oracle list if needed; do NOT derive any oracle from templates/state machine at runtime (docstring `:1-32`). `EXPECTED_STEP_ORDER` (`:54-73`) is unchanged because no step was added.

**Complexity**: Medium-High
**Token estimate**: ~45k
**Agent strategy**: Parallel analysis, sequential integration — one line of work on the Docker environment seed, one on the Python verifier oracles; integrate and run the harbor task once both are ready. Depends on Phases 2.1–2.2 (the behaviour under test). The seeded convention's distinctive token is the oracle's anchor — choose one unlikely to appear incidentally in a JWT plan.

## Testing Strategy

Per-phase testing detail:

- **Phase 1.1** — Go unit test for `Set.Conventions()`: multi-scope concatenation in configured order, full bodies returned with correct scope/path, and the missing-`conventions/`-directory → empty (no error) case. This is the heaviest unit coverage because it carries the spec's two subtle guarantees.
- **Phase 1.2** — extend the existing `TestSearch_RipgrepAndFallbackEquivalent` fixture with a `conventions/` file containing the needle; one assertion on the shared result set that no `conventions/`-prefixed hit is returned. No per-path duplicate assertion — the equivalence check already proves parity.
- **Phase 1.3** — Go init test rendered into `t.TempDir` via the production `project.Init`: conventions dir created and the starter seeded inside it; init no longer writes a flat `conventions.md`. No migration case (no migration performed).
- **Phases 2.1–2.2** — covered behaviourally by the end-to-end harbor task rather than unit tests, since these are template/instruction edits whose effect is only observable in a full plan run.
- **Phase 3.1** — the harbor verifier itself: hand-maintained oracles for conventions-read-in-discovery-window, `conventions` in `EXPECTED_PLAN_SECTIONS`, and a seeded-convention content assertion. Oracles are never derived from templates/state machine at runtime.

Cross-cutting conventions (project rules): tests own their filesystem (render via the production install path into `t.TempDir`, never walk the repo's real `.spektacular`); test oracles are independent and hand-maintained, never derived at runtime from the subject; no redundant assertions (one mechanism per bug class).

## Project References

- Knowledge layer: `internal/knowledge/set.go` (`Set`, `Entry`, `Search/List/Read/Write` fan-out, `listFiles`).
- Store abstraction & search: `internal/store/store.go` (`Store`, `FileStore`, `DirEntry`, `Hit`, `forceFallback`, `ErrNotFound`), `internal/store/search.go` (+ `search_test.go`).
- Knowledge CLI: `cmd/knowledge.go` (subcommand pattern, schemas, `init()` registration).
- Plan workflow: `internal/steps/plan/steps.go` (+ `steps_test.go`), `templates/steps/plan/02-discovery.md`, `03-architecture.md`, `13-verification.md`, `templates/scaffold/plan.md`, `templates/templates.go`, `internal/stepkit/stepkit.go`.
- Init/config: `internal/project/init.go` (+ `init_test.go`), `cmd/init.go`, `internal/config/config.go`, `templates/.spektacular/conventions.md`.
- E2E: `tests/harbor/plan-workflow/environment/Dockerfile`, `environment/user-auth.md`, `tests/test_plan_workflow.py`.
- Rehydration skill: `go run . skill spawn-planning-agents`.

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|-------------|----------------|
| Low | ~10k | Single agent, sequential |
| Medium | ~25k | 2-3 parallel agents |
| High | ~50k+ | Parallel analysis, sequential integration |

Phase mapping: 1.2 (~25k), 1.3 (~20k) and 2.1 (~20k) are Low/sequential; 1.1 (~40k) and 2.2 (~35k) are Medium/sequential (each touches tightly-coupled files that must agree); 3.1 (~45k) is the only one warranting parallel analysis (Docker seed vs. Python oracles) with sequential integration and a single harbor run.

## Migration Notes

None. By explicit decision (the project is still fresh and accepts breaking changes), no migration of an existing flat `conventions.md` is performed — this waives the spec's "existing projects must keep working" constraint for this project. Init simply stops writing the flat `.spektacular/knowledge/conventions.md` and seeds the starter into `knowledge/conventions/` instead (`init.go:85-92`). Because conventions live only under the `conventions/` directory, the `knowledge search` exclusion (which targets that directory) and the full read by `knowledge conventions` are consistent with no stray flat file existing. If migration is ever needed for an external project, it would be separate follow-up work.

## Performance Considerations

Negligible. `knowledge conventions` adds one fan-out read of typically a handful of small files per scope at planning time — bounded by the number of conventions, not the size of the reference knowledge base. The search exclusion is strictly a reduction in work (fewer files walked/matched). The spec's intent that "planning does not get slower or noisier as the topic-specific knowledge base grows" is served by Tier-2 searches being surface-targeted rather than broad — fewer, more relevant queries regardless of base size.
