## Step {{step}}: {{title}}

The implement workflow is complete for `{{plan_name}}`.

### Summary

- All phases in `{{plan_path}}` under `## Milestones & Phases` have been checked off.
- Per-phase implementation entries have been appended to the inline `{{changelog_section_name}}` section of `{{plan_path}}`.
- A user-facing release note has been prepended to the repo-level `CHANGELOG.md` under the `## {{plan_name}}` heading.
- A changelog record for this feature has been written to `{{changelog_path}}`.
- The specification's Requirements and Acceptance Criteria have been reconciled against the completed work; see `{{spec_path}}` for the updated checkbox state.

### What to do next

Report to the user:

- The phases that were completed (read the `#### - [x] Phase` headings from `{{plan_path}}`).
- Any deviations from the plan that were recorded in the inline changelog.
- The location of the repo `CHANGELOG.md` entry so the user can review or edit before releasing.
- The location of the changelog record at `{{changelog_path}}`.
- The specification's completion status: read `{{spec_path}}` via `{{config.command}} spec file read {{plan_name}}.md`, and tell the user which Requirements/Acceptance-Criteria items are now checked, and for any still unchecked, the reason recorded during `reconcile_spec` (deferred, descoped, or not attempted).

This is the terminal state of the implement workflow. Do **not** emit a `goto` command — no further steps exist.
