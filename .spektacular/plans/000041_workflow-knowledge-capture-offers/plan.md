---
created_date: "2026-07-31"
status: completed
closed_date: "2026-07-31"
---

# Plan: 000041_workflow-knowledge-capture-offers

<!-- Metadata -->
<!-- Created: 2026-07-31T10:42:22Z -->
<!-- Commit: 5c37a68 -->
<!-- Branch: f-improve-plan -->
<!-- Repository: git@github.com:jumppad-labs/spektacular.git -->

## Overview

The implement and plan workflows will recognize the two moments when durable project knowledge most often surfaces — a non-obvious discovery made while implementing a phase, and a correction the user makes while reviewing a drafted plan — and offer, in that moment, to save the insight to the project's knowledge base. Today that recognition relies on a standing background instruction that demonstrably fails during long working sessions, so durable insights end up buried in per-change records that future work never reads. Everyone who runs these workflows benefits: an insight captured once, when it appears, saves the next plan or implementation from re-discovering it, while runs with nothing durable to capture read exactly as they do today.

## Conventions

No project conventions apply to this feature. The always-applied knowledge loaded during discovery (`go run . knowledge always-applied`) returned only the conventions and glossary category README placeholders — the knowledge base holds no populated project-specific conventions — and the targeted knowledge searches for the surfaces this feature touches (workflow templates, knowledge tooling, testing) surfaced architecture and gotcha entries but no binding conventions.

## Architecture & Design Decisions

The feature ships entirely as instruction-prose additions to two existing step templates, with template-content tests as the enforcement layer — no Go changes, no new FSM steps, no new template variables, and no new write path. This is the narrowest of the three shapes considered, and the one the spec's own technical approach points at: in this codebase workflow judgement lives in step-template prose, and the repeated in-band directive is the mechanism with a demonstrated track record (the context-refresh directive at the foot of every template), while the standing repository-level trigger alone is the mechanism with a demonstrated failure (four durable discoveries, zero offers, in the motivating implement run).

Both offer points reuse a single shared behavioral shape, expressed in each template's own words: after the workflow records the item in its per-change home, assess it for durability — does it hold beyond this change (an undocumented coupling, a surprising contract, a convention inferred from behavior, a wrong default the user had to correct)? If nothing clears that bar, say nothing and continue exactly as today; the selectivity bar (durable *and* non-obvious beyond the current change) is stated in the prose itself so typical runs produce zero offers. If an item clears it, offer once — naming what would be captured and why it is worth keeping, phrased as the general lesson rather than the specific edit — and route an acceptance to the `spek-knowledge` skill, whose existing propose-then-confirm flow owns scope selection, category routing, and the actual write. A decline is final for that item for the rest of the conversation; a deferral may be re-raised. The per-change records (the phase's `**Discoveries**` entry, the applied plan edit) are written exactly as today before any assessment happens, keeping the offer strictly additive.

The two attachment points are moments where each workflow already converses with the user, so no new interruption point is created. In the implement workflow's `update_changelog` template, the assessment beat sits between writing the phase's `**Discoveries**` entry and the existing continue-or-pause prompt. In the plan workflow's `walkthrough` template, it attaches to the apply-immediately change-request path — after the correction is committed and confirmed, before the walkthrough resumes — which also covers challenged drafting assumptions for free, since the template already routes those through the same path. One deliberate mechanical choice: the prose hands off to the `spek-knowledge` skill by name (or the raw `knowledge` CRUD commands), never via the CLI's `skill` subcommand, because that subcommand cannot resolve workflow skills — a runtime error no test would catch.

