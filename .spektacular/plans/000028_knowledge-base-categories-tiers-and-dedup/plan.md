# Plan: 000028_knowledge-base-categories-tiers-and-dedup

<!-- Metadata -->
<!-- Created: 2026-06-17T10:48:32Z -->
<!-- Commit: 83838ef2a5bdfe08e9cd27b3c0e1c9d54bebef1f -->
<!-- Branch: main -->
<!-- Repository: git@github.com:jumppad-labs/spektacular.git -->

## Overview

This work makes the project's knowledge base clearer, more trustworthy, and easier for an AI assistant to use correctly. Knowledge is organised into well-defined categories — each with a real, self-explanatory definition and a declared retrieval tier — so contributors and the assistant know exactly what belongs where, the rules and shared vocabulary a team always wants are loaded on every task, and the larger reference body is looked up only when relevant. It also solves a trust problem: a lookup now returns a single consolidated, source-cited answer with duplicates removed and the most specific source taking precedence, instead of a pile of overlapping copies. Everyone benefits — contributors get a tidy, self-documenting knowledge base, and the assistant gives more accurate, less cluttered answers.

## Conventions

No project conventions apply to this feature. The project's knowledge base currently holds only the placeholder convention README ("This directory contains conventions documentation."), which carries no actionable guidance — fittingly, replacing exactly these circular placeholders with real category definitions is part of this feature's scope.

## Architecture & Design Decisions

The knowledge base gains a **single category registry** as its source of truth. A new `internal/knowledge/category.go` declares each category once as `Category{Name, Purpose, Boundary, Tier, EntryShape}`, where `Tier` is either *always-applied* or *looked-up*. This one declaration drives everything downstream: project init scaffolds the category directories and their README definitions from it (replacing the circular "This directory contains X documentation" placeholders generated at `internal/project/init.go:87`), the always-applied set is read from it, and a new `knowledge categories` command surfaces the definitions to the contribution skill so new entries are routed to the right category. Adding a category to the always-applied tier becomes a one-field change in the registry — satisfying the spec's "stated once, behaviours follow" requirement. Two new categories are added: `glossary` (always-applied; entry shape is a term plus a short gloss) and `decisions` (looked-up; ADR-style rationale).

Tier behaviour is unified through that single declaration, eliminating today's two independent hardcodes of the literal `"conventions"` — the search-skip at `internal/store/search.go:61` and the reader at `internal/knowledge/set.go:167`. The store layer is made category-agnostic: the conventions skip is removed from `FileStore.search`, and `knowledge.Set.Search` instead tags each hit with its `category` (derived from the first path segment) and drops hits whose category is always-applied. The dedicated `Conventions()` reader generalises to a registry-driven always-applied reader that returns every always-applied category's full content (conventions *and* glossary), each tagged with its category; the existing `knowledge conventions` command is preserved for backward compatibility, and the five step templates that load conventions today switch to the generalised loader so glossary content is also present on every task. Two fields are added additively to `store.Hit` — `category` and a `checksum` (SHA-256 over the raw entry bytes) — which keeps existing search consumers working, as the spec's "additive only" constraint requires.

De-duplication is a **post-read consolidation step**, not a search-time one, because a search hit carries no body and duplication can only be judged from full content. The flow: `knowledge search` returns ranked candidates (now carrying `category` and `checksum`); the skill's lookup branch groups candidates by checksum to remove **byte-identical** copies deterministically — pure equality, no judgement, no threshold, satisfying the spec's determinism constraint — then hands the surviving unique bodies to a consolidation **sub-agent**. The sub-agent reads each body and classifies the *relationship* between entries: same-point-different-words (merge), refinement (most-specific scope wins, per the chosen **layered** precedence), genuine contradiction (surface, never silently drop), or distinct (keep both). It returns a single cited answer; the raw per-source hit list is never the output. No new `knowledge lookup` command is introduced — checksum is the only deterministic primitive needed, and it rides on the existing search result.

The decision to keep the mechanical layer at **exact byte-identity** (rather than a normalized or similarity-based fingerprint) is deliberate and load-bearing: lexical distance does not track semantic distance. "Always retry" vs "never retry" is ~95% lexically identical yet a total contradiction; "add jitter to retries" vs "randomize backoff intervals" share almost no words yet state the same knowledge. A graded similarity score (edit-distance, SimHash/MinHash) would mis-rank exactly these cases and reduces to a brittle magic threshold — and a mechanical similarity index for equivalence is the explicit "no embedding/semantic index" Non-Goal in spirit. The need is not a difference *scalar* but a relationship *classification* (a 70%-similar pair could be either a refinement or a contradiction), which only an LLM does reliably. So exact-equivalence is the most effective *mechanical* primitive precisely because it is narrow: it claims the one provable point (zero difference) for free and routes everything semantic to the judgement layer, fed the inputs that make that judgement good — full bodies, scope/specificity order, and the category label. This rationale is captured in the durable documentation deliverable (`docs/knowledge-base.md`). See `research.md#alternatives-considered-and-rejected` for the rejected config-driven-tier, store-level-category, and search-time-dedup options with citations.

## Component Breakdown

**Category registry (new).** The single source of truth for the category model. Declares every category once — its name, purpose, boundary (what looks similar but belongs elsewhere), retrieval tier (always-applied or looked-up), and expected entry shape. Owns the canonical definitions and the derived always-applied set. Consumed by project init (scaffolding), the knowledge Set (tier behaviour), and the categories command (contribution routing). Adding or re-tiering a category is a change to this registry alone.

