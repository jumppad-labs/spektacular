package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

const (
	SpecIDMethodTimestamp = "timestamp"
	SpecIDMethodCounter   = "counter"
	SpecIDMethodExternal  = "external"
)

const (
	SpecTriggerThresholdStrict   = "strict"
	SpecTriggerThresholdModerate = "moderate"
	SpecTriggerThresholdLenient  = "lenient"
)

// ProviderFile is the only storage provider this release ships. The provider
// field on the spec, plan, and knowledge sections names a backend; today it
// must always be this value.
const ProviderFile = "file"

// ProviderGit is the only repo provider this release ships. The provider
// field on a repos entry names how the repo is resolved to a local
// directory; today it must always be this value (or empty, which defaults
// to it).
const ProviderGit = "git"

const (
	// DefaultSpecDir is the spec output directory used when none is configured.
	// It is resolved relative to the project root, like the knowledge location.
	DefaultSpecDir = ".spektacular/specs"
	// DefaultPlanDir is the plan output directory used when none is configured.
	// It is resolved relative to the project root, like the knowledge location.
	DefaultPlanDir = ".spektacular/plans"
	// DefaultChangelogDir is the changelog output directory used when none is
	// configured. It is resolved relative to the project root, like the
	// knowledge location.
	DefaultChangelogDir = ".spektacular/changelog"
	// DefaultKnowledgeScope is the scope of the synthesised default knowledge source.
	DefaultKnowledgeScope = "project"
	// DefaultKnowledgeLocation is the project-relative location of the
	// synthesised default knowledge source.
	DefaultKnowledgeLocation = ".spektacular/knowledge"
)

// DebugConfig holds debug logging configuration.
type DebugConfig struct {
	Enabled bool `yaml:"enabled"`
}

// SpecConfig holds configuration for specification creation. It names a
// storage provider, the provider-agnostic spec identifier method, and the
// provider's own settings.
type SpecConfig struct {
	Provider string         `yaml:"provider"`
	IDMethod string         `yaml:"id_method"`
	Config   FileSpecConfig `yaml:"config"`
}

// FileSpecConfig is the file-provider configuration for the spec section.
type FileSpecConfig struct {
	Directory string `yaml:"directory"`
}

// PlanConfig holds configuration for plan creation. It names a storage
// provider and carries that provider's settings.
type PlanConfig struct {
	Provider string         `yaml:"provider"`
	Config   FilePlanConfig `yaml:"config"`
}

// FilePlanConfig is the file-provider configuration for the plan section.
type FilePlanConfig struct {
	Directory string `yaml:"directory"`
}

// ChangelogConfig holds configuration for changelog record storage. It names
// a storage provider and carries that provider's settings.
type ChangelogConfig struct {
	Provider string              `yaml:"provider"`
	Config   FileChangelogConfig `yaml:"config"`
}

// FileChangelogConfig is the file-provider configuration for the changelog section.
type FileChangelogConfig struct {
	Directory string `yaml:"directory"`
}

// KnowledgeConfig holds the ordered list of configured knowledge sources.
type KnowledgeConfig struct {
	Sources []SourceConfig `yaml:"sources,omitempty"`
}

// SourceConfig is a single knowledge source. Each source names its own
// provider and scope, so scopes can use different backends independently.
type SourceConfig struct {
	Scope    string              `yaml:"scope"`
	Provider string              `yaml:"provider"`
	Config   FileKnowledgeConfig `yaml:"config"`
	// Repo is the registry name of the repo whose config declared this
	// source, set programmatically during aggregation for attribution; it is
	// never declared in a config file. Empty for project-owned sources.
	Repo string `yaml:"-"`
}

// FileKnowledgeConfig is the file-provider configuration for a knowledge source.
type FileKnowledgeConfig struct {
	Location string `yaml:"location"`
}

// RepoEntry is a single member repo in the project's registry. It carries
// membership only — identity, location, and project-scoped dependencies —
// deliberately provider-agnostic siblings of the provider block, mirroring
// how knowledge sources keep scope outside their provider config. A repo's
// descriptive metadata (description, role, tags, deployment) lives in the
// repo's own configuration, not here, so it is never duplicated across the
// projects that register it. At least one of Address/Local is required;
// when both are set, Local wins and Address serves as provenance metadata.
type RepoEntry struct {
	Name         string        `yaml:"name"`
	Address      string        `yaml:"address,omitempty"`
	Local        string        `yaml:"local,omitempty"`
	Dependencies []string      `yaml:"dependencies,omitempty"`
	Provider     string        `yaml:"provider,omitempty"`
	Config       GitRepoConfig `yaml:"config,omitempty"`
}

