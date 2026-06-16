package store

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// longMatchLine is a single line, well over maxExcerptBytes (256), that
// contains the keyword "needle". It makes the excerpt-budget assertion
// meaningful: a naive implementation would emit an excerpt > 256 bytes.
const longMatchLine = "alpha beta gamma delta epsilon zeta eta theta iota kappa lambda mu nu xi " +
	"omicron pi rho sigma tau upsilon phi chi psi omega the needle is buried deep within " +
	"this very long line of padding text padding padding padding padding padding padding " +
	"padding padding padding padding padding padding padding padding done"

// writeSearchFixture writes a known set of files into a fresh temp dir and
// returns that dir. It includes a nested subdirectory and one file with a
// match line longer than maxExcerptBytes.
func writeSearchFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	fx := NewFileStore(dir, "project")

	require.NoError(t, fx.Write("top.txt", []byte("the needle in the haystack\nunrelated content here\n")))
	require.NoError(t, fx.Write("nomatch.txt", []byte("nothing of interest here\njust filler text\n")))
	require.NoError(t, fx.Write("nested/deep.txt", []byte("a NEEDLE with different case\ntrailing line\n")))
	require.NoError(t, fx.Write("long.txt", []byte(longMatchLine+"\n")))
	// A file under conventions/ that DOES contain "needle": if the exclusion
	// were broken, this would be returned, which is what makes the exclusion
	// assertion meaningful.
	require.NoError(t, fx.Write("conventions/style.md", []byte("the needle lives in a convention\n")))

	return dir
}

// Criterion 1: every excerpt stays within the compact budget.
func TestSearch_ExcerptWithinBudget(t *testing.T) {
	dir := writeSearchFixture(t)
	st := NewFileStore(dir, "project")

	hits, err := st.Search("needle")
	require.NoError(t, err)
	require.NotEmpty(t, hits, "fixture should yield matches for 'needle'")

	for _, h := range hits {
		require.NotEmpty(t, h.Excerpts,
			"hit for %s should carry at least one excerpt", h.Path)
		for _, ex := range h.Excerpts {
			require.LessOrEqual(t, len(ex), maxExcerptBytes,
				"excerpt for %s exceeds budget", h.Path)
		}
	}
}

// Criterion 1 (helper-level): trimExcerpt caps a long string at the budget.
func TestTrimExcerpt_CapsLongString(t *testing.T) {
	require.Equal(t, "short text", trimExcerpt("  short   text  "))

	long := strings.Repeat("x", maxExcerptBytes*2)
	got := trimExcerpt(long)
	require.Equal(t, maxExcerptBytes, len(got))
}

// Criterion 3: each hit carries the store's scope and a Path that round-trips
// through Read; a no-match query returns an empty result and no error.
func TestSearch_ScopeAndLocatorRoundTrip(t *testing.T) {
	dir := writeSearchFixture(t)
	st := NewFileStore(dir, "project")

	hits, err := st.Search("needle")
	require.NoError(t, err)
	require.NotEmpty(t, hits)

	for _, h := range hits {
		require.Equal(t, st.Scope(), h.Scope, "hit scope should match store scope")
		data, readErr := st.Read(h.Path)
		require.NoError(t, readErr, "hit Path %q should round-trip through Read", h.Path)
		require.NotEmpty(t, data)
	}

	noHits, err := st.Search("zzz-does-not-exist-zzz")
	require.NoError(t, err)
	require.Empty(t, noHits)
}

// Matching is case-insensitive, conventions/ is excluded from search, and a
// query with no terms — empty or whitespace-only — returns an empty result
// and no error.
func TestSearch_CaseInsensitiveAndExcludesConventions(t *testing.T) {
	dir := writeSearchFixture(t)
	st := NewFileStore(dir, "project")

	hits, err := st.Search("needle")
	require.NoError(t, err)
	require.NotEmpty(t, hits)

	var paths []string
	for _, h := range hits {
		paths = append(paths, h.Path)
	}

	// nested/deep.txt's only occurrence is the uppercase "NEEDLE", so its
	// presence proves the lowercase query matched case-insensitively.
	require.Contains(t, paths, "nested/deep.txt",
		"the lowercase query should match the uppercase NEEDLE")

	// The fixture's conventions/style.md DOES contain "needle", so the absence
	// of any conventions/ path proves the exclusion rather than a non-match.
	for _, p := range paths {
		require.False(t, strings.HasPrefix(p, "conventions/"),
			"hit %q should have been excluded from search", p)
	}

	emptyHits, err := st.Search("")
	require.NoError(t, err)
	require.Empty(t, emptyHits)

	// A whitespace-only query tokenizes to zero terms and behaves like the
	// empty query: no hits, no error.
	blankHits, err := st.Search(" \t\n ")
	require.NoError(t, err)
	require.Empty(t, blankHits)
}

