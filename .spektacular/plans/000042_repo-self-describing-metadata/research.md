---
created_date: "2026-08-13"
status: completed
closed_date: "2026-08-13"
---

# Research: 000042_repo-self-describing-metadata

## Alternatives considered and rejected

- **Store descriptive metadata in a brand-new repo-level file** (e.g.
  `repo-metadata.yaml`) instead of the existing `repo.yaml`. Rejected: the
  spec's own constraint forbids a new file ("must live in the existing
  repo-level configuration file, not in a new additional file"), and
  `internal/config/repo.go` already defines exactly one repo-level config
  type (`RepoConfig`, repo.go:19-22) with an established loader/writer/footprint
  path — adding a second file would fork that path for no benefit.

- **Give the repo-level config a pointer back to its owning project(s)** so
  listing could resolve metadata via that reference instead of a directory
  read. Rejected: the spec explicitly forbids this ("a repository's
  configuration must not carry a pointer back to any project"), and it is
  already a standing project principle from spec 000039 — "the repo
  footprint is minimal and knows nothing of its projects" (000039 spec,
  consulted via `plan file read`) — that 000042 must not violate.

- **Fetch repo-level config over the network for unmaterialized repos** (read
  `repo.yaml` from the remote without cloning) so listing could show full
  metadata even before a repo is fetched locally. Rejected: explicitly out of
  scope per the spec's Non-Goals, and it would also conflict with the
  side-effect-free listing constraint if implemented via a shallow clone or
  similar. Unmaterialized repos list with membership only, metadata absent,
  by design.

- **Migrate/backfill existing project-registry descriptive fields into repo
  configs automatically on next command run.** Rejected: the spec explicitly
  states no migration or backward-compatibility support is required, because
  the prior layout (metadata in project registry, from spec 000039) has not
  shipped in a release. Implementer can simply delete the fields from
  `RepoEntry` and stop writing them, with no transitional code path.

- **Model the split ad hoc (bespoke merge logic) rather than reusing an
  existing pattern.** Rejected in favor of the knowledge-sources
  aggregation precedent (see Chosen approach below) — spec 000039's plan
  (consulted via `plan file read 000039_project-level-capabilities/research.md`)
  already solved an equivalent "what belongs to the repo vs. the project"
  question for `Knowledge.Sources` (repo-declared sources vs. project-owned
  additions, combined at read time) and that pattern already has tests
  (`internal/knowledge/set_test.go`, `cmd/knowledge_test.go`) worth mirroring
  for structure, even though the repo-metadata case is simpler (no
  additive/override semantics needed — repo.yaml is the sole source of
  descriptive fields, project holds none at all).

- **Rely on `EnsureFootprint` alone to carry metadata updates on re-add**
  (architecture-step alternative). Rejected: `EnsureFootprint`
  (internal/repo/footprint.go:50) treats an existing on-disk `repo.yaml` as
  authoritative over the config passed in, so it silently drops updates to
  an already-footprinted repo's descriptive metadata — which would violate
  the "two projects, one description... with no per-project edit"
  acceptance criterion the moment a user edits metadata via `repo add` a
  second time.

- **Full knowledge-sources-style aggregation with an optional project-level
  override layer** (architecture-step alternative). Rejected as
  over-engineered: unlike knowledge sources, which legitimately have
  project-owned entries with no repo-level equivalent, this spec explicitly
  leaves the project registry with zero descriptive fields, so there is
  nothing left to aggregate against; the repo-level file is simply the sole
  source.

## Chosen approach — evidence

Move `Description`, `Role`, `Tags`, `Deployment` off `config.RepoEntry`
(internal/config/config.go:126-137) and onto `config.RepoConfig`
(internal/config/repo.go:19-22, currently only `Knowledge`/`Changelog`).
`Dependencies` stays on `RepoEntry` — it references other repos by their
project-local registry names, which only make sense in the context of one
project's registry (a repo cannot know its own project-scoped dependency
names, since the same repo may be registered under different names in
different projects).

Evidence for each piece of the approach:

