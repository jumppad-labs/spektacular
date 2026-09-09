---
created_date: "2026-09-07"
status: completed
closed_date: "2026-09-07"
---

# Research: 000049_guided-repo-add

## Alternatives considered and rejected

### A. Where the guided add's workflow state lives

- **A2 — extend the single `state.json` to hold one entry per concurrent workflow kind.** Rejected. `workflow.State`'s on-disk shape (`spektacular:internal/workflow/state.go:14-21`) is read by four independent consumers that would all break at once: the debug session probe (`spektacular:cmd/root.go:187-215`, its own `stateSnapshotFile` mirror), `internal/sessionlog`, the cross-kind test's `writeInProgressState` fixture helper (`spektacular:cmd/cross_kind_test.go`), and the harbor python oracles which read `state["data"]["name"]` directly (`spektacular:tests/harbor/spec-workflow/tests/test_spec_workflow.py:53-58`). The blast radius buys nothing over A1.
- **A3 — leave the add as a static skill playbook and only adopt the interview style.** Rejected by the spec's own mandated-mechanism constraint ("a workflow owned by the CLI, not a static playbook the agent improvises from"). It is also what exists today (`spektacular:templates/skills/workflows/spek-manage-repos/SKILL.md`, which states outright that it is "a static playbook, not an interactive CLI state machine, so there is no session to resume") and is exactly what fails the resume requirement.
- **A1 is chosen** — see evidence below.

### B. Who examines the repository to produce the proposals

- **B1 — the agent reads README/manifest/layout with its own file tools.** Rejected. This is today's behaviour (`spektacular:templates/skills/workflows/spek-manage-repos/SKILL.md` step 3, "Read the repo before asking anything else"), and it leaves the acceptance criterion "Proposals are drawn from the repository, not the folder name" with no oracle at all: nothing in Go can assert what the agent chose to open, nothing bounds the read against the "no deep code analysis" constraint, and nothing guarantees the examination happened before the second question.
- **B2 — the CLI produces the finished name/description/role/tags itself.** Rejected. There is no language model in the binary and no NLP or language-detection dependency in `spektacular:go.mod:5-12` (mustache, fsm, go-gitignore, cobra, testify, yaml only). A mechanically derived description degenerates into the restatement of the folder name that the acceptance criteria explicitly forbid.
- **B3 is chosen** — the CLI gathers and bounds the evidence, the agent synthesises the proposal from it. See evidence below.

### C. Who performs the write

- **C1 — the `register` step's instruction tells the agent to run `repo add --data '<payload>'`.** Rejected. It reintroduces the exact failure this spec exists to fix. The working context's diagnosis of the original vocabulary leak was that "the leak is not a set of bad sentences to delete, it is the skill teaching the agent a vocabulary and a shape that surface in user-facing speech" (`spektacular:.spektacular/context.md`); putting a rendered `--data` payload carrying `location`, `source`, `provider` into the agent's context at the final step is that same teaching, at the one moment the agent is also composing the user-facing confirmation. It also leaves "nothing is written before confirmation" resting on agent obedience rather than on the state machine.
- **C2 is chosen** — extract the registration core shared by both entry points. See evidence below.

### D. The command surface for the guided flow

- **D2 — `repo add --guided`.** Rejected. It modifies the command the spec requires to be behaviourally unchanged, and cobra flag state on `repoAddCmd` is package-level, reset between tests by `resetRepoFlags` (`spektacular:cmd/repo_test.go:41-49`); a mode flag on that command is the highest-risk possible place to put this.
- **D3 — bare `repo add` with no `--data` enters the guided flow.** Rejected. It deletes the missing-`--data` error path (`spektacular:cmd/repo.go:350-356`) that existing tests assert, and it makes a scripted caller's typo silently open an interactive session.
- **D1 is chosen** — `repo new` / `repo goto`, mirroring `spec new` / `spec goto` exactly.

### E. Where the gathered answers live between steps

- **E2 — per-section working files under `.spektacular/work/<name>/`, as spec and plan use.** Rejected. That convention exists because spec and plan sections are multi-paragraph documents assembled later by a dedicated assemble step (`spektacular:templates/work_files_test.go:23-48`). A guided add gathers four short scalar values; a working directory would need naming, creating, cleaning, and a resume-time read-back for no benefit.
- **E1 is chosen** — the workflow `Data` map, persisted in the repo workflow's own state file. Answers arrive as `--data` on each `goto` and are already persisted by the engine, so resume retains them with no new mechanism.

## Chosen approach — evidence

### A1: a second, independent workflow state file

