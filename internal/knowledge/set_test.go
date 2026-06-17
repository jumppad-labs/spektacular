package knowledge

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/stretchr/testify/require"
)

// writeFile creates dir/name (including parents) with the given content.
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(name))
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
}

// twoScopeSet stands up two file-backed sources, "project" and "team", at
// fresh temp dirs and returns the Set plus the two backing directories. Each
// dir is seeded with a top-level file and a file nested under architecture/.
// Both scopes contain a file with the keyword "compass" so overlap can be
// asserted.
func twoScopeSet(t *testing.T) (set *Set, projectDir, teamDir string) {
	t.Helper()
	projectDir = t.TempDir()
	teamDir = t.TempDir()

	writeFile(t, projectDir, "readme.md", "project readme: the compass points north\n")
	writeFile(t, projectDir, "architecture/initial-idea.md", "an architecture note about widgets\n")

	writeFile(t, teamDir, "guidelines.md", "team guidelines reference the compass too\n")
	writeFile(t, teamDir, "architecture/overview.md", "team overview of the system\n")

	cfg := config.NewDefault()
	cfg.Knowledge.Sources = []config.SourceConfig{
		{
			Scope:    "project",
			Provider: config.ProviderFile,
			Config:   config.FileKnowledgeConfig{Location: projectDir},
		},
		{
			Scope:    "team",
			Provider: config.ProviderFile,
			Config:   config.FileKnowledgeConfig{Location: teamDir},
		},
	}

	set, err := NewSet(cfg, t.TempDir())
	require.NoError(t, err)
	return set, projectDir, teamDir
}

// Criterion 1: List, Read, and Search fan across every configured scope and
// include entries nested in subdirectories.
func TestSet_FansAcrossScopesIncludingSubdirs(t *testing.T) {
	set, _, _ := twoScopeSet(t)

	entries, err := set.List()
	require.NoError(t, err)
	require.ElementsMatch(t, []Entry{
		{Scope: "project", Path: "readme.md"},
		{Scope: "project", Path: "architecture/initial-idea.md"},
		{Scope: "team", Path: "guidelines.md"},
		{Scope: "team", Path: "architecture/overview.md"},
	}, entries)

	data, err := set.Read("team", "architecture/overview.md")
	require.NoError(t, err)
	require.Equal(t, []byte("team overview of the system\n"), data)

	hits, err := set.Search("compass")
	require.NoError(t, err)
	scopes := map[string]bool{}
	for _, h := range hits {
		scopes[h.Scope] = true
	}
	require.True(t, scopes["project"], "search should yield a project hit")
	require.True(t, scopes["team"], "search should yield a team hit")
}

// Criterion 2: an overlapping topic surfaces from both scopes, each result
// correctly tagged with its scope.
func TestSet_OverlappingEntriesTaggedPerScope(t *testing.T) {
	projectDir := t.TempDir()
	teamDir := t.TempDir()

	// Same Path "notes/topic.md" exists in both scopes, both mentioning "compass".
	writeFile(t, projectDir, "notes/topic.md", "project view: compass discussion\n")
	writeFile(t, teamDir, "notes/topic.md", "team view: compass discussion\n")

	cfg := config.NewDefault()
	cfg.Knowledge.Sources = []config.SourceConfig{
		{Scope: "project", Provider: config.ProviderFile, Config: config.FileKnowledgeConfig{Location: projectDir}},
		{Scope: "team", Provider: config.ProviderFile, Config: config.FileKnowledgeConfig{Location: teamDir}},
	}
	set, err := NewSet(cfg, t.TempDir())
	require.NoError(t, err)

	hits, err := set.Search("compass")
	require.NoError(t, err)
	hitScopes := map[string]bool{}
	for _, h := range hits {
		hitScopes[h.Scope] = true
	}
	require.True(t, hitScopes["project"], "compass hit should be tagged project")
	require.True(t, hitScopes["team"], "compass hit should be tagged team")

	entries, err := set.List()
	require.NoError(t, err)
	require.ElementsMatch(t, []Entry{
		{Scope: "project", Path: "notes/topic.md"},
		{Scope: "team", Path: "notes/topic.md"},
	}, entries)
}

