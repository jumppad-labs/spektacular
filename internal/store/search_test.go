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

	hits, err := st.Search([]string{"needle"}, SearchOptions{})
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

// Criterion 3: each hit carries a Path that round-trips through Read; a
// no-match query returns an empty result and no error.
func TestSearch_LocatorRoundTrips(t *testing.T) {
	dir := writeSearchFixture(t)
	st := NewFileStore(dir, "project")

	hits, err := st.Search([]string{"needle"}, SearchOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, hits)

	for _, h := range hits {
		data, readErr := st.Read(h.Path)
		require.NoError(t, readErr, "hit Path %q should round-trip through Read", h.Path)
		require.NotEmpty(t, data)
	}

	noHits, err := st.Search([]string{"zzz-does-not-exist-zzz"}, SearchOptions{})
	require.NoError(t, err)
	require.Empty(t, noHits)
}

// Matching is case-insensitive, the store walks every directory (including
// conventions/), and a query with no terms — empty or whitespace-only —
// returns an empty result and no error.
func TestSearch_CaseInsensitiveAndIncludesAllDirectories(t *testing.T) {
	dir := writeSearchFixture(t)
	st := NewFileStore(dir, "project")

	hits, err := st.Search([]string{"needle"}, SearchOptions{})
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

	// The store is deliberately category-agnostic: it no longer excludes
	// conventions/, so its matching file is returned like any other. Skipping
	// always-applied tiers now lives in the knowledge layer (Set.Search) and is
	// tested in the knowledge package, not here.
	require.Contains(t, paths, "conventions/style.md",
		"the store should no longer exclude conventions/ from search")

	// The store no longer tokenizes, so "no terms" arrives as an empty slice.
	// Turning a query string into terms — and so deciding that "" and a
	// whitespace-only string both yield none — now belongs to knowledge.Terms
	// and is tested there.
	emptyHits, err := st.Search([]string{}, SearchOptions{})
	require.NoError(t, err)
	require.Empty(t, emptyHits)

	// A nil terms slice behaves like an allocated empty one: no hits, no error.
	blankHits, err := st.Search(nil, SearchOptions{})
	require.NoError(t, err)
	require.Empty(t, blankHits)
}

// The bare store attributes nothing: deriving a category from the path, and
// stamping the tier and name of the store a hit came from, are both the
// knowledge layer's job. The store's own label is deliberately not copied onto
// a hit, so every hit the store returns carries an empty Category, Tier, and
// Name.
func TestSearch_LeavesAttributionEmpty(t *testing.T) {
	dir := writeSearchFixture(t)
	st := NewFileStore(dir, "project:project")

	hits, err := st.Search([]string{"needle"}, SearchOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, hits)

	for _, h := range hits {
		require.Empty(t, h.Category,
			"hit %q should carry an empty Category from the store", h.Path)
		require.Empty(t, h.Tier,
			"hit %q should carry an empty Tier from the store", h.Path)
		require.Empty(t, h.Name,
			"hit %q should carry an empty Name from the store", h.Path)
	}
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

	hits, err := st.Search([]string{"needle"}, SearchOptions{})
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

	hits, err := st.Search([]string{"needle"}, SearchOptions{})
	require.NoError(t, err, "an oversized line must not fail the search")

	var paths []string
	for _, h := range hits {
		paths = append(paths, h.Path)
	}
	require.ElementsMatch(t, []string{"oversized.txt", "sibling.txt"}, paths,
		"the pre-long-line match and the sibling match should both survive")
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

	hits, err := st.Search([]string{"needle"}, SearchOptions{})
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

	hits, err := st.Search([]string{"needle"}, SearchOptions{})
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

	hits, err := st.Search([]string{"widget", "gear"}, SearchOptions{})
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

	hits, err := st.Search([]string{"needle"}, SearchOptions{})
	require.NoError(t, err)

	require.Len(t, hits, 1)
	require.Len(t, hits[0].Excerpts, 3,
		"five matching lines must be capped at three excerpts")
}

// Phase 3.1: a hit's Checksum is the hex SHA-256 over the file's exact raw
// bytes. The expected digest is an INDEPENDENT oracle produced by the
// sha256sum CLI over the exact bytes "# Title\n\nneedle here\n", hard-coded as
// a literal so the test never re-derives it through the production code.
func TestSearch_ChecksumMatchesKnownSHA256(t *testing.T) {
	dir := t.TempDir()
	fx := NewFileStore(dir, "project")
	require.NoError(t, fx.Write("doc.md", []byte("# Title\n\nneedle here\n")))

	st := NewFileStore(dir, "project")

	hits, err := st.Search([]string{"needle"}, SearchOptions{})
	require.NoError(t, err)
	require.Len(t, hits, 1)
	require.Equal(t,
		"50aeeec588de4ca1d772e3029bd0dba3522cd186003b38ae3681eecbdff804ec",
		hits[0].Checksum,
		"checksum should be the SHA-256 of the file's exact raw bytes")
}

