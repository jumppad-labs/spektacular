# Plan: 000024_resume

<!-- Metadata -->
<!-- Created: 2026-06-02T10:04:27Z -->
<!-- Commit: df74be5d3da24f8d678913315a91cff1c5239a1f -->
<!-- Branch: main -->
<!-- Repository: git@github.com:jumppad-labs/spektacular.git -->

## Overview

This plan makes an interrupted specification, plan, or implementation workflow resumable. Today every entry command (`spec new`, `plan new`, `implement new`) wipes existing workflow state before starting, so a session ended by a crash, reboot, or close loses its half-finished progress. With this change, re-running an entry command detects an in-progress workflow and reports it instead of destroying it, the driving agent can offer the user resume-or-start-new, and a resumed workflow continues from the exact step it stopped on — with completed work and the agent's accumulated working context intact. Anyone driving these workflows benefits: progress survives interruption rather than restarting from scratch.

## Architecture & Design Decisions

The workflow engine is already a single shared FSM (`internal/workflow`) that all three kinds drive by injecting their own step lists, and it already persists `current_step`, `completed_steps`, and `data` to `.spektacular/state.json` on every transition. The per-step persistence the spec calls for therefore already exists; the real blocker is that each `new` command unconditionally removes that state file before starting. The chosen direction keeps the single shared state file (gaining one new `kind` field) and turns "delete then start" into "detect then report": without a flag, `new` loads any existing state and, if it represents an unfinished workflow, returns a resume report and changes nothing on disk; with `--force` it overwrites and starts fresh as today. One workflow is in progress at a time; recovery of anything overwritten is handled out-of-band by git, not by an in-tool history feature.

Resume rides entirely on one repurposed branch in the engine rather than a new command surface. `current_step` is the step that was *in progress* when interrupted — it is not yet in `completed_steps`, and its work was never persisted — so resuming must re-present that step, which is exactly spec AC #3's "advance to step N+1" (the first not-yet-completed step). Because jumping to the already-current step is a no-op today, that no-op branch is changed to re-render: it re-invokes the current step's callback to re-emit its instruction without an FSM transition. The driving agent therefore resumes with the same `goto <current_step>` command it already uses. This is safe because step callbacks are render-only with at most read-only store access (see `research.md#chosen-approach--evidence`).

A new `kind` marker (`spec`/`plan`/`implement`) is recorded in state, stamped once when the workflow is created, so a resume report can identify what is in progress and point the agent at the correct kind's command — including the cross-kind case, where `plan new` surfaces an in-progress spec by simply echoing the stored `kind`. This is a deliberate reinterpretation of spec AC #8 (per-kind independence): with a single global slot, an in-progress workflow of another kind is *surfaced* rather than silently ignored or clobbered. The user accepted this trade-off as safer than the letter of the criterion. There is no inference of kind for older, kind-less state files: a state file with no `kind` marker is rejected with a clear error rather than guessed at — the user can start fresh with `--force` or finish the existing workflow via `goto` (which needs no kind). Cold-process resume is completed by a git-tracked `.spektacular/context.md` working file that the agent refreshes at each step and reads back on resume — the one piece of workflow state deliberately left agent-owned rather than CLI-managed.