// Phase 2.1 criterion 1: Search ranks hits globally by score, so a strong
// match in the later-configured "team" source outranks a weak match in the
// earlier "project" source. Scores are hand-computed from the fixtures: the
// team file holds three occurrences of "beacon", the project file one.
func TestSet_SearchRanksAcrossSourcesByScore(t *testing.T) {
	projectDir := t.TempDir()
	teamDir := t.TempDir()

	writeFile(t, projectDir, "notes.md", "a single beacon here\n")
	writeFile(t, teamDir, "signals.md", "beacon beacon\nthe beacon shines\n")

	cfg := config.NewDefault()
	cfg.Knowledge.Sources = []config.SourceConfig{
		{Scope: "project", Provider: config.ProviderFile, Config: config.FileKnowledgeConfig{Location: projectDir}},
		{Scope: "team", Provider: config.ProviderFile, Config: config.FileKnowledgeConfig{Location: teamDir}},
	}
	set, err := NewSet(cfg, t.TempDir())
	require.NoError(t, err)

	hits, err := set.Search("beacon")
	require.NoError(t, err)

	order := make([][2]string, len(hits))
	for i, h := range hits {
		order[i] = [2]string{h.Scope, h.Path}
	}
	require.Equal(t, [][2]string{
		{"team", "signals.md"},  // score 3 ranks first despite team being configured second
		{"project", "notes.md"}, // score 1
	}, order)
}

// Phase 2.1 criterion 2: equal-score hits order by configured source order then
// by path ascending, identically on every call. "lantern" occurs exactly once
// per file, and each scope pairs notes.md with notes/zz.md — names whose
// walk-discovery order (the notes/ dir is descended before the notes.md file)
// differs from path-ascending order ("notes.md" sorts before "notes/zz.md").
func TestSet_SearchTieBreaksBySourceOrderThenPath(t *testing.T) {
	projectDir := t.TempDir()
	teamDir := t.TempDir()

	writeFile(t, projectDir, "notes.md", "project lantern\n")
	writeFile(t, projectDir, "notes/zz.md", "project nested lantern\n")
	writeFile(t, teamDir, "notes.md", "team lantern\n")
	writeFile(t, teamDir, "notes/zz.md", "team nested lantern\n")

	cfg := config.NewDefault()
	cfg.Knowledge.Sources = []config.SourceConfig{
		{Scope: "project", Provider: config.ProviderFile, Config: config.FileKnowledgeConfig{Location: projectDir}},
		{Scope: "team", Provider: config.ProviderFile, Config: config.FileKnowledgeConfig{Location: teamDir}},
	}
	set, err := NewSet(cfg, t.TempDir())
	require.NoError(t, err)

	expected := [][2]string{
		{"project", "notes.md"},
		{"project", "notes/zz.md"},
		{"team", "notes.md"},
		{"team", "notes/zz.md"},
	}

	first, err := set.Search("lantern")
	require.NoError(t, err)
	second, err := set.Search("lantern")
	require.NoError(t, err)

	firstOrder := make([][2]string, len(first))
	for i, h := range first {
		firstOrder[i] = [2]string{h.Scope, h.Path}
	}
	require.Equal(t, expected, firstOrder)
	require.Equal(t, first, second, "repeated searches must return identical slices")
}

// Phase 2.1 criterion 3: a hit's Scope and Path feed straight into set.Read,
// which returns the full original fixture content. "overview" matches only the
// team scope's nested file, so the round trip crosses into a non-first source.
func TestSet_SearchHitRoundTripsThroughRead(t *testing.T) {
	set, _, _ := twoScopeSet(t)

	hits, err := set.Search("overview")
	require.NoError(t, err)
	require.Len(t, hits, 1)

	data, err := set.Read(hits[0].Scope, hits[0].Path)
	require.NoError(t, err)
	require.Equal(t, []byte("team overview of the system\n"), data)
}

// Criterion 3: a source pointing at an unreachable location fails NewSet with
// an error that names the offending scope.
func TestNewSet_UnreachableSourceFailsNamingScope(t *testing.T) {
	good := t.TempDir()
	missing := filepath.Join(t.TempDir(), "does-not-exist")

	cfg := config.NewDefault()
	cfg.Knowledge.Sources = []config.SourceConfig{
		{Scope: "project", Provider: config.ProviderFile, Config: config.FileKnowledgeConfig{Location: good}},
		{Scope: "team", Provider: config.ProviderFile, Config: config.FileKnowledgeConfig{Location: missing}},
	}

	set, err := NewSet(cfg, t.TempDir())
	require.Error(t, err)
	require.Nil(t, set)
	require.Contains(t, err.Error(), "team")
}

// Criterion 4: a write persists into exactly the chosen scope and leaves every
// other scope untouched.
func TestSet_WriteIsolatedToChosenScope(t *testing.T) {
	set, projectDir, teamDir := twoScopeSet(t)

	require.NoError(t, set.Write("project", "note.md", []byte("scoped note")))

	// File exists on disk in the project store dir only.
	require.FileExists(t, filepath.Join(projectDir, "note.md"))
	_, statErr := os.Stat(filepath.Join(teamDir, "note.md"))
	require.True(t, os.IsNotExist(statErr), "note.md must not appear in the team scope dir")

	// Reading back from project returns the content; from team it errors.
	data, err := set.Read("project", "note.md")
	require.NoError(t, err)
	require.Equal(t, []byte("scoped note"), data)

	_, err = set.Read("team", "note.md")
	require.Error(t, err)
}

