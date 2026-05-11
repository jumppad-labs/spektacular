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
	require.Equal(t, "local", cfg.Artifacts.Backend)
	require.Equal(t, "cache/notion", cfg.Artifacts.CacheDir)
	require.Equal(t, "Spec ID", cfg.Artifacts.Notion.SpecIDProperty)
	require.Equal(t, "Plan ID", cfg.Artifacts.Notion.PlanIDProperty)
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
	require.Equal(t, "local", cfg.Artifacts.Backend)
	require.Equal(t, "cache/notion", cfg.Artifacts.CacheDir)
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

func TestFromYAMLFile_UnknownArtifactBackendReturnsError(t *testing.T) {
	yaml := `artifacts:
  backend: unsupported`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(yaml), 0644)
	require.NoError(t, err)

	_, err = FromYAMLFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "artifacts.backend")
}

func TestFromYAMLFile_NotionBackendRequiresExternalSpecIDs(t *testing.T) {
	yaml := `artifacts:
  backend: notion
  cache_dir: cache/notion
  notion:
    specs_data_source: collection://specs
    plans_data_source: collection://plans`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(yaml), 0644)
	require.NoError(t, err)

	_, err = FromYAMLFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "spec.id_method")
	require.Contains(t, err.Error(), "external")
}

func TestFromYAMLFile_NotionBackendRequiresLinkedLocations(t *testing.T) {
	yaml := `spec:
  id_method: external
artifacts:
  backend: notion
  cache_dir: cache/notion
  notion:
    plans_data_source: collection://plans`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(yaml), 0644)
	require.NoError(t, err)

	_, err = FromYAMLFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "artifacts.notion.specs_data_source")
}

func TestFromYAMLFile_NotionBackendRequiresIdentifierProperties(t *testing.T) {
	yaml := `spec:
  id_method: external
artifacts:
  backend: notion
  cache_dir: cache/notion
  notion:
    specs_data_source: collection://specs
    plans_data_source: collection://plans
    spec_id_property: ""
    plan_id_property: "Plan ID"`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(yaml), 0644)
	require.NoError(t, err)

	_, err = FromYAMLFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "artifacts.notion.spec_id_property")
}

func TestFromYAMLFile_NotionBackendLoadsWithRequiredFields(t *testing.T) {
	yaml := `spec:
  id_method: external
artifacts:
  backend: notion
  cache_dir: cache/notion
  notion:
    base_page_url: https://notion.example/base
    specs_data_source: collection://specs
    plans_data_source: collection://plans`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(yaml), 0644)
	require.NoError(t, err)

	cfg, err := FromYAMLFile(path)
	require.NoError(t, err)
	require.Equal(t, "external", cfg.Spec.IDMethod)
	require.Equal(t, "notion", cfg.Artifacts.Backend)
	require.Equal(t, "cache/notion", cfg.Artifacts.CacheDir)
	require.Equal(t, "https://notion.example/base", cfg.Artifacts.Notion.BasePageURL)
	require.Equal(t, "collection://specs", cfg.Artifacts.Notion.SpecsDataSource)
	require.Equal(t, "collection://plans", cfg.Artifacts.Notion.PlansDataSource)
	require.Equal(t, "Spec ID", cfg.Artifacts.Notion.SpecIDProperty)
	require.Equal(t, "Plan ID", cfg.Artifacts.Notion.PlanIDProperty)
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
	require.Equal(t, cfg.Artifacts.Backend, loaded.Artifacts.Backend)
	require.Equal(t, cfg.Artifacts.CacheDir, loaded.Artifacts.CacheDir)
	require.Equal(t, cfg.Artifacts.Notion.SpecIDProperty, loaded.Artifacts.Notion.SpecIDProperty)
	require.Equal(t, cfg.Artifacts.Notion.PlanIDProperty, loaded.Artifacts.Notion.PlanIDProperty)
}
