---
created_date: "2026-08-13"
status: completed
closed_date: "2026-08-13"
---

# Context: 000042_repo-self-describing-metadata

## Current State Analysis

As of this plan's research (2026-08-13), spec 000042 is marked `completed`
in the spec store, but the actual code has not yet been changed: this
repository's own `.spektacular/config.yaml` still carries `description`,
`role`, `tags`, and `deployment` on the `docs` repo entry
(spektacular:.spektacular/config.yaml). The spec's closure reflects
design/requirements sign-off only; this plan is the implementation.

Today's structures:

- `RepoEntry` (spektacular:internal/config/config.go:126-137) — the
  project registry entry type, currently carrying `Name`, `Address`,
  `Local`, `Description`, `Role`, `Tags`, `Dependencies`, `Deployment`,
  `Provider`, `Config`. `validateRepos` (config.go:287-307) validates only
  name/uniqueness/location/provider; none of the four descriptive fields
  are validated today. `RepoEntry.WithDefaults()` (config.go:311-316)
  defaults `Provider` to `git`.
- `RepoConfig` (spektacular:internal/config/repo.go:19-22) — the
  repo-level config type, currently `Knowledge`/`Changelog` only, backing
  `.spektacular/repo.yaml` inside each member repo's own root
  (`RepoConfigFileName`, repo.go:13). Fully separate loader/validator/
  writer from the project config: `RepoConfigFromYAMLFile` (repo.go:50-66),
  `Validate` (repo.go:69-74), `WithDefaults` (repo.go:79-82), `ToYAMLFile`
  (repo.go:85-94), `NewDefaultRepoConfig` (repo.go:26-46).
- `runRepoAdd` (spektacular:cmd/repo.go:113-203) — builds a `RepoEntry`
  from input including all four descriptive fields (lines 128-137),
  upserts into project config (lines 139-160), writes project config
  (lines 166-174), then calls `repo.EnsureFootprint(resolved.Root,
  config.NewDefaultRepoConfig())` (lines 176-199) with a blank default —
  no descriptive data reaches the repo's own config today.
- `EnsureFootprint` (spektacular:internal/repo/footprint.go:28-88) —
  creates `repo.yaml` if missing, repairs it if unparseable, but loads the
  *existing* on-disk config as authoritative once already present (line
  50) — meaning a caller-supplied `repoCfg` has no effect on an
  already-healthy footprint.
- `runRepoList` (spektacular:cmd/repo.go:205-259) — reads descriptive
  fields straight from `config.RepoEntry` via `set.Entries()` (lines
  229-239); never reads any repo's own `repo.yaml`. Uses
  `set.Present(e.Name)` (line 240, backed by
  spektacular:internal/repo/set.go:74-110) as the existing "is this repo
  materialized" gate before resolving further.
- `repoRoster` (spektacular:cmd/plan.go:144-163) — the sole feed for the
  plan workflow's discovery/architecture templates (called at
  cmd/plan.go:132 and :220); reads only `cfg.Repos` today; does not
  project `Dependencies` at all.
- Docs site (spektacular-website): zero existing documentation of
  `repos:`, `dependencies`, `address`, `local`, `role`, `tags`, or
  `deployment` anywhere in `src/pages/` (confirmed by full-corpus search).
  `configuration.mdx`'s top-level-keys list (lines 35-37) omits `repos`
  entirely. This is greenfield doc work, not an edit-in-place split.

Prior design record (spec 000039, consulted via `plan file read`/`spec
file read`, not treated as current-state truth): `RepoEntry`'s descriptive
fields were modeled wholesale on `KnowledgeConfig.Sources`'s shape, with no
separate deliberation on repo-vs-project ownership for the descriptive
subset — inherited from the pattern it was cloned from, not a considered
choice. The closest prior precedent for this plan's split is that same
spec's mid-implementation walkback of the *knowledge* section from
"moves wholesale to repo config" to a split (repo-declared sources plus an
optional project-owned addition, aggregated at read time) — proven and
already tested (`internal/knowledge/set_test.go`, `cmd/knowledge_test.go`).
Standing invariants from that spec that this plan must not violate: "the
repo footprint knows nothing of its projects" (no pointer back to any
project) and "nothing is duplicated between the two levels."

## Per-Phase Technical Notes

### Phase 1.1: Add descriptive fields to a repository's own configuration

