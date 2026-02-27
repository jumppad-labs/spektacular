package cmd

import (
	"fmt"
	"os"

	"github.com/nicholasjackson/spektacular/internal/spec"
	"github.com/spf13/cobra"
)

var newTitle string
var newDescription string

var newCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Create a new specification from template",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}
		specPath, err := spec.Create(cwd, name, newTitle, newDescription)
		if err != nil {
			return err
		}
		fmt.Printf("Created spec: %s\n", specPath)
		return nil
	},
}

func init() {
	newCmd.Flags().StringVar(&newTitle, "title", "", "Feature title")
	newCmd.Flags().StringVar(&newDescription, "description", "", "Feature description")
}
