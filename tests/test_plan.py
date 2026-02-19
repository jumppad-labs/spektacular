"""Tests for the plan module."""

from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest

from spektacular.plan import load_knowledge, write_plan_output, load_agent_prompt
from spektacular.config import SpektacularConfig


class TestLoadKnowledge:
    def test_loads_md_files(self, tmp_path):
        knowledge_dir = tmp_path / ".spektacular" / "knowledge"
        knowledge_dir.mkdir(parents=True)
        (knowledge_dir / "notes.md").write_text("content here")

        result = load_knowledge(tmp_path)
        assert "notes.md" in result
        assert result["notes.md"] == "content here"

    def test_missing_knowledge_dir(self, tmp_path):
        result = load_knowledge(tmp_path)
        assert result == {}

    def test_nested_files(self, tmp_path):
        knowledge_dir = tmp_path / ".spektacular" / "knowledge"
        sub = knowledge_dir / "architecture"
        sub.mkdir(parents=True)
        (sub / "overview.md").write_text("arch content")

        result = load_knowledge(tmp_path)
        assert "architecture/overview.md" in result


class TestWritePlanOutput:
    def test_creates_plan_file(self, tmp_path):
        plan_dir = tmp_path / "plans" / "my-spec"
        write_plan_output(plan_dir, "# Plan\n\nContent here")

        assert (plan_dir / "plan.md").exists()
        assert (plan_dir / "plan.md").read_text() == "# Plan\n\nContent here"

    def test_creates_directories(self, tmp_path):
        plan_dir = tmp_path / "deep" / "nested" / "dir"
        write_plan_output(plan_dir, "content")
        assert plan_dir.exists()


class TestLoadAgentPrompt:
    def test_returns_string(self):
        prompt = load_agent_prompt()
        assert isinstance(prompt, str)
        assert len(prompt) > 0
