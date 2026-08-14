package cmd

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/output"
	"github.com/jumppad-labs/spektacular/internal/repo"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// repoListEntry mirrors the repoInfo JSON envelope emitted by `repo list`.
type repoListEntry struct {
	Name         string   `json:"name"`
	Address      string   `json:"address"`
	Local        string   `json:"local"`
	Root         string   `json:"root"`
	Provider     string   `json:"provider"`
	Description  string   `json:"description"`
	Role         string   `json:"role"`
	Tags         []string `json:"tags"`
	Dependencies []string `json:"dependencies"`
	Deployment   string   `json:"deployment"`
	Materialized bool     `json:"materialized"`
	StaleNote    string   `json:"stale_note"`
	MetadataNote string   `json:"metadata_note"`
}

// repoAddResult mirrors the `repo add` success envelope.
type repoAddResult struct {
	Registered   bool   `json:"registered"`
	Footprint    string `json:"footprint"`
	MetadataNote string `json:"metadata_note"`
}

// resetRepoFlags clears the persistent and per-command flags between runs so
// a flag set by one subtest does not leak into the next.
func resetRepoFlags(t *testing.T) {
	t.Helper()
	reset := func() {
		require.NoError(t, repoCmd.PersistentFlags().Set("schema", "false"))
		require.NoError(t, repoAddCmd.Flags().Set("data", ""))
	}
	reset()
	t.Cleanup(reset)
}

// runRepo invokes the repo command tree via runRoot (the same wrapper Execute
// uses) and returns the captured stdout and stderr buffers. On failure, err
// is the *output.ErrorResponse unmarshaled from stdout, matching what a real
// invocation of the CLI returns.
func runRepo(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	resetRepoFlags(t)
	out, errBuf := setupImplementCmd(t)
	rootCmd.SetArgs(append([]string{"repo"}, args...))
	if code := runRoot(); code != 0 {
		var er output.ErrorResponse
		require.NoError(t, json.Unmarshal(out.Bytes(), &er))
		err = &er
	}
	return out.String(), errBuf.String(), err
}

// repoProject lays out a minimal temp project rooted at a t.TempDir() and
// chdirs into it, returning the project root.
func repoProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	writeSpecCommandConfig(t, dir, "")
	return dir
}

// repoAddJSON marshals the given fields into a --data payload, so paths land
// in the JSON correctly escaped.
func repoAddJSON(t *testing.T, fields map[string]any) string {
	t.Helper()
	b, err := json.Marshal(fields)
	require.NoError(t, err)
	return string(b)
}

// swapRepoGit substitutes the package-level GitRunner the repo commands use,
// restoring the original when the test finishes.
func swapRepoGit(t *testing.T, g repo.GitRunner) {
	t.Helper()
	orig := repoGit
	repoGit = g
	t.Cleanup(func() { repoGit = orig })
}

// stubGit is a counting repo.GitRunner for the cmd layer: Clone only creates
// the directory, and the head queries return canned values.
type stubGit struct {
	localHead  string
	remoteHead string
	calls      int
}

func (s *stubGit) Clone(url, dir string) error {
	s.calls++
	return os.MkdirAll(dir, 0o755)
}

func (s *stubGit) LocalHead(dir string) (string, error) {
	s.calls++
	return s.localHead, nil
}

func (s *stubGit) RemoteHead(url string) (string, error) {
	s.calls++
	return s.remoteHead, nil
}

// listPaths returns every entry under root — directories and files — as
// root-relative slash paths, directories marked with a trailing slash.
func listPaths(t *testing.T, root string) []string {
	t.Helper()
	var paths []string
	require.NoError(t, filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			rel += "/"
		}
		paths = append(paths, rel)
		return nil
	}))
	return paths
}

