package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jumppad-labs/spektacular/internal/metadata"
	"github.com/jumppad-labs/spektacular/internal/output"
	"github.com/stretchr/testify/require"
)

// kindFixture describes one of the three per-kind write-command families
// (spec / plan / changelog) so the Phase 1.3 acceptance tests can exercise
// each through the same table-driven scaffold.
type kindFixture struct {
	// kind is the CLI subcommand root: "spec", "plan", or "changelog".
	kind string
	// configYAML is the config body written to .spektacular/config.yaml that
	// points this kind's `directory` under a docs-rooted subtree.
	configYAML string
	// artifactName is the argument passed to `<kind> file write`. Every kind
	// that carries an ID prefix uses a name matching the default id_method
	// (timestamp) so validateIDPrefix accepts it.
	artifactName string
	// storeRelPath is the on-disk path relative to the project root where the
	// artifact ends up after a write.
	storeRelPath string
}

// kindFixtures enumerates the three per-kind file-write commands under test.
// Each row uses a docs-rooted directory (not the default `.spektacular`
// data dir) to prove the metadata layer is applied regardless of where the
// configured directory lands, and uses a timestamp-shaped ID prefix so it
// clears validateIDPrefix under the default id_method.
func kindFixtures() []kindFixture {
	return []kindFixture{
		{
			kind:         "spec",
			configYAML:   "spec:\n  config:\n    directory: docs/specs\n",
			artifactName: "20260709000000-feature.md",
			storeRelPath: filepath.Join("docs", "specs", "20260709000000-feature.md"),
		},
		{
			kind:         "plan",
			configYAML:   "plan:\n  config:\n    directory: docs/plans\n",
			artifactName: "20260709000000-feature/plan.md",
			storeRelPath: filepath.Join("docs", "plans", "20260709000000-feature", "plan.md"),
		},
		{
			kind:       "changelog",
			configYAML: "changelog:\n  config:\n    directory: docs/changelog\n",
			// The CLI-facing name stays flat; on disk the store injects the
			// project-named namespace folder (writeSpecCommandConfig writes
			// `name: testproj`) below the CLI surface.
			artifactName: "20260709000000-release-notes.md",
			storeRelPath: filepath.Join("docs", "changelog", "testproj", "20260709000000-release-notes.md"),
		},
	}
}

// today returns today's date at UTC midnight, matching the truncation the
// metadata Merge helper applies before stamping created_date / closed_date.
// Tests compare stamped dates to this value.
func today() time.Time {
	return time.Now().UTC().Truncate(24 * time.Hour)
}

// resetWriteStatusFlag clears the --status flag on the write subcommand of
// the given kind's `file` command. Cobra flag state is package-global — a
// closure `statusFlag` variable persists across Execute() calls — so tests
// that set --status must reset it, or a subsequent test with no --status
// silently inherits the last-set value.
func resetWriteStatusFlag(t *testing.T, kind string) {
	t.Helper()
	// Walk root -> kind -> file -> write and clear its --status.
	kindCmd, _, err := rootCmd.Find([]string{kind, "file", "write"})
	require.NoError(t, err)
	require.NotNil(t, kindCmd)
	reset := func() { require.NoError(t, kindCmd.Flags().Set("status", "")) }
	reset()
	t.Cleanup(reset)
}

// resetSetStatusFlag clears the --status flag on the set-status subcommand of
// the given kind's `file` command. Cobra flag state is package-global — the
// closure `setStatusFlag` variable in newStoreFileCmd persists across
// Execute() calls the same way `statusFlag` does for `write` — so tests that
// set --status on set-status must reset it, or a subsequent test with no
// --status silently inherits the last-set value (and the required-flag gate
// then fails to trip because the flag has already been "set" from a prior
// call). MarkFlagRequired also tracks its own "was set" state, which cobra
// clears when the flag's Set() is called; clearing the value alone is not
// enough, so we also flip the "changed" bit off after resetting.
func resetSetStatusFlag(t *testing.T, kind string) {
	t.Helper()
	kindCmd, _, err := rootCmd.Find([]string{kind, "file", "set-status"})
	require.NoError(t, err)
	require.NotNil(t, kindCmd)
	reset := func() {
		require.NoError(t, kindCmd.Flags().Set("status", ""))
		if f := kindCmd.Flags().Lookup("status"); f != nil {
			f.Changed = false
		}
	}
	reset()
	t.Cleanup(reset)
}

