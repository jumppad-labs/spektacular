// Package knowledge provides a multi-source knowledge layer over the store
// abstraction. A Set is an ordered collection of addressed stores; it fans
// List, Search, and the always-applied reader across its members and tags every
// result with the tier and store name it came from, while Read and Write
// resolve to exactly one store or refuse.
package knowledge

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/output"
	"github.com/jumppad-labs/spektacular/internal/store"
)

// scopedStore pairs a configured store with the address it answers to and the
// metadata of the source that produced it. The tier and name together are the
// store's identity: they are what a request addresses and what every result
// reports.
type scopedStore struct {
	tier     Tier
	name     string
	provider string
	location string
	store    store.Store
}

// Set is an ordered collection of addressed knowledge stores.
type Set struct {
	sources []scopedStore
}

// Entry is a single knowledge entry, tagged with the store it lives in. The
// triple {tier, name, path} is exactly what a read requires, so any entry can be
// fetched back from what it carries.
type Entry struct {
	Tier Tier   `json:"tier"`
	Name string `json:"name"`
	Path string `json:"path"`
}

// Convention is a single always-apply convention, tagged with the store it
// lives in and carrying its full body. Unlike Entry it includes Content,
// because the convention reader returns every convention's text in one call.
type Convention struct {
	Tier    Tier   `json:"tier"`
	Name    string `json:"name"`
	Path    string `json:"path"`
	Content string `json:"content"`
}

// AlwaysAppliedEntry is a single entry from an always-applied category, tagged
// with the store it lives in, its category, and carrying its full body. The
// Category field lets a consumer tell a convention from a glossary term when
// the always-applied reader returns more than one category at once.
type AlwaysAppliedEntry struct {
	Tier     Tier   `json:"tier"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Content  string `json:"content"`
	Category string `json:"category"`
}

// SourceInfo describes one configured knowledge store by the address it
// answers to. The tier and name together are the store's identity, so there is
// no separate repo attribution: a repo-tier store's name is its repo's registry
// name.
type SourceInfo struct {
	Tier     Tier   `json:"tier"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Location string `json:"location"`
}

// NewSet resolves the configured knowledge sources into live, addressed
// stores. A project declaring no shared stores is valid and simply yields an
// empty project tier; nothing is synthesised, since a repo's own store is the
// repo's to declare. Relative source locations resolve against projectRoot.
// NewSet fails fast: if any source names an unknown provider or points at an
// unreachable location it returns an error naming that store and no Set.
func NewSet(cfg config.Config, projectRoot string) (*Set, error) {
	set := &Set{}
	for _, src := range cfg.Knowledge.Sources {
		tier, name := addressOf(src)
		switch src.Provider {
		case config.ProviderFile:
			location := src.Config.Location
			if !filepath.IsAbs(location) {
				location = filepath.Join(projectRoot, location)
			}
			info, err := os.Stat(location)
			if err != nil || !info.IsDir() {
				return nil, fmt.Errorf("knowledge store %q in the %q tier is unreachable at %s", name, tier, location)
			}
			set.sources = append(set.sources, scopedStore{
				tier:     tier,
				name:     name,
				provider: src.Provider,
				location: location,
				store:    store.NewSourceStore(location, string(tier)+":"+name),
			})
		default:
			return nil, fmt.Errorf("knowledge store %q in the %q tier: provider %q is not supported", name, tier, src.Provider)
		}
	}
	return set, nil
}

// addressOf returns the address a source was stamped with during aggregation.
// A source that reaches NewSet unstamped — the knowledge package's own tests
// and any direct caller that builds a config by hand — falls back to the
// project tier, which is the tier a source declared in the project's own config
// belongs to anyway, so a single-tier project needs no aggregator.
func addressOf(src config.SourceConfig) (Tier, string) {
	tier := Tier(src.Tier)
	if tier == "" {
		tier = TierProject
	}
	return tier, src.Name
}

