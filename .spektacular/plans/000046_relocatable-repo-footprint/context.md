---
created_date: "2026-09-02"
status: completed
closed_date: "2026-09-02"
---

# Context: 000046_relocatable-repo-footprint

## Current State Analysis

- A registered repo is resolved to exactly one directory, `ResolvedRepo.Root` (`internal/repo/set.go:18-24`), located by the registry's `local` (joined to the project root when relative) or by a clone of the registry's `address` into `.spektacular/repos/<name>/` (`set.go:161-199`). Every consumer, whether it needs the code or the repo's own Spektacular files, reads that one field.
- The registry entry `RepoEntry` (`internal/config/config.go:129-136`) carries `Name`, `Address`, `Local`, `Dependencies`, `Provider`, `Config`; `validateRepos` (`config.go:286-306`) requires at least one of address/local; `WithDefaults` (`:310-315`) defaults the provider. `repo add` builds the entry from `name/address/local/dependencies` (`cmd/repo.go:133-138`) and `reposEqual` compares those fields (`:322-327`). Project init seeds `{Name, Local: "."}` (`internal/project/init.go:57-60`) and knowledge aggregation synthesises the same entry for registry-less configs (`cmd/knowledge.go:181-186`). This repository's own `.spektacular/config.yaml` registers `docs` with both `address` and `local`.
- Consumers of the repo's own files all derive their paths from `Root`: footprint validation (`set.go:223-232`), descriptive metadata (`set.go:120-130`), footprint creation and repair (`internal/repo/footprint.go:28-88`), registration write-back (`cmd/repo.go:197,206`), knowledge aggregation (`cmd/knowledge.go:197-231`, with a colocated-repo tolerance keyed on `root != projectRoot` at `:208`), the repo-routed changelog store (`cmd/storefile.go:98-134`), and project init's footprint cascade (`internal/project/init.go:168-191`).
- Consumers of the code are fewer: `repo list`'s `root` output (`cmd/repo.go:280`), the staleness check's `git -C` (`set.go:206`, `internal/repo/git.go:77`, materialized clones only), and template prose that sends the agent to "resolved roots" via `repo list` or, for the project's own repo, to the directory it is running in.
- `RepoConfig` (`internal/config/repo.go:20-27`) has description, role, tags, deployment, knowledge, and changelog; the loader expands `${VAR}` over the whole file before parsing (`repo.go:61`) and unmarshals over `NewDefaultRepoConfig()` so absent keys take defaults. A default file serialises only knowledge and changelog (`internal/config/repo_test.go:52-66`).
- Registration (`cmd/repo.go:118-237`) upserts the registry entry, ensures the footprint, then loads repo.yaml and writes back only non-empty descriptive fields when they differ (`:202-228`).
- The roster rendered into instructions (`internal/repo/roster.go:18-40`) carries identity fields only and deliberately omits paths; it is injected by the plan and spec commands (`cmd/plan.go:137,206`, `cmd/spec.go:236,307`) and rendered by three templates (`templates/steps/plan/02-discovery.md:7-14`, `03-architecture.md:5-12`, `templates/steps/spec/00b-interview.md:13-20`). The implement command injects nothing (`cmd/implement.go:146-147,215-219`) and none of its callbacks pass an Extra (`internal/steps/implement/steps.go:76-186`).
- The implement FSM has thirteen states including `update_repo_changelog` (`steps.go:32-33`), whose template writes a release note to each changed repo's root `CHANGELOG.md`, defaulting the colocated repo to the project root (`templates/steps/implement/08-update_repo_changelog.md:3-30`). Step 10 already writes one project record and one derived record per affected repo (`10-update_feature_changelog.md:44-68`), attributing files by a `<repo>: ` prefix and defining unprefixed paths as "the colocated repo (the one that shares this project's working tree)" (`:31-32`; same rule at `07-update_changelog.md:32`).
- Stand-in phrases today: "not just the directory you are running in" (`02-discovery.md:28`, `templates/skills/workflows/spek-plan/SKILL.md:51`), the working-tree definition above, and roster fallbacks naming "the colocated repo" (`02-discovery.md:13`, `03-architecture.md:11`, `00b-interview.md:19`). Already correct: `templates/skills/skill_spawn-implementation-agents.md:7`, `skill_spawn-planning-agents.md:9`, `templates/skills/workflows/spek-implement/SKILL.md:41`.
- The project config already has a `source` key meaning a git address for changelog provenance (`internal/config/config.go:149`, `cmd/storefile.go:146`, `README.md:157`).
- Docs: repo.yaml reference is the "Repository configuration: repo.yaml" section of `docs:src/pages/configuration.mdx:206-298`; the `repos` key is `:183-202` (documents `address` at `:190`); the layout narrative lives in `docs:src/pages/projects.mdx:67-99`, the registration example at `:51-65` (address at `:56,62`), the `repo list` JSON at `:33-46` (address at `:41`), and the cloning section at `:118-129`; in-repo docs are `README.md:144-217` (address at `:180-181,190`) and `templates/skills/skill_manage-repos.md` (address at `:7,19,28,33`; served raw, pinned by `cmd/skill_test.go:48-68`).
- Tests: template-contract patterns in `templates/section_drafting_test.go:45-74` and `work_files_test.go:73-85`; roster render tests in `internal/steps/plan/steps_test.go:577-660`; address/local usages in `internal/repo/set_test.go` (17), `cmd/repo_test.go` (13), `cmd/init_test.go` (8), `internal/config/config_test.go` (5), `internal/project/init_test.go` (1); harbor implement suite oracles in `tests/harbor/implement-workflow/tests/test_implement_workflow.py:17-103` with fixtures in `environment/` (`config.yaml:17-21` uses `local`).

