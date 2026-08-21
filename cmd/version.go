package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/output"
	"github.com/spf13/cobra"
)

// VersionCheckResult is returned by the version check command. The
// "error": false discriminant is injected by the output writer — never
// declared on the struct.
type VersionCheckResult struct {
	Status           string `json:"status"`                      // "match" | "mismatch" | "missing" | "migration_needed"
	InstalledVersion string `json:"installed_version,omitempty"` // recorded in .spektacular/version; empty when missing
	CurrentVersion   string `json:"current_version"`             // the running binary's version
	Action           string `json:"action,omitempty"`            // set on mismatch/missing/migration_needed: instruction to relay to the user
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Report and check the installed Spektacular version",
	RunE:  runUnknownSubcommand,
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
					"status":            {Type: "string", Enum: []string{"match", "mismatch", "missing", "migration_needed"}},
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

	// Check for migration before version comparison
	migrationNeeded, err := detectMigrationNeeded(dir)
	if err != nil {
		return output.NewError("migration_check_failed", fmt.Sprintf("checking for migration: %v", err)).
			WithNextAction("Verify .spektacular directory structure and permissions.")
	}
	if migrationNeeded {
		result := VersionCheckResult{
			Status:         "migration_needed",
			CurrentVersion: version,
			Action:         migrationPrompt(),
		}
		return output.New(cmd.OutOrStdout(), globalFields).WriteResult(result)
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

// migrationPrompt composes the instruction text for the migration_needed status.
func migrationPrompt() string {
	return "the installed Spektacular configuration uses the legacy single-file format — " +
		"a config.yaml/repo.yaml split is needed for multi-repo project support. " +
		"The migration will create a backup (config.yaml.old) and generate repo.yaml with " +
		"metadata scanned from your project files. Run the migration to proceed."
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
	if cfg, err := loadConfigLenient(); err == nil {
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

// detectMigrationNeeded checks whether a legacy single-file config.yaml
// exists without a corresponding repo.yaml, indicating migration is needed.
// Returns true when migration should be offered, false otherwise.
func detectMigrationNeeded(dataDir string) (bool, error) {
	configPath := filepath.Join(dataDir, "config.yaml")
	repoPath := filepath.Join(dataDir, "repo.yaml")

	// Check if config.yaml exists
	_, err := os.Stat(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No config.yaml means no project initialized yet
			return false, nil
		}
		return false, fmt.Errorf("checking config.yaml: %w", err)
	}

	// Check if repo.yaml exists
	_, err = os.Stat(repoPath)
	if err != nil {
		if os.IsNotExist(err) {
			// config.yaml exists but repo.yaml doesn't - migration needed
			return true, nil
		}
		return false, fmt.Errorf("checking repo.yaml: %w", err)
	}

	// Both files exist - already migrated
	return false, nil
}

// scanProjectMetadata reads project files to infer appropriate repo.yaml
// metadata values. Uses best-effort heuristics with graceful fallback to
// defaults. Always returns a valid RepoConfig, never fails.
func scanProjectMetadata(projectRoot string) (*config.RepoConfig, error) {
	cfg := config.NewDefaultRepoConfig()

	// Try to extract description from README.md
	readmePath := filepath.Join(projectRoot, "README.md")
	if data, err := os.ReadFile(readmePath); err == nil {
		content := string(data)
		// Try to find first H1 title
		if idx := strings.Index(content, "# "); idx != -1 {
			end := strings.Index(content[idx:], "\n")
			if end != -1 {
				cfg.Description = strings.TrimSpace(content[idx+2 : idx+end])
			}
		}
		// If no H1, try first paragraph
		if cfg.Description == "" {
			lines := strings.Split(content, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" && !strings.HasPrefix(line, "#") {
					cfg.Description = line
					break
				}
			}
		}
	}

	// Detect Go projects via go.mod
	goModPath := filepath.Join(projectRoot, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		cfg.Role = "application"
		cfg.Tags = append(cfg.Tags, "go")
	}

	// Detect Node.js projects via package.json
	packageJSONPath := filepath.Join(projectRoot, "package.json")
	if _, err := os.Stat(packageJSONPath); err == nil {
		if cfg.Role == "" {
			cfg.Role = "application"
		}
		cfg.Tags = append(cfg.Tags, "nodejs")
	}

	// Infer deployment method from Makefile/Dockerfile
	makefilePath := filepath.Join(projectRoot, "Makefile")
	dockerfilePath := filepath.Join(projectRoot, "Dockerfile")
	if _, err := os.Stat(dockerfilePath); err == nil {
		cfg.Deployment = "docker"
	} else if _, err := os.Stat(makefilePath); err == nil {
		cfg.Deployment = "make"
	}

	// Ensure defaults if nothing was detected
	if cfg.Description == "" {
		cfg.Description = "A Spektacular project"
	}
	if cfg.Role == "" {
		cfg.Role = "application"
	}
	if len(cfg.Tags) == 0 {
		cfg.Tags = []string{"general"}
	}
	if cfg.Deployment == "" {
		cfg.Deployment = "manual"
	}

	return &cfg, nil
}

// executeMigration performs the migration sequence: create a config.yaml.old
// backup of the legacy config, write repo.yaml with the provided metadata,
// and verify both landed on disk.
func executeMigration(dataDir string, repoCfg *config.RepoConfig) error {
	configPath := filepath.Join(dataDir, "config.yaml")
	backupPath := filepath.Join(dataDir, "config.yaml.old")
	repoPath := filepath.Join(dataDir, "repo.yaml")

	// Step 1: Create backup of config.yaml
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("reading config.yaml for backup: %w", err)
	}
	if err := os.WriteFile(backupPath, configData, 0644); err != nil {
		return fmt.Errorf("creating config.yaml.old backup: %w", err)
	}

	// Step 2: Write repo.yaml with provided metadata
	if err := repoCfg.ToYAMLFile(repoPath); err != nil {
		return fmt.Errorf("writing repo.yaml: %w", err)
	}

	// Step 3: Verify both files exist
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("verifying config.yaml.old exists: %w", err)
	}
	if _, err := os.Stat(repoPath); err != nil {
		return fmt.Errorf("verifying repo.yaml exists: %w", err)
	}

	return nil
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
