---
created_date: "2026-08-13"
status: completed
closed_date: "2026-08-13"
---

# Context: 000039_project-level-capabilities

## Current State Analysis

Spektacular today is single-repo by construction. Configuration is one file, `.spektacular/config.yaml`, loaded strictly from the current working directory — `configFilePath()`/`loadConfig()` at cmd/root.go:185-205 — with no parent-directory search; a missing file silently yields `config.NewDefault()` (cmd/root.go:201-203), so running outside a project proceeds on defaults and fails only indirectly later. `projectRoot()` is unconditionally the cwd (cmd/root.go:221-227). There is no repo or multi-repo concept anywhere: `Config` (internal/config/config.go:111-120) carries `command`, `agent`, `spec_trigger_threshold`, `debug`, and the `spec`/`plan`/`changelog`/`knowledge` sections, nothing else.

The one list-of-provider-entries pattern — the model for the new repos registry — is `KnowledgeConfig.Sources` (internal/config/config.go:93-108) with per-entry validation and scope uniqueness (config.go:250-268), defaults synthesized at load by `WithDefaults(projectRoot)` (config.go:274-289), and dispatch via a literal provider switch in `knowledge.NewSet` (internal/knowledge/set.go:70-95) that fail-fasts on unknown providers and unreachable sources. `store.NewFileStore(root, scope)` already supports arbitrary roots (used with foreign roots at internal/knowledge/set.go:88).

The store layer (internal/store/store.go:37-151) has no exclusion mechanism: search is a pure-Go walk that "scans every directory and never excludes one by name" (internal/store/search.go:38-43), and listing recursion lives in callers (internal/knowledge/set.go:290-313; cmd/artifacts.go:180-268). Changelog storage is flat — `ChangelogFilePath` = `<dir>/<name>.md` (internal/steps/implement/strategy.go:32-34) — written through the generic store-file factory (cmd/storefile.go:110-328) with ID-prefix enforcement on the leading path segment (cmd/storefile.go:66-80). Front matter carries only created/status/closed (internal/metadata/metadata.go:48-52).

There is no git usage anywhere (go.mod:6-12 lists five direct deps; no os/exec in the tree) and no gitignore-pattern matching. `init` (cmd/init.go:21-57 → internal/project/init.go:17-117) scaffolds directories with MkdirAll top-up idempotency, writes config only if absent, overwrites the embedded `.spektacular/.gitignore`, renders four workflow skills and five heading-delimited AGENTS.md managed sections per agent (internal/agent/), and writes the version file (cmd/version.go:126-135). No command today writes config.yaml back after init.

## Per-Phase Technical Notes

### Phase 1.1: Config splits into project and repo types

**File changes**:
- `internal/config/config.go:111-120` — add `Name string` (yaml `name`) and `Source string` (yaml `source`) to `Config`; the `Knowledge KnowledgeConfig` field STAYS as an optional project-owned sources list (walkthrough amendment — repo stores are declared in repo configs; project config carries only sources owned by the project, e.g. team/global scopes). `KnowledgeConfig`/`SourceConfig`/`FileKnowledgeConfig` types (`config.go:93-108`) are shared by both config levels.
- `internal/config/config.go:123-164` — `NewDefault()`: stop seeding `Knowledge.Sources` (:153-161) — the default project-level list is empty; the repo's own store is seeded in `NewDefaultRepoConfig` instead. Drop/retire `KnowledgeConfig.WithDefaults`'s project-scope synthesis (`config.go:274-289`) at project level; the equivalent default lives in the repo config type. `Name` defaults empty (validated non-empty only at explicit `Validate` time for loaded projects — `NewDefault` is also used pre-init).
- `internal/config/config.go:186-205` — `Config.Validate()`: add required slug-safe `name` check (new `validateSlug` helper: lowercase alphanumeric + `-`/`_`, no path separators — reuse pattern style from `internal/identifier/identifier.go:34-35`); the existing `KnowledgeConfig.Validate` continues to validate the (now optional, possibly empty) project-level list.
- New `internal/config/repo.go` — `RepoConfig{Knowledge KnowledgeConfig; Changelog ChangelogConfig}` with `RepoConfigFromYAMLFile` (mirror `FromYAMLFile` `config.go:167-183`: env expansion + defaults-prefilled unmarshal + validate), `ToYAMLFile` (mirror `config.go:292-301`), `NewDefaultRepoConfig()`, `Validate()` (reuse `KnowledgeConfig.Validate` `config.go:250-268` and `ChangelogConfig.Validate` `config.go:238-246`), `WithDefaults` (mirror `config.go:274-289`). Repo config filename constant: `repo.yaml` beside `config.yaml`.
- `internal/config/config_test.go` — update round-trip test (:105-142) for name/source and the now-empty-by-default project knowledge list (a populated project-level list must still round-trip); new validation cases in the style of :231-261 (assert on config key path in error message); new `repo_test.go` cases for `RepoConfig` parse/validate/round-trip including its default-seeded own store.

