package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// This file asserts the repo-root documentation shipped with Phase 4.2:
// README.md's project/repo configuration split and CHANGELOG.md's
// breaking-change entry. These two files ARE the artifacts under test, so
// they are read read-only from the repository root (relative to this package
// directory) rather than rendered into a temp dir.

// readRepoRootFile reads a file at the repository root, relative to this
// package's directory. Content is treated strictly read-only.
func readRepoRootFile(t *testing.T, name string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", name))
	require.NoError(t, err, "%s must exist at the repository root", name)
	return string(body)
}

// Criterion 5: the README describes the project-level config.yaml vs per-repo
// repo.yaml split, the repos registry, the ignore file, and the
// breaking-change re-initialization notice.
func TestREADMEDescribesProjectRepoConfigurationSplit(t *testing.T) {
	readme := readRepoRootFile(t, "README.md")

	require.Contains(t, readme, "repo.yaml",
		"README must describe the per-repo repo.yaml configuration")
	require.Contains(t, readme, "repos:",
		"README must show the project config's repos registry")
	require.Contains(t, readme, ".spektacular_ignore",
		"README must document the .spektacular_ignore exclusion file")
	require.Contains(t, readme, "Breaking change",
		"README must carry the breaking-change notice")
	require.Contains(t, readme, "re-initialize",
		"README must tell existing setups to re-initialize")
}

// Criterion 5: CHANGELOG.md records the 000039 project-level-capabilities
// entry's breaking change. New entries are prepended above it by every
// subsequent plan's update_repo_changelog step, so this only checks that
// the 000039 section itself still carries the notice, not that it remains
// the file's top entry.
func TestChangelogTopSectionRecordsBreakingChange(t *testing.T) {
	changelog := strings.TrimSpace(readRepoRootFile(t, "CHANGELOG.md"))

	const heading = "## 000039_project-level-capabilities"
	start := strings.Index(changelog, heading)
	require.GreaterOrEqual(t, start, 0, "CHANGELOG.md must retain the %s entry", heading)

	// The section is everything between its heading and the next `## ` heading.
	rest := changelog[start+len(heading):]
	next := strings.Index(rest, "\n## ")
	require.Greater(t, next, 0, "CHANGELOG.md must retain the earlier entries below the 000039 section")
	section := rest[:next]

	require.Contains(t, section, "Breaking change",
		"the 000039 changelog section must record the breaking change")
	require.Contains(t, section, "re-initialize",
		"the 000039 changelog section must direct existing setups to re-initialize")
}
