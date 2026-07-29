package cmd

import (
	"testing"
	"time"

	"github.com/jumppad-labs/spektacular/internal/metadata"
	"github.com/jumppad-labs/spektacular/internal/output"
	"github.com/stretchr/testify/require"
)

// mustDate parses a YYYY-MM-DD string to a UTC time.Time or fails the test —
// keeps the table-driven cases in this file focused on filter semantics rather
// than repeating time.Parse boilerplate.
func mustDate(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := time.Parse("2006-01-02", s)
	require.NoError(t, err)
	return tm.UTC()
}

// TestArtifactFilter_ZeroValueIsInactiveAndMatchesEverything asserts that a
// zero-value artifactFilter is inactive (no filter flag set) and unconditionally
// matches any Metadata — the "unfiltered" list-command path relies on this.
func TestArtifactFilter_ZeroValueIsInactiveAndMatchesEverything(t *testing.T) {
	var f artifactFilter
	require.False(t, f.active())

	cases := []metadata.Metadata{
		{},
		{Status: metadata.StatusInProgress, CreatedDate: mustDate(t, "2026-01-15")},
		{Status: metadata.StatusCompleted, CreatedDate: mustDate(t, "2026-02-15"), ClosedDate: mustDate(t, "2026-03-01")},
		{Status: metadata.StatusArchived},
		{Status: metadata.StatusSuperseded, CreatedDate: mustDate(t, "2025-12-31"), ClosedDate: mustDate(t, "2026-01-01")},
	}
	for i, m := range cases {
		require.Truef(t, f.matches(m), "zero filter must match every metadata (case %d)", i)
	}
}

// TestParseListFilter_EmptyReturnsInactive asserts that empty strings for every
// flag produce a zero-value filter that reports itself inactive.
func TestParseListFilter_EmptyReturnsInactive(t *testing.T) {
	f, err := parseListFilter("", "", "", "", "")
	require.NoError(t, err)
	require.False(t, f.active())
}

// TestParseListFilter_StatusSetsActive asserts a valid status alone activates
// the filter and lands in the parsed struct as the typed metadata.Status.
func TestParseListFilter_StatusSetsActive(t *testing.T) {
	for _, s := range []metadata.Status{
		metadata.StatusInProgress,
		metadata.StatusCompleted,
		metadata.StatusSuperseded,
		metadata.StatusArchived,
	} {
		t.Run(string(s), func(t *testing.T) {
			f, err := parseListFilter(string(s), "", "", "", "")
			require.NoError(t, err)
			require.True(t, f.active())
			require.Equal(t, s, f.status)
		})
	}
}

// TestParseListFilter_DateFlagsParseAsUTC asserts each of the four date flags
// parses independently as a YYYY-MM-DD date and lands in the corresponding
// artifactFilter field, in UTC.
func TestParseListFilter_DateFlagsParseAsUTC(t *testing.T) {
	f, err := parseListFilter("", "2026-01-01", "2026-01-31", "2026-02-01", "2026-02-28")
	require.NoError(t, err)
	require.True(t, f.active())
	require.True(t, f.createdAfter.Equal(mustDate(t, "2026-01-01")))
	require.True(t, f.createdBefore.Equal(mustDate(t, "2026-01-31")))
	require.True(t, f.closedAfter.Equal(mustDate(t, "2026-02-01")))
	require.True(t, f.closedBefore.Equal(mustDate(t, "2026-02-28")))
	require.Equal(t, time.UTC, f.createdAfter.Location())
	require.Equal(t, time.UTC, f.closedBefore.Location())
}

// TestParseListFilter_InvalidStatusReturnsInvalidStatusCode asserts a status
// value outside the four-value enum is rejected with the "invalid_status"
// error code and an actionable message naming the offending value.
func TestParseListFilter_InvalidStatusReturnsInvalidStatusCode(t *testing.T) {
	for _, bad := range []string{"bogus", "IN-PROGRESS", "done", "in_progress"} {
		t.Run(bad, func(t *testing.T) {
			_, err := parseListFilter(bad, "", "", "", "")
			require.Error(t, err)
			var er *output.ErrorResponse
			require.ErrorAs(t, err, &er)
			require.Equal(t, "invalid_status", er.Code)
			require.Contains(t, er.Message, bad)
			require.NotEmpty(t, er.NextAction)
		})
	}
}

