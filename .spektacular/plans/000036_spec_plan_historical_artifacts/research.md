# Research: 000036_spec_plan_historical_artifacts

## Alternatives considered and rejected

- **Filesystem-level or tool-level access blocks on `.spektacular/specs/` and `.spektacular/plans/`.** Would trivially satisfy the "no discovery reads" requirement but breaks the archaeology exception (requirement #3) and the "workflow may access its own live artifact" exception (requirement #7). Explicitly rejected by the spec's Constraint (`.spektacular/specs/000036_spec_plan_historical_artifacts.md:63`).
- **Storing the rule in `.spektacular/config.yaml` under a new `agent_rules` key.** The config file is not surfaced to fresh agent sessions the way `AGENTS.md` is (`AGENTS.md:1-52` is auto-loaded via `CLAUDE.md`'s `@AGENTS.md` import at `internal/agent/claude.go:38`). Storing rules in config would create a second, hidden instruction surface and violate the spec's "durable, session-auto-picked-up" requirement (`.spektacular/specs/000036_spec_plan_historical_artifacts.md:42-43`).
- **Only patching every affected step template (no AGENTS.md change).** Would leave ad-hoc, unrelated-skill, and general-exploration sessions uncovered (requirement #4 — "applies everywhere", `.spektacular/specs/000036_spec_plan_historical_artifacts.md:36-37`). Step templates are only reached inside their own workflow steps, so this approach cannot satisfy the "fresh agent session already knows the rule" acceptance criterion (`.spektacular/specs/000036_spec_plan_historical_artifacts.md:91-92`).
- **Adding a paragraph to each `spek-*/SKILL.md` (top-of-skill notes).** Duplicative once AGENTS.md carries the rule — the SKILL.md files are consumed inside their own workflow context, and AGENTS.md already reaches every session. Rejected as maintenance surface without additional coverage.
- **Retroactive metadata / status markers on the 36 existing specs and ~30 existing plans.** Out of scope by spec framing (this spec is behavioral, not artifact-format). The archaeology framing works on today's file layout without markers because agents can distinguish current-state questions from historical questions via user intent.

## Chosen approach — evidence

Two-surface fix: a new managed section in `AGENTS.md` (the always-loaded, session-global rule) plus narrowly-scoped clarifiers in the handful of step templates where existing instructions to read prior specs/plans could look like they contradict the new rule.

Evidence for the AGENTS.md pattern:
- `internal/agent/memory_context.go:16-56` — install function pattern (`installMemoryContextSection`) with locate/append/replace helpers.
- `internal/agent/spec_trigger.go:16-57` — parallel install function (`installSpecTriggerSection`).
- `templates/agents/memory-context.md:1-14` — template file pattern (heading + managed-by disclaimer blockquote + prose).
- `templates/agents/spec-trigger.md:1-31` — template file pattern with `{{command}}` mustache.
- `internal/agent/claude.go:25-30` — wiring point in Claude agent's `Install()`.
- `internal/agent/bob.go:20-23` — wiring point in Bob agent's `Install()`.
- `internal/agent/codex.go:17-20` — wiring point in Codex agent's `Install()`.
- `internal/agent/memory_context_test.go:1-153` — test coverage pattern (create-from-missing, append-after-tessl, idempotent, preserves surrounding, picks up template change, cross-agent idempotency).
- `internal/agent/spec_trigger_test.go:1-157` — parallel test file to mirror.
- `templates/templates.go` (`//go:embed all:*`) — new template gets embedded automatically; no manifest change needed.

Evidence for the surgical step-template clarifiers:
- `templates/steps/plan/02-discovery.md:18` — the plan discovery step explicitly tells agents to `plan file list` / `spec file list` and read prior plans/specs. Needs a small framing tweak so the reads are described as historical archaeology in service of new-plan research, not as current-state discovery.
- `templates/steps/implement/01-read_plan.md:7-12`, `02-analyze.md:7-15`, `06-update_plan.md:17`, `07-update_changelog.md:9-46`, `09-test_plan.md:10-40`, `10-update_feature_changelog.md:7-11`, `11-reconcile_spec.md:7-11`, `12-finished.md:21` — all read the plan and/or spec via CLI as the *owning workflow's own live artifact*. These are covered by requirement #7's exception. No content change is strictly required — the exception itself is stated in AGENTS.md — but a brief acknowledgment in `01-read_plan.md` (which is the first read) that this is the owning workflow's own artifact will prevent readers from perceiving contradiction.

Evidence for the changelog note requirement:
- `templates/steps/implement/10-update_feature_changelog.md:1-40` — the step that writes the per-feature changelog entry (`.spektacular/changelog/<n>_<name>.md`) at implement time.
- `.spektacular/changelog/000035_plan-walkthrough-conversation.md:1-32` — canonical shape of a per-feature changelog entry (What was built / Why it matters / Deviations from the plan).
- The plan must therefore include an explicit checkbox in the appropriate phase (or as an acceptance criterion) that the changelog entry contains the "process document, not product document" website-docs follow-up note.

## Files examined

- `AGENTS.md:1-52` — Current agent-instruction file at repo root; contains two `spektacular init`-managed sections (Memory & Context, Spec-Worthy Discussion Recognition) plus the tessl-managed header.
- `CLAUDE.md` — `@AGENTS.md` re-export line only; ensures Claude Code loads AGENTS.md automatically.
- `.tessl/RULES.md` — externally-managed placeholder file; not part of this feature's surface.
- `templates/agents/memory-context.md:1-14` — model template: heading + managed-by disclaimer + prose body.
- `templates/agents/spec-trigger.md:1-31` — model template using `{{command}}` mustache variable.
- `internal/agent/memory_context.go:16-144` — full install pattern (constants, install, locate, append, replace, atomic writer).
- `internal/agent/spec_trigger.go:16-117` — parallel install pattern.
- `internal/agent/claude.go:14-83` — Claude agent registration and Install() calls, plus `ensureClaudeImportsAGENTS` that maintains the `@AGENTS.md` import.
- `internal/agent/bob.go:9-32` — Bob agent Install() wiring.
- `internal/agent/codex.go:9-25` — Codex agent Install() wiring.
- `internal/agent/memory_context_test.go:1-153` — five-test pattern to mirror for new install function.
- `internal/agent/spec_trigger_test.go:1-157` — parallel test file.
- `internal/agent/instruction_surface_test.go:1-109` — repo-level guard against stdin/heredoc CLI surface leaking into instruction files; the new template must respect it (it will — no stdin references needed).
- `internal/agent/skills.go` — hardcoded `workflowSkills` list; not affected by this change (skills unchanged).
- `templates/steps/plan/02-discovery.md:1-58` — plan discovery step template; line 18 is the prior-plans/specs read that needs a framing tweak.
- `templates/steps/implement/01-read_plan.md:7-12`, `02-analyze.md:7-15`, `10-update_feature_changelog.md:7-11`, `11-reconcile_spec.md:7-11`, `12-finished.md:21` — implement-workflow read sites; owning-workflow exception covers them.
- `templates/steps/implement/10-update_feature_changelog.md:1-40` — the step that writes per-feature changelog entries; this feature's own changelog note lands here at implement time.
- `.spektacular/changelog/000035_plan-walkthrough-conversation.md:1-32` — canonical prior-feature changelog entry showing structure and voice.
- `.spektacular/config.yaml` — no changes required; already exposes `command`, `agent`, and `spec_trigger_threshold`.
- `templates/templates.go` — `//go:embed all:*` directive; new template auto-included in the embed FS.

## External references

None. This feature is entirely an internal instruction-surface change; no external RFCs, papers, or libraries apply.

## Prior plans / specs consulted

- `.spektacular/specs/000036_spec_plan_historical_artifacts.md:1-144` — the spec being planned. Establishes the eight requirements, one Constraint (no filesystem blocks), nine acceptance criteria, one Non-Goal (website-docs update deferred), and three Success Metrics that must land in the Testing Approach step.
- `.spektacular/changelog/000035_plan-walkthrough-conversation.md:1-32` — most recent per-feature changelog entry; used as the shape reference for the "changelog entry contains the follow-up note" requirement.
- No prior *plan* documents in `.spektacular/plans/` are directly relevant — this is a new instruction surface, not a follow-on to a prior plan. (`plan file list` was consulted but returned only unrelated plans.)

## Open assumptions

- **The archaeology exception can be expressed in prose without any code-level enforcement.** The spec's Constraint forbids access blocks and the requirement is behavioral, so prose in `AGENTS.md` is the enforcement mechanism. If this proves insufficient in practice (e.g. agents still reach for specs/plans during discovery), the follow-up would be to tighten the wording, not to add gates.
- **The "owning workflow" exception is self-evident because only the owning workflow uses the `spec file` / `plan file` CLI commands during its run.** The exception does not require a machine-checked "who owns this artifact right now" marker; the CLI itself is the signal (an agent reaching for `plan file read` inside the plan-workflow discovery step is the owning workflow by construction).
- **The changelog note requirement (spec req #8, AC #9) is satisfied by prose content in the per-feature changelog entry that ships with this feature's implementation.** No new schema field or CLI flag is needed. The plan will include an explicit acceptance check that the written changelog entry contains the "process document, not product document" website-docs follow-up note.
- **`spektacular init` is expected to be re-run in the repo after this feature ships to inject the new AGENTS.md section.** Existing repos already have the pattern for other managed sections; the new one follows the same lifecycle.
- **Two soft edits to existing step templates (`plan/02-discovery.md` and `implement/01-read_plan.md`) are net-positive.** Framing the plan-discovery read of prior plans/specs as historical archaeology, and framing the implement read as the owning workflow's own live artifact, prevents the appearance of contradiction between the new AGENTS.md rule and the existing step instructions. If either edit turns out to add noise rather than clarity during implementation, the fallback is to drop it — the AGENTS.md rule alone is enough to satisfy all spec requirements and acceptance criteria.

## Rehydration cues

- Re-read the spec: `go run . spec file read 000036_spec_plan_historical_artifacts.md`
- Re-inspect the template pattern to model against: `Read templates/agents/memory-context.md`, `Read templates/agents/spec-trigger.md`
- Re-inspect the install function pattern to mirror: `Read internal/agent/memory_context.go`, `Read internal/agent/spec_trigger.go`
- Re-inspect the three wire-in points: `Read internal/agent/claude.go`, `Read internal/agent/bob.go`, `Read internal/agent/codex.go`
- Re-inspect the test pattern to mirror: `Read internal/agent/memory_context_test.go`, `Read internal/agent/spec_trigger_test.go`
- Re-inspect the surgical clarifier targets: `Read templates/steps/plan/02-discovery.md`, `Read templates/steps/implement/01-read_plan.md`
- Re-inspect the changelog entry shape reference: `Read .spektacular/changelog/000035_plan-walkthrough-conversation.md`
- Confirm the embed FS captures the new template automatically: `Read templates/templates.go` (look for `//go:embed all:*`).
