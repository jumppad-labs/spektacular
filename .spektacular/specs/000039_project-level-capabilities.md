---
created_date: "2026-07-30"
status: completed
closed_date: "2026-07-30"
---

# Feature: 000039_project-level-capabilities

## Overview

Spektacular currently treats every codebase it works in as a single, self-contained project — one configuration, one place for specs and plans, one root directory. In practice, real work often spans more than one repository at once: a feature might require a code change in one repository and a documentation update in a companion repository. Today, Spektacular has no way to represent that a repository is part of a larger project alongside others, so it cannot help plan or carry out work that crosses repository boundaries.

This feature introduces a project-level layer above individual repositories. A project can register the repositories that belong to it, along with what each one does. When planning a piece of work, Spektacular can then look across all the registered repositories to figure out where each requirement actually belongs, and carry out the resulting changes in the right place — even when that means code in one repository and documentation in another. Teams working across multiple related repositories benefit by being able to plan and execute work as a single, coherent unit, rather than coordinating it manually across separate Spektacular setups.

## Requirements

- [ ] **Separate project-level and repository-level configuration**
  A project's settings (how it runs, where specs/plans/changelogs live, agent behavior) are stored independently from an individual repository's settings (where that repository's local knowledge lives). A single repository can still act as a whole project on its own, but its project-level and repository-level settings remain two distinct configurations.

- [ ] **Single point of execution for a project**
  Spektacular is always run against a project as a whole, never against an individual member repository in isolation. There is no requirement for Spektacular to search upward through parent directories to locate the project it belongs to.

- [ ] **Repository-level initialization is minimal**
  Initializing a repository on its own sets up only what that repository needs to participate in a project (its local configuration and knowledge storage) and leaves it with lightweight guidance pointing to the project it belongs to, without installing the full set of project-level tooling.

- [ ] **Project-level initialization sets up full capabilities**
  Initializing a project sets up everything needed to run Spektacular for the whole project — its configuration, its full agent guidance and capabilities, and its spec/plan/changelog storage — as an explicit, distinct action from initializing a single repository.

- [ ] **A project can register its member repositories**
  A project's configuration can list the repositories that belong to it, where each is located, and descriptive information about it (such as what it does, its role, what it depends on, and how it's deployed).

- [ ] **Repository registration supports multiple ways of accessing a repository**
  The mechanism for registering a repository is extensible to different ways of reaching that repository's files (for example, a repository already present on disk versus one that must be fetched from elsewhere), without changing how the rest of the project interacts with it.

- [ ] **Specs, plans, and changelogs remain centrally located**
  A project keeps a single, shared location for its specs, plans, and changelog, regardless of how many repositories are registered to it.

- [ ] **Planning can determine which repository a requirement belongs to**
  When working out how to satisfy a requirement, Spektacular can examine the repositories registered to a project and determine which repository (and which files within it) that requirement should be carried out against.

- [ ] **Work can be carried out in the correct repository**
  When a piece of work affects more than one registered repository (for example, a code change in one and a documentation update in another), Spektacular carries out each part of the change in the repository it actually belongs to.

