# Rich TUI - Research Notes

## Specification Analysis

### Original Requirements
- Display structured output from the agent in a readable format
- Enable navigation scrolling
- Allow interaction with the agent (asking for clarifications, additional information)

### Implicit Requirements
- Must integrate with the existing `plan` command workflow in `plan.py`
- Must support streaming output (Claude outputs line-by-line JSONL)
- Must render markdown content from agent responses
- Must handle the `<!--QUESTION:...-->` marker detection and present interactive Q&A
- Must support session resumption (answers sent back to agent)
- Should support emojis and rich formatting

### Constraints Identified
- Python >=3.12 required
- Current CLI uses Click framework
- Agent output is stream-JSON (JSONL), not raw text
- Questions use HTML comment markers embedded in text content
- The existing `run_plan()` loop in `plan.py` currently uses `click.echo()` for all output

## Research Process

### Sub-agents Spawned
1. **Codebase Explorer** - Full codebase structure, all modules, dependencies, integration points
2. **TUI Library Researcher** - Compared Textual, Rich, urwid, prompt_toolkit

### Files Examined

| File | Lines | Summary |
|------|-------|---------|
| `src/spektacular/cli.py` | 87 | Click CLI with init, run, new, plan commands |
| `src/spektacular/config.py` | 126 | Pydantic config models including AgentConfig |
| `src/spektacular/plan.py` | 113 | Plan orchestration - main integration target |
| `src/spektacular/runner.py` | 146 | Claude subprocess runner, event parsing, question detection |
| `src/spektacular/spec.py` | 73 | Spec creation from templates |
| `src/spektacular/init.py` | 80 | Project initialization |
| `src/spektacular/__init__.py` | 5 | Version declaration |
| `tests/test_plan.py` | 55 | Tests for plan module |
| `tests/test_runner.py` | 156 | Tests for runner module |
| `tests/test_config.py` | 37 | Tests for config module |
| `pyproject.toml` | 29 | Project dependencies and config |
| `.spektacular/plans/1_plan_mode/plan.md` | 493 | Reference plan format |

### Patterns Discovered

#### Event Processing Loop (`plan.py:85-105`)
The main loop iterates over `run_claude()` events, echoing text and detecting questions. This is the primary integration point -- the TUI must replace this loop with its own rendering pipeline.

```python
# Current pattern (plan.py:85-105)
while True:
    questions_found = []
    for event in run_claude(current_prompt, config, session_id, project_path):
        if event.session_id:
            session_id = event.session_id
        if text := event.text_content:
            click.echo(text)  # <-- Replace with TUI rendering
            detected = detect_questions(text)
            if detected:
                questions_found.extend(detected)
        if event.is_result:
            ...
```

#### Question Presentation (`plan.py:32-55`)
Currently uses `click.echo()` and `click.prompt()` for Q&A. The TUI needs to present these as interactive widgets.

#### ClaudeEvent Structure (`runner.py:16-47`)
Events have `type` (system/assistant/result), with text content extracted from assistant message blocks. The TUI should handle different event types differently (e.g., show progress for tool_use, render markdown for text).

## Key Findings

### TUI Library Comparison

| Feature | Textual | Rich | urwid | prompt_toolkit |
|---------|---------|------|-------|----------------|
| Markdown rendering | Built-in + streaming (v4.0) | Built-in (static) | None | None |
| Scrollable views | Full support | None | ListBox only | Limited |
| Interactive widgets | Full set | Basic prompts | Low-level | Excellent prompts |
| Emoji/Unicode | Excellent | Excellent | Limited | Good |
| Streaming support | Purpose-built (v4.0) | Live display (workaround) | Manual | Not designed for |
| Integration effort | Moderate (full-screen app) | Trivial (print-based) | Poor | Good |

### Architecture Insights

1. **Textual v4.0 "The Streaming Release"** (July 2025) added `Markdown.append()` specifically for LLM streaming output. This is purpose-built for our use case.

2. **Textual is built on Rich**, so all Rich renderables work inside Textual widgets. No compatibility concerns.

3. **Full-screen app model** means the TUI takes over the terminal. The existing Click command can launch the Textual app, which then manages its own event loop.

