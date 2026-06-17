## Step {{step}}: {{title}}

Decide the shape of the solution and lock in the chosen direction. This step produces the **Architecture & Design Decisions** section of `plan.md` — the load-bearing section of the whole plan. A reviewer should be able to spot missing patterns or design gaps from this section alone.

### Step 1: Present Options

Present 2-3 design options to the user. For each:

- **Option name** and brief description
- **Pros**: Advantages, with file:line references from research.md where they apply
- **Cons**: Disadvantages, risks, complexity
- **Effort estimate**: Relative complexity (Low / Medium / High)

Ground each option in the research findings gathered during the discovery step.

### Step 2: Get Agreement

Get the user's explicit agreement on:

1. **Chosen direction** — which option to pursue
2. **Key design decisions** — the non-obvious trade-offs the chosen direction makes

Rejected options go into `research.md § Alternatives considered and rejected` with citations when the verification step writes the files.

### Step 3: Draft the Architecture & Design Decisions section

Draft 2-4 short paragraphs describing:

- The shape of the chosen solution
- The key design decisions and their trade-offs
- Why this direction beats the alternatives
- A cross-reference to `research.md#alternatives-considered-and-rejected` so plan.md readers can drill into the evidence

Keep this section self-contained. Do NOT write `see context.md for …` — plan.md must stand on its own for readers outside the Milestones & Phases block.

Before advancing, save this section to its working file. Using your own `Write` tool, write the agreed **Architecture & Design Decisions** content (body only — no `## ` heading line) to `.spektacular/work/{{plan_name}}/architecture.md`. This working file is git-tracked and is read back on resume and when the plan documents are assembled, so it must hold the final content. It is **not** a plan store document — write it directly with your file tools and do **not** route it through `{{config.command}} plan file write` (that command is only for the final plan documents).

### Step 4: Select the conventions that apply

Now the design shape is locked and you know the surfaces this feature touches, select — from the conventions you loaded in full during discovery (the conventions-category entries returned by `{{config.command}} knowledge always-applied`) — the subset that actually bears on this work. For each one you keep, write a one-line rationale for **why it applies to this feature**, and cite it inline in the Architecture & Design Decisions content above wherever it drives a specific choice. Include only the genuinely relevant conventions — not the whole knowledge base.

Relevance is **proposed, not auto-decided**: present the conventions you propose to apply (and any you deliberately dropped) to the user and get their confirmation before saving. If no conventions are relevant, or the project has none, say so plainly rather than padding the list — an empty or generic list is a visible signal the knowledge base was not consulted.

Save the result to its working file. Using your own `Write` tool, write the **Conventions** content (body only — no `## ` heading line) to `.spektacular/work/{{plan_name}}/conventions.md`. Use a bullet list, one convention per line in the form `- **<convention name / one-line summary>** — <why it applies to this feature>.`. When nothing applies, write a single explicit sentence instead, e.g. `No project conventions apply to this feature.`. This working file is always required — write it even in the "none apply" case — and like the others it is git-tracked, read back on resume and when the plan documents are assembled; do **not** route it through `{{config.command}} plan file write`.

Once the user has agreed on the chosen direction and the draft is ready, advance:

{{config.command}} plan goto --data '{"step":"{{next_step}}"}'

---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
