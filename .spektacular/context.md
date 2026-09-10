# Working context — 000050_knowledge-entry-tags

Source: GitHub issue [#35](https://github.com/jumppad-labs/spektacular/issues/35)
and its two follow-up comments. Everything below was established empirically
while preparing the "Spec Driven Development with Spektacular and Bob" demo.

## The problem

`knowledge search` ranks by summing query-term occurrences across a document.
With no length normalisation, a long document that mentions a query term
incidentally outranks a short document that is entirely about it.

The practical consequence: an agent runs its discovery search, gets a
confident-looking but irrelevant top hit, reads that one, and never reads the
entry that would actually have changed the plan.

## Measured evidence

A probe entry (a routing standard: "all Go services route HTTP endpoints through
chi, middleware mounted in a fixed order") was written to
`architecture/http-routing.md` in this repo and removed afterwards.

| Query | Top hit | Score | Routing doc |
| --- | --- | --- | --- |
| `http endpoints go` | `architecture/initial-idea.md` | 40 | 2nd, score 6 |
| `http routing` | `architecture/initial-idea.md` | 37 | 3rd, score 6 |
| `routing` | `architecture/initial-idea.md` | 21 | **not in top 4** |
| `http middleware` | `architecture/http-routing.md` | 6 | 1st, only hit |

`initial-idea.md` won three of four queries. It is a long early-concept document
that mentions "Go", "API" and "http" in passing and says nothing about routing.
On `routing` — the single most on-topic word available — the document actually
about routing did not surface at all.

## Field evidence: the query is not stable between runs

Real plan runs with Bob, same spec, same repo:

| Run | Query Bob issued | Result |
| --- | --- | --- |
| 1 | `http endpoints go` | matched (score 21 against a populated store) |
| 2 | `http handler` | matched, rank 1 and only hit, score 13 |
| 3 | `http endpoints handler routing` | matched |

Nothing about the spec changed between runs. The design surface the agent named
for itself did.

The entry survived all three only because it redundantly contained "endpoints",
"go", "handler" *and* "routing" — an accident of drafting, not a property anyone
designed for. The control case: **`api` returns nothing**, out of 10 hits,
despite being an entirely plausible surface name for exactly that work. The
entry is about HTTP API routing and never uses the word "API".

Rewording the same rule to repeat agent vocabulary in headings and prose, with no
change in meaning or length, moved its score from 6 to 21. **Authors are
currently incentivised to keyword-stuff, and an author who writes plainly is
penalised.**

## Why tags specifically

The author cannot write for the query, because the query is chosen by the agent
at discovery time and varies run to run. The only currently available mitigation
is packing an entry with every synonym an agent might pick — which degrades the
document for human readers and still cannot be exhaustive.

Tags decouple *what an entry is about* from *which words it happens to contain*.

## Scope discussed

1. **Tags on knowledge entries** (frontmatter or equivalent), weighted well above
   body-term occurrences.
2. **Length normalisation** — BM25-style, or at minimum divide by document
   length, so a long document cannot dominate by volume alone.
3. **Title and heading matches weighted above body matches.** The probe's rank
   improved largely because vocabulary reached its headings; that should be an
   explicit ranking signal rather than a side effect of term counting.
4. **The capture flow should propose tags.** Tags only pay off if they are
   applied, and asking an author to invent a taxonomy by hand is the friction
   that leaves entries untagged.

## The symmetry argument (user's framing, worth preserving)

The agent that *searches* the knowledge base and the agent that *saves* to it are
the same kind of agent. If the writing agent proposes the tags it would itself
search for, retrieval and capture converge on one vocabulary by construction —
rather than a human guessing at the terms a model will later choose, which is the
failure this issue is about.

Capture already proposes tier, store name, path and content and waits for explicit
confirmation. Tags become one more proposed field in that same moment: shown,
editable, confirmed with everything else, never written silently. No change to the
"never persists unprompted" rule.

Write time is also the only moment the context exists — the agent is proposing an
entry because of something it just did.

## Design note carried from the issue

**Suggest from what already exists.** Free-text tags drift fast (`http` / `HTTP` /
`http-api` / `web`), and a fragmented vocabulary retrieves worse than none. The
suggestion step should read the tags already present across the stores in scope
and prefer an existing tag over minting a near-duplicate, proposing a new one only
when nothing fits. That keeps the vocabulary converging as the knowledge base
grows, without anyone curating it.

## Scoping note: retrieval is the only weak link

On the run where the entry matched, everything downstream worked: Bob read the
entry in full, extracted the constraint, and carried it into the plan, which chose
chi over `net/http` and cited the document. This is not a comprehension or
plan-integration problem. The entire failure mode lives in whether the entry is
returned at all.

## Out of scope (established)

The exclusion of `conventions` and `glossary` from search results is correct and
not in scope. Those are always-applied and loaded in full, so surfacing them in
search would duplicate them. This concerns ranking within the looked-up categories
only: `architecture`, `gotchas`, `learnings`, `decisions`.

---

# Plan workflow — 000050_knowledge-entry-tags

Plan started 2026-09-10 against the completed spec. The sections above are the
spec interview's context and remain the authoritative background for *why*.
Everything below is plan-workflow working context: decisions taken while
designing the implementation.

## Plan decisions log

**Discovery.** Two repos in scope: `spektacular` (search engine, knowledge layer,
CLI, skill template) and `docs` (published documentation only). Full research in
`.spektacular/work/000050_knowledge-entry-tags/research.md`; judgement calls in
`assumptions.md` beside it.

Key decisions taken so far:

- **Ranked OR is new work, not existing behaviour.** `internal/store/search.go:70-75`
  still requires every query term. The binding decision entry
  `decisions/ranked-or-over-boolean-and.md` states the target; knowledge outranks
  the code, so it is planned as work to do.
- **Scoring splits at the store/knowledge seam.** Per-document score in the store
  (one file read, already there); cross-store merge, global rank and the
  best-hit-relative cutoff in `knowledge.Set.Search`, because the strongest hit may
  live in another store.
- **Tags are YAML frontmatter, parsed by a new small reader.** `internal/metadata`
  cannot be reused: it hard-requires `created_date` and a valid `status`, and its
  `Render` drops unknown keys, so a tags-only block would be rejected and any
  round-trip would delete the tags.
- **`--tag` follows the `--tier`/`--filter` shape** on `knowledge search`, per the
  spec's technical steer.
- **Tag proposal stays a skill judgement**; the CLI only reports the vocabulary in
  use (a mechanical `knowledge tags` listing), mirroring how `knowledge categories`
  already feeds the contribute flow.

**Validated by prototype.** A Python prototype over this repo's real knowledge base,
with the routing probe recreated and tagged, scored
`(Σ [8·isTag(t) + damp(count(t))]) × coverage²`, `damp(n)=1+log₂(n)`, cutoff
`0.25 × best`. All four observed queries (`http endpoints go`, `http handler`,
`http endpoints handler routing`, `api`) put the on-topic entry **first** — the
spec's "0 to 4" success metric. `damp(1)=1` keeps every existing `Score: 1`
assertion green.

**Open question to raise at the walkthrough.** The acceptance criterion "Volume does
not beat relevance" is planned as satisfied by the *tag weight*, with a tagged short
entry vs an untagged long one. Damping alone cannot reverse that pairing, and both
alternative signals (length normalisation, title weighting) are explicit Non-Goals.
If the user meant it to hold between two untagged entries, the scoring model needs
length normalisation and the Non-Goals need revisiting.

**Deferred offer.** The `internal/metadata` incompatibility (assumption 4 in
`assumptions.md`) is a durable, non-obvious gotcha worth a knowledge entry. Offer it
at the walkthrough step, which is where the workflow handles capture offers.

**Assembly.** All eleven section working files plus `assumptions.md` are complete under
`.spektacular/work/000050_knowledge-entry-tags/`. The three documents are staged to
`.spektacular/tmp/{plan,context,research}_template.md`. Note `milestones.md` and
`phases_plan.md` both carry `### Milestone N:` headings, so they were **interleaved**
(milestone heading, its What-changes/Validation body, then its phases) rather than
concatenated. All 13 `*Technical detail:*` anchors and both `research.md`
cross-references were verified to resolve.

**Plan shape as assembled.** 4 milestones, 13 phases, no High-complexity phase, roughly
215k tokens total. Milestone 1 (ranked OR, no tags) is deliberately separable from
milestone 2 (tags) so the riskiest change (invalidating existing score oracles) has its
own validation point.

**Resolved during planning, not parked.** Checked every `.md` entry in both repos'
knowledge stores for a leading `---`: none has one, so frontmatter-aware scanning
cannot misread an existing entry. Only two genuine open questions remain, both recorded
in the plan: post-tuning benchmark stability, and the "about that term" reading of the
volume-vs-relevance criterion.

**Committed.** All three documents are in the plan store at
`.spektacular/plans/000050_knowledge-entry-tags/`. The per-section working directory
has been removed. Three fixes were applied during verification: context.md's
`Project References` moved after `Testing Strategy`; research.md's
`Chosen approach: evidence` renamed to the scaffold's `Chosen approach — evidence`
(same anchor, so cross-references still resolve); and `Drafting assumptions` moved
before `Rehydration cues`. research.md's leading `Repos in scope` became bold preamble
so the required `##` sections start where the scaffold expects.

**Remaining: the walkthrough.** Two things to raise with the user there:
1. The "about that term" reading of the volume-vs-relevance criterion (Open Question 2).
   This is the one thing most likely to be wrong in the plan.
2. The deferred knowledge-capture offer: the `internal/metadata` incompatibility is a
   durable, non-obvious gotcha worth an entry via `spek-knowledge`.

**Walkthrough correction (applied).** The user asked what happens when a new store is
implemented. That exposed a real flaw in the committed design: scoring lived inside
the store, which makes the ranking formula a contract every future provider must
reproduce exactly, and any drift silently corrupts the cross-store merge and the
best-hit-relative cutoff.

Revised design: **stores report evidence, the knowledge layer ranks.** A store returns
per-term body occurrence counts plus the entry's tags (and locator, title, excerpts,
checksum) and computes no score. `knowledge.Set.Search` tokenizes the query, scores
every hit through one function, merges, sorts, applies the cutoff, and enforces the
tag filter. Precedent already in the type: `Hit.Tier`/`Name`/`Category` are documented
as "left empty by the store and populated by the knowledge layer"; `Score` joins them.

Notable knock-ons: `Store.Search` becomes `(terms []string, opts SearchOptions)` so
tokenization is shared and evidence indexing agrees by construction; `SearchOptions` is
introduced empty in Phase 1.1 so Phase 2.3 is purely additive; the store's tag filter
becomes an optimisation while the knowledge layer is the authority; Phase 1.1 is
retitled "Move ranking above the storage layer" and raised Low → Medium (3 packages,
not 1 file); scoring tests move to `internal/knowledge`.

**Accepted limit, now documented in the interface:** a provider must be able to report
per-term occurrence counts to participate in ranking. A purely semantic backend that
returns only its own opaque relevance score cannot, and is out of scope.

**Original rejection was wrong and is corrected in research.md**, with a note recording
that the first draft rejected knowledge-layer scoring on the false grounds that it
would force a second read of every file. It confused where a score is computed with
where a file is read.

**Second walkthrough correction (applied).** The user asked whether the docs would be
updated too. They were not, fully: the first revision fixed the architecture and missed
its own documentation consequence.

`docs:src/pages/extending.mdx` is the published reference for writing a storage
backend. It carries the `Store` interface block (`Search(query string)` at `:47`), the
`Hit` struct (`:59-65`) and the method-contract bullets (`:78-84`) — exactly the
contract this plan changes. Added **Phase 4.3: Correct the storage-backend reference**
(repo `docs`, Low, ~12k), with a Content example per the docs repo's
`plan-content-pages` convention. The knowledge-entry phase renumbered 4.3 → 4.4. Plan
is now 4 milestones / 14 phases / ~233k tokens.

The published `Hit` on that page is **already wrong today**, independently of this
work: it shows `Excerpt string` (singular) where the code has `Excerpts []string`, and
omits `Title`, `Category` and `Checksum`. Phase 4.3 fixes that in the same pass rather
than shipping a second inaccurate version.

Root cause not addressed and flagged as a follow-up: nothing ties `extending.mdx` to
the Go source, so nothing caught the drift and nothing will catch the next one. Closing
that gap is out of scope for this plan.

**Pattern worth noting across both corrections:** the user's two questions each found a
consequence one layer out from where I was looking — first the provider dimension of a
layering choice, then the documentation dimension of a contract change. Worth checking
"what else publishes this contract?" whenever an interface moves.

**Third walkthrough correction (applied).** User: "We should be detailing the search
algorithm and frontmatter in the docs too." This **reverses** an earlier call of mine.

The earlier draft deliberately described ranking qualitatively and kept numbers off the
page, reasoning "no numbers, so tuning the weights does not date the page". Now the docs
publish the algorithm in full: tokenization, the two kinds of per-term evidence, the
`damp` / `termScore` / `coverage` / `score` formula, the shipped constants, the relative
cutoff, ordering and tie-breaks, plus a worked two-entry example carried through to a
number (verified by hand: A=18, B=1, cutoff 0.25×18=4.5 cuts B). And the frontmatter
contract in full: block placement and fencing, both YAML list forms, `tags` as the only
key read, lower-casing and de-duplication, exact matching, block excluded from body
text, optionality, malformed-block tolerance.

Structural consequence: the content outgrew the lifecycle section, so Phase 4.2 now adds
**two** new bands ("How a search is ranked" `surface`, "Tagging an entry" plain) after
the lifecycle section. Exactly two, in that order, preserves the page's plain/surface
alternation with **no downstream edits**; one band would force every band below to flip.
Phase 4.2 raised Low → Medium, ~16k → ~22k. Phase 4.1 gains the same detail plus a note
on *where* each part is computed. Plan now ~243k tokens.

**Accepted cost, and the mitigation.** Publishing the constants couples the docs to the
code: tuning a weight is now a four-file change (`internal/knowledge/ranking.go` plus
three documents that quote it). Mitigation: the named constants in the code are the
single source of truth, each document says so, the cross-check is in the acceptance
criteria of 4.1 and 4.2, and the obligation is stated inside the 4.4 knowledge entry
itself, since a planning agent retrieving it is the likeliest reader about to change a
weight. Documentation is already sequenced after tuning, which limits the exposure.

**Fourth walkthrough correction (applied).** User, on hearing the exact-matching
decision: tag matching needs a specific instruction in the capture skill, covering
plurals (`apple`/`apples`) and derivatives (`http`/`https`, where the trailing `s` is
not a plural).

This exposed a real tension the plan had not resolved. The convergence rule was drafted
as "propose `http` rather than a near-duplicate such as `HTTP` or `http-api`". Read
naively that licenses collapsing `https` into `http`, which loses a genuine
distinction. `HTTP` and `http-api` are spellings of one subject; `https` is a different
subject.

Three tag-form rules now written into the capture flow as load-bearing prose (concrete
wording drafted in context.md Phase 3.2 so the implementer does not invent it):
1. Carry every form genuinely likely to be searched (`test` **and** `testing`), rather
   than picking a canonical one. Judgement, not exhaustiveness.
2. Never derive a variant mechanically. A trailing `s` is not a plural: `https`, `tls`,
   `dns`, `aws`, `cors`, `css`, `js`, `redis`, `kubernetes` are whole words.
3. An explicit boundary on prefer-an-existing-tag: reuse when it means the same thing,
   never collapse a distinct subject into a similar-looking existing tag.

Propagated to: Phase 3.2 (summary, 3 new acceptance criteria, concrete skill prose),
Phase 3.3 (the audit inherits the rules explicitly, since the audit is the likeliest
place to over-apply convergence; 2 new criteria), Phase 4.2 (a new item 4 in the
docs Tagging section, plus an acceptance criterion, since users write entries too).

**Residual risk accepted and recorded**: this is prose guidance to an agent, not a
mechanism. Nothing enforces it and a bad tag set still fails silently; the audit is the
backstop. A mechanical aid (warn when a tag is a strict prefix of another) is a
follow-up if this proves common in practice, not part of this plan.

**Also fixed this turn**: three drafting assumptions had gone stale against earlier
revisions (the tag-filter split, the docs-band decision, the effort shape). Corrected
before walking them. Assumption count 38.

**Scope note from the user**: a metrics/observability surface for the ranking is the
intended next piece of work, to be specced separately. Recorded in the plan's Out of
Scope with a pointer to Open Question 1, and explicitly not a prerequisite.

**Fifth walkthrough correction (applied).** User proposed partial tag matching with a
reduced score (`https`=100, `http`=80, `https-security` matches weakly) and asked
whether a word-distance algorithm exists. Adopted, but **not** with edit distance.

**Measured during the walkthrough.** On every pair that should match, edit distance and
prefix matching give identical scores (`http`/`https` 0.80, `apple`/`apples` 0.83,
`https`/`https-security` 0.36). On near-misses they diverge badly: edit distance scores
`test`/`rest` 0.75, `cors`/`core` 0.75, `tls`/`tld` 0.67, `dns`/`dos` 0.67, `api`/`apt`
0.67, all unrelated subjects and all plausible tags here. Prefix scores them 0. Edit
distance is worst exactly where tags live (short technical tokens) and buys nothing.

**Adopted**: bidirectional prefix with proportional partial credit.
`tagAffinity = len(shorter)/len(longer)` when one string opens the other and the shorter
is >= `minPrefixLen` (4); 1.0 on exact; strongest matching tag wins, affinities do not
sum. `Evidence.TagHits []bool` → `TagAffinity []float64`; `termScore = tagWeight *
affinity + damp(bodyCount)`. The user's own 100/80 falls out of the length ratio (4/5),
which is what confirmed the shape.

**Two guards are load-bearing**: anchor at the start (stops `test` hitting `latest`) and
minimum length (stops `go` hitting `golang`). Dropping either reintroduces the
substring failure that ruled out unanchored matching.

**Known gap accepted**: `route` does not match `routing`. Prefix matching is positional,
not morphological. Stemming stays a Non-Goal.

**Knock-on**: the tag-form prose from the fourth correction had to *shrink*, or it would
now mislead. "Carry both `apple` and `apples`" became wrong advice. Rules reduced to:
one form per subject; a second tag only where a form differs by more than its ending;
never collapse distinct subjects. The audit gained the inverse guard: report a tag made
redundant by prefix matching as removable.

**ACTION OUTSTANDING — spec amendment.** The spec's Non-Goal says "Query expansion —
stemming, synonyms, or plural handling — is not addressed. `router` and `routers` remain
distinct terms." Under this plan they match at 6/7. Plan and spec now contradict on
scope, and the spec governs scope. Recorded as item 0 in the plan's Open Questions with
proposed replacement wording. **Not yet made — needs the user's approval**, since the
plan workflow does not own the spec artifact. Raise before Phase 2.2 if still open.

Assumptions now 40 (two superseded entries retained with their reasoning).

**Terminology note**: "decay" was unhelpful jargon of mine. Use **partial credit** in
all user-facing text.

**Prior art check (user asked: surely this is already solved?).** It is. Verified at the
walkthrough and recorded in research.md with citations:
- Prefix matching is standard: **edge n-grams** in Lucene/Elasticsearch expand a term
  into every prefix at index time. Same relation, computed at scan time here because
  Spektacular has no index (full filesystem walk, no analyzer pipeline).
- The short-term guard is standard: Elasticsearch **`fuzziness: AUTO`** allows 0 edits
  for 1-2 chars, 1 for 3-5, 2 for 6+. Corroborates `minPrefixLen = 4`.
- **The one genuine departure**: Lucene's `PrefixQuery` uses a *constant score* -
  prefix matches are binary. It can afford that because BM25/IDF/field-norms
  discriminate downstream. This project has none of those, and length normalisation is
  a Non-Goal, so discrimination has to live in the match itself. That is what the
  length ratio does. Do not "simplify" it to constant-score by copying Lucene.
- **Stemming (Porter 1980 / Snowball) is the established answer we decline**, recorded
  as knowingly rejected rather than unknown.

**Sequencing set by the user**: ship this plan → build metrics → then evaluate stemming
against those metrics. Stemming is the expected next lever but must come after metrics,
because a stemmer behaves unpredictably on technical tokens (`https`, `kubernetes`,
`redis`) and adopting one without being able to measure whether hits moved is guessing.
Recorded in the plan's Out of Scope alongside the metrics entry.

**Spec amended (user approved).** `.spektacular/specs/000050_knowledge-entry-tags.md`
Non-Goals: "Query expansion - stemming, synonyms, or plural handling - is not addressed.
`router` and `routers` remain distinct terms." replaced with a narrowed exclusion
(stemming and synonyms only, `route`/`routing` remain distinct) plus a dated note
recording what changed and why. Plan and spec now agree. Open Questions item 0 updated
from outstanding action to resolved.

**Walkthrough status**: the user has reviewed approach, phases, scope and the four
load-bearing drafting assumptions, and declined the remaining two assumption groups by
moving on. Explicit sign-off still to be confirmed before `goto finished`.


---

# Implement session — 000050_knowledge-entry-tags

Started 2026-09-10. Plan selected by the user from `plan file list`; branch
`b-knowledge`, clean at start.

## Validation gate (step `read_plan`)

- **Structural**: all ten required `##` sections present in `plan.md`. Fourteen
  `#### - [ ]` phases across four milestones, every one carrying a
  `*Technical detail:*` link that resolves to a matching `### Phase N.M:` heading in
  `context.md`. Pass.
- **Spec coverage**: all 13 requirements and all 15 acceptance criteria in
  `.spektacular/specs/000050_knowledge-entry-tags.md` map to a phase. No gaps, no
  descoped requirements needed.
- **Changelog mode**: no `## Changelog` section in `plan.md` → **first-phase**
  invocation. Pick up at Phase 1.1.

## Drift check: no missing targets, but line numbers have moved

Every file, package, symbol, command and template named in `plan.md` and
`context.md` exists in the repo it is attributed to. **No mismatch requires the
plan to be fixed.** Several *line* references have drifted since the plan was
written; the symbols are the authority, not the line numbers:

| Plan reference | Actual |
| --- | --- |
| `internal/store/store.go:25-36` (`Hit`) | `:25-33` |
| `internal/store/store.go:38-57` (`Store`) | `:35-58`, `Search` at `:57` |
| `internal/knowledge/address.go:159-172` (`Selector.covers`) | `:115` |
| `internal/knowledge/set.go:178-189` (post-filter loop) | `:170-186` |
| `internal/knowledge/set.go:201-208` (`alwaysAppliedSet`) | `:199-206` |
| `internal/knowledge/set.go:216-222` (`Set.Write`) | `:234` |
| `internal/knowledge/set.go:262-296` (`AlwaysAppliedEntries`) | `:275` |
| `internal/knowledge/set.go:412-434` (`listFiles`) | `:430` |
| `cmd/knowledge.go:495-506` (`knowledgeSelector`) | `:471-473` |
| `cmd/knowledge.go:520-526` (narrowing-flag loop) | `:535-539` |
| `cmd/knowledge.go:529` (`AddCommand`) | `:541` |

Confirmed exact: `internal/store/search.go:46,70-77,126-133,135-215`;
`internal/store/ignore.go:107`; `internal/knowledge/set.go:140`;
`cmd/knowledge.go:107-112,114`; `internal/store/search_test.go:153,286,437`;
`internal/knowledge/set_test.go:124,157`;
`cmd/knowledge_test.go:19,245,253,418,772,780,788,1160,1666`;
`internal/agent/instruction_surface_test.go:93`;
`templates/skills/workflows/spek-knowledge/SKILL.md` (83 lines, intents at
`:27,48,64,79`); `docs/knowledge-base.md:78,185`; `README.md:137`;
`docs:src/pages/knowledge-base.mdx:125` and its five `<Section>` bands at
`:17,44,94,152,247`; `docs:src/pages/extending.mdx:47,59,63,64`.

## Stale statement in research.md, superseded by the plan

`research.md` Open assumption 4 ("Matching a tag is exact, not substring") is a
pre-walkthrough leftover. The plan supersedes it with **bidirectional prefix
matching with proportional partial credit** (`minPrefixLen = 4`). Follow the plan,
not that assumption.

## Two questions the plan says to STOP on

1. **Weight tuning.** Tune freely while the Phase 2.4 benchmark stays 4/4 and the
   acceptance criteria of 1.2, 1.3 and 2.2 hold. Escalate only if no set of values
   satisfies all of them, which would mean the formula's *shape* is wrong.
2. **"Volume does not beat relevance"** (Open Question 2). The plan reads "about
   that term" as "tagged with that term", so the Phase 2.2 fixture pairs a **tagged**
   short entry against an **untagged** long one. Ask the user before writing that
   test with two untagged entries.

## Repo roots (from `repo list`)

- `spektacular` → `/home/nicj/code/github.com/jumppad-labs/spektacular` (phases 1.1
  through 3.3, 4.1, 4.4)
- `docs` → `/home/nicj/code/github.com/jumppad-labs/spektacular-website` (phases 4.2,
  4.3)

## Phase 1.1 analysis

Research done in the main context rather than by sub-agents: the validation gate
had already read every file the phase touches, and the phase spans three small,
tightly coupled call sites.

**Confirmed integration points.** `Store.Search` has exactly the three sites the
plan promised: `internal/store/search.go:46` (`FileStore.Search`, which delegates
to the unexported `f.search(terms)` at `:56` — the only caller of that helper),
`internal/store/ignore.go:107` (pass-through decorator), and
`internal/knowledge/set.go:153` (sole consumer). `cmd/knowledge.go:331` calls
`set.Search(query, selector)`, whose signature does **not** change.

**Test call sites the signature change reaches (mechanical, no expected value
moves).** 25 store-level calls of the form `st.Search("query")`:
`internal/store/search_test.go:43,72,82,94,115,121,135,167,200,224,255,281,306,327,352,376,395,419,441`
and `internal/store/ignore_test.go:209,273,275`. Phase 1.1 acceptance criterion 4
forbids changing *expected values*, not adapting a call to a new signature.
`search_test.go:115,121` assert an empty/whitespace query returns nothing; with
tokenization moved up, those become empty-`terms` calls and the query-level
version of the assertion moves to the new `Terms` test in `internal/knowledge`.

**Decision: the store still skips documents with no evidence at all.** The plan
says the walk callback "stops excluding on a zero count". Read literally that
would emit a `Hit` for every file in the tree, including binaries and total
non-matches, which is noise the knowledge layer would immediately score at 0 and
drop. Ranked OR means "missing *some* terms is still a match"; a document
matching *no* term is not a candidate at all. So the callback keeps one guard —
skip when every term's count is zero — and drops the per-term early return that
implemented boolean AND. Behaviour is identical either way because `score`
returns 0 for no evidence and the knowledge layer drops zero-scoring hits; this
just avoids materialising the whole store as hits. Comment both ends of the
relocation so a later reader does not restore the AND guard.

**Shape being written.** New `internal/knowledge/ranking.go`: exported
`Terms(query)` and `Evidence{BodyCounts, TagAffinity}`, unexported `score`, and
the named tunables. `TagAffinity` is declared but unread until Phase 2.2, per the
plan. `store.SearchOptions` lands empty now so Phase 2.3 is additive.
`Hit.BodyCounts []int` carries `json:"-"`, and `Score` moves into the
knowledge-layer-stamped field group with the other three.

### Phase 1.1 implementation notes

Written: `internal/knowledge/ranking.go` (new), and edits to
`internal/store/store.go`, `internal/store/search.go`, `internal/store/ignore.go`
and `internal/knowledge/set.go`. `go build ./...` passes.

Decisions taken while writing, beyond what the plan spelled out:

- **`Evidence.TagAffinity` is left nil in 1.1, not allocated zero-length.** The
  plan said the field is "declared here but stays zero for every term until Phase
  2.2". A nil slice is exactly that, and `score` does not read the field yet, so
  allocating one zero-filled slice per hit would be a dead allocation on every
  search. Phase 2.2 populates it for real.
- **`FileStore.search` takes `_ SearchOptions`.** The options value is threaded
  through the whole path now so Phase 2.3 is purely additive, but nothing in 1.1
  reads it. The blank parameter name says so rather than implying an unused
  field was forgotten.
- **`tagWeight` and `coverageExponent` are declared but unread.** Go permits
  unused package-level constants, and declaring them now keeps the tunables in
  one place from the start, which is the point of the phase.
- **`Set.Search` returns early on an empty term list.** Tokenization moved up, so
  this is now where "an empty or all-whitespace query searches nothing" is
  decided. `FileStore.Search` keeps its own `len(terms) == 0` guard, since a
  store must stay correct when called directly.

`store.Hit` field order changed (store-populated group first, stamped group
second). Field order is not part of the JSON contract, and no test constructs a
`Hit` positionally — confirmed by grep — so this is safe.

### Phase 1.1 tests — and a genuine conflict inside the plan

Test authoring surfaced an internal inconsistency in the plan's own Phase 1.1,
worth recording because a later phase or a reviewer will hit the same thing.

**The conflict.** Phase 1.1 acceptance criterion 4 says "Every pre-existing test
in the affected package passes without its expected values being changed", and
context.md's file-change list says "No existing test changes in this phase".
But the same file-change list also says the walk callback "stops computing a
score **and stops excluding on a zero count** ... **Exclusion moves out of the
store entirely**", and criteria 2 and 3 require that scoring leave the store
altogether. Three pre-existing tests assert precisely the two behaviours being
relocated, from a bare `FileStore`:

- `TestSearch_ScoreSumsOccurrencesAcrossDocument` — asserts `Hit.Score` from the store
- `TestSearch_OneHitPerDocument` — asserts `Hit.Score` from the store
- `TestSearch_MultiWordScatteredAcrossLines` — asserts the store applies boolean AND

No implementation can satisfy criterion 4 and criteria 2/3 simultaneously for
those three. This is a plan-internal contradiction, not code drift and not a
broken refactor.

**Resolution taken (no user escalation).** The three tests were *relocated* to
`internal/knowledge/set_test.go` and now assert through `Set.Search`, carrying
every fixture byte and every hand-computed expected value across verbatim
(`{twice:2, once:1, upper:1, spread:4}`, `[]string{"scattered.txt"}`,
`float64(8)`). Nothing was recomputed or weakened — only the layer the assertion
travels through changed, which is exactly the layer the behaviour moved to. That
honours criterion 4's *intent* (no expected value changed) while letting criteria
2 and 3 hold. Renamed to the destination file's `TestSet_...` convention; each
carries a comment recording that it moved up in Phase 1.1 and why.

**Applies to later phases:** whenever a plan phase says "no existing test
changes" but also relocates a responsibility across a package boundary, prefer
relocating the assertion with its oracle intact over rewriting or deleting it.

**New tests added.** `internal/knowledge/ranking_test.go` — `TestTerms_TokenizesQuery`
(6 cases) and `TestScore_FromEvidence` (7 cases, including the still-in-force
boolean-AND zero case, commented as the historical rule Phase 1.2 replaces).
`internal/store/search_test.go` — `TestSearch_ReportsEvidenceWithoutScoring`,
which pins criterion 3: the store leaves `Score` zero and `Tier`/`Name`/`Category`
empty while reporting `BodyCounts`, and reports partial-evidence documents rather
than filtering them. `internal/knowledge/set_test.go` — `TestSet_SearchStampsScoreOnHits`,
plus a `singleSourceSet(t, dir)` helper beside the existing `twoScopeSet`.

**Verified**, not taken on report: `go build ./... && go vet ./... && go test ./...`
all clean from the spektacular root. `make lint` is `go vet` only and CI (Dagger,
`dagger/main.go:190`) runs `go test -race ./...`, so the deliberately-unused
`tagWeight`/`coverageExponent` constants trip nothing; an editor LSP flags them,
which is expected until Phases 1.2 and 2.2 consume them.

### Phase 1.1 verification

Run from the `spektacular` root, all exit 0: `make lint` (which is `go vet ./...`),
`make build`, and `go test -race -count=1 ./...` uncached — 20 packages ok, zero
failures, no race reports. The race flag matters because CI runs exactly that
(`dagger/main.go:190`).

Criteria mapping: 1 and 4 are the uncached full suite passing with the three
relocated oracles intact; 2 is `internal/knowledge/ranking_test.go`, which
exercises `score` and `Terms` with no filesystem at all; 3 is
`TestSearch_ReportsEvidenceWithoutScoring`.

### Phase 1.1 closed

All four acceptance criteria ticked in plan.md, phase heading flipped to `[x]`.
Criterion 4 is marked passed on the relocation reasoning recorded above: every
pre-existing test passes and no expected value was changed, three of them now
asserting through `Set.Search` instead of a bare `FileStore`. Next unchecked
phase is 1.2 (Rank partial matches instead of discarding them).

### Phase 1.1 changelog written

`## Changelog` section created in plan.md (first invocation) with the Phase 1.1
entry: the two deviations (store still skips zero-evidence documents; three tests
relocated rather than left untouched), the nine changed files, and five
discoveries. 13 unchecked phases remain; next is 1.2.

## User decisions (this session)

- **Autonomous mode, granted at the end of Phase 1.1.** The user chose "run all
  remaining phases" — loop through the plan without asking between phases. Stop
  only for the two escalations the plan itself flags: Open Question 1 (no set of
  weights satisfies the benchmark and the acceptance criteria at once) and Open
  Question 2 (the "volume does not beat relevance" reading), plus Phase 4.4's
  explicit pre-write confirmation gate, which autonomous mode does not waive.
- **Knowledge capture declined** for the test-relocation lesson from Phase 1.1.
  Final for that discovery; do not offer it again this conversation.

## Phase 1.2 analysis

**Hand-computed oracles for every score-dependent assertion.** `damp(n) = 1 + log₂(n)`
for `n > 0`, `0` otherwise; `score = Σ damp(count) × coverage²`. Values marked
*exact* need no `InDelta`, because `log₂` of a power of two is exact in float64:

| damp | value |
| --- | --- |
| `damp(1)` | `1` (exact) |
| `damp(2)` | `2` (exact) |
| `damp(3)` | `2.5849625007211562` |
| `damp(4)` | `3` (exact) |
| `damp(5)` | `3.321928094887362` |
| `damp(8)` | `4` (exact) |

Tests that change, all in `internal/knowledge/set_test.go`:

- `TestSet_SearchScoreSumsOccurrencesAcrossDocument` — query `needle`, one term so
  coverage is 1. Only `spread.txt` moves: `4 → damp(4) = 3`. New oracle
  `{twice:2, once:1, upper:1, spread:3}`, all exact.
- `TestSet_SearchOneHitPerDocument` — `widget gear`, counts 3 and 5, coverage 1 →
  `damp(3)+damp(5) = 5.906890595608518`. Irrational, needs `InDelta`.
- `TestSet_SearchStampsScoreOnHits` — `widget gear`; `tools.md` (3, 2) →
  `damp(3)+damp(2) = 4.5849625007211562`; `partial.md` (0, 1) →
  `1 × (1/2)² = 0.25`. **Now returns two hits, not one** — the partial file stops
  being dropped, which is the phase's whole point. Rewrite, don't just renumber.
- `TestSet_SearchRequiresEveryTerm` — **its premise is what this phase deletes.**
  `alpha bravo charlie`: `scattered.txt` (1,1,1) → 3 at coverage 1;
  `partial.txt` (1,1,0) → `2 × (2/3)² = 0.888…`. Both now return, scattered
  first. Rename to say it ranks partial matches below full ones.
- `TestSet_SearchRanksAcrossSourcesByScore` — order survives
  (`damp(3) = 2.585` still beats `damp(1) = 1`); only the comment's arithmetic
  goes stale.
- `TestSet_SearchTieBreaksBySourceOrderThenPath` — four files, `lantern` once
  each → all score exactly 1. Unaffected.

Tests confirmed green **without** edits, verified by hand rather than by running:

- `cmd/knowledge_test.go:245,253,772,780,788,1160` — every one is a *single-term*
  query (`compass`, `beacon`, `lighthouse`) against a file holding one
  occurrence, so `damp(1) × (1/1)² = 1` exactly. And for a single-term query the
  returned *set* is identical under ranked OR, since a document either has the
  term or scores zero. `damp(1) = 1` is doing real work here.
- `internal/store/search_test.go` — the store neither scores nor excludes after
  Phase 1.1, so nothing in that package is sensitive to the formula.
- `TestKnowledgeSearch_EmptyAndNoMatchQueriesReturnEmptyHits` — an empty query
  yields no terms and a no-match query yields no evidence; both still empty.

`TestScore_FromEvidence` in `ranking_test.go` also needs recomputing: its two
"any zero count → 0" cases are exactly the rule this phase removes.

### Phase 1.2 implementation notes

`internal/knowledge/ranking.go`: added `damp(n int) float64`, rewrote `score` to
count covered terms, sum `damp(count)` rather than raw counts, return 0 only when
nothing is covered, and scale by `coverage^coverageExponent`. `math` imported.
`coverageExponent` is now consumed; `tagWeight` stays unused until Phase 2.2.
`internal/knowledge/set.go`: `Set.Search` doc comment now states matching is a
ranked OR and warns that an empty result is not proof of absence — the point
`decisions/ranked-or-over-boolean-and.md` already records. `go build ./...` clean.

`damp(1) = 1` is commented at the function as a property to preserve, not an
accident, because six `cmd` assertions and the exactness of several oracles
depend on it.

**Oracles independently re-derived.** The Phase 1.2 expected values were computed
twice, by hand and then again with a separate calculator, before any test ran:
`damp(3) = 2.584962500721156`, `damp(5) = 3.321928094887362`,
`damp(20) = 5.321928094887363`; scores `{3,5} → 5.906890595608518`,
`{3,2} → 4.584962500721156`, `{0,1} → 0.25`, `{1,1,1} → 3`,
`{1,1,0} → 0.8888888888888888`, `{9,0,4} → 3.1866333339743607`,
`{0,7} → 0.951838730514401`. Neither derivation involved running the code under
test. `damp` of a power of two is exact in float64, which is why several oracles
can stay `require.Equal` rather than `InDelta`.

### Phase 1.2 tests and verification

All four criteria covered; `make lint` and an uncached `go test -race -count=1 ./...`
both exit 0. Every hand-computed oracle agreed with the implementation on first
run — no number was adjusted to fit.

Tests changed: `TestSet_SearchScoreSumsOccurrencesAcrossDocument` (`spread.txt`
4 → 3, still `require.Equal` since a single-term query at coverage 1 keeps every
value exact); `TestSet_SearchOneHitPerDocument` (8 → 5.906890595608518, now
`InDelta`); `TestSet_SearchStampsScoreOnHits` (now two hits — the partial file
stops being dropped); `TestSet_SearchRequiresEveryTerm` **renamed** to
`TestSet_SearchRanksPartialMatchesBelowFullOnes`, since its premise is exactly
what this phase deletes; `TestSet_SearchRanksAcrossSourcesByScore` comment-only.
Added `TestDamp_DiminishingReturns` and
`TestSet_SearchReturnsAnEntryMatchingOnlyOneOfThreeTerms`.

The six `cmd/knowledge_test.go` `Score: 1` assertions stayed green untouched, as
computed in advance. `damp(1) = 1` is now load-bearing in production.

**Note for anyone reading editor diagnostics during this work:** the LSP reported
`undefined: damp` in `ranking_test.go` while `go build`, `go vet` and `go test`
were all clean. It was a stale snapshot taken mid-edit. Trust the toolchain, not
the diagnostics, when they disagree.

### Phase 1.2 closed — plus a correction that matters for 1.3

Checkboxes ticked, changelog entry written. 12 phases remain; next is 1.3.

**Corrected before it caused trouble.** The changelog first claimed the
two-of-three fixture in `TestSet_SearchRanksPartialMatchesBelowFullOnes` would be
dropped by Phase 1.3's cutoff. It will not: `2 × (2/3)² = 0.889` sits *above*
`0.25 × 3 = 0.75`. The fixture the cutoff actually bites on is the one-of-three
case, `1 × (1/3)² = 0.111`. So when implementing 1.3:

- `TestSet_SearchRanksPartialMatchesBelowFullOnes` must **keep returning both**
  hits — it is a regression guard that the cutoff is not set too aggressively.
- `TestSet_SearchReturnsAnEntryMatchingOnlyOneOfThreeTerms` is the cutoff fixture:
  alone it survives (a lone hit is its own best score), and it must disappear once
  a three-of-three entry is present.
- `TestSet_SearchStampsScoreOnHits` has `0.25` against a best of `4.585`; the
  cutoff is `1.146`, so **that second hit will be dropped** and that test needs
  updating in 1.3. Caught by arithmetic, not by waiting for a red test.

## Phase 1.3 analysis

Single site: the post-sort loop in `internal/knowledge/set.go` that already
stamps tier/name/category and excludes always-applied entries. The slice is
sorted score-descending before it, so the best score is the first *surviving*
element's.

**Ordering is the one real decision, and the plan flags it.** The best score must
come from hits that survive the category exclusion, not from `merged[0]`. An
always-applied entry scoring highest would otherwise set the bar and then be
dropped itself, silently raising the threshold for everything else. That forces
two passes over the sorted slice: exclude and stamp first, then cut. Cheap, and
the shape makes the dependency obvious.

`cutoffFraction = 0.25` goes in `ranking.go` beside `tagWeight` and
`coverageExponent`, not in `set.go`. The plan's Phase 4.1/4.2/4.4 all commit to
`internal/knowledge/ranking.go` being the single source of truth for every
published constant, `CUTOFF_FRACTION` included, so splitting it across two files
would break that promise on the day it is made.

Fixtures already in the suite, with the cutoff arithmetic pre-checked:
`TestSet_SearchRanksPartialMatchesBelowFullOnes` (0.889 vs best 3, cutoff 0.75 →
**survives**, no edit), `TestSet_SearchReturnsAnEntryMatchingOnlyOneOfThreeTerms`
(0.111 alone → survives as its own best), `TestSet_SearchStampsScoreOnHits`
(0.25 vs best 4.585, cutoff 1.146 → **the second hit is now cut**, needs
updating).

### Phase 1.3 implementation notes

`cutoffFraction = 0.25` added to `ranking.go` beside the other tunables.
`Set.Search`'s post-sort loop split into two passes: build `eligible` (stamped,
category-excluded), then cut everything below `eligible[0].Score * cutoffFraction`.
Filter-in-place via `eligible[:0]`, matching the idiom already used in
`internal/store/ignore.go`. Doc comment records that the cutoff is post-merge on
purpose.

