# Working context — implement for 000030_agent-actionable-errors

## Key decisions / substitutions
- Plan `000030_agent-actionable-errors` (7 phases across 3 milestones) is fully committed to the
  plan store (plan.md, context.md, research.md all written by the earlier planning session — see
  git history for that session's own context.md if deeper planning rationale is needed).
- No `## Changelog` section exists yet in plan.md → this is a first-phase invocation. Pick up at
  Phase 1.1 (Introduce the shared error shape).

## Validation results (read_plan step)
- Structural validation: all 10 required `## ` sections present in plan.md; 7 phase checkboxes
  under Milestones & Phases, each with a working `*Technical detail:*` link resolving to a
  matching `### Phase N.M:` heading in context.md.
- Drift check against working tree: ran a full file:line/symbol verification (23 claims spanning
  internal/output/writer.go, cmd/root.go, cmd/spec.go, cmd/plan.go, cmd/implement.go,
  cmd/knowledge.go, cmd/resume.go, cmd/storefile.go, internal/workflow/workflow.go,
  internal/store/store.go, internal/steps/spec/identifier.go, internal/knowledge/set.go, test
  files, go.mod versions, templates, skill docs, knowledge-base gotchas, import-cycle safety).
  Result: **no blocking mismatches**. Every named file/line/symbol still matches current code
  (a couple of ±1 line drifts, within tolerance). One cosmetic-only inconsistency in the plan's
  own prose: `cmd/knowledge.go` step lists 16 line numbers but the parenthetical says "15 sites" —
  actual count is 16, all 16 lines confirmed correct. Not worth fixing before proceeding; note it
  if writing to that section later.
- Cleared to proceed with implementation with no user decision needed (no mismatches to choose
  between fix/proceed/abandon on).

## Status
- read_plan step complete.
- analyze step complete for Phase 1.1 (Introduce the shared error shape, Low complexity — done
  directly, no sub-agent delegation needed). Confirmed current `internal/output/writer.go`
  (88 lines): `Writer{w, fields}`, `New`, `WriteResult` (delegates to `Write`), `WriteError` (free
  func, `map[string]string{"error": ...}`, to be deleted), `Write` (marshal → optional field
  filter via `--fields`). Test convention confirmed: `github.com/stretchr/testify v1.9.0`,
  `require.*` assertions, `t.TempDir()` for scratch state — matches context.md's Phase 1.1 plan
  (new `internal/output/writer_test.go`, package currently has zero test files).
- Advancing to `implement` for Phase 1.1.

## Phase 1.1 implementation notes
- Fixed a real defect in the plan's own Data Structures spec: `ErrorResponse` can't have both an
  `Error bool` field and an `Error() string` method (Go forbids field/method name collision on the
  same type). Resolved by renaming the Go field to `IsError` while keeping its JSON tag `json:"error"`
  — wire format is unchanged from what the plan specified, only the Go identifier differs. The
  `Error() string` method (required for the `error` interface, used throughout later phases'
  `return err` / `toErrorResponse` type-switch pattern) is untouched.
- Deliberately did NOT delete `WriteError` in this phase, despite context.md's Phase 1.1 file-change
  list saying to delete it. Reason: `cmd/spec.go`, `cmd/plan.go`, `cmd/implement.go`, `cmd/knowledge.go`
  (26 sites total) still call it; deleting now would break the build until Phase 1.2 replaces those
  call sites with `return err`. This also matches plan.md's own Phase 1.1 milestone description
  ("this phase only adds new capability; nothing in the tool uses it yet"), which is inconsistent
  with deleting a function 26 call sites still depend on. `WriteError` deletion is deferred to
  Phase 1.2, done together with its call-site removal, so every phase compiles independently.
- Added `WriteFailure(w io.Writer, er *ErrorResponse, fieldsJSON string) error` per context.md spec —
  marshals an already-built `*ErrorResponse` through the same field-filter path as `Write`.
- `Write` now injects `"error": false` into any marshaled value that is not a `*ErrorResponse`, via
  unmarshal-to-map/set-key/re-marshal, exactly as context.md specified. `*ErrorResponse` values skip
  injection since `NewError` already sets `IsError: true` (marshals to `"error": true`).
- `go build ./...` passes cleanly. No test files written yet (next step).

## Phase 1.1 test step notes
- Delegated to a sub-agent per follow-test-patterns: wrote `internal/output/writer_test.go` (11
  tests, testify/require, hand-written expected JSON literals — no self-referential oracles).
  Covers both Phase 1.1 acceptance criteria (builder produces expected fields; discriminant
  injection on success). `go vet`/`go test ./internal/output/...` clean.
- Running the FULL repo suite (`go test ./...`) after the sub-agent's change surfaced two real
  issues, both fixed directly (not deferred — leaving the suite red is not acceptable mid-phase):
  1. **Bug found and fixed**: `Write`'s discriminant injection unconditionally set `m["error"] =
     false` on any non-`*ErrorResponse` value — including `WriteError`'s own
     `map[string]string{"error": err.Error()}`, whose `"error"` key already holds the failure
     message string. This silently corrupted `WriteError`'s output (string → bool), breaking
     `cmd/knowledge_test.go`'s two envelope tests (`TestKnowledgeRead_MissingDataEmitsErrorEnvelope`,
     `TestKnowledgeRead_UnknownScopeEmitsErrorEnvelope`) which — correctly, per context.md's own
     Phase 1.2 scope — still expect `WriteError`'s pre-1.2 stderr string shape during this phase.
     Fixed by decoupling `WriteError` from `Write` entirely: it now marshals+writes directly,
     unaffected by the new injection logic. This is transient scaffolding — `WriteError` itself is
     deleted in Phase 1.2 per plan, so this fix only needs to hold until then.
  2. **Pre-existing test updated (expected, correct consequence of Phase 1.1's own acceptance
     criterion 2 — "marshaling any existing successful result automatically carries the
     discriminant")**: `cmd/knowledge_test.go:201`
     (`TestKnowledgeSearch_EmptyAndNoMatchQueriesReturnEmptyHits`) asserted the exact literal
     `{"hits": []}` on a success path; now correctly gets `"error": false` injected too. Updated
     the literal to `{"error": false, "hits": []}`. Grepped the whole repo for other exact-JSON
     success assertions (`JSONEq` / literal map comparisons) that might have the same issue — this
     was the only one outside the new `internal/output/writer_test.go` file itself.
- Full repo suite (`go build ./...`, `go vet ./...`, `go test ./...`) green after both fixes.
- Next: advance to `verify` step for Phase 1.1.

## Phase 1.1 verify step notes
- Sub-agent ran `go build ./...`, `go vet ./...`, `go test ./...` — all green, no regressions
  across all 13 tested packages. Both acceptance criteria mapped to passing tests (see test-file
  function names in the test-step notes above).
- Phase 1.1 complete. Advancing to `update_plan`.

## Phase 1.1 update_plan step notes
- plan.md's Phase 1.1 heading and both acceptance-criteria checkboxes flipped to `[x]` via
  `plan file write` (staged at `.spektacular/tmp/plan_update.md`, scratch file removed after).
  Verified by reading the checked-in version back.
- Advancing to `update_changelog`.

## Phase 1.1 update_changelog step notes
- Appended first `## Changelog` section (didn't exist yet) to plan.md with a full Phase 1.1 entry
  (what/deviations/files/discoveries) covering the field-rename fix, the deferred WriteError
  deletion, the WriteError-corruption bug fix, and the pre-existing test update.
- 6 unchecked phases remain (1.2 through 3.2). Per established preference (drive multi-step
  workflows straight through without pausing for confirmation on non-design-decision points) and
  Auto Mode being active this session, looping straight to `analyze` for Phase 1.2 rather than
  asking to continue.
- Phase 1.1 (Introduce the shared error shape) is DONE: plan.md checkboxes flipped, changelog
  entry written, all tests green.

## Phase 1.2 implement step notes
- **Real scope correction found during analysis**: context.md's Phase 1.2 file list names a
  handful of test helpers (`setupImplementCmd`, `runSpecNewForTest`, `runKnowledge`, "equivalent
  helpers in plan_test.go/cross_kind_test.go/resume_test.go") as needing to switch from
  `rootCmd.Execute()` to the new wrapper. Actual grep found ~45 *inline* `rootCmd.Execute()` calls
  across 9 test files (`implement_test.go`, `knowledge_test.go`, `cross_kind_test.go`,
  `plan_file_test.go`, `file_test.go`, `init_test.go`, `spec_test.go`, `plan_test.go`) — far more
  call sites than the file:line inventory suggested. This is a scale surprise, not a design
  contradiction (the transformation shape context.md describes — tests move from reading the raw
  Go error to reading the wrapper's formatted response — is unchanged), so proceeded without
  stopping rather than treating it as a STOP-on-mismatch case.
- **Turned out not to matter in practice**: ran the full suite right after the production change
  and only 2 tests broke (`TestKnowledgeRead_MissingDataEmitsErrorEnvelope`,
  `TestKnowledgeRead_UnknownScopeEmitsErrorEnvelope` — both already named in context.md's own
  Phase 1.2 scope). Reason: `SilenceErrors`/`SilenceUsage` only suppress cobra's own printing, they
  don't change what `rootCmd.Execute()` *returns* — every other call site checks the Go `error`
  value itself (still returned correctly, in fact *more* correctly now that `RunE` handlers
  `return err` instead of the old `WriteError`-then-return-nil pattern), not the old
  stderr-formatted text. Only tests that specifically parsed the old inline-`WriteError`-written
  stderr JSON needed rewriting. **Learning for future phases**: don't assume a wide file:line
  undercount in context.md means wide test breakage — verify against the actual `go test` run
  before doing large-scale rewrites.
- Production changes: `cmd/root.go` — `Execute()` now `os.Exit(runRoot())`; new `runRoot() int`
  (calls `rootCmd.Execute()`, on nil returns 0, on error calls `toErrorResponse` +
  `output.WriteFailure(rootCmd.OutOrStdout(), er, globalFields)`, returns 1); new
  `toErrorResponse(err) *output.ErrorResponse` (passthrough if already `*ErrorResponse`, else
  `output.NewError("internal_error", err.Error())`); `init()` now sets
  `rootCmd.SilenceErrors = true` / `rootCmd.SilenceUsage = true`.
- Mechanical replacement: all 22 `return output.WriteError(cmd.ErrOrStderr(), err)` sites across
  `cmd/spec.go` (2), `cmd/plan.go` (2), `cmd/implement.go` (2), `cmd/knowledge.go` (16) →
  `return err`. `output` import still needed in all four files (still call `output.Write`/
  `output.New` elsewhere) — no import cleanup needed.
- Deleted `output.WriteError` (deferred from Phase 1.1 per that phase's own decision log) — zero
  remaining callers confirmed by repo-wide grep before deleting.
- Fixed the 2 broken tests: `runKnowledge` helper (`cmd/knowledge_test.go`) now calls the new
  `runRoot()` instead of `rootCmd.Execute()` directly, and on non-zero exit unmarshals stdout into
  an `*output.ErrorResponse` to use as its returned `error` (so existing `require.NoError`/
  `require.Error` callers elsewhere in the file need no changes — only the two tests that inspected
  the error shape directly were rewritten to expect stdout + `IsError`/`Message` instead of the old
  stderr + bare string `error` field).
- Manually smoke-tested the real binary (`go build -o /tmp/spek-test .`): missing `--data`,
  a success path, and an unknown-subcommand case. Confirmed: single JSON response on stdout only,
  process exit code matches (1 for failure, 0 implied by earlier `go test` runs for success),
  no duplicate cobra usage/help text on unknown-command (previously this would print `Error: ...`
  + full usage twice). Binary and scratch dir cleaned up after.
- `go build ./...`, `go vet ./...`, `go test ./...` all green.
- Next: advance to `test` step — delegate writing dedicated integration tests for Phase 1.2's
  3 acceptance criteria (same-stream delivery, discriminant+exit-code correctness, no duplicate
  printing) across a representative failure in each command family (spec, plan, implement,
  knowledge, file, skill, init), per context.md's own Testing Strategy for this phase.

## Phase 1.2 test step notes
- **Process note for future phases**: hit a real FSM rejection mid-implementation — called
  `implement goto {"step":"test"}` directly from `analyze` without going through `implement` first
  (I'd written all the Phase 1.2 code already but never called the intermediate `goto implement`).
  Got `"event test inappropriate in current state analyze"` — the exact class of unhelpful raw-FSM
  error this whole plan exists to fix (Phase 2.1 will add state-aware next-action text; doesn't
  exist yet at this point in implementation). Recovered by calling
  `implement goto {"step":"implement"}` first, then `{"step":"test"}`. **Lesson: always call each
  intermediate `goto` even if the code for that step is already done — the FSM state must track
  actual step-by-step progression, not just code completeness.**
- Delegated to a sub-agent: wrote `cmd/root_test.go` (new file, 399 lines) testing `runRoot`'s
  wrapper behavior specifically (as opposed to each command's business logic, covered elsewhere).
  Three tests, each iterating all 7 command families (spec/plan/implement/knowledge/file/skill/
  init): `TestWrapper_SuccessAndFailureBothStreamOnStdoutOnly` (criterion 1),
  `TestWrapper_ErrorDiscriminantAndExitCode` (criterion 2, hand-checks `"error"` bool + exit code),
  `TestWrapper_FailureIsPrintedExactlyOnceWithNoCobraBoilerplate` (criterion 3, plus an
  unknown-top-level-command case proving cobra's own printer is silenced).
  Added `resetPlanCommandFlags`/`resetImplementCommandFlags` test helpers (mirroring existing
  `resetSpecCommandFlags`/`resetKnowledgeFlags`) since plan/implement flags persist across
  `Execute()` calls within one test binary.
- Sub-agent found and worked around a real, pre-existing test-only quirk (not a production bug):
  `spec new`'s resume-check runs before its name-validation, so two `spec new` calls in the same
  directory turn a would-be validation failure into a resume-report success. Fixed by using
  separate `t.TempDir()`s per invocation in the new tests — no production code touched.
- I independently reviewed the full `cmd/root_test.go` file and reran `go build ./...`,
  `go vet ./...`, `go test ./...` myself — all green, no regressions.
- Next: advance to `verify`.

## Phase 1.2 verify step notes
- Sub-agent ran full build/vet/test plus the phase-specific `TestWrapper_*` tests, mapped each to
  its acceptance criterion, and independently smoke-tested the real binary against an unknown
  top-level command (exit 1, single `"error": true` JSON on stdout, stderr completely empty —
  confirms `SilenceErrors`/`SilenceUsage` actually silences cobra's default printer outside the
  test harness, not just inside it). All green.
- Phase 1.2 complete. Advancing to `update_plan`.

## Phase 1.2 update_plan step notes
- plan.md's Phase 1.2 heading and all 3 acceptance-criteria checkboxes flipped to `[x]`, verified
  by reading back. Advancing to `update_changelog`.

## Phase 1.2 update_changelog step notes
- Appended Phase 1.2 changelog entry (what/deviations/files/discoveries — including the ~45 vs
  handful call-site-count correction and the live FSM-rejection discovery).
- 5 unchecked phases remain (2.1 through 3.2). Milestone 1 (single channel + reliable discriminant
  + correct exit code) is now fully DONE. Looping straight to `analyze` for Phase 2.1 (Milestone 2:
  error text quality) per established preference + Auto Mode.

## Phase 2.1 implement step notes
- Confirmed `looplab/fsm@v1.0.3`'s exact error types by reading `$GOMODCACHE/.../errors.go`:
  `InvalidEventError{Event, State}` and `UnknownEventError{Event}` are value types (not pointers)
  returned by `FSM.Event()` for a genuinely rejected transition; `CanceledError{Err}` wraps a step
  callback's own domain error (from `e.Cancel(err)`) and must NOT be translated — it already
  `Unwrap()`s to the real cause, so it passes through `errors.As` checks for the other two types
  untouched, exactly as context.md specified ("errors from step callbacks... pass through
  unchanged").
- Added `translateTransitionError(event string, err error) error` and
  `nextActionForSteps(valid []string) string` to `internal/workflow/workflow.go`. Both `Next()` and
  `Goto()` now route their `w.FSM.Event(...)` error through `translateTransitionError` before
  returning. On a real `InvalidEventError`/`UnknownEventError`, it builds
  `output.NewError("invalid_transition", ...).WithState(current, validSteps).WithNextAction(...)`
  using `w.FSM.AvailableTransitions()` (already computed, no new state) and `w.cfg.Command`/
  `w.cfg.Kind` to render a concrete `<command> <kind> goto --data '{"step":"<step>"}'` — same style
  as the resume template's `{{config.command}} {{kind}} goto ...` convention, so next-action text
  is consistent across the whole tool. Any other error (including `CanceledError`) passes through
  unchanged.
- No import cycle: confirmed `internal/output` doesn't import `internal/workflow` (checked during
  the read_plan drift check too) — `internal/workflow` importing `internal/output` is safe.
- `go build ./...`, `go vet ./...`, `go test ./...` (full repo) all green — zero regressions,
  including the two existing tests (`TestGotoInvalidStepFails`, `TestGotoBackwardFails`) that only
  assert `require.Error`, so they still pass unchanged (will be extended for stronger assertions in
  the test step).
- Live smoke test with the real binary reproduced the plan's own motivating incident: `spec new`
  then `spec goto --data '{"step":"milestones"}'` (a step unreachable from `overview`) now returns
  `{"error":true,"code":"invalid_transition","message":"cannot run step \"milestones\" from the
  current step \"overview\"","state":{"current":"overview","valid_actions":["requirements"]},
  "next_action":"run: spektacular spec goto --data '{\"step\":\"requirements\"}'"}` — a direct,
  actionable improvement over the original raw FSM wording
  (`{"error":"event milestones inappropriate in current state overview"}`) that research.md
  documented as the motivating incident. Scratch binary/dir cleaned up after.
- Next: advance to `test` step — extend `TestGotoInvalidStepFails`/`TestGotoBackwardFails` per
  context.md's Phase 2.1 spec to assert `State.Current`/`State.ValidActions` content, not just
  `require.Error`.

## Phase 2.1 test step notes
- **Process reminder (recurred)**: hit the same FSM-sequencing slip as Phase 1.2 — jumped from
  `analyze` straight to `test` without calling `goto implement` first, even though the code was
  already written. The tool's own new Phase 2.1 error caught it immediately and told me the exact
  fix (`run: go run . implement goto --data '{"step":"implement"}'`) — a live demonstration of this
  phase's own feature working correctly on itself. **Firm habit going forward: always call the
  intermediate `goto` for the step you're about to do BEFORE doing that step's work, never after —
  do this before writing any code, not just before advancing past it.**
- Delegated to a sub-agent: extended `TestGotoInvalidStepFails` and `TestGotoBackwardFails` in
  `internal/workflow/workflow_test.go` (both previously only `require.Error`) to assert the
  returned error `errors.As`-unwraps to `*output.ErrorResponse` with hand-verified
  `State.Current`/`State.ValidActions` (checked against `testSteps`' own Src/Dst wiring, not
  re-derived from the code under test). Added a new `TestNextCallbackErrorNotTranslated` proving a
  step callback's own domain error passes through `translateTransitionError` completely unchanged
  (not turned into an `invalid_transition` ErrorResponse) — confirmed against `looplab/fsm`
  v1.0.3 source that `after_` callback errors aren't wrapped in `fsm.CanceledError`, so
  `Event()` returns the raw callback error directly.
- I independently reran `go build ./...`, `go vet ./...`, `go test ./...` (full repo) — all green.
  Diff is 53 lines added to one test file, no production code touched.
- Next: advance to `verify`.

## Phase 2.1 verify + update_plan step notes
- Sub-agent ran full build/vet/test plus the phase-specific tests, mapped both acceptance criteria
  to their asserting tests, and independently smoke-tested the real binary (`spec new` then a
  rejected `spec goto`) confirming the exact motivating-incident-fixing JSON shape. All green.
- Phase 2.1 (State-aware rejected-transition messages) is DONE: both acceptance criteria verified,
  Milestone 2's first phase complete. Next: flip plan.md checkboxes and continue the changelog loop
  to Phase 2.2.

## Phase 2.2 analyze + implement step notes
- **Real mismatch found and resolved without stopping**: context.md's Phase 2.2 file list named 4
  touchpoints. Checked all 4 against the actual code before writing anything:
  1. `internal/steps/spec/identifier.go:185` (`errors.Is(err, store.ErrNotFound)`) — MISMATCH.
     This is a success path (`nextCounterFromStore`): "directory doesn't exist yet" is swallowed to
     `return 1, nil` (start counting from 1), never propagated as an error to any caller. Applying
     the planned "wrap with WithResource" treatment here would be a no-op at best (dead code — the
     branch never returns an error) or would wrongly turn a benign case into a failure.
  2. `internal/knowledge/set.go:240` (same check) — MISMATCH, same shape: `readCategories` treats
     a missing category directory as "skip it" (`continue`), not a failure.
  3. `internal/store/store.go:84` (`"path escapes store root"`) — CONFIRMED VALID, a real
     propagated error with no path attached.
  4. `cmd/storefile.go`'s `file read` `RunE` — CONFIRMED VALID, the exact gap context.md described:
     `st.Read(...)`'s raw `store.ErrNotFound` reached the caller as bare `"not found"`, no filename.
  Grepped the whole repo for every `ErrNotFound` reference (5 total) to confirm sites 1 and 2 are
  the ONLY two `errors.Is(err, store.ErrNotFound)` callers anywhere, and neither is a genuine
  "requesting a thing that doesn't exist" failure path — confirming this isn't a narrow oversight,
  it's the plan's full description of those two sites being wrong. Resolved by implementing only
  the 2 valid touchpoints (3 and 4) and skipping 1 and 2 — this still fully satisfies the phase's
  single acceptance criterion ("requesting a file/entry/identifier that doesn't exist produces a
  message naming it"), since 1 and 2 were never failure paths the criterion could apply to. Treated
  this as a mechanical scope correction (justified by exhaustive grep + reading actual behavior),
  not a design decision requiring a pause — the direction was unambiguous once the code was read.
- Implemented: `internal/store/store.go`'s `abs()` now returns
  `fmt.Errorf("path %q escapes store root", path)` (was a bare `"path escapes store root"`).
  `cmd/storefile.go`'s `file read` now checks `errors.Is(err, store.ErrNotFound)` and returns
  `output.NewError("not_found", fmt.Sprintf("file %q not found", args[0])).WithResource(args[0])`
  instead of the raw store error.
- `go build ./...`, `go vet ./...`, `go test ./...` (full repo) all green, zero regressions.
- Live smoke test: `spec file read does-not-exist.md` now returns
  `{"error":true,"code":"not_found","message":"file \"does-not-exist.md\" not found",
  "resource":"does-not-exist.md","next_action":""}` — filename present in both the message and the
  structured `resource` field. Scratch binary/dir cleaned up after.
- Next: advance to `test` step (already at `implement` state — transitioned there before writing
  any code this time, per the Phase 1.2/2.1 process lesson).

## Phase 2.2 test step notes
- Delegated to a sub-agent: added `TestPathTraversal_ErrorNamesAttemptedPath` in
  `internal/store/store_test.go` (asserts exact error message includes the attempted escaping
  path, across Read/Write/Delete/List) and `TestSpecFileRead_MissingFileNamesResourceInError` in
  `cmd/file_test.go` (drives `spec file read missing.md` through `runRoot()`, asserts exit 1 +
  `ErrorResponse{IsError:true, Code:"not_found", Message contains "missing.md", Resource:
  "missing.md"}`). Both reused existing fixtures/helpers, hand-maintained expected strings.
  I independently reran build/vet/full test suite — green, 45-line diff across 2 files.
- Next: advance to `verify`.

## Phase 2.2 verify step notes
- Sub-agent ran full build/vet/test plus the phase-specific tests, confirmed the single acceptance
  criterion is covered by both new tests, and smoke-tested the real binary
  (`spec file read nonexistent-doc.md` → filename present in message + resource field). All green.
- Phase 2.2 (Name the resource in resource-specific failures) is DONE, with the documented 2-of-4
  scope correction (identifier.go/set.go sites skipped as non-failure paths). Next: flip plan.md
  checkboxes and continue the loop to Phase 2.3.

## Phase 2.3 implement step notes
- **Scope correction (again, same shape as Phase 1.2)**: context.md named `cmd/cross_kind_test.go`
  and `cmd/resume_test.go` as needing `ResumeReport` updates, but `plan_test.go`, `implement_test.go`,
  and `spec_test.go` also reference it (grepped first, confirmed 5 files / ~9 sites total). Handled
  directly rather than treating as a blocking mismatch — same lesson as before: context.md's
  file:line inventories are a useful map, not an exhaustive guarantee; grep before delegating.
- Removed `ResumeReport` entirely from `cmd/resume.go`. `emitResumeReport(command, expectedKind
  string, state *workflow.State) error` (dropped its `*cobra.Command` param — nothing writes
  anymore) now returns an `*output.ErrorResponse`: `code` is `"workflow_in_progress"` (same-kind)
  or `"cross_kind_workflow_in_progress"` (cross-kind), `.WithResource(name)`,
  `.WithState(state.CurrentStep, nil)` (ValidActions left nil/empty — not applicable here, the
  concept doesn't map to "legal next steps" the way it does for Phase 2.1's FSM case),
  `.WithNextAction(instruction)` where `instruction` is the SAME `resumeInstruction`/
  `mismatchInstruction` template rendering as before — only the outer shape changed, not the
  template content, per plan.md's design.
- `resumeOrClear`/`guardKind` also dropped their `*cobra.Command` param (no longer needed) and now
  return the `*output.ErrorResponse` as `err` on the `handled=true` path instead of writing +
  `nil`. Updated all 9 call sites across `cmd/spec.go`/`cmd/plan.go`/`cmd/implement.go`: dropped
  the `cmd,` argument and changed `if handled { return nil }` → `if handled { return err }` — same
  branching shape, just propagating the now-non-nil error, exactly as context.md specified.
- Updated all three skill docs (`spek-plan`, `spek-new`, `spek-implement` `SKILL.md`) per this
  phase's own explicit acceptance criterion — replaced every "resume report" /
  `resumable`/`kind`/`current_step`/`instruction` reference with the new shape:
  `{"error": true, "code": "workflow_in_progress"|"cross_kind_workflow_in_progress", "message",
  "resource", "state": {"current", "valid_actions"}, "next_action"}`. `spek-new`'s "one of two
  things comes back" framing (previously resume-report-as-success vs. error) was reworked since
  BOTH outcomes are now failures — distinguished by `code` instead of by shape. Grepped afterward
  to confirm zero remaining "resumable"/"resume report" references in any skill doc.
- Delegated the 5-file, ~9-site test migration to a sub-agent (all now use `runRootCmd`/
  `output.ErrorResponse` instead of `rootCmd.Execute()`/`ResumeReport`, preserving each test's
  original protected property under the new shape — e.g. `TestSpecNew_CleanDirHasNoResumeReport`
  became `TestSpecNew_CleanDirSucceedsWithoutError` since a clean dir now produces a plain success
  envelope, not a resume-shaped anything). I independently reran build/vet/full test suite — green.
  Also cleaned up 3 stale "ResumeReport" mentions left in `cmd/resume.go`'s own doc comments
  (referring to the deleted type name) after the sub-agent's pass.
- Live smoke test: both same-kind (`spec new` twice) and cross-kind (`spec new` then `plan new`)
  scenarios confirmed — `"error": true`, correct `code`, `resource` (instance name), `state.current`
  (step), and `next_action` carrying the full, unchanged rendered instruction template. Exit 1 in
  both cases. Scratch binary/dir cleaned up after.
- Caught the FSM state up with the actual `goto` sequence (`implement` → `test` → `verify`), since
  I'd done both the code and test work before making the calls this time too.

## Phase 2.3 verify step notes
- Sub-agent ran full build/vet/test, confirmed zero remaining `ResumeReport` type references (only
  the kept `emitResumeReport` function name) and zero stale "resumable"/"resume report" mentions in
  any skill doc, and smoke-tested both same-kind and cross-kind scenarios live — all 3 acceptance
  criteria confirmed. The sub-agent hit and self-corrected a shell-state mistake (a `$SCRATCH` env
  var didn't persist across separate Bash calls, so a `cd` silently no-opped) — I independently
  verified via `git status`/`git diff --stat .spektacular/state.json` that no stray files or state
  corruption resulted; the diff is exactly our own session's expected step progression.
- Phase 2.3 (Fold in-progress/wrong-workflow detection into shared shape) is DONE. This completes
  Milestone 2 (error text quality) entirely. Next: flip plan.md checkboxes and continue the loop to
  Phase 3.1 (Milestone 3: workflow step-commit ordering fix).

## Phase 3.1 implement + test step notes
- The fix itself: `internal/workflow/workflow.go`'s `New()` — changed `callbacks["after_"+s.Name]`
  to `callbacks["before_"+s.Name]` (one line, per the plan and the pre-existing gotcha doc
  `fsm-cancel-only-works-before-transition-commits.md`), plus an explanatory comment block.
  Confirmed via `looplab/fsm@v1.0.3` source: `beforeEventCallbacks` runs and returns before
  `f.current = dst`/`enterStateCallbacks` (saveState) ever execute, so `Cancel()` from `before_`
  genuinely prevents both — unlike `after_`, which runs after the transition and persistence are
  already done.
- **Real, predicted-in-advance side effect confirmed and fixed**: read the library source before
  making the change and found `beforeEventCallbacks` wraps a canceled error as
  `fsm.CanceledError{Err: err}` (`.Error()` becomes `"transition canceled with error: <original>"`),
  whereas `afterEventCallbacks` returns the raw `e.Err` unwrapped. This meant Phase 2.1's own
  `TestNextCallbackErrorNotTranslated` (which asserted exact string equality on the callback error)
  would break the moment this phase's wiring changed — confirmed by running the suite immediately
  after the one-line change. Fixed by switching that assertion from `require.Equal(t,
  callbackErr.Error(), err.Error())` to `require.ErrorIs(t, err, callbackErr)`, which correctly
  follows `CanceledError.Unwrap()` to find the original error regardless of wrapping. This is a
  durable lesson for the codebase: any future check on a step-callback error's exact text must
  account for FSM-library wrapping; `errors.Is`/`errors.As` are robust to it, string equality isn't.
- Delegated the phase's actual regression test to a sub-agent:
  `TestFailedStepDoesNotAdvancePersistedState` in `internal/workflow/workflow_test.go` — a 3-step
  workflow where step "two"'s callback always fails; asserts (a) the error surfaces correctly and
  isn't mistranslated, (b) `wf.Current()`/`wf.State().CurrentStep` remain at "one" in memory, and
  (c) — the actual gap this test closes — independently reading `state.json` off disk (not just
  trusting the in-memory struct) confirms it ALSO still shows "one", with "one" absent from
  `CompletedSteps`. Also reran the 4 pre-existing successful-advancement tests
  (`TestNextAdvancesThroughAllSteps`, `TestGotoForward`, `TestCallbackGotoAdvances`,
  `TestMultiSourceTransition`) unmodified — all still pass, confirming criterion 2.
  `go build ./...`, `go vet ./...`, `go test ./...` (full repo, verified independently) all green.
- Live smoke test: a full `spec new` → `spec status` round trip confirms normal successful
  advancement is completely unaffected by the before_/after_ rewiring (progress "1/10", correct
  completed/current/pending step breakdown). The actual bug-fix proof is the unit test above — a
  CLI-level repro of a mid-workflow step-callback failure would require a deliberately-broken step
  config not exposed through any real command, so the targeted unit test is the right evidence
  here, not a live repro.
- Next: advance to `verify`.

## Phase 3.1 verify step notes
- Sub-agent ran full build/vet/test plus named tests, confirmed both acceptance criteria are
  covered, and specifically confirmed the on-disk `state.json` check in
  `TestFailedStepDoesNotAdvancePersistedState` is independent of the in-memory struct (the part
  that actually proves the bug is fixed, not just that the caller sees an error). All green.
- Phase 3.1 (Stop failed steps from silently advancing saved progress) is DONE. Next: flip plan.md
  checkboxes and continue the loop to Phase 3.2 — the final phase.

## Phase 3.2 implement + test step notes
- No production code changes (correctly — plan.md's own Phase 3.2 file-change list says "No further
  production code change expected"). Transitioned through `implement` as a no-op, straight to `test`.
- Delegated the final regression test: new `cmd/error_response_test.go`,
  `TestGoto_UnreachableStepReturnsActionableError`. Drives the REAL stack end-to-end (`runRootCmd`
  → `runRoot()` → cobra → `internal/workflow`, not calling the workflow package directly) — seeds a
  `plan new` workflow (lands on `overview`), issues `plan goto --data '{"step":"milestones"}'`
  (unreachable — hand-verified against `internal/steps/plan/steps.go`: milestones' Src is
  `["testing_approach"]`, the only step reachable from `overview` is `discovery`). Asserts: exit 1,
  stderr completely empty, stdout is a single valid `ErrorResponse` JSON object with `error: true`,
  `state.current == "overview"`, `state.valid_actions == ["discovery"]` (hand-verified, not derived
  from the code under test), `next_action` non-empty and contains a runnable `goto --data` command
  naming `discovery`, and the failure text appears exactly once (no double-printing).
- Confirmed against plan.md's own Testing Approach § Success Metrics language before finalizing —
  the sub-agent read plan.md via `plan file read` (never the Read tool) to check its assertions
  matched what that section claims this regression test proves.
- I independently reran the specific test plus the full repo suite — both green.
- Next: advance to `verify`. This is the plan's final phase — verification here effectively
  confirms the whole plan's two Success Metrics (per plan.md § Testing Approach): the structural
  half (every error category carries a non-empty NextAction and, where applicable, State/Resource)
  is machine-verifiable and covered across this session's tests; the qualitative half (a fresh
  agent session, shown only a captured error, proposes the correct next command unaided) is
  explicitly flagged Manual in plan.md and cannot be certified by this session's automated
  verification — noting this so it isn't mistaken for a gap in this phase's own work.

## Phase 3.2 verify step notes
- Full build/vet/test green, phase-specific test passes. Final live smoke test reproduced the
  plan's original motivating incident exactly (`spec new` → `spec goto` to an unreachable step) and
  confirmed the fix end-to-end: exit code 1 (was 0 before this plan — a failure that looked like
  success), stderr completely empty, single clean JSON on stdout naming the current step
  (`overview`), the actually-reachable next step (`requirements`), and a concrete runnable
  next_action command. A second, unrelated failure mode (`spec file read` on a missing file, from
  Phase 2.2) was also re-checked and still works — no end-of-plan regression.
  I independently confirmed the repo's working tree is clean after the sub-agent's smoke test (only
  the expected `.spektacular/state.json` diff from this implement session's own step progression).
- **All 7 phases across all 3 milestones are now complete.** Plan 000030_agent-actionable-errors
  is fully implemented, tested, and verified.

## Final wrap-up notes
- plan.md's Phase 3.2 checkbox + its single acceptance criterion flipped to `[x]`; final changelog
  entry appended summarizing all 7 phases. `grep -c "^#### - \[ \] Phase"` on the checked-in plan.md
  now returns 0 — every phase checked off.
- Next: advance to `update_repo_changelog` (the repo-level release-note step) since this was the
  last unchecked phase.
- `update_repo_changelog` step complete: prepended a user-facing `## 000030_agent-actionable-errors`
  section to the repo-root `CHANGELOG.md` (above `## 000029_readme-refresh`), written for a reader
  who never saw the plan — no file paths or package names, behavior-focused (single reliable
  response channel, actionable failure text naming position/resource/recovery command, no more
  silent over-advancement, no double-printing).
- Next: advance to `test_plan`.
- `test_plan` step complete: wrote `.spektacular/plans/000030_agent-actionable-errors/test-plan.md`
  covering both success metrics' manual halves (per plan.md's Testing Approach § Success metrics):
  Metric 1 (agent can act correctly from error text alone) grounded in real captured JSON from this
  session's own live smoke tests across all 3 error categories (invalid_transition,
  workflow_in_progress/cross_kind, not_found); Metric 2 (incident class doesn't recur) as an
  ongoing production-observation procedure for future incident triage. Committed via `plan file
  write`, confirmed by reading back.
- Next: advance to `finished` — this is the last step.

## WORKFLOW COMPLETE
- `implement` workflow reached its terminal `finished` state. All 7 phases across 3 milestones
  implemented, tested, and verified. Final independent check before reporting to the user:
  `go build ./...`, `go vet ./...`, `go test ./...` (full repo) all green; `git diff --stat` shows
  30 modified files + 3 new test files (`cmd/error_response_test.go`, `cmd/root_test.go`,
  `internal/output/writer_test.go`), 1077 insertions / 353 deletions, no unexpected files touched
  (the `templates/steps/spec/*.md`/`templates/scaffold/spec.md` diffs predate this session — visible
  in the original git status at conversation start, not something this implement run touched).
- Nothing left to do for this plan. If resumed, there is no further work — re-running `implement
  new` for this plan name would find no unchecked phases.
