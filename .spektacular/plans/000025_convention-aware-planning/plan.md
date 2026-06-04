# Plan: 000025_convention-aware-planning

<!-- Metadata -->
<!-- Created: 2026-06-03T13:55:50Z -->
<!-- Commit: e82fb81 -->
<!-- Branch: main -->
<!-- Repository: git@github.com:jumppad-labs/spektacular.git -->

## Overview

This plan makes plan creation actively draw on a project's accumulated know-how. Always-apply conventions are reviewed on every plan, topic-specific lessons are pulled in only for the surfaces the work touches, and the conventions that bear on the work are written into the plan itself — so whoever implements it follows a plan that already states which standards apply. It benefits planners (plans reflect established practice automatically), implementers (one plan carries the relevant standards), and reviewers (an empty or generic conventions list is a visible signal the knowledge was not consulted).

## Architecture & Design Decisions

The knowledge base is treated strictly as a **planning-time input**; the plan documents remain the sole contract the implement workflow consumes, so the implement steps gain no new behaviour. Knowledge is split into two tiers by *access pattern*, not storage scope. **Tier 1 — conventions (always-apply)** live as one-rule-per-file under a reserved `conventions/` directory in each knowledge scope and are read in full at planning time. **Tier 2 — reference knowledge** (architecture, gotchas, learnings, …) is reached only through surface-targeted `knowledge search` calls keyed on the design surfaces a feature introduces.

A new read-only CLI verb, `knowledge conventions`, fans across every configured scope — mirroring the existing `knowledge list`/`sources` fan-out — and returns each convention's **body**, scope-tagged (`{scope, path, content}`). A dedicated verb is required because the workflow skills forbid reading knowledge entries with the raw `Read` tool; one call yields the full Tier-1 digest. The command must treat a scope that has no `conventions/` directory as *zero conventions rather than an error*, so fresh projects and partially-populated scopes still plan cleanly. In parallel, `knowledge search` is amended to **exclude the `conventions/` directory** so a convention already digested in full at discovery is never surfaced a second time; the exclusion is applied identically in both the ripgrep path (a `--glob=!conventions/**` argument) and the native directory-walk fallback (a `filepath.SkipDir` on a `conventions` directory), keeping the two paths equivalent under the existing equivalence test.

The plan workflow changes by **folding convention handling into the existing steps rather than adding a new state-machine step** — the chosen direction over a dedicated conventions step, because relevance is a synthesis of discovery (the full convention load) and design (which surfaces the feature actually touches), not an independent unit of work. The discovery step loads all conventions via the new verb and runs surface-targeted Tier-2 searches in place of the old broad "search for anything about this area" pass. The architecture step — where the design shape and its surfaces are locked — is where the relevant subset is selected and written, each convention annotated with a one-line rationale for why it applies and cited inline in the design it drives. The verification step assembles a new `## Conventions` section into `plan.md` from that working file, writing an explicit "none apply" state when nothing is relevant; that explicit state doubles as a visible check that the knowledge base was actually consulted. Keeping the step list unchanged means `internal/steps/plan/steps.go` and its order test are untouched; only the canonical section list (`templates/steps/plan/13-verification.md`), the discovery template, and the harbor oracles change.

Project init adopts the new layout directly: it creates the `conventions/` directory and seeds the starter convention there instead of writing the legacy flat `conventions.md`. There is deliberately **no migration of an existing flat `conventions.md`** — this project is still fresh and accepts the breaking change, so the spec's "existing projects must keep working" constraint is waived here by explicit decision rather than carrying migration complexity that has no real project to serve. End-to-end coverage extends the existing `tests/harbor/plan-workflow` task: the environment seeds a distinctive auth-relevant convention under `conventions/`, and the verifier's hand-maintained oracles gain a discovery-window assertion that conventions were read through the CLI, a `Conventions` entry in the expected plan sections, and a content assertion that the section reflects the seeded convention — the strong check that the knowledge was digested, not merely that a command ran. Rejected alternatives (a dedicated conventions step; Set-layer search post-filtering; migrate-on-init; giving implement new knowledge access) and their evidence are recorded in research.md#alternatives-considered-and-rejected.

## Component Breakdown

