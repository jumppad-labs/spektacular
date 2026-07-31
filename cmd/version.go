package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jumppad-labs/spektacular/internal/output"
	"github.com/spf13/cobra"
)

// VersionCheckResult is returned by the version check command. The
// "error": false discriminant is injected by the output writer — never
// declared on the struct.
type VersionCheckResult struct {
	Status           string `json:"status"`                      // "match" | "mismatch" | "missing"
	InstalledVersion string `json:"installed_version,omitempty"` // recorded in .spektacular/version; empty when missing
	CurrentVersion   string `json:"current_version"`             // the running binary's version
	Action           string `json:"action,omitempty"`            // set on mismatch/missing: instruction to relay to the user
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Report and check the installed Spektacular version",
}

var versionCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check whether the installed files match the current binary version",
	RunE:  runVersionCheck,
}

func runVersionCheck(cmd *cobra.Command, _ []string) error {
	if schema, _ := cmd.Flags().GetBool("schema"); schema {
		s := commandSchema{
			Input: &schemaObj{
				Type:       "object",
				Properties: map[string]*schemaProp{},
			},
			Output: &schemaObj{
				Type: "object",
				Properties: map[string]*schemaProp{
					"status":            {Type: "string", Enum: []string{"match", "mismatch", "missing"}},
					"installed_version": {Type: "string"},
					"current_version":   {Type: "string"},
					"action":            {Type: "string"},
				},
			},
		}
		return output.Write(cmd.OutOrStdout(), s, "")
	}

	dir, err := dataDir()
	if err != nil {
		return err
	}
	path := versionFilePath(dir)

	var recorded string
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		recorded = string(data)
	case os.IsNotExist(err):
		recorded = ""
	default:
		return output.NewError("version_file_unreadable", fmt.Sprintf("reading version file: %v", err)).
			WithResource(path).
			WithNextAction("Check file permissions on .spektacular/version.")
	}

	status, installed := classifyVersion(recorded, version)

	result := VersionCheckResult{
		Status:           status,
		InstalledVersion: installed,
		CurrentVersion:   version,
	}
	if status != "match" {
		result.Action = staleAction(status)
	}

	return output.New(cmd.OutOrStdout(), globalFields).WriteResult(result)
}

// classifyVersion compares the recorded version-file content against the
// current binary version. Both sides are plain parameters so the decision
// stays a pure function: recorded content is trimmed of surrounding
// whitespace, an empty result is "missing", equality is "match", anything
// else is "mismatch". The trimmed recorded value is returned for reporting.
func classifyVersion(recorded, current string) (status, installed string) {
	installed = strings.TrimSpace(recorded)
	switch installed {
	case "":
		return "missing", ""
	case current:
		return "match", installed
	default:
		return "mismatch", installed
	}
}

// staleAction composes the instruction an agent relays to the user when the
// installation is stale. The suggested init command includes the configured
// agent when one is recorded in the project config.
func staleAction(status string) string {
	initCmd := "spektacular init"
	if cfg, err := loadConfig(); err == nil {
		initCmd = cfg.Command + " init"
		if cfg.Agent != "" {
			initCmd += " " + cfg.Agent
		}
	}
	reason := "the installed Spektacular files are out of date"
	if status == "missing" {
		reason = "the installed Spektacular files have no recorded version"
	}
	return fmt.Sprintf("%s — ask the user to re-run `%s` to refresh them", reason, initCmd)
}

// versionFilePath returns the path of the version file inside the given
// .spektacular data directory. The file records the bare version of the
// binary that last ran init (no sha suffix, so two builds of the same
// release always match).
func versionFilePath(dataDir string) string {
	return filepath.Join(dataDir, "version")
}

// writeVersionFile records v as the installed version at path, overwriting
// any previous record. Re-running init is the one way the recorded version
// advances, which is what clears a mismatch.
func writeVersionFile(path, v string) error {
	return os.WriteFile(path, []byte(v+"\n"), 0644)
}

func init() {
	versionCheckCmd.Flags().Bool("schema", false, "Print the input/output schema and exit")
	versionCmd.AddCommand(versionCheckCmd)
}
