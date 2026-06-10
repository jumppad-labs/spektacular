## Step {{step}}: {{title}}

Ask the user: Are there any hard constraints or boundaries the solution must operate within?

Examples:
• Must integrate with the existing authentication system
• Cannot introduce breaking changes to the public API
• Must support the current minimum supported runtime versions

**Constraints are the hard rules and boundaries the solution MUST honor.** If the user states something as a binding rule — a "must", "must not", or "cannot" about how or within what bounds the solution is built — it is a constraint. This explicitly includes a **mandated technology**: "must use X" is a constraint, not a design decision to be moved elsewhere.

Apply this test:

> *Is this a hard rule the solution is required to honor, or just a preferred direction the planner could adapt or replace?*

Hard rules belong here. Non-binding direction — "use X if convenient", "prefer Y", "consider Z" — is **not** a constraint; it belongs in Technical Approach, which is guidance the planner may revise. Phrasing is the strongest signal: "must" / "must not" / "cannot" → constraint; "prefer" / "consider" / "use … if" → Technical Approach.

• Constraint: *"must use SQLite"* — a binding technology mandate (e.g. the whole point of a migration). It stays a constraint; do **not** move it to Technical Approach.
• Constraint: *"must integrate with the existing user store"* — a required integration boundary.
• Constraint: *"must not break the shape of the public JSON response"* — removing it lets the feature break downstream consumers.
• Constraint: *"must run embedded, with no separate database server"* — a deployment boundary.
• Not a constraint: *"prefer a table-per-entity schema"* — a non-binding design preference; that's Technical Approach.

(Hard rules about what the feature must *do* — its behaviour — are Requirements, captured in the previous step. Constraints are the hard rules about *how / within what bounds* it is built.)

Capture their response. Before accepting "no constraints", check the most common sources of real boundaries so an empty section is a deliberate choice and not an oversight — ask the user about each that the work might touch:

• An **existing system** the work must integrate with or replace (e.g. an existing datastore, file format, or API whose data or shape must be preserved).
• A **deployment or runtime** boundary (e.g. must run as a single self-contained binary, no separate server process, a fixed runtime or platform).
• A **compatibility** guarantee (e.g. must not break a public response shape or an existing deployment contract).

If, after checking these, the user confirms there are genuinely none, note that there are no constraints — a blank section is a valid outcome.

Before advancing, save this section to its working file. Using your own `Write` tool, write the agreed **Constraints** content (the body only — no `## ` heading line) to `.spektacular/work/{{spec_name}}/constraints.md`. This working file is git-tracked and is read back on resume and when the spec is assembled, so it must hold the final agreed content for this section. It is **not** a spec store document — write it directly with your file tools and do **not** route it through `{{config.command}} spec file write` (that command is only for the final assembled spec).

Once you are satisfied, move to the next step by running the command:

{{config.command}} spec goto --data '{"step":"{{next_step}}"}'


---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
