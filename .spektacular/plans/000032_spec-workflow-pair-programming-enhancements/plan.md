# Plan: 000032_spec-workflow-pair-programming-enhancements

<!-- Metadata -->
<!-- Created: 2026-07-09T10:01:07Z -->
<!-- Commit: bde4136 -->
<!-- Branch: f-conversational -->
<!-- Repository: git@github.com:jumppad-labs/spektacular.git -->

## Overview

Spektacular currently captures specifications only when a user deliberately starts the spec workflow. This plan adds proactive spec-creation offers during open-ended discussions: when a conversation produces something substantial enough to warrant a specification, the assistant recognizes the moment and offers to capture it, carrying forward what's already been decided so the user isn't forced to re-explain from scratch. The threshold for "substantial enough" is configurable per-project through a new `spec_trigger_threshold` field in `.spektacular/config.yaml` (values: `"strict"`, `"moderate"`, `"lenient"`), read live by the agent so a config change takes effect immediately, with a sensible default (`"moderate"`) that works out-of-the-box. The entire feature is delivered through two coordinated mechanisms: the configuration field, and a single managed instruction section in `AGENTS.md` that tells coding agents when and how to recognize, offer, and carry conversation context forward.

## Conventions

No project conventions apply to this feature.

The always-applied knowledge loaded during discovery contained only the category definitions for conventions and glossary (from `.spektacular/knowledge/conventions/README.md` and `glossary/README.md`), not actual project-specific conventions. The feature touches agent instructions (AGENTS.md), configuration (config.yaml), and reuses the existing spec-workflow step templates unchanged, but no coding standards, naming schemes, or required patterns were found in the knowledge base that bear on these surfaces.

## Architecture & Design Decisions

The feature is delivered through two coordinated mechanisms, both landing in existing, proven extension points: a new `spec_trigger_threshold` field in `.spektacular/config.yaml`, and a single new managed instruction section in `AGENTS.md` that tells agents when and how to offer spec creation, how to carry conversation context forward, and how to handle defer/decline. There is no third mechanism for context carry-forward — that is the central correction from an earlier, abandoned attempt at this plan (see below).

The threshold configuration follows the exact shape of every other enum-like field in `internal/config/config.go`: a top-level `SpecTriggerThreshold string` field (yaml `spec_trigger_threshold`), defaulted to `"moderate"` in `NewDefault()`, and validated with the same `switch`-with-empty-case-and-`default:`-error pattern used by `SpecConfig.IDMethod` (`internal/config/config.go:203-207`). This satisfies the spec's constraint that the threshold "must be stored in the project's existing configuration mechanism" with zero new machinery — no new file, no new loader, no new validation style for the agent or a future maintainer to learn.

The agent instruction is delivered as a new managed section in `AGENTS.md`, identified by its `## Spec-Worthy Discussion Recognition` heading, installed by a new `installSpecTriggerSection(projectPath, cfg, out)` function that is a structural clone of `installMemoryContextSection` (`internal/agent/memory_context.go:25-144`): read an embedded template, render it with mustache, locate the heading, replace/append/create exactly as the existing function does, write atomically. Each agent's `Install()` (`claude.go`, `codex.go`, `bob.go`) gains one call to it, mirroring the existing `installMemoryContextSection` call sites. The threshold is read dynamically by the agent from `.spektacular/config.yaml` at the moment it is deciding whether to offer, rather than being baked into the rendered instruction at `init` time — chosen over the init-time-bake alternative (which mirrors how `command` is handled in `memory-context.md` today) because a config change should take effect immediately without requiring the user to remember to re-run `spektacular init <agent>`. The trade-off is that the instruction must tell the agent to read a file at decision time — a new kind of behavior no existing managed section asks for — but this is a one-line addition to the instruction prose, not new code, and keeps the feature responsive to its own configuration.

