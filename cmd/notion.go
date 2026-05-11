package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/notion/schema"
	"github.com/jumppad-labs/spektacular/internal/notion/setup"
	"github.com/jumppad-labs/spektacular/internal/output"
	"github.com/jumppad-labs/spektacular/internal/project"
	"github.com/spf13/cobra"
)

var notionCmd = &cobra.Command{
	Use:   "notion",
	Short: "Validate and link Notion artifact databases",
}

var notionInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Print Notion MCP instructions for creating Spektacular databases",
	RunE:  runNotionInit,
}

var notionLinkCmd = &cobra.Command{
	Use:   "link",
	Short: "Validate and link existing Notion artifact databases",
	RunE:  runNotionLink,
}

var notionDoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Validate linked Notion databases and prepare safe repairs",
	RunE:  runNotionDoctor,
}

var notionStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Notion artifact backend status",
	RunE:  runNotionStatus,
}

type notionInput struct {
	BasePageURL     string          `json:"base_page_url"`
	CacheDir        string          `json:"cache_dir"`
	SpecsDataSource string          `json:"specs_data_source"`
	PlansDataSource string          `json:"plans_data_source"`
	SpecIDProperty  string          `json:"spec_id_property"`
	PlanIDProperty  string          `json:"plan_id_property"`
	Specs           json.RawMessage `json:"specs"`
	Plans           json.RawMessage `json:"plans"`
	Apply           bool            `json:"apply"`
	ApprovedFixIDs  []string        `json:"approved_fix_ids"`
}

type notionResult struct {
	Status           string              `json:"status"`
	Configured       bool                `json:"configured,omitempty"`
	Backend          string              `json:"backend,omitempty"`
	CacheDir         string              `json:"cache_dir,omitempty"`
	BasePageURL      string              `json:"base_page_url,omitempty"`
	SpecsDataSource  string              `json:"specs_data_source,omitempty"`
	PlansDataSource  string              `json:"plans_data_source,omitempty"`
	SpecIDProperty   string              `json:"spec_id_property,omitempty"`
	PlanIDProperty   string              `json:"plan_id_property,omitempty"`
	Report           schema.Report       `json:"report,omitempty"`
	Instructions     []setup.Instruction `json:"instructions,omitempty"`
	ApprovalRequired bool                `json:"approval_required,omitempty"`
	AppliedRepairs   []schema.Repair     `json:"applied_repairs,omitempty"`
	UpdatedSpecs     *schema.DataSource  `json:"updated_specs,omitempty"`
	UpdatedPlans     *schema.DataSource  `json:"updated_plans,omitempty"`
	ReportAfter      *schema.Report      `json:"report_after,omitempty"`
	Next             string              `json:"next,omitempty"`
}

func runNotionInit(cmd *cobra.Command, _ []string) error {
	if schemaFlag, _ := cmd.Flags().GetBool("schema"); schemaFlag {
		return writeNotionSchema(cmd, false)
	}

	input, err := readNotionInput(cmd)
	if err != nil {
		return err
	}
	res := notionResult{
		Status:       "instructions",
		BasePageURL:  input.BasePageURL,
		Instructions: setup.CreateDatabaseInstructions(input.BasePageURL),
		Next:         "Use Notion MCP to create/fetch the databases, then run `spektacular notion link` with the fetched snapshots.",
	}
	return output.Write(cmd.OutOrStdout(), res, globalFields)
}

