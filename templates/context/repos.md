<!-- spektacular:repos:start -->
## Repos

> Managed by `{{config.command}}` — regenerated from the repo registry on every workflow command. Hand edits to this section will not survive; everything below the end marker is yours.

**Where the code lives.** Carry out every code-touching step — research, analysis, implementation, tests, verification — in each repo's source listed here; never assume the directory you started in is a repo's code. Pass the relevant repo's source to any sub-agent you launch.

{{#repos}}
- **{{name}}**{{#description}} — {{description}}{{/description}}{{#role}} (role: {{role}}){{/role}}{{#tags}} [tags: {{tags}}]{{/tags}}{{#deployment}} (deployment: {{deployment}}){{/deployment}}{{#source}} (source: `{{source}}`){{/source}}{{^source}} (code not on disk yet: run `{{config.command}} repo list` for its registered location, then `{{config.command}} repo add` for it to clone its source){{/source}}
{{/repos}}
{{^repos}}
- No repos are registered in this project's configuration; this is a project of one repo. Run `{{config.command}} repo list` for its source.
{{/repos}}
<!-- spektacular:repos:end -->
