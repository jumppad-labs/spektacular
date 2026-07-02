package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jumppad-labs/spektacular/internal/output"
	"github.com/stretchr/testify/require"
)

// This file tests cmd/root.go's response-wrapper behavior (runRoot): every
// command's success or failure outcome must be funneled through the same
// envelope and the same output stream, exactly once, with no cobra-generated
// boilerplate. Business logic for each command family is covered elsewhere
// (spec_test.go, plan_test.go, implement_test.go, knowledge_test.go,
// plan_file_test.go, file_test.go, init_test.go, skill.go's own tests); the
// tests here only exercise a representative success and failure per family to
// prove the wrapper itself behaves correctly.

// resetPlanCommandFlags clears the persistent and per-command flags used by
// the plan command tree between runs, mirroring resetSpecCommandFlags /
// resetKnowledgeFlags, so a flag set by one subtest does not leak into the
// next.
func resetPlanCommandFlags(t *testing.T) {
	t.Helper()
	reset := func() {
		require.NoError(t, planCmd.PersistentFlags().Set("schema", "false"))
		require.NoError(t, planCmd.PersistentFlags().Set("dry-run", "false"))
		require.NoError(t, planNewCmd.Flags().Set("data", ""))
		require.NoError(t, planNewCmd.Flags().Set("force", "false"))
		require.NoError(t, planNewCmd.Flags().Set("stdin", ""))
		require.NoError(t, planNewCmd.Flags().Set("file", ""))
		require.NoError(t, planGotoCmd.Flags().Set("data", ""))
		require.NoError(t, planGotoCmd.Flags().Set("stdin", ""))
		require.NoError(t, planGotoCmd.Flags().Set("file", ""))
	}
	reset()
	t.Cleanup(reset)
}

// resetImplementCommandFlags is the implement-command-tree equivalent of
// resetPlanCommandFlags.
func resetImplementCommandFlags(t *testing.T) {
	t.Helper()
	reset := func() {
		require.NoError(t, implementCmd.PersistentFlags().Set("schema", "false"))
		require.NoError(t, implementCmd.PersistentFlags().Set("dry-run", "false"))
		require.NoError(t, implementNewCmd.Flags().Set("data", ""))
		require.NoError(t, implementNewCmd.Flags().Set("force", "false"))
		require.NoError(t, implementNewCmd.Flags().Set("stdin", ""))
		require.NoError(t, implementNewCmd.Flags().Set("file", ""))
		require.NoError(t, implementGotoCmd.Flags().Set("data", ""))
		require.NoError(t, implementGotoCmd.Flags().Set("stdin", ""))
		require.NoError(t, implementGotoCmd.Flags().Set("file", ""))
	}
	reset()
	t.Cleanup(reset)
}

// runRootCmd invokes rootCmd's args through runRoot — the same wrapper
// Execute() uses in production, rather than calling rootCmd.Execute()
// directly — so tests observe the wrapper's formatted output and exit code,
// not just cobra's raw error return.
func runRootCmd(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	out, errBuf := setupImplementCmd(t)
	rootCmd.SetArgs(args)
	code = runRoot()
	return out.String(), errBuf.String(), code
}

