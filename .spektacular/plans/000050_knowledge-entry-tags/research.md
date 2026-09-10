---
created_date: "2026-09-10"
status: completed
closed_date: "2026-09-10"
---

# Research: 000050_knowledge-entry-tags

**Repos in scope.**

Two registered repos, both touched:

| Repo | Root | Why it is in scope |
| --- | --- | --- |
| `spektacular` | `/home/nicj/code/github.com/jumppad-labs/spektacular` | Owns the search engine, the knowledge layer, the CLI surface, and the `spek-knowledge` skill template. Requirements 1-10, 12, 13. |
| `docs` | `/home/nicj/code/github.com/jumppad-labs/spektacular-website` | Owns the published user documentation at spektacular.dev. Requirement 11 only. |

Attribution per requirement is carried in the plan's context document.

---

## Alternatives considered and rejected

### Where tags are declared on an entry

- **Reuse `internal/metadata` for the tag frontmatter.** Rejected.
  `metadata.Metadata.UnmarshalYAML` (`spektacular:internal/metadata/metadata.go:83-108`)
  hard-requires a parseable `created_date` and a `status` drawn from a
  four-value enum, and `metadata.Render` (`:59-73`) re-marshals through a fixed
  `yamlShape` (`:55-63`) that silently drops unknown keys. An entry carrying only
  `tags:` would be rejected as malformed frontmatter, and any round-trip would
  delete the tags. That package owns the *workflow artifact* schema (spec, plan,
  changelog), which is a different schema with different mandatory fields.
- **Add a `Tags` field to `metadata.Metadata`.** Rejected. It puts a
  knowledge-only field on every spec, plan and changelog artifact, and leaves
  `created_date`/`status` mandatory on knowledge entries, which the spec's
  no-migration constraint forbids.
- **A sidecar tag index file per store.** Rejected: a second file to keep in sync
  with the entry, invisible in a diff of the entry itself, and it breaks the
  constraint that an untagged base needs no migration and no command to run.
- **Chosen: a leading YAML frontmatter block parsed by a small dedicated reader.**
  A block-less entry is not an error, exactly as
  `spektacular:internal/metadata/frontmatter.go:16-21` already establishes for
  artifacts (`bytes.HasPrefix(raw, "---\n")` false → return the raw body).

### Where the score is computed

> **Corrected during the plan walkthrough.** An earlier draft of this plan put
> per-document scoring inside the store and rejected scoring in the knowledge layer on
> the grounds that it "would need a second read of every candidate file". That
> reasoning was wrong: it conflated *where the score is computed* with *where the file
> is read*. A store can report the evidence from the same single pass it already
> performs, and the knowledge layer can score that evidence without touching the
> filesystem at all. The rejection below is the corrected one.

- **Score inside each store (per-provider scoring).** Rejected. `Store` is an
  interface (`spektacular:internal/store/store.go:38-57`) with one implementation
  today and more planned, and the project's published documentation already lists
  further knowledge providers as forthcoming. Scoring per provider makes the ranking
  formula a contract every future implementation must reproduce exactly. Any drift in
  the tag weight, damping curve or coverage exponent yields scores that are silently
  incomparable in the cross-store merge at
  `spektacular:internal/knowledge/set.go:167-176`, and the best-hit-relative cutoff
  would then be comparing one provider's scale against another's. Nothing fails
  loudly; results are simply quietly wrong. The spec's requirements ("a tag match
  outweighs a body occurrence", "a weak match appears alone and disappears in
  company") are statements about the whole result set, so they can only be guaranteed
  where the whole result set exists.
- **Compute the whole score, including the weak-match cutoff, in
  `FileStore.Search`.** Rejected for the same reason, and additionally because the
  cutoff is relative to the strongest hit and the strongest hit may live in a
  *different* store. `spektacular:internal/knowledge/set.go:140-190` is the only place
  hits from every covered store are merged, so a per-store cutoff would keep a weak
  hit alive purely because its own store held nothing stronger.
- **Chosen: stores report evidence, the knowledge layer ranks.** A store returns, per
  candidate document, the per-term body occurrence counts and the entry's tags, plus
  the locator, title, excerpts and checksum. The knowledge layer tokenizes the query,
  scores every reported hit through one function, merges, sorts and applies the
  cutoff. This costs no extra I/O: `spektacular:internal/store/search.go:135-215`
  (`scanFile`) still reads each file exactly once and already collects exactly this
  evidence; it hands it up rather than collapsing it to a number first.
  The precedent is already in the type: `Hit.Tier`, `Hit.Name` and `Hit.Category` are
  documented at `spektacular:internal/store/store.go:25-36` as "left empty by the
  store and populated by the knowledge layer", because a store has no notion of its
  caller's addressing scheme. It equally has no notion of the stores it will be ranked
  against, so `Score` joins that group.
  **Known limit accepted:** a provider must be able to report per-term occurrence
  counts to participate. A purely semantic backend that can only return its own opaque
  relevance score cannot rank coherently against the others, and is out of scope for
  this design.

### Ranking mechanics

- **Keep boolean AND matching.** Rejected. `spektacular:internal/store/search.go:70-75`
  currently returns early when any term's count is zero, so a document missing one
  term is excluded rather than ranked low. The binding decision entry
  `decisions/ranked-or-over-boolean-and.md` records this as already decided
  against, with the alternatives (AND plus richer entries, AND with an OR
  fallback, query expansion, an absolute threshold) each considered and rejected
  there.
- **Length normalisation / BM25 to satisfy "volume does not beat relevance".**
  Rejected as out of scope: the spec's Non-Goals exclude "full BM25-style scoring
  over the corpus... not a general relevance model", and it introduces a
  corpus-dependent average-length term that changes meaning as a store grows.
  Aboutness is carried by the tag weight instead. See Open assumptions.
- **Weighting title and heading matches.** Rejected: explicitly a Non-Goal
  ("beyond the tag weighting introduced here"). Note the machinery already exists
  (`spektacular:internal/store/search.go:218-232` `headingText`), so this is a
  deliberate omission, not an unavailable option.
- **An absolute score cutoff.** Rejected per the spec's Technical Approach and the
  decision entry: it needs retuning per corpus.

### How a query term is matched against a tag

Measured at the plan walkthrough, after the user proposed partial matching with a
reduced score. Scores below are `1 - levenshtein/max(len)` for edit distance, and
`len(shorter)/len(longer)` for prefix, with prefix gated on the shorter string
reaching four characters.

| search | tag | edit distance | prefix | what it is |
| --- | --- | --- | --- | --- |
| `https` | `https` | 1.00 | 1.00 | exact |
| `http` | `https` | 0.80 | 0.80 | should match, reduced |
| `https` | `https-security` | 0.36 | 0.36 | should match, weakly |
| `apple` | `apples` | 0.83 | 0.83 | should match |
| `test` | `testing` | 0.57 | 0.57 | should match |
| `route` | `routing` | 0.57 | **0.00** | prefix misses this |
| `test` | `rest` | **0.75** | 0.00 | unrelated |
| `cors` | `core` | **0.75** | 0.00 | unrelated |
| `tls` | `tld` | **0.67** | 0.00 | unrelated |
| `dns` | `dos` | **0.67** | 0.00 | unrelated |
| `api` | `apt` | **0.67** | 0.00 | unrelated |
| `go` | `golang` | 0.33 | 0.00 | overreach |

- **Exact matching only.** Rejected, having originally been chosen. It makes an entry
  tagged `apple` unreachable by a search for `apples`, and the only mitigation
  available was instructing the capture agent to carry both forms, which nothing
  enforces and which fails silently when missed. Replacing prose guidance with
  arithmetic is strictly better here, because the failure it prevents is invisible.
- **Edit distance (Levenshtein, or a normalised variant).** Rejected. It is the
  obvious candidate and it is actively dangerous on this corpus. Tags are
  overwhelmingly short technical tokens, which is exactly where edit distance is
  weakest: the table above shows it scoring five unrelated pairs between 0.67 and 0.75,
  every one of them a plausible tag in this codebase. `test`/`rest` at 0.75 would give
  an entry about REST most of the tag weight for a search about testing. On every pair
  that *should* match, it produces scores identical to prefix matching, so it buys
  nothing in exchange for that risk.
- **Jaro-Winkler.** Rejected for the same reason in weaker form: it boosts common
  prefixes, which helps, but it still assigns non-zero similarity to unrelated short
  tokens, and it is far harder to explain in published documentation than a length
  ratio. The docs now publish the algorithm, which puts a real premium on
  explicability.
- **Plain substring matching.** Rejected: an unanchored match lets any short term hit
  every tag containing it, firing the tag weight constantly and destroying it as a
  discriminating signal. This is the failure the two guards on prefix matching exist to
  prevent.
**Prior art.** This is the *vocabulary mismatch* problem, one of the oldest in
information retrieval, and the design below is a deliberate selection from established
technique rather than an invention. Three points of contact, checked at the
walkthrough:

- **Prefix matching is standard, and has a name.** Lucene and Elasticsearch implement
  it as *edge n-grams*: at index time a term is expanded into every prefix, so a search
  for `ret` finds `return`. That is the same relation this design computes at scan
  time. We compute rather than index because Spektacular has no index: search is a full
  filesystem walk over markdown, with no analyzer pipeline and no inverted index. The
  idea transfers; the machinery does not.
