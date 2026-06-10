## Step {{step}}: {{title}}

Ask the user: What is explicitly OUT of scope for this feature?

Examples:
• 'Mobile support is out of scope (tracked in #456)'
• 'Internationalisation will be addressed in a follow-up spec'

You may **propose** likely non-goals inferred from the spec so far — operations the requirements mention but do not require (e.g. delete when only create/update/retrieve are required), capabilities deferred to a follow-up, or alternatives the design explicitly excludes. But proposing is not deciding: present any inferred non-goals to the user as a list and get their **explicit confirmation** (and any edits) before saving. Never write inferred non-goals as settled without the user confirming them — non-goals shape scope, and a wrong one silently narrows the feature.

**Do not duplicate the Constraints section.** A non-goal must not merely restate the inverse of a constraint already captured. If Constraints says *"must use SQLite"*, then *"other datastores are out of scope"* adds nothing — it is already implied by the constraint; leave it out. Likewise *"must replace the file storage"* already implies *"no dual-writing the legacy files."* Non-Goals is for scope exclusions the constraints do **not** already imply: capabilities deferred to a follow-up, operations the requirements mention but do not require (e.g. delete when only create/update/retrieve are required), or adjacent features a reader might reasonably expect that this feature will not address. Before saving, check each non-goal against the Constraints section and drop any that is just a constraint stated in reverse.

Capture the confirmed response. If the user confirms there are none, note that no non-goals have been defined.

Before advancing, save this section to its working file. Using your own `Write` tool, write the agreed **Non-Goals** content (the body only — no `## ` heading line) to `.spektacular/work/{{spec_name}}/non_goals.md`. This working file is git-tracked and is read back on resume and when the spec is assembled, so it must hold the final agreed content for this section. It is **not** a spec store document — write it directly with your file tools and do **not** route it through `{{config.command}} spec file write` (that command is only for the final assembled spec).

Once you have captured the information from the user move to the next step by running the command:

{{config.command}} spec goto --data '{"step":"{{next_step}}"}'


---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
