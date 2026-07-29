package implement

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jumppad-labs/spektacular/internal/metadata"
	"github.com/jumppad-labs/spektacular/internal/store"
	"github.com/jumppad-labs/spektacular/internal/workflow"
	"github.com/stretchr/testify/require"
)

// today returns time.Now().UTC() truncated to day, matching the value the
// metadata package stamps when UpdateOptions.Today is unset. Used by the
// Phase 1.5 finish-step tests to assert closed_date without pinning wall time.
func today() time.Time {
	return time.Now().UTC().Truncate(24 * time.Hour)
}

type testData struct {
	values map[string]any
}

func (d *testData) Get(key string) (any, bool) {
	v, ok := d.values[key]
	return v, ok
}

func (d *testData) Set(key string, value any) {
	d.values[key] = value
}

type captureWriter struct {
	result Result
}

func (c *captureWriter) WriteResult(v any) error {
	c.result = v.(Result)
	return nil
}

func renderStep(t *testing.T, cb workflow.StepCallback) string {
	t.Helper()
	data := &testData{values: map[string]any{"name": "test"}}
	writer := &captureWriter{}
	st := store.NewFileStore(t.TempDir(), "project")
	_, err := cb(data, writer, st, workflow.Config{Command: "spektacular"})
	require.NoError(t, err)
	return writer.result.Instruction
}

func TestStepsOrderMatchesExpected(t *testing.T) {
	expected := []string{
		"new",
		"read_plan",
		"analyze",
		"implement",
		"test",
		"verify",
		"update_plan",
		"update_changelog",
		"update_repo_changelog",
		"test_plan",
		"update_feature_changelog",
		"reconcile_spec",
		"finished",
	}
	got := Steps()
	require.Len(t, got, len(expected))
	for i, step := range got {
		require.Equal(t, expected[i], step.Name, "step %d name mismatch", i)
	}
}

func TestAnalyzeStepHasMultiSourceTransition(t *testing.T) {
	for _, s := range Steps() {
		if s.Name == "analyze" {
			require.ElementsMatch(t, []string{"read_plan", "update_changelog"}, s.Src,
				"analyze must be reachable from both read_plan and update_changelog")
			return
		}
	}
	t.Fatal("analyze step not found")
}

func TestReconcileSpecStepWiring(t *testing.T) {
	steps := Steps()

	var reconcileSpecStep, finishedStep *workflow.StepConfig
	for i := range steps {
		switch steps[i].Name {
		case "reconcile_spec":
			reconcileSpecStep = &steps[i]
		case "finished":
			finishedStep = &steps[i]
		}
	}

	require.NotNil(t, reconcileSpecStep, "reconcile_spec step not found")
	require.NotNil(t, finishedStep, "finished step not found")

	require.Equal(t, []string{"update_feature_changelog"}, reconcileSpecStep.Src,
		"reconcile_spec must only be reachable from update_feature_changelog")
	require.Equal(t, []string{"reconcile_spec"}, finishedStep.Src,
		"finished must only be reachable from reconcile_spec")
}

func TestFSMWalkFromNewToFinished(t *testing.T) {
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")
	st := store.NewFileStore(tmp, "project")
	writer := &captureWriter{}

	wf := workflow.New(Steps(), statePath, workflow.Config{Command: "spektacular", DryRun: true}, st, writer)
	wf.SetData("name", "test")

	require.Equal(t, "start", wf.Current())

	// new auto-advances to read_plan, so the first Next lands on read_plan.
	// Walk forward with Next() up through update_changelog. Then use explicit
	// Goto() to disambiguate the multi-source exit (update_changelog has two
	// legal successors — analyze via the loop, and update_repo_changelog —
	// so Next() cannot pick one deterministically).
	linear := []string{
		"read_plan",
		"analyze",
		"implement",
		"test",
		"verify",
		"update_plan",
		"update_changelog",
	}
	for _, want := range linear {
		require.NoError(t, wf.Next(), "transition to %s failed", want)
		require.Equal(t, want, wf.Current(), "expected state %s after transition", want)
	}

	require.NoError(t, wf.Goto("update_repo_changelog"))
	require.Equal(t, "update_repo_changelog", wf.Current())
	require.NoError(t, wf.Goto("test_plan"))
	require.Equal(t, "test_plan", wf.Current())
	require.NoError(t, wf.Goto("update_feature_changelog"))
	require.Equal(t, "update_feature_changelog", wf.Current())
	require.NoError(t, wf.Goto("reconcile_spec"))
	require.Equal(t, "reconcile_spec", wf.Current())
	require.NoError(t, wf.Goto("finished"))
	require.Equal(t, "finished", wf.Current())
}

