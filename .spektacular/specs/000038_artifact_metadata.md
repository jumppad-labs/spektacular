---
created_date: "2026-07-29"
status: in-progress
---

# Feature: 000038_artifact_metadata

<!--
  OVERVIEW
  A concise 2-3 sentence summary of the feature. Answer three questions:
    1. What is being built?
    2. What problem does it solve?
    3. Who benefits and why does it matter?
  Avoid implementation details — this should be readable by any stakeholder.
-->
## Overview

Attach metadata to each workflow document that Spektacular writes — the specifications, plans, per-change changelog entries, and test plans — so that every artifact carries its own creation date, current status (in-progress, completed, superseded, archived), and closing date if it has one. This makes it possible to find documents by their lifecycle status and age, and gives anyone reading a historical document a clear signal of how old the information is. Both users triaging accumulated workflow output and agents reaching for a historical document during archaeology benefit from having this signal on the artifact itself, rather than having to infer it from indirect signals.

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

- [x] **Every workflow document Spektacular writes has associated metadata**
  Each specification, plan document (including every sibling document within a plan directory, whatever those are today), changelog entry, and test plan that Spektacular writes as part of any workflow has its own associated metadata, independent of any other document's metadata.

- [x] **Metadata records when the document was created**
  The metadata on every document identifies the date the document was first written by its workflow.

- [x] **Metadata records the document's current status**
  The metadata on every document identifies its current status as one of: in-progress (workflow still open), completed (workflow closed successfully), superseded (replaced by a later document), or archived (removed from active view).

- [x] **Metadata records when the document reached a closed status**
  When a document's status transitions from in-progress to any closed status, its metadata records the date of that transition.

- [x] **The workflow keeps metadata current as it transitions the document's state**
  When a workflow that owns a document opens it, completes it, or supersedes it, the workflow updates the document's metadata to reflect the new state, so metadata does not drift relative to the document's actual lifecycle.

- [x] **Metadata is accessible through the same channel as the artifact itself**
  Anyone consuming an artifact through its normal channel (opening the file, calling an API, etc.) can obtain the metadata through that same channel — no separate credentials, service, or tooling beyond what accessing the artifact already requires.

- [x] **Metadata is structured and machine-readable**
  Automated processes (e.g., archival sweeps, age-based triage) can extract every field of an artifact's metadata as structured data, without parsing prose or applying heuristics.

- [x] **Metadata is written only on documents produced after this feature ships**
  Existing documents in the specs, plans, and changelog stores at the time this feature ships remain bare — no backfill is attempted, and downstream consumers accept that pre-existing documents carry no metadata.

- [x] **Users and agents can list artifacts by metadata**
  A user or agent can list artifacts filtered by metadata fields — status, creation date range, and closed date range — with filters combinable in a single query and applicable across all covered artifact classes (specifications, plans, changelog entries, test plans).

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

None.

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

- [x] **New workflow documents have associated metadata on write**
  When any workflow writes a new document, obtaining that document through its normal access channel returns the document with associated metadata present.

- [x] **The metadata includes a creation date**
  Every newly-written document's metadata contains a field identifying the date the document was first written.

- [x] **The metadata includes the current status**
  Every document's metadata contains a status field, whose value is one of: in-progress, completed, superseded, archived.

- [x] **Metadata records the closing date when status transitions to a closed state**
  When a workflow closes a document (marks it completed, superseded, or archived), retrieving the document afterwards shows a closed-date field in the metadata identifying the date of that transition.

- [x] **Metadata reflects the actual lifecycle state after each transition**
  After a workflow performs an operation that changes the document's state (completing, superseding, archiving), the metadata retrieved for that document reflects the new state — status and any related date fields are updated in the same operation.

- [x] **Metadata is retrievable through the same channel as the document**
  Fetching a document through its normal access channel returns its metadata alongside the document — no separate call, service, or credential beyond what fetching the document already required.

- [x] **Metadata is extractable as structured data**
  An automated process can extract every field of a document's metadata as structured data using a parser appropriate to that document's storage substrate — no natural-language parsing or heuristic detection required.

- [x] **Existing documents remain bare after this feature ships**
  Documents that existed in the specs, plans, and changelog stores prior to this feature shipping are unchanged after the release — their content, file structure, and associated metadata are identical to what they were before.

- [x] **Filtering by status returns matching artifacts**
  A query filtered by status returns the set of artifacts whose current status matches the requested value(s), and excludes artifacts whose status does not match.

- [x] **Filtering by date range returns matching artifacts**
  A query filtered by a date range on creation date or closed date returns the set of artifacts whose date falls within that range, and excludes those outside.

- [x] **Combined filters intersect correctly**
  A query that specifies multiple filters (e.g., status and a date range together) returns only artifacts satisfying all of the filters simultaneously — not the union of their individual matches.

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

- The workflow that writes a document also owns updating its metadata. Whether that ownership is exposed as a dedicated update command or handled implicitly on the next write is deferred to plan-phase design.
- The choice of metadata storage substrate is delegated to each file writer, not centrally prescribed — this leaves the door open for non-file substrates (e.g. an external document-metadata API) without changing the spec.
- For current markdown artifacts, YAML frontmatter is the natural fit.
- The metadata schema is limited to the core fields captured in the requirements (creation date, status with the four listed values, closed-date-on-transition). The plan phase may propose additional fields if useful, but is not required to.

<!--
  SUCCESS METRICS
  How you will know the feature is working well after delivery. Be specific:
    - Quantitative: "p99 latency < 200ms", "error rate < 0.1%"
    - Behavioural: "users complete the flow without support intervention"
  Format: one bullet point per metric.
  Leave blank if not applicable.
-->
## Success Metrics

- Agents reading historical documents can extract and act on the metadata (e.g., weight information by age, distinguish completed from in-progress documents during archaeology reads).
- Users can perform archive triage on accumulated artifacts using only the metadata, without inspecting git history or reading document bodies.
- When someone opens a historical document written after this feature ships, they can determine its age and lifecycle status directly from the document's metadata, without any secondary lookup.
- Every new document produced by any Spektacular workflow (spec, plan, implement, changelog) carries metadata, verified by spot-checking recently-written artifacts after the feature ships.
- The metadata and search capabilities provide sufficient signal to enable future downstream features (e.g., an automatic archive action that mutates stale artifacts, or reports and dashboards built on the search results) without requiring changes to the metadata schema itself.

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

- Automatically archiving artifacts (moving them or mutating their status based on age) is out of scope — this feature enables discovery of stale artifacts through search, but does not perform the archive action itself.
- User-facing tooling or UI for viewing or editing metadata is out of scope — metadata is written and updated by workflows, and consumed by whoever reads the artifact through its normal access channel.
- Metadata on Spektacular's internal state and configuration files (workflow state, config) is out of scope — this feature covers the artifact documents that workflows produce as output, not the internal machinery.
- Metadata on transient scratch files that workflows create and remove during execution is out of scope — those files exist only for the duration of a workflow run and are removed at completion, so lifecycle metadata does not apply.