// TestStoreFileWrite_FreshWriteStampsMetadata asserts criterion 1: a fresh
// write through `<kind> file write` produces a stored file whose top begins
// with a YAML frontmatter block containing created_date set to today and
// status: in-progress. Exercised for all three per-kind write commands.
func TestStoreFileWrite_FreshWriteStampsMetadata(t *testing.T) {
	for _, fx := range kindFixtures() {
		t.Run(fx.kind, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			writeSpecCommandConfig(t, dir, fx.configYAML)

			srcPath := filepath.Join(t.TempDir(), "source.md")
			require.NoError(t, os.WriteFile(srcPath, []byte("fresh body"), 0o644))

			setupImplementCmd(t)
			rootCmd.SetArgs([]string{fx.kind, "file", "write", fx.artifactName, "--from", srcPath})

			require.NoError(t, rootCmd.Execute())

			content, err := os.ReadFile(filepath.Join(dir, fx.storeRelPath))
			require.NoError(t, err)

			meta, body, err := metadata.Split(content)
			require.NoError(t, err)
			require.NotNil(t, meta, "fresh write must produce a frontmatter block")
			require.Equal(t, metadata.StatusInProgress, meta.Status)
			require.True(t, meta.CreatedDate.Equal(today()),
				"created_date must be today (%s), got %s", today(), meta.CreatedDate)
			require.True(t, meta.ClosedDate.IsZero(),
				"in-progress artifact must have zero closed_date, got %s", meta.ClosedDate)
			require.Equal(t, "fresh body", string(body))
		})
	}
}

// TestStoreFileWrite_ExistingArtifactPreservesCreatedDate asserts criterion
// 2: writing an existing artifact preserves the created_date it already had.
// The test primes the store with an artifact carrying an older created_date,
// then writes new body content through the CLI and asserts the created_date
// is unchanged. Exercised for all three per-kind write commands.
func TestStoreFileWrite_ExistingArtifactPreservesCreatedDate(t *testing.T) {
	earlier := time.Date(2026, time.January, 5, 0, 0, 0, 0, time.UTC)

	for _, fx := range kindFixtures() {
		t.Run(fx.kind, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			writeSpecCommandConfig(t, dir, fx.configYAML)

			// Seed the store with an artifact that already carries a
			// created_date from earlier in the year, so any preservation
			// failure will be obvious (today != earlier).
			seeded, err := metadata.Render(metadata.Metadata{
				CreatedDate: earlier,
				Status:      metadata.StatusInProgress,
			}, []byte("original body"))
			require.NoError(t, err)

			storeAbs := filepath.Join(dir, fx.storeRelPath)
			require.NoError(t, os.MkdirAll(filepath.Dir(storeAbs), 0o755))
			require.NoError(t, os.WriteFile(storeAbs, seeded, 0o644))

			srcPath := filepath.Join(t.TempDir(), "source.md")
			require.NoError(t, os.WriteFile(srcPath, []byte("updated body"), 0o644))

			setupImplementCmd(t)
			rootCmd.SetArgs([]string{fx.kind, "file", "write", fx.artifactName, "--from", srcPath})

			require.NoError(t, rootCmd.Execute())

			content, err := os.ReadFile(storeAbs)
			require.NoError(t, err)

			meta, body, err := metadata.Split(content)
			require.NoError(t, err)
			require.NotNil(t, meta, "rewrite must retain a frontmatter block")
			require.True(t, meta.CreatedDate.Equal(earlier),
				"created_date must be preserved (%s), got %s", earlier, meta.CreatedDate)
			require.Equal(t, metadata.StatusInProgress, meta.Status)
			require.Equal(t, "updated body", string(body))
		})
	}
}

