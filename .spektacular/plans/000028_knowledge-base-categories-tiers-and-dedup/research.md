# Research: 000028_knowledge-base-categories-tiers-and-dedup

## Overview of the surface

The knowledge base is a three-layer stack:

- **`internal/store`** — generic file-backed store. `Store` interface (Read/Write/Delete/List/Search/Root/Exists). `FileStore` implements in-process full-text search (the ripgrep replacement, spec 000026) in `internal/store/search.go`. `List` is one level deep; recursion lives in the knowledge layer. The `Hit` struct (`internal/store/store.go:24-30`) carries `Scope, Path, Title, Excerpts, Score` — **no category field today**.
- **`internal/knowledge`** — `Set` is an ordered collection of scoped stores (`set.go`). Fans Read/Write/List/Search/Conventions across sources, tags by scope, ranks search globally (score desc, then source order, then path).
- **`cmd/knowledge.go`** — the CLI surface: `search`, `read`, `list`, `write`, `sources`, `conventions`. Each emits a JSON envelope with a `--schema` mode.

Categories today = the first path segment / subdirectory under a scope (`conventions/`, `architecture/`, `gotchas/`, `learnings/`). They are a filesystem convention, not a modelled concept. Their "definitions" are circular placeholder READMEs ("This directory contains X documentation.") generated at init.

The skill `spek-knowledge` (`templates/skills/workflows/spek-knowledge/SKILL.md`) is a static playbook with three intents (lookup / contribute / update). Lookup just dumps ranked search hits; contribute asks for scope+path with **no category guidance**; there is **no dedup/consolidation and no lookup sub-agent** today.

## Alternatives considered and rejected

- **Tier membership in config.yaml (an `always_applied: [conventions, glossary]` list under KnowledgeConfig).** Rejected as the *source of truth*: the spec wants a category's tier stated once *alongside* its definition (purpose/boundary/shape), and config.yaml has no place for definitions. Splitting tier (config) from definition (elsewhere) reintroduces the very "asserted in more than one place" problem the spec calls out (`spec` Technical Approach, `set.go:167` + `search.go:61` are today's two places). A single Go category registry holds tier + definition together. (Evidence: `internal/config/config.go:70-86` has only Sources; no category notion.)
- **Add `Category` to the generic `store.Hit` and populate it inside `FileStore`.** Rejected: "category = first path segment" is a knowledge-domain rule; the store is deliberately generic (`store.go:24-30`). Populate the additive `Category` field in `knowledge.Set.Search` instead, where path→category meaning lives. Keeps the store dumb.
- **Keep the conventions-skip in `FileStore.search` and add a glossary-skip beside it.** Rejected: that grows the hardcoded-literal problem (two→four literals) and keeps tier knowledge in the wrong layer. Better: remove the skip from the store entirely and have `Set.Search` drop hits whose category is in the always-applied set (registry-driven). One declaration governs both exclusion and full-load. (Evidence of the skip: `internal/store/search.go:57-63`.)
- **Enforce the glossary "short definition only" shape with a Go length check at write.** Rejected: spec AC says a long entry is *steered* to another category (judgement), and retroactive enforcement is an explicit Non-Goal. Shape lives in the category definition + skill prose, not a CLI guard. (Spec Non-Goals; AC "Category model".)
- **Build dedup into search (tag duplicates during ranking).** Rejected and pre-empted by the spec Technical Approach: a `Hit` has no content, so duplication cannot be judged at search time (same path/diff content = override; diff path/same content = copy). Dedup is post-read. (Evidence: `store.Hit` has no content field, `store.go:24-30`.)
- **A new `knowledge lookup` command bundling search→read→exact-dedup.** Considered (would centralise the deterministic exact-dedup in testable Go). Rejected after design discussion: a content `checksum` added additively to the search `Hit` (computed during the existing search walk) gives deterministic byte-identical de-dup straight from the search result, so no new command is needed — the skill's lookup branch groups by checksum and the consolidation sub-agent does the rest. (Evidence: `search.go` already reads each file in its walk, so the hash is free; `store.Hit` is the additive carrier, `store.go:24-30`.)
- **Normalized or fuzzy/similarity fingerprint (SimHash/MinHash) instead of an exact checksum.** Rejected: lexical distance does not track semantic distance ("always retry"/"never retry" ~95% lexically identical but contradictory; "add jitter"/"randomize backoff" ~0 overlap but equivalent). A similarity threshold mis-ranks exactly these cases and is a mechanical equivalence index — the explicit "no embedding/semantic index" Non-Goal in spirit. The need is relationship *classification* (equivalent/refines/contradicts/distinct), which is LLM judgement. Exact byte-identity is the narrow, provable mechanical primitive; everything non-identical is the sub-agent's job. (Spec Non-Goals; architecture decision confirmed with user.)

## Chosen approach — evidence

