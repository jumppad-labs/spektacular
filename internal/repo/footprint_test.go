package repo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/stretchr/testify/require"
)

// footprintCategories is the hand-maintained list of knowledge categories a
// footprint must scaffold, kept independent of the registry the production
// code reads.
var footprintCategories = []string{
	"conventions", "glossary", "architecture", "gotchas", "learnings", "decisions",
}

// EnsureFootprint on a directory with no repo.yaml creates the full minimal
// footprint: a valid repo config plus a directory and README for every
// knowledge category.
func TestEnsureFootprint_FreshDirCreates(t *testing.T) {
	root := t.TempDir()

	status, err := EnsureFootprint(root, config.NewDefaultRepoConfig())
	require.NoError(t, err)
	require.Equal(t, FootprintCreated, status)

	// The written repo.yaml parses back as a valid repo config.
	_, err = config.RepoConfigFromYAMLFile(filepath.Join(root, ".spektacular", config.RepoConfigFileName))
	require.NoError(t, err)

	for _, cat := range footprintCategories {
		dir := filepath.Join(root, ".spektacular", "knowledge", cat)
		require.DirExists(t, dir)
		readme := filepath.Join(dir, "README.md")
		require.FileExists(t, readme)
		content, err := os.ReadFile(readme)
		require.NoError(t, err)
		require.NotEmpty(t, content)
	}
}

// A deleted README is recreated on the next run, and files that were not
// missing are left exactly as they were — repair is strictly additive.
func TestEnsureFootprint_MissingREADMERepairedWithoutRewritingOthers(t *testing.T) {
	root := t.TempDir()
	_, err := EnsureFootprint(root, config.NewDefaultRepoConfig())
	require.NoError(t, err)

	// A user-customised README is the sentinel: repair must not rewrite it.
	sentinel := filepath.Join(root, ".spektacular", "knowledge", "conventions", "README.md")
	require.NoError(t, os.WriteFile(sentinel, []byte("customised by hand\n"), 0o644))

	missing := filepath.Join(root, ".spektacular", "knowledge", "gotchas", "README.md")
	require.NoError(t, os.Remove(missing))

	status, err := EnsureFootprint(root, config.NewDefaultRepoConfig())
	require.NoError(t, err)
	require.Equal(t, FootprintRepaired, status)

	require.FileExists(t, missing)
	content, err := os.ReadFile(sentinel)
	require.NoError(t, err)
	require.Equal(t, "customised by hand\n", string(content), "an existing knowledge file must never be overwritten")
}

// A repo.yaml that does not parse is rewritten from the given defaults, so
// the footprint ends the call valid.
func TestEnsureFootprint_BrokenRepoYAMLRepairedWithDefaults(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".spektacular")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	path := filepath.Join(dir, config.RepoConfigFileName)
	require.NoError(t, os.WriteFile(path, []byte("{{ this is not yaml"), 0o644))

	status, err := EnsureFootprint(root, config.NewDefaultRepoConfig())
	require.NoError(t, err)
	require.Equal(t, FootprintRepaired, status)

	_, err = config.RepoConfigFromYAMLFile(path)
	require.NoError(t, err, "a repaired repo.yaml must parse as a valid repo config")
	require.FileExists(t, filepath.Join(root, ".spektacular", "knowledge", "conventions", "README.md"))
}

// A healthy, complete footprint is reported unchanged.
func TestEnsureFootprint_HealthyFootprintUnchanged(t *testing.T) {
	root := t.TempDir()
	_, err := EnsureFootprint(root, config.NewDefaultRepoConfig())
	require.NoError(t, err)

	status, err := EnsureFootprint(root, config.NewDefaultRepoConfig())
	require.NoError(t, err)
	require.Equal(t, FootprintUnchanged, status)
}

// A healthy existing repo.yaml is the authority for its own footprint: a
// custom knowledge location in it drives where scaffolding lands, and the
// passed-in defaults' location is ignored.
func TestEnsureFootprint_ExistingConfigCustomLocationDrivesScaffolding(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".spektacular")
	require.NoError(t, os.MkdirAll(dir, 0o755))

	custom := config.NewDefaultRepoConfig()
	custom.Knowledge.Sources[0].Config.Location = "kb"
	require.NoError(t, custom.ToYAMLFile(filepath.Join(dir, config.RepoConfigFileName)))

	status, err := EnsureFootprint(root, config.NewDefaultRepoConfig())
	require.NoError(t, err)
	require.Equal(t, FootprintRepaired, status, "topping up the missing knowledge tree is a repair")

	for _, cat := range footprintCategories {
		require.FileExists(t, filepath.Join(root, "kb", cat, "README.md"))
	}
	require.NoDirExists(t, filepath.Join(root, ".spektacular", "knowledge"),
		"the defaults' location must not be scaffolded when the existing config names another")
}
