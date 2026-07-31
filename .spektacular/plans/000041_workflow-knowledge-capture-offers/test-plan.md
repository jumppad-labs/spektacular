---
created_date: "2026-07-31"
status: completed
closed_date: "2026-07-31"
---

# Test Plan: 000041_workflow-knowledge-capture-offers

The presence and wording of both assessment-and-offer directions is covered by automated template-content tests (`TestUpdateChangelogStepOffersKnowledgeCaptureForDurableDiscoveries` in `internal/steps/implement/steps_test.go`, `TestWalkthroughStepOffersKnowledgeCaptureForRevealingCorrections` in `internal/steps/plan/steps_test.go`), and the plan-workflow harbor suite confirmed the changed templates disturb no E2E oracle (93 passed, zero oracle edits). The metrics below are behavioral outcomes of live agent runs and require manual observation.

## 1. Durable discoveries raise an offer without user prompting

- **What to measure**: In an implement run whose phase surfaces a durable, non-obvious discovery, the agent's conversation contains a knowledge-capture offer — naming what would be captured and why — before the workflow finishes, without the user having asked.
- **How**: Run a real implement workflow (`go run . implement new --data '{"name":"<plan>"}'`) on a plan whose phase is known to surface a durable discovery (e.g. one touching an undocumented coupling). Watch the `update_changelog` step's conversation after the phase's `**Discoveries**` entry is written.
- **Expected result**: An offer appears in the same message as the continue-or-pause prompt, phrased as the general lesson. No `knowledge write` occurs before explicit acceptance.
- **Who / when**: The maintainer, during the first few real implement runs after this feature ships.

## 2. Accepted plan-correction offers produce knowledge entries; subsequent plans stop repeating the corrected assumption

- **What to measure**: (a) Accepting a walkthrough-correction offer results in a matching entry in the knowledge store; (b) a later plan run does not repeat the corrected drafting assumption.
- **How**: During a plan walkthrough, issue a correction that reveals a wrong drafting default (e.g. challenge a recorded drafting assumption). Accept the resulting offer and let the `spek-knowledge` flow complete. Verify the entry with `go run . knowledge read --data '{"scope":"project","path":"<category>/<entry>.md"}'`. Then run a new plan touching the same surface and check its `## Conventions`/drafting output.
- **Expected result**: An entry exists after acceptance (and none after a decline); the next plan's draft reflects the captured lesson instead of repeating the old default.
- **Who / when**: The maintainer, on the next plan walkthrough that produces a genuine correction.

## 3. Offers stay rare enough to be signal

- **What to measure**: A typical run whose discoveries and corrections are all change-local produces zero knowledge-capture offers; users do not experience offer fatigue.
- **How**: Over the next several routine implement and plan runs (aim for at least five), count knowledge-capture offers raised per run and note whether each offered item was genuinely durable.
- **Expected result**: Runs with nothing durable produce zero offers — the workflow reads exactly as before this feature; when offers do appear, they are judged worth raising (no more than ~one per run in practice).
- **Who / when**: The maintainer, passively across normal usage in the weeks after shipping.

## 4. The knowledge base grows from workflow output rather than only ad-hoc requests

- **What to measure**: Over time, knowledge entries originate from workflow-raised offers, not solely from the user asking after the fact.
- **How**: Periodically review the knowledge store's git history (`git log --oneline -- .spektacular/knowledge/`) and note which entries trace to an in-run offer versus an ad-hoc user request.
- **Expected result**: A meaningful share of new entries (any nonzero number is progress over the motivating incident's zero) originate from workflow offers.
- **Who / when**: The maintainer, reviewed casually after a month of normal usage.
