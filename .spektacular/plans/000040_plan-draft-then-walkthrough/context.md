---
created_date: "2026-07-31"
status: completed
closed_date: "2026-07-31"
---

# Context: 000040_plan-draft-then-walkthrough

## Current State Analysis

The plan workflow is a 19-step FSM (`internal/steps/plan/steps.go:31-53`, `new → overview → discovery → architecture → components → data_structures → implementation_detail → dependencies → testing_approach → milestones → phases → open_questions → out_of_scope → assemble → verification → write_plan → write_context → write_research → finished`) driven by step templates in `templates/steps/plan/`. Ten of the gathering steps (02–12) end with a present-for-review gate ("Present it to the user for review… Once the user is happy, advance") — roughly ten interruptions per plan. The conversational walkthrough exists today only as an optional offer inside the terminal step's success branch (`templates/steps/plan/18-finished.md:18-36`): offer once (`:26`), three narrative beats (`:28-32`), interrupt-and-apply-immediately via stage→`plan file write`→`rm` (`:34`), explicit-affirmative close (`:36`).

Key machinery relied on as-is: `State.InProgress()` (`internal/workflow/state.go:25`) treats only `current_step == "finished"` as done; same-step `Goto` re-renders without transitioning (`internal/workflow/workflow.go:184-187`, `renderStep:243`); the cross-kind guard blocks `implement new` while any workflow is in progress (`cmd/resume.go:123-151`); documents get `status: in-progress` on first write (`internal/metadata/merge.go:52`) and are closed to `completed` only by `finished()` (`internal/steps/plan/steps.go:287-324`); working files under `.spektacular/work/<name>/` are template-driven only (no Go references) and are removed by `17-write_research.md:22`; step callbacks are one-line `writeStep` wrappers (`steps.go:67`); templates are mustache, rendered by `internal/stepkit/stepkit.go:55-121` with vars from `internal/steps/plan/strategy.go:18`.

Enforcement surfaces that constrain template edits: `templates/context_directive_test.go:15-96` (every non-terminal step template must contain the marker "not a copy of content already captured elsewhere" exactly once; `*-finished.md` zero times; floor of 29 directive-bearing templates), `templates/work_files_test.go:23-131` (gathering steps 02–12 must reference the work dir; only `spec/08-verification.md` and `plan/17-write_research.md` clean up), `internal/steps/plan/steps_test.go:55-160` (template-content and step-order assertions), and the harbor E2E suite `tests/harbor/plan-workflow/` (canonical step order at `tests/test_plan_workflow.py:599`, rendered-next-step validation at `:894`).

## Per-Phase Technical Notes

### Phase 1.1: Insert the walkthrough step into the plan workflow

**File changes**:
- `internal/steps/plan/steps.go:31-53` — insert `{Name: "walkthrough", Src: []string{"write_research"}, Dst: "walkthrough", Callback: walkthrough()}` before the `finished` entry; retarget `finished`'s `Src` from `["write_research"]` to `["walkthrough"]`.
- `internal/steps/plan/steps.go` (near `:277`, alongside `writeResearch()`/`finished()`) — add `walkthrough()` as a one-line `writeStep("walkthrough", "finished", "steps/plan/18-walkthrough.md", …)` callback, same shape as siblings. No extras needed; the standard + strategy vars (`plan_path`, `context_path`, `research_path`, `plan_name`, `config.command`) supply everything the template uses.
- `templates/steps/plan/18-finished.md` → `git mv` to `templates/steps/plan/19-finished.md`; update the template path string in `finished()` (`steps.go:287-299`).
- New `templates/steps/plan/18-walkthrough.md` — lift the walkthrough prose from the old `18-finished.md:18-36` (beats, interrupt handling, apply-changes-immediately via stage→`{{config.command}} plan file write {{plan_name}}/plan.md --from <scratch>`→`rm`, explicit-affirmative closing gate at `:36`), reframed as mandatory (no offer); reads the committed documents via `plan file read`; closes with `goto --data '{"step":"finished"}'` only after sign-off. Must contain the context-directive marker "not a copy of content already captured elsewhere" exactly once (`templates/context_directive_test.go:15`).
- `templates/steps/plan/19-finished.md` — remove the walkthrough/offer prose from the `{{^plan_incomplete}}` branch (old lines 26-36); keep the document listing (old lines 20-25) and the `{{#plan_incomplete}}` recovery branch (lines 3-16) unchanged.
- `internal/steps/plan/steps_test.go:102` (`TestStepsOrderMatchesExpected`) and `:131` (`TestFSMWalkFromNewToFinished`) — add `walkthrough` to the expected order; add a wiring assertion that `finished`'s `Src` is `["walkthrough"]` only.
- `internal/steps/plan/steps_test.go:203-343` — `finished()` close/skip tests unchanged in behavior; confirm they still pass with the renamed template path.