func runNotionLink(cmd *cobra.Command, _ []string) error {
	if schemaFlag, _ := cmd.Flags().GetBool("schema"); schemaFlag {
		return writeNotionSchema(cmd, true)
	}

	input, specs, plans, opts, err := readNotionValidationInput(cmd)
	if err != nil {
		return err
	}
	report := schema.ValidatePair(specs, plans, opts)
	if !report.Valid {
		res := notionResult{
			Status:          "needs_doctor",
			Configured:      false,
			BasePageURL:     input.BasePageURL,
			SpecsDataSource: opts.SpecsDataSource,
			PlansDataSource: opts.PlansDataSource,
			SpecIDProperty:  opts.SpecIDProperty,
			PlanIDProperty:  opts.PlanIDProperty,
			Report:          report,
			Instructions:    setup.RepairInstructions(report.Repairs),
			Next:            "Run `spektacular notion doctor` and apply approved safe repairs before linking.",
		}
		if len(report.BlockingIssues) > 0 {
			res.Status = "blocking_issues"
			res.Next = "Resolve blocking Notion schema issues, then run `spektacular notion link` again."
		}
		return output.Write(cmd.OutOrStdout(), res, globalFields)
	}

	cfg, err := notionConfigFromInput(input, opts)
	if err != nil {
		return err
	}
	if err := ensureNotionProject(cfg); err != nil {
		return err
	}
	cfgPath, err := configFilePath()
	if err != nil {
		return err
	}
	if err := cfg.ToYAMLFile(cfgPath); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	res := notionResult{
		Status:          "linked",
		Configured:      true,
		Backend:         cfg.Artifacts.Backend,
		CacheDir:        cfg.Artifacts.CacheDir,
		BasePageURL:     cfg.Artifacts.Notion.BasePageURL,
		SpecsDataSource: cfg.Artifacts.Notion.SpecsDataSource,
		PlansDataSource: cfg.Artifacts.Notion.PlansDataSource,
		SpecIDProperty:  cfg.Artifacts.Notion.SpecIDProperty,
		PlanIDProperty:  cfg.Artifacts.Notion.PlanIDProperty,
		Report:          report,
		Next:            "Pull or create Notion-backed artifacts before starting workflow work.",
	}
	return output.Write(cmd.OutOrStdout(), res, globalFields)
}

func runNotionDoctor(cmd *cobra.Command, _ []string) error {
	if schemaFlag, _ := cmd.Flags().GetBool("schema"); schemaFlag {
		return writeNotionSchema(cmd, true)
	}

	input, specs, plans, opts, err := readNotionValidationInput(cmd)
	if err != nil {
		return err
	}
	applyFlag, _ := cmd.Flags().GetBool("apply")
	approveAll, _ := cmd.Flags().GetBool("approve-additive")
	approvedFlag, _ := cmd.Flags().GetString("approve")
	input.Apply = input.Apply || applyFlag
	input.ApprovedFixIDs = append(input.ApprovedFixIDs, splitCSV(approvedFlag)...)

	report := schema.ValidatePair(specs, plans, opts)
	res := notionResult{
		Status:          "checked",
		BasePageURL:     input.BasePageURL,
		SpecsDataSource: opts.SpecsDataSource,
		PlansDataSource: opts.PlansDataSource,
		SpecIDProperty:  opts.SpecIDProperty,
		PlanIDProperty:  opts.PlanIDProperty,
		Report:          report,
		Instructions:    setup.RepairInstructions(report.Repairs),
	}
	if report.Valid {
		res.Status = "valid"
		return output.Write(cmd.OutOrStdout(), res, globalFields)
	}
	if !input.Apply {
		res.Next = "Review fixable repairs, resolve blocking issues, then rerun with --apply and explicit approval."
		return output.Write(cmd.OutOrStdout(), res, globalFields)
	}
	if len(report.BlockingIssues) > 0 {
		res.Status = "blocking_issues"
		res.Next = "Resolve blocking Notion schema issues before applying safe additive repairs."
		return output.Write(cmd.OutOrStdout(), res, globalFields)
	}

	approvedIDs := input.ApprovedFixIDs
	if approveAll {
		approvedIDs = make([]string, 0, len(report.Repairs))
		for _, repair := range report.Repairs {
			approvedIDs = append(approvedIDs, repair.ID)
		}
	}
	if len(approvedIDs) == 0 {
		res.Status = "approval_required"
		res.ApprovalRequired = true
		res.Next = "Rerun with --approve-additive or --approve <repair-id> after user/agent approval."
		return output.Write(cmd.OutOrStdout(), res, globalFields)
	}

	updatedSpecs, updatedPlans, applied := schema.ApplyApprovedRepairs(specs, plans, report.Repairs, approvedIDs)
	after := schema.ValidatePair(updatedSpecs, updatedPlans, opts)
	instructions := setup.RepairInstructions(applied)
	res.Status = "repairs_prepared"
	res.Instructions = instructions
	res.AppliedRepairs = applied
	res.UpdatedSpecs = &updatedSpecs
	res.UpdatedPlans = &updatedPlans
	res.ReportAfter = &after
	res.Next = "Apply the prepared instructions through Notion MCP, fetch fresh snapshots, then rerun `spektacular notion link`."
	if after.Valid {
		res.Next = "Apply the prepared instructions through Notion MCP, then rerun `spektacular notion link` with fresh snapshots."
	}
	return output.Write(cmd.OutOrStdout(), res, globalFields)
}

