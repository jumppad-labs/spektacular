# Working Context: implement workflow for 000041_workflow-knowledge-capture-offers

## Session state

- WORKFLOW FINISHED 2026-07-31. All 3 phases complete, harbor green (93/93, zero oracle edits), repo CHANGELOG.md updated, test-plan.md + changelog record committed, spec reconciled (6 requirements [x], 4 acceptance criteria left [ ] pending manual verification per test-plan.md). Outstanding item surfaced to user: stale `goto finished` advance line in the update_feature_changelog template (FSM requires reconcile_spec first).

- Implement workflow started for plan 000041_workflow-knowledge-capture-offers (user picked it explicitly; newest plan, untracked in git). read_plan step complete:
  - Structural validation passed — all 10 sections present, 3 phases (1.1, 2.1, 2.2) with resolving technical-detail links.
  - Drift check clean — every referenced file/symbol/line verified current: 07-update_changelog.md Discoveries at :28 / Step 3 at :44; 18-walkthrough.md assumptions beat :18, apply-immediately :20, first `plan goto` :27; cmd/skill.go confirmed unable to resolve skills/workflows/ (skills/skill_<name>.md only, listSkills no recursion); all named test funcs exist at cited lines; `make harbor-test-plan` target exists at Makefile:73.
  - Spec coverage check passed — all 6 requirements + 4 acceptance criteria map to phases 1.1/2.1; no descoping.
  - Changelog mode: NO `## Changelog` in plan.md → first-phase invocation; analyze picks up at Phase 1.1.
- Advancing to analyze.

## Discovery learnings (beyond what research.md records)

- `{{config.command}} skill spek-knowledge` is UNREACHABLE (cmd/skill.go resolves only templates/skills/skill_<name>.md, no recursion into skills/workflows/) — new prose must name the spek-knowledge skill or use raw `knowledge` CRUD commands.
- No implement-workflow harbor suite exists — 07-update_changelog.md changes have zero harbor coverage; plan-workflow harbor verifier never asserts walkthrough prose content; walkthrough window is exempt from the no-AskUserQuestion rule (test_plan_workflow.py:847).
- Live constraint: INSTRUCTION_NEXT_STEP_RE takes the FIRST `plan goto` match in a rendered instruction — new 18-walkthrough.md prose must not introduce an earlier `plan goto` occurrence.
- Test model to copy: TestDiscoveryStepUsesKnowledgeCommands (internal/steps/plan/steps_test.go:121-132), renderStep + require.Contains, ToLower for prose.
- Pre-existing drift (out of scope): EXPECTED_SKILLS_PER_STEP lists discover-project-commands / discover-test-patterns which no template references.

## Key facts carried forward

- Motivating incident: implement run for plan 000040 surfaced four durable discoveries, zero offers. Root causes: no in-band reinforcement in implement templates; Discoveries slot is a decoy sink; discoveries surface under instruction pressure.
- Standing contract: offer-then-confirm; capture via existing spek-knowledge tooling; no new write path; no new interruption points; per-change record shape unchanged; decline final for conversation (prose-only, no state).
- Selectivity bar: offers must be rare (durable + non-obvious beyond current change); no tunable setting — agent judgement.
- Phase plan: 1.1 = 07-update_changelog.md Step 2b beat + TestUpdateChangelogStepOffersKnowledgeCaptureForDurableDiscoveries (after :311-317); 2.1 = 18-walkthrough.md change-request-path beat + TestWalkthroughStepOffersKnowledgeCaptureForRevealingCorrections (beside :418-437), both with require.NotContains("skill spek-knowledge"); 2.2 = `make harbor-test-plan` confirming run (~25 min, Docker + harbor CLI + Claude creds), expected green with zero oracle edits.

## Phase 1.1 analysis (complete)

- Insertion: `### Step 2b: Assess discoveries for durable knowledge` between the `rm .spektacular/tmp/plan_update.md` block (line 42) and `### Step 3` (line 44) of templates/steps/implement/07-update_changelog.md.
- renderStep harness uses Config{Command: "spektacular"} → `{{config.command}}` renders "spektacular"; template's existing `skill update-changelog` reference cannot false-positive the NotContains("skill spek-knowledge") guard.
- New test after steps_test.go:311-317 (TestUpdateChangelogStepBranchesOnUncheckedPhases); model = TestDiscoveryStepUsesKnowledgeCommands; anchors: "durable", "offer", explicit acceptance/confirm, decline-not-offered-again, "spek-knowledge".

