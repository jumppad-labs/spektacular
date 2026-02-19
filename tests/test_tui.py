"""Tests for the TUI module."""

from pathlib import Path
from unittest.mock import patch

import pytest
from rich.text import Text
from textual.widgets import Label, RichLog, Static

from spektacular.config import SpektacularConfig
from spektacular.runner import Question
from spektacular.tui import (
    AgentComplete,
    AgentError,
    AgentOutput,
    AgentQuestion,
    AnswerSelected,
    Palette,
    PlanTUI,
    QuestionPanel,
    _PALETTES,
    _RICH_MARKDOWN_THEMES,
    _TEXTUAL_THEMES,
    _THEME_ORDER,
)


@pytest.fixture
def spec_file(tmp_path):
    spec = tmp_path / ".spektacular" / "specs" / "test-spec.md"
    spec.parent.mkdir(parents=True)
    spec.write_text("# Test Spec\n\nA test specification.")
    return spec


@pytest.fixture
def config():
    return SpektacularConfig()


def _make_app(spec_file, config):
    """Create a PlanTUI with run_claude mocked out to prevent subprocess spawning."""
    return PlanTUI(spec_file, spec_file.parent.parent.parent, config)


def _sample_questions(count=1):
    """Build a list of sample Question objects."""
    qs = []
    for i in range(count):
        qs.append(
            Question(
                question=f"Question {i + 1}?",
                header=f"Header{i + 1}",
                options=[
                    {"label": f"Option A{i + 1}", "description": f"Desc A{i + 1}"},
                    {"label": f"Option B{i + 1}", "description": f"Desc B{i + 1}"},
                ],
            )
        )
    return qs


# ---------------------------------------------------------------------------
# Message dataclass tests
# ---------------------------------------------------------------------------

class TestMessages:
    def test_agent_output(self):
        msg = AgentOutput("hello world")
        assert msg.text == "hello world"

    def test_agent_question(self):
        q = Question(question="Which?", header="H", options=[{"label": "A", "description": ""}])
        msg = AgentQuestion([q])
        assert len(msg.questions) == 1
        assert msg.questions[0].question == "Which?"

    def test_agent_complete(self, tmp_path):
        msg = AgentComplete(tmp_path)
        assert msg.plan_dir == tmp_path

    def test_agent_error(self):
        msg = AgentError("something went wrong")
        assert msg.error == "something went wrong"

    def test_answer_selected(self):
        msg = AnswerSelected("Option A")
        assert msg.answer == "Option A"


# ---------------------------------------------------------------------------
# Palette / theme consistency tests
# ---------------------------------------------------------------------------

class TestPalettes:
    def test_all_palettes_have_required_fields(self):
        """Every Palette instance has the five required colour fields."""
        for name, palette in _PALETTES.items():
            assert isinstance(palette, Palette), f"{name} is not a Palette"
            assert palette.output, f"{name} missing output"
            assert palette.answer, f"{name} missing answer"
            assert palette.success, f"{name} missing success"
            assert palette.error, f"{name} missing error"
            assert palette.question, f"{name} missing question"

    def test_theme_order_matches_palettes(self):
        """_THEME_ORDER contains exactly the same keys as _PALETTES."""
        assert set(_THEME_ORDER) == set(_PALETTES.keys())

    def test_rich_themes_match_palettes(self):
        """_RICH_MARKDOWN_THEMES has an entry for every palette."""
        assert set(_RICH_MARKDOWN_THEMES.keys()) == set(_PALETTES.keys())

    def test_textual_themes_match_palettes(self):
        """_TEXTUAL_THEMES has an entry for every palette."""
        assert set(_TEXTUAL_THEMES.keys()) == set(_PALETTES.keys())


# ---------------------------------------------------------------------------
# QuestionPanel widget tests
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_question_panel_renders(spec_file, config):
    questions = [
        Question(
            question="Which approach?",
            header="Approach",
            options=[
                {"label": "Fast", "description": "Quick"},
                {"label": "Careful", "description": "Thorough"},
            ],
        )
    ]
    app = _make_app(spec_file, config)
    with patch("spektacular.tui.run_claude", return_value=iter([])):
        async with app.run_test() as pilot:
            area = app.query_one("#question-area")
            await area.mount(QuestionPanel(questions, _PALETTES["github-dark"]))
            await pilot.pause()

            labels = app.query("Label")
            rendered = [str(label.render()) for label in labels]
            assert any("Fast" in r for r in rendered)
            assert any("Careful" in r for r in rendered)


