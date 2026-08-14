---
created_date: "2026-07-30"
status: completed
closed_date: "2026-07-30"
---

# Feature: 000039_project-level-capabilities

## Overview

Spektacular currently treats every codebase it works in as a single, self-contained unit — one configuration, one place for specs and plans, one root directory. In practice, real work often spans more than one repository at once: a feature might require a code change in one repository and a documentation update in a companion repository. Today, Spektacular has no way to represent that relationship, so it cannot help plan or carry out work that crosses repository boundaries.

This feature introduces two distinct concepts with precise meanings used throughout this spec:

- A **project** is a collection of code repositories, with its own configuration, identity, and shared storage for specs, plans, and changelogs. Spektacular always runs against a project.
- A **repo** is an individual code repository. A repo participates in a project by being registered to it, and carries only its own repo-scoped configuration (knowledge storage, changelog provider).

A project registers the repos that belong to it, along with what each one does. When planning a piece of work, Spektacular looks across all registered repos to figure out where each requirement actually belongs, and carries out the resulting changes in the right place — even when that means code in one repo and documentation in another.

A project may be colocated with a single repo (the solo-repo and monorepo cases — a solo repo is simply a project of one), and a repo may belong to more than one project at the same time. Teams working across multiple related repositories benefit by being able to plan and execute work as a single, coherent unit, rather than coordinating it manually across separate Spektacular setups.

## Requirements

- [x] **Separate project-level and repository-level configuration, composed without duplication**
  A project's settings (identity, member repos, where specs/plans/changelogs live, agent behavior) are stored independently from an individual repo's settings (its knowledge storage and changelog provider). The project config registers repos and loads each repo's own config from the repo itself; repo-scoped concerns are declared only in the repo's config, and Spektacular honors that repo's config when writing to it. Nothing is duplicated between the two levels.

- [x] **A project has an identity**
  Project configuration carries a required `name` — stable and slug-safe, since it is used as a directory name — and an optional `source` (the project's git remote URL), used when referencing the project from outside its own directory.

- [x] **Single point of execution for a project**
  Spektacular is always run against a project as a whole, never against an individual repo in isolation. There is no searching upward through parent directories to locate a project: running Spektacular where no project is configured produces an explicit error.

- [x] **The repo footprint is minimal and knows nothing of its projects**
  A repo's footprint — created by `repo add`, not by a user-facing repo-level init command — consists of the repo's config file (with knowledge and changelog sections) and its knowledge storage. It contains no pointer to any project: membership is recorded only in project configs, so a repo can belong to multiple projects without carrying stale membership state.

- [x] **Project-level initialization sets up full capabilities and is idempotent**
  Initializing a project sets up everything needed to run Spektacular for the whole project — its configuration, its full agent guidance and capabilities, and its spec/plan/changelog storage. Init is safely re-runnable: a second run produces no changes, and re-running it cascades over registered repos, creating or repairing their footprints as needed.

- [x] **A project can register its member repos**
  A project's configuration lists the repos that belong to it. Each entry carries a required slug-safe `name` (plans and changelog attribution reference repos by name), an `address` (remote URL), a `local` path, and descriptive metadata (what it does, its role, what it depends on, how it's deployed). At least one of `address`/`local` is required; when both are set, `local` wins and `address` serves as provenance metadata. The same repo may be registered to multiple projects. Registration always targets a repo, never a project.

- [x] **Repo registration is provider-based and supports two access methods**
  Repo entries pair a provider with provider-specific configuration, modeled on the knowledge-sources pattern, so the mechanism is extensible without changing how the rest of the project interacts with a repo. Two access methods ship in this release: a repo already present on disk (via `local`), and a repo materialized by cloning from its `address`. Both resolve to a local directory.

- [x] **Repos can be materialized automatically**
  A registered repo with an `address` but no usable `local` path is automatically cloned into a working folder under the project (`.spektacular/repos/<repo-name>/`). Cloning happens only when the repo is absent — Spektacular never fetches or pulls on its own. When a previously cloned repo is used, Spektacular checks whether it is stale relative to its remote and warns the user, but takes no action itself.

- [x] **Specs, plans, and changelogs remain centrally located, with derived per-repo changelog entries**
  A project keeps a single, shared location for its specs, plans, and changelog, regardless of how many repos are registered to it. In addition, each repo affected by a piece of work receives a changelog entry in its own changelog store, derived automatically from the central entry and scoped to only the changes made to that repo, written via that repo's configured changelog provider. Each derived entry carries the project's name, the project's source (when set), and the spec and plan identifiers as structured data, plus a human-readable reference line in the body.

- [x] **Changelog stores are namespaced by project**
  Every changelog store — the project's central store and each repo's store — organizes entries as `changelog/<project-name>/<spec-id>_<slug>.md`. A repo touched by multiple projects accumulates entries in separate per-project folders with no possibility of filename collision; a project's own central store simply contains a single folder bearing its own name.

- [x] **Planning can determine which repo a requirement belongs to**
  When working out how to satisfy a requirement, Spektacular can examine the repos registered to a project — their metadata and contents — and determine which repo (and which files within it) that requirement should be carried out against.

