# Manage Repos

Guide for driving the repo-management CLI on the user's behalf: registering a project's member repos, inspecting the registry, and handling materialization, staleness, and footprint repair.

## Concepts

A **project** is a collection of repos with central spec/plan/changelog storage; a **repo** participates by being registered in the project's configuration. Each registered entry carries a required slug-safe `name` (plans and changelog attribution reference repos by name), a required `location`, and optional `dependencies`.

A repo has two locations. The registry's `location` (the older `local` key still works and means the same thing) names the folder holding its `repo.yaml`, alongside its knowledge and changelog; a relative `location` is resolved from the folder holding `config.yaml` and nothing is appended to it, so the project's own footprint is `.` and a repo folder in the project is `../repos/<name>`. The repo's `source`, in its own `repo.yaml`, names its code as a provider block like every other section: `provider: file` with a `config.location` path (absolute, relative to the folder holding `repo.yaml`, or using `${VAR}`), or `provider: git` with a `config.location` git address (`git://`, `ssh://`, `https://`, or `user@host:path`) that is cloned into the project's `.spektacular/repos/<name>/` on first use. A `repo add` payload still gives `source` as one value and the transport decides the provider written to the file. A repo is **colocated** when its Spektacular files sit inside its code, in a `.spektacular/` folder whose `repo.yaml` declares a file source at `..`; this is what `repo add` scaffolds. A repo is **separate** when its Spektacular files live in a folder of their own (for example one folder per repo under the project) whose `source` points at the code elsewhere, so the code repository receives only code changes. The registry no longer takes `address`; a repo's git origin is its `source`.

Descriptive metadata (what the repo does, its role, and tags) also lives in `repo.yaml`. A repo's own footprint is minimal (its `repo.yaml` config and knowledge storage) and carries no pointer back to any project, so one repo can belong to many projects.

## Registering a repo

Run the `repo add` command with a JSON payload:

A colocated repo (Spektacular files inside its code) needs only a `location`:

```
repo add --data '{"name":"docs","location":"../../docs","description":"the documentation repo","role":"documentation","tags":["docs"]}'
```

A separate repo adds a `source` for its code: a file location for a checkout already on disk, or a git location for Spektacular to clone:

```
repo add --data '{"name":"api","location":"../repos/api","source":"file://${HOME}/code/api","description":"the API service","role":"backend","tags":["go"]}'
repo add --data '{"name":"docs","location":"../repos/docs","source":"git@example.com:org/docs.git","description":"the documentation repo","role":"documentation","tags":["docs"]}'
```

- `location` is required and points at the repo's code; `repo add` creates a `.spektacular/` folder inside it to hold the repo's `repo.yaml`, knowledge, and changelog, writes a file source pointing at `..` there, and registers that folder. `source` is optional and is written into that `repo.yaml` alongside the descriptive metadata; re-adding without `source` leaves the stored value alone.
- Registration is idempotent: re-adding the same entry changes nothing; re-adding with different metadata updates the registry entry in place; adding a repo already initialized by another project registers it here without disturbing its existing footprint.
- The command registers the entry in the project config and creates or repairs the target repo's minimal footprint in one operation, reporting `{"registered": true, "footprint": "created"|"repaired"|"unchanged"}`.
- A git source is cloned on registration, so expect the first add of a git-source repo to take as long as a clone. A file source is not checked; a wrong path surfaces when git or the agent touches it.
- A payload carrying `address` is rejected with the corrected payload; put the git URL in `source`.
- Fill in the descriptive metadata whenever the user can supply it; planning quality depends on it: the plan workflow uses each repo's description, role, and tags to attribute requirements to the right repo.

## Inspecting the registry

```
repo list
```

Reports every registered repo with its `location`, its metadata, the resolved source of its code as `root` (the file source, the clone of a git source, or the location itself when no source is set), the `provider` that repo declares for its source (`file`, `git`, or absent when it declares none), whether it is a project-managed clone (`materialized`), and a `stale_note` when a clone has fallen behind its remote. Listing is side-effect-free: a repo whose git source has not been cloned yet reports an empty root rather than triggering a clone.

## Materialization and staleness

- A repo with no `source`, or with a file source, resolves to that directory; git is never involved.
- A repo whose `source` is a git location is cloned into `.spektacular/repos/<name>/` inside the project the first time it is actually used, and reused thereafter. The folder is gitignored, so clones never enter the project's history.
- Spektacular never fetches or pulls on its own. A stale clone produces a warning (`stale_note`); if the user wants it updated, they update it themselves (e.g. `git -C .spektacular/repos/<name> pull`). Always confirm with the user before suggesting commands that change a clone.

## Footprint repair

Touching a registered repo whose `repo.yaml` is missing or invalid produces a structured `repo_footprint` error offering repair, never a silent failure. To repair, re-run `repo add` with the repo's name (its registration is preserved; only the footprint is recreated), or re-run the project's init command, which cascades over all registered repos and repairs their footprints.

## Removal

Removal is deliberately a manual edit: delete the repo's entry from the `repos` list in the project's `config.yaml`. No command is provided, and nothing in the repo itself needs cleaning up; its footprint carries no project pointer. Confirm with the user before editing their config.
