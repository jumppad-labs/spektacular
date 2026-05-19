## Step {{step}}: {{title}}

{{#plan_incomplete}}
⚠️ One or more plan documents are missing from the plan store, or still hold the empty scaffold. Before telling the user the workflow is done, commit the missing documents through the CLI:

```
cat .spektacular/tmp/plan_template.md     | {{config.command}} plan file write {{plan_name}}/plan.md
cat .spektacular/tmp/context_template.md  | {{config.command}} plan file write {{plan_name}}/context.md
cat .spektacular/tmp/research_template.md | {{config.command}} plan file write {{plan_name}}/research.md
```

Never write or edit the plan documents with the `Write` or `Edit` tools — `{{config.command}} plan file write` is the only supported way to write them. Verify each document with `{{config.command}} plan file read {{plan_name}}/<doc>.md`, then re-run this step.
{{/plan_incomplete}}
{{^plan_incomplete}}
The plan workflow is complete. Three documents are now in the plan store under `{{plan_dir}}`:

- `{{plan_path}}` — the user-scannable plan
- `{{context_path}}` — technical detail for implementation
- `{{research_path}}` — the decision log and rehydration cues

Read any of them back with `{{config.command}} plan file read {{plan_name}}/<doc>.md`.

Inform the user that the plan workflow is finished and the three documents are ready for review.
{{/plan_incomplete}}
