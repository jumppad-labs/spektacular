## Step {{step}}: {{title}}

This step is the **validation and drift gate** for the implement workflow. Nothing else runs until it passes. If any check below fails, STOP and report to the user with a three-option prompt — do not silently continue past a failed check.

### Step 1: Full plan read

Read the three plan documents **in full** through the plan store. The plan documents are owned by spektacular — always read them with `{{config.command}} plan file read`, never with the `Read` tool, which bypasses the CLI:

```
{{config.command}} plan file read {{plan_name}}/plan.md
{{config.command}} plan file read {{plan_name}}/context.md
{{config.command}} plan file read {{plan_name}}/research.md
```

These are the source of truth for every downstream step.

### Step 2: Structural validation

Verify `{{plan_path}}` has the complete plan-scaffold shape. Every one of these `## ` sections must be present:

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

Then verify the phase structure:

- At least one `#### - [ ] Phase N.M:` checkbox heading exists under `## Milestones & Phases`.
- Every phase has a `*Technical detail:* [context.md#phase-NM](./context.md#...)` link.
- Every `*Technical detail:*` link target resolves to a matching `### Phase N.M:` heading inside `{{context_path}}`.

If any structural check fails, STOP and report the failures to the user.

### Step 3: Drift check against the working tree

For every **file path**, **package path**, **function name**, **type name**, **command path**, and **template path** named in `{{plan_path}}` or `{{context_path}}` (including inside code blocks and in `file:line` references), verify the target still exists in the current codebase.

**Method**:

- For file/directory paths: use `ls` or attempt to read the file.
- For Go symbols and package import paths: use `grep -rn` or delegate to a codebase-locator sub-agent.
- For CLI commands like `{{config.command}} <something>`: check `cmd/` wiring.
- For template paths like `templates/steps/plan/01-overview.md`: check the file exists.

Collect **every** mismatch into a list — do not fix them silently as you find them.

If the list is non-empty, STOP and report all mismatches to the user in one block. Ask the user to pick one of three options:

1. **Fix the plan first.** Update `{{plan_path}}` and/or `{{context_path}}` to match the current codebase, then restart this step.
2. **Proceed with the mismatches noted in memory.** The agent will adapt during implementation, mapping stale pointers to their current equivalents on the fly.
3. **Abandon the workflow.** Stop the implement run entirely.

Do **not** continue to Step 4 until the user has picked an option.

### Step 4: Changelog mode detection

Check whether a `{{changelog_section_name}}` section already exists inside `{{plan_path}}`.

- **Present** → this is a **subsequent-phase** invocation. Later steps will append new phase entries under the existing section. During `analyze`, pick up at the first unchecked `#### - [ ]` phase in the plan.
- **Absent** → this is a **first-phase** invocation. The `update_changelog` step will create the section on first use. During `analyze`, pick up at the first `#### - [ ]` phase (which will be the first one, unless the user has partially checked off phases manually).

### Advance

Once validation passes, drift is resolved, and changelog mode is known:

```
{{config.command}} implement goto --data '{"step":"{{next_step}}"}'
```
