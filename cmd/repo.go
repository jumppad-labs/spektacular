package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/output"
	"github.com/jumppad-labs/spektacular/internal/repo"
	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage the project's registered member repos",
	RunE:  runUnknownSubcommand,
}

var repoAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Register a repo in the project config and create its minimal footprint",
	RunE:  runRepoAdd,
}

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the project's registered repos with metadata and resolved locations",
	RunE:  runRepoList,
}

// repoGit is the GitRunner the repo commands resolve with; a variable so
// tests can substitute a fake.
var repoGit repo.GitRunner = repo.NewGitRunner()

var repoAddInputSchema = &schemaObj{
	Type: "object",
	Properties: map[string]*schemaProp{
		"name":         {Type: "string"},
		"address":      {Type: "string"},
		"local":        {Type: "string"},
		"description":  {Type: "string"},
		"role":         {Type: "string"},
		"tags":         {Type: "array", Items: &schemaProp{Type: "string"}},
		"dependencies": {Type: "array", Items: &schemaProp{Type: "string"}},
		"deployment":   {Type: "string"},
	},
	Required: []string{"name"},
}

var repoAddOutputSchema = &schemaObj{
	Type: "object",
	Properties: map[string]*schemaProp{
		"registered":    {Type: "boolean"},
		"footprint":     {Type: "string", Enum: []string{repo.FootprintCreated, repo.FootprintRepaired, repo.FootprintUnchanged}},
		"metadata_note": {Type: "string"},
	},
}

var repoListOutputSchema = &schemaObj{
	Type: "object",
	Properties: map[string]*schemaProp{
		"repos": {
			Type: "array",
			Items: &schemaProp{
				Type: "object",
				Properties: map[string]*schemaProp{
					"name":         {Type: "string"},
					"address":      {Type: "string"},
					"local":        {Type: "string"},
					"root":         {Type: "string"},
					"provider":     {Type: "string"},
					"description":  {Type: "string"},
					"role":         {Type: "string"},
					"tags":         {Type: "array", Items: &schemaProp{Type: "string"}},
					"dependencies": {Type: "array", Items: &schemaProp{Type: "string"}},
					"deployment":   {Type: "string"},
					"materialized":  {Type: "boolean"},
					"stale_note":    {Type: "string"},
					"metadata_note": {Type: "string"},
				},
			},
		},
	},
}

// repoAddInput is the --data payload for repo add, mirroring RepoEntry's
// provider-agnostic identity and metadata fields.
type repoAddInput struct {
	Name         string   `json:"name"`
	Address      string   `json:"address"`
	Local        string   `json:"local"`
	Description  string   `json:"description"`
	Role         string   `json:"role"`
	Tags         []string `json:"tags"`
	Dependencies []string `json:"dependencies"`
	Deployment   string   `json:"deployment"`
}

