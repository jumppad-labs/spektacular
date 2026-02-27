package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <spec-file>",
	Short: "Run Spektacular on a specification file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		specFile := args[0]
		fmt.Printf("Running spec: %s\n", specFile)
		// TODO: implement spec processing
		fmt.Println("Spec processing not yet implemented")
		return nil
	},
}