This direction was chosen over per-kind-keyed state files and per-(kind, name) state (which would let multiple workflows coexist and satisfy AC #8 literally) because the user explicitly chose a single state file with git as the recovery mechanism; those alternatives and their rejection rationale — along with the rejected "infer legacy kind" option — are recorded in `research.md#alternatives-considered-and-rejected`.

## Component Breakdown

- **Persisted workflow state** — Gains a `kind` marker identifying which workflow the single persisted slot belongs to. Stamped when state is first created. A state file that predates this change and carries no marker is treated as an error by detection, not migrated. Load/save is otherwise unchanged.
- **Workflow engine — current-step re-render** — The engine's "jump to a step" operation, when asked to jump to the step already current, currently does nothing; it is repurposed to re-emit that step's instruction without advancing the state machine. This is the entire resume mechanism: an interrupted step is re-presented and no completed step is disturbed. Owns the guarantee that re-rendering has no side effects beyond producing the instruction.
- **Resume-detection helper (shared)** — One helper used by all three `new` commands: loads state, classifies it as absent / finished / in-progress, and on in-progress builds the resume report from the stored `kind`. Centralises the "detect instead of delete" logic so the three commands do not duplicate it and behaviour stays identical across kinds. A kind-less in-progress state is surfaced as an error.
- **`new` command behaviour (spec / plan / implement)** — Each `new` gains a force flag. Without it, the command consults the detection helper and either emits a resume report (mutating nothing — no artifact, no state change) or starts fresh as today. With the flag, it overwrites any existing state and starts fresh. Artifact creation happens only on a fresh start.
- **Resume report (output contract)** — The JSON the driving agent consumes when an in-progress workflow is found: a resumable flag, the in-progress kind and name, the current step, and a rendered instruction. A shared shape so all three kinds report identically.
- **Resume-prompt template** — A mustache template rendered into the resume report's instruction. It embeds the current step and the two follow-up commands — resume via `goto <current_step>`, start fresh via `new --force` — and tells the agent to ask the user which to do and to re-read the working context before continuing.
- **Workflow context file** — A git-tracked `.spektacular/context.md` holding the agent's running working context, written and read directly by the agent via its own file tools (the one explicit exception to "state goes through the CLI"). Refreshed on every step, read back by resume to rehydrate a cold agent.
- **Step instruction templates (all three kinds)** — Each step template gains a short, uniform directive to refresh the working context when advancing.
- **Workflow skill playbooks (spek-new / spek-plan / spek-implement)** — Updated so the driving agent calls `new`, interprets a resume report, asks the user resume-vs-start-new, and follows up with `goto <current_step>` or `new --force`, replacing the current "inspect `state.json` for the active name" guidance.

## Data Structures & Interfaces

**`workflow.State` — new `Kind` field**
```go
type State struct {
    Kind           string         `json:"kind,omitempty"` // "spec" | "plan" | "implement"
    CurrentStep    string         `json:"current_step"`
    CompletedSteps []string       `json:"completed_steps"`
    CreatedAt      time.Time      `json:"created_at"`
    UpdatedAt      time.Time      `json:"updated_at"`
    Data           map[string]any `json:"data"`
}
```
`Kind` identifies which workflow the single persisted slot belongs to. `omitempty` keeps it absent on legacy files; an absent marker on an in-progress state is an error at detection time, not a value to be inferred. A predicate `(*State).InProgress() bool` ≡ `CurrentStep != "" && CurrentStep != "finished"` is the single definition of "resumable".

**`workflow.Config` — new `Kind` field**
```go
type Config struct {
    Command string
    Kind    string // stamped onto freshly-created State by New()
    DryRun  bool
    SpecDir string
    PlanDir string
}
```
The cmd layer already builds `Config` per kind; it now also sets `Kind`, which `New()` writes into fresh state.

**`workflow.Workflow.Goto` — extended semantics (no signature change)**
`Goto(name string) error`: when `name == Current()`, instead of returning nil it re-renders that step (re-invokes the step callback to emit its instruction; no FSM transition). Backed by a private `renderStep(name string) error`.

**Resume report — shared output contract**
```go
type ResumeReport struct {
    Resumable   bool   `json:"resumable"`    // always true when present
    Kind        string `json:"kind"`         // in-progress workflow's kind (from state)
    Name        string `json:"name"`
    CurrentStep string `json:"current_step"`
    Instruction string `json:"instruction"`  // rendered resume-prompt template
}
```
Returned by all three `new` commands when an in-progress workflow is detected, in place of the kind's normal `Result`. Carries `Kind` (read straight from state) so the agent resumes the correct kind even when surfaced cross-kind. The `new` output schema documents both shapes.

**Resume-prompt template** — a shared mustache template (`templates/steps/resume.md`) rendered with `{kind, name, current_step}` into `ResumeReport.Instruction`.

**`--force` flag** — a bool flag added to each `<kind> new`; when set, bypasses detection and overwrites state.

**No structure for `context.md`** — it is free-form markdown authored by the agent via its own file tools; the CLI defines no type for it.

## Implementation Detail

**Detect-instead-of-delete, factored into one shared helper.** The three `new` commands each duplicate the same "remove state, then construct workflow" prologue today. Rather than thread `--force` and detection through three near-identical copies, the detection logic lands in a single shared cmd-level helper that every `new` calls first. This delivers the feature and removes the existing triplication — each command shrinks to "consult the helper; if it reports in-progress, emit its resume report and return; otherwise proceed (or `--force` was set, so overwrite and proceed)." This follows the codebase's existing direction of pushing shared workflow logic into shared layers (the engine, `stepkit`).

**Resume is an engine affordance, not a new command surface.** The whole resume path rides on one repurposed branch in the engine: jumping to the already-current step re-renders rather than no-ops. No new subcommand, no new FSM concepts, no change to how steps are declared. Because step callbacks are already render-only with read-only store access, re-rendering needs no new "is this safe to re-run" contract.

**The `kind` marker is stamped once at creation; legacy state is not inferred.** New state gets its `kind` from `Config.Kind` at construction — the one place state is created. A pre-existing state file with no marker is not guessed at: detection rejects it with a clear error, leaving the user to `--force` a fresh start or finish the existing workflow via `goto`. There is no migration command, no version field, no fingerprint inference, and no flag day — which keeps the change small and avoids a heuristic that could misclassify state.

**Cross-kind reporting reads the stored kind.** Because every workflow created under this change records its `kind`, a `new` command can describe and point at an in-progress workflow of a different kind simply by echoing `state.Kind` into the resume report — no registry of other kinds' steps is needed.

**Working context is an agent-owned sidecar, deliberately outside the CLI.** `.spektacular/context.md` is the one piece of workflow state the CLI does not manage — the agent reads and writes it with its own file tools. This is a conscious exception to the "state goes through the CLI" convention: the file is a free-form rehydration scratchpad, git-tracked so it follows branches, and the CLI's only involvement is that step templates instruct the agent to refresh it and the resume prompt instructs the agent to read it. Keeping it out of the CLI avoids inventing a serialization contract for what is fundamentally prose.

**Template changes are mechanical and broad.** Every step instruction template across all three workflows gains a short, uniform context-refresh directive, and the skill playbooks change from "inspect `state.json` to find the active name" to "call `new`, react to a possible resume report." These are wide but shallow edits following the existing mustache-template and skill-playbook patterns.

## Dependencies

- **`internal/workflow`** — Owns `State`, `Config`, and the FSM engine. **Changes required**: add `Kind` to `State` and `Config`, add `(*State).InProgress()`, stamp `Kind` on fresh state, and repurpose the same-step `Goto` branch to re-render. The foundational change everything else builds on.
- **`internal/steps/{spec,plan,implement}`** — Each provides its kind's `Steps()` and result builder. **Changes required**: every step's instruction template gains the context-refresh directive. No structural change to the packages.
- **`cmd` (spec/plan/implement command groups)** — Owns `new`/`goto`/`status`/`steps` wiring, flag parsing, and `stateFilePath`. **Changes required**: add `--force`; replace the three `os.Remove`+construct prologues with the shared detection helper; host the `ResumeReport` output and the resume-prompt render. The bulk of the new code lands here.
- **`internal/stepkit`** — Renders step templates and builds results. **Likely change**: render the new shared resume-prompt template into `ResumeReport.Instruction`, reusing the existing mustache/embedded-FS path. No new contract.
- **`templates/` (embedded FS)** — Step instruction templates and skill playbooks. **Changes required**: add `resume.md`; add the context-refresh directive to every step template; rewrite the `spek-new`/`spek-plan`/`spek-implement` SKILL.md playbooks to react to the resume report. Wide but shallow.
- **`looplab/fsm`** — The external FSM library. **No change**: re-render sidesteps the FSM entirely (no X→X transition is attempted), so no new library capability is needed.
- **Planning dependencies** — None. This plan builds entirely on shipped behavior; no upstream spec or prior plan must land first.
- **No new external/runtime dependencies** — no new Go modules, services, or config. `.spektacular/context.md` is a plain git-tracked file requiring no new tooling.

## Testing Approach

Almost everything is exercised by **unit tests** at two layers, matching the existing split: engine-level tests in the `workflow` package (`testify-require`, `t.TempDir()` state files, rebuild-from-disk to simulate a fresh process) and command-level tests in the `cmd` package (drive `rootCmd` with `SetArgs`, `t.Chdir(t.TempDir())`, parse the JSON output — the pattern the spec/implement command tests already use). No new test framework or harness, and no end-to-end agent test: the agent-facing behaviour is the JSON contract, which the command tests assert directly.

Heaviest coverage goes to the engine resume affordance and the shared detection helper, because they are the load-bearing pieces reused across three kinds; the single shared helper means one set of behaviours is proven once and trusted for all kinds, with thin per-kind command tests confirming each wires it up.

The load-bearing assertions, in plain language:
- *Re-render, not no-op:* jumping to the already-current step emits that step's instruction and leaves `current_step`/`completed_steps` unchanged (replacing the existing "same-step goto is a silent no-op" assertion, an intentionally-changed contract).
- *Survives process exit:* after persisting state, a freshly-constructed workflow reading only the file resumes at the same step with completed work intact — proven by never reusing the in-memory workflow.
- *Detect, don't clobber:* `new` against an in-progress slot returns a resume report with the right kind/name/current step and preserves the captured work (current step, completed steps, data) byte-for-byte.
- *Force overwrites:* `new --force` against an in-progress slot starts a fresh workflow.
- *Clean start is silent:* `new` with no existing state produces the normal first-step result with no resumable flag.
- *Kind recorded:* fresh state carries its `kind`; a kind-less in-progress state file is rejected with a clear error rather than clobbered or guessed.
- *Cross-kind surfacing:* `plan new` against an in-progress spec reports `kind: "spec"` and points at the spec's current step.

Oracles stay independent: expected step names, the resumable flag, and error outcomes are hand-written constants, never derived at runtime from the subject's own `Steps()`. Deliberate gaps: no automated test for `.spektacular/context.md` content (agent-authored prose with no CLI contract), and the skill-playbook edits are covered only by the existing skill-install tests, not by behavioural tests.

## Milestones & Phases

### Milestone 1: Workflow state knows its kind and can be re-presented in place

**What changes**: Purely internal — no user-visible behaviour yet. The workflow engine learns three things it needs before resume can exist: persisted state records which kind of workflow it belongs to, the engine can tell whether a workflow is still in progress, and asking the engine to "go to" the step that is already current now re-emits that step's instruction instead of doing nothing. Isolating this lets the load-bearing engine change be proven by unit tests before any command depends on it.

#### - [x] Phase 1.1: Record workflow kind in state

Add a kind marker to the persisted workflow state and to the engine's runtime config, so freshly-created state records which workflow (spec, plan, or implement) it belongs to. Add a single predicate the rest of the feature uses to ask "is this workflow still in progress?". No behaviour changes for existing commands yet.

*Technical detail:* [context.md#phase-11](./context.md#phase-11-record-workflow-kind-in-state)

**Acceptance criteria**:

- [x] A newly created workflow's state file records its kind (`spec`/`plan`/`implement`).
- [x] The engine reports a workflow with an unfinished current step as in-progress, and one sitting on `finished` as not in-progress.
- [x] Existing spec/plan/implement commands behave exactly as before.

#### - [x] Phase 1.2: Re-render the current step on same-step goto

Change the engine so that asking to go to the step that is already current re-emits that step's instruction instead of silently doing nothing. This is the entire resume mechanism — it lets an interrupted step be presented again without advancing the state machine or disturbing completed steps.

*Technical detail:* [context.md#phase-12](./context.md#phase-12-re-render-the-current-step-on-same-step-goto)

**Acceptance criteria**:

- [x] `goto <current_step>` returns that step's full instruction.
- [x] After such a call, the current step and the list of completed steps are unchanged.
- [x] Forward `goto` and `next` behave exactly as before.

### Milestone 2: Re-running an entry command detects in-progress work instead of destroying it

**What changes**: Re-running `spec new` / `plan new` / `implement new` on a workflow that was interrupted no longer silently wipes it. Instead the command reports the in-progress workflow — its kind, name, and the step it stopped on — and changes nothing on disk, so the caller can decide what to do. Passing `--force` restores the old behaviour (overwrite and start fresh). With no workflow in progress, `new` starts fresh silently, exactly as today. At this point a workflow is fully resumable from the CLI: detect via `new`, continue via `goto <current_step>`.

#### - [x] Phase 2.1: Detection primitives and the resume report

Introduce the building blocks the entry commands need to report an in-progress workflow: the resume-report output shape and a resume-prompt template that tells the agent how to proceed. Nothing is wired into the commands yet.

*Technical detail:* [context.md#phase-21](./context.md#phase-21-detection-primitives-and-the-resume-report)

**Acceptance criteria**:

- [x] The resume report carries the in-progress workflow's kind, name, and current step, plus a rendered instruction.
- [x] The resume instruction tells the agent to ask resume-vs-new, and gives the resume (`goto <current_step>`) and start-new (`new --force`) commands.

#### - [x] Phase 2.2: Entry commands detect, report, and force

Rewire the three `new` commands so that, instead of unconditionally wiping state, they consult the shared detection helper: an in-progress workflow is reported (changing nothing on disk) and a clean slot starts fresh as today. A `--force` flag restores overwrite-and-start-fresh. A pre-existing state file with no kind marker is rejected with a clear error. The three near-identical prologues collapse into one shared helper.

*Technical detail:* [context.md#phase-22](./context.md#phase-22-entry-commands-detect-report-and-force)

**Acceptance criteria**:

- [x] Re-running `new` on an in-progress workflow returns the resume report and leaves the captured work (current step, completed steps, data) intact.
- [x] `new --force` on an in-progress workflow starts a fresh workflow.
- [x] `new` on a clean directory starts fresh with no resume report.
- [x] `plan new` while a spec is in progress reports the spec, and vice-versa.
- [x] A state file without a kind marker produces a clear error rather than being clobbered.

### Milestone 3: The agent prompts to resume and carries context across sessions

**What changes**: The end-to-end user experience lands. When the driving agent hits an in-progress workflow it asks the user whether to resume or start new, then either continues from the interrupted step or forces a fresh start. Every workflow step now refreshes a git-tracked `.spektacular/context.md` working file, and resume points the agent at it — so an agent starting cold in a brand-new process (after a crash, reboot, or branch switch) recovers its working context and picks the effort back up rather than restarting.

#### - [x] Phase 3.1: Step templates refresh the working context

Add a short, uniform directive to every workflow step instruction telling the agent to refresh the git-tracked `.spektacular/context.md` working file as it advances. This is what makes a cold resume possible — the agent's accumulated decisions and findings survive process exit on disk.

*Technical detail:* [context.md#phase-31](./context.md#phase-31-step-templates-refresh-the-working-context)

**Acceptance criteria**:

- [x] Every spec, plan, and implement step instruction asks the agent to refresh `.spektacular/context.md` before advancing.
- [x] The directive is consistent in wording across all three workflows.
- [x] The rendered instructions remain otherwise unchanged.

#### - [x] Phase 3.2: Resume-aware skill playbooks

Rewrite the spek-new, spek-plan, and spek-implement skill playbooks so the driving agent calls `new`, reacts to a resume report by asking the user resume-vs-new, and then either resumes (reading `.spektacular/context.md` and running `goto <current_step>`) or starts fresh (`new --force`). The current "inspect `state.json` to find the active name" guidance is removed.

*Technical detail:* [context.md#phase-32](./context.md#phase-32-resume-aware-skill-playbooks)

**Acceptance criteria**:

- [x] On a resume report, the agent prompts the user to resume or start new.
- [x] Choosing resume reads the working context file and continues from the interrupted step.
- [x] Choosing start-new forces a fresh workflow.
- [x] No skill playbook instructs the agent to read `state.json` directly.

## Open Questions

- **Are all step callbacks side-effect-free, so re-rendering the current step is safe?** The resume mechanism (Phase 1.2) re-invokes the current step's callback to re-emit its instruction. This is safe only if every step callback is render-only / read-only (renders a template, at most reads the store). The plan workflow's callbacks were confirmed read-only during discovery, but the spec and implement callbacks were not exhaustively audited. *Depends on:* the actual bodies of the spec/implement step callbacks, fully visible only while editing them. *What the implementer should do:* while implementing Phase 1.2, read every spec and implement step callback; if any performs a write (to the store, the artifact, or the state) as a side effect of rendering, **STOP and ask the user** before proceeding — re-render would otherwise repeat that side effect on resume, and the design assumes it will not.

## Out of Scope

- **Arbitrary rewind / step history.** Resume only continues an unfinished workflow from where it stopped. Jumping back to an earlier completed step or viewing version history is not provided. (Spec § Non-Goals.)
- **Multi-user / concurrent resume.** Coordinating two people resuming the same workflow at once is not handled. (Spec § Non-Goals.)
- **Cross-machine sync.** Resuming on a different machine than where the workflow was interrupted is not addressed; the git-tracked `context.md` + `state.json` travel via the repo, but no sync mechanism is built. (Spec § Non-Goals.)
- **Simultaneous in-progress workflows of different kinds.** A deliberate consequence of the single global state slot agreed during architecture: only one workflow (of any kind) is in progress at a time. Starting one kind while another is mid-flight surfaces the existing one rather than letting both coexist. This reinterprets spec AC #8 — accepted by the user. A future plan could move to per-kind-keyed state if simultaneous workflows are ever wanted.
- **In-tool recovery of overwritten state.** When the user chooses start-new (`--force`), the prior workflow's state is overwritten. Recovering it is done via git, not an in-tool undo/restore feature — a deliberate decision made during architecture.
- **Automatic migration of legacy kind-less state.** A `state.json` written before this change carries no `kind`; rather than infer it, detection errors and the user starts fresh (`--force`) or finishes via `goto`. No inference or upgrade path is built.
- **CLI management of `context.md`.** The working-context file is authored and read by the agent with its own file tools; the CLI defines no schema for it, does not write it, and does not validate it. Promoting it to a CLI-managed artifact is explicitly deferred.
- **Migrating away from the single shared state file.** This plan keeps one `.spektacular/state.json` (now with a `kind` field). Per-kind or per-(kind,name) storage layouts were considered and rejected; revisiting them is left to a future plan.

## Changelog

### 2026-06-02 — Phase 1.1: Record workflow kind in state

**What was done**: Added a `Kind` marker to the persisted workflow state and to the engine's runtime `Config`, so freshly-created state records which workflow (`spec`/`plan`/`implement`) it belongs to. `New()` stamps `cfg.Kind` onto fresh state in its create branch (the load-from-disk branch is untouched, so legacy kind-less files still parse). Added the `(*State).InProgress()` predicate (`CurrentStep != "" && CurrentStep != "finished"`) as the single definition of "resumable", and wired each `new` command's `workflow.Config` literal to set its kind.

**Deviations**: None.

**Files changed**:
- `internal/workflow/state.go`
- `internal/workflow/workflow.go`
- `cmd/spec.go`
- `cmd/plan.go`
- `cmd/implement.go`
- `internal/workflow/workflow_test.go`

**Discoveries**: `Kind` uses `json:"kind,omitempty"`, so old state files without the marker still unmarshal cleanly and round-trip — relied on by the legacy-state handling in later phases. Each command file has two `workflow.Config{...}` literals (one in the `new` handler, one in `goto`/`status`); only the `new`-handler literal is stamped, since `Kind` matters solely at creation and the `goto` handler preserves whatever the loaded state already carries. The `new`-handler literal is disambiguated from the `goto` one by the following `wf := workflow.New(steps, statePath, ...)` line (the `goto` handler uses `stateFilePath(dataDir)` instead).

### 2026-06-02 — Phase 1.2: Re-render the current step on same-step goto

**What was done**: Repurposed `Goto`'s same-step branch (`workflow.go`) from a silent no-op (`return nil`) into `return w.renderStep(name)`, and added the private `renderStep` method. `renderStep` finds the named step and re-invokes its callback (`step.Callback(w.data, w.out, w.store, w.cfg)`) to re-emit its instruction without firing an FSM transition, so `current_step` and `completed_steps` are left untouched — the entire resume mechanism. If the callback returns a non-empty next step it follows via `Goto`, mirroring `Next`.

**Deviations**: None to the code. The Phase 1.2 open question (re-render safety) was resolved: an audit of all spec and implement step callbacks found only one writer — the spec `new` bootstrap callback (`internal/steps/spec/steps.go:64-82`), which `st.Write`s the scaffold then returns `"overview"`. The user reviewed and chose to proceed as planned: `"new"` is never a normal resting `current_step` (the `spec new` command drives `start → new → overview` synchronously), and in the only edge that could rest there (a crash mid-bootstrap) re-render re-writes an idempotent scaffold and `renderStep`'s follow-next-step path self-heals forward to `overview`. The implement workflow's bootstrap (`newStep`) performs no write. No special-casing added.

**Files changed**:
- `internal/workflow/workflow.go`
- `internal/workflow/workflow_test.go`

**Discoveries**: `renderStep` deliberately follows a returned `nextStep` (via `Goto`) rather than blocking it — this is what makes the one writing step (spec `new`) self-heal on resume instead of getting stuck. The contract "re-render leaves current_step/completed_steps unchanged" holds for every *normal resting step* (all render-only, returning `""`); it intentionally does not hold for the bootstrap `new` step, which is never a normal resting point. The replaced `TestGotoSameStepIsNoop` was an intentional contract change — the new `TestGotoSameStepReRenders` needs a step set with a non-nil recording callback plus a capture `ResultWriter`, because the package's shared `testSteps` have nil callbacks.

### 2026-06-02 — Phase 2.1: Detection primitives and the resume report

**What was done**: Added the shared `ResumeReport` output shape (`cmd/resume.go`) — `{Resumable, Kind, Name, CurrentStep, Instruction}` with JSON tags — and `resumeInstruction(...)`, which renders the new `templates/steps/resume.md` mustache template into the report's instruction. The rendered instruction tells the agent to ask the user resume-vs-new and embeds both follow-up commands (resume via `<cmd> <kind> goto` on the current step after reading `.spektacular/context.md`, or start fresh via `<cmd> <kind> new --force`). Nothing is wired into the commands yet (Phase 2.2).

**Deviations**: `resumeInstruction`'s signature gained a leading `command string` parameter not shown in context.md (`resumeInstruction(command, kind, name, currentStep)`). This is required: `resume.md` is a *runtime*-rendered step template, so it interpolates the CLI prefix via `{{config.command}}` (the convention used by every other step template, fed from `workflow.Config.Command`) — not the install-time `{{command}}` placeholder used in SKILL.md files. The command value must therefore flow into the render call; Phase 2.2 will pass `cfg.Command` from the command handler.

**Files changed**:
- `cmd/resume.go` (new)
- `templates/steps/resume.md` (new)
- `cmd/resume_test.go` (new)

**Discoveries**: There are TWO command-placeholder conventions in this repo: runtime step templates use `{{config.command}}` (rendered by `stepkit` from `cfg.Command` via a nested `"config": {"command": ...}` var), while SKILL.md files use bare `{{command}}` (rendered at skill-install time). `resume.md` is the former. `stepkit.RenderTemplate` reads from the embedded `templates.FS` (`//go:embed all:*`), so `resumeInstruction` needs no filesystem/working-dir setup and its tests are pure (no `t.Chdir`). The default config `Command` is `"spektacular"` (`internal/config/config.go:101`).

### 2026-06-02 — Phase 2.2: Entry commands detect, report, and force

**What was done**: Rewired the three `new` commands so they no longer unconditionally wipe state. Added two shared helpers to `cmd/resume.go`: `detectInProgress(statePath)` (reads state.json, returns the state only when `InProgress()`, never mutates disk) and `resumeOrClear(cmd, statePath, command, force)` (the collapsed prologue: `--force`→remove+fresh; no in-progress→remove+fresh; in-progress with kind→write a `ResumeReport` and signal handled; in-progress without a kind→clear error, no clobber). Each `new` handler now reads a new `--force` flag, and its `os.Remove` prologue is replaced by a `resumeOrClear` call. Cross-kind reporting falls out for free — the report reads `state.Kind`, so `plan new` over an in-progress spec reports `kind:"spec"`.

**Deviations**: The detection-path error (kind-less state, or a state-read failure) is returned RAW (`return err`) rather than via `output.WriteError`, unlike the `wf.Next()` failure path. This matches how every other *input/precondition* error in these handlers propagates (`--data is required`, `name must match`, `plan file not found`, identifier-resolution errors all `return` raw), so the error surfaces as a real non-zero exit and is assertable from `rootCmd.Execute()`. Caught by a test: `output.WriteError` writes JSON to stderr and returns nil, which swallowed the error.

**Files changed**:
- `cmd/resume.go` (added `detectInProgress`, `resumeOrClear`, imports)
- `cmd/spec.go` (`--force` flag + read + prologue swap)
- `cmd/plan.go` (same; `os` import dropped — `os.Remove` was its only use)
- `cmd/implement.go` (same; `os` retained for the plan-precondition `os.Stat`)
- `cmd/spec_test.go` (full helper matrix: resume/force/clean/kind-less; `resetSpecCommandFlags` now resets `force`)
- `cmd/plan_test.go` (new — cross-kind wiring test)
- `cmd/implement_test.go` (thin wiring test + shared `writeInProgressState` helper)

**Discoveries**: cobra does NOT reset flag values between `rootCmd.Execute()` calls in tests — the new `--force` bool persists across invocations, so `resetSpecCommandFlags` had to gain a `force` reset (the plan/implement wiring tests sidestep this by never passing `--force`). The detection prologue sits where the old `os.Remove` was — for `implement` that is AFTER the plan-file precondition (`os.Stat`), so `implement new` for a missing plan still errors on the plan before detection runs; acceptable since implement always requires a plan. Test oracle for "no clobber" is a byte-for-byte `os.ReadFile` compare of state.json before/after; the resume report is parsed directly into the production `ResumeReport` struct (its JSON tags are the contract).

### 2026-06-02 — Phase 3.1: Step templates refresh the working context

**What was done**: Appended one identical "refresh working context" directive to every non-terminal step instruction template — 32 files across `templates/steps/spec/` (01–08), `templates/steps/plan/` (01–16), and `templates/steps/implement/` (01–08). The directive tells the agent to refresh the git-tracked `.spektacular/context.md` with its current working context (decisions, findings, what's done, what's next) using its own file tools before running the `goto` command. The `*-finished.md` terminal templates were deliberately skipped (no advance follows). No Go production changes.

**Deviations**: None. Placement is a uniform trailing block (after each template's `goto` command, phrased "before running the `goto` command above") rather than an inline edit per template — the templates' endings vary (some fence the goto command, some don't), so a trailing append is the only way to keep the wording byte-identical across all 32.

**Files changed**:
- `templates/steps/spec/01-overview.md` … `08-verification.md` (8)
- `templates/steps/plan/01-overview.md` … `16-write_research.md` (16)
- `templates/steps/implement/01-read_plan.md` … `08-update_repo_changelog.md` (8)
- `templates/context_directive_test.go` (new — walks the embedded FS asserting the directive is present exactly once in each non-terminal template and absent from `*-finished.md`)

**Discoveries**: There are two distinct `.spektacular/context.md` notions that must not be conflated — the root working-context sidecar this directive targets (`.spektacular/context.md`) versus a plan workflow's own plan document `.spektacular/plans/<name>/context.md`; the directive's explicit path keeps them separate. The existing template tests (`internal/stepkit`, `internal/steps/*`) assert via `Contains` on substrings, so appending content did not break them. The new test reads `templates.FS` directly (internal `package templates`), using a `>= 30` floor rather than an exact 32 so adding a future step doesn't break it.

### 2026-06-02 — Phase 3.2: Resume-aware skill playbooks

**What was done**: Rewrote the "How to start" section of all three driving-agent playbooks (`templates/skills/workflows/spek-{new,plan,implement}/SKILL.md`) so the agent reacts to a resume report instead of inspecting state. Each now: runs `<kind> new`, and on a `"resumable": true` response (a) prompts the user resume-vs-start-new, (b) on resume reads `.spektacular/context.md` then runs the report's `goto` on `current_step` (honouring the report's `kind`, which may be cross-kind), and (c) on start-fresh re-runs `new --force`. The old "inspect `.spektacular/state.json` for the active name" guidance is gone from `spek-plan`/`spek-implement`, and `spek-new` (which never referenced state) gained the same resume handling for consistency.

**Deviations**: None from intent. Per the user's earlier steer ("the skill shouldn't need to know anything about state.json"), the playbooks drop the `state.json` reference entirely rather than keeping a negative "do not read state.json" instruction — they just tell the agent the CLI detects an in-progress workflow for it. The playbooks lean on the resume report's own `instruction` field (which already restates the options) rather than fully duplicating it, while still spelling out the three steps to satisfy the acceptance criteria.

**Files changed**:
- `templates/skills/workflows/spek-new/SKILL.md`
- `templates/skills/workflows/spek-plan/SKILL.md`
- `templates/skills/workflows/spek-implement/SKILL.md`
- `templates/skill_resume_test.go` (new — content guard: each workflow playbook contains the resume markers and no `state.json` reference)

**Discoveries**: No existing test asserted on the old `state.json`/"How to start" prose, so the rewrite broke nothing — the skill-install tests (`cmd/init_test.go`, `internal/agent/*_test.go`) verify installation, not this content. The plan deliberately left playbook *behaviour* untested (an agent asking the user can't be unit-tested); the new test is a cheap static content guard for AC #4 (no `state.json`) plus the resume markers, not a behavioural simulation. With this phase the feature is complete end to end: interrupt → `new` detects and reports → agent prompts → resume re-renders the current step with working context rehydrated from `.spektacular/context.md`, or `--force` starts fresh.
