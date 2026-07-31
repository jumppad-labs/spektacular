---
created_date: "2026-07-31"
status: completed
closed_date: "2026-07-31"
---

# Research: 000041_workflow-knowledge-capture-offers

## Alternatives considered and rejected

### Option A: Rely on the standing repository-level trigger alone

Keep knowledge-capture recognition where it is today — `templates/agents/knowledge-trigger.md`, injected into AGENTS.md by `init`.

**Rejected**: on direct evidence — the motivating incident for this spec was an implement run (plan 000040) that surfaced four durable discoveries and produced zero offers, with the trigger prose active the whole time. Background prose loses salience against the per-step imperative CLI instructions the agent is executing; the mechanism that demonstrably works in this repo is in-band, per-step prose (cf. the context-refresh directive repeated at the foot of every step template, enforced by `templates/context_directive_test.go:57`).

### Option B: A new FSM step (e.g. `knowledge_review`)

Insert a dedicated knowledge-review step after `update_changelog` and/or before the plan workflow's `finished`.

**Rejected**: the spec constraint forbids new interruption points — offers may only surface where the workflow already converses with the user. A new step would also churn `internal/steps/{implement,plan}/steps.go` step tables and the harbor `EXPECTED_STEP_ORDER` oracle (`tests/harbor/plan-workflow/tests/test_plan_workflow.py:61-82`) for no benefit: both offer moments already sit inside existing conversational steps.

### Option C: Referencing the capture skill as `{{config.command}} skill spek-knowledge`

Have the new prose direct the agent to fetch the capture playbook through the CLI skill subcommand, like other step templates do for helper skills.

**Rejected**: unreachable. `cmd/skill.go:49-57` resolves `skill <name>` only to `templates/skills/skill_<name>.md`, and `listSkills()` (`cmd/skill.go:68-81`) does not recurse into `skills/workflows/`, where `spek-knowledge/SKILL.md` lives. The call would error "unknown skill", and no Go test catches a dangling skill reference in a template. New prose must instead name the `spek-knowledge` skill (agent-harness slash-command path, matching `templates/agents/knowledge-trigger.md`) or use the raw `{{config.command}} knowledge sources/write` CRUD surface (matching `templates/steps/plan/02-discovery.md:47`).

### Option D: Detecting durable discoveries in Go code

Have the Go layer flag durable discoveries and emit the offer mechanically.

**Rejected**: durability assessment is judgement, and in this codebase workflow judgement lives in step-template prose, not Go (see knowledge entry `architecture/testing-architecture.md` — "Spektacular's behavior is prose-driven"). There is no signal available to Go at `update_changelog` time that distinguishes a durable discovery from a change-local one.

## Chosen approach — evidence

Prose additions at exactly the two moments the spec names, enforced by rendered-template content tests:

