package agent

import (
	"io"

	"github.com/jumppad-labs/spektacular/internal/config"
)

type codexAgent struct{}

func (codexAgent) Name() string { return "codex" }

func (codexAgent) Install(projectPath string, cfg config.Config, out io.Writer) error {
	if err := installWorkflowSkills(projectPath, ".agents/skills", cfg, out); err != nil {
		return err
	}
	return installMemoryContextSection(projectPath, cfg, out)
}

func init() {
	register(codexAgent{})
}