**Complexity**: Medium
**Token estimate**: ~40k tokens
**Agent strategy**: Single agent, sequential execution (slice, callback, templates, and tests are tightly coupled).

### Phase 1.2: Align the driving skill and end-to-end harness with the new step

**File changes**:
- `templates/skills/workflows/spek-plan/SKILL.md` — extend the loop description (`:13-22`) to include the walkthrough step; update "How to start" resume guidance (`:42-66`) so a resume report with `current_step: walkthrough` reads back the committed documents (via `plan file read`) rather than working files (removed at `17-write_research.md:22`); adjust the completion framing so "finished" means "after sign-off". Re-run `go run . init` locally to refresh `.claude/skills/spek-plan/SKILL.md` (generated copy).
- `templates/context_directive_test.go:50-96` — bump the directive-bearing floor from 29 to 30; `18-walkthrough.md` is non-terminal (must carry marker), `19-finished.md` matches the `*-finished.md` zero-marker rule automatically.
- `templates/skill_resume_test.go` / `templates/skill_list_command_test.go` — verify assertions still hold against the edited SKILL.md (no expected change; run to confirm).
- `tests/harbor/plan-workflow/instruction.md` — add `walkthrough` to the canonical step list and success criteria (workflow reaches finished only via walkthrough sign-off).
- `tests/harbor/plan-workflow/tests/test_plan_workflow.py:599` (`test_steps_executed_in_order`) and `:894` (`test_every_rendered_next_step_is_valid`) — extend the canonical order with `walkthrough`; add an assertion that the `finished` instruction was rendered only after a `walkthrough` instruction.

**Complexity**: Low
**Token estimate**: ~25k tokens
**Agent strategy**: Single agent, sequential execution.

### Phase 2.1: Make the gathering steps draft autonomously and record assumptions

**File changes**:
- `templates/steps/plan/02-discovery.md:42-43` — "Ask only questions the code cannot answer" stays (it is the blocking-question path); "Present a summary of key discoveries to the user" becomes a report-in-passing, not a gate. Knowledge-write propose-then-confirm (`:47`) stays — it is a standing rule outside this spec's scope.
- `templates/steps/plan/03-architecture.md:5-23` — Step 1 keeps the 2-3 options analysis; Step 2 "Get Agreement" is replaced by "pick the best-grounded direction and record it as the leading assumption-log entry (decision, rationale, rejected alternatives)"; `:42` conventions confirmation becomes propose-in-log; `:46` closing gate becomes save-and-advance.
- `templates/steps/plan/04-components.md:15,19`, `05-data_structures.md:15,19`, `06-implementation_detail.md:27,31`, `07-dependencies.md:15,19`, `08-testing_approach.md:36,40`, `09-milestones.md:15,20`, `10-phases.md:35,46`, `11-open_questions.md:7,16,29`, `12-out_of_scope.md:18,22` — remove "Present it to the user for review" / "Once the user is happy/agreed, advance" gates; replace "write the agreed X content" with "write the drafted X content"; `09-milestones.md:15` "resolve any uncertainties now by asking the user" and `11-open_questions.md:16` "ask the user now" become the proceed-unless-blocked rule.
- All ten gathering templates — add the shared block: append judgement calls to `.spektacular/work/{{plan_name}}/assumptions.md` (entry shape: `### <title> (<step>)` / Decision / Rationale / Rejected), and the blocking-question conditional modeled on `templates/steps/implement/01-read_plan.md:73-79` ("if a decision has no reasonable default or requires information only the user holds, STOP and present the options in one block; do not advance past the dependent point until answered — otherwise proceed without interruption").
- `internal/steps/plan/steps_test.go:55` (`TestArchitectureStepContainsOptionsAndAgreementBeat`) — rewrite to assert options analysis + assumption recording, and the *absence* of the agreement gate; extend the other template-content tests (`:65-101`) to assert the assumption-log and proceed-unless-blocked prose in each gathering template and the absence of "Once the user is happy".
- `templates/work_files_test.go:23-63` — extend the gathering-step working-file test so steps 02-12 also reference `assumptions.md` (or add a sibling test); cleanup assertions (`:82`) unchanged — `17-write_research.md` still removes the whole work dir, which includes the assumption log (safe: by then it is folded into research.md, Phase 2.2).