- [ ] **Repositories can exclude paths from discovery**
  Each registered repository (and a project's own storage locations) can define paths that should be excluded when Spektacular searches or lists its contents.

## Constraints

- Repository registration must use the same provider mechanism that specs, plans, and changelogs already use, so the existing file read/write/search capabilities are reused rather than duplicated.
- All repositories must be accessible through the local filesystem for this release — no remote or network-based access is required or assumed.
- The exclusion (ignore) file must be named `.spektacular_ignore`, must follow the gitignore pattern convention, and must be rooted at the source it applies to — each source carries its own exclusion file rather than one global list.
- Exclusions must affect only listing and search operations; direct access to an explicitly named path must never be blocked by an exclusion rule.
- Project and repository configuration must remain two separate files, even when a single-repository project places them in the same directory.
- Backward compatibility with the existing single-file configuration is not required — this is a breaking change, and existing setups are expected to re-initialize.

## Acceptance Criteria

- [ ] **Project and repository configs are independently readable**
  A project's configuration file and a repository's configuration file can each be inspected on disk and contain only the settings scoped to that level; removing or changing one does not require changing the other.

- [ ] **Running Spektacular from inside a member repository without a project fails or is rejected**
  Attempting to run Spektacular against a directory that is only a registered repository, without its project context available, produces an explicit error rather than falling back to searching parent directories.

- [ ] **Repository-only initialization produces a minimal footprint**
  After initializing a directory as a repository (not a project), the resulting files include the repository's local configuration and knowledge storage, a short note pointing to the project root, and do not include the full agent guidance or tooling files that project initialization produces.

- [ ] **Project initialization produces the full footprint**
  After initializing a directory as a project, the resulting files include the project configuration, the full agent guidance and tooling files, and the spec/plan/changelog storage locations — all present in a single initialization action.

- [ ] **A registered repository appears in project configuration with its metadata**
  After a repository is registered to a project, the project's configuration reflects that repository's location and its descriptive information (what it does, its role, dependencies, deployment type), retrievable by inspecting the configuration.

- [ ] **A repository can be registered using more than one access method**
  The project configuration accepts at least two different ways of specifying how a registered repository's files are reached, and Spektacular operates correctly against a repository registered either way.

- [ ] **Specs, plans, and changelog entries land in one place regardless of repository count**
  Creating a spec, plan, or changelog entry in a project with multiple registered repositories results in that item being stored in the project's single shared location, not duplicated or split per repository.

- [ ] **A cross-repository requirement is attributed to the correct repository during planning**
  Given a project with multiple registered repositories and a requirement that clearly belongs to one of them (for example, "update the documentation"), the resulting plan identifies that repository and the relevant files as the target for that requirement.

- [ ] **A single unit of work produces changes in more than one repository when required**
  When carrying out a piece of work whose requirements span two registered repositories, the resulting changes are found in both repositories, each in the location appropriate to that repository, not only in the project's own directory.

- [ ] **Excluded paths are absent from listings and search but still directly accessible**
  A path marked for exclusion in a repository does not appear in directory listings or search results produced by Spektacular for that repository, but can still be read or written when referenced directly by its exact path.

## Technical Approach

- Model the repository list in project configuration on the existing knowledge-sources pattern — a list of entries, each pairing a provider with its provider-specific configuration — rather than the single-instance shape used by spec/plan/changelog sections.
- Keep each repository entry's descriptive metadata (description, tags, role, dependencies, deployment type) alongside the provider block as provider-agnostic fields, not nested inside provider configuration.
- For cross-repository file access, construct a file store rooted at the repository's configured path directly when needed — the configuration already carries everything required, so no new registry or lookup abstraction is warranted.
- During planning, have the discovery/architecture steps consult the registered repositories' metadata and search within each repository to attribute requirements to a repository and files; record that resolution in the plan's existing single project-level context document rather than introducing a new document type.
- The implementation-time changelog step should resolve which repository's root to write to, rather than assuming a single repository.
- Distinguish the two initialization modes with an explicit flag on the existing init command, with repository-level as the default and project-level opt-in.

## Success Metrics

- A multi-repository project (e.g. a code repo plus a docs repo) can go from spec through plan through implementation entirely within one Spektacular project, with no manual coordination between separate Spektacular setups.
- Plans produced for cross-repository features correctly attribute requirements to the right repository without the user having to correct the attribution.
- Cross-repository searches during planning stay useful — exclusion rules keep noise (build artifacts, dependency directories) out of results.
- Setting up a new project — including registering its repositories — takes minutes, not hours.

## Non-Goals

- The plan/spec viewer tool (rich rendering, inline review comments, a collaborative server mode) is out of scope and will be captured as a separate spec.
- Distinguishing local (private) plans from shared plans, and sharing plans via pull request, is deferred to follow-up work.
- Automatically detecting that completed work has invalidated other dependent plans, and triggering their update, is deferred to follow-up work.
- Optimizing how much planning context is loaded for large multi-repository projects (token efficiency) is a known concern deliberately parked for later.
- A home-directory-level registry of projects is explicitly excluded — readers of the originating issue (#25) might expect one, and it is not part of this design.
- Fully autonomous cross-repository execution (resolving dependency order, parallelizing work across repositories without supervision) is out of scope.