- `spektacular:internal/workflow/workflow.go:73` — `func New(steps []StepConfig, statePath string, cfg Config, st store.Store, out ResultWriter) *Workflow`. The engine takes the state path as a parameter and has no constant or global tying it to `state.json`.
- `spektacular:cmd/spec.go:106-108` — `stateFilePath(dataDir)` returns `filepath.Join(dataDir, "state.json")`. This is the single place the shared filename is fixed; every caller (`cmd/spec.go:192,300,355`, `cmd/plan.go:103,194,246,253`, `cmd/implement.go:108,206,258,265`) recomputes it rather than reading a cached value.
- `spektacular:cmd/root.go:307-315` — `dataDir()` with the comment "Both spec and plan workflows share this directory (and a single state.json)". The sharing is a `cmd`-layer convention, not an engine constraint.
- `spektacular:cmd/resume.go:133-152` — `resumeOrClear(statePath, command, expectedKind, force)` already takes both the path and the expected kind as parameters, so it is reusable verbatim against a different file with kind `repo`. Its cross-kind branch simply never fires when only one kind ever writes that file.
- `spektacular:cmd/resume.go:169-181` — `guardKind(statePath, command, expectedKind)`, same shape, used by `goto`.
- `spektacular:internal/workflow/workflow.go:87` — `New` stamps `cfg.Kind` onto a freshly created `State`; `spektacular:internal/workflow/workflow.go:17-29` — `Config.Kind` is the field carrying it. A new kind string needs no type change.
- `spektacular:internal/workflow/state.go:25-27` — `State.InProgress()` hardcodes the literal terminal step name `"finished"`. The repo workflow's last `StepConfig.Dst` must be named `finished` or resume detection misreports.
- `spektacular:internal/workflow/workflow.go:113-133` — callbacks register as `before_<event>`, so a callback error vetoes the transition before `enter_state` persists. This is what makes an FSM-enforced "nothing written before confirmation" real: a failing `register` callback leaves the state on `confirm`.
- `spektacular:internal/workflow/workflow.go:147-156` — `enter_state` is the single mutation-and-persist point; `spektacular:internal/workflow/workflow.go:287-303` — `commitTerminal()` marks the final step complete.
- `spektacular:internal/workflow/workflow_test.go` — `TestMultiSourceTransition` confirms a step may declare several `Src` states, which is what lets the delegation shortcut skip the description/role/tags steps.

### B3: CLI gathers bounded evidence, agent synthesises

- `spektacular:internal/stepkit/stepkit.go:55-94` — `WriteStepResult` merges standard vars, then `PathStrategy.PathVars`, then per-call `Extra` (later wins). `Extra` is the existing, precedented channel for a step callback to push computed facts into its own instruction; `spektacular:internal/steps/plan/steps.go:88-97` shows the roster doing exactly this today.
- `spektacular:internal/stepkit/stepkit.go:121-127` — `RenderTemplate` renders through `cbroglie/mustache` against `templates.FS`, so `{{#…}}`/`{{^…}}` sections give the "evidence present" vs "nothing readable, ask without proposals" branch the unreadable-repo constraint requires, with no Go branching in the callback.
- `spektacular:internal/repo/footprint.go:28-85` and `spektacular:internal/repo/set.go:57-313` — the existing repo domain package already owns filesystem reads against a repo root, so the examiner has an obvious home and does not need a new package.
- `spektacular:go.mod:5-12` — no language-detection, README-parsing, or HTTP dependency exists; the examiner must be hand-rolled from `os`/`filepath`, which naturally bounds it to the README, manifest filenames, top-level entries, and an extension histogram, matching the "no deep code analysis" constraint.
- `spektacular:cmd/repo_test.go:70-94` — the testing idiom for this area (`t.TempDir()` + `t.Chdir`, package-level `repoGit` swapped for a stub via `swapRepoGit`) makes a pure examiner function trivially table-testable against synthetic repo trees.

### C2: one shared registration core

- `spektacular:cmd/repo.go:113-252` — `runRepoAdd` is a single linear handler: validate payload, load config, compute `entry.Location = filepath.Join(input.Location, ".spektacular")` (line 130-138), idempotent upsert into the registry by name, `EnsureFootprint`, write the repo's own `repo.yaml` fields, then build the result including `metadata_note`. Everything from the config load onward is entry-point-agnostic and extracts cleanly.
- `spektacular:cmd/repo.go:246-249` and `spektacular:cmd/repo.go:384-386` — `repoConfigDescriptiveFieldsEmpty` fires the missing-metadata note when description, role and tags are all empty. A guided add that always supplies all three provably never trips it, which is the "guided add produces complete metadata" criterion, checkable in Go.
- `spektacular:internal/repo/footprint.go:13-17,28-85` — `EnsureFootprint` is idempotent and additive (`FootprintCreated`/`FootprintRepaired`/`FootprintUnchanged`), so the shared core is safe to call from either entry point.
- Extracting rather than duplicating is what the standing DRY preference requires, and it is the only way "the direct add is unchanged" is guaranteed by construction rather than by parallel maintenance.

### D1 / step-template authoring

