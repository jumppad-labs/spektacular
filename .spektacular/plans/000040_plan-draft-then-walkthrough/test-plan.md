---
created_date: "2026-07-31"
status: completed
closed_date: "2026-07-31"
---

# Test Plan: 000040_plan-draft-then-walkthrough

Manual verification procedures for the success metrics the plan's `## Testing Approach` classified as not automatable. The automated proxy for the interruption-count metric already runs in the harbor suite (`tests/harbor/plan-workflow`, `TestAutonomousDrafting`) and is not restated here.

## 1. Comments during the walkthrough produce visible plan revisions

**What to measure**: A change requested mid-walkthrough results in a real revision to the committed plan documents before the workflow concludes — the review is not a rubber stamp.

**How**:
1. In a Spektacular-initialized repository with an approved spec, run a full plan workflow (`/spek-plan`, or `go run . plan new --data '{"name":"<spec>"}'` driven by the skill loop) and let it draft straight through to the `walkthrough` step.
2. During the walkthrough, interrupt with a concrete change request (e.g. "move X out of scope" or challenge one of the drafting assumptions raised in beat 4).
3. After the agent confirms the revision, run `go run . plan file read <plan>/plan.md` (or `research.md` if the change landed there) and locate the revised text. Also run `git -C .spektacular diff` if the plan store is git-tracked to see the diff directly.

**Expected result**: The committed document contains the requested change before sign-off is given; the agent re-summarized the affected part; the revision was made via `plan file write` (no direct `Write`/`Edit` on the store documents).

**Who / when**: Whoever lands this feature, on the first real plan run after merge; repeat spot-checks during normal use.

## 2. No more section-by-section approval prompts in real use

**What to measure**: Across real plan-creation sessions, the user is not asked to approve individual sections between `plan new` and the walkthrough; only genuinely blocking questions (no reasonable default) interrupt.

**How**: Over the next 3–5 real plan workflows run by the team, note every time the agent stops and waits for user input before the walkthrough begins. Classify each stop: blocking question (acceptable) vs. section-approval/confirmation request (regression).

**Expected result**: Zero section-approval stops. Blocking-question stops only where the spec genuinely under-determines a decision. If a typical spec produces any confirmation stop, inspect which gathering step's rendered instruction triggered it and file it against the template (the `TestGatheringStepsProceedWithoutApprovalGates` unit test bans the known phrasings; a novel phrasing means the banned-phrase oracle needs extending).

**Who / when**: All plan-workflow users, informally, over the first weeks after merge.

## 3. No regression in downstream plan quality

**What to measure**: Plans produced under the autonomous flow remain implementable — the implement workflow does not need more clarification round-trips (STOP-and-ask events) than plans produced under the old confirm-each-section flow.

**How**: For the first 2–3 plans drafted under the new flow, run them through `/spek-implement` and count the STOP events that ask the user to resolve something the plan should have settled (drift mismatches excluded). Compare informally against recent pre-change implement runs (e.g. plans 000034–000038).

**Expected result**: STOP-and-ask frequency is not noticeably higher than before. Watch specifically for the plan's recorded open question: whether any implement-side surface rejects the eight-section research.md (`## Drafting assumptions`) — none is known, but if one appears, STOP and raise it rather than patching the implement workflow (out of scope for 000040).

**Who / when**: Whoever runs the first implement workflows against new-flow plans, in the release cycle after merge.
