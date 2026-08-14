---
created_date: "2026-08-14"
status: completed
closed_date: "2026-08-14"
---

# Changelog: 000043_flipped-interaction-spec-interview

## What was built

The spec workflow no longer asks a scripted question per section. Every spec-creation session now opens with a single adaptive interview — before any section is drafted — modelled on the Flipped Interaction pattern (White et al., arXiv:2302.11382). The agent asks open, adapting questions toward "enough understanding to draft a credible first pass," decides each turn whether material ambiguity remains, and stops once it does not. Every section step (Overview through Non-Goals) then drafts its content from the interview and presents the draft for the user to confirm or correct.

- A new `interview` FSM step was inserted between `new` and `overview` (`internal/steps/spec/steps.go`), backed by a new step template `templates/steps/spec/00b-interview.md`. Synthesized findings land in `.spektacular/work/{{spec_name}}/interview.md`, and the existing per-workflow cleanup handles it as one more file.
- The plan workflow's repo-roster mechanism (`repoRoster`/`repoRosterExtra`) was relocated out of `cmd/plan.go` and `internal/steps/plan/steps.go` into shared homes (`internal/repo.Roster`, `internal/stepkit.RepoRosterExtra`) so the spec workflow's interview step can render the same full repo roster and, in a multi-repo project focused on one repo, ask a cross-repo impact question shaped by each other repo's own role/description.
- The opening instructions of all seven section-gathering step templates were rewritten to draft-and-confirm from the interview rather than ask their own scripted question; every section-specific rule (format, altitude, working-file conventions) was left unchanged.
- All eight gathering step templates (interview plus the seven sections) now carry a shared closing-beat paragraph: rejecting a drafted section opens a follow-up conversation, and the resulting edits land in whichever working file(s) they actually belong to — possibly a different, already-confirmed section — without a fresh confirmation gate for that other section. The end-of-workflow verification remains the single point everything is checked together.
- The driving skill (`templates/skills/workflows/spek-new/SKILL.md`), the harbor E2E harness (`tests/harbor/spec-workflow/`), and the template-contract tests (`templates/work_files_test.go`, and two new tests `templates/section_drafting_test.go` and `templates/rejection_repair_directive_test.go`) all now reflect the new step and interaction model. A new E2E scenario asserts a cross-section amendment triggered by rejecting a later section's draft.
- The docs site's `how-it-works.mdx` was rewritten to name the Flipped Interaction pattern with attribution, describe the interview phase and its stopping condition, and show a cross-repo worked example. The homepage's `FeaturesGrid` gained a new "Flipped Interaction interview" card so the capability surfaces as a differentiator rather than only on a deeper page.

## Why it matters / what it enables

The old scripted-question spec workflow assumed the writer already knew every question they needed to answer, and asked those questions in a fixed order regardless of what had already been said. People writing specs for features they had not fully thought through got a spec that reflected the script's assumptions rather than what they actually needed.

An adaptive interview that runs once, up front, and remembers what was said means each subsequent section can draft from real understanding — with the agent free to skip questions the interview already answered, ask something the script would never have thought to ask, and register cross-repo impact before the writer has to think about it. Rejection-repair lets a review of one section change another without an out-of-band correction step, so a spec can be refined the way people actually think about their features (in whatever order the reflection surfaces) rather than the order the FSM asked them.

## Deviations from the plan

None materially affected scope. Two test files not anticipated in context.md's original per-phase file lists needed fixing as side effects of inserting the new step: `cmd/root_test.go`'s two session-log tests hardcoded a direct `new` → `overview` transition (fixed inside Phase 1.1 rather than deferred), and `internal/steps/spec/steps_test.go`'s FSM-order tests were deliberately left failing at the end of Phase 1.1 and fixed by Phase 1.3 as planned.

Phase 2.2's rejection-repair paragraph wording was strengthened once mid-phase to fully satisfy its own acceptance criterion 3 (edits to more than one section, or none, not bound to exactly one change). Phase 3.1 embedded the interview example exchanges as plain fenced code blocks inside the existing `Fragment slot="body"` on `PipelineStage`, since the assumed `slot="example"` support does not exist on that component — extending it was out of scope for a content-only plan. Phase 2.3 resolved an Open Question about `solution/solve.sh` by confirming it is a Harbor standalone reference script never invoked by the pytest-driven `test.sh`, and left a comment there rather than reproducing the scenario twice.

A self-corrected implementation mistake during Phase 1.2 — an initial `config.RepoConfig{...}` struct literal in the relocated repo-roster tests silently failed `DescriptiveMetadata` because `config.RepoConfigFromYAMLFile`'s `cfg.Validate()` rejects a config whose `Knowledge`/`Changelog` sub-configs are zero-value — was caught and fixed within the phase. Any future test or caller constructing a `config.RepoConfig` from scratch (not read from an existing YAML file) must go through `NewDefaultRepoConfig()` first even when only the descriptive fields are what the test cares about.