- **Single category registry in Go** (new `internal/knowledge/category.go`): canonical list of `Category{Name, Purpose, Boundary, Tier, EntryShape}`. Drives (a) init scaffolding of READMEs, (b) the always-applied set for both search-exclusion and full-load, (c) a new `knowledge categories` command surfaced to the skill at contribution time. This is the "single declaration" the spec asks for. Mirrors the existing constant-driven style in `internal/config/config.go:14-37` and the hardcoded category loop in `internal/project/init.go:87`.
- **`Category` field added to `store.Hit`** (additive; existing consumers unaffected per Constraint), populated in `knowledge.Set.Search` (`set.go:90-120`) from the first path segment. The search output schema (`cmd/knowledge.go:57-74`) gains a `category` property.
- **Generalize the always-applied reader.** `Set.Conventions()` (`set.go:164-183`) reads only `conventions/`. With glossary also always-applied, introduce a registry-driven `Set.AlwaysApplied()` that reads every always-applied category, tagged with category. Keep `knowledge conventions` working for backward-compat (Constraint: existing always-applied behaviour preserved), and update the 5 step templates that load conventions to load the full always-applied set. (Touchpoints: `templates/steps/plan/{02-discovery,03-architecture,08-testing_approach,13-assemble}.md`, `templates/steps/implement/04-test.md`.)
- **Dedup pipeline = deterministic Go exact-dedup (via checksum on the hit) + assistant-judgement semantic merge.** Spec Constraint mandates the split. The store adds an exact `Checksum` (SHA-256 of raw bytes) to each search `Hit`, computed during the existing search walk; the skill's lookup branch groups candidates by checksum to drop byte-identical copies deterministically (no new command, no threshold, no judgement), then hands the surviving unique bodies to a sub-agent that applies layered precedence, merges differently-worded equivalents, surfaces genuine conflicts, and returns a single cited answer. Sub-agent orchestration is already an assumed capability (`go run . skill spawn-planning-agents`: "Use your agent orchestration capability").
- **Documentation deliverable**: no `docs/` dir exists today; the durable doc is a new markdown file (e.g. `docs/knowledge-base.md`) covering category model, two tiers, search+category label, dedup/consolidation, precedence.

## Files examined

- `internal/store/store.go:24-30` — `Hit` struct (scope/path/title/excerpts/score; no category). `:32-52` Store interface; `:122-139` List is one-level (os.ReadDir).
- `internal/store/search.go:40-103` — in-process search; `:57-63` the hardcoded `conventions` SkipDir (place #1 of the two-place tier hardcode).
- `internal/knowledge/set.go:90-120` — `Set.Search` global ranking (best place to tag category + filter always-applied). `:144-156` List recursion. `:164-183` `Conventions()` reads literal `"conventions"` dir (place #2). `:31-44` Entry/Convention structs.
- `cmd/knowledge.go:51-55,228-245` — `conventions` command. `:57-74` search output schema (add `category`). `:39-43,193-214` write command (no category routing).
- `internal/config/config.go:70-86` — KnowledgeConfig/SourceConfig (scopes only, no tiers). `:14-37` constants style. `:199-217` knowledge Validate.
- `internal/project/init.go:35-46` dir scaffold (hardcoded 4 categories), `:87-94` circular README generation loop — both must become registry-driven and add glossary+decisions.
- `templates/skills/workflows/spek-knowledge/SKILL.md` — full skill; lookup (`:22-29`) dumps hits, contribute (`:31-43`) has no category routing. Needs: category definitions at contribute time + consolidation sub-agent at lookup time. Uses `{{command}}` placeholder.
- `internal/agent/skills.go:25-65` — skill templating; mustache `{{command}}` → `cfg.Command`; 4 workflow skills rendered into `.claude/skills/`.
- `templates/agents/memory-context.md` + `internal/agent/memory_context.go:24-56` — AGENTS.md memory-context install (routes writes through spek-knowledge).
- Tests: `internal/store/{store,search}_test.go` and `internal/knowledge/set_test.go`, `cmd/knowledge_test.go` — **testify/require**, `t.TempDir()` fixtures, `Test<Func>_<Criterion>` naming, standalone (not table-driven) scenarios. `set_test.go:275-332` conventions tests; `cmd/knowledge_test.go:142-170` search hit-fields test.

## External references

- None required. Self-contained Go + skill-prose change. (Constraint forbids external services/network in dedup; sub-agent uses the assistant capability the project already uses.)

## Prior plans / specs consulted

- `000026_ripgrep-replace` / `000027_document-level-search` — established the in-process `FileStore.search` and the one-hit-per-document ranked model that this work tags with category. Confirms search is in-process and must stay so (Constraint).
- Knowledge-base search surfaced only stale JSON transcript dumps under `architecture/example-output/` and `architecture/bob_json/` — no relevant prior design notes; conventions are placeholder-only.

## Open assumptions

- **Sub-agent orchestration is available to the executing agent** for the lookup-consolidation step. Justified by `spawn-planning-agents` already assuming it, but if the target agent lacks it, the consolidation must run inline in the skill's main context (degraded: main agent sees raw candidate dump). Implement workflow should confirm before relying on isolation.
- **Layered scope precedence** (most-specific scope wins for the same locator; genuine disagreement surfaced) — CONFIRMED with the user during architecture (no longer open). Recorded here as the resolution of the spec's open decision.
- `knowledge conventions` must keep returning conventions-only for backward compat; the *templates* switch to the generalized always-applied loader. Assumed no external consumer depends on `conventions` excluding glossary.
- Adding `Category` to `store.Hit` is safe additive change; no marshalled consumer breaks on an extra JSON field.

## Rehydration cues

- Re-read this surface: `internal/store/search.go:40-103`, `internal/knowledge/set.go:90-183`, `cmd/knowledge.go:51-74,228-245`, `internal/project/init.go:35-94`, `internal/config/config.go:70-133`, `templates/skills/workflows/spek-knowledge/SKILL.md`.
- Find the two-place tier hardcode: `grep -rn '"conventions"' --include='*.go' internal cmd | grep -v _test`.
- Find template touchpoints for always-applied loading: `grep -rln conventions templates/steps`.
- Test conventions: `internal/{store,knowledge}/*_test.go`, `cmd/knowledge_test.go` — testify/require, `t.TempDir()`, `Test<Func>_<Criterion>`.
- Spec: `go run . spec file read 000028_knowledge-base-categories-tiers-and-dedup.md`.
