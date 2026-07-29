package plan

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jumppad-labs/spektacular/internal/metadata"
	"github.com/jumppad-labs/spektacular/internal/stepkit"
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

// --- Phase 1.5: terminal-step closure across all three siblings ---

// seedFilledPlanDocs writes a filled (non-scaffold) body wrapped in an
// in-progress frontmatter block dated `created` for every planDoc under
// planDir. Returns the created date so callers can assert preservation.
func seedFilledPlanDocs(t *testing.T, st store.Store, planDir, planName string, created time.Time) {
	t.Helper()
	for _, doc := range planDocs {
		scaffold, err := stepkit.RenderTemplate(doc.scaffold, map[string]any{"name": planName})
		require.NoError(t, err)
		filled := append([]byte{}, []byte(scaffold)...)
		filled = append(filled, "\n\n## Real content\n"...)
		wrapped, err := metadata.Render(metadata.Metadata{
			CreatedDate: created,
			Status:      metadata.StatusInProgress,
		}, filled)
		require.NoError(t, err)
		require.NoError(t, st.Write(doc.path(planDir, planName), wrapped))
	}
}

// TestPlanFinished_ClosesAllThreeSiblings seeds the store with filled plan,
// context and research docs and asserts finished() transitions every one of
// them to completed with today's closed_date, preserving each original
// created_date.
func TestPlanFinished_ClosesAllThreeSiblings(t *testing.T) {
	tmp := t.TempDir()
	st := store.NewFileStore(tmp, "project")
	cfg := workflow.Config{Command: "spektacular", PlanDir: "plans", SpecDir: "specs"}
	planName := "fixture"

	created := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	seedFilledPlanDocs(t, st, cfg.PlanDir, planName, created)

	data := &testData{values: map[string]any{"name": planName}}
	writer := &captureWriter{}

	_, err := finished()(data, writer, st, cfg)
	require.NoError(t, err)

	for _, doc := range planDocs {
		p := doc.path(cfg.PlanDir, planName)
		raw, err := st.Read(p)
		require.NoError(t, err, "%s must be readable after finished()", p)

		meta, _, err := metadata.Split(raw)
		require.NoError(t, err, "%s must carry parseable frontmatter", p)
		require.NotNil(t, meta)
		require.Equal(t, metadata.StatusCompleted, meta.Status, "%s must be completed", p)
		require.True(t, meta.CreatedDate.Equal(created), "%s created_date must be preserved, got %s", p, meta.CreatedDate)
		require.True(t, meta.ClosedDate.Equal(today()), "%s closed_date must be today, got %s", p, meta.ClosedDate)
	}
}

// TestPlanFinished_LeavesUnwrittenPlanAlone seeds two filled docs and one
// still-scaffold doc, then asserts finished() does not transition ANY of the
// three to completed — the incomplete gate short-circuits before the close
// loop runs.
func TestPlanFinished_LeavesUnwrittenPlanAlone(t *testing.T) {
	tmp := t.TempDir()
	st := store.NewFileStore(tmp, "project")
	cfg := workflow.Config{Command: "spektacular", PlanDir: "plans", SpecDir: "specs"}
	planName := "fixture"

	created := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)

	// Fill plan.md and context.md; leave research.md as a bare scaffold with
	// in-progress frontmatter so planDocStillScaffold reports it unwritten.
	for i, doc := range planDocs {
		scaffold, err := stepkit.RenderTemplate(doc.scaffold, map[string]any{"name": planName})
		require.NoError(t, err)
		body := []byte(scaffold)
		if i < 2 {
			body = append(body, "\n\n## Real content\n"...)
		}
		wrapped, err := metadata.Render(metadata.Metadata{
			CreatedDate: created,
			Status:      metadata.StatusInProgress,
		}, body)
		require.NoError(t, err)
		require.NoError(t, st.Write(doc.path(cfg.PlanDir, planName), wrapped))
	}

	data := &testData{values: map[string]any{"name": planName}}
	writer := &captureWriter{}

	_, err := finished()(data, writer, st, cfg)
	require.NoError(t, err, "finished() must not error when a plan doc is still the scaffold")

	// Assert none of the docs transitioned to completed.
	for _, doc := range planDocs {
		p := doc.path(cfg.PlanDir, planName)
		raw, err := st.Read(p)
		require.NoError(t, err)
		meta, _, err := metadata.Split(raw)
		require.NoError(t, err)
		require.NotNil(t, meta)
		require.Equal(t, metadata.StatusInProgress, meta.Status,
			"%s must remain in-progress when the plan is incomplete", p)
		require.True(t, meta.ClosedDate.IsZero(), "%s must not gain a closed_date", p)
	}
}

// TestPlanFinished_SkipsClose_WhenPlanIncomplete is the twin of the previous
// test: it verifies the finished() callback surfaces the plan_incomplete
// warning (via the extras passed to writeStep) and does not mutate any
// artifact metadata when the incomplete gate trips.
func TestPlanFinished_SkipsClose_WhenPlanIncomplete(t *testing.T) {
	tmp := t.TempDir()
	st := store.NewFileStore(tmp, "project")
	cfg := workflow.Config{Command: "spektacular", PlanDir: "plans", SpecDir: "specs"}
	planName := "fixture"

	created := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)

	// Only the first sibling missing is enough to trip the gate.
	scaffold, err := stepkit.RenderTemplate(planDocs[0].scaffold, map[string]any{"name": planName})
	require.NoError(t, err)
	bareScaffold, err := metadata.Render(metadata.Metadata{
		CreatedDate: created,
		Status:      metadata.StatusInProgress,
	}, []byte(scaffold))
	require.NoError(t, err)
	require.NoError(t, st.Write(planDocs[0].path(cfg.PlanDir, planName), bareScaffold))

	// Fill the other two so only plan.md trips the gate.
	for _, doc := range planDocs[1:] {
		scaffold, err := stepkit.RenderTemplate(doc.scaffold, map[string]any{"name": planName})
		require.NoError(t, err)
		filled := append([]byte{}, []byte(scaffold)...)
		filled = append(filled, "\n\n## Real content\n"...)
		wrapped, err := metadata.Render(metadata.Metadata{
			CreatedDate: created,
			Status:      metadata.StatusInProgress,
		}, filled)
		require.NoError(t, err)
		require.NoError(t, st.Write(doc.path(cfg.PlanDir, planName), wrapped))
	}

	data := &testData{values: map[string]any{"name": planName}}
	writer := &captureWriter{}

	_, err = finished()(data, writer, st, cfg)
	require.NoError(t, err)

	// The scaffold-check warning surfaces via the rendered instruction — the
	// finished template gates on the plan_incomplete extra.
	require.NotEmpty(t, writer.result.Instruction, "finished() must emit an instruction")

	// None of the sibling artifacts should have flipped to completed.
	for _, doc := range planDocs {
		p := doc.path(cfg.PlanDir, planName)
		raw, err := st.Read(p)
		require.NoError(t, err)
		meta, _, err := metadata.Split(raw)
		require.NoError(t, err)
		require.NotNil(t, meta)
		require.NotEqual(t, metadata.StatusCompleted, meta.Status,
			"%s must not be closed when the plan is incomplete", p)
	}
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
