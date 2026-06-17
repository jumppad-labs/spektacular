## Step {{step}}: {{title}}

Research the codebase to understand what's needed to implement the spec you read in the previous step. The output of this step is `research.md` — a **decision log**, not a transcript — that captures your investigation so a future cold session can rehydrate without re-doing the work.

### Step 1: Project Context

First, load the project's **always-applied** knowledge in full with `{{config.command}} knowledge always-applied`. This returns every always-applied entry's body — the **conventions** and the **glossary** — each tagged with the scope it came from (e.g. `project`, `team`, `global`) and its category. You only need to load this once: it then stays in your context for the rest of the plan. Read all of it — the conventions are the standards every plan must reflect (the next step selects the relevant subset into the plan), and the glossary is the shared vocabulary you need to read the rest of the knowledge base and the code correctly.

Then pull in topic-specific knowledge for the **design surfaces** this feature introduces — the concrete areas the work touches (for example `database`, `auth`, `http-middleware`, `cli`). For each surface, run a targeted `{{config.command}} knowledge search <surface>` rather than one broad "anything about this area" sweep; this keeps research focused as the knowledge base grows. Results come back ranked — one per matching document, strongest match first, each tagged with its scope and category and carrying a title, score, and excerpts; the category label tells you what kind of knowledge a hit is (e.g. a `gotchas` warning, an `architecture` fact, a `learnings` finding). Read a promising one in full with `{{config.command}} knowledge read --data '{"scope":"<scope>","path":"<path>"}'`. The always-applied categories (conventions and glossary) are deliberately excluded from search results (you already loaded them in full above), so search surfaces only reference looked-up knowledge — architecture notes, gotchas, prior learnings. If something relevant exists, read it before investigating; it may already answer your questions or flag dead ends. Nothing is required to exist — the knowledge sources can be empty.

If the plan touches tests, read the relevant test files directly as part of Step 2 to understand conventions (framework, naming, fixtures, mocking) before planning changes. Don't cache findings — the test files are the source of truth.

### Step 2: Codebase Research

Research the codebase in parallel to find:

1. **Files related to the plan** — Organize by category (implementation, tests, config, docs)
2. **Prior research** — Existing plans, research, or tickets: search the knowledge sources with `{{config.command}} knowledge search <query>`, list prior plans with `{{config.command}} plan file list` and prior specs with `{{config.command}} spec file list`, then read any that look relevant with `{{config.command}} plan file read <name>/plan.md` or `{{config.command}} spec file read <name>.md`. Always reach plans and specs through these CLI commands — never with the `Read` tool
3. **Similar implementations** — Code examples to model after, with file:line references
4. **Architecture and integration points** — How the relevant components fit together
5. **Alternatives to consider** — Identify 2-3 viable approaches so the next step can compare them with evidence

Use your agent orchestration capability to parallelize this research. For guidance: `{{config.command}} skill spawn-planning-agents`

### Step 3: Distill findings into research.md — the decision log

You are gathering the content for research.md's required sections — you will save it to a working file at the end of this step, and the verification step assembles it into the store document. The required sections are:

- **Alternatives considered and rejected** — options you considered; for each, what it is, why rejected, with citation (file:line or external reference). This prevents future agents from re-proposing the same dead ends.
- **Chosen approach — evidence** — the file:line or external references that support the option you'll recommend in the next step. Evidence, not the decision itself.
- **Files examined** — terse one-liner per file: `path:line — what was learned`.
- **External references** — papers, RFCs, library docs, blog posts, with a one-line "why this mattered".
- **Prior plans / specs consulted** — links with what was learned from each.
- **Open assumptions** — things assumed but not verified. If any turn out wrong, the implement workflow must STOP and ask.
- **Rehydration cues** — skill invocations and file re-reads that can rebuild context from cold.

Keep this dense. Assume a future agent will read it cold and need to make decisions from it.

### Step 4: Read and Clarify

- Read all findings fully
- Ask only questions the code cannot answer
- Present a summary of key discoveries to the user

### Step 5: Capturing a learning (optional)

If your research surfaces a durable learning, gotcha, or convention worth keeping for future plans, you may persist it with `{{config.command}} knowledge write`. Before writing, run `{{config.command}} knowledge sources` to see the available scopes, then **propose to the user a target scope and the exact content you intend to write, and wait for explicit confirmation**. Do not invoke `{{config.command}} knowledge write` until the user has confirmed. Propose, then wait for confirmation — never write to a knowledge source unprompted.

Before advancing, save your research to its working file. Using your own `Write` tool, write the distilled research content — all seven sections above (alternatives, chosen-approach evidence, files examined, external references, prior plans/specs, open assumptions, rehydration cues), body content only — to `.spektacular/work/{{plan_name}}/research.md`. This working file is git-tracked and is read back on resume and when the plan documents are assembled, so it must hold the final content. It is **not** a plan store document — write it directly with your file tools and do **not** route it through `{{config.command}} plan file write` (that command is only for the final plan documents).

Once research is complete, advance:

{{config.command}} plan goto --data '{"step":"{{next_step}}"}'

---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
