# Plan Mode - Research Notes

## Specification Analysis

### Original Requirements
1. Read specification markdown file as input
2. Call the configured coding agent with the spec content
3. Parse the agent's response and present to the user in a structured format
4. Integration with the coding agent should be interactive -- agent can ask for clarifications
5. CLI invocation: `spektacular plan <spec-file>`

### Implicit Requirements
- **Session management**: Interactive Q&A requires tracking session IDs and resuming sessions
- **Prompt construction**: The planner agent prompt, knowledge base, and spec must all be combined and sent to the agent
- **Stream parsing**: Claude Code outputs JSONL (stream-json) that must be parsed line-by-line in real-time
- **Question detection**: `<!--QUESTION:...-->` HTML comment markers must be regex-matched from assistant text blocks
- **Error handling**: Agent process failures, malformed JSON, and session errors must be handled gracefully
- **Config extension**: The current config model has no agent section -- needs one for the "configured agent" requirement

### Constraints Identified
- Initial implementation targets Claude Code only (spec: "we are going to assume the configured agent is claude code")
- Agent interaction is via subprocess stdin/stdout (spec: "spawning a process and communicating via stdin/stdout")
- Knowledge files must be available to the agent (spec: "agent should have the information in .spektacular/knowledge at its disposal")

## Research Process

### Sub-agents Spawned
1. **Codebase Explorer** -- Inventoried all Python source files, defaults, and knowledge documents
2. **Architecture Analyst** -- Mapped CLI -> config -> init/spec data flow and identified extension points
3. **Pattern Researcher** -- Analyzed Click command patterns, Pydantic model conventions, importlib.resources usage
4. **Testing Strategist** -- Confirmed no test infrastructure exists; identified pytest as the framework to use

### Files Examined

| File | Lines | Summary |
|------|-------|---------|
| `src/spektacular/cli.py` | 1-66 | Click CLI with `init`, `run` (stub), `new` commands |
| `src/spektacular/config.py` | 1-109 | Pydantic config: api, models, complexity, output; YAML load/save; env var expansion |
| `src/spektacular/init.py` | 1-80 | Creates .spektacular/ directory tree, copies defaults |
| `src/spektacular/spec.py` | 1-72 | Reads spec-template.md, replaces placeholders, writes to specs/ |
| `src/spektacular/__init__.py` | 1-2 | Version string only |
| `pyproject.toml` | 1-16 | Dependencies: anthropic, click, jinja2, markdown, pyyaml. **Missing: pydantic** |
| `src/spektacular/defaults/agents/planner.md` | 1-287 | Full planner agent system prompt |
| `src/spektacular/defaults/conventions.md` | 1-19 | PEP 8, testing, documentation standards |
| `src/spektacular/defaults/spec-template.md` | 1-27 | Spec template with placeholders |
| `src/spektacular/defaults/.gitignore` | 1-8 | Ignores tmp, log, env, pycache |
| `.spektacular/config.yaml` | 1-15 | Generated default config |
| `.spektacular/knowledge/architecture/claude-output-spec.md` | 1-281 | Complete Claude Code stream-JSON output specification |
| `.spektacular/knowledge/learnings/running-claud-cli.md` | 1-12 | Example claude CLI invocations with session resume |
| `.spektacular/knowledge/conventions.md` | 1-19 | Same as defaults/conventions.md |

### Patterns Discovered

**Click Command Pattern** (`cli.py:19-33`):
```python
@cli.command()
@click.option("--flag", is_flag=True, help="...")
def command_name(flag):
    """Docstring shown in --help."""
    try:
        # ... logic ...
        click.echo(f"Success message")
    except Exception as e:
        click.echo(f"Error: {e}", err=True)
        raise click.Abort()
```

**Pydantic Model Pattern** (`config.py:10-12`):
```python
class SomeConfig(BaseModel):
    """Description."""
    field: type = Field(default=value, description="...")
```

**importlib.resources Pattern** (`init.py:3,11`):
```python
from importlib.resources import files
content = files("spektacular").joinpath("defaults/file.md").read_text(encoding="utf-8")
```

**Logic Separation Pattern**: CLI commands in `cli.py` delegate to functions in dedicated modules (`init.py`, `spec.py`). New `plan` logic should follow this with `plan.py` and `runner.py`.

## Key Findings

### Architecture Insights
- The project follows a clean separation: `cli.py` is thin (Click commands) -> delegates to domain modules
- Configuration is Pydantic-based with YAML serialization and env var expansion
- Default files are bundled as package data via `importlib.resources`
- The `.spektacular/` directory is the project-level workspace (created by `init`)