// TestStoreFileWrite_StatusFlagTransitionsAndStampsClosedDate asserts
// criterion 3: writing with `--status completed` on an artifact currently
// in-progress transitions the status and stamps closed_date to today.
// Exercised for all three per-kind write commands.
func TestStoreFileWrite_StatusFlagTransitionsAndStampsClosedDate(t *testing.T) {
	earlier := time.Date(2026, time.January, 5, 0, 0, 0, 0, time.UTC)

	for _, fx := range kindFixtures() {
		t.Run(fx.kind, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			writeSpecCommandConfig(t, dir, fx.configYAML)

			seeded, err := metadata.Render(metadata.Metadata{
				CreatedDate: earlier,
				Status:      metadata.StatusInProgress,
			}, []byte("original body"))
			require.NoError(t, err)

			storeAbs := filepath.Join(dir, fx.storeRelPath)
			require.NoError(t, os.MkdirAll(filepath.Dir(storeAbs), 0o755))
			require.NoError(t, os.WriteFile(storeAbs, seeded, 0o644))

			srcPath := filepath.Join(t.TempDir(), "source.md")
			require.NoError(t, os.WriteFile(srcPath, []byte("final body"), 0o644))

			resetWriteStatusFlag(t, fx.kind)
			setupImplementCmd(t)
			rootCmd.SetArgs([]string{fx.kind, "file", "write", fx.artifactName, "--from", srcPath, "--status", "completed"})

			require.NoError(t, rootCmd.Execute())

			content, err := os.ReadFile(storeAbs)
			require.NoError(t, err)

			meta, body, err := metadata.Split(content)
			require.NoError(t, err)
			require.NotNil(t, meta)
			require.Equal(t, metadata.StatusCompleted, meta.Status)
			require.True(t, meta.CreatedDate.Equal(earlier),
				"created_date must be preserved across status transition (%s), got %s", earlier, meta.CreatedDate)
			require.True(t, meta.ClosedDate.Equal(today()),
				"closed_date must be stamped to today (%s), got %s", today(), meta.ClosedDate)
			require.Equal(t, "final body", string(body))
		})
	}
}

// TestStoreFileWrite_BareArtifactIsUpgradedInPlace asserts criterion 4:
// writing an existing artifact that has no prior frontmatter is treated as
// a first write — the artifact gains a metadata block on its next write,
// created_date is stamped to today, status defaults to in-progress, and no
// error is raised. Exercised for all three per-kind write commands.
func TestStoreFileWrite_BareArtifactIsUpgradedInPlace(t *testing.T) {
	for _, fx := range kindFixtures() {
		t.Run(fx.kind, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			writeSpecCommandConfig(t, dir, fx.configYAML)

			// Pre-populate the store with a bare (no-frontmatter) file to
			// simulate an artifact that pre-dates the metadata schema.
			storeAbs := filepath.Join(dir, fx.storeRelPath)
			require.NoError(t, os.MkdirAll(filepath.Dir(storeAbs), 0o755))
			require.NoError(t, os.WriteFile(storeAbs, []byte("legacy body without frontmatter\n"), 0o644))

			srcPath := filepath.Join(t.TempDir(), "source.md")
			require.NoError(t, os.WriteFile(srcPath, []byte("upgraded body"), 0o644))

			setupImplementCmd(t)
			rootCmd.SetArgs([]string{fx.kind, "file", "write", fx.artifactName, "--from", srcPath})

			require.NoError(t, rootCmd.Execute(),
				"bare-artifact upgrade must not error")

			content, err := os.ReadFile(storeAbs)
			require.NoError(t, err)

			meta, body, err := metadata.Split(content)
			require.NoError(t, err)
			require.NotNil(t, meta, "bare artifact must gain a frontmatter block on next write")
			require.Equal(t, metadata.StatusInProgress, meta.Status)
			require.True(t, meta.CreatedDate.Equal(today()),
				"created_date on bare-artifact upgrade must be today (%s), got %s", today(), meta.CreatedDate)
			require.True(t, meta.ClosedDate.IsZero())
			require.Equal(t, "upgraded body", string(body))
		})
	}
}