// GitRepoConfig is the git-provider configuration for a repos entry. It is
// empty in this release and reserved for provider-specific settings.
type GitRepoConfig struct{}

// Config is the top-level project configuration. It carries the project's
// identity, agent behaviour, and the central spec/plan/changelog storage.
// The knowledge section lists only project-owned sources (team or global
// shares, for example); each repo's own knowledge sources are declared in
// that repo's RepoConfig instead.
type Config struct {
	Name                 string          `yaml:"name"`
	Source               string          `yaml:"source,omitempty"`
	Command              string          `yaml:"command"`
	Agent                string          `yaml:"agent"`
	SpecTriggerThreshold string          `yaml:"spec_trigger_threshold"`
	Debug                DebugConfig     `yaml:"debug"`
	Spec                 SpecConfig      `yaml:"spec"`
	Plan                 PlanConfig      `yaml:"plan"`
	Changelog            ChangelogConfig `yaml:"changelog"`
	Knowledge            KnowledgeConfig `yaml:"knowledge,omitempty"`
	Repos                []RepoEntry     `yaml:"repos,omitempty"`
}

// NewDefault returns a Config populated with default values.
func NewDefault() Config {
	return Config{
		Command:              "spektacular",
		SpecTriggerThreshold: SpecTriggerThresholdModerate,
		Debug: DebugConfig{
			Enabled: false,
		},
		Spec: SpecConfig{
			Provider: ProviderFile,
			IDMethod: SpecIDMethodTimestamp,
			Config: FileSpecConfig{
				Directory: DefaultSpecDir,
			},
		},
		Plan: PlanConfig{
			Provider: ProviderFile,
			Config: FilePlanConfig{
				Directory: DefaultPlanDir,
			},
		},
		Changelog: ChangelogConfig{
			Provider: ProviderFile,
			Config: FileChangelogConfig{
				Directory: DefaultChangelogDir,
			},
		},
		// Knowledge is empty by default: the project level lists only sources
		// owned by the project itself (team or global shares the user adds by
		// hand). Each repo's own store is declared in its RepoConfig.
	}
}

// FromYAMLFile loads a Config from a YAML file, expanding ${VAR} patterns.
func FromYAMLFile(path string) (Config, error) {
	cfg, err := ParseYAMLFile(path)
	if err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validating config file %s: %w", path, err)
	}
	return cfg, nil
}

// ParseYAMLFile loads a Config from a YAML file without validating it,
// expanding ${VAR} patterns and prefilling defaults. It exists for init,
// which must be able to read a config that is missing its required name so
// it can backfill one; every other caller wants FromYAMLFile.
func ParseYAMLFile(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("reading config file %s: %w", path, err)
	}

	expanded := expandEnvVars(string(raw))

	cfg := NewDefault()
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config file %s: %w", path, err)
	}
	return cfg, nil
}

// slugPattern matches slug/filesystem-safe identifiers: lowercase letters,
// digits, hyphens, and underscores, with no path separators.
var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// validateSlug checks that value is a slug/filesystem-safe identifier,
// returning an error that names field when it is not.
func validateSlug(field, value string) error {
	if value == "" {
		return fmt.Errorf("%s must not be empty", field)
	}
	if !slugPattern.MatchString(value) {
		return fmt.Errorf("%s %q must contain only lowercase letters, digits, '-' or '_', and must start with a letter or digit", field, value)
	}
	return nil
}

// SlugifyName converts an arbitrary name (such as a directory basename) into
// a slug-safe identifier: lowercased, with every run of unsupported
// characters collapsed to a single hyphen.
func SlugifyName(name string) string {
	slug := strings.ToLower(name)
	slug = nonSlugRunPattern.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-_")
	if slug == "" {
		return "project"
	}
	return slug
}

var nonSlugRunPattern = regexp.MustCompile(`[^a-z0-9_-]+`)

// Validate checks whether the config contains supported values.
func (c Config) Validate() error {
	if err := validateSlug("name", c.Name); err != nil {
		return err
	}
	switch c.SpecTriggerThreshold {
	case "", SpecTriggerThresholdStrict, SpecTriggerThresholdModerate, SpecTriggerThresholdLenient:
	default:
		return fmt.Errorf("spec_trigger_threshold must be one of %q, %q, or %q", SpecTriggerThresholdStrict, SpecTriggerThresholdModerate, SpecTriggerThresholdLenient)
	}
	if err := c.Spec.Validate(); err != nil {
		return err
	}
	if err := c.Plan.Validate(); err != nil {
		return err
	}
	if err := c.Changelog.Validate(); err != nil {
		return err
	}
	if err := c.Knowledge.Validate(); err != nil {
		return err
	}
	if err := validateRepos(c.Repos); err != nil {
		return err
	}
	return nil
}

