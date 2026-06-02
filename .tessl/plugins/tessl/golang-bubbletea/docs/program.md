# Program Management

Core program lifecycle management for Bubble Tea applications, including initialization, execution, configuration options, and terminal state control.

## Capabilities

### Program Creation

Creates a new Program instance with a model and optional configuration.

```go { .api }
/**
 * Creates a new Bubble Tea program with the given model and options
 * @param model - Initial model implementing the Model interface
 * @param opts - Variable number of ProgramOption functions
 * @returns New Program instance ready to run
 */
func NewProgram(model Model, opts ...ProgramOption) *Program

type Program struct {
    // Internal fields (not directly accessible)
}
```

**Usage Example:**

```go
type myModel struct {
    counter int
}

func (m myModel) Init() tea.Cmd { return nil }
func (m myModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m myModel) View() string { return "Hello World" }

// Basic program creation
p := tea.NewProgram(myModel{})

// With options
p := tea.NewProgram(myModel{},
    tea.WithAltScreen(),
    tea.WithMouseCellMotion(),
)
```

### Program Execution

Runs the program's main event loop, blocking until termination.

```go { .api }
/**
 * Runs the program's event loop and blocks until termination
 * @returns Final model state and any error that occurred
 */
func (p *Program) Run() (Model, error)

/**
 * Deprecated: Use Run() instead.
 * Initializes program and returns final model state
 * @returns Final model state and any error that occurred
 */
func (p *Program) StartReturningModel() (Model, error)

/**
 * Deprecated: Use Run() instead.
 * Initializes program, discards final model state
 * @returns Any error that occurred during execution
 */
func (p *Program) Start() error
```

**Usage Example:**

```go
p := tea.NewProgram(model{})
finalModel, err := p.Run()
if err != nil {
    fmt.Printf("Program error: %v\n", err)
}
```

### Message Injection

Sends messages to the running program from external goroutines or systems.

```go { .api }
/**
 * Sends a message to the program's update function
 * Safe to call from any goroutine, non-blocking if program hasn't started
 * No-op if program has already terminated
 * @param msg - Message to send to the update function
 */
func (p *Program) Send(msg Msg)
```

**Usage Example:**

```go
type tickMsg time.Time

// From another goroutine
go func() {
    ticker := time.NewTicker(time.Second)
    for {
        select {
        case t := <-ticker.C:
            p.Send(tickMsg(t))
        }
    }
}()
```

### Program Termination

Methods for stopping the program execution.

```go { .api }
/**
 * Gracefully quit the program, equivalent to sending QuitMsg
 * Safe to call from any goroutine
 */
func (p *Program) Quit()

/**
 * Immediately kill the program and restore terminal state
 * Skips final render, returns ErrProgramKilled
 */
func (p *Program) Kill()

/**
 * Blocks until the program has finished shutting down
 * Useful for coordinating with other goroutines
 */
func (p *Program) Wait()
```

### Terminal State Management

Control terminal state for integrating with external processes.

```go { .api }
/**
 * Restores original terminal state and cancels input reader
 * Use with RestoreTerminal to temporarily release control
 * @returns Error if restoration fails
 */
func (p *Program) ReleaseTerminal() error

/**
 * Reinitializes program after ReleaseTerminal
 * Restores terminal state and repaints screen
 * @returns Error if restoration fails
 */
func (p *Program) RestoreTerminal() error
```

**Usage Example:**

```go
// Temporarily release terminal for external command
p.ReleaseTerminal()
cmd := exec.Command("vim", "file.txt")
cmd.Run()
p.RestoreTerminal()
```

### Output Methods

Print output above the program interface.

```go { .api }
/**
 * Prints above the program interface (unmanaged output)
 * Output persists across renders, no effect if altscreen is active
 * @param args - Values to print, similar to fmt.Print
 */
func (p *Program) Println(args ...interface{})

/**
 * Printf-style printing above the program interface
 * Message printed on its own line, no effect if altscreen is active
 * @param template - Format string
 * @param args - Values for format string
 */
func (p *Program) Printf(template string, args ...interface{})
```

## Program Configuration Options

Program options customize behavior during initialization.

### Context and Lifecycle Options

