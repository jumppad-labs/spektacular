// Package knowledge provides a multi-source knowledge layer over the store
// abstraction. A Set is an ordered collection of scoped stores; it fans Read,
// List, Search, and Write across every member and tags results by scope.
package knowledge

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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

// Convention is a single always-apply convention, tagged with the scope it
// lives in and carrying its full body. Unlike Entry it includes Content,
// because the convention reader returns every convention's text in one call.
type Convention struct {
	Scope   string `json:"scope"`
	Path    string `json:"path"`
	Content string `json:"content"`
}

// AlwaysAppliedEntry is a single entry from an always-applied category, tagged
// with the scope it lives in, its category, and carrying its full body. The
// Category field lets a consumer tell a convention from a glossary term when
// the always-applied reader returns more than one category at once.
type AlwaysAppliedEntry struct {
	Scope    string `json:"scope"`
	Path     string `json:"path"`
	Content  string `json:"content"`
	Category string `json:"category"`
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

// Search runs the query against every source and returns the merged,
// scope-tagged hits ranked globally: score descending, with ties broken by
// configured source order and then by path, so the returned slice is the
// display order and is deterministic across runs. If any source errors,
// Search returns an error naming that source and no results.
func (s *Set) Search(query string) ([]store.Hit, error) {
	type rankedHit struct {
		hit    store.Hit
		source int // index of the originating source, the first tie-break
	}
	var merged []rankedHit
	for i, src := range s.sources {
		h, err := src.store.Search(query)
		if err != nil {
			return nil, fmt.Errorf("searching knowledge source %q: %w", src.scope, err)
		}
		for _, hit := range h {
			merged = append(merged, rankedHit{hit: hit, source: i})
		}
	}
	sort.SliceStable(merged, func(a, b int) bool {
		ra, rb := merged[a], merged[b]
		if ra.hit.Score != rb.hit.Score {
			return ra.hit.Score > rb.hit.Score
		}
		if ra.source != rb.source {
			return ra.source < rb.source
		}
		return ra.hit.Path < rb.hit.Path
	})
	alwaysApplied := alwaysAppliedSet()
	var hits []store.Hit
	for _, r := range merged {
		hit := r.hit
		hit.Category = categoryOf(hit.Path)
		if alwaysApplied[hit.Category] {
			// Always-applied categories are loaded in full on every task via
			// AlwaysAppliedEntries; exclude them from search so the same
			// content is never surfaced twice. This is the single registry-driven
			// place the exclusion lives — the store no longer special-cases it.
			continue
		}
		hits = append(hits, hit)
	}
	return hits, nil
}

// categoryOf returns an entry's category — the first segment of its
// store-relative path. An entry sitting at the scope root, with no path
// separator, has no category and yields the empty string.
func categoryOf(path string) string {
	if category, _, found := strings.Cut(path, "/"); found {
		return category
	}
	return ""
}

// alwaysAppliedSet returns the always-applied category names as a set for
// membership tests, derived from the single registry declaration.
func alwaysAppliedSet() map[string]bool {
	set := make(map[string]bool)
	for _, name := range AlwaysApplied() {
		set[name] = true
	}
	return set
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

// AlwaysAppliedEntries reads the full body of every entry in every
// always-applied category (conventions, glossary, …) across every configured
// scope, concatenated in configured scope order then registry category order,
// each tagged with its scope, path, and category. These categories are loaded
// in full on every task rather than searched. A scope that lacks a category's
// directory contributes nothing rather than erroring, so fresh or
// partially-populated scopes still resolve cleanly.
func (s *Set) AlwaysAppliedEntries() ([]AlwaysAppliedEntry, error) {
	return s.readCategories(AlwaysApplied())
}

// Conventions reads every always-apply convention across every configured
// scope, concatenated in configured order, and returns each one's full body
// tagged with its scope and path. Conventions live under each scope's
// "conventions/" directory. It is the backward-compatible, conventions-only
// view over the generalised always-applied reader, so its JSON shape is
// unchanged. A scope that has no such directory contributes nothing rather than
// erroring.
func (s *Set) Conventions() ([]Convention, error) {
	entries, err := s.readCategories([]string{"conventions"})
	if err != nil {
		return nil, err
	}
	var conventions []Convention
	for _, e := range entries {
		conventions = append(conventions, Convention{Scope: e.Scope, Path: e.Path, Content: e.Content})
	}
	return conventions, nil
}

// readCategories reads the full body of every entry in each named category
// across every configured scope, in configured scope order then the given
// category order, tagging each entry with its category. A scope missing a
// category's directory contributes nothing rather than erroring.
func (s *Set) readCategories(categories []string) ([]AlwaysAppliedEntry, error) {
	var entries []AlwaysAppliedEntry
	for _, src := range s.sources {
		for _, category := range categories {
			files, err := listFiles(src.store, category)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					continue
				}
				return nil, fmt.Errorf("listing %s in knowledge source %q: %w", category, src.scope, err)
			}
			for _, f := range files {
				content, err := src.store.Read(f)
				if err != nil {
					return nil, fmt.Errorf("reading %s entry %q in knowledge source %q: %w", category, f, src.scope, err)
				}
				entries = append(entries, AlwaysAppliedEntry{Scope: src.scope, Path: f, Content: string(content), Category: category})
			}
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