Enforcement follows the established pattern for prose-driven behavior: one new rendered-template content test per offer point, modeled on the existing discovery-step knowledge test, asserting stable anchor phrases (the durability assessment, the offer-then-confirm contract, the decline-is-final rule) so removing the direction fails `go test ./...`. This direction beats the two rejected shapes — a dedicated knowledge-review FSM step (adds an interruption point the spec forbids, plus step-table and harbor step-order churn) and further strengthening the background trigger or skill files (the surface that already failed, and one no content test guards) — see `research.md#alternatives-considered-and-rejected` for the evidence. Harbor exposure is bounded and verified rather than assumed: no implement-workflow harbor suite exists, no plan-workflow oracle asserts walkthrough prose content, and the one live constraint (the verifier reads the first `plan goto` occurrence in a rendered instruction) is honored by adding no `plan goto` text ahead of the existing one.

## Component Breakdown

- **Implement changelog step instructions (changed)** — the implement workflow's per-phase changelog step. After writing the phase's Discoveries entry exactly as today, it gains an assessment beat: weigh each recorded discovery for durability beyond the current change, and when one clears the bar, offer — in the same conversational moment as the existing continue-or-pause prompt — to save it to the knowledge base, naming what would be captured and why. Declined items are not re-offered for the rest of the conversation; accepted items are handed to the knowledge-capture skill. When nothing clears the bar the step behaves byte-for-byte as it does today.

- **Plan walkthrough step instructions (changed)** — the plan workflow's mandatory review step. Its apply-immediately change-request path gains the mirror assessment beat: after a requested correction (including a challenged drafting assumption) is committed and confirmed, weigh what the correction reveals — a wrong drafting default, an unrecorded constraint or convention, a misunderstood term — and when it reveals something durable, offer in the same review conversation to capture the general lesson, not the specific edit. Decline/defer semantics match the changelog step's.

- **Shared offer contract (new, prose convention)** — the common behavioral shape both templates express in their own words: assess for durability with a deliberately high bar (durable and non-obvious beyond the current change, so typical runs produce zero offers), offer at most once per item with what-and-why, treat a decline as final for the conversation, and never write without explicit acceptance. Not a shared file or mechanism — a convention kept consistent across the two templates and pinned by their content tests.

- **Knowledge-capture skill (unchanged, reused)** — the existing skill that owns the actual write: scope selection, category routing, staging, and its own propose-then-confirm gate. Both offer points hand accepted items to it by name; this feature adds recognition and offer moments, not a new write path. The templates deliberately do not invoke it through the CLI's `skill` subcommand, which cannot resolve workflow skills.

- **Template-content tests (changed)** — the enforcement layer for prose-driven behavior. The implement and plan step test suites each gain one rendered-instruction test asserting the new beats' stable anchor phrases (durability assessment, offer-then-confirm, decline-is-final), modeled on the existing discovery-step knowledge test, so removing either direction fails the standard Go test run.

- **Per-change records (unchanged, relied upon)** — the phase Discoveries entry and the applied plan-document edit keep their exact shape and are always written before any assessment happens; the knowledge offer is strictly additive on top of them.

## Data Structures & Interfaces

No new Go types, interfaces, template variables, or serialization formats are introduced, and no existing ones change shape. The feature is prose-only; every contract it relies on already exists and is reused unchanged:

**Template render bundles (unchanged)** — both edited templates keep their existing variable sets. The implement changelog template continues to receive the base step variables plus the implement path variables (`plan_name`, `plan_path`, `changelog_section_name`, `config.command`, …) with no `Extra` payload; the plan walkthrough template continues to receive the base variables plus the plan path variables. The new prose spells CLI commands with the existing `{{config.command}}` variable and introduces no new placeholder, so the Go render pipeline and both step callbacks are untouched.

**Knowledge CRUD surface (unchanged, reused)** — accepted offers flow through the existing `knowledge sources` / `knowledge categories` / `knowledge write --data '{"scope":…,"path":…}' --file <staged>` command shapes exactly as the knowledge-capture skill already invokes them. No new command, flag, or payload field.

**Per-change record shapes (unchanged)** — the phase changelog entry keeps its exact field set (`What was done`, `Deviations`, `Files changed`, `Discoveries`), and plan-document corrections keep flowing through the existing `plan file read` / `plan file write --from` surface. The spec constraint that these records not change is satisfied structurally: the offer prose only reads what was just recorded.

