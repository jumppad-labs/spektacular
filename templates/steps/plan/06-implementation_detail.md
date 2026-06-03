## Step {{step}}: {{title}}

Draft the **Implementation Detail** section of `plan.md`.

### This section is high-level only

Sketch new patterns being introduced, major code-shape changes, and code-structure UX — enough for a reviewer to spot missing patterns or design gaps. This section is **high-level only**. Per-phase file:line work stays in `context.md`.

If you find yourself writing "in file X at line Y", stop and move that content to context.md. The test for "too low-level" is: could this be written before the phases are defined? If no, it belongs in context.md.

### What to include

- New patterns the plan introduces (e.g. a new state machine, a new agent orchestration shape, a new module boundary)
- Major code-shape changes (e.g. splitting one package into three, or introducing a new interface that replaces a concrete type across the codebase)
- Code-structure UX: what a developer reading the changed code will experience
- Where existing patterns in the codebase are being followed vs. where new ones are being introduced

### What NOT to include

- Specific file paths or line numbers
- Per-phase file changes
- Function signatures that belong in § Data Structures & Interfaces
- Shell commands

### What to produce

A draft Implementation Detail section ready to drop into plan.md at verification time. Present it to the user for review.

Before advancing, save this section to its working file. Using your own `Write` tool, write the agreed **Implementation Detail** content (body only — no `## ` heading line) to `.spektacular/work/{{plan_name}}/implementation_detail.md`. This working file is git-tracked and is read back on resume and when the plan documents are assembled, so it must hold the final content. It is **not** a plan store document — write it directly with your file tools and do **not** route it through `{{config.command}} plan file write` (that command is only for the final plan documents).

Once the user is happy with the sketch, advance:

{{config.command}} plan goto --data '{"step":"{{next_step}}"}'

---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
