# Research: 000032_spec-workflow-pair-programming-enhancements

## Alternatives considered and rejected

### Alternative 1: Hardcoded heuristic in agent system prompt
**What it is**: Embed fixed rules in each agent's system prompt (CLAUDE.md, AGENTS.md, .bob/rules/) defining when to offer a spec, e.g. "after 3+ requirements mentioned" or "when user says 'let's build'".

**Why rejected**: Violates the spec's core constraint that "the trigger threshold must be user-configurable". A hardcoded heuristic cannot be dialed up or down per-project without editing the agent instructions themselves, which is not a configuration mechanism. Additionally, different organizations need different sensitivities (some want specs for small bug fixes, others only for substantial features), and a single fixed rule cannot serve both.

**Citation**: Spec § Constraints: "The trigger threshold must be user-configurable — the assistant cannot hardcode a single fixed sensitivity for offering a spec."

### Alternative 2: Runtime detection via CLI command
**What it is**: Add a `spektacular detect-spec-worthy` command that analyzes conversation history (passed via stdin or file) and returns a boolean "should offer spec" decision based on configured thresholds.

**Why rejected**: Requires the agent to explicitly invoke a detection command during every turn, which adds latency and complexity. More critically, it assumes the agent runtime can pass conversation history to the CLI, which is not guaranteed across all three supported agents (Claude, Codex, Bob). The spec requires the feature to work for all three without agent-specific runtime changes.

**Citation**: Spec § Overview: "without any agent runtime change"; Spec § Constraints: "Must not disrupt or replace the existing explicit spec-workflow entry point — the recognize-and-offer behavior is additive".

### Alternative 3: Separate instruction file for threshold behavior
**What it is**: Create a new file (e.g. `.spektacular/agent-instructions/spec-trigger.md`) containing the threshold-based offer logic, and reference it from each agent's entrypoint.

**Why rejected**: Adds a new file that must be maintained separately from AGENTS.md, which all three agents already load. The existing memory-redirect pattern (shipped under spec 000023) demonstrates that AGENTS.md can hold managed sections for cross-agent instructions. Introducing a separate file fragments the instruction surface without clear benefit.

**Citation**: internal/agent/memory_context.go:24-26 (installMemoryContextSection writes to AGENTS.md); plan 000023_context-over-memory § Architecture: "All three supported agents already load AGENTS.md at session start".

## Chosen approach — evidence

### Evidence 1: Configuration via config.yaml
**File**: internal/config/config.go:1-280

The project's configuration mechanism is `config.yaml` at `.spektacular/config.yaml`, loaded via `config.FromYAMLFile`. The `Config` struct is the single source of truth for project settings (spec directory, plan directory, knowledge sources, debug flags). Adding a new top-level field (e.g. `spec_trigger_threshold: string`) follows the established pattern and satisfies the spec's constraint that "the threshold setting must be stored in the project's existing configuration mechanism".

**What this supports**: A `spec_trigger_threshold` field in `config.yaml` (with values like `"strict"`, `"moderate"`, `"lenient"`) can be read by the agent via instructions in AGENTS.md, allowing per-project threshold configuration without introducing a new config file.

### Evidence 2: Agent instructions via AGENTS.md managed section
**File**: internal/agent/memory_context.go:24-135

The memory-redirect feature (spec 000023) demonstrates the pattern for adding cross-agent instructions: a managed section in AGENTS.md, written by `spektacular init`, identified by its markdown heading (`## Memory & Context`), and updated idempotently on re-init. The same pattern applies here: a new managed section (e.g. `## Spec-Worthy Discussion Recognition`) can hold the threshold-based offer logic, referencing the configured threshold from `config.yaml`.

**What this supports**: The instruction to "recognize when a discussion warrants a spec and proactively offer to capture it" can be delivered as prose in AGENTS.md, with the threshold value interpolated from config at init time (or read dynamically by the agent if the instruction references `config.yaml` directly).

### Evidence 3: Spec workflow accepts pre-populated data
**File**: cmd/spec.go:95-100, 195-200

The `spec new` command accepts arbitrary key-value pairs via `--data` (JSON) and `--stdin`/`--file` flags, storing them in the workflow's data map. The `spec goto` command similarly accepts extra data beyond the required `step` field. This means the spec workflow can receive pre-populated context (e.g. `overview`, `requirements`) from the agent before the first step runs, satisfying the spec's requirement that "context carries forward into the spec".

**What this supports**: When the agent offers to create a spec and the user accepts, the agent can invoke `spec new --data '{"name":"...", "overview":"<already-discussed-content>"}'` (or use `--stdin overview` to pass a longer body), and the spec workflow will have that context available when the `overview` step runs. The step template can detect pre-populated data and skip re-asking.

### Evidence 4: Step templates can detect pre-populated content
**File**: templates/steps/spec/01-overview.md:1-30

The `overview` step template instructs the agent to "ask the user to describe this feature in 2-3 sentences" and "capture their response". If the workflow data already contains an `overview` key (passed via `--data` or `--stdin`), the template can be modified to check for it first and only prompt if absent. This is the mechanism for "carrying forward already-established context" without forcing the user to re-answer.

**What this supports**: The spec workflow does not need structural changes to support pre-populated context. The step templates can be updated to check `{{overview}}` (or similar mustache variable) and skip prompting if present, while still allowing the user to refine it.

