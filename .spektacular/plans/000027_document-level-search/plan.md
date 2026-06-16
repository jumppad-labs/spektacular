# Plan: 000027_document-level-search

<!-- Metadata -->
<!-- Created: 2026-06-11T15:35:38Z -->
<!-- Commit: e3fd998 -->
<!-- Branch: f-ripgrep-replace -->
<!-- Repository: git@github.com:jumppad-labs/spektacular.git -->

<!--
  OVERVIEW
  A concise 2-3 sentence summary of the plan. Answer:
    1. What is being built?
    2. What problem does it solve?
    3. Who benefits?
  No file paths, no commands, no implementation detail. A reviewer should be
  able to decide whether the plan is worth reading in full from this section
  alone.
-->
## Overview

This plan makes knowledge search answer at the level searchers act on — the document:
a multi-word query matches an entry when every word appears somewhere in it, and each
matching entry comes back exactly once, ranked by match strength and carrying its
title and most relevant excerpts. It fixes both failure modes of today's per-line
search — queries like "workflow steps" that find nothing because the words never share
a line, and broad queries that bury the right entry under dozens of repetitive line
hits. Anyone searching a project's knowledge base — a person at the CLI or an agent
deciding what to read — can tell at a glance which entries are worth opening.

<!--
  CONVENTIONS
  The project conventions (always-apply standards) that bear on this work,
  each with a one-line reason it applies — and only the relevant ones, not
  the whole knowledge base. Conventions are loaded in full during discovery
  and the relevant subset is chosen as the design is locked, then confirmed
  by the user. Cite a convention inline in the sections it drives. If no
  conventions are relevant (or the project has none), state that plainly,
  e.g. "No project conventions apply to this feature." An empty or generic
  list is a visible signal the knowledge base was not consulted.
-->
## Conventions

No project conventions apply to this feature: the project's conventions store
was loaded in full during discovery and is empty.

<!--
  ARCHITECTURE & DESIGN DECISIONS
  The chosen design direction in 2-4 short paragraphs. Explain the shape of
  the solution, the key decisions and their trade-offs, and why the chosen
  direction beats the alternatives. Cross-reference
  research.md#alternatives-considered-and-rejected so readers can drill into
  the evidence for rejected options. This is plan.md's load-bearing section —
  a reviewer should be able to spot missing architectural patterns or design
  gaps from this section without needing to read context.md.
-->
## Architecture & Design Decisions

The solution evolves the existing in-process scanner rather than adding any new
search machinery. The query is tokenized into lower-cased, whitespace-separated
terms (`strings.Fields`); a query with zero terms — empty or all whitespace —
returns an empty result immediately, exactly as the empty query does today. The
per-file scan (`scanFile`, internal/store/search.go:96) changes from "collect
matching lines" to a single-pass per-file aggregate: for each line it records
how many times each term occurs, remembers the first ATX-heading line it sees
as the document title, and keeps candidate excerpt lines. A file matches when
every term occurs at least once somewhere in it, and a matching file produces
exactly one `Hit` carrying its scope, locator path, title (locator when the
file has no heading — e.g. JSON entries), a score that is the sum of all
terms' case-insensitive occurrence counts across the document, and up to three
excerpts. Excerpt lines are chosen by most distinct terms on the line, then
highest total occurrences, then earliest in the file, each still trimmed to
the existing 256-byte excerpt budget.

Ranking lives in `knowledge.Set.Search` (internal/knowledge/set.go:87) — the
single point where every scope's hits meet — which changes from plain
concatenation to a merge sorted by score descending, with ties broken by
configured source order and then by path so output is deterministic for tests
and for agents comparing runs. Sorting in the CLI layer instead was rejected
because the library API would keep returning unranked results that every
future consumer must re-sort. The `Hit` struct (internal/store/store.go:23)
changes shape: `excerpt` (string) becomes `excerpts` (array of strings),
`title` is added, and `score`'s meaning shifts from per-line occurrence count
to per-document term-occurrence sum; `scope` and `path` are untouched, so the
search → read round trip — pass a hit's scope and path to `knowledge read` —
is preserved exactly. This shape change is sanctioned by the spec, which
requires all shipped consumers (command schema and help text, the
spek-knowledge skill template and its rendered copy, the plan discovery step
template, the planning-agents skill template, README, CHANGELOG) to move in
lockstep within this change.

