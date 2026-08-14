package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// writeIgnoreFile writes root/.spektacular_ignore with the given pattern lines.
func writeIgnoreFile(t *testing.T, root, lines string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(root, IgnoreFileName), []byte(lines), 0o644))
}

// ignoreProbe is one path checked against a compiled ignore file.
type ignoreProbe struct {
	path    string
	isDir   bool
	ignored bool
}

// Criterion 4: standard gitignore semantics behave as a git user expects.
// Each subtest compiles a real .spektacular_ignore file through LoadIgnore and
// probes one pattern class against the matcher.
func TestLoadIgnore_GitignoreSemanticsMatrix(t *testing.T) {
	cases := []struct {
		name   string
		lines  string
		probes []ignoreProbe
	}{
		{
			// An unanchored name matches at any depth, like git.
			name:  "simple name matches at any depth",
			lines: "secret.md\n",
			probes: []ignoreProbe{
				{path: "secret.md", ignored: true},
				{path: "sub/secret.md", ignored: true},
				{path: "other.md", ignored: false},
			},
		},
		{
			// A trailing slash matches the directory itself and everything
			// beneath it, but never a plain file of the same name.
			name:  "directory pattern excludes dir and contents but not same-named file",
			lines: "node_modules/\n",
			probes: []ignoreProbe{
				{path: "node_modules", isDir: true, ignored: true},
				{path: "node_modules/pkg/index.js", ignored: true},
				{path: "node_modules", isDir: false, ignored: false},
			},
		},
		{
			// A leading slash anchors the pattern to the source root.
			name:  "anchored pattern matches only at root",
			lines: "/top.md\n",
			probes: []ignoreProbe{
				{path: "top.md", ignored: true},
				{path: "sub/top.md", ignored: false},
			},
		},
		{
			name:  "glob matches at any depth",
			lines: "*.log\n",
			probes: []ignoreProbe{
				{path: "a.log", ignored: true},
				{path: "sub/b.log", ignored: true},
				{path: "a.log.txt", ignored: false},
			},
		},
		{
			// `a/**/b` matches zero or more intermediate directories, like git.
			name:  "double-star between names spans zero or more dirs",
			lines: "docs/**/tmp\n",
			probes: []ignoreProbe{
				{path: "docs/tmp", ignored: true},
				{path: "docs/a/tmp", ignored: true},
				{path: "docs/a/b/tmp", ignored: true},
				{path: "other/tmp", ignored: false},
			},
		},
		{
			name:  "leading double-star matches at every depth",
			lines: "**/build\n",
			probes: []ignoreProbe{
				{path: "build", isDir: true, ignored: true},
				{path: "x/build", isDir: true, ignored: true},
				{path: "x/build/out.txt", ignored: true},
			},
		},
		{
			// A later `!pattern` re-includes what an earlier pattern excluded.
			name:  "negation re-includes an earlier exclusion",
			lines: "*.md\n!keep.md\n",
			probes: []ignoreProbe{
				{path: "keep.md", ignored: false},
				{path: "sub/keep.md", ignored: false},
				{path: "drop.md", ignored: true},
			},
		},
		{
			name:  "comments and blank lines are inert",
			lines: "# secret.md\n\nreal.md\n",
			probes: []ignoreProbe{
				{path: "secret.md", ignored: false},
				{path: "real.md", ignored: true},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeIgnoreFile(t, root, tc.lines)
			m := LoadIgnore(root)
			for _, p := range tc.probes {
				require.Equal(t, p.ignored, m.Ignored(p.path, p.isDir),
					"Ignored(%q, isDir=%v)", p.path, p.isDir)
			}
		})
	}
}

// Criterion 4 (known divergence from git): real git never re-includes a path
// whose parent directory is excluded — with `dir/` then `!dir/file.md`, git
// leaves dir/file.md ignored because it does not descend into an excluded
// directory. sabhiram/go-gitignore evaluates patterns per path instead, so the
// negation wins and dir/file.md IS re-included. This test pins the library's
// observed behavior so a future matcher swap surfaces the change; it documents
// the quirk rather than endorsing it.
func TestLoadIgnore_NegationUnderExcludedParentReincludes_LibQuirk(t *testing.T) {
	root := t.TempDir()
	writeIgnoreFile(t, root, "dir/\n!dir/file.md\n")
	m := LoadIgnore(root)

	require.True(t, m.Ignored("dir", true))
	require.True(t, m.Ignored("dir/other.md", false))
	// Real git would keep this ignored; the library re-includes it.
	require.False(t, m.Ignored("dir/file.md", false))
}

// Criterion 3: a source with no ignore file yields a matcher that passes
// everything.
func TestLoadIgnore_MissingFilePassesEverything(t *testing.T) {
	m := LoadIgnore(t.TempDir())
	require.False(t, m.Ignored("anything.md", false))
	require.False(t, m.Ignored("any/dir", true))
}