// Search runs the query against every store the selector covers and returns the
// merged hits ranked globally: score descending, with ties broken by configured
// store order and then by path, so the returned slice is the display order and
// is deterministic across runs. A store the selector does not cover is never
// queried at all. If any store errors, Search returns an error naming it and no
// results.
func (s *Set) Search(query string, sel Selector) ([]store.Hit, error) {
	if err := s.validateSelector(sel); err != nil {
		return nil, err
	}
	type rankedHit struct {
		hit    store.Hit
		source int // index of the originating store, the first tie-break
	}
	var merged []rankedHit
	for i, src := range s.sources {
		if !sel.covers(src) {
			continue
		}
		h, err := src.store.Search(query)
		if err != nil {
			return nil, fmt.Errorf("searching knowledge store %q: %w", src.name, err)
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
		hit.Tier = string(s.sources[r.source].tier)
		hit.Name = s.sources[r.source].name
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

// Read returns the full content of a knowledge entry from the one store the
// address names. An address that does not name exactly one store is refused
// rather than resolved to whichever store happens to look closest.
func (s *Set) Read(addr Address, path string) ([]byte, error) {
	src, err := s.resolve(addr)
	if err != nil {
		return nil, err
	}
	content, err := src.store.Read(path)
	if errors.Is(err, store.ErrNotFound) {
		return nil, output.NewError(
			ErrCodeEntryNotFound,
			fmt.Sprintf("no knowledge entry at %q in the %q store", path, addr.Name),
		).WithResource(path).WithNextAction(fmt.Sprintf(
			`list that store's entries with "knowledge list --tier %s --filter %s", or search across stores with "knowledge search <query>"`,
			addr.Tier, addr.Name,
		))
	}
	return content, err
}

// Write persists a knowledge entry into the one store the address names,
// leaving every other store untouched. An address that does not name exactly
// one store is refused and nothing is written anywhere.
func (s *Set) Write(addr Address, path string, content []byte) error {
	src, err := s.resolve(addr)
	if err != nil {
		return err
	}
	return src.store.Write(path, content)
}

// List recursively enumerates every file entry across every store the selector
// covers, concatenated in configured order. Subdirectories are descended into;
// only file locators are emitted.
func (s *Set) List(sel Selector) ([]Entry, error) {
	if err := s.validateSelector(sel); err != nil {
		return nil, err
	}
	var entries []Entry
	for _, src := range s.sources {
		if !sel.covers(src) {
			continue
		}
		files, err := listFiles(src.store, "")
		if err != nil {
			return nil, fmt.Errorf("listing knowledge store %q: %w", src.name, err)
		}
		for _, f := range files {
			entries = append(entries, Entry{Tier: src.tier, Name: src.name, Path: f})
		}
	}
	return entries, nil
}

// AlwaysAppliedEntries reads the full body of every entry in every
// always-applied category (conventions, glossary, …) across every store the
// selector covers, concatenated in configured store order then registry
// category order, each tagged with its store, path, and category. These
// categories are loaded in full on every task rather than searched. A store that
// lacks a category's directory contributes nothing rather than erroring, so
// fresh or partially-populated stores still resolve cleanly.
//
// The selector is honoured with no exceptions: a store the selector does not
// name contributes nothing, whichever tier it sits in.
func (s *Set) AlwaysAppliedEntries(sel Selector) ([]AlwaysAppliedEntry, error) {
	return s.readCategories(AlwaysApplied(), sel)
}

// Conventions reads every always-apply convention across every store the
// selector covers, concatenated in configured order, and returns each one's full
// body tagged with its store and path. Conventions live under each store's
// "conventions/" directory. It is the conventions-only view over the generalised
// always-applied reader, so its JSON shape is unchanged. A store that has no such
// directory contributes nothing rather than erroring.
func (s *Set) Conventions(sel Selector) ([]Convention, error) {
	entries, err := s.readCategories([]string{"conventions"}, sel)
	if err != nil {
		return nil, err
	}
	var conventions []Convention
	for _, e := range entries {
		conventions = append(conventions, Convention{Tier: e.Tier, Name: e.Name, Path: e.Path, Content: e.Content})
	}
	return conventions, nil
}

// readCategories reads the full body of every entry in each named category
// across every store the selector covers, in configured store order then the
// given category order, tagging each entry with its category. A store missing a
// category's directory contributes nothing rather than erroring.
//
// Membership is decided by the selector alone, applied to every store without
// special cases. There is deliberately no exemption for a store the reader
// cannot attribute to a repo: a store the selector does not name contributes
// nothing.
func (s *Set) readCategories(categories []string, sel Selector) ([]AlwaysAppliedEntry, error) {
	if err := s.validateSelector(sel); err != nil {
		return nil, err
	}
	var entries []AlwaysAppliedEntry
	for _, src := range s.sources {
		if !sel.covers(src) {
			continue
		}
		for _, category := range categories {
			files, err := listFiles(src.store, category)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					continue
				}
				return nil, fmt.Errorf("listing %s in knowledge store %q: %w", category, src.name, err)
			}
			for _, f := range files {
				content, err := src.store.Read(f)
				if err != nil {
					return nil, fmt.Errorf("reading %s entry %q in knowledge store %q: %w", category, f, src.name, err)
				}
				entries = append(entries, AlwaysAppliedEntry{Tier: src.tier, Name: src.name, Path: f, Content: string(content), Category: category})
			}
		}
	}
	return entries, nil
}

// Sources reports the configured stores by tier and name, with their providers
// and resolved locations, in configured order. It is the authoritative list of
// the names a request may address or narrow to.
func (s *Set) Sources() []SourceInfo {
	infos := make([]SourceInfo, len(s.sources))
	for i, src := range s.sources {
		infos[i] = SourceInfo{Tier: src.tier, Name: src.name, Provider: src.provider, Location: src.location}
	}
	return infos
}

// resolve turns an address into exactly one configured store, or into a
// refusal. It is the single place the question "which store?" is answered on
// the read and write path, so no caller can reintroduce a first-match guess.
// Every refusal names the stores available in the tier concerned.
func (s *Set) resolve(addr Address) (scopedStore, error) {
	if err := addr.Validate(); err != nil {
		var names []string
		if validTier(addr.Tier) && addr.Tier != TierAll {
			names = s.namesIn(addr.Tier)
		} else {
			names = append(s.namesIn(TierRepo), s.namesIn(TierProject)...)
		}
		return scopedStore{}, withAvailable(err, addr.Tier, names)
	}
	for _, src := range s.sources {
		if src.tier == addr.Tier && src.name == addr.Name {
			return src, nil
		}
	}
	return scopedStore{}, output.NewError(
		ErrCodeStoreUnknown,
		fmt.Sprintf("no knowledge store named %q in the %q tier", addr.Name, addr.Tier),
	).WithNextAction(availableNames(addr.Tier, s.namesIn(addr.Tier)))
}

// validateSelector checks a selector against the configured stores: the tier
// must be one of the three, and every name in the filter must belong to a store
// the tier reaches. A filter naming a store the tier does not reach is refused
// rather than quietly returning nothing, so a caller is never left reading an
// empty result as "no such knowledge". Validation lives here rather than on the
// value because only the set knows which names exist.
func (s *Set) validateSelector(sel Selector) error {
	if err := sel.Validate(); err != nil {
		return withAvailable(err, sel.Tier, s.namesIn(TierAll))
	}
	if len(sel.Filter) == 0 {
		return nil
	}
	reachable := make(map[string]bool)
	for _, name := range s.namesIn(sel.Tier) {
		reachable[name] = true
	}
	for _, name := range sel.Filter {
		if !reachable[name] {
			return output.NewError(
				ErrCodeStoreUnknown,
				fmt.Sprintf("no knowledge store named %q in the %q tier", name, sel.Tier),
			).WithNextAction(availableNames(sel.Tier, s.namesIn(sel.Tier)))
		}
	}
	return nil
}

// namesIn returns the names of the configured stores in a tier, in configured
// order. TierAll spans both tiers. It is what every refusal's next action is
// built from.
func (s *Set) namesIn(tier Tier) []string {
	var names []string
	for _, src := range s.sources {
		if tier == TierAll || src.tier == tier {
			names = append(names, src.name)
		}
	}
	return names
}

// withAvailable appends the available store names to a validation refusal's
// next action, so a caller is told both how to correct the request and what it
// may name.
func withAvailable(err error, tier Tier, names []string) error {
	var er *output.ErrorResponse
	if !errors.As(err, &er) {
		return err
	}
	listed := tier
	if !validTier(listed) || listed == TierAll {
		listed = TierAll
	}
	return er.WithNextAction(er.NextAction + "; " + availableNames(listed, names))
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
