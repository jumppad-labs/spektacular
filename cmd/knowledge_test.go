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
	Scope   string  `json:"scope"`
	Path    string  `json:"path"`
	Excerpt string  `json:"excerpt"`
	Score   float64 `json:"score"`
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

// Criterion 1 & 2: `knowledge search` returns scope-tagged hits carrying a
// locator and a non-empty excerpt in the {"hits":[...]} envelope.
func TestKnowledgeSearch_ReturnsScopeTaggedHits(t *testing.T) {
	twoScopeProject(t)

	stdout, _, err := runKnowledge(t, "search", "compass")
	require.NoError(t, err)

	var result struct {
		Hits []knowledgeHit `json:"hits"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	require.Len(t, result.Hits, 2)

	byScope := map[string]knowledgeHit{}
	for _, h := range result.Hits {
		byScope[h.Scope] = h
	}
	require.Equal(t, "readme.md", byScope["project"].Path)
	require.Contains(t, byScope["project"].Excerpt, "compass")
	require.Equal(t, "guidelines.md", byScope["team"].Path)
	require.NotEmpty(t, byScope["team"].Excerpt)
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
