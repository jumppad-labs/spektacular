package templates

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// specSectionGatheringSteps are the 7 spec-workflow section templates rewritten
// so each drafts its section from the interview transcript and presents the
// draft for the user to confirm or correct, instead of asking its own
// scripted question from a blank prompt. This is a fixed list (not a
// directory walk) because the acceptance criterion is specifically about
// these 7 known section-gathering templates — not "every template in a
// directory". steps/spec/00b-interview.md (the interview step itself) and
// steps/spec/08-verification.md (assembly, a different phase's concern) are
// deliberately excluded.
var specSectionGatheringSteps = []string{
	"steps/spec/01-overview.md",
	"steps/spec/02-requirements.md",
	"steps/spec/03-acceptance_criteria.md",
	"steps/spec/04-constraints.md",
	"steps/spec/05-technical_approach.md",
	"steps/spec/06-success_metrics.md",
	"steps/spec/07-non_goals.md",
}

// draftConfirmMarker is the stable, hand-written substring shared by all 7
// section templates instructing the agent to present its draft for the user
// to confirm or correct, rather than ask its own scripted question.
const draftConfirmMarker = "Present the draft to the user and ask them to confirm it or tell you what's wrong."

// interviewFileMarker is the shared reference to the interview working file
// that every section template now drafts from.
const interviewFileMarker = ".spektacular/work/{{spec_name}}/interview.md"

// oldScriptedQuestionPhrases are representative fragments of the old,
// scripted opening question each template used to ask from a blank prompt
// before this rewrite. None of these must appear anywhere in the 7 section
// templates any more. This is deliberately a list of distinct fragments
// (not one shared marker) because the old phrasing differed per section
// (e.g. "Ask the user to describe..." for Overview vs. "Ask the user: Are
// there any hard constraints..." for Constraints) — there was no single old
// marker shared across all 7 the way there is for the new phrasing.
var oldScriptedQuestionPhrases = []string{
	"Ask the user to describe",
	"Ask the user to list",
	"Ask the user: Are there any hard constraints",
	"Ask the user: Do you have any technical direction",
	"Ask the user: How will you know",
	"Ask the user: What is explicitly OUT of scope",
}

// TestSectionStepsDraftFromInterviewAndConfirm verifies the acceptance
// criterion for Phase 2.1: template unit tests confirm the scripted-question
// phrasing is gone and draft-and-confirm phrasing is present, across all 7
// spec section-gathering templates.
func TestSectionStepsDraftFromInterviewAndConfirm(t *testing.T) {
	for _, f := range specSectionGatheringSteps {
		content := mustReadTemplate(t, f)

		require.Containsf(t, content, interviewFileMarker,
			"%s must draft its section from the interview working file %s", f, interviewFileMarker)

		require.Containsf(t, content, draftConfirmMarker,
			"%s must present its draft to the user for confirmation or correction", f)

		for _, old := range oldScriptedQuestionPhrases {
			require.NotContainsf(t, content, old,
				"%s must not retain the old scripted-question phrasing %q", f, old)
		}
	}
}

// TestInterviewAndVerificationStepsUntouched guards the scope boundary for
// Phase 2.1: the interview step itself (which correctly still drives its own
// adaptive questioning) and the verification/assembly step (a different
// phase's concern) must not be mistaken for section-gathering templates and
// must not carry the new draft-and-confirm marker, since neither drafts a
// section from the interview transcript the way the 7 section steps do.
func TestInterviewAndVerificationStepsUntouched(t *testing.T) {
	interview := mustReadTemplate(t, "steps/spec/00b-interview.md")
	require.NotContainsf(t, interview, draftConfirmMarker,
		"steps/spec/00b-interview.md is the interview step itself and must not carry the section draft-and-confirm marker")

	verification := mustReadTemplate(t, "steps/spec/08-verification.md")
	require.NotContainsf(t, verification, draftConfirmMarker,
		"steps/spec/08-verification.md is the assembly step and must not carry the section draft-and-confirm marker")
}
