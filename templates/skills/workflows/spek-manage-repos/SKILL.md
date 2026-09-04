---
name: spek-manage-repos
description: Register a new repo in the current Spektacular project, inspect the registry, and repair a repo's footprint.
---

> **Version check first.** Before running any other command, run `{{command}} version check`.
> - On `status: "match"`, continue with the skill and produce no version-related output.
> - On `"mismatch"` or `"missing"`, the installed Spektacular files are out of date: relay the response's `action` message to the user, ask them to re-run `{{command}} init <agent>`, and wait for their decision before continuing.
> - Never modify or re-install any installed files yourself. Refreshing the installation is always an explicit, user-initiated re-run of init.

# What this skill does

This skill drives the `{{command}} repo` surface: registering a repo in the current project, inspecting what is already registered, and repairing a repo whose footprint is missing or broken. It is a static playbook, not an interactive CLI state machine, so there is no session to resume. Recognise what the user is asking for, pick the matching section below, and run the commands directly.

# When to invoke

- "Add a repo." / "Register the docs repo." / "Add this project to Spektacular."
- "What repos are in this project?" / "Where does the API repo's code live?"
- Any `repo_footprint` or `repo_footprint_missing` error returned by another command.
- Before attributing work to a repo, when you need each repo's resolved `root`.

# Concepts

A **project** is a collection of repos with central spec, plan, and changelog storage. A **repo** joins a project by being registered in the project's `config.yaml`. Each registered entry carries a slug-safe `name` (plans and changelog attribution reference repos by name), a `location`, and optional `dependencies`.

A repo has two locations, and keeping them apart is the whole trick:

- The **footprint** is the folder holding the repo's `repo.yaml`, alongside its knowledge and changelog. This is what the registry's `location` records. (The older `local` key still works and means the same thing.)
- The **source** is where the repo's *code* lives. It is declared inside that `repo.yaml` as a provider block, the same shape every other section uses: `provider: file` with a `config.location` path, or `provider: git` with a `config.location` git address that is cloned on registration.

A repo is **colocated** when its footprint sits inside its code, in a `.spektacular/` folder whose `repo.yaml` declares a file source of `..`. This is what `repo add` scaffolds. A repo is **separate** when its footprint lives in a folder of its own and its `source` points at code elsewhere, so the code repository receives only code changes.

Descriptive metadata (`description`, `role`, `tags`) lives in `repo.yaml`, never in the project config. A repo's footprint carries no pointer back to any project, so one repo can belong to many projects.

# Adding a repo to this project

## 1. See what is already registered

```
{{command}} repo list
```

Never skip this. It tells you whether the repo is already registered (in which case you are updating, not adding) and shows the shape of the existing entries.

## 2. Work out the code location

`location` is the folder holding the repo's **code**. `repo add` creates a `.spektacular/` folder *inside* it and registers that folder, so the registry ends up recording `<location>/.spektacular`.

> **A relative `location` is anchored at the project's `.spektacular/` folder, not at the project root.** It is stored in `config.yaml` verbatim and resolved from the folder that file lives in. So a sibling of the project directory is `../../<sibling>`, not `../<sibling>`.
>
> **Prefer an absolute path** unless the user specifically wants a portable relative one. It is unambiguous and it is what `repo list` reports back.

## 3. Gather the descriptive metadata

Always fill in `description`, `role`, and `tags` when the user can supply them. Planning quality depends on it: the plan workflow uses each repo's description, role, and tags to attribute requirements to the right repo. If the user cannot supply them, `repo add` returns a `metadata_note` saying so, and so does every later `repo list`.

## 4. Decide whether to give a `source`

Omit `source` for a colocated repo. The scaffolded `repo.yaml` already declares a file source of `..`, which is the code the footprint sits inside.

Give `source` only when the footprint and the code are in different places:

- a path (plain or `file://`) for a checkout already on disk;
- a git location (`git://`, `ssh://`, `https://`, or `user@host:path`) for code Spektacular should clone.

Both forms accept `${VAR}`, expanded from the environment when the file is read. Any other URL scheme is rejected.