4. **Async architecture** - Textual uses asyncio internally. The `run_claude()` generator (which blocks on subprocess stdout) should run in a worker thread to avoid blocking the UI event loop.

5. **Reference implementations** - Elia (ChatGPT TUI), toolong (JSONL viewer) demonstrate the exact pattern needed.

### Existing Implementation Review

The current `plan.py:run_plan()` function handles three concerns:
1. **Orchestration** - building prompts, managing sessions
2. **Event processing** - iterating events, detecting questions
3. **User interaction** - printing output, prompting for answers

The TUI refactor should separate these concerns:
- **Orchestration** stays in `plan.py` (or a new orchestrator)
- **Event processing** becomes a bridge between runner and TUI
- **User interaction** moves entirely to the Textual app

### Testing Infrastructure

Existing tests use `pytest` with `unittest.mock`. Textual provides `textual.testing` with `pilot` for programmatic UI testing (pressing keys, clicking, asserting widget state). This integrates well with pytest.

## Design Decisions

### Decision: Use Textual as TUI Framework
- **Options Considered**: Textual, Rich, Rich+Click, urwid, prompt_toolkit
- **Rationale**: Textual provides all three requirements (markdown rendering, scrollable views, interactive widgets) in a single package with purpose-built streaming markdown support (v4.0). It eliminates the need to combine multiple libraries.
- **Trade-offs**: Higher learning curve than Rich alone. Full-screen app model requires restructuring the plan loop. Adds a significant dependency (~50+ transitive deps).

### Decision: Full-Screen TUI Application
- **Options Considered**: (a) Full-screen Textual app, (b) Inline Rich rendering, (c) Hybrid with --tui flag
- **Rationale**: The spec explicitly asks for "navigation scrolling" and "interaction with the agent". Inline rendering cannot support scrolling backwards through output. A full-screen app provides the best UX for long-running plan generation.
- **Trade-offs**: Users lose the ability to see other terminal output during plan generation. The app takes over the terminal.

### Decision: Worker Thread for Claude Process
- **Options Considered**: (a) asyncio subprocess, (b) threading.Thread worker, (c) Textual `run_worker()`
- **Rationale**: Textual's built-in `run_worker()` wraps threading cleanly, provides cancellation, and integrates with the message system. Using asyncio subprocess would work but adds complexity without benefit since Claude's output is line-buffered.
- **Trade-offs**: Worker thread means UI updates must go through Textual's message system (thread-safe).

### Decision: Separate TUI Module
- **Options Considered**: (a) TUI code in plan.py, (b) Separate tui.py module, (c) tui/ package
- **Rationale**: A single `tui.py` module keeps things simple for the initial implementation. The TUI app class, widgets, and event handlers are cohesive and belong together. Can be split into a package later if it grows beyond ~400 lines.
- **Trade-offs**: Single file may get long with CSS, widgets, and event handlers. But premature splitting adds navigation overhead.

## Code Examples & Patterns

### Textual Streaming Markdown (v4.0+)
```python
# File: Textual documentation example
from textual.app import App, ComposeResult
from textual.widgets import Markdown

class StreamApp(App):
    def compose(self) -> ComposeResult:
        yield Markdown(id="output")

    def on_mount(self) -> None:
        self.run_worker(self.stream_content)

    async def stream_content(self) -> None:
        md = self.query_one("#output", Markdown)
        for chunk in get_chunks():
            md.append(chunk)  # Efficient incremental rendering
```

### Textual Worker Pattern
```python
# File: Textual documentation
from textual.worker import Worker, WorkerState

class MyApp(App):
    def run_agent(self) -> None:
        self.agent_worker = self.run_worker(
            self._process_events,
            thread=True,  # Run in thread for blocking I/O
        )

    def _process_events(self) -> None:
        for event in run_claude(prompt, config):
            self.post_message(AgentEvent(event))  # Thread-safe message
```

### Textual Testing with Pilot
```python
# File: Textual testing documentation
from textual.testing import AppTest

async def test_app_displays_markdown():
    app = PlanTUI()
    async with app.run_test() as pilot:
        await pilot.press("q")  # Simulate keyboard input
        assert app.query_one("#output", Markdown).document is not None
```

## Open Questions (All Must Be Resolved)

All questions resolved during research. The implementation approach is clear.