// TestParseListFilter_InvalidDateReturnsInvalidDateCode asserts a date value
// not in YYYY-MM-DD form (either free-form English or a numerically impossible
// date) is rejected with the "invalid_date" error code. Each of the four date
// flags is exercised so no single-flag rejection is missed.
func TestParseListFilter_InvalidDateReturnsInvalidDateCode(t *testing.T) {
	badDates := []string{"yesterday", "2026-13-01", "2026/01/01", "not-a-date"}
	for _, bad := range badDates {
		t.Run("created-after "+bad, func(t *testing.T) {
			_, err := parseListFilter("", bad, "", "", "")
			require.Error(t, err)
			var er *output.ErrorResponse
			require.ErrorAs(t, err, &er)
			require.Equal(t, "invalid_date", er.Code)
			require.Contains(t, er.Message, "created-after")
			require.Contains(t, er.Message, bad)
		})
		t.Run("created-before "+bad, func(t *testing.T) {
			_, err := parseListFilter("", "", bad, "", "")
			require.Error(t, err)
			var er *output.ErrorResponse
			require.ErrorAs(t, err, &er)
			require.Equal(t, "invalid_date", er.Code)
			require.Contains(t, er.Message, "created-before")
		})
		t.Run("closed-after "+bad, func(t *testing.T) {
			_, err := parseListFilter("", "", "", bad, "")
			require.Error(t, err)
			var er *output.ErrorResponse
			require.ErrorAs(t, err, &er)
			require.Equal(t, "invalid_date", er.Code)
			require.Contains(t, er.Message, "closed-after")
		})
		t.Run("closed-before "+bad, func(t *testing.T) {
			_, err := parseListFilter("", "", "", "", bad)
			require.Error(t, err)
			var er *output.ErrorResponse
			require.ErrorAs(t, err, &er)
			require.Equal(t, "invalid_date", er.Code)
			require.Contains(t, er.Message, "closed-before")
		})
	}
}

// TestArtifactFilter_StatusMatchesOnlyEqual asserts the status filter matches
// only artifacts whose stored status is byte-equal to the filter's status.
func TestArtifactFilter_StatusMatchesOnlyEqual(t *testing.T) {
	f := artifactFilter{status: metadata.StatusCompleted}
	require.True(t, f.active())
	require.True(t, f.matches(metadata.Metadata{Status: metadata.StatusCompleted}))
	require.False(t, f.matches(metadata.Metadata{Status: metadata.StatusInProgress}))
	require.False(t, f.matches(metadata.Metadata{Status: metadata.StatusSuperseded}))
	require.False(t, f.matches(metadata.Metadata{Status: metadata.StatusArchived}))
	require.False(t, f.matches(metadata.Metadata{}))
}

// TestArtifactFilter_CreatedDateBoundariesInclusive asserts that the
// created-date range is inclusive on both ends: a created_date equal to the
// after or before bound matches, while one day outside is excluded.
func TestArtifactFilter_CreatedDateBoundariesInclusive(t *testing.T) {
	f := artifactFilter{
		createdAfter:  mustDate(t, "2026-01-15"),
		createdBefore: mustDate(t, "2026-01-15"),
	}
	require.True(t, f.active())
	require.True(t, f.matches(metadata.Metadata{CreatedDate: mustDate(t, "2026-01-15")}),
		"a created_date equal to both bounds must match — dates on the boundary are inclusive")
	require.False(t, f.matches(metadata.Metadata{CreatedDate: mustDate(t, "2026-01-14")}),
		"one day before the after bound must not match")
	require.False(t, f.matches(metadata.Metadata{CreatedDate: mustDate(t, "2026-01-16")}),
		"one day after the before bound must not match")
}

