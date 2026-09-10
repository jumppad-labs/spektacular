package knowledge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// This file pins the retrieval benchmark: the one measurement the tag-and-
// ranking work in ranking.go exists to move. It is not a unit test of the
// formula — ranking_test.go covers the parts — it is the end-to-end check that
// the parts still add up to the outcome that justified them.
//
// The observation being guarded. An agent planning against this knowledge base
// was seen searching for the same subject three different ways on three
// different runs — "http endpoints go", then "http handler", then "http
// endpoints handler routing" — and a fourth plausible phrasing, "api", returned
// nothing at all. Worse, a long early-concept document that mentions "go",
// "api" and "http" in passing, and says nothing whatever about routing, beat
// the document that was actually about routing. Relevance lost to volume, and
// the answer an agent got depended on which words it happened to reach for.
//
// The fix was tags plus a ranking that weighs them heavily: an entry declares
// what it is about, and score = Σ(tagWeight*tagAffinity(term) + damp(count)) ×
// coverage² lets "about X" outrank "mentions X a lot". tagWeight is 8 precisely
// so no plausible number of passing mentions can reach a single tag match — a
// term would have to appear 128 times to earn 8 from damp alone.
//
// Why this file matters to anyone touching the constants. tagWeight,
// coverageExponent and cutoffFraction are tunable, and a plausible-looking tune
// can quietly hand the whole improvement back: lower tagWeight far enough and
// the long rambling document wins again, on exactly the queries below, with
// every unit test in ranking_test.go still green. If a change here turns these
// four queries red, that is the benchmark doing its job. Do not fix it by
// adjusting the fixture or weakening the assertion to "is present somewhere in
// the results" — establish first whether the ranking genuinely got worse.
//
// Why the fixture is self-contained, and why that is non-negotiable. Everything
// below is built inside t.TempDir(). It never points at .spektacular/knowledge
// or any other real directory, and it must never be changed to. This was
// measured, not assumed: adding the real decisions/ranked-or-over-boolean-and.md
// entry — which quotes all four of these queries verbatim, and so scores highly
// on every one of them — to the live corpus moves the target entry from rank 1
// to rank 2 on "http handler". A benchmark bound to a directory that grows every
// time the project writes down what it learned would flap for reasons that have
// nothing to do with the code it is meant to guard, and the flapping would look
// like a regression. The corpus is part of the measurement, so the corpus is
// fixed here.
//
// What is asserted is rank, never score. The absolute numbers depend on the
// exact wording of the fixture prose, and rewording a sentence below is not a
// regression. The metric was always "does the right entry come first", so that
// is what these tests check. The single exception is documented on the test
// that makes it.

// benchmarkTarget is the on-topic entry: the one that should win every query
// below. Its frontmatter declares what it is about, which is the whole point —
// the body is short, and against the historical boolean-AND-over-prose rule it
// lost to the long document further down.
//
// Its body deliberately never uses the word "api". That omission is load-bearing
// and is asserted in TestBenchmark_TagOnlyRetrieval: it is what makes a search
// for "api" a test of tag-based retrieval rather than a test of prose matching.
const benchmarkTargetTags = `---
tags: [go, http, api, routing, handler, endpoints, middleware, chi]
---
`

const benchmarkTargetBody = `# HTTP routing

Every inbound request is routed through a single chi router built at startup.
Routes are declared in one file so the routing table can be read top to bottom.

Middleware is mounted in a fixed order and that order is load-bearing: request
id, panic recovery, structured logging, then authentication. A handler never
re-implements any of those concerns; it reads the request, calls into the
domain, and writes a response.

Each handler is a plain http.HandlerFunc. Sub-routers are mounted per resource,
so adding a group of routes is one mount rather than a scattering of new rules.
`

// benchmarkCompetitor is the long early-concept document that used to win. It
// mentions "go", "api", "http" and "endpoints" repeatedly and in passing, and
// says nothing at all about routing — there is no "routing" and no "handler"
// anywhere in it. It carries no tags, because entries like this predate tags
// and nobody would think to tag a rambling idea note.
//
// The repetition is the point and is not padding: this document has to be a
// genuine competitor on raw occurrence counts, because "volume beat relevance"
// is the failure being guarded against. A straw man with one mention each would
// pass these tests under any weighting, including the ones that are broken.
const benchmarkCompetitorBody = `# Initial idea

This is the earliest concept note for the project, written long before any code
existed. It is long, rambling, and mentions a great deal of technology in
passing without committing to any of it.

The plan is to write the service in go, because go is what the team already
knows and go builds a single static binary. A go module per service, go tests
in the same package, go vet in continuous integration.

The service will expose an api. The api should be small at first and grow only
when a consumer asks for it. We may add a second api later for internal tools,
and an api for the admin console after that. Whatever the api ends up being, it
must be versioned, and the versioned endpoints must keep working. A handful of
read-only endpoints first, write endpoints once we know the shape of the data.

Everything talks over http. http is boring, universal and well understood, and
http caching is free if we do not fight it. We might put a http proxy in front
of the service later, and http keep-alive should be on by default.

None of this says anything about how a request finds the code that answers it.
That is a later problem, and this note deliberately does not try to solve it.
`

