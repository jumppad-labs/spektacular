package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jumppad-labs/spektacular/internal/agent"
	"github.com/jumppad-labs/spektacular/internal/project"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init <agent>",
	Short: "Initialise a Spektacular project for the specified agent (" + strings.Join(agent.Supported(), ", ") + ")",
	Args:  cobra.ExactArgs(1),
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	a, err := agent.Lookup(args[0])
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	if err := project.Init(cwd, true); err != nil {
		return fmt.Errorf("initialising project: %w", err)
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	cfg.Agent = a.Name()
	cfgPath := filepath.Join(cwd, ".spektacular", "config.yaml")
	if err := cfg.ToYAMLFile(cfgPath); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	versionPath := versionFilePath(filepath.Join(cwd, ".spektacular"))
	if err := writeVersionFile(versionPath, version); err != nil {
		return fmt.Errorf("writing version file: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Spektacular initialised for %s.\n", a.Name())
	fmt.Fprintf(cmd.OutOrStdout(), "  Project:  %s\n", filepath.Join(cwd, ".spektacular"))
	fmt.Fprintf(cmd.OutOrStdout(), "  Version:  %s (%s)\n", versionPath, version)

	return a.Install(cwd, cfg, cmd.OutOrStdout())
}