Context carry-forward — originally planned as a second milestone with its own Go and template changes (a `spec new --data` payload, mustache conditionals in `templates/steps/spec/01-overview.md`) — turned out to need neither. Tracing the render path confirmed `stepkit.WriteStepResult` never merges `workflow.Data` into the templates it renders (`internal/stepkit/stepkit.go:78-85`), so that design would not have worked as specified even before considering whether it was the right approach. More importantly, it solved a problem that doesn't exist: the agent proposing the spec-creation offer, having it accepted, and starting `spec new` is the same agent in the same live conversation turn — it already holds the discussion in its own context. It needs no CLI mechanism to hand that context back to itself. The corrected design folds carry-forward into the same AGENTS.md instruction as the recognition/offer behavior: once the user accepts, the agent drives the existing spec workflow's steps (unmodified — `templates/steps/spec/01-overview.md` and its siblings keep asking "describe this feature," etc., exactly as today) but answers from what it already knows, proposing a draft to the user for confirmation or refinement rather than asking cold. This is pure agent behavior interpreting an unchanged instruction surface — no new workflow state, no new template variables, no new CLI flags exercised. The already-shipped `.spektacular/context.md`-clearing behavior in the spec workflow's `new()` step (`internal/steps/spec/steps.go:64-93`) is unaffected and untouched: it exists to let a *future, cold* session resume an interrupted spec workflow, which is a different problem from a live agent using context it already has in the same turn.

