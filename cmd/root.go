package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/output"
	"github.com/spf13/cobra"
)

var version = "0.1.0"

// globalFields holds the raw --fields JSON array string, available to all subcommands.
var globalFields string

var rootCmd = &cobra.Command{
	Use:     "spektacular",
	Short:   "Agent-driven tool for spec-driven development",
	Version: version,
}

func Execute() {
	os.Exit(runRoot())
}

// runRoot runs the root command and returns the process exit code. It is the
// single place a command's outcome (result or error) is translated into the
// response envelope and written to the output stream — extracted from
// Execute so tests can exercise the full wrapping behavior without an
// os.Exit call.
func runRoot() int {
	err := rootCmd.Execute()
	if err == nil {
		return 0
	}
	output.WriteFailure(rootCmd.OutOrStdout(), toErrorResponse(err), globalFields)
	return 1
}

// toErrorResponse converts any error returned by a command into the shared
// ErrorResponse shape: an already-built *output.ErrorResponse passes through
// unchanged, anything else falls back to a generic internal_error.
func toErrorResponse(err error) *output.ErrorResponse {
	if er, ok := err.(*output.ErrorResponse); ok {
		return er
	}
	return output.NewError("internal_error", err.Error())
}

func configFilePath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	return filepath.Join(cwd, ".spektacular", "config.yaml"), nil
}

// loadConfig loads the project config from the current working directory.
// Returns defaults if the config file does not exist.
// Returns an error if the config file exists but is invalid.
func loadConfig() (config.Config, error) {
	cfgPath, err := configFilePath()
	if err != nil {
		return config.Config{}, err
	}
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return config.NewDefault(), nil
	}
	return config.FromYAMLFile(cfgPath)
}

// dataDir returns the .spektacular directory for the current working directory.
// Both spec and plan workflows share this directory (and a single state.json).
func dataDir() (string, error) {
	root, err := projectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".spektacular"), nil
}

// projectRoot returns the project root — the current working directory. Spec,
// plan, and knowledge directories from the config are all resolved relative to
// this, so the configured paths (e.g. ".spektacular/specs") are project-root
// relative rather than relative to the .spektacular data directory.
func projectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	return cwd, nil
}

func init() {
	// The response envelope wrapper in runRoot is the only place a command's
	// outcome is ever printed, so the CLI framework's own default error/usage
	// printing is turned off to avoid printing a failure a second time.
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
	rootCmd.PersistentFlags().StringVar(&globalFields, "fields", "", `JSON array of output fields to include (e.g. '["step","instruction"]')`)
	rootCmd.AddCommand(specCmd)
	rootCmd.AddCommand(planCmd)
	rootCmd.AddCommand(implementCmd)
	rootCmd.AddCommand(knowledgeCmd)
	rootCmd.AddCommand(skillCmd)
	rootCmd.AddCommand(initCmd)
}
