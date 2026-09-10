---
created_date: "2026-09-10"
status: completed
closed_date: "2026-09-10"
---

# Plan: 000050_knowledge-entry-tags

<!-- Metadata -->
<!-- Created: 2026-09-10T11:37:29Z -->
<!-- Commit: 09f2bb7 -->
<!-- Branch: b-knowledge -->
<!-- Repository: git@github.com:jumppad-labs/spektacular.git -->

## Overview

Knowledge entries gain optional tags that say what an entry is about, independently
of the words its prose happens to use, and knowledge search changes from
all-or-nothing keyword matching to genuine ranking. Today an entry is found only if
it shares vocabulary with whatever query the coding agent invents at the time, and
that query is not stable between runs of the same work, so an entry that is exactly
on topic can be missed entirely and the agent plans without knowledge the team has
already written down. Tags are proposed by the agent at the moment an entry is
captured, drawn from the tags already in use, so the knowledge base stays findable as
it grows without anyone maintaining a taxonomy by hand. Everyone who plans against
this knowledge base benefits, and no existing entry has to change for it to take
effect.

## Conventions

Selected from the always-applied conventions loaded during discovery across both
registered repos (`spektacular` and `docs`).

**`spektacular` repo — bind the CLI and engine work:**

- **Error messages must describe the problem and suggest remediation** — every new refusal this feature adds (a `--tag` value no entry carries, a malformed frontmatter block, an unreadable tag list) must be built with `output.NewError(code, message).WithNextAction(<runnable command>)`, never a bare `fmt.Errorf`. For the tag-filter case the next action is `knowledge tags`, which is precisely the command that lists the values the caller was missing.
- **Passing tests are required before calling work done** — this feature deliberately changes the scores that several existing tests assert (`internal/store/search_test.go:153,286,437`; the hand-computed comment at `internal/knowledge/set_test.go:122`). Those are expected breakages caused by this change, so each must be recomputed and fixed here, never dismissed as pre-existing, and `go test ./...` must pass in full before the work is reported done.

**`docs` repo — bind the documentation phase only:**

- **Plans must sketch content structure, not just summarize it** — the documentation requirement changes a substantial block of copy on an existing page, so its phase must carry a labelled `**Content example**` block giving the headings in order and an illustrative excerpt per section, with the technical facts (flag names, ranking factors, the exact `--tag` syntax) fixed by this plan's research rather than left for the implementer to invent.
- **No em dashes** — all authored prose this feature adds to the docs site, and the commit and PR descriptions for that repo, must use a comma, colon, parentheses or a separate sentence instead of `—`.
- **MDX authoring conventions** — the ranking and tag copy lands in `src/pages/knowledge-base.mdx`, so Rule 1 (no `<div>`, `<section>` or `class=` in a page body; guard `grep -nE "<div|<section|class=" src/pages/*.mdx` returns zero), Rule 3 (blank line after an opening tag and before a closing tag on any slotted component) and Rule 4 (code samples are fenced markdown, never a JSX code prop) all apply directly to the edit.
- **Alternate section background shading** — if the documentation change adds a new top-level band rather than editing the existing "The lifecycle of an entry" section, it must set `surface` explicitly to the opposite of the band before it. The current run on that page is: What it is (plain) → The six categories (`surface`) → The lifecycle of an entry (plain) → Configuration (`surface`) → Why it works this way (plain), so a band appended before the closing `CtaBanner` must be `surface`.
- **Site layout conventions** — the change must compose from the existing components under `src/components/sections/` (`Section` + `Prose` for a band with a markdown table, `ConfigurationKeys` + `ConfigKey` for a flag list) rather than introducing markup or a new frame width.

**Deliberately not applied:**

- **Label before filename in file-scoped reference headings** (`docs`) — this feature edits prose inside an existing page and adds no new file-scoped reference section, so the heading rule has nothing to bind.
- The two category `README.md` entries returned by the always-applied load in each repo are the knowledge base's own self-documentation, not rules about the code, and carry no obligation for this work.

## Architecture & Design Decisions

### The shape of the solution

A knowledge entry declares its tags in a leading YAML frontmatter block
(`tags: [go, http]`). Retrieval changes from boolean AND to a ranked OR whose
score has three factors: a **tag hit**, weighted well above prose; a **damped body
count**, so repetition saturates; and a **coverage factor**, so an entry carrying
evidence for more of the query outranks one carrying evidence for less. Weak hits
are then dropped relative to the strongest hit in the merged result set, not
against an absolute threshold. At capture time the `spek-knowledge` skill proposes
tags alongside the tier, store, path and body it already proposes, drawing them
from the vocabulary already in use, and a new audit intent reviews existing
entries the same way. Nothing is written without explicit confirmation.

The work splits across four surfaces in `spektacular` — the scan (`internal/store`),
the merge (`internal/knowledge`), the CLI contract (`cmd/knowledge.go`), and the
agent-facing prose (`templates/`) — plus one page in `docs`.

### Decision 1: stores report evidence, the knowledge layer scores

A store's job is to find candidate documents and report what it found: for each
query term, how many times it occurs in the entry's body; which terms match one of
the entry's tags; and the entry's tags, title, excerpts and checksum. A store does
**not** compute a score. Turning that evidence into a ranking number happens once,
in `knowledge.Set.Search` (`internal/knowledge/set.go:140-190`), alongside the
cross-store merge and the weak-match cutoff.

This follows a pattern the codebase already uses. `Hit`'s tier, store name and
category fields are documented as "left empty by the store and populated by the
knowledge layer" (`internal/store/store.go:25-36`), because a store has no notion of
its caller's addressing scheme. Ranking is the same kind of thing: a store has no
notion of the other stores its results will be merged against. `Score` simply joins
the list of fields the knowledge layer stamps.

The alternative, each store scoring its own results, fails as soon as there is more
than one implementation. Every new store would have to reproduce the tag weight, the
damping curve and the coverage exponent exactly, or the cross-store merge would be
sorting numbers that do not mean the same thing, and the cutoff (which is relative to
the strongest hit anywhere) would be comparing one store's scale against another's.
Nothing would fail loudly; results would simply be quietly wrong. Requirements such
as "a tag match outweighs a body occurrence" and "a weak match disappears in company"
are statements about the whole result set, so they can only be guaranteed at the
layer where the whole result set exists.

This costs no extra I/O. `scanFile` (`internal/store/search.go:135-215`) still reads
each file exactly once and still collects per-term counts, the title, the excerpt
candidates and a SHA-256 in that single pass. It hands that evidence up instead of
collapsing it to a number first.

Two consequences worth stating plainly. Query tokenization moves up to the knowledge
layer as well, so both sides index evidence by the same term order rather than
agreeing by convention. And a store must be *able* to report per-term occurrence
counts in order to participate in this ranking at all: a purely semantic backend that
can only return its own opaque relevance score cannot, and that is a genuine
constraint on future providers rather than an oversight. It is recorded here so the
next provider's author meets it in the design rather than in a debugging session.

`Store.Search` therefore takes the tokenized terms plus an options value, rather than
a raw query string. The blast radius is contained: the only implementors and callers
in the tree are `internal/store/search.go:46`, the pass-through decorator at
`internal/store/ignore.go:107`, and `internal/knowledge/set.go:153`. Nothing outside
the knowledge layer consumes the store's search at all.

### Decision 2: tags are frontmatter, read by a new parser, not `internal/metadata`

Tags go in a YAML frontmatter block because that is the one place a reader can find
them without them polluting the prose, and because an entry that has none is
simply an entry with no block — no migration, no backfill, no command to run, which
is what the spec's constraint demands.

The trade-off is that this cannot reuse `internal/metadata`, despite the surface
similarity. `metadata.Metadata.UnmarshalYAML` (`internal/metadata/metadata.go:83-108`)
hard-requires a parseable `created_date` and a `status` from a four-value enum, and
`metadata.Render` (`:59-73`) re-marshals through a fixed shape that silently drops
unknown keys. An entry carrying only `tags:` would be rejected as malformed, and any
round-trip through that package would delete the tags. That package owns the
*workflow artifact* schema — spec, plan, changelog — which is a genuinely different
schema with different mandatory fields. A small dedicated reader copies the proven
block-detection shape from `internal/metadata/frontmatter.go:16-52` without
inheriting its schema. The cost is a second frontmatter code path; the alternative
was making `created_date` and `status` mandatory on every knowledge entry, which
the no-migration constraint rules out outright.

Tag matching is by **prefix, with partial credit**, after lower-casing. A query term
matching a tag exactly earns the full tag weight. A term that is a prefix of a tag, or
a tag that is a prefix of the term, earns a fraction of it: the length of the shorter
string over the length of the longer. So searching `http` against a `https` tag earns
four fifths of the weight, and searching `https` against a `https-security` tag earns
five fourteenths. Body matching stays a plain substring count, unchanged.

The prefix relation is deliberately not a general string-distance measure. Edit
distance was the obvious candidate and is actively dangerous here, because tags are
overwhelmingly short technical tokens and edit distance is at its worst on exactly
those: it scores `test` against `rest` at 0.75 and `cors` against `core` at 0.75, with
`tls`/`tld`, `dns`/`dos` and `api`/`apt` all at 0.67. Every one of those pairs is two
unrelated subjects, and every one is a plausible tag in this codebase. Prefix matching
scores all of them zero while producing identical scores to edit distance on every case
that should match. The evidence table is in
`research.md#alternatives-considered-and-rejected`.

Two guards keep the prefix rule from becoming the "any short term hits everything"
failure that ruled out plain substring matching. The match must be anchored at the
start, so `go` does not hit `golang` by appearing anywhere inside it; and the shorter
string must reach a minimum length, so two- and three-character terms match only
exactly. Partial credit does the rest: a loose match is scored like a loose match
rather than being given the full weight of a real one.

This mechanism is load-bearing for a problem the earlier draft handled with prose. With
exact matching, an entry tagged `apple` was simply never retrieved by a search for
`apples`, and the only mitigation was instructing the capture agent to carry both
forms, which nothing enforced and which failed silently when missed. Prefix partial
credit makes both directions match at 0.83 arithmetically, on every search, whether or
not any agent remembered the rule.

### Decision 3: the CLI reports the vocabulary, the skill exercises the judgement

Convergence of the tag vocabulary — the spec's fourth success metric — depends on a
proposal preferring an existing tag over minting a near-duplicate. That splits
cleanly into a mechanical half and a judgement half. The mechanical half is "what
tags are already in use in the stores in scope", which is a listing the CLI can
produce exactly and cheaply. The judgement half is "does one of them fit this
entry", which is the agent's job and cannot be a rule.

So the CLI gains a `knowledge tags` listing and nothing more, and the
`spek-knowledge` skill's contribute flow loads it exactly as it already loads
`knowledge categories` (`templates/skills/workflows/spek-knowledge/SKILL.md:52`)
before routing an entry to a category by judgement at `:53`. The audit intent is
built from the same three existing primitives — `knowledge list`, `knowledge read`,
`knowledge tags` — and proposes per-entry changes back through the existing
`knowledge write`. No new write path, no bulk operation, and the
propose-then-confirm contract at `:56` is untouched and still load-bearing.

The narrowing flag follows the established shape rather than inventing one: a
repeatable `--tag` registered beside `--tier` and `--filter`
(`cmd/knowledge.go:520-526`) and carried on `knowledge.Selector`
(`internal/knowledge/address.go:41-44`), so narrowing a search stays one idea. The
filter is absolute — an entry lacking the tag is never returned, however well it
scores — which is why it is a filter and not another scoring factor.

### Why this beats the alternatives

The rejected shape, scoring inside each store, fails on provider count. It looks
cheaper, and for a single file store it is, but it makes the ranking formula a
contract every future provider must reproduce exactly rather than a rule the system
applies once. A second implementation that rounds differently, or tunes differently,
or simply forgets the coverage exponent, produces scores that are silently
incomparable in the cross-store merge, and the relative cutoff then means something
different depending on how a project happened to split its knowledge. Reporting
evidence upward costs nothing extra, because the store already collects exactly that
evidence during the single file pass it performs today. On the
ranking side, length normalisation and title weighting were both rejected as
explicit Non-Goals of the spec, leaving the tag weight as the deliberate carrier
of "this entry is about X" — a choice validated against this repo's real corpus,
where it takes all four observed queries to rank 1. A sidecar tag index was
rejected for breaking the no-migration constraint. Full citations and the measured
prototype are in `research.md#alternatives-considered-and-rejected` and
`research.md#chosen-approach-evidence`.

### Conventions this design is bound by

The CLI's **error-messages-must-suggest-remediation** convention drives the shape of
every new refusal: a `--tag` naming a tag no entry carries, or a malformed
frontmatter block, must be `output.NewError(code, msg).WithNextAction(...)` with a
runnable next step — for the tag case, `knowledge tags` is the command that lists
the valid values, which is exactly the information the caller is missing. On the
docs side, **plan-content-pages** requires the documentation phase of this plan to
carry a labelled content block rather than a prose summary, and **no-em-dashes**
and the four **mdx-authoring** rules bind the copy and markup that phase produces.

## Component Breakdown

All components below live in the `spektacular` repo except the last, which lives in
the `docs` repo.

### New components

- **Entry frontmatter reader** *(new, `spektacular`)* — Owns the single question
  "what tags does this entry declare?". Detects a leading fenced YAML block, parses
  a `tags` list out of it, lower-cases and de-duplicates the values, and returns the
  remaining body. An entry with no block is not an error and yields no tags; a block
  that is present but malformed is reported rather than silently swallowed. It is
  deliberately separate from the existing workflow-artifact frontmatter component,
  which owns a different schema with mandatory fields a knowledge entry must not be
  forced to carry. Consumed only by the document scanner.

- **Document scorer** *(new, `spektacular`)* — Owns the per-document scoring
  formula: the tag weight, the damping curve applied to repeated body occurrences,
  and the coverage factor that scales the result by the share of query terms with
  any evidence. It is a pure function over the evidence a store reported, which is
  what makes the weights tunable and unit-testable without touching the filesystem.
  It lives in the knowledge layer, not in any store, so the formula is applied once
  to every result regardless of which provider produced it, and a new store can
  never rank on a different scale. It knows nothing about individual stores,
  cutoffs, or categories. The tunable constants live here as named values, not
  scattered literals, so tuning is a single-site change.

- **Query tokenizer** *(new, `spektacular`)* — Owns splitting a query string into
  the ordered terms everything else indexes by. It moves up to the knowledge layer
  with the scorer, because once evidence is reported as per-term arrays, the store
  and the scorer must agree on term order by construction rather than by convention.
  Tokenization is a retrieval-policy decision, not a storage one.

- **Tag vocabulary lister** *(new, `spektacular`)* — Owns "which tags are already in
  use across the stores in scope". Walks the covered stores, collects the distinct
  tags their entries declare with an occurrence count for each, and returns them
  ordered so the established vocabulary is visibly established. It is purely
  mechanical: it reports what exists and never judges what a new entry should
  carry. Surfaced as a new `knowledge tags` subcommand and consumed by the capture
  and audit flows. It reuses the knowledge set's existing store-narrowing rather
  than introducing its own notion of scope.

### Meaningfully changed components

