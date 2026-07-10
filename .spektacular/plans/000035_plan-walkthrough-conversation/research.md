# Research: 000035_plan-walkthrough-conversation

## Alternatives considered and rejected

### Alternative 1: New terminal FSM step (e.g. `discuss_plan`) appended after `finished`
**What it is**: Add a new `workflow.StepConfig` (e.g. `{Name: "discuss_plan", Src: []string{"finished"}, Dst: "discuss_plan", ...}`) to `internal/steps/plan/steps.go`'s `Steps()` (currently `finished` is the last entry, `steps.go:48`), making the walkthrough a first-class FSM state the agent transitions into via `plan goto`.

**Why rejected**: `internal/workflow/state.go:25-27`'s `InProgress()` hardcodes the literal string `"finished"` as the terminal step for **all three** workflow kinds (spec, plan, implement): `return s.CurrentStep != "" && s.CurrentStep != "finished"`. Appending a step after `finished` would make `InProgress()` report `true` for a plan sitting in the new terminal state — indistinguishable from a plan genuinely stuck mid-workflow — breaking the resume-detection contract documented in `gotchas/goto-to-current-step-is-a-silent-noop.md` and used by `plan new`'s resume-report path. Fixing this would require making `InProgress()` kind-aware (or comparing against `steps[len(steps)-1].Name` instead of a magic string), a shared-code change touching spec/implement's resume behavior too — disproportionate to a feature that is pure conversational instruction. This mirrors the exact reasoning plan 000032 used to reject adding new workflow state for its own "offer during conversation" feature (see plan 000032 research.md, "Alternative 5: Defer/decline is agent behavior, not CLI state").

**Citation**: `internal/workflow/state.go:25-27`; `.spektacular/knowledge/gotchas/goto-to-current-step-is-a-silent-noop.md`; `internal/steps/plan/steps.go:48`.

### Alternative 2: Skill-level instruction only, no template change
**What it is**: Add the offer/walkthrough instruction entirely to `.claude/skills/spek-plan/SKILL.md`, with no change to `templates/steps/plan/18-finished.md` — the skill file tells the agent what to do once the CLI's `finished` instruction is received, without the CLI's own rendered text mentioning the offer at all.