- `spektacular:internal/steps/spec/steps.go:25-39` — `Steps()` is a linear slice of `workflow.StepConfig{Name, Src, Dst, Callback}` with the terminal step named `finished`; `spektacular:internal/steps/spec/steps.go:108-112` — a callback is a one-line `writeStep(name, next, "steps/spec/NN-x.md", …)`. A `internal/steps/repo` package mirrors this shape exactly.
- `spektacular:internal/stepkit/stepkit.go:20-31` — `PathStrategy` requires `PathVars` and `PrimaryPathField`; the repo workflow needs a trivial implementation whose primary path is the target repo folder.
- `spektacular:templates/templates.go:6` — `//go:embed all:*` picks up a new `templates/steps/repo/` directory with no build change.
- `spektacular:templates/steps/spec/00b-interview.md` — the canonical house style for a conversational step: `## Step {{step}}: {{title}}` heading, bold lead-in sub-rules, an explicit stopping condition, the trailing `{{config.command}} <workflow> goto --data '{"step":"{{next_step}}"}'`, then `---` and the standing refresh-context footer.
- `spektacular:templates/steps/plan/02-discovery.md:62-71` — the "Record your judgement calls" / "Proceed unless genuinely blocked" pattern.
- `spektacular:templates/steps/resume.md` and `spektacular:templates/steps/resume_mismatch.md` — rendered by `resumeInstruction`/`mismatchInstruction` (`spektacular:cmd/resume.go:22-46`) with vars `config.command`, `kind`, `name`, `current_step`. `resume.md` currently branches on spec/plan vs implement for working files and will need a repo branch (or the repo workflow needs its own resume text), since a repo add has no `.spektacular/work/<name>/`.

### Skill and install wiring

- `spektacular:internal/agent/skills.go:25-31` — `workflowSkills` is the source-of-truth table mapping skill name to template path; `spektacular:internal/agent/skills.go:41-66` — `installWorkflowSkills` mustache-renders each against `{"command": cfg.Command}` and writes `<projectPath>/<skillsDir>/<name>/SKILL.md`.
- `spektacular:internal/agent/commands.go:18-24` — `workflowDescriptions` must gain an entry too, for the agents that install command wrappers (bob).
- `spektacular:internal/agent/claude.go:16-44`, `bob.go:13-36`, `codex.go:13-33` — per-agent install targets `.claude/skills/`, `.bob/skills/` + `.bob/commands/`, `.agents/skills/`.
- `spektacular:templates/skills/workflows/spek-plan/SKILL.md` — the model for a workflow-driving skill: version-check block, the standing "STOP … a single successful CLI call is NOT task completion" loop discipline, the instruction→perform→`goto`→repeat loop description, and the resume-report branch (`resumable`, `kind`, `current_step`, `--force`).
- `spektacular:cmd/version.go:36-103,118-146,276-285` — `version check` compares the binary's version against `<dataDir>/version`, written only by `init` (`spektacular:cmd/init.go:55-58`). Any template change ships with a re-init requirement, which the skills' version-check block surfaces.

### Test surfaces that must be extended

- `spektacular:templates/context_directive_test.go:19-25` — `stepDirs` is a fixed list of `steps/spec`, `steps/plan`, `steps/implement`. A new `steps/repo` directory is **not** covered until added here; once added, the walk auto-covers every new template.
- `spektacular:templates/skill_resume_test.go` — fixed `workflowSkills` list asserting resume-report markers (`resumable`, `--force`, "resume"/"start a new") and the absence of `state.json`; the repo-workflow skill belongs in it.
- `spektacular:templates/repo_source_test.go` — walks all step and skill templates for banned stand-in phrases; auto-covers new templates.
- `spektacular:templates/rejection_repair_directive_test.go`, `spektacular:templates/section_drafting_test.go`, `spektacular:templates/work_files_test.go` — fixed lists scoped to spec/plan gathering steps; the repo workflow does not join these (it drafts no document sections and uses no working files).
- `spektacular:cmd/cross_kind_test.go` — drives the real cobra tree via `runRootCmd`, seeds state with `writeInProgressState`, and asserts the on-disk state bytes are byte-identical before and after a refused cross-kind op. This is the exact pattern for proving a guided add neither disturbs nor is blocked by an in-progress spec.
- `spektacular:cmd/repo_test.go:159-175,185-238` — `minimalFootprint` (a hand-maintained in-code path list) plus the representative `TestRepoAdd_RegistersEntryAndCreatesMinimalFootprint`, which asserts the JSON result, the registry YAML (unmarshalled to `map[string]any` to catch stray fields), the footprint tree, and the target's `repo.yaml`.
- `spektacular:tests/harbor/spec-workflow/tests/test_spec_workflow.py:93-125,163-172` — `extract_tool_calls` / `iter_transcript_objects` / `extract_builtin_file_edits` parse the real agent transcript under `/logs/agent`. This is the only mechanism in the project capable of checking the transcript-level acceptance criteria (one question per exchange, no internal vocabulary in what the user was shown, delegation records the proposals, confirmation names repository/folder/code location). Harbor suites are run by hand via `make harbor-test-*` (`spektacular:Makefile:43-88`) and are not in CI.

