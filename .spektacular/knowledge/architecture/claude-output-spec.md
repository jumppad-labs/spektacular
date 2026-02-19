# Claude Code Output Specification

## Overview

Claude Code with `--output-format stream-json` produces structured JSONL (JSON Lines) output that can be parsed for orchestration and UI routing. This document specifies the output format based on analysis of the `claude-schedule` project.

## Command Structure

```bash
claude -p "prompt text" \\
  --output-format stream-json \\
  --verbose \\
  --dangerously-skip-permissions \\
  --resume SESSION_ID \\
  --allowedTools "Bash,Read,Write,Edit,WebFetch,WebSearch" \\
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
