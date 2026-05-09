# Feature: spec-id-prefix-method

<!--
  OVERVIEW
  A concise 2-3 sentence summary of the feature. Answer three questions:
    1. What is being built?
    2. What problem does it solve?
    3. Who benefits and why does it matter?
  Avoid implementation details — this should be readable by any stakeholder.
-->
## Overview

Specification creation should generate stable, migration-style identifiers by default so new specs sort chronologically and avoid ambiguous naming collisions. Teams can choose timestamp-based or counter-based generated prefixes, while external systems can supply their own normalized identifier when they create a spec, making automated integrations predictable without removing manual flexibility.


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

- [ ] **Timestamp prefixes by default**
      The system must generate timestamp-style prefixes for new specifications when no other identifier method is configured or requested.
- [ ] **Counter prefixes available**
      Users can choose counter-style generated prefixes for new specifications.
- [ ] **External identifiers accepted**
      Users and external systems can pass an explicit identifier during specification creation, and the system must use the normalized value instead of generating one.
- [ ] **Input normalization and validation**
      The system must normalize accepted spec names and identifiers into valid workflow names without silently trimming leading or trailing whitespace.
- [ ] **Configurable identifier method**
      Users can set a default identifier method in configuration so future specification creation uses that method unless an explicit identifier is provided.
- [ ] **Explicit identifiers override method defaults**
      The system must accept a passed identifier regardless of the configured identifier method.
- [ ] **Required identifiers for external mode**
      When configuration selects the externally supplied identifier method, specification creation must require an explicit identifier value.
- [ ] **Native creation behavior**
      Identifier selection and validation must be available through the standard specification creation workflow without requiring a separate wrapper or integration script.


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

- Existing specification creation requests that pass only a name must remain valid and must not require a config change.
- Existing project config files without an identifier method field must remain valid.
- Identifier values must not be able to create files outside the specification directory.
- Generated and externally supplied prefixes must produce valid specification filenames on supported filesystems.


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

- [ ] **Default timestamp prefix is used**
      When a user creates a specification with name `billing-export` and no identifier method or identifier value is configured or passed, the created specification filename starts with a sortable timestamp prefix followed by `billing-export`.
- [ ] **Timestamp prefixes sort chronologically**
      When two specifications are created at different times with the default method, sorting their filenames alphabetically orders them from older to newer.
- [ ] **Counter prefix can be selected**
      When a user selects the counter method and creates a specification named `billing-export`, the created specification filename starts with the next available counter prefix followed by `billing-export`.
- [ ] **Counter prefixes increment**
      When two specifications are created with the counter method in the same project, the second filename uses the next stored counter value after the first.
- [ ] **Passed identifier is normalized as the prefix**
      When a user creates a specification named `billing-export` and passes identifier `EXT-123`, the created specification filename starts with `ext-123` followed by `billing-export`.
- [ ] **Passed identifier works with any configured generated method**
      When the project is configured for timestamp or counter generation and a user passes identifier `EXT-123`, the created filename uses `ext-123` rather than a generated timestamp or counter.
- [ ] **Special characters are normalized**
      When a user creates a specification with identifier `Team.Alpha@Roadmap`, the created filename starts with `team-alpha-roadmap` followed by the requested name.
- [ ] **Whitespace is not silently trimmed**
      When a user passes a spec name or identifier with leading or trailing whitespace, creation fails with a clear error and no specification file is created.
- [ ] **Configured method becomes the default**
      When a project config sets the identifier method to counter and the user creates a specification without passing an identifier value, the created filename uses a counter prefix.
- [ ] **External method requires an identifier**
      When a project config selects the externally supplied identifier method and a user creates a specification without passing an identifier value, creation fails with a clear error and no specification file is created.
- [ ] **External method succeeds with an identifier**
      When a project config selects the externally supplied identifier method and a user creates a specification with identifier `EXT-123`, creation succeeds and the created filename starts with `ext-123` followed by the requested name.
- [ ] **Timestamp collisions advance by seconds**
      When the timestamp method would create a filename that already exists for the requested name, creation uses the next available second-level timestamp prefix instead of overwriting the existing file.


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

- Implement identifier generation inside the `spektacular spec new` path so installed skills and command wrappers inherit the behavior automatically.
- Extend spec creation input to accept an optional `id` value. When present, normalize and validate it for filename safety, then use it as the prefix regardless of configured generation method.
- Add input normalization for spec names and identifiers. Normalization lowercases accepted input and replaces special separators such as `.`, `@`, and internal whitespace with `-`; leading or trailing whitespace must fail rather than being trimmed.
- Add optional config fields for the spec identifier method and current counter value. Supported methods should be `timestamp`, `counter`, and an externally supplied identifier mode. If the method field is omitted, use `timestamp`.
- Treat the externally supplied identifier mode as a contract that creation requests must provide `id`; generated modes may still accept `id` as an override.
- Prefer a timestamp format that sorts lexicographically in creation order, such as `YYYYMMDDHHMMSS`, similar to database migration filenames. If the computed timestamp target already exists, increment the timestamp by one second until an unused target is found.
- For counter mode, persist the current counter value in project config and increment it for each counter-created spec. Use a stable, zero-padded numeric prefix so alphabetical order matches numeric order.
- Keep the human-readable requested name as the suffix after the generated or supplied prefix.
- Update command schemas, config loading/defaults, init output, and tests so the new inputs and config field are visible to agents and external callers.


<!--
  SUCCESS METRICS
  How you will know the feature is working well after delivery. Be specific:
    - Quantitative: "p99 latency < 200ms", "error rate < 0.1%"
    - Behavioural: "users complete the flow without support intervention"
  Leave blank if not applicable.
-->
## Success Metrics

- New specifications created without custom settings are alphabetically sortable by creation time.
- Projects can switch between timestamp and counter prefixes through configuration without changing installed agent skills.
- External systems can create specifications whose filenames carry the normalized external identifier, with creation failing clearly when external-id mode is configured but no identifier is supplied.

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

- Renaming or migrating existing specification files is out of scope.
- Building a direct integration with any specific external system is out of scope.
- Changing the contents or step order of the specification authoring workflow is out of scope.
- Applying the same identifier method to plans or implementation artifacts is out of scope unless a later spec requests it.
