# Research: 000030_agent-actionable-errors

## Alternatives considered and rejected

- **Fix formatting at each error call site individually** (no central point). Rejected: the
  spec's own Technical Approach explicitly prefers a single central point ("so consistency is
  structural rather than something every new command has to remember to do correctly"), and the
  codebase already demonstrates the failure mode of the per-site approach — `output.WriteError`
  is opt-in and only actually used at <50% of error-return sites across `cmd/spec.go`,
  `cmd/plan.go`, `cmd/implement.go` (consistent only in `cmd/knowledge.go`). Adding one more
  call site convention would not close that gap; it would just add a fourth divergent shape.
- **Keep errors on stderr, put a "detect JSON on either stream" burden on the agent.** Rejected
  outright by the spec's own constraint ("must not rely on the caller watching two different
  channels"). Confirmed live that the current implementation violates this today (success →
  stdout via `output.Write`/`WriteResult`, error → stderr via `output.WriteError` or the
  `cmd/root.go:25` fallback) — this is the change, not a considered-and-rejected alternative,
  but recorded here because it's tempting to treat as "good enough since it's already JSON
  half the time."
- **Rely on process exit code as the success/error signal instead of (or in addition to) a
  response-body field.** Rejected as the *sole* signal: confirmed live that today a rejected FSM
  transition written via `output.WriteError` still returns exit code 0 (`WriteError` returns nil
  on a successful *write*, so `RunE` sees no error) — exit code and body currently disagree. A
  design that also treats exit code as authoritative without fixing this would leave that
  divergence in place. Exit code can still be *part* of the design as a secondary, consistent
  signal, but the primary "reliable error signal" (Requirement 2 / AC 2) needs a body field that
  is set correctly on every path, independent of exit-code plumbing.
- **Reuse `ResumeReport` (`cmd/resume.go:20-27`) unchanged as the general error shape.**
  Rejected as-is because it is a *success*-shaped response (written via `WriteResult`, no
  error/success discriminant of its own) scoped to one case (in-progress/cross-kind workflow
  detection). It is the strongest *precedent* for "name current state + valid next actions" — its
  `Kind`/`CurrentStep`/`Instruction` shape and its template-rendered instruction
  (`templates/steps/resume.md`, `templates/steps/resume_mismatch.md`) are worth generalizing —
  but the plan needs a shape that is unambiguously an *error* (Requirement 2), which
  `ResumeReport` deliberately is not. (Resolution reached during the architecture step:
  `ResumeReport` is folded into the new `ErrorResponse` shape rather than kept separate — see
  plan.md § Architecture & Design Decisions.)
