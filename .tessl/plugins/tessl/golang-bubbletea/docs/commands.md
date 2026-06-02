# Command System

Asynchronous operations and side effects through commands, including built-in commands for timers, batching, external process execution, and custom async operations.

## Capabilities

### Command Fundamentals

Commands represent I/O operations that run asynchronously and return messages.

```go { .api }
/**
 * Cmd represents an I/O operation that returns a message
 * Commands are returned from Init() and Update() to perform side effects
 * Return nil for no-op commands
 */
type Cmd func() Msg

// Core message interface that all messages implement
type Msg interface{}
```

Commands are the primary way to perform side effects in Bubble Tea:

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() == "r" {
            // Return a command to refresh data
            return m, m.fetchDataCmd()
        }
    case dataFetchedMsg:
        m.data = msg.data
        return m, nil
    }
    return m, nil
}

func (m model) fetchDataCmd() tea.Cmd {
    return func() tea.Msg {
        data := fetchDataFromAPI() // This runs in a goroutine
        return dataFetchedMsg{data: data}
    }
}
```

### Command Batching

Execute multiple commands concurrently with no ordering guarantees.

```go { .api }
/**
 * Batch executes commands concurrently with no ordering guarantees
 * Use for independent operations that can run simultaneously
 * @param cmds - Variable number of commands to execute
 * @returns Single command that runs all provided commands
 */
func Batch(cmds ...Cmd) Cmd

/**
 * BatchMsg is sent when batch execution completes
 * You typically don't need to handle this directly
 */
type BatchMsg []Cmd
```

**Usage Example:**

```go
func (m model) Init() tea.Cmd {
    return tea.Batch(
        m.loadUserCmd(),
        m.loadSettingsCmd(),
        m.checkUpdatesCmd(),
    )
}

func (m model) loadUserCmd() tea.Cmd {
    return func() tea.Msg {
        user := loadUser()
        return userLoadedMsg{user}
    }
}

func (m model) loadSettingsCmd() tea.Cmd {
    return func() tea.Msg {
        settings := loadSettings()
        return settingsLoadedMsg{settings}
    }
}
```

### Command Sequencing

Execute commands one at a time in specified order.

```go { .api }
/**
 * Sequence executes commands one at a time in order
 * Each command waits for the previous to complete
 * @param cmds - Commands to execute in sequence
 * @returns Single command that runs commands sequentially
 */
func Sequence(cmds ...Cmd) Cmd
```

**Usage Example:**

```go
func (m model) saveAndExit() tea.Cmd {
    return tea.Sequence(
        m.saveFileCmd(),
        m.showSavedMessageCmd(),
        tea.Quit(),
    )
}

func (m model) saveFileCmd() tea.Cmd {
    return func() tea.Msg {
        err := saveFile(m.filename, m.content)
        return fileSavedMsg{err: err}
    }
}

func (m model) showSavedMessageCmd() tea.Cmd {
    return func() tea.Msg {
        return showMessageMsg{text: "File saved!"}
    }
}
```

### Timer Commands

Create time-based operations for animations, polling, and scheduled events.

```go { .api }
/**
 * Tick creates a timer that fires once after the specified duration
 * Timer starts when the command is executed (not when created)
 * @param d - Duration to wait before sending message
 * @param fn - Function that returns message when timer fires
 * @returns Command that sends timed message
 */
func Tick(d time.Duration, fn func(time.Time) Msg) Cmd

/**
 * Every creates a timer that syncs with the system clock
 * Useful for synchronized ticking (e.g., every minute on the minute)
 * @param duration - Duration for timer alignment
 * @param fn - Function that returns message when timer fires
 * @returns Command that sends system-clock-aligned message
 */
func Every(duration time.Duration, fn func(time.Time) Msg) Cmd
```

**Usage Examples:**

```go
// Simple tick timer
type tickMsg time.Time

func (m model) Init() tea.Cmd {
    return tea.Tick(time.Second, func(t time.Time) tea.Msg {
        return tickMsg(t)
    })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg.(type) {
    case tickMsg:
        // Update every second
        m.counter++
        return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
            return tickMsg(t)
        })
    }
    return m, nil
}

// System clock alignment
func (m model) startClockSync() tea.Cmd {
    return tea.Every(time.Minute, func(t time.Time) tea.Msg {
        return clockUpdateMsg{time: t}
    })
}
```

### Window and Terminal Commands

Commands for controlling terminal and window properties.

```go { .api }
/**
 * SetWindowTitle sets the terminal window title
 * @param title - New title for the terminal window
 * @returns Command to set window title
 */
func SetWindowTitle(title string) Cmd

/**
 * WindowSize queries the current terminal size
 * Delivers results via WindowSizeMsg
 * Note: Size messages are automatically sent on start and resize
 * @returns Command to query terminal dimensions
 */
func WindowSize() Cmd
```

**Usage Example:**

```go
func (m model) Init() tea.Cmd {
    return tea.Batch(
        tea.SetWindowTitle("My App - " + m.filename),
        tea.WindowSize(),
    )
}

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

### External Process Execution