// Phase 3.1: byte-identical files share a Checksum, while a file differing by
// exactly one byte gets a different one — the identity property de-dup relies
// on. Checksums are compared between returned hits, never recomputed in the
// test.
func TestSearch_ChecksumIdentityAndDifference(t *testing.T) {
	dir := t.TempDir()
	fx := NewFileStore(dir, "project")
	// Two byte-identical files at different paths, plus a third that differs by
	// a single byte ("needle." vs "needle,"), all containing the query term.
	require.NoError(t, fx.Write("a.md", []byte("the needle is here.\n")))
	require.NoError(t, fx.Write("b.md", []byte("the needle is here.\n")))
	require.NoError(t, fx.Write("c.md", []byte("the needle is here,\n")))

	st := NewFileStore(dir, "project")

	hits, err := st.Search([]string{"needle"}, SearchOptions{})
	require.NoError(t, err)

	sums := make(map[string]string, len(hits))
	for _, h := range hits {
		sums[h.Path] = h.Checksum
	}
	require.Len(t, sums, 3)

	require.Equal(t, sums["a.md"], sums["b.md"],
		"byte-identical files must share a checksum")
	require.NotEqual(t, sums["a.md"], sums["c.md"],
		"a one-byte difference must change the checksum")
}

// Phase 1.1 regression: a single-word query finds every shared-fixture file
// containing "needle". The oracle is the hand-maintained list of those files,
// which now includes conventions/style.md since the store is category-agnostic.
func TestSearch_SingleWordMatchesSameFiles(t *testing.T) {
	dir := writeSearchFixture(t)
	st := NewFileStore(dir, "project")

	hits, err := st.Search([]string{"needle"}, SearchOptions{})
	require.NoError(t, err)

	var paths []string
	for _, h := range hits {
		paths = append(paths, h.Path)
	}
	require.ElementsMatch(t,
		[]string{"top.txt", "nested/deep.txt", "long.txt", "conventions/style.md"}, paths)
}

// Phase 1.1 criterion 3: a storage provider reports what it found and never
// computes a score. Every hit therefore leaves Score at zero and Tier, Name and
// Category empty, while BodyCounts carries the per-term occurrence counts the
// knowledge layer ranks on. The expected counts are hand-computed from the
// fixture below, indexed like the terms slice passed to Search.
func TestSearch_ReportsEvidenceWithoutScoring(t *testing.T) {
	dir := t.TempDir()
	fx := NewFileStore(dir, "project:project")
	// widget 2 + 1 = 3, gear 1 + 1 = 2.
	require.NoError(t, fx.Write("both.txt", []byte(
		"widget widget on the first line\n"+
			"a gear sits on the second line\n"+
			"another widget beside another gear\n")))
	// Only the second term occurs, so its BodyCounts entry pins that the store
	// reports partial evidence rather than filtering the document out.
	require.NoError(t, fx.Write("gearonly.txt", []byte("just a gear in here\n")))

	st := NewFileStore(dir, "project:project")

	hits, err := st.Search([]string{"widget", "gear"}, SearchOptions{})
	require.NoError(t, err)

	// Independent oracle: hand-maintained per-term counts per path. The length
	// check pins exactly one hit per matching document.
	wantCounts := map[string][]int{
		"both.txt":     {3, 2},
		"gearonly.txt": {0, 1},
	}
	require.Len(t, hits, len(wantCounts))

	gotCounts := make(map[string][]int, len(hits))
	for _, h := range hits {
		gotCounts[h.Path] = h.BodyCounts

		require.Zero(t, h.Score,
			"hit %q must leave Score at zero; ranking is the knowledge layer's job", h.Path)
		require.Empty(t, h.Tier,
			"hit %q should carry an empty Tier from the store", h.Path)
		require.Empty(t, h.Name,
			"hit %q should carry an empty Name from the store", h.Path)
		require.Empty(t, h.Category,
			"hit %q should carry an empty Category from the store", h.Path)
	}
	require.Equal(t, wantCounts, gotCounts)
}

