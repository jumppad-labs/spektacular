// Package runner defines the Runner interface and shared types for agent backends.
package runner

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/jumppad-labs/spektacular/internal/config"
)

var questionPattern = regexp.MustCompile(`<!--QUESTION:([\s\S]*?)-->`)

// Runner is the interface that all agent backends must implement.
type Runner interface {
	// Run starts the agent with the given options and returns a channel of
	// events and an error channel. The caller must drain both channels;
	// the event channel is closed when the agent finishes.
	Run(opts RunOptions) (<-chan Event, <-chan error)
}

// Event is a single parsed event from an agent's output stream.
type Event struct {
	Type string
	Data map[string]any
}

// SessionID returns the session_id field if present.
func (e Event) SessionID() string {
	v, _ := e.Data["session_id"].(string)
	return v
}

// IsResult reports whether this is a terminal result event.
func (e Event) IsResult() bool { return e.Type == "result" }

// IsError reports whether this is an error result.
func (e Event) IsError() bool {
	if !e.IsResult() {
		return false
	}
	v, _ := e.Data["is_error"].(bool)
	return v
}

// ResultText returns the result text from a result event, or empty string.
func (e Event) ResultText() string {
	if !e.IsResult() {
		return ""
	}
	v, _ := e.Data["result"].(string)
	return v
}

// TextContent extracts concatenated text blocks from an assistant event.
func (e Event) TextContent() string {
	if e.Type != "assistant" {
		return ""
	}
	msg, _ := e.Data["message"].(map[string]any)
	content, _ := msg["content"].([]any)
	var texts []string
	for _, item := range content {
		block, _ := item.(map[string]any)
		if block["type"] == "text" {
			if t, ok := block["text"].(string); ok {
				texts = append(texts, t)
			}
		}
	}
	return strings.Join(texts, "\n")
}

// ToolUses extracts tool_use blocks from an assistant event.
func (e Event) ToolUses() []map[string]any {
	if e.Type != "assistant" {
		return nil
	}
	msg, _ := e.Data["message"].(map[string]any)
	content, _ := msg["content"].([]any)
	var tools []map[string]any
	for _, item := range content {
		block, _ := item.(map[string]any)
		if block["type"] == "tool_use" {
			tools = append(tools, block)
		}
	}
	return tools
}

// Question is a structured question detected in agent output.
type Question struct {
	Question string
	Header   string
	Options  []map[string]any
}

// detectQuestions finds <!--QUESTION:{...}--> markers in text and returns parsed questions.
func detectQuestions(text string) []Question {
	var questions []Question
	for _, match := range questionPattern.FindAllStringSubmatch(text, -1) {
		var payload struct {
			Questions []struct {
				Question string           `json:"question"`
				Header   string           `json:"header"`
				Options  []map[string]any `json:"options"`
			} `json:"questions"`
		}
		if err := json.Unmarshal([]byte(match[1]), &payload); err != nil {
			continue
		}
		for _, q := range payload.Questions {
			questions = append(questions, Question{
				Question: q.Question,
				Header:   q.Header,
				Options:  q.Options,
			})
		}
	}
	return questions
}

// DetectQuestions is the exported wrapper used by other packages.
func DetectQuestions(text string) []Question { return detectQuestions(text) }

// BuildPrompt assembles the user prompt: knowledge hint + spec content.
func BuildPrompt(specContent string) string {
	return BuildPromptWithHeader(specContent, "Specification to Plan")
}

// BuildPromptWithHeader assembles the user prompt with a custom content section header.
func BuildPromptWithHeader(content string, header string) string {
	var b strings.Builder
	b.WriteString("Additional project knowledge, architectural context, and past learnings can be found in `.spektacular/knowledge/`. Use your available tools to explore this directory as needed.\n\n")
	fmt.Fprintf(&b, "---\n\n# %s\n\n%s", header, content)
	return b.String()
}

// RunOptions holds parameters for running an agent.
type RunOptions struct {
	Prompt       string
	SystemPrompt string // passed as --system-prompt; use for agent specialization
	Config       config.Config
	SessionID    string
	CWD          string
	Command      string // used only for debug log filename
}