## Per-Phase Technical Notes

### Phase 1.1: Add `source` to the repo configuration

**File changes**:
- `internal/config/repo.go:20-27` — add `Source string \`yaml:"source,omitempty"\`` after `Deployment`; extend the type comment (`:15-19`) to name the two locations.
- `internal/config/repo.go` (new, after `WithDefaults` `:84-87`) — `type SourceKind int` with `SourceNone`, `SourceFile`, `SourceGit`; `func (c RepoConfig) ParseSource(configDir string) (SourceKind, string, error)`: empty → `SourceNone`; `file://` prefix stripped then treated as a path; `git://`, `ssh://`, `http://`, `https://`, `git+*://`, or scp-style (`^[^/@:]+@[^/:]+:`) → `SourceGit` with the value unchanged; anything else → `SourceFile`, `filepath.Clean` when absolute, `filepath.Join(configDir, v)` otherwise. Env vars are already expanded at `:61`, so no expansion here. Return an error only for an unsupported explicit scheme (e.g. `s3://`).
- `internal/config/repo_test.go:52-66` — keep the two-top-level-keys assertion unchanged (guards `omitempty`); add a case that a config with `Source` set serialises three keys.
- `internal/config/repo_test.go` (new tests) — `TestRepoConfig_ParseSource_{Absolute,Relative,EnvVar,FileScheme,GitSchemes,ScpStyle,Unset,UnsupportedScheme}` modelled on `:222-239` (`t.Setenv`, `t.TempDir`).
- `internal/config/config.go:147-149` — no change; add a comment on `Config.Source` noting it is a git address for provenance, distinct from `RepoConfig.Source`.

**Complexity**: Low
**Token estimate**: ~18k tokens
**Agent strategy**: Single agent, sequential execution

### Phase 1.2: Replace the registry's `local` and `address` with `location`

