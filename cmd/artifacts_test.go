package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/jumppad-labs/spektacular/internal/metadata"
	"github.com/jumppad-labs/spektacular/internal/output"
	"github.com/stretchr/testify/require"
)

// artifactsListConfigYAML is the config used by every acceptance test in this
// file: all three stored-artifact classes (spec, plan, changelog) point at
// docs/-rooted directories, so a single seed helper can lay out any mix of
// artifacts without reconfiguring per subtest.
const artifactsListConfigYAML = "spec:\n  config:\n    directory: docs/specs\nplan:\n  config:\n    directory: docs/plans\nchangelog:\n  config:\n    directory: docs/changelog\n"

// artifactsListFlagNames enumerates the six per-subtest cobra flags exposed by
// `spektacular artifacts list`. Every one of them is a package-global closure
// on artifactsCmd's list subcommand, so a flag set by one test leaks into the
// next unless reset — the same trap resetListFilterFlags / resetWriteStatusFlag
// / resetSetStatusFlag guard for their respective subcommands.
var artifactsListFlagNames = []string{"kind", "status", "created-after", "created-before", "closed-after", "closed-before"}

// resetArtifactsListFlags clears every flag on `artifacts list` and its
// `Changed` bit, both immediately and via t.Cleanup, so no state leaks into
// or out of the current test.
func resetArtifactsListFlags(t *testing.T) {
	t.Helper()
	listCmd, _, err := rootCmd.Find([]string{"artifacts", "list"})
	require.NoError(t, err)
	require.NotNil(t, listCmd)
	reset := func() {
		for _, name := range artifactsListFlagNames {
			require.NoError(t, listCmd.Flags().Set(name, ""))
			if f := listCmd.Flags().Lookup(name); f != nil {
				f.Changed = false
			}
		}
	}
	reset()
	t.Cleanup(reset)
}

// artifactsResponse is the shape `spektacular artifacts list` writes on
// success: `{"error": false, "artifacts": [...]}`. Each entry carries at
// minimum kind, name, and path; frontmatter-bearing entries additionally
// carry created_date, status, and (when non-zero) closed_date.
type artifactsResponse struct {
	Error     bool             `json:"error"`
	Artifacts []map[string]any `json:"artifacts"`
}

// runArtifactsListJSON invokes `artifacts list [args...]` through runRoot
// (the same wrapper Execute uses in production) and decodes the response
// envelope. It requires success — a failing invocation fails the test so
// callers can focus on asserting the returned artifacts array.
func runArtifactsListJSON(t *testing.T, args ...string) artifactsResponse {
	t.Helper()
	stdout, stderr, code := runRootCmd(t, append([]string{"artifacts", "list"}, args...)...)
	require.Equalf(t, 0, code, "artifacts list failed: stdout=%s stderr=%s", stdout, stderr)
	require.Empty(t, stderr)

	var resp artifactsResponse
	require.NoError(t, json.Unmarshal([]byte(stdout), &resp))
	require.False(t, resp.Error)
	return resp
}

// artifactKinds returns the sorted list of kind discriminants across every
// artifact in resp. Sorted so test assertions are stable regardless of scan
// order across the three appendXArtifacts calls.
func artifactKinds(resp artifactsResponse) []string {
	kinds := make([]string, 0, len(resp.Artifacts))
	for _, a := range resp.Artifacts {
		if k, ok := a["kind"].(string); ok {
			kinds = append(kinds, k)
		}
	}
	sort.Strings(kinds)
	return kinds
}

// findArtifactByKindAndName locates the unique entry matching (kind, name),
// or nil if none is present. Names are unique per kind in every seed used in
// this file, so this returns at most one match.
func findArtifactByKindAndName(artifacts []map[string]any, kind, name string) map[string]any {
	for _, a := range artifacts {
		if a["kind"] == kind && a["name"] == name {
			return a
		}
	}
	return nil
}

// TestArtifactsList_EmptyStoreReturnsEmptyEnvelope asserts acceptance criterion
// 5: with the store configured but no artifacts on disk anywhere, the command
// returns a well-formed envelope with an empty artifacts array — never a nil
// slice or a missing key. Same shape as a populated store, just empty.
func TestArtifactsList_EmptyStoreReturnsEmptyEnvelope(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeSpecCommandConfig(t, dir, artifactsListConfigYAML)

	resetArtifactsListFlags(t)
	resp := runArtifactsListJSON(t)

	require.NotNil(t, resp.Artifacts, "the artifacts key must always be present, even when empty")
	require.Empty(t, resp.Artifacts)
}