## Phase 1.1 progress

- implement step done: Step 2b beat inserted in 07-update_changelog.md between the plan-file-write block and Step 3. Anchors used in prose: "durable", "beyond this one change", "offer", "most phases produce none", "explicit acceptance", "not offered again", "spek-knowledge". Build green.
- test step done (sub-agent): TestUpdateChangelogStepOffersKnowledgeCaptureForDurableDiscoveries added after the branching test in internal/steps/implement/steps_test.go; 6 positive anchors + NotContains("skill spek-knowledge") guard.
- verify done: go build / go vet / go test ./... all green (no lint target documented; thoughts/notes/commands.md absent).
- update_plan done: Phase 1.1 heading + all 4 acceptance criteria ticked in plan store.
- update_changelog done: ## Changelog section created with Phase 1.1 entry (Deviations: None). Step 2b self-assessment: discovery was change-local, no knowledge offer raised. Looping to analyze for Phase 2.1 (user's standing preference: run simple features without pausing).

## Phase 2.1 analysis (complete)

- Insertion: new paragraph in templates/steps/plan/18-walkthrough.md immediately after the apply-immediately paragraph (line 20), before the sign-off paragraph (line 22). Existing anchors to preserve: "apply it immediately" (:428), "plan file write" (:427), "challenged assumption is a change request" (:470), "explicit affirmative"/"not agreement" (:431-432). No `plan goto` may appear before line 27's; no `{{config.command}} skill <name>` text.
- New test TestWalkthroughStepOffersKnowledgeCaptureForRevealingCorrections beside TestWalkthroughStepIsMandatoryGuidedReview (:418-437); anchors: "what the correction reveals", "general lesson", explicit acceptance, not-offered-again, "spek-knowledge", NotContains("skill spek-knowledge"). Keep anchor vocabulary aligned with Phase 1.1's beat.

## Phase 2.1 progress

- implement step done: assessment paragraph added to 18-walkthrough.md after the apply-immediately paragraph. Build green; single `plan goto` still at file end (line 29).
- test step done (sub-agent): TestWalkthroughStepOffersKnowledgeCaptureForRevealingCorrections added after TestWalkthroughStepWalksDraftingAssumptions; anchors "what the correction reveals", "general lesson", "produce no offer", "explicit acceptance", "not offered again", "spek-knowledge" + NotContains guard.
- verify done: build/vet/test all green; grep confirms exactly 1 `plan goto` in 18-walkthrough.md.
- update_plan done: Phase 2.1 heading + all 4 criteria ticked.
- update_changelog done: Phase 2.1 entry appended (Deviations: None); Step 2b assessment — change-local, no offer. 1 unchecked phase remains (2.2 harbor run). Looping to analyze.

## Phase 2.2 analysis (complete)

- Confirming harbor run: `make harbor-test-plan` (Makefile:73) — builds linux binary into tests/harbor/plan-workflow/environment/, runs harbor CLI w/ claude-code agent, model claude-sonnet-4-6, auth from ~/.claude/.credentials.json. Prereqs verified: harbor 0.3.0 on PATH, Docker daemon up (5.7.0), OAuth token present. No file changes expected; expected green with zero oracle edits. ~20-25 min wall-clock. implement/test steps are no-ops for this phase; the run itself happens at verify.

## Phase 2.2 progress

- Harbor run GREEN: 93 passed, 0 failed, zero oracle edits — research conclusion confirmed. Phase 2.2 ticked (both criteria), changelog entry written. All 3 phases complete.
- update_repo_changelog done: user-facing section prepended to CHANGELOG.md.
- test_plan done: test-plan.md committed to plan store with 4 manual procedures.
- update_feature_changelog done: changelog record committed to changelog store (Deviations: None).
- NOTE (possible durable discovery): the update_feature_changelog template's advance line says `goto finished`, but the FSM only allows `reconcile_spec` from there — the CLI's invalid_transition error self-corrects it, but the template prose is stale. Candidate knowledge/bug offer for the user.
- reconcile_spec done: all 6 spec Requirements flipped [x]; 4 Acceptance Criteria deliberately left [ ] — they describe live-run behavioral outcomes the plan classifies as manual (see test-plan.md). Remaining: finished.

## Decisions / user answers this session

- User selected plan 000041_workflow-knowledge-capture-offers to implement (recommended as newest/untracked).
