package plan

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cbroglie/mustache"
	"github.com/jumppad-labs/spektacular/internal/store"
	"github.com/jumppad-labs/spektacular/internal/workflow"
	"github.com/jumppad-labs/spektacular/templates"
)

// PlanFilePath returns the store-relative path for a plan file.
func PlanFilePath(name string) string {
	return "plans/" + name + ".md"
}

// Steps returns the ordered step configs for a plan workflow.
func Steps() []workflow.StepConfig {
	return []workflow.StepConfig{
		{Name: "new", Src: []string{"start"}, Dst: "new", Callback: newStep()},
		{Name: "overview", Src: []string{"new"}, Dst: "overview", Callback: overview()},
		{Name: "discovery", Src: []string{"overview"}, Dst: "discovery", Callback: discovery()},
		{Name: "approach", Src: []string{"discovery"}, Dst: "approach", Callback: approach()},
		{Name: "milestones", Src: []string{"approach"}, Dst: "milestones", Callback: milestones()},
		{Name: "phases", Src: []string{"milestones"}, Dst: "phases", Callback: phases()},
		{Name: "verification", Src: []string{"phases"}, Dst: "verification", Callback: verification()},
		{Name: "finished", Src: []string{"verification"}, Dst: "finished", Callback: finished()},
	}
}

// newStep initializes state only — no document created yet.
func newStep() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) error {
		if cfg.DryRun {
			return nil
		}
		return nil
	}
}

func overview() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) error {
		return writeStepResult("overview", "discovery", "plan-steps/overview.md", data, out, st, cfg)
	}
}

func discovery() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) error {
		return writeStepResult("discovery", "approach", "plan-steps/discovery.md", data, out, st, cfg)
	}
}

func approach() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) error {
		return writeStepResult("approach", "milestones", "plan-steps/approach.md", data, out, st, cfg)
	}
}

func milestones() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) error {
		return writeStepResult("milestones", "phases", "plan-steps/milestones.md", data, out, st, cfg)
	}
}

func phases() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) error {
		return writeStepResult("phases", "verification", "plan-steps/phases.md", data, out, st, cfg)
	}
}

func verification() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) error {
		planName := getString(data, "name")
		scaffold, err := renderTemplate("plan-scaffold.md", map[string]any{"name": planName})
		if err != nil {
			return err
		}
		return writeStepResult("verification", "finished", "plan-steps/verification.md", data, out, st, cfg,
			map[string]any{"plan_template": scaffold})
	}
}

func finished() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) error {
		planName := getString(data, "name")
		planPath := PlanFilePath(planName)
		if content := getString(data, "plan_template"); content != "" {
			if err := st.Write(planPath, []byte(content)); err != nil {
				return err
			}
		}
		return writeStepResult("finished", "", "plan-steps/finished.md", data, out, st, cfg)
	}
}

// writeStepResult renders the step template and writes the result to out.
func writeStepResult(name, nextStep, templatePath string, data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config, extra ...map[string]any) error {
	planName := getString(data, "name")
	planPath := filepath.Join(st.Root(), PlanFilePath(planName))
	overview := getString(data, "overview")

	vars := map[string]any{
		"step":      name,
		"title":     stepTitle(name),
		"plan_path": planPath,
		"plan_name": planName,
		"next_step": nextStep,
		"overview":  overview,
		"config":    map[string]any{"command": cfg.Command},
	}
	for _, m := range extra {
		for k, v := range m {
			vars[k] = v
		}
	}

	instruction, err := renderTemplate(templatePath, vars)
	if err != nil {
		return err
	}
	return out.WriteResult(Result{
		Step:        name,
		PlanPath:    planPath,
		PlanName:    planName,
		Instruction: instruction,
	})
}

func getString(data workflow.Data, key string) string {
	v, ok := data.Get(key)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func renderTemplate(templatePath string, data map[string]any) (string, error) {
	tmplBytes, err := templates.FS.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("loading template %s: %w", templatePath, err)
	}
	return mustache.Render(string(tmplBytes), data)
}

func stepTitle(name string) string {
	words := strings.Split(name, "_")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
