# Context: 000028_knowledge-base-categories-tiers-and-dedup

## Current State Analysis

The knowledge base is a three-layer stack:

- **`internal/store`** — generic file-backed store. `Store` interface (Read/Write/Delete/List/Search/Root/Exists) at `store.go:32-52`. `FileStore` implements in-process full-text search (the ripgrep replacement, spec 000026) in `search.go:40-103`; `List` (`store.go:122-139`) is one level deep, recursion lives in the knowledge layer. `Hit` (`store.go:24-30`) carries `Scope, Path, Title, Excerpts, Score` — no category/checksum today. Search excludes the `conventions` dir via a `filepath.SkipDir` at `search.go:57-63` (one of the two hardcoded always-applied spots).
- **`internal/knowledge`** — `Set` (`set.go:26-29`) is an ordered collection of scoped stores. `Search` (`set.go:90-120`) fans across sources, ranks score desc → source order → path. `Conventions()` (`set.go:164-183`) reads the literal `"conventions"` dir (the second hardcoded spot). `Convention{Scope,Path,Content}` at `set.go:40-44`. `Entry{Scope,Path}` at `set.go:31-35`.
- **`cmd/knowledge.go`** — subcommands `search`/`read`/`list`/`write`/`sources`/`conventions`, each with a `--schema` mode and JSON envelope. Search output schema at `:57-74`; conventions cmd at `:51-55,228-245`; sources cmd (template for the new `categories` cmd) at `:45-49,216-226`; `AddCommand` registration at `:295`.
- **`internal/config/config.go`** — `KnowledgeConfig{Sources}` / `SourceConfig` at `:70-86`; no tier notion. Constants style at `:14-37`.
- **`internal/project/init.go`** — scaffolds hardcoded category dirs at `:35-46` and writes circular READMEs via a `[]string{"learnings","architecture","gotchas","conventions"}` loop at `:87-94`.
- **`internal/agent/skills.go:40-65`** — renders skill templates with mustache `{{command}}` → `cfg.Command`; the rendered `.claude/skills/spek-knowledge/SKILL.md` is regenerated from the template on init — edit the template, never the rendered copy.
- **`templates/skills/workflows/spek-knowledge/SKILL.md`** — lookup intent (`:22-29`) dumps raw hits; contribute intent (`:31-43`) has no category routing; no dedup/consolidation today.
- **Templates loading conventions** (must switch to the always-applied set): `templates/steps/plan/02-discovery.md`, `03-architecture.md`, `08-testing_approach.md`, `13-assemble.md`, `templates/steps/implement/04-test.md`.

Categories today = first path segment / subdirectory; "definitions" are circular placeholder READMEs. Tests across `internal/{store,knowledge}` and `cmd` use **testify/require**, `t.TempDir()` fixtures, `Test<Func>_<Criterion>` naming, standalone (not table-driven). Existing conventions tests: `set_test.go:275-332`; existing conventions-exclusion search test: `search_test.go:91-125`; search hit-fields/schema tests: `cmd/knowledge_test.go:142-170,256-281`.

**Resolved design decisions (carried from planning):** layered scope precedence (most-specific wins; conflicts surfaced); exact byte-identical checksum (no normalization, no fuzzy/SimHash — lexical distance ≠ semantic distance, relationship classification is judgement); category registry in Go as single source of truth (not config.yaml); additive `Category`+`Checksum` on `store.Hit`; no new `knowledge lookup` command; consolidation is a skill sub-agent.

## Per-Phase Technical Notes

### Phase 1.1: Category registry and tiers

- New `internal/knowledge/category.go`: define `Tier` (string const `TierAlwaysApplied`, `TierLookedUp`) and `Category{Name, Purpose, Boundary, Tier, EntryShape}`. Export the canonical ordered list `Categories` and accessors `AlwaysApplied() []string` (names where `Tier==TierAlwaysApplied`) and a lookup-by-name. Populate the six categories with final definition wording: `conventions` (always-applied), `glossary` (always-applied, shape "a term and a short gloss"), `architecture`, `gotchas`, `learnings`, `decisions` (all looked-up). The definition wording is a deliverable of this work — write real purpose/boundary text, not placeholders.
- New `internal/knowledge/category_test.go`: assert each category has non-empty Purpose/Boundary/EntryShape; assert glossary∈always-applied and decisions∈looked-up; assert `AlwaysApplied()` == {conventions, glossary}.