// A hit's Score sums non-overlapping, case-insensitive occurrences of the
// query across the whole document, whether they sit on one line or are spread
// over several. Uses its own fixture: scoring needs files with a known
// occurrence count, which writeSearchFixture does not provide.
func TestSearch_ScoreSumsOccurrencesAcrossDocument(t *testing.T) {
	dir := t.TempDir()
	fx := NewFileStore(dir, "project")
	require.NoError(t, fx.Write("twice.txt", []byte("a needle next to another needle on one line\n")))
	require.NoError(t, fx.Write("once.txt", []byte("just one needle here\n")))
	require.NoError(t, fx.Write("upper.txt", []byte("a single NEEDLE in caps\n")))
	// Occurrences spread over three lines: 1 + 2 + 1 = 4.
	require.NoError(t, fx.Write("spread.txt", []byte(
		"first needle on its own line\n"+
			"a needle beside a NEEDLE midway\n"+
			"closing needle at the end\n")))

	st := NewFileStore(dir, "project")

	hits, err := st.Search("needle")
	require.NoError(t, err)

	// Independent oracle: hand-maintained expected score per path. The length
	// check pins exactly one hit per matching document — the map alone would
	// hide a duplicate.
	want := map[string]float64{
		"twice.txt":  2,
		"once.txt":   1,
		"upper.txt":  1,
		"spread.txt": 4,
	}
	require.Len(t, hits, len(want))

	got := make(map[string]float64, len(hits))
	for _, h := range hits {
		got[h.Path] = h.Score
	}
	require.Equal(t, want, got)
}

// A file whose leading bytes contain a NUL byte is classified as binary and
// quietly skipped, while a sibling text file still matches.
func TestSearch_SkipsBinaryFiles(t *testing.T) {
	dir := t.TempDir()
	fx := NewFileStore(dir, "project")
	// The binary file contains the needle, so a hit from it would prove the
	// NUL-byte skip is broken rather than the file simply not matching.
	require.NoError(t, fx.Write("blob.bin", []byte("\x00\x01\x02 a needle hidden in binary data\n")))
	require.NoError(t, fx.Write("plain.txt", []byte("a needle in plain text\n")))

	st := NewFileStore(dir, "project")

	hits, err := st.Search("needle")
	require.NoError(t, err)

	var paths []string
	for _, h := range hits {
		paths = append(paths, h.Path)
	}
	require.Equal(t, []string{"plain.txt"}, paths,
		"only the text file should hit; the binary file must be skipped")
}

// A line longer than scanBufferBytes skips the remainder of that file only —
// matches found before the long line and matches in sibling files are still
// returned, and the search reports success.
func TestSearch_OversizedLineSkipsFileNotSearch(t *testing.T) {
	dir := t.TempDir()
	fx := NewFileStore(dir, "project")
	oversized := "the needle before the long line\n" +
		strings.Repeat("x", scanBufferBytes+10) + "\n"
	require.NoError(t, fx.Write("oversized.txt", []byte(oversized)))
	require.NoError(t, fx.Write("sibling.txt", []byte("a needle in the sibling file\n")))

	st := NewFileStore(dir, "project")

	hits, err := st.Search("needle")
	require.NoError(t, err, "an oversized line must not fail the search")

	var paths []string
	for _, h := range hits {
		paths = append(paths, h.Path)
	}
	require.ElementsMatch(t, []string{"oversized.txt", "sibling.txt"}, paths,
		"the pre-long-line match and the sibling match should both survive")
}

// Phase 1.1: a multi-word query matches a document whose words sit on
// different lines, in a different order than queried; a document missing any
// one of the words is not returned.
func TestSearch_MultiWordScatteredAcrossLines(t *testing.T) {
	dir := t.TempDir()
	fx := NewFileStore(dir, "project")
	// All three query words present, each on its own line, in reverse order.
	require.NoError(t, fx.Write("scattered.txt", []byte(
		"charlie opens the file\n"+
			"some filler in between\n"+
			"bravo turns up later\n"+
			"and alpha closes it out\n")))
	// Contains alpha and bravo but never charlie, so its exclusion proves the
	// every-term requirement rather than a trivial non-match.
	require.NoError(t, fx.Write("partial.txt", []byte(
		"alpha is here\n"+
			"and bravo is here too\n")))

	st := NewFileStore(dir, "project")

	hits, err := st.Search("alpha bravo charlie")
	require.NoError(t, err)

	var paths []string
	for _, h := range hits {
		paths = append(paths, h.Path)
	}
	require.Equal(t, []string{"scattered.txt"}, paths,
		"only the document containing every term should match")
}

