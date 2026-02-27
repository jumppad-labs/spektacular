// Package spec handles creation of new specification files from the embedded template.
package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nicholasjackson/spektacular/internal/defaults"
)

// Create writes a new spec file to .spektacular/specs/<name>.md inside projectPath.
// title and description are optional; sensible defaults are derived from name if empty.
// Returns the path of the created file.
func Create(projectPath, name, title, description string) (string, error) {
	if title == "" {
		title = toTitle(name)
	}
	if description == "" {
		description = fmt.Sprintf("Add description for %s here.", title)
	}

	templateBytes, err := defaults.ReadFile("spec-template.md")
	if err != nil {
		return "", fmt.Errorf("reading spec template: %w", err)
	}

	content := string(templateBytes)
	replacements := map[string]string{
		"{title}":          title,
		"{description}":    description,
		"{requirement_1}":  "Add first requirement",
		"{requirement_2}":  "Add second requirement",
		"{requirement_3}":  "Add third requirement",
		"{constraint_1}":   "Add first constraint",
		"{constraint_2}":   "Add second constraint",
		"{criteria_1}":     "Add first acceptance criterion",
		"{criteria_2}":     "Add second acceptance criterion",
		"{criteria_3}":     "Add third acceptance criterion",
		"{technical_notes}": "Add technical approach details",
		"{success_metrics}": "Add success metrics",
		"{non_goals}":      "Add non-goals",
	}
	for placeholder, replacement := range replacements {
		content = strings.ReplaceAll(content, placeholder, replacement)
	}

	filename := name
	if !strings.HasSuffix(filename, ".md") {
		filename += ".md"
	}

	specPath := filepath.Join(projectPath, ".spektacular", "specs", filename)
	if _, err := os.Stat(specPath); err == nil {
		return "", fmt.Errorf("spec file already exists: %s", specPath)
	}

	if err := os.WriteFile(specPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("writing spec file: %w", err)
	}
	return specPath, nil
}

// toTitle converts a kebab-case or snake_case name to Title Case.
func toTitle(name string) string {
	base := strings.TrimSuffix(name, ".md")
	words := strings.FieldsFunc(base, func(r rune) bool { return r == '-' || r == '_' })
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