// benchmarkCorpus writes the two-entry fixture into a fresh temp dir and returns
// a Set over it. The dir is created by t.TempDir() and is torn down with the
// test: no real knowledge store is read, and none can be.
func benchmarkCorpus(t *testing.T) (*Set, string) {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, dir, "architecture/http-routing.md", benchmarkTargetTags+"\n"+benchmarkTargetBody)
	writeFile(t, dir, "architecture/initial-idea.md", benchmarkCompetitorBody)
	return singleSourceSet(t, dir), dir
}

// The headline metric: all four of the phrasings an agent was observed reaching
// for return the entry that is actually about the subject first. Before tags and
// ranked OR, one of these returned the rambling idea note ahead of it and one
// returned nothing at all.
//
// The four queries are not variations invented for coverage. They are the exact
// strings that were searched, in order, across three planning runs plus the
// fourth phrasing that came up empty; they differ in length, in word order and
// in which words they share with the target's prose, which is precisely why a
// ranking that only worked for one of them was not good enough.
func TestBenchmark_ObservedQueriesRankTheOnTopicEntryFirst(t *testing.T) {
	set, _ := benchmarkCorpus(t)

	// Every one of the four expects the same answer. That is the metric: the
	// phrasing an agent happens to choose must stop deciding what it finds.
	const target = "architecture/http-routing.md"

	queries := []struct {
		query string
		first string
	}{
		{query: "http endpoints go", first: target},
		{query: "http handler", first: target},
		{query: "http endpoints handler routing", first: target},
		{query: "api", first: target},
	}

	for _, tc := range queries {
		t.Run(tc.query, func(t *testing.T) {
			hits, err := set.Search(tc.query, Selector{Tier: TierAll})
			require.NoError(t, err)

			// Logged rather than asserted: the scores move whenever the fixture
			// prose is reworded, so they are diagnostics, not expectations. When
			// this test fails they are the first thing you want to see.
			for i, hit := range hits {
				t.Logf("rank %d: %s score=%v", i+1, hit.Path, hit.Score)
			}

			// Checked before indexing so a regression that returns nothing
			// reports "no hits" instead of panicking on an out-of-range index —
			// "returned nothing at all" is one of the two failure modes this
			// benchmark exists to catch, and it should say so.
			require.NotEmpty(t, hits, "query %q returned no hits at all", tc.query)
			require.Equal(t, tc.first, hits[0].Path,
				"query %q must rank the on-topic entry first, ahead of the long idea note", tc.query)
		})
	}
}

// Criterion 2 in its sharpest form: "api" appears nowhere in the target's prose,
// yet searching for "api" returns the target first, ahead of the idea note that
// says "api" five times. The entry is found for what it declares itself to be
// about, not for the words it happens to contain — which is the entire argument
// for tags, and the reason the fourth observed phrasing used to return nothing.
func TestBenchmark_TagOnlyRetrieval(t *testing.T) {
	set, dir := benchmarkCorpus(t)

	// Read the fixture back off disk and assert the body genuinely lacks the
	// term, rather than trusting the constant above to have stayed that way.
	// Without this the test could silently decay into "the body happened to
	// mention it" the next time someone rewords the fixture, and would then be
	// asserting nothing at all.
	raw, err := os.ReadFile(filepath.Join(dir, "architecture", "http-routing.md"))
	require.NoError(t, err)
	_, body, found := strings.Cut(string(raw), "\n---\n")
	require.True(t, found, "fixture must carry a frontmatter block")
	require.NotContains(t, strings.ToLower(body), "api",
		"the target's body must not mention the query term, or this test proves nothing about tags")

	hits, err := set.Search("api", Selector{Tier: TierAll})
	require.NoError(t, err)
	require.NotEmpty(t, hits, `searching "api" returned no hits at all`)
	require.Equal(t, "architecture/http-routing.md", hits[0].Path,
		"an entry tagged api must outrank an entry that merely says api repeatedly")

	// The one score worth pinning exactly, and the reason it is exact: a
	// single-term query matched purely on an exact tag scores
	// tagWeight*1 + damp(0) = 8, scaled by a coverage of 1/1 squared — 8 * 1² =
	// 8.0, with no body occurrences contributing anything at all. Every value in
	// that product is exactly representable in float64, so this is require.Equal
	// and not an InDelta. Asserting it documents where the retrieval came from:
	// if this number ever changes, the entry is being found by something other
	// than its tag, and the test above would keep passing while the property it
	// is named for had quietly stopped holding.
	require.Equal(t, tagWeight, hits[0].Score,
		"the target must score exactly one full tag weight, earned from its tag alone")
}