// TestArtifactFilter_ClosedDateBoundariesInclusive is the closed-date twin of
// TestArtifactFilter_CreatedDateBoundariesInclusive. It also confirms that an
// artifact with a non-zero closed_date on the boundary is accepted.
func TestArtifactFilter_ClosedDateBoundariesInclusive(t *testing.T) {
	f := artifactFilter{
		closedAfter:  mustDate(t, "2026-02-15"),
		closedBefore: mustDate(t, "2026-02-15"),
	}
	require.True(t, f.matches(metadata.Metadata{
		Status:      metadata.StatusCompleted,
		ClosedDate:  mustDate(t, "2026-02-15"),
		CreatedDate: mustDate(t, "2026-02-01"),
	}))
	require.False(t, f.matches(metadata.Metadata{
		Status:     metadata.StatusCompleted,
		ClosedDate: mustDate(t, "2026-02-14"),
	}))
	require.False(t, f.matches(metadata.Metadata{
		Status:     metadata.StatusCompleted,
		ClosedDate: mustDate(t, "2026-02-16"),
	}))
}

// TestArtifactFilter_ClosedDateFiltersRejectZeroClosedDate asserts the load-
// bearing "in-progress artifacts have no closed_date" semantics: any active
// closed-date filter must reject a Metadata whose ClosedDate is the zero
// time.Time. Without this, an in-progress artifact would silently satisfy a
// `--closed-after 2020-01-01` query.
func TestArtifactFilter_ClosedDateFiltersRejectZeroClosedDate(t *testing.T) {
	inProgress := metadata.Metadata{
		Status:      metadata.StatusInProgress,
		CreatedDate: mustDate(t, "2026-01-15"),
	}
	require.True(t, inProgress.ClosedDate.IsZero(),
		"precondition: an in-progress artifact has a zero closed_date")

	afterOnly := artifactFilter{closedAfter: mustDate(t, "2020-01-01")}
	require.False(t, afterOnly.matches(inProgress),
		"closed-after must reject an artifact with no closed_date")

	beforeOnly := artifactFilter{closedBefore: mustDate(t, "2099-12-31")}
	require.False(t, beforeOnly.matches(inProgress),
		"closed-before must reject an artifact with no closed_date")
}

// TestArtifactFilter_CombinedFiltersIntersect asserts that setting more than
// one filter dimension yields the intersection, not the union — an artifact
// must satisfy every set predicate to match.
func TestArtifactFilter_CombinedFiltersIntersect(t *testing.T) {
	f := artifactFilter{
		status:        metadata.StatusCompleted,
		createdAfter:  mustDate(t, "2026-02-01"),
		createdBefore: mustDate(t, "2026-02-28"),
		closedAfter:   mustDate(t, "2026-03-01"),
	}
	require.True(t, f.active())

	require.True(t, f.matches(metadata.Metadata{
		Status:      metadata.StatusCompleted,
		CreatedDate: mustDate(t, "2026-02-15"),
		ClosedDate:  mustDate(t, "2026-03-10"),
	}), "an artifact satisfying every predicate must match")

	require.False(t, f.matches(metadata.Metadata{
		Status:      metadata.StatusInProgress,
		CreatedDate: mustDate(t, "2026-02-15"),
		ClosedDate:  mustDate(t, "2026-03-10"),
	}), "wrong status must exclude even when dates satisfy the range")

	require.False(t, f.matches(metadata.Metadata{
		Status:      metadata.StatusCompleted,
		CreatedDate: mustDate(t, "2026-01-15"),
		ClosedDate:  mustDate(t, "2026-03-10"),
	}), "created_date outside the range must exclude even with matching status and closed_date")

	require.False(t, f.matches(metadata.Metadata{
		Status:      metadata.StatusCompleted,
		CreatedDate: mustDate(t, "2026-02-15"),
		ClosedDate:  mustDate(t, "2026-02-20"),
	}), "closed_date outside the range must exclude even with matching status and created_date")
}
