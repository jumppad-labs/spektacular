package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jumppad-labs/spektacular/internal/artifact"
	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/output"
	"github.com/jumppad-labs/spektacular/internal/steps/plan"
	"github.com/jumppad-labs/spektacular/internal/store"
	"github.com/jumppad-labs/spektacular/internal/workflow"
	"github.com/spf13/cobra"
)

var planResultOutputSchema = &schemaObj{
	Type: "object",
	Properties: map[string]*schemaProp{
		"step":        {Type: "string"},
		"plan_path":   {Type: "string"},
		"plan_name":   {Type: "string"},
		"instruction": {Type: "string"},
	},
}

var planStatusOutputSchema = &schemaObj{
	Type: "object",
	Properties: map[string]*schemaProp{
		"plan_name":       {Type: "string"},
		"plan_path":       {Type: "string"},
		"current_step":    {Type: "string"},
		"completed_steps": {Type: "array", Items: &schemaProp{Type: "string"}},
		"total_steps":     {Type: "integer"},
		"progress":        {Type: "string"},
		"steps":           {Type: "array"},
	},
}

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Manage plan workflow",
}

var planNewCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new plan workflow",
	RunE:  runPlanNew,
}

var planGotoCmd = &cobra.Command{
	Use:   "goto",
	Short: "Jump to a named step",
	RunE:  runPlanGoto,
}

var planStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current workflow progress",
	RunE:  runPlanStatus,
}

var planStepsCmd = &cobra.Command{
	Use:   "steps",
	Short: "List available workflow step names",
	RunE:  runPlanSteps,
}

