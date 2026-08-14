---
created_date: "2026-08-13"
status: completed
closed_date: "2026-08-13"
---

# Plan: 000042_repo-self-describing-metadata

<!-- Metadata -->
<!-- Created: 2026-08-13T15:28:12Z -->
<!-- Commit: 3ccbbd2 -->
<!-- Branch: main -->
<!-- Repository: git@github.com:jumppad-labs/spektacular.git -->

## Overview

Today, a project's `repos:` registry entries carry both membership fields
(name, address, local path, provider) and descriptive fields (description,
role, tags, deployment), so every project that registers a repository
duplicates that repository's own description, and the copies drift apart
when a repository is shared across projects. This plan moves the
descriptive fields out of the project registry entirely and into each
repository's own configuration file, so a repository describes itself in
exactly one place; the project registry keeps only what it needs to
identify and locate its member repositories, plus project-scoped
dependencies between them. Users and agents working across multiple
projects that share a repository get a consistent description with no
per-project editing, and registering a repository a second time under a
different project requires no re-entry of its metadata.

## Conventions

- **MDX authoring conventions (ConfigKey pattern, no layout HTML in page bodies, blank lines around slot content, fenced code blocks)** — this plan creates a new docs page (repo-level config reference) and edits an existing one (configuration.mdx); both must follow the established `ConfigKey`/`ConfigurationKeys` component pattern rather than ad hoc markup, per docs:.spektacular/knowledge/conventions/mdx-authoring.md.
- **Plans must sketch content structure, not just summarize it** — the phase(s) covering the new repo-config reference page and the trimmed configuration.mdx must include a concrete Content outline/example with headings and illustrative ConfigKey usage, not just a prose summary, since this plan introduces genuinely new page content.
- **No em dashes** — applies to all authored prose in this plan's own content (plan.md, context.md, research.md) and to any new documentation copy drafted for the docs site pages.
- **Alternate section background shading** — does not apply: this plan's docs work is limited to `ConfigurationKeys`/`ConfigKey` content inside existing or new reference pages, not adding new top-level page sections with a `surface` prop; no new `Section`-style component is introduced.
- **Prefer DRY refactors over accepting duplication** (feedback memory, not a knowledge-base convention, but load-bearing for this plan) — directly drove the architecture's choice to extract one shared repo-metadata-merge helper in `internal/repo` rather than duplicating the "read RepoConfig if materialized, else absent" logic across `cmd/repo.go` and `cmd/plan.go`.

## Architecture & Design Decisions

The chosen shape moves `Description`, `Role`, `Tags`, and `Deployment` off
`config.RepoEntry` (spektacular:internal/config/config.go:126-137) and onto
`config.RepoConfig` (spektacular:internal/config/repo.go:19-22), the
existing repo-level config type already backing each repo's own
`.spektacular/repo.yaml`. `Dependencies` stays on `RepoEntry` because it
names other repos by project-local registry keys, which are meaningless
outside the project that assigned them; a repo cannot describe its own
project-scoped dependencies since the same repo may be registered under
different names in different projects. This gives the project registry
exactly the membership shape the spec calls for (name, address/local,
provider, dependencies) with zero descriptive fields left duplicated
anywhere, satisfying the "repos describe themselves" and "project registry
holds membership only" requirements directly from the struct definitions
rather than from convention.

Three write/read surfaces change to match: `runRepoAdd`
(spektacular:cmd/repo.go:113-203) stops writing descriptive input into the
project config and instead folds it into the `RepoConfig` passed to the
repo's footprint. Because `EnsureFootprint`
(spektacular:internal/repo/footprint.go:28-88) treats an already-present
`repo.yaml` as authoritative over the config passed in (footprint.go:50),
a bare `EnsureFootprint` call is not sufficient for the "re-add updates
metadata" case (documented today by
`TestRepoAdd_UpsertUpdatesMetadataWithoutTouchingFootprint`,
cmd/repo_test.go:343-375, which will invert). `runRepoAdd` therefore calls
`EnsureFootprint` first to guarantee the footprint exists (create/repair
path unchanged), then separately loads the resulting `repo.yaml`, applies
the input's descriptive fields onto it, and writes it back only when they
differ, mirroring the existing idempotent-upsert behavior `reposEqual`
(cmd/repo.go:279-297) already provides on the project side today. This
keeps footprint creation/repair semantics untouched for every other caller
(notably `internal/project/init.go`'s cascade, which continues to pass a
metadata-free default and must keep working unchanged, since the spec
requires no migration).

