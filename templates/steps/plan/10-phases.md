## Step {{step}}: {{title}}

For each milestone, define implementation phases. Each phase has **two outputs**, both finalized during the verification step:

- **plan.md entry** — a user-scannable summary and outcome-based acceptance criteria
- **context.md entry** — dense technical notes with file:line detail

### Phase content in plan.md

Each phase in plan.md must have:

- **Heading**: `#### - [ ] Phase N.M: <short title>` (markdown checkbox, not `####` alone)
- **Summary**: 2-4 plain-language sentences explaining what the phase does and why. No file:line references. No shell commands. A reader should understand the phase from this paragraph alone without opening context.md.
- **Technical detail link**: `*Technical detail:* [context.md#phase-NM](./context.md#phase-NM-<slug>)`
- **Acceptance criteria**: A `**Acceptance criteria**:` heading followed by `- [ ]` checkboxes. Each checkbox is an outcome statement in plain language — something a human can read and understand without running a command. "`spec` and `plan` produce the same JSON output as before the refactor" is good; "`go test ./...`" is not.

### Phase content in context.md

Each phase in context.md must have:

- **Heading**: `### Phase N.M: <title matching plan.md>` so plan.md's `*Technical detail:*` link resolves.
- **File changes**: Specific file:line changes based on research findings
- **Complexity**: Low / Medium / High
- **Token estimate**: ~Nk tokens (rough estimate for agent context usage)
- **Agent strategy**:
  - Low: Single agent, sequential execution
  - Medium: 2-3 parallel agents for independent changes
  - High: Parallel analysis, sequential integration

For guidance on agent orchestration: `{{config.command}} skill spawn-implementation-agents`

### Rules

- Every file change must reference a specific file (and line range where applicable) in context.md.
- NO open questions — resolve any uncertainties now.
- Acceptance criteria in plan.md are outcome statements, not shell commands. Verification commands belong in the agent's head, not in plan.md.
- The phase summary in plan.md is the primary artifact the user reads — prioritize clarity over completeness.

Before advancing, save this step's work to its **two** working files. Using your own `Write` tool, write:

- the **plan.md** phase content (the `#### - [ ] Phase N.M:` headings, summaries, technical-detail links, and acceptance criteria described under "Phase content in plan.md" above) to `.spektacular/work/{{plan_name}}/phases_plan.md`, and
- the **context.md** phase content (the `### Phase N.M:` headings with file:line detail, complexity, token estimate, and agent strategy described under "Phase content in context.md" above) to `.spektacular/work/{{plan_name}}/phases_context.md`.

Both working files are git-tracked and are read back on resume and when the plan documents are assembled, so they must hold the final content. They are **not** plan store documents — write them directly with your file tools and do **not** route them through `{{config.command}} plan file write` (that command is only for the final plan documents).

Present the phases to the user for review. Once agreed, advance:

{{config.command}} plan goto --data '{"step":"{{next_step}}"}'

---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