### Docs surfaces

- `docs:src/pages/projects.mdx:159-194` — `<Section heading="Registering a repository">`, `surface` unset (false). Lines 161-169 frame the add as "one command"; 171-177 and 179-186 carry the three `repo add --data '{...}'` examples that supply `name`, `location`, `description`, `role` as arguments. This is the passage the documentation requirement targets.
- `docs:src/pages/projects.mdx` band order and `surface` values: Hero (13), Section "What a multi-repo project is" (18, false), Section "Where a repository's files live" (110, true), Section "Registering a repository" (159, false), ConfigurationKeys "How configuration is split" (196, default true), Section "Work that spans repositories" (241, true), Section "Cloning a git source" (276, false), Section "Excluding paths from search" (301, true), CtaBanner (320). Note bands 5 and 6 are both shaded — a pre-existing deviation; any new section must be set opposite its own predecessor rather than copying that adjacency.
- `docs:src/pages/projects.mdx:230-237` — `<ConfigKey name="description, role, tags">` explains that these live only in the repository's own configuration and that planning reads them, which is the existing motivation the new copy should connect to.
- `docs:src/pages/how-it-works.mdx:228-233,235-245,247-248,250-261,263-270` — the house voice for a guided flow: names Flipped Interaction, then a fenced worked dialogue of agent/you exchanges, then the "stops once no further question would change the draft" and "presented back for you to confirm or correct, rather than asked from a blank prompt" phrasing. The guided add's copy should mirror this shape, including a fenced dialogue.
- `docs:src/pages/configuration.mdx:180-209,213-254,269-287` — the `repos` registry key and the `repo.yaml` `description`/`role`/`tags`/`source` key reference; `configuration.mdx:222-225` mentions `repo add` only as the creation trigger. Low-risk, optional touch-up.
- `docs:src/components/Nav.astro:15-16` — `/projects/` and `/configuration/` already sit under the Resources group; no nav change is implied.
- `docs:package.json:7-12` — `npm run build`; `npx astro check` via the `@astrojs/check` devDependency. `docs:.github/workflows/deploy.yml:38-41` runs only `npm run build`. The `grep -nE "<div|<section|class=" src/pages/*.mdx` guard is documented in the mdx-authoring convention, not enforced in CI.

## Files examined

