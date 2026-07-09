# Research: 000032_spec-workflow-pair-programming-enhancements

## Alternatives considered and rejected

### Alternative 1: Hardcoded heuristic in agent system prompt
**What it is**: Embed fixed rules directly in each agent's system prompt (CLAUDE.md, AGENTS.md, .bob/rules/) defining when to offer a spec, e.g. "after 3+ requirements mentioned."

**Why rejected**: Violates the spec's constraint that "the trigger threshold must be user-configurable." A hardcoded heuristic cannot be dialed per-project without editing the instructions themselves, which is not a configuration mechanism.

**Citation**: Spec § Constraints: "The trigger threshold must be user-configurable — the assistant cannot hardcode a single fixed sensitivity for offering a spec."

### Alternative 2: Runtime detection via CLI command
**What it is**: A `spektacular detect-spec-worthy` command that analyzes conversation history (passed via stdin/file) and returns a boolean decision.

**Why rejected**: Requires the agent to invoke a detection command every turn (latency), and assumes the CLI can receive full conversation history — not guaranteed to be practical across all three supported agents. The spec requires the feature to work for all three without agent-runtime-specific plumbing.

**Citation**: Spec § Overview: recognize-and-offer is additive to the existing explicit entry point, not a replacement requiring new runtime wiring.

### Alternative 3 (superseded — this was the original plan's core mistake): CLI data-passing / mustache pre-population for context carry-forward
**What it is**: When the user accepts the offer, the agent invokes `spec new --data '{"overview":"<text>"}'` (or `--stdin`), and step templates (e.g. `templates/steps/spec/01-overview.md`) are modified with mustache conditionals (`{{#overview}}...{{/overview}}`) to detect and present the pre-populated value instead of prompting from scratch.

**Why rejected**: Traced the actual render path — `overview()` (`internal/steps/spec/steps.go:95-99`) calls `writeStep(..., data, ..., nil)`, and `stepkit.WriteStepResult` (`internal/stepkit/stepkit.go:55-94`) builds its mustache `vars` map only from `pathVars`, `step`/`title`/`next_step`/`config`, and `req.Extra` (lines 78-85) — the raw `workflow.Data` map is **never merged into `vars`**. A value passed via `--data '{"overview":...}'` would land in `data` but never reach the template as `{{overview}}` unless the callback explicitly forwarded it via `Extra` (as `verification()`/`finished()` do for `spec_template`/`spec_unwritten` — steps.go:137-148, 150-168). This alone made the original design non-functional as specified.

But the deeper problem, independent of that bug: **the whole mechanism is unnecessary**. The agent proposing the offer, accepting it, and starting `spec new` is the *same agent, same live conversation turn* — it already holds the full discussion in its own context. It does not need to serialize anything through the CLI to "give itself" that context back; it can simply read the existing "ask the user..." instruction in each step template and, because it already knows the answer, propose a draft for the user to confirm/refine rather than asking cold. No new `--data` keys, no template conditionals, no `Extra`-forwarding plumbing anywhere.

**Citation**: `internal/steps/spec/steps.go:95-99`, `internal/stepkit/stepkit.go:55-94, 78-85`, `templates/steps/spec/01-overview.md:7-12` (confirmed via direct codebase-analyzer research, 2026-07-09).

### Alternative 4: Separate instruction file for threshold behavior
**What it is**: A new file (e.g. `.spektacular/agent-instructions/spec-trigger.md`) holding the threshold-based offer logic, referenced from each agent's entrypoint.

