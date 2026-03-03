// Package tui provides the Bubble Tea TUI for the plan command.
package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/runner"
)

// ---------------------------------------------------------------------------
// Tea messages
// ---------------------------------------------------------------------------

// agentEventMsg carries one parsed event plus the still-open channels so the
// next waitForEvent call can continue reading without storing channels in the
// model (which breaks Bubble Tea's copy-on-update semantics).
type agentEventMsg struct {
	event  runner.Event
	events <-chan runner.Event
	errc   <-chan error
}

type agentDoneMsg struct{}           // runner finished cleanly
type agentErrMsg struct{ err error } // runner returned an error

// ---------------------------------------------------------------------------
// Workflow
// ---------------------------------------------------------------------------

// WorkflowStep defines one step in a multi-step TUI workflow.
// BuildRunOptions is called at step start to produce the runner options.
// The TUI handles all BubbleTea machinery; callers only supply data.
type WorkflowStep struct {
	Name            string // unique step name; used as target for <!-- GOTO: name -->
	StatusLabel     string
	BuildRunOptions func(cfg config.Config, cwd string) (runner.RunOptions, error)
}

// Workflow defines a multi-step agent pipeline for the TUI.
// Steps are executed in order; OnDone is called after the last step completes.
// LogFile is the debug log path for the whole workflow run; empty disables logging.
// Preamble is optional markdown text displayed in the viewport before the first step runs.
type Workflow struct {
	LogFile  string
	Preamble string
	Steps    []WorkflowStep
	OnDone   func() (string, error)
}

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

	// free-text input state (when user selects "Other")
	otherInput bool
	otherText  string

	// Enhanced text input state
	textareaActive bool           // True when textarea has focus
	textarea       textarea.Model // Multi-line text input component

	// state
	themeIdx    int
	followMode  bool
	detailMode  bool
	done        bool
	statusText  string
	currentStep int // index into workflow.Steps

	// result
	resultDir string
	errMsg    string

	// agent context (read-only after init, safe to copy)
	workflow    Workflow
	projectPath string
	cfg         config.Config
	sessionID   string
	logFile     string // path to debug log for the current step; empty disables logging
}

func initialModel(wf Workflow, projectPath string, cfg config.Config) model {
	label := ""
	if len(wf.Steps) > 0 {
		label = wf.Steps[0].StatusLabel
	}
	return model{
		workflow:    wf,
		projectPath: projectPath,
		cfg:         cfg,
		themeIdx:    0, // dracula
		followMode:  true,
		statusText:  "* thinking  " + label,
		logFile:     wf.LogFile,
	}
}

// currentStepLabel returns the StatusLabel of the active step.
func (m model) currentStepLabel() string {
	if m.currentStep < len(m.workflow.Steps) {
		return m.workflow.Steps[m.currentStep].StatusLabel
	}
	return ""
}

// initTextarea creates and configures a new textarea
func (m *model) initTextarea(placeholder string) {
	ta := textarea.New()
	ta.Placeholder = placeholder
	ta.Focus()
	ta.CharLimit = 10000 // Reasonable limit for spec sections
	ta.SetWidth(m.width - 4)  // Leave room for borders
	ta.SetHeight(10)          // Default height, adjustable

	// Style
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false

	m.textarea = ta
	m.textareaActive = true
}

// ---------------------------------------------------------------------------
// Init
// ---------------------------------------------------------------------------

func (m model) Init() tea.Cmd {
	return m.startCurrentStep()
}

// startCurrentStep builds a tea.Cmd that starts the current workflow step.
func (m model) startCurrentStep() tea.Cmd {
	if m.currentStep >= len(m.workflow.Steps) {
		return nil
	}
	step := m.workflow.Steps[m.currentStep]
	logFile := m.logFile
	sessionID := m.sessionID // carry session forward so the model retains context
	return func() tea.Msg {
		opts, err := step.BuildRunOptions(m.cfg, m.projectPath)
		if err != nil {
			return agentErrMsg{err: fmt.Errorf("building run options: %w", err)}
		}
		opts.LogFile = logFile
		opts.SessionID = sessionID
		r, err := runner.NewRunner(m.cfg)
		if err != nil {
			return agentErrMsg{err: fmt.Errorf("creating runner: %w", err)}
		}
		events, errc := r.Run(opts)
		return readNext(events, errc)
	}
}

