package spec

import (
	"os"
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

func TestStepsOrderMatchesExpected(t *testing.T) {
	expected := []string{
		"new",
		"overview",
		"requirements",
		"acceptance_criteria",
		"constraints",
		"technical_approach",
		"success_metrics",
		"non_goals",
		"verification",
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

	wf := workflow.New(Steps(), statePath, workflow.Config{Command: "spektacular", DryRun: true}, st, writer)
	wf.SetData("name", "test")
	wf.SetData("spec_template", "spec content")

	require.Equal(t, "start", wf.Current())

	// After the "new" step change, the workflow stays at "new" state and returns
	// an instruction. The next transition advances to "overview".
	expectedStates := []string{
		"new",
		"overview",
		"requirements",
		"acceptance_criteria",
		"constraints",
		"technical_approach",
		"success_metrics",
		"non_goals",
		"verification",
		"finished",
	}

	for _, want := range expectedStates {
		require.NoError(t, wf.Next(), "transition to %s failed", want)
		require.Equal(t, want, wf.Current(), "expected state %s after transition", want)
	}
}

func TestOverviewStepRendersInstruction(t *testing.T) {
	out := renderStep(t, overview())
	require.NotEmpty(t, out)
}

func TestVerificationStepPassesSpecTemplate(t *testing.T) {
	// Verification must render with a spec_template extra var populated from
	// the scaffold so the template can embed the scaffold body.
	tmp := t.TempDir()
	data := &testData{values: map[string]any{"name": "test"}}
	writer := &captureWriter{}
	st := store.NewFileStore(tmp, "project")

	_, err := verification()(data, writer, st, workflow.Config{Command: "spektacular"})
	require.NoError(t, err)
	require.NotEmpty(t, writer.result.Instruction)
}

func TestNewStepWritesScaffold(t *testing.T) {
	tmp := t.TempDir()
	
	// Create .spektacular directory for context.md
	spektacularDir := filepath.Join(tmp, ".spektacular")
	require.NoError(t, os.MkdirAll(spektacularDir, 0755))
	
	// Change to temp directory so relative path resolution works
	origWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmp))
	defer os.Chdir(origWd)
	
	data := &testData{values: map[string]any{"name": "fixture"}}
	writer := &captureWriter{}
	st := store.NewFileStore(tmp, "project")

	next, err := new()(data, writer, st, workflow.Config{Command: "spektacular", SpecDir: "specs"})
	require.NoError(t, err)
	require.Equal(t, "", next, "new step should return empty string when using writeStep")
	require.True(t, st.Exists(SpecFilePath("specs", "fixture")))
}

// TestSpecFilePath_UsesConfiguredDirectory asserts the path helper roots the
// spec file under the given directory argument (Phase 2.2, criterion 1).
func TestSpecFilePath_UsesConfiguredDirectory(t *testing.T) {
	require.Equal(t, "my-specs/x.md", SpecFilePath("my-specs", "x"))
}

// TestNewStep_WritesUnderConfiguredSpecDir runs the new callback with a
// non-default SpecDir and asserts the spec file lands under that directory
// (Phase 2.2, criterion 1).
func TestNewStep_WritesUnderConfiguredSpecDir(t *testing.T) {
	tmp := t.TempDir()
	
	// Create .spektacular directory for context.md
	spektacularDir := filepath.Join(tmp, ".spektacular")
	require.NoError(t, os.MkdirAll(spektacularDir, 0755))
	
	// Change to temp directory so relative path resolution works
	origWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmp))
	defer os.Chdir(origWd)
	
	data := &testData{values: map[string]any{"name": "fixture"}}
	writer := &captureWriter{}
	st := store.NewFileStore(tmp, "project")

	_, err = new()(data, writer, st, workflow.Config{Command: "spektacular", SpecDir: "my-specs"})
	require.NoError(t, err)
	require.True(t, st.Exists(SpecFilePath("my-specs", "fixture")), "spec must land under my-specs")
	require.False(t, st.Exists(SpecFilePath("specs", "fixture")), "spec must not land under default specs")
}

