package knowledge

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Phase 1.1 criterion 2: tokenization is a pure function of the query string,
// exercised directly with no store and no filesystem. Every expected slice is
// hand-written rather than derived from Terms itself.
func TestTerms_TokenizesQuery(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  []string
	}{
		{
			name:  "plain multi-word query splits on single spaces",
			query: "alpha bravo charlie",
			want:  []string{"alpha", "bravo", "charlie"},
		},
		{
			name:  "mixed case is lower-cased",
			query: "Alpha BRAVO ChArLiE",
			want:  []string{"alpha", "bravo", "charlie"},
		},
		{
			name:  "runs of spaces, tabs and newlines collapse",
			query: "  alpha\t\tbravo \n charlie  ",
			want:  []string{"alpha", "bravo", "charlie"},
		},
		{
			name:  "an empty query yields no terms",
			query: "",
			want:  nil,
		},
		{
			name:  "an all-whitespace query yields no terms",
			query: " \t\n ",
			want:  nil,
		},
		{
			name: "term order follows the query, not sort order",
			// Reverse-alphabetical, so a sorted result would be visibly wrong.
			query: "zulu yankee xray",
			want:  []string{"zulu", "yankee", "xray"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Terms(tc.query)
			if len(tc.want) == 0 {
				// "No terms" is the contract; whether that is a nil or an
				// allocated empty slice is not, so it is not asserted.
				require.Empty(t, got, "query %q should tokenize to no terms", tc.query)
				return
			}
			require.Equal(t, tc.want, got)
		})
	}
}

// Phase 1.1 criterion 2: the ranking rule lives in one place and is exercised
// directly — no store, no temp dir, no files. Every expected score is
// hand-computed from the evidence literal beside it, from damp(n) = 1 + log2(n)
// and the coverage factor (covered/total)^2.
//
// Cases whose arithmetic lands on an exact float64 assert with require.Equal;
// the ones involving log2 of a non-power of two are irrational and assert with
// require.InDelta, which is what a non-zero delta below selects.
func TestScore_FromEvidence(t *testing.T) {
	tests := []struct {
		name     string
		evidence Evidence
		want     float64
		delta    float64 // 0 means the value is exact: assert with require.Equal
	}{
		{
			// damp(3) + damp(5) = 2.5849625007211562 + 3.321928094887362,
			// coverage 2/2 = 1, so the sum stands unscaled.
			name:     "all terms present sums their damped counts",
			evidence: Evidence{BodyCounts: []int{3, 5}},
			want:     5.906890595608518,
			delta:    1e-9,
		},
		{
			// damp(1) = 1 + log2(1) = 1 exactly, coverage 1/1 = 1. This is the
			// anchor the curve preserves, so every assertion pinning 1.0 for a
			// single occurrence of a single term stays meaningful.
			name:     "a single term occurring once scores one",
			evidence: Evidence{BodyCounts: []int{1}},
			want:     1,
		},
		{
			// Partial credit: a missing term costs coverage, not the whole
			// score. damp(9) + damp(0) + damp(4) = 4.169925001442312 + 0 + 3 =
			// 7.169925001442312, coverage 2/3, so 7.169925001442312 * 4/9.
			name:     "a zero count in the middle costs coverage, not the whole score",
			evidence: Evidence{BodyCounts: []int{9, 0, 4}},
			want:     3.1866333339743607,
			delta:    1e-9,
		},
		{
			// Same partial credit, with the missing term first rather than in
			// the middle. damp(0) + damp(7) = 3.807354922057604, coverage 1/2,
			// so 3.807354922057604 * 0.25.
			name:     "a leading zero count costs coverage, not the whole score",
			evidence: Evidence{BodyCounts: []int{0, 7}},
			want:     0.951838730514401,
			delta:    1e-9,
		},
		{
			name:     "empty body counts score zero",
			evidence: Evidence{BodyCounts: []int{}},
			want:     0,
		},
		{
			name:     "nil body counts score zero",
			evidence: Evidence{},
			want:     0,
		},
		{
			// A tag match is added to the term's damped body occurrences rather
			// than replacing them: tagWeight * 1 + damp(2) = 8 + 2 = 10, with
			// coverage 1/1 = 1 leaving it unscaled. Both halves are exact in
			// float64.
			name:     "a tag match adds the tag weight on top of body occurrences",
			evidence: Evidence{BodyCounts: []int{2}, TagAffinity: []float64{1}},
			want:     10,
		},
		{
			// The sibling of the case above, and the one that pins tag evidence
			// as evidence in its own right: the term never occurs in the body,
			// so damp(0) = 0 and the whole score is the tag weight — 8 * 1 + 0 =
			// 8. Coverage is 1/1 rather than 0/1, because a term with a tag
			// match counts as covered; were coverage counted from body
			// occurrences alone this would score 0 and the entry would be
			// dropped as a non-match.
			name:     "a term with tag affinity but no body occurrence still counts as covered",
			evidence: Evidence{BodyCounts: []int{0}, TagAffinity: []float64{1}},
			want:     8,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.delta == 0 {
				require.Equal(t, tc.want, score(tc.evidence))
				return
			}
			require.InDelta(t, tc.want, score(tc.evidence), tc.delta)
		})
	}
}

