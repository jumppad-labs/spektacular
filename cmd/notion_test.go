package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jumppad-labs/spektacular/internal/config"
	notionschema "github.com/jumppad-labs/spektacular/internal/notion/schema"
	"github.com/stretchr/testify/require"
)

func resetNotionCommandFlags(t *testing.T) {
	t.Helper()
	reset := func() {
		require.NoError(t, notionCmd.PersistentFlags().Set("schema", "false"))
		require.NoError(t, notionCmd.PersistentFlags().Set("data", ""))
		require.NoError(t, notionLinkCmd.Flags().Set("specs-file", ""))
		require.NoError(t, notionLinkCmd.Flags().Set("plans-file", ""))
		require.NoError(t, notionDoctorCmd.Flags().Set("specs-file", ""))
		require.NoError(t, notionDoctorCmd.Flags().Set("plans-file", ""))
		require.NoError(t, notionDoctorCmd.Flags().Set("apply", "false"))
		require.NoError(t, notionDoctorCmd.Flags().Set("approve-additive", "false"))
		require.NoError(t, notionDoctorCmd.Flags().Set("approve", ""))
	}
	reset()
	t.Cleanup(reset)
}

func notionSpecsSnapshotForTest() notionschema.DataSource {
	return notionschema.DataSource{
		Name: "Specs",
		URL:  "collection://specs",
		Schema: map[string]notionschema.Property{
			"Name":        {Name: "Name", Type: notionschema.TypeTitle},
			"Spec ID":     {Name: "Spec ID", Type: notionschema.TypeAutoIncrementID},
			"Created":     {Name: "Created", Type: notionschema.TypeCreatedTime},
			"Last Edited": {Name: "Last Edited", Type: notionschema.TypeLastEditedTime},
			"Status":      {Name: "Status", Type: notionschema.TypeSelect},
			"Approval":    {Name: "Approval", Type: notionschema.TypeSelect},
			"Tags":        {Name: "Tags", Type: notionschema.TypeMultiSelect},
			"Author":      {Name: "Author", Type: notionschema.TypePerson},
			"Reviewers":   {Name: "Reviewers", Type: notionschema.TypePerson},
			"Plan":        {Name: "Plan", Type: notionschema.TypeRelation, DataSourceURL: "collection://plans"},
		},
	}
}

func notionPlansSnapshotForTest() notionschema.DataSource {
	return notionschema.DataSource{
		Name: "Plans",
		URL:  "collection://plans",
		Schema: map[string]notionschema.Property{
			"Name":          {Name: "Name", Type: notionschema.TypeTitle},
			"Plan ID":       {Name: "Plan ID", Type: notionschema.TypeAutoIncrementID},
			"Created":       {Name: "Created", Type: notionschema.TypeCreatedTime},
			"Last Edited":   {Name: "Last Edited", Type: notionschema.TypeLastEditedTime},
			"Status":        {Name: "Status", Type: notionschema.TypeSelect},
			"Approval":      {Name: "Approval", Type: notionschema.TypeSelect},
			"Tags":          {Name: "Tags", Type: notionschema.TypeMultiSelect},
			"Author":        {Name: "Author", Type: notionschema.TypePerson},
			"Reviewers":     {Name: "Reviewers", Type: notionschema.TypePerson},
			"Spec":          {Name: "Spec", Type: notionschema.TypeRelation, DataSourceURL: "collection://specs"},
			"Linear Issues": {Name: "Linear Issues", Type: notionschema.TypeURL},
		},
	}
}

func notionDataForTest(t *testing.T, specs, plans notionschema.DataSource) string {
	t.Helper()
	data, err := json.Marshal(map[string]any{
		"base_page_url":     "https://notion.example/base",
		"specs_data_source": "collection://specs",
		"plans_data_source": "collection://plans",
		"spec_id_property":  "Spec ID",
		"plan_id_property":  "Plan ID",
		"specs":             specs,
		"plans":             plans,
	})
	require.NoError(t, err)
	return string(data)
}

func runNotionForTest(t *testing.T, args ...string) (map[string]any, error) {
	t.Helper()
	resetNotionCommandFlags(t)
	stdout, _ := setupImplementCmd(t)
	rootCmd.SetArgs(append([]string{"notion"}, args...))

	err := rootCmd.Execute()
	if err != nil {
		return nil, err
	}
	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	return result, nil
}

