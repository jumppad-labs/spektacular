---
created_date: "2026-07-31"
status: completed
closed_date: "2026-07-31"
---

# Context: 000041_workflow-knowledge-capture-offers

## Current State Analysis

Knowledge capture during workflow runs currently depends on a standing background trigger (`templates/agents/knowledge-trigger.md`, injected into AGENTS.md by `init`) plus one in-band beat: the plan discovery step's optional learning capture (`templates/steps/plan/02-discovery.md:45-47`). Neither of the two moments this feature targets has any in-band direction today:

- `templates/steps/implement/07-update_changelog.md` — Step 2 (line 13-42) writes the per-phase changelog entry whose `**Discoveries**` field (line 28) captures "anything the next phase or a future maintainer should know"; Step 3 (line 44-64) is the continue-or-pause conversation. Nothing between them assesses discoveries for durability; no implement template mentions the knowledge base at all, and neither does `spek-implement/SKILL.md`. The Discoveries field acts as a decoy capture sink: the plan changelog becomes a historical artifact once the workflow closes.
- `templates/steps/plan/18-walkthrough.md` — line 18 routes challenged drafting assumptions into the change-request path; line 20 is the apply-immediately sequence (edit → stage → `plan file write` → confirm → remove scratch → resume). Corrections are applied without asking what they reveal.
- Enforcement surfaces exist and are proven: `renderStep` helpers (`internal/steps/implement/steps_test.go:44-52`, `internal/steps/plan/steps_test.go:45-53`) render real templates; `TestDiscoveryStepUsesKnowledgeCommands` (`internal/steps/plan/steps_test.go:121-132`) is the model knowledge-prose content test.
- The motivating incident: the implement run for plan 000040 produced four durable discoveries (harbor oracle drift couplings) and zero offers; the knowledge entry `architecture/testing-architecture.md` had to be written after the user prompted for it.

## Per-Phase Technical Notes

### Phase 1.1: Discovery assessment and offer in the implement changelog step

**File changes**:

- `templates/steps/implement/07-update_changelog.md` — insert a new subsection between the end of Step 2 (after the `rm .spektacular/tmp/plan_update.md` block, line 42) and `### Step 3` (line 44), e.g. `### Step 2b: Assess discoveries for durable knowledge`. Content: (1) framing — re-read the `**Discoveries**` entry just written; (2) durability test with examples (undocumented coupling, surprising contract, convention inferred from behavior) and the explicit bar: only items that hold beyond this change and are non-obvious qualify — most phases produce none, and when none qualify, say nothing and continue to Step 3; (3) the offer — name what would be captured and why it is worth keeping, then hand an acceptance to the `spek-knowledge` skill (its propose-then-confirm flow owns scope/category and the write); never invoke `{{config.command}} knowledge write` without the user's explicit acceptance; (4) outcomes — decline is final for that item for the rest of the conversation, deferral may be re-raised later. Do NOT reference `{{config.command}} skill spek-knowledge` (unresolvable — `cmd/skill.go:49-57,68-81` only maps `templates/skills/skill_<name>.md`); name the skill or the raw `{{config.command}} knowledge` commands, mirroring `templates/steps/plan/02-discovery.md:45-47` and `templates/agents/knowledge-trigger.md`.
- `internal/steps/implement/steps_test.go` — add `TestUpdateChangelogStepOffersKnowledgeCaptureForDurableDiscoveries` beside the existing update_changelog tests (after `TestUpdateChangelogStepBranchesOnUncheckedPhases`, :311-317). Use `renderStep(t, updateChangelog())` (:44-52); model assertions on `TestDiscoveryStepUsesKnowledgeCommands` (`internal/steps/plan/steps_test.go:121-132`): `require.Contains` on distinctive lowercase anchors via `strings.ToLower(out)` — the durability assessment ("durable"), the offer ("offer"), the confirm gate ("explicit acceptance"/"confirm"), decline-finality ("not offered again" or equivalent), and `spek-knowledge`; plus `require.NotContains(out, "skill spek-knowledge")` guarding the unreachable-invocation mistake. Keep anchors distinctive per `templates/context_directive_test.go:17` convention.

**Verification**: `go test ./...` green; existing tests `TestUpdateChangelogStepSpecifiesEntryFields` (:292-302), `TestUpdateChangelogStepCreatesSectionOnFirstInvocation` (:304-309), `TestUpdateChangelogStepBranchesOnUncheckedPhases` (:311-317), `TestStopOnMismatchDirectivePresentInEveryNonTerminalTemplate` (:329-347) unchanged and passing.

**Complexity**: Low
**Token estimate**: ~20k
**Agent strategy**: Single agent, sequential execution

### Phase 2.1: Correction assessment and offer in the plan walkthrough step

**File changes**:

