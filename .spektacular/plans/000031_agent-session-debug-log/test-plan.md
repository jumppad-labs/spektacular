# Test Plan: 000031_agent-session-debug-log

Two of the spec's success metrics were classified in plan.md's Testing Approach as manual —
not mechanically assertable by an automated test — and are covered here. The "no side effects"
half of the second metric is already covered by an automated test
(`TestSessionLog_EnabledDoesNotChangeCallerVisibleOutput` in `cmd/root_test.go`) and is not
repeated below.

## 1. Reconstruction: a diagnosing agent can identify the stuck point from the record alone

**What to measure**: Given only the recorded session log from a session where an agent stalled or
made no progress, a different agent (with no access to the original session's context or memory)
can correctly name the point things went wrong.

**How**:
1. In a scratch project, set `debug.enabled: true` in `.spektacular/config.yaml`.
2. Reproduce the documented `goto-to-current-step-is-a-silent-noop` gotcha live: run
   `spektacular spec new --data '{"name":"billing"}'` (lands on the `overview` step), then run
   `spektacular spec goto --data '{"step":"overview"}'` (the already-current step — returns exit
   code 0, no error, and looks like a normal response).
3. Locate the resulting record at `.spektacular/debug/session-log.jsonl` in that scratch project.
4. Start a **fresh** agent session with no other context. Hand it only the contents of
   `session-log.jsonl` (not this test plan, not the plan documents, not any explanation of what
   happened) and ask: "This is a record of a session where an agent may have gotten stuck. Where,
   if anywhere, did it stop making progress?"

**Expected result**: The fresh agent identifies the second entry (`seq: 1`, the `spec goto` call)
as the point of no progress — specifically citing that `"advanced": false` despite `"exit_code": 0`
and no error in `"response"`, and that `"state_before"` and `"state_after"` are identical — rather
than concluding the session succeeded throughout, or being unable to answer at all.

**Who / when**: A maintainer, before each release that touches `cmd/root.go`,
`internal/sessionlog`, or the workflow FSM's step-transition logic — since a regression here (the
record losing its diagnostic value) would not be caught by any automated test in this plan.

## 2. Cost / ongoing adoption: recording is not something operators disable due to overhead

**What to measure**: Whether enabling `debug.enabled` in a real, ongoingly-used project becomes
something an operator turns back off, or avoids turning on, because of perceived cost (disk usage,
performance, clutter) rather than because the investigation it was turned on for is over.

**How**: This project's own `.spektacular/config.yaml` already ships with `debug.enabled: true`,
so real dogfooding usage against the actual implement/plan/spec workflows this repo runs on itself
is the observation vehicle — no separate test harness needed. Over a defined observation window
(recommended: 4 weeks of normal repo usage following this plan's merge), a maintainer checks:
- Was `debug.enabled` ever turned off in `.spektacular/config.yaml`, and if so, why (check git
  history / commit messages for that line)?
- Does `.spektacular/debug/session-log.jsonl`'s size (`ls -la .spektacular/debug/session-log.jsonl`)
  remain a size nobody has complained about or needed to manually clear?
- Any note, issue, or comment (in this repo or its issue tracker) citing the recording as a source
  of slowdown or annoyance?

**Expected result**: `debug.enabled` remains `true` throughout the window, no manual intervention
on the log file was needed, and no complaint surfaced. If any of these fail, that's a signal the
"no retention/rotation policy" Out-of-Scope decision (plan.md) may need revisiting in a future
plan — not evidence this plan's implementation is incorrect.

**Who / when**: A maintainer, ~4 weeks after this plan merges, as a lightweight retrospective
check — not a release gate.