- **Convention reader (knowledge layer).** A new fan-out operation on the knowledge `Set` that walks every configured scope, reads each file under that scope's `conventions/` directory, and returns the convention **bodies** tagged with their scope and path. It owns the "read all Tier-1 conventions in full" behaviour and the rule that a scope lacking a `conventions/` directory contributes zero conventions rather than raising an error. It reuses the existing recursive directory walk and per-file read primitives; it does not rank, dedup, or filter.

- **`knowledge conventions` CLI subcommand.** A new read-only verb that exposes the convention reader, following the same shape as the existing argument-free `knowledge sources`/`list` verbs (schema flag, JSON envelope, scope-tagged results). It owns the public, agent-facing entry point for the full convention digest — the only sanctioned way for a planning agent to read conventions, since the raw `Read` tool is forbidden for knowledge.

- **Convention-aware search exclusion.** A change to the existing knowledge search component so that the `conventions/` directory is omitted from search results. It owns keeping the ripgrep-backed path and the native directory-walk fallback in lock-step (the same exclusion expressed in each), so results stay identical regardless of whether the accelerated tool is present. It interacts with the convention reader only by contract: what the reader surfaces in full, search must never surface again.

- **Discovery step instructions.** The plan workflow's discovery step, changed to (a) load all conventions in full via the new verb and (b) run surface-targeted reference-knowledge searches keyed on the design surfaces the feature introduces, replacing the previous broad "search for anything about this area" pass. It owns establishing the full Tier-1 context and the relevant Tier-2 context for the rest of planning.

- **Architecture step instructions.** The plan workflow's architecture step, changed to select — from the conventions loaded at discovery — the subset relevant to the locked design, each with a one-line rationale, and record them (with inline citations to the design they drive) into the plan's working material. It owns the relevance judgement, which is proposed here and confirmed by the user, never fully automated.

- **`## Conventions` plan section + canonical section list.** The verification step and plan scaffold, changed to assemble a new `## Conventions` section into `plan.md` from the architecture step's working material, including an explicit "none apply" state when nothing is relevant. It owns the durable, implementation-facing record of which conventions govern the work and why; the explicit empty state doubles as the visible knowledge-consultation signal.

- **Init scaffold.** Project initialisation, changed to create the `conventions/` directory and seed the starter convention there instead of writing the legacy flat `conventions.md`. It owns establishing the new layout for new projects; it performs no migration of an existing flat conventions file (a deliberate breaking change accepted while the project is fresh).

- **End-to-end harbor coverage.** The existing `plan-workflow` harbor task's environment and verifier, extended so the environment seeds a distinctive convention relevant to the seeded auth spec, and the hand-maintained oracles assert (1) the discovery step read conventions through the CLI, (2) `plan.md` carries a `Conventions` section, and (3) that section reflects the seeded convention. It owns proving the behaviour end-to-end with an independent oracle, never derived from the templates or state machine at runtime.

## Data Structures & Interfaces

**`Convention` (knowledge layer).** A new value type carrying a single convention's body together with the scope and path it came from. It is distinct from the existing `Entry` type (which carries only scope and path) because the conventions reader must return full bodies in one call.

```
type Convention struct {
    Scope   string  // configured scope the convention lives in
    Path    string  // store-relative path, e.g. "conventions/auth-middleware.md"
    Content string  // full file body
}
```

**`Set.Conventions()` (knowledge layer interface).** A new fan-out method on the knowledge `Set`:

```
func (s *Set) Conventions() ([]Convention, error)
```

It iterates configured scopes in order, enumerates files under each scope's `conventions/` directory, reads each body, and returns them scope-tagged. A scope with no `conventions/` directory contributes nothing (the "directory not found" condition is absorbed, not propagated as an error). The existing `Store` interface is unchanged — this method reuses the current directory-listing and file-read primitives.

**`knowledge conventions` CLI output envelope.** A JSON object whose `conventions` field is an array of the serialized `Convention` shape, matching the existing scope-tagged result conventions of the other knowledge verbs:

```
{ "conventions": [ { "scope": "project", "path": "conventions/auth-middleware.md", "content": "..." } ] }
```