### Evidence 5: Defer/decline is agent behavior, not CLI state
**File**: templates/steps/spec/01-overview.md:1-30, internal/workflow/workflow.go:1-200

The spec's defer/decline requirements ("the user can defer the offer while investigation continues" / "the user can decline the offer outright") are agent-side conversation management, not workflow state. The CLI has no concept of "offer declined" — it only knows whether a workflow is in progress or not. The agent must track whether an offer was declined within the current conversation and suppress re-offering for the same topic.

**What this supports**: The AGENTS.md instruction can tell the agent to "remember if the user declined the offer for this discussion and do not re-offer for the same topic", relying on the agent's own conversation memory (or `.spektacular/context.md` for cross-session persistence) rather than introducing new CLI state.

## Files examined

- `internal/config/config.go:1-280` — Configuration loading and validation. Confirmed `config.yaml` is the single project config file; adding a new top-level field follows the established pattern.
- `internal/agent/memory_context.go:24-135` — Memory-redirect implementation. Demonstrates the AGENTS.md managed-section pattern for cross-agent instructions.
- `cmd/spec.go:95-100, 195-200` — Spec workflow entry points. Confirmed `--data`, `--stdin`, and `--file` allow pre-populating workflow data before the first step.
- `templates/steps/spec/01-overview.md:1-30` — Overview step template. Shows where to add pre-populated-context detection logic.
- `internal/workflow/workflow.go:1-200` — Workflow state machine. Confirmed defer/decline is not workflow state; it's agent conversation management.
- `internal/steps/spec/steps.go:17-175` — Spec step definitions. Confirmed steps receive workflow data via `workflow.Data` interface; pre-populated keys are accessible.
- `templates/agents/memory-context.md:1-20` — Memory-redirect instruction template. Model for the new spec-trigger instruction section.
- `.spektacular/knowledge/conventions.md`, `glossary.md` — Always-applied knowledge. No existing conventions about spec triggers or thresholds.

## External references

None. The feature is entirely internal to Spektacular's agent instruction and configuration mechanisms.

## Prior plans / specs consulted

- **Plan 000023_context-over-memory** (`.spektacular/plans/000023_context-over-memory/plan.md`) — Learned the AGENTS.md managed-section pattern: a markdown heading identifies the section, `installMemoryContextSection` writes it idempotently, and all three agents load it natively. The same pattern applies to the spec-trigger instruction.
- **Spec 000022_spek-knowledge-skill** (`.spektacular/specs/000022_spek-knowledge-skill.md`) — Learned the propose-then-confirm pattern for agent-driven writes. The spec-trigger instruction should follow the same "offer, wait for confirmation" flow rather than auto-starting a spec.

## Open assumptions

1. **Threshold values are prose labels, not numeric scores.** The spec does not define what "strict", "moderate", or "lenient" mean in concrete terms (e.g. "3+ requirements" vs "5+ requirements"). The plan must either define these mappings or leave them as natural-language guidance for the agent to interpret. If the latter, the instruction must be clear enough that different agents (Claude, Codex, Bob) converge on similar behavior for the same threshold setting.

2. **The agent can read `config.yaml` or the instruction interpolates the threshold at init time.** Two implementation paths: (a) the AGENTS.md instruction says "read `.spektacular/config.yaml` and check the `spec_trigger_threshold` field", or (b) `spektacular init` renders the instruction with the current threshold value baked in (e.g. `{{threshold}}`), requiring re-init to pick up config changes. The plan must choose one and document the trade-off (dynamic read = always current, baked-in = simpler instruction but stale until re-init).

3. **"Same discussion" is defined by the agent's conversation memory, not a CLI-tracked identifier.** The spec requires "the user can decline the offer outright" and "the assistant must drop the offer for the remainder of that discussion". The CLI has no concept of "discussion ID" — it only tracks workflow state. The agent must use its own memory (or `.spektacular/context.md` for cross-session persistence) to remember "user declined spec for feature X" and suppress re-offering. If this assumption is wrong (i.e. the CLI must track declines), the plan must add new state fields to `workflow.State`.

4. **Pre-populated context is optional, not required.** The spec says "carry forward already-established context" but does not mandate that *all* sections must be pre-filled. The plan assumes the agent can pass as much or as little context as the conversation has established (e.g. only `overview`, or `overview` + `requirements`), and the spec workflow will prompt for any missing sections. If the spec requires *all* sections to be pre-filled before starting, the plan must add validation to `spec new`.

## Rehydration cues

To rebuild this research from cold:

1. **Load always-applied knowledge**: `go run . knowledge always-applied` (conventions and glossary).
2. **Search for configuration patterns**: `rg "config\.yaml|Config struct" internal/config/` to understand the project's config mechanism.
3. **Search for AGENTS.md usage**: `rg "AGENTS\.md|installMemoryContextSection" internal/agent/` to find the managed-section pattern.
4. **Read spec workflow entry points**: `internal/steps/spec/steps.go`, `cmd/spec.go` to understand how data flows into the workflow.
5. **Read step templates**: `templates/steps/spec/*.md` to see where pre-populated context would be consumed.
6. **Consult prior plans**: `go run . plan file read 000023_context-over-memory/plan.md` for the AGENTS.md pattern; `go run . spec file read 000022_spek-knowledge-skill.md` for propose-then-confirm.
