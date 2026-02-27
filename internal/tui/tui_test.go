package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/nicholasjackson/spektacular/internal/config"
	"github.com/nicholasjackson/spektacular/internal/runner"
	"github.com/stretchr/testify/require"
)

func TestToolDescription_KnownTool(t *testing.T) {
	desc := toolDescription("Bash", map[string]any{"command": "ls -la"})
	require.Contains(t, desc, "Bash")
	require.Contains(t, desc, "ls -la")
}

func TestToolDescription_UnknownTool_FirstValue(t *testing.T) {
	desc := toolDescription("Custom", map[string]any{"arg": "value"})
	require.Contains(t, desc, "Custom")
	require.Contains(t, desc, "value")
}

func TestToolDescription_LongValue_Truncated(t *testing.T) {
	long := strings.Repeat("a", 200)
	desc := toolDescription("Bash", map[string]any{"command": long})
	require.LessOrEqual(t, len(desc), 200)
	require.Contains(t, desc, "…")
}

func TestStripExt_RemovesExtension(t *testing.T) {
	require.Equal(t, "my-spec", stripExt("my-spec.md"))
}

func TestStripExt_NoExtension_Unchanged(t *testing.T) {
	require.Equal(t, "my-spec", stripExt("my-spec"))
}

func TestGlamourStyle_Dracula(t *testing.T) {
	require.Equal(t, "dracula", glamourStyle("dracula"))
}

func TestGlamourStyle_Other_ReturnsDark(t *testing.T) {
	require.Equal(t, "dark", glamourStyle("nord"))
	require.Equal(t, "dark", glamourStyle("github-dark"))
}

func TestCurrentPalette_DefaultIsDracula(t *testing.T) {
	m := initialModel("spec.md", "/tmp", config.NewDefault())
	p := m.currentPalette()
	require.Equal(t, palettes["dracula"], p)
}

func TestThemeCycling_AdvancesIndex(t *testing.T) {
	m := initialModel("spec.md", "/tmp", config.NewDefault())
	initial := themeOrder[m.themeIdx]
	m.themeIdx = (m.themeIdx + 1) % len(themeOrder)
	next := themeOrder[m.themeIdx]
	require.NotEqual(t, initial, next)
}

func TestBulletPrefix_SingleLine(t *testing.T) {
	result := bulletPrefix("•", "Hello world")
	require.Equal(t, "• Hello world", result)
}

func TestBulletPrefix_MultiLine_IndentsSubsequentLines(t *testing.T) {
	result := bulletPrefix("•", "line one\nline two\nline three")
	lines := strings.Split(result, "\n")
	require.Equal(t, "• line one", lines[0])
	require.Equal(t, "  line two", lines[1])
	require.Equal(t, "  line three", lines[2])
}

func TestBulletPrefix_LeadingTrailingWhitespaceStripped(t *testing.T) {
	result := bulletPrefix("•", "\n\n  hello\n\n")
	require.Equal(t, "• hello", result)
}

func TestBulletPrefix_EmptyRendered(t *testing.T) {
	result := bulletPrefix("•", "   ")
	require.Equal(t, "•", result)
}

func TestWithLine_AccumulatesContent(t *testing.T) {
	m := initialModel("spec.md", "/tmp", config.NewDefault())
	m = m.withLine("line one\n")
	m = m.withLine("line two\n")
	require.Len(t, m.content, 2)
	require.Equal(t, "line one\n", m.content[0])
	require.Equal(t, "line two\n", m.content[1])
}

func TestWithLine_IsSafeToCopy(t *testing.T) {
	m := initialModel("spec.md", "/tmp", config.NewDefault())
	m = m.withLine("first\n")
	// Copy the model (simulates Bubble Tea's Update pattern) and write again
	m2 := m
	m2 = m2.withLine("second\n")
	// Original should be unchanged
	require.Len(t, m.content, 1)
	require.Len(t, m2.content, 2)
}

func TestReadNext_ClosedChannel_ReturnsDoneMsg(t *testing.T) {
	events := make(chan runner.ClaudeEvent)
	errc := make(chan error, 1)
	close(events)
	msg := readNext(events, errc)
	_, ok := msg.(agentDoneMsg)
	require.True(t, ok)
}

func TestReadNext_ClosedChannelWithError_ReturnsErrMsg(t *testing.T) {
	events := make(chan runner.ClaudeEvent)
	errc := make(chan error, 1)
	errc <- fmt.Errorf("runner failed")
	close(events)
	msg := readNext(events, errc)
	errMsg, ok := msg.(agentErrMsg)
	require.True(t, ok)
	require.Contains(t, errMsg.err.Error(), "runner failed")
}

func TestReadNext_OpenChannel_ReturnsEventMsg(t *testing.T) {
	events := make(chan runner.ClaudeEvent, 1)
	errc := make(chan error, 1)
	events <- runner.ClaudeEvent{Type: "assistant"}
	msg := readNext(events, errc)
	evMsg, ok := msg.(agentEventMsg)
	require.True(t, ok)
	require.Equal(t, "assistant", evMsg.event.Type)
	// Channels are embedded in the message so the next read can continue
	require.NotNil(t, evMsg.events)
	require.NotNil(t, evMsg.errc)
}