**Complexity**: Medium
**Token estimate**: ~45k tokens
**Agent strategy**: 2-3 parallel agents for independent changes (template edits can be split across steps 02-07 / 08-12; test updates sequential after both).

### Phase 2.2: Fold assumptions into the research document and raise them in the walkthrough

**File changes**:
- `templates/scaffold/research.md` — add a `## Drafting assumptions` section (comment brief: the judgement calls made while drafting — decision, rationale, rejected alternatives — presented for challenge at the walkthrough) after `## Open assumptions`.
- `templates/steps/plan/13-assemble.md:19-39` — add `assumptions.md` → `research.md § Drafting assumptions` to the mapping table; a missing `assumptions.md` is not a STOP (a plan can legitimately have zero judgement calls) — instruct writing an explicit "No drafting assumptions were recorded" line instead.
- `templates/steps/plan/14-verification.md:13,46` — extend research.md's required-section list from 7 to 8 (`## Drafting assumptions`, present and filled or explicitly empty).
- `templates/steps/plan/18-walkthrough.md` — add the assumptions beat: read `research.md § Drafting assumptions` via `plan file read {{plan_name}}/research.md` and walk the entries one by one (decision, rationale, rejected alternatives), inviting challenge, before the closing sign-off question; a challenged assumption is a change request handled by the existing apply-immediately path.
- `templates/skills/workflows/spek-plan/SKILL.md:36-38` — extend the working-files paragraph to name `assumptions.md` as part of the contract and read-back-on-resume set; reinforce the loop framing ("drafting steps do not stop for section approval; only blocking questions interrupt").
- `internal/steps/plan/scaffold_test.go` / `steps_test.go` — extend scaffold-shape and template-content assertions for the new section and the walkthrough beat.

**Complexity**: Medium
**Token estimate**: ~30k tokens
**Agent strategy**: Single agent, sequential execution (scaffold, verification list, assemble mapping, and walkthrough beat must stay mutually consistent).

### Phase 2.3: Verify the autonomous flow end to end

**File changes**:
- `tests/harbor/plan-workflow/instruction.md` — success criteria updated: zero user-directed confirmation questions between `plan new` and the walkthrough; assumption log populated; documents `status: in-progress` until sign-off, `completed` after.
- `tests/harbor/plan-workflow/tests/test_plan_workflow.py` — new tests: (a) no drafting-step instruction response contains a question that waits on the user (scan the transcript between `overview` and `walkthrough` steps for the agent posing confirmation questions); (b) `research.md` contains a filled `## Drafting assumptions` section with decision/rationale/rejected entries; (c) frontmatter `status` of all three docs is `completed` only after the `finished` step, `in-progress` at the `walkthrough` step (extend the existing frontmatter helpers); existing `test_no_unfilled_scaffold_slots` (`:612`) covers the new scaffold section automatically.
- `tests/harbor/plan-workflow/solution/solve.sh` — update the scripted solution for the new flow if it enumerates steps.
- `Makefile:73-78` — no change; run `harbor-test-plan` to validate.

**Complexity**: Medium
**Token estimate**: ~35k tokens
**Agent strategy**: Single agent, sequential execution (test authoring against one harness).

## Testing Strategy

