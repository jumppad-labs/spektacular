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

// Phase 1.1 criterion 1: a repo.yaml with no source key loads exactly as
// before — Source is empty and ParseSource reports SourceNone.
func TestRepoConfigFromYAMLFile_NoSourceIsNone(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, RepoConfigFileName)
	minimal := "changelog:\n  provider: file\n  config:\n    directory: docs/changelog\n"
	require.NoError(t, os.WriteFile(path, []byte(minimal), 0644))

	cfg, err := RepoConfigFromYAMLFile(path)
	require.NoError(t, err)
	require.Equal(t, "", cfg.Source)

	kind, value, err := cfg.ParseSource(dir)
	require.NoError(t, err)
	require.Equal(t, SourceNone, kind)
	require.Equal(t, "", value)
}

// Phase 1.1 criterion 1: a config whose Source is set serialises exactly
// three top-level keys — source, knowledge, and changelog.
func TestRepoConfig_ToYAMLFileWritesSourceWhenSet(t *testing.T) {
	cfg := NewDefaultRepoConfig()
	cfg.Source = "/srv/code/api"

	dir := t.TempDir()
	path := filepath.Join(dir, RepoConfigFileName)
	require.NoError(t, cfg.ToYAMLFile(path))

	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	var top map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &top))
	require.Len(t, top, 3)
	require.Contains(t, top, "source")
	require.Contains(t, top, "knowledge")
	require.Contains(t, top, "changelog")
	require.Equal(t, "/srv/code/api", top["source"])
}

// Phase 1.1 criterion 1: an unset (empty or whitespace-only) source
// classifies as SourceNone with an empty value and no error.
func TestRepoConfig_ParseSource_Unset(t *testing.T) {
	for _, src := range []string{"", "   ", "\n\t"} {
		kind, value, err := RepoConfig{Source: src}.ParseSource("/cfg")
		require.NoError(t, err)
		require.Equal(t, SourceNone, kind)
		require.Equal(t, "", value)
	}
}

// Phase 1.1 criterion 2: an absolute path classifies as a file source and
// resolves to that same directory, cleaned.
func TestRepoConfig_ParseSource_AbsoluteFile(t *testing.T) {
	code := t.TempDir()
	configDir := t.TempDir()

	kind, value, err := RepoConfig{Source: code}.ParseSource(configDir)
	require.NoError(t, err)
	require.Equal(t, SourceFile, kind)
	require.Equal(t, code, value)

	// A trailing separator is cleaned away; configDir plays no part.
	kind, value, err = RepoConfig{Source: code + string(filepath.Separator)}.ParseSource(configDir)
	require.NoError(t, err)
	require.Equal(t, SourceFile, kind)
	require.Equal(t, code, value)
}

// Phase 1.1 criterion 2: a path relative to the configuration file's
// directory classifies as a file source and resolves against configDir —
// ../code beside a configDir of <parent>/repo resolves to <parent>/code.
func TestRepoConfig_ParseSource_RelativeFile(t *testing.T) {
	parent := t.TempDir()
	configDir := filepath.Join(parent, "repo")
	expected := filepath.Join(parent, "code")

	kind, value, err := RepoConfig{Source: "../code"}.ParseSource(configDir)
	require.NoError(t, err)
	require.Equal(t, SourceFile, kind)
	require.Equal(t, expected, value)

	// A plain child path anchors under configDir as well.
	kind, value, err = RepoConfig{Source: "src"}.ParseSource(configDir)
	require.NoError(t, err)
	require.Equal(t, SourceFile, kind)
	require.Equal(t, filepath.Join(parent, "repo", "src"), value)
}

// Phase 1.1 criterion 2: the file:// form of an absolute and of a relative
// path classifies as a file source and resolves to the same directory as the
// plain form.
func TestRepoConfig_ParseSource_FileScheme(t *testing.T) {
	code := t.TempDir()
	parent := t.TempDir()
	configDir := filepath.Join(parent, "repo")

	kind, value, err := RepoConfig{Source: "file://" + code}.ParseSource(configDir)
	require.NoError(t, err)
	require.Equal(t, SourceFile, kind)
	require.Equal(t, code, value)

	kind, value, err = RepoConfig{Source: "file://../code"}.ParseSource(configDir)
	require.NoError(t, err)
	require.Equal(t, SourceFile, kind)
	require.Equal(t, filepath.Join(parent, "code"), value)
}

// Phase 1.1 criterion 2: a ${VAR} source, plain and in file:// form, is
// expanded by the loader and then classifies as a file source resolving to
// the variable's directory.
func TestRepoConfigFromYAMLFile_SourceEnvVar(t *testing.T) {
	code := t.TempDir()
	t.Setenv("TEST_REPO_SRC", code)

	for _, form := range []string{"${TEST_REPO_SRC}", "file://${TEST_REPO_SRC}"} {
		dir := t.TempDir()
		path := filepath.Join(dir, RepoConfigFileName)
		body := "source: \"" + form + "\"\n"
		require.NoError(t, os.WriteFile(path, []byte(body), 0644))

		cfg, err := RepoConfigFromYAMLFile(path)
		require.NoError(t, err)

		kind, value, err := cfg.ParseSource(dir)
		require.NoError(t, err)
		require.Equal(t, SourceFile, kind, form)
		require.Equal(t, code, value, form)
	}
}

// Phase 1.1 criterion 3: git://, ssh://, http://, https://, and git+ssh://
// values classify as a git source and are returned unchanged.
func TestRepoConfig_ParseSource_GitSchemes(t *testing.T) {
	cases := []string{
		"git://github.com/org/api.git",
		"ssh://git@github.com/org/api.git",
		"http://github.com/org/api.git",
		"https://github.com/org/api.git",
		"git+ssh://git@github.com/org/api.git",
	}
	for _, src := range cases {
		t.Run(src, func(t *testing.T) {
			kind, value, err := RepoConfig{Source: src}.ParseSource("/cfg")
			require.NoError(t, err)
			require.Equal(t, SourceGit, kind)
			require.Equal(t, src, value)
		})
	}
}

// Phase 1.1 criterion 3: an scp-style user@host:path value classifies as a
// git source and is returned unchanged.
func TestRepoConfig_ParseSource_ScpStyle(t *testing.T) {
	kind, value, err := RepoConfig{Source: "git@github.com:org/api.git"}.ParseSource("/cfg")
	require.NoError(t, err)
	require.Equal(t, SourceGit, kind)
	require.Equal(t, "git@github.com:org/api.git", value)
}

// Phase 1.1: a source with a scheme that is neither file:// nor a git
// transport is rejected with an error naming the unsupported scheme.
func TestRepoConfig_ParseSource_UnsupportedScheme(t *testing.T) {
	kind, value, err := RepoConfig{Source: "s3://bucket/code"}.ParseSource("/cfg")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported source scheme")
	require.Contains(t, err.Error(), "s3")
	require.Equal(t, SourceNone, kind)
	require.Equal(t, "", value)
}

// Phase 1.1 criterion 4: a repo config that sets source round-trips through
// repo.yaml without losing it, alongside the other sections.
func TestRepoConfig_ToYAMLFileRoundTripWithSource(t *testing.T) {
	cfg := RepoConfig{
		Description: "Handles order processing and fulfillment.",
		Source:      "git@github.com:org/api.git",
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
	require.Equal(t, "git@github.com:org/api.git", loaded.Source)
}
