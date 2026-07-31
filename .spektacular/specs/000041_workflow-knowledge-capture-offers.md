---
created_date: "2026-07-31"
status: completed
closed_date: "2026-07-31"
---

# Feature: 000041_workflow-knowledge-capture-offers

## Overview

The development workflows will actively recognize the two moments when durable project knowledge most often surfaces — a non-obvious discovery made while implementing a change, and a correction the user makes while reviewing a drafted plan — and offer, in that moment, to save the insight to the project's knowledge base. Today that recognition relies on a standing background instruction that demonstrably fails during long working sessions: discoveries get recorded only in per-change history that future work never reads, and plan corrections are applied without asking what they reveal, so the same mistakes and re-discoveries repeat. Everyone who runs these workflows benefits — an insight captured once, at the moment it appears, saves the next plan or implementation from paying for it again.

## Requirements

- [x] **Implementation discoveries are assessed for knowledge-worthiness**
  When the implement workflow records what a completed phase discovered, the workflow assesses each discovery for durability — whether it holds beyond the current change (an undocumented coupling, a surprising contract, a convention inferred from behavior) — rather than only filing it in the change's own history.

- [x] **Durable discoveries trigger an offer during the run**
  When a discovery is judged durable, the workflow offers the user, during the run itself and without waiting to be asked, to save it to the project knowledge base — stating what would be captured and why it is worth keeping.

- [x] **Plan corrections are assessed for what they reveal**
  When the user requests a change or challenges an assumption while reviewing a drafted plan, the workflow — after applying the correction — assesses whether the correction reveals durable knowledge: a wrong default the drafting agent chose, a constraint or convention the user knew that the codebase does not record, or a term that was misunderstood.

- [x] **Revealing corrections trigger an offer during the review**
  When a plan correction is judged revealing, the workflow offers the user, during the same review conversation, to save the underlying insight to the knowledge base — phrased as the general lesson, not the specific edit.

- [x] **Declines are final for the conversation**
  When the user declines an offer, the same item is not offered again in that conversation; only an explicit acceptance leads to a capture.

- [x] **Non-durable items stay where they are**
  Discoveries and corrections that only matter to the current change are not offered — they continue to live solely in the change's own record. The offer must be selective enough that it does not become noise on every phase or every correction.

## Constraints

- Knowledge writes must remain offer-then-confirm — the workflows must never write to the knowledge base without the user's explicit acceptance, honoring the existing propose-then-confirm contract of the knowledge tooling.
- The actual capture must go through the existing knowledge tooling and its scope/category routing — this feature adds recognition and offer points, not a new write path.
- Must not change the shape or content of the existing per-change records — discoveries continue to be recorded in the change's own history exactly as today; the knowledge offer is additive.
- Must not add new interruption points during autonomous drafting or phase implementation — offers may only surface at moments where the workflow already converses with the user (recording a phase's outcome, or the plan review conversation).
- Must remain driveable by the same agent-and-CLI mechanism the workflows use today — no new interactive UI.

## Acceptance Criteria

- [ ] **A durable discovery produces a visible offer**
  In an implement run whose phase surfaces a durable, non-obvious discovery, the agent's conversation contains an offer to save that discovery to the knowledge base — naming what would be captured — before the workflow finishes, without the user having prompted for it.

- [ ] **A revealing plan correction produces a visible offer**
  In a plan review where the user requests a change that reveals a wrong drafting assumption or an unrecorded constraint, the agent's conversation contains, after the correction is applied, an offer to capture the underlying lesson in the knowledge base.

- [ ] **Acceptance produces a knowledge entry; decline produces none**
  When the user accepts an offer, a matching entry exists in the knowledge store afterward; when the user declines, no entry is written and the same item is not offered again in that conversation.

- [ ] **Trivial items produce no offer**
  In a run whose discoveries and corrections are all change-specific (nothing durable), no knowledge-capture offer appears — the workflow completes exactly as it does today.

## Technical Approach

- Prefer expressing the assessment-and-offer behavior as workflow instruction prose (the mechanism that demonstrably works in this codebase), following the existing precedent of the plan discovery step's optional learning-capture beat — the analysis behind this spec found the standing repository-level trigger alone fails during long workflow sessions precisely because it is not repeated in-band.
- The natural offer points are the moments the workflows already talk to the user about outcomes: the implement workflow's per-phase discovery recording, and the plan review conversation after a correction is applied (consistent with the no-new-interruption-points constraint).
- Prefer enforcing the presence of the assessment/offer direction with the existing template-content test style, so removing it fails the suite.
- Known risk to weigh in planning: offer fatigue — the assessment needs a selectivity bar (durable and non-obvious beyond the current change) so typical runs produce few or no offers.

## Success Metrics

- Durable discoveries stop requiring a user prompt to be captured: in sessions where a genuinely durable insight surfaces, the workflow raises the offer itself — the user no longer has to ask "shouldn't this go in the knowledge base?" after the fact (the motivating incident: four durable discoveries in one implement run, zero offers).
- Plan corrections that reveal wrong drafting assumptions result in knowledge entries when the user accepts, and subsequent plans stop repeating the corrected assumption.
- Offers stay rare enough to be signal: a typical run with no durable discoveries produces zero offers, and users do not report offer fatigue.
- The knowledge base grows from real workflow output (discoveries and corrections) rather than only from ad-hoc user requests.

## Non-Goals

- The spec workflow's interaction points are untouched — corrections made while gathering or reviewing a specification are not assessed for knowledge-worthiness in this feature; that is a possible follow-up.
- Correction moments outside the plan review conversation (for example, when a user amends a plan through the implement workflow's plan-mismatch gates) are out of scope for the assessment behavior.
- No retroactive mining: existing per-change records, past plans, and historical changelogs are not scanned for knowledge candidates.
- No tunable sensitivity setting for what counts as durable or offer-worthy — the assessing agent uses its judgement, consistent with the workflows' existing approach to judgement thresholds.
