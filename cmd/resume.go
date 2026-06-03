package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jumppad-labs/spektacular/internal/output"
	"github.com/jumppad-labs/spektacular/internal/stepkit"
	"github.com/jumppad-labs/spektacular/internal/workflow"
)

// ResumeReport is the output a `new` command emits when it detects an
// in-progress workflow instead of starting a fresh one. It is the shared shape
// returned by all three kinds (spec/plan/implement) so the driving agent reacts
// identically regardless of which kind is in progress — including the cross-kind
// case, where Kind is read straight from the stored state.
type ResumeReport struct {
	Resumable   bool   `json:"resumable"`    // always true when a ResumeReport is emitted
	Kind        string `json:"kind"`         // in-progress workflow's kind (from state)
	Name        string `json:"name"`         // in-progress workflow's instance name
	CurrentStep string `json:"current_step"` // step the workflow stopped on
	Instruction string `json:"instruction"`  // rendered resume-prompt template
}

// resumeInstruction renders the shared resume-prompt template into the
// instruction carried by a ResumeReport. It tells the driving agent to ask the
// user resume-vs-new and embeds both follow-up commands: resume via
// `<command> <kind> goto` on the current step, or start fresh via
// `<command> <kind> new --force`.
//
// command is the CLI invocation prefix (workflow.Config.Command), rendered into
// the template via {{config.command}} to match the convention used by every
// other runtime-rendered step template.
func resumeInstruction(command, kind, name, currentStep string) (string, error) {
	return stepkit.RenderTemplate("steps/resume.md", map[string]any{
		"config":       map[string]any{"command": command},
		"kind":         kind,
		"name":         name,
		"current_step": currentStep,
	})
}

// detectInProgress loads the persisted workflow state at statePath and returns
// it only when it represents an in-progress (resumable) workflow. It returns
// (nil, nil) when no state file exists or the state is not in progress (e.g.
// finished), and an error only when the file exists but cannot be read or
// parsed. It never mutates anything on disk.
func detectInProgress(statePath string) (*workflow.State, error) {
	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	var state workflow.State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parsing state file: %w", err)
	}
	if !state.InProgress() {
		return nil, nil
	}
	return &state, nil
}

// resumeOrClear is the shared prologue for the three `new` commands. It decides
// what to do with any existing workflow state before a fresh workflow starts:
//
//   - force=true        → remove the state file and proceed fresh (handled=false).
//   - no in-progress    → remove any stale state file and proceed fresh (handled=false).
//   - in-progress, kind → write a ResumeReport (using the stored kind, so a
//     cross-kind workflow is surfaced) and return handled=true; the caller
//     returns immediately, leaving state and artifacts untouched.
//   - in-progress, no kind → return an error; a kind-less (pre-feature) state
//     is never clobbered or guessed at.
//
// command is the CLI invocation prefix rendered into the resume instruction.
func resumeOrClear(cmd *cobra.Command, statePath, command string, force bool) (handled bool, err error) {
	if force {
		_ = os.Remove(statePath)
		return false, nil
	}

	state, err := detectInProgress(statePath)
	if err != nil {
		return false, err
	}
	if state == nil {
		_ = os.Remove(statePath)
		return false, nil
	}
	if state.Kind == "" {
		return false, fmt.Errorf("existing workflow state at %s has no kind marker; run with --force to start fresh, or finish the existing workflow with `goto`", statePath)
	}

	name, _ := state.Data["name"].(string)
	instruction, err := resumeInstruction(command, state.Kind, name, state.CurrentStep)
	if err != nil {
		return false, err
	}

	out := output.New(cmd.OutOrStdout(), globalFields)
	return true, out.WriteResult(ResumeReport{
		Resumable:   true,
		Kind:        state.Kind,
		Name:        name,
		CurrentStep: state.CurrentStep,
		Instruction: instruction,
	})
}
