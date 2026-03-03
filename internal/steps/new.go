package steps

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/runner"
	"github.com/jumppad-labs/spektacular/internal/spec"
	"github.com/jumppad-labs/spektacular/internal/tui"
)

// User prompt for each section. Each prompt is self-contained: it tells the agent
// what file to read, what question to ask (including the exact HTML comment format),
// how to validate the response, what to write, and when to output <!-- FINISHED -->.

var overviewMsg = `The spec file has been created at '%s'. Read it so you understand the template structure.

Your task for this session: collect the **Overview** section only.

Ask the user this question:

<!--QUESTION:{"questions":[{"question":"Describe this feature in 2-3 sentences:\n• What is being built?\n• What problem does it solve?\n• Who benefits?\n\nBe specific — avoid generic phrases like 'improve the experience'.","header":"Overview","type":"text"}]}-->

If the response is too vague (e.g. 'make it better', 'add search'), ask one clarifying question using the same format. Maximum one clarification round.

Once you have the overview, edit the Overview section of the spec file with their response. Then output:

<!-- FINISHED -->`

var requirementsMsg = `The spec file is at '%s'. Read it.

Your task for this session: collect the **Requirements** section only.

Ask the user this question:

<!--QUESTION:{"questions":[{"question":"List the specific, testable behaviours this feature must deliver.\n\nUse active voice:\n• 'Users can...'\n• 'The system must...'\n\nEach item should be independently verifiable. One behaviour per line.","header":"Requirements","type":"text"}]}-->

If the response is too vague, ask one clarifying question. Maximum one clarification round.

Format the requirements as a markdown checklist and write them to the Requirements section:
- [ ] **Title** — description

Then output:

<!-- FINISHED -->`

var acMsg = `The spec file is at '%s'. Read it to find all requirements in the Requirements section.

Your task for this session: collect **Acceptance Criteria** for every requirement, one at a time.

**Step 1** — Enumerate the requirements. Write them out before asking anything:

> I found [N] requirements:
> 1. [Title]: [text]
> 2. [Title]: [text]
> ...
> Let's define acceptance criteria for each one.

Then ask about requirement 1 and STOP.

**Step 2** — One requirement per turn, no exceptions. For each requirement ask:

<!--QUESTION:{"questions":[{"question":"Requirement [N] of [total]: [Title]\n[requirement text]\n\nWhat is the pass/fail condition that proves this is done?\n\nA good criterion:\n• Describes an observable outcome\n• Passes or fails — no subjective judgment\n• Is traceable to this requirement\n\nExample: 'When X happens, Y is visible / saved / returned.'","header":"AC: [Title]","type":"text"}]}-->

After the question: STOP. Do not write about the next requirement.

**Step 3** — Validate before moving on:
- Clear and binary → "Got it." Move to the next requirement.
- Too vague ("it works") → ask: "What exactly would you observe? How do you distinguish pass from fail?" Re-ask same question. Stop.
- After 2 clarification rounds, accept and move on.

**Step 4** — After the last requirement: write all criteria to the Acceptance Criteria section, then output:

<!-- FINISHED -->`

var constraintsMsg = `The spec file is at '%s'. Read it.

Your task for this session: collect the **Constraints** section only.

Ask the user this question:

<!--QUESTION:{"questions":[{"question":"Are there any hard constraints or boundaries the solution must operate within?\n\nExamples:\n• Must integrate with the existing authentication system\n• Cannot introduce breaking changes to the public API\n• Must support the current minimum supported runtime versions\n\nLeave blank if there are no constraints.","header":"Constraints","type":"text"}]}-->

Write their response to the Constraints section. If blank, write 'None.' Then output:

<!-- FINISHED -->`

var technicalApproachMsg = `The spec file is at '%s'. Read it.

Your task for this session: collect the **Technical Approach** section only.

Ask the user this question:

<!--QUESTION:{"questions":[{"question":"Do you have any technical direction already decided?\n\nFor example:\n• Key architectural decisions already made\n• Preferred patterns or technologies\n• Integration points with existing systems\n• Known risks or areas of uncertainty\n\nLeave blank to let the planner propose the approach.","header":"Technical Approach","type":"text"}]}-->

Write their response to the Technical Approach section. If blank, write 'None.' Then output:

<!-- FINISHED -->`

