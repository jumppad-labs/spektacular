package knowledge

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/output"
	"github.com/stretchr/testify/require"
)

// writeFile creates dir/name (including parents) with the given content.
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(name))
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
}

// twoScopeSet stands up two file-backed sources at fresh temp dirs and returns
// the Set plus the two backing directories. Neither source carries a stamped
// tier, so both fall back to the project tier, addressed under the names they
// declare: "project" and "team". Each dir is seeded with a top-level file and a file nested under
// architecture/. Both stores contain a file with the keyword "compass" so
// overlap can be asserted.
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
			Name:     "project",
			Provider: config.ProviderFile,
			Config:   config.FileKnowledgeConfig{Location: projectDir},
		},
		{
			Name:     "team",
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

	entries, err := set.List(Selector{Tier: TierAll})
	require.NoError(t, err)
	require.ElementsMatch(t, []Entry{
		{Tier: TierProject, Name: "project", Path: "readme.md"},
		{Tier: TierProject, Name: "project", Path: "architecture/initial-idea.md"},
		{Tier: TierProject, Name: "team", Path: "guidelines.md"},
		{Tier: TierProject, Name: "team", Path: "architecture/overview.md"},
	}, entries)

	data, err := set.Read(Address{Tier: TierProject, Name: "team"}, "architecture/overview.md")
	require.NoError(t, err)
	require.Equal(t, []byte("team overview of the system\n"), data)

	hits, err := set.Search("compass", Selector{Tier: TierAll})
	require.NoError(t, err)
	addresses := map[[2]string]bool{}
	for _, h := range hits {
		addresses[[2]string{h.Tier, h.Name}] = true
	}
	require.True(t, addresses[[2]string{"project", "project"}], "search should yield a project hit")
	require.True(t, addresses[[2]string{"project", "team"}], "search should yield a team hit")
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
		{Name: "project", Provider: config.ProviderFile, Config: config.FileKnowledgeConfig{Location: projectDir}},
		{Name: "team", Provider: config.ProviderFile, Config: config.FileKnowledgeConfig{Location: teamDir}},
	}
	set, err := NewSet(cfg, t.TempDir())
	require.NoError(t, err)

	hits, err := set.Search("compass", Selector{Tier: TierAll})
	require.NoError(t, err)
	hitAddresses := map[[2]string]bool{}
	for _, h := range hits {
		hitAddresses[[2]string{h.Tier, h.Name}] = true
	}
	require.True(t, hitAddresses[[2]string{"project", "project"}], `compass hit should be tagged tier "project", name "project"`)
	require.True(t, hitAddresses[[2]string{"project", "team"}], `compass hit should be tagged tier "project", name "team"`)

	entries, err := set.List(Selector{Tier: TierAll})
	require.NoError(t, err)
	require.ElementsMatch(t, []Entry{
		{Tier: TierProject, Name: "project", Path: "notes/topic.md"},
		{Tier: TierProject, Name: "team", Path: "notes/topic.md"},
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
		{Name: "project", Provider: config.ProviderFile, Config: config.FileKnowledgeConfig{Location: projectDir}},
		{Name: "team", Provider: config.ProviderFile, Config: config.FileKnowledgeConfig{Location: teamDir}},
	}
	set, err := NewSet(cfg, t.TempDir())
	require.NoError(t, err)

	hits, err := set.Search("beacon", Selector{Tier: TierAll})
	require.NoError(t, err)

	order := make([][3]string, len(hits))
	for i, h := range hits {
		order[i] = [3]string{h.Tier, h.Name, h.Path}
	}
	require.Equal(t, [][3]string{
		{"project", "team", "signals.md"},  // score 3 ranks first despite team being configured second
		{"project", "project", "notes.md"}, // score 1
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
		{Name: "project", Provider: config.ProviderFile, Config: config.FileKnowledgeConfig{Location: projectDir}},
		{Name: "team", Provider: config.ProviderFile, Config: config.FileKnowledgeConfig{Location: teamDir}},
	}
	set, err := NewSet(cfg, t.TempDir())
	require.NoError(t, err)

	expected := [][3]string{
		{"project", "project", "notes.md"},
		{"project", "project", "notes/zz.md"},
		{"project", "team", "notes.md"},
		{"project", "team", "notes/zz.md"},
	}

	first, err := set.Search("lantern", Selector{Tier: TierAll})
	require.NoError(t, err)
	second, err := set.Search("lantern", Selector{Tier: TierAll})
	require.NoError(t, err)

	firstOrder := make([][3]string, len(first))
	for i, h := range first {
		firstOrder[i] = [3]string{h.Tier, h.Name, h.Path}
	}
	require.Equal(t, expected, firstOrder)
	require.Equal(t, first, second, "repeated searches must return identical slices")
}

// Phase 2.1 criterion 3: a hit's Path feeds straight into set.Read at the hit's
// store address, which returns the full original fixture content. "overview"
// matches only the team store's nested file, so the round trip crosses into a
// non-first source. A hit carries its store's tier and name, which are exactly
// the two halves of the address it round-trips through.
func TestSet_SearchHitRoundTripsThroughRead(t *testing.T) {
	set, _, _ := twoScopeSet(t)

	hits, err := set.Search("overview", Selector{Tier: TierAll})
	require.NoError(t, err)
	require.Len(t, hits, 1)
	require.Equal(t, "project", hits[0].Tier)
	require.Equal(t, "team", hits[0].Name)

	data, err := set.Read(Address{Tier: Tier(hits[0].Tier), Name: hits[0].Name}, hits[0].Path)
	require.NoError(t, err)
	require.Equal(t, []byte("team overview of the system\n"), data)
}

// Criterion 3: a source pointing at an unreachable location fails NewSet with
// an error that names the offending store by its address.
func TestNewSet_UnreachableSourceFailsNamingStore(t *testing.T) {
	good := t.TempDir()
	missing := filepath.Join(t.TempDir(), "does-not-exist")

	cfg := config.NewDefault()
	cfg.Knowledge.Sources = []config.SourceConfig{
		{Name: "project", Provider: config.ProviderFile, Config: config.FileKnowledgeConfig{Location: good}},
		{Name: "team", Provider: config.ProviderFile, Config: config.FileKnowledgeConfig{Location: missing}},
	}

	set, err := NewSet(cfg, t.TempDir())
	require.Error(t, err)
	require.Nil(t, set)
	require.Contains(t, err.Error(), `knowledge store "team" in the "project" tier is unreachable`)
}

// A relative source location resolves from the folder holding config.yaml, the
// same base a repos entry uses — not from the project root. The two bases are
// one directory apart, so a config written against the wrong one silently
// addresses a sibling of the intended store rather than failing loudly; this
// asserts the base directly rather than trusting a path that happens to exist.
func TestNewSet_RelativeLocationResolvesFromTheConfigDir(t *testing.T) {
	projectRoot := t.TempDir()

	// The store lives beside config.yaml, inside .spektacular/.
	wanted := filepath.Join(projectRoot, ".spektacular", "shared-kb")
	require.NoError(t, os.MkdirAll(filepath.Join(wanted, "architecture"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(wanted, "architecture", "note.md"), []byte("from the config dir\n"), 0o644))

	// A decoy at the project root, which is where the pre-1.0 rule would have
	// looked. Resolving against the wrong base finds this one instead.
	decoy := filepath.Join(projectRoot, "shared-kb")
	require.NoError(t, os.MkdirAll(filepath.Join(decoy, "architecture"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(decoy, "architecture", "note.md"), []byte("from the project root\n"), 0o644))

	cfg := config.NewDefault()
	cfg.Knowledge.Sources = []config.SourceConfig{
		{Name: "team", Provider: config.ProviderFile, Config: config.FileKnowledgeConfig{Location: "shared-kb"}},
	}

	set, err := NewSet(cfg, projectRoot)
	require.NoError(t, err)

	sources := set.Sources()
	require.Len(t, sources, 1)
	require.Equal(t, wanted, sources[0].Location,
		"a relative knowledge location must resolve from the config.yaml folder, not the project root")

	body, err := set.Read(Address{Tier: TierProject, Name: "team"}, "architecture/note.md")
	require.NoError(t, err)
	require.Equal(t, []byte("from the config dir\n"), body,
		"the store read must come from the config-dir store, not the project-root decoy")
}

// A config written against the pre-1.0 rule points at a directory that exists,
// just one level up from where the location now resolves. The failure must name
// the corrected value outright, so the error migrates the config rather than
// only reporting that something is missing.
func TestNewSet_LegacyRelativeLocationReportsTheCorrectedValue(t *testing.T) {
	projectRoot := t.TempDir()

	// The store sits where the old rule resolved to: <projectRoot>/team-knowledge.
	legacy := filepath.Join(projectRoot, "team-knowledge")
	require.NoError(t, os.MkdirAll(legacy, 0o755))

	cfg := config.NewDefault()
	cfg.Knowledge.Sources = []config.SourceConfig{
		{Name: "team", Provider: config.ProviderFile, Config: config.FileKnowledgeConfig{Location: "team-knowledge"}},
	}

	set, err := NewSet(cfg, projectRoot)
	require.Error(t, err)
	require.Nil(t, set)

	require.Contains(t, err.Error(), `knowledge store "team" in the "project" tier is unreachable`)

	// The corrected value belongs in next_action, not the message: the message
	// names the problem, next_action gives the exact runnable fix.
	var resp *output.ErrorResponse
	require.ErrorAs(t, err, &resp)
	require.Contains(t, resp.NextAction, filepath.Join("..", "team-knowledge"),
		"next_action must name the corrected relative location, not merely report the miss")
	require.Contains(t, resp.NextAction, legacy,
		"next_action must name where the store was actually found")
}

// Criterion 4: a write persists into exactly the chosen scope and leaves every
// other scope untouched.
func TestSet_WriteIsolatedToChosenScope(t *testing.T) {
	set, projectDir, teamDir := twoScopeSet(t)

	require.NoError(t, set.Write(Address{Tier: TierProject, Name: "project"}, "note.md", []byte("scoped note")))

	// File exists on disk in the project store dir only.
	require.FileExists(t, filepath.Join(projectDir, "note.md"))
	_, statErr := os.Stat(filepath.Join(teamDir, "note.md"))
	require.True(t, os.IsNotExist(statErr), "note.md must not appear in the team scope dir")

	// Reading back from project returns the content; from team it errors.
	data, err := set.Read(Address{Tier: TierProject, Name: "project"}, "note.md")
	require.NoError(t, err)
	require.Equal(t, []byte("scoped note"), data)

	_, err = set.Read(Address{Tier: TierProject, Name: "team"}, "note.md")
	require.Error(t, err)
}

// Sources reports the configured stores by tier and name, with their providers
// and locations, in configured order.
func TestSet_SourcesReportsConfiguredStores(t *testing.T) {
	set, projectDir, teamDir := twoScopeSet(t)

	require.Equal(t, []SourceInfo{
		{Tier: TierProject, Name: "project", Provider: config.ProviderFile, Location: projectDir},
		{Tier: TierProject, Name: "team", Provider: config.ProviderFile, Location: teamDir},
	}, set.Sources())
}

// Phase 2.2 criterion 4: a project that declares no shared stores is valid.
// Nothing is synthesised in its place, so the project tier is simply empty, and
// every fan-out read restricted to that tier returns nothing rather than
// erroring.
func TestNewSet_NoDeclaredSharedStoresYieldsAnEmptyProjectTier(t *testing.T) {
	set, err := NewSet(config.NewDefault(), t.TempDir())
	require.NoError(t, err)
	require.Empty(t, set.Sources(), "a project declaring no shared stores must synthesise none")

	projectOnly := Selector{Tier: TierProject}

	entries, err := set.List(projectOnly)
	require.NoError(t, err)
	require.Empty(t, entries)

	hits, err := set.Search("compass", projectOnly)
	require.NoError(t, err)
	require.Empty(t, hits)

	applied, err := set.AlwaysAppliedEntries(projectOnly)
	require.NoError(t, err)
	require.Empty(t, applied)

	conventions, err := set.Conventions(projectOnly)
	require.NoError(t, err)
	require.Empty(t, conventions)
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

	conventions, err := set.Conventions(Selector{Tier: TierAll})
	require.NoError(t, err)
	require.ElementsMatch(t, []Convention{
		{Tier: TierProject, Name: "project", Path: "conventions/style.md", Content: "project: use tabs not spaces\n"},
		{Tier: TierProject, Name: "project", Path: "conventions/naming/files.md", Content: "project: kebab-case file names\n"},
		{Tier: TierProject, Name: "team", Path: "conventions/review.md", Content: "team: two approvals required\n"},
	}, conventions)
}

// Conventions concatenates per-scope in configured order: every project-scope
// convention precedes every team-scope one. Asserted via the sequence of the
// returned .Name fields, independent of within-store filesystem ordering.
func TestSet_ConventionsConcatenateInConfiguredScopeOrder(t *testing.T) {
	set, projectDir, teamDir := twoScopeSet(t)

	writeFile(t, projectDir, "conventions/a.md", "project a\n")
	writeFile(t, projectDir, "conventions/b.md", "project b\n")
	writeFile(t, teamDir, "conventions/c.md", "team c\n")
	writeFile(t, teamDir, "conventions/d.md", "team d\n")

	conventions, err := set.Conventions(Selector{Tier: TierAll})
	require.NoError(t, err)

	nameSeq := make([]string, len(conventions))
	for i, c := range conventions {
		nameSeq[i] = c.Name
	}
	require.Equal(t, []string{"project", "project", "team", "team"}, nameSeq)
}

// A scope without a conventions/ directory contributes nothing and produces no
// error: only the populated scope's conventions are returned.
func TestSet_ConventionsSkipScopeWithoutConventionsDir(t *testing.T) {
	set, projectDir, _ := twoScopeSet(t)

	// Only the project scope gets a conventions/ dir; team has none.
	writeFile(t, projectDir, "conventions/style.md", "project: lint before commit\n")

	conventions, err := set.Conventions(Selector{Tier: TierAll})
	require.NoError(t, err)
	require.Equal(t, []Convention{
		{Tier: TierProject, Name: "project", Path: "conventions/style.md", Content: "project: lint before commit\n"},
	}, conventions)
}

// When no scope has a conventions/ directory, Conventions returns an empty
// result and no error.
func TestSet_ConventionsEmptyWhenNoScopeHasConventions(t *testing.T) {
	set, _, _ := twoScopeSet(t)

	conventions, err := set.Conventions(Selector{Tier: TierAll})
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

	hits, err := set.Search("sextant", Selector{Tier: TierAll})
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

	hits, err := set.Search("astrolabe", Selector{Tier: TierAll})
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
	hits, err := set.Search("binnacle", Selector{Tier: TierAll})
	require.NoError(t, err)
	require.Len(t, hits, 1)
	require.Equal(t, "gotchas", hits[0].Category)

	entries, err := set.AlwaysAppliedEntries(Selector{Tier: TierAll})
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
			modified[i].Tier = CategoryTierAlwaysApplied
		}
	}
	Categories = modified
	defer func() { Categories = original }()

	// After the flip: gotchas is now excluded from search AND now loaded by the
	// always-applied reader, tagged with its category.
	hits, err = set.Search("binnacle", Selector{Tier: TierAll})
	require.NoError(t, err)
	require.Empty(t, hits, "gotchas must be excluded from search after the flip")

	entries, err = set.AlwaysAppliedEntries(Selector{Tier: TierAll})
	require.NoError(t, err)
	require.Contains(t, entries, AlwaysAppliedEntry{
		Tier:     TierProject,
		Name:     "project",
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

	entries, err := set.AlwaysAppliedEntries(Selector{Tier: TierAll})
	require.NoError(t, err)
	require.ElementsMatch(t, []AlwaysAppliedEntry{
		{Tier: TierProject, Name: "project", Path: "conventions/c.md", Content: "project: use tabs\n", Category: "conventions"},
		{Tier: TierProject, Name: "project", Path: "glossary/g.md", Content: "compass: a navigation term\n", Category: "glossary"},
		{Tier: TierProject, Name: "team", Path: "glossary/term.md", Content: "sextant: another term\n", Category: "glossary"},
	}, entries)
}

// repoScopedSet stands up three file-backed sources: one the project declares
// itself, which carries no stamped tier and so is addressed in the project tier
// under its declared name "project", and two stamped into the repo tier under
// the registry names "repo-a" and "repo-b". Each store gets one convention file
// so filtering can be asserted per store.
func repoScopedSet(t *testing.T) (set *Set, projectDir, repoADir, repoBDir string) {
	t.Helper()
	projectDir = t.TempDir()
	repoADir = t.TempDir()
	repoBDir = t.TempDir()

	writeFile(t, projectDir, "conventions/proj.md", "project: use tabs\n")
	writeFile(t, repoADir, "conventions/a.md", "repo-a: use spaces\n")
	writeFile(t, repoBDir, "conventions/b.md", "repo-b: use semicolons\n")

	cfg := config.NewDefault()
	cfg.Knowledge.Sources = []config.SourceConfig{
		{
			Name:     "project",
			Provider: config.ProviderFile,
			Config:   config.FileKnowledgeConfig{Location: projectDir},
		},
		{
			Provider: config.ProviderFile,
			Config:   config.FileKnowledgeConfig{Location: repoADir},
			Tier:     string(TierRepo),
			Name:     "repo-a",
		},
		{
			Provider: config.ProviderFile,
			Config:   config.FileKnowledgeConfig{Location: repoBDir},
			Tier:     string(TierRepo),
			Name:     "repo-b",
		},
	}

	set, err := NewSet(cfg, t.TempDir())
	require.NoError(t, err)
	return set, projectDir, repoADir, repoBDir
}

// narrowingSet stands up four file-backed stores spanning both tiers: two the
// project declares itself, addressed in the project tier as "project" and
// "team", and two contributed by registered repos, addressed in the repo tier
// as "repo-a" and "repo-b". Two stores per tier is what makes a filter naming
// several names meaningful in either tier, which repoScopedSet — one project
// store, two repo stores — cannot express; it is a separate fixture so the
// tests built on repoScopedSet keep their exact expectations.
//
// Each store carries the same three entries under its own name: an
// always-applied convention, an always-applied glossary term, and a looked-up
// gotcha holding the shared search keyword "sextant" so one query reaches every
// store.
func narrowingSet(t *testing.T) *Set {
	t.Helper()

	var sources []config.SourceConfig
	for _, s := range []struct {
		name string
		tier Tier
	}{
		{name: "project", tier: TierProject},
		{name: "team", tier: TierProject},
		{name: "repo-a", tier: TierRepo},
		{name: "repo-b", tier: TierRepo},
	} {
		dir := t.TempDir()
		writeFile(t, dir, "conventions/"+s.name+".md", s.name+" convention\n")
		writeFile(t, dir, "glossary/"+s.name+".md", s.name+" term\n")
		writeFile(t, dir, "gotchas/"+s.name+".md", s.name+" sextant gotcha\n")
		sources = append(sources, config.SourceConfig{
			Provider: config.ProviderFile,
			Config:   config.FileKnowledgeConfig{Location: dir},
			Tier:     string(s.tier),
			Name:     s.name,
		})
	}

	cfg := config.NewDefault()
	cfg.Knowledge.Sources = sources
	set, err := NewSet(cfg, t.TempDir())
	require.NoError(t, err)
	return set
}

// distinct returns the unique values in names, in first-seen order.
func distinct(names []string) []string {
	seen := map[string]bool{}
	var unique []string
	for _, name := range names {
		if seen[name] {
			continue
		}
		seen[name] = true
		unique = append(unique, name)
	}
	return unique
}

// Criterion 1: a search restricted to the repo tier returns hits only from the
// repos' own stores, one restricted to the project tier only from the shared
// stores, and one asking for both returns hits from all of them. Every store
// holds the query word exactly once, so the tier is the only thing deciding
// which hits come back.
func TestSet_SearchRestrictedToATierReturnsOnlyThatTiersHits(t *testing.T) {
	set := narrowingSet(t)

	repoHits := [][3]string{
		{"repo", "repo-a", "gotchas/repo-a.md"},
		{"repo", "repo-b", "gotchas/repo-b.md"},
	}
	projectHits := [][3]string{
		{"project", "project", "gotchas/project.md"},
		{"project", "team", "gotchas/team.md"},
	}

	for name, tc := range map[string]struct {
		tier Tier
		want [][3]string
	}{
		"repo tier":    {TierRepo, repoHits},
		"project tier": {TierProject, projectHits},
		"both tiers":   {TierAll, append(append([][3]string{}, repoHits...), projectHits...)},
	} {
		t.Run(name, func(t *testing.T) {
			hits, err := set.Search("sextant", Selector{Tier: tc.tier})
			require.NoError(t, err)

			got := make([][3]string, len(hits))
			for i, h := range hits {
				got[i] = [3]string{h.Tier, h.Name, h.Path}
			}
			require.ElementsMatch(t, tc.want, got)
		})
	}
}

// Criterion 2: the same search narrowed to one store name returns hits only
// from that store, while unnarrowed it reaches every store in its tier.
func TestSet_SearchNarrowedToOneNameReturnsOnlyThatStoresHits(t *testing.T) {
	set := narrowingSet(t)

	narrowed, err := set.Search("sextant", Selector{Tier: TierRepo, Filter: []string{"repo-a"}})
	require.NoError(t, err)
	got := make([][3]string, len(narrowed))
	for i, h := range narrowed {
		got[i] = [3]string{h.Tier, h.Name, h.Path}
	}
	require.Equal(t, [][3]string{{"repo", "repo-a", "gotchas/repo-a.md"}}, got)

	unnarrowed, err := set.Search("sextant", Selector{Tier: TierRepo})
	require.NoError(t, err)
	got = make([][3]string, len(unnarrowed))
	for i, h := range unnarrowed {
		got[i] = [3]string{h.Tier, h.Name, h.Path}
	}
	require.ElementsMatch(t, [][3]string{
		{"repo", "repo-a", "gotchas/repo-a.md"},
		{"repo", "repo-b", "gotchas/repo-b.md"},
	}, got)
}

// Criterion 3, and the "a caller can predict what a request covers" success
// metric: every tier crossed with every shape of filter — none, one name,
// several — covers exactly the stores the selector names, and search, list,
// conventions and the always-applied load all cover the same set. The expected
// coverage of each cell is hand-written, and no cell is a special case.
func TestSet_SelectorCoverageMatrixIsUniformAcrossRetrievalPaths(t *testing.T) {
	set := narrowingSet(t)

	for _, tc := range []struct {
		name string
		sel  Selector
		want []string
	}{
		{"project tier, no narrowing", Selector{Tier: TierProject}, []string{"project", "team"}},
		{"project tier, one name", Selector{Tier: TierProject, Filter: []string{"team"}}, []string{"team"}},
		{"project tier, several names", Selector{Tier: TierProject, Filter: []string{"project", "team"}}, []string{"project", "team"}},
		{"repo tier, no narrowing", Selector{Tier: TierRepo}, []string{"repo-a", "repo-b"}},
		{"repo tier, one name", Selector{Tier: TierRepo, Filter: []string{"repo-a"}}, []string{"repo-a"}},
		{"repo tier, several names", Selector{Tier: TierRepo, Filter: []string{"repo-a", "repo-b"}}, []string{"repo-a", "repo-b"}},
		{"both tiers, no narrowing", Selector{Tier: TierAll}, []string{"project", "team", "repo-a", "repo-b"}},
		{"both tiers, one name", Selector{Tier: TierAll, Filter: []string{"repo-a"}}, []string{"repo-a"}},
		{"both tiers, several names", Selector{Tier: TierAll, Filter: []string{"repo-a", "team"}}, []string{"repo-a", "team"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hits, err := set.Search("sextant", tc.sel)
			require.NoError(t, err)
			var searched []string
			for _, h := range hits {
				searched = append(searched, h.Name)
			}
			require.ElementsMatch(t, tc.want, distinct(searched), "search")

			entries, err := set.List(tc.sel)
			require.NoError(t, err)
			var listed []string
			for _, e := range entries {
				listed = append(listed, e.Name)
			}
			require.ElementsMatch(t, tc.want, distinct(listed), "list")

			conventions, err := set.Conventions(tc.sel)
			require.NoError(t, err)
			var convened []string
			for _, c := range conventions {
				convened = append(convened, c.Name)
			}
			require.ElementsMatch(t, tc.want, distinct(convened), "conventions")

			applied, err := set.AlwaysAppliedEntries(tc.sel)
			require.NoError(t, err)
			var loaded []string
			for _, e := range applied {
				loaded = append(loaded, e.Name)
			}
			require.ElementsMatch(t, tc.want, distinct(loaded), "always-applied")
		})
	}
}

// Criterion 4: the always-applied load narrowed to one repository returns that
// repository's entries and no others — none from the other repository, and none
// from either shared, project-tier store.
func TestSet_AlwaysAppliedNarrowedToOneRepoExcludesSharedStores(t *testing.T) {
	set := narrowingSet(t)

	entries, err := set.AlwaysAppliedEntries(Selector{Tier: TierRepo, Filter: []string{"repo-a"}})
	require.NoError(t, err)
	require.ElementsMatch(t, []AlwaysAppliedEntry{
		{Tier: TierRepo, Name: "repo-a", Path: "conventions/repo-a.md", Content: "repo-a convention\n", Category: "conventions"},
		{Tier: TierRepo, Name: "repo-a", Path: "glossary/repo-a.md", Content: "repo-a term\n", Category: "glossary"},
	}, entries)
}

// Criterion 5: the always-applied load across both tiers, narrowed to one
// repository plus one shared store, returns exactly those two stores' entries.
func TestSet_AlwaysAppliedAcrossTiersNarrowedToARepoAndASharedStore(t *testing.T) {
	set := narrowingSet(t)

	entries, err := set.AlwaysAppliedEntries(Selector{Tier: TierAll, Filter: []string{"repo-b", "team"}})
	require.NoError(t, err)
	require.ElementsMatch(t, []AlwaysAppliedEntry{
		{Tier: TierProject, Name: "team", Path: "conventions/team.md", Content: "team convention\n", Category: "conventions"},
		{Tier: TierProject, Name: "team", Path: "glossary/team.md", Content: "team term\n", Category: "glossary"},
		{Tier: TierRepo, Name: "repo-b", Path: "conventions/repo-b.md", Content: "repo-b convention\n", Category: "conventions"},
		{Tier: TierRepo, Name: "repo-b", Path: "glossary/repo-b.md", Content: "repo-b term\n", Category: "glossary"},
	}, entries)
}

// The "planning loads less irrelevant knowledge" success metric: with four
// stores holding always-applied entries, narrowing the load to one of them
// returns strictly fewer entries than the same load with no narrowing, and
// returns exactly that store's entries.
func TestSet_NarrowedAlwaysAppliedLoadIsStrictlySmaller(t *testing.T) {
	set := narrowingSet(t)

	all, err := set.AlwaysAppliedEntries(Selector{Tier: TierAll})
	require.NoError(t, err)
	require.Len(t, all, 8, "four stores each holding one convention and one glossary term")

	narrowed, err := set.AlwaysAppliedEntries(Selector{Tier: TierAll, Filter: []string{"team"}})
	require.NoError(t, err)
	require.Less(t, len(narrowed), len(all), "narrowing must load less, not the same")
	require.ElementsMatch(t, []AlwaysAppliedEntry{
		{Tier: TierProject, Name: "team", Path: "conventions/team.md", Content: "team convention\n", Category: "conventions"},
		{Tier: TierProject, Name: "team", Path: "glossary/team.md", Content: "team term\n", Category: "glossary"},
	}, narrowed)
}

// Criterion 6: narrowing to a name the chosen tier does not contain is refused,
// on every retrieval path, with guidance naming the names that tier does hold —
// rather than quietly returning nothing for a caller to misread as "no such
// knowledge".
func TestSet_SelectorNamingAStoreOutsideTheTierIsRefused(t *testing.T) {
	set := narrowingSet(t)

	// "team" is a real store, but it lives in the project tier, so a repo-tier
	// request cannot reach it.
	outsideTier := Selector{Tier: TierRepo, Filter: []string{"team"}}
	for name, call := range map[string]func(Selector) error{
		"search":         func(sel Selector) error { _, err := set.Search("sextant", sel); return err },
		"list":           func(sel Selector) error { _, err := set.List(sel); return err },
		"conventions":    func(sel Selector) error { _, err := set.Conventions(sel); return err },
		"always-applied": func(sel Selector) error { _, err := set.AlwaysAppliedEntries(sel); return err },
	} {
		t.Run(name, func(t *testing.T) {
			envelope := requireRefusal(t, call(outsideTier), ErrCodeStoreUnknown)
			require.Contains(t, envelope.Message, `"team"`)
			require.Contains(t, envelope.NextAction, `stores available in the "repo" tier: repo-a, repo-b`)

			// Across both tiers every configured store is offered, and no tier
			// is named, since the request narrowed to neither.
			envelope = requireRefusal(t, call(Selector{Tier: TierAll, Filter: []string{"repo-c"}}), ErrCodeStoreUnknown)
			require.Contains(t, envelope.NextAction, "stores available: project, team, repo-a, repo-b")
		})
	}
}

// Criterion 6: every retrieval path validates its selector before it opens a
// store, so a selector stating no tier at all, or an unrecognised one, is
// refused with the code saying which part is wrong.
func TestSet_RetrievalPathsRefuseAnEmptyOrUnknownTier(t *testing.T) {
	set := narrowingSet(t)

	for name, call := range map[string]func(Selector) error{
		"search":         func(sel Selector) error { _, err := set.Search("sextant", sel); return err },
		"list":           func(sel Selector) error { _, err := set.List(sel); return err },
		"conventions":    func(sel Selector) error { _, err := set.Conventions(sel); return err },
		"always-applied": func(sel Selector) error { _, err := set.AlwaysAppliedEntries(sel); return err },
	} {
		t.Run(name, func(t *testing.T) {
			requireRefusal(t, call(Selector{}), ErrCodeTierRequired)
			requireRefusal(t, call(Selector{Tier: "shared"}), ErrCodeTierInvalid)
		})
	}
}

// Criterion 1: a write whose address omits the tier, the store name, or both is
// refused, records nothing in any store, and leaves the location empty — a
// later read of that same location, at a fully-stated address, finds nothing.
func TestSet_WriteWithoutAFullAddressRecordsNothing(t *testing.T) {
	cases := map[string]struct {
		addr Address
		code string
	}{
		"tier omitted": {Address{Name: "project"}, ErrCodeTierRequired},
		"name omitted": {Address{Tier: TierProject}, ErrCodeNameRequired},
		"both omitted": {Address{}, ErrCodeTierRequired},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			set, projectDir, teamDir := twoScopeSet(t)

			requireRefusal(t, set.Write(tc.addr, "learnings/dropped.md", []byte("must not persist")), tc.code)

			require.NoFileExists(t, filepath.Join(projectDir, "learnings", "dropped.md"))
			require.NoFileExists(t, filepath.Join(teamDir, "learnings", "dropped.md"))

			for _, store := range []string{"project", "team"} {
				_, err := set.Read(Address{Tier: TierProject, Name: store}, "learnings/dropped.md")
				require.Error(t, err, "the refused write must leave nothing to read back in %q", store)
			}
		})
	}
}

// Criterion 2: the same omission is refused in a project holding exactly one
// store, where there is only one candidate an under-stated address could have
// meant. The single store is never guessed at.
func TestSet_WriteWithoutAFullAddressRefusedWithASingleStore(t *testing.T) {
	onlyDir := t.TempDir()
	cfg := config.NewDefault()
	cfg.Knowledge.Sources = []config.SourceConfig{
		{Name: "project", Provider: config.ProviderFile, Config: config.FileKnowledgeConfig{Location: onlyDir}},
	}
	set, err := NewSet(cfg, t.TempDir())
	require.NoError(t, err)

	requireRefusal(t, set.Write(Address{Name: "project"}, "learnings/x.md", []byte("no")), ErrCodeTierRequired)
	requireRefusal(t, set.Write(Address{Tier: TierProject}, "learnings/x.md", []byte("no")), ErrCodeNameRequired)
	requireRefusal(t, set.Write(Address{}, "learnings/x.md", []byte("no")), ErrCodeTierRequired)

	require.NoFileExists(t, filepath.Join(onlyDir, "learnings", "x.md"))
}

// Criterion 3: a refusal names the stores available in the tier concerned, and
// those names are exactly the ones Sources reports for that tier. The expected
// names are hand-written; the assertion that Sources agrees with them is what
// ties the guidance to the configuration a caller can inspect.
func TestSet_RefusalNamesTheStoresAvailableInTheTier(t *testing.T) {
	set, _, _, _ := repoScopedSet(t)

	var repoNames, projectNames []string
	for _, src := range set.Sources() {
		switch src.Tier {
		case TierRepo:
			repoNames = append(repoNames, src.Name)
		case TierProject:
			projectNames = append(projectNames, src.Name)
		}
	}
	require.Equal(t, []string{"repo-a", "repo-b"}, repoNames)
	require.Equal(t, []string{"project"}, projectNames)

	// An unknown name in the repo tier is told which repo-tier stores exist,
	// and is not offered the project tier's names.
	_, err := set.Read(Address{Tier: TierRepo, Name: "repo-c"}, "conventions/a.md")
	envelope := requireRefusal(t, err, ErrCodeStoreUnknown)
	require.Contains(t, envelope.NextAction, `stores available in the "repo" tier: repo-a, repo-b`)
	require.NotContains(t, envelope.NextAction, "project")

	// An omitted name in the repo tier is told the same list alongside the
	// instruction to restate the address.
	envelope = requireRefusal(t, set.Write(Address{Tier: TierRepo}, "conventions/a.md", []byte("x")), ErrCodeNameRequired)
	require.Contains(t, envelope.NextAction, `stores available in the "repo" tier: repo-a, repo-b`)

	// With no tier to narrow to, every store in either tier is offered.
	envelope = requireRefusal(t, set.Write(Address{Name: "repo-a"}, "conventions/a.md", []byte("x")), ErrCodeTierRequired)
	require.Contains(t, envelope.NextAction, `stores available: repo-a, repo-b, project`)
}

// Criterion 4: a write addressed to a named store is readable back at that same
// address, and the same location under any other name, or in the other tier,
// holds nothing.
func TestSet_WriteIsReadableOnlyAtTheAddressItWasWrittenTo(t *testing.T) {
	set, projectDir, repoADir, repoBDir := repoScopedSet(t)

	target := Address{Tier: TierRepo, Name: "repo-a"}
	require.NoError(t, set.Write(target, "learnings/note.md", []byte("repo-a note")))

	data, err := set.Read(target, "learnings/note.md")
	require.NoError(t, err)
	require.Equal(t, []byte("repo-a note"), data)
	require.FileExists(t, filepath.Join(repoADir, "learnings", "note.md"))

	// Another name in the same tier holds nothing at that location.
	_, err = set.Read(Address{Tier: TierRepo, Name: "repo-b"}, "learnings/note.md")
	require.Error(t, err)
	require.NoFileExists(t, filepath.Join(repoBDir, "learnings", "note.md"))

	// The other tier has no store by that name at all.
	_, err = set.Read(Address{Tier: TierProject, Name: "repo-a"}, "learnings/note.md")
	requireRefusal(t, err, ErrCodeStoreUnknown)
	require.NoFileExists(t, filepath.Join(projectDir, "learnings", "note.md"))
}

// Criterion 5: two repos holding an entry at the same location stay distinct —
// each name returns its own repo's content and never the other's, in either
// direction.
func TestSet_TwoRepoStoresAtTheSameLocationStayDistinct(t *testing.T) {
	set, _, repoADir, repoBDir := repoScopedSet(t)

	writeFile(t, repoADir, "learnings/dup.md", "repo-a version\n")
	writeFile(t, repoBDir, "learnings/dup.md", "repo-b version\n")

	fromA, err := set.Read(Address{Tier: TierRepo, Name: "repo-a"}, "learnings/dup.md")
	require.NoError(t, err)
	require.Equal(t, []byte("repo-a version\n"), fromA)

	fromB, err := set.Read(Address{Tier: TierRepo, Name: "repo-b"}, "learnings/dup.md")
	require.NoError(t, err)
	require.Equal(t, []byte("repo-b version\n"), fromB)

	// A write to one name overwrites only that repo's copy.
	require.NoError(t, set.Write(Address{Tier: TierRepo, Name: "repo-b"}, "learnings/dup.md", []byte("repo-b rewritten\n")))

	fromA, err = set.Read(Address{Tier: TierRepo, Name: "repo-a"}, "learnings/dup.md")
	require.NoError(t, err)
	require.Equal(t, []byte("repo-a version\n"), fromA)

	fromB, err = set.Read(Address{Tier: TierRepo, Name: "repo-b"}, "learnings/dup.md")
	require.NoError(t, err)
	require.Equal(t, []byte("repo-b rewritten\n"), fromB)
}

// Criterion 6: Read and Write both address exactly one store, so a tier that is
// neither of the two addressable tiers is rejected, and so is the fan-out tier
// "all" even though it is a declared tier.
func TestSet_ReadAndWriteRefuseFanOutAndUnknownTiers(t *testing.T) {
	set, _, repoADir, _ := repoScopedSet(t)

	for name, tier := range map[string]Tier{
		"fan-out tier": TierAll,
		"unknown tier": Tier("team"),
	} {
		t.Run(name, func(t *testing.T) {
			addr := Address{Tier: tier, Name: "repo-a"}

			_, err := set.Read(addr, "conventions/a.md")
			requireRefusal(t, err, ErrCodeTierInvalid)

			requireRefusal(t, set.Write(addr, "learnings/fanned.md", []byte("no")), ErrCodeTierInvalid)
			require.NoFileExists(t, filepath.Join(repoADir, "learnings", "fanned.md"))
		})
	}
}

// Criterion 4: a read at a correctly addressed store that holds no entry at
// the requested path is refused through the standard error envelope rather
// than escaping as a bare store error with no guidance. The refusal names the
// entry and the store it was looked for in, carries the path as the resource,
// and its next action is the exact narrowed listing that would show what that
// store does hold — the tier and the name the caller already used.
func TestSet_ReadMissingEntryOffersTheListingForThatStore(t *testing.T) {
	set, _, _, _ := repoScopedSet(t)

	_, err := set.Read(Address{Tier: TierRepo, Name: "repo-a"}, "learnings/absent.md")

	envelope := requireRefusal(t, err, ErrCodeEntryNotFound)
	require.Contains(t, envelope.Message, "learnings/absent.md")
	require.Contains(t, envelope.Message, "repo-a")
	require.Equal(t, "learnings/absent.md", envelope.Resource)
	require.Contains(t, envelope.NextAction, "knowledge list --tier repo --filter repo-a",
		"the correction must be runnable as printed, naming the store that was addressed")
	require.Contains(t, envelope.NextAction, "knowledge search")
}
