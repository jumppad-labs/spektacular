---
name: spek-new
description: Create a new Specification for a feature.
---

# What this skill does

This skill drives a **multi-step interactive workflow** that produces a complete specification file at the `spec_path` returned by the CLI. The workflow is owned by the `{{command}}` CLI, not by you — the CLI is the state machine and you are the executor.

On each turn, the CLI returns JSON containing an `instruction` field. That instruction describes exactly one step (e.g. overview, requirements, acceptance criteria, …). You must:

1. Read the `instruction` carefully.
2. Perform the step — usually this means interviewing the user, capturing their answers, and writing the relevant section of the spec file.
3. When the step is complete, run the `goto` command named at the bottom of the instruction to advance the state machine.
4. Read the next `instruction` from the new JSON response and repeat.

**This is a loop. Do not stop after the first step.** Keep looping — step → goto → next instruction → step — until a returned instruction tells you the workflow is *finished*. Only then should you report completion to the user.

# How to start

Spec name: $ARGUMENTS

If no spec name was provided, ask the user for one before proceeding.

Start the spec workflow by running:

```
{{command}} spec new --data '{"name": "<spec_name>"}'
```

External systems may also supply an identifier with:

```
{{command}} spec new --data '{"name": "<spec_name>", "id": "<external_id>"}'
```

## Notion artifact mode

Before starting, run:

```
{{command}} notion status
```

If the returned `status` is `configured`, artifacts are Notion-backed. Use Notion MCP to create or fetch the Spec page first, then read its required `Spec ID` auto-increment value and returned metadata. Start the workflow with the Notion ID as the external identifier and include the normalized remote metadata:

```
{{command}} spec new --data '{"name":"<spec_name>","remote":{"notion_url":"<page_url>","page_id":"<page_id>","data_source_url":"<specs_data_source>","external_id":"<spec_id>","remote_version":"<last_edited_time>","title":"<title>"}}'
```

When the final spec content is ready, update the Notion Spec page body through Notion MCP first. Then submit the same content to Spektacular with the returned remote metadata so the local cache and manifest are aligned:

```
{{command}} spec goto --data '{"step":"finished","remote":{"notion_url":"<page_url>","page_id":"<page_id>","data_source_url":"<specs_data_source>","external_id":"<spec_id>","remote_version":"<new_last_edited_time>","title":"<title>"}}' --file .spektacular/tmp/spec_template.md
```

If Notion reports that the page changed since the local baseline, stop and use `{{command}} notion cache prepare-push` to surface the merge request. Present the baseline, local, and remote content to the user/agent, run `{{command}} notion cache resolve-merge` after resolution, then retry the Notion update.

The CLI may normalize and prefix the requested name. Always use the returned `spec_name` and `spec_path` as the source of truth for follow-up workflows.

This creates the spec file and state file automatically and returns the first `instruction`. From that point on, follow the loop above: do what the instruction says, then call `{{command}} spec goto --data '{"step":"<next_step>"}'` to get the next one. Do not invent step names — every instruction tells you the exact `goto` command to run next.
