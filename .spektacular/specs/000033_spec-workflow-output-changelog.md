# Feature: 000033_spec-workflow-output-changelog

## Overview

When a spec or plan workflow finishes, its output today serves the immediate purpose — building the feature. But that same work is often valuable input for other downstream activities that happen afterward, like writing user-facing documentation or a blog post announcing the change. Right now nothing captures the completed work in a form suited to that later use: someone doing docs or marketing has to re-read the spec, the plan, and often the conversation itself to reconstruct what happened and why.

This change produces a durable, dense record of each completed unit of work — capturing enough context that a separate process, running later and without access to the original conversation, can pick it up and generate accurate documentation or announcements from it.

## Requirements

- [ ] **A durable record is produced when a unit of work is fully implemented**
  When the implement workflow for a feature completes, the system produces a persistent record of that completed work, without requiring the user to manually write it.

- [ ] **The record captures what was actually done, not what was planned**
  Unlike the plan (which describes intended work before it happens), the record describes the completed change in retrospect — what was built and why it matters — even where the actual implementation diverged from the original plan.

- [ ] **The record is self-contained**
  A reader — human or agent — with no access to the original conversation must be able to understand what was built, why, and what it enables for the user, from the record alone.

- [ ] **The record serves both human and agent consumers**
  The record must be structured and dense enough for an agent to reliably extract facts from it, while remaining readable prose a human can use directly when writing documentation or an announcement.

- [ ] **Records accumulate over time as a sequential history**
  Each completed unit of work adds to an ongoing collection rather than overwriting or discarding earlier records, so the record reflects the project's history of changes.

- [ ] **A downstream process can retrieve the record for a specific completed feature**
  Something consuming the changelog later must be able to find and read the record(s) relevant to a specific completed feature, not just the collection as a whole.

## Constraints

- Must not require a separate service, database, or external dependency — stays consistent with Spektacular running as a single self-contained binary.
- The record must be a file suitable for version control, stored alongside the project's other Spektacular artifacts — not written to an ephemeral location or external system.
- The assistant must not automatically commit the record to version control — the user retains control over when and how it is committed, the same as any other file change.
- Must not replace or repurpose `research.md` or `plan.md` — those remain prospective, planning-internal documents; this is a distinct, additional, retrospective artifact.

## Acceptance Criteria

- [ ] **Record produced on implementation completion**
  When an implement workflow finishes for a feature, a new changelog record for that feature exists afterward, without the user having written it themselves.

- [ ] **Record reflects actual outcome, not the original plan**
  Where the implemented feature differs from what the plan described, the record's content reflects the actual outcome — a reader comparing the record to the plan can spot the divergence and understand why it happened.

- [ ] **Record is understandable without the conversation**
  A reader given only the record (no chat transcript, no plan, no spec) can state what was built, why it was built, and what it enables — correctly and without guessing.

- [ ] **Record works for both an agent and a human reader**
  An agent processing the record can reliably extract structured facts (e.g. feature name, what changed, why) from it, and a human reading the same record can use its prose directly, without a separate human-only version being produced.

- [ ] **Records accumulate rather than replace**
  After multiple features are implemented, records for all of them still exist and are individually readable — completing a new feature does not remove or overwrite an earlier feature's record.

- [ ] **A specific feature's record can be retrieved on its own**
  Given the name or identifier of a completed feature, its corresponding record can be retrieved individually, without needing to read or search the entire history.

## Technical Approach

- No technical direction has been decided beyond the captured constraints; the storage shape (e.g. one growing file vs. one file per feature), its location relative to specs and plans, and how a downstream process indexes or locates a specific feature's record are left for the plan workflow to propose.

## Success Metrics

- The changelog record for 000032 (once implemented) is sufficient on its own to write user-facing documentation or a blog post about it, without needing to re-read the spec, plan, or conversation.
- Users rarely need to supplement a changelog record with outside context when writing docs or announcements from it.

## Non-Goals

- Building the downstream doc-writing or blog-writing process itself — this feature produces the input artifact only; consuming it to actually generate docs or a blog post is separate future work.
- Retroactively generating records for features implemented before this capability existed — the changelog starts capturing from when this feature ships, not backfilled for prior history.
- A viewer, browsable UI, or search interface for the changelog — retrieval by a downstream process is required, but a dedicated UI for humans to browse it is not.
- Automatic redaction, retention policies, or lifecycle management of old records — records are kept indefinitely with no automated pruning or sensitive-data handling.