// TestArtifactsList_UnfilteredReturnsAllKindsTagged asserts acceptance criteria
// 1 and 2: an unfiltered scan produces one entry per artifact across all three
// classes, each tagged with the correct kind discriminant and carrying kind /
// name / path / metadata fields.
func TestArtifactsList_UnfilteredReturnsAllKindsTagged(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeSpecCommandConfig(t, dir, artifactsListConfigYAML)

	created := time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC)
	m := metadata.Metadata{CreatedDate: created, Status: metadata.StatusInProgress}

	// One spec, one plan directory with all four sibling docs, two changelog entries.
	seedArtifactWithMetadata(t, dir, filepath.Join("docs", "specs", "20260709000000-feature.md"), m, []byte("spec body"))
	seedArtifactWithMetadata(t, dir, filepath.Join("docs", "plans", "20260709000000-feature", "plan.md"), m, []byte("plan body"))
	seedArtifactWithMetadata(t, dir, filepath.Join("docs", "plans", "20260709000000-feature", "context.md"), m, []byte("context body"))
	seedArtifactWithMetadata(t, dir, filepath.Join("docs", "plans", "20260709000000-feature", "research.md"), m, []byte("research body"))
	seedArtifactWithMetadata(t, dir, filepath.Join("docs", "plans", "20260709000000-feature", "test-plan.md"), m, []byte("test-plan body"))
	seedArtifactWithMetadata(t, dir, filepath.Join("docs", "changelog", "testproj", "20260710000000-first.md"), m, []byte("changelog 1"))
	seedArtifactWithMetadata(t, dir, filepath.Join("docs", "changelog", "testproj", "20260711000000-second.md"), m, []byte("changelog 2"))

	resetArtifactsListFlags(t)
	resp := runArtifactsListJSON(t)

	require.Len(t, resp.Artifacts, 7, "one entry per seeded artifact across all three classes")

	require.Equal(t,
		[]string{
			"changelog", "changelog",
			"plan.context", "plan.plan", "plan.research", "plan.test-plan",
			"spec",
		},
		artifactKinds(resp),
		"kind discriminants must be tagged 1xspec + 4xplan.* + 2xchangelog")

	// Sanity: every entry carries kind, name, path — the always-present triple.
	for i, a := range resp.Artifacts {
		require.Containsf(t, a, "kind", "entry %d missing kind", i)
		require.Containsf(t, a, "name", "entry %d missing name", i)
		require.Containsf(t, a, "path", "entry %d missing path", i)
	}
}

// TestArtifactsList_KindFilterNarrowsToRequestedClasses asserts acceptance
// criterion 4 and the invalid-kind error path: --kind narrows the result to
// only the requested classes (single or comma-separated), and any unknown
// value in the list returns an invalid_kind error envelope.
func TestArtifactsList_KindFilterNarrowsToRequestedClasses(t *testing.T) {
	seed := func(t *testing.T, dir string) {
		created := time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC)
		m := metadata.Metadata{CreatedDate: created, Status: metadata.StatusInProgress}
		seedArtifactWithMetadata(t, dir, filepath.Join("docs", "specs", "20260709000000-feature.md"), m, []byte("spec"))
		seedArtifactWithMetadata(t, dir, filepath.Join("docs", "plans", "20260709000000-feature", "plan.md"), m, []byte("plan"))
		seedArtifactWithMetadata(t, dir, filepath.Join("docs", "plans", "20260709000000-feature", "context.md"), m, []byte("context"))
		seedArtifactWithMetadata(t, dir, filepath.Join("docs", "plans", "20260709000000-feature", "research.md"), m, []byte("research"))
		seedArtifactWithMetadata(t, dir, filepath.Join("docs", "plans", "20260709000000-feature", "test-plan.md"), m, []byte("test-plan"))
		seedArtifactWithMetadata(t, dir, filepath.Join("docs", "changelog", "testproj", "20260710000000-first.md"), m, []byte("cl1"))
		seedArtifactWithMetadata(t, dir, filepath.Join("docs", "changelog", "testproj", "20260711000000-second.md"), m, []byte("cl2"))
	}

	t.Run("single kind narrows to just that class", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		writeSpecCommandConfig(t, dir, artifactsListConfigYAML)
		seed(t, dir)

		resetArtifactsListFlags(t)
		resp := runArtifactsListJSON(t, "--kind", "plan.context")

		require.Len(t, resp.Artifacts, 1)
		require.Equal(t, "plan.context", resp.Artifacts[0]["kind"])
		require.Equal(t, "context.md", resp.Artifacts[0]["name"])
	})

	t.Run("comma-separated list unions requested classes", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		writeSpecCommandConfig(t, dir, artifactsListConfigYAML)
		seed(t, dir)

		resetArtifactsListFlags(t)
		resp := runArtifactsListJSON(t, "--kind", "spec,changelog")

		require.Len(t, resp.Artifacts, 3, "1 spec + 2 changelog = 3")
		require.Equal(t, []string{"changelog", "changelog", "spec"}, artifactKinds(resp))
	})

	t.Run("invalid kind value returns invalid_kind error", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		writeSpecCommandConfig(t, dir, artifactsListConfigYAML)
		seed(t, dir)

		resetArtifactsListFlags(t)
		stdout, _, code := runRootCmd(t, "artifacts", "list", "--kind", "bogus")
		require.Equal(t, 1, code)

		var er output.ErrorResponse
		require.NoError(t, json.Unmarshal([]byte(stdout), &er))
		require.True(t, er.IsError)
		require.Equal(t, "invalid_kind", er.Code)
		require.Contains(t, er.Message, "bogus")
	})
}

