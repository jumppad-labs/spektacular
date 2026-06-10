package plan

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

func TestArchitectureStepContainsOptionsAndAgreementBeat(t *testing.T) {
	out := renderStep(t, architecture())
	require.Contains(t, strings.ToLower(out), "option", "architecture step must prompt the agent to present design options")
	require.Contains(t, strings.ToLower(out), "agreement", "architecture step must prompt the agent to get user agreement")
}

// TestDiscoveryStepUsesKnowledgeCommands asserts the discovery step drives the
// agent through the `knowledge` CLI commands instead of a hardcoded knowledge
// directory, and gates knowledge writes behind explicit user confirmation
// (Phase 3.1, acceptance criteria 1 and 2).
func TestDiscoveryStepUsesKnowledgeCommands(t *testing.T) {
	out := renderStep(t, discovery())

	// Criterion 1: uses the knowledge commands, no hardcoded knowledge path.
	require.Contains(t, out, "knowledge search", "discovery step must instruct the agent to use `knowledge search`")
	require.Contains(t, out, "knowledge read", "discovery step must instruct the agent to use `knowledge read`")
	require.NotContains(t, out, ".spektacular/knowledge/", "discovery step must not hardcode the knowledge directory path")

	// Criterion 2: knowledge writes require explicit user confirmation.
	require.Contains(t, out, "knowledge write", "discovery step must reference `knowledge write` for capturing learnings")
	require.Contains(t, strings.ToLower(out), "confirm", "discovery step must require explicit user confirmation before a knowledge write")
}

func TestImplementationDetailStepIsHighLevelOnly(t *testing.T) {
	out := renderStep(t, implementationDetail())
	require.Contains(t, strings.ToLower(out), "high-level", "implementation_detail step must enforce high-level only content")
	require.Contains(t, out, "context.md", "implementation_detail step must redirect per-phase detail to context.md")
}

func TestTestingApproachStepIsHighLevelOnly(t *testing.T) {
	out := renderStep(t, testingApproach())
	require.Contains(t, strings.ToLower(out), "high-level", "testing_approach step must enforce high-level only content")
	require.Contains(t, out, "context.md", "testing_approach step must redirect per-phase detail to context.md")
}

func TestOpenQuestionsStepRestrictsToImplTimeUncertainties(t *testing.T) {
	out := renderStep(t, openQuestions())
	require.Contains(t, strings.ToLower(out), "implementation", "open_questions step must restrict the section to impl-time uncertainties")
	require.Contains(t, strings.ToLower(out), "cannot be resolved", "open_questions step must state the cannot-resolve-now rule")
}

func TestOutOfScopeStepCoversExclusions(t *testing.T) {
	out := renderStep(t, outOfScope())
	require.Contains(t, out, "Out of Scope", "out_of_scope step must name the section it populates")
	require.Contains(t, strings.ToLower(out), "exclusion", "out_of_scope step must prompt for explicit exclusions")
}

func TestStepsOrderMatchesExpected(t *testing.T) {
	expected := []string{
		"new",
		"overview",
		"discovery",
		"architecture",
		"components",
		"data_structures",
		"implementation_detail",
		"dependencies",
		"testing_approach",
		"milestones",
		"phases",
		"open_questions",
		"out_of_scope",
		"assemble",
		"verification",
		"write_plan",
		"write_context",
		"write_research",
		"finished",
	}
	got := Steps()
	require.Len(t, got, len(expected))
	for i, step := range got {
		require.Equal(t, expected[i], step.Name, "step %d name mismatch", i)
	}
}

func TestFSMWalkFromNewToFinished(t *testing.T) {
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")
	st := store.NewFileStore(tmp, "project")
	writer := &captureWriter{}

	wf := workflow.New(Steps(), statePath, workflow.Config{Command: "spektacular", DryRun: true, PlanDir: "plans", SpecDir: "specs"}, st, writer)
	wf.SetData("name", "test")

	require.Equal(t, "start", wf.Current())

	expectedStates := []string{
		"overview",
		"discovery",
		"architecture",
		"components",
		"data_structures",
		"implementation_detail",
		"dependencies",
		"testing_approach",
		"milestones",
		"phases",
		"open_questions",
		"out_of_scope",
		"assemble",
		"verification",
		"write_plan",
		"write_context",
		"write_research",
		"finished",
	}

	for _, want := range expectedStates {
		require.NoError(t, wf.Next(), "transition to %s failed", want)
		require.Equal(t, want, wf.Current(), "expected state %s after transition", want)
	}
}

// TestPlanFilePaths_UseConfiguredDirectory asserts the path helpers root plan,
// context and research files under the given directory argument (Phase 2.2,
// acceptance criterion 2).
func TestPlanFilePaths_UseConfiguredDirectory(t *testing.T) {
	require.Equal(t, "my-plans/x/plan.md", PlanFilePath("my-plans", "x"))
	require.Equal(t, "my-plans/x/context.md", ContextFilePath("my-plans", "x"))
	require.Equal(t, "my-plans/x/research.md", ResearchFilePath("my-plans", "x"))
}

// TestWriteStep_CommitsOwnDocument asserts a write step's callback reads its own
// document back through the store: when the document is missing it instructs the
// agent to commit it via `plan file write`; once committed it reports the
// document is already in the store and drops the commit instruction.
func TestWriteStep_CommitsOwnDocument(t *testing.T) {
	tmp := t.TempDir()
	st := store.NewFileStore(tmp, "project")
	writer := &captureWriter{}
	data := &testData{values: map[string]any{"name": "test"}}
	cfg := workflow.Config{Command: "spektacular", PlanDir: "my-plans", SpecDir: "specs"}

	// plan.md absent from the store — the step must instruct the commit.
	_, err := writePlan()(data, writer, st, cfg)
	require.NoError(t, err)
	require.Contains(t, writer.result.Instruction, "--from .spektacular/tmp/plan_template.md",
		"write_plan must instruct committing plan.md from its scratch file when it is not yet in the store")

	// A committed, filled plan.md — no commit command, reports done.
	require.NoError(t, st.Write(PlanFilePath("my-plans", "test"), []byte("# real filled plan")))
	_, err = writePlan()(data, writer, st, cfg)
	require.NoError(t, err)
	require.Contains(t, writer.result.Instruction, "already been committed",
		"write_plan must report plan.md is already committed once it is in the store")
	require.NotContains(t, writer.result.Instruction, "--from .spektacular/tmp/plan_template.md",
		"write_plan must not re-instruct the commit once plan.md is in the store")
}
