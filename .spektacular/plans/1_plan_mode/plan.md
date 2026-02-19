# Plan Mode - Implementation Plan

## Overview
- **Specification**: `.spektacular/specs/1_plan_mode.md`
- **Complexity**: Medium-Complex
- **Dependencies**: click, pyyaml, pydantic, subprocess, json (stdlib)

## Current State Analysis

### What Exists
- **CLI framework** (`src/spektacular/cli.py:1-66`): Click-based CLI with `init`, `run` (stub), and `new` commands
- **Configuration** (`src/spektacular/config.py:1-109`): Pydantic models for api, models, complexity, output settings
- **Init system** (`src/spektacular/init.py:1-80`): Creates `.spektacular/` directory structure including `plans/`
- **Spec creation** (`src/spektacular/spec.py:1-72`): Creates spec files from templates
- **Knowledge base**: `.spektacular/knowledge/` with architecture docs including `claude-output-spec.md`
- **Agent prompts**: `src/spektacular/defaults/agents/planner.md` -- the planner agent system prompt

### What's Missing
- No `plan` CLI command
- No agent configuration in `SpektacularConfig` (no command, args, allowed_tools fields)
- No subprocess runner for spawning the coding agent (claude)
- No stream-JSON parser for Claude Code output
- No question detection / interactive session loop
- No plan file writer (output to `.spektacular/plans/`)
- No test infrastructure (no `tests/` directory, no pytest config)
- **Bug**: `pydantic` is imported in `config.py` but not listed in `pyproject.toml` dependencies

### Integration Points
- CLI (`cli.py`) -- new `plan` command entry point
- Config (`config.py`) -- extended with agent configuration
- Knowledge files -- read and concatenated into agent prompt
- Agent prompt (`defaults/agents/planner.md`) -- system instructions for the claude process
- Claude Code binary -- spawned as subprocess with `--output-format stream-json`

## Implementation Strategy

### High-Level Approach
Build a `plan` command that:
1. Reads the spec file and knowledge base
2. Constructs a prompt combining the planner agent instructions + knowledge + spec
3. Spawns `claude` as a subprocess with `--output-format stream-json`
4. Parses the streaming JSON output line-by-line
5. Detects `<!--QUESTION:...-->` markers and prompts the user interactively
6. Resumes the claude session with answers using `--resume <session-id>`
7. Collects the final output and writes plan files to `.spektacular/plans/<spec-name>/`

### Phasing Strategy
- **Phase 1**: Configuration extension + fix missing pydantic dependency
- **Phase 2**: Core claude process runner with stream-JSON parsing
- **Phase 3**: Plan command with interactive question loop
- **Phase 4**: Testing

---

## Phase 1: Configuration & Dependencies

### Changes Required

- **File**: `pyproject.toml:6-11`
  - **Current**:
    ```toml
    dependencies = [
        "anthropic>=0.82.0",
        "click>=8.3.1",
        "jinja2>=3.1.6",
        "markdown>=3.10.2",
        "pyyaml>=6.0.3",
    ]
    ```
  - **Proposed**:
    ```toml
    dependencies = [
        "anthropic>=0.82.0",
        "click>=8.3.1",
        "jinja2>=3.1.6",
        "markdown>=3.10.2",
        "pydantic>=2.0.0",
        "pyyaml>=6.0.3",
    ]
    ```
  - **Rationale**: `config.py` imports pydantic but it's not declared as a dependency. Fix this before adding more Pydantic models.

