// Package project handles Spektacular project initialisation.
package project

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/templates"
)

// InitOptions configures Spektacular project initialisation.
type InitOptions struct {
	Force                   bool
	Config                  config.Config
	CreateLocalArtifactDirs bool
	CreateNotionCacheDir    bool
}

// Init creates the .spektacular directory structure in projectPath.
// If force is false and the directory already exists, an error is returned.
func Init(projectPath string, force bool) error {
	return InitWithOptions(projectPath, InitOptions{
		Force:                   force,
		Config:                  config.NewDefault(),
		CreateLocalArtifactDirs: true,
	})
}

// InitWithOptions creates the .spektacular directory structure in projectPath
// using mode-specific artifact directory options.
func InitWithOptions(projectPath string, opts InitOptions) error {
	spektacularDir := filepath.Join(projectPath, ".spektacular")

	if _, err := os.Stat(spektacularDir); err == nil && !opts.Force {
		return fmt.Errorf(".spektacular directory already exists at %s; use --force to overwrite", spektacularDir)
	}

	cfg := opts.Config
	if reflect.DeepEqual(cfg, config.Config{}) {
		cfg = config.NewDefault()
	}

	dirs := []string{
		spektacularDir,
		filepath.Join(spektacularDir, "knowledge"),
		filepath.Join(spektacularDir, "knowledge", "learnings"),
		filepath.Join(spektacularDir, "knowledge", "architecture"),
		filepath.Join(spektacularDir, "knowledge", "gotchas"),
	}
	if opts.CreateLocalArtifactDirs {
		dirs = append(dirs,
			filepath.Join(spektacularDir, "plans"),
			filepath.Join(spektacularDir, "specs"),
		)
	}
	if opts.CreateNotionCacheDir {
		dirs = append(dirs, filepath.Join(spektacularDir, cfg.Artifacts.CacheDir))
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}

	// Write default config.yaml only if it does not already exist.
	configPath := filepath.Join(spektacularDir, "config.yaml")
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
