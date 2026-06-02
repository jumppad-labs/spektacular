## Step {{step}}: {{title}}

Draft the **Component Breakdown** section of `plan.md`. This section lists the components (new or changed) that make up the chosen solution, their responsibilities, and how they interact.

### Rules

- One bullet or short paragraph per component.
- For each component: name it, state what it owns, and describe its relationship to the other components.
- Do NOT list file paths or line numbers in plan.md — those belong in context.md. Component responsibilities, not implementation sites.
- Reuse existing components wherever possible; a new component needs justification.
- Cover both new components and meaningfully-changed existing ones.

### What to produce

A draft Component Breakdown section ready to drop into plan.md at verification time. Present it to the user for review.

Before advancing, save this section to its working file. Using your own `Write` tool, write the agreed **Component Breakdown** content (body only — no `## ` heading line) to `.spektacular/work/{{plan_name}}/components.md`. This working file is git-tracked and is read back on resume and when the plan documents are assembled, so it must hold the final content. It is **not** a plan store document — write it directly with your file tools and do **not** route it through `{{config.command}} plan file write` (that command is only for the final plan documents).

Once the user is happy with the component list, advance:

{{config.command}} plan goto --data '{"step":"{{next_step}}"}'

---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
