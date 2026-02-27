// Package tui provides the Bubble Tea TUI for the plan command.
package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/nicholasjackson/spektacular/internal/config"
	"github.com/nicholasjackson/spektacular/internal/plan"
	"github.com/nicholasjackson/spektacular/internal/runner"
)

// ---------------------------------------------------------------------------
// Tea messages
// ---------------------------------------------------------------------------

// agentEventMsg carries one parsed event plus the still-open channels so the
// next waitForEvent call can continue reading without storing channels in the
// model (which breaks Bubble Tea's copy-on-update semantics).
type agentEventMsg struct {
	event  runner.ClaudeEvent
	events <-chan runner.ClaudeEvent
	errc   <-chan error
}

type agentDoneMsg struct{}           // runner finished cleanly
type agentErrMsg struct{ err error } // runner returned an error

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

type model struct {
	// layout
	width, height int
	ready         bool
	vp            viewport.Model

	// content — []string avoids strings.Builder copy-after-write panic
	content   []string // accumulated rendered output
	toolLine  string   // current tool status (hidden when "")
	questions []runner.Question
	answers   []string

	// state
	themeIdx   int
	followMode bool
	done       bool
	statusText string

	// result
	resultDir string
	errMsg    string

	// agent context (read-only after init, safe to copy)
	specPath    string
	projectPath string
	cfg         config.Config
	sessionID   string
}

func initialModel(specPath, projectPath string, cfg config.Config) model {
	return model{
		specPath:    specPath,
		projectPath: projectPath,
		cfg:         cfg,
		themeIdx:    0, // dracula
		followMode:  true,
		statusText:  "* thinking  " + filepath.Base(specPath),
	}
}

// ---------------------------------------------------------------------------
// Init
// ---------------------------------------------------------------------------

func (m model) Init() tea.Cmd {
	return startAgentCmd(m.specPath, m.projectPath, m.cfg, "")
}

// startAgentCmd builds the prompt and spawns the runner, returning the first
// event (or error) as a message. Channels flow forward via agentEventMsg.
func startAgentCmd(specPath, projectPath string, cfg config.Config, sessionID string) tea.Cmd {
	return func() tea.Msg {
		specContent, err := os.ReadFile(specPath)
		if err != nil {
			return agentErrMsg{err: fmt.Errorf("reading spec: %w", err)}
		}
		agentPrompt := plan.LoadAgentPrompt()
		knowledge := plan.LoadKnowledge(projectPath)
		prompt := runner.BuildPrompt(string(specContent), agentPrompt, knowledge)

		if cfg.Debug.Enabled {
			planDir := filepath.Join(projectPath, ".spektacular", "plans", stripExt(filepath.Base(specPath)))
			_ = os.MkdirAll(planDir, 0755)
			_ = os.WriteFile(filepath.Join(planDir, "prompt.md"), []byte(prompt), 0644)
		}

		events, errc := runner.RunClaude(runner.RunOptions{
			Prompt:    prompt,
			Config:    cfg,
			SessionID: sessionID,
			CWD:       projectPath,
			Command:   "plan",
		})
		return readNext(events, errc)
	}
}

// resumeAgentCmd starts a new runner turn with the user's answer.
func resumeAgentCmd(cfg config.Config, sessionID, projectPath, answer string) tea.Cmd {
	return func() tea.Msg {
		events, errc := runner.RunClaude(runner.RunOptions{
			Prompt:    answer,
			Config:    cfg,
			SessionID: sessionID,
			CWD:       projectPath,
			Command:   "plan",
		})
		return readNext(events, errc)
	}
}

// waitForEvent returns a Cmd that reads the NEXT event from already-open channels.
func waitForEvent(events <-chan runner.ClaudeEvent, errc <-chan error) tea.Cmd {
	return func() tea.Msg { return readNext(events, errc) }
}

