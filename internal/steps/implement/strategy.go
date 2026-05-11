package implement

import (
	"path/filepath"

	"github.com/jumppad-labs/spektacular/internal/artifact"
	"github.com/jumppad-labs/spektacular/internal/stepkit"
	"github.com/jumppad-labs/spektacular/internal/workflow"
)

// PlanFilePath returns the store-relative path for a plan's plan.md file.
// Kept as a copy of internal/steps/plan.PlanFilePath to avoid a cross-package
// dependency for a 10-line constant function.
func PlanFilePath(name string) string {
	return "plans/" + name + "/plan.md"
}

// ContextFilePath returns the store-relative path for a plan's context.md file.
func ContextFilePath(name string) string {
	return "plans/" + name + "/context.md"
}

// ResearchFilePath returns the store-relative path for a plan's research.md file.
func ResearchFilePath(name string) string {
	return "plans/" + name + "/research.md"
}

// strategy implements stepkit.PathStrategy for the implement workflow.
type strategy struct{}

func (strategy) PrimaryPathField() string { return "plan_path" }

func (strategy) PathVars(instanceName, storeRoot string) map[string]any {
	planPath := filepath.Join(storeRoot, PlanFilePath(instanceName))
	contextPath := filepath.Join(storeRoot, ContextFilePath(instanceName))
	researchPath := filepath.Join(storeRoot, ResearchFilePath(instanceName))
	return map[string]any{
		"plan_path":              planPath,
		"context_path":           contextPath,
		"research_path":          researchPath,
		"plan_dir":               filepath.Dir(planPath),
		"plan_name":              instanceName,
		"changelog_section_name": "## Changelog",
	}
}

func (s strategy) PathVarsWithConfig(instanceName, storeRoot string, cfg workflow.Config) map[string]any {
	planPath, err := artifact.Path(cfg.Project, artifact.KindPlan, instanceName)
	if err != nil {
		return s.PathVars(instanceName, storeRoot)
	}
	contextPath, err := artifact.Path(cfg.Project, artifact.KindContext, instanceName)
	if err != nil {
		return s.PathVars(instanceName, storeRoot)
	}
	researchPath, err := artifact.Path(cfg.Project, artifact.KindResearch, instanceName)
	if err != nil {
		return s.PathVars(instanceName, storeRoot)
	}
	planAbsPath := filepath.Join(storeRoot, planPath)
	return map[string]any{
		"plan_path":              planAbsPath,
		"context_path":           filepath.Join(storeRoot, contextPath),
		"research_path":          filepath.Join(storeRoot, researchPath),
		"plan_dir":               filepath.Dir(planAbsPath),
		"plan_name":              instanceName,
		"changelog_section_name": "## Changelog",
	}
}

// Compile-time interface check.
var _ stepkit.PathStrategy = strategy{}
