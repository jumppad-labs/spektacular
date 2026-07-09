# Test Plan: 000032_spec-workflow-pair-programming-enhancements

## Manual Verification Procedures

All three success metrics from the spec require manual verification through live agent sessions and production usage monitoring.

### Metric 1: Natural Timing of Spec Offers

**What to measure**: Whether substantial discussions consistently get offered a spec at a point that feels natural rather than premature or too late.

**How**:
1. Initialize a test project: `spektacular init claude` (or `bob`, `codex`)
2. Start a coding session with the agent
3. Engage in discussions of varying substantiality:
   - **Trivial**: "Add a print statement to main.go"
   - **Small**: "Add error handling to the login function"
   - **Moderate**: "Add user authentication with JWT tokens, password hashing, and session management"
   - **Substantial**: "Design and implement a complete REST API with authentication, rate limiting, caching, and monitoring"
4. Record at what point (if any) the agent offers to create a spec for each discussion type
5. Repeat with different threshold settings (`strict`, `moderate`, `lenient`) in `.spektacular/config.yaml`

**Expected result**: 
- Trivial and small discussions: no spec offer
- Moderate discussions with `moderate` threshold: spec offer after 2-3 requirements mentioned
- Substantial discussions: spec offer consistently appears before implementation begins
- Timing feels natural to the tester (not too early, not too late)

**Who / when**: Product team member during pre-release testing against a staging environment with each supported agent (claude, bob, codex).

### Metric 2: User Surprise and Annoyance

**What to measure**: Whether users rarely feel surprised by the offer behavior — neither annoyed by over-triggering on trivial work, nor missing an offer they expected on substantial work.

**How**:
1. Recruit 3-5 developers unfamiliar with the feature
2. Have each complete 5 tasks of varying complexity in a Spektacular-initialized project:
   - 2 trivial tasks (single-line changes)
   - 2 moderate tasks (multi-file changes with 2-3 requirements)
   - 1 substantial task (feature with 5+ requirements)
3. After each task, ask: "Did the spec offer behavior surprise you? (yes/no)" and "If yes, was it annoying or helpful?"
4. Record responses and calculate surprise rate

**Expected result**:
- Surprise rate < 20% across all tasks
- When surprised, majority (>70%) report it as helpful rather than annoying
- No reports of missing expected offers on substantial tasks
- No reports of annoying over-triggering on trivial tasks

**Who / when**: UX researcher or product manager during beta testing with external users.

### Metric 3: Default Threshold Calibration

**What to measure**: Whether users rarely need to adjust the default threshold, indicating the out-of-the-box default (`moderate`) is well-calibrated.

**How**:
1. Instrument the `spektacular init` command to log (with user consent) when a project's `.spektacular/config.yaml` contains a non-default `spec_trigger_threshold` value
2. Deploy instrumented version to production
3. After 30 days, query logs for:
   - Total projects initialized
   - Projects with modified `spec_trigger_threshold`
   - Calculate adjustment rate: (modified / total) * 100

**Expected result**:
- Adjustment rate < 10% (fewer than 1 in 10 users change the default)
- If adjustment rate > 10%, analyze which direction users adjust (stricter or more lenient) to inform default recalibration

**Who / when**: Engineering team monitoring production telemetry 30 days post-release.

## Notes

Phase 2.1 only implements the context-persistence mechanism (clearing and instructing to write to context.md). The spec-trigger threshold configuration and AGENTS.md instruction delivery are deferred to future phases (Milestone 1). These test procedures will become executable once those phases are implemented.
