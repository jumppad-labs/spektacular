---
created_date: "2026-09-02"
status: completed
closed_date: "2026-09-02"
---

# Plan: 000046_relocatable-repo-footprint

<!-- Metadata -->
<!-- Created: 2026-09-02T11:32:24Z -->
<!-- Commit: cf0f87723dd4b00f23cb34133ab62dad9ac0c133 -->
<!-- Branch: main -->
<!-- Repository: git@github.com:jumppad-labs/spektacular.git -->

## Overview

A repo's Spektacular configuration gains a single optional `source` setting naming where its code lives, either a file location on disk or a git repository to clone, so a team can keep a repo's configuration, knowledge, and changelog in a project folder of their own while the code repository stays untouched. The project registry shrinks to `name` plus `location` (the folder holding the repo's configuration), Spektacular resolves the source once and reports it as the repo's root, tells agents to work there in every workflow, and stops writing a release note into the code. The docs, README, and repo-management skill show the colocated layout and the separate layout with both source kinds. Teams that cannot or prefer not to commit Spektacular files into their code repositories benefit, and existing colocated setups keep working.

## Conventions

- **Error messages must describe the problem and suggest remediation** — the new failure paths (a registry entry still using `address`, an unresolvable source, a `repo add` without `location`) must be built with `output.NewError(...).WithNextAction(...)` naming the exact corrective command or key.
- **Passing tests are required before calling work done** — `go test ./...` must pass after every phase, and the harbor implement suite must be run once its oracles and fixtures change, since it does not run in CI.
- **No em dashes** — applies to every prose surface this plan edits: the docs site pages, the README, the repo-management skill, and the rewritten step templates.
- **MDX authoring conventions** — the docs-site changes must use the `ConfigKey` slot pattern with blank lines around slot content, fenced code blocks for the directory trees and YAML, and no layout HTML in page bodies.
- **Label before filename in file-scoped reference headings** — new or edited headings on the configuration page keep the "Repository configuration: repo.yaml" form and keep the repo.yaml overview, example, and key reference adjacent.
- **Alternate section background shading** — if the projects page gains a new top-level section for the layouts, its `surface` value must be the opposite of the preceding section's.
- **Plans must sketch content structure, not just summarize it** — the docs, README, and skill phases must carry a content outline or content example with the exact `source` and `location` keys, sample values, and the layout trees.

## Architecture & Design Decisions

**Options weighed.** Three shapes were viable for the code location. *Option A, source on the resolved repo*: repo resolution reads the optional `source` from repo.yaml once and hands out a second path, so every code consumer (`repo list` root, git operations, the rendered roster, and the templates that direct the agent into a repo) sees the source, while every consumer of the repo's own files (footprint validation, knowledge aggregation, the repo-routed changelog store, registration write-back) keeps the directory holding `.spektacular/` (`internal/repo/set.go:18-24,161-232`; consumers enumerated in research.md). Pros: one resolver, one definition of "code location", the roster and listing cannot diverge (plan 000042's single-resolver rule), and the ordering problem of reading `source` from a file found via the root is solved in the one place that already parses that file. Cons: `ResolvedRepo` grows a field and eight call sites must be triaged into the two groups. Effort: Medium. *Option B, source resolved at the edges*: leave `internal/repo` untouched and have `cmd/repo.go` and the templates read repo.yaml themselves. Pros: smallest diff. Cons: the same read is duplicated in listing, the roster, and any future consumer, `staleNote`'s `git -C` keeps running in the wrong directory (`set.go:206`), and the listing and roster can disagree. Effort: Low, but fragile. *Option C, root becomes source*: redefine `ResolvedRepo.Root` as the code directory and track the footprint directory separately. Pros: `repo list` needs no change. Cons: every Spektacular-files consumer changes instead of the smaller code-consumer set, `FootprintError.Root` changes meaning, and `cmd/knowledge.go:208`'s colocated-repo tolerance silently flips for a colocated repo that declares a source. Effort: High and riskier. Option A is chosen.

**Where the code's origin is declared.** During the walkthrough the user moved the git origin out of the project registry and into the repo's own configuration: a registry entry now says only which folder holds the repo's Spektacular files (`name` plus `location`, with the old `local` key accepted as an alias), and the repo's `source` says where its code is, as a file location (`file://` or a plain path, absolute, relative to the directory holding repo.yaml, or `${VAR}`-expanded) or a git location (`git://`, `ssh://`, `http(s)://`, or scp-style). A git source is cloned into the project's gitignored working folder `.spektacular/repos/<name>/` on first use, exactly where address-registered repos were cloned before, and reused thereafter; the staleness check compares that clone against the git source. The registry's `address` key is removed outright with no migration, since nothing has shipped with it, and a configuration that still carries it fails to load with an error that names the repo and says to set `source` in its repo.yaml. This beats keeping `address` in the registry because a remote code repo no longer needs any Spektacular file checked in: the project folder holds the repo folder, and the repo folder holds the origin.

**Templates and run-time provision.** The roster entries gain a resolved `source` (empty when the repo is not on disk, so listing and roster stay clone-free), rendered on the existing roster lines in the spec interview and plan discovery and architecture steps, and the implement workflow starts injecting the roster the same way (`wf.SetData("repos", …)` on `implement new` and `goto`), passing it to the steps that touch code: read_plan's drift check, analyze, implement, test, verify, and the feature-changelog step. Each of those steps opens with the roster and an instruction to carry out work in the listed source; the unprefixed-path convention in the changelog steps is redefined so that a path prefix is required whenever more than one repo is registered and an unprefixed path belongs to the only registered repo, removing the "shares this project's working tree" stand-in. The `update_repo_changelog` step and its template are removed from the implement FSM, `update_changelog` advances straight to `test_plan`, and the per-repo derived record written by `update_feature_changelog` carries the user-facing summary the release note used to hold, so exactly one project record and one record per affected repo are written, each at the location its configuration declares. A new template-contract test walks the step and skill templates and bans the stand-in phrases ("directory you are running in", "shares this project's working tree", any root `CHANGELOG.md` instruction) and asserts the code-touching implement steps render the roster with sources, following the banned-phrase pattern already used for section drafting.

**Documentation and end-to-end proof.** In the docs repo the repo.yaml reference on the configuration page gains the `source` key in the established `ConfigKey` shape with both source kinds, the project `repos` key is rewritten around `name` and `location` (noting the `local` alias and the removal of `address`), the env-var note is extended to repo.yaml, and the projects page's "How configuration is split" section gains a walkthrough of the colocated layout and the separate layout with a file source and with a git source; the project-level `source` (a git URL recorded for provenance) is called out as a different setting. In the spektacular repo the README's Configuration section and the repo-management skill show the same layouts and a `repo add` payload carrying `location` and `source`. The harbor implement suite gains a separate-source repo (footprint under the project, code elsewhere) and its hand-maintained step order and path oracles move in the same change, giving the spec's clean-`git status` success metric a behavioural check. Conventions driving these choices: error messages must suggest remediation (the `address` rejection and source failures), passing tests before done, no em dashes and the MDX authoring rules for every prose surface, and plan content pages for the docs phases. Rejected options and their evidence are recorded in `research.md#alternatives-considered-and-rejected`.

## Component Breakdown

**spektacular repo**

- **Repo configuration model** (changed) — owns the `source` setting as an optional field of the repo-level config, serialised only when set, together with the classification of a value as a file location or a git location and the helper that turns a file location (plain or `file://`, relative or `${VAR}`-bearing) into an absolute path anchored at the directory holding the config file. Loading, validation, and writing are otherwise unchanged; the whole-file environment expansion the loader already performs covers the new field.