Running the suite produced **exactly one** failure, the one predicted by
arithmetic before the code was written: `TestSet_SearchStampsScoreOnHits`, whose
weak hit at 0.25 falls below the 1.146 floor. Nothing else moved. Predicting the
blast radius from the numbers first, then confirming it, is a cheaper check than
reading a red suite and guessing which failures were intended.

### Phase 1.3 closed — Milestone 1 complete

Cutoff landed, all four criteria ticked, changelog written. `make lint` and
`go test -race -count=1 ./...` both 0. 11 phases remain; next is 2.1 (Read tags
from an entry), which starts Milestone 2.

**Constant-tuning envelope now known, for Open Question 1.** `cutoffFraction`
has a hard ceiling around **0.296**: above it,
`TestSet_SearchRanksPartialMatchesBelowFullOnes` breaks, because a two-of-three
match scores `0.889` against a full match's `3`. Phase 2.4's benchmark will
constrain from the other side. If tuning ever needs to cross 0.296, that is the
signal the plan means by "the scoring *shape* is wrong rather than its
constants" — escalate rather than move the guard.

## Phase 2.1 analysis

**Reconciling two statements the plan makes about malformed blocks.** The
Component Breakdown says a malformed block "is reported rather than silently
swallowed"; the recorded decision in research.md
(§ "An unterminated leading block is 'no frontmatter', not an error") says the
reader returns no tags and the raw bytes, *not* an error, because erroring would
make an entry unsearchable over a formatting accident. The declared contract
keeps an `err` return, which would be dead if nothing ever set it.

