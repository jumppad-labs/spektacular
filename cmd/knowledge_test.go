package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/output"
	"github.com/jumppad-labs/spektacular/internal/repo"
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

// knowledgeSource mirrors the knowledge.SourceInfo JSON envelope emitted by
// sources. Repo names the registry repo whose config declared the source; it
// is absent for project-owned sources.
type knowledgeSource struct {
	Scope    string `json:"scope"`
	Provider string `json:"provider"`
	Location string `json:"location"`
	Repo     string `json:"repo"`
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
// into it. Two file-backed knowledge scopes are configured: the repo-owned
// "project" scope comes from the colocated repo config's defaults (no
// repo.yaml is written, so the default project source at
// .spektacular/knowledge applies), and a project-owned "team" scope is listed
// in .spektacular/config.yaml. Each scope is seeded with a top-level file,
// and the project scope also carries a file nested in a subdirectory. The
// keyword "compass" appears in one file per scope. It returns the project
// root plus the two scope locations.
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

	cfg := "name: testproj\n" +
		"knowledge:\n" +
		"  sources:\n" +
		"    - scope: team\n" +
		"      provider: file\n" +
		"      config:\n" +
		"        location: " + teamLoc + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "config.yaml"), []byte(cfg), 0o644))

	return root, projectLoc, teamLoc
}

// runKnowledge invokes the knowledge command tree via runRoot (the same
// wrapper Execute uses) and returns the captured stdout and stderr buffers,
// reusing the setupImplementCmd harness from implement_test.go and the
// t.Chdir working-dir pattern from spec_test.go. On failure, err is the
// *output.ErrorResponse unmarshaled from stdout, matching what a real
// invocation of the CLI returns.
func runKnowledge(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	resetKnowledgeFlags(t)
	out, errBuf := setupImplementCmd(t)
	rootCmd.SetArgs(append([]string{"knowledge"}, args...))
	if code := runRoot(); code != 0 {
		var er output.ErrorResponse
		require.NoError(t, json.Unmarshal(out.Bytes(), &er))
		err = &er
	}
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
	// The repo-supplied "project" source is attributed to the implicitly
	// registered colocated repo (named after the project); the project-owned
	// "team" source carries no repo attribution.
	require.Equal(t, []knowledgeSource{
		{Scope: "project", Provider: "file", Location: projectLoc, Repo: "testproj"},
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
			require.JSONEq(t, `{"error": false, "hits": []}`, stdout)
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

// Criterion 2: a failing subcommand emits the standard ErrorResponse envelope
// on stdout (the same stream success uses) and the command reports failure.
func TestKnowledgeRead_MissingDataEmitsErrorEnvelope(t *testing.T) {
	twoScopeProject(t)

	stdout, stderr, err := runKnowledge(t, "read")
	require.Error(t, err)
	require.Empty(t, stderr)

	var envelope output.ErrorResponse
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope))
	require.True(t, envelope.IsError)
	require.Contains(t, envelope.Message, "--data is required")
}

// Criterion 2: reading from an unconfigured scope surfaces through the same
// ErrorResponse envelope.
func TestKnowledgeRead_UnknownScopeEmitsErrorEnvelope(t *testing.T) {
	twoScopeProject(t)

	stdout, stderr, err := runKnowledge(t, "read", "--data", `{"scope":"missing","path":"readme.md"}`)
	require.Error(t, err)
	require.Empty(t, stderr)

	var envelope output.ErrorResponse
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope))
	require.True(t, envelope.IsError)
	require.Contains(t, envelope.Message, "missing")
}

// alwaysAppliedProject lays out a temp project with a single file-backed
// "project" scope — supplied by the colocated repo config's defaults at
// .spektacular/knowledge, since no repo.yaml is written — seeded with one
// entry under conventions/ and one under glossary/, then chdirs into it. It
// returns the project root. This fixture is independent of twoScopeProject so
// the always-applied tests don't perturb the exact-match expectations of the
// other suites.
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

	cfg := "name: testproj\n"
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "config.yaml"), []byte(cfg), 0o644))

	return root
}

