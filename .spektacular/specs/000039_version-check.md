---
created_date: "2026-07-31"
status: completed
closed_date: "2026-07-31"
---

# Feature: 000039_version-check

## Overview

Spektacular installs skills and other agent files into a repository when the user runs init. When the user later upgrades the Spektacular binary, those installed files can become stale — the new binary may ship updated or additional skills — but nothing detects this today; the user must simply remember to re-run init.

This feature records which binary version installed the agent files and detects when the installed files no longer match the binary in use. Whenever a skill is invoked, the recorded version is checked against the current binary, and on a mismatch the user is prompted to re-run init to refresh the installed files.

Anyone using Spektacular across binary upgrades benefits: they no longer need to remember to re-initialize after upgrading, and they stop silently working with outdated skills and agent instructions.

## Requirements

- [x] **Record the installing version**
  When the user initializes a repository, the system records which binary version performed the installation alongside the installed agent files.

- [x] **Check version on skill invocation**
  Every time a skill is invoked, the recorded installed version is compared against the version of the binary currently in use.

- [x] **Prompt to re-initialize on mismatch**
  When the recorded version does not match the current binary version, the user is prompted to re-run init to refresh the installed files.

- [x] **Re-initializing clears the mismatch**
  After the user re-runs init with the current binary, the recorded version matches again and subsequent skill invocations proceed without a prompt.

- [x] **Missing or unreadable recorded version is handled**
  If no recorded version exists (files installed by a binary that predates this feature), or the recorded version is empty or cannot be parsed, the check treats the installation as out of date and prompts the user to re-run init rather than failing.

- [x] **Matching versions stay silent**
  When the recorded version matches the current binary, skill invocations proceed with no version-related output or interruption.

## Constraints

- The recorded version must live in a dedicated version file written at init time, containing the version of the binary that installed the files.
- The version recorded must be the binary's own build version — the same version the binary reports for itself — not a separately maintained number.
- The check must not modify or re-install any files itself; refreshing the installation remains an explicit, user-initiated re-run of init.
- The feature must not break repositories initialized before it existed.

## Acceptance Criteria

- [x] **Version file exists after init**
  After running init in a repository, a version file exists on disk whose content is the version of the binary that ran the init.

- [x] **Mismatch produces a prompt**
  With a recorded version that differs from the current binary's version, invoking any skill surfaces a message telling the user the installed files are out of date and asking them to re-run init.

- [x] **Mismatch does not modify installed files**
  With a mismatched version, invoking a skill leaves all installed agent files and the version file unchanged — only the prompt is produced.

- [x] **Re-running init resolves the mismatch**
  After re-running init with the current binary, the version file contains the current binary's version and a subsequent skill invocation produces no version warning.

- [x] **Missing or invalid version file is treated as stale**
  In a repository with no version file, an empty version file, or a version file whose content cannot be parsed as a version, invoking a skill surfaces the same re-run-init prompt instead of erroring.

- [x] **Matching versions produce no output**
  With a recorded version equal to the current binary's version, invoking a skill produces no version-related message and the skill proceeds normally.

## Technical Approach

- The suggested name for the version file is `.spektacular_version`; the plan may settle its exact name and location (e.g. inside the tool's existing dotdirectory) alongside existing conventions.
- The version check is expected to be driven by the agent at skill-invocation time — likely via an instruction rendered into the installed skill files telling the agent to compare the version file against the binary — rather than by a runtime hook in the binary.
- Consider exposing a cheap way for the agent to perform the comparison (e.g. a single CLI call that reports match/mismatch) so every skill does not need to implement comparison logic itself.
- Known risk: during development the binary runs via `go run .` and reports the default version, so version-string comparison cannot distinguish dev builds whose templates differ; a content hash of the embedded templates was discussed as a possible complementary signal. The plan may address or explicitly descope this.

## Success Metrics

- After upgrading the binary, users are informed of stale installed files on their next skill invocation instead of discovering it themselves — no one has to remember to re-run init unprompted.
- No false alarms in day-to-day use: when the installation matches the binary, users never see a version warning.
- Reports of "skills behaving oddly after an upgrade" attributable to stale installed files drop to zero.

## Non-Goals

- Detecting user hand-edits to installed files is out of scope; the check compares recorded install version to binary version, not file contents.
- Checking on every CLI command run is out of scope.
- No migration tooling for existing installations: repositories that predate this feature simply see the re-run-init prompt until they re-initialize.
