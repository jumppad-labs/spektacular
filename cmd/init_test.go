package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jumppad-labs/spektacular/internal/agent"
	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/stretchr/testify/require"
)

func TestInit_Claude(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	rootCmd.SetArgs([]string{"init", "claude"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	// .spektacular directory created
	_, err = os.Stat(filepath.Join(dir, ".spektacular"))
	require.NoError(t, err)

	// Three SKILL.md files exist, each rendered with the default `spektacular`
	// command. Each workflow embeds a distinctive `<command> <subcmd> new` line
	// that proves the {{command}} placeholder was expanded.
	skillAssertions := map[string]string{
		"spek-new":       "spektacular spec new",
		"spek-plan":      "spektacular plan new",
		"spek-implement": "spektacular implement new",
	}
	for skill, expected := range skillAssertions {
		skillPath := filepath.Join(dir, ".claude", "skills", skill, "SKILL.md")
		data, err := os.ReadFile(skillPath)
		require.NoError(t, err, "expected skill file %s to exist", skillPath)
		require.Contains(t, string(data), expected)
		require.NotContains(t, string(data), "{{command}}")
	}

	// Three command wrappers exist under .claude/commands/spek/, each
	// delegating to its corresponding installed skill.
	commandAssertions := map[string]string{
		"new.md":       "`spek-new` skill",
		"plan.md":      "`spek-plan` skill",
		"implement.md": "`spek-implement` skill",
	}
	for base, expected := range commandAssertions {
		cmdPath := filepath.Join(dir, ".claude", "commands", "spek", base)
		data, err := os.ReadFile(cmdPath)
		require.NoError(t, err, "expected command file %s to exist", cmdPath)
		require.Contains(t, string(data), expected)
		require.NotContains(t, string(data), "{{command}}")
		require.NotContains(t, string(data), "{{skill}}")
	}
}

func TestInit_Bob(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	rootCmd.SetArgs([]string{"init", "bob"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	// .spektacular directory created
	_, err = os.Stat(filepath.Join(dir, ".spektacular"))
	require.NoError(t, err)

	// Three SKILL.md files under .bob/skills/spek-{new,plan,implement}/.
	skillAssertions := map[string]string{
		"spek-new":       "spektacular spec new",
		"spek-plan":      "spektacular plan new",
		"spek-implement": "spektacular implement new",
	}
	for skill, expected := range skillAssertions {
		skillPath := filepath.Join(dir, ".bob", "skills", skill, "SKILL.md")
		data, err := os.ReadFile(skillPath)
		require.NoError(t, err, "expected skill file %s to exist", skillPath)
		require.Contains(t, string(data), expected)
		require.NotContains(t, string(data), "{{command}}")
	}

	// Three command wrappers under .bob/commands/ — Bob keeps the `spek-`
	// prefix in the basename.
	commandAssertions := map[string]string{
		"spek-new.md":       "`spek-new` skill",
		"spek-plan.md":      "`spek-plan` skill",
		"spek-implement.md": "`spek-implement` skill",
	}
	for base, expected := range commandAssertions {
		cmdPath := filepath.Join(dir, ".bob", "commands", base)
		data, err := os.ReadFile(cmdPath)
		require.NoError(t, err, "expected command file %s to exist", cmdPath)
		require.Contains(t, string(data), expected)
		require.NotContains(t, string(data), "{{command}}")
		require.NotContains(t, string(data), "{{skill}}")
	}
}

func TestInit_Codex(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	rootCmd.SetArgs([]string{"init", "codex"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	// .spektacular directory created
	_, err = os.Stat(filepath.Join(dir, ".spektacular"))
	require.NoError(t, err)

	// Three SKILL.md files under .agents/skills/spek-{new,plan,implement}/.
	skillAssertions := map[string]string{
		"spek-new":       "spektacular spec new",
		"spek-plan":      "spektacular plan new",
		"spek-implement": "spektacular implement new",
	}
	for skill, expected := range skillAssertions {
		skillPath := filepath.Join(dir, ".agents", "skills", skill, "SKILL.md")
		data, err := os.ReadFile(skillPath)
		require.NoError(t, err, "expected skill file %s to exist", skillPath)
		require.Contains(t, string(data), expected)
		require.NotContains(t, string(data), "{{command}}")
		require.Contains(t, string(data), "notion status")
	}

	// Codex has no per-repo slash-command mechanism — no command wrappers or
	// other agent roots should be created.
	require.NoDirExists(t, filepath.Join(dir, ".agents", "commands"))
	require.NoDirExists(t, filepath.Join(dir, ".claude"))
	require.NoDirExists(t, filepath.Join(dir, ".bob"))
}

func TestInit_InvalidAgent(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	rootCmd.SetArgs([]string{"init", "unknown"})
	err := rootCmd.Execute()
	require.Error(t, err)
	require.True(t, errors.Is(err, agent.ErrUnknownAgent), "error should wrap agent.ErrUnknownAgent, got %v", err)
	require.Contains(t, err.Error(), "claude")
	require.Contains(t, err.Error(), "bob")
	require.Contains(t, err.Error(), "codex")
}

func TestInit_CustomCommand(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	// First init to create .spektacular with default config
	rootCmd.SetArgs([]string{"init", "claude"})
	require.NoError(t, rootCmd.Execute())

	// Override the command in config
	configPath := filepath.Join(dir, ".spektacular", "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("command: \"go run .\"\n"), 0644))

	// Re-init — should use the custom command when rendering templates.
	rootCmd.SetArgs([]string{"init", "claude"})
	require.NoError(t, rootCmd.Execute())

	cfg, err := config.FromYAMLFile(configPath)
	require.NoError(t, err)
	require.Equal(t, "go run .", cfg.Command)
	require.Equal(t, "timestamp", cfg.Spec.IDMethod)
	require.Equal(t, 0, cfg.Spec.Counter)

	skillPath := filepath.Join(dir, ".claude", "skills", "spek-new", "SKILL.md")
	skillData, err := os.ReadFile(skillPath)
	require.NoError(t, err)
	require.Contains(t, string(skillData), "go run . spec new")
	require.NotContains(t, string(skillData), "{{command}}")
}

func TestInit_Idempotent(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	// Pre-create sibling files under both the commands and skills trees. Both
	// must survive re-init.
	commandsDir := filepath.Join(dir, ".claude", "commands")
	require.NoError(t, os.MkdirAll(commandsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(commandsDir, "other.md"), []byte("keep"), 0644))

	siblingSkillDir := filepath.Join(dir, ".claude", "skills", "other")
	require.NoError(t, os.MkdirAll(siblingSkillDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(siblingSkillDir, "SKILL.md"), []byte("keep-skill"), 0644))

	rootCmd.SetArgs([]string{"init", "claude"})
	require.NoError(t, rootCmd.Execute())

	rootCmd.SetArgs([]string{"init", "claude"})
	require.NoError(t, rootCmd.Execute())

	// Sibling command file still exists and is untouched.
	data, err := os.ReadFile(filepath.Join(commandsDir, "other.md"))
	require.NoError(t, err)
	require.Equal(t, "keep", string(data))

	// Sibling skill file still exists and is untouched.
	skillData, err := os.ReadFile(filepath.Join(siblingSkillDir, "SKILL.md"))
	require.NoError(t, err)
	require.Equal(t, "keep-skill", string(skillData))
}
