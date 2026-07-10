package cmd

import "github.com/jumppad-labs/spektacular/internal/config"

// The `changelog file` subcommand group reads and writes changelog records
// within the configured changelog directory. See newStoreFileCmd for the
// shared implementation.
func init() {
	changelogCmd.AddCommand(newStoreFileCmd(
		"Read and write files in the changelog store",
		func(c config.Config) string { return c.Changelog.Config.Directory },
		true,
	))
}
