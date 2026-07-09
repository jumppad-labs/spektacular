# Plan: 000032_spec-workflow-pair-programming-enhancements

<!-- Metadata -->
<!-- Created: 2026-07-08T15:38:50Z -->
<!-- Commit: c6e4796 -->
<!-- Branch: f-conversational -->
<!-- Repository: git@github.com:jumppad-labs/spektacular.git -->

## Overview

Spektacular currently captures specifications only when a user deliberately starts the spec workflow. This plan adds proactive spec-creation offers during open-ended discussions: when a conversation produces something substantial enough to warrant a specification, the assistant recognizes the moment and offers to capture it, carrying forward what's already been decided so the user isn't forced to re-explain from scratch. The threshold for "substantial enough" is configurable per-project through a new `spec_trigger_threshold` field in `.spektacular/config.yaml` (values: `"strict"`, `"moderate"`, `"lenient"`), with a sensible default (`"moderate"`) that works out-of-the-box. The feature is delivered through two coordinated mechanisms: the configuration field that sets the threshold, and a managed instruction section in `AGENTS.md` that tells coding agents when and how to offer spec creation.

## Conventions

No project conventions apply to this feature.

The always-applied knowledge loaded during discovery contained only the category definitions for conventions and glossary (from `.spektacular/knowledge/conventions/README.md` and `glossary/README.md`), not actual project-specific conventions. The feature touches agent instructions (AGENTS.md), configuration (config.yaml), and the spec workflow's data-passing mechanism, but no coding standards, naming schemes, or required patterns were found in the knowledge base that bear on these surfaces.

## Architecture & Design Decisions

The feature is delivered through two coordinated mechanisms: a configuration field in `config.yaml` that sets the project's spec-trigger threshold, and a managed instruction section in `AGENTS.md` that tells coding agents when and how to offer spec creation during open-ended discussions.

The threshold configuration lives as a new top-level `spec_trigger_threshold` field in `.spektacular/config.yaml`, accepting string values like `"strict"`, `"moderate"`, or `"lenient"`. This follows the project's established configuration pattern (all project settings live in `config.yaml`, loaded via `config.FromYAMLFile`) and satisfies the spec's constraint that the threshold must be stored in the existing configuration mechanism. When no value is configured, a sensible default (`"moderate"`) applies, allowing the feature to work out-of-the-box while remaining adjustable for teams with different process requirements.

The agent instruction is delivered as a new managed section in `AGENTS.md` — identified by its `## Spec-Worthy Discussion Recognition` markdown heading — following the same pattern established by the memory-redirect feature (spec 000023). The section is written by `spektacular init`, updated idempotently on re-init, and loaded natively by all three supported agents (Claude via `@AGENTS.md` in `CLAUDE.md`, Codex and Bob directly). The instruction tells agents to recognize when an ongoing discussion has crossed the configured threshold (e.g., multiple requirements mentioned, scoped decision reached, substantial feature described) and proactively offer to capture it as a spec rather than proceeding silently to implementation. The offer is always propose-then-confirm: the agent presents the option, waits for the user's decision (accept, defer, or decline), and only starts the spec workflow on explicit acceptance.

Context carry-forward is handled through the spec workflow's existing `--data`, `--stdin`, and `--file` flags. When the user accepts the offer, the agent invokes `spec new` with pre-populated data (e.g., `--data '{"name":"...", "overview":"<already-discussed-content>"}'` or `--stdin overview` for longer bodies), and the spec workflow receives that context before the first step runs. The step templates are modified to detect pre-populated content and skip re-prompting when present, so the user is not forced to re-answer questions the conversation already covered. This mechanism requires no new workflow state — it uses the existing data map that every step already receives.

Defer and decline tracking is agent-side conversation management, not CLI state. The instruction tells agents to remember if the user declined the offer for a specific discussion topic and suppress re-offering for that topic within the same conversation, relying on the agent's own memory (or `.spektacular/context.md` for cross-session persistence if needed). The CLI has no concept of "discussion ID" or "offer declined" — it only knows whether a workflow is in progress — so this responsibility stays with the agent where conversation context naturally lives.

