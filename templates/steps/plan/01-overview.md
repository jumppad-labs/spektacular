## Step {{step}}: {{title}}

Read the specification file at `{{spec_path}}` to understand what is being planned.

This spec is the source of truth for the plan's scope, requirements, constraints, and success metrics. Keep it in mind throughout the remaining planning steps — subsequent prompts will not repeat its contents. In particular, the success metrics must be carried into the Testing Approach step later (each becomes a behavioural test or a flagged manual check), because the implement workflow reads the plan, not the spec.

Once you have read and understood the spec, advance to the next step:

{{config.command}} plan goto --data '{"step":"{{next_step}}"}'

---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
