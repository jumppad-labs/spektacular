# Feature: 000023_context-over-memory

## Overview

Add a project-level instruction to this repo so that any coding agent (Claude, Bob, Codex, and similar) redirects what it would normally save into its per-user, per-machine memory into a shareable knowledge store instead. Today, learned conventions, feedback, and project facts get trapped on one developer's machine, so collaborators never benefit from them. The whole team benefits: guidance becomes shared, reviewable, and travels with the project.

## Requirements

- **Redirect to shareable context.** When working in this repository, an agent that would normally persist information to its per-user memory store must instead persist that information to the project's configured shareable context destination.
- **Applies to all coding agents.** The instruction must apply to all coding agents the team uses (Claude, Bob, Codex, and any future ones), without per-agent duplication.
- **Discoverable at session start.** The instruction must be discoverable by an agent at the start of a session, without the user having to mention it.
- **Scoped to this repo.** The instruction must not change agent behavior outside this repository — only behavior inside the repo is affected.
- **Reviewable by humans.** Information written to the shareable destination must remain reviewable and editable by humans.
- **Installed and updated by `spektacular init`.** The instruction is written into the project's agent-instruction surface by `spektacular init`, not maintained by hand. Re-running `spektacular init` (for the same agent or a different one) must update the managed section in place rather than appending a duplicate.

## Constraints

- Must not modify agent runtimes — the change can only touch project-level files.
- Must not break tessl integration with this repo.

## Acceptance Criteria

- **Redirect.** In an agent session started inside this repo, when the agent decides to persist something it would have written to per-user memory, the resulting persisted artifact appears in the configured shareable context destination and does not appear in the agent's per-user memory store.
- **Cross-agent.** The same instruction in this repo produces the redirect behavior in Claude, Bob, and Codex sessions, with no per-agent file or branch added.
- **Discoverable on entry.** In a fresh session inside this repo — with no user message referencing memory, context, or this instruction — the agent still redirects the first persistence opportunity to the shareable destination.
- **Scoped to this repo.** The same agent, started in a different repository, continues to write to its per-user memory store as normal.
- **Human-reviewable.** A collaborator can open the shareable destination with standard tooling, edit or remove an entry an agent wrote, and a subsequent session reflects that edit.
- **Idempotent install.** Running `spektacular init <agent>` twice in a row, or running it once for each of Claude / Codex / Bob, leaves exactly one managed Memory & Context section in `AGENTS.md`. No duplicate sections appear, and any unrelated content elsewhere in `AGENTS.md` (including a tessl-managed block, when present) is preserved byte-for-byte.

## Technical Approach

No technical direction pre-decided. Choice of file, structure, and wording is left to the plan.

Known risks / uncertainty:
- Agent adherence to project-level instructions is best-effort, not guaranteed.
- The instruction assumes each agent already has a configured shareable context destination for this repo; if it doesn't, there is nothing to redirect to.

## Success Metrics

No formal success metrics. Adoption of the instruction by any given agent is best-effort, so we will not track a pass/fail signal. Success is informal: collaborators notice that picked-up project context is increasingly shared rather than trapped on one machine.

## Non-Goals

- Migrating existing per-user memory entries into the shared context destination. Only new persistence triggered after the instruction is in place is affected.
- Building or modifying the shareable context destination itself. We rely on it already being configured per agent.
- Enforcing adherence. We are not adding any verification, hook, or guardrail that detects an agent writing to per-user memory.
- Defining a canonical destination across all team members. Different collaborators may have different destinations configured.
