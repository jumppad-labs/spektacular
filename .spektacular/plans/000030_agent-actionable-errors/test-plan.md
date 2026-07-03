# Test Plan: 000030_agent-actionable-errors

Two of the spec's success metrics have a qualitative half that an automated test cannot certify
(see plan.md's Testing Approach § Success metrics). Both are captured here as manual procedures.
The mechanical/structural half of each is already covered by automated tests written during
implementation (`cmd/root_test.go`'s `TestWrapper_*` family, `internal/workflow/workflow_test.go`'s
FSM-translation tests, `cmd/error_response_test.go`'s end-to-end incident regression, and the
resource-naming tests in `internal/store/store_test.go`/`cmd/file_test.go`).

## Metric 1: An agent can determine and take a correct next action directly from an error's text

**What to measure**: Whether a fresh agent session, shown only a captured error response with no
other context, can correctly propose the right next command — the qualitative judgment a
unit/integration test can't make (it can only confirm the fields are non-empty, not that their
content is sufficient).

**How**: Take a real captured error response from this implementation's own verification passes —
for example, this exact JSON, captured live during Phase 2.1/3.2 verification by running
`spektacular spec new --data '{"name":"billing"}'` followed by
`spektacular spec goto --data '{"step":"milestones"}'` in a fresh project directory:

```json
{
  "error": true,
  "code": "invalid_transition",
  "message": "cannot run step \"milestones\" from the current step \"overview\"",
  "state": {
    "current": "overview",
    "valid_actions": ["requirements"]
  },
  "next_action": "run: spektacular spec goto --data '{\"step\":\"requirements\"}'"
}
```

Open a brand-new agent session (no prior conversation history, no access to this repo's
`.spektacular/context.md` or plan documents). Paste only the JSON above and the single-sentence
prompt: "This is the response you got from running a command. What do you do next?" Do not give
any additional hints.

Repeat with at least one response from each of the other error categories this plan produces —
a `workflow_in_progress`/`cross_kind_workflow_in_progress` resume response (Phase 2.3) and a
`not_found` resource-naming response (Phase 2.2), both of which can be captured live the same way
(`spec new` twice in the same directory; `spec file read` on a missing file).

**Expected result**: In each case, the fresh agent session proposes running the exact command named
in `next_action` (or a command consistent with `state.valid_actions`/`resource`) as its next step,
without asking a clarifying question first and without proposing to retry the failed command
unchanged.

**Who / when**: A maintainer or the implementing agent, once per release that touches error-handling
behavior in `cmd/root.go`, `internal/workflow`, `internal/output`, or `cmd/resume.go`. Not required
for unrelated changes.

## Metric 2: No agent session gets stuck retrying or looping because an error didn't explain what to do next

**What to measure**: Whether the specific incident class that motivated this plan — an agent
hitting a rejected step transition and getting stuck with no way forward — actually stops recurring
in live use. This is an ongoing production observation, not something CI can certify: the automated
regression test (`cmd/error_response_test.go`'s `TestGoto_UnreachableStepReturnsActionableError`)
gives the direct mechanical guarantee (the response's `NextAction`/`State` would have given the
agent the correct next command), but only live usage over time confirms the incident class doesn't
resurface in a form this plan didn't anticipate.

**How**: When reviewing future agent-session incident reports or bug reports involving this CLI,
check whether the reported problem is "an agent got stuck because an error response didn't explain
what to do next." If one surfaces, capture the exact response JSON the agent received (same
capture method as Metric 1) and add it as a new case to Metric 1's manual spot-check the next time
this test plan is run, so the corpus of real-world cases grows over time.

**Expected result**: Zero incident reports of this specific class after this plan ships. Any that do
surface should be traceable to either (a) an error category outside this plan's scope (see plan.md
§ Out of Scope) or (b) a genuine gap, which should be filed as a new spec against this repo's own
spec/plan/implement workflow.

**Who / when**: Whoever triages agent-session incident reports for this repo, on an ongoing basis —
not a one-time pre-release check like Metric 1.
