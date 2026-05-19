## Step {{step}}: {{title}}

{{#research_unwritten}}
⚠️ research.md was not found in the plan store, or still holds the empty scaffold — it was never committed. Commit the filled research.md before continuing:

```
cat .spektacular/tmp/research_template.md | {{config.command}} plan file write {{plan_name}}/research.md
```
{{/research_unwritten}}
{{^research_unwritten}}
research.md has been committed to the plan store.

All three plan documents are in the plan store.
{{/research_unwritten}}

Advance to the finished step:

```
{{config.command}} plan goto --data '{"step":"{{next_step}}"}'
```