This approach beats the alternatives because it reuses two patterns the codebase has already proven out (the config-enum-validation pattern from `SpecConfig.IDMethod`, and the AGENTS.md managed-section pattern from spec 000023's `installMemoryContextSection`), and because it recognizes that context carry-forward needs no new mechanism at all rather than inventing one. Rejected alternatives — a runtime detection CLI command, hardcoded per-agent heuristics, a separate instruction file outside AGENTS.md, and the original CLI-data-passing design for carry-forward — are documented with citations in [research.md § Alternatives considered and rejected](./research.md#alternatives-considered-and-rejected).

## Component Breakdown

- **`config.yaml` (changed)** — Gains a new top-level `spec_trigger_threshold` field accepting `"strict"`, `"moderate"`, or `"lenient"`. When absent, treated as `"moderate"`. This is the single source of truth for the project's spec-trigger sensitivity, read by agents at decision time.

- **`Config` struct (changed)** — Gains a `SpecTriggerThreshold` field with the corresponding default and validation, following the exact pattern already used for every other enum-like configuration value in this package.

- **`AGENTS.md` (managed)** — Gains (or, on re-init, has updated in place) a new `## Spec-Worthy Discussion Recognition` section, written from an embedded template. Not hand-edited. Loaded natively by all three supported agents. Owns the entire behavioral contract for this feature: when to recognize a spec-worthy discussion, how to offer, what "accept" means (start the spec workflow and drive it from the conversation already had, proposing drafts for confirmation rather than asking cold), how to read the configured threshold, and how to track defer vs. decline within the conversation.

- **Spec-trigger instruction template (new)** — The embedded prose template that the managed section above is rendered from. Holds the full instruction: recognition criteria, the propose-then-confirm offer flow, the instruction to read the threshold from config at decision time, the carry-forward behavior for accepted offers, and the defer/decline handling rule.

- **Spec-trigger section installer (new)** — A small installer, structurally identical to the existing memory-context installer, that writes or updates the managed section in `AGENTS.md`. Owns only the mechanics of locating, replacing, appending, or creating the section — no behavioral logic of its own.

- **Per-agent install sequences (changed)** — Each supported agent's install sequence gains one additional step that invokes the new installer, alongside its existing install steps (skills, command wrappers, the memory-context section).

- **Spec workflow (unchanged, referenced)** — The existing `spec new`/`spec goto` workflow and its step templates are the target the new instruction drives. Nothing about them changes; the instruction tells the agent how to *use* the existing "ask the user..." prompts differently (propose-from-memory-then-confirm) when arriving with live conversation context, not to alter what those prompts ask.

- **Spec workflow's existing context.md resume behavior (unchanged, referenced)** — Already handles the separate case of a cold/interrupted session resuming a spec workflow. This plan does not touch it; it solves a different problem than live carry-forward.

- **Project knowledge store (unchanged, referenced)** — `.spektacular/knowledge/` and the `spek-knowledge` skill are referenced in the instruction as the destination for any learnings the agent identifies during the discussion. This plan does not modify them.

## Data Structures & Interfaces

The feature introduces one small configuration field and one installer function signature. It introduces no new workflow data keys, no new template variables for context carry-forward, and no changes to any existing interface — the corrected design (see Architecture) folds carry-forward into agent-instruction prose rather than a data contract.

### Configuration field

The `Config` struct gains a single new field:

```go
type Config struct {
    // ... existing fields ...
    SpecTriggerThreshold string `yaml:"spec_trigger_threshold"`
}
```

This accepts one of `"strict"`, `"moderate"`, or `"lenient"`. An absent or empty value is treated as `"moderate"`.

### Installer function signature

A new installer function matches the shape of the existing memory-context installer:

```go
func installSpecTriggerSection(projectPath string, cfg config.Config, out io.Writer) error
```

Called once from each supported agent's install sequence. Writes or updates the managed `## Spec-Worthy Discussion Recognition` section in `AGENTS.md`. Its signature is identical in shape to the existing memory-context installer, keeping the agent-installation surface uniform.

### No new workflow data contract

The spec workflow's existing data interface (get/set by key) is unchanged and unused by this feature — the earlier, abandoned design would have introduced new keys (e.g. `overview`) flowing through it; the corrected design does not, because context carry-forward happens entirely in agent behavior, not through any CLI-to-workflow data channel. No new template variables are introduced for step templates either; the only template variable the new AGENTS.md instruction template needs is `{{command}}` (matching the existing memory-context template), since the threshold is read by the agent directly from `config.yaml` at decision time rather than being rendered into the instruction.

## Implementation Detail

### Configuration extension pattern

The implementation extends the existing configuration mechanism by adding a single field and its validation, following the established pattern for every other enum-like configuration value: an exported, YAML-tagged struct field; a sensible default set alongside every other default; a switch-based validator that accepts the empty string (meaning "not configured, use default") and the supported literal values, and returns a descriptive error otherwise. No new configuration file, loading path, or validation style is introduced.

### Managed-section pattern reuse

The AGENTS.md instruction delivery reuses the managed-section pattern already proven for the memory-redirect feature: a new installer function structurally identical to the existing one, differing only in the heading it looks for, the template it renders, and the call sites it's wired into. The replace/append/create/idempotency algorithm itself is not reinvented — it is copied verbatim in shape. This keeps the agent-installation surface uniform: a developer reading any agent's install sequence sees a list of interchangeable install steps with no special cases for this one.

### Instruction prose as the entire behavioral surface

This is the plan's one genuinely new pattern, and it is deliberately *not* a code pattern: the full behavioral contract for this feature — when to recognize a spec-worthy discussion, how to phrase the offer, how to read the configured threshold, how to carry conversation context into the spec workflow once accepted, and how to track defer vs. decline — lives entirely as prose in the new AGENTS.md section. No Go code branches on "is this discussion spec-worthy," no workflow state tracks "was an offer made," no CLI mechanism transports conversation content. This is a deliberate continuation of the existing split in this codebase between what the CLI's state machine owns (step sequencing, file writes, validation) and what agent instructions own (judgment calls, conversational behavior, natural-language interpretation) — the same split the memory-redirect instruction and the propose-then-confirm pattern from the knowledge-write skill already rely on.

### Carry-forward as an instruction to reuse existing prompts differently, not a new prompt flow

The spec workflow's step templates (the "ask the user to describe this feature" prompts and their siblings) are not modified. The new instruction tells the agent to answer those same, unmodified prompts using the conversation it already has instead of asking the user cold — proposing a draft and asking for confirmation or correction. A developer reading the spec workflow's templates in isolation would see nothing different from today; the difference is entirely in how an agent that already has relevant context chooses to engage with an unmodified prompt.

### Code-structure UX

A developer encountering this feature in the codebase will see:
- A new field in `Config` that looks like every other config field, with matching default and validation style
- A new installer function in the agent package that looks like the existing memory-context installer, differing only in heading/template/name
- A new template that looks like the existing memory-context template in structure and length
- One new call in each agent's install sequence, indistinguishable in shape from the calls already there
- No changes anywhere in the spec workflow's step callbacks, templates, or data-passing surface

The feature integrates into existing patterns without introducing new abstractions, new packages, or new control flow anywhere in the CLI. The only genuinely novel element is the instruction prose itself, which is easy to iterate on without touching Go code — consistent with how the rest of this codebase already treats agent-facing behavior as configuration-adjacent rather than hardcoded.

## Dependencies

- **Spec 000020 (context)** — Already shipped. Provides the project knowledge store at `.spektacular/knowledge/` that the new instruction references as the destination for any learnings identified during a discussion. No changes required.

- **Spec 000022 (spek-knowledge skill)** — Already shipped. Provides the agent-facing skill and CLI used for propose-then-confirm knowledge writes. The new offer/carry-forward instruction follows the same propose-then-confirm shape. No changes required.

- **Spec 000023 (context-over-memory)** — Already shipped. Provides the managed-section pattern this plan's installer clones. No changes required.

- **`internal/config`** — Existing package. Provides the `Config` struct, its loader, and its validation pipeline. This plan adds one new field and its validation; no breaking changes to the package's API.

- **`internal/agent`** — Existing package. Provides the per-agent `Install()` methods and the embedded template filesystem. This plan adds one new installer function following the existing pattern and wires one call into each agent's install sequence; no changes to any existing function or interface.

- **`templates`** — Existing embedded filesystem. Provides the mustache templates for skills and agent instructions. This plan adds one new template; no changes to the embed directive or loading logic.

- **`github.com/cbroglie/mustache`** — External library, already in use. Renders the new instruction template. No version change or new usage pattern required.

- **`gopkg.in/yaml.v3`** — External library, already in use. Unmarshals the new config field. No version change or new usage pattern required.

- **The existing spec workflow (`internal/steps/spec`, `templates/steps/spec/*`)** — Existing, unmodified. This plan depends on it staying exactly as it is today: the new instruction tells agents how to *use* its existing prompts, and any change to those prompts' wording is out of scope here.

No new external dependencies are introduced. All referenced prior specs (000020, 000022, 000023) have already shipped and require no further changes before this plan can start.

## Testing Approach

### Test types and coverage

**Unit tests** cover the two genuinely new pieces of code: the configuration field's default and validation, and the installer function's idempotency contract. The installer function receives the same coverage shape as the existing memory-context installer it's cloned from: create-from-missing, append-after-existing-content, update-in-place idempotency, preserve-surrounding-content, template-change-picked-up, and cross-agent idempotency (claude → codex → bob leaves exactly one managed section). Config validation tests assert the field accepts only the three supported values plus the empty/absent case, and defaults to `"moderate"`. These follow the project's existing table-driven test conventions and live alongside the code they exercise.

**Integration tests are deliberately omitted**, for the same reason as the installer's own package already omits them: the installer's contract is fully exercised by unit tests against a temp directory, and the instruction's actual effectiveness — whether agents recognize spec-worthy discussions, offer correctly, honor the threshold, and carry context forward — is a property of agent behavior interpreting prose, not of code paths that can be asserted by an automated test. Automating "does the agent behave as instructed" would require mocking agent decision-making, which is brittle and does not reflect real usage.

**Manual smoke tests** verify that the instruction is loaded and honored: start a session with each of Claude, Codex, and Bob after running `spektacular init`, engage in a discussion that crosses the configured threshold, and observe that the agent offers to create a spec, then accepts and confirms the carried-forward draft rather than re-asking from scratch. Separately, confirm the same agents do not exhibit the behavior in a repository that has not been Spektacular-initialized (out-of-repo control), and that changing `spec_trigger_threshold` in `config.yaml` changes offer behavior immediately without re-running `init` (since the threshold is read dynamically, not baked in). These are recorded in the PR description, following the same pattern already established for the memory-redirect feature's manual verification.

### Load-bearing assertions

The unit tests guarantee:
- The config field accepts only `"strict"`, `"moderate"`, `"lenient"`, and the empty/absent value, rejecting everything else
- The config field defaults to `"moderate"` when absent
- The installer writes the managed section when `AGENTS.md` is missing
- The installer appends the section when `AGENTS.md` exists but the section is absent
- The installer updates the section in place on re-init without duplicating it
- The installer preserves surrounding content (e.g. the existing Memory & Context section, or an unrelated section)
- The installer picks up template changes on re-run
- Running `spektacular init` for all three agents against the same project leaves exactly one managed section

The manual smoke tests verify:
- Agents load the instruction from `AGENTS.md` at session start
- Agents recognize spec-worthy discussions and offer to capture them, honoring the configured threshold
- Accepted offers result in the agent proposing carried-forward drafts for confirmation rather than re-asking from scratch
- The threshold is read live from `config.yaml`, not baked in at init time
- The behavior is scoped to Spektacular-initialized repositories (out-of-repo control)

### Success metrics verification

The spec defines three success metrics, all of which are properties of real conversational behavior over time rather than deterministic code paths:

1. **"Substantial discussions consistently get offered a spec at a point that feels natural rather than premature or too late."** — Manual, captured in the implementation test plan. Depends on subjective judgment and real conversation flow; the implement workflow will produce a concrete test procedure describing live agent sessions with varied discussion patterns.

2. **"Users rarely feel surprised by the offer behavior."** — Manual, captured in the implementation test plan. Depends on human perception of over-triggering and under-triggering across discussion types; the test plan will describe scenarios covering both directions.

3. **"Users rarely need to adjust the default threshold."** — Manual, captured in the implementation test plan. Observable only through real usage patterns over time, not assertable in a test environment; the test plan will describe how to note threshold adjustments during dogfooding as a proxy signal.

All three are flagged for manual verification because they depend on subjective user experience or usage patterns over time that cannot be captured as a deterministic assertion at implementation time.

## Milestones & Phases

### Milestone 1: Agents recognize spec-worthy discussions, offer to capture them, and carry the conversation forward into the resulting spec

**What changes**: When a developer works in a Spektacular-initialized repository with any of the supported coding agents (Claude, Codex, Bob), and the conversation produces something substantial enough to warrant a specification — multiple requirements mentioned, a scoped decision reached, a feature described in detail — the agent recognizes the moment and proactively offers to capture the discussion as a spec. The offer is always propose-then-confirm: the agent presents the option, waits for the user's decision, and only starts the spec workflow on explicit acceptance. If the user isn't ready, the agent continues normally and may raise the offer again later as the discussion develops; if the user declines outright, the agent drops the offer for the remainder of that discussion. The threshold for "substantial enough" is configurable per-project through a new `spec_trigger_threshold` field in `.spektacular/config.yaml` (`"strict"`, `"moderate"`, or `"lenient"`), read live by the agent so a config change takes effect immediately, with a sensible default (`"moderate"`) when unconfigured. When the user accepts the offer, the agent starts the spec workflow and drives its existing prompts using the conversation it already had — proposing a draft answer for the user to confirm or refine, rather than asking the same questions from scratch. Outside Spektacular-initialized repositories, agents continue their normal behavior without offering specs. Running `spektacular init` for any agent installs or updates the instruction that enables all of this; re-running init (for the same agent or a different one) updates it in place without duplication.

**Validation point**: After running `spektacular init <agent>` in this repository, start a fresh session with that agent and engage in a discussion that describes a feature with multiple requirements. The agent offers to create a spec before proceeding to implementation; accepting the offer starts the spec workflow with the agent proposing drafted answers from the conversation rather than asking cold. Deferring keeps the conversation open and allows a later re-offer; declining suppresses further offers for that discussion. Changing `spec_trigger_threshold` in `config.yaml` changes the offer's sensitivity for the same conversation shape without needing to re-run `init`. Start a session in an unrelated, non-initialized repository with the same agent and confirm it does not exhibit the behavior. Re-run `spektacular init <agent>` and confirm the instruction in `AGENTS.md` is updated in place without duplication.

#### - [x] Phase 1.1: Add the `spec_trigger_threshold` config field

Add a new top-level field to the project configuration that lets a team configure how readily agents should offer to capture a discussion as a spec, accepting `"strict"`, `"moderate"`, or `"lenient"`, defaulting to `"moderate"` when not set. Validation rejects any other value with a clear error, following the exact style already used for the project's other configurable enum-like settings.

*Technical detail:* [context.md#phase-11](./context.md#phase-11-add-the-spec_trigger_threshold-config-field)

**Acceptance criteria**:

- [x] The configuration accepts a `spec_trigger_threshold` value of `"strict"`, `"moderate"`, or `"lenient"` in `.spektacular/config.yaml`
- [x] When the field is absent, the project behaves as though it were set to `"moderate"`
- [x] Setting the field to any other value produces a clear validation error naming the field and the allowed values
- [x] Existing config files that don't mention this field continue to load and validate exactly as before

#### - [x] Phase 1.2: Add the AGENTS.md managed instruction section (installer plumbing)

Add the mechanism that installs and keeps up to date a new managed section in `AGENTS.md`, following the exact structure of the existing memory-and-context managed section: an embedded instruction template, an installer that locates the section by its heading and replaces it in place (or appends it if absent, or creates the file if missing), and a single wiring call added to each supported agent's install sequence. This phase delivers the plumbing with placeholder instruction content; Phase 1.3 replaces the placeholder with the real instruction.

*Technical detail:* [context.md#phase-12](./context.md#phase-12-add-the-agentsmd-managed-instruction-section-installer-plumbing)

**Acceptance criteria**:

- [x] The managed section is written to `AGENTS.md` when the file is missing
- [x] The managed section is appended when `AGENTS.md` exists but the section is absent
- [x] The managed section is updated in place on re-init without duplicating it
- [x] Surrounding content in `AGENTS.md` (e.g. the existing Memory & Context section) is preserved untouched
- [x] Running `spektacular init` for each of the three supported agents against the same project leaves exactly one managed section

#### - [x] Phase 1.3: Write the spec-trigger instruction content

Replace the placeholder content from Phase 1.2 with the full instruction: how to recognize a spec-worthy discussion, how to read the project's configured threshold at decision time (not baked in at install time), how to phrase and offer the propose-then-confirm choice, how to handle a deferral (keep the conversation open, may re-offer later) versus an outright decline (stop offering for that discussion), and — once accepted — how to start the spec workflow and drive its existing prompts using the conversation already had, proposing draft answers for the user to confirm or refine rather than asking from scratch.

*Technical detail:* [context.md#phase-13](./context.md#phase-13-write-the-spec-trigger-instruction-content)

**Acceptance criteria**:

- [x] The instruction tells the agent to check the project's configured threshold before deciding whether a discussion qualifies
- [x] The instruction describes the propose-then-confirm offer flow, including deferral and outright decline as distinct outcomes
- [x] The instruction tells the agent, once an offer is accepted, to drive the spec workflow's existing prompts from the conversation already had, proposing a draft for user confirmation rather than asking from scratch
- [x] The instruction states that the threshold is read live from the project's configuration, not fixed at install time

## Open Questions

No open questions remain. All design decisions have been resolved during planning:

- **Configuration mechanism**: Resolved to `spec_trigger_threshold` field in `config.yaml` with validation matching `SpecConfig.IDMethod`'s pattern.
- **Instruction delivery**: Resolved to a managed section in `AGENTS.md` following the spec 000023 pattern.
- **Threshold interpretation timing**: Resolved to a dynamic read of `config.yaml` at decision time, not baked in at `spektacular init` time — chosen so a config change takes effect immediately.
- **Context carry-forward mechanism**: Resolved to pure agent-instruction prose — the agent drives the existing, unmodified spec workflow prompts using conversation context it already has, proposing drafts for user confirmation. No CLI or template changes.
- **Defer/decline tracking**: Resolved to agent-side conversation memory, scoped to the single conversation (no new workflow state, no cross-session tracking).
- **Milestone/phase structure**: Resolved to a single milestone, three phases (config field; installer plumbing with placeholder content; real instruction prose).
- **Threshold interpretation itself ("strict" vs "moderate" vs "lenient" in concrete terms)**: Left as natural-language guidance in the instruction prose for the agent to interpret — this is a deliberate design choice (see Architecture), not an unresolved question.

The implementer should proceed with the phases as defined. If any assumption proves incorrect during implementation (e.g. agents do not interpret the threshold prose consistently across Claude/Codex/Bob, or reading `config.yaml` mid-conversation turns out to need different handling than expected), STOP and ask the user before adjusting the approach.

## Out of Scope

- **Polishing the requirements-gathering step's interview UX** — propose-then-confirm drafting, surfacing rephrasing out loud, prompting for completeness gaps. Deferred to a separate future spec (noted in spec § Non-Goals).

- **Adding a user-acceptance walkthrough to the plan workflow's verification step** — a separately identified gap, not addressed here (noted in spec § Non-Goals).

- **A durable changelog/context artifact for downstream doc/blog generation** — deferred to its own separate spec (noted in spec § Non-Goals). Plan 000033 (spec-workflow-output-changelog) addresses a related but distinct problem — an output record after implementation — and is not a dependency of this plan.

- **Recognizing spec-worthy discussion across multiple separate conversations or sessions** — this feature operates only within a single, ongoing conversation. Cross-session recognition would require persistent conversation-ID tracking, which is not part of this design (noted in spec § Non-Goals).

- **Automatically creating a spec without the user's explicit acceptance of the offer** — the assistant always offers and waits for a decision; it never starts a spec workflow unilaterally (noted in spec § Non-Goals).

- **Any CLI or workflow-level mechanism for context carry-forward** — rejected during discovery/architecture. An earlier attempt at this plan proposed passing conversation context through `spec new --data` and mustache template conditionals; this was found to be both non-functional as designed (the CLI never merges workflow data into rendered templates) and unnecessary (the agent already holds the conversation live and needs no CLI round-trip to use it). The corrected design is pure agent-instruction prose with zero CLI/template changes (documented in research.md § Alternatives considered and rejected).

- **Modifying the existing spec workflow's step templates** — the "ask the user..." prompts in `templates/steps/spec/*.md` are not changed by this plan. The new instruction changes how an agent engages with those unmodified prompts, not the prompts themselves.

- **Modifying the existing `context.md`-clearing/resume behavior in the spec workflow's `new()` step** — already shipped, solves a different problem (resuming a cold/interrupted session), and is unaffected by this plan.

- **Runtime detection command** — rejected at the architecture step in favor of instruction-based recognition. A `spektacular detect-spec-worthy` CLI command would add latency and assume agents can pass conversation history to the CLI, which is not guaranteed across all three supported agents (documented in research.md § Alternatives considered and rejected).

- **Hardcoded agent heuristics** — rejected because it violates the spec's constraint that the trigger threshold must be user-configurable. Fixed rules in agent instructions cannot be adjusted per-project without editing the instructions themselves (documented in research.md § Alternatives considered and rejected).

- **Baking the threshold value into the rendered instruction at `spektacular init` time** — considered and rejected in favor of a dynamic read at decision time, so a `config.yaml` change takes effect without requiring a re-init (decided during the architecture step).

- **Enforcing adherence** — no hook, lint, pre-commit check, or runtime guardrail detects whether an agent actually offers, reads the threshold, or carries context forward correctly. Adherence is best-effort per agent, verified only by manual smoke testing.

## Changelog

### FINAL SUMMARY

This plan delivered proactive spec-creation offers: agents now recognize when an open-ended discussion has become substantial enough to warrant a specification, offer to capture it (honoring a per-project `spec_trigger_threshold` config value), and — on acceptance — drive the existing spec workflow using conversation context already in hand rather than asking cold. All three phases shipped as designed, with no scope changes; the only correction along the way was resolving a stale prediction in context.md about needing to update test fixtures in Phase 1.3, which turned out to be unnecessary once traced through the actual test code.

**Total phases**: 3/3 completed

**Notable deviations from the plan**: A necessary aside during Phase 1.1's verification: full-repo tests surfaced pre-existing stale tests (in `cmd/` and `templates/`) left over from the immediately-prior commit's intentional change to the spec workflow's `new()` step (it now stops at `"new"` instead of auto-advancing to `"overview"`, so the agent can write `context.md` first). These were fixed as part of getting a clean verification gate — not a scope change to this plan's own three phases, but recorded here since it touched files outside the plan's own component list.

### 2026-07-09 — Phase 1.1: Add the `spec_trigger_threshold` config field

**What was done**: Added a new top-level `spec_trigger_threshold` field to the project's `Config` struct, accepting `"strict"`, `"moderate"`, or `"lenient"`, defaulting to `"moderate"` when absent. Validation rejects any other value with an error naming the field and the allowed values, following the same pattern already used for the existing `spec.id_method` field.

**Deviations**: While verifying this phase, full-repo `go test ./...` surfaced pre-existing failures in `cmd/` and `templates/` unrelated to this change — stale tests written against the old auto-advance behavior of the spec workflow's `new()` step, which was intentionally changed to stop at `"new"` (not auto-advance to `"overview"`) by the immediately-prior commit so the agent can write `context.md` before proceeding. Fixed these five stale tests and added a documented exemption for `steps/spec/00-new.md` in the shared context-directive test, so the full suite is green again. This was a necessary aside to get a clean verification gate, not part of this phase's own scope.

**Files changed**:
- `internal/config/config.go`
- `internal/config/config_test.go`
- `cmd/root_test.go`
- `cmd/spec_test.go`
- `templates/context_directive_test.go`

**Discoveries**: The spec workflow's `new()` step (`internal/steps/spec/steps.go`) no longer auto-advances to `"overview"` — it intentionally stops at `"new"` so the agent can write conversation context first. Any test or tooling that assumes `spec new` lands on `"overview"` needs to add an explicit `goto overview` step. `templates/steps/spec/00-new.md` also does not carry the generic "refresh context.md" directive marker used by every other non-terminal step template, since it has its own bespoke, more detailed context-writing instruction — this is intentional and now explicitly exempted in `templates/context_directive_test.go`.

### 2026-07-09 — Phase 1.2: Add the AGENTS.md managed instruction section (installer plumbing)

**What was done**: Added a new managed section installer, `installSpecTriggerSection` in `internal/agent/spec_trigger.go`, structurally cloned from the existing `installMemoryContextSection`. It creates, appends, or replaces in place a `## Spec-Worthy Discussion Recognition` section in a project's `AGENTS.md`, rendered from a new embedded template `templates/agents/spec-trigger.md` (placeholder body — Phase 1.3 fills in the real instruction prose). Wired one call to the new installer into each of `claude.go`, `codex.go`, and `bob.go`'s `Install()` sequence, immediately after their existing memory-context install call.

**Deviations**: None.

**Files changed**:
- `internal/agent/spec_trigger.go`
- `internal/agent/spec_trigger_test.go`
- `internal/agent/claude.go`
- `internal/agent/codex.go`
- `internal/agent/bob.go`
- `templates/agents/spec-trigger.md`

**Discoveries**: None beyond what Phase 1.1 already surfaced. The installer reuses the existing `writeAGENTSAtomic`/`writeFileAtomic` helpers from `memory_context.go` rather than duplicating them, since those helpers aren't heading-specific. The cross-agent idempotency test now verifies both managed sections (`## Memory & Context` and `## Spec-Worthy Discussion Recognition`) survive together with exactly one heading each — useful joint coverage now that two managed sections install side by side.

### 2026-07-09 — Phase 1.3: Write the spec-trigger instruction content

**What was done**: Replaced `templates/agents/spec-trigger.md`'s placeholder body with the full instruction: recognition criteria for a spec-worthy discussion, reading `spec_trigger_threshold` live from `.spektacular/config.yaml` at decision time (defaulting to `"moderate"` when absent), the propose-then-confirm offer flow, the three outcomes (accept/defer/decline) with decline being final for that discussion topic, and the carry-forward behavior for accepted offers (drive the existing spec workflow prompts using conversation context already had, proposing drafts for user confirmation).

**Deviations**: None from the plan's acceptance criteria. One correction to context.md's prediction: the Phase 1.3 technical-detail note anticipated needing to update Phase 1.2's test fixture strings to match the real content — this turned out not to be necessary, since `spec_trigger_test.go` uses its own independent in-memory fixture (mirroring how `memory_context_test.go` already worked), not the real template body. No test files were touched in this phase.

**Files changed**:
- `templates/agents/spec-trigger.md`

**Discoveries**: None beyond the fixture-independence correction noted above. This completes all 3 phases of Milestone 1 — the feature (config field, AGENTS.md installer, instruction content) is now fully implemented.
