## Step {{step}}: {{title}}

The three plan documents were assembled and staged in the previous step at:

- `.spektacular/tmp/plan_template.md`
- `.spektacular/tmp/context_template.md`
- `.spektacular/tmp/research_template.md`

Verify them here. This step **only checks correctness** — it writes nothing to the plan store (the write steps that follow do that). If a scratch file is missing (the `.spektacular/tmp/` path is git-ignored and does not survive a crash), re-assemble it from the per-section working files under `.spektacular/work/{{plan_name}}/` before verifying.

### Step 1: Every required section is present and filled

A common failure mode is silently dropping a section when assembling. Check each staged document against the section list below and confirm the heading is present AND filled with real content (not empty, not a placeholder).

**plan.md — required `##` sections** (in order):

1. `## Overview`
2. `## Conventions`
3. `## Architecture & Design Decisions`
4. `## Component Breakdown`
5. `## Data Structures & Interfaces`
6. `## Implementation Detail`
7. `## Dependencies`
8. `## Testing Approach`
9. `## Milestones & Phases`
10. `## Open Questions`
11. `## Out of Scope`

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

### Step 2: Quality

- **plan.md** — readable in under a minute; every phase has a summary paragraph, a `*Technical detail:*` link, and outcome-based acceptance criteria; no shell commands anywhere.
- **context.md** — per-phase technical notes under headings matching plan.md's `*Technical detail:*` anchors.
- **research.md** — alternatives considered and rejected with citations. Dense enough to rehydrate a cold session.

### Step 3: Fix and re-stage

If any section is missing, empty, or fails a quality check, fix it in the **owning working file** under `.spektacular/work/{{plan_name}}/` (the working files are the durable source), then re-assemble the affected scratch file from the working files so the staged document reflects the change. Re-run this verification until every section in every list above is present and filled. Do **not** advance until the staged documents are correct.

This step does not touch the plan store. Never write or edit the plan documents with the `Write` or `Edit` tools — the write steps that follow commit them through `{{config.command}} plan file write`, which is the only supported way.

Then advance:

```
{{config.command}} plan goto --data '{"step":"{{next_step}}"}'
```

---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
