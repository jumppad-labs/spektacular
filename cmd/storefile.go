package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/identifier"
	"github.com/jumppad-labs/spektacular/internal/metadata"
	"github.com/jumppad-labs/spektacular/internal/output"
	"github.com/jumppad-labs/spektacular/internal/store"
	"github.com/spf13/cobra"
)

// storeDirFunc selects the configured, project-root-relative directory that a
// `file` subcommand group operates within.
type storeDirFunc func(config.Config) string

// stripLeadingFrontmatterBlocks removes zero or more leading YAML frontmatter
// blocks from raw. Each `<kind> file write` is idempotent under repeated
// invocation: the caller may pass source content that already carries a
// frontmatter block (e.g. because they cp'd a stored artifact), and the write
// handler must treat that block as prior metadata to be discarded so
// metadata.Merge can prepend a single, freshly-authored block. Malformed or
// absent frontmatter is not an error — Split returns the input unchanged and
// the loop terminates.
func stripLeadingFrontmatterBlocks(raw []byte) []byte {
	body := raw
	for {
		fm, next, err := metadata.Split(body)
		if err != nil || fm == nil {
			return body
		}
		body = next
	}
}

// metadataOptsForStatus parses the shared `--status` flag value into an
// UpdateOptions payload. An empty value means "no status change" (Merge picks
// the default for a fresh write and preserves for an existing artifact); any
// other value must match the four-value enum, and is rejected with an
// actionable error otherwise.
func metadataOptsForStatus(raw string) (metadata.UpdateOptions, error) {
	if raw == "" {
		return metadata.UpdateOptions{}, nil
	}
	s := metadata.Status(raw)
	switch s {
	case metadata.StatusInProgress, metadata.StatusCompleted, metadata.StatusSuperseded, metadata.StatusArchived:
		return metadata.UpdateOptions{Status: &s}, nil
	}
	return metadata.UpdateOptions{}, output.NewError("invalid_status",
		fmt.Sprintf("--status %q is not one of the four allowed values", raw)).
		WithNextAction("Pass one of in-progress, completed, superseded, archived.")
}

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

	var (
		fromPath   string
		statusFlag string
	)
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
			storePath := filepath.Join(storeDir, args[0])
			existing, err := st.Read(storePath)
			if err != nil && !errors.Is(err, store.ErrNotFound) {
				return err
			}
			opts, err := metadataOptsForStatus(statusFlag)
			if err != nil {
				return err
			}
			body := stripLeadingFrontmatterBlocks(content)
			merged, err := metadata.Merge(existing, body, opts)
			if err != nil {
				return output.NewError("metadata_merge_failed", err.Error()).WithResource(args[0])
			}
			return st.Write(storePath, merged)
		},
	}
	write.Flags().StringVar(&fromPath, "from", "", "Path to the source file whose contents will be written into the store")
	_ = write.MarkFlagRequired("from")
	write.Flags().StringVar(&statusFlag, "status", "", "Optional lifecycle status to apply: one of in-progress, completed, superseded, archived")

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

	var (
		listStatus        string
		listCreatedAfter  string
		listCreatedBefore string
		listClosedAfter   string
		listClosedBefore  string
	)
	list := &cobra.Command{
		Use:   "list [path]",
		Short: "List files in the store, optionally filtered by metadata",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, err := parseListFilter(listStatus, listCreatedAfter, listCreatedBefore, listClosedAfter, listClosedBefore)
			if err != nil {
				return err
			}
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
			files := make([]map[string]any, 0, len(entries))
			for _, e := range entries {
				entryPath := filepath.Join(path, e.Name)
				storeRel := filepath.ToSlash(strings.TrimPrefix(entryPath, st.Root()+string(filepath.Separator)))
				item := map[string]any{
					"name": e.Name,
					"path": storeRel,
				}
				var fm *metadata.Metadata
				if !e.IsDir {
					raw, readErr := st.Read(entryPath)
					if readErr == nil {
						if parsed, _, splitErr := metadata.Split(raw); splitErr == nil && parsed != nil {
							fm = parsed
							item["created_date"] = parsed.CreatedDate.Format("2006-01-02")
							item["status"] = string(parsed.Status)
							if !parsed.ClosedDate.IsZero() {
								item["closed_date"] = parsed.ClosedDate.Format("2006-01-02")
							}
						}
					}
				}
				if filter.active() {
					if e.IsDir || fm == nil {
						continue
					}
					if !filter.matches(*fm) {
						continue
					}
				}
				files = append(files, item)
			}
			return output.Write(cmd.OutOrStdout(), map[string]any{"files": files}, "")
		},
	}
	list.Flags().StringVar(&listStatus, "status", "", "Filter to artifacts whose lifecycle status matches; one of in-progress, completed, superseded, archived")
	list.Flags().StringVar(&listCreatedAfter, "created-after", "", "Filter to artifacts whose created_date is on or after this YYYY-MM-DD date")
	list.Flags().StringVar(&listCreatedBefore, "created-before", "", "Filter to artifacts whose created_date is on or before this YYYY-MM-DD date")
	list.Flags().StringVar(&listClosedAfter, "closed-after", "", "Filter to artifacts whose closed_date is on or after this YYYY-MM-DD date")
	list.Flags().StringVar(&listClosedBefore, "closed-before", "", "Filter to artifacts whose closed_date is on or before this YYYY-MM-DD date")

	var setStatusFlag string
	setStatus := &cobra.Command{
		Use:   "set-status <path>",
		Short: "Update the lifecycle status of a stored artifact without rewriting its body",
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
			opts, err := metadataOptsForStatus(setStatusFlag)
			if err != nil {
				return err
			}
			if opts.Status == nil {
				return output.NewError("missing_status",
					"--status is required for set-status").
					WithNextAction("Pass --status with one of in-progress, completed, superseded, archived.")
			}
			st, storeDir, err := storeFileStore(dir)
			if err != nil {
				return err
			}
			storePath := filepath.Join(storeDir, args[0])
			existing, err := st.Read(storePath)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					return output.NewError("not_found", fmt.Sprintf("file %q not found", args[0])).
						WithResource(args[0])
				}
				return err
			}
			body := existing
			if _, split, splitErr := metadata.Split(existing); splitErr == nil {
				body = split
			}
			merged, err := metadata.Merge(existing, body, opts)
			if err != nil {
				return output.NewError("metadata_merge_failed", err.Error()).WithResource(args[0])
			}
			if err := st.Write(storePath, merged); err != nil {
				return err
			}
			fm, _, splitErr := metadata.Split(merged)
			payload := map[string]any{
				"path":   args[0],
				"status": string(*opts.Status),
			}
			if splitErr == nil && fm != nil && !fm.ClosedDate.IsZero() {
				payload["closed_date"] = fm.ClosedDate.Format("2006-01-02")
			}
			return output.Write(cmd.OutOrStdout(), payload, "")
		},
	}
	setStatus.Flags().StringVar(&setStatusFlag, "status", "", "Lifecycle status to apply: one of in-progress, completed, superseded, archived")
	_ = setStatus.MarkFlagRequired("status")

	fileCmd.AddCommand(write, read, del, list, setStatus)
	return fileCmd
}