**File changes**:
- `internal/config/repo.go:19-22` — add `Description string`, `Role
  string`, `Tags []string`, `Deployment string` fields to `RepoConfig`,
  with `yaml:"description,omitempty"`, `yaml:"role,omitempty"`,
  `yaml:"tags,omitempty"`, `yaml:"deployment,omitempty"` tags, matching the
  tag style already used on `config.RepoEntry` (config.go:126-137).
- `internal/config/repo.go:69-74` (`RepoConfig.Validate()`) — decide
  whether any validation is needed for the new fields; per discovery,
  `validateRepos` (config.go:287-307) validates none of these fields
  today, so the simplest consistent choice is no new validation beyond
  what YAML unmarshaling already enforces (e.g. `tags` must be a string
  list). Confirm this against `RepoEntry`'s current (absent) validation
  before adding anything stricter.
- `internal/config/repo.go:79-82` (`RepoConfig.WithDefaults`) — no change
  expected; the four new fields default to their Go zero values (empty
  string / nil slice) with no synthesis needed, unlike `Knowledge`, which
  synthesizes a default source.
- `internal/config/repo.go:26-46` (`NewDefaultRepoConfig`) — no change;
  continues to leave the four new fields at zero value, matching how a
  freshly-created repo has no description until registration supplies one.

**Test changes**:
- `internal/config/repo_test.go` — extend or add round-trip tests
  analogous to the existing 10 tests (lines 14-187) asserting the four new
  fields serialize and deserialize correctly via `ToYAMLFile`/
  `RepoConfigFromYAMLFile`, and that a `RepoConfig` with none of them set
  still round-trips to the same minimal YAML shape the existing
  `TestRepoConfig_ToYAMLFileWritesOnlyKnowledgeAndChangelog` (line 50)
  currently asserts — that test's name and body need updating to reflect
  the new, larger-but-still-omitempty field set.

**Complexity**: Low
**Token estimate**: ~10k tokens
**Agent strategy**: Single agent, sequential execution.

### Phase 1.2: Registering a repository writes its description into the repository itself

