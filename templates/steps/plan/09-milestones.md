## Step {{step}}: {{title}}

Define 2-4 milestones that deliver the spec you have in context.

Each milestone must have:

- **Title**: A user-facing description of what changes (not an engineering label like "Refactor X").
- **What changes**: A one-paragraph summary describing the user-visible difference when the milestone lands. Written in plain language, no file paths, no shell commands. This paragraph is what a reader of plan.md uses to decide whether the milestone is worth doing — make it outcome-focused, not implementation-focused.
- **Validation point**: How to confirm the milestone is done before moving on.

Rules:

- Each milestone should be independently deliverable.
- Milestones should build on each other in order.
- NO open questions — resolve any uncertainties now by asking the user.
- Purely internal cleanups (e.g. refactors with no user-visible effect) are allowed, but the "What changes" paragraph must say so explicitly and explain why the cleanup is worth its own milestone.

Before advancing, save this section to its working file. Using your own `Write` tool, write the agreed **Milestones** content — the milestone titles and their "What changes" paragraphs that open the plan.md `## Milestones & Phases` section (body only, no `## ` heading line) — to `.spektacular/work/{{plan_name}}/milestones.md`. This working file is git-tracked and is read back on resume and when the plan documents are assembled, so it must hold the final content. It is **not** a plan store document — write it directly with your file tools and do **not** route it through `{{config.command}} plan file write` (that command is only for the final plan documents).

Present the milestones to the user for review. Once agreed, advance:

{{config.command}} plan goto --data '{"step":"{{next_step}}"}'

---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
