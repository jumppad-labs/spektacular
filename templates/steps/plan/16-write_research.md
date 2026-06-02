## Step {{step}}: {{title}}

{{#research_unwritten}}
⚠️ research.md was not found in the plan store, or still holds the empty scaffold — it was never committed. Commit the filled research.md before continuing, then remove the scratch file:

```
{{config.command}} plan file write {{plan_name}}/research.md --from .spektacular/tmp/research_template.md
rm .spektacular/tmp/research_template.md
```
{{/research_unwritten}}
{{^research_unwritten}}
research.md has been committed to the plan store.

All three plan documents are in the plan store. The per-section working files have served their purpose — remove the working directory with your own file tools:

```
rm -rf .spektacular/work/{{plan_name}}
```
{{/research_unwritten}}

Advance to the finished step:

```
{{config.command}} plan goto --data '{"step":"{{next_step}}"}'
```

---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
