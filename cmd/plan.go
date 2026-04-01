package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jumppad-labs/spektacular/internal/output"
	"github.com/jumppad-labs/spektacular/internal/plan"
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

// planDataDir returns the .spektacular/plan-<name> directory.
func planDataDir(name string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	return filepath.Join(cwd, ".spektacular", "plan-"+name), nil
}

// planStateFilePath returns the state.json path within the plan data dir.
func planStateFilePath(dataDir string) string {
	return filepath.Join(dataDir, "state.json")
}

func runPlanNew(cmd *cobra.Command, _ []string) error {
	if schema, _ := cmd.Flags().GetBool("schema"); schema {
		s := commandSchema{
			Input: &schemaObj{
				Type: "object",
				Properties: map[string]*schemaProp{
					"name":     {Type: "string", Pattern: "^[a-z0-9_-]+$", MaxLen: 64},
					"overview": {Type: "string"},
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
		Name     string `json:"name"`
		Overview string `json:"overview"`
	}
	if err := json.Unmarshal([]byte(dataStr), &input); err != nil {
		return fmt.Errorf("parsing --data: %w", err)
	}
	if input.Name == "" || !nameRegexp.MatchString(input.Name) || len(input.Name) > 64 {
		return fmt.Errorf("name must match ^[a-z0-9_-]+$ and be at most 64 characters")
	}

	dataDir, err := planDataDir(input.Name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("creating plan data directory: %w", err)
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	statePath := planStateFilePath(dataDir)
	if dryRun {
		statePath += ".dryrun-tmp"
	} else {
		_ = os.Remove(statePath)
	}

	wfCfg := workflow.Config{Command: cfg.Command, DryRun: dryRun}
	steps := plan.Steps()
	wf := workflow.New(steps, statePath, wfCfg, store.NewFileStore(filepath.Join(dataDir, "..")))
	wf.SetData("name", input.Name)
	if input.Overview != "" {
		wf.SetData("overview", input.Overview)
	}

	stdinKey, _ := cmd.Flags().GetString("stdin")
	if err := readStdinIntoWorkflow(cmd, wf, stdinKey); err != nil {
		return err
	}

	// Advance through the internal "new" step (initializes state, no output).
	if err := wf.Next(); err != nil {
		return output.WriteError(cmd.ErrOrStderr(), err)
	}

	// Advance to "overview" and write the instruction.
	out := output.New(cmd.OutOrStdout(), globalFields)
	if err := wf.Next(out); err != nil {
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
					"step": {Type: "string", Enum: workflow.New(plan.Steps(), "", workflow.Config{}, nil).StepNames()},
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

	// We need the plan name from the persisted state. Find the plan data dir.
	planName, dataDir, err := findActivePlan()
	if err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	wfCfg := workflow.Config{Command: cfg.Command, DryRun: dryRun}
	steps := plan.Steps()
	wf := workflow.New(steps, planStateFilePath(dataDir), wfCfg, store.NewFileStore(filepath.Join(dataDir, "..")))

	for k, v := range input {
		if k != "step" {
			wf.SetData(k, v)
		}
	}

	// Ensure name is set from state.
	if _, ok := wf.GetData("name"); !ok {
		wf.SetData("name", planName)
	}

	stdinKey, _ := cmd.Flags().GetString("stdin")
	if err := readStdinIntoWorkflow(cmd, wf, stdinKey); err != nil {
		return err
	}

	out := output.New(cmd.OutOrStdout(), globalFields)
	if err := wf.Goto(stepVal, out); err != nil {
		return output.WriteError(cmd.ErrOrStderr(), err)
	}
	return nil
}

func runPlanStatus(cmd *cobra.Command, _ []string) error {
	if schema, _ := cmd.Flags().GetBool("schema"); schema {
		s := commandSchema{Input: nil, Output: planStatusOutputSchema}
		return output.Write(cmd.OutOrStdout(), s, "")
	}

	planName, dataDir, err := findActivePlan()
	if err != nil {
		return err
	}

	steps := plan.Steps()
	wf := workflow.New(steps, planStateFilePath(dataDir), workflow.Config{}, nil)
	st := wf.State()

	planPath := filepath.Join(filepath.Dir(dataDir), "plans", planName+".md")

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

	wf := workflow.New(plan.Steps(), "", workflow.Config{}, nil)
	out := output.New(cmd.OutOrStdout(), globalFields)
	return out.WriteResult(plan.StepsResult{Steps: wf.StepNames()})
}

// findActivePlan locates the most recently updated plan-* directory.
func findActivePlan() (string, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("getting working directory: %w", err)
	}
	spektDir := filepath.Join(cwd, ".spektacular")
	entries, err := os.ReadDir(spektDir)
	if err != nil {
		return "", "", fmt.Errorf("reading .spektacular directory: %w", err)
	}

	var latestName, latestDir string
	var latestTime int64
	for _, e := range entries {
		if !e.IsDir() || len(e.Name()) <= 5 || e.Name()[:5] != "plan-" {
			continue
		}
		dir := filepath.Join(spektDir, e.Name())
		stPath := filepath.Join(dir, "state.json")
		info, err := os.Stat(stPath)
		if err != nil {
			continue
		}
		if t := info.ModTime().UnixNano(); t > latestTime {
			latestTime = t
			latestName = e.Name()[5:] // strip "plan-" prefix
			latestDir = dir
		}
	}

	if latestDir == "" {
		return "", "", fmt.Errorf("no active plan found — run 'plan new' first")
	}
	return latestName, latestDir, nil
}

func init() {
	planCmd.PersistentFlags().Bool("schema", false, "Print the input/output schema for this subcommand and exit")
	planCmd.PersistentFlags().BoolP("dry-run", "n", false, "Validate and preview without writing any files or persisting state")

	planNewCmd.Flags().StringP("data", "d", "", `JSON input (e.g. '{"name":"my-feature"}')`)
	planNewCmd.Flags().String("stdin", "", "Read stdin and store it in workflow data under this key")
	planGotoCmd.Flags().StringP("data", "d", "", `JSON input (e.g. '{"step":"discovery"}')`)
	planGotoCmd.Flags().String("stdin", "", "Read stdin and store it in workflow data under this key")

	planCmd.AddCommand(planNewCmd, planGotoCmd, planStatusCmd, planStepsCmd)
}