The single-pass aggregation is the load-bearing decision: it preserves every
hardened behaviour of the scanner for free — binary-file sniffing, the
`conventions/` directory exclusion, and the over-long-line policy of keeping
what was scanned and silently skipping the remainder of the file — because the
walk and the scan loop are untouched; only what is accumulated per file
changes. The accepted trade-off is that a term occurring only after an
over-long line is invisible, so such a document may fail to match a query its
full text contains; this preserves the existing behaviour and the spec
requires only that the search succeed and other entries be returned. The
single-word regression guarantee holds by construction: with one term,
"some line contains the substring" is equivalent to "the term occurs somewhere
in the document", so exactly the files that produced line hits before now
produce document hits.

Two alternatives were rejected with evidence: reading whole files into memory
and checking terms with `strings.Contains` (loses the streaming long-line
resilience and bounds nothing), and running one scan per term then
intersecting per-file results (N directory walks per query, and no single
pass ever sees all terms, making title capture and distinct-term excerpt
selection impossible without re-reading). See
research.md#alternatives-considered-and-rejected for the full analysis and
citations.

<!--
  COMPONENT BREAKDOWN
  The components (new or changed) that make up the solution, with their
  responsibilities and how they interact. One bullet or short paragraph per
  component. Name the component, state what it owns, and describe its
  relationship to the other components. Do not list file paths or line
  numbers here — component responsibilities, not implementation sites.
-->
## Component Breakdown

All components except the search-output documentation set are existing
components being changed; no new component is introduced.

- **Query tokenizer (new responsibility inside the store search entry
  point)** — owns turning the raw query string into the set of lower-cased
  terms that must all be present, and the rule that zero terms (empty or
  whitespace-only query) short-circuits to an empty result. Feeds the
  per-file scanner; nothing else sees the raw query string.

- **Per-file scanner (changed: `scanFile` and its result type)** — owns the
  single streaming pass over one file: binary detection, line scanning with
  the bounded buffer, the over-long-line keep-and-skip policy, per-term
  occurrence counting, capture of the first ATX-heading line as the title,
  and selection of the best excerpt lines (most distinct terms, then most
  occurrences, then earliest). Returns one per-file aggregate to the store
  search instead of a list of matching lines. It remains the only component
  that reads file contents.

- **Store search (changed: `FileStore.Search`)** — owns the walk of one
  store's root (including the `conventions/` exclusion), the
  all-terms-present match decision over each file's aggregate, and the
  construction of exactly one scope-tagged result per matching document —
  applying the title fallback to the locator and the per-excerpt trim
  budget. Emits results to the knowledge set; ordering within a store is
  walk order, ranking is not its concern.

- **Result type (changed: `Hit`)** — owns the wire shape of one search
  result: scope, locator path, title, document-level score, and the
  excerpts array. The scope and path fields are deliberately untouched so a
  result still feeds the existing read command unchanged. Shared vocabulary
  between the store search, the knowledge set, the CLI command, and the
  CLI's output schema.

- **Knowledge set search (changed: `Set.Search`)** — owns the fan-out
  across configured scope-tagged sources and, newly, the global ranking of
  the merged results: score descending, ties broken by configured source
  order then by path, so output is deterministic. It is the single point
  where results from all scopes meet, which is why ranking lives here and
  not in the CLI.

- **Knowledge search command (changed: the CLI search subcommand)** — owns
  the user-facing contract: argument handling, the declared output schema,
  the `hits` JSON envelope, and the empty-result normalisation. Adapts to
  the new result shape and updated schema/help text but keeps the envelope
  and the read command untouched.

- **Search-output documentation set (changed: shipped templates and docs)**
  — owns every shipped description of what search returns: the
  spek-knowledge skill template and its rendered copy, the plan-workflow
  discovery step template, the planning-agents skill template, the README
  command table, and the CHANGELOG entry. Moves in lockstep with the result
  type so nothing shipped still describes per-line results.

- **Search and set tests (changed)** — own the executable specification of
  the new behaviour: document-level matching, ranking, title and excerpt
  selection, the regression guarantees (single-word discovery set,
  conventions exclusion, binary skip, long-line resilience), and the
  CLI-level output shape. Updated alongside each component above.

<!--
  DATA STRUCTURES & INTERFACES
  The types, interface signatures, and serialization boundaries introduced or
  changed by the plan. Show type shapes in pseudocode or a short code block
  where useful. Focus on the contract between components, not internal
  representation detail.
-->
## Data Structures & Interfaces

One public type changes shape, one internal contract changes, and one
serialization boundary follows the public type. No new public types or
interfaces are introduced.

**`store.Hit` (changed — the search result, and the wire shape).** One hit
now represents one matching document rather than one matching line:

```go
type Hit struct {
    Scope    string   `json:"scope"`    // unchanged: scope label of the originating store
    Path     string   `json:"path"`     // unchanged: locator, relative to the store root — pass to Read
    Title    string   `json:"title"`    // new: first heading, or the locator when none
    Score    float64  `json:"score"`    // changed meaning: sum of all terms' occurrences across the document
    Excerpts []string `json:"excerpts"` // replaces excerpt: up to three budget-trimmed matching lines
}
```

`Scope` and `Path` are deliberately untouched: a hit still feeds the existing
read command (`knowledge read --data '{"scope":…,"path":…}'`) without any
change to that contract.

**`Store.Search(query string) ([]Hit, error)` (signature unchanged, semantics
changed).** The interface method keeps its signature; what changes is the
contract: a multi-word query matches a document only when every
whitespace-separated term occurs in it (literal, case-insensitive substring,
any order or distance), each matching document yields exactly one Hit, and a
query with no terms returns an empty result without error.

**`Set.Search(query string) ([]store.Hit, error)` (signature unchanged,
semantics changed).** Previously documented as performing no ranking; now its
contract includes global ordering of the merged results — score descending,
ties broken by configured source order then by path — so the slice it returns
is the display order.

**Per-file scan aggregate (internal contract between scanner and store
search, replaces `lineMatch`).** The scanner returns one aggregate per file
instead of a slice of matching lines: which terms were seen and how often in
total, the first heading line if any, and the best candidate excerpt lines.
The store search derives the match decision, score, title fallback, and
trimmed excerpts from this aggregate. Internal to the store package; exact
field layout is implementation detail.

**CLI output schema (follows `store.Hit`).** The search command keeps its
`{"hits": [...]}` envelope and empty-array normalisation; the declared item
schema gains `title` and `excerpts` and drops `excerpt`. The read command's
input and output schemas are untouched.

<!--
  IMPLEMENTATION DETAIL
  High-level only. Sketch new patterns being introduced, major code-shape
  changes, and code-structure UX — enough for a reviewer to spot missing
  patterns or design gaps. This is NOT per-phase file:line work — that
  belongs in context.md. If you find yourself writing "in file X at line Y",
  stop and move it to context.md.
-->
## Implementation Detail

No new patterns are introduced; the change deepens the pattern the scanner
already follows. The streaming per-file scan stays the only place file
contents are read, and the existing layering — scanner produces raw per-file
facts, store search turns facts into scope-tagged results, knowledge set
merges across stores, CLI serialises — is kept; each layer simply gains the
document-level responsibility that belongs at its altitude. The one
deliberate shift in code shape is that the scanner's result type changes
from "a list of matching lines" to "one aggregate describing the whole
file", which moves the match decision out of the scan loop (where a line
either matched or didn't) up into the store search (where "did every term
appear" can only be answered once the file is finished).

Term handling follows the codebase's existing lower-case-once idiom: the
query is tokenized and lower-cased a single time at the search entry point,
and the scanner compares lower-cased lines against the prepared terms — the
multi-term generalisation of how the single needle works today. Title
capture and excerpt-candidate selection ride the same loop iteration that
counts term occurrences, so the file is still read exactly once, the binary
sniff and over-long-line policies apply to all three concerns uniformly, and
no second pass or buffering of the whole file is ever needed. Excerpt
selection keeps a small bounded set of best lines as it goes rather than
collecting all matching lines and sorting at the end, preserving the
bounded-memory property the scanner was just rebuilt to have.

Ranking is introduced where merging already happens: the knowledge set's
search changes from append-only concatenation to a single deterministic sort
of the merged slice. A developer reading the changed code sees the same
files doing the same jobs — a scanner, a store walk, a set merge, a thin CLI
— with the per-line fan-out loop replaced by a per-document constructor, and
the set's documented "no ranking" note replaced by an explicit ordering
contract. Test structure follows the existing conventions in the touched
packages: fixtures written into the test's own temp directory, and
hand-maintained expected values rather than oracles derived from the code
under test.

The documentation set (skill templates, plan-step templates, README,
CHANGELOG, command help and schema) is updated in the same change as the
result type, following the project's established rule that templates are the
source of truth and rendered copies (such as the repo's own installed
skills) are regenerated rather than hand-edited where a render path exists.

<!--
  DEPENDENCIES
  The internal packages, external libraries, upstream specs, or prior plans
  this work depends on. One bullet per dependency with a one-line note on
  what it provides and whether it needs any changes.
-->
## Dependencies

