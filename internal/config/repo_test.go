package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Criterion 2: the default repo config seeds exactly the repo's own knowledge
// store and a file-backed changelog.
func TestNewDefaultRepoConfig_SeedsRepoStoreAndChangelog(t *testing.T) {
	cfg := NewDefaultRepoConfig()

	require.Len(t, cfg.Knowledge.Sources, 1)
	src := cfg.Knowledge.Sources[0]
	require.Equal(t, "project", src.Scope)
	require.Equal(t, "file", src.Provider)
	require.Equal(t, ".spektacular/knowledge", src.Config.Location)

	require.Equal(t, "file", cfg.Changelog.Provider)
	require.Equal(t, ".spektacular/changelog", cfg.Changelog.Config.Directory)
}

// Criterion 2: a repo.yaml parses and validates independently of any project
// config — a minimal file in a directory with NO config.yaml loads fine, with
// defaults filling the absent sections.
func TestRepoConfigFromYAMLFile_LoadsWithoutProjectConfig(t *testing.T) {
	dir := t.TempDir()
	// Deliberately no config.yaml anywhere in dir: only repo.yaml exists.
	path := filepath.Join(dir, RepoConfigFileName)
	minimal := "changelog:\n  provider: file\n  config:\n    directory: docs/changelog\n"
	require.NoError(t, os.WriteFile(path, []byte(minimal), 0644))

	cfg, err := RepoConfigFromYAMLFile(path)
	require.NoError(t, err)

	// The configured section is honoured...
	require.Equal(t, "docs/changelog", cfg.Changelog.Config.Directory)
	// ...and the absent knowledge section keeps the seeded default store.
	require.Len(t, cfg.Knowledge.Sources, 1)
	require.Equal(t, "project", cfg.Knowledge.Sources[0].Scope)
	require.Equal(t, ".spektacular/knowledge", cfg.Knowledge.Sources[0].Config.Location)
}

// Criterion 2: a minimal repo config with no descriptive fields set (they are
// all omitempty and zero-valued) contains only knowledge and changelog
// settings — the serialized repo.yaml carries exactly those two top-level
// sections.
func TestRepoConfig_ToYAMLFileWritesOnlyKnowledgeAndChangelog(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, RepoConfigFileName)
	require.NoError(t, NewDefaultRepoConfig().ToYAMLFile(path))

	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	var top map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &top))
	require.Len(t, top, 2)
	require.Contains(t, top, "knowledge")
	require.Contains(t, top, "changelog")
}

// Criterion 2: a repo config round-trips through repo.yaml unchanged.
func TestRepoConfig_ToYAMLFileRoundTrip(t *testing.T) {
	cfg := RepoConfig{
		Knowledge: KnowledgeConfig{
			Sources: []SourceConfig{
				{
					Scope:    "project",
					Provider: ProviderFile,
					Config:   FileKnowledgeConfig{Location: "kb/repo"},
				},
			},
		},
		Changelog: ChangelogConfig{
			Provider: ProviderFile,
			Config:   FileChangelogConfig{Directory: "docs/changelog"},
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, RepoConfigFileName)
	require.NoError(t, cfg.ToYAMLFile(path))

	loaded, err := RepoConfigFromYAMLFile(path)
	require.NoError(t, err)
	require.Equal(t, cfg, loaded)
}

// Criterion 2: a repo config with descriptive metadata (description, role,
// tags, deployment) round-trips through repo.yaml unchanged, alongside the
// existing knowledge and changelog sections.
func TestRepoConfig_ToYAMLFileRoundTripWithDescriptiveFields(t *testing.T) {
	cfg := RepoConfig{
		Description: "Handles order processing and fulfillment.",
		Role:        "backend-service",
		Tags:        []string{"go", "orders", "team-checkout"},
		Deployment:  "kubernetes",
		Knowledge: KnowledgeConfig{
			Sources: []SourceConfig{
				{
					Scope:    "project",
					Provider: ProviderFile,
					Config:   FileKnowledgeConfig{Location: "kb/repo"},
				},
			},
		},
		Changelog: ChangelogConfig{
			Provider: ProviderFile,
			Config:   FileChangelogConfig{Directory: "docs/changelog"},
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, RepoConfigFileName)
	require.NoError(t, cfg.ToYAMLFile(path))

	loaded, err := RepoConfigFromYAMLFile(path)
	require.NoError(t, err)
	require.Equal(t, cfg, loaded)
}

// Criterion 3: a repo knowledge source missing its required location fails
// RepoConfig validation with an error naming the config key.
func TestRepoConfigFromYAMLFile_MissingSourceLocationReturnsError(t *testing.T) {
	body := "knowledge:\n" +
		"  sources:\n" +
		"    - scope: project\n" +
		"      provider: file\n" +
		"      config:\n" +
		"        location: \"\"\n"
	dir := t.TempDir()
	path := filepath.Join(dir, RepoConfigFileName)
	require.NoError(t, os.WriteFile(path, []byte(body), 0644))

	_, err := RepoConfigFromYAMLFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "config.location")
}

// Criterion 3: an unknown changelog provider in repo.yaml fails RepoConfig
// validation with an error naming the config key.
func TestRepoConfigFromYAMLFile_UnknownChangelogProviderReturnsError(t *testing.T) {
	body := "changelog:\n  provider: bogus\n  config:\n    directory: changelog\n"
	dir := t.TempDir()
	path := filepath.Join(dir, RepoConfigFileName)
	require.NoError(t, os.WriteFile(path, []byte(body), 0644))

	_, err := RepoConfigFromYAMLFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "changelog.provider")
}