// Criterion 2: knowledge write, read, and search behave exactly as before the
// project/repo config split for a solo-repo project. A freshly initialised
// project — whose config.yaml lists no knowledge sources; the "project" scope
// comes entirely from the init-written repo.yaml — round-trips an entry
// through write, read, and search end-to-end.
func TestKnowledge_RoundTripAfterInit_SoloRepo(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	resetInitFlags(t)

	rootCmd.SetArgs([]string{"init", "claude"})
	require.NoError(t, rootCmd.Execute())

	// The colocated repo config exists and is the knowledge authority.
	require.FileExists(t, filepath.Join(dir, ".spektacular", "repo.yaml"))

	contentPath := filepath.Join(t.TempDir(), "payload.md")
	require.NoError(t, os.WriteFile(contentPath, []byte("the zanzibar protocol\n"), 0o644))

	// Write into the repo-supplied "project" scope.
	stdout, _, err := runKnowledge(t, "write",
		"--data", `{"scope":"project","path":"learnings/zanzibar.md"}`,
		"--file", contentPath)
	require.NoError(t, err)
	var wrote struct {
		Scope string `json:"scope"`
		Path  string `json:"path"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &wrote))
	require.Equal(t, "project", wrote.Scope)
	require.Equal(t, "learnings/zanzibar.md", wrote.Path)
	require.FileExists(t, filepath.Join(dir, ".spektacular", "knowledge", "learnings", "zanzibar.md"))

	// Read it back.
	stdout, _, err = runKnowledge(t, "read",
		"--data", `{"scope":"project","path":"learnings/zanzibar.md"}`)
	require.NoError(t, err)
	var read struct {
		Scope   string `json:"scope"`
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &read))
	require.Equal(t, "project", read.Scope)
	require.Equal(t, "the zanzibar protocol\n", read.Content)

	// Search finds it. "zanzibar" appears in no init-seeded README, so the
	// written entry is the only hit.
	stdout, _, err = runKnowledge(t, "search", "zanzibar")
	require.NoError(t, err)
	var searched struct {
		Hits []knowledgeHit `json:"hits"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &searched))
	require.Len(t, searched.Hits, 1)
	require.Equal(t, "project", searched.Hits[0].Scope)
	require.Equal(t, "learnings/zanzibar.md", searched.Hits[0].Path)

	// Criterion 3 (Phase 2.4): a solo-repo project's single project-scope
	// source is attributed to the colocated repo's registry name — the name
	// init recorded in config.yaml.
	cfg, err := config.FromYAMLFile(filepath.Join(dir, ".spektacular", "config.yaml"))
	require.NoError(t, err)
	stdout, _, err = runKnowledge(t, "sources")
	require.NoError(t, err)
	var listed struct {
		Sources []knowledgeSource `json:"sources"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &listed))
	require.Equal(t, []knowledgeSource{
		{Scope: "project", Provider: "file", Location: filepath.Join(dir, ".spektacular", "knowledge"), Repo: cfg.Name},
	}, listed.Sources)
}

// seedKnowledgeFile writes a knowledge entry at loc/name, creating parents.
func seedKnowledgeFile(t *testing.T, loc, name, content string) {
	t.Helper()
	full := filepath.Join(loc, filepath.FromSlash(name))
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
}

// memberRegistryProject lays out a temp project whose config registers the
// colocated repo (named "testproj", local ".") plus one member repo (named
// "member", at a sibling temp dir carrying a default repo.yaml), and declares
// a project-owned "team" source. Both repos' default project-scope knowledge
// stores exist but are empty; the caller seeds entries. It chdirs into the
// project root and returns the three roots.
func memberRegistryProject(t *testing.T) (root, member, teamLoc string) {
	t.Helper()
	root = t.TempDir()
	member = t.TempDir()
	teamLoc = filepath.Join(root, "team-kb")
	t.Chdir(root)

	// The colocated repo has no repo.yaml (it predates the config split), so
	// its store is synthesised from defaults. The member carries a default
	// repo.yaml, as `repo add` would have written.
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".spektacular", "knowledge"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(member, ".spektacular", "knowledge"), 0o755))
	require.NoError(t, os.MkdirAll(teamLoc, 0o755))
	require.NoError(t, config.NewDefaultRepoConfig().ToYAMLFile(
		filepath.Join(member, ".spektacular", config.RepoConfigFileName)))

	writeSpecCommandConfig(t, root,
		"repos:\n"+
			"  - name: testproj\n"+
			"    local: \".\"\n"+
			"  - name: member\n"+
			"    local: "+member+"\n"+
			"knowledge:\n"+
			"  sources:\n"+
			"    - scope: team\n"+
			"      provider: file\n"+
			"      config:\n"+
			"        location: "+teamLoc+"\n")

	return root, member, teamLoc
}

// memberWithoutFootprintProject lays out a temp project registering the
// colocated repo plus a member repo that exists on disk but carries no
// .spektacular footprint at all, chdirs into the project root, and returns
// both roots.
func memberWithoutFootprintProject(t *testing.T) (root, member string) {
	t.Helper()
	root = t.TempDir()
	member = t.TempDir()
	t.Chdir(root)

	require.NoError(t, os.MkdirAll(filepath.Join(root, ".spektacular", "knowledge"), 0o755))
	writeSpecCommandConfig(t, root,
		"repos:\n"+
			"  - name: testproj\n"+
			"    local: \".\"\n"+
			"  - name: member\n"+
			"    local: "+member+"\n")

	return root, member
}

// Criterion 2: `knowledge search` results include entries from every
// registered repo's sources — colocated and member — and from project-owned
// sources declared in the project config, each hit tagged with the scope of
// the source it came from. Every entry contains "beacon" exactly once, so all
// scores tie and the order is the aggregated source order: colocated repo,
// member repo, then project-owned.
func TestKnowledgeSearch_AggregatesColocatedMemberAndProjectOwnedSources(t *testing.T) {
	root, member, teamLoc := memberRegistryProject(t)
	seedKnowledgeFile(t, filepath.Join(root, ".spektacular", "knowledge"),
		"learnings/colocated-note.md", "the beacon shines in the colocated repo\n")
	seedKnowledgeFile(t, filepath.Join(member, ".spektacular", "knowledge"),
		"learnings/member-note.md", "the beacon shines in the member repo\n")
	seedKnowledgeFile(t, teamLoc, "guidelines.md", "the beacon shines in the team source\n")

	stdout, _, err := runKnowledge(t, "search", "beacon")
	require.NoError(t, err)

	var result struct {
		Hits []knowledgeHit `json:"hits"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	require.Equal(t, []knowledgeHit{
		{
			Scope:    "project",
			Path:     "learnings/colocated-note.md",
			Title:    "learnings/colocated-note.md",
			Excerpts: []string{"the beacon shines in the colocated repo"},
			Score:    1,
		},
		{
			Scope:    "project",
			Path:     "learnings/member-note.md",
			Title:    "learnings/member-note.md",
			Excerpts: []string{"the beacon shines in the member repo"},
			Score:    1,
		},
		{
			Scope:    "team",
			Path:     "guidelines.md",
			Title:    "guidelines.md",
			Excerpts: []string{"the beacon shines in the team source"},
			Score:    1,
		},
	}, result.Hits)
}

