package implement

import (
	"github.com/jumppad-labs/spektacular/internal/stepkit"
	"github.com/jumppad-labs/spektacular/internal/store"
	"github.com/jumppad-labs/spektacular/internal/workflow"
)

// Steps returns the ordered step configs for an implement workflow.
//
// The FSM uses a multi-source transition on analyze: both `read_plan` and
// `update_changelog` can lead into `analyze`. This encodes the phase-loop
// directly in the FSM declaration — when `update_changelog` detects remaining
// unchecked phases in the plan, it advances back to `analyze`; otherwise it
// advances to `update_repo_changelog`.
func Steps() []workflow.StepConfig {
	return []workflow.StepConfig{
		{Name: "new", Src: []string{"start"}, Dst: "new", Callback: newStep()},
		{Name: "read_plan", Src: []string{"new"}, Dst: "read_plan", Callback: readPlan()},
		{Name: "analyze", Src: []string{"read_plan", "update_changelog"}, Dst: "analyze", Callback: analyze()},
		{Name: "implement", Src: []string{"analyze"}, Dst: "implement", Callback: implementStep()},
		{Name: "test", Src: []string{"implement"}, Dst: "test", Callback: testStep()},
		{Name: "verify", Src: []string{"test"}, Dst: "verify", Callback: verify()},
		{Name: "update_plan", Src: []string{"verify"}, Dst: "update_plan", Callback: updatePlan()},
		{Name: "update_changelog", Src: []string{"update_plan"}, Dst: "update_changelog", Callback: updateChangelog()},
		{Name: "update_repo_changelog", Src: []string{"update_changelog"}, Dst: "update_repo_changelog", Callback: updateRepoChangelog()},
		{Name: "finished", Src: []string{"update_repo_changelog"}, Dst: "finished", Callback: finished()},
	}
}

// buildResult is the stepkit.ResultBuilder for the implement workflow.
func buildResult(stepName, instanceName, primaryPath, instruction string) any {
	return Result{
		Step:        stepName,
		PlanPath:    primaryPath,
		PlanName:    instanceName,
		Instruction: instruction,
	}
}

// writeStep is a thin wrapper around stepkit.WriteStepResult pre-applied with
// the implement strategy and result builder.
func writeStep(stepName, nextStep, templatePath string, data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config, extra map[string]any) error {
	return stepkit.WriteStepResult(
		stepkit.StepRequest{
			StepName:     stepName,
			NextStep:     nextStep,
			TemplatePath: templatePath,
			Strategy:     strategy{planDir: cfg.PlanDir},
			Extra:        extra,
		},
		data, out, st, cfg,
		buildResult,
	)
}

// newStep initializes state only — auto-advances to read_plan without writing
// anything. Mirrors the plan workflow's `new` callback.
func newStep() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		return "read_plan", nil
	}
}

func readPlan() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		return "", writeStep("read_plan", "analyze", "steps/implement/01-read_plan.md", data, out, st, cfg, nil)
	}
}

func analyze() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		return "", writeStep("analyze", "implement", "steps/implement/02-analyze.md", data, out, st, cfg, nil)
	}
}

func implementStep() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		return "", writeStep("implement", "test", "steps/implement/03-implement.md", data, out, st, cfg, nil)
	}
}

func testStep() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		return "", writeStep("test", "verify", "steps/implement/04-test.md", data, out, st, cfg, nil)
	}
}

func verify() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		return "", writeStep("verify", "update_plan", "steps/implement/05-verify.md", data, out, st, cfg, nil)
	}
}

func updatePlan() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		return "", writeStep("update_plan", "update_changelog", "steps/implement/06-update_plan.md", data, out, st, cfg, nil)
	}
}

// updateChangelog has two legal exits encoded in the template:
//   - goto analyze (loop back) when unchecked phases remain
//   - goto update_repo_changelog when no unchecked phases remain
//
// NextStep is set to "update_repo_changelog" for the default advance path; the
// template instructs the agent to branch based on plan-file state.
func updateChangelog() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		return "", writeStep("update_changelog", "update_repo_changelog", "steps/implement/07-update_changelog.md", data, out, st, cfg, nil)
	}
}

func updateRepoChangelog() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		return "", writeStep("update_repo_changelog", "finished", "steps/implement/08-update_repo_changelog.md", data, out, st, cfg, nil)
	}
}

func finished() workflow.StepCallback {
	return func(data workflow.Data, out workflow.ResultWriter, st store.Store, cfg workflow.Config) (string, error) {
		return "", writeStep("finished", "", "steps/implement/09-finished.md", data, out, st, cfg, nil)
	}
}
