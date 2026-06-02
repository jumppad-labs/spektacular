## Step {{step}}: {{title}}

Draft the **Out of Scope** section of `plan.md`. This section lists the explicit exclusions agreed during planning — what this plan is NOT doing, and where those things are tracked instead.

### Rules

- One bullet per exclusion.
- State what is not being done, in plain language.
- Where a follow-up exists (another spec, another plan, a ticket), name it so a reader can find it.
- Pull out-of-scope items from three places:
  1. The spec's § Non-Goals section
  2. Anything the user said "not now" to during the architecture step
  3. Anything the chosen design deliberately leaves to a later plan
- If the plan truly has no exclusions worth naming, say so explicitly — an empty section is usually a sign something was missed.

### What to produce

A draft Out of Scope section ready to drop into plan.md at verification time. Present it to the user for review and confirm each exclusion.

Before advancing, save this section to its working file. Using your own `Write` tool, write the agreed **Out of Scope** content (body only — no `## ` heading line) to `.spektacular/work/{{plan_name}}/out_of_scope.md`. This working file is git-tracked and is read back on resume and when the plan documents are assembled, so it must hold the final content. It is **not** a plan store document — write it directly with your file tools and do **not** route it through `{{config.command}} plan file write` (that command is only for the final plan documents).

Once the user agrees on the exclusions, advance:

{{config.command}} plan goto --data '{"step":"{{next_step}}"}'

---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