Resolution: **the parser reports, the scan degrades.** `parseEntry` returns
`(nil, raw, err)` when a block is present and closed but its YAML will not
unmarshal, so the information exists for Phase 3.3's audit flow to surface.
`scanFile` ignores that error and takes the raw body, so a search can never fail
because of one bad entry. Both statements hold, the `err` return is real, and the
decision's actual requirement — search must not break — is guaranteed. An
*unterminated* opener is not an error at all: it is a horizontal rule, exactly as
the decision says.

**Design: parse from the sniff buffer, not by buffering lines.** `scanFile`
already reads the first `binarySniffBytes` (8000) into `sniff` for the binary
check, and already feeds those bytes to the hasher before the line scan starts.
Frontmatter is by definition at the very start, so it can be parsed straight out
of `sniff`, and the scanner then built from `MultiReader(bytes.NewReader(body), tee)`
where `body` is the post-block subslice. This beats the alternatives the plan
offered:

- No line buffering and no replay, so a file opening with `---` and never closing
  cannot accumulate in memory.
- No `inFrontmatter` flag threaded through the scanner loop.
- Term counting, title detection and excerpt collection all begin at the body for
  free, which is criterion 4 satisfied structurally rather than by extra checks.
- The checksum is untouched: the hasher still receives all of `sniff` and the
  whole tee, so it covers the file's exact raw bytes including the frontmatter,
  which is what keeps two entries differing only in tags distinct under
  exact-byte de-dup.