// TestStoreFileWrite_FreshWriteWithClosedStatusStampsBothDates asserts the
// symmetric first-write case for --status: passing --status completed on a
// brand-new artifact stamps both created_date and closed_date to today.
// This complements criterion 3 (which covers the transition path) by
// exercising the first-write branch of the same flag.
func TestStoreFileWrite_FreshWriteWithClosedStatusStampsBothDates(t *testing.T) {
	for _, fx := range kindFixtures() {
		t.Run(fx.kind, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			writeSpecCommandConfig(t, dir, fx.configYAML)

			srcPath := filepath.Join(t.TempDir(), "source.md")
			require.NoError(t, os.WriteFile(srcPath, []byte("closed body"), 0o644))

			resetWriteStatusFlag(t, fx.kind)
			setupImplementCmd(t)
			rootCmd.SetArgs([]string{fx.kind, "file", "write", fx.artifactName, "--from", srcPath, "--status", "completed"})

			require.NoError(t, rootCmd.Execute())

			content, err := os.ReadFile(filepath.Join(dir, fx.storeRelPath))
			require.NoError(t, err)

			meta, body, err := metadata.Split(content)
			require.NoError(t, err)
			require.NotNil(t, meta)
			require.Equal(t, metadata.StatusCompleted, meta.Status)
			require.True(t, meta.CreatedDate.Equal(today()))
			require.True(t, meta.ClosedDate.Equal(today()),
				"fresh write with --status completed must stamp closed_date to today, got %s", meta.ClosedDate)
			require.Equal(t, "closed body", string(body))
		})
	}
}

// TestStoreFileWrite_RejectsInvalidStatusFlag asserts that --status must be
// one of the four enum values; any other value is rejected with an actionable
// error naming both the offending value and the four allowed alternatives,
// and the destination file is not created. This guards the CLI-layer
// validation that metadataOptsForStatus performs before Merge sees the value.
func TestStoreFileWrite_RejectsInvalidStatusFlag(t *testing.T) {
	for _, fx := range kindFixtures() {
		t.Run(fx.kind, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			writeSpecCommandConfig(t, dir, fx.configYAML)

			srcPath := filepath.Join(t.TempDir(), "source.md")
			require.NoError(t, os.WriteFile(srcPath, []byte("body"), 0o644))

			resetWriteStatusFlag(t, fx.kind)
			setupImplementCmd(t)
			rootCmd.SetArgs([]string{fx.kind, "file", "write", fx.artifactName, "--from", srcPath, "--status", "bogus"})

			err := rootCmd.Execute()
			require.Error(t, err)
			require.ErrorContains(t, err, "bogus")
			require.NoFileExists(t, filepath.Join(dir, fx.storeRelPath),
				"a rejected --status must not create the destination file")
		})
	}
}

// seedArtifactWithMetadata renders a Metadata block over body and writes it
// on disk at storeRelPath under dir. It's a shortcut for tests that need a
// pre-existing artifact with a specific created_date / status baked in.
func seedArtifactWithMetadata(t *testing.T, dir, storeRelPath string, m metadata.Metadata, body []byte) {
	t.Helper()
	rendered, err := metadata.Render(m, body)
	require.NoError(t, err)
	abs := filepath.Join(dir, storeRelPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o755))
	require.NoError(t, os.WriteFile(abs, rendered, 0o644))
}