- **The minimum-length guard is standard too.** Elasticsearch's `fuzziness: AUTO`
  permits 0 edits for terms of 1-2 characters, 1 edit for 3-5, and 2 edits for 6 or
  more, on the reasoning that a fixed tolerance is far too permissive on short terms.
  That is precisely the failure this plan measured on `test`/`rest` and `tls`/`tld`,
  arrived at independently. `minPrefixLen = 4` sits inside the band where Elasticsearch
  is already being cautious, which is a useful corroboration of the starting value.
- **Proportional partial credit is where this design departs from the standard, and
  the departure is deliberate.** Lucene's `PrefixQuery` rewrites to a *constant score*:
  a prefix match is binary, and every match scores the same. It can afford that because
  BM25, inverse document frequency and field-length norms do the discriminating
  downstream, so a weak prefix hit on a common term is separated from a strong one by
  other signals. This project has none of those: no corpus statistics, no IDF, and
  length normalisation is an explicit Non-Goal. With nothing downstream to separate
  them, the discrimination has to live in the match itself, which is what the length
  ratio provides.

**The established answer we are declining.** For `route` versus `routing`, the
canonical solution is *stemming* (Porter, 1980; Snowball), which reduces both to a
common root, or lemmatisation, which does it with a dictionary. Both are mature,
ubiquitous, and would close the gap prefix matching leaves. They are declined because
query expansion is a spec Non-Goal, because a stemmer is a model of English that
behaves unpredictably on technical tokens (`https`, `kubernetes`, `redis`), and because
the user judged at the walkthrough that `route` and `routing` are substantially
different subjects and the gap does not matter. Recorded so a future reader knows the
named alternative was considered and rejected, rather than unknown.

- **Chosen: bidirectional prefix with proportional partial credit.** A term matching a
  tag exactly earns the full tag weight. Where one of the two strings is a prefix of
  the other and the shorter reaches a minimum length, it earns
  `len(shorter)/len(longer)` of it. Where several tags match, the strongest wins rather
  than accumulating. It reproduces edit distance's scores on every case that should
  match, scores zero on every near-miss pair tested, and its numbers fall out of the
  intuition that prompted it: the user independently proposed 100 for `https`/`https`
  and 80 for `http`/`https`, and `4/5 = 0.80`.
  **Known gap accepted:** `route` does not match `routing`, because `route` is not the
  opening of `routing`. Prefix matching is positional, not morphological. Closing that
  gap needs stemming, which remains a Non-Goal; the capture flow carries a rule to add
  a second tag where a form differs by more than its ending.

### Tag-filter surface

- **A new `knowledge tags` subcommand or a bespoke query syntax (`tag:http`).**
  Rejected: the spec's Technical Approach directs us to "follow the existing
  `--tier` / `--filter` narrowing shape... so narrowing a search stays one idea".
  The existing narrowing flags are registered together at
  `spektacular:cmd/knowledge.go:522-525`.
- **Chosen: a repeatable `--tag` flag on `knowledge search`**, registered beside
  `--tier`/`--filter`, carried on `knowledge.Selector`
  (`spektacular:internal/knowledge/address.go:41-44`).

### Where tag proposal at capture time lives

- **A CLI command that proposes tags.** Rejected: proposal is a judgement the
  agent makes from the entry it just wrote; the CLI's job is to report the
  vocabulary in use. A `knowledge tags` *listing* is mechanical and belongs in the
  CLI; choosing from it is the skill's job. This mirrors how the contribute flow
  already works, loading `knowledge categories`
  (`spektacular:templates/skills/workflows/spek-knowledge/SKILL.md:52`) and then
  routing by judgement at `:53`.
- **Chosen: a mechanical `knowledge tags` listing command + skill prose.**

---

## Chosen approach — evidence

### The scoring seam already exists

- `spektacular:internal/store/search.go:46-52` — `FileStore.Search(query string)`
  splits the query into lower-cased terms and delegates to `search(terms)`.
  Extending the signature with an options struct is contained: the only
  implementors/callers of `Store.Search` in the whole tree are
  `spektacular:internal/store/search.go:46`,
  `spektacular:internal/store/ignore.go:107` (a pass-through decorator), and
  `spektacular:internal/knowledge/set.go:153`. Nothing outside the knowledge layer
  consumes it, so the interface change has no blast radius.
- `spektacular:internal/store/search.go:56-99` — `search` walks the root and
  computes `score` at `:70-77`; this is the single site where AND-exclusion and
  score summing both live.
- `spektacular:internal/store/search.go:126-133` — `fileAggregate` already carries
  `counts []int` per term. A `tags []string` field alongside it is the natural
  place for parsed tags.
- `spektacular:internal/store/search.go:135-215` — `scanFile` is the one-pass
  reader. The frontmatter block is the first thing in the file, so tags can be
  parsed from the leading lines of the same scan without a second read.

### The merge/cutoff seam already exists

- `spektacular:internal/knowledge/set.go:140-190` — `Set.Search` merges hits from
  every covered store, sorts by score desc then source index then path
  (`:167-176`), and then post-filters (`:178-189`, the always-applied category
  exclusion). The relative cutoff drops into that same post-filter loop, after the
  sort, where the best score is `merged[0]`.
- `spektacular:internal/knowledge/set.go:167-176` — the tie-break is already
  deterministic (source order then path), which the new float scores must not
  disturb. `TestSet_SearchTieBreaksBySourceOrderThenPath`
  (`spektacular:internal/knowledge/set_test.go:157`) pins it.

### The tag filter seam already exists

- `spektacular:internal/knowledge/address.go:41-44` — `Selector{Tier, Filter}` is
  the value every fan-out command travels on.
- `spektacular:internal/knowledge/address.go:159-172` — `Selector.covers` is store
  membership only; a tag filter is per-entry, not per-store, so it belongs
  alongside the always-applied exclusion in the post-filter loop (or in the store
  via options), not in `covers`.
- `spektacular:cmd/knowledge.go:520-526` — `init()` registers `--tier` and
  `--filter` across the four fan-out commands; `--tag` registers on
  `knowledgeSearchCmd` only.
- `spektacular:cmd/knowledge.go:107-112` — `knowledgeNarrowingFlags` is the schema
  the `--schema` mode advertises; a new flag must be declared here or the
  agent-facing contract goes stale.

### Frontmatter parsing has a proven local shape

- `spektacular:internal/metadata/frontmatter.go:16-52` — `Split` is the working
  reference implementation: `---\n` prefix check, find `\n---`, require the closer
  on its own line, drop one blank separator line, `yaml.Unmarshal` the block.
  Copy the shape, not the schema.

### The `--schema` contract is enforced by tests

- `spektacular:cmd/knowledge_test.go:418` —
  `TestKnowledgeSearch_SchemaDeclaresPerDocumentHitFields` asserts the advertised
  hit fields. Adding `tags` to `store.Hit` requires adding it here.
- `spektacular:cmd/knowledge_test.go:1666` —
  `TestKnowledgeWrite_SchemaAdvertisedFieldsAloneIssueASuccessfulWrite` asserts the
  schema is sufficient to drive a real write.

### The skill is a single template with generated copies

- Source of truth: `spektacular:templates/skills/workflows/spek-knowledge/SKILL.md`
  (83 lines). Contribute flow at `:52-62`, lookup flow at `:29-46`, update flow at
  `:68-77`. There is no tag concept anywhere in it today, and no audit intent (the
  skill has exactly three: lookup / contribute / update).
- `spektacular:templates/templates.go:6-7` — `//go:embed all:*` over `templates/`.
- `spektacular:internal/agent/skills.go:25-31` — `workflowSkills` registry;
  `spek-knowledge` at `:29`.
- `spektacular:internal/agent/skills.go:41-66` — `installWorkflowSkills` renders
  `{{command}}` via mustache at `:48` and writes `SKILL.md` at `:58-61`.
- `spektacular:cmd/init.go:67` — `runInit` ends in `a.Install(...)`, which
  unconditionally overwrites the installed skill files.
- Generated, git-tracked copies that must be regenerated and committed:
  `spektacular:.claude/skills/spek-knowledge/SKILL.md` and
  `spektacular:.bob/skills/spek-knowledge/SKILL.md`, via `go run . init claude`
  and `go run . init bob`.
- `spektacular:internal/agent/commands.go:22` — `workflowDescriptions` duplicates
  the SKILL.md frontmatter description; both change together or neither.

### The capture-time trigger points

- `spektacular:templates/agents/knowledge-trigger.md:15-25` — the managed
  AGENTS.md section that tells an agent to *offer* to capture, and on accept to
  invoke the `spek-knowledge` skill. Guarded by
  `spektacular:internal/agent/knowledge_trigger_test.go:117` (picks up template
  changes).
- `spektacular:templates/steps/implement/07-update_changelog.md:47-51` and
  `spektacular:templates/steps/plan/18-walkthrough.md:22` — the two workflow-side
  hand-offs into the skill. Contract tests at
  `spektacular:internal/steps/implement/steps_test.go:370-401` and
  `spektacular:internal/steps/plan/steps_test.go:487-516` require the hand-off
  names the skill and never emits a `skill spek-knowledge` CLI call.