### Existing Implementations
- No similar subprocess-spawning code exists in the project
- The `claude-output-spec.md` contains Python pseudocode for parsing (lines 194-259) that can be adapted
- The `running-claud-cli.md` learning doc shows the exact CLI invocation pattern with `--allowedTools` and `--dangerously-skip-permissions`

### Reusable Components
- `SpektacularConfig.from_yaml_file()` -- config loading
- `importlib.resources` / `files("spektacular")` -- loading bundled defaults
- Click's `click.prompt()` -- for interactive question answering

### Testing Infrastructure
- **No tests exist** -- no `tests/` directory, no pytest config
- `pyproject.toml` has no test dependencies
- Need to set up pytest, add dev dependencies, create test directory

## Questions & Answers

- **Q**: Should agent config go in the main config.yaml or a separate file?
- **A**: Main config.yaml -- follows existing pattern where all config is in one Pydantic model
- **Impact**: Added `AgentConfig` as a nested model in `SpektacularConfig`

- **Q**: Should `plan` be a new command or replace the `run` stub?
- **A**: New command -- the spec explicitly says `spektacular plan <spec-file>`, and `run` may serve a different purpose later
- **Impact**: Added `plan` command alongside existing commands

- **Q**: How should knowledge and agent prompt be passed to claude?
- **A**: Concatenated into a single prompt string via `-p` flag -- simplest approach, avoids file management
- **Impact**: Created `build_prompt()` function in `runner.py`

## Design Decisions

### Decision 1: Separate `runner.py` and `plan.py` modules
- **Options Considered**: (A) Single `plan.py` module, (B) Separate runner + plan, (C) Full agent framework
- **Rationale**: Option B follows the single-responsibility principle. `runner.py` handles subprocess/parsing concerns, `plan.py` handles orchestration/user interaction. This matches the project's existing pattern of focused modules.
- **Trade-offs**: Two files instead of one, but each is cohesive and independently testable.

### Decision 2: Generator-based event streaming
- **Options Considered**: (A) Collect all output then parse, (B) Generator yielding events, (C) Callback-based
- **Rationale**: Generator (B) enables real-time question detection during streaming. Collecting all output (A) would block until claude finishes, preventing interactive Q&A. Callbacks (C) add unnecessary complexity.
- **Trade-offs**: Generator is slightly harder to test than batch processing, but enables the interactive requirement.

### Decision 3: Question loop with session resume
- **Options Considered**: (A) Pipe answers via stdin, (B) Session resume via `--resume`, (C) Separate prompt calls
- **Rationale**: Option B matches the Claude Code API as documented in `claude-output-spec.md:140-151`. The `--resume <session-id>` flag maintains conversation context.
- **Trade-offs**: Requires session ID tracking, but well-documented in the output spec.

### Decision 4: Agent output stored as single plan.md initially
- **Options Considered**: (A) Parse agent output into plan.md/research.md/context.md, (B) Store raw output as plan.md
- **Rationale**: Option B for initial implementation -- the planner agent is instructed to produce structured output, so the raw result is already a plan. Splitting into multiple files can be added later.
- **Trade-offs**: Simpler implementation, but puts the burden on the agent prompt to produce well-structured output.

## Code Examples & Patterns

### Claude CLI Invocation (from learnings)
```bash
# File: .spektacular/knowledge/learnings/running-claud-cli.md:3-5
claude -p "Load and understand the planner agent specification from src/spektacular/defaults/agents/planner.md" --output-format stream-json --verbose
```

### Session Resume with Tools (from learnings)
```bash
# File: .spektacular/knowledge/learnings/running-claud-cli.md:10-11
claude -p "Now use the planner agent workflow to process .spektacular/specs/1_plan_mode.md and create implementation plans" --resume <session-id> --output-format stream-json --verbose --allowedTools "Bash,Read,Write,Edit" --dangerously-skip-permissions
```

### Stream-JSON Parsing (from claude-output-spec.md)
```python
# File: .spektacular/knowledge/architecture/claude-output-spec.md:194-200
def parse_claude_output(lines):
    events = []
    for line in lines:
        if line.strip():
            events.append(json.loads(line))
    return events
```

### Question Detection (from claude-output-spec.md)
```python
# File: .spektacular/knowledge/architecture/claude-output-spec.md:203-209
def detect_question(assistant_content):
    for block in assistant_content:
        if block["type"] == "text":
            if "<!--QUESTION:" in block["text"]:
                return extract_question_json(block["text"])
    return None
```

## Open Questions (All Must Be Resolved)

- None -- all questions have been resolved through research and informed defaults.
