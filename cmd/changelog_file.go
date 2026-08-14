package cmd

import (
	"path/filepath"

	"github.com/jumppad-labs/spektacular/internal/config"
)

// The `changelog file` subcommand group reads and writes changelog records
// within the configured changelog directory, namespaced by the project's
// name so multiple projects writing to one store can never collide.
// Agent-facing paths stay `<name>.md`; the project folder is injected here,
// below the CLI surface. See newStoreFileCmd for the shared implementation.
func init() {
	changelogCmd.AddCommand(newStoreFileCmd(
		"Read and write files in the changelog store",
		func(c config.Config) string { return filepath.Join(c.Changelog.Config.Directory, c.Name) },
		true,
		true,
	))
}