// Phase 1.1: a document with query words on many lines collapses into exactly
// one hit, scored as the hand-computed sum of every term's occurrences across
// the whole document.
func TestSearch_OneHitPerDocument(t *testing.T) {
	dir := t.TempDir()
	fx := NewFileStore(dir, "project")
	// Hand-computed: widget 2+1 = 3, gear 1+1+3 = 5, total 8.
	require.NoError(t, fx.Write("machine.txt", []byte(
		"widget widget on the first line\n"+
			"a gear sits on the second line\n"+
			"another widget and a gear together\n"+
			"gear gear gear to finish\n")))

	st := NewFileStore(dir, "project")

	hits, err := st.Search("widget gear")
	require.NoError(t, err)

	require.Len(t, hits, 1, "four matching lines must still collapse into one hit")
	require.Equal(t, "machine.txt", hits[0].Path)
	require.Equal(t, float64(8), hits[0].Score)
}

// Phase 1.2: a hit's Title is the text of the document's FIRST ATX heading —
// even though the heading line itself never matches the query — and a later
// second heading does not displace it.
func TestSearch_TitleFromFirstHeading(t *testing.T) {
	dir := t.TempDir()
	fx := NewFileStore(dir, "project")
	// The first heading contains no query word; only body lines match. A
	// second heading later in the file makes "first heading wins" meaningful.
	require.NoError(t, fx.Write("guide.md", []byte(
		"# Workflow Guide\n"+
			"\n"+
			"the needle appears in the body\n"+
			"another needle further down\n"+
			"## Appendix Notes\n")))

	st := NewFileStore(dir, "project")

	hits, err := st.Search("needle")
	require.NoError(t, err)

	require.Len(t, hits, 1)
	require.Equal(t, "Workflow Guide", hits[0].Title,
		"Title should be the first heading's text, not a matching line or the later heading")
}

// Phase 1.2: a document with no ATX heading falls back to using the hit's
// locator (relative path) as its Title.
func TestSearch_TitleFallsBackToLocator(t *testing.T) {
	dir := t.TempDir()
	fx := NewFileStore(dir, "project")
	// JSON-ish content: matches the query but contains no heading line.
	require.NoError(t, fx.Write("nested/data.json", []byte(
		"{\n"+
			"  \"comment\": \"the needle hides in json\"\n"+
			"}\n")))

	st := NewFileStore(dir, "project")

	hits, err := st.Search("needle")
	require.NoError(t, err)

	require.Len(t, hits, 1)
	require.Equal(t, "nested/data.json", hits[0].Path)
	require.Equal(t, hits[0].Path, hits[0].Title,
		"a heading-less document should fall back to the locator as its Title")
}

// Phase 1.2: with a two-word query, the line containing BOTH words ranks
// ahead of lines containing only one, so it becomes the first excerpt. The
// expected string is the hand-written trimmed line.
func TestSearch_ExcerptPrefersLineWithMoreTerms(t *testing.T) {
	dir := t.TempDir()
	fx := NewFileStore(dir, "project")
	// The single-term lines come first in the file, so file order alone would
	// put one of them at the front — only term-count ranking promotes the
	// two-word line.
	require.NoError(t, fx.Write("machine.txt", []byte(
		"only a widget on this line\n"+
			"just a gear over here\n"+
			"a widget meets a gear at last\n")))

	st := NewFileStore(dir, "project")

	hits, err := st.Search("widget gear")
	require.NoError(t, err)

	require.Len(t, hits, 1)
	require.NotEmpty(t, hits[0].Excerpts)
	require.Equal(t, "a widget meets a gear at last", hits[0].Excerpts[0],
		"the line containing both query words should be the first excerpt")
}

// Phase 1.2: a document with more matching lines than the cap still carries
// exactly maxExcerptsPerHit (3) excerpts.
func TestSearch_ExcerptCountCapped(t *testing.T) {
	dir := t.TempDir()
	fx := NewFileStore(dir, "project")
	// Five matching lines — two more than the cap.
	require.NoError(t, fx.Write("many.txt", []byte(
		"needle line one\n"+
			"needle line two\n"+
			"needle line three\n"+
			"needle line four\n"+
			"needle line five\n")))

	st := NewFileStore(dir, "project")

	hits, err := st.Search("needle")
	require.NoError(t, err)

	require.Len(t, hits, 1)
	require.Len(t, hits[0].Excerpts, 3,
		"five matching lines must be capped at three excerpts")
}

// Phase 1.1 regression: a single-word query finds exactly the same files the
// per-line implementation did. The oracle is the hand-maintained list of
// shared-fixture files containing "needle" outside conventions/.
func TestSearch_SingleWordMatchesSameFiles(t *testing.T) {
	dir := writeSearchFixture(t)
	st := NewFileStore(dir, "project")

	hits, err := st.Search("needle")
	require.NoError(t, err)

	var paths []string
	for _, h := range hits {
		paths = append(paths, h.Path)
	}
	require.ElementsMatch(t, []string{"top.txt", "nested/deep.txt", "long.txt"}, paths)
}
