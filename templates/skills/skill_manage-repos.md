# Manage Repos

Guide for driving the repo-management CLI on the user's behalf: registering a project's member repos, inspecting the registry, and handling materialization, staleness, and footprint repair.

## Concepts

A **project** is a collection of repos with central spec/plan/changelog storage; a **repo** participates by being registered in the project's configuration. Each registered entry carries a required slug-safe `name` (plans and changelog attribution reference repos by name), an `address` (remote URL) and/or a `local` path — at least one is required, and `local` wins when both are set — plus descriptive metadata: what the repo does, its role, tags, dependencies, and deployment type. A repo's own footprint is minimal (its `repo.yaml` config and knowledge storage) and carries no pointer back to any project, so one repo can belong to many projects.

## Registering a repo

Run the `repo add` command with a JSON payload:

```
repo add --data '{"name":"docs","local":"../docs","description":"the documentation repo","role":"documentation","tags":["docs"],"deployment":"static-site"}'
```

- Registration is idempotent: re-adding the same entry changes nothing; re-adding with different metadata updates the registry entry in place; adding a repo already initialized by another project registers it here without disturbing its existing footprint.
- The command registers the entry in the project config and creates or repairs the target repo's minimal footprint in one operation, reporting `{"registered": true, "footprint": "created"|"repaired"|"unchanged"}`.
- For a repo registered only by `address`, adding it materializes it by cloning — so expect the first add of a remote-only repo to take as long as a clone. A `local` path must already exist on disk.
- Fill in the descriptive metadata whenever the user can supply it — planning quality depends on it: the plan workflow uses each repo's description, role, and tags to attribute requirements to the right repo.

## Inspecting the registry

```
repo list
```

Reports every registered repo with its metadata, resolved local path, whether it is a project-managed clone (`materialized`), and a `stale_note` when a clone has fallen behind its remote. Listing is side-effect-free: an address-only repo that has not been materialized yet reports an empty root rather than triggering a clone.

## Materialization and staleness

- A repo with a usable `local` path always resolves to it; git is never involved.
- A repo with only an `address` is cloned into `.spektacular/repos/<name>/` inside the project the first time it is actually used, and reused thereafter. The folder is gitignored, so clones never enter the project's history.
- Spektacular never fetches or pulls on its own. A stale clone produces a warning (`stale_note`); if the user wants it updated, they update it themselves (e.g. `git -C .spektacular/repos/<name> pull`) — always confirm with the user before suggesting commands that change a clone.

## Footprint repair

Touching a registered repo whose `repo.yaml` is missing or invalid produces a structured `repo_footprint` error offering repair, never a silent failure. To repair, re-run `repo add` with the repo's name (its registration is preserved; only the footprint is recreated), or re-run the project's init command, which cascades over all registered repos and repairs their footprints.

## Removal

Removal is deliberately a manual edit: delete the repo's entry from the `repos` list in the project's `config.yaml`. No command is provided, and nothing in the repo itself needs cleaning up — its footprint carries no project pointer. Confirm with the user before editing their config.
