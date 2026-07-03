# Plan: 000030_agent-actionable-errors

<!-- Metadata -->
<!-- Created: 2026-07-02T14:30:26Z -->
<!-- Commit: 35181e2 -->
<!-- Branch: main -->
<!-- Repository: git@github.com:jumppad-labs/spektacular.git -->

## Overview

This plan replaces spektacular's inconsistent, ad hoc error handling with a single response
envelope that every command routes through: one reliable success/error signal, one output
channel, specific and resource-named failure text, and — for rejected multi-step requests — the
caller's current position and the actions that would currently succeed, all delivered without
duplicate or extraneous noise. It also fixes a workflow-engine timing bug where a failed step's
own logic could leave persisted progress silently ahead of what actually completed. The result is
a CLI whose errors are self-explanatory enough for an unsupervised AI agent to recover from
immediately, rather than guessing, retrying blindly, or getting stuck — the incident that
motivated this work.

## Conventions

No project conventions apply to this feature.

## Architecture & Design Decisions

The tool moves to a single choke point through which every command's outcome passes, rather than
fixing formatting at each of the ~40 existing error-return sites individually. `cmd/root.go`'s
`Execute()` — already the one place every `RunE` return value converges — becomes a wrapper that
inspects the outcome of every command and emits exactly one response: on success, the command's
existing result value (unchanged — `spec.Result`, `ResumeReport`, `StatusResult`, the ad hoc
`knowledge` maps, etc. all keep their current field sets) with one discriminant field injected
centrally in `internal/output` (marshal to `map[string]json.RawMessage`, splice in `"error":
false`, re-marshal); on failure, a single shared `ErrorResponse` shape (code, message, the
offending resource when one is named, current state + valid next actions when the failure is a
rejected transition, and an explicit next action) written to the same stream (stdout) success
already uses, with a non-zero exit code. `output.WriteError`'s 26 existing call sites and the ad
hoc `fmt.Errorf`/`return err` sites both collapse into "just return the error" — the wrapper is
the only code that ever formats one. `rootCmd.SilenceErrors`/`SilenceUsage` are set so cobra's own
default error/usage printer never runs, closing the confirmed-live duplicate-presentation bug
(today's fallback path prints an error a second time after the CLI framework's own printer
already has). This directly follows the spec's Technical Approach ("prefer a single, central
point... rather than fixing formatting at each place an error can occur") and is the option with
the smallest blast radius of three considered (full comparison and evidence:
research.md#alternatives-considered-and-rejected) — it requires no changes to any of the ~10
existing result struct definitions, so skill templates that already read `step`/`instruction` at
the top level keep working unmodified for every response type except the one folded in below.

Two error sources get explicit translation rather than passing their native wording through
unrewritten, per the spec's constraint that "where an error originates from the tool's own
internal machinery... prefer rewriting it with caller-relevant context." A rejected `looplab/fsm`
transition is intercepted where the workflow engine calls the FSM library and rebuilt using the
current step plus the set of steps that could legally run next — both already computed internally
by the workflow engine — so the `ErrorResponse`'s state/valid-actions fields are populated without
adding new state. `cmd/resume.go`'s `ResumeReport` (an in-progress or cross-kind workflow
detection) is folded into the same `ErrorResponse` shape instead of staying a separately-shaped
success response: from the caller's perspective, being told "you can't do that yet, here's what to
do instead" is the same kind of outcome whether the cause is an FSM transition or an in-progress
workflow, and collapsing it removes one of the two existing state-aware-message shapes in the
codebase rather than leaving two to keep in sync. The existing template-rendering approach that
produces the resume/mismatch instruction text carries over unchanged as the mechanism that
produces the `ErrorResponse`'s next-action text — only the outer shape it's embedded in changes.

Separately from the envelope work, the workflow engine's state-persistence-before-callback-failure
bug (confirmed against the underlying `looplab/fsm` library's source: its `enter_state` hook —
which persists workflow state to disk — runs before the hook where a step's own logic executes and
can fail) is fixed by moving the domain step-callback registration to an earlier point in the
library's callback lifecycle, one that runs before the library commits the transition or invokes
`enter_state`. A failing step callback's existing cancellation call now genuinely prevents both the
transition and its persistence, instead of the state having already silently advanced by the time
the caller sees the error. This is a persistence-timing fix, not a change to step sequencing,
ordering, or naming, so it does not conflict with the spec's Non-Goals; it is in scope because a
state/report mismatch is precisely the class of silent failure the spec's "no silent failures"
requirement targets. Step-chaining behavior (a step advancing directly to another step) is
unaffected, since it is read by the wrapping code after the library's transition call returns,
independent of which callback sets it.

One deliberate, documented exception: the `file read` command's success path writes raw file bytes
to stdout (not JSON), so a piped write round-trips file content exactly. This predates and is
orthogonal to the new envelope — success there stays raw bytes unchanged, while a failure on that
same command (e.g. the file not existing) still goes through the shared `ErrorResponse`, since an
error is always JSON regardless of what a successful call would have returned.

## Component Breakdown

- **Response envelope** (extends the existing output-writing component). Owns the wire shape of
  every command's outcome: on success, marshals the command's existing result value and injects
  the one always-present discriminant field; on failure, marshals the shared error shape. Nothing
  else in the tool is allowed to write a command's JSON result directly — every outcome passes
  through this component exactly once.