// Phase 1.2 criterion 3: occurrences buy ever less as they pile up, so a long
// document cannot climb the results on volume alone. Every expected value is
// hand-computed from damp(n) = 1 + log2(n); the powers of two land on exact
// float64 values and assert with require.Equal.
func TestDamp_DiminishingReturns(t *testing.T) {
	tests := []struct {
		name  string
		n     int
		want  float64
		delta float64 // 0 means the value is exact: assert with require.Equal
	}{
		{name: "no occurrences are worth nothing", n: 0, want: 0},
		{name: "one occurrence is worth exactly one", n: 1, want: 1},
		{name: "two occurrences: 1 + log2(2)", n: 2, want: 2},
		{name: "four occurrences: 1 + log2(4)", n: 4, want: 3},
		{name: "eight occurrences: 1 + log2(8)", n: 8, want: 4},
		{name: "sixteen occurrences: 1 + log2(16)", n: 16, want: 5},
		{name: "three occurrences: 1 + log2(3)", n: 3, want: 2.5849625007211562, delta: 1e-9},
		{name: "five occurrences: 1 + log2(5)", n: 5, want: 3.321928094887362, delta: 1e-9},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.delta == 0 {
				require.Equal(t, tc.want, damp(tc.n))
				return
			}
			require.InDelta(t, tc.want, damp(tc.n), tc.delta)
		})
	}

	// Criterion 3 in its clearest form: twenty mentions is better than two, but
	// nowhere near ten times better. damp(20) = 1 + log2(20) = 5.321928094887363
	// against 10 * damp(2) = 20.
	require.Greater(t, damp(20), damp(2),
		"twenty mentions must still beat two")
	require.Less(t, damp(20), 10*damp(2),
		"twenty mentions must not score ten times two mentions")
}

