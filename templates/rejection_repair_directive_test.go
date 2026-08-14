package templates

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// rejectionRepairMarker is a hand-written, stable substring of the shared
// "if the user rejects this draft" paragraph appended to every spec
// gathering-step instruction template (the interview step plus all seven
// section-drafting steps). We assert on this substring rather than the whole
// paragraph so the test is not brittle against minor prose changes.
const rejectionRepairMarker = "If the user indicates this draft is wrong, ask a follow-up question to understand why before changing anything"

// gatheringStepsWithRejectionRepair are the eight spec-workflow templates
// that present a draft for the user to confirm or correct, and so must all
// carry the identical rejection-repair paragraph: the interview step plus
// every section-drafting step from Overview through Non-Goals.
var gatheringStepsWithRejectionRepair = []string{
	"steps/spec/00b-interview.md",
	"steps/spec/01-overview.md",
	"steps/spec/02-requirements.md",
	"steps/spec/03-acceptance_criteria.md",
	"steps/spec/04-constraints.md",
	"steps/spec/05-technical_approach.md",
	"steps/spec/06-success_metrics.md",
	"steps/spec/07-non_goals.md",
}

// TestRejectionRepairPresent verifies Phase 2.2's acceptance criteria: every
// spec gathering step (the interview plus all seven section-drafting steps)
// carries the identical rejection-repair paragraph exactly once, generalizing
// the triage-and-fix pattern that 08-verification.md's end-of-workflow review
// already established for the assembled spec as a whole.
func TestRejectionRepairPresent(t *testing.T) {
	for _, f := range gatheringStepsWithRejectionRepair {
		body := mustReadTemplate(t, f)
		count := strings.Count(body, rejectionRepairMarker)
		require.Equalf(t, 1, count,
			"%s must contain the rejection-repair directive exactly once (found %d)", f, count)
	}
}

// TestRejectionRepairAbsentFromVerification guards the scope boundary: the
// end-of-workflow verification step is where the reviewed triage-and-fix loop
// already lives (in its own, more elaborate form) — it must not also carry
// the generalized per-section marker, which would be a sign the two patterns
// were accidentally conflated.
func TestRejectionRepairAbsentFromVerification(t *testing.T) {
	body := mustReadTemplate(t, "steps/spec/08-verification.md")
	require.NotContains(t, body, rejectionRepairMarker)
}
