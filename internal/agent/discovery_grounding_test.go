package agent

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/stretchr/testify/require"
)

// discoveryGroundingRenderedDefault is the real templates/agents/discovery-grounding.md
// content with {{command}} rendered by hand for cfg.Command = "go run .". Kept
// byte-accurate with the source template rather than computed via mustache.Render.
const discoveryGroundingRenderedDefault = "## Discovery Grounding\n" +
	"\n" +
	"> Managed by `go run . init` — edit `templates/agents/discovery-grounding.md`\n" +
	"> in the Spektacular source, not this section in place. Hand edits will not\n" +
	"> survive the next init.\n" +
	"\n" +
	"When investigating this codebase — answering \"how does X work\" or \"why does\n" +
	"Y look like this\" outside of a structured spec/plan/implement workflow —\n" +
	"ground your discovery in the current code and conversation context, not in\n" +
	"specs or plans. Source code is the source of truth for what the system\n" +
	"*does*; a spec or plan can be stale, partially implemented, or superseded,\n" +
	"so treating one as ground truth risks investigating a fiction instead of\n" +
	"the actual system. If a spec or plan is genuinely relevant to a question\n" +
	"the code can't answer (an intentional but non-obvious constraint or\n" +
	"tradeoff), it's fine to read it — but verify what it claims against the\n" +
	"code before relying on it.\n" +
	"\n" +
	"The `.spektacular/` directory is not part of the codebase for this purpose.\n" +
	"It holds generated artifacts *about* the codebase — specs, plans, knowledge\n" +
	"entries, context, changelogs — not the system itself. Don't sweep it in\n" +
	"when asked to search or read \"the codebase\"; a broad grep or file scan\n" +
	"should treat `.spektacular/` as out of scope unless the task explicitly\n" +
	"concerns specs, plans, or knowledge.\n" +
	"\n" +
	"This does not apply to the spec/plan/implement workflows themselves, which\n" +
	"read `.spektacular/` deliberately and by design (e.g. `spek-implement`\n" +
	"reading the approved plan, or a discovery step's own prior-research lookup)\n" +
	"— those steps already say when to consult it.\n"

func TestInstallDiscoveryGroundingSection_CreatesFromMissing(t *testing.T) {
	tmp := t.TempDir()
	cfg := config.Config{Command: "go run ."}

	require.NoError(t, installDiscoveryGroundingSection(tmp, cfg, io.Discard))

	got, err := os.ReadFile(filepath.Join(tmp, "AGENTS.md"))
	require.NoError(t, err)
	require.Equal(t, discoveryGroundingRenderedDefault, string(got))
}

func TestInstallDiscoveryGroundingSection_AppendsAfterTesslBlock(t *testing.T) {
	tmp := t.TempDir()
	cfg := config.Config{Command: "go run ."}

	existing := "# Agent Rules <!-- tessl-managed -->\n\n@.tessl/RULES.md\n"
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte(existing), 0644))

	require.NoError(t, installDiscoveryGroundingSection(tmp, cfg, io.Discard))

	want := "# Agent Rules <!-- tessl-managed -->\n" +
		"\n" +
		"@.tessl/RULES.md\n" +
		"\n" +
		discoveryGroundingRenderedDefault
	got, err := os.ReadFile(filepath.Join(tmp, "AGENTS.md"))
	require.NoError(t, err)
	require.Equal(t, want, string(got))
}

func TestInstallDiscoveryGroundingSection_IsIdempotent(t *testing.T) {
	tmp := t.TempDir()
	cfg := config.Config{Command: "go run ."}

	existing := "# Agent Rules <!-- tessl-managed -->\n\n@.tessl/RULES.md\n"
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte(existing), 0644))

	require.NoError(t, installDiscoveryGroundingSection(tmp, cfg, io.Discard))
	first, err := os.ReadFile(filepath.Join(tmp, "AGENTS.md"))
	require.NoError(t, err)

	require.NoError(t, installDiscoveryGroundingSection(tmp, cfg, io.Discard))
	second, err := os.ReadFile(filepath.Join(tmp, "AGENTS.md"))
	require.NoError(t, err)

	require.Equal(t, string(first), string(second), "second run must produce byte-identical output")
}

func TestInstallDiscoveryGroundingSection_PreservesSurroundingContent(t *testing.T) {
	tmp := t.TempDir()
	cfg := config.Config{Command: "go run ."}

	existing := "# Agent Rules <!-- tessl-managed -->\n" +
		"\n" +
		"@.tessl/RULES.md\n" +
		"\n" +
		"## Discovery Grounding\n" +
		"\n" +
		"STALE BODY THAT SHOULD BE REPLACED.\n" +
		"\n" +
		"## Other Section\n" +
		"\n" +
		"unrelated content preserved verbatim\n"
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte(existing), 0644))

	require.NoError(t, installDiscoveryGroundingSection(tmp, cfg, io.Discard))

	want := "# Agent Rules <!-- tessl-managed -->\n" +
		"\n" +
		"@.tessl/RULES.md\n" +
		"\n" +
		discoveryGroundingRenderedDefault +
		"\n" +
		"## Other Section\n" +
		"\n" +
		"unrelated content preserved verbatim\n"
	got, err := os.ReadFile(filepath.Join(tmp, "AGENTS.md"))
	require.NoError(t, err)
	require.Equal(t, want, string(got))
}

func TestInstallDiscoveryGroundingSection_PicksUpTemplateChange(t *testing.T) {
	tmp := t.TempDir()

	require.NoError(t, installDiscoveryGroundingSection(tmp, config.Config{Command: "go run ."}, io.Discard))
	require.NoError(t, installDiscoveryGroundingSection(tmp, config.Config{Command: "spektacular"}, io.Discard))

	got, err := os.ReadFile(filepath.Join(tmp, "AGENTS.md"))
	require.NoError(t, err)
	require.Contains(t, string(got), "> Managed by `spektacular init`")
	require.NotContains(t, string(got), "go run . init")
}
