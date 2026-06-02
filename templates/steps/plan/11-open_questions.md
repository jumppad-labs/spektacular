## Step {{step}}: {{title}}

Draft the **Open Questions** section of `plan.md`.

### Strict scope

This section is **strictly for questions that genuinely cannot be resolved until implementation begins**. Anything that can be answered by asking the user, reading the code, or running a quick experiment must be resolved now — not parked here.

Examples of what belongs here:

- "Whether the downstream API returns X or Y under condition Z — only discoverable by exercising it"
- "Whether the refactor exposes a hidden assumption in an untested code path — will surface during implementation"

Examples of what does NOT belong here:

- "Which library should we use?" → ask the user now
- "What is the current shape of the API?" → read the code now
- "How should we name the new field?" → decide now
- "Is there a test already covering this?" → check now

### What to produce

A draft Open Questions section. If, after a genuine pass, there are no impl-time-only uncertainties, state that explicitly: an empty Open Questions section is the expected healthy outcome.

For every item you keep: state the question, state what it depends on, and state what the implementer should do when they hit it (usually: STOP and ask the user).

Before advancing, save this section to its working file. Using your own `Write` tool, write the agreed **Open Questions** content (body only — no `## ` heading line) to `.spektacular/work/{{plan_name}}/open_questions.md`. This working file is git-tracked and is read back on resume and when the plan documents are assembled, so it must hold the final content. It is **not** a plan store document — write it directly with your file tools and do **not** route it through `{{config.command}} plan file write` (that command is only for the final plan documents).

Once the user agrees the list is correctly scoped, advance:

{{config.command}} plan goto --data '{"step":"{{next_step}}"}'

---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