- **Plan 000026_ripgrep-replace (planning dependency — must land first).**
  Provides the self-contained in-process scanner this feature aggregates
  over, including the binary-sniff, bounded-buffer, and over-long-line
  behaviours this plan preserves. Its implementation is complete but sits
  uncommitted on the same branch (`f-ripgrep-replace`); it must be committed
  (or merged) before this work starts so the two changes stay reviewable
  separately.
- **`internal/store` (internal package — primary change site).** Owns the
  scanner, the store walk, and the `Hit` type; this plan changes all three.
- **`internal/knowledge` (internal package — changes).** Owns the
  multi-scope fan-out (`Set.Search`) that gains ranking, and the read path
  (`Set.Read`) that must not change.
- **`cmd` knowledge command (internal package — changes).** Owns the search
  subcommand's schema, help text, and JSON envelope; adapts to the new
  result shape.
- **`internal/output` writer (internal package — no changes).** Generic
  JSON serialisation; indifferent to the result shape.
- **Shipped templates and docs (change in lockstep).** The spek-knowledge
  skill template and its rendered copy, the plan discovery step template,
  the planning-agents skill template, the README command table, and the
  CHANGELOG all describe search output and move with the shape change — a
  spec constraint, not an optional cleanup.
- **External libraries: none.** The feature is constrained to remain fully
  self-contained — standard library only, no new module requirements, no
  external tools or services.
- **Spec 000027_document-level-search (upstream spec).** Source of truth
  for requirements, constraints, non-goals, and the success metrics the
  testing approach must carry.

<!--
  TESTING APPROACH
  High-level overview of the testing strategy: what kinds of tests
  (unit, integration, contract, regression), which components get the most
  coverage, and what the load-bearing assertions are. Per-phase testing
  detail — which specific tests live in which specific files — stays in
  context.md.
-->
## Testing Approach

All testing follows the conventions already established in the touched
packages: every fixture is written into the test's own temp directory (no
test reads a real knowledge base), and expected results are hand-maintained
oracles — never derived at runtime from the code under test. Coverage
concentrates on the store package, because that is where the matching model
changes; the knowledge set and CLI layers get thinner contract tests since
they only merge, rank, and serialise what the store produces.

**Unit tests on the store search** carry the load-bearing assertions, each
traceable to a spec requirement or acceptance criterion:

- Document-level AND matching: a multi-word query matches a file whose words
  are scattered across different lines; a file missing any one word is
  excluded; matching stays literal and case-insensitive.
- One result per matching document, however many lines contain query words.
- Score equals the hand-computed sum of all terms' occurrences across the
  document.
- Title is the file's first heading; a file with no heading falls back to
  its locator.
- Excerpts: at least one excerpt containing a query word; lines with more
  distinct terms win; each excerpt stays within the existing byte budget;
  the per-result excerpt cap holds.
- Regression guard: for a fixed fixture, the set of files found by a
  single-word query is identical to today's per-line behaviour — asserted
  with a hand-maintained oracle listing the expected files.
- Edge cases preserved: empty and whitespace-only queries return empty
  results without error; a convention file containing every query word
  never appears; a binary file containing the query words never appears; a
  file with a pathologically long line neither fails the search nor blocks
  other files' results.

**Unit tests on the knowledge set** assert the ranking contract: merged
results are ordered by score descending, ties break by configured source
order then path, and the ordering is deterministic across runs. The
search→read round trip is asserted once here — a hit's scope and path,
passed to read, return the entry's content — guarding the untouched read
contract.

**CLI-level tests on the knowledge command** assert the wire contract: the
`hits` envelope with the new per-document fields (scope, path, title, score,
excerpts), the declared output schema matching what is emitted, and the
empty-query and no-match cases serialising as an empty list, not an error.

**Deliberate gaps:** no performance or latency tests (knowledge bases are
small by standing assumption, and smarter relevance or limits are explicit
non-goals); no integration tests beyond the CLI command tests, because every
layer boundary is already exercised by the contract tests above; and per the
project's no-redundant-assertions convention, behaviours fully guarded at
the store layer are not re-asserted through the CLI layer.

**Success metrics — verification mapping.** The spec's three metrics are
measured against this project's own live knowledge base, which tests must
not read (tests own their filesystem), so each metric splits into a
behavioural test of the mechanism on fixtures plus a live check where
needed:

1. *The motivating queries now succeed* ("workflow steps", "workflow
   state", "interrupted workflow" each return the relevant architecture
   entry ranked first): the mechanism — scattered words match and stronger
   matches rank first — is asserted by behavioural tests on fixtures; the
   live-knowledge-base outcome itself is **manual — captured in the
   implementation test plan**.
