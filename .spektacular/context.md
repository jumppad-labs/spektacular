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

