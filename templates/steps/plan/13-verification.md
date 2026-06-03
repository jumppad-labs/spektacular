## Step {{step}}: {{title}}

Time to fill the three plan documents. Spektacular will write them to disk — you produce the content and pipe it back.

### Step 1: Gather Metadata

Use the `gather-project-metadata` skill to collect: ISO timestamp, git commit, branch, and repository info.
For skill details: `{{config.command}} skill gather-project-metadata`

### Step 2: Determine Feature Slug

Use the `determine-feature-slug` skill to determine the plan file namespace and number.
For skill details: `{{config.command}} skill determine-feature-slug`

### Step 3: Fill in the Three Scaffolds

Assemble all three scaffolds from the per-section working files you wrote during the earlier steps — no placeholders, no open questions. The working files hold body content only; each scaffold below owns its headings and their order. Drop each working file's content under the matching heading.

**plan.md** ← files under `.spektacular/work/{{plan_name}}/`:
- `conventions.md` → `## Conventions`
- `architecture.md` → `## Architecture & Design Decisions`
- `components.md` → `## Component Breakdown`
- `data_structures.md` → `## Data Structures & Interfaces`
- `implementation_detail.md` → `## Implementation Detail`
- `dependencies.md` → `## Dependencies`
- `testing_approach.md` → `## Testing Approach`
- `milestones.md` + `phases_plan.md` → `## Milestones & Phases`
- `open_questions.md` → `## Open Questions`
- `out_of_scope.md` → `## Out of Scope`
- `## Overview` ← derive from the spec you read in step 01 and your `.spektacular/context.md` notes.

**context.md** ← files under `.spektacular/work/{{plan_name}}/`:
- `phases_context.md` → `## Per-Phase Technical Notes`
- `testing_approach.md` → `## Testing Strategy` (recast at per-phase granularity)
- `## Current State Analysis`, `## Project References`, `## Token Management Strategy`, `## Migration Notes`, `## Performance Considerations` ← your research findings in `research.md` and `.spektacular/context.md`.

**research.md** ← `.spektacular/work/{{plan_name}}/research.md` (it maps directly onto research.md's seven sections).

If a required working file is missing, the matching gathering step was not completed — STOP and complete it before assembling.

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

Then verify quality:

- **plan.md** — readable in under a minute; every phase has a summary paragraph, a `*Technical detail:*` link, and outcome-based acceptance criteria; no shell commands anywhere.
- **context.md** — per-phase technical notes under headings matching plan.md's `*Technical detail:*` anchors.
- **research.md** — alternatives considered and rejected with citations. Dense enough to rehydrate a cold session.

If any section is missing from any document, add it and re-review before proceeding. Do **not** advance until every section in every list above is present. If a section came out empty, check its working file under `.spektacular/work/{{plan_name}}/` — the gathering step may have been skipped. In particular, confirm both `phases_plan.md` and `phases_context.md` exist, since the phases step writes two files that feed plan.md and context.md respectively.

### Step 5: Commit plan.md

The plan documents are owned by spektacular. **Never write or edit them with the `Write` or `Edit` tools** — `{{config.command}} plan file write` is the only supported way to write them. It routes the write through the CLI into the configured plan directory.

Stage each document through a scratch file. Use the `Write` tool to write the filled plan.md to the scratch path `.spektacular/tmp/plan_template.md`, point `plan file write` at it with `--from`, then remove the scratch file:

```
{{config.command}} plan file write {{plan_name}}/plan.md --from .spektacular/tmp/plan_template.md
rm .spektacular/tmp/plan_template.md
```

The path argument is the plan-directory-relative document path — `plan file write` resolves it against the configured plan directory for you.

Then advance:

```
{{config.command}} plan goto --data '{"step":"{{next_step}}"}'
```

---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