Accepted limit, worth documenting in the code: a frontmatter block extending past
8000 bytes reads as no frontmatter. That degrades to today's behaviour rather
than failing, and no plausible tag block comes close.

A file with no leading fence costs one `bytes.HasPrefix` over four bytes, which
is the "must gain no cost" requirement met.

### Phase 2.1 implementation notes

New `internal/store/frontmatter.go` with `parseEntry` and `normaliseTags`;
`fileAggregate` gained `tags`; `scanFile` parses the block out of the existing
sniff buffer and builds its scanner from the post-block subslice. Full suite
still green (nothing tested tags yet, so that only proves no regression).

Smoke-checked the wiring before handing off to test authoring, with a throwaway
in-package test since removed. Confirmed: `[Go, HTTP, go]` normalises to
`[go http]`; the body comes back with the block and its blank separator stripped;
an unterminated opener yields no tags, no error and the full raw bytes; a closed
block with broken YAML yields no tags *and* an error, which `scanFile` swallows.
Most usefully, an entry tagged `go` whose body never says "go" was **not**
returned for the query `go` — criterion 4 holding structurally rather than by a
guard. Tags do not retrieve anything until Phase 2.2, which is correct here.

### Phase 2.1 closed

All four criteria ticked, changelog written, `make lint` and
`go test -race -count=1 ./...` both 0. 10 phases remain; next is 2.2 (Let tags
retrieve and rank an entry), the phase that makes tags actually do something.