### The retrieval-side prose that goes stale

- `spektacular:templates/steps/plan/02-discovery.md:16` — describes the ranked-hit
  shape to the planning agent; must gain tags and `--tag`.
- `spektacular:templates/skills/skill_spawn-planning-agents.md:22` — same,
  for research agents.
- `spektacular:docs/knowledge-base.md:78-93` — "Search behaviour", currently states
  "A document matches when every query word occurs somewhere in it". This becomes
  false.
- `spektacular:docs/knowledge-base.md:185-197` — the command reference table.
- `spektacular:README.md:137` — one-line `knowledge search` description.

### The published documentation

- `docs:src/pages/knowledge-base.mdx` (315 lines), route `/knowledge-base/`. The
  ranking sentence to replace is at `:122-130`: "Results are scored by how often
  the query terms appear, with ties broken by store order and then path, so the
  ordering is stable." `knowledge write` is documented at `:106-120`,
  `knowledge search` at `:122-139`.
- Section run for the `surface` alternation convention: What it is (plain, `:17`)
  → The six categories (`surface`, `:44`) → The lifecycle of an entry (plain,
  `:94`) → Configuration (`surface`, `:152`) → Why it works this way (plain,
  `:247`) → `CtaBanner` (`:304`). A new band appended before the CtaBanner must be
  `surface`.
- `docs:src/components/sections/Section.astro` + `Prose.astro` for a band with a
  markdown table (the pattern at `docs:src/pages/knowledge-base.mdx:69-76`);
  `ConfigurationKeys.astro` + `ConfigKey.astro` for a flag list (canonical example
  at `docs:src/pages/configuration.mdx:166-180`).
- `docs:src/components/Nav.astro:6-21` — nav registration, trailing slashes
  required by the active-state matcher at `:25-30`. No new page is needed, so no
  nav change is expected.
- Build/verify: `npm run build` (`docs:package.json:6-11`) and `make check` →
  `npx astro check`. CI (`docs:.github/workflows/deploy.yml`) runs build only.
- Binding docs-repo conventions (loaded in full this session): `mdx-authoring.md`
  Rule 1 (no `<div>`/`<section>`/`class=` in `src/pages/*.mdx`; guard is
  `grep -nE "<div|<section|class=" src/pages/*.mdx` → 0 matches), Rule 3 (blank
  line inside slotted components), Rule 4 (fenced markdown code blocks);
  `no-em-dashes.md`; `alternate-section-background.md`; `plan-content-pages.md`
  (the plan **must** carry a labelled `**Content outline**` / `**Content example**`
  block for the docs change).

### Prototype: the proposed weights hit the success metric

Prototyped in Python over this repo's real knowledge base
(`spektacular:.spektacular/knowledge`), with the routing probe entry from the
spec's working context recreated and tagged
`[go, http, api, routing, handler, endpoints, middleware, chi]`, and the
`decisions/ranked-or-over-boolean-and.md` entry excluded so the corpus matches the
one the baseline was measured against.

Formula: `score = (Σ_terms [tagWeight·isTag(t) + damp(count(t))]) × coverage²`,
where `damp(n) = 0 for n=0, else 1 + log₂(n)`, `coverage` = fraction of query terms
with any evidence, `tagWeight = 8`, and hits below `0.25 × bestScore` are dropped
after the cross-store merge.

| Query | Baseline rank (spec) | Prototype rank | Prototype score |
| --- | --- | --- | --- |
| `http endpoints go` | 2nd | **1st** of 2 | 28.58 vs 11.52 |
| `http handler` | not measured on this corpus | **1st** of 1 | 18.58 |
| `http endpoints handler routing` | not first | **1st** of 1 | 36.58 |
| `api` | **no results at all** | **1st** of 7 | 8.00 vs 4.70 |
| `routing` (extra) | not in top 4 | **1st** of 5 | 9.00 vs 5.39 |
| `http middleware` (extra) | 1st | **1st** of 1 | 19.58 |

This clears the spec's success metric ("the observed-query benchmark goes from 0
to 4") and the `api` reversal. Two properties worth carrying into implementation:

1. `damp(1) = 1`, so a single occurrence of a single term still scores exactly
   `1.0`. That keeps every existing `Score: 1` assertion green
   (`spektacular:cmd/knowledge_test.go:245,253,772,780,788,1160`).
2. The relative cutoff is doing real work: `http endpoints go` returns 2 of the
   corpus rather than 10, and `http handler` returns 1.

The weights are a validated starting point, not a fixed contract; the spec
explicitly expects tuning during implementation.

### Existing tests that must change (not merely be added to)

- `spektacular:internal/store/search_test.go:153`
  `TestSearch_ScoreSumsOccurrencesAcrossDocument` — hand-written oracle
  `{twice:2, once:1, upper:1, spread:4}`. Under damping, `spread` becomes
  `1+log₂(4)=3`. The oracle must be recomputed by hand, not derived.
- `spektacular:internal/store/search_test.go:286` — asserts `float64(8)` for
  `widget gear` over one document (`widget` 3 + `gear` 5).
- `spektacular:internal/store/search_test.go:437`
  `TestSearch_SingleWordMatchesSameFiles` and the AND-matching assumption
  throughout — ranked OR changes which files come back for multi-term queries.
- `spektacular:internal/knowledge/set_test.go:124`
  `TestSet_SearchRanksAcrossSourcesByScore` — the *order* survives (3 occurrences
  → 2.58 still beats 1 → 1.0), but the comment's hand-computed scores go stale.
- `spektacular:cmd/knowledge_test.go:19-26` — the `knowledgeHit` mirror struct.
  Adding `Tags` to `store.Hit` does **not** break the existing `require.Equal`
  comparisons (unknown JSON fields are ignored on unmarshal), so these only need
  changing where tags are actually being asserted.

### Test conventions to follow

- Fixtures are written into `t.TempDir()` via a `NewFileStore`, never read from a
  real directory (`spektacular:internal/store/search_test.go:19-36`).
- Expected values are hand-maintained literal oracles, explicitly *not* derived
  from the subject under test (`spektacular:internal/store/search_test.go:176-179`,
  and the comment at `spektacular:internal/agent/instruction_surface_test.go:79-81`).
- Skill assertions render through the real install path into a temp dir
  (`spektacular:internal/agent/instruction_surface_test.go:82-104`), never reading
  the committed `.claude/` copy.
- `spektacular:.spektacular/knowledge/conventions/error-messages-must-suggest-remediation.md`
  — every new error path must be `output.NewError(code, msg).WithNextAction(...)`
  with a runnable next step, never a bare `fmt.Errorf`.
- `spektacular:.spektacular/knowledge/conventions/tests-must-pass-for-done.md` —
  `go test ./...` must pass in full before the work is called done.

---

## Files examined

**spektacular (`/home/nicj/code/github.com/jumppad-labs/spektacular`)**

- `internal/store/search.go:46-52` — `Search` entry point; splits and lower-cases terms, returns empty for a no-term query.
- `internal/store/search.go:70-77` — the AND-exclusion and score-summing site; the single place ranking changes.
- `internal/store/search.go:126-133` — `fileAggregate`: per-term counts, title, excerpt candidates, checksum.
- `internal/store/search.go:135-215` — `scanFile`: the one-pass reader (binary sniff, SHA-256 tee, line scan). Frontmatter parses here.
- `internal/store/search.go:218-232` — `headingText`; heading machinery exists but title weighting is a Non-Goal.
- `internal/store/store.go:25-36` — `Hit`; `Score` is already `float64`, so damped scores need no type change. `Tags` goes here.
- `internal/store/store.go:38-57` — `Store` interface; `Search(query string)` is the signature to extend.
- `internal/store/ignore.go:107-119` — `ignoreStore.Search` pass-through decorator; must forward any new options.
- `internal/knowledge/set.go:140-190` — `Set.Search`: cross-store merge, global sort, post-filter. The cutoff and tag filter land here.
- `internal/knowledge/set.go:192-199` — `categoryOf`; category is derived from the path's first segment.
- `internal/knowledge/address.go:36-44` — `Address` and `Selector`; `Selector` gains the tag list.
- `internal/knowledge/address.go:46-70` — knowledge error codes; a tag-related refusal needs one.
- `internal/knowledge/category.go:57-113` — the `Categories` registry, the model for how a tag-vocabulary listing should be shaped.
- `internal/metadata/frontmatter.go:16-52` — `Split`; the reference frontmatter shape to copy.
- `internal/metadata/metadata.go:83-108` — `UnmarshalYAML`; proves why this package cannot be reused for knowledge tags.
- `cmd/knowledge.go:107-112` — `knowledgeNarrowingFlags`, the advertised flag schema.
- `cmd/knowledge.go:114-134` — `knowledgeSearchOutputSchema`; `tags` must be declared here.
- `cmd/knowledge.go:315-337` — `runKnowledgeSearch`.
- `cmd/knowledge.go:495-506` — `knowledgeSelector()`, built from the flag vars.
- `cmd/knowledge.go:516-527` — flag registration `init()`.
- `templates/skills/workflows/spek-knowledge/SKILL.md:29-46` — lookup intent.
- `templates/skills/workflows/spek-knowledge/SKILL.md:52-62` — contribute intent; steps 1-7 of the capture flow.
- `templates/skills/workflows/spek-knowledge/SKILL.md:68-77` — update intent.
- `templates/agents/knowledge-trigger.md:15-25` — the capture offer.
- `templates/steps/plan/02-discovery.md:16` — the ranked-hit description given to planning agents.
- `templates/skills/skill_spawn-planning-agents.md:22` — same for research agents.
- `internal/agent/skills.go:25-31,41-66` — skill registry and the `{{command}}` render/install path.
- `internal/agent/commands.go:18-24` — `workflowDescriptions`, duplicated from SKILL.md frontmatter.
- `cmd/init.go:25-68` — `runInit`; skills are unconditionally regenerated.
- `internal/agent/instruction_surface_test.go:82-104` — the hand-maintained CRUD-invocation guard to extend.
- `internal/agent/claude_test.go:26-32`, `bob_test.go:24-33,49`, `codex_test.go:29` — installed-skill marker assertions.
- `docs/knowledge-base.md:78-93,185-197` — internal maintainer doc; search behaviour and command reference go stale.
- `README.md:137` — one-line search description.
- `.spektacular/knowledge/decisions/ranked-or-over-boolean-and.md` — binding; ranked OR, coverage scaling, relative cutoff.

