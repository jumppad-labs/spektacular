package repo

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/stretchr/testify/require"
)

// requireGit skips the test when git is unavailable or -short is set.
func requireGit(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping git integration test in -short mode")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
}

// runGit executes git with args in dir, failing the test on error. Author
// and committer identity come from the environment so tests never depend on
// (or touch) the user's git configuration, which is masked out entirely.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=spektacular-test",
		"GIT_AUTHOR_EMAIL=test@spektacular.invalid",
		"GIT_COMMITTER_NAME=spektacular-test",
		"GIT_COMMITTER_EMAIL=test@spektacular.invalid",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_CONFIG_SYSTEM="+os.DevNull,
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, out)
	return strings.TrimSpace(string(out))
}

// newSourceRepo builds a local "remote" as a plain path: a git repo in a
// fresh temp dir holding a committed README.md and a committed, valid
// .spektacular/repo.yaml footprint, so a clone of it passes the footprint
// check. Returns the repo's path, usable directly as a RepoEntry Address.
func newSourceRepo(t *testing.T) string {
	t.Helper()
	src := t.TempDir()
	runGit(t, src, "init")
	require.NoError(t, os.WriteFile(filepath.Join(src, "README.md"), []byte("member repo\n"), 0o644))
	writeFootprint(t, src)
	runGit(t, src, "add", ".")
	runGit(t, src, "commit", "-m", "initial")
	return src
}

// commitChange writes name in src with content and commits it, advancing the
// source repo's HEAD.
func commitChange(t *testing.T, src, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(src, name), []byte(content), 0o644))
	runGit(t, src, "add", ".")
	runGit(t, src, "commit", "-m", "change "+name)
}

// Criterion 2: an address-only entry resolves by a real clone into
// <projectRoot>/.spektacular/repos/<name>, and the cloned working tree
// contains the committed file.
func TestIntegration_AddressOnlyResolvesByCloning(t *testing.T) {
	requireGit(t)
	src := newSourceRepo(t)
	projectRoot := t.TempDir()

	set := newSet(t, projectRoot, NewGitRunner(), config.RepoEntry{Name: "member", Address: src})

	r, err := set.Resolve("member")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(projectRoot, ".spektacular", MaterializeDirName, "member"), r.Root)
	require.True(t, r.Materialized)

	data, err := os.ReadFile(filepath.Join(r.Root, "README.md"))
	require.NoError(t, err)
	require.Equal(t, "member repo\n", string(data))
}

// Criterion 2: a second resolve reuses the existing clone without cloning
// again — a marker file placed in the clone survives the second resolve.
func TestIntegration_SecondResolveReusesCloneWithoutCloning(t *testing.T) {
	requireGit(t)
	src := newSourceRepo(t)
	projectRoot := t.TempDir()

	set := newSet(t, projectRoot, NewGitRunner(), config.RepoEntry{Name: "member", Address: src})

	first, err := set.Resolve("member")
	require.NoError(t, err)

	marker := filepath.Join(first.Root, "marker.txt")
	require.NoError(t, os.WriteFile(marker, []byte("still here"), 0o644))

	second, err := set.Resolve("member")
	require.NoError(t, err)
	require.Equal(t, first.Root, second.Root)
	require.FileExists(t, marker, "a re-clone would have destroyed the marker")
}

// Criterion 3: after materialization the project's git status shows no new
// tracked or gitlinked entries — the clone lands under the gitignored
// .spektacular/repos/ folder and never enters the project's history.
func TestIntegration_MaterializationLeavesProjectGitClean(t *testing.T) {
	requireGit(t)
	src := newSourceRepo(t)

	// The project root is itself a git repo, gitignoring repos/ the same way
	// project init does.
	projectRoot := t.TempDir()
	runGit(t, projectRoot, "init")
	require.NoError(t, os.MkdirAll(filepath.Join(projectRoot, ".spektacular"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, ".spektacular", ".gitignore"), []byte("repos/\n"), 0o644))
	runGit(t, projectRoot, "add", ".")
	runGit(t, projectRoot, "commit", "-m", "project init")

	set := newSet(t, projectRoot, NewGitRunner(), config.RepoEntry{Name: "member", Address: src})
	_, err := set.Resolve("member")
	require.NoError(t, err)

	status := runGit(t, projectRoot, "status", "--porcelain")
	require.Empty(t, status, "materializing a repo must leave the project's git status clean")
}

// Criterion 4: a fresh clone matches its remote's HEAD (empty StaleNote);
// after the source gains a new commit, resolving again produces a non-empty
// warning mentioning the clone path while resolution still succeeds.
func TestIntegration_CloneBehindRemoteWarns(t *testing.T) {
	requireGit(t)
	src := newSourceRepo(t)
	projectRoot := t.TempDir()

	set := newSet(t, projectRoot, NewGitRunner(), config.RepoEntry{Name: "member", Address: src})

	fresh, err := set.Resolve("member")
	require.NoError(t, err)
	require.Empty(t, fresh.StaleNote, "a fresh clone is at its remote's HEAD")

	commitChange(t, src, "new-file.txt", "newer upstream content\n")

	behind, err := set.Resolve("member")
	require.NoError(t, err, "a behind-remote clone must still resolve")
	require.NotEmpty(t, behind.StaleNote)
	require.Contains(t, behind.StaleNote, behind.Root)
}

// Criterion 4: NewGitRunner's LocalHead and RemoteHead round-trip — right
// after a clone the local head equals the remote head.
func TestIntegration_LocalAndRemoteHeadRoundTrip(t *testing.T) {
	requireGit(t)
	src := newSourceRepo(t)
	runner := NewGitRunner()

	clone := filepath.Join(t.TempDir(), "clone")
	require.NoError(t, runner.Clone(src, clone))

	local, err := runner.LocalHead(clone)
	require.NoError(t, err)
	remote, err := runner.RemoteHead(src)
	require.NoError(t, err)
	require.NotEmpty(t, local)
	require.Equal(t, remote, local)
}
