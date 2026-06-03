# Input Handling

Comprehensive keyboard and mouse input handling system providing support for all standard terminal input including function keys, mouse events, focus events, and bracketed paste mode.

## Capabilities

### Keyboard Input

Keyboard input is delivered through KeyMsg messages containing detailed key information.

```go { .api }
/**
 * KeyMsg contains information about a keypress
 * Always sent to the program's update function for keyboard events
 */
type KeyMsg Key

type Key struct {
    Type  KeyType  // Key type (special keys, control keys, or runes)
    Runes []rune   // Character data (always contains at least one rune for KeyRunes)
    Alt   bool     // Alt/Option modifier key pressed
    Paste bool     // Key event originated from bracketed paste
}

/**
 * Returns friendly string representation of key
 * Safe and encouraged for key comparison in switch statements
 */
func (k Key) String() string
func (k KeyMsg) String() string
```

**Usage Example:**

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Method 1: String comparison (simpler)
        switch msg.String() {
        case "ctrl+c", "q":
            return m, tea.Quit()
        case "enter":
            return m.handleEnter(), nil
        case "a":
            return m.addItem(), nil
        }

        // Method 2: Type-based matching (more robust)
        switch msg.Type {
        case tea.KeyEnter:
            return m.handleEnter(), nil
        case tea.KeyRunes:
            switch string(msg.Runes) {
            case "a":
                return m.addItem(), nil
            }
        }
    }
    return m, nil
}
```

### Key Types and Constants

```go { .api }
type KeyType int

// Control key constants
const (
    KeyNull      KeyType // null character
    KeyBreak     KeyType // ctrl+c
    KeyEnter     KeyType // enter/return
    KeyBackspace KeyType // backspace
    KeyTab       KeyType // tab
    KeyEsc       KeyType // escape
    KeyEscape    KeyType // escape (alias)
)

// Ctrl+Letter combinations
const (
    KeyCtrlA KeyType  // ctrl+a through ctrl+z
    KeyCtrlB KeyType
    // ... (all ctrl+letter combinations available)
    KeyCtrlZ KeyType
)

// Navigation and special keys
const (
    KeyRunes    KeyType // Regular character input
    KeyUp       KeyType // Arrow keys
    KeyDown     KeyType
    KeyRight    KeyType
    KeyLeft     KeyType
    KeyShiftTab KeyType // Shift+tab
    KeyHome     KeyType // Home/end keys
    KeyEnd      KeyType
    KeyPgUp     KeyType // Page up/down
    KeyPgDown   KeyType
    KeyDelete   KeyType // Delete/insert
    KeyInsert   KeyType
    KeySpace    KeyType // Space bar
)

// Function keys
const (
    KeyF1  KeyType  // Function keys F1-F20
    KeyF2  KeyType
    KeyF3  KeyType
    KeyF4  KeyType
    KeyF5  KeyType
    KeyF6  KeyType
    KeyF7  KeyType
    KeyF8  KeyType
    KeyF9  KeyType
    KeyF10 KeyType
    KeyF11 KeyType
    KeyF12 KeyType
    KeyF13 KeyType  // Extended function keys (F13-F20)
    KeyF14 KeyType
    KeyF15 KeyType
    KeyF16 KeyType
    KeyF17 KeyType
    KeyF18 KeyType
    KeyF19 KeyType
    KeyF20 KeyType
)

// Modified navigation keys
const (
    KeyCtrlUp    KeyType  // Ctrl+arrow combinations
    KeyCtrlDown  KeyType
    KeyCtrlLeft  KeyType
    KeyCtrlRight KeyType
    KeyCtrlPgUp    KeyType  // Ctrl+page up/down
    KeyCtrlPgDown  KeyType
    KeyShiftUp   KeyType  // Shift+arrow combinations
    KeyShiftDown KeyType
    KeyShiftLeft KeyType
    KeyShiftRight KeyType
    KeyCtrlShiftUp    KeyType  // Ctrl+Shift+arrow combinations
    KeyCtrlShiftDown  KeyType
    KeyCtrlShiftLeft  KeyType
    KeyCtrlShiftRight KeyType
    KeyCtrlShiftHome  KeyType  // Ctrl+Shift+home/end
    KeyCtrlShiftEnd   KeyType
)
```

### Mouse Input

Mouse events are delivered through MouseMsg messages with position and button information.

```go { .api }
/**
 * MouseMsg contains information about a mouse event
 * Sent when mouse activity occurs (must be enabled first)
 */
type MouseMsg MouseEvent

type MouseEvent struct {
    X      int          // Mouse coordinates (0-based)
    Y      int
    Shift  bool         // Modifier keys pressed during event
    Alt    bool
    Ctrl   bool
    Action MouseAction  // Type of mouse action
    Button MouseButton  // Which button was involved
}

/**
 * Checks if the mouse event is a wheel event
 */
func (m MouseEvent) IsWheel() bool

/**
 * Returns string representation of mouse event
 */
func (m MouseEvent) String() string
func (m MouseMsg) String() string
```

**Usage Example:**

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.MouseMsg:
        switch msg.Action {
        case tea.MouseActionPress:
            if msg.Button == tea.MouseButtonLeft {
                return m.handleClick(msg.X, msg.Y), nil
            }
        case tea.MouseActionRelease:
            return m.handleRelease(), nil
        case tea.MouseActionMotion:
            if msg.Button == tea.MouseButtonLeft {
                return m.handleDrag(msg.X, msg.Y), nil
            }
        }

        // Handle wheel events
        if msg.IsWheel() {
            switch msg.Button {
            case tea.MouseButtonWheelUp:
                return m.scrollUp(), nil
            case tea.MouseButtonWheelDown:
                return m.scrollDown(), nil
            }
        }
    }
    return m, nil
}
```