- **Project registry model** (changed) — owns the registry entry shape: `name`, `location` (folder holding the repo's `.spektacular/`), dependencies, provider. It accepts `local` as an alias for `location` on load, writes `location` on save, and rejects `address` at validation time with an error that says where the git source now belongs.

- **Repo resolution** (changed) — owns the two locations of a resolved repo: the root that holds `.spektacular/` (located by the registry's `location`) and the source that holds the code (declared by the repo config, defaulting to the root). For a file source it resolves the path; for a git source it clones into the project's working folder when absent and reuses the clone otherwise, marking the repo materialized and running the staleness check against the git source. It resolves the source during the same validation pass that already parses the repo config and exposes a git-free reader of the source for callers that must not clone. Footprint creation and repair keep working against the root.

- **Repo commands** (changed) — `repo list` reports the resolved source as each repo's `root` when one is declared and the existing root otherwise, still without side effects. `repo add` accepts `location` (creating the folder when missing) and `source` in its payload, no longer accepts `address`, and writes `source` into the repo's own config through the existing create-then-update path, alongside description, role, tags, and deployment.

- **Project init** (changed) — seeds the project's own registry entry with `location: .` instead of `local: .`; the footprint cascade is unchanged.

- **Repo roster** (changed) — the projection of registered repos that workflow instructions embed gains each repo's resolved source (empty when the repo is not on disk). It reads the source through the same git-free reader listing uses, so the roster and `repo list` never disagree.

- **Implement workflow command** (changed) — injects the roster into workflow data on `implement new` and `implement goto`, mirroring what the plan and spec commands already do, so every implement step can render it.

- **Implement workflow steps** (changed) — the FSM drops the `update_repo_changelog` step; `update_changelog` advances directly to `test_plan`. The steps that touch code (read plan, analyze, implement, test, verify, feature changelog) render the roster with sources and direct the agent to work in each repo's source. The feature-changelog step's per-repo record absorbs the user-facing summary the removed release note carried, and the path-prefix rule for attributing files to repos is restated without reference to the working tree.

- **Plan and spec workflow steps** (changed) — the existing roster blocks in the spec interview and the plan discovery and architecture steps render the source per repo, and the "not just the directory you are running in" and "colocated repo" fallbacks are reworded to point at the sources.

- **Skills** (changed) — the repo-management skill documents `location`, `source` in both kinds, the colocated and separate layouts, and a `repo add` payload carrying them; the update-changelog skill and the plan and implement workflow skills drop references to the root `CHANGELOG.md` step and to the running directory.

- **Template-contract tests** (new) — a test over the embedded step and skill templates that bans the stand-in phrases (running directory, working-tree colocation, root `CHANGELOG.md`) and asserts the code-touching implement steps render the roster with sources. Existing step-order, FSM-walk, and roster tests are updated for the removed step and the new roster key.

- **Harbor implement suite** (changed) — the seeded fixtures use `location` in the registry and gain a repo whose Spektacular files live under the project while its code lives elsewhere via a file source, and the hand-maintained step-order and changelog-path oracles are updated in the same change, giving the workflow an end-to-end check that only the project and per-repo records are written.

- **In-repo documentation** (changed) — the README's configuration section shows `location` in the registry and `source` in the repo configuration with both kinds, and describes both layouts at front-door altitude, linking to the docs site for the full reference.

**docs repo**

- **Configuration reference page** (changed) — the repo.yaml section gains a `source` key in the existing key-reference shape covering file and git locations, the project `repos` key is rewritten around `name` and `location` with the `local` alias and the removal of `address`, the environment-variable note is extended to repo.yaml, and the project-level `source` (a git URL for provenance) is explicitly contrasted with the repo-level one.

- **Projects page** (changed) — the registration example and the "How configuration is split" section are rewritten for `location` and `source`, with a walkthrough of the colocated layout and the separate layout for a file source and for a git source, each with a directory tree and a `repo add` example.

## Data Structures & Interfaces

**Repo configuration (`repo.yaml`)** gains one optional key. Absent means "the code lives at the repo root", which is today's behaviour for every existing file. A value is a file location (plain path or `file://`) or a git location (`git://`, `ssh://`, `http://`, `https://`, or scp-style `user@host:path`).

```yaml
# .spektacular/repo.yaml
description: ...
source: file://${HOME}/code/api   # or ../api, or git@github.com:org/api.git
knowledge: ...
changelog: ...
```

```go
type RepoConfig struct {
    Description, Role, Deployment string
    Tags                          []string
    Source                        string   // yaml:"source,omitempty"
    Knowledge                     KnowledgeConfig
    Changelog                     ChangelogConfig
}

type SourceKind int // SourceNone, SourceFile, SourceGit

// ParseSource classifies Source. For a file source it returns the absolute
// directory, anchoring a relative value at configDir (the directory holding
// repo.yaml) and stripping a file:// prefix; for a git source it returns the
// URL unchanged. SourceNone when Source is unset.
func (c RepoConfig) ParseSource(configDir string) (SourceKind, string, error)
```

**Project registry entry (`config.yaml`)** locates only the folder holding the repo's Spektacular files.

```yaml
repos:
  - name: api
    location: ./repos/api     # folder holding the repo's .spektacular/; "local" still accepted
```

```go
type RepoEntry struct {
    Name         string   `yaml:"name"`
    Location     string   `yaml:"location,omitempty"`
    Local        string   `yaml:"local,omitempty"` // deprecated alias; folded into Location on load, never written
    Dependencies []string
    Provider     string
    Config       GitRepoConfig
}
// validateRepos: Location (after alias folding) is required; a present
// "address" key is rejected with a next_action naming the repo's repo.yaml source.
```

**Resolved repo** carries both locations. `Root` keeps its meaning (the directory holding `.spektacular/`, used by every consumer of the repo's own files); `Source` is new and always an absolute directory: the file source, the clone of a git source, or `Root` when unset.

```go
type ResolvedRepo struct {
    Name         string
    Root         string // holds .spektacular/; knowledge, changelog, repo.yaml live under it
    Source       string // the code: file source, git clone, or Root
    Entry        config.RepoEntry
    Materialized bool   // true when Source is a project-managed clone of a git source
    StaleNote    string
}

// LocalSource mirrors LocalRoot: the named repo's code directory when it is
// on disk (file source present, or git clone present), resolved from its
// repo.yaml without invoking git.
func (s *Set) LocalSource(name string) (string, bool)
```

**Registration payload and listing projection.** `repo add --data` accepts `location` and `source` and no longer accepts `address`; `repo list` drops `address` from its output, keeps `local` only as the raw registry value when the alias was used, and reports the resolved source in `root`.

```json
// repo add --data
{"name":"api","location":"./repos/api","source":"git@github.com:org/api.git","description":"...","role":"...","tags":[],"deployment":"..."}

// repo list (per repo) — root is the resolved source when declared
{"name":"api","location":"./repos/api","root":"/proj/.spektacular/repos/api","materialized":true, ...}
```

**Roster entry** rendered into workflow instructions gains `source` (resolved absolute path, empty string when the repo's code is not on disk), alongside the existing identity fields.

```go
map[string]any{"name":..., "description":..., "role":..., "tags":..., "deployment":..., "source":...}
```

**Implement workflow data** gains the `repos` key carrying the roster, the same shape the plan and spec workflows already persist in `state.json`; the implement step callbacks pass it through the existing roster-extra helper. The implement FSM loses the `update_repo_changelog` state; `update_changelog` transitions to `test_plan`.

No changelog document formats change: the project record and the per-repo derived record keep their existing front matter and provenance; the per-repo record's body gains a short user-facing summary paragraph.

## Implementation Detail

**Two-location resolution, one resolver.** The central shape change is that a resolved repo stops being "one directory" and becomes "a root that holds the repo's Spektacular files and a source that holds its code". The repo package resolves the source in the same pass that validates the footprint, so the ordering problem (the source is declared in a file that is found via the root) is settled in one place and never leaks to callers. Every existing call site is triaged into one of two groups and touches exactly one of the two fields: footprint creation and repair, descriptive metadata, knowledge aggregation, the repo-routed changelog store, and the registration write-back stay on the root; listing, the git staleness query, the roster, and the agent-facing instructions move to the source. A developer reading the resolved-repo type sees the two fields documented side by side with that split, and the git-free `LocalSource` sits beside `LocalRoot` with the same "on disk, never clone" contract, so the no-side-effects guarantee of listing is preserved by construction rather than by care.

**One `source`, two kinds.** The repo config classifies its `source` once, into a file location or a git location, and the resolver branches on the kind: a file location becomes the code directory directly; a git location reuses the clone-into-working-folder path that address-registered repos used before, now keyed by the repo's name and driven by the repo's own configuration rather than the registry. The clone step, the reuse-if-present rule, the never-fetch rule, and the staleness notice all keep their existing code; only their input moves. Scheme-less values are file paths, so a colocated repo can write `source: ..` and a project-hosted repo folder can write `source: ../../code/api` without ceremony.

**Registry hard break with an alias.** The registry entry loses `address` and renames `local` to `location`. The alias is folded at load time so existing project configs keep working and are rewritten with the new key the next time registration saves the file; `address` is rejected during validation with a message that names the repo and the exact `source` line to add to its repo.yaml, since the value is not lost, only relocated. No migration code is written. The project's own config in this repository is updated by hand in the same change.

**Registration follows the descriptive-field pattern.** `repo add` treats `source` exactly like description, role, tags, and deployment: it is parsed from the payload, merged into the loaded repo config only when non-empty, written back only when something changed, and compared by the same equality helper. `location` is required and its folder is created when missing, because that folder is the project's own footprint for the repo; no validation of the source itself is added.

**Run-time provision reuses the roster pipeline.** The roster projection gains one key, the implement command starts setting the roster into workflow data the way the plan and spec commands already do, and the implement step callbacks that touch code pass the existing roster-extra helper. The templates render the source on the same roster line the plan and spec steps already use, so the agent sees one consistent "name, description, role, tags, deployment, source" block in every workflow. This reverses an earlier decision to keep resolved paths out of rendered instructions; the trade-off is accepted because the roster is refreshed from config on every `new` and `goto`, so the value is at most one invocation old, and the instructions still direct the agent to `repo list` when a repo is missing from the roster because it was not on disk at render time.

**Removing a workflow step.** Deleting `update_repo_changelog` is a straight FSM edit: one fewer state, one retargeted transition, one deleted template and callback, and the goto branch in the preceding step rewritten. The tests that enumerate the step order, walk the FSM, and pin the deleted template's phrases are updated rather than worked around. The per-repo derived record written by the feature-changelog step gains a leading user-facing summary so the information the release note carried is not lost, without introducing any new file.

**Prose behaviour is enforced by template-contract tests.** The spec's requirement that no template use the running directory as a stand-in is expressed as a banned-phrase test in the established shape: a hand-maintained list of forbidden substrings walked over the embedded step and skill templates, plus positive assertions that the code-touching implement steps render a roster carrying sources when the workflow data has one and a sensible fallback when it does not.

**Documentation shows the layouts.** The docs site, README, and repo-management skill each present the same pictures: a colocated repo whose `.spektacular/` sits inside the code, and a separate layout where a project folder holds one folder per repo with its own repo.yaml, knowledge, and changelog, pointing at code on disk through a file source or at a git repository through a git source. The project-level `source` (a git URL) is named once as a different setting so readers do not conflate them.

**End-to-end proof in harbor.** The implement suite's fixtures move one repo into the separate layout with a file source and switch the registry to `location`, and the hand-maintained oracles (step order, three changelog paths, no built-in-tool writes) are updated in the same change, following the testing-architecture rule that oracle couplings move with the surfaces they mirror.

## Dependencies

- **`internal/config`** — provides `RepoConfig`, `RepoEntry`, the loaders with whole-file `${VAR}` expansion, registry validation, and yaml serialisation; changed to add `Source` with its kind classification, rename `Local` to `Location` with an alias, and reject `address`. No change to the migration detector.
- **`internal/repo`** — provides registry resolution, footprint repair, the git runner, and the roster; changed to carry and resolve the source, clone git sources, and query git against the source. The `GitRunner` interface is unchanged.
- **`internal/project`** — provides init; changed only to seed `location: .`.
- **`cmd` (repo, implement, init, storefile, knowledge)** — provides the `repo add`/`repo list` commands, the implement workflow driver, and the repo-routed changelog store; `repo` and `implement` change, `storefile` and `knowledge` must stay on the root and are touched only to confirm that.
- **`internal/stepkit` and `internal/workflow`** — provide the mustache render pipeline and the `Extra`/`Data` plumbing the roster rides on; no change, reused as-is.
- **`internal/steps/implement`, `internal/steps/plan`, `internal/steps/spec`** — own the FSM tables and template callbacks; implement changes (state removed, roster extra added), plan and spec change only in templates.
- **`templates` (embedded FS)** — the step and skill templates and their contract tests; changed throughout as listed in the phases.
- **`gopkg.in/yaml.v3`** (`omitempty`, alias folding after unmarshal) and **`github.com/cbroglie/mustache`** (section iteration over `[]any`) — existing libraries; no version change.
- **Harbor E2E harness** (`harbor` CLI, Docker, Claude credentials) — required to run the implement suite after its fixtures and oracles change; not in CI, run manually by the implementer.
- **docs repo (spektacular-website)** — Astro 5 + MDX + `astro-expressive-code`; `npm run build` and `npx astro check` are the verification commands. No new dependency.
- **Spec 000046_relocatable-repo-footprint** — the upstream spec, updated during the plan walkthrough to the registry and source shape this plan implements.
- **Prior plans 000039, 000042, 000044, 000045** — already landed; this plan builds on their resolved-repo type, registration write-back, projects page, and one-shot migration detector, reverses 000039's "no resolved paths in rendered instructions" call, and moves 000039's registry `address` into the repo config.
- **This repository's own `.spektacular/config.yaml`** — registers the docs repo with `address` and `local`; it must be updated by hand (drop `address`, rename to `location`) in the same change or the CLI will refuse to load it.
- **No dependency must land before this plan starts.** The spektacular repo work is self-contained; the docs repo work depends only on the final key names and semantics, which this plan fixes.

## Testing Approach

**Layers.** The work is covered by the project's three existing layers. Go unit tests carry the mechanics: config round-trips, source classification and resolution in each accepted form, the registry alias and the `address` rejection, the two-location resolved repo including a git source cloned into the working folder, `repo add` writing and preserving `source`, `repo list` reporting the source as root, the roster carrying `source`, and the implement FSM without the removed state. Template-contract tests carry the prose behaviour: a hand-maintained banned-phrase list walked over the embedded step and skill templates, and positive render assertions that the code-touching implement steps embed the roster with sources. The harbor implement suite carries the end-to-end proof with a separate-source repo in its fixtures. The docs repo is verified by its build and type check plus the MDX guard greps, not by content tests.

**Where coverage concentrates.** Repo resolution, the registry model, and the repo commands get the most attention because they own the root-versus-source split and the registry break, and a wrong triage there silently writes Spektacular files into the code, runs git in the wrong directory, or locks a user out of a config that used to load. Every consumer that must stay on the root keeps a regression test proving it still reads the footprint from the root when a source is set: footprint repair, descriptive metadata, knowledge aggregation, and the repo-routed changelog store, including the colocated repo that declares a source pointing elsewhere.

**Load-bearing assertions.** In plain language the tests guarantee that: a repo.yaml without `source` behaves exactly as before and a default repo.yaml still serialises only knowledge and changelog; an absolute, a relative, a `${VAR}`, and a `file://` source all resolve to the same absolute path and `repo list` reports it as root; a git source is cloned into the working folder on first use, reused afterwards, reported as root, and checked for staleness against its URL; a registry entry with `local` loads the same as one with `location`, and one with `address` fails with an error naming the repo and the fix; a repo without a source reports the same root as today; registration writes `source` and re-registration with an empty payload never blanks it; the roster and `repo list` agree on the source; the implement workflow's step order no longer contains the root-changelog step and `update_changelog` advances to `test_plan`; no step or skill template tells the agent to use the running directory, the shared working tree, or a root `CHANGELOG.md`; and the code-touching implement steps render each repo's source when the workflow data has a roster and fall back sensibly when it does not.

**Conventions followed.** New Go tests use `t.TempDir` and the existing helpers (fake git runner, footprint writers, `renderStepWithData`), one exported test per acceptance criterion with a doc comment naming it, `require` assertions, and hand-maintained expected values rather than values derived from the code under test. Template tests use the embedded FS walk with a floor so an empty walk cannot pass. Harbor oracles are updated by hand in the same change, per the testing-architecture rule.

**Success metrics.**

- *A user can set up a project folder that describes an external code repository, run a plan and an implement workflow end to end, and finish with a clean `git status` in the code repository apart from the intended code changes.* **Behavioural test** at the harbor layer: the implement suite's fixtures gain a repo whose Spektacular files live under the project while its code lives elsewhere via a file source, and the verifier asserts the per-repo record lands under the project-side footprint, the project record lands at the project's location, no built-in file tool wrote a changelog, and the code directory contains no `.spektacular/` or `CHANGELOG.md`. The git-source variant is covered at the unit layer with a fake git runner. The plan-workflow half of the metric is covered by the unit-level roster and listing assertions; the harbor plan suite is not extended.
- *An agent following the repo-management skill registers a repo with a separate source correctly on the first attempt, without the user explaining the colocated versus separate layouts.* **Manual, captured in the implementation test plan.** The skill's content is pinned by unit tests (it must show `repo add` payloads with `location` and both source kinds and the layouts), but first-attempt success by a live agent is observed, not asserted.

**Deliberate gaps.** No Go-level guard is added to the `finished` step to detect a stray `CHANGELOG.md`; the workflow config carries no repo set and the template-contract and harbor layers already cover the behaviour. No test validates that a file source exists or is a git working tree, because the spec does not require validation. No real-network clone test is added; the existing git integration test already covers the exec-backed runner. The harbor plan and spec suites are not modified; their fixtures register no repos and their oracles do not mention the affected wording.

## Milestones & Phases

### Milestone 1: Repos declare where their code lives, and the registry only says where the repo folder is

**What changes**: A repo's configuration file gains an optional `source` setting naming where its code lives: a file location (absolute, relative to the configuration file's directory, or using `${VAR}`, with or without `file://`) or a git location that Spektacular clones into its working folder on first use. The project registry entry shrinks to `name` and `location`, the folder holding the repo's configuration; the old `local` key keeps working and the old `address` key is refused with a message pointing at `source`. Registering a repo can set `location` and `source`, and the repo listing reports the resolved source as the repo's root, while knowledge and changelog keep landing where the repo's configuration says. Repos without the setting behave exactly as before.

#### - [x] Phase 1.1: Add `source` to the repo configuration
**Repo:** spektacular

The repo configuration file gains an optional `source` key with a classifier that tells a file location from a git location and a helper that turns a file location into an absolute path anchored at the directory holding the configuration file. Nothing else about loading, validating, or writing the file changes, so every existing configuration keeps working and a freshly written default file still contains only its knowledge and changelog sections.

*Technical detail:* [context.md#phase-11](./context.md#phase-11-add-source-to-the-repo-configuration)

**Acceptance criteria**:
- [x] A repo configuration with no `source` loads exactly as before, and a default configuration written to disk contains only the knowledge and changelog keys.
- [x] An absolute path, a path relative to the configuration file's directory, a `${VAR}` path, and the `file://` form of each all classify as a file source and resolve to the same absolute directory.
- [x] `git://`, `ssh://`, `https://`, and scp-style values classify as a git source and are returned unchanged.
- [x] A configuration that sets `source` round-trips through write and read without losing it.

#### - [x] Phase 1.2: Replace the registry's `local` and `address` with `location`
**Repo:** spektacular

The project registry entry is renamed around what it actually locates: the folder holding the repo's Spektacular files. `location` becomes the key, `local` is accepted as an alias on load and never written back, and `address` is refused at load time with an error that names the repo and tells the user to set `source` in that repo's configuration. Project init seeds the project's own entry with the new key, and this repository's own project configuration is updated by hand.

*Technical detail:* [context.md#phase-12](./context.md#phase-12-replace-the-registrys-local-and-address-with-location)

**Acceptance criteria**:
- [x] A registry entry with `location` and one with `local` resolve to the same folder, and saving the configuration writes `location` only.
- [x] A registry entry carrying `address` fails configuration loading with an error naming the repo and the `source` line to add to its repo configuration.
- [x] A registry entry with neither `location` nor `local` fails validation with an error naming the repo.
- [x] A freshly initialised project registers itself with `location: .`, and this repository's own configuration loads with the new keys.

#### - [x] Phase 1.3: Resolve the source once during repo resolution
**Repo:** spektacular

A resolved repo now carries two locations: the root that holds its Spektacular files, found by the registry as today, and the source that holds its code. A file source resolves to its directory; a git source is cloned into the project's working folder when absent and reused otherwise, with the staleness check run against it; an unset source falls back to the root. The source is resolved in the same pass that already validates the configuration, and a git-free reader exposes it for listing and the roster. Footprint creation and repair keep working against the root.

*Technical detail:* [context.md#phase-13](./context.md#phase-13-resolve-the-source-once-during-repo-resolution)

**Acceptance criteria**:
- [x] A repo whose configuration declares a file source resolves with its root unchanged and its source set to that directory; one without a source resolves with source equal to root.
- [x] A repo whose configuration declares a git source is cloned into the project's working folder on first resolution, reused on the next, reported as materialized, and given a staleness notice when the clone is behind its remote.
- [x] Footprint validation and repair still look for the configuration under the root, even when the source points elsewhere.
- [x] Reading a repo's source for listing never clones or invokes git, and reports absent for a git source not yet cloned.

#### - [x] Phase 1.4: Registration writes `source` and listing reports it as root
**Repo:** spektacular

Registering a repo now takes `location` (creating the folder when missing) and `source`, writes `source` into the repo's own configuration the same way its description, role, tags, and deployment are written, and no longer accepts `address`. Re-registering without `source` leaves the stored value alone. The repo listing reports the resolved source as the repo's `root` when one is declared and the existing root otherwise, and drops `address` from its output. Every consumer of the repo's own files (knowledge aggregation, the per-repo changelog store, project init's footprint cascade) is confirmed by test to stay on the root.

*Technical detail:* [context.md#phase-14](./context.md#phase-14-registration-writes-source-and-listing-reports-it-as-root)

**Acceptance criteria**:
- [x] `repo add` with `location` and `source` creates the folder if needed and writes `source` into the repo's configuration; a later `repo add` for the same repo without `source` keeps the stored value.
- [x] `repo add` without `location` fails with an error that shows the expected payload, and a payload with `address` fails with an error pointing at `source`.
- [x] `repo list` reports the resolved source as `root` for a repo with a source set, and the same root as before for a repo without one, still without cloning.
- [x] Writing a knowledge entry and a per-repo changelog record for a repo whose source is elsewhere lands both under the folder holding its configuration, and the source directory is untouched.
- [x] A colocated project repo that declares a source pointing elsewhere still has its missing-configuration tolerance and footprint cascade applied against the project directory.

### Milestone 2: Only the project and per-repo changelogs are written

**What changes**: The implement workflow no longer writes a release note to a `CHANGELOG.md` at the root of each changed repo. A feature finishes with exactly one project changelog record and one record per affected repo, each at the location its configuration declares, and the per-repo record now opens with the short user-facing summary the release note used to carry.

#### - [x] Phase 2.1: Remove the root changelog step from the implement workflow
**Repo:** spektacular

The implement workflow drops the step that prepended a release note to a `CHANGELOG.md` at the root of each changed repo. The changelog step now advances straight to the test-plan step, the template and callback are deleted, and every test, skill, and harbor oracle that enumerated the old step is updated in the same change.

*Technical detail:* [context.md#phase-21](./context.md#phase-21-remove-the-root-changelog-step-from-the-implement-workflow)

**Acceptance criteria**:
- [x] The implement workflow's step list no longer contains a root-changelog step, and the step after the changelog step is the test-plan step.
- [x] Driving the implement workflow end to end never instructs the agent to create or modify a `CHANGELOG.md`.
- [x] The finished step's summary and the update-changelog skill no longer mention a repo-level `CHANGELOG.md`.

#### - [x] Phase 2.2: Fold the release note into the per-repo changelog record
**Repo:** spektacular

The per-repo record written by the feature-changelog step now opens with the short, user-facing summary the removed release note used to carry, so a reader scanning a repo's changelog still sees what shipped without reading the plan. The project record is unchanged, and the finished step reports the locations of the project record and each per-repo record.

*Technical detail:* [context.md#phase-22](./context.md#phase-22-fold-the-release-note-into-the-per-repo-changelog-record)

**Acceptance criteria**:
- [x] The feature-changelog step instructs the agent to open each per-repo record with a two-to-four sentence user-facing summary written for someone who has never seen the plan.
- [x] The finished step reports where the project record and each per-repo record were written.
- [x] Running the implement workflow produces exactly one project record and one record per affected repo, each at the location its configuration declares.

### Milestone 3: Agents are told each repo's source at run time

**What changes**: Every workflow that sends an agent into a repo's code now embeds each registered repo's resolved source in its instructions, in the plan and spec steps that already list the repos and, for the first time, in the implement steps that read, change, test, and verify code. No instruction refers to the directory the agent is running in or the project's working tree as a stand-in for a repo, and a test guards that wording from returning. Running an implement workflow against a project whose repo keeps its Spektacular files separately from its code leaves the code repository with nothing but the intended code changes.

#### - [x] Phase 3.1: Carry each repo's source in the roster and give it to the implement workflow
**Repo:** spektacular

The roster of registered repos that workflow instructions embed gains each repo's resolved source, read through the same git-free path the listing uses. The implement workflow starts injecting the roster into its state on start and on every advance, as the plan and spec workflows already do, and the steps that read, analyze, change, test, verify code, and write the feature changelog receive it for rendering.

*Technical detail:* [context.md#phase-31](./context.md#phase-31-carry-each-repos-source-in-the-roster-and-give-it-to-the-implement-workflow)

**Acceptance criteria**:
- [x] The roster entry for a repo whose code is on disk carries its resolved source; for a repo whose code is not on disk the source is empty.
- [x] Starting or advancing an implement workflow stores the roster in its state without cloning any repo.
- [x] Rendering the read-plan, analyze, implement, test, verify, and feature-changelog steps with a roster in the workflow data embeds each repo's name and source; rendering without one produces a sensible fallback.

#### - [x] Phase 3.2: Reword templates to direct the agent to each repo's source
**Repo:** spektacular

Every step and skill that sends the agent into a repo's code now tells it to use that repo's source, listed in the roster block, and nothing refers to the directory the agent is running in or the project's working tree as a stand-in. The roster lines in the spec interview and the plan discovery and architecture steps show the source; the implement steps that touch code open with the roster and a direction to work in each listed source; the path-prefix convention in the changelog steps is restated so a prefix is required whenever more than one repo is registered and an unprefixed path belongs to the only registered repo. A template-contract test bans the stand-in phrases and asserts the roster renders with sources.

*Technical detail:* [context.md#phase-32](./context.md#phase-32-reword-templates-to-direct-the-agent-to-each-repos-source)

**Acceptance criteria**:
- [x] No rendered step or skill template contains "directory you are running in", "shares this project's working tree", or an instruction to write a root `CHANGELOG.md`, and a test fails if any returns.
- [x] The spec interview and the plan discovery and architecture steps render each repo's source on its roster line.
- [x] The implement steps that touch code render the roster with sources and direct the agent to carry out each repo's work in that repo's source.
- [x] The changelog steps define unprefixed paths as belonging to the only registered repo, and require prefixes when more than one repo is registered.

#### - [x] Phase 3.3: Prove the separate layout end to end in the harbor implement suite
**Repo:** spektacular

The harbor implement suite's fixtures switch the registry to `location`, move one repo into the separate layout with its configuration, knowledge, and changelog under the project folder and its code elsewhere via a file source, and the hand-maintained step-order and path oracles are updated to match. A passing run shows the project record and both per-repo records at their declared locations and the code directory free of Spektacular files, which is the spec's clean-`git status` success metric.

*Technical detail:* [context.md#phase-33](./context.md#phase-33-prove-the-separate-layout-end-to-end-in-the-harbor-implement-suite)

**Acceptance criteria**:
- [x] The seeded project registers its repos with `location` and one repo's configuration declares a file source outside the project.
- [x] The suite's expected step order matches the workflow without the root-changelog step, and its expected changelog paths point at the project-side footprint for the separate repo.
- [x] A full harbor run passes, and afterwards the separate repo's source directory contains no `.spektacular/` directory and no `CHANGELOG.md`.

### Milestone 4: Documentation covers `source`, `location`, and the layouts

**What changes**: The documentation site, the README, and the repo-management skill describe the `source` setting in both kinds and the registry's `location` key, show a repo colocated with its Spektacular files and a repo kept separately from them with a file source and with a git source, and explain how the project-level `source` (a git address recorded for provenance) differs from the repo-level one. An agent following the skill can register a separate-source repo without the user explaining the layouts.

#### - [x] Phase 4.1: Document `source`, `location`, and the layouts on the documentation site
**Repo:** docs

The configuration reference gains the `source` key in the repo configuration section covering file and git locations, rewrites the project `repos` key around `name` and `location` (with the `local` alias and the removal of `address`), extends the environment-variable note to repo configuration, and contrasts the project-level `source` git address. The projects page's registration example and configuration-split section are rewritten for the new keys, with a walkthrough of the colocated layout and the separate layout for a file source and for a git source, each with a directory tree and a registration example.

*Technical detail:* [context.md#phase-41](./context.md#phase-41-document-source-location-and-the-layouts-on-the-documentation-site)

**Content outline** (configuration page, project configuration section, `repos` key):
```mdx
<ConfigKey name="repos" type="list" defaultValue="none">

  The project's member repositories. Each entry locates the folder that
  holds the repository's own `.spektacular/`:

  - `repos[].name`: slug-safe unique name; plans and changelog entries
    refer to repositories by it.
  - `repos[].location`: the folder holding the repository's `repo.yaml`,
    absolute or relative to the project root. The older `local` key is
    still accepted and means the same thing.
  - `repos[].dependencies`, `repos[].provider`: unchanged.

  Where the repository's *code* lives is the repository's own concern,
  declared by `source` in its `repo.yaml` (below). The older `address`
  key is no longer accepted here; move its value to that `source`.

</ConfigKey>
```

**Content outline** (configuration page, repository configuration section):
- YAML example gains `source: file://${HOME}/code/api   # or git@github.com:org/api.git`.
- New `ConfigKey` after `deployment`:
  ```mdx
  <ConfigKey name="source" type="string" defaultValue="the directory holding <code>.spektacular/</code>">

    Where this repository's code lives. A file location (a plain path or
    `file://` path, absolute, relative to the directory holding
    `repo.yaml`, or using `${VAR}`) or a git location (`git://`, `ssh://`,
    `https://`, or `user@host:path`). A git location is cloned into the
    project's `.spektacular/repos/<name>/` folder on first use and never
    fetched or pulled afterwards. When set, the repo listing reports it as
    the repository's root and workflows send agents there; knowledge and
    changelog entries still land where this file's `knowledge` and
    `changelog` keys say. Not to be confused with the project-level
    `source`, which records a git address for provenance only.

  </ConfigKey>
  ```
- Env-var paragraph: "…when `config.yaml` or `repo.yaml` is read."

**Content outline** (projects page):
- "Registering a repository": example becomes
  ```bash
  spektacular repo add --data '{"name":"docs","location":"./repos/docs","source":"git@example.com:org/docs.git"}'
  ```
  with the sentence "give it a `location` for its Spektacular files and a `source` for its code".
- "How configuration is split": sub-paragraph gains one sentence (a repository's Spektacular files can sit inside its code or in a folder of their own that points at the code); the `repos` ConfigKey lists `repos[].name` and `repos[].location`; a new `ConfigKey name="source"` mirrors the configuration page and links to `/configuration/`.
- New `Section heading="Colocated or separate"` (opposite `surface` to the preceding section) with three fenced trees:
  ```
  api/                          # colocated: Spektacular files inside the code
  ├── .spektacular/
  │   ├── repo.yaml             # no source needed
  │   ├── knowledge/
  │   └── changelog/
  └── src/
  ```
  ```
  my-project/                   # separate, file source: code checked out elsewhere
  ├── .spektacular/config.yaml  # repos: [{name: api, location: ./repos/api}]
  └── repos/api/.spektacular/
      ├── repo.yaml             # source: file://${HOME}/code/api
      ├── knowledge/
      └── changelog/
  ```
  ```
  my-project/                   # separate, git source: Spektacular clones the code
  ├── .spektacular/
  │   ├── config.yaml           # repos: [{name: api, location: ./repos/api}]
  │   └── repos/api/            # the clone, gitignored
  └── repos/api/.spektacular/
      ├── repo.yaml             # source: git@github.com:org/api.git
      ├── knowledge/
      └── changelog/
  ```
  followed by the two registration commands and one sentence: the code repository receives only code changes.
- "How a registered repository becomes available": "registered by remote address" becomes "whose `source` is a git location".

**Acceptance criteria**:
- [x] The configuration page documents `repos[].location` with the `local` alias and the removal of `address`, documents `source` in the repo configuration key reference and example with both kinds, distinguishes it from the project-level `source`, and notes environment-variable expansion for repo configuration.
- [x] The projects page shows a colocated layout, a separate layout with a file source, and a separate layout with a git source, each with a directory tree, and registration commands that set `location` and `source`.
- [x] The site builds and type-checks, and the MDX guard finds no layout markup in page bodies.

#### - [x] Phase 4.2: Update the README and the repo-management skill
**Repo:** spektacular

The README's configuration section shows `location` in the registry example and `source` in the repo configuration example, states that a repo's Spektacular files can be colocated with its code or kept in a folder that points at the code on disk or at a git repository, and links to the documentation site for the walkthrough. The repo-management skill explains the layouts, shows registration payloads carrying `location` and each source kind, and states what `location` and `source` each locate, so an agent can register a separate-source repo on the first attempt.

*Technical detail:* [context.md#phase-42](./context.md#phase-42-update-the-readme-and-the-repo-management-skill)

**Content example** (README, project configuration `repos` block):
```yaml
repos:
  - name: my-project                # the colocated repo
    location: .
  - name: lib                       # a repo folder in this project; its code is cloned from a git source
    location: ./repos/lib
```
Followed by: "Each repo entry needs a slug-safe unique `name` and a `location`: the folder holding that repo's `.spektacular/` (`local` is still accepted). Where the code lives is declared in the repo's own `repo.yaml` as `source`; the old `address` key is no longer read."

**Content example** (README, repo configuration block):
```yaml
description: the documentation repo
role: documentation
tags: [docs]
deployment: static-site
source: git@example.com:org/docs.git   # optional; a path (file://) or a git URL; default: this folder
knowledge: ...
```
Followed by: "A repo's Spektacular files can sit inside its code (the default) or in a folder of their own, for example one folder per repo under a project, with `source` pointing at a checkout on disk or at a git repository that Spektacular clones into `.spektacular/repos/<name>/`. See spektacular.dev/projects/ for the layouts."

**Content example** (skill, Concepts and Registering sections):
- Concepts: "A repo has two locations. The registry's `location` (the older `local` still works) names the folder holding its `.spektacular/`: `repo.yaml`, knowledge, changelog. The repo's `source`, in its `repo.yaml`, names its code: a file location (plain or `file://` path, absolute, relative to that folder, or `${VAR}`) or a git location (`git://`, `ssh://`, `https://`, `user@host:path`) that is cloned into the project's `.spektacular/repos/<name>/` on first use. With no `source`, the code is the same folder (colocated). The registry no longer takes `address`."
- Registering gains two payloads:
  ```
  repo add --data '{"name":"api","location":"./repos/api","source":"file://${HOME}/code/api","description":"the API service","role":"backend","tags":["go"],"deployment":"kubernetes"}'
  repo add --data '{"name":"docs","location":"./repos/docs","source":"git@example.com:org/docs.git","description":"the documentation repo","role":"documentation","tags":["docs"],"deployment":"static-site"}'
  ```
  and bullets: "`location` is required and is created if missing"; "expect the first add of a git-source repo to take as long as a clone"; "`repo list` reports the resolved source as `root`".
- Materialization: "a repo whose `source` is a git location is cloned into `.spektacular/repos/<name>/`…" replacing "a repo with only an `address`".

**Acceptance criteria**:
- [x] The README's registry and repo configuration examples show `location` and `source`, and the surrounding prose describes the layouts and what `location` and `source` each locate.
- [x] The repo-management skill shows `repo add` payloads with `location` and each source kind, explains the colocated and separate layouts, and still documents `repo add`, `repo list`, and that Spektacular never fetches or pulls.
- [x] No em dashes appear in the changed prose.

## Open Questions

- **Does a live agent attribute changed files correctly under the restated prefix rule once the roster is embedded?** Depends on how the agent reads the reworded changelog steps in a two-repo project, which only the harbor implement run (Phase 3.3) exercises. If the run writes a per-repo record for the wrong repo or misses one, tighten the prefix wording in `07-update_changelog.md` and `10-update_feature_changelog.md`, re-run the suite, and record the wording change in the plan's changelog; STOP and ask the user only if two runs disagree.

Everything else was resolved during planning: the key names, the alias, the hard break, and the source kinds come from the spec as updated at the walkthrough; `${VAR}` expansion and `omitempty` behaviour were verified in the loader and its tests; the mustache renderer already treats an empty roster value as absent (the existing description and role sections rely on it); and no migration is needed because an absent `source` loads as its zero value and `address` is a deliberate hard break.

## Out of Scope

- **Moving a repo's descriptive metadata, knowledge, or changelog into the project configuration.** Considered and rejected in the spec's Non-Goals; the repo keeps owning its own files.
- **Cloning a git source anywhere other than Spektacular's working folder** (`.spektacular/repos/<name>/`). The clone location is not user-configurable; a user who wants the code elsewhere clones it themselves and sets a file source.
- **Migrating registry entries that use `address`.** Hard break, no migration; the CLI's error says where the value now goes. Nothing shipped with the key.
- **Removing the `local` alias.** It is accepted indefinitely in this plan; dropping it is a later decision.
- **More than one source per repo, or a file source that is not a git working tree.** Not supported and not validated; a wrong path surfaces when git or the agent touches it.
- **Migrating existing colocated repos to a separate layout.** Users move the files themselves and set `source`; the one-shot config migration detector is not extended to suggest or populate `source`.
- **Validating a file source at registration** (existence, git status). Dropped with the spec's other "nothing requirements"; a follow-up spec can add it if agents misregister paths in practice.
- **A Go-level guard in the finished step that no root `CHANGELOG.md` was written.** Enforced by the template-contract test and the harbor suite instead; adding it would need repo plumbing in the workflow config that plan 000039 deliberately kept out.
- **Rendering run-time data into library skills.** Library skills are served raw; they keep prose that points at `repo list` and the roster. Templating them is a separate capability change.
- **Extending the harbor plan and spec suites.** Their fixtures register no repos and their oracles are unaffected; only the implement suite gains the separate-source fixture.
- **Renumbering the implement step templates after deleting step 08.** Optional for the implementer; not required by this plan.
- **Broader `$VAR` or `~` expansion in configuration files.** Only the existing `${VAR}` form is supported, for every field, as today.
- **Fixing unrelated documentation drift** beyond the two items touched in passing (the "eight top-level sections" count on the configuration page and the README structure tree missing `repo.yaml` and `changelog/`).

## Changelog

### 2026-09-02 — Phase 1.1: Add `source` to the repo configuration

**What was done**: `RepoConfig` gained an optional `source` key (serialised only when set) together with a `SourceKind` classifier and a `ParseSource(configDir)` method that tells a file location from a git location and anchors relative file paths at the directory holding `repo.yaml`. The project-level `Config.Source` doc comment now states that it is a git address for provenance, distinct from the repo-level code location.

**Deviations**: None.

**Files changed**:
- `internal/config/repo.go`
- `internal/config/repo_test.go`
- `internal/config/config.go`

**Discoveries**: `cmd/root.go`'s `toErrorResponse` recovers a structured `*output.ErrorResponse` with a direct type assertion rather than `errors.As`, so a remediation error built inside `internal/config` must be returned unwrapped (never through `fmt.Errorf("%w")`) or its `next_action` is lost. `RepoEntry.WithDefaults()` is applied only when a repo set is built, not at config load, so Phase 1.2 must fold the `local` alias in the parser before validation as well.

### 2026-09-02 — Phase 1.2: Replace the registry's `local` and `address` with `location`

**What was done**: The registry entry now carries `name` and `location` (the folder holding the repo's `.spektacular/`); `local` is folded into `location` at parse time and by `WithDefaults` and is never written back; a leftover `address` key fails loading with a `config_invalid` error whose next action names the exact `source:` line to add to the repo's `repo.yaml`; a missing location fails validation naming the repo. Project init and knowledge aggregation seed `location: .`, the repo command's payload, listing projection, schemas, and equality moved to `location`, and this repository's own `.spektacular/config.yaml` was updated by hand.

**Deviations**: The Phase 1.3 source resolution (`ResolvedRepo.Source`, `Set.LocalSource`, git-source cloning) was implemented in the same code pass, because the address-driven clone tests could only be rewritten meaningfully once a git `source` could clone; Phase 1.3 is ticked in its own loop with its own criterion tests. `cmd/root.go`'s `toErrorResponse` now recovers a wrapped `*output.ErrorResponse` with `errors.As` so the validation error's next action survives the loader's wrapping. `repo list` no longer echoes `local` at all (the alias is folded before listing sees it), a small departure from the plan's "keeps `local` when the alias was used" wording.

**Files changed**:
- `internal/config/config.go`
- `internal/config/config_test.go`
- `internal/repo/set.go`
- `internal/repo/git.go`
- `internal/repo/set_test.go`
- `internal/repo/git_integration_test.go`
- `internal/repo/roster_test.go`
- `internal/repo/footprint_test.go`
- `internal/project/init.go`
- `internal/project/init_test.go`
- `cmd/root.go`
- `cmd/repo.go`
- `cmd/repo_test.go`
- `cmd/knowledge.go`
- `cmd/knowledge_test.go`
- `cmd/init_test.go`
- `cmd/changelog_file_test.go`
- `.spektacular/config.yaml`

**Discoveries**: `RepoEntry.WithDefaults()` runs only when a repo set is built, never at config load, so alias folding had to live in the parser as well or validation would never see `location`. A plain local path is a file source, so the git integration tests serve their fixture repository over git's dumb HTTP protocol (`httptest` file server plus `git update-server-info`) to exercise a real git source without a network.

### 2026-09-02 — Phase 1.3: Resolve the source once during repo resolution

**What was done**: A resolved repo now carries `Root` (the registered location holding `.spektacular/`) and `Source` (the code directory). Resolution requires the location to exist, loads the footprint from the root, then parses `source`: a file source becomes `Source` directly; a git source is cloned into `.spektacular/repos/<name>/` only when absent, marked materialized, and checked for staleness against its URL; no source means `Source == Root`. `Set.LocalSource` mirrors `LocalRoot` for the code directory without invoking git, reporting absent for an uncloned git source. Footprint creation and repair stay anchored at the root.

**Deviations**: The code was written during Phase 1.2's implement step (see that entry); this loop added no code and confirmed the criterion tests. `checkFootprint` became `loadFootprint`, returning the parsed config so `source` is read in the same pass.

**Files changed**:
- `internal/repo/set.go`
- `internal/repo/git.go`
- `internal/repo/set_test.go`
- `internal/repo/git_integration_test.go`
- `internal/repo/footprint_test.go`

**Discoveries**: A relative file source anchors at `<root>/.spektacular` (the directory holding `repo.yaml`), so a colocated repo pointing at a sibling checkout writes `source: ../../sibling`, not `../sibling`; the docs phases must state this anchor explicitly.

### 2026-09-02 — Phase 1.4: Registration writes `source` and listing reports it as root

**What was done**: `repo add` now requires `location` (creating the folder when missing), accepts `source`, writes it into the repo's own configuration the way description, role, tags, and deployment are written (and only when something changed), rejects a raw `address` key with a corrected payload, and ensures the footprint and writes `repo.yaml` before resolving so a git source given in the same call is cloned on registration. `repo list` reports the resolved source as each repo's `root`, using the git-free source reader so an uncloned git source is listed with an empty root and never cloned. Knowledge aggregation, the repo-routed changelog store, and init's cascade were confirmed by regression tests to stay on the root.

**Deviations**: None in behaviour. The relative-source oracle in the tests uses `../../../code` from `repos/api/.spektacular/repo.yaml`, since relative sources anchor at the directory holding `repo.yaml`.

**Files changed**:
- `cmd/repo.go`
- `cmd/repo_test.go`
- `cmd/knowledge_test.go`
- `cmd/changelog_file_test.go`
- `cmd/init_test.go`

**Discoveries**: `repo add` must write `repo.yaml` before calling `Resolve`, because resolution is what clones a git source and it reads the source from that file; the previous order (resolve first, then write metadata) would have missed a git source declared in the same registration. The colocated missing-config tolerance in knowledge aggregation keys on the footprint root, so a colocated repo that declares a source elsewhere keeps its knowledge at the project root.

### 2026-09-02 — Phase 2.1: Remove the root changelog step from the implement workflow

**What was done**: The `update_repo_changelog` state, its callback, and its template were deleted; `update_changelog` now advances straight to `test_plan`. The finished step and the update-changelog skill no longer mention a repo-level `CHANGELOG.md`, and the step-order, FSM-walk, STOP-map, and branch tests were updated, with two new tests pinning that the removed step is not a legal state and that no implement template names `CHANGELOG.md`. The harbor implement suite's expected step order and its seeded `state.json` fixture dropped the step.

**Deviations**: The harbor suite's `environment/state.json` fixture also listed the removed step in `completed_steps`; the plan did not name it, so it was updated alongside `EXPECTED_STEP_ORDER`. The finished step's replacement bullets are added in Phase 2.2.

**Files changed**:
- `internal/steps/implement/steps.go`
- `internal/steps/implement/steps_test.go`
- `templates/steps/implement/08-update_repo_changelog.md` (deleted)
- `templates/steps/implement/07-update_changelog.md`
- `templates/steps/implement/12-finished.md`
- `templates/skills/skill_update-changelog.md`
- `templates/context_directive_test.go`
- `cmd/docs_test.go`
- `cmd/implement_test.go`
- `tests/harbor/implement-workflow/tests/test_implement_workflow.py`
- `tests/harbor/implement-workflow/environment/state.json`

**Discoveries**: The command-level implement tests hard-code the step count (thirteen) in three places and the goto schema test asserted the removed step by name, so any future FSM change must touch `cmd/implement_test.go` as well as the steps package tests.

### 2026-09-02 — Phase 2.2: Fold the release note into the per-repo changelog record

**What was done**: The feature-changelog step now tells the agent to open each per-repo record with a two-to-four sentence user-facing summary written for a reader who has never seen the plan, framing that record as the repo's release note, with the reference line directly beneath it. The finished step's summary and report now name both the project record's path and each per-repo record (read through `changelog file read --repo`).

**Deviations**: None. Criterion 3 (exactly one project record and one per affected repo at run time) is behavioural and is proven by the harbor run in Phase 3.3.

**Files changed**:
- `templates/steps/implement/10-update_feature_changelog.md`
- `templates/steps/implement/12-finished.md`
- `internal/steps/implement/steps_test.go`

**Discoveries**: None beyond the phase.

### 2026-09-02 — Phase 3.1: Carry each repo's source in the roster and give it to the implement workflow

**What was done**: Roster entries now carry each repo's resolved `source` (read through the same git-free reader listing uses; empty when the code is not on disk). The implement command injects the roster into workflow data on `new` and on every `goto`, and the read-plan, analyze, implement, test, verify, and feature-changelog callbacks pass it to their templates, which open with a "Where the code lives" block listing each repo's name, metadata, and source, with fallbacks for a repo whose code is not on disk and for a project with no registered repos.

**Deviations**: The roster block was added to the six templates in this phase rather than 3.2, because the phase's third criterion (rendering embeds each repo's name and source) cannot pass without it; Phase 3.2 rewords the remaining stand-in phrases and adds the contract test.

**Files changed**:
- `internal/repo/roster.go`
- `internal/repo/roster_test.go`
- `cmd/implement.go`
- `cmd/implement_test.go`
- `internal/steps/implement/steps.go`
- `internal/steps/implement/steps_test.go`
- `templates/steps/implement/01-read_plan.md`
- `templates/steps/implement/02-analyze.md`
- `templates/steps/implement/03-implement.md`
- `templates/steps/implement/04-test.md`
- `templates/steps/implement/05-verify.md`
- `templates/steps/implement/10-update_feature_changelog.md`

**Discoveries**: The command-level implement tests share one cobra root command, and the `--schema` flag set by an earlier schema test persists into later tests in the same package run; a test that drives `implement new` after them must call `resetImplementCommandFlags` first or the command prints a schema and never writes state.

### 2026-09-02 — Phase 3.2: Reword templates to direct the agent to each repo's source

**What was done**: The roster lines in the spec interview and the plan discovery and architecture steps now show each repo's source (with a `repo list` pointer for a repo whose code is not on disk), and their no-repos fallbacks describe a project of one repo instead of "the colocated repo". The implement read-plan drift check, test, and verify steps say the work runs in the repo's source; the changelog steps define the path prefix by the number of registered repos (required whenever more than one is registered, unprefixed meaning the only registered repo). The spawn-implementation, spawn-planning, and gather-metadata skills and the spek-plan and spek-implement workflow skills point at each repo's source rather than a resolved root or the running directory. A new template-contract test bans the stand-in phrases across every step and skill template and asserts the nine roster templates render sources and fallbacks.

**Deviations**: The banned-phrase list also caught "resolved local path" in the repo-management skill, so that one phrase was reworded here ahead of the skill's full rewrite in Phase 4.2. The plan's line references for the gather-metadata skill were stale (the file is 39 lines); one sentence was added under its Commands heading instead.

**Files changed**:
- `templates/steps/plan/02-discovery.md`
- `templates/steps/plan/03-architecture.md`
- `templates/steps/spec/00b-interview.md`
- `templates/steps/implement/01-read_plan.md`
- `templates/steps/implement/04-test.md`
- `templates/steps/implement/05-verify.md`
- `templates/steps/implement/07-update_changelog.md`
- `templates/steps/implement/10-update_feature_changelog.md`
- `templates/skills/workflows/spek-plan/SKILL.md`
- `templates/skills/workflows/spek-implement/SKILL.md`
- `templates/skills/skill_spawn-implementation-agents.md`
- `templates/skills/skill_spawn-planning-agents.md`
- `templates/skills/skill_gather-project-metadata.md`
- `templates/skills/skill_manage-repos.md`
- `templates/repo_source_test.go`
- `internal/steps/plan/steps_test.go`
- `internal/steps/spec/steps_test.go`
- `internal/steps/implement/steps_test.go`
- `internal/agent/instruction_surface_test.go`
- `cmd/skill_test.go`

**Discoveries**: Template wording is pinned in four places (the steps package render tests, the templates package contract tests, the rendered-skill test in the agent package, and the cmd skill tests), so any future rewording must be checked against all four.

### 2026-09-02 — Phase 3.3: Prove the separate layout end to end in the harbor implement suite

**What was done**: The harbor implement suite's fixtures now register both repos with `location`, keep the `docs` repo's Spektacular files under the project at `/app/repos/docs/.spektacular/` with `source: file:///opt/docs-repo`, and leave `/opt/docs-repo` as a bare code checkout. The hand-maintained oracles moved with them (the docs record path, the artifact copies, the instruction's success criteria), and two verifier tests were added asserting the source directory gained no `.spektacular/` and that no `CHANGELOG.md` exists under the source or the project. One full harbor run (`tests/harbor/jobs/2026-09-02__14-04-20`) passed all 14 verifier tests.

**Deviations**: None. The plan's open question (whether a live agent attributes files correctly under the restated prefix rule) is settled by this run: the `docs: ` prefixed files produced exactly one docs record at the project-side footprint and one colocated record, with no stray file.

**Files changed**:
- `tests/harbor/implement-workflow/environment/config.yaml`
- `tests/harbor/implement-workflow/environment/docs-repo.yaml`
- `tests/harbor/implement-workflow/environment/Dockerfile`
- `tests/harbor/implement-workflow/task.toml`
- `tests/harbor/implement-workflow/tests/test_implement_workflow.py`
- `tests/harbor/implement-workflow/instruction.md`

**Discoveries**: The verifier runs inside the container against the live paths (`/app`, `/opt/docs-repo`), not against the copied artifacts, so source-cleanliness checks can read the code checkout directly; the artifact copies exist only for post-mortem inspection.

### 2026-09-02 — Phase 4.1: Document `source`, `location`, and the layouts on the documentation site

**What was done**: On the configuration page the project `repos` key is rewritten around `name` and `location` (noting the `local` alias and the removed `address`), the environment-variable note covers `repo.yaml`, the repository example and key reference gain `source` with both the file and git forms and the contrast with the project-level `source`, and the section count says nine. On the projects page the `repo list` sample and the registration example use `location` and `source`, the configuration-split section gains a `source` key, a new "Colocated or separate" section shows the colocated layout and the separate layout with a file source and with a git source, each as a directory tree, followed by both registration commands, and the cloning section describes a git `source`.

**Deviations**: None. The site builds, `astro check` reports no page diagnostics, the MDX guard finds no layout markup, and the two pages contain no em dashes.

**Files changed**:
- `docs: src/pages/configuration.mdx`
- `docs: src/pages/projects.mdx`

**Discoveries**: The new section's `surface={false}` slots between the configuration-split block (surface true) and the "Work that spans repositories" section (surface true), so the alternating shading held without touching the following sections.

### 2026-09-02 — Phase 4.2: Update the README and the repo-management skill

**What was done**: The README's configuration section now shows `location` in the registry example and `source` in the repo configuration example, explains that a repo's Spektacular files can sit inside its code or in a folder of their own pointing at a checkout on disk or at a git repository, notes the `local` alias and the removed `address` key, and links to the documentation site for the layouts; the project structure tree gained `repo.yaml` and `changelog/`. The repo-management skill explains the two locations and the colocated and separate layouts, shows `repo add` payloads with `location` and with a file source and a git source, states that `location` is required and created when missing, and describes listing and materialization in terms of `source`. A new command test pins the skill's payloads, layouts, and wording.

**Deviations**: The skill still names `address` twice, only to say the registry no longer takes it and that a payload carrying it is rejected; the test therefore forbids any `"address":` payload key and address-only wording rather than the bare word. One pre-existing em dash inside the README's configuration section (the breaking-change note) was replaced while editing that section; em dashes elsewhere in the README were left as they were.

**Files changed**:
- `README.md`
- `templates/skills/skill_manage-repos.md`
- `cmd/skill_test.go`

**Discoveries**: None beyond the phase.
