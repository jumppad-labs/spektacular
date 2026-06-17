# Feature: 000028_knowledge-base-categories-tiers-and-dedup

## Overview

This work makes the project's knowledge base clearer, more trustworthy, and easier for an AI assistant to use correctly. Knowledge is organised into well-defined categories, and each category is given a clear purpose so that both people and the assistant know exactly what belongs where. Some kinds of knowledge — the rules and shared vocabulary a team always wants applied — are made available to the assistant on every task, while the larger body of reference material is looked up only when relevant, keeping the assistant focused and its attention uncluttered.

The work also solves a trust problem: when the same fact lives in more than one place, the assistant should not be handed several overlapping or contradictory copies. Instead, lookups are gathered together, redundant copies are removed, and a single, clear answer is returned — with the most specific source taking precedence when sources disagree. Search results are tagged with the kind of knowledge they came from, so a warning reads as a warning and a rule reads as a rule.

Everyone working in the project benefits: contributors get a knowledge base that stays tidy and self-explanatory, and the assistant gives more accurate, less cluttered answers. Finally, how the knowledge base works is written down in durable documentation so the design is understood and maintained over time.

## Requirements

**Category model**

- The knowledge base must organise entries into categories that each have a clearly defined, distinct purpose, so a contributor can determine the correct category for a new entry without guessing.
- The system must include a category for shared domain vocabulary (a glossary).
- The system must include a category for recording decisions and the reasoning behind them.
- Each category must carry a definition — its purpose, what belongs in it, what looks similar but belongs elsewhere, which retrieval behaviour it follows, and the expected shape of its entries — and that definition must be presented to the assistant at the moment it contributes a new entry, so entries are routed to the correct category.
- The final wording of every category's definition must be produced as part of this work, covering all categories (the existing ones and the newly added glossary and decisions), and must replace the current circular placeholder descriptions.

**Retrieval behaviour (two tiers)**

- Every category must follow exactly one of two retrieval behaviours: "always-applied" (its content is made available to the assistant on every task) or "looked-up" (its content is retrieved only when relevant to a query).
- The glossary must be an always-applied category.
- The decisions category must be a looked-up category.
- Declaring that a category is always-applied must be a single, self-consistent action: a category must not be able to end up partially applied (for example, loaded on every task but still also returned by search, or the reverse).
- Glossary entries must be short definitions (a term and a brief gloss). Longer explanatory content must belong to a different category, so that always-applied content stays compact.

**Search**

- Each search result must identify the kind of knowledge it came from, so a consumer can distinguish a warning from a rule from a design fact from an empirical learning.

**De-duplication and consolidation**

- A knowledge lookup must return a single consolidated answer rather than multiple overlapping or identical copies drawn from different sources.
- Identical copies of the same entry must be removed from a lookup result.
- Entries that express the same knowledge in different words must be merged in the answer rather than returned separately.
- A lookup must yield only the consolidated answer; the unconsolidated set of individual per-source results must not be exposed as the lookup's output.
- The consolidated answer must cite the sources it was drawn from.

**Source precedence**

- When two sources hold the same item with differing content, the most specific source must take precedence.
- A genuine disagreement between sources (as opposed to a redundant copy) must be surfaced rather than silently dropped.

**Documentation**

- The knowledge base's function and core concepts must be captured in durable project documentation, covering the category model, the two retrieval behaviours, how search behaves, how de-duplication and consolidation work, and how source precedence is resolved.

## Constraints

- Must build on the existing knowledge base rather than replacing it: the current categories and the stored entries within them must continue to work unchanged, and the existing always-applied category must keep its current behaviour.
- Must preserve the existing read, write, search, and listing behaviour of the knowledge base. Adding a category label to search results must be additive and must not break existing consumers of those results.
- Must operate as a single self-contained binary with no separate server process and no dependency on an external search tool; search and de-duplication run in-process. (The project has already removed its external search-tool dependency and must not reintroduce one.)
- The de-duplication and consolidation step must not require network access or an external service beyond the assistant capabilities the project already uses.
- Removing byte-identical duplicates must be deterministic and not rely on assistant judgement; only the merging of differently-worded but equivalent entries may use assistant judgement.

## Acceptance Criteria

**Category model**

