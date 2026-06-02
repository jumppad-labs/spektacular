# Terminal Control

Complete terminal and screen control functionality including alternate screen buffer, mouse modes, cursor control, focus reporting, and bracketed paste mode.

## Capabilities

### Screen Buffer Management

Control alternate screen buffer for full-screen terminal applications.

```go { .api }
/**
 * ClearScreen clears the screen before next update
 * Moves cursor to top-left and clears visual clutter
 * Only needed for special cases, not regular redraws
 * @returns Message to clear screen
 */
func ClearScreen() Msg

/**
 * EnterAltScreen switches to alternate screen buffer
 * Provides full-window mode, hiding terminal history
 * Screen automatically restored on program exit
 * @returns Message to enter alternate screen
 */
func EnterAltScreen() Msg

/**
 * ExitAltScreen switches back to main screen buffer
 * Returns to normal terminal with history visible
 * Usually not needed as altscreen auto-exits on quit
 * @returns Message to exit alternate screen
 */
func ExitAltScreen() Msg
```

**Usage Examples:**

```go
// Initialize with alternate screen
p := tea.NewProgram(model{}, tea.WithAltScreen())

// Or use commands for dynamic control
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "f":
            return m, tea.EnterAltScreen()
        case "ctrl+z":
            // Clear screen before suspending
            return m, tea.Batch(
                tea.ClearScreen(),
                tea.Suspend(),
            )
        }
    }
    return m, nil
}
```

### Mouse Control

Enable and configure mouse input modes.

```go { .api }
/**
 * EnableMouseCellMotion enables mouse click, release, and wheel events
 * Also captures mouse movement when button is pressed (drag events)
 * Better terminal compatibility than all motion mode
 * @returns Message to enable cell motion mouse mode
 */
func EnableMouseCellMotion() Msg

/**
 * EnableMouseAllMotion enables all mouse events including hover
 * Captures click, release, wheel, and motion without button press
 * Less widely supported, use cell motion if unsure
 * @returns Message to enable all motion mouse mode
 */
func EnableMouseAllMotion() Msg

/**
 * DisableMouse stops listening for mouse events
 * Mouse automatically disabled when program exits
 * @returns Message to disable mouse input
 */
func DisableMouse() Msg
```

**Usage Examples:**

```go
// Static configuration (recommended)
p := tea.NewProgram(model{},
    tea.WithMouseCellMotion(), // or WithMouseAllMotion()
)

// Dynamic mouse control
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "m":
            if m.mouseEnabled {
                m.mouseEnabled = false
                return m, tea.DisableMouse()
            } else {
                m.mouseEnabled = true
                return m, tea.EnableMouseCellMotion()
            }
        }
    case tea.MouseMsg:
        // Handle mouse events
        return m.handleMouse(msg), nil
    }
    return m, nil
}
```

### Cursor Control

Show and hide the terminal cursor.

```go { .api }
/**
 * HideCursor hides the terminal cursor
 * Cursor automatically hidden during normal program lifetime
 * Rarely needed unless terminal operations show cursor unexpectedly
 * @returns Message to hide cursor
 */
func HideCursor() Msg

/**
 * ShowCursor shows the terminal cursor
 * Useful for input fields or when cursor should be visible
 * @returns Message to show cursor
 */
func ShowCursor() Msg
```

**Usage Examples:**

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "i":
            // Enter input mode, show cursor
            m.inputMode = true
            return m, tea.ShowCursor()
        case "esc":
            // Exit input mode, hide cursor
            m.inputMode = false
            return m, tea.HideCursor()
        }
    }
    return m, nil
}
```

### Bracketed Paste Control

Manage bracketed paste mode for handling large clipboard content.

```go { .api }
/**
 * EnableBracketedPaste enables bracketed paste mode
 * Large pastes delivered as single KeyMsg with Paste=true
 * Bracketed paste enabled by default unless disabled via option
 * @returns Message to enable bracketed paste
 */
func EnableBracketedPaste() Msg

/**
 * DisableBracketedPaste disables bracketed paste mode
 * Each character in paste delivered as separate KeyMsg
 * @returns Message to disable bracketed paste
 */
func DisableBracketedPaste() Msg
```

**Usage Examples:**

```go
// Disable via program option
p := tea.NewProgram(model{}, tea.WithoutBracketedPaste())

// Dynamic control
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.Paste {
            // Handle large paste operation
            m.content += string(msg.Runes)
            return m, nil
        }
        // Handle regular keystrokes
        return m.handleKey(msg), nil
    }
    return m, nil
}
```

### Focus Reporting

Enable terminal focus and blur event reporting.

```go { .api }
/**
 * EnableReportFocus enables focus/blur event reporting
 * Sends FocusMsg when terminal gains focus
 * Sends BlurMsg when terminal loses focus
 * @returns Message to enable focus reporting
 */
func EnableReportFocus() Msg

/**
 * DisableReportFocus disables focus/blur event reporting
 * @returns Message to disable focus reporting
 */
func DisableReportFocus() Msg
```

**Usage Examples:**

```go
// Enable via program option
p := tea.NewProgram(model{}, tea.WithReportFocus())