- **File**: `src/spektacular/config.py:10-52`
  - **Current**: `SpektacularConfig` has fields for `api`, `models`, `complexity`, `output`
  - **Proposed**: Add `AgentConfig` model and `agent` field:
    ```python
    class AgentConfig(BaseModel):
        """Agent configuration for the coding tool."""
        command: str = Field(
            default="claude",
            description="The coding agent CLI command to execute"
        )
        args: list[str] = Field(
            default_factory=lambda: [
                "--output-format", "stream-json",
                "--verbose",
            ],
            description="Default arguments passed to the agent"
        )
        allowed_tools: list[str] = Field(
            default_factory=lambda: [
                "Bash", "Read", "Write", "Edit",
                "Glob", "Grep", "WebFetch", "WebSearch",
            ],
            description="Tools the agent is allowed to use"
        )
        dangerously_skip_permissions: bool = Field(
            default=False,
            description="Skip permission prompts (use with caution)"
        )
    ```
  - Add to `SpektacularConfig`:
    ```python
    agent: AgentConfig = Field(default_factory=AgentConfig, description="Agent settings")
    ```
  - **Rationale**: The spec says the agent is "configured by the user in .spektacular/config.yaml". This makes the agent command, args, and tool permissions configurable while defaulting to Claude Code.

### Success Criteria
#### Automated Verification
- [ ] `python -c "from spektacular.config import AgentConfig; print(AgentConfig())"` succeeds
- [ ] `python -c "from spektacular.config import SpektacularConfig; c = SpektacularConfig(); print(c.agent.command)"` prints "claude"

---

## Phase 2: Claude Process Runner

### Changes Required

- **New File**: `src/spektacular/runner.py`
  - **Proposed**: Create the core module that manages the claude subprocess:
    ```python
    """Claude Code process runner with stream-JSON parsing."""

    import json
    import subprocess
    import re
    from dataclasses import dataclass, field
    from pathlib import Path
    from typing import Generator

    from .config import SpektacularConfig


    @dataclass
    class ClaudeEvent:
        """Represents a parsed event from Claude's stream-JSON output."""
        type: str
        data: dict = field(default_factory=dict)

        @property
        def session_id(self) -> str | None:
            if self.type == "system":
                return self.data.get("session_id")
            return self.data.get("session_id")

        @property
        def is_result(self) -> bool:
            return self.type == "result"

        @property
        def is_error(self) -> bool:
            return self.is_result and self.data.get("is_error", False)

        @property
        def result_text(self) -> str | None:
            if self.is_result:
                return self.data.get("result")
            return None

        @property
        def text_content(self) -> str | None:
            """Extract text blocks from assistant messages."""
            if self.type != "assistant":
                return None
            message = self.data.get("message", {})
            content = message.get("content", [])
            texts = [block["text"] for block in content
                     if block.get("type") == "text"]
            return "\n".join(texts) if texts else None


    @dataclass
    class Question:
        """A structured question detected in Claude output."""
        question: str
        header: str
        options: list[dict]


    QUESTION_PATTERN = re.compile(r"<!--QUESTION:(.*?)-->", re.DOTALL)


    def detect_questions(text: str) -> list[Question]:
        """Detect structured question markers in text."""
        questions = []
        for match in QUESTION_PATTERN.finditer(text):
            try:
                data = json.loads(match.group(1))
                for q in data.get("questions", []):
                    questions.append(Question(
                        question=q["question"],
                        header=q.get("header", ""),
                        options=q.get("options", []),
                    ))
            except (json.JSONDecodeError, KeyError):
                continue
        return questions


    def build_prompt(
        spec_content: str,
        agent_prompt: str,
        knowledge_contents: dict[str, str],
    ) -> str:
        """Build the combined prompt for the claude process.

        Concatenates: agent instructions + knowledge base + spec.
        """
        parts = [agent_prompt, "\n\n---\n\n# Knowledge Base\n"]
        for filename, content in knowledge_contents.items():
            parts.append(f"\n## {filename}\n{content}\n")
        parts.append(
            f"\n---\n\n# Specification to Plan\n\n{spec_content}")
        return "\n".join(parts)


    def run_claude(
        prompt: str,
        config: SpektacularConfig,
        session_id: str | None = None,
        cwd: Path | None = None,
    ) -> Generator[ClaudeEvent, None, None]:
        """Spawn claude process and yield parsed events."""
        cmd = [config.agent.command, "-p", prompt]
        cmd.extend(config.agent.args)

        if config.agent.allowed_tools:
            cmd.extend(["--allowedTools",
                         ",".join(config.agent.allowed_tools)])

        if config.agent.dangerously_skip_permissions:
            cmd.append("--dangerously-skip-permissions")

        if session_id:
            cmd.extend(["--resume", session_id])

        process = subprocess.Popen(
            cmd,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            cwd=cwd or Path.cwd(),
        )

        for line in process.stdout:
            line = line.strip()
            if not line:
                continue
            try:
                data = json.loads(line)
                yield ClaudeEvent(
                    type=data.get("type", "unknown"), data=data)
            except json.JSONDecodeError:
                continue

        process.wait()
        if process.returncode != 0:
            stderr = process.stderr.read() if process.stderr else ""
            raise RuntimeError(
                f"Claude process exited with code "
                f"{process.returncode}: {stderr}")
    ```
  - **Rationale**: Encapsulates all Claude interaction -- process spawning, stream-JSON parsing, question detection, and prompt construction. Matches patterns from `claude-output-spec.md:194-259`.

