package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/jumppad-labs/spektacular/internal/metadata"
	"github.com/stretchr/testify/require"
)

// listFlagNames is the set of `<kind> file list` filter flags that carry
// cobra-package-global closure state. Any test that sets one of these — or
// even reads the list command in a way that could cache Changed=true — must
// reset them between runs, or the next runRoot invocation inherits the
// stale value.
var listFlagNames = []string{"status", "created-after", "created-before", "closed-after", "closed-before"}

// resetListFilterFlags clears every `<kind> file list` filter flag on the given
// kind's list subcommand — the same package-global closure trap that
// resetWriteStatusFlag / resetSetStatusFlag guard for the write and set-status
// subcommands. Every integration test in this file that runs a list command
// (filtered or not) must call this to isolate itself from prior runs.
func resetListFilterFlags(t *testing.T, kind string) {
	t.Helper()
	kindCmd, _, err := rootCmd.Find([]string{kind, "file", "list"})
	require.NoError(t, err)
	require.NotNil(t, kindCmd)
	reset := func() {
		for _, name := range listFlagNames {
			require.NoError(t, kindCmd.Flags().Set(name, ""))
			if f := kindCmd.Flags().Lookup(name); f != nil {
				f.Changed = false
			}
		}
	}
	reset()
	t.Cleanup(reset)
}

// listResponse is the shape the `<kind> file list` command writes to stdout in
// Phase 2.1: `{"files": [ {name, path, ...metadata}, ... ], "error": false }`.
// Only files carry metadata fields; directory entries carry only name and path.
type listResponse struct {
	Files []map[string]any `json:"files"`
	Error bool             `json:"error"`
}

// runListJSON runs `<kind> file list [args...]` via runRoot (mirroring
// production wrapping) and decodes the response envelope into a listResponse.
// It fails the test if the command returns a non-zero exit code, so callers
// can focus on asserting the shape and contents of the returned files list.
func runListJSON(t *testing.T, kind string, args ...string) listResponse {
	t.Helper()
	stdout, stderr, code := runRootCmd(t, append([]string{kind, "file", "list"}, args...)...)
	require.Equalf(t, 0, code, "list command failed: stdout=%s stderr=%s", stdout, stderr)
	require.Empty(t, stderr)

	var resp listResponse
	require.NoError(t, json.Unmarshal([]byte(stdout), &resp))
	require.False(t, resp.Error)
	return resp
}

// listFilterFixture describes one of the three per-kind file-list command
// families for the Phase 2.1 acceptance tests. Each row uses a docs-rooted
// directory (not the default .spektacular data dir) and a listPath the tests
// pass to `<kind> file list` — for spec/changelog that's "" (list the store
// root, where the artifacts sit directly), for plan it's the plan
// subdirectory so metadata-bearing files inside are listed instead of the
// per-plan directory entries at the top level.
type listFilterFixture struct {
	kind       string
	configYAML string
	// artifactPath is a function that returns the on-disk store-relative path
	// for a given filename, so plan's per-plan subdirectory nesting can be
	// expressed once and reused for every seed.
	artifactPath func(name string) string
	// listPath is passed as the positional argument to `<kind> file list`
	// (empty means the store root).
	listPath string
}

// listFilterFixtures enumerates the three per-kind list-filter command
// surfaces. Spec and changelog list directly under their configured
// directory; plan lists inside a specific plan subdirectory to exercise
// metadata filtering on the plan.md / context.md files inside a plan folder.
func listFilterFixtures() []listFilterFixture {
	return []listFilterFixture{
		{
			kind:         "spec",
			configYAML:   "spec:\n  config:\n    directory: docs/specs\n",
			artifactPath: func(name string) string { return filepath.Join("docs", "specs", name) },
			listPath:     "",
		},
		{
			kind:         "changelog",
			configYAML:   "changelog:\n  config:\n    directory: docs/changelog\n",
			artifactPath: func(name string) string { return filepath.Join("docs", "changelog", name) },
			listPath:     "",
		},
		{
			kind:       "plan",
			configYAML: "plan:\n  config:\n    directory: docs/plans\n",
			// Seed each artifact into the same plan subdirectory so a
			// non-empty listPath returns them all as file entries with
			// metadata rather than a single directory entry at the top level.
			artifactPath: func(name string) string {
				return filepath.Join("docs", "plans", "20260709000000-feature", name)
			},
			listPath: "20260709000000-feature",
		},
	}
}

