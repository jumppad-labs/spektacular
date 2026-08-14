package agent

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jumppad-labs/spektacular/internal/config"
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

// TestRenderedSkillsAvoidStdinInstructionSurface renders every workflow skill
// into a freshly-created temp directory via the real install path and asserts
// no rendered SKILL.md contains a pattern from the old stdin/heredoc CLI
// surface. The test owns the directory it walks — it does not depend on any
// pre-existing on-disk state.
func TestRenderedSkillsAvoidStdinInstructionSurface(t *testing.T) {
	tmp := t.TempDir()
	cfg := config.NewDefault()

	require.NoError(t, installWorkflowSkills(tmp, ".claude/skills", cfg, io.Discard))

	root := filepath.Join(tmp, ".claude", "skills")
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
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

// TestRenderedSpekKnowledgeBodyContainsCRUDInvocations renders the workflow
// skills into a fresh temp directory and asserts the rendered spek-knowledge
// SKILL.md contains every CRUD entry point its prose orchestrates. This is a
// regression guard against a future edit accidentally dropping a load-bearing
// CLI reference; the expected substrings are hand-maintained as a literal Go
// slice rather than derived from the file. The test owns the directory it
// reads — it does not depend on any pre-existing on-disk state.
func TestRenderedSpekKnowledgeBodyContainsCRUDInvocations(t *testing.T) {
	tmp := t.TempDir()
	cfg := config.NewDefault()

	require.NoError(t, installWorkflowSkills(tmp, ".claude/skills", cfg, io.Discard))

	path := filepath.Join(tmp, ".claude", "skills", "spek-knowledge", "SKILL.md")
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

// TestRenderedWorkflowSkillsCarryCrossRepoNotes renders the workflow skills
// through the real install path and asserts the spek-plan and spek-implement
// SKILL.md files carry their Phase 4.2 cross-repo notes — roster-driven repo
// attribution for planning (criterion 2), attributed-repo execution with
// per-repo derived changelog entries for implementation (criterion 3) — with
// the {{command}} placeholder rendered away. The test owns the directory it
// reads — it does not depend on any pre-existing on-disk state.
func TestRenderedWorkflowSkillsCarryCrossRepoNotes(t *testing.T) {
	tmp := t.TempDir()
	cfg := config.NewDefault()

	require.NoError(t, installWorkflowSkills(tmp, ".claude/skills", cfg, io.Discard))

	readSkill := func(t *testing.T, name string) string {
		t.Helper()
		body, err := os.ReadFile(filepath.Join(tmp, ".claude", "skills", name, "SKILL.md"))
		require.NoError(t, err)
		return string(body)
	}

	t.Run("spek-plan", func(t *testing.T) {
		body := readSkill(t, "spek-plan")

		// Criterion 2: planning attributes every requirement to its repo.
		require.Contains(t, body, "Cross-repo planning",
			"spek-plan must carry the cross-repo planning note")
		require.Contains(t, body, "attribute every requirement to the repo",
			"spek-plan must direct attributing every requirement to a repo")
		require.Contains(t, body, "spektacular repo list",
			"the {{command}} placeholder must render to the configured command")
		require.NotContains(t, body, "{{command}}",
			"the rendered skill must not leak the {{command}} placeholder")
	})

	t.Run("spek-implement", func(t *testing.T) {
		body := readSkill(t, "spek-implement")

		// Criterion 3: work runs in the attributed repo's resolved root and
		// derived changelog entries follow each affected repo.
		require.Contains(t, body, "Cross-repo implementation",
			"spek-implement must carry the cross-repo implementation note")
		require.Contains(t, body, "attributed repo's resolved root",
			"spek-implement must direct work into the attributed repo's resolved root")
		require.Contains(t, body, "one derived entry per affected repo",
			"spek-implement must direct one derived changelog entry per affected repo")
		require.Contains(t, body, "--repo <name>",
			"derived entries must be written via `changelog file write ... --repo <name>`")
		require.NotContains(t, body, "{{command}}",
			"the rendered skill must not leak the {{command}} placeholder")
	})
}

func assertNoForbiddenSubstring(t *testing.T, path, body string) {
	t.Helper()
	for _, needle := range forbiddenInstructionSubstrings {
		require.NotContains(t, body, needle, "%s contains forbidden instruction-surface pattern %q", path, needle)
	}
}