- **`RepoEntry` today** (internal/config/config.go:126-137) — the exact
  fields, yaml tags, and their current home; `validateRepos`
  (config.go:287-307) performs no validation on the four descriptive fields
  today, so moving them requires adding that validation to `RepoConfig`
  instead, not carrying over existing validation.
- **`RepoConfig` today** (internal/config/repo.go:19-94) — already a fully
  separate Go type, YAML file (`repo.yaml`), loader
  (`RepoConfigFromYAMLFile`), validator (`Validate`), defaulter
  (`WithDefaults`), and writer (`ToYAMLFile`) from the project config. Adding
  four fields here is additive to an established, independently-tested path
  (internal/config/repo_test.go has 10 existing tests against this exact
  shape).
- **`runRepoAdd`** (cmd/repo.go:113-203) currently builds a `RepoEntry` with
  the descriptive fields (lines 128-137) and writes it to the *project*
  config (`cfg.ToYAMLFile`, lines 166-174), then separately calls
  `repo.EnsureFootprint(resolved.Root, config.NewDefaultRepoConfig())`
  (lines 176-199) with a blank default that carries no descriptive data —
  confirming there is currently no code path threading registration input
  into repo-level config. This is the exact call site to change: build a
  `RepoConfig` (not just `NewDefaultRepoConfig()`) carrying the input's
  description/role/tags/deployment, and pass that into `EnsureFootprint`
  instead.
- **`EnsureFootprint`** (internal/repo/footprint.go:28-88) treats an
  existing on-disk `repo.yaml` as authoritative over the passed-in
  `repoCfg` once the footprint already exists (line 50: "loads the existing
  on-disk config as authoritative") — so re-running `repo add` against an
  already-footprinted repo with *changed* descriptive metadata will not
  currently update `repo.yaml` through `EnsureFootprint` alone.
  `TestRepoAdd_UpsertUpdatesMetadataWithoutTouchingFootprint`
  (cmd/repo_test.go:343-375) documents today's behavior explicitly (update
  lands in project config, footprint untouched) and will need to become the
  inverse: update lands in the repo's `repo.yaml`, project config untouched.
  The update path therefore needs new logic beyond create/repair: when the
  footprint already exists, merge/overwrite the descriptive fields onto the
  loaded `RepoConfig` and write it back, rather than relying on
  `EnsureFootprint`'s existing-wins semantics.
- **`runRepoList`** (cmd/repo.go:205-259) reads descriptive fields straight
  off `config.RepoEntry` via `set.Entries()` (lines 229-239) with no
  repo-level read at all today. `set.Present(name)` (line 240, backed by
  internal/repo/set.go:74-110) is the existing "is this repo materialized
  locally" gate already used to decide whether to call `set.Resolve` for
  `Root`/`Materialized`/`StaleNote` — this is the natural gate to reuse: when
  `Present` is true, additionally attempt `config.RepoConfigFromYAMLFile` at
  the resolved root and merge its descriptive fields into `repoInfo`; when
  false, or when the read fails/errors, treat as "no metadata" exactly as an
  unreadable repo config today falls back gracefully elsewhere (consistent
  with the plan's Technical Approach: "treat a missing or unreadable repo
  config as 'no metadata' rather than an error").
- **`repoRoster`** (cmd/plan.go:144-163) is the sole feed for the plan
  workflow's discovery/architecture templates (called at cmd/plan.go:132 and
  :220), and reads only `cfg.Repos` today, explicitly commented as carrying
  "config-owned identity only." Templates consuming it:
  templates/steps/plan/02-discovery.md:15-24 and
  templates/steps/plan/03-architecture.md:5-14, both rendering
  `{{name}} — {{description}} (role: {{role}}) [tags: {{tags}}]
  (deployment: {{deployment}})`. Once descriptive fields move to `repo.yaml`,
  `repoRoster` must become repo-config-aware: for each registered repo,
  resolve materialization (same `set.Present`/`set.Resolve` gate as `repo
  list`) and read `repo.yaml` when present, falling back to name-only when
  not. This is the same merge logic as `runRepoList`, so the implementation
  should share a helper between `cmd/repo.go` and `cmd/plan.go` rather than
  duplicate the merge twice.
  `TestDiscoveryAndArchitectureStepsRenderRepoRoster`
  (internal/steps/plan/steps_test.go:577-622) currently feeds a synthetic
  roster map directly into the step functions (bypassing `repoRoster`
  itself), so that test's shape/contract is unaffected by where the data
  comes from — only `repoRoster`'s own (currently absent, needs adding)
  tests and any `cmd/plan_test.go` coverage of it are affected.