// TestNewStep_ClearsContextMd verifies that the new step clears
// .spektacular/context.md after creating the spec scaffold (Phase 2.1).
func TestNewStep_ClearsContextMd(t *testing.T) {
	// Create a temp directory that will serve as the working directory
	tmp := t.TempDir()
	
	// Create .spektacular directory and a pre-existing context.md with content
	spektacularDir := filepath.Join(tmp, ".spektacular")
	require.NoError(t, os.MkdirAll(spektacularDir, 0755))
	contextPath := filepath.Join(spektacularDir, "context.md")
	require.NoError(t, os.WriteFile(contextPath, []byte("old context"), 0644))
	
	// Change to the temp directory so relative path resolution works
	origWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmp))
	defer os.Chdir(origWd)
	
	data := &testData{values: map[string]any{"name": "fixture"}}
	writer := &captureWriter{}
	st := store.NewFileStore(tmp, "project")
	
	_, err = new()(data, writer, st, workflow.Config{Command: "spektacular", SpecDir: "specs"})
	require.NoError(t, err)
	
	// Verify context.md was cleared
	content, err := os.ReadFile(contextPath)
	require.NoError(t, err)
	require.Empty(t, content, "context.md should be cleared to empty")
}

// TestNewStep_ReturnsInstructionToWriteContext verifies that the new step
// returns an instruction (via writeStep) telling the agent to write
// conversation context to .spektacular/context.md (Phase 2.1).
func TestNewStep_ReturnsInstructionToWriteContext(t *testing.T) {
	tmp := t.TempDir()
	
	// Create .spektacular directory
	spektacularDir := filepath.Join(tmp, ".spektacular")
	require.NoError(t, os.MkdirAll(spektacularDir, 0755))
	
	// Change to the temp directory so relative path resolution works
	origWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmp))
	defer os.Chdir(origWd)
	
	data := &testData{values: map[string]any{"name": "fixture"}}
	writer := &captureWriter{}
	st := store.NewFileStore(tmp, "project")
	
	_, err = new()(data, writer, st, workflow.Config{Command: "spektacular", SpecDir: "specs"})
	require.NoError(t, err)
	
	// Verify instruction was written
	require.NotEmpty(t, writer.result.Instruction, "new step should return an instruction")
	require.Contains(t, writer.result.Instruction, "context.md", "instruction should mention context.md")
	require.Contains(t, writer.result.Instruction, "conversation context", "instruction should mention conversation context")
}

// TestNewStep_InstructionIncludesDetailedFormat verifies that the instruction
// specifies what to capture: problem, requirements, constraints, alternatives,
// exact phrasing (Phase 2.1 acceptance criteria).
func TestNewStep_InstructionIncludesDetailedFormat(t *testing.T) {
	tmp := t.TempDir()
	
	// Create .spektacular directory
	spektacularDir := filepath.Join(tmp, ".spektacular")
	require.NoError(t, os.MkdirAll(spektacularDir, 0755))
	
	// Change to the temp directory so relative path resolution works
	origWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmp))
	defer os.Chdir(origWd)
	
	data := &testData{values: map[string]any{"name": "fixture"}}
	writer := &captureWriter{}
	st := store.NewFileStore(tmp, "project")
	
	_, err = new()(data, writer, st, workflow.Config{Command: "spektacular", SpecDir: "specs"})
	require.NoError(t, err)
	
	instruction := strings.ToLower(writer.result.Instruction)
	require.Contains(t, instruction, "problem", "instruction should specify capturing the problem")
	require.Contains(t, instruction, "requirements", "instruction should specify capturing requirements")
	require.Contains(t, instruction, "constraints", "instruction should specify capturing constraints")
	require.Contains(t, instruction, "alternatives", "instruction should specify capturing alternatives")
	require.Contains(t, instruction, "exact phrasing", "instruction should specify capturing exact phrasing")
}

// --- Phase 1.5: metadata stamping and terminal-step closure ---