- `templates/steps/plan/18-walkthrough.md` — extend the change-request paragraph (line 20). After "confirm the update to the user, remove the scratch file" and before "then resume the walkthrough", or as an immediately following paragraph, add the assessment beat: (1) framing — before resuming, consider what the correction reveals: a wrong default chosen while drafting, a constraint or convention the user knew that the codebase does not record, a term that was misunderstood; (2) the bar — most corrections are change-local edits and produce no offer; only a durable, generalizable lesson qualifies; (3) the offer — in this same review conversation, offer to capture the general lesson (not the specific edit) via the `spek-knowledge` skill, wait for explicit acceptance before any write; (4) decline is final for that item for the remainder of the conversation. Constraint: introduce no `plan goto` text ahead of the existing occurrence at line 27 — the harbor verifier's `INSTRUCTION_NEXT_STEP_RE` (`tests/harbor/plan-workflow/tests/test_plan_workflow.py:187-190`, consumed at :1101-1148) takes the first match. Do not direct any `{{config.command}} skill <name>` invocation (would require an `EXPECTED_SKILLS_PER_STEP` walkthrough entry, :88-103).
- `internal/steps/plan/steps_test.go` — add `TestWalkthroughStepOffersKnowledgeCaptureForRevealingCorrections` beside `TestWalkthroughStepIsMandatoryGuidedReview` (:418-437). Use `renderStep(t, walkthrough())` (:45-53); assert lowercase anchors for the reveal-assessment ("what the correction reveals" or equivalent), the general-lesson phrasing ("general lesson"), the confirm gate, decline-finality, and `spek-knowledge`; `require.NotContains(out, "skill spek-knowledge")`.

**Verification**: `go test ./...` green; existing walkthrough tests (`TestWalkthroughStepIsMandatoryGuidedReview` :418-437, `TestWalkthroughStepWalksDraftingAssumptions` :464+, `TestWalkthroughWiring` :191-200) unchanged and passing.

**Complexity**: Low
**Token estimate**: ~20k
**Agent strategy**: Single agent, sequential execution

### Phase 2.2: End-to-end confirmation that the workflow surface is undisturbed

**File changes**: none expected. Run `make harbor-test-plan` (requires the `harbor` CLI, Docker, and Claude credentials; ~20–25 minutes — see knowledge entry `architecture/testing-architecture.md`). Expected outcome: green with zero oracle edits — the change adds no steps (`EXPECTED_STEP_ORDER` :61-82 untouched), no skill references (`EXPECTED_SKILLS_PER_STEP` :88-103 untouched), no early `plan goto` in the walkthrough instruction, and the walkthrough window is exempt from `test_no_confirmation_questions_during_drafting` (:831-857, exemption at :847). If the run surfaces a genuine drift caused by the new prose, fix the prose first; only update an oracle if the prose is correct and the oracle's expectation is what changed, and record that in the phase changelog entry. Note: the scripted reference solution (`tests/harbor/plan-workflow/solution/solve.sh:299-303`) never issues a correction, so this run confirms non-disturbance, not offer behavior.

**Complexity**: Low (long wall-clock, minimal editing)
**Token estimate**: ~10k
**Agent strategy**: Single agent, sequential execution

## Testing Strategy

- **Phase 1.1** — template-contract layer only: one new rendered-instruction test (`renderStep(t, updateChangelog())`) asserting the assessment/offer/confirm/decline anchors and `NotContains("skill spek-knowledge")`; the four existing update_changelog tests act as regression guards that entry shape, section creation, branching, and STOP-on-mismatch prose survived the insertion. No Go mechanics change, so no new unit tests.
- **Phase 2.1** — same shape against `renderStep(t, walkthrough())`: new content test for the correction-assessment beat; existing mandatory-review, assumptions-beat, and wiring tests guard the host behavior. The `NotContains` guard and the no-early-`plan goto` constraint are the two negative assertions that matter.
- **Phase 2.2** — the harbor plan-workflow suite as a whole-system regression check, expected green with zero oracle edits. It does not exercise the offer behaviorally (scripted solution issues no corrections) — that residual proof is manual, captured in the implementation test plan along with the spec's four success metrics (offer-without-prompting, entries-on-acceptance, offer rarity, knowledge-base growth from workflow output).

## Project References

- Spec (source of truth): `go run . spec file read 000041_workflow-knowledge-capture-offers.md`
- Prior plan precedent: `go run . plan file read 000040_plan-draft-then-walkthrough/plan.md` — template-prose feature shape, walkthrough step origin.
- Knowledge entries: `architecture/testing-architecture.md` (three test layers, harbor coupling rules), `architecture/workflow-steps.md` (step/template/bundle mechanics) — via `go run . knowledge read --data '{"scope":"project","path":"<path>"}'`.
- Capture tooling the offers hand off to: `templates/skills/workflows/spek-knowledge/SKILL.md`; recognition vocabulary: `templates/agents/knowledge-trigger.md`.
- Harbor suite and oracles: `tests/harbor/plan-workflow/tests/test_plan_workflow.py`, reference solution `tests/harbor/plan-workflow/solution/solve.sh`; run via `make harbor-test-plan`.

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|-------------|----------------|
| Low | ~10k | Single agent, sequential |
| Medium | ~25k | 2-3 parallel agents |
| High | ~50k+ | Parallel analysis, sequential integration |

All three phases are Low tier; the whole plan is comfortably a single-session, single-agent implementation (~50k tokens total including test runs).

## Migration Notes

None. The change is additive prose; existing plans, changelogs, knowledge entries, and in-flight workflows are unaffected. Repositories pick up the new templates the next time the embedded templates ship in their CLI build — no `init` re-run is required for step templates (they render from the embedded FS at runtime).

## Performance Considerations

None. Two templates grow by a paragraph-scale beat, which marginally lengthens rendered instructions; no runtime code paths change. The only cost worth naming is conversational: the selectivity bar in the prose exists precisely to keep offer frequency near zero on typical runs, per the spec's offer-fatigue risk.
