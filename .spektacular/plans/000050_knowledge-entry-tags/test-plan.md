---
created_date: "2026-09-10"
status: completed
closed_date: "2026-09-10"
---

# Manual test plan: 000050_knowledge-entry-tags

Two of the spec's four success metrics are trends across many capture events over
time and cannot be asserted in a single run. Both are recorded here with a
procedure grounded in the shipped implementation.

The other two are covered by automated behavioural tests and are **not** repeated
here:

- *"The `api` case reverses"* and *"the observed-query benchmark goes from 0 to
  4"* are pinned by `internal/knowledge/benchmark_test.go`
  (`TestBenchmark_ObservedQueriesRankTheOnTopicEntryFirst` and
  `TestBenchmark_TagOnlyRetrieval`). Run `go test ./internal/knowledge/` to check
  them; they need no manual procedure.

---

## Metric 3: Newly captured entries carry tags

**What to measure.** Entries written through the `spek-knowledge` capture flow
after this ships carry at least one tag, without anyone having designed a
taxonomy first.

**Pass condition.** Of the entries added since this feature shipped, **at least
90% carry one or more tags**, and any untagged entry is one a human deliberately
chose to leave untagged rather than one the flow forgot to propose tags for.

**How.** In a project that has been using the skill for a while, from the project
root:

```bash
# every entry in scope, and the vocabulary those entries produce
go run . knowledge list --tier all
go run . knowledge tags --tier all
```

Then, for entries added since the feature shipped, read a sample back and check
for a frontmatter block:

```bash
go run . knowledge read --data '{"tier":"repo","name":"<store>","path":"<category>/<slug>.md"}' | head -5
```

An entry that carries tags opens with `---`, a `tags:` line, and a closing `---`.
Cross-check the count against the store's own history:

```bash
git -C <store-root> log --diff-filter=A --name-only --since=<ship-date> -- '*.md'
```

**The automatable precondition is already covered.** That the rendered skill
instructs an agent to load the vocabulary, propose tags, and show them before
writing is asserted by `TestRenderedSpekKnowledgeCaptureFlowProposesTags` in
`internal/agent/instruction_surface_test.go`. This procedure measures the
*outcome* that precondition is meant to produce, which is a property of agent
behaviour over real work.

**Who and when.** The project maintainer, after roughly twenty entries have been
captured through the skill, or at the first release following one month of use,
whichever comes first.

---

## Metric 4: The tag vocabulary converges rather than fragments

**What to measure.** As entries accumulate, most new entries reuse existing tags,
distinct tags grow visibly more slowly than entries, and near-duplicates such as
`http` / `HTTP` / `http-api` do not accumulate beside each other.

**Pass condition.** Three checks, all of which must hold:

1. **Growth ratio.** Distinct tags grow more slowly than entries. Concretely,
   over any window in which the entry count grows by N, the distinct-tag count
   grows by **fewer than N/2**.
2. **Reuse.** The most-used tag has a count of **at least 3** once twenty entries
   exist, i.e. the vocabulary has a head rather than being flat.
3. **No near-duplicates.** No two tags in the listing are casing or punctuation
   variants of the same subject (`http` beside `http-api` or `http_api`).
   Genuinely distinct neighbouring subjects (`http` beside `https`) are correct
   and must **not** be counted as fragmentation.

**How.** The vocabulary listing is ordered most-used first, which is exactly the
shape this metric reads:

```bash
go run . knowledge tags --tier all
go run . knowledge list --tier all | wc -l
```

Record both numbers at two points in time and compare. Check 3 is a read of the
listing by eye: scan for pairs that differ only in case, punctuation or a
suffix like `-api`.

**Two supporting properties are already covered automatically**, and do not need
re-checking by hand: tags are lower-cased and de-duplicated on read (so `http`
and `HTTP` can never both exist), asserted in
`internal/store/frontmatter_test.go`; and the listing is ordered by how widely a
tag is used, asserted in
`TestSet_TagsReportsEachTagWithItsEntryCountMostUsedFirst`. The semantic
near-duplicate case is agent judgement and stays manual.

**A known gap to watch while running check 3.** The capture flow tells agents not
to carry both a singular and a plural, on the grounds that prefix matching
relates them. That reasoning fails for any singular shorter than four characters,
because `minPrefixLen = 4` rejects it: `tag` does not retrieve a `tags` tag, and
the same applies to `api`, `job` and `key`. If short singular/plural pairs start
appearing in the listing, that is the guidance being worked around rather than
the vocabulary fragmenting, and the fix belongs in the tag-form rules in
`templates/skills/workflows/spek-knowledge/SKILL.md`.

**Who and when.** The project maintainer, at the same review as Metric 3, and
again one quarter later, since convergence is only visible over a longer window
than adoption.
