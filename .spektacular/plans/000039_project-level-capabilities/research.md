---
created_date: "2026-08-13"
status: completed
closed_date: "2026-08-13"
---

# Research: 000039_project-level-capabilities

Decision log for planning the project/repo split, provider-based repo registry, clone-on-absent materialization, namespaced + derived changelogs, `.spektacular_ignore` exclusion, and cross-repo planning/implementation.

## Alternatives considered and rejected

Dead ends recorded in prior work — do not re-propose:

- **Parent-directory search for config/project root** — never existed; `projectRoot()` is cwd unconditionally (cmd/root.go:221-227). The spec explicitly forbids adding it (spec Requirements: "Single point of execution").
- **Ripgrep / any external tool for search** — deliberately removed by 000026; store search is pure Go, in-process (internal/store/search.go:36-40 documents "no external tool is consulted"). Any new external-binary dependency must be explicitly argued.
- **Growing the `store.Store` interface for filesystem conveniences** — rejected in 000021 (`WriteFrom` rejected); file-reading is a CLI-layer concern. Repo providers must not become per-file proxies (also a spec constraint).
- **Manifest/index or single growing file for changelogs** — rejected in 000033; flat one-file-per-feature, filename-as-identifier is load-bearing. Namespacing must preserve retrieval-by-name (`changelog file read <project>/<name>.md`).
- **Config-declared registries for code-owned truths** — 000028 rejected tiers-in-config; keep code-owned registries (categories, providers) in code. The project/repo split moves only genuinely user-varying settings.
- **Verbose written defaults** — 000020 rejected baking defaults into written config.yaml; defaults are synthesized at load via `WithDefaults(projectRoot)` (internal/config/config.go:274-289). Preserve this grain for both new config files.
- **FSM workflow for simple CRUD-orchestration features** — 000022 rejected a knowledge FSM in favor of a thin SKILL.md over CLI commands. A repo-management skill should be a thin playbook, not a workflow.
- **`--confirmed` flags / confirmation in Go** — 000022: propose-then-confirm lives in skill/template prose, not binary code. Applies to clone confirmation and repair offers.

Live alternatives the architecture step must weigh (2-3 viable approaches each):

1. **Git access for clone + staleness check**
   a. Shell out to `git` binary via `os/exec` — zero new Go deps, but the codebase's first-ever subprocess and mandatory external tool (contradicts 000026's grain; needs explicit argument that git is different: it's a repo-provider concern, only invoked for the `address`-based provider, and the user's repos are git repos by definition).
   b. `github.com/go-git/go-git/v5` — pure Go, no external binary, and its `plumbing/format/gitignore` package also solves the `.spektacular_ignore` matcher; but a heavy transitive dependency tree on a codebase with only 5 direct deps (go.mod:6-12).
   c. Hybrid: git binary for clone/fetch-check, hand-rolled or small-lib matcher for ignore.
2. **`.spektacular_ignore` matcher**
   a. `go-git/v5/plumbing/format/gitignore` (comes free with option 1b).
   b. `github.com/sabhiram/go-gitignore` — tiny, single-purpose.
   c. Hand-rolled subset matcher — matches house style (hand-rolled search in internal/store/search.go) but gitignore semantics (negation, anchoring, `**`) are easy to get subtly wrong.
3. **Where exclusion hooks in**
   a. A filter predicate passed into `FileStore` walk/list — one place, but contradicts the documented store-is-domain-agnostic boundary (internal/store/search.go:41-43; 000028 moved category exclusion *out* of the store).
   b. Store stays agnostic; a shared matcher built once and applied by each consumer layer (knowledge set, artifacts scan, storefile list) — follows 000028 precedent but there are three independent walkers to teach (internal/store/search.go:57-98; internal/knowledge/set.go:290-313; cmd/artifacts.go:180-268).
   c. New `IgnoreAwareStore` wrapper implementing `store.Store` around `FileStore` — keeps `FileStore` pristine, single hook point, composes with arbitrary roots (repo stores).
