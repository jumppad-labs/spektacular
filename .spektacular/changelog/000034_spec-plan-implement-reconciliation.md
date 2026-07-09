# Spec-Plan-Implement Reconciliation

## What was built

The `implement` workflow now guards both ends of a feature's lifecycle against drift from its specification.

Before any code gets written, a new check in the `read_plan` step reads the specification a plan was created from and confirms every checkbox under its Requirements and Acceptance Criteria has corresponding coverage somewhere in the plan's Milestones & Phases. If something is missing — whether the plan never covered it, or someone hand-edited the plan afterward and dropped it — the workflow stops and asks the user to either update the plan or explicitly accept the gap as an intentional descope. An accepted gap is recorded directly in the plan document itself (a `**Descoped requirements**:` list), so it is not re-flagged on a later run against the same plan. A plan that already covers everything proceeds exactly as before, with no added friction.

After implementation finishes, a new `reconcile_spec` step compares the specification's checklist against the plan's accumulated phase-by-phase implementation record and checks off every item the completed work genuinely satisfies, writing the result back to the specification file. The workflow's final report — the same one that already summarizes the changelog and completed phases — now also states which specification items are satisfied and, for anything still open, the reason recorded during reconciliation (deferred, descoped, or not attempted), so nobody has to separately open the specification to find out.

## Why it matters / what it enables

A plan can quietly drift from the specification it was created to satisfy — a milestone redefined mid-flight, scope narrowed without anyone noticing — and until now nothing checked the finished work against what the specification actually promised. Users only discovered this by manually re-reading the specification against the plan after the fact, if they thought to at all. This feature closes that gap at both ends: a coverage gap is now surfaced at the moment it matters, before implementation starts on an incomplete plan, and the specification's own record is kept honest afterward, so "done" in the specification actually means done.

## Deviations from the plan

None in design or scope. Two incidental fixes were needed beyond the plan's own listed files: adding the new `reconcile_spec` FSM step changed the `implement` workflow's total step count from 12 to 13, which broke two hardcoded assertions in `cmd/implement_test.go` (`TestImplementSteps_ListsAllSteps`, `TestImplementStatus_ReportsUncheckedPhases`) — both updated to match the new count and step list. No behavior, scope, or design decision changed from what the plan specified.
