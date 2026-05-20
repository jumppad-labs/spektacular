package store

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// maxExcerptBytes caps a Hit excerpt so an agent can scan many hits cheaply.
const maxExcerptBytes = 256

// scanBufferBytes bounds how long a single matched line may be; lines longer
// than this are skipped rather than aborting the whole scan.
const scanBufferBytes = 1024 * 1024

// Search returns hits for a free-form keyword query, scanning only this store.
// It prefers the ripgrep binary when one is on PATH — shelling out and decoding
// rg's JSON event stream — and otherwise falls back to a native Go directory
// walk. Both paths perform a literal, case-insensitive substring match and
// produce equivalent scope-tagged hits, so no caller can observe which ran.
// An empty query, or a query with no matches, returns an empty result, not an
// error.
func (f *FileStore) Search(query string) ([]Hit, error) {
	if query == "" {
		return nil, nil
	}
	if !f.forceFallback {
		if rgPath, err := exec.LookPath("rg"); err == nil {
			return f.searchRipgrep(rgPath, query)
		}
	}
	return f.searchNative(query)
}

// rgEvent is the subset of a ripgrep --json event this package decodes. Only
// "match" events carry the fields below; other event types are ignored.
type rgEvent struct {
	Type string `json:"type"`
	Data struct {
		Path struct {
			Text string `json:"text"`
		} `json:"path"`
		Lines struct {
			Text string `json:"text"`
		} `json:"lines"`
		Submatches []struct {
			Match struct {
				Text string `json:"text"`
			} `json:"match"`
		} `json:"submatches"`
	} `json:"data"`
}

// searchRipgrep runs `rg` over the store root and decodes its JSON event
// stream into hits. --fixed-strings and --ignore-case make rg's matching
// literal and case-insensitive, matching the native fallback exactly.
func (f *FileStore) searchRipgrep(rgPath, query string) ([]Hit, error) {
	cmd := exec.Command(rgPath, "--json", "--no-heading", "--fixed-strings", "--ignore-case", query, f.root)
	out, err := cmd.Output()
	if err != nil {
		// rg exits 1 when there are simply no matches — not an error.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return nil, nil
		}
		return nil, fmt.Errorf("ripgrep search: %w", err)
	}

	var hits []Hit
	scanner := bufio.NewScanner(bytes.NewReader(out))
	scanner.Buffer(make([]byte, 0, 64*1024), scanBufferBytes)
	for scanner.Scan() {
		var ev rgEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			// Defensive: skip any line that is not a well-formed event.
			continue
		}
		if ev.Type != "match" {
			continue
		}
		rel, err := filepath.Rel(f.root, ev.Data.Path.Text)
		if err != nil {
			rel = ev.Data.Path.Text
		}
		hits = append(hits, Hit{
			Scope:   f.scope,
			Path:    rel,
			Excerpt: trimExcerpt(ev.Data.Lines.Text),
			Score:   float64(len(ev.Data.Submatches)),
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("ripgrep search: %w", err)
	}
	return hits, nil
}

// searchNative walks the store root and scans every file line by line for a
// case-insensitive substring match. It is the fallback when rg is unavailable.
func (f *FileStore) searchNative(query string) ([]Hit, error) {
	needle := strings.ToLower(query)
	var hits []Hit

	err := filepath.WalkDir(f.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		fileHits, err := scanFile(path, needle)
		if err != nil {
			return err
		}
		for _, line := range fileHits {
			rel, relErr := filepath.Rel(f.root, path)
			if relErr != nil {
				rel = path
			}
			hits = append(hits, Hit{
				Scope:   f.scope,
				Path:    rel,
				Excerpt: trimExcerpt(line),
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("native search: %w", err)
	}
	return hits, nil
}

// scanFile returns every line of the file at path that contains needle, which
// must already be lower-cased.
func scanFile(path, needle string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var matches []string
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), scanBufferBytes)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(strings.ToLower(line), needle) {
			matches = append(matches, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return matches, nil
}

// trimExcerpt collapses runs of whitespace in s and caps the result at
// maxExcerptBytes, trimming on a rune boundary so the excerpt stays valid
// UTF-8. It is the single place the excerpt budget is enforced, shared by
// both search paths.
func trimExcerpt(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= maxExcerptBytes {
		return s
	}
	cut := s[:maxExcerptBytes]
	for len(cut) > 0 && !utf8.ValidString(cut) {
		cut = cut[:len(cut)-1]
	}
	return cut
}