- **Docs precedent for the split**: no existing page documents `repos:` at
  all (confirmed via full-corpus search of the docs repo) — this is new
  content, not an edit. The `ConfigKey` MDX component
  (docs repo: src/components/sections/ConfigKey.astro) and its established
  usage pattern in src/pages/configuration.mdx:78-93 (a `type="section"`
  key with nested bullet-documented sub-fields) is the template to follow
  for both: (a) documenting the trimmed `repos:` entry (name, address,
  local, provider, dependencies) on the existing `configuration.mdx` page,
  and (b) a new page for the repo-level `repo.yaml` file's fields
  (knowledge, changelog, plus the new description/role/tags/deployment),
  added to nav (docs repo: src/components/Nav.astro:6-20, "Resources"
  dropdown) alongside "Configuration."

## Files examined

- spektacular:internal/config/config.go:126-163 — `RepoEntry` struct (all
  current fields + yaml tags), `Config.Repos` wiring.
- spektacular:internal/config/config.go:287-316 — `validateRepos` (no
  validation of descriptive fields today), `RepoEntry.WithDefaults`
  (defaults `Provider` to `git`).
- spektacular:internal/config/repo.go:1-94 — `RepoConfig` struct (currently
  `Knowledge`+`Changelog` only), `RepoConfigFileName = "repo.yaml"`,
  `NewDefaultRepoConfig`, `RepoConfigFromYAMLFile`, `Validate`,
  `WithDefaults`, `ToYAMLFile` — the existing repo-level config surface the
  four fields must be added to.
- spektacular:cmd/repo.go:19-48 — `repoAddCmd`, `repoAddInputSchema` (already
  accepts description/role/tags/deployment/dependencies as input).
- spektacular:cmd/repo.go:86-95 — `repoAddInput` struct.
- spektacular:cmd/repo.go:113-203 — `runRepoAdd`: builds `RepoEntry` from
  input (128-137), upserts into project config (139-160), writes project
  config (166-174), calls `EnsureFootprint` with a blank default config
  (176-199) — the exact site needing to change to write descriptive fields
  into the repo's own config instead.
- spektacular:cmd/repo.go:205-259 — `runRepoList`: reads descriptive fields
  from project `RepoEntry` only (229-239), uses `set.Present`/`set.Resolve`
  for materialization (240-249) — the gate to reuse for a repo-config read.
- spektacular:cmd/repo.go:279-297 — `reposEqual` (compares descriptive
  fields on project-side `RepoEntry` today; will need to stop comparing
  fields that move to repo-level, or compare against loaded `RepoConfig`
  instead for idempotency).
- spektacular:cmd/repo_test.go:169-247 — `TestRepoAdd_RegistersEntryAndCreatesMinimalFootprint`,
  `TestRepoAdd_ReAddChangesNothingInTargetRepo` — assert today's shape
  (descriptive fields in project config; footprint is metadata-free); both
  will invert once the split lands.
- spektacular:cmd/repo_test.go:252-303 — `TestRepoList_ReportsRegisteredReposWithMetadataAndRoots`
  — asserts `repo list` sources description/role from project config; needs
  updating to assert repo-config sourcing instead, plus a new case for
  materialized-with-metadata vs. unmaterialized-without.
- spektacular:cmd/repo_test.go:343-375 — `TestRepoAdd_UpsertUpdatesMetadataWithoutTouchingFootprint`
  — documents today's "update lands in project config only" behavior; the
  inverse test is required post-change.
- spektacular:internal/repo/footprint.go:28-88 — `EnsureFootprint`: creates
  or repairs `repo.yaml`; existing on-disk config wins over the passed-in
  `repoCfg` once already present (line 50) — the semantics `runRepoAdd`'s
  update path must work around for descriptive-field updates on re-add.
