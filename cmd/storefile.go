package cmd

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/output"
	"github.com/jumppad-labs/spektacular/internal/store"
	"github.com/spf13/cobra"
)

// storeDirFunc selects the configured, project-root-relative directory that a
// `file` subcommand group operates within.
type storeDirFunc func(config.Config) string

// storeFileStore builds a store rooted at the project root and returns it
// alongside the configured directory the `file` subcommands resolve paths
// against.
func storeFileStore(dir storeDirFunc) (store.Store, string, error) {
	root, err := projectRoot()
	if err != nil {
		return nil, "", err
	}
	cfg, err := loadConfig()
	if err != nil {
		return nil, "", err
	}
	return store.NewFileStore(root, "project"), dir(cfg), nil
}

// newStoreFileCmd builds a `file` subcommand group (write/read/delete/list)
// that reads and writes files within a configured store directory. Path
// arguments are resolved relative to that directory, so callers pass a file
// name rather than a full project path. It backs both `spec file` and
// `plan file`, which differ only in which configured directory they target.
func newStoreFileCmd(short string, dir storeDirFunc) *cobra.Command {
	fileCmd := &cobra.Command{Use: "file", Short: short}

	write := &cobra.Command{
		Use:   "write <path>",
		Short: "Write stdin to a file in the store",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, storeDir, err := storeFileStore(dir)
			if err != nil {
				return err
			}
			content, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return fmt.Errorf("reading stdin: %w", err)
			}
			return st.Write(filepath.Join(storeDir, args[0]), content)
		},
	}

	read := &cobra.Command{
		Use:   "read <path>",
		Short: "Read a file from the store and write it to stdout",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, storeDir, err := storeFileStore(dir)
			if err != nil {
				return err
			}
			content, err := st.Read(filepath.Join(storeDir, args[0]))
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(content)
			return err
		},
	}

	del := &cobra.Command{
		Use:   "delete <path>",
		Short: "Delete a file from the store",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			st, storeDir, err := storeFileStore(dir)
			if err != nil {
				return err
			}
			return st.Delete(filepath.Join(storeDir, args[0]))
		},
	}

	list := &cobra.Command{
		Use:   "list [path]",
		Short: "List files in the store",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, storeDir, err := storeFileStore(dir)
			if err != nil {
				return err
			}
			path := storeDir
			if len(args) > 0 {
				path = filepath.Join(storeDir, args[0])
			}
			entries, err := st.List(path)
			if err != nil {
				return err
			}
			names := make([]string, len(entries))
			for i, e := range entries {
				names[i] = e.Name
			}
			return output.Write(cmd.OutOrStdout(), map[string]any{"files": names}, "")
		},
	}

	fileCmd.AddCommand(write, read, del, list)
	return fileCmd
}