- **Command outcome wrapper** (replaces today's top-level command execution fallback). Owns
  catching the single outcome (result or error) every command produces, handing it to the
  response envelope, writing it to the one output stream shared by success and error, and setting
  the corresponding process exit code. Also owns suppressing the CLI framework's own default
  error/usage printing, so this is the only place in the process that ever writes an outcome.
  Individual commands no longer decide for themselves whether or how to report failure.

- **Shared error shape**. Not a runtime component but the common vocabulary every failure is
  expressed in before it reaches the command outcome wrapper: a failure code, a specific message,
  the named resource when one is involved, a state block (current position + the actions that
  would currently succeed) when the failure is a rejected action, and an explicit next action.
  Hand-authored errors (input validation, config problems) and translated internal errors (below)
  both produce this same shape.

- **Internal-error translator** (within the workflow engine). Owns converting a rejected state
  transition from the underlying finite-state-machine library into the shared error shape, using
  information the workflow engine already computes internally (current step, the set of steps
  that could legally run next). This is what stops a library's own internal wording from ever
  reaching a caller unrewritten, and is what supplies the state-aware "here's where you are, here's
  what you can do" content for rejected transitions.

- **Workflow step-commit ordering** (within the workflow engine). Owns *when* a step's own logic
  runs relative to persisting the workflow's on-disk state. Changed so a step's failure is
  guaranteed to prevent both the transition and its persistence, rather than the two being able to
  disagree — closing the case where a failed step could leave the persisted state one step ahead
  of what was actually completed.

- **Resume / cross-kind detector** (existing component, behavior changed). Owns detecting that a
  request can't proceed because a different or already in-progress workflow occupies the current
  state. Its output now flows through the shared error shape and the response envelope like any
  other failure, instead of being a separately-shaped response — the detector still owns deciding
  *when* this condition applies and producing the human/agent-facing next-action text, just not the
  outer shape that content is delivered in.

- **Command handlers** (spec/plan/implement/knowledge/skill/init/file commands). Unchanged in
  responsibility: each still owns parsing its own input and producing either a result value or an
  error. What each handler no longer owns is deciding how that outcome gets formatted, which
  stream it goes on, or what the process exit code should be — that responsibility moves entirely
  to the command outcome wrapper and response envelope.

## Data Structures & Interfaces

**`ErrorResponse`** — the shared error shape every failure is expressed in. Implements Go's
`error` interface (`Error() string`) so it can be constructed deep inside any package and returned
as a normal error all the way up to the command outcome wrapper without that call chain needing to
know about JSON at all.

```go
type ErrorResponse struct {
    Error      bool       `json:"error"`                 // always true; the discriminant on the error side
    Code       string     `json:"code"`                  // short, stable failure category, e.g. "invalid_transition"
    Message    string     `json:"message"`                // specific failure text naming what went wrong
    Resource   string     `json:"resource,omitempty"`     // the named file/step/identifier involved, when applicable
    State      *StateInfo `json:"state,omitempty"`        // populated only for state-aware rejections
    NextAction string     `json:"next_action"`            // explicit corrective action or valid next command
}

type StateInfo struct {
    Current      string   `json:"current"`       // the caller's current position (e.g. current step name)
    ValidActions []string `json:"valid_actions"` // the actions/steps that would currently succeed
}
```

Built via small constructor/fluent helpers (e.g. `output.NewError(code, message string)
*ErrorResponse`, `.WithResource(name string)`, `.WithState(current string, valid []string)`,
`.WithNextAction(text string)`) rather than struct literals scattered across call sites, so every
error is built through the same narrow surface.

**Success discriminant** — no new type. Every existing result value (`spec.Result`, `ResumeReport`
survivors, `StatusResult`, the ad hoc `knowledge` maps, etc.) keeps its current field set; the
response envelope injects one field, `"error": false`, when marshaling any value that isn't an
`ErrorResponse`. This is a marshaling-time behavior change inside the existing `WriteResult`
contract, not a new interface — every call site that already calls `WriteResult` needs no changes.

**Command outcome wrapper contract** — the top-level `Execute()` function's shape changes from
"print whatever error text `RunE` returned" to a single translation step:

```go
func Execute() {
    err := rootCmd.Execute()
    if err == nil {
        return
    }
    writeFailure(os.Stdout, toErrorResponse(err))
    os.Exit(1)
}

func toErrorResponse(err error) *ErrorResponse {
    if er, ok := err.(*ErrorResponse); ok {
        return er // already fully-formed by the originating code
    }
    return NewError("internal_error", err.Error()) // generic fallback for un-translated errors
}
```

This is the only place `ErrorResponse` values are ever written to the output stream — individual
`RunE` functions stop calling any error-writing helper themselves and simply `return` an error
(ideally an already-built `*ErrorResponse`, falling back to a plain Go error where translation
hasn't been added yet).

**Internal-error translator** — not a new exported type, but a defined contract inside the
workflow engine: a function that recognizes a rejected-transition error from the underlying FSM
library and converts it to an `*ErrorResponse` populated with `State{Current: <current step>,
ValidActions: <legal next steps>}` and a `NextAction` naming the concrete `goto` command, before
`Goto`/`Next` return it. Everywhere else in the workflow engine that already returns a plain `error`
is unaffected — only the FSM-library-originated case is translated.

**Resume/cross-kind detector** — `ResumeReport` is removed as a standalone type; the condition it
represented (in-progress or cross-kind workflow) is now expressed by constructing an
`ErrorResponse` (code identifying the case, `State` carrying the in-progress workflow's kind/step,
`NextAction` carrying the same rendered resume-or-overwrite instruction text it produces today).

## Implementation Detail

**Errors become values, not side effects.** Today, code that hits a failure has to decide, at the
point of failure, whether to format and write a response itself or just return a Go error and hope
something upstream handles it — the inconsistency the whole feature exists to fix. Going forward,
every piece of code that can fail does exactly one thing: return an error. Nothing below the
command outcome wrapper ever writes to the output stream on the failure path. This is a pattern
shift as much as a structural one — "how do I fail" becomes a single idiom (build or propagate an
error value) instead of a per-call-site judgment call between several existing helpers.

**The CLI framework's own default error handling is turned off.** The root command explicitly
disables the framework's built-in error/usage printing, formalizing that response presentation
belongs entirely to the tool's own wrapper, never to the framework or to an individual command. A
developer reading any command's handler going forward can assume their function is never the last
thing that touches the output stream on either the success or the failure path — this removes a
question ("do I also need to print something here?") that today has two different answers
depending on which existing command you copy from.

**The resume/cross-kind detection code changes shape from "writes as a side effect" to "returns a
value."** Its two shared entry points currently decide internally whether to write a report and
signal that back to the caller with a boolean; they now simply return the appropriate error value
(or none) like any other function, and the caller's existing "did this handle it" branching
collapses into ordinary Go error handling. This removes one of the two different ways a command
can currently short-circuit with a structured response, leaving one.

**The workflow engine gains one narrow translation seam.** Where it currently lets a rejected
transition's underlying error escape unchanged, it now recognizes that one specific error type at
the point it's returned and rebuilds it using state the engine already tracks internally — no new
state, no new persistence, just a rewrite at the boundary before the error leaves the package.
Every other error the workflow engine already returns (store failures, template failures, generic
step failures) is untouched by this feature; only the rejected-transition case gets translation,
because it's the only one whose raw wording is both misleading and non-obvious to reconstruct from
outside the package.

**The workflow engine's step-callback wiring moves to an earlier hook in the underlying state
machine's callback lifecycle**, so a step's own failure is evaluated, and can veto the transition,
before anything is persisted — rather than after, as today. This is a narrow, mechanical change to
*when* one callback fires, not to what step callbacks are allowed to do or how they're written;
existing step callbacks need no changes.

**Input-validation error text across the three near-identical command families (spec/plan/implement)
gets an incidental cleanup.** Because every validation failure now flows through the same small set
of error-construction helpers instead of ad hoc `fmt.Errorf` calls, the triplicated `--data`
parsing/name-validation logic in those three files naturally converges on the same failure
categories and wording instead of three independently-drifting copies — a side effect of adopting
the new pattern everywhere it touches, not a separate refactor.

**The output package's role narrows and consolidates.** It keeps its low-level job (marshal,
optionally field-filter, write) as the one place JSON is actually serialized, but gains ownership
of the error shape and its constructors, and loses its several different "write an error" entry
points in favor of one. Every existing call site that writes a success result today keeps calling
the same function it already calls — the discriminant field becomes an implementation detail of
that function rather than something callers opt into.

## Dependencies

- **`internal/output`** — the package this plan extends into the response-envelope component
  (error shape, constructors, discriminant injection). No external change needed first; this plan
  changes it directly.
- **`internal/workflow`** — owns the finite-state-machine wrapper this plan changes in two ways
  (rejected-transition translation, step-callback commit-ordering fix). No external change needed
  first; this plan changes it directly.
- **`cmd` package** (root command plus the spec/plan/implement/knowledge/skill/init/file command
  groups) — every command handler is a caller of the new error-construction and outcome-wrapper
  contract. No external change needed first; this plan changes the shared entry point and expects
  each handler to keep working through it, with input-validation call sites converging onto the
  new helpers incidentally.
- **`internal/stepkit`** — supplies the existing template-rendering used for state-aware
  next-action text (already used by the resume/cross-kind detector). Unchanged by this plan;
  consumed as-is.
- **`github.com/looplab/fsm` (v1.0.3)** — the external state-machine library whose error types and
  callback-ordering semantics this plan works around (translating `InvalidEventError` and similar,
  and moving step-callback registration to an earlier lifecycle hook). No version change needed;
  this plan adapts to its existing documented behavior rather than requiring a library change.
- **`github.com/spf13/cobra` (v1.8.1)** — the CLI framework whose default error/usage-printing
  behavior this plan explicitly disables (`SilenceErrors`/`SilenceUsage`) in favor of the tool's
  own wrapper. No version change needed.
- **Spec `000030_agent-actionable-errors`** — the spec this plan implements; already finished and
  committed. No further spec work needed before this plan starts.
- **Spec `000024_resume`** — the spec that produced the resume/cross-kind detection mechanism this
  plan folds into the new error shape. Already finished and committed; this plan changes its
  output shape but not the detection logic or conditions it introduced.
- **No prior plan exists for 000030** — this is the first plan against this spec, so there is
  nothing to reconcile with or supersede.
- **Spec `000005_adhoc_json_protocol`** — reviewed during discovery as prior art for envelope
  design, but its protocol is scoped to a distinct, apparently-unimplemented `adhoc` subcommand.
  Not a dependency: this plan does not build on, require, or modify it.

## Testing Approach

**Test kinds.** Unit tests cover the new error-shape construction (builders produce the expected
fields; marshaling injects the discriminant correctly for both success and error values) and the
two workflow-engine changes in isolation (the rejected-transition translator; the step-commit
ordering fix). Integration-style tests, following this codebase's existing convention of driving
commands through the real root command and asserting on captured output, cover the command outcome
wrapper end-to-end — invalid input, rejected transitions, in-progress/cross-kind detection, and
unexpected internal failures, each exercised as a real command invocation rather than a mocked
unit. A small set of regression tests reproduce, as close to literally as possible, the specific
incident described in the spec's Overview (a rejected step transition with no indication of the
correct next step) to guard against exactly that scenario recurring.

