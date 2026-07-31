# Working Context: plan-draft-then-walkthrough (spec 000040)

## Problem identified

The plan workflow currently stops for explicit user confirmation on nearly every
section-gathering step (~10 of the 18 steps in `templates/steps/plan/`). The user
finds this tedious and unrealistic: "at the moment you ask me for confirmation on
every section, this is quite tedious and not really how a team would work."

## Desired behaviour (user's own framing)

"What would happen is you would produce the plan and then walk me through it
allowing me to comment, you would then modify the plan based on my changes.
Assumptions would be discussed with the final plan, blocking questions would be
covered as they arise."

## Key decisions (user answered directly)

- **Architecture options during autonomous drafting**: the agent picks the
  best-grounded direction itself and records it (with rationale and rejected
  alternatives) as a first-class assumption for the walkthrough — "pick & flag",
  not blocking, not threshold-based.
- **Scope**: plan workflow only. The spec workflow keeps its per-section
  confirmation model; applying the same model there is a possible follow-up.

## Drafting approach for this spec session

Per the user's stored preference (skip per-step confirmation prompts, drive
workflows straight through), all section working files were drafted by the agent
from the discussion and will be presented together for review at the final
walkthrough. Non-goals in particular were inferred, not user-stated, and are
flagged for explicit confirmation during the final review:
- no config toggle to keep the old per-section flow (replacement is outright)
- plan content/sections unchanged, only interaction model changes
- implement workflow untouched
- no tunable "blocking question" threshold

## Relevant current-state observations

- `templates/steps/plan/18-finished.md` already implements a walkthrough with
  interrupt/comment/apply-change/resume semantics and explicit closing sign-off —
  the desired model promotes it to primary review mechanism.
- Working files under `.spektacular/work/<plan_name>/` are the existing
  session-resume persistence; assumptions will likely need a similar durable record
  (left to the plan to design).
