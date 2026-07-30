# Context: Project Level Capabilities

## Problem and motivation

GitHub issue #25 ("Project Level Capabilities") proposed a multi-repo project
structure for Spektacular: a central `.context` directory, a home-directory
`projects.yaml` registry, two-way repo linking, and bubble-up discovery so an
agent working in any linked repo automatically finds the wider project.

Working through the issue in discussion narrowed and re-scoped it
substantially. The core motivation that survived: a single unit of work (a
spec/plan) can legitimately span multiple repositories — e.g. a change that
adds code in one repo and updates documentation in another. Today Spektacular
has no notion of "project" versus "repo" at all: everything (spec, plan,
changelog, knowledge, command, agent settings) lives in one
`.spektacular/config.yaml`, resolved from `cwd` via `projectRoot()`
(`cmd/root.go:217-227`), with no home-directory config and no multi-repo
concept anywhere in the codebase (confirmed by direct code research, not
assumed).

The issue's original scope (viewer tool, GitHub PR inline commenting,
git-repo plugin, plan-dependency invalidation workflow, enterprise-tier
autonomous cross-repo execution) was deliberately narrowed down over the
course of discussion to a concrete, buildable foundation. Several original
ideas were explicitly dropped or deferred (see "Alternatives considered and
rejected" below) once we found they weren't needed to solve the actual
motivating problem.

## Requirements agreed in discussion

### 1. Config split: project config vs. repo config

- **Project config** (lives at the project root only) carries everything
  that governs how Spektacular itself runs: `command`, `agent`,
  `spec_trigger_threshold`, `debug`, `spec`, `plan`, `changelog`, and a new
  `repos` section (see below). Rationale: all of these are meaningful only
  once, for the whole project, never per-repo.
- **Repo config** (lives per repo, e.g. `.spektacular/config.yaml` inside
  each repo directory) carries only the knowledge source location for that
  repo — a single source of truth for "where does this repo's knowledge
  base live."
- **Single-repo projects remain valid.** In that case the project root and
  the repo root are the same directory, but there are still **two separate
  config files**, not one merged file. This preserves symmetry: a
  single-repo project is not a special case, just a project with one repo
  registered.
- User's own framing: "I think for a single repo you could have a project
  config and a repo config, there are two separate files" and "we need to
  move most stuff from repo config (current config) to the project config.
  Repo config should define the location of the knowledge base but this
  should be single source then spectacular uses configured projects in
  project config and reads their knowledge config."

### 2. Execution model: project-root-only, no bubble-up discovery

- Spektacular is **only ever invoked at the project root**, directly — not
  from inside an arbitrary repo subdirectory. There is no "bubble up parent
  directories to find `.context`" mechanism as originally proposed in the
  issue.
- Reasoning surfaced during discussion: `command` (how to invoke
  Spektacular's own build, e.g. `go run .`) can only be a single
  project-level value — you cannot run a per-repo command in a multi-repo,
  multi-language project. Once `command` was settled as project-level-only,
  bubble-up discovery from within a repo became unnecessary, since there is
  only ever one place execution starts.
- User's exact words: "spektacular is only executed at a project level" and,
  when asked directly whether bubble-up discovery is still needed: "No —
  must be invoked from the project root directly."

### 3. `init` forks on project vs. repo scope

- `init` (default, no flag) performs **repo-level** init: creates
  `.spektacular/` with the repo config (knowledge source location only) plus
  knowledge folders/READMEs, and writes a **minimal** `AGENTS.md`/`CLAUDE.md`
  stub whose only job is to point up to the project root for full
  instructions (so a repo opened standalone in an editor/agent session isn't
  left without any guidance).
- `init --project` performs **project-level** init: creates the full project
  config, writes the **full** `AGENTS.md`/`CLAUDE.md` (today's complete
  template set under `templates/agents/*.md`), installs `.claude/skills/*`
  (spek-new, spek-plan, spek-implement, spek-knowledge), and scaffolds
  spec/plan/changelog directories. Skills and full agent instructions are
  project-only artifacts — a repo joining a project does not get its own
  copy.
- User's exact words: "I think init with --project with repo being default,
  project should also create the agent config and skills, this is not
  needed at a repo level," and, on the minimal-stub question: "Repo still
  gets a minimal AGENTS.md pointing up to the project."

### 4. `repos` section in project config

- Modeled directly on the existing `knowledge.sources` pattern (a list of
  entries, not a single-instance config like today's `spec`/`plan`/
  `changelog`), reusing the existing `provider` + `config` convention and
  the existing `Store` interface (`Read`/`Write`/`Delete`/`List`/`Exists`/
  `Search`, `internal/store/store.go:37-55`) that already backs
  spec/plan/changelog/knowledge via `FileStore`.
- Each `repos` entry has three distinct parts, per the user: "It should have
  the provider, provider config, and metadata about the project,
  description, tags, etc."
  - `provider`: the storage/access mechanism. Only `file` exists in code
    today; `git` is referenced only as aspirational/future work in existing
    plans and specs (`.spektacular/plans/000014_spektacular_store/plan.md`,
    `.spektacular/specs/000020_context.md`) but is not implemented anywhere.
  - `config`: provider-specific configuration (e.g. a local path for
    `file`).
  - `metadata`: provider-agnostic descriptive information about the repo
    itself — **not** nested inside the provider config block. Agreed fields,
    carried forward in full from the original issue's projects.yaml
    description rather than trimmed: `description`, `tags`, `role`
    (architecture role), `depends_on` (dependencies on other repo entries),
    `deployment_type`.

### 5. Spec/plan/changelog storage stays single-location

- Plans, specs, and changelogs continue to live in **one project-level
  location each**, same shape as today's config, just relocated to project
  config. No per-repo storage splitting, no local-vs-shared plan
  distinction, no cross-plan dependency invalidation workflow — all three
  ideas from the original issue are explicitly deferred, not designed here.
- User's exact words when narrowing this: "For now, lets only have a single
  location for plans, specs, etc, we can deal with local later."

### 6. Multi-repo planning resolution

- This is the requirement that brought multi-repo execution back into scope
  after it was initially deferred. The motivating example, in the user's own
  words: "a single spec may create some code in one repo and update docs in
  another," and more specifically: "The agent should be able to update the
  code, so during plan phase, if for example a req is to update the
  documentation. The agent should be able to determine that the repo for
  that change is x and the files are y."
- Concretely: during the **plan** phase (discovery/architecture steps), the
  agent must be able to consult the `repos` list's metadata (`role`, `tags`,
  `description`) and search across each listed repo — not just the project
  root — to resolve a requirement to a specific repo and specific files
  within it.
- `context.md` remains a **single, project-level document** — it is not
  split per repo. It records the resolved repo/file information as part of
  its existing per-file/per-phase detail (context.md already carries
  file:line detail per the existing plan template design; this is additive,
  not a new document type).
- The deeper question of how to keep this efficient at scale (large numbers
  of repos/files blowing up context size) was explicitly raised and parked:
  "for plans we probably should think about how we are more efficient on
  tokens, but for now I think we can park that as a do later and have
  context project level."

### 7. Cross-repo file access mechanism

- Investigated directly against the `Store`/`FileStore` code
  (`internal/store/store.go`) before deciding scope. Key finding: `FileStore`
  is already fully generic — it takes an arbitrary `root` and `scope` at
  construction (`NewFileStore(root, scope string) *FileStore`,
  `store.go:66-68`) and every operation is path-jailed to that root via
  `abs()` (`store.go:80-88`), which rejects path traversal. Nothing about it
  assumes the root is the project root.
- Conclusion reached with the user: **no new provider and no new registry
  type is needed**. Each `repos` entry's `config` path already **is** a
  valid `FileStore` root. Once planning has resolved a file to its owning
  repo (per item 6), implement (and any repo-aware step, including
  changelog writing) constructs a `FileStore` directly —
  `store.NewFileStore(repoPath, repoName)` — using that repo's configured
  path, instead of always resolving through the single project-wide
  `projectRoot()` (`cmd/root.go:217-227`).
- This directly replaces today's hardcoded single-repo changelog assumption
  in `templates/steps/implement/08-update_repo_changelog.md` ("Create or
  update CHANGELOG.md at the repo root ... not inside .spektacular/ or the
  plan directory") — that step needs to resolve to the *correct* repo's
  root, not unconditionally "the" repo root.
- User's own framing when the discussion reached the minimal shape: "well,
  the file provider only really needs a root path and that is in the
  config" — i.e., no `repos.Set`-style lookup/registry object is required
  either, just direct `FileStore` construction per resolved repo using data
  already present in the `repos` list.

### 8. `.spektacular_ignore`

- A gitignore-syntax ignore file, **rooted at each provider source** (i.e.
  each `FileStore`'s own `Root()`, not one global file) — consistent with
  each `FileStore` being independently rooted per repo/knowledge source.
- Filename is exactly `.spektacular_ignore` (underscore, not the `ignore`
  suffix pattern of `.gitignore`) — user's exact words: ".spektacular_ignore
  would be rooted at the provider source and would follow the convention of
  gitignore."
- **Scope of effect:** honored only by `List` and `Search` — i.e., it
  filters discovery/indexing operations (so cross-repo planning search,
  item 6, doesn't wander into `node_modules`, `vendor`, build artifacts,
  etc., across every repo it now has access to). `Read`, `Write`, `Delete`,
  and `Exists` are unaffected by it — an explicitly-named path still works
  even if it matches an ignore pattern. This mirrors how `.gitignore` itself
  behaves (hides things from listing/status, but direct access still
  works).
- Why now, not later: becomes materially more important once file access
  spans arbitrary repo roots (item 7) rather than just the project's own
  `.spektacular/` tree.

## Spec workflow progress

- **overview** step completed and confirmed by the user with a simple "ok."
  Content saved to `.spektacular/work/000039_project-level-capabilities/overview.md`.
  Stakeholder-readable framing agreed: Spektacular currently treats every
  codebase as a single self-contained project; this feature introduces a
  project-level layer above individual repositories so a project can
  register its member repositories and Spektacular can plan/execute work
  (like a code change in one repo + a doc update in another) as one
  coherent unit instead of coordinating separate Spektacular setups
  manually.
- **requirements** step completed and confirmed by the user with a simple
  "ok." Content saved to
  `.spektacular/work/000039_project-level-capabilities/requirements.md`, ten
  checklist items covering: project/repo config separation, single point of
  execution (no bubble-up search), minimal repo-level init vs. full
  project-level init, repo registration with descriptive metadata,
  provider-extensible repo registration, centrally-located specs/plans/
  changelog, planning resolving requirements to a repo+files, carrying out
  cross-repo work in the correct repository, and path exclusion from
  discovery per repository. Deliberately kept at WHAT-level — no mention of
  gitignore syntax, `.spektacular_ignore` filename, `FileStore`, provider
  names, or config field names; those belong in Technical Approach / the
  plan workflow.
- **acceptance_criteria** step completed and confirmed by the user ("looks
  good"). Content saved to
  `.spektacular/work/000039_project-level-capabilities/acceptance_criteria.md`,
  ten blackbox criteria traceable one-to-one to the requirements. Note one
  criterion ("A repository can be registered using more than one access
  method") requires the config to *accept* at least two access methods —
  this tests extensibility of the registration mechanism, not that a second
  provider (e.g. git) is fully implemented; the git provider itself remains
  deferred per the scope decisions above.
- **constraints** step completed. Six constraints saved to
  `.spektacular/work/000039_project-level-capabilities/constraints.md`:
  provider-mechanism reuse, filesystem-only access this release,
  gitignore-convention ignore file rooted per source, exclusions affect
  List/Search only, two separate config files always, and — newly decided at
  this step — **no backward compatibility required**: the config split is a
  breaking change and existing single-config setups are expected to
  re-initialize (user chose this explicitly over old-shape-must-load and
  migration-command options).
- **technical_approach** step completed and confirmed ("this is ok"). Eight
  steers saved to
  `.spektacular/work/000039_project-level-capabilities/technical_approach.md`:
  knowledge-sources-style repos list, metadata alongside (not inside)
  provider block, direct file-store construction per repo (no registry),
  plan-phase resolution recorded in the existing context document, repo-aware
  changelog step, init flag with repo default / project opt-in,
  `.spektacular_ignore` filename, and parked future work (token efficiency,
  git provider).
- **success_metrics** step completed and confirmed ("ok"). Four behavioral
  metrics saved to
  `.spektacular/work/000039_project-level-capabilities/success_metrics.md`:
  end-to-end multi-repo flow within one project, correct requirement
  attribution without user correction, exclusion rules keeping search
  useful, and fast project setup. These were proposed by the assistant
  (metrics were not covered in the original discussion) and accepted as-is.
- **non_goals** step completed and confirmed ("ok"). Seven exclusions saved
  to `.spektacular/work/000039_project-level-capabilities/non_goals.md`:
  viewer tool (separate spec), local-vs-shared plans + PR sharing,
  dependent-plan invalidation, token-efficiency optimization, home-directory
  registry + upward discovery (called out because issue #25 readers might
  expect them), git-based provider, autonomous cross-repo execution.
  Migration tooling deliberately omitted — it is the inverse of the
  no-backward-compatibility constraint.
- **verification** step completed. Fresh-eyes subagent review returned six
  findings (all duplication/placement, no leaks or format issues); user
  approved all six proposed fixes ("ok"): dropped the parked-concerns bullet
  from Technical Approach, dropped the git-provider non-goal (implied by the
  filesystem-only constraint), trimmed the upward-discovery half of the
  home-directory-registry non-goal, trimmed the "coexist in the same
  location" clause from the config-split requirement, trimmed the
  direct-access clause from the exclusion requirement, and moved the
  `.spektacular_ignore` name mandate into the ignore-file constraint. Final
  spec committed via `spec file write` to
  `.spektacular/specs/000039_project-level-capabilities.md`, verified by
  reading back, working directory removed.

## Alternatives considered and explicitly rejected or deferred

These were raised during discussion, evaluated against the actual code, and
deliberately **not** carried into this spec's scope. They are recorded here
so future readers don't re-litigate them without knowing they were
considered:

- **Home-directory `projects.yaml`** (`~/.spectacular`, set once per user) —
  from the original issue. Superseded by putting `repos` directly in
  project config instead; no separate home-directory registry file is part
  of this design.
- **Bubble-up directory discovery** to find a project root from inside a
  repo — from the original issue. Rejected once `command` was established
  as project-root-only; the agent is never invoked from inside a bare repo
  in the first place, so there is nothing to bubble up from.
- **Per-repo `command`** — considered when it was noticed that `command`
  (e.g. `go run .`) is inherently language/repo-specific. Rejected: "Not
  worth deciding yet" and "command stays per-repo" options were both
  presented, but the user's answer ("spektacular is only executed at a
  project level") made the question moot — there is only ever one `command`
  for the whole project.
- **Viewer tool** (Mermaid rendering, inline GitHub PR comments, binary +
  daemon/server model) — from the original issue. User: "The viewer tool can
  be a separate spec, for now, we assume file accessible." Fully out of
  scope; assume all repos are filesystem-accessible for now (i.e. `provider:
  file` is sufficient; no remote/API-backed provider is required by this
  spec).
- **Local-vs-shared plans with explicit PR-based sharing, and
  dependent-plan invalidation workflow at end of implementation** — from the
  original issue. User: "For now, lets only have a single location for
  plans, specs, etc, we can deal with local later." Deferred entirely.
- **Token-efficiency strategy for large multi-repo plan context** — raised
  by the user as a real concern ("we probably should think about how we are
  more efficient on tokens") but explicitly parked rather than designed:
  "for now I think we can park that as a do later and have context project
  level."
- **A `repos.Set` registry/lookup abstraction** mirroring
  `knowledge.Set` (`internal/knowledge/set.go:70-95`) — considered as the
  natural-looking precedent (knowledge already does exactly this: one
  `FileStore` per configured source, switched by scope). Rejected as
  unnecessary extra abstraction once the user pointed out the `repos` config
  already contains everything needed to construct a `FileStore` directly,
  per file/per resolved repo, without a persistent registry object.
- **Enterprise-tier autonomous cross-repo execution** (agent resolves all
  dependencies, determines parallel vs. sequential work, executes without
  supervision) — from the original issue's "Next Steps." Not discussed
  further after the initial issue summary; not part of this spec's scope.
- **Git-repo plugin implementation** (the `git` provider itself, to sync a
  central context directory) — from the original issue, listed as a
  precursor to a centralized knowledge base. Referenced repeatedly as
  aspirational/future (comments in
  `.spektacular/plans/000014_spektacular_store/plan.md` and
  `.spektacular/specs/000020_context.md` already say "future backends" and
  "a future remote backend"), but not designed or scheduled here — this spec
  only requires `provider: file` to work end-to-end.

## Code grounded during discussion (verified, not assumed)

- `internal/config/config.go` — today's single `Config` struct holds
  `Command`, `Agent`, `SpecTriggerThreshold`, `Debug`, `Spec`, `Plan`,
  `Changelog`, `Knowledge` (lines ~111-120), each provider-backed field
  following a `Provider string` + `Config <FileXConfig>` shape.
  `KnowledgeConfig.Sources []SourceConfig` (93-103) is the only existing
  multi-instance, scope-tagged list — the direct precedent for the new
  `repos` list. `SourceConfig.Scope` (100) is validated only for
  non-emptiness/uniqueness, not against a fixed enum, and
  `DefaultKnowledgeScope = "project"` (43) is just a synthesized default.
- `internal/store/store.go` — `Store` interface (37-55):
  `Root/Read/Write/Delete/List/Exists/Search`. `FileStore` (59-152) is the
  only implementation, constructed via `NewFileStore(root, scope string)`
  (66-68), with all paths jailed to `root` via `abs()` (80-88). No provider
  besides `file` exists in code anywhere (confirmed by grep across the
  whole repo, excluding an unrelated vendored Dagger SDK).
- `internal/knowledge/set.go:70-95` (`NewSet`) — the pattern the new
  `repos` list should mirror: loops configured `Sources`, switches on
  `Provider`, builds one `FileStore` per source, resolves relative
  locations against the project root.
- `internal/project/init.go` (`Init`, lines 15-117) — today's single-pass
  init that unconditionally creates `.spektacular/`, writes `config.yaml`,
  and scaffolds spec/plan/knowledge directories with no project/repo
  distinction. This is the function that needs the `--project` fork.
- `cmd/root.go:185-227` — `configFilePath()` (185-190, hardcoded
  `cwd/.spektacular/config.yaml`, no upward search, no home-dir fallback),
  `loadConfig()` (196-205), and `projectRoot()` (217-227, returns `cwd`,
  documented as "spec, plan, and knowledge directories from the config are
  all resolved relative to this"). This is today's sole single-root
  resolution point that multi-repo file writes need to route around (via
  per-repo `FileStore` construction) rather than replace.
- `templates/steps/implement/08-update_repo_changelog.md:7` — today's
  explicit single-repo assumption: "Create or update CHANGELOG.md at the
  repo root (not inside .spektacular/ or the plan directory)."
- `templates/scaffold/plan.md` — has only a singular, unpopulated
  `<!-- Repository: -->` metadata placeholder; no structured
  affected-files/affected-repos manifest field exists today. Per-file
  detail is deliberately pushed to `context.md` already (plan.md explicitly
  forbids file:line references), which is why `context.md` was the natural
  place agreed on for recording per-file repo resolution (item 6) rather
  than inventing a new field in `plan.md`.
- `templates/agents/*.md`, `templates/skills/workflows/spek-*/SKILL.md`,
  `templates/skills/skill_*.md` — the full set of files that `init
  --project` installs and that repo-level `init` does not.

## User's exact phrasing for key requirements (preserved verbatim)

- "I think for a single repo you could have a project config and a repo
  config, there are two separate files"
- "I think we need to move most stuff from repo config (current config) to
  the project config. Repo config should define the location of the
  knowledge base but this should be single source then spectacular uses
  configured projects in project config and reads their knowledge config"
- "spektacular is only executed at a project level"
- "I think init with --project with repo being default, project should also
  create the agent config and skills, this is not needed at a repo level."
- "Repo still gets a minimal AGENTS.md pointing up to the project"
- "In the config a repo should be added to the project config using a
  provider approach used in changelogs, etc. This way we can leverage the
  same capabilities for reading and writing files"
- "It should have the provider, provider config, and metadata about the
  project, description, tags, etc"
- "The viewer tool can be a separate spec, for now, we assume file
  accessible"
- "For now, lets only have a single location for plans, specs, etc, we can
  deal with local later."
- "we need to think about the posibility of creating work across multiple
  projects, a single spec may create some code in one repo and update docs
  in another"
- "The agent should be able to update the code, so during plan phase, if
  for example a req is to update the documentation. The agent should be
  able to determine that the repo for that change is x and the files are y."
- "well, the file provider only really needs a root path and that is in the
  config"
- "I am thinking we have a spektacular ignore file that the file provider
  honors"
- ".spektacular_ignore would be rooted at the provider source and would
  follow the convention of gitignore"