This approach beats the alternatives because it uses proven patterns from the codebase (the AGENTS.md managed-section pattern from spec 000023, the `config.yaml` configuration pattern from `internal/config/config.go`, the workflow data pre-population from `cmd/spec.go`), works identically for all three supported agents without agent-specific code, and requires no new CLI commands or runtime detection logic. The threshold remains user-configurable per-project, the instruction surface stays consolidated in `AGENTS.md` (which all agents already load), and the spec workflow's existing data-passing mechanisms handle context carry-forward without structural changes. Rejected alternatives — including a runtime detection command (adds latency, assumes agents can pass conversation history) and hardcoded heuristics (violates the configurability constraint) — are documented with citations in [research.md § Alternatives considered and rejected](./research.md#alternatives-considered-and-rejected).

## Component Breakdown

- **`config.yaml` (changed)** — Gains a new top-level `spec_trigger_threshold` field accepting string values (`"strict"`, `"moderate"`, `"lenient"`). When absent, defaults to `"moderate"`. This is the single source of truth for the project's spec-trigger sensitivity.

- **`internal/config/config.go` (changed)** — The `Config` struct gains a `SpecTriggerThreshold string` field. The `NewDefault()` function sets it to `"moderate"`. The `Validate()` method checks that the value, if present, is one of the three supported values. Follows the existing pattern for all other config fields.

- **`AGENTS.md` (managed)** — Gains (or, on re-init, has updated in place) a new `## Spec-Worthy Discussion Recognition` section written from the embedded template. Not hand-edited. Loaded natively by all three agents (Claude via `@AGENTS.md` in `CLAUDE.md`, Codex and Bob directly). Contains the instruction to recognize spec-worthy discussions, check the configured threshold, and offer to capture the conversation as a spec.

- **`templates/agents/spec-trigger.md` (new)** — Embedded markdown template for the managed section, rendered through `mustache` against `cfg.Command` and `cfg.SpecTriggerThreshold`. Holds the prose instruction that tells agents when to offer a spec (threshold-based trigger), how to offer (propose-then-confirm), what to do on accept (invoke `spec new` with pre-populated data), and how to handle defer/decline (agent-side tracking, no re-offer for declined topics).

- **`internal/agent/spec_trigger.go` (new)** — Shared helper exposing `installSpecTriggerSection(projectPath, cfg, out)` that reads the embedded template, renders it, locates the `## Spec-Worthy Discussion Recognition` heading in `AGENTS.md`, and replaces from that heading to the next H1/H2 or EOF (or appends if absent, or creates the file if missing). Idempotent. Follows the exact pattern from `memory_context.go` (spec 000023).

- **`internal/agent/claude.go`, `codex.go`, `bob.go` (changed)** — Each `Install()` method gains a single call to `installSpecTriggerSection(projectPath, cfg, out)` after its existing skill/command-wrapper calls. No other change to the per-agent files.

- **`templates/steps/spec/01-overview.md` (changed)** — Modified to check for a pre-populated `overview` key in workflow data (passed via `spec new --data` or `--stdin`). If present, the template presents it to the user for confirmation/refinement rather than prompting from scratch. If absent, prompts as normal. This is the mechanism for "carrying forward already-established context".

- **`templates/steps/spec/02-requirements.md`, `03-acceptance_criteria.md`, etc. (optionally changed)** — May be modified to detect pre-populated data for their respective sections (e.g., `requirements`, `acceptance_criteria`), following the same pattern as the overview step. The extent of pre-population support is determined during implementation based on what the agent can realistically extract from an open-ended discussion.

- **Project knowledge store (unchanged, referenced)** — The `.spektacular/knowledge/` store and the `spek-knowledge` skill (shipped under spec 000022) are referenced in the instruction as the destination for any learnings the agent identifies during the discussion. This plan does not modify them; it only directs agents to use them.

## Data Structures & Interfaces

The feature introduces minimal new data structures, focusing instead on extending existing configuration and leveraging the project's established template rendering and workflow data mechanisms.

## Configuration Extension

The `Config` struct in `internal/config/config.go` gains a single new field:

```go
type Config struct {
    // ... existing fields ...
    SpecTriggerThreshold string `yaml:"spec_trigger_threshold"`
}
```

