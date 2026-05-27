## Step {{step}}: {{title}}

{{#spec_unwritten}}
⚠️ The spec file `{{spec_name}}.md` still holds the empty scaffold — the completed spec was never committed to the store.

Before telling the user the workflow is done, write the spec through Spektacular. Stage the completed spec to `.spektacular/tmp/spec_template.md` with the `Write` tool, point `spec file write` at it with `--from`, then remove the scratch file:

```
{{config.command}} spec file write {{spec_name}}.md --from .spektacular/tmp/spec_template.md
rm .spektacular/tmp/spec_template.md
```

Never edit the spec file with the `Write` or `Edit` tools — `{{config.command}} spec file write` is the only supported way to write it.
{{/spec_unwritten}}
{{^spec_unwritten}}
The spec is complete.

Inform the user that the spec workflow is finished and the spec file is ready to use.
{{/spec_unwritten}}