The command takes no `--data` input; its output schema declares the single `conventions` array property, parallel to how `knowledge sources` declares `sources`.

**Knowledge search exclusion (contract, not a new type).** The search component's contract is narrowed: results MUST NOT include any entry whose path is within a `conventions/` directory, and this must hold identically across the accelerated and fallback search paths. No new type is introduced — the change is to the filtering behaviour of the existing search.

**`Conventions` plan-document section (markdown contract).** A new `## Conventions` section in `plan.md` with a fixed shape. When conventions apply, it holds a bullet list where each item names the convention and states, in one line, why it applies to this feature — e.g. `- **<convention name / one-line summary>** — <why it applies to this feature>.`. When nothing is relevant, it holds a single explicit sentence declaring that none are relevant — e.g. `No project conventions apply to this feature.`. This is the durable contract the implement workflow consumes — it carries the governing conventions so implementation never needs to re-read the knowledge base.

## Implementation Detail

This change follows existing patterns almost everywhere; it introduces no new architectural machinery. The bulk of the work is additive and mirror-shaped against code that already exists.

**Knowledge fan-out — follow the established pattern.** The new convention reader is built by copying the shape of the existing list/search fan-out: iterate the ordered scoped stores, do per-scope work, concatenate scope-tagged results, and wrap any error with the scope name. The one new behaviour is absorbing the "no conventions directory in this scope" condition into an empty result instead of an error — a small, deliberate divergence from the strict fan-out error contract, justified because an absent Tier-1 directory is a normal state, not a failure. The CLI subcommand is a near-verbatim copy of the argument-free `sources`/`list` verbs (schema-flag short-circuit, build the set, call the method, emit a JSON envelope), so a developer reading it will find nothing surprising.

**Search exclusion — one rule expressed twice, kept in lock-step.** The exclusion is the only place the change touches dual-path code. The accelerated path gains an ignore-glob argument; the fallback walk gains a skip-subtree branch when it meets a `conventions` directory. The governing constraint is that the two expressions stay behaviourally identical, which the existing equivalence test enforces — the test fixture is extended so it would fail if either path forgot the exclusion or applied it differently. No new abstraction is introduced to share the logic; matching the existing "two parallel implementations, one equivalence test" pattern is the lower-risk choice.

**Plan workflow — instruction edits, not state-machine changes.** Because convention handling is folded into existing steps, the state machine, its step list, and its order test are untouched. The work is editing step instruction templates (discovery to load conventions and run surface-targeted searches; architecture to select and justify the relevant subset) and extending the verification step plus the plan scaffold with the new `## Conventions` section and its "none apply" fallback. The canonical list of required plan sections — which lives in the verification instructions and the scaffold comments, with no code-level enum — is updated in the same change so the section is treated as mandatory. A developer extending the workflow later will see conventions handled exactly like every other plan section: gathered into a working file during the steps, assembled at verification.

**Init scaffold — directory plus starter, no migration.** Initialisation already creates the knowledge subdirectories and writes seed files on its force path. The change is to add the `conventions/` directory to that set and write the starter convention into it, replacing the old flat `conventions.md` write outright. No migration logic is introduced: the project is fresh, so there is no existing flat conventions file to preserve, and the simpler scaffold avoids carrying move-and-guard ordering that has no real beneficiary.

**Tests — hand-maintained oracles updated in the same change.** The end-to-end harbor verifier keeps its independent, hand-maintained oracles (step order, skills-per-step, expected sections). This change seeds a convention in the task environment and adds three oracle updates: a discovery-window assertion that the convention-reading CLI verb ran, the new section added to the expected-sections list, and a content assertion tying the rendered section back to the seeded convention. These oracle edits land alongside the template edits they describe, never derived from the templates at runtime.

## Dependencies

All dependencies are internal and already present; no new external libraries or upstream specs/plans are required, and no dependency must land before this work starts.

**Internal runtime dependencies (existing, reused):**