- **Document scanner** *(changed, `spektacular`)* — Already reads each file exactly
  once, accumulating per-term occurrence counts, the title, the strongest excerpt
  lines and a content checksum in a single pass. It gains two responsibilities:
  invoking the frontmatter reader on the leading portion of that same pass, and
  handing the collected evidence plus the parsed tags to the document scorer. It
  does not gain a second read of any file. Crucially, term counting must now run
  against the entry body rather than the raw bytes, so a tag is not also counted as
  a prose occurrence of itself.

- **Store search** *(changed, `spektacular`)* — Owns walking one store and emitting
  one result per candidate document, together with the evidence that document
  offers. Four changes: it takes the query pre-tokenized rather than splitting it
  itself; it stops excluding a document that is missing some query terms (ranked OR
  replaces boolean AND); it accepts a search options value so a caller can pass the
  tag filter, rather than growing a second method; and it reports each result's tags
  and per-term body counts alongside the existing locator, title, excerpts and
  checksum. It deliberately does **not** compute a score and does **not** apply the
  weak-match cutoff. Both are decisions about the whole result set, which a single
  store cannot see. It may pre-apply the tag filter as an optimisation, since
  skipping a document during the walk is cheaper than reporting it, but correctness
  does not depend on it doing so: the knowledge layer enforces the filter regardless,
  which is what keeps a remote provider from having to be trusted on it.

- **Ignore-aware store decorator** *(changed, `spektacular`)* — Owns excluding
  ignored paths from listing and search. Its only change is to forward the new
  search options through to the store it wraps. Called out because a decorator that
  silently drops an option would make the tag filter appear to work in tests and
  fail against any store carrying an ignore file.

- **Search result** *(changed, `spektacular`)* — The value describing one matching
  document. Gains a tags field so a consumer can see why an entry was returned and
  so the audit flow can read an entry's current tags without a separate fetch. It
  always carries a list, empty rather than absent for an untagged entry, so no
  consumer has to distinguish "no tags" from "no tag support".

- **Knowledge set search** *(changed, `spektacular`)* — Owns fanning a query across
  every covered store, merging the results, ranking them globally, and stamping each
  with the tier, store name and category it came from. It gains three
  responsibilities: tokenizing the query before fanning it out, scoring every
  reported result through the document scorer so one formula covers every provider,
  and applying the weak-match cutoff after the merge and sort, which is the only
  point at which the strongest hit across all stores is known. It also enforces the
  tag filter, whatever a store did or did not do about it. The existing
  deterministic tie-break (store order, then path) and the existing always-applied
  category exclusion are unchanged and must survive alongside the new steps.

- **Store selector** *(changed, `spektacular`)* — The value naming which stores a
  fan-out request covers. Gains the requested tag list. This keeps narrowing a
  single idea travelling on a single value, rather than threading a second
  parameter through every layer. Note the tag list narrows *entries*, whereas the
  existing tier and name fields narrow *stores*; the value carries both axes but
  they are applied at different points.

- **Knowledge CLI** *(changed, `spektacular`)* — Owns the agent-facing command
  surface and its machine-readable schemas. It registers the repeatable `--tag`
  flag on the search command beside the existing narrowing flags, adds the `tags`
  field to the advertised search-result schema, and adds the new `knowledge tags`
  subcommand with its own schema. Every refusal it adds names a runnable next step,
  per the repo's error-message convention.

- **`spek-knowledge` skill template** *(changed, `spektacular`)* — Owns the prose
  contract for the three existing capture and retrieval intents, and gains a fourth.
  The contribute flow loads the tag vocabulary exactly as it already loads the
  category definitions, proposes tags as one more field in the block the user
  confirms, and passes them through on the write. The lookup flow learns that tags
  affect ranking and that `--tag` narrows. The update flow preserves an entry's
  existing tags unless the change is about them. A new audit intent reads entries in
  scope, reports tags an entry's content does not support and tags it is missing,
  and routes every proposed change back through the existing confirmed write, one
  entry at a time. The propose-then-confirm contract is unchanged and remains
  enforced by this prose rather than by a CLI guard.

- **Agent-facing retrieval prose** *(changed, `spektacular`)* — The managed
  knowledge-capture trigger section, the planning workflow's discovery step, and the
  research-agent library skill each describe the shape of a search result and the
  capture offer to a driving agent. They are updated together so an agent is never
  told that a document must contain every query word, and so a capture offer
  mentions the tags it will propose. These are separate files but one concern: the
  description of retrieval given to agents.

- **Maintainer documentation** *(changed, `spektacular`)* — The repo's own knowledge
  base document and README describe search behaviour and list the command surface.
  Both currently state that a document matches only when every query word occurs in
  it, which this change makes false.

- **Published knowledge base page** *(changed, `docs`)* — Owns the user-facing
  explanation of what the knowledge base is and how entries are written and found.
  Its ranking sentence is replaced with an account of the three ranking factors and
  the relative cutoff, the write flow gains tags, and the narrowing prose gains
  `--tag`. It is composed from the site's existing section, prose and config-key
  components; no new component is introduced and no navigation change is needed.

## Data Structures & Interfaces

All contracts below are in the `spektacular` repo. The `docs` repo introduces no
types; its change is content only.

### The on-disk entry format

A knowledge entry may open with a fenced YAML block declaring its tags. The block
is optional, and everything that already reads entries continues to work when it is
absent.

```markdown
---
tags: [go, http, routing]
---

# HTTP routing standard

All Go services route HTTP endpoints through chi.
```

`tags` is the only key this feature reads. An entry with no block, an empty list, or
a block carrying only unrelated keys simply has no tags. This is the whole of the
migration story: there is no version marker, no backfill, and no command to run.

### `SearchOptions` (new)

The narrowing a caller wants applied while a store is walked. It exists so
`Store.Search` gains one options parameter rather than a second method, and so a
later narrowing axis costs no further signature churn.

```go
type SearchOptions struct {
    // Tags restricts results to entries carrying every listed tag.
    // Empty means no tag restriction.
    Tags []string
}
```

The zero value means "no narrowing", so every existing call site keeps its present
meaning. The filter is absolute: an entry lacking a listed tag is never returned,
whatever it scores.

### `Store.Search` (changed signature)

```go
// before
Search(query string) ([]Hit, error)

// after — terms are pre-tokenized by the caller, and the store reports
// evidence rather than a score
Search(terms []string, opts SearchOptions) ([]Hit, error)
```

Taking `terms` rather than `query` is what lets the caller and the store index
evidence by the same term order by construction. The interface has one real
implementation and one pass-through decorator, and no consumer outside the knowledge
layer, so this is a contained change rather than a public break. The decorator must
forward `opts` unchanged.

**The contract a new provider must meet.** To participate in ranking, a store must
report, per returned document, the per-term body occurrence counts and the entry's
tags. A provider that can only return its own opaque relevance score cannot rank
coherently against the others and is out of scope for this design. Say so in the
interface's doc comment, so the next provider's author meets this in the design
rather than in a debugging session.

### `Hit` (changed)

The value describing one matching document gains the entry's tags and the per-term
evidence the store observed. `Score` changes owner rather than type: the store now
leaves it zero and the knowledge layer stamps it, exactly as it already does for
`Tier`, `Name` and `Category`.

```go
type Hit struct {
    // Populated by the store
    Path       string
    Title      string
    Excerpts   []string
    Checksum   string
    Tags       []string // the entry's declared tags; empty, never absent
    BodyCounts []int    // occurrences per query term, indexed like the terms

    // Left zero by the store, stamped by the knowledge layer
    Tier, Name string
    Category   string
    Score      float64 // composite: tag weight + damped body counts, scaled by coverage
}
```

Grouping the fields this way in the struct definition, with the two comments, is
worth doing literally: it is what stops a future provider author from assuming
`Score` is theirs to fill in.

`Tags` serialises as `"tags": []` for an untagged entry rather than being omitted,
so a consumer never has to tell "this entry has no tags" apart from "this build has
no tag support". `BodyCounts` is internal plumbing rather than part of the published
result, so it carries `json:"-"`; the CLI's search output is unchanged apart from
gaining `tags`.

### `Selector` (changed)

The value naming what a fan-out request covers. It gains the requested tags, so
every narrowing axis travels on one value.

```go
type Selector struct {
    Tier   Tier     // which tier(s) of stores
    Filter []string // which store names within the tier
    Tags   []string // which entries within those stores
}
```

The two axes are applied at different points despite travelling together: `Tier` and
`Filter` decide which stores are queried at all, while `Tags` narrows entries within
them. `Selector.covers` remains a store-membership test and must not learn about
tags. The knowledge layer is the authority on the tag filter; a store may pre-apply
it to avoid reporting documents that will be discarded, but is not trusted to.

### Entry frontmatter reader (new contract)

```go
// ParseEntry splits an optional leading YAML frontmatter block from raw,
// returning the entry's declared tags (lower-cased, de-duplicated, order
// preserved) and the remaining body. Raw with no leading block yields no tags
// and raw unchanged, which is not an error.
func ParseEntry(raw []byte) (tags []string, body []byte, err error)
```

The body return is load-bearing rather than incidental: term occurrence counting
runs against it, so a tag is not also counted as a prose mention of itself.

### Document scorer (new contract)

A pure function over the evidence a store reported, with no filesystem access, so the
weights are tunable and testable in isolation. It lives in the knowledge layer beside
the cutoff, so every provider's results pass through the same formula.

```go
// Evidence is what one scanned document offers for one query.
type Evidence struct {
    BodyCounts  []int     // occurrences per query term, indexed like the terms
    TagAffinity []float64 // per term: 0 for no tag match, 1 for an exact tag,
                          // and the prefix partial-credit fraction in between
}

// tagAffinity returns how strongly term matches the best of an entry's tags:
// 1 for an exact match; len(shorter)/len(longer) when one string is a prefix of
// the other and the shorter reaches minPrefixLen; 0 otherwise. Where several
// tags match, the strongest wins rather than accumulating.
func tagAffinity(term string, tags []string) float64

// Score returns the document's composite ranking score, or 0 when the
// document carries no evidence for any term.
func Score(e Evidence) float64
```

The tunable constants (tag weight, damping curve, coverage exponent, minimum prefix
length) are named values beside this function, so tuning is a single-site change
rather than an edit scattered across the scan.

`TagAffinity` is `[]float64` rather than `[]bool` precisely so partial credit is
expressible. An exact match is simply the value 1, so exact matching is the special
case of the general rule rather than a separate code path.

### Tag vocabulary listing (new contract)

What the capture and audit flows read to prefer an established tag over a new one.

```go
// TagUse is one distinct tag and how widely it is already used.
type TagUse struct {
    Tag   string `json:"tag"`
    Count int    `json:"count"` // entries carrying it, across the covered stores
}

func (s *Set) Tags(sel Selector) ([]TagUse, error)
```

Returned most-used first so the established vocabulary is visibly established,
with ties broken alphabetically for determinism.

### CLI surface (changed)

- `knowledge search <query> [--tier T] [--filter N] [--tag G]...` — `--tag` is
  repeatable and registered beside the existing narrowing flags. It must also be
  declared in the command's advertised flag schema, or the agent-facing contract
  goes stale.
- `knowledge tags [--tier T] [--filter N]` — new subcommand, emitting
  `{"tags": [{"tag": "...", "count": N}, ...]}`, with its own input/output schema.
- The search result schema gains `tags` as an array of strings.
- `knowledge write` is **unchanged**. Tags are part of the entry body's frontmatter,
  so they travel in the staged file the skill already writes; no change to the
  write payload, and therefore no change to the propose-then-confirm contract.

## Implementation Detail

### One new pattern: stores gather evidence, one layer above them ranks

Today the ranking rule is three lines embedded in the middle of a directory walk,
where the decision to exclude a document and the decision to score it are the same
statement, and both live inside a single provider. This plan separates *gathering*
evidence from *judging* it, and puts the two on opposite sides of the store
interface. A store reports what it found; the knowledge layer decides what that is
worth.

This is the one genuinely new pattern the plan introduces, and it is doing two jobs.
It makes the spec's instruction to tune the weights during implementation practical:
a developer tuning the tag weight or the damping curve edits named constants beside a
pure function they can exercise with a table test, rather than constructing a temp
directory to watch a number move. And it is what keeps ranking coherent as providers
multiply. The project already anticipates more of them, and a formula duplicated
across providers is a formula that will drift; a formula applied once above them
cannot. A developer adding a provider implements "find and describe", never "find,
describe and rank", and the interface's doc comment should say so in as many words.

The cost is that `Hit` becomes partly an intermediate value rather than purely a
result: it carries per-term counts that no consumer outside the ranking step reads.
That is a real, if small, loss of tidiness, accepted deliberately in exchange for
never having two providers disagree about what a score means.

Everything else follows existing patterns in the codebase. The options-struct
parameter on the store search mirrors how the codebase already extends
agent-facing surfaces without multiplying methods. The new tag listing follows the
shape of the existing category listing: a mechanical enumeration the CLI reports and
the skill exercises judgement over. The new subcommand follows the established
schema-carrying command shape, including a `--schema` mode.

### A code-shape change worth calling out: exclusion becomes ranking

Removing boolean AND is a small edit with a wide behavioural blast radius. The walk
currently returns early from its callback when any term's count is zero, which means
"missing a term" and "not a match" are the same code path. Afterwards they are
different: a document with evidence for one term of three is a real result with a low
score, and only the cutoff decides whether it is shown. A developer reading the
changed walk will see a function that no longer rejects anything, with rejection
having moved to a single place downstream. That relocation is the whole point, and
it is worth a comment at both ends so the next reader does not "restore" the early
return as an optimisation.

The consequence for existing tests is that several stop being about the same thing.
Tests currently asserting which files come back for a multi-term query are now
asserting a *ranking* with a cutoff, and tests asserting exact scores are asserting
a formula that has changed. These are expected breakages produced by the change
itself. Each hand-written expected value must be recomputed by hand from the fixture
and the documented formula, never regenerated by running the new code and recording
what it printed, or the tests stop being an independent check on the thing they
test.

### Frontmatter parsing without a second file read

The scan is a single streaming pass that also computes a content checksum, so
parsing tags cannot mean reading the file again. The frontmatter block is by
definition the first thing in a file, so tag extraction happens in the opening lines
of the pass already underway, and the scan transitions from "reading frontmatter"
to "reading body" once the closing fence is seen. Term counting starts at that
transition. The checksum continues to cover the file's exact raw bytes including the
frontmatter, since it identifies the file rather than its prose.

A file with no leading fence takes the body path from its first line, which is the
overwhelmingly common case today and must stay free of any added cost.

### Two frontmatter readers, deliberately

The codebase will have two frontmatter parsers after this change, which looks like
duplication and is not. The existing one owns the workflow-artifact schema, where
`created_date` and a valid `status` are mandatory and an entry lacking them is
genuinely malformed. The new one owns the knowledge-entry schema, where every field
is optional and a missing block is normal. Sharing them would mean either forcing
artifact fields onto knowledge entries, which the no-migration constraint forbids,
or loosening artifact validation, which would let a malformed spec through silently.
The shared part is about fifteen lines of block detection; the divergent part is the
schema and its validation, which is the part that matters. Whichever is done, the
new reader's doc comment should say why it is not the other one, because this is
exactly the kind of apparent duplication a future contributor will try to helpfully
consolidate.

### Skill prose gains an intent, and the capture flow gains a step