// TestArtifactsList_StatusFilterAppliedAcrossAllKinds asserts acceptance
// criterion 3: --status filters uniformly across every scanned class, not
// just one — a mixed seed of in-progress and completed artifacts across
// specs, plan siblings, and changelog entries returns exactly the completed
// subset when --status completed is set.
func TestArtifactsList_StatusFilterAppliedAcrossAllKinds(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeSpecCommandConfig(t, dir, artifactsListConfigYAML)

	created := time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC)
	closed := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)
	ip := metadata.Metadata{CreatedDate: created, Status: metadata.StatusInProgress}
	done := metadata.Metadata{CreatedDate: created, Status: metadata.StatusCompleted, ClosedDate: closed}

	// One in-progress + one completed of each class (specs, plan siblings, changelog).
	seedArtifactWithMetadata(t, dir, filepath.Join("docs", "specs", "20260701000000-open.md"), ip, []byte("open"))
	seedArtifactWithMetadata(t, dir, filepath.Join("docs", "specs", "20260702000000-done.md"), done, []byte("done"))
	seedArtifactWithMetadata(t, dir, filepath.Join("docs", "plans", "20260709000000-feature", "plan.md"), ip, []byte("open"))
	seedArtifactWithMetadata(t, dir, filepath.Join("docs", "plans", "20260709000000-feature", "context.md"), done, []byte("done"))
	seedArtifactWithMetadata(t, dir, filepath.Join("docs", "changelog", "testproj", "20260710000000-open.md"), ip, []byte("open"))
	seedArtifactWithMetadata(t, dir, filepath.Join("docs", "changelog", "testproj", "20260711000000-done.md"), done, []byte("done"))

	resetArtifactsListFlags(t)
	resp := runArtifactsListJSON(t, "--status", "completed")

	require.Len(t, resp.Artifacts, 3, "3 completed artifacts across 3 classes")

	// Every returned entry must carry status: completed.
	for _, a := range resp.Artifacts {
		require.Equal(t, "completed", a["status"])
	}
	// And they must span the classes seeded as completed — not all in one class.
	require.Equal(t, []string{"changelog", "plan.context", "spec"}, artifactKinds(resp))
}