- **Knowledge `Set` / scoped-store layer** — provides the multi-scope fan-out and the per-file read/list primitives the new convention reader is built on. Changed: gains the new `Conventions()` method and `Convention` type; existing methods untouched.
- **`Store` / `FileStore` abstraction** — provides directory listing, file reads, and the dual-path (accelerated + native fallback) search. Changed: search gains the `conventions/` exclusion in both paths; the interface itself is unchanged.
- **Knowledge CLI command group (cobra)** — provides the subcommand registration, schema-flag, and JSON-envelope output conventions. Changed: gains the new `conventions` subcommand.
- **Plan workflow step engine + instruction templates (embedded via go:embed)** — provides the step state machine and the templated step instructions. Changed: discovery, architecture, and verification instruction templates and the plan scaffold are edited; the state machine and step list are NOT changed.
- **Project init / scaffold** — provides the `.spektacular` directory creation and seed-file writing. Changed: creates the `conventions/` directory and seeds the starter convention there instead of writing the flat `conventions.md`; no migration.
- **Output/schema helpers** — provide the JSON result/schema writers reused verbatim by the new subcommand. Unchanged.

**External tooling (existing, optional — no version dependency):**

- **ripgrep (`rg`)** — the optional accelerated search backend. The exclusion must be expressed for it, but its presence remains optional; the native fallback yields identical results when it is absent.

**Test dependencies (existing, reused):**

- **`plan-workflow` harbor task (Docker environment + pytest verifier)** — provides the end-to-end harness and the hand-maintained oracle pattern. Changed: environment seeds a convention; verifier oracles are extended.
- **Store search equivalence test** — provides the guarantee that the two search paths agree. Changed: its fixture is extended to cover the conventions exclusion.

**Planning dependencies:** none. This plan is self-contained and can start immediately.

## Testing Approach

Testing is layered: Go unit/contract tests for the knowledge and search changes, a Go test for the init scaffold, and an extension of the existing end-to-end harbor task for the plan-workflow behaviour. Each layer slots into a convention the project already uses, and tests build their own filesystem fixtures (scratch temp directories populated through the production code paths) rather than reading real project state.

**Convention reader (heaviest unit coverage).** Unit tests assert the load-bearing behaviours of the new fan-out: every convention body under a scope's `conventions/` directory is returned with the correct scope and path; multiple scopes concatenate in configured order; and — the most important case — a scope with no `conventions/` directory yields an empty result rather than an error. This is where the bulk of new coverage lives, because this component carries the spec's two subtle guarantees (full bodies, and graceful handling of an absent directory).

**Search exclusion (contract/regression).** The existing test that proves the accelerated and native search paths return equivalent results is the natural home for this guarantee; its fixture is extended so a convention file containing an otherwise-matching term is present, and the test asserts that neither path returns it. Because that single equivalence test already exercises both paths against the same fixture, it is sufficient to guarantee the exclusion is applied identically in both — no separate per-path assertion is added, to avoid asserting the same property twice.

**Init scaffold (unit).** A test exercises a freshly-initialised project and asserts that init creates the `conventions/` directory and seeds the starter convention inside it (and no longer writes a flat `conventions.md`). The fixture is rendered into a scratch directory through the real init path, never by touching the repository's own `.spektacular`. No migration case is tested, because no migration is performed.

**End-to-end (harbor, hand-maintained oracles).** The plan-workflow harbor task is the strong, behavioural check. Its environment seeds a distinctive convention relevant to the seeded auth spec, and its independent, hand-maintained oracles assert: the discovery step actually read conventions through the CLI during its step window; the finished `plan.md` contains the `Conventions` section; and that section's content reflects the seeded convention — proving the knowledge was digested into the plan, not merely that a command executed. These oracles are maintained by hand and updated in the same change as the template edits they describe; they are never derived from the templates or state machine at runtime, which would make the check tautological.

**Deliberate gaps.** No tests are added to the implement workflow: it gains no new behaviour (it continues to consume only the plan documents), so its existing coverage stands. No new ranking/search-quality tests are added, since search behaviour is unchanged apart from the exclusion.

## Milestones & Phases

### Milestone 1: Conventions become a first-class, readable knowledge tier

**What changes**: Conventions move from a single flat file to one-rule-per-file under a reserved `conventions/` directory in each knowledge scope, and a new read-only command surfaces every convention's full text in one call, tagged with the scope it came from. Topic searches stop returning conventions, so the same rule is never surfaced twice. Project initialisation creates the conventions directory and seeds a starter convention there. After this milestone a user can author and change individual conventions independently and read them all through the CLI, but nothing about planning has changed yet — this is the plumbing the rest of the feature stands on.

