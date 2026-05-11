## Step {{step}}: {{title}}

Time to fill the three plan documents. Spektacular will write them to disk — you produce the content and pipe it back.

### Step 1: Gather Metadata

Use the `gather-project-metadata` skill to collect: ISO timestamp, git commit, branch, and repository info.
For skill details: `{{config.command}} skill gather-project-metadata`

### Step 2: Determine Feature Slug

Use the `determine-feature-slug` skill to determine the plan file namespace and number.
For skill details: `{{config.command}} skill determine-feature-slug`

### Step 3: Fill in the Three Scaffolds

Fill in ALL sections of all three scaffolds — no placeholders, no open questions.

#### plan.md scaffold

```markdown
{{plan_template}}
```

#### context.md scaffold

```markdown
{{context_template}}
```

#### research.md scaffold

```markdown
{{research_template}}
```

### Step 4: Review

Before piping any document, verify **every required section is present**. A common failure mode is silently dropping a section when assembling the final doc. Check each document against the section list below and confirm the heading is present AND filled with real content (not empty, not a placeholder).

**plan.md — required `##` sections** (in order):

1. `## Overview`
2. `## Architecture & Design Decisions`
3. `## Component Breakdown`
4. `## Data Structures & Interfaces`
5. `## Implementation Detail`
6. `## Dependencies`
7. `## Testing Approach`
8. `## Milestones & Phases`
9. `## Open Questions`
10. `## Out of Scope`

**context.md — required `##` sections** (in order):

1. `## Current State Analysis`
2. `## Per-Phase Technical Notes`
3. `## Testing Strategy`
4. `## Project References`
5. `## Token Management Strategy`
6. `## Migration Notes`
7. `## Performance Considerations`

**research.md — required `##` sections** (in order):

1. `## Alternatives considered and rejected`
2. `## Chosen approach — evidence`
3. `## Files examined`
4. `## External references`
5. `## Prior plans / specs consulted`
6. `## Open assumptions`
7. `## Rehydration cues`

Then verify quality:

- **plan.md** — readable in under a minute; every phase has a summary paragraph, a `*Technical detail:*` link, and outcome-based acceptance criteria; no shell commands anywhere.
- **context.md** — per-phase technical notes under headings matching plan.md's `*Technical detail:*` anchors.
- **research.md** — alternatives considered and rejected with citations. Dense enough to rehydrate a cold session.

If any section is missing from any document, add it and re-review before proceeding. Do **not** advance until every section in every list above is present.

### Step 5: Submit plan.md

Do **not** write directly to `{{plan_path}}` — spektacular owns that file. Instead, use the `Write` tool to stage the filled plan.md at `.spektacular/tmp/plan_template.md`, then submit it via `--file`:

```
{{config.command}} plan goto --data '{"step":"{{next_step}}"}' --file .spektacular/tmp/plan_template.md
```

Spektacular reads the file and stores its contents under the workflow key derived from the filename (`plan_template`), then writes the final plan to `{{plan_path}}`. The `--file` flag is required here (not `--stdin`) because large plans exceed the tool-call size limit when inlined as a heredoc.

If `{{config.command}} notion status` reports Notion mode as configured, update the matching Notion artifact through MCP before each write step and include the returned metadata in the `plan goto` `--data` payload: `remote` for plan.md, `context_remote` for context.md, and `research_remote` for research.md. If `notion cache prepare-push` returns `merge_required`, stop and resolve the merge before advancing.
