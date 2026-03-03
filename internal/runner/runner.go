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

// QuestionType controls how the TUI renders a question.
// "text" shows a free-text textarea. "choice" shows numbered options with an automatic "Other" entry.
// Defaults to "text" when not specified or when no options are provided.
type QuestionType string

const (
	QuestionTypeText   QuestionType = "text"
	QuestionTypeChoice QuestionType = "choice"
)

// Question is a structured question detected in agent output.
type Question struct {
	Question string
	Header   string
	Type     QuestionType
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
				Type     string           `json:"type"`
				Options  []map[string]any `json:"options"`
			} `json:"questions"`
		}
		if err := json.Unmarshal([]byte(match[1]), &payload); err != nil {
			continue
		}
		for _, q := range payload.Questions {
			qt := QuestionTypeText
			if q.Type == string(QuestionTypeChoice) && len(q.Options) > 0 {
				qt = QuestionTypeChoice
			}
			questions = append(questions, Question{
				Question: q.Question,
				Header:   q.Header,
				Type:     qt,
				Options:  q.Options,
			})
		}
	}
	return questions
}

// DetectQuestions is the exported wrapper used by other packages.
func DetectQuestions(text string) []Question { return detectQuestions(text) }

var finishedPattern = regexp.MustCompile(`<!--\s*FINISHED\s*-->`)
var gotoPattern = regexp.MustCompile(`<!--\s*GOTO:\s*([\w][\w\s-]*?)\s*-->`)

// DetectFinished reports whether the agent output contains a <!-- FINISHED --> marker.
func DetectFinished(text string) bool {
	return finishedPattern.MatchString(text)
}

// DetectGoto returns the target step name if the agent output contains a
// <!-- GOTO: step name --> marker, and whether one was found.
func DetectGoto(text string) (string, bool) {
	m := gotoPattern.FindStringSubmatch(text)
	if m == nil {
		return "", false
	}
	return strings.TrimSpace(m[1]), true
}

// StripFinishedTag removes the <!-- FINISHED --> marker from text before display.
func StripFinishedTag(text string) string {
	return strings.TrimSpace(finishedPattern.ReplaceAllString(text, ""))
}

// StripMarkers removes <!-- FINISHED -->, <!-- GOTO:... -->, and <!--QUESTION:...--> markers
// from text before display.
func StripMarkers(text string) string {
	text = finishedPattern.ReplaceAllString(text, "")
	text = gotoPattern.ReplaceAllString(text, "")
	text = questionPattern.ReplaceAllString(text, "")
	return strings.TrimSpace(text)
}

// Prompts bundles the user prompt and system prompt for an agent invocation.
type Prompts struct {
	User   string // initial user message
	System string // system prompt; empty uses the agent's default
}

// Step defines one agent step in a multi-step pipeline.
type Step struct {
	Prompts Prompts
	LogFile string // path to debug log file; empty disables logging
}

// RunSteps executes a sequence of Steps in order. Within each step, questions are answered
// by calling onQuestion and the session is resumed. Steps advance on <!-- FINISHED --> or
// on a natural result event. Returns an error if any step fails.
func RunSteps(
	r Runner,
	steps []Step,
	cfg config.Config,
	cwd string,
	onText func(string),
	onQuestion func([]Question) string,
) error {
	for _, step := range steps {
		if err := runStep(r, step, cfg, cwd, onText, onQuestion); err != nil {
			return err
		}
	}
	return nil
}

func runStep(
	r Runner,
	step Step,
	cfg config.Config,
	cwd string,
	onText func(string),
	onQuestion func([]Question) string,
) error {
	sessionID := ""
	currentUser := step.Prompts.User

	for {
		var questionsFound []Question
		var stepDone bool

		events, errc := r.Run(RunOptions{
			Prompts:   Prompts{User: currentUser, System: step.Prompts.System},
			Config:    cfg,
			SessionID: sessionID,
			CWD:       cwd,
			LogFile:   step.LogFile,
		})

		for event := range events {
			if id := event.SessionID(); id != "" {
				sessionID = id
			}
			if text := event.TextContent(); text != "" {
				if DetectFinished(text) {
					stepDone = true
				}
				displayText := StripFinishedTag(text)
				if onText != nil && displayText != "" {
					onText(displayText)
				}
				questionsFound = append(questionsFound, DetectQuestions(text)...)
			}
			if event.IsResult() {
				if event.IsError() {
					return fmt.Errorf("agent error: %s", event.ResultText())
				}
				stepDone = true
			}
		}

		if err := <-errc; err != nil {
			return fmt.Errorf("runner error: %w", err)
		}

		if !stepDone && len(questionsFound) > 0 && onQuestion != nil {
			answer := onQuestion(questionsFound)
			currentUser = answer
			continue
		}

		return nil
	}
}

// BuildPrompt assembles the user prompt: knowledge hint + spec content.
func BuildPrompt(specContent string) string {
	return BuildPromptWithHeader(specContent, "Specification to Plan")
}

// BuildPlanPrompt assembles the user prompt for the planner, including the exact
// plan directory the agent must write its output files into.
func BuildPlanPrompt(specContent, planDir string) string {
	var b strings.Builder
	b.WriteString("Additional project knowledge, architectural context, and past learnings can be found in `.spektacular/knowledge/`. Use your available tools to explore this directory as needed.\n\n")
	fmt.Fprintf(&b, "Write all plan output files to this exact directory: `%s`\n\n", planDir)
	fmt.Fprintf(&b, "---\n\n# Specification to Plan\n\n%s", specContent)
	return b.String()
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
	Prompts   Prompts
	Config    config.Config
	SessionID string
	CWD       string
	LogFile   string // path to debug log file; empty disables logging
	Model     string // model override; empty uses the agent default
}