**Where coverage concentrates.** The command outcome wrapper and the workflow engine's two changed
behaviors get the deepest coverage, since they are the only places this plan changes runtime
behavior rather than just message content. The output package's envelope mechanics get thorough
unit coverage since every other test indirectly depends on it producing the right shape. Packages
whose error *messages* pass through the new wrapper unchanged (store, config, knowledge, agent/install)
are not given new tests of their own message wording — they're covered indirectly by the wrapper's
generic-fallback path, exercised once, rather than once per origin package.

**Load-bearing assertions.** Tests guarantee: every response, success or error, carries the
discriminant field and only one of the two shapes; a rejected transition's error names the current
step and lists exactly the steps that would currently succeed; a failing step callback leaves
persisted state unchanged rather than advancing past the failed step; a resource-specific failure
(missing file, unknown scope, bad identifier) names that resource in the message text, not a
generic phrase; an unexpected internal error still produces a visible error response rather than a
silent success-shaped one; the exact same invalid request issued twice produces byte-identical
error text; and a sampled set of error scenarios produce output containing the failure text exactly
once, with nothing else written to either stream.

**Test conventions followed.** Command-level tests continue the project's existing pattern of
constructing the real root command, setting args, capturing stdout/stderr separately, and asserting
on the parsed JSON result — the same shape already used for the resume/cross-kind tests this plan
changes the output shape of. Workflow-engine tests continue the existing table-style pattern already
used for transition tests in that package.

