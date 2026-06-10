---
name: spek-new
description: Create a new Specification for a feature.
---

# What this skill does

This skill drives a **multi-step interactive workflow** that produces a complete specification file at the `spec_path` returned by the CLI. The workflow is owned by the `{{command}}` CLI, not by you — the CLI is the state machine and you are the executor.

On each turn, the CLI returns JSON containing an `instruction` field. That instruction describes exactly one step (e.g. overview, requirements, acceptance criteria, …). You must:

1. Read the `instruction` carefully.
2. Perform the step — usually this means interviewing the user and capturing their answers. Some steps tell you to commit the gathered content to the spec file.
3. When the step is complete, run the `goto` command named at the bottom of the instruction to advance the state machine.
4. Read the next `instruction` from the new JSON response and repeat.

**This is a loop. Do not stop after the first step.** Keep looping — step → goto → next instruction → step — until a returned instruction tells you the workflow is *finished*. Only then should you report completion to the user.

# Reading and writing the spec file

The CLI owns the spec file. **Never read or write it with the `Write`, `Edit`, or `Read` tools** — those bypass Spektacular and the configured spec directory. All spec file access goes through `{{command}} spec file`:

- `{{command}} spec file read <name>.md` — read a spec file from the spec store.
- `{{command}} spec file write <name>.md --from <source-path>` — write a spec file into the spec store from a source file on disk. Stage the body under `.spektacular/tmp/` first, then `rm` the scratch file after a successful write.
- `{{command}} spec file list` — list spec files in the spec store.

Path arguments are spec file names; `spec file` resolves them against the configured spec directory itself.

# Working files vs. the store document

While you gather each section, write that section's agreed content directly to its own git-tracked working file under `.spektacular/work/<spec_name>/<section>.md` using your own `Write` tool. These working files are **not** store documents — writing them directly with `Write` is correct and expected, and is the one deliberate exception to the "never use `Write`/`Edit`" rule above. That rule protects only the **final assembled** spec, which is written solely through `{{command}} spec file write`. The per-section working files are scratch-but-durable: the verification step reads them back to assemble the final spec, and then the working directory is removed once the store write succeeds.

`.spektacular/context.md` has a narrower role: it holds only your cross-cutting learnings and the answers the user gave to your questions — never a copy of section content (that lives in the per-section working files). On resume, read back **both** the section working files in `.spektacular/work/<spec_name>/` and `.spektacular/context.md`, so you continue from the interrupted step without re-asking for sections already completed.

# How to start

Spec name: $ARGUMENTS

**First, check whether a workflow is already in progress — before asking the user for a spec name.** Run the new command with no `--data`:

```
{{command}} spec new
```

This reads the project's single workflow state and changes nothing on disk. One of two things comes back:

- **A resume report** — a JSON object with `"resumable": true` plus the in-progress workflow's `kind`, `name`, and `current_step`, and an `instruction` field. A workflow was interrupted and is still in progress. Do **not** prompt for a spec name — the in-progress workflow already has one. Handle it under "Resuming an in-progress workflow" below. (It may be a *different* kind — a plan or implement run left open.)
- **An error that a name is required** — no workflow is in progress, so there is nothing to resume. Proceed to "Starting a new spec" below.

## Starting a new spec

Only once you know there is no workflow to resume:

If no spec name was provided in $ARGUMENTS, ask the user for one now. Then run:

```
{{command}} spec new --data '{"name": "<spec_name>"}'
```

External systems may also supply an identifier with:

```
{{command}} spec new --data '{"name": "<spec_name>", "id": "<external_id>"}'
```

The CLI may normalize and prefix the requested name. Always use the returned `spec_name` and `spec_path` as the source of truth for follow-up workflows.

The command creates the spec file and state file automatically and returns the first `instruction`. From that point on, follow the loop above: do what the instruction says, then call `{{command}} spec goto --data '{"step":"<next_step>"}'` to get the next one. Do not invent step names — every instruction tells you the exact `goto` command to run next.

## Resuming an in-progress workflow

When the in-progress check above returns a resume report:

**First check the report's `kind`.** If it is **not** `spec`, a *different* workflow (a plan or implement run) is in progress — you cannot resume it from the spec skill, and the CLI will refuse to. Do **not** run a `spec goto`. Instead follow the report's `instruction`: tell the user a `<kind>` workflow is in progress and let them choose — continue it with that workflow's skill (`{{command}} <kind> goto`), or discard it and start the spec with `{{command}} spec new --force`. Only proceed with the steps below when the report's `kind` is `spec`.

1. Ask the user whether to **resume** the in-progress spec or **start a new one**. (The report's `instruction` field restates both options.)
2. **To resume**, first read back the previous session's work with your own file tools: the per-section working files under `.spektacular/work/<name>/` (sections already completed) **and** `.spektacular/context.md` (learnings + the user's answers). Then run the resume command using the report's `current_step`:

   ```
   {{command}} spec goto --data '{"step":"<current_step>"}'
   ```
3. **To start fresh instead** (discarding the in-progress workflow — it remains recoverable via git), re-run with `--force` and a name:

   ```
   {{command}} spec new --force --data '{"name": "<spec_name>"}'
   ```
