# Bob CLI Output Specification

## Overview

Bob (Bob-Shell) is an IBM-developed AI coding agent CLI (v1.0.0). It provides interactive and non-interactive modes for code generation, planning, and general-purpose AI assistance. With `--output-format stream-json` it produces structured JSONL (JSON Lines) output that can be parsed for orchestration and UI routing.

## CLI Reference

### Installation

Binary located at: `/home/nicj/.nvm/versions/node/v24.14.0/bin/bob` (Node.js/npm package)

### Usage

```bash
bob [options] [query..]
```

- **Default mode**: Interactive shell
- **Non-interactive**: Use `-p`/`--prompt` or positional arguments for one-shot execution
- **Hybrid**: Use `-i`/`--prompt-interactive` to run a prompt then continue interactively

### Core Options

| Flag | Description |
|------|-------------|
| `-p, --prompt <text>` | Non-interactive prompt (deprecated; use positional args instead) |
| `-i, --prompt-interactive <text>` | Execute prompt then continue interactively |
| `-m, --model <name>` | Model selection |
| `-o, --output-format <fmt>` | Output format: `text`, `json`, `stream-json` |
| `-r, --resume <id\|"latest"\|index>` | Resume previous session by ID, `"latest"`, or index number |
| `-s, --sandbox` | Run in sandbox mode |
| `-y, --yolo` | Auto-approve all actions |
| `-v, --version` | Show version |
| `-h, --help` | Show help |

### Chat Modes

```bash
bob --chat-mode <mode>
```

| Mode | Purpose |
|------|---------|
| `plan` | Planning mode |
| `code` | Code generation mode |
| `advanced` | Advanced/agentic mode |
| `ask` | Question-answering mode |

### Approval Modes

```bash
bob --approval-mode <mode>
```

| Mode | Behavior |
|------|----------|
| `default` | Prompt for approval on each action |
| `auto_edit` | Auto-approve edit tools, prompt for others |
| `yolo` | Auto-approve all tools (same as `-y`) |

### Tool Control

| Flag | Description |
|------|-------------|
| `--allowed-tools <tools...>` | Tools allowed to run without confirmation |
| `--pre-check-auto-approved` | Pre-check if auto-approved commands are safe |

### Budget Control

| Flag | Description |
|------|-------------|
| `--max-coins <number>` | Stop with exit code 1 if budget exceeded |

### Session Management

| Flag | Description |
|------|-------------|
| `--resume <id>` | Resume session by ID, `"latest"`, or numeric index |
| `--list-sessions` | List available sessions and exit |
| `--delete-session <index>` | Delete a session by index number |

### Workspace Options

| Flag | Description |
|------|-------------|
| `--include-directories <dirs...>` | Additional directories to include (comma-separated or repeated) |
| `--trust` | Specify trust level for current workspace |
| `--instance-id <id>` | Instance ID for this session |
| `--team-id <id>` | Team ID for this session |

### Output Control

| Flag | Description |
|------|-------------|
| `--hide-intermediary-output` | Suppress all output, show only final `attempt_completion` result |
| `--screen-reader` | Enable screen reader mode for accessibility |

### MCP Server Management

```bash
bob mcp add <name> <commandOrUrl> [args...]   # Add a server
bob mcp remove <name>                          # Remove a server
bob mcp list                                   # List configured servers
```

### Extensions

```bash
bob extensions install <source>       # Install from git URL or local path
bob extensions uninstall <names..>    # Uninstall extensions
bob extensions list                   # List installed extensions
bob extensions update [<name>|--all]  # Update extensions
bob extensions disable <name>         # Disable an extension
bob extensions enable <name>          # Enable an extension
bob extensions link <path>            # Link local extension (live updates)
bob extensions new <path> [template]  # Create new extension from boilerplate
bob extensions validate <path>        # Validate a local extension
```

### Authentication