// validateRepos checks every registry entry for a slug-safe unique name, a
// usable location, and a supported provider.
func validateRepos(repos []RepoEntry) error {
	seen := make(map[string]bool, len(repos))
	for i, r := range repos {
		if err := validateSlug(fmt.Sprintf("repos[%d].name", i), r.Name); err != nil {
			return err
		}
		if seen[r.Name] {
			return fmt.Errorf("repos: name %q is configured more than once", r.Name)
		}
		seen[r.Name] = true
		if r.Address == "" && r.Local == "" {
			return fmt.Errorf("repo %q: at least one of address or local is required", r.Name)
		}
		switch r.Provider {
		case "", ProviderGit:
		default:
			return fmt.Errorf("repo %q: provider %q is not supported (only %q)", r.Name, r.Provider, ProviderGit)
		}
	}
	return nil
}

// WithDefaults returns the entry with its provider defaulted to git when
// unset, mirroring how absent config sections resolve to defaults at load.
func (r RepoEntry) WithDefaults() RepoEntry {
	if r.Provider == "" {
		r.Provider = ProviderGit
	}
	return r
}

// Validate checks whether the spec config names a supported provider and
// carries valid provider settings.
func (c SpecConfig) Validate() error {
	if c.Provider != ProviderFile {
		return fmt.Errorf("spec.provider %q is not supported (only %q)", c.Provider, ProviderFile)
	}
	if c.Config.Directory == "" {
		return fmt.Errorf("spec.config.directory must not be empty")
	}
	switch c.IDMethod {
	case "", SpecIDMethodTimestamp, SpecIDMethodCounter, SpecIDMethodExternal:
	default:
		return fmt.Errorf("spec.id_method must be one of %q, %q, or %q", SpecIDMethodTimestamp, SpecIDMethodCounter, SpecIDMethodExternal)
	}
	return nil
}

// Validate checks whether the plan config names a supported provider and
// carries valid provider settings.
func (c PlanConfig) Validate() error {
	if c.Provider != ProviderFile {
		return fmt.Errorf("plan.provider %q is not supported (only %q)", c.Provider, ProviderFile)
	}
	if c.Config.Directory == "" {
		return fmt.Errorf("plan.config.directory must not be empty")
	}
	return nil
}

// Validate checks whether the changelog config names a supported provider
// and carries valid provider settings.
func (c ChangelogConfig) Validate() error {
	if c.Provider != ProviderFile {
		return fmt.Errorf("changelog.provider %q is not supported (only %q)", c.Provider, ProviderFile)
	}
	if c.Config.Directory == "" {
		return fmt.Errorf("changelog.config.directory must not be empty")
	}
	return nil
}

// Validate checks every knowledge source for a supported provider, required
// fields, and a unique scope.
func (c KnowledgeConfig) Validate() error {
	seen := make(map[string]bool, len(c.Sources))
	for i, src := range c.Sources {
		if src.Scope == "" {
			return fmt.Errorf("knowledge.sources[%d].scope must not be empty", i)
		}
		if seen[src.Scope] {
			return fmt.Errorf("knowledge.sources: scope %q is configured more than once", src.Scope)
		}
		seen[src.Scope] = true
		if src.Provider != ProviderFile {
			return fmt.Errorf("knowledge source %q: provider %q is not supported (only %q)", src.Scope, src.Provider, ProviderFile)
		}
		if src.Config.Location == "" {
			return fmt.Errorf("knowledge source %q: config.location must not be empty", src.Scope)
		}
	}
	return nil
}

// WithDefaults returns a KnowledgeConfig guaranteed to carry at least one
// source: if none are configured it synthesises the default project source
// pointing at the init-created knowledge directory under projectRoot. A
// configuration that already lists sources is returned unchanged.
func (c KnowledgeConfig) WithDefaults(projectRoot string) KnowledgeConfig {
	if len(c.Sources) > 0 {
		return c
	}
	return KnowledgeConfig{
		Sources: []SourceConfig{
			{
				Scope:    DefaultKnowledgeScope,
				Provider: ProviderFile,
				Config: FileKnowledgeConfig{
					Location: filepath.Join(projectRoot, DefaultKnowledgeLocation),
				},
			},
		},
	}
}

// ToYAMLFile writes the Config to a YAML file.
func (c Config) ToYAMLFile(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing config file %s: %w", path, err)
	}
	return nil
}

// expandEnvVars replaces ${VAR} patterns in s with the current environment values.
func expandEnvVars(s string) string {
	return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		name := match[2 : len(match)-1] // strip ${ and }
		return os.Getenv(name)
	})
}