// findFile returns the entry in files whose name matches, or nil.
func findFile(files []map[string]any, name string) map[string]any {
	for _, f := range files {
		if f["name"] == name {
			return f
		}
	}
	return nil
}

// fileNames returns the sorted names of every entry in files, for stable set
// comparisons in the tests below.
func fileNames(files []map[string]any) []string {
	names := make([]string, 0, len(files))
	for _, f := range files {
		if n, ok := f["name"].(string); ok {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	return names
}

// TestStoreFileList_UnfilteredReturnsAllWithMetadataFields asserts acceptance
// criterion 5 (and half of 6): unfiltered `<kind> file list` returns the same
// set of artifacts as before, plus the new metadata fields on entries whose
// stored bytes carry a frontmatter block. A bare (pre-shipping) artifact is
// included, but carries only name and path — no metadata fields.
func TestStoreFileList_UnfilteredReturnsAllWithMetadataFields(t *testing.T) {
	inProgressCreated := time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC)
	completedCreated := time.Date(2026, time.January, 20, 0, 0, 0, 0, time.UTC)
	completedClosed := time.Date(2026, time.February, 5, 0, 0, 0, 0, time.UTC)

	for _, fx := range listFilterFixtures() {
		t.Run(fx.kind, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			writeSpecCommandConfig(t, dir, fx.configYAML)

			seedArtifactWithMetadata(t, dir, fx.artifactPath("in-progress.md"), metadata.Metadata{
				CreatedDate: inProgressCreated,
				Status:      metadata.StatusInProgress,
			}, []byte("body in progress"))

			seedArtifactWithMetadata(t, dir, fx.artifactPath("completed.md"), metadata.Metadata{
				CreatedDate: completedCreated,
				Status:      metadata.StatusCompleted,
				ClosedDate:  completedClosed,
			}, []byte("body completed"))

			// Bare (pre-shipping) artifact: raw bytes, no frontmatter block.
			bareAbs := filepath.Join(dir, fx.artifactPath("bare.md"))
			require.NoError(t, os.MkdirAll(filepath.Dir(bareAbs), 0o755))
			require.NoError(t, os.WriteFile(bareAbs, []byte("bare legacy body"), 0o644))

			resetListFilterFlags(t, fx.kind)
			var resp listResponse
			if fx.listPath == "" {
				resp = runListJSON(t, fx.kind)
			} else {
				resp = runListJSON(t, fx.kind, fx.listPath)
			}

			require.ElementsMatch(t, []string{"bare.md", "completed.md", "in-progress.md"}, fileNames(resp.Files),
				"unfiltered list must include every seeded artifact, bare ones included")

			inProgress := findFile(resp.Files, "in-progress.md")
			require.NotNil(t, inProgress)
			require.Equal(t, "in-progress", inProgress["status"])
			require.Equal(t, "2026-01-10", inProgress["created_date"])
			_, hasClosed := inProgress["closed_date"]
			require.False(t, hasClosed, "an in-progress artifact must not carry a closed_date field")

			completed := findFile(resp.Files, "completed.md")
			require.NotNil(t, completed)
			require.Equal(t, "completed", completed["status"])
			require.Equal(t, "2026-01-20", completed["created_date"])
			require.Equal(t, "2026-02-05", completed["closed_date"])

			bare := findFile(resp.Files, "bare.md")
			require.NotNil(t, bare)
			_, hasStatus := bare["status"]
			require.False(t, hasStatus, "a bare artifact must not carry a status field")
			_, hasCreated := bare["created_date"]
			require.False(t, hasCreated, "a bare artifact must not carry a created_date field")
		})
	}
}

