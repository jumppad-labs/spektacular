package store

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// maxExcerptBytes caps a Hit excerpt so an agent can scan many hits cheaply.
const maxExcerptBytes = 256

// maxExcerptsPerHit caps how many excerpt lines a single Hit carries.
const maxExcerptsPerHit = 3

// scanBufferBytes bounds how long a single scanned line may be; a file
// containing a longer line is scanned up to that line and then skipped.
const scanBufferBytes = 1024 * 1024

// binarySniffBytes is how many leading bytes of a file are checked for a NUL
// byte to classify it as binary — the same convention git uses.
const binarySniffBytes = 8000

// Search returns hits for a free-form keyword query, scanning only this store.
// It walks the store root in process — no external tool is consulted. The
// query is split into whitespace-separated terms, each matched as a literal,
// case-insensitive substring; a file matches when every term occurs somewhere
// in it, and each matching file produces exactly one scope-tagged Hit whose
// Score is the sum of all terms' occurrence counts across the file.
// Directories named "conventions" are excluded, binary files are skipped, and
// a file containing an over-long line is scanned only up to that line. A
// query with no terms — empty or all whitespace — or one with no matches
// returns an empty result, not an error.
func (f *FileStore) Search(query string) ([]Hit, error) {
	terms := strings.Fields(strings.ToLower(query))
	if len(terms) == 0 {
		return nil, nil
	}
	return f.search(terms)
}

// search walks the store root, scans every file once, and emits one Hit per
// file that contains every term. Terms must already be lower-cased.
func (f *FileStore) search(terms []string) ([]Hit, error) {
	var hits []Hit

	err := filepath.WalkDir(f.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Conventions are read in full via the dedicated conventions
			// reader; exclude them from search so they are never surfaced
			// twice.
			if d.Name() == "conventions" {
				return filepath.SkipDir
			}
			return nil
		}
		agg, err := scanFile(path, terms)
		if err != nil {
			return err
		}
		score := 0
		for _, count := range agg.counts {
			if count == 0 {
				// A document matches only when every term occurs in it.
				return nil
			}
			score += count
		}
		rel, relErr := filepath.Rel(f.root, path)
		if relErr != nil {
			rel = path
		}
		title := agg.title
		if title == "" {
			title = rel
		}
		excerpts := make([]string, 0, len(agg.best))
		for _, c := range agg.best {
			excerpts = append(excerpts, trimExcerpt(c.text))
		}
		hits = append(hits, Hit{
			Scope:    f.scope,
			Path:     rel,
			Title:    title,
			Excerpts: excerpts,
			Score:    float64(score),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("native search: %w", err)
	}
	return hits, nil
}

// candidateLine is one matching line considered for a Hit's excerpts,
// together with the strength used to rank it: how many distinct terms occur
// on it and the total occurrences of all terms on it.
type candidateLine struct {
	text     string
	distinct int
	total    int
}

// fileAggregate describes one scanned file as a whole: how often each term
// occurred across the file, the text of its first ATX heading, and the
// strongest matching lines, bounded at maxExcerptsPerHit.
type fileAggregate struct {
	counts   []int           // total occurrences per term, indexed like the terms slice
	title    string          // first ATX-heading text, "" when the file has none
	titleSet bool            // whether a heading line was seen, so later ones never win
	best     []candidateLine // strongest lines: distinct terms desc, total desc, file order asc
}

// addCandidate ranks line into the aggregate's bounded excerpt candidates —
// most distinct terms first, then most total occurrences, then earliest in
// the file. Inserting after equal entries keeps file order for ties, and the
// bound keeps memory line-sized however many lines match.
func (a *fileAggregate) addCandidate(c candidateLine) {
	pos := len(a.best)
	for i, b := range a.best {
		if c.distinct > b.distinct || (c.distinct == b.distinct && c.total > b.total) {
			pos = i
			break
		}
	}
	if pos == maxExcerptsPerHit {
		return
	}
	a.best = append(a.best, candidateLine{})
	copy(a.best[pos+1:], a.best[pos:])
	a.best[pos] = c
	if len(a.best) > maxExcerptsPerHit {
		a.best = a.best[:maxExcerptsPerHit]
	}
}

// scanFile reads the file at path once and aggregates, per term, the number
// of non-overlapping case-insensitive occurrences across the whole file,
// along with the file's first ATX-heading text and its strongest matching
// lines. Terms must already be lower-cased. Binary files — a NUL byte within
// the leading binarySniffBytes — quietly yield an empty aggregate, and once a
// line exceeds scanBufferBytes the remainder of the file is skipped, keeping
// the aggregate collected so far.
func scanFile(path string, terms []string) (fileAggregate, error) {
	agg := fileAggregate{counts: make([]int, len(terms))}

	file, err := os.Open(path)
	if err != nil {
		return agg, err
	}
	defer file.Close()

	sniff := make([]byte, binarySniffBytes)
	n, err := io.ReadFull(file, sniff)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return agg, err
	}
	sniff = sniff[:n]
	if bytes.IndexByte(sniff, 0) >= 0 {
		return agg, nil
	}

	scanner := bufio.NewScanner(io.MultiReader(bytes.NewReader(sniff), file))
	scanner.Buffer(make([]byte, 0, 64*1024), scanBufferBytes)
	for scanner.Scan() {
		line := scanner.Text()
		if !agg.titleSet {
			if text, ok := headingText(line); ok {
				agg.title = text
				agg.titleSet = true
			}
		}
		lowered := strings.ToLower(line)
		cand := candidateLine{text: line}
		for i, term := range terms {
			if count := strings.Count(lowered, term); count > 0 {
				agg.counts[i] += count
				cand.distinct++
				cand.total += count
			}
		}
		if cand.total > 0 {
			agg.addCandidate(cand)
		}
	}
	if err := scanner.Err(); err != nil {
		// A scanner is unrecoverable after an over-long line, so skip the
		// rest of this file and keep the aggregate collected so far.
		if !errors.Is(err, bufio.ErrTooLong) {
			return agg, err
		}
	}
	return agg, nil
}

// headingText reports whether line is an ATX heading — its trimmed form is
// one or more '#'s standing alone or followed by whitespace — and returns
// the heading text with the markers stripped.
func headingText(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	rest := strings.TrimLeft(trimmed, "#")
	if rest == trimmed {
		return "", false
	}
	if rest != "" && rest[0] != ' ' && rest[0] != '\t' {
		// A '#' run flush against text (e.g. "#hashtag") is not a heading.
		return "", false
	}
	return strings.TrimSpace(rest), true
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
