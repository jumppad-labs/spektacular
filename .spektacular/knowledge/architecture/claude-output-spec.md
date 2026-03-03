# Claude Code Output Specification

## Overview

Claude Code is Anthropic's official AI coding agent CLI (v2.1.63). It provides interactive and non-interactive modes for code generation, planning, and general-purpose AI assistance. With `--output-format stream-json` it produces structured JSONL (JSON Lines) output that can be parsed for orchestration and UI routing. This document specifies the CLI and output format based on analysis of the `claude-schedule` project.

## CLI Reference

### Installation

```bash
npm install -g @anthropic-ai/claude-code
```

### Usage

```bash
claude [options] [command] [prompt]
```

- **Default mode**: Interactive session
- **Non-interactive**: Use `-p`/`--print` for one-shot execution (prints response and exits)
- **Continue**: Use `-c`/`--continue` to resume the most recent conversation in the current directory

### Core Options

| Flag | Description |
|------|-------------|
| `-p, --print` | Print response and exit (non-interactive, for pipes) |
| `-c, --continue` | Continue the most recent conversation in the current directory |
| `-r, --resume [value]` | Resume a conversation by session ID, or open interactive picker |
| `--session-id <uuid>` | Use a specific session ID (must be valid UUID) |
| `--fork-session` | Create a new session ID when resuming (use with `--resume` or `--continue`) |
| `--model <model>` | Model alias (`sonnet`, `opus`) or full name (`claude-sonnet-4-6`) |
| `--fallback-model <model>` | Auto-fallback when default model is overloaded (only with `--print`) |
| `--effort <level>` | Effort level: `low`, `medium`, `high` |
| `-o, --output-format <fmt>` | Output format: `text`, `json`, `stream-json` (only with `--print`) |
| `--input-format <fmt>` | Input format: `text` (default), `stream-json` (only with `--print`) |
| `--verbose` | Override verbose mode setting from config |
| `-d, --debug [filter]` | Enable debug mode with optional category filtering (e.g., `"api,hooks"`) |
| `--debug-file <path>` | Write debug logs to a specific file path |
| `-v, --version` | Show version |
| `-h, --help` | Show help |

### Permission Modes

| Flag | Description |
|------|-------------|
| `--permission-mode <mode>` | Permission mode: `default`, `acceptEdits`, `bypassPermissions`, `dontAsk`, `plan` |
| `--dangerously-skip-permissions` | Bypass all permission checks (sandbox-only recommended) |
| `--allow-dangerously-skip-permissions` | Enable bypass as an option without enabling by default |

### Tool Control

| Flag | Description |
|------|-------------|
| `--allowedTools, --allowed-tools <tools...>` | Tools allowed without confirmation (e.g., `"Bash(git:*) Edit"`) |
| `--disallowedTools, --disallowed-tools <tools...>` | Tools to deny (e.g., `"Bash(git:*) Edit"`) |
| `--tools <tools...>` | Available tool set: `""` (none), `"default"` (all), or specific names |

### Prompt Customization

| Flag | Description |
|------|-------------|
| `--system-prompt <prompt>` | System prompt for the session |
| `--append-system-prompt <prompt>` | Append to the default system prompt |
| `--json-schema <schema>` | JSON Schema for structured output validation |

### Budget Control

| Flag | Description |
|------|-------------|
| `--max-budget-usd <amount>` | Maximum dollar amount for API calls (only with `--print`) |

### Agent Configuration

| Flag | Description |
|------|-------------|
| `--agent <agent>` | Agent for the current session (overrides `agent` setting) |
| `--agents <json>` | JSON object defining custom agents |

### Workspace Options

| Flag | Description |
|------|-------------|
| `--add-dir <directories...>` | Additional directories to allow tool access to |
| `-w, --worktree [name]` | Create a new git worktree for this session |
| `--tmux` | Create a tmux session for the worktree (requires `--worktree`) |

### Streaming Options

