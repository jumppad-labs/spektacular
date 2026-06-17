// Package project handles Spektacular project initialisation.
package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/knowledge"
	"github.com/jumppad-labs/spektacular/templates"
)

// Init creates the .spektacular directory structure in projectPath.
// If force is false and the directory already exists, an error is returned.
func Init(projectPath string, force bool) error {
	spektacularDir := filepath.Join(projectPath, ".spektacular")

	if _, err := os.Stat(spektacularDir); err == nil && !force {
		return fmt.Errorf(".spektacular directory already exists at %s; use --force to overwrite", spektacularDir)
	}

	// Resolve the spec, plan, and knowledge configuration from an existing
	// config.yaml, or fall back to the defaults when no config exists yet.
	configPath := filepath.Join(spektacularDir, "config.yaml")
	cfg := config.NewDefault()
	if _, err := os.Stat(configPath); err == nil {
		loaded, err := config.FromYAMLFile(configPath)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		cfg = loaded
	}

	dirs := []string{
		spektacularDir,
		// Spec and plan directories are configured as project-root-relative
		// paths (e.g. ".spektacular/plans"), like the knowledge location.
		filepath.Join(projectPath, cfg.Plan.Config.Directory),
		filepath.Join(projectPath, cfg.Spec.Config.Directory),
		filepath.Join(spektacularDir, "knowledge"),
	}

	// Scaffold a directory for every category in the knowledge registry, which
	// is the single source of truth for the category model (including the
	// always-applied glossary and the looked-up decisions category).
	for _, c := range knowledge.Categories {
		dirs = append(dirs, filepath.Join(spektacularDir, "knowledge", c.Name))
	}

	// Create the directory for the project knowledge source so the knowledge
	// commands can reach it. Only the project scope is created by init; team
	// and global sources are shared and expected to exist independently.
	// WithDefaults covers the synthesised default project source, and relative
	// locations resolve against the project root as knowledge.NewSet resolves them.
	for _, src := range cfg.Knowledge.WithDefaults(projectPath).Sources {
		if src.Provider != config.ProviderFile || src.Scope != config.DefaultKnowledgeScope {
			continue
		}
		location := src.Config.Location
		if !filepath.IsAbs(location) {
			location = filepath.Join(projectPath, location)
		}
		dirs = append(dirs, location)
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}

	// Write default config.yaml only if it does not already exist.
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := cfg.ToYAMLFile(configPath); err != nil {
			return fmt.Errorf("writing config: %w", err)
		}
	}

	// Write embedded .gitignore
	gitignoreContent, err := templates.FS.ReadFile(".spektacular/.gitignore")
	if err != nil {
		return fmt.Errorf("reading embedded .gitignore: %w", err)
	}
	if err := os.WriteFile(filepath.Join(spektacularDir, ".gitignore"), gitignoreContent, 0644); err != nil {
		return fmt.Errorf("writing .gitignore: %w", err)
	}

	// Write a README for each knowledge category, rendered from its registry
	// definition so the directory documents its own purpose, boundary, tier,
	// and entry shape rather than carrying a circular placeholder.
	for _, c := range knowledge.Categories {
		title := strings.Title(c.Name) //nolint:staticcheck // simple capitalisation
		content := fmt.Sprintf(
			"# %s\n\n**Tier:** %s\n\n**Purpose:** %s\n\n**Belongs elsewhere:** %s\n\n**Entry shape:** %s\n",
			title, c.Tier, c.Purpose, c.Boundary, c.EntryShape,
		)
		readmePath := filepath.Join(spektacularDir, "knowledge", c.Name, "README.md")
		if err := os.WriteFile(readmePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("writing %s README: %w", c.Name, err)
		}
	}

	return nil
}
