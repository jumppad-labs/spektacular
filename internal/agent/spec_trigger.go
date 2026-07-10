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
	specTriggerTemplatePath = "agents/spec-trigger.md"
	specTriggerHeading      = "## Spec-Worthy Discussion Recognition"
)

// installSpecTriggerSection writes (or updates in place) the managed
// "Spec-Worthy Discussion Recognition" section in <projectPath>/AGENTS.md,
// rendering the embedded template against cfg.Command. Idempotent:
// re-running for the same projectPath leaves a single section and does not
// duplicate.
func installSpecTriggerSection(projectPath string, cfg config.Config, out io.Writer) error {
	tmplBytes, err := fs.ReadFile(sourceFS, specTriggerTemplatePath)
	if err != nil {
		return fmt.Errorf("reading embedded template %s: %w", specTriggerTemplatePath, err)
	}

	rendered, err := mustache.Render(string(tmplBytes), map[string]string{"command": cfg.Command})
	if err != nil {
		return fmt.Errorf("rendering template %s: %w", specTriggerTemplatePath, err)
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

	startIdx, endIdx, found := locateSpecTriggerSection(existing)
	if !found {
		body := appendSpecTriggerSection(existing, rendered)
		return writeAGENTSAtomic(agentsPath, body, out, "appended Spec-Worthy Discussion Recognition section to")
	}

	body := replaceSpecTriggerSection(existing, startIdx, endIdx, rendered)
	return writeAGENTSAtomic(agentsPath, body, out, "updated Spec-Worthy Discussion Recognition section in")
}

// locateSpecTriggerSection returns the byte range [startIdx, endIdx) covering
// the managed section in body. The range starts at the first byte of the
// "## Spec-Worthy Discussion Recognition" heading line and ends at the first
// byte of the next `# ` or `## ` heading line (or len(body) if there is
// none). Returns found=false if the heading is absent.
func locateSpecTriggerSection(body []byte) (startIdx, endIdx int, found bool) {
	lines := strings.SplitAfter(string(body), "\n")
	offset := 0
	start := -1
	for _, line := range lines {
		trimmed := strings.TrimRight(line, "\n\r \t")
		switch {
		case start == -1 && trimmed == specTriggerHeading:
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

// appendSpecTriggerSection appends the rendered section to body, ensuring
// exactly one blank line separates any prior content from the section and
// that no trailing-whitespace accumulation occurs across repeated runs.
func appendSpecTriggerSection(body []byte, rendered string) []byte {
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

// replaceSpecTriggerSection replaces body[startIdx:endIdx] with rendered,
// preserving exactly one blank line of separation between the section and
// any non-blank content immediately before or after it.
func replaceSpecTriggerSection(body []byte, startIdx, endIdx int, rendered string) []byte {
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