var successMetricsMsg = `The spec file is at '%s'. Read it.

Your task for this session: collect the **Success Metrics** section only.

Ask the user this question:

<!--QUESTION:{"questions":[{"question":"How will you know this feature is working well after delivery?\n\nBe specific:\n• Quantitative: 'p99 latency < 200ms', 'error rate < 0.1%'\n• Behavioural: 'users complete the flow without support intervention'\n\nLeave blank if not applicable.","header":"Success Metrics","type":"text"}]}-->

Write their response to the Success Metrics section. If blank, write 'None.' Then output:

<!-- FINISHED -->`

var nonGoalsMsg = `The spec file is at '%s'. Read it.

Your task for this session: collect the **Non-Goals** section only.

Ask the user this question:

<!--QUESTION:{"questions":[{"question":"What is explicitly OUT of scope for this feature?\n\nExamples:\n• 'Mobile support is out of scope (tracked in #456)'\n• 'Internationalisation will be addressed in a follow-up spec'\n\nLeave blank if there are no explicit exclusions.","header":"Non-Goals","type":"text"}]}-->

Write their response to the Non-Goals section. If blank, write 'None.' Then output:

<!-- FINISHED -->`

// SpecCreatorWorkflow returns the TUI workflow for interactively creating a spec file.
// The workflow runs one step per spec section.
func SpecCreatorWorkflow(name, projectPath string, cfg config.Config) tui.Workflow {
	specPath := filepath.Join(projectPath, ".spektacular", "specs", name+".md")

	logFile := ""
	if cfg.Debug.Enabled && cfg.Debug.LogDir != "" {
		logDir := filepath.Join(projectPath, cfg.Debug.LogDir)
		_ = os.MkdirAll(logDir, 0755)
		logFile = filepath.Join(logDir, time.Now().Format("2006-01-02_15-04-05")+"_new-spec.log")
	}

	return tui.Workflow{
		LogFile: logFile,
		Preamble: "## Creating spec: " + name + "\n\n" +
			"I'll guide you through **7 sections** to build a complete specification. " +
			"Answer each question when prompted — the spec file is updated as we go.\n\n" +
			"**Sections:** Overview → Requirements → Acceptance Criteria → Constraints → Technical Approach → Success Metrics → Non-Goals",
		Steps: []tui.WorkflowStep{
			overviewStep(specPath),
			requirementsStep(specPath),
			acStep(specPath),
			constraintsStep(specPath),
			technicalApproachStep(specPath),
			successMetricsStep(specPath),
			nonGoalsStep(specPath),
		},
		OnDone: func() (string, error) {
			return specPath, nil
		},
	}
}

func overviewStep(specPath string) tui.WorkflowStep {
	userPrompt := runner.BuildPromptWithHeader(fmt.Sprintf(overviewMsg, specPath), "Overview")
	systemPrompt := spec.LoadAgentSystemPrompt()

	return tui.WorkflowStep{
		Name:        "overview",
		StatusLabel: "collecting overview",
		BuildRunOptions: func(cfg config.Config, cwd string) (runner.RunOptions, error) {
			if err := spec.InitTemplate(specPath); err != nil {
				return runner.RunOptions{}, err
			}
			return runner.RunOptions{
				Prompts: runner.Prompts{User: userPrompt, System: systemPrompt},
				Config:  cfg,
				CWD:     cwd,
				Model:   "claude-haiku-4-5-20251001",
			}, nil
		},
	}
}

func requirementsStep(specPath string) tui.WorkflowStep {
	userPrompt := runner.BuildPromptWithHeader(fmt.Sprintf(requirementsMsg, specPath), "Requirements")
	systemPrompt := spec.LoadAgentSystemPrompt()

	return tui.WorkflowStep{
		Name:        "requirements",
		StatusLabel: "collecting requirements",
		BuildRunOptions: func(cfg config.Config, cwd string) (runner.RunOptions, error) {
			return runner.RunOptions{
				Prompts: runner.Prompts{User: userPrompt, System: systemPrompt},
				Config:  cfg,
				CWD:     cwd,
				Model:   "claude-haiku-4-5-20251001",
			}, nil
		},
	}
}

