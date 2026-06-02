# Bubble Tea

Bubble Tea is a powerful Go framework for building rich, interactive terminal user interfaces using functional programming paradigms based on The Elm Architecture. The library provides a comprehensive event-driven system where applications are built around three core concepts: a Model that represents application state, an Update function that handles incoming events and state changes, and a View function that renders the UI.

## Package Information

- **Package Name**: bubbletea
- **Package Type**: go module
- **Language**: Go
- **Installation**: `go get github.com/charmbracelet/bubbletea`
- **Go Version**: Requires Go 1.24.0+
- **Import Path**: `github.com/charmbracelet/bubbletea`

## Core Imports

```go
import tea "github.com/charmbracelet/bubbletea"
```

Alternative direct imports for specific functionality:

```go
import (
    tea "github.com/charmbracelet/bubbletea"
    "context"
    "time"
)
```

## Basic Usage

```go
package main

import (
    "fmt"
    tea "github.com/charmbracelet/bubbletea"
)

// Define your model
type model struct {
    choices  []string
    cursor   int
    selected map[int]struct{}
}

// Implement the Model interface
func (m model) Init() tea.Cmd {
    // Return initial command (or nil for no initial command)
    return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "ctrl+c", "q":
            return m, tea.Quit()
        case "up", "k":
            if m.cursor > 0 {
                m.cursor--
            }
        case "down", "j":
            if m.cursor < len(m.choices)-1 {
                m.cursor++
            }
        }
    }
    return m, nil
}

func (m model) View() string {
    s := "What should we buy at the grocery store?\n\n"

    for i, choice := range m.choices {
        cursor := " "
        if m.cursor == i {
            cursor = ">"
        }
        s += fmt.Sprintf("%s %s\n", cursor, choice)
    }

    s += "\nPress q to quit.\n"
    return s
}

func main() {
    m := model{
        choices: []string{"Apples", "Oranges", "Bananas"},
        selected: make(map[int]struct{}),
    }

    p := tea.NewProgram(m)
    if _, err := p.Run(); err != nil {
        fmt.Printf("Error: %v", err)
    }
}
```

## Architecture

Bubble Tea is built around The Elm Architecture pattern with these key components:

- **Model Interface**: Represents application state and defines three core methods (Init, Update, View)
- **Message System**: All events flow through the Msg interface, enabling type-safe event handling
- **Command System**: Represents side effects and asynchronous operations that return messages
- **Program**: Manages the event loop, terminal state, and renders the UI
- **Renderer**: Handles ANSI escape codes, terminal control, and efficient screen updates

The architecture ensures predictable state management through immutable updates and clear separation of concerns between state (Model), logic (Update), and presentation (View).

## Capabilities

### Program Management

Core program lifecycle management including initialization, execution, and cleanup with full terminal state control.

```go { .api }
func NewProgram(model Model, opts ...ProgramOption) *Program

func (p *Program) Run() (Model, error)
func (p *Program) Send(msg Msg)
func (p *Program) Quit()
func (p *Program) Kill()
```

[Program Management](./program.md)

### Input Handling

Comprehensive keyboard and mouse input handling with support for all standard terminal input including function keys, mouse events, focus events, and bracketed paste.

```go { .api }
type KeyMsg Key
type MouseMsg MouseEvent

type Key struct {
    Type  KeyType
    Runes []rune
    Alt   bool
    Paste bool
}

type MouseEvent struct {
    X      int
    Y      int
    Shift  bool
    Alt    bool
    Ctrl   bool
    Action MouseAction
    Button MouseButton
}
```

[Input Handling](./input.md)

### Command System

Asynchronous operations and side effects through commands, including built-in commands for timers, batching, and external process execution.

```go { .api }
type Cmd func() Msg

func Batch(cmds ...Cmd) Cmd
func Sequence(cmds ...Cmd) Cmd
func Tick(d time.Duration, fn func(time.Time) Msg) Cmd
func Every(duration time.Duration, fn func(time.Time) Msg) Cmd
```

[Command System](./commands.md)

### Terminal Control

Complete terminal and screen control including alternate screen buffer, mouse modes, cursor control, focus reporting, and bracketed paste mode.

```go { .api }
func ClearScreen() Msg
func EnterAltScreen() Msg
func ExitAltScreen() Msg
func EnableMouseCellMotion() Msg
func EnableMouseAllMotion() Msg
func DisableMouse() Msg
```

[Terminal Control](./screen.md)

## Core Types

```go { .api }
// Core interfaces
type Model interface {
    Init() Cmd
    Update(Msg) (Model, Cmd)
    View() string
}

type Msg interface{}

type Cmd func() Msg

// Program options
type ProgramOption func(*Program)

// Key and mouse types
type KeyType int
type MouseAction int
type MouseButton int

// Window size information
type WindowSizeMsg struct {
    Width  int
    Height int
}
```

## Error Handling

```go { .api }
var (
    ErrProgramPanic = errors.New("program experienced a panic")
    ErrProgramKilled = errors.New("program was killed")
    ErrInterrupted = errors.New("program was interrupted")
)
```

These errors are returned by `Program.Run()` under different termination conditions:
- `ErrProgramPanic`: Program recovered from a panic
- `ErrProgramKilled`: Program was forcibly terminated
- `ErrInterrupted`: Program received SIGINT or interrupt message