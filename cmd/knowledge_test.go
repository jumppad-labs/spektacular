package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// knowledgeHit mirrors the store.Hit JSON envelope emitted inside a search result.
type knowledgeHit struct {
	Scope    string   `json:"scope"`
	Path     string   `json:"path"`
	Title    string   `json:"title"`
	Excerpts []string `json:"excerpts"`
	Score    float64  `json:"score"`
}

// knowledgeEntry mirrors the knowledge.Entry JSON envelope emitted by list.
type knowledgeEntry struct {
	Scope string `json:"scope"`
	Path  string `json:"path"`
}

// knowledgeSource mirrors the knowledge.SourceInfo JSON envelope emitted by sources.
type knowledgeSource struct {
	Scope    string `json:"scope"`
	Provider string `json:"provider"`
	Location string `json:"location"`
}

// alwaysAppliedEntry mirrors the knowledge.AlwaysAppliedEntry JSON envelope
// emitted by always-applied.
type alwaysAppliedEntry struct {
	Scope    string `json:"scope"`
	Path     string `json:"path"`
	Content  string `json:"content"`
	Category string `json:"category"`
}

// knowledgeCategory mirrors the knowledge.Category JSON envelope emitted by categories.
type knowledgeCategory struct {
	Name       string `json:"name"`
	Purpose    string `json:"purpose"`
	Boundary   string `json:"boundary"`
	Tier       string `json:"tier"`
	EntryShape string `json:"entryShape"`
}

// resetKnowledgeFlags clears the persistent and per-command flags between runs
// so a flag set by one subtest does not leak into the next.
func resetKnowledgeFlags(t *testing.T) {
	t.Helper()
	reset := func() {
		require.NoError(t, knowledgeCmd.PersistentFlags().Set("schema", "false"))
		require.NoError(t, knowledgeReadCmd.Flags().Set("data", ""))
		require.NoError(t, knowledgeWriteCmd.Flags().Set("data", ""))
		require.NoError(t, knowledgeWriteCmd.Flags().Set("file", ""))
	}
	reset()
	t.Cleanup(reset)
}

// twoScopeProject lays out a temp project rooted at a t.TempDir() and chdirs
// into it. It writes a .spektacular/config.yaml configuring two file-backed
// knowledge scopes ("project" and "team"), each seeded with a top-level file
// and a file nested in a subdirectory. The keyword "compass" appears in one
// file per scope. It returns the project root plus the two scope locations.
func twoScopeProject(t *testing.T) (root, projectLoc, teamLoc string) {
	t.Helper()
	root = t.TempDir()
	t.Chdir(root)

	dataDir := filepath.Join(root, ".spektacular")
	require.NoError(t, os.MkdirAll(dataDir, 0o755))

	projectLoc = filepath.Join(dataDir, "knowledge")
	teamLoc = filepath.Join(root, "team-knowledge")

	seed := func(loc, name, content string) {
		full := filepath.Join(loc, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
	}
	seed(projectLoc, "readme.md", "project readme: the compass points north\n")
	seed(projectLoc, "architecture/initial-idea.md", "an architecture note about widgets\n")
	seed(teamLoc, "guidelines.md", "team guidelines reference the compass too\n")

	cfg := "knowledge:\n" +
		"  sources:\n" +
		"    - scope: project\n" +
		"      provider: file\n" +
		"      config:\n" +
		"        location: " + projectLoc + "\n" +
		"    - scope: team\n" +
		"      provider: file\n" +
		"      config:\n" +
		"        location: " + teamLoc + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "config.yaml"), []byte(cfg), 0o644))

	return root, projectLoc, teamLoc
}

// runKnowledge invokes the knowledge command tree via rootCmd and returns the
// captured stdout and stderr buffers, reusing the setupImplementCmd harness
// from implement_test.go and the t.Chdir working-dir pattern from spec_test.go.
func runKnowledge(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	resetKnowledgeFlags(t)
	out, errBuf := setupImplementCmd(t)
	rootCmd.SetArgs(append([]string{"knowledge"}, args...))
	err = rootCmd.Execute()
	return out.String(), errBuf.String(), err
}