func runNotionStatus(cmd *cobra.Command, _ []string) error {
	if schemaFlag, _ := cmd.Flags().GetBool("schema"); schemaFlag {
		return output.Write(cmd.OutOrStdout(), commandSchema{
			Input: nil,
			Output: &schemaObj{Type: "object", Properties: map[string]*schemaProp{
				"status":            {Type: "string"},
				"configured":        {Type: "boolean"},
				"backend":           {Type: "string"},
				"cache_dir":         {Type: "string"},
				"specs_data_source": {Type: "string"},
				"plans_data_source": {Type: "string"},
			}},
		}, "")
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	configured := cfg.Artifacts.Backend == config.ArtifactBackendNotion &&
		cfg.Artifacts.Notion.SpecsDataSource != "" &&
		cfg.Artifacts.Notion.PlansDataSource != ""
	status := "local"
	if cfg.Artifacts.Backend == config.ArtifactBackendNotion {
		status = "configured"
		if !configured {
			status = "incomplete"
		}
	}
	return output.Write(cmd.OutOrStdout(), notionResult{
		Status:          status,
		Configured:      configured,
		Backend:         cfg.Artifacts.Backend,
		CacheDir:        cfg.Artifacts.CacheDir,
		BasePageURL:     cfg.Artifacts.Notion.BasePageURL,
		SpecsDataSource: cfg.Artifacts.Notion.SpecsDataSource,
		PlansDataSource: cfg.Artifacts.Notion.PlansDataSource,
		SpecIDProperty:  cfg.Artifacts.Notion.SpecIDProperty,
		PlanIDProperty:  cfg.Artifacts.Notion.PlanIDProperty,
	}, globalFields)
}

func writeNotionSchema(cmd *cobra.Command, requiresSnapshots bool) error {
	props := map[string]*schemaProp{
		"base_page_url":     {Type: "string"},
		"cache_dir":         {Type: "string"},
		"specs_data_source": {Type: "string"},
		"plans_data_source": {Type: "string"},
		"spec_id_property":  {Type: "string"},
		"plan_id_property":  {Type: "string"},
		"specs":             {Type: "object"},
		"plans":             {Type: "object"},
	}
	required := []string(nil)
	if requiresSnapshots {
		required = []string{"specs", "plans"}
	}
	return output.Write(cmd.OutOrStdout(), commandSchema{
		Input: &schemaObj{Type: "object", Properties: props, Required: required},
		Output: &schemaObj{Type: "object", Properties: map[string]*schemaProp{
			"status":       {Type: "string"},
			"configured":   {Type: "boolean"},
			"report":       {Type: "object"},
			"instructions": {Type: "array"},
		}},
	}, "")
}

func readNotionValidationInput(cmd *cobra.Command) (notionInput, schema.DataSource, schema.DataSource, schema.ValidationOptions, error) {
	input, err := readNotionInput(cmd)
	if err != nil {
		return notionInput{}, schema.DataSource{}, schema.DataSource{}, schema.ValidationOptions{}, err
	}

	specsRaw, err := snapshotRaw(cmd, input.Specs, "specs-file")
	if err != nil {
		return notionInput{}, schema.DataSource{}, schema.DataSource{}, schema.ValidationOptions{}, err
	}
	plansRaw, err := snapshotRaw(cmd, input.Plans, "plans-file")
	if err != nil {
		return notionInput{}, schema.DataSource{}, schema.DataSource{}, schema.ValidationOptions{}, err
	}

	specs, err := schema.ParseDataSource(specsRaw)
	if err != nil {
		return notionInput{}, schema.DataSource{}, schema.DataSource{}, schema.ValidationOptions{}, fmt.Errorf("parsing specs snapshot: %w", err)
	}
	plans, err := schema.ParseDataSource(plansRaw)
	if err != nil {
		return notionInput{}, schema.DataSource{}, schema.DataSource{}, schema.ValidationOptions{}, fmt.Errorf("parsing plans snapshot: %w", err)
	}

	opts := schema.ValidationOptions{
		SpecsDataSource: input.SpecsDataSource,
		PlansDataSource: input.PlansDataSource,
		SpecIDProperty:  input.SpecIDProperty,
		PlanIDProperty:  input.PlanIDProperty,
	}
	if opts.SpecsDataSource == "" {
		opts.SpecsDataSource = specs.URL
	}
	if opts.PlansDataSource == "" {
		opts.PlansDataSource = plans.URL
	}
	if opts.SpecIDProperty == "" {
		opts.SpecIDProperty = config.DefaultSpecIDPropertyName
	}
	if opts.PlanIDProperty == "" {
		opts.PlanIDProperty = config.DefaultPlanIDPropertyName
	}
	return input, specs, plans, opts, nil
}

func readNotionInput(cmd *cobra.Command) (notionInput, error) {
	dataStr, _ := cmd.Flags().GetString("data")
	if dataStr == "" {
		return notionInput{}, nil
	}
	var input notionInput
	if err := json.Unmarshal([]byte(dataStr), &input); err != nil {
		return notionInput{}, fmt.Errorf("parsing --data: %w", err)
	}
	return input, nil
}

func snapshotRaw(cmd *cobra.Command, inline json.RawMessage, fileFlag string) ([]byte, error) {
	filePath, _ := cmd.Flags().GetString(fileFlag)
	if len(inline) > 0 && filePath != "" {
		return nil, fmt.Errorf("%s and inline snapshot data are mutually exclusive", fileFlag)
	}
	if len(inline) > 0 {
		return inline, nil
	}
	if filePath == "" {
		return nil, fmt.Errorf("snapshot is required via --data or --%s", fileFlag)
	}
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", fileFlag, err)
	}
	return raw, nil
}

