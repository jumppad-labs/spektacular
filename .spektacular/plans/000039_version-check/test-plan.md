---
created_date: "2026-07-31"
status: completed
closed_date: "2026-07-31"
---

# Test Plan: 000039_version-check

Manual verification procedures for the success metrics not covered by
automated behavioural tests. Metric 2 ("no false alarms when the
installation matches") is fully covered by the automated suite
(`cmd/version_test.go`: a match carries no `action` text; the installed
skill preamble instructs silence on `match`) and is deliberately not
restated here.

## 1. Users are informed of stale files on next skill invocation (agent-relay portion)

The CLI half of this metric is automated: a mismatched recording yields
`status: "mismatch"` plus a relay-ready `action` message, and every
installed skill begins with the version-check preamble. What CI cannot
assert is that a live agent actually runs the check and relays the prompt.

- **What to measure**: on the first workflow-skill invocation in a repo
  whose recorded version differs from the binary, the agent tells the user
  their installed Spektacular files are out of date and asks them to
  re-run init before proceeding — without being asked to check.
- **How**:
  1. In a scratch git repository, run `spektacular init claude` (or the
     configured agent) with a released binary.
  2. Overwrite the record to force a mismatch:
     `echo "0.0.1" > .spektacular/version`.
  3. Start a fresh agent session in that repository and invoke any
     workflow skill (e.g. `/spek-new`).
  4. Observe the agent's first actions and first message.
  5. Re-run `spektacular init claude`, invoke the same skill again in a
     fresh session, and observe that no version-related output appears.
- **Expected result**: in step 4 the agent runs `spektacular version
  check` before any other command, surfaces an out-of-date message naming
  re-run of init, and waits for the user's decision; in step 5 the skill
  proceeds with zero version-related output.
- **Who / when**: the release manager (or any maintainer), once per
  release that changes the skill templates, against the release binary.
  Repeat once per supported agent (claude, bob, codex) when the agent
  integrations change.

## 2. Reports of "skills behaving oddly after upgrades" drop to zero

A field-observation metric with no automatable assertion.

- **What to measure**: the count of user-reported issues attributable to
  stale installed files (skills missing new instructions, agents following
  outdated workflows after a binary upgrade).
- **How**: review incoming GitHub issues and internal reports for
  staleness-attributable symptoms; label them (e.g. `stale-install`) and
  compare the count for the release windows before and after this feature
  shipped.
- **Expected result**: zero new staleness-attributable reports from
  repositories running a feature-bearing binary; any such report instead
  presents as the user having seen and ignored (or never received) the
  re-run-init prompt, which would indicate a preamble or relay defect to
  investigate under procedure 1.
- **Who / when**: maintainers triaging issues, ongoing; reviewed at each
  release retrospective.
