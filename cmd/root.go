package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/output"
	"github.com/jumppad-labs/spektacular/internal/sessionlog"
	"github.com/spf13/cobra"
)

// version and sha are set at build time via -ldflags
// (see dagger/main.go's Build function).
var (
	version = "0.1.0"
	sha     = ""
)

// globalFields holds the raw --fields JSON array string, available to all subcommands.
var globalFields string

var rootCmd = &cobra.Command{
	Use:     "spektacular",
	Short:   "Agent-driven tool for spec-driven development",
	Version: versionString(),
}

// versionString combines the build-time version and sha into the string
// Cobra prints for --version. sha is omitted when unset (e.g. local go
// build/go run, which never pass -ldflags).
func versionString() string {
	if sha == "" {
		return version
	}
	return fmt.Sprintf("%s (%s)", version, sha)
}

func Execute() {
	os.Exit(runRoot())
}

// runRoot runs the root command and returns the process exit code. It is the
// single place a command's outcome (result or error) is translated into the
// response envelope and written to the output stream — extracted from
// Execute so tests can exercise the full wrapping behavior without an
// os.Exit call.
//
// When the project's debug.enabled config option is on, runRoot also
// records the command issued and the exact response returned to a local
// session log, for reconstructing a session after the fact. This is
// strictly additive: with the option off (the default), runRoot's behavior
// is byte-for-byte identical to what it is without any of this.
func runRoot() int {
	cfg, cfgErr := loadConfig()
	debugEnabled := cfgErr == nil && cfg.Debug.Enabled

	var argv []string
	var start time.Time
	var orig io.Writer
	var buf *bytes.Buffer
	var stateBefore *sessionlog.StateSnapshot

	if debugEnabled {
		argv = os.Args[1:]
		start = time.Now()
		stateBefore = readStateSnapshot()
		orig = rootCmd.OutOrStdout()
		buf = &bytes.Buffer{}
		rootCmd.SetOut(io.MultiWriter(orig, buf))
	}

	err := rootCmd.Execute()

	exitCode := 0
	if err != nil {
		output.WriteFailure(rootCmd.OutOrStdout(), toErrorResponse(err), globalFields)
		exitCode = 1
	}

	if debugEnabled {
		stateAfter := readStateSnapshot()
		if logPath, pathErr := sessionLogPath(); pathErr == nil {
			sessionlog.Record(logPath, sessionlog.Event{
				Timestamp:   start,
				SessionID:   sessionlog.SessionID(stateAfter),
				Command:     argv,
				DurationMS:  time.Since(start).Milliseconds(),
				ExitCode:    exitCode,
				Response:    buf.String(),
				StateBefore: stateBefore,
				StateAfter:  stateAfter,
				Advanced:    stateAdvanced(stateBefore, stateAfter),
			})
		}
		rootCmd.SetOut(orig)
	}

	return exitCode
}

// stateSnapshotFile is the subset of internal/workflow's State shape needed
// to build a sessionlog.StateSnapshot, read directly rather than importing
// internal/workflow — cmd already owns the path-construction logic for
// state.json and this keeps sessionlog a pure leaf package.
type stateSnapshotFile struct {
	Kind           string         `json:"kind"`
	CurrentStep    string         `json:"current_step"`
	CompletedSteps []string       `json:"completed_steps"`
	Data           map[string]any `json:"data"`
}

// readStateSnapshot reads .spektacular/state.json and returns the small
// view of it a session record needs, or nil if no workflow has run yet (no
// state file exists) or it can't be read/parsed — never an error, since a
// snapshot read can never be allowed to affect the command's own outcome.
func readStateSnapshot() *sessionlog.StateSnapshot {
	dir, err := dataDir()
	if err != nil {
		return nil
	}

	data, err := os.ReadFile(stateFilePath(dir))
	if err != nil {
		return nil
	}

	var sf stateSnapshotFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil
	}

	name, _ := sf.Data["name"].(string)
	return &sessionlog.StateSnapshot{
		Kind:           sf.Kind,
		Name:           name,
		CurrentStep:    sf.CurrentStep,
		CompletedSteps: sf.CompletedSteps,
	}
}

// stateAdvanced reports whether after meaningfully differs from before —
// the signal that a command actually moved a piece of work forward, as
// opposed to a rejected or silently no-op command (e.g. Goto to the
// already-current step) that returns no error but changes nothing. This is
// deliberately independent of whatever error rootCmd.Execute() returned.
func stateAdvanced(before, after *sessionlog.StateSnapshot) bool {
	if (before == nil) != (after == nil) {
		return true
	}
	if before == nil {
		return false
	}
	return before.Kind != after.Kind ||
		before.Name != after.Name ||
		before.CurrentStep != after.CurrentStep ||
		len(before.CompletedSteps) != len(after.CompletedSteps)
}

// sessionLogPath returns the path to the local session record file, inside
// the project's already-gitignored debug directory.
func sessionLogPath() (string, error) {
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "debug", "session-log.jsonl"), nil
}

// toErrorResponse converts any error returned by a command into the shared
// ErrorResponse shape: an already-built *output.ErrorResponse passes through
// unchanged, anything else falls back to a generic internal_error.
func toErrorResponse(err error) *output.ErrorResponse {
	if er, ok := err.(*output.ErrorResponse); ok {
		return er
	}
	return output.NewError("internal_error", err.Error())
}

func configFilePath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	return filepath.Join(cwd, ".spektacular", "config.yaml"), nil
}

// loadConfig loads the project config from the current working directory.
// Returns defaults if the config file does not exist.
// Returns an error if the config file exists but is invalid.
func loadConfig() (config.Config, error) {
	cfgPath, err := configFilePath()
	if err != nil {
		return config.Config{}, err
	}
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return config.NewDefault(), nil
	}
	return config.FromYAMLFile(cfgPath)
}

// dataDir returns the .spektacular directory for the current working directory.
// Both spec and plan workflows share this directory (and a single state.json).
func dataDir() (string, error) {
	root, err := projectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".spektacular"), nil
}

// projectRoot returns the project root — the current working directory. Spec,
// plan, and knowledge directories from the config are all resolved relative to
// this, so the configured paths (e.g. ".spektacular/specs") are project-root
// relative rather than relative to the .spektacular data directory.
func projectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	return cwd, nil
}

func init() {
	// The response envelope wrapper in runRoot is the only place a command's
	// outcome is ever printed, so the CLI framework's own default error/usage
	// printing is turned off to avoid printing a failure a second time.
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
	rootCmd.PersistentFlags().StringVar(&globalFields, "fields", "", `JSON array of output fields to include (e.g. '["step","instruction"]')`)
	rootCmd.AddCommand(specCmd)
	rootCmd.AddCommand(planCmd)
	rootCmd.AddCommand(changelogCmd)
	rootCmd.AddCommand(implementCmd)
	rootCmd.AddCommand(knowledgeCmd)
	rootCmd.AddCommand(skillCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(artifactsCmd)
}
