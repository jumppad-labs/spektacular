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

# How to start

Plan name: $ARGUMENTS

If no plan name was provided, check `.spektacular/state.json` for an active plan under `data.name`. If one exists, ask the user whether they want to implement that plan, offering the option to name a different one. If no active plan is found, ask the user which plan to implement before proceeding.

Before enforcing the plan-file precondition, run:

```
{{command}} notion status
```

If the returned `status` is not `configured`, the plan file must already exist at `.spektacular/plans/<plan_name>/plan.md`. If it does not, stop and tell the user to run `{{command}} plan` first.

If Notion mode is configured, the plan cache must exist under `.spektacular/cache/notion/plans/<plan_name>/plan.md`. If it is missing, fetch the Plan page, `context.md`, and `research.md` child pages through Notion MCP and record them with `{{command}} notion cache pull`. You can pull by Notion URL, page ID, or Spektacular external ID when another user or agent already created the work.

During implementation, any step that changes a cached artifact must sync before advancing. Run `{{command}} notion cache prepare-push` with the latest Notion `remote_version`; if it returns `ready`, update Notion through MCP and then run `{{command}} notion cache commit-push` with the returned metadata. If it returns `merge_required`, stop and surface the merge request to the user/agent. After resolution, run `{{command}} notion cache resolve-merge`, retry `prepare-push`, update Notion, and then commit the returned metadata.

Doctor/setup flow: if Notion databases are not linked or validation reports issues, run `{{command}} notion link` to validate existing data sources and `{{command}} notion doctor` for fixable/blocking reports. Only apply additive doctor repairs after explicit user approval.

Start the implement workflow by running:

```
{{command}} implement new --data '{"name": "<plan_name>"}'
```

This creates the state file automatically and returns the first `instruction`. From that point on, follow the loop above: do what the instruction says, then call `{{command}} implement goto --data '{"step":"<next_step>"}'` to get the next one. Do not invent step names — every instruction tells you the exact `goto` command to run next.
