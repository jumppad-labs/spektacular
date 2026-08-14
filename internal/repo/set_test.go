package repo

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/stretchr/testify/require"
)

// writeFootprint creates a valid .spektacular/repo.yaml footprint under root.
func writeFootprint(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, ".spektacular")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, config.NewDefaultRepoConfig().ToYAMLFile(filepath.Join(dir, config.RepoConfigFileName)))
}

// writeRepoYAML creates root/.spektacular/repo.yaml with the given raw content.
func writeRepoYAML(t *testing.T, root, content string) {
	t.Helper()
	dir := filepath.Join(root, ".spektacular")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, config.RepoConfigFileName), []byte(content), 0o644))
}

// writeRepoConfig creates a valid .spektacular/repo.yaml under root carrying
// the given RepoConfig, for tests that need to control the repo's
// descriptive metadata rather than just have a valid, empty footprint.
func writeRepoConfig(t *testing.T, root string, cfg config.RepoConfig) {
	t.Helper()
	dir := filepath.Join(root, ".spektacular")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, cfg.ToYAMLFile(filepath.Join(dir, config.RepoConfigFileName)))
}

// newSet builds a Set over projectRoot from the given entries, failing the
// test if construction errors.
func newSet(t *testing.T, projectRoot string, git GitRunner, entries ...config.RepoEntry) *Set {
	t.Helper()
	cfg := config.NewDefault()
	cfg.Repos = entries
	s, err := New(cfg, projectRoot, git)
	require.NoError(t, err)
	return s
}

type cloneCall struct{ URL, Dir string }

// fakeGit is a recording GitRunner. Clone materializes the target directory
// (with a valid footprint unless omitFootprint is set) so resolution can
// proceed past the footprint check without a real git binary.
type fakeGit struct {
	t *testing.T

	CloneCalls      []cloneCall
	LocalHeadCalls  int
	RemoteHeadCalls int

	cloneErr      error
	omitFootprint bool
	localHead     string
	localHeadErr  error
	remoteHead    string
	remoteHeadErr error
}

// newFakeGit returns a fake whose local and remote heads agree, so tests that
// don't care about staleness see an empty StaleNote.
func newFakeGit(t *testing.T) *fakeGit {
	return &fakeGit{t: t, localHead: "abc123", remoteHead: "abc123"}
}

func (f *fakeGit) Clone(url, dir string) error {
	f.CloneCalls = append(f.CloneCalls, cloneCall{URL: url, Dir: dir})
	if f.cloneErr != nil {
		return f.cloneErr
	}
	require.NoError(f.t, os.MkdirAll(dir, 0o755))
	if !f.omitFootprint {
		writeFootprint(f.t, dir)
	}
	return nil
}

func (f *fakeGit) LocalHead(dir string) (string, error) {
	f.LocalHeadCalls++
	return f.localHead, f.localHeadErr
}

func (f *fakeGit) RemoteHead(url string) (string, error) {
	f.RemoteHeadCalls++
	return f.remoteHead, f.remoteHeadErr
}

// gitCalls is the total number of git invocations the fake has seen.
func (f *fakeGit) gitCalls() int {
	return len(f.CloneCalls) + f.LocalHeadCalls + f.RemoteHeadCalls
}

// Criterion 1: a repo with a usable local path resolves to it without
// touching git, even when an address is also set.
func TestResolve_LocalWinsOverAddressWithoutGit(t *testing.T) {
	localDir := t.TempDir()
	writeFootprint(t, localDir)
	git := newFakeGit(t)

	set := newSet(t, t.TempDir(), git, config.RepoEntry{
		Name:    "lib",
		Local:   localDir,
		Address: "https://example.com/lib.git",
	})

	r, err := set.Resolve("lib")
	require.NoError(t, err)
	require.Equal(t, localDir, r.Root)
	require.False(t, r.Materialized)
	require.Empty(t, r.StaleNote)
	require.Zero(t, git.gitCalls(), "resolving a local repo must never invoke git")
}