2. *Broad queries become triageable* ("workflow" returns one ranked result
   per matching entry instead of 60 line hits, most-matching entry first):
   one-result-per-document and ranking are behavioural tests on fixtures;
   the live-knowledge-base outcome is **manual — captured in the
   implementation test plan**.
3. *No discovery regression* (every single-word query returns the same
   entry set as before): behavioural test on a fixed fixture with a
   hand-maintained oracle; additionally confirmed against the live
   knowledge base as **manual — captured in the implementation test plan**.

<!--
  MILESTONES & PHASES
  2-4 milestones. Each milestone leads with a "What changes" summary
  paragraph describing the user-visible difference when the milestone lands.
  Each phase has a 2-4 sentence plain-language summary, a *Technical detail:*
  link to context.md, and an **Acceptance criteria**: checkbox list with
  outcome statements (not shell commands). No file:line references in
  plan.md phase content — those live in context.md.
-->
## Milestones & Phases

### Milestone 1: Searches find entries by all their words, anywhere in the document

**What changes:** A search of several words now finds a knowledge entry when
every word appears somewhere in it — in any order, at any distance, on any
line — instead of only when the exact phrase sits on a single line. Each
matching entry comes back exactly once, carrying its title, a match-strength
score summed across the whole document, and up to three excerpts showing the
strongest matching lines. Everything searchers rely on today still holds:
single-word searches find exactly the entries they found before, matching
stays literal and case-insensitive, convention files and binary files stay
out of the results, an entry with a pathologically long line cannot break a
search, and empty or unmatched queries return cleanly empty results.

**Validation point:** The store-level test suite passes, including the new
document-matching, title, score, and excerpt assertions and the
hand-maintained single-word regression oracle; a fixture file whose query
words never share a line is found, and a fixture missing one word is not.

#### - [x] Phase 1.1: Per-document aggregation and matching

The scanner stops emitting one result per matching line and instead builds
one aggregate per file: how often each query word occurs across the whole
document. The query is split into words, and a document matches only when
every word appears somewhere in it; each matching document becomes exactly
one result whose score is the sum of all word occurrences. The result type
gains its new shape (title and an excerpts list) in this phase, populated
simply for now — the locator as title, the first matching lines as excerpts
— so the shape changes once. All the hardened behaviours (binary skip,
conventions exclusion, long-line resilience, clean empty results) are
preserved and re-asserted, and a hand-maintained oracle proves single-word
queries find exactly the same files as before.