- spektacular:internal/repo/footprint_test.go:22-109 — existing
  `EnsureFootprint` tests, all knowledge/changelog-scoped; none touch
  descriptive fields yet.
- spektacular:internal/repo/set.go:74-110 — `Set.Present`/`Set.LocalRoot`,
  the existing materialization check to reuse for the merge decision in both
  `repo list` and `repoRoster`.
- spektacular:internal/repo/set.go:203-212 — `Set.checkFootprint` (validates
  presence/parseability of `repo.yaml`, returns `*FootprintError` if
  missing/broken — a model for "unreadable repo config" tolerance).
- spektacular:cmd/plan.go:144-163 — `repoRoster` — sole feed for plan
  workflow discovery/architecture templates; reads only `cfg.Repos` today;
  explicitly comments "config-owned identity only"; does not project
  `Dependencies` at all.
- spektacular:cmd/plan.go:132,220 — call sites setting `wf.SetData("repos",
  repoRoster(cfg))` in `runPlanNew` and the resume/goto path.
- spektacular:internal/steps/plan/steps.go:95-118 — `repoRosterExtra`,
  `discovery()`, `architecture()` — how the roster round-trips through
  workflow JSON state into template extras; unaffected by where `repoRoster`
  itself sources its data.
- spektacular:templates/steps/plan/02-discovery.md:15-24,
  templates/steps/plan/03-architecture.md:5-14 — the Mustache templates
  rendering the roster bullet list agents see (`**{{name}}** — {{description}}
  (role: {{role}}) [tags: {{tags}}] (deployment: {{deployment}})`).
- spektacular:templates/skills/skill_manage-repos.md:7,14,20 and
  templates/skills/skill_spawn-planning-agents.md:9 — narrative skill docs
  telling agents where descriptive metadata lives / how to use it to scope
  research; will need wording updates once the source of truth moves.
- spektacular:internal/steps/plan/steps_test.go:577-695 —
  `TestDiscoveryAndArchitectureStepsRenderRepoRoster`,
  `TestDiscoveryAndArchitectureStepsRenderEmptyRegistryFallback`,
  `TestDiscoveryStepDirectsRepoScopedResearch`,
  `TestArchitectureStepRequiresRepoAttribution` — feed synthetic roster data
  directly into step functions; contract unaffected by `repoRoster`'s
  data source.
- spektacular:internal/project/init.go:70-186 — `Init`: writes the
  colocated repo's own `RepoConfig` only if absent (138-144); cascades
  `EnsureFootprint` with a blank default over every materialized registered
  repo (168-186) — same blank-default gap as `runRepoAdd`; likely needs the
  same fix if init is expected to seed descriptive fields for
  already-registered repos (spec's "no migration required" suggests this is
  out of scope — init only needs to keep working for the metadata-free
  membership fields it already handles).
- spektacular:internal/project/init_test.go:269-439 — init cascade tests;
  none assert descriptive-field content today, so no expected breakage from
  the split itself, only from any new fields added to `RepoConfig`'s zero
  value affecting `WithDefaults`/round-trip tests.
- spektacular:internal/config/config_test.go:397-519 — repo-entry
  validation and round-trip tests (`TestToYAMLFile_ReposRoundTrip`,
  `TestValidateRepos_*`, `TestRepoEntry_WithDefaultsResolvesProviderToGit`)
  — the round-trip test currently exercises Description/Role/Tags/Deployment
  on `RepoEntry` and will need to move its assertions to `RepoConfig`'s own
  round-trip coverage in repo_test.go instead.
- spektacular:internal/config/repo_test.go:14-187 — existing `RepoConfig`
  test suite (10 tests), including
  `TestRepoConfig_ToYAMLFileWritesOnlyKnowledgeAndChangelog` (line 50) whose
  name literally asserts today's minimal shape — will need renaming/updating
  once descriptive fields are added.
- spektacular:.spektacular/config.yaml — confirms the CURRENT, unmigrated
  state in this very repo: the `docs` repo entry still carries
  `description`, `role`, `tags`, `deployment` in the project config today,
  despite spec 000042 being marked `completed` in `spec file list` — the
  spec's closure only reflects design/requirements sign-off, not that this
  repository's own code has been changed yet. This plan is the vehicle to
  actually implement it here.
