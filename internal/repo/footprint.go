package repo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/knowledge"
)

// Footprint statuses reported by EnsureFootprint.
const (
	FootprintCreated   = "created"
	FootprintRepaired  = "repaired"
	FootprintUnchanged = "unchanged"
)

// EnsureFootprint creates or repairs a repo's minimal Spektacular footprint
// at root: the repo config file plus the knowledge storage its sources
// declare — and nothing else (no agent guidance, no skills, no version
// file). It is idempotent and strictly additive: an existing repo.yaml is
// kept (and drives the scaffolding) unless it is broken, existing knowledge
// files are never overwritten, and a repo initialized by another project is
// left undisturbed. The returned status reports what happened: created (no
// repo.yaml existed), repaired (repo.yaml existed but was broken, or parts
// of the knowledge storage were missing), or unchanged.
func EnsureFootprint(root string, repoCfg config.RepoConfig) (string, error) {
	spektacularDir := filepath.Join(root, ".spektacular")
	repoConfigPath := filepath.Join(spektacularDir, config.RepoConfigFileName)

	status := FootprintUnchanged
	if _, err := os.Stat(repoConfigPath); os.IsNotExist(err) {
		status = FootprintCreated
		if err := os.MkdirAll(spektacularDir, 0755); err != nil {
			return "", fmt.Errorf("creating directory %s: %w", spektacularDir, err)
		}
		if err := repoCfg.ToYAMLFile(repoConfigPath); err != nil {
			return "", err
		}
	} else if loaded, err := config.RepoConfigFromYAMLFile(repoConfigPath); err != nil {
		// A broken repo config is repaired by rewriting it from the given
		// defaults — the footprint must end the call valid.
		status = FootprintRepaired
		if err := repoCfg.ToYAMLFile(repoConfigPath); err != nil {
			return "", err
		}
	} else {
		// A healthy existing config is the authority for its own footprint.
		repoCfg = loaded
	}

	// Scaffold the knowledge storage the repo config declares: the source
	// root plus a directory and README for every category in the registry.
	// Only missing pieces are created, so an already-initialized repo is
	// never disturbed.
	for _, src := range repoCfg.WithDefaults(root).Knowledge.Sources {
		if src.Provider != config.ProviderFile || src.Scope != config.DefaultKnowledgeScope {
			continue
		}
		location := src.Config.Location
		if !filepath.IsAbs(location) {
			location = filepath.Join(root, location)
		}
		for _, c := range knowledge.Categories {
			dir := filepath.Join(location, c.Name)
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				if status == FootprintUnchanged {
					status = FootprintRepaired
				}
				if err := os.MkdirAll(dir, 0755); err != nil {
					return "", fmt.Errorf("creating directory %s: %w", dir, err)
				}
			}
			readmePath := filepath.Join(dir, "README.md")
			if _, err := os.Stat(readmePath); os.IsNotExist(err) {
				if status == FootprintUnchanged {
					status = FootprintRepaired
				}
				if err := os.WriteFile(readmePath, []byte(c.README()), 0644); err != nil {
					return "", fmt.Errorf("writing %s README: %w", c.Name, err)
				}
			}
		}
	}

	return status, nil
}
