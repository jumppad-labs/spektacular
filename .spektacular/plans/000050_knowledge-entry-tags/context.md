---
created_date: "2026-09-10"
status: completed
closed_date: "2026-09-10"
---

# Context: 000050_knowledge-entry-tags

## Current State Analysis


**Repos and roots.** Two registered repos, both touched. `spektacular` at
`/home/nicj/code/github.com/jumppad-labs/spektacular` owns the search engine, the
knowledge layer, the CLI and the skill templates. `docs` at
`/home/nicj/code/github.com/jumppad-labs/spektacular-website` owns the published
documentation site (Astro 5 + Tailwind v4). Requirements 1-10, 12 and 13 land in
`spektacular`; requirement 11 lands in `docs`. Never assume the working directory is
either root; confirm with `go run . repo list`.

### How search works today

`FileStore.Search` (`internal/store/search.go:46-52`) lower-cases the query, splits it
on whitespace, and walks the store root. `scanFile` (`:135-215`) opens each file once
and, in a single streaming pass, sniffs the first 8000 bytes for a NUL byte to skip
binaries, tees the remainder through a SHA-256 hasher, and for each line records the
first ATX heading as the title, counts case-insensitive substring occurrences of every
term, and ranks the line as an excerpt candidate. The result is a `fileAggregate`
(`:126-133`) holding per-term counts, the title, up to three excerpt lines, and the
checksum.

Two facts about the current ranking matter:

- **Matching is boolean AND.** `internal/store/search.go:70-75` returns early from the
  walk callback as soon as any term's count is zero. A document missing one term is
  not ranked low, it is excluded. Adding a term to a query can therefore only ever
  remove results, never reorder them.
- **Score is a raw sum.** `:76` sums the per-term counts with no normalisation, so a
  long document that mentions a term incidentally outranks a short document that is
  entirely about it.

`Set.Search` (`internal/knowledge/set.go:140-190`) fans the query across every store
the selector covers, merges the hits, sorts them score-descending with deterministic
tie-breaks on configured store order then path (`:167-176`), then post-filters to drop
always-applied categories (`:178-189`). That post-filter loop is where the new
relative cutoff goes.

`ignoreStore` (`internal/store/ignore.go:107-119`) decorates any store to drop
`.spektacular_ignore`-excluded paths from `List` and `Search`. It is a pass-through
that must forward the new search options.

### The blast radius of the interface change

`Store.Search` has exactly one implementation (`internal/store/search.go:46`), one
decorator (`internal/store/ignore.go:107`) and one consumer
(`internal/knowledge/set.go:153`). Nothing outside the knowledge layer uses it, so
changing the signature is contained rather than a public break.

**But one implementation today is not one implementation forever.** The project
already advertises further knowledge providers as planned, and the interface exists
precisely so there can be more. That is why this plan puts ranking *above* the
interface rather than inside the file store: a scoring formula duplicated per provider
would have to be reproduced exactly by each one, and any drift would silently corrupt
the cross-store merge and the best-hit-relative cutoff, which compare results from
different providers against each other. The precedent is already in the code:
`Hit.Tier`, `Hit.Name` and `Hit.Category` are documented at
`internal/store/store.go:25-36` as "left empty by the store and populated by the
knowledge layer", because a store cannot know its caller's addressing scheme. It
equally cannot know the other stores it will be ranked against, so `Score` belongs in
the same group.

The corollary is a real constraint on future providers, worth recording before someone
meets it in a debugging session: a store must be able to report per-term body
occurrence counts and the entry's tags to participate in this ranking. A purely
semantic backend that can only return its own opaque relevance score cannot rank
coherently alongside a file store and is out of scope for this design.

### Knowledge entries have no frontmatter today

Entries are written as raw bytes by `Set.Write` (`internal/knowledge/set.go:216-222`)
and are plain markdown starting with a `# Heading`. Verified during planning: **no
`.md` entry in either repo's knowledge store opens with a `---` line**, so
frontmatter-aware scanning cannot newly misread any existing entry. One entry
(`.spektacular/knowledge/learnings/bob-custom-rules-vs-claude-skills.md`) has a
horizontal rule on its third line, which a leading-prefix check does not touch.

### The trap: `internal/metadata` cannot be reused

`internal/metadata` looks like the right home for tag frontmatter and is not.
`Metadata.UnmarshalYAML` (`internal/metadata/metadata.go:83-108`) hard-requires a
parseable `created_date` and a `status` drawn from a four-value enum, and `Render`
(`internal/metadata/frontmatter.go:59-73`) re-marshals through a fixed `yamlShape`
(`internal/metadata/metadata.go:55-63`) that silently drops unknown keys. A knowledge
entry carrying only `tags:` would be rejected as malformed, and any round trip would
delete the tags. That package owns the *workflow artifact* schema (spec, plan,
changelog), a genuinely different schema. Copy the block-detection shape from
`internal/metadata/frontmatter.go:16-52`; do not import the package.

### Knowledge outranks the code here

`.spektacular/knowledge/decisions/ranked-or-over-boolean-and.md` records ranked OR,
coverage scaling and the relative cutoff as already decided. The code has not caught
up. Per the project's standing rule, plan to the entry and treat the difference as
work to do. That entry is currently **untracked in git** and should be committed as
part of this work.

### The measured baseline

Recorded in `.spektacular/context.md` from issue
[#35](https://github.com/jumppad-labs/spektacular/issues/35). A probe entry (a routing
standard) was written to `architecture/http-routing.md` and removed afterwards. A long
early-concept document, `architecture/initial-idea.md`, that mentions "Go", "API" and
"http" in passing and says nothing about routing, won three of four queries. On
`routing`, the single most on-topic word available, the document actually about
routing did not surface at all. Across three real plan runs against one unchanged
spec, the agent searched `http endpoints go`, then `http handler`, then
`http endpoints handler routing`; a fourth plausible phrasing, `api`, returned nothing
at all.

### Prototype validation

A Python prototype of the proposed formula was run during planning over the real
knowledge base with the probe entry recreated and tagged, and with
`decisions/ranked-or-over-boolean-and.md` excluded so the corpus matched the one the
baseline was measured against. Formula:
`score = (Σ_terms [tagWeight·isTag(t) + damp(count(t))]) × coverage²`, with
`damp(n) = 1 + log₂(n)` for `n > 0`, `tagWeight = 8`, `coverageExponent = 2`, and hits
below `0.25 × bestScore` dropped after the cross-store merge.

| Query | Prototype rank | Score vs runner-up |
| --- | --- | --- |
| `http endpoints go` | 1st of 2 | 28.58 vs 11.52 |
| `http handler` | 1st of 1 | 18.58 |
| `http endpoints handler routing` | 1st of 1 | 36.58 |
| `api` | 1st of 7 | 8.00 vs 4.70 |
| `routing` (extra) | 1st of 5 | 9.00 vs 5.39 |
| `http middleware` (extra) | 1st of 1 | 19.58 |

Four of four on the spec's benchmark. Two properties to preserve: `damp(1) = 1`, which
keeps every existing `Score: 1` assertion green
(`cmd/knowledge_test.go:245,253,772,780,788,1160`); and the cutoff visibly shortening
result lists rather than being cosmetic.

### The template regeneration discipline

`templates/skills/workflows/spek-knowledge/SKILL.md` is the source of truth (83 lines).
`internal/agent/skills.go:41-66` renders `{{command}}` via mustache and installs it;
`cmd/init.go:67` calls that unconditionally. The rendered copies at
`.claude/skills/spek-knowledge/SKILL.md` and `.bob/skills/spek-knowledge/SKILL.md` are
**git-tracked**. Any template edit must be followed by `go run . init claude` and
`go run . init bob`, with the regenerated files committed. Nothing fails until an agent
reads a stale copy.

## Per-Phase Technical Notes


