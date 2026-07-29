package templates

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestWorkflowSkillsDirectAgentToCLIList verifies that each workflow skill
// tells its driving agent to enumerate the plan/spec store via the CLI's
// `file list` command and explicitly forbids poking `.spektacular/plans/`
// or `.spektacular/specs/` with `ls`/`find`/`Read`. Both signals must be
// present so a future edit that quietly drops one fails this test rather
// than silently regressing the guardrail.
func TestWorkflowSkillsDirectAgentToCLIList(t *testing.T) {
	cases := []struct {
		skill    string
		listCmd  string
		storeDir string
	}{
		{"skills/workflows/spek-new/SKILL.md", "spec file list", ".spektacular/specs/"},
		{"skills/workflows/spek-plan/SKILL.md", "spec file list", ".spektacular/specs/"},
		{"skills/workflows/spek-implement/SKILL.md", "plan file list", ".spektacular/plans/"},
	}
	for _, c := range cases {
		body := mustReadTemplate(t, c.skill)
		require.Containsf(t, body, c.listCmd,
			"%s must document the %q command so the agent enumerates via the CLI, not the filesystem", c.skill, c.listCmd)
		require.Containsf(t, body, "Do not",
			"%s must explicitly forbid ls/find/Read against the store directory", c.skill)
		require.Containsf(t, body, c.storeDir,
			"%s must name %s as the store directory the agent must not poke directly", c.skill, c.storeDir)
	}
}