- `spektacular:internal/workflow/workflow.go:17-303` — engine: `Config` (with `Kind`), `StepConfig`, `StepCallback`, `New(steps, statePath, cfg, st, out)`, `Next`, `Goto`, `before_<event>` callback registration, `enter_state` persistence, `commitTerminal`.
- `spektacular:internal/workflow/state.go:14-61` — persisted `State` shape (`kind`, `current_step`, `completed_steps`, timestamps, `data`), `InProgress()`'s hardcoded `"finished"`, `saveState`.
- `spektacular:internal/workflow/data.go:6-40` — `Data` interface and the persisted-base/in-memory-overlay `mapData`.
- `spektacular:cmd/resume.go:22-181` — `resumeInstruction`, `mismatchInstruction`, `emitResumeReport`, `readState`, `detectInProgress`, `resumeOrClear`, `guardKind`; `--force` is a plain `os.Remove(statePath)`.
- `spektacular:cmd/root.go:187-215,307-327` — `readStateSnapshot`'s private `stateSnapshotFile` mirror of the state shape, `dataDir()`, `projectRoot()`; also `runUnknownSubcommand` (150-165 region) which every command group uses.
- `spektacular:cmd/spec.go:106-108,192-203,300-309,348-355` — `stateFilePath`, the `new` handler's resume-then-seed-then-`Next` shape, the `goto` handler's `guardKind`-then-`Goto` shape, `--dry-run`'s `.dryrun-tmp` state diversion.
- `spektacular:cmd/plan.go:103-114,194-253` — the same shape for plan.
- `spektacular:cmd/repo.go:16-32,38-113,113-252,255-329,332-358,384-404` — the whole `repo` command family: `repoCmd`/`repoAddCmd`/`repoListCmd`, the input/output schemas, `repoAddInput`, `runRepoAdd`, `repoInfo`, `runRepoList`, `repoAddData` validation (name and location required, `address` rejected), `repoConfigDescriptiveFieldsEmpty`, command registration. **There is no `repo repair` subcommand** — repair is a status value and a behaviour of `EnsureFootprint`, reached by re-running `repo add` or `init`.
- `spektacular:cmd/repo.go:130-138` — the hardcoded `entry.Location = filepath.Join(input.Location, ".spektacular")`: the footprint always lands inside the folder named by `location`, so "separate" placement is purely a matter of the caller passing a project-side `location` plus a `source`. No flag, config, or heuristic chooses between them in Go today.
- `spektacular:internal/config/config.go:155-200,211-223,258-296,331-357` — `RepoEntry` (name, location, deprecated `local` alias, dependencies, provider; deliberately no description/role/tags), `ResolvedLocation` anchored at `<projectRoot>/.spektacular`, `Config.Repos`, YAML round-trip, legacy `address` rejection.
- `spektacular:internal/config/repo.go:17,33-40,63-111,137-143,155-198,231-325` — `repo.yaml`'s `RepoConfig` (description, role, tags, source, knowledge, changelog), loader with `${VAR}` expansion and legacy-shape rejection, `DefaultRepoSource` (`file`, location `..`), `RepoSourceConfig`, `classifyLegacySource`/`ParseSource`/`SourceFromInput`/`resolveFileSource`, `ToYAMLFile`.
- `spektacular:internal/repo/set.go:12-15,57-313` — `Set`, `Entries`, `Present`, `LocalRoot`, `LocalSource`, `DescriptiveMetadata`, `Resolve`/`ResolveAll`, git-source cloning into `<projectRoot>/.spektacular/repos/<name>/`, `staleNote` (warn-only, never fetches).
- `spektacular:internal/repo/footprint.go:13-17,19-85` — `EnsureFootprint` and its three status values; strictly additive, never clobbers an existing README or a parsable `repo.yaml`.
- `spektacular:internal/repo/git.go:22-93` — `GitRunner` (`Clone`, `LocalHead`, `RemoteHead`) and the non-interactive `execGitRunner`.
- `spektacular:internal/stepkit/stepkit.go:20-127` — `PathStrategy`, `StepRequest` (incl. `Extra`), `ResultBuilder`, `WriteStepResult`'s merge order, `StepTitle`, `GetString`, `RenderTemplate`.
- `spektacular:internal/steps/spec/steps.go:22-39,53-65,101-141` — `Steps()` table, the local `writeStep` wrapper, one-line gathering callbacks.
- `spektacular:internal/steps/plan/steps.go:30-51,69-97` — the same, plus `stepkit.RepoRosterExtra(data)` as the precedent for pushing computed facts into an instruction via `Extra`.
- `spektacular:internal/output/writer.go:31-80` — `ErrorResponse` (`error`, `code`, `message`, `resource`, `state`, `next_action`), `NewError`, `WithResource`, `WithState`, `WithNextAction`; the shape every new failure path must use.
- `spektacular:internal/agent/skills.go:25-66`, `commands.go:18-47`, `claude.go:16-44`, `bob.go:13-36`, `codex.go:13-33`, `managed_section.go:31-113` — skill and managed-section install machinery, per-agent targets, install-time `{{command}}` substitution.
- `spektacular:templates/templates.go:6` — `//go:embed all:*`.
- `spektacular:templates/steps/spec/00b-interview.md` — full house style for a conversational step, including the Flipped Interaction framing this feature reuses.
- `spektacular:templates/steps/plan/01-overview.md`, `02-discovery.md:62-71` — the judgement-call/`assumptions.md` and "proceed unless genuinely blocked" patterns.
- `spektacular:templates/steps/resume.md`, `resume_mismatch.md` — the resume and cross-kind instruction text, and their spec/plan-vs-implement working-file branch.
- `spektacular:templates/skills/workflows/spek-plan/SKILL.md` — the workflow-driving skill model.
- `spektacular:templates/skills/workflows/spek-manage-repos/SKILL.md` — the current static playbook, with uncommitted edits carrying the vocabulary rule and an eight-step add flow. The spec designates this an illustration, not a design to preserve.
- `spektacular:templates/context_directive_test.go:14-36`, `work_files_test.go:23-48`, `rejection_repair_directive_test.go`, `section_drafting_test.go`, `repo_source_test.go`, `skill_resume_test.go`, `skill_list_command_test.go` — the template-contract test suite; which are directory-walked and which are fixed lists.
- `spektacular:cmd/repo_test.go:41-49,70-94,131-175,185-238,495-535,688-711,823-857` — `resetRepoFlags`, `repoProject`, `repoAddJSON`, `swapRepoGit`/`stubGit`, `listPaths`, `minimalFootprint`, the representative registration test, staleness, schema and unmaterialized-metadata tests.
- `spektacular:internal/workflow/workflow_test.go` — engine tests including `TestKindPersistedOnNew`, `TestInProgress`, `TestMultiSourceTransition`, `TestAutoSaveOnTransition`, `TestFailedStepDoesNotAdvancePersistedState`.
- `spektacular:cmd/cross_kind_test.go` — end-to-end cross-kind refusal tests with byte-identical state assertions.
- `spektacular:tests/harbor/spec-workflow/{task.toml,instruction.md,tests/test_spec_workflow.py,tests/test.sh}` and `spektacular:Makefile:43-88` — the agent-in-a-container end-to-end harness, its transcript parsing helpers, its 900s agent timeout, its artifact capture of `/app/.spektacular`, and the per-agent render-then-`harbor run` make targets.
- `spektacular:README.md:196,199-215` — the published in-repo description of the registry, `repo add`, and the colocated vs separate layouts.
- `spektacular:go.mod:5-12` — the complete dependency set.
- `spektacular:.spektacular/knowledge/architecture/workflow-steps.md` — the recorded step-authoring architecture (note: it documents an older 3-arg `StepCallback`; the live signature takes `store.Store` and returns `(string, error)`).
- `spektacular:.spektacular/knowledge/architecture/working-with-files-from-steps.md` — the `store.Store` interface and the "callbacks must not write files directly" rule, which the chosen `register` step deliberately qualifies.
- `spektacular:.spektacular/knowledge/gotchas/fsm-cancel-only-works-before-transition-commits.md` — why callbacks register as `before_<event>`.
- `docs:src/pages/projects.mdx:13-320`, `configuration.mdx:180-320`, `how-it-works.mdx:212-330`, `src/components/Nav.astro:6-21`, `src/components/sections/*.astro`, `package.json:7-24`, `.github/workflows/deploy.yml:38-41` — the documentation surfaces, component inventory, and build commands.

