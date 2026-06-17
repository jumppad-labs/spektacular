# The Knowledge Base

Knowledge is the accumulated know-how a project draws on when planning — the rules
it commits to, the shared vocabulary it speaks, how the system is built, the traps
to avoid, what past work taught, and why decisions were made. It is a
**planning-time input**: the planning agent reads it while producing a plan, and
the relevant parts are written into the plan itself. The implement workflow then
consumes only the plan documents, so it never re-reads the knowledge base — the
plan is the contract.

This document describes how the knowledge base is organised and how it behaves, so
a maintainer new to the project can understand it without reading the source.

## The category model

Every knowledge entry belongs to a **category**, determined by the first segment of
its path within a knowledge source (an entry at `gotchas/timeouts.md` is a
`gotchas` entry). There are six categories, each with a real definition — a
purpose, a boundary (what looks similar but belongs elsewhere), a retrieval tier,
and an expected entry shape:

| Category | Tier | What it holds |
|----------|------|---------------|
| `conventions` | always-applied | The rules a team always wants honoured — coding standards, naming, formatting, required patterns, house style. A standing rule stated as an instruction to follow. |
| `glossary` | always-applied | The shared vocabulary of the project — the domain and project-specific terms needed to read the rest of the knowledge base and the code. A term and a short gloss. |
| `architecture` | looked-up | How the system is built and fits together — components, responsibilities, boundaries, data and control flow. What exists and how it works. |
| `gotchas` | looked-up | Sharp edges and non-obvious traps — surprising behaviours and easy mistakes. A warning about a specific trap and how to avoid it. |
| `learnings` | looked-up | Empirical knowledge from doing the work — what was tried, what worked, what didn't. An observation from experience. |
| `decisions` | looked-up | The reasoning behind choices (ADR-style) — options considered, trade-offs weighed, why one path was taken. The why, not the what or how. |

These definitions are the categories' contract with both contributors and the
assistant. The boundaries are what keep the categories distinct: the *reasoning*
behind a rule is a `decision`, not a `convention`; a *defined term* is a
`glossary` entry, not `architecture`; an *empirical finding* is a `learning`, not
a `gotcha`.

### A single source of truth

The category model is declared **once**, in a registry in the code
(`internal/knowledge/category.go`). That one declaration drives everything
downstream:

- **Project initialisation** scaffolds a directory and a self-documenting README
  for each category from the registry, so a freshly initialised project contains
  every category with a real definition rather than a placeholder.
- **Retrieval behaviour** (below) reads each category's tier from the registry.
- **Contribution routing** surfaces the definitions to the assistant via the
  `spektacular knowledge categories` command, so a new entry is filed in the right
  place.

Adding a category, or moving one between tiers, is a change to this registry alone.

## The two retrieval tiers

Categories are split into two tiers by *how they are retrieved*, declared per
category in the registry:

- **Always-applied** (`conventions`, `glossary`). Loaded **in full on every task**
  and deliberately **excluded from search**, so the same content is never surfaced
  twice. These are the rules every plan must reflect and the vocabulary needed to
  read everything else. Keep them compact — their whole content is paid for on
  every task.
- **Looked-up** (`architecture`, `gotchas`, `learnings`, `decisions`). The larger
  reference body, retrieved **only when a query matches**. This is reached through
  keyword search, targeted at the specific surfaces a feature touches.

Because tier membership is read from the single registry declaration, **re-tiering
is one self-consistent action**: flipping a category to always-applied
simultaneously makes the full-load reader include it *and* makes search stop
returning it. The two behaviours cannot drift out of step — there is no second
place to forget to update.

The always-applied set is loaded with `spektacular knowledge always-applied`
(every always-applied entry across all sources, each tagged with its category).
The legacy `spektacular knowledge conventions` command remains for backward
compatibility, returning only the conventions.

## Search behaviour

Search is keyword-based and runs in-process. A document matches when every query
word occurs somewhere in it, in any order; results come back ranked — one result
per matching document, strongest match first. Each result carries:

- its **scope** (which configured source it came from — see precedence below),
- its **category** label (so a consumer can tell a `gotchas` warning from an
  `architecture` fact from a `learnings` finding),
- a **checksum** (a content hash, used for de-duplication — see below),
- a **title**, a **score**, and up to three **excerpts** of the strongest matching
  lines.

Always-applied categories never appear in search results — they are already loaded
in full — so search surfaces only the looked-up reference knowledge.

## De-duplication and consolidation

