package repo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jumppad-labs/spektacular/internal/config"
)

// MaterializeDirName is the project-relative folder clones land in:
// .spektacular/repos/<repo-name>/. Project init gitignores it so a
// materialized repo can never enter the project's git history.
const MaterializeDirName = "repos"

// ResolvedRepo is a registered repo resolved to a usable local directory —
// the only access handle the registry hands out.
type ResolvedRepo struct {
	Name         string
	Root         string // absolute local directory
	Entry        config.RepoEntry
	Materialized bool   // true when Root is a project-managed clone
	StaleNote    string // non-empty ⇒ warn-only behind-remote (or check-failed) notice
}

// FootprintError reports a registered repo whose .spektacular footprint is
// missing or invalid. It carries what the CLI layer needs to surface the
// problem with an offer to repair, rather than a bare failure.
type FootprintError struct {
	Repo string // registry name
	Root string // resolved local root
	Err  error  // what is wrong with the footprint
}

func (e *FootprintError) Error() string {
	return fmt.Sprintf("repo %q at %s has a missing or invalid footprint: %v", e.Repo, e.Root, e.Err)
}

func (e *FootprintError) Unwrap() error { return e.Err }

// Set is the project's repo registry resolved against a project root. It is
// constructed from configuration, dispatches providers fail-fast, and
// resolves entries on demand — materializing clones only when a repo is
// actually used.
type Set struct {
	projectRoot string
	git         GitRunner
	entries     []config.RepoEntry // defaults applied, registry order
}

// New builds the registry from the project configuration. Unknown providers
// fail fast here, before any entry is resolved, mirroring knowledge.NewSet's
// contract.
func New(cfg config.Config, projectRoot string, git GitRunner) (*Set, error) {
	entries := make([]config.RepoEntry, 0, len(cfg.Repos))
	for _, e := range cfg.Repos {
		e = e.WithDefaults()
		if e.Provider != config.ProviderGit {
			return nil, fmt.Errorf("repo %q: provider %q is not supported (only %q)", e.Name, e.Provider, config.ProviderGit)
		}
		entries = append(entries, e)
	}
	return &Set{projectRoot: projectRoot, git: git, entries: entries}, nil
}

// Entries returns the registry entries in configuration order, with
// defaults applied. It performs no resolution and touches nothing on disk.
func (s *Set) Entries() []config.RepoEntry {
	out := make([]config.RepoEntry, len(s.entries))
	copy(out, s.entries)
	return out
}

// Present reports whether the named repo is already on disk — its local
// path exists, or a materialized clone exists — without resolving it and
// without ever invoking git. Callers that must stay side-effect-free (like
// listing) use this to avoid triggering a clone.
func (s *Set) Present(name string) bool {
	_, ok := s.LocalRoot(name)
	return ok
}

// LocalRoot returns the named repo's on-disk root — its local path when that
// exists, otherwise its materialized clone when that exists — without
// resolving it and without ever invoking git. The second return is false
// when the repo is not on disk (or not registered).
func (s *Set) LocalRoot(name string) (string, bool) {
	for _, e := range s.entries {
		if e.Name != name {
			continue
		}
		if e.Local != "" {
			root := e.Local
			if !filepath.IsAbs(root) {
				root = filepath.Join(s.projectRoot, root)
			}
			if info, err := os.Stat(root); err == nil && info.IsDir() {
				return root, true
			}
		}
		if e.Address != "" {
			clone := filepath.Join(s.projectRoot, ".spektacular", MaterializeDirName, e.Name)
			if info, err := os.Stat(clone); err == nil && info.IsDir() {
				return clone, true
			}
		}
		return "", false
	}
	return "", false
}

// DescriptiveMetadata returns the named repo's own descriptive metadata —
// what it says about itself in its own repo.yaml — if the repo is on disk
// and its config is readable. It never clones or fetches: an unmaterialized
// repo, or one whose repo.yaml is missing or unreadable, reports absent
// rather than an error, mirroring how checkFootprint tolerates the same
// conditions for its own purposes. This is the single place that decides
// how a repo's descriptive metadata is read, so every caller that surfaces
// it (repo listing, the plan workflow's repo roster) sees the same view.
func (s *Set) DescriptiveMetadata(name string) (config.RepoConfig, bool) {
	root, ok := s.LocalRoot(name)
	if !ok {
		return config.RepoConfig{}, false
	}
	cfg, err := config.RepoConfigFromYAMLFile(filepath.Join(root, ".spektacular", config.RepoConfigFileName))
	if err != nil {
		return config.RepoConfig{}, false
	}
	return cfg, true
}

