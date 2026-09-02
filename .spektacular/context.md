# Working context: relocatable-repo-footprint

## Problem

The user has a situation where they do not want any Spektacular files in the
repo containing the code — "no repo.yaml, no knowledge". Their idea: run
Spektacular from a separate project repo. Today the project-as-separate-repo
shape already exists (registry of member repos via `local`/`address`,
clones materialize into the gitignored `.spektacular/repos/<name>/`), but
every registered member repo is forced to carry a footprint inside the code:
`.spektacular/repo.yaml`, the knowledge tree with per-category READMEs, and
per-repo derived changelog records. Resolution fails with a repair offer if
repo.yaml is missing (`internal/repo/set.go` checkFootprint).

## Direction chosen (user's proposal)

User's exact phrasing: "add a source directory to the repo.yaml, this would
allow the root of the code to be located away from the repo.yaml. This way I
could have a project folder, which contains repo folders each containing
knowledge, changelog, etc."

So a repo has two locations: the **footprint directory** (holds repo.yaml,
knowledge, changelog) and the **source root** (the code), pointed at by a new
`source` field in repo.yaml. Default `source: ..` preserves every existing
colocated layout unchanged (repo.yaml already sits at `<code>/.spektacular/`).

## Alternative rejected

A per-repo `footprint: none` opt-out that moves repo metadata/knowledge/
changelog into the project config. Rejected because it inverts the
"repo owns its own metadata" model and re-adds descriptive fields to the
registry entry; the `source` approach keeps the model and everything already
built keeps working.

Stopgap noted but not the answer: `.git/info/exclude` in the code repo.

## Design decisions discussed (assistant recommendations, user did not object)

- Project-hosted footprints live at `.spektacular/repos/<name>/` (tracked);
  a materialized clone lands at `.spektacular/repos/<name>/source/`;
  gitignore narrows from `repos/` to `repos/*/source/`. `source: ./source`
  natural for address-only repos; local checkouts use absolute or
  `${VAR}`-expanded paths (repo config already expands env vars).
- Relative paths in repo.yaml resolve against the directory holding
  repo.yaml (single anchor). Defaults change from `.spektacular/knowledge`
  and `.spektacular/changelog` to `knowledge` and `changelog`. Existing
  colocated configs are rewritten by the config migration mechanism.
- Project finds the footprint by convention: project-hosted folder first,
  fall back to `<local>/.spektacular/`. `repo add` gets a flag choosing
  project-hosted at registration. `local`/`address` keep meaning "where the
  code is".
- Project's own colocated footprint (init writes repo.yaml at project root,
  project treated as repo of one) becomes optional for a code-free project
  repo; the implement templates' "paths with no prefix belong to the
  colocated repo" rule applies only when a colocated repo exists.
- Root `CHANGELOG.md` release note written at the source root by the
  implement workflow should be routable to the footprint for zero-touch.

## Additional requirements from the user

- "we also need to update the docs"
- "cater for breaking changes in existing config"

## Code touchpoints identified