// writeSpecFileFixture lays out a temp project rooted at a t.TempDir() (and
// chdirs into it) with the spec directory configured to docs/specs and one
// existing file seeded at feature.md, for exercising `spec file`.
func writeSpecFileFixture(t *testing.T) (dir string) {
	t.Helper()
	dir = t.TempDir()
	t.Chdir(dir)
	writeSpecCommandConfig(t, dir, "spec:\n  config:\n    directory: docs/specs\n")
	specPath := filepath.Join(dir, "docs", "specs", "feature.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(specPath), 0o755))
	require.NoError(t, os.WriteFile(specPath, []byte("stored body"), 0o644))
	return dir
}

// Criterion 1: every command family reports both success and failure on
// stdout, and stderr stays empty in both cases.
func TestWrapper_SuccessAndFailureBothStreamOnStdoutOnly(t *testing.T) {
	t.Run("spec", func(t *testing.T) {
		// Separate temp dirs per invocation: `spec new`'s resume check runs
		// before its name validation, so a successful call followed by a
		// second call in the same directory would be treated as a resume
		// (success) instead of exercising the validation failure.
		t.Run("success", func(t *testing.T) {
			t.Chdir(t.TempDir())
			resetSpecCommandFlags(t)
			stdout, stderr, code := runRootCmd(t, "spec", "new", "--data", `{"name":"billing"}`)
			require.Equal(t, 0, code)
			require.NotEmpty(t, stdout)
			require.Empty(t, stderr)
		})
		t.Run("failure", func(t *testing.T) {
			t.Chdir(t.TempDir())
			resetSpecCommandFlags(t)
			stdout, stderr, code := runRootCmd(t, "spec", "new", "--data", `{"name":" billing"}`)
			require.Equal(t, 1, code)
			require.NotEmpty(t, stdout)
			require.Empty(t, stderr)
		})
	})

	t.Run("plan", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)

		resetPlanCommandFlags(t)
		stdout, stderr, code := runRootCmd(t, "plan", "new", "--data", `{"name":"myplan"}`)
		require.Equal(t, 0, code)
		require.NotEmpty(t, stdout)
		require.Empty(t, stderr)

		resetPlanCommandFlags(t)
		stdout, stderr, code = runRootCmd(t, "plan", "new")
		require.Equal(t, 1, code)
		require.NotEmpty(t, stdout)
		require.Empty(t, stderr)
	})

	t.Run("implement", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		dataDir := filepath.Join(dir, ".spektacular")
		require.NoError(t, os.MkdirAll(dataDir, 0o755))
		writeFixturePlan(t, dataDir, "fixture")

		resetImplementCommandFlags(t)
		stdout, stderr, code := runRootCmd(t, "implement", "new", "--data", `{"name":"fixture"}`)
		require.Equal(t, 0, code)
		require.NotEmpty(t, stdout)
		require.Empty(t, stderr)

		resetImplementCommandFlags(t)
		stdout, stderr, code = runRootCmd(t, "implement", "new", "--data", `{"name":"nosuch"}`)
		require.Equal(t, 1, code)
		require.NotEmpty(t, stdout)
		require.Empty(t, stderr)
	})

	t.Run("knowledge", func(t *testing.T) {
		twoScopeProject(t)

		stdout, stderr, err := runKnowledge(t, "sources")
		require.NoError(t, err)
		require.NotEmpty(t, stdout)
		require.Empty(t, stderr)

		stdout, stderr, err = runKnowledge(t, "read")
		require.Error(t, err)
		require.NotEmpty(t, stdout)
		require.Empty(t, stderr)
	})

	t.Run("file", func(t *testing.T) {
		writeSpecFileFixture(t)

		// file read's success path is a deliberate, documented exception: it
		// writes raw file bytes to stdout, not the JSON envelope. Only its
		// failure path goes through the envelope.
		stdout, stderr, code := runRootCmd(t, "spec", "file", "read", "feature.md")
		require.Equal(t, 0, code)
		require.Equal(t, "stored body", stdout)
		require.Empty(t, stderr)

		stdout, stderr, code = runRootCmd(t, "spec", "file", "read", "missing.md")
		require.Equal(t, 1, code)
		require.NotEmpty(t, stdout)
		require.Empty(t, stderr)
	})

	t.Run("skill", func(t *testing.T) {
		stdout, stderr, code := runRootCmd(t, "skill", "determine-feature-slug")
		require.Equal(t, 0, code)
		require.NotEmpty(t, stdout)
		require.Empty(t, stderr)

		stdout, stderr, code = runRootCmd(t, "skill", "does-not-exist")
		require.Equal(t, 1, code)
		require.NotEmpty(t, stdout)
		require.Empty(t, stderr)
	})

	t.Run("init", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)

		stdout, stderr, code := runRootCmd(t, "init", "claude")
		require.Equal(t, 0, code)
		require.NotEmpty(t, stdout)
		require.Empty(t, stderr)

		stdout, stderr, code = runRootCmd(t, "init", "does-not-exist")
		require.Equal(t, 1, code)
		require.NotEmpty(t, stdout)
		require.Empty(t, stderr)
	})
}