Carry into 2.2: `parseEntry` returns nil (not `[]string{}`) when an entry has no
tags, and Phase 2.2 requires `Hit.Tags` to serialise as `"tags": []` rather than
being absent — so the conversion to an empty slice happens where the Hit is
built, not in the parser. Also note the plan's warning that adding `Tags` to the
`knowledgeHit` mirror struct in `cmd/knowledge_test.go:19` makes six existing
literals assert `Tags: nil` against a JSON `[]`, which produces confusing
nil-vs-empty diffs.

## Phase 2.2 analysis

**My Phase 1.1 deviation becomes wrong here, and must be extended.** In 1.1 I had
the store skip documents with zero body evidence for every term, on the reasoning
that such a document is not a match under any ranking rule. That held through
2.1. It breaks the moment tags retrieve: an entry tagged `go, http` whose body
contains neither word has zero body evidence, so the store would never report it
and the knowledge layer would never get the chance to score its tags. That is
precisely criterion 1 of this phase.

Fix: a document is a candidate if it has body evidence for any term **or declares
any tags at all**. This keeps ranking out of the store — "this document carries
evidence I cannot evaluate" is not a judgement about relevance, it is a refusal to
make one. The knowledge layer scores a tagged-but-irrelevant entry at 0 and drops
it before the merge, so the results are identical to reporting every file; the
guard just stops the store materialising the whole tree as hits.

Cost accepted: every tagged entry is reported for every query. Knowledge stores
hold tens to low hundreds of entries and search already walks the tree reading
every byte of every file, so this is not a new order of cost. Alternatives were
worse: a tag-relevance test inside the store would duplicate the affinity rule
across the interface, which is the exact drift Phase 1.1 exists to prevent.

**Arithmetic checks on the plan's own worked example**, re-derived rather than
trusted. Query `http routing`, `TAG_WEIGHT = 8`:

- Entry A tagged `http, routing`, "routing" twice in body: `http` → `8×1 + 0 = 8`;
  `routing` → `8×1 + damp(2) = 10`; coverage 2/2 → **18**.
- Entry B untagged, "http" 8 times: `http` → `0 + damp(8) = 4`; `routing` → 0;
  coverage 1/2 → `4 × 0.25` = **1**.
- Cutoff `0.25 × 18 = 4.5`, and `1 < 4.5`, so B is dropped.

The published example in the plan is correct as written.

**`tagAffinity` spot-checks against the phase's own criteria**: `http`/`https`
= 4/5 = 0.8; `apple`/`apples` = 5/6; `https`/`https-security` = 5/14;
`test`/`rest` = 0 (not prefix-related); `cors`/`core` = 0; `go`/`golang` = 0
(shorter is 2, below `minPrefixLen = 4`); `test`/`latest` = 0 (suffix, not
prefix — the start anchor). Lengths counted in runes, not bytes, so a non-ASCII
tag cannot get a distorted ratio.

**Coverage now means "any evidence", tag or body.** A term with a tag match but
no body occurrence counts as covered, otherwise a tagged entry would be penalised
for the tag having done its job.

CLI fallout confirmed by reading: `knowledgeSearchOutputSchema`
(`cmd/knowledge.go:114`) needs a `tags` property, and the `knowledgeHit` mirror
struct (`cmd/knowledge_test.go:19`) needs a `Tags []string` field — which makes
its six existing literals assert `Tags: nil` against a JSON `[]`, so each needs
`Tags: []string{}`.

### Phase 2.2 implementation notes

`Hit.Tags` added (no `omitempty`, built as `[]string{}` when absent); the store's
candidacy guard now counts declared tags as evidence; `tagAffinity` and
`minPrefixLen = 4` added to `ranking.go`; `score` computes
`tagWeight*affinity + damp(count)` per term with coverage counting a term as
covered if it has *either* kind of evidence; `Set.Search` builds the affinity
slice; `knowledgeSearchOutputSchema` advertises `tags`.

Rune counts, not byte lengths, decide both the `minPrefixLen` guard and the
credit ratio, so a non-ASCII tag cannot get a distorted score.

Exactly two tests fail, both predicted breakages of this phase:

1. `TestScore_FromEvidence/tag_affinity_does_not_yet_affect_the_score` — its name
   states the old contract. Now `{BodyCounts: {2}, TagAffinity: {1}}` scores 10,
   not 2.