- **Implement offer point**: `templates/steps/implement/07-update_changelog.md` — Step 2 writes the per-phase `**Discoveries**` entry (line 28); Step 3 (line 44) is already a conversational moment (continue/pause prompt). The assessment-and-offer beat slots between writing the Discoveries entry and the continue prompt, adding no new interruption point.
- **Plan-review offer point**: `templates/steps/plan/18-walkthrough.md:20` — the apply-immediately change-request path ("If the interruption is a change request, apply it immediately … confirm the update to the user … then resume the walkthrough"). The assessment beat attaches after "confirm the update", before "resume the walkthrough". A challenged assumption is already routed through this same path (line 18), so it is covered for free.
- **Prose pattern to copy**: `templates/steps/plan/02-discovery.md:45-47` (Step 5, "Capturing a learning (optional)") — the propose-then-confirm anchors: "propose to the user a target scope and the exact content you intend to write, and wait for explicit confirmation", "never write to a knowledge source unprompted". Vocabulary for outcomes: `templates/agents/knowledge-trigger.md` Accept/Defer/Decline model, decline-is-final-for-the-conversation.
- **Enforcement pattern to copy**: `internal/steps/plan/steps_test.go:121-132` (`TestDiscoveryStepUsesKnowledgeCommands`) — `renderStep(t, cb)` + `require.Contains` on the rendered instruction, case-insensitive prose via `strings.ToLower`. Natural homes: new test funcs beside `internal/steps/implement/steps_test.go:292-317` (existing `update_changelog` assertions, incl. the `Discoveries` field at :292-302) and `internal/steps/plan/steps_test.go:418-437` (`TestWalkthroughStepIsMandatoryGuidedReview`, which already asserts the "apply it immediately" anchor at :428).
- **Render bundles**: neither template gets knowledge-related variables — `07-update_changelog.md` receives base vars + implement `strategy.PathVars` (`internal/steps/implement/strategy.go:47-63`; `changelog_section_name` hardcoded `"## Changelog"` at :59, `Extra` nil per `internal/steps/implement/steps.go:114-118`); `18-walkthrough.md` receives base + plan PathVars. Command names in new prose are spelled with `{{config.command}}` exactly as the precedents do.
- **Harbor impact is minimal and bounded**: there is **no implement-workflow harbor suite** (only `tests/harbor/plan-workflow` and `tests/harbor/spec-workflow`), so the `07-update_changelog.md` change has zero harbor coverage. No plan-workflow verifier asserts on the rendered *content* of `18-walkthrough.md`; `test_no_confirmation_questions_during_drafting` (`test_plan_workflow.py:831-857`) explicitly exempts the `walkthrough` window (:847), so an offer there cannot trip it. Two live constraints: `INSTRUCTION_NEXT_STEP_RE` (`test_plan_workflow.py:187-190`, consumed at :1101-1148) takes the *first* `plan goto` match in a rendered instruction — new walkthrough prose must not introduce a `plan goto` occurrence earlier than the existing one; and `EXPECTED_SKILLS_PER_STEP` (:88-103) would need a `walkthrough:` entry only if the prose directs `{{config.command}} skill <name>` — avoided by not using that surface (see Option C).

## Files examined

- `templates/steps/implement/07-update_changelog.md:28,44` — Discoveries entry shape; Step 3 continue/pause prompt is the existing conversational moment.
- `templates/steps/plan/18-walkthrough.md:18-20` — assumptions beat routes challenges into the change-request path; apply-immediately sequence ends "resume the walkthrough" — attachment point for the offer.
- `templates/steps/plan/02-discovery.md:45-60` — Step 5 optional learning capture; propose-then-confirm anchor phrases; judgement-call log instructions.
- `templates/agents/knowledge-trigger.md` — recognition/offer vocabulary, Accept/Defer/Decline outcomes, decline-final-for-conversation.
- `templates/skills/workflows/spek-knowledge/SKILL.md:47-77` — write mechanics (sources/categories, staged `--file`, decline handling); the skill the offers hand off to.
- `.claude/skills/spek-implement/SKILL.md` — confirms zero knowledge-base mention anywhere in the implement driver (root cause 1 of the motivating incident).
- `internal/steps/implement/steps.go:24,114-118` — `update_changelog` callback wiring; multi-source `analyze` edge; `Extra` nil.
- `internal/steps/implement/strategy.go:47-63` — implement template PathVars; `changelog_section_name` at :59.
- `internal/stepkit/stepkit.go:76-87` — bundle assembly order (base, PathVars, Extra).
- `internal/steps/implement/steps_test.go:44-52,292-347` — `renderStep` helper; existing `update_changelog` content tests; STOP-on-mismatch whole-table test at :329-347.
- `internal/steps/plan/steps_test.go:45-53,121-132,191-200,418-437,464+` — `renderStep`; discovery knowledge-commands test (the model); walkthrough wiring + content tests.
- `templates/context_directive_test.go:17,57` — stable-anchor-substring convention and whole-tree walk pattern for cross-template invariants.
- `templates/work_files_test.go:12-17,34,73,101` — `mustReadTemplate` raw-FS read pattern (mustache tags asserted literally).
- `cmd/skill.go:49-57,68-81` — skill name resolution; `skills/workflows/` not reachable via `skill <name>`.
- `tests/harbor/plan-workflow/tests/test_plan_workflow.py:61-82,88-103,107,120-121,187-190,831-857,936,1056-1195` — all hand-maintained oracles and the two live constraints on walkthrough prose (first-match goto regex; skills-per-step oracle).
- `tests/harbor/plan-workflow/solution/solve.sh:299-303` — scripted happy path never exercises a user correction; walkthrough offer not driven by the reference solution.