| Flag | Description |
|------|-------------|
| `--logout` | Remove saved credentials |
| `--accept-license` | Accept IBM license agreement |
| `--show-license` | Show full path to license files |

## Non-Interactive Command Structure

For Spektacular integration, use non-interactive mode with streaming JSON:

```bash
bob --output-format stream-json \
  -p "prompt text" \
  -m premium \
  -y \
  --max-coins 100
```

Or using positional prompt (preferred):

```bash
bob --output-format stream-json \
  -y \
  --max-coins 100 \
  "prompt text"
```

## Event Types

### 1. Init Event

Initialize session and provide metadata. Always the first event.

```json
{
  "type": "init",
  "timestamp": "2026-03-03T18:02:54.695Z",
  "session_id": "41648071-0c52-45bb-b1e6-88162745166b",
  "model": "premium"
}
```

**Fields**:
- `type`: Always `"init"`
- `timestamp`: ISO 8601 timestamp
- `session_id`: UUID for session continuity
- `model`: Model tier being used (e.g., `"premium"`, `"standard"`)

**Purpose**: Session initialization, provides session ID for resumption.

### 2. Message Events

Content from user input and assistant responses. These come in two forms:

#### User Message (non-streaming)
```json
{
  "type": "message",
  "timestamp": "2026-03-03T18:02:54.696Z",
  "role": "user",
  "content": "Examine this code repository and provide a summary"
}
```

#### Assistant Message (streaming deltas)
```json
{
  "type": "message",
  "timestamp": "2026-03-03T18:02:57.421Z",
  "role": "assistant",
  "content": "The ",
  "delta": true
}
```

**Fields**:
- `type`: Always `"message"`
- `timestamp`: ISO 8601 timestamp
- `role`: `"user"` or `"assistant"`
- `content`: Message text (full message for user, token chunk for assistant)
- `delta`: Boolean, `true` for streaming tokens (assistant only)

**Purpose**:
- User messages capture the input prompt
- Assistant messages stream token-by-token for real-time display
- Concatenate consecutive assistant deltas to build complete response

### 3. Tool Use Events

When the assistant invokes a tool/function.

```json
{
  "type": "tool_use",
  "timestamp": "2026-03-03T18:03:00.603Z",
  "tool_name": "read_file",
  "tool_id": "tool-1",
  "parameters": {
    "file_path": "/home/user/project/README.md",
    "absolute_path": "/home/user/project/README.md"
  }
}
```

**Fields**:
- `type`: Always `"tool_use"`
- `timestamp`: ISO 8601 timestamp
- `tool_name`: Name of the tool being called (e.g., `"read_file"`, `"write_file"`, `"execute_command"`)
- `tool_id`: Unique identifier for this tool invocation (e.g., `"tool-1"`, `"tool-2"`)
- `parameters`: Object containing tool-specific input parameters

**Common Tools**:
- `read_file`: Read file contents (`file_path`, `absolute_path`)
- `write_file`: Write/create files
- `execute_command`: Run shell commands
- `search_files`: Search for files/content
- `attempt_completion`: Signal task completion

### 4. Tool Result Events

Results from tool execution.

```json
{
  "type": "tool_result",
  "timestamp": "2026-03-03T18:03:00.613Z",
  "tool_id": "tool-1",
  "status": "success",
  "output": ""
}
```

**Fields**:
- `type`: Always `"tool_result"`
- `timestamp`: ISO 8601 timestamp
- `tool_id`: Matches the `tool_id` from corresponding `tool_use` event
- `status`: `"success"` or `"error"`
- `output`: Tool output (may be empty string for file reads where content is internal)

**Error Variant**:
```json
{
  "type": "tool_result",
  "timestamp": "2026-03-03T18:03:00.613Z",
  "tool_id": "tool-1",
  "status": "error",
  "output": "File not found: /path/to/file.txt"
}
```

### 5. Result Event

Final summary of execution with statistics. Always the last event.