@pytest.mark.asyncio
async def test_question_panel_shows_hint(spec_file, config):
    """Panel shows 'press a number to select' hint label."""
    questions = _sample_questions(1)
    app = _make_app(spec_file, config)
    with patch("spektacular.tui.run_claude", return_value=iter([])):
        async with app.run_test() as pilot:
            area = app.query_one("#question-area")
            await area.mount(QuestionPanel(questions, _PALETTES["github-dark"]))
            await pilot.pause()

            labels = app.query("Label")
            rendered = [str(label.render()) for label in labels]
            assert any("press a number" in r for r in rendered)


@pytest.mark.asyncio
async def test_question_panel_shows_descriptions(spec_file, config):
    """Options with descriptions show the description text."""
    questions = _sample_questions(1)
    app = _make_app(spec_file, config)
    with patch("spektacular.tui.run_claude", return_value=iter([])):
        async with app.run_test() as pilot:
            area = app.query_one("#question-area")
            await area.mount(QuestionPanel(questions, _PALETTES["github-dark"]))
            await pilot.pause()

            labels = app.query("Label")
            rendered = [str(label.render()) for label in labels]
            assert any("Desc A1" in r for r in rendered)


@pytest.mark.asyncio
async def test_question_panel_option_without_description(spec_file, config):
    """Options without a description don't show a dash separator."""
    questions = [
        Question(
            question="Pick?",
            header="H",
            options=[{"label": "NoDesc"}],
        )
    ]
    app = _make_app(spec_file, config)
    with patch("spektacular.tui.run_claude", return_value=iter([])):
        async with app.run_test() as pilot:
            area = app.query_one("#question-area")
            await area.mount(QuestionPanel(questions, _PALETTES["github-dark"]))
            await pilot.pause()

            labels = app.query("Label")
            rendered = [str(label.render()) for label in labels]
            # "NoDesc" should appear but no dash separator
            assert any("NoDesc" in r for r in rendered)
            # No description text should be present (no " — " separator for this option)
            option_labels = [r for r in rendered if "NoDesc" in r]
            assert len(option_labels) == 1


@pytest.mark.asyncio
async def test_answer_selection_by_keypress(spec_file, config):
    questions = [
        Question(
            question="Pick one",
            header="Choice",
            options=[{"label": "Yes", "description": "Affirmative"}],
        )
    ]
    app = _make_app(spec_file, config)

    with patch("spektacular.tui.run_claude", return_value=iter([])):
        async with app.run_test() as pilot:
            area = app.query_one("#question-area")
            panel = QuestionPanel(questions, _PALETTES["github-dark"])
            await area.mount(panel)
            await pilot.pause()
            panel.focus()
            await pilot.press("1")
            await pilot.pause()


@pytest.mark.asyncio
async def test_out_of_range_keypress_ignored(spec_file, config):
    """Pressing a number beyond the option count does not post AnswerSelected."""
    questions = [
        Question(
            question="Pick?",
            header="H",
            options=[{"label": "Only"}],
        )
    ]
    app = _make_app(spec_file, config)
    received = []

    with patch("spektacular.tui.run_claude", return_value=iter([])):
        async with app.run_test() as pilot:
            # Stage questions through proper flow so on_answer_selected works
            app.on_agent_question(AgentQuestion(list(questions)))
            await pilot.pause()

            panel = app.query_one(QuestionPanel)
            panel.focus()
            # Press "2" but there's only 1 option
            await pilot.press("2")
            await pilot.pause()

            # _pending_questions should still have the question (not popped)
            assert len(app._pending_questions) == 1


# ---------------------------------------------------------------------------
# PlanTUI theme cycling tests
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_cycle_theme_updates_status(spec_file, config):
    """Pressing 't' cycles theme and updates status bar."""
    app = _make_app(spec_file, config)
    with patch("spektacular.tui.run_claude", return_value=iter([])):
        async with app.run_test() as pilot:
            await pilot.pause()
            initial_index = app._theme_index
            await pilot.press("t")
            await pilot.pause()

            assert app._theme_index == (initial_index + 1) % len(_THEME_ORDER)
            status = app.query_one("#status", Static)
            status_text = str(status.render()).lower()
            new_theme = _THEME_ORDER[app._theme_index]
            assert new_theme in status_text


