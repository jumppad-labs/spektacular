package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nicholasjackson/spektacular/internal/config"
	"github.com/nicholasjackson/spektacular/internal/plan"
	"github.com/nicholasjackson/spektacular/internal/runner"
	"github.com/nicholasjackson/spektacular/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var planCmd = &cobra.Command{
	Use:   "plan <spec-file>",
	Short: "Generate an implementation plan from a specification",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		specFile := args[0]

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}

		configPath := filepath.Join(cwd, ".spektacular", "config.yaml")
		var cfg config.Config
		if _, err := os.Stat(configPath); err == nil {
			cfg, err = config.FromYAMLFile(configPath)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
		} else {
			cfg = config.NewDefault()
		}

		var planDir string
		if term.IsTerminal(int(os.Stdout.Fd())) {
			planDir, err = tui.RunPlanTUI(specFile, cwd, cfg)
			if err != nil {
				return fmt.Errorf("plan generation failed: %w", err)
			}
		} else {
			// No TTY — stream output to stdout directly
			planDir, err = plan.RunPlan(specFile, cwd, cfg,
				func(text string) { fmt.Print(text) },
				func(questions []runner.Question) string {
					// Non-interactive: print question and return empty answer
					if len(questions) > 0 {
						fmt.Printf("\n[Question] %s\n", questions[0].Question)
					}
					return ""
				},
			)
			if err != nil {
				return fmt.Errorf("plan generation failed: %w", err)
			}
		}
		if planDir != "" {
			fmt.Printf("Plan generated: %s\n", planDir)
		}
		return nil
	},
}
