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

// ChangelogFilePath returns the store-relative path for a feature's changelog
// record under the configured changelog directory: a flat file per feature
// (unlike PlanFilePath's per-feature subdirectory), namespaced under a folder
// bearing the project's name so multiple projects sharing a store never
// collide.
func ChangelogFilePath(dir, projectName, name string) string {
	return dir + "/" + projectName + "/" + name + ".md"
}

// strategy implements stepkit.PathStrategy for the implement workflow. planDir
// is the configured plan directory; changelogDir and specDir are the
// configured changelog and spec directories. projectName namespaces the
// changelog path.
type strategy struct {
	planDir      string
	changelogDir string
	specDir      string
	projectName  string
}

func (strategy) PrimaryPathField() string { return "plan_path" }

func (s strategy) PathVars(instanceName, storeRoot string) map[string]any {
	planPath := filepath.Join(storeRoot, PlanFilePath(s.planDir, instanceName))
	contextPath := filepath.Join(storeRoot, ContextFilePath(s.planDir, instanceName))
	researchPath := filepath.Join(storeRoot, ResearchFilePath(s.planDir, instanceName))
	changelogPath := filepath.Join(storeRoot, ChangelogFilePath(s.changelogDir, s.projectName, instanceName))
	specPath := filepath.Join(storeRoot, s.specDir, instanceName+".md")
	return map[string]any{
		"plan_path":              planPath,
		"context_path":           contextPath,
		"research_path":          researchPath,
		"plan_dir":               filepath.Dir(planPath),
		"plan_name":              instanceName,
		"changelog_section_name": "## Changelog",
		"changelog_path":         changelogPath,
		"spec_path":              specPath,
	}
}

// Compile-time interface check.
var _ stepkit.PathStrategy = strategy{}
