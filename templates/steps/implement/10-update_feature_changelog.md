## Step {{step}}: {{title}}

Produce a durable, self-contained changelog record for this feature in the changelog store, grounded in what was actually built. This step runs last, after every phase is implemented and verified and the test plan is written, so the record reflects the real outcome — including anything that turned out differently than planned — not the original intent.

### Step 1: Gather source material

Read the feature's spec and the plan's accumulated implementation history through their respective stores — never with the `Read` tool:

```
{{config.command}} spec file read {{plan_name}}.md
{{config.command}} plan file read {{plan_name}}/plan.md
```

From the spec, take the Overview/Requirements — this is the "why it matters" framing for a reader who has not seen the spec. From the plan, take the `{{changelog_section_name}}` section — the phase-by-phase implementation audit log already written by the `update_changelog` step, including each phase's **What was done**, **Deviations**, **Files changed**, and **Discoveries** entries. This is the only place the actual-vs-planned divergence is already captured, so lean on it rather than re-deriving anything from the original plan text.

### Step 2: Author the record

Write a self-contained Markdown record — understandable without the spec, plan, or this conversation open alongside it — covering:

- **What was built** — a plain-language description of the feature, grounded in the plan's `{{changelog_section_name}}` entries, not the original plan's stated intent.
- **Why it matters / what it enables** — drawn from the spec's framing.
- **Deviations from the plan** — anything the phase entries recorded as different from what was originally planned. State "None" explicitly if there were none.

### Step 3: Write the artifact to the changelog store

The changelog record is a changelog-store artifact. **Never write it with the `Write`/`Edit` tools directly into the changelog directory** — stage it, then commit it through the CLI:

1. Use the `Write` tool to stage the content at the scratch path `.spektacular/tmp/changelog_record.md`.
2. Commit it and remove the scratch file:

   ```
   {{config.command}} changelog file write {{plan_name}}.md --from .spektacular/tmp/changelog_record.md
   rm .spektacular/tmp/changelog_record.md
   ```

After the write succeeds, confirm it with `{{config.command}} changelog file read {{plan_name}}.md`.

### Step 4: Derive one entry per affected repo

Work that touched registered member repos leaves a record in every repo it touched. After the central record is written:

1. Run `{{config.command}} repo list` to see the registered repos and their resolved roots.
2. Identify the **affected repos**: from the plan's `{{changelog_section_name}}` section, collect every path in the phase entries' **Files changed** lists and match each to the repo it belongs to (entries there are grouped or prefixed by repo name; paths with no repo prefix belong to the project's own colocated repo). A repo with no changed files is not affected and gets no entry.
3. For each affected repo **other than the project's own colocated repo** (the central record already covers it), author a derived entry scoped to **only that repo's changes**: what was changed in that repo and why, grounded in the same phase entries, plus a human-readable reference line at the top of the body naming the project, its source (when configured), and the spec/plan identifier — e.g. `> Derived from project <project> (<source>), spec/plan {{plan_name}}.`
4. Stage each entry with the `Write` tool at `.spektacular/tmp/changelog_derived_<repo>.md`, then write it through that repo's own changelog store and remove the scratch file:

   ```
   {{config.command}} changelog file write {{plan_name}}.md --repo <repo-name> --from .spektacular/tmp/changelog_derived_<repo>.md
   rm .spektacular/tmp/changelog_derived_<repo>.md
   ```

   The CLI stamps the project name, project source, and spec/plan identifiers into the entry's front matter automatically — the body carries the readable reference line, the front matter carries the structured provenance.

If no registered repo other than the colocated one was touched, skip this step entirely — do not write empty derived entries.

### STOP-on-mismatch

If the feature's spec cannot be found under `{{plan_name}}.md`, or the plan's `{{changelog_section_name}}` section is missing or empty, STOP and report it to the user: ask whether to (a) locate the correct spec name and retry, (b) author the record from the plan alone, or (c) skip this step. Do not silently invent a record from nothing. Likewise, if a derived write fails because a repo's footprint is missing or broken, surface the repair offer from the error to the user rather than skipping the repo silently.

### Advance

Once the changelog record is written and confirmed:

```
{{config.command}} implement goto --data '{"step":"{{next_step}}"}'
```

---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
