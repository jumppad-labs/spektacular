---
created_date: "2026-08-20"
status: completed
closed_date: "2026-08-20"
---

# Feature: 000045_config-file-migration

<!--
  OVERVIEW
  A concise 2-3 sentence summary of the feature. Answer three questions:
    1. What is being built?
    2. What problem does it solve?
    3. Who benefits and why does it matter?
  Avoid implementation details — this should be readable by any stakeholder.
-->
## Overview

Spektacular will automatically detect and migrate old-format configuration files when users upgrade to versions with multi-repo project support. When the version check detects a legacy single-file config.yaml, it prompts the user to migrate to the new split format (project settings in config.yaml, repository metadata in repo.yaml), scanning the project to suggest appropriate metadata values. This ensures existing Spektacular users can upgrade smoothly without manual configuration changes or errors.

<!--
  REQUIREMENTS
  Specific, testable behaviours the feature must deliver.
  Format: bold title on the checkbox line, detail indented below.
  Rules:
    - Use active voice: "Users can...", "The system must..."
    - Each requirement should be independently verifiable
    - Focus on WHAT, not HOW — avoid prescribing implementation
    - Keep each item atomic — one behaviour per line
-->
## Requirements

- [x] **Old Config Detection**
  The system must detect when a legacy single-file config.yaml exists without a corresponding repo.yaml file during version check.

- [x] **Migration Prompt**
  Users must be prompted to migrate their configuration when an old format is detected, with clear explanation of what will change.

- [x] **Automatic Metadata Generation**
  The system must scan the project to suggest appropriate values for repository metadata (description, role, tags, deployment method).

- [x] **Non-Destructive Migration**
  The system must preserve the original config.yaml as a backup before making any changes.

- [x] **Immediate Migration**
  Users can confirm the migration prompt to immediately execute the migration without requiring a preview step.

- [x] **Graceful Handling of Existing Files**
  The system must skip migration if repo.yaml already exists, treating it as already migrated.

- [x] **Fallback Defaults**
  The system must provide sensible default values for repository metadata when project scanning cannot determine appropriate values.

- [x] **Rollback on Failure**
  The system must restore the original configuration if migration fails partway through.

<!--
  CONSTRAINTS
  Hard boundaries the solution must operate within. These are non-negotiable.
  Format: one bullet point per constraint.
  Examples:
    - Must integrate with the existing authentication system
    - Cannot introduce breaking changes to the public API
    - Must support the current minimum supported runtime versions
  Leave blank if there are no constraints.
-->
## Constraints

- Must integrate with the existing `version check` command without requiring a separate migration command
- Must not break existing workflows or commands during or after migration

<!--
  ACCEPTANCE CRITERIA
  The specific, binary conditions that define "done".
  Format: bold title on the checkbox line, verifiable detail indented below.
  Each criterion must be:
    - Independently verifiable (pass/fail, not subjective)
    - Traceable back to a requirement above
    - Testable by someone who didn't write the code
-->
## Acceptance Criteria

- [x] **Detection Triggers Prompt**
  When version check runs with an old-format config.yaml and no repo.yaml, the output includes a migration prompt explaining the config split.

- [x] **Prompt Explains Changes**
  The migration prompt clearly states that config.yaml will be backed up and repo.yaml will be created with scanned metadata.

- [x] **Scanned Metadata Appears**
  After successful migration, repo.yaml exists and contains description, role, tags, and deployment fields populated from project scanning.

- [x] **Backup File Created**
  After successful migration, config.yaml.old exists and contains the exact content of the original config.yaml.

- [x] **Migration Completes on Confirmation**
  When the user confirms the migration prompt, both repo.yaml and config.yaml.old are created without requiring additional steps.

- [x] **Already-Migrated Skip**
  When version check runs with both config.yaml and repo.yaml present, no migration prompt appears.

- [x] **Defaults When Scan Fails**
  When project scanning cannot determine metadata values, repo.yaml is created with default values (description: "Spektacular project", role: "application", tags: [], deployment: "manual").

- [x] **Rollback on Error**
  When migration fails partway through, config.yaml matches its pre-migration state and no repo.yaml or config.yaml.old files exist.

<!--
  TECHNICAL APPROACH
  High-level technical direction to guide the planning agent. Include:
    - Key architectural decisions already made
    - Preferred patterns or technologies if known
    - Integration points with existing systems
    - Known risks or areas of uncertainty
  Format: one bullet point per direction/steer.
  Leave blank if you want the planner to propose the approach.
-->
## Technical Approach

- Scan project files (README.md, build files like go.mod or package.json, directory structure) to suggest appropriate values for repository metadata fields
- Provide sensible defaults (description: "Spektacular project", role: "application", tags: [], deployment: "manual") when project scanning cannot determine appropriate values

<!--
  SUCCESS METRICS
  How you will know the feature is working well after delivery. Be specific:
    - Quantitative: "p99 latency < 200ms", "error rate < 0.1%"
    - Behavioural: "users complete the flow without support intervention"
  Format: one bullet point per metric.
  Leave blank if not applicable.
-->
## Success Metrics

No specific success metrics have been defined for this feature. The migration is considered successful when it completes without errors and produces the expected config.yaml.old and repo.yaml files.

<!--
  NON-GOALS
  Explicitly state what this spec does NOT cover. This is as important as
  the requirements — it prevents scope creep and sets clear expectations.
  Format: one bullet point per exclusion.
  Examples:
    - "Mobile support is out of scope (tracked in #456)"
    - "Internationalisation will be addressed in a follow-up spec"
  Leave blank if there are no explicit exclusions to call out.
-->
## Non-Goals

- Migrating registered repositories (e.g., the docs repo) - only the current repository receives a repo.yaml file
- Interactive editing of suggested metadata during migration - users can manually edit repo.yaml after migration completes
- Migrating other configuration formats or file types beyond the config.yaml → config.yaml + repo.yaml split
- Providing a preview of changes before migration - migration executes immediately after user confirmation