**Knowledge Set (changed).** The multi-source fan-out layer. Three changes: (1) Search now tags every hit with its `category`, derived from the entry's first path segment, and excludes hits belonging to always-applied categories — consulting the registry rather than a hardcoded category name, so the store layer no longer special-cases any category. (2) The conventions reader generalises into a registry-driven always-applied reader that returns the full content of *every* always-applied category (conventions and glossary), each tagged with its category. (3) It surfaces each hit's content checksum (see Store). The existing per-scope fan-out, ranking, read, write, and list behaviours are unchanged.

**Store / FileStore search (changed).** The generic file-backed store and its in-process search. Two changes: (1) the hardcoded exclusion of the `conventions` directory is removed — the store becomes category-agnostic, and exclusion moves up to the Set where the registry lives. (2) The search result hit gains two additive fields: `category` (populated by the Set) and `checksum`, a SHA-256 computed over the entry's exact raw bytes during the existing file walk. Existing hit fields, scoring, ranking, and excerpts are untouched.

**Knowledge CLI (changed/new).** The `knowledge` command surface. A new `categories` subcommand emits the registry's definitions (purpose, boundary, tier, entry shape) as a structured payload for the contribution skill. The existing `search` and (where present) read schemas gain the additive `category` and `checksum` fields. The `conventions` subcommand is preserved as-is for backward compatibility; a means to load the full always-applied set is exposed for the step templates and skill to call.

**Project init / scaffold (changed).** The `spektacular init` path. Stops hardcoding the category list and circular README text; instead it scaffolds each category's directory and writes its README definition from the category registry, so a freshly initialised project contains the glossary and decisions categories alongside the existing ones, each with a real purpose statement rather than a placeholder.

**spek-knowledge skill (changed).** The contribute/lookup/update playbook. *Contribute* gains a category-routing step: before writing, it loads the category definitions (via the categories command) and routes the entry to the category whose purpose matches, steering long content away from the glossary. *Lookup* gains the consolidation flow: it runs search, groups candidates by checksum to drop byte-identical copies deterministically, then delegates the surviving unique entries to the consolidation sub-agent and presents that single cited answer rather than the raw hit list.

**Consolidation sub-agent (new, prose-defined).** Invoked by the lookup branch. Reads the full bodies of the de-duplicated candidates and classifies the relationship between entries — equivalent (merge into one point), refinement (most-specific scope wins, per layered precedence), genuine contradiction (surface, never silently drop), or distinct (keep both) — using full bodies, scope/specificity order, and category labels as input. Returns a single consolidated, source-cited answer to the main agent, keeping the raw per-source dump out of the main context. Realised as skill prose plus the assistant's existing sub-agent orchestration capability, not as Go code.

**Always-applied loading in step templates (changed).** The workflow step templates that load conventions today switch to loading the full always-applied set, so glossary content is present on every task without being searched for. No new component — a redirection of existing template instructions to the generalised loader.

**Knowledge-base documentation (new).** A durable project document describing the category model, the two retrieval tiers, search behaviour including the per-result category label, how de-duplication and consolidation work, the exact-checksum rationale, and how layered source precedence is resolved — written so a reader new to the project understands the knowledge base without reading the source.

## Data Structures & Interfaces

**`Category` (new).** The registry record describing one category. Its fields are the category's contract with both humans and the assistant:

```
Category {
  Name       string  // directory name and path prefix, e.g. "glossary"
  Purpose    string  // what this category is for
  Boundary   string  // what looks similar but belongs elsewhere
  Tier       Tier    // TierAlwaysApplied | TierLookedUp
  EntryShape string  // expected shape of an entry, e.g. "a term and a short gloss"
}

Tier  // an enum/string-constant with exactly two values:
      //   always-applied — loaded in full every task, excluded from search
      //   looked-up      — retrieved only when a query matches
```

The registry exposes the canonical ordered list of categories and a derived `AlwaysApplied()` accessor returning the names of always-applied categories. Tier is declared here once; both search-exclusion and full-load read it from here.

**`store.Hit` (changed — additive).** The search result record gains two fields; all existing fields and their meaning are unchanged:

```
Hit {
  Scope    string    // existing
  Path     string    // existing
  Title    string    // existing
  Excerpts []string  // existing
  Score    float64   // existing
  Category string    // NEW — the entry's category (first path segment); set by the knowledge Set
  Checksum string    // NEW — SHA-256 over the entry's exact raw bytes; the identity key for byte-identical de-dup
}
```

`Category` is empty as produced by the bare store and populated by `Set.Search`; `Checksum` is produced by the store during its file walk. Both serialize as additive JSON fields (`category`, `checksum`), so existing consumers of the search envelope keep working.

**Always-applied entry (changed shape of an existing contract).** The current `Convention{Scope, Path, Content}` generalises so the always-applied reader can return entries from more than one category. The returned record carries `Scope`, `Path`, `Content`, and `Category`, letting a consumer tell a convention from a glossary term. The existing `knowledge conventions` command keeps emitting the conventions-only payload it does today for backward compatibility.

**`knowledge categories` output (new contract).** A structured payload listing each category's `name`, `purpose`, `boundary`, `tier`, and `entryShape` — the registry projected to JSON — consumed by the contribution skill to route a new entry and to steer content to the right category. Carries a `--schema` declaration like the other knowledge subcommands.

**Consolidated lookup answer (new contract, prose-level).** Not a Go type — the shape the consolidation sub-agent returns to the main agent: a single merged answer composed of consolidated points, each citing the source scope(s)/path(s) it was drawn from, with genuine contradictions presented as surfaced conflicts rather than collapsed. The raw per-source candidate list is explicitly *not* part of this contract.

