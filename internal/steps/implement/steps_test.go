package implement

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/jumppad-labs/spektacular/internal/store"
	"github.com/jumppad-labs/spektacular/internal/workflow"
	"github.com/stretchr/testify/require"
)

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

	// Second exit: update_changelog → update_repo_changelog → test_plan → finished.
	require.NoError(t, wf.Goto("update_repo_changelog"))
	require.Equal(t, "update_repo_changelog", wf.Current())
	require.NoError(t, wf.Goto("test_plan"))
	require.Equal(t, "test_plan", wf.Current())
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
		"read_plan":             readPlan(),
		"analyze":               analyze(),
		"implement":             implementStep(),
		"test":                  testStep(),
		"verify":                verify(),
		"update_plan":           updatePlan(),
		"update_changelog":      updateChangelog(),
		"update_repo_changelog": updateRepoChangelog(),
		"test_plan":             testPlan(),
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
