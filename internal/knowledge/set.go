// Package knowledge provides a multi-source knowledge layer over the store
// abstraction. A Set is an ordered collection of scoped stores; it fans Read,
// List, Search, and Write across every member and tags results by scope.
package knowledge

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/store"
)

// scopedStore pairs a configured store with the metadata of the source that
// produced it.
type scopedStore struct {
	scope    string
	provider string
	location string
	store    store.Store
}

// Set is an ordered collection of scoped knowledge stores.
type Set struct {
	sources []scopedStore
}

// Entry is a single knowledge entry, tagged with the scope it lives in.
type Entry struct {
	Scope string `json:"scope"`
	Path  string `json:"path"`
}

// SourceInfo describes one configured knowledge source.
type SourceInfo struct {
	Scope    string `json:"scope"`
	Provider string `json:"provider"`
	Location string `json:"location"`
}

// NewSet resolves the configured knowledge sources into live stores. The
// default project source is synthesised when none are configured. Relative
// source locations resolve against projectRoot. NewSet fails fast: if any
// source names an unknown provider or points at an unreachable location it
// returns an error naming that source and no Set.
func NewSet(cfg config.Config, projectRoot string) (*Set, error) {
	kc := cfg.Knowledge.WithDefaults(projectRoot)
	set := &Set{}
	for _, src := range kc.Sources {
		switch src.Provider {
		case config.ProviderFile:
			location := src.Config.Location
			if !filepath.IsAbs(location) {
				location = filepath.Join(projectRoot, location)
			}
			info, err := os.Stat(location)
			if err != nil || !info.IsDir() {
				return nil, fmt.Errorf("knowledge source %q is unreachable at %s", src.Scope, location)
			}
			set.sources = append(set.sources, scopedStore{
				scope:    src.Scope,
				provider: src.Provider,
				location: location,
				store:    store.NewFileStore(location, src.Scope),
			})
		default:
			return nil, fmt.Errorf("knowledge source %q: provider %q is not supported", src.Scope, src.Provider)
		}
	}
	return set, nil
}

// Search runs the query against every source in configured order and
// concatenates the scope-tagged hits. It performs no ranking or dedup. If any
// source errors, Search returns an error naming that source and no results.
func (s *Set) Search(query string) ([]store.Hit, error) {
	var hits []store.Hit
	for _, src := range s.sources {
		h, err := src.store.Search(query)
		if err != nil {
			return nil, fmt.Errorf("searching knowledge source %q: %w", src.scope, err)
		}
		hits = append(hits, h...)
	}
	return hits, nil
}

// Read returns the full content of a knowledge entry from a named scope.
func (s *Set) Read(scope, path string) ([]byte, error) {
	src, err := s.byScope(scope)
	if err != nil {
		return nil, err
	}
	return src.store.Read(path)
}

// Write persists a knowledge entry into a named scope, leaving every other
// scope untouched.
func (s *Set) Write(scope, path string, content []byte) error {
	src, err := s.byScope(scope)
	if err != nil {
		return err
	}
	return src.store.Write(path, content)
}

// List recursively enumerates every file entry across every configured scope,
// concatenated in configured order. Subdirectories are descended into; only
// file locators are emitted.
func (s *Set) List() ([]Entry, error) {
	var entries []Entry
	for _, src := range s.sources {
		files, err := listFiles(src.store, "")
		if err != nil {
			return nil, fmt.Errorf("listing knowledge source %q: %w", src.scope, err)
		}
		for _, f := range files {
			entries = append(entries, Entry{Scope: src.scope, Path: f})
		}
	}
	return entries, nil
}

// Sources reports the configured scopes and their locations, in order.
func (s *Set) Sources() []SourceInfo {
	infos := make([]SourceInfo, len(s.sources))
	for i, src := range s.sources {
		infos[i] = SourceInfo{Scope: src.scope, Provider: src.provider, Location: src.location}
	}
	return infos
}

// byScope finds the scoped store for a scope, erroring when none matches.
func (s *Set) byScope(scope string) (scopedStore, error) {
	for _, src := range s.sources {
		if src.scope == scope {
			return src, nil
		}
	}
	return scopedStore{}, fmt.Errorf("no knowledge source configured for scope %q", scope)
}

// listFiles recursively walks a store from dir, returning store-relative file
// locators. Directories are descended into via Store.List, which stays one
// level deep — the recursion lives here in the knowledge layer.
func listFiles(st store.Store, dir string) ([]string, error) {
	children, err := st.List(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, child := range children {
		childPath := child.Name
		if dir != "" {
			childPath = dir + "/" + child.Name
		}
		if child.IsDir {
			sub, err := listFiles(st, childPath)
			if err != nil {
				return nil, err
			}
			files = append(files, sub...)
			continue
		}
		files = append(files, childPath)
	}
	return files, nil
}