// TestStoreFileList_StatusFilterExcludesNonMatching asserts acceptance
// criterion 1 and the second half of criterion 6: `--status completed`
// returns only artifacts whose stored status is `completed`, and the bare
// (no-frontmatter) artifact is silently excluded because it can't satisfy
// any metadata predicate.
func TestStoreFileList_StatusFilterExcludesNonMatching(t *testing.T) {
	inProgressCreated := time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC)
	completedCreated := time.Date(2026, time.January, 20, 0, 0, 0, 0, time.UTC)
	completedClosed := time.Date(2026, time.February, 5, 0, 0, 0, 0, time.UTC)

	for _, fx := range listFilterFixtures() {
		t.Run(fx.kind, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			writeSpecCommandConfig(t, dir, fx.configYAML)

			seedArtifactWithMetadata(t, dir, fx.artifactPath("in-progress.md"), metadata.Metadata{
				CreatedDate: inProgressCreated,
				Status:      metadata.StatusInProgress,
			}, []byte("body in progress"))
			seedArtifactWithMetadata(t, dir, fx.artifactPath("completed.md"), metadata.Metadata{
				CreatedDate: completedCreated,
				Status:      metadata.StatusCompleted,
				ClosedDate:  completedClosed,
			}, []byte("body completed"))

			bareAbs := filepath.Join(dir, fx.artifactPath("bare.md"))
			require.NoError(t, os.MkdirAll(filepath.Dir(bareAbs), 0o755))
			require.NoError(t, os.WriteFile(bareAbs, []byte("bare legacy body"), 0o644))

			resetListFilterFlags(t, fx.kind)
			args := []string{"--status", "completed"}
			if fx.listPath != "" {
				args = append([]string{fx.listPath}, args...)
			}
			resp := runListJSON(t, fx.kind, args...)

			require.Equal(t, []string{"completed.md"}, fileNames(resp.Files),
				"a --status completed filter must return only the completed artifact and exclude the bare one")
			require.Equal(t, "completed", resp.Files[0]["status"])
		})
	}
}

// TestStoreFileList_CreatedDateRangeInclusive asserts acceptance criterion 2:
// `--created-after X --created-before Y` returns only artifacts whose
// created_date falls inclusively within the range. Uses X == Y so a matching
// artifact must be exactly on the boundary — the strongest form of the
// inclusivity claim.
func TestStoreFileList_CreatedDateRangeInclusive(t *testing.T) {
	jan1 := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	jan15 := time.Date(2026, time.January, 15, 0, 0, 0, 0, time.UTC)
	jan31 := time.Date(2026, time.January, 31, 0, 0, 0, 0, time.UTC)

	for _, fx := range listFilterFixtures() {
		t.Run(fx.kind, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			writeSpecCommandConfig(t, dir, fx.configYAML)

			seedArtifactWithMetadata(t, dir, fx.artifactPath("jan1.md"), metadata.Metadata{
				CreatedDate: jan1,
				Status:      metadata.StatusInProgress,
			}, []byte("jan1"))
			seedArtifactWithMetadata(t, dir, fx.artifactPath("jan15.md"), metadata.Metadata{
				CreatedDate: jan15,
				Status:      metadata.StatusInProgress,
			}, []byte("jan15"))
			seedArtifactWithMetadata(t, dir, fx.artifactPath("jan31.md"), metadata.Metadata{
				CreatedDate: jan31,
				Status:      metadata.StatusInProgress,
			}, []byte("jan31"))

			resetListFilterFlags(t, fx.kind)
			args := []string{"--created-after", "2026-01-15", "--created-before", "2026-01-15"}
			if fx.listPath != "" {
				args = append([]string{fx.listPath}, args...)
			}
			resp := runListJSON(t, fx.kind, args...)

			require.Equal(t, []string{"jan15.md"}, fileNames(resp.Files),
				"only the boundary-inclusive artifact must appear when after == before")
			require.Equal(t, "2026-01-15", resp.Files[0]["created_date"])
		})
	}
}