// Criterion 3: the project and repo knowledge lists validate independently at
// their own levels — an invalid repo source fails RepoConfig validation while
// a valid project config is unaffected, and vice versa.
func TestRepoConfigValidation_IndependentOfProjectConfig(t *testing.T) {
	invalidKnowledge := KnowledgeConfig{
		Sources: []SourceConfig{
			{Scope: "project", Provider: ProviderFile, Config: FileKnowledgeConfig{Location: ""}},
		},
	}
	validKnowledge := KnowledgeConfig{
		Sources: []SourceConfig{
			{Scope: "team", Provider: ProviderFile, Config: FileKnowledgeConfig{Location: "/shared/team/knowledge"}},
		},
	}

	// Invalid repo sources fail at the repo level; a valid project config is
	// unaffected.
	repoCfg := NewDefaultRepoConfig()
	repoCfg.Knowledge = invalidKnowledge
	require.Error(t, repoCfg.Validate())

	projectCfg := NewDefault()
	projectCfg.Name = "testproj"
	projectCfg.Knowledge = validKnowledge
	require.NoError(t, projectCfg.Validate())

	// And vice versa: invalid project sources fail at the project level while
	// a valid repo config is unaffected.
	projectCfg.Knowledge = invalidKnowledge
	require.Error(t, projectCfg.Validate())

	repoCfg.Knowledge = validKnowledge
	require.NoError(t, repoCfg.Validate())
}

// Criterion 2: WithDefaults synthesises the repo's own store rooted at
// repoRoot when no sources are configured.
func TestRepoConfig_WithDefaultsSynthesisesRepoStore(t *testing.T) {
	cfg := RepoConfig{}.WithDefaults("/some/repo")

	require.Len(t, cfg.Knowledge.Sources, 1)
	src := cfg.Knowledge.Sources[0]
	require.Equal(t, DefaultKnowledgeScope, src.Scope)
	require.Equal(t, ProviderFile, src.Provider)
	require.Equal(t, filepath.Join("/some/repo", DefaultKnowledgeLocation), src.Config.Location)
}

// Criterion 2: WithDefaults leaves already-configured repo sources unchanged.
func TestRepoConfig_WithDefaultsKeepsConfiguredSources(t *testing.T) {
	configured := RepoConfig{
		Knowledge: KnowledgeConfig{
			Sources: []SourceConfig{
				{Scope: "project", Provider: ProviderFile, Config: FileKnowledgeConfig{Location: "/elsewhere/kb"}},
			},
		},
		Changelog: NewDefaultRepoConfig().Changelog,
	}

	result := configured.WithDefaults("/some/repo")
	require.Equal(t, configured, result)
}

// Criterion 2: repo.yaml expands ${VAR} environment patterns like config.yaml
// does.
func TestRepoConfigFromYAMLFile_ExpandsEnvVars(t *testing.T) {
	t.Setenv("TEST_REPO_KB", "env/knowledge")

	body := "knowledge:\n" +
		"  sources:\n" +
		"    - scope: project\n" +
		"      provider: file\n" +
		"      config:\n" +
		"        location: \"${TEST_REPO_KB}\"\n"
	dir := t.TempDir()
	path := filepath.Join(dir, RepoConfigFileName)
	require.NoError(t, os.WriteFile(path, []byte(body), 0644))

	cfg, err := RepoConfigFromYAMLFile(path)
	require.NoError(t, err)
	require.Len(t, cfg.Knowledge.Sources, 1)
	require.Equal(t, "env/knowledge", cfg.Knowledge.Sources[0].Config.Location)
}