// minimalFootprint is the exact expected content of a freshly footprinted
// repo: the repo config plus the knowledge tree, and nothing else — no agent
// guidance, no version file, no skills. The category list is hand-maintained
// to stay independent of the registry the production code scaffolds from.
var minimalFootprint = []string{
	".spektacular/",
	".spektacular/repo.yaml",
	".spektacular/knowledge/",
	".spektacular/knowledge/conventions/",
	".spektacular/knowledge/conventions/README.md",
	".spektacular/knowledge/glossary/",
	".spektacular/knowledge/glossary/README.md",
	".spektacular/knowledge/architecture/",
	".spektacular/knowledge/architecture/README.md",
	".spektacular/knowledge/gotchas/",
	".spektacular/knowledge/gotchas/README.md",
	".spektacular/knowledge/learnings/",
	".spektacular/knowledge/learnings/README.md",
	".spektacular/knowledge/decisions/",
	".spektacular/knowledge/decisions/README.md",
}

// Criterion 1: after `repo add`, the project config reflects the entry's
// name, location, and metadata, and the target repo contains exactly its
// config file and knowledge storage — nothing else.
func TestRepoAdd_RegistersEntryAndCreatesMinimalFootprint(t *testing.T) {
	project := repoProject(t)
	target := t.TempDir()

	stdout, _, err := runRepo(t, "add", "--data", repoAddJSON(t, map[string]any{
		"name":         "docs",
		"local":        target,
		"description":  "the documentation repo",
		"role":         "documentation",
		"tags":         []string{"docs", "markdown"},
		"dependencies": []string{"api"},
		"deployment":   "static site on the CDN",
	}))
	require.NoError(t, err)

	var result repoAddResult
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	require.True(t, result.Registered)
	require.Equal(t, "created", result.Footprint)

	// The project config carries the entry's membership fields only.
	cfg, err := config.FromYAMLFile(filepath.Join(project, ".spektacular", "config.yaml"))
	require.NoError(t, err)
	require.Len(t, cfg.Repos, 1)
	entry := cfg.Repos[0]
	require.Equal(t, "docs", entry.Name)
	require.Equal(t, target, entry.Local)
	require.Equal(t, []string{"api"}, entry.Dependencies)

	// The project config's own YAML carries no descriptive metadata keys at
	// all under the repo entry — RepoEntry is membership-only, so this isn't
	// just "the Go struct has no field", the serialized file itself must not
	// carry these keys either.
	rawCfg, err := os.ReadFile(filepath.Join(project, ".spektacular", "config.yaml"))
	require.NoError(t, err)
	var generic struct {
		Repos []map[string]any `yaml:"repos"`
	}
	require.NoError(t, yaml.Unmarshal(rawCfg, &generic))
	require.Len(t, generic.Repos, 1)
	for _, field := range []string{"description", "role", "tags", "deployment"} {
		require.NotContains(t, generic.Repos[0], field, "project config must carry no descriptive metadata fields")
	}

	// The target repo contains exactly the minimal footprint.
	require.ElementsMatch(t, minimalFootprint, listPaths(t, target))

	// The target repo's own config also carries the descriptive metadata: the
	// input is written back into the repo, not only into the project config.
	targetCfg, err := config.RepoConfigFromYAMLFile(filepath.Join(target, ".spektacular", config.RepoConfigFileName))
	require.NoError(t, err)
	require.Equal(t, "the documentation repo", targetCfg.Description)
	require.Equal(t, "documentation", targetCfg.Role)
	require.Equal(t, []string{"docs", "markdown"}, targetCfg.Tags)
	require.Equal(t, "static site on the CDN", targetCfg.Deployment)
}

// Criterion 2: re-running `repo add` for the same repo — from this project or
// another one — changes nothing in the target repo.
func TestRepoAdd_ReAddChangesNothingInTargetRepo(t *testing.T) {
	repoProject(t)
	target := t.TempDir()
	data := repoAddJSON(t, map[string]any{
		"name":        "docs",
		"local":       target,
		"description": "the documentation repo",
		"role":        "documentation",
	})

	_, _, err := runRepo(t, "add", "--data", data)
	require.NoError(t, err)
	before := snapshotDir(t, target)

	// Re-add from the same project: byte-identical target, "unchanged" status.
	stdout, _, err := runRepo(t, "add", "--data", data)
	require.NoError(t, err)
	var result repoAddResult
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	require.Equal(t, "unchanged", result.Footprint)
	require.Equal(t, before, snapshotDir(t, target))

	// Add the same target from a second project: the second project's config
	// gains the entry, the target repo is still untouched.
	project2 := t.TempDir()
	t.Chdir(project2)
	writeSpecCommandConfig(t, project2, "")

	stdout, _, err = runRepo(t, "add", "--data", data)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	require.Equal(t, "unchanged", result.Footprint)
	require.Equal(t, before, snapshotDir(t, target))

	cfg2, err := config.FromYAMLFile(filepath.Join(project2, ".spektacular", "config.yaml"))
	require.NoError(t, err)
	require.Len(t, cfg2.Repos, 1)
	require.Equal(t, "docs", cfg2.Repos[0].Name)
	require.Equal(t, target, cfg2.Repos[0].Local)
}

