// Package repo resolves a project's registered member repos to local
// directories: a repo's configured local path when present, otherwise a
// clone materialized into the project's working folder. Resolution hands out
// paths, never file contents — agents and stores work directly against the
// resolved root.
package repo

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// GitRunner is the narrow surface the git provider needs from git: a plain
// clone and two read-only head queries for the staleness check. It is
// deliberately small so tests can fake it and it can never grow into a git
// façade; the exec-backed implementation below is the only place in the
// codebase that shells out.
type GitRunner interface {
	// Clone performs a plain clone of url into dir (never a submodule).
	Clone(url, dir string) error
	// LocalHead returns the commit hash dir's HEAD points at. Read-only.
	LocalHead(dir string) (string, error)
	// RemoteHead returns the commit hash url's HEAD points at, without
	// fetching. Read-only and best-effort: callers treat errors as a notice,
	// never a failure.
	RemoteHead(url string) (string, error)
}

// NewGitRunner returns the GitRunner backed by the user's git binary,
// resolved from PATH at call time (git.exe included on Windows via
// LookPath). Authentication deliberately delegates to whatever credential
// machinery the platform's git provides.
func NewGitRunner() GitRunner {
	return execGitRunner{}
}

type execGitRunner struct{}

// run executes git with args, returning trimmed stdout. The child
// environment disables interactive credential prompts so an auth failure
// surfaces as an error instead of hanging an agent-driven session.
func (execGitRunner) run(args ...string) (string, error) {
	bin, err := exec.LookPath("git")
	if err != nil {
		return "", fmt.Errorf("git is not installed or not on PATH (required to resolve repos by address): %w", err)
	}

	cmd := exec.Command(bin, args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if os.Getenv("GIT_SSH_COMMAND") == "" {
		cmd.Env = append(cmd.Env, "GIT_SSH_COMMAND=ssh -oBatchMode=yes")
	}

	out, err := cmd.Output()
	if err != nil {
		var stderr string
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = strings.TrimSpace(string(ee.Stderr))
		}
		if stderr != "" {
			return "", fmt.Errorf("git %s: %s", args[0], stderr)
		}
		return "", fmt.Errorf("git %s: %w", args[0], err)
	}
	// TrimSpace (not just newline trimming) absorbs Windows CRLF endings.
	return strings.TrimSpace(string(out)), nil
}

func (g execGitRunner) Clone(url, dir string) error {
	_, err := g.run("clone", "--", url, dir)
	return err
}

func (g execGitRunner) LocalHead(dir string) (string, error) {
	return g.run("-C", dir, "rev-parse", "HEAD")
}

func (g execGitRunner) RemoteHead(url string) (string, error) {
	out, err := g.run("ls-remote", "--", url, "HEAD")
	if err != nil {
		return "", err
	}
	// Output shape: "<hash>\tHEAD" (possibly more refs on later lines).
	fields := strings.Fields(out)
	if len(fields) == 0 {
		return "", fmt.Errorf("git ls-remote returned no HEAD for %s", url)
	}
	return fields[0], nil
}