| Flag | Description |
|------|-------------|
| `--include-partial-messages` | Include partial message chunks as they arrive (with `--print` and `stream-json`) |
| `--replay-user-messages` | Re-emit user messages from stdin on stdout (with `stream-json` I/O) |
| `--no-session-persistence` | Don't save sessions to disk (only with `--print`) |

### MCP Server Management

```bash
claude mcp add <name> <commandOrUrl> [args...]   # Add stdio/HTTP server
claude mcp add --transport http <name> <url>      # Add HTTP server explicitly
claude mcp add -e API_KEY=xxx <name> -- <cmd>     # Add with env vars
claude mcp add-json <name> <json>                 # Add from JSON config
claude mcp add-from-claude-desktop                # Import from Claude Desktop
claude mcp get <name>                             # Get server details
claude mcp list                                   # List configured servers
claude mcp remove <name>                          # Remove a server
claude mcp serve                                  # Start Claude Code as MCP server
claude mcp reset-project-choices                  # Reset project server approvals
```

### MCP Configuration File

```bash
claude --mcp-config config.json            # Load MCP servers from file
claude --strict-mcp-config                 # Only use servers from --mcp-config
```

### Plugin Management

```bash
claude plugin install <plugin>             # Install from marketplace
claude plugin uninstall <plugin>           # Uninstall a plugin
claude plugin list                         # List installed plugins
claude plugin update <plugin>              # Update a plugin
claude plugin enable <plugin>              # Enable a disabled plugin
claude plugin disable [plugin]             # Disable an enabled plugin
claude plugin marketplace                  # Manage marketplaces
claude plugin validate <path>              # Validate plugin/manifest
```

### Authentication

```bash
claude auth login                          # Sign in to Anthropic account
claude auth logout                         # Log out
claude auth status                         # Show authentication status
claude setup-token                         # Set up long-lived auth token
```

### Other Commands

```bash
claude agents                              # List configured agents
claude doctor                              # Check auto-updater health
claude install [target]                    # Install native build (stable/latest/version)
claude update                              # Check for and install updates
```

### Miscellaneous Options

| Flag | Description |
|------|-------------|
| `--chrome` / `--no-chrome` | Enable/disable Chrome integration |
| `--ide` | Auto-connect to IDE on startup |
| `--disable-slash-commands` | Disable all skills |
| `--betas <betas...>` | Beta headers for API requests (API key users) |
| `--setting-sources <sources>` | Setting sources to load: `user`, `project`, `local` |
| `--settings <file-or-json>` | Additional settings from file or JSON string |
| `--plugin-dir <paths...>` | Load plugins from directories for this session |
| `--file <specs...>` | File resources to download at startup (`file_id:path`) |
| `--from-pr [value]` | Resume session linked to a PR by number/URL |

## Non-Interactive Command Structure

For Spektacular integration, use non-interactive mode with streaming JSON:

```bash
claude -p "prompt text" \
  --output-format stream-json \
  --verbose \
  --dangerously-skip-permissions \
  --resume SESSION_ID \
  --allowedTools "Bash,Read,Write,Edit,WebFetch,WebSearch" \
  --mcp-config config.json
```

## Event Types

### 1. System Events

Initialize session and provide metadata.

```json
{
  "type": "system",
  "subtype": "init", 
  "session_id": "sess_abc123def456"
}
```

**Purpose**: Session initialization, provides session ID for continuity.

### 2. Assistant Events

Main content from Claude including text, tool calls, and questions.

```json
{
  "type": "assistant",
  "message": {
    "role": "assistant",
    "content": [
      {
        "type": "text",
        "text": "I'll help you implement the authentication feature."
      },
      {
        "type": "tool_use",
        "name": "Read",
        "id": "tool_abc123",
        "input": {
          "file": "config.py"
        }
      }
    ]
  }
}
```

**Content Block Types**:
- `text`: Regular assistant response text
- `tool_use`: Tool/function calls with input parameters
- `tool_result`: Results from tool execution

### 3. Result Events

Final summary of the execution.

```json
{
  "type": "result",
  "result": "Successfully implemented OAuth2 authentication with Google and GitHub providers.",
  "is_error": false
}
```