@pytest.mark.asyncio
async def test_cycle_theme_wraps_around(spec_file, config):
    """Cycling through all themes wraps back to the first."""
    app = _make_app(spec_file, config)
    with patch("spektacular.tui.run_claude", return_value=iter([])):
        async with app.run_test() as pilot:
            await pilot.pause()
            start = app._theme_index
            # Press 't' for every theme
            for _ in range(len(_THEME_ORDER)):
                await pilot.press("t")
                await pilot.pause()

            assert app._theme_index == start


# ---------------------------------------------------------------------------
# PlanTUI message handler tests
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_agent_output_renders_in_richlog(spec_file, config):
    """AgentOutput messages appear in the RichLog widget."""
    app = _make_app(spec_file, config)
    with patch("spektacular.tui.run_claude", return_value=iter([])):
        async with app.run_test() as pilot:
            await pilot.pause()
            app.on_agent_output(AgentOutput("hello from agent"))
            await pilot.pause()

            log = app.query_one("#output", RichLog)
            # RichLog should have at least one line written
            assert len(log.lines) > 0


@pytest.mark.asyncio
async def test_agent_complete_updates_status_and_result(spec_file, config):
    """AgentComplete shows 'done' in status and sets result_plan_dir."""
    app = _make_app(spec_file, config)
    plan_dir = spec_file.parent.parent / "plans" / "test-spec"
    with patch("spektacular.tui.run_claude", return_value=iter([])):
        async with app.run_test() as pilot:
            await pilot.pause()
            app.on_agent_complete(AgentComplete(plan_dir))
            await pilot.pause()

            status = app.query_one("#status", Static)
            assert "done" in str(status.render()).lower()
            assert app.result_plan_dir == plan_dir


@pytest.mark.asyncio
async def test_agent_error_updates_ui(spec_file, config):
    app = _make_app(spec_file, config)
    with patch("spektacular.tui.run_claude", return_value=iter([])):
        async with app.run_test() as pilot:
            await pilot.pause()
            app.on_agent_error(AgentError("test error"))
            await pilot.pause()

            status = app.query_one("#status", Static)
            assert "error" in str(status.render()).lower()


@pytest.mark.asyncio
async def test_agent_error_writes_to_richlog(spec_file, config):
    """AgentError also writes the error text to the RichLog."""
    app = _make_app(spec_file, config)
    with patch("spektacular.tui.run_claude", return_value=iter([])):
        async with app.run_test() as pilot:
            await pilot.pause()
            app.on_agent_error(AgentError("detailed failure"))
            await pilot.pause()

            log = app.query_one("#output", RichLog)
            assert len(log.lines) > 0


# ---------------------------------------------------------------------------
# PlanTUI question → answer flow tests
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_question_shows_panel_in_area(spec_file, config):
    """AgentQuestion mounts a QuestionPanel in the question-area."""
    app = _make_app(spec_file, config)
    questions = _sample_questions(1)
    with patch("spektacular.tui.run_claude", return_value=iter([])):
        async with app.run_test() as pilot:
            await pilot.pause()
            app.on_agent_question(AgentQuestion(questions))
            await pilot.pause()

            panels = app.query(QuestionPanel)
            assert len(panels) == 1


@pytest.mark.asyncio
async def test_question_sets_waiting_status(spec_file, config):
    """After showing a question, status shows 'waiting for answer'."""
    app = _make_app(spec_file, config)
    questions = _sample_questions(1)
    with patch("spektacular.tui.run_claude", return_value=iter([])):
        async with app.run_test() as pilot:
            await pilot.pause()
            app.on_agent_question(AgentQuestion(questions))
            await pilot.pause()

            status = app.query_one("#status", Static)
            assert "waiting" in str(status.render()).lower()


@pytest.mark.asyncio
async def test_answer_removes_panel_and_shows_in_log(spec_file, config):
    """Answering the last question removes the panel and shows the answer."""
    app = _make_app(spec_file, config)
    questions = _sample_questions(1)
    with patch("spektacular.tui.run_claude", return_value=iter([])):
        async with app.run_test() as pilot:
            await pilot.pause()
            app.on_agent_question(AgentQuestion(questions))
            await pilot.pause()

            # Answer via the panel
            panel = app.query_one(QuestionPanel)
            panel.focus()
            await pilot.press("1")
            await pilot.pause()

            # Panel should be removed
            panels = app.query(QuestionPanel)
            assert len(panels) == 0

            # Answer should appear in log
            log = app.query_one("#output", RichLog)
            assert len(log.lines) > 0