// repoInfo is the list projection agents consume for cross-repo attribution.
type repoInfo struct {
	Name         string   `json:"name"`
	Address      string   `json:"address,omitempty"`
	Local        string   `json:"local,omitempty"`
	Root         string   `json:"root"`
	Provider     string   `json:"provider"`
	Description  string   `json:"description,omitempty"`
	Role         string   `json:"role,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
	Deployment   string   `json:"deployment,omitempty"`
	Materialized bool     `json:"materialized"`
	StaleNote    string   `json:"stale_note,omitempty"`
	MetadataNote string   `json:"metadata_note,omitempty"`
}

func runRepoAdd(cmd *cobra.Command, _ []string) error {
	if schema, _ := cmd.Flags().GetBool("schema"); schema {
		return output.Write(cmd.OutOrStdout(), commandSchema{Input: repoAddInputSchema, Output: repoAddOutputSchema}, "")
	}

	input, err := repoAddData(cmd)
	if err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	entry := config.RepoEntry{
		Name:         input.Name,
		Address:      input.Address,
		Local:        input.Local,
		Dependencies: input.Dependencies,
	}

	// Idempotent registration by name: an identical entry is a no-op, a
	// changed entry updates in place, a new one appends. The config write is
	// skipped entirely when nothing changed.
	changed := true
	found := false
	for i, existing := range cfg.Repos {
		if existing.Name == entry.Name {
			found = true
			// Preserve provider settings the input shape doesn't carry.
			entry.Provider = existing.Provider
			entry.Config = existing.Config
			if reposEqual(existing, entry) {
				changed = false
			} else {
				cfg.Repos[i] = entry
			}
			break
		}
	}
	if !found {
		cfg.Repos = append(cfg.Repos, entry)
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	if changed {
		cfgPath, err := configFilePath()
		if err != nil {
			return err
		}
		if err := cfg.ToYAMLFile(cfgPath); err != nil {
			return fmt.Errorf("writing config: %w", err)
		}
	}

	// Create or repair the target repo's minimal footprint on its resolved
	// root. Resolution materializes an address-only repo by cloning; a
	// footprint error from resolution is expected here — creating that
	// footprint is this command's job.
	root, err := projectRoot()
	if err != nil {
		return err
	}
	set, err := repo.New(cfg, root, repoGit)
	if err != nil {
		return err
	}
	resolved, err := set.Resolve(entry.Name)
	if err != nil {
		var fpErr *repo.FootprintError
		if !errors.As(err, &fpErr) {
			return err
		}
	}

	status, err := repo.EnsureFootprint(resolved.Root, config.NewDefaultRepoConfig())
	if err != nil {
		return err
	}

	// Write the input's descriptive fields into the target repo's own
	// config, updating an already-footprinted repo rather than only a
	// freshly created one. EnsureFootprint's caller-supplied config has no
	// effect once a repo.yaml already exists, so this is a separate step.
	repoConfigPath := filepath.Join(resolved.Root, ".spektacular", config.RepoConfigFileName)
	repoCfg, err := config.RepoConfigFromYAMLFile(repoConfigPath)
	if err != nil {
		return fmt.Errorf("reading repo config: %w", err)
	}
	updated := repoCfg
	if input.Description != "" {
		updated.Description = input.Description
	}
	if input.Role != "" {
		updated.Role = input.Role
	}
	if len(input.Tags) > 0 {
		updated.Tags = input.Tags
	}
	if input.Deployment != "" {
		updated.Deployment = input.Deployment
	}
	if !repoConfigDescriptiveFieldsEqual(repoCfg, updated) {
		if err := updated.ToYAMLFile(repoConfigPath); err != nil {
			return fmt.Errorf("writing repo config: %w", err)
		}
	}

	result := map[string]any{"registered": true, "footprint": status}
	if repoConfigDescriptiveFieldsEmpty(updated) {
		result["metadata_note"] = fmt.Sprintf("repo %q has no descriptive metadata set; pass description/role/tags/deployment so it's consistently described", entry.Name)
	}

	out := output.New(cmd.OutOrStdout(), globalFields)
	return out.WriteResult(result)
}

func runRepoList(cmd *cobra.Command, _ []string) error {
	if schema, _ := cmd.Flags().GetBool("schema"); schema {
		return output.Write(cmd.OutOrStdout(), commandSchema{Input: nil, Output: repoListOutputSchema}, "")
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	root, err := projectRoot()
	if err != nil {
		return err
	}
	set, err := repo.New(cfg, root, repoGit)
	if err != nil {
		return err
	}

	// Listing is inspection and must stay side-effect-free: entries whose
	// repo is already on disk resolve to their root (no cloning happens for
	// them), while address-only repos not yet materialized are reported
	// unresolved rather than triggering a clone.
	infos := make([]repoInfo, 0, len(set.Entries()))
	for _, e := range set.Entries() {
		info := repoInfo{
			Name:         e.Name,
			Address:      e.Address,
			Local:        e.Local,
			Provider:     e.Provider,
			Dependencies: e.Dependencies,
		}
		if set.Present(e.Name) {
			resolved, err := set.Resolve(e.Name)
			if err != nil {
				var fpErr *repo.FootprintError
				if !errors.As(err, &fpErr) {
					return err
				}
				// A broken footprint doesn't hide the repo from the listing;
				// repair is offered when the repo is actually used.
			}
			info.Root = resolved.Root
			info.Materialized = resolved.Materialized
			info.StaleNote = resolved.StaleNote
			// Descriptive metadata comes from the repo's own config, the
			// single source now that the registry carries membership only.
			meta, ok := set.DescriptiveMetadata(e.Name)
			if ok {
				info.Description = meta.Description
				info.Role = meta.Role
				info.Tags = meta.Tags
				info.Deployment = meta.Deployment
			}
			if !ok || repoConfigDescriptiveFieldsEmpty(meta) {
				info.MetadataNote = fmt.Sprintf("repo %q has no descriptive metadata set; run 'repo add' with description/role/tags/deployment to describe it", e.Name)
			}
		}
		infos = append(infos, info)
	}

	out := output.New(cmd.OutOrStdout(), globalFields)
	return out.WriteResult(map[string]any{"repos": infos})
}

// repoAddData parses and validates the --data flag for repo add.
func repoAddData(cmd *cobra.Command) (repoAddInput, error) {
	dataStr, _ := cmd.Flags().GetString("data")
	if dataStr == "" {
		return repoAddInput{}, fmt.Errorf(`--data is required (e.g. --data '{"name":"docs","address":"git@example.com:org/docs.git","description":"the documentation repo"}')`)
	}
	var input repoAddInput
	if err := json.Unmarshal([]byte(dataStr), &input); err != nil {
		return repoAddInput{}, fmt.Errorf("parsing --data: %w", err)
	}
	if input.Name == "" {
		return repoAddInput{}, fmt.Errorf(`--data must include a non-empty "name"`)
	}
	return input, nil
}

// reposEqual reports whether two registry entries are identical field for
// field. RepoEntry carries membership only; descriptive metadata lives in
// the repo's own config and is compared separately.
func reposEqual(a, b config.RepoEntry) bool {
	if a.Name != b.Name || a.Address != b.Address || a.Local != b.Local || a.Provider != b.Provider {
		return false
	}
	return stringSlicesEqual(a.Dependencies, b.Dependencies)
}

func repoConfigDescriptiveFieldsEqual(a, b config.RepoConfig) bool {
	if a.Description != b.Description || a.Role != b.Role || a.Deployment != b.Deployment {
		return false
	}
	return stringSlicesEqual(a.Tags, b.Tags)
}

// repoConfigDescriptiveFieldsEmpty reports whether a repo config carries no
// descriptive metadata at all, the case this project wants surfaced with a
// warn-only notice so every repo ends up consistently described.
func repoConfigDescriptiveFieldsEmpty(c config.RepoConfig) bool {
	return c.Description == "" && c.Role == "" && c.Deployment == "" && len(c.Tags) == 0
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func init() {
	repoCmd.PersistentFlags().Bool("schema", false, "Print the input/output schema for this subcommand and exit")
	repoAddCmd.Flags().StringP("data", "d", "", `JSON input (e.g. '{"name":"docs","address":"git@example.com:org/docs.git"}')`)
	repoCmd.AddCommand(repoAddCmd, repoListCmd)
}
