# Test Plan: 000034_spec-plan-implement-reconciliation

The automated test suite (`internal/steps/implement/steps_test.go`) covers FSM wiring and asserts that every new template's rendered instructions *direct* the correct agent behavior. It cannot verify that an agent actually *follows* those instructions correctly against real content, since the judgment involved (matching a spec requirement to plan coverage; deciding a changelog entry satisfies an acceptance criterion) is inherently semantic. The three procedures below cover that judgment gap.

## 1. A gap like 000032 is caught by the pre-implementation check before implementation proceeds

**What to measure**: Whether `implement`'s `read_plan` step (Step 3.5) actually stops and surfaces a coverage gap, rather than silently proceeding, when a plan's Milestones & Phases omit something the spec requires.

**How**:
1. Pick or construct a plan whose `## Milestones & Phases` section does not cover one of its spec's `## Requirements` or `## Acceptance Criteria` checkboxes (or reuse spec `000032_spec-workflow-pair-programming-enhancements`, whose plan is documented in this feature's own research.md as having silently dropped a milestone — do not run implement against it destructively; inspect the gap by hand instead, or fabricate a disposable test plan/spec pair under a temp `.spektacular` directory).
2. Run `go run . implement new --data '{"name":"<plan-name>"}'` and step through to `read_plan`.
3. Observe the agent's behavior at Step 3.5.

**Expected result**: The agent reads the spec, identifies the uncovered requirement, and stops with a two-option prompt (fix the plan / accept as descoped) — it does not silently continue to `analyze`. Choosing "accept as descoped" writes a `**Descoped requirements**:` entry into the plan and, on a subsequent `read_plan` run against the same plan, the same gap is not re-flagged.

**Who / when**: A maintainer, once after this feature ships, and again after any future change to `templates/steps/implement/01-read_plan.md`'s Step 3.5.

## 2. After implement completes, the user can tell which spec items are satisfied without opening the spec file

**What to measure**: Whether the `finished` step's rendered report actually states spec completion status in the terminal report the user sees, and whether that status is accurate.

**How**:
1. Run a full `implement` workflow for a plan with a real spec end-to-end (this plan's own run against spec `000034_spec-plan-implement-reconciliation` is a live instance of this — see the `finished` step's output from this very run as the first data point).
2. Read the `finished` step's `instruction` field in the JSON response.
3. Independently read the spec via `go run . spec file read 000034_spec-plan-implement-reconciliation.md` and compare its checkbox state against what the report claims.

**Expected result**: The report names both satisfied and unsatisfied spec items without the user needing to separately open the spec file, and the claim matches the spec's actual on-disk checkbox state.

**Who / when**: A maintainer, at the end of this plan's own implement run (immediately available) and spot-checked on the next few `implement` runs against other plans.

## 3. Specification files for completed features accurately reflect what was built

**What to measure**: Whether `reconcile_spec` only checks off spec items that the plan's `## Changelog` record actually shows were satisfied, and leaves unsatisfied ones unchecked.

**How**:
1. After `reconcile_spec` runs for this plan (imminent in this same implement run), read `go run . spec file read 000034_spec-plan-implement-reconciliation.md`.
2. For each `## Requirements`/`## Acceptance Criteria` checkbox now marked `[x]`, manually trace it back to a specific phase entry in this plan's `## Changelog` section and confirm the entry genuinely delivers what the checkbox describes.
3. For each checkbox still `[ ]`, confirm there is no changelog evidence it was actually done (i.e. it wasn't wrongly left unchecked).

**Expected result**: Every `[x]` traces to real changelog evidence; no checkbox is flipped on a partial or speculative match, per the `reconcile_spec` template's own "if in doubt, leave it unchecked" instruction.

**Who / when**: A maintainer, immediately after this plan's own `reconcile_spec` step runs (self-verifying, since this feature's own spec is the first real subject), and periodically on other plans afterward as a spot-check.
