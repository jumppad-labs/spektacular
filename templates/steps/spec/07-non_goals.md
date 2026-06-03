## Step {{step}}: {{title}}

Ask the user: What is explicitly OUT of scope for this feature?

Examples:
• 'Mobile support is out of scope (tracked in #456)'
• 'Internationalisation will be addressed in a follow-up spec'

Capture their response. If blank, note that no non-goals have been defined.

Before advancing, save this section to its working file. Using your own `Write` tool, write the agreed **Non-Goals** content (the body only — no `## ` heading line) to `.spektacular/work/{{spec_name}}/non_goals.md`. This working file is git-tracked and is read back on resume and when the spec is assembled, so it must hold the final agreed content for this section. It is **not** a spec store document — write it directly with your file tools and do **not** route it through `{{config.command}} spec file write` (that command is only for the final assembled spec).

Once you have captured the information from the user move to the next step by running the command:

{{config.command}} spec goto --data '{"step":"{{next_step}}"}'


---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