func TestFSMLoopFromUpdateChangelogBackToAnalyze(t *testing.T) {
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")
	st := store.NewFileStore(tmp, "project")
	writer := &captureWriter{}

	wf := workflow.New(Steps(), statePath, workflow.Config{Command: "spektacular", DryRun: true}, st, writer)
	wf.SetData("name", "test")

	// Walk through to update_changelog the first time.
	for _, want := range []string{"read_plan", "analyze", "implement", "test", "verify", "update_plan", "update_changelog"} {
		require.NoError(t, wf.Next())
		require.Equal(t, want, wf.Current())
	}

	// Loop back via the multi-source edge: update_changelog → analyze.
	require.NoError(t, wf.Goto("analyze"))
	require.Equal(t, "analyze", wf.Current())

	// Walk forward again to update_changelog.
	for _, want := range []string{"implement", "test", "verify", "update_plan", "update_changelog"} {
		require.NoError(t, wf.Next())
		require.Equal(t, want, wf.Current())
	}

	// Second exit: update_changelog → update_repo_changelog → test_plan → update_feature_changelog → reconcile_spec → finished.
	require.NoError(t, wf.Goto("update_repo_changelog"))
	require.Equal(t, "update_repo_changelog", wf.Current())
	require.NoError(t, wf.Goto("test_plan"))
	require.Equal(t, "test_plan", wf.Current())
	require.NoError(t, wf.Goto("update_feature_changelog"))
	require.Equal(t, "update_feature_changelog", wf.Current())
	require.NoError(t, wf.Goto("reconcile_spec"))
	require.Equal(t, "reconcile_spec", wf.Current())
	require.NoError(t, wf.Goto("finished"))
	require.Equal(t, "finished", wf.Current())
}

// --- Per-template content assertions ---

func TestReadPlanStepContainsFullReadDirective(t *testing.T) {
	out := renderStep(t, readPlan())
	lower := strings.ToLower(out)
	require.Contains(t, lower, "in full", "read_plan must direct a full read of the plan documents")
	// Plan documents are read through the CLI, never the built-in Read tool.
	require.Contains(t, out, "plan file read", "read_plan must read the plan documents via `plan file read`")
	require.Contains(t, out, "context.md")
	require.Contains(t, out, "research.md")
}

func TestReadPlanStepMentionsChangelog(t *testing.T) {
	out := renderStep(t, readPlan())
	require.Contains(t, out, "## Changelog")
	require.Contains(t, strings.ToLower(out), "first-phase")
	require.Contains(t, strings.ToLower(out), "subsequent-phase")
}

func TestReadPlanTemplateDirectsStructuralValidation(t *testing.T) {
	out := renderStep(t, readPlan())
	// Every required scaffold section must be mentioned.
	for _, section := range []string{
		"## Overview",
		"## Architecture & Design Decisions",
		"## Component Breakdown",
		"## Data Structures & Interfaces",
		"## Implementation Detail",
		"## Dependencies",
		"## Testing Approach",
		"## Milestones & Phases",
		"## Open Questions",
		"## Out of Scope",
	} {
		require.Contains(t, out, section, "read_plan must require section %q", section)
	}
	require.Contains(t, out, "#### - [ ] Phase")
	require.Contains(t, out, "*Technical detail:*")
}

