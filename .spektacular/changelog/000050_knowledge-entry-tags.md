---
created_date: "2026-09-10"
status: completed
closed_date: "2026-09-10"
---

# Knowledge entry tags and search ranking

## What was built

Knowledge entries can now declare what they are about, and knowledge search
changed from all-or-nothing keyword matching to genuine ranking.

**Search ranks instead of excluding.** A document no longer has to contain every
word of a query. It is returned if it carries evidence for any of them, and
scored as `sum(tagWeight * tagAffinity(term) + damp(count)) * coverage^2`, with
`damp(n) = 1 + log2(n)` so repeated mentions give diminishing returns and
`coverage` scaling by the share of the query an entry actually covers. After every
store's hits are merged, anything scoring below a quarter of the best hit is
dropped, so a loosely related entry surfaces when it is the only thing available
and disappears once something genuinely relevant exists.

**Entries carry optional tags.** An entry may open with a YAML frontmatter block
declaring `tags`. A tag match is weighted far above a prose mention, so an entry
tagged `go, http` is found by searching `go http router` even when its body
contains neither word. Tag matching is exact at full strength and prefix-related
at proportional partial credit, so `http` finds a `https` tag at four fifths and
`apple` and `apples` find each other, with two guards: the match is anchored at
the start, and the shorter string must reach four characters. Searches can be
narrowed with a repeatable `--tag`, which is an absolute filter rather than a
ranking hint.

**Tags get applied without anyone designing a taxonomy.** A new `knowledge tags`
command reports the vocabulary already in use, most-used first. The
`spek-knowledge` capture flow loads it and proposes tags alongside the tier,
store, path and body, inside the single confirmation gate it already had. A new
audit intent reviews existing entries the same way, reporting tags an entry's
content does not bear out and proposing ones it is missing, one entry at a time.

**Ranking is computed in one place.** A storage backend now reports the evidence
it found and never computes a score. Tokenizing, scoring, merging, sorting and
the cutoff all happen once in the knowledge layer, so a second backend cannot
introduce a second ranking scale.

## Why it matters

An entry was previously found only if it shared vocabulary with whatever query an
agent happened to invent at the time, and that query is not stable between runs.
The measured case: across three real planning runs against one unchanged spec, an
agent searched `http endpoints go`, then `http handler`, then
`http endpoints handler routing`, and a fourth plausible phrasing, `api`,
returned nothing at all. A long early-concept document that mentioned "Go", "API"
and "http" in passing beat the document actually about routing on three of four
queries. So an agent read a confident-looking but irrelevant top hit and planned
without knowledge the team had already written down.

All four of those queries now return the on-topic entry first. The `api` case is
the sharpest: it now retrieves on tags alone, with the word appearing nowhere in
the entry's prose.

Nothing has to change for this to take effect. Untagged entries remain valid and
fully searchable, there is no migration, no backfill, and no command to run.

## Deviations from the plan

- **The store still skips documents with no evidence at all.** Taken literally the
  plan removed all exclusion from the storage walk, which would emit a hit for
  every file in the tree including binaries. A document matching no term is not a
  partial match, so one guard remains. When tags landed, that guard was extended
  to count declared tags as evidence, because an entry tagged `go` whose prose
  never says "go" has to reach the ranking layer to be scored at all.
- **Three pre-existing tests were relocated rather than left untouched.** The
  plan's Phase 1.1 required both that every existing test pass unchanged and that
  scoring leave the storage layer entirely; three tests asserted scoring and
  boolean-AND matching from a bare store, so both could not hold. They moved to
  the knowledge layer carrying every fixture and every hand-computed expected
  value verbatim.
- **Two Phase 4.2 acceptance criteria were stale and were corrected.** They
  asserted that tag matching is exact and that a singular does not retrieve a
  plural, both superseded when the plan walkthrough introduced prefix partial
  credit, and both contradicted by that same phase's own content outline.
- **An unknown `--tag` is refused rather than silently returning nothing**,
  resolving a deferral the plan left conditional. The sibling `--filter` flag
  already refuses an unknown store for the same reason.
- **The parser reports a malformed frontmatter block, the scan swallows it.** The
  plan said both that a malformed block must be reported and that it must not
  error; both now hold at different layers, so a search can never fail because one
  entry has a formatting accident.

## Also fixed in passing

- The published storage-backend reference had been wrong independently of this
  work: it showed `Excerpt string` where the code has `Excerpts []string`, and
  omitted `Title`, `Category` and `Checksum`. Corrected in the same pass.
- `AGENTS.md` had been stale since commit `a76e04c`, which edited its template
  without re-running init, so the committed file never received the paragraph
  declaring knowledge entries binding over the code. Regenerating for this work
  picked it up.

## Known follow-ups

- Nothing ties the published backend reference to the Go source, which is how it
  drifted. The same drift will recur the next time the `Store` contract changes.
- The tag-form guidance tells agents not to carry both a singular and a plural
  because prefix matching relates them. That reasoning fails for any singular
  under four characters: `tag` does not retrieve a `tags` tag, and the same
  applies to `api`, `job` and `key`.
- No metrics or observability surface for the ranking. Tuning is currently done
  against the four-query benchmark and the acceptance criteria, by hand.