Paths are relative to each repo's root as reported by `repo list`. Paths in the
`docs` repo are prefixed `docs:`; unprefixed paths are in `spektacular`
(`/home/nicj/code/github.com/jumppad-labs/spektacular`). The `docs` root is
`/home/nicj/code/github.com/jumppad-labs/spektacular-website`.

**Requirement to repo resolution.** Requirements 1-10, 12 and 13 are carried out in
`spektacular`; requirement 11 (published user documentation) is carried out in `docs`.
No requirement spans both repos.

### Phase 1.1: Move ranking above the storage layer

**File changes**

- `internal/knowledge/ranking.go` — **new file, in the knowledge layer, not the
  store.** Add `func Terms(query string) []string` (currently
  `strings.Fields(strings.ToLower(query))` at `internal/store/search.go:47`),
  `Evidence{BodyCounts []int; TagAffinity []float64}` (the affinity slice is declared
  here but stays zero for every term until Phase 2.2, since tags do not exist yet),
  `func score(e Evidence) float64`, and
  the named tunable constants (`tagWeight`, `coverageExponent`, and the damping
  helper). For this phase only, `score` reproduces the current rule exactly: return 0
  if any count is 0, else the sum of counts. Import `math` is deferred to 1.2.
- `internal/store/store.go:25-36` — add `BodyCounts []int \`json:"-"\`` to `Hit`, and
  regroup the fields into store-populated and knowledge-layer-stamped, with a comment
  on each group. Move `Score` into the stamped group; its doc comment must say the
  store leaves it zero, exactly as `Tier`, `Name` and `Category` already do.
- `internal/store/store.go:38-57` — change the interface method to
  `Search(terms []string, opts SearchOptions) ([]Hit, error)`. `SearchOptions` arrives
  empty in this phase and gains `Tags` in 2.3; introducing the parameter now avoids a
  second signature change later. Add the provider contract to the doc comment: a store
  reports per-term body counts and (from 2.1) tags, and never a score.
- `internal/store/search.go:46-52` — accept pre-tokenized terms, drop the internal
  `strings.Fields` call.
- `internal/store/search.go:70-77` — delete the scoring block. The callback stops
  computing a score and stops excluding on a zero count; it sets `BodyCounts` from
  `agg.counts` and emits the hit. **Exclusion moves out of the store entirely.**
- `internal/store/ignore.go:107-119` — forward the new signature.
- `internal/knowledge/set.go:140-190` — tokenize with `Terms(query)` before the
  fan-out; pass terms to each store; after collecting hits, stamp
  `hit.Score = score(...)` and drop hits scoring 0, **before** the existing sort at
  `:167-176` so ranking is unchanged.
- `internal/knowledge/ranking_test.go` — new table-driven unit tests for `score` and
  `Terms` alone, asserting the current behaviour.
- No existing test changes in this phase; the whole suite passing untouched is the
  phase's own check that the move is behaviour-preserving.

**Complexity**: Medium
**Token estimate**: ~18k tokens
**Agent strategy**: 2 parallel agents — one moving scoring and tokenization into
`internal/knowledge`, one updating the store interface, the file store and the ignore
decorator. Sequential integration; the signature change couples them.

**Notes.** Raised to Medium from the original Low: this now spans three packages
rather than one file. It is still a behaviour-preserving move and the existing suite
is the oracle. `Hit.Score` stays `float64` (`internal/store/store.go:33`); only its
owner changes. Add a comment at the `WalkDir` callback recording that the store no
longer rejects anything, so a later reader does not reintroduce the early return as
an optimisation.

### Phase 1.2: Rank partial matches instead of discarding them

**File changes**

- `internal/knowledge/ranking.go` — `score`: remove the zero-count exclusion. Compute
  `covered` as the number of terms with any evidence; return `0` when `covered == 0`;
  otherwise return `sum * math.Pow(float64(covered)/float64(len(terms)), coverageExponent)`.
  Add `damp(n int) float64` returning `0` for `n == 0` and `1 + math.Log2(float64(n))`
  otherwise, and sum `damp(count)` rather than `count`. Import `math`.
  Starting values validated by the research prototype: `coverageExponent = 2`.
- `internal/store/search.go:46-52` — update the `Search` doc comment; it currently
  states "a file matches when every term occurs somewhere in it", which becomes false.
- `internal/store/store.go:33` — the `Score` field comment, now describing the
  composite score the knowledge layer stamps rather than a sum the store computes.
- `internal/store/store.go:56` and `internal/store/search.go` package doc — the
  interface comment describing match semantics.
- `internal/store/search_test.go:153` `TestSearch_ScoreSumsOccurrencesAcrossDocument`
  — rename and recompute. Under `damp`, the hand-written oracle
  `{twice:2, once:1, upper:1, spread:4}` becomes `{twice:2, once:1, upper:1, spread:3}`
  (`1+log₂(4) = 3`). Recompute by hand from the fixture, per
  `.spektacular/knowledge/conventions` and the repo's independent-oracle practice.
- `internal/store/search_test.go:286` — `require.Equal(t, float64(8), hits[0].Score)`
  for `widget gear` over one document (`widget` 3, `gear` 5) becomes
  `damp(3) + damp(5) = (1+log₂3) + (1+log₂5) ≈ 2.585 + 3.322 = 5.907`, coverage 2/2 = 1.
  Use `require.InDelta` for float comparison, or assert relative order instead.
- `internal/store/search_test.go:437` `TestSearch_SingleWordMatchesSameFiles` — single
  term, so the returned set is unchanged; verify and keep.
- `internal/store/search_test.go` — new tests: an entry with evidence for one term of
  three is returned; a three-of-three entry outranks a one-of-three entry; a
  twenty-occurrence document does not score ten times a two-occurrence document.
- `internal/knowledge/set_test.go:122` — the comment claims hand-computed scores of 3
  and 1; recompute to `damp(3) = 2.585` and `damp(1) = 1`. The asserted *order* is
  unchanged, so only the comment and any score literal move.
- `cmd/knowledge_test.go:245,253,772,780,788,1160` — all assert `Score: 1` for a
  single term occurring once. `damp(1) = 1` and coverage `1/1 = 1`, so these stay
  green. Verify rather than edit.

**Complexity**: Medium
**Token estimate**: ~25k tokens
**Agent strategy**: 2 parallel agents — one on `internal/store` (implementation plus
its tests), one auditing `internal/knowledge` and `cmd` for score-dependent
assertions. Sequential integration, since the second agent's findings may change the
first's test expectations.

**Notes.** `damp(1) = 1` is a deliberate property: it keeps every existing single-
occurrence assertion valid and is worth a comment at the function so a later tuner
does not break it casually. Float equality is now a hazard in tests that previously
compared integers exactly.

### Phase 1.3: Drop weak matches relative to the strongest hit

**File changes**

- `internal/knowledge/set.go:178-189` — inside the existing post-sort loop that
  excludes always-applied categories, add the relative cutoff. The slice is already
  sorted score-descending at `:167-176`, so the best score is the first element's.
  Capture it before the loop and skip any hit scoring below `cutoffFraction * best`.
  Starting value validated by the research prototype: `cutoffFraction = 0.25`, declared
  as a named constant in this package.
- `internal/knowledge/set.go:140-146` — update the `Search` doc comment to state the
  cutoff and that it is applied post-merge, deliberately, because the strongest hit may
  live in another store.
- `internal/knowledge/set_test.go` — new tests: a one-of-three match is returned when
  alone; the same match is absent once a three-of-three match is present; and, the
  load-bearing case, the weak and strong matches placed in **different** stores, which
  is the only assertion that would catch the cutoff being applied per store.
- `internal/knowledge/set_test.go:157`
  `TestSet_SearchTieBreaksBySourceOrderThenPath` — four equal-score hits; equal scores
  are all at the maximum so none is cut. Verify still passes.
- `internal/store/search.go` — **no change**. Neither the score nor the cutoff is
  computed in a store; after Phase 1.1 the store has no scoring code left to touch.

