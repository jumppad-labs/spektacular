package spec

import (
	"fmt"

	"github.com/jumppad-labs/spektacular/internal/artifact"
	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/stepkit"
	"github.com/jumppad-labs/spektacular/internal/store"
	"github.com/jumppad-labs/spektacular/internal/workflow"
)

// SpecFilePath returns the store-relative path for a spec file.
func SpecFilePath(name string) string {
	return "specs/" + name + ".md"
}

func specArtifactPath(cfg workflow.Config, name string) (string, error) {
	return artifact.Path(cfg.Project, artifact.KindSpec, name)
}

// Steps returns the ordered step configs for a spec workflow.
// Each step has an explicit named callback — no string-based dispatch.
// The first step "new" is internal: it creates the spec file and produces no
// output, allowing the caller to automatically advance to "overview".
func Steps() []workflow.StepConfig {
	return []workflow.StepConfig{
		{Name: "new", Src: []string{"start"}, Dst: "new", Callback: new()},
		{Name: "overview", Src: []string{"new"}, Dst: "overview", Callback: overview()},
		{Name: "requirements", Src: []string{"overview"}, Dst: "requirements", Callback: requirements()},
		{Name: "acceptance_criteria", Src: []string{"requirements"}, Dst: "acceptance_criteria", Callback: acceptanceCriteria()},
		{Name: "constraints", Src: []string{"acceptance_criteria"}, Dst: "constraints", Callback: constraints()},
		{Name: "technical_approach", Src: []string{"constraints"}, Dst: "technical_approach", Callback: technicalApproach()},
		{Name: "success_metrics", Src: []string{"technical_approach"}, Dst: "success_metrics", Callback: successMetrics()},
		{Name: "non_goals", Src: []string{"success_metrics"}, Dst: "non_goals", Callback: nonGoals()},
		{Name: "verification", Src: []string{"non_goals"}, Dst: "verification", Callback: verification()},
		{Name: "finished", Src: []string{"verification"}, Dst: "finished", Callback: finished()},
	}
}

// buildResult is the stepkit.ResultBuilder for the spec workflow.
func buildResult(stepName, instanceName, primaryPath, instruction string) any {
	return Result{
		Step:        stepName,
		SpecPath:    primaryPath,
		SpecName:    instanceName,
		Instruction: instruction,
	}
}

// writeStep is a thin wrapper around stepkit.WriteStepResult pre-applied with
// the spec strategy and result builder.
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

// new creates the spec file and produces no output.
// The caller is expected to immediately advance to "overview".
func new() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		if cfg.DryRun {
			return "overview", nil
		}
		if st == nil {
			return "", fmt.Errorf("store required for new step")
		}
		name := stepkit.GetString(data, "name")
		rendered, err := stepkit.RenderTemplate("scaffold/spec.md", map[string]any{"name": name})
		if err != nil {
			return "", err
		}
		specPath, err := specArtifactPath(cfg, name)
		if err != nil {
			return "", err
		}
		if err := st.Write(specPath, []byte(rendered)); err != nil {
			return "", err
		}
		if err := recordSpecRemote(data, st, cfg, name, []byte(rendered), true); err != nil {
			return "", err
		}
		return "overview", nil
	}
}

func overview() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		return "", writeStep("overview", "requirements", "steps/spec/01-overview.md", data, out, st, cfg, nil)
	}
}

func requirements() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		return "", writeStep("requirements", "acceptance_criteria", "steps/spec/02-requirements.md", data, out, st, cfg, nil)
	}
}

func acceptanceCriteria() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		return "", writeStep("acceptance_criteria", "constraints", "steps/spec/03-acceptance_criteria.md", data, out, st, cfg, nil)
	}
}

func constraints() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		return "", writeStep("constraints", "technical_approach", "steps/spec/04-constraints.md", data, out, st, cfg, nil)
	}
}

func technicalApproach() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		return "", writeStep("technical_approach", "success_metrics", "steps/spec/05-technical_approach.md", data, out, st, cfg, nil)
	}
}

func successMetrics() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		return "", writeStep("success_metrics", "non_goals", "steps/spec/06-success_metrics.md", data, out, st, cfg, nil)
	}
}

func nonGoals() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		return "", writeStep("non_goals", "verification", "steps/spec/07-non_goals.md", data, out, st, cfg, nil)
	}
}

func verification() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		specName := stepkit.GetString(data, "name")
		scaffold, err := stepkit.RenderTemplate("scaffold/spec.md", map[string]any{"name": specName})
		if err != nil {
			return "", err
		}
		return "", writeStep("verification", "finished", "steps/spec/08-verification.md", data, out, st, cfg, map[string]any{
			"spec_template": scaffold,
		})
	}
}

func finished() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		specName := stepkit.GetString(data, "name")
		if content := stepkit.GetString(data, "spec_template"); content != "" {
			specPath, err := specArtifactPath(cfg, specName)
			if err != nil {
				return "", err
			}
			if err := st.Write(specPath, []byte(content)); err != nil {
				return "", err
			}
			if err := recordSpecRemote(data, st, cfg, specName, []byte(content), true); err != nil {
				return "", err
			}
		}
		return "", writeStep("finished", "", "steps/spec/09-finished.md", data, out, st, cfg, nil)
	}
}

func recordSpecRemote(data workflow.Data, st store.Store, cfg workflow.Config, name string, content []byte, required bool) error {
	if cfg.Project.Artifacts.Backend != config.ArtifactBackendNotion {
		return nil
	}
	value, ok := data.Get("remote")
	if !ok || value == nil {
		if required {
			return fmt.Errorf("remote metadata is required in Notion mode")
		}
		return nil
	}
	remote, _, err := artifact.RemoteMetadataFromAny(value)
	if err != nil {
		return fmt.Errorf("reading remote metadata: %w", err)
	}
	if err := artifact.ValidateRemoteMetadata(remote); err != nil {
		return err
	}
	if _, err := artifact.RecordPull(st, cfg.Project, artifact.KindSpec, name, content, remote); err != nil {
		return err
	}
	data.Set("remote", nil)
	return nil
}
