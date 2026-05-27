package agent

import (
	"io"

	"github.com/jumppad-labs/spektacular/internal/config"
)

type claudeAgent struct{}

func (claudeAgent) Name() string { return "claude" }

func (claudeAgent) Install(projectPath string, cfg config.Config, out io.Writer) error {
	// Claude Code surfaces installed skills directly in its slash-command menu
	// (e.g. `/spek-new`), so a separate command wrapper would be redundant. Other
	// agents that lack a skill mechanism still get wrappers via installCommandWrappers.
	if err := installWorkflowSkills(projectPath, ".claude/skills", cfg, out); err != nil {
		return err
	}
	return installMemoryContextSection(projectPath, cfg, out)
}

func init() {
	register(claudeAgent{})
}
