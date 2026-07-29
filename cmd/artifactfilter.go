package cmd

import (
	"fmt"
	"time"

	"github.com/jumppad-labs/spektacular/internal/metadata"
	"github.com/jumppad-labs/spektacular/internal/output"
)

// artifactFilter represents the five combinable filter flags exposed on
// `<kind> file list` (and reused by `spektacular artifacts list`). Every field
// is optional; a zero-value filter matches every artifact — the `active` guard
// lets callers preserve the pre-shipping bare-artifact behaviour when no flag
// is set.
type artifactFilter struct {
	status        metadata.Status
	createdAfter  time.Time
	createdBefore time.Time
	closedAfter   time.Time
	closedBefore  time.Time
}

// active reports whether any filter flag has been set. A caller that wants to
// preserve the pre-shipping behaviour on bare-string listings ("show every
// entry, metadata-less ones included") should skip metadata matching when
// active returns false.
func (f artifactFilter) active() bool {
	return f.status != "" ||
		!f.createdAfter.IsZero() ||
		!f.createdBefore.IsZero() ||
		!f.closedAfter.IsZero() ||
		!f.closedBefore.IsZero()
}

// matches reports whether m satisfies every set filter (AND semantics — the
// intersection, not the union).
func (f artifactFilter) matches(m metadata.Metadata) bool {
	if f.status != "" && m.Status != f.status {
		return false
	}
	created := m.CreatedDate
	if !f.createdAfter.IsZero() && created.Before(f.createdAfter) {
		return false
	}
	if !f.createdBefore.IsZero() && created.After(f.createdBefore) {
		return false
	}
	closed := m.ClosedDate
	if !f.closedAfter.IsZero() {
		if closed.IsZero() || closed.Before(f.closedAfter) {
			return false
		}
	}
	if !f.closedBefore.IsZero() {
		if closed.IsZero() || closed.After(f.closedBefore) {
			return false
		}
	}
	return true
}

// parseListFilter validates and parses the five raw filter-flag values from
// the `<kind> file list` and `spektacular artifacts list` command surfaces.
// Empty strings mean "no filter". A status outside the enum or a date not in
// YYYY-MM-DD form is rejected with an actionable error.
func parseListFilter(status, createdAfter, createdBefore, closedAfter, closedBefore string) (artifactFilter, error) {
	var f artifactFilter
	if status != "" {
		s := metadata.Status(status)
		switch s {
		case metadata.StatusInProgress, metadata.StatusCompleted, metadata.StatusSuperseded, metadata.StatusArchived:
			f.status = s
		default:
			return artifactFilter{}, output.NewError("invalid_status",
				fmt.Sprintf("--status %q is not one of the four allowed values", status)).
				WithNextAction("Pass one of in-progress, completed, superseded, archived.")
		}
	}
	parseDate := func(flag, raw string) (time.Time, error) {
		if raw == "" {
			return time.Time{}, nil
		}
		t, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return time.Time{}, output.NewError("invalid_date",
				fmt.Sprintf("--%s %q is not a valid YYYY-MM-DD date", flag, raw)).
				WithNextAction("Pass a date in YYYY-MM-DD form, e.g. 2026-07-01.")
		}
		return t.UTC(), nil
	}
	var err error
	if f.createdAfter, err = parseDate("created-after", createdAfter); err != nil {
		return artifactFilter{}, err
	}
	if f.createdBefore, err = parseDate("created-before", createdBefore); err != nil {
		return artifactFilter{}, err
	}
	if f.closedAfter, err = parseDate("closed-after", closedAfter); err != nil {
		return artifactFilter{}, err
	}
	if f.closedBefore, err = parseDate("closed-before", closedBefore); err != nil {
		return artifactFilter{}, err
	}
	return f, nil
}
