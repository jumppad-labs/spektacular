package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/spf13/cobra"
)

const version = "0.1.0"

// globalFields holds the raw --fields JSON array string, available to all subcommands.
var globalFields string

var rootCmd = &cobra.Command{
	Use:     "spektacular",
	Short:   "Agent-driven tool for spec-driven development",
	Version: version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// loadConfig loads the project config from the current working directory.
// Returns defaults if the config file does not exist.
// Returns an error if the config file exists but is invalid.
func loadConfig() (config.Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return config.Config{}, fmt.Errorf("getting working directory: %w", err)
	}
	cfgPath := filepath.Join(cwd, ".spektacular", "config.yaml")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return config.NewDefault(), nil
	}
	return config.FromYAMLFile(cfgPath)
}

// specDataDir returns the .spektacular directory for the current working directory.
func specDataDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	return filepath.Join(cwd, ".spektacular"), nil
}

func init() {
	rootCmd.PersistentFlags().StringVar(&globalFields, "fields", "", `JSON array of output fields to include (e.g. '["step","instruction"]')`)
	rootCmd.AddCommand(specCmd)
	rootCmd.AddCommand(planCmd)
	rootCmd.AddCommand(skillCmd)
	rootCmd.AddCommand(initCmd)
}
