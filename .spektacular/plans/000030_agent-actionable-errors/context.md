# Context: 000030_agent-actionable-errors

## Current State Analysis

- **Dual output channels today.** Success responses write to stdout via `output.Write`/
  `WriteResult` (`internal/output/writer.go:37-88`); errors write to stderr via
  `output.WriteError(cmd.ErrOrStderr(), err)` (`internal/output/writer.go:29-31`, called at
  `cmd/spec.go:240,312`, `cmd/plan.go:136,212`, `cmd/implement.go:148,224`, and 15 sites in
  `cmd/knowledge.go`) or via the plain-text fallback in `cmd/root.go:23-28`. This directly
  violates the spec's constraint that success and error must share one channel.
- **Exit code lies today.** `output.WriteError` returns `nil` on a successful *write*, so `RunE`
  returns `nil` even when the underlying request failed — confirmed live, a rejected FSM
  transition currently exits `0`.
- **Cobra's defaults double-print today.** `cobra.Command.SilenceErrors`/`SilenceUsage` are never
  set anywhere in `cmd/*.go` (zero grep hits). A bare `return fmt.Errorf(...)`/`return err` (used
  by roughly half of all error-return sites) hits cobra's own default error+usage printer, then
  `cmd/root.go:25` prints the same message a second time — confirmed live on a missing-`--data`
  invocation.
- **Raw FSM wording leaks today.** `internal/workflow/workflow.go:158` (`Next`) and `:177`
  (`Goto`) both do a bare `return err` on `w.FSM.Event(...)` failure. Confirmed live:
  `{"error":"event milestones inappropriate in current state overview"}` — sourced from
  `looplab/fsm@v1.0.3/errors.go:23-30` (`InvalidEventError.Error()`), which names the current
  state but never valid next actions.
- **The workflow engine already computes what's needed for state-aware messages.**
  `w.FSM.AvailableTransitions()` (used internally at `workflow.go:153`) and `w.StepNames()`
  (`workflow.go:270-276`) are both ready-made inputs; neither is currently surfaced into an error
  path.
- **`cmd/resume.go`'s `ResumeReport` is the strongest existing state-aware-message precedent.**
  `ResumeReport{Resumable, Kind, RequestedKind, Name, CurrentStep, Instruction}`
  (`cmd/resume.go:20-27`), rendered via `stepkit.RenderTemplate` from
  `templates/steps/resume.md`/`resume_mismatch.md` (`cmd/resume.go:38-62`) — but it is
  success-shaped (written via `WriteResult`, no discriminant of its own), not error-shaped.
