package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/jumppad-labs/spektacular/internal/artifact"
	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/output"
	"github.com/jumppad-labs/spektacular/internal/steps/spec"
	"github.com/jumppad-labs/spektacular/internal/store"
	"github.com/jumppad-labs/spektacular/internal/workflow"
	"github.com/spf13/cobra"
)

var nameRegexp = regexp.MustCompile(`^[a-z0-9_-]+$`)

const identifierInputPattern = `^[^\s/\\\x00-\x1F\x7F](?:[^/\\\x00-\x1F\x7F]*[^\s/\\\x00-\x1F\x7F])?$`

var specIdentifierNow = time.Now

// Schema types for --schema output.
type schemaProp struct {
	Type    string      `json:"type"`
	Enum    []string    `json:"enum,omitempty"`
	Pattern string      `json:"pattern,omitempty"`
	MaxLen  int         `json:"maxLength,omitempty"`
	Items   *schemaProp `json:"items,omitempty"`
}

type schemaObj struct {
	Type       string                 `json:"type"`
	Properties map[string]*schemaProp `json:"properties"`
	Required   []string               `json:"required,omitempty"`
}

type commandSchema struct {
	Input  *schemaObj `json:"input"`
	Output *schemaObj `json:"output"`
}

var resultOutputSchema = &schemaObj{
	Type: "object",
	Properties: map[string]*schemaProp{
		"step":        {Type: "string"},
		"spec_path":   {Type: "string"},
		"spec_name":   {Type: "string"},
		"instruction": {Type: "string"},
	},
}

var statusOutputSchema = &schemaObj{
	Type: "object",
	Properties: map[string]*schemaProp{
		"spec_name":       {Type: "string"},
		"spec_path":       {Type: "string"},
		"current_step":    {Type: "string"},
		"completed_steps": {Type: "array", Items: &schemaProp{Type: "string"}},
		"total_steps":     {Type: "integer"},
		"progress":        {Type: "string"},
		"steps":           {Type: "array"},
	},
}

var specCmd = &cobra.Command{
	Use:   "spec",
	Short: "Manage spec workflow",
}

var specNewCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new spec workflow",
	RunE:  runSpecNew,
}

var specGotoCmd = &cobra.Command{
	Use:   "goto",
	Short: "Jump to a named step",
	RunE:  runSpecGoto,
}

var specStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current workflow progress",
	RunE:  runSpecStatus,
}

var specStepsCmd = &cobra.Command{
	Use:   "steps",
	Short: "List available workflow step names",
	RunE:  runSpecSteps,
}

func stateFilePath(dataDir string) string {
	return filepath.Join(dataDir, "state.json")
}

type workflowDataBuffer map[string]any

func (b workflowDataBuffer) SetData(key string, value any) {
	b[key] = value
}

// readInputIntoWorkflow reads content from either --stdin or --file and stores
// it in the workflow data. --stdin <key> reads from standard input and stores
// under <key>. --file <path> reads the file at <path> (relative paths resolve
// against the process cwd) and stores under the filename's basename without
// extension. Only one of the two flags may be set at a time.
func readInputIntoWorkflow(cmd *cobra.Command, wf interface{ SetData(string, any) }) error {
	stdinKey, _ := cmd.Flags().GetString("stdin")
	filePath, _ := cmd.Flags().GetString("file")

	if stdinKey != "" && filePath != "" {
		return fmt.Errorf("--stdin and --file are mutually exclusive")
	}

	if stdinKey != "" {
		content, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
		wf.SetData(stdinKey, string(content))
		return nil
	}

	if filePath != "" {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("reading file %s: %w", filePath, err)
		}
		base := filepath.Base(filePath)
		key := strings.TrimSuffix(base, filepath.Ext(base))
		if key == "" {
			return fmt.Errorf("--file path %q has no filename", filePath)
		}
		wf.SetData(key, string(content))
		return nil
	}

	return nil
}

