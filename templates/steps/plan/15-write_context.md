## Step {{step}}: {{title}}

{{#context_unwritten}}
⚠️ context.md was not found in the plan store, or still holds the empty scaffold — it was never committed. Commit the filled context.md before continuing, then remove the scratch file:

```
{{config.command}} plan file write {{plan_name}}/context.md --from .spektacular/tmp/context_template.md
rm .spektacular/tmp/context_template.md
```
{{/context_unwritten}}
{{^context_unwritten}}
context.md has been committed to the plan store.
{{/context_unwritten}}

Now commit research.md the same way. Use the `Write` tool to stage the filled research.md at the scratch path `.spektacular/tmp/research_template.md`, point `plan file write` at it with `--from`, then remove the scratch file:

```
{{config.command}} plan file write {{plan_name}}/research.md --from .spektacular/tmp/research_template.md
rm .spektacular/tmp/research_template.md
```

Never write or edit the plan documents with the `Write` or `Edit` tools — `{{config.command}} plan file write` is the only supported way to write them.

Then advance:

```
{{config.command}} plan goto --data '{"step":"{{next_step}}"}'
```