**docs (`/home/nicj/code/github.com/jumppad-labs/spektacular-website`)**

- `src/pages/knowledge-base.mdx:94-150` — "The lifecycle of an entry"; holds both `knowledge write` (`:106-120`) and `knowledge search` (`:122-139`).
- `src/pages/knowledge-base.mdx:122-130` — the exact ranking sentence to replace.
- `src/pages/knowledge-base.mdx:44-92` — "The six categories"; the `Section` + `Prose` + markdown-table pattern to model on.
- `src/pages/knowledge-base.mdx:228-241` — existing `--tier`/`--filter` narrowing prose; `--tag` joins it.
- `src/pages/knowledge-base.mdx:282-290` — "De-duplication is exact, not fuzzy"; argues against similarity scoring, so new ranking prose must not contradict it.
- `src/pages/configuration.mdx:166-180` — the canonical `ConfigKey` authoring pattern.
- `src/components/sections/Section.astro`, `Prose.astro`, `ConfigurationKeys.astro`, `ConfigKey.astro` — the components a docs change would use.
- `src/pages/knowledge-base.mdx:94-150` — "The lifecycle of an entry"; the two new
  ranking and tagging bands are inserted immediately after it, before "Configuration"
  (`:152`), which keeps the plain/surface alternation intact with no downstream edits.
- `src/pages/extending.mdx:29-48` — the published `Store` interface block, including
  `Search(query string) ([]Hit, error)` at `:47`. This is the contract a new provider
  author builds against, and this plan changes it.
- `src/pages/extending.mdx:59-65` — the published `Hit` struct. **Already drifted from
  the code before this work**: shows `Excerpt string` (singular) where
  `spektacular:internal/store/store.go:25-36` has `Excerpts []string`, and omits
  `Title`, `Category` and `Checksum`. Nothing ties this page to the Go source, so
  nothing caught it.
- `src/pages/extending.mdx:64` — `Score float64 // optional relevance score; 0 if
  absent`. Becomes actively misleading under the revised design: `Score` is not
  optional-for-the-store, it is not the store's to set.
- `src/pages/extending.mdx:78-84` — the `Search` method-contract bullet, which already
  articulates the `Tier`/`Name` stamping precedent this plan's Decision 1 leans on.
  The natural place to extend that reasoning to `Score`.
- `src/pages/extending.mdx:126-135` — "Registering a backend"; names
  `internal/knowledge/set.go`'s provider switch as the worked example.
- `src/components/Nav.astro:6-21` — nav array (no change expected).
- `package.json:6-11`, `Makefile` — `npm run build`, `make check`.
- `.spektacular/knowledge/conventions/{mdx-authoring,site-layout,alternate-section-background,no-em-dashes,plan-content-pages}.md` — binding authoring rules.

---

## External references

- BM25 / Okapi ranking (Robertson & Zaragoza, *The Probabilistic Relevance
  Framework*). Consulted for the shape of saturation and length normalisation.
  Why it mattered: it is the reference the spec's Non-Goals point at when
  excluding "full BM25-style scoring", and it confirms that saturation of repeated
  occurrences (the `k1` term) is separable from length normalisation (the `b`
  term). This plan takes the first and deliberately leaves the second.
