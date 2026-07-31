---
created_date: "2026-07-31"
status: completed
closed_date: "2026-07-31"
---

# Changelog: 000039_version-check

## What was built

Spektacular now detects when the agent files it installed into a repository
have gone stale after a binary upgrade, and prompts the user to refresh them.
The feature landed in three parts:

1. **Version recording at init.** Running init writes a dedicated
   `.spektacular/version` file containing the bare version string of the
   binary that performed the installation (newline-terminated, no build-sha
   suffix, so two builds of the same release always match). Re-running init
   overwrites the file unconditionally — that re-run is the entire repair
   path for a stale installation. Init's human-readable output reports the
   recorded version. The file is committed to git by default, making
   staleness team-visible.

2. **A `version check` command.** A new read-only subcommand reads the
   recorded version, trims whitespace, and compares it for exact string
   equality against the running binary's version. It reports one of three
   states through the standard JSON success envelope with exit code 0:
   `match` (both versions, no action text), `mismatch`, or `missing` (no
   file, or empty/whitespace-only content). The two stale states carry an
   `action` field with a ready-to-relay instruction to re-run init, naming
   the configured command and agent (e.g. `spektacular init claude`). Exit
   code 1 and the agent-actionable error shape are reserved for genuine
   faults such as an unreadable version file. The command supports
   `--schema` self-description and `--fields` filtering, and joined the
   project-wide envelope contract tests alongside every other command
   family.

3. **A version-check preamble in every workflow skill.** All four workflow
   skill templates (spek-new, spek-plan, spek-implement, spek-knowledge)
   begin with an identical blockquote instructing the agent to run
   `{{command}} version check` before any other command, proceed silently
   on `match`, and on `mismatch`/`missing` relay the `action` message and
   wait for the user to re-run init — never modifying installed files
   itself. Because all three supported agents (claude, bob, codex) install
   byte-identical renders of these templates, one template edit per skill
   reached every agent. This repository was re-initialized as part of the
   work, so its own installed skills carry the preamble and its version
   file records the dev binary's `0.1.0`.

Test coverage: exhaustive unit tests for the pure classification function;
command-level tests for all three states, the read-only guarantee, the
unreadable-file fault path, and schema output; init tests for fresh
recording, stale-overwrite, and idempotency; per-agent install assertions
guarding the preamble in all twelve rendered skill files. A test plan in the
plan store captures the two manual checks: that a live agent actually relays
the stale prompt, and the field observation that stale-install bug reports
drop to zero.

## Why it matters

Before this feature, upgrading the Spektacular binary silently left a
repository's installed skills and agent instructions stale — the new binary
might ship updated skills, but nothing noticed, and users had to remember to
re-run init or discover the drift as mysteriously outdated agent behavior.
Now staleness announces itself at the next skill invocation: the agent
checks freshness first, tells the user their installed files are out of
date, and asks them to re-run init. When the installation matches, users see
nothing at all. Repositories initialized before the feature existed simply
report a missing record and receive the same re-run-init prompt rather than
erroring.

## Deviations from the plan

None. All three phases shipped as planned. Two planning watch-points
resolved favorably during implementation: the envelope contract tests
accepted the two-word `version check` family as plain subtests with no
harness change, and the per-agent install tests' existing assertion loops
covered all four skills, so no separate template-walking guard test was
needed. One test-hygiene addition: the `--schema` flag persists on the
package-level command across in-process test invocations, so a
`resetVersionCheckFlags` helper was added to prevent flag-state leakage.
