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

context.md was assembled in the verification step from your working files (`phases_context.md`, `testing_approach.md`, and your research notes); if the `.spektacular/tmp/context_template.md` scratch file is gone, re-assemble it from `.spektacular/work/{{plan_name}}/`. Now commit context.md the same way. Use the `Write` tool to stage the filled context.md at the scratch path `.spektacular/tmp/context_template.md`, point `plan file write` at it with `--from`, then remove the scratch file:

```
{{config.command}} plan file write {{plan_name}}/context.md --from .spektacular/tmp/context_template.md
rm .spektacular/tmp/context_template.md
```

Never write or edit the plan documents with the `Write` or `Edit` tools — `{{config.command}} plan file write` is the only supported way to write them.

Then advance:

```
{{config.command}} plan goto --data '{"step":"{{next_step}}"}'
```

---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