// TestStoreFileList_ClosedDateRangeExcludesInProgress asserts acceptance
// criterion 3: `--closed-after X` returns only artifacts whose closed_date
// falls at or after X. An in-progress artifact (no closed_date) must not be
// returned even though it "trivially" satisfies "> X" for a very old X.
func TestStoreFileList_ClosedDateRangeExcludesInProgress(t *testing.T) {
	created := time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC)
	completedClosed := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)

	for _, fx := range listFilterFixtures() {
		t.Run(fx.kind, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			writeSpecCommandConfig(t, dir, fx.configYAML)

			seedArtifactWithMetadata(t, dir, fx.artifactPath("open-a.md"), metadata.Metadata{
				CreatedDate: created,
				Status:      metadata.StatusInProgress,
			}, []byte("still open a"))
			seedArtifactWithMetadata(t, dir, fx.artifactPath("open-b.md"), metadata.Metadata{
				CreatedDate: created,
				Status:      metadata.StatusInProgress,
			}, []byte("still open b"))
			seedArtifactWithMetadata(t, dir, fx.artifactPath("closed.md"), metadata.Metadata{
				CreatedDate: created,
				Status:      metadata.StatusCompleted,
				ClosedDate:  completedClosed,
			}, []byte("closed"))

			resetListFilterFlags(t, fx.kind)
			args := []string{"--closed-after", "2026-01-01"}
			if fx.listPath != "" {
				args = append([]string{fx.listPath}, args...)
			}
			resp := runListJSON(t, fx.kind, args...)

			require.Equal(t, []string{"closed.md"}, fileNames(resp.Files),
				"only the artifact with a non-zero closed_date on or after the bound must appear")
			require.Equal(t, "2026-02-01", resp.Files[0]["closed_date"])
		})
	}
}

// TestStoreFileList_CombinedFiltersIntersect asserts acceptance criterion 4:
// combined filter flags AND together, not OR. Seeds four artifacts spanning
// (status × date) combinations and asserts `--status completed --created-after
// 2026-02-01` returns only the artifact that satisfies both — never the
// union of the two individual filter sets.
func TestStoreFileList_CombinedFiltersIntersect(t *testing.T) {
	jan15 := time.Date(2026, time.January, 15, 0, 0, 0, 0, time.UTC)
	feb15 := time.Date(2026, time.February, 15, 0, 0, 0, 0, time.UTC)
	closedDate := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)

	for _, fx := range listFilterFixtures() {
		t.Run(fx.kind, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			writeSpecCommandConfig(t, dir, fx.configYAML)

			// (status, created_date) matrix — only completed-feb matches.
			seedArtifactWithMetadata(t, dir, fx.artifactPath("in-progress-jan.md"), metadata.Metadata{
				CreatedDate: jan15,
				Status:      metadata.StatusInProgress,
			}, []byte("ip jan"))
			seedArtifactWithMetadata(t, dir, fx.artifactPath("in-progress-feb.md"), metadata.Metadata{
				CreatedDate: feb15,
				Status:      metadata.StatusInProgress,
			}, []byte("ip feb"))
			seedArtifactWithMetadata(t, dir, fx.artifactPath("completed-jan.md"), metadata.Metadata{
				CreatedDate: jan15,
				Status:      metadata.StatusCompleted,
				ClosedDate:  closedDate,
			}, []byte("done jan"))
			seedArtifactWithMetadata(t, dir, fx.artifactPath("completed-feb.md"), metadata.Metadata{
				CreatedDate: feb15,
				Status:      metadata.StatusCompleted,
				ClosedDate:  closedDate,
			}, []byte("done feb"))

			resetListFilterFlags(t, fx.kind)
			args := []string{"--status", "completed", "--created-after", "2026-02-01"}
			if fx.listPath != "" {
				args = append([]string{fx.listPath}, args...)
			}
			resp := runListJSON(t, fx.kind, args...)

			require.Equal(t, []string{"completed-feb.md"}, fileNames(resp.Files),
				"the intersection of --status completed AND --created-after 2026-02-01 must contain only completed-feb.md — not the union")
		})
	}
}