No changes to the `config.Config` structures: tiers live in the category registry, not in `config.yaml` (see `research.md#alternatives-considered-and-rejected`).

## Implementation Detail

**New pattern: a category registry as the tiering source of truth.** Today the always-applied tier is expressed implicitly and in two disconnected places — a literal category name skipped in store search, and a literal category name read by the conventions reader — and the category set itself is a hardcoded list duplicated between init's directory creation and its README generation. The plan replaces all of that with one declarative registry in the knowledge package. Every place that needs to know "what are the categories" or "which categories are always-applied" reads the registry instead of restating a literal. A reader auditing tier behaviour now finds one declaration and follows references out from it, rather than grepping for a magic string. This is the central code-shape change and the thing that makes "re-tier a category by editing one place" true.

**Module boundary shift: category awareness moves up out of the store.** The generic store currently knows one category by name (it skips `conventions` during search). That coupling is removed: the store goes back to being category-agnostic and simply reports what it finds, plus a content checksum it can compute cheaply while it is already reading each file. The knowledge Set — the layer that legitimately understands the path-to-category mapping — becomes the single place that derives a hit's category and applies the always-applied exclusion. This follows the codebase's existing layering (the recursion-lives-in-the-knowledge-layer comment already signals this boundary) and avoids teaching the generic store about a knowledge-domain concept.

**Checksum computation rides the existing file walk.** The search walk already opens and reads every candidate file; the SHA-256 is accumulated over the raw bytes in that same pass, so no second read is introduced. The checksum is the entry's identity for the deterministic de-dup step and is emitted as an ordinary additive result field — no new index, no new storage, nothing persisted.

**De-dup is a two-tier pipeline split along the mechanical/judgement line.** The deterministic tier is plain equality: group candidate hits by checksum, collapse identical ones, union their citations — code a reviewer can read top-to-bottom with no thresholds or tuning. The judgement tier is the consolidation sub-agent, which is *not* code: it is a new agent-orchestration shape expressed in skill prose. The skill's lookup branch stops dumping raw hits and instead runs search, de-dups by checksum, then spawns a sub-agent that reads the surviving bodies and returns a single cited answer. This mirrors the existing planning-agent orchestration the project already relies on, so it introduces a new *use* of an established pattern rather than a brand-new mechanism. The main agent never sees the raw candidate dump, preserving its working context.

**Skill UX: definitions are pulled at the moment of contribution.** Rather than expecting a contributor (human or assistant) to already know the category layout, the contribute branch fetches the registry's definitions just-in-time and routes on them — including steering over-long content away from the glossary. This is a behavioural change to a prose playbook, not new control flow in Go; the categories command exists to feed it.

**Following existing conventions elsewhere.** The new `categories` subcommand and the additive schema fields follow the established `knowledge` command shape (cobra subcommand + JSON envelope + `--schema` mode). Init's scaffold continues to write category directories and READMEs exactly as before — only the *source* of the names and the README text changes from hardcoded literals to the registry. The generalised always-applied reader keeps the conventions reader's fan-across-scopes behaviour and simply widens which categories it covers. Tests follow the package's existing testify/`t.TempDir()` style.

## Dependencies

**Runtime / internal packages**

- **`internal/store`** — the generic file store and in-process search. Changed: drops the hardcoded `conventions` skip, adds `Category` + `Checksum` to its hit type, and computes the checksum during the existing file walk.
- **`internal/knowledge`** — the multi-source Set. Changed: hosts the new category registry, tags hits with category, applies registry-driven always-applied exclusion in search, and generalises the conventions reader into an always-applied reader.
- **`internal/config`** — project configuration. No change: tiers deliberately live in the category registry, not in `config.yaml` (see `research.md#alternatives-considered-and-rejected`). Listed to record that it was considered and intentionally left untouched.
- **`internal/project`** — init/scaffold. Changed: sources category directory names and README definitions from the registry instead of hardcoded literals.
- **`internal/agent`** — skill rendering (mustache `{{command}}` substitution). No code change expected; it already renders the spek-knowledge skill template, which gains new content.
- **`cmd` (knowledge command)** — the CLI surface. Changed: new `categories` subcommand, additive schema fields, an always-applied loader entry point; existing `conventions` subcommand preserved.

**External libraries**

- **Go standard library `crypto/sha256`** — provides the checksum. Standard library, no new module dependency, no network — satisfies the self-contained-binary and no-external-tool constraints.
- **`cobra` (CLI) and the project's JSON output envelope** — already in use; the new subcommand follows the existing pattern. No version change.
- No new third-party dependency is introduced. The de-duplication and consolidation step adds no library and no external service: the deterministic tier is standard-library hashing, and the judgement tier reuses the assistant's existing sub-agent orchestration capability.

**Templates / assets**

- **`templates/skills/workflows/spek-knowledge`** — the skill playbook; gains category-routing and consolidation flows. Source of truth that renders into the project's `.claude/skills`.
- **Workflow step templates that load conventions** — switch to loading the full always-applied set. Changed content, no new mechanism.
- **New documentation asset** (`docs/knowledge-base.md`) — a new file the plan creates; depends on nothing.

**Planning dependencies (prior specs / plans)**

- **Specs 000026 (ripgrep-replace) / 000027 (document-level-search)** — already landed; they established the in-process `FileStore` search and the one-hit-per-document ranked model this work extends. No further work needed from them; this plan builds directly on that search as-is.
- No prior plan must land before this one — all dependencies are already in `main`.

## Testing Approach