- [x] **Work can be carried out in the correct repo**
  When a piece of work affects more than one registered repo (for example, a code change in one and a documentation update in another), Spektacular carries out each part of the change in the repo it actually belongs to.

- [x] **Registered repos are validated on use, with repair offered**
  Whenever Spektacular touches a registered repo, it first checks that the repo's config is present and valid. If it is missing or broken, Spektacular surfaces the problem and offers to fix it (re-creating the footprint) rather than hard-failing.

- [x] **The CLI manages the repo lifecycle**
  A `repo add` command registers a repo in the project config and creates its footprint in one operation. It is idempotent: adding a repo already initialized by another project registers it without disturbing its existing footprint. A `repo list` command reports the project's registered repos. Removal is performed by editing the project config directly — no command is provided.

- [x] **Agents are given skills for repo management and cross-repo work**
  New skill(s) document how an agent drives the repo-management CLI on the user's behalf, in the style of the existing skills. The existing planning and implementation skills are updated for cross-repo behavior: attributing requirements to repos during planning, and writing derived per-repo changelog entries during implementation.

- [x] **Repos can exclude paths from discovery**
  Each registered repo (and a project's own storage locations) can define paths that should be excluded when Spektacular searches or lists its contents.

## Constraints

- Repo registration must use the same provider mechanism that specs, plans, and changelogs already use, so the existing file read/write/search capabilities are reused rather than duplicated.
- **Spektacular mediates store documents, never source files.** Specs, plans, changelogs, and knowledge go through the CLI because providers own their storage. For registered repos, Spektacular's job is resolution and discovery — providing the repo's local root path, its metadata, and its exclusion rules — after which agents use their own native file tools directly against that path. Repo providers must resolve a repository to a local directory (a provider for repos not already on disk must materialize them locally first); they must never proxy per-file access.
- All code must be accessible on the local filesystem — this is a hard boundary, not a temporary limitation of this release. Remote access is only ever "materialize locally, then hand back a path".
- Cloned repos must not become git submodules: materialization uses a plain clone, never `git submodule add`, and project init ensures the working folder (`.spektacular/repos/`) is listed in the project's `.gitignore` so a commit to the project repo can never commit or gitlink a member repo.
- A project cannot be a member of a project — there is no recursion. Registering a member repo never exposes that repo's own project-level world (its specs, plans, member list, or central changelog), even if the repo happens to also be a standalone project.
- The project `name` must be filesystem-safe, since it is used as a changelog directory name.
- There is exactly one initialization command, and it operates at project level. Repo footprints are created only by `repo add` or by the project init cascade — there is no repo-level init command.
- The exclusion (ignore) file must be named `.spektacular_ignore`, must follow the gitignore pattern convention, and must be rooted at the source it applies to — each source carries its own exclusion file rather than one global list.
- Exclusions must affect only Spektacular's own listing and search operations; direct access to an explicitly named path must never be blocked by an exclusion rule, and agents' native file tools are not bound by it.
- Project and repo configuration must remain two separate files, even when a colocated single-repo project places them in the same directory.
- Backward compatibility with the existing single-file configuration is not required — this is a breaking change, and existing setups are expected to re-initialize.

## Acceptance Criteria

- [x] **Project and repo configs are independently readable**
  A project's configuration file and a repo's configuration file can each be inspected on disk and contain only the settings scoped to that level; removing or changing one does not require changing the other.

- [x] **Running Spektacular without a project fails explicitly**
  Attempting to run Spektacular in a directory with no project configured — including inside a registered repo whose project context is not available — produces an explicit error rather than falling back to searching parent directories.

- [x] **`repo add` produces the minimal footprint**
  After `repo add`, the target repo contains its config file (with knowledge and changelog sections) and knowledge storage — and nothing else: no pointer to the project, no agent guidance or tooling files.

- [x] **Project initialization produces the full footprint and is idempotent**
  After initializing a directory as a project, the resulting files include the project configuration (with its `name`), the full agent guidance and tooling files, and the spec/plan/changelog storage locations. Running init a second time produces no changes; running it after a member repo's footprint has been broken repairs that footprint.

- [x] **A registered repo appears in project configuration with its metadata**
  After a repo is registered, the project's configuration reflects its `name`, its `address` and/or `local` location, and its descriptive metadata (what it does, its role, dependencies, deployment type), retrievable by inspecting the configuration.

- [x] **Registration is idempotent across projects**
  Adding a repo that is already a member of another project registers it in this project's config without altering the repo's existing footprint.

- [x] **Both access methods work end-to-end**
  A repo registered with a `local` path and a repo registered with only an `address` (materialized by cloning into `.spektacular/repos/<repo-name>/`) both operate correctly for planning and implementation. Nothing outside the provider layer assumes how a repo's files came to be on disk.

- [x] **Cloning is absent-only, with a staleness warning**
  A repo with an `address` is cloned only when not already present. Using a previously cloned repo that is behind its remote produces a warning to the user; Spektacular performs no fetch or pull of its own.

- [x] **Cloned repos never leak into the project's git history**
  After materializing a repo under `.spektacular/repos/`, the project's `.gitignore` covers the working folder, `git status` in the project shows no new tracked or gitlinked entries, and a commit made in the project repo contains nothing from the member repo.

- [x] **Cross-repo work produces central and derived changelog entries**
  After implementing work that touches repos R1 and R2: the project's central changelog contains the entry under the project's namespace folder, and R1 and R2 each contain a derived entry under that same project-named folder in their own changelog stores, each listing only that repo's changes, each carrying the project name, project source (when set), and spec/plan identifiers as structured data plus a human-readable reference line.

- [x] **Multiple projects touching one repo cannot collide**
  Two projects that each implement work against the same repo produce derived entries in separate project-named folders within that repo's changelog store, with no filename collision and no ambiguity about which project's spec an entry refers to.

- [x] **A broken repo config is surfaced with an offer to repair**
  Touching a registered repo whose config file is missing or invalid results in a clear report of the problem and an offer to fix it, not a hard failure and not silent continuation.

- [x] **A cross-repo requirement is attributed to the correct repo during planning**
  Given a project with multiple registered repos and a requirement that clearly belongs to one of them (for example, "update the documentation"), the resulting plan identifies that repo and the relevant files as the target for that requirement.

- [x] **A single unit of work produces changes in more than one repo when required**
  When carrying out a piece of work whose requirements span two registered repos, the resulting changes are found in both repos, each in the location appropriate to that repo, not only in the project's own directory.

- [x] **Excluded paths are absent from listings and search but still directly accessible**
  A path marked for exclusion in a repo does not appear in directory listings or search results produced by Spektacular for that repo, but can still be read or written when referenced directly by its exact path.

## Technical Approach

- Model the repo list in project configuration on the existing knowledge-sources pattern — a list of entries, each pairing a provider with its provider-specific configuration — rather than the single-instance shape used by spec/plan/changelog sections.
- Keep each repo entry's identity and descriptive metadata (`name`, `address`, `local`, description, tags, role, dependencies, deployment type) alongside the provider block as provider-agnostic fields, not nested inside provider configuration.
- Add `name` (required) and `source` (optional) fields to project configuration; validate `name` as filesystem-safe at load time.
- For cross-repo file access, construct a file store rooted at the repo's resolved local path directly when needed — the configuration already carries everything required, so no new registry or lookup abstraction is warranted. Resolution order: use `local` when set and present; otherwise clone `address` into `.spektacular/repos/<repo-name>/` (creating the `.gitignore` entry if missing) and use that path.
- During planning, have the discovery/architecture steps consult the registered repos' metadata and search within each repo to attribute requirements to a repo and files; record that resolution in the plan's existing single project-level context document rather than introducing a new document type.
- The implementation-time changelog step writes the central entry under the project's namespace folder, then derives one entry per affected repo — filtered to that repo's changes, carrying project name/source and spec/plan identifiers as structured front matter plus a reference line in the body — and writes each via that repo's configured changelog provider.
- There is a single project-level init command, idempotent and re-runnable, which also cascades over registered repos to create or repair their footprints; `repo add` performs registration plus footprint creation for one repo. No init flag distinguishes modes — repo-level init does not exist as a user-facing action.

## Success Metrics

- A multi-repo project (e.g. a code repo plus a docs repo) can go from spec through plan through implementation entirely within one Spektacular project, with no manual coordination between separate Spektacular setups.
- Plans produced for cross-repo features correctly attribute requirements to the right repo without the user having to correct the attribution.
- A reader of a repo's changelog alone can unambiguously identify, for every entry, which project and which spec/plan produced it — even when multiple projects write to that repo.
- Cross-repo searches during planning stay useful — exclusion rules keep noise (build artifacts, dependency directories) out of results.
- Setting up a new project — including registering its repos — takes minutes, not hours.

## Non-Goals

- The plan/spec viewer tool (rich rendering, inline review comments, a collaborative server mode) is out of scope and will be captured as a separate spec.
- Machine-resolvable cross-install spec/plan links, and any tooling that follows changelog references, are deferred — the derived entries record project name, source, and spec/plan identifiers as structured data precisely so such tooling can be built later without migrating old entries, but nothing in this release follows those references.
- Recursive projects — a project as a member of another project — are explicitly excluded.
- Automatic cleanup of a repo's derived changelog entries when a project deregisters that repo is excluded; no automated cleanup of any kind accompanies removal, which is itself a manual config edit.
- Automatic synchronization of cloned repos (fetch/pull) is excluded — materialization is clone-if-absent, plus a staleness warning; keeping clones current is the user's responsibility.
- Distinguishing local (private) plans from shared plans, and sharing plans via pull request, is deferred to follow-up work.
- Automatically detecting that completed work has invalidated other dependent plans, and triggering their update, is deferred to follow-up work.
- Optimizing how much planning context is loaded for large multi-repo projects (token efficiency) is a known concern deliberately parked for later.
- A home-directory-level registry of projects is explicitly excluded — readers of the originating issue (#25) might expect one, and it is not part of this design.
- Fully autonomous cross-repo execution (resolving dependency order, parallelizing work across repos without supervision) is out of scope.