**File changes**:
- `cmd/repo.go:113-203` (`runRepoAdd`) — after the existing
  `repo.EnsureFootprint(resolved.Root, config.NewDefaultRepoConfig())` call
  (lines 176-199, footprint create/repair, unchanged), add a new step:
  load the resulting `repo.yaml` via `config.RepoConfigFromYAMLFile`, apply
  the input's `Description`/`Role`/`Tags`/`Deployment` (from
  `repoAddInput`, lines 86-95) onto the loaded `RepoConfig`, and write it
  back via `RepoConfig.ToYAMLFile` only if any field actually changed
  (mirroring the existing `reposEqual`-gated write pattern used for the
  project config at lines 139-160, but comparing against the loaded
  `RepoConfig`'s current descriptive fields instead).
- `cmd/repo.go:128-137` — the `RepoEntry` built from input keeps its
  `Description`/`Role`/`Tags`/`Deployment` assignments for now (Phase 2.1
  removes them from `RepoEntry` itself); Phase 1.2 only adds the
  repo-config write alongside the existing project-config write, so both
  are populated simultaneously during this milestone.
- `internal/repo/footprint.go` — no change; `EnsureFootprint`'s
  create/repair semantics (lines 28-88) stay exactly as they are. The new
  update step in `runRepoAdd` is a separate call after `EnsureFootprint`
  returns, not a change to `EnsureFootprint` itself, per the architecture
  decision to avoid teaching it two responsibilities.

**Test changes**:
- `cmd/repo_test.go:169-204`
  (`TestRepoAdd_RegistersEntryAndCreatesMinimalFootprint`) — extend to
  assert the target repo's `repo.yaml` now also contains the supplied
  descriptive fields (this test's "minimal footprint" premise changes;
  rename or add a sibling test rather than silently redefining
  "minimal").
- `cmd/repo_test.go:343-375`
  (`TestRepoAdd_UpsertUpdatesMetadataWithoutTouchingFootprint`) — this
  test's name and assertions describe exactly the behavior Phase 1.2
  changes; add a new test asserting the inverse (re-add with a changed
  description updates the repo's own `repo.yaml`), and update or remove
  the "footprint untouched" assertion in the original once Phase 2.1 also
  lands (footprint-touched is now correct; whether the project-config side
  of this test still holds depends on Phase 2.1's changes — track both
  phases together when updating this specific test to avoid asserting a
  contradiction mid-milestone).

**Complexity**: Medium
**Token estimate**: ~25k tokens
**Agent strategy**: Single agent, sequential execution (the change touches
one function's control flow plus its tests; not parallelizable
meaningfully, but large enough to exceed the Low tier's simple-field-add
shape).

### Phase 2.1: The project's repository registry no longer stores descriptive fields

**File changes**:
- `internal/config/config.go:126-137` (`RepoEntry`) — remove
  `Description`, `Role`, `Tags`, `Deployment` fields entirely. Retain
  `Name`, `Address`, `Local`, `Dependencies`, `Provider`, `Config`.
- `internal/config/config.go:287-307` (`validateRepos`) — no logic change
  expected (it never validated the removed fields), but confirm no
  dangling references remain.
- `cmd/repo.go:128-137` (`runRepoAdd`'s `RepoEntry` construction) — stop
  assigning `Description`/`Role`/`Tags`/`Deployment` onto the `RepoEntry`
  literal; continue assigning `Name`, `Address`, `Local`, `Dependencies`,
  `Provider`.
- `cmd/repo.go:279-297` (`reposEqual`) — remove comparison of
  `Description`/`Role`/`Deployment`/`Tags`; retain comparison of the
  fields that still exist on `RepoEntry` (`Address`, `Local`,
  `Dependencies`, `Provider`) so re-add idempotency still works for
  membership fields.

**Test changes**:
- `internal/config/config_test.go:487-519`
  (`TestToYAMLFile_ReposRoundTrip`) — remove assertions on
  `Description`/`Role`/`Tags`/`Deployment` for `RepoEntry`; the equivalent
  coverage now lives in `internal/config/repo_test.go` from Phase 1.1.
- `cmd/repo_test.go:169-247` — update
  `TestRepoAdd_RegistersEntryAndCreatesMinimalFootprint` and
  `TestRepoAdd_ReAddChangesNothingInTargetRepo` to assert the project
  config's `RepoEntry` carries no descriptive fields.
- `cmd/repo_test.go:252-303`
  (`TestRepoList_ReportsRegisteredReposWithMetadataAndRoots`) — this test
  currently asserts `repo list` sources description/role from a
  project-config entry (including a synthetic address-only entry added
  directly to `config.yaml`); it must be updated in tandem with Phase 2.3
  (below), since removing the fields here without also switching the read
  path in `runRepoList` would break listing entirely — implement Phase 2.1
  and Phase 2.3 as a tightly sequenced pair even though they are separate
  phases for review granularity.

**Complexity**: Low
**Token estimate**: ~10k tokens
**Agent strategy**: Single agent, sequential execution.

**Sequencing note**: Phase 2.1 must land together with Phase 2.3 — removing
the fields from `RepoEntry` without also switching `runRepoList`/
`repoRoster` to read from repo-level config would break listing between
the two phases landing. Do not implement or merge them independently.

### Phase 2.2: A shared way to read a repository's own description

**File changes**:
- New function in `internal/repo` (e.g. alongside `Set` in
  `internal/repo/set.go`, near the existing `Present`/`LocalRoot` methods
  at lines 74-110): a function such as `DescriptiveMetadata(root string)
  (config.RepoConfig, bool)` (exact name TBD by implementer) that: checks
  materialization the same way `Set.Present`/`Set.LocalRoot` already do,
  and if present, calls `config.RepoConfigFromYAMLFile` on the resolved
  root's `repo.yaml`; returns a zero-value result and `false` (or
  equivalent "absent" signal) on any read/parse error, never surfacing the
  error to the caller as a failure — mirroring how `Set.checkFootprint`
  (set.go:203-212) already tolerates a missing/broken `repo.yaml` for its
  own purposes.
- No change to `internal/repo/footprint.go` or `Set.Resolve` — this is a
  new, narrower read path that does not need full footprint resolution,
  only materialization-and-read.

**Test changes**:
- New test file section (e.g. in `internal/repo/set_test.go` or a new
  `internal/repo/metadata_test.go`) covering: materialized + readable
  config → returns metadata; unmaterialized → returns absent, asserts no
  clone/fetch side effect occurred (e.g. via a fake/observable git
  provider or by asserting no new directories were created); materialized
  but corrupt/missing `repo.yaml` → returns absent, not an error.

**Complexity**: Medium
**Token estimate**: ~20k tokens
**Agent strategy**: Single agent, sequential execution (new, self-contained
function plus its direct tests; no dependency on Phase 2.1/2.3's changes
to compile or test in isolation).

### Phase 2.3: Listing repositories and planning both show a repository's own description

**File changes**:
- `cmd/repo.go:205-259` (`runRepoList`) — at the point description/role/
  tags/deployment are currently copied from `e := set.Entries()[i]` (lines
  229-239), replace with a call to Phase 2.2's new resolution function,
  gated the same way materialization is already checked via
  `set.Present(e.Name)` (line 240); when absent, leave `repoInfo`'s
  descriptive fields at zero value exactly as the unmaterialized case
  already produces today.
- `cmd/plan.go:144-163` (`repoRoster`) — currently reads only
  `cfg.Repos` and projects `name`/`description`/`role`/`tags`/`deployment`
  (no `dependencies`, confirmed in discovery). Update to also resolve each
  repo's root (reusing the same resolution the `internal/repo.Set` already
  provides — `repoRoster` will need access to a `Set` or equivalent
  resolver, not just the raw `cfg.Repos` list) and call Phase 2.2's
  function per repo, falling back to name-only (empty description/role/
  tags/deployment) when absent, matching `runRepoList`'s fallback exactly.
- `cmd/plan.go:132,220` — call sites of `repoRoster` continue passing
  `cfg`; if `repoRoster`'s signature needs to grow to accept a resolver,
  update both call sites (`runPlanNew` and the resume/goto path)
  accordingly.

**Test changes**:
- `cmd/repo_test.go:252-303`
  (`TestRepoList_ReportsRegisteredReposWithMetadataAndRoots`) — update to
  set up a materialized repo with its own `repo.yaml` carrying descriptive
  metadata (rather than a project-config entry carrying it), and assert
  `repo list` reports it; add a case for an unmaterialized, address-only
  entry asserting metadata is absent and no clone/fetch occurs (this may
  already be close to existing coverage per discovery's note that the test
  already adds a synthetic address-only entry).
- `internal/steps/plan/steps_test.go:577-622`
  (`TestDiscoveryAndArchitectureStepsRenderRepoRoster`) — per discovery,
  this test feeds a synthetic roster directly into `discovery()`/
  `architecture()`, bypassing `repoRoster` itself, so it should not need
  changes; add new, separate test coverage directly for `repoRoster` in
  `cmd/plan_test.go` (create if it does not already assert this function)
  covering: materialized repo with metadata → full roster entry;
  unmaterialized repo → name-only roster entry.
- New or extended test covering the "two projects, one repo, one edit"
  success metric: register the same local-path repo into two separate
  project configs, edit the repo's own `repo.yaml` description once,
  assert `repo list` under both projects reports the updated value.

**Complexity**: Medium
**Token estimate**: ~25k tokens
**Agent strategy**: Single agent, sequential execution (touches two call
sites that must stay behaviorally consistent with each other and with
Phase 2.2's new function; splitting across agents risks the two call
sites drifting from each other's fallback behavior).

**Sequencing note**: see Phase 2.1's sequencing note — 2.1 and 2.3 must
land together.

### Phase 3.1: The project configuration reference no longer documents repository descriptions

**File changes** (target repo: **docs**, resolved root
`/home/nicj/code/github.com/jumppad-labs/spektacular-website`):
- `docs:src/pages/configuration.mdx` — per discovery, this page currently
  documents no `repos:` section at all (top-level keys list at lines
  35-37 omits it entirely). This phase adds a new `ConfigKey
  name="repos"` block (see the Content example in plan.md's Phase 3.1)
  documenting only membership fields, inserted into the existing
  `<ConfigurationKeys heading="Top-level keys">` block alongside the
  existing `command`/`agent`/`spec_trigger_threshold`/`debug`/`spec`/
  `plan`/`changelog`/`knowledge` entries (lines 31-139), and updates the
  "Seven top-level sections" prose (lines 33-37) to include `repos` and
  reflect the new count.
- `docs:src/pages/configuration.mdx` frontmatter `description` (lines
  1-4) — no change required unless the one-line site description should
  mention repos; leave as-is unless the implementer judges it materially
  stale.

**Test changes**: none (docs content changes verified by the site's
existing build-time guards per the Testing Approach; no new automated
content test).

**Complexity**: Low
**Token estimate**: ~10k tokens
**Agent strategy**: Single agent, sequential execution.

**Sequencing note**: this phase's new `repos` `ConfigKey` block links to
the page Phase 3.2 creates; land 3.1 and 3.2 in the same work session, or
add the cross-link as part of 3.2 instead, to avoid a broken intermediate
link.

### Phase 3.2: A new reference page documents a repository's own configuration

**File changes** (target repo: **docs**):
- New file `docs:src/pages/repo-configuration.mdx` (exact slug at
  implementer's discretion, consistent with existing kebab-case page
  names like `how-it-works.mdx`, `knowledge-base.mdx`) — built from
  `Hero`, `Section`, `ConfigurationKeys`, `ConfigKey` per the Content
  outline in plan.md's Phase 3.2, following `configuration.mdx`'s exact
  structural pattern (frontmatter shape at configuration.mdx:1-4, `Hero`
  usage at configuration.mdx:13-16, `Section`/`ConfigurationKeys`
  alternating `surface` per the alternate-section-background convention —
  first `Section` explicit `surface={false}` equivalent already implied
  by `Section.astro`'s own default, following `ConfigurationKeys`
  defaulting `surface=true`, so alternate accordingly starting from
  whichever the new page's first block resolves to).
- `docs:src/components/Nav.astro:6-20` — add a fourth entry to the
  "Resources" dropdown's `children` array, e.g. `{ label: "Repository
  Configuration", href: "/repo-configuration/" }`, immediately after the
  existing "Configuration" entry.
- Cross-link: `docs:src/pages/configuration.mdx`'s new `repos` `ConfigKey`
  block (Phase 3.1) should link to this new page once it exists — if
  Phase 3.1 lands first, its link target will 404 until this phase lands;
  implement Phase 3.1 and 3.2 in the same work session or add the link as
  part of this phase instead to avoid a broken intermediate state.

**Test changes**: none beyond the existing build-time guards (MDX
layout-HTML lint, `astro check`, `npm run build`); confirm these pass with
the new page present.

**Complexity**: Medium
**Token estimate**: ~20k tokens
**Agent strategy**: Single agent, sequential execution (net-new page
authored as one coherent unit; splitting page sections across agents
would risk inconsistent voice/structure on a single reference page).

## Project References

- **spektacular** (local: `.`, resolved root
  `/home/nicj/code/github.com/jumppad-labs/spektacular`) — all Go/CLI/struct
  work: Phases 1.1, 1.2, 2.1, 2.2, 2.3.
- **docs** (address `git@github.com:jumppad-labs/spektacular-website.git`,
  local: `../spektacular-website`, resolved root
  `/home/nicj/code/github.com/jumppad-labs/spektacular-website`, role:
  documentation, tags: docs/website/astro, deployment: static-site) — all
  documentation work: Phases 3.1, 3.2. Already materialized locally at the
  resolved root above; no fetch needed before starting docs work.

## Testing Strategy

Per-phase testing detail is folded into each phase's Technical Notes above
(Test changes subsections). At the plan level: Phase 1.1 and 2.1 add/adjust
schema-level round-trip and validation tests; Phase 1.2 and 2.3 add
regression tests that explicitly invert two existing tests documenting
today's (pre-change) behavior; Phase 2.2 is covered by new, self-contained
unit tests for the new resolution function, including an explicit
no-side-effect assertion for the unmaterialized case; Phase 3.1/3.2 rely on
the docs site's existing build-time structural guards rather than new
content-assertion tests. See plan.md § Testing Approach for the
success-metrics-to-verification mapping.

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|-------------|----------------|
| Low | ~10k | Single agent, sequential |
| Medium | ~25k | 2-3 parallel agents |
| High | ~50k+ | Parallel analysis, sequential integration |

All seven phases in this plan are Low or Medium complexity; none reach the
High tier, so no phase requires parallel-analysis-then-sequential-
integration orchestration. Two phase pairs (2.1+2.3, 3.1+3.2) have hard
landing-order dependencies noted in their Technical Notes above, even
though neither individually reaches High complexity.

## Migration Notes

None. Per the spec's constraints, the prior layout (descriptive metadata
in the project registry) has not shipped in a release, so this plan
removes those fields outright with no migration, deprecation period, or
read-compatibility path for old project configuration files.

## Performance Considerations

None beyond what the existing code already does: the new repo-metadata
resolution helper (Phase 2.2) performs at most one additional local file
read per materialized repo when listing or planning, matching the cost
profile of the file reads `repo list` already performs for
materialization/staleness checks today. No network calls are introduced;
the side-effect-free listing constraint explicitly rules out any new
network or clone cost.