The skill is a static playbook, not a CLI state machine, so its change is prose
discipline rather than code. The contribute flow gains one step in an established
rhythm it already follows for categories: load what exists, choose from it by
judgement, show the choice, wait for confirmation. Tags slot into the block the user
already confirms rather than adding a second confirmation, which keeps the number of
approval gates unchanged.

The audit intent is the one genuinely new flow, and its shape is deliberately
conservative: it composes only existing read primitives and the existing confirmed
write, proposes per entry rather than in bulk, and has no command of its own. A
reviewer should check specifically that it cannot write without passing through the
same confirmation the other intents use, since that contract is enforced by prose
and not by a guard in the CLI.

### The generated-copy discipline

Skill and agent instruction files are templates that are rendered and installed into
per-agent directories, and those rendered copies are committed to the repository.
Editing a template without regenerating leaves the repository internally
inconsistent in a way nothing fails on until an agent reads the stale copy. Any phase
that touches a template therefore ends with regeneration and the regenerated files
being committed alongside the template, and the existing guard tests that assert on
rendered content are the check that this happened.

## Dependencies

**No new third-party dependency is introduced, in either repo.** Everything this
plan needs is already vendored and in use.

### External libraries (`spektacular`) — all already present, none need changing

- **`gopkg.in/yaml.v3`** — parses the tag frontmatter block. Already a direct
  dependency, used by the workflow-artifact frontmatter reader and by config
  loading. No version change.
- **`github.com/cbroglie/mustache`** — renders the `{{command}}` placeholder when
  skill and agent-instruction templates are installed. Needed unchanged by every
  template phase.
- **`github.com/stretchr/testify`** — the assertion library every test in the repo
  uses. Unchanged.
- **`github.com/sabhiram/go-gitignore`** — backs the ignore-aware store decorator
  that must forward the new search options. Unchanged; listed because the decorator
  is on the critical path and a silently-dropped option there would be invisible in
  tests that use a store with no ignore file.
- **Go standard library `math`** — supplies the logarithm for the damping curve.
  New import in one package; no module change.

### Internal packages (`spektacular`)

- **`internal/store`** — owns the per-file scan, the search interface, and the hit
  value. **Changes**: gains the options parameter, the tags field, ranked-OR
  matching, frontmatter-aware scanning, and the scoring function. This is the
  largest change in the plan.
- **`internal/knowledge`** — owns the multi-store set, the address and selector
  values, the category registry, and cross-store ranking. **Changes**: the
  weak-match cutoff, the tag field on the selector, and the tag vocabulary listing.
- **`cmd`** — owns the cobra command surface and the machine-readable schemas.
  **Changes**: the `--tag` flag, the new `knowledge tags` subcommand, and the
  updated search schema.
- **`internal/output`** — provides the error type carrying a remediation next
  action. **Used unchanged**; every new refusal must be built with it, per the
  repo's error-message convention.
- **`internal/metadata`** — owns the workflow-artifact frontmatter schema.
  **Explicitly not used and not changed.** It is listed here because it is the
  obvious-looking dependency that must be avoided: its parser requires a
  `created_date` and a valid `status`, and its renderer drops unknown keys, so
  routing knowledge entries through it would reject a tags-only block and delete
  tags on any round trip.
- **`templates`** — the embedded template tree. **Changes**: the `spek-knowledge`
  skill, the managed knowledge-capture trigger section, the plan discovery step,
  and the research-agent library skill.
- **`internal/agent`** — owns skill registration, `{{command}}` rendering, and
  per-agent installation. **No source change expected**, but it is the mechanism
  every template phase depends on: template edits reach agents only by re-running
  init, and its guard tests are what catch a missed regeneration.

### Planning dependencies

- **Spec `000050_knowledge-entry-tags`** — the source of this plan. Complete and
  closed; nothing outstanding.
- **Knowledge entry `decisions/ranked-or-over-boolean-and.md`** — **a hard
  prerequisite in substance, already satisfied on paper.** It is the binding record
  that search must rank rather than exclude, that coverage scales the score, and
  that the cutoff is relative rather than absolute. The code has not yet caught up
  to it, so this plan implements it. Note this entry is currently untracked in git;
  it should be committed rather than left uncommitted while work proceeds against
  it.
- **Prior plan `000028_knowledge-base-categories-tiers-and-dedup`** — landed.
  Established the category registry, the always-applied/looked-up split, and the
  exact-byte de-dup layer this plan must not disturb. The tag listing follows its
  single-declaration discipline.
- **Prior plan `000047_repo-scoped-knowledge-addressing`** — landed. Established the
  address and selector values and the `--tier`/`--filter` narrowing shape that
  `--tag` extends.
- **Prior spec `000022_spek-knowledge-skill`** — landed. Established the
  propose-then-confirm capture contract tags must slot into without weakening.

**Nothing must land before this plan starts.** All planning dependencies are
already delivered.

### Cross-repo dependency (`docs`)

- The documentation change in the `docs` repo depends on the `spektacular` engine
  and CLI changes being settled, because it documents their observable behaviour:
  the ranking factors, the exact `--tag` syntax, and the tag frontmatter shape.
  Sequence the documentation phase after the behaviour it describes is final,
  otherwise the published page describes weights that then get tuned.
- The `docs` repo itself needs **no dependency, tooling or configuration change**.
  Its existing Astro build and `astro check` are the verification, and the page is
  composed from components that already exist.

## Testing Approach

### Strategy

Testing concentrates on the scoring function, because that is where every behavioural
requirement in the spec actually lives and because it is the one component that can
be exercised without touching a filesystem. Most acceptance criteria are statements
about *relative* rank between two comparable entries, which makes them expressible as
small, readable table cases rather than as elaborate fixtures.

Four layers, in decreasing density of coverage:

1. **Unit tests over the scorer.** Pure function, no I/O. Each of the four ranking
   behaviours the spec names gets direct cases: a tag hit outweighs a prose
   occurrence, coverage of more query terms scores higher, repeated occurrences
   damp, and a document with evidence for nothing scores zero. The tag-affinity rule
   gets its own table alongside, since prefix partial credit is where a subtle mistake
   is most likely and least visible: the pairs that must match and at what strength,
   the near-miss pairs that must score nothing at all, and both of the guards that keep
   a short term from matching everything. These are cheap enough to be exhaustive about
   boundaries.
2. **Unit tests over the frontmatter reader.** Entry with tags, entry with no block,
   entry with an empty list, entry with a block carrying only unrelated keys, entry
   with a malformed block, and a tag list needing case-normalisation and
   de-duplication. The load-bearing case is the one with **no block at all**, since
   that is every entry that exists today.
3. **Store-level tests over the scan.** These prove the scorer is wired to real
   files correctly: that tags are read from disk, that a tag is not double-counted as
   a body occurrence, that ranked OR returns partial matches, and that the tag filter
   excludes absolutely. Fixtures are written into a temp directory the test owns,
   following the existing convention in this package.
4. **Knowledge-layer and CLI tests.** The cross-store cutoff (which cannot be tested
   below this layer), the tag vocabulary listing, and the CLI's advertised schema and
   flag surface. Plus rendered-template guard tests asserting the skill prose carries
   its new invocations, following the existing pattern of rendering through the real
   install path into a temp directory rather than reading the committed copy.

### What the tests guarantee, in plain language

- An entry tagged with a word is found by searching that word, even when the entry's
  prose never uses it.
- A near-miss on a tag is scored as a near-miss rather than as either a miss or a full
  hit: a search for `http` finds a `https` tag at reduced strength, `apple` and
  `apples` find each other, and `test` finds nothing at all in a `rest` tag.
- Between two otherwise comparable entries, the one tagged with the query term ranks
  above the one that merely mentions it.
- An entry matching some query terms is returned and ranked, not silently dropped.
- An entry matching more of the query outranks one matching less of it.
- A weak match is returned when it is the only thing there, and disappears once
  something strong is present, and this holds when the strong and weak matches live
  in **different stores**.
- Narrowing by tag is absolute: an entry without the tag is never returned, whatever
  it would otherwise score.
- A knowledge base whose entries carry no tags behaves exactly as it does today, with
  no migration and no command run.

### Existing tests that change rather than being added to

Several existing tests assert the old ranking rule directly: exact occurrence-sum
scores, and which files a multi-term query returns under boolean AND. These are
expected breakages caused by this change, not incidental failures, and the plan
treats fixing them as in-scope work rather than as collateral. Every recomputed
expected value is worked out **by hand** from the fixture and the documented formula.
None is produced by running the new implementation and recording its output, which
would leave the test asserting only that the code does what it does. This follows the
repo's established convention of hand-maintained oracles independent of the subject
under test.

The repo convention that all tests must pass before work is called done applies with
particular force here, since this change deliberately invalidates existing
assertions; none may be dismissed as pre-existing.

### Where tests slot in

New tests follow the conventions already in place: table-driven cases with
`require`-style assertions, fixtures built into a test-owned temp directory rather
than read from any real directory on the machine, and hand-written expected values.
Skill and template assertions render through the production install path into a
scratch directory, never reading a committed generated copy, so a stale copy cannot
make a test pass.

### Deliberate gaps

- **No benchmark against the live knowledge directory.** The observed-query benchmark
  runs against a controlled fixture instead. Research showed exactly why: adding one
  real entry to the live corpus (the decision record that quotes all four benchmark
  queries verbatim) moves the target entry from rank 1 to rank 2. A test bound to a
  directory that grows as the project writes knowledge would flap for reasons
  unrelated to the code.
- **No end-to-end agent test of the capture flow.** The propose-then-confirm contract
  is enforced by skill prose, not by a CLI guard, and there is no harness in this
  repo that drives an agent through a confirmation. Coverage stops at asserting the
  rendered prose carries the required instructions and that the CLI primitives it
  composes behave correctly.
- **No performance test.** The change adds a logarithm per term per document and a
  frontmatter parse of a file's opening lines, against a scan that already reads
  every byte and computes a SHA-256. The cost is not plausibly material and a test
  would only pin noise.
- **No test of the docs site's content.** Its build and typecheck are the check; the
  prose itself is verified by review.

### Success metrics: how each is verified

The spec names four.

1. **"The `api` case reverses."** Searching a plausible surface name that appears
   nowhere in an entry's prose returns that entry on tags alone; today `api` returns
   nothing for the routing entry.
   → **Behavioural test.** A fixture entry tagged `api` whose body never contains the
   string "api", asserted to be returned for the query `api`. This is the single
   sharpest test in the plan, because it fails under every part of the old design and
   passes only when tags genuinely carry retrieval on their own.

2. **"The observed-query benchmark goes from 0 to 4."** Against a knowledge base, the
   entry actually about a topic ranks first for all four queries agents have been
   observed to use for it: `http endpoints go`, `http handler`,
   `http endpoints handler routing`, and `api`.
   → **Behavioural test.** A dedicated benchmark test over a controlled fixture that
   reconstructs the measured scenario: the on-topic routing entry, tagged as the
   capture flow would propose, alongside the long incidental-mention document that
   beat it in the original measurement. It asserts the on-topic entry ranks **first**
   for all four queries, as a table of query→expected-first-path. Research validated
   this is achievable: a prototype of the proposed formula put the on-topic entry
   first for all four. The test is the guarantee that later weight tuning does not
   quietly give the metric back.

3. **"Newly captured entries carry tags."** Entries written through the capture flow
   after this ships have at least one tag, without anyone having designed a taxonomy.
   → **Manual — captured in the implementation test plan.** This is a property of how
   an agent behaves when driving the skill over real work, observed over time. What
   *can* be asserted automatically, and will be, is the precondition: that the
   rendered skill prose instructs the agent to load the existing vocabulary, propose
   tags, and show them for confirmation before writing. That guard is a behavioural
   test; the outcome it is meant to produce is the manual observation.

4. **"The tag vocabulary converges rather than fragments."** As entries accumulate,
   most new entries reuse existing tags, distinct tags grow visibly slower than
   entries, and near-duplicates such as `http` / `HTTP` / `http-api` do not
   accumulate.
   → **Manual — captured in the implementation test plan.** Convergence is a trend
   across many capture events over time and cannot be asserted in a single run. Two
   supporting properties are covered by behavioural tests: that tags are normalised
   to lower case on read, so `http` and `HTTP` can never both exist as distinct tags;
   and that the vocabulary listing returns tags ordered by how widely they are
   already used, which is the signal the capture flow relies on to prefer an
   established tag. The semantic near-duplicate case (`http-api` beside `http`) is
   agent judgement and stays manual.

No success metric is dropped.

## Milestones & Phases

Four milestones, each independently deliverable and building on the one before.

### Milestone 1: Search finds the entry that is actually on topic

**What changes.** Searching the knowledge base stops requiring an entry to contain
every word of the query and starts ranking by how good a match it is. An entry
carrying evidence for some of the query is returned and ranked rather than silently
dropped, an entry covering more of the query outranks one covering less, and
repeating a word many times stops being a way to climb the results. Weak matches
show up when nothing better exists and fall away once something stronger is present,
including when the stronger match lives in a different knowledge store. For anyone
using the tool, the visible difference is that a search which used to return nothing,
or return a long tangentially-related document first, now surfaces the entry that is
actually about the subject. Nothing needs to be added to the knowledge base for this
to take effect, and no entry changes.

**Validation point.** Two comparable entries, one covering all query terms and one
covering a single term, come back in that order; the single-term entry is returned
when searched alone and absent once the fuller match is present; and a long document
repeating a term does not outrank a shorter one on volume alone. Every pre-existing
test in the two affected packages passes with its expected values recomputed by hand.

#### - [x] Phase 1.1: Move ranking above the storage layer

**Repo:** spektacular

Today the rule that decides whether a document matches and how highly it scores is a
few lines buried in the middle of one storage provider's directory walk, where
"missing a term" and "not a match" are the same statement. This phase separates
finding from judging: a store reports what evidence each document offers, and a
single function one layer above it turns that evidence into a score. Nothing about
the observable behaviour changes yet, and every existing test still passes unchanged.
It earns its own phase because it is what makes the next two phases small, what makes
the weights tunable as the spec expects, and what means a future storage provider
never has to reimplement the ranking formula to rank consistently with the others.

