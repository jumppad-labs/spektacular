package plan

import (
	"github.com/jumppad-labs/spektacular/internal/stepkit"
	"github.com/jumppad-labs/spektacular/internal/store"
	"github.com/jumppad-labs/spektacular/internal/workflow"
)

// PlanFilePath returns the store-relative path for a plan file under the
// configured plan directory.
func PlanFilePath(dir, name string) string {
	return dir + "/" + name + "/plan.md"
}

// ContextFilePath returns the store-relative path for a plan's context file
// under the configured plan directory.
func ContextFilePath(dir, name string) string {
	return dir + "/" + name + "/context.md"
}

// ResearchFilePath returns the store-relative path for a plan's research file
// under the configured plan directory.
func ResearchFilePath(dir, name string) string {
	return dir + "/" + name + "/research.md"
}

// Steps returns the ordered step configs for a plan workflow.
func Steps() []workflow.StepConfig {
	return []workflow.StepConfig{
		{Name: "new", Src: []string{"start"}, Dst: "new", Callback: new()},
		{Name: "overview", Src: []string{"new"}, Dst: "overview", Callback: overview()},
		{Name: "discovery", Src: []string{"overview"}, Dst: "discovery", Callback: discovery()},
		{Name: "architecture", Src: []string{"discovery"}, Dst: "architecture", Callback: architecture()},
		{Name: "components", Src: []string{"architecture"}, Dst: "components", Callback: components()},
		{Name: "data_structures", Src: []string{"components"}, Dst: "data_structures", Callback: dataStructures()},
		{Name: "implementation_detail", Src: []string{"data_structures"}, Dst: "implementation_detail", Callback: implementationDetail()},
		{Name: "dependencies", Src: []string{"implementation_detail"}, Dst: "dependencies", Callback: dependencies()},
		{Name: "testing_approach", Src: []string{"dependencies"}, Dst: "testing_approach", Callback: testingApproach()},
		{Name: "milestones", Src: []string{"testing_approach"}, Dst: "milestones", Callback: milestones()},
		{Name: "phases", Src: []string{"milestones"}, Dst: "phases", Callback: phases()},
		{Name: "open_questions", Src: []string{"phases"}, Dst: "open_questions", Callback: openQuestions()},
		{Name: "out_of_scope", Src: []string{"open_questions"}, Dst: "out_of_scope", Callback: outOfScope()},
		{Name: "verification", Src: []string{"out_of_scope"}, Dst: "verification", Callback: verification()},
		{Name: "write_plan", Src: []string{"verification"}, Dst: "write_plan", Callback: writePlan()},
		{Name: "write_context", Src: []string{"write_plan"}, Dst: "write_context", Callback: writeContext()},
		{Name: "write_research", Src: []string{"write_context"}, Dst: "write_research", Callback: writeResearch()},
		{Name: "finished", Src: []string{"write_research"}, Dst: "finished", Callback: finished()},
	}
}

// buildResult is the stepkit.ResultBuilder for the plan workflow.
func buildResult(stepName, instanceName, primaryPath, instruction string) any {
	return Result{
		Step:        stepName,
		PlanPath:    primaryPath,
		PlanName:    instanceName,
		Instruction: instruction,
	}
}

// writeStep is a one-liner wrapper around stepkit.WriteStepResult with the
// plan strategy and result builder pre-applied. Step callbacks below call it.
func writeStep(stepName, nextStep, templatePath string, data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config, extra map[string]any) error {
	return stepkit.WriteStepResult(
		stepkit.StepRequest{
			StepName:     stepName,
			NextStep:     nextStep,
			TemplatePath: templatePath,
			Strategy:     strategy{planDir: cfg.PlanDir, specDir: cfg.SpecDir},
			Extra:        extra,
		},
		data, out, st, cfg,
		buildResult,
	)
}

// new initializes state only — no document created yet.
func new() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		return "overview", nil
	}
}

func overview() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		return "", writeStep("overview", "discovery", "steps/plan/01-overview.md", data, out, st, cfg, nil)
	}
}

func discovery() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		return "", writeStep("discovery", "architecture", "steps/plan/02-discovery.md", data, out, st, cfg, nil)
	}
}

func architecture() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		return "", writeStep("architecture", "components", "steps/plan/03-architecture.md", data, out, st, cfg, nil)
	}
}

func components() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		return "", writeStep("components", "data_structures", "steps/plan/04-components.md", data, out, st, cfg, nil)
	}
}

func dataStructures() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		return "", writeStep("data_structures", "implementation_detail", "steps/plan/05-data_structures.md", data, out, st, cfg, nil)
	}
}

func implementationDetail() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		return "", writeStep("implementation_detail", "dependencies", "steps/plan/06-implementation_detail.md", data, out, st, cfg, nil)
	}
}

func dependencies() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		return "", writeStep("dependencies", "testing_approach", "steps/plan/07-dependencies.md", data, out, st, cfg, nil)
	}
}

func testingApproach() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		return "", writeStep("testing_approach", "milestones", "steps/plan/08-testing_approach.md", data, out, st, cfg, nil)
	}
}

