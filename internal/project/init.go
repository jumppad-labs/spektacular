// Package project handles Spektacular project initialisation.
package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nicholasjackson/spektacular/internal/config"
	"github.com/nicholasjackson/spektacular/internal/defaults"
)

// Init creates the .spektacular directory structure in projectPath.
// If force is false and the directory already exists, an error is returned.
func Init(projectPath string, force bool) error {
	spektacularDir := filepath.Join(projectPath, ".spektacular")

	if _, err := os.Stat(spektacularDir); err == nil && !force {
		return fmt.Errorf(".spektacular directory already exists at %s; use --force to overwrite", spektacularDir)
	}

	dirs := []string{
		spektacularDir,
		filepath.Join(spektacularDir, "plans"),
		filepath.Join(spektacularDir, "specs"),
		filepath.Join(spektacularDir, "knowledge"),
		filepath.Join(spektacularDir, "knowledge", "learnings"),
		filepath.Join(spektacularDir, "knowledge", "architecture"),
		filepath.Join(spektacularDir, "knowledge", "gotchas"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}

	// Write default config.yaml
	cfg := config.NewDefault()
	if err := cfg.ToYAMLFile(filepath.Join(spektacularDir, "config.yaml")); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	// Write embedded .gitignore
	gitignoreContent, err := defaults.ReadFile(".gitignore")
	if err != nil {
		return fmt.Errorf("reading embedded .gitignore: %w", err)
	}
	if err := os.WriteFile(filepath.Join(spektacularDir, ".gitignore"), gitignoreContent, 0644); err != nil {
		return fmt.Errorf("writing .gitignore: %w", err)
	}

	// Write embedded conventions.md
	conventionsContent, err := defaults.ReadFile("conventions.md")
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