**Complexity**: Low
**Token estimate**: ~14k tokens
**Agent strategy**: Single agent, sequential execution.

**Notes.** Order the new step after the always-applied exclusion or before it, but pick
deliberately: an always-applied entry scoring highest would otherwise set the cutoff
bar and then be dropped, raising the effective threshold for everything else. Compute
the best score from hits that survive the category exclusion.

### Phase 2.1: Read tags from an entry

**File changes**

- `internal/store/frontmatter.go` — new file. `func parseEntry(raw []byte) (tags []string, body []byte, err error)`.
  Model the block detection on `internal/metadata/frontmatter.go:16-52`: `---\n`
  prefix, find `\n---`, require the closer alone on its line, drop one blank separator.
  Unmarshal into a local `struct{ Tags []string \`yaml:"tags"\` }` so unknown keys are
  ignored. Lower-case and de-duplicate, preserving first-seen order. No leading block
  returns `(nil, raw, nil)`.
  **Do not use `internal/metadata`**: `internal/metadata/metadata.go:83-108` requires a
  parseable `created_date` and a `status` from a four-value enum, and
  `internal/metadata/frontmatter.go:59-73` re-marshals through a fixed shape that drops
  unknown keys, so a tags-only block would be rejected and any round trip would delete
  the tags. Put that reasoning in the new file's doc comment.
- `internal/store/search.go:126-133` — add `tags []string` to `fileAggregate`.
- `internal/store/search.go:135-215` `scanFile` — parse the frontmatter during the
  existing single pass and start term counting at the body. Two viable shapes: buffer
  the leading block during the scan and hand it to `parseEntry`, or track an
  `inFrontmatter` flag across the scanner loop, skipping term counting, title
  detection and excerpt collection until the closing fence. Either way the SHA-256 tee
  at `:170-176` must continue to cover the file's **exact raw bytes including the
  frontmatter**, so two entries differing only in tags do not collapse during
  exact-byte de-dup at `docs/knowledge-base.md:100-104`.
  A file with no leading fence takes the body path from line one and must gain no cost.
- `internal/store/search_test.go` — new tests for `parseEntry`: tags present; no block
  at all (the load-bearing case, since it is every entry that exists today); empty
  list; block with only unrelated keys; malformed block; case-normalisation and
  de-duplication. Plus a scan-level test that a tag is **not** also counted as a body
  occurrence of itself.
- `cmd/knowledge_test.go` — a test that every knowledge subcommand runs unchanged
  against a fixture of entirely untagged entries.

**Complexity**: Medium
**Token estimate**: ~22k tokens
**Agent strategy**: 2 parallel agents — one on `parseEntry` and its unit tests (pure,
no I/O), one on the `scanFile` integration. Sequential integration.

**Notes.** `internal/store` is deliberately domain-agnostic
(`internal/store/search.go:24-31` states it is category-agnostic). A frontmatter block
is a document-format concern rather than a knowledge-domain one, so this does not
breach that boundary; say so in the doc comment.

### Phase 2.2: Let tags retrieve and rank an entry

**File changes**

- `internal/store/store.go:25-36` — add `Tags []string \`json:"tags"\`` to `Hit`.
  No `omitempty`: an untagged entry reports `"tags": []`. Populate as `[]string{}`
  rather than nil where the walk builds the hit, so the JSON is an empty array.
- `internal/store/search.go:78-98` — set `Tags` on the constructed `Hit` from the
  frontmatter parsed in 2.1. This is the store's whole contribution: it reports the
  tags, it does not score them.
- `internal/knowledge/ranking.go` — add
  `func tagAffinity(term string, tags []string) float64`, returning the strongest match
  across the entry's tags (not a sum): `1.0` on exact equality; otherwise, for any tag
  where one of the two strings is a prefix of the other **and** the shorter is at least
  `minPrefixLen` runes, `len(shorter)/len(longer)`; otherwise `0`. Both sides are
  already lower-cased. Starting value: `minPrefixLen = 4`.
- `internal/knowledge/ranking.go` — populate `Evidence.TagAffinity`, which Phase 1.1
  declared and left zero for every term because tags did not exist yet, by calling
  `tagAffinity(term, hit.Tags)`. `score` then computes
  `tagWeight*affinity + damp(bodyCount)` per term. An exact match is `affinity == 1`,
  so exact matching is the general rule's special case, not a separate branch.
  Starting value validated by the prototype: `tagWeight = 8`.
- **Prefix, deliberately not edit distance.** Measured during the walkthrough: on the
  cases that should match, Levenshtein-derived and prefix scores are identical
  (`http`/`https` 0.80, `apple`/`apples` 0.83, `https`/`https-security` 0.36). On
  near-miss pairs they diverge badly — edit distance scores `test`/`rest` 0.75,
  `cors`/`core` 0.75, `tls`/`tld` 0.67, `dns`/`dos` 0.67, `api`/`apt` 0.67, all of
  which are unrelated subjects and plausible tags in this codebase, where prefix scores
  0. Do not "improve" this to a general string-distance measure.
- **Both guards are load-bearing.** Anchoring at the start is what stops `test` hitting
  `latest`; `minPrefixLen` is what stops `go` hitting `golang`, `google` and `gorilla`.
  Dropping either reintroduces the "any short term hits everything" failure that ruled
  out plain substring matching in the first place.
- Computing affinity here rather than in the store means a provider only has to report
  its tags, never to know they affect rank.
- `cmd/knowledge.go:114-134` `knowledgeSearchOutputSchema` — add
  `"tags": {Type: "array", Items: &schemaProp{Type: "string"}}`.
- `cmd/knowledge_test.go:418`
  `TestKnowledgeSearch_SchemaDeclaresPerDocumentHitFields` — extend the expected field
  set with `tags`.
- `cmd/knowledge_test.go:19-26` `knowledgeHit` — add `Tags []string`. Note the
  existing `require.Equal` comparisons at `:245,253,772,780,788,1160` currently omit
  the field; adding it to the struct means those literals now assert `Tags: nil`
  against a JSON `[]`, so each needs `Tags: []string{}`.
- `internal/knowledge/ranking_test.go` — table-driven tests for `tagAffinity` alone,
  with a hand-written oracle. Cover, at minimum: exact match is 1.0; `http`/`https` is
  0.8 in both directions; `apple`/`apples` is 5/6 in both directions;
  `https`/`https-security` is 5/14; `test`/`rest`, `cors`/`core`, `tls`/`tld` and
  `api`/`apt` are all 0; `go`/`golang` is 0 via `minPrefixLen`; `test`/`latest` is 0
  via the start anchor; and where several tags are prefix-related to one term, the
  strongest wins rather than the affinities summing.
- `internal/knowledge/set_test.go` — search-level tests: an entry tagged `go, http`
  whose body contains neither word is returned for `go http router`; a tagged entry
  outranks a comparable prose mention; a long many-occurrence entry does not outrank a
  short tagged one; an entry tagged `https` is returned for `http` and scores below
  what it scores for `https`; an entry tagged `https-security` is returned for `https`
  but ranks below one tagged `https` exactly.

**Complexity**: Medium
**Token estimate**: ~20k tokens
**Agent strategy**: 2 parallel agents — one on the tag scoring in
`internal/knowledge` plus the store's tag reporting, one on the CLI schema and the
`cmd` test-struct fallout. Sequential integration.

**Notes.** The `Tags: []string{}` fallout in `cmd/knowledge_test.go` is easy to miss
and produces confusing nil-vs-empty diffs; call it out to whoever takes this phase.
Ranking assertions belong in `internal/knowledge`, not `internal/store`: after 1.1 the
store has no score to assert on. The store-level tests here check that tags are read
and reported; the ranking tests check what they are worth.

### Phase 2.3: Narrow a search to tagged entries

**File changes**

