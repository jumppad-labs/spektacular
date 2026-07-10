## Step {{step}}: {{title}}

{{#plan_incomplete}}
⚠️ One or more plan documents are missing from the plan store, or still hold the empty scaffold. Before telling the user the workflow is done, commit the missing documents through the CLI and remove the scratch files:

```
{{config.command}} plan file write {{plan_name}}/plan.md     --from .spektacular/tmp/plan_template.md
{{config.command}} plan file write {{plan_name}}/context.md  --from .spektacular/tmp/context_template.md
{{config.command}} plan file write {{plan_name}}/research.md --from .spektacular/tmp/research_template.md
rm .spektacular/tmp/plan_template.md .spektacular/tmp/context_template.md .spektacular/tmp/research_template.md
```

If a scratch file under `.spektacular/tmp/` is gone (that path is git-ignored and does not survive a crash), re-assemble the affected document from the per-section working files under `.spektacular/work/{{plan_name}}/` before committing — they are the durable source.

Never write or edit the plan documents with the `Write` or `Edit` tools — `{{config.command}} plan file write` is the only supported way to write them. Verify each document with `{{config.command}} plan file read {{plan_name}}/<doc>.md`, then re-run this step.
{{/plan_incomplete}}
{{^plan_incomplete}}
The plan workflow is complete. Three documents are now in the plan store under `{{plan_dir}}`:

- `{{plan_path}}` — the user-scannable plan
- `{{context_path}}` — technical detail for implementation
- `{{research_path}}` — the decision log and rehydration cues

Read any of them back with `{{config.command}} plan file read {{plan_name}}/<doc>.md`.

Inform the user that the plan workflow is finished and the three documents are ready for review. Then offer a choice, once: read the documents directly (the path above, unchanged), or have you walk through the plan conversationally. If the user declines, or doesn't take you up on it, the workflow is complete exactly as described above — do not repeat the offer.

If the user accepts, present the plan as a colleague would explain it aloud, not as a recitation of `{{plan_path}}`'s section text. Pace it as a small number of natural beats with a brief pause between each, rather than one uninterrupted monologue or a rigid sign-off gate after every point:

1. The chosen approach and why it was picked over the alternatives.
2. How the work breaks into stages/phases.
3. What was deliberately left out of scope, and why.

At any point during the walkthrough, the user may interrupt with a question or a request to change something. Respond to it directly in your next message before returning to the beat you were on. If the interruption is a change request, apply it immediately rather than deferring it: edit the affected section, stage the updated document under `.spektacular/tmp/`, commit it with `{{config.command}} plan file write {{plan_name}}/plan.md --from <scratch>` (the same command shown above), confirm the update to the user, remove the scratch file, then resume the walkthrough, re-summarizing anything the change affected.

Only conclude the walkthrough once the user gives an explicit affirmative answer to a direct closing question (e.g. "does this look right, ready to proceed?"). Silence, a topic change, or an ambiguous reply is not agreement — keep the walkthrough open until they explicitly confirm.
{{/plan_incomplete}}