// Criterion 3: `repo list` reports every registered repo with descriptive
// metadata sourced from the repo's OWN repo.yaml (not from any project-config
// entry — RepoEntry no longer carries descriptive fields at all); an
// address-only repo that is not materialized reports metadata absent, an
// empty root, and materialized false, and listing never invokes git or
// clones it.
func TestRepoList_ReportsRegisteredReposWithMetadataAndRoots(t *testing.T) {
	project := repoProject(t)
	git := &stubGit{}
	swapRepoGit(t, git)

	// A materialized local repo carrying its own descriptive metadata in its
	// own repo.yaml.
	target := t.TempDir()
	_, _, err := runRepo(t, "add", "--data", repoAddJSON(t, map[string]any{
		"name":        "docs",
		"local":       target,
		"description": "the documentation repo",
		"role":        "documentation",
	}))
	require.NoError(t, err)

	// Register an address-only repo directly in the config: going through
	// `repo add` would materialize it by cloning, and this criterion needs an
	// entry that is registered but not on disk. RepoEntry carries no
	// descriptive fields, so there is nothing to set here beyond membership.
	cfgPath := filepath.Join(project, ".spektacular", "config.yaml")
	cfg, err := config.FromYAMLFile(cfgPath)
	require.NoError(t, err)
	cfg.Repos = append(cfg.Repos, config.RepoEntry{
		Name:    "remote-only",
		Address: "https://example.invalid/remote-only.git",
	})
	require.NoError(t, cfg.ToYAMLFile(cfgPath))

	stdout, _, err := runRepo(t, "list")
	require.NoError(t, err)

	var result struct {
		Repos []repoListEntry `json:"repos"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	require.Len(t, result.Repos, 2)

	docs := result.Repos[0]
	require.Equal(t, "docs", docs.Name)
	require.Equal(t, "the documentation repo", docs.Description)
	require.Equal(t, "documentation", docs.Role)
	require.Equal(t, target, docs.Root, "an on-disk local repo resolves to its absolute root")
	require.True(t, filepath.IsAbs(docs.Root))
	require.False(t, docs.Materialized)

	remote := result.Repos[1]
	require.Equal(t, "remote-only", remote.Name)
	require.Empty(t, remote.Description, "an unmaterialized repo has no readable repo.yaml, so metadata is absent")
	require.Empty(t, remote.Role)
	require.Equal(t, "", remote.Root, "an unmaterialized address-only repo reports no root")
	require.False(t, remote.Materialized)

	require.Zero(t, git.calls, "listing must never invoke git or clone the unmaterialized repo")
}

// Criterion: the same physical repo registered into two separate projects
// reports the identical, current description under both projects once it is
// edited a single time in the repo's own repo.yaml — descriptive metadata is
// never duplicated per-project.
func TestRepoList_SameRepoAcrossTwoProjectsSeesOneEditedDescription(t *testing.T) {
	target := t.TempDir()

	project1 := repoProject(t)
	_, _, err := runRepo(t, "add", "--data", repoAddJSON(t, map[string]any{
		"name":        "shared",
		"local":       target,
		"description": "original description",
	}))
	require.NoError(t, err)

	project2 := t.TempDir()
	t.Chdir(project2)
	writeSpecCommandConfig(t, project2, "")
	_, _, err = runRepo(t, "add", "--data", repoAddJSON(t, map[string]any{
		"name":        "shared",
		"local":       target,
		"description": "original description",
	}))
	require.NoError(t, err)

	// Edit the repo's own repo.yaml once, directly — not through either
	// project.
	repoCfgPath := filepath.Join(target, ".spektacular", config.RepoConfigFileName)
	repoCfg, err := config.RepoConfigFromYAMLFile(repoCfgPath)
	require.NoError(t, err)
	repoCfg.Description = "updated description"
	require.NoError(t, repoCfg.ToYAMLFile(repoCfgPath))

	listDescription := func(projectDir string) string {
		t.Chdir(projectDir)
		stdout, _, err := runRepo(t, "list")
		require.NoError(t, err)
		var result struct {
			Repos []repoListEntry `json:"repos"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))
		require.Len(t, result.Repos, 1)
		return result.Repos[0].Description
	}

	require.Equal(t, "updated description", listDescription(project1))
	require.Equal(t, "updated description", listDescription(project2))
}