- `internal/store/store.go` — add `Tags []string` to the `SearchOptions` struct
  introduced empty in Phase 1.1. **No interface signature change here**; 1.1 already
  moved `Search` to `(terms []string, opts SearchOptions)` precisely so this phase is
  additive.
- `internal/store/search.go:46-52` — apply `opts.Tags` during the walk as an
  optimisation: skip a document that does not carry every requested tag, so it is
  never reported. This is a fast path, not the correctness guarantee.
- `internal/knowledge/set.go:178-189` — **enforce** the tag filter in the post-merge
  loop, beside the always-applied exclusion, using `hit.Tags`. This is the
  authoritative check, so a provider that ignores or mis-implements `opts.Tags` cannot
  leak an untagged entry into the results. Cheap, because the hit already carries its
  tags.
- `internal/store/ignore.go:107-119` — forward `opts` to the wrapped store. A dropped
  option here is invisible in any test whose fixture has no ignore file.
- `internal/knowledge/address.go:41-44` — add `Tags []string \`json:"tags"\`` to
  `Selector`. `Selector.covers` at `:159-172` stays a store-membership test and must
  **not** learn about tags; add a comment saying so.
- `internal/knowledge/set.go:153` — pass `store.SearchOptions{Tags: sel.Tags}`.
- `cmd/knowledge.go:76-79` — add a `knowledgeTags []string` flag var.
- `cmd/knowledge.go:520-526` — register `--tag` as a repeatable `StringArrayVar` on
  `knowledgeSearchCmd` only, not on the other three fan-out commands.
- `cmd/knowledge.go:107-112` `knowledgeNarrowingFlags` — the schema is shared by all
  four fan-out commands, so `--tag` cannot simply be added to it. Either give search
  its own flags map, or add `tag` there and accept it being advertised on commands
  that ignore it. Prefer a search-specific map.
- `cmd/knowledge.go:495-506` `knowledgeSelector()` — carry `Tags: knowledgeTags`.
- `internal/knowledge/set.go` / `internal/knowledge/address.go:46-70` — if a `--tag`
  naming a tag no entry carries should be refused rather than silently returning
  nothing, add an error code and build the refusal with
  `output.NewError(code, msg).WithNextAction("… knowledge tags …")` per
  `.spektacular/knowledge/conventions/error-messages-must-suggest-remediation.md`.
  Note this depends on Phase 3.1's vocabulary lister; if 3.1 has not landed, return
  empty and defer the refusal to 3.1.
