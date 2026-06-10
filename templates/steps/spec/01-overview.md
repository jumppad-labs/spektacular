## Step {{step}}: {{title}}

**You are writing a spec, not a plan.** A spec captures *what* is being built and the direction the user has *already decided* — requirements, acceptance criteria, constraints, and any high-level technical direction. It does **not** design the *how*, and it does **not** investigate the codebase: do not read source files, run code searches, or ground decisions in specific files, types, or functions. That grounding is the plan workflow's `discovery` step. Stay at spec altitude in every step that follows.

Ask the user to describe this feature in 2-3 sentences:
• What is being built?
• What problem does it solve?
• Who benefits?

Capture their response. Be specific — avoid generic phrases like 'improve the experience'.

**Keep the overview stakeholder-readable.** No file paths, no section names, no step names, no framework or library names, no code identifiers. A non-engineer should be able to read it and understand the value. If you can't explain the feature without naming an implementation artifact, ask the user one more "so that…" question until you find the user-visible value.

If the user volunteers implementation detail ("we add a new FSM state", "append to research.md"), capture it mentally and tell them it will land in Technical Approach — then rephrase the overview at the behavior level.

Ask for clarification if the description is vague, incomplete, or leaks implementation before moving on.

Before advancing, save this section to its working file. Using your own `Write` tool, write the agreed **Overview** content (the body only — no `## ` heading line) to `.spektacular/work/{{spec_name}}/overview.md`. This working file is git-tracked and is read back on resume and when the spec is assembled, so it must hold the final agreed content for this section. It is **not** a spec store document — write it directly with your file tools and do **not** route it through `{{config.command}} spec file write` (that command is only for the final assembled spec).

Once you are satisfied with the overview, move to the next step by running the command:

{{config.command}} spec goto --data '{"step":"{{next_step}}"}'

---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
