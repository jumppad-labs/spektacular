## Step {{step}}: {{title}}

Commit the staged **context.md** to the plan store. Never write or edit the plan documents with the `Write` or `Edit` tools — `{{config.command}} plan file write` is the only supported way to write them.

{{#context_unwritten}}
context.md was assembled and staged at `.spektacular/tmp/context_template.md` in the assemble step. Commit it now, then remove the scratch file:

```
{{config.command}} plan file write {{plan_name}}/context.md --from .spektacular/tmp/context_template.md
rm .spektacular/tmp/context_template.md
```

If the `.spektacular/tmp/context_template.md` scratch file is gone (that path is git-ignored and does not survive a crash), re-assemble it from the per-section working files under `.spektacular/work/{{plan_name}}/` before committing — they are the durable source.
{{/context_unwritten}}
{{^context_unwritten}}
context.md has already been committed to the plan store. If `.spektacular/tmp/context_template.md` is still present, remove it.
{{/context_unwritten}}

Then advance:

```
{{config.command}} plan goto --data '{"step":"{{next_step}}"}'
```

---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