**Error variant**:
```json
{
  "type": "result", 
  "result": "Failed to modify config.py: Permission denied",
  "is_error": true
}
```

## Question Detection System

Claude Code can output structured questions for user interaction using HTML comment markers.

### Question Format

```html
<!--QUESTION:{"questions":[{"question":"Which authentication method should I use?","header":"Auth Method","options":[{"label":"OAuth2","description":"Use OAuth2 with Google/GitHub"},{"label":"JWT","description":"JSON Web Tokens for stateless auth"}]}]}-->
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
claude -p "prompt" --output-format stream-json
```

### Resuming Session  
```bash
claude -p "follow-up prompt" --resume sess_abc123def456 --output-format stream-json
```

### Answering Questions
```bash
claude -p "OAuth2" --resume sess_abc123def456 --output-format stream-json
```

## MCP Server Integration

### Configuration File Format
```json
{
  "mcpServers": {
    "weather": {
      "type": "http",
      "url": "https://weather-api.example.com/mcp"
    },
    "database": {
      "type": "stdio", 
      "command": "python",
      "args": ["-m", "db_mcp_server"],
      "env": {
        "DB_URL": "postgresql://localhost/mydb"
      }
    }
  }
}
```

### Tool Access
- Default tools: `Bash,Read,Write,Edit,WebFetch,WebSearch`
- MCP tools: `mcp__servername__toolname`
- Combined: `--allowedTools "Bash,Read,Write,mcp__weather__*"`

## Error Handling

### Stream-JSON Errors
1. **Result Event Error**: `is_error: true` with human-readable message
2. **Assistant Text**: Last assistant message before failure
3. **Process Error**: Non-zero exit code with stderr

### Session Errors
- `"No conversation found"`: Session expired, start fresh
- Permission errors: Use `--dangerously-skip-permissions`

## Parsing Strategy for Spektacular

### 1. Line-by-Line Processing
```python
def parse_claude_output(lines):
    events = []
    for line in lines:
        if line.strip():
            events.append(json.loads(line))
    return events
```

### 2. Question Detection
```python
def detect_question(assistant_content):
    for block in assistant_content:
        if block["type"] == "text":
            if "<!--QUESTION:" in block["text"]:
                return extract_question_json(block["text"])
    return None
```

### 3. Session Tracking
```python
def extract_session_id(events):
    for event in events:
        if event.get("type") == "system" and "session_id" in event:
            return event["session_id"]
    return None
```

### 4. UI Routing Points

**Question Events** → Route to appropriate surface:
- GitHub Issues: Create comment with option buttons
- OpenClaw: Interface with interactive buttons  
- CLI: Terminal prompts with number selection
- Discord: Message with reaction options

**Progress Events** → Update UI:
- Tool use events show progress
- Text blocks provide context
- Result events show completion

**Error Events** → Error handling:
- Display error message on appropriate surface
- Provide debugging information
- Offer retry/abort options

## Integration with Spektacular Architecture

### Command Wrapper
```python
def run_claude_with_spec(spec_file, surface="cli"):
    cmd = ["claude", "-p", f"Implement spec: {spec_file}", 
           "--output-format", "stream-json", "--verbose"]
    
    process = subprocess.Popen(cmd, stdout=subprocess.PIPE)
    
    for line in process.stdout:
        event = json.loads(line)
        
        if question := detect_question(event):
            answer = route_question_to_surface(question, surface)
            # Resume with answer...
        
        elif event["type"] == "result":
            return event["result"]
```

### Surface Routing
```python
def route_question_to_surface(question, surface):
    if surface == "github":
        return create_github_comment(question)
    elif surface == "openclaw": 
        return call_openclaw_interface(question)
    elif surface == "cli":
        return prompt_terminal(question)
```

## References

- **Claude Schedule Implementation**: `/home/nicj/code/github.com/nicholasjackson/claude-schedule/internal/executor/claude.go`
- **Test Cases**: `claude_test.go` - Examples of parsing different event types
- **Claude Code Documentation**: https://code.claude.com/docs/

---

*Generated from analysis of claude-schedule project*  
*Last updated: 2026-02-19*
