---
name: spek-plan
description: Create a new Plan from an approved Specification.
---

# What this skill does

This skill drives a **multi-step interactive workflow** that produces a complete implementation plan in `.spektacular/plans/<name>.md` from an existing spec. The workflow is owned by the `go run .` CLI, not by you — the CLI is the state machine and you are the executor.

On each turn, the CLI returns JSON containing an `instruction` field. That instruction describes exactly one step (e.g. discovery, data structures, phases, testing approach, …). You must:

1. Read the `instruction` carefully.
2. Perform the step — this may mean researching the codebase, spawning subagents, interviewing the user, or writing a section of the plan file.
3. When the step is complete, run the `goto` command named at the bottom of the instruction to advance the state machine.
4. Read the next `instruction` from the new JSON response and repeat.

**This is a loop. Do not stop after the first step.** Keep looping — step → goto → next instruction → step — until a returned instruction tells you the workflow is *finished*. Only then should you report completion to the user.

# How to start

Spec name: $ARGUMENTS

If no spec name was provided, check `.spektacular/state.json` for an active spec under `data.name`. If one exists, ask the user whether they want to plan against that spec, offering the option to name a different one. If no active spec is found, ask the user which spec to plan against before proceeding.

Start the plan workflow by running:

```
go run . plan new --data '{"name": "<spec_name>"}'
```

## Notion artifact mode

Before starting, run:

```
go run . notion status
```

If the returned `status` is `configured`, artifacts are Notion-backed. Ensure the Spec cache exists before planning. If another user or agent already created the spec, locate it by Notion URL, page ID, or external Spec ID; if it is missing locally, fetch the Spec page through Notion MCP and record it:

```
go run . notion cache pull --data '{"kind":"spec","name":"<spec_name>","content":"<spec_markdown>","remote":{"notion_url":"<page_url>","page_id":"<page_id>","data_source_url":"<specs_data_source>","external_id":"<spec_id>","remote_version":"<last_edited_time>","title":"<title>"}}'
```

Create or fetch the Plan row through Notion MCP before the write steps. Use the required `Plan ID` auto-increment value as `remote.external_id`. Create or fetch child pages for `context.md` and `research.md` under the Plan page. When submitting each artifact-changing step, update Notion first, then pass the returned metadata with the file:

```
go run . plan goto --data '{"step":"write_plan","remote":{"notion_url":"<plan_page_url>","page_id":"<plan_page_id>","data_source_url":"<plans_data_source>","external_id":"<plan_id>","remote_version":"<new_last_edited_time>","title":"<title>"}}' --file .spektacular/tmp/plan_template.md
go run . plan goto --data '{"step":"write_context","context_remote":{"notion_url":"<context_page_url>","page_id":"<context_page_id>","data_source_url":"<plans_data_source>","external_id":"<context_id>","remote_version":"<new_last_edited_time>","title":"context.md"}}' --file .spektacular/tmp/context_template.md
go run . plan goto --data '{"step":"write_research","research_remote":{"notion_url":"<research_page_url>","page_id":"<research_page_id>","data_source_url":"<plans_data_source>","external_id":"<research_id>","remote_version":"<new_last_edited_time>","title":"research.md"}}' --file .spektacular/tmp/research_template.md
```

Before any Notion update, use `go run . notion cache prepare-push` with the latest Notion `remote_version`. If it returns `merge_required`, stop and present the merge request; after resolution, run `go run . notion cache resolve-merge` and retry.

This creates the plan file and state file automatically and returns the first `instruction`. From that point on, follow the loop above: do what the instruction says, then call `go run . plan goto --data '{"step":"<next_step>"}'` to get the next one. Do not invent step names — every instruction tells you the exact `goto` command to run next.
