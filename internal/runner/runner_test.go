package runner

import (
	"testing"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Event property tests
// ---------------------------------------------------------------------------

func TestEvent_SessionID(t *testing.T) {
	e := Event{Type: "system", Data: map[string]any{"session_id": "sess-123"}}
	require.Equal(t, "sess-123", e.SessionID())
}

func TestEvent_SessionID_Missing(t *testing.T) {
	e := Event{Type: "system", Data: map[string]any{}}
	require.Equal(t, "", e.SessionID())
}

func TestEvent_IsResult_True(t *testing.T) {
	e := Event{Type: "result"}
	require.True(t, e.IsResult())
}

func TestEvent_IsResult_False(t *testing.T) {
	e := Event{Type: "assistant"}
	require.False(t, e.IsResult())
}

func TestEvent_IsError_True(t *testing.T) {
	e := Event{Type: "result", Data: map[string]any{"is_error": true}}
	require.True(t, e.IsError())
}

func TestEvent_IsError_False_WhenNotResult(t *testing.T) {
	e := Event{Type: "assistant", Data: map[string]any{"is_error": true}}
	require.False(t, e.IsError())
}

func TestEvent_ResultText(t *testing.T) {
	e := Event{Type: "result", Data: map[string]any{"result": "plan text"}}
	require.Equal(t, "plan text", e.ResultText())
}

func TestEvent_ResultText_EmptyWhenNotResult(t *testing.T) {
	e := Event{Type: "assistant", Data: map[string]any{"result": "plan text"}}
	require.Equal(t, "", e.ResultText())
}

func TestEvent_TextContent_ExtractsTextBlocks(t *testing.T) {
	e := Event{
		Type: "assistant",
		Data: map[string]any{
			"message": map[string]any{
				"content": []any{
					map[string]any{"type": "text", "text": "hello"},
					map[string]any{"type": "tool_use", "name": "Bash"},
					map[string]any{"type": "text", "text": " world"},
				},
			},
		},
	}
	require.Equal(t, "hello\n world", e.TextContent())
}

func TestEvent_TextContent_EmptyWhenNotAssistant(t *testing.T) {
	e := Event{Type: "result"}
	require.Equal(t, "", e.TextContent())
}

func TestEvent_ToolUses(t *testing.T) {
	e := Event{
		Type: "assistant",
		Data: map[string]any{
			"message": map[string]any{
				"content": []any{
					map[string]any{"type": "tool_use", "name": "Bash", "input": map[string]any{"command": "ls"}},
					map[string]any{"type": "text", "text": "output"},
				},
			},
		},
	}
	tools := e.ToolUses()
	require.Len(t, tools, 1)
	require.Equal(t, "Bash", tools[0]["name"])
}

// ---------------------------------------------------------------------------
// detectQuestions tests
// ---------------------------------------------------------------------------

func TestDetectQuestions_FindsQuestion(t *testing.T) {
	text := `some text <!--QUESTION:{"questions":[{"question":"Which approach?","header":"Approach","options":[{"label":"A"},{"label":"B"}]}]}--> more`
	questions := detectQuestions(text)
	require.Len(t, questions, 1)
	require.Equal(t, "Which approach?", questions[0].Question)
	require.Equal(t, "Approach", questions[0].Header)
	require.Len(t, questions[0].Options, 2)
}

func TestDetectQuestions_NoMarker_ReturnsEmpty(t *testing.T) {
	questions := detectQuestions("no markers here")
	require.Empty(t, questions)
}

func TestDetectQuestions_InvalidJSON_Skipped(t *testing.T) {
	text := `<!--QUESTION:not-valid-json-->`
	questions := detectQuestions(text)
	require.Empty(t, questions)
}

func TestDetectQuestions_MultilineMarker(t *testing.T) {
	text := "<!--QUESTION:{\"questions\":[{\"question\":\"Q?\",\"header\":\"H\",\"options\":[{\"label\":\"X\"}]}]}-->"
	questions := detectQuestions(text)
	require.Len(t, questions, 1)
}

// ---------------------------------------------------------------------------
// buildPrompt tests
// ---------------------------------------------------------------------------

func TestBuildPrompt_ContainsSpecAndKnowledgeHint(t *testing.T) {
	prompt := BuildPrompt("my spec")
	require.Contains(t, prompt, "my spec")
	require.Contains(t, prompt, ".spektacular/knowledge/")
}

func TestBuildPromptWithHeader_UsesCustomHeader(t *testing.T) {
	prompt := BuildPromptWithHeader("plan content", "Implementation Plan")
	require.Contains(t, prompt, "# Implementation Plan")
	require.Contains(t, prompt, "plan content")
	require.NotContains(t, prompt, "Specification to Plan")
}

// ---------------------------------------------------------------------------
// NewRunner factory tests
// ---------------------------------------------------------------------------

func TestNewRunner_ReturnsErrorForUnknownCommand(t *testing.T) {
	cfg := config.Config{Agent: config.AgentConfig{Command: "unknown-agent"}}
	r, err := NewRunner(cfg)
	require.Error(t, err)
	require.Nil(t, r)
	require.Contains(t, err.Error(), "unsupported runner")
	require.Contains(t, err.Error(), "unknown-agent")
}

func TestNewRunner_ReturnsRunnerForRegisteredCommand(t *testing.T) {
	// Register a test runner.
	Register("test-runner", func() Runner {
		return &stubRunner{}
	})
	defer func() {
		delete(registry, "test-runner")
	}()

	cfg := config.Config{Agent: config.AgentConfig{Command: "test-runner"}}
	r, err := NewRunner(cfg)
	require.NoError(t, err)
	require.NotNil(t, r)
}

// stubRunner is a minimal runner for testing the registry.
type stubRunner struct{}

func (s *stubRunner) Run(_ RunOptions) (<-chan Event, <-chan error) {
	events := make(chan Event)
	errc := make(chan error)
	close(events)
	close(errc)
	return events, errc
}