This field accepts one of three string values: `"strict"`, `"moderate"`, or `"lenient"`. When absent from `config.yaml`, `NewDefault()` sets it to `"moderate"`. The `Validate()` method checks that the value, if present, is one of the three supported values and returns an error otherwise.

## Template Variables

The embedded template at `templates/agents/spec-trigger.md` is rendered through `mustache.Render` against a map containing:

```go
map[string]string{
    "command":   cfg.Command,           // e.g. "spektacular"
    "threshold": cfg.SpecTriggerThreshold, // e.g. "moderate"
}
```

This follows the same pattern as the existing skill templates and the memory-context template from spec 000023. The `{{command}}` placeholder is used for CLI invocations in the instruction prose, and `{{threshold}}` allows the instruction to reference the configured sensitivity level.

## Workflow Data Keys

The spec workflow's data map (accessed via `workflow.Data` interface) may contain pre-populated section content passed from the agent via `spec new --data` or `--stdin`/`--file`. The keys match section names:

- `overview` — pre-populated overview text
- `requirements` — pre-populated requirements list
- `acceptance_criteria` — pre-populated acceptance criteria
- (and so on for other sections)

These are plain string values, not structured types. The step templates check for their presence using mustache conditionals (e.g., `{{#overview}}...{{/overview}}`) and present the pre-populated content to the user for confirmation/refinement rather than prompting from scratch.

## Installer Function Signature

The new `internal/agent/spec_trigger.go` exposes a single function matching the pattern from `memory_context.go`:

```go
func installSpecTriggerSection(projectPath string, cfg config.Config, out io.Writer) error
```

This is called from each agent's `Install()` method and writes (or updates in place) the managed section in `AGENTS.md`. The function signature is identical to `installMemoryContextSection` except for the name, maintaining consistency across the agent installation surface.

No new exported types, no new package boundaries, and no changes to the existing `Agent` interface. The feature operates entirely through configuration extension, template rendering, and the workflow's existing data-passing mechanism.

## Implementation Detail

## Configuration Extension Pattern

The implementation extends the existing configuration mechanism by adding a single field to the `Config` struct and updating the validation logic to check for supported values. This follows the established pattern used for all other configuration fields: the struct field is exported and tagged for YAML unmarshaling, `NewDefault()` sets a sensible default, and `Validate()` enforces constraints. No new configuration file or loading mechanism is introduced — the feature integrates into the existing `config.yaml` → `Config` struct → validation pipeline.

## Managed Section Pattern Reuse

The AGENTS.md instruction delivery reuses the managed-section pattern shipped under spec 000023 (memory-redirect). A new installer function (`installSpecTriggerSection`) follows the exact structure of `installMemoryContextSection`: read embedded template, render with mustache, locate section by markdown heading, replace or append, write atomically. The pattern is proven, tested, and already works for all three agents. The only differences are the template path, the heading text, and the template variables — the algorithm and idempotency contract are identical.

## Template Rendering Consistency

The embedded template is rendered through the same `mustache.Render` path used by workflow skills and the memory-context section. The template receives a map of string variables (`command`, `threshold`) and produces plain markdown prose. This keeps the instruction-generation surface uniform: all agent-facing prose is template-driven, all templates live under `templates/`, and all rendering goes through the same mustache helper. A developer adding a new managed section in the future will follow this exact pattern.

## Step Template Pre-Population Detection

The spec step templates gain optional pre-population detection: check for a workflow data key matching the section name (e.g., `{{#overview}}...{{/overview}}`), and if present, display the pre-populated content for confirmation/refinement rather than prompting from scratch. This is a backward-compatible addition — templates that don't check for pre-population continue to prompt as normal, and the workflow data map already supports arbitrary keys. The pattern is: check, present if found, prompt if absent. No new workflow state or step logic is required.

## Agent Install Path Extension

Each agent's `Install()` method gains a single call to the new installer function, placed after the existing skill and command-wrapper installs. This follows the established pattern: `Install()` is a sequence of independent install operations (skills, commands, AGENTS.md sections), each returning an error that halts the sequence if it fails. The new call is indistinguishable from the existing ones — same signature, same error-handling, same one-line-per-artifact output format. A developer reading any agent's `Install()` will see a uniform list of install steps with no special cases.

