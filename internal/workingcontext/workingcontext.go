// Package workingcontext manages the CLI-owned section of the agent's working
// context file, .spektacular/context.md.
//
// The file as a whole belongs to the agent: it records the cross-cutting
// learnings and user answers a session leaves behind for a resumed session to
// read back. The one exception is the "## Repos" block at the top, which the
// CLI renders from the repo registry on every workflow command so an agent —
// starting fresh or resuming mid-workflow — always finds each registered repo
// and where its code lives in the file it already reads first, without any
// step having to repeat the roster.
package workingcontext

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jumppad-labs/spektacular/internal/stepkit"
)

const (
	// RelPath is the working-context file's location relative to the project
	// root.
	RelPath = ".spektacular/context.md"

	// StartMarker and EndMarker delimit the CLI-managed repos block.
	StartMarker = "<!-- spektacular:repos:start -->"
	EndMarker   = "<!-- spektacular:repos:end -->"

	templatePath = "context/repos.md"
)

// Path returns the working-context file's absolute path under root.
func Path(root string) string {
	return filepath.Join(root, filepath.FromSlash(RelPath))
}

// Render returns the managed repos block, markers included, for the given
// command name and roster. roster is the workflow's "repos" data: a slice of
// per-repo maps carrying name, description, role, tags, deployment, and
// source. A nil or empty roster renders the no-repos fallback.
func Render(command string, roster any) (string, error) {
	out, err := stepkit.RenderTemplate(templatePath, map[string]any{
		"config": map[string]any{"command": command},
		"repos":  roster,
	})
	if err != nil {
		return "", err
	}
	return strings.TrimRight(out, "\n") + "\n", nil
}

// WriteRoster upserts the managed repos block at the top of the file at path,
// preserving everything the agent wrote outside the markers. A missing file
// is created holding only the block.
func WriteRoster(path, command string, roster any) error {
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	return write(path, command, roster, Strip(string(existing)))
}

// Reset replaces the file at path with only the managed repos block,
// discarding any agent-owned content. It is the "clear context.md" operation
// a fresh workflow performs.
func Reset(path, command string, roster any) error {
	return write(path, command, roster, "")
}

// Strip returns body with the managed repos block removed. Content outside
// the markers is returned unchanged apart from the leading blank lines that
// separated it from the block.
func Strip(body string) string {
	start := strings.Index(body, StartMarker)
	if start < 0 {
		return body
	}
	end := strings.Index(body[start:], EndMarker)
	if end < 0 {
		return body
	}
	rest := body[start+end+len(EndMarker):]
	return body[:start] + strings.TrimLeft(rest, "\n")
}

func write(path, command string, roster any, agentBody string) error {
	block, err := Render(command, roster)
	if err != nil {
		return err
	}
	content := block
	if agentBody != "" {
		content += "\n" + agentBody
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