func milestones() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		return "", writeStep("milestones", "phases", "steps/plan/09-milestones.md", data, out, st, cfg, nil)
	}
}

func phases() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		return "", writeStep("phases", "open_questions", "steps/plan/10-phases.md", data, out, st, cfg, nil)
	}
}

func openQuestions() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		return "", writeStep("open_questions", "out_of_scope", "steps/plan/11-open_questions.md", data, out, st, cfg, nil)
	}
}

func outOfScope() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		return "", writeStep("out_of_scope", "verification", "steps/plan/12-out_of_scope.md", data, out, st, cfg, nil)
	}
}

func verification() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		planName := stepkit.GetString(data, "name")
		planScaffold, err := stepkit.RenderTemplate("scaffold/plan.md", map[string]any{"name": planName})
		if err != nil {
			return "", err
		}
		contextScaffold, err := stepkit.RenderTemplate("scaffold/context.md", map[string]any{"name": planName})
		if err != nil {
			return "", err
		}
		researchScaffold, err := stepkit.RenderTemplate("scaffold/research.md", map[string]any{"name": planName})
		if err != nil {
			return "", err
		}
		return "", writeStep("verification", "write_plan", "steps/plan/13-verification.md", data, out, st, cfg, map[string]any{
			"plan_template":     planScaffold,
			"context_template":  contextScaffold,
			"research_template": researchScaffold,
		})
	}
}

// planDocPathFunc is the signature shared by PlanFilePath, ContextFilePath and
// ResearchFilePath: it maps the configured plan directory and plan name to a
// store-relative document path.
type planDocPathFunc func(dir, name string) string

// planDoc pairs a generated plan document's store-path helper with the
// scaffold template it is verified against.
type planDoc struct {
	path     planDocPathFunc
	scaffold string
}

// planDocs are the three documents the plan workflow produces, each committed
// to the store by the agent via `plan file write` during the write steps.
var planDocs = []planDoc{
	{PlanFilePath, "scaffold/plan.md"},
	{ContextFilePath, "scaffold/context.md"},
	{ResearchFilePath, "scaffold/research.md"},
}

// planDocStillScaffold reads a generated plan document back through the store
// and reports whether it is missing or still holds the unfilled scaffold — i.e.
// the agent has not yet committed it with `plan file write`.
func planDocStillScaffold(st store.Store, doc planDoc, planDir, planName string) (bool, error) {
	stored, err := st.Read(doc.path(planDir, planName))
	if err != nil {
		return true, nil
	}
	scaffold, err := stepkit.RenderTemplate(doc.scaffold, map[string]any{"name": planName})
	if err != nil {
		return false, err
	}
	return string(stored) == scaffold, nil
}

// docWarning checks one plan document and returns the template extras: when the
// document is missing or still the scaffold it sets warnKey=true so the step
// instruction can warn the agent to commit it via `plan file write`.
func docWarning(data workflow.Data, st store.Store, cfg workflow.Config, doc planDoc, warnKey string) (map[string]any, error) {
	if cfg.DryRun || st == nil {
		return nil, nil
	}
	unwritten, err := planDocStillScaffold(st, doc, cfg.PlanDir, stepkit.GetString(data, "name"))
	if err != nil {
		return nil, err
	}
	if unwritten {
		return map[string]any{warnKey: true}, nil
	}
	return nil, nil
}

// writePlan verifies plan.md was committed to the store via `plan file write`.
func writePlan() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		extra, err := docWarning(data, st, cfg, planDocs[0], "plan_unwritten")
		if err != nil {
			return "", err
		}
		return "", writeStep("write_plan", "write_context", "steps/plan/14-write_plan.md", data, out, st, cfg, extra)
	}
}

// writeContext verifies context.md was committed to the store via `plan file write`.
func writeContext() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		extra, err := docWarning(data, st, cfg, planDocs[1], "context_unwritten")
		if err != nil {
			return "", err
		}
		return "", writeStep("write_context", "write_research", "steps/plan/15-write_context.md", data, out, st, cfg, extra)
	}
}

// writeResearch verifies research.md was committed to the store via `plan file write`.
func writeResearch() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		extra, err := docWarning(data, st, cfg, planDocs[2], "research_unwritten")
		if err != nil {
			return "", err
		}
		return "", writeStep("write_research", "finished", "steps/plan/16-write_research.md", data, out, st, cfg, extra)
	}
}

func finished() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		// Read every plan document back through the store; if any is missing
		// or still the scaffold, surface a warning in the finished instruction
		// rather than erroring (a fatal error here would strand the workflow
		// state on the terminal step).
		var extra map[string]any
		if !cfg.DryRun && st != nil {
			planName := stepkit.GetString(data, "name")
			for _, doc := range planDocs {
				unwritten, err := planDocStillScaffold(st, doc, cfg.PlanDir, planName)
				if err != nil {
					return "", err
				}
				if unwritten {
					extra = map[string]any{"plan_incomplete": true}
					break
				}
			}
		}
		return "", writeStep("finished", "", "steps/plan/17-finished.md", data, out, st, cfg, extra)
	}
}