// gotoStep jumps to the named step, preserving the session ID so the agent resumes in
// the same conversation. If no step with that name exists, it is a no-op.
func (m model) gotoStep(name string) (tea.Model, tea.Cmd) {
	for i, step := range m.workflow.Steps {
		if step.Name == name {
			m.currentStep = i
			m.questions = nil
			m.answers = nil
			m.textareaActive = false
			m.statusText = "* thinking  " + m.currentStepLabel()
			return m, m.startCurrentStep()
		}
	}
	return m, nil
}

// advanceStep moves to the next workflow step, or calls OnDone if all steps are complete.
func (m model) advanceStep() (tea.Model, tea.Cmd) {
	m.currentStep++
	if m.currentStep < len(m.workflow.Steps) {
		m.questions = nil
		m.answers = nil
		m.textareaActive = false
		m.statusText = "* thinking  " + m.workflow.Steps[m.currentStep].StatusLabel
		return m, m.startCurrentStep()
	}
	// All steps done.
	if m.workflow.OnDone != nil {
		resultDir, err := m.workflow.OnDone()
		if err != nil {
			m.errMsg = err.Error()
			m.done = true
			m.statusText = "error  press q to exit"
			p := m.currentPalette()
			m = m.withLine(lipgloss.NewStyle().Foreground(p.errColor).Render("• error: "+m.errMsg) + "\n")
			return m, nil
		}
		m.resultDir = resultDir
	}
	m.done = true
	m.statusText = "done  press q to exit"
	p := m.currentPalette()
	if m.resultDir != "" {
		m = m.withLine(lipgloss.NewStyle().Foreground(p.success).Render(
			fmt.Sprintf("• completed  output: %s", m.resultDir),
		) + "\n")
	} else {
		m = m.withLine(lipgloss.NewStyle().Foreground(p.success).Render("• completed") + "\n")
	}
	return m, nil
}

// resumeAgentCmd starts a new runner turn with the user's answer.
func resumeAgentCmd(cfg config.Config, sessionID, projectPath, answer, logFile string) tea.Cmd {
	return func() tea.Msg {
		r, err := runner.NewRunner(cfg)
		if err != nil {
			return agentErrMsg{err: fmt.Errorf("creating runner: %w", err)}
		}

		events, errc := r.Run(runner.RunOptions{
			Prompts:   runner.Prompts{User: answer},
			Config:    cfg,
			SessionID: sessionID,
			CWD:       projectPath,
			LogFile:   logFile,
		})
		return readNext(events, errc)
	}
}

// waitForEvent returns a Cmd that reads the NEXT event from already-open channels.
func waitForEvent(events <-chan runner.Event, errc <-chan error) tea.Cmd {
	return func() tea.Msg { return readNext(events, errc) }
}

// readNext reads one event from the channel and returns the appropriate message.
// Channels are embedded in agentEventMsg so they propagate without model storage.
func readNext(events <-chan runner.Event, errc <-chan error) tea.Msg {
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
			m.vp.MouseWheelEnabled = true
			m.vp.SetContent(strings.Join(m.content, ""))
			m.ready = true
			if m.workflow.Preamble != "" {
				rendered := m.renderMarkdown(m.workflow.Preamble)
				p := m.currentPalette()
				bullet := lipgloss.NewStyle().Foreground(p.output).Render("•")
				m = m.withLine(bulletPrefix(bullet, rendered) + "\n")
			}
		} else {
			m.vp.Width = msg.Width
			m.vp.Height = m.viewportHeight()
		}
		return m, nil

	case tea.MouseMsg:
		prevOffset := m.vp.YOffset
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		if m.vp.YOffset < prevOffset {
			m.followMode = false
		}
		return m, cmd

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
	// Textarea has priority when active
	if m.textareaActive {
		return m.handleTextareaInput(msg)
	}

	if m.otherInput {
		return m.handleOtherInput(msg)
	}

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

	case "v", "V":
		m.detailMode = !m.detailMode
		return m, nil

	case "up", "k":
		m.followMode = false

	case "enter":
		// Re-activate textarea for text questions if it was dismissed
		if len(m.questions) > 0 && m.questions[0].Type == runner.QuestionTypeText && !m.textareaActive {
			q := m.questions[0]
			placeholder := fmt.Sprintf("Enter your response for %s...", q.Header)
			m.initTextarea(placeholder)
			m.syncViewport()
			return m, nil
		}

	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		return m.handleNumberKey(msg.String())
	}

	var cmd tea.Cmd
	prevOffset := m.vp.YOffset
	m.vp, cmd = m.vp.Update(msg)
	if m.vp.YOffset < prevOffset {
		m.followMode = false
	}
	return m, cmd
}

