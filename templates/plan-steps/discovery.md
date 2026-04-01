## Step {{step}}: {{title}}

Research the codebase to understand what's needed for: **{{overview}}**

### Step 1: Project Context

Check if reference docs exist and create them if missing:

- Check if `thoughts/notes/commands.md` exists. If not, use the `discover-project-commands` skill. For skill details: `{{config.command}} skill discover-project-commands`
- Check if `thoughts/notes/testing.md` exists. If not, use the `discover-test-patterns` skill. For skill details: `{{config.command}} skill discover-test-patterns`

### Step 2: Codebase Research

Research the codebase in parallel to find:

1. **Files related to the plan** — Organize by category (implementation, tests, config, docs)
2. **Prior research** — Find existing plans, research, or tickets in `thoughts/`, `.spektacular/plans/`, `.spektacular/specs/`
3. **Similar implementations** — Find code examples that can be modelled after, with file:line references
4. **Architecture and integration points** — How do the relevant components fit together?

Use your agent orchestration capability to parallelize this research. For guidance on structuring agent orchestration: `{{config.command}} skill spawn-planning-agents`

### Step 3: Read and Clarify

- Read all findings fully
- Ask only questions that the code cannot answer
- Present a summary of key discoveries to the user

Once research is complete, advance:

{{config.command}} plan goto --data '{"step":"{{next_step}}"}'
