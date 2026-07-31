---
created_date: "2026-07-31"
status: completed
closed_date: "2026-07-31"
---

# Feature: 000040_plan-draft-then-walkthrough

## Overview

The planning workflow will produce a complete draft plan autonomously and then walk the user through it for review, instead of stopping to ask for confirmation after each individual section. While drafting, the agent records the assumptions and judgement calls it makes and presents them for discussion alongside the finished plan; only genuinely blocking questions — decisions with no reasonable default — interrupt the drafting. This mirrors how a team actually works (a planner produces a proposal and reviews it with stakeholders) and removes the tedium of a dozen serial approval gates for anyone creating a plan.

## Requirements

- [ ] **Autonomous drafting**
  The system produces a complete draft plan — every section — without asking the user to confirm individual sections along the way.

- [ ] **Assumptions are recorded, not asked**
  When drafting requires a judgement call that has a reasonable default (including the choice of design direction from among the alternatives the agent considered), the system makes the call, records it as an assumption together with its rationale and the alternatives it rejected, and continues drafting.

- [ ] **Blocking questions interrupt immediately**
  When drafting hits a decision with no reasonable default — mutually exclusive directions the agent cannot responsibly choose between, or information only the user has — the system asks the user at that moment rather than deferring the question to the end or guessing.

- [ ] **Assumptions are presented with the final plan**
  When the draft plan is presented for review, the recorded assumptions are presented alongside it as a first-class part of the review, so the user can see every judgement call the agent made and challenge any of them.

- [ ] **Review is a guided walkthrough**
  After drafting completes, the system walks the user through the plan section by section, inviting comments as it goes rather than asking for a single up-front approval.

- [ ] **Comments produce revisions**
  When the user comments or requests a change during the walkthrough, the system revises the affected parts of the plan, confirms the revision, and re-summarizes anything the change affected before continuing.

- [ ] **Explicit sign-off ends review**
  The plan is only treated as approved when the user gives an explicit affirmative answer to a direct closing question; silence, topic changes, or ambiguous replies keep the review open, and the walkthrough can loop through further comment-and-revise rounds until sign-off.

- [ ] **Interrupted drafting can resume**
  If a drafting session is interrupted before the walkthrough completes, a later session can resume from where it left off without losing already-drafted sections or recorded assumptions.

## Constraints

- Must apply to the plan workflow only — the spec workflow's existing per-section confirmation model must not change as part of this work.
- Must not change the shape, names, or location of the final plan documents the workflow produces, so the downstream implement workflow continues to consume approved plans unchanged.
- Must remain driveable by the same agent-and-CLI mechanism the workflows use today — no new interactive UI or external service may be required to review a plan.

## Acceptance Criteria

- [ ] **No per-section approval prompts**
  Running the plan workflow end to end on a spec with no genuinely blocking decisions produces a complete draft plan with zero confirmation questions asked between starting the draft and beginning the walkthrough.

- [ ] **Assumption log exists and is populated**
  After drafting completes, a record of the assumptions made during drafting is visible to the user, and each entry states the decision taken, the rationale, and any alternatives that were rejected — including the chosen design direction.

- [ ] **Blocking question stops the draft**
  When the spec being planned contains a decision with no reasonable default (e.g. a choice only the user can make), the user is asked that question during drafting, and the drafting does not proceed past the point that depends on the answer until the user responds.

- [ ] **Walkthrough presents plan and assumptions**
  When drafting completes, the user is walked through the draft plan section by section, and the recorded assumptions are presented as part of that walkthrough — not omitted, and not dumped only into a file without being raised.

- [ ] **Comment revises the plan**
  When the user requests a change during the walkthrough, the saved plan documents are updated to reflect the change before the walkthrough concludes, and the user is shown what was revised.

- [ ] **No sign-off, no approval**
  If the user never gives an explicit affirmative answer to the closing question, the plan is not marked approved and the downstream implement workflow does not treat it as ready.

- [ ] **Resume preserves draft state**
  Interrupting a session mid-draft and starting a new session resumes the plan workflow with all previously drafted sections and recorded assumptions intact — none are re-asked or re-drafted from scratch.

## Technical Approach

- Prefer adapting the existing plan workflow steps rather than building a new workflow.
- The plan workflow's existing finishing walkthrough already has the right review shape (section-by-section walkthrough, interrupt-to-comment, apply changes immediately, explicit closing sign-off); prefer promoting it to be the primary review mechanism rather than inventing a new one.
- Assumptions likely need a durable record that survives session interruption, in the same spirit as the workflow's existing working files; the exact form is left to the plan.

## Success Metrics

- The number of times the user is interrupted between starting a plan and beginning the walkthrough drops from roughly one prompt per section (~10 per plan today) to only genuinely blocking questions — zero for a typical plan.
- Plans reviewed via the walkthrough are still meaningfully reviewed: user comments during the walkthrough result in visible plan revisions, rather than the walkthrough becoming a rubber stamp.
- The user no longer reports being asked for approval section-by-section while a plan is being produced.
- No regression in downstream quality: plans produced under the new flow are implementable by the implement workflow without more clarification round-trips than before.

## Non-Goals

- Keeping the old per-section confirmation flow available as a configurable alternative is out of scope — the draft-then-walkthrough model replaces it outright.
- Changing what content the plan gathers is out of scope — the set of plan sections stays the same; only when the user is consulted changes.
- Applying the same model to the implement workflow's interaction points is out of scope.
- A tunable threshold for what counts as a "blocking" question (akin to the spec trigger threshold) is out of scope — the drafting agent uses its judgement.
