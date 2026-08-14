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
	require.Equal(t, SpecTriggerThresholdModerate, cfg.SpecTriggerThreshold)
	require.Empty(t, cfg.Name)
	require.Empty(t, cfg.Knowledge.Sources)
}

func TestFromYAMLFile_LoadsAndExpandsEnvVars(t *testing.T) {
	t.Setenv("TEST_CMD", "go run .")

	yaml := `name: testproj
command: "${TEST_CMD}"`
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
	yaml := `name: testproj
command: "go run ."`
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
	yaml := `name: testproj
spec:
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

func TestFromYAMLFile_UnknownSpecTriggerThresholdReturnsError(t *testing.T) {
	yaml := `name: testproj
spec_trigger_threshold: unsupported`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(yaml), 0644)
	require.NoError(t, err)

	_, err = FromYAMLFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "spec_trigger_threshold")
}

func TestFromYAMLFile_MissingFile_ReturnsError(t *testing.T) {
	_, err := FromYAMLFile("/nonexistent/path/config.yaml")
	require.Error(t, err)
}

// Criterion 4: spec/plan/changelog configuration behaviour is unchanged by
// the project/repo split — the default config still round-trips.
func TestToYAMLFile_RoundTrip(t *testing.T) {
	cfg := NewDefault()
	cfg.Name = "testproj"
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
	cfg.Name = "testproj"
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
	cfg.Changelog = ChangelogConfig{
		Provider: ProviderFile,
		Config:   FileChangelogConfig{Directory: "docs/changelog"},
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
	require.Equal(t, cfg.Changelog, loaded.Changelog)
	require.Equal(t, cfg.Knowledge, loaded.Knowledge)
}

// Criterion 2: a config with a section absent yields the documented default.
func TestFromYAMLFile_AbsentProviderSectionsUseDefaults(t *testing.T) {
	yaml := `name: testproj
command: "go run ."`
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
	require.Equal(t, ProviderFile, cfg.Changelog.Provider)
	require.Equal(t, DefaultChangelogDir, cfg.Changelog.Config.Directory)
	// The project-level knowledge list holds only project-owned sources and is
	// empty by default; the repo's own store lives in RepoConfig.
	require.Empty(t, cfg.Knowledge.Sources)
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
	yaml := `name: testproj
spec:
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
	yaml := `name: testproj
spec:
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

// Criterion 3: an unknown changelog provider is rejected with a clear
// validation error.
func TestFromYAMLFile_UnknownChangelogProviderReturnsError(t *testing.T) {
	yaml := `name: testproj
changelog:
  provider: bogus
  config:
    directory: changelog`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(yaml), 0644)
	require.NoError(t, err)

	_, err = FromYAMLFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "changelog.provider")
}

// Criterion 3: an empty required changelog config field is rejected.
func TestFromYAMLFile_EmptyChangelogDirectoryReturnsError(t *testing.T) {
	yaml := `name: testproj
changelog:
  provider: file
  config:
    directory: ""`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(yaml), 0644)
	require.NoError(t, err)

	_, err = FromYAMLFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "changelog.config.directory")
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

// Criterion 1: a project config file with a missing name fails validation
// with an error naming the field.
func TestFromYAMLFile_MissingNameReturnsError(t *testing.T) {
	yaml := `command: "go run ."`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(yaml), 0644)
	require.NoError(t, err)

	_, err = FromYAMLFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "name")
}

// TestParseYAMLFile_MissingNameParsesWithoutValidation asserts that the
// unvalidated parse path reads a config with no name — the shape init must
// accept so it can backfill one — while still prefilling defaults, where the
// validating loader rejects the same file.
func TestParseYAMLFile_MissingNameParsesWithoutValidation(t *testing.T) {
	yaml := `command: "go run ."`
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0644))

	cfg, err := ParseYAMLFile(path)
	require.NoError(t, err)
	require.Empty(t, cfg.Name)
	require.Equal(t, "go run .", cfg.Command)
	require.Equal(t, DefaultSpecDir, cfg.Spec.Config.Directory, "defaults are still prefilled")
}

// Criterion 1: a non-slug-safe name fails validation with an error naming
// the field.
func TestFromYAMLFile_NonSlugNameReturnsError(t *testing.T) {
	yaml := `name: "Has Spaces/UPPER"`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(yaml), 0644)
	require.NoError(t, err)

	_, err = FromYAMLFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "name")
}

// Criterion 1: the project's name and optional source round-trip through
// config.yaml unchanged.
func TestToYAMLFile_NameAndSourceRoundTrip(t *testing.T) {
	cfg := NewDefault()
	cfg.Name = "my-project"
	cfg.Source = "github.com/example/my-project"
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	require.NoError(t, cfg.ToYAMLFile(path))

	loaded, err := FromYAMLFile(path)
	require.NoError(t, err)
	require.Equal(t, "my-project", loaded.Name)
	require.Equal(t, "github.com/example/my-project", loaded.Source)
}

// Criterion 1: SlugifyName converts arbitrary directory basenames into
// slug-safe project names. The expected values are hand-maintained.
func TestSlugifyName_HandMaintainedCases(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"My Project", "my-project"},
		{"spektacular", "spektacular"},
		{"Has Spaces/UPPER", "has-spaces-upper"},
		{"snake_case_ok", "snake_case_ok"},
		{"___", "project"},
		{"", "project"},
	}
	for _, tc := range cases {
		require.Equal(t, tc.want, SlugifyName(tc.in), "SlugifyName(%q)", tc.in)
	}
}

// Criterion 1: every slugified name passes the same validation that
// config.yaml loading applies to the name field.
func TestSlugifyName_ProducesValidNames(t *testing.T) {
	for _, in := range []string{"My Project", "spektacular", "___", "9lives", "-lead-trim-"} {
		cfg := NewDefault()
		cfg.Name = SlugifyName(in)
		require.NoError(t, cfg.Validate(), "slugified %q should validate", in)
	}
}

// Criterion 1: a registry entry whose name duplicates an earlier entry fails
// validation with an error naming the duplicated entry.
func TestValidateRepos_DuplicateNameReturnsError(t *testing.T) {
	cfg := NewDefault()
	cfg.Name = "testproj"
	cfg.Repos = []RepoEntry{
		{Name: "api", Local: "."},
		{Name: "api", Address: "github.com/example/api"},
	}

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), `repos: name "api" is configured more than once`)
}

// Criterion 1: a registry entry with a non-slug-safe name fails validation
// with an error naming the entry's index.
func TestValidateRepos_NonSlugNameReturnsError(t *testing.T) {
	cfg := NewDefault()
	cfg.Name = "testproj"
	cfg.Repos = []RepoEntry{
		{Name: "api", Local: "."},
		{Name: "Has Spaces/UPPER", Local: "./other"},
	}

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "repos[1].name")
}

// Criterion 1: a registry entry with neither address nor local fails
// validation with an error naming the entry.
func TestValidateRepos_NeitherAddressNorLocalReturnsError(t *testing.T) {
	cfg := NewDefault()
	cfg.Name = "testproj"
	cfg.Repos = []RepoEntry{
		{Name: "api"},
	}

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), `repo "api": at least one of address or local is required`)
}

// Criterion 2: an unknown repo provider is rejected at load time with an
// error naming the provider.
func TestFromYAMLFile_UnknownRepoProviderReturnsError(t *testing.T) {
	yaml := `name: testproj
repos:
  - name: api
    local: "."
    provider: svn`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(yaml), 0644)
	require.NoError(t, err)

	_, err = FromYAMLFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "provider")
}

// Criterion 2: an entry with no provider is accepted at load time — the
// provider is optional and defaults to git.
func TestFromYAMLFile_EmptyRepoProviderIsAccepted(t *testing.T) {
	yaml := `name: testproj
repos:
  - name: api
    local: "."`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(yaml), 0644)
	require.NoError(t, err)

	cfg, err := FromYAMLFile(path)
	require.NoError(t, err)
	require.Len(t, cfg.Repos, 1)
	require.Empty(t, cfg.Repos[0].Provider)
}

// Criterion 2: WithDefaults resolves an empty provider to git and leaves an
// explicit provider untouched.
func TestRepoEntry_WithDefaultsResolvesProviderToGit(t *testing.T) {
	entry := RepoEntry{Name: "api", Local: "."}
	require.Equal(t, ProviderGit, entry.WithDefaults().Provider)

	explicit := RepoEntry{Name: "api", Local: ".", Provider: ProviderGit}
	require.Equal(t, ProviderGit, explicit.WithDefaults().Provider)
}

// A two-entry registry with every membership field populated round-trips
// through config.yaml unchanged. RepoEntry carries membership only —
// descriptive metadata (description/role/tags/deployment) lives in the
// repo's own repo.yaml and is covered by internal/config/repo_test.go
// instead.
func TestToYAMLFile_ReposRoundTrip(t *testing.T) {
	cfg := NewDefault()
	cfg.Name = "testproj"
	cfg.Repos = []RepoEntry{
		{
			Name:         "api",
			Address:      "github.com/example/api",
			Local:        "./api",
			Dependencies: []string{"db"},
			Provider:     ProviderGit,
		},
		{
			Name:  "db",
			Local: "./db",
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, cfg.ToYAMLFile(path))

	loaded, err := FromYAMLFile(path)
	require.NoError(t, err)
	require.Equal(t, cfg.Repos, loaded.Repos)
}

// Criterion 3: a project config may declare project-owned knowledge sources,
// and a populated project-level knowledge list round-trips through
// config.yaml.
func TestToYAMLFile_ProjectOwnedKnowledgeSourcesRoundTrip(t *testing.T) {
	cfg := NewDefault()
	cfg.Name = "testproj"
	cfg.Knowledge = KnowledgeConfig{
		Sources: []SourceConfig{
			{
				Scope:    "team",
				Provider: ProviderFile,
				Config:   FileKnowledgeConfig{Location: "/shared/team/knowledge"},
			},
			{
				Scope:    "global",
				Provider: ProviderFile,
				Config:   FileKnowledgeConfig{Location: "/shared/global/knowledge"},
			},
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, cfg.ToYAMLFile(path))

	loaded, err := FromYAMLFile(path)
	require.NoError(t, err)
	require.Equal(t, cfg.Knowledge, loaded.Knowledge)
}
