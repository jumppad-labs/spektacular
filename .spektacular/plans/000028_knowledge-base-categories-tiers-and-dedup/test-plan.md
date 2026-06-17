# Implementation Test Plan — 000028_knowledge-base-categories-tiers-and-dedup

These procedures cover the success metrics that are judgement- or
documentation-based and so cannot be asserted by an automated test. The
deterministic halves of these metrics (path-derived category labels, the
`categories`/`always-applied` command contracts, byte-identical checksum
collapse, the one-place re-tier coupling, the preserved `conventions` output)
**are** covered by the automated suite (`internal/knowledge`, `internal/store`,
`internal/project`, `cmd`) and are not repeated here.

In this repository the CLI is invoked with `go run .`; in an installed project it
is `spektacular`. Substitute as appropriate. Each procedure assumes a
Spektacular-initialised project with at least a `project` scope, and (for
multi-scope checks) a second `team` scope configured under `knowledge.sources`.

---

## Metric 1 — New entries are filed in the correct category without correction

**What to measure:** Over real use, contributions land in the category whose
purpose matches, and over-long content is kept out of the glossary — with no
human correction needed.

**How:**
1. Run `go run . knowledge categories` and confirm it returns all six categories,
   each with a non-empty purpose, boundary, tier, and entry shape. (This is the
   routing input the assistant uses.)
2. In a session, invoke the `spek-knowledge` skill's contribute flow with a
   handful of representative entries, one clearly belonging to each category — e.g.
   a standing rule ("always run gofmt before commit"), a defined term, a
   structural fact, a trap, an empirical finding, and a recorded decision.
3. For each, confirm the assistant proposes the matching category (path
   `<category>/<slug>.md`) before writing.
4. Submit a deliberately long, multi-paragraph entry and confirm the assistant
   steers it **out of** `glossary` toward architecture/learnings/decisions.

**Expected result:** Each entry is proposed in the category whose purpose matches;
the long entry is not routed to `glossary`. Mis-routing requiring correction is
rare across the sample (target: the assistant routes correctly without the user
re-directing it).

**Who / when:** Author or reviewer, during pre-release manual review and
periodically as real contributions accrue.

---

## Metric 2 — Lookups return consolidated, cited answers (merge & citation quality)

**What to measure:** The consolidation sub-agent merges differently-worded
equivalents into single points, and every point cites the source(s) it came from.
(The mechanical half — byte-identical copies collapsing by checksum, and the
output being the consolidated answer rather than the raw hit list — is automated.)

**How:**
1. Seed two entries that say the same thing in different words across two scopes,
   e.g. `project: gotchas/backoff.md` = "add jitter to retries" and
   `team: gotchas/backoff.md` = "randomize backoff intervals".
2. Run the `spek-knowledge` lookup flow for a query that matches both (e.g.
   "retry backoff").
3. Inspect the answer presented to the user.

**Expected result:** A single merged point covering the shared knowledge, citing
**both** sources (scope + path). The raw per-source hit list is not presented as
the result. Citations are accurate (each cited source actually contains the point).

**Who / when:** Reviewer, pre-release and after any change to the lookup prose.

---

## Metric 3 — The always-applied tier stays compact

**What to measure:** The glossary holds short term-and-gloss entries, not
essay-length content, so the always-applied set loaded on every task stays small.

**How:**
1. Run `go run . knowledge always-applied` and review the returned `glossary`
   entries.
2. Spot-check entry lengths: each glossary entry should be a term and a short
   gloss (roughly one or two sentences), per the category's declared entry shape.

**Expected result:** No glossary entry is essay-length; long material lives in
architecture/learnings/decisions instead. This is an observed property over time,
enforced by the definition plus the contribute-flow steering (Metric 1), not a
hard guard.

**Who / when:** Maintainer, periodically and during pre-release review.

---

## Metric 4 — Source disagreements are visible

**What to measure:** When two sources genuinely contradict each other, the lookup
surfaces the conflict rather than silently picking one or averaging.

**How:**
1. Seed a genuine contradiction across scopes, e.g.
   `project: conventions/retry.md` is excluded from search (always-applied), so
   instead use a looked-up category: `project: learnings/retry.md` = "never retry
   network calls" and `team: learnings/retry.md` = "always retry network calls".
2. Run the lookup flow for a matching query (e.g. "retry network calls").

**Expected result:** The answer presents the disagreement explicitly, naming both
sources (scope + path), rather than returning only one side or a blended claim.
(Note: a refinement where one scope is merely more specific should instead be
resolved in favour of the most-specific scope — `project` over `team` over
`global` — with the override noted; confirm that case separately by seeding a
general vs. specific pair.)

**Who / when:** Reviewer, pre-release and after any change to the consolidation
prose.

---

## Metric 5 — Maintainers can explain the knowledge base from the documentation alone

**What to measure:** A reader new to the project can describe how the knowledge
base works using `docs/knowledge-base.md` without reading the source.

**How:**
1. Give `docs/knowledge-base.md` to someone unfamiliar with the implementation.
2. Ask them to explain, from the document alone: the six categories and how to
   tell them apart; the two retrieval tiers and that tier is declared once; that
   search results carry a category label; how de-duplication and consolidation
   work; why the de-dup layer is exact rather than fuzzy; and how layered source
   precedence resolves overlaps and surfaces conflicts.

**Expected result:** The reader can cover all of the above accurately from the
document, with no need to consult the code.

**Who / when:** A maintainer or reviewer not involved in the implementation,
pre-release.
