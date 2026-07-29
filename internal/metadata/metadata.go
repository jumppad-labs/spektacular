// Package metadata owns the artifact-frontmatter schema Spektacular writes at
// the top of every workflow-produced document. It is the single Go module that
// parses, renders, and merges the `created_date` / `status` / `closed_date`
// block; every write site in the codebase — the CLI `<kind> file write`
// handler and each workflow's own `st.Write` callbacks — routes through this
// package so the schema stays consistent across sites.
package metadata

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// Status is the lifecycle state of a workflow-produced artifact. It is one of
// the four values declared below; any other value is rejected at the type
// boundary.
type Status string

const (
	// StatusInProgress marks an artifact whose owning workflow is still open.
	StatusInProgress Status = "in-progress"
	// StatusCompleted marks an artifact whose owning workflow reached its
	// terminal step successfully.
	StatusCompleted Status = "completed"
	// StatusSuperseded marks an artifact replaced by a later document.
	StatusSuperseded Status = "superseded"
	// StatusArchived marks an artifact removed from active view.
	StatusArchived Status = "archived"
)

const dateFormat = "2006-01-02"

// Metadata is the in-memory mirror of an artifact's YAML frontmatter block.
// CreatedDate is stamped on first write and preserved thereafter. ClosedDate
// is zero while Status is in-progress and stamped once at the transition to
// any closed status.
type Metadata struct {
	CreatedDate time.Time
	Status      Status
	ClosedDate  time.Time
}

// yamlShape is the on-disk representation used by MarshalYAML / UnmarshalYAML.
// It carries the two dates as YYYY-MM-DD strings so day precision is enforced
// at the type boundary rather than at every call site.
type yamlShape struct {
	CreatedDate string `yaml:"created_date"`
	Status      Status `yaml:"status"`
	ClosedDate  string `yaml:"closed_date,omitempty"`
}

// MarshalYAML implements yaml.Marshaler.
func (m Metadata) MarshalYAML() (interface{}, error) {
	out := yamlShape{
		CreatedDate: m.CreatedDate.Format(dateFormat),
		Status:      m.Status,
	}
	if !m.ClosedDate.IsZero() {
		out.ClosedDate = m.ClosedDate.Format(dateFormat)
	}
	return out, nil
}

// UnmarshalYAML implements yaml.Unmarshaler. It parses the two date fields as
// YYYY-MM-DD and rejects a status value outside the four-value enum.
func (m *Metadata) UnmarshalYAML(node *yaml.Node) error {
	var in yamlShape
	if err := node.Decode(&in); err != nil {
		return err
	}
	created, err := time.Parse(dateFormat, in.CreatedDate)
	if err != nil {
		return fmt.Errorf("parsing created_date %q: %w", in.CreatedDate, err)
	}
	m.CreatedDate = created
	if err := validateStatus(in.Status); err != nil {
		return err
	}
	m.Status = in.Status
	if in.ClosedDate != "" {
		closed, err := time.Parse(dateFormat, in.ClosedDate)
		if err != nil {
			return fmt.Errorf("parsing closed_date %q: %w", in.ClosedDate, err)
		}
		m.ClosedDate = closed
	}
	return nil
}

// validateStatus returns an error if s is not one of the four enum values.
func validateStatus(s Status) error {
	switch s {
	case StatusInProgress, StatusCompleted, StatusSuperseded, StatusArchived:
		return nil
	}
	return fmt.Errorf("invalid status %q; must be one of %q, %q, %q, %q",
		s, StatusInProgress, StatusCompleted, StatusSuperseded, StatusArchived)
}

// isClosed reports whether s is any status other than in-progress.
func isClosed(s Status) bool {
	return s == StatusCompleted || s == StatusSuperseded || s == StatusArchived
}
