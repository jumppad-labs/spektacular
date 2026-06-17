## Step {{step}}: {{title}}

Ask the user: Do you have any technical direction already decided?

Examples:
• Key architectural decisions already made
• Preferred patterns or technologies
• Integration points with existing systems
• Known risks or areas of uncertainty

Technical Approach is **non-binding direction** — preferences and suggestions the planning agent may adopt, adapt, or replace. It holds no hard rules. If the user states something as a hard "must" (e.g. "must use SQLite", "must not break the API"), that is a **constraint**, not technical direction — it belongs in the Constraints section, not here.

**Capture only the direction the user has already decided — do not design it for them, and do not investigate the codebase to "ground" it.** Reading source files, listing the repo, and reporting existing types, functions, or routes is plan-discovery work; it does not belong in a spec. If the user has decided nothing here, that is fine — note that no technical direction has been decided and let the plan workflow propose it. Keep whatever you capture at high-level-direction altitude (e.g. "use an embedded datastore", "integrate with the existing user store"), not implementation detail discovered from the code.

**Stay at direction altitude — do not write the design itself.** This is the section where mechanism is *welcome*, which makes it the easiest place to drift too low. Technical Approach is a *steer*, not a worked solution: name each decision or preference in a sentence or two, with its *why* if it helps, and stop. Designing the *how* is the downstream **plan workflow's** job — its steps own discovery, architecture, components, data structures, implementation detail, dependencies, testing approach, milestones, and phases. Apply this test before writing anything down: *would this content be re-derived by one of those plan steps?* If yes, it belongs to the plan, not here. Concrete tells that you have dropped too low: a numbered pipeline or algorithm, step-by-step processing, data shapes, field or function names, the ordering of operations, or anything that reads like "first do X, then Y, then Z". Compress each to a one-line steer (e.g. not the five steps of a de-duplication pipeline, but "prefer consolidating results behind a dedicated lookup step, because duplication can only be judged after the full content is read") and leave the worked design to the plan.

**Do not restate content already captured in another section.** Anything that belongs in Constraints (e.g. "must use an embedded datastore", "must replace the existing file storage", "the database file location must be configurable") lives there, not here — do not copy it back into Technical Approach. Capture only *additional* technical direction that is not already a requirement or constraint. If there is none beyond what is already captured, say exactly that in one line — e.g. "No technical direction has been decided beyond the captured constraints; the detailed design is left for the plan workflow to propose." — without re-listing those constraints.

Capture their response. If blank, note that no technical direction has been decided.

Before advancing, save this section to its working file. Using your own `Write` tool, write the agreed **Technical Approach** content (the body only — no `## ` heading line) to `.spektacular/work/{{spec_name}}/technical_approach.md`. This working file is git-tracked and is read back on resume and when the spec is assembled, so it must hold the final agreed content for this section. It is **not** a spec store document — write it directly with your file tools and do **not** route it through `{{config.command}} spec file write` (that command is only for the final assembled spec).

Once you are satisfied, move to the next step by running the command:

{{config.command}} spec goto --data '{"step":"{{next_step}}"}'


---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