// Criterion 2: a repo with only an address is cloned into the project's
// working folder on first use.
func TestResolve_AddressOnlyClonesIntoWorkingFolder(t *testing.T) {
	projectRoot := t.TempDir()
	git := newFakeGit(t)

	set := newSet(t, projectRoot, git, config.RepoEntry{
		Name:    "member",
		Address: "https://example.com/member.git",
	})

	r, err := set.Resolve("member")
	require.NoError(t, err)

	wantRoot := filepath.Join(projectRoot, ".spektacular", MaterializeDirName, "member")
	require.Equal(t, wantRoot, r.Root)
	require.True(t, r.Materialized)
	require.Equal(t, []cloneCall{{URL: "https://example.com/member.git", Dir: wantRoot}}, git.CloneCalls)
}

// Criterion 2: an existing clone is reused — a second Resolve makes no
// further Clone calls.
func TestResolve_ExistingCloneReusedWithoutCloning(t *testing.T) {
	projectRoot := t.TempDir()
	git := newFakeGit(t)

	set := newSet(t, projectRoot, git, config.RepoEntry{
		Name:    "member",
		Address: "https://example.com/member.git",
	})

	_, err := set.Resolve("member")
	require.NoError(t, err)
	require.Len(t, git.CloneCalls, 1)

	r, err := set.Resolve("member")
	require.NoError(t, err)
	require.True(t, r.Materialized)
	require.Len(t, git.CloneCalls, 1, "second resolve must not clone again")
}

// Criterion 4: a materialized clone whose head differs from its remote's
// produces a non-empty warn note while resolution still succeeds.
func TestResolve_StaleNoteWhenHeadsDiffer(t *testing.T) {
	git := newFakeGit(t)
	git.localHead = "aaa111"
	git.remoteHead = "bbb222"

	set := newSet(t, t.TempDir(), git, config.RepoEntry{
		Name:    "member",
		Address: "https://example.com/member.git",
	})

	r, err := set.Resolve("member")
	require.NoError(t, err, "staleness must never fail resolution")
	require.NotEmpty(t, r.StaleNote)
}

// Criterion 4: an unreachable remote degrades to a could-not-check notice
// while resolution still succeeds.
func TestResolve_RemoteHeadErrorDegradesToNotice(t *testing.T) {
	git := newFakeGit(t)
	git.remoteHeadErr = errors.New("network unreachable")

	set := newSet(t, t.TempDir(), git, config.RepoEntry{
		Name:    "member",
		Address: "https://example.com/member.git",
	})

	r, err := set.Resolve("member")
	require.NoError(t, err, "an unreachable remote must never fail resolution")
	require.Contains(t, r.StaleNote, "could not check")
}

// Criterion 4: equal local and remote heads produce no stale note.
func TestResolve_EqualHeadsProduceNoStaleNote(t *testing.T) {
	git := newFakeGit(t) // local and remote heads agree by default

	set := newSet(t, t.TempDir(), git, config.RepoEntry{
		Name:    "member",
		Address: "https://example.com/member.git",
	})

	r, err := set.Resolve("member")
	require.NoError(t, err)
	require.Empty(t, r.StaleNote)
}

// Criterion 4: a LocalHead failure silences the staleness check entirely —
// empty note, successful resolution.
func TestResolve_LocalHeadErrorSilencesStaleCheck(t *testing.T) {
	git := newFakeGit(t)
	git.localHeadErr = errors.New("not a git repository")

	set := newSet(t, t.TempDir(), git, config.RepoEntry{
		Name:    "member",
		Address: "https://example.com/member.git",
	})

	r, err := set.Resolve("member")
	require.NoError(t, err)
	require.Empty(t, r.StaleNote)
}

// Criterion 5: a local repo directory without a .spektacular/repo.yaml
// resolves to a *FootprintError carrying the repo name and root, while the
// returned ResolvedRepo still has its Root populated for the repair offer.
func TestResolve_MissingFootprintReturnsFootprintError(t *testing.T) {
	localDir := t.TempDir() // exists, but no footprint inside

	set := newSet(t, t.TempDir(), newFakeGit(t), config.RepoEntry{
		Name:  "lib",
		Local: localDir,
	})

	r, err := set.Resolve("lib")
	require.Error(t, err)

	var fe *FootprintError
	require.ErrorAs(t, err, &fe)
	require.Equal(t, "lib", fe.Repo)
	require.Equal(t, localDir, fe.Root)
	require.Equal(t, localDir, r.Root, "the resolved root must accompany the footprint error")
}

