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
- `{{command}} plan file write <name>/<doc>.md` — write a plan document into the plan store (reads stdin).
- `{{command}} plan file list` — list plans in the plan store.

Path arguments are plan-directory-relative document paths (e.g. `my-feature/plan.md`); `plan file` resolves them against the configured plan directory itself.

# How to start

Spec name: $ARGUMENTS

If no spec name was provided, check `.spektacular/state.json` for an active spec under `data.name`. If one exists, ask the user whether they want to plan against that spec, offering the option to name a different one. If no active spec is found, ask the user which spec to plan against before proceeding.

Start the plan workflow by running:

```
{{command}} plan new --data '{"name": "<spec_name>"}'
```

This creates the plan file and state file automatically and returns the first `instruction`. From that point on, follow the loop above: do what the instruction says, then call `{{command}} plan goto --data '{"step":"<next_step>"}'` to get the next one. Do not invent step names — every instruction tells you the exact `goto` command to run next.