**File changes**:
- `internal/config/config.go:120-136` — `RepoEntry`: add `Location string \`yaml:"location,omitempty"\``; keep `Local string \`yaml:"local,omitempty"\`` documented as a deprecated alias that is folded into `Location` and never written; remove `Address`. Update the type comment.
- `internal/config/config.go:310-315` — `WithDefaults`: if `Location == "" && Local != ""` set `Location = Local`; clear `Local` so `ToYAMLFile` writes only `location`.
- `internal/config/config.go:286-306` — `validateRepos`: require `Location` (after alias folding) with `output.NewError("config_invalid", …).WithNextAction("set repos[i].location to the folder holding <name>'s .spektacular/")`; detect a leftover `address` key by decoding `repos` into a `[]map[string]any` alongside (or via a `yaml.Node` pass in `FromYAMLFile` `:216`) and fail with `WithNextAction("remove repos[i].address and set 'source: <address>' in <location>/.spektacular/repo.yaml")`. Apply defaults before validation so the alias case passes.
- `internal/config/config_test.go` (5 address/local usages) — rewrite fixtures to `location`; new tests: alias loads and round-trips as `location`; `address` rejected with the next_action text; missing location rejected.
- `internal/project/init.go:57-60` — seed `{Name: cfg.Name, Location: "."}`; `internal/project/init_test.go` and `cmd/init_test.go` (8 usages) — update fixtures and the expected written config.
- `cmd/knowledge.go:181-186` — synthesised entry uses `Location: "."`.
- `cmd/repo.go:133-138,322-327` — entry built from `name/location/dependencies`; `reposEqual` compares `Location` instead of `Address`/`Local` (full command changes in Phase 1.4).
- `.spektacular/config.yaml` (this repository) — `docs` entry: drop `address`, rename `local` to `location`; the docs repo stays colocated (its `.spektacular/` is inside the checkout), so no `source` is needed. `spektacular` entry: `local: .` → `location: .`.
- `README.md:180-190` — updated in Phase 4.2; leave for now.

**Complexity**: Medium
**Token estimate**: ~30k tokens
**Agent strategy**: 2 parallel agents (config model + tests; init/knowledge/repo-cmd entry sites), sequential integration

### Phase 1.3: Resolve the source once during repo resolution

**File changes**:
- `internal/repo/set.go:18-24` — add `Source string` to `ResolvedRepo`; document the root/source split and that `Materialized` now means "Source is a project-managed clone of a git source".
- `internal/repo/set.go:87-110` — `LocalRoot`: resolve `e.Location` (absolute or joined to `projectRoot`); drop the address/clone branch (the clone dir is now a *source*, not a root).
- `internal/repo/set.go:161-199` — `resolve(e)`: root from `Location` (error with next_action if missing); `cfg, err := loadFootprint(root)` (refactor of `checkFootprint` `:223-232` to return the parsed config); on `*FootprintError` return `ResolvedRepo{Root: root, Source: root}` with the error; else `kind, v, err := cfg.ParseSource(filepath.Join(root, ".spektacular"))`: `SourceNone` → `Source = root`; `SourceFile` → `Source = v`; `SourceGit` → `clone := filepath.Join(s.projectRoot, ".spektacular", MaterializeDirName, e.Name)`, `git.Clone(v, clone)` only when absent (`:184-189`), `Source = clone`, `Materialized = true`, `StaleNote = s.staleNote(clone, v)`.
- `internal/repo/set.go` (new, beside `LocalRoot`) — `func (s *Set) LocalSource(name string) (string, bool)`: `LocalRoot` + `config.RepoConfigFromYAMLFile` (missing/broken file → root); `SourceFile` → path; `SourceGit` → clone dir only if it exists, else `("", false)`; never invokes git. `Present` (`:78-81`) keeps meaning "root on disk".
- `internal/repo/set.go:11-14` — comment on `MaterializeDirName` now describes git sources.
- `internal/repo/set_test.go` (17 address/local usages; helpers `:13-47`, `fakeGit` `:51-90`, analogue `:344-358`) — rewrite fixtures to `Location`; new tests: file source absolute / relative to `.spektacular` / `file://`; git source clones on first resolve and not on second, `Materialized` true, `StaleNote` via `fakeGit`; footprint check still reads `<root>/.spektacular/repo.yaml` when source differs; `LocalSource` for file, cloned git, uncloned git, unset; missing location errors with next_action.
- `internal/repo/footprint.go:28-88` — no change; add a test asserting `EnsureFootprint` on a root whose repo.yaml declares a source scaffolds knowledge under the root.
- `internal/repo/git_integration_test.go` — no change (covers the exec runner).

**Complexity**: Medium
**Token estimate**: ~40k tokens
**Agent strategy**: 2 parallel agents (resolution code + tests), sequential integration

### Phase 1.4: Registration writes `source` and listing reports it as root

