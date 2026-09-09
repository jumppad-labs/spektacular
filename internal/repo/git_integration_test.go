package repo

import (
	"net/http"
	"net/http/httptest"
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

// newSourceRepo builds a local "remote": a git repo in a fresh temp dir
// holding a committed README.md. The repo carries no Spektacular footprint —
// that lives at the registered location, not in the code. Returns the
// repo's path.
func newSourceRepo(t *testing.T) string {
	t.Helper()
	src := t.TempDir()
	runGit(t, src, "init")
	require.NoError(t, os.WriteFile(filepath.Join(src, "README.md"), []byte("member repo\n"), 0o644))
	runGit(t, src, "add", ".")
	runGit(t, src, "commit", "-m", "initial")
	runGit(t, src, "update-server-info")
	return src
}

// commitChange writes name in src with content and commits it, advancing the
// source repo's HEAD (and refreshing the dumb-HTTP ref advertisement).
func commitChange(t *testing.T, src, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(src, name), []byte(content), 0o644))
	runGit(t, src, "add", ".")
	runGit(t, src, "commit", "-m", "change "+name)
	runGit(t, src, "update-server-info")
}

// serveGitOverHTTP exposes src over git's dumb HTTP protocol from a loopback
// server and returns the URL a git client clones it from. A plain path
// would be classified as a file source by repo.yaml, so the real-git tests
// need a genuine git transport; static files behind http:// are the one git
// accepts without an ssh daemon or a smart server.
func serveGitOverHTTP(t *testing.T, src string) string {
	t.Helper()
	t.Setenv("NO_PROXY", "127.0.0.1")
	srv := httptest.NewServer(http.FileServer(http.Dir(src)))
	t.Cleanup(srv.Close)
	return srv.URL + "/.git"
}

// newIntegrationSet registers a single repo named "member" whose location is
// a fresh temp dir holding a repo.yaml with the given git source, over
// projectRoot with the real git runner. Returns the set and the location.
func newIntegrationSet(t *testing.T, projectRoot, source string) (*Set, string) {
	t.Helper()
	root := t.TempDir()
	writeSourceFootprint(t, root, source)
	return newSet(t, projectRoot, NewGitRunner(), config.RepoEntry{Name: "member", Location: root}), root
}

// Criterion 2: a git source declared in repo.yaml resolves by a real clone
// into <projectRoot>/.spektacular/repos/<name>: the root stays the
// registered location, the source is the clone, and the cloned working tree
// contains the committed file.
func TestIntegration_GitSourceResolvesByCloning(t *testing.T) {
	requireGit(t)
	url := serveGitOverHTTP(t, newSourceRepo(t))
	projectRoot := t.TempDir()
	set, root := newIntegrationSet(t, projectRoot, url)

	r, err := set.Resolve("member")
	require.NoError(t, err)
	require.Equal(t, root, r.Root)
	require.Equal(t, filepath.Join(projectRoot, ".spektacular", "repos", "member"), r.Source)
	require.True(t, r.Materialized)

	data, err := os.ReadFile(filepath.Join(r.Source, "README.md"))
	require.NoError(t, err)
	require.Equal(t, "member repo\n", string(data))
}

// Criterion 2: a second resolve reuses the existing clone without cloning
// again — a marker file placed in the clone survives the second resolve.
func TestIntegration_SecondResolveReusesCloneWithoutCloning(t *testing.T) {
	requireGit(t)
	url := serveGitOverHTTP(t, newSourceRepo(t))
	set, _ := newIntegrationSet(t, t.TempDir(), url)

	first, err := set.Resolve("member")
	require.NoError(t, err)

	marker := filepath.Join(first.Source, "marker.txt")
	require.NoError(t, os.WriteFile(marker, []byte("still here"), 0o644))

	second, err := set.Resolve("member")
	require.NoError(t, err)
	require.Equal(t, first.Source, second.Source)
	require.FileExists(t, marker, "a re-clone would have destroyed the marker")
}

// Criterion 3: after materialization the project's git status shows no new
// tracked or gitlinked entries — the clone lands under the gitignored
// .spektacular/repos/ folder and never enters the project's history.
func TestIntegration_MaterializationLeavesProjectGitClean(t *testing.T) {
	requireGit(t)
	url := serveGitOverHTTP(t, newSourceRepo(t))

	// The project root is itself a git repo, gitignoring repos/ the same way
	// project init does.
	projectRoot := t.TempDir()
	runGit(t, projectRoot, "init")
	require.NoError(t, os.MkdirAll(filepath.Join(projectRoot, ".spektacular"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, ".spektacular", ".gitignore"), []byte("repos/\n"), 0o644))
	runGit(t, projectRoot, "add", ".")
	runGit(t, projectRoot, "commit", "-m", "project init")

	set, _ := newIntegrationSet(t, projectRoot, url)
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
	url := serveGitOverHTTP(t, src)
	set, _ := newIntegrationSet(t, t.TempDir(), url)

	fresh, err := set.Resolve("member")
	require.NoError(t, err)
	require.Empty(t, fresh.StaleNote, "a fresh clone is at its remote's HEAD")

	commitChange(t, src, "new-file.txt", "newer upstream content\n")

	behind, err := set.Resolve("member")
	require.NoError(t, err, "a behind-remote clone must still resolve")
	require.NotEmpty(t, behind.StaleNote)
	require.Contains(t, behind.StaleNote, behind.Source)
}

// Criterion 4: NewGitRunner's LocalHead and RemoteHead round-trip — right
// after a clone the local head equals the remote head.
func TestIntegration_LocalAndRemoteHeadRoundTrip(t *testing.T) {
	requireGit(t)
	url := serveGitOverHTTP(t, newSourceRepo(t))
	runner := NewGitRunner()

	clone := filepath.Join(t.TempDir(), "clone")
	require.NoError(t, runner.Clone(url, clone))

	local, err := runner.LocalHead(clone)
	require.NoError(t, err)
	remote, err := runner.RemoteHead(url)
	require.NoError(t, err)
	require.NotEmpty(t, local)
	require.Equal(t, remote, local)
}
