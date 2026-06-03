---
name: spek-implement
description: Execute an approved Plan to implement the feature.
---

# What this skill does

This skill drives a **multi-step interactive workflow** that executes an approved plan in `.spektacular/plans/<name>/plan.md`, producing working code, tests, and a changelog. The workflow is owned by the `{{command}}` CLI, not by you — the CLI is the state machine and you are the executor.

On each turn, the CLI returns JSON containing an `instruction` field. That instruction describes exactly one step (e.g. analyze, implement a phase, verify, update changelog, …). You must:

1. Read the `instruction` carefully.
2. Perform the step — this may mean reading the plan, spawning subagents, editing code, running tests, or writing to the changelog.
3. When the step is complete, run the `goto` command named at the bottom of the instruction to advance the state machine.
4. Read the next `instruction` from the new JSON response and repeat.

**This is a loop. Do not stop after the first step.** Keep looping — step → goto → next instruction → step — until a returned instruction tells you the workflow is *finished*. Only then should you report completion to the user.

# Reading and writing plan files

The CLI owns the plan documents — `plan.md`, `context.md`, and `research.md`. **Never read or write them with the `Write`, `Edit`, or `Read` tools** — those bypass Spektacular and the configured plan directory. All plan document access goes through `{{command}} plan file`:

- `{{command}} plan file read <name>/<doc>.md` — read a plan document from the plan store.
- `{{command}} plan file write <name>/<doc>.md --from <source-path>` — write a plan document into the plan store from a source file on disk. Stage the body under `.spektacular/tmp/` first, then `rm` the scratch file after a successful write.

This includes the edits the implement workflow makes to `plan.md` — ticking phase checkboxes and appending changelog entries. Read the document with `plan file read`, apply the change, and commit it with `plan file write`. Never edit a plan document in place with the `Edit` tool. Path arguments are plan-directory-relative document paths (e.g. `my-feature/plan.md`).

# How to start

Plan name: $ARGUMENTS

If no plan name was provided, ask the user which plan to implement before proceeding. You don't need to look for an in-progress workflow yourself — the CLI detects and reports one for you (see below).

The plan file must already exist at `.spektacular/plans/<plan_name>/plan.md`. If it does not, stop and tell the user to run `{{command}} plan` first.

Start the implement workflow by running:

```
{{command}} implement new --data '{"name": "<plan_name>"}'
```

**If a workflow was interrupted and is still in progress**, this command does not start a fresh one. Instead it returns a *resume report* — a JSON object with `"resumable": true` plus the in-progress workflow's `kind`, `name`, and `current_step`, and an `instruction` field — and changes nothing on disk. When you get a resume report:

1. Ask the user whether to **resume** the in-progress workflow or **start a new one**. (The report's `instruction` field restates both options.)
2. **To resume**, first read `.spektacular/context.md` — the git-tracked working-context file the previous session left behind — to recover its learnings and the answers you gave to the user's questions, then run the resume command using the report's `kind` and `current_step` (follow the report's `instruction` field, which is tailored to that kind):

   ```
   {{command}} <kind> goto --data '{"step":"<current_step>"}'
   ```

   The in-progress workflow may be a *different* kind (a spec or plan left open); use the `kind` from the report, not necessarily `implement`.
3. **To start fresh** (discarding the in-progress workflow — it remains recoverable via git), re-run with `--force`:

   ```
   {{command}} implement new --force --data '{"name": "<plan_name>"}'
   ```

Otherwise the command returns the first `instruction` and a fresh workflow has started. From that point on, follow the loop above: do what the instruction says, then call `{{command}} implement goto --data '{"step":"<next_step>"}'` to get the next one. Do not invent step names — every instruction tells you the exact `goto` command to run next.
