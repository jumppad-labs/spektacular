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

research.md was assembled from your working file `.spektacular/work/{{plan_name}}/research.md`; if the `.spektacular/tmp/research_template.md` scratch file is gone, re-assemble it from there. Now commit research.md the same way. Use the `Write` tool to stage the filled research.md at the scratch path `.spektacular/tmp/research_template.md`, point `plan file write` at it with `--from`, then remove the scratch file:

```
{{config.command}} plan file write {{plan_name}}/research.md --from .spektacular/tmp/research_template.md
rm .spektacular/tmp/research_template.md
```

Never write or edit the plan documents with the `Write` or `Edit` tools — `{{config.command}} plan file write` is the only supported way to write them.

Then advance:

```
{{config.command}} plan goto --data '{"step":"{{next_step}}"}'
```

---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