func (m model) handleOtherInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.otherText == "" {
			return m, nil
		}
		label := m.otherText
		m.otherInput = false
		m.otherText = ""
		m.answers = append(m.answers, label)
		m.questions = m.questions[1:]

		p := m.currentPalette()
		m = m.withLine(lipgloss.NewStyle().Foreground(p.answer).Render("> "+label) + "\n")

		if len(m.questions) > 0 {
			return m, nil
		}

		answer := strings.Join(m.answers, "\n")
		m.answers = nil
		m.statusText = "* thinking  " + m.currentStepLabel()
		return m, resumeAgentCmd(m.cfg, m.sessionID, m.projectPath, answer, m.logFile)

	case "esc":
		m.otherInput = false
		m.otherText = ""
		return m, nil

	case "backspace", "ctrl+h":
		if len(m.otherText) > 0 {
			runes := []rune(m.otherText)
			m.otherText = string(runes[:len(runes)-1])
		}
		return m, nil

	case "ctrl+c":
		return m, tea.Quit
	}

	if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace {
		m.otherText += msg.String()
	}
	return m, nil
}

// handleTextareaInput processes keys when textarea is active
func (m model) handleTextareaInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "ctrl+d", "ctrl+s":
		// Submit multi-line input (Ctrl+D or Ctrl+S)
		answer := m.textarea.Value()
		if answer == "" {
			return m, nil
		}

		// Render the submitted answer
		p := m.currentPalette()
		// For multi-line answers, show a preview of the first line
		firstLine := strings.Split(answer, "\n")[0]
		if len(firstLine) > 60 {
			firstLine = firstLine[:60] + "..."
		}
		lineCount := len(strings.Split(answer, "\n"))
		previewText := fmt.Sprintf("> %s (%d lines)", firstLine, lineCount)
		m = m.withLine(lipgloss.NewStyle().Foreground(p.answer).Render(previewText) + "\n")

		// Deactivate textarea
		m.textareaActive = false
		m.textarea.Reset()

		// Add answer and proceed
		m.answers = append(m.answers, answer)
		m.questions = m.questions[1:]

		// If more questions remain, check if next needs textarea
		if len(m.questions) > 0 {
			nextQ := m.questions[0]
			if nextQ.Type == runner.QuestionTypeText {
				placeholder := fmt.Sprintf("Enter your response for %s...", nextQ.Header)
				m.initTextarea(placeholder)
			}
			m.syncViewport()
			return m, nil
		}

		// All questions answered, resume agent
		fullAnswer := joinAnswers(m.answers)
		m.answers = nil
		m.statusText = "* thinking  " + m.currentStepLabel()
		m.syncViewport()
		return m, resumeAgentCmd(m.cfg, m.sessionID, m.projectPath, fullAnswer, m.logFile)

	case "esc":
		// Cancel input
		m.textareaActive = false
		m.textarea.Reset()
		m.syncViewport()
		return m, nil

	case "ctrl+c":
		// Force quit
		return m, tea.Quit

	default:
		// Delegate all other keys to textarea (typing, navigation, etc.)
		m.textarea, cmd = m.textarea.Update(msg)
	}

	return m, cmd
}

// joinAnswers combines multiple answers with proper formatting
func joinAnswers(answers []string) string {
	var result strings.Builder
	for i, answer := range answers {
		if i > 0 {
			result.WriteString("\n\n---\n\n")
		}
		result.WriteString(answer)
	}
	return result.String()
}

func (m model) handleNumberKey(key string) (tea.Model, tea.Cmd) {
	if len(m.questions) == 0 {
		return m, nil
	}
	q := m.questions[0]
	if q.Type != runner.QuestionTypeChoice {
		return m, nil
	}

	n := int(key[0] - '0')
	if n < 1 {
		return m, nil
	}

	otherIdx := len(q.Options) + 1
	if n == otherIdx {
		// "Other" selected — open textarea
		placeholder := fmt.Sprintf("Enter your response for %s...", q.Header)
		m.initTextarea(placeholder)
		m.syncViewport()
		return m, nil
	}

	if n > len(q.Options) {
		return m, nil
	}

	opt := q.Options[n-1]
	label, _ := opt["label"].(string)

	p := m.currentPalette()
	m = m.withLine(lipgloss.NewStyle().Foreground(p.answer).Render(fmt.Sprintf("> %s", label)) + "\n")

	m.answers = append(m.answers, label)
	m.questions = m.questions[1:]

	if len(m.questions) > 0 {
		nextQ := m.questions[0]
		if nextQ.Type == runner.QuestionTypeText {
			placeholder := fmt.Sprintf("Enter your response for %s...", nextQ.Header)
			m.initTextarea(placeholder)
		}
		return m, nil
	}

	answer := joinAnswers(m.answers)
	m.answers = nil
	m.statusText = "* thinking  " + m.currentStepLabel()
	return m, resumeAgentCmd(m.cfg, m.sessionID, m.projectPath, answer, m.logFile)
}