The bulk of coverage is **Go unit tests** in the existing testify/`t.TempDir()` style, concentrated on the layers that carry the deterministic, machine-checkable behaviour: the category registry, the knowledge Set (category tagging, always-applied exclusion, the generalised always-applied reader), the store search (checksum computation), and project init (scaffolding from the registry). **CLI/contract tests** cover the `knowledge` command envelopes and `--schema` output for the new `categories` subcommand and the additive `category`/`checksum` fields. **Regression tests** lock the preserved behaviours the spec's constraints require: existing search ranking, read, write, list, and the backward-compatible `knowledge conventions` payload must be unchanged. The judgement-tier consolidation sub-agent is deliberately **not** unit-tested — it is prose-driven LLM behaviour with no deterministic oracle; its guarantees are verified manually (see metric mapping).

**Load-bearing assertions, in plain language:**

- Every category carries a real purpose definition, not a circular placeholder; the glossary and decisions categories exist with the correct tiers.
- A search hit reports the category matching the entry's actual category, and that label is correct for entries in every category.
- Content in an always-applied category never appears as a search hit; content in a looked-up category appears only when a query matches it.
- Re-tiering is a one-place change: flipping a category to always-applied in the registry simultaneously makes the always-applied reader load it *and* makes search stop returning it — asserted together so the two behaviours can never drift apart (this is the spec's "single, self-consistent action" requirement).
- Two byte-identical entries in different scopes collapse to a single result keyed on checksum; two entries that differ by even one byte do not collapse. The checksum oracle is an independently known SHA-256 of a fixed fixture, not a value recomputed through the production code path, so the test cannot rubber-stamp a wrong implementation.
- The always-applied reader returns conventions *and* glossary content, each tagged with its category, while the legacy `conventions` command still returns only conventions.
- A freshly initialised project contains the glossary and decisions directories with their real README definitions.

These slot directly into the existing `internal/store`, `internal/knowledge`, `internal/project`, and `cmd` test suites; no new test framework or harness is introduced. Tests render fixtures into the test's own scratch directory via the production paths rather than reading the repo's live knowledge tree.

**Deliberate gaps:** no automated test asserts the *quality* of semantic merging, conflict surfacing, or category-routing judgement — these are sub-agent/skill-prose behaviours without a deterministic oracle, and are classified as manual below. No new integration/e2e harness is added; the command contracts are exercised by the CLI tests.

**Success-metric verification mapping** (every metric in the spec is carried here):

1. *New entries filed in the correct category without correction.* Partly **behavioural** — a unit/CLI test asserts the `categories` command surfaces each category's definition (the routing input) and that an entry's path-derived category is reported correctly. The "mis-filing is rare in practice" quality claim is **Manual — captured in the implementation test plan** (depends on real assistant routing judgement over time).
2. *Lookups return consolidated answers with citations, not overlapping copies.* The mechanical half is **behavioural**: byte-identical candidates collapse by checksum, and the lookup's exposed output is the consolidated answer, not the raw per-source hit list. The merge-quality and citation-faithfulness of the sub-agent's answer is **Manual — captured in the implementation test plan**.
3. *Always-applied tier stays compact (glossary small, no essay-length growth).* **Manual — captured in the implementation test plan** — the glossary shape rule is enforced by definition + skill steering (judgement), and "stays compact over time" is an observed property, not a deterministic assertion.
4. *Source disagreements are visible (conflict surfaced, not silently resolved).* **Manual — captured in the implementation test plan** — conflict detection is sub-agent judgement over full bodies, with no deterministic oracle.
5. *Maintainers can explain the knowledge base from the documentation alone.* **Manual — captured in the implementation test plan** — verified by a reader walking the docs without source access; documentation completeness is not machine-checkable.

## Milestones & Phases

### Milestone 1 — A self-explanatory category model

**What changes.** A freshly initialised project gains two new knowledge categories — a glossary for shared vocabulary and a decisions log for recording the reasoning behind choices — alongside the existing categories. Every category, old and new, now carries a real definition stating its purpose, what belongs in it, what looks similar but belongs elsewhere, its retrieval tier, and the expected shape of its entries — replacing today's circular "this directory contains X documentation" placeholders. A contributor (or the assistant) can ask the tool for these definitions and so knows exactly where a new entry belongs. This milestone establishes the single declared source of truth for the category model that the later milestones build on.

#### - [x] Phase 1.1: Category registry and tiers

The single source of truth for the category model. A new registry declares every category once — its purpose, boundary, retrieval tier, and expected entry shape — and exposes the derived set of always-applied categories. This includes the final, real definitions for the existing categories and the two new ones (glossary, decisions), replacing the circular placeholders. Nothing else can be built until this exists.

*Technical detail:* [context.md#phase-11](./context.md#phase-11-category-registry-and-tiers)

**Acceptance criteria**:
- [x] Every category (conventions, architecture, gotchas, learnings, glossary, decisions) has a purpose-stating definition with a boundary, a declared tier, and an entry shape.
- [x] The glossary is declared always-applied; the decisions category is declared looked-up.
- [x] The always-applied set is derived from the per-category tier declaration, stated in exactly one place.

#### - [x] Phase 1.2: Initialise projects from the registry

Project initialisation stops hardcoding the category list and the placeholder README text and instead scaffolds each category's directory and writes its definition from the registry. A freshly initialised project now contains the glossary and decisions categories with real definitions.

*Technical detail:* [context.md#phase-12](./context.md#phase-12-initialise-projects-from-the-registry)

**Acceptance criteria**:
- [x] A freshly initialised project contains directories and README definitions for every category in the registry, including glossary and decisions.
- [x] No category README contains the old circular "this directory contains X documentation" text.

#### - [x] Phase 1.3: Expose category definitions to contributors

A new knowledge subcommand emits the registry's definitions as structured output, so the contribution flow can present them at the moment an entry is filed. Follows the existing knowledge-command shape, including a schema declaration.

*Technical detail:* [context.md#phase-13](./context.md#phase-13-expose-category-definitions-to-contributors)

**Acceptance criteria**:
- [x] A categories command returns every category's purpose, boundary, tier, and entry shape.
- [x] The command declares its output schema like the other knowledge subcommands.

### Milestone 2 — Tier-aware retrieval with labelled results

**What changes.** Knowledge now behaves according to each category's declared tier. Always-applied categories — conventions and the new glossary — are present in the assistant's context on every task and are never also returned by search, so nothing is surfaced twice; looked-up categories are returned only when a query matches. Every search result is tagged with the kind of knowledge it came from, so a consumer can tell a warning from a rule from a definition. Moving a category into the always-applied tier is a single change in one place that both loads it everywhere and removes it from search at once — it can no longer end up half-applied. This replaces the previous arrangement where the one always-applied category was special-cased independently in two disconnected spots.

#### - [x] Phase 2.1: Make the store category-agnostic and label-ready

The generic store stops special-casing any category by name: the hardcoded exclusion of the conventions directory from search is removed, and the search result gains an additive category field (left empty by the store). This unwinds one of the two places the always-applied tier was hardcoded.

*Technical detail:* [context.md#phase-21](./context.md#phase-21-make-the-store-category-agnostic-and-label-ready)

**Acceptance criteria**:
- [x] The store no longer skips any directory by name during search.
- [x] The search result type carries a category field, serialized additively so existing consumers are unaffected.

#### - [x] Phase 2.2: Tier behaviour and category tagging in the knowledge layer

The knowledge layer becomes the single place that understands tiers: search tags each hit with its category and drops hits belonging to always-applied categories, both driven by the registry; and the conventions reader generalises into an always-applied reader that returns every always-applied category's content tagged with its category. The legacy conventions reader behaviour is preserved.

*Technical detail:* [context.md#phase-22](./context.md#phase-22-tier-behaviour-and-category-tagging-in-the-knowledge-layer)

**Acceptance criteria**:
- [x] A search hit reports the category matching the entry's actual category.
- [x] Content in an always-applied category never appears in search; content in a looked-up category appears only on a matching query.
- [x] Re-tiering one category in the registry both adds it to the always-applied load and removes it from search, shown together.
- [x] The always-applied reader returns conventions and glossary content, each tagged; the legacy conventions output is unchanged.

#### - [x] Phase 2.3: Surface the label and load the full always-applied set

The search command output exposes the category label, and the workflow steps that load conventions today switch to loading the full always-applied set so glossary content is present on every task without being searched for.

*Technical detail:* [context.md#phase-23](./context.md#phase-23-surface-the-label-and-load-the-full-always-applied-set)

**Acceptance criteria**:
- [x] The search command result and its schema include the per-result category label.
- [x] The steps that previously loaded only conventions now load the full always-applied set, including the glossary.
- [x] Existing search ranking, read, write, list, and conventions outputs are unchanged.

### Milestone 3 — Consolidated, de-duplicated lookups

**What changes.** Asking the knowledge base a question now yields a single consolidated answer instead of a pile of overlapping or identical copies. Byte-identical copies of an entry held in more than one source are removed deterministically; entries that say the same thing in different words are merged into one point; when a more specific source and a less specific one cover the same item, the more specific one takes precedence; and a genuine disagreement between sources is surfaced rather than silently dropped. The answer cites the sources it drew from, and the raw per-source hit list is no longer the output. Contributions are also routed to the right category at the moment they are made, with over-long content steered out of the glossary.

#### - [x] Phase 3.1: Content checksum on results

Each search result gains a checksum computed over the entry's exact raw bytes, accumulated during the search's existing file read so no second read is added. This is the deterministic identity key for byte-identical de-duplication.

*Technical detail:* [context.md#phase-31](./context.md#phase-31-content-checksum-on-results)

**Acceptance criteria**:
- [x] Two byte-identical entries produce the same checksum; entries differing by any byte produce different checksums.
- [x] The checksum is present on the search result and in its schema, serialized additively.

#### - [x] Phase 3.2: Consolidated lookup with de-dup and precedence

The lookup flow stops dumping raw hits. It groups candidates by checksum to remove byte-identical copies deterministically, then a consolidation sub-agent reads the surviving bodies, merges differently-worded equivalents, applies layered precedence (most-specific source wins), surfaces genuine conflicts, and returns one cited answer.

*Technical detail:* [context.md#phase-32](./context.md#phase-32-consolidated-lookup-with-de-dup-and-precedence)

**Acceptance criteria**:
- [x] A query whose knowledge exists byte-for-byte in two sources returns it once, with both sources cited.
- [x] Differently-worded equivalents present as a single merged, cited point; a genuine conflict is reported rather than silently resolved.
- [x] The lookup output is the consolidated answer; the raw per-source hit list is not exposed as the result.

#### - [x] Phase 3.3: Route contributions to the right category

The contribution flow loads the category definitions at the moment of filing and routes the entry to the category whose purpose matches, steering over-long content out of the glossary toward a more suitable category.

*Technical detail:* [context.md#phase-33](./context.md#phase-33-route-contributions-to-the-right-category)

**Acceptance criteria**:
- [x] When contributing, the category definitions are presented and an entry is filed in the category whose purpose it matches.
- [x] A short term-and-gloss entry is accepted into the glossary; a long multi-paragraph entry is steered to a different category.

### Milestone 4 — Durable knowledge-base documentation

**What changes.** The knowledge base's design is written down in durable project documentation so a maintainer new to the project can understand it without reading the source: the category model, the two retrieval tiers, how search behaves including the per-result category label, how de-duplication and consolidation work, why the de-dup layer is exact rather than fuzzy, and how layered source precedence is resolved. This is primarily a documentation deliverable; it is worth its own milestone because the spec makes "maintainers can explain the knowledge base from the documentation alone" an explicit success metric, and the document can only be finalised once the preceding milestones fix the behaviour it describes.

#### - [x] Phase 4.1: Write the knowledge-base documentation

A durable document describes the category model, the two retrieval tiers, search behaviour including the per-result category label, how de-duplication and consolidation work, why the de-dup layer is exact rather than fuzzy, and how layered source precedence is resolved — written for a reader new to the project.

*Technical detail:* [context.md#phase-41](./context.md#phase-41-write-the-knowledge-base-documentation)

**Acceptance criteria**:
- [x] The document covers the category model, the two tiers, search and the category label, de-duplication and consolidation, the exact-checksum rationale, and layered precedence.
- [x] A reader unfamiliar with the source can describe how the knowledge base works from the document alone.

## Open Questions

- **Does the executing agent support spawning a consolidation sub-agent?** The lookup flow's consolidation step assumes the assistant running the spek-knowledge skill can spawn a sub-agent for context isolation — the same orchestration capability the planning/implementation flows already rely on. Spektacular is agent-agnostic (Claude / Bob / Codex), so whether a given agent has this capability cannot be settled by reading the Spektacular source; it is a property of the runtime agent. *What the implementer should do:* implement the skill prose so the de-dup + consolidation is described as a sub-agent step, but if the target agent lacks sub-agent orchestration, fall back to running the consolidation inline in the main context (functionally identical output, weaker context isolation). Do not block on this; note which path was taken. STOP and ask the user only if neither path can produce a single consolidated answer.

There are no other implementation-time-only uncertainties. All design decisions — the layered precedence model, the exact (non-fuzzy) checksum, the registry-as-source-of-truth, the additive hit fields, the category-routing-at-contribution, the backward-compatible conventions command, and the documentation deliverable — were resolved during planning.

## Out of Scope

- **Category-scoped search filtering.** Each result is tagged with its category, but restricting a query to a single category (e.g. "search only the gotchas") is not built here. The label is the enabler for that future capability, not the capability itself. (Spec § Non-Goals; a later plan can build the filter on top of the label.)
- **Semantic / embedding retrieval index.** No vector index or embedding-based retrieval is introduced. Merging differently-worded but equivalent entries is done by the lookup sub-agent's judgement at lookup time, not by a similarity index. This was reaffirmed during the architecture step when choosing an exact (non-fuzzy) checksum over a SimHash/MinHash similarity fingerprint. (Spec § Non-Goals; architecture decision — see `research.md#alternatives-considered-and-rejected`.)
- **Retroactive re-classification or rewriting of existing entries.** Existing entries are not moved, re-categorised, or rewritten to fit the new model, and the glossary's term-and-gloss shape rule is not applied to entries that already exist. The new categories, definitions, and rules apply to contributions going forward. (Spec § Non-Goals.)
- **A new `knowledge lookup` CLI command.** Considered during the architecture step and deliberately dropped: the checksum rides on the existing search result, so the deterministic de-dup needs no new command and the consolidation lives in the skill + sub-agent. (Architecture decision.)
- **Tiers in `config.yaml`.** Making the always-applied set a configuration list was considered and rejected in favour of the category registry as the single source of truth, so a category's tier sits beside its definition rather than in a separate file. (See `research.md#alternatives-considered-and-rejected`.)
- **Normalized or fuzzy de-duplication of near-identical entries.** Only byte-identical copies are removed mechanically; entries that differ even by whitespace flow to the judgement tier. Normalization rules were explicitly declined in favour of strict byte-identity. (Architecture decision.)

## Changelog

### 2026-06-17 — Phase 1.1: Category registry and tiers

**What was done**: Added a new category registry as the single source of truth for the knowledge category model. `internal/knowledge/category.go` declares a `Tier` type (`TierAlwaysApplied`/`TierLookedUp`), a `Category{Name, Purpose, Boundary, Tier, EntryShape}` record, the canonical ordered `Categories` list with real definition wording for all six categories (conventions, glossary, architecture, gotchas, learnings, decisions), and the accessors `AlwaysApplied()` and `CategoryByName()`. glossary is always-applied; decisions is looked-up.

**Deviations**: None.

**Files changed**:
- `internal/knowledge/category.go`
- `internal/knowledge/category_test.go`

**Discoveries**: The `Category` JSON tags are `name`/`purpose`/`boundary`/`tier`/`entryShape` — Phase 1.3's `knowledge categories` output schema must mirror these exactly. `AlwaysApplied()` returns names in registry order (`conventions`, `glossary`); downstream phases (2.2 search exclusion + always-applied reader) consume this single derivation, so re-tiering stays one-place. No integration points were touched yet — the registry is consumed starting in 1.2.

### 2026-06-17 — Phase 1.2: Initialise projects from the registry

**What was done**: `internal/project/init.go` now drives knowledge scaffolding from the category registry instead of hardcoded literals. It creates a directory under `.spektacular/knowledge/` for every `knowledge.Categories` entry (so fresh projects gain `glossary` and `decisions` alongside the existing four) and writes each category's README rendered from its registry definition (title, tier, purpose, boundary, entry shape), replacing the circular "This directory contains X documentation" placeholder.

**Deviations**: None.

**Files changed**:
- `internal/project/init.go`
- `internal/project/init_test.go`

**Discoveries**: `internal/project` now imports `internal/knowledge` (no import cycle — knowledge does not import project). The README format is `# <Title>` then `**Tier:** / **Purpose:** / **Belongs elsewhere:** / **Entry shape:**` blocks; tests assert presence of `**Purpose:**` and absence of `"This directory contains"`. `init.go` keeps the deprecated `strings.Title` with its existing `//nolint:staticcheck` for capitalising the title — lint stays green.

### 2026-06-17 — Phase 1.3: Expose category definitions to contributors

**What was done**: Added a `knowledge categories` CLI subcommand that emits the category registry as a structured `{"categories": [...]}` payload (each entry carrying name, purpose, boundary, tier, entryShape), so the contribution flow can present the definitions at filing time. It follows the existing knowledge-command shape, including a `--schema` declaration.

**Deviations**: None.

**Files changed**:
- `cmd/knowledge.go`
- `cmd/knowledge_test.go`

**Discoveries**: The command needs no `knowledge.Set` — it returns the package-level `knowledge.Categories` directly, unlike `sources`/`conventions` which fan out over configured stores. The output schema follows the minimal array idiom (`{"categories": {Type:"array"}}`) used by `sources`/`conventions`/`list`, not the per-field item schema of `search`. Phase 3.3 (contribute routing) will consume this command's output.

### 2026-06-17 — Phase 2.1: Make the store category-agnostic and label-ready

**What was done**: Removed the hardcoded exclusion of `conventions` directories from `FileStore` search so the generic store no longer special-cases any category by name, and updated its package comment to state the store is category-agnostic (tier exclusion now lives in the knowledge layer). Added an additive `Category` field (json `category`) to `store.Hit`, left empty by the store for the knowledge layer to populate.

**Deviations**: None.

**Files changed**:
- `internal/store/store.go`
- `internal/store/search.go`
- `internal/store/search_test.go`

**Discoveries**: Two existing store tests depended on the removed conventions-exclusion: `TestSearch_CaseInsensitiveAndExcludesConventions` (renamed to `...IncludesAllDirectories`, now asserts `conventions/style.md` IS returned) and `TestSearch_SingleWordMatchesSameFiles` (its hand-maintained expected-paths list gained `conventions/style.md`). The `conventions/`-skip in the store is gone; the equivalent always-applied exclusion is now Phase 2.2's job in `knowledge.Set.Search`, where the registry lives.

### 2026-06-17 — Phase 2.2: Tier behaviour and category tagging in the knowledge layer

**What was done**: Made the knowledge `Set` the single place that understands tiers. `Set.Search` now tags each hit's `Category` (first path segment) and drops hits whose category is in the registry's always-applied set, so always-applied content never appears in search while looked-up content does. Generalised the conventions reader into `Set.AlwaysAppliedEntries`, which returns every always-applied category's content (conventions and glossary) tagged with its category; `Set.Conventions` is preserved as a conventions-only view so the `knowledge conventions` JSON is byte-for-byte unchanged.

**Deviations**: The generalised reader method is named `AlwaysAppliedEntries` (and returns a new `AlwaysAppliedEntry` type), not `AlwaysApplied` as research.md sketched, to avoid colliding with the package-level `AlwaysApplied()` accessor. No behavioural deviation from the plan.

**Files changed**:
- `internal/knowledge/set.go`
- `internal/knowledge/set_test.go`

**Discoveries**: `Category` is deliberately NOT added to the `Convention` struct — it lives on the new `AlwaysAppliedEntry` — which is what keeps the legacy `knowledge conventions` payload identical. `Search`, `AlwaysApplied()`, and `AlwaysAppliedEntries` all read the registry `Categories` var live each call, which is what makes the one-place re-tier test work: mutating `Categories` flips both load and search-exclusion together. A private `Set.readCategories([]string)` helper backs both `Conventions` and `AlwaysAppliedEntries` (one read loop, no duplication). Phase 2.3 will surface the `category` field in the search schema and add the always-applied loader command + template edits that call `AlwaysAppliedEntries`.

### 2026-06-17 — Phase 2.3: Surface the label and load the full always-applied set

**What was done**: Added the per-result `category` field to the `knowledge search` output schema, and a new `knowledge always-applied` CLI command that returns every always-applied entry (conventions and glossary) across scopes, each tagged with scope, path, content, and category. Switched the plan workflow's discovery step to load the full always-applied set once via that command (and to note search hits now carry a category label and that the always-applied categories are excluded from search); updated the architecture step's back-reference accordingly.

**Deviations**: The plan/context listed five step templates to update, but only two actually perform a knowledge-base conventions load — `templates/steps/plan/02-discovery.md` (the load) and `templates/steps/plan/03-architecture.md` (a back-reference to it). The other three (`08-testing_approach.md`, `implement/04-test.md`, `13-assemble.md`) use "conventions" to mean testing-framework conventions or the plan's assembled `## Conventions` section, not a KB load, so they were intentionally left unchanged. Confirmed with the user, who noted the always-applied set only needs loading once (discovery) since it then stays in context.

**Files changed**:
- `cmd/knowledge.go`
- `cmd/knowledge_test.go`
- `templates/steps/plan/02-discovery.md`
- `templates/steps/plan/03-architecture.md`

**Discoveries**: Templates invoke CLI commands (not Go functions), so exposing the always-applied set required a real subcommand (`always-applied`) backed by `Set.AlwaysAppliedEntries`; its result key is `entries`. The `knowledge conventions` command is retained for backward compatibility but no template calls it any more. The repo's own live `.spektacular/knowledge/` tree still holds the pre-feature circular READMEs — existing-entry migration is an explicit Non-Goal, so they are left as-is. Phase 3.x will add the checksum field (3.1) and rewrite the spek-knowledge skill's lookup/contribute prose (3.2/3.3) which consumes `category` and the new `categories`/`always-applied` commands.

### 2026-06-17 — Phase 3.1: Content checksum on results

**What was done**: Added an additive `Checksum` field (hex SHA-256 over the entry's exact raw bytes) to `store.Hit`, computed inside the existing search file walk so no second read is added, and surfaced it in the `knowledge search` output schema. This is the deterministic identity key the lookup flow will use to collapse byte-identical copies.

**Deviations**: None.

**Files changed**:
- `internal/store/store.go`
- `internal/store/search.go`
- `cmd/knowledge.go`
- `internal/store/search_test.go`

**Discoveries**: The hash is computed in `scanFile` by writing the already-read binary-sniff bytes to the hasher up front, then teeing the rest of the file through it via `io.TeeReader`, and finally draining any bytes the line scanner left unread (`io.Copy(io.Discard, tee)`) so an over-long line can't truncate the digest. Binary files short-circuit before hashing and get an empty checksum, but they never produce a hit anyway. The test oracle is a hand-written SHA-256 hex literal computed independently via `sha256sum` (`50aeeec5…` for the bytes `# Title\n\nneedle here\n`), not recomputed through the production path. Phase 3.2 consumes this `checksum` in the skill's lookup branch to group byte-identical candidates before the consolidation sub-agent runs.

### 2026-06-17 — Phase 3.2: Consolidated lookup with de-dup and precedence

**What was done**: Rewrote the `spek-knowledge` skill's "Intent: lookup" prose so a lookup no longer dumps raw hits. The new flow is: run `knowledge search`; deterministically group hits by `checksum` to collapse byte-identical copies (unioning their scope/path citations); then delegate the unique candidates to a consolidation sub-agent that reads the bodies, classifies relationships (equivalent → merge, refinement → most-specific scope wins, contradiction → surface, distinct → keep both), applies layered precedence (project › team › global), and returns a single source-cited answer that is presented to the user. The raw per-source hit list is never the output.

**Deviations**: None functionally. The skill documents a sub-agent path plus an explicit inline fallback for executing agents that cannot spawn sub-agents (resolving the plan's one open question — Claude Code supports orchestration, so the sub-agent path is primary). The behaviours here are judgement-tier (semantic merge, conflict surfacing) with no deterministic oracle, so they have no Go unit tests — they are verified manually per the implementation test plan.

**Files changed**:
- `templates/skills/workflows/spek-knowledge/SKILL.md`

**Discoveries**: The edit is to the skill TEMPLATE; the rendered `.claude/skills/.../SKILL.md` is regenerated from it on `init`, so it was not hand-edited. The template uses only the `{{command}}` mustache placeholder (verified no stray tags). Phase 3.3 edits the SAME file's "Intent: contribute" section next — sequenced after this to avoid edit conflicts.

### 2026-06-17 — Phase 3.3: Route contributions to the right category

**What was done**: Added a category-routing step to the `spek-knowledge` skill's "Intent: contribute" prose. Before staging, the flow now loads the category definitions with `knowledge categories`, picks the category whose Purpose matches and whose Boundary does not exclude the entry, and steers over-long or multi-paragraph content out of the always-applied glossary toward a more fitting category. The entry path is now `<category>/<slug>.md`, and the confirm/write/cleanup steps were renumbered to follow.

**Deviations**: None. Routing quality (correct-category judgement, glossary steering) is judgement-tier with no deterministic oracle, so it has no Go unit tests — verified manually per the implementation test plan.

**Files changed**:
- `templates/skills/workflows/spek-knowledge/SKILL.md`

**Discoveries**: The contribute flow already called `knowledge sources` for scope selection; category routing is additive alongside it (sources → scope, categories → path prefix). With both lookup (3.2) and contribute (3.3) now rewritten in the same template, Milestone 3 is complete; the only remaining work is the Milestone 4 documentation (`docs/knowledge-base.md`), which describes the now-final behaviour.

### 2026-06-17 — Phase 4.1: Write the knowledge-base documentation

**What was done**: Created `docs/knowledge-base.md` (a new `docs/` directory), a durable, standalone description of the knowledge base for a reader new to the project. It covers the category model and the six categories' definitions, the single-registry source of truth, the two retrieval tiers and the one-place re-tiering property, search behaviour including the per-result category label and checksum, the two-stage de-duplication (exact checksum) and consolidation (sub-agent semantic merge), the exact-vs-fuzzy rationale, layered source precedence with conflict surfacing, contribution routing, and the full `spektacular knowledge` command reference. Linked it from the README's Knowledge section.

**Deviations**: Scope-limited the README change to a single pointer to the new doc rather than rewriting the README's existing `## Knowledge` section. That section still describes the pre-feature model (conventions as the only always-applied tier, the older command table) and would benefit from a refresh, but a full rewrite is beyond Phase 4.1's stated scope (new doc + optional link); `docs/knowledge-base.md` is now the authoritative reference. Documentation comprehension is a manual success metric, verified by reading, not a Go test.

**Files changed**:
- `docs/knowledge-base.md`
- `README.md`

**Discoveries**: The doc is human-facing prose, so it spells out real `spektacular knowledge ...` commands rather than the `{{command}}` placeholder used in skill templates. Follow-up worth considering (out of scope here): modernise the README's `## Knowledge` section body and its command table to match the new category/tier model. This completes all four milestones of the plan.