**Why rejected**: Fragments the instruction surface. AGENTS.md already holds managed sections for cross-agent instructions (spec 000023's precedent) and all three agents already load it — a second file adds maintenance surface with no benefit.

**Citation**: `internal/agent/memory_context.go:24-26` (installs into AGENTS.md); plan 000023 § Architecture: "All three supported agents already load AGENTS.md at session start."

## Chosen approach — evidence

### Evidence 1: Configuration via config.yaml
**File**: `internal/config/config.go:105-113` (Config struct), `116-156` (`NewDefault()`), `178-192` (top-level `Validate()`), `196-209` (`SpecConfig.Validate()` — the enum-check pattern to copy).

The `Config` struct is a flat set of top-level fields (`Command`, `Agent`, plus nested `Debug`/`Spec`/`Plan`/`Changelog`/`Knowledge` sections). A new top-level `SpecTriggerThreshold string` field (yaml `spec_trigger_threshold`) fits directly alongside `Command`/`Agent` (after line 107). `NewDefault()` sets defaults in a struct literal (`Command: "spektacular"` at line 118) — add `SpecTriggerThreshold: "moderate"` the same way. The enum-validation pattern lives at lines 203-207 (`SpecConfig.Validate()`'s `IDMethod` switch): a `switch` over the field with `""` accepted (unset/default) and a `default:` branch returning `fmt.Errorf("<field> must be one of %q, %q, or %q", ...)`. The new field's validation should copy this exact shape: `switch c.SpecTriggerThreshold { case "", "strict", "moderate", "lenient": default: return fmt.Errorf("spec_trigger_threshold must be one of %q, %q, or %q", ...) }`, called from the top-level `Validate()` (lines 178-192) since the field is top-level, not nested.

Existing test convention (`internal/config/config_test.go`): `TestNewDefault_HasExpectedDefaults` (asserts defaults via `require.Equal`) and `TestFromYAMLFile_UnknownSpecIDMethodReturnsError` (writes invalid YAML, asserts `require.Error` + `require.Contains(err.Error(), "spec.id_method")`) — naming convention `TestFromYAMLFile_Unknown<Field>ReturnsError`.

**What this supports**: A `spec_trigger_threshold` field in `config.yaml`, validated the same way every other enum-like field is, satisfying the spec's "existing configuration mechanism" constraint with zero new machinery.

### Evidence 2: Agent instructions via AGENTS.md managed section
**File**: `internal/agent/memory_context.go:17-18` (constants), `25-56` (`installMemoryContextSection` algorithm), `58-81` (`locateMemoryContextSection`), `86-96`/`101-116` (append/replace helpers), `120-144` (atomic write); call sites at `claude.go:25`, `codex.go:17`, `bob.go:20`; template at `templates/agents/memory-context.md` (15 lines, static prose, heading `## Memory & Context`, no mustache variables used despite `command` being passed in).

`installMemoryContextSection(projectPath string, cfg config.Config, out io.Writer) error` is the exact structural template to copy: read embedded template → render with mustache (`map[string]string{"command": cfg.Command}`) → ensure trailing newline → read `AGENTS.md` → if missing, create; if present and heading absent, append (trim + `"\n\n"` + rendered); if present and heading found (`locateMemoryContextSection` scans for the exact trimmed heading line, then the next `"# "`/`"## "` line marks the end), replace that span in place. All writes go through `writeFileAtomic` (temp file + `os.Rename`). Each agent's `Install()` calls this after its skill/command-wrapper installs, as a one-line tail call or `if err := ...; err != nil { return err }`.

Test coverage to mirror (`memory_context_test.go`): create-from-missing, append-after-existing-content-without-heading, idempotent-rerun (byte-identical), preserve-surrounding-content (unrelated headings before/after untouched), template-change-picked-up (different `cfg.Command` re-renders), cross-agent-idempotency (claude→codex→bob against one shared file leaves exactly one heading).

**What this supports**: `installSpecTriggerSection(projectPath, cfg, out)` following this exact algorithm with a new heading (`## Spec-Worthy Discussion Recognition`), a new template path (`templates/agents/spec-trigger.md`), and the same single template variable (`command`) as the pattern it clones — zero deviation from a proven, already-tested pattern. (The threshold value itself is deliberately *not* a template variable — see Open Assumption 2 below, settled during the architecture step to be a dynamic read instead.)

### Evidence 3: The "propose from memory, user confirms" carry-forward mechanism requires zero CLI/template changes
**File**: `internal/steps/spec/steps.go:66-99` (`new()`/`overview()`), `internal/stepkit/stepkit.go:55-117` (`WriteStepResult`, `GetString`), `templates/steps/spec/01-overview.md:7-12` (the "ask the user" prose), `cmd/spec.go:110-147, 165, 200-227, 308, 393` (`--data`/`--stdin`/`--file` flags).

Confirmed directly: nothing in `workflow.Data`, `stepkit.WriteStepResult`, or the `new()`/`overview()` callbacks constrains *how* the agent arrives at the answer to a step's prompt. The CLI's entire job is to render and return an instruction string (e.g. "Ask the user to describe this feature in 2-3 sentences: • What is being built? • What problem does it solve? • Who benefits?" — `01-overview.md:7-11`). Whether the agent asks that question cold or already knows the answer from a live conversation and proposes a draft for the user to confirm is purely a matter of agent behavior when interpreting that text — no code path distinguishes the two, and none needs to.

`--data`/`--stdin`/`--file` (`cmd/spec.go`) are generic, pre-existing input mechanisms — `--data` supplies only spec identity (`name`/`id`) to `spec new`; `--stdin`/`--file` stash arbitrary text into `workflow.Data` under a key that nothing in `new()`/`overview()` reads. They are unrelated to and unnecessary for this carry-forward behavior.

**What this supports**: Context carry-forward is entirely an *agent instruction*, delivered via the same AGENTS.md managed section as the recognition/offer instruction. It tells the agent: after the user accepts, start `spec new` and drive its steps using what you already know from the conversation, proposing drafts for confirmation rather than asking from scratch. No Go changes, no template changes, no new workflow data keys — this collapses what was an earlier, separate milestone (with its own Go and template changes) down to a single already-shipped mechanism (context.md clearing, kept as-is for resume) plus prose in this plan's one instruction.

### Evidence 4: `.spektacular/context.md` clearing/write-instruction is a *resume* mechanism, distinct from live carry-forward
**File**: `internal/steps/spec/steps.go:64-93` (`new()` — comment at 64-65: "creates the spec file, clears context.md, and returns an instruction to write conversation context before proceeding to overview"; clearing at lines 83-89), `templates/steps/spec/00-new.md` (the rendered instruction: capture problem/motivation, requirements/constraints, alternatives rejected, exact phrasing — or leave empty if nothing meaningful).

This is a **disk relay for a discontinuous/future session** — `context.md` is git-tracked and read back by a later, cold agent session that has no live memory of the original conversation (`01-overview.md`'s "Before you advance" footer: "a resumed session reads it back to pick up where you left off"). It is refreshed at the end of every step so an interrupted spec workflow can resume without re-asking. This is fundamentally different from the proactive-capture scenario in this plan: there, the *same* agent, in the *same* live turn, still holds the conversation in its own context window — no read-back of `context.md` happens in that moment (though the agent still populates it going forward, for the benefit of any later resume, per the existing instruction).

**What this supports**: The already-shipped `new()`/`00-new.md` behavior stays exactly as-is; it is not the mechanism this plan needs to add for live carry-forward, and should not be modified or reverted. The plan's job is only to add the *recognition + offer + drive-from-memory* instruction, not to touch the CLI/workflow layer for carry-forward at all.

### Evidence 5: Defer/decline is agent behavior, not CLI state
**File**: `internal/workflow/workflow.go` (state machine has no "discussion" or "offer" concept — only step/data), `templates/steps/spec/01-overview.md` (agent-facing instruction surface).

The CLI has no concept of "discussion ID" or "offer declined" — only whether a workflow is in progress. Defer/decline tracking must live in the agent's own conversation memory (session-scoped), consistent with the spec's constraint that this is a single-conversation feature (Non-Goal: "Recognizing spec-worthy discussion across multiple separate conversations or sessions").

**What this supports**: The AGENTS.md instruction tells the agent to remember, within the conversation, whether the user deferred (may re-offer later as the discussion develops) or declined outright (suppress re-offering for that topic for the rest of the conversation) — no new workflow state, no new CLI command.

## Files examined

- `internal/config/config.go:105-113, 116-156, 178-192, 196-209` — Config struct shape, defaults, and the exact enum-validation pattern (`SpecConfig.IDMethod`) to copy for the new threshold field.
- `internal/config/config_test.go` — test naming/structure convention for enum-field defaults and validation-error tests.
- `internal/agent/memory_context.go:17-144` — full managed-section installer algorithm (locate/replace/append/create, atomic write) to copy verbatim for the new spec-trigger section.
- `internal/agent/memory_context_test.go` — six test cases defining the required coverage shape for the new installer's tests.
- `internal/agent/claude.go:25`, `codex.go:17`, `bob.go:20` — exact call-site pattern and ordering for wiring a new installer call into each agent's `Install()`.
- `templates/agents/memory-context.md` — structure/length/heading-style reference for the new `spec-trigger.md` template.
- `internal/steps/spec/steps.go:1-99` — `new()`/`overview()` callbacks; confirms `workflow.Data` is never merged into template `vars`, and that this doesn't matter because carry-forward doesn't need CLI data-passing at all.
- `internal/stepkit/stepkit.go:55-127` — `WriteStepResult`/`RenderTemplate`; confirms the template-rendering pipeline's exact inputs (`pathVars`, `step`/`title`/`next_step`/`config`, `Extra` — never raw `data`).
- `templates/steps/spec/00-new.md`, `01-overview.md` — exact current instruction prose; `01-overview.md:7-12` is the "ask the user" text the new AGENTS.md instruction must tell the agent to answer from memory-with-confirmation rather than cold, when arriving with live context.
- `cmd/spec.go:110-147, 165, 200-227, 308, 393` — confirms `--data`/`--stdin`/`--file` are generic, unrelated to this feature's carry-forward mechanism.
- `.spektacular/plans/000033_spec-workflow-output-changelog/plan.md` — unrelated mechanism (implement-workflow *output* changelog, not spec-input carry-forward); no overlap with this plan.
- `.spektacular/knowledge/` (via `go run . knowledge always-applied` and `search`) — no populated conventions/glossary entries yet; one relevant learning found (see below).

## External references

None. The feature is entirely internal to Spektacular's agent instruction and configuration mechanisms.

## Prior plans / specs consulted

- **Plan 000023_context-over-memory** — the direct precedent for the managed-section pattern: a markdown heading identifies the section, an installer function writes it idempotently, all three agents load it natively. This plan's `installSpecTriggerSection` is a structural clone of `installMemoryContextSection` from this prior work.
- **Spec 000022_spek-knowledge-skill** — established the propose-then-confirm pattern for agent-driven writes/actions. The spec-trigger instruction (offer, wait for accept/defer/decline) and the carry-forward instruction (propose a draft from memory, wait for user confirmation) both follow this same shape.
- **Plan 000033_spec-workflow-output-changelog** (skimmed for overlap) — touches the spec/implement workflow's step machinery but solves a different problem (durable output record after implementation, not input carry-forward before/during spec creation). No overlap; no shared code paths.

## Open assumptions

1. **Threshold values remain prose labels, not numeric scores.** The spec doesn't define "strict"/"moderate"/"lenient" in concrete terms. This plan leaves the interpretation to the AGENTS.md instruction's natural-language guidance. If different agents converge on visibly different behavior for the same setting in practice, that's a signal the instruction prose needs sharper language — not a signal to add code-side heuristics.
2. **Settled during the architecture step (no longer an assumption): the instruction tells the agent to read `spec_trigger_threshold` from `.spektacular/config.yaml` dynamically, at the moment it decides whether to offer** — not baked into the rendered instruction at `spektacular init` time the way `command` is for memory-context.md. This was chosen over the init-time-bake alternative so a config change takes effect immediately, without requiring a re-init. Trade-off accepted: the instruction asks the agent to read a project file at decision time, a kind of behavior no existing managed section currently asks for — a one-line addition to the instruction prose, not new code.
3. **Carry-forward drafts, not silently fills.** "Propose the answer, let the user confirm/refine" is the intended shape for every step the agent has enough conversation context to answer — not "skip the question and record my own guess unchallenged." This must be explicit in the instruction prose to satisfy the spec's requirement that answers are still user-owned decisions.
4. **Defer vs. decline stays in agent memory, scoped to the single conversation.** No new persistence is added for this. If a later session needs to know a discussion was declined, that's out of scope per the spec's Non-Goals (cross-session recognition is explicitly excluded).

## Rehydration cues

To rebuild this research from cold:

1. **Load always-applied knowledge**: `go run . knowledge always-applied` (conventions and glossary — currently only category placeholders).
2. **Search for the managed-section pattern**: `rg "installMemoryContextSection|AGENTS\.md" internal/agent/` and read `internal/agent/memory_context.go` + `internal/agent/memory_context_test.go` in full — this is the direct template for the new installer.
3. **Search for the config enum pattern**: `rg "IDMethod" internal/config/config.go` to find the exact validation style to copy.
4. **Read the spec workflow's rendering path**: `internal/steps/spec/steps.go`, `internal/stepkit/stepkit.go` — confirms `workflow.Data` never reaches templates directly, which is *why* an earlier CLI-data-passing design for carry-forward was wrong, and *why* the corrected design needs no code changes there at all.
5. **Re-read `templates/steps/spec/01-overview.md`** to see the exact "ask the user" prose the new instruction must tell the agent to handle differently when it already has context.