func runSpecNew(cmd *cobra.Command, _ []string) error {
	if schema, _ := cmd.Flags().GetBool("schema"); schema {
		s := commandSchema{
			Input: &schemaObj{
				Type: "object",
				Properties: map[string]*schemaProp{
					"name": {Type: "string", Pattern: identifierInputPattern, MaxLen: spec.MaxIdentifierPartLength},
					"id":   {Type: "string", Pattern: identifierInputPattern, MaxLen: spec.MaxIdentifierPartLength},
				},
				Required: []string{"name"},
			},
			Output: resultOutputSchema,
		}
		return output.Write(cmd.OutOrStdout(), s, "")
	}

	dataStr, _ := cmd.Flags().GetString("data")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if dataStr == "" {
		return fmt.Errorf("--data is required (e.g. --data '{\"name\":\"my-feature\"}')")
	}
	var input struct {
		Name   string                  `json:"name"`
		ID     string                  `json:"id"`
		Remote artifact.RemoteMetadata `json:"remote"`
	}
	if err := json.Unmarshal([]byte(dataStr), &input); err != nil {
		return fmt.Errorf("parsing --data: %w", err)
	}

	dataDir, err := dataDir()
	if err != nil {
		return err
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if cfg.Artifacts.Backend == config.ArtifactBackendNotion {
		if err := artifact.ValidateRemoteMetadata(input.Remote); err != nil {
			return err
		}
		if input.ID == "" {
			input.ID = input.Remote.ExternalID
		}
	}

	st := store.NewFileStore(dataDir)
	extraData := workflowDataBuffer{}
	if err := readInputIntoWorkflow(cmd, extraData); err != nil {
		return err
	}

	resolved, err := spec.ResolveIdentifier(spec.IdentifierRequest{
		Name:    input.Name,
		ID:      input.ID,
		Method:  cfg.Spec.IDMethod,
		Counter: cfg.Spec.Counter,
		Store:   st,
		Now:     specIdentifierNow,
	})
	if err != nil {
		return err
	}

	if !dryRun && resolved.Counter != cfg.Spec.Counter {
		cfg.Spec.Counter = resolved.Counter
		cfgPath, err := configFilePath()
		if err != nil {
			return err
		}
		if err := cfg.ToYAMLFile(cfgPath); err != nil {
			return fmt.Errorf("writing config: %w", err)
		}
	}

	statePath := stateFilePath(dataDir)
	if dryRun {
		statePath += ".dryrun-tmp"
	} else {
		_ = os.Remove(statePath)
	}

	wfCfg := workflow.Config{Command: cfg.Command, DryRun: dryRun, Project: cfg}
	steps := spec.Steps()
	out := output.New(cmd.OutOrStdout(), globalFields)
	wf := workflow.New(steps, statePath, wfCfg, st, out)
	for k, v := range extraData {
		if k != "name" {
			wf.SetData(k, v)
		}
	}
	wf.SetData("name", resolved.Name)
	if cfg.Artifacts.Backend == config.ArtifactBackendNotion {
		wf.SetData("remote", input.Remote)
	}

	if err := wf.Next(); err != nil {
		return output.WriteError(cmd.ErrOrStderr(), err)
	}
	return nil
}

func runSpecGoto(cmd *cobra.Command, _ []string) error {
	if schema, _ := cmd.Flags().GetBool("schema"); schema {
		s := commandSchema{
			Input: &schemaObj{
				Type: "object",
				Properties: map[string]*schemaProp{
					"step": {Type: "string", Enum: workflow.New(spec.Steps(), "", workflow.Config{}, nil, nil).StepNames()},
				},
				Required: []string{"step"},
			},
			Output: resultOutputSchema,
		}
		return output.Write(cmd.OutOrStdout(), s, "")
	}

	dataStr, _ := cmd.Flags().GetString("data")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if dataStr == "" {
		return fmt.Errorf("--data is required (e.g. --data '{\"step\":\"requirements\"}')")
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
	steps := spec.Steps()
	out := output.New(cmd.OutOrStdout(), globalFields)
	wf := workflow.New(steps, stateFilePath(dataDir), wfCfg, store.NewFileStore(dataDir), out)

	for k, v := range input {
		if k != "step" {
			wf.SetData(k, v)
		}
	}

	if err := readInputIntoWorkflow(cmd, wf); err != nil {
		return err
	}

	if err := wf.Goto(stepVal); err != nil {
		return output.WriteError(cmd.ErrOrStderr(), err)
	}
	return nil
}

func runSpecStatus(cmd *cobra.Command, _ []string) error {
	if schema, _ := cmd.Flags().GetBool("schema"); schema {
		s := commandSchema{Input: nil, Output: statusOutputSchema}
		return output.Write(cmd.OutOrStdout(), s, "")
	}

	dataDir, err := dataDir()
	if err != nil {
		return err
	}

	steps := spec.Steps()
	wf := workflow.New(steps, stateFilePath(dataDir), workflow.Config{}, nil, nil)
	st := wf.State()

	specName, _ := wf.GetData("name")
	specPath := filepath.Join(dataDir, spec.SpecFilePath(fmt.Sprintf("%v", specName)))

	stepInfos := wf.StepStatus()
	entries := make([]spec.StepEntry, len(stepInfos))
	for i, info := range stepInfos {
		entries[i] = spec.StepEntry{Name: info.Name, Status: info.Status}
	}

	out := output.New(cmd.OutOrStdout(), globalFields)
	return out.WriteResult(spec.StatusResult{
		SpecName:       fmt.Sprintf("%v", specName),
		SpecPath:       specPath,
		CurrentStep:    wf.Current(),
		CompletedSteps: st.CompletedSteps,
		TotalSteps:     len(steps),
		Progress:       fmt.Sprintf("%d/%d", len(st.CompletedSteps), len(steps)),
		Steps:          entries,
	})
}

func runSpecSteps(cmd *cobra.Command, _ []string) error {
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

	wf := workflow.New(spec.Steps(), "", workflow.Config{}, nil, nil)
	out := output.New(cmd.OutOrStdout(), globalFields)
	return out.WriteResult(spec.StepsResult{Steps: wf.StepNames()})
}

func init() {
	specCmd.PersistentFlags().Bool("schema", false, "Print the input/output schema for this subcommand and exit")
	specCmd.PersistentFlags().BoolP("dry-run", "n", false, "Validate and preview without writing any files or persisting state")

	specNewCmd.Flags().StringP("data", "d", "", `JSON input (e.g. '{"name":"my-feature"}')`)
	specNewCmd.Flags().String("stdin", "", "Read stdin and store it in workflow data under this key")
	specNewCmd.Flags().String("file", "", "Read a file at <path> (relative to cwd) and store its contents under the filename's basename (without extension)")
	specGotoCmd.Flags().StringP("data", "d", "", `JSON input (e.g. '{"step":"requirements"}')`)
	specGotoCmd.Flags().String("stdin", "", "Read stdin and store it in workflow data under this key")
	specGotoCmd.Flags().String("file", "", "Read a file at <path> (relative to cwd) and store its contents under the filename's basename (without extension)")

	specCmd.AddCommand(specNewCmd, specGotoCmd, specStatusCmd, specStepsCmd)
}
