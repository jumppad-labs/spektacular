# Feature: 000031_agent-session-debug-log

## Overview

Spektacular's workflows are carried out entirely by an AI agent, with no person watching along the way. When a session goes wrong — an agent stalls, repeats itself, or takes a wrong turn — there is currently no record of what actually happened, so understanding it after the fact means guessing from whatever fragments remain in the agent's own memory of the conversation.

This feature introduces a record of what took place during a session, built and worded specifically so that another agent can read it back and reconstruct events efficiently — not primarily so a person can read it. Whoever or whatever needs to understand a past session afterward — most often another agent brought in to diagnose a problem — can reconstruct what happened without needing a person to have been watching, or to manually piece the story together from scattered output.

## Requirements

- [ ] **Opt-in toggle**
  An operator can turn this recording on or off; it is off by default and only captures a session when explicitly enabled.

- [ ] **Captures every command and response**
  While enabled, the system captures every command the agent issues to the tool and the tool's full response, including responses that reject the command, produce an error, or otherwise make no forward progress.

- [ ] **Sequence reconstruction**
  The captured record allows the sequence and order of events within a session to be reconstructed after the session has ended.

- [ ] **Self-contained record**
  The record is self-contained: understanding what happened in a past session from the record does not depend on also having the original agent's conversation or memory of that session.

- [ ] **Agent-first content**
  The record's content and structure are organized primarily for another agent to consume and reconstruct events from, not primarily for a person to read directly.

- [ ] **Non-invasive**
  Turning this recording on does not change the tool's behavior or outcome for the agent carrying out the workflow — only enables the record.

- [ ] **Session isolation**
  A session's record is distinguishable from the record of any other session, so that reconstructing one session does not require untangling it from others.

## Constraints

- The session record must never be committed to the project's version control. It must be treated as local, disposable working state, not shared or checked in by default.
- The solution must work within the existing single-binary command-line tool. It must not introduce a new external service, background process, or additional runtime dependency to produce, store, or transmit the record.
- The record must not be a substitute for the tool's own error messages being self-sufficient — an agent must still be able to resolve an ordinary error from the response it received at the time, without needing to consult this record. (This mirrors the corresponding constraint already established for the tool's error messages in a separate feature.)

## Acceptance Criteria

- [ ] **Record presence follows the toggle**
  When the recording has not been enabled, running a workflow session produces no session record. When an operator has enabled it, running a workflow session produces a record for that session.

- [ ] **Non-progress events included**
  For a session that includes at least one command whose response was a rejection, an error, or made no forward progress, the resulting record contains an entry for every command sent during the session and the response received for each — including those non-progress ones — not only the commands that succeeded.

- [ ] **Unambiguous ordering**
  Given the record of a completed session, the order in which its captured events occurred can be determined without ambiguity.

- [ ] **Reconstructable without the agent's memory**
  A reader given only the session's record — without the original agent's conversation — can determine what took place during that session.

- [ ] **Machine-extractable detail**
  An automated reader can extract the details of each captured event directly, without needing to interpret free-form narrative prose to figure out what happened.

- [ ] **No behavioral side effects**
  Running the same workflow session with recording enabled produces the same responses to the agent as running it with recording disabled.

- [ ] **Sessions not intermingled**
  Given the records of two different sessions, each can be identified as belonging to its own session and is not intermingled with the other's events.

## Technical Approach

- Consider one discrete record per captured event as a structuring choice.
- Consider timestamping each captured event, to support reconstructing precise timing and ordering, not just relative sequence.
- Prefer reusing any existing diagnostic/debug toggle mechanism already available to operators, rather than introducing a new, separate way to turn this on.
- Consider capturing the outcome of each command — not just that a command was issued and a response returned, but what state (if any) actually changed as a result — so a diagnosing agent can tell a real advance apart from a rejected or no-op one.
- Beyond this direction, no further technical design has been decided; the detailed design is left for the plan workflow to propose.

## Success Metrics

- Given the record from a session where an agent got stuck or behaved unexpectedly, another agent can reconstruct what happened and identify the point things went wrong without needing a person to explain the session or without access to the original agent's own memory of it.
- Enabling this recording does not become something operators avoid using because of its cost or side effects — it is left on for as long as an investigation needs it without concern.

## Non-Goals

- Automatic detection or diagnosis of loops or problems is out of scope; this feature only produces the record, a diagnosing agent performs the analysis.
- A viewer, UI, or dashboard for browsing records is out of scope; the raw record is read directly by whoever or whatever consumes it.
- Automatic redaction or scrubbing of sensitive content from the record is out of scope; keeping the record out of version control is separate from sanitizing what it contains.
- A retention, rotation, or cleanup policy for old records is out of scope.
- Using the record to automatically correct or heal an agent's behavior is out of scope.
- Capturing the agent's own internal reasoning or thought process is out of scope; only the tool's own observable side of the interaction — the commands it receives and the responses it returns — can be captured at all.
