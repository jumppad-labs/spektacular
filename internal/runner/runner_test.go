package runner

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// ClaudeEvent property tests
// ---------------------------------------------------------------------------

func TestClaudeEvent_SessionID(t *testing.T) {
	e := ClaudeEvent{Type: "system", Data: map[string]any{"session_id": "sess-123"}}
	require.Equal(t, "sess-123", e.SessionID())
}

func TestClaudeEvent_SessionID_Missing(t *testing.T) {
	e := ClaudeEvent{Type: "system", Data: map[string]any{}}
	require.Equal(t, "", e.SessionID())
}

func TestClaudeEvent_IsResult_True(t *testing.T) {
	e := ClaudeEvent{Type: "result"}
	require.True(t, e.IsResult())
}

func TestClaudeEvent_IsResult_False(t *testing.T) {
	e := ClaudeEvent{Type: "assistant"}
	require.False(t, e.IsResult())
}

func TestClaudeEvent_IsError_True(t *testing.T) {
	e := ClaudeEvent{Type: "result", Data: map[string]any{"is_error": true}}
	require.True(t, e.IsError())
}

func TestClaudeEvent_IsError_False_WhenNotResult(t *testing.T) {
	e := ClaudeEvent{Type: "assistant", Data: map[string]any{"is_error": true}}
	require.False(t, e.IsError())
}

func TestClaudeEvent_ResultText(t *testing.T) {
	e := ClaudeEvent{Type: "result", Data: map[string]any{"result": "plan text"}}
	require.Equal(t, "plan text", e.ResultText())
}

func TestClaudeEvent_ResultText_EmptyWhenNotResult(t *testing.T) {
	e := ClaudeEvent{Type: "assistant", Data: map[string]any{"result": "plan text"}}
	require.Equal(t, "", e.ResultText())
}

func TestClaudeEvent_TextContent_ExtractsTextBlocks(t *testing.T) {
	e := ClaudeEvent{
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

func TestClaudeEvent_TextContent_EmptyWhenNotAssistant(t *testing.T) {
	e := ClaudeEvent{Type: "result"}
	require.Equal(t, "", e.TextContent())
}

func TestClaudeEvent_ToolUses(t *testing.T) {
	e := ClaudeEvent{
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

func TestBuildPrompt_ContainsAllParts(t *testing.T) {
	prompt := BuildPrompt("my spec", "agent instructions", map[string]string{
		"arch.md": "architecture content",
	})
	require.Contains(t, prompt, "agent instructions")
	require.Contains(t, prompt, "Knowledge Base")
	require.Contains(t, prompt, "arch.md")
	require.Contains(t, prompt, "architecture content")
	require.Contains(t, prompt, "my spec")
}

func TestBuildPrompt_NoKnowledge_StillIncludesSpecAndAgent(t *testing.T) {
	prompt := BuildPrompt("spec text", "agent text", nil)
	require.Contains(t, prompt, "agent text")
	require.Contains(t, prompt, "spec text")
}