func TestReadPlanTemplateDirectsDriftCheck(t *testing.T) {
	out := renderStep(t, readPlan())
	lower := strings.ToLower(out)
	require.Contains(t, lower, "drift")
	require.Contains(t, lower, "working tree", "drift check must name the target")
	require.Contains(t, lower, "stop")
	// The three-option prompt (fix / proceed / abandon).
	require.Contains(t, lower, "fix the plan first")
	require.Contains(t, lower, "proceed with")
	require.Contains(t, lower, "abandon")
}

func TestReadPlanTemplateDirectsSpecCoverageCheck(t *testing.T) {
	out := renderStep(t, readPlan())
	lower := strings.ToLower(out)
	require.Contains(t, out, "spec file read test.md", "spec coverage check must read the spec via `spec file read`")
	require.Contains(t, out, "## Requirements")
	require.Contains(t, out, "## Acceptance Criteria")
	require.Contains(t, lower, "stop")
	// The two-option prompt (fix / accept as descoped).
	require.Contains(t, lower, "fix the plan first")
	require.Contains(t, lower, "accept the gap as descoped")
}

func TestReadPlanTemplateDirectsDescopedMarkerMechanics(t *testing.T) {
	out := renderStep(t, readPlan())
	require.Contains(t, out, "**Descoped requirements**:", "descoped gaps must use the documented marker format")
	require.Contains(t, strings.ToLower(out), "already recorded as accepted", "spec coverage check must skip gaps already recorded as accepted")
	require.Contains(t, out, "plan file write test/plan.md --from .spektacular/tmp/plan_update.md", "descoped marker must be committed via `plan file write`")
}

func TestAnalyzeStepReferencesSpawnImplementationAgents(t *testing.T) {
	out := renderStep(t, analyze())
	require.Contains(t, out, "skill spawn-implementation-agents")
}

func TestImplementStepForbidsInlineTests(t *testing.T) {
	out := renderStep(t, implementStep())
	lower := strings.ToLower(out)
	require.Contains(t, lower, "test")
	require.Contains(t, lower, "next step", "implement step must defer tests to the next step")
}

func TestTestStepReferencesFollowTestPatterns(t *testing.T) {
	out := renderStep(t, testStep())
	require.Contains(t, out, "skill follow-test-patterns")
	require.Contains(t, strings.ToLower(out), "sub-agent")
}

func TestVerifyStepReferencesVerifyImplementation(t *testing.T) {
	out := renderStep(t, verify())
	require.Contains(t, out, "skill verify-implementation")
	require.Contains(t, strings.ToLower(out), "pass/fail")
}

func TestUpdatePlanStepDirectsCheckboxMarking(t *testing.T) {
	out := renderStep(t, updatePlan())
	require.Contains(t, out, "[x]")
	require.Contains(t, out, "- [ ]")
	require.Contains(t, out, "#### - [ ] Phase")
}

func TestUpdateChangelogStepSpecifiesEntryFields(t *testing.T) {
	out := renderStep(t, updateChangelog())
	for _, field := range []string{
		"What was done",
		"Deviations",
		"Files changed",
		"Discoveries",
	} {
		require.Contains(t, out, field, "update_changelog must specify field %q", field)
	}
}

func TestUpdateChangelogStepCreatesSectionOnFirstInvocation(t *testing.T) {
	out := renderStep(t, updateChangelog())
	require.Contains(t, out, "## Changelog")
	require.Contains(t, strings.ToLower(out), "first")
	require.Contains(t, out, "## Out of Scope", "update_changelog must anchor the new section relative to Out of Scope")
}