// Sources reports the configured scopes, providers, and locations in order.
func TestSet_SourcesReportsConfiguredScopes(t *testing.T) {
	set, projectDir, teamDir := twoScopeSet(t)

	require.Equal(t, []SourceInfo{
		{Scope: "project", Provider: config.ProviderFile, Location: projectDir},
		{Scope: "team", Provider: config.ProviderFile, Location: teamDir},
	}, set.Sources())
}

// NewSet with no configured knowledge sources synthesises the default project
// source under projectRoot/.spektacular/knowledge.
func TestNewSet_SynthesisesDefaultProjectSource(t *testing.T) {
	projectRoot := t.TempDir()
	knowledgeDir := filepath.Join(projectRoot, config.DefaultKnowledgeLocation)
	require.NoError(t, os.MkdirAll(knowledgeDir, 0o755))

	set, err := NewSet(config.NewDefault(), projectRoot)
	require.NoError(t, err)

	require.Equal(t, []SourceInfo{
		{Scope: config.DefaultKnowledgeScope, Provider: config.ProviderFile, Location: knowledgeDir},
	}, set.Sources())
}

// Conventions returns the full body of every always-apply convention across
// every configured scope, tagged with its scope and path, including those
// nested in subdirectories. The expected set is hand-written and exact, so it
// also proves non-convention files (readme.md, architecture/*) are excluded.
func TestSet_ConventionsFanAcrossScopesIncludingSubdirs(t *testing.T) {
	set, projectDir, teamDir := twoScopeSet(t)

	writeFile(t, projectDir, "conventions/style.md", "project: use tabs not spaces\n")
	writeFile(t, projectDir, "conventions/naming/files.md", "project: kebab-case file names\n")
	writeFile(t, teamDir, "conventions/review.md", "team: two approvals required\n")

	conventions, err := set.Conventions()
	require.NoError(t, err)
	require.ElementsMatch(t, []Convention{
		{Scope: "project", Path: "conventions/style.md", Content: "project: use tabs not spaces\n"},
		{Scope: "project", Path: "conventions/naming/files.md", Content: "project: kebab-case file names\n"},
		{Scope: "team", Path: "conventions/review.md", Content: "team: two approvals required\n"},
	}, conventions)
}

// Conventions concatenates per-scope in configured order: every project-scope
// convention precedes every team-scope one. Asserted via the sequence of the
// returned .Scope fields, independent of within-scope filesystem ordering.
func TestSet_ConventionsConcatenateInConfiguredScopeOrder(t *testing.T) {
	set, projectDir, teamDir := twoScopeSet(t)

	writeFile(t, projectDir, "conventions/a.md", "project a\n")
	writeFile(t, projectDir, "conventions/b.md", "project b\n")
	writeFile(t, teamDir, "conventions/c.md", "team c\n")
	writeFile(t, teamDir, "conventions/d.md", "team d\n")

	conventions, err := set.Conventions()
	require.NoError(t, err)

	scopeSeq := make([]string, len(conventions))
	for i, c := range conventions {
		scopeSeq[i] = c.Scope
	}
	require.Equal(t, []string{"project", "project", "team", "team"}, scopeSeq)
}

// A scope without a conventions/ directory contributes nothing and produces no
// error: only the populated scope's conventions are returned.
func TestSet_ConventionsSkipScopeWithoutConventionsDir(t *testing.T) {
	set, projectDir, _ := twoScopeSet(t)

	// Only the project scope gets a conventions/ dir; team has none.
	writeFile(t, projectDir, "conventions/style.md", "project: lint before commit\n")

	conventions, err := set.Conventions()
	require.NoError(t, err)
	require.Equal(t, []Convention{
		{Scope: "project", Path: "conventions/style.md", Content: "project: lint before commit\n"},
	}, conventions)
}

// When no scope has a conventions/ directory, Conventions returns an empty
// result and no error.
func TestSet_ConventionsEmptyWhenNoScopeHasConventions(t *testing.T) {
	set, _, _ := twoScopeSet(t)

	conventions, err := set.Conventions()
	require.NoError(t, err)
	require.Empty(t, conventions)
}

