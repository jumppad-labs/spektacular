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

// Config is the top-level Spektacular configuration.
type Config struct {
	Command string      `yaml:"command"`
	Agent   string      `yaml:"agent"`
	Debug   DebugConfig `yaml:"debug"`
	Spec    SpecConfig  `yaml:"spec"`
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
