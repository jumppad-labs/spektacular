## Step {{step}}: {{title}}

{{#plan_unwritten}}
⚠️ plan.md was not found in the plan store, or still holds the empty scaffold — it was never committed. Commit the filled plan.md before continuing:

```
cat .spektacular/tmp/plan_template.md | {{config.command}} plan file write {{plan_name}}/plan.md
```
{{/plan_unwritten}}
{{^plan_unwritten}}
plan.md has been committed to the plan store.
{{/plan_unwritten}}

Now commit context.md the same way. Use the `Write` tool to stage the filled context.md at the scratch path `.spektacular/tmp/context_template.md`, then commit it to the plan store:

```
cat .spektacular/tmp/context_template.md | {{config.command}} plan file write {{plan_name}}/context.md
```

Never write or edit the plan documents with the `Write` or `Edit` tools — `{{config.command}} plan file write` is the only supported way to write them.

Then advance:

```
{{config.command}} plan goto --data '{"step":"{{next_step}}"}'
```
