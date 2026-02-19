# Plan Mode - Context

## Quick Summary
Implement the `spektacular plan <spec-file>` CLI command that reads a specification, spawns a Claude Code subprocess with the planner agent prompt and knowledge base, parses streaming JSON output with interactive question handling, and writes the resulting implementation plan to `.spektacular/plans/<spec-name>/plan.md`.

## Key Files & Locations

### Existing (to modify)
- **CLI Entry Point**: `src/spektacular/cli.py` -- add `plan` command
- **Configuration**: `src/spektacular/config.py` -- add `AgentConfig` model
- **Dependencies**: `pyproject.toml` -- add pydantic, pytest

### New (to create)
- **Process Runner**: `src/spektacular/runner.py` -- Claude subprocess management, stream-JSON parsing, question detection
- **Plan Orchestration**: `src/spektacular/plan.py` -- plan workflow, knowledge loading, user interaction, output writing
- **Tests**: `tests/test_runner.py`, `tests/test_plan.py`, `tests/test_config.py`

### Reference (read-only)
- **Agent Prompt**: `src/spektacular/defaults/agents/planner.md`
- **Output Spec**: `.spektacular/knowledge/architecture/claude-output-spec.md`
- **CLI Learnings**: `.spektacular/knowledge/learnings/running-claud-cli.md`
- **Project Config**: `.spektacular/config.yaml`

## Dependencies

### Code Dependencies (internal)
- `spektacular.config.SpektacularConfig` -- configuration loading
- `spektacular.config.AgentConfig` -- new agent config model
- `importlib.resources` -- loading bundled default agent prompts

### External Dependencies
- `click` -- CLI framework (existing)
- `pydantic` -- config models (existing import, **missing from pyproject.toml**)
- `pyyaml` -- config serialization (existing)
- `subprocess` -- stdlib, spawning claude process
- `json` -- stdlib, parsing stream-JSON
- `re` -- stdlib, question marker detection
- `pytest` -- new dev dependency for testing

### Database Changes
- None

## Environment Requirements

### Configuration Variables
- No new environment variables required
- Existing `ANTHROPIC_API_KEY` used by the claude process itself (not by spektacular directly)

### Prerequisites
- `claude` CLI must be installed and accessible on PATH
- `.spektacular/` directory must exist (created by `spektacular init`)

## Integration Points

### CLI Commands
- **New**: `spektacular plan <spec-file>` -- main entry point for plan generation
- **Existing**: `spektacular init` -- creates the directory structure plan mode writes to
- **Existing**: `spektacular new <name>` -- creates spec files that plan mode consumes

### External Processes
- **Claude Code CLI** -- spawned as subprocess with `--output-format stream-json`
- Communication: prompt via `-p` flag, resume via `--resume <session-id>`
- Tool access: `--allowedTools "Bash,Read,Write,Edit"` (configurable)
- Permissions: `--dangerously-skip-permissions` (configurable, default off)
- Output: JSONL stream on stdout with system/assistant/result events

### File System
- **Reads**: `.spektacular/specs/*.md`, `.spektacular/knowledge/**/*.md`, `src/spektacular/defaults/agents/planner.md`
- **Writes**: `.spektacular/plans/<spec-name>/plan.md`