*Technical detail:* [context.md#phase-11](./context.md#phase-11-per-document-aggregation-and-matching)

**Acceptance criteria**:

- [x] A query whose words appear in a document on different lines, in any order, returns that document; a query with one word absent from the document does not.
- [x] A document containing query words on many lines appears exactly once in the results, with a score equal to the hand-computed sum of all words' occurrences across it.
- [x] For the fixed test fixture, every single-word query returns exactly the same set of files as the per-line implementation did.
- [x] Empty and whitespace-only queries, and queries matching nothing, return an empty result without error; convention files and binary files containing the query words stay absent; a file with a pathologically long line neither fails the search nor hides other files' results.

#### - [x] Phase 1.2: Titles and best-line excerpts

Each result becomes self-describing: the document's first heading is
captured during the same single pass and becomes the result's title, falling
back to the locator for documents without headings (such as raw JSON
files), and the excerpts become the document's strongest lines — those
containing the most distinct query words, then the most occurrences — capped
at three per result and each trimmed to the existing compact budget.

*Technical detail:* [context.md#phase-12](./context.md#phase-12-titles-and-best-line-excerpts)

**Acceptance criteria**:

- [x] A markdown document's result shows its first heading as the title; a document with no heading shows its locator instead.
- [x] Every result carries at least one excerpt containing a query word; a line containing more distinct query words is chosen over a line containing fewer.
- [x] No result carries more than three excerpts, and every excerpt stays within the existing excerpt size budget.

### Milestone 2: Search results arrive ranked and ready to act on

**What changes:** The search command now returns one ranked result per
matching entry: the strongest match is listed first, and every result shows
where it came from, its locator, its title, its score, and its excerpts — so
a person or an agent can tell at a glance which entry to open and open it
with the read command exactly as before. The motivating failures disappear:
"workflow steps" finds the workflow architecture entry instead of nothing,
and a broad search for "workflow" returns a handful of ranked entries
instead of dozens of repetitive line hits.

**Validation point:** The knowledge-set and CLI test suites pass, including
ranking determinism and the search-to-read round trip; running the
motivating queries against this repository's own knowledge base returns the
relevant entry ranked first, and the broad "workflow" query returns
single-digit ranked results.

#### - [x] Phase 2.1: Globally ranked results across all knowledge sources

Merged results from all configured knowledge sources arrive ranked: the
strongest match first, with deterministic ordering for equal scores so two
runs of the same query always list results identically. The search-to-read
round trip is guarded: any result's source and locator still fetch the full
entry through the read command exactly as before.

*Technical detail:* [context.md#phase-21](./context.md#phase-21-globally-ranked-results-across-all-knowledge-sources)

**Acceptance criteria**:

- [x] Given one entry matching strongly and another weakly, the stronger entry is listed first regardless of which knowledge source each came from.
- [x] Results with equal scores appear in a stable, documented order (source configuration order, then locator), identical across repeated runs.
- [x] Passing any result's scope and locator to the read command returns that entry's full content, unchanged from today.

#### - [x] Phase 2.2: The search command speaks the new result shape

The command-line surface catches up with the engine: the search command's
declared output schema and help text describe ranked per-document results
(scope, locator, title, score, excerpts), while the response envelope and
the empty-result behaviour stay exactly as they are. The read command is
untouched.

*Technical detail:* [context.md#phase-22](./context.md#phase-22-the-search-command-speaks-the-new-result-shape)

**Acceptance criteria**:

- [x] The search command's JSON output lists per-document results carrying scope, locator, title, score, and excerpts, inside the same envelope as today.
- [x] The command's declared output schema matches what it actually emits.
- [x] An empty query and a no-match query both produce an empty result list, not an error, at the command level.

### Milestone 3: Everything shipped describes the new results

**What changes:** Every shipped surface that documents or teaches search —
the command's declared output schema and help text, the knowledge skill,
the planning workflow's instructions to agents, the README, and the
changelog — describes the ranked per-document results, so nothing shipped
still tells a reader or an agent to expect per-line hits. This is the
lockstep update the spec makes a hard constraint, delivered as its own
milestone so the documentation sweep is verified deliberately rather than
left as an afterthought.

**Validation point:** A sweep of the shipped templates, docs, and help text
finds no remaining description of per-line results or the old single-excerpt
field; the full project test suite passes, including the instruction-surface
tests over rendered templates.

#### - [x] Phase 3.1: Shipped documentation moves in lockstep

Every shipped surface that describes search output is updated in the same
change: the knowledge skill template and the repository's rendered copy, the
planning workflow's discovery instructions, the planning-agents skill, the
README command table, and the changelog. After this phase nothing shipped
still describes per-line results or the old single-excerpt field.

*Technical detail:* [context.md#phase-31](./context.md#phase-31-shipped-documentation-moves-in-lockstep)

**Acceptance criteria**:

- [x] No shipped template, skill, README section, or help text still describes one-result-per-line output or a singular excerpt field.
- [x] The knowledge skill's documented search-then-read flow matches the new output: ranked, scope-tagged, per-document results whose scope and locator feed the read command.
- [x] The changelog records the behaviour change, and the full project test suite — including the rendered-template checks — passes.

<!--
  OPEN QUESTIONS
  Strictly for questions that genuinely cannot be resolved until
  implementation begins. Anything resolvable by asking the user, reading the
  code, or running a quick experiment must be resolved now — not parked
  here. If this section is empty, that is the expected outcome of a healthy
  planning pass.
-->
## Open Questions

One genuine implementation-time uncertainty remains; everything else was
resolved during discovery and architecture.

- **Whether occurrence-count scoring ranks the relevant entry first on the
  live knowledge base.** The first success metric requires the motivating
  queries ("workflow steps", "workflow state", "interrupted workflow") to
  return the relevant architecture entry ranked *first*, but this project's
  knowledge base contains large raw JSON transcript files that produced a
  large share of the old per-line hits, and whether their summed occurrence
  counts outscore the genuinely relevant entry is only discoverable by
  running the finished search against the live knowledge base. The spec
  assumes occurrence counts "separate strong matches from incidental ones
  by a wide margin", and smarter relevance is an explicit non-goal. **If a
  noise entry outranks the relevant entry for any motivating query during
  the manual test-plan check, STOP and ask the user** — the fix could range
  from accepting it, to curating the knowledge base, to revisiting scoring
  in a follow-up spec, and that is the user's call, not the implementer's.

<!--
  OUT OF SCOPE
  Explicit exclusions agreed during planning. Each bullet states what is NOT
  being done and, where useful, where it is tracked instead. This is as
  important as the requirements — it prevents scope creep and sets clear
  expectations for reviewers.
-->
## Out of Scope

All exclusions below come from the spec's Non-Goals (each weighed against
the success metrics during the spec workflow and confirmed unnecessary) or
from deliberate choices made during this plan's design; each is a candidate
follow-up spec, not a hidden dependency. No follow-up specs exist yet for
any of them.

- **No approximate matching** — no typo tolerance, fuzzy matching, stemming,
  or synonyms. Substring matching is deliberately asymmetric: stem-form
  query words ("step") match longer forms ("steps", "stepping"), so
  stem-form queries are the recommended idiom; full stemming is deferred
  until the reverse miss proves a problem in practice (spec § Non-Goals).
- **No query syntax** — no quoted exact-phrase operator, no OR/NOT
  booleans. A query is simply words that must all be present (spec
  § Non-Goals).
- **No filename matching** — a query word appearing only in an entry's file
  name or locator does not make the entry match. Named in the spec as the
  cheapest follow-up if real misses surface after delivery (spec
  § Non-Goals).
- **No smarter relevance, result limits, or pagination** — match strength
  stays simple occurrence counts; no TF-IDF, semantic, or embedding-based
  ranking, no result caps or paging (spec § Non-Goals). If occurrence
  counts prove insufficient on the live knowledge base, that surfaces via
  this plan's open question and becomes a user decision, not silent scope
  growth.
- **No change to how a chosen entry is read** — the read command's input,
  output, and behaviour are untouched; this plan changes only what search
  returns (spec § Constraints).
- **No knowledge-base curation** — cleaning up noisy entries (such as raw
  JSON transcript files) that may dilute search results is not part of this
  plan, even though the open question notes it as one possible response if
  ranking disappoints on the live knowledge base.
- **No tuning of the scanner's resilience thresholds** — the binary-sniff
  window, the line-length cap, and the keep-and-skip long-line policy are
  inherited from plan 000026_ripgrep-replace unchanged; this plan only
  aggregates over what that scanner produces.

## Changelog

### 2026-06-11 — Phase 1.1: Per-document aggregation and matching

**What was done**: Replaced per-line search with document-level matching in
the store engine: the query is tokenized into lower-cased whitespace-separated
terms, a file matches only when every term occurs somewhere in it, and each
matching file yields exactly one `Hit` whose score is the summed occurrences
of all terms across the document. `Hit` changed shape — `Title` added
(locator-valued until Phase 1.2) and `Excerpt string` replaced by
`Excerpts []string` (first matching lines, max 3, each budget-trimmed). All
hardened scanner behaviours (binary sniff, conventions exclusion, over-long-
line keep-and-skip, empty/whitespace query → clean empty result) preserved
and re-asserted, plus a hand-maintained single-word regression oracle.

**Deviations**: Two, both user-sanctioned or noted for a later phase:
(1) the plan's prerequisite that the uncommitted 000026_ripgrep-replace work
land before Phase 1.1 was waived by the user — both changes coexist
uncommitted on branch `f-ripgrep-replace`; (2) `cmd/knowledge_test.go` was
minimally adapted in this phase (local hit struct `excerpt` → `excerpts`
array) because the Hit JSON shape change would otherwise break the suite
before Phase 2.2, which still owns the CLI contract assertions.

**Files changed**:
- `internal/store/store.go`
- `internal/store/search.go`
- `internal/store/search_test.go`
- `cmd/knowledge_test.go`

**Discoveries**: The 3-excerpt cap is an inline literal in `search` for now —
Phase 1.2 should hoist it to the planned `maxExcerptsPerHit` const when it
adds best-line selection. Binary files now produce a zero aggregate rather
than a nil match list; they fall out naturally via the every-term-count>0
match rule. The per-file aggregate (`fileAggregate{counts []int, lines
[]string}`) indexes counts by term position, ready for Phase 1.2 to extend
with per-line distinct-term/occurrence tracking.

### 2026-06-11 — Phase 1.2: Titles and best-line excerpts

**What was done**: Results are now self-describing: the scan captures the
file's first ATX heading (markers stripped) as the hit's title, falling back
to the locator for heading-less files, and excerpts are the document's
strongest lines — ranked by distinct query terms, then total occurrences,
then file order — kept as a bounded top-3 during the single pass
(`maxExcerptsPerHit`), each still trimmed by `trimExcerpt`.

**Deviations**: None.

**Files changed**:
- `internal/store/search.go`
- `internal/store/search_test.go`

**Discoveries**: `headingText` rejects `#hashtag`-style lines (a `#` run flush
against text) and treats a bare `#` run as a heading with empty text — the
`titleSet` flag means such a degenerate first heading wins the "first heading"
slot and the empty text falls back to the locator, so a later real heading
never becomes the title. Excerpt candidates insert after equal-ranked entries,
which is what preserves file order as the final tie-break. "Every excerpt
contains a query word" is guaranteed by construction (candidates are matching
lines) and pinned by the exact-string oracle in
`TestSearch_ExcerptPrefersLineWithMoreTerms` — no separate assertion added,
per the no-redundant-assertions convention.

### 2026-06-11 — Phase 2.1: Globally ranked results across all knowledge sources

**What was done**: `knowledge.Set.Search` now returns the merged hits from
all configured sources globally ranked — score descending, ties broken by
configured source order then by path ascending — replacing plain
concatenation; its doc comment now states the ordering contract instead of
"no ranking or dedup". The set-level search→read round trip is guarded by a
new test feeding a hit's scope and path into `Set.Read`.

**Deviations**: None.

**Files changed**:
- `internal/knowledge/set.go`
- `internal/knowledge/set_test.go`

**Discoveries**: The tie-break test fixture pairs `notes.md` with
`notes/zz.md` deliberately: `filepath.WalkDir` discovers `notes/zz.md`
before `notes.md`, while path-ascending sorts them the other way ('.' <
'/'), so the test genuinely discriminates the documented tie-break from
incidental walk order — keep that property if the fixture is ever changed.

### 2026-06-11 — Phase 2.2: The search command speaks the new result shape

**What was done**: The search subcommand's declared output schema now
describes `hits` items as per-document objects (`scope`, `path`, `title`,
`score`, `excerpts`), and its help text says ranked, one-per-document
results. The `{"hits": [...]}` envelope, the nil→`[]` empty normalisation,
and the read command are untouched. CLI tests pin the exact emitted hit
list (including the documented project-before-team tie order), the schema
declaration, and the empty-query/no-match empty-list behaviour.

**Deviations**: One small, necessary addition not named in the plan:
`schemaProp` (cmd/spec.go) gained an optional `Properties` map so an array's
item schema can describe object fields — previously inexpressible. Additive
and `omitempty`, so every other command's `--schema` output is unchanged.

**Files changed**:
- `cmd/spec.go`
- `cmd/knowledge.go`
- `cmd/knowledge_test.go`

**Discoveries**: `search --schema` still requires a dummy positional
argument because `cobra.ExactArgs(1)` validates before the `--schema`
short-circuit inside `RunE` — pre-existing behaviour, shared by other
commands, noted in the schema test. The CLI-level title assertions exercise
only the locator-fallback branch (the shared CLI fixture has no headings);
heading-derived titles are guarded at the store layer.

### 2026-06-11 — Phase 3.1: Shipped documentation moves in lockstep

**What was done**: Every shipped surface describing search output now
speaks the new shape: the spek-knowledge skill template's lookup flow
(ranked one-per-document results with scope, path, title, score, excerpts,
all-words matching), the plan discovery step, the spawn-planning-agents
skill, the README (command table row, the `Hit` struct example, and the
built-in `Search` bullet), and a new top CHANGELOG entry recording the
behaviour and JSON shape change. The repository's rendered skill copies
were regenerated through `go run . init claude` rather than hand-edited.

**Deviations**: Two small ones. (1) The sweep found a stale surface the
plan's file list missed: the README's `Hit` struct code block (~line 200)
still showed `Excerpt string` — updated. (2) The spawn-planning-agents
wording also dropped "conventions" from its list of knowledge-search
targets, since conventions have been excluded from search results since
000025 — a correctness fix adjacent to the planned wording change.

**Files changed**:
- `templates/skills/workflows/spek-knowledge/SKILL.md`
- `templates/steps/plan/02-discovery.md`
- `templates/skills/skill_spawn-planning-agents.md`
- `README.md`
- `CHANGELOG.md`
- `.claude/skills/*` (regenerated via init; gitignored, not tracked)

**Discoveries**: `.claude/skills/` is gitignored — the rendered copies are
not committed, so "regenerate via init" is how they stay in sync rather
than a tracked artifact to review. The instruction-surface and plan-step
tests pin only stable strings ("knowledge search", stdin-pattern bans),
so template wording can evolve without test churn.
