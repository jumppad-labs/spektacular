package implement

import (
	"path/filepath"

	"github.com/jumppad-labs/spektacular/internal/stepkit"
)

// PlanFilePath returns the store-relative path for a plan's plan.md file under
// the configured plan directory.
// Kept as a copy of internal/steps/plan.PlanFilePath to avoid a cross-package
// dependency for a 10-line constant function.
func PlanFilePath(dir, name string) string {
	return dir + "/" + name + "/plan.md"
}

// ContextFilePath returns the store-relative path for a plan's context.md file
// under the configured plan directory.
func ContextFilePath(dir, name string) string {
	return dir + "/" + name + "/context.md"
}

// ResearchFilePath returns the store-relative path for a plan's research.md file
// under the configured plan directory.
func ResearchFilePath(dir, name string) string {
	return dir + "/" + name + "/research.md"
}

// strategy implements stepkit.PathStrategy for the implement workflow. planDir
// is the configured plan directory.
type strategy struct {
	planDir string
}

func (strategy) PrimaryPathField() string { return "plan_path" }

func (s strategy) PathVars(instanceName, storeRoot string) map[string]any {
	planPath := filepath.Join(storeRoot, PlanFilePath(s.planDir, instanceName))
	contextPath := filepath.Join(storeRoot, ContextFilePath(s.planDir, instanceName))
	researchPath := filepath.Join(storeRoot, ResearchFilePath(s.planDir, instanceName))
	return map[string]any{
		"plan_path":              planPath,
		"context_path":           contextPath,
		"research_path":          researchPath,
		"plan_dir":               filepath.Dir(planPath),
		"plan_name":              instanceName,
		"changelog_section_name": "## Changelog",
	}
}

// Compile-time interface check.
var _ stepkit.PathStrategy = strategy{}