```go { .api }
/**
 * Sets context for program execution, enables external cancellation
 * Program exits with ErrProgramKilled when context is cancelled
 */
func WithContext(ctx context.Context) ProgramOption

/**
 * Disables the built-in signal handler for SIGINT/SIGTERM
 * Useful when you want to handle signals yourself
 */
func WithoutSignalHandler() ProgramOption

/**
 * Disables panic catching and recovery
 * Warning: Terminal may be left in unusable state after panic
 */
func WithoutCatchPanics() ProgramOption

/**
 * Ignores OS signals (mainly useful for testing)
 * Prevents automatic handling of SIGINT/SIGTERM
 */
func WithoutSignals() ProgramOption
```

### Input/Output Options

```go { .api }
/**
 * Sets custom output writer, defaults to os.Stdout
 * @param output - Writer for program output
 */
func WithOutput(output io.Writer) ProgramOption

/**
 * Sets custom input reader, defaults to os.Stdin
 * Pass nil to disable input entirely
 * @param input - Reader for program input
 */
func WithInput(input io.Reader) ProgramOption

/**
 * Opens a new TTY for input instead of using stdin
 * Useful when stdin is redirected or piped
 */
func WithInputTTY() ProgramOption

/**
 * Sets environment variables for the program
 * Useful for remote sessions (SSH) to pass environment
 * @param env - Environment variables slice
 */
func WithEnvironment(env []string) ProgramOption
```

### Display Options

```go { .api }
/**
 * Starts program in alternate screen buffer (full window mode)
 * Screen automatically restored when program exits
 */
func WithAltScreen() ProgramOption

/**
 * Disables the renderer for non-TUI mode
 * Output behaves like regular command-line tool
 */
func WithoutRenderer() ProgramOption

/**
 * Sets custom FPS for renderer (default 60, max 120)
 * @param fps - Target frames per second
 */
func WithFPS(fps int) ProgramOption

/**
 * Deprecated: Removes redundant ANSI sequences for smaller output
 * This incurs a noticeable performance hit and will be optimized automatically in future releases
 */
func WithANSICompressor() ProgramOption
```

### Mouse and Input Options

```go { .api }
/**
 * Enables mouse in "cell motion" mode
 * Captures clicks, releases, wheel, and drag events
 */
func WithMouseCellMotion() ProgramOption

/**
 * Enables mouse in "all motion" mode
 * Captures all mouse events including hover (not all terminals support)
 */
func WithMouseAllMotion() ProgramOption

/**
 * Disables bracketed paste mode
 * Bracketed paste is enabled by default
 */
func WithoutBracketedPaste() ProgramOption

/**
 * Enables focus/blur event reporting
 * Sends FocusMsg/BlurMsg when terminal gains/loses focus
 */
func WithReportFocus() ProgramOption
```

### Advanced Options

```go { .api }
/**
 * Sets an event filter function to pre-process messages
 * Filter can modify, replace, or drop messages before Update
 * @param filter - Function that processes messages
 */
func WithFilter(filter func(Model, Msg) Msg) ProgramOption
```

**Filter Usage Example:**

```go
func preventQuit(m tea.Model, msg tea.Msg) tea.Msg {
    // Prevent quitting if there are unsaved changes
    if _, ok := msg.(tea.QuitMsg); ok {
        model := m.(myModel)
        if model.hasUnsavedChanges {
            return nil // Drop the quit message
        }
    }
    return msg
}

p := tea.NewProgram(model{}, tea.WithFilter(preventQuit))
```

## Control Messages

Built-in messages for program control.

```go { .api }
// Control message types
type QuitMsg struct{}
type SuspendMsg struct{}
type ResumeMsg struct{}
type InterruptMsg struct{}

// Control message constructors
func Quit() Msg
func Suspend() Msg
func Interrupt() Msg
```

**Usage in Update function:**

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() == "ctrl+c" {
            return m, tea.Quit()
        }
    case tea.QuitMsg:
        // Program is quitting
        return m, nil
    case tea.SuspendMsg:
        // Program was suspended (Ctrl+Z)
        return m, nil
    case tea.ResumeMsg:
        // Program was resumed
        return m, nil
    }
    return m, nil
}
```