func TestUpdateChangelogStepBranchesOnUncheckedPhases(t *testing.T) {
	out := renderStep(t, updateChangelog())
	// Both legal exits must be present.
	require.Contains(t, out, `"step":"analyze"`)
	require.Contains(t, out, `"step":"update_repo_changelog"`)
	require.Contains(t, strings.ToLower(out), "ask the user")
}

func TestUpdateRepoChangelogTemplateContainsDirectives(t *testing.T) {
	out := renderStep(t, updateRepoChangelog())
	require.Contains(t, out, "CHANGELOG.md")
	// Mustache substitutes {{plan_name}} with the instance name "test" —
	// assert the resolved value (the section header "## test") is present.
	require.Contains(t, out, "## test")
	require.Contains(t, out, `"step":"test_plan"`)
	require.Contains(t, strings.ToLower(out), "prepend")
}

func TestStopOnMismatchDirectivePresentInEveryNonTerminalTemplate(t *testing.T) {
	nonTerminal := map[string]workflow.StepCallback{
		"read_plan":                readPlan(),
		"analyze":                  analyze(),
		"implement":                implementStep(),
		"test":                     testStep(),
		"verify":                   verify(),
		"update_plan":              updatePlan(),
		"update_changelog":         updateChangelog(),
		"update_repo_changelog":    updateRepoChangelog(),
		"test_plan":                testPlan(),
		"update_feature_changelog": updateFeatureChangelog(),
		"reconcile_spec":           reconcileSpec(),
	}
	for name, cb := range nonTerminal {
		out := renderStep(t, cb)
		require.Contains(t, strings.ToUpper(out), "STOP", "%s template must contain a STOP directive", name)
	}
}

// TestPlanFilePaths_UseConfiguredDirectory asserts the implement path helpers
// root plan, context and research files under the given directory argument
// (Phase 2.2).
func TestPlanFilePaths_UseConfiguredDirectory(t *testing.T) {
	require.Equal(t, "my-plans/x/plan.md", PlanFilePath("my-plans", "x"))
	require.Equal(t, "my-plans/x/context.md", ContextFilePath("my-plans", "x"))
	require.Equal(t, "my-plans/x/research.md", ResearchFilePath("my-plans", "x"))
}

func TestFinishedStepEmitsNoGoto(t *testing.T) {
	out := renderStep(t, finished())
	require.NotContains(t, out, "implement goto", "finished template must not emit a goto command")
	require.Contains(t, strings.ToLower(out), "terminal")
}

func TestFinishedStepMentionsChangelogPath(t *testing.T) {
	out := renderStep(t, finished())
	require.Contains(t, out, "test.md", "finished template must mention the resolved changelog record path")
}

func TestFinishedStepReportsSpecCompletionStatus(t *testing.T) {
	out := renderStep(t, finished())
	require.Contains(t, out, "spec file read test.md", "finished template must direct reading the spec via `spec file read`")
	require.Contains(t, out, "Requirements/Acceptance-Criteria", "finished template must report Requirements/Acceptance-Criteria status")
	require.Contains(t, out, "reconcile_spec", "finished template must reference reconcile_spec as the source of unchecked-item reasons")
	require.Contains(t, out, "deferred, descoped, or not attempted", "finished template must enumerate the possible reasons for unchecked items")
}

func TestUpdateFeatureChangelogStepMentionsSourcesAndCommitCommand(t *testing.T) {
	out := renderStep(t, updateFeatureChangelog())
	require.Contains(t, out, "spec file read test.md", "update_feature_changelog must read the feature's spec via `spec file read`")
	require.Contains(t, out, "plan file read test/plan.md", "update_feature_changelog must read the plan's implementation history via `plan file read`")
	require.Contains(t, out, ".spektacular/tmp/changelog_record.md", "update_feature_changelog must stage its record at the scratch path")
	require.Contains(t, out, "changelog file write test.md --from .spektacular/tmp/changelog_record.md", "update_feature_changelog must commit the record via `changelog file write`")
}

