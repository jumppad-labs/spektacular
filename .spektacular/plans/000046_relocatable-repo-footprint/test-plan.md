---
created_date: "2026-09-02"
status: completed
closed_date: "2026-09-02"
---

# Test Plan: 000046_relocatable-repo-footprint

One success metric is verified manually; the other (a clean `git status` in an external code repository after an implement run) is covered by the harbor implement suite's separate-source fixture and its two source-cleanliness verifier tests, so it is not restated here.

## Metric: an agent registers a separate-source repo correctly on the first attempt

**What to measure**: Following only the repo-management skill, a coding agent registers a repo whose Spektacular files live apart from its code without the user explaining the colocated versus separate layouts. Pass condition: the first `repo add` the agent runs carries both `location` and `source`, and the resulting registration is correct.

**How**:

1. In a fresh directory, run `spektacular init claude` (or the agent of choice) so the skills are installed.
2. Have a code checkout elsewhere on disk with no `.spektacular/` directory, for example `${HOME}/code/api`.
3. Start a new agent session in the project directory and give it only this instruction, with no further explanation: "Register the api service whose code is checked out at ~/code/api as a member of this project. Keep its Spektacular files in this project folder, not in the code repository. Use the manage-repos skill."
4. Watch the first `spektacular repo add` command the agent runs.
5. Afterwards run `spektacular repo list` and inspect `.spektacular/config.yaml`, `repos/<name>/.spektacular/repo.yaml`, and `~/code/api`.

**Expected result**:

- The first `repo add` payload contains `"location":"./repos/api"` (or another folder under the project) and `"source":"file://${HOME}/code/api"` (or the plain absolute path); the agent does not ask which layout to use and does not use an `address` key.
- `config.yaml` registers `api` with `location` only.
- `repos/api/.spektacular/repo.yaml` contains `source: file://…/code/api` and the knowledge and changelog directories exist beside it.
- `repo list` reports `root` as the absolute path of `~/code/api` and `materialized: false`.
- `~/code/api` contains no `.spektacular/` directory and `git -C ~/code/api status --short` prints nothing.

Repeat once with a git source (an instruction naming a git URL instead of a checkout): the first payload carries `"source":"<git url>"`, the clone appears under `.spektacular/repos/api/`, and `repo list` reports the clone as `root` with `materialized: true`.

**Who / when**: The maintainer, once per release that changes `templates/skills/skill_manage-repos.md`, with the agent the team ships (Claude Code today) before tagging the release.