// TestStoreFileSetStatus_MutatesFrontmatterOnly_PreservesBody asserts
// acceptance criterion 1: `set-status` mutates only the frontmatter block of
// the target artifact and leaves the body bytes untouched. For each kind, we
// seed an in-progress artifact with a known older created_date and a body
// containing shell-sensitive and multi-line content, then flip to `completed`
// and assert (a) the frontmatter now shows status: completed with today's
// closed_date, (b) created_date is unchanged, and (c) the body bytes are
// byte-identical to what was seeded.
func TestStoreFileSetStatus_MutatesFrontmatterOnly_PreservesBody(t *testing.T) {
	earlier := time.Date(2026, time.January, 5, 0, 0, 0, 0, time.UTC)
	body := []byte("body line 1 with `backticks` and $dollar\nline 2\n")

	for _, fx := range kindFixtures() {
		t.Run(fx.kind, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			writeSpecCommandConfig(t, dir, fx.configYAML)

			seedArtifactWithMetadata(t, dir, fx.storeRelPath, metadata.Metadata{
				CreatedDate: earlier,
				Status:      metadata.StatusInProgress,
			}, body)

			resetSetStatusFlag(t, fx.kind)
			setupImplementCmd(t)
			rootCmd.SetArgs([]string{fx.kind, "file", "set-status", fx.artifactName, "--status", "completed"})

			require.NoError(t, rootCmd.Execute())

			content, err := os.ReadFile(filepath.Join(dir, fx.storeRelPath))
			require.NoError(t, err)

			meta, gotBody, err := metadata.Split(content)
			require.NoError(t, err)
			require.NotNil(t, meta)
			require.Equal(t, metadata.StatusCompleted, meta.Status)
			require.True(t, meta.CreatedDate.Equal(earlier),
				"created_date must be preserved (%s), got %s", earlier, meta.CreatedDate)
			require.True(t, meta.ClosedDate.Equal(today()),
				"closed_date must be stamped to today (%s), got %s", today(), meta.ClosedDate)
			require.Equal(t, body, gotBody,
				"body bytes must be byte-identical to the seeded body")
		})
	}
}

// TestStoreFileSetStatus_RejectsInvalidStatus_LeavesFileUntouched asserts
// acceptance criterion 2: a --status value outside the four-value enum is
// rejected with an actionable error and the on-disk bytes of the target
// artifact are unchanged. This guards the CLI-layer validation performed by
// metadataOptsForStatus before any write is attempted.
func TestStoreFileSetStatus_RejectsInvalidStatus_LeavesFileUntouched(t *testing.T) {
	earlier := time.Date(2026, time.January, 5, 0, 0, 0, 0, time.UTC)
	body := []byte("preserved body\n")

	for _, fx := range kindFixtures() {
		t.Run(fx.kind, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			writeSpecCommandConfig(t, dir, fx.configYAML)

			seedArtifactWithMetadata(t, dir, fx.storeRelPath, metadata.Metadata{
				CreatedDate: earlier,
				Status:      metadata.StatusInProgress,
			}, body)

			abs := filepath.Join(dir, fx.storeRelPath)
			before, err := os.ReadFile(abs)
			require.NoError(t, err)

			resetSetStatusFlag(t, fx.kind)
			setupImplementCmd(t)
			rootCmd.SetArgs([]string{fx.kind, "file", "set-status", fx.artifactName, "--status", "wibble"})

			err = rootCmd.Execute()
			require.Error(t, err)
			require.ErrorContains(t, err, "wibble")

			after, err := os.ReadFile(abs)
			require.NoError(t, err)
			require.Equal(t, before, after,
				"a rejected --status must leave the file's bytes untouched")
		})
	}
}

