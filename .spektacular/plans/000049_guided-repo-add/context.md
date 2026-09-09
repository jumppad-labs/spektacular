---
created_date: "2026-09-07"
status: completed
closed_date: "2026-09-07"
---

# Context: 000049_guided-repo-add

## Current State Analysis

**The workflow engine is already general; only the command layer assumes a singleton.** `internal/workflow/workflow.go:73` takes `statePath` as a construction parameter, `Config.Kind` (`workflow.go:17-29`) is stamped onto a freshly created state at `workflow.go:87`, and `internal/workflow/state.go:14-21` is the persisted shape. The singleton assumption lives entirely in `cmd`: `stateFilePath(dataDir)` (`cmd/spec.go:106-108`) always returns `.spektacular/state.json`, and `dataDir()` (`cmd/root.go:307-315`) carries the comment saying spec and plan share it. `cmd/resume.go:133-152` (`resumeOrClear`) and `cmd/resume.go:169-181` (`guardKind`) already take both the state path and the expected kind as parameters, so both are reusable against a different file. `--force` is a plain `os.Remove(statePath)` at `cmd/resume.go:135`.

Two engine details are load-bearing for this plan. Callbacks are registered as `before_<event>` (`workflow.go:113-133`) so a callback error vetoes the transition before `enter_state` persists (`workflow.go:147-156`); the reason is recorded in `.spektacular/knowledge/gotchas/fsm-cancel-only-works-before-transition-commits.md`. And `State.InProgress()` (`state.go:25-27`) compares against the literal step name `"finished"`, so a new workflow's terminal step must be called that.

**The state file has four independent readers**, which is why it is not being reshaped: `cmd/root.go:187-215`'s private `stateSnapshotFile` mirror, `internal/sessionlog`, `cmd/cross_kind_test.go`'s `writeInProgressState` fixture helper, and the harbor python oracles which index `state["data"]["name"]` (`tests/harbor/spec-workflow/tests/test_spec_workflow.py:53-58`).

**Registration is one linear handler.** `runRepoAdd` (`cmd/repo.go:113-252`) validates through `repoAddData` (`cmd/repo.go:332-358`, requiring `name` and `location`, rejecting the legacy `address` key), then computes `entry.Location = filepath.Join(input.Location, ".spektacular")` (`cmd/repo.go:130-138`), upserts into the registry by name, calls `EnsureFootprint` (`internal/repo/footprint.go:28-85`), writes the descriptive fields into the repository's own `repo.yaml`, and assembles `{registered, footprint, metadata_note}` with the note coming from `repoConfigDescriptiveFieldsEmpty` (`cmd/repo.go:384-386`, also called by `runRepoList` at `cmd/repo.go:308-310`).

**Colocated versus separate placement is not decided anywhere in Go.** The footprint always lands inside the folder named by `location`; "separate" is achieved purely by a caller passing a project-side `location` plus a `source`. The only place that decision is made today is the repository-management skill, which is why it leaked terminology. `config.DefaultRepoSource` (`internal/config/repo.go:137-143`) is a file source of `..`, which is what makes a colocated footprint treat its parent as the code.

**There is no `repo repair` command.** `cmd/repo.go:400-404` registers only `add` and `list`. Repair is a status value (`internal/repo/footprint.go:13-17`) and a behaviour of `EnsureFootprint`, reached by re-running `repo add` or `init` (`internal/project/init.go:167-190`).

**Nothing reads a repository to characterise it**, and nothing in `go.mod:5-12` helps: the dependency set is mustache, looplab/fsm, go-gitignore, cobra, testify and yaml. The behaviour this feature specifies exists today only as prose, in the eight-step flow currently uncommitted in `templates/skills/workflows/spek-manage-repos/SKILL.md`, which that file itself describes as a static playbook with no session to resume.

**Templates and skills.** `templates/templates.go:6` embeds the whole tree with `//go:embed all:*`. Step name to template file is a hardcoded string per callback, not a naming convention. `internal/stepkit/stepkit.go:55-94` merges standard vars, then `PathStrategy.PathVars`, then per-call `Extra`, and renders through mustache at `stepkit.go:121-127`, so section and inverted-section tags give a template its branches without Go-side branching. Skills install from the table at `internal/agent/skills.go:25-31` via `installWorkflowSkills` (`skills.go:41-66`), rendered against `{"command": cfg.Command}` at install time.