// Criterion 5: an invalid repo.yaml (unsupported changelog provider) also
// yields a *FootprintError rather than a silent continuation.
func TestResolve_InvalidFootprintReturnsFootprintError(t *testing.T) {
	localDir := t.TempDir()
	writeRepoYAML(t, localDir, "changelog:\n  provider: unknown\n")

	set := newSet(t, t.TempDir(), newFakeGit(t), config.RepoEntry{
		Name:  "lib",
		Local: localDir,
	})

	_, err := set.Resolve("lib")
	var fe *FootprintError
	require.ErrorAs(t, err, &fe)
	require.Equal(t, "lib", fe.Repo)
	require.Equal(t, localDir, fe.Root)
}

// New fails fast on an unsupported provider, naming the repo and the provider.
func TestNew_UnknownProviderFailsFast(t *testing.T) {
	cfg := config.NewDefault()
	cfg.Repos = []config.RepoEntry{{Name: "lib", Provider: "svn", Local: t.TempDir()}}

	set, err := New(cfg, t.TempDir(), newFakeGit(t))
	require.Error(t, err)
	require.Nil(t, set)
	require.Contains(t, err.Error(), "lib")
	require.Contains(t, err.Error(), "svn")
}

// Resolving a name that is not registered errors.
func TestResolve_UnregisteredNameErrors(t *testing.T) {
	set := newSet(t, t.TempDir(), newFakeGit(t))

	_, err := set.Resolve("ghost")
	require.Error(t, err)
	require.Contains(t, err.Error(), "ghost")
}

// An empty provider defaults to git and resolves fine.
func TestNew_EmptyProviderDefaultsToGit(t *testing.T) {
	localDir := t.TempDir()
	writeFootprint(t, localDir)

	set := newSet(t, t.TempDir(), newFakeGit(t), config.RepoEntry{
		Name:  "lib",
		Local: localDir,
	})

	entries := set.Entries()
	require.Len(t, entries, 1)
	require.Equal(t, config.ProviderGit, entries[0].Provider)

	r, err := set.Resolve("lib")
	require.NoError(t, err)
	require.Equal(t, localDir, r.Root)
}

// A local path that does not exist, with no address to fall back on, errors
// naming the missing path; git is never invoked.
func TestResolve_LocalMissingWithoutAddressErrorsNamingPath(t *testing.T) {
	projectRoot := t.TempDir()
	git := newFakeGit(t)

	set := newSet(t, projectRoot, git, config.RepoEntry{
		Name:  "lib",
		Local: "gone",
	})

	_, err := set.Resolve("lib")
	require.Error(t, err)
	require.Contains(t, err.Error(), filepath.Join(projectRoot, "gone"))
	require.Zero(t, git.gitCalls())
}

// A local path that does not exist falls through to the address when one is
// configured, materializing a clone.
func TestResolve_LocalMissingFallsThroughToAddress(t *testing.T) {
	projectRoot := t.TempDir()
	git := newFakeGit(t)

	set := newSet(t, projectRoot, git, config.RepoEntry{
		Name:    "member",
		Local:   "gone",
		Address: "https://example.com/member.git",
	})

	r, err := set.Resolve("member")
	require.NoError(t, err)
	require.True(t, r.Materialized)
	require.Equal(t, filepath.Join(projectRoot, ".spektacular", MaterializeDirName, "member"), r.Root)
	require.Len(t, git.CloneCalls, 1)
}

// A relative local path is joined to the project root.
func TestResolve_RelativeLocalJoinedToProjectRoot(t *testing.T) {
	projectRoot := t.TempDir()
	localDir := filepath.Join(projectRoot, "vendor", "lib")
	require.NoError(t, os.MkdirAll(localDir, 0o755))
	writeFootprint(t, localDir)

	set := newSet(t, projectRoot, newFakeGit(t), config.RepoEntry{
		Name:  "lib",
		Local: filepath.Join("vendor", "lib"),
	})

	r, err := set.Resolve("lib")
	require.NoError(t, err)
	require.Equal(t, localDir, r.Root)
}

