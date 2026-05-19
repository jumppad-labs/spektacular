## Step {{step}}: {{title}}

Append a phase entry to the inline `{{changelog_section_name}}` section of plan.md, then decide whether to loop back for another phase or advance to the repo-level changelog.

The plan documents are owned by spektacular. **Never read or edit plan.md with the `Read`, `Write`, or `Edit` tools** — go through `{{config.command}} plan file read` and `{{config.command}} plan file write`.

### Step 1: Ensure the `{{changelog_section_name}}` section exists

Read plan.md with `{{config.command}} plan file read {{plan_name}}/plan.md`. If the `{{changelog_section_name}}` heading is absent, this is the first `update_changelog` invocation for this plan — append a new `{{changelog_section_name}}` section **after** the existing `## Out of Scope` section (or at the very end of the file if `## Out of Scope` is missing).

If the `{{changelog_section_name}}` heading is present, append new entries under the existing section, after any entries already there.

### Step 2: Write the phase entry

For the phase you just completed, append an entry with this shape:

```
### <YYYY-MM-DD> — Phase N.M: <phase title>

**What was done**: <1-3 sentences summarizing the code change in plain language>

**Deviations**: <anything that didn't match the plan, or "None" explicitly>

**Files changed**:
- `path/to/file.go`
- `path/to/another/file.go`

**Discoveries**: <anything the next phase or a future maintainer should know — a tricky API, a hidden constraint, a renamed symbol, a missed edge case>
```

For the exact format and more examples, launch a sub-agent with:

```
{{config.command}} skill update-changelog
```

Apply the entry by reading plan.md with `{{config.command}} plan file read {{plan_name}}/plan.md`, adding the entry, staging the updated document with the `Write` tool at the scratch path `.spektacular/tmp/plan_update.md`, then committing it:

```
cat .spektacular/tmp/plan_update.md | {{config.command}} plan file write {{plan_name}}/plan.md
```

### Step 3: Check for remaining unchecked phases

Re-read plan.md with `{{config.command}} plan file read {{plan_name}}/plan.md` and count `#### - [ ] Phase` (unchecked) headings under `## Milestones & Phases`.

**If unchecked phases remain**:

- By default, ask the user whether to continue with the next phase or pause here. Example prompt: "Phase N.M is complete. The next phase is `Phase N.(M+1): <title>`. Continue, or stop here for review?"
- If the user has previously said "run without asking" (or equivalent autonomous mode), skip the prompt and loop automatically.
- To loop, advance to `analyze` — this uses the multi-source FSM transition that lets `analyze` be reached from `update_changelog`:

  ```
  {{config.command}} implement goto --data '{"step":"analyze"}'
  ```

**If no unchecked phases remain**:

- This was the last phase. Advance to `update_repo_changelog` to write the user-facing release-note summary:

  ```
  {{config.command}} implement goto --data '{"step":"update_repo_changelog"}'
  ```

### STOP-on-mismatch

If the plan file's state after Step 2 is inconsistent (e.g. you just wrote a changelog entry for Phase N.M but that phase's checkbox is still unchecked after the previous `update_plan` step), STOP and report the inconsistency to the user.