// Criterion 2: a failing command's response carries "error": true and
// runRoot() returns 1; a succeeding command's response carries "error": false
// and runRoot() returns 0. Checked across every family whose success path
// goes through the JSON envelope; init and file read's success paths are
// deliberate plain-text/raw-bytes exceptions (see TestWrapper_
// SuccessAndFailureBothStreamOnStdoutOnly), so only their failure paths are
// asserted here.
func TestWrapper_ErrorDiscriminantAndExitCode(t *testing.T) {
	assertSuccessEnvelope := func(t *testing.T, stdout string, code int) {
		t.Helper()
		require.Equal(t, 0, code)
		var m map[string]any
		require.NoError(t, json.Unmarshal([]byte(stdout), &m))
		require.Equal(t, false, m["error"])
	}
	assertFailureEnvelope := func(t *testing.T, stdout string, code int) {
		t.Helper()
		require.Equal(t, 1, code)
		var er output.ErrorResponse
		require.NoError(t, json.Unmarshal([]byte(stdout), &er))
		require.True(t, er.IsError)
	}

	t.Run("spec", func(t *testing.T) {
		// Separate temp dirs per invocation — see the identical note in
		// TestWrapper_SuccessAndFailureBothStreamOnStdoutOnly.
		t.Run("success", func(t *testing.T) {
			t.Chdir(t.TempDir())
			resetSpecCommandFlags(t)
			stdout, _, code := runRootCmd(t, "spec", "new", "--data", `{"name":"billing"}`)
			assertSuccessEnvelope(t, stdout, code)
		})
		t.Run("failure", func(t *testing.T) {
			t.Chdir(t.TempDir())
			resetSpecCommandFlags(t)
			stdout, _, code := runRootCmd(t, "spec", "new", "--data", `{"name":" billing"}`)
			assertFailureEnvelope(t, stdout, code)
		})
	})

	t.Run("plan", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)

		resetPlanCommandFlags(t)
		stdout, _, code := runRootCmd(t, "plan", "new", "--data", `{"name":"myplan"}`)
		assertSuccessEnvelope(t, stdout, code)

		resetPlanCommandFlags(t)
		stdout, _, code = runRootCmd(t, "plan", "new")
		assertFailureEnvelope(t, stdout, code)
	})

	t.Run("implement", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		dataDir := filepath.Join(dir, ".spektacular")
		require.NoError(t, os.MkdirAll(dataDir, 0o755))
		writeFixturePlan(t, dataDir, "fixture")

		resetImplementCommandFlags(t)
		stdout, _, code := runRootCmd(t, "implement", "new", "--data", `{"name":"fixture"}`)
		assertSuccessEnvelope(t, stdout, code)

		resetImplementCommandFlags(t)
		stdout, _, code = runRootCmd(t, "implement", "new", "--data", `{"name":"nosuch"}`)
		assertFailureEnvelope(t, stdout, code)
	})

	t.Run("knowledge", func(t *testing.T) {
		twoScopeProject(t)

		stdout, _, err := runKnowledge(t, "sources")
		require.NoError(t, err)
		assertSuccessEnvelope(t, stdout, 0)

		stdout, _, err = runKnowledge(t, "read")
		require.Error(t, err)
		assertFailureEnvelope(t, stdout, 1)
	})

	t.Run("file", func(t *testing.T) {
		writeSpecFileFixture(t)

		// `list` (unlike `read`) goes through the standard envelope on
		// success, so it stands in as this family's envelope-shaped success
		// representative.
		stdout, _, code := runRootCmd(t, "spec", "file", "list")
		assertSuccessEnvelope(t, stdout, code)

		stdout, _, code = runRootCmd(t, "spec", "file", "read", "missing.md")
		assertFailureEnvelope(t, stdout, code)
	})

	t.Run("skill", func(t *testing.T) {
		stdout, _, code := runRootCmd(t, "skill", "determine-feature-slug")
		assertSuccessEnvelope(t, stdout, code)

		stdout, _, code = runRootCmd(t, "skill", "does-not-exist")
		assertFailureEnvelope(t, stdout, code)
	})

	t.Run("init", func(t *testing.T) {
		// init's success path is a deliberate plain-text exception like file
		// read's (see cmd/init.go's runInit — it fmt.Fprintf's human-readable
		// lines, not the JSON envelope); its exit code is covered by
		// TestWrapper_SuccessAndFailureBothStreamOnStdoutOnly. Only the
		// failure path is envelope-shaped, so only it is asserted here.
		dir := t.TempDir()
		t.Chdir(dir)

		stdout, _, code := runRootCmd(t, "init", "does-not-exist")
		assertFailureEnvelope(t, stdout, code)
	})
}

