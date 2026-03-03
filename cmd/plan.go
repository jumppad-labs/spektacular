package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/plan"
	"github.com/jumppad-labs/spektacular/internal/runner"
	"github.com/jumppad-labs/spektacular/internal/spec"
	"github.com/jumppad-labs/spektacular/internal/steps"
	"github.com/jumppad-labs/spektacular/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var planCmd = &cobra.Command{
	Use:   "plan <spec-file>",
	Short: "Generate an implementation plan from a specification",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}

		specFile, err := spec.ResolveSpecFile(args[0], cwd)
		if err != nil {
			return err
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

		var planDirOut string
		if term.IsTerminal(int(os.Stdout.Fd())) {
			wf := steps.PlanWorkflow(specFile, cwd, cfg)
			planDirOut, err = tui.RunAgentTUI(wf, cwd, cfg)
			if err != nil {
				return fmt.Errorf("plan generation failed: %w", err)
			}
		} else {
			// No TTY — stream output to stdout directly
			planDirOut, err = plan.RunPlan(specFile, cwd, cfg,
				func(text string) { fmt.Print(text) },
				func(questions []runner.Question) string {
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
		if planDirOut != "" {
			fmt.Printf("Plan generated: %s\n", planDirOut)
		}
		return nil
	},
}
