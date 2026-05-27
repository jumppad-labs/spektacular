package agent

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/stretchr/testify/require"
)

func TestCodexAgent_Name(t *testing.T) {
	require.Equal(t, "codex", codexAgent{}.Name())
}

func TestCodexAgent_Install(t *testing.T) {
	tmp := t.TempDir()
	cfg := config.NewDefault()

	err := codexAgent{}.Install(tmp, cfg, io.Discard)
	require.NoError(t, err)

	// Exactly three SKILL.md files under .agents/skills/spek-{new,plan,implement}/.
	skillAssertions := map[string]string{
		"spek-new":       "spektacular spec new",
		"spek-plan":      "spektacular plan new",
		"spek-implement": "spektacular implement new",
		"spek-knowledge": "knowledge",
	}
	for skill, expected := range skillAssertions {
		skillPath := filepath.Join(tmp, ".agents", "skills", skill, "SKILL.md")
		require.FileExists(t, skillPath)
		data, err := os.ReadFile(skillPath)
		require.NoError(t, err)
		require.Contains(t, string(data), expected)
		require.NotContains(t, string(data), "{{command}}")
	}

	// Codex has no per-repo slash-command mechanism, so no command wrappers
	// should be installed and no other agent roots should appear under tmp.
	require.NoDirExists(t, filepath.Join(tmp, ".agents", "commands"))
	require.NoDirExists(t, filepath.Join(tmp, ".claude"))
	require.NoDirExists(t, filepath.Join(tmp, ".bob"))

	// Each installed SKILL.md must have a valid frontmatter block that
	// satisfies the agentskills.io naming rules.
	for skill := range skillAssertions {
		validateSkillFrontmatter(t, filepath.Join(tmp, ".agents", "skills", skill, "SKILL.md"))
	}
}
