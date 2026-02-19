"""Tests for the config module."""

import pytest

from spektacular.config import AgentConfig, SpektacularConfig


class TestAgentConfig:
    def test_defaults(self):
        config = AgentConfig()
        assert config.command == "claude"
        assert "--output-format" in config.args
        assert "stream-json" in config.args
        assert "Bash" in config.allowed_tools
        assert config.dangerously_skip_permissions is False

    def test_custom_command(self):
        config = AgentConfig(command="my-agent")
        assert config.command == "my-agent"


class TestSpektacularConfig:
    def test_has_agent_field(self):
        config = SpektacularConfig()
        assert isinstance(config.agent, AgentConfig)
        assert config.agent.command == "claude"

    def test_yaml_round_trip(self, tmp_path):
        config = SpektacularConfig()
        config_path = tmp_path / "config.yaml"
        config.to_yaml_file(config_path)

        loaded = SpektacularConfig.from_yaml_file(config_path)
        assert loaded.agent.command == config.agent.command
        assert loaded.agent.allowed_tools == config.agent.allowed_tools
        assert loaded.agent.dangerously_skip_permissions == config.agent.dangerously_skip_permissions
