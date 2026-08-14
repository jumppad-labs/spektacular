package repo

import (
	"path/filepath"
	"testing"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/stretchr/testify/require"
)

// Roster projects each configured repo's identity plus its own descriptive
// metadata (sourced the same way `repo list` sources it: from the repo's own
// repo.yaml when it is present on disk, absent otherwise) into the shape
// workflow templates render.

// A materialized repo with its own repo.yaml carrying descriptive metadata
// produces a roster entry with the full description/role/tags/deployment,
// tags joined into a comma-separated string.
func TestRoster_MaterializedRepoReportsOwnMetadata(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "vendor", "docs")
	repoCfg := config.NewDefaultRepoConfig()
	repoCfg.Description = "the documentation repo"
	repoCfg.Role = "documentation"
	repoCfg.Tags = []string{"docs", "markdown"}
	repoCfg.Deployment = "static site on the CDN"
	writeRepoConfig(t, target, repoCfg)

	cfg := config.NewDefault()
	cfg.Name = "testproj"
	cfg.Repos = []config.RepoEntry{
		{Name: "docs", Local: target},
	}

	roster := Roster(cfg, root, NewGitRunner())
	require.Len(t, roster, 1)
	entry := roster[0]
	require.Equal(t, "docs", entry["name"])
	require.Equal(t, "the documentation repo", entry["description"])
	require.Equal(t, "documentation", entry["role"])
	require.Equal(t, "docs, markdown", entry["tags"])
	require.Equal(t, "static site on the CDN", entry["deployment"])
}

// An unmaterialized, address-only repo produces a roster entry with the name
// only — descriptive fields are all empty strings, and no error occurs.
func TestRoster_UnmaterializedRepoReportsNameOnly(t *testing.T) {
	root := t.TempDir()
	cfg := config.NewDefault()
	cfg.Name = "testproj"
	cfg.Repos = []config.RepoEntry{
		{Name: "remote-only", Address: "https://example.invalid/remote-only.git"},
	}

	roster := Roster(cfg, root, NewGitRunner())
	require.Len(t, roster, 1)
	entry := roster[0]
	require.Equal(t, "remote-only", entry["name"])
	require.Equal(t, "", entry["description"])
	require.Equal(t, "", entry["role"])
	require.Equal(t, "", entry["tags"])
	require.Equal(t, "", entry["deployment"])
}