```json
{
  "type": "result",
  "timestamp": "2026-03-03T18:03:46.319Z",
  "status": "success",
  "stats": {
    "total_tokens": 116898,
    "input_tokens": 115063,
    "output_tokens": 2835,
    "duration_ms": 51626,
    "session_costs": 0.18708195,
    "max_budget": 100,
    "budget_spend": 0.19,
    "tool_calls": 9
  }
}
```

**Fields**:
- `type`: Always `"result"`
- `timestamp`: ISO 8601 timestamp
- `status`: `"success"` or `"error"`
- `stats`: Execution statistics object
  - `total_tokens`: Combined input + output tokens
  - `input_tokens`: Tokens consumed by prompts/context
  - `output_tokens`: Tokens generated in responses
  - `duration_ms`: Total execution time in milliseconds
  - `session_costs`: Actual cost for this session
  - `max_budget`: Maximum allowed budget
  - `budget_spend`: Amount spent (may differ slightly from session_costs due to rounding)
  - `tool_calls`: Number of tool invocations

## Special Content Patterns

### Thinking Blocks

Bob may include thinking/reasoning in `<thinking>` tags within message content:

```json
{"type":"message","role":"assistant","content":"<thinking>\n","delta":true}
{"type":"message","role":"assistant","content":"Let me analyze...","delta":true}
{"type":"message","role":"assistant","content":"</thinking>\n","delta":true}
```

**Purpose**: Internal reasoning that can be filtered or displayed differently in UI.

### Tool Announcement

Before tool use, Bob streams a human-readable announcement:

```json
{"type":"message","role":"assistant","content":"[using tool read_file: README.md]\n","delta":true}
```

**Pattern**: `[using tool {tool_name}: {brief_description}]\n`

This appears in the message stream before the corresponding `tool_use` event.

### Completion Announcement

For `attempt_completion` tool, includes result summary and cost:

```json
{"type":"message","role":"assistant","content":"[using tool attempt_completion: Successfully completed | Cost: 0.19]\n","delta":true}
```

## Question Detection System

Bob can output structured questions for user interaction using HTML comment markers within message content.

### Question Format

```html
<!--QUESTION:{"questions":[{"question":"Which authentication method should I use?","header":"Auth Method","options":[{"label":"OAuth2","description":"Use OAuth2 with Google/GitHub"},{"label":"JWT","description":"JSON Web Tokens"}]}]}-->
```

### Multiple Choice Questions

```json
{
  "questions": [
    {
      "question": "Which database should I use?",
      "header": "Database",
      "options": [
        {
          "label": "PostgreSQL",
          "description": "Full-featured relational database"
        },
        {
          "label": "SQLite",
          "description": "Lightweight file-based database"
        }
      ]
    }
  ]
}
```

### Free Text Questions

```json
{
  "questions": [
    {
      "question": "What should I name the API endpoint?",
      "header": "Endpoint",
      "options": [],
      "freeText": true
    }
  ]
}
```

## Session Management

### Starting New Session
```bash
bob --output-format json-stream --prompt "prompt"
```

### Resuming Session
```bash
bob --output-format json-stream --prompt "follow-up" --resume SESSION_ID
```

### Answering Questions
```bash
bob --output-format json-stream --prompt "OAuth2" --resume SESSION_ID
```

## Event Sequence

A typical session follows this event sequence:

```
init                    # Session initialization
message (user)          # User prompt
message (assistant)*    # Streaming response tokens (may include thinking)
tool_use               # Tool invocation
tool_result            # Tool output
message (assistant)*    # More streaming tokens
... (repeat tool cycle as needed)
result                 # Final statistics
```

## Key Differences from Claude Code

| Feature | Bob | Claude Code |
|---------|-----|-------------|
| Init event | `type: "init"` with session_id, model | `type: "system", subtype: "init"` |
| User messages | Separate `message` event | Not streamed |
| Streaming | `delta: true` on message events | Content blocks in assistant events |
| Tool calls | Separate `tool_use` events | `tool_use` blocks within assistant message content |
| Tool results | Separate `tool_result` events | `tool_result` blocks within assistant message content |
| Completion | `attempt_completion` tool | `result` event |
| Stats | In `result.stats` object | In `result` event at top level |
| Costs | `session_costs`, `budget_spend` | Not included |