// readNext reads one event from the channel and returns the appropriate message.
// Channels are embedded in agentEventMsg so they propagate without model storage.
func readNext(events <-chan runner.ClaudeEvent, errc <-chan error) tea.Msg {
	event, ok := <-events
	if !ok {
		select {
		case err := <-errc:
			if err != nil {
				return agentErrMsg{err: err}
			}
		default:
		}
		return agentDoneMsg{}
	}
	return agentEventMsg{event: event, events: events, errc: errc}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.vp = viewport.New(msg.Width, m.viewportHeight())
			m.vp.SetContent(strings.Join(m.content, ""))
			m.ready = true
		} else {
			m.vp.Width = msg.Width
			m.vp.Height = m.viewportHeight()
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case agentEventMsg:
		return m.handleAgentEvent(msg)

	case agentDoneMsg:
		// channel closed without a result event — unexpected but non-fatal
		return m, nil

	case agentErrMsg:
		p := m.currentPalette()
		errLine := lipgloss.NewStyle().Foreground(p.errColor).Render("• " + msg.err.Error())
		m = m.withLine(errLine + "\n")
		m.errMsg = msg.err.Error()
		m.done = true
		m.statusText = "error  press q to exit"
		return m, nil
	}

	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "Q":
		if m.done || len(m.questions) == 0 {
			return m, tea.Quit
		}

	case "ctrl+c":
		return m, tea.Quit

	case "t", "T":
		m.themeIdx = (m.themeIdx + 1) % len(themeOrder)
		m.statusText = fmt.Sprintf("theme: %s  (t to cycle)", themeOrder[m.themeIdx])
		return m, nil

	case "f", "F":
		m.followMode = true
		if m.ready {
			m.vp.GotoBottom()
		}
		return m, nil

	case "up", "k":
		m.followMode = false

	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		return m.handleNumberKey(msg.String())
	}

	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func (m model) handleNumberKey(key string) (tea.Model, tea.Cmd) {
	if len(m.questions) == 0 {
		return m, nil
	}
	idx := int(key[0] - '1')
	q := m.questions[0]
	if idx < 0 || idx >= len(q.Options) {
		return m, nil
	}

	label, _ := q.Options[idx]["label"].(string)
	m.answers = append(m.answers, label)
	m.questions = m.questions[1:]

	p := m.currentPalette()
	m = m.withLine(lipgloss.NewStyle().Foreground(p.answer).Render("> "+label) + "\n")

	if len(m.questions) > 0 {
		return m, nil
	}

	answer := strings.Join(m.answers, "\n")
	m.answers = nil
	m.statusText = "* thinking  " + filepath.Base(m.specPath)
	return m, resumeAgentCmd(m.cfg, m.sessionID, m.projectPath, answer)
}