// Phase 2.1 criterion 4: the frontmatter block is not part of the entry's
// prose, so a tag is never also counted as a mention of itself. tagged.md
// declares the tags "go" and "http" and its body contains neither word; the
// sibling plain file mentions "go" once in its body and carries no block at
// all.
//
// Both files come back, and that is Phase 2.2's candidacy rule rather than a
// weakening of this one: declared tags are evidence in their own right, because
// an entry tagged "go" whose prose never says "go" has to reach the ranking
// layer to be scored on those tags at all. The store reports it without judging
// how relevant the tags are.
//
// The tag not being counted as prose is therefore asserted where it actually
// lives — in the counts. tagged.md's BodyCounts are {0} despite its "go" tag,
// while plain.md's are {1}, so the two files differ by exactly the one thing
// under test. That is a stronger statement than the file's earlier absence,
// which a scanner that simply failed to find anything would have satisfied just
// as well.
func TestSearch_TagsAreNotCountedAsBodyMentions(t *testing.T) {
	dir := t.TempDir()
	fx := NewFileStore(dir, "project")
	require.NoError(t, fx.Write("tagged.md", []byte(
		"---\n"+
			"tags: [go, http]\n"+
			"---\n"+
			"# Channel Notes\n"+
			"\n"+
			"Use channels for message passing.\n")))
	require.NoError(t, fx.Write("plain.md", []byte(
		"the go runtime is fast\n")))

	st := NewFileStore(dir, "project")

	hits, err := st.Search([]string{"go"}, SearchOptions{})
	require.NoError(t, err)

	counts := map[string][]int{}
	for _, h := range hits {
		counts[h.Path] = h.BodyCounts
	}
	require.Equal(t, map[string][]int{
		"tagged.md": {0},
		"plain.md":  {1},
	}, counts,
		"the tagged entry must be a candidate on its tags alone, with zero body counts; "+
			"the plain entry's single prose mention is the only body evidence in the store")
}

// Phase 2.1 criterion 4: title detection starts at the body too. The block
// below opens with a YAML comment line that would read as an ATX heading if the
// block were scanned as prose, so "Real Heading" winning proves the scanner
// starts after the block rather than at the first byte of the file.
func TestSearch_TitleComesFromBodyNotFrontmatter(t *testing.T) {
	dir := t.TempDir()
	fx := NewFileStore(dir, "project")
	require.NoError(t, fx.Write("entry.md", []byte(
		"---\n"+
			"# Block Comment\n"+
			"tags: [go, http]\n"+
			"---\n"+
			"# Real Heading\n"+
			"\n"+
			"the needle is in the body\n")))

	st := NewFileStore(dir, "project")

	hits, err := st.Search([]string{"needle"}, SearchOptions{})
	require.NoError(t, err)

	require.Len(t, hits, 1)
	require.Equal(t, "Real Heading", hits[0].Title,
		"the Title must come from the body's first heading, not a line inside the block")
	require.Equal(t, []string{"the needle is in the body"}, hits[0].Excerpts,
		"excerpts must be collected from the body only")
}

// Phase 2.1: the checksum deliberately still covers the file's exact raw bytes,
// frontmatter included, because it identifies the file rather than its prose.
// Two entries with identical bodies but different tags must therefore stay
// distinct, or exact-byte de-duplication would collapse them into one
// candidate and silently drop one entry's tags. Checksums are compared between
// returned hits, never recomputed in the test; the excerpt equality alongside
// them is what pins that the two bodies really are identical.
func TestSearch_ChecksumCoversFrontmatterSoTagsKeepEntriesDistinct(t *testing.T) {
	dir := t.TempDir()
	fx := NewFileStore(dir, "project")
	// Everything from the closing delimiter onwards is byte-identical; only the
	// tags line differs.
	const closeAndBody = "---\nthe needle is here.\n"
	require.NoError(t, fx.Write("a.md", []byte("---\ntags: [go]\n"+closeAndBody)))
	require.NoError(t, fx.Write("b.md", []byte("---\ntags: [http]\n"+closeAndBody)))

	st := NewFileStore(dir, "project")

	hits, err := st.Search([]string{"needle"}, SearchOptions{})
	require.NoError(t, err)

	sums := make(map[string]string, len(hits))
	excerpts := make(map[string][]string, len(hits))
	for _, h := range hits {
		sums[h.Path] = h.Checksum
		excerpts[h.Path] = h.Excerpts
	}
	require.Len(t, sums, 2)

	require.Equal(t, excerpts["a.md"], excerpts["b.md"],
		"the two entries must differ only in their tags, not in their bodies")
	require.NotEqual(t, sums["a.md"], sums["b.md"],
		"entries differing only in their tags must not collapse during exact-byte de-duplication")
}

