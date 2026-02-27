// Package plan orchestrates the plan-generation workflow.
package plan

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/defaults"
	"github.com/jumppad-labs/spektacular/internal/runner"
)

// LoadKnowledge returns all markdown files from .spektacular/knowledge/, keyed by
// their path relative to the knowledge directory.
func LoadKnowledge(projectPath string) map[string]string {
	knowledgeDir := filepath.Join(projectPath, ".spektacular", "knowledge")
	result := make(map[string]string)

	entries, err := os.ReadDir(knowledgeDir)
	if err != nil {
		return result // dir missing — no knowledge
	}

	walkDir(knowledgeDir, knowledgeDir, result, entries)
	return result
}

func walkDir(base, dir string, out map[string]string, entries []os.DirEntry) {
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			subEntries, err := os.ReadDir(path)
			if err == nil {
				walkDir(base, path, out, subEntries)
			}
			continue
		}
		if filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		rel, err := filepath.Rel(base, path)
		if err != nil {
			rel = entry.Name()
		}
		data, err := os.ReadFile(path)
		if err == nil {
			out[rel] = string(data)
		}
	}
}

// LoadAgentPrompt returns the embedded planner agent prompt.
func LoadAgentPrompt() string {
	return string(defaults.MustReadFile("agents/planner.md"))
}

// PreparePlanDir creates the plan directory and removes any stale plan.md so
// that WritePlanOutput can detect whether Claude wrote one via the Write tool.
func PreparePlanDir(planDir string) error {
	if err := os.MkdirAll(planDir, 0755); err != nil {
		return fmt.Errorf("creating plan directory: %w", err)
	}
	_ = os.Remove(filepath.Join(planDir, "plan.md"))
	return nil
}

// WritePlanOutput verifies that the agent wrote plan.md to planDir.
// The agent is always responsible for producing the file via its Write tool.
func WritePlanOutput(planDir, _ string) error {
	planFile := filepath.Join(planDir, "plan.md")
	if _, err := os.Stat(planFile); err != nil {
		return fmt.Errorf("agent did not produce plan.md in %s", planDir)
	}
	return nil
}

// RunPlan executes the full plan-generation loop for specPath.
// It prints progress to stdout and returns the plan directory path on success.
// onText is called with each text chunk from the agent (may be nil).
// onQuestion is called when questions are detected; it must return the answer string.
func RunPlan(
	specPath, projectPath string,
	cfg config.Config,
	onText func(string),
	onQuestion func([]runner.Question) string,
) (string, error) {
	specContent, err := os.ReadFile(specPath)
	if err != nil {
		return "", fmt.Errorf("reading spec file: %w", err)
	}

	agentPrompt := LoadAgentPrompt()
	prompt := runner.BuildPrompt(string(specContent))

	specName := stripExt(filepath.Base(specPath))
	planDir := filepath.Join(projectPath, ".spektacular", "plans", specName)

	if err := PreparePlanDir(planDir); err != nil {
		return "", err
	}

	if cfg.Debug.Enabled {
		debugDir := filepath.Join(projectPath, ".spektacular", "debug")
		_ = os.MkdirAll(debugDir, 0755)
		_ = os.WriteFile(filepath.Join(debugDir, "plan-prompt.md"), []byte(prompt), 0644)
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
			Command:      "plan",
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
		if err := WritePlanOutput(planDir, finalResult); err != nil {
			return "", err
		}
		return planDir, nil
	}
}

func stripExt(name string) string {
	ext := filepath.Ext(name)
	if ext == "" {
		return name
	}
	return name[:len(name)-len(ext)]
}
