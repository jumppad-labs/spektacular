---
created_date: "2026-08-13"
status: completed
closed_date: "2026-08-13"
---

# Test Plan: 000039_project-level-capabilities

Success metrics 3 (derived changelog entries unambiguously identify project and spec/plan, including the two-projects-one-repo collision case) and 4 (`.spektacular_ignore` exclusions keep listings and search clean while direct access still works) are covered by automated behavioural tests (`cmd/changelog_file_test.go`, `internal/store/ignore_test.go`, `cmd/knowledge_ignore_test.go`) and are deliberately not restated here. The three remaining metrics are agent-judgement or wall-clock observations that need a live run.

## Metric 1 — A multi-repo project goes spec → plan → implementation in one Spektacular project with no manual coordination

- **What to measure**: A feature whose requirements span two repos (one code change, one documentation change) completes the full spec → plan → implement workflow inside a single Spektacular project, with no hand-copying of context between repos and no separate Spektacular setup in the second repo.
- **How**: Build a two-repo fixture: `mkdir code docs && git init` each, `spektacular init claude --name demo` inside `code/`, then `spektacular repo add --data '{"name":"docs","local":"../docs","description":"user documentation for the code repo","role":"documentation"}'`. Drive a real agent session (Claude Code with the installed spek-new / spek-plan / spek-implement skills) through a feature like "add a --verbose flag to the CLI and document it in the docs repo": `/spek-new`, then the plan workflow, then the implement workflow, answering only the workflow's own questions.
- **Expected result**: The run finishes with changes committed-ready in both working trees (code change in `code/`, doc change in `docs/`), a central changelog entry under `code/.spektacular/changelog/demo/`, and a derived entry under `docs/.spektacular/changelog/demo/` — with the operator never having to point the agent at the second repo manually.
- **Who / when**: Maintainer, before tagging the release containing this feature; re-run when the plan or implement step templates change. (A dedicated harbor scenario for this flow is recorded follow-up work; until it exists this stays manual.)

## Metric 2 — Cross-repo plans attribute requirements to the right repo without user correction

- **What to measure**: During the Metric 1 run's plan workflow, the drafted architecture and phases place each requirement in the correct repo on the first draft — zero attribution corrections from the operator during the walkthrough.
- **How**: In the same session as Metric 1, when the plan reaches its walkthrough, count how many times the operator has to redirect a requirement to a different repo (e.g. "no, the flag documentation belongs in the docs repo"). Inspect the committed `plan.md` phases and `context.md` file lists for `docs:`-prefixed paths landing on documentation work and unprefixed paths on code work.
- **Expected result**: Zero repo-attribution corrections; the documentation-shaped requirement is attributed to the docs repo (with plausible files) in the first presented draft. The automatable slice — that the rendered discovery/architecture instructions carry the repo roster and attribution directives — is already asserted by `internal/steps/plan/steps_test.go`; this procedure judges the resulting agent behaviour.
- **Who / when**: Maintainer, same run as Metric 1.

## Metric 5 — Setting up a new project including repo registration takes minutes

- **What to measure**: Wall-clock time from an empty directory to a fully set-up project with two registered repos, target under 5 minutes.
- **How**: Time the following, run cold: `spektacular init claude --name timing-test`; `spektacular repo add --data '{"name":"local-repo","local":"../some-existing-repo"}'`; `spektacular repo add --data '{"name":"remote-repo","address":"<a real private git URL you can clone>"}'` (this one includes the clone — use a repository of realistic size); `spektacular repo list` to confirm both resolve.
- **Expected result**: Total elapsed time ≤ 5 minutes, dominated by the clone; every command succeeds first try with no manual file editing required. The private-remote clone also doubles as the check that authentication delegates to the user's own git credentials (the reason the git binary was chosen) — the clone must succeed without Spektacular-specific credential setup, and this is the one place remote auth is exercised, since the automated suite uses only local path remotes.
- **Who / when**: Maintainer, once per release on at least Linux plus one of macOS/Windows — cross-platform behaviour of the git executor (PATH lookup, CRLF handling, credential managers) is a recorded open question with no CI matrix coverage, so the non-Linux run is part of this procedure.
