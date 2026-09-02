package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
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
	Short: "List the project's registered repos with metadata and the resolved root of each repo's code",
	RunE:  runRepoList,
}

// repoGit is the GitRunner the repo commands resolve with; a variable so
// tests can substitute a fake.
var repoGit repo.GitRunner = repo.NewGitRunner()

var repoAddInputSchema = &schemaObj{
	Type: "object",
	Properties: map[string]*schemaProp{
		"name":         {Type: "string"},
		"location":     {Type: "string"},
		"source":       {Type: "string"},
		"description":  {Type: "string"},
		"role":         {Type: "string"},
		"tags":         {Type: "array", Items: &schemaProp{Type: "string"}},
		"dependencies": {Type: "array", Items: &schemaProp{Type: "string"}},
	},
	Required: []string{"name", "location"},
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
					"name":          {Type: "string"},
					"location":      {Type: "string"},
					"root":          {Type: "string"},
					"provider":      {Type: "string"},
					"description":   {Type: "string"},
					"role":          {Type: "string"},
					"tags":          {Type: "array", Items: &schemaProp{Type: "string"}},
					"dependencies":  {Type: "array", Items: &schemaProp{Type: "string"}},
					"materialized":  {Type: "boolean"},
					"stale_note":    {Type: "string"},
					"metadata_note": {Type: "string"},
				},
			},
		},
	},
}

// repoAddInput is the --data payload for repo add: the registry entry's
// identity and location, plus the fields written into the repo's own
// config — its descriptive metadata and, when given, the source its code
// lives at.
type repoAddInput struct {
	Name         string   `json:"name"`
	Location     string   `json:"location"`
	Source       string   `json:"source"`
	Description  string   `json:"description"`
	Role         string   `json:"role"`
	Tags         []string `json:"tags"`
	Dependencies []string `json:"dependencies"`
}

// repoInfo is the list projection agents consume for cross-repo attribution.
type repoInfo struct {
	Name         string   `json:"name"`
	Location     string   `json:"location,omitempty"`
	Root         string   `json:"root"`
	Provider     string   `json:"provider,omitempty"`
	Description  string   `json:"description,omitempty"`
	Role         string   `json:"role,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
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

	// `repo add` is pointed at the repo's code. Its Spektacular footprint is
	// scaffolded into a .spektacular folder inside it, and that folder — the
	// one that ends up holding repo.yaml — is what the registry records,
	// since a registered location names the folder holding repo.yaml and
	// nothing is appended when reading it back.
	entry := config.RepoEntry{
		Name:         input.Name,
		Location:     filepath.Join(input.Location, ".spektacular"),
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

	// Create or repair the target repo's minimal footprint at its registered
	// location, creating that folder when missing: the location is the
	// footprint folder holding the repo's repo.yaml, so making it exist is
	// registration's job.
	root, err := projectRoot()
	if err != nil {
		return err
	}
	set, err := repo.New(cfg, root, repoGit)
	if err != nil {
		return err
	}
	location := entry.ResolvedLocation(root)
	if err := os.MkdirAll(location, 0755); err != nil {
		return fmt.Errorf("creating repo location %s: %w", location, err)
	}

	// A scaffolded footprint sits inside the repo's code, so its default
	// source is the parent folder. An explicit source in this same add
	// overrides it below.
	scaffold := config.NewDefaultRepoConfig()
	scaffold.Source = config.DefaultRepoSource
	status, err := repo.EnsureFootprint(location, scaffold)
	if err != nil {
		return err
	}

	// Write the input's descriptive fields and source into the target repo's
	// own config, updating an already-footprinted repo rather than only a
	// freshly created one. EnsureFootprint's caller-supplied config has no
	// effect once a repo.yaml already exists, so this is a separate step.
	// It runs before resolution so a git source given in this same add is
	// honoured by the clone below.
	repoConfigPath := filepath.Join(location, config.RepoConfigFileName)
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
	if input.Source != "" {
		src, err := config.SourceFromInput(input.Source)
		if err != nil {
			return output.NewError("invalid_data", err.Error()).
				WithNextAction(`give "source" as a path (plain or file://) or a git location (git://, ssh://, https://, or user@host:path)`)
		}
		updated.Source = src
	}
	if !repoConfigDescriptiveFieldsEqual(repoCfg, updated) {
		if err := updated.ToYAMLFile(repoConfigPath); err != nil {
			return fmt.Errorf("writing repo config: %w", err)
		}
	}

	// Resolve once so a git source is materialized (cloned into the
	// project's working folder) on registration rather than on first use.
	if _, err := set.Resolve(entry.Name); err != nil {
		return err
	}

	result := map[string]any{"registered": true, "footprint": status}
	if repoConfigDescriptiveFieldsEmpty(updated) {
		result["metadata_note"] = fmt.Sprintf("repo %q has no descriptive metadata set; pass description/role/tags so it's consistently described", entry.Name)
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
	// location is on disk resolve to their root (no cloning happens for
	// them), while repos whose location is missing are reported unresolved
	// rather than triggering any side effect.
	infos := make([]repoInfo, 0, len(set.Entries()))
	for _, e := range set.Entries() {
		info := repoInfo{
			Name:         e.Name,
			Location:     e.ResolvedLocation(root),
			Dependencies: e.Dependencies,
		}
		// A registered repo whose location is not on disk, or whose footprint
		// is missing or unreadable, is a misregistration: report it as an
		// error naming the path that was looked at, rather than listing the
		// repo with an empty root and no metadata as if that were fine.
		meta, err := set.Footprint(e.Name)
		if err != nil {
			var fpErr *repo.FootprintError
			if errors.As(err, &fpErr) {
				return output.NewError("repo_footprint_missing", fpErr.Error()).
					WithResource(fpErr.Root).
					WithNextAction(fmt.Sprintf("expected %s; run 'repo add' with name %q and location %q to scaffold it, or correct repos[].location for %q", filepath.Join(fpErr.Root, config.RepoConfigFileName), e.Name, e.Location, e.Name))
			}
			return err
		}
		// Descriptive metadata comes from the repo's own config at its
		// location, the single source now that the registry carries
		// membership only.
		// The reported provider is the one the repo declares for its own
		// source, so it says how the code at `root` was reached: `file` for a
		// directory, `git` for a clone. A repo declaring no source has none.
		info.Provider = meta.Source.Provider
		info.Description = meta.Description
		info.Role = meta.Role
		info.Tags = meta.Tags
		if repoConfigDescriptiveFieldsEmpty(meta) {
			info.MetadataNote = fmt.Sprintf("repo %q has no descriptive metadata set; run 'repo add' with description/role/tags to describe it", e.Name)
		}
		// The reported root is the repo's code: its source when one is
		// declared, otherwise the location itself. A git source that has
		// not been cloned yet is reported unresolved rather than cloned —
		// resolution only runs once the code is already on disk.
		if _, ok := set.LocalSource(e.Name); ok {
			resolved, err := set.Resolve(e.Name)
			if err != nil {
				return err
			}
			info.Root = resolved.Source
			info.Materialized = resolved.Materialized
			info.StaleNote = resolved.StaleNote
		}
		infos = append(infos, info)
	}

	out := output.New(cmd.OutOrStdout(), globalFields)
	return out.WriteResult(map[string]any{"repos": infos})
}