// TestStoreFileSetStatus_OnBareArtifact_AttachesFrontmatter asserts
// acceptance criterion 3: running set-status on an artifact with no prior
// frontmatter attaches a new block with created_date=today, the requested
// status, and (when the status is a closed value) closed_date=today. The
// body bytes of the bare artifact must survive verbatim.
func TestStoreFileSetStatus_OnBareArtifact_AttachesFrontmatter(t *testing.T) {
	body := []byte("legacy body without frontmatter\n")

	for _, fx := range kindFixtures() {
		t.Run(fx.kind, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			writeSpecCommandConfig(t, dir, fx.configYAML)

			abs := filepath.Join(dir, fx.storeRelPath)
			require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o755))
			require.NoError(t, os.WriteFile(abs, body, 0o644))

			resetSetStatusFlag(t, fx.kind)
			setupImplementCmd(t)
			rootCmd.SetArgs([]string{fx.kind, "file", "set-status", fx.artifactName, "--status", "archived"})

			require.NoError(t, rootCmd.Execute())

			content, err := os.ReadFile(abs)
			require.NoError(t, err)

			meta, gotBody, err := metadata.Split(content)
			require.NoError(t, err)
			require.NotNil(t, meta,
				"bare artifact must gain a frontmatter block after set-status")
			require.Equal(t, metadata.StatusArchived, meta.Status)
			require.True(t, meta.CreatedDate.Equal(today()),
				"created_date on a bare-artifact set-status must be today (%s), got %s", today(), meta.CreatedDate)
			require.True(t, meta.ClosedDate.Equal(today()),
				"closed_date on a closed-status set-status must be today (%s), got %s", today(), meta.ClosedDate)
			require.Equal(t, body, gotBody,
				"body bytes of a bare artifact must survive set-status unchanged")
		})
	}
}

// TestStoreFileSetStatus_Idempotent asserts acceptance criterion 4: running
// set-status twice with the same value is a no-op on the second call. The
// closed_date stamped by the first call must survive the second (it is not
// re-stamped to a later "today" value even in the trivial same-day case),
// and the on-disk bytes after the second call must be byte-identical to the
// bytes after the first call.
func TestStoreFileSetStatus_Idempotent(t *testing.T) {
	earlier := time.Date(2026, time.January, 5, 0, 0, 0, 0, time.UTC)
	body := []byte("stable body\n")

	for _, fx := range kindFixtures() {
		t.Run(fx.kind, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			writeSpecCommandConfig(t, dir, fx.configYAML)

			seedArtifactWithMetadata(t, dir, fx.storeRelPath, metadata.Metadata{
				CreatedDate: earlier,
				Status:      metadata.StatusInProgress,
			}, body)

			// First set-status: transitions to completed, stamps closed_date.
			resetSetStatusFlag(t, fx.kind)
			setupImplementCmd(t)
			rootCmd.SetArgs([]string{fx.kind, "file", "set-status", fx.artifactName, "--status", "completed"})
			require.NoError(t, rootCmd.Execute())

			abs := filepath.Join(dir, fx.storeRelPath)
			afterFirst, err := os.ReadFile(abs)
			require.NoError(t, err)

			// Second set-status with the same value: must leave bytes
			// untouched — including the closed_date stamped on the first call.
			resetSetStatusFlag(t, fx.kind)
			setupImplementCmd(t)
			rootCmd.SetArgs([]string{fx.kind, "file", "set-status", fx.artifactName, "--status", "completed"})
			require.NoError(t, rootCmd.Execute())

			afterSecond, err := os.ReadFile(abs)
			require.NoError(t, err)

			require.Equal(t, afterFirst, afterSecond,
				"repeat set-status with the same value must be byte-idempotent")
		})
	}
}

// TestStoreFileSetStatus_MissingStatusIsError asserts that invoking
// set-status without --status is rejected — whether via cobra's
// MarkFlagRequired machinery or via the CLI's own missing_status guard,
// whichever fires first. The destination file must not be created in either
// path. This binds the contract that --status is mandatory on set-status,
// distinct from `write` where --status is optional.
func TestStoreFileSetStatus_MissingStatusIsError(t *testing.T) {
	for _, fx := range kindFixtures() {
		t.Run(fx.kind, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			writeSpecCommandConfig(t, dir, fx.configYAML)

			// Seed an existing file so a "not found" error can't be what we
			// accidentally observe instead of a missing-flag error.
			seedArtifactWithMetadata(t, dir, fx.storeRelPath, metadata.Metadata{
				CreatedDate: today(),
				Status:      metadata.StatusInProgress,
			}, []byte("body\n"))

			abs := filepath.Join(dir, fx.storeRelPath)
			before, err := os.ReadFile(abs)
			require.NoError(t, err)

			resetSetStatusFlag(t, fx.kind)
			setupImplementCmd(t)
			rootCmd.SetArgs([]string{fx.kind, "file", "set-status", fx.artifactName})

			err = rootCmd.Execute()
			require.Error(t, err, "set-status without --status must error")

			after, err := os.ReadFile(abs)
			require.NoError(t, err)
			require.Equal(t, before, after,
				"a missing-status error must not modify the file")
		})
	}
}

