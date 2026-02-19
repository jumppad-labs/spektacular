"""Tests for the runner module."""

import json
import subprocess
from unittest.mock import MagicMock, patch

import pytest

from spektacular.runner import (
    ClaudeEvent,
    Question,
    build_prompt,
    detect_questions,
    run_claude,
)
from spektacular.config import SpektacularConfig


class TestClaudeEvent:
    def test_session_id(self):
        event = ClaudeEvent(type="system", data={"session_id": "abc123"})
        assert event.session_id == "abc123"

    def test_is_result(self):
        assert ClaudeEvent(type="result", data={}).is_result is True
        assert ClaudeEvent(type="assistant", data={}).is_result is False

    def test_is_error(self):
        assert ClaudeEvent(type="result", data={"is_error": True}).is_error is True
        assert ClaudeEvent(type="result", data={"is_error": False}).is_error is False

    def test_result_text(self):
        event = ClaudeEvent(type="result", data={"result": "done"})
        assert event.result_text == "done"

    def test_result_text_non_result(self):
        assert ClaudeEvent(type="assistant", data={}).result_text is None

    def test_text_content(self):
        event = ClaudeEvent(
            type="assistant",
            data={"message": {"content": [{"type": "text", "text": "hello"}]}},
        )
        assert event.text_content == "hello"

    def test_text_content_multiple_blocks(self):
        event = ClaudeEvent(
            type="assistant",
            data={"message": {"content": [
                {"type": "text", "text": "hello"},
                {"type": "tool_use", "name": "Bash"},
                {"type": "text", "text": "world"},
            ]}},
        )
        assert event.text_content == "hello\nworld"

    def test_text_content_non_assistant(self):
        assert ClaudeEvent(type="system", data={}).text_content is None


class TestDetectQuestions:
    def test_single_question(self):
        payload = json.dumps({"questions": [
            {"question": "Which approach?", "header": "Approach", "options": [
                {"label": "A", "description": "Option A"},
            ]}
        ]})
        text = f"<!--QUESTION:{payload}-->"
        questions = detect_questions(text)
        assert len(questions) == 1
        assert questions[0].question == "Which approach?"
        assert questions[0].header == "Approach"
        assert questions[0].options[0]["label"] == "A"

    def test_no_questions(self):
        assert detect_questions("no markers here") == []

    def test_invalid_json_skipped(self):
        assert detect_questions("<!--QUESTION:not json-->") == []

    def test_multiple_questions_in_one_marker(self):
        payload = json.dumps({"questions": [
            {"question": "Q1?", "header": "H1", "options": []},
            {"question": "Q2?", "header": "H2", "options": []},
        ]})
        questions = detect_questions(f"<!--QUESTION:{payload}-->")
        assert len(questions) == 2

    def test_question_without_options(self):
        payload = json.dumps({"questions": [{"question": "Q?", "header": "H", "options": []}]})
        questions = detect_questions(f"<!--QUESTION:{payload}-->")
        assert questions[0].options == []


class TestBuildPrompt:
    def test_contains_all_parts(self):
        prompt = build_prompt("spec content", "agent prompt", {"file.md": "knowledge"})
        assert "spec content" in prompt
        assert "agent prompt" in prompt
        assert "knowledge" in prompt
        assert "file.md" in prompt

    def test_empty_knowledge(self):
        prompt = build_prompt("spec", "agent", {})
        assert "spec" in prompt
        assert "agent" in prompt

    def test_ordering(self):
        prompt = build_prompt("SPEC", "AGENT", {"k.md": "KNOWLEDGE"})
        assert prompt.index("AGENT") < prompt.index("KNOWLEDGE") < prompt.index("SPEC")


class TestRunClaude:
    def test_yields_parsed_events(self):
        config = SpektacularConfig()
        events = [
            json.dumps({"type": "system", "session_id": "s1"}),
            json.dumps({"type": "result", "result": "done", "is_error": False}),
            "",
        ]
        mock_process = MagicMock()
        mock_process.stdout = iter(events)
        mock_process.stderr.read.return_value = ""
        mock_process.returncode = 0

        with patch("spektacular.runner.subprocess.Popen", return_value=mock_process):
            result = list(run_claude("prompt", config))

        assert result[0].type == "system"
        assert result[1].type == "result"

    def test_raises_on_nonzero_exit(self):
        config = SpektacularConfig()
        mock_process = MagicMock()
        mock_process.stdout = iter([])
        mock_process.stderr.read.return_value = "error output"
        mock_process.returncode = 1

        with patch("spektacular.runner.subprocess.Popen", return_value=mock_process):
            with pytest.raises(RuntimeError, match="exited with code 1"):
                list(run_claude("prompt", config))

    def test_skips_invalid_json_lines(self):
        config = SpektacularConfig()
        events = ["not json", json.dumps({"type": "result", "result": "ok", "is_error": False})]
        mock_process = MagicMock()
        mock_process.stdout = iter(events)
        mock_process.stderr.read.return_value = ""
        mock_process.returncode = 0

        with patch("spektacular.runner.subprocess.Popen", return_value=mock_process):
            result = list(run_claude("prompt", config))

        assert len(result) == 1
        assert result[0].type == "result"