## External references

- White et al., "A Prompt Pattern Catalog to Enhance Prompt Engineering with ChatGPT", arXiv:2302.11382 — the Flipped Interaction pattern. Cited by name in the spec's Technical Approach and already cited verbatim in `spektacular:templates/steps/spec/00b-interview.md`; the guided add is a narrower instance of the same pattern, so the new templates should cite it the same way for consistency.
- `github.com/looplab/fsm` v1.0.3 — the FSM the workflow engine wraps. Its `Event()` runs `beforeEventCallbacks` before committing `f.current` and before `enter_state`, which is what makes a `before_<event>` callback able to veto a transition and its persistence.
- `github.com/cbroglie/mustache` v1.4.0 — the template engine. Section (`{{#x}}`) and inverted-section (`{{^x}}`) tags are what give the step templates their "evidence present / nothing readable" branches without Go-side branching.
- Harbor (harborframework.com) — the container harness that runs a real coding agent against the CLI and captures its transcript; the only place transcript-shaped acceptance criteria can be verified.

## Prior plans / specs consulted

- `spektacular:.spektacular/plans/000043_flipped-interaction-spec-interview/plan.md` — the closest precedent. Establishes that a conversational change of this kind is implemented almost entirely as step-template instruction prose plus one new FSM step; shows the exact mechanical shape of inserting a step into a `Steps()` table and retargeting its neighbours' `Src`/`Dst`; and records the decision to relocate shared helpers rather than reach across packages. Its "no new Go types, reuse every existing contract" framing is the altitude this plan should aim for, with the deliberate exception of the second state file and the examiner.
- `spektacular:.spektacular/plans/000046_relocatable-repo-footprint/plan.md` — establishes the current `location` + `source` model, why the registry holds membership only while `repo.yaml` holds description/role/tags/source, and how the docs and README were kept in step with a behaviour change. Its Conventions section is the model for this plan's own.
- `spektacular:.spektacular/specs/000043_flipped-interaction-spec-interview.md` — the interview requirements the guided add narrows: adaptive questions, an explicit stopping condition, draft-then-confirm rather than ask-from-blank.
- `spektacular:.spektacular/context.md` — the working context carried forward from this feature's own spec phase. Records the three verbatim user corrections about leaked internals, the diagnosis that the leak is a taught vocabulary rather than a set of bad sentences, the user's own words choosing step-by-step-with-suggestions, and the alternatives already rejected during the spec (pure step-by-step interrogation; infer-everything-with-one-confirmation; ask-only-what-was-not-stated).

## Open assumptions

If any of these turns out to be wrong, the implement workflow must STOP and ask.