**Test baseline.** `go test ./...` passes in full on `6c72adc`. The template-contract suite is a mix of directory walks and fixed lists: `templates/context_directive_test.go:19-25`'s `stepDirs` is an allow-list of three workflow directories, `templates/skill_resume_test.go` a fixed skill list, `templates/repo_source_test.go` a walk. `cmd/repo_test.go` covers `repo add` and `repo list` end to end through the real cobra tree, with `t.TempDir()` plus `t.Chdir`, a `stubGit` swapped in via `swapRepoGit`, and a hand-maintained `minimalFootprint` path list at `cmd/repo_test.go:159-175`.

## Per-Phase Technical Notes

### Phase 1.1: One way to register a repository

**File changes**

- `cmd/repo.go:113-252` — split `runRepoAdd`. Everything from the `loadConfig()` call at 125-128 onward moves out: the `entry.Location = filepath.Join(input.Location, ".spektacular")` computation (130-138), the idempotent by-name upsert into `cfg.Repos` (139-~195), the source parse via `config.SourceFromInput` (226-233), `EnsureFootprint` (200), the write of description/role/tags into the repo's own `repo.yaml`, and the result assembly including the `metadata_note` from `repoConfigDescriptiveFieldsEmpty` (246-249). What stays in `runRepoAdd`: the `--schema` short-circuit (114-116), `repoAddData(cmd)` (118-121), and the `output.Write` of the returned result.
- `internal/repo/register.go` (new) — `Registration` and `RegistrationResult` types plus `Register(cfg *config.Config, projectRoot string, in Registration) (RegistrationResult, error)`, holding the moved body verbatim. Reuses `EnsureFootprint` (`internal/repo/footprint.go:28-85`) and `config.SourceFromInput` (`internal/config/repo.go:288-304`) unchanged. Note the package already imports `internal/config` (`internal/repo/set.go`), so no new dependency edge is created.
- `cmd/repo.go:384-386` — `repoConfigDescriptiveFieldsEmpty` moves to `internal/repo` alongside `Register`, since it is the source of `MetadataNote`. Its second caller, `runRepoList` at `cmd/repo.go:308-310`, switches to the relocated symbol.
- `internal/repo/register_test.go` (new) — direct unit coverage of `Register`: fresh registration, idempotent no-op on an identical entry, in-place update on a changed entry, footprint repair status, and the metadata note firing only when description, role and tags are all empty.
- Do **not** edit `cmd/repo_test.go`. Its 25 tests, the `minimalFootprint` list (`cmd/repo_test.go:159-175`) and the schema assertions (`688`, `711`) are the specification of "unchanged". An edit needed here means the extraction changed behaviour.

**Complexity:** Low
**Token estimate:** ~25k
**Agent strategy:** Single agent, sequential. This is one mechanical move with one relocation; parallelism would only create merge friction in a single file.

### Phase 1.2: A repository that can describe itself

**File changes**

- `internal/repo/examine.go` (new) — `Evidence` struct and `func Examine(dir string) Evidence`. Reads, in order: a case-insensitive `README*` at the top level, taking prose up to a bounded length and skipping the leading heading; the first present manifest among `go.mod`, `package.json`, `Cargo.toml`, `pyproject.toml`, `pom.xml`, `Gemfile`, extracting the declared name and, where the format carries one, the declared description; `os.ReadDir(dir)` for top-level entries, directories first, excluding dotfiles; and a bounded extension histogram over the top two directory levels only, mapped to language names. No recursion beyond that depth, no file contents read for the histogram, and a hard cap on entries walked so the whole call stays well inside a conversational pause. `Readable` is true when at least one of `Readme`, `Identity`, `Summary` or `Languages` is populated.
- `internal/repo/examine_test.go` (new) — table-driven over `t.TempDir()` trees built in-test, matching the fixture style of `internal/repo/footprint_test.go`. Cases: Go repo with README and `go.mod`; JavaScript repo with `package.json` carrying a `description`; a repo with a README but no manifest; a repo with a manifest but no README; an empty directory; a directory that does not exist; a README whose only content is a heading. Assert the identity comes from the manifest and not from `filepath.Base(dir)` — this is the oracle behind the "not merely a restatement of the folder name" criterion, so the test's Go-repo fixture must deliberately use a directory name that differs from the module name.
- Honour the existing exclusion mechanism: `internal/repo` already resolves repos that may carry `.spektacular_ignore`; the examiner should skip ignored paths using the same `sabhiram/go-gitignore` reader the knowledge listing uses, so vendored and build directories do not dominate the language histogram.

**Complexity:** Medium
**Token estimate:** ~35k
**Agent strategy:** Single agent, sequential. The function and its table of fixtures are one unit of thought; splitting them across agents produces fixtures that do not match the parser.

### Phase 2.1: A guided add that can be started, advanced and resumed

**File changes**