**Deliberate gaps.** No new tests are added purely for input-validation wording in the
spec/plan/implement command families beyond confirming they now route through the shared error
shape — the wording convergence described in the Implementation Detail section is a byproduct, not
a behavior this plan is contractually locking down test-by-test. No load or concurrency testing is
added; nothing in this plan changes concurrent behavior.

**Success metrics.**

- *"An agent that encounters any error from the tool can determine and take a correct next action
  directly from that error's text, without a human interpreting it and without probing multiple
  other commands to figure out what happened."* — Behavioural test covers the structural half: for
  every category of error this plan produces (rejected transition, in-progress/cross-kind, missing
  resource, invalid input, internal failure), a test asserts the response carries a non-empty
  `NextAction` and, where applicable, `State`/`Resource` — i.e. the information an agent would need
  is mechanically present every time. Whether that text is actually sufficient for an agent to act
  on unaided is a qualitative judgment a unit/integration test can't make — Manual, captured in the
  implementation test plan: a spot-check where a fresh agent session is shown a captured error
  response with no other context and is confirmed to propose the correct next command.
- *"No agent session gets stuck retrying or looping in place because an error it received didn't
  explain what to do next — the kind of incident that motivated this work does not recur."* —
  Behavioural test: the regression test reproducing the original incident's exact rejected-transition
  scenario asserts the new response's `NextAction`/`State` would have given the agent the correct
  next command, which is the direct mechanical guarantee this plan can make. Whether the incident
  class actually stops recurring in live use is an ongoing production observation, not something CI
  can certify — Manual, captured in the implementation test plan as a note to watch for in future
  agent-session incident reports.

## Milestones & Phases

### Milestone 1: Every response arrives on one channel with a signal that can't lie

**What changes**: Today, an agent driving this tool has to watch two different output streams to
reliably see whether a command succeeded, and even when it does, a failed command can currently
report success (exit code 0, or no signal distinguishing it from a real result). After this
milestone, every command — success or failure — writes to the same stream, every response carries
one always-correct field saying which it is, and the process exit code always agrees with it. The
CLI framework's own default error/usage printing is turned off, so nothing prints a message more
than once. This milestone doesn't yet make error *text* more informative — it fixes the mechanics
an agent depends on just to reliably notice a failure happened at all.

#### - [x] Phase 1.1: Introduce the shared error shape

Adds the vocabulary every failure will be described in — a code, a specific message, an optional
named resource, an optional current-position-and-valid-actions block, and an explicit next action —
plus the small mechanism that marks every successful response with the same always-present signal.
This phase only adds new capability; nothing in the tool uses it yet.

