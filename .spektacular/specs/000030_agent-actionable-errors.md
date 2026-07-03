# Feature: 000030_agent-actionable-errors

## Overview

Spektacular's workflows are driven entirely by an AI agent that reads each command's output and decides what to do next. When something goes wrong — a bad input, a step run out of order, an internal failure — the agent has nothing else to go on: no human is watching the terminal to interpret a cryptic message or notice a usage screen. Today, error output varies unpredictably in shape and clarity across the tool, so agents sometimes cannot tell what failed or what to do about it. In one observed incident, an agent hit a rejected step transition with no indication of the correct next step, and burned significant time searching for it by trial and error before recovering — work a human wouldn't have needed to do, since the answer was available, just not communicated.

This feature makes every error the tool can produce meaningful and self-explanatory to an agent: consistently delivered, clearly worded, and carrying enough diagnostic detail (what failed, why, and what a correct next action looks like) that the agent can resolve the problem itself rather than guessing, retrying blindly, or stalling. The people who benefit are everyone relying on agent-driven workflows staying productive and un-supervised — the agents don't need someone to translate a confusing failure for them, and the developers using this tool spend less time rescuing stuck sessions.

## Requirements

- [ ] **Consistent delivery format**
  The system must deliver every error to the calling agent in a single, consistent format, regardless of which part of the tool the failure originates in.

- [ ] **Reliable error signal**
  An agent must be able to reliably distinguish an error response from a successful response using the same signal every time.

- [ ] **Specific failure description**
  Every error message must state what specifically failed, not just that something failed.

- [ ] **State-aware rejection**
  When an action is rejected because of the caller's current state (for example, an out-of-sequence or invalid request given where the caller is in a multi-step process), the error must identify that state and what actions are currently valid.

- [ ] **Named resources**
  When an error message references a specific resource (a file, an entry, a named step, an identifier), it must name that resource specifically rather than referring to it generically.

- [ ] **No silent failures**
  No failure may be silently discarded. Every operation that does not fully succeed must produce a visible error to the caller, even failures that are unexpected or internal to the tool.

- [ ] **Self-sufficient messages**
  An error message must not require the agent to consult external documentation or the tool's source code to understand what went wrong or what to try next.

- [ ] **No duplicate presentation**
  The system must not present the same error more than once for a single failure.

- [ ] **No extraneous noise**
  The system must not present extraneous information alongside an error unless that information directly helps resolve the specific failure.

- [ ] **Deterministic errors**
  Given the same invalid request in the same state, the system must produce the same error every time.

## Constraints

- The error response shape is not required to stay backward-compatible with today's shape — it may be redesigned (new fields, restructured content) as long as the result is applied consistently everywhere.
- Success and error responses must be delivered to the calling agent through the same output channel. The solution must not rely on the caller watching two different channels to reliably see every response.
- An error message must be self-sufficient at the point it is delivered. The solution must not satisfy the "no docs/source diving" requirement by directing the agent to a separate log, transcript, or file it must go read — the diagnostic information belongs in the error response itself.
- The solution must work within the existing single-binary command-line tool. It must not introduce a new external service, background process, or additional runtime dependency to deliver or store error information.

## Acceptance Criteria

- [ ] **Uniform error structure**
  Every error response returned by the tool, regardless of which command or internal part of the tool produced it, has the same structure, so a caller can parse any error response the same way without special-casing.

- [ ] **Fixed success/error signal**
  A caller can determine whether a given command's response is an error or a success using one fixed, always-present signal — every response the tool produces carries that signal correctly.

- [ ] **Specific failure text**
  For any given failure, the error text names the specific condition that caused it (for example, which value or which check failed) rather than a generic phrase that could describe many different failures.

- [ ] **State and valid actions named**
  When a request is rejected because of the caller's current position in a multi-step process, the error names that current position and states which action(s) would currently succeed.

- [ ] **Resource named explicitly**
  When an error concerns a specific file, entry, step, or identifier, the error text includes that specific name or identifier rather than referring to it generically ("the file", "the step").

- [ ] **No silent completion**
  Every operation that does not complete successfully — including unexpected internal failures — results in a visible error returned to the caller; none complete silently as if they had succeeded.

- [ ] **Explicit next action**
  Each error message includes an explicit statement of what the caller should do next (a corrective action, a valid input example, or a valid next step), not only a description of what went wrong.

- [ ] **Single presentation**
  For a single failure, the tool outputs the text describing it exactly once, not repeated multiple times in the same response.

- [ ] **No extraneous content**
  An error response contains only information relevant to that error — no unrelated instructional or help text is appended alongside it.

- [ ] **Deterministic repeat**
  Issuing the same invalid request against the same state twice in a row produces identical error text both times.

## Technical Approach

- Prefer a single, central point through which every error passes on its way to the caller, rather than fixing formatting at each place an error can occur — so consistency is structural rather than something every new command has to remember to do correctly.
- Where an error originates from the tool's own internal machinery (for example, rejecting a request that is out of sequence), prefer rewriting it with caller-relevant context rather than passing that internal component's own wording through unchanged.
- Beyond this direction, no further technical design has been decided; the detailed design is left for the plan workflow to propose.

## Success Metrics

- An agent that encounters any error from the tool can determine and take a correct next action directly from that error's text, without a human interpreting it and without probing multiple other commands to figure out what happened.
- No agent session gets stuck retrying or looping in place because an error it received didn't explain what to do next — the kind of incident that motivated this work does not recur.

## Non-Goals

- A debug or session action-log for reconstructing agent behavior after the fact is out of scope; it is left for a separate, future spec.
- Changing the workflow's step sequencing, ordering, or naming is out of scope; this feature changes what is communicated when something goes wrong, not the structure of the steps themselves.
- Validating, correcting, or otherwise constraining agent behavior (for example, preventing an agent from guessing an invalid step name) is out of scope; this feature makes errors recoverable, it does not police the caller.
- Error-rate telemetry, monitoring, or alerting is out of scope; it is distinct from the qualitative success metrics already captured for this feature.
