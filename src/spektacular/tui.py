"""Textual TUI for plan mode - clean terminal-style output with interactive Q&A."""

import io
from dataclasses import dataclass
from pathlib import Path

from rich.console import Console
from rich.markdown import Markdown as RichMarkdown
from rich.table import Table
from rich.text import Text
from rich.theme import Theme as RichTheme
from textual import work
from textual.app import App, ComposeResult
from textual.containers import Vertical, VerticalScroll
from textual.message import Message
from textual.theme import Theme
from textual.widgets import Label, RichLog, Static

from .config import SpektacularConfig
from .plan import load_agent_prompt, load_knowledge, write_plan_output
from .runner import Question, build_prompt, detect_questions, run_claude


# ---------------------------------------------------------------------------
# Palettes
# ---------------------------------------------------------------------------

@dataclass(frozen=True)
class Palette:
    output: str   # assistant text bullet
    answer: str   # user answer prefix
    success: str  # completion bullet
    error: str    # error bullet
    question: str # question number highlight


_PALETTES: dict[str, Palette] = {
    "github-dark": Palette(
        output="#c9d1d9",
        answer="#58a6ff",
        success="#3fb950",
        error="#f85149",
        question="#58a6ff",
    ),
    "dracula": Palette(
        output="#f8f8f2",
        answer="#8be9fd",
        success="#50fa7b",
        error="#ff5555",
        question="#bd93f9",
    ),
    "nord": Palette(
        output="#d8dee9",
        answer="#88c0d0",
        success="#a3be8c",
        error="#bf616a",
        question="#81a1c1",
    ),
    "solarized": Palette(
        output="#839496",
        answer="#268bd2",
        success="#859900",
        error="#dc322f",
        question="#2aa198",
    ),
    "monokai": Palette(
        output="#f8f8f2",
        answer="#66d9e8",
        success="#a6e22e",
        error="#f92672",
        question="#e6db74",
    ),
}

_THEME_ORDER = list(_PALETTES.keys())

# Rich console themes for markdown rendering — controls inline code colour.
_RICH_MARKDOWN_THEMES: dict[str, RichTheme] = {
    "github-dark": RichTheme({
        "markdown.code": "bold #e6edf3 on #161b22",
        "markdown.h1": "bold #58a6ff",
        "markdown.h2": "bold #58a6ff",
        "markdown.h3": "bold #58a6ff",
        "table.header": "bold #c9d1d9",
    }),
    "dracula": RichTheme({
        "markdown.code": "bold #f1fa8c on #44475a",
        "markdown.h1": "bold #bd93f9",
        "markdown.h2": "bold #bd93f9",
        "markdown.h3": "bold #50fa7b",
        "table.header": "bold #f8f8f2",
    }),
    "nord": RichTheme({
        "markdown.code": "bold #ebcb8b on #3b4252",
        "markdown.h1": "bold #81a1c1",
        "markdown.h2": "bold #81a1c1",
        "markdown.h3": "bold #a3be8c",
        "table.header": "bold #d8dee9",
    }),
    "solarized": RichTheme({
        "markdown.code": "bold #b58900 on #073642",
        "markdown.h1": "bold #268bd2",
        "markdown.h2": "bold #268bd2",
        "markdown.h3": "bold #859900",
        "table.header": "bold #93a1a1",
    }),
    "monokai": RichTheme({
        "markdown.code": "bold #e6db74 on #3e3d32",
        "markdown.h1": "bold #a6e22e",
        "markdown.h2": "bold #a6e22e",
        "markdown.h3": "bold #66d9e8",
        "table.header": "bold #f8f8f2",
    }),
}

_TEXTUAL_THEMES: dict[str, Theme] = {
    "github-dark": Theme(
        name="github-dark",
        primary="#58a6ff",
        secondary="#8b949e",
        accent="#f78166",
        foreground="#c9d1d9",
        background="#0d1117",
        surface="#161b22",
        panel="#21262d",
        boost="#30363d",
        dark=True,
    ),
    "dracula": Theme(
        name="dracula",
        primary="#bd93f9",
        secondary="#6272a4",
        accent="#ff79c6",
        foreground="#f8f8f2",
        background="#282a36",
        surface="#343746",
        panel="#44475a",
        boost="#44475a",
        dark=True,
    ),
    "nord": Theme(
        name="nord",
        primary="#81a1c1",
        secondary="#4c566a",
        accent="#88c0d0",
        foreground="#d8dee9",
        background="#2e3440",
        surface="#3b4252",
        panel="#434c5e",
        boost="#4c566a",
        dark=True,
    ),
    "solarized": Theme(
        name="solarized",
        primary="#268bd2",
        secondary="#586e75",
        accent="#2aa198",
        foreground="#839496",
        background="#002b36",
        surface="#073642",
        panel="#073642",
        boost="#586e75",
        dark=True,
    ),
    "monokai": Theme(
        name="monokai",
        primary="#a6e22e",
        secondary="#75715e",
        accent="#66d9e8",
        foreground="#f8f8f2",
        background="#272822",
        surface="#3e3d32",
        panel="#3e3d32",
        boost="#49483e",
        dark=True,
    ),
}


