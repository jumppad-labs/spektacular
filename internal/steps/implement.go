package steps

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/implement"
	"github.com/jumppad-labs/spektacular/internal/runner"
	"github.com/jumppad-labs/spektacular/internal/tui"
)

// ImplementWorkflow returns the TUI workflow for executing an implementation plan.
func ImplementWorkflow(planDir, projectPath string, cfg config.Config) tui.Workflow {
	logFile := ""
	if cfg.Debug.Enabled && cfg.Debug.LogDir != "" {
		logDir := filepath.Join(projectPath, cfg.Debug.LogDir)
		_ = os.MkdirAll(logDir, 0755)
		logFile = filepath.Join(logDir, time.Now().Format("2006-01-02_15-04-05")+"_implement.log")
	}

	return tui.Workflow{
		LogFile: logFile,
		Steps:   []tui.WorkflowStep{implementStep(planDir)},
		OnDone:  func() (string, error) { return planDir, nil },
	}
}

func implementStep(planDir string) tui.WorkflowStep {
	systemPrompt := implement.LoadAgentPrompt()

	return tui.WorkflowStep{
		Name:        "implement",
		StatusLabel: filepath.Base(planDir),
		BuildRunOptions: func(cfg config.Config, cwd string) (runner.RunOptions, error) {
			planContent, err := implement.LoadPlanContent(planDir)
			if err != nil {
				return runner.RunOptions{}, fmt.Errorf("loading plan: %w", err)
			}
			if cfg.Debug.Enabled {
				debugDir := filepath.Join(cwd, ".spektacular", "debug")
				_ = os.MkdirAll(debugDir, 0755)
				_ = os.WriteFile(filepath.Join(debugDir, "implement-prompt.md"), []byte(planContent), 0644)
			}
			return runner.RunOptions{
				Prompts: runner.Prompts{
					User:   runner.BuildPromptWithHeader(planContent, "Implementation Plan"),
					System: systemPrompt,
				},
				Config: cfg,
				CWD:    cwd,
				Model:  "claude-sonnet-4-6",
			}, nil
		},
	}
}
