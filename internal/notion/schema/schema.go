package schema

import (
	"fmt"
	"sort"
)

const (
	SeverityBlocking = "blocking"
	SeverityFixable  = "fixable"

	DataSourceSpecs = "specs"
	DataSourcePlans = "plans"

	TypeAutoIncrementID = "auto_increment_id"
	TypeCreatedTime     = "created_time"
	TypeLastEditedTime  = "last_edited_time"
	TypeMultiSelect     = "multi_select"
	TypePerson          = "person"
	TypeRelation        = "relation"
	TypeSelect          = "select"
	TypeTitle           = "title"
	TypeURL             = "url"
)

// DataSource is the small schema snapshot Spektacular needs from Notion MCP.
type DataSource struct {
	Name   string              `json:"name,omitempty"`
	URL    string              `json:"url"`
	Schema map[string]Property `json:"schema"`
}

// Property describes a Notion data source property in MCP fetch output.
type Property struct {
	Name          string   `json:"name"`
	Type          string   `json:"type"`
	DataSourceURL string   `json:"dataSourceUrl,omitempty"`
	Options       []Option `json:"options,omitempty"`
}

// Option describes a select or multi-select option.
type Option struct {
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

// ValidationOptions controls linked database validation.
type ValidationOptions struct {
	SpecsDataSource string
	PlansDataSource string
	SpecIDProperty  string
	PlanIDProperty  string
}

// Issue is a validation finding for a linked Notion data source.
type Issue struct {
	ID              string `json:"id"`
	Severity        string `json:"severity"`
	DataSource      string `json:"data_source"`
	Property        string `json:"property,omitempty"`
	ExpectedType    string `json:"expected_type,omitempty"`
	ActualType      string `json:"actual_type,omitempty"`
	Message         string `json:"message"`
	Repairable      bool   `json:"repairable"`
	RepairStatement string `json:"repair_statement,omitempty"`
}

// Repair is a safe additive schema change the agent can apply through Notion MCP.
type Repair struct {
	ID               string `json:"id"`
	DataSource       string `json:"data_source"`
	DataSourceURL    string `json:"data_source_url"`
	Property         string `json:"property"`
	Type             string `json:"type"`
	Statement        string `json:"statement"`
	RequiresApproval bool   `json:"requires_approval"`
}

// Report is the full schema validation result.
type Report struct {
	Valid          bool     `json:"valid"`
	BlockingIssues []Issue  `json:"blocking_issues"`
	FixableIssues  []Issue  `json:"fixable_issues"`
	Repairs        []Repair `json:"repairs"`
}

type requirement struct {
	dataSource string
	property   string
	typ        string
	target     string
	blocking   bool
	statement  string
}

// ValidatePair validates Specs and Plans data source snapshots together.
func ValidatePair(specs, plans DataSource, opts ValidationOptions) Report {
	opts = applyDefaults(opts)

	reqs := []requirement{
		{dataSource: DataSourceSpecs, property: "Name", typ: TypeTitle, blocking: true},
		{dataSource: DataSourceSpecs, property: opts.SpecIDProperty, typ: TypeAutoIncrementID, blocking: true},
		{dataSource: DataSourceSpecs, property: "Created", typ: TypeCreatedTime, statement: `ADD COLUMN "Created" CREATED_TIME`},
		{dataSource: DataSourceSpecs, property: "Last Edited", typ: TypeLastEditedTime, statement: `ADD COLUMN "Last Edited" LAST_EDITED_TIME`},
		{dataSource: DataSourceSpecs, property: "Status", typ: TypeSelect, statement: `ADD COLUMN "Status" SELECT('Drafting':gray, 'In Review':yellow, 'Approved':green, 'Building':blue, 'Shipped':purple, 'Superseded':red)`},
		{dataSource: DataSourceSpecs, property: "Approval", typ: TypeSelect, statement: `ADD COLUMN "Approval" SELECT('Draft':gray, 'Approved':green)`},
		{dataSource: DataSourceSpecs, property: "Tags", typ: TypeMultiSelect, statement: `ADD COLUMN "Tags" MULTI_SELECT('security':red, 'compliance':orange, 'platform':blue, 'infra':purple, 'growth':green, 'developer-experience':yellow, 'ai':pink)`},
		{dataSource: DataSourceSpecs, property: "Author", typ: TypePerson, statement: `ADD COLUMN "Author" PEOPLE`},
		{dataSource: DataSourceSpecs, property: "Reviewers", typ: TypePerson, statement: `ADD COLUMN "Reviewers" PEOPLE`},
		{dataSource: DataSourceSpecs, property: "Plan", typ: TypeRelation, target: opts.PlansDataSource, statement: fmt.Sprintf(`ADD COLUMN "Plan" RELATION('%s')`, opts.PlansDataSource)},

		{dataSource: DataSourcePlans, property: "Name", typ: TypeTitle, blocking: true},
		{dataSource: DataSourcePlans, property: opts.PlanIDProperty, typ: TypeAutoIncrementID, blocking: true},
		{dataSource: DataSourcePlans, property: "Created", typ: TypeCreatedTime, statement: `ADD COLUMN "Created" CREATED_TIME`},
		{dataSource: DataSourcePlans, property: "Last Edited", typ: TypeLastEditedTime, statement: `ADD COLUMN "Last Edited" LAST_EDITED_TIME`},
		{dataSource: DataSourcePlans, property: "Status", typ: TypeSelect, statement: `ADD COLUMN "Status" SELECT('Drafting':gray, 'In Review':yellow, 'Approved':green, 'Building':blue, 'Shipped':purple, 'Superseded':red)`},
		{dataSource: DataSourcePlans, property: "Approval", typ: TypeSelect, statement: `ADD COLUMN "Approval" SELECT('Draft':gray, 'Approved':green)`},
		{dataSource: DataSourcePlans, property: "Tags", typ: TypeMultiSelect, statement: `ADD COLUMN "Tags" MULTI_SELECT('security':red, 'compliance':orange, 'platform':blue, 'infra':purple, 'growth':green, 'developer-experience':yellow, 'ai':pink)`},
		{dataSource: DataSourcePlans, property: "Author", typ: TypePerson, statement: `ADD COLUMN "Author" PEOPLE`},
		{dataSource: DataSourcePlans, property: "Reviewers", typ: TypePerson, statement: `ADD COLUMN "Reviewers" PEOPLE`},
		{dataSource: DataSourcePlans, property: "Spec", typ: TypeRelation, target: opts.SpecsDataSource, statement: fmt.Sprintf(`ADD COLUMN "Spec" RELATION('%s')`, opts.SpecsDataSource)},
		{dataSource: DataSourcePlans, property: "Linear Issues", typ: TypeURL, statement: `ADD COLUMN "Linear Issues" URL`},
	}

	var report Report
	report.BlockingIssues = append(report.BlockingIssues, validateDataSourceIdentity(DataSourceSpecs, specs, opts.SpecsDataSource)...)
	report.BlockingIssues = append(report.BlockingIssues, validateDataSourceIdentity(DataSourcePlans, plans, opts.PlansDataSource)...)

	for _, req := range reqs {
		ds := specs
		if req.dataSource == DataSourcePlans {
			ds = plans
		}
		issue, repair, ok := validateRequirement(ds, req)
		if !ok {
			continue
		}
		if issue.Severity == SeverityBlocking {
			report.BlockingIssues = append(report.BlockingIssues, issue)
			continue
		}
		report.FixableIssues = append(report.FixableIssues, issue)
		report.Repairs = append(report.Repairs, repair)
	}

	sortIssues(report.BlockingIssues)
	sortIssues(report.FixableIssues)
	sortRepairs(report.Repairs)
	report.Valid = len(report.BlockingIssues) == 0 && len(report.FixableIssues) == 0
	return report
}

// ApplyApprovedRepairs applies approved safe additive repairs to local snapshots.
func ApplyApprovedRepairs(specs, plans DataSource, repairs []Repair, approvedIDs []string) (DataSource, DataSource, []Repair) {
	specs = cloneDataSource(specs)
	plans = cloneDataSource(plans)

	approved := make(map[string]bool, len(approvedIDs))
	for _, id := range approvedIDs {
		approved[id] = true
	}

	var applied []Repair
	for _, repair := range repairs {
		if !approved[repair.ID] {
			continue
		}
		target := &specs
		if repair.DataSource == DataSourcePlans {
			target = &plans
		}
		if target.Schema == nil {
			target.Schema = map[string]Property{}
		}
		if _, exists := target.Schema[repair.Property]; exists {
			continue
		}
		prop := Property{Name: repair.Property, Type: repair.Type}
		if repair.Type == TypeRelation {
			if repair.DataSource == DataSourceSpecs {
				prop.DataSourceURL = plans.URL
			} else {
				prop.DataSourceURL = specs.URL
			}
		}
		target.Schema[repair.Property] = prop
		applied = append(applied, repair)
	}

	sortRepairs(applied)
	return specs, plans, applied
}

func applyDefaults(opts ValidationOptions) ValidationOptions {
	if opts.SpecIDProperty == "" {
		opts.SpecIDProperty = "Spec ID"
	}
	if opts.PlanIDProperty == "" {
		opts.PlanIDProperty = "Plan ID"
	}
	return opts
}

func validateDataSourceIdentity(kind string, ds DataSource, expectedURL string) []Issue {
	var issues []Issue
	if ds.URL == "" {
		issues = append(issues, Issue{
			ID:         issueID(kind, "data_source", "missing"),
			Severity:   SeverityBlocking,
			DataSource: kind,
			Message:    fmt.Sprintf("%s data source snapshot is missing url", kind),
		})
	} else if expectedURL != "" && ds.URL != expectedURL {
		issues = append(issues, Issue{
			ID:           issueID(kind, "data_source", "url"),
			Severity:     SeverityBlocking,
			DataSource:   kind,
			ExpectedType: expectedURL,
			ActualType:   ds.URL,
			Message:      fmt.Sprintf("%s data source url %q does not match configured url %q", kind, ds.URL, expectedURL),
		})
	}
	if len(ds.Schema) == 0 {
		issues = append(issues, Issue{
			ID:         issueID(kind, "schema", "missing"),
			Severity:   SeverityBlocking,
			DataSource: kind,
			Message:    fmt.Sprintf("%s data source snapshot is missing schema", kind),
		})
	}
	return issues
}

func validateRequirement(ds DataSource, req requirement) (Issue, Repair, bool) {
	prop, ok := ds.Schema[req.property]
	if !ok {
		if req.blocking {
			return Issue{
				ID:           issueID(req.dataSource, req.property, "missing"),
				Severity:     SeverityBlocking,
				DataSource:   req.dataSource,
				Property:     req.property,
				ExpectedType: req.typ,
				Message:      fmt.Sprintf("%s data source is missing required %q property of type %q", req.dataSource, req.property, req.typ),
			}, Repair{}, true
		}
		repair := Repair{
			ID:               issueID(req.dataSource, req.property, "add"),
			DataSource:       req.dataSource,
			DataSourceURL:    ds.URL,
			Property:         req.property,
			Type:             req.typ,
			Statement:        req.statement,
			RequiresApproval: true,
		}
		return Issue{
			ID:              repair.ID,
			Severity:        SeverityFixable,
			DataSource:      req.dataSource,
			Property:        req.property,
			ExpectedType:    req.typ,
			Message:         fmt.Sprintf("%s data source can add missing %q property of type %q", req.dataSource, req.property, req.typ),
			Repairable:      true,
			RepairStatement: req.statement,
		}, repair, true
	}

	if prop.Type != req.typ {
		return Issue{
			ID:           issueID(req.dataSource, req.property, "type"),
			Severity:     SeverityBlocking,
			DataSource:   req.dataSource,
			Property:     req.property,
			ExpectedType: req.typ,
			ActualType:   prop.Type,
			Message:      fmt.Sprintf("%s data source property %q has type %q, expected %q", req.dataSource, req.property, prop.Type, req.typ),
		}, Repair{}, true
	}

	if req.target != "" && prop.DataSourceURL != "" && prop.DataSourceURL != req.target {
		return Issue{
			ID:           issueID(req.dataSource, req.property, "relation_target"),
			Severity:     SeverityBlocking,
			DataSource:   req.dataSource,
			Property:     req.property,
			ExpectedType: req.target,
			ActualType:   prop.DataSourceURL,
			Message:      fmt.Sprintf("%s data source relation %q targets %q, expected %q", req.dataSource, req.property, prop.DataSourceURL, req.target),
		}, Repair{}, true
	}

	return Issue{}, Repair{}, false
}

func issueID(dataSource, property, suffix string) string {
	return fmt.Sprintf("%s.%s.%s", dataSource, sanitizeIDPart(property), suffix)
}

func sanitizeIDPart(s string) string {
	out := make([]rune, 0, len(s))
	lastDash := false
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
			out = append(out, r+'a'-'A')
			lastDash = false
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			out = append(out, r)
			lastDash = false
		default:
			if !lastDash {
				out = append(out, '-')
				lastDash = true
			}
		}
	}
	for len(out) > 0 && out[len(out)-1] == '-' {
		out = out[:len(out)-1]
	}
	if len(out) == 0 {
		return "property"
	}
	return string(out)
}

func sortIssues(issues []Issue) {
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].ID < issues[j].ID
	})
}

func sortRepairs(repairs []Repair) {
	sort.Slice(repairs, func(i, j int) bool {
		return repairs[i].ID < repairs[j].ID
	})
}

func cloneDataSource(ds DataSource) DataSource {
	cloned := DataSource{Name: ds.Name, URL: ds.URL, Schema: map[string]Property{}}
	for k, v := range ds.Schema {
		cloned.Schema[k] = v
	}
	return cloned
}
