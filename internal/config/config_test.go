package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewDefault_HasExpectedDefaults(t *testing.T) {
	cfg := NewDefault()

	require.Equal(t, "spektacular", cfg.Command)
	require.False(t, cfg.Debug.Enabled)
	require.Equal(t, "timestamp", cfg.Spec.IDMethod)
	require.Equal(t, 0, cfg.Spec.Counter)
}

func TestFromYAMLFile_LoadsAndExpandsEnvVars(t *testing.T) {
	t.Setenv("TEST_CMD", "go run .")

	yaml := `command: "${TEST_CMD}"`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(yaml), 0644)
	require.NoError(t, err)

	cfg, err := FromYAMLFile(path)
	require.NoError(t, err)
	require.Equal(t, "go run .", cfg.Command)
	require.Equal(t, "timestamp", cfg.Spec.IDMethod)
	require.Equal(t, 0, cfg.Spec.Counter)
}

func TestFromYAMLFile_MissingSpecConfigUsesDefaults(t *testing.T) {
	yaml := `command: "go run ."`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(yaml), 0644)
	require.NoError(t, err)

	cfg, err := FromYAMLFile(path)
	require.NoError(t, err)
	require.Equal(t, "go run .", cfg.Command)
	require.Equal(t, "timestamp", cfg.Spec.IDMethod)
	require.Equal(t, 0, cfg.Spec.Counter)
}

func TestFromYAMLFile_UnknownSpecIDMethodReturnsError(t *testing.T) {
	yaml := `spec:
  id_method: unsupported`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(yaml), 0644)
	require.NoError(t, err)

	_, err = FromYAMLFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "spec.id_method")
}

func TestFromYAMLFile_MissingFile_ReturnsError(t *testing.T) {
	_, err := FromYAMLFile("/nonexistent/path/config.yaml")
	require.Error(t, err)
}

func TestToYAMLFile_RoundTrip(t *testing.T) {
	cfg := NewDefault()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	err := cfg.ToYAMLFile(path)
	require.NoError(t, err)

	loaded, err := FromYAMLFile(path)
	require.NoError(t, err)
	require.Equal(t, cfg.Command, loaded.Command)
	require.Equal(t, cfg.Debug.Enabled, loaded.Debug.Enabled)
	require.Equal(t, cfg.Spec.IDMethod, loaded.Spec.IDMethod)
	require.Equal(t, cfg.Spec.Counter, loaded.Spec.Counter)
}