## Instruction Prose as Configuration

The threshold-based trigger logic lives entirely in the AGENTS.md instruction prose, not in Go code. The instruction tells agents "when you recognize a discussion has crossed the `{{threshold}}` threshold, offer to capture it as a spec" — the interpretation of "strict", "moderate", or "lenient" is left to the agent's natural-language understanding. This keeps the CLI free of conversation-analysis logic and avoids hardcoding heuristics that would need to evolve as agent capabilities improve. The trade-off: consistency across agents depends on clear instruction wording, not enforced by code.

## No New CLI Commands or State

The feature introduces no new CLI commands, no new workflow state fields, and no new state transitions. The spec workflow's existing `--data`, `--stdin`, and `--file` flags handle context pre-population, and the workflow's existing data map stores it. Defer and decline tracking is agent-side conversation management, not CLI state. The only CLI-visible change is the new configuration field, which is read at init time and rendered into the instruction template. A developer working on the CLI will see no new command handlers, no new state machine logic, and no new JSON output schemas.

## Code-Structure UX

A developer encountering this feature in the codebase will see:
- A new field in `Config` that looks like every other config field
- A new installer function in `internal/agent/` that looks like `installMemoryContextSection`
- A new template in `templates/agents/` that looks like `memory-context.md`
- A new call in each agent's `Install()` that looks like the existing calls
- Optional pre-population checks in spec step templates that follow the mustache conditional pattern

The feature integrates into existing patterns without introducing new abstractions, new packages, or new control flow. The only novel element is the instruction prose itself, which is template-driven and therefore easy to iterate on without touching Go code.

## Dependencies

- **Spec 000020 (context)** — Already shipped. Provides the project knowledge store at `.spektacular/knowledge/` that the instruction references as the destination for learnings identified during discussions. No changes required.

- **Spec 000022 (spek-knowledge skill)** — Already shipped. Provides the agent-facing skill and `go run . knowledge {search,read,write}` CLI used in the instruction's propose-then-confirm pattern. No changes required.

- **Spec 000023 (context-over-memory)** — Already shipped. Provides the managed-section pattern (`installMemoryContextSection` in `internal/agent/memory_context.go`) that this plan reuses for the spec-trigger instruction. No changes required.

- **`internal/config`** — Existing package. Provides the `Config` struct, `FromYAMLFile` loader, and validation logic. This plan extends the struct with one new field and adds validation for it; no breaking changes to the package's API.

- **`internal/agent`** — Existing package. Provides the per-agent `Install()` methods and the `sourceFS` variable for template loading. This plan adds a new installer function following the existing pattern and wires it into each agent's `Install()`; no changes to the `Agent` interface or existing functions.

- **`internal/workflow`** — Existing package. Provides the `Data` interface and the workflow state machine. This plan uses the existing data-passing mechanism (`SetData`, `GetData`) without modification; no changes to the package.

- **`templates`** — Existing embedded filesystem. Provides the mustache templates for skills and agent instructions. This plan adds one new template (`agents/spec-trigger.md`) following the existing structure; no changes to the embed directive or loading logic.

- **`github.com/cbroglie/mustache`** — External library, already in use. Provides template rendering for the new instruction template. No version change or new usage patterns required.

- **`gopkg.in/yaml.v3`** — External library, already in use. Provides YAML unmarshaling for the new config field. No version change or new usage patterns required.

- **`io/fs`** — Standard library, already in use. Provides filesystem abstraction for reading the embedded template. No new usage required.

No new external dependencies are introduced. All referenced prior work (specs 000020, 000022, 000023) has already shipped and requires no changes. The feature integrates into existing packages and libraries without breaking changes or version bumps.

## Testing Approach

## Test Types and Coverage

The feature is tested through three layers:

**Unit tests** cover the configuration extension, the installer function's idempotency contract, and template rendering. The installer function (`installSpecTriggerSection`) receives the same test coverage as `installMemoryContextSection` from spec 000023: create-from-missing, append-after-existing-content, update-in-place idempotency, preserve-surrounding-content, template-change-picked-up, and cross-agent idempotency (claude → codex → bob). Config validation tests assert that `SpecTriggerThreshold` accepts only the three supported values and defaults to `"moderate"` when absent. These tests follow the project's existing table-driven test pattern and live alongside the production code they exercise.