## Parsing Strategy for Spektacular

### 1. Line-by-Line Processing

```go
func parseBobOutput(scanner *bufio.Scanner) <-chan Event {
    events := make(chan Event)
    go func() {
        defer close(events)
        for scanner.Scan() {
            line := scanner.Text()
            if line == "" {
                continue
            }
            var raw map[string]any
            if err := json.Unmarshal([]byte(line), &raw); err != nil {
                continue
            }
            events <- Event{
                Type: raw["type"].(string),
                Data: raw,
            }
        }
    }()
    return events
}
```

### 2. Message Aggregation

```go
func aggregateMessages(events <-chan Event) string {
    var builder strings.Builder
    for event := range events {
        if event.Type == "message" {
            if role, _ := event.Data["role"].(string); role == "assistant" {
                if content, ok := event.Data["content"].(string); ok {
                    builder.WriteString(content)
                }
            }
        }
    }
    return builder.String()
}
```

### 3. Session ID Extraction

```go
func extractSessionID(event Event) string {
    if event.Type == "init" {
        if sid, ok := event.Data["session_id"].(string); ok {
            return sid
        }
    }
    return ""
}
```

### 4. Tool Tracking

```go
type ToolCall struct {
    ID         string
    Name       string
    Parameters map[string]any
    Status     string
    Output     string
}

func trackTools(events <-chan Event) map[string]*ToolCall {
    tools := make(map[string]*ToolCall)
    for event := range events {
        switch event.Type {
        case "tool_use":
            id := event.Data["tool_id"].(string)
            tools[id] = &ToolCall{
                ID:         id,
                Name:       event.Data["tool_name"].(string),
                Parameters: event.Data["parameters"].(map[string]any),
            }
        case "tool_result":
            id := event.Data["tool_id"].(string)
            if tc, ok := tools[id]; ok {
                tc.Status = event.Data["status"].(string)
                tc.Output, _ = event.Data["output"].(string)
            }
        }
    }
    return tools
}
```

### 5. Question Detection

```go
var questionPattern = regexp.MustCompile(`<!--QUESTION:([\s\S]*?)-->`)

func detectQuestions(text string) []Question {
    var questions []Question
    for _, match := range questionPattern.FindAllStringSubmatch(text, -1) {
        var payload struct {
            Questions []Question `json:"questions"`
        }
        if err := json.Unmarshal([]byte(match[1]), &payload); err != nil {
            continue
        }
        questions = append(questions, payload.Questions...)
    }
    return questions
}
```

## Integration with Spektacular Runner Interface

The bob runner should implement the existing `Runner` interface:

```go
type Runner interface {
    Run(opts RunOptions) (<-chan Event, <-chan error)
}
```

Event mapping from bob output to Spektacular events:

| Bob Event | Spektacular Event |
|-----------|-------------------|
| `init` | `Event{Type: "system", Data: {"session_id": ...}}` |
| `message` (assistant) | Accumulate into `Event{Type: "assistant", Data: {...}}` |
| `tool_use` | Include in assistant event content blocks |
| `tool_result` | Include in assistant event content blocks |
| `result` | `Event{Type: "result", Data: {...}}` |

## Error Handling

### Stream-JSON Errors

1. **Result Event Error**: `status: "error"` with error details in output
2. **Tool Result Error**: `status: "error"` with error message
3. **Process Error**: Non-zero exit code with stderr

### Session Errors

- Invalid session ID: Start fresh session
- Budget exceeded: Check `budget_spend` vs `max_budget` in result
- Timeout: Check `duration_ms` in result stats

---

*Generated from analysis of bob CLI json-stream output*
*Last updated: 2026-03-03*