func notionConfigFromInput(input notionInput, opts schema.ValidationOptions) (config.Config, error) {
	cfg, err := loadConfig()
	if err != nil {
		if !os.IsNotExist(err) {
			return config.Config{}, err
		}
		cfg = config.NewDefault()
	}
	if input.CacheDir == "" {
		input.CacheDir = config.DefaultNotionCacheDir
	}
	cfg.Spec.IDMethod = config.SpecIDMethodExternal
	cfg.Artifacts.Backend = config.ArtifactBackendNotion
	cfg.Artifacts.CacheDir = input.CacheDir
	cfg.Artifacts.Notion.BasePageURL = input.BasePageURL
	cfg.Artifacts.Notion.SpecsDataSource = opts.SpecsDataSource
	cfg.Artifacts.Notion.PlansDataSource = opts.PlansDataSource
	cfg.Artifacts.Notion.SpecIDProperty = opts.SpecIDProperty
	cfg.Artifacts.Notion.PlanIDProperty = opts.PlanIDProperty
	if err := cfg.Validate(); err != nil {
		return config.Config{}, err
	}
	return cfg, nil
}

func ensureNotionProject(cfg config.Config) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	if err := project.InitWithOptions(cwd, project.InitOptions{
		Force:                   true,
		Config:                  cfg,
		CreateLocalArtifactDirs: false,
		CreateNotionCacheDir:    true,
	}); err != nil {
		return fmt.Errorf("initialising Notion project cache: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(cwd, ".spektacular", cfg.Artifacts.CacheDir), 0o755); err != nil {
		return fmt.Errorf("creating Notion cache: %w", err)
	}
	return nil
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func init() {
	notionCmd.PersistentFlags().Bool("schema", false, "Print the input/output schema for this subcommand and exit")
	notionCmd.PersistentFlags().StringP("data", "d", "", "JSON input")

	notionLinkCmd.Flags().String("specs-file", "", "Read the Notion MCP Specs data source snapshot from a JSON file")
	notionLinkCmd.Flags().String("plans-file", "", "Read the Notion MCP Plans data source snapshot from a JSON file")
	notionDoctorCmd.Flags().String("specs-file", "", "Read the Notion MCP Specs data source snapshot from a JSON file")
	notionDoctorCmd.Flags().String("plans-file", "", "Read the Notion MCP Plans data source snapshot from a JSON file")
	notionDoctorCmd.Flags().Bool("apply", false, "Prepare approved safe additive repairs")
	notionDoctorCmd.Flags().Bool("approve-additive", false, "Approve all safe additive repairs")
	notionDoctorCmd.Flags().String("approve", "", "Comma-separated repair IDs to approve")

	notionCmd.AddCommand(notionInitCmd, notionLinkCmd, notionDoctorCmd, notionStatusCmd)
}