// TestArtifactsList_CreatedDateRangeFiltersAcrossAllKinds asserts acceptance
// criterion 3 again for the created-date filter, and additionally that the
// inclusive-range semantics from Phase 2.1 (`parseListFilter`) are inherited
// verbatim: a seed spanning Jan 1 / Jan 15 / Jan 31 with a range of Jan 10 -
// Jan 20 returns only the Jan 15 entry, across all classes.
func TestArtifactsList_CreatedDateRangeFiltersAcrossAllKinds(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeSpecCommandConfig(t, dir, artifactsListConfigYAML)

	jan1 := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	jan15 := time.Date(2026, time.January, 15, 0, 0, 0, 0, time.UTC)
	jan31 := time.Date(2026, time.January, 31, 0, 0, 0, 0, time.UTC)

	// One artifact per date in each of the three classes — nine total, of
	// which three (one per class) sit on the Jan 15 boundary.
	for _, d := range []struct {
		when time.Time
		suf  string
	}{
		{jan1, "jan1"},
		{jan15, "jan15"},
		{jan31, "jan31"},
	} {
		m := metadata.Metadata{CreatedDate: d.when, Status: metadata.StatusInProgress}
		seedArtifactWithMetadata(t, dir, filepath.Join("docs", "specs", "20260701000000-"+d.suf+".md"), m, []byte(d.suf))
		seedArtifactWithMetadata(t, dir, filepath.Join("docs", "plans", "20260709000000-"+d.suf, "plan.md"), m, []byte(d.suf))
		seedArtifactWithMetadata(t, dir, filepath.Join("docs", "changelog", "testproj", "20260710000000-"+d.suf+".md"), m, []byte(d.suf))
	}

	resetArtifactsListFlags(t)
	resp := runArtifactsListJSON(t, "--created-after", "2026-01-10", "--created-before", "2026-01-20")

	require.Len(t, resp.Artifacts, 3, "one Jan 15 artifact per class must survive the range")
	for _, a := range resp.Artifacts {
		require.Equal(t, "2026-01-15", a["created_date"], "only Jan 15 artifacts must appear")
	}
	require.Equal(t, []string{"changelog", "plan.plan", "spec"}, artifactKinds(resp),
		"the surviving Jan 15 entries must span every configured class")
}

// TestArtifactsList_CombinedFiltersIntersect asserts acceptance criterion 3's
// AND semantics: combining --kind and --status yields the intersection, not
// the union. Only specs that are also completed must appear — a completed
// changelog and an in-progress spec both fall outside the intersection.
func TestArtifactsList_CombinedFiltersIntersect(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeSpecCommandConfig(t, dir, artifactsListConfigYAML)

	created := time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC)
	closed := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)
	ip := metadata.Metadata{CreatedDate: created, Status: metadata.StatusInProgress}
	done := metadata.Metadata{CreatedDate: created, Status: metadata.StatusCompleted, ClosedDate: closed}

	// Two specs: one in-progress, one completed. Two changelog entries: same.
	// Only spec+completed satisfies --kind spec --status completed.
	seedArtifactWithMetadata(t, dir, filepath.Join("docs", "specs", "20260701000000-open.md"), ip, []byte("open"))
	seedArtifactWithMetadata(t, dir, filepath.Join("docs", "specs", "20260702000000-done.md"), done, []byte("done"))
	seedArtifactWithMetadata(t, dir, filepath.Join("docs", "changelog", "testproj", "20260710000000-open.md"), ip, []byte("open"))
	seedArtifactWithMetadata(t, dir, filepath.Join("docs", "changelog", "testproj", "20260711000000-done.md"), done, []byte("done"))

	resetArtifactsListFlags(t)
	resp := runArtifactsListJSON(t, "--kind", "spec", "--status", "completed")

	require.Len(t, resp.Artifacts, 1, "only spec + completed survives the intersection")
	require.Equal(t, "spec", resp.Artifacts[0]["kind"])
	require.Equal(t, "20260702000000-done.md", resp.Artifacts[0]["name"])
	require.Equal(t, "completed", resp.Artifacts[0]["status"])
}

