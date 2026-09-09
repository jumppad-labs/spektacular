package templates

import (
	"io/fs"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// vocabularyContainmentMarker is a hand-written, stable substring of the
// standing "internals are yours, not the user's" rule carried by every
// guided-add instruction. We assert on this substring rather than the whole
// paragraph so the test is not brittle against minor prose changes.
//
// The rule exists because the failure it prevents is not a wording slip: an
// instruction that teaches the agent Spektacular's vocabulary gets that
// vocabulary reproduced in the agent's own words, in front of a user who
// never asked to learn it.
const vocabularyContainmentMarker = "did not ask to learn them"

// TestVocabularyContainmentPresent verifies that every guided-add
// instruction carries the vocabulary-containment rule exactly once,
// terminal instruction included. Unlike the refresh-context directive,
// which a terminal step must not carry, this rule binds every step: the
// closing report is spoken to the user like any other and can leak just as
// easily.
//
// The directory is walked rather than listed, so an instruction added later
// is covered automatically.
func TestVocabularyContainmentPresent(t *testing.T) {
	covered := 0

	err := fs.WalkDir(FS, "steps/repo", func(p string, d fs.DirEntry, err error) error {
		require.NoError(t, err)
		if d.IsDir() || path.Ext(p) != ".md" {
			return nil
		}

		body := mustReadTemplate(t, p)
		count := strings.Count(body, vocabularyContainmentMarker)
		require.Equalf(t, 1, count,
			"guided-add instruction %s must carry the vocabulary-containment rule exactly once (found %d)", p, count)
		covered++
		return nil
	})
	require.NoError(t, err, "walking steps/repo")

	require.GreaterOrEqual(t, covered, 9,
		"expected at least the nine guided-add instructions to carry the rule")
}

// TestGuidedAddInstructionsQuoteNoCommandToTheUser verifies that the guided
// add's user-facing example phrasing never contains command text. The
// instructions necessarily tell the agent which command to run next, so this
// asserts the narrower property that matters: the illustrative sentences
// offered as things to say to the user carry no command name.
func TestGuidedAddInstructionsQuoteNoCommandToTheUser(t *testing.T) {
	// The confirmation step is the one that supplies a worked sentence for
	// the agent to say, so it is where a leak would land.
	body := mustReadTemplate(t, "steps/repo/07-confirm.md")

	require.Contains(t, body, "Do not put any command in front of them",
		"the confirmation step must forbid showing command text to the user")

	for _, banned := range []string{"repo add", "--data", "--force"} {
		require.NotContainsf(t, exampleSentence(body), banned,
			"the confirmation step's worked example must not contain %q", banned)
	}
}

// exampleSentence returns the quoted worked example the confirmation
// instruction offers, which is the only sentence in these templates written
// to be said aloud to the user.
func exampleSentence(body string) string {
	const open = `"I'll register`
	i := strings.Index(body, open)
	if i < 0 {
		return ""
	}
	rest := body[i+1:]
	j := strings.Index(rest, `"`)
	if j < 0 {
		return rest
	}
	return rest[:j]
}
