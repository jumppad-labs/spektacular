// Package runner spawns the claude CLI subprocess and streams parsed events.
package runner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/nicholasjackson/spektacular/internal/config"
)

var questionPattern = regexp.MustCompile(`<!--QUESTION:([\s\S]*?)-->`)

// ClaudeEvent is a single parsed event from the claude stream-JSON output.
type ClaudeEvent struct {
	Type string
	Data map[string]any
}

// SessionID returns the session_id field if present.
func (e ClaudeEvent) SessionID() string {
	v, _ := e.Data["session_id"].(string)
	return v
}

// IsResult reports whether this is a terminal result event.
func (e ClaudeEvent) IsResult() bool { return e.Type == "result" }

// IsError reports whether this is an error result.
func (e ClaudeEvent) IsError() bool {
	if !e.IsResult() {
		return false
	}
	v, _ := e.Data["is_error"].(bool)
	return v
}

// ResultText returns the result text from a result event, or empty string.
func (e ClaudeEvent) ResultText() string {
	if !e.IsResult() {
		return ""
	}
	v, _ := e.Data["result"].(string)
	return v
}

// TextContent extracts concatenated text blocks from an assistant event.
func (e ClaudeEvent) TextContent() string {
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
func (e ClaudeEvent) ToolUses() []map[string]any {
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

// Question is a structured question detected in Claude output.
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

// BuildPrompt assembles the combined prompt: agent instructions + knowledge + spec.
func BuildPrompt(specContent, agentPrompt string, knowledge map[string]string) string {
	var b strings.Builder
	b.WriteString(agentPrompt)
	b.WriteString("\n\n---\n\n# Knowledge Base\n")
	for filename, content := range knowledge {
		fmt.Fprintf(&b, "\n## %s\n%s\n", filename, content)
	}
	fmt.Fprintf(&b, "\n---\n\n# Specification to Plan\n\n%s", specContent)
	return b.String()
}

// RunOptions holds parameters for RunClaude.
type RunOptions struct {
	Prompt    string
	Config    config.Config
	SessionID string
	CWD       string
	Command   string // used only for debug log filename
}

// RunClaude spawns the claude subprocess and returns a channel of events and an error channel.
// The caller must drain both channels; the event channel is closed when the process exits.
func RunClaude(opts RunOptions) (<-chan ClaudeEvent, <-chan error) {
	events := make(chan ClaudeEvent, 64)
	errc := make(chan error, 1)

	go func() {
		defer close(events)
		if err := runClaude(opts, events); err != nil {
			errc <- err
		}
		close(errc)
	}()

	return events, errc
}

func runClaude(opts RunOptions, events chan<- ClaudeEvent) error {
	cfg := opts.Config
	cmd := []string{cfg.Agent.Command, "-p", opts.Prompt}
	cmd = append(cmd, cfg.Agent.Args...)

	if len(cfg.Agent.AllowedTools) > 0 {
		cmd = append(cmd, "--allowedTools", strings.Join(cfg.Agent.AllowedTools, ","))
	}
	if cfg.Agent.DangerouslySkipPermissions {
		cmd = append(cmd, "--dangerously-skip-permissions")
	}
	if opts.SessionID != "" {
		cmd = append(cmd, "--resume", opts.SessionID)
	}

	cwd := opts.CWD
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}
	}

	proc := exec.Command(cmd[0], cmd[1:]...) //nolint:gosec
	proc.Dir = cwd
	proc.Stderr = io.Discard

	stdout, err := proc.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating stdout pipe: %w", err)
	}
	if err := proc.Start(); err != nil {
		return fmt.Errorf("starting claude process: %w", err)
	}

	var debugLog *os.File
	if cfg.Debug.Enabled {
		debugLog = openDebugLog(cfg, opts.Command, cwd)
		if debugLog != nil {
			defer debugLog.Close()
		}
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1 MiB lines
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if debugLog != nil {
			fmt.Fprintln(debugLog, line)
		}
		var data map[string]any
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			continue
		}
		eventType, _ := data["type"].(string)
		events <- ClaudeEvent{Type: eventType, Data: data}
	}

	if err := proc.Wait(); err != nil {
		return fmt.Errorf("claude process exited with error: %w", err)
	}
	return nil
}

func openDebugLog(cfg config.Config, command, cwd string) *os.File {
	logDir := filepath.Join(cwd, cfg.Debug.LogDir)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil
	}
	ts := time.Now().Format("2006-01-02_150405")
	filename := fmt.Sprintf("%s_%s_%s.log", ts, cfg.Agent.Command, command)
	f, err := os.Create(filepath.Join(logDir, filename))
	if err != nil {
		return nil
	}
	return f
}
