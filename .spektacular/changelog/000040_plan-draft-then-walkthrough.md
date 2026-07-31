---
created_date: "2026-07-31"
status: completed
closed_date: "2026-07-31"
---

# Changelog: 000040_plan-draft-then-walkthrough

## What was built

The plan workflow's interaction model changed from "confirm each section as it is gathered" to "draft everything autonomously, then review once, guided, at the end."

A new `walkthrough` step was inserted into the plan workflow's state machine between the last document-write step and the terminal `finished` step. It owns the mandatory guided review: the agent walks the user through the committed plan in natural beats (approach and alternatives, stages, out-of-scope, and the drafting assumptions), applies requested changes immediately through the store write path, and only advances to `finished` — the sole place documents are stamped `status: completed` — after the user answers an explicit closing question affirmatively. Because the workflow sits on a non-terminal step until sign-off, an interrupted review resumes in a later session, and the shared in-progress state blocks `implement new` against an unreviewed plan. The old optional walkthrough offer in the terminal step was removed.

All eleven section-gathering step templates (discovery through out-of-scope) lost their per-section approval gates. Each now drafts its section, saves it to its working file, and advances, carrying two identical shared behaviors: appending every judgement call to a git-tracked assumption log (`.spektacular/work/<plan>/assumptions.md`, one entry per call with decision, rationale, and rejected alternatives) and a proceed-unless-blocked rule — stop only for a decision with no reasonable default or information only the user holds. The architecture step now picks the best-grounded design direction itself and records it as the leading assumption instead of asking the user to choose. At assembly, the log is folded into a new `## Drafting assumptions` section of research.md (verified as a required section), which the walkthrough raises entry by entry for challenge.

The spek-plan skill was updated to teach the new loop (draft straight through, mandatory sign-off review, assumption log in the working-file contract, walkthrough-aware resume). Enforcement lives in the extended Go template/step test suites (step order and wiring, banned approval phrasings, shared-block anchors, scaffold shape) and in the harbor end-to-end suite, which now asserts zero user-directed confirmation questions during drafting, a populated drafting-assumptions section, walkthrough-before-finished ordering, and the in-progress-until-sign-off status lifecycle. The full E2E suite passes 93/93.

## Why it matters

Creating a plan previously interrupted the user roughly ten times with serial section-approval prompts. The new flow mirrors how a team actually works: a planner produces a complete proposal and reviews it with stakeholders once, with every judgement call on the table. The user is interrupted only by genuinely blocking questions (zero for a typical spec) yet keeps real review power — assumptions are first-class review material, comments revise the saved documents immediately, and nothing is treated as approved without an explicit yes. The final documents keep their exact shape, names, and location, so the downstream implement workflow is unaffected.

## Deviations from the plan

- Phases 1.1–2.2: none.
- Phase 2.3: the plan's open question about detecting confirmation questions resolved favorably — `AskUserQuestion` tool calls proved a reliable transcript signal, so the assertion was not narrowed (prose-question detection was deliberately not attempted). Four pre-existing harness/product drifts surfaced by the E2E runs were fixed beyond the planned file list: the harbor agent timeout was raised 900→1800s (the old budget predated the extra step), the seeded spec was renamed `20260101000000-user-auth.md` to satisfy the store's ID-prefix contract, the conventions-read oracle was updated from `knowledge conventions` to `knowledge always-applied`, and an over-generic scaffold-leftover literal was replaced. One template hardening rode along: the walkthrough now mandates reading all three committed documents at its start (rather than "as you need them"), grounding resumed reviews in committed state.