**Integration tests** are deliberately omitted. The installer function's contract (write or update a managed section in AGENTS.md) is fully exercised by unit tests against a temp directory, and the instruction's effectiveness (whether agents actually recognize spec-worthy discussions and offer to capture them) is verified through manual smoke testing rather than automated integration tests. Attempting to automate agent behavior would require mocking conversation state and agent decision-making, which is brittle and does not reflect real usage.

**Manual smoke tests** verify that the instruction is loaded by each agent and that agents honor the threshold configuration. These are recorded in the PR description rather than committed as automated tests: start a session in this repo with each of Claude, Codex, and Bob after running `spektacular init`, engage in a discussion that crosses the configured threshold, and observe that the agent offers to create a spec. Then start a session in an unrelated repo and confirm the same agents do not exhibit the behavior (out-of-repo control). These checks are best-effort and not enforced by CI.

## Load-Bearing Assertions

The unit tests guarantee:
- The config field accepts only `"strict"`, `"moderate"`, or `"lenient"` and rejects other values
- The config field defaults to `"moderate"` when absent
- The installer function writes the managed section to AGENTS.md when the file is missing
- The installer function appends the section when AGENTS.md exists but the section is absent
- The installer function updates the section in place on re-init without duplicating it
- The installer function preserves surrounding content (e.g., a tessl-managed block above, an unrelated section below)
- The installer function picks up template changes when re-run with a different config
- Running `spektacular init` for three different agents against the same project leaves exactly one managed section

The manual smoke tests verify:
- Agents load the instruction from AGENTS.md at session start
- Agents recognize spec-worthy discussions and offer to capture them
- The behavior is scoped to Spektacular-initialized repos (out-of-repo control)

## Success Metrics Verification

The spec defines three success metrics. Each is mapped to its verification approach:

1. **"Substantial discussions consistently get offered a spec at a point that feels natural rather than premature or too late."**
   - **Manual — captured in the implementation test plan.** This metric depends on subjective judgment ("feels natural") and real conversation flow, which cannot be expressed as an automated behavioral test. The implement workflow will produce a test plan that describes how to evaluate this metric through live agent sessions with varied discussion patterns.

2. **"Users rarely feel surprised by the offer behavior — neither annoyed by over-triggering on trivial work, nor missing an offer they expected on substantial work."**
   - **Manual — captured in the implementation test plan.** This metric is about user perception ("feel surprised", "annoyed", "expected") across a range of discussion types, which requires human evaluation rather than automated assertion. The test plan will describe scenarios for both over-triggering and under-triggering and how to assess user reaction.

3. **"Users rarely need to adjust the default threshold, indicating the out-of-the-box default is well-calibrated."**
   - **Manual — captured in the implementation test plan.** This metric is observable only through production usage patterns (how often users change the config field) and cannot be asserted in a test environment. The test plan will describe how to instrument and monitor threshold adjustments in real deployments to validate calibration.

All three success metrics are flagged for manual verification because they depend on subjective user experience, real conversation dynamics, or production telemetry. The implement workflow will produce concrete test procedures for each once the code exists.

## Test Conventions