@pytest.mark.asyncio
async def test_multiple_questions_presented_sequentially(spec_file, config):
    """With 2 questions, answering the first shows the second."""
    app = _make_app(spec_file, config)
    questions = _sample_questions(2)
    with patch("spektacular.tui.run_claude", return_value=iter([])):
        async with app.run_test() as pilot:
            await pilot.pause()
            app.on_agent_question(AgentQuestion(questions))
            await pilot.pause()

            # First question should be shown
            labels = app.query("Label")
            rendered = [str(l.render()) for l in labels]
            assert any("Question 1" in r for r in rendered)

            # Answer first question
            panel = app.query_one(QuestionPanel)
            panel.focus()
            await pilot.press("1")
            await pilot.pause()

            # Second question should now be shown
            labels = app.query("Label")
            rendered = [str(l.render()) for l in labels]
            assert any("Question 2" in r for r in rendered)


@pytest.mark.asyncio
async def test_on_answer_selected_with_empty_pending_is_noop(spec_file, config):
    """on_answer_selected with no pending questions does nothing (bug fix guard)."""
    app = _make_app(spec_file, config)
    with patch("spektacular.tui.run_claude", return_value=iter([])):
        async with app.run_test() as pilot:
            await pilot.pause()
            # Directly call the handler with no pending questions
            app.on_answer_selected(AnswerSelected("stray answer"))
            await pilot.pause()
            # Should not crash; pending lists remain empty
            assert app._pending_questions == []
            assert app._pending_answers == []


# ---------------------------------------------------------------------------
# Markdown rendering tests
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_render_markdown_returns_text(spec_file, config):
    """_render_markdown returns a Rich Text object."""
    app = _make_app(spec_file, config)
    with patch("spektacular.tui.run_claude", return_value=iter([])):
        async with app.run_test() as pilot:
            await pilot.pause()
            result = app._render_markdown("hello **bold** world")
            assert isinstance(result, Text)
            assert "hello" in result.plain
            assert "bold" in result.plain


@pytest.mark.asyncio
async def test_render_markdown_handles_code_block(spec_file, config):
    """_render_markdown handles fenced code blocks without error."""
    app = _make_app(spec_file, config)
    with patch("spektacular.tui.run_claude", return_value=iter([])):
        async with app.run_test() as pilot:
            await pilot.pause()
            result = app._render_markdown("```python\nprint('hi')\n```")
            assert isinstance(result, Text)
            assert "print" in result.plain


# ---------------------------------------------------------------------------
# Quit keybinding test
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_quit_keybinding(spec_file, config):
    app = _make_app(spec_file, config)
    with patch("spektacular.tui.run_claude", return_value=iter([])):
        async with app.run_test() as pilot:
            await pilot.pause()
            await pilot.press("q")
            # App should exit cleanly without raising


# ---------------------------------------------------------------------------
# Initial compose / layout tests
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_initial_layout_has_expected_widgets(spec_file, config):
    """PlanTUI composes output-scroll, question-area, and status widgets."""
    app = _make_app(spec_file, config)
    with patch("spektacular.tui.run_claude", return_value=iter([])):
        async with app.run_test() as pilot:
            await pilot.pause()
            assert app.query_one("#output-scroll") is not None
            assert app.query_one("#output", RichLog) is not None
            assert app.query_one("#question-area") is not None
            assert app.query_one("#status", Static) is not None


@pytest.mark.asyncio
async def test_initial_status_shows_spec_name(spec_file, config):
    """Status bar is initially set with the spec filename."""
    app = _make_app(spec_file, config)
    # Patch _run_agent to prevent the background worker from overwriting status
    with patch.object(PlanTUI, "_run_agent"):
        async with app.run_test() as pilot:
            await pilot.pause()
            status = app.query_one("#status", Static)
            assert "test-spec.md" in str(status.render())


@pytest.mark.asyncio
async def test_default_theme_is_dracula(spec_file, config):
    """PlanTUI starts with the dracula theme."""
    app = _make_app(spec_file, config)
    with patch("spektacular.tui.run_claude", return_value=iter([])):
        async with app.run_test() as pilot:
            await pilot.pause()
            assert _THEME_ORDER[app._theme_index] == "dracula"
