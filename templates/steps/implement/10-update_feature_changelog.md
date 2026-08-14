## Step {{step}}: {{title}}

Produce durable changelog records for this feature, grounded in what was actually built. This step runs last, after every phase is implemented and verified and the test plan is written, so the records reflect the real outcome — including anything that turned out differently than planned — not the original intent.

Two kinds of record are written every time:

- One **project-level record** containing every change made across every affected repo. This is the single artifact readers reach for when they want to understand the feature as a whole.
- One **repo-level record per affected repo** containing only the changes that touched that repo. These live alongside each repo's own history and stand alone — a reader viewing only that repo can understand what changed and why.

### Step 1: Gather source material

Read the feature's spec and the plan's accumulated implementation history through their respective stores — never with the `Read` tool:

```
{{config.command}} spec file read {{plan_name}}.md
{{config.command}} plan file read {{plan_name}}/plan.md
```

From the spec, take the Overview/Requirements — the "why it matters" framing for a reader who has not seen the spec. From the plan, take the `{{changelog_section_name}}` section — the phase-by-phase implementation audit log already written by the `update_changelog` step, including each phase's **What was done**, **Deviations**, **Files changed**, and **Discoveries** entries. This is where the actual-vs-planned divergence is captured; lean on it rather than re-deriving anything from the original plan text.

### Step 2: Identify affected repos

Discover the registered repos:

```
{{config.command}} repo list
```

From the plan's `{{changelog_section_name}}` section, collect every path in the phase entries' **Files changed** lists and match each to a repo:

- Paths carrying a `<repo-name>: ` prefix belong to that named repo.
- Paths with no prefix belong to the project's colocated repo (the one that shares this project's working tree).

A repo with no changed files is not affected and gets no repo-level record. **Every** affected repo — including the colocated one — gets its own repo-level record; there is no "the project-level already covers it" carve-out.

### Step 3: Write the project-level record

Author a self-contained Markdown record covering the entire feature, understandable without the spec, plan, or this conversation open alongside it:

- **What was built** — a plain-language description grounded in the plan's `{{changelog_section_name}}` entries across all affected repos.
- **Why it matters / what it enables** — drawn from the spec's framing.
- **Deviations from the plan** — anything the phase entries recorded as different from what was originally planned. State "None" explicitly if there were none.

Stage it with the `Write` tool at `.spektacular/tmp/changelog_project.md`, then commit it and remove the scratch file:

```
{{config.command}} changelog file write {{plan_name}}.md --from .spektacular/tmp/changelog_project.md
rm .spektacular/tmp/changelog_project.md
```

Confirm the write with `{{config.command}} changelog file read {{plan_name}}.md`. This record is a required artifact — the workflow's `finished` step will error if it is missing.

### Step 4: Write one record per affected repo

For **each** affected repo identified in Step 2, including the colocated repo, author a repo-scoped record covering only that repo's changes:

- **What changed in this repo** — grounded in the phase entries' Files-changed and What-was-done items that touched this repo. Include only that repo's changes.
- **Why** — the same framing from the spec, tightened to why *this repo* got the change.
- A human-readable reference line at the top of the body naming the project and the spec/plan identifier — e.g. `> Derived from project <project> (<source>), spec/plan {{plan_name}}. See the project-level record for the full feature.`

Stage each record with the `Write` tool at `.spektacular/tmp/changelog_<repo>.md`, then commit it and remove the scratch file:

```
{{config.command}} changelog file write {{plan_name}}.md --repo <repo-name> --from .spektacular/tmp/changelog_<repo>.md
rm .spektacular/tmp/changelog_<repo>.md
```

The `--repo` flag routes the write into that repo's own changelog store, under a `<project>/` subfolder (so multiple projects sharing a member repo cannot collide), and stamps the project name, source, and spec/plan identifiers into the front matter automatically — the body carries the readable reference line, the front matter carries the structured provenance.

Confirm each write with `{{config.command}} changelog file read {{plan_name}}.md --repo <repo-name>`.

### STOP-on-mismatch

If the feature's spec cannot be found under `{{plan_name}}.md`, or the plan's `{{changelog_section_name}}` section is missing or empty, STOP and report it to the user: ask whether to (a) locate the correct spec name and retry, (b) author the records from the plan alone, or (c) skip this step. Do not silently invent a record from nothing. Likewise, if a repo-level write fails because a repo's footprint is missing or broken, surface the repair offer from the error to the user rather than skipping the repo silently.

### Advance

Once the project-level record and every affected repo's record are written and confirmed:

```
{{config.command}} implement goto --data '{"step":"{{next_step}}"}'
```

---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