func acStep(specPath string) tui.WorkflowStep {
	userPrompt := runner.BuildPromptWithHeader(fmt.Sprintf(acMsg, specPath), "Acceptance Criteria")
	systemPrompt := spec.LoadAgentSystemPrompt()

	return tui.WorkflowStep{
		Name:        "acceptance-criteria",
		StatusLabel: "collecting acceptance criteria",
		BuildRunOptions: func(cfg config.Config, cwd string) (runner.RunOptions, error) {
			return runner.RunOptions{
				Prompts: runner.Prompts{User: userPrompt, System: systemPrompt},
				Config:  cfg,
				CWD:     cwd,
				Model:   "claude-haiku-4-5-20251001",
			}, nil
		},
	}
}

func constraintsStep(specPath string) tui.WorkflowStep {
	userPrompt := runner.BuildPromptWithHeader(fmt.Sprintf(constraintsMsg, specPath), "Constraints")
	systemPrompt := spec.LoadAgentSystemPrompt()

	return tui.WorkflowStep{
		Name:        "constraints",
		StatusLabel: "collecting constraints",
		BuildRunOptions: func(cfg config.Config, cwd string) (runner.RunOptions, error) {
			return runner.RunOptions{
				Prompts: runner.Prompts{User: userPrompt, System: systemPrompt},
				Config:  cfg,
				CWD:     cwd,
				Model:   "claude-haiku-4-5-20251001",
			}, nil
		},
	}
}

func technicalApproachStep(specPath string) tui.WorkflowStep {
	userPrompt := runner.BuildPromptWithHeader(fmt.Sprintf(technicalApproachMsg, specPath), "Technical Approach")
	systemPrompt := spec.LoadAgentSystemPrompt()

	return tui.WorkflowStep{
		Name:        "technical-approach",
		StatusLabel: "collecting technical approach",
		BuildRunOptions: func(cfg config.Config, cwd string) (runner.RunOptions, error) {
			return runner.RunOptions{
				Prompts: runner.Prompts{User: userPrompt, System: systemPrompt},
				Config:  cfg,
				CWD:     cwd,
				Model:   "claude-haiku-4-5-20251001",
			}, nil
		},
	}
}

func successMetricsStep(specPath string) tui.WorkflowStep {
	userPrompt := runner.BuildPromptWithHeader(fmt.Sprintf(successMetricsMsg, specPath), "Success Metrics")
	systemPrompt := spec.LoadAgentSystemPrompt()

	return tui.WorkflowStep{
		Name:        "success-metrics",
		StatusLabel: "collecting success metrics",
		BuildRunOptions: func(cfg config.Config, cwd string) (runner.RunOptions, error) {
			return runner.RunOptions{
				Prompts: runner.Prompts{User: userPrompt, System: systemPrompt},
				Config:  cfg,
				CWD:     cwd,
				Model:   "claude-haiku-4-5-20251001",
			}, nil
		},
	}
}

func nonGoalsStep(specPath string) tui.WorkflowStep {
	userPrompt := runner.BuildPromptWithHeader(fmt.Sprintf(nonGoalsMsg, specPath), "Non-Goals")
	systemPrompt := spec.LoadAgentSystemPrompt()

	return tui.WorkflowStep{
		Name:        "non-goals",
		StatusLabel: "collecting non-goals",
		BuildRunOptions: func(cfg config.Config, cwd string) (runner.RunOptions, error) {
			return runner.RunOptions{
				Prompts: runner.Prompts{User: userPrompt, System: systemPrompt},
				Config:  cfg,
				CWD:     cwd,
				Model:   "claude-haiku-4-5-20251001",
			}, nil
		},
	}
}

// stripExt removes the file extension from a filename.
func stripExt(name string) string {
	ext := filepath.Ext(name)
	if ext == "" {
		return name
	}
	return name[:len(name)-len(ext)]
}
