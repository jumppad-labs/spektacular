package cmd

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/jumppad-labs/spektacular/internal/output"
	"github.com/jumppad-labs/spektacular/internal/store"
	"github.com/spf13/cobra"
)

var specFileCmd = &cobra.Command{
	Use:   "file",
	Short: "Read and write files in the spec store",
}

var specFileWriteCmd = &cobra.Command{
	Use:   "write <path>",
	Short: "Write stdin to a file in the spec store",
	Args:  cobra.ExactArgs(1),
	RunE:  runSpecFileWrite,
}

var specFileReadCmd = &cobra.Command{
	Use:   "read <path>",
	Short: "Read a file from the spec store and write it to stdout",
	Args:  cobra.ExactArgs(1),
	RunE:  runSpecFileRead,
}

var specFileDeleteCmd = &cobra.Command{
	Use:   "delete <path>",
	Short: "Delete a file from the spec store",
	Args:  cobra.ExactArgs(1),
	RunE:  runSpecFileDelete,
}

var specFileListCmd = &cobra.Command{
	Use:   "list [path]",
	Short: "List files in the spec store",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runSpecFileList,
}

// specFileStore returns a store rooted at the project root together with the
// configured spec directory. Path arguments to the `spec file` subcommands are
// resolved relative to that configured directory, so callers pass a spec file
// name (e.g. "000020_context.md") rather than a full project path.
func specFileStore() (store.Store, string, error) {
	root, err := projectRoot()
	if err != nil {
		return nil, "", err
	}
	cfg, err := loadConfig()
	if err != nil {
		return nil, "", err
	}
	return store.NewFileStore(root, "project"), cfg.Spec.Config.Directory, nil
}

func runSpecFileWrite(cmd *cobra.Command, args []string) error {
	st, specDir, err := specFileStore()
	if err != nil {
		return err
	}
	content, err := io.ReadAll(cmd.InOrStdin())
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}
	return st.Write(filepath.Join(specDir, args[0]), content)
}

func runSpecFileRead(cmd *cobra.Command, args []string) error {
	st, specDir, err := specFileStore()
	if err != nil {
		return err
	}
	content, err := st.Read(filepath.Join(specDir, args[0]))
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(content)
	return err
}

func runSpecFileDelete(_ *cobra.Command, args []string) error {
	st, specDir, err := specFileStore()
	if err != nil {
		return err
	}
	return st.Delete(filepath.Join(specDir, args[0]))
}

func runSpecFileList(cmd *cobra.Command, args []string) error {
	st, specDir, err := specFileStore()
	if err != nil {
		return err
	}
	path := specDir
	if len(args) > 0 {
		path = filepath.Join(specDir, args[0])
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
}

func init() {
	specFileCmd.AddCommand(specFileWriteCmd, specFileReadCmd, specFileDeleteCmd, specFileListCmd)
	specCmd.AddCommand(specFileCmd)
}