*Technical detail:* [context.md#phase-11](./context.md#phase-11-move-ranking-above-the-storage-layer)

**Acceptance criteria**:

- [x] Searching behaves exactly as it did before this phase, with identical results and identical scores for every query.
- [x] The rule that turns a document's matches into a score lives in one place that can be exercised directly, without creating any files.
- [x] A storage provider reports what it found and never computes a score, so adding a provider cannot introduce a second ranking scale.
- [x] Every pre-existing test in the affected package passes without its expected values being changed.

#### - [x] Phase 1.2: Rank partial matches instead of discarding them

**Repo:** spektacular

Search stops requiring every query word to appear in a document. An entry carrying
evidence for some of the query is now returned and ranked rather than dropped, and
its score is scaled by the share of the query it actually covers, so an entry
matching one word of three ranks far below one matching all three. Repeated
occurrences of a word also start giving diminishing returns, so a long document
cannot climb the results by mentioning a term over and over. This is the change that
makes a search which previously returned nothing return the closest thing available.

*Technical detail:* [context.md#phase-12](./context.md#phase-12-rank-partial-matches-instead-of-discarding-them)

**Acceptance criteria**:

- [x] An entry with evidence for one query term out of three is returned rather than omitted.
- [x] Between two comparable entries, the one with evidence for all three query terms ranks above the one with evidence for a single term.
- [x] A document that mentions a term twenty times does not score ten times higher than one that mentions it twice.
- [x] Existing tests that asserted the old occurrence-sum scores carry new expected values worked out by hand from the fixtures, not copied from what the new code prints.

#### - [x] Phase 1.3: Drop weak matches relative to the strongest hit

**Repo:** spektacular

With partial matching in place a query returns far more results, most of them barely
relevant, so this phase adds the cutoff that keeps the result list useful. A hit
scoring far below the best hit in the set is dropped, which means a weak match still
surfaces when it is the only thing available and disappears once something genuinely
relevant is present. The cutoff is expressed as a share of the strongest hit rather
than as a fixed number, so it keeps its meaning as a knowledge base grows. It is
applied after results from every store have been brought together, so a weak hit
cannot survive merely because its own store held nothing better.

*Technical detail:* [context.md#phase-13](./context.md#phase-13-drop-weak-matches-relative-to-the-strongest-hit)

**Acceptance criteria**:

- [x] An entry matching one of three query terms is returned when it is the only match.
- [x] That same entry is not returned once an entry matching all three query terms is also present.
- [x] The weak match still disappears when the strong match lives in a different knowledge store from it.
- [x] Results remain in a stable, repeatable order across identical searches.

### Milestone 2: Entries can say what they are about, and be found by it

**What changes.** A knowledge entry can declare tags describing its subject, and
those tags retrieve it. An entry tagged `go, http` is returned for the query
`go http router` whether or not those words appear anywhere in its text, and between
two otherwise comparable entries the tagged one ranks above the one that merely
mentions the word in passing. A search can also be narrowed to entries carrying given
tags, in which case an untagged or differently-tagged entry is never returned however
well it would otherwise score. Entries that carry no tags remain completely valid and
searchable, and upgrading requires no migration, no backfill, and no command to run.
This is the milestone that decouples what an entry is about from which words it
happens to use, which is the whole point of the feature.

**Validation point.** An entry whose body contains neither word, tagged `go, http`,
is returned for `go http router`; a tagged entry outranks a prose mention of the same
word; a tag-narrowed search returns only entries carrying that tag; and a knowledge
base of entirely untagged entries behaves exactly as it did before, with every
command running unchanged. The four queries agents have actually been observed to use
for one subject all rank the on-topic entry first, against the benchmark fixture.

#### - [x] Phase 2.1: Read tags from an entry

**Repo:** spektacular

An entry can now open with a short block declaring its tags. This phase teaches the
reader to recognise that block, take the tags out of it, and hand back the rest of
the entry as the body. An entry that has no such block is not an error and simply has
no tags, which is every entry that exists today, so nothing needs migrating and no
command needs running. Tags are normalised to lower case as they are read, so the same
tag written two ways can never end up as two tags.

*Technical detail:* [context.md#phase-21](./context.md#phase-21-read-tags-from-an-entry)

**Acceptance criteria**:

- [x] An entry declaring tags reports exactly those tags, normalised to lower case and with duplicates removed.
- [x] An entry with no tag block, or with an empty tag list, is read successfully and reports no tags.
- [x] Every existing knowledge command runs unchanged against a knowledge base whose entries carry no tags, and no entry is rejected as invalid.
- [x] The tag block itself is not treated as part of the entry's prose, so a tag is not also counted as a mention of itself.

#### - [x] Phase 2.2: Let tags retrieve and rank an entry

**Repo:** spektacular

Tags become a retrieval signal. A query term matching one of an entry's tags counts
far more heavily than the same word appearing in its text, which means an entry tagged
`go, http` is found by searching `go http router` even when its body contains neither
word, and between two otherwise comparable entries the tagged one comes first. A tag
does not have to be typed exactly: a term and a tag that are prefix-related still
count, at a strength proportional to how much of the longer word the shorter one
covers, so a search for `http` finds a `https` tag at four fifths strength and `apple`
and `apples` find each other without either being written twice. Each result reports
the tags of the entry it came from, so a reader can see why it was returned.

*Technical detail:* [context.md#phase-22](./context.md#phase-22-let-tags-retrieve-and-rank-an-entry)

**Acceptance criteria**:

- [x] An entry tagged `go, http` whose body contains neither word is returned for the search `go http router`.
- [x] Given two entries of comparable length, one tagged `http` and one merely mentioning "http" in its body, a search for `http` returns the tagged entry first.
- [x] A long entry mentioning a term many times does not outrank a short entry tagged with that term.
- [x] An entry tagged `https` is returned for a search for `http`, and scores lower than it would for a search for `https`.
- [x] An entry tagged `apple` is returned for a search for `apples`, and an entry tagged `apples` is returned for a search for `apple`, without either form being written down twice.
- [x] An entry tagged `https-security` is returned for a search for `https`, ranked below an entry tagged `https` exactly.
- [x] A search term that merely resembles a tag without sharing its opening earns nothing from it: `test` gains nothing from a `rest` tag, and `cors` gains nothing from a `core` tag.
- [x] A two- or three-letter search term matches a tag only exactly, so `go` gains nothing from a `golang` tag.
- [x] Where a term is prefix-related to more than one of an entry's tags, the strongest match decides the score rather than the matches adding together.
- [x] Every search result reports the tags of the entry it came from, reporting an empty list rather than nothing at all for an untagged entry.

#### - [x] Phase 2.3: Narrow a search to tagged entries

**Repo:** spektacular

A search can be restricted to entries carrying particular tags, following the same
shape as the existing options for narrowing to a tier or to named stores, so
narrowing a search stays one idea rather than two. The restriction is absolute: an
entry that lacks the tag is never returned no matter how well it would otherwise
score. Asking for more than one tag narrows further rather than widening, since the
point is to restrict.

*Technical detail:* [context.md#phase-23](./context.md#phase-23-narrow-a-search-to-tagged-entries)

**Acceptance criteria**:

- [x] A search narrowed to the tag `http` returns only entries carrying that tag.
- [x] An entry without that tag is never returned by such a search, regardless of its score.
- [x] Asking for two tags returns only entries carrying both.
- [x] The narrowing option is discoverable from the command's own published description of itself, alongside the existing narrowing options.

#### - [x] Phase 2.4: Prove the observed queries now find the right entry

**Repo:** spektacular

The spec's central promise is that four specific searches, which agents were actually
observed using for the same subject on different runs, should all surface the entry
that is about that subject. This phase builds a fixture reproducing the measured
situation, including the long tangential document that used to beat the on-topic
entry, and pins the outcome as a test. It exists so that later tuning of the weights
cannot quietly give the improvement back.

*Technical detail:* [context.md#phase-24](./context.md#phase-24-prove-the-observed-queries-now-find-the-right-entry)

**Acceptance criteria**:

- [x] For each of the four observed queries, the entry actually about the subject is returned first.
- [x] A search for a plausible name that appears nowhere in the entry's prose returns that entry on the strength of its tags alone.
- [x] The check runs against a self-contained fixture, so adding or editing real knowledge entries cannot make it start or stop passing.

### Milestone 3: Tags get applied without anyone designing a taxonomy

**What changes.** When an entry is captured, tags are proposed alongside the tier,
store, path and content already proposed, and shown for confirmation or editing
before anything is written. The proposals are drawn from the tags already in use in
the stores in scope, so a new entry about HTTP reuses an existing `http` rather than
minting a near-duplicate like `HTTP` or `http-api`, and the vocabulary converges as
the knowledge base grows instead of fragmenting. Existing entries can be audited the
same way: the assistant reviews entries in scope, reports tags an entry's content
does not bear out, and proposes tags it is clearly missing, one entry at a time. As
with everything else in the knowledge base, nothing is written without explicit
confirmation. This milestone is what makes milestone 2 pay off in practice, since
tags only improve retrieval if they actually get applied.

**Validation point.** A capture shows proposed tags together with the destination and
waits for confirmation before writing; a capture into a store already using `http`
proposes `http` rather than a variant; the audit flags a tag an entry does not support
and proposes one it is missing; and no entry is modified until the user confirms.

#### - [x] Phase 3.1: Report the tag vocabulary already in use

**Repo:** spektacular

Before anything can prefer an existing tag over a new one, it has to be able to see
what already exists. This phase adds a way to ask which tags are in use across the
stores in scope and how widely each is used, ordered so that the established
vocabulary is visibly established. It reports only what is there and never judges what
a new entry ought to carry; that judgement belongs to the capture flow in the next
phase.

*Technical detail:* [context.md#phase-31](./context.md#phase-31-report-the-tag-vocabulary-already-in-use)

**Acceptance criteria**:

- [x] Asking for the tag vocabulary returns each distinct tag in use with a count of how many entries carry it.
- [x] The most widely used tags come first, with a stable order between equally used ones.
- [x] The listing honours narrowing to a tier or to named stores exactly as the other listing commands do.
- [x] A knowledge base whose entries carry no tags reports an empty vocabulary rather than failing.

#### - [x] Phase 3.2: Propose tags when an entry is captured

**Repo:** spektacular

Capturing an entry now proposes tags for it, alongside the destination and content it
already proposes. The proposals are drawn from the vocabulary already in use, so an
entry about HTTP written into a store that already uses `http` is proposed `http`
rather than a near-duplicate like `HTTP` or `http-api`, and a new tag is proposed only
where nothing existing fits. Prefix partial credit means the flow no longer has to
guess at word forms, since a tag and a search term that share an opening find each
other automatically; what it must still get right is not collapsing genuinely distinct
subjects into one tag, and carrying a second tag where a form differs by more than its
ending. The tags are shown for confirmation or editing together
with everything else, and as before nothing is written until the user explicitly
agrees. This is what makes tags actually get applied, without anyone being asked to
design a taxonomy first.

*Technical detail:* [context.md#phase-32](./context.md#phase-32-propose-tags-when-an-entry-is-captured)

**Acceptance criteria**:

- [x] When an entry is captured, proposed tags are displayed together with the tier, store and path, before anything is written.
- [x] No entry reaches a knowledge store until the user confirms the proposal, tags included.
- [x] Capturing an entry about HTTP into a store already using the tag `http` proposes `http` rather than a variant such as `HTTP` or `http-api`.
- [x] An entry about a subject that merely resembles an existing tag gets its own tag: an entry about HTTPS is tagged `https` even though `http` is already in the vocabulary, because they are different subjects rather than two spellings of one.
- [x] Singulars and plurals are not proposed as separate tags, because prefix matching already relates them; a second tag is proposed only where a form differs by more than its ending, such as `route` beside `routing`.
- [x] Updating an existing entry keeps the tags it already has, unless the change is about the tags themselves.
- [x] Searching is described to agents in a way that reflects how results are now ranked and narrowed, rather than the old rule that every query word had to appear.

#### - [x] Phase 3.3: Audit the tags on existing entries

**Repo:** spektacular

Entries written before tags existed, or tagged carelessly, can now be reviewed. The
assistant reads the entries in scope, reports any tag an entry's content does not bear
out, and proposes tags the entry is clearly missing, again preferring a tag already in
use over a new one. Changes are proposed one entry at a time and nothing is modified
until confirmed, so a bad suggestion costs a "no" rather than a cleanup.

*Technical detail:* [context.md#phase-33](./context.md#phase-33-audit-the-tags-on-existing-entries)

**Acceptance criteria**:

- [x] An entry carrying a tag its content does not bear out has that tag reported as unsupported.
- [x] An entry whose content is clearly about a subject it carries no tag for has that tag proposed, preferring one already in use.
- [x] A tag that is a genuinely distinct term from a similar-looking existing tag is never reported as unsupported or proposed for merging: an entry tagged `https` is not told to use `http` instead.
- [x] A tag made redundant by prefix matching, such as `apples` sitting beside `apple`, is reported as removable rather than left to accumulate.
- [x] No entry is modified until the user confirms the proposed changes for it.
- [x] Changes are proposed per entry, so accepting one entry's changes never silently applies another's.

### Milestone 4: The behaviour is written down where people will find it

**What changes.** The published documentation explains how results are ranked, what
tags contribute to that ranking, and how to narrow a search by tag, replacing the
current description which says a document matches only when every query word occurs
in it. The project's own maintainer documentation and command reference are brought
in line at the same time. The reference page for people writing their own storage
backend is corrected too, since this work changes what a backend must provide and, in
particular, establishes that a backend never scores its own results. Finally the
design of the ranking itself is recorded as a knowledge entry so a future planning run
retrieves it rather than re-deriving it from the code. Without this milestone the
feature works but nobody can tell that it does, the published documentation actively
describes behaviour that no longer exists, and anyone writing a storage backend builds
against a contract that has moved.

**Validation point.** The published page states the ranking factors, the effect of a
tag, and the tag-narrowing syntax; the backend reference states what a backend must
report and that scoring is not its job; and the documentation site builds and
typechecks clean. The project's knowledge base contains an entry describing the
ranking behaviour, and searching for that subject returns it, which also serves as a
real-world exercise of the feature this plan delivers.

#### - [x] Phase 4.1: Correct the project's own documentation

**Repo:** spektacular

The project's maintainer documentation and its command reference currently state that
a document matches only when every query word occurs in it, which this work makes
false. This phase brings them in line: the ranking factors, the effect of a tag, the
tag block on an entry, and the new ways to list the vocabulary and narrow a search.
It is separated from the published site because it lives in a different repository and
carries no build step.

*Technical detail:* [context.md#phase-41](./context.md#phase-41-correct-the-projects-own-documentation)

**Acceptance criteria**:

- [x] No remaining statement in the project's documentation claims a document must contain every query word.
- [x] The command reference lists the new way to see the tag vocabulary and the new way to narrow a search by tag.
- [x] The shape of an entry's tag block is shown, along with the fact that an entry without one is perfectly valid.

#### - [x] Phase 4.2: Publish the retrieval documentation

**Repo:** docs

The public documentation site explains how results are ranked, what tags contribute to
that ranking, and how to narrow a search by tag. This replaces the current sentence
saying results are scored by how often the query terms appear, which stops being true
the moment this work ships. Users of the tool have no other place to learn that tags
exist or that they can search by them.

*Technical detail:* [context.md#phase-42](./context.md#phase-42-publish-the-retrieval-documentation)

**Content outline** — two new sections on the knowledge base page, placed immediately
after "The lifecycle of an entry". The first is `surface`, the second takes the default
plain background, which preserves the page's existing alternation with no changes
further down. Structure and technical facts below are fixed by this plan's research;
the wording is illustrative, and the constants must be taken from the shipped code
rather than copied from here.

**Section A: "How a search is ranked"** (`surface`)

1. *Opening paragraph, replacing the current ranking sentence in the lifecycle
   section.* Establishes that matching is no longer all-or-nothing:

   > `knowledge search` runs a keyword query across the stores the request covers and
   > returns ranked, one-per-document hits. An entry does not have to contain every
   > word of your query: it is returned if it carries evidence for any of them, and
   > ranked on how good a match it is.

2. *How a query becomes terms.* Short, and it sets up everything below:

   > Your query is lower-cased and split on whitespace into terms. There is no
   > stemming, no synonyms and no plural handling, so `router` and `routers` are
   > different terms.

3. *What counts as evidence, per term per entry.* Two kinds, and the distinction is
   the whole point of the feature:

   > For each term, an entry offers at most two kinds of evidence:
   >
   > - a **tag match**: full strength when the term equals one of the entry's tags,
   >   and reduced when the two are prefix-related, described under partial credit
   >   below.
   > - **body occurrences**, the number of case-insensitive substring matches in the
   >   entry's text. The frontmatter block is not part of the text, so a tag is never
   >   also counted as a mention of itself.

4. *The formula*, as a fenced block. This is the part the section exists for:

   > ```text
   > damp(n)        = 0                        if n = 0
   >                = 1 + log2(n)              otherwise
   >
   > tagAffinity(t) = 1                         if t equals one of the tags
   >                = len(shorter)/len(longer)  if t and a tag are prefix-related
   >                                            and the shorter is >= MIN_PREFIX
   >                = 0                         otherwise
   >
   > termScore(t)   = TAG_WEIGHT * tagAffinity(t) + damp(bodyCount(t))
   >
   > coverage       = (terms with any evidence) / (total terms)
   >
   > score          = sum(termScore(t) for t in terms) * coverage ^ COVERAGE_EXPONENT
   > ```
   >
   > with `TAG_WEIGHT = 8`, `COVERAGE_EXPONENT = 2` and `MIN_PREFIX = 4`. Where a term
   > is prefix-related to more than one tag, the strongest match wins; affinities do
   > not add up.

5. *A worked example*, which is what makes the formula legible rather than decorative.
   Use a two-entry comparison for the query `http routing`:

   > Searching `http routing` against two entries:
   >
   > | | Entry A, tagged `http, routing` | Entry B, untagged, mentions "http" 8 times |
   > |---|---|---|
   > | `http` | tag match, 0 in body: 8 + 0 = 8 | no tag, 8 in body: 0 + (1 + log2 8) = 4 |
   > | `routing` | tag match, 2 in body: 8 + 2 = 10 | no evidence: 0 |
   > | coverage | 2 of 2 = 1.0 | 1 of 2 = 0.5 |
   > | **score** | (8 + 10) * 1.0² = **18** | 4 * 0.5² = **1** |
   >
   > Entry B mentions the word far more often and still ranks well below Entry A,
   > because Entry A is *about* both terms and Entry B is about neither.

6. *A paragraph on the cutoff*, explaining behaviour a reader will otherwise find
   surprising (a result appearing, then vanishing as they refine a query):

   > Once every store has been searched and the results merged, any hit scoring below
   > `CUTOFF_FRACTION` of the best hit is dropped, with `CUTOFF_FRACTION = 0.25`. The
   > threshold is relative rather than fixed, so a loosely related entry surfaces when
   > nothing better exists and falls away once something genuinely relevant is
   > present, and it keeps its meaning as a knowledge base grows. In the example
   > above, Entry B is dropped: 1 is below 0.25 * 18.

7. *Ordering and what never appears*:

   > Ties break by configured store order and then by path, so repeated searches
   > return the same order. Always-applied categories (`conventions` and `glossary`)
   > never appear in search results, because they are already loaded in full.

**Section B: "Tagging an entry"** (default plain background)

1. *What tags are for*:

   > Tags say what an entry is about, independently of the words it happens to use.
   > They are proposed for you when an entry is captured, drawn from the tags already
   > in use, so the vocabulary converges without anyone maintaining a taxonomy.

2. *The frontmatter block*, in full, as a fenced markdown block:

   > An entry may open with a YAML frontmatter block. It must be the very first thing
   > in the file, opened and closed by a line containing only `---`:
   >
   > ```markdown
   > ---
   > tags: [go, http, routing]
   > ---
   >
   > # HTTP routing standard
   >
   > All Go services route HTTP endpoints through chi.
   > ```
   >
   > The block form of a YAML list works too:
   >
   > ```markdown
   > ---
   > tags:
   >   - go
   >   - http
   >   - routing
   > ---
   > ```

3. *The rules*, as a short list. Each of these is a real behaviour a reader can
   otherwise only discover by experiment:

   > - **Tags are optional.** An entry without a block is perfectly valid and fully
   >   searchable. There is nothing to migrate and no command to run.
   > - **Tags are lower-cased and de-duplicated** when read, so `HTTP` and `http`
   >   cannot both exist as separate tags.
   > - **`tags` is the only key read.** Any other key in the block is ignored, so the
   >   block is safe to use for your own metadata.
   > - **A malformed or unclosed block is treated as no frontmatter**, not as an
   >   error. An entry that happens to open with a horizontal rule keeps working.

4. *Partial credit*, which is what stops a near-miss from being a total miss:

   > A tag does not have to be typed exactly to be found. When a search term and a tag
   > are prefix-related, the tag still counts, at a strength equal to the shorter
   > string divided by the longer:
   >
   > | search | tag | counts as |
   > |---|---|---|
   > | `https` | `https` | 100% |
   > | `http` | `https` | 80% |
   > | `apples` | `apple` | 83% |
   > | `https` | `https-security` | 36% |
   > | `go` | `golang` | nothing |
   >
   > This is why plurals and word forms mostly look after themselves: `apple` and
   > `apples` find each other without either being written down twice. It works in
   > both directions, so it does not matter which form the tag uses.
   >
   > Two limits are deliberate. The match must start at the beginning of the word, so
   > `test` does not find `latest`. And the shorter of the two must be at least four
   > characters, so very short terms like `go` or `id` match only exactly, rather than
   > matching every tag that happens to start with them.
   >
   > What it does not do is understand words. `route` does not find `routing`, because
   > `route` is not a prefix of `routing`. Where a form is genuinely different rather
   > than merely longer, carry both tags.

5. *A pointer to the capture flow*, so the reader knows they are not expected to do
   this by hand:

   > You rarely write this block yourself. The `spek-knowledge` skill proposes tags
   > when it captures an entry, and shows them alongside the destination for you to
   > confirm or edit before anything is written.

**Section C: narrowing** — a short block joining the existing `--tier` and `--filter`
prose in the Configuration section, with a fenced bash example and an explicit note on
the asymmetry:

> ```bash
> spektacular knowledge search "routing" --tag http
> ```
>
> An entry without that tag is never returned, however well it would otherwise score,
> and it is excluded before ranking rather than filtered out of the results. Repeat
> `--tag` to narrow further: an entry must carry every tag listed. (This differs from
> `--filter`, which names alternative stores and so widens.)

**Acceptance criteria**:

- [x] The published page gives the scoring formula in full, including the damping curve, the coverage factor and the constants as shipped, not only a prose description of the factors.
- [x] The published constants match the values actually in the code, checked against it rather than against this plan.
- [x] A worked example shows two entries scored against the same query, so a reader can follow the formula through to a number.
- [x] The page explains how a query becomes terms, and that there is no stemming or synonym handling.
- [x] The page distinguishes a tag match from a body occurrence, and states how a tag is matched: exact equality at full strength, and a prefix relation at proportional partial credit. *(Criterion corrected during implementation: it previously read "states that tag matching is exact", which the walkthrough's move to prefix partial credit superseded. This phase's own Content outline already described partial credit.)*
- [x] The page states the cutoff rule, its constant, that it is relative to the best hit, and that it is applied after every store has been searched.
- [x] The page documents the frontmatter block in full: where it goes, both YAML list forms, that `tags` is the only key read, that tags are lower-cased and de-duplicated, and that a malformed or unclosed block is treated as no frontmatter rather than an error.
- [x] The page explains what prefix matching means for choosing tag forms: that a singular and a plural find each other so neither needs writing twice, that a second tag is needed only where a form differs by more than its ending (`route` beside `routing`), and that similar-looking tags such as `http` and `https` are distinct subjects rather than variants. *(Criterion corrected during implementation, for the same reason as the one above: under prefix partial credit a singular does retrieve a plural, at 83%.)*
- [x] The page says an entry without tags is valid, fully searchable, and needs no migration.
- [x] The page shows how to narrow a search by tag, and says that repeating the option narrows further rather than widening.
- [x] No statement remains on the site claiming results are scored purely by how often query terms appear.
- [x] The documentation site builds and typechecks without errors or warnings, and the page contains no layout markup in its body.

#### - [x] Phase 4.3: Correct the storage-backend reference

**Repo:** docs

The public reference page for writing a storage backend publishes the exact contract
this work changes: what a search call receives, what a result carries, and which parts
of a result the backend fills in. This phase brings it in line, and states explicitly
that a backend reports what it found and never scores it, because a backend that
scored its own results would rank on a different scale from every other one and
nothing would fail loudly. The page also documents a result shape that has already
drifted from the code independently of this work, so that is corrected in the same
pass rather than left as a second wrong version.

*Technical detail:* [context.md#phase-43](./context.md#phase-43-correct-the-storage-backend-reference)

**Content example** — the revised Store-interface material. Structure and technical
facts are fixed by this plan; wording is illustrative.

1. *The `Search` signature in the published interface block*, updated to take
   pre-tokenized terms and an options value.

2. *The result type*, regrouped so ownership is visible at a glance rather than
   described underneath:

   > ```go
   > type Hit struct {
   >     // Filled in by the store
   >     Path       string   // locator, relative to the store root
   >     Title      string   // the document's first heading, or the locator
   >     Excerpts   []string // compact excerpts, each capped at the budget
   >     Checksum   string   // SHA-256 over the entry's raw bytes
   >     Tags       []string // the entry's declared tags
   >     BodyCounts []int    // occurrences per query term, in term order
   >
   >     // Left zero by the store, stamped by the knowledge layer
   >     Tier, Name string
   >     Category   string
   >     Score      float64
   > }
   > ```

3. *A revised `Search` contract bullet*, extending the reasoning the page already
   gives for `Tier` and `Name`:

   > A store leaves `Tier` and `Name` empty because it has no notion of the caller's
   > addressing scheme. For the same reason it leaves `Score` at zero: a store cannot
   > see the other stores its results will be merged and ranked against. Report what
   > you found, per term, and the knowledge layer scores it. A backend that computed
   > its own score would rank on a scale unrelated to every other backend's, and the
   > merged ordering would be quietly wrong rather than visibly broken.

4. *A short note naming the limit*, so it is met in design rather than in debugging:

   > This means a backend has to be able to report how often each query term occurs.
   > A purely semantic backend that can only return its own opaque relevance score
   > cannot take part in this ranking.

**Acceptance criteria**:

- [x] The published backend reference shows the search call and result type as they actually are after this work.
- [x] The page states that a backend reports evidence and never computes a score, and gives the reason.
- [x] The page names the limit that a backend unable to report per-term occurrence counts cannot participate in ranking.
- [x] The result fields that were already wrong before this work are correct, with no remaining field the code does not have.
- [x] The documentation site builds and typechecks without errors or warnings.

#### - [x] Phase 4.4: Record the ranking design as knowledge

**Repo:** spektacular

How ranking works is captured as an entry in the project's own knowledge base, so a
future planning run retrieves it instead of re-deriving it from the code. Writing it
last means it describes the weights as they actually shipped rather than as they were
proposed. Because the entry has to be found to be useful, confirming that a search for
its subject returns it doubles as a real end-to-end exercise of everything this plan
delivers.

*Technical detail:* [context.md#phase-44](./context.md#phase-44-record-the-ranking-design-as-knowledge)

**Acceptance criteria**:

- [x] The project's knowledge base contains an entry describing how ranking works, including the ranking factors and the relative cutoff.
- [x] The entry carries tags, proposed through the same capture flow this plan delivers.
- [x] A search for that subject returns the entry.
- [x] The entry was written only after the user confirmed its destination and content.

## Open Questions

Two items genuinely cannot be settled before implementation, and one item requires an
amendment to the spec before this plan is internally consistent. Everything else that
came up during planning was either decided and recorded as an assumption, or checked
during planning and answered.

### 0. The spec's Non-Goal on query expansion must be amended

**Not an open question so much as an outstanding action.** Prefix partial-credit tag
matching was added to this plan at the walkthrough, at the user's direction. The spec
currently states, under Non-Goals:

> Query expansion — stemming, synonyms, or plural handling — is not addressed.
> `router` and `routers` remain distinct terms.

Under this plan `router` and `routers` *do* match, at 6/7 strength. The plan and the
spec therefore contradict each other on a point of scope, and the spec is the document
that governs scope.

**Resolved at the walkthrough.** The user approved the amendment and it has been made:
the spec's Non-Goal now reads "Stemming and synonym expansion are not addressed... so
`route` and `routing` remain distinct terms", followed by a dated note recording that
the original wording also excluded plural handling, that positional prefix matching
with partial credit was added during the plan walkthrough, and that `router` and
`routers` therefore now match at reduced strength. Plan and spec agree. Nothing
outstanding; retained here so the change is discoverable from the plan.

### 1. Do the weights still clear the benchmark after tuning?

**The question.** The tag weight, the damping curve, the coverage exponent and the
cutoff fraction were validated as a set, against one corpus, by a prototype. The spec
explicitly expects them to be tuned during implementation. Tuning any one of them can
move a result the others were compensating for, and that interaction is only
observable once the real implementation runs against the real fixtures.

**What it depends on.** Phases 1.2, 1.3 and 2.2 landing, and the benchmark fixture
from Phase 2.4 existing.

**What the implementer should do.** Tune freely, as long as the Phase 2.4 benchmark
keeps all four observed queries returning the on-topic entry first and the acceptance
criteria of Phases 1.2, 1.3 and 2.2 still hold. That is in-scope work needing no
escalation. **STOP and ask the user** only if no set of values satisfies all of them at
once, because that would mean the scoring *shape* is wrong rather than its constants,
and changing the shape is a design decision rather than a tuning one.

### 2. Does "about that term" mean "tagged with that term"?

**The question.** The acceptance criterion is that a long entry mentioning a term many
times must not outrank a short entry that is *about* that term. This plan reads
"about" as "tagged with", because the two alternative signals (document-length
normalisation and title weighting) are both explicit Non-Goals of the spec, and under
any monotonic damping function twenty body occurrences still outscore two. The plan's
test fixture therefore pairs a **tagged** short entry against an **untagged** long one.
If that reading is wrong, no amount of tuning fixes it.

**What it depends on.** Nothing technical. It depends on what the criterion was meant
to assert, which is information only the user holds. It surfaces the moment someone
writes the Phase 2.2 test.

**What the implementer should do.** **STOP and ask the user** before writing that test
with two untagged entries. If the criterion is meant to hold between two entries
neither of which is tagged, the scoring model needs length normalisation and the
spec's Non-Goals need revisiting, which is a change to this plan rather than to the
code. This is the single most likely thing in this plan to be wrong.

---

### Checked during planning, not open

- **Does frontmatter-aware scanning break any entry already on disk?** No. Every
  `.md` entry in both registered repos' knowledge stores was inspected, and none opens
  with a `---` line, so none will be newly misread as frontmatter. One entry
  (`learnings/bob-custom-rules-vs-claude-skills.md`) has a horizontal rule on its third
  line, which a leading-prefix check does not touch. The plan additionally pins that an
  unterminated leading block is treated as "no frontmatter" rather than as an error, so
  even a future entry opening with a bare horizontal rule degrades to today's
  behaviour.

Decided during planning and recorded as assumptions rather than questions: whether
tags reuse the existing artifact frontmatter package (no, and why); whether repeated
`--tag` is AND or OR (AND); whether tags travel on the write payload or in the entry
body (the body); whether the benchmark runs against the live knowledge base or a
fixture (a fixture, with the evidence for why); and where the cutoff is applied (after
the cross-store merge).

## Out of Scope

### Carried from the spec's Non-Goals

- **Full BM25-style relevance scoring is not built.** This plan adds diminishing
  returns on repeated occurrences, not a general relevance model. In particular there
  is no document-length normalisation and no corpus-wide term-frequency weighting.
  Tracked as the remainder of [#35](https://github.com/jumppad-labs/spektacular/issues/35).
- **Title and heading matches are not weighted above body matches.** The scanner
  already extracts a document's first heading and could weight it cheaply, so this is
  a deliberate omission rather than an unavailable option. The only "aboutness" signal
  this plan introduces is the tag weight. Also part of [#35](https://github.com/jumppad-labs/spektacular/issues/35).
- **No stemming and no synonyms.** Prefix partial credit is the single relaxation of
  exact matching this plan makes, and it is purely positional: it relates two words
  when one opens the other, and nothing more. It does not know that `route` and
  `routing` are the same word, or that two unrelated spellings mean the same thing.
  Anything needing a model of English morphology, or a synonym table, stays out of
  scope. The spec's Non-Goal was narrowed accordingly during this plan's walkthrough;
  see § Open Questions item 0.

  **Stemming is the expected next lever, and deliberately comes after metrics.** The
  established solution to the `route`/`routing` gap is a stemmer (Porter, or Snowball),
  and it is the most likely candidate for the next real improvement in retrieval. It is
  sequenced behind the metrics work above on purpose: a stemmer is a model of English
  that behaves unpredictably on technical tokens such as `https`, `kubernetes` and
  `redis`, so adopting one without the ability to measure whether hits actually moved
  would be guessing. Metrics first, then stemming evaluated against them. Neither is in
  this plan.
- **No bulk rename or merge of a tag across a store.** The audit proposes changes one
  entry at a time with a human confirming each. Renaming `http-api` to `http`
  everywhere remains a manual job.
- **No change to how always-applied knowledge is loaded.** Conventions and glossary
  continue to be loaded in full on every task and to be excluded from search results,
  so the same content is never surfaced twice. They are also excluded from the tag
  vocabulary listing, for the same reason.

### Left to a later plan by the chosen design

- **No tag validation, hierarchy, or namespacing.** Tags are flat, lower-cased,
  free-form strings. There is no controlled vocabulary file, no allowed-tag list, and
  nothing rejects a tag at write time. Convergence comes from the capture flow
  preferring an established tag, not from enforcement. If the vocabulary is later
  observed to fragment despite that, adding validation is a separate piece of work.
- **No tag-based narrowing on `list`, `conventions` or `always-applied`.** The `--tag`
  option is added to `search` only. The other fan-out commands keep the `--tier` and
  `--filter` narrowing they have today. Extending it is mechanical but has no
  demonstrated need.
- **No semantic or embedding-based retrieval.** The knowledge base deliberately does
  not build a similarity index, and the reasoning is already recorded in the project's
  own documentation on why the de-duplication layer is exact rather than fuzzy. Tags
  are the low-tech alternative this plan commits to.
- **The two frontmatter readers are not consolidated.** The workflow-artifact reader
  and the new knowledge-entry reader share roughly fifteen lines of block detection and
  keep incompatible schemas. Extracting a shared helper is a reasonable refactor and is
  explicitly not forbidden, but it is not part of this plan's scope and should not
  gate it.
- **No performance work.** The change adds a logarithm per term per document and a
  parse of a file's opening lines, on top of a scan that already reads every byte and
  computes a SHA-256. No benchmark, no caching, no index. If a knowledge base grows
  large enough for the full-scan search to hurt, that is a distinct piece of work and
  it will hurt with or without this change.

- **No metrics or observability surface for the ranking.** This plan ships the
  algorithm and pins its behaviour with tests, but nothing that reports *why* a given
  entry scored what it did, or how ranking behaves across a real corpus over time.
  Tuning under this plan is therefore done against the benchmark fixture and the
  acceptance criteria, by hand. Agreed at the plan walkthrough as the natural next
  piece of work, to be specced separately: a surface that exposes per-term evidence,
  the score breakdown and corpus-level ranking behaviour would make tuning empirical
  rather than iterative, and would be the right tool for the weight tuning Open
  Question 1 anticipates. It is deliberately not a prerequisite: the four-query
  benchmark is sufficient to know this plan's own success metric is met.

### Deliberately not automated

- **Whether newly captured entries actually carry tags, and whether the vocabulary
  converges**, are the spec's third and fourth success metrics. Both are trends across
  many agent-driven capture events over time and are classified as manual checks
  captured in the implementation test plan. Their mechanical preconditions are tested;
  the outcomes are observed.
- **No end-to-end test that an agent asks for confirmation before writing.** The
  propose-then-confirm contract is enforced by skill prose, and the repo has no harness
  that drives an agent through a confirmation gate. Coverage stops at asserting the
  rendered prose carries the required instructions. Adding a CLI-level guard so it
  could be tested would change the write contract, which the spec does not ask for.

## Changelog

### 2026-09-10 — Phase 1.1: Move ranking above the storage layer

**What was done**: Scoring and query tokenization moved out of the file store and
into the knowledge layer. A store now reports the evidence it observed while
reading a document (per-term body occurrence counts, alongside the locator,
title, excerpts and checksum) and leaves `Score` at zero; `Set.Search` tokenizes
the query, scores every reported hit through one function in the new
`internal/knowledge/ranking.go`, and only then merges and sorts. `Store.Search`
took a signature change from `Search(query string)` to
`Search(terms []string, opts SearchOptions)`, with `SearchOptions` landing empty
so Phase 2.3's tag filter is purely additive. Behaviour is unchanged: `score`
still reproduces the historical rule exactly (every term must occur; the score is
the sum of the counts), so results and orderings are identical.

**Deviations**:

- **The store still skips documents with no evidence at all.** The plan said the
  walk callback "stops excluding on a zero count". Read literally that emits a
  `Hit` for every file in the tree, including binaries and total non-matches,
  which the knowledge layer would immediately score at 0 and drop. Ranked OR
  means "missing *some* terms is still a match"; a document matching *no* term is
  not a candidate. So the callback keeps a single guard — skip when every count
  is zero — and drops the per-term early return that implemented boolean AND.
  Results are identical either way; this only avoids materialising the whole
  store as hits.
- **Three pre-existing tests were relocated rather than left untouched.** The
  plan's own Phase 1.1 is internally inconsistent here: criterion 4 requires
  every pre-existing test in the affected package to pass with unchanged expected
  values, while criteria 2 and 3 and the file-change list require scoring and
  match-exclusion to leave the store entirely.
  `TestSearch_ScoreSumsOccurrencesAcrossDocument`, `TestSearch_OneHitPerDocument`
  and `TestSearch_MultiWordScatteredAcrossLines` assert exactly those two
  behaviours from a bare `FileStore`, so no implementation can satisfy both sides.
  They were moved to `internal/knowledge/set_test.go` and now assert through
  `Set.Search`, carrying every fixture byte and every hand-computed expected value
  across verbatim (`{twice:2, once:1, upper:1, spread:4}`,
  `[]string{"scattered.txt"}`, `float64(8)`). No number was recomputed; only the
  layer the assertion travels through changed.
- `Evidence.TagAffinity` is declared but left nil rather than allocated
  zero-length. Nothing reads it until Phase 2.2, and a per-hit zero-filled slice
  would be a dead allocation on every search.

**Files changed**:

- `spektacular: internal/knowledge/ranking.go` (new)
- `spektacular: internal/knowledge/ranking_test.go` (new)
- `spektacular: internal/knowledge/set.go`
- `spektacular: internal/knowledge/set_test.go`
- `spektacular: internal/store/store.go`
- `spektacular: internal/store/search.go`
- `spektacular: internal/store/search_test.go`
- `spektacular: internal/store/ignore.go`
- `spektacular: internal/store/ignore_test.go`

**Discoveries**:

- **`damp(1) = 1` is load-bearing for Phase 1.2 and is already relied on.** Six
  assertions in `cmd/knowledge_test.go` (`:245,253,772,780,788,1160`) pin
  `Score: 1` for a single term occurring once. They stay green through 1.2 only
  because the damping curve is `1 + log₂(n)`. A tuner who changes the curve's
  shape breaks them.
- **`Set.Search` returning early on an empty term list is now the only place
  "an empty query searches nothing" is decided.** `FileStore.Search` keeps its own
  `len(terms) == 0` guard for direct callers, so the behaviour is stated twice
  deliberately, at two layers with different callers.
- **The `partial.txt` case was verified empirically, not by reasoning.** A
  document with evidence for 2 of 3 terms does reach the knowledge layer
  (`BodyCounts=[1 1 0]`) and is dropped by the score-zero path, not by the
  always-applied category exclusion — `categoryOf` returns `""` for a root-level
  entry and `""` is not in the always-applied set. Phase 1.2 turns that same hit
  into a returned result, so the distinction matters.
- **`store.Hit` field order changed** (store-populated group first, stamped group
  second). Safe because nothing constructs a `Hit` positionally, confirmed by
  grep, and field order is not part of the JSON contract. Worth knowing before
  adding `Tags` in Phase 2.2.
- **Neither `make lint` nor CI runs a strict linter.** `make lint` is `go vet`,
  and CI (`dagger/main.go:190`) runs `go test -race ./...`. The deliberately
  unused `tagWeight` and `coverageExponent` constants therefore trip nothing,
  though an editor LSP flags them until Phases 1.2 and 2.2 consume them.

### 2026-09-10 — Phase 1.2: Rank partial matches instead of discarding them

**What was done**: Boolean AND matching is gone. `score` now sums `damp(count)`
rather than raw counts, and scales the total by the share of query terms the
document has any evidence for, squared. A document missing some of the query is
returned and ranked low instead of being dropped; only a document with evidence
for nothing at all still scores zero. `damp(n) = 1 + log₂(n)` gives repeated
occurrences diminishing returns, so twenty mentions of a term scores 5.32 against
two mentions' 2.0, not ten times as much.

**Deviations**: None.

**Files changed**:

- `spektacular: internal/knowledge/ranking.go`
- `spektacular: internal/knowledge/ranking_test.go`
- `spektacular: internal/knowledge/set.go`
- `spektacular: internal/knowledge/set_test.go`

**Discoveries**:

- **`TestSet_SearchRequiresEveryTerm` was renamed, not merely re-valued.** Its
  premise — that a document missing one term is excluded — is exactly what this
  phase deletes, so it is now
  `TestSet_SearchRanksPartialMatchesBelowFullOnes` and asserts both documents
  come back in score order. A test whose *name* asserts the old contract is worth
  hunting for whenever a phase inverts a rule; re-valuing it alone would have left
  a lie in the suite.
- **Several oracles are exact in float64 and should stay `require.Equal`.**
  `damp` of a power of two is exact (`damp(2)=2`, `damp(4)=3`, `damp(8)=4`,
  `damp(16)=5`), and a coverage of 1 contributes `math.Pow(1,2) = 1` exactly.
  Only genuinely irrational values need `InDelta`. Blanket-converting the suite
  to `InDelta` would have discarded real precision.
- **`damp(1) = 1` is now a production constraint, not a coincidence.** Six
  assertions in `cmd/knowledge_test.go` (`:245,253,772,780,788,1160`) pin
  `Score: 1` for a single term occurring once, and they passed untouched only
  because of it. It is commented at the function. A future tuner reshaping the
  curve must preserve it or fix those six.
- **For a single-term query, ranked OR returns exactly the same result set as
  boolean AND** — a document either contains the term or scores zero. That is why
  every single-term test in the suite survived this change without an edit, and
  it is worth knowing when judging which tests a future ranking change can affect.
- **Which fixture Phase 1.3's cutoff actually bites on, checked arithmetically.**
  A two-of-three match is *not* weak enough to be cut: `partial.txt` scores
  `2 × (2/3)² = 8/9 = 0.889` against `scattered.txt`'s 3, and `0.889` is above
  `0.25 × 3 = 0.75`, so `TestSet_SearchRanksPartialMatchesBelowFullOnes` must keep
  returning both hits after the cutoff lands. The fixture the cutoff does bite on
  is a **one**-of-three match: `1 × (1/3)² = 1/9 = 0.111`, comfortably below 0.75.
  `TestSet_SearchReturnsAnEntryMatchingOnlyOneOfThreeTerms` is therefore the
  Phase 1.3 fixture, and its "returned when it is the only match" case still holds
  because a lone hit is its own best score.

### 2026-09-10 — Phase 1.3: Drop weak matches relative to the strongest hit

**What was done**: `Set.Search` now drops any hit scoring below
`cutoffFraction × the best surviving hit`, with `cutoffFraction = 0.25` declared
in `ranking.go` beside the other tunables. The post-sort loop was split into two
passes: stamp tier/name/category and exclude always-applied entries into an
`eligible` slice, then cut against `eligible[0].Score`. A loosely related entry is
returned when it is the only thing there and vanishes once something genuinely
relevant is present. Milestone 1 is complete: search now ranks rather than
excludes, and returns a useful list rather than everything with a pulse.

**Deviations**: None.

**Files changed**:

- `spektacular: internal/knowledge/ranking.go`
- `spektacular: internal/knowledge/set.go`
- `spektacular: internal/knowledge/set_test.go`

**Discoveries**:

- **The bar must come from hits that survive the category exclusion, not from
  the raw merged slice.** An always-applied entry scoring highest would otherwise
  set the threshold and then be dropped itself, silently raising the bar for
  everything else. The two candidate readings differ by 27× on a simple fixture
  (floor `0.75` and an empty result, versus floor `1/36` and a returned hit), so
  this is not a subtlety that would have surfaced later — it would have shipped as
  "search sometimes returns nothing for no visible reason".
- **`TestSet_SearchCutoffIgnoresAlwaysAppliedWhenSettingTheBar` was
  mutation-checked, not just written.** Swapping `eligible[0].Score` for
  `merged[0].hit.Score` in `set.go` makes exactly that test fail, and nothing
  else. Worth repeating for any future test whose whole purpose is to pin a
  choice between two orderings: without the mutation check there is no evidence
  the fixture discriminates at all.
- **`cutoffFraction` cannot be raised above ~0.296 without breaking a deliberate
  guard.** `TestSet_SearchRanksPartialMatchesBelowFullOnes` pins that a
  two-of-three match (`0.889`) survives against a full match (`3`), and
  `0.889 / 3 = 0.296` is the ceiling. That test is now doing double duty as the
  "cutoff is not too aggressive" regression guard, and says so in its comment.
  Phase 2.4's benchmark constrains the constants from the other direction.
- **The cutoff needs the cross-store test to mean anything.** A per-store cutoff
  passes every single-store fixture, because a lone weak hit is always its own
  best score. `TestSet_SearchCutoffAppliesAcrossStores` puts the weak hit alone in
  one store and the strong hit in another, which is the only shape that catches
  it.

### 2026-09-10 — Phase 2.1: Read tags from an entry

**What was done**: A knowledge entry may now open with a YAML frontmatter block
declaring `tags`. A new `internal/store/frontmatter.go` parses it, lower-casing
and de-duplicating the values, and `scanFile` reads it out of the bytes it
already holds for the binary sniff, then scans the body from after the block. An
entry with no block is not an error and simply has no tags, so there is nothing
to migrate and no command to run. Tags do not affect retrieval yet — that is
Phase 2.2.

**Deviations**:

- **Malformed input: the parser reports, the scan degrades.** The plan says both
  that a malformed block "is reported rather than silently swallowed" (Component
  Breakdown) and that it returns no error (research.md's recorded decision). Both
  are now true: `parseEntry` returns an error for a properly delimited block whose
  YAML will not unmarshal, alongside safe values (no tags, raw bytes as body), and
  `scanFile` ignores that error so a search can never fail because of one bad
  entry. An *unterminated* opener is not an error at all — it reads as a
  horizontal rule, exactly as the decision requires. This keeps the `err` return
  in the declared contract from being permanently dead, and gives Phase 3.3's
  audit flow something to surface.
- **The block is parsed from the sniff buffer rather than by buffering scanned
  lines.** The plan offered two shapes: buffer the leading block during the scan,
  or thread an `inFrontmatter` flag through the scanner loop. Both were rejected
  for a third: `scanFile` already reads the first 8000 bytes for the binary check
  and already feeds them to the hasher, and frontmatter is by definition at the
  very start, so the block can be parsed straight out of those bytes and the
  scanner built from the post-block subslice. No buffering (so a file opening with
  `---` and never closing cannot accumulate in memory), no replay path, no flag,
  and term counting, title detection and excerpt collection all start at the body
  for free. Accepted limit, documented in the code: a block extending past 8000
  bytes reads as no frontmatter, degrading to today's behaviour.

**Files changed**:

- `spektacular: internal/store/frontmatter.go` (new)
- `spektacular: internal/store/frontmatter_test.go` (new)
- `spektacular: internal/store/search.go`
- `spektacular: internal/store/search_test.go`
- `spektacular: cmd/knowledge_test.go`

**Discoveries**:

- **The checksum must keep covering the frontmatter, and now has a test saying
  so.** It identifies the file, not its prose, so two entries with byte-identical
  bodies and different tags must hash differently or exact-byte de-duplication
  collapses them into one candidate. `TestSearch_ChecksumCoversFrontmatterSoTagsKeepEntriesDistinct`
  pins it. This is the one place the "scan the body, not the block" rule must not
  be applied.
- **Only one blank line after the closing `---` is dropped**; a second survives
  into the body. Pinned by an explicit test case rather than left as folklore.
- **Close detection searches for the byte sequence `"\n---"`**, so a block whose
  last content line itself begins with `---` terminates early. Not reachable
  through any specified case, and it is what makes a `---foo` closing line degrade
  to "no frontmatter" instead of erroring. Recorded because it is an implicit
  property of the scan rather than a stated rule.
- **A criterion-4 test needs a sibling to be non-vacuous.** "An entry tagged `go`
  is not returned for `go`" passes trivially if the query matches nothing at all.
  The fixture pairs it with a plain file that does mention "go" once, so the
  contrast proves the block was skipped rather than the search being broken. Both
  criterion-4 tests were mutation-checked by forcing `scanFile` back to the raw
  bytes; both failed, then the change was reverted.

### 2026-09-10 — Phase 2.2: Let tags retrieve and rank an entry

**What was done**: Tags became a retrieval signal. `store.Hit` gained `Tags`,
`tagAffinity` landed in `ranking.go` with `minPrefixLen = 4`, and `score` now
earns each term `tagWeight*affinity + damp(count)` with `tagWeight = 8`. A term
counts as covered if it has evidence of *either* kind. An entry tagged
`go, http` whose body contains neither word now scores 7.11 for
`go http router` and is found; a tag match at 8 outranks any realistic number of
prose mentions, since a term would have to appear 128 times to earn as much from
damping alone. The CLI advertises `tags` on search results, always as a list.

**Deviations**:

- **The store's candidacy guard now counts declared tags as evidence.** This
  extends the Phase 1.1 deviation, and had to. That guard skipped documents with
  no body evidence for any term — correct while only prose could match, and wrong
  the moment tags retrieve, because an entry tagged `go` whose prose never says
  "go" would never have reached the ranking layer to be scored on its tags at
  all. That is this phase's first acceptance criterion. A document is now reported
  if it has body evidence **or** declares any tags. Ranking stays out of the
  store: "this document carries evidence I cannot evaluate" is a refusal to judge
  relevance, not a judgement. An irrelevant tagged entry scores 0 upstream and is
  dropped before the merge, so results are identical to reporting every file. The
  cost is that every tagged entry is reported for every query, which is not a new
  order of cost against a search that already walks the tree reading every byte.

**Files changed**:

- `spektacular: internal/knowledge/ranking.go`
- `spektacular: internal/knowledge/ranking_test.go`
- `spektacular: internal/knowledge/set.go`
- `spektacular: internal/knowledge/set_test.go`
- `spektacular: internal/store/store.go`
- `spektacular: internal/store/search.go`
- `spektacular: internal/store/search_test.go`
- `spektacular: cmd/knowledge.go`
- `spektacular: cmd/knowledge_test.go`

**Discoveries**:

- **The plan undercounted the `Tags: []string{}` fallout: eight literals, not
  six.** The two extra sit inside
  `TestKnowledge_EveryCommandRunsAgainstAnUntaggedKnowledgeBase`, which Phase 2.1
  added — so the plan's list was accurate when written and went stale within one
  phase. Worth remembering that a plan's enumeration of call sites ages against
  the work the plan itself causes.
- **A tag match usually removes the prose entry rather than merely outranking
  it.** A tagged entry scores 8 for a single-term query, putting the cutoff floor
  at 2, while one prose mention scores `damp(1) = 1`. So "the tagged entry ranks
  first" often shows up as the prose entry not being returned at all. A fixture
  meant to assert *ordering* needs the prose sibling to mention the term at least
  three times (`damp(3) = 2.585`) to clear the floor. Correct behaviour, but it
  will look like a broken fixture to anyone who has not done the arithmetic.
- **Asserting "the JSON key is present but empty" needs a `*[]string`.** Decoding
  into a plain `[]string` cannot distinguish an omitted key from `[]`, so a test
  meant to pin `"tags": []` rather than an absent field silently asserts nothing.
  The same trap applies to `require.Empty`, which passes on nil — hence
  `require.NotNil` beside it.
- **`8 * 0.8` and the literal `6.4` are the same float64**, so an exact
  comparison would also have passed there. Kept `InDelta` anyway, because relying
  on a coincidence of binary representation is not a property worth depending on.

### 2026-09-10 — Phase 2.3: Narrow a search to tagged entries

**What was done**: `--tag` narrows a search to entries carrying every tag listed.
`SearchOptions.Tags` travels down to the store as a walk fast path,
`Selector.Tags` carries it through the knowledge layer, and `Set.Search`
independently enforces the filter after the merge so a provider is never trusted
on it. Repeated `--tag` is AND; the filter is absolute, excluding an entry
however well it would otherwise score; and matching is exact, so `--tag http`
does not reach an entry tagged `https`. `--tag` is registered on `search` alone.

**Deviations**:

- **Search got its own flag schema rather than extending the shared one.**
  `knowledgeNarrowingFlags` is what all four fan-out commands advertise under
  `--schema`. The plan noted the conflict and preferred a search-specific map;
  that is what landed, as `knowledgeSearchFlags`, so `list`, `conventions` and
  `always-applied` do not advertise a flag they ignore.
- **Refusing an unknown `--tag` is deferred to Phase 3.1, as the plan permits.**
  The refusal's next action has to be `knowledge tags`, which 3.1 delivers. Until
  then an unmatched tag returns an empty result, and a test pins that so the
  behaviour is deliberate rather than unnoticed.

**Files changed**:

- `spektacular: internal/store/store.go`
- `spektacular: internal/store/search.go`
- `spektacular: internal/store/search_test.go`
- `spektacular: internal/store/ignore_test.go`
- `spektacular: internal/knowledge/address.go`
- `spektacular: internal/knowledge/address_test.go`
- `spektacular: internal/knowledge/set.go`
- `spektacular: internal/knowledge/set_test.go`
- `spektacular: cmd/knowledge.go`
- `spektacular: cmd/knowledge_test.go`

**Discoveries**:

- **A real test-harness bug, not a test-writing inconvenience: `--tag` leaks
  between CLI subtests.** `StringArrayVar` binds a package-level slice that
  cobra *appends* to, so without clearing `knowledgeTags` in
  `resetKnowledgeFlags` one subtest's tags silently narrow the next one's search,
  producing order-dependent failures. `knowledgeTier` and `knowledgeFilter` have
  the same shape, so any future repeatable flag needs the same treatment.
- **`CarriesEveryTag` is exported so the rule has exactly one definition.** The
  store applies it while walking and the knowledge layer applies it after the
  merge; two copies would be two chances for the fast path and the authority to
  disagree, which is the drift the whole Phase 1.1 architecture exists to prevent.
- **The authority test needs a fake store, and `NewSet` deliberately offers no
  injection point** — providers come from configuration. The test therefore
  constructs a real `Set` with a real `scopedStore` from inside the package,
  swapping only the provider, which exercises the genuine `Set.Search` over the
  genuine source list. It also asserts the fake still *received* `opts.Tags`,
  pinning that the store is offered the fast path but not relied on for it.
- **The exclusion test is only meaningful if the excluded entry outranks the
  kept one.** The fixture gives the untagged entry 200 body occurrences
  (`damp(200) = 8.644`) so it genuinely beats the exact-tag entry's 8.0, and the
  unnarrowed search asserts that order first. Without that, "the filter excluded
  it" is indistinguishable from "it ranked below the cutoff anyway".
- **An `ignoreStore` forwarding test is vacuous without an ignore file.** A
  fixture with no `.spektacular_ignore` exercises a pass-through that cannot
  drop anything, so the option being dropped would be invisible. The fixture
  carries a real ignore file, and one assertion then catches both a dropped
  option and a dropped exclusion.

### 2026-09-10 — Phase 2.4: Prove the observed queries now find the right entry

**What was done**: A self-contained benchmark in
`internal/knowledge/benchmark_test.go` reconstructs the measured scenario — the
short on-topic routing entry, tagged, against the long early-concept document
that used to beat it — and pins the on-topic entry at rank 1 for all four queries
agents were actually observed using. **The spec's central success metric goes
from 0 to 4, with every constant exactly as the research prototype proposed.**
Nothing needed tuning, so Open Question 1 closes without escalation.

| Query | 1st | 2nd |
| --- | --- | --- |
| `http endpoints go` | `http-routing.md` 26.0 | `initial-idea.md` 9.49 |
| `http handler` | `http-routing.md` 20.585 | (competitor cut by the floor) |
| `http endpoints handler routing` | `http-routing.md` 38.585 | (competitor cut by the floor) |
| `api` | `http-routing.md` 8.0 | `initial-idea.md` 3.32 |

The `api` row is the success metric "the `api` case reverses" in one line: the
routing entry wins on its tag alone, its body containing no occurrence of "api"
at all, against a document with five real mentions. That query previously
returned nothing for the routing entry.

**Deviations**: None. This phase is test-only; no production file was touched and
no constant moved.

**Files changed**:

- `spektacular: internal/knowledge/benchmark_test.go` (new)

**Discoveries**:

- **Both the plan's Open Questions are now answered, and neither needed the
  user.** Question 1 (do the weights still clear the benchmark after tuning?) is
  moot — no tuning was required at any point across Milestones 1 and 2. Question 2
  ("does 'about that term' mean 'tagged with that term'?") was answered in
  practice at Phase 2.2 without ambiguity: every fixture pairing a tagged short
  entry against an untagged long one behaves as the plan predicted, and no
  criterion ever required the comparison to hold between two *untagged* entries,
  which is the case that would have needed length normalisation and a spec change.
- **The tuning envelope is now bounded on both sides and is comfortably open.**
  `cutoffFraction` cannot exceed ~0.296 without breaking
  `TestSet_SearchRanksPartialMatchesBelowFullOnes` (Phase 1.3), and this benchmark
  constrains from below. At the shipped 0.25 there is real headroom, and the
  benchmark's margins are wide (26 vs 9.49 on the closest query), so a modest
  retune will not silently give the metric back.
- **The benchmark's assertion is rank, never score.** Fixture prose differs
  between any two reconstructions of this scenario, so absolute scores are not
  reproducible while ranks are: two independent fixtures produced 26/20.585/38.585/8
  and 25/17/34/8 with identical ordering. Asserting scores would make the test
  brittle to a wording change in its own fixture, which is not what it guards.
- **The criterion-2 test reads its own fixture back and asserts the absence of
  the term.** Without that, someone rewording the fixture body could reintroduce
  "api" into the prose and the test would keep passing while no longer testing
  tag-only retrieval at all.

### 2026-09-10 — Phase 3.1: Report the tag vocabulary already in use

**What was done**: `knowledge tags` reports every distinct tag in use across the
stores a request covers, with an entry count each, most-used first and ties
broken alphabetically. `Set.Tags` walks the covered stores with `listFiles`,
reads each entry and extracts its tags via the now-exported `store.ParseEntry`.
Always-applied categories are excluded, consistent with search. The listing is
purely mechanical: it reports what exists and never judges what a new entry
should carry, which is the split that lets Phase 3.2's capture flow exercise
judgement over a vocabulary the CLI states exactly.

**Deviations**:

- **An unknown `--tag` is now refused, resolving the deferral from Phase 2.3.**
  The plan left this conditional. It should refuse, because the sibling narrowing
  flag already does: `validateSelector` rejects a `--filter` naming a store the
  tier does not reach, precisely so "a caller is never left reading an empty
  result as 'no such knowledge'". A typo'd `--tag` is the same failure. The new
  `ErrCodeTagUnknown` refusal names the offending tag and points at
  `knowledge tags`, per the repo's error-message convention.
- **The check runs only on the empty-result path.** Establishing the vocabulary
  costs a read of every entry in scope — as much again as the search itself — so
  paying it on every successful search to guard against a typo would be a poor
  trade. A tag-narrowed search that returns hits never builds the vocabulary at
  all.

**Files changed**:

- `spektacular: internal/store/frontmatter.go`
- `spektacular: internal/store/frontmatter_test.go`
- `spektacular: internal/store/search.go`
- `spektacular: internal/knowledge/address.go`
- `spektacular: internal/knowledge/set.go`
- `spektacular: internal/knowledge/set_test.go`
- `spektacular: cmd/knowledge.go`
- `spektacular: cmd/knowledge_test.go`

**Discoveries**:

- **A refusal must not over-fire, and that needs its own test.** The distinction
  the implementation turns on is between "this tag exists nowhere" (refuse) and
  "this tag exists but its entries did not match your query" (empty result, no
  error). Only the first is a caller mistake. Without
  `TestSet_SearchKnownTagWithNoMatchingEntryReturnsEmptyNotARefusal`, a
  refactor could quietly turn every fruitless tag-narrowed search into an error.
- **Adding a subcommand means auditing the test-file enumerations, and some of
  them deliberately should not grow.** `tags` was added to
  `knowledgeNarrowingCmds` (required — it shares the `--tier`/`--filter` backing
  vars, so its flag state would otherwise leak between runs),
  `knowledgeConfigLoadingCmds`, and two schema tests. It was deliberately left
  out of four others whose premises do not hold for it: a tag use carries no
  store address, and `tags` does publish flags. A blanket "add the new command
  everywhere" would have produced tests asserting things that are not true of it.
- **`refuseUnknownTags` swallows a vocabulary-read failure on purpose.** If the
  vocabulary cannot be built, the search returns its honest empty result rather
  than an error, because the vocabulary is only needed to *explain* an empty
  result and must never be able to fail a search that otherwise succeeded.
- **The next action hardcodes `"knowledge tags"` rather than the configured
  command prefix.** That matches the neighbouring refusals in the same file
  (`internal/knowledge/set.go:345` writes `"knowledge list --tier %s"` the same
  way), so it is consistent rather than an oversight — but the whole package
  would need changing together if these are ever made prefix-aware.

### 2026-09-10 — Phase 3.2: Propose tags when an entry is captured

**What was done**: The `spek-knowledge` capture flow now loads the tag
vocabulary with `knowledge tags`, chooses tags by judgement against it, stages a
body opening with a `tags:` frontmatter block, and shows the proposed tags inside
the confirmation it already required — one gate, not two. The three tag-form
rules are written down once, where Phase 3.3's audit intent can reference them.
The update intent carries an entry's existing block through unchanged unless the
revision is about the tags. Four other agent-facing surfaces stopped describing
retrieval as boolean AND. Templates were regenerated into the git-tracked
`.claude/` and `.bob/` copies.

**Deviations**: None.

**Files changed**:

- `spektacular: templates/skills/workflows/spek-knowledge/SKILL.md`
- `spektacular: templates/agents/knowledge-trigger.md`
- `spektacular: templates/steps/plan/02-discovery.md`
- `spektacular: templates/skills/skill_spawn-planning-agents.md`
- `spektacular: internal/agent/instruction_surface_test.go`
- `spektacular: .claude/skills/spek-knowledge/SKILL.md` (regenerated)
- `spektacular: .bob/skills/spek-knowledge/SKILL.md` (regenerated)
- `spektacular: AGENTS.md` (regenerated — see the pre-existing staleness below)

**Discoveries**:

- **`AGENTS.md` had been stale since commit `a76e04c`, and this phase's
  regeneration fixed it as a side effect.** That commit ("Ensure knowledge base
  takes precidence over patterns in code base") edited
  `templates/agents/historical-artifacts.md` without re-running init, so the
  committed `AGENTS.md` never received the paragraph declaring knowledge entries
  binding over the code. Verified: `git show HEAD:AGENTS.md` lacks it, the
  template has it. Every agent session since then read an AGENTS.md missing a rule
  the project intended to be binding — a live instance of exactly the failure the
  generated-copy discipline warns about. Kept, because a generated file is
  supposed to match its template.
- **Three placeholder namespaces, and only two are rendered.** Skills use
  `{{command}}`, plan steps use `{{config.command}}`, and
  `skill_spawn-planning-agents.md` uses **neither** — `cmd/skill.go`'s `runSkill`
  serves it verbatim from the embedded FS, so the template *is* the surface an
  agent sees and there is no generated copy that can go stale. Worth knowing
  before adding a placeholder to that file, which would render literally.
- **The obvious stale-claim guard would have been wrong.** A `NotContains` on
  "every query word" fails against the *new* prose, which deliberately says "a
  result is not proof that every query word appeared in it". The guard has to key
  on verbatim fragments of the superseded sentence
  ("matches when every query word", "every query word occurs") instead. A negative
  assertion needs checking against the replacement text, not just the removed text.
- **Prose edits made by string replacement need reading back.** One edit in this
  phase initially left a bullet ending mid-sentence ("...before concluding it
  is"). The replacement applied successfully and the suite stayed green; only
  re-reading the rendered line caught it.
- **The rendered-copy guards were mutation-checked.** Reintroducing the boolean
  claim in the plan step, removing the `route`/`routing` example, and weakening
  "one gate, not two" each made the corresponding guard fail. Regeneration was
  also confirmed idempotent: re-running `init claude` and `init bob` produces no
  further change.

### 2026-09-10 — Phase 3.3: Audit the tags on existing entries

**What was done**: The `spek-knowledge` skill gained a fourth intent. The audit
enumerates entries with `knowledge list`, loads the vocabulary with
`knowledge tags`, reads each entry, and reports unsupported tags and missing ones
— preferring a tag already in use — then proposes and confirms **per entry**
before writing through the existing `knowledge write` at the entry's original
address. No new CLI command, no bulk operation, no second write path. The
preamble, the invoke list and `# Decline handling` were updated alongside it, and
both agent copies regenerated. Milestone 3 is complete: tags now get applied, and
existing entries can be brought up to standard, without anyone designing a
taxonomy.

**Deviations**: None.

**Files changed**:

- `spektacular: templates/skills/workflows/spek-knowledge/SKILL.md`
- `spektacular: internal/agent/instruction_surface_test.go`
- `spektacular: .claude/skills/spek-knowledge/SKILL.md` (regenerated)
- `spektacular: .bob/skills/spek-knowledge/SKILL.md` (regenerated)

**Discoveries**:

- **Adding an intent means updating the places that count them.** The skill's own
  preamble said "one of three branches" and the invoke list said "all three
  intents". An intent added without changing those exists but is never reached, so
  the guard asserts the new counts *and* `NotContains` the stale phrasings. The
  same trap will apply to any fifth intent.
- **The audit's two rules are opposite errors, which is why both need pinning.**
  Over-merging (telling an entry tagged `https` to use `http`) and under-pruning
  (leaving `apples` beside `apple`) pull in opposite directions, and the audit is
  the one place an agent is invited to make either. Both guards were
  mutation-checked: replacing each rule with vague wording made the test fail.
- **A prose guard can check for invented commands, not just missing text.** The
  audit section is extracted between its heading and the next, then every
  `knowledge <verb>` in it is checked against the set `cmd/knowledge.go` actually
  registers. Injecting a plausible-sounding `knowledge retag-all` was caught. That
  is a stronger guarantee than asserting the four expected invocations are
  present, because the risk in a "composes only existing primitives" contract is
  an invented fifth.
- **The subcommand set is hand-maintained in the test because `cmd` imports
  `internal/agent`** — reading it from `cmd` would be an import cycle. Noted in a
  comment so the next person does not try.
- **`git checkout --` is dangerous during this work and cost a near-miss.** A
  sub-agent used it to revert a mutation-test edit on a template whose Phase 3.x
  changes were uncommitted, reverting real work; it was restored from a byte-exact
  copy taken beforehand and verified by checksum. Mutation testing on an
  uncommitted tree must restore from a copy, never from git. Independently
  re-verified afterwards: all twelve Phase 3.2 and 3.3 edits are present in the
  template, the other three templates are intact, and the rendered copies match a
  fresh render.

### 2026-09-10 — Phase 4.1: Correct the project's own documentation

**What was done**: `docs/knowledge-base.md` now carries the full retrieval
algorithm rather than a summary, matching its stated purpose that a maintainer can
understand the project without reading the source. Three new subsections: "How a
result is ranked" (tokenization, both kinds of per-term evidence, the formula as a
fenced block with the shipped constants, a worked two-entry example, prefix
partial credit with both guards and why edit distance was rejected, the relative
cutoff), "Where each part is computed" (a store reports evidence, the knowledge
layer ranks, and why that split exists), and "Tags on an entry" (both YAML list
forms and the six behavioural rules). The command reference gained `--tag` and a
`knowledge tags` row, "Contributing knowledge" gained tag proposal and the audit
intent, and `README.md` gained a `knowledge tags` bullet.

**Deviations**: None.

**Files changed**:

- `spektacular: docs/knowledge-base.md`
- `spektacular: README.md`

**Discoveries**:

- **The new ranking prose would have read as contradicting an existing section.**
  "Why the mechanical layer is exact, not fuzzy" argues against similarity
  matching, and ranking is now emphatically inexact. They are not in conflict
  because they answer different questions — de-duplication asks whether two
  entries are *the same* (a yes-or-no fact, decided on exact bytes), ranking asks
  which is a better *answer* (a matter of degree, therefore scored) — but a reader
  meeting both would reasonably see a contradiction. A paragraph now says so
  explicitly and scopes the older argument to the first question.
- **The constants were checked against the code, not against this plan.** All five
  match `internal/knowledge/ranking.go` as shipped, and the worked example's
  arithmetic was recomputed independently (A = 18, B = 1, floor 4.5, B cut). The
  plan's own values came from a prototype and were explicitly expected to be
  tuned; they happened not to move, but verifying against the source rather than
  the plan is the habit that matters.
- **The document now names its own source of truth.** It states that the constants
  live in `internal/knowledge/ranking.go` and that changing one is a four-file
  change — the code plus all three documents that quote it. Phases 4.2 and 4.4
  must carry the same sentence, or a later tuner reading only one of them will
  believe the edit is finished when it is not.

### 2026-09-10 — Phase 4.2: Publish the retrieval documentation

**Repo:** docs

**What was done**: Two new bands on `src/pages/knowledge-base.mdx`, "How a search
is ranked" (`surface`) and "Tagging an entry" (plain), carrying the full scoring
formula with the shipped constants, a worked two-entry example, prefix partial
credit with both guards, the relative cutoff, the frontmatter block in both YAML
list forms, and the six rules governing it. Edited in place: the stale ranking
sentence, the "no required frontmatter" line, the `knowledge write` block, the
narrowing prose (now covering `--tag`, the AND-vs-OR asymmetry, and the refusal
for an unknown tag), and the de-duplication section.

**Deviations**:

- **Two acceptance criteria were stale and were corrected rather than ticked as
  written.** Criteria 5 and 8 asserted that "tag matching is exact" and that "a
  singular does not retrieve a plural". Both describe the design the walkthrough
  superseded when it introduced prefix matching with proportional partial credit,
  and both contradict this phase's **own Content outline** in the same document,
  which says a tag match is "reduced when the two are prefix-related" and that
  "plurals and word forms mostly look after themselves". research.md records the
  supersession explicitly. The criteria text now describes the delivered
  behaviour, with an inline note saying what changed and why; ticking them as
  written would have recorded the page as documenting something false.
- **The de-duplication section needed scoping, in both documents.** Its argument
  against similarity matching now reads as contradicting a ranking model built on
  partial credit. A paragraph distinguishes the questions: de-duplication asks
  whether two entries are the same (yes-or-no, exact bytes), ranking asks which is
  a better answer (a matter of degree, therefore scored).

**Files changed**:

- `docs: src/pages/knowledge-base.mdx`

**Discoveries**:

- **Adding exactly two bands in that order preserved the page's alternation with
  no changes below it**, as the plan predicted: plain, surface, plain, then the
  two new bands as surface and plain, then Configuration's surface and the closing
  plain band. One band, or three, would have forced every section below to flip.
- **The page publishes the constants, so it is now coupled to the code.** It
  quotes them in pseudo-code form (`TAG_WEIGHT`) for readability, so it also names
  the real Go identifiers and points at `internal/knowledge/ranking.go` as the
  single source of truth. All four values were checked against the source rather
  than against this plan, whose numbers came from a prototype expected to be
  tuned.
- **`astro check` reports three pre-existing hints** about a deprecated
  `document.execCommand` in `Shell.astro` and a minified asset. They are unrelated
  to this change and were left alone; the acceptance criterion is zero errors and
  zero warnings, which is met.
- **The no-em-dashes convention binds this repo and not `spektacular`.** Verified
  by grep after writing: zero em dashes on the page, while the equivalent
  `spektacular` documentation in Phase 4.1 uses them throughout, correctly.

### 2026-09-10 — Phase 4.3: Correct the storage-backend reference

**Repo:** docs

**What was done**: `src/pages/extending.mdx` now publishes the `Store` contract as
it actually is: `Search(terms []string, opts SearchOptions)`, the `SearchOptions`
type, and the `Hit` struct regrouped into what a store fills in and what the
knowledge layer stamps. Four method-contract bullets were added, covering why a
store leaves `Score` at zero, that matching is a ranked OR rather than an AND, the
per-term-count requirement that excludes a purely semantic backend from this
ranking, and that the tag filter is an optimisation a store may skip.

**Deviations**: None.

**Files changed**:

- `docs: src/pages/extending.mdx`

**Discoveries**:

- **The published `Hit` had been wrong since before this work, and is now
  correct.** It showed `Excerpt string` where the code has `Excerpts []string`,
  and omitted `Title`, `Category` and `Checksum` entirely. Fixed in the same pass
  rather than shipping a second wrong version, as the plan directed.
- **The correction was verified programmatically, not by eye.** The published
  field list was parsed out of the MDX and compared against
  `internal/store/store.go`: same fields, same order, both groups. Reading a
  struct against a code block by eye is exactly how the original drift survived.
- **Nothing ties this page to the Go source, which is why it drifted.** There is
  no test, no generator and no CI check connecting the published interface to the
  code it documents, so the same drift will recur the next time the `Store`
  contract changes. The plan puts closing that gap out of scope; it is worth a
  follow-up, and it is a stronger candidate than it looks, because this page is the
  contract third-party backend authors build against.

### 2026-09-10 — Phase 4.4: Record the ranking design as knowledge

**What was done**: `architecture/knowledge-search-ranking.md` written to the
`repo`/`spektacular` store, tagged `[knowledge, search, ranking, retrieval,
tags]`, describing the three ranking factors, the relative cutoff and why it is
relative, where each part is computed and why, and the shipped constants. It
states that tuning a constant is a four-file change and names the tuning envelope
from both sides. Written through the capture flow this plan delivers, after
explicit user confirmation of destination, tags and body.

**Deviations**: None.

**Files changed**:

- `spektacular: .spektacular/knowledge/architecture/knowledge-search-ranking.md` (new)

**Discoveries**:

- **The plan's sharpest success metric was demonstrated on live data, not a
  fixture.** The word `retrieval` appears only in the entry's frontmatter block
  and nowhere in its prose (verified by grep against the body alone), and
  searching `retrieval` returns the entry at exactly **8.0**: pure tag weight,
  coverage 1, zero body contribution. Under the old design that query returned
  nothing. The same run confirmed the entry ranks first for its own subject
  (32.977 against the decision entry's 8.322), that `--tag ranking` narrows to it,
  and that `--tag bogus` is refused while naming the real vocabulary.
- **This entry seeded the tag vocabulary, which was empty until now.** Every tag
  it carries is new, so its five tags are what the next capture will see and
  prefer. That makes the first tagged entry disproportionately influential on
  where the vocabulary converges, which is worth knowing when starting a fresh
  knowledge base.
- **The "plurals look after themselves" rule has an exception below four
  characters.** `tag` and `tags` do *not* find each other, because the shorter is
  three runes and `minPrefixLen = 4` rejects it. The tag-form rule says one form
  per subject on the grounds that prefix matching relates them, and that reasoning
  silently fails for any singular shorter than four characters (`tag`, `api`,
  `job`, `key`). The vocabulary here uses `tags`, which is the natural noun for the
  subject, but a search for `tag` will not reach it. Worth a follow-up to the
  tag-form guidance rather than a change to this plan.