4. **Config split mechanics**
   a. Two files, two types (`ProjectConfig`, `RepoConfig`), merged at load into the existing `config.Config` value so ~17 `loadConfig()` call sites stay untouched (cmd/root.go:196-205 is the single loader).
   b. Two files surfaced as two separate types plumbed everywhere — touches every call site; avoid.

## Chosen approach — evidence

Evidence supporting the recommendation the architecture step will formalize (pattern: clone the knowledge-sources shape for `repos`, keep a single merged `config.Config`, provider dispatch by switch, stores rooted at resolved repo paths):

- **Repos list must mirror knowledge-sources exactly**: `KnowledgeConfig{Sources []SourceConfig}` internal/config/config.go:93-103; per-entry `{scope, provider, config}` with typed provider block `FileKnowledgeConfig` config.go:106-108; per-entry validation + uniqueness check config.go:250-268; `WithDefaults(projectRoot)` config.go:274-289. Spec's Technical Approach names this pattern explicitly.
- **Provider dispatch exemplar**: `knowledge.NewSet` switch on `src.Provider`, unknown provider fail-fast, relative paths joined to projectRoot, `os.Stat` reachability check (internal/knowledge/set.go:70-95). Clone-on-absent = provider-level materialization running *before* the reachability contract, not a weakening of it (framing from 000020's fail-fast design).
- **Stores at arbitrary roots already work**: `store.NewFileStore(location, scope)` used with foreign roots at internal/knowledge/set.go:88; tests exercise foreign temp roots (internal/knowledge/set_test.go:26-52). Derived per-repo changelog writes = a `FileStore` rooted at the member repo, path `changelog/<project-name>/<name>.md`, reusing the write path (read-existing → strip frontmatter → metadata.Merge → Write) at cmd/storefile.go:115-153.
- **Single config seam**: config path computed only at cmd/root.go:185-191; only loader cmd/root.go:196-205; missing file currently returns `NewDefault()` silently (root.go:201-203) — this is the exact line the spec's "explicit error when no project" changes. All ~17 call sites consume `config.Config` by value (list in cmd files: root.go:60, init.go:36, version.go:109, knowledge.go:144, storefile.go:90/122/270, artifacts.go:76, spec.go:177/284/332, plan.go:107/179/231, implement.go:112/191/243, internal/project/init.go:29).
- **CLI command shape for `repo add`/`repo list`**: `--data` JSON + `--schema` pattern in cmd/knowledge.go:15-67, 152-154, 305-323, 337-346; `repo list` ≈ `runKnowledgeSources` (cmd/knowledge.go:236-250, `SourceInfo` projection internal/knowledge/set.go:57-62, 258-264). New top-level group registered at cmd/root.go:236-244.
- **Config write-back**: only writer is `Config.ToYAMLFile` (config.go:292-301), used by internal/project/init.go:83-87 (written only if absent). `repo add` is the first CLI config mutation — full marshal round-trip drops user comments (accepted; see assumptions).
- **Init cascade seams**: cmd/init.go:21-57 (cwd at :27 is the sole root notion; force=true always); internal/project/init.go:17-117 — `MkdirAll` top-up is the existing create-or-repair pattern (:55-80, comment :64-68); config written only if absent :83-87; `.gitignore` from embed :90-96 (template templates/.spektacular/.gitignore — currently *overwritten* on re-init, so `repos/` entry belongs in the template; append-if-missing for user-edited files would be new behavior). Version file write cmd/version.go:126-135; `staleAction` composes `<command> init <agent>` (version.go:107-120).
- **Repo footprint = subset of project init**: knowledge dirs + per-category READMEs (internal/project/init.go:69-74, 102-114) + repo config file. No agent guidance, no skills, no version file (spec: minimal footprint).
- **Skills**: library skills are convention-resolved flat files `templates/skills/skill_<name>.md`, served raw/unrendered (cmd/skill.go:50-63) — cheapest path for a repo-management skill, but NOTE: no `{{command}}`/`{{config.command}}` substitution happens there. Workflow skills need Go registration (internal/agent/skills.go:25-30 AND internal/agent/commands.go:18-23). Best structural model: templates/skills/skill_update-changelog.md (create-vs-repair branching).
- **Workflow template surfaces for cross-repo behavior** (mostly template-only changes): plan discovery templates/steps/plan/02-discovery.md:13-37 (research list + research.md's 7 sections), architecture 03-architecture.md:25-36; implement changelog trio — 07-update_changelog.md:24-27 (files-changed list), 08-update_repo_changelog.md:7-22 (assumes exactly one repo root — most impacted), 10-update_feature_changelog.md:24-36 (store write; derived entries extend here). New template *variables* (e.g. registered repos) require Go: strategy PathVars (internal/steps/implement/strategy.go:47-63, internal/steps/plan/strategy.go) or `Extra` map (example internal/steps/spec/steps.go:150); `workflow.Config` (internal/workflow/workflow.go:17-29) has no repo field today.
- **Changelog metadata**: front matter owned by internal/metadata (metadata.go:48-52: created_date/status/closed_date only) — derived entries need extra structured fields (project name, source, spec/plan ids), so metadata schema or a second front-matter shape must be extended. ID-prefix rule enforced on changelog writes via requireID=true (cmd/changelog_file.go:11-13, validateIDPrefix cmd/storefile.go:66-80) — namespaced paths `<project>/<name>.md` must keep the ID in the correct path segment (validateIDPrefix checks the *leading* segment — will need adjustment for project-folder prefixes).
- **AGENTS.md managed sections**: heading-delimited, installManagedSection (internal/agent/managed_section.go:21-52); adding a section = template + 3-line wrapper + one call in each of claude.go:25-39, bob.go:20-32, codex.go:17-29 (triplicated). historical_artifacts.go is a divergent copy — model new wrappers on memory_context.go instead.
- **Testing conventions**: testify/require, t.TempDir/t.Chdir, cobra flag reset footgun (cmd/knowledge_test.go:53-64), CLI tests through `runRoot()` (cmd/knowledge_test.go:112+), config validation tests assert on key path in error (internal/config/config_test.go:231-261), init idempotency contract tests (cmd/init_test.go:193-218 — init must never disturb non-managed files; the cascade inherits this), instruction-surface regression tests (internal/agent/instruction_surface_test.go).

## Files examined

- internal/config/config.go:111-120 — Config struct: command/agent/threshold/debug/spec/plan/changelog/knowledge; no repo concept.
- internal/config/config.go:93-108,250-289 — knowledge-sources list shape, validation w/ scope uniqueness, WithDefaults; the template for `repos`.
- internal/config/config.go:29 — ProviderFile is the only provider; provider-keyed config decoding deferred since 000020.
- internal/config/config.go:167-183,292-301 — FromYAMLFile (env expansion, defaults-prefilled unmarshal, Validate); ToYAMLFile (only config writer).
- cmd/root.go:185-227 — configFilePath/loadConfig/dataDir/projectRoot: strictly cwd; missing config → silent NewDefault() (the line the "explicit error" requirement changes).
- internal/knowledge/set.go:70-95 — provider dispatch switch; fail-fast unreachable-source check; foreign-root store construction (:88).
- internal/knowledge/set.go:102-142,258-274,290-313 — Search fan-out/ranking, Sources() introspection, listFiles recursion (a walker needing exclusion).
- internal/store/store.go:37-151 — Store interface (Root/Read/Write/Delete/List/Exists/Search); FileStore; abs() path-escape guard (:81-88); List is one-level-deep.
- internal/store/search.go:36-102,154-219 — pure-Go walk, AND substring terms, no exclusion mechanism; doc comment declaring store never excludes by name (:38-43).
- internal/metadata/metadata.go:48-52; frontmatter.go:16-70; merge.go:29-82 — front-matter fields and merge invariants; derived entries need new structured fields.
- cmd/storefile.go:66-80,90-153,326 — generic store-file command factory; validateIDPrefix on leading path segment; write path (strip/merge/write).
- cmd/changelog_file.go:8-14 — changelog file commands = factory instantiation with requireID=true.
- internal/steps/implement/strategy.go:32-34,47-63 — ChangelogFilePath flat layout; PathVars exposed to implement templates.
- cmd/knowledge.go:15-67,143-153,236-250,305-346 — the CLI shape to copy for `repo add`/`repo list`.
- cmd/init.go:21-57 — init flow: Lookup → Getwd → project.Init(force=true) → config agent set → version file → agent.Install.
- internal/project/init.go:17-117 — dirs via MkdirAll (top-up/repair pattern), config only-if-absent, .gitignore from embed (overwritten on re-init), category READMEs overwritten.
- cmd/version.go:34-135 — version check states match/mismatch/missing, always exit 0; staleAction builds `<command> init <agent>`.
- internal/agent/agent.go:20-45; claude.go:18-87; bob.go:13-33; codex.go:13-30 — Agent interface, registry, triplicated install lists; claude adds CLAUDE.md @AGENTS.md import.
- internal/agent/skills.go:25-65 — workflowSkills manifest (source of truth), mustache {{command}} render at install.
- internal/agent/managed_section.go:21-140 — heading-delimited managed sections, atomic writes.
- cmd/skill.go:50-103 — library skill resolution by filename convention, raw/unrendered serving, auto-discovery in listSkills.
- templates/steps/plan/02-discovery.md:13-37; 03-architecture.md:25-36 — where cross-repo attribution guidance goes.
- templates/steps/implement/07-update_changelog.md:24-27; 08-update_repo_changelog.md:7-22; 10-update_feature_changelog.md:24-36 — the three distinct changelogs; 08 assumes a single repo root.
- internal/workflow/workflow.go:17-29,43-46,73-158 — workflow.Config (no repo field), StepConfig, FSM construction.
- internal/stepkit/stepkit.go:21-30,55-94,121-127 — PathStrategy, var merge order, template rendering.
- go.mod:6-12 — five direct deps; no git library, no exec usage anywhere (greenfield for clone).
- templates/.spektacular/.gitignore — current ignore template; needs `repos/` entry.
- cmd/knowledge_test.go:13-112; internal/knowledge/set_test.go:12-58; internal/config/config_test.go:105-261; cmd/init_test.go:39-218 — test conventions and contracts to preserve.

## External references

- https://github.com/go-git/go-git (v5) — pure-Go git; would cover clone + behind-remote check + `plumbing/format/gitignore` for `.spektacular_ignore` matching; one dep solves two greenfield gaps but brings a large transitive tree.
- https://github.com/sabhiram/go-gitignore — minimal gitignore-pattern matcher; alternative if git access is via the git binary.
- gitignore(5) pattern semantics (https://git-scm.com/docs/gitignore) — the spec constraint says `.spektacular_ignore` "must follow the gitignore pattern convention"; negation/anchoring/`**` semantics are the correctness bar for any matcher choice.
- `git rev-list --count HEAD..@{upstream}` / `git fetch --dry-run` style staleness checks — relevant only if the git-binary route is chosen; note spec forbids any actual fetch/pull.

## Prior plans / specs consulted

(All via `go run . spec file read` / `plan file read`; historical intent, not current behavior.)

- **000020_context** (spec+plan+research) — birthplace of provider pattern; provider-keyed config decoding explicitly deferred "until a second provider lands" (this feature is that trigger); defaults synthesized not written; NewSet fail-fast contract; rejected: multi-store through StepCallback, agent-greps-directories, Walk() method.
- **000014_spektacular_store** (plan) — Store interface origin; store is kind-agnostic and directory-rooted; kinds live in config+cmd wiring only (why changelog store cost zero store code).
- **000021_file-writing** (spec+plan) — `--from` staging convention; rejected store-interface growth; instruction-surface regression tests police agent-facing command shapes.
- **000022_spek-knowledge-skill** (spec+plan) — thin SKILL.md over CLI CRUD beat an FSM; confirmation gates live in prose; workflowSkills registry changes require per-agent tests + fixture-FS tests in lockstep.
- **000028_knowledge-base-categories-tiers-and-dedup** (spec+plan) — code-owned registries stay in code; generic store made category-agnostic, exclusion moved up to domain layer (the layering precedent for `.spektacular_ignore`); layered scope precedence chosen.
- **000026_ripgrep-replace / 000027_document-level-search** (specs) — zero-external-tool constraint (strongest prior tension with git clone); no index/cache/daemon; document-level one-hit-per-doc ranking.
- **000033_spec-workflow-output-changelog** (spec+plan) — changelog store pattern (third clone of "configured store kind"); flat filename-as-identifier is load-bearing (manifest and single-file rejected); unconditional FSM step writes the record; plan-slug-equals-spec-slug convention underpins spec_path.
- **000039_version-check** (spec+plan) — version file inside `.spektacular/` (precedent tension with root-level `.spektacular_ignore` naming — but spec fixes the name and gitignore convention puts ignore files at the root they govern); staleness is a state report not an error; skills forbid agent self-reinstall (cascade must stay user-initiated).
- **000013_sensible_defaults** (spec) — empty template; nothing to learn.

## Open assumptions

- The locally modified spec on disk (uncommitted changes to 000039_project-level-capabilities.md) is the authoritative version to plan from — confirmed implicitly by the user choosing it, not explicitly re-verified against git HEAD.
- Losing user comments/formatting in `.spektacular/config.yaml` when `repo add` writes config back (full yaml.Marshal round-trip) is acceptable. If not, a comment-preserving YAML editor (e.g. yaml.Node surgery) becomes a requirement — implement workflow must STOP and ask if this surfaces as a problem.
- `git` availability: if the git-binary route is chosen, git being on PATH becomes a runtime requirement for address-based repos only; assumed acceptable since target users work in git repos.
- Two specs share ID 000039 (`000039_version-check` and `000039_project-level-capabilities`); plan directories differ by slug so no collision, but ID-based references are ambiguous — assumed tolerable, worth flagging to the user.
- The spec's "running Spektacular where no project is configured produces an explicit error" is assumed to still permit `init` (and probably `version`/`skill`) to run without a project — otherwise a project could never be created. The exact command allowlist is an architecture-step decision.
- "Full agent guidance and capabilities" for project init = current init output (skills + AGENTS.md sections + CLAUDE.md import); the repo footprint = repo config + knowledge storage only, per spec acceptance criteria.
- Assumed the plan should keep spec/plan/changelog config sections at project level and knowledge+changelog-provider sections at repo level, per spec Requirement 1 — the project's own colocated repo also carries a repo config file (spec constraint: two separate files even when colocated).

## Drafting assumptions

### Plan slug taken from the workflow, not renumbered (assemble)
- **Decision**: the plan keeps the CLI-assigned name `000039_project-level-capabilities` (matching the spec) without proposing a renumbered slug to the user.
- **Rationale**: `plan new` already fixed the name, and the plan-slug-equals-spec-slug convention is load-bearing for spec_path resolution in the implement workflow; the determine-feature-slug skill's 4-digit renumbering guidance predates that convention.
- **Rejected**: proposing `0042-...` style renumbering via AskUserQuestion (would break the spec↔plan link and diverge from every existing plan).

### Project name from directory basename, overridable with --name (phases)
- **Decision**: `init` derives the required project name by slugifying the current directory's basename, with an optional `--name` flag to override.
- **Rationale**: init currently takes only the agent argument; a mandatory second argument would break every documented invocation, and the directory name is almost always the intended identity.
- **Rejected**: mandatory `--name` (breaks `init <agent>` muscle memory and harbor suites); interactive prompt (init is non-interactive today).

### Changelog namespace injected below the CLI surface (phases)
- **Decision**: the project-name folder is added by the changelog storeDirFunc and path strategy, not by agents — agent-facing paths stay `<plan_name>.md` and `validateIDPrefix` is unchanged.
- **Rationale**: satisfies the on-disk layout requirement with zero changes to agent-facing contracts, templates that reference paths, or ID validation.
- **Rejected**: making agents write `<project>/<plan_name>.md` (touches every template and the ID-prefix validator for no user-visible gain).

### workflow.Config gains a ProjectName scalar (phases)
- **Decision**: amend the earlier "no repo fields in workflow.Config" call: a single `ProjectName string` scalar is added so the implement path strategy can render the true changelog path; registry data still never flows through workflow config.
- **Rationale**: consistent with the existing ChangelogDir scalar precedent; the alternative (templates guessing the namespace) breaks the changelog_path variable's accuracy.
- **Rejected**: resolving the namespace inside templates via CLI calls (changelog_path is a render-time variable, not agent output).

### repo list and init never clone; materialization happens on use (phases)
- **Decision**: `repo list` reports address-only repos as unmaterialized rather than cloning them, and the init cascade skips unmaterialized repos with a notice; cloning is triggered only when a repo is actually resolved for work.
- **Rationale**: the spec ties cloning to use ("cloning happens only when the repo is absent" on touch); list and init are inspection/repair operations and must stay fast and side-effect-free.
- **Rejected**: eager materialization at registration or init (surprising network side effects; contradicts warn-don't-act posture).

### Duplicate knowledge scopes across repos resolve by registry order (phases)
- **Decision**: aggregated knowledge sources keep their repo-config scopes; scope-addressed read/write picks the first match in registry order (colocated repo first); `knowledge sources` output gains repo attribution.
- **Rationale**: per-config scope uniqueness still holds; cross-repo collisions are legal and need a deterministic, explainable rule without new addressing syntax.
- **Rejected**: compound `repo/scope` addressing (breaks existing CLI contracts and skills); forbidding duplicate scopes across repos (a repo can't know what other repos declare — violates repos-know-nothing).

### Provenance auto-stamped by the CLI on derived writes (phases)
- **Decision**: `changelog file write --repo` stamps project name, project source, and spec/plan identifiers into front matter itself, from config and the filename convention; agents author only the body and reference line.
- **Rationale**: structured provenance is mechanical and must be reliable — agents forget; the CLI already owns front-matter merging.
- **Rejected**: agent-authored front matter (unreliable, and the metadata layer strips/normalizes it anyway).

### Exclusion ships with cross-repo planning, not with the registry (milestones)
- **Decision**: `.spektacular_ignore` support lands in Milestone 4 alongside cross-repo planning/implementation, not in Milestone 2 with repo registration.
- **Rationale**: the spec motivates exclusion by search usefulness during planning ("cross-repo searches during planning stay useful"); it has no consumer until agents search across repos, and M2 is already the largest milestone.
- **Rejected**: bundling into M2 (front-loads an independent mechanism into the heaviest milestone); its own milestone (too small to stand alone).

### Repo info flows to agents via CLI, not template variables (data_structures — AMENDED at walkthrough)
- **Decision (amended)**: split by volatility. Identity metadata (name, description, role, tags, deployment) IS preloaded into the rendered discovery and architecture instructions via a repo-roster template section, read from config at render time and passed through the existing `Extra` map — no `workflow.Config` or stepkit changes. Dynamic state (resolved roots, materialization, staleness) still flows only through `repo list`, and `changelog file write --repo` is unchanged.
- **Rationale**: the user challenged the original call during the walkthrough — an instruction that contains the roster is more reliable than one that asks the agent to fetch it, and attribution quality (spec success metric 2) depends on the agent actually seeing the repos; config-sourced identity is exactly as fresh at render time as a CLI call. The original rationale still governs the dynamic half.
- **Rejected**: the original CLI-only design (agent may skip the command); an ambient AGENTS.md roster section (repo removal is a manual config edit with no command hook, so a cached roster is guaranteed to rot — config must stay the single source of truth); rendering resolved paths/staleness into instructions (genuinely stale-at-render-time).

### Derived changelog writes via `--repo` flag on the existing factory (data_structures)
- **Decision**: extend the shared changelog `file write` command with an optional `--repo <name>` that routes the write through the named repo's configured changelog provider, instead of adding a new top-level command.
- **Rationale**: reuses the staging/`--from`/metadata-merge pipeline verbatim; keeps one write path for all changelog entries as the spec's reuse constraint demands.
- **Rejected**: `repo changelog write` command (duplicates the storefile pipeline); Go-side automatic derivation with no agent involvement (the entry body is filtered/authored content — agent work per house trust model).

### Chosen direction: git binary + small gitignore library + store wrapper + single loader seam (architecture)
- **Decision**: Option B — clone/staleness via the user's `git` binary (`os/exec`, scoped to the git repo provider; staleness = read-only `ls-remote` comparison); `.spektacular_ignore` matching via a minimal pure-Go gitignore library; exclusion implemented as an ignore-aware `store.Store` wrapper filtering only List/Search; config split into `.spektacular/config.yaml` (project) + `.spektacular/repo.yaml` (repo) loaded through the single existing `loadConfig` seam, with missing project config becoming an explicit error (bootstrap commands exempt).
- **Rationale**: git-binary cloning inherits the user's credential helpers/SSH config — go-git cannot clone private repos without bespoke auth plumbing, which makes the feature unshippable in practice; gitignore semantics are too subtle to hand-roll safely; the wrapper keeps FileStore domain-agnostic per the documented boundary and the 000028 layering precedent; the single seam keeps ~17 config call sites untouched.
- **Rejected**: Option A (go-git for everything — auth gap, heavy dep tree); Option C (hand-rolled matcher — silent correctness risk in planning-time search results).

### Project config keeps the name config.yaml; repo config is repo.yaml (architecture)
- **Decision**: project-level file stays `.spektacular/config.yaml`; the new repo-level file is `.spektacular/repo.yaml` in each member repo; colocated solo projects hold both files in one `.spektacular/` directory.
- **Rationale**: keeping config.yaml minimizes churn across code, tests, and docs; the spec requires two separate files but does not name them; `repo.yaml` is self-describing.
- **Rejected**: renaming to `project.yaml`+`repo.yaml` (gratuitous breakage of every existing path reference); nesting repo settings inside config.yaml for the colocated case (spec constraint forbids single-file merge).

### Single `git` repo provider owning local-first resolution (architecture)
- **Decision**: ship one repo provider constant (`git`) whose resolution order is: use `local` when set and present, else clone `address` into `.spektacular/repos/<name>/` when absent; a local-only entry under this provider simply never clones.
- **Rationale**: the spec keeps `name`/`address`/`local` provider-agnostic, so the two shipped access methods are resolution branches of one provider rather than two providers; this matches "when both are set, local wins" as provider behavior.
- **Rejected**: separate `local` and `git` providers (forces the user to choose a provider that the local/address fields already imply; complicates the both-set precedence rule).

### Knowledge section moves wholesale to repo config (architecture — AMENDED at walkthrough)
- **Decision (amended)**: `repo.yaml` is the single knowledge authority for its repo — its own store by default, plus any additional sources that repo declares. Project config RETAINS an optional `knowledge.sources` list for sources owned by the project rather than any single repo (team/global scopes etc.), empty by default. The effective set aggregates repo-declared sources (registry order, colocated repo first) followed by project-owned sources; scope-addressed operations take the first match in that combined order.
- **Rationale**: the user corrected the original wholesale move during the walkthrough — forcing project-owned sources (e.g. a team knowledge share) into some repo's file puts them at the wrong level; each source declared exactly once at the level that owns it satisfies the spec's no-duplication constraint while keeping repo knowledge traveling with the repo.
- **Rejected**: the original wholesale move (no home for project-owned sources); duplicating knowledge config at both levels (spec forbids duplication — this design declares each source once, so it is not duplication); keeping everything project-level (contradicts the repo footprint definition).

### Staleness check is best-effort (architecture)
- **Decision**: the behind-remote check uses a read-only remote-refs comparison; failure to reach the remote (offline, auth) degrades to a notice, never an error, and no fetch/pull ever runs.
- **Rationale**: spec mandates warn-only; a hard failure on network absence would block offline work on an already-materialized clone.
- **Rejected**: `git fetch --dry-run` (closer to mutation, slower); failing hard when the remote is unreachable.

### Stale conventions entry mostly dropped (architecture)
- **Decision**: the legacy knowledge `conventions.md` (PEP 8 boilerplate in a Go repo) was dropped except its generic testing/documentation bullets, which match observed practice.
- **Rationale**: it is scaffold content contradicting the actual language and toolchain; citing it wholesale would be padding.
- **Rejected**: treating it as authoritative; ignoring it entirely (its README/CHANGELOG bullets genuinely bind this breaking change).

### Treat locally modified spec as authoritative (discovery)
- **Decision**: planned from the working-tree version of 000039_project-level-capabilities.md (which has uncommitted modifications), not git HEAD.
- **Rationale**: the user opened and selected this spec knowing it was modified; the working tree is what they're iterating on.
- **Rejected**: diffing against HEAD and asking which version to use — no signal the modifications were accidental.

### Defer the git-access decision to the architecture step (discovery)
- **Decision**: recorded go-git vs git-binary vs hybrid as live alternatives with evidence instead of stopping to ask the user during discovery.
- **Rationale**: the architecture step exists precisely to weigh options with evidence; both routes are viable and the comparison needs the full design context (the ignore-matcher choice interacts with it).
- **Rejected**: picking one during discovery (premature), asking the user now (not a blocking gap — no information only the user holds yet).

### Accept comment loss on CLI config writes (discovery)
- **Decision**: assume `repo add` may rewrite `.spektacular/config.yaml` via the existing full-marshal `ToYAMLFile`, dropping user comments.
- **Rationale**: matches the only existing config writer; comment-preserving YAML surgery is significant extra machinery not demanded by the spec.
- **Rejected**: yaml.Node-based comment-preserving editing — flagged in research.md open assumptions so the user/architecture step can override.

### "Explicit error without a project" exempts bootstrap commands (discovery)
- **Decision**: assume `init` (and likely `version`, `skill`) must still run where no project exists; the explicit-error rule applies to project-operating commands.
- **Rationale**: otherwise a project could never be created; the spec's intent is to kill silent default-config fallback, not to break bootstrap.
- **Rejected**: literal reading (every command errors) — self-contradictory with the init requirement.

## Rehydration cues

- Re-read the spec: `.spektacular/specs/000039_project-level-capabilities.md` (via Read within the active workflow, or `go run . spec file read 000039_project-level-capabilities.md`).
- `go run . knowledge always-applied` (conventions/glossary — currently placeholder READMEs only).
- `go run . knowledge read --data '{"scope":"project","path":"architecture/workflow-steps.md"}'` — FSM/step/template architecture primer.
- Key files to re-read cold, in order: internal/config/config.go, cmd/root.go:180-245, internal/knowledge/set.go, internal/store/store.go + search.go, cmd/init.go + internal/project/init.go, cmd/storefile.go, internal/steps/implement/strategy.go, templates/steps/plan/02-discovery.md, templates/steps/implement/08-update_repo_changelog.md.
- Prior-art re-reads: `go run . plan file read 000020_context/research.md`, `go run . plan file read 000033_spec-workflow-output-changelog/plan.md`.
- Working files for this plan: `.spektacular/work/000039_project-level-capabilities/` (this file, assumptions.md, later architecture.md etc.); cross-cutting state in `.spektacular/context.md`.