// stale_note projection: a materialized address-only repo whose local head
// differs from its remote's carries a non-empty stale note in the listing.
func TestRepoList_StaleNoteFromDifferingHeads(t *testing.T) {
	project := repoProject(t)
	git := &stubGit{localHead: "aaa111", remoteHead: "bbb222"}
	swapRepoGit(t, git)

	writeSpecCommandConfig(t, project,
		"repos:\n"+
			"  - name: member\n"+
			"    address: https://example.com/member.git\n")

	// Materialize the clone by hand — an existing clone with a valid
	// footprint — so listing resolves it without cloning.
	clone := filepath.Join(project, ".spektacular", repo.MaterializeDirName, "member")
	require.NoError(t, os.MkdirAll(filepath.Join(clone, ".spektacular"), 0o755))
	require.NoError(t, config.NewDefaultRepoConfig().ToYAMLFile(
		filepath.Join(clone, ".spektacular", config.RepoConfigFileName)))

	stdout, _, err := runRepo(t, "list")
	require.NoError(t, err)

	var result struct {
		Repos []repoListEntry `json:"repos"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	require.Len(t, result.Repos, 1)

	member := result.Repos[0]
	require.Equal(t, "member", member.Name)
	require.Equal(t, clone, member.Root)
	require.True(t, member.Materialized)
	require.Contains(t, member.StaleNote, "aaa111")
	require.Contains(t, member.StaleNote, "bbb222")
}

// upsert: re-adding a registered name with different metadata updates the
// config entry in place, leaves the target repo's footprint shape (its set of
// files) unchanged, but writes the new descriptive metadata into the target
// repo's own repo.yaml.
func TestRepoAdd_UpsertUpdatesMetadataAndRepoConfigWithoutChangingFootprintShape(t *testing.T) {
	project := repoProject(t)
	target := t.TempDir()

	_, _, err := runRepo(t, "add", "--data", repoAddJSON(t, map[string]any{
		"name":        "docs",
		"local":       target,
		"description": "first description",
		"role":        "documentation",
		"tags":        []string{"docs"},
		"deployment":  "static site",
	}))
	require.NoError(t, err)
	beforePaths := listPaths(t, target)

	stdout, _, err := runRepo(t, "add", "--data", repoAddJSON(t, map[string]any{
		"name":        "docs",
		"local":       target,
		"description": "second description",
		"role":        "reference",
		"tags":        []string{"docs", "api"},
		"deployment":  "static site v2",
	}))
	require.NoError(t, err)

	var result repoAddResult
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	require.True(t, result.Registered)
	require.Equal(t, "unchanged", result.Footprint)

	cfg, err := config.FromYAMLFile(filepath.Join(project, ".spektacular", "config.yaml"))
	require.NoError(t, err)
	require.Len(t, cfg.Repos, 1, "an upsert must update in place, not append a duplicate")
	require.Equal(t, "docs", cfg.Repos[0].Name)

	// The footprint's file set is unchanged — same paths present, still the
	// minimal footprint shape.
	require.ElementsMatch(t, beforePaths, listPaths(t, target))

	// But repo.yaml's content legitimately changed: the target repo's own
	// config now carries the new descriptive metadata.
	targetCfg, err := config.RepoConfigFromYAMLFile(filepath.Join(target, ".spektacular", config.RepoConfigFileName))
	require.NoError(t, err)
	require.Equal(t, "second description", targetCfg.Description)
	require.Equal(t, "reference", targetCfg.Role)
	require.Equal(t, []string{"docs", "api"}, targetCfg.Tags)
	require.Equal(t, "static site v2", targetCfg.Deployment)
}

// Criterion 3: registering a repository with no descriptive metadata
// supplied leaves an already-described repository's own configuration
// unchanged — a re-add that omits description/role/tags/deployment must not
// blank out metadata written by an earlier, fuller registration.
func TestRepoAdd_ReAddWithNoMetadataLeavesRepoConfigUnchanged(t *testing.T) {
	repoProject(t)
	target := t.TempDir()

	_, _, err := runRepo(t, "add", "--data", repoAddJSON(t, map[string]any{
		"name":        "docs",
		"local":       target,
		"description": "the documentation repo",
		"role":        "documentation",
		"tags":        []string{"docs", "markdown"},
		"deployment":  "static site on the CDN",
	}))
	require.NoError(t, err)

	// Re-register the same repo supplying only name + local — no descriptive
	// fields at all.
	_, _, err = runRepo(t, "add", "--data", repoAddJSON(t, map[string]any{
		"name":  "docs",
		"local": target,
	}))
	require.NoError(t, err)

	targetCfg, err := config.RepoConfigFromYAMLFile(filepath.Join(target, ".spektacular", config.RepoConfigFileName))
	require.NoError(t, err)
	require.Equal(t, "the documentation repo", targetCfg.Description, "an omitted description must not blank out an existing one")
	require.Equal(t, "documentation", targetCfg.Role)
	require.Equal(t, []string{"docs", "markdown"}, targetCfg.Tags)
	require.Equal(t, "static site on the CDN", targetCfg.Deployment)
}

// `repo add` with neither address nor local fails validation, naming the repo.
func TestRepoAdd_NeitherAddressNorLocalErrors(t *testing.T) {
	repoProject(t)

	_, stderr, err := runRepo(t, "add", "--data", `{"name":"ghost"}`)
	require.Error(t, err)
	require.Empty(t, stderr)
	require.Contains(t, err.Error(), "ghost")
	require.Contains(t, err.Error(), "address or local")
}

// `repo add` with no --data fails with a message carrying a --data example.
func TestRepoAdd_MissingDataErrors(t *testing.T) {
	repoProject(t)

	stdout, stderr, err := runRepo(t, "add")
	require.Error(t, err)
	require.Empty(t, stderr)

	var envelope output.ErrorResponse
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope))
	require.True(t, envelope.IsError)
	require.Contains(t, envelope.Message, "--data is required")
	require.Contains(t, envelope.Message, `"name"`, "the error must carry a usable --data example")
}

// `repo add --schema` prints the documented input/output schema instead of
// running the command.
func TestRepoAdd_SchemaDocumentsInputAndOutput(t *testing.T) {
	repoProject(t)

	stdout, _, err := runRepo(t, "add", "--schema")
	require.NoError(t, err)

	var schema commandSchema
	require.NoError(t, json.Unmarshal([]byte(stdout), &schema))
	require.NotNil(t, schema.Input)
	for _, field := range []string{"name", "address", "local", "description", "role", "tags", "dependencies", "deployment"} {
		require.Contains(t, schema.Input.Properties, field)
	}
	require.Equal(t, []string{"name"}, schema.Input.Required)
	require.NotNil(t, schema.Output)
	require.Contains(t, schema.Output.Properties, "registered")
	require.Contains(t, schema.Output.Properties, "footprint")
	require.ElementsMatch(t, []string{"created", "repaired", "unchanged"}, schema.Output.Properties["footprint"].Enum)
}

// `repo list --schema` declares the per-repo projection the listing emits.
func TestRepoList_SchemaDeclaresReposArray(t *testing.T) {
	repoProject(t)

	stdout, _, err := runRepo(t, "list", "--schema")
	require.NoError(t, err)

	var schema commandSchema
	require.NoError(t, json.Unmarshal([]byte(stdout), &schema))
	require.NotNil(t, schema.Output)
	require.Contains(t, schema.Output.Properties, "repos")

	repos := schema.Output.Properties["repos"]
	require.Equal(t, "array", repos.Type)
	require.NotNil(t, repos.Items)
	for _, field := range []string{"name", "root", "materialized", "stale_note"} {
		require.Contains(t, repos.Items.Properties, field)
	}
}

// `repo add` with no descriptive metadata supplied returns a metadata_note
// naming the repo, nudging the user toward describing it.
func TestRepoAdd_NoMetadataReturnsMetadataNote(t *testing.T) {
	repoProject(t)
	target := t.TempDir()

	stdout, _, err := runRepo(t, "add", "--data", repoAddJSON(t, map[string]any{
		"name":  "docs",
		"local": target,
	}))
	require.NoError(t, err)

	var result repoAddResult
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	require.NotEmpty(t, result.MetadataNote, "registering with no descriptive metadata must produce a metadata_note")
	require.Contains(t, result.MetadataNote, "docs", "the note must name the repo")
}

// `repo add` with descriptive metadata supplied returns no metadata_note.
func TestRepoAdd_WithMetadataReturnsNoMetadataNote(t *testing.T) {
	repoProject(t)
	target := t.TempDir()

	stdout, _, err := runRepo(t, "add", "--data", repoAddJSON(t, map[string]any{
		"name":        "docs",
		"local":       target,
		"description": "the documentation repo",
		"role":        "documentation",
	}))
	require.NoError(t, err)

	var result repoAddResult
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	require.Empty(t, result.MetadataNote, "registering with descriptive metadata must not produce a metadata_note")
}

// `repo list` reports a metadata_note for a materialized repo whose own
// repo.yaml carries no descriptive metadata.
func TestRepoList_MaterializedNoMetadataReturnsMetadataNote(t *testing.T) {
	repoProject(t)
	target := t.TempDir()

	_, _, err := runRepo(t, "add", "--data", repoAddJSON(t, map[string]any{
		"name":  "docs",
		"local": target,
	}))
	require.NoError(t, err)

	stdout, _, err := runRepo(t, "list")
	require.NoError(t, err)

	var result struct {
		Repos []repoListEntry `json:"repos"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	require.Len(t, result.Repos, 1)
	require.NotEmpty(t, result.Repos[0].MetadataNote, "a materialized repo with no descriptive metadata must carry a metadata_note")
	require.Contains(t, result.Repos[0].MetadataNote, "docs")
}

// `repo list` reports no metadata_note for a materialized repo whose own
// repo.yaml carries descriptive metadata.
func TestRepoList_MaterializedWithMetadataReturnsNoMetadataNote(t *testing.T) {
	repoProject(t)
	target := t.TempDir()

	_, _, err := runRepo(t, "add", "--data", repoAddJSON(t, map[string]any{
		"name":        "docs",
		"local":       target,
		"description": "the documentation repo",
		"role":        "documentation",
	}))
	require.NoError(t, err)

	stdout, _, err := runRepo(t, "list")
	require.NoError(t, err)

	var result struct {
		Repos []repoListEntry `json:"repos"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	require.Len(t, result.Repos, 1)
	require.Empty(t, result.Repos[0].MetadataNote, "a materialized repo with descriptive metadata must carry no metadata_note")
}

// `repo list` reports no metadata_note for an unmaterialized, address-only
// repo: the note only makes sense once a repo can actually be inspected, and
// an entry not yet fetched already reports no metadata for an unrelated
// reason (it hasn't been cloned), so nagging about it here would be
// premature and misleading.
func TestRepoList_UnmaterializedReturnsNoMetadataNote(t *testing.T) {
	project := repoProject(t)
	git := &stubGit{}
	swapRepoGit(t, git)

	cfgPath := filepath.Join(project, ".spektacular", "config.yaml")
	cfg, err := config.FromYAMLFile(cfgPath)
	require.NoError(t, err)
	cfg.Repos = append(cfg.Repos, config.RepoEntry{
		Name:    "remote-only",
		Address: "https://example.invalid/remote-only.git",
	})
	require.NoError(t, cfg.ToYAMLFile(cfgPath))

	stdout, _, err := runRepo(t, "list")
	require.NoError(t, err)

	var result struct {
		Repos []repoListEntry `json:"repos"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	require.Len(t, result.Repos, 1)
	require.False(t, result.Repos[0].Materialized)
	require.Empty(t, result.Repos[0].MetadataNote, "an unmaterialized repo must carry no metadata_note")
	require.Zero(t, git.calls, "listing must never invoke git or clone the unmaterialized repo")
}