*Technical detail:* [context.md#phase-11](./context.md#phase-11-introduce-the-shared-error-shape)

**Acceptance criteria**:
- [x] A failure can be constructed with a code, message, and optionally a resource name, a
      current-position-and-valid-actions block, and a next action.
- [x] Marshaling any existing successful result automatically carries the same success/error
      signal field, with no changes required to that result's own definition.

#### - [x] Phase 1.2: Route every command through one response wrapper

Every command in the tool now produces its outcome through a single, shared point instead of each
deciding independently how (or whether) to format a failure. A failure and a success are written
to the same stream, the process's exit status always agrees with which one happened, and the
underlying CLI framework's own default error/usage output is turned off so nothing is ever printed
more than once.

*Technical detail:* [context.md#phase-12](./context.md#phase-12-route-every-command-through-one-response-wrapper)

**Acceptance criteria**:
- [x] Every command (spec, plan, implement, knowledge, file, skill, init) reports both success and
      failure on the same output stream.
- [x] A failing command's response carries the failure signal and the process exits non-zero;
      a succeeding command's response carries the success signal and the process exits zero.
- [x] A failure is printed exactly once, with no additional framework-generated usage or help text.

### Milestone 2: Error text tells an agent where it is and what it can do next

**What changes**: Building on the reliable channel from Milestone 1, error content itself gets
rewritten to be self-sufficient. A rejected step transition now states the caller's current
position and the actions that would currently succeed, instead of the state machine library's own
internal wording. A request blocked because a different or already in-progress workflow is active
states that condition and the concrete follow-up commands, using the same shared shape as any other
error rather than a separately-shaped response. Any error that concerns a specific file, step, or
identifier names that resource directly. This is the milestone that actually delivers the spec's
core promise — an agent can read an error and know what to do next without guessing or asking a
human.

#### - [x] Phase 2.1: State-aware rejected-transition messages

When a request is rejected because it doesn't make sense at the caller's current position in a
multi-step workflow, the response now says exactly where the caller is and which actions would
currently succeed, instead of surfacing the underlying state-machine library's own internal
wording.

*Technical detail:* [context.md#phase-21](./context.md#phase-21-state-aware-rejected-transition-messages)

**Acceptance criteria**:
- [x] A rejected step request names the current step.
- [x] A rejected step request lists the step(s) that would currently succeed.

#### - [x] Phase 2.2: Name the resource in resource-specific failures

Failures that concern a specific file, entry, or identifier now name that resource directly in the
message rather than using a generic phrase like "the file" or "not found."

*Technical detail:* [context.md#phase-22](./context.md#phase-22-name-the-resource-in-resource-specific-failures)

**Acceptance criteria**:
- [x] Requesting a file, entry, or identifier that doesn't exist produces a message naming that
      specific file, entry, or identifier.

#### - [x] Phase 2.3: Fold in-progress/wrong-workflow detection into the shared shape

When a command can't proceed because a different or already-in-progress workflow occupies the
current state, that condition is now reported through the same shared shape as any other failure —
naming the in-progress workflow and giving the concrete follow-up commands — instead of as a
separately-shaped response only this one case used.

*Technical detail:* [context.md#phase-23](./context.md#phase-23-fold-in-progresswrong-workflow-detection-into-the-shared-shape)

**Acceptance criteria**:
- [x] Attempting to start or continue a workflow while a different (or already in-progress) one is
      active produces a response in the same shared shape as any other failure.
- [x] That response names the in-progress workflow, its current step, and the concrete commands to
      resume it or discard it.
- [x] The skill instructions that tell an agent how to interpret this situation describe the new
      response shape, not the old one.

### Milestone 3: A failed step can never look like it succeeded

**What changes**: Independently of how errors are reported, this milestone closes a deeper gap:
today, if a step's own logic fails partway through, the workflow's saved progress can still advance
past that step before the failure is reported, so a later resume or status check would treat the
failed step as done. After this milestone, a failing step can no longer advance the saved progress
— the two can never disagree. This milestone also reproduces the specific incident described in the
spec (an agent stuck retrying a rejected transition with no way forward) end-to-end, confirming the
combination of Milestones 1 and 2 actually resolves it.

#### - [x] Phase 3.1: Stop failed steps from silently advancing saved progress

Fixes the underlying timing bug where a step's own logic could fail after the workflow had already
saved progress past that step. After this phase, a step's failure is guaranteed to prevent the
save from happening at all, so the workflow's saved position can never say more progress happened
than actually did.

*Technical detail:* [context.md#phase-31](./context.md#phase-31-stop-failed-steps-from-silently-advancing-saved-progress)

**Acceptance criteria**:
- [x] When a step's own logic fails, the workflow's saved progress still shows that step as not
      completed.
- [x] Every existing successful step-advancement scenario continues to work exactly as before.

#### - [x] Phase 3.2: Confirm the motivating incident is resolved

Reproduces the exact scenario described in the spec — an agent hitting a rejected step request
with no indication of the correct next step — end to end, confirming the combined effect of the
earlier phases actually gives the agent what it needs to recover on its own.

*Technical detail:* [context.md#phase-32](./context.md#phase-32-confirm-the-motivating-incident-is-resolved)

**Acceptance criteria**:
- [x] Replaying the original incident's exact sequence produces a response that names the correct
      next command, on the same stream as every other response, with nothing extraneous alongside it.

## Open Questions

None. Every design decision this plan depends on — the central-wrapper architecture, the shared
error shape's fields, the fate of `ResumeReport`, the scope of the workflow persistence-ordering
fix, the `runRoot()` test-wiring approach, and the need to update the three skill docs describing
the old resume-report shape — was resolved during discovery, architecture, and phases rather than
deferred. Nothing identified so far depends on information that can only surface once
implementation begins.

## Out of Scope

- **A debug or session action-log for reconstructing agent behavior after the fact.** Excluded by
  the spec's Non-Goals; tracked separately in spec `000031_agent-session-debug-log`.
- **Changing the workflow's step sequencing, ordering, or naming.** Excluded by the spec's
  Non-Goals. This plan changes what a rejected request communicates, not which requests are valid
  or how steps are ordered.
- **Validating, correcting, or otherwise constraining agent behavior** (for example, preventing an
  agent from guessing an invalid step name before it tries). Excluded by the spec's Non-Goals —
  this plan makes errors recoverable, it does not police the caller.
- **Error-rate telemetry, monitoring, or alerting.** Excluded by the spec's Non-Goals; distinct
  from the qualitative success metrics this plan verifies.
- **Closing the `goto`-to-current-step silent no-op** (documented as a knowledge-base gotcha). Not
  a spec Non-Goal by name, but decided out of scope during the milestones step: that behavior is a
  deliberate, working mechanism the resume flow depends on (re-rendering the current step without
  losing progress), not a failure — so it doesn't violate "no silent completion," and changing it
  would risk conflicting with the sequencing/ordering Non-Goal above.
- **Adopting the nested envelope shape from spec `000005_adhoc_json_protocol`.** Considered as
  Architecture Option 3 and not chosen (see `research.md#alternatives-considered-and-rejected`):
  higher effort and blast radius than this spec requires, and that spec's protocol remains
  unimplemented and untouched by this plan.
- **Rewording or restructuring error message content in packages this plan doesn't otherwise
  touch** (e.g. `internal/config`, `internal/agent`) beyond routing their existing errors through
  the new shared wrapper. Their current message wording already names the resource involved in
  most cases (per research.md's file-by-file findings) and isn't part of the gap this plan closes;
  revisiting their wording is left for whoever next touches those packages.

## Changelog

### 2026-07-02 — Phase 1.1: Introduce the shared error shape

**What was done**: Added `StateInfo` and `ErrorResponse` (implements `error`) to
`internal/output/writer.go`, plus the `NewError`/`WithResource`/`WithState`/`WithNextAction`
builder chain and a `WriteFailure` function. `Write` now injects `"error": false` into any
marshaled value that isn't an `*ErrorResponse`, so every existing success result gets the
discriminant with no changes to its own definition.

**Deviations**:
- Fixed a defect in the plan's own Data Structures spec: a struct can't have both an `Error bool`
  field and an `Error() string` method in Go. Renamed the field to `IsError` (JSON tag stays
  `"error"`, so the wire shape matches the plan exactly); the `Error()` method is unchanged.
- Did not delete `WriteError` yet, despite context.md's Phase 1.1 file list saying to. 26 call
  sites across `cmd/*.go` still depend on it until Phase 1.2 replaces them with `return err` — this
  matches plan.md's own Phase 1.1 description ("this phase only adds new capability; nothing in
  the tool uses it yet"). Deletion deferred to Phase 1.2.
- Found and fixed a real bug during the full-suite test run: the new discriminant injection was
  unconditionally overwriting `WriteError`'s own `"error"` key (its failure-message string) with
  the boolean `false`. Fixed by having `WriteError` marshal and write directly, bypassing `Write`'s
  injection path — transient scaffolding until `WriteError` is deleted in Phase 1.2.
- Updated one pre-existing test (`cmd/knowledge_test.go`'s
  `TestKnowledgeSearch_EmptyAndNoMatchQueriesReturnEmptyHits`) whose literal `{"hits": []}`
  assertion needed `"error": false` added — the correct, expected consequence of this phase's own
  second acceptance criterion.

**Files changed**:
- `internal/output/writer.go`
- `internal/output/writer_test.go` (new)
- `cmd/knowledge_test.go`

**Discoveries**: Any future `Write`-adjacent change must watch for values whose JSON already uses
an `"error"` key for something other than the success/failure discriminant — the injection logic
assumes `"error"` is exclusively the discriminant's name, which only fully holds once `WriteError`
is gone (Phase 1.2).

### 2026-07-02 — Phase 1.2: Route every command through one response wrapper

**What was done**: `cmd/root.go`'s `Execute()` is now `os.Exit(runRoot())`. The new `runRoot() int`
runs `rootCmd.Execute()`, returns 0 on success, and on failure formats the error through
`toErrorResponse` + `output.WriteFailure` (writing to the same stream success uses) before
returning 1. `rootCmd.SilenceErrors`/`SilenceUsage` are now set, so cobra's own default error/usage
printer never runs. All 22 `output.WriteError` call sites across `cmd/spec.go`, `cmd/plan.go`,
`cmd/implement.go`, `cmd/knowledge.go` were replaced with a plain `return err` — no command handler
formats or writes its own failure anymore. `output.WriteError` itself (deferred from Phase 1.1) is
now deleted, since nothing calls it.

**Deviations**:
- context.md's file list named only a handful of test helpers as needing to switch from
  `rootCmd.Execute()` to the new wrapper; the actual count was ~45 inline `rootCmd.Execute()` calls
  across 9 test files. In practice only 2 tests needed changes (the two envelope tests already
  named in context.md) — `SilenceErrors`/`SilenceUsage` don't change what `rootCmd.Execute()`
  returns, only whether cobra prints it, so every other call site (which checks the Go error value,
  not printed text) kept working unchanged.
- `cmd/knowledge_test.go`'s `runKnowledge` helper now calls `runRoot()` instead of
  `rootCmd.Execute()` directly, and on non-zero exit unmarshals stdout into an
  `*output.ErrorResponse` to return as its `error` value, so its other (passing) callers needed no
  changes.

**Files changed**:
- `cmd/root.go`
- `cmd/spec.go`, `cmd/plan.go`, `cmd/implement.go`, `cmd/knowledge.go` (mechanical call-site removal)
- `internal/output/writer.go` (`WriteError` deleted)
- `cmd/knowledge_test.go` (helper + two envelope tests rewritten for the new stdout-based shape)
- `cmd/root_test.go` (new — dedicated wrapper-behavior tests for this phase's 3 acceptance criteria)

**Discoveries**: A rejected FSM transition (e.g. calling `implement goto` for a step unreachable
from the current state) still surfaces via the generic `internal_error` fallback in this phase,
with no state-aware next-action text yet — hit this live mid-implementation when a `goto` was
called out of sequence. Phase 2.1 is what adds the state-aware translation; until then this
wrapper's fallback path is functionally correct (single stream, correct exit code) but not yet
informative. Also: `spec new`'s resume-check runs before its name-validation, so two `spec new`
calls in the same directory turn a would-be validation failure into a resume-report success —
matters for any test exercising `spec new` failure paths (use a fresh directory per invocation).

### 2026-07-02 — Phase 2.1: State-aware rejected-transition messages

**What was done**: `internal/workflow/workflow.go`'s `Next()`/`Goto()` now route the error from
`w.FSM.Event(...)` through a new `translateTransitionError`. When the error is a genuine rejected
transition (`fsm.InvalidEventError`/`fsm.UnknownEventError`, checked via `errors.As`), it's rebuilt
as an `*output.ErrorResponse` naming the current step and the steps that would currently succeed
(`w.FSM.AvailableTransitions()`), with a `NextAction` rendering a concrete
`<command> <kind> goto --data '{"step":"<step>"}'` in the same style as the resume template. Any
other error (a step callback's own domain failure) passes through unchanged.

**Deviations**: None from the plan's design. Process-only deviation: mid-implementation, advancing
the *implement workflow itself* (this same CLI, dogfooded) from `analyze` straight to `test`
without the intermediate `goto implement` triggered exactly the kind of rejected-transition error
this phase fixes — and, working correctly, it told me precisely what to do next. Recovered by
calling `implement goto {"step":"implement"}` first. Not a plan/code deviation, but a live
confirmation the feature works as intended on its own tool.

**Files changed**:
- `internal/workflow/workflow.go`
- `internal/workflow/workflow_test.go`

**Discoveries**: `looplab/fsm@v1.0.3`'s `InvalidEventError`/`UnknownEventError` are value types
(not pointers) returned directly by `FSM.Event()`; a step callback's own error (set via
`e.Cancel(err)`) is NOT wrapped in `fsm.CanceledError` when raised from an `after_<event>`
callback — `Event()` returns it completely unwrapped, confirmed by both reading the library source
and a new regression test (`TestNextCallbackErrorNotTranslated`). This matters for Phase 3.1, which
moves the callback registration from `after_` to `before_` — worth re-confirming the same
unwrapped-error behavior holds for `before_` callbacks too when that phase lands.

### 2026-07-02 — Phase 2.2: Name the resource in resource-specific failures

**What was done**: `internal/store/store.go`'s `FileStore.abs()` now includes the attempted path
in its path-traversal-rejection error (`fmt.Errorf("path %q escapes store root", path)`, was a
bare string). `cmd/storefile.go`'s `file read <path>` now checks `errors.Is(err, store.ErrNotFound)`
and returns `output.NewError("not_found", ...).WithResource(args[0])` naming the missing file,
instead of the raw, unnamed `store.ErrNotFound`.

**Deviations**: Real mismatch found during analysis, resolved without stopping. Of context.md's 4
named touchpoints, 2 (`internal/steps/spec/identifier.go:185`, `internal/knowledge/set.go:240`)
turned out to be benign "not found yet, treat as empty/skip" swallow paths that never propagate an
error to any caller — not resource-lookup failures the acceptance criterion could apply to.
Confirmed by grepping every `ErrNotFound` reference in the repo (5 total): these two are the only
`errors.Is(err, store.ErrNotFound)` callers anywhere, and neither returns an error on the
not-found branch. Implemented only the 2 valid touchpoints (`store.go`, `storefile.go`), which
alone fully satisfy the phase's single acceptance criterion.

**Files changed**:
- `internal/store/store.go`
- `internal/store/store_test.go`
- `cmd/storefile.go`
- `cmd/file_test.go`

**Discoveries**: None beyond the mismatch above (see Deviations).

### 2026-07-02 — Phase 2.3: Fold in-progress/wrong-workflow detection into the shared shape

**What was done**: `cmd/resume.go`'s `ResumeReport` type is removed entirely. `emitResumeReport`
(now `(command, expectedKind string, state *workflow.State) error`, dropped its `*cobra.Command`
param) returns an `*output.ErrorResponse` — `code` is `"workflow_in_progress"` (same-kind) or
`"cross_kind_workflow_in_progress"` (cross-kind), with `.WithResource(name)`,
`.WithState(state.CurrentStep, nil)`, and `.WithNextAction(instruction)` carrying the exact same
`resumeInstruction`/`mismatchInstruction` template rendering as before — only the outer shape
changed. `resumeOrClear`/`guardKind` (also dropped their `*cobra.Command` param) now propagate
that as `err` on the `handled=true` path instead of writing output and returning nil; all 9 call
sites in `cmd/spec.go`/`cmd/plan.go`/`cmd/implement.go` updated (`return nil` → `return err` on the
handled branch). All three skill docs (`spek-plan`, `spek-new`, `spek-implement` `SKILL.md`)
rewritten to describe the new response shape in place of the old `resumable`/`kind`/`current_step`/
`instruction` fields.

**Deviations**: Scope correction (same pattern as Phase 1.2): context.md named only
`cmd/cross_kind_test.go` and `cmd/resume_test.go` as needing `ResumeReport` updates, but
`cmd/plan_test.go`, `cmd/implement_test.go`, and `cmd/spec_test.go` also referenced it (5 files,
~9 sites total, confirmed by grep before delegating). All five migrated to `output.ErrorResponse`,
preserving each test's original protected property under the new shape.

**Files changed**:
- `cmd/resume.go`
- `cmd/spec.go`, `cmd/plan.go`, `cmd/implement.go` (9 call-site updates)
- `cmd/resume_test.go`, `cmd/cross_kind_test.go`, `cmd/plan_test.go`, `cmd/implement_test.go`,
  `cmd/spec_test.go`
- `.claude/skills/spek-plan/SKILL.md`, `.claude/skills/spek-new/SKILL.md`,
  `.claude/skills/spek-implement/SKILL.md`

**Discoveries**: `spek-new/SKILL.md`'s original "one of two things comes back" framing (a
success-shaped resume report vs. an error) no longer holds — `spec new` with no data now always
fails (non-zero exit) whether or not a workflow is in progress; the two cases are distinguished by
`code` (`workflow_in_progress`/`cross_kind_workflow_in_progress` vs. anything else) rather than by
outcome shape. This completes Milestone 2 (error text quality) — all three of its phases are done.

### 2026-07-02 — Phase 3.1: Stop failed steps from silently advancing saved progress

**What was done**: `internal/workflow/workflow.go`'s `New()` changed the step-callback
registration key from `callbacks["after_"+s.Name]` to `callbacks["before_"+s.Name]`. Per
`looplab/fsm` v1.0.3's callback ordering, `before_` callbacks run — and can `Cancel()` to genuinely
veto — before the library commits the transition (`f.current = dst`) or invokes `enter_state`
(which persists `state.json`), whereas an `after_` callback's `Cancel()` previously only set the
returned error after both had already happened.

**Deviations**: A predicted-and-confirmed side effect, not a plan deviation: `looplab/fsm` wraps a
`before_` callback's canceled error as `fsm.CanceledError{Err: err}` (changing `.Error()`'s text),
unlike `after_` which returns the raw error unwrapped. This broke Phase 2.1's
`TestNextCallbackErrorNotTranslated`, which asserted exact string equality on the callback error —
fixed by switching to `require.ErrorIs(t, err, callbackErr)`, which correctly follows
`CanceledError.Unwrap()`.

**Files changed**:
- `internal/workflow/workflow.go`
- `internal/workflow/workflow_test.go` (new regression test + one Phase 2.1 test fix)

**Discoveries**: Any future check on a step-callback error's exact text must account for FSM-library
wrapping going forward — `errors.Is`/`errors.As` are robust to it, string equality on `.Error()` is
not. This completes Milestone 3's persistence-correctness half; Phase 3.2 (end-to-end incident
reproduction) is the plan's final phase.

### 2026-07-02 — Phase 3.2: Confirm the motivating incident is resolved

**What was done**: Added a single end-to-end regression test,
`cmd/error_response_test.go`'s `TestGoto_UnreachableStepReturnsActionableError`, driving the real
CLI stack (`runRootCmd` → `runRoot()` → cobra → `internal/workflow`) through the exact class of
scenario described in the spec's motivating incident: a `new` command lands on a step, then a
`goto` to an unreachable step is issued. Asserts exit code 1, empty stderr, a single valid JSON
`ErrorResponse` on stdout with the correct `state.current`/`state.valid_actions`/`next_action`.
No production code changes — every fix this test exercises was already in place from Phases
1.1–3.1; this phase is a verification pass confirming their combination actually resolves the
incident.

**Deviations**: None.

**Files changed**:
- `cmd/error_response_test.go` (new)

**Discoveries**: Live-reproduced the original incident with the real binary as a final sanity
check beyond the automated test: `spec new` then `spec goto` to an unreachable step now returns
exit code 1 (was 0 before this plan — a failure that previously looked like success), stderr
completely empty, and a single clean JSON response on stdout naming the real current step, the
actually-reachable next step, and a concrete runnable recovery command — compared directly against
the original incident's raw output (`{"error":"event milestones inappropriate in current state
overview"}`, exit 0, no recovery information) documented in research.md. A second, unrelated
failure mode from an earlier phase (`spec file read` on a missing file) was re-checked at the same
time and still works correctly, confirming no end-of-plan regression.

**All 7 phases across all 3 milestones are now complete. Plan 000030_agent-actionable-errors is
fully implemented, tested, and verified.**
