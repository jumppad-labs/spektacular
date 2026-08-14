package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// RepoConfigFileName is the filename of the repo-level configuration file,
// stored in the repo's .spektacular directory beside config.yaml when the
// repo is colocated with a project.
const RepoConfigFileName = "repo.yaml"

// RepoConfig is the repo-level configuration. It holds only repo-scoped
// concerns — the repo's own descriptive metadata, knowledge sources, and
// changelog provider — and carries no pointer back to any project:
// membership is recorded only in project configs, so a repo can belong to
// multiple projects.
type RepoConfig struct {
	Description string          `yaml:"description,omitempty"`
	Role        string          `yaml:"role,omitempty"`
	Tags        []string        `yaml:"tags,omitempty"`
	Deployment  string          `yaml:"deployment,omitempty"`
	Knowledge   KnowledgeConfig `yaml:"knowledge"`
	Changelog   ChangelogConfig `yaml:"changelog"`
}

// NewDefaultRepoConfig returns a RepoConfig populated with default values:
// the repo's own knowledge store and a file-backed changelog.
func NewDefaultRepoConfig() RepoConfig {
	return RepoConfig{
		Knowledge: KnowledgeConfig{
			Sources: []SourceConfig{
				{
					Scope:    DefaultKnowledgeScope,
					Provider: ProviderFile,
					Config: FileKnowledgeConfig{
						Location: DefaultKnowledgeLocation,
					},
				},
			},
		},
		Changelog: ChangelogConfig{
			Provider: ProviderFile,
			Config: FileChangelogConfig{
				Directory: DefaultChangelogDir,
			},
		},
	}
}

// RepoConfigFromYAMLFile loads a RepoConfig from a YAML file, expanding
// ${VAR} patterns.
func RepoConfigFromYAMLFile(path string) (RepoConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return RepoConfig{}, fmt.Errorf("reading repo config file %s: %w", path, err)
	}

	expanded := expandEnvVars(string(raw))

	cfg := NewDefaultRepoConfig()
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return RepoConfig{}, fmt.Errorf("parsing repo config file %s: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return RepoConfig{}, fmt.Errorf("validating repo config file %s: %w", path, err)
	}
	return cfg, nil
}

// Validate checks whether the repo config contains supported values.
func (c RepoConfig) Validate() error {
	if err := c.Knowledge.Validate(); err != nil {
		return err
	}
	return c.Changelog.Validate()
}

// WithDefaults returns a RepoConfig whose knowledge section is guaranteed to
// carry at least one source: if none are configured it synthesises the repo's
// own store under repoRoot, mirroring KnowledgeConfig.WithDefaults.
func (c RepoConfig) WithDefaults(repoRoot string) RepoConfig {
	c.Knowledge = c.Knowledge.WithDefaults(repoRoot)
	return c
}

// ToYAMLFile writes the RepoConfig to a YAML file.
func (c RepoConfig) ToYAMLFile(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshalling repo config: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing repo config file %s: %w", path, err)
	}
	return nil
}