// handleAgentEvent processes one event from the runner.
// It always receives the full agentEventMsg so it can return the next waitForEvent
// command with the correct channel references.
func (m model) handleAgentEvent(msg agentEventMsg) (tea.Model, tea.Cmd) {
	event := msg.event

	if id := event.SessionID(); id != "" {
		m.sessionID = id
	}

	// Tool use — log inline in detail mode, otherwise update status line.
	for _, tool := range event.ToolUses() {
		name, _ := tool["name"].(string)
		input, _ := tool["input"].(map[string]any)
		desc := toolDescription(name, input)
		if m.detailMode {
			p := m.currentPalette()
			line := lipgloss.NewStyle().Foreground(p.faint).Render("  ⚙ " + desc)
			m = m.withLine(line + "\n")
		} else {
			m.toolLine = desc
		}
	}
	// Sync viewport height when toolLine appears (withLine not called for tool-only events).
	if m.ready && m.toolLine != "" {
		m.vp.Height = m.viewportHeight()
	}

	// Text content — render and append
	if text := event.TextContent(); text != "" {
		m.toolLine = ""
		finished := runner.DetectFinished(text)
		gotoTarget, hasGoto := runner.DetectGoto(text)
		displayText := runner.StripMarkers(text)
		if displayText != "" {
			rendered := m.renderMarkdown(displayText)
			p := m.currentPalette()
			bullet := lipgloss.NewStyle().Foreground(p.output).Render("•")
			m = m.withLine(bulletPrefix(bullet, rendered) + "\n")
		}

		newQuestions := runner.DetectQuestions(text)
		m.questions = append(m.questions, newQuestions...)

		// For text-type questions, auto-activate textarea; choice type shows options
		if len(newQuestions) > 0 && !m.textareaActive {
			q := newQuestions[0]
			if q.Type == runner.QuestionTypeText {
				placeholder := fmt.Sprintf("Enter your response for %s...", q.Header)
				m.initTextarea(placeholder)
			}
		}

		// Sync viewport height now that questions/textarea state is final.
		m.syncViewport()

		if finished {
			return m.advanceStep()
		}
		if hasGoto {
			return m.gotoStep(gotoTarget)
		}
	}

	// Result event — advance to next step or complete.
	if event.IsResult() {
		m.toolLine = "" // clear any lingering tool status
		if event.IsError() {
			m.errMsg = event.ResultText()
			m.done = true
			m.statusText = "error  press q to exit"
			p := m.currentPalette()
			m = m.withLine(lipgloss.NewStyle().Foreground(p.errColor).Render("• "+m.errMsg) + "\n")
			return m, nil
		}
		// If questions are still pending, wait for the user to answer before advancing.
		// The user's answer will resume the session via resumeAgentCmd.
		if len(m.questions) > 0 {
			return m, nil
		}
		return m.advanceStep()
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
	detailHint := "v: detail"
	if m.detailMode {
		detailHint = "v: simple"
	}
	sections = append(sections, statusStyle.Render(fmt.Sprintf("%s  %s  %s", m.statusText, followHint, detailHint)))

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
	faintStyle := lipgloss.NewStyle().Foreground(p.faint)
	optStyle := lipgloss.NewStyle().Foreground(p.answer)

	// Wrap question text to available width
	wrapWidth := m.width - 6 // account for border + padding
	if wrapWidth < 20 {
		wrapWidth = 20
	}

	var lines []string
	lines = append(lines, headerStyle.Render(q.Header))
	lines = append(lines, wordWrap(q.Question, wrapWidth))

	// Textarea is active — shown for text questions and "Other" in choice questions
	if m.textareaActive {
		lines = append(lines, "")
		lines = append(lines, m.textarea.View())
		lines = append(lines, "")
		lines = append(lines, faintStyle.Render("ctrl+d or ctrl+s to submit  •  esc to cancel  •  supports markdown"))
		return borderStyle.Render(strings.Join(lines, "\n"))
	}

	if q.Type == runner.QuestionTypeChoice {
		// Render numbered options + automatic "Other"
		lines = append(lines, "")
		for i, opt := range q.Options {
			label, _ := opt["label"].(string)
			desc, _ := opt["description"].(string)
			line := optStyle.Render(fmt.Sprintf("  %d. %s", i+1, label))
			if desc != "" {
				line += faintStyle.Render(" — "+desc)
			}
			lines = append(lines, line)
		}
		otherIdx := len(q.Options) + 1
		lines = append(lines, faintStyle.Render(fmt.Sprintf("  %d. Other (free text)", otherIdx)))
		lines = append(lines, "")
		lines = append(lines, faintStyle.Render("press number to select"))
		return borderStyle.Render(strings.Join(lines, "\n"))
	}

	// Text type — textarea was deactivated (esc); show re-activate hint
	lines = append(lines, "")
	lines = append(lines, faintStyle.Render("press enter to start typing"))
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
		m.vp.Height = m.viewportHeight()
		m.vp.SetContent(strings.Join(m.content, ""))
		if m.followMode {
			m.vp.GotoBottom()
		}
	}
	return m
}

