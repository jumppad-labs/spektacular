# Feature: 000034_spec-plan-implement-reconciliation

## Overview

Today, a plan can quietly drift away from the specification it was created to satisfy — a milestone can be redefined mid-flight, scope can shrink, and nothing checks the final work against what the specification actually promised. Users only discover this by manually re-reading the specification against the plan and changelog after the fact, if they think to do so at all.

This feature closes that gap by having the assistant check the plan against the specification's requirements before implementation begins, and check the completed work against the specification's requirements after implementation finishes. Requirements and acceptance criteria that are genuinely satisfied get marked as complete; anything left unaddressed is called out explicitly to the user — met, deferred, or descoped — as part of the standard completion report, rather than staying silently unresolved. This gives users confidence that what gets built matches what was asked for, and makes any gap visible at the moment it matters instead of being discovered later by accident.

## Requirements

- [ ] **Plans are checked against their specification before implementation proceeds**
  Before implementation work begins in earnest, the assistant must verify that the plan addresses every requirement and acceptance criterion in the specification it was created from.

- [ ] **Coverage gaps are surfaced to the user, not silently accepted**
  If a requirement or acceptance criterion from the specification has no corresponding coverage in the plan, the assistant must surface this to the user before proceeding, rather than continuing implementation on an incomplete plan.

- [ ] **The user can decide how to resolve a coverage gap**
  When a coverage gap is surfaced, the user must be able to choose how to proceed — for example, updating the plan to add the missing coverage, or explicitly accepting the gap as an intentional descope — rather than the assistant unilaterally deciding.

- [ ] **Completed work is checked against the specification after implementation finishes**
  After implementation work is complete, the assistant must compare what was actually built against the specification's requirements and acceptance criteria.

- [ ] **Satisfied requirements and acceptance criteria are marked complete in the specification**
  Requirements and acceptance criteria that the completed work genuinely satisfies must be marked as complete in the specification, so the specification's own record reflects reality.

- [ ] **Unsatisfied requirements and acceptance criteria are reported, not left silently unresolved**
  Requirements and acceptance criteria that remain unsatisfied after implementation must be explicitly reported to the user, along with why — for example, deferred, descoped, or simply not yet done — rather than being left unmentioned.

- [ ] **Specification completion status is part of the standard completion report**
  When implementation work concludes, the summary presented to the user must include the specification's completion status alongside the usual summary of changes made, so the user does not have to separately go looking for it.

## Constraints

- Must operate within the existing plan and implement workflows — this is additive verification within those workflows, not a new standalone workflow or command.
- Must not require the user to manually locate or open the specification file to see its completion status — the status must reach the user through the existing completion reporting.
- Must work for plans/specifications that are resumed or interrupted partway through, not only for a workflow run start-to-finish in one sitting.

## Acceptance Criteria

- [ ] **Plan coverage check runs before implementation proceeds**
  For a plan created from a specification, before implementation work proceeds, a coverage comparison between the plan and the specification's Requirements and Acceptance Criteria has occurred and its outcome is visible to the user.

- [ ] **Every specification requirement and acceptance criterion is accounted for in the coverage check**
  The coverage comparison addresses every checklist item under the specification's Requirements and Acceptance Criteria sections — none are silently skipped.

- [ ] **A plan missing coverage for a requirement is flagged before implementation proceeds**
  Given a plan whose milestones/phases do not address one or more of the specification's requirements or acceptance criteria, the user is shown which ones are missing before implementation work begins.

- [ ] **A fully-covered plan proceeds without requiring extra user action**
  Given a plan that addresses every requirement and acceptance criterion in the specification, implementation proceeds without the user having to resolve any flagged gaps.

- [ ] **The user's resolution of a flagged gap is honored**
  Given a flagged coverage gap, after the user chooses to either update the plan or explicitly accept the gap as descoped, implementation proceeds consistent with that choice — an accepted gap is not flagged again for the same plan.

- [ ] **Specification checkboxes reflect satisfied requirements and acceptance criteria after implementation**
  After implementation work concludes, checklist items under the specification's Requirements and Acceptance Criteria sections that the completed work satisfies are marked checked (`- [x]`) in the specification file.

- [ ] **Specification checkboxes are not marked complete without justification**
  After implementation work concludes, no checklist item in the specification is marked checked unless the completed work (as reflected in the plan's implementation record) actually satisfies it.

- [ ] **Unsatisfied specification items are named with a reason in the completion report**
  The completion report presented to the user at the end of implementation lists any specification requirement or acceptance criterion that remains unchecked, together with a stated reason (e.g. deferred, descoped, not yet done).

- [ ] **Specification status appears in the same report as the changelog summary**
  The completion report presented to the user contains both the summary of changes made and the specification's completion status, without requiring the user to open a separate file to find either.

## Technical Approach

- The pre-implementation check could be placed alongside the plan workflow's existing structural verification, or at the start of the implement workflow — the planning process should choose based on which fits the existing step sequence better.
- The post-implementation check could be its own step, or be folded into the implement workflow's existing final reporting step — again left to the planning process to decide.
- Consider reusing the plan's already-accumulated implementation record (phase-by-phase notes on what was actually done) as the source of truth for what to compare against the specification, rather than re-deriving it from scratch, since that record already exists for the changelog.

## Success Metrics

- A gap like the one found in spec 000032 — a plan silently descoping a milestone relative to its specification — is caught by the pre-implementation check before implementation proceeds, rather than discovered later by manual review.
- After an implement workflow completes, a user can tell which specification requirements and acceptance criteria are satisfied and which are not without opening the specification file themselves.
- Specification files for completed features accurately reflect what was built, rather than remaining entirely unchecked despite the linked work being finished.

## Non-Goals

- Retroactively auditing or fixing specifications completed before this feature ships (e.g. going back to reconcile spec 000032 itself) — this feature applies going forward only.
- Enforcing coverage checks on the spec workflow itself (e.g. checking a specification against some upstream discussion that produced it) — this feature covers plan-vs-specification and implementation-vs-specification only.
- Automatically fixing or writing plan content to close a coverage gap — the assistant surfaces gaps and the user decides how to resolve them; it does not auto-generate missing plan content.
- Blocking or hard-failing the workflow on a coverage gap — the check surfaces the gap and asks the user, it does not make the workflow refuse to proceed.
