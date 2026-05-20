package store

import (
	"errors"
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

// Hit is a generic search result produced by a store's Search. It carries a
// locator and a compact excerpt, never the full file body.
type Hit struct {
	Scope   string  `json:"scope"`   // scope label of the originating store
	Path    string  `json:"path"`    // locator, relative to the store root — pass to Read
	Excerpt string  `json:"excerpt"` // compact excerpt, capped at the excerpt budget
	Score   float64 `json:"score"`   // optional cheap relevance score; 0 when the backend has none
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
	// store. Hits carry the store's own scope so callers can attribute them.
	Search(query string) ([]Hit, error)
}

// FileStore implements Store over the local filesystem.
// All paths are resolved relative to root and must not escape it.
type FileStore struct {
	root  string
	scope string
	// forceFallback makes Search skip the ripgrep path and always use the
	// native Go scan. Set only by tests so the fallback can be exercised
	// deterministically regardless of whether rg is installed on the host.
	forceFallback bool
}

// NewFileStore creates a FileStore rooted at root, labelled with scope.
// The scope tags any hits the store produces so callers can attribute them.
func NewFileStore(root, scope string) *FileStore {
	return &FileStore{root: filepath.Clean(root), scope: scope}
}

// Root returns the absolute path to the store root directory.
func (f *FileStore) Root() string {
	return f.root
}

// Scope returns the scope label the store was constructed with.
func (f *FileStore) Scope() string {
	return f.scope
}

// abs resolves a relative path against the root, rejecting path traversal.
func (f *FileStore) abs(path string) (string, error) {
	joined := filepath.Join(f.root, path)
	rel, err := filepath.Rel(f.root, joined)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", errors.New("path escapes store root")
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