The unit tests follow the project's established patterns:
- Table-driven tests with descriptive names (e.g., `TestInstallSpecTriggerSection_CreateFromMissing`)
- Filesystem interaction rooted in `t.TempDir()` (no reads or writes outside the test's scratch directory)
- Expected file contents hand-maintained as string constants (independent oracle, not derived from production renderer)
- Test fixtures for the embedded template substituted via the existing `withSourceFS` helper (reused from `agent_test.go`)

The manual smoke tests follow the pattern established by spec 000023 (memory-redirect): recorded in the PR description, not committed as code, and treated as best-effort verification rather than CI-enforced gates.

## Milestones & Phases

### Milestone 1: Agents recognize spec-worthy discussions and offer to capture them

**What changes**: When a developer works in a Spektacular-initialized repository with any of the supported coding agents (Claude, Codex, Bob), and the conversation produces something substantial enough to warrant a specification — multiple requirements mentioned, a scoped decision reached, a feature described in detail — the agent recognizes the moment and proactively offers to capture the discussion as a spec. The offer is always propose-then-confirm: the agent presents the option, waits for the user's decision (accept, defer, or decline), and only starts the spec workflow on explicit acceptance. The threshold for "substantial enough" is configurable per-project through a new `spec_trigger_threshold` field in `.spektacular/config.yaml` (values: `"strict"`, `"moderate"`, `"lenient"`), with a sensible default (`"moderate"`) that works out-of-the-box. Outside Spektacular-initialized repositories, the same agents continue their normal behavior without offering specs. Running `spektacular init` for any agent installs or updates the instruction that enables this behavior, and re-running init (for the same agent or a different one) updates the instruction in place without duplicating it.

**Validation point**: After running `spektacular init <agent>` in this repository, start a fresh session with that agent and engage in a discussion that describes a feature with multiple requirements. The agent offers to create a spec before proceeding to implementation. Start a session in an unrelated repository with the same agent and confirm it does not exhibit the behavior (out-of-repo control). Re-run `spektacular init <agent>` and confirm the instruction in `AGENTS.md` is updated in place without duplication.

### Milestone 2: Context from the discussion carries forward into the spec

**What changes**: When the user accepts the agent's offer to create a spec, the agent invokes the spec workflow with pre-populated context from the conversation — the overview, requirements, or other sections already discussed — so the user is not forced to re-answer questions the conversation already covered. The spec workflow's first step (overview) detects the pre-populated content, presents it to the user for confirmation or refinement, and only prompts from scratch if nothing was pre-populated. This mechanism works for any section the agent can extract from the discussion, and the spec workflow remains backward-compatible: if no context is pre-populated, the workflow prompts as normal.
### Milestone 2: Context from the discussion persists for resumption

**What changes**: When the spec workflow starts via `spec new`, it clears `.spektacular/context.md` and returns an instruction telling the agent to write the current conversation context to context.md if meaningful context exists. The agent writes detailed discussion context (problem, requirements, constraints, alternatives, exact phrasing), which persists for resumption. The existing skill already tells agents to read context.md on resume, so no template modifications are needed.

**Validation point**: Start a session with a supported agent, discuss a feature in detail, accept the agent's offer to create a spec, and observe that the `new` step returns an instruction to write context. Verify context.md contains the discussion details. Interrupt the workflow (close terminal), restart, and observe that the skill tells the agent to read context.md on resume, enabling the agent to answer prompts without re-asking.

#### - [x] Phase 2.1: Modify new step to clear context.md and return write instruction

Update the `new()` step callback in `internal/steps/spec/steps.go` to: (1) clear `.spektacular/context.md` by writing an empty file after creating the spec scaffold, and (2) return an instruction (via the step template) telling the agent: "Write the current conversation context to `.spektacular/context.md` if meaningful context exists. Capture the full discussion in detail: what problem was identified and why it needs solving, all requirements and constraints discussed, alternatives considered and why rejected, and the user's exact phrasing for key requirements. If no meaningful context exists, leave context.md empty."

*Technical detail:* [context.md#phase-21](./context.md#phase-21-modify-new-step-to-clear-contextmd-and-return-write-instruction)

**Acceptance criteria**:

- [x] When `spec new` runs, the `new()` step creates a fresh empty `.spektacular/context.md` file after creating the spec scaffold
- [x] The `new` step returns an instruction (not silent) telling the agent to write conversation context to context.md
- [x] The instruction specifies detailed format: problem, requirements, constraints, alternatives, exact phrasing
- [x] The instruction includes caveat: skip if no meaningful context exists
- [x] When resuming via `spec goto`, the `new` step does not run (context.md preserved)
- [x] Existing spec workflows continue to work unchanged
- [x] The template's pre-population detection uses mustache conditionals (e.g., `{{#overview}}...{{/overview}}`) and does not require changes to the step callback or workflow state machine
- [x] Existing spec workflows (those not using pre-population) continue to work unchanged

## Open Questions

No open questions remain. All design decisions have been resolved during planning:

- **Configuration mechanism**: Resolved to `spec_trigger_threshold` field in `config.yaml` with validation
- **Instruction delivery**: Resolved to managed section in AGENTS.md following spec 000023 pattern
- **Context carry-forward**: Resolved to use existing `spec new --data`/`--stdin`/`--file` flags
- **Defer/decline tracking**: Resolved to agent-side conversation management
- **Threshold interpretation**: Resolved to natural-language prose in instruction template (agents interpret "strict"/"moderate"/"lenient")
- **Template wording**: Will be drafted in Phase 1.2 following the pattern from `memory-context.md`
- **Pre-population extent**: Resolved to start with overview step (Phase 2.1), other sections optional for future expansion
- **Test coverage**: Resolved to unit tests for installer idempotency, manual smoke tests for agent behavior

The implementer should proceed with the phases as defined. If any assumption proves incorrect during implementation (e.g., agents do not interpret the threshold prose consistently, or the pre-population mechanism does not work as expected), STOP and ask the user before adjusting the approach.

## Out of Scope

- **Polishing the requirements-gathering step's interview UX** — propose-then-confirm drafting, surfacing rephrasing out loud, prompting for completeness gaps. Deferred to a separate future spec (noted in spec § Non-Goals).

- **Adding a user-acceptance walkthrough to the plan workflow's verification step** — a separately identified gap, not addressed here (noted in spec § Non-Goals).

- **A durable changelog/context artifact for downstream doc/blog generation** — deferred to its own separate spec, to be sequenced before this one's implementation (noted in spec § Non-Goals).

- **Recognizing spec-worthy discussion across multiple separate conversations or sessions** — this feature operates only within a single, ongoing conversation. Cross-session recognition would require persistent conversation-ID tracking, which is not part of this design (noted in spec § Non-Goals).

- **Automatically creating a spec without the user's explicit acceptance of the offer** — the assistant always offers and waits for a decision; it never starts a spec workflow unilaterally (noted in spec § Non-Goals).

- **Pre-population support for all spec sections** — Milestone 2 implements pre-population detection only for the overview step. Extending this to requirements, acceptance criteria, constraints, and other sections is left for future work if the pattern proves valuable. The mechanism is designed to be extensible (step templates check for workflow data keys), but only overview is implemented in this plan.

- **Runtime detection command** — rejected at the architecture step in favor of instruction-based recognition. A `spektacular detect-spec-worthy` CLI command would add latency and assume agents can pass conversation history, which is not guaranteed across all three supported agents (documented in research.md § Alternatives considered and rejected).

- **Hardcoded agent heuristics** — rejected at the architecture step because it violates the spec's core constraint that "the trigger threshold must be user-configurable". Fixed rules in agent instructions cannot be adjusted per-project without editing the instructions themselves (documented in research.md § Alternatives considered and rejected).

- **Enforcing adherence** — no hook, lint, pre-commit check, or runtime guardrail that detects an agent writing to per-user memory is added. Adherence is best-effort per agent (noted in spec § Non-Goals).

- **Migrating existing per-user memory entries into the shared knowledge store** — only persistence triggered after the redirect lands is affected. Existing per-developer memory stays where it is (noted in spec § Non-Goals).

## Changelog

### 2026-07-08 — Phase 2.1: Modify new step to clear context.md and return write instruction

**What was done**: Modified the spec workflow's `new()` step to clear `.spektacular/context.md` after creating the spec scaffold and return an instruction (via new template `templates/steps/spec/00-new.md`) telling the agent to write detailed conversation context to context.md. The instruction specifies capturing problem, requirements, constraints, alternatives, and exact phrasing, with a caveat to skip if no meaningful context exists.

**Deviations**: None. Implementation followed the plan exactly.

**Files changed**:
- `templates/steps/spec/00-new.md` (created)
- `internal/steps/spec/steps.go` (modified new() callback)
- `internal/steps/spec/steps_test.go` (added 5 new tests)

**Discoveries**: The new() step now uses writeStep() instead of returning "overview" directly, which changes the FSM behavior slightly - the workflow stays at "new" state after the first transition and returns an instruction. Tests were updated to reflect this. The relative path `.spektacular/context.md` resolves correctly against the current working directory (project root) when running `go run . spec new`.