- A freshly initialised project contains the glossary and decisions categories alongside the existing categories, each with a definition that states its purpose rather than a circular placeholder.
- When the assistant is asked to contribute a new entry, the category definitions are available to it at that moment, and an entry whose content matches one category's purpose is placed in that category and not another.
- A glossary entry consisting of a term and a brief definition is accepted; an attempt to file a long multi-paragraph explanation as a glossary entry is steered to a different category.

**Retrieval behaviour (two tiers)**

- The full content of every always-applied category (including the glossary) is present in the assistant's context for a task without the assistant having to search for it.
- Content in an always-applied category is not also returned as a search result for a matching query (it is not surfaced twice).
- Content in a looked-up category is returned only when a query matches it, and is not loaded wholesale on every task.
- Adding one more category to the always-applied set, then running a query that matches an entry in it, shows that entry is now always-applied and no longer appears in search results — achieved by changing the configuration in one place.

**Search**

- A search result for any matching entry reports which category (kind of knowledge) it came from, and that label correctly reflects the entry's category.

**De-duplication and consolidation**

- When the same entry exists byte-for-byte in two sources, a lookup returns it once, not twice.
- When two sources hold entries that state the same thing in different words, the lookup answer presents that knowledge once, as a single merged point.
- A lookup answer cites the source(s) it was drawn from.
- A lookup's output is a single consolidated answer; the raw list of individual per-source hits is not exposed as the result.

**Source precedence**

- When two sources hold the same item with differing content, the answer reflects the most specific source's version.
- When two sources genuinely disagree on the same item, the disagreement is reported in the answer rather than one side being silently discarded.

**Documentation**

- Durable project documentation exists that describes the category model, the two retrieval behaviours, how search behaves (including the per-result category label), how de-duplication and consolidation work, and how source precedence is resolved; a reader new to the project can understand the knowledge base from it without reading the source code.

## Technical Approach

The following is non-binding direction discussed so far; the plan workflow owns the detailed design and may adopt, adapt, or replace any of it.

- **Retrieval tiers.** Prefer a single declared set of always-applied categories as the source of truth, so a category's tier is stated once and its behaviours follow from that one declaration, rather than being asserted independently in more than one place as today.
- **Category on results.** A result's category can be derived from where the entry already lives, so tagging results with their category is expected to be a lightweight, additive change rather than a new index.
- **De-duplication.** Prefer placing consolidation behind a dedicated knowledge lookup agent that reads the candidate entries and returns a single consolidated answer to the caller, rather than attempting it during search. The reason it belongs in an agent: duplication can only be judged once full content is read, and search returns lightweight candidates without content.
- **Source precedence — open decision.** Whether scopes are *layered* (a more-specific scope overrides a less-specific one for the same item) or *additive* (every scope contributes; only identical copies are removed) is not yet decided, and it affects both de-duplication and precedence behaviour. The plan should resolve this explicitly before implementation.
- **Category definitions.** Each category's definition (purpose, boundary, tier, entry shape) should replace today's placeholder descriptions and be surfaced to the assistant at contribution time.

## Success Metrics

- New entries are filed in the correct category without correction: when the assistant contributes knowledge, the category it chooses matches the category's stated purpose, and mis-filing is rare in practice.
- Lookups return consolidated answers: a question whose knowledge exists in more than one source yields a single merged answer with citations, not a list of overlapping copies.
- The always-applied tier stays compact: glossary and other always-applied content remains small enough that loading it on every task does not crowd out the assistant's working context, and it does not grow unbounded with essay-length entries.
- Source disagreements are visible: when two sources conflict, the conflict shows up in the answer rather than being silently resolved.
- Maintainers can explain the knowledge base from the documentation alone: someone new to the project can describe the category model, the two tiers, search behaviour, de-duplication, and precedence without reading the source.

## Non-Goals

- Category-scoped search filtering is out of scope. Each result is tagged with its category, but the ability to restrict a query to a single category (e.g. "search only the gotchas") is not part of this work; the label is the enabler for that future capability, not the capability itself.
- Building a semantic/embedding retrieval index is out of scope. Merging differently-worded but equivalent entries is done by the lookup agent's judgement at lookup time; no vector index or embedding-based retrieval is introduced.
- Retroactively re-classifying or rewriting existing entries to satisfy the new category model is out of scope, including applying the glossary shape rule to entries that already exist. The new categories, definitions, and rules apply to contributions going forward.
