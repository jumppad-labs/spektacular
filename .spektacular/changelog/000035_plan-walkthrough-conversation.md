# Feature: Conversational plan walkthrough

## What was built

After Spektacular finishes generating an implementation plan (`plan.md`, `context.md`, `research.md`), the assistant no longer just reports the documents are ready and stops. Its final message now also offers the user a choice: read the documents directly (unchanged from before), or have the assistant walk through the plan conversationally, the way one colleague would explain it to another.

If the user accepts, the assistant presents the plan as a small number of natural conversational beats, paced with brief pauses rather than delivered as one uninterrupted monologue or a rigid per-phase sign-off gate:

1. The chosen approach and the reasoning behind it.
2. How the work breaks into stages/phases.
3. What was deliberately left out of scope, and why.

At any point during the walkthrough, the user can interrupt with a question or a request to change something. The assistant responds directly before returning to the beat it was on. If the interruption is a change request, the assistant applies it immediately — editing the affected section, committing it through the same `plan file write <name>/plan.md --from <scratch>` command the rest of the workflow already uses (not a separate copy or transient draft), confirming the update to the user, and then resuming the walkthrough, re-summarizing anything the change affected.

The walkthrough only concludes once the user gives an explicit affirmative answer to a direct closing question ("does this look right, ready to proceed?"). Silence, a topic change, or an ambiguous reply is never treated as agreement.

Declining the initial offer — or simply not taking it up — leaves the workflow's existing behavior completely unchanged: the three documents are ready, and reading them directly remains a fully valid path with no missing step.

Implemented entirely as instruction prose added to two existing files:
- `templates/steps/plan/18-finished.md` — the plan workflow's terminal step template, which every agent driving Spektacular's CLI protocol reads, gained the offer and the full walkthrough behavior.
- `.claude/skills/spek-plan/SKILL.md` — the Claude-Code skill that drives the plan workflow loop, gained one clarifying sentence so an agent reading it doesn't report completion and stop short of actually presenting the new offer.

No Go code, no new workflow state, no new CLI command, and no new template variables were introduced. Mid-walkthrough edits reuse the exact same `plan file write` mechanism the workflow already relied on to commit the plan in the first place.

## Why it matters

Previously, once a plan finished generating, the user was left to read dense planning documents alone to catch anything they might want changed. This feature lets the assistant instead walk the user through the plan's approach, structure, and boundaries as a live conversation — closer to how two colleagues would discuss a design together — so the user can fully understand and shape the plan through dialogue rather than solitary document review. Any change surfaced in that conversation lands in the same plan document the rest of the workflow reads and writes, so the discussion and the plan never fall out of sync.

## Deviations from the plan

None. Both phases (the `18-finished.md` walkthrough instruction, and the `SKILL.md` consistency edit) were implemented exactly as scoped in the plan's Milestones & Phases section, with no drift discovered between the plan/context documents and the actual state of either target file. `make test` and `make lint` pass; zero `.go` files were touched, matching the plan's explicit zero-Go-diff claim.
