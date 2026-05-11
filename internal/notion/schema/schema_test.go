package schema

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func compatibleSpecsSnapshot() DataSource {
	return DataSource{
		Name: "Specs",
		URL:  "collection://specs",
		Schema: map[string]Property{
			"Name":        {Name: "Name", Type: TypeTitle},
			"Spec ID":     {Name: "Spec ID", Type: TypeAutoIncrementID},
			"Created":     {Name: "Created", Type: TypeCreatedTime},
			"Last Edited": {Name: "Last Edited", Type: TypeLastEditedTime},
			"Status":      {Name: "Status", Type: TypeSelect},
			"Approval":    {Name: "Approval", Type: TypeSelect},
			"Tags":        {Name: "Tags", Type: TypeMultiSelect},
			"Author":      {Name: "Author", Type: TypePerson},
			"Reviewers":   {Name: "Reviewers", Type: TypePerson},
			"Plan":        {Name: "Plan", Type: TypeRelation, DataSourceURL: "collection://plans"},
		},
	}
}

func compatiblePlansSnapshot() DataSource {
	return DataSource{
		Name: "Plans",
		URL:  "collection://plans",
		Schema: map[string]Property{
			"Name":          {Name: "Name", Type: TypeTitle},
			"Plan ID":       {Name: "Plan ID", Type: TypeAutoIncrementID},
			"Created":       {Name: "Created", Type: TypeCreatedTime},
			"Last Edited":   {Name: "Last Edited", Type: TypeLastEditedTime},
			"Status":        {Name: "Status", Type: TypeSelect},
			"Approval":      {Name: "Approval", Type: TypeSelect},
			"Tags":          {Name: "Tags", Type: TypeMultiSelect},
			"Author":        {Name: "Author", Type: TypePerson},
			"Reviewers":     {Name: "Reviewers", Type: TypePerson},
			"Spec":          {Name: "Spec", Type: TypeRelation, DataSourceURL: "collection://specs"},
			"Linear Issues": {Name: "Linear Issues", Type: TypeURL},
		},
	}
}

func validationOptions() ValidationOptions {
	return ValidationOptions{
		SpecsDataSource: "collection://specs",
		PlansDataSource: "collection://plans",
		SpecIDProperty:  "Spec ID",
		PlanIDProperty:  "Plan ID",
	}
}

func TestValidatePair_CompatibleSnapshotsPass(t *testing.T) {
	report := ValidatePair(compatibleSpecsSnapshot(), compatiblePlansSnapshot(), validationOptions())

	require.True(t, report.Valid)
	require.Empty(t, report.BlockingIssues)
	require.Empty(t, report.FixableIssues)
	require.Empty(t, report.Repairs)
}

func TestValidatePair_MissingRequiredIDFieldsAreBlocking(t *testing.T) {
	specs := compatibleSpecsSnapshot()
	delete(specs.Schema, "Spec ID")

	report := ValidatePair(specs, compatiblePlansSnapshot(), validationOptions())

	require.False(t, report.Valid)
	require.Len(t, report.BlockingIssues, 1)
	require.Equal(t, "specs.spec-id.missing", report.BlockingIssues[0].ID)
	require.Equal(t, TypeAutoIncrementID, report.BlockingIssues[0].ExpectedType)
	require.Empty(t, report.Repairs)
}

func TestValidatePair_MissingSafePropertiesAreFixable(t *testing.T) {
	specs := compatibleSpecsSnapshot()
	plans := compatiblePlansSnapshot()
	delete(specs.Schema, "Tags")
	delete(plans.Schema, "Linear Issues")

	report := ValidatePair(specs, plans, validationOptions())

	require.False(t, report.Valid)
	require.Empty(t, report.BlockingIssues)
	require.Len(t, report.FixableIssues, 2)
	require.Len(t, report.Repairs, 2)
	require.Equal(t, "plans.linear-issues.add", report.Repairs[0].ID)
	require.Equal(t, "specs.tags.add", report.Repairs[1].ID)

	updatedSpecs, updatedPlans, applied := ApplyApprovedRepairs(specs, plans, report.Repairs, []string{"specs.tags.add", "plans.linear-issues.add"})
	require.Len(t, applied, 2)
	after := ValidatePair(updatedSpecs, updatedPlans, validationOptions())
	require.True(t, after.Valid)
}

func TestValidatePair_IncompatiblePropertyTypesAreBlocking(t *testing.T) {
	specs := compatibleSpecsSnapshot()
	specs.Schema["Status"] = Property{Name: "Status", Type: TypeURL}

	report := ValidatePair(specs, compatiblePlansSnapshot(), validationOptions())

	require.False(t, report.Valid)
	require.Len(t, report.BlockingIssues, 1)
	require.Equal(t, "specs.status.type", report.BlockingIssues[0].ID)
	require.Empty(t, report.Repairs)
}

func TestParseDataSource_ReadsNotionMCPFetchWrapper(t *testing.T) {
	direct := compatibleSpecsSnapshot()
	directJSON, err := json.Marshal(direct)
	require.NoError(t, err)
	mcpState := `<data-source-state>` + string(directJSON) + `</data-source-state>`
	inner, err := json.Marshal(map[string]any{
		"metadata": map[string]any{"type": "data_source"},
		"text":     mcpState,
	})
	require.NoError(t, err)
	wrapper, err := json.Marshal([]map[string]string{{"type": "text", "text": string(inner)}})
	require.NoError(t, err)

	parsed, err := ParseDataSource(wrapper)
	require.NoError(t, err)
	require.Equal(t, direct.URL, parsed.URL)
	require.Equal(t, TypeAutoIncrementID, parsed.Schema["Spec ID"].Type)
}
