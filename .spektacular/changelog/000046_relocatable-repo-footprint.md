---
created_date: "2026-09-02"
status: completed
closed_date: "2026-09-02"
---

# Relocatable repo footprint

## What was built

A repo's Spektacular files (its `repo.yaml`, knowledge store, and changelog) can now live apart from its code. The repo configuration gained one optional `source` key naming where the code lives: a file location (a plain path or `file://` path, absolute, relative to the folder holding `repo.yaml`, or using `${VAR}`) or a git location (`git://`, `ssh://`, `https://`, or `user@host:path`) that Spektacular clones into the project's `.spektacular/repos/<name>/` folder on first use and never fetches or pulls afterwards. With no `source`, a repo behaves exactly as before, with its Spektacular files colocated inside its code.

The project registry now says only where each repo's Spektacular files are: an entry is `name` plus `location` (the folder holding the repo's `.spektacular/`). The older `local` key is accepted as an alias and written back as `location`; the `address` key is no longer accepted, and a configuration that still carries it fails to load with an error naming the repo and the exact `source:` line to add to that repo's `repo.yaml`. Registration (`repo add`) requires `location`, creates the folder when it is missing, accepts `source` and writes it into the repo's own configuration like the descriptive fields, rejects an `address` key with a corrected payload, and clones a git source on the spot. Listing (`repo list`) reports the resolved source as each repo's `root` and never clones: a git source not yet cloned shows an empty root.

Resolution carries two locations per repo, a root holding the Spektacular files and a source holding the code. Everything that reads or writes the repo's own files (footprint repair, knowledge aggregation, the per-repo changelog store, registration write-back, project init) stays on the root; everything that touches code (listing, git operations, the roster embedded in workflow instructions) uses the source. Regression tests prove knowledge entries and per-repo changelog records land under the folder holding `repo.yaml` while a separate source directory stays byte-identical.

The implement workflow no longer writes a release note to a `CHANGELOG.md` at the root of each changed repo; that step was removed, the changelog step advances straight to the test plan, and the per-repo changelog record written at the end now opens with a short user-facing summary. A feature finishes with exactly one project record and one record per affected repo, each at the location its configuration declares.

Every workflow that sends an agent into a repo's code now embeds each registered repo's resolved source in its instructions: the spec interview and the plan discovery and architecture steps show it on their roster lines, and the implement workflow's read-plan, analyze, implement, test, verify, and feature-changelog steps now receive the roster too and open with a "Where the code lives" block. No step or skill template refers to the directory the agent is running in or the project's working tree as a stand-in for a repo, and a template-contract test keeps those phrases out. The path-prefix convention in the changelog steps is now defined by the number of registered repos.

The harbor implement suite moves its `docs` repo into the separate layout (Spektacular files under the project, code elsewhere via a file source) and asserts the code checkout gains no `.spektacular/` directory and no `CHANGELOG.md`; a full run passed. The documentation site, the README, and the repo-management skill describe `source` and `location`, contrast the repo-level `source` with the project-level git address of the same name, and show the colocated layout and the separate layout with a file source and with a git source.

## Why it matters

Teams that cannot, or prefer not to, commit Spektacular files into a code repository can now run everything from a project folder that holds one folder per repo and points each at its code. That project folder can be committed and shared while the code repositories receive only code changes, and existing colocated setups keep working without any migration.

## Deviations from the plan

- Phases 1.2 and 1.3 were implemented in one code pass, because the address-driven clone tests could only be rewritten meaningfully once a git `source` could clone; each phase was still verified and ticked in its own workflow loop.
- The command layer's error conversion now recovers a wrapped structured error with `errors.As`, so validation errors built inside the config loader keep their remediation text.
- `repo list` no longer echoes `local` at all; the alias is folded when the configuration is parsed.
- The roster block was added to the six implement templates in Phase 3.1 rather than 3.2, since Phase 3.1's rendering criterion could not pass without it.
- The template-contract test's banned-phrase list caught "resolved local path" in the repo-management skill, which was reworded ahead of that skill's full rewrite.
- The repo-management skill still names `address`, but only to say the registry no longer takes it and that a payload carrying it is rejected.
- The harbor suite's `environment/state.json` fixture, not named in the plan, also listed the removed step and was updated.
- The git integration tests serve their fixture repository over git's dumb HTTP protocol, since a plain local path is a file source rather than a git source.