`runRepoList` (spektacular:cmd/repo.go:205-259) and `repoRoster`
(spektacular:cmd/plan.go:144-163) both need the same merge: membership from
the project registry, descriptive fields from the target repo's own
`repo.yaml` when the repo is materialized locally, absent otherwise. Both
already have (or need) the same materialization gate
(`internal/repo.Set.Present`, internal/repo/set.go:74-110) that `repo list`
uses today to decide whether to resolve further. Rather than duplicate the
"read RepoConfig if present, else return zero-value metadata, treating a
missing or unreadable file as absence rather than an error" logic in two
places, it is extracted once as a small helper in `internal/repo` (the
package that already owns `Set`/`EnsureFootprint`/materialization checks)
and called from both `cmd/repo.go` and `cmd/plan.go`. This directly
satisfies "listing merges both sources," "unmaterialized repos list without
metadata," and "workflows read metadata from the repo," and because the
helper is a pure read with no fallback to cloning or fetching, it preserves
the side-effect-free listing constraint by construction rather than by
convention at each call site.

This direction was chosen over two alternatives (see
research.md#alternatives-considered-and-rejected for full evidence): a
narrower option that relied on `EnsureFootprint` alone to carry metadata
updates was rejected because its existing-wins semantics silently drop
updates to an already-footprinted repo, which would violate the "two
projects, one description... with no per-project edit" acceptance
criterion the moment a user edits metadata via `repo add` a second time. A
broader option that mirrored the knowledge-sources project/repo split in
full (an optional project-level override layer aggregated with the
repo-level declaration at read time) was rejected as over-engineered: unlike
knowledge sources, which legitimately have project-owned entries with no
repo-level equivalent, this spec explicitly leaves the project registry
with no descriptive fields at all, so there is nothing to aggregate. The
repo-level file is simply the sole source, which the chosen direction
already gives it without extra machinery. **Convention applied**: [Don't
add features, refactor, or introduce abstractions beyond what the task
requires](CLAUDE.md project-instruction) rules out the aggregation option
on the same grounds inside plan.md's own architecture reasoning, not only
in research.md's alternatives log.

Documentation work lands entirely in the `docs` repo
(spektacular-website), since no repo-level config reference page exists
today (confirmed by research: a full-corpus search of `src/pages/` found
zero references to `repos:`, `dependencies`, or any of the four descriptive
fields). The existing `configuration.mdx` page is trimmed to describe only
the project-side membership shape; a new page documents `repo.yaml`
(knowledge, changelog, and the four newly-added descriptive fields),
following the `ConfigKey`/`ConfigurationKeys` component pattern already
used throughout `configuration.mdx` (docs:src/components/sections/ConfigKey.astro,
usage example at docs:src/pages/configuration.mdx:78-93) and added as a
fourth entry in the "Resources" nav dropdown (docs:src/components/Nav.astro:6-20),
consistent with how "Configuration," "Extending," and "Debugging" are
already grouped there.

## Component Breakdown

**Repo-level config schema (changed)** — owns the shape of a repository's
own configuration file. Gains the four descriptive fields (description,
role, tags, deployment) alongside its existing knowledge and changelog
sections, plus validation and defaulting for the new fields. This becomes
the single source of truth for what a repository is, replacing the
project-side copy.

**Project registry schema (changed)** — owns the shape of a project's repo
membership entries. Loses the four descriptive fields entirely, retaining
only identity (name, address, local path, provider) and project-scoped
dependencies. Its existing validation continues to apply only to the
fields that remain.

**Repo registration (changed)** — owns turning a `repo add` invocation into
on-disk state. Continues to own writing the project's membership entry
unchanged in shape (minus the fields it no longer accepts for that
purpose), and gains responsibility for writing descriptive input into the
target repo's own config as part of the same registration step, updating
an already-footprinted repo's descriptive fields on re-registration rather
than only on first creation.

**Repo footprint management (unchanged in responsibility, extended in
scope)** — continues to own creating and repairing a repo's on-disk
`.spektacular/` structure. Its create/repair behavior for callers that pass
no descriptive metadata (such as project initialization's cascade over
already-registered repos) is unaffected; registration layers its own
update step on top rather than changing footprint semantics.

**Repo metadata resolution (new)** — owns answering "what is this repo's
descriptive metadata, if any" for a given registered repo: checks whether
the repo is materialized locally, and if so reads its own config, treating
absence or an unreadable config the same way as a repo that has not yet
been fetched. Consumed by both repo listing and the plan workflow's repo
roster, so that both present the same merged view and neither duplicates
the read-and-merge logic.

**Repo listing (changed)** — owns reporting a project's registered repos
to the user. Continues to own membership and materialization reporting,
and now delegates to repo metadata resolution for descriptive fields
instead of reading them from the project registry, without introducing any
new side effects such as cloning.

**Plan workflow repo roster (changed)** — owns describing the project's
repos to planning agents at the start of discovery and architecture steps.
Continues to own project-scoped identity (name) and now delegates to the
same repo metadata resolution component as listing, so the roster an agent
sees during planning matches what `repo list` reports for the same repo.

**Documentation reference pages (changed and new, docs repo)** — the
existing project configuration reference page owns describing the
project's config file and is trimmed to drop the descriptive fields it no
longer documents; a new repo configuration reference page owns describing
a repo's own config file, including the newly added descriptive fields
alongside its existing knowledge and changelog documentation. Both reuse
the site's existing config-reference component pattern rather than
introducing new page-layout components.

## Data Structures & Interfaces

**`RepoConfig`** (repo-level config type) gains four fields alongside its
existing `Knowledge` and `Changelog` sections:

```go
type RepoConfig struct {
    Description string   // what the repo is
    Role        string   // the role it plays
    Tags        []string
    Deployment  string   // how it is deployed
    Knowledge   KnowledgeConfig
    Changelog   ChangelogConfig
}
```

This is the sole on-disk representation of a repository's descriptive
identity; nothing else duplicates these fields.

**`RepoEntry`** (project registry entry type) loses the four fields above.
Its resulting contract is membership-only:

```go
type RepoEntry struct {
    Name         string
    Address      string
    Local        string
    Provider     string
    Dependencies []string
    Config       GitRepoConfig
}
```

**Repo metadata resolution** is a new read-only contract: given a
registered repo's identity, return its descriptive metadata if the repo is
materialized locally and its own config is present and parseable,
otherwise return an empty/absent result. It never triggers cloning or
fetching. This is the single function both `repo list` and the plan
workflow's repo roster call, so their view of a repo's descriptive
metadata cannot diverge.

**Repo listing output** (the shape `repo list` reports per entry) is
unchanged from the caller's perspective: each entry still carries identity
and materialization fields plus description/role/tags/deployment when
available. What changes is only where those descriptive values are
sourced from internally (via repo metadata resolution instead of directly
off the registry entry); no new fields are added to the output shape.

**Plan workflow repo roster** (the data handed to discovery/architecture
templates) is unchanged in shape (name, description, role, tags,
deployment per repo) for the same reason: only its internal source
changes.

No new persisted file formats, network protocols, or external interfaces
are introduced. All changes are to the field sets of two existing
configuration types and to which internal function produces a given
value, not to any wire format or external contract.

## Implementation Detail

The core pattern this plan introduces is a **single resolver, two
consumers**: rather than each caller that needs a repo's descriptive
metadata deciding independently how to get it, one small resolution
function becomes the only place that knows the rule ("materialized and
readable → read it; otherwise, absent, never fetch"). Repo listing and the
plan workflow's repo roster both become thin callers of that resolver
instead of each re-implementing the materialization check and the
config-read fallback. This is a straightforward extraction rather than a
new architectural layer: the codebase already has a materialization check
and a repo-config loader; this plan's only new code is the small function
that composes them and defines the "absence, not error" fallback once.

Registration gains a **two-phase write** where it previously had one.
Today, registering a repo with descriptive metadata is a single write to
the project config. After this change, registration still performs that
project-config write (now carrying a smaller set of fields), and
separately performs a write to the target repo's own config. The two
writes are not transactional with each other, matching how footprint
creation already happens as a distinct step after the project-config
write today; a developer reading the registration code will see the same
overall shape as before (validate input, write project entry, ensure
target footprint) with one more step folded into the footprint stage
rather than a restructured flow.

Within that footprint stage, this plan introduces a **create-then-update**
sequence where today there is only create-or-repair. The existing
footprint step continues to guarantee the repo's config file exists and is
well-formed, unchanged. A new step follows it: load the (now guaranteed
well-formed) config, apply the registration input's descriptive fields to
it, and write it back only if something changed. This keeps the existing
footprint logic untouched (still the only thing responsible for deciding
whether a file needs to be created from scratch or repaired from
corruption) and adds a narrow, separate responsibility (updating specific
fields on a known-good file) rather than teaching the existing step two
jobs at once.

On the documentation side, the pattern is **split by ownership, not by
file**: the two config reference pages are being reshaped to each
describe exactly one config file's current contents, mirroring the
code-level split. A developer or user reading either page afterward sees
a page whose scope matches a real file on disk, rather than a page that
mixes fields from two different files under one heading as the current
project-config page does today for the repo entries it (currently, still)
documents in outline.

No existing interface is being replaced across the codebase, no package is
being split, and no new orchestration or state-machine shape is
introduced; this plan is a field relocation plus one small shared read
helper, deliberately scoped to avoid restructuring anything the spec did
not ask to change.

## Dependencies

- **`internal/config`** — provides both `RepoEntry` and `RepoConfig`, the
  two types this plan directly modifies. No external change needed; this
  plan changes it directly.
- **`internal/repo`** — provides the existing footprint creation/repair
  logic and the materialization/resolution checks this plan builds its new
  metadata-resolution helper on top of. No external change needed; this
  plan extends it directly.
- **`cmd` package (`repo.go`, `plan.go`)** — the CLI command layer this
  plan changes (`repo add`, `repo list`) and the plan-workflow command that
  builds the repo roster (`plan new`/`plan goto`). No external change
  needed; this plan changes it directly.
- **`internal/steps/plan`** — consumes the repo roster produced by
  `cmd/plan.go` and renders it into workflow templates. Its own step logic
  is unaffected (confirmed in research: its existing tests feed a synthetic
  roster directly into the step functions, bypassing where the data comes
  from), so no change is required here, only downstream-shape stability
  from the `cmd` layer.
- **Spec 000039 (project-level capabilities)** — the prior, already-shipped
  spec and plan that introduced `RepoEntry`'s descriptive fields and the
  repo footprint concept this plan builds on. Fully landed; no further
  change needed from it, but its design record (consulted during research)
  is the basis for this plan's "no migration needed" assumption and its
  "footprint knows nothing of its projects" constraint.
- **`gopkg.in/yaml.v3`** (external library, already a project dependency) —
  used by both `Config.ToYAMLFile`/`FromYAMLFile` and
  `RepoConfig.ToYAMLFile`/`RepoConfigFromYAMLFile` for marshaling. No
  version change or new capability needed; this plan uses it exactly as
  the existing loaders already do.
- **docs repo (spektacular-website)** — the site whose configuration
  reference pages this plan updates. Must be checked out locally (as it
  already is, resolved via `go run . repo list` to
  `../spektacular-website`) for the documentation phase; no CLI-side change
  is a prerequisite for docs work, since the docs phase can describe the
  target end-state independently of the code's landing order, though
  landing the code first is preferred so the docs can be verified against
  real command output rather than only the design.
- **Astro 5 / `@astrojs/mdx` / `astro-expressive-code`** (external,
  already docs-repo dependencies) — the toolchain the new/edited MDX pages
  build under. No version change needed.

No dependency needs to land or change before this plan starts; everything
listed already exists in the codebase or is already a project dependency.

## Testing Approach

Coverage is predominantly unit-level, matching the project's existing test
style for `internal/config`, `internal/repo`, and `cmd`: table-driven Go
tests against real temp-directory fixtures rather than mocks. The two
schema types (`RepoConfig`, `RepoEntry`) get the most direct coverage,
since they are the load-bearing contract change: round-trip and validation
tests confirm the four descriptive fields serialize correctly on
`RepoConfig` and are entirely absent from `RepoEntry`'s YAML output, which
is the most direct possible check of "repos describe themselves" and
"project registry holds membership only." The new repo-metadata-resolution
helper and its two callers (`repo add`, `repo list`) get integration-style
unit tests that exercise real footprint creation and real file reads in a
temp directory, since the side-effect-free and materialization-gated
behavior this plan depends on is only meaningfully verified against real
filesystem state, not mocked reads.

A smaller set of regression tests locks in the two behavior changes that
existing tests currently assert the *opposite* of: the re-add/update path
(today's `TestRepoAdd_UpsertUpdatesMetadataWithoutTouchingFootprint`
documents metadata updates landing in the project config and leaving the
footprint untouched; this plan's equivalent test asserts the inverse) and
the footprint-minimality test (today's assertion that a fresh footprint is
metadata-free is superseded by an assertion that it now carries whatever
descriptive metadata registration supplied). The plan workflow's own step
rendering tests are not expected to change, since they already exercise
the roster template contract against a synthetic roster independent of
where the roster's data comes from; coverage there is a deliberate gap,
justified because the roster's shape is untouched and the change is
entirely in `repoRoster`'s data source, which gets its own direct test
instead.

Documentation changes are verified by the site's existing build-time
guards (the MDX layout-HTML lint, `astro check`, and a successful
`npm run build`) rather than new automated content tests, consistent with
how the docs repo already treats prose content: structurally checked, not
content-asserted.

**Success metrics → verification:**

- *"A repository's description exists in exactly one place: searching all
  project configurations for descriptive metadata fields finds none."* —
  Behavioural test. A test registers a repo with full descriptive metadata,
  writes the project config, and asserts the marshaled project config YAML
  contains none of the four descriptive field keys, while the target
  repo's own config does contain them.
- *"Agents working across projects that share a repository never receive
  conflicting descriptions of it."* — Behavioural test. A test registers
  the same repo (by local path) into two separate project configs, edits
  the repo's own config description once, and asserts both projects'
  repo-metadata resolution (and, by extension, `repo list` and the plan
  roster) report the identical, updated description with no per-project
  edit.
- *"Users setting up a second project over an already-described repository
  do not need to re-enter any of its metadata."* — Behavioural test. A
  test registers a repo with metadata into project A, then registers the
  same repo (by local path, with no descriptive fields supplied) into a
  separate project B, and asserts `repo list` under project B reports the
  full descriptive metadata anyway, sourced from the repo's own already-populated
  config rather than from project B's registration input.

All three success metrics are covered by automated behavioural tests; none
require a manual test-plan entry.

## Milestones & Phases

### Milestone 1: A repository's own configuration can hold its description

**What changes**: When a user registers a repository with a description,
role, tags, and deployment, those values are now written into the
repository's own configuration file instead of the project's.
Re-registering the same repository with updated metadata updates that same
file rather than the project's. At this point the project's configuration
still additionally carries the old copy of these fields (removed in
Milestone 2), so nothing user-facing looks different yet from `repo list`
or planning, but the repository's own file is now the place a user edits
to change how it describes itself going forward.

#### - [x] Phase 1.1: Add descriptive fields to a repository's own configuration

A repository's own configuration file gains four new fields — description,
role, tags, and deployment — alongside its existing knowledge and
changelog settings. These fields are validated and given sensible
zero-value defaults the same way the file's existing fields already are,
so a repository with no descriptive metadata continues to load exactly as
it does today.

*Technical detail:* [context.md#phase-11](./context.md#phase-11-add-descriptive-fields-to-a-repositorys-own-configuration)

**Acceptance criteria**:
- [x] A repository's own configuration file can declare a description,
      role, tags, and deployment, and reading that file back returns the
      same values.
- [x] A repository's own configuration file with none of these fields set
      continues to load successfully with empty defaults, unchanged from
      today's behavior.

#### - [x] Phase 1.2: Registering a repository writes its description into the repository itself

When a user registers a repository and supplies a description, role, tags,
or deployment, those values are now written into the target repository's
own configuration file as part of the same step that already creates or
repairs that repository's footprint. Re-registering the same repository
with changed descriptive values updates that file rather than leaving it
stale. The project's own configuration continues to be written as before
for now; it stops carrying these fields in Milestone 2.

*Technical detail:* [context.md#phase-12](./context.md#phase-12-registering-a-repository-writes-its-description-into-the-repository-itself)

**Acceptance criteria**:
- [x] Registering a repository with a description, role, tags, and
      deployment results in those values being present in the target
      repository's own configuration file.
- [x] Registering the same repository again with a changed description
      updates the value already stored in the repository's own
      configuration file.
- [x] Registering a repository with no descriptive metadata supplied
      leaves an already-described repository's own configuration
      unchanged.

### Milestone 2: Listing and planning read a repository's description from the repository itself, and the project no longer stores a copy

**What changes**: The project's configuration entry for a repository is
trimmed down to membership only (name, address or local path, provider,
and dependencies); the four descriptive fields are removed from it
entirely. `repo list` and the plan workflow's repository roster now source
description, role, tags, and deployment from each repository's own
configuration, merging that with the project's membership record. A
repository that has been registered but not yet fetched locally still
lists with its membership details and simply shows no descriptive
metadata until it becomes available; once fetched, its metadata appears on
the next listing without re-registering it. From this point, editing a
shared repository's description in its own configuration is immediately
reflected by every project that registers it, with no per-project edit
needed.

#### - [x] Phase 2.1: The project's repository registry no longer stores descriptive fields

A project's configuration entry for a registered repository is trimmed to
hold only what identifies and locates it: its project-local name, remote
address and/or local path, provider, and any declared dependencies on
other registered repositories. Description, role, tags, and deployment are
removed from this entry entirely, since Milestone 1 already made the
repository's own configuration the place these values are written.

*Technical detail:* [context.md#phase-21](./context.md#phase-21-the-projects-repository-registry-no-longer-stores-descriptive-fields)

**Acceptance criteria**:
- [x] Inspecting a project's configuration after registering a repository
      with a description, role, tags, and deployment shows none of those
      four fields on the repository's entry.
- [x] A dependency declared between two registered repositories is still
      present in the project's configuration after this change.

#### - [x] Phase 2.2: A shared way to read a repository's own description

A single piece of logic now answers "what does this repository say about
itself, if anything" for any registered repository: if the repository is
available locally, its own configuration is read; if it is not yet
available, or its configuration cannot be read, the answer is simply "no
description available" rather than an error. This is the one place that
logic lives, so that everywhere a repository's description is shown, it is
retrieved the same way.

*Technical detail:* [context.md#phase-22](./context.md#phase-22-a-shared-way-to-read-a-repositorys-own-description)

**Acceptance criteria**:
- [x] Asking for a materialized repository's descriptive metadata returns
      the values stored in that repository's own configuration.
- [x] Asking for an unmaterialized repository's descriptive metadata
      returns no metadata, without attempting to clone or fetch anything.
- [x] Asking for descriptive metadata for a repository whose own
      configuration is missing or unreadable returns no metadata rather
      than an error.

#### - [x] Phase 2.3: Listing repositories and planning both show a repository's own description

Listing a project's repositories now reports, for each locally available
repository, its membership details from the project alongside its
descriptive metadata read from the repository's own configuration, using
the shared lookup from Phase 2.2. The same lookup feeds the plan
workflow's repository roster, so an agent planning across a multi-repo
project sees the same description, role, tags, and deployment that
listing reports. A repository that exists only as a remote address and
has not yet been fetched still appears with its membership details, simply
without descriptive metadata, and listing still triggers no cloning or
fetching.

*Technical detail:* [context.md#phase-23](./context.md#phase-23-listing-repositories-and-planning-both-show-a-repositorys-own-description)

**Acceptance criteria**:
- [x] Listing a project's repositories shows, for each locally available
      repository, both its membership details and its descriptive
      metadata sourced from its own configuration.
- [x] Listing a project with a registered, unfetched, address-only
      repository completes successfully, shows that repository without
      descriptive metadata, and performs no clone or fetch.
- [x] After a previously address-only repository is fetched locally,
      listing the project's repositories includes its descriptive
      metadata without re-registering it.
- [x] Two projects that both register the same repository report the same
      description after a single edit to the repository's own
      configuration, with no per-project edit needed.
- [x] During a plan workflow run in a multi-repo project, the repository
      information shown to the agent includes the description, role,
      tags, and deployment read from each repository's own configuration.

### Milestone 3: The documentation site describes the new split

**What changes**: The product documentation site's configuration
reference is updated so that its existing project-configuration page no
longer shows description, role, tags, or deployment as fields of a
registered repository entry, and a new reference page documents a
repository's own configuration file, including these four descriptive
fields alongside its existing knowledge and changelog documentation. A
reader of the documentation now finds each field documented under the
configuration file that actually holds it.

#### - [x] Phase 3.1: The project configuration reference no longer documents repository descriptions

*(target repo: **docs**)* The documentation site's existing configuration
reference page, which describes `.spektacular/config.yaml`, is updated so
its description of a registered repository entry lists only membership
fields — name, address or local path, provider, and dependencies — with
description, role, tags, and deployment removed from that page's account
of the registry entry.

**Content example**: the page's registry-entry documentation (following
the existing `ConfigKey` pattern already used for other sections on this
page) becomes:

```mdx
<ConfigKey name="repos" type="section" defaultValue="none">

  How a project registers its member repositories. Each entry identifies
  and locates one repository:

  - `repos[].name`: the repository's project-local name, unique within
    this project.
  - `repos[].address`: the repository's remote URL, when it is not
    colocated with the project.
  - `repos[].local`: a local path to the repository, when it is colocated
    or already checked out.
  - `repos[].provider`: the repository's provider. Only `git` ships today.
  - `repos[].dependencies`: the project-local names of other registered
    repositories this one depends on.

  A repository's description, role, tags, and deployment are not declared
  here — they live in the repository's own configuration file. See
  [Repository configuration](/repo-configuration/).

</ConfigKey>
```

*Technical detail:* [context.md#phase-31](./context.md#phase-31-the-project-configuration-reference-no-longer-documents-repository-descriptions)

**Acceptance criteria**:
- [x] The configuration reference page's description of a registered
      repository entry lists only membership fields, with no mention of
      description, role, tags, or deployment as fields of that entry.
- [x] The page links to the new repository-configuration reference page
      for where descriptive metadata is now documented.

#### - [x] Phase 3.2: A new reference page documents a repository's own configuration

*(target repo: **docs**)* A new documentation page describes a
repository's own configuration file, covering its existing knowledge and
changelog sections plus the four descriptive fields introduced in
Milestone 1: description, role, tags, and deployment. The page is added to
the site's navigation alongside the existing configuration reference page.

**Content outline** (new page, `ConfigurationKeys`/`ConfigKey` pattern,
mirroring `configuration.mdx`'s structure):

- **Hero**: heading "Repository Configuration", subtitle explaining this
  page covers `.spektacular/config.yaml` as it appears inside a member
  repository's own root, distinct from a project's configuration.
- **Section — "Where repository configuration lives"**: prose explaining
  this file is created as part of a repository's footprint (on `repo add`
  or project init), and that it carries no reference back to any project
  that registers the repository.
- **ConfigurationKeys — "Descriptive metadata"**: one `ConfigKey` per new
  field, e.g.:

  ```mdx
  <ConfigKey name="description" type="string" defaultValue="none">

    A short account of what this repository is or does. Shown wherever
    this repository's details are surfaced, such as `spektacular repo
    list` or the repository roster an agent sees during planning.

  </ConfigKey>

  <ConfigKey name="tags" type="list of strings" defaultValue="none">

    Free-form labels for the repository, such as `docs`, `website`, or
    `backend`.

  </ConfigKey>
  ```

  …with matching entries for `role` and `deployment` in the same shape.
- **ConfigurationKeys — "Knowledge and changelog"**: cross-references the
  existing knowledge-base documentation rather than duplicating it, since
  those sections are unchanged by this plan.

*Technical detail:* [context.md#phase-32](./context.md#phase-32-a-new-reference-page-documents-a-repositorys-own-configuration)

**Acceptance criteria**:
- [x] A new reference page documents description, role, tags, and
      deployment as fields of a repository's own configuration file.
- [x] The new page is reachable from the site's navigation.
- [x] The documentation site builds successfully with the new page in
      place.

## Open Questions

None. A genuine pass over the plan found no uncertainty that can only be
resolved once implementation begins: the current shape of every affected
type, function, and test was read directly during discovery
(research.md), and every design choice (what moves where, how the update
path works around `EnsureFootprint`'s existing-wins semantics, how listing
and planning share one resolver, how the two documentation pages split)
was decided during architecture and phases with direct code citations
rather than left open. The two sequencing notes recorded in context.md
(Phase 2.1 must land with 2.3; Phase 3.1's cross-link depends on 3.2) are
implementation-order constraints, not unresolved questions, and are
already actionable as written.

## Out of Scope

- **Fetching a repository's own configuration over the network without
  cloning it locally.** An unmaterialized repository lists with membership
  details only, and its descriptive metadata stays absent until it is
  fetched by some other means; this plan does not add any command-specific
  or general mechanism to read a remote repository's configuration without
  a local clone. Per the spec's Non-Goals, this applies to every command,
  not only listing.
- **Introducing new descriptive metadata fields.** This plan relocates the
  existing four fields (description, role, tags, deployment) from the
  project's registry to the repository's own configuration; it does not
  add any new kind of descriptive field to either file.
- **Restructuring the documentation site beyond the two configuration
  reference pages.** Navigation, layout, and every other page on the docs
  site are unaffected; only `configuration.mdx` and the new
  repository-configuration reference page change.
- **Migrating or preserving backward compatibility for the prior layout.**
  Per the spec's constraints, the previous arrangement (descriptive fields
  in the project registry) has not shipped in a release, so this plan
  removes those fields outright rather than deprecating them, versioning
  the config format, or providing any read-compatibility path for old
  project configuration files that still contain them.
- **A project-level override or aggregation layer for descriptive
  metadata.** The architecture step considered and rejected mirroring the
  knowledge-sources pattern (an optional project-owned addition merged
  with repo-declared data at read time, spektacular:internal/knowledge)
  for descriptive fields; since the project registry retains none of these
  fields after this plan, there is nothing left to aggregate, and no such
  layer is introduced. If a future need arises for a project to describe a
  repository differently than the repository describes itself, that would
  be a new spec, not an extension of this plan.

## Changelog

### 2026-08-13 — Phase 1.1: Add descriptive fields to a repository's own configuration

**What was done**: Added `Description`, `Role`, `Tags`, and `Deployment`
fields (all `omitempty`) to `config.RepoConfig`, matching the tag style
already used by `config.RepoEntry`. No changes were needed to validation
or default synthesis, since the four fields need none beyond what YAML
unmarshaling already enforces and default correctly to Go zero values.

**Deviations**: None.

**Files changed**:
- `internal/config/repo.go`
- `internal/config/repo_test.go`

**Discoveries**: None beyond what context.md's Current State Analysis
already documented (spec 000042 was marked completed for design sign-off
only; no code had actually changed before this plan's implementation
began).

### 2026-08-13 — Phase 1.2: Registering a repository writes its description into the repository itself

**What was done**: `runRepoAdd` now writes the input's descriptive fields
into the target repository's own `repo.yaml` as a new step after
`EnsureFootprint` returns, updating an already-footprinted repo's
descriptive fields on re-registration rather than only on first creation.
The project's own configuration continues to be written unchanged for now
(Milestone 2 removes these fields from it).

**Deviations**: During test authoring, a bug was found and fixed within
this phase (not deferred): the new write step originally overwrote every
descriptive field unconditionally from input, which meant omitting a
field in `--data` (Go zero value) silently blanked an already-described
repo's existing metadata on re-registration — violating this phase's own
third acceptance criterion. Fixed by only overwriting a field when the
input supplies a non-zero value.

**Files changed**:
- `cmd/repo.go`
- `cmd/repo_test.go`

**Discoveries**: The equivalent "does omitting a field in `--data` blank
existing metadata" question already existed on the project-side `RepoEntry`
upsert path (same unconditional-overwrite pattern) but was left as-is,
since Phase 2.1 removes those fields from `RepoEntry` entirely rather than
needing a parallel fix.

### 2026-08-13 — Phase 2.1: The project's repository registry no longer stores descriptive fields

**What was done**: Removed `Description`, `Role`, `Tags`, and `Deployment`
from `config.RepoEntry`, leaving it membership-only (name, address, local,
dependencies, provider, config). Updated `runRepoAdd`'s `RepoEntry`
construction and `reposEqual` to match. Confirmed `validateRepos` needed no
change, since it never validated these fields.

**Deviations**: None.

**Files changed**:
- `internal/config/config.go`
- `internal/config/config_test.go`
- `cmd/repo.go`
- `cmd/repo_test.go`

**Discoveries**: This repo's own `.spektacular/config.yaml` still carries
the old descriptive fields on its `docs` entry; since `internal/config`
uses plain `yaml.Unmarshal` (not strict/`KnownFields` mode), the now-unknown
YAML keys are silently ignored rather than erroring, so this repo's own
config keeps loading fine with no migration needed, exactly matching the
plan's explicit no-migration stance. Not edited as part of this plan (data
cleanup is a separate, later concern).

### 2026-08-13 — Phase 2.2: A shared way to read a repository's own description

**What was done**: Added `(*repo.Set) DescriptiveMetadata(name string)
(config.RepoConfig, bool)`, a new function that returns a repo's own
descriptive metadata if it is on disk and its config is readable, and an
absent result (never an error) otherwise — mirroring how the existing
`checkFootprint` already tolerates a missing or broken `repo.yaml` for its
own purposes. Never clones or fetches.

**Deviations**: None.

**Files changed**:
- `internal/repo/set.go`
- `internal/repo/set_test.go`

**Discoveries**: None.

### 2026-08-13 — Phase 2.3: Listing repositories and planning both show a repository's own description

**What was done**: `runRepoList` (cmd/repo.go) and `repoRoster` (cmd/plan.go)
both now source descriptive metadata via the Phase 2.2 resolver instead of
reading it off `RepoEntry`, gated the same way materialization was already
checked. `repoRoster`'s signature gained a `root string` parameter (both
call sites already had `root` in scope) so it can build a `repo.Set` and
call the shared resolver per repo, falling back to empty fields on any
resolution error — mirroring `runRepoList`'s existing fallback exactly.
Unmaterialized repos still list with membership only and trigger no clone
or fetch, unchanged from before.

**Deviations**: None.

**Files changed**:
- `cmd/repo.go`
- `cmd/plan.go`
- `cmd/repo_test.go`
- `cmd/plan_test.go`

**Discoveries**: `internal/steps/plan/steps_test.go`'s
`TestDiscoveryAndArchitectureStepsRenderRepoRoster` needed no changes, as
the plan predicted — it feeds a synthetic roster directly into the step
functions, independent of where the roster's data comes from.

### 2026-08-13 — Phase 3.1: The project configuration reference no longer documents repository descriptions

**What was done**: Added a `repos` `ConfigKey` block to `configuration.mdx`
listing only membership fields (name, address, local, provider,
dependencies), with explicit prose noting description/role/tags/deployment
now live in the repository's own configuration and a link to the new
repository-configuration page. Updated the "top-level keys" count from
seven to eight.

**Deviations**: None — matches the plan's own Content example verbatim.

**Files changed**:
- `docs: src/pages/configuration.mdx`

**Discoveries**: None.

### 2026-08-13 — Phase 3.2: A new reference page documents a repository's own configuration

**What was done**: Added a new `repo-configuration.mdx` page documenting a
repository's own `.spektacular/config.yaml` — description, role, tags, and
deployment as `ConfigKey` entries, plus a cross-reference to the existing
knowledge-base and configuration pages for knowledge/changelog rather than
duplicating that documentation. Added a "Repository Configuration" entry
to the site's Resources navigation dropdown, immediately after
"Configuration".

**Deviations**: Initially linked to `/configuration/#changelog` for the
changelog cross-reference, but `ConfigKey`/`ConfigurationKeys` render no
`id` attributes on this site, so any `#fragment` anchor would be dead.
Changed to a plain page link instead.

**Files changed**:
- `docs: src/pages/repo-configuration.mdx`
- `docs: src/components/Nav.astro`

**Discoveries**: The docs repo's own `.spektacular/config.yaml` is missing
from its working tree (deleted, no recent modification time) and an
untracked `.spektacular/repo.yaml` exists with default knowledge/changelog
config, both predating this session's work and unrelated to any command
this plan's implementation ran. Left untouched as out of scope; flagged so
it is not mistaken for something Phase 3.1/3.2 caused, and so it is not
accidentally staged alongside this plan's actual docs changes when
committing.