- docs:src/pages/configuration.mdx:31-139 — confirms zero existing
  documentation of `repos:`/`dependencies`/`address`/`local`/`role`/`tags`/
  `deployment` anywhere on the page; top-level keys list (35-37) omits
  `repos` entirely.
- docs:src/pages/configuration.mdx:78-93 — the `ConfigKey` usage pattern to
  imitate (`type="section"` key, nested bullet docs of sub-fields, blank
  lines around slot content per Rule 3).
- docs:src/components/sections/ConfigKey.astro:1-23 — the component
  definition (`name`, `type`, `defaultValue` props, default slot for body).
- docs:src/components/Nav.astro:6-20 — flat nav list + "Resources" dropdown
  (Configuration, Extending, Debugging) — a new repo-config reference page
  needs a fourth entry here.
- docs:.spektacular/knowledge/conventions/mdx-authoring.md:9-77 — the four
  MDX authoring rules (no layout HTML in page bodies, ConfigKey/slot pattern
  over string props, blank lines around slot content, fenced code blocks) —
  binding on any new/edited page content for this plan.

## External references

None. This is an internal refactor with no external library or protocol
dependency; no external references were needed.

## Prior plans / specs consulted

- `go run . plan file read 000039_project-level-capabilities/plan.md`,
  `.../context.md`, `.../research.md` — the plan that originally introduced
  `RepoEntry`'s descriptive fields into the *project* config. Learned: the
  fields were modeled wholesale on `KnowledgeConfig.Sources`'s shape (a
  provider-agnostic list of entries) with no separate consideration of
  repo-vs-project ownership for the descriptive subset — i.e., no prior
  design deliberately chose "project" for these fields; it was inherited
  from the pattern it was cloned from. Also surfaced the strongest
  precedent for 000042's approach: mid-implementation, the *knowledge*
  section was walked back from "moves wholesale to repo config" to a
  split — repo-declared sources plus an optional project-owned
  `Knowledge.Sources` list, aggregated at read time — proving this kind of
  split is both precedented and already tested in this codebase
  (`internal/knowledge/set_test.go`, `cmd/knowledge_test.go`). Also
  reconfirmed the standing invariant "the repo footprint knows nothing of
  its projects" and the "nothing is duplicated between the two levels"
  principle, both of which 000042's design must continue to satisfy.
- `go run . spec file read 000039_project-level-capabilities.md` — original
  requirement bundled membership + descriptive metadata into one registry
  entry description; acceptance criterion "a registered repo appears in
  project configuration with its metadata" is the exact assumption 000042
  revises. Also reconfirmed the footprint-knows-nothing-of-projects
  principle as an explicit design statement, not just an implementation
  detail.
