package agent

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/cbroglie/mustache"
	"github.com/jumppad-labs/spektacular/internal/config"
)

const (
	memoryContextTemplatePath = "agents/memory-context.md"
	memoryContextHeading      = "## Memory & Context"
)

// installMemoryContextSection writes (or updates in place) the managed
// "Memory & Context" section in <projectPath>/AGENTS.md, rendering the
// embedded template against cfg.Command. Idempotent: re-running for the
// same projectPath leaves a single section and does not duplicate.
func installMemoryContextSection(projectPath string, cfg config.Config, out io.Writer) error {
	tmplBytes, err := fs.ReadFile(sourceFS, memoryContextTemplatePath)
	if err != nil {
		return fmt.Errorf("reading embedded template %s: %w", memoryContextTemplatePath, err)
	}

	rendered, err := mustache.Render(string(tmplBytes), map[string]string{"command": cfg.Command})
	if err != nil {
		return fmt.Errorf("rendering template %s: %w", memoryContextTemplatePath, err)
	}
	if !strings.HasSuffix(rendered, "\n") {
		rendered += "\n"
	}

	agentsPath := filepath.Join(projectPath, "AGENTS.md")
	existing, err := os.ReadFile(agentsPath)
	switch {
	case os.IsNotExist(err):
		return writeAGENTSAtomic(agentsPath, []byte(rendered), out, "created")
	case err != nil:
		return fmt.Errorf("reading %s: %w", agentsPath, err)
	}

	startIdx, endIdx, found := locateMemoryContextSection(existing)
	if !found {
		body := appendMemoryContextSection(existing, rendered)
		return writeAGENTSAtomic(agentsPath, body, out, "appended Memory & Context section to")
	}

	body := replaceMemoryContextSection(existing, startIdx, endIdx, rendered)
	return writeAGENTSAtomic(agentsPath, body, out, "updated Memory & Context section in")
}

// locateMemoryContextSection returns the byte range [startIdx, endIdx)
// covering the managed section in body. The range starts at the first byte
// of the "## Memory & Context" heading line and ends at the first byte of
// the next `# ` or `## ` heading line (or len(body) if there is none).
// Returns found=false if the heading is absent.
func locateMemoryContextSection(body []byte) (startIdx, endIdx int, found bool) {
	lines := strings.SplitAfter(string(body), "\n")
	offset := 0
	start := -1
	for _, line := range lines {
		trimmed := strings.TrimRight(line, "\n\r \t")
		switch {
		case start == -1 && trimmed == memoryContextHeading:
			start = offset
		case start != -1 && (strings.HasPrefix(trimmed, "# ") || strings.HasPrefix(trimmed, "## ")):
			return start, offset, true
		}
		offset += len(line)
	}
	if start == -1 {
		return 0, 0, false
	}
	return start, len(body), true
}

// appendMemoryContextSection appends the rendered section to body, ensuring
// exactly one blank line separates any prior content from the section and
// that no trailing-whitespace accumulation occurs across repeated runs.
func appendMemoryContextSection(body []byte, rendered string) []byte {
	trimmed := bytes.TrimRight(body, "\n\r \t")
	if len(trimmed) == 0 {
		return []byte(rendered)
	}
	var buf bytes.Buffer
	buf.Write(trimmed)
	buf.WriteString("\n\n")
	buf.WriteString(rendered)
	return buf.Bytes()
}

// replaceMemoryContextSection replaces body[startIdx:endIdx] with rendered,
// preserving exactly one blank line of separation between the section and
// any non-blank content immediately before or after it.
func replaceMemoryContextSection(body []byte, startIdx, endIdx int, rendered string) []byte {
	prefixTrimmed := bytes.TrimRight(body[:startIdx], "\n\r \t")
	suffixTrimmed := bytes.TrimLeft(body[endIdx:], "\n\r \t")

	var buf bytes.Buffer
	if len(prefixTrimmed) > 0 {
		buf.Write(prefixTrimmed)
		buf.WriteString("\n\n")
	}
	buf.WriteString(rendered)
	if len(suffixTrimmed) > 0 {
		buf.WriteByte('\n')
		buf.Write(suffixTrimmed)
	}
	return buf.Bytes()
}

// writeAGENTSAtomic writes content to path atomically. Emits one line to out
// describing the action, matching the format used by installWorkflowSkills.
func writeAGENTSAtomic(path string, content []byte, out io.Writer, action string) error {
	if err := writeFileAtomic(path, content); err != nil {
		return err
	}
	if action == "created" {
		fmt.Fprintf(out, "  AGENTS.md: created %s\n", path)
	} else {
		fmt.Fprintf(out, "  AGENTS.md: %s %s\n", action, path)
	}
	return nil
}

// writeFileAtomic writes content to path via a sibling temp file + rename so a
// mid-write failure cannot truncate an existing file.
func writeFileAtomic(path string, content []byte) error {
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, content, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("renaming %s to %s: %w", tmpPath, path, err)
	}
	return nil
}
