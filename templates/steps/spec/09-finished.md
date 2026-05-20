## Step {{step}}: {{title}}

{{#spec_unwritten}}
⚠️ The spec file `{{spec_name}}.md` still holds the empty scaffold — the completed spec was never committed to the store.

Before telling the user the workflow is done, write the spec through Spektacular. Stage the completed spec to a scratch file with the `Write` tool, then:

```
{{config.command}} spec file write {{spec_name}}.md
```

Never edit the spec file with the `Write` or `Edit` tools — `{{config.command}} spec file write` is the only supported way to write it.
{{/spec_unwritten}}
{{^spec_unwritten}}
The spec is complete.

Inform the user that the spec workflow is finished and the spec file is ready to use.
{{/spec_unwritten}}