Per-phase test surfaces (details in each phase's notes above):

- **Phase 1.1** — `internal/steps/plan/steps_test.go`: extend `TestStepsOrderMatchesExpected` (`:102`) and `TestFSMWalkFromNewToFinished` (`:131`) with `walkthrough`; add a wiring assertion that `finished.Src == ["walkthrough"]`; add template-content assertions for the walkthrough template (sign-off gate phrase, apply-immediately store-write instruction) and for the slimmed `19-finished.md` (offer prose absent, recovery branch intact). Existing `finished()` close/skip tests (`:203-343`) unchanged.
- **Phase 1.2** — `templates/context_directive_test.go`: floor 29→30; `templates/skill_resume_test.go` and `skill_list_command_test.go` re-run against the edited SKILL.md; harbor `test_steps_executed_in_order` (`:599`) and `test_every_rendered_next_step_is_valid` (`:894`) extended with `walkthrough`, plus a finished-only-after-walkthrough ordering assertion.
- **Phase 2.1** — rewrite `TestArchitectureStepContainsOptionsAndAgreementBeat` (`steps_test.go:55`) to assert options analysis + assumption recording and the absence of the agreement gate; extend the sibling template-content tests (`:65-101`) to assert assumption-log and proceed-unless-blocked prose per gathering step and the absence of "Once the user is happy"; extend `templates/work_files_test.go` so gathering steps reference `assumptions.md`.
- **Phase 2.2** — `internal/steps/plan/scaffold_test.go` extended for the `## Drafting assumptions` research.md section; template-content assertions for the assemble mapping row, the verification section list (7→8 for research.md), and the walkthrough assumptions beat.
- **Phase 2.3** — harbor `tests/test_plan_workflow.py`: new zero-confirmation-question test over the drafting segment of the transcript, populated-assumptions test against the committed research.md, and status-lifecycle test (in-progress at walkthrough, completed after finished). Existing `test_no_unfilled_scaffold_slots` (`:612`) covers the new section automatically.

Per project convention, no redundant re-assertion of behaviors already covered: same-step re-render resume (`internal/workflow/workflow_test.go:87`), cross-kind blocking (`cmd/cross_kind_test.go`), and store-write mechanics (`cmd/plan_file_test.go`) are relied on, not re-tested.

## Project References

- Spec: `.spektacular/specs/000040_plan-draft-then-walkthrough.md` (read via `go run . spec file read 000040_plan-draft-then-walkthrough.md`)
- Prior plan precedents: `000035_plan-walkthrough-conversation` (walkthrough prose, instruction-prose-over-Go split), `000034_spec-plan-implement-reconciliation` (step insertion before `finished`) — read via `go run . plan file read <name>/plan.md`
- Core source: `internal/steps/plan/steps.go`, `internal/workflow/{workflow,state}.go`, `cmd/resume.go`, `internal/stepkit/stepkit.go`
- Templates: `templates/steps/plan/*.md`, `templates/scaffold/research.md`, `templates/skills/workflows/spek-plan/SKILL.md`
- Test suites: `internal/steps/plan/steps_test.go`, `templates/{context_directive,work_files,skill_resume}_test.go`, `tests/harbor/plan-workflow/` (run via `make harbor-test-plan`, `Makefile:73-78`)

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|-------------|----------------|
| Low | ~10k | Single agent, sequential |
| Medium | ~25k | 2-3 parallel agents |
| High | ~50k+ | Parallel analysis, sequential integration |

Phase budgets: 1.1 ~40k (Medium, sequential — tightly coupled slice/template/test edits), 1.2 ~25k (Low), 2.1 ~45k (Medium — template edits parallelizable 02-07 / 08-12, tests sequential after), 2.2 ~30k (Medium, sequential — scaffold/verification/assemble/walkthrough must stay consistent), 2.3 ~35k (Medium, sequential).

## Migration Notes

No data migration. Existing committed plans are untouched — the new walkthrough step only affects workflows started (or resumed pre-`write_research`) after the change. A workflow interrupted exactly on `write_research` or `finished` across the upgrade boundary would resume against the new step table; `write_research`'s next step becomes `walkthrough` (benign — the state file stores only the current step name, and both old and new tables contain it). Regenerate installed skill copies (`.claude/skills/spek-plan/SKILL.md` etc.) via `go run . init` after Phase 1.2 and 2.2 land. Template renumbering (`18-finished.md` → `19-finished.md`) is a `git mv` plus the one path-string update in `finished()`.

## Performance Considerations

None. The change adds one FSM step and template prose; no hot paths, no new I/O beyond one additional instruction render per plan run.
