package config

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// APIConfig holds API-related configuration.
type APIConfig struct {
	AnthropicAPIKey string `yaml:"anthropic_api_key"`
	Timeout         int    `yaml:"timeout"`
}

// ModelTiers defines model names for each complexity tier.
type ModelTiers struct {
	Simple  string `yaml:"simple"`
	Medium  string `yaml:"medium"`
	Complex string `yaml:"complex"`
}

// ModelsConfig holds model selection configuration.
type ModelsConfig struct {
	Default string     `yaml:"default"`
	Tiers   ModelTiers `yaml:"tiers"`
}

// ComplexityThresholds defines score boundaries for model tier selection.
type ComplexityThresholds struct {
	Simple  float64 `yaml:"simple"`
	Medium  float64 `yaml:"medium"`
	Complex float64 `yaml:"complex"`
}

// ComplexityConfig holds complexity analysis configuration.
type ComplexityConfig struct {
	Thresholds ComplexityThresholds `yaml:"thresholds"`
}

// OutputConfig holds output format configuration.
type OutputConfig struct {
	Format          string `yaml:"format"`
	IncludeMetadata bool   `yaml:"include_metadata"`
}

// DebugConfig holds debug logging configuration.
type DebugConfig struct {
	Enabled bool   `yaml:"enabled"`
	LogDir  string `yaml:"log_dir"`
}

// AgentConfig holds configuration for the coding agent subprocess.
type AgentConfig struct {
	Command                  string   `yaml:"command"`
	Args                     []string `yaml:"args"`
	AllowedTools             []string `yaml:"allowed_tools"`
	DangerouslySkipPermissions bool  `yaml:"dangerously_skip_permissions"`
}

// Config is the top-level Spektacular configuration.
type Config struct {
	API        APIConfig        `yaml:"api"`
	Models     ModelsConfig     `yaml:"models"`
	Complexity ComplexityConfig `yaml:"complexity"`
	Output     OutputConfig     `yaml:"output"`
	Agent      AgentConfig      `yaml:"agent"`
	Debug      DebugConfig      `yaml:"debug"`
}

// NewDefault returns a Config populated with default values.
func NewDefault() Config {
	return Config{
		API: APIConfig{
			AnthropicAPIKey: "${ANTHROPIC_API_KEY}",
			Timeout:         60,
		},
		Models: ModelsConfig{
			Default: "anthropic/claude-3-5-sonnet-20241022",
			Tiers: ModelTiers{
				Simple:  "anthropic/claude-3-5-haiku-20241022",
				Medium:  "anthropic/claude-3-5-sonnet-20241022",
				Complex: "anthropic/claude-3-opus-20240229",
			},
		},
		Complexity: ComplexityConfig{
			Thresholds: ComplexityThresholds{
				Simple:  0.3,
				Medium:  0.6,
				Complex: 0.8,
			},
		},
		Output: OutputConfig{
			Format:          "markdown",
			IncludeMetadata: true,
		},
		Agent: AgentConfig{
			Command:                  "claude",
			Args:                     []string{"--output-format", "stream-json", "--verbose"},
			AllowedTools:             []string{"Bash", "Read", "Write", "Edit", "Glob", "Grep", "WebFetch", "WebSearch"},
			DangerouslySkipPermissions: false,
		},
		Debug: DebugConfig{
			Enabled: false,
			LogDir:  ".spektacular/logs",
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
	return cfg, nil
}

// ToYAMLFile writes the Config to a YAML file, creating parent directories as needed.
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

// GetModelForComplexity returns the appropriate model name for a given complexity score.
func (c Config) GetModelForComplexity(score float64) string {
	t := c.Complexity.Thresholds
	switch {
	case score < t.Simple:
		return c.Models.Tiers.Simple
	case score < t.Medium:
		return c.Models.Tiers.Medium
	default:
		return c.Models.Tiers.Complex
	}
}

// expandEnvVars replaces ${VAR} patterns in s with the current environment values.
func expandEnvVars(s string) string {
	return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		name := match[2 : len(match)-1] // strip ${ and }
		return os.Getenv(name)
	})
}
