---
created_date: "2026-08-13"
status: completed
closed_date: "2026-08-13"
---

# Feature: 000042_repo-self-describing-metadata

## Overview

Move the descriptive details of a member repository — what it is, what role it plays, how it is deployed — out of the project's configuration and into the repository's own configuration, so that each repository describes itself in exactly one place. Today every project that includes a repository must carry its own copy of that description, so the copies drift apart, and the problem compounds when a repository is shared by several projects or when several projects live inside one repository. With this change a project's registry only records which repositories belong to it and where to find them, while each repository remains the single source of truth about itself; anyone (or any agent) consulting the project gets consistent, up-to-date descriptions without duplication. The product documentation site will be updated to describe the new configuration layout.

## Requirements

- [x] **Repos describe themselves**
  A repository's descriptive metadata — description, role, tags, and deployment — is declared in the repository's own configuration, not in the project's configuration.

- [x] **Project registry holds membership only**
  A project's repo registry entry records only what is needed to identify and locate the repository (its project-local name, remote address and/or local path, and provider), plus project-scoped relationships.

- [x] **Dependencies stay project-scoped**
  Dependencies between repositories remain declared in the project's registry, since they reference other repositories by their project-local names and describe the project's topology rather than any single repository.

- [x] **Registering a repo stores metadata with the repo**
  When a user registers a repository with descriptive metadata, that metadata is written into the repository's own configuration as part of registration, not into the project's configuration.

- [x] **Listing merges both sources**
  Listing a project's repositories reports each repository's membership details from the project together with its descriptive metadata read from the repository's own configuration, without triggering any side effects such as cloning.

- [x] **Unmaterialized repos list without metadata**
  A registered repository that exists only as a remote address and has not yet been fetched locally is still listed with its membership details; its descriptive metadata is simply absent until the repository is available locally.

- [x] **Workflows read metadata from the repo**
  Any workflow that surfaces repository descriptions to users or agents (such as planning) obtains that metadata from the repository's own configuration.

- [x] **Shared repos stay consistent**
  When a repository belongs to more than one project, or several projects live inside one repository, every project observes the same descriptive metadata for it, because there is only one copy.

- [x] **Documentation reflects the new layout**
  The product documentation site's configuration reference describes the new split: descriptive metadata in the repository's own configuration, membership in the project's configuration.

## Constraints

- The repository's descriptive metadata must live in the existing repo-level configuration file, not in a new additional file.
- A repository's configuration must not carry a pointer back to any project; membership is recorded only on the project side, preserving the existing one-repo-many-projects design principle.
- Listing a project's repositories must remain free of side effects: it must never clone or fetch a repository to obtain metadata.
- No migration or backward-compatibility support is required for descriptive metadata previously held in project registry entries: the prior layout has not shipped in a release, so it can be removed outright.

## Acceptance Criteria

- [x] **Metadata lands in the repo's config**
  After registering a repository with a description, role, tags, and deployment, those values are present in the repository's own configuration file and absent from the project's configuration file.

- [x] **Project entry is membership-only**
  Inspecting the project's configuration after registration shows the repository entry contains only its name, address and/or local path, provider, and any dependencies — no description, role, tags, or deployment.

- [x] **Listing shows merged view**
  Listing the project's repositories shows, for each locally available repository, both its membership details and the descriptive metadata that lives in the repository's own configuration.

- [x] **Listing is still side-effect-free**
  Listing repositories when a registered repository exists only as a remote address completes successfully, shows that entry without descriptive metadata, and does not clone or fetch anything.

- [x] **Metadata appears once materialized**
  After a previously address-only repository is fetched locally, listing the project's repositories includes that repository's descriptive metadata without re-registering it.

- [x] **Two projects, one description**
  With two projects registering the same repository, editing the repository's description in its own configuration changes what both projects report when listing, with no per-project edit.

- [x] **Planning surfaces repo-owned metadata**
  During a plan workflow in a multi-repo project, the repository information presented to the agent contains the description, role, tags, and deployment read from each repository's own configuration.

- [x] **Dependencies unaffected**
  A dependency declared between two registered repositories is still recorded in the project's configuration and still appears in the repository listing after the change.

- [x] **Docs describe the split**
  The documentation site's configuration reference documents descriptive metadata under the repository configuration file and membership fields under the project configuration file, with neither page still showing descriptive metadata as project registry fields.

## Technical Approach

- The repo-level configuration file is created as part of a repo's footprint; registration should write the metadata into the target repo's footprint as part of the same step that already creates or repairs that footprint.
- Listing should merge the two sources per entry: membership from the project config, description from the repo's own config when the repo is present on disk; treat a missing or unreadable repo config as "no metadata" rather than an error, consistent with the unmaterialized-repo behaviour.
- The documentation changes land in the docs member repo already registered with this project (the configuration reference pages for the project and repo config files).

## Success Metrics

- A repository's description exists in exactly one place: searching all project configurations for descriptive metadata fields finds none.
- Agents working across projects that share a repository never receive conflicting descriptions of it.
- Users setting up a second project over an already-described repository do not need to re-enter any of its metadata.

## Non-Goals

- Fetching metadata for unmaterialized repositories over the network (e.g. reading the repo config from the remote without cloning) is out of scope — this applies to every command, not just listing.
- No new descriptive metadata fields are introduced; this change only relocates the existing ones.
- Restructuring the docs site beyond the configuration reference updates is out of scope.
