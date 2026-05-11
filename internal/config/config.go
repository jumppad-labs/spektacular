package config

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

const (
	SpecIDMethodTimestamp = "timestamp"
	SpecIDMethodCounter   = "counter"
	SpecIDMethodExternal  = "external"

	ArtifactBackendLocal  = "local"
	ArtifactBackendNotion = "notion"

	DefaultNotionCacheDir     = "cache/notion"
	DefaultSpecIDPropertyName = "Spec ID"
	DefaultPlanIDPropertyName = "Plan ID"
)

// DebugConfig holds debug logging configuration.
type DebugConfig struct {
	Enabled bool `yaml:"enabled"`
}

// SpecConfig holds configuration for specification creation.
type SpecConfig struct {
	IDMethod string `yaml:"id_method"`
	Counter  int    `yaml:"counter"`
}

// ArtifactsConfig holds configuration for Spektacular artifact storage.
type ArtifactsConfig struct {
	Backend  string       `yaml:"backend"`
	CacheDir string       `yaml:"cache_dir"`
	Notion   NotionConfig `yaml:"notion"`
}

// NotionConfig holds configuration for linked Notion artifact databases.
type NotionConfig struct {
	BasePageURL     string `yaml:"base_page_url"`
	SpecsDataSource string `yaml:"specs_data_source"`
	PlansDataSource string `yaml:"plans_data_source"`
	SpecIDProperty  string `yaml:"spec_id_property"`
	PlanIDProperty  string `yaml:"plan_id_property"`
}

// Config is the top-level Spektacular configuration.
type Config struct {
	Command   string          `yaml:"command"`
	Agent     string          `yaml:"agent"`
	Debug     DebugConfig     `yaml:"debug"`
	Spec      SpecConfig      `yaml:"spec"`
	Artifacts ArtifactsConfig `yaml:"artifacts"`
}

// NewDefault returns a Config populated with default values.
func NewDefault() Config {
	return Config{
		Command: "spektacular",
		Debug: DebugConfig{
			Enabled: false,
		},
		Spec: SpecConfig{
			IDMethod: SpecIDMethodTimestamp,
			Counter:  0,
		},
		Artifacts: ArtifactsConfig{
			Backend:  ArtifactBackendLocal,
			CacheDir: DefaultNotionCacheDir,
			Notion: NotionConfig{
				SpecIDProperty: DefaultSpecIDPropertyName,
				PlanIDProperty: DefaultPlanIDPropertyName,
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
	if err := c.Artifacts.Validate(c.Spec); err != nil {
		return err
	}
	return nil
}

// Validate checks whether the spec config contains supported values.
func (c SpecConfig) Validate() error {
	switch c.IDMethod {
	case "", SpecIDMethodTimestamp, SpecIDMethodCounter, SpecIDMethodExternal:
		return nil
	default:
		return fmt.Errorf("spec.id_method must be one of %q, %q, or %q", SpecIDMethodTimestamp, SpecIDMethodCounter, SpecIDMethodExternal)
	}
}

// Validate checks whether the artifact config contains supported values.
func (c ArtifactsConfig) Validate(spec SpecConfig) error {
	backend := c.Backend
	if backend == "" {
		backend = ArtifactBackendLocal
	}

	switch backend {
	case ArtifactBackendLocal:
		return nil
	case ArtifactBackendNotion:
		return c.validateNotion(spec)
	default:
		return fmt.Errorf("artifacts.backend must be one of %q or %q", ArtifactBackendLocal, ArtifactBackendNotion)
	}
}

func (c ArtifactsConfig) validateNotion(spec SpecConfig) error {
	if spec.IDMethod != SpecIDMethodExternal {
		return fmt.Errorf("spec.id_method must be %q when artifacts.backend is %q", SpecIDMethodExternal, ArtifactBackendNotion)
	}
	if c.CacheDir == "" {
		return fmt.Errorf("artifacts.cache_dir is required when artifacts.backend is %q", ArtifactBackendNotion)
	}
	if c.Notion.SpecsDataSource == "" {
		return fmt.Errorf("artifacts.notion.specs_data_source is required when artifacts.backend is %q", ArtifactBackendNotion)
	}
	if c.Notion.PlansDataSource == "" {
		return fmt.Errorf("artifacts.notion.plans_data_source is required when artifacts.backend is %q", ArtifactBackendNotion)
	}
	if c.Notion.SpecIDProperty == "" {
		return fmt.Errorf("artifacts.notion.spec_id_property is required when artifacts.backend is %q", ArtifactBackendNotion)
	}
	if c.Notion.PlanIDProperty == "" {
		return fmt.Errorf("artifacts.notion.plan_id_property is required when artifacts.backend is %q", ArtifactBackendNotion)
	}
	return nil
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