// Criterion 2: `knowledge sources` attributes each source to its repo — the
// colocated repo's source carries the project repo's registry name, the
// member's carries the member's name, and the project-owned "team" source has
// no repo field.
func TestKnowledgeSources_AttributesEachSourceToItsRepo(t *testing.T) {
	root, member, teamLoc := memberRegistryProject(t)

	stdout, _, err := runKnowledge(t, "sources")
	require.NoError(t, err)

	var result struct {
		Sources []knowledgeSource `json:"sources"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	require.Equal(t, []knowledgeSource{
		{Scope: "project", Provider: "file", Location: filepath.Join(root, ".spektacular", "knowledge"), Repo: "testproj"},
		{Scope: "project", Provider: "file", Location: filepath.Join(member, ".spektacular", "knowledge"), Repo: "member"},
		{Scope: "team", Provider: "file", Location: teamLoc},
	}, result.Sources)
}

// Duplicate scope across repos resolves by registry order: with both the
// colocated and the member repo declaring scope "project", `knowledge read`
// and `knowledge write` against that scope hit the first — colocated — repo's
// store, never the member's.
func TestKnowledgeReadWrite_DuplicateScopeResolvesToFirstRegistryRepo(t *testing.T) {
	root, member, _ := memberRegistryProject(t)
	seedKnowledgeFile(t, filepath.Join(root, ".spektacular", "knowledge"),
		"learnings/dup.md", "colocated version\n")
	seedKnowledgeFile(t, filepath.Join(member, ".spektacular", "knowledge"),
		"learnings/dup.md", "member version\n")

	// Read resolves to the colocated repo's copy.
	stdout, _, err := runKnowledge(t, "read", "--data", `{"scope":"project","path":"learnings/dup.md"}`)
	require.NoError(t, err)
	var read struct {
		Content string `json:"content"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &read))
	require.Equal(t, "colocated version\n", read.Content)

	// Write lands in the colocated repo's store, not the member's.
	contentPath := filepath.Join(t.TempDir(), "payload.md")
	require.NoError(t, os.WriteFile(contentPath, []byte("fresh entry\n"), 0o644))
	_, _, err = runKnowledge(t, "write",
		"--data", `{"scope":"project","path":"learnings/fresh.md"}`,
		"--file", contentPath)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(root, ".spektacular", "knowledge", "learnings", "fresh.md"))
	require.NoFileExists(t, filepath.Join(member, ".spektacular", "knowledge", "learnings", "fresh.md"))
}

