package agent

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/stretchr/testify/require"
)

func TestBobAgent_Name(t *testing.T) {
	require.Equal(t, "bob", bobAgent{}.Name())
}

func TestBobAgent_Install(t *testing.T) {
	tmp := t.TempDir()
	cfg := config.NewDefault()

	err := bobAgent{}.Install(tmp, cfg, io.Discard)
	require.NoError(t, err)

	// Exactly three SKILL.md files under .bob/skills/spek-{new,plan,implement}/.
	skillAssertions := map[string]string{
		"spek-new":       "spektacular spec new",
		"spek-plan":      "spektacular plan new",
		"spek-implement": "spektacular implement new",
		"spek-knowledge": "knowledge",
	}
	for skill, expected := range skillAssertions {
		skillPath := filepath.Join(tmp, ".bob", "skills", skill, "SKILL.md")
		require.FileExists(t, skillPath)
		data, err := os.ReadFile(skillPath)
		require.NoError(t, err)
		require.Contains(t, string(data), expected)
		require.NotContains(t, string(data), "{{command}}")
	}

	// Exactly three command wrappers under .bob/commands/, basenames keep the
	// `spek-` prefix.
	commandAssertions := map[string]string{
		"spek-new.md":       "`spek-new` skill",
		"spek-plan.md":      "`spek-plan` skill",
		"spek-implement.md": "`spek-implement` skill",
		"spek-knowledge.md": "`spek-knowledge` skill",
	}
	for base, expected := range commandAssertions {
		cmdPath := filepath.Join(tmp, ".bob", "commands", base)
		require.FileExists(t, cmdPath)
		data, err := os.ReadFile(cmdPath)
		require.NoError(t, err)
		require.Contains(t, string(data), expected)
		require.NotContains(t, string(data), "{{command}}")
		require.NotContains(t, string(data), "{{skill}}")
	}

	// Bob command filenames keep the `spek-` prefix — make sure the stripped
	// variants do NOT exist on disk.
	for _, stripped := range []string{"new.md", "plan.md", "implement.md", "knowledge.md"} {
		require.NoFileExists(t, filepath.Join(tmp, ".bob", "commands", stripped))
	}

	// Each installed SKILL.md must have a valid frontmatter block that
	// satisfies the agentskills.io naming rules.
	for skill := range skillAssertions {
		validateSkillFrontmatter(t, filepath.Join(tmp, ".bob", "skills", skill, "SKILL.md"))
	}
}