**Complexity**: Low. Pure data + accessors; no integration yet.
**Token estimate**: ~12k
**Agent strategy**: Single agent, sequential. This phase blocks all others — land and verify first.

### Phase 1.2: Initialise projects from the registry

- `internal/project/init.go:35-46`: replace the hardcoded `knowledge/<cat>` dir list with a loop over `knowledge.Categories` (keep the `knowledge` root + spec/plan dirs as-is).
- `internal/project/init.go:87-94`: replace the `[]string{"learnings","architecture","gotchas","conventions"}` loop and the `fmt.Sprintf("# %s\n\nThis directory contains %s documentation.\n", ...)` circular text with README content rendered from each category's registry definition (title + purpose + boundary + tier + entry shape).
- Add/extend `internal/project/init_test.go` (or the existing init test): assert a fresh init creates glossary + decisions dirs and that no README contains "This directory contains".

**Complexity**: Low
**Token estimate**: ~12k
**Agent strategy**: Single agent, sequential. Depends on 1.1.

### Phase 1.3: Expose category definitions to contributors

- `cmd/knowledge.go`: add `knowledgeCategoriesCmd` (cobra subcommand `categories`) following the shape of `knowledgeSourcesCmd:45-49` + `runKnowledgeSources:216-226`; add `runKnowledgeCategories` returning `{"categories": <registry projected to JSON>}`; add `knowledgeCategoriesOutputSchema` mirroring `knowledgeSourcesOutputSchema:98-101`; register it in `init()` at the `AddCommand` call `:295`.
- `cmd/knowledge_test.go`: add a test asserting the command lists every category with purpose/boundary/tier/entryShape, plus a `--schema` test mirroring the existing schema tests `:238-280`.

**Complexity**: Low
**Token estimate**: ~14k
**Agent strategy**: Single agent, sequential. Depends on 1.1.

### Phase 2.1: Make the store category-agnostic and label-ready

- `internal/store/search.go:57-63`: remove the `if d.Name() == "conventions" { return filepath.SkipDir }` block (and update the package comment at `:36` that documents the exclusion).
- `internal/store/store.go:24-30`: add a `Category` field (json tag `category`) to `Hit` (left empty by the store; populated by the knowledge layer).
- `internal/store/search_test.go:91-125`: the existing `TestSearch_..ExcludesConventions` test must be updated/retired — at the store level conventions are no longer excluded (exclusion moves to the knowledge layer, tested in 2.2). Keep a test asserting search now *does* return a conventions-dir file at the store layer, documenting the moved responsibility.

**Complexity**: Low-Medium (touches a load-bearing existing test).
**Token estimate**: ~16k
**Agent strategy**: Single agent, sequential. This is a behaviour move, not a deletion — be explicit in the test about where exclusion now lives.

### Phase 2.2: Tier behaviour and category tagging in the knowledge layer

- `internal/knowledge/set.go:90-120` (`Search`): after merging hits, set each hit's `Category` to the first path segment of `hit.Path`; drop hits whose category ∈ `knowledge.AlwaysApplied()`. Keep the existing ranking/tie-break.
- `internal/knowledge/set.go:158-183` (`Conventions`): generalise into an always-applied reader that iterates `AlwaysApplied()` categories (not the literal `"conventions"` at `:167`), reading each via `listFiles`, returning records tagged with category. Keep a thin `Conventions()` that returns conventions-only for the backward-compat CLI command, or have the command filter the always-applied result to conventions — preserve the existing `knowledge conventions` JSON exactly.
- Update `Convention` struct (`set.go:40-44`) or add an always-applied entry type carrying `Category`.
- `internal/knowledge/set_test.go`: add tests — category tagging on hits; always-applied categories excluded from search; **the one-place re-tier test** (via a test category flip, assert load+exclude flip together); always-applied reader returns conventions+glossary tagged; `set_test.go:275-332` conventions tests still pass.

**Complexity**: Medium. Core behavioural change; the re-tier test is the load-bearing assertion.
**Token estimate**: ~24k
**Agent strategy**: Single agent, sequential. Depends on 1.1 and 2.1. Verify before 2.3.

### Phase 2.3: Surface the label and load the full always-applied set

- `cmd/knowledge.go:57-74`: add `category` to `knowledgeSearchOutputSchema` hit properties. If a dedicated always-applied loader command is added, mirror `knowledgeConventionsCmd:51-55` + schema `:103-106`; otherwise document that templates call the generalised reader.
- Templates that load conventions → load the full always-applied set: `templates/steps/plan/02-discovery.md`, `templates/steps/plan/03-architecture.md`, `templates/steps/plan/08-testing_approach.md`, `templates/steps/plan/13-assemble.md`, `templates/steps/implement/04-test.md`. Update wording to load conventions + glossary (the always-applied set) and to note search results now carry a category label.
- `cmd/knowledge_test.go:256-281`: extend the search-schema test to assert the `category` field.