- `go run . knowledge search` (multiple queries: "repo metadata", "project
  config", "repo registry", "multi-repo", "repo footprint", "repo init") —
  no relevant hits in the knowledge base; this domain has no captured
  knowledge-base entry, so the plan must rely on the plan/spec record and
  direct code reading rather than any distilled convention.

## Open assumptions

- **`Dependencies` stays on the project-side `RepoEntry`** (not moved to
  `RepoConfig`). This follows directly from the spec's explicit "Dependencies
  stay project-scoped" requirement, so it is not really an assumption so
  much as a direct requirement restatement — flagged here only because the
  codebase research found `Dependencies` handled identically to the four
  moving fields in `add`/`list`/`reposEqual` today, so the implementer must
  deliberately keep it out of the fields that move, not treat it as bundled.
  If this is somehow wrong, the implement workflow must stop and ask —
  but the spec text is unambiguous, so this is low-risk.
- **The repo-add "update" path needs new merge logic, not just a changed
  `EnsureFootprint` call.** Assumed from `EnsureFootprint`'s documented
  existing-wins semantics (footprint.go:50) plus the existing
  `TestRepoAdd_UpsertUpdatesMetadataWithoutTouchingFootprint` test's
  documented current behavior. Not verified against any explicit design
  note (000039 never anticipated updates to repo-owned metadata via `repo
  add` because the fields didn't live there yet) — if the phases step
  finds a simpler existing mechanism for "overwrite specific fields on an
  existing config," this assumption should be revisited, but no such
  mechanism was found in the research.
- **`repoRoster` (cmd/plan.go) and `runRepoList` (cmd/repo.go) should share
  a single merge helper** rather than duplicating the "read RepoConfig when
  materialized, else no metadata" logic twice. This is an implementation
  preference inferred from both functions needing identical logic, not a
  spec requirement — the phases step should decide the concrete function
  signature/location (likely a new small function in `internal/repo` or
  `internal/config`, callable from both `cmd/repo.go` and `cmd/plan.go`).
- **A new repo-level config reference doc page is needed on the docs site**,
  since none currently exists, rather than folding repo.yaml docs into the
  existing `configuration.mdx` page. Assumed from the existing page's title/
  scope ("Configuration" documents the *project* config file specifically)
  and the Nav "Resources" dropdown pattern accommodating a fourth peer page
  cleanly. If the user prefers a single combined page instead, that's a
  presentation choice the architecture step should confirm, not re-research.

## Drafting assumptions

### Dependencies excluded from the moving fields (discovery)
- **Decision**: `Dependencies []string` stays on the project-side `RepoEntry`; only `Description`, `Role`, `Tags`, `Deployment` move to `RepoConfig`.
- **Rationale**: The spec explicitly requires "Dependencies stay project-scoped" since they reference other repos by project-local registry names. The codebase handles `Dependencies` identically to the four descriptive fields in `add`/`list`/`reposEqual` today, so this had to be a deliberate exclusion, not an oversight.
- **Rejected**: Moving `Dependencies` alongside the other four for symmetry — rejected because it would break the moment the same repo is registered under different local names in two projects, which the spec's "shared repos stay consistent" requirement depends on staying project-scoped.

### Repo-add update path needs new merge logic (discovery)
- **Decision**: Assume `runRepoAdd`'s handling of an update to an already-registered, already-footprinted repo's descriptive metadata will need new logic beyond calling `EnsureFootprint` as-is, since `EnsureFootprint` treats an existing on-disk `repo.yaml` as authoritative over the passed-in config once the footprint already exists.
- **Rationale**: Direct code evidence from `internal/repo/footprint.go:50` and the existing test `TestRepoAdd_UpsertUpdatesMetadataWithoutTouchingFootprint` (cmd/repo_test.go:343-375), which documents today's inverse behavior explicitly.
- **Rejected**: Assuming `EnsureFootprint` alone suffices — no evidence supports this; its documented semantics are create-or-repair, not update-in-place.

### Shared merge helper between repo list and plan roster (discovery)
- **Decision**: Assume `runRepoList` (cmd/repo.go) and `repoRoster` (cmd/plan.go) should share one merge helper for "read RepoConfig when materialized, else no metadata" rather than duplicating it.
- **Rationale**: Both functions need identical logic (same materialization gate via `set.Present`, same fallback-to-absent behavior on missing/unreadable repo config) — DRY preference, not a spec requirement.
- **Rejected**: Duplicating the merge in both call sites — would work but violates the project's general DRY preference and doubles the surface for a future bug if the merge logic changes.

### New docs page for repo-level config (discovery)
- **Decision**: Assume a new docs page is needed for the repo-level `repo.yaml` reference, rather than folding it into the existing `configuration.mdx` (which documents the project config specifically).
- **Rationale**: `configuration.mdx`'s title/scope is specifically the project config file; the site's Nav "Resources" dropdown pattern (Configuration, Extending, Debugging) accommodates a fourth peer page cleanly; no existing page currently documents `repo.yaml` at all.
- **Rejected**: Combining both config file references onto a single page — a plausible alternative; flagged as a presentation choice for the architecture step to confirm rather than lock in now.

### Chosen architecture direction (architecture)
- **Decision**: Move Description/Role/Tags/Deployment to `RepoConfig`; `runRepoAdd` calls `EnsureFootprint` for create/repair, then separately loads and upserts descriptive fields onto the resulting `repo.yaml`, writing back only on change; a new shared `internal/repo` helper provides "read RepoConfig if materialized, else absent" for both `repo list` and the plan roster. New docs page for repo.yaml, trimmed configuration.mdx for the project side.
- **Rationale**: Directly resolves the footprint update-semantics gap found in discovery (EnsureFootprint's existing-wins behavior) without touching create/repair semantics for other callers (notably project init's cascade); avoids duplicating merge logic across two call sites (DRY convention); avoids over-engineering an aggregation layer the spec doesn't call for (project registry has zero descriptive fields post-change, so there's nothing to aggregate).
- **Rejected**: (a) relying on EnsureFootprint alone for updates — silently drops metadata edits on re-add, violating the "two projects, one description, no per-project edit" acceptance criterion; (b) full knowledge-sources-style aggregation with an optional project-level override layer — unlike knowledge sources, there's no legitimate project-owned descriptive field left to aggregate against, so the extra machinery has no use case here.
- Chosen unilaterally per the architecture step's instruction not to put the choice to the user; grounded entirely in direct code evidence from research.md, not a judgement call requiring confirmation.

### Two-milestone write/read split instead of one atomic milestone (milestones)
- **Decision**: Split into Milestone 1 (write descriptive metadata to repo config, project config still also carries it) and Milestone 2 (remove from project config, switch all reads to repo config) rather than one atomic milestone that does both at once.
- **Rationale**: Each milestone is independently deliverable and separately validatable (write-path correctness vs. read-path correctness), and it avoids a moment where writes and reads are both mid-flight in the same change. The spec's "no migration needed" constraint means there's no compatibility window to preserve, so this split is purely for review/validation granularity, not correctness.
- **Rejected**: One atomic milestone doing the full swap — would work equally correctly given no migration is needed, but produces a single large, harder-to-validate change; rejected in favor of the more granular, independently-checkable split.

### Phase granularity and sequencing notes (phases)
- **Decision**: Split Milestone 2 into three phases (2.1 remove fields, 2.2 shared resolver, 2.3 wire consumers) despite flagging in context.md that 2.1 and 2.3 must be implemented as a tightly sequenced pair (removing fields without switching the read path breaks listing). Similarly, Milestone 3's two phases are flagged as needing to land in the same work session because Phase 3.1's cross-link target doesn't exist until Phase 3.2 lands.
- **Rationale**: Phase boundaries here follow the "one phase = one reviewable unit of acceptance criteria" convention rather than "one phase = a state where the whole system stays green in isolation." This matches the plan format's own guidance (each phase has independent acceptance criteria) while being explicit in context.md about real sequencing dependencies so the implement workflow doesn't parallelize what shouldn't be parallelized.
- **Rejected**: Merging 2.1+2.3 into a single phase (and 3.1+3.2 into a single phase) to guarantee no intermediate broken state — rejected because it would produce phases with two unrelated "what changes" concerns (schema removal vs. read-path rewiring; one doc page's trim vs. another's creation) bundled together, working against the plan's own granularity rules. The context.md sequencing note is judged sufficient to prevent the implement workflow from landing them out of order.

## Rehydration cues

- Re-run `go run . repo list` from the spektacular repo root to see current
  (as of research) live state: `docs` repo still carries description/role/
  tags/deployment via the *project* config today (unmigrated) — confirms
  scope of work still needed even though the spec is marked `completed`.
- Re-read `internal/config/config.go:126-163` and `internal/config/repo.go`
  in full to refresh exact current field sets before drafting data
  structures.
- Re-read `cmd/repo.go:113-297` in full (runRepoAdd, runRepoList,
  reposEqual) before drafting phases — these three are the primary
  behavior-change surface.
- Re-read `internal/repo/footprint.go:28-88` (EnsureFootprint) before
  deciding the exact update-on-re-add mechanism.
- `go run . skill spawn-planning-agents` was already invoked this session
  for research fan-out guidance; re-invoke if additional research rounds
  are needed later in the workflow.
- Docs repo root: `/home/nicj/code/github.com/jumppad-labs/spektacular-website`
  (resolved via `go run . repo list`, local path `../spektacular-website`
  relative to the spektacular repo root). `ConfigKey`/`ConfigurationKeys`
  components and `Nav.astro` are the concrete files to edit/extend there.