**Offer-beat prose contract (new, informal)** — the only new "interface" is a convention between the two templates and their content tests: each offer beat must contain stable anchor phrasing for the durability assessment, the offer-then-confirm rule, and decline-finality. This contract is pinned by string assertions in the step test suites, not by any runtime type.

## Implementation Detail

This change extends the codebase's established division of labour — Go owns step sequencing and document mechanics, instruction prose owns judgement and conversation — without adding anything on the Go side. A developer reading the diff will see two template files gain a self-contained "assess and offer" beat, two test files gain a matching content test each, and nothing else change.

**The offer beat as a written pattern.** Each beat follows the same internal structure, adapted to its host template's voice: first a framing sentence anchoring it to the record just written (the phase's discoveries; the correction just applied), then the durability test stated as concrete examples of what clears the bar and an explicit instruction that most items do not, then the offer itself (name the insight, say why it outlasts this change, phrase a correction as the general lesson), then the outcome rules — acceptance hands off to the knowledge-capture skill whose own propose-then-confirm flow governs the write; a decline is final for that item for the rest of the conversation; silence or deflection is not acceptance. The beat ends by returning the agent to the host template's existing flow (the continue-or-pause prompt; resuming the walkthrough), reinforcing that the offer is a parenthesis inside an existing conversation, not a new stopping point.

**Following existing patterns rather than inventing them.** The propose-then-confirm phrasing mirrors the plan discovery step's optional learning-capture beat — the one place in-band knowledge prose already exists — so agents encounter one consistent vocabulary. The decline/defer/accept outcome model reuses the repository-level knowledge trigger's wording. The enforcement tests copy the rendered-instruction assertion style of the existing step content tests (render the step through the real template path, assert stable lowercase anchor substrings), placed alongside the host steps' existing tests. Anchor phrases are chosen to be distinctive enough not to false-positive on unrelated prose, per the house convention for content-test markers.

**One new written convention.** Because the same behavioral contract now lives in two templates, the plan deliberately keeps the two beats' anchor phrasing aligned (same key phrases for the durability bar, the confirm gate, and decline-finality) so the two content tests can assert a consistent contract. This is the only novelty: a cross-template prose convention, maintained editorially and pinned by tests, not extracted into a shared include — the template system has no include mechanism and adding one for two call sites would be machinery the feature does not need.

## Dependencies

- **spek-knowledge skill (`templates/skills/workflows/spek-knowledge/SKILL.md`)** — provides the entire capture path an accepted offer hands off to: scope selection, category routing, staged write, and the propose-then-confirm gate. No changes needed; the feature depends on it existing and behaving as documented.
- **`knowledge` CLI surface (`sources`, `categories`, `write`, backed by `internal/`)** — the write mechanics beneath the skill. No changes needed.
- **Plan 000040 (plan-draft-then-walkthrough) — already landed** — shipped the walkthrough step and its apply-immediately change-request path, which is the attachment point for the plan-review offer. Without it there is no correction moment to hook; it is on `main`'s history (branch `f-improve-plan`), so no sequencing risk remains.
- **Implement changelog template (`templates/steps/implement/07-update_changelog.md`) and plan walkthrough template (`templates/steps/plan/18-walkthrough.md`)** — the two host surfaces being edited; both stable, no upstream changes pending against them.
- **Step test harnesses (`renderStep` helpers in `internal/steps/implement/steps_test.go` and `internal/steps/plan/steps_test.go`)** — provide the rendered-instruction fixture the new content tests use. No changes needed beyond adding test functions.
- **Mustache template render pipeline (`internal/stepkit`, `templates` embed.FS)** — renders the edited templates; the new prose uses only existing variables, so no changes needed.
- **No external libraries** — no new imports, services, or third-party dependencies of any kind.

## Testing Approach

Testing follows the project's three-layer architecture for prose-driven behavior, with the weight on the template-contract layer — the layer whose whole purpose is making instruction prose regression-testable.

