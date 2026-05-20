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
