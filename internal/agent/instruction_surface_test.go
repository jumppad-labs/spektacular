package agent

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jumppad-labs/spektacular/templates"
	"github.com/stretchr/testify/require"
)

// forbiddenInstructionSubstrings is the closed list of literal patterns that
// must never appear in the agent-facing instruction surface (skill templates,
// step templates, and the dogfooded rendered skills under .claude/skills/).
// Each entry encodes a piece of the old stdin/heredoc interface that the
// `--from <path>` flag replaced.
var forbiddenInstructionSubstrings = []string{
	"cat .spektacular/tmp/",
	"| {{config.command}} spec file write",
	"| {{config.command}} plan file write",
	"| go run . spec file write",
	"| go run . plan file write",
	"reads stdin",
}

// TestEmbeddedTemplatesAvoidStdinInstructionSurface walks the embedded
// templates filesystem under skills/workflows/ and steps/ and asserts no
// markdown file contains a pattern from the old stdin/heredoc CLI surface.
func TestEmbeddedTemplatesAvoidStdinInstructionSurface(t *testing.T) {
	roots := []string{"skills/workflows", "steps"}
	for _, root := range roots {
		err := fs.WalkDir(templates.FS, root, func(path string, d fs.DirEntry, err error) error {
			require.NoError(t, err)
			if d.IsDir() || !strings.HasSuffix(path, ".md") {
				return nil
			}
			body, err := fs.ReadFile(templates.FS, path)
			require.NoError(t, err)
			assertNoForbiddenSubstring(t, path, string(body))
			return nil
		})
		require.NoError(t, err)
	}
}

// TestRenderedSkillsAvoidStdinInstructionSurface walks the dogfooded
// .claude/skills/ tree at the project root and asserts no rendered SKILL.md
// contains a pattern from the old stdin/heredoc CLI surface.
func TestRenderedSkillsAvoidStdinInstructionSurface(t *testing.T) {
	root := filepath.Join("..", "..", ".claude", "skills")
	info, err := os.Stat(root)
	require.NoError(t, err, "rendered skills tree must exist at %s — run `go run . init claude` to regenerate", root)
	require.True(t, info.IsDir(), "%s is not a directory", root)

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		require.NoError(t, err)
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		body, err := os.ReadFile(path)
		require.NoError(t, err)
		assertNoForbiddenSubstring(t, path, string(body))
		return nil
	})
	require.NoError(t, err)
}

// TestRenderedSpekKnowledgeBodyContainsCRUDInvocations asserts the dogfooded
// rendered spek-knowledge SKILL.md contains every CRUD entry point its prose
// orchestrates. This is a regression guard against a future edit accidentally
// dropping a load-bearing CLI reference; the expected substrings are
// hand-maintained as a literal Go slice rather than derived from the file.
func TestRenderedSpekKnowledgeBodyContainsCRUDInvocations(t *testing.T) {
	path := filepath.Join("..", "..", ".claude", "skills", "spek-knowledge", "SKILL.md")
	_, err := os.Stat(path)
	require.NoError(t, err, "rendered spek-knowledge SKILL.md must exist at %s — run `go run . init claude` to regenerate", path)

	body, err := os.ReadFile(path)
	require.NoError(t, err)
	rendered := string(body)

	expectedCRUDInvocations := []string{
		"knowledge search",
		"knowledge read",
		"knowledge sources",
		"knowledge write",
	}
	for _, needle := range expectedCRUDInvocations {
		require.Contains(t, rendered, needle, "%s is missing load-bearing CRUD invocation %q", path, needle)
	}
}

func assertNoForbiddenSubstring(t *testing.T, path, body string) {
	t.Helper()
	for _, needle := range forbiddenInstructionSubstrings {
		require.NotContains(t, body, needle, "%s contains forbidden instruction-surface pattern %q", path, needle)
	}
}