### Mouse Actions and Buttons

```go { .api }
type MouseAction int

const (
    MouseActionPress   MouseAction  // Mouse button pressed
    MouseActionRelease MouseAction  // Mouse button released
    MouseActionMotion  MouseAction  // Mouse moved
)

type MouseButton int

const (
    MouseButtonNone       MouseButton  // No button (motion/release events)
    MouseButtonLeft       MouseButton  // Left mouse button
    MouseButtonMiddle     MouseButton  // Middle button (scroll wheel click)
    MouseButtonRight      MouseButton  // Right mouse button
    MouseButtonWheelUp    MouseButton  // Scroll wheel up
    MouseButtonWheelDown  MouseButton  // Scroll wheel down
    MouseButtonWheelLeft  MouseButton  // Scroll wheel left (horizontal)
    MouseButtonWheelRight MouseButton  // Scroll wheel right (horizontal)
    MouseButtonBackward   MouseButton  // Browser backward button
    MouseButtonForward    MouseButton  // Browser forward button
    MouseButton10         MouseButton  // Additional mouse buttons
    MouseButton11         MouseButton
)
```

### Focus Events

Terminal focus and blur events when focus reporting is enabled.

```go { .api }
/**
 * FocusMsg sent when terminal gains focus
 * Requires WithReportFocus() program option
 */
type FocusMsg struct{}

/**
 * BlurMsg sent when terminal loses focus
 * Requires WithReportFocus() program option
 */
type BlurMsg struct{}
```

**Usage Example:**

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg.(type) {
    case tea.FocusMsg:
        m.focused = true
        return m, nil
    case tea.BlurMsg:
        m.focused = false
        return m, nil
    }
    return m, nil
}

// Enable focus reporting
p := tea.NewProgram(model{}, tea.WithReportFocus())
```

### Window Size Events

Window resize events are automatically sent when terminal size changes.

```go { .api }
/**
 * WindowSizeMsg reports terminal dimensions
 * Sent automatically on program start and when terminal is resized
 */
type WindowSizeMsg struct {
    Width  int  // Terminal width in characters
    Height int  // Terminal height in characters
}
```

**Usage Example:**

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        return m, nil
    }
    return m, nil
}
```

## Input Mode Configuration

### Enabling Mouse Input

Mouse input must be explicitly enabled through program options or commands.

```go { .api }
// Program options (recommended)
func WithMouseCellMotion() ProgramOption  // Clicks, releases, wheel, drag events
func WithMouseAllMotion() ProgramOption   // All motion including hover (not all terminals)

// Runtime commands (for dynamic control)
func EnableMouseCellMotion() Msg
func EnableMouseAllMotion() Msg
func DisableMouse() Msg
```

**Cell Motion vs All Motion:**

```go
// Cell motion: Better supported, captures drag events
p := tea.NewProgram(model{}, tea.WithMouseCellMotion())

// All motion: Includes hover, less widely supported
p := tea.NewProgram(model{}, tea.WithMouseAllMotion())

// Runtime control
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "m":
            return m, tea.EnableMouseCellMotion()
        case "M":
            return m, tea.DisableMouse()
        }
    }
    return m, nil
}
```

### Bracketed Paste Mode

Handles large clipboard pastes without triggering individual key events.

```go { .api }
// Bracketed paste is enabled by default, disable with:
func WithoutBracketedPaste() ProgramOption

// Runtime control
func EnableBracketedPaste() Msg
func DisableBracketedPaste() Msg
```

When bracketed paste is enabled, pasted content is marked with the `Paste` field:

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.Paste {
            // Handle pasted content
            return m.handlePaste(string(msg.Runes)), nil
        }
        // Handle regular keystrokes
        return m.handleKey(msg), nil
    }
    return m, nil
}
```

### Focus Reporting

Enable terminal focus/blur event reporting.

```go { .api }
// Program option (recommended)
func WithReportFocus() ProgramOption

// Runtime commands
func EnableReportFocus() Msg
func DisableReportFocus() Msg
```

## Input Processing Patterns

### Key Binding Systems

Common pattern for handling key bindings:

```go
type KeyBindings struct {
    Up     []string
    Down   []string
    Select []string
    Quit   []string
}

func (kb KeyBindings) Matches(msg tea.KeyMsg, action string) bool {
    var keys []string
    switch action {
    case "up":
        keys = kb.Up
    case "down":
        keys = kb.Down
    // ... other actions
    }

    for _, key := range keys {
        if msg.String() == key {
            return true
        }
    }
    return false
}

// Usage
bindings := KeyBindings{
    Up:     []string{"up", "k"},
    Down:   []string{"down", "j"},
    Select: []string{"enter", "space"},
    Quit:   []string{"q", "ctrl+c", "esc"},
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch {
        case bindings.Matches(msg, "up"):
            return m.moveUp(), nil
        case bindings.Matches(msg, "down"):
            return m.moveDown(), nil
        case bindings.Matches(msg, "quit"):
            return m, tea.Quit()
        }
    }
    return m, nil
}
```

### Mouse Hit Testing

Pattern for handling mouse clicks on UI elements:

```go
type Button struct {
    Text string
    X, Y int
    W, H int
}

func (b Button) Contains(x, y int) bool {
    return x >= b.X && x < b.X+b.W && y >= b.Y && y < b.Y+b.H
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.MouseMsg:
        if msg.Action == tea.MouseActionPress {
            for i, button := range m.buttons {
                if button.Contains(msg.X, msg.Y) {
                    return m.handleButtonClick(i), nil
                }
            }
        }
    }
    return m, nil
}
```