# ---------------------------------------------------------------------------
# Messages
# ---------------------------------------------------------------------------

class AgentOutput(Message):
    """Streaming text chunk from the agent."""
    def __init__(self, text: str) -> None:
        super().__init__()
        self.text = text


class AgentQuestion(Message):
    """Questions detected in agent output requiring user input."""
    def __init__(self, questions: list[Question]) -> None:
        super().__init__()
        self.questions = questions


class AgentComplete(Message):
    """Plan generation finished successfully."""
    def __init__(self, plan_dir: Path) -> None:
        super().__init__()
        self.plan_dir = plan_dir


class AgentError(Message):
    """Agent encountered an error."""
    def __init__(self, error: str) -> None:
        super().__init__()
        self.error = error


class AgentToolUse(Message):
    """A tool call detected in agent output."""
    def __init__(self, tool_name: str, description: str) -> None:
        super().__init__()
        self.tool_name = tool_name
        self.description = description


class AnswerSelected(Message):
    """User selected an answer from the question panel."""
    def __init__(self, answer: str) -> None:
        super().__init__()
        self.answer = answer


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

_TOOL_INPUT_KEYS: dict[str, str] = {
    "Bash": "command",
    "Read": "file_path",
    "Write": "file_path",
    "Edit": "file_path",
    "Glob": "pattern",
    "Grep": "pattern",
    "WebFetch": "url",
    "Task": "description",
    "WebSearch": "query",
}


def _tool_description(name: str, input_data: dict) -> str:
    key = _TOOL_INPUT_KEYS.get(name)
    val = str(input_data.get(key, "") if key else (next(iter(input_data.values()), "") if input_data else ""))
    return (val[:100] + "…") if len(val) > 100 else val


# ---------------------------------------------------------------------------
# Widgets
# ---------------------------------------------------------------------------

class QuestionPanel(Static):
    """Interactive widget for answering structured agent questions via number keys."""

    can_focus = True

    DEFAULT_CSS = """
    QuestionPanel {
        height: auto;
    }
    """

    BINDINGS = [
        ("1", "select(0)", ""),
        ("2", "select(1)", ""),
        ("3", "select(2)", ""),
        ("4", "select(3)", ""),
        ("5", "select(4)", ""),
        ("6", "select(5)", ""),
        ("7", "select(6)", ""),
        ("8", "select(7)", ""),
        ("9", "select(8)", ""),
    ]

    def __init__(self, questions: list[Question], palette: Palette) -> None:
        super().__init__()
        self.questions = questions
        self.palette = palette

    def on_mount(self) -> None:
        self.focus()

    def compose(self) -> ComposeResult:
        p = self.palette
        q = self.questions[0]
        yield Label(f"[bold]{q.header}[/bold]: {q.question}")
        for i, opt in enumerate(q.options):
            yield Label(
                f"  [bold {p.question}]{i + 1}[/bold {p.question}]  {opt['label']}"
                + (f"  [dim]— {opt['description']}[/dim]" if opt.get("description") else "")
            )
        yield Label("[dim]press a number to select[/dim]")

    def action_select(self, idx: int) -> None:
        q = self.questions[0]
        if 0 <= idx < len(q.options):
            self.post_message(AnswerSelected(q.options[idx]["label"]))


# ---------------------------------------------------------------------------
# App
# ---------------------------------------------------------------------------

