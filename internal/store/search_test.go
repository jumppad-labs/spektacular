package store

import (
	"os/exec"
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

	return dir
}

// triple is the comparable projection of a Hit used for order-independent
// equivalence checks. Score is deliberately excluded: the rg path sets it to
// a submatch count while the native fallback leaves it 0.
type triple struct {
	Scope   string
	Path    string
	Excerpt string
}

func project(hits []Hit) []triple {
	out := make([]triple, 0, len(hits))
	for _, h := range hits {
		out = append(out, triple{Scope: h.Scope, Path: h.Path, Excerpt: h.Excerpt})
	}
	return out
}

// Criterion 1: every excerpt stays within the compact budget.
func TestSearch_ExcerptWithinBudget(t *testing.T) {
	dir := writeSearchFixture(t)
	st := NewFileStore(dir, "project")
	st.forceFallback = true

	hits, err := st.Search("needle")
	require.NoError(t, err)
	require.NotEmpty(t, hits, "fixture should yield matches for 'needle'")

	for _, h := range hits {
		require.LessOrEqual(t, len(h.Excerpt), maxExcerptBytes,
			"excerpt for %s exceeds budget", h.Path)
	}
}

// Criterion 1 (helper-level): trimExcerpt caps a long string at the budget.
func TestTrimExcerpt_CapsLongString(t *testing.T) {
	require.Equal(t, "short text", trimExcerpt("  short   text  "))

	long := strings.Repeat("x", maxExcerptBytes*2)
	got := trimExcerpt(long)
	require.Equal(t, maxExcerptBytes, len(got))
}

// Criterion 2: the ripgrep path and the native fallback return equivalent
// hits ({Scope, Path, Excerpt} sets, order-independent) for the same fixture.
func TestSearch_RipgrepAndFallbackEquivalent(t *testing.T) {
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("rg not on PATH: the 'normal' store would also run the fallback, making this test vacuous")
	}

	dir := writeSearchFixture(t)

	rgStore := NewFileStore(dir, "project")
	fbStore := NewFileStore(dir, "project")
	fbStore.forceFallback = true

	rgHits, err := rgStore.Search("needle")
	require.NoError(t, err)
	fbHits, err := fbStore.Search("needle")
	require.NoError(t, err)

	require.NotEmpty(t, rgHits, "rg path should find matches")
	require.ElementsMatch(t, project(fbHits), project(rgHits))
}

// Criterion 3: each hit carries the store's scope and a Path that round-trips
// through Read; a no-match query returns an empty result and no error.
func TestSearch_ScopeAndLocatorRoundTrip(t *testing.T) {
	dir := writeSearchFixture(t)
	st := NewFileStore(dir, "project")
	st.forceFallback = true

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

// Criterion 3 (rg-path variant, guarded): a no-match query and an empty query
// both return an empty result and no error when rg is available.
func TestSearch_RipgrepEmptyResults(t *testing.T) {
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("rg not on PATH")
	}

	dir := writeSearchFixture(t)
	st := NewFileStore(dir, "project")

	noHits, err := st.Search("zzz-does-not-exist-zzz")
	require.NoError(t, err)
	require.Empty(t, noHits)

	emptyHits, err := st.Search("")
	require.NoError(t, err)
	require.Empty(t, emptyHits)
}