// A member repo missing its repo.yaml breaks knowledge aggregation with the
// structured "repo_footprint" error whose next action offers `repo add` as
// the repair.
func TestKnowledgeSources_MemberMissingFootprintErrorsWithRepairOffer(t *testing.T) {
	memberWithoutFootprintProject(t)

	_, stderr, err := runKnowledge(t, "sources")
	require.Error(t, err)
	require.Empty(t, stderr)

	var envelope *output.ErrorResponse
	require.ErrorAs(t, err, &envelope)
	require.Equal(t, "repo_footprint", envelope.Code)
	require.Contains(t, envelope.Message, "member")
	require.Contains(t, envelope.NextAction, "repo add")
}

// A member repo whose repo.yaml exists but does not parse fails with the same
// "repo_footprint" error shape as a missing one.
func TestKnowledgeSources_MemberInvalidFootprintErrorsWithRepairOffer(t *testing.T) {
	_, member := memberWithoutFootprintProject(t)
	seedKnowledgeFile(t, member, ".spektacular/"+config.RepoConfigFileName, "{{{ not yaml\n")

	_, stderr, err := runKnowledge(t, "sources")
	require.Error(t, err)
	require.Empty(t, stderr)

	var envelope *output.ErrorResponse
	require.ErrorAs(t, err, &envelope)
	require.Equal(t, "repo_footprint", envelope.Code)
	require.Contains(t, envelope.NextAction, "repo add")
}

// An address-only registry entry that is not materialized locally is skipped
// by knowledge commands — they succeed with the remaining sources, never
// error, and never clone.
func TestKnowledgeSources_SkipsUnmaterializedAddressOnlyRepo(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	git := &stubGit{}
	swapRepoGit(t, git)

	require.NoError(t, os.MkdirAll(filepath.Join(root, ".spektacular", "knowledge"), 0o755))
	writeSpecCommandConfig(t, root,
		"repos:\n"+
			"  - name: testproj\n"+
			"    local: \".\"\n"+
			"  - name: ghost\n"+
			"    address: https://example.invalid/ghost.git\n")

	stdout, _, err := runKnowledge(t, "sources")
	require.NoError(t, err)

	var result struct {
		Sources []knowledgeSource `json:"sources"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	require.Equal(t, []knowledgeSource{
		{Scope: "project", Provider: "file", Location: filepath.Join(root, ".spektacular", "knowledge"), Repo: "testproj"},
	}, result.Sources, "the unmaterialized repo contributes no source")

	require.Zero(t, git.calls, "knowledge aggregation must never invoke git")
	require.NoDirExists(t, filepath.Join(root, ".spektacular", repo.MaterializeDirName, "ghost"))
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