// DescriptiveMetadata returns the named repo's own descriptive metadata when
// the repo is materialized (on disk, local path present) and its repo.yaml
// is readable.
func TestDescriptiveMetadata_MaterializedAndReadableReturnsMetadata(t *testing.T) {
	localDir := t.TempDir()
	repoCfg := config.NewDefaultRepoConfig()
	repoCfg.Description = "the documentation repo"
	repoCfg.Role = "documentation"
	repoCfg.Tags = []string{"docs", "markdown"}
	repoCfg.Deployment = "static site on the CDN"
	writeRepoConfig(t, localDir, repoCfg)

	set := newSet(t, t.TempDir(), newFakeGit(t), config.RepoEntry{
		Name:  "docs",
		Local: localDir,
	})

	meta, ok := set.DescriptiveMetadata("docs")
	require.True(t, ok)
	require.Equal(t, "the documentation repo", meta.Description)
	require.Equal(t, "documentation", meta.Role)
	require.Equal(t, []string{"docs", "markdown"}, meta.Tags)
	require.Equal(t, "static site on the CDN", meta.Deployment)
}

// DescriptiveMetadata reports absent (zero value, false) for an
// unmaterialized repo — no local path present and no clone on disk — and
// never invokes git to find out.
func TestDescriptiveMetadata_UnmaterializedReturnsAbsentWithoutGit(t *testing.T) {
	git := newFakeGit(t)
	set := newSet(t, t.TempDir(), git, config.RepoEntry{
		Name:    "member",
		Address: "https://example.com/member.git",
	})

	meta, ok := set.DescriptiveMetadata("member")
	require.False(t, ok)
	require.Equal(t, config.RepoConfig{}, meta)
	require.Zero(t, git.gitCalls(), "DescriptiveMetadata must never clone or fetch")
}

// DescriptiveMetadata reports absent, not an error, when the repo is
// materialized but its repo.yaml is missing.
func TestDescriptiveMetadata_MaterializedMissingRepoYAMLReturnsAbsent(t *testing.T) {
	localDir := t.TempDir() // exists, but no .spektacular/repo.yaml inside

	set := newSet(t, t.TempDir(), newFakeGit(t), config.RepoEntry{
		Name:  "lib",
		Local: localDir,
	})

	meta, ok := set.DescriptiveMetadata("lib")
	require.False(t, ok)
	require.Equal(t, config.RepoConfig{}, meta)
}

// DescriptiveMetadata reports absent, not an error, when the repo is
// materialized but its repo.yaml is present and corrupt/invalid.
func TestDescriptiveMetadata_MaterializedCorruptRepoYAMLReturnsAbsent(t *testing.T) {
	localDir := t.TempDir()
	writeRepoYAML(t, localDir, "changelog:\n  provider: unknown\n")

	set := newSet(t, t.TempDir(), newFakeGit(t), config.RepoEntry{
		Name:  "lib",
		Local: localDir,
	})

	meta, ok := set.DescriptiveMetadata("lib")
	require.False(t, ok)
	require.Equal(t, config.RepoConfig{}, meta)
}

// DescriptiveMetadata reports absent for a name that isn't registered at
// all, without error.
func TestDescriptiveMetadata_UnregisteredNameReturnsAbsent(t *testing.T) {
	set := newSet(t, t.TempDir(), newFakeGit(t))

	meta, ok := set.DescriptiveMetadata("ghost")
	require.False(t, ok)
	require.Equal(t, config.RepoConfig{}, meta)
}

// ResolveAll returns every repo in registry (configuration) order.
func TestResolveAll_ReturnsRegistryOrder(t *testing.T) {
	dirB := t.TempDir()
	writeFootprint(t, dirB)
	dirA := t.TempDir()
	writeFootprint(t, dirA)

	set := newSet(t, t.TempDir(), newFakeGit(t),
		config.RepoEntry{Name: "bravo", Local: dirB},
		config.RepoEntry{Name: "alpha", Local: dirA},
	)

	resolved, err := set.ResolveAll()
	require.NoError(t, err)

	names := make([]string, len(resolved))
	for i, r := range resolved {
		names[i] = r.Name
	}
	require.Equal(t, []string{"bravo", "alpha"}, names)
}