// Criterion 3: a failure is printed exactly once on stdout, with no
// additional cobra-generated usage/help text anywhere in stdout or stderr —
// guards against the old double-print bug where cobra's own default error
// printer ran before the tool's own fallback.
func TestWrapper_FailureIsPrintedExactlyOnceWithNoCobraBoilerplate(t *testing.T) {
	assertNoCobraBoilerplate := func(t *testing.T, stdout, stderr string, code int) {
		t.Helper()
		require.Equal(t, 1, code)
		require.Empty(t, stderr, "cobra's default error/usage printer must be silenced")
		require.Equal(t, 1, strings.Count(stdout, `"error": true`), "the failure envelope must appear exactly once")
		require.NotContains(t, stdout, "Usage:")
		require.NotContains(t, stdout, "Error:")
	}

	t.Run("spec", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		resetSpecCommandFlags(t)
		stdout, stderr, code := runRootCmd(t, "spec", "new", "--data", `{"name":" billing"}`)
		assertNoCobraBoilerplate(t, stdout, stderr, code)
	})

	t.Run("plan", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		resetPlanCommandFlags(t)
		stdout, stderr, code := runRootCmd(t, "plan", "new")
		assertNoCobraBoilerplate(t, stdout, stderr, code)
	})

	t.Run("implement", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".spektacular"), 0o755))
		resetImplementCommandFlags(t)
		stdout, stderr, code := runRootCmd(t, "implement", "new", "--data", `{"name":"nosuch"}`)
		assertNoCobraBoilerplate(t, stdout, stderr, code)
	})

	t.Run("knowledge", func(t *testing.T) {
		twoScopeProject(t)
		stdout, stderr, err := runKnowledge(t, "read")
		require.Error(t, err)
		assertNoCobraBoilerplate(t, stdout, stderr, 1)
	})

	t.Run("file", func(t *testing.T) {
		writeSpecFileFixture(t)
		stdout, stderr, code := runRootCmd(t, "spec", "file", "read", "missing.md")
		assertNoCobraBoilerplate(t, stdout, stderr, code)
	})

	t.Run("skill", func(t *testing.T) {
		stdout, stderr, code := runRootCmd(t, "skill", "does-not-exist")
		assertNoCobraBoilerplate(t, stdout, stderr, code)
	})

	t.Run("init", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		stdout, stderr, code := runRootCmd(t, "init", "does-not-exist")
		assertNoCobraBoilerplate(t, stdout, stderr, code)
	})

	// An unknown top-level command/flag never reaches any RunE — cobra itself
	// rejects it — so this is the one case where runRoot's fallback
	// (toErrorResponse's internal_error branch) is what stops cobra's default
	// printer from producing a second, differently-shaped failure message.
	t.Run("unknown top-level command", func(t *testing.T) {
		stdout, stderr, code := runRootCmd(t, "no-such-command")
		assertNoCobraBoilerplate(t, stdout, stderr, code)
	})
}