**Template-contract tests (primary coverage).** One new rendered-instruction content test per offer point, in the implement and plan step test suites respectively, following the existing pattern of rendering the step through the real template path and asserting stable anchor substrings. The load-bearing guarantees, in plain language: the changelog step's instructions direct the agent to assess recorded discoveries for durability and to offer — not silently write — a knowledge capture; the walkthrough step's instructions direct the same assessment for applied corrections, phrased as the general lesson; both contain the offer-then-confirm gate and the decline-is-final rule. Removing or weakening either beat fails the standard Go test run in CI. Existing content tests on the two host templates (the Discoveries field assertions, the walkthrough's mandatory-review and apply-immediately assertions) double as regression guards that the new prose did not displace the behavior it sits inside.

**Go unit/step tests (no changes expected, run as regression).** No step wiring, callbacks, or render bundles change, so the existing suites simply must stay green.

**Harbor E2E (confirming run, deliberate gap acknowledged).** The plan-workflow harbor suite is run once to confirm no oracle drift, since a workflow template changed — the analysis says no oracle needs editing, and the run verifies that claim rather than assuming it. Deliberate gaps: the harbor reference solution is a scripted happy path that never issues a plan correction, so the walkthrough offer is not behaviorally exercised end-to-end, and no implement-workflow harbor suite exists at all. Building correction-exercising or implement-workflow harbor scenarios would dwarf the feature and is not part of this plan; the offer beats' behavioral proof therefore rests on the content tests plus real-use observation.

**Success metrics from the spec, each carried here:**

- *Durable discoveries raise an offer without user prompting* — the presence and wording of the direction is a behavioural test (the changelog-step content test); whether real runs actually produce the offer at the right moments is **manual — captured in the implementation test plan**.
- *Accepted plan-correction offers produce knowledge entries, and subsequent plans stop repeating the corrected assumption* — the offer direction and its hand-off to the confirm-gated capture path is a behavioural test (the walkthrough-step content test); entry creation on acceptance and the downstream stop-repeating effect are **manual — captured in the implementation test plan**.
- *Offers stay rare enough to be signal; a run with no durable discoveries produces zero offers* — the selectivity bar's presence in both templates is a behavioural test; actual offer frequency and the absence of fatigue in typical runs is **manual — captured in the implementation test plan**.
- *The knowledge base grows from workflow output rather than only ad-hoc requests* — observational by nature; **manual — captured in the implementation test plan**.

## Milestones & Phases

### Milestone 1: Implement runs offer to capture durable discoveries

**What changes**: When an implement run records what a completed phase discovered, it now weighs each discovery for durability — whether it matters beyond this one change — and, for anything that clears the bar, offers the user right then to save it to the project knowledge base, saying what it would capture and why. Change-local discoveries produce no offer and the run reads exactly as today; a decline is respected for the rest of the conversation; an acceptance goes through the existing confirm-gated knowledge tooling. The motivating failure — durable insights silently buried in a phase changelog nobody re-reads — stops happening for implement runs.

#### - [x] Phase 1.1: Discovery assessment and offer in the implement changelog step

The implement workflow's per-phase changelog instructions gain an assessment beat: after the phase's discoveries are recorded exactly as today, the agent weighs each one for durability beyond the current change and, for anything that clears the bar, offers the user — in the same breath as the existing continue-or-pause conversation — to save it to the knowledge base, naming what would be captured and why. The beat states the selectivity bar, the offer-then-confirm gate, and that a decline is final for the conversation. A matching content test pins the new direction so removing it fails the test suite.

*Technical detail:* [context.md#phase-11](./context.md#phase-11-discovery-assessment-and-offer-in-the-implement-changelog-step)

**Acceptance criteria**:

- [x] After a phase's Discoveries entry is written, the rendered changelog-step instructions direct the agent to assess those discoveries for durability and to offer a knowledge-base capture for durable ones, naming what would be captured and why
- [x] The instructions state that change-local discoveries produce no offer, that capture happens only on explicit acceptance via the existing knowledge tooling, and that a declined item is not offered again in the conversation
- [x] The per-phase changelog entry shape and the step's existing continue-or-pause and goto behavior are unchanged
- [x] A content test fails if the assessment-and-offer direction is removed from the template

### Milestone 2: Plan reviews offer to capture what corrections reveal

**What changes**: When a user corrects a drafted plan during the walkthrough — requesting a change or challenging a recorded assumption — the workflow, after applying the correction as it already does, now considers what the correction reveals: a wrong default the drafting agent chose, a constraint the user knew that the codebase never recorded, a misunderstood term. When there is a durable lesson, it offers in the same review conversation to capture the general lesson (not the specific edit) in the knowledge base, under the same offer-then-confirm and decline-is-final rules. Reviews with only cosmetic or change-local corrections read exactly as today.

#### - [x] Phase 2.1: Correction assessment and offer in the plan walkthrough step

The plan walkthrough's apply-immediately change-request path gains the mirror beat: once a requested correction (including a challenged drafting assumption) is committed and confirmed, the agent considers what the correction reveals — a wrong drafting default, an unrecorded constraint or convention, a misunderstood term — and, when there is a durable lesson, offers in the same review conversation to capture it phrased as the general lesson rather than the specific edit. The same selectivity bar, confirm gate, and decline-is-final rules apply, and a matching content test pins the direction.

*Technical detail:* [context.md#phase-21](./context.md#phase-21-correction-assessment-and-offer-in-the-plan-walkthrough-step)

**Acceptance criteria**:

- [x] After a walkthrough correction is applied and confirmed, the rendered walkthrough instructions direct the agent to assess what the correction reveals and to offer capturing the underlying lesson — phrased generally, not as the specific edit — before resuming the walkthrough
- [x] The instructions carry the offer-then-confirm gate and the decline-is-final rule, and corrections revealing nothing durable produce no offer
- [x] The walkthrough's existing mandatory-review, apply-immediately, and sign-off behavior is unchanged
- [x] A content test fails if the correction-assessment direction is removed from the template

#### - [x] Phase 2.2: End-to-end confirmation that the workflow surface is undisturbed

With both prose beats and their tests in place, the plan-workflow end-to-end harbor suite is run once to confirm the template changes disturbed none of its hand-maintained oracles — the research concluded no oracle needs editing, and this run verifies that conclusion instead of assuming it. This phase changes no code; it is the verification the project's testing architecture prescribes for any workflow-template change.

*Technical detail:* [context.md#phase-22](./context.md#phase-22-end-to-end-confirmation-that-the-workflow-surface-is-undisturbed)

**Acceptance criteria**:

- [x] The plan-workflow harbor suite passes against the changed templates with no oracle modifications
- [x] Any failure is resolved by correcting the new prose (or, if genuinely required, a deliberate oracle update recorded in the changelog), not by skipping the run

## Open Questions

One genuinely implementation-time uncertainty remains:

- **Whether the confirming harbor run passes with zero oracle edits.** The research concluded the prose changes disturb no plan-workflow oracle, but the harbor suite runs a live agent against the real templates and does not run in CI, so the conclusion can only be proven by executing the run in Phase 2.2. If it fails: first assume the new prose caused the drift and adjust the prose; only if the prose is correct and an oracle's expectation is genuinely what changed, update the oracle deliberately and record it — and if neither resolves cleanly, STOP and ask the user.

Everything else that could have been a question was resolved during planning and recorded as a drafting assumption (offer routing that avoids the unreachable `skill spek-knowledge` invocation, prose-only decline-finality, scope limited to the two templates).

## Out of Scope

- **The spec workflow's interaction points** — corrections made while gathering or reviewing a specification are not assessed for knowledge-worthiness; the spec names this a possible follow-up (spec 000041 § Non-Goals). No follow-up spec exists yet.
- **Correction moments outside the plan walkthrough** — user amendments made through the implement workflow's plan-mismatch gates (read_plan drift checks, STOP-on-mismatch) get no assessment beat (spec 000041 § Non-Goals).
- **Retroactive mining** — existing per-change changelogs, past plans, and historical records are not scanned for knowledge candidates (spec 000041 § Non-Goals).
- **A tunable sensitivity setting** for what counts as durable or offer-worthy — the assessing agent uses judgement, consistent with the workflows' existing thresholds (spec 000041 § Non-Goals).
- **Changes to the knowledge tooling itself** — the `spek-knowledge` skill, the `knowledge` CLI commands, and their scope/category routing are reused untouched; this plan adds recognition and offer points only.
- **Making `spek-knowledge` reachable via the CLI `skill` subcommand** — the subcommand's inability to resolve workflow skills is a pre-existing limitation this plan routes around, not fixes.
- **New harbor scenarios** — no correction-exercising plan-workflow scenario and no implement-workflow harbor suite are built; the existing scripted suite is only run for confirmation. Behavioral E2E proof of the offers is deliberately left to real use.
- **Fixing pre-existing harbor oracle staleness** — `EXPECTED_SKILLS_PER_STEP` lists two skills (`discover-project-commands`, `discover-test-patterns`) no template references; that drift predates this feature and is left as found.
- **Edits to the workflow driver skill files** (`spek-implement`/`spek-plan` SKILL.md) — the offer behavior lives entirely in the two step templates, per the recorded drafting assumption.

## Changelog

### 2026-07-31 — Phase 1.1: Discovery assessment and offer in the implement changelog step

**What was done**: Added a `### Step 2b: Assess discoveries for durable knowledge` beat to the implement workflow's per-phase changelog template, between writing the Discoveries entry and the continue-or-pause prompt — it states the durability bar (most phases produce none), the offer-then-explicit-acceptance gate routed to the `spek-knowledge` skill, and decline-is-final-for-the-conversation. Added a content test pinning those anchors so removing the direction fails `go test ./...`.

**Deviations**: None.

**Files changed**:
- `templates/steps/implement/07-update_changelog.md`
- `internal/steps/implement/steps_test.go`

**Discoveries**: The `renderStep` test harness renders with `Config{Command: "spektacular"}`, so the template's legitimate `skill update-changelog` reference cannot false-positive the new `NotContains("skill spek-knowledge")` guard — no anchor collision to work around. Otherwise the phase matched the plan exactly.

### 2026-07-31 — Phase 2.1: Correction assessment and offer in the plan walkthrough step

**What was done**: Added the mirror assessment paragraph to the plan walkthrough template, between the apply-immediately change-request path and the sign-off gate — after a correction (including a challenged drafting assumption) is applied and confirmed, the agent weighs what it reveals and, for a durable generalizable lesson, offers to capture the general lesson (not the specific edit) via the `spek-knowledge` skill under the same explicit-acceptance and decline-is-final rules. Added a content test pinning the beat's anchors.

**Deviations**: None.

**Files changed**:
- `templates/steps/plan/18-walkthrough.md`
- `internal/steps/plan/steps_test.go`

**Discoveries**: The template still contains exactly one `plan goto` occurrence (verified by grep), honoring the harbor verifier's first-match constraint noted in the plan. Nothing else surfaced beyond what the plan already recorded.

### 2026-07-31 — Phase 2.2: End-to-end confirmation that the workflow surface is undisturbed

**What was done**: Ran the plan-workflow harbor E2E suite (`make harbor-test-plan`) against the two changed templates. All 93 verifier tests passed with zero oracle modifications, confirming the research conclusion that the added prose disturbs no hand-maintained oracle — including the first-match `plan goto` constraint and the walkthrough's exemption from the no-confirmation-questions rule.

**Deviations**: None.

**Files changed**: None — this phase was a verification run only.

**Discoveries**: None beyond what the plan predicted; the run confirmed the zero-oracle-edit expectation exactly.