- **Verified structural bug: state persists before a failing step callback can veto it.**
  `internal/workflow/workflow.go:110-119` registers the domain step callback as
  `callbacks["after_"+s.Name]`; `:134-143` registers `enter_state`, which calls `saveState`.
  Per `looplab/fsm@v1.0.3/fsm.go:296-403`, the library sets `f.current = dst` and invokes
  `enterStateCallbacks` (`enter_state` → `saveState`) *before* `afterEventCallbacks` (`after_
  <step>` → the domain callback) runs. Confirmed by reading `event.go:47-56`: `Event.Cancel()`'s
  documented use is `before_<EVENT>`/`leave_<STATE>`, both of which run before the transition
  commits — calling it from `after_<event>` (as today's code does) can no longer prevent the
  state change or its persistence, only alter the returned error. No existing test in
  `internal/workflow/workflow_test.go` exercises a failing step callback's effect on persisted
  state.
- **`store.ErrNotFound` (`internal/store/store.go:11-12`) is a generic sentinel** (`"not found"`)
  with no resource name at the point it's raised; checked via `errors.Is` at
  `internal/steps/spec/identifier.go:185` and `internal/knowledge/set.go:240`, both of which have
  the specific path in scope and can name it. `cmd/storefile.go:65-77` (`file read`) currently
  returns this raw, unrewritten.
- **Only `cmd/knowledge_test.go:342-371` locks in an exact JSON error shape today**
  (`{"error":...}` on stderr); every other error assertion in the suite
  (`cmd/spec_test.go`, `cmd/cross_kind_test.go`, `cmd/resume_test.go`,
  `internal/workflow/workflow_test.go`) checks only `require.Error`/a substring, so wording is
  free to change without breaking unrelated tests.
- **`internal/output` has no existing test file** — a gap and a freedom for the redesign.
- Full citations, alternatives considered, and rehydration cues: see `research.md`.

## Per-Phase Technical Notes

### Phase 1.1: Introduce the shared error shape

**File changes:**
- `internal/output/writer.go` — add `ErrorResponse{Error bool, Code, Message, Resource string,
  State *StateInfo, NextAction string}` and `StateInfo{Current string, ValidActions []string}`;
  add `(*ErrorResponse) Error() string` (returns `Message`, satisfies the `error` interface); add
  constructor `NewError(code, message string) *ErrorResponse` plus fluent
  `WithResource(name string)`, `WithState(current string, valid []string)`,
  `WithNextAction(text string)` methods, each returning `*ErrorResponse` for chaining.
- `internal/output/writer.go:37-88` (`Write`) — after marshaling `v` and before the optional
  field filter, if `v` is not already an `*ErrorResponse`, inject `"error": false` into the
  marshaled object (unmarshal to `map[string]json.RawMessage`, set the key, re-marshal) so every
  existing success caller of `Write`/`WriteResult` gets the discriminant with no caller changes.
- `internal/output/writer.go:29-31` (`WriteError` free function) — delete; superseded by the new
  `WriteFailure(w io.Writer, er *ErrorResponse, fieldsJSON string) error`, which marshals `er`
  (already carrying `Error: true` from the constructor) through the same field-filter path as
  `Write`.
- New file `internal/output/writer_test.go` (package currently has none) — unit tests: discriminant
  injection on a representative existing result shape; `ErrorResponse` builder chaining produces
  the expected JSON; field-filter (`--fields`) interaction with both the success-discriminant and
  error paths.

**Complexity:** Medium — new types plus a marshal-injection step in a package with no prior test
coverage.
**Token estimate:** ~15k
**Agent strategy:** Low — single agent, sequential. Small, self-contained package; no benefit to
parallelizing.

### Phase 1.2: Route every command through one response wrapper

**File changes:**
- `cmd/root.go:23-28` (`Execute`) — replace with a testable `runRoot() int`: calls
  `rootCmd.Execute()`; on `nil`, returns `0`; on error, calls `toErrorResponse(err)` and
  `output.WriteFailure(rootCmd.OutOrStdout(), er, globalFields)`, returns `1`. `Execute()` becomes
  `func Execute() { os.Exit(runRoot()) }`. Add `toErrorResponse(err error) *output.ErrorResponse`:
  type-switches on `*output.ErrorResponse` (passthrough), else wraps as
  `output.NewError("internal_error", err.Error())`.
- `cmd/root.go:74-82` (`init`) — add `rootCmd.SilenceErrors = true` and
  `rootCmd.SilenceUsage = true` so the CLI framework's own default error/usage printer never runs.
- `cmd/spec.go:240,312`; `cmd/plan.go:136,212`; `cmd/implement.go:148,224` — remove each
  `return output.WriteError(cmd.ErrOrStderr(), err)`, replace with `return err`.
- `cmd/knowledge.go` (all 15 sites: 161, 165, 180, 184, 188, 204, 208, 223, 227, 231, 234, 246,
  258, 262, 285, 289) — same replacement, each becomes `return err`.
- `cmd/storefile.go`, `cmd/init.go`, `cmd/skill.go` — no code change; these already return raw
  errors and now benefit from the wrapper automatically.
- **Test infrastructure (cross-cutting):** every `cmd/*_test.go` helper that currently calls
  `rootCmd.Execute()` directly switches to calling the new `runRoot()` so tests observe the
  wrapper's formatted output on the captured stdout buffer instead of a raw Go error. Confirmed
  call sites to update: `cmd/implement_test.go:50-62` (`setupImplementCmd`, used broadly across
  spec/plan/implement/knowledge/cross-kind/resume tests), `cmd/spec_test.go:60-74`
  (`runSpecNewForTest`), `cmd/knowledge_test.go:109-116` (`runKnowledge`), and the equivalent
  helpers in `cmd/plan_test.go`, `cmd/cross_kind_test.go`, `cmd/resume_test.go`. Each helper's
  return contract changes from "the raw error `rootCmd.Execute()` returned" to "the parsed
  response body plus whether it carried `error: true`" — callers that branched on a non-nil `err`
  today branch on the parsed response's discriminant instead.
- `cmd/knowledge_test.go:342-371` (`TestKnowledgeRead_MissingDataEmitsErrorEnvelope`,
  `TestKnowledgeRead_UnknownScopeEmitsErrorEnvelope`) — rewritten to assert the `ErrorResponse`
  JSON now appears on **stdout** with `"error": true`, not on stderr.

**Complexity:** High — touches every command file plus the widest-reaching test-infrastructure
change in the plan.
**Token estimate:** ~35k
**Agent strategy:** Medium — 2-3 parallel agents: (a) `cmd/root.go` + `toErrorResponse` +
`SilenceErrors`/`SilenceUsage`, (b) the mechanical `WriteError`-call-site removals across
spec.go/plan.go/implement.go/knowledge.go, (c) test-helper rewiring across the `*_test.go` files.
(a) must land before (c)'s helpers will compile against `runRoot()`, so integrate/verify
sequentially once all three land.

### Phase 2.1: State-aware rejected-transition messages

**File changes:**
- `internal/workflow/workflow.go:152-166` (`Next`) and `:168-185` (`Goto`) — wrap the error from
  `w.FSM.Event(...)`: when it is a transition-rejection error (`fsm.InvalidEventError` or
  `fsm.UnknownEventError`), build
  `output.NewError("invalid_transition", <message naming the event and current step>)
   .WithState(w.Current(), <legal next step names>)
   .WithNextAction(<suggested goto command using the legal next step(s)>)`
  using `w.FSM.AvailableTransitions()` (already computed internally at `workflow.go:153`) and
  `w.StepNames()` (`workflow.go:270-276`) to populate `ValidActions`; return that in place of the
  raw library error. Errors from step callbacks (store/template/etc., already surfaced via
  `after_`/`before_` callbacks — see Phase 3.1) pass through unchanged; only the FSM library's own
  transition-rejection errors are translated here.
- No import-cycle risk: `internal/output` does not import `internal/workflow` (confirmed during
  discovery), so `internal/workflow` importing `internal/output` for `ErrorResponse` is safe.
- `internal/workflow/workflow_test.go:125-131` (`TestGotoInvalidStepFails`),
  `:160-171` (`TestGotoBackwardFails`) — extend beyond `require.Error(t, err)` to assert the
  returned error is an `*output.ErrorResponse` with a non-empty `State.Current` and a
  `State.ValidActions` containing the actually-legal next step(s).

**Complexity:** Medium.
**Token estimate:** ~12k
**Agent strategy:** Low — single agent, sequential. Isolated to one package, small surface.

### Phase 2.2: Name the resource in resource-specific failures

**File changes:**
- `internal/steps/spec/identifier.go:185` and `internal/knowledge/set.go:240` — at each existing
  `errors.Is(err, store.ErrNotFound)` check, construct
  `output.NewError("not_found", <message>).WithResource(<path already in scope at that call
  site>)` instead of propagating the generic sentinel-derived message. `store.ErrNotFound` itself
  (`internal/store/store.go:11-12`) stays a generic sentinel for the `errors.Is` check — its
  wording is not what reaches the caller.
- `internal/store/store.go:84` (`"path escapes store root"`) — include the attempted path
  (already in scope at that call site) directly in the message.
- `cmd/storefile.go:65-77` (`file read` `RunE`) — wrap the error from `st.Read(...)` as
  `output.NewError("not_found", ...).WithResource(args[0])` instead of returning the raw store
  error unrewritten. This closes the concrete gap found in discovery: today this path returns
  `store.ErrNotFound` (`"not found"`) with no filename attached.

**Complexity:** Low.
**Token estimate:** ~8k
**Agent strategy:** Low — single agent, sequential.

### Phase 2.3: Fold in-progress/wrong-workflow detection into the shared shape

**File changes:**
- `cmd/resume.go:20-27` (`ResumeReport`) — remove the type entirely.
- `cmd/resume.go:64-99` (`emitResumeReport`) — changes from writing a `ResumeReport` via
  `out.WriteResult(...)` to *returning* an `*output.ErrorResponse` (code distinguishing
  same-kind-in-progress vs. cross-kind-mismatch, `State{Current: state.CurrentStep, ValidActions:
  ...}`, `NextAction` populated from the existing rendered instruction text).
  `resumeInstruction`/`mismatchInstruction` (`resume.go:38-62`) are unchanged — their rendered
  template output becomes the `NextAction` value instead of the old `Instruction` field.
- `cmd/resume.go:142-161` (`resumeOrClear`) and `:172-181` (`guardKind`) — the `(handled bool, err
  error)` contract is unchanged in shape, but the "handled" path's `err` is now the constructed
  `*output.ErrorResponse` instead of `nil` (today they write the report as a side effect and
  return `nil`); callers keep the same `if handled { return err }`-style branching, just
  propagating a non-nil, already-formatted error instead of a side-effect-then-nil.
- `cmd/spec.go`, `cmd/plan.go`, `cmd/implement.go` — the existing `new`/`goto`/`status` call sites
  that call `resumeOrClear`/`guardKind` and branch on `handled` need no branching-logic changes,
  only to propagate the now-non-nil `err` on the handled path instead of returning `nil`.
- `cmd/cross_kind_test.go` (`TestSpecNew_CrossKindReturnsMismatchReport`,
  `TestSpecGoto_CrossKindRefusesAndPreservesState`, `TestMismatchInstruction_RendersBothPathsAcrossKinds`)
  and `cmd/resume_test.go` — rewritten to unmarshal the new shape (`error`, `code`, `state.current`,
  `next_action`) in place of the old flat `ResumeReport` fields (`resumable`, `kind`,
  `requested_kind`, `current_step`, `instruction`).
- `.claude/skills/spek-plan/SKILL.md:47`, `.claude/skills/spek-new/SKILL.md:47`,
  `.claude/skills/spek-implement/SKILL.md:42` — each describes the old resume-report shape
  (`"resumable": true` plus `kind`/`name`/`current_step`/`instruction`) that agents are told to
  expect on resume; update each to describe the new shape instead (`"error": true`, a code
  identifying the in-progress/cross-kind case, `state.current`/`state.valid_actions`, and
  `next_action` in place of `instruction`), so the skills that drive this very workflow stay
  accurate.

**Complexity:** Medium.
**Token estimate:** ~20k
**Agent strategy:** Low — single agent, sequential. One file's type removal plus its few call
sites, two test files, and three skill docs are all tightly coupled to the same shape change;
parallelizing would just create merge conflicts on the same small file set.

### Phase 3.1: Stop failed steps from silently advancing saved progress

**File changes:**
- `internal/workflow/workflow.go:108-121` — change the callback registration key from
  `"after_"+s.Name` (`workflow.go:110`) to `"before_"+s.Name`, so `step.Callback` runs inside
  `beforeEventCallbacks` instead of `afterEventCallbacks`. No change to the callback body itself —
  per `looplab/fsm@v1.0.3/fsm.go:296-403`, `beforeEventCallbacks` runs before the FSM ever sets
  `f.current = dst` or invokes `enter_state`/`saveState` (`workflow.go:134-143`), so a failing
  step callback's existing `e.Cancel(err)` now genuinely prevents both the transition and its
  persistence.
- `internal/workflow/workflow.go:187-212` (`renderStep`) — no change needed; it invokes
  `step.Callback` directly rather than through the FSM event/callback mechanism at all, so it is
  unaffected by the before_/after_ rewiring. Confirmed via an explicit regression test (below)
  rather than left as an assumption.
- `internal/workflow/workflow_test.go` — new test: register a step whose callback always returns
  an error, fire the transition, assert (a) the error still surfaces to the caller as before, and
  (b) `w.State().CurrentStep` / `w.Current()` remain the source step rather than the
  would-be-destination — the direct regression guard for the bug found in discovery. Existing
  tests (`TestNextAdvancesThroughAllSteps`, `TestGotoForward`, `TestCallbackGotoAdvances`,
  `TestMultiSourceTransition`) re-run unchanged as a regression check on the successful-callback
  path, since none of them depend on before_/after_ ordering.

**Complexity:** Medium — the code change is small, but the correctness property being fixed is
subtle and needs a precise new test.
**Token estimate:** ~14k
**Agent strategy:** Low — single agent, sequential.

### Phase 3.2: Confirm the motivating incident is resolved

**File changes:**
- New test in `cmd/cross_kind_test.go` (or a new `cmd/error_response_test.go`) — reproduce the
  original incident end to end: seed an in-progress workflow, issue a `goto` to an unreachable
  step through `runRoot()` (Phase 1.2), assert the stdout response's `state.current` and
  `state.valid_actions`/`next_action` name the correct recovery command, exit code is non-zero,
  and stderr is empty.
- No further production code change expected; this phase is a verification pass confirming the
  combination of Phases 1.1-3.1 satisfies the spec's Success Metrics and the ten Acceptance
  Criteria across a representative sample of commands already exercised by earlier phases' tests.

**Complexity:** Low.
**Token estimate:** ~6k
**Agent strategy:** Low — single agent, sequential.

## Testing Strategy

- **Phase 1.1** — unit tests in a new `internal/output/writer_test.go`: discriminant injection on
  an existing result shape, `ErrorResponse` builder-chain output, `--fields` filter interaction
  with both paths.
- **Phase 1.2** — integration tests via the new `runRoot()`, across a representative failure in
  each command family (spec, plan, implement, knowledge, file, skill, init): same-stream delivery,
  discriminant correctness, matching exit code, single occurrence of the failure text with no
  framework-generated noise. `cmd/knowledge_test.go`'s two existing envelope tests rewritten to
  match (stdout, not stderr).
- **Phase 2.1** — `internal/workflow/workflow_test.go`'s existing invalid-transition tests extended
  to assert `State.Current`/`State.ValidActions` content, not just `require.Error`.
- **Phase 2.2** — targeted tests on the resource-naming call sites (store identifier lookups,
  knowledge-set lookups, `file read`) asserting the specific name appears in the message.
- **Phase 2.3** — `cmd/cross_kind_test.go` and `cmd/resume_test.go` rewritten against the new
  shape; the three updated skill docs checked by reading them back against the new shape.
- **Phase 3.1** — new regression test asserting persisted state is unchanged when a step callback
  fails; full existing successful-advancement test suite re-run unchanged as a regression check.
- **Phase 3.2** — new end-to-end test reproducing the original incident; a manual sweep confirming
  each of the ten acceptance criteria against a representative command, covered piecemeal by the
  above.
- Both spec Success Metrics are covered by a behavioural test for their mechanical half (see plan.md
  § Testing Approach) plus an explicit Manual flag for their qualitative/production half.

## Project References

- `.spektacular/specs/000030_agent-actionable-errors.md` — the spec this plan implements.
- `.spektacular/specs/000024_resume.md` — source of the `ResumeReport`/resume-instruction pattern
  this plan folds into the new error shape.
- `.spektacular/specs/000005_adhoc_json_protocol.md` — envelope-design prior art, reviewed as
  Architecture Option 3 and not chosen.
- `.spektacular/knowledge/gotchas/goto-to-current-step-is-a-silent-noop.md` — the related gotcha
  considered and deliberately left out of scope (see plan.md § Out of Scope).
- `.spektacular/knowledge/gotchas/fsm-cancel-only-works-before-transition-commits.md` — new
  gotcha written during discovery, documenting the general `looplab/fsm` `Cancel()` semantics
  this plan's Phase 3.1 fix relies on.
- `.spektacular/knowledge/architecture/cli-design-for-ai-agents.md` — pre-existing design ethos
  doc this feature extends into error handling.
- `.claude/skills/spek-plan/SKILL.md`, `.claude/skills/spek-new/SKILL.md`,
  `.claude/skills/spek-implement/SKILL.md` — updated in Phase 2.3 to describe the new response
  shape in place of the removed `ResumeReport` shape.
- `templates/steps/resume.md`, `templates/steps/resume_mismatch.md` — unchanged template content
  reused as the source of the new `ErrorResponse.NextAction` text.

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|-------------|----------------|
| Low | ~10k | Single agent, sequential |
| Medium | ~25k | 2-3 parallel agents |
| High | ~50k+ | Parallel analysis, sequential integration |

## Migration Notes

- `ResumeReport`'s JSON shape (`resumable`, `kind`, `requested_kind`, `name`, `current_step`,
  `instruction`) is removed with no backward-compatible alias, per the spec's explicit constraint
  that the response shape is not required to stay backward-compatible. The three skill docs that
  describe this shape to driving agents (`spek-plan`, `spek-new`, `spek-implement` `SKILL.md`) are
  updated in Phase 2.3 in lockstep, so no external migration step is needed beyond this repo.
- `output.WriteError`'s stderr-based `{"error": "..."}` shape is removed entirely (Phase 1.2). Any
  external tooling that depended on reading errors from stderr in that shape would need to switch
  to reading stdout with the new `ErrorResponse` shape instead — there are no known external
  consumers outside this repo's own skills and tests.

## Performance Considerations

None. The response envelope adds one additional JSON unmarshal/re-marshal per command invocation
(to inject the discriminant field), which is negligible against the cost of a CLI process
invocation. Nothing in this plan touches a hot path, adds I/O, or changes concurrent behavior.
