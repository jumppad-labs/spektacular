## Step {{step}}: {{title}}

Draft the **Dependencies** section of `plan.md`. List the internal packages, external libraries, upstream specs, and prior plans this work depends on.

### Rules

- One bullet per dependency.
- For each: a one-line note on what it provides and whether it needs any changes.
- Cover both runtime dependencies (imported packages, external services) and planning dependencies (prior specs or plans that must land first).
- If a dependency must land or change before this plan starts, flag that explicitly.
- If there are no meaningful dependencies, say so explicitly — an empty section is not acceptable.

### What to produce

A draft Dependencies section ready to drop into plan.md at verification time. Present it to the user for review.

Before advancing, save this section to its working file. Using your own `Write` tool, write the agreed **Dependencies** content (body only — no `## ` heading line) to `.spektacular/work/{{plan_name}}/dependencies.md`. This working file is git-tracked and is read back on resume and when the plan documents are assembled, so it must hold the final content. It is **not** a plan store document — write it directly with your file tools and do **not** route it through `{{config.command}} plan file write` (that command is only for the final plan documents).

Once the user is happy with the dependency list, advance:

{{config.command}} plan goto --data '{"step":"{{next_step}}"}'

---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
