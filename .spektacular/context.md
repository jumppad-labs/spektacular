# Working Context — implement 000025_convention-aware-planning

> This file is the implement-workflow working context. It replaces the
> earlier plan-workflow notes (preserved in the plan store's context.md).

## Validation gate (read_plan) — PASSED

- Structural: all 10 plan.md sections present; 6 phases (1.1, 1.2, 1.3, 2.1, 2.2, 3.1),
  each with a resolving `*Technical detail:*` link into the plan store context.md.
- Drift: no mismatches. Every file/symbol/command/template named in the plan exists.
  Minor line-number drift only (e.g. harbor `find_plan_cli_calls` at :339 not :329,
  `_bash_command` at :325) — within the plan's stated ranges; map on the fly.
- Changelog mode: **first-phase invocation** (no `## Changelog` section yet).
  `update_changelog` will create it on first use. Analyze starts at Phase 1.1.

## Phase order (dependencies)

1.1 (conventions reader + `knowledge conventions` cmd) → 1.2 (search exclusion) →
1.3 (init scaffold) → 2.1 (discovery template; depends on 1.1) →
2.2 (plan `## Conventions` section; depends on 2.1) →
3.1 (harbor e2e; depends on 2.1–2.2).

## Key implementation decisions carried from the plan

- **NO MIGRATION** (final decision — corrects the stale plan-workflow note that said
  "migrate on init only"). Project is fresh; breaking change accepted. Init simply
  STOPS writing the flat `.spektacular/knowledge/conventions.md` and seeds the starter
  into `knowledge/conventions/` instead. No move-and-guard logic.
- **No new state-machine step.** Conventions folded into existing steps:
  discovery loads conventions + surface-targeted searches; architecture selects the
  relevant subset into a `conventions.md` working file; verification assembles the
  `## Conventions` plan section. ⇒ `steps.go` / `steps_test.go` UNTOUCHED;
  harbor `EXPECTED_STEP_ORDER` unchanged.
- `knowledge conventions` returns **bodies** (`{scope,path,content}`) — new `Convention`
  struct (Entry has no content field) + `Set.Conventions()` modelled on `List`.
- A scope with no `conventions/` dir ⇒ zero conventions, NOT an error
  (`Store.List` returns `ErrNotFound`; absorb with `errors.Is`).
- Search exclusion in BOTH paths, kept equivalent by the existing equivalence test:
  ripgrep `--glob=!conventions/**` arg; native walk `filepath.SkipDir` on `conventions` dir.

## Cross-cutting project conventions (apply during impl)

- Tests own their filesystem: render via production install path into `t.TempDir`,
  never walk the repo's real `.spektacular`.
- Test oracles are independent + hand-maintained, never derived at runtime from the subject.
- No redundant assertions: one mechanism per bug class (e.g. 1.2 adds a single
  shared-result-set assertion, not a per-path duplicate).

## Phase 1.1 analysis (confirmed shapes)

- `Set.Conventions()` mirrors `List()` (set.go:111-123): iterate `s.sources`,
  `listFiles(src.store, "conventions")`, then `src.store.Read(f)` each body.
  Absorb `store.ErrNotFound` from `listFiles` via `errors.Is` ⇒ zero for that scope.
  New `Convention{Scope,Path,Content}` struct near `Entry` (set.go:30-33). Add `errors` import.
- CLI: `knowledgeConventionsCmd` (Use "conventions", no Args, no --data) like
  `knowledgeSourcesCmd` (cmd/knowledge.go:45-49); `knowledgeConventionsOutputSchema`
  = `{conventions: array}` like sources schema (:78-81); `runKnowledgeConventions`
  like `runKnowledgeSources` (:191-201) — schema short-circuit, nil-guard to
  `[]knowledge.Convention{}`, emit `{"conventions": ...}`; register in AddCommand (:251).
- Tests: `set_test.go` uses `writeFile(t,dir,name,content)` + `twoScopeSet(t)` +
  testify `require`. Add `TestSet_Conventions...` covering multi-scope concat in
  configured order, full bodies, and missing-`conventions/`-dir → empty (no error).

## Progress

- **ALL PHASES COMPLETE & VERIFIED.** Live `make harbor-test-plan` PASSES (user confirmed) — the
  full end-to-end convention-aware planning behaviour is proven against a real agent. Implement
  workflow reached `finished`; repo CHANGELOG.md entry written. Nothing committed yet.
- Harbor auth gotcha (fixed in Makefile): the Claude.ai OAuth `accessToken` must be passed as
  `CLAUDE_CODE_OAUTH_TOKEN`, NOT `ANTHROPIC_AUTH_TOKEN`. Harbor's claude_code agent maps
  `ANTHROPIC_AUTH_TOKEN → ANTHROPIC_API_KEY` (claude_code.py:943-947), and an OAuth token is not a
  valid API key → 401 authentication_failed before the agent runs any step.
- **Phase 3.1 — CODE COMPLETE, static checks green; live harbor run is the open verification.**
  - Seeded convention `environment/auth-audit-logging.md` (distinctive token `AUTH_AUDIT_V2`,
    clearly JWT/auth-relevant). Dockerfile mkdir+COPY into knowledge/conventions/ (named ≠ init
    starter `conventions.md`, so init's idempotent re-create won't clobber it).
  - Python oracles (test_plan_workflow.py): `EXPECTED_PLAN_SECTIONS += "conventions"`;
    constants `CONVENTIONS_READ_COMMAND`/`SEEDED_CONVENTION_TOKEN`; new
    `TestConventionAwarePlanning` (discovery-window `knowledge conventions` read + section content
    assertion); docstring oracle list updated. 84 tests collect; py_compile OK.
  - DEVIATION (beyond plan's literal file list): also added a discovery hint to
    `instruction.md` (`spektacular knowledge conventions`) mirroring the existing skill hints, to
    reduce the open-question flakiness risk. Reliability aid, not the oracle.
  - Static verify: go build/test/lint pass; pytest --collect-only OK.
  - USER will run `make harbor-test-plan` themselves (their decision; their API creds).
  - EXTRA (user-requested) fail-fast: added session-scoped autouse fixture `_abort_on_failed_agent_run`
    in test_plan_workflow.py — on the 3 fatal preflight conditions (no transcript / auth failure /
    agent didn't finish) it calls `pytest.exit(..., returncode=1)` so the run reports ONE clear
    reason instead of ~80 misleading cascade failures. Genuine workflow regressions still run the
    full suite. Verified locally: aborts fast, pytest exit code 1 (→ reward 0). Works in pytest 8.4.1
    (Docker) and 9.x (local).
- **Phase 2.2 — DONE & checked off + changelogged.** 3 template files agree on `## Conventions`
  (after Overview), working file `conventions.md`. Phases 1.1–2.2 ALL complete.
- **FINAL: Phase 3.1 — End-to-end harbor coverage** (Medium-High). Touches:
  1. `tests/harbor/plan-workflow/environment/Dockerfile` — add a step (mirror spec COPY at
     Dockerfile:17-18) creating `/app/.spektacular/knowledge/conventions/` and copying in a
     distinctive auth/JWT-relevant convention file. Add the convention source under `environment/`.
     Choose a DISTINCTIVE anchor token unlikely to appear incidentally in a JWT plan.
  2. `tests/harbor/plan-workflow/tests/test_plan_workflow.py` — (a) add `"conventions"` to
     `EXPECTED_PLAN_SECTIONS`; (b) add a hand-maintained conventions-read oracle: within the
     discovery step window (resolve_step_windows; window keyed on `plan goto` transitions) assert a
     Bash call's command contains `knowledge conventions` (model on `_bash_command`/
     `find_plan_cli_calls` + the per-step-window parametrization used for `EXPECTED_SKILLS_PER_STEP`);
     (c) content assertion that the parsed `conventions` section (parse_sections) contains the
     seeded convention's distinctive token. Do NOT derive oracles from templates/state machine at
     runtime (docstring forbids). `EXPECTED_STEP_ORDER` UNCHANGED (no step added).
  - Open question (from plan): a live harbor run may be flaky about surfacing the seeded convention
    into the section. If flaky: make the seeded convention more clearly JWT-relevant + more
    distinctive token; if still failing after instructions are correct, STOP and ask user before
    weakening the assertion.
  - Strategy: parallel analysis (Docker seed vs Python oracles), sequential integration, single
    harbor run. NOTE: running the actual harbor task may be slow/require Docker — check availability.
- **Phase 2.1 — DONE & checked off + changelogged** (discovery template rewritten).
- **Next: Phase 2.2 — Plans carry a Conventions section** (Medium; 3 template files must AGREE on
  section name `## Conventions`, placement = directly after `## Overview`, and working-file name
  `conventions.md`):
  1. `templates/steps/plan/03-architecture.md` — add instruction: once design shape is locked,
     select from the conventions loaded at discovery the subset relevant to the design, annotate
     each with a one-line rationale (cite inline where they drive a choice), write to
     `.spektacular/work/{{plan_name}}/conventions.md` via own `Write` tool, with explicit
     "none apply" fallback. Relevance is PROPOSED and user-confirmed, not auto-decided. Model the
     write-to-working-file wording on the existing block at end of 03-architecture.md (architecture.md).
  2. `templates/steps/plan/13-verification.md` — add `## Conventions` to canonical plan.md section
     list (after `## Overview`, before `## Architecture...`) at the `**plan.md — required ## sections**`
     list (~line 64-73); add `conventions.md` → `## Conventions` to the section→working-file mapping
     (~line 19-29).
  3. `templates/scaffold/plan.md` — add a `## Conventions` comment block after `## Overview`,
     matching the verification list ordering.
  No steps.go / steps_test.go change. EXPECTED_STEP_ORDER unchanged.
- **Milestone 1 COMPLETE** (Phases 1.1, 1.2, 1.3 all done, checked off, changelogged). Plumbing
  in place: `knowledge conventions` command, search exclusion, init scaffolds `conventions/` dir.
- **Next: Phase 2.1 — Discovery loads conventions and targets topic searches** (Low; pure template
  edit, no Go/state-machine change). Edit `templates/steps/plan/02-discovery.md:7` — replace the
  broad "search for anything about this area … `knowledge search <query>`" instruction with two
  directives: (a) load ALL conventions in full via `{{config.command}} knowledge conventions`, and
  (b) run surface-targeted `{{config.command}} knowledge search <surface>` calls keyed on the design
  surfaces the feature introduces. Keep the existing test-convention guidance and the rest of the
  step intact. Use the `{{config.command}}` placeholder, never the rendered command.
- **Phase 1.2 — DONE & checked off + changelogged** (glob deviation `--glob=!**/conventions/**`).
- **Phase 1.3 test conflict to handle:** existing `TestInit_CreatesConventionsMd`
  (init_test.go:58-66) asserts the FLAT `knowledge/conventions.md` exists. Since the flat
  write is being removed, that test must be UPDATED (not just added to) to assert the new
  `knowledge/conventions/conventions.md` location AND that no flat `conventions.md` exists.
- Destination chosen for starter: `knowledge/conventions/conventions.md` (embedded source path
  stays `.spektacular/conventions.md`).
- **Next: Phase 1.3 — Convention directory scaffolding** (Low). In `internal/project/init.go`:
  add `filepath.Join(spektacularDir, "knowledge", "conventions")` to the `dirs` slice (init.go:35-45);
  replace the flat write at init.go:85-91 (reads embedded `.spektacular/conventions.md`, writes
  `knowledge/conventions.md`) with a write of the starter into `knowledge/conventions/` (e.g.
  `conventions/conventions.md`); STOP writing the flat `knowledge/conventions.md`. No migration.
  Keep `templates/.spektacular/conventions.md` as the embedded starter body (content stays, target
  path changes). Test (init_test.go): render via production `project.Init` into `t.TempDir`; assert
  `conventions/` dir exists with the starter, and that no flat `conventions.md` is written.
- **Phase 1.1 — DONE & checked off + changelogged.** `Convention` struct + `Set.Conventions()`
  in set.go; `knowledge conventions` CLI in cmd/knowledge.go; 4 unit tests in set_test.go.
  Verified all green (build/test/vet). Changelog section created in plan.md.
- **Next: Phase 1.2 — Exclude conventions from topic search.** (Low complexity, single agent.)
  - ripgrep: insert `"--glob=!conventions/**"` into args at search.go:66, between
    `"--ignore-case"` and `query`.
  - native fallback: in WalkDir callback, before the `d.IsDir()` skip (search.go:116-118),
    add `if d.IsDir() && d.Name() == "conventions" { return filepath.SkipDir }`.
  - test: extend `writeSearchFixture` (search_test.go:22-33) to also write a
    `conventions/<f>.md` containing the needle; in `TestSearch_RipgrepAndFallbackEquivalent`
    (:79-97) add ONE assertion on the shared result set that no hit's Path is under
    `conventions/`. NO per-path duplicate (equivalence check already proves parity).
- Autonomous mode: per user feedback, loop phases without confirmation prompts; only stop
  for real design decisions or verification failures.

## Phase 1.2 — DONE (code + tests, all green)

- **DEVIATION from plan glob.** Plan/context specified `--glob=!conventions/**`, but verified
  empirically that ripgrep does NOT exclude with that form (it leaks `conventions/*`). The
  working form is **`--glob=!**​/conventions/**`** — excludes a `conventions` dir at any depth
  AND its full subtree, matching the native fallback's `d.Name()=="conventions"` SkipDir.
  The equivalence test (`TestSearch_RipgrepAndFallbackEquivalent`) caught the mismatch — it's
  doing its job. Implemented in search.go searchRipgrep.
- native fallback: SkipDir on dir named `conventions` (searchNative, inside the `d.IsDir()` block).
- tests: extended `writeSearchFixture` with `conventions/style.md` (contains "needle"); added one
  exclusion assertion to the equivalence test on `rgHits` only (parity already proven). All
  `TestSearch*` pass.

## Substitutions / answers given to the user

- Plan selected: 000025_convention-aware-planning (only active/untracked plan; user confirmed).
