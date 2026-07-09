package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/identifier"
	"github.com/jumppad-labs/spektacular/internal/output"
	"github.com/jumppad-labs/spektacular/internal/store"
	"github.com/spf13/cobra"
)

// storeDirFunc selects the configured, project-root-relative directory that a
// `file` subcommand group operates within.
type storeDirFunc func(config.Config) string

// validateIDPrefix checks that the leading path segment of a store-relative
// write path carries an ID matching the configured spec.id_method scheme
// (e.g. "000034_feature/plan.md" for counter, "20260709062525-feature.md"
// for timestamp). Plan and changelog entries must reuse the ID minted for
// their originating spec rather than mint their own, so a write whose name
// lacks a matching ID is rejected with guidance to reuse the spec's ID.
func validateIDPrefix(cfg config.Config, writePath string) error {
	name := writePath
	if idx := strings.IndexAny(name, "/\\"); idx >= 0 {
		name = name[:idx]
	}
	name = strings.TrimSuffix(name, filepath.Ext(name))

	if identifier.HasPrefix(cfg.Spec.IDMethod, name) {
		return nil
	}
	return output.NewError("missing_id_prefix",
		fmt.Sprintf("%q has no ID prefix matching the configured spec.id_method (%q) — reuse the ID from the originating spec's filename rather than inventing a new one", writePath, cfg.Spec.IDMethod)).
		WithResource(writePath).
		WithNextAction("Find the originating spec's ID (e.g. via `spec file list`) and prefix this name with it, matching spec.id_method's format.")
}

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
// name rather than a full project path. It backs `spec file`, `plan file`,
// and `changelog file`, which differ only in which configured directory they
// target and whether writes require an ID prefix.
//
// requireID gates ID-prefix validation on write. Spec is where an ID is
// born (resolved by ResolveIdentifier before spec new ever calls this write
// path), so `spec file` passes false. Plan and changelog names must reuse
// that same spec ID rather than mint their own, so `plan file` and
// `changelog file` pass true: a write whose leading path segment lacks an ID
// matching the configured spec.id_method scheme is rejected.
func newStoreFileCmd(short string, dir storeDirFunc, requireID bool) *cobra.Command {
	fileCmd := &cobra.Command{Use: "file", Short: short}

	var fromPath string
	write := &cobra.Command{
		Use:   "write <path>",
		Short: "Write the contents of a source file into the store",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if requireID {
				if err := validateIDPrefix(cfg, args[0]); err != nil {
					return err
				}
			}
			st, storeDir, err := storeFileStore(dir)
			if err != nil {
				return err
			}
			content, err := os.ReadFile(fromPath)
			if err != nil {
				return fmt.Errorf("reading source file %q: %w", fromPath, err)
			}
			return st.Write(filepath.Join(storeDir, args[0]), content)
		},
	}
	write.Flags().StringVar(&fromPath, "from", "", "Path to the source file whose contents will be written into the store")
	_ = write.MarkFlagRequired("from")

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
				if errors.Is(err, store.ErrNotFound) {
					return output.NewError("not_found", fmt.Sprintf("file %q not found", args[0])).
						WithResource(args[0])
				}
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