// Criterion 1 & 2: `knowledge sources` lists every configured scope with its
// provider and resolved location in the documented {"sources":[...]} envelope.
func TestKnowledgeSources_ListsConfiguredScopes(t *testing.T) {
	_, projectLoc, teamLoc := twoScopeProject(t)

	stdout, _, err := runKnowledge(t, "sources")
	require.NoError(t, err)

	var result struct {
		Sources []knowledgeSource `json:"sources"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	require.Equal(t, []knowledgeSource{
		{Scope: "project", Provider: "file", Location: projectLoc},
		{Scope: "team", Provider: "file", Location: teamLoc},
	}, result.Sources)
}

// Criterion 1 & 2: `knowledge list` enumerates entries across all scopes,
// including a file nested in a subdirectory, in the {"entries":[...]} envelope.
func TestKnowledgeList_EnumeratesAllScopesIncludingNested(t *testing.T) {
	twoScopeProject(t)

	stdout, _, err := runKnowledge(t, "list")
	require.NoError(t, err)

	var result struct {
		Entries []knowledgeEntry `json:"entries"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	require.ElementsMatch(t, []knowledgeEntry{
		{Scope: "project", Path: "readme.md"},
		{Scope: "project", Path: "architecture/initial-idea.md"},
		{Scope: "team", Path: "guidelines.md"},
	}, result.Entries)
}

// Criterion 1 & 2: `knowledge search` returns scope-tagged, one-per-document
// hits carrying a locator, title, score, and excerpts in the {"hits":[...]}
// envelope. Neither fixture file has an ATX heading, so each title falls back
// to the path; "compass" occurs once per file, so both scores are 1, and the
// tie is broken by configured source order: project before team.
func TestKnowledgeSearch_ReturnsScopeTaggedHits(t *testing.T) {
	twoScopeProject(t)

	stdout, _, err := runKnowledge(t, "search", "compass")
	require.NoError(t, err)

	var result struct {
		Hits []knowledgeHit `json:"hits"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	require.Equal(t, []knowledgeHit{
		{
			Scope:    "project",
			Path:     "readme.md",
			Title:    "readme.md",
			Excerpts: []string{"project readme: the compass points north"},
			Score:    1,
		},
		{
			Scope:    "team",
			Path:     "guidelines.md",
			Title:    "guidelines.md",
			Excerpts: []string{"team guidelines reference the compass too"},
			Score:    1,
		},
	}, result.Hits)
}

// Criterion 3: an empty query and a query matching no document both succeed
// at the command level with an empty (non-nil) hits array, not an error.
func TestKnowledgeSearch_EmptyAndNoMatchQueriesReturnEmptyHits(t *testing.T) {
	twoScopeProject(t)

	for name, query := range map[string]string{
		"empty query": "",
		"no match":    "zzz-no-match-zzz",
	} {
		t.Run(name, func(t *testing.T) {
			stdout, stderr, err := runKnowledge(t, "search", query)
			require.NoError(t, err)
			require.Empty(t, stderr)
			require.JSONEq(t, `{"hits": []}`, stdout)
		})
	}
}

// Criterion 1 & 2: `knowledge read` returns the full entry content for a named
// scope and locator in the {"scope","path","content"} envelope.
func TestKnowledgeRead_ReturnsFullEntry(t *testing.T) {
	twoScopeProject(t)

	stdout, _, err := runKnowledge(t, "read", "--data", `{"scope":"project","path":"architecture/initial-idea.md"}`)
	require.NoError(t, err)

	var result struct {
		Scope   string `json:"scope"`
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	require.Equal(t, "project", result.Scope)
	require.Equal(t, "architecture/initial-idea.md", result.Path)
	require.Equal(t, "an architecture note about widgets\n", result.Content)
}

// Criterion 1 & 2: `knowledge write` persists an entry into a named scope and
// echoes the {"scope","path"} envelope; the file lands under that scope's
// configured location.
func TestKnowledgeWrite_PersistsEntry(t *testing.T) {
	_, _, teamLoc := twoScopeProject(t)

	contentPath := filepath.Join(t.TempDir(), "payload.md")
	require.NoError(t, os.WriteFile(contentPath, []byte("freshly written knowledge\n"), 0o644))

	stdout, _, err := runKnowledge(t, "write",
		"--data", `{"scope":"team","path":"learnings/new.md"}`,
		"--file", contentPath)
	require.NoError(t, err)

	var result struct {
		Scope string `json:"scope"`
		Path  string `json:"path"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	require.Equal(t, "team", result.Scope)
	require.Equal(t, "learnings/new.md", result.Path)

	persisted := filepath.Join(teamLoc, "learnings", "new.md")
	require.FileExists(t, persisted)
	data, err := os.ReadFile(persisted)
	require.NoError(t, err)
	require.Equal(t, "freshly written knowledge\n", string(data))
}

// Criterion 1 & 2: `knowledge categories` projects the category registry to the
// {"categories":[...]} envelope — every category carries a purpose, boundary,
// tier, and entry shape, and the set of names is the full, ordered model.
func TestKnowledgeCategories_ListsEveryCategoryWithDefinition(t *testing.T) {
	twoScopeProject(t)

	stdout, _, err := runKnowledge(t, "categories")
	require.NoError(t, err)

	var result struct {
		Categories []knowledgeCategory `json:"categories"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	require.Len(t, result.Categories, 6)

	var names []string
	for _, c := range result.Categories {
		require.NotEmpty(t, c.Purpose)
		require.NotEmpty(t, c.Boundary)
		require.NotEmpty(t, c.Tier)
		require.NotEmpty(t, c.EntryShape)
		names = append(names, c.Name)
	}
	require.ElementsMatch(t,
		[]string{"conventions", "glossary", "architecture", "gotchas", "learnings", "decisions"},
		names)
}

// Criterion 2: `categories --schema` declares the output envelope — categories
// is an array property.
func TestKnowledgeCategories_SchemaDeclaresCategoriesArray(t *testing.T) {
	twoScopeProject(t)

	stdout, _, err := runKnowledge(t, "categories", "--schema")
	require.NoError(t, err)

	var schema commandSchema
	require.NoError(t, json.Unmarshal([]byte(stdout), &schema))
	require.NotNil(t, schema.Output)
	require.Contains(t, schema.Output.Properties, "categories")
	require.Equal(t, "array", schema.Output.Properties["categories"].Type)
}

// Criterion 2: the --schema persistent flag prints the documented input/output
// schema envelope for a subcommand instead of running it.
func TestKnowledgeRead_SchemaDocumentsInputAndOutput(t *testing.T) {
	twoScopeProject(t)

	stdout, _, err := runKnowledge(t, "read", "--schema")
	require.NoError(t, err)

	var schema commandSchema
	require.NoError(t, json.Unmarshal([]byte(stdout), &schema))
	require.NotNil(t, schema.Input)
	require.Contains(t, schema.Input.Properties, "scope")
	require.Contains(t, schema.Input.Properties, "path")
	require.NotNil(t, schema.Output)
	require.Contains(t, schema.Output.Properties, "content")
}

// Criterion 2: `search --schema` declares the per-document hit shape that the
// command emits — hits is an array whose items carry scope, path, title,
// score, and excerpts with their documented types.
func TestKnowledgeSearch_SchemaDeclaresPerDocumentHitFields(t *testing.T) {
	twoScopeProject(t)

	stdout, _, err := runKnowledge(t, "search", "--schema", "ignored")
	require.NoError(t, err)

	var schema commandSchema
	require.NoError(t, json.Unmarshal([]byte(stdout), &schema))
	require.NotNil(t, schema.Output)
	require.Contains(t, schema.Output.Properties, "hits")

	hits := schema.Output.Properties["hits"]
	require.Equal(t, "array", hits.Type)
	require.NotNil(t, hits.Items)
	require.Equal(t, "object", hits.Items.Type)
	require.Equal(t, "string", hits.Items.Properties["scope"].Type)
	require.Equal(t, "string", hits.Items.Properties["path"].Type)
	require.Equal(t, "string", hits.Items.Properties["title"].Type)
	require.Equal(t, "number", hits.Items.Properties["score"].Type)
	require.Equal(t, "array", hits.Items.Properties["excerpts"].Type)
	require.NotNil(t, hits.Items.Properties["excerpts"].Items)
	require.Equal(t, "string", hits.Items.Properties["excerpts"].Items.Type)
	require.Equal(t, "string", hits.Items.Properties["category"].Type)
}

// Criterion 2: a failing subcommand emits the standard {"error":...} envelope
// on stderr and the command itself returns nil.
func TestKnowledgeRead_MissingDataEmitsErrorEnvelope(t *testing.T) {
	twoScopeProject(t)

	stdout, stderr, err := runKnowledge(t, "read")
	require.NoError(t, err)
	require.Empty(t, stdout)

	var envelope struct {
		Error string `json:"error"`
	}
	require.NoError(t, json.Unmarshal([]byte(stderr), &envelope))
	require.Contains(t, envelope.Error, "--data is required")
}

// Criterion 2: reading from an unconfigured scope surfaces through the same
// {"error":...} envelope.
func TestKnowledgeRead_UnknownScopeEmitsErrorEnvelope(t *testing.T) {
	twoScopeProject(t)

	_, stderr, err := runKnowledge(t, "read", "--data", `{"scope":"missing","path":"readme.md"}`)
	require.NoError(t, err)

	var envelope struct {
		Error string `json:"error"`
	}
	require.NoError(t, json.Unmarshal([]byte(stderr), &envelope))
	require.Contains(t, envelope.Error, "missing")
}

// alwaysAppliedProject lays out a temp project with a single file-backed
// "project" scope seeded with one entry under conventions/ and one under
// glossary/, then chdirs into it. It returns the project root. This fixture is
// independent of twoScopeProject so the always-applied tests don't perturb the
// exact-match expectations of the other suites.
func alwaysAppliedProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Chdir(root)

	dataDir := filepath.Join(root, ".spektacular")
	require.NoError(t, os.MkdirAll(dataDir, 0o755))

	loc := filepath.Join(dataDir, "knowledge")
	seed := func(name, content string) {
		full := filepath.Join(loc, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
	}
	seed("conventions/style.md", "always use tabs\n")
	seed("glossary/compass.md", "compass: a tool that points north\n")

	cfg := "knowledge:\n" +
		"  sources:\n" +
		"    - scope: project\n" +
		"      provider: file\n" +
		"      config:\n" +
		"        location: " + loc + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "config.yaml"), []byte(cfg), 0o644))

	return root
}

// Phase 2.3: `knowledge always-applied` returns every always-applied entry —
// both conventions and glossary — across all scopes in the {"entries":[...]}
// envelope, each tagged with its scope, path, content, and the category it came
// from so a consumer can tell a convention from a glossary term.
func TestKnowledgeAlwaysApplied_ReturnsConventionsAndGlossaryTagged(t *testing.T) {
	alwaysAppliedProject(t)

	stdout, _, err := runKnowledge(t, "always-applied")
	require.NoError(t, err)

	var result struct {
		Entries []alwaysAppliedEntry `json:"entries"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	require.ElementsMatch(t, []alwaysAppliedEntry{
		{
			Scope:    "project",
			Path:     "conventions/style.md",
			Content:  "always use tabs\n",
			Category: "conventions",
		},
		{
			Scope:    "project",
			Path:     "glossary/compass.md",
			Content:  "compass: a tool that points north\n",
			Category: "glossary",
		},
	}, result.Entries)
}

// Phase 2.3: `always-applied --schema` declares the output envelope — entries
// is an array property.
func TestKnowledgeAlwaysApplied_SchemaDeclaresEntriesArray(t *testing.T) {
	alwaysAppliedProject(t)

	stdout, _, err := runKnowledge(t, "always-applied", "--schema")
	require.NoError(t, err)

	var schema commandSchema
	require.NoError(t, json.Unmarshal([]byte(stdout), &schema))
	require.NotNil(t, schema.Output)
	require.Contains(t, schema.Output.Properties, "entries")
	require.Equal(t, "array", schema.Output.Properties["entries"].Type)
}