**File changes**:
- `cmd/repo.go:37-50` — `repoAddInputSchema`: remove `address`, add `location` and `source`; `Required: ["name","location"]`. `:90-99` — `repoAddInput`: `Location`, `Source`; remove `Address`.
- `cmd/repo.go:303-317` — `repoAddData`: require `location` with an error showing the payload shape; reject a raw `address` key (decode into `map[string]any` first) with `WithNextAction` pointing at `source`.
- `cmd/repo.go:133-138` — entry from `name/location/dependencies`; before resolution, `os.MkdirAll(resolvedLocation, 0755)` so a fresh repo folder can be registered (the footprint is created inside it by `EnsureFootprint` `:197`).
- `cmd/repo.go:211-228` — `if input.Source != "" { updated.Source = input.Source }`; write repo.yaml *before* the resolution that may clone, so a git source declared in the same `repo add` is honoured on first use (order: upsert registry → ensure footprint at the location → merge descriptive fields + source → resolve to materialize). `:329-341` — include `Source` in `repoConfigDescriptiveFieldsEqual`, exclude it from `repoConfigDescriptiveFieldsEmpty`.
- `cmd/repo.go:61-86,102-116` — `repoListOutputSchema`/`repoInfo`: remove `address`; add `location`; keep `root`; `:265` — `Location: e.Location`; `:280` — `info.Root = resolved.Source`; update the `Short` at `:29` and the comment at `:257-260`. Listing gates on `set.Present` (root on disk) and, for a git source not yet cloned, reports `root: ""` and `materialized: false` (no clone) via `resolved.Source` being empty when `LocalSource` is absent: implement by having `runRepoList` use `LocalSource` rather than `Resolve` for the root when the source is git and uncloned.
- `cmd/repo_test.go` (13 usages; helpers `:42-118`, `minimalFootprint` `:148-165`, schema tests `:534-574`, no-blanking test `:472`) — rewrite fixtures to `location`; new tests: add with location+source creates the folder and writes source (read repo.yaml back); re-add without source preserves it; add without location errors with the payload shape; add with `address` errors pointing at `source`; list reports the source as root (file absolute, relative, `${VAR}`, `file://`; git cloned → clone dir; git uncloned → empty root, zero git calls); list without source unchanged; schema has `location`/`source` and not `address`.
- `cmd/knowledge.go:197-231` — no change; new test in `cmd/knowledge_test.go` that a repo with a file source elsewhere aggregates knowledge from `<root>/.spektacular/knowledge` and that the colocated tolerance at `:208` holds when the colocated repo.yaml declares a source.
- `cmd/storefile.go:98-134` — no change; new test in `cmd/changelog_file_test.go` that `changelog file write --repo` for a separate-source repo lands under `<root>/<changelog dir>/<project>/`.
- `internal/project/init.go:168-191` — no change; test in `cmd/init_test.go` (using `snapshotDir` `:32-54`) that init with a registered separate-source repo leaves the source directory byte-identical.

**Complexity**: Medium
**Token estimate**: ~45k tokens
**Agent strategy**: 2-3 parallel agents (repo cmd + tests; knowledge/changelog/init regression tests), sequential integration

### Phase 2.1: Remove the root changelog step from the implement workflow

**File changes**:
- `internal/steps/implement/steps.go:32-33` — delete the `update_repo_changelog` row; `test_plan`'s `Src` becomes `[]string{"update_changelog"}`; `:19-21` and `:112-118` comments and `writeStep("update_changelog", "test_plan", …)`; delete `updateRepoChangelog()` (`:122-126`).
- `templates/steps/implement/08-update_repo_changelog.md` — delete; renumbering of `09-12` is optional.
- `templates/steps/implement/07-update_changelog.md:71-75` — goto target and sentence become `test_plan`.
- `templates/steps/implement/12-finished.md:9,19` — drop the two `CHANGELOG.md` bullets (replaced in Phase 2.2).
- `templates/skills/skill_update-changelog.md:24` — "The `update_feature_changelog` step then writes the project and per-repo changelog records."
- `internal/steps/implement/steps_test.go:77-95,100-131,133-214,390-406` — update step order, FSM walk, STOP map; delete `:379-386` and `:506-520`.
- `templates/context_directive_test.go:90-96` — floor stays 30; update the comment counts.
- `cmd/docs_test.go:45-67` — rewrite the comment at `:46-47`; keep the assertion.
- `tests/harbor/implement-workflow/tests/test_implement_workflow.py:53-68` — remove `"update_repo_changelog"` from `EXPECTED_STEP_ORDER` (full fixture update in Phase 3.3).
- `templates/skills/workflows/spek-implement/SKILL.md` — grep for `update_repo_changelog`/`CHANGELOG.md`; none expected, confirm.