- `internal/steps/repo/steps.go` (new) — `Steps()` returning the ten `workflow.StepConfig` entries in the shape of `internal/steps/spec/steps.go:25-39`, with `placement` carrying `Src: []string{"name","description","role","tags"}` and the terminal step named `finished` (required by `internal/workflow/state.go:25-27`). Local `writeStep` wrapper mirroring `internal/steps/spec/steps.go:53-65`.
- `internal/steps/repo/strategy.go` (new) — `stepkit.PathStrategy` implementation supplying `repo_name` and `repo_path` (the absolute folder the repository's code lives at, derived from the `location` in workflow data), with `PrimaryPathField()` returning `"repo_path"`.
- `internal/steps/repo/result.go` (new) — the `stepkit.ResultBuilder` producing `{step, repo_name, repo_path, instruction}`, mirroring `internal/steps/spec`'s builder at `steps.go:42-49`.
- `cmd/repo.go` — add `repoNewCmd` and `repoGotoCmd` beside `repoAddCmd`/`repoListCmd`, registered in `init()` at `cmd/repo.go:400-404`. `runRepoNew` follows `runSpecNew`'s shape (`cmd/spec.go:192-203`): resolve `dataDir()`, `projectRoot()`, `loadConfig()`, build the state path, call `resumeOrClear(repoStatePath, cfg.Command, "repo", force)` and return early when handled, then `workflow.New(reposteps.Steps(), repoStatePath, wfCfg, store.NewSourceStore(root,"project"), out)` with `wfCfg.Kind = "repo"`, seed any `location` from `--data`, and `wf.Next()`. `runRepoGoto` follows `runSpecGoto` (`cmd/spec.go:348-355`): `guardKind(repoStatePath, cfg.Command, "repo")`, then `workflow.New(...)`, seed every non-`step` field of `--data` into workflow data via the existing `readInputIntoWorkflow` helper (`cmd/spec.go:121`), then `wf.Goto(step)`.
- `cmd/repo.go` — add `repoStateFilePath(dataDir string) string` returning `filepath.Join(dataDir, "repo-state.json")`, a sibling of `stateFilePath` (`cmd/spec.go:106-108`). `--dry-run` suffixes `.dryrun-tmp` as the others do.
- Tracking: `state.json` is committed, not ignored (`.spektacular/.gitignore` lists only `*.tmp`, `*.log`, `.env` and `repos/`), so `repo-state.json` is committed too and needs no gitignore change. Note that the `repos/` entry there is scoped to `.spektacular/repos/`, the clone directory, and does not affect the `<projectRoot>/repos/<name>` folder a declined placement writes to, which is correctly tracked because it holds that repository's knowledge and changelog.
- `templates/steps/repo/00-new.md` … `09-finished.md` (new) — ten structural templates at this phase: `## Step {{step}}: {{title}}`, a one-line statement of the step's job, the `{{config.command}} repo goto --data '{"step":"{{next_step}}"}'` line, then `---` and the standing refresh-context footer verbatim from `templates/steps/spec/00b-interview.md`. Full instruction content lands in Phase 2.4.
- `templates/context_directive_test.go:19-25` — add `"steps/repo"` to `stepDirs`. This is an allow-list, not a discovery; without it the new templates are uncovered.
- `cmd/repo_workflow_test.go` (new) — `runRootCmd`-driven coverage in the style of `cmd/spec_test.go`: full forward walk, refused out-of-order `goto` with a `next_action`, resume report on a second `repo new`, `--force` discard, and answers persisted across a rebuild of the workflow from the same state path.
- `cmd/cross_kind_test.go` — new cases: a guided add runs to completion with an in-progress spec seeded by `writeInProgressState`, asserting `state.json`'s bytes are identical before and after and that the spec then resumes at its recorded step; and `spec new` while an add is in progress is not refused.

**Complexity:** High
**Token estimate:** ~90k
**Agent strategy:** Parallel analysis, sequential integration. One agent maps `cmd/spec.go`'s new/goto handlers and one maps `internal/steps/spec`'s package layout, then a single agent writes the new package, the handlers and the templates in one pass, because the three are one contract.

### Phase 2.2: The flow knows what the repository says about itself

**File changes**

- `internal/steps/repo/steps.go` — the `name` callback resolves the target directory from workflow data's `location`, calls `repo.Examine`, stores the result under `evidence` via `data.Set`, and passes it into `stepkit.StepRequest.Extra` as `map[string]any{"evidence": ...}`. This mirrors how `internal/steps/plan/steps.go:88-97` passes the repo roster.
- `internal/steps/repo/steps.go` — the `description`, `role` and `tags` callbacks read `evidence` back from workflow data and pass it through `Extra` without re-examining. Because `workflow.Data`'s base map is the persisted `State.Data` (`internal/workflow/workflow.go:105`), it survives an interruption for free.
- `internal/steps/repo/steps.go` — the `locate` callback seeds `name` in workflow data with `filepath.Base` of the resolved location, so `stepkit.WriteStepResult`'s instance name (`internal/stepkit/stepkit.go:69`) and the resume report's `name` (`cmd/resume.go:56-80`) are never empty.
- `internal/steps/repo/probe.go` (new) or a helper in `internal/repo` — the writability probe: create and immediately remove a uniquely named temporary entry in the target directory, returning a bool. Permission bits are not consulted; they do not answer the question on the platforms in play. The `placement` callback passes the result through `Extra` as `writable`.
- `templates/steps/repo/*.md` — the four proposal templates and the placement template gain their `{{#evidence.readable}}` / `{{^evidence.readable}}` and `{{#writable}}` / `{{^writable}}` sections. Content is still skeletal at this phase; only the branch structure is wired.
- `internal/steps/repo/steps_test.go` (new) — in the style of `internal/steps/spec/steps_test.go`: assert the rendered instruction for each proposal step contains terms from a seeded fixture repository's README and manifest, that the empty-evidence branch renders instead when the fixture is bare, and that `Examine` is invoked once across a full walk.

**Complexity:** Medium
**Token estimate:** ~45k
**Agent strategy:** 2 parallel agents: one on the callback wiring and the probe, one on the template branch structure and its rendering tests. They meet at the `Extra` variable names, which should be fixed in writing before either starts.

### Phase 2.3: Nothing is written until the user says so

**File changes**

- `internal/steps/repo/steps.go` — the `register` callback assembles a `repo.Registration` from workflow data and calls `repo.Register` from Phase 1.1. Placement translation lives here: `placement == "inside"` sets `Location` to the target folder expressed relative to the project's `.spektacular/` directory and leaves `Source` empty, so the written `repo.yaml` takes `config.DefaultRepoSource` (`internal/config/repo.go:137-143`, a file source of `..`); `placement == "project"` sets `Location` to `repos/<name>` under the project root, again expressed relative to `.spektacular/`, and `Source` to the target folder's absolute path. The relative form matters: `RepoEntry.ResolvedLocation` anchors at `<projectRoot>/.spektacular` (`internal/config/config.go:195-200`), not the working directory.
- `internal/steps/repo/steps.go` — the callback returns an error, not a partial result, on failure. Because `internal/workflow/workflow.go:113-133` registers callbacks as `before_<event>`, that error vetoes the transition before `enter_state` persists (`workflow.go:147-156`), leaving the workflow on `confirm`. Do not move this to an `after_` hook; see `.spektacular/knowledge/gotchas/fsm-cancel-only-works-before-transition-commits.md`.
- `internal/steps/repo/steps.go` — the `finished` callback passes the `RegistrationResult` through `Extra` so the terminal instruction can report what happened.
- Errors on this path are built with `output.NewError(code, message).WithNextAction(...)` per `internal/output/writer.go:60-80` and the project's error-message convention.
- `cmd/repo_workflow_test.go` — cases: at `confirm`, the target directory contains no `.spektacular` and `config.yaml` is byte-identical to its pre-add content; a failing register leaves `repo-state.json`'s `current_step` at `confirm`; a successful add produces a registry entry and a `repo.yaml` carrying the gathered description, role and tags and no metadata note; a `project` placement produces a footprint under `<projectRoot>/repos/<name>/.spektacular` with a `source` pointing at the target and leaves the target with no `.spektacular`.

**Complexity:** Medium
**Token estimate:** ~40k
**Agent strategy:** Single agent, sequential. The placement translation and the confirm-gate guarantee are the same piece of reasoning and must not be split.

### Phase 2.4: The conversation itself

**File changes**

- `templates/steps/repo/00-new.md` — no user-facing question; records the location when supplied and advances.
- `templates/steps/repo/01-locate.md` — the only cold question. `{{#location}}` branch: record it and advance without asking. `{{^location}}` branch: ask which repository, in terms of the folder its code lives in.
- `templates/steps/repo/02-name.md`, `03-description.md`, `04-role.md`, `05-tags.md` — each states a concrete suggestion drawn from `evidence` inside the question, such that agreement alone records it; each asks exactly one thing; each carries the delegation clause naming the shortcut (`repo goto --data '{"step":"placement", ...}'` carrying all four values) so a user who declines to review them individually is asked nothing further. Only `02-name.md` mentions delegation as an offer; the rest handle it if it arrives.
- `templates/steps/repo/06-placement.md` — `{{#writable}}`: record the default silently and advance, unless the conversation has established the repository should hold nothing but code. `{{^writable}}`, or that exception: ask whether a folder may be added, phrased without the placement terms.
- `templates/steps/repo/07-confirm.md` — state the repository being registered, the folder that will be created and where the code lives, and wait for explicit confirmation. No command text, no argument list.
- `templates/steps/repo/08-register.md`, `09-finished.md` — the write step's instruction and the terminal report.
- Every one of the ten carries a standing block, worded once and repeated verbatim, naming the off-limits vocabulary: the words footprint, source, provider, colocated and separate used as terms of art; the name of any configuration file Spektacular writes; and any command name, argument or flag. Model the wording on the uncommitted rule already in `templates/skills/workflows/spek-manage-repos/SKILL.md`, which is the user's own articulation of it. Cite the Flipped Interaction pattern the way `templates/steps/spec/00b-interview.md` does.
- `templates/vocabulary_containment_test.go` (new) — a hand-written stable marker string, asserted present exactly once in every file under `templates/steps/repo/`, following the pattern of `templates/context_directive_test.go:14-36` and `templates/rejection_repair_directive_test.go`. Directory-walked, so future templates are covered automatically.
- Conventions: no em dashes in any of this prose.

**Complexity:** Medium
**Token estimate:** ~50k
**Agent strategy:** Single agent, sequential. This is one voice across ten files; splitting it produces ten voices.

### Phase 2.5: An agent that drives the flow instead of improvising it

**File changes**

- `templates/skills/workflows/spek-manage-repos/SKILL.md` — rewritten in the shape of `templates/skills/workflows/spek-plan/SKILL.md`: the version-check block (unchanged), the standing "a single successful CLI call is not task completion" loop discipline, the instruction-perform-advance-repeat description, the `repo new` start and the resume-report branch on `resumable`/`kind`/`current_step` with `--force` as the discard path. Delete the eight-step add flow and the vocabulary rule from the body, both of which now live in the step templates; the uncommitted edits in this file are superseded rather than built on. Keep the registry-inspection and footprint-repair sections as they are.
- Mirror the rewrite into the two rendered installs that currently carry the uncommitted edits, `.claude/skills/spek-manage-repos/SKILL.md` and `.bob/skills/spek-manage-repos/SKILL.md`, by re-running init rather than hand-editing.
- `internal/agent/commands.go:18-24` — update the `spek-manage-repos` description to describe a guided flow. `internal/agent/skills.go:25-31` needs no change; the skill's name is unchanged.
- `templates/steps/resume.md` — the working-files instruction currently branches spec/plan versus implement. Add the add branch: an interrupted add has no gathered section files, because its answers travel in the workflow itself.
- `templates/skill_resume_test.go` — add `spek-manage-repos` to the fixed `workflowSkills` list so its resume-report markers are asserted.
- `internal/agent/instruction_surface_test.go` — the rewritten skill is re-rendered through the real install path by `TestRenderedSkillsAvoidStdinInstructionSurface`; confirm it still passes rather than editing it.
- Conventions: no em dashes.

**Complexity:** Medium
**Token estimate:** ~35k
**Agent strategy:** Single agent, sequential. One document rewrite plus three small wiring edits.

### Phase 3.1: Proving the conversation, not just the result

**File changes**

- `tests/harbor/repo-workflow/` (new) — modelled file-for-file on `tests/harbor/spec-workflow/`. `task.toml` with an agent timeout sized like the spec suite's 900s and the same `[[artifacts]]` capture of `/app/.spektacular`; `environment/Dockerfile` plus the prebuilt linux binary the make target drops in; `instruction.md` with the `{{agent}}` and skill-invocation placeholders the make target substitutes, seeding a second repository on disk with a README and a manifest for the agent to add, and carrying the spec suite's "you are running non-interactively, you play both roles" framing; `solution/solve.sh`; `tests/test.sh` writing `/logs/verifier/reward.txt`.
- `tests/harbor/repo-workflow/tests/test_repo_workflow.py` (new) — copy `iter_transcript_objects`, `extract_tool_calls` and `extract_builtin_file_edits` from `tests/harbor/spec-workflow/tests/test_spec_workflow.py:93-172`. New assertions: the expected step order from `internal/steps/repo/steps.go` as a hand-maintained oracle; exactly one question precedes the first `repo goto`; each of the four proposal questions appears in its own assistant turn, in order; each contains a value that also appears in the payload of the `goto` that follows it; the confirmation turn names the repository, the folder and the code location and matches no command-text pattern; and a banned-term scan over every assistant turn shown to the user covering footprint, source, provider, colocated, separate as terms of art, `repo.yaml`, `config.yaml`, `state.json`, and any `repo add`/`--data`/`--force` text.
- A second scenario in the same suite, or a second suite directory, exercising delegation: the instruction answers the first proposal with "just use what you think", and the verifier asserts no further proposal questions and a registration carrying the suggested values.
- Negative self-checks: each transcript rule is exercised against a small synthetic transcript fixture that breaks it, so a rule that can never fail is caught in review.
- `Makefile` — a `harbor-test-repo` target beside `harbor-test-spec`/`harbor-test-plan`/`harbor-test-implement` (`Makefile:43-88`), following the spec target's render-then-run shape if the instruction needs agent-specific substitution, or the plan target's simpler shape if it does not.
- `README.md` testing section — mention the new suite alongside the existing three.

**Complexity:** High
**Token estimate:** ~70k
**Agent strategy:** 2-3 parallel agents: one on the container scaffolding and make target, one on the verifier's transcript assertions, one on the negative fixtures. Sequential integration, since the step-order oracle must match the shipped step table exactly.

### Phase 4.1: The published documentation describes the guided add

**File changes**

- `docs:src/pages/projects.mdx:159-194` — rewrite the body of `<Section heading="Registering a repository">` to the content outline in plan.md. The `sub` Fragment at 161-169 loses its "in one command" framing. The paragraph at 171-177 and its example, and the paragraph at 179-186 with its two examples, are retained but reframed as the non-interactive path and moved after the guided description. The idempotency and removal notes at 188-192 are unchanged.
- The worked exchange is a fenced triple-backtick block, per the MDX authoring convention's rule 4; blank lines before and after every slot body, per rule 3; no `<div>`, `<section>` or `class=` anywhere in the page body, per rule 1. If the section needs a shape no existing component provides, add one under `docs:src/components/sections/` rather than inlining markup.
- Section shading: this section currently resolves to `surface={false}` (the prop is unset and `Section.astro:8` defaults to false), following a shaded predecessor at `projects.mdx:110`. If the rewrite splits it into two bands, set each `surface` explicitly to the opposite of its own predecessor; do not copy the existing shaded-then-shaded adjacency between `projects.mdx:196` and `241`.
- `docs:src/pages/projects.mdx:230-237` — the `<ConfigKey name="description, role, tags">` entry explains that these live in the repository's own configuration and that planning reads them. Leave it in place and point the new copy at it as the reason the flow asks for them.
- `docs:src/pages/configuration.mdx:222-225` mentions `repo add` only as the trigger that creates a repository's configuration. Optional touch-up only; do not restructure the configuration reference.
- Voice: match `docs:src/pages/how-it-works.mdx:228-261`, which is the site's existing treatment of a guided flow, including its fenced agent/you dialogue.
- Conventions: no em dashes.
- Verify with `npm run build` and `npx astro check` (0 errors, 0 warnings) in the docs repo, and `grep -nE "<div|<section|class=" src/pages/*.mdx` returning nothing.

**Complexity:** Medium
**Token estimate:** ~35k
**Agent strategy:** Single agent, sequential, working in the docs repo at the root `repo list` reports for it. Prose consistency matters more than throughput here.

### Phase 4.2: The README tells the same story

**File changes**

- `README.md:196` — replace the sentence beginning "`repo add` is pointed at a repo's code: it scaffolds a `.spektacular/` inside it..." with the content example in plan.md. The surrounding sentences about `name` and `location`, the removed `address` key, `repo list`, manual removal, and the no-automatic-fetch behaviour are unchanged.
- `README.md:199-215` — the repo configuration section and the colocated/separate layout paragraph are unchanged; they describe file layout, not how a repository is added.
- Conventions: no em dashes.

**Complexity:** Low
**Token estimate:** ~15k
**Agent strategy:** Single agent, sequential. One paragraph.

## Testing Strategy

Per-phase testing detail, keyed to the phases above.

- **Phase 1.1** — `internal/repo/register_test.go` (new): fresh registration, idempotent no-op on an identical entry, in-place update on a changed entry, footprint status transitions, and the metadata note firing only when description, role and tags are all empty. The load-bearing rule is negative: `cmd/repo_test.go` must not be edited. Its 25 tests, `minimalFootprint` (`cmd/repo_test.go:159-175`) and the schema assertions (`cmd/repo_test.go:688,711`) are the definition of "the direct add is unchanged". An edit needed there means the extraction changed behaviour; stop the phase.
- **Phase 1.2** — `internal/repo/examine_test.go` (new), table-driven over `t.TempDir()` trees, matching the fixture style of `internal/repo/footprint_test.go`. The oracle behind "not merely a restatement of the folder name" is a Go fixture whose directory name deliberately differs from its module name; the test asserts the identity comes from the manifest. Also: an empty directory and a nonexistent directory both yield `Readable: false` with no error, and the language histogram reflects seeded file extensions.
- **Phase 2.1** — `cmd/repo_workflow_test.go` (new), `runRootCmd`-driven in the style of `cmd/spec_test.go`: full forward walk, refused out-of-order `goto` carrying a `next_action`, resume report on a second `repo new`, `--force` discard, and answers surviving a rebuild of the workflow from the same state path. `cmd/cross_kind_test.go` gains the isolation cases: a guided add completing with a spec seeded by `writeInProgressState`, asserted by comparing `state.json`'s bytes before and after; the spec then resuming at its recorded step; and `spec new` not being refused while an add is in progress.
- **Phase 2.2** — `internal/steps/repo/steps_test.go` (new), in the style of `internal/steps/spec/steps_test.go`: the rendered instruction for each proposal step contains terms from a seeded fixture repository's README and manifest; the empty-evidence branch renders when the fixture is bare; `Examine` is invoked once across a full walk.
- **Phase 2.3** — extends `cmd/repo_workflow_test.go`: at `confirm` the target directory has no `.spektacular` and `config.yaml` is byte-identical to its pre-add content; a failing register leaves `repo-state.json`'s `current_step` at `confirm`; a successful add produces a registry entry plus a `repo.yaml` carrying the gathered metadata and no metadata note; a declined placement produces a footprint under `<projectRoot>/repos/<name>/.spektacular` with a `source` pointing at the target, leaving the target with no `.spektacular`.
- **Phase 2.4** — `templates/vocabulary_containment_test.go` (new): a hand-written stable marker asserted present exactly once in every file under `templates/steps/repo/`, directory-walked in the style of `templates/context_directive_test.go:14-36`. `templates/context_directive_test.go:19-25` gains `"steps/repo"` in this phase or the previous one, whichever adds the templates.
- **Phase 2.5** — `templates/skill_resume_test.go` gains `spek-manage-repos` in its fixed list. `internal/agent/instruction_surface_test.go`'s `TestRenderedSkillsAvoidStdinInstructionSurface` re-renders the rewritten skill through the real install path and must pass unedited.
- **Phase 3.1** — `tests/harbor/repo-workflow/tests/test_repo_workflow.py` (new), reusing `iter_transcript_objects`, `extract_tool_calls` and `extract_builtin_file_edits` from `tests/harbor/spec-workflow/tests/test_spec_workflow.py:93-172`, with a hand-maintained step-order oracle matching `internal/steps/repo/steps.go`. Every transcript rule is paired with a synthetic transcript fixture that breaks it, so a rule that can never fail is caught in review. Run by hand via a new `harbor-test-repo` target beside `Makefile:43-88`; not in CI.
- **Phases 4.1 and 4.2** — no automated tests. Verified by `npm run build` and `npx astro check` in the docs repo, `grep -nE "<div|<section|class=" src/pages/*.mdx` returning nothing, and review of the prose against the content outline and content example in plan.md.

**Success metric verification.** Two metrics are behavioural: the typical add supplying exactly one free-form value (Phase 3.1's transcript assertions) and adding a repository during an in-progress specification (Phase 2.1's isolation cases plus a Phase 3.1 transcript check for the absence of a conflict report). Three are **Manual, captured in the implementation test plan**: the acceptance-versus-correction rate for proposed values, the under-a-minute attention cost, and a second or third add completed without consulting documentation. The implement workflow produces the procedure for those three once the code exists.

## Project References

**Repos this plan spans.** Run `go run . repo list` at the start of any session; never assume the working directory holds a repo's code.

- `spektacular` (role: tool) at `/home/nicj/code/github.com/jumppad-labs/spektacular` — carries every requirement except the documentation one.
- `docs` (role: documentation) at `/home/nicj/code/github.com/jumppad-labs/spektacular-website` — carries "The published documentation describes the guided add" (Phase 4.1).

**Requirement to repo and files.** All of the following are `spektacular` unless stated.

| Requirement | Where it is carried out |
|---|---|
| Only the repository itself is asked for cold | `templates/steps/repo/01-locate.md`, `internal/steps/repo/steps.go` |
| Every remaining detail arrives as a proposed value | `internal/repo/examine.go`, `internal/steps/repo/steps.go`, `templates/steps/repo/02-name.md` through `05-tags.md` |
| Questions are asked one at a time, in a fixed order | `internal/steps/repo/steps.go` (step table), `templates/steps/repo/02-name.md` through `05-tags.md` |
| The user can delegate the whole set | `internal/steps/repo/steps.go` (`placement` multi-source `Src`), `templates/steps/repo/02-name.md` |
| Internal vocabulary never reaches the user | every file under `templates/steps/repo/`, `templates/vocabulary_containment_test.go`, `tests/harbor/repo-workflow/` |
| File placement decided silently by default | `internal/steps/repo/probe.go`, `templates/steps/repo/06-placement.md` |
| The user confirms before anything is written | `templates/steps/repo/07-confirm.md`, `internal/steps/repo/steps.go` (`register` as a `before_` callback) |
| A guided add always produces descriptive metadata | `internal/repo/register.go`, `cmd/repo_workflow_test.go` |
| An interrupted add resumes where it stopped | `cmd/repo.go` (`repoStateFilePath`, `runRepoNew`), `cmd/resume.go` reused unchanged, `templates/steps/resume.md` |
| An add can run alongside a spec or plan | `cmd/repo.go` (separate state path and kind), `cmd/cross_kind_test.go` |
| The direct add is unaffected | `cmd/repo.go:113-252` reduced to a shell over `internal/repo/register.go`; `cmd/repo_test.go` unedited |
| The published documentation describes the guided add | `docs:src/pages/projects.mdx:159-194`, plus `README.md:196` |

**Key source references.** `internal/workflow/workflow.go:73,87,113-133,147-156`; `internal/workflow/state.go:14-21,25-27`; `cmd/resume.go:22-46,133-181`; `cmd/spec.go:106-108,121,192-203,348-355`; `cmd/repo.go:38-59,89-97,113-252,308-310,332-358,384-386,400-404`; `internal/config/config.go:170-200`; `internal/config/repo.go:33-40,137-143,288-304`; `internal/repo/footprint.go:13-17,28-85`; `internal/repo/set.go:57-313`; `internal/stepkit/stepkit.go:20-127`; `internal/steps/spec/steps.go:25-39,42-49,53-65,108-112`; `internal/steps/plan/steps.go:88-97`; `internal/agent/skills.go:25-66`; `internal/agent/commands.go:18-24`; `templates/templates.go:6`; `templates/context_directive_test.go:14-36`; `cmd/repo_test.go:41-49,70-94,159-175,185-238`; `tests/harbor/spec-workflow/tests/test_spec_workflow.py:53-58,93-172`; `Makefile:43-88`.

**Knowledge consulted.** `spektacular` conventions: error messages must suggest remediation; passing tests required for done. `docs` conventions: plan content pages, MDX authoring, alternate section background, no em dashes. Architecture notes: `architecture/workflow-steps.md` (note its recorded `StepCallback` signature is stale; the live one is at `internal/workflow/workflow.go:41`), `architecture/working-with-files-from-steps.md`. Gotcha: `gotchas/fsm-cancel-only-works-before-transition-commits.md`.

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|-------------|----------------|
| Low | ~10k | Single agent, sequential |
| Medium | ~25k | 2-3 parallel agents |
| High | ~50k+ | Parallel analysis, sequential integration |

Phase estimates: 1.1 ~25k, 1.2 ~35k, 2.1 ~90k, 2.2 ~45k, 2.3 ~40k, 2.4 ~50k, 2.5 ~35k, 3.1 ~70k, 4.1 ~35k, 4.2 ~15k. Phases 2.1 and 3.1 are the two that warrant parallel analysis with sequential integration; 2.4 must stay with a single agent regardless of size, because ten instruction templates written by several agents read as several voices.

## Migration Notes

No migration. No serialised format changes, no configuration keys are added, removed or renamed, and no existing file is rewritten in place. Two additive changes touch a user's project: a new `repo-state.json` appears beside `state.json` in `.spektacular/` once a guided add is first started, and it is committed rather than ignored, matching how `state.json` is treated (`.spektacular/.gitignore` lists only `*.tmp`, `*.log`, `.env` and `repos/`, and that `repos/` entry is scoped to the clone directory). Projects initialised before this change need `init` re-run to pick up the rewritten skill, which the version-check block at the top of every workflow skill already surfaces (`cmd/version.go:36-146`, comparing against the file written only by `init` at `cmd/init.go:55-58`).

## Performance Considerations

One place performance is a functional requirement rather than a nicety: the examination runs while the user waits between the first question and the second, so it is bounded by construction. It reads a single README up to a fixed length, at most one manifest, one directory listing, and an extension histogram over no more than two directory levels with a hard cap on entries walked, honouring `.spektacular_ignore` so vendored and build directories cannot dominate the result. It reads no file contents for the histogram and never recurses into a source tree. If a repository is large enough that this is still slow, the correct response is to tighten the cap, not to widen what is read; widening it is a change to the specification's boundary.

Nothing else in this plan is on a hot path. The guided flow performs one bounded filesystem probe per add for writability, and its state file is written once per step transition, the same cost the existing workflows already pay.