### Success Criteria
#### Automated Verification
- [ ] `detect_questions()` correctly parses valid question markers
- [ ] `build_prompt()` concatenates agent + knowledge + spec
- [ ] `ClaudeEvent` properties extract session_id, result, text correctly

---

## Phase 3: Plan Command & Orchestration

### Changes Required

- **New File**: `src/spektacular/plan.py`
  - **Proposed**: Plan command orchestration logic:
    ```python
    """Plan mode orchestration -- transforms specs into plans."""

    import click
    from pathlib import Path

    from .config import SpektacularConfig
    from .runner import build_prompt, detect_questions, run_claude


    def load_knowledge(project_path: Path) -> dict[str, str]:
        """Load all knowledge files from .spektacular/knowledge/."""
        knowledge_dir = project_path / ".spektacular" / "knowledge"
        contents = {}
        if knowledge_dir.exists():
            for md_file in sorted(knowledge_dir.rglob("*.md")):
                relative = md_file.relative_to(knowledge_dir)
                contents[str(relative)] = md_file.read_text(
                    encoding="utf-8")
        return contents


    def load_agent_prompt() -> str:
        """Load the planner agent prompt from package defaults."""
        from importlib.resources import files
        return (
            files("spektacular")
            .joinpath("defaults/agents/planner.md")
            .read_text(encoding="utf-8"))


    def prompt_user_for_answer(questions: list) -> str:
        """Prompt the user in the terminal to answer questions."""
        answers = []
        for q in questions:
            click.echo(f"\n{'='*60}")
            click.echo(f"  {q.header}: {q.question}")
            click.echo(f"{'='*60}")
            if q.options:
                for i, opt in enumerate(q.options, 1):
                    click.echo(
                        f"  {i}. {opt['label']}"
                        f" -- {opt.get('description', '')}")
                click.echo()
                choice = click.prompt(
                    "  Select option (number) or type custom answer",
                    default="1")
                try:
                    idx = int(choice) - 1
                    if 0 <= idx < len(q.options):
                        answers.append(q.options[idx]["label"])
                    else:
                        answers.append(choice)
                except ValueError:
                    answers.append(choice)
            else:
                answer = click.prompt("  Your answer")
                answers.append(answer)
        return "; ".join(answers)


    def write_plan_output(plan_dir: Path, result_text: str) -> None:
        """Write the agent's output to plan directory."""
        plan_dir.mkdir(parents=True, exist_ok=True)
        (plan_dir / "plan.md").write_text(
            result_text, encoding="utf-8")


    def run_plan(
        spec_path: Path,
        project_path: Path,
        config: SpektacularConfig,
    ) -> Path:
        """Execute the plan workflow for a specification."""
        spec_content = spec_path.read_text(encoding="utf-8")
        agent_prompt = load_agent_prompt()
        knowledge = load_knowledge(project_path)
        prompt = build_prompt(spec_content, agent_prompt, knowledge)

        spec_name = spec_path.stem
        plan_dir = project_path / ".spektacular" / "plans" / spec_name

        session_id = None
        final_result = None

        click.echo(f"Starting plan generation for: {spec_path.name}")
        click.echo(f"Output directory: {plan_dir}\n")

        current_prompt = prompt
        while True:
            questions_found = []
            for event in run_claude(
                current_prompt, config, session_id, project_path
            ):
                if event.type == "system" and event.session_id:
                    session_id = event.session_id
                if text := event.text_content:
                    detected = detect_questions(text)
                    if detected:
                        questions_found.extend(detected)
                if event.is_result:
                    if event.is_error:
                        raise RuntimeError(
                            f"Agent error: {event.result_text}")
                    final_result = event.result_text

            if questions_found:
                answer = prompt_user_for_answer(questions_found)
                click.echo("\nResuming agent with answer...")
                current_prompt = answer
                continue
            break

        if final_result:
            write_plan_output(plan_dir, final_result)
            click.echo(f"\nPlan written to: {plan_dir}/plan.md")
        else:
            raise RuntimeError(
                "Agent completed without producing a result")
        return plan_dir
    ```