- Tests: `internal/knowledge/set_test.go` (filter excludes absolutely, two tags
  require both, and a hit reported by a store that ignored `opts.Tags` is still
  excluded — the assertion that pins the knowledge layer as the authority),
  `internal/store/search_test.go` (the walk's fast path skips non-matching documents),
  `cmd/knowledge_test.go` (flag plumbs through; schema advertises `tag`).

**Complexity**: Medium
**Token estimate**: ~20k tokens
**Agent strategy**: 2-3 parallel agents — store fast path, knowledge selector plus
authoritative filter, CLI flag and schema. These are more independent than in 1.1
because the interface signature is already settled; integrate sequentially anyway.

**Notes.** `knowledgeTier` and `knowledgeFilter` at `cmd/knowledge.go:76-79` are
package-level vars bound across four commands by the loop at `:522-525`; follow that
existing shape rather than fighting it. Repeated `--tag` is AND (narrowing), unlike
`--filter` which is OR (alternative stores) — a deliberate asymmetry to document.

### Phase 2.4: Prove the observed queries now find the right entry

**File changes**

- `internal/knowledge/set_test.go` (or a new `benchmark_test.go` in the same package)
  — a fixture in `t.TempDir()` reconstructing the measured scenario recorded in
  `.spektacular/context.md`: the on-topic routing entry tagged
  `[go, http, api, routing, handler, endpoints, middleware, chi]`, plus a long
  early-concept document that mentions "go", "api" and "http" in passing and says
  nothing about routing (the `architecture/initial-idea.md` shape). Table of the four
  observed queries → expected first path: `http endpoints go`, `http handler`,
  `http endpoints handler routing`, `api`.
- Same file — the sharpest single assertion: an entry tagged `api` whose body never
  contains the string "api" is returned for the query `api`. This fails under every
  part of the old design.

**Complexity**: Low
**Token estimate**: ~12k tokens
**Agent strategy**: Single agent, sequential execution.

**Notes.** The fixture must be self-contained in `t.TempDir()`, never pointed at
`.spektacular/knowledge`. Demonstrated during research: adding the real
`decisions/ranked-or-over-boolean-and.md` entry (which quotes all four benchmark
queries verbatim) to the live corpus moves the target from rank 1 to rank 2 on
`http handler`. A test bound to a growing directory would flap.
The research prototype validated the four-of-four outcome with `tagWeight = 8`,
`coverageExponent = 2`, `damp(n) = 1 + log₂(n)`, `cutoffFraction = 0.25`. If tuning
during earlier phases moved those, re-verify here; this test is the guard that tuning
does not give the success metric back.

### Phase 3.1: Report the tag vocabulary already in use

**File changes**

- `internal/knowledge/set.go` — add `TagUse{Tag string; Count int}` and
  `func (s *Set) Tags(sel Selector) ([]TagUse, error)`. Reuse `validateSelector` and
  `covers` for narrowing, and `listFiles` (`:412-434`) to enumerate entries. Read each
  entry and extract tags via the Phase 2.1 parser. Exclude always-applied categories
  using `alwaysAppliedSet()` (`:201-208`), consistent with search. Sort count
  descending, then tag ascending for determinism.
- `internal/store/store.go` or `internal/store/frontmatter.go` — export the tag parser
  (e.g. `store.ParseEntry`) so the knowledge layer can call it, or add a store-level
  `Tags()` helper. Prefer exporting the parser: it keeps the walk in the knowledge
  layer where `listFiles` already lives.
- `cmd/knowledge.go` — `knowledgeTagsCmd` with `Use: "tags"`, its own output schema
  `{"tags": [{"tag": string, "count": number}]}`, `--tier`/`--filter` registered via
  the existing loop at `:522-525` (add the command to that slice), a `--schema` mode,
  and registration in `AddCommand` at `:529`.
- `cmd/knowledge_test.go` — vocabulary listing returns tags with counts; ordering is
  count-desc then alpha; narrowing is honoured; an untagged base returns an empty
  array not an error.

**Complexity**: Medium
**Token estimate**: ~18k tokens
**Agent strategy**: 2 parallel agents — knowledge-layer listing, CLI command and
schema. Sequential integration.

**Notes.** Follow the single-declaration discipline of `internal/knowledge/category.go`
(the `Categories` registry): the vocabulary is derived from entries on disk, never
hard-coded anywhere. Reading every entry is O(entries) I/O; acceptable at current
scale and consistent with `AlwaysAppliedEntries` (`:262-296`), which already does it.

### Phase 3.2: Propose tags when an entry is captured

**File changes**

- `templates/skills/workflows/spek-knowledge/SKILL.md:52` — step 1 of the contribute
  flow: load the tag vocabulary with `{{command}} knowledge tags` alongside the
  existing `knowledge sources` and `knowledge categories`.
- `templates/skills/workflows/spek-knowledge/SKILL.md:53-55` — add a tag-selection
  step: prefer a tag already in the vocabulary; propose a new one only where nothing
  fits; never mint a near-duplicate of an existing tag (`HTTP` or `http-api` beside an
  existing `http`).
- `templates/skills/workflows/spek-knowledge/SKILL.md:53-55` — **and, in the same step,
  the tag-form rules.** These are load-bearing rather than advisory: exact matching
  with no stemming means a badly chosen form is simply never retrieved, which is the
  exact failure this feature exists to fix. Write them into the skill prose
  substantially as below (wording may be tightened, the three rules may not be
  dropped):

  > **Choosing tag forms.** A search term finds a tag when the two are equal, and also
  > when one opens the other, at reduced strength (`http` finds a `https` tag at four
  > fifths; `apple` and `apples` find each other). You therefore do **not** need to
  > carry both a singular and a plural, and should not.
  >
  > - **One form per subject, the natural one.** Tag `apple`, not `apple` and
  >   `apples`. The extra tag buys nothing and clutters the vocabulary.
  > - **Add a second tag only where a form differs by more than its ending.**
  >   `route` and `routing` do not find each other, because `route` is not the opening
  >   of `routing`. Where both are genuinely likely search words, carry both.
  > - **Do not collapse distinct subjects into one tag.** Reuse an existing tag when it
  >   means the same thing; an entry about HTTPS is tagged `https` even though `http`
  >   is already in the vocabulary, because those are different subjects rather than
  >   two spellings of one. The convergence rule exists to stop `HTTP`, `http-api` and
  >   `http_api` accumulating beside `http`; it does not license merging `https` into
  >   `http`. Prefix matching already relates `http` and `https` at reduced strength,
  >   which is the correct relationship between them: related, not identical.
- `templates/skills/workflows/spek-knowledge/SKILL.md:55` — the staged body under
  `.spektacular/tmp/<slug>.md` now opens with the `tags:` frontmatter block. The
  `knowledge write` invocation at `:57-61` is **unchanged**: tags travel in the file,
  not in `--data`.
- `templates/skills/workflows/spek-knowledge/SKILL.md:56` — the confirmation block
  shows the proposed tags alongside tier, store name and path. One gate, not two.
- `templates/skills/workflows/spek-knowledge/SKILL.md:31` — the lookup flow's
  description of a hit: add `tags` to the field list, replace "A document matches when
  every query word occurs somewhere in it, in any order" with the ranked-OR
  description, and mention `--tag` narrowing.
- `templates/skills/workflows/spek-knowledge/SKILL.md:68-76` — the update flow
  preserves existing tags unless the change is about them.
- `templates/agents/knowledge-trigger.md:15-25` — the capture offer mentions the tags
  it will propose.
- `templates/steps/plan/02-discovery.md:16` — the ranked-hit description given to
  planning agents; same corrections as SKILL.md:31.
- `templates/skills/skill_spawn-planning-agents.md:22` — same, for research agents.
- `internal/agent/instruction_surface_test.go:94-99` — extend the hand-maintained
  `expectedCRUDInvocations` slice with `knowledge tags`.
- Regenerate and commit: `go run . init claude && go run . init bob`, producing
  `.claude/skills/spek-knowledge/SKILL.md` and `.bob/skills/spek-knowledge/SKILL.md`.
  Both are git-tracked (`internal/agent/skills.go:41-66` writes them unconditionally
  via `cmd/init.go:67`).
- Verify unchanged: `internal/agent/claude_test.go:26-32`, `bob_test.go:24-33,49`,
  `codex_test.go:29` (marker strings), `internal/agent/knowledge_trigger_test.go:117`
  (picks up template changes), `templates/skill_resume_test.go:13-18`
  (`spek-knowledge` deliberately excluded from the resume-aware set).

**Complexity**: Medium
**Token estimate**: ~24k tokens
**Agent strategy**: 2 parallel agents — one on the `spek-knowledge` SKILL.md, one on
the four other agent-facing prose files. Sequential integration, then regeneration and
the full test run as one step.

**Notes.** `internal/agent/commands.go:22` `workflowDescriptions` duplicates the
SKILL.md frontmatter description; only touch it if the description changes.
`templates/steps/implement/07-update_changelog.md:47-51` and
`templates/steps/plan/18-walkthrough.md:22` hand off *to* the skill and need no change,
but their contract tests (`internal/steps/implement/steps_test.go:370-401`,
`internal/steps/plan/steps_test.go:487-516`) require the hand-off names the skill and
never emits `skill spek-knowledge`.

### Phase 3.3: Audit the tags on existing entries

**File changes**

- `templates/skills/workflows/spek-knowledge/SKILL.md` — the audit intent must inherit
  the tag-form rules from Phase 3.2 explicitly, because the audit is the single most
  likely place to apply the convergence rule too aggressively. Two guards to state in
  the prose: a tag that is a genuinely distinct term from a similar-looking existing
  tag is never reported as unsupported (do not tell an entry tagged `https` to use
  `http`, since prefix matching already relates them at reduced strength, which is the
  correct relationship); and, conversely, a tag made redundant *because* prefix
  matching reaches it, such as `apples` beside `apple`, should be reported as
  removable. Reference the tag-form rules rather than restating them, so the two
  intents cannot drift.
- `templates/skills/workflows/spek-knowledge/SKILL.md` — a new `# Intent: audit`
  section after the update intent (currently ending `:77`), and a corresponding
  trigger line in `# When to invoke` (`:15-25`). The flow: enumerate entries in scope
  with `{{command}} knowledge list`; load the vocabulary with
  `{{command}} knowledge tags`; read each entry with `{{command}} knowledge read`;
  for each, report tags the content does not bear out and propose missing ones,
  preferring an existing tag; show the proposed change for **that entry** and wait for
  explicit confirmation; only then stage the revised body and write it with the
  existing `{{command}} knowledge write` at the entry's original tier, name and path;
  `rm` the scratch file. No new CLI command.
- Same file, `# Decline handling` (`:79-83`) — extend to cover the audit intent, so a
  decline on one entry does not carry to the next and never triggers a write.
- `internal/agent/instruction_surface_test.go` — a guard asserting the rendered
  spek-knowledge SKILL.md carries the audit intent and its confirmation language
  (hand-maintained literal slice, per the existing pattern at `:94-99`).
- Regenerate and commit the `.claude` and `.bob` copies as in Phase 3.2.

**Complexity**: Low
**Token estimate**: ~14k tokens
**Agent strategy**: Single agent, sequential execution.

**Notes.** Explicitly per-entry. Bulk rename or merge of a tag across a store is a
spec Non-Goal. The propose-then-confirm contract is enforced by prose, not by a CLI
guard (`SKILL.md:83` says so); a reviewer should check the audit intent cannot reach
`knowledge write` without passing the same gate.

### Phase 4.1: Correct the project's own documentation

**File changes**

- `docs/knowledge-base.md:78-93` — the "Search behaviour" section. Replace "A document
  matches when every query word occurs somewhere in it, in any order" with the
  ranked-OR description, and add `tags` to the per-result field list at `:82-91`.
  This document's stated purpose is that "a maintainer new to the project can
  understand it without reading the source", so it carries the **full algorithm**, not
  a summary: tokenization, the two kinds of per-term evidence, the `tagAffinity` /
  `damp` / `termScore` / `coverage` / `score` formula with the shipped constants
  (including `MIN_PREFIX`), the prefix partial-credit rule and both of its guards, and
  the relative cutoff with its constant. State explicitly that prefix matching is
  positional only and is not stemming. Reuse the worked two-entry example from Phase
  4.2's content outline so the two documents agree line for line.
- `docs/knowledge-base.md` — a note on **where** each part is computed, which the
  published site deliberately omits and a maintainer needs: a store reports per-term
  evidence and the entry's tags, and the knowledge layer tokenizes, scores, merges,
  sorts and applies the cutoff. Cross-reference the storage-backend reference.
- `docs/knowledge-base.md:185-197` — the command reference table: update the
  `knowledge search` row for `--tag`, and add a `knowledge tags` row.
- `docs/knowledge-base.md:173-183` — "Contributing knowledge": the proposal now
  includes tags alongside tier, store name and path.
- `docs/knowledge-base.md` — a new subsection documenting the frontmatter contract in
  full: the block must lead the file and be fenced by lines containing only `---`;
  both YAML list forms are accepted; `tags` is the only key read and others are
  ignored; values are lower-cased and de-duplicated; tag matching is exact, so `go`
  does not match `golang`; the block is not counted as body text; a missing block
  means no tags and is valid; and a malformed or unclosed block is treated as no
  frontmatter rather than an error. Natural home is after the category model
  (`:14-51`) or within "Search behaviour".
- `docs/knowledge-base.md:122-143` — "Why the mechanical layer is exact, not fuzzy"
  argues against similarity-based matching; confirm the new ranking prose does not
  read as contradicting it. Ranking is not de-dup; a sentence distinguishing them may
  help.
- `README.md:137` — the one-line `knowledge search` description; add a `knowledge tags`
  bullet beside it.

**Complexity**: Low
**Token estimate**: ~14k tokens
**Agent strategy**: Single agent, sequential execution. Prose only, no build step.

**Notes.** This is the `spektacular` repo's internal maintainer documentation, distinct
from the published site in Phase 4.2. The docs repo's no-em-dashes convention does
**not** bind this repo.

**Three documents now state the scoring formula**: this one, the published page
(Phase 4.2), and the knowledge entry (Phase 4.4). That is a real duplication hazard,
accepted because the three serve different readers (maintainer, user, planning agent).
The mitigation is that **the named constants in `internal/knowledge/ranking.go` are the
single source of truth** and all three quote them; say so explicitly in each document,
so a later tuner knows the edit is not complete until all three agree. Whoever tunes a
constant must update all three. This is called out again in Phase 4.4.

### Phase 4.2: Publish the retrieval documentation

**File changes**

- `docs:src/pages/knowledge-base.mdx:122-139` — the `knowledge search` block inside
  the "The lifecycle of an entry" section. Replace the ranking sentence at `:122-130`
  ("Results are scored by how often the query terms appear, with ties broken by store
  order and then path, so the ordering is stable") with the copy in the plan's
  **Content example** for this phase.
- `docs:src/pages/knowledge-base.mdx:106-120` — the `knowledge write` block: the
  proposal now includes tags.
- `docs:src/pages/knowledge-base.mdx:228-241` — the `--tier`/`--filter` narrowing
  prose in the Configuration section: add `--tag`, including the AND-vs-OR asymmetry.
- `docs:src/pages/knowledge-base.mdx:282-290` — "De-duplication is exact, not fuzzy":
  verify the new ranking copy does not read as contradicting it.
- `docs:src/pages/knowledge-base.mdx` — **two new top-level `Section` bands**, "How a
  search is ranked" and "Tagging an entry", inserted immediately after "The lifecycle
  of an entry" (`:94`) and before "Configuration" (`:152`). The first takes `surface`,
  the second takes the default plain background. Adding exactly two bands in that
  order preserves the page's existing alternation with **no changes needed further
  down**: the run becomes plain (What it is) → surface (six categories) → plain
  (lifecycle) → **surface (ranked)** → **plain (tagging)** → surface (Configuration) →
  plain (Why it works this way) → CtaBanner. Verify this against
  `docs:.spektacular/knowledge/conventions/alternate-section-background.md` before
  writing; if the content ends up as one band rather than two, every band below it must
  flip.
  This supersedes the earlier intent to extend the lifecycle section in place: the
  content is now a full algorithm reference plus a full frontmatter reference, which is
  too much to nest inside a lifecycle narrative.
- `docs:src/pages/knowledge-base.mdx` — the formula and the worked example are fenced
  blocks (```text and a markdown table) inside `<Prose nested>`, following the pattern
  at `:69-76`. Rule 4 of **mdx-authoring** means fenced markdown, never a JSX code
  prop; `astro-expressive-code` supplies the chrome automatically.
- Components: compose from `docs:src/components/sections/Section.astro` and
  `Prose.astro` (the pattern at `docs:src/pages/knowledge-base.mdx:44-92`); use
  `ConfigurationKeys.astro` + `ConfigKey.astro` only if a flag-card list is wanted
  (canonical example at `docs:src/pages/configuration.mdx:166-180`). No new component.
- `docs:src/components/Nav.astro:6-21` — **no change**; the page already exists.
- Verify: `npm run build` and `make check` (`npx astro check`, 0 errors 0 warnings) in
  the docs root; `grep -nE "<div|<section|class=" src/pages/*.mdx` returns zero
  matches.

**Complexity**: Medium
**Token estimate**: ~22k tokens
**Agent strategy**: Single agent, sequential execution. Must run in the `docs` root
`/home/nicj/code/github.com/jumppad-labs/spektacular-website`, not the project
directory.

**Notes.** Binding docs-repo conventions, all loaded from that repo's own knowledge
store: **mdx-authoring** Rule 1 (no `<div>`, `<section>` or `class=` in a page body),
Rule 3 (blank line after an opening tag and before a closing tag on any slotted
component), Rule 4 (fenced markdown code blocks, never a JSX code prop);
**no-em-dashes** (applies to the copy and to the commit and PR description for this
repo); **site-layout** (compose from existing components, one frame width, one heading
scale); **plan-content-pages** (satisfied by the Content outline block in plan.md).
CI (`docs:.github/workflows/deploy.yml`) runs build only, so `astro check` must be run
locally.

**The constants are now published, so the page is coupled to the code.** Take
`TAG_WEIGHT`, `COVERAGE_EXPONENT`, `MIN_PREFIX` and `CUTOFF_FRACTION` from
`internal/knowledge/ranking.go` as shipped, never from this plan, whose values were set
by a prototype and are explicitly expected to be tuned during implementation. Re-check
them immediately before writing the page, and re-derive the worked example's arithmetic
if any of them moved. This coupling is deliberate and was the user's call at the
walkthrough; the earlier draft kept numbers off the page to avoid it.

**Check the worked example's arithmetic rather than trusting it.** With the plan's
values it reads: Entry A tagged `http, routing` with "routing" twice in the body scores
(8 + 0) + (8 + damp(2)=2) = 18 at coverage 1.0; Entry B untagged with "http" eight
times scores damp(8) = 4 at coverage 0.5, giving 4 × 0.25 = 1; and 1 falls below
0.25 × 18 = 4.5, so Entry B is cut. If a constant changed, this whole example changes.

### Phase 4.3: Correct the storage-backend reference

**File changes**

- `docs:src/pages/extending.mdx:47` — the published `Store` interface block still shows
  `Search(query string) ([]Hit, error)`. Update to
  `Search(terms []string, opts SearchOptions) ([]Hit, error)` and add the
  `SearchOptions` type beside the existing `DirEntry` helper block at `:52-57`.
- `docs:src/pages/extending.mdx:59-65` — the published `Hit` struct. **Already stale
  before this work**, independently of it: it shows `Excerpt string` (singular) where
  the code has `Excerpts []string`, and omits `Title`, `Category` and `Checksum`
  entirely (`spektacular:internal/store/store.go:25-36`). Replace with the regrouped
  shape from the plan's Content example: store-filled fields first, then the fields
  the knowledge layer stamps. Correct the pre-existing drift in the same pass rather
  than shipping a second wrong version.
- `docs:src/pages/extending.mdx:64` — `Score float64 // optional relevance score; 0 if
  absent` is the single most misleading line on the page after this work. It reads as
  "fill this in if you can". Move `Score` into the stamped group and say the store
  leaves it at zero.
- `docs:src/pages/extending.mdx:78-84` — the `Search` method-contract bullet. It
  already explains the `Tier`/`Name` stamping precedent; extend the same reasoning to
  `Score`, and add the per-term-counts limit for semantic backends.
- `docs:src/pages/extending.mdx:126-135` — the "Registering a backend" section names
  `internal/knowledge/set.go`'s provider switch as the worked example. Verify it still
  reads correctly once tokenization and scoring live in that file; the provider switch
  itself is unchanged by this work, so this is a read-through rather than an edit.
- `docs:src/pages/extending.mdx:1-4` — the page `description` frontmatter mentions the
  `Store` interface shape; check it still holds.
- Verify: `npm run build` and `make check` (`npx astro check`, 0 errors 0 warnings) in
  the docs root; `grep -nE "<div|<section|class=" src/pages/*.mdx` returns zero
  matches.

**Complexity**: Low
**Token estimate**: ~12k tokens
**Agent strategy**: Single agent, sequential execution. Must run in the `docs` root
`/home/nicj/code/github.com/jumppad-labs/spektacular-website`, not the project
directory.

**Notes.** This page is the reason the Phase 1.1 architecture matters to anyone
outside this repo, so it is the one place the "stores report, the knowledge layer
ranks" rule must be stated as a rule rather than implied. Sequence it after Phase 2.3,
when the signature and `SearchOptions` are final. The page is authored in MDX inside a
single `<Prose>` block, so the same docs-repo conventions bind as Phase 4.2:
**mdx-authoring** Rules 1, 3 and 4, **no-em-dashes**, and **site-layout**. Note the
existing page uses fenced ```go blocks inside `<Prose>`, which is Rule 4 compliant;
follow that shape.

**On the pre-existing drift.** Worth flagging to the user when this phase lands: the
published `Hit` has been wrong since `Title`, `Category` and `Checksum` were added and
`Excerpt` became `Excerpts`. That suggests this page is not currently covered by
anything that would catch drift, which is a gap this plan does not close. Out of scope
here, but worth a follow-up.

### Phase 4.4: Record the ranking design as knowledge

**File changes**

- A new entry in the `spektacular` repo's knowledge store, category `architecture` per
  the spec's technical approach, e.g. `architecture/knowledge-search-ranking.md`.
  Content: the three ranking factors (tag weight, damped body occurrences, coverage
  scaling), the relative cutoff and why it is relative rather than absolute, where each
  is computed (per-document in the store, cutoff post-merge in the knowledge layer) and
  why, and the shipped values of the tunable constants. It carries a `tags:`
  frontmatter block, proposed through the Phase 3.2 capture flow.
- Written via the `spek-knowledge` skill, which stages under `.spektacular/tmp/`,
  proposes tier, store name, path, tags and body, waits for explicit confirmation, then
  runs `knowledge write --data '{"tier":"repo","name":"spektacular","path":"architecture/…"}' --file …`
  and removes the scratch file.
- Verify by searching for the subject and confirming the entry is returned, which is a
  live end-to-end exercise of the whole plan.

**Complexity**: Low
**Token estimate**: ~8k tokens
**Agent strategy**: Single agent, sequential execution. Requires user confirmation
before the write; do not automate past that gate.

**Notes.** **Tuning a constant is a four-file change**, not a one-file change: the
named constants in `internal/knowledge/ranking.go`, plus the three documents that quote
them (`docs/knowledge-base.md` from Phase 4.1, `docs:src/pages/knowledge-base.mdx` from
Phase 4.2, and this entry). State that obligation inside this entry itself, since a
planning agent retrieving it is the most likely reader to be about to change a weight.

It complements rather than replaces the existing
`.spektacular/knowledge/decisions/ranked-or-over-boolean-and.md`, which records *why*
ranked OR was chosen. This entry records *how* ranking works. Keep the decision/
architecture boundary from `internal/knowledge/category.go:57-113`. That decision entry
is currently untracked in git and should be committed as part of this work.

## Testing Strategy


Coverage concentrates on the scorer, because every behavioural requirement in the spec
is a statement about it and it is the one component testable without touching a
filesystem. Conventions to follow throughout, taken from the existing suite:

- Fixtures are written into `t.TempDir()` through a `NewFileStore`, never read from any
  real directory (`internal/store/search_test.go:19-36`).
- Expected values are hand-maintained literals, explicitly not derived from the subject
  under test (`internal/store/search_test.go:176-179`, and the comment at
  `internal/agent/instruction_surface_test.go:79-81`).
- Skill assertions render through the production install path into a temp dir
  (`internal/agent/instruction_surface_test.go:82-104`), never reading the committed
  `.claude/` copy.
- New error paths use `output.NewError(...).WithNextAction(...)`, per
  `.spektacular/knowledge/conventions/error-messages-must-suggest-remediation.md`.
- `go test ./...` must pass in full before the work is called done, per
  `.spektacular/knowledge/conventions/tests-must-pass-for-done.md`. This change
  deliberately invalidates existing assertions, so none may be dismissed as
  pre-existing.

### Per-phase testing detail

**Phase 1.1** — Table-driven unit tests for `score` and `Terms` in isolation, in
`internal/knowledge`, asserting the *current* behaviour. No existing test changes; the
whole suite passing untouched is this phase's own proof the move is
behaviour-preserving. Note that ranking tests now live in `internal/knowledge` rather
than `internal/store`, which is where later phases must add them too.

**Phase 1.2** — Unit cases on `score`: one-of-three returned, three-of-three above
one-of-three, twenty occurrences not ten times two. Recompute existing oracles by hand:
`internal/store/search_test.go:153` `{twice:2, once:1, upper:1, spread:4}` becomes
`spread: 1+log₂4 = 3`; `internal/store/search_test.go:286` `float64(8)` becomes
`damp(3)+damp(5) ≈ 5.907`, so switch to `require.InDelta` or assert relative order.
`internal/store/search_test.go:437` is single-term and unchanged. Audit
`internal/knowledge/set_test.go:122` (comment only) and
`cmd/knowledge_test.go:245,253,772,780,788,1160` (stay green via `damp(1)=1`).
Float equality is now a hazard where integers were compared exactly.

**Phase 1.3** — `internal/knowledge/set_test.go`: a one-of-three match returned when
alone; absent once a three-of-three match is present; and the load-bearing case, the
weak and strong matches placed in **different stores**. Only that third case catches a
cutoff wrongly applied per store. Verify
`TestSet_SearchTieBreaksBySourceOrderThenPath` (`:157`) still passes, since its four
equal-score hits are all at the maximum and none should be cut.

**Phase 2.1** — Unit tests for `parseEntry`: tags present; **no block at all** (the
load-bearing case, since that is every entry that exists today); empty list; block with
only unrelated keys; malformed or unterminated block (returns raw bytes and no tags,
never an error); case-normalisation and de-duplication. Scan-level test that a tag is
not also counted as a body occurrence of itself. A `cmd/knowledge_test.go` test that
every knowledge subcommand runs unchanged against an entirely untagged fixture.

**Phase 2.2** — An entry tagged `go, http` whose body contains neither word is returned
for `go http router`. A tagged entry outranks a comparable prose mention. A long
many-occurrence entry does not outrank a short tagged one. Plus the `tagAffinity`
table described in that phase's notes, which is where a subtle prefix-matching mistake
is most likely and least visible. Extend
`cmd/knowledge_test.go:418` (schema field set) with `tags`. Note the fallout: adding
`Tags` to the `knowledgeHit` mirror struct (`cmd/knowledge_test.go:19-26`) makes the
existing literals assert `Tags: nil` against a JSON `[]`, so each needs
`Tags: []string{}`. Easy to miss, produces confusing nil-vs-empty diffs.

**Phase 2.3** — Tag filter excludes absolutely regardless of score; two tags require
both; the flag plumbs through to the store; the advertised schema declares `tag`. Two
assertions carry real weight here. First, a hit reported by a store that *ignored*
`opts.Tags` is still excluded from the results, which is what pins the knowledge layer
as the authority rather than the provider; a fake store returning everything is the
cleanest way to write it. Second, `ignoreStore` forwards the options, using a fixture
that *has* an ignore file, since a fixture without one cannot catch a dropped
option.

**Phase 2.4** — The benchmark. Fixture in `t.TempDir()` reconstructing the measured
scenario: the on-topic routing entry tagged
`[go, http, api, routing, handler, endpoints, middleware, chi]`, plus a long
early-concept document mentioning "go", "api" and "http" in passing. Table of the four
observed queries to expected first path. Plus the sharpest single assertion in the
plan: an entry tagged `api` whose body never contains "api" is returned for the query
`api`. Must be self-contained: adding
`decisions/ranked-or-over-boolean-and.md` to the live corpus moved the target from rank
1 to rank 2 on `http handler` during prototyping, so a test bound to the real directory
would flap.

**Phase 3.1** — Vocabulary listing returns tags with counts; ordering is count-desc then
alphabetical; narrowing by tier and filter is honoured; an untagged base returns an
empty array rather than an error; always-applied categories are excluded.

**Phase 3.2** — Extend the hand-maintained `expectedCRUDInvocations` slice at
`internal/agent/instruction_surface_test.go:94-99` with `knowledge tags`. Assert the
rendered skill instructs loading the existing vocabulary, proposing tags, and showing
them before writing, which is the automatable precondition for success metric 3.
Verify unchanged: `internal/agent/claude_test.go:26-32`, `bob_test.go:24-33,49`,
`codex_test.go:29`, `knowledge_trigger_test.go:117`, and
`templates/skill_resume_test.go:13-18` (which deliberately excludes `spek-knowledge`).

**Phase 3.3** — A guard asserting the rendered skill carries the audit intent and its
per-entry confirmation language, as a hand-maintained literal slice following the
existing pattern.

**Phase 4.1** — No automated tests; prose only. Grep for remaining claims that a
document must contain every query word. Additionally, read the formula and constants
written into the document against `internal/knowledge/ranking.go` and confirm they
match, and check the worked example's arithmetic by hand.

**Phase 4.2** — `npm run build` and `make check` (`npx astro check`, zero errors and
zero warnings) in the docs root, plus
`grep -nE "<div|<section|class=" src/pages/*.mdx` returning zero matches. CI runs build
only, so `astro check` must be run locally. Because this phase publishes the constants,
two manual checks matter as much as the build: the published `TAG_WEIGHT`,
`COVERAGE_EXPONENT`, `MIN_PREFIX` and `CUTOFF_FRACTION` must match
`internal/knowledge/ranking.go` as shipped, and the worked example's arithmetic must be
recomputed by hand against those values rather than copied from the plan.

**Phase 4.3** — Same build and typecheck as 4.2. Additionally, read the published
`Store` interface and `Hit` blocks against
`spektacular:internal/store/store.go:25-57` line by line and confirm every field and
signature matches. There is no automated guard tying this page to the Go source, which
is how it drifted in the first place; the check is manual and deliberate.

**Phase 4.4** — Verified by searching for the entry's subject and confirming it is
returned, which doubles as a live end-to-end exercise of the whole plan.

### Success metric verification

| Metric | How verified |
| --- | --- |
| The `api` case reverses | **Behavioural** — Phase 2.4: an entry tagged `api` whose body never contains "api" is returned for `api`. |
| The observed-query benchmark goes from 0 to 4 | **Behavioural** — Phase 2.4: table of four queries to expected first path, over a self-contained fixture. Prototype confirmed 4/4 is achievable. |
| Newly captured entries carry tags | **Manual — captured in the implementation test plan.** The precondition (rendered skill instructs proposing tags before writing) is asserted behaviourally in Phase 3.2. |
| The tag vocabulary converges rather than fragments | **Manual — captured in the implementation test plan.** Two supporting properties are behavioural: tags normalise to lower case on read, so `http` and `HTTP` cannot both exist; and the vocabulary listing orders by how widely a tag is used. Semantic near-duplicates (`http-api` beside `http`) are agent judgement and stay manual. |

### Deliberate gaps

No benchmark against the live knowledge directory (it would flap as the project writes
knowledge). No end-to-end agent test of the confirmation contract (enforced by prose,
and no harness drives an agent through a confirmation gate). No performance test (a
logarithm per term per document, against a scan that already reads every byte and
computes a SHA-256). No test of the docs site's prose beyond build and typecheck.

## Project References


- Spec: `.spektacular/specs/000050_knowledge-entry-tags.md`
- Working context and measured baseline: `.spektacular/context.md`
- Binding decision: `.spektacular/knowledge/decisions/ranked-or-over-boolean-and.md`
- Maintainer documentation to correct: `docs/knowledge-base.md:78-93,185-197`
- Prior plans: `000028_knowledge-base-categories-tiers-and-dedup` (category registry,
  tier split, exact-byte de-dup), `000047_repo-scoped-knowledge-addressing` (address
  and selector, `--tier`/`--filter` shape)
- Prior spec: `000022_spek-knowledge-skill` (the propose-then-confirm capture contract)
- Source issue: [#35](https://github.com/jumppad-labs/spektacular/issues/35)

## Token Management Strategy


| Tier | Token Budget | Agent Strategy |
|------|-------------|----------------|
| Low | ~10k | Single agent, sequential |
| Medium | ~25k | 2-3 parallel agents |
| High | ~50k+ | Parallel analysis, sequential integration |

Per-phase estimates and strategies are recorded with each phase above. No phase in this
plan is High tier. Total across the fourteen phases is roughly 243k tokens, so the plan
is comfortably executable phase by phase without a single agent needing to hold the
whole thing at once.

Two phases carry most of the coordination cost and are worth reading in full before
starting: Phase 1.2 (score-dependent assertions are spread across three packages) and
Phase 3.2 (five template files plus two regenerated copies plus four guard tests).

## Migration Notes


**There is no migration.** This is a hard constraint from the spec, and the design
satisfies it structurally rather than by a migration step:

- An entry with no frontmatter block yields no tags and its raw bytes as its body. No
  entry is rejected as invalid, no backfill is needed, and no command has to be run.
- The tag frontmatter block is optional forever, not a transitional state.
- Verified during planning: no `.md` entry in either repo's knowledge store currently
  opens with a `---` line, so no existing entry will be newly reinterpreted.
- An unterminated or unparseable leading block is treated as "no frontmatter" rather
  than as an error, so even a future entry opening with a bare horizontal rule degrades
  to today's behaviour instead of becoming unsearchable.

**One thing does change silently for existing users**: search results. A query that
returned nothing may now return something, and a query that returned ten results may
now return three. That is the intended behaviour, and it is why Phase 4.1 and Phase 4.2
correct the documentation that currently promises the old semantics. Anything treating
"no results" as proof of absence is relying on a guarantee search no longer makes, as
`decisions/ranked-or-over-boolean-and.md` already records.

**The checksum is deliberately unchanged in meaning.** It continues to cover the file's
exact raw bytes including the frontmatter, so two entries differing only in their tags
stay distinct during exact-byte de-duplication (`docs/knowledge-base.md:100-104`) rather
than collapsing into one candidate.

## Performance Considerations


The change is not plausibly material, and no performance work is in scope.

- **Per document, per term**: one `math.Log2` call plus one exact string comparison
  against the entry's tags. This sits on top of a scan that already opens the file,
  reads every byte, computes a SHA-256 over all of them, and does a case-insensitive
  substring count per term per line.
- **Frontmatter parsing**: a prefix check on the file's first four bytes, and for the
  small minority of files that have a block, a YAML unmarshal of a few lines. A file
  with no leading fence takes the body path from line one and must gain no measurable
  cost; this is worth preserving deliberately rather than by accident.
- **Ranked OR increases work slightly**: documents that were previously abandoned as
  soon as one term's count hit zero are now scored. But the abandonment happened
  *after* the full file scan, in the walk callback, so no additional I/O results, only
  arithmetic.
- **The tag vocabulary listing is O(entries) I/O**: it reads every entry in the covered
  stores. This matches what `AlwaysAppliedEntries` (`internal/knowledge/set.go:262-296`)
  already does on every planning task, so it introduces no new class of cost. It runs
  once per capture, not per search.
- **No index, no cache.** Search remains a full scan. If a knowledge base grows large
  enough for that to hurt, it will hurt with or without this change, and addressing it
  is distinct work.

