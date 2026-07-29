package agent

import (
	"io"

	"github.com/jumppad-labs/spektacular/internal/config"
)

const (
	discoveryGroundingTemplatePath = "agents/discovery-grounding.md"
	discoveryGroundingHeading      = "## Discovery Grounding"
)

// installDiscoveryGroundingSection writes (or updates in place) the managed
// "Discovery Grounding" section in <projectPath>/AGENTS.md, rendering the
// embedded template against cfg.Command. Idempotent: re-running for the same
// projectPath leaves a single section and does not duplicate.
func installDiscoveryGroundingSection(projectPath string, cfg config.Config, out io.Writer) error {
	return installManagedSection(projectPath, cfg, out, discoveryGroundingTemplatePath, discoveryGroundingHeading, "Discovery Grounding section")
}