2. `TestSearch_TagsAreNotCountedAsBodyMentions` — the store now reports the tagged
   entry as a candidate, so the hit count moved from 1 to 2. Criterion 4 still
   holds and is now *more* directly visible: the tagged entry comes back with
   `BodyCounts [0]`. The assertion should move from "not returned" to "returned
   with zero body evidence", which is the stronger statement.

**Cutoff interaction worth knowing when writing this phase's tests.** A tagged
entry scores 8 for a one-term query, so the floor is 2, and a plain prose mention
of the same term scores `damp(1) = 1` — below it. So "the tagged entry ranks
above the prose mention" often shows up as *the prose mention not being returned
at all*. That is correct behaviour, not a broken fixture, but a test asserting
two hits in order will fail unless the prose entry mentions the term enough times
to clear 2 (`damp(n) > 2` needs n ≥ 3).

### Phase 2.2 closed

All ten criteria ticked, changelog written, lint and race suite both 0. Every
hand-computed oracle matched the implementation first time. 9 phases remain; next
is 2.3 (Narrow a search to tagged entries).

Milestone 2's core is now live: tags retrieve an entry on their own, and outrank
prose mentions of the same word.

## Phase 2.3 analysis

**The tag filter is exact membership, not prefix-related.** Prefix partial credit
is a *ranking* rule; `--tag` is a *filter*, and the plan is explicit that "an
entry without that tag is never returned, however well it would otherwise score".
So an entry tagged `https` is not returned for `--tag http`, even though it would
score against the query `http`. Flag values are lower-cased on the way in, since
tags are normalised on read and `--tag HTTP` should obviously find `http`.
Repeated `--tag` is AND (narrowing), unlike `--filter`, which is OR (alternative
stores) — a deliberate asymmetry the docs phase has to state.

**Refusing an unknown tag is deferred to Phase 3.1, as the plan permits.** The
error would need `WithNextAction("knowledge tags")`, and that command does not
exist until 3.1. Until then an unmatched `--tag` returns an empty result rather
than a refusal. Recorded so 3.1 picks it up rather than losing it.

**`knowledgeNarrowingFlags` needs splitting.** That map is the `--schema` output
for all four fan-out commands (`cmd/knowledge.go:109`, consumed at `:321`, `:369`,
`:423`, `:450`). `--tag` is registered on search alone, so adding it to the shared
map would advertise a flag the other three silently ignore. Search gets its own
map that embeds the shared pair plus `tag`.

**Enforcement is the knowledge layer's, the store's copy is an optimisation.**
`Set.Search`'s post-merge loop is the authority, using `hit.Tags`; the store may
skip non-matching documents during the walk to avoid reporting them, but nothing
depends on it doing so. That is what stops a remote provider having to be trusted
on the filter. The load-bearing test is a fake store that ignores `opts.Tags`
entirely and still has its hits excluded.

`Selector.covers` (`internal/knowledge/address.go:115`) must **not** learn about
tags: it is a store-membership test, while tags narrow *entries* within a store.
The two axes travel on one value but are applied at different points.

### Phase 2.3 implementation notes

`SearchOptions.Tags` added; `CarriesEveryTag` exported from `internal/store` so
the filter means exactly one thing on both sides of the interface — the store
applies it as a walk fast path, the knowledge layer enforces it after the merge.
`Selector.Tags` added with a comment on `covers` saying why it must stay
tag-blind. `--tag` registered on `knowledgeSearchCmd` alone as a repeatable
`StringArrayVar`, lower-cased in `knowledgeSelector()`.

`knowledgeNarrowingFlags` was **not** extended. Search got its own
`knowledgeSearchFlags`, built by copying the shared map and adding `tag`, so the
other three fan-out commands do not advertise a flag they ignore.

Smoke-tested against this repo's real knowledge base before handing off:
`knowledge search --schema` now lists flags `[filter tag tier]` and hit fields
including `tags`; `--tag nonexistent-tag` returns `"hits": []`; and a plain
search for "ranking" returns `decisions/ranked-or-over-boolean-and.md` carrying
`"tags": []`. Full suite green with no edits needed, which is expected — nothing
existing asserted on a flag that did not exist.

### Phase 2.3 closed — Milestone 2 nearly done

All four criteria ticked, changelog written, lint and race suite 0. The agent
mutation-checked four separate breakages (knowledge-layer filter removed, ignore
decorator dropping opts, search publishing the shared flag map, store fast path
removed) and confirmed each new test fails, so the tests are load-bearing rather
than merely green. 8 phases remain; next is 2.4 (the observed-query benchmark).

**Tooling gotcha hit while writing the changelog, worth avoiding again.** In a
quoted heredoc (`<<'PY'`) bash passes the body verbatim, so Python sees `'\\n'`
as a literal backslash-plus-n, not a newline. Joining the changelog entry with
that inserted a stray `\n` line into plan.md. Caught by inspecting the join with
`cat -A` rather than trusting the write. Use a real newline in the Python string,
and verify the seam after any generated append.

Note also that the document legitimately contains the characters `\n` inside a
Phase 2.1 discovery about close detection, so a blanket "no backslash-n anywhere"
assertion is wrong — match the stray line specifically.

## Phase 2.4 analysis — the benchmark passes, Open Question 1 does not need escalating

