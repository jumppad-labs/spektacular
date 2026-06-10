## Step {{step}}: {{title}}

The implement workflow is complete for `{{plan_name}}`.

### Summary

- All phases in `{{plan_path}}` under `## Milestones & Phases` have been checked off.
- Per-phase implementation entries have been appended to the inline `{{changelog_section_name}}` section of `{{plan_path}}`.
- A user-facing release note has been prepended to the repo-level `CHANGELOG.md` under the `## {{plan_name}}` heading.

### What to do next

Report to the user:

- The phases that were completed (read the `#### - [x] Phase` headings from `{{plan_path}}`).
- Any deviations from the plan that were recorded in the inline changelog.
- The location of the repo `CHANGELOG.md` entry so the user can review or edit before releasing.

This is the terminal state of the implement workflow. Do **not** emit a `goto` command — no further steps exist.
