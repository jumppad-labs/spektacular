---
created_date: "2026-08-14"
status: completed
closed_date: "2026-08-14"
---

# Changelog: 000042_repo-self-describing-metadata

## What was built

A repository now describes itself in one place — its own `repo.yaml` — rather than every project that registers it repeating the same description in its own `config.yaml`. Four descriptive fields moved out of the project registry entirely and into each repository's own configuration:

- `internal/config/repo.go` grew `Description`, `Role`, `Tags`, and `Deployment` on `config.RepoConfig` (all `omitempty`, matching the tag style already used on `config.RepoEntry`).
- `runRepoAdd` (`cmd/repo.go`) now, on every registration, writes the input's descriptive fields into the target repository's own `repo.yaml` after `EnsureFootprint` returns — and updates an already-footprinted repo on re-registration, not only on first creation. Fields left unset in `--data` are preserved rather than blanked (a bug caught mid-phase by that acceptance criterion and fixed within the same phase, not deferred).
- The same four fields were removed from `config.RepoEntry` (`internal/config/config.go`), leaving the project registry membership-only (name, address, local, dependencies, provider, config). `reposEqual` and the `RepoEntry` construction in `runRepoAdd` were updated to match; `validateRepos` needed no change.
- A new helper, `(*repo.Set) DescriptiveMetadata(name string)`, returns a registered repository's own descriptive metadata when it is on disk and its `repo.yaml` is readable, and an absent result — never an error — otherwise. It mirrors how `checkFootprint` already tolerates a missing or broken `repo.yaml`. It never clones or fetches.

Enforcement runs through existing tests in `internal/config/config_test.go`, `internal/config/repo_test.go`, `cmd/repo_test.go`, and `internal/repo/set_test.go` (the last covering the new helper directly).

## Why it matters / what it enables

Before this change, a repository shared across two projects was described twice, and the copies drifted apart the moment either project's registration was edited independently. Users and agents working across those projects saw whichever description happened to be in the config they read first, with no signal that another version disagreed.

Now the description belongs to the repository, not to any single project that registers it. Registering a repository under a second project requires no re-entry of its metadata, cross-project tooling can pull a consistent description from one authoritative place, and the project registry stays small and focused on what it uniquely needs (membership and project-scoped dependencies between repositories).

## Deviations from the plan

None materially. During Phase 1.2's verify pass a self-corrected implementation bug was found and fixed within the same phase (the initial unconditional-overwrite would blank existing metadata on re-registration when a field was omitted from `--data`), rather than being deferred. Every phase's stated acceptance criteria were met; no scope was moved between phases and no plan text needed post-hoc amendment.

The equivalent unconditional-overwrite pattern still existed on the project-side `RepoEntry` upsert path but was intentionally left as-is, since Phase 2.1 removes those fields from `RepoEntry` entirely rather than needing a parallel fix. This repository's own `.spektacular/config.yaml` still carries the removed descriptive fields on its `docs` entry; the config loader uses non-strict `yaml.Unmarshal`, so the now-unknown keys are silently ignored and no migration was performed as part of this plan.