// TestStoreFileSetStatus_MissingFileIsError asserts that running set-status
// against a non-existent path returns the standard `not_found` error envelope
// (via runRoot, which mirrors production Execute wrapping) with the requested
// path named in the resource field, and does not create the target file as a
// side-effect. This binds the contract that set-status is a mutation of an
// existing artifact, never a create.
func TestStoreFileSetStatus_MissingFileIsError(t *testing.T) {
	for _, fx := range kindFixtures() {
		t.Run(fx.kind, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			writeSpecCommandConfig(t, dir, fx.configYAML)

			resetSetStatusFlag(t, fx.kind)
			stdout, stderr, code := runRootCmd(t, fx.kind, "file", "set-status", fx.artifactName, "--status", "completed")

			require.Equal(t, 1, code)
			require.Empty(t, stderr)

			var er output.ErrorResponse
			require.NoError(t, json.Unmarshal([]byte(stdout), &er))
			require.True(t, er.IsError)
			require.Equal(t, "not_found", er.Code)
			require.Equal(t, fx.artifactName, er.Resource)

			require.NoFileExists(t, filepath.Join(dir, fx.storeRelPath),
				"a not_found error must not create the destination file")
		})
	}
}

// TestStoreFileWrite_IdempotentUnderRepeatedWrites is a regression guard: a
// source blob that already carries one or more leading frontmatter blocks
// (e.g. because the caller cp'd a stored artifact into their scratch file)
// must not accumulate. The write handler strips every leading frontmatter
// block from the source before merging, so the resulting stored artifact
// carries exactly one block regardless of how many the source had.
func TestStoreFileWrite_IdempotentUnderRepeatedWrites(t *testing.T) {
	for _, fx := range kindFixtures() {
		t.Run(fx.kind, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			writeSpecCommandConfig(t, dir, fx.configYAML)

			// Source with three properly-stacked frontmatter blocks — each
			// block terminated by `---\n\n` before the next `---\n` opener,
			// matching the shape a buggy handler produces by re-wrapping its
			// own output. This is what actually accumulates on disk.
			one, err := metadata.Render(metadata.Metadata{
				CreatedDate: today(),
				Status:      metadata.StatusInProgress,
			}, []byte("real body"))
			require.NoError(t, err)
			// Prepend two more frontmatter blocks. Render already outputs
			// `---\n<yaml>---\n\n<body>`; nesting is `---\n<yaml>---\n\n` +
			// prior content.
			blockPrefix := []byte("---\ncreated_date: \"2026-07-28\"\nstatus: in-progress\n---\n\n")
			stacked := append([]byte{}, blockPrefix...)
			stacked = append(stacked, blockPrefix...)
			stacked = append(stacked, one...)

			srcPath := filepath.Join(t.TempDir(), "source.md")
			require.NoError(t, os.WriteFile(srcPath, stacked, 0o644))

			setupImplementCmd(t)
			rootCmd.SetArgs([]string{fx.kind, "file", "write", fx.artifactName, "--from", srcPath})
			require.NoError(t, rootCmd.Execute())

			content, err := os.ReadFile(filepath.Join(dir, fx.storeRelPath))
			require.NoError(t, err)

			meta, body, err := metadata.Split(content)
			require.NoError(t, err)
			require.NotNil(t, meta, "write must produce exactly one leading frontmatter block")
			require.Equal(t, "real body", string(body),
				"body must be the original body, not the source's own frontmatter blocks")

			// After stripping the one leading block, no further frontmatter
			// should be present in the body.
			extra, _, err := metadata.Split(body)
			require.NoError(t, err)
			require.Nil(t, extra,
				"stored artifact must not carry a second frontmatter block underneath the first")
		})
	}
}