// TestStoreFileList_BareArtifactsExcludedFromAnyFilter asserts the second half
// of acceptance criterion 6: a bare (no-frontmatter) artifact appears in an
// unfiltered listing but is silently excluded from any listing that sets a
// metadata filter, because a bare artifact carries no metadata for the filter
// to match against. Runs unfiltered and filtered back-to-back within the same
// subtest — the reset helper isolates the two calls.
func TestStoreFileList_BareArtifactsExcludedFromAnyFilter(t *testing.T) {
	for _, fx := range listFilterFixtures() {
		t.Run(fx.kind, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			writeSpecCommandConfig(t, dir, fx.configYAML)

			seedArtifactWithMetadata(t, dir, fx.artifactPath("in-progress.md"), metadata.Metadata{
				CreatedDate: time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC),
				Status:      metadata.StatusInProgress,
			}, []byte("body"))

			bareAbs := filepath.Join(dir, fx.artifactPath("bare.md"))
			require.NoError(t, os.MkdirAll(filepath.Dir(bareAbs), 0o755))
			require.NoError(t, os.WriteFile(bareAbs, []byte("bare legacy body"), 0o644))

			// Unfiltered: both artifacts appear.
			resetListFilterFlags(t, fx.kind)
			var unfiltered listResponse
			if fx.listPath == "" {
				unfiltered = runListJSON(t, fx.kind)
			} else {
				unfiltered = runListJSON(t, fx.kind, fx.listPath)
			}
			require.ElementsMatch(t, []string{"bare.md", "in-progress.md"}, fileNames(unfiltered.Files),
				"unfiltered list must include the bare artifact")

			// Filtered: only the in-progress artifact appears; the bare one is silently excluded.
			resetListFilterFlags(t, fx.kind)
			args := []string{"--status", "in-progress"}
			if fx.listPath != "" {
				args = append([]string{fx.listPath}, args...)
			}
			filtered := runListJSON(t, fx.kind, args...)
			require.Equal(t, []string{"in-progress.md"}, fileNames(filtered.Files),
				"any metadata filter must exclude the bare artifact silently")
		})
	}
}

// TestStoreFileList_TopLevelPlanDirectoryEntriesCarryNameAndPathOnly is
// plan-specific coverage of the "directory entries become objects but carry
// only name and path" claim from the Phase 2.1 note. `plan file list` at the
// top of the plans directory yields one entry per plan sub-dir; those
// directory entries must not carry the metadata fields (created_date,
// status, closed_date) even though their contained plan.md does.
func TestStoreFileList_TopLevelPlanDirectoryEntriesCarryNameAndPathOnly(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeSpecCommandConfig(t, dir, "plan:\n  config:\n    directory: docs/plans\n")

	// Seed a plan.md inside a plan sub-directory carrying full metadata.
	seedArtifactWithMetadata(t,
		dir,
		filepath.Join("docs", "plans", "20260709000000-feature", "plan.md"),
		metadata.Metadata{
			CreatedDate: time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC),
			Status:      metadata.StatusInProgress,
		},
		[]byte("plan body"),
	)

	resetListFilterFlags(t, "plan")
	resp := runListJSON(t, "plan")

	require.Equal(t, []string{"20260709000000-feature"}, fileNames(resp.Files),
		"top-level plan list must return the plan sub-directory as its own entry")

	planDir := resp.Files[0]
	require.Equal(t, "20260709000000-feature", planDir["name"])
	require.Contains(t, planDir, "path")
	// Directory entries must not carry metadata fields — those live on the
	// plan.md file inside, not on the containing directory.
	_, hasStatus := planDir["status"]
	require.False(t, hasStatus, "directory entries must not carry a status field")
	_, hasCreated := planDir["created_date"]
	require.False(t, hasCreated, "directory entries must not carry a created_date field")
	_, hasClosed := planDir["closed_date"]
	require.False(t, hasClosed, "directory entries must not carry a closed_date field")
}