// setupNewStepEnv creates a temp dir with `.spektacular/context.md` and chdirs
// into it, returning the dir and a cleanup func that restores the original
// working directory. new() writes context.md to a relative path off the cwd,
// so callers exercising new() must run inside a suitable working directory.
func setupNewStepEnv(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	spektacularDir := filepath.Join(tmp, ".spektacular")
	require.NoError(t, os.MkdirAll(spektacularDir, 0755))

	origWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	return tmp
}

// TestSpecNew_StampsMetadataOnFirstWrite asserts new() routes its scaffold
// write through metadata.Merge so the stored spec carries an in-progress
// frontmatter block with created_date=today and no closed_date.
func TestSpecNew_StampsMetadataOnFirstWrite(t *testing.T) {
	tmp := setupNewStepEnv(t)

	data := &testData{values: map[string]any{"name": "fixture"}}
	writer := &captureWriter{}
	st := store.NewFileStore(tmp, "project")

	_, err := new()(data, writer, st, workflow.Config{Command: "spektacular", SpecDir: "specs"})
	require.NoError(t, err)

	raw, err := st.Read(SpecFilePath("specs", "fixture"))
	require.NoError(t, err)

	meta, _, err := metadata.Split(raw)
	require.NoError(t, err)
	require.NotNil(t, meta, "new() must attach frontmatter to the initial scaffold write")
	require.Equal(t, metadata.StatusInProgress, meta.Status, "initial write must be in-progress")
	require.True(t, meta.CreatedDate.Equal(today()), "created_date must be today, got %s", meta.CreatedDate)
	require.True(t, meta.ClosedDate.IsZero(), "closed_date must be absent on an in-progress artifact")
}

// TestSpecFinished_ClosesTheSpec seeds the store with a filled (non-scaffold)
// spec body carrying an in-progress frontmatter block dated in the past and
// asserts finished() transitions the metadata to completed with today's
// closed_date while preserving the seeded created_date.
func TestSpecFinished_ClosesTheSpec(t *testing.T) {
	tmp := t.TempDir()
	st := store.NewFileStore(tmp, "project")
	cfg := workflow.Config{Command: "spektacular", SpecDir: "specs"}

	// Seed a filled body with in-progress frontmatter dated in the past.
	created := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	filled, err := metadata.Render(metadata.Metadata{
		CreatedDate: created,
		Status:      metadata.StatusInProgress,
	}, []byte("# Fixture\n\nFilled body that is not the scaffold.\n"))
	require.NoError(t, err)
	require.NoError(t, st.Write(SpecFilePath("specs", "fixture"), filled))

	data := &testData{values: map[string]any{"name": "fixture"}}
	writer := &captureWriter{}

	_, err = finished()(data, writer, st, cfg)
	require.NoError(t, err)

	raw, err := st.Read(SpecFilePath("specs", "fixture"))
	require.NoError(t, err)

	meta, _, err := metadata.Split(raw)
	require.NoError(t, err)
	require.NotNil(t, meta)
	require.Equal(t, metadata.StatusCompleted, meta.Status, "finished() must transition status to completed")
	require.True(t, meta.CreatedDate.Equal(created), "created_date must be preserved from seed, got %s", meta.CreatedDate)
	require.True(t, meta.ClosedDate.Equal(today()), "closed_date must be today, got %s", meta.ClosedDate)
}