// TestArtifactsList_BareArtifactsSurfaceInUnfilteredButNotFiltered asserts the
// bare-artifact behaviour scanArtifact inherits from Phase 2.1: a
// no-frontmatter artifact appears in an unfiltered scan carrying only kind /
// name / path (no metadata fields), but is silently excluded from any
// filtered scan because it can't satisfy a metadata predicate.
func TestArtifactsList_BareArtifactsSurfaceInUnfilteredButNotFiltered(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeSpecCommandConfig(t, dir, artifactsListConfigYAML)

	created := time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC)
	ip := metadata.Metadata{CreatedDate: created, Status: metadata.StatusInProgress}
	seedArtifactWithMetadata(t, dir, filepath.Join("docs", "specs", "20260701000000-in-progress.md"), ip, []byte("open"))

	// Bare spec: raw body with no frontmatter block at all.
	bareAbs := filepath.Join(dir, "docs", "specs", "20260702000000-bare.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(bareAbs), 0o755))
	require.NoError(t, os.WriteFile(bareAbs, []byte("bare legacy body"), 0o644))

	// Unfiltered: both surface, bare carrying only kind/name/path.
	resetArtifactsListFlags(t)
	resp := runArtifactsListJSON(t)
	require.Len(t, resp.Artifacts, 2)

	bare := findArtifactByKindAndName(resp.Artifacts, "spec", "20260702000000-bare.md")
	require.NotNil(t, bare)
	require.Contains(t, bare, "kind")
	require.Contains(t, bare, "name")
	require.Contains(t, bare, "path")
	_, hasStatus := bare["status"]
	require.False(t, hasStatus, "a bare artifact must not carry a status field")
	_, hasCreated := bare["created_date"]
	require.False(t, hasCreated, "a bare artifact must not carry a created_date field")

	// Filtered: bare artifact is silently dropped.
	resetArtifactsListFlags(t)
	resp = runArtifactsListJSON(t, "--status", "in-progress")
	require.Len(t, resp.Artifacts, 1, "any metadata filter must exclude bare artifacts")
	require.Equal(t, "20260701000000-in-progress.md", resp.Artifacts[0]["name"])
}

// TestArtifactsList_MetadataFieldsPresentPerEntry asserts acceptance
// criterion 2 for both a completed (closed_date present) and an in-progress
// (closed_date absent) artifact — a positive assertion that every carried
// field lands in the entry, complementing the bare-artifact test's negative
// assertion.
func TestArtifactsList_MetadataFieldsPresentPerEntry(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeSpecCommandConfig(t, dir, artifactsListConfigYAML)

	created := time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC)
	closed := time.Date(2026, time.February, 5, 0, 0, 0, 0, time.UTC)

	seedArtifactWithMetadata(t, dir, filepath.Join("docs", "specs", "20260701000000-open.md"),
		metadata.Metadata{CreatedDate: created, Status: metadata.StatusInProgress}, []byte("open"))
	seedArtifactWithMetadata(t, dir, filepath.Join("docs", "specs", "20260702000000-done.md"),
		metadata.Metadata{CreatedDate: created, Status: metadata.StatusCompleted, ClosedDate: closed}, []byte("done"))

	resetArtifactsListFlags(t)
	resp := runArtifactsListJSON(t)

	require.Len(t, resp.Artifacts, 2)

	open := findArtifactByKindAndName(resp.Artifacts, "spec", "20260701000000-open.md")
	require.NotNil(t, open)
	require.Equal(t, "spec", open["kind"])
	require.Equal(t, "20260701000000-open.md", open["name"])
	require.Contains(t, open, "path")
	require.Equal(t, "2026-01-10", open["created_date"])
	require.Equal(t, "in-progress", open["status"])
	_, hasClosed := open["closed_date"]
	require.False(t, hasClosed, "an in-progress artifact must not carry closed_date")

	done := findArtifactByKindAndName(resp.Artifacts, "spec", "20260702000000-done.md")
	require.NotNil(t, done)
	require.Equal(t, "spec", done["kind"])
	require.Equal(t, "20260702000000-done.md", done["name"])
	require.Contains(t, done, "path")
	require.Equal(t, "2026-01-10", done["created_date"])
	require.Equal(t, "completed", done["status"])
	require.Equal(t, "2026-02-05", done["closed_date"])
}

// TestArtifactsList_MissingConfiguredDirectoriesAreTreatedAsEmpty asserts the
// defence-in-depth behaviour for fresh workspaces: with the three directories
// configured but never materialised on disk (no docs/specs, docs/plans,
// docs/changelog), the command returns an empty envelope with no error —
// never a "directory does not exist" failure that would break every fresh
// project.
func TestArtifactsList_MissingConfiguredDirectoriesAreTreatedAsEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeSpecCommandConfig(t, dir, artifactsListConfigYAML)

	// Deliberately do not create docs/specs, docs/plans, or docs/changelog.
	resetArtifactsListFlags(t)
	resp := runArtifactsListJSON(t)

	require.NotNil(t, resp.Artifacts)
	require.Empty(t, resp.Artifacts)
}

