package implement

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestStrategyPathVars_ChangelogPathIsFlat asserts the implement strategy's
// changelog_path template var resolves the feature's project-level changelog
// record directly under the configured changelog directory — no `<project>/`
// subfolder. Per-repo entries are routed by the CLI's `--repo` flag under
// the target repo's own changelog store; they are not surfaced through this
// template var.
func TestStrategyPathVars_ChangelogPathIsFlat(t *testing.T) {
	s := strategy{
		planDir:      "plans",
		changelogDir: "changelog",
		specDir:      "specs",
	}

	vars := s.PathVars("000039_feature", "/store")

	require.Equal(t,
		filepath.Join("/store", "changelog", "000039_feature.md"),
		vars["changelog_path"],
		"changelog_path must resolve to <changelogDir>/<name>.md — no project subfolder")
}

// TestChangelogFilePath_IsFlat pins the path helper itself: no project name
// segment; the file lives directly under the configured directory.
func TestChangelogFilePath_IsFlat(t *testing.T) {
	require.Equal(t, "changelog/000039_feature.md",
		ChangelogFilePath("changelog", "000039_feature"))
}