// hitPaths returns the store-relative locators of hits, in the order the store
// reported them. Declared once so the tag-narrowing tests below state their
// expectation as a plain hand-written list of paths.
func hitPaths(hits []Hit) []string {
	paths := make([]string, 0, len(hits))
	for _, h := range hits {
		paths = append(paths, h.Path)
	}
	return paths
}

// Phase 2.3: CarriesEveryTag is the single definition of what "narrowed to
// these tags" means — the store applies it while walking, the knowledge layer
// applies it again after the merge — so the two sides can only agree if this
// one function is pinned exactly.
//
// The oracle is hand-written per case. Two properties are worth calling out
// because they are the ones a plausible implementation gets wrong: the relation
// is AND across want (every listed tag must be present, not any), and it is
// exact rather than the prefix relation tagAffinity uses for ranking — an entry
// tagged "https" is not an entry about "http" for the purposes of a filter,
// however much it would score for the query.
func TestCarriesEveryTag_Matrix(t *testing.T) {
	cases := []struct {
		name string
		have []string
		want []string
		ok   bool
	}{
		{"empty want matches a nil have", nil, nil, true},
		{"empty want matches an empty have", []string{}, []string{}, true},
		{"empty want matches a tagged entry", []string{"http", "routing"}, nil, true},
		{"nil have fails a non-empty want", nil, []string{"http"}, false},
		{"empty have fails a non-empty want", []string{}, []string{"http"}, false},
		{"single tag present", []string{"http", "routing"}, []string{"http"}, true},
		{"single tag absent", []string{"routing"}, []string{"http"}, false},
		{"two tags both present", []string{"go", "http", "routing"}, []string{"http", "routing"}, true},
		{"two tags only one present", []string{"http"}, []string{"http", "routing"}, false},
		{"a longer tag does not satisfy a shorter want", []string{"https"}, []string{"http"}, false},
		{"a shorter tag does not satisfy a longer want", []string{"http"}, []string{"https"}, false},
		{"duplicates in want are satisfied by one tag", []string{"http"}, []string{"http", "http"}, true},
		{"duplicates in want do not excuse an absent tag", []string{"http"}, []string{"http", "http", "routing"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.ok, CarriesEveryTag(tc.have, tc.want),
				"CarriesEveryTag(%v, %v)", tc.have, tc.want)
		})
	}
}

// Phase 2.3 criteria 1 and 2 at the store layer: SearchOptions.Tags skips
// documents during the walk. The contrast is the assertion — the same query
// with no options reports both entries, so the single-entry result under
// Tags:{"http"} can only come from the option being honoured.
//
// This is the fast path, not the authority: the knowledge layer enforces the
// same filter after the merge. A store that ignored the option would be slower,
// never wrong — which is why the knowledge-layer test is the load-bearing one
// and this only pins the optimisation.
func TestSearch_TagsOptionNarrowsTheWalk(t *testing.T) {
	dir := t.TempDir()
	fx := NewFileStore(dir, "project")
	require.NoError(t, fx.Write("tagged.md", []byte(
		"---\n"+
			"tags: [http, routing]\n"+
			"---\n"+
			"# Timeouts\n"+
			"\n"+
			"the needle is in this entry\n")))
	require.NoError(t, fx.Write("plain.md", []byte(
		"the needle is in this one too\n")))

	st := NewFileStore(dir, "project")

	all, err := st.Search([]string{"needle"}, SearchOptions{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"tagged.md", "plain.md"}, hitPaths(all),
		"without narrowing both entries carry evidence for the query")

	narrowed, err := st.Search([]string{"needle"}, SearchOptions{Tags: []string{"http"}})
	require.NoError(t, err)
	require.Equal(t, []string{"tagged.md"}, hitPaths(narrowed),
		"the untagged entry must be skipped during the walk")
}

// Phase 2.3: the narrowing is exact, so an entry tagged "https" is not reported
// for Tags:{"http"} — even though that same entry would rank for the *query*
// "http" through the prefix affinity the ranking layer applies. Filtering and
// ranking use deliberately different relations, and this pins that they do.
func TestSearch_TagsOptionMatchesExactlyNotByPrefix(t *testing.T) {
	dir := t.TempDir()
	fx := NewFileStore(dir, "project")
	require.NoError(t, fx.Write("secure.md", []byte(
		"---\n"+
			"tags: [https]\n"+
			"---\n"+
			"# Transport\n"+
			"\n"+
			"the needle is in this entry\n")))

	st := NewFileStore(dir, "project")

	hits, err := st.Search([]string{"needle"}, SearchOptions{Tags: []string{"http"}})
	require.NoError(t, err)
	require.Empty(t, hits, `an entry tagged "https" must not answer a narrowing to "http"`)
}
