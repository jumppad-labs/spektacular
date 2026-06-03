## Step {{step}}: {{title}}

Draft the **Data Structures & Interfaces** section of `plan.md`. This section captures the types, interface signatures, and serialization boundaries introduced or changed by the plan.

### Rules

- Focus on contracts between components, not internal representation detail.
- Show type shapes in pseudocode or a short code block where it helps — but keep it concise; this is a plan, not source.
- Name each type or interface and describe its role in one or two sentences.
- If no new data structures or interfaces are introduced, say so explicitly — an empty section is not acceptable.
- Per-field implementation detail (defaults, validation, wire-format nuance) belongs in context.md, not here.

### What to produce

A draft Data Structures & Interfaces section ready to drop into plan.md at verification time. Present it to the user for review.

Before advancing, save this section to its working file. Using your own `Write` tool, write the agreed **Data Structures & Interfaces** content (body only — no `## ` heading line) to `.spektacular/work/{{plan_name}}/data_structures.md`. This working file is git-tracked and is read back on resume and when the plan documents are assembled, so it must hold the final content. It is **not** a plan store document — write it directly with your file tools and do **not** route it through `{{config.command}} plan file write` (that command is only for the final plan documents).

Once the user is happy with the contracts, advance:

{{config.command}} plan goto --data '{"step":"{{next_step}}"}'

---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
