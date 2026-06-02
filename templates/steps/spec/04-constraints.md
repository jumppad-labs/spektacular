## Step {{step}}: {{title}}

Ask the user: Are there any hard constraints or boundaries the solution must operate within?

Examples:
• Must integrate with the existing authentication system
• Cannot introduce breaking changes to the public API
• Must support the current minimum supported runtime versions

**Constraints are boundaries, not design decisions.** Apply this test to every constraint before you write it down:

> *If this constraint were removed, would the feature become impossible, or just implemented differently?*

Only the first kind belongs here. The second kind is a design decision and belongs in Technical Approach.

• Real constraint: *"must not break the shape of the public JSON response"* — removing it lets the feature break downstream consumers.
• Real constraint: *"must not require new config keys"* — removing it changes the deployment contract.
• Not a constraint: *"must use the existing FSM engine"* — removing it just lets you pick a different mechanism. That's a design choice; move it to Technical Approach.
• Not a constraint: *"must use markdown templates with mustache rendering"* — same reason.

Constraints are usually phrased as "must not break X" or "must stay compatible with Y", not "must use Z". If the user gives you a "must use" item, ask whether removing it would make the feature impossible — if not, park it for Technical Approach.

Capture their response. If blank, note that there are no constraints.

Before advancing, save this section to its working file. Using your own `Write` tool, write the agreed **Constraints** content (the body only — no `## ` heading line) to `.spektacular/work/{{spec_name}}/constraints.md`. This working file is git-tracked and is read back on resume and when the spec is assembled, so it must hold the final agreed content for this section. It is **not** a spec store document — write it directly with your file tools and do **not** route it through `{{config.command}} spec file write` (that command is only for the final assembled spec).

Once you are satisfied, move to the next step by running the command:

{{config.command}} spec goto --data '{"step":"{{next_step}}"}'


---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
