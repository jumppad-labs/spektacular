// Package project handles Spektacular project initialisation.
package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jumppad-labs/spektacular/internal/config"
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
		filepath.Join(spektacularDir, "knowledge", "learnings"),
		filepath.Join(spektacularDir, "knowledge", "architecture"),
		filepath.Join(spektacularDir, "knowledge", "gotchas"),
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

	// Write embedded conventions.md
	conventionsContent, err := templates.FS.ReadFile(".spektacular/conventions.md")
	if err != nil {
		return fmt.Errorf("reading embedded conventions.md: %w", err)
	}
	if err := os.WriteFile(filepath.Join(spektacularDir, "knowledge", "conventions.md"), conventionsContent, 0644); err != nil {
		return fmt.Errorf("writing conventions.md: %w", err)
	}

	// Write README files for knowledge subdirectories
	for _, sub := range []string{"learnings", "architecture", "gotchas"} {
		title := strings.Title(sub) //nolint:staticcheck // simple capitalisation
		content := fmt.Sprintf("# %s\n\nThis directory contains %s documentation.\n", title, sub)
		readmePath := filepath.Join(spektacularDir, "knowledge", sub, "README.md")
		if err := os.WriteFile(readmePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("writing %s README: %w", sub, err)
		}
	}

	return nil
}