1. **`repo new` / `repo goto` is the acceptable command surface.** Chosen for consistency with `spec new`/`plan new`, but `repo new` sits confusingly close to `repo add` in a way `spec new` does not sit close to anything. If the user prefers a different verb, only the command registration, the skill text, and the docs copy change; the FSM, templates, and examiner are unaffected.
2. **A `repo status` / `repo steps` parity command is not required.** Resume detection is delivered entirely by `repo new` returning a resume report, so the surface stays at two commands. Assumed, not confirmed.
3. **The guided workflow's state file is not tracked by the debug session log.** `spektacular:cmd/root.go:187-215` reads only `state.json`, and `isWorkflowStart` (`cmd/root.go:~150-165`) hardcodes `spec`/`plan`/`implement`. A guided add would therefore not open a session log. Assumed acceptable; extending it is a one-line change if not.
4. **The `register` step callback may write outside `.spektacular/`.** `spektacular:.spektacular/knowledge/architecture/working-with-files-from-steps.md` says callbacks must not write files directly and must go through `store.Store`, which is rooted at `.spektacular/`. Registration necessarily writes into the target repository. Reading that rule as "do not do ad-hoc file plumbing" rather than "never call a domain operation that persists", satisfied by routing through the same shared registration function `repo add` uses.
5. **Writability is probed by attempting a real filesystem operation.** Permission bits alone are unreliable across the platforms in play (WSL2, macOS, containers). Assumed a create-then-remove probe in the target folder is acceptable; it is a write to a repository before confirmation, albeit a transient one that leaves nothing behind.
6. **Existing `repo add` callers pass `location` relative to the project's `.spektacular/` directory.** Inferred from `RepoEntry.ResolvedLocation` anchoring at `<projectRoot>/.spektacular` (`spektacular:internal/config/config.go:195-200`) and from this project's own `config.yaml` (`location: ../../spektacular-website/.spektacular`) and the docs' `location: ../../docs` example. The guided flow must compute the same relative form when it proposes a placement, and get it wrong in neither direction.
7. **The harbor suite is the acceptance vehicle for the transcript-shaped criteria, and is run by hand.** No CI runs `make harbor-test-*`. Criteria such as "no internal vocabulary appears in the transcript" therefore ship as a manually-run suite plus a Go-level template-contract test, not as a CI gate.
8. **`templates/steps/resume.md` can be extended with a repo branch rather than needing a separate repo resume template.** Its current text branches on spec/plan vs implement for working files; a third branch is small. Assumed; a dedicated template is the fallback.

## Drafting assumptions

The judgement calls made while drafting this plan, each a decision taken on a reasonable default rather than put to the user. They are presented for challenge at the walkthrough.

