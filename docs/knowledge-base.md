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

- its **tier** and store **name** (which configured store it came from — together
  with the path, exactly what a read needs to fetch it back),
- its **category** label (so a consumer can tell a `gotchas` warning from an
  `architecture` fact from a `learnings` finding),
- a **checksum** (a content hash, used for de-duplication — see below),
- a **title**, a **score**, and up to three **excerpts** of the strongest matching
  lines.

Always-applied categories never appear in search results — they are already loaded
in full — so search surfaces only the looked-up reference knowledge.

## De-duplication and consolidation

The same knowledge often lives in more than one place — copied across stores, or
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
   - **refinement** (one is a more specific case) → kept alongside what it
     refines and noted, never hidden (there is no precedence between stores —
     see below);
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
judgement good (full bodies, the store each came from, and the category label).

## Two tiers, and how a store is addressed

Knowledge lives in one of two tiers. Every registered repo contributes exactly one
store, holding what is true of that repo's own code, declared as a single provider
block in its `repo.yaml` and addressed by the name the project registered the repo
under. The project declares any number of shared stores under `knowledge.sources`
in `.spektacular/config.yaml`, each with its own `name`, for knowledge that belongs
to no single repo. Names must be unique within a tier; the same name may appear
once in each, because a store is identified by its tier and name together.

Reading and writing address exactly one store, so they take a `tier` and a `name`
alongside the path. A request that leaves either out is refused rather than
resolved to whichever store looks closest, and the refusal lists the names
available in that tier so it can be reissued immediately.

Searching, listing, conventions, and the always-applied load take `--tier
<project|repo|all>` and a repeatable `--filter <name>`. Omitting the narrowing
covers every store the tier reaches; naming stores covers exactly those. No store
is ever included or excluded implicitly, which is what lets planning work on one
repo skip every other repo's standing rules.

**There is no precedence between stores.** A repo's own store answers what is true
of that repo's code, and the project's shared stores answer what spans repos; they
answer different questions rather than overriding one another. During consolidation
a refinement is kept alongside what it refines and noted rather than hidden, and a
*genuine* disagreement is surfaced as a conflict naming both stores, not silently
resolved.

## Contributing knowledge

Contributions are routed to the right category at the moment they are filed. The
`spek-knowledge` skill loads the category definitions
(`spektacular knowledge categories`), picks the category whose purpose matches the
entry and whose boundary does not exclude it, and steers over-long or
multi-paragraph content out of the always-applied `glossary` toward a more fitting
category — keeping the always-applied retrieval tier compact. The entry is filed at
`<category>/<slug>.md`. As always, the assistant proposes the destination — the
tier, the store name, and the path — along with the body, and waits for explicit
confirmation before writing.

## Command reference

Agents (and you) reach knowledge through the `spektacular knowledge` commands
rather than reading the files directly, so access stays consistent across stores.
Each command has a `--schema` mode that prints its input/output schema.

| Command | Purpose |
|---------|---------|
| `spektacular knowledge search <query> [--tier T] [--filter N]` | Keyword-search the stores the request covers (always-applied categories excluded); ranked, one tier-, store- and category-tagged result per matching document, each with title, score, excerpts, and a content checksum |
| `spektacular knowledge read --data '{"tier":"repo","name":"docs","path":"architecture/x.md"}'` | Read one entry's full body from one addressed store |
| `spektacular knowledge list [--tier T] [--filter N]` | List every entry across the stores the request covers |
| `spektacular knowledge write --data '{"tier":"repo","name":"docs","path":"gotchas/x.md"}' --file <path>` | Write an entry into one addressed store (content from `--file`, or stdin) |
| `spektacular knowledge sources` | List the configured stores by tier and name, with their locations |
| `spektacular knowledge categories` | List the category definitions — purpose, boundary, tier, and entry shape |
| `spektacular knowledge always-applied [--tier T] [--filter N]` | Read every always-applied entry (conventions and glossary) across the stores the request covers, each tagged with its category |
| `spektacular knowledge conventions [--tier T] [--filter N]` | Read every convention across the stores the request covers (the conventions-only view) |
