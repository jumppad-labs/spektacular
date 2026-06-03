## Step {{step}}: {{title}}

Review the requirements captured in the previous step.

For each requirement, ask the user: "What is the pass/fail condition that proves this is done?"

A good criterion:
• Describes an observable outcome
• Passes or fails — no subjective judgment
• Is traceable to this requirement

Example: "When X happens, Y is visible / saved / returned."

**Stay at the blackbox level.** A criterion should be something a tester who has never read the source could verify by observing inputs and outputs — files on disk, API responses, UI state, log lines. Avoid criteria that inspect internal plumbing:

• Good: *"After the user accepts a candidate, a file exists under the knowledge directory containing the candidate's title and content."*
• Bad: *"Running the FSM goto from source state X is accepted and from source state Y is rejected."* — that's a unit test of an internal state machine, not an acceptance criterion.

If the user gives you a criterion that names internal states, private functions, or specific code paths, rephrase it as an observable outcome and tell them the internal check belongs in tests, not the spec.

Capture all criteria. Ask for clarification on any that are subjective, not traceable to a requirement, or that test internal plumbing before moving on.

Before advancing, save this section to its working file. Using your own `Write` tool, write the agreed **Acceptance Criteria** content (the body only — no `## ` heading line) to `.spektacular/work/{{spec_name}}/acceptance_criteria.md`. This working file is git-tracked and is read back on resume and when the spec is assembled, so it must hold the final agreed content for this section. It is **not** a spec store document — write it directly with your file tools and do **not** route it through `{{config.command}} spec file write` (that command is only for the final assembled spec).

Once you are satisfied with the acceptance criteria, move to the next step by running the command:

{{config.command}} spec goto --data '{"step":"{{next_step}}"}'


---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