// TestSpecFinished_LeavesUnwrittenSpecAlone writes a scaffold-only spec (still
// wearing an in-progress frontmatter block from a hypothetical new() call) and
// asserts finished() does NOT transition it to completed — the still-scaffold
// gate should prevent the close.
func TestSpecFinished_LeavesUnwrittenSpecAlone(t *testing.T) {
	tmp := t.TempDir()
	st := store.NewFileStore(tmp, "project")
	cfg := workflow.Config{Command: "spektacular", SpecDir: "specs"}

	scaffold, err := stepkit.RenderTemplate("scaffold/spec.md", map[string]any{"name": "fixture"})
	require.NoError(t, err)
	created := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	seed, err := metadata.Render(metadata.Metadata{
		CreatedDate: created,
		Status:      metadata.StatusInProgress,
	}, []byte(scaffold))
	require.NoError(t, err)
	require.NoError(t, st.Write(SpecFilePath("specs", "fixture"), seed))

	data := &testData{values: map[string]any{"name": "fixture"}}
	writer := &captureWriter{}

	_, err = finished()(data, writer, st, cfg)
	require.NoError(t, err, "finished() must not crash when the spec is still the scaffold")

	// The stored bytes must be byte-identical to the seed — no metadata mutation.
	raw, err := st.Read(SpecFilePath("specs", "fixture"))
	require.NoError(t, err)
	require.Equal(t, string(seed), string(raw), "still-scaffold gate must prevent any write on finished()")

	meta, _, err := metadata.Split(raw)
	require.NoError(t, err)
	require.NotNil(t, meta)
	require.Equal(t, metadata.StatusInProgress, meta.Status, "still-scaffold artifact must remain in-progress")
	require.True(t, meta.ClosedDate.IsZero(), "still-scaffold artifact must not gain a closed_date")
}

// TestSpecStillScaffold_FrontmatterTolerant confirms specStillScaffold strips a
// leading frontmatter block before comparing against the rendered scaffold so
// the wrapped and bare scaffolds are treated equivalently, and a filled body
// wrapped in frontmatter is still reported as no-longer-scaffold.
func TestSpecStillScaffold_FrontmatterTolerant(t *testing.T) {
	const specName = "fixture"
	cfg := workflow.Config{Command: "spektacular", SpecDir: "specs"}

	scaffold, err := stepkit.RenderTemplate("scaffold/spec.md", map[string]any{"name": specName})
	require.NoError(t, err)
	scaffoldBytes := []byte(scaffold)

	fm := metadata.Metadata{
		CreatedDate: time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC),
		Status:      metadata.StatusInProgress,
	}

	t.Run("scaffold_with_frontmatter_is_still_scaffold", func(t *testing.T) {
		st := store.NewFileStore(t.TempDir(), "project")
		wrapped, err := metadata.Render(fm, scaffoldBytes)
		require.NoError(t, err)
		require.NoError(t, st.Write(SpecFilePath(cfg.SpecDir, specName), wrapped))

		still, err := specStillScaffold(st, cfg, specName)
		require.NoError(t, err)
		require.True(t, still, "scaffold wrapped in frontmatter must be reported as still scaffold")
	})

	t.Run("filled_body_with_frontmatter_is_not_scaffold", func(t *testing.T) {
		st := store.NewFileStore(t.TempDir(), "project")
		filled := append([]byte{}, scaffoldBytes...)
		filled = append(filled, "\n\n## Real content\n"...)
		wrapped, err := metadata.Render(fm, filled)
		require.NoError(t, err)
		require.NoError(t, st.Write(SpecFilePath(cfg.SpecDir, specName), wrapped))

		still, err := specStillScaffold(st, cfg, specName)
		require.NoError(t, err)
		require.False(t, still, "a filled body wrapped in frontmatter must not be reported as still scaffold")
	})
}

// TestNewStep_InstructionIncludesCaveat verifies that the instruction includes
// the caveat to skip if no meaningful context exists (Phase 2.1 acceptance criteria).
func TestNewStep_InstructionIncludesCaveat(t *testing.T) {
	tmp := t.TempDir()
	
	// Create .spektacular directory
	spektacularDir := filepath.Join(tmp, ".spektacular")
	require.NoError(t, os.MkdirAll(spektacularDir, 0755))
	
	// Change to the temp directory so relative path resolution works
	origWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmp))
	defer os.Chdir(origWd)
	
	data := &testData{values: map[string]any{"name": "fixture"}}
	writer := &captureWriter{}
	st := store.NewFileStore(tmp, "project")
	
	_, err = new()(data, writer, st, workflow.Config{Command: "spektacular", SpecDir: "specs"})
	require.NoError(t, err)
	
	instruction := writer.result.Instruction
	require.Contains(t, instruction, "no meaningful context", "instruction should include caveat about skipping if no context")
}