Execute external processes and programs while suspending the TUI.

```go { .api }
/**
 * ExecCallback is called when command execution completes
 * @param error - Error from command execution (nil if successful)
 * @returns Message to send to update function
 */
type ExecCallback func(error) Msg

/**
 * ExecCommand interface for executable commands
 * Implemented by exec.Cmd and custom command types
 */
type ExecCommand interface {
    Run() error
    SetStdin(io.Reader)
    SetStdout(io.Writer)
    SetStderr(io.Writer)
}

/**
 * Exec executes arbitrary I/O in blocking fashion
 * Program is suspended while command runs
 * @param c - Command implementing ExecCommand interface
 * @param fn - Callback function for completion notification
 * @returns Command that blocks and executes external process
 */
func Exec(c ExecCommand, fn ExecCallback) Cmd

/**
 * ExecProcess executes an *exec.Cmd in blocking fashion
 * Convenience wrapper around Exec for standard commands
 * @param c - Standard library exec.Cmd
 * @param fn - Callback for completion (can be nil)
 * @returns Command that executes external process
 */
func ExecProcess(c *exec.Cmd, fn ExecCallback) Cmd
```

**Usage Examples:**

```go
import "os/exec"

type editorFinishedMsg struct {
    err error
}

func (m model) openEditor() tea.Cmd {
    cmd := exec.Command("vim", m.filename)
    return tea.ExecProcess(cmd, func(err error) tea.Msg {
        return editorFinishedMsg{err: err}
    })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() == "e" {
            return m, m.openEditor()
        }
    case editorFinishedMsg:
        if msg.err != nil {
            m.status = "Editor failed: " + msg.err.Error()
        } else {
            m.status = "File edited successfully"
            return m, m.reloadFileCmd()
        }
        return m, nil
    }
    return m, nil
}

// Simple execution without callback
func (m model) runGitStatus() tea.Cmd {
    cmd := exec.Command("git", "status")
    return tea.ExecProcess(cmd, nil)
}
```

## Custom Command Patterns

### HTTP Requests

Common pattern for making HTTP requests:

```go
type httpResponseMsg struct {
    status int
    body   []byte
    err    error
}

func fetchURL(url string) tea.Cmd {
    return func() tea.Msg {
        resp, err := http.Get(url)
        if err != nil {
            return httpResponseMsg{err: err}
        }
        defer resp.Body.Close()

        body, err := io.ReadAll(resp.Body)
        return httpResponseMsg{
            status: resp.StatusCode,
            body:   body,
            err:    err,
        }
    }
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case httpResponseMsg:
        if msg.err != nil {
            m.error = msg.err.Error()
            return m, nil
        }
        m.data = string(msg.body)
        return m, nil
    }
    return m, nil
}
```

### File I/O Operations

Pattern for file operations:

```go
type fileReadMsg struct {
    filename string
    content  []byte
    err      error
}

type fileWriteMsg struct {
    filename string
    err      error
}

func readFileCmd(filename string) tea.Cmd {
    return func() tea.Msg {
        content, err := os.ReadFile(filename)
        return fileReadMsg{
            filename: filename,
            content:  content,
            err:      err,
        }
    }
}

func writeFileCmd(filename string, content []byte) tea.Cmd {
    return func() tea.Msg {
        err := os.WriteFile(filename, content, 0644)
        return fileWriteMsg{
            filename: filename,
            err:      err,
        }
    }
}
```

### Periodic Tasks

Pattern for recurring operations:

```go
type pollMsg time.Time

func startPolling(interval time.Duration) tea.Cmd {
    return tea.Tick(interval, func(t time.Time) tea.Msg {
        return pollMsg(t)
    })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg.(type) {
    case pollMsg:
        // Do periodic work
        cmd := m.checkStatusCmd()

        // Schedule next poll
        nextPoll := startPolling(30 * time.Second)

        return m, tea.Batch(cmd, nextPoll)
    }
    return m, nil
}
```

### Conditional Commands

Pattern for conditional command execution:

```go
func conditionalCmd(condition bool, cmd tea.Cmd) tea.Cmd {
    if condition {
        return cmd
    }
    return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() == "s" {
            return m, conditionalCmd(
                m.hasUnsavedChanges,
                m.saveFileCmd(),
            )
        }
    }
    return m, nil
}
```

### Long-Running Tasks with Progress

Pattern for tasks that provide progress updates:

```go
type progressMsg struct {
    percent int
    status  string
}

type completedMsg struct {
    result interface{}
    err    error
}

func longRunningTask() tea.Cmd {
    return func() tea.Msg {
        // This would typically be in a separate goroutine
        // sending progress updates via a channel

        for i := 0; i <= 100; i += 10 {
            // In real implementation, send progress via program.Send()
            time.Sleep(100 * time.Millisecond)
        }

        return completedMsg{result: "Task complete!"}
    }
}
```

## Deprecated Functions

```go { .api }
/**
 * Sequentially executes commands but returns first non-nil message
 * Deprecated: Use Sequence instead for better behavior
 */
func Sequentially(cmds ...Cmd) Cmd
```