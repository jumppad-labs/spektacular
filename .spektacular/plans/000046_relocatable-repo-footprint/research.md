---
created_date: "2026-09-02"
status: completed
closed_date: "2026-09-02"
---

# Research: 000046_relocatable-repo-footprint

## Alternatives considered and rejected

- **Per-repo `footprint: none` opt-out that moves repo metadata/knowledge/changelog into the project config.** Rejected during the spec interview and recorded as a spec Non-Goal: it inverts the "repo owns its own metadata, no back-pointer to any project" invariant (spec 000039 Requirements; `internal/config/repo.go:15-19`) and would re-add descriptive fields to `config.RepoEntry` (`internal/config/config.go:129-136`), which spec 000042 deliberately stripped.
- **Keeping `address` in the project registry alongside a repo-level `source`.** The first plan draft kept the registry's `local`/`address` untouched and added only `source` to repo.yaml. Rejected by the user at the walkthrough: with `address` in the registry, a remote code repo is cloned into `.spektacular/repos/<name>/` and its footprint has to live inside that clone (`internal/repo/set.go:184-198` clones then `checkFootprint`s the same directory), so the code repository still needs Spektacular files checked in. Moving the git origin into the repo folder's own `source` lets the project folder own everything.
- **Two repo-level fields, `source` for the path and `address` for the git URL.** Rejected by the user in favour of a single `source` distinguished by scheme (`file://` versus `git://`, `ssh://`, `https://`, scp-style), so one key answers "where is the code" for both layouts.
- **A migration for registry `address`.** Rejected by the user (hard break, no migration): nothing has shipped with the key, the value is not lost (the error names the `source` line to add), and the existing detector is a one-shot config.yaml/repo.yaml split check (`cmd/version.go:148-177`), not a versioned framework, so extending it would be new machinery.
- **A prescribed project-hosted layout** (`.spektacular/repos/<name>/` holding the footprint with the clone under `.spektacular/repos/<name>/source/`, gitignore narrowed to `repos/*/source/`). Rejected by the user during the spec interview ("the structure can be flexible"); the spec keeps `.spektacular/repos/` materialization and `templates/.spektacular/.gitignore` untouched (`internal/repo/set.go:11-14,184-189`).
- **Changing the anchor or defaults of the knowledge/changelog locations** (e.g. `knowledge` instead of `.spektacular/knowledge`, anchored at the repo.yaml directory). Rejected: it forces a path migration of every colocated repo.yaml for no functional gain; `DefaultKnowledgeLocation`/`DefaultChangelogDir` (`internal/config/config.go:45-53`) and their consumers (`cmd/knowledge.go:227-231`, `internal/repo/footprint.go:61-64`, `cmd/storefile.go:132-133`) stay as they are.
- **Making `ResolvedRepo.Root` itself become the source directory.** Rejected: every consumer of the repo's own Spektacular files derives its path from `Root` (`internal/repo/set.go:224` checkFootprint, `:125` DescriptiveMetadata, `cmd/knowledge.go:203`, `cmd/storefile.go:128`, `cmd/repo.go:197,206`, `internal/project/init.go:178-188`), and `source` is itself read from `<Root>/.spektacular/repo.yaml`, so redefining `Root` creates a circular resolution and breaks footprint validation for every repo with a source. The resolved repo must carry two paths.
- **Widening env-var expansion to bare `$VAR` / `os.ExpandEnv`.** Rejected: expansion runs on the whole raw file before unmarshal (`internal/config/repo.go:61`, `internal/config/config.go:413-419`) so any widening changes semantics for every existing value with a `$`; the existing `${VAR}` form already satisfies the spec's "environment variable references" and needs no code.
- **Passing repo sources through `workflow.Config`.** Rejected: plan 000039 admitted exactly one scalar (`ProjectName`) and stated "registry data never flows through workflow config"; the roster already reaches templates through workflow data (`cmd/plan.go:137,206`, `cmd/spec.go:236,307` → `wf.SetData("repos", …)`; `internal/stepkit/stepkit.go:129-140` `RepoRosterExtra`). Extending that data with each repo's source is the established path.
- **Keeping the `update_repo_changelog` step and redirecting its release note into the repo's changelog directory.** Rejected: the spec requires exactly two changelogs (project + one per affected repo), and the per-repo derived record already exists and is written by step 10 (`templates/steps/implement/10-update_feature_changelog.md:54-68`, `cmd/storefile.go:98-134`). A third file duplicates it. Plan 000015 introduced the root `CHANGELOG.md` as a plan-level addition beyond its spec, and plan 000039's research flagged the step as "assumes exactly one repo root, most impacted". Remove the step.
- **Go-level enforcement in `finished()` that no root `CHANGELOG.md` was written.** Rejected for this plan: `workflow.Config` carries no repo set and the implement store is project-rooted (`internal/steps/implement/steps.go:159-187`), so the check would need new plumbing; the spec's Technical Approach asks for a template-rendering test instead, and the harbor implement suite already asserts no built-in-tool changelog writes (`tests/harbor/implement-workflow/tests/test_implement_workflow.py:308-325`).
- **Validating a file `source` at registration (exists, is a git working tree).** Rejected: the spec dropped missing-path and validation requirements as "nothing requirements", and non-git sources are a Non-Goal, not a validation target. `repo add` writes the field like description/role (`cmd/repo.go:202-228`); git operations fail naturally if the path is wrong.
- **Rendering run-time data into library skills (`skill_*.md`).** Not possible without a new mechanism: `cmd/skill.go:49-64` serves them raw; only `skills/workflows/*/SKILL.md` is rendered and only with `{"command"}` (`internal/agent/skills.go:47`). Skills keep prose ("run `repo list`, work in each repo's source"); run-time provision is done in step templates.

