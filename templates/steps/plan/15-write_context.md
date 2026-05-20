## Step {{step}}: {{title}}

{{#context_unwritten}}
⚠️ context.md was not found in the plan store, or still holds the empty scaffold — it was never committed. Commit the filled context.md before continuing:

```
cat .spektacular/tmp/context_template.md | {{config.command}} plan file write {{plan_name}}/context.md
```
{{/context_unwritten}}
{{^context_unwritten}}
context.md has been committed to the plan store.
{{/context_unwritten}}

Now commit research.md the same way. Use the `Write` tool to stage the filled research.md at the scratch path `.spektacular/tmp/research_template.md`, then commit it to the plan store:

```
cat .spektacular/tmp/research_template.md | {{config.command}} plan file write {{plan_name}}/research.md
```

Never write or edit the plan documents with the `Write` or `Edit` tools — `{{config.command}} plan file write` is the only supported way to write them.

Then advance:

```
{{config.command}} plan goto --data '{"step":"{{next_step}}"}'
```