// Resolve resolves the named repo to a local directory: its local path when
// set and present, otherwise a clone of its address materialized into the
// project's working folder — only when absent, never fetched or pulled.
// After resolution the repo's footprint is validated; a missing or broken
// footprint returns a *FootprintError carrying a repair offer, never a
// silent continuation.
func (s *Set) Resolve(name string) (ResolvedRepo, error) {
	for _, e := range s.entries {
		if e.Name == name {
			return s.resolve(e)
		}
	}
	return ResolvedRepo{}, fmt.Errorf("repo %q is not registered in this project", name)
}

// ResolveAll resolves every registered repo in registry order, failing on
// the first repo that cannot be resolved.
func (s *Set) ResolveAll() ([]ResolvedRepo, error) {
	resolved := make([]ResolvedRepo, 0, len(s.entries))
	for _, e := range s.entries {
		r, err := s.resolve(e)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, r)
	}
	return resolved, nil
}

func (s *Set) resolve(e config.RepoEntry) (ResolvedRepo, error) {
	// Local wins when set and present; git is never invoked for it.
	if e.Local != "" {
		root := e.Local
		if !filepath.IsAbs(root) {
			root = filepath.Join(s.projectRoot, root)
		}
		if info, err := os.Stat(root); err == nil && info.IsDir() {
			r := ResolvedRepo{Name: e.Name, Root: root, Entry: e}
			return r, s.checkFootprint(r)
		}
		if e.Address == "" {
			return ResolvedRepo{}, fmt.Errorf("repo %q: local path %s does not exist and no address is configured to clone from", e.Name, root)
		}
	}

	if e.Address == "" {
		return ResolvedRepo{}, fmt.Errorf("repo %q: no usable local path and no address configured", e.Name)
	}

	// Materialize by cloning into the project's working folder — only when
	// the clone is absent. An existing clone is reused as-is; keeping it
	// current is the user's responsibility.
	root := filepath.Join(s.projectRoot, ".spektacular", MaterializeDirName, e.Name)
	if _, err := os.Stat(root); os.IsNotExist(err) {
		if err := s.git.Clone(e.Address, root); err != nil {
			return ResolvedRepo{}, fmt.Errorf("repo %q: cloning %s: %w", e.Name, e.Address, err)
		}
	}

	r := ResolvedRepo{
		Name:         e.Name,
		Root:         root,
		Entry:        e,
		Materialized: true,
		StaleNote:    s.staleNote(root, e.Address),
	}
	return r, s.checkFootprint(r)
}

// staleNote compares a materialized clone's head against its remote and
// composes a warn-only notice. Every failure degrades to a notice or
// silence — the staleness check never fails resolution, and no fetch or
// pull ever runs.
func (s *Set) staleNote(root, address string) string {
	local, err := s.git.LocalHead(root)
	if err != nil {
		return ""
	}
	remote, err := s.git.RemoteHead(address)
	if err != nil {
		return fmt.Sprintf("could not check %s against its remote: %v", root, err)
	}
	if local != remote {
		return fmt.Sprintf("the clone at %s is not at its remote's HEAD (local %s, remote %s); Spektacular never fetches or pulls — update it yourself if needed", root, local, remote)
	}
	return ""
}

// checkFootprint validates the resolved repo's minimal footprint — a
// parseable, valid .spektacular/repo.yaml — returning a *FootprintError
// when it is missing or broken.
func (s *Set) checkFootprint(r ResolvedRepo) error {
	path := filepath.Join(r.Root, ".spektacular", config.RepoConfigFileName)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &FootprintError{Repo: r.Name, Root: r.Root, Err: fmt.Errorf("missing %s", filepath.Join(".spektacular", config.RepoConfigFileName))}
	}
	if _, err := config.RepoConfigFromYAMLFile(path); err != nil {
		return &FootprintError{Repo: r.Name, Root: r.Root, Err: err}
	}
	return nil
}