### Command surface is `repo new` / `repo goto` (discovery)
- **Decision**: the guided flow gets its own commands, `repo new` to start and `repo goto` to advance, mirroring `spec new`/`spec goto` exactly.
- **Rationale**: the spec's binding constraint is to follow the convention established by the spec and plan workflows, and those are `<noun> new` + `<noun> goto`. It also keeps `repo add` untouched, which a second constraint requires outright.
- **Rejected**: `repo add --guided` (modifies the command that must stay unchanged, and cobra flag state on `repoAddCmd` is package-level); bare `repo add` with no `--data` entering the guided flow (deletes an asserted error path and turns a scripted caller's typo into an interactive session).

### No `repo status` or `repo steps` parity command (discovery)
- **Decision**: the guided workflow ships two commands, not four.
- **Rationale**: `repo new` returning a resume report is the whole of what the resume requirement needs; `spec status`/`spec steps` exist for introspecting long multi-document workflows, which a nine-step add is not.
- **Rejected**: full parity with the spec/plan command surface, as speculative surface area with no requirement behind it.

### Gathered answers live in the workflow data map, not in working files (discovery)
- **Decision**: name, description, role, tags, location and placement are seeded through `--data` on each `goto` and persisted by the engine in the repo workflow's state file.
- **Rationale**: they are four short scalar values, and the engine already persists `Data` on every transition, so resume retains them with no new mechanism, no directory to create, and nothing to clean up.
- **Rejected**: per-section working files under `.spektacular/work/<name>/` as spec and plan use, since that convention exists to assemble multi-paragraph document sections later, which does not apply here.

### The CLI gathers the evidence, the agent writes the proposal (discovery)
- **Decision**: a Go examiner reads the README, the manifest, the top-level layout and the languages present, and the step callback pushes that evidence into its own instruction; the agent turns it into a proposed name, description, role and tags.
- **Rationale**: it puts a testable oracle behind "proposals are drawn from the repository, not the folder name", bounds the read against the no-deep-analysis constraint, and still lets prose be written by the only component capable of writing prose. There is no language model and no language-detection dependency in the binary.
- **Rejected**: the agent examining with its own file tools (today's behaviour, untestable and unbounded); the CLI producing the finished values (degenerates into the folder-name restatement the acceptance criteria forbid).

### Chosen direction: a fourth CLI-owned workflow on its own state file (architecture)
- **Decision**: the guided add becomes a linear ten-state FSM in a new `internal/steps/repo` package, persisted to `.spektacular/repo-state.json` under the kind `repo`, driven by new `repo new` and `repo goto` commands, with a Go examiner supplying bounded evidence to the proposal steps and a shared registration core called by both `repo add` and the workflow's `register` step.
- **Rationale**: it satisfies the spec's mandated mechanism and its non-competition constraint structurally rather than behaviourally, and costs no engine change at all, because `workflow.New`, `resumeOrClear` and `guardKind` already take the state path and the kind as parameters.
- **Rejected**: a single state file holding several concurrent workflows keyed by kind (four independent readers of the current shape would break together, for one new caller); a stateless renderer carrying answers only in the agent's context (fails the resume requirement outright).

### Conditional questions live in the instruction, not in the step graph (architecture)
- **Decision**: `locate` and `placement` remain unconditional states whose templates branch on a mustache inverted section, so a repository already named up front and a writable default placement both pass through without a question. Only the delegation shortcut is expressed in the graph, as a multi-source `Src` on `placement`.
- **Rationale**: it matches how the spec workflow already handles a step that may have nothing to ask, keeps the FSM linear and readable, and keeps `completed_steps` an honest record of the path taken. A branching graph would need conditional callbacks and would make resume harder to reason about.
- **Rejected**: modelling every conditional as a graph branch; asking the questions anyway and letting the agent discard the answers.

### Vocabulary containment is tested at two levels (architecture)
- **Decision**: a Go template-contract test asserts every repo step template carries the standing "internals are yours, not the user's" rule, and a harbor container suite parses the real agent transcript for the banned terms.
- **Rationale**: the criterion is phrased about what the user was shown, which only a transcript can answer, but harbor suites are run by hand and not in CI, so a fast Go-level check has to guard the templates themselves on every commit.
- **Rejected**: relying on the harbor suite alone (no CI coverage); relying on the template test alone (proves the rule is present, not that it was followed).

### The repo name is seeded provisionally at `locate` (data structures)
- **Decision**: the `locate` step seeds the workflow data's `name` with the target folder's base name, and the `name` step then proposes a better one from the manifest identity when there is one and asks the user to confirm or correct it.
- **Rationale**: the shared step-rendering pipeline reads the workflow instance name from `data["name"]`, and so does the resume report. Without a provisional value, an add interrupted before the name is agreed reports an unfinished workflow with an empty name. Seeding it costs nothing and the value is superseded moments later.
- **Rejected**: leaving it empty until the name step (produces an unreadable resume report); using the folder name as the final proposal (the acceptance criteria require the description and tags to come from the repository's contents, and the manifest identity is a better name source when present).

### `Examine` returns a value, never an error (data structures)
- **Decision**: the examiner reports readability as a field and has no error return.
- **Rationale**: the spec requires that a repository offering nothing to read is still addable, with the questions asked without proposals rather than the add failing. Making unreadability an ordinary value keeps that path in the instruction templates, where it belongs, instead of in error handling.
- **Rejected**: returning `(Evidence, error)` and treating a missing README as an error condition, which would make the fallback path an error path and invite a caller to abort on it.

### Four milestones, ten phases, with the conversation content late (phases)
- **Decision**: the machine is built first (state machine and commands, then evidence, then the write), and the instruction content that carries the actual conversation is written afterwards, in one phase, in one voice.
- **Rationale**: an instruction template can only state a suggestion once the variable carrying that suggestion exists, so writing the conversation before the facts are wired would mean writing it twice. Concentrating all ten instructions in one phase is also the only way they read as one voice rather than ten.
- **Rejected**: writing each step's full instruction alongside the callback that feeds it, which spreads one document across four phases and guarantees drift in tone.

### Registration lives in `internal/repo`, not in `cmd` (phases)
- **Decision**: the extracted `Register` function and the relocated metadata-note helper go into `internal/repo` rather than staying as unexported helpers in `cmd`.
- **Rationale**: the step callbacks that need it are in `internal/steps/repo`, and a step package importing `cmd` would invert the dependency direction the whole codebase follows. `internal/repo` already imports `internal/config` and already owns footprint creation, so it is the natural home and creates no new edge.
- **Rejected**: an unexported helper in `cmd` called by a `cmd`-level shim (leaves the step package unable to reach it); a new package for registration alone (splits repository concerns across two packages for one function).

## Rehydration cues

To rebuild this context cold:

1. `go run . repo list` — the two repos and the absolute `root` each one's code lives at. Never assume the working directory holds a repo's code.
2. `go run . knowledge always-applied --tier repo --filter spektacular --filter docs` — the conventions and glossary this plan must honour.
3. `go run . knowledge read --data '{"tier":"repo","name":"spektacular","path":"architecture/workflow-steps.md"}'` — the step-authoring architecture. Read it knowing the recorded `StepCallback` signature is stale; the live one is in `internal/workflow/workflow.go:41`.
4. `go run . knowledge read --data '{"tier":"repo","name":"spektacular","path":"architecture/working-with-files-from-steps.md"}'` and `.../gotchas/fsm-cancel-only-works-before-transition-commits.md`.
5. Read, in order: `internal/workflow/workflow.go`, `internal/workflow/state.go`, `cmd/resume.go`, `cmd/repo.go`, `internal/stepkit/stepkit.go`, `internal/steps/spec/steps.go`.
6. Read `templates/steps/spec/00b-interview.md` for the conversational-step house style and `templates/skills/workflows/spek-plan/SKILL.md` for the workflow-driving-skill house style.
7. `go run . plan file read 000043_flipped-interaction-spec-interview/plan.md` and `go run . plan file read 000046_relocatable-repo-footprint/plan.md` — the two closest precedents.
8. `go test ./...` — the baseline. It passed in full at the time this plan was researched.
9. For the docs repo, read `src/pages/projects.mdx` (the "Registering a repository" section) and `src/pages/how-it-works.mdx:212-330` (the house voice for describing a guided flow).