// An unreadable ignore file (here: a directory squatting on the name) also
// degrades to a pass-everything matcher instead of erroring.
func TestLoadIgnore_UnreadableFilePassesEverything(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, IgnoreFileName), 0o755))
	m := LoadIgnore(root)
	require.False(t, m.Ignored("anything.md", false))
}

// ignoredFixture lays out a source directory whose .spektacular_ignore
// excludes the noise/ directory and *.log files, seeds entries on both sides
// of the exclusion, and returns the wrapped source store. The marker term
// "zebra" appears in every seeded file.
func ignoredFixture(t *testing.T) Store {
	t.Helper()
	root := t.TempDir()
	writeIgnoreFile(t, root, "noise/\n*.log\n")

	seed := func(name, content string) {
		full := filepath.Join(root, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
	}
	seed("kept.md", "zebra grazes here\n")
	seed("a.log", "zebra in a root log\n")
	seed("sub/keep.md", "zebra again\n")
	seed("sub/b.log", "zebra in a nested log\n")
	seed("noise/secret.md", "zebra hides in the noise\n")

	return NewSourceStore(root, "project")
}

// Criterion 1: List omits excluded entries at each level — the excluded
// directory and matching files disappear from their parent's listing while
// everything else, including the ignore file itself, still appears.
func TestIgnoreStore_ListOmitsExcludedEntries(t *testing.T) {
	st := ignoredFixture(t)

	entries, err := st.List("")
	require.NoError(t, err)
	require.ElementsMatch(t, []DirEntry{
		{Name: IgnoreFileName, IsDir: false},
		{Name: "kept.md", IsDir: false},
		{Name: "sub", IsDir: true},
	}, entries, "root listing must omit noise/ and a.log")

	entries, err = st.List("sub")
	require.NoError(t, err)
	require.ElementsMatch(t, []DirEntry{
		{Name: "keep.md", IsDir: false},
	}, entries, "nested listing must omit sub/b.log")
}

// Criterion 1: Search omits hits in excluded files and in files beneath an
// excluded directory.
func TestIgnoreStore_SearchOmitsExcludedHits(t *testing.T) {
	st := ignoredFixture(t)

	hits, err := st.Search("zebra")
	require.NoError(t, err)

	var paths []string
	for _, h := range hits {
		paths = append(paths, h.Path)
	}
	require.ElementsMatch(t, []string{"kept.md", "sub/keep.md"}, paths)
}

// Criterion 2: an excluded path named directly by its exact path is still
// readable — Read returns its content and Exists reports it.
func TestIgnoreStore_ExcludedPathStillReadable(t *testing.T) {
	st := ignoredFixture(t)

	require.True(t, st.Exists("noise/secret.md"))
	data, err := st.Read("noise/secret.md")
	require.NoError(t, err)
	require.Equal(t, []byte("zebra hides in the noise\n"), data)
}

// Criterion 2: an excluded path named directly by its exact path is still
// writable — Write lands under the exclusion and reads back, even though the
// path never surfaces in listings.
func TestIgnoreStore_ExcludedPathStillWritable(t *testing.T) {
	st := ignoredFixture(t)

	require.NoError(t, st.Write("noise/new.md", []byte("written past the exclusion\n")))
	data, err := st.Read("noise/new.md")
	require.NoError(t, err)
	require.Equal(t, []byte("written past the exclusion\n"), data)

	entries, err := st.List("")
	require.NoError(t, err)
	for _, e := range entries {
		require.NotEqual(t, "noise", e.Name, "the excluded dir must stay out of listings after the write")
	}
}

// Criterion 3: a source with no ignore file behaves exactly as before — the
// wrapped store's List and Search results are identical to a bare FileStore's
// over the same fixture.
func TestNewSourceStore_NoIgnoreFileMatchesBareFileStore(t *testing.T) {
	root := t.TempDir()
	seed := func(name, content string) {
		full := filepath.Join(root, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
	}
	seed("kept.md", "zebra grazes here\n")
	seed("sub/keep.md", "zebra again\n")
	seed("sub/deep/note.md", "zebra deep down\n")

	wrapped := NewSourceStore(root, "project")
	bare := NewFileStore(root, "project")

	for _, dir := range []string{"", "sub", "sub/deep"} {
		wrappedEntries, err := wrapped.List(dir)
		require.NoError(t, err)
		bareEntries, err := bare.List(dir)
		require.NoError(t, err)
		require.Equal(t, bareEntries, wrappedEntries, "List(%q)", dir)
	}

	wrappedHits, err := wrapped.Search("zebra")
	require.NoError(t, err)
	bareHits, err := bare.Search("zebra")
	require.NoError(t, err)
	require.Equal(t, bareHits, wrappedHits)
}
