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
}

func TestFromYAMLFile_UnknownSpecIDMethodReturnsError(t *testing.T) {
	yaml := `spec:
  provider: file
  id_method: unsupported
  config:
    directory: specs`
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
}

// Criterion 1: spec, plan, and knowledge each round-trip a provider plus
// config block through YAML, with knowledge carrying multiple independently
// configured sources.
func TestToYAMLFile_ProviderSectionsRoundTrip(t *testing.T) {
	cfg := NewDefault()
	cfg.Spec = SpecConfig{
		Provider: ProviderFile,
		IDMethod: SpecIDMethodCounter,
		Config: FileSpecConfig{
			Directory: "docs/specs",
		},
	}
	cfg.Plan = PlanConfig{
		Provider: ProviderFile,
		Config:   FilePlanConfig{Directory: "docs/plans"},
	}
	cfg.Knowledge = KnowledgeConfig{
		Sources: []SourceConfig{
			{
				Scope:    "project",
				Provider: ProviderFile,
				Config:   FileKnowledgeConfig{Location: ".spektacular/knowledge"},
			},
			{
				Scope:    "team",
				Provider: ProviderFile,
				Config:   FileKnowledgeConfig{Location: "/shared/team/knowledge"},
			},
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	err := cfg.ToYAMLFile(path)
	require.NoError(t, err)

	loaded, err := FromYAMLFile(path)
	require.NoError(t, err)

	require.Equal(t, cfg.Spec, loaded.Spec)
	require.Equal(t, cfg.Plan, loaded.Plan)
	require.Equal(t, cfg.Knowledge, loaded.Knowledge)
}

// Criterion 2: a config with a section absent yields the documented default.
func TestFromYAMLFile_AbsentProviderSectionsUseDefaults(t *testing.T) {
	yaml := `command: "go run ."`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(yaml), 0644)
	require.NoError(t, err)

	cfg, err := FromYAMLFile(path)
	require.NoError(t, err)

	require.Equal(t, ProviderFile, cfg.Spec.Provider)
	require.Equal(t, DefaultSpecDir, cfg.Spec.Config.Directory)
	require.Equal(t, SpecIDMethodTimestamp, cfg.Spec.IDMethod)
	require.Equal(t, ProviderFile, cfg.Plan.Provider)
	require.Equal(t, DefaultPlanDir, cfg.Plan.Config.Directory)
	require.Len(t, cfg.Knowledge.Sources, 1)
	require.Equal(t, DefaultKnowledgeScope, cfg.Knowledge.Sources[0].Scope)
	require.Equal(t, ProviderFile, cfg.Knowledge.Sources[0].Provider)
	require.Equal(t, DefaultKnowledgeLocation, cfg.Knowledge.Sources[0].Config.Location)
}

// Criterion 2: an empty knowledge config synthesises exactly one default
// project source via WithDefaults.
func TestKnowledgeConfig_WithDefaultsSynthesisesProjectSource(t *testing.T) {
	knowledge := KnowledgeConfig{}.WithDefaults("/some/root")

	require.Len(t, knowledge.Sources, 1)
	src := knowledge.Sources[0]
	require.Equal(t, DefaultKnowledgeScope, src.Scope)
	require.Equal(t, ProviderFile, src.Provider)
	require.Equal(t, filepath.Join("/some/root", DefaultKnowledgeLocation), src.Config.Location)
}

// Criterion 2: WithDefaults leaves an already-configured knowledge config
// unchanged.
func TestKnowledgeConfig_WithDefaultsKeepsConfiguredSources(t *testing.T) {
	configured := KnowledgeConfig{
		Sources: []SourceConfig{
			{
				Scope:    "team",
				Provider: ProviderFile,
				Config:   FileKnowledgeConfig{Location: "/shared/knowledge"},
			},
		},
	}

	result := configured.WithDefaults("/some/root")
	require.Equal(t, configured, result)
}

// Criterion 3: an unknown provider is rejected with a clear validation error.
func TestFromYAMLFile_UnknownSpecProviderReturnsError(t *testing.T) {
	yaml := `spec:
  provider: bogus
  config:
    directory: specs`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(yaml), 0644)
	require.NoError(t, err)

	_, err = FromYAMLFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "spec.provider")
}

// Criterion 3: an empty required config field is rejected.
func TestFromYAMLFile_EmptySpecDirectoryReturnsError(t *testing.T) {
	yaml := `spec:
  provider: file
  config:
    directory: ""`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(yaml), 0644)
	require.NoError(t, err)

	_, err = FromYAMLFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "spec.config.directory")
}

// Criterion 3: a knowledge source missing its required location is rejected.
func TestKnowledgeConfig_ValidateRejectsMissingLocation(t *testing.T) {
	knowledge := KnowledgeConfig{
		Sources: []SourceConfig{
			{Scope: "project", Provider: ProviderFile, Config: FileKnowledgeConfig{Location: ""}},
		},
	}

	err := knowledge.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "config.location")
}

// Criterion 3: a duplicate knowledge scope is rejected with a clear error.
func TestKnowledgeConfig_ValidateRejectsDuplicateScope(t *testing.T) {
	knowledge := KnowledgeConfig{
		Sources: []SourceConfig{
			{Scope: "project", Provider: ProviderFile, Config: FileKnowledgeConfig{Location: "/a"}},
			{Scope: "project", Provider: ProviderFile, Config: FileKnowledgeConfig{Location: "/b"}},
		},
	}

	err := knowledge.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "more than once")
}