func TestReconcileSpecStepMentionsSourcesAndCommitCommand(t *testing.T) {
	out := renderStep(t, reconcileSpec())
	require.Contains(t, out, "spec file read test.md", "reconcile_spec must read the feature's spec via `spec file read`")
	require.Contains(t, out, "plan file read test/plan.md", "reconcile_spec must read the plan's implementation history via `plan file read`")
	require.Contains(t, out, ".spektacular/tmp/spec_reconcile.md", "reconcile_spec must stage its record at the scratch path")
	require.Contains(t, out, "spec file write test.md --from .spektacular/tmp/spec_reconcile.md", "reconcile_spec must commit the record via `spec file write`")
}

// --- Phase 1.5: terminal-step closure of test-plan and changelog artifacts ---

// TestImplementFinished_ClosesTestPlanAndChangelog seeds both terminal
// artifacts (test-plan.md under PlanDir and <name>.md under ChangelogDir) with
// in-progress metadata dated in the past, then asserts finished() transitions
// both to completed with today's closed_date while preserving created_date.
func TestImplementFinished_ClosesTestPlanAndChangelog(t *testing.T) {
	tmp := t.TempDir()
	st := store.NewFileStore(tmp, "project")
	cfg := workflow.Config{
		Command:      "spektacular",
		PlanDir:      "plans",
		ChangelogDir: "changelog",
		SpecDir:      "specs",
	}
	planName := "fixture"

	created := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	testPlanPath := filepath.Join(cfg.PlanDir, planName, "test-plan.md")
	changelogPath := filepath.Join(cfg.ChangelogDir, planName+".md")

	for _, p := range []string{testPlanPath, changelogPath} {
		seed, err := metadata.Render(metadata.Metadata{
			CreatedDate: created,
			Status:      metadata.StatusInProgress,
		}, []byte("# Body of "+p+"\n"))
		require.NoError(t, err)
		require.NoError(t, st.Write(p, seed))
	}

	data := &testData{values: map[string]any{"name": planName}}
	writer := &captureWriter{}

	_, err := finished()(data, writer, st, cfg)
	require.NoError(t, err)

	for _, p := range []string{testPlanPath, changelogPath} {
		raw, err := st.Read(p)
		require.NoError(t, err, "%s must remain readable after finished()", p)
		meta, _, err := metadata.Split(raw)
		require.NoError(t, err)
		require.NotNil(t, meta, "%s must carry frontmatter", p)
		require.Equal(t, metadata.StatusCompleted, meta.Status, "%s must be completed", p)
		require.True(t, meta.CreatedDate.Equal(created), "%s created_date must be preserved, got %s", p, meta.CreatedDate)
		require.True(t, meta.ClosedDate.Equal(today()), "%s closed_date must be today, got %s", p, meta.ClosedDate)
	}
}

// TestImplementFinished_SkipsMissingArtifacts asserts finished() treats
// store.ErrNotFound as a no-op — it must return cleanly without creating
// either artifact when neither is present in the store.
func TestImplementFinished_SkipsMissingArtifacts(t *testing.T) {
	tmp := t.TempDir()
	st := store.NewFileStore(tmp, "project")
	cfg := workflow.Config{
		Command:      "spektacular",
		PlanDir:      "plans",
		ChangelogDir: "changelog",
		SpecDir:      "specs",
	}
	planName := "fixture"

	data := &testData{values: map[string]any{"name": planName}}
	writer := &captureWriter{}

	_, err := finished()(data, writer, st, cfg)
	require.NoError(t, err, "finished() must not error when the artifacts are missing")

	testPlanPath := filepath.Join(cfg.PlanDir, planName, "test-plan.md")
	changelogPath := filepath.Join(cfg.ChangelogDir, planName+".md")
	require.False(t, st.Exists(testPlanPath), "finished() must not create test-plan.md when it was missing")
	require.False(t, st.Exists(changelogPath), "finished() must not create the changelog when it was missing")
}