// Phase 2.2: the term-to-tag relation is a pure function of one term and one
// entry's tags, exercised directly with no store, no files and no scoring. Every
// expected affinity is hand-written from the rule rather than read back from the
// implementation: 1 for an exact match; otherwise, when one string opens the
// other and the shorter reaches minPrefixLen = 4 runes, the shorter's rune count
// over the longer's; and 0 for everything else.
//
// The zero cases are the load-bearing half of this table. Three guards produce
// them and each is asserted separately, because a "smarter" matcher would break
// a different one:
//
//   - the start anchor: "test" earns nothing from "latest", which shares its
//     letters but not its opening;
//   - the length floor: "go" earns nothing from "golang", or every two-letter
//     term would hit every tag beginning with those letters;
//   - prefix rather than string distance: tls/tld, dns/dos and api/apt are the
//     pairs an edit-distance measure would score at 0.67 and test/rest and
//     cors/core the pairs it would score at 0.75 — every one of them a pair of
//     unrelated subjects. Prefix scores all five zero while producing identical
//     numbers on every pair that should match, which is why the doc comment on
//     tagAffinity says not to "improve" it into a similarity metric.
//
// Cases whose arithmetic is a clean binary value (0 and 1) assert with
// require.Equal; the fractions are not exact in float64 and assert with
// require.InDelta, which is what a non-zero delta below selects.
func TestTagAffinity_MatchesExactlyOrByAnchoredPrefix(t *testing.T) {
	tests := []struct {
		name  string
		term  string
		tags  []string
		want  float64
		delta float64 // 0 means the value is exact: assert with require.Equal
	}{
		{
			name: "an exact match is full credit",
			term: "go",
			tags: []string{"go"},
			want: 1,
		},
		{
			name: "an exact match is full credit for a longer tag too",
			term: "https",
			tags: []string{"https"},
			want: 1,
		},
		{
			// The term opens the tag: 4 runes over 5.
			name:  "a term that opens a longer tag earns the length ratio",
			term:  "http",
			tags:  []string{"https"},
			want:  0.8,
			delta: 1e-9,
		},
		{
			// The tag opens the term, which is the same rule in the other
			// direction: 4 runes over 5 again.
			name:  "a tag that opens a longer term earns the same ratio",
			term:  "https",
			tags:  []string{"http"},
			want:  0.8,
			delta: 1e-9,
		},
		{
			// Singular and plural find each other without either form being
			// written down twice: 5 runes over 6.
			name:  "a singular term finds its plural tag",
			term:  "apple",
			tags:  []string{"apples"},
			want:  0.8333333333333334,
			delta: 1e-9,
		},
		{
			name:  "a plural term finds its singular tag",
			term:  "apples",
			tags:  []string{"apple"},
			want:  0.8333333333333334,
			delta: 1e-9,
		},
		{
			// A compound tag is a weak match for the term it opens with: 5
			// runes over 14.
			name:  "a term opening a compound tag earns the proportional share",
			term:  "https",
			tags:  []string{"https-security"},
			want:  0.35714285714285715,
			delta: 1e-9,
		},
		{
			// The start anchor: "latest" ends with "test" but does not begin
			// with it, so there is no credit at all.
			name: "a term that a tag merely ends with earns nothing",
			term: "test",
			tags: []string{"latest"},
			want: 0,
		},
		{
			// Edit distance would score this pair 0.75. Testing and REST are
			// unrelated subjects and both are plausible tags here.
			name: "test earns nothing from a rest tag",
			term: "test",
			tags: []string{"rest"},
			want: 0,
		},
		{
			// Edit distance would score this pair 0.75 as well. CORS and a
			// "core" tag are unrelated.
			name: "cors earns nothing from a core tag",
			term: "cors",
			tags: []string{"core"},
			want: 0,
		},
		{
			// The three pairs edit distance would score at 0.67, all three of
			// them unrelated subjects.
			name: "tls earns nothing from a tld tag",
			term: "tls",
			tags: []string{"tld"},
			want: 0,
		},
		{
			name: "dns earns nothing from a dos tag",
			term: "dns",
			tags: []string{"dos"},
			want: 0,
		},
		{
			name: "api earns nothing from an apt tag",
			term: "api",
			tags: []string{"apt"},
			want: 0,
		},
		{
			// The length floor: the shorter side is 2 runes, below
			// minPrefixLen = 4, so the prefix relation counts for nothing.
			name: "a two-letter term earns nothing from a longer tag it opens",
			term: "go",
			tags: []string{"golang"},
			want: 0,
		},
		{
			// The same two-letter term still matches exactly. The floor
			// restricts short terms to exact matching, it does not exclude them.
			name: "a two-letter term still matches its tag exactly",
			term: "go",
			tags: []string{"golang", "go"},
			want: 1,
		},
		{
			// A three-letter term is below the floor too, so a shared opening
			// buys nothing.
			name: "a three-letter term earns nothing from a longer tag it opens",
			term: "api",
			tags: []string{"apis"},
			want: 0,
		},
		{
			// Prefix matching is positional, not morphological: "route" is not
			// the opening of "routing".
			name: "a term is not stemmed onto a tag with a different ending",
			term: "route",
			tags: []string{"routing"},
			want: 0,
		},
		{
			name: "a term unrelated to every tag earns nothing",
			term: "router",
			tags: []string{"go", "http"},
			want: 0,
		},
		{
			name: "an entry with no tags earns nothing",
			term: "http",
			tags: nil,
			want: 0,
		},
		{
			name: "an entry with an empty tag list earns nothing",
			term: "http",
			tags: []string{},
			want: 0,
		},
		{
			// The strongest match decides rather than the matches accumulating:
			// this is 1, not 1 + 5/14. Listed exact-first here and
			// weakest-first below so neither answer can come from tag order.
			name: "an exact match wins over a prefix match on the same entry",
			term: "https",
			tags: []string{"https", "https-security"},
			want: 1,
		},
		{
			name: "the exact match still wins when it is listed last",
			term: "https",
			tags: []string{"https-security", "https"},
			want: 1,
		},
		{
			// Two prefix matches of different strengths, neither exact: 4/5
			// beats 4/14, and they do not add.
			name:  "the strongest prefix match wins over a weaker one",
			term:  "http",
			tags:  []string{"https-security", "https"},
			want:  0.8,
			delta: 1e-9,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.delta == 0 {
				require.Equal(t, tc.want, tagAffinity(tc.term, tc.tags))
				return
			}
			require.InDelta(t, tc.want, tagAffinity(tc.term, tc.tags), tc.delta)
		})
	}
}
