// Package implement orchestrates the plan-execution workflow.
package implement

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/defaults"
	"github.com/jumppad-labs/spektacular/internal/runner"
)

// LoadAgentPrompt returns the embedded executor agent prompt.
func LoadAgentPrompt() string {
	return string(defaults.MustReadFile("agents/executor.md"))
}

// LoadPlanContent reads plan files from planDir and returns the combined content.
// plan.md is required; context.md and research.md are optional.
func LoadPlanContent(planDir string) (string, error) {
	planFile := filepath.Join(planDir, "plan.md")
	planContent, err := os.ReadFile(planFile)
	if err != nil {
		return "", fmt.Errorf("plan.md not found in %s: %w", planDir, err)
	}

	var b strings.Builder

	if ctx, err := os.ReadFile(filepath.Join(planDir, "context.md")); err == nil {
		b.WriteString("## context.md\n")
		b.Write(ctx)
		b.WriteString("\n\n")
	}

	b.WriteString("## plan.md\n")
	b.Write(planContent)
	b.WriteString("\n\n")

	if research, err := os.ReadFile(filepath.Join(planDir, "research.md")); err == nil {
		b.WriteString("## research.md\n")
		b.Write(research)
		b.WriteString("\n\n")
	}

	return b.String(), nil
}

// ResolvePlanDir resolves the plan directory from the given argument.
// It checks: (1) direct path, (2) relative to cwd, (3) plan name in .spektacular/plans/.
func ResolvePlanDir(arg, cwd string) (string, error) {
	candidates := []string{
		arg,
		filepath.Join(cwd, arg),
		filepath.Join(cwd, ".spektacular", "plans", arg),
	}
	for _, dir := range candidates {
		if _, err := os.Stat(filepath.Join(dir, "plan.md")); err == nil {
			return dir, nil
		}
	}
	return "", fmt.Errorf("plan.md not found: tried %s, %s, and .spektacular/plans/%s",
		arg, filepath.Join(cwd, arg), arg)
}

// RunImplement executes the full implementation loop for the given plan directory.
// onText is called with each text chunk from the agent (may be nil).
// onQuestion is called when questions are detected; it must return the answer string.
func RunImplement(
	planDir, projectPath string,
	cfg config.Config,
	onText func(string),
	onQuestion func([]runner.Question) string,
) (string, error) {
	planContent, err := LoadPlanContent(planDir)
	if err != nil {
		return "", err
	}

	agentPrompt := LoadAgentPrompt()
	prompt := runner.BuildPromptWithHeader(planContent, "Implementation Plan")

	if cfg.Debug.Enabled {
		debugDir := filepath.Join(projectPath, ".spektacular", "debug")
		_ = os.MkdirAll(debugDir, 0755)
		_ = os.WriteFile(filepath.Join(debugDir, "implement-prompt.md"), []byte(prompt), 0644)
	}

	r, err := runner.NewRunner(cfg)
	if err != nil {
		return "", fmt.Errorf("creating runner: %w", err)
	}

	sessionID := ""
	currentPrompt := prompt

	for {
		var questionsFound []runner.Question
		var finalResult string

		events, errc := r.Run(runner.RunOptions{
			Prompt:       currentPrompt,
			SystemPrompt: agentPrompt,
			Config:       cfg,
			SessionID:    sessionID,
			CWD:          projectPath,
			Command:      "implement",
		})

		for event := range events {
			if id := event.SessionID(); id != "" {
				sessionID = id
			}
			if text := event.TextContent(); text != "" {
				if onText != nil {
					onText(text)
				}
				questionsFound = append(questionsFound, runner.DetectQuestions(text)...)
			}
			if event.IsResult() {
				if event.IsError() {
					return "", fmt.Errorf("agent error: %s", event.ResultText())
				}
				finalResult = event.ResultText()
			}
		}

		if err := <-errc; err != nil {
			return "", fmt.Errorf("runner error: %w", err)
		}

		if len(questionsFound) > 0 && onQuestion != nil {
			answer := onQuestion(questionsFound)
			currentPrompt = answer
			continue
		}

		if finalResult == "" {
			return "", fmt.Errorf("agent completed without producing a result")
		}
		return planDir, nil
	}
}