func runPlanNew(cmd *cobra.Command, _ []string) error {
	if schema, _ := cmd.Flags().GetBool("schema"); schema {
		s := commandSchema{
			Input: &schemaObj{
				Type: "object",
				Properties: map[string]*schemaProp{
					"name": {Type: "string", Pattern: "^[a-z0-9_-]+$", MaxLen: 64},
				},
				Required: []string{"name"},
			},
			Output: planResultOutputSchema,
		}
		return output.Write(cmd.OutOrStdout(), s, "")
	}

	dataStr, _ := cmd.Flags().GetString("data")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if dataStr == "" {
		return fmt.Errorf("--data is required (e.g. --data '{\"name\":\"my-feature\"}')")
	}
	var input struct {
		Name           string                  `json:"name"`
		Remote         artifact.RemoteMetadata `json:"remote"`
		ContextRemote  artifact.RemoteMetadata `json:"context_remote"`
		ResearchRemote artifact.RemoteMetadata `json:"research_remote"`
	}
	if err := json.Unmarshal([]byte(dataStr), &input); err != nil {
		return fmt.Errorf("parsing --data: %w", err)
	}
	if input.Name == "" || !nameRegexp.MatchString(input.Name) || len(input.Name) > 64 {
		return fmt.Errorf("name must match ^[a-z0-9_-]+$ and be at most 64 characters")
	}

	dataDir, err := dataDir()
	if err != nil {
		return err
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	statePath := stateFilePath(dataDir)
	if dryRun {
		statePath += ".dryrun-tmp"
	} else {
		_ = os.Remove(statePath)
	}

	wfCfg := workflow.Config{Command: cfg.Command, DryRun: dryRun, Project: cfg}
	steps := plan.Steps()
	out := output.New(cmd.OutOrStdout(), globalFields)
	wf := workflow.New(steps, statePath, wfCfg, store.NewFileStore(dataDir), out)
	wf.SetData("name", input.Name)
	if cfg.Artifacts.Backend == config.ArtifactBackendNotion {
		if input.Remote.RemoteVersion != "" {
			wf.SetData("remote", input.Remote)
		}
		if input.ContextRemote.RemoteVersion != "" {
			wf.SetData("context_remote", input.ContextRemote)
		}
		if input.ResearchRemote.RemoteVersion != "" {
			wf.SetData("research_remote", input.ResearchRemote)
		}
	}

	if err := readInputIntoWorkflow(cmd, wf); err != nil {
		return err
	}

	if err := wf.Next(); err != nil {
		return output.WriteError(cmd.ErrOrStderr(), err)
	}
	return nil
}

func runPlanGoto(cmd *cobra.Command, _ []string) error {
	if schema, _ := cmd.Flags().GetBool("schema"); schema {
		s := commandSchema{
			Input: &schemaObj{
				Type: "object",
				Properties: map[string]*schemaProp{
					"step": {Type: "string", Enum: workflow.New(plan.Steps(), "", workflow.Config{}, nil, nil).StepNames()},
				},
				Required: []string{"step"},
			},
			Output: planResultOutputSchema,
		}
		return output.Write(cmd.OutOrStdout(), s, "")
	}

	dataStr, _ := cmd.Flags().GetString("data")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if dataStr == "" {
		return fmt.Errorf("--data is required (e.g. --data '{\"step\":\"discovery\"}')")
	}
	var input map[string]any
	if err := json.Unmarshal([]byte(dataStr), &input); err != nil {
		return fmt.Errorf("parsing --data: %w", err)
	}
	stepVal, _ := input["step"].(string)
	if stepVal == "" {
		return fmt.Errorf("\"step\" is required in --data")
	}

	dataDir, err := dataDir()
	if err != nil {
		return err
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	wfCfg := workflow.Config{Command: cfg.Command, DryRun: dryRun, Project: cfg}
	steps := plan.Steps()
	out := output.New(cmd.OutOrStdout(), globalFields)
	wf := workflow.New(steps, stateFilePath(dataDir), wfCfg, store.NewFileStore(dataDir), out)

	for k, v := range input {
		if k != "step" {
			wf.SetData(k, v)
		}
	}

	if _, ok := wf.GetData("name"); !ok {
		return fmt.Errorf("no active plan found — run 'plan new' first")
	}

	if err := readInputIntoWorkflow(cmd, wf); err != nil {
		return err
	}

	if err := wf.Goto(stepVal); err != nil {
		return output.WriteError(cmd.ErrOrStderr(), err)
	}
	return nil
}

func runPlanStatus(cmd *cobra.Command, _ []string) error {
	if schema, _ := cmd.Flags().GetBool("schema"); schema {
		s := commandSchema{Input: nil, Output: planStatusOutputSchema}
		return output.Write(cmd.OutOrStdout(), s, "")
	}

	dataDir, err := dataDir()
	if err != nil {
		return err
	}

	steps := plan.Steps()
	wf := workflow.New(steps, stateFilePath(dataDir), workflow.Config{}, nil, nil)
	st := wf.State()

	nameVal, ok := wf.GetData("name")
	if !ok {
		return fmt.Errorf("no active plan found — run 'plan new' first")
	}
	planName := fmt.Sprintf("%v", nameVal)
	planPath := filepath.Join(dataDir, plan.PlanFilePath(planName))

	stepInfos := wf.StepStatus()
	entries := make([]plan.StepEntry, len(stepInfos))
	for i, info := range stepInfos {
		entries[i] = plan.StepEntry{Name: info.Name, Status: info.Status}
	}

	out := output.New(cmd.OutOrStdout(), globalFields)
	return out.WriteResult(plan.StatusResult{
		PlanName:       planName,
		PlanPath:       planPath,
		CurrentStep:    wf.Current(),
		CompletedSteps: st.CompletedSteps,
		TotalSteps:     len(steps),
		Progress:       fmt.Sprintf("%d/%d", len(st.CompletedSteps), len(steps)),
		Steps:          entries,
	})
}

func runPlanSteps(cmd *cobra.Command, _ []string) error {
	if schema, _ := cmd.Flags().GetBool("schema"); schema {
		s := commandSchema{
			Input: nil,
			Output: &schemaObj{
				Type: "object",
				Properties: map[string]*schemaProp{
					"steps": {Type: "array", Items: &schemaProp{Type: "string"}},
				},
			},
		}
		return output.Write(cmd.OutOrStdout(), s, "")
	}

	wf := workflow.New(plan.Steps(), "", workflow.Config{}, nil, nil)
	out := output.New(cmd.OutOrStdout(), globalFields)
	return out.WriteResult(plan.StepsResult{Steps: wf.StepNames()})
}

func init() {
	planCmd.PersistentFlags().Bool("schema", false, "Print the input/output schema for this subcommand and exit")
	planCmd.PersistentFlags().BoolP("dry-run", "n", false, "Validate and preview without writing any files or persisting state")

	planNewCmd.Flags().StringP("data", "d", "", `JSON input (e.g. '{"name":"my-feature"}')`)
	planNewCmd.Flags().String("stdin", "", "Read stdin and store it in workflow data under this key")
	planNewCmd.Flags().String("file", "", "Read a file at <path> (relative to cwd) and store its contents under the filename's basename (without extension)")
	planGotoCmd.Flags().StringP("data", "d", "", `JSON input (e.g. '{"step":"discovery"}')`)
	planGotoCmd.Flags().String("stdin", "", "Read stdin and store it in workflow data under this key")
	planGotoCmd.Flags().String("file", "", "Read a file at <path> (relative to cwd) and store its contents under the filename's basename (without extension)")

	planCmd.AddCommand(planNewCmd, planGotoCmd, planStatusCmd, planStepsCmd)
}