// syncViewport updates the viewport height to match the current layout state.
// Call this whenever textareaActive or questions change outside of withLine.
func (m *model) syncViewport() {
	if m.ready {
		m.vp.Height = m.viewportHeight()
		m.vp.SetContent(strings.Join(m.content, ""))
		if m.followMode {
			m.vp.GotoBottom()
		}
	}
}

// questionPanelLines returns the actual number of terminal lines the question panel occupies.
// This accounts for multi-line question text after word-wrapping.
func (m model) questionPanelLines() int {
	if len(m.questions) == 0 {
		return 0
	}
	q := m.questions[0]
	wrapWidth := m.width - 6 // matches renderQuestionPanel
	if wrapWidth < 20 {
		wrapWidth = 20
	}
	questionLines := strings.Count(wordWrap(q.Question, wrapWidth), "\n") + 1

	if m.textareaActive {
		// border(1) + header(1) + question(N) + blank(1) + textarea + blank(1) + hint(1)
		return 1 + 1 + questionLines + 1 + m.textarea.Height() + 1 + 1
	}
	if q.Type == runner.QuestionTypeChoice {
		// border(1) + header(1) + question(N) + blank(1) + options + Other(1) + blank(1) + hint(1)
		return 1 + 1 + questionLines + 1 + len(q.Options) + 1 + 1 + 1
	}
	// Text type, textarea not yet active: border(1) + header(1) + question(N) + blank(1) + hint(1)
	return 1 + 1 + questionLines + 1 + 1
}

func (m model) viewportHeight() int {
	reserved := 1 // status bar
	if m.toolLine != "" {
		reserved++
	}
	if len(m.questions) > 0 {
		reserved += m.questionPanelLines()
	}
	h := m.height - reserved
	if h < 3 {
		h = 3
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

// wordWrap breaks s into lines of at most width runes, preserving existing newlines.
func wordWrap(s string, width int) string {
	if width <= 0 {
		return s
	}
	paragraphs := strings.Split(s, "\n")
	wrapped := make([]string, len(paragraphs))
	for i, p := range paragraphs {
		wrapped[i] = wrapLine(p, width)
	}
	return strings.Join(wrapped, "\n")
}

// wrapLine wraps a single line (no embedded newlines) to at most width runes.
func wrapLine(s string, width int) string {
	var result strings.Builder
	lineLen := 0
	for _, word := range strings.Fields(s) {
		wl := len([]rune(word))
		if lineLen > 0 {
			if lineLen+1+wl > width {
				result.WriteByte('\n')
				lineLen = 0
			} else {
				result.WriteByte(' ')
				lineLen++
			}
		}
		result.WriteString(word)
		lineLen += wl
	}
	return result.String()
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

// RunAgentTUI is the generic TUI entry point. Callers provide a Workflow that
// controls how the prompt is built and how the result is handled.
func RunAgentTUI(wf Workflow, projectPath string, cfg config.Config) (string, error) {
	m := initialModel(wf, projectPath, cfg)

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

