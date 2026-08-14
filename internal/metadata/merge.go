package metadata

import (
	"time"
)

// UpdateOptions carries caller-supplied field updates for Merge. Every field
// is optional. Status nil means "no status change" (or "in-progress" on a
// first-ever write). Today is caller-injected for determinism in tests; when
// zero, time.Now().UTC() truncated to day is used.
type UpdateOptions struct {
	Status *Status
	Today  time.Time
	// Provenance updates for derived changelog entries. Same semantics as
	// Status: an empty value means "no change", a non-empty incoming value
	// wins, and existing values are preserved otherwise.
	Project       string
	ProjectSource string
	Spec          string
	Plan          string
}

// Merge computes the on-disk bytes for a write by combining an existing store
// blob with caller-supplied field updates and a new body. Invariants:
//
//   - On first write (existing has no metadata), CreatedDate is stamped to
//     today and Status defaults to StatusInProgress unless overridden.
//   - On subsequent writes, CreatedDate is preserved.
//   - Status is enum-validated when opts.Status is non-nil.
//   - ClosedDate is stamped exactly once, on the transition from in-progress
//     to any closed status; subsequent same-or-different closed states keep
//     the first-transition date.
//   - A transition back to in-progress clears ClosedDate.
//   - Malformed existing frontmatter propagates as an error rather than being
//     silently replaced.
func Merge(existing []byte, newBody []byte, opts UpdateOptions) ([]byte, error) {
	today := opts.Today
	if today.IsZero() {
		today = time.Now().UTC()
	}
	today = today.UTC().Truncate(24 * time.Hour)

	var current Metadata
	fresh := true
	if len(existing) > 0 {
		fm, _, err := Split(existing)
		if err != nil {
			return nil, err
		}
		if fm != nil {
			current = *fm
			fresh = false
		}
	}

	var result Metadata
	if fresh {
		result.Project = opts.Project
		result.ProjectSource = opts.ProjectSource
		result.Spec = opts.Spec
		result.Plan = opts.Plan
		result.CreatedDate = today
		result.Status = StatusInProgress
		if opts.Status != nil {
			if err := validateStatus(*opts.Status); err != nil {
				return nil, err
			}
			result.Status = *opts.Status
			if isClosed(result.Status) {
				result.ClosedDate = today
			}
		}
	} else {
		result.CreatedDate = current.CreatedDate
		result.Status = current.Status
		result.ClosedDate = current.ClosedDate
		result.Project = current.Project
		result.ProjectSource = current.ProjectSource
		result.Spec = current.Spec
		result.Plan = current.Plan
		if opts.Project != "" {
			result.Project = opts.Project
		}
		if opts.ProjectSource != "" {
			result.ProjectSource = opts.ProjectSource
		}
		if opts.Spec != "" {
			result.Spec = opts.Spec
		}
		if opts.Plan != "" {
			result.Plan = opts.Plan
		}
		if opts.Status != nil {
			if err := validateStatus(*opts.Status); err != nil {
				return nil, err
			}
			result.Status = *opts.Status
			if isClosed(result.Status) {
				if result.ClosedDate.IsZero() {
					result.ClosedDate = today
				}
			} else {
				result.ClosedDate = time.Time{}
			}
		}
	}

	return Render(result, newBody)
}