Pre-verified with a scratch fixture (since removed) before writing the real test,
because this phase is where Open Question 1 ("do the weights still clear the
benchmark?") is answered. **4 of 4 with the constants exactly as shipped** —
`tagWeight = 8`, `coverageExponent = 2`, `damp(n) = 1 + log₂(n)`,
`cutoffFraction = 0.25`. No tuning required, so nothing to escalate.

| Query | 1st | score | runner-up |
| --- | --- | --- | --- |
| `http endpoints go` | `architecture/http-routing.md` | 25.0 | `initial-idea.md` 6.585 |
| `http handler` | `architecture/http-routing.md` | 17.0 | (only hit) |
| `http endpoints handler routing` | `architecture/http-routing.md` | 34.0 | (only hit) |
| `api` | `architecture/http-routing.md` | 8.0 | `initial-idea.md` 3.0 |

The `api` row is the success metric "the `api` case reverses" in one line: the
routing entry wins on its tag alone, its body never containing the string "api",
against a document with four real mentions of it. Under the old design that query
returned nothing for the routing entry at all.

Ranks match the research prototype (which measured 28.58 / 18.58 / 36.58 / 8.00);
the absolute scores differ only because the fixture's prose is not byte-identical
to the corpus the prototype ran over. Ranks are what the metric is about.

**This phase is test-only** — there is no production change. The `implement` step
is a deliberate no-op.

Fixture must stay self-contained in `t.TempDir()`: research showed that adding
the real `decisions/ranked-or-over-boolean-and.md` (which quotes all four
benchmark queries verbatim) to the live corpus moves the target from rank 1 to
rank 2 on `http handler`. A test bound to the growing real directory would flap.

### Phase 2.4 closed — Milestone 2 complete, both Open Questions answered

Benchmark holds **4 of 4** with every constant exactly as proposed; nothing was
tuned at any point in Milestones 1 or 2. 7 phases remain; next is 3.1 (Report the
tag vocabulary already in use), which starts Milestone 3.

**Open Question 1 closes without escalation** — no tuning was ever needed, so the
"no set of values satisfies everything" condition never arose.

**Open Question 2 closes in practice.** "About that term" did mean "tagged with
that term": every acceptance criterion across Milestone 2 was satisfied by
pairing a tagged entry against an untagged one, and none ever required the
comparison to hold between two *untagged* entries — which is the only reading
that would have needed length normalisation and a spec amendment. The plan called
this "the single most likely thing to be wrong"; it was not wrong.

### Phase 3.1 groundwork (already read, no need to re-derive)

- `Set.List` (`internal/knowledge/set.go:318`) is the shape to follow: validate
  the selector, loop sources, skip those `sel.covers` rejects, `listFiles(src.store, "")`.
- `listFiles` (`:503`) recurses the tree and returns store-relative paths.
- `alwaysAppliedSet()` (`:274`) is the exclusion set; `categoryOf` (`:265`) maps a
  path to its category.
- `parseEntry` in `internal/store/frontmatter.go` must be **exported** as
  `ParseEntry` so the knowledge layer can extract tags per entry.
- 3.1 also owns the deferred work from 2.3: refusing a `--tag` naming a tag no
  entry carries, built with
  `output.NewError(...).WithNextAction("... knowledge tags ...")` now that the
  command exists to name.

## Phase 3.1 analysis

**Decision: refuse an unknown `--tag`, resolving the deferral from Phase 2.3.**
The plan left this conditional ("if a `--tag` naming a tag no entry carries
should be refused rather than silently returning nothing"). It should, for one
concrete reason: the sibling narrowing flag already does. `validateSelector`
(`internal/knowledge/set.go:449`) refuses a `--filter` naming a store the tier
does not reach, precisely so "a caller is never left reading an empty result as
'no such knowledge'". A typo'd `--tag` is the same failure, and the repo's
error-messages convention is explicit that a refusal must name a runnable next
step — which `knowledge tags` now is.

**Cost is kept off the common path.** Validating a tag needs the vocabulary,
which is O(entries) of I/O — as much again as the search itself. So the check
runs only when a tag-narrowed search returns **nothing**: on the happy path it
costs zero, and on the empty path the user gets a refusal naming the available
tags instead of an ambiguous empty list. A tag that *is* in the vocabulary but
whose entries did not match the query still returns empty, correctly — the tag
exists, the query simply did not hit.

New error code `ErrCodeTagUnknown = "knowledge_tag_unknown"`, alongside the five
existing ones, with a next action mirroring `availableNames`' shape.

**This breaks a Phase 2.3 test by design**:
`TestKnowledgeSearch_UnknownTagReturnsEmptyHitsNotAnError` pins the interim
behaviour and carries a comment saying 3.1 is where a refusal would go. It must
be inverted here, not deleted — the case still matters, the expected outcome
changes.

`parseEntry` gets exported as `ParseEntry` so the knowledge layer can read tags
per entry, per the plan's preference for exporting the parser over adding a
store-level `Tags()` helper — it keeps the walk in the knowledge layer where
`listFiles` already lives.

Always-applied categories are excluded from the vocabulary, consistent with
search: those entries are loaded in full on every task and never surfaced as
hits, so their tags are not part of the vocabulary a capture flow chooses from.

### Phase 3.1 implementation notes

`ParseEntry` exported; `TagUse` and `Set.Tags` added to `internal/knowledge/set.go`
(count-desc then alpha, always-applied categories excluded, a malformed block
yielding no tags rather than failing the listing); `ErrCodeTagUnknown` added;
`refuseUnknownTags` wired into `Set.Search`'s empty-result path only;
`knowledgeTagsCmd` registered with its own output schema and the shared
`--tier`/`--filter` loop.

Smoke-tested against this repo's real store before handing off:
`knowledge tags` → `{"tags": []}` (nothing is tagged yet, correctly);
`knowledge tags --schema` publishes `tier`/`filter` and the
`{tag, count}` output shape; and
`knowledge search "ranking" --tag nonexistent` now returns
`knowledge_tag_unknown` with next action
`list the vocabulary with "knowledge tags" and reissue with a tag that is in use
(no entry in scope carries any tag yet)` — a refusal that names a runnable
command, per the repo convention.

Exactly one test fails, as predicted:
`TestKnowledgeSearch_UnknownTagReturnsEmptyHitsNotAnError`, the Phase 2.3
placeholder whose own comment says 3.1 is where the refusal belongs. It must be
**inverted**, not deleted — the case still matters, only its expected outcome
changed.

### Phase 3.1 closed

All four criteria ticked, changelog written, lint 0 / race 0 / zero FAIL.
6 phases remain; next is 3.2 (Propose tags when an entry is captured), which is
the first **prose** phase — templates, not Go.

**Phase 3.2/3.3 are a different kind of work and carry a different failure
mode.** They edit `templates/`, and the rendered copies under `.claude/` and
`.bob/` are git-tracked. Nothing fails until an agent reads a stale copy, so
every template edit must be followed by `go run . init claude && go run . init bob`
with the regenerated files committed. The guard tests
(`internal/agent/instruction_surface_test.go`, `knowledge_trigger_test.go`) are
what catch a missed regeneration — and they render through the production install
path into a temp dir rather than reading the committed copy, so a stale committed
file cannot make them pass.

Files 3.2 touches: `templates/skills/workflows/spek-knowledge/SKILL.md` (the
source of truth, 83 lines, intents at `:27`, `:48`, `:64`, `:79`),
`templates/agents/knowledge-trigger.md`, `templates/steps/plan/02-discovery.md`,
`templates/skills/skill_spawn-planning-agents.md`, plus
`internal/agent/instruction_surface_test.go`'s hand-maintained
`expectedCRUDInvocations` slice at `:93`.

Per the repo's own convention (recalled from memory), SKILL.md files write the
CLI as `{{command}}`, never the rendered `go run .`.

## Phase 3.2 analysis

Five template files plus one guard test, then regeneration. Doing the prose edits
in the main context rather than delegating: this is the agent-facing contract, the
edits are small and surgical, and the wording is the deliverable.

**Two placeholder namespaces, do not mix them.** Skill templates
(`templates/skills/**`) and agent sections (`templates/agents/**`) use
`{{command}}`; plan step templates (`templates/steps/plan/**`) use
`{{config.command}}`. `02-discovery.md` is the latter. Substituting the wrong one
renders literally and is invisible until an agent reads it.

**Contribute flow gets one new step, not a second approval gate.** Tag selection
slots in after category routing — both are "classify this entry" judgements — and
the proposed tags are shown inside the block the user already confirms. The
number of approval gates stays at one, which the plan is explicit about.

**Where the tag-form rules must not drift.** Phase 3.3's audit intent has to
inherit these same rules, and the plan says to *reference* them rather than
restate them, so the two intents cannot diverge. Writing them once under
contribute, with a named heading the audit can point at, is what makes that
possible.

Acceptance criterion 7 ("searching is described to agents in a way that reflects
how results are now ranked") spans three files, not one: SKILL.md's lookup step,
`templates/steps/plan/02-discovery.md`, and
`templates/skills/skill_spawn-planning-agents.md`. All three currently describe
retrieval to a driving agent, and one of them states the boolean-AND rule
outright.

### Phase 3.2 implementation notes

Edited five templates: `spek-knowledge/SKILL.md` (lookup gains `tags` and the
ranked-OR description plus `--tag`; contribute gains `knowledge tags` in step 1,
a new step 3 carrying the three tag-form rules, a frontmatter-opening staged
body, and tags inside the single existing confirmation gate; update preserves the
block), `templates/agents/knowledge-trigger.md`,
`templates/steps/plan/02-discovery.md`, and
`templates/skills/skill_spawn-planning-agents.md`. Extended
`expectedCRUDInvocations` with `knowledge tags`. Regenerated with
`init claude` + `init bob`.

Verified after regeneration: no `{{command}}` or `{{config.command}}` placeholder
leaked into any rendered copy, `knowledge tags` is present in both rendered
SKILL.md files, and **no agent-facing prose anywhere still claims a document must
contain every query word** (grep over `templates/`, `.claude/`, `.bob/`,
`AGENTS.md` returns nothing). Full suite green.

Caught one self-inflicted slip before it shipped: the
`skill_spawn-planning-agents.md` bullet initially ended mid-sentence
("...before concluding it is"). Prose edits done by string replacement need
reading back, not just asserting the replacement applied.

## Pre-existing repo problem found (unrelated to this plan)

**`AGENTS.md` has been stale since commit `a76e04c`.** That commit ("Ensure
knowledge base takes precidence over patterns in code base") edited
`templates/agents/historical-artifacts.md` but never re-ran init, so the
committed `AGENTS.md` never received the paragraph declaring knowledge entries
binding over the code. Confirmed: `git show HEAD:AGENTS.md` does not contain it;
the template does.

Running `init` for this phase regenerated it, so that paragraph now appears in my
diff as an unrelated change. **Keeping it is correct** — the generated file is
supposed to match its template, and this is precisely the "nothing fails until an
agent reads the stale copy" failure the plan's generated-copy discipline warns
about. Flagged to the user rather than buried, because it is not this plan's
change and it means every agent session since `a76e04c` has been reading an
AGENTS.md missing a rule the project intended to be binding.

### Phase 3.2 closed

All seven criteria ticked, changelog written, lint 0 / race 0 / zero FAIL.
Regeneration confirmed idempotent. 5 phases remain; next is 3.3 (Audit the tags
on existing entries), the last prose phase before documentation.

**Seams established in 3.2 that 3.3 reuses**: prose guards render through
`installWorkflowSkills(tmp, ".claude/skills", cfg, io.Discard)` into a
`t.TempDir()` and assert hand-maintained literal substrings; there is now a
`renderSpekKnowledgeSkill` helper in `internal/agent/instruction_surface_test.go`
doing exactly that. The tag-form rules live under contribute step 3 with a bolded
**Choosing tag forms** heading, which is what 3.3's audit intent should reference
rather than restate.

## Phase 3.3 analysis

Adds a fourth intent to a skill whose own preamble says it has three, so three
places need updating together: the "picks one of three branches" sentence at
`:13`, the `# When to invoke` trigger list, and the new `# Intent: audit`
section. Missing the first two would leave the skill telling an agent the audit
does not exist.

**The audit composes only existing primitives** — `knowledge list`,
`knowledge tags`, `knowledge read`, and the existing confirmed
`knowledge write`. No new CLI command, no bulk operation. `knowledge list`
becomes load-bearing prose for the first time, so it joins
`expectedCRUDInvocations` in the guard test.

**The two guards the plan calls for are opposite errors, and the audit is where
both are most likely.** Over-merging: telling an entry tagged `https` to use
`http` because the convergence rule says prefer an existing tag — wrong, they are
different subjects, and prefix matching already relates them at reduced strength.
Under-pruning: leaving `apples` beside `apple` when prefix matching already
reaches it — a redundant tag that should be reported as removable. The audit
must reference the **Choosing tag forms** rules from contribute step 3 rather
than restate them, so the two intents cannot drift apart.

**Per-entry confirmation is the load-bearing contract.** The plan is explicit
that a reviewer should check the audit cannot reach `knowledge write` without
passing the same gate the other intents use, and that a decline on one entry must
not carry to the next or silently apply another's changes. `# Decline handling`
needs extending to say so.

### Phase 3.3 implementation notes

Added `# Intent: audit` (rendered at `:102`), updated the preamble from three
branches to four, added a trigger line to `# When to invoke`, and extended
`# Decline handling` with the per-entry rule. `knowledge list` added to
`expectedCRUDInvocations` since the audit makes it load-bearing prose for the
first time. Regenerated both agent copies; verified all four intents render, no
`{{` placeholder survives, and a second `init` produces no change. Suite green.

The audit references the **Choosing tag forms** rules rather than restating them,
as the plan requires, and names the two auditing-specific failure modes: do not
over-merge (`https` is not `http`), do prune what prefix matching already reaches
(`apples` beside `apple`).

## Milestone 4 groundwork (read ahead, no edits yet)

**Docs repo confirmed present and clean**: `/home/nicj/code/github.com/jumppad-labs/spektacular-website`,
branch `main`, working tree clean. Phases 4.2 and 4.3 land there; 4.1 and 4.4 in
`spektacular`.

**Phase 4.1 targets, located:**

- `docs/knowledge-base.md:78` "## Search behaviour" — opens with "A document
  matches when every query word occurs somewhere in it, in any order", the
  sentence this work makes false. Its result-field bullet list needs `tags`.
  Section runs to `:94`.
- `docs/knowledge-base.md:185` "## Command reference" — the `knowledge search`
  row needs `--tag`; a `knowledge tags` row needs adding.
- `docs/knowledge-base.md:173` "## Contributing knowledge" — the proposal now
  includes tags alongside tier, store name and path.
- `docs/knowledge-base.md:122` "### Why the mechanical layer is exact, not fuzzy"
  — argues against similarity matching for de-dup. The new ranking prose must not
  read as contradicting it; de-dup is exact, ranking is not, and a sentence
  distinguishing them will help.
- Natural home for the frontmatter contract subsection: after the category model
  (`:14-52`) or inside "Search behaviour".
- `README.md:137` — the `knowledge search` bullet; a `knowledge tags` bullet goes
  beside it.

**The three-document duplication hazard the plan flags.** After Milestone 4 the
scoring formula and its constants are stated in three places:
`docs/knowledge-base.md` (4.1), `docs:src/pages/knowledge-base.mdx` (4.2), and
the knowledge entry (4.4). `internal/knowledge/ranking.go` is the single source
of truth and all three must quote it; each document must say so, because tuning a
constant is then a four-file change and a later tuner needs to know that from
whichever document they happen to be reading.

**Constants as shipped, for all three documents**: `tagWeight = 8`,
`coverageExponent = 2`, `minPrefixLen = 4`, `cutoffFraction = 0.25`,
`damp(n) = 1 + log₂(n)`. Verified unchanged since Phase 2.2.

### Phase 3.3 closed — Milestone 3 complete

All six criteria ticked, changelog written, lint 0 / race 0 / zero FAIL,
regeneration in sync. 4 phases remain, all documentation: 4.1 (spektacular's own
docs), 4.2 and 4.3 (**docs repo**), 4.4 (the knowledge entry, which needs an
explicit user confirmation gate that autonomous mode does not waive).

**Near-miss worth carrying forward: never `git checkout --` during this work.**
A sub-agent restoring a mutation-test edit used it on a template carrying
uncommitted Phase 3.x changes and reverted real work. It was recovered from a
byte-exact copy and independently re-verified here (all twelve edits present,
rendered copies matching a fresh render). Every phase so far is uncommitted in
the working tree, so any restore must come from a copy taken beforehand, never
from git. Brief future sub-agents accordingly.

## Phase 4.1 analysis

Prose only, `spektacular` repo, no build step. Doing it in the main context: I
hold the algorithm and every constant, and the risk here is factual drift rather
than effort.

**This document carries the full algorithm, not a summary.** Its stated purpose
is that "a maintainer new to the project can understand it without reading the
source", so it gets tokenization, both kinds of per-term evidence, the whole
`tagAffinity`/`damp`/`termScore`/`coverage`/`score` formula with the shipped
constants including `minPrefixLen`, both prefix guards, and the relative cutoff.
The published site (4.2) gets the same content for a different reader.

**The docs repo's no-em-dashes convention does NOT bind this repo.** The plan is
explicit, and the existing file uses them throughout.

**Worked example re-verified against the shipped constants**, query
`http routing`: Entry A tagged `http, routing` with "routing" twice scores
`(8 + 0) + (8 + damp(2)=2) = 18` at coverage 1; Entry B untagged with "http" eight
times scores `damp(8) = 4` at coverage 1/2, so `4 × 0.25 = 1`; the floor is
`0.25 × 18 = 4.5`, so B is cut. The same example goes into 4.2 verbatim so the
two documents cannot disagree.

**Must not read as contradicting "Why the mechanical layer is exact, not fuzzy"**
(`:122`). That section argues de-duplication is exact rather than similarity-based.
Ranking is now emphatically not exact, so the two need a sentence distinguishing
them: de-dup decides whether two entries are the *same*, ranking decides which is
a better *answer*. Different questions, different tools.

**Each of the three documents must name `internal/knowledge/ranking.go` as the
single source of truth for the constants**, because tuning one is a four-file
change and a reader of any single document needs to know that.

### Phase 4.1 implementation notes

`docs/knowledge-base.md` gained three subsections under "Search behaviour" — "How
a result is ranked" (tokenization, both evidence kinds, the full formula as a
fenced block, the worked two-entry example, prefix partial credit with both
guards and the edit-distance rejection, the relative cutoff), "Where each part is
computed" (store reports, knowledge layer ranks, and why), and "Tags on an entry"
(both YAML list forms and the six behavioural rules). The command reference gained
`--tag` on search and a `knowledge tags` row; "Contributing knowledge" gained tag
proposal and the audit intent; `README.md` gained a `knowledge tags` bullet and a
corrected `knowledge search` bullet.

Added a paragraph to "Why the mechanical layer is exact, not fuzzy" distinguishing
the two questions, because the new ranking prose would otherwise read as
contradicting it: de-duplication asks whether two entries are *the same* (exact,
yes-or-no), ranking asks which is a better *answer* (scored, a matter of degree).
The existing argument is about the first only.

**Constants cross-checked against the code, not against the plan**: `tagWeight = 8`,
`coverageExponent = 2`, `minPrefixLen = 4`, `cutoffFraction = 0.25`,
`damp(n) = 1 + log2(n)` — all match `internal/knowledge/ranking.go`. The worked
example's arithmetic was recomputed independently: A = 18, B = 1, floor 4.5, B
cut. Both go into 4.2 verbatim so the two documents cannot disagree.

The document now names `internal/knowledge/ranking.go` as the single source of
truth and states that changing a constant is a four-file change. 4.2 and 4.4 must
carry the same sentence.

No stale "every query word" claim remains anywhere in `docs/` or `README.md`.

### Phase 4.1 closed

All three criteria ticked, changelog written. 3 phases remain: **4.2 and 4.3 in
the docs repo**, then 4.4 (the knowledge entry, needing an explicit user
confirmation gate).

### Phase 4.2 implementation notes (docs repo)

Two new `Section` bands in `docs:src/pages/knowledge-base.mdx`: "How a search is
ranked" (`surface`) and "Tagging an entry" (plain), inserted between the
lifecycle section and Configuration. Alternation verified end to end: plain (17)
→ surface (44) → plain (94) → **surface (158) → plain (233)** → surface (318) →
plain (430). Exactly two bands in that order means **nothing below needed
flipping**, as the plan predicted.

Also edited in place: the stale ranking sentence in the lifecycle section, the
"no required frontmatter" line (now points at the optional tags block), the
`knowledge write` block (the skill proposes tags with everything else), the
`--tier`/`--filter` narrowing prose (gains `--tag`, the AND-vs-OR asymmetry, and
that an unknown tag is refused), and the de-duplication section (scoped to the
"are these the same?" question so it does not read as contradicting inexact
ranking).

**Verification, all clean**: `make check` reports 0 errors, 0 warnings (3
pre-existing hints about deprecated `document.execCommand`, untouched);
`npm run build` builds 12 pages; `grep -nE "<div|<section|class=" src/pages/*.mdx`
returns nothing (Rule 1); and `grep -c '—'` on the page returns **0**, satisfying
the no-em-dashes convention that binds this repo but not spektacular.

**Constants cross-checked against the Go source, not the plan**: all four values
match. The page uses pseudo-code names (`TAG_WEIGHT`) in the formula block, so it
now also names the real identifiers and points at
`internal/knowledge/ranking.go` as the single source of truth, which is the
sentence Phase 4.1 already carries and 4.4 still needs.

### Phase 4.2 closed

All 12 criteria ticked, two of them **after correcting stale text**. Criteria 5
and 8 asserted exact tag matching and that a singular does not retrieve a plural,
both superseded at the walkthrough by prefix partial credit, and both contradicted
by this phase's own Content outline in the same document. Corrected in place with
an inline note rather than ticked as written, because ticking them would have
recorded the page as documenting something false.

**Pattern worth watching for in the remaining phases**: this plan's acceptance
criteria were written before the walkthrough changed the tag-matching design, so
any criterion mentioning "exact" tag matching is suspect. Phase 4.3's criteria
were checked and are unaffected (they concern the backend contract, not matching).

2 phases remain: 4.3 (docs repo, storage-backend reference) and 4.4 (the
knowledge entry, which needs an explicit user confirmation gate).

### Phase 4.3 implementation notes (docs repo)

`docs:src/pages/extending.mdx`: `Search` signature updated to
`(terms []string, opts SearchOptions)`, `SearchOptions` added beside `DirEntry`,
the `Hit` struct replaced with the regrouped correct shape, and four method
contract bullets added covering why a store leaves `Score` at zero, that matching
is a ranked OR, the per-term-count limit that excludes purely semantic backends,
and that `SearchOptions.Tags` is an optimisation rather than a guarantee.

**Pre-existing drift corrected in the same pass**, as the plan directed. The
published `Hit` had been wrong independently of this work: it showed
`Excerpt string` where the code has `Excerpts []string`, and omitted `Title`,
`Category` and `Checksum` entirely. Verified programmatically after editing: the
published field list now matches `internal/store/store.go` field-for-field in
order, and the published `Search` signature matches the source exactly.

`make check` 0 errors 0 warnings, `npm run build` 12 pages, zero em dashes, Rule 1
guard clean.

**Worth raising with the user**: nothing ties this page to the Go source, which
is how it drifted. The plan flags this as out of scope but worth a follow-up.

### Phase 4.4 closed — all 14 phases complete

Knowledge entry written after explicit user confirmation. End-to-end verification
on live data: the entry ranks first for its own subject (32.977 vs 8.322), is
retrieved by `retrieval` at exactly 8.0 on its tags alone (that word appears only
in its frontmatter, verified against the body), `--tag ranking` narrows to it, and
`--tag bogus` is refused naming the real vocabulary.

**Open follow-up found at the very end**: `tag` and `tags` do not find each other,
because the shorter is 3 runes and `minPrefixLen = 4` rejects it. The tag-form
rule tells agents not to carry both a singular and a plural on the grounds that
prefix matching relates them, and that reasoning fails for any singular under four
characters (`tag`, `api`, `job`, `key`). Not a bug in the code, a gap in the
guidance. Worth raising with the user.

### Test plan written

`000050_knowledge-entry-tags/test-plan.md` covers the two success metrics the
plan classified as manual (newly captured entries carry tags; the vocabulary
converges rather than fragments), each with a pass condition stated as a number,
the real commands to gather it, and who runs it when. Metrics 1 and 2 are
explicitly delegated to `internal/knowledge/benchmark_test.go` rather than
restated. The `tag`/`tags` sub-four-character gap is recorded there as a caveat
for whoever runs the convergence check, so it is not mistaken for fragmentation.

### Feature changelog records written

Three records: project-level (`000050_knowledge-entry-tags.md`), plus one each for
the `spektacular` and `docs` repos, both stamped with project/spec/plan
provenance in their front matter and carrying the readable reference line in the
body. Each repo record opens with a user-facing summary before any file path.

The project record carries the five deviations, the two things fixed in passing
(the stale published `Hit`, the stale `AGENTS.md`), and three known follow-ups.

### Spec reconciled

All 28 checkboxes (13 requirements + 15 acceptance criteria) flipped to `[x]`,
each judged against the plan's phase changelog rather than blanket-flipped. Zero
remain unchecked; nothing was dropped, deferred or descoped.

Requirements 9, 10, 12 and 13 and acceptance criteria 9, 10, 12, 13 and 14 are
delivered as **prose contracts** in the `spek-knowledge` skill rather than as
code, and are asserted by rendered-prose guards that render through the
production install path. That is the delivery mechanism the plan specified for
them, so they are genuinely satisfied, but a reader should know the enforcement
is prose plus a guard test, not a CLI check.
