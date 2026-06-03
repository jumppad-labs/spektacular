package cmd

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResumeReport_JSONCarriesWorkflowIdentityAndInstruction(t *testing.T) {
	instruction, err := resumeInstruction("spektacular", "spec", "000024_resume", "overview")
	require.NoError(t, err)
	require.NotEmpty(t, instruction)

	report := ResumeReport{
		Resumable:   true,
		Kind:        "spec",
		Name:        "000024_resume",
		CurrentStep: "overview",
		Instruction: instruction,
	}

	encoded, err := json.Marshal(report)
	require.NoError(t, err)
	out := string(encoded)

	require.Contains(t, out, `"resumable":true`)
	require.Contains(t, out, `"kind":"spec"`)
	require.Contains(t, out, `"name":"000024_resume"`)
	require.Contains(t, out, `"current_step":"overview"`)
	require.Contains(t, out, `"instruction":`)

	var roundTrip ResumeReport
	require.NoError(t, json.Unmarshal(encoded, &roundTrip))
	require.NotEmpty(t, roundTrip.Instruction)
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
