package plan

import (
	"fmt"

	"github.com/jumppad-labs/spektacular/internal/artifact"
	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/stepkit"
	"github.com/jumppad-labs/spektacular/internal/store"
	"github.com/jumppad-labs/spektacular/internal/workflow"
)

// PlanFilePath returns the store-relative path for a plan file.
func PlanFilePath(name string) string {
	return "plans/" + name + "/plan.md"
}

// ContextFilePath returns the store-relative path for a plan's context file.
func ContextFilePath(name string) string {
	return "plans/" + name + "/context.md"
}

// ResearchFilePath returns the store-relative path for a plan's research file.
func ResearchFilePath(name string) string {
	return "plans/" + name + "/research.md"
}

func planArtifactPath(cfg workflow.Config, kind, name string) (string, error) {
	return artifact.Path(cfg.Project, kind, name)
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
			Strategy:     strategy{},
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

// writePlan writes plan.md from the plan_template data key.
func writePlan() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		planName := stepkit.GetString(data, "name")
		content := stepkit.GetString(data, "plan_template")
		if content == "" {
			return "", fmt.Errorf("plan_template missing — submit the filled plan.md via --file or --stdin plan_template")
		}
		if !cfg.DryRun {
			planPath, err := planArtifactPath(cfg, artifact.KindPlan, planName)
			if err != nil {
				return "", err
			}
			if err := st.Write(planPath, []byte(content)); err != nil {
				return "", err
			}
			if err := recordPlanRemote(data, st, cfg, artifact.KindPlan, planName, "remote", []byte(content)); err != nil {
				return "", err
			}
		}
		return "", writeStep("write_plan", "write_context", "steps/plan/14-write_plan.md", data, out, st, cfg, nil)
	}
}

// writeContext writes context.md from the context_template data key.
func writeContext() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		planName := stepkit.GetString(data, "name")
		content := stepkit.GetString(data, "context_template")
		if content == "" {
			return "", fmt.Errorf("context_template missing — submit the filled context.md via --file or --stdin context_template")
		}
		if !cfg.DryRun {
			contextPath, err := planArtifactPath(cfg, artifact.KindContext, planName)
			if err != nil {
				return "", err
			}
			if err := st.Write(contextPath, []byte(content)); err != nil {
				return "", err
			}
			if err := recordPlanRemote(data, st, cfg, artifact.KindContext, planName, "context_remote", []byte(content)); err != nil {
				return "", err
			}
		}
		return "", writeStep("write_context", "write_research", "steps/plan/15-write_context.md", data, out, st, cfg, nil)
	}
}

// writeResearch writes research.md from the research_template data key.
func writeResearch() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		planName := stepkit.GetString(data, "name")
		content := stepkit.GetString(data, "research_template")
		if content == "" {
			return "", fmt.Errorf("research_template missing — submit the filled research.md via --file or --stdin research_template")
		}
		if !cfg.DryRun {
			researchPath, err := planArtifactPath(cfg, artifact.KindResearch, planName)
			if err != nil {
				return "", err
			}
			if err := st.Write(researchPath, []byte(content)); err != nil {
				return "", err
			}
			if err := recordPlanRemote(data, st, cfg, artifact.KindResearch, planName, "research_remote", []byte(content)); err != nil {
				return "", err
			}
		}
		return "", writeStep("write_research", "finished", "steps/plan/16-write_research.md", data, out, st, cfg, nil)
	}
}

func finished() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		if cfg.DryRun {
			return "", writeStep("finished", "", "steps/plan/17-finished.md", data, out, st, cfg, nil)
		}
		planName := stepkit.GetString(data, "name")
		paths := make([]string, 0, 3)
		for _, kind := range []string{artifact.KindPlan, artifact.KindContext, artifact.KindResearch} {
			path, err := planArtifactPath(cfg, kind, planName)
			if err != nil {
				return "", err
			}
			paths = append(paths, path)
		}
		for _, p := range paths {
			if !st.Exists(p) {
				return "", fmt.Errorf("expected file %s not found — the preceding write step should have written it", p)
			}
		}
		return "", writeStep("finished", "", "steps/plan/17-finished.md", data, out, st, cfg, nil)
	}
}

func recordPlanRemote(data workflow.Data, st store.Store, cfg workflow.Config, kind, name, key string, content []byte) error {
	if cfg.Project.Artifacts.Backend != config.ArtifactBackendNotion {
		return nil
	}
	value, ok := data.Get(key)
	if !ok || value == nil {
		return fmt.Errorf("%s metadata is required in Notion mode", key)
	}
	remote, _, err := artifact.RemoteMetadataFromAny(value)
	if err != nil {
		return fmt.Errorf("reading %s metadata: %w", key, err)
	}
	if err := artifact.ValidateRemoteMetadata(remote); err != nil {
		return err
	}
	if _, err := artifact.RecordPull(st, cfg.Project, kind, name, content, remote); err != nil {
		return err
	}
	data.Set(key, nil)
	return nil
}