**Complexity**: Low
**Token estimate**: ~20k tokens
**Agent strategy**: Single agent, sequential execution

### Phase 2.2: Fold the release note into the per-repo changelog record

**File changes**:
- `templates/steps/implement/10-update_feature_changelog.md:54-58` — the per-repo record's first body element becomes a 2-4 sentence user-facing summary "for a reader who has never seen the plan; no file paths, no internal package names" (wording lifted from the deleted `08-update_repo_changelog.md:14`), followed by the existing reference line and sections.
- `templates/steps/implement/12-finished.md:5-22` — summary bullet "A project changelog record was written to `{{changelog_path}}` and one derived record per affected repo to that repo's configured changelog directory"; the report list names both locations (per-repo path via `{{config.command}} changelog file read {{plan_name}}.md --repo <name>`).
- `internal/steps/implement/steps_test.go` — template assertions for the feature-changelog step gain the summary phrase; finished-step test (`:58-72` helper) asserts the new bullets and `NotContains("CHANGELOG.md")`.
- `tests/harbor/implement-workflow/instruction.md:107-124` — no change needed; confirm it names only the three records.

**Complexity**: Low
**Token estimate**: ~12k tokens
**Agent strategy**: Single agent, sequential execution

### Phase 3.1: Carry each repo's source in the roster and give it to the implement workflow

**File changes**:
- `internal/repo/roster.go:18-40` — add `"source": ""` to each entry; when `set.LocalSource(r.Name)` succeeds set it; update the comment at `:9-17` to record that the resolved source is now rendered (reversing the "paths always come from `repo list`" note) and why (refreshed on every `new`/`goto`).
- `internal/repo/roster_test.go:19-63` — assert the `source` key for file-source, cloned-git, uncloned-git, and absent repos, and that no git call is made.
- `cmd/implement.go:146-147` — after `wf.SetData("name", …)`, `wf.SetData("repos", repo.Roster(cfg, root, repoGit))`; `:215-219` — same on goto before the data loop (mirror `cmd/plan.go:137,206`).
- `internal/steps/implement/steps.go:76,82,88,94,100,145` — pass `stepkit.RepoRosterExtra(data)` instead of `nil` for read_plan, analyze, implement, test, verify, update_feature_changelog.
- `internal/steps/implement/steps_test.go:44-51` — add `renderStepWithData` copied from `internal/steps/plan/steps_test.go:53-61`; new tests modelled on `:577-660`: each of the six steps renders `name` + `source` from a roster of `[]any{map[string]any{...}}`, and the fallback renders without mustache residue.
- `cmd/implement_test.go` — new test: `implement new` and `goto` persist `repos` in `state.json` with `source`, using a fake git that records zero clones.

**Complexity**: Medium
**Token estimate**: ~30k tokens
**Agent strategy**: 2 parallel agents (roster + cmd injection; step callbacks + render tests), sequential integration

### Phase 3.2: Reword templates to direct the agent to each repo's source

