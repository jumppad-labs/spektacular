// Package plan orchestrates the plan-generation workflow.
package plan

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nicholasjackson/spektacular/internal/config"
	"github.com/nicholasjackson/spektacular/internal/defaults"
	"github.com/nicholasjackson/spektacular/internal/runner"
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

// WritePlanOutput writes the agent result text to <planDir>/plan.md.
func WritePlanOutput(planDir, resultText string) error {
	if err := os.MkdirAll(planDir, 0755); err != nil {
		return fmt.Errorf("creating plan directory: %w", err)
	}
	planFile := filepath.Join(planDir, "plan.md")
	if err := os.WriteFile(planFile, []byte(resultText), 0644); err != nil {
		return fmt.Errorf("writing plan.md: %w", err)
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
	knowledge := LoadKnowledge(projectPath)
	prompt := runner.BuildPrompt(string(specContent), agentPrompt, knowledge)

	specName := stripExt(filepath.Base(specPath))
	planDir := filepath.Join(projectPath, ".spektacular", "plans", specName)

	if cfg.Debug.Enabled {
		if err := os.MkdirAll(planDir, 0755); err == nil {
			_ = os.WriteFile(filepath.Join(planDir, "prompt.md"), []byte(prompt), 0644)
		}
	}

	sessionID := ""
	currentPrompt := prompt

	for {
		var questionsFound []runner.Question
		var finalResult string

		events, errc := runner.RunClaude(runner.RunOptions{
			Prompt:    currentPrompt,
			Config:    cfg,
			SessionID: sessionID,
			CWD:       projectPath,
			Command:   "plan",
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
