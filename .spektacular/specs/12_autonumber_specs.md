# Feature: 12 Autonumber Specs

<!--
  OVERVIEW
  A concise 2-3 sentence summary of the feature. Answer three questions:
    1. What is being built?
    2. What problem does it solve?
    3. Who benefits and why does it matter?
  Avoid implementation details — this should be readable by any stakeholder.
-->
## Overview



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

- [ ] **Auto-assign spec number** — The system must automatically determine the next spec number by incrementing from the highest existing spec number in the specs directory.
- [ ] **Name-only input** — Users can create a new spec by providing only the spec name, without specifying a number.
- [ ] **Numbered filename generation** — The system must generate the spec filename using the auto-assigned number and the user-provided name (e.g., `13_my_feature.md`).

<!--
  CONSTRAINTS
  Hard boundaries the solution must operate within. These are non-negotiable.
  Examples:
    - Must integrate with the existing authentication system
    - Cannot introduce breaking changes to the public API
    - Must support the current minimum supported runtime versions
  Leave blank if there are no constraints.
-->
## Constraints

- Must allow user-provided spec IDs in any format (e.g., alphanumeric, random GUID) — auto-numbering only applies when no ID is given.

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

- [ ] **Auto-assign when no number provided** — When no spec number is provided, the system automatically assigns a number that is one greater than the highest existing spec number in the specs directory.
- [ ] **Skip auto-assign when number provided** — When the user explicitly provides a spec number, no number is auto-generated and the provided number is used as-is.
- [ ] **Name-only creates numbered spec** — When only the spec name is provided (no number), the resulting spec file is created with an auto-incremented number prefixed to the name.
- [ ] **Explicit number bypasses auto-numbering** — When both a name and number are provided, the system takes no auto-numbering action and uses the given number.
- [ ] **Filename convention** — The system generates all spec filenames following the convention `<number>_<spec_name>.md` (e.g., `13_my_feature.md`).

<!--
  TECHNICAL APPROACH
  High-level technical direction to guide the planning agent. Include:
    - Key architectural decisions already made
    - Preferred patterns or technologies if known
    - Integration points with existing systems
    - Known risks or areas of uncertainty
  Leave blank if you want the planner to propose the approach.
-->
## Technical Approach

None.

<!--
  SUCCESS METRICS
  How you will know the feature is working well after delivery. Be specific:
    - Quantitative: "p99 latency < 200ms", "error rate < 0.1%"
    - Behavioural: "users complete the flow without support intervention"
  Leave blank if not applicable.
-->
## Success Metrics

None.

<!--
  NON-GOALS
  Explicitly state what this spec does NOT cover. This is as important as
  the requirements — it prevents scope creep and sets clear expectations.
  Examples:
    - "Mobile support is out of scope (tracked in #456)"
    - "Internationalisation will be addressed in a follow-up spec"
  Leave blank if there are no explicit exclusions to call out.
-->
## Non-Goals

None.