**File changes**:
- `templates/steps/plan/02-discovery.md:9-11`, `03-architecture.md:7-9`, `templates/steps/spec/00b-interview.md:15-17` — roster line gains `{{#source}} (source: `{{source}}`){{/source}}`; fallbacks at `02-discovery.md:13`, `03-architecture.md:11`, `00b-interview.md:19` reworded to "the project's single repo; run `{{config.command}} repo list` for its source".
- `templates/steps/plan/02-discovery.md:28` — "Research across every repo identified in Step 1, in each repo's source listed above"; `:30` keep the `repo list` direction for repos whose source is not yet on disk (an uncloned git source is materialized by `repo list`'s resolution only when the repo is used; direct the agent to run `repo add` for the repo to clone it).
- `templates/skills/workflows/spek-plan/SKILL.md:51` — replace "not just the directory you are running in" with "in each repo's source, which the instructions list".
- `templates/steps/implement/01-read_plan.md` (top, before `:40`), `02-analyze.md`, `03-implement.md`, `04-test.md`, `05-verify.md` — add a shared roster block (same mustache as the plan steps) headed "**Where the code lives.** Carry out this step's work in each repo's source below; never assume the directory you started in is a repo's code" with the `{{^repos}}` fallback; the drift check at `01-read_plan.md:40-49` and the test/verify commands at `04-test.md:13-20`, `05-verify.md:13-18` say "run from that repo's source".
- `templates/steps/implement/07-update_changelog.md:32` and `10-update_feature_changelog.md:31-32` — prefix rule: "prefix every path with its repo's name whenever more than one repo is registered; an unprefixed path belongs to the only registered repo".
- `templates/steps/implement/10-update_feature_changelog.md:20-26` — roster block replaces the bare `repo list` step, keeping `repo list` as the fallback.
- `templates/skills/skill_spawn-implementation-agents.md:7`, `skill_spawn-planning-agents.md:9` — "resolved root" becomes "source (reported as `root` by `repo list`)"; `skill_gather-project-metadata.md:114-125` — note the commands run in the target repo's source.
- `templates/repo_source_test.go` (new) — modelled on `templates/section_drafting_test.go:45-74` + `work_files_test.go:73-85`: banned list `{"directory you are running in", "shares this project's working tree", "root `CHANGELOG.md`", "project root's `CHANGELOG.md`", "update_repo_changelog"}` walked over `steps/` and `skills/`; positive: the six implement templates and the three roster templates contain `{{#repos}}` and `{{source}}`; floor on files visited.
- `internal/agent/instruction_surface_test.go:111-153` — extend the rendered-skill assertions for spek-plan/spek-implement with the new wording and `NotContains("running in")`.
- `internal/steps/plan/steps_test.go:577-660`, `internal/steps/spec/steps_test.go:478-568` — roster data gains `source`; assert it renders.
- `tests/harbor/plan-workflow/tests/test_plan_workflow.py:88-131` — confirm no oracle embeds the reworded phrases (none expected).

**Complexity**: Medium
**Token estimate**: ~45k tokens
**Agent strategy**: 2-3 parallel agents (plan/spec/skill templates; implement templates; contract test), sequential integration

### Phase 3.3: Prove the separate layout end to end in the harbor implement suite

**File changes**:
- `tests/harbor/implement-workflow/environment/config.yaml:17-21` — `repos: [{name: auth, location: .}, {name: docs, location: ./repos/docs}]`.
- `tests/harbor/implement-workflow/environment/docs-repo.yaml` — add `source: file:///opt/docs-repo`; `environment/Dockerfile:40-41` — copy it to `/app/repos/docs/.spektacular/repo.yaml` and create `/app/repos/docs/.spektacular/{knowledge,changelog}`; keep `/opt/docs-repo` as a bare code checkout with no `.spektacular/`.
- `tests/harbor/implement-workflow/tests/test_implement_workflow.py:46-103` — `EXTERNAL_REPO_CHANGELOG_PATH = /app/repos/docs/.spektacular/changelog/auth-service/<plan>.md`; `EXTERNAL_WRITE_SUBSTR` unchanged (`--repo docs`); add `EXTERNAL_REPO_SOURCE = /opt/docs-repo` and two tests: no `/opt/docs-repo/.spektacular` exists and no `CHANGELOG.md` exists under `/opt/docs-repo` or `/app` after the run; `:53-68` step order already updated in Phase 2.1.
- `tests/harbor/implement-workflow/instruction.md:107-124` — success criteria name the new external record path and the clean-source condition.
- `tests/harbor/implement-workflow/task.toml:20-26` — add `/app/repos/docs/.spektacular` to the artifact copies.
- Run the implement harbor suite once; record the run outcome in the plan's changelog entry.

**Complexity**: Medium
**Token estimate**: ~25k tokens plus one ~25-minute harbor run
**Agent strategy**: Single agent, sequential execution

### Phase 4.1: Document `source`, `location`, and the layouts on the documentation site

**File changes**:
- `docs:src/pages/configuration.mdx:4,15` — mention `source` and `location` in the page description and hero list.
- `docs:src/pages/configuration.mdx:26-28` — env-var paragraph covers `repo.yaml`.
- `docs:src/pages/configuration.mdx:33-71` — config.yaml example: `repos` entries use `location`; drop any `address`.
- `docs:src/pages/configuration.mdx:183-202` — rewrite the `repos` ConfigKey per the plan's content outline (`location`, `local` alias, `address` removed, pointer to `source`).
- `docs:src/pages/configuration.mdx:223-242` — repo.yaml example gains `source:`; `:277-282` — insert the `source` `ConfigKey` after `deployment`; `:79-81` fix "Eight top-level sections" to nine in passing.
- `docs:src/pages/projects.mdx:33-46` — `repo list` JSON: replace `address`/`local` with `location`; `:51-65` — registration example per the outline; `:69-79` — extend the sub-paragraph; `:83-89` — `repos` key lists `name` and `location`; add a `source` `ConfigKey`; after `:99` — new `Section heading="Colocated or separate"` with `surface={false}` (the preceding `ConfigurationKeys` defaults to `true`), three fenced trees and the two `repo add` examples per the outline; re-alternate `surface` on the following sections (`:101`, `:118`, `:131`); `:118-129` — "registered by remote address" becomes "whose `source` is a git location".
- Verify: `npm run build`, `npx astro check` (`Makefile:1-22`), and `grep -nE "<div|<section|class=" src/pages/*.mdx` returns nothing.

**Complexity**: Medium
**Token estimate**: ~25k tokens
**Agent strategy**: Single agent, sequential execution

### Phase 4.2: Update the README and the repo-management skill

**File changes**:
- `README.md:146` — sentence on colocated vs separate; `:176-190` — registry example and paragraph per the content example (`location`, no `address`); `:192-211` — repo.yaml block gains `source:` and the paragraph per the content example; `:157` — comment on the project-level `source` clarified as a git address; `:79-91` — add `repo.yaml` and `changelog/` to the structure tree (existing drift).
- `templates/skills/skill_manage-repos.md:7` — Concepts paragraph per the content example; `:11-20` — two payloads and the new bullets (`:19` "A `local` path must already exist" becomes the `location` created-if-missing bullet); `:28` — `root` is the resolved source; `:33` — git-source materialization wording. Keep "repo add", "repo list", "never fetches or pulls" and no `{{command}}` (`cmd/skill_test.go:48-68`).
- `cmd/skill_test.go:48-56` — add assertions that the skill mentions `"location"`, `"source"`, `file://`, a git URL form, and both layouts, and does not mention `"address"`.
- Grep both files for em dashes after editing.

**Complexity**: Low
**Token estimate**: ~15k tokens
**Agent strategy**: Single agent, sequential execution

## Testing Strategy

- **Phase 1.1**: unit tests in `internal/config/repo_test.go` for `ParseSource` in each form (absolute, relative, `${VAR}`, `file://`, git schemes, scp-style, unset, unsupported scheme) and for `omitempty` (two keys without source, three with).
- **Phase 1.2**: unit tests in `internal/config/config_test.go` for the `local` alias (loads, saves as `location`), `address` rejection with next_action, missing location; init tests assert the seeded `location: .`.
- **Phase 1.3**: unit tests in `internal/repo/set_test.go` with the fake git runner: file source resolved on the local path, git source cloned once and reused, `Materialized` and `StaleNote` set for git sources, footprint still read from the root, `LocalSource` never calls git and is absent for an uncloned git source; a footprint test that scaffolding stays under the root.
- **Phase 1.4**: command tests in `cmd/repo_test.go` (add creates the location folder and writes source, re-add preserves, missing location and `address` payload errors, list reports the source as root in every form, zero git calls for listing, schema); regression tests in `cmd/knowledge_test.go`, `cmd/changelog_file_test.go`, and `cmd/init_test.go` proving root-bound consumers stay on the root and the source directory is untouched.
- **Phase 2.1**: step-order, FSM-walk, and STOP-map tests in `internal/steps/implement/steps_test.go` updated; deleted-template tests removed; harbor `EXPECTED_STEP_ORDER` updated; `templates/context_directive_test.go` floor confirmed.
- **Phase 2.2**: template assertions for the feature-changelog and finished steps (summary phrase present, `CHANGELOG.md` absent).
- **Phase 3.1**: `internal/repo/roster_test.go` asserts the `source` key per source kind; `internal/steps/implement/steps_test.go` gains `renderStepWithData` and per-step roster render and fallback tests; `cmd/implement_test.go` asserts `repos` persisted in state without cloning.
- **Phase 3.2**: new `templates/repo_source_test.go` (banned phrases over `steps/` and `skills/`, positive roster assertions with a visited-files floor); `internal/agent/instruction_surface_test.go` extended for rendered skills; plan and spec roster tests assert `source`.
- **Phase 3.3**: harbor implement run with the separate-source fixture; new verifier tests for the absence of `.spektacular/` and `CHANGELOG.md` under the source. Manual, captured in the implementation test plan: an agent following the repo-management skill registers a separate-source repo first try.
- **Phase 4.1**: `npm run build`, `npx astro check`, and the MDX layout-markup grep in the docs repo.
- **Phase 4.2**: `cmd/skill_test.go` assertions for `location`, `source`, both kinds, both layouts, and no `address`; em-dash grep over README and skill.
- Every phase ends with `go test ./...` passing in full.

## Project References

- Spec: `go run . spec file read 000046_relocatable-repo-footprint.md` (updated at the plan walkthrough: registry `location`, single `source` with file and git kinds, `address` removed)
- Knowledge: `go run . knowledge always-applied --repo spektacular --repo docs`; gotcha `gotchas/repoconfig-must-start-from-default.md` (construct `RepoConfig` from `NewDefaultRepoConfig()`); architecture `architecture/testing-architecture.md` (harbor oracles move with the surfaces they mirror)
- Repos: `go run . repo list` (spektacular at the project root; docs at `../spektacular-website`); this repository's own `.spektacular/config.yaml` must be updated in Phase 1.2
- Prior plans for the resolved-repo model and registration write-back: `go run . plan file read 000039_project-level-capabilities/plan.md`, `go run . plan file read 000042_repo-self-describing-metadata/plan.md`
- Docs site verification: `make build`, `make check` in the docs repo (`Makefile:1-22`); MDX guard `grep -nE "<div|<section|class=" src/pages/*.mdx`
- Harbor implement suite: `tests/harbor/implement-workflow/` (README/Makefile target for running it; ~25 minutes, needs Docker and Claude credentials)

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|-------------|----------------|
| Low | ~10k | Single agent, sequential |
| Medium | ~25k | 2-3 parallel agents |
| High | ~50k+ | Parallel analysis, sequential integration |

Phases 1.1, 2.1, 2.2, 4.2 are Low; 1.2, 1.3, 1.4, 3.1, 3.2, 3.3, 4.1 are Medium. Phase 3.3 additionally costs one ~25-minute harbor run outside the agent's context.

## Migration Notes

- **Registry `address`**: hard break, no migration, by the user's decision. A config that still carries it fails to load with an error naming the repo and the `source` line to add to its repo.yaml. This repository's own `.spektacular/config.yaml` is updated by hand in Phase 1.2.
- **Registry `local`**: accepted as an alias for `location` indefinitely; folded on load, written back as `location` when registration saves the file.
- **Repo `source`**: none required. An absent `source` loads as the zero value through the existing defaulting loader, and the one-shot migration detector (`cmd/version.go:148-177`) keys only on the presence of `config.yaml` without `repo.yaml`. Users who want the separate layout move their files by hand and set `source` (spec Non-Goal).

## Performance Considerations

Resolution parses repo.yaml once per repo as it already does for footprint validation; a git source clones once, on first use, exactly as address-registered repos did before. The roster's source lookup adds one file read per registered repo on each `new`/`goto`, never a git call. No measurable change.
