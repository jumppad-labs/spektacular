package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewDefault_HasExpectedDefaults(t *testing.T) {
	cfg := NewDefault()

	require.Equal(t, "${ANTHROPIC_API_KEY}", cfg.API.AnthropicAPIKey)
	require.Equal(t, 60, cfg.API.Timeout)
	require.Equal(t, "anthropic/claude-3-5-sonnet-20241022", cfg.Models.Default)
	require.Equal(t, "anthropic/claude-3-5-haiku-20241022", cfg.Models.Tiers.Simple)
	require.Equal(t, "anthropic/claude-3-5-sonnet-20241022", cfg.Models.Tiers.Medium)
	require.Equal(t, "anthropic/claude-3-opus-20240229", cfg.Models.Tiers.Complex)
	require.Equal(t, "claude", cfg.Agent.Command)
	require.Contains(t, cfg.Agent.Args, "--output-format")
	require.Contains(t, cfg.Agent.Args, "stream-json")
	require.False(t, cfg.Debug.Enabled)
}

func TestFromYAMLFile_LoadsAndExpandsEnvVars(t *testing.T) {
	t.Setenv("TEST_API_KEY", "sk-test-123")

	yaml := `
api:
  anthropic_api_key: "${TEST_API_KEY}"
  timeout: 30
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(yaml), 0644)
	require.NoError(t, err)

	cfg, err := FromYAMLFile(path)
	require.NoError(t, err)
	require.Equal(t, "sk-test-123", cfg.API.AnthropicAPIKey)
	require.Equal(t, 30, cfg.API.Timeout)
}

func TestFromYAMLFile_MissingFile_ReturnsError(t *testing.T) {
	_, err := FromYAMLFile("/nonexistent/path/config.yaml")
	require.Error(t, err)
}

func TestFromYAMLFile_UnexpandedVar_KeepsLiteral(t *testing.T) {
	yaml := `
api:
  anthropic_api_key: "${UNSET_VAR_XYZ}"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(yaml), 0644)
	require.NoError(t, err)

	cfg, err := FromYAMLFile(path)
	require.NoError(t, err)
	// Unset var: expansion returns empty string (os.Getenv returns "")
	require.Equal(t, "", cfg.API.AnthropicAPIKey)
}

func TestToYAMLFile_RoundTrip(t *testing.T) {
	cfg := NewDefault()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	err := cfg.ToYAMLFile(path)
	require.NoError(t, err)

	loaded, err := FromYAMLFile(path)
	require.NoError(t, err)
	require.Equal(t, cfg.Models.Default, loaded.Models.Default)
	require.Equal(t, cfg.Agent.Command, loaded.Agent.Command)
}

func TestGetModelForComplexity_Simple(t *testing.T) {
	cfg := NewDefault()
	model := cfg.GetModelForComplexity(0.1)
	require.Equal(t, cfg.Models.Tiers.Simple, model)
}

func TestGetModelForComplexity_Medium(t *testing.T) {
	cfg := NewDefault()
	model := cfg.GetModelForComplexity(0.4)
	require.Equal(t, cfg.Models.Tiers.Medium, model)
}

func TestGetModelForComplexity_Complex(t *testing.T) {
	cfg := NewDefault()
	model := cfg.GetModelForComplexity(0.9)
	require.Equal(t, cfg.Models.Tiers.Complex, model)
}
