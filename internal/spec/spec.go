// Package spec handles creation of new specification files from the embedded template.
package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jumppad-labs/spektacular/internal/defaults"
)

// NextSpecNumber scans specsDir and returns one greater than the highest numeric prefix
// found in existing spec filenames (e.g. "12_foo.md" → next is 13). Returns 1 if empty.
func NextSpecNumber(specsDir string) int {
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		return 1
	}
	max := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if n, ok := numericPrefix(e.Name()); ok && n > max {
			max = n
		}
	}
	return max + 1
}

// AutoNumberName prepends the next spec number to name if name does not already have a
// numeric prefix (e.g. "my-feature" → "13_my-feature"). If name already starts with a
// number prefix it is returned unchanged.
func AutoNumberName(name, specsDir string) string {
	if _, ok := numericPrefix(name); ok {
		return name
	}
	return fmt.Sprintf("%d_%s", NextSpecNumber(specsDir), name)
}

// numericPrefix returns the leading integer and true for filenames like "12_foo.md" or
// names like "12_foo". Returns 0, false if no numeric prefix is present.
func numericPrefix(name string) (int, bool) {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	idx := strings.IndexByte(base, '_')
	if idx <= 0 {
		return 0, false
	}
	n, err := strconv.Atoi(base[:idx])
	if err != nil {
		return 0, false
	}
	return n, true
}

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

	specsDir := filepath.Join(projectPath, ".spektacular", "specs")
	filename := AutoNumberName(name, specsDir)
	if !strings.HasSuffix(filename, ".md") {
		filename += ".md"
	}

	specPath := filepath.Join(specsDir, filename)
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

// InitTemplate creates the spec file at specPath with the standard template structure,
// deriving the title from the file name. It is a no-op if the file already exists.
func InitTemplate(specPath string) error {
	if _, err := os.Stat(specPath); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(specPath), 0755); err != nil {
		return fmt.Errorf("creating specs directory: %w", err)
	}
	name := strings.TrimSuffix(filepath.Base(specPath), filepath.Ext(specPath))
	title := toTitle(name)

	templateBytes, err := defaults.ReadFile("spec-template.md")
	if err != nil {
		return fmt.Errorf("reading spec template: %w", err)
	}
	content := strings.ReplaceAll(string(templateBytes), "{title}", title)
	content = strings.ReplaceAll(content, "{description}", "")
	content = strings.ReplaceAll(content, "- [ ] **{Requirement title}**\n  {Describe what must be true and any relevant detail.}", "")
	content = strings.ReplaceAll(content, "- [ ] **{Criteria title}**\n  {Describe exactly what can be observed or tested to confirm this passes.}", "")

	if err := os.WriteFile(specPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing spec file: %w", err)
	}
	return nil
}

// ResolveSpecFile resolves the spec file path from the given argument.
// It checks: (1) direct path, (2) relative to cwd, (3) spec name in .spektacular/specs/,
// (4) spec name without .md extension in .spektacular/specs/.
func ResolveSpecFile(arg, cwd string) (string, error) {
	candidates := []string{
		arg,
		filepath.Join(cwd, arg),
		filepath.Join(cwd, ".spektacular", "specs", arg),
		filepath.Join(cwd, ".spektacular", "specs", arg+".md"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("spec file not found: tried %s, %s, and .spektacular/specs/%s",
		arg, filepath.Join(cwd, arg), arg)
}

// LoadAgentSystemPrompt returns the shared minimal system prompt for the spec creator agent.
// It defines the agent's role, question format rules, and completion signal only —
// the specific task instructions for each section live in the user prompts.
func LoadAgentSystemPrompt() string {
	return string(defaults.MustReadFile("agents/spec.md"))
}