// handleAgentEvent processes one event from the runner.
// It always receives the full agentEventMsg so it can return the next waitForEvent
// command with the correct channel references.
func (m model) handleAgentEvent(msg agentEventMsg) (tea.Model, tea.Cmd) {
	event := msg.event

	if id := event.SessionID(); id != "" {
		m.sessionID = id
	}

	// Tool use — update status line
	for _, tool := range event.ToolUses() {
		name, _ := tool["name"].(string)
		input, _ := tool["input"].(map[string]any)
		m.toolLine = toolDescription(name, input)
	}

	// Text content — render and append
	if text := event.TextContent(); text != "" {
		m.toolLine = ""
		rendered := m.renderMarkdown(text)
		p := m.currentPalette()
		bullet := lipgloss.NewStyle().Foreground(p.output).Render("•")
		m = m.withLine(bulletPrefix(bullet, rendered) + "\n")
		m.questions = append(m.questions, runner.DetectQuestions(text)...)
	}

	// Result event — terminal
	if event.IsResult() {
		m.toolLine = ""
		if event.IsError() {
			m.errMsg = event.ResultText()
			m.done = true
			m.statusText = "error  press q to exit"
			p := m.currentPalette()
			m = m.withLine(lipgloss.NewStyle().Foreground(p.errColor).Render("• "+m.errMsg) + "\n")
			return m, nil
		}

		specName := stripExt(filepath.Base(m.specPath))
		planDir := filepath.Join(m.projectPath, ".spektacular", "plans", specName)
		if err := plan.WritePlanOutput(planDir, event.ResultText()); err != nil {
			m.errMsg = err.Error()
			m.done = true
			m.statusText = "error  press q to exit"
			return m, nil
		}
		m.resultDir = planDir
		m.done = true
		m.statusText = "done  press q to exit"
		p := m.currentPalette()
		m = m.withLine(lipgloss.NewStyle().Foreground(p.success).Render(
			fmt.Sprintf("• plan written to %s/plan.md", planDir),
		) + "\n")
		return m, nil
	}

	// Not a result — keep reading
	return m, waitForEvent(msg.events, msg.errc)
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (m model) View() string {
	if !m.ready {
		return "initializing…\n"
	}

	p := m.currentPalette()
	var sections []string

	sections = append(sections, m.vp.View())

	if m.toolLine != "" {
		toolStyle := lipgloss.NewStyle().
			Background(p.bg).
			Foreground(p.answer).
			Width(m.width)
		sections = append(sections, toolStyle.Render("⚙ "+m.toolLine))
	}

	if len(m.questions) > 0 {
		sections = append(sections, m.renderQuestionPanel(p))
	}

	statusStyle := lipgloss.NewStyle().
		Background(p.bg).
		Foreground(p.faint).
		Width(m.width)
	followHint := "f: enable follow"
	if m.followMode {
		followHint = "f: disable follow"
	}
	sections = append(sections, statusStyle.Render(fmt.Sprintf("%s  %s", m.statusText, followHint)))

	return strings.Join(sections, "\n")
}

func (m model) renderQuestionPanel(p palette) string {
	q := m.questions[0]

	borderStyle := lipgloss.NewStyle().
		BorderTop(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(p.answer).
		Padding(0, 1)

	headerStyle := lipgloss.NewStyle().Bold(true)
	numStyle := lipgloss.NewStyle().Foreground(p.question).Bold(true)
	faintStyle := lipgloss.NewStyle().Foreground(p.faint)

	var lines []string
	lines = append(lines, headerStyle.Render(q.Header)+": "+q.Question)
	for i, opt := range q.Options {
		label, _ := opt["label"].(string)
		desc, _ := opt["description"].(string)
		line := fmt.Sprintf("  %s  %s", numStyle.Render(fmt.Sprintf("%d", i+1)), label)
		if desc != "" {
			line += "  " + faintStyle.Render("— "+desc)
		}
		lines = append(lines, line)
	}
	lines = append(lines, faintStyle.Render("press a number to select"))

	return borderStyle.Render(strings.Join(lines, "\n"))
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// bulletPrefix places bullet on the first non-empty line of rendered and
// indents all subsequent lines by 2 spaces to keep them aligned.
func bulletPrefix(bullet, rendered string) string {
	trimmed := strings.TrimSpace(rendered)
	if trimmed == "" {
		return bullet
	}
	lines := strings.Split(trimmed, "\n")
	lines[0] = bullet + " " + lines[0]
	for i := 1; i < len(lines); i++ {
		lines[i] = "  " + lines[i]
	}
	return strings.Join(lines, "\n")
}

// withLine appends s to the content buffer and refreshes the viewport.
// Returns the updated model value (safe to call from value-receiver methods).
func (m model) withLine(s string) model {
	m.content = append(m.content, s)
	if m.ready {
		m.vp.SetContent(strings.Join(m.content, ""))
		if m.followMode {
			m.vp.GotoBottom()
		}
	}
	return m
}

func (m model) viewportHeight() int {
	reserved := 1 // status bar
	if m.toolLine != "" {
		reserved++
	}
	if len(m.questions) > 0 {
		reserved += 3 + len(m.questions[0].Options) // border + header + options + hint
	}
	h := m.height - reserved
	if h < 1 {
		h = 1
	}
	return h
}

func (m model) currentPalette() palette {
	name := themeOrder[m.themeIdx]
	if p, ok := palettes[name]; ok {
		return p
	}
	return palettes["dracula"]
}

func (m model) renderMarkdown(text string) string {
	width := m.width - 2 // leave room for "• " prefix
	if width < 20 {
		width = 80
	}
	style := glamourStyle(themeOrder[m.themeIdx])
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(style),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return text
	}
	out, err := r.Render(text)
	if err != nil {
		return text
	}
	return strings.TrimRight(out, "\n")
}

var toolInputKeys = map[string]string{
	"Bash":      "command",
	"Read":      "file_path",
	"Write":     "file_path",
	"Edit":      "file_path",
	"Glob":      "pattern",
	"Grep":      "pattern",
	"WebFetch":  "url",
	"Task":      "description",
	"WebSearch": "query",
}

func toolDescription(name string, input map[string]any) string {
	key := toolInputKeys[name]
	var val string
	if key != "" {
		val = fmt.Sprintf("%v", input[key])
	} else if len(input) > 0 {
		for _, v := range input {
			val = fmt.Sprintf("%v", v)
			break
		}
	}
	if len(val) > 100 {
		val = val[:100] + "…"
	}
	return fmt.Sprintf("%s  %s", name, val)
}

func stripExt(name string) string {
	ext := filepath.Ext(name)
	if ext == "" {
		return name
	}
	return name[:len(name)-len(ext)]
}

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

// RunPlanTUI launches the interactive TUI for plan generation.
// Returns the plan directory path on success, or empty string if the user quit early.
func RunPlanTUI(specPath, projectPath string, cfg config.Config) (string, error) {
	m := initialModel(specPath, projectPath, cfg)

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	fm := finalModel.(model)
	if fm.errMsg != "" {
		return "", fmt.Errorf("%s", fm.errMsg)
	}
	return fm.resultDir, nil
}