// Handle focus events
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg.(type) {
    case tea.FocusMsg:
        m.focused = true
        // Maybe pause animations when not focused
        return m, m.startAnimationCmd()
    case tea.BlurMsg:
        m.focused = false
        // Pause updates when not focused
        return m, nil
    }
    return m, nil
}
```

### Window Size Information

Handle terminal resize events and query current dimensions.

```go { .api }
/**
 * WindowSizeMsg reports current terminal dimensions
 * Automatically sent on program start and terminal resize
 * Width and height in character cells (not pixels)
 */
type WindowSizeMsg struct {
    Width  int  // Terminal width in characters
    Height int  // Terminal height in characters
}
```

**Usage Examples:**

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height

        // Adjust layout based on size
        if msg.Width < 80 {
            m.layout = "compact"
        } else {
            m.layout = "full"
        }

        return m, nil
    }
    return m, nil
}

func (m model) View() string {
    // Use m.width and m.height for responsive layout
    content := m.renderContent()

    // Ensure content fits terminal
    if len(content) > m.height-1 {
        content = content[:m.height-1]
    }

    return strings.Join(content, "\n")
}
```

## Deprecated Program Methods

These methods exist for backwards compatibility but should be avoided:

```go { .api }
/**
 * Deprecated: Use WithAltScreen program option instead
 * EnterAltScreen enters alternate screen if renderer is available
 */
func (p *Program) EnterAltScreen()

/**
 * Deprecated: Altscreen automatically exits when program ends
 * ExitAltScreen exits alternate screen if renderer is available
 */
func (p *Program) ExitAltScreen()

/**
 * Deprecated: Use WithMouseCellMotion program option instead
 * EnableMouseCellMotion enables mouse cell motion
 */
func (p *Program) EnableMouseCellMotion()

/**
 * Deprecated: Mouse automatically disabled on program exit
 * DisableMouseCellMotion disables mouse cell motion
 */
func (p *Program) DisableMouseCellMotion()

/**
 * Deprecated: Use WithMouseAllMotion program option instead
 * EnableMouseAllMotion enables mouse all motion
 */
func (p *Program) EnableMouseAllMotion()

/**
 * Deprecated: Mouse automatically disabled on program exit
 * DisableMouseAllMotion disables mouse all motion
 */
func (p *Program) DisableMouseAllMotion()

/**
 * Deprecated: Use SetWindowTitle command instead
 * SetWindowTitle sets window title if renderer is available
 */
func (p *Program) SetWindowTitle(title string)
```

## Terminal State Management Patterns

### Responsive Layout

Pattern for handling different terminal sizes:

```go
type layout struct {
    compact bool
    columns int
    rows    int
}

func (l layout) maxContentWidth() int {
    if l.compact {
        return l.columns - 2  // Leave margins
    }
    return min(l.columns-4, 120)  // Max width for readability
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.layout = layout{
            compact: msg.Width < 80,
            columns: msg.Width,
            rows:    msg.Height,
        }
        return m, nil
    }
    return m, nil
}

func (m model) View() string {
    maxWidth := m.layout.maxContentWidth()
    content := wrapText(m.content, maxWidth)

    if m.layout.compact {
        return m.renderCompactView(content)
    }
    return m.renderFullView(content)
}
```

### Focus-Aware Updates

Pattern for pausing updates when terminal loses focus:

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg.(type) {
    case tea.FocusMsg:
        m.focused = true
        // Resume periodic updates
        return m, m.startPeriodicUpdate()

    case tea.BlurMsg:
        m.focused = false
        // Don't schedule more updates
        return m, nil

    case tickMsg:
        if !m.focused {
            // Skip update if not focused
            return m, nil
        }
        // Normal update and reschedule
        m.lastUpdate = time.Now()
        return m, m.startPeriodicUpdate()
    }
    return m, nil
}
```

### Progressive Feature Detection

Pattern for enabling features based on terminal capabilities:

```go
func (m model) Init() tea.Cmd {
    return tea.Batch(
        tea.WindowSize(),  // Get initial size
        m.detectCapabilitiesCmd(),
    )
}

type capabilitiesMsg struct {
    hasMouseSupport   bool
    hasFocusReporting bool
    hasAltScreen      bool
}

func (m model) detectCapabilitiesCmd() tea.Cmd {
    return func() tea.Msg {
        // Simple capability detection
        term := os.Getenv("TERM")

        return capabilitiesMsg{
            hasMouseSupport:   term != "dumb",
            hasFocusReporting: strings.Contains(term, "xterm"),
            hasAltScreen:      term != "dumb",
        }
    }
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case capabilitiesMsg:
        var cmds []tea.Cmd

        if msg.hasMouseSupport {
            cmds = append(cmds, tea.EnableMouseCellMotion())
        }
        if msg.hasFocusReporting {
            cmds = append(cmds, tea.EnableReportFocus())
        }
        if msg.hasAltScreen && m.wantsFullScreen {
            cmds = append(cmds, tea.EnterAltScreen())
        }

        return m, tea.Batch(cmds...)
    }
    return m, nil
}
```