**Complexity**: Medium
**Token estimate**: ~40k tokens
**Agent strategy**: Single agent, sequential (config.go and repo.go are tightly coupled; tests follow).

### Phase 1.2: Running without a project fails explicitly

**File changes**:
- `cmd/root.go:196-205` — `loadConfig()`: when `configFilePath()` does not exist, return a structured error (`output.NewError` — `internal/output/writer.go:63-80` — code e.g. `no_project`, message naming the cwd, `WithNextAction("<command> init <agent>")`) instead of `config.NewDefault()`. Add `loadConfigLenient()` preserving the old fall-back for bootstrap callers.
- `cmd/init.go:36` — switch to `loadConfigLenient()` (init must run pre-project).
- `cmd/version.go:109` (`staleAction`) — switch to `loadConfigLenient()` so `version check` still composes advice without a project.
- `cmd/skill.go` — verify it does not call `loadConfig` (research says it reads only `templates.FS`); no change expected.
- Command tests that ran in bare temp dirs and relied on the silent default now need a minimal `.spektacular/config.yaml` fixture (with `name:`) — sweep `cmd/*_test.go` fixtures (`cmd/knowledge_test.go:69-105` `twoScopeProject`, `cmd/implement_test.go:51-63` harness, spec/plan/storefile/artifacts tests). Add new gate tests: project-operating command errors with next-action; `init`/`version`/`skill` succeed without a project.
- Note: `runRoot` (`cmd/root.go:60`) loads config for `debug.enabled` — must tolerate no-project (use lenient loader there; the gate belongs to command handlers, not the debug probe).

**Complexity**: Medium (small code change, wide test fallout)
**Token estimate**: ~45k tokens
**Agent strategy**: 2 parallel agents — one for the loader/gate change + new tests, one sweeping existing test fixtures; sequential merge.

### Phase 1.3: Init establishes project identity and the colocated repo

