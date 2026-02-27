package cmd

import (
	"fmt"
	"os"

	"github.com/nicholasjackson/spektacular/internal/project"
	"github.com/spf13/cobra"
)

var initForce bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Spektacular project structure",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}
		if err := project.Init(cwd, initForce); err != nil {
			return err
		}
		fmt.Printf("Initialized Spektacular project in %s\n", cwd)
		return nil
	},
}

func init() {
	initCmd.Flags().BoolVar(&initForce, "force", false, "Overwrite existing .spektacular directory if it exists")
}
