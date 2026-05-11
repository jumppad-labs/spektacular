package setup

import (
	"fmt"
	"strings"

	"github.com/jumppad-labs/spektacular/internal/notion/schema"
)

// Instruction describes an MCP action for an agent to perform.
type Instruction struct {
	Tool          string `json:"tool"`
	DataSource    string `json:"data_source,omitempty"`
	DataSourceURL string `json:"data_source_url,omitempty"`
	Statement     string `json:"statement,omitempty"`
	Title         string `json:"title,omitempty"`
	Schema        string `json:"schema,omitempty"`
	Description   string `json:"description"`
}

// RepairInstructions converts approved repairs into Notion MCP update steps.
func RepairInstructions(repairs []schema.Repair) []Instruction {
	instructions := make([]Instruction, 0, len(repairs))
	for _, repair := range repairs {
		instructions = append(instructions, Instruction{
			Tool:          "notion_update_data_source",
			DataSource:    repair.DataSource,
			DataSourceURL: repair.DataSourceURL,
			Statement:     repair.Statement,
			Description:   fmt.Sprintf("Apply approved additive repair %s for %s.%s.", repair.ID, repair.DataSource, repair.Property),
		})
	}
	return instructions
}

// CreateDatabaseInstructions returns MCP create steps for a managed initial setup.
func CreateDatabaseInstructions(basePageURL string) []Instruction {
	parent := "the chosen Notion parent page"
	if basePageURL != "" {
		parent = basePageURL
	}

	return []Instruction{
		{
			Tool:        "notion_create_database",
			Title:       "Specs",
			Schema:      specsDatabaseDDL(),
			Description: fmt.Sprintf("Create the Specs data source under %s, then fetch it and pass the resulting data-source snapshot to `spektacular notion link`.", parent),
		},
		{
			Tool:        "notion_create_database",
			Title:       "Plans",
			Schema:      plansDatabaseDDL(),
			Description: fmt.Sprintf("Create the Plans data source under %s, then fetch it and pass the resulting data-source snapshot to `spektacular notion link`.", parent),
		},
	}
}

func specsDatabaseDDL() string {
	cols := []string{
		`"Name" TITLE`,
		`"Spec ID" UNIQUE_ID PREFIX 'SPEC'`,
		`"Status" SELECT('Drafting':gray, 'In Review':yellow, 'Approved':green, 'Building':blue, 'Shipped':purple, 'Superseded':red)`,
		`"Approval" SELECT('Draft':gray, 'Approved':green)`,
		`"Tags" MULTI_SELECT('security':red, 'compliance':orange, 'platform':blue, 'infra':purple, 'growth':green, 'developer-experience':yellow, 'ai':pink)`,
		`"Author" PEOPLE`,
		`"Reviewers" PEOPLE`,
	}
	return "CREATE TABLE (" + strings.Join(cols, ", ") + ")"
}

func plansDatabaseDDL() string {
	cols := []string{
		`"Name" TITLE`,
		`"Plan ID" UNIQUE_ID PREFIX 'PLAN'`,
		`"Status" SELECT('Drafting':gray, 'In Review':yellow, 'Approved':green, 'Building':blue, 'Shipped':purple, 'Superseded':red)`,
		`"Approval" SELECT('Draft':gray, 'Approved':green)`,
		`"Tags" MULTI_SELECT('security':red, 'compliance':orange, 'platform':blue, 'infra':purple, 'growth':green, 'developer-experience':yellow, 'ai':pink)`,
		`"Author" PEOPLE`,
		`"Reviewers" PEOPLE`,
		`"Linear Issues" URL`,
	}
	return "CREATE TABLE (" + strings.Join(cols, ", ") + ")"
}
