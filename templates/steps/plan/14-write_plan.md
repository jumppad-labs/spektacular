## Step {{step}}: {{title}}

{{#plan_unwritten}}
⚠️ plan.md was not found in the plan store, or still holds the empty scaffold — it was never committed. Commit the filled plan.md before continuing, then remove the scratch file:

```
{{config.command}} plan file write {{plan_name}}/plan.md --from .spektacular/tmp/plan_template.md
rm .spektacular/tmp/plan_template.md
```
{{/plan_unwritten}}
{{^plan_unwritten}}
plan.md has been committed to the plan store.
{{/plan_unwritten}}

Now commit context.md the same way. Use the `Write` tool to stage the filled context.md at the scratch path `.spektacular/tmp/context_template.md`, point `plan file write` at it with `--from`, then remove the scratch file:

```
{{config.command}} plan file write {{plan_name}}/context.md --from .spektacular/tmp/context_template.md
rm .spektacular/tmp/context_template.md
```

Never write or edit the plan documents with the `Write` or `Edit` tools — `{{config.command}} plan file write` is the only supported way to write them.

Then advance:

```
{{config.command}} plan goto --data '{"step":"{{next_step}}"}'
```
