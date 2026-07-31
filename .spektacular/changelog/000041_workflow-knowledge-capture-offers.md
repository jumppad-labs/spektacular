---
created_date: "2026-07-31"
status: completed
closed_date: "2026-07-31"
---

# Changelog: 000041_workflow-knowledge-capture-offers

## What was built

The implement and plan workflows now recognize the two moments when durable project knowledge most often surfaces, and offer — in that moment, inside a conversation the workflow was already having — to save the insight to the project's knowledge base.

- **Implement workflow**: the per-phase changelog step (`templates/steps/implement/07-update_changelog.md`) gained a "Step 2b: Assess discoveries for durable knowledge" beat. After a phase's `**Discoveries**` entry is written exactly as before, the agent weighs each discovery for durability — whether it holds beyond the current change (an undocumented coupling, a surprising contract, a convention inferred from behavior, a corrected wrong default). Anything that clears that bar is offered to the user in the same message as the existing continue-or-pause prompt, naming what would be captured and why. Most phases produce no offer.
- **Plan workflow**: the walkthrough step (`templates/steps/plan/18-walkthrough.md`) gained the mirror beat on its apply-immediately change-request path. After a correction — including a challenged drafting assumption — is applied and confirmed, the agent considers what the correction reveals (a wrong drafting default, a constraint the user knew that the codebase never recorded, a misunderstood term) and, for a durable generalizable lesson, offers to capture the general lesson rather than the specific edit before resuming the walkthrough.

Both beats share one contract: capture happens only on the user's explicit acceptance, routed to the existing `spek-knowledge` skill whose propose-then-confirm flow owns scope selection, category routing, and the write; a decline is final for that item for the rest of the conversation; a deferral may be re-raised. The per-change records (the phase Discoveries entry, the applied plan edit) keep their exact shape and are always written before any assessment. The prose deliberately never directs the CLI invocation `skill spek-knowledge`, which cannot resolve workflow skills.

Enforcement is by rendered-template content tests: `TestUpdateChangelogStepOffersKnowledgeCaptureForDurableDiscoveries` (`internal/steps/implement/steps_test.go`) and `TestWalkthroughStepOffersKnowledgeCaptureForRevealingCorrections` (`internal/steps/plan/steps_test.go`) pin the durability assessment, the offer, the selectivity bar, the explicit-acceptance gate, decline-finality, and the `spek-knowledge` hand-off — and each guards against the unreachable `skill spek-knowledge` invocation. Removing either direction fails `go test ./...`.

The change is prose-only: no Go code, no new FSM steps, no new template variables, no new write path. The plan-workflow harbor E2E suite was run once as confirmation and passed all 93 tests with zero oracle modifications.

## Why it matters / what it enables

Knowledge capture previously relied on a standing background instruction that demonstrably failed during long working sessions — the motivating incident was an implement run that surfaced four durable discoveries and produced zero offers, leaving the insights buried in a per-change changelog that future work never reads. With the recognition moved in-band to the two moments where durable knowledge actually surfaces, an insight is captured once, when it appears, and the next plan or implementation stops paying to re-discover it. Runs with nothing durable to capture read exactly as they did before, and the knowledge base can now grow from real workflow output rather than only ad-hoc user requests.

## Deviations from the plan

None. All three phases landed as planned, and the harbor confirmation run validated the research conclusion that no E2E oracle needed editing.