**Why rejected**: `SKILL.md` is a Claude-Code-specific artifact (`.claude/skills/spek-plan/SKILL.md`) that only this one agent surface reads; the CLI's rendered instruction text in `templates/steps/plan/18-finished.md` is agent-agnostic (any coding agent driving the JSON protocol sees it, per `architecture/writing-files.md`'s note that the agent "can not be assumed to have access to the file system" beyond the CLI's own output). Putting the entire behavioral contract only in a Claude-Code-only skill file would make the feature invisible to any other agent surface driving this same CLI, unlike spec 000032's AGENTS.md placement (loaded natively by Claude, Codex, and Bob per that plan's Evidence 2). The template is the one place guaranteed to reach every agent using the workflow.

**Citation**: `.claude/skills/spek-plan/SKILL.md:1-4` (frontmatter, Claude-Code-only skill); `.spektacular/knowledge/architecture/writing-files.md`; plan 000032 plan.md § Architecture ("All three supported agents already load AGENTS.md at session start").

### Alternative 3: New `workflow.Data` key / template `Extra` field to track offer-once state
**What it is**: Persist "offer already made/declined for this plan" as a new key in `workflow.Data` (e.g. `walkthrough_declined: true`), written by the `finished` callback and checked before rendering the offer, so the offer-once behavior is enforced by the CLI/state rather than the agent's own conversational memory.

**Why rejected**: The `finished` step is the FSM's terminal state — the workflow does not re-enter it via a normal transition (no `Src` lists `finished` as a source for any other step), so there is no code path where the CLI would re-render the offer and need to check a "already offered" flag; the offer happens exactly once because the callback that renders it (`finished()`, `steps.go:277-299`) fires exactly once per plan run. Adding persisted state to prevent a re-offer that structurally cannot happen is solving a non-problem, and mirrors plan 000032's rejected "CLI-data-passing" alternative in spirit: the offer-once and interrupt/resume/close behaviors during the walkthrough conversation itself are properties of a single live agent turn, not something the FSM needs to track across restarts (a killed/resumed session re-running `plan goto --data '{"step":"finished"}'` would harmlessly re-render the same offer once, which is acceptable per the spec's "within a single plan's completion" scoping, not cross-session).

**Citation**: `internal/steps/plan/steps.go:277-299` (`finished()` — no `Src` elsewhere names `finished`, confirming single-fire); spec 000035 § Non-Goals ("Cross-session or cross-plan memory of prior walkthroughs... is out of scope").

## Chosen approach — evidence

### Evidence 1: The exact integration point is the `finished` step's rendered instruction
**File**: `templates/steps/plan/18-finished.md:17-27` (the `{{^plan_incomplete}}` branch, rendered once all three documents are confirmed written).

Verbatim current content of the success branch:
```
The plan workflow is complete. Three documents are now in the plan store under `{{plan_dir}}`:

- `{{plan_path}}` — the user-scannable plan
- `{{context_path}}` — technical detail for implementation
- `{{research_path}}` — the decision log and rehydration cues

Read any of them back with `{{config.command}} plan file read {{plan_name}}/<doc>.md`.

Inform the user that the plan workflow is finished and the three documents are ready for review.
```
Line 26 ("Inform the user that the plan workflow is finished...") is the precise hook: the walkthrough offer is one more sentence of instruction added here, telling the agent to offer a choice between reading the documents directly (the existing behavior, lines 20-25, entirely unchanged) and a conversational walkthrough (new). No mustache variable, no new `Extra` key, no Go change is needed to add this — `finished()` (`steps.go:277-299`) already renders this template with the existing `plan_dir`/`plan_path`/`context_path`/`research_path`/`plan_name`/`config.command` variables, none of which need to change.

**What this supports**: The offer, the walkthrough's structure (approach/reasoning → stage breakdown → scope boundaries), the mid-walkthrough edit protocol, and the explicit-agreement closing gate are all addable as pure instruction prose appended to this one existing template — zero Go changes, zero FSM changes, following the exact "instruction prose is the entire behavioral surface" pattern plan 000032 established (see that plan's plan.md § Architecture, "Instruction prose as the entire behavioral surface").

### Evidence 2: Mid-walkthrough edits reuse the exact same `plan file write` mechanism the workflow already uses — no new write path
**File**: `cmd/plan_file.go` (the `plan file` subcommand group — `read`/`write`/`list`, standalone CLI commands, not gated by FSM state); `.claude/skills/spek-plan/SKILL.md:26-32` (documents `plan file write <name>/<doc>.md --from <source-path>` as the sole supported write path for the three store documents, callable any time, independent of which workflow step is current); `templates/steps/plan/18-finished.md:7-9` (shows the exact same command form already used mid-workflow: `{{config.command}} plan file write {{plan_name}}/plan.md --from .spektacular/tmp/plan_template.md`).

`plan file write` is not an FSM-gated step callback — it is a standalone `cmd/plan_file.go` command that resolves the plan's document path from the configured plan directory (via `store.Store`, same as every other document write in this workflow) and overwrites it, regardless of the workflow's `current_step`. This means a mid-walkthrough edit uses **exactly** the same call the workflow itself used to commit `plan.md` originally (`18-finished.md:7`, `15-write_plan.md`'s own instruction) — there is no separate "edit mode" or second write path to build. The spec's constraint ("changes... not a separate copy or transient state") is satisfied by construction: there is only ever one write path into the plan store, and the walkthrough instruction just tells the agent to use it again.

**What this supports**: The new instruction's mid-walkthrough-change handling is: stage the edited section under `.spektacular/tmp/` (same convention as `18-finished.md:4-11`'s recovery path and the working-file convention documented in `SKILL.md:34-38`), run `plan file write <name>/plan.md --from <scratch>`, confirm to the user, `rm` the scratch file, resume the walkthrough. No new CLI surface, no new Go code.

### Evidence 3: Precedent feature (spec 000032) already identified this exact gap and validated the "instruction-only" shape for a conversational offer+interrupt+respond loop
**File**: spec 000032 § Non-Goals: "Adding a user-acceptance walkthrough to the plan workflow's verification step — a separately identified gap, not addressed here." Plan 000032's plan.md (in full) and research.md (in full) — the sibling feature this plan's approach is modeled on.

Plan 000032 solved a structurally similar problem — "the agent should proactively offer something during a conversation, honor accept/defer/decline, and the offer/response loop is pure agent behavior, not CLI state" — entirely through instruction prose in an existing, already-loaded file (`AGENTS.md`), with zero FSM/workflow-state changes. That plan's own research explicitly traced and rejected a CLI-data-passing mechanism for a related "carry conversation context into a workflow" problem, concluding "the agent proposing the offer, accepting it,... is the same agent in the same live conversation turn — it already holds the discussion in its own context" (000032 research.md, Alternative 3). The same reasoning applies directly here: the agent walking the user through the plan, taking a change request, and editing the plan is the same agent in the same live turn holding the whole plan in context — no CLI mechanism is needed to feed it back to itself mid-walkthrough.

**What this supports**: High confidence that "instruction prose appended to `18-finished.md`, no FSM/Go changes" is not just possible but is this codebase's established, twice-validated pattern for "agent offers something conversationally, handles accept/defer/decline, applies changes live" features.

### Evidence 4: `SKILL.md`'s existing "do not stop" framing needs one addition to avoid contradicting the new offer
**File**: `.claude/skills/spek-plan/SKILL.md:20-22` — "Keep looping... until a returned instruction tells you the workflow is *finished*. Only then should you report completion to the user." No mention anywhere in the skill file of an offered walkthrough after that point.

The skill file's current framing ends at "report completion" with no acknowledgment that the `finished` instruction (once this feature ships) will also ask the agent to offer a walkthrough. Left unedited, an agent reading `SKILL.md` in isolation could report completion and stop before actually presenting the offer described in the (now-updated) `18-finished.md` instruction it just received — the skill file doesn't tell it to look for or act on that offer text. A small addition to `SKILL.md` (one or two sentences near line 20-22) closing this gap is in scope for this plan, alongside the template change, so the two files stay consistent — this is the one place besides the template itself that needs a prose edit.

**Citation**: `.claude/skills/spek-plan/SKILL.md:20-22`.

### Evidence 5: `writeStep`/`stepkit.WriteStepResult` render pipeline confirms no data-plumbing changes are needed
**File**: `internal/steps/plan/steps.go:64-76` (`writeStep` helper, wraps `stepkit.WriteStepResult`), `steps.go:277-299` (`finished()` — only sets `Extra["plan_incomplete"]` conditionally, otherwise `nil`).

Confirms directly (no need to re-derive from plan 000032's stepkit tracing, which covered the sibling `spec` package's `stepkit.WriteStepResult` call path — the `plan` package uses the identical `stepkit` module): the `finished` template already receives everything it needs (`plan_dir`, `plan_path`, `context_path`, `research_path`, `plan_name`, `config.command`) via the existing `buildResult`/`Strategy` machinery. Adding walkthrough-offer prose to the template requires no new `Extra` key, no new template variable, and no change to `finished()`'s Go logic — purely a Markdown/template-body edit.

**What this supports**: Confirms Evidence 1's claim that this is a zero-Go-diff, template-and-skill-prose-only change.

## Files examined

- `templates/steps/plan/18-finished.md:1-27` — full current content of the terminal step's instruction; line 26 is the exact insertion point for the walkthrough offer.
- `internal/steps/plan/steps.go:28-48` (`Steps()`) — full FSM step list for the plan workflow; `finished` (line 48) is positionally last, `Src: []string{"write_research"}`, nothing transitions out of it.
- `internal/steps/plan/steps.go:64-76` (`writeStep`) — confirms the template-rendering call shape (`stepkit.StepRequest` with `Extra`), and that no new `Extra` key is needed for this feature.
- `internal/steps/plan/steps.go:255-299` (`writeContext`, `writeResearch`, `finished`) — confirms `finished()` fires exactly once per plan run (nothing re-enters it) and its only conditional behavior is the `plan_incomplete` warning branch, untouched by this feature.
- `internal/workflow/state.go:1-61`, specifically `InProgress():25-27` — the exact hardcoded `"finished"` terminal-step check that rules out adding any new FSM step after `finished` without a shared-code change across all three workflow kinds.
- `.spektacular/knowledge/architecture/workflow-steps.md` (read in full via `knowledge read`) — general FSM/step-adding mechanics; confirms steps are added via `StepConfig` in a domain package's `Steps()`, and that a callback "must not write to any file directly — use the `Data` store" (not relevant here since this feature adds no callback logic, only template prose).
- `.spektacular/knowledge/gotchas/goto-to-current-step-is-a-silent-noop.md` (read in full) — confirms `Workflow.Goto` treats same-step calls as silent no-ops, reinforcing why `finished` is genuinely terminal in practice, not just in `InProgress()`'s check.
- `cmd/plan_file.go` — the standalone `plan file read|write|list` subcommand group; confirms `plan file write` is callable independent of FSM state, which is exactly the mechanism mid-walkthrough edits reuse.
- `.claude/skills/spek-plan/SKILL.md:1-67` (read in full) — the Claude-Code-specific skill driving this very workflow; §"Reading and writing plan files" (lines 24-32) confirms the `plan file write --from <scratch>` convention; lines 20-22 identify the one place needing a small consistency edit alongside the template change.
- `.spektacular/specs/000035_plan-walkthrough-conversation.md` (read in full) — source spec; Technical Approach explicitly leaves "the integration point in the plan workflow" to this planning step's discretion, now resolved.
- `.spektacular/specs/000032_spec-workflow-pair-programming-enhancements.md` § Non-Goals — the direct precedent confirming this exact gap was identified previously and deliberately deferred to its own spec (this one).
- `.spektacular/plans/000032_spec-workflow-pair-programming-enhancements/plan.md` (read in full) and `research.md` (read in full) — the sibling feature's plan/research, used as the structural template for this plan's own architecture decision (instruction-prose-only, zero FSM changes) and as direct evidence for Alternative 1's and Alternative 3's rejections.

## External references

None. The feature is entirely internal to Spektacular's existing plan-workflow template and skill-instruction surfaces.

## Prior plans / specs consulted

- **Plan/spec 000032 (spec-workflow-pair-programming-enhancements)** — direct precedent and the source of the "separately identified gap" note that named this exact feature. Its research.md's rejection of CLI-data-passing for conversational context and its plan.md's "instruction prose is the entire behavioral surface" architecture section are the template this plan's own architecture step should follow.
- **Plan/spec 000034 (spec-plan-implement-reconciliation)** — read for context (a related but distinct plan-workflow enhancement, verification-focused rather than conversational); no direct reuse found, no overlap — the reconciliation check and this walkthrough feature both extend the plan workflow's tail end but address different concerns (spec-coverage verification vs. user-facing narrative explanation) and can coexist without conflict in `18-finished.md`/adjacent steps.

## Open assumptions

- **Assumed**: the walkthrough offer belongs in `18-finished.md` (after all three documents are confirmed committed) rather than earlier (e.g. in `14-verification.md`, before the documents are written to the store). This assumes the user wants to walk through the *final*, already-persisted plan, not a pre-commit draft — consistent with the spec's Overview ("After Spektacular finishes generating an implementation plan..."). If this is wrong, the insertion point moves earlier and mid-walkthrough edits would need to happen before `write_plan`/`write_context`/`write_research` fire, which is a materially different design. **If wrong, STOP and ask the user before proceeding with implementation.**
- **Assumed**: no change is needed to `internal/steps/plan/steps.go`'s `finished()` Go function itself — only the template it renders (`18-finished.md`) and the skill file (`SKILL.md`). Verified directly (Evidence 5) that the existing render pipeline supplies everything needed; low risk, but flagged since it's the crux of the "zero Go diff" claim.
- **Assumed**: the "propose-then-confirm" and "offer once, no re-offer" semantics can be fully expressed as natural-language instruction and don't need any persisted flag, per Alternative 3's rejection. This mirrors plan 000032's validated approach for a structurally similar conversational offer, but that plan's offer could recur across multiple points in an open-ended conversation (spec 000032 explicitly allows "may raise the offer again"); this feature's offer is explicitly "exactly once" per the spec's acceptance criteria, which is actually a **simpler** constraint to satisfy via prose (fire-once callback, no persisted state needed) — assumption carries lower risk than its precedent, not higher.

## Rehydration cues

- Re-read `templates/steps/plan/18-finished.md` in full (27 lines) to see the exact current text the walkthrough prose will be appended to.
- Re-read `.claude/skills/spek-plan/SKILL.md` in full (67 lines) for the skill-level consistency edit.
- Re-run `go run . knowledge read --data '{"scope":"project","path":"architecture/workflow-steps.md"}'` for the general step-architecture background if a future session needs to re-verify no FSM change is required.
- Re-read `.spektacular/plans/000032_spec-workflow-pair-programming-enhancements/plan.md` and its `research.md` — the structural template this plan's Architecture/Component Breakdown/Testing Approach sections should follow (instruction-prose-only feature shape).
- `internal/steps/plan/steps.go:28-48` and `:277-299` for the FSM step list and `finished()` callback, to re-confirm the zero-Go-diff claim if implementation discovers otherwise.