## External references

None — the feature is internal to this repository's own workflow templates; no external libraries or standards are involved.

## Prior plans / specs consulted

- Spec `000041_workflow-knowledge-capture-offers` (source of truth) — two offer points, offer-then-confirm contract, decline-final, selectivity bar, no new interruption points, template-content-test enforcement preferred.
- Plan `000040_plan-draft-then-walkthrough` — direct precedent for shipping workflow behavior as template prose plus mechanical test updates; established the walkthrough step, the assumptions beat, and the apply-immediately change-request path this feature attaches to; documents the house pattern "Go owns sequencing, prose owns judgement".
- Knowledge entries `architecture/testing-architecture.md` (three test layers; template changes put harbor surfaces in scope; harbor not in CI so drift is invisible) and `architecture/workflow-steps.md` (step/template/bundle mechanics).

## Open assumptions

- **Decline-is-final needs no mechanism** — assumed prose alone ("do not raise the offer again in this conversation", per `knowledge-trigger.md`'s model) satisfies the spec's decline-finality requirement; no state is tracked. If the implement workflow must *provably* suppress repeat offers across phases of one run, prose is the only available lever anyway.
- **No harbor oracle changes needed** — assumed adding prose (no new steps, no `skill <name>` references, no earlier `plan goto` occurrence) leaves every plan-workflow oracle untouched. A confirming harbor run is still prudent per `architecture/testing-architecture.md`, but no oracle edit is planned.
- **`EXPECTED_SKILLS_PER_STEP` staleness** (`discover-project-commands`, `discover-test-patterns` at :91-92 match no template) is pre-existing drift, out of scope for this feature.
- **Scope stays at the two templates** — `spek-implement/SKILL.md` and `spek-plan/SKILL.md` are not edited; the spec's technical approach targets the in-band step templates, and the skill files carry no per-step behavioral prose today.
- **The walkthrough offer also covers challenged assumptions** — assumed because line 18 routes a challenged assumption through the same change-request path; no separate prose needed for that case.

## Drafting assumptions

### Chosen direction: prose-only additions to the two templates, content-test enforced (architecture)
- **Decision**: Implement the feature entirely as instruction-prose additions to `templates/steps/implement/07-update_changelog.md` (assessment beat between the Discoveries entry and the continue-or-pause prompt) and `templates/steps/plan/18-walkthrough.md` (assessment beat at the end of the apply-immediately change-request path), sharing one behavioral shape (assess durability → offer once with what/why → accept routes to spek-knowledge → decline final for conversation), enforced by one new rendered-template content test per offer point. No Go changes, no new FSM steps, no new template variables.
- **Rationale**: The spec's technical approach prescribes prose; in-band per-step prose is the mechanism with a demonstrated success record in this repo, and both offer moments already sit inside existing conversational beats, so the no-new-interruption-points constraint is satisfied structurally. Lowest effort of the viable options and byte-compatible with every downstream contract.
- **Rejected**: (a) A dedicated knowledge-review FSM step — forbidden interruption point, plus step-table and harbor EXPECTED_STEP_ORDER churn. (b) Strengthening the background trigger / SKILL.md files — the surface that demonstrably failed, and one no content test guards. (c) Go-code durability detection — judgement is prose-owned in this codebase and no usable signal exists at the offer points.

### No conventions selected (architecture)
- **Decision**: Record "no project conventions apply" rather than padding a list.
- **Rationale**: `knowledge always-applied` returns only the two category README placeholders; there are no populated project conventions to select from.
- **Rejected**: Citing architecture/gotcha knowledge entries as conventions — they informed research but are not conventions-category entries.

### Template edit and its content test land in the same phase (phases)
- **Decision**: Each offer point is one phase bundling the template prose and its content test; the harbor confirmation is a separate final phase.
- **Rationale**: The test is the enforcement of the prose — landing them together keeps every phase independently green and mirrors how plan 000040's phases paired templates with their tests. The harbor run is separated because it needs external tooling (Docker, harbor CLI, credentials, ~25 min) and changes no code.
- **Rejected**: A single all-in-one phase (loses the independently-deliverable milestone boundary); separate prose/test phases (a prose-only phase would ship unenforced behavior).

### Offers reference the spek-knowledge skill by name, not `skill spek-knowledge` (discovery)
- **Decision**: New template prose will hand capture off to the `spek-knowledge` skill by name (and/or the raw `{{config.command}} knowledge` CRUD surface), never via `{{config.command}} skill spek-knowledge`.
- **Rationale**: `cmd/skill.go:49-57,68-81` cannot resolve skills under `templates/skills/workflows/` — the invocation would error at runtime, and no Go test catches a dangling skill reference. Both existing precedents (`02-discovery.md:47`, `knowledge-trigger.md`) use the surfaces chosen here.
- **Rejected**: Adding `spek-knowledge` to the CLI `skill` command's search path — a Go-code change out of scope for a prose feature.

### No harbor oracle updates planned (discovery)
- **Decision**: Treat all `tests/harbor/plan-workflow` oracles as unaffected; plan a confirming harbor run but no oracle edits.
- **Rationale**: The change adds prose only — no new steps (`EXPECTED_STEP_ORDER`), no `skill <name>` references (`EXPECTED_SKILLS_PER_STEP`), no content assertions exist on the two templates, and the walkthrough window is exempt from the no-confirmation-questions rule (`test_plan_workflow.py:847`). Constraint honored in drafting: no `plan goto` occurrence may appear in `18-walkthrough.md` earlier than the existing one (`INSTRUCTION_NEXT_STEP_RE` takes the first match).
- **Rejected**: Pre-emptively editing oracles — nothing to change; and fixing the pre-existing `EXPECTED_SKILLS_PER_STEP` staleness is unrelated drift, out of scope.

### Scope limited to the two step templates (discovery)
- **Decision**: Only `templates/steps/implement/07-update_changelog.md` and `templates/steps/plan/18-walkthrough.md` gain prose; `spek-implement`/`spek-plan` SKILL.md files are untouched.
- **Rationale**: The spec's technical approach prescribes in-band step-template prose as the working mechanism; SKILL.md files describe the driving loop, not per-step behavior, and the spec's offer points are both step-level moments.
- **Rejected**: Also adding a knowledge-base mention to `spek-implement/SKILL.md` — additive but unrequired; would dilute the one-mechanism story and add an untested prose surface.

## Rehydration cues

- `go run . knowledge always-applied`; `go run . knowledge read --data '{"scope":"project","path":"architecture/testing-architecture.md"}'`; same for `architecture/workflow-steps.md`.
- Re-read: `templates/steps/implement/07-update_changelog.md`, `templates/steps/plan/18-walkthrough.md`, `templates/steps/plan/02-discovery.md` (Step 5), `templates/agents/knowledge-trigger.md`, `templates/skills/workflows/spek-knowledge/SKILL.md`.
- Re-read tests: `internal/steps/plan/steps_test.go:121-132,418-437`; `internal/steps/implement/steps_test.go:292-347`; `templates/context_directive_test.go`.
- Harbor oracles: `tests/harbor/plan-workflow/tests/test_plan_workflow.py:61-190,831-857`.
- Prior plan: `go run . plan file read 000040_plan-draft-then-walkthrough/plan.md`.
- Spec: `go run . spec file read 000041_workflow-knowledge-capture-offers.md`.