## 5. Show the payload and confirm

`repo add` writes into another repository on disk, creating a `.spektacular/` folder there. Show the user the exact payload and say which folder will be created before running it, and wait for explicit confirmation.

## 6. Run the command

A colocated repo, which is the common case:

```
{{command}} repo add --data '{"name":"docs","location":"/home/me/code/docs","description":"the documentation repo","role":"documentation","tags":["docs"]}'
```

That scaffolds `/home/me/code/docs/.spektacular/` and registers that folder.

A separate repo adds a `source` naming the code. Here the `location` is a footprint folder held under the project and the code stays untouched elsewhere:

```
{{command}} repo add --data '{"name":"api","location":"../repos/api","source":"file://${HOME}/code/api","description":"the API service","role":"backend","tags":["go"]}'
{{command}} repo add --data '{"name":"docs","location":"../repos/docs","source":"git@example.com:org/docs.git","description":"the documentation repo","role":"documentation","tags":["docs"]}'
```

Each of those scaffolds `<project>/.spektacular/../repos/<name>/.spektacular/` and registers that folder. Read the anchoring note in step 2 again before using a relative `location`.

## 7. Read the result, then verify

Success is `{"registered": true, "footprint": "created"|"repaired"|"unchanged"}`, plus a `metadata_note` when the repo still has no descriptive metadata. Follow it with `{{command}} repo list` and check that the new entry's `root` is the code you expected.

# Rules that apply to every add

- Registration is idempotent. Re-adding the same entry changes nothing; re-adding with different metadata updates it in place; adding a repo another project already initialized registers it here without disturbing its existing footprint. Re-adding without `source` leaves the stored source alone.
- A git source is cloned on registration, so the first add of a git-source repo takes as long as a clone.
- The location is resolved as part of the add, so a footprint that cannot be created or a clone that fails surfaces immediately rather than later.
- A payload carrying `address` is rejected with the corrected payload. A repo's git origin is its `source`.

# Inspecting the registry

```
{{command}} repo list
```

Each entry reports:

- `name`, and the resolved `location` of its footprint;
- the resolved source of its code as `root`, which is the file source, the clone of a git source, or the location itself when the repo declares no source;
- the `provider` the repo declares for its source (`file`, `git`, or absent when it declares none);
- `description`, `role`, `tags`, and `dependencies`;
- `materialized`, true when the root is a project-managed clone;
- `stale_note` when a clone has fallen behind its remote;
- `metadata_note` when the repo has no descriptive metadata set.

Listing does not clone: a repo whose git source has not been materialized yet reports an empty `root` rather than triggering a clone. A registered repo whose footprint is missing is reported as a `repo_footprint_missing` error naming the path that was looked at, not listed with an empty root as if that were fine.

# Materialization and staleness

- A repo with no `source`, or with a file source, resolves to that directory. Git is never involved.
- A repo whose `source` is a git location is cloned into `.spektacular/repos/<name>/` inside the project. Project init gitignores that folder, so clones never enter the project's history.
- Spektacular never fetches or pulls on its own. A stale clone produces a `stale_note` warning and nothing more. If the user wants it updated they update it themselves, for example `git -C .spektacular/repos/<name> pull`. Always confirm with the user before suggesting a command that changes a clone.

# Footprint repair

Touching a registered repo whose `repo.yaml` is missing or invalid produces a structured error offering repair, never a silent failure: `repo_footprint_missing` from `repo list`, and `repo_footprint` from the knowledge and store-file commands. To repair, either re-run `repo add` with the repo's name and its code location (the registration is preserved; only the footprint is recreated), or re-run `{{command}} init <agent>`, which cascades over every registered repo and repairs their footprints.

If the reported path is wrong rather than missing, the fix is the registry, not the footprint: correct that repo's `location` in the project's `config.yaml`.

# Removal

Removal is deliberately a manual edit: delete the repo's entry from the `repos` list in the project's `config.yaml`. No command is provided, and nothing in the repo itself needs cleaning up, since its footprint carries no project pointer. Confirm with the user before editing their config.
