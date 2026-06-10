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
	Resumable     bool   `json:"resumable"`                // always true when a ResumeReport is emitted
	Kind          string `json:"kind"`                     // in-progress workflow's kind (from state)
	RequestedKind string `json:"requested_kind,omitempty"` // kind the user tried to run; set only on a cross-kind mismatch
	Name          string `json:"name"`                     // in-progress workflow's instance name
	CurrentStep   string `json:"current_step"`             // step the workflow stopped on
	Instruction   string `json:"instruction"`              // rendered resume-prompt template
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

// mismatchInstruction renders the cross-kind resume-prompt template. It is used
// when the in-progress workflow's kind differs from the kind the user is trying
// to run (e.g. running a spec command while a plan workflow is in progress). The
// instruction tells the agent a different-kind workflow is in progress and gives
// both follow-ups: continue it with its own skill (`<command> <kind> goto`), or
// overwrite it and start the requested kind (`<command> <requestedKind> new
// --force`).
func mismatchInstruction(command, kind, requestedKind, name, currentStep string) (string, error) {
	return stepkit.RenderTemplate("steps/resume_mismatch.md", map[string]any{
		"config":         map[string]any{"command": command},
		"kind":           kind,
		"requested_kind": requestedKind,
		"name":           name,
		"current_step":   currentStep,
	})
}

// emitResumeReport writes the appropriate ResumeReport for an in-progress
// workflow. When the stored kind matches expectedKind it renders the normal
// same-kind resume prompt; when it differs it renders the cross-kind mismatch
// prompt and sets RequestedKind so the caller can see the mismatch in the
// structured output as well as the instruction.
func emitResumeReport(cmd *cobra.Command, command, expectedKind string, state *workflow.State) error {
	name, _ := state.Data["name"].(string)
	out := output.New(cmd.OutOrStdout(), globalFields)

	if state.Kind != expectedKind {
		instruction, err := mismatchInstruction(command, state.Kind, expectedKind, name, state.CurrentStep)
		if err != nil {
			return err
		}
		return out.WriteResult(ResumeReport{
			Resumable:     true,
			Kind:          state.Kind,
			RequestedKind: expectedKind,
			Name:          name,
			CurrentStep:   state.CurrentStep,
			Instruction:   instruction,
		})
	}

	instruction, err := resumeInstruction(command, state.Kind, name, state.CurrentStep)
	if err != nil {
		return err
	}
	return out.WriteResult(ResumeReport{
		Resumable:   true,
		Kind:        state.Kind,
		Name:        name,
		CurrentStep: state.CurrentStep,
		Instruction: instruction,
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
//   - in-progress, same kind → write a same-kind resume ResumeReport and return
//     handled=true; the caller returns immediately, leaving state untouched.
//   - in-progress, different kind → write a cross-kind mismatch ResumeReport
//     (continue that kind's workflow with its skill, or overwrite with the
//     requested kind via --force) and return handled=true. The in-progress
//     workflow is never silently resumed or clobbered.
//   - in-progress, no kind → return an error; a kind-less (pre-feature) state
//     is never clobbered or guessed at.
//
// expectedKind is the kind of the `new` command being run ("spec"/"plan"/
// "implement"); command is the CLI invocation prefix rendered into the
// instruction.
func resumeOrClear(cmd *cobra.Command, statePath, command, expectedKind string, force bool) (handled bool, err error) {
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

	return true, emitResumeReport(cmd, command, expectedKind, state)
}

// guardKind is the shared prologue for the `goto` and `status` commands. Those
// commands operate on the persisted state directly, so they must refuse to act
// on a workflow whose kind differs from the command's own kind — otherwise a
// `spec goto` would apply spec steps to a plan's state. It returns handled=true
// (after writing a cross-kind mismatch ResumeReport) when a different-kind
// workflow is in progress; the caller must then return immediately. A same-kind
// in-progress workflow (the normal target), a finished workflow, a kind-less
// legacy state, or no state at all all return handled=false so the caller
// proceeds as before.
func guardKind(cmd *cobra.Command, statePath, command, expectedKind string) (handled bool, err error) {
	state, err := detectInProgress(statePath)
	if err != nil {
		return false, err
	}
	if state == nil || state.Kind == "" || state.Kind == expectedKind {
		return false, nil
	}
	return true, emitResumeReport(cmd, command, expectedKind, state)
}