**File changes**:
- `cmd/init.go:14-19,21-57` — add optional `--name` flag; default = slugified basename of cwd; set `cfg.Name` before the config write at :41-45. Config write remains only-if-absent for the rest of the file but must backfill `name` when missing (load-modify-write via `ToYAMLFile`).
- `internal/project/init.go:26-53,69-87,102-114` — move knowledge-directory scaffolding to be driven by the colocated repo config: write `.spektacular/repo.yaml` (via `RepoConfig.ToYAMLFile`, only-if-absent like config.yaml at :83-87); create knowledge dirs + category READMEs from `RepoConfig.Knowledge` instead of `cfg.Knowledge` (:45 filter and :69-74 loop).
- `cmd/knowledge.go:143-153` (`newKnowledgeSet`) — interim wiring: load the colocated `.spektacular/repo.yaml` and concatenate its `Knowledge.Sources` with `cfg.Knowledge.Sources` from project config (repo sources first, project-owned sources after), passing the combined list to `knowledge.NewSet` (signature unchanged — `internal/knowledge/set.go:70-95`; only the caller's assembly of `KnowledgeConfig` changes). Registry-wide aggregation replaces the colocated half in Phase 2.4.
- `internal/project/init_test.go` — new assertions: repo.yaml created, both files present, second run no-op (compare directory state); keep `TestInit_CreatesGitignore` (:49-56) green.
- `cmd/init_test.go:54-72,193-218` — extend idempotency and stale-version tests for name backfill; `cmd/version_test.go:59-80` — staleAction unchanged behavior.

**Complexity**: Medium
**Token estimate**: ~35k tokens
**Agent strategy**: Single agent, sequential (init flow is one path).

### Phase 2.1: The project config gains a repo registry

**File changes**:
- `internal/config/config.go` — add `ProviderGit = "git"` beside `ProviderFile` (:29); add `Repos []RepoEntry` to `Config` (:111-120); new types `RepoEntry{Name, Address, Local, Description, Role string; Tags, Dependencies []string; Deployment string; Provider string; Config GitRepoConfig}` and `GitRepoConfig{}` (empty, reserved) — identity/metadata fields deliberately siblings of the provider block, mirroring `SourceConfig` (:99-103).
- `internal/config/config.go:186-205` — wire a `ReposConfig`-style validation into `Config.Validate`: per-entry slug-safe `name` (reuse Phase 1.1's `validateSlug`), name uniqueness (model on scope uniqueness :255-259), at-least-one-of `address`/`local`, provider ∈ {git} with empty defaulting to git (default applied in a `WithDefaults` pass modeled on :274-289).
- `internal/project/init.go` — seed `cfg.Repos = [{Name: cfg.Name, Local: "."}]` when the list is empty, alongside the Phase 1.3 identity write, so the colocated repo is registered from birth.
- `internal/config/config_test.go` — validation matrix: dup name, bad slug, neither address nor local, unknown provider, empty provider defaults to git; round-trip with a two-entry registry.

**Complexity**: Low
**Token estimate**: ~25k tokens
**Agent strategy**: Single agent, sequential.

### Phase 2.2: Repos resolve to local directories, cloning when absent

**File changes**:
- New `internal/repo/set.go` — registry component modeled structurally on `knowledge.NewSet` (`internal/knowledge/set.go:70-95`): `New(cfg config.Config, projectRoot string, git GitRunner) (*Set, error)`; provider dispatch = literal switch on `entry.Provider`, `default:` fail-fast `repo %q: provider %q is not supported`. `Resolve(name)` / `ResolveAll()` returning `ResolvedRepo{Name, Root, Entry, Materialized, StaleNote}`. Resolution order: `Local` set and dir exists → use (join to projectRoot when relative, mirroring set.go:76-79); else `Address` set → clone target `.spektacular/repos/<name>/` under projectRoot, clone only when target absent; neither usable → error. After resolution, footprint check: `<root>/.spektacular/repo.yaml` parses and validates; failure returns a structured error built with `output`-compatible fields (surfaced by cmd layer via `output.NewError(...).WithResource(root).WithNextAction("<command> repo add ...")`) — report + repair offer, never silent continuation.
- New `internal/repo/git.go` — `GitRunner` interface `{Clone(url, dir string) error; LocalHead(dir string) (string, error); RemoteHead(url string) (string, error)}`; `execGitRunner` implementation via `os/exec` (`git clone --`, `git -C <dir> rev-parse HEAD`, `git ls-remote <url> HEAD`); missing-binary detection (`exec.LookPath`) with a dedicated error. This file is the codebase's only subprocess site.
- Cross-platform handling in the executor (walkthrough addition): `exec.LookPath("git")` resolves `git.exe` via PATHEXT on Windows — never hardcode the binary name with an extension; `strings.TrimSpace` (not just `\n` trimming) on `rev-parse`/`ls-remote` output to absorb CRLF; clone targets built with `filepath.Join` throughout; set `GIT_TERMINAL_PROMPT=0` in the child env on every platform (and `GIT_SSH_COMMAND` batch-mode guard where applicable); auth is deliberately whatever the platform's git provides (GCM on Windows, keychain on macOS) — never reimplemented.
- Staleness in `set.go`: when `Materialized` (root under `.spektacular/repos/`), compare `LocalHead` vs `RemoteHead(entry.Address)`; differing heads → `StaleNote` populated; any error from `RemoteHead` (offline/auth) → best-effort note or empty, never a resolution failure. Never invoke fetch/pull.
- `templates/.spektacular/.gitignore` — add `repos/` entry (file is overwritten on re-init per `internal/project/init.go:90-96`, so existing projects pick it up on re-init).
- New `internal/repo/set_test.go` — fake `GitRunner` covering: local-wins-over-address, clone-only-when-absent, no-git-invocation for local repos, stale note population, remote-error degradation, unknown provider, footprint-missing repair error. Model fixture style on `internal/knowledge/set_test.go:12-53`.
- New `internal/repo/git_integration_test.go` — real git against local-path remotes in `t.TempDir()` (plain paths, not `file://` URLs — portable across Windows drive-letter paths; skip with `testing.Short()` or `exec.LookPath` guard): clone lands, re-resolve reuses, `git status --porcelain` in the "project" shows nothing after materialization (no gitlink), behind-remote detection after a commit to the fixture remote.

**Complexity**: High
**Token estimate**: ~55k tokens
**Agent strategy**: Parallel analysis (set.go vs git.go vs tests can be drafted concurrently), sequential integration — the resolution semantics must be assembled by one agent.

### Phase 2.3: The CLI manages the repo lifecycle

**File changes**:
- New `cmd/repo.go` — `repoCmd` group + `repo add`/`repo list`, Shape A JSON commands modeled on `cmd/knowledge.go:15-67` (declarations), :152-154 (`--schema` short-circuit), :305-323 (`--data` parse with example-payload error), :337-346 (flag registration). `repo add` input mirrors `RepoEntry` metadata fields; flow: `loadConfig` → validate entry → idempotent registration (by name: identical entry → no-op; new → append) → `cfg.ToYAMLFile(configFilePath())` (`internal/config/config.go:292-301` — first CLI config write; comment loss accepted per assumption log) → footprint create/repair on the resolved target (shared routine in `internal/repo`: write `repo.yaml` only-if-absent, create knowledge dirs + category READMEs — extract the scaffolding from `internal/project/init.go:69-74,102-114` into a reusable `repo.EnsureFootprint(root, repoCfg)` so init and `repo add` share it). Output: `{registered, footprint: created|repaired|unchanged}`.
- `repo list` — `newRepoSet` helper (model `newKnowledgeSet` `cmd/knowledge.go:143-153`); output projection modeled on `runKnowledgeSources` (`cmd/knowledge.go:236-250`): name, address, local, resolved root, provider, metadata, `stale_note` when present. Listing must not clone: resolve lazily — entries with no usable local root and an address report `root: ""`/`materialized: false` rather than triggering a clone (cloning happens on use, not on list).
- `cmd/root.go:236-244` — register `repoCmd`.
- New `cmd/repo_test.go` — model on `cmd/knowledge_test.go`: flag-reset helper (:53-64 pattern), project fixture with registry, JSON-mirror structs; cases: fresh add creates footprint, re-add no-op, add-from-second-project leaves footprint untouched (build two project fixtures pointing at one repo dir), add with neither address nor local errors, list projection including a stale-note fake.
- `cmd/artifacts.go` / schema listing — confirm no interaction needed (repo command group is additive).

**Complexity**: Medium
**Token estimate**: ~45k tokens
**Agent strategy**: 2 parallel agents (command implementation vs test suite), sequential merge.

### Phase 2.4: Init cascades over members, and knowledge flows through the registry

**File changes**:
- `internal/project/init.go` — after project scaffolding, iterate `cfg.Repos`, resolve each via `internal/repo` (local-only resolution during init — do not clone during init; skip address-only entries not yet materialized, with a reported notice) and run `repo.EnsureFootprint` on each resolved root — create-or-repair is the same call `repo add` uses. Cascade must not touch non-managed files (contract: `cmd/init_test.go:193-218`).
- `cmd/knowledge.go:143-153` (`newKnowledgeSet`) — replace the Phase 1.3 colocated shortcut: build the aggregate source list by resolving all registered repos (again local-only, skip unmaterialized with notice), loading each repo's `repo.yaml`, concatenating their `Knowledge.Sources` in registry order (colocated/self entry first by construction), then appending the project config's own `Knowledge.Sources` last (project-owned sources — repo-declared sources take precedence). Scope-addressed `read`/`write` (`internal/knowledge/set.go:165-181`, `byScope` :267-274): first match in that combined order wins; duplicate scopes across levels are allowed (per-file uniqueness still enforced by `KnowledgeConfig.Validate`).
- `internal/knowledge/set.go` — no interface change; `SourceInfo` (:57-62) gains a `Repo` attribution field so `knowledge sources` output (`cmd/knowledge.go:236-250`) says which repo each source came from.
- Tests: `cmd/knowledge_test.go` — two-repo project fixture with distinct knowledge entries; search results attributed correctly; solo-project regression (identical output to pre-phase). `internal/project/init_test.go` — cascade repairs a deleted member `repo.yaml`; healthy re-init no-op.

**Complexity**: Medium
**Token estimate**: ~40k tokens
**Agent strategy**: 2 parallel agents (init cascade vs knowledge aggregation), sequential merge.

### Phase 3.1: Changelog stores namespace by project

**File changes**:
- `cmd/changelog_file.go:8-14` — the `storeDirFunc` closure returns `filepath.Join(c.Changelog.Config.Directory, c.Name)` instead of the bare directory. Agent-facing paths stay `<plan_name>.md`; the project folder is injected below the CLI surface, so `validateIDPrefix` (`cmd/storefile.go:66-80`) continues to see the ID as the leading segment of the *user-supplied* path — verify and add a regression test rather than modifying it.
- `internal/steps/implement/strategy.go:32-34,51` — `ChangelogFilePath` gains the project-name segment so the `changelog_path` template var renders the true on-disk path; project name reaches the strategy via `workflow.Config` gaining a `ProjectName string` field (`internal/workflow/workflow.go:17-29`) populated at `cmd/implement.go:112-140` (and the plan/spec constructors for symmetry) — a scalar config value, consistent with the existing `ChangelogDir` precedent.
- `cmd/artifacts.go:259-284` (`appendChangelogArtifacts`) — list one folder level down (`st.List(join(dir, project))`), tolerant of the folder not existing yet.
- `internal/metadata/metadata.go:48-52` — add `Project`, `ProjectSource`, `Spec`, `Plan` string fields (yaml `project`, `project_source`, `spec`, `plan`, all omitempty); `internal/metadata/merge.go:29-82` — preserve the new fields with same-as-status semantics (incoming value wins when set, existing preserved otherwise); `internal/metadata/frontmatter.go` — no change (marshal handles new fields).
- Tests: `internal/metadata/metadata_test.go:22-100` style additions for the four fields + merge preservation; `cmd/storefile_*` / `cmd/artifacts` tests updated for the nested layout; strategy test for the new path var.

**Complexity**: Medium
**Token estimate**: ~35k tokens
**Agent strategy**: 2 parallel agents (metadata vs path/listing changes), sequential merge.

### Phase 3.2: Implemented work derives per-repo changelog entries

**File changes**:
- `cmd/storefile.go:110-328` — extend the factory (or the changelog instantiation only) with a `--repo <name>` flag on `write`/`read`/`list`: when set, build the store rooted at the resolved member repo (`store.NewFileStore(resolved.Root, "repo:"+name)` — the foreign-root construction proven at `internal/knowledge/set.go:88`) and the directory from that repo's `repo.yaml` `Changelog.Config.Directory` joined with the project name. Flag registered only for the changelog group (`cmd/changelog_file.go`), not spec/plan.
- Auto-stamped provenance on `--repo` writes: the write path (`cmd/storefile.go:115-153`) passes `metadata.Merge` options carrying `Project: cfg.Name`, `ProjectSource: cfg.Source`, `Spec`/`Plan` derived from the plan-slug-equals-spec-slug convention (the written filename's ID-prefixed name, cf. `internal/steps/implement/strategy.go:13-15`).
- `templates/steps/implement/10-update_feature_changelog.md:16-36` — after the central `changelog file write`, add the derivation procedure: run `repo list`, identify repos affected by the plan's `## Changelog` file lists, author one filtered entry per affected repo (stage under `.spektacular/tmp/`, include the human-readable reference line naming project + spec/plan), write each with `--repo <name>`.
- `templates/steps/implement/07-update_changelog.md:24-27` — the per-phase Files-changed list groups paths by repo (repo name prefix) so 10- can filter per repo mechanically.
- `templates/steps/implement/08-update_repo_changelog.md:7-22` — generalize "the repo root" to "each affected repo's root CHANGELOG.md".
- Tests: `cmd/` test for `--repo` routing end-to-end (two-repo fixture; assert path, front matter fields, no collision across two projects — spec criterion); `internal/steps/implement/steps_test.go` render-anchor tests (model :311-317 area) for the three template changes; instruction-surface test stays green (`internal/agent/instruction_surface_test.go:33`).

**Complexity**: Medium-High
**Token estimate**: ~50k tokens
**Agent strategy**: 2-3 parallel agents (CLI routing / templates / tests), sequential integration on the flag contract.

### Phase 4.1: Sources can exclude paths from listing and search

**File changes**:
- `go.mod` — add the gitignore matcher dependency (default `github.com/sabhiram/go-gitignore`; confirm negation/anchoring/`**` behavior against gitignore(5) in tests before committing to it).
- New `internal/store/ignore.go` — `IgnoreMatcher` interface; `LoadIgnore(root string)` reading `<root>/.spektacular_ignore` (missing file → pass-everything matcher); `NewIgnoreStore(inner Store, m IgnoreMatcher) Store` decorator: `List` filters returned `DirEntry`s, `Search` post-filters hits by path; `Read`/`Write`/`Exists`/`Delete`/`Root` delegate untouched (spec constraint: direct access never blocked). Amend the store doc comment at `internal/store/search.go:38-43` to note exclusion exists as a wrapper, not in `FileStore`.
- Wiring at construction sites: a small helper (e.g. `store.NewSourceStore(root, scope)` = `NewIgnoreStore(NewFileStore(root, scope), LoadIgnore(root))`) adopted at `internal/knowledge/set.go:88`, `cmd/storefile.go:94`, `cmd/artifacts.go:84`, `cmd/spec.go:211,300`, `cmd/plan.go:128,195`, `cmd/implement.go:140,207` — project storage locations and repo-rooted stores alike. Note the ignore file is rooted at the *store root* each store is constructed with (project root for project stores, repo root for repo stores) per the spec's "rooted at the source it applies to".
- Tests: new `internal/store/ignore_test.go` — excluded path absent from `List`/`Search`, present via `Read`/`Exists`; negation pattern re-includes; directory pattern excludes subtree; no ignore file = passthrough (byte-identical results). `cmd`-level test: knowledge search in a fixture with an excluded noise directory.

**Complexity**: Medium
**Token estimate**: ~35k tokens
**Agent strategy**: Single agent, sequential (one decorator + mechanical wiring).

### Phase 4.2: Workflows and skills go cross-repo

**File changes**:
- Repo-roster preload plumbing (walkthrough amendment): `cmd/plan.go:107-147` — the plan `new`/`goto` handlers already hold the loaded `config.Config`; pass `cfg.Repos` identity metadata into the workflow so step renders can consume it. Route: the `Extra` map at the `writeStep` call sites for the discovery and architecture callbacks (`internal/steps/plan/steps.go:68-99`; `Extra` example at `internal/steps/spec/steps.go:150`) — stepkit merges Extra into template vars (`internal/stepkit/stepkit.go:78-85`), no stepkit or `workflow.Config` change. Templates render the roster with a mustache section (`{{#repos}}… name/description/role/tags/deployment …{{/repos}}`, plus a `{{^repos}}`/empty fallback). Registry data (resolved roots, staleness) is deliberately NOT rendered — CLI only.
- `templates/steps/plan/02-discovery.md:13-37` — Step 2 opens with the rendered repo roster ("the repos this project spans") followed by the instruction: run `{{config.command}} repo list` for resolved paths and staleness before touching a repo, use each repo's metadata to scope research, search within each registered repo; the research.md "Files examined" section (:31) records entries as `<repo>:path:line`. Note exclusions keep noise out and that agents' native tools are not bound by them.
- `templates/steps/plan/03-architecture.md:25-36` — opens with the same rendered roster; attribution requirement: the drafted architecture names, per requirement, the repo (and files) it is carried out against; resolution recorded in the plan's context document (per spec Technical Approach — no new document type).
- `templates/steps/plan/10-phases.md:8,17` — phase entries name their target repo when not the project's own colocated repo.
- `templates/skills/skill_spawn-planning-agents.md` — fan research agents out per registered repo for multi-repo projects.
- `templates/skills/skill_spawn-implementation-agents.md` — carry out each phase's work in its attributed repo's resolved root.
- New `templates/skills/skill_manage-repos.md` — library skill (auto-discovered by `cmd/skill.go:50,68-81`; served raw — no `{{command}}`/`{{config.command}}` placeholders, per the placeholder-convention gotcha): documents `repo add`/`repo list`, registration idempotency, materialization/staleness semantics, repair flow. Model structure on `templates/skills/skill_update-changelog.md`.
- `templates/skills/workflows/spek-plan/SKILL.md` and `templates/skills/workflows/spek-implement/SKILL.md` — cross-repo notes (these render `{{command}}` at install — `internal/agent/skills.go:47`).
- `README.md:148-174` — rewrite the configuration section: project vs repo config files, the repos registry, the changelog section (fixing the existing docs drift), `.spektacular_ignore`; breaking-change re-init notice. `CHANGELOG.md` — breaking-change entry.
- Tests: `internal/steps/plan/steps_test.go` (model `TestDiscoveryStepUsesKnowledgeCommands` :121-132 — renderStep + require.Contains, ToLower) for discovery/architecture/phases anchors; `internal/steps/implement/steps_test.go` for the spawn-implementation reference; `cmd/skill_test.go` — `manage-repos` listed and retrievable; `internal/agent/instruction_surface_test.go` — no forbidden shapes introduced; per-agent skill-install tests untouched (no new workflow skill — library skill only).

**Complexity**: Medium
**Token estimate**: ~45k tokens
**Agent strategy**: 2-3 parallel agents (plan templates / implement templates + skill / docs), sequential review pass for vocabulary consistency across templates.

## Testing Strategy

Per-phase test work (conventions: testify require, t.TempDir/t.Chdir fixtures, one-behaviour-one-func, `// Criterion N:` traceability, cobra flag-reset hygiene from cmd/knowledge_test.go:53-64; CLI tests drive through runRoot):

- **Phase 1.1** — config unit tests in internal/config: name slug validation, repo config parse/validate/round-trip (own store seeded by default), project-level knowledge list optional/empty-by-default but round-trips when populated; model error-message assertions on config key paths (config_test.go:231-261 style).
- **Phase 1.2** — gate tests in cmd: project-operating command errors with next-action in a bare dir; init/version/skill succeed; sweep existing cmd test fixtures to add minimal project configs.
- **Phase 1.3** — init tests: both files created with level-scoped contents, second run no-op, name backfill, knowledge regression for solo project.
- **Phase 2.1** — validation matrix for RepoEntry (dup/bad name, neither address nor local, unknown provider, provider defaulting).
- **Phase 2.2** — internal/repo unit tests against fake GitRunner (local-wins, clone-only-when-absent, stale note, remote-error degradation, footprint repair error); real-git integration tests with file:// remotes in t.TempDir (clone lands, reuse, clean project git status, behind-remote detection), guarded by exec.LookPath.
- **Phase 2.3** — cmd/repo_test.go: fresh add creates footprint, re-add no-op, second-project add leaves footprint untouched, list projection with stale note.
- **Phase 2.4** — cascade repair test in internal/project; two-repo-plus-project-source knowledge aggregation (repo sources first, project-owned last, attribution correct) + solo-project regression in cmd/knowledge_test.go.
- **Phase 3.1** — metadata field round-trip/merge preservation; nested changelog layout in storefile/artifacts tests; strategy path-var test.
- **Phase 3.2** — end-to-end --repo routing (two-repo fixture, provenance front matter, two-projects-no-collision — spec criterion); implement template render-anchor tests (steps_test.go:311-317 model).
- **Phase 4.1** — gitignore semantics matrix in internal/store/ignore_test.go (excluded absent from List/Search, direct Read works, negation, directory patterns, no-file passthrough byte-identical).
- **Phase 4.2** — plan/implement template anchor tests (TestDiscoveryStepUsesKnowledgeCommands model, internal/steps/plan/steps_test.go:121-132), including a roster-render test (renderStep with a two-repo config asserts both repo names/descriptions appear in the rendered discovery and architecture instructions, and the empty-registry fallback renders cleanly); skill listing test; instruction-surface test green; full harbor plan-workflow suite as the milestone gate.

Success-metric mapping (from plan.md § Testing Approach): metrics 3 and 4 are behavioural tests (Phase 3.2 and 4.1 respectively); metrics 1, 2, and 5 are Manual — captured in the implementation test plan.

## Project References

- Spec: `.spektacular/specs/000039_project-level-capabilities.md` (working-tree version is authoritative; note a second spec shares the 000039 ID — `000039_version-check`).
- Repository: git@github.com:jumppad-labs/spektacular.git, branch `main`, planned at commit 3ccbbd2.
- Prior art to re-read when implementing: plan store docs `000020_context/research.md` (provider pattern, fail-fast contract), `000033_spec-workflow-output-changelog/plan.md` (changelog store pattern) — via `plan file read`, not the Read tool.
- Knowledge base: `architecture/workflow-steps.md` (FSM/step/template primer; its callback signature is stale vs current code, which also passes a store).
- External: gitignore(5) semantics (https://git-scm.com/docs/gitignore); candidate matcher `github.com/sabhiram/go-gitignore`; fallback `go-git/v5/plumbing/format/gitignore`.
- Originating issue: jumppad-labs/spektacular#25 (readers may expect a home-directory project registry — explicitly excluded).

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|-------------|----------------|
| Low | ~10k | Single agent, sequential |
| Medium | ~25k | 2-3 parallel agents |
| High | ~50k+ | Parallel analysis, sequential integration |

Phase estimates total ~450k tokens across 11 phases (largest: Phase 2.2 at ~55k, Phase 3.2 at ~50k). Phases 1.2, 2.3, 2.4, 3.1, 3.2, and 4.2 parallelize into 2-3 agents; 2.2 uses parallel analysis with sequential integration; the rest run single-agent.

## Migration Notes

Breaking change, no migration tooling by design (spec constraint): existing setups re-run `init <agent>`. Specifics implementers should not "fix" silently:

- Pre-split `config.yaml` files fail validation after Phase 1.1/1.2 (missing `name`; stray `knowledge` key rejected with a pointer to repo.yaml). The error message is the migration UX — make it name the re-init command.
- Existing flat changelog entries (`.spektacular/changelog/<name>.md`) are not moved. After Phase 3.1, new entries land under `changelog/<project-name>/`; artifact listing reads the nested layout only. Old entries remain readable by direct path but drop out of listings — acceptable per the authorized breaking change; note it in the CHANGELOG entry (Phase 4.2).
- `templates/.spektacular/.gitignore` is overwritten on re-init, so existing projects pick up the `repos/` entry by re-running init — no append logic needed.
- The harbor suites bundle a linux binary + rendered instructions; they pick up the new behavior on the next `make harbor-test-*` run, but their environment fixtures may carry pre-split config.yaml files that now need a `name` field.

## Performance Considerations

- The ignore wrapper post-filters List/Search results; walks still traverse ignored subtrees (e.g. node_modules). Accepted for this release — pushing the predicate into the walk is a recorded follow-up (plan.md § Out of Scope), consistent with the parked token-efficiency non-goal.
- Staleness checks add one network round-trip (`git ls-remote`) per materialized-repo resolution; best-effort with failure degrading to silence/notice, so offline resolution stays fast on the happy path.
- First resolution of an address-only repo pays full clone cost; subsequent resolutions are pure filesystem checks.
- Knowledge aggregation now loads one repo.yaml per registered repo per command invocation — trivial for realistic registry sizes; no caching added (house rule: every search is a fresh scan, no index/cache/daemon).