func TestNotionLink_CompatibleSnapshotsWriteConfig(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	result, err := runNotionForTest(t, "link", "--data", notionDataForTest(t, notionSpecsSnapshotForTest(), notionPlansSnapshotForTest()))
	require.NoError(t, err)

	require.Equal(t, "linked", result["status"])
	require.Equal(t, true, result["configured"])
	require.DirExists(t, filepath.Join(dir, ".spektacular", "cache", "notion"))
	require.NoDirExists(t, filepath.Join(dir, ".spektacular", "specs"))
	require.NoDirExists(t, filepath.Join(dir, ".spektacular", "plans"))

	cfg, err := config.FromYAMLFile(filepath.Join(dir, ".spektacular", "config.yaml"))
	require.NoError(t, err)
	require.Equal(t, config.ArtifactBackendNotion, cfg.Artifacts.Backend)
	require.Equal(t, config.SpecIDMethodExternal, cfg.Spec.IDMethod)
	require.Equal(t, "collection://specs", cfg.Artifacts.Notion.SpecsDataSource)
	require.Equal(t, "collection://plans", cfg.Artifacts.Notion.PlansDataSource)
}

func TestNotionLink_BlockingIssuesLeaveProjectUnconfigured(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	specs := notionSpecsSnapshotForTest()
	delete(specs.Schema, "Spec ID")

	result, err := runNotionForTest(t, "link", "--data", notionDataForTest(t, specs, notionPlansSnapshotForTest()))
	require.NoError(t, err)

	require.Equal(t, "blocking_issues", result["status"])
	report := result["report"].(map[string]any)
	require.Len(t, report["blocking_issues"], 1)
	require.NoFileExists(t, filepath.Join(dir, ".spektacular", "config.yaml"))
}

func TestNotionDoctor_ReportsFixableAndBlockingIssues(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	specs := notionSpecsSnapshotForTest()
	plans := notionPlansSnapshotForTest()
	delete(specs.Schema, "Tags")
	plans.Schema["Status"] = notionschema.Property{Name: "Status", Type: notionschema.TypeURL}

	result, err := runNotionForTest(t, "doctor", "--data", notionDataForTest(t, specs, plans))
	require.NoError(t, err)

	require.Equal(t, "checked", result["status"])
	report := result["report"].(map[string]any)
	require.Len(t, report["fixable_issues"], 1)
	require.Len(t, report["blocking_issues"], 1)
	require.Len(t, result["instructions"], 1)
}

func TestNotionDoctorApplyRequiresAndUsesApproval(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	specs := notionSpecsSnapshotForTest()
	delete(specs.Schema, "Tags")
	data := notionDataForTest(t, specs, notionPlansSnapshotForTest())

	needsApproval, err := runNotionForTest(t, "doctor", "--apply", "--data", data)
	require.NoError(t, err)
	require.Equal(t, "approval_required", needsApproval["status"])
	require.Equal(t, true, needsApproval["approval_required"])

	applied, err := runNotionForTest(t, "doctor", "--apply", "--approve-additive", "--data", data)
	require.NoError(t, err)
	require.Equal(t, "repairs_prepared", applied["status"])
	require.Len(t, applied["applied_repairs"], 1)
	reportAfter := applied["report_after"].(map[string]any)
	require.Equal(t, true, reportAfter["valid"])
}

func TestNotionLink_AcceptsSnapshotFiles(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	specsPath := filepath.Join(dir, "specs.json")
	plansPath := filepath.Join(dir, "plans.json")
	specsRaw, err := json.Marshal(notionSpecsSnapshotForTest())
	require.NoError(t, err)
	plansRaw, err := json.Marshal(notionPlansSnapshotForTest())
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(specsPath, specsRaw, 0o644))
	require.NoError(t, os.WriteFile(plansPath, plansRaw, 0o644))

	result, err := runNotionForTest(t, "link", "--specs-file", specsPath, "--plans-file", plansPath, "--data", `{"base_page_url":"https://notion.example/base"}`)
	require.NoError(t, err)
	require.Equal(t, "linked", result["status"])
}