- **Silence cobra's default error/usage printing (`SilenceErrors`/`SilenceUsage`) without adding
  a JSON envelope, i.e. just quiet cobra down and let plain-text errors through cleanly.**
  Rejected: would fix "no duplicate presentation" / "no extraneous noise" but not "consistent
  delivery format," "reliable error signal," or "specific failure description" in a
  machine-parseable way. Silencing cobra is necessary but not sufficient — it needs to be paired
  with an actual envelope. (This became Architecture Option 2, "minimal patch," rejected for the
  same reason plus the risk of reproducing today's opt-in-call-site problem.)
- **Architecture Option 3 — full nested envelope** (`{"ok": bool, "data": {...}, "error":
  {...}}`, aligned with spec `000005_adhoc_json_protocol`'s vocabulary). Rejected: highest blast
  radius of the three options — every skill template that reads `.step`/`.instruction` at the top
  level would need updating to read through a `.data` wrapper, and it changes the shape of every
  success response, not just errors, for no additional requirement coverage over the chosen
  option. `000005`'s protocol is scoped to a distinct, apparently-unimplemented `adhoc`
  subcommand and remains untouched.

## Chosen approach — evidence

- **A single `RunE` wrapper / centralized `Execute()` path is feasible and is the natural
  choke point.** Evidence: `cmd/root.go:23-28` (`Execute()`) is already the one place every
  command's return value passes through; no `PersistentPreRunE`/`PersistentPostRunE` hook is
  used anywhere yet (`cmd/root.go` — confirmed via grep, zero hits), so there is no existing
  behavior to preserve there. `cobra.Command.SilenceErrors`/`SilenceUsage` are never set anywhere
  in `cmd/*.go` (grep, zero hits) — cobra's own default double-printing is live today and would
  need to be silenced as part of this fix.
- **A single error-envelope type belongs in (or replaces) `internal/output`.** Evidence:
  `internal/output/writer.go:29-31` (`WriteError`) is already the closest thing to a shared error
  path, but its shape (`{"error": string}`) has no code/state/next-action fields and no
  success/error discriminant field shared with `WriteResult`'s output. `internal/output` has no
  test file today (confirmed — no `internal/output/*_test.go` exists), so there is no existing
  contract locking in the current shape.
- **`workflow.Goto`/`Next` need a translation layer for `looplab/fsm` errors, not just
  pass-through.** Evidence: `internal/workflow/workflow.go:158,177` both do a bare
  `return err` on `w.FSM.Event(...)` failure. The raw wording, confirmed live
  (`{"error":"event milestones inappropriate in current state overview"}`), comes from
  `looplab/fsm@v1.0.3/errors.go:28-30` (`InvalidEventError.Error()`) and names the current state
  but never the valid next actions. The workflow already has the data needed to fix this without
  new state: `w.FSM.AvailableTransitions()` (used internally at `workflow.go:153`) and
  `w.StepNames()` (`workflow.go:270-276`) are both ready-made inputs for "current state + valid
  next actions" (Requirement 4 / AC 4).
- **`cmd/resume.go`'s `ResumeReport` + `stepkit.RenderTemplate` pattern is the model to mirror**
  for state-aware messages, not reinvent. Evidence: `cmd/resume.go:38-62` already renders
  caller-facing "here's your state, here's what to do next" copy from a template
  (`templates/steps/resume.md`, `templates/steps/resume_mismatch.md`) rather than hand-formatting
  strings inline — this is the one existing example of the exact shape Requirement 4 wants,
  just scoped to one case (in-progress-workflow detection) instead of all rejected transitions.
- **A pre-existing, unrelated correctness bug sits directly in the "no silent failures" path and
  needed an explicit scope decision (user-confirmed in scope; see plan.md).** —
  `internal/workflow/workflow.go:110-119,134-143` plus
  `looplab/fsm@v1.0.3/fsm.go:349-364` (verified by reading the library source): the FSM's
  `enter_state` callback (which persists `state.json` via `saveState`) runs *before* the named
  `after_<step>` callback (where `step.Callback` executes and can fail). By the time a failing
  step callback calls `e.Cancel(err)`, `f.current` has already been set to the destination state
  and `enter_state` has already written that new `CurrentStep` to disk — `Cancel` at that point
  only sets the returned error, it does not revert the persisted state (its doc comment in
  `event.go:47-48` says it's meant to be called from `before_<EVENT>` or `leave_<STATE>`, both of
  which run *before* the transition commits — `after_<event>` is documented as running after).
  Net effect: the immediate caller does see an error (so this is not "invisible" to that one
  request), but the on-disk state has already advanced past the step that reported failure, so a
  *subsequent* command (a retry, a `status`, a resume) will silently treat the failed step as
  completed. No existing test exercises this. **Fix adopted:** move the domain step-callback
  registration from `after_<event>` to `before_<event>` (see plan.md § Implementation Detail,
  Phase 3.1) — `beforeEventCallbacks` runs before the FSM commits the transition or invokes
  `enter_state`, so `Cancel()` there genuinely prevents both.
- A durable knowledge-base gotcha was written documenting the general `looplab/fsm` `Cancel()`
  semantics (`before_`/`leave_` vs. `after_`), independent of this specific bug's fix, for any
  future `internal/workflow` change:
  `.spektacular/knowledge/gotchas/fsm-cancel-only-works-before-transition-commits.md`.

## Files examined

- `internal/output/writer.go:1-88` — the only existing output/error-writing code; `Write`,
  `WriteResult`, `WriteError`; no shared success/error envelope; `WriteError` writes to whatever
  `io.Writer` it's given (stderr, by every call site).
- `cmd/root.go:23-28` — sole true central point today; plain-text fallback, no JSON, no
  `SilenceErrors`/`SilenceUsage`.
- `cmd/spec.go`, `cmd/plan.go`, `cmd/implement.go` — near-triplicated command handlers; each
  independently and inconsistently decides whether a given error goes through
  `output.WriteError` or returns raw.
- `cmd/knowledge.go` (15 sites, e.g. lines 161–289) — the one command family that consistently
  routes every error through `output.WriteError`; still stderr-only.
- `cmd/storefile.go:65-77` — `file read`'s `RunE` returns `st.Read(...)`'s raw error
  unrewritten (would be `store.ErrNotFound` = `"not found"`, no filename) directly to cobra;
  success path writes raw file bytes to stdout (not JSON) — a real edge case the envelope design
  must account for (not every success response is a JSON object today).
- `cmd/resume.go:1-182` — `ResumeReport`, `resumeInstruction`/`mismatchInstruction`,
  `resumeOrClear`, `guardKind` — the existing state-aware-messaging precedent to generalize.
- `internal/workflow/workflow.go:26-36` (`ResultWriter`, `StepCallback`), `:68-147` (`New`,
  callback wiring: `after_<step>` at 110-119, `enter_state` at 134-143), `:152-185` (`Next`,
  `Goto`, both bare `return err` on FSM failure), `:187-212` (`renderStep` — the same-step
  silent-no-op path documented in the gotchas entry below).
- `looplab/fsm@v1.0.3/errors.go:15-119` — all seven error types the FSM can return and their
  exact `.Error()` wording; `fsm.go:296-403` (`Event`) — confirmed transition/callback ordering
  (`f.current = dst` and `enterStateCallbacks` run before `afterEventCallbacks`); `event.go:47-56`
  (`Cancel`) — confirmed its doc-commented intended use is `before_`/`leave_`, not `after_`.
- `internal/store/store.go:11-12` (`ErrNotFound = errors.New("not found")` — generic, no
  resource name at the point it's raised), `:84` (`"path escapes store root"` — no path named),
  `:89-141` (`Read`/`Write`/`Delete`/`List` — raw stdlib `os.*` errors pass through unwrapped in
  places).
- `internal/knowledge/set.go` — the best-quality existing error text in the codebase (names
  scope/path/provider consistently, e.g. `:82,91,111,191,243,248,273`) — a good tone/content
  model for "specific failure description" even though it isn't enveloped or state-aware.
- `cmd/knowledge_test.go:342-371` — the only test in the suite that locks in an exact JSON error
  shape (`{"error":...}` on stderr) — will need updating under this redesign; every other error
  assertion in the suite (`cmd/spec_test.go`, `cmd/cross_kind_test.go`, `cmd/resume_test.go`,
  `internal/workflow/workflow_test.go`) only checks `require.Error`/substring, so wording is free
  to change.
- `templates/steps/resume.md`, `templates/steps/resume_mismatch.md` — the concrete existing copy
  pattern ("state, then two concrete next commands, one per outcome") to mirror in tone for the
  new generalized rejected-transition message.
- `.spektacular/knowledge/gotchas/goto-to-current-step-is-a-silent-noop.md` — documents that
  `Workflow.Goto(name)` where `name == Current()` (`workflow.go:172-174`, `renderStep`) re-emits
  the current step's instruction with no error and no state change. Considered against
  Requirement 6 during planning and deliberately left **out of scope** (see plan.md § Out of
  Scope): it's a working, intentional resume-flow mechanism, not a failure.
- `.spektacular/knowledge/architecture/cli-design-for-ai-agents.md` — pre-existing design ethos
  doc (predictability, exhaustive validation, schema introspection) this feature extends into
  error handling; worth aligning tone/terminology with.
- `.claude/skills/spek-plan/SKILL.md:47`, `.claude/skills/spek-new/SKILL.md:47`,
  `.claude/skills/spek-implement/SKILL.md:42` — document the old `ResumeReport` shape agents are
  told to expect; found during the open-questions due-diligence pass (not during initial
  discovery) and folded into Phase 2.3's scope.

## External references

- `github.com/looplab/fsm` v1.0.3 (`go.mod`) — the FSM library backing
  `internal/workflow`. Its `errors.go` and `fsm.go` are the authoritative source for exact
  error wording and callback-ordering semantics any central rewriter must intercept/account for.
  Not vendored in-repo; read directly from `$GOMODCACHE` (`go list -m -f '{{.Dir}}'
  github.com/looplab/fsm`).
- `github.com/spf13/cobra` v1.8.1 (`go.mod`) — confirmed default behavior (via live run): on a
  `RunE` error with `SilenceErrors`/`SilenceUsage` unset, cobra prints `Error: <msg>` plus full
  usage/help text to stderr *before* `cmd/root.go:25` prints the same message again — the
  concrete mechanism behind the "no duplicate presentation" / "no extraneous noise" violations.

## Prior plans / specs consulted

- `.spektacular/specs/000030_agent-actionable-errors.md` — this feature's own spec (already read
  in full in the overview step); Technical Approach explicitly directs a single central point and
  rewriting internal-machinery errors with caller-relevant context.
- `.spektacular/specs/000005_adhoc_json_protocol.md` — prior art for a strict, single-envelope,
  exactly-one-terminal-event protocol (a distinct `adhoc` subcommand's JSONL frames, stdout-only,
  stderr diagnostics-only). No `adhoc` command exists in `cmd/` today, so this was likely never
  implemented, but its design (one envelope, one channel, deterministic terminal state) is
  directly analogous to what 000030 wants for ordinary command responses generally. Reviewed
  during the architecture step as Option 3 and not chosen — see "Alternatives considered and
  rejected" above.
- `.spektacular/specs/000024_resume.md` — the spec that produced `cmd/resume.go`'s
  `ResumeReport`; already required "current state + valid next actions" for the one in-progress-
  workflow-detection case. Confirms that pattern was a deliberate, previously-reviewed design
  choice, not incidental.
- `.spektacular/specs/000021_file-writing.md` — minor precedent only: its acceptance criteria
  reference naming "the offending path and the reason" on a write error; consistent with, but not
  as strong as, the new spec's "Named resources" requirement.
- No `.spektacular/plans/` document exists yet for 000030 (`plan file list` tops out at
  `000029_readme-refresh`) — this is genuinely greenfield; there is no prior plan attempt to
  reconcile with.
- `.spektacular/specs/000016_plan_format.md`, `000023_context-over-memory.md`,
  `000025_convention-aware-planning.md` — checked, not relevant (plan template structure,
  agent-memory redirection, convention-aware plan generation respectively).

## Open assumptions

None remaining. The one assumption carried out of discovery — whether the FSM
state-persists-before-callback-can-cancel bug is in scope — was resolved by explicit user
confirmation during the architecture step (in scope; see plan.md § Architecture & Design
Decisions and § Out of Scope for the related, deliberately-excluded goto-same-step case). No
further assumptions were introduced during architecture, components, data structures,
implementation detail, dependencies, testing approach, milestones, phases, or open-questions —
each of those steps' decisions was either derived directly from code/spec or confirmed with the
user in turn, per the plan's Open Questions section.

## Rehydration cues

- Re-run `go run . knowledge always-applied` then `go run . knowledge search error` /
  `... search "json output"` / `... search "state machine"` / `... search cli` to reload the
  same knowledge-base hits used here (conventions/README.md, architecture/workflow-steps.md,
  architecture/bob-output-spec.md, architecture/claude-output-spec.md,
  architecture/cli-design-for-ai-agents.md, gotchas/goto-to-current-step-is-a-silent-noop.md,
  gotchas/fsm-cancel-only-works-before-transition-commits.md,
  learnings/step-advancement-loops-are-not-a-per-agent-wrapper-bug.md).
- Re-read `internal/output/writer.go`, `internal/workflow/workflow.go`, `cmd/root.go`,
  `cmd/resume.go`, `cmd/storefile.go`, `cmd/knowledge.go` in full — these six files are the
  entire surface the architecture-level design touches.
- To re-verify the live behavior claims (dual-channel output, exit-code-0-on-reported-error,
  duplicate cobra printing, raw FSM wording): `go build -o /tmp/spek .` then from a temp dir with
  `.spektacular/state.json` seeded to an in-progress workflow, run `spek plan goto --data
  '{"step":"<unreachable-step>"}'` (an unreachable step name triggers `InvalidEventError`)
  capturing stdout/stderr/exit-code separately, and separately run any subcommand with `--data`
  omitted.
- `go list -m -f '{{.Dir}}' github.com/looplab/fsm` to relocate `errors.go`/`fsm.go`/`event.go`
  for the exact callback-ordering/error-wording evidence cited above.
- `.spektacular/specs/000030_agent-actionable-errors.md` (the spec, read in full already) and
  `.spektacular/specs/000005_adhoc_json_protocol.md` (envelope-design precedent, rejected as
  Option 3) are the two specs worth re-reading in full if resuming cold.
- The full plan is staged across
  `.spektacular/work/000030_agent-actionable-errors/*.md` (one file per plan section) until
  written to the store; `.spektacular/context.md` holds the turn-by-turn decision log for this
  planning session.
