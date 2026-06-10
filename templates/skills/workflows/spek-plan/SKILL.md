---
name: spek-plan
description: Create a new Plan from an approved Specification.
---

# What this skill does

This skill drives a **multi-step interactive workflow** that produces a complete implementation plan in `.spektacular/plans/<name>.md` from an existing spec. The workflow is owned by the `{{command}}` CLI, not by you — the CLI is the state machine and you are the executor.

On each turn, the CLI returns JSON containing an `instruction` field. That instruction describes exactly one step (e.g. discovery, data structures, phases, testing approach, …). You must:

1. Read the `instruction` carefully.
2. Perform the step — this may mean researching the codebase, spawning subagents, interviewing the user, or committing a plan document to the store.
3. When the step is complete, run the `goto` command named at the bottom of the instruction to advance the state machine.
4. Read the next `instruction` from the new JSON response and repeat.

**This is a loop. Do not stop after the first step.** Keep looping — step → goto → next instruction → step — until a returned instruction tells you the workflow is *finished*. Only then should you report completion to the user.

# Reading and writing plan files

The CLI owns the plan documents — `plan.md`, `context.md`, and `research.md`. **Never read or write them with the `Write`, `Edit`, or `Read` tools** — those bypass Spektacular and the configured plan directory. All plan document access goes through `{{command}} plan file`:

- `{{command}} plan file read <name>/<doc>.md` — read a plan document from the plan store.
- `{{command}} plan file write <name>/<doc>.md --from <source-path>` — write a plan document into the plan store from a source file on disk. Stage the body under `.spektacular/tmp/` first, then `rm` the scratch file after a successful write.
- `{{command}} plan file list` — list plans in the plan store.

Path arguments are plan-directory-relative document paths (e.g. `my-feature/plan.md`); `plan file` resolves them against the configured plan directory itself.

# Working files vs. the store documents

While you gather each section, write that section's agreed content directly to its own git-tracked working file under `.spektacular/work/<plan_name>/<section>.md` using your own `Write` tool (the phases step writes two: `phases_plan.md` and `phases_context.md`). These working files are **not** store documents — writing them directly with `Write` is correct and expected, and is the one deliberate exception to the "never use `Write`/`Edit`" rule above. That rule protects only the **final assembled** `plan.md`, `context.md`, and `research.md`, which are written solely through `{{command}} plan file write`. The per-section working files are scratch-but-durable: the assemble step reads them back to build the three documents (staged to `.spektacular/tmp/`), the verification step checks the staged documents, the write steps commit them, and then the working directory is removed once all three store writes succeed.

The working sidecar `.spektacular/context.md` (at the repo's `.spektacular/` root — not the plan's own `context.md` document) has a narrower role: it holds only your cross-cutting learnings and the answers the user gave to your questions — never a copy of section content (that lives in the per-section working files). On resume, read back **both** the section working files in `.spektacular/work/<plan_name>/` and `.spektacular/context.md`, so you continue from the interrupted step without re-asking for sections already completed.

# How to start

Spec name: $ARGUMENTS

If no spec name was provided, ask the user which spec to plan against before proceeding. You don't need to look for an in-progress workflow yourself — the CLI detects and reports one for you (see below).

Start the plan workflow by running:

```
{{command}} plan new --data '{"name": "<spec_name>"}'
```

**If a workflow was interrupted and is still in progress**, this command does not start a fresh one. Instead it returns a *resume report* — a JSON object with `"resumable": true` plus the in-progress workflow's `kind`, `name`, and `current_step`, and an `instruction` field — and changes nothing on disk. When you get a resume report:

**First check the report's `kind`.** If it is **not** `plan`, a *different* workflow (a spec or implement run) is in progress — you cannot resume it from the plan skill, and the CLI will refuse to. Do **not** run a `plan goto`. Instead follow the report's `instruction`: tell the user a `<kind>` workflow is in progress and let them choose — continue it with that workflow's skill (`{{command}} <kind> goto`), or discard it and start the plan with `{{command}} plan new --force`. Only proceed with the steps below when the report's `kind` is `plan`.

1. Ask the user whether to **resume** the in-progress plan or **start a new one**. (The report's `instruction` field restates both options.)
2. **To resume**, first read back the previous session's work with your own file tools: the per-section working files under `.spektacular/work/<name>/` (sections already completed) **and** `.spektacular/context.md` (learnings + the user's answers). Then run the resume command using the report's `current_step`:

   ```
   {{command}} plan goto --data '{"step":"<current_step>"}'
   ```
3. **To start fresh** (discarding the in-progress workflow — it remains recoverable via git), re-run with `--force`:

   ```
   {{command}} plan new --force --data '{"name": "<spec_name>"}'
   ```

Otherwise the command returns the first `instruction` and a fresh workflow has started. From that point on, follow the loop above: do what the instruction says, then call `{{command}} plan goto --data '{"step":"<next_step>"}'` to get the next one. Do not invent step names — every instruction tells you the exact `goto` command to run next.
