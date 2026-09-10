package store

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Phase 2.1 criteria 1 & 2: ParseEntry splits an optional leading YAML
// frontmatter block off an entry, reporting the tags it declares and the body
// that remains. The body is asserted alongside the tags in every case because
// the body is what term counting, title detection and excerpt collection run
// against — getting the tags right while handing back the wrong body would
// silently corrupt every search.
//
// Every expected value below is hand-written from the input beside it, never
// derived by running the parser.
func TestParseEntry_TagsAndBody(t *testing.T) {
	for name, tc := range map[string]struct {
		raw      string
		wantTags []string
		wantBody string
		wantErr  bool
	}{
		// The published docs promise both YAML list spellings, so both are
		// pinned: the inline flow sequence...
		"flow list": {
			raw:      "---\ntags: [go, http]\n---\n# Title\n\nprose here\n",
			wantTags: []string{"go", "http"},
			wantBody: "# Title\n\nprose here\n",
		},
		// ...and the indented block sequence.
		"block list": {
			raw:      "---\ntags:\n  - go\n  - http\n---\n# Title\n\nprose here\n",
			wantTags: []string{"go", "http"},
			wantBody: "# Title\n\nprose here\n",
		},
		// THE LOAD-BEARING CASE. Every knowledge entry written before tags
		// existed has no frontmatter block at all. It must parse successfully,
		// report no tags, and hand back its bytes completely untouched — that is
		// what lets tags ship with no migration and no backfill.
		"no block at all": {
			raw:      "# Existing Entry\n\nwritten long before tags existed\n",
			wantTags: nil,
			wantBody: "# Existing Entry\n\nwritten long before tags existed\n",
		},
		// A declared-but-empty list is indistinguishable from declaring nothing.
		"empty list": {
			raw:      "---\ntags: []\n---\nprose here\n",
			wantTags: nil,
			wantBody: "prose here\n",
		},
		// tags is the only key read; a team's own metadata is ignored, not
		// rejected.
		"only unrelated keys": {
			raw:      "---\nowner: platform\nreviewed: 2026-01-01\n---\nprose here\n",
			wantTags: nil,
			wantBody: "prose here\n",
		},
		// An opener that is never closed reads as a horizontal rule in prose:
		// no tags, no error, and the raw input unchanged as the body.
		"unterminated block": {
			raw:      "---\ntags: [go]\nthis entry never closes its block\n",
			wantTags: nil,
			wantBody: "---\ntags: [go]\nthis entry never closes its block\n",
		},
		// Likewise a closing marker that is not alone on its line.
		"closing marker not alone on its line": {
			raw:      "---\ntags: [go]\n---foo\nprose here\n",
			wantTags: nil,
			wantBody: "---\ntags: [go]\n---foo\nprose here\n",
		},
		// A properly delimited block whose YAML will not unmarshal is the one
		// case that returns an error — but the tags and body returned alongside
		// it are still safe to use, so a search can carry on while an audit flow
		// can report it. The flow sequence below is never closed, which is a
		// scanner error.
		"malformed yaml in a delimited block": {
			raw:      "---\ntags: [go\n---\nprose here\n",
			wantTags: nil,
			wantBody: "---\ntags: [go\n---\nprose here\n",
			wantErr:  true,
		},
		// Lower-casing on read is what stops "HTTP" and "http" ever existing as
		// two separate tags; trimming and de-duplication collapse the rest,
		// keeping first-seen order.
		"case normalised and de-duplicated": {
			raw:      "---\ntags: [Go, HTTP, go, \" http \"]\n---\nprose here\n",
			wantTags: []string{"go", "http"},
			wantBody: "prose here\n",
		},
		// A single blank line between the closing marker and the body is a
		// formatting convention, not content, so it is dropped.
		"blank separator line dropped": {
			raw:      "---\ntags: [go]\n---\n\n# Title\n\nprose here\n",
			wantTags: []string{"go"},
			wantBody: "# Title\n\nprose here\n",
		},
		// Only one blank line is dropped; a second is body content.
		"second blank line kept": {
			raw:      "---\ntags: [go]\n---\n\n\n# Title\n",
			wantTags: []string{"go"},
			wantBody: "\n# Title\n",
		},
		// A block at the very end of the file, with nothing following it at all,
		// yields an empty body rather than failing to find its close.
		"block at end of file": {
			raw:      "---\ntags: [go]\n---",
			wantTags: []string{"go"},
			wantBody: "",
		},
	} {
		t.Run(name, func(t *testing.T) {
			tags, body, err := ParseEntry([]byte(tc.raw))
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.wantTags, tags)
			require.Equal(t, tc.wantBody, string(body),
				"the body is what term counting runs against and must be exact")
		})
	}
}

// Phase 2.1 criterion 1: normaliseTags lower-cases, trims, drops empties and
// de-duplicates while preserving first-seen order, and returns nil rather than
// an empty slice so "declared no tags" and "declared an empty list" are the
// same thing to every caller.
func TestNormaliseTags_LowerCasesTrimsAndDeduplicates(t *testing.T) {
	for name, tc := range map[string]struct {
		raw  []string
		want []string
	}{
		"nil input":                 {nil, nil},
		"empty slice":               {[]string{}, nil},
		"already normalised":        {[]string{"go", "http"}, []string{"go", "http"}},
		"mixed case collapses":      {[]string{"Go", "HTTP", "go"}, []string{"go", "http"}},
		"surrounding space trimmed": {[]string{" http ", "\tgo\n"}, []string{"http", "go"}},
		"empty entries dropped":     {[]string{"", "   ", "go"}, []string{"go"}},
		"nothing survives":          {[]string{"", "  "}, nil},
		"first-seen order kept":     {[]string{"zeta", "alpha", "ZETA", "mu"}, []string{"zeta", "alpha", "mu"}},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, normaliseTags(tc.raw))
		})
	}
}