// repoAddData parses and validates the --data flag for repo add.
func repoAddData(cmd *cobra.Command) (repoAddInput, error) {
	const example = `{"name":"docs","location":"../repos/docs","source":"git@example.com:org/docs.git","description":"the documentation repo"}`
	dataStr, _ := cmd.Flags().GetString("data")
	if dataStr == "" {
		return repoAddInput{}, fmt.Errorf(`--data is required (e.g. --data '%s')`, example)
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(dataStr), &raw); err != nil {
		return repoAddInput{}, fmt.Errorf("parsing --data: %w", err)
	}
	if addr, ok := raw["address"]; ok {
		return repoAddInput{}, output.NewError("invalid_data", "--data no longer accepts \"address\"; a repo's git origin is its \"source\", written into the repo's own repo.yaml").
			WithNextAction(fmt.Sprintf(`re-run with --data '{"name":%q,"location":"<folder holding the repo's .spektacular/>","source":"%v"}'`, raw["name"], addr))
	}
	var input repoAddInput
	if err := json.Unmarshal([]byte(dataStr), &input); err != nil {
		return repoAddInput{}, fmt.Errorf("parsing --data: %w", err)
	}
	if input.Name == "" {
		return repoAddInput{}, fmt.Errorf(`--data must include a non-empty "name"`)
	}
	if input.Location == "" {
		return repoAddInput{}, output.NewError("invalid_data", fmt.Sprintf(`--data must include a non-empty "location": the folder holding %q's .spektacular/ directory (created if missing)`, input.Name)).
			WithNextAction(fmt.Sprintf(`re-run with --data '%s'`, example))
	}
	return input, nil
}

// reposEqual reports whether two registry entries are identical field for
// field. RepoEntry carries membership only; descriptive metadata lives in
// the repo's own config and is compared separately.
func reposEqual(a, b config.RepoEntry) bool {
	if a.Name != b.Name || a.Location != b.Location || a.Provider != b.Provider {
		return false
	}
	return stringSlicesEqual(a.Dependencies, b.Dependencies)
}

// repoConfigDescriptiveFieldsEqual reports whether the fields repo add
// writes — the descriptive metadata and the source — match, so the repo's
// config is rewritten only when registration actually changed something.
func repoConfigDescriptiveFieldsEqual(a, b config.RepoConfig) bool {
	if a.Description != b.Description || a.Role != b.Role || a.Source != b.Source {
		return false
	}
	return stringSlicesEqual(a.Tags, b.Tags)
}

// repoConfigDescriptiveFieldsEmpty reports whether a repo config carries no
// descriptive metadata at all, the case this project wants surfaced with a
// warn-only notice so every repo ends up consistently described. Source is
// a location, not a description, so it does not count.
func repoConfigDescriptiveFieldsEmpty(c config.RepoConfig) bool {
	return c.Description == "" && c.Role == "" && len(c.Tags) == 0
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