## Chosen approach — evidence

- **Field placement**: `RepoConfig` (`internal/config/repo.go:20-27`) gains `Source string \`yaml:"source,omitempty"\``. `omitempty` is mandatory: `internal/config/repo_test.go:52-66` asserts a default repo.yaml serialises exactly two top-level keys. Loading unmarshals over `NewDefaultRepoConfig()` (`repo.go:63-64`) so an absent key is the zero value; `Validate` (`repo.go:74-79`) needs no change for the field itself. No migration: `cmd/version.go:148-177` detects only "config.yaml present, repo.yaml missing" and spec 000045 excludes registered repos.
- **Source kinds**: the value is classified by scheme; `file://` and scheme-less values are paths, `git://`/`ssh://`/`http(s)://`/scp-style are git URLs. The git runner's `Clone(url, dir)` already takes any URL git accepts (`internal/repo/git.go:71-74`, `git clone -- url dir`), and `RemoteHead(url)` (`:80-91`) drives the staleness notice, so a git source reuses the existing clone-into-`.spektacular/repos/<name>/` path (`internal/repo/set.go:181-198`) with the URL coming from repo.yaml instead of the registry.
- **Registry shape**: `RepoEntry` (`internal/config/config.go:129-136`) currently has `Address` and `Local`; `validateRepos` (`:286-306`) requires one of them and is the natural place to reject `address` and require `location`; `WithDefaults` (`:310-315`) is the natural place to fold the `local` alias. `ToYAMLFile` writes the struct, so clearing `Local` after folding guarantees only `location` is written. Entry sites: `cmd/repo.go:133-138,322-327`, `internal/project/init.go:57-60`, `cmd/knowledge.go:181-186`, and this repository's own `.spektacular/config.yaml`.
- **Env-var expansion is free**: `RepoConfigFromYAMLFile` expands `${VAR}` over the raw text before parsing (`repo.go:61`; pattern `internal/config/config.go:13`). Test template: `internal/config/repo_test.go:222-239` (`t.Setenv`).
- **Relative anchoring**: the spec fixes the anchor for file sources as the directory holding repo.yaml, i.e. `<Root>/.spektacular`. `RepoConfigFromYAMLFile(path)` knows only `path` (`repo.go:55`), so the resolver is a method taking the config directory (pattern precedent: each consumer anchors its own relative paths, `cmd/knowledge.go:229-231`, `footprint.go:61-64`).
- **Two paths on the resolved repo**: `ResolvedRepo` (`internal/repo/set.go:18-24`) keeps `Root` (directory holding `.spektacular/`, used by every Spektacular-files consumer listed above) and gains a resolved absolute `Source` (file path, git clone, or `Root`). `resolve()` already parses repo.yaml in `checkFootprint` (`set.go:223-232`) on every path, so source can be derived there without an extra read. `LocalRoot` (`set.go:87-110`) stays git-free and footprint-only; `DescriptiveMetadata` (`set.go:120-130`) is the single reader of repo.yaml for listing/roster.
- **Consumers that must move to `Source`** (code location): `repo list` root output (`cmd/repo.go:280`, schema `:72`, struct `:105`), `staleNote`'s `git -C` (`set.go:206` → `git.go:77`), the roster (`internal/repo/roster.go:18-40`), and every template instruction that sends the agent into a repo. Consumers that must stay on `Root`: `checkFootprint`, `DescriptiveMetadata`, `EnsureFootprint` (`footprint.go:28-88`), `cmd/repo.go:197,206`, `cmd/knowledge.go:197-231` (including the `root != projectRoot` colocated tolerance at `:208`), `cmd/storefile.go:128-133`, `internal/project/init.go:168-191`.
- **`.spektacular_ignore`**: loaded at the store root by `NewSourceStore` (`internal/store/ignore.go:76`); repo-routed stores stay rooted at `Root`, so the ignore file keeps living beside repo.yaml.
- **Registration writes the field**: `repoAddInput` (`cmd/repo.go:90-99`), `repoAddInputSchema` (`:37-50`), `repoAddData` (`:303-317`), the create-then-update write-back (`:202-228`, empty input never blanks an existing value, tested at `cmd/repo_test.go:472`), and `repoConfigDescriptiveFieldsEqual` (`:329-341`). `EnsureFootprint`'s caller-supplied config is ignored once repo.yaml exists (`footprint.go:48-51`), so the update path is the only correct writer (plan 000042 "single resolver, two consumers"). `EnsureFootprint` uses `os.MkdirAll` (`footprint.go:35`), so a missing location folder is created for free once resolution stops requiring it to pre-exist.
- **Run-time provision to the agent**: roster entries gain a resolved `source` (via `Set.LocalSource`, no git). Existing roster render sites: `templates/steps/plan/02-discovery.md:7-14`, `03-architecture.md:5-12`, `templates/steps/spec/00b-interview.md:13-20`; data plumbing `internal/stepkit/stepkit.go:129-140`, `internal/steps/plan/steps.go:97,103`, `internal/steps/spec/steps.go:104`. The implement workflow has no roster today: `cmd/implement.go:147,212` sets only `name`; none of the 12 callbacks in `internal/steps/implement/steps.go` pass an Extra. Implement steps that touch code and must gain the roster: `01-read_plan.md:40-49` (drift check against the working tree), `02-analyze.md`, `03-implement.md`, `04-test.md`, `05-verify.md`, `10-update_feature_changelog.md:23-32`. Test scaffolding to copy: `internal/steps/plan/steps_test.go:53-61` `renderStepWithData` and `:577-660` roster tests; `internal/repo/roster_test.go:19-63`.
- **Stand-in phrases to remove**: "not just the directory you are running in" (`templates/steps/plan/02-discovery.md:28`, `templates/skills/workflows/spek-plan/SKILL.md:51`); "the project's own colocated repo (the one that shares this project's working tree)" (`10-update_feature_changelog.md:32`, `07-update_changelog.md:32`); "the project root's `CHANGELOG.md`" (`08-update_repo_changelog.md:7`); roster fallbacks "research the colocated repo" / "all work targets the colocated repo" (`02-discovery.md:13`, `03-architecture.md:11`, `00b-interview.md:19`). Already-correct wording to keep: `templates/skills/skill_spawn-implementation-agents.md:7`, `skill_spawn-planning-agents.md:9`, `templates/skills/workflows/spek-implement/SKILL.md:41`.
- **Removing step 08** touches: FSM table `internal/steps/implement/steps.go:32-33` (`test_plan`'s only Src is `update_repo_changelog`) and `:122-126`; goto branch `templates/steps/implement/07-update_changelog.md:71-75`; `12-finished.md:9,19`; `templates/skills/skill_update-changelog.md:24`; tests `internal/steps/implement/steps_test.go:77-95,100-131,133-214,379-386,390-406,506-520`; harbor `tests/harbor/implement-workflow/tests/test_implement_workflow.py:53-68` EXPECTED_STEP_ORDER; the comment in `cmd/docs_test.go:46-47`. The `templates/context_directive_test.go:95` floor of 30 survives (33 directive-bearing templates today, 42 step templates total).
- **Template-contract test to add** (spec Technical Approach): model on `templates/section_drafting_test.go:45-74` (banned-phrase list + `NotContainsf`) walking `steps/` and `skills/` like `templates/work_files_test.go:73-85`; rendered-install variant in `internal/agent/instruction_surface_test.go:111-153`.
- **Harbor**: the implement suite seeds `repos: [{auth, local: .}, {docs, local: /opt/docs-repo}]` (`tests/harbor/implement-workflow/environment/config.yaml:17-21`) with footprints at `/app/.spektacular` and `/opt/docs-repo/.spektacular` (`Dockerfile:25-41`) and hand-maintained path oracles (`test_implement_workflow.py:46-103`). It is the natural place for a separate-source variant (docs footprint under `/app/repos/docs/.spektacular/repo.yaml` with `source: file:///opt/docs-repo`). `instruction.md:107-124` never mentions a root `CHANGELOG.md`.
- **Docs surfaces**: docs repo `src/pages/configuration.mdx:206-298` (repo.yaml section: example at `:223-242`, `ConfigurationKeys` at `:246-298`, ConfigKey shape at `:277-282`), `:183-202` (project `repos` key; documents `address` at `:190`), `:26-28` (env-var wording currently claims only config.yaml), `src/pages/projects.mdx:33-46` (`repo list` JSON with `address`), `:51-65` (`repo add` example with `address`), `:67-99` ("How configuration is split"), `:118-129` (cloning "by remote address"); verify with `npm run build` and `npx astro check` (`Makefile:1-22`; CI runs only build). Spektacular repo: `README.md:144-152,176-211` (Configuration, registry example with `address` at `:180-181,190`, repo.yaml block), `README.md:79-91` (only directory-tree precedent); `templates/skills/skill_manage-repos.md` (`:7` Concepts with `address`/`local`, `:11-20` `repo add` payload, `:19,28,33` address wording; pinned by `cmd/skill_test.go:48-68`: must keep "repo add", "repo list", "never fetches or pulls", and no `{{command}}`).
- **Naming collision to document**: project `config.yaml` already has `source:` meaning a git remote URL for provenance (`internal/config/config.go:149`, `cmd/storefile.go:146`, `README.md:157`, rendered at `10-update_feature_changelog.md:60`). Repo-level `source` is a code location. Docs must state the difference once.

## Files examined

- spektacular:internal/config/repo.go:20-27,31-51,55-71,74-79,84-87 — RepoConfig shape, defaults, whole-file `${VAR}` expansion, no path anchoring in the loader
- spektacular:internal/config/config.go:13,45-53,87-97,120-136,147-159,286-315,384-399,413-419 — defaults, ChangelogConfig, RepoEntry (address/local), project `Source` (URL), validateRepos, WithDefaults, knowledge defaulting, expandEnvVars
- spektacular:internal/config/repo_test.go:52-66,222-239 — two-top-level-keys assertion; env-var test template
- spektacular:internal/config/config_test.go — five address/local fixtures to rewrite
- spektacular:internal/repo/set.go:11-14,18-24,87-110,120-130,161-199,205-232 — MaterializeDirName, ResolvedRepo, LocalRoot, DescriptiveMetadata, resolve (local then clone), staleNote (git -C Root), checkFootprint
- spektacular:internal/repo/footprint.go:28-88 — EnsureFootprint; MkdirAll; loaded config wins; knowledge scaffolding anchored at root
- spektacular:internal/repo/git.go:20-91 — GitRunner surface; Clone takes any URL; LocalHead runs `git -C dir`; RemoteHead by URL
- spektacular:internal/repo/roster.go:18-40 — roster keys; deliberately omits paths
- spektacular:internal/repo/set_test.go:13-90,344-358 — helpers, fakeGit, relative-local test analogue; 17 address/local usages
- spektacular:cmd/repo.go:37-116,118-237,239-301,303-341 — schemas, input/info structs, repo add write-back, repo list root, repoAddData, equality helpers
- spektacular:cmd/repo_test.go:18-165,325,472,534-574 — mirror structs, helpers, list/add tests, schema tests; 13 address/local usages
- spektacular:cmd/knowledge.go:171-231 — aggregation anchors knowledge at repo root; synthesised `Local: "."` entry; `root != projectRoot` tolerance
- spektacular:cmd/storefile.go:86-150,219-225,344-349 — project vs repo-routed changelog stores; provenance uses project Source
- spektacular:cmd/changelog_file.go:8-26 — changelog store wiring (repoRouted)
- spektacular:cmd/implement.go:143-147,212-219 — no roster injection today
- spektacular:cmd/plan.go:137,206; cmd/spec.go:236,307 — roster injection sites
- spektacular:cmd/version.go:61-74,148-177,179-282 — migration is a one-shot legacy detector; execute/scan are test-only
- spektacular:cmd/skill.go:49-64 — library skills served raw
- spektacular:cmd/skill_test.go:48-68,82,97 — manage-repos content pins
- spektacular:cmd/docs_test.go:45-67 — asserts root CHANGELOG.md keeps 000039 section; comment references step 08
- spektacular:cmd/init_test.go — eight address/local usages; snapshotDir at :32-54
- spektacular:internal/project/init.go:57-60,70-80,140-144,168-191 — seeds `{name, local: .}`, writes default repo.yaml only if absent, footprint cascade
- spektacular:internal/stepkit/stepkit.go:55-94,121-140 — render pipeline, Extra merge order, RepoRosterExtra
- spektacular:internal/steps/implement/steps.go:19-33,76-186 — FSM table, callbacks with nil Extra, step 08 callback, finished post-condition
- spektacular:internal/steps/implement/strategy.go:29-66 — changelog path vars
- spektacular:internal/steps/implement/steps_test.go:22-72,77-214,379-406,506-520 — renderStep (no data variant), order tests, step 08 tests
- spektacular:internal/steps/plan/steps.go:97,103; steps_test.go:45-61,577-660 — roster render + tests
- spektacular:internal/steps/spec/steps.go:104; steps_test.go:478-568 — interview roster
- spektacular:internal/store/ignore.go:76-80 — ignore file loaded at store root
- spektacular:internal/agent/skills.go:47; instruction_surface_test.go:21-160 — SKILL.md render data; banned-substring test pattern
- spektacular:templates/steps/implement/01-read_plan.md:40-49, 02-analyze.md:25-31, 03-implement.md, 04-test.md:13-20, 05-verify.md:13-18, 07-update_changelog.md:32,71-75, 08-update_repo_changelog.md:3-30, 10-update_feature_changelog.md:5-8,20-68, 12-finished.md:5-22 — cwd assumptions and changelog steps
- spektacular:templates/steps/plan/02-discovery.md:7-14,28,30,48; 03-architecture.md:5-14; 10-phases.md:13,23; 14-verification.md:52 — roster and attribution
- spektacular:templates/steps/spec/00b-interview.md:13-20 — roster
- spektacular:templates/skills/skill_manage-repos.md:1-42; skill_spawn-implementation-agents.md:7; skill_spawn-planning-agents.md:9; skill_update-changelog.md:20-26; skill_gather-project-metadata.md:114-125; workflows/spek-plan/SKILL.md:51; workflows/spek-implement/SKILL.md:41
- spektacular:templates/work_files_test.go:12-118; section_drafting_test.go:45-74; context_directive_test.go:17-97 — template-contract test patterns
- spektacular:tests/harbor/implement-workflow/{tests/test_implement_workflow.py:17-103,215-360, environment/Dockerfile:25-41, environment/config.yaml:17-21, instruction.md:107-124, task.toml:20-26} — oracles and fixtures
- spektacular:tests/harbor/plan-workflow/tests/test_plan_workflow.py:61-190 — plan-suite oracles (unaffected unless roster wording changes)
- spektacular:README.md:75-93,144-217 — structure tree, Configuration section (registry `address` at 180-181,190)
- docs:src/pages/configuration.mdx:1-28,33-71,183-202,206-298 — repo.yaml reference, `repos` key (address at 190), env-var note
- docs:src/pages/projects.mdx:18-99,118-129 — layout narrative, `repo list` JSON (address at 41), repo add example (address at 56,62), config split, cloning section
- docs:src/components/sections/ConfigKey.astro:2-19; Section.astro:1-16; ConfigurationKeys.astro:1-9 — component props
- docs:package.json:7-21; Makefile:1-22; .github/workflows/deploy.yml — build/check commands

## External references

- yaml.v3 `omitempty` semantics (already relied on by `RepoConfig`'s descriptive fields): an unset `source` must not appear in a freshly written repo.yaml, and a cleared `Local` must not be written back.
- git URL forms accepted by `git clone` (`git://`, `ssh://`, `http(s)://`, `file://`, scp-style `user@host:path`): the source classifier mirrors these so any URL git accepts is a git source, with `file://` reserved for the on-disk case.
- cbroglie/mustache (used by `internal/stepkit`): section rendering `{{#repos}}…{{/repos}}` iterates `[]any` of `map[string]any`, which is the state.json round-trip shape the tests emulate.

## Prior plans / specs consulted

- spec/plan 000039_project-level-capabilities (+research.md) — repo footprint is minimal and project-agnostic; `ResolvedRepo.Root` is "the only access handle"; resolved paths deliberately kept out of rendered instructions as stale-at-render-time (this plan reverses that call for sources); `workflow.Config` admits no registry data; step 08 flagged as "assumes exactly one repo root"; registry `address` and clone-on-first-use introduced here (this plan moves the origin into repo.yaml and keeps the clone location).
- spec/plan 000042_repo-self-describing-metadata (+research.md) — registration writes descriptive fields via create-then-update; single resolver for list + roster; docs live in the docs repo using ConfigKey.
- spec/plan 000045_config-file-migration — migration is file-existence detection only; optional keys need none; nothing versioned to extend for the `address` break.
- spec/plan 000044_projects-feature-documentation — projects.mdx outline; config split explained on one page; README stays a front door.
- spec/plan 000033_spec-workflow-output-changelog and plan 000015_implementation — origin of the three changelogs; root CHANGELOG.md was a plan-level addition beyond spec.
- plan 000041, 000037, 000029 — prose-in-templates + phrase-assertion tests as the enforcement pattern; README altitude.
- spec 000046_relocatable-repo-footprint — the source of truth for this plan, updated at the walkthrough to the `location`/`source` shape.

## Open assumptions

- A git `source` is cloned into `.spektacular/repos/<name>/` and never fetched or pulled, exactly as address-registered repos were; the clone location is not configurable (spec Non-Goal).
- Scheme-less `source` values are file paths. Only `file://` and the git URL forms are recognised schemes; an unknown scheme is a load error with remediation. If a user relies on another git transport form, STOP and ask.
- A relative file `source` resolves against the directory holding repo.yaml (`<Root>/.spektacular`), per the spec, even though knowledge/changelog relative paths resolve against `Root`. The docs state both anchors explicitly.
- The project's own colocated repo (registry entry `location: .`, or the implicit repo-of-one) may declare `source` too; when it does, `repo list` reports that source as its root and `cmd/knowledge.go:208`'s colocated tolerance keeps comparing footprint roots, not sources.
- `repo add` requires `location` and creates the folder when missing; resolution of an already-registered entry whose location is missing errors with remediation rather than creating it.
- Removing `update_repo_changelog` (rather than renaming) is acceptable; the release-note role is folded into the per-repo derived record written by step 10.
- The harbor implement suite is extended with a separate file-source repo and run once by the implementer; the git-source path is covered by unit tests with the fake runner and the existing git integration test; the plan suite needs no fixture change.
- `repo list` reports a resolved file source without stat-ing it; a nonexistent source surfaces when git or the agent touches it.

## Drafting assumptions

### Two paths on the resolved repo, not one (discovery)
- **Decision**: keep `ResolvedRepo.Root` as the directory holding `.spektacular/` and add a resolved `Source` that defaults to `Root`; only code consumers (repo list root, git -C, roster, templates) switch to `Source`.
- **Rationale**: every Spektacular-files consumer derives its path from `Root`, and `source` is read from `<Root>/.spektacular/repo.yaml`, so redefining `Root` is circular and breaks footprint validation.
- **Rejected**: redefining `Root` as the source; a separate "config root" field with `Root` becoming source (same churn, more call sites).

### Remove step 08 rather than redirect it (discovery)
- **Decision**: delete `update_repo_changelog` from the implement FSM and fold the release-note intent into the per-repo derived record written by `update_feature_changelog`.
- **Rationale**: the spec allows exactly two changelogs; step 10 already writes one per affected repo at the repo config's location; the root CHANGELOG.md was a plan-level addition never mandated by a spec.
- **Rejected**: keeping the step but writing into the repo changelog directory (a third file); enforcing "no CHANGELOG.md" in Go (needs repo plumbing the workflow config lacks).

### Sources reach the agent through the existing roster data (discovery)
- **Decision**: extend the roster entries with each repo's resolved `source` and inject the roster into the implement workflow the same way plan and spec do, rendering it in the steps that touch code.
- **Rationale**: the `repos` workflow-data path and `RepoRosterExtra` already exist; plan 000039 forbade registry data in `workflow.Config`; library skills cannot be templated.
- **Rejected**: `workflow.Config` fields; templating library skills; leaving it to `repo list` only (spec requires provision at run time).

### No validation of `source` at registration (discovery)
- **Decision**: `repo add` writes `source` like the other descriptive fields without checking a file source's existence or git status.
- **Rationale**: the spec explicitly dropped missing-path/validation requirements; non-git file sources are a Non-Goal, not a validation target.
- **Rejected**: stat/`git rev-parse` checks in `repo add`.

### `${VAR}` is the only env-var form (discovery)
- **Decision**: rely on the existing whole-file `${VAR}` expansion; do not add `$VAR` or `~`.
- **Rationale**: expansion is file-wide, so widening it changes semantics for every existing value; `${VAR}` satisfies the spec.
- **Rejected**: `os.ExpandEnv`; field-specific expansion.

### Chosen direction: source on the resolved repo (architecture)
- **Decision**: Option A. `ResolvedRepo` carries `Root` (directory holding `.spektacular/`) and a resolved `Source` (defaults to `Root`); resolution reads `source` once; code consumers use `Source`, Spektacular-files consumers keep `Root`; roster and listing share one git-free reader.
- **Rationale**: single resolver, no divergence between `repo list` and the roster, solves the read-source-from-the-file-found-via-root ordering in the one place that already parses repo.yaml.
- **Rejected**: Option B (edge-resolved, duplicated reads, git -C left wrong); Option C (Root becomes source, touches every footprint consumer and flips the colocated tolerance).

### Path-prefix convention redefined without a location stand-in (architecture)
- **Decision**: in the changelog steps, a `<repo-name>: ` prefix is required whenever the project registers more than one repo; an unprefixed path belongs to the only registered repo.
- **Rationale**: the old rule defined unprefixed paths by "the one that shares this project's working tree", which the spec forbids as a stand-in; step 10 still derives per-repo records from prefixes.
- **Rejected**: always requiring a prefix (breaks the single-repo project's habit); keeping the working-tree definition.

### Release note folded into the per-repo derived record (architecture)
- **Decision**: remove `update_repo_changelog` and have `update_feature_changelog`'s per-repo record include the short user-facing summary; `update_changelog` advances directly to `test_plan`.
- **Rationale**: spec allows exactly two changelogs; the per-repo record at the repo config's location already exists.
- **Rejected**: renaming the step to write into the repo changelog dir (a third file).

### `source` counts for equality but not for the metadata note (architecture)
- **Decision**: `repo add` rewrites repo.yaml when `source` changes, but a repo with only `source` set still gets the "no descriptive metadata" note.
- **Rationale**: source is a location, not a description; the note exists to improve planning quality.
- **Rejected**: treating source as descriptive metadata for the note.

### Harbor implement suite gains a separate-source repo (architecture)
- **Decision**: move the seeded `docs` footprint under the project (`/app/repos/docs/.spektacular/`) with `source: file:///opt/docs-repo`, switch the registry to `location`, and update the path oracles and step order.
- **Rationale**: the spec's success metric (clean `git status` in the code repo) is only provable end to end; the testing-architecture knowledge entry requires oracle updates in the same change.
- **Rejected**: leaving harbor untouched and relying on unit tests alone; a git-source fixture (needs a reachable remote inside the container).

### Conventions selected (architecture)
- **Decision**: apply error-remediation, tests-pass-before-done, no-em-dashes, MDX authoring, file-scoped headings, alternate section background, and plan-content-pages. No convention was dropped as irrelevant.
- **Rationale**: the work spans Go error paths, tests, docs-site MDX, README prose, and step-template prose; every listed convention bears on at least one of those surfaces.
- **Rejected**: none.

### Source classification helper lives on RepoConfig (data_structures)
- **Decision**: `RepoConfig.ParseSource(configDir)` returns the kind and the resolved value (absolute path for files, URL for git, "" when unset); the repo package applies the `Root` default and the clone.
- **Rationale**: the loader only knows the file path, and every consumer already anchors its own relative paths; a method keeps the scheme and anchoring rules beside the field they govern.
- **Rejected**: changing `RepoConfigFromYAMLFile`'s signature to take a base directory (eight call sites, and the other relative fields anchor at Root, not the config dir); classifying in `internal/repo` (splits the field's meaning across packages).

### `repo list` keeps its schema; no separate `source` output field (data_structures)
- **Decision**: only `root` changes meaning; `address` is dropped and `location` added; no new `source` field.
- **Rationale**: the spec says root is replaced by the source; adding a second field invites agents to pick the wrong one.
- **Rejected**: emitting both `root` and `source`.

### Harbor plan suite left unchanged (testing_approach)
- **Decision**: only the implement harbor suite is extended; the plan and spec suites are untouched.
- **Rationale**: their fixtures register no repos, and their oracles do not include the reworded roster fallbacks; the plan-side metric is covered by unit-level roster and listing tests.
- **Rejected**: adding a multi-repo plan-suite variant (25-minute run for coverage the unit layer already provides).

### Step 08 deleted without renumbering later templates (phases)
- **Decision**: delete `08-update_repo_changelog.md` and leave `09-12` filenames as they are.
- **Rationale**: templates are referenced by explicit path, and renumbering touches every reference and test for no behavioural gain; the implementer may renumber if they prefer.
- **Rejected**: mandatory renumbering.

### Harbor separate repo lives at `/app/repos/docs` (phases)
- **Decision**: the seeded separate-source footprint is placed under the project at `repos/docs/` (a plain tracked folder, not the gitignored `.spektacular/repos/`), with `source: file:///opt/docs-repo`.
- **Rationale**: it is the layout the spec's overview describes (a project folder with one folder per repo), and it keeps the materialization directory for clones only.
- **Rejected**: placing it under `.spektacular/repos/docs/` (gitignored, conflates with clones).

### Roster fallback wording (phases)
- **Decision**: when no repos are registered, templates say "the project's single repo; run `repo list` for its source" rather than "the colocated repo".
- **Rationale**: avoids naming a location while still telling the agent where to look.
- **Rejected**: keeping "colocated repo" (implies the working tree).

### Em dashes kept where the plan scaffold prescribes them (verification)
- **Decision**: the plan documents keep em dashes in the formats the workflow itself mandates (the `- **name** — why` conventions bullet, the `path:line — learned` research line, the "Chosen approach — evidence" heading).
- **Rationale**: the no-em-dashes convention governs the project's authored docs, README, skill, and changelog prose, which the phases enforce; the plan store documents are workflow artifacts whose scaffolds dictate these forms.
- **Rejected**: rewriting every scaffold-prescribed line (would break the required heading and the verification step's format expectations).

### Git origin moves from the registry into the repo's `source` (walkthrough, user decision)
- **Decision**: the registry entry carries `name` and `location` (folder holding `.spektacular/`), with `local` accepted as an alias; `address` is removed with no migration; a repo's single `source` is a file location or a git location distinguished by scheme; git sources clone into `.spektacular/repos/<name>/`.
- **Rationale**: the user's call at the walkthrough: with the origin in the repo folder, a remote code repo never needs Spektacular files checked in, and one key answers "where is the code" for both layouts. The spec was updated to match.
- **Rejected**: keeping `address` in the registry (footprint would have to live inside the clone); separate `address` and `source` fields in repo.yaml (two keys for one question); migrating `address` (nothing shipped with it).

### Scheme-less `source` is a file path (walkthrough)
- **Decision**: a `source` without a scheme is treated as a file path; `file://` is the explicit form; `git://`, `ssh://`, `http(s)://`, and scp-style are git; other schemes are a load error with remediation.
- **Rationale**: keeps `source: ..` and `source: ../../code/api` ergonomic, matches how git itself reads scheme-less values as local paths, and makes the git case unambiguous.
- **Rejected**: requiring an explicit scheme always (ceremony for the common relative-path case); treating scheme-less values as git remotes (surprising for a path).

### `location` is required and created on registration (walkthrough)
- **Decision**: `repo add` requires `location` and creates the folder when missing; `repo list`/resolution of an entry whose location is missing errors with remediation.
- **Rationale**: the location is the project's own footprint folder for the repo, so creating it is registration's job; a missing folder for an already-registered repo signals a broken checkout of the project, which should be surfaced, not papered over.
- **Rejected**: defaulting `location` to a conventional folder (the user asked for no prescribed layout); leaving creation to the user (fails the "registers correctly on the first attempt" metric).

### `repo list` never clones an uncloned git source (walkthrough)
- **Decision**: listing reports an empty `root` and `materialized: false` for a git source that has not been cloned yet; the clone happens on first use (`repo add`, or a workflow resolving the repo).
- **Rationale**: preserves the side-effect-free listing contract from plan 000042 and the existing `repo add` clone-on-add behaviour.
- **Rejected**: cloning during listing.

## Rehydration cues

- `go run . spec file read 000046_relocatable-repo-footprint.md` (the updated spec: registry `location`, single `source` with file and git kinds, `address` removed)
- `go run . knowledge always-applied --repo spektacular --repo docs`; `go run . knowledge read --data '{"scope":"project","path":"gotchas/repoconfig-must-start-from-default.md"}'`; `go run . knowledge read --data '{"scope":"project","path":"architecture/testing-architecture.md"}'`
- `go run . repo list` (docs repo root is `../spektacular-website`)
- Re-read: `internal/config/config.go:120-136,286-315`, `internal/config/repo.go`, `internal/repo/set.go`, `cmd/repo.go:118-341`, `internal/repo/roster.go`, `internal/stepkit/stepkit.go:129-140`, `internal/steps/implement/steps.go:19-33`, `templates/steps/implement/08-update_repo_changelog.md`, `templates/steps/implement/10-update_feature_changelog.md:20-70`
- `grep -rn "running in\|colocated repo\|CHANGELOG.md" templates/` for the stand-in phrases; `grep -rn "address" cmd internal templates README.md tests/harbor` for every registry-address touchpoint
- `tests/harbor/implement-workflow/tests/test_implement_workflow.py:17-103` for the hand-maintained oracles