// Criterion 2: changelog artifacts remain visible to artifact listing under
// the new project-namespaced layout — the scanner lists one folder level down
// (`<changelogDir>/<projectName>/`), so an entry seeded there surfaces with
// kind changelog and its flat name.
func TestArtifactsList_ChangelogEntriesListedFromProjectNamespaceFolder(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeSpecCommandConfig(t, dir, artifactsListConfigYAML)

	created := time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC)
	seedArtifactWithMetadata(t, dir, filepath.Join("docs", "changelog", "testproj", "000002_x.md"),
		metadata.Metadata{CreatedDate: created, Status: metadata.StatusInProgress}, []byte("changelog body"))

	resetArtifactsListFlags(t)
	resp := runArtifactsListJSON(t)

	require.Len(t, resp.Artifacts, 1)
	entry := findArtifactByKindAndName(resp.Artifacts, "changelog", "000002_x.md")
	require.NotNil(t, entry, "the namespaced changelog entry must appear in the listing")
	require.Equal(t, "docs/changelog/testproj/000002_x.md", entry["path"])
	require.Equal(t, "2026-01-10", entry["created_date"])
	require.Equal(t, "in-progress", entry["status"])
}

// Criterion 2: a project whose changelog namespace folder does not exist yet
// lists cleanly — the changelog directory is present but no entry has ever
// been written for this project, so no `<projectName>/` folder exists, and
// the scan must treat that as empty rather than erroring.
func TestArtifactsList_MissingChangelogNamespaceFolderListsCleanly(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeSpecCommandConfig(t, dir, artifactsListConfigYAML)

	// The changelog directory itself exists, but holds no testproj/ folder.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "docs", "changelog"), 0o755))

	created := time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC)
	seedArtifactWithMetadata(t, dir, filepath.Join("docs", "specs", "20260709000000-feature.md"),
		metadata.Metadata{CreatedDate: created, Status: metadata.StatusInProgress}, []byte("spec body"))

	resetArtifactsListFlags(t)
	resp := runArtifactsListJSON(t)

	require.Len(t, resp.Artifacts, 1, "only the spec must appear; the missing namespace folder is empty, not an error")
	require.Equal(t, "spec", resp.Artifacts[0]["kind"])
}

// TestParseKindFlag_EmptyMeansAllKinds asserts the private helper's contract:
// an empty or whitespace-only raw value returns the full six-kind set that
// the unfiltered scan uses.
func TestParseKindFlag_EmptyMeansAllKinds(t *testing.T) {
	for _, raw := range []string{"", "   ", "\t"} {
		t.Run(raw, func(t *testing.T) {
			set, err := parseKindFlag(raw)
			require.NoError(t, err)
			require.Len(t, set, 6, "an empty --kind must mean every recognised class")
			for _, k := range allArtifactKinds {
				require.Truef(t, set[k], "%q must be in the resulting set", k)
			}
		})
	}
}

// TestParseKindFlag_InvalidValueReturnsError asserts that any unknown entry
// in the comma-separated list is rejected with invalid_kind and an actionable
// message naming both the offending value and the recognised alternatives.
func TestParseKindFlag_InvalidValueReturnsError(t *testing.T) {
	for _, bad := range []string{"bogus", "PLAN", "plan.other", "spec,mystery"} {
		t.Run(bad, func(t *testing.T) {
			_, err := parseKindFlag(bad)
			require.Error(t, err)
			var er *output.ErrorResponse
			require.ErrorAs(t, err, &er)
			require.Equal(t, "invalid_kind", er.Code)
			require.NotEmpty(t, er.NextAction)
		})
	}
}

// TestParseKindFlag_CommaSeparatedNarrows asserts a valid comma-separated
// list returns exactly the requested subset — every listed kind present,
// every other kind absent. Whitespace around entries is tolerated.
func TestParseKindFlag_CommaSeparatedNarrows(t *testing.T) {
	set, err := parseKindFlag("spec, plan.context ,changelog")
	require.NoError(t, err)
	require.Len(t, set, 3)
	require.True(t, set[artifactKindSpec])
	require.True(t, set[artifactKindPlanContext])
	require.True(t, set[artifactKindChangelog])
	require.False(t, set[artifactKindPlanPlan])
	require.False(t, set[artifactKindPlanResearch])
	require.False(t, set[artifactKindPlanTestPlan])
}
