package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

const (
	SpecIDMethodTimestamp = "timestamp"
	SpecIDMethodCounter   = "counter"
	SpecIDMethodExternal  = "external"
)

// ProviderFile is the only storage provider this release ships. The provider
// field on the spec, plan, and knowledge sections names a backend; today it
// must always be this value.
const ProviderFile = "file"

const (
	// DefaultSpecDir is the spec output directory used when none is configured.
	// It is resolved relative to the project root, like the knowledge location.
	DefaultSpecDir = ".spektacular/specs"
	// DefaultPlanDir is the plan output directory used when none is configured.
	// It is resolved relative to the project root, like the knowledge location.
	DefaultPlanDir = ".spektacular/plans"
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

// KnowledgeConfig holds the ordered list of configured knowledge sources.
type KnowledgeConfig struct {
	Sources []SourceConfig `yaml:"sources"`
}

// SourceConfig is a single knowledge source. Each source names its own
// provider and scope, so scopes can use different backends independently.
type SourceConfig struct {
	Scope    string              `yaml:"scope"`
	Provider string              `yaml:"provider"`
	Config   FileKnowledgeConfig `yaml:"config"`
}

// FileKnowledgeConfig is the file-provider configuration for a knowledge source.
type FileKnowledgeConfig struct {
	Location string `yaml:"location"`
}

// Config is the top-level Spektacular configuration.
type Config struct {
	Command   string          `yaml:"command"`
	Agent     string          `yaml:"agent"`
	Debug     DebugConfig     `yaml:"debug"`
	Spec      SpecConfig      `yaml:"spec"`
	Plan      PlanConfig      `yaml:"plan"`
	Knowledge KnowledgeConfig `yaml:"knowledge"`
}

// NewDefault returns a Config populated with default values.
func NewDefault() Config {
	return Config{
		Command: "spektacular",
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
		Knowledge: KnowledgeConfig{
			// The project knowledge source is configured by default so a
			// freshly written config.yaml shows it explicitly. Team and
			// global sources are opt-in additions the user configures by hand.
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
	}
}

// FromYAMLFile loads a Config from a YAML file, expanding ${VAR} patterns.
func FromYAMLFile(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("reading config file %s: %w", path, err)
	}

	expanded := expandEnvVars(string(raw))

	cfg := NewDefault()
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config file %s: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validating config file %s: %w", path, err)
	}
	return cfg, nil
}

// Validate checks whether the config contains supported values.
func (c Config) Validate() error {
	if err := c.Spec.Validate(); err != nil {
		return err
	}
	if err := c.Plan.Validate(); err != nil {
		return err
	}
	if err := c.Knowledge.Validate(); err != nil {
		return err
	}
	return nil
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
