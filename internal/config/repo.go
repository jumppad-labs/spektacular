package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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
//
// A repo has two locations. The directory holding this file's .spektacular/
// folder is where the repo's own Spektacular files live: repo.yaml, the
// knowledge store, and the changelog. Source, when set, names where the
// repo's code lives instead of assuming it is that same directory: a file
// location (a plain path or a file:// path, absolute, relative to the
// directory holding repo.yaml, or using ${VAR}) or a git location that
// Spektacular clones into the project's working folder on first use. When
// Source is unset the code and the Spektacular files are colocated.
type RepoConfig struct {
	Description string          `yaml:"description,omitempty"`
	Role        string          `yaml:"role,omitempty"`
	Tags        []string        `yaml:"tags,omitempty"`
	Deployment  string          `yaml:"deployment,omitempty"`
	Source      string          `yaml:"source,omitempty"`
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

// SourceKind classifies a RepoConfig's Source value.
type SourceKind int

const (
	// SourceNone means Source is unset: the code is colocated with the
	// repo's Spektacular files.
	SourceNone SourceKind = iota
	// SourceFile means Source is a directory on disk.
	SourceFile
	// SourceGit means Source is a git location to clone.
	SourceGit
)

// String returns the kind's name for error messages and logs.
func (k SourceKind) String() string {
	switch k {
	case SourceFile:
		return "file"
	case SourceGit:
		return "git"
	default:
		return "none"
	}
}

const fileSourceScheme = "file://"

// gitSourceSchemes are the URL schemes that mark a Source as a git location.
var gitSourceSchemes = []string{"git://", "ssh://", "http://", "https://", "git+"}

// scpStyleSource matches git's scp-style remote form, user@host:path.
var scpStyleSource = regexp.MustCompile(`^[^/@:]+@[^/:]+:`)

// explicitScheme matches a leading URL scheme such as "s3://".
var explicitScheme = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*://`)

// ParseSource classifies Source and resolves it. For a file source it returns
// the absolute directory, anchoring a relative value at configDir (the
// directory holding repo.yaml) and stripping a file:// prefix; for a git
// source it returns the value unchanged. It returns SourceNone and an empty
// value when Source is unset. Environment variables are already expanded by
// the loader, so none are expanded here. A value with a scheme that is neither
// file:// nor a git transport is an error.
func (c RepoConfig) ParseSource(configDir string) (SourceKind, string, error) {
	v := strings.TrimSpace(c.Source)
	if v == "" {
		return SourceNone, "", nil
	}

	if strings.HasPrefix(v, fileSourceScheme) {
		return SourceFile, resolveFileSource(strings.TrimPrefix(v, fileSourceScheme), configDir), nil
	}

	for _, scheme := range gitSourceSchemes {
		if strings.HasPrefix(v, scheme) {
			return SourceGit, v, nil
		}
	}
	if scpStyleSource.MatchString(v) {
		return SourceGit, v, nil
	}

	if m := explicitScheme.FindString(v); m != "" {
		return SourceNone, "", fmt.Errorf("unsupported source scheme %q in %q: use a file:// path, a plain path, or a git location (git://, ssh://, https://, or user@host:path)", strings.TrimSuffix(m, "://"), v)
	}

	return SourceFile, resolveFileSource(v, configDir), nil
}

// resolveFileSource turns a file source into an absolute, cleaned directory
// path, anchoring a relative path at configDir.
func resolveFileSource(path, configDir string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(configDir, path)
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
