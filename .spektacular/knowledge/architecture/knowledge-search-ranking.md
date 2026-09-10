---
tags: [knowledge, search, ranking, retrieval, tags]
---

# How knowledge search ranks results

Search is a ranked OR over a full filesystem walk. A document does not have to
contain every word of a query: it is returned if it carries evidence for any of
them, and ranked on how good a match it is. Two consequences bind anything
consuming search — a returned hit is not proof that every query word appeared in
it, and an empty result is not proof that nothing on the subject exists.

## The three factors

For each query term, an entry earns `tagWeight * tagAffinity(term) + damp(count)`:

- **A tag match** is weighted far above prose. It is the only signal in the
  formula that says an entry is *about* a subject rather than merely mentioning
  it. At `tagWeight = 8`, a term would have to appear 128 times in the body to
  earn as much from damping alone.
- **Damped body occurrences**, `damp(n) = 1 + log₂(n)` for `n > 0` and `0`
  otherwise, so repetition saturates. Twenty mentions scores 5.32 against two
  mentions' 2.0, not ten times as much. `damp(1) = 1` exactly, and that is
  load-bearing: several assertions pin a single occurrence of a single term at
  exactly 1.0.
- **Coverage**, the share of query terms with any evidence at all, tag or body,
  raised to `coverageExponent = 2` and multiplied through. An entry covering half
  the query keeps a quarter of its score.

Tag matching is exact at full strength, and prefix-related at proportional
partial credit: `len(shorter)/len(longer)` when one string opens the other and
the shorter reaches `minPrefixLen = 4` runes. So `http` earns four fifths from a
`https` tag. Two guards are load-bearing — the match is anchored at the start, so
`test` earns nothing from `latest`, and the length floor stops `go` reaching
`golang`. This is positional, not morphological: `route` does not find `routing`.

## The cutoff is relative, not absolute

After every store's hits are merged and sorted, anything scoring below
`cutoffFraction = 0.25` of the best hit is dropped. Relative rather than fixed,
so a loosely related entry surfaces when it is the only thing there and vanishes
once something genuinely relevant is present, and so the threshold keeps its
meaning as a knowledge base grows rather than needing a retune per corpus.

## Where each part is computed, and why

A store finds candidate documents and describes them: per-term body occurrence
counts, the entry's tags, locator, title, excerpts, checksum. It never computes a
score. Tokenizing, scoring, merging, sorting and the cutoff all happen once in
the knowledge layer, above the store interface.

The reason is the merge. Hits from every covered store are ranked against each
other, so every score must be on one scale. A formula reproduced inside each
provider would have to be reproduced exactly, and any drift would leave the
merged ordering quietly wrong rather than visibly broken. `Hit.Tier`, `Hit.Name`
and `Hit.Category` are already left empty by the store because it has no notion
of its caller's addressing scheme; `Score` joins them because it has no notion of
the stores it will be ranked against. The cutoff in particular can only be
applied after the merge, because the strongest hit may live in a different store
from the weak one being judged.

The constraint this places on future providers is real and worth meeting in
design rather than in a debugging session: a store must be able to report
per-term occurrence counts to take part in this ranking at all. A purely semantic
backend that can only return its own opaque relevance score cannot.

## Tuning a constant is a four-file change

`internal/knowledge/ranking.go` holds `tagWeight`, `coverageExponent`,
`minPrefixLen` and `cutoffFraction`, and is the single source of truth. Three
documents quote those values: `docs/knowledge-base.md`, the published
`knowledge-base.mdx` page on the docs site, and this entry. Changing one constant
means changing all four, or they disagree.

Two guards bound the envelope. `cutoffFraction` cannot exceed roughly 0.296
without breaking the pinned case that a two-of-three match survives against a
full match. The four-query benchmark in `internal/knowledge/benchmark_test.go`
constrains from the other side; it exists precisely so later tuning cannot
quietly hand back the improvement the weights were chosen for.

This entry records *how* ranking works. `decisions/ranked-or-over-boolean-and.md`
records *why* ranked OR was chosen over boolean AND, including the alternatives
rejected.
