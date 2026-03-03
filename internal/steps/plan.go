// Package steps defines the TUI workflow steps for each Spektacular command.
package steps

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/plan"
	"github.com/jumppad-labs/spektacular/internal/runner"
	"github.com/jumppad-labs/spektacular/internal/tui"
)

// PlanWorkflow returns the TUI workflow for generating a plan from a spec file.
func PlanWorkflow(specFile, projectPath string, cfg config.Config) tui.Workflow {
	planDir := filepath.Join(projectPath, ".spektacular", "plans", stripExt(filepath.Base(specFile)))

	logFile := ""
	if cfg.Debug.Enabled && cfg.Debug.LogDir != "" {
		logDir := filepath.Join(projectPath, cfg.Debug.LogDir)
		_ = os.MkdirAll(logDir, 0755)
		logFile = filepath.Join(logDir, time.Now().Format("2006-01-02_15-04-05")+"_plan.log")
	}

	specName := stripExt(filepath.Base(specFile))
	return tui.Workflow{
		LogFile: logFile,
		Preamble: "## Planning: " + specName + "\n\n" +
			"I'll read your specification and generate a structured implementation plan. " +
			"The plan will be written to `.spektacular/plans/" + specName + "/` and includes:\n\n" +
			"- **Tasks** — ordered, actionable implementation steps\n" +
			"- **Context** — architectural notes and key decisions\n" +
			"- **Research** — relevant patterns and references\n\n" +
			"I may ask clarifying questions if the spec is ambiguous.",
		Steps: []tui.WorkflowStep{planStep(specFile, planDir), feedbackStep(planDir)},
		OnDone: func() (string, error) {
			if err := plan.WritePlanOutput(planDir, ""); err != nil {
				return "", err
			}
			return planDir, nil
		},
	}
}

func feedbackStep(planDir string) tui.WorkflowStep {
	return tui.WorkflowStep{
		Name:        "feedback",
		StatusLabel: "feedback",
		BuildRunOptions: func(cfg config.Config, cwd string) (runner.RunOptions, error) {
			relPlanDir, err := filepath.Rel(cwd, planDir)
			if err != nil {
				relPlanDir = planDir
			}
			return runner.RunOptions{
				Prompts: runner.Prompts{
					User: buildFeedbackPrompt(relPlanDir),
				},
				Config: cfg,
				CWD:    cwd,
				Model:  "claude-sonnet-4-6",
			}, nil
		},
	}
}

func buildFeedbackPrompt(planDir string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Re-read all plan files in `%s` (plan.md, and context.md/research.md if they exist). ", planDir)
	b.WriteString("The user may have edited them directly.\n\n")
	b.WriteString("Ask the user for feedback:\n\n")
	b.WriteString("<!--QUESTION:{\"questions\":[{\"question\":\"Please review the plan above. Do you have any feedback or changes?\\n\\nLeave blank and submit to finish.\",\"header\":\"Plan Feedback\",\"type\":\"text\"}]}-->\n\n")
	b.WriteString("If the user provides feedback:\n")
	b.WriteString("- If unclear, ask one clarifying question using the same format before making changes\n")
	b.WriteString("- Incorporate the feedback into the plan files\n")
	b.WriteString("- Output: <!-- GOTO: feedback -->\n\n")
	b.WriteString("If the user leaves the response empty or says they are done:\n")
	b.WriteString("- Output: <!-- FINISHED -->")
	return b.String()
}

func planStep(specFile, planDir string) tui.WorkflowStep {
	systemPrompt := plan.LoadAgentPrompt()

	return tui.WorkflowStep{
		Name:        "plan",
		StatusLabel: filepath.Base(specFile),
		BuildRunOptions: func(cfg config.Config, cwd string) (runner.RunOptions, error) {
			specContent, err := os.ReadFile(specFile)
			if err != nil {
				return runner.RunOptions{}, fmt.Errorf("reading spec: %w", err)
			}
			if err := plan.PreparePlanDir(planDir); err != nil {
				return runner.RunOptions{}, err
			}
			if cfg.Debug.Enabled {
				debugDir := filepath.Join(cwd, ".spektacular", "debug")
				_ = os.MkdirAll(debugDir, 0755)
				_ = os.WriteFile(filepath.Join(debugDir, "plan-prompt.md"), specContent, 0644)
			}
			// Use a relative plan dir so the path is stable regardless of machine.
			relPlanDir, relErr := filepath.Rel(cwd, planDir)
			if relErr != nil {
				relPlanDir = planDir
			}
			return runner.RunOptions{
				Prompts: runner.Prompts{
					User:   runner.BuildPlanPrompt(string(specContent), relPlanDir),
					System: systemPrompt,
				},
				Config: cfg,
				CWD:    cwd,
				Model:  "claude-opus-4-6",
			}, nil
		},
	}
}