- [Lucene `PrefixQuery`](https://lucene.apache.org/core/4_6_0/core/org/apache/lucene/search/PrefixQuery.html)
  — rewrites via `CONSTANT_SCORE_AUTO_REWRITE_DEFAULT`. Why it mattered: it establishes
  that the standard treatment of a prefix match is *binary*, which is what makes this
  plan's proportional partial credit a conscious departure rather than an oversight.
  Lucene can afford a constant score because BM25 and IDF discriminate downstream; this
  project has neither.
- [Elasticsearch fuzzy query / `fuzziness: AUTO`](https://www.elastic.co/docs/reference/query-languages/query-dsl/query-dsl-fuzzy-query)
  — 0 edits for terms of 1-2 characters, 1 for 3-5, 2 for 6 or more (configurable as
  `AUTO:lo,hi`, default `AUTO:3,6`). Why it mattered: independent confirmation that
  short terms need a length guard, which this plan reached empirically from
  `test`/`rest` and `tls`/`tld`. It corroborates `minPrefixLen = 4` as a sane starting
  value.
- Elasticsearch *edge n-grams* — the standard indexing technique for prefix matching,
  expanding a term into every prefix at index time. Why it mattered: it is the
  established name for the relation this plan computes at scan time. Spektacular has no
  index, so the idea transfers and the machinery does not.
- Porter, M.F. (1980), *An algorithm for suffix stripping* — the Porter stemmer, and
  its Snowball successors. Why it mattered: this is the canonical established solution
  to the `route`/`routing` gap that prefix matching leaves open. Recorded as knowingly
  declined (query expansion is a Non-Goal; stemmers behave unpredictably on technical
  tokens; and the user judged the gap immaterial), not as unknown.
- GitHub issue [#35](https://github.com/jumppad-labs/spektacular/issues/35) and its
  two follow-up comments — the origin of this work and the source of the measured
  baseline table reproduced in `.spektacular/context.md`.

---

## Prior plans / specs consulted

- `.spektacular/specs/000050_knowledge-entry-tags.md` — the spec this plan
  implements. Requirements, constraints, acceptance criteria, non-goals.
- `.spektacular/context.md` — the spec interview's working context. Carries the
  measured baseline table (query → top hit → score → routing-doc rank), the
  three-run field evidence that the agent's query is unstable, and the symmetry
  argument for why the writing agent should propose the tags it would itself
  search for.
- Prior plan `000028_knowledge-base-categories-tiers-and-dedup` — established the
  category registry, the always-applied/looked-up tier split, and the exact-byte
  de-dup layer. Learned: the category model is deliberately declared **once** in
  `internal/knowledge/category.go` and projected everywhere else; a tag vocabulary
  listing should follow the same single-declaration discipline rather than
  hard-coding a tag list anywhere.
- Prior plan `000047_repo-scoped-knowledge-addressing` — established
  `Address`/`Selector`, the `--tier`/`--filter` narrowing shape, and the rule that
  a filter naming an unreachable store is refused rather than silently empty. The
  `--tag` filter inherits that discipline.
- Prior spec `000022_spek-knowledge-skill` — the origin of the propose-then-confirm
  capture contract that tags must slot into without weakening.

---

## Open assumptions

These are assumed but not verified. **If any turns out wrong, the implement
workflow must STOP and ask.**

1. **"About that term" means "tagged with that term".** The acceptance criterion
   "Volume does not beat relevance" is satisfied by the tag weight, not by
   document-length normalisation, and its test fixture pairs a **tagged** short
   entry against an **untagged** long entry. Under any monotonic damping function,
   20 body occurrences still outscore 2, so damping alone cannot reverse the
   pairing; and both alternative signals (length normalisation, title weighting)
   are explicit Non-Goals. If the user intends this criterion to hold between two
   *untagged* entries, the scoring model needs length normalisation and the
   Non-Goals need revisiting. **This is the single most likely thing to be wrong in
   this plan.**
2. **The tag weight of 8 and cutoff of 0.25 are starting values.** Validated
   against this repo's corpus (see the prototype table above), but the spec says
   "expect the weights, damping curve and cutoff fraction to be tuned during
   implementation". Tuning that keeps the benchmark at 4/4 is in scope and needs
   no escalation; changing the *shape* of the formula does.
3. **Tags are flat, lower-case, free-form strings.** No hierarchy, no namespaces,
   no controlled vocabulary file. Convergence comes from the capture flow
   preferring an existing tag, not from validation. Nothing in the spec asks for
   more, and the bulk-rename/merge tooling that a controlled vocabulary would need
   is an explicit Non-Goal.
4. **Matching a tag is exact, not substring.** A query term matches a tag when the
   two are equal after lower-casing. Body matching stays substring, as today. This
   keeps `go` from matching a `golang` tag and, more importantly, keeps a short
   query term from matching every tag that contains it.
5. **The benchmark test uses a controlled fixture in `t.TempDir()`, not the live
   `.spektacular/knowledge` directory.** The prototype above exposed exactly why:
   adding `decisions/ranked-or-over-boolean-and.md` to the corpus (an entry that
   quotes all four benchmark queries verbatim) moves the probe from rank 1 to rank
   2 on `http handler`. A benchmark bound to a live, growing directory would flap.
6. **`store.Hit.Tags` serialises as `[]string` with no `omitempty`.** An untagged
   entry reports `"tags": []`, not an absent key, so a consumer never has to
   distinguish "no tags" from "this build has no tag support".
7. **The audit intent reads and proposes only.** It has no CLI command of its own
   beyond `knowledge list` / `knowledge read` / `knowledge tags`; every proposed
   change goes back through the existing `knowledge write` after explicit
   confirmation, per entry. The Non-Goals forbid bulk rename/merge.
8. **Both repos are on their default branches and writable at the roots reported
   by `repo list`.** Verified present this session; not verified as clean or
   pushed.

---

## Drafting assumptions

The judgement calls made while drafting this plan: what was decided, why it was
the reasonable default, and what was rejected. Presented for challenge at the
walkthrough.

### "Aboutness" is carried by tags, not by title or length (discovery)
- **Decision**: The acceptance criterion "Volume does not beat relevance" (a long
  entry mentioning a term many times must not outrank a short entry that is
  *about* that term) is satisfied by the tag weight, not by document-length
  normalisation. Its test fixture pairs a **tagged** short entry against an
  untagged long entry.
- **Rationale**: The spec's own Non-Goals rule out both full BM25-style scoring
  ("not a general relevance model") and weighting titles/headings above body
  matches. With those two excluded, the only signal available to express
  "this entry is about X" is a tag. Damping alone cannot reverse the pairing:
  under any monotonic damp, 20 body occurrences still score above 2.
- **Rejected**: (a) Divide the body component by document length. Directly adjacent
  to the excluded "general relevance model", and introduces a corpus-dependent
  average-length term the spec asks us to avoid. (b) Weight the title. Explicitly
  a non-goal. (c) Test the criterion with two untagged entries. Would make the
  criterion unsatisfiable under the spec's own constraints.

### Ranked-OR is planned as new work, not assumed present (discovery)
- **Decision**: Treat `decisions/ranked-or-over-boolean-and.md` as binding and plan
  the ranked-OR matcher as work to do, even though `internal/store/search.go:70-75`
  still requires every query term to occur.
- **Rationale**: The project's standing rule is that a knowledge entry outranks the
  code it describes. The entry states the target behaviour; the code has not caught
  up. The spec's requirements restate the same behaviour independently.
- **Rejected**: Reading the code as evidence the decision entry is stale.

### The relative cutoff is applied after cross-store merge (discovery)
- **Decision**: Per-document scoring lives in the store; the "drop weak matches
  relative to the best hit" cutoff lives in `knowledge.Set.Search`, after hits from
  every covered store are merged.
- **Rationale**: The strongest hit for a query may sit in a different store from a
  weak one. A per-store cutoff would keep a weak hit alive purely because its own
  store had nothing better, which is exactly the behaviour the criterion "a weak
  match appears alone and disappears in company" forbids.
- **Rejected**: Applying the cutoff inside `FileStore.Search`. Cheaper, but wrong
  across stores.

### Knowledge tag frontmatter does not reuse internal/metadata (discovery)
- **Decision**: Parse knowledge-entry tags with a small dedicated frontmatter
  reader rather than `metadata.Split`.
- **Rationale**: `metadata.Metadata.UnmarshalYAML` (internal/metadata/metadata.go:83-108)
  hard-requires a parseable `created_date` and a `status` from a four-value enum,
  and `Render` drops unknown keys. A knowledge entry carrying only `tags:` would be
  rejected as malformed, and a round-trip through `Render` would delete the tags.
  That package owns the *workflow artifact* schema, which is a different schema.
- **Rejected**: Extending `metadata.Metadata` with a `Tags` field. It would put a
  knowledge-only field on every spec, plan and changelog artifact and still leave
  `created_date`/`status` mandatory on knowledge entries, which the spec's
  no-migration constraint forbids.

### Chosen direction: stores report evidence, the knowledge layer ranks (architecture, revised at walkthrough)
- **Decision**: Stores find candidate documents and report evidence (per-term body
  occurrence counts, the entry's tags, plus locator, title, excerpts and checksum) and
  compute no score. The knowledge layer tokenizes the query, applies the scoring
  formula, merges across stores, sorts, and applies the best-hit-relative cutoff and
  the tag filter. Tags are declared as YAML frontmatter read by a new small parser.
  `--tag` is a repeatable filter beside `--tier`/`--filter`. The CLI provides a
  mechanical `knowledge tags` listing; which tag to propose stays a `spek-knowledge`
  skill judgement.
- **Rationale**: Ranking is a property of the whole result set, so it belongs at the
  layer that has the whole result set. Putting it there also costs nothing: the store
  already collects exactly this evidence in the single file pass it performs today.
  The alternative makes the formula a contract every future provider must reproduce
  exactly, and the project plans further providers.
- **Rejected**: (a) Per-provider scoring inside each store — the originally drafted
  direction. It looks cheaper for a single file store, but a second implementation
  that tunes or rounds differently produces scores that are silently incomparable in
  the cross-store merge, and the relative cutoff then spans two scales. (b) Scoring
  including the cutoff inside `FileStore.Search` — same objection, plus the store
  cannot see sibling stores at all, so "a weak match disappears in company" fails
  across stores.
- **History**: The first draft chose (a), rejecting knowledge-layer scoring on the
  grounds that it would force a second read of every file. That was a mistake: it
  confused where a score is computed with where a file is read. The user caught it at
  the walkthrough by asking what happens when a new store is implemented.
- **Consequence accepted**: `Hit` becomes partly an intermediate value, carrying
  per-term counts that no consumer outside the ranking step reads (they are
  `json:"-"`, so the published result shape is unaffected). And a provider that cannot
  report per-term counts cannot participate in this ranking, which is a real
  constraint on future providers, now stated in the interface's doc comment rather
  than discovered later.

### Effort shape of the chosen direction (architecture, revised at walkthrough)
- **Decision**: Treat the engine change (store evidence reporting + knowledge-layer
  ranking) as Medium, the CLI surface as Low, the skill/template prose as Medium (five
  template files plus regenerated copies and their guard tests), and the documentation
  as Low except the published knowledge-base page, which is Medium.
- **Rationale**: The seams already exist and are well tested; most of the engine risk
  is in recomputing the existing hand-written score oracles rather than in new code.
- **Rejected**: Treating the template work as Low. It touches five templates, two
  generated-and-committed copies, and four guard tests that pin their content.
- **Revised twice at the walkthrough**: Phase 1.1 moved Low → Medium when ranking moved
  above the store interface, since it now spans three packages rather than one file.
  Phase 4.2 moved Low → Medium when the docs took on the full algorithm, the constants
  and a worked example, plus two new page bands.

### Conventions selected without asking (architecture)
- **Decision**: Applied two `spektacular` conventions (error remediation, passing
  tests) and five `docs` conventions (plan-content-pages, no-em-dashes,
  mdx-authoring, alternate-section-background, site-layout). Dropped the
  file-scoped-heading convention and both category README entries.
- **Rationale**: The step directs that relevance is decided and recorded, not
  asked. The dropped heading rule binds new file-scoped reference sections; this
  feature edits prose inside an existing page and creates none.
- **Rejected**: Listing every loaded convention. The step explicitly calls a padded
  list a visible signal the knowledge base was not consulted.

### Tag matching is prefix-with-partial-credit, body matching stays substring (architecture, revised at walkthrough)
- **Decision**: A query term earns the full tag weight on an exact tag match, and
  `len(shorter)/len(longer)` of it where the term and a tag are prefix-related and the
  shorter reaches `minPrefixLen` (starting at 4). The strongest matching tag decides;
  affinities do not sum. Body matching stays a case-insensitive substring count.
- **Rationale**: Exact matching pushed the entire burden of word forms onto the capture
  agent, enforced only by prose that fails silently when ignored. Prefix partial credit
  makes `apple`/`apples` relate arithmetically on every search. The two guards (anchor
  at the start, minimum length) prevent the "any short term hits everything" failure
  that ruled out plain substring matching.
- **Rejected**: edit distance and Jaro-Winkler, both of which score unrelated short
  technical tokens dangerously high (`test`/`rest` 0.75, `tls`/`tld` 0.67) while adding
  nothing on the cases that should match. Full evidence table in
  `research.md#alternatives-considered-and-rejected`.
- **Caught by**: the user, at the walkthrough, proposing partial matching with a
  reduced score and asking whether a word-distance algorithm existed. It does; it is
  the wrong one for this corpus, and the right one turned out to be simpler.
- **Prior art checked**: the user then asked whether this problem was already solved.
  It is, and the design was reconciled against it. Prefix matching is standard (edge
  n-grams in Lucene/Elasticsearch) and the short-term length guard is standard
  (Elasticsearch `fuzziness: AUTO`, which allows 0 edits below 3 characters). The one
  genuine departure is scoring a prefix match *proportionally*: Lucene scores prefix
  matches with a constant, because BM25 and IDF discriminate downstream, and this
  project has neither. Stemming is the established answer to the `route`/`routing` gap
  and is knowingly declined. See § External references.
- **Cost accepted**: one more tunable (`minPrefixLen`), and a contradiction with the
  spec's Non-Goal on query expansion that must be amended before the plan is
  internally consistent. Recorded as item 0 in § Open Questions.
- **Supersedes**: the original exact-matching decision, below.

### Superseded: tag matching is exact, body matching stays substring (architecture)
> Superseded at the walkthrough by prefix partial credit, above. Retained because its
> rejection of *unanchored substring* matching still stands and still explains why the
> replacement carries two guards.
- **Decision**: A query term matches a tag only on equality after lower-casing,
  while body matching remains a case-insensitive substring count as it is today.
- **Rationale**: Substring tag matching would let `go` hit a `golang` tag, and more
  damagingly would let any short query term hit every tag containing it, firing the
  large tag weight almost constantly and destroying its value as a discriminating
  signal. Body substring matching is existing behaviour and changing it is not in
  scope.
- **Rejected**: Substring matching on tags for symmetry with the body; stemming or
  prefix matching, which the spec lists under Non-Goals as query expansion.
- **Consequence, addressed separately**: exact matching moves the whole burden of
  vocabulary to capture time. See "Exact matching is mitigated at capture time, with
  explicit tag-form rules" below for the instructions that carry that burden.

### Body term counting excludes the frontmatter block (components)
- **Decision**: Once tags are parsed out, per-term occurrence counting runs against
  the entry **body**, not the raw file bytes, so a tag is not additionally counted
  as a prose occurrence of itself.
- **Rationale**: Counting the frontmatter as prose would double-count every tag,
  adding an uncontrolled `damp(1)`-sized bonus on top of the deliberate tag weight
  and making the tag weight's real magnitude impossible to reason about or tune.
- **Rejected**: Counting the raw bytes for simplicity. It silently entangles the
  two signals the spec asks us to weigh against each other.

### The knowledge layer owns the tag filter; a store may pre-apply it (components, revised at walkthrough)
- **Decision**: The tag filter is **enforced** in `knowledge.Set.Search`, in the same
  post-merge loop as the always-applied exclusion and the cutoff. A store may
  additionally pre-apply `opts.Tags` during its walk, but only as an optimisation:
  correctness does not depend on it doing so.
- **Rationale**: Once ranking moved above the store interface, a provider is no longer
  trusted with anything that decides what the user sees. The filter is absolute
  ("however well it would otherwise score"), so a provider that ignored or
  mis-implemented it would leak entries the user explicitly excluded. Enforcing above
  the interface costs almost nothing, because the hit already carries its tags. The
  pre-apply path is kept because skipping a document during the walk is genuinely
  cheaper than reporting it, and a remote provider may be able to push the filter to
  its backend.
- **Rejected**: (a) Filtering only in the store, the originally drafted decision. It
  makes every provider individually responsible for a user-visible guarantee. (b)
  Filtering only in the knowledge layer, with no store fast path. Correct, but it gives
  up a free optimisation and forbids server-side filtering for remote providers.
- **Supersedes**: the earlier "the tag filter applies in the store, the cutoff in the
  knowledge layer" split, which predates the evidence-reporting revision.
- **Testable consequence**: a fake store that ignores `opts.Tags` must still not leak
  an untagged entry into results. That assertion is what pins the knowledge layer as
  the authority, and it is called out in Phase 2.3.

### Tag narrowing rides on the existing Selector value (components)
- **Decision**: Carry the requested tags on `knowledge.Selector` alongside `Tier`
  and `Filter`, rather than adding a separate parameter to `Set.Search`.
- **Rationale**: The spec's technical steer is that narrowing a search stays one
  idea. One value carrying every narrowing axis keeps the CLI, the knowledge layer
  and the store agreeing on what "scope" means.
- **Rejected**: A separate `tags []string` parameter. It would make `Selector` a
  partial description of scope and invite the two to drift.
- **Note**: The tag list narrows *entries* while `Tier`/`Filter` narrow *stores*, so
  they are applied at different points despite travelling together. `Selector.covers`
  stays a store-membership test and must not learn about tags.

### `--tag` is AND across repeated values (data_structures)
- **Decision**: Repeating `--tag` narrows further: an entry must carry **every**
  listed tag to be returned.
- **Rationale**: It matches the meaning of narrowing, which is what the spec calls
  this ("Users can narrow a search to tagged entries"). AND is also the safe default
  because it is unambiguous to widen later (adding an explicit any-of mode) but a
  breaking change to tighten.
- **Rejected**: OR across repeated tags. It would make each extra `--tag` return
  *more* results, which reads as the opposite of narrowing, and it duplicates what
  the ranked-OR scoring already does for query terms.
- **Note**: `--filter` is OR because it names alternative stores; `--tag` is AND
  because it names required properties of one entry. Worth stating in the docs so
  the asymmetry does not read as an inconsistency.

### `knowledge write` payload is unchanged (data_structures)
- **Decision**: Tags travel inside the entry body's frontmatter, in the file the
  skill already stages, rather than as a new field on the `knowledge write --data`
  payload.
- **Rationale**: The entry file is the single source of truth for its own tags. A
  separate payload field would create two places tags could live and a question
  about which wins on a re-write. It also leaves the write command, its schema, and
  the propose-then-confirm contract completely untouched.
- **Rejected**: A `"tags": [...]` field on the write payload with the CLI rendering
  the frontmatter. Tidier for the agent, but it makes the CLI a partial editor of
  entry content, and an update that round-tripped through it would need
  frontmatter-preserving rewrite logic the spec does not ask for.

### Score's meaning changes under a field that keeps its name (data_structures)
- **Decision**: Keep the `Score` field name and `float64` type; change its
  documented meaning from "sum of occurrences" to the composite score, and update
  the doc comment at the type.
- **Rationale**: The field is already `float64` and already means "rank by this,
  descending". Renaming it would churn every consumer for no gain in clarity.
- **Rejected**: A new field name alongside a deprecated `Score`. There is exactly
  one consumer of this type outside the package; a deprecation cycle buys nothing.

### Two frontmatter readers rather than one shared one (implementation_detail)
- **Decision**: Ship a second, knowledge-specific frontmatter reader instead of
  generalising the existing artifact one, and document at the new reader why it is
  not the other one.
- **Rationale**: The shared part is roughly fifteen lines of block detection; the
  divergent part is the schema and its validation, which is the part that carries
  the risk. Generalising would mean either forcing `created_date`/`status` onto
  knowledge entries (forbidden by the no-migration constraint) or loosening artifact
  validation so a malformed spec passes silently.
- **Rejected**: Extracting a shared block-splitting helper used by both. Defensible,
  and a reasonable refactor to take if the implementer finds the duplication
  larger than expected, but it adds an indirection between two callers with
  incompatible schemas for a small saving. Flagged rather than forbidden.
- **Note**: This is the one place this plan knowingly accepts near-duplication.
  The doc comment is the mitigation, because a future contributor will otherwise
  try to consolidate them.

### The checksum keeps covering the raw bytes (implementation_detail)
- **Decision**: The content checksum continues to be computed over the file's exact
  raw bytes, frontmatter included, even though term counting now runs against the
  body only.
- **Rationale**: The checksum identifies the *file* for exact-byte de-duplication.
  Two entries that differ only in their tags are genuinely different entries and
  must not collapse into one candidate during de-dup.
- **Rejected**: Checksumming the body alone so tagging an entry does not change its
  identity. It would silently merge a tagged and an untagged copy of the same prose
  during consolidation, hiding the tagged one.

### Existing expected values are recomputed by hand, not regenerated (implementation_detail)
- **Decision**: Every existing test oracle this change invalidates is recomputed by
  hand from the fixture and the documented formula.
- **Rationale**: The repo's tests deliberately use hand-maintained oracles that are
  independent of the subject under test. Regenerating them from the new
  implementation would make the tests assert only that the code does what it does.
- **Rejected**: Recording the new implementation's output as the expected value.

### Documentation phase is sequenced last (dependencies)
- **Decision**: The `docs` repo change is sequenced after the engine and CLI
  behaviour is final, rather than run in parallel.
- **Rationale**: The page documents observable behaviour, including the ranking
  factors and the exact flag syntax, and the spec explicitly expects the weights to
  be tuned during implementation. Publishing first would describe numbers that then
  change.
- **Rejected**: Writing the docs in parallel to shorten the critical path. The two
  repos are independent enough that it is technically possible, but it trades a
  small schedule gain for a real risk of shipping a page that is wrong on the day
  it lands.

### No new third-party dependency (dependencies)
- **Decision**: Implement YAML frontmatter parsing and the damping curve with
  `gopkg.in/yaml.v3` and the standard library's `math`, both already in the module.
- **Rationale**: The parse is a leading fenced block and a single list-valued key;
  the damping curve is one logarithm. Neither warrants a dependency, and the repo
  already proves the yaml.v3 approach works for exactly this shape.
- **Rejected**: A dedicated frontmatter library, or a text-search library that
  brings its own scoring. The latter would import a general relevance model the
  spec's Non-Goals rule out.

### The benchmark runs against a controlled fixture, not the live knowledge base (testing_approach)
- **Decision**: Success metric 2 (the four observed queries) is asserted against a
  fixture built in a test-owned temp directory that reconstructs the measured
  scenario, not against `.spektacular/knowledge` on disk.
- **Rationale**: Demonstrated during research. Adding one real entry to the live
  corpus (`decisions/ranked-or-over-boolean-and.md`, which quotes all four benchmark
  queries verbatim) moved the target entry from rank 1 to rank 2 on `http handler`.
  A test bound to a directory that grows every time the project writes knowledge
  would flap for reasons unrelated to the code. It also matches the repo convention
  that a test owns the filesystem it reads.
- **Rejected**: Pointing the benchmark at the real knowledge directory so it measures
  "the real thing". It would be a monitor of the corpus, not a test of the code.

### Metrics 3 and 4 are classified manual, with automatable preconditions split out (testing_approach)
- **Decision**: Classify "newly captured entries carry tags" and "the tag vocabulary
  converges" as *Manual — captured in the implementation test plan*, while asserting
  their mechanical preconditions behaviourally (rendered skill prose instructs tag
  proposal; tags normalise to lower case; the vocabulary listing orders by use).
- **Rationale**: Both metrics are trends across many agent-driven capture events over
  time. Neither is observable in a single test run. But each rests on a mechanical
  property that is testable, and leaving those untested would mean the manual
  observation could fail for a reason a test should have caught.
- **Rejected**: Classifying them wholly behavioural (not achievable), or wholly
  manual with no automated support (leaves a preventable failure mode uncovered).

### No end-to-end test of the confirmation contract (testing_approach)
- **Decision**: Do not attempt an automated test that an agent asks for confirmation
  before writing.
- **Rationale**: The contract is enforced by skill prose, and the repo has no harness
  that drives an agent through a confirmation gate. The existing precedent is to
  assert on rendered prose content instead.
- **Rejected**: Adding a CLI-level guard that refuses an unconfirmed write so it
  could be tested. That would change the write contract, which the spec does not ask
  for, and confirmation is a property of the conversation rather than of the call.

### Ranking ships before tags, as its own milestone (milestones)
- **Decision**: Milestone 1 delivers ranked OR, coverage scaling, damping and the
  relative cutoff with no tags at all; milestone 2 adds tags on top.
- **Rationale**: They are separable and each is independently valuable. Ranked OR
  alone already fixes the "returns nothing" and "long tangential document wins"
  failures for the entire existing corpus, without a single entry being edited.
  Shipping them together would make a large behavioural change land in one step with
  no intermediate point at which the existing tests are known good.
- **Rejected**: One combined engine milestone. Faster on paper, but it merges the
  riskiest change (invalidating existing score oracles) with the newest one (tag
  parsing) into a single validation point.

### The ranking knowledge entry lands in milestone 4, not milestone 1 (milestones)
- **Decision**: Requirement 13 (record the ranking design as a knowledge entry) is
  delivered with the documentation milestone rather than with the ranking milestone.
- **Rationale**: The spec's own technical approach says the entry is "written once
  the design settles", and the spec expects the weights to be tuned during
  implementation. Writing it in milestone 1 would describe numbers that then change.
  It also makes the entry's own retrievability a live end-to-end exercise of the
  finished feature.
- **Rejected**: Writing it in milestone 1 so the design is captured while fresh. The
  research and assumption files already capture it durably in the interim.

### Four milestones rather than three (milestones)
- **Decision**: Keep documentation as its own milestone rather than folding it into
  milestone 3.
- **Rationale**: It is the only milestone that lands in the `docs` repo, it has its
  own build and typecheck validation, and the published page is currently *wrong*
  the moment milestone 1 ships, which is a distinct user-visible harm worth tracking
  separately rather than burying inside a capture-flow milestone.
- **Rejected**: Three milestones with docs folded in. It would hide a cross-repo
  deliverable inside a milestone whose validation point is about confirmation
  prompts.

### Phase 1.1 is a behaviour-preserving refactor with no user-visible change (phases)
- **Decision**: Spend a phase extracting the scorer before changing any behaviour,
  with the explicit acceptance criterion that every existing test passes untouched.
- **Rationale**: It gives a clean checkpoint where the refactor is proven correct
  before the semantics change, so a later failure is unambiguously attributable to the
  ranking change rather than to the extraction. The milestones step permits a purely
  internal phase provided it says so.
- **Rejected**: Folding the extraction into Phase 1.2. It would mean landing a
  refactor and a behavioural change in one diff, with the old tests already rewritten,
  and no point at which either is independently verified.

### Cutoff is computed from hits surviving the category exclusion (phases)
- **Decision**: In Phase 1.3, compute the best score from hits that survive the
  always-applied category exclusion, not from the raw merged list.
- **Rationale**: An always-applied entry scoring highest would otherwise set the
  cutoff bar and then be dropped, silently raising the effective threshold for every
  real result. The two filters are adjacent in the same loop, so the ordering is easy
  to get wrong without noticing.
- **Rejected**: Computing the best score before the exclusion, which is the more
  obvious reading of "relative to the strongest hit".

### `--tag` gets a search-specific flag schema (phases)
- **Decision**: Give `knowledge search` its own flags map rather than adding `tag` to
  the shared `knowledgeNarrowingFlags`.
- **Rationale**: That map is advertised by all four fan-out commands. Adding `tag`
  there would publish a flag on `list`, `conventions` and `always-applied` that they
  do not accept, which is exactly the kind of stale agent-facing contract the repo's
  error-message convention exists to prevent.
- **Rejected**: Adding it to the shared map for brevity, and accepting the
  over-advertisement.

### Tag-filter refusal is deferred to Phase 3.1 if ordering slips (phases)
- **Decision**: Refusing a `--tag` that names a tag no entry carries depends on the
  vocabulary lister from Phase 3.1. If 3.1 has not landed, Phase 2.3 returns an empty
  result and the refusal is added in 3.1.
- **Rationale**: The repo's convention requires a refusal to name a runnable next
  action, and the only useful next action here is `knowledge tags`. Shipping a refusal
  that cannot name it would be worse than shipping none.
- **Rejected**: Reordering so 3.1 precedes 2.3. It would push the vocabulary lister
  into the ranking milestone, where it has no other purpose.

### An unterminated leading block is "no frontmatter", not an error (open_questions)
- **Decision**: If an entry opens with `---` but the block is never closed, or the
  YAML inside fails to parse, the reader returns no tags and the **raw bytes
  unchanged** as the body, rather than returning an error.
- **Rationale**: The workflow-artifact reader errors in this situation because a
  malformed spec genuinely is malformed. A knowledge entry has no required
  frontmatter at all, so the safe interpretation is that the `---` was prose (a
  horizontal rule) and the entry simply has no tags. Erroring would make a knowledge
  entry unsearchable because of a formatting accident, which is a worse failure than
  missing a tag. Verified low-risk: no entry in either registered repo's store
  currently opens with `---`.
- **Rejected**: Erroring, mirroring the artifact reader. It would let one malformed
  entry break a search across a whole store.
- **Note**: A malformed block should still be *reported* somewhere the audit flow can
  see it, so it is not silently invisible forever; it must not fail the search.

### Question 3 resolved during planning rather than parked (open_questions)
- **Decision**: Inspected every `.md` entry in both registered repos' knowledge stores
  for a leading `---` instead of deferring the check to implementation.
- **Rationale**: The step's rule is that anything answerable by reading the code or
  running a quick experiment must be resolved now. This was a two-command check.
- **Rejected**: Parking it as an open question. It would have sent the implementer to
  do work that took under a minute here.

### Tag-form rules survive prefix matching, but shrink (walkthrough, revised twice)
- **Decision**: The capture flow keeps tag-form instructions, reduced from three rules
  to three narrower ones: one form per subject (do **not** carry `apple` and `apples`,
  since prefix matching relates them); add a second tag only where a form differs by
  more than its ending (`route` beside `routing`); and never collapse distinct subjects
  into one tag (`https` is not `http`). The audit inherits them, and additionally
  reports a tag made redundant by prefix matching as removable.
- **Rationale**: Prefix partial credit converted the largest of these from guidance
  into mechanism, so the guidance had to shrink or it would actively mislead: telling
  an agent to carry both `apple` and `apples` is now wrong advice that clutters the
  vocabulary. What the mechanism cannot do is judge whether two subjects are the same,
  which is what the remaining rules cover.
- **Rejected**: dropping the tag-form guidance entirely. Prefix matching is positional
  and cannot tell `route` from `routing`, nor `https` from `http` as *subjects*.
- **History**: first added at the walkthrough when the user raised plurals and
  derivatives under exact matching; rewritten in the same walkthrough once prefix
  partial credit was adopted, which made two of the three original rules obsolete.

### Superseded: exact matching is mitigated at capture time, with explicit tag-form rules (walkthrough)
- **Decision**: Because tag matching is exact and query expansion is a spec Non-Goal,
  the capture flow carries explicit, load-bearing instructions on choosing tag *forms*:
  (a) carry every form genuinely likely to be searched, tagging both `test` and
  `testing` rather than picking one; (b) never derive a variant mechanically, since a
  trailing `s` is not evidence of a plural; (c) an explicit boundary on the
  prefer-an-existing-tag rule, so a distinct subject is never collapsed into a
  similar-looking existing tag. The same rules bind the audit intent, and the
  user-facing documentation states them too.
- **Rationale**: Exact matching was already decided and is right (substring matching
  would let any short term hit every tag containing it, firing the tag weight
  constantly). But it pushes the entire burden of vocabulary onto the moment of
  capture, and the plan as drafted said nothing about it. Left unstated, the two
  failure modes are silent: an entry tagged `apple` is simply never found by `apples`,
  and an over-eager convergence rule merges `https` into `http` and loses the
  distinction. Neither surfaces as an error; both surface as a search that quietly
  returns the wrong thing, which is the failure this whole feature exists to fix.
- **The tension this resolves**: the convergence rule was drafted as "propose `http`
  rather than a near-duplicate such as `HTTP` or `http-api`". Read naively, that
  licenses collapsing `https` into `http`. It must not: `HTTP` and `http-api` are
  spellings of one subject, `https` is a different subject. The rule now carries that
  boundary explicitly rather than relying on the reader to infer it.
- **Rejected**: (a) Adding stemming or plural handling to the matcher. It is an
  explicit spec Non-Goal, and it would make results harder to explain and reintroduce
  the guessing-at-vocabulary problem this feature replaces. (b) A canonical-form rule
  ("always tag the singular"). Cheaper to state, but it is exactly the mechanical
  derivation that breaks `https`, `dns`, `cors` and `kubernetes`, and it guarantees the
  plural query misses. (c) Leaving it to agent judgement without instruction, which is
  what the plan did before this walkthrough.
- **Caught by**: the user, at the walkthrough, on the plural and derivative cases
  (`apple`/`apples`, `http`/`https`) after the exact-matching decision was read out.
- **Residual risk accepted**: this is prose guidance to an agent, not a mechanism.
  Nothing enforces it, and a poorly chosen tag set still fails silently. The audit
  intent is the backstop. If tag-form mistakes turn out to be common in practice, a
  mechanical aid (warning on a tag that is a strict prefix of another, say) is a
  follow-up, not part of this plan.

### The docs publish the algorithm and the frontmatter contract in full (walkthrough, reverses an earlier call)
- **Decision**: The user-facing and maintainer documentation state the scoring
  algorithm in full, including the `damp` / `termScore` / `coverage` / `score` formula,
  the shipped values of `TAG_WEIGHT`, `COVERAGE_EXPONENT` and `CUTOFF_FRACTION`, and a
  worked two-entry example carried through to a number. The frontmatter contract is
  likewise documented in full: block placement and fencing, both YAML list forms,
  `tags` as the only key read, lower-casing and de-duplication, exact tag matching, the
  block excluded from body text, optionality, and malformed-block tolerance.
- **Rationale**: The user's explicit direction at the walkthrough. It is also the right
  call for this audience: the primary readers are engineers and coding agents deciding
  how to write a retrievable entry, and "tags count for more than body mentions" does
  not let anyone predict a result, whereas the formula does. A worked example is what
  makes the formula legible rather than decorative.
- **Reverses**: The earlier draft deliberately kept numbers off the published page,
  with the stated reason "no numbers, so tuning the weights does not date the page".
- **Cost accepted**: the page is now coupled to the code. Tuning a constant becomes a
  four-file change (`internal/knowledge/ranking.go` plus three documents that quote it).
  The mitigation is to name the constants in the code as the single source of truth, say
  so in each document, and add the cross-check to the acceptance criteria of Phases 4.1
  and 4.2 and to the knowledge entry in 4.4. The plan already sequences documentation
  after tuning, which limits how often this bites.
- **Rejected**: (a) Prose-only description of the factors, the earlier draft. Cheaper to
  maintain, but a reader cannot predict or debug a ranking from it. (b) Publishing the
  formula but not the constants. Halfway house: it still dates when the *shape* changes,
  and a formula with unnamed weights cannot be worked through an example, which is the
  part that makes it land.

### The ranking sections become two new bands rather than an extension (walkthrough)
- **Decision**: On the published page, "How a search is ranked" and "Tagging an entry"
  become two new top-level `Section` bands after "The lifecycle of an entry", the first
  `surface` and the second plain.
- **Rationale**: The content grew from a paragraph into a full algorithm reference plus
  a full frontmatter reference once the formula went in, which is too much to nest
  inside a lifecycle narrative. Adding exactly **two** bands in that order also preserves
  the page's existing plain/surface alternation with no changes further down, where one
  band would force every band below it to flip.
- **Rejected**: Extending the existing lifecycle section in place, which is what the
  earlier draft specified and what a smaller change would have warranted. Also rejected:
  a single combined band, which is both a worse split of two topics and more churn.

### The storage-backend reference gets its own phase (walkthrough)
- **Decision**: Add Phase 4.3 in the `docs` repo to correct
  `src/pages/extending.mdx`, rather than folding the change into Phase 4.2 or leaving
  it out.
- **Rationale**: Once ranking moved above the store interface, the contract a provider
  author builds against changed, and that contract is published on a page the original
  plan did not touch at all. It is a different page, a different audience (backend
  implementers, not knowledge-base users) and a different acceptance test, so folding
  it into the user-docs phase would bury a contract change inside a copy edit.
- **Rejected**: (a) Folding it into Phase 4.2. (b) Leaving the page alone, on the
  grounds that the interface is internal. It is not internal in any useful sense: it
  is published, with a worked example, as the way to add a backend.
- **Caught by**: the user, at the walkthrough, asking whether the docs would be updated
  too. The first revision fixed the architecture and missed its own documentation
  consequence.

### Pre-existing drift on the backend reference is fixed in passing, not tracked separately (walkthrough)
- **Decision**: Phase 4.3 also corrects the fields that were already wrong on
  `docs:src/pages/extending.mdx` before this work (`Excerpt` vs `Excerpts`, and the
  missing `Title`, `Category` and `Checksum`), rather than fixing only what this plan
  changes.
- **Rationale**: The phase is rewriting that exact struct anyway. Correcting half of it
  and knowingly leaving the other half wrong would ship a second inaccurate version and
  waste the only pass anyone is going to make over it.
- **Rejected**: Fixing only the fields this plan touches and raising the rest as a
  separate issue. Defensible on scope-purity grounds, but it costs more to track than
  to fix.
- **Not addressed**: *why* the page drifted. Nothing ties it to the Go source, so
  nothing will catch the next drift either. Closing that gap is genuinely out of scope
  here and is flagged in the phase notes as a follow-up.

### `--tag` lands on `search` only (out_of_scope)
- **Decision**: Add the `--tag` narrowing option to `knowledge search` and not to
  `list`, `conventions` or `always-applied`.
- **Rationale**: The spec's requirement is about narrowing a *search*. Adding it to
  the other fan-out commands is mechanical but has no demonstrated need, and
  `conventions`/`always-applied` cover categories that are excluded from tagging
  anyway.
- **Rejected**: Adding it to all four for symmetry, via the existing shared flag loop.
  It would advertise a filter on commands where it is either meaningless or
  vacuous.

### Always-applied categories are excluded from the tag vocabulary (out_of_scope)
- **Decision**: The `knowledge tags` listing excludes conventions and glossary, the
  same categories search excludes.
- **Rationale**: Consistency with search is what makes the vocabulary useful: the
  capture flow proposes tags for entries that will be *retrieved* by them, and an
  always-applied entry is never retrieved by search. Including their tags would
  suggest vocabulary that can never pay off.
- **Rejected**: Listing every tag in the store regardless of category. Simpler, but it
  would seed the proposal step with tags that cannot affect retrieval.

## Rehydration cues

To rebuild this context from cold, in order:

1. `go run . repo list` — the two roots. **Never assume the working directory is a
   repo's code.**
2. `go run . knowledge always-applied --tier repo --filter spektacular --filter docs`
   — both repos' binding conventions, including the docs site's MDX authoring
   rules and the CLI's error-message and passing-tests rules.
3. `go run . knowledge search "knowledge search ranking"` → read
   `decisions/ranked-or-over-boolean-and.md` in full. It is **binding**, and the
   code at `internal/store/search.go:70-75` has not caught up to it.
4. Read `.spektacular/context.md` — the measured baseline and the three-run field
   evidence. Do not re-derive it.
5. Read, in this order: `internal/store/search.go`, `internal/store/store.go`,
   `internal/knowledge/set.go`, `cmd/knowledge.go`.
6. Read `templates/skills/workflows/spek-knowledge/SKILL.md` — the whole file, 83
   lines. It is the source of truth; `.claude/` and `.bob/` copies are generated.
7. `go run . skill spawn-planning-agents` — the research fan-out guidance, if more
   codebase research is needed.
8. Re-run the scoring prototype to re-validate weights before tuning: the script
   is reproduced in the plan's research section and rebuilt in ~60 lines of Python
   over `.spektacular/knowledge`.
9. After any template edit: `go run . init claude && go run . init bob`, then
   commit the regenerated `.claude/` and `.bob/` skill files.
10. `go test ./...` in the spektacular root; `npm run build && make check` in the
    docs root.
