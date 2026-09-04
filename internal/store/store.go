package store

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ErrNotFound is returned when the requested file or directory does not exist.
var ErrNotFound = errors.New("not found")

// DirEntry is a typed direct child of a directory returned by List. IsDir lets
// a caller tell a file from a subdirectory and recurse into the tree.
type DirEntry struct {
	Name  string // child name, not a full path
	IsDir bool   // true for a subdirectory — recurse into it via List
}

// Hit is a generic search result produced by a store's Search. It describes
// one matching document, carrying a locator and compact excerpts, never the
// full file body.
type Hit struct {
	Tier     string   `json:"tier"`     // addressing tier of the originating store; left empty by the store and populated by the knowledge layer
	Name     string   `json:"name"`     // name of the originating store; left empty by the store and populated by the knowledge layer
	Path     string   `json:"path"`     // locator, relative to the store root — pass to Read
	Title    string   `json:"title"`    // the document's first heading, or the locator when it has none
	Excerpts []string `json:"excerpts"` // compact excerpts, each capped at the excerpt budget
	Score    float64  `json:"score"`    // sum of all query terms' case-insensitive occurrences across the document
	Category string   `json:"category"` // the entry's category; left empty by the store and populated by the knowledge layer from the path
	Checksum string   `json:"checksum"` // hex SHA-256 over the entry's exact raw bytes; the identity key for byte-identical de-dup
}

// Store provides read/write access to a project's data directory.
// All paths are relative to the store root.
type Store interface {
	// Root returns the absolute path to the store root directory.
	Root() string
	// Read returns the contents of the file at path.
	Read(path string) ([]byte, error)
	// Write creates or overwrites the file at path with content.
	// Parent directories are created automatically.
	Write(path string, content []byte) error
	// Delete removes the file at path. Returns nil if the file does not exist.
	Delete(path string) error
	// List returns the direct children of the directory at path. Each entry
	// reports whether it is a directory, so a caller can recurse the tree.
	List(path string) ([]DirEntry, error)
	// Exists reports whether a file or directory exists at path.
	Exists(path string) bool
	// Search returns hits for a free-form keyword query, scanning only this
	// store. Hits are left unattributed: the store has no notion of what its
	// caller's addressing scheme is, so filling that in is the caller's job.
	Search(query string) ([]Hit, error)
}

// FileStore implements Store over the local filesystem.
// All paths are resolved relative to root and must not escape it.
type FileStore struct {
	root  string
	label string
}

// NewFileStore creates a FileStore rooted at root, carrying label. The label is
// an opaque diagnostic tag: the store never interprets it and never puts it on a
// hit. Attributing a hit to where it came from is the caller's job, since the
// generic store layer has no notion of what a knowledge tier or a changelog repo
// is.
func NewFileStore(root, label string) *FileStore {
	return &FileStore{root: filepath.Clean(root), label: label}
}

// Root returns the absolute path to the store root directory.
func (f *FileStore) Root() string {
	return f.root
}

// Label returns the diagnostic label the store was constructed with.
func (f *FileStore) Label() string {
	return f.label
}

// abs resolves a relative path against the root, rejecting path traversal.
func (f *FileStore) abs(path string) (string, error) {
	joined := filepath.Join(f.root, path)
	rel, err := filepath.Rel(f.root, joined)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path %q escapes store root", path)
	}
	return joined, nil
}

func (f *FileStore) Read(path string) ([]byte, error) {
	abs, err := f.abs(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(abs)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNotFound
	}
	return data, err
}

func (f *FileStore) Write(path string, content []byte) error {
	abs, err := f.abs(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		return err
	}
	return os.WriteFile(abs, content, 0644)
}

func (f *FileStore) Delete(path string) error {
	abs, err := f.abs(path)
	if err != nil {
		return err
	}
	err = os.Remove(abs)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

func (f *FileStore) List(path string) ([]DirEntry, error) {
	abs, err := f.abs(path)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(abs)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	result := make([]DirEntry, len(entries))
	for i, e := range entries {
		result[i] = DirEntry{Name: e.Name(), IsDir: e.IsDir()}
	}
	return result, nil
}

func (f *FileStore) Exists(path string) bool {
	abs, err := f.abs(path)
	if err != nil {
		return false
	}
	_, err = os.Stat(abs)
	return err == nil
}
