package templates

import (
	"io/fs"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// contextDirectiveMarker is a hand-written, stable substring of the
// "refresh working context" directive appended to every non-terminal step
// instruction template. We assert on this substring (not the whole block) so
// the test is not brittle against the em-dashes and trailing prose in the
// rendered directive.
const contextDirectiveMarker = "not a copy of content already captured elsewhere"

// stepDirs are the three workflows whose non-terminal step templates must all
// carry the refresh-context directive.
var stepDirs = []string{
	"steps/spec",
	"steps/plan",
	"steps/implement",
}

// TestContextDirectivePresent verifies the acceptance criteria for Phase 3.1:
//
//  1. Every spec, plan, and implement non-terminal step instruction asks the
//     agent to refresh .spektacular/context.md before advancing.
//  2. The directive wording is identical across all three workflows (enforced
//     by matching the single hand-written marker everywhere).
//  3. Terminal *-finished.md templates are unchanged and do NOT contain the
//     directive — guarding against a future blanket-append that wrongly hits
//     the finished templates.
func TestContextDirectivePresent(t *testing.T) {
	// directiveBearing counts non-terminal templates that contain the marker.
	// Used only as a lower-bound sanity check so an empty or misrouted walk
	// cannot pass vacuously.
	directiveBearing := 0

	for _, dir := range stepDirs {
		err := fs.WalkDir(FS, dir, func(p string, d fs.DirEntry, err error) error {
			require.NoError(t, err)
			if d.IsDir() {
				return nil
			}
			if path.Ext(p) != ".md" {
				return nil
			}

			content, readErr := FS.ReadFile(p)
			require.NoError(t, readErr, "reading %s", p)
			body := string(content)
			count := strings.Count(body, contextDirectiveMarker)

			if strings.HasSuffix(path.Base(p), "-finished.md") {
				// Terminal templates must never carry the directive.
				require.Zerof(t, count,
					"terminal template %s must NOT contain the refresh-context directive", p)
				return nil
			}

			// Non-terminal templates must carry the directive exactly once.
			require.Equalf(t, 1, count,
				"non-terminal template %s must contain the refresh-context directive exactly once (found %d)", p, count)
			directiveBearing++
			return nil
		})
		require.NoErrorf(t, err, "walking %s", dir)
	}

	// Floor, not an exact match: spec (8) + plan (16) + implement (8) = 32
	// non-terminal templates today. Kept as a lower bound so adding a step
	// does not break the test, while still catching a walk that finds nothing.
	require.GreaterOrEqual(t, directiveBearing, 30,
		"expected at least 30 directive-bearing step templates across the three workflows")
}