class PlanTUI(App):
    """Full-screen TUI for plan generation."""

    CSS = """
    Screen {
        layout: vertical;
    }
    #output-scroll {
        height: 1fr;
        padding: 0 2;
    }
    #output {
        height: auto;
    }
    #tool-area {
        height: 1;
        padding: 0 2;
        background: $panel;
        color: $text-muted;
        display: none;
    }
    #question-area {
        height: auto;
        max-height: 40%;
        padding: 1 2 0 2;
        border-top: solid $accent;
    }
    #status {
        height: 1;
        padding: 0 2;
        background: $boost;
        color: $text-muted;
    }
    """

    BINDINGS = [
        ("q", "quit", "Quit"),
        ("t", "cycle_theme", "Theme"),
    ]

    def __init__(
        self,
        spec_path: Path,
        project_path: Path,
        config: SpektacularConfig,
    ) -> None:
        super().__init__()
        self.spec_path = spec_path
        self.project_path = project_path
        self.config = config
        self.plan_dir = project_path / ".spektacular" / "plans" / spec_path.stem
        self._session_id: str | None = None
        self.result_plan_dir: Path | None = None
        self._theme_index = _THEME_ORDER.index("dracula")
        self._pending_questions: list[Question] = []
        self._pending_answers: list[str] = []

        for t in _TEXTUAL_THEMES.values():
            self.register_theme(t)

    @property
    def _palette(self) -> Palette:
        return _PALETTES[_THEME_ORDER[self._theme_index]]

    def compose(self) -> ComposeResult:
        with VerticalScroll(id="output-scroll"):
            yield RichLog(id="output", highlight=True, markup=True)
        yield Static("", id="tool-area")
        yield Vertical(id="question-area")
        yield Static(f"* thinking  {self.spec_path.name}", id="status")

    def on_mount(self) -> None:
        self.theme = "dracula"
        spec_content = self.spec_path.read_text(encoding="utf-8")
        agent_prompt = load_agent_prompt()
        knowledge = load_knowledge(self.project_path)
        prompt = build_prompt(spec_content, agent_prompt, knowledge)
        self._run_agent(prompt)

    def action_cycle_theme(self) -> None:
        self._theme_index = (self._theme_index + 1) % len(_THEME_ORDER)
        name = _THEME_ORDER[self._theme_index]
        self.theme = name
        self.query_one("#status", Static).update(
            f"theme: {name}  (t to cycle)"
        )

    @work(thread=True)
    def _run_agent(self, prompt: str) -> None:
        questions_found: list[Question] = []
        final_result = None

        try:
            for event in run_claude(
                prompt, self.config, self._session_id, self.project_path
            ):
                if event.session_id:
                    self._session_id = event.session_id
                for tool in event.tool_uses:
                    name = tool.get("name", "tool")
                    desc = _tool_description(name, tool.get("input", {}))
                    self.post_message(AgentToolUse(name, desc))
                if text := event.text_content:
                    self.post_message(AgentOutput(text))
                    questions_found.extend(detect_questions(text))
                if event.is_result:
                    if event.is_error:
                        self.post_message(AgentError(event.result_text or "Unknown error"))
                        return
                    final_result = event.result_text
        except Exception as e:
            self.post_message(AgentError(str(e)))
            return

        if questions_found:
            self.post_message(AgentQuestion(questions_found))
        elif final_result:
            write_plan_output(self.plan_dir, final_result)
            self.post_message(AgentComplete(self.plan_dir))
        else:
            self.post_message(AgentError("Agent completed without producing a result"))

    def _render_markdown(self, text: str) -> Text:
        """Render markdown through a palette-aware Rich console."""
        theme_name = _THEME_ORDER[self._theme_index]
        rich_theme = _RICH_MARKDOWN_THEMES[theme_name]
        buf = io.StringIO()
        console = Console(file=buf, force_terminal=True, width=200, theme=rich_theme, highlight=False)
        console.print(RichMarkdown(text), end="")
        return Text.from_ansi(buf.getvalue())

    def on_agent_tool_use(self, message: AgentToolUse) -> None:
        p = self._palette
        tool_area = self.query_one("#tool-area", Static)
        tool_area.display = True
        tool_area.update(
            Text.assemble(
                Text("⚙ ", style=p.answer),
                Text(message.tool_name, style=f"bold {p.answer}"),
                Text("  ", style=""),
                Text(message.description, style="dim"),
            )
        )

    def _clear_tool_area(self) -> None:
        tool_area = self.query_one("#tool-area", Static)
        tool_area.display = False
        tool_area.update("")

    def on_agent_output(self, message: AgentOutput) -> None:
        self._clear_tool_area()
        p = self._palette
        grid = Table.grid(padding=(0, 1))
        grid.add_column(width=1, no_wrap=True)
        grid.add_column()
        grid.add_row(Text("•", style=p.output), self._render_markdown(message.text))
        self.query_one("#output", RichLog).write(grid)

    def on_agent_question(self, message: AgentQuestion) -> None:
        self._pending_questions = list(message.questions)
        self._pending_answers = []
        self._show_next_question()

    def _show_next_question(self) -> None:
        area = self.query_one("#question-area", Vertical)
        area.query("*").remove()
        area.mount(QuestionPanel(self._pending_questions, self._palette))
        self.query_one("#status", Static).update("? waiting for answer")

    def on_answer_selected(self, message: AnswerSelected) -> None:
        if not self._pending_questions:
            return
        p = self._palette
        self._pending_questions.pop(0)
        self._pending_answers.append(message.answer)
        self.query_one("#output", RichLog).write(
            Text(f"> {message.answer}", style=p.answer)
        )
        if self._pending_questions:
            self._show_next_question()
        else:
            self.query_one("#question-area", Vertical).query("*").remove()
            self.query_one("#status", Static).update(
                f"* thinking  {self.spec_path.name}"
            )
            self._run_agent("\n".join(self._pending_answers))

    def on_agent_complete(self, message: AgentComplete) -> None:
        self._clear_tool_area()
        p = self._palette
        self.query_one("#output", RichLog).write(
            Text(f"• plan written to {message.plan_dir}/plan.md", style=p.success)
        )
        self.query_one("#status", Static).update("done  press q to exit")
        self.result_plan_dir = message.plan_dir

    def on_agent_error(self, message: AgentError) -> None:
        self._clear_tool_area()
        p = self._palette
        self.query_one("#output", RichLog).write(
            Text(f"• {message.error}", style=p.error)
        )
        self.query_one("#status", Static).update("error  press q to exit")


def run_plan_tui(
    spec_path: Path,
    project_path: Path,
    config: SpektacularConfig,
) -> Path | None:
    """Launch the TUI for plan generation. Returns plan directory on success."""
    app = PlanTUI(spec_path, project_path, config)
    app.run()
    return app.result_plan_dir