**Complexity**: Medium (many template touchpoints, but mechanical).
**Token estimate**: ~22k
**Agent strategy**: 2 parallel agents — one for the CLI/schema change, one for the five template edits — then integrate. Depends on 2.2.

### Phase 3.1: Content checksum on results

- `internal/store/store.go:24-30`: add a `Checksum` field (json tag `checksum`) to `Hit`.
- `internal/store/search.go` (`scanFile` / the walk at `:50-103`,`:154-204`): accumulate `crypto/sha256` over the file's raw bytes during the existing read; set `Hit.Checksum` to the hex digest. Ensure the hash covers exact bytes (read full content for the hash even though scanning is line-oriented).
- `internal/store/search_test.go`: assert identical fixtures → identical checksum, one-byte-different → different. Oracle = an independently-known SHA-256 hex of a fixed fixture string (do NOT recompute via `scanFile`).

**Complexity**: Low-Medium
**Token estimate**: ~16k
**Agent strategy**: Single agent, sequential.

### Phase 3.2: Consolidated lookup with de-dup and precedence

- `templates/skills/workflows/spek-knowledge/SKILL.md:22-29` (Intent: lookup): rewrite. New flow: run `{{command}} knowledge search`; group candidate hits by `checksum`, collapse byte-identical copies (union their scope/path citations) — deterministic, no judgement; spawn a consolidation sub-agent that reads the surviving unique bodies via `{{command}} knowledge read`, merges differently-worded equivalents, applies layered precedence (project>team>global / most-specific wins), surfaces genuine contradictions, and returns one cited answer; present that consolidated answer, never the raw hit list. Document the sub-agent contract and the layered-precedence rule inline.
- The rendered `.claude/skills/spek-knowledge/SKILL.md` is regenerated on init (`internal/agent/skills.go:40-65`) — edit the template, not the rendered copy.

**Complexity**: Medium (prose design; no Go). Judgement-tier behaviours verified manually per the implementation test plan.
**Token estimate**: ~20k
**Agent strategy**: Single agent, sequential. Depends on 3.1 (checksum) and 2.3 (category label).

### Phase 3.3: Route contributions to the right category

- `templates/skills/workflows/spek-knowledge/SKILL.md:31-43` (Intent: contribute): insert a routing step before staging — run `{{command}} knowledge categories`, present the definitions, pick the category whose Purpose matches and whose Boundary doesn't exclude the entry; steer over-long content out of `glossary` per its entry shape. Path becomes `<category>/<slug>.md`.

**Complexity**: Low (prose).
**Token estimate**: ~12k
**Agent strategy**: Single agent, sequential. Depends on 1.3 (categories command). Can share the agent with 3.2 since both edit the same SKILL.md — sequence them to avoid edit conflicts.

### Phase 4.1: Write the knowledge-base documentation

- New `docs/knowledge-base.md` (no `docs/` dir exists yet — create it). Cover: category model + the six categories and their definitions; the two retrieval tiers and the single-declaration source of truth; search behaviour incl. the per-result category label; de-duplication (checksum exact-dedup, deterministic) and consolidation (sub-agent semantic merge); the exact-vs-fuzzy rationale (lexical≠semantic distance; relationship classification is judgement); layered source precedence and conflict surfacing.
- Consider linking it from the top-level `README.md` if it indexes docs.

**Complexity**: Low (writing).
**Token estimate**: ~14k
**Agent strategy**: Single agent, sequential. Last — describes the now-final behaviour. Verified manually per the implementation test plan.

## Testing Strategy

Per-phase test focus:

- **1.1** — `internal/knowledge/category_test.go`: registry completeness (non-empty Purpose/Boundary/EntryShape per category), tier membership (glossary always-applied, decisions looked-up), `AlwaysApplied()` == {conventions, glossary}.
- **1.2** — `internal/project` init test: fresh init creates glossary + decisions dirs; no README contains "This directory contains".
- **1.3 / 2.3** — `cmd/knowledge_test.go`: `categories` command lists all categories with definition fields + `--schema`; search schema includes `category` (and later `checksum`).
- **2.1** — `internal/store/search_test.go`: store no longer excludes the conventions dir (responsibility-moved test); `Hit` has additive `Category`.
- **2.2** — `internal/knowledge/set_test.go`: hit category tagging; always-applied exclusion in search; **one-place re-tier test** (flip a category's tier → load + search-exclusion flip together — the spec's "single self-consistent action"); always-applied reader returns conventions+glossary tagged; existing conventions tests (`set_test.go:275-332`) still green; legacy `conventions` output unchanged.
- **3.1** — `internal/store/search_test.go`: checksum equal for byte-identical, different for one-byte-diff; **oracle is an independently-known SHA-256 hex of a fixed fixture, not recomputed via production code**.
- **3.2 / 3.3 / 4.1** — judgement/prose behaviours (semantic merge, conflict surfacing, routing quality, doc comprehension): **Manual — captured in the implementation test plan**; no deterministic oracle.

Conventions: testify/require, `t.TempDir()`, `Test<Func>_<Criterion>`, standalone scenarios. Tests render fixtures into their own scratch dir via production paths — never read the repo's live knowledge tree. Regression: existing search ranking, read, write, list, and `conventions` outputs must be unchanged.

## Project References

Key implementation sites (full detail in the per-phase notes above):

- **Category registry (new)**: `internal/knowledge/category.go` + `category_test.go`.
- **Store hit + search**: `internal/store/store.go:24-30` (`Hit`), `internal/store/search.go:40-103` (walk), `:57-63` (the conventions skip to remove), `:154-204` (`scanFile`, where the checksum accumulates).
- **Knowledge Set**: `internal/knowledge/set.go:90-120` (`Search` — tag category + exclude always-applied), `:158-183` (`Conventions` → always-applied reader), `:40-44` (`Convention`/entry types).
- **Knowledge CLI**: `cmd/knowledge.go:45-49,216-226` (sources cmd — template for new `categories`), `:51-55,228-245` (conventions cmd — preserve), `:57-74` (search schema — add `category`/`checksum`), `:295` (`AddCommand`).
- **Init/scaffold**: `internal/project/init.go:35-46` (dir list), `:87-94` (README loop).
- **Skill template**: `templates/skills/workflows/spek-knowledge/SKILL.md` (lookup `:22-29`, contribute `:31-43`); rendered via `internal/agent/skills.go:40-65`.
- **Always-applied step templates**: `templates/steps/plan/{02-discovery,03-architecture,08-testing_approach,13-assemble}.md`, `templates/steps/implement/04-test.md`.
- **Docs deliverable (new)**: `docs/knowledge-base.md`.
- **Tests**: `internal/store/{store,search}_test.go`, `internal/knowledge/set_test.go` (conventions tests `:275-332`), `cmd/knowledge_test.go` (search fields `:142-170`, schema `:238-281`).

Prior specs: `000026_ripgrep-replace`, `000027_document-level-search` (established the in-process search this builds on). Spec under plan: `.spektacular/specs/000028_knowledge-base-categories-tiers-and-dedup.md`.

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|-------------|----------------|
| Low | ~10k | Single agent, sequential |
| Medium | ~25k | 2-3 parallel agents |
| High | ~50k+ | Parallel analysis, sequential integration |

Most phases are Low–Medium and run single-agent sequentially because they share the `internal/knowledge` and `cmd/knowledge.go` files and have hard ordering (1.1 blocks all; 2.1→2.2→2.3; 3.1→3.2). Phase 2.3 is the one good parallel split (CLI/schema vs. five template edits). Phases 3.2 and 3.3 edit the same `SKILL.md` — sequence them, do not parallelise.

## Migration Notes

No data migration. Existing entries are not moved or re-categorised (explicit Non-Goal). The two new categories (glossary, decisions) appear in freshly initialised projects; existing projects gain them only on re-init or by hand — acceptable because the new categories/rules apply to contributions going forward. The `category` and `checksum` fields are additive on the search envelope, so existing JSON consumers continue to parse. The `knowledge conventions` command output is preserved byte-for-byte for backward compatibility.

## Performance Considerations

The checksum (SHA-256) is computed during the search walk's existing per-file read, so it adds hashing cost but no extra I/O pass — negligible for a markdown knowledge base. De-duplication and consolidation read full bodies only for the candidates a query already surfaced (and only the unique survivors after checksum-dedup), so the read volume scales with result count, not with knowledge-base size. No new index is built or persisted; everything is computed in-process per lookup, honouring the self-contained-binary constraint.