The same knowledge often lives in more than one place — copied across scopes, or
worded differently in two entries. A lookup does not return that pile of
overlapping hits. Instead it returns a single consolidated, source-cited answer,
produced in two strictly separated stages:

1. **Exact de-duplication (deterministic).** Search results are grouped by their
   `checksum`. Results sharing a checksum are **byte-identical** copies of the same
   entry; each group collapses to one candidate, unioning the source citations of
   every copy. This is pure equality — no thresholds, no judgement.
2. **Consolidation (judgement).** The surviving unique candidates are handed to a
   consolidation **sub-agent**, which reads their full bodies and classifies the
   *relationship* between them:
   - **equivalent** (same point, different words) → merged into one cited point;
   - **refinement** (one is a more specific case) → resolved by layered precedence
     (below);
   - **genuine contradiction** (sources actually disagree) → **surfaced** as a
     conflict naming both sources, never silently dropped;
   - **distinct** → both kept.

   The sub-agent returns one consolidated answer, each point citing the source(s)
   it was drawn from. The raw per-source hit list is never the output. (Where an
   executing agent cannot spawn a sub-agent, the identical consolidation runs
   inline; only the context isolation differs.)

### Why the mechanical layer is exact, not fuzzy

The deterministic de-dup step matches on **exact byte-identity** rather than a
normalized or similarity-based fingerprint. This is deliberate, because **lexical
distance does not track semantic distance**:

- "Always retry" and "never retry" are ~95% lexically identical yet a total
  contradiction.
- "Add jitter to retries" and "randomize backoff intervals" share almost no words
  yet state the same knowledge.

A graded similarity score (edit distance, SimHash/MinHash) would mis-rank exactly
these cases and reduces to a brittle magic threshold — and a mechanical similarity
index for equivalence is precisely the kind of embedding/semantic index the
project does not build. The real need is not a difference *scalar* but a
relationship *classification* (a 70%-similar pair could be a refinement or a
contradiction), which only an LLM does reliably.

So exact-equivalence is the most effective *mechanical* primitive precisely because
it is narrow: it claims the one provable point — zero difference — for free, and
routes everything semantic to the judgement layer, fed the inputs that make that
judgement good (full bodies, scope/specificity order, and the category label).

## Layered source precedence

A knowledge source has a **scope** label. A project ships one scope, `project`,
backed by `.spektacular/knowledge/`; additional scopes — for example a shared
`team` directory or a machine-wide `global` one — can be configured. Every read,
search, and always-applied load fans across all configured scopes in order, and
every result is tagged with the scope it came from.

When more than one scope covers the same item, precedence is **layered**: the
**most-specific scope wins**. Specificity runs `project` (most specific) → `team`
→ `global` (least specific) — a project-local entry overrides a team entry, which
overrides a global one. This is applied during consolidation: a refinement is
resolved in favour of the most-specific source, and the override is noted rather
than hidden. A *genuine* disagreement between sources is surfaced as a conflict,
not silently resolved.

## Contributing knowledge

Contributions are routed to the right category at the moment they are filed. The
`spek-knowledge` skill loads the category definitions
(`spektacular knowledge categories`), picks the category whose purpose matches the
entry and whose boundary does not exclude it, and steers over-long or
multi-paragraph content out of the always-applied `glossary` toward a more fitting
category — keeping the always-applied tier compact. The entry is filed at
`<category>/<slug>.md`. As always, the assistant proposes the scope, path, and body
and waits for explicit confirmation before writing.

## Command reference

Agents (and you) reach knowledge through the `spektacular knowledge` commands
rather than reading the files directly, so access stays consistent across scopes.
Each command has a `--schema` mode that prints its input/output schema.

| Command | Purpose |
|---------|---------|
| `spektacular knowledge search <query>` | Keyword-search every scope (always-applied categories excluded); ranked, one scope- and category-tagged result per matching document, each with title, score, excerpts, and a content checksum |
| `spektacular knowledge read --data '{"scope":"project","path":"architecture/x.md"}'` | Read one entry's full body from a named scope |
| `spektacular knowledge list` | List every entry across all scopes |
| `spektacular knowledge write --data '{"scope":"project","path":"gotchas/x.md"}' --file <path>` | Write an entry into a named scope (content from `--file`, or stdin) |
| `spektacular knowledge sources` | List the configured scopes and their locations |
| `spektacular knowledge categories` | List the category definitions — purpose, boundary, tier, and entry shape |
| `spektacular knowledge always-applied` | Read every always-applied entry (conventions and glossary) across all scopes, each tagged with its category |
| `spektacular knowledge conventions` | Read every convention across all scopes (the conventions-only, backward-compatible view) |
