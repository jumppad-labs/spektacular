package repo

import (
	"os"
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
		{Name: "docs", Location: target},
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

// A repo whose registered location is not on disk produces a roster entry
// with the name only — descriptive fields are all empty strings, nothing is
// cloned, and no error occurs.
func TestRoster_UnmaterializedRepoReportsNameOnly(t *testing.T) {
	root := t.TempDir()
	cfg := config.NewDefault()
	cfg.Name = "testproj"
	cfg.Repos = []config.RepoEntry{
		{Name: "remote-only", Location: filepath.Join(root, "vendor", "remote-only")},
	}

	roster := Roster(cfg, root, NewGitRunner())
	require.Len(t, roster, 1)
	entry := roster[0]
	require.Equal(t, "remote-only", entry["name"])
	require.Equal(t, "", entry["description"])
	require.Equal(t, "", entry["role"])
	require.Equal(t, "", entry["tags"])
	require.Equal(t, "", entry["deployment"])
	require.NoDirExists(t, filepath.Join(root, ".spektacular", "repos", "remote-only"), "the roster must never clone")
}

// rosterGitURL is the git location the roster tests declare as a source.
const rosterGitURL = "https://example.com/x.git"

// Phase 3.1 criterion 1: every roster entry carries a "source" key holding
// the repo's resolved code directory — the location itself when repo.yaml
// declares no source, the file-source directory (bare or file://), or the
// clone of a git source when one exists on disk — and an empty string when
// the code is not on disk: an uncloned git source, or a location that does
// not exist. The roster never invokes git to find any of this out.
func TestRoster_SourceReportsCodeDirWithoutGit(t *testing.T) {
	cases := []struct {
		name string
		// setup lays out the repo under root and returns its registered
		// (absolute) location and the source the roster must report.
		setup func(t *testing.T, root string) (location, want string)
	}{
		{
			name: "no source resolves to the location",
			setup: func(t *testing.T, root string) (string, string) {
				location := filepath.Join(root, "vendor", "member")
				writeFootprint(t, location)
				return location, location
			},
		},
		{
			name: "file source resolves to the code dir",
			setup: func(t *testing.T, root string) (string, string) {
				code := filepath.Join(root, "code")
				require.NoError(t, os.MkdirAll(code, 0o755))
				location := filepath.Join(root, "vendor", "member")
				writeSourceFootprint(t, location, code)
				return location, code
			},
		},
		{
			name: "file scheme source resolves to the code dir",
			setup: func(t *testing.T, root string) (string, string) {
				code := filepath.Join(root, "code")
				require.NoError(t, os.MkdirAll(code, 0o755))
				location := filepath.Join(root, "vendor", "member")
				writeSourceFootprint(t, location, "file://"+code)
				return location, code
			},
		},
		{
			name: "cloned git source resolves to the clone",
			setup: func(t *testing.T, root string) (string, string) {
				location := filepath.Join(root, "vendor", "member")
				writeSourceFootprint(t, location, rosterGitURL)
				clone := filepath.Join(root, ".spektacular", "repos", "member")
				require.NoError(t, os.MkdirAll(clone, 0o755)) // a clone left by an earlier resolution
				return location, clone
			},
		},
		{
			name: "uncloned git source is empty",
			setup: func(t *testing.T, root string) (string, string) {
				location := filepath.Join(root, "vendor", "member")
				writeSourceFootprint(t, location, rosterGitURL)
				return location, ""
			},
		},
		{
			name: "location not on disk is empty",
			setup: func(t *testing.T, root string) (string, string) {
				return filepath.Join(root, "vendor", "gone"), ""
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			location, want := tc.setup(t, root)
			cfg := config.NewDefault()
			cfg.Name = "testproj"
			cfg.Repos = []config.RepoEntry{{Name: "member", Location: location}}
			git := newFakeGit(t)

			roster := Roster(cfg, root, git)

			require.Len(t, roster, 1)
			require.Equal(t, "member", roster[0]["name"])
			require.Equal(t, want, roster[0]["source"])
			require.Zero(t, git.gitCalls(), "the roster must never invoke git")
			require.Empty(t, git.CloneCalls, "the roster must never clone")
		})
	}
}