internal/repo/set.go, internal/repo/footprint.go, cmd/repo.go,
cmd/knowledge.go (aggregateKnowledgeSources), cmd/storefile.go
(repoRoutedStore), internal/project/init.go, cmd/version.go (migration),
templates/skills/skill_manage-repos.md, templates/steps/implement/*.md,
templates/.spektacular/.gitignore, docs/.

## Clarification during interview (supersedes the "Design decisions discussed" above)

User: "The structure can be flexible, currently the projects already
contains a location to repo, all repo needs is location to source.
Sometimes you would want the project and repos to be checked in so this
change needs to add optional features."

Resulting decisions:
- The repo root stays the directory holding `.spektacular/`. `source`
  defaults to the repo root (colocated). Relative-path anchoring and
  default knowledge/changelog locations are UNCHANGED — no path migration.
- No prescribed layout, no gitignore changes; `.spektacular/repos/`
  materialization untouched. Footprints placed elsewhere are ordinary
  tracked files.
- Registry `local`/`address` keep locating the repo root; only repo.yaml
  locates the source. Registration accepts `source` and writes it.
- The project's own colocated repo.yaml may declare `source`, enabling a
  registry-free single-external-repo project.
- Release-note CHANGELOG.md goes to the repo root (so it stays out of a
  separated source with no extra setting).
- Clone-into-source on first use left to the plan to weigh.

Spec sections drafted and staged at .spektacular/tmp/spec_template.md;
fresh-eyes review pending, then user review, then commit.

## Review pass with the user (spec trimmed heavily)

The user found the first draft over-specified and walked the requirements
one by one. Rules that emerged, worth carrying into the plan:

- Do not state requirements for behaviour that already exists or that is
  implied by another requirement ("nothing requirements"). Registration,
  re-registration, missing-path errors, attribution, backward compat, and
  migration were all dropped on that basis: source is one optional field
  in repo.yaml and everything else follows.
- Vocabulary: "source" and "colocated". Avoid coined terms (footprint,
  repo root, source root).
- `repo list` already reports `root`; the requirement is that root is
  replaced by source when set.
- Exactly two changelogs: project (at project config's location) and one
  per repo (at repo config's location). The implement step that writes a
  release note to a root CHANGELOG.md in the code is therefore removed or
  folded into the repo changelog.
- Templates: user is concerned about multiple sources; the spec says only
  that templates tell the agent to use the repo source location and that
  source locations are provided to the agent at run time — mechanism left
  to the plan. Member-repo templates already use `repo list`; the gap is
  templates that treat the project's own repo as "the directory you are
  running in".
- Constraints section deliberately empty beyond the requirements.
- Non-goals confirmed by the user's "ok, let's go".

## Plan workflow (started 2026-09-02)

Planning against spec `000046_relocatable-repo-footprint` (status: completed).
Working files go under `.spektacular/work/000046_relocatable-repo-footprint/`.
Everything above this heading is the spec interview's record; the "Review
pass with the user" section is the most current statement of intent and
supersedes the earlier "Design decisions discussed" list where they conflict.

Carry into Testing Approach: the two success metrics (clean `git status` in
an external source after plan+implement; agent registers a separate-source
repo first try from the repo-management skill).

### Discovery learnings (plan workflow)

- Registry `local`/`address` locate the directory holding `.spektacular/` (the
  "root"); `source` in repo.yaml is a second path. Keep `ResolvedRepo.Root`
  for Spektacular-files consumers, add `Source` for code consumers.
- `${VAR}` expansion is already file-wide in `RepoConfigFromYAMLFile`; no
  code needed for the env-var form. `omitempty` on the new field is required
  by an existing two-keys test.
- No migration needed: an absent key loads as zero value; the migration
  mechanism is a one-shot config.yaml/repo.yaml split detector.
- The implement workflow gets no roster today (only plan and spec do);
  providing sources at run time means adding `wf.SetData("repos", …)` in
  cmd/implement.go and `RepoRosterExtra` on the code-touching steps.
- Step 08 `update_repo_changelog` (root CHANGELOG.md) is removed; step 10
  already writes the per-repo derived record. Harbor implement suite's
  EXPECTED_STEP_ORDER and fixtures must move in the same change.
- Project config.yaml already has `source:` meaning a git URL for
  provenance; docs must disambiguate from the repo-level directory `source`.
- Docs: repo.yaml reference is in docs:src/pages/configuration.mdx (the
  separate repo-configuration page was merged); layout narrative belongs in
  projects.mdx "How configuration is split".
- Library skills (`skill_*.md`) are served raw, never templated.

### Architecture decisions (plan workflow)

- Option A chosen: `ResolvedRepo{Root, Source}`; `repo list` root = Source;
  roster gains `source`; implement workflow injects the roster into the
  code-touching steps; step 08 removed; per-repo record carries the release
  note; new template-contract test bans stand-in phrases.
- Path-prefix rule in changelog steps: prefix required when >1 repo is
  registered; unprefixed = the only registered repo.
- Harbor implement suite gets a separate-source `docs` repo fixture.
- Components: 11 changed/new in spektacular (config model, resolution, repo
  cmds, roster, implement cmd, implement/plan/spec steps, skills, template
  tests, harbor, README) + 2 pages in docs (configuration.mdx, projects.mdx).
- Data contracts: `RepoConfig.Source` + `ResolveSource(configDir)`;
  `ResolvedRepo.Source`; `Set.LocalSource(name)`; roster key `source`;
  `repo list` schema unchanged (root = source).
- Implementation detail drafted: no new patterns beyond the two-location
  resolved repo; registration, roster, FSM edit, and banned-phrase tests all
  follow existing shapes.
- Dependencies: all internal; nothing must land first; harbor run is a manual
  verification dependency; docs repo depends only on the fixed key semantics.
- Testing: metric 1 = behavioural (harbor implement suite with a
  separate-source repo); metric 2 = manual, captured in the implementation
  test plan. Only the implement harbor suite changes.
- Milestones: (1) source in config/resolution/commands, (2) remove root
  changelog step, (3) sources at run time + template test + harbor, (4) docs.
- Phases: 1.1 config field, 1.2 resolution, 1.3 repo cmds + root-bound
  regression tests, 2.1 remove step 08, 2.2 fold summary into per-repo
  record, 3.1 roster source + implement injection, 3.2 template rewording +
  contract test, 3.3 harbor separate-source fixture, 4.1 docs site, 4.2
  README + skill.
- Open questions: one, impl-time only (agent attribution under the restated
  prefix rule, settled by the harbor run).
- Out of scope recorded (spec non-goals + validation, Go guard, skill
  templating, other harbor suites, renumbering, wider env expansion).
- Assembled: plan/context/research staged under .spektacular/tmp/
  (*_template.md); 10 phases, 4 milestones, headings verified.
- Verification passed: sections, repo lines, anchors, criteria all present;
  Project References added to context; em-dash call recorded.
- plan.md committed to the store.
- context.md committed to the store.
- research.md committed; working dir removed. Next: walkthrough with the user.

### Walkthrough change (user decision, 2026-09-02)

- Git origin moves from the registry into the repo's own `source`. Registry
  entry = `name` + `location` (folder holding `.spektacular/`); `local`
  accepted as an alias; `address` removed, hard break, no migration.
- Single `source` in repo.yaml: file location (plain or `file://`, abs /
  relative to the repo.yaml dir / `${VAR}`) or git location (`git://`,
  `ssh://`, `http(s)://`, scp-style), cloned into `.spektacular/repos/<name>/`
  on first use. Scheme-less = file path.
- Spec updated via `spec file write` to match; plan/context/research
  rewritten and recommitted (11 phases: 1.1-1.4, 2.1-2.2, 3.1-3.3, 4.1-4.2).
- This repo's own .spektacular/config.yaml must be updated in Phase 1.2.
- Walkthrough: user signed off on the revised plan ("I think this is good").

## Implement workflow (started 2026-09-02)

Implementing plan `000046_relocatable-repo-footprint`. The sections above are
the spec and plan records; the "Walkthrough change" section is the final
statement of the design (registry `name`+`location`, single repo-level
`source` with file and git kinds, `address` hard break, step 08 removed).

### read_plan learnings

- Validation passed: 10 sections, 11 phases (1.1-1.4, 2.1-2.2, 3.1-3.3,
  4.1-4.2) all linked to context.md headings; no file or symbol drift found;
  all spec requirements/criteria covered; no `## Changelog` section yet, so
  this is the first-phase invocation (start at Phase 1.1).
- Docs repo is colocated at `../spektacular-website` (repo list root); it is
  the `docs` member repo for Phase 4.1.
- Current `.spektacular/config.yaml` still uses `local` + `address` for
  `docs`; Phase 1.2 must hand-edit it or the CLI stops loading.
- Line numbers in the plan are approximately current (verified symbols at
  the named locations).

### Phase 1.1 analysis

- `internal/config` does not import `internal/output` and `output` does not
  import `config`, so config validation can build `output.NewError(...)`.
- `cmd/root.go` `toErrorResponse` uses a direct type assertion on
  `*output.ErrorResponse`, not `errors.As`: a remediation error built in
  `internal/config` must be returned unwrapped (no `fmt.Errorf("%w")`)
  or the next_action is lost. Applies to Phase 1.2's `address` rejection.
- `RepoConfigFromYAMLFile` expands `${VAR}` file-wide before unmarshal, so
  `ParseSource` does no expansion. Tests use `t.Setenv` + `t.TempDir`.
- Existing two-top-level-keys test (`repo_test.go:52`) guards `omitempty`.

### Phase 1.1 implemented

- `RepoConfig.Source` (`yaml:"source,omitempty"`), `SourceKind`
  (`SourceNone/SourceFile/SourceGit`), and `RepoConfig.ParseSource(configDir)`
  added in `internal/config/repo.go`. Git schemes: `git://`, `ssh://`,
  `http://`, `https://`, `git+*`, scp-style `user@host:path`. Any other
  explicit `scheme://` is an error; scheme-less is a file path anchored at
  configDir. `file://` prefix is stripped then treated as a path.
- Phase 1.1 tests appended to `internal/config/repo_test.go` (11 tests,
  criteria 1-4 plus unset and unsupported-scheme cases).
- For Phase 1.2: `RepoEntry.WithDefaults()` is only called from
  `repo.New` (set construction), not at config load, so the `local` alias
  must be folded in `ParseYAMLFile` (after unmarshal, before validation) as
  well as in `WithDefaults`, or `validateRepos` never sees `Location`.
- Phase 1.1 verified green (build, vet, `go test ./...`, lint). `gofmt -l .`
  reports 6 pre-existing unformatted files on main (cmd/repo.go,
  cmd/version_test.go, internal/runner/runner.go,
  internal/stepkit/stepkit_test.go, internal/steps/spec/steps.go,
  internal/steps/spec/steps_test.go); not caused by this work, left alone
  unless a later phase edits them.
- Phase 1.1 changelog entry written; 10 phases remain. Looping without
  pausing per the user's standing preference to drive workflows straight
  through and stop only for real design decisions.

### Phase 1.2 analysis (and sequencing decision)

- Address-driven clone tests (7 in `internal/repo/set_test.go`, 4 in
  `git_integration_test.go`, 3 in `cmd/repo_test.go`, 1 each in
  `roster_test.go`, `cmd/knowledge_test.go`, `cmd/init_test.go` helper
  `initProjectWithAddressOnlyRepo`) only make sense once a git `source`
  can clone. Decision: implement the 1.2 registry change and the 1.3
  source resolution in one code pass so those tests are rewritten once as
  git-source tests; 1.2 and 1.3 are still ticked in separate workflow loops
  with their own criterion tests. Deviation to record in both changelogs.
- `address` rejection lives in `ParseYAMLFile` (raw YAML key scan) and is
  returned unwrapped as `output.NewError("config_invalid", ...)`; FromYAMLFile
  passes ParseYAMLFile errors through without wrapping. Missing location is
  rejected in `validateRepos` (wrapped by FromYAMLFile; switch
  `cmd/root.go toErrorResponse` to `errors.As` so next_action survives).
- `RepoEntry.WithDefaults` must not be applied at parse time (it sets
  `provider: git`, which would change every written config); fold only the
  `local` alias in the parser.
- `cmd/repo.go` in 1.2: input/list/schema/equality move to `location`
  (compile-level); `source` payload, required-location, MkdirAll and
  root=source stay in 1.4.

### Phase 1.2 (+1.3 resolution) implemented

- `RepoEntry{Name, Location, Local(alias), Dependencies, Provider, Config}`;
  alias folded in `ParseYAMLFile` and `WithDefaults`; `address` rejected by
  a raw-YAML scan in `ParseYAMLFile` (`config_invalid`, next_action names
  the `source:` line); missing location rejected in `validateRepos`.
- `cmd/root.go toErrorResponse` now uses `errors.As`.
- `ResolvedRepo` gained `Source`; `Set.LocalSource`, `locationRoot`,
  `cloneDir`, `loadFootprint` (replaces `checkFootprint`); `resolve` errors
  `repo_location_missing` when the location dir is absent; git sources
  clone into `.spektacular/repos/<name>/`, `Materialized=true`, stale note
  against the source URL.
- `cmd/repo.go`: payload key `location`, list field `location` (no `local`
  echo: the alias is folded at parse so the raw value is gone; minor
  deviation from the 1.4 text), `reposEqual` on Location. `root` still
  reports Root until 1.4.
- `.spektacular/config.yaml` hand-updated; `repo list` loads.
- Test files referencing `Address` no longer compile; the test step must
  rewrite them (git-source equivalents).
- Phase 1.2 tests: config alias/address/missing-location tests, init
  `location: .` tests, cmd fixtures moved to `location`, absent-location
  entries replace address-only ones. Phase 1.3 criteria 1-4 already covered
  in `internal/repo/set_test.go` + `footprint_test.go` (file/git source
  resolution, footprint at root, `LocalSource`). Integration tests serve the
  source repo over git's dumb HTTP protocol via `httptest` (plain paths are
  file sources, so `http://` is the only local-friendly git form).
- `internal/repo/git.go` package comment and git-missing error reworded.
- Phase 1.2 verified green (full suite incl. git integration).
- Phase 1.2 ticked and changelog written; 9 phases remain. Next loop: Phase 1.3 (code already in place; add any missing criterion tests, verify, tick).

### Phase 1.3 loop

- Code landed in the 1.2 pass; criteria 1-4 covered by
  `TestResolve_FileSourceResolvesSourceDirWithoutGit`,
  `TestResolve_GitSourceClonedOnceAndReused`,
  `TestResolve_FootprintValidatedAtRootNotSource` +
  `TestEnsureFootprint_SourceDeclaredScaffoldsUnderRoot`,
  `TestLocalSource_ReportsCodeDirWithoutGit`. No new code this loop.
- Phase 1.3 verified green (repo/config/cmd packages).
- Phase 1.3 ticked and logged; 8 remain. Next: Phase 1.4 (repo add source/location/MkdirAll, list root=source, root-bound regression tests).

### Phase 1.4 analysis

- `cmd/knowledge.go` (LocalRoot), `cmd/storefile.go` (resolved.Root), and
  `internal/project/init.go` cascade (LocalRoot) are already root-bound; no
  code change, regression tests only.
- `runRepoAdd` order must become: upsert registry → write config → MkdirAll
  location → EnsureFootprint(location) → merge descriptive fields + source
  into repo.yaml → `set.Resolve` to materialize a git source. Resolving
  before writing `source` would miss a git source given in the same add.
- `runRepoList` must use `LocalSource` to decide whether to resolve: an
  uncloned git source reports `root: ""`, `materialized: false`, no git call.

### Phase 1.4 implemented

- `repo add`: payload `location` required (`invalid_data` with example
  payload), `source` optional and written into repo.yaml like the
  descriptive fields, raw `address` key rejected (`invalid_data`, next_action
  gives the corrected payload); location dir MkdirAll'd; footprint ensured at
  the location; repo.yaml written before `set.Resolve` materializes a git
  source. `repo list`: `root` = resolved `Source`; uses `LocalSource` so an
  uncloned git source reports `root: ""`, `materialized: false`, no clone.
- Expected test fallout (to fix in test step): schema test (location
  required, source present), `TestRepoAdd_MissingLocationErrors` (now
  `invalid_data` from repoAddData), `TestRepoList_StaleNoteFromDifferingHeads`
  (root is now the clone dir, not the location).
- Phase 1.4 tests added in cmd (add/list/knowledge/changelog/init). Note:
  relative `source` anchors at `<location>/.spektacular`, so from
  `repos/api/.spektacular/repo.yaml` the project-level `code/` dir is
  `../../../code`. `stubGit` in cmd now records `clones` (url, dir).
- Phase 1.4 verified green (full suite).
- Phase 1.4 ticked and logged; Milestone 1 complete, 7 phases remain. Next: Phase 2.1 remove update_repo_changelog.

### Phase 2.1 analysis

- Touchpoints confirmed: FSM rows + `updateRepoChangelog` in
  `internal/steps/implement/steps.go`; template `08-update_repo_changelog.md`;
  goto in `07-update_changelog.md:71-75`; two bullets in `12-finished.md`;
  `skill_update-changelog.md:7,24`; tests in `steps_test.go` (order, walk,
  loop, branch assertion, STOP map, two step-08 tests); harbor
  `EXPECTED_STEP_ORDER`; comments in `cmd/docs_test.go:47` and
  `templates/context_directive_test.go` (floor 30 stays).
- Phase 2.1 implemented: FSM row + callback + template removed,
  `update_changelog` → `test_plan`, finished/skill/07 wording updated, harbor
  `EXPECTED_STEP_ORDER` and `environment/state.json` fixture updated.
- Phase 2.1 tests updated (FSM order/walk/loop, new TestFSMHasNoUpdateRepoChangelogStep, TestNoImplementTemplateMentionsRootChangelogFile, cmd implement tests step counts 13→12).
- Phase 2.1 verified green (full suite).
- Phase 2.1 ticked and logged; 6 remain. Next: Phase 2.2 (fold release note into per-repo record; finished step reports both locations).

### Phase 2.2 analysis

- `10-update_feature_changelog.md` Step 4 gets a leading 2-4 sentence
  user-facing summary bullet; `12-finished.md` summary and report bullets
  name the project record path and the per-repo `changelog file read --repo`
  path; tests at `steps_test.go:425-490` cover finished/feature-changelog.
  Harbor `instruction.md` already names only the three records.
- Phase 2.2 implemented: per-repo summary bullet in step 10; finished step names project + per-repo record locations.
- Phase 2.2 tests added (finished locations; per-repo summary).
- Phase 2.2 verified green (full suite).
- Phase 2.2 ticked and logged; Milestone 2 done, 5 remain. Next: Phase 3.1 roster source + implement roster injection.

### Phase 3.1 analysis

- `Roster` gains `"source"` via `set.LocalSource` (git-free). Implement
  command injects `repos` on new/goto like plan/spec. Six implement
  callbacks pass `stepkit.RepoRosterExtra(data)`. Criterion 3 needs the six
  templates to render the roster block, so the shared block is added here;
  Phase 3.2 rewords stand-ins and adds the contract test.
- Phase 3.1 implemented: `Roster` entries carry `source`; `cmd/implement.go`
  sets `repos` on new/goto; six implement callbacks pass
  `RepoRosterExtra`; templates 01-05 gained a shared "Where the code
  lives" roster block, 10's Step 2 renders the roster with `repo list` as
  fallback.
- Phase 3.1 tests added (roster source table; six-step render + fallback; cmd persistence test). Note: cmd implement tests must call resetImplementCommandFlags(t) when running after the schema tests.
- Phase 3.1 verified green (full suite).
- Phase 3.1 ticked and logged; 4 remain. Next: Phase 3.2 rewording + templates/repo_source_test.go.

### Phase 3.2 analysis

- Stand-ins to remove: `02-discovery.md:13,28`, `03-architecture.md:11`,
  `00b-interview.md:19`, spek-plan `SKILL.md:51` ("directory you are
  running in"), `07:32`/`10:37` (working-tree prefix rule), spawn skills
  ("resolved root"/"resolved local path"), spek-implement `SKILL.md:41`.
- Tests to update: plan `steps_test.go` fallback strings (discovery,
  architecture), spec `steps_test.go` interview fallback const,
  `internal/agent/instruction_surface_test.go` ("attributed repo's resolved
  root"). New `templates/repo_source_test.go` (banned list + positive).
- `skill_gather-project-metadata.md` is 39 lines (plan's :114-125 is stale);
  add one line that commands run in the target repo's source.
- Phase 3.2 implemented: roster lines in discovery/architecture/interview show source; fallbacks reworded; implement drift/test/verify point at each repo's source; prefix rule restated; spawn skills + spek-plan/spek-implement SKILL reworded; gather-metadata note.
- Phase 3.2 tests: new templates/repo_source_test.go (banned list + roster/source + prefix-rule assertions); plan/spec fallback strings, agent instruction-surface, implement drift/prefix, cmd skill tests updated.
- Phase 3.2 verified green (full suite).
- Phase 3.2 ticked and logged; 3 remain (3.3 harbor, 4.1 docs, 4.2 README/skill).

### Phase 3.3 analysis

- Harbor implement suite: fixtures under `tests/harbor/implement-workflow/environment/`,
  oracles in `tests/test_implement_workflow.py` (verifier reads container
  paths `/app/.spektacular`, `/opt/docs-repo/.spektacular`), artifacts in
  `task.toml`, agent narrative in `instruction.md`. Run via
  `make harbor-test-implement` (harbor + docker present locally; ~25 min).
- Change: `docs` footprint moves to `/app/repos/docs/.spektacular/` with
  `source: file:///opt/docs-repo`; `/opt/docs-repo` becomes a bare code
  checkout with no `.spektacular/`.
- Phase 3.3 implemented: harbor fixtures (config.yaml `location`, docs
  footprint at /app/repos/docs/.spektacular with `source: file:///opt/docs-repo`,
  /opt/docs-repo a bare code checkout), oracles (`DOCS_REPO_SPEK_DIR`,
  two new source-cleanliness tests), task.toml artifacts, instruction.md.
  The suite's pytest file IS the phase's test; verification = one harbor run.
- While the harbor run proceeds (background, started during 3.3 verify),
  Phase 4.1 docs edits were prepared in `../spektacular-website`
  (configuration.mdx, projects.mdx; `npm run build` exit 0, `astro check`
  clean for pages, MDX guard clean, no em dashes) and Phase 4.2 README +
  `skill_manage-repos.md` + `cmd/skill_test.go` edits were prepared (cmd
  tests green). Both will be ticked in their own loops after 3.3.
- 4.2 deviation: the skill names `address` only to say it is no longer
  accepted; the test forbids `"address":` payloads and address-only
  wording rather than the bare word.
- Phase 3.3 verified: harbor run tests/harbor/jobs/2026-09-02__14-04-20 passed 14/14 (incl. separate-source cleanliness tests). Open question (prefix attribution) settled: docs record landed at /app/repos/docs/.spektacular/changelog/auth-service/.
- Phase 3.3 ticked and logged; Milestone 3 done. Next loops: 4.1 (docs, prepared), 4.2 (README/skill, prepared).
- Phase 4.1 loop: analysis/implement/test done earlier; verify = build exit 0, astro check no page diagnostics, MDX guard 0, em dashes 0.
- Phase 4.1 ticked and logged; 1 remains (4.2).
- Phase 4.2 verified green (full suite; README config section and skill em-dash free).
- Phase 4.2 ticked and logged; all 11 phases complete. Advancing to test_plan.
- test_plan written (manual metric: agent registers separate-source repo first try; harbor covers the clean-source metric).
- update_feature_changelog: project record + spektacular and docs per-repo records written via CLI.
- reconcile_spec: all 8 requirements and 8 acceptance criteria marked satisfied.
- Workflow finished (state.json current_step=finished). Spec and plan closed as completed.