// Phase 2.2 criterion: every search hit is tagged with its category, the first
// segment of its store-relative path. The fixtures live in two looked-up
// categories so the hits survive search; the expected category per path is
// hand-written.
func TestSet_SearchTagsHitsWithCategory(t *testing.T) {
	set, projectDir, _ := twoScopeSet(t)

	writeFile(t, projectDir, "gotchas/x.md", "watch out for the sextant trap\n")
	writeFile(t, projectDir, "architecture/y.md", "the sextant module wiring\n")

	hits, err := set.Search("sextant")
	require.NoError(t, err)

	byPath := map[string]string{}
	for _, h := range hits {
		byPath[h.Path] = h.Category
	}
	require.Equal(t, "gotchas", byPath["gotchas/x.md"])
	require.Equal(t, "architecture", byPath["architecture/y.md"])
}

// Phase 2.2 criterion: always-applied categories are excluded from search while
// looked-up categories are not. conventions/ and glossary/ entries that match
// the query never appear; the matching gotchas/ entry does.
func TestSet_SearchExcludesAlwaysAppliedCategories(t *testing.T) {
	set, projectDir, _ := twoScopeSet(t)

	writeFile(t, projectDir, "conventions/c.md", "the astrolabe rule\n")
	writeFile(t, projectDir, "glossary/g.md", "astrolabe: a term\n")
	writeFile(t, projectDir, "gotchas/h.md", "astrolabe gotcha\n")

	hits, err := set.Search("astrolabe")
	require.NoError(t, err)

	categories := map[string]bool{}
	for _, h := range hits {
		categories[h.Category] = true
	}
	require.False(t, categories["conventions"], "conventions must be excluded from search")
	require.False(t, categories["glossary"], "glossary must be excluded from search")
	require.True(t, categories["gotchas"], "looked-up gotchas must appear in search")
}

// Phase 2.2 load-bearing criterion: re-tiering a category in the registry is a
// single self-consistent action. Flipping gotchas to always-applied
// simultaneously makes Search exclude it AND makes AlwaysAppliedEntries load it,
// so the two behaviours can never drift. The mutation is restored in defer so no
// other test is affected. Not parallel: it mutates package state.
func TestRetier_FlipsLoadAndSearchExclusionTogether(t *testing.T) {
	set, projectDir, _ := twoScopeSet(t)

	writeFile(t, projectDir, "gotchas/trap.md", "the binnacle trap\n")

	// Before the flip: gotchas is looked-up — it appears in search and not in
	// the always-applied reader.
	hits, err := set.Search("binnacle")
	require.NoError(t, err)
	require.Len(t, hits, 1)
	require.Equal(t, "gotchas", hits[0].Category)

	entries, err := set.AlwaysAppliedEntries()
	require.NoError(t, err)
	for _, e := range entries {
		require.NotEqual(t, "gotchas", e.Category, "gotchas must not be always-applied before the flip")
	}

	// Flip gotchas to always-applied via a single registry edit, restoring the
	// exact original registry afterwards.
	original := Categories
	modified := make([]Category, len(original))
	copy(modified, original)
	for i := range modified {
		if modified[i].Name == "gotchas" {
			modified[i].Tier = TierAlwaysApplied
		}
	}
	Categories = modified
	defer func() { Categories = original }()

	// After the flip: gotchas is now excluded from search AND now loaded by the
	// always-applied reader, tagged with its category.
	hits, err = set.Search("binnacle")
	require.NoError(t, err)
	require.Empty(t, hits, "gotchas must be excluded from search after the flip")

	entries, err = set.AlwaysAppliedEntries()
	require.NoError(t, err)
	require.Contains(t, entries, AlwaysAppliedEntry{
		Scope:    "project",
		Path:     "gotchas/trap.md",
		Content:  "the binnacle trap\n",
		Category: "gotchas",
	})
}

// Phase 2.2 criterion: AlwaysAppliedEntries returns entries from every
// always-applied category — conventions AND glossary — across all scopes, each
// tagged with its category and full content. Expected values are hand-written.
func TestSet_AlwaysAppliedEntriesReturnsAllAlwaysAppliedCategories(t *testing.T) {
	set, projectDir, teamDir := twoScopeSet(t)

	writeFile(t, projectDir, "conventions/c.md", "project: use tabs\n")
	writeFile(t, projectDir, "glossary/g.md", "compass: a navigation term\n")
	writeFile(t, teamDir, "glossary/term.md", "sextant: another term\n")

	entries, err := set.AlwaysAppliedEntries()
	require.NoError(t, err)
	require.ElementsMatch(t, []AlwaysAppliedEntry{
		{Scope: "project", Path: "conventions/c.md", Content: "project: use tabs\n", Category: "conventions"},
		{Scope: "project", Path: "glossary/g.md", Content: "compass: a navigation term\n", Category: "glossary"},
		{Scope: "team", Path: "glossary/term.md", Content: "sextant: another term\n", Category: "glossary"},
	}, entries)
}