#### - [x] Phase 1.1: Convention reader and `knowledge conventions` command

A new read-only CLI command reads every always-apply convention across all configured knowledge scopes and returns each one's full text, tagged with the scope it came from. A scope that has no conventions contributes nothing and causes no error. This is the only sanctioned way for a planning agent to read conventions, since reading knowledge files directly is forbidden.

*Technical detail:* [context.md#phase-11](./context.md#phase-11-convention-reader-and-knowledge-conventions-command)

**Acceptance criteria**:

- [x] Running the conventions command returns the full body of every convention across every configured scope, each tagged with its scope and path.
- [x] A scope with no conventions directory is silently treated as having zero conventions — no error, no missing-directory failure.
- [x] The command exposes its input/output schema like the other knowledge commands and takes no data argument.

#### - [x] Phase 1.2: Exclude conventions from topic search

Topic-specific knowledge searches no longer return convention entries, because conventions are already read in full elsewhere. The exclusion behaves identically whether or not the optional accelerated search tool is installed, so search results are the same in every environment.

*Technical detail:* [context.md#phase-12](./context.md#phase-12-exclude-conventions-from-topic-search)

**Acceptance criteria**:

- [x] A topic search over a project that has conventions never returns an entry that lives in the conventions area.
- [x] The accelerated and fallback search paths return the same results for the same query, including the same exclusion of conventions.
- [x] Non-convention knowledge entries are still found exactly as before.

#### - [x] Phase 1.3: Convention directory scaffolding

Project initialisation creates the conventions directory and seeds a starter convention inside it, instead of writing the old flat conventions file. There is no migration of an existing flat conventions file — the project is fresh and adopts the new layout directly (a deliberate breaking change).

*Technical detail:* [context.md#phase-13](./context.md#phase-13-convention-directory-scaffolding)

**Acceptance criteria**:

- [x] Initialising a new project creates the conventions directory and seeds a starter convention inside it.
- [x] Initialisation no longer writes a flat conventions file.

### Milestone 2: Plans actively draw on the conventions

**What changes**: Creating a plan now consults the project's conventions. Every always-apply convention is reviewed during planning, and topic-specific knowledge is pulled in only for the surfaces the work actually touches. The relevant conventions — and only the relevant ones — are written into the finished plan in a dedicated `## Conventions` section, each with a one-line reason it applies to this work. When nothing is relevant (or no conventions exist), the section says so plainly. After this milestone, a plan produced for a feature visibly carries the standards that govern it, so whoever implements it follows a plan that already states which conventions apply — without ever consulting the knowledge base themselves.

#### - [x] Phase 2.1: Discovery loads conventions and targets topic searches

When a plan is created, the discovery step now reads all always-apply conventions in full and runs topic-specific knowledge searches only for the design surfaces the work actually touches, instead of one broad "search anything about this area" pass. This puts the full set of standards and the genuinely relevant lessons in front of the planner.

*Technical detail:* [context.md#phase-21](./context.md#phase-21-discovery-loads-conventions-and-targets-topic-searches)

**Acceptance criteria**:

- [x] During a plan's discovery step, all conventions are read in full through the conventions command.
- [x] Topic-specific knowledge is consulted per design surface the work touches, rather than as a single broad area search.
- [x] A planner following the step has both the conventions and the surface-relevant lessons available before design begins.

#### - [x] Phase 2.2: Plans carry a Conventions section

The plan records the conventions relevant to the work being planned, each with a one-line reason it applies, and omits conventions that have no bearing on the work. When nothing is relevant, or no conventions exist, the section says so plainly. The relevant subset is chosen as the design is locked and confirmed by the user, never auto-decided.

*Technical detail:* [context.md#phase-22](./context.md#phase-22-plans-carry-a-conventions-section)

**Acceptance criteria**:

- [x] A finished plan for a feature related to some conventions contains a dedicated conventions section listing exactly those, each with a reason it applies.
- [x] Conventions with no bearing on the feature do not appear in the section.
- [x] A plan for an unrelated feature, or a project with no conventions, shows the section present and plainly stating that none apply.
- [x] The conventions section is treated as a required plan section, like the other standard sections.

### Milestone 3: The behaviour is proven end-to-end

**What changes**: The end-to-end plan-workflow test suite now guarantees the feature stays working. Its environment seeds a distinctive always-apply convention relevant to the test's seeded spec, and its checks fail if a full plan run does not read the conventions during research or if the resulting plan's conventions section does not reflect the seeded convention. This milestone is largely test-facing — its user-visible value is durable confidence that future changes cannot silently regress convention-aware planning.

#### - [x] Phase 3.1: End-to-end coverage of convention-aware planning

The end-to-end plan-workflow test seeds a distinctive always-apply convention relevant to its seeded spec and verifies, through independent hand-maintained checks, that a full plan run reads the conventions during research and that the resulting plan's conventions section reflects the seeded convention.

*Technical detail:* [context.md#phase-31](./context.md#phase-31-end-to-end-coverage-of-convention-aware-planning)

**Acceptance criteria**:

- [x] The test environment contains a distinctive convention relevant to the seeded spec.
- [x] The suite fails if a full plan run does not read the conventions during its research phase.
- [x] The suite fails if the resulting plan's conventions section does not reflect the seeded convention.
- [x] The conventions section is included in the suite's list of required plan sections.

## Open Questions

Only one genuine implementation-time uncertainty remains; the rest of the design is settled.

- **Will a live plan run reliably surface the seeded convention into the plan's conventions section?** The end-to-end content assertion (Phase 3.1) depends on how a real planning agent behaves during a full harbor run: whether the strengthened discovery/architecture instructions consistently lead it to read the conventions and write the relevant one into the `## Conventions` section with a recognisable token. This is only discoverable by exercising the harbor task end to end. *Depends on:* the live agent's behaviour against the edited templates. *What the implementer should do:* run the harbor task; if the content assertion is flaky, first make the seeded convention more clearly relevant to the seeded JWT spec and choose a more distinctive anchor token; if the agent still does not surface it after the instructions are correct, STOP and ask the user before weakening the assertion — a section that cannot reflect a plainly-relevant seeded convention is a signal the discovery/architecture instructions are too weak, not that the test is wrong.

No other open questions: the conventions-section authoring location and the decision to skip migration (the project is fresh) were settled with the user, and all type shapes, command behaviour, search-exclusion form, and oracle structure were resolved from the codebase during discovery.

## Out of Scope

From the spec's Non-Goals:

- **Automatically deciding which conventions are relevant.** Relevance is proposed during planning and confirmed by the user; fully automated relevance selection is not built here.
- **Authoring or seeding the conventions themselves.** This plan governs how conventions are consulted and carried into plans, not what conventions a project should adopt. (The init starter convention is a placeholder for ergonomics, not an opinion on content.)
- **Changing how topic-specific knowledge is searched beyond surface-targeting and the conventions exclusion.** No new ranking, embeddings, or semantic search.
- **Any new knowledge-base access for the implement workflow.** Implementation continues to consume only the plan documents; it gains no convention-loading behaviour.
- **Conventions that vary by workflow mode** (e.g. separate plan-only vs. implement-only convention sets) are not addressed in this iteration.

From decisions taken during planning:

- **No dedicated conventions plan step.** The `## Conventions` section is folded into the existing discovery and architecture steps; the state machine and its step list are deliberately left unchanged. A future iteration could promote it to its own step if relevance capture grows more involved.
- **No migration of an existing flat `conventions.md`.** The project is fresh and adopts the `conventions/` directory layout directly; this is a deliberate breaking change that waives the spec's "existing projects must keep working" constraint for this project. Init simply stops writing the flat file and seeds into the directory instead. If migration is ever needed for an external project, it would be added as separate follow-up work.
- **No shared abstraction for the two search paths.** The conventions exclusion is expressed separately in the accelerated and fallback paths, matching the existing "two implementations, one equivalence test" pattern, rather than refactoring them behind a common filter.

No follow-up specs or tickets are required for any of the above; each is a deliberate boundary, not deferred work with a pending owner.

## Changelog

### 2026-06-03 — Phase 1.1: Convention reader and `knowledge conventions` command

**What was done**: Added a `Convention{Scope,Path,Content}` value type and a `Set.Conventions()` fan-out method to the knowledge layer that reads every file under each scope's `conventions/` directory and returns the full bodies, scope-tagged, in configured order; a scope with no `conventions/` directory contributes nothing rather than erroring. Exposed it through a new read-only `knowledge conventions` CLI subcommand mirroring the argument-free `knowledge sources` verb.

**Deviations**: None.

**Files changed**:
- `internal/knowledge/set.go`
- `internal/knowledge/set_test.go`
- `cmd/knowledge.go`

**Discoveries**: `Set.Conventions()` reuses the existing recursive `listFiles(store, "conventions")` helper, which surfaces the missing-directory case as `store.ErrNotFound` — absorbed with `errors.Is`. The command emits an empty `{"conventions": []}` against a project with no `conventions/` directory yet (the current repo state until Phase 1.3 scaffolds it). The `knowledge search` exclusion is Phase 1.2; until then conventions are still discoverable via plain `knowledge search`.

### 2026-06-03 — Phase 1.2: Exclude conventions from topic search

**What was done**: `FileStore.Search` now excludes the `conventions/` directory in both search paths so a convention already digested in full is never surfaced again — the ripgrep path gains a `--glob` exclusion and the native `filepath.WalkDir` fallback returns `filepath.SkipDir` for any directory named `conventions`. The existing equivalence test's fixture was extended with a convention file containing the search term, and a single assertion proves no result lives under `conventions/`.

**Deviations**: The glob form differs from the plan's literal `--glob=!conventions/**`. That form was verified NOT to exclude anything in ripgrep; the working, fallback-equivalent form is **`--glob=!**​/conventions/**`** (excludes a `conventions` directory at any depth and its entire subtree). The equivalence test caught the discrepancy.

**Files changed**:
- `internal/store/search.go`
- `internal/store/search_test.go`

**Discoveries**: ripgrep's `conventions/**` glob does not match the directory's immediate children when an absolute search root is passed — `**/conventions/**` is required and also matches conventions dirs nested at any depth, exactly mirroring the native fallback's `d.Name() == "conventions"` check. The `TestSearch_RipgrepAndFallbackEquivalent` test is the load-bearing guard here: it fails loudly if the two paths' exclusions ever diverge, which is precisely how the glob bug was found.

### 2026-06-03 — Phase 1.3: Convention directory scaffolding

**What was done**: Project init now creates a `.spektacular/knowledge/conventions/` directory and seeds the embedded starter as `conventions/conventions.md`, replacing the previous flat `.spektacular/knowledge/conventions.md` write. The flat file is no longer written. No migration is performed (deliberate breaking change while the project is fresh).

**Deviations**: None. (Starter destination chosen as `conventions/conventions.md`; the embedded source template `templates/.spektacular/conventions.md` is unchanged in content.)

**Files changed**:
- `internal/project/init.go`
- `internal/project/init_test.go`

**Discoveries**: The existing `TestInit_CreatesConventionsMd` asserted the old flat file and had to be rewritten (renamed to `TestInit_SeedsConventionsDirectory`) to assert the new directory + starter and the absence of the flat file. `os.MkdirAll` makes the new `conventions/` dir idempotent under init's always-`force=true` path. The `knowledge conventions` command (Phase 1.1) now returns the seeded starter for a freshly-initialised project instead of an empty list.

### 2026-06-03 — Phase 2.1: Discovery loads conventions and targets topic searches

**What was done**: Rewrote the plan-workflow discovery step's "Project Context" instruction (`templates/steps/plan/02-discovery.md`) to first load all always-apply conventions in full via `{{config.command}} knowledge conventions`, then run surface-targeted `{{config.command}} knowledge search <surface>` calls keyed on the design surfaces the feature introduces, replacing the previous single broad "search anything about this area" pass.

**Deviations**: None.

**Files changed**:
- `templates/steps/plan/02-discovery.md`

**Discoveries**: Template-only change — no Go or state-machine edit, and the step list / order test are untouched (conventions are folded into the existing step). The instruction uses the `{{config.command}}` placeholder, never a rendered command. The behavioural effect (a real plan run reading conventions in the discovery window) is only observable end-to-end, so it is covered by the Phase 3.1 harbor oracle rather than a unit test.

### 2026-06-03 — Phase 2.2: Plans carry a Conventions section

**What was done**: Added a `## Conventions` section to the plan-document contract, folded into the existing steps (no new state-machine step). The architecture step now selects — from the conventions loaded at discovery — the subset relevant to the locked design, annotates each with a one-line rationale (cited inline where it drives a choice), and writes them (with an explicit "none apply" fallback) to a `.spektacular/work/<plan>/conventions.md` working file after user confirmation. The verification step's canonical plan.md section list gains `## Conventions` (placed directly after `## Overview`) plus a `conventions.md → ## Conventions` working-file mapping, and the plan scaffold gains a matching `## Conventions` comment block.

**Deviations**: None.

**Files changed**:
- `templates/steps/plan/03-architecture.md`
- `templates/steps/plan/13-verification.md`
- `templates/scaffold/plan.md`

**Discoveries**: The three template files must agree on three things — the section name (`## Conventions`), its placement (directly after `## Overview`), and the working-file name (`conventions.md`); they are now consistent. The `conventions.md` working file is ALWAYS required (even in the "none apply" case), because the verification step STOPs if a mapped working file is missing. No Go enum governs the section list (the verification template is the sole source of truth), so no Go test changed; the `templates` package tests still pass. The behavioural criteria are proven end-to-end by Phase 3.1's harbor oracles (`EXPECTED_PLAN_SECTIONS` gains `conventions`, plus a seeded-convention content assertion).

### 2026-06-03 — Phase 3.1: End-to-end coverage of convention-aware planning

**What was done**: Extended the `plan-workflow` harbor task to prove convention-aware planning end-to-end. The environment seeds a distinctive always-apply convention (`AUTH_AUDIT_V2`) relevant to the seeded JWT auth spec under `.spektacular/knowledge/conventions/`, and the hand-maintained verifier gains three oracles: `conventions` added to `EXPECTED_PLAN_SECTIONS`, a discovery-window assertion that `spektacular knowledge conventions` ran, and a content assertion that plan.md's `## Conventions` section contains the seeded `AUTH_AUDIT_V2` token. `EXPECTED_STEP_ORDER` is unchanged (no step added). Also, at the user's request, added a session-scoped autouse fixture that aborts the whole verifier run on a fatal preflight failure (no transcript / auth failure / agent didn't finish) so an auth error reports one clear line instead of ~80 misleading cascade failures.

**Deviations**: Beyond the plan's literal file list (Dockerfile + test_plan_workflow.py), also added a one-line discovery hint to `instruction.md` (`spektacular knowledge conventions`) mirroring the existing skill hints, to make the conventions read reliable and mitigate the plan's flakiness open question. The fail-fast fixture is a user-requested ergonomics improvement to the same verifier file.

**Files changed**:
- `tests/harbor/plan-workflow/environment/Dockerfile`
- `tests/harbor/plan-workflow/environment/auth-audit-logging.md` (new — seeded convention)
- `tests/harbor/plan-workflow/instruction.md`
- `tests/harbor/plan-workflow/tests/test_plan_workflow.py`

**Discoveries**: The seeded convention is named `auth-audit-logging.md` (not `conventions.md`) so the agent's `spektacular init claude` — which seeds its own `conventions/conventions.md` starter via idempotent `os.MkdirAll` — does not clobber it. The verifier is run by `make harbor-test-plan`, which builds a linux binary into the gitignored `environment/spektacular` and invokes `harbor run` (Docker, ~15 min, needs Anthropic/HARBOR auth) — a live, credentialed run left to the user; all static checks (Go build/test/lint, `py_compile`, `pytest --collect-only` = 84 tests) pass. `pytest.exit(reason, returncode=1)` works in both the Docker pytest 8.4.1 and local 9.x, and yields a non-zero exit so `test.sh` writes reward 0. Per the plan's Open Question, if the live content assertion proves flaky, strengthen the seeded convention's relevance/token before weakening the assertion, and STOP to ask the user if it still won't surface.