- **File**: `src/spektacular/cli.py`
  - **Proposed**: Add imports and `plan` command:
    ```python
    from .plan import run_plan
    from .config import SpektacularConfig
    ```
    New command (follows existing `init`/`new` pattern):
    ```python
    @cli.command()
    @click.argument("spec_file",
                    type=click.Path(exists=True, path_type=Path))
    def plan(spec_file):
        """Generate an implementation plan from a specification."""
        try:
            project_path = Path.cwd()
            config_path = project_path / ".spektacular" / "config.yaml"
            if config_path.exists():
                config = SpektacularConfig.from_yaml_file(config_path)
            else:
                config = SpektacularConfig()
            plan_dir = run_plan(spec_file, project_path, config)
            click.echo(f"Plan generated: {plan_dir}")
        except Exception as e:
            click.echo(f"Error generating plan: {e}", err=True)
            raise click.Abort()
    ```

### Success Criteria
#### Automated Verification
- [ ] `spektacular plan --help` shows help text
- [ ] `spektacular plan .spektacular/specs/1_plan_mode.md` spawns claude and writes output

#### Manual Verification
- [ ] Interactive questions appear and can be answered
- [ ] Session resumes correctly after answering
- [ ] Plan file written to `.spektacular/plans/1_plan_mode/plan.md`

---

## Phase 4: Testing

### Changes Required

- **File**: `pyproject.toml` -- Add test dependencies:
    ```toml
    [project.optional-dependencies]
    dev = ["pytest>=8.0.0", "pytest-cov>=4.0.0"]

    [tool.pytest.ini_options]
    testpaths = ["tests"]
    ```

- **New File**: `tests/__init__.py` -- empty
- **New File**: `tests/test_runner.py` -- test detect_questions, build_prompt, ClaudeEvent, run_claude (mocked subprocess)
- **New File**: `tests/test_plan.py` -- test load_knowledge, load_agent_prompt, write_plan_output, run_plan (mocked runner)
- **New File**: `tests/test_config.py` -- test AgentConfig defaults, YAML round-trip with agent section

### Success Criteria
#### Automated Verification
- [ ] `pytest tests/ -v` -- all tests pass
- [ ] `pytest tests/ --cov=spektacular --cov-report=term-missing` -- >80% coverage on new modules

---

## Migration & Rollout

### Data Migration
- None -- greenfield feature

### Breaking Changes
- Adding `agent` to `SpektacularConfig` is additive (has defaults); existing configs remain valid

### Rollback Plan
- Remove `plan` command from `cli.py`, remove `runner.py` and `plan.py`, revert config changes

---

## References
- **Original specification**: `.spektacular/specs/1_plan_mode.md`
- **Claude output spec**: `.spektacular/knowledge/architecture/claude-output-spec.md`
- **Key files**: `cli.py:1-66`, `config.py:1-109`, `init.py:1-80`, `spec.py:1-72`, `planner.md:1-287`, `pyproject.toml:1-16`
- **Patterns**: Click command (`cli.py:19-33`), Pydantic model (`config.py:10-52`), importlib.resources (`init.py:3,11`)
