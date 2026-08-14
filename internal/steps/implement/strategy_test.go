package implement

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestStrategyPathVars_ChangelogPathIncludesProjectNamespace asserts the
// implement strategy's changelog_path template var routes the feature's
// changelog record under the project-named namespace folder:
// `<changelogDir>/<projectName>/<name>.md`.
func TestStrategyPathVars_ChangelogPathIncludesProjectNamespace(t *testing.T) {
	s := strategy{
		planDir:      "plans",
		changelogDir: "changelog",
		specDir:      "specs",
		projectName:  "testproj",
	}

	vars := s.PathVars("000039_feature", "/store")

	require.Equal(t,
		filepath.Join("/store", "changelog", "testproj", "000039_feature.md"),
		vars["changelog_path"],
		"changelog_path must contain the <changelogDir>/<projectName>/<name>.md segment")
}

// TestChangelogFilePath_IncludesProjectSegment pins the path helper itself:
// the project name sits between the configured directory and the flat
// per-feature file name.
func TestChangelogFilePath_IncludesProjectSegment(t *testing.T) {
	require.Equal(t, "changelog/testproj/000039_feature.md",
		ChangelogFilePath("changelog", "testproj", "000039_feature"))
}
