package cmd

import (
	"encoding/json"
	"testing"

	"github.com/jumppad-labs/spektacular/internal/output"
	"github.com/jumppad-labs/spektacular/internal/workflow"
	"github.com/stretchr/testify/require"
)

// TestEmitResumeReport_JSONCarriesWorkflowIdentityAndInstruction asserts that
// a same-kind in-progress workflow produces the shared ErrorResponse shape
// (error=true, workflow_in_progress code, the workflow's name as Resource,
// its current step as State.Current, and the rendered resume instruction as
// NextAction) and that this shape round-trips through JSON, since it flows
// all the way to stdout via that encoding in production.
func TestEmitResumeReport_JSONCarriesWorkflowIdentityAndInstruction(t *testing.T) {
	instruction, err := resumeInstruction("spektacular", "spec", "000024_resume", "overview")
	require.NoError(t, err)
	require.NotEmpty(t, instruction)

	state := &workflow.State{
		Kind:        "spec",
		CurrentStep: "overview",
		Data:        map[string]any{"name": "000024_resume"},
	}

	reportErr := emitResumeReport("spektacular", "spec", state)
	require.Error(t, reportErr)
	er, ok := reportErr.(*output.ErrorResponse)
	require.True(t, ok, "emitResumeReport must return an *output.ErrorResponse")

	require.True(t, er.IsError)
	require.Equal(t, "workflow_in_progress", er.Code)
	require.Equal(t, "000024_resume", er.Resource)
	require.NotNil(t, er.State)
	require.Equal(t, "overview", er.State.Current)
	require.Equal(t, instruction, er.NextAction)

	encoded, err := json.Marshal(er)
	require.NoError(t, err)
	out := string(encoded)

	require.Contains(t, out, `"error":true`)
	require.Contains(t, out, `"code":"workflow_in_progress"`)
	require.Contains(t, out, `"resource":"000024_resume"`)
	require.Contains(t, out, `"current":"overview"`)
	require.Contains(t, out, `"next_action":`)

	var roundTrip output.ErrorResponse
	require.NoError(t, json.Unmarshal(encoded, &roundTrip))
	require.NotEmpty(t, roundTrip.NextAction)
}

func TestResumeInstruction_AsksResumeVsNewWithBothCommands(t *testing.T) {
	out, err := resumeInstruction("spektacular", "spec", "000024_resume", "overview")
	require.NoError(t, err)

	require.NotContains(t, out, "{{")

	require.Contains(t, out, "spec")
	require.Contains(t, out, "000024_resume")
	require.Contains(t, out, "overview")

	require.Contains(t, out, `spektacular spec goto --data '{"step":"overview"}'`)
	require.Contains(t, out, "spektacular spec new --force")
	require.Contains(t, out, ".spektacular/context.md")

	require.Contains(t, out, "resume")
	require.Contains(t, out, "new")
}

func TestResumeInstruction_InterpolatesAcrossKinds(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		kind        string
		instance    string
		currentStep string
		wantGoto    string
		wantNew     string
	}{
		{
			name:        "spec",
			command:     "spektacular",
			kind:        "spec",
			instance:    "000024_resume",
			currentStep: "overview",
			wantGoto:    `spektacular spec goto --data '{"step":"overview"}'`,
			wantNew:     "spektacular spec new --force",
		},
		{
			name:        "plan",
			command:     "spek",
			kind:        "plan",
			instance:    "000024_resume",
			currentStep: "tasks",
			wantGoto:    `spek plan goto --data '{"step":"tasks"}'`,
			wantNew:     "spek plan new --force",
		},
		{
			name:        "implement",
			command:     "spektacular",
			kind:        "implement",
			instance:    "000024_resume",
			currentStep: "execute",
			wantGoto:    `spektacular implement goto --data '{"step":"execute"}'`,
			wantNew:     "spektacular implement new --force",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := resumeInstruction(tt.command, tt.kind, tt.instance, tt.currentStep)
			require.NoError(t, err)

			require.NotContains(t, out, "{{")
			require.Contains(t, out, tt.kind)
			require.Contains(t, out, tt.instance)
			require.Contains(t, out, tt.currentStep)
			require.Contains(t, out, tt.wantGoto)
			require.Contains(t, out, tt.wantNew)
			require.Contains(t, out, ".spektacular/context.md")
		})
	}
}
