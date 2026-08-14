---
created_date: "2026-08-13"
status: completed
closed_date: "2026-08-13"
---

# Plan: 000039_project-level-capabilities

<!-- Metadata -->
<!-- Created: 2026-08-13T07:56:52Z -->
<!-- Commit: 3ccbbd2 -->
<!-- Branch: main -->
<!-- Repository: git@github.com:jumppad-labs/spektacular.git -->

## Overview

Spektacular becomes project-aware: configuration splits into a project-level file (identity, a registry of member repos, central spec/plan/changelog storage) and a minimal repo-level file, so one Spektacular project can plan and carry out work spanning multiple repositories — attributing each requirement to the repo it belongs to, materializing registered repos by cloning when needed, and leaving a provenance-stamped changelog entry in every repo the work touches. This removes today's one-directory-one-setup limit (and the silent default behavior outside an initialized project); teams working across related repositories — a code repo plus a docs repo, say — get one coherent spec-to-implementation workflow instead of manually coordinating separate Spektacular setups.

## Conventions

The project knowledge base's conventions category contains only its placeholder README — no real always-applied conventions exist yet. One legacy root-level `conventions.md` entry exists but is stale scaffold (it prescribes PEP 8 for what is a Go codebase), so it was deliberately dropped except for two generic points that match observed practice:

- **Write unit tests for all new functionality** (knowledge `conventions.md` §Testing) — every new surface here (config split, repo registry, git provider, ignore wrapper, namespaced changelog paths) follows the observed testify/require + t.TempDir conventions and the `// Criterion N:` traceability comments.
- **Update README.md for user-facing changes and document API changes in CHANGELOG.md** (knowledge `conventions.md` §Documentation) — this feature is a breaking configuration change and adds a `repo` command group; README's configuration section (already drifted: it omits the changelog section) and CHANGELOG.md must be updated as part of the work.

Conventions inferred from code rather than the knowledge base, applied because the plan touches their surfaces: store operations stay pure in-process Go (no external tools in the store layer, internal/store/search.go:36-40); agent-facing procedure lives in template/skill prose, not Go flags; defaults are synthesized at load, never baked into written config.yaml.

## Architecture & Design Decisions

**Options weighed.** Three coherent directions were compared for the greenfield surfaces (git access, gitignore matching, exclusion layering, config split):

- **Option A — Pure-Go maximalism**: adopt `go-git/v5` for clone and staleness checks and reuse its `plumbing/format/gitignore` package as the `.spektacular_ignore` matcher. *Pros*: no external binary, one dependency covers two gaps, stays inside the "no external runtime tool" grain established when ripgrep was removed (internal/store/search.go:36-40). *Cons*: a large transitive dependency tree lands on a codebase with five direct deps (go.mod:6-12), and — decisively — go-git does not inherit the user's credential helpers, SSH agent config, or `~/.gitconfig`, so cloning private repos fails exactly where real users need it to work. Effort: Medium.
- **Option B — Git binary for git work, small pure-Go matcher for ignore rules**: shell out to the user's `git` for clone and staleness (`git ls-remote` compared against the clone's HEAD — read-only, never fetch/pull), and take a minimal zero-dependency gitignore-matcher library for `.spektacular_ignore`. *Pros*: clones inherit the user's existing authentication and configuration verbatim; the external-tool objection is scoped — git is required only by the clone-based access method, and a user registering repos by remote URL has git by definition; gitignore semantics (negation, anchoring, `**`) come battle-tested rather than hand-rolled. *Cons*: introduces the codebase's first `os/exec` call, which must be sandboxed to the repo-provider package and error clearly when git is absent. Effort: Medium.
- **Option C — No new dependencies at all**: git binary plus a hand-rolled gitignore-subset matcher. *Pros*: zero go.mod growth, matches the hand-rolled-search house style. *Cons*: gitignore semantics are easy to get subtly wrong, and a subtly wrong matcher silently hides or exposes files during planning — a correctness risk with no offsetting benefit over a tiny library. Effort: Medium-High.

**Chosen direction: Option B.** Authentication is the deciding factor for git access — a materialization feature that cannot clone private repositories is not shippable — and correctness is the deciding factor for the matcher. The prior "no external tools" decision (the ripgrep removal) is honoured in spirit: search and all store operations remain pure in-process Go; only the git repo provider, an inherently git-shaped operation the user opts into per-repo, shells out. Full comparison evidence is in `research.md#alternatives-considered-and-rejected`.

**Configuration splits into two files with one loader seam.** Project configuration stays at `.spektacular/config.yaml` and gains required `name` (validated slug/filesystem-safe at load), optional `source`, and a `repos` list; it keeps `command`, `agent`, `spec_trigger_threshold`, `debug`, and the central `spec`/`plan`/`changelog` sections. Repo configuration is a new, separate `.spektacular/repo.yaml` holding only repo-scoped concerns — `knowledge.sources` and the repo's `changelog` provider — with no pointer back to any project; a colocated solo-repo project simply has both files in the same `.spektacular/` directory. Knowledge is declared at the level that owns it: `repo.yaml` is the single knowledge authority for its repo (the repo's own store by default, plus any additional sources that repo declares), while project config retains an optional `knowledge.sources` list for sources owned by the project rather than any single repo (e.g. team or global scopes); the effective knowledge set aggregates both levels, and nothing is duplicated because every source is declared exactly once. All loading continues through the single existing seam (`configFilePath`/`loadConfig`, cmd/root.go:185-205): the silent fall-back to `NewDefault()` when no config exists is replaced by an explicit "no project configured here" error with a next-action pointing at init, exempting only bootstrap commands (`init`, `version`, `skill`); there is still no parent-directory search. Repo configs are loaded on demand when a registered repo is touched, validated at that moment, with a structured repair offer (via the existing `output.NewError(...).WithNextAction(...)` shape) when missing or broken.

**The repos registry clones the knowledge-sources pattern, and repos resolve to plain directories.** The `repos` list mirrors `KnowledgeConfig.Sources` structurally (internal/config/config.go:93-108): each entry carries provider-agnostic identity and metadata (`name` required and slug-safe with uniqueness enforced like scope uniqueness at config.go:255-259; `address`, `local`, description, tags, role, dependencies, deployment type) beside a `provider` block. One provider ships — `git` — and it owns resolution: use `local` when set and present; otherwise clone `address` into `.spektacular/repos/<repo-name>/` only when absent, then warn (never act) if the clone is behind its remote. Dispatch is a literal switch with fail-fast unknown-provider errors, modeled on `knowledge.NewSet` (internal/knowledge/set.go:70-95), with materialization running *before* the reachability check so the existing fail-fast contract is preserved rather than weakened. Resolution yields a local directory path and nothing more: per the spec's mediation boundary, Spektacular hands agents the resolved root, metadata, and exclusion rules, and agents use their native tools against that path — providers never proxy per-file access. Project init ensures `.spektacular/repos/` is gitignored (the embedded `templates/.spektacular/.gitignore` gains the entry) so clones can never become tracked content or gitlinks. `repo add` registers an entry and creates the target repo's minimal footprint (repo.yaml + knowledge storage) idempotently; `repo list` reports the registry with resolved paths, following the `--data`/`--schema` JSON command shape (cmd/knowledge.go:15-67, 236-250). `repo add` performs the first CLI-driven config write via the existing full-marshal `ToYAMLFile` (config.go:292-301), accepting comment loss in config.yaml.

**Exclusion is a store wrapper; changelogs namespace by project and derive per repo.** `.spektacular_ignore` (gitignore syntax, rooted at the source it governs) is honoured by an ignore-aware wrapper implementing `store.Store` around `FileStore`: it filters only `List` and `Search`, leaving `Read`/`Write`/`Exists`/`Delete` untouched so a directly named path is never blocked. This keeps `FileStore` domain-agnostic — the documented boundary (internal/store/search.go:41-43) and the layering precedent from the knowledge-tiers work both survive — while giving every consumer (knowledge sets, artifact scans, store-file listing, future repo-rooted stores) one composable hook instead of teaching three walkers separately. Every changelog store adopts the namespaced layout `changelog/<project-name>/<id>_<slug>.md`; retrieval stays filename-addressed (no manifest — that rejection is load-bearing), and `validateIDPrefix` (cmd/storefile.go:66-80) learns to check the segment after the project folder. After the central entry is written, the implement workflow derives one entry per affected repo — filtered to that repo's changes, carrying project name, project source, and spec/plan identifiers as structured front matter (extending `internal/metadata` to carry these additional keys) plus a human-readable reference line — and writes it through a store rooted at that repo via its own configured changelog provider, exactly the foreign-root store construction already proven at internal/knowledge/set.go:88. Cross-repo planning and implementation are delivered almost entirely through template and skill prose: the plan discovery/architecture step templates instruct the agent to consult `repo list` and attribute requirements to repos, the implement changelog steps gain the derivation procedure, and a new library skill documents driving the repo CLI — following the standing rule that agent-facing procedure lives in prose, not Go.

## Component Breakdown

- **Project configuration (changed)** — the existing config type, still the single value handed to every command. Gains the project identity (required slug-safe `name`, validated at load; optional `source`) and the `repos` registry list; keeps `knowledge.sources` as an optional list of project-owned sources (no longer seeded by default — each repo's own store is declared in its repo config instead). Owns validation of the registry: per-entry required slug-safe unique `name`, at-least-one-of `address`/`local`, provider supported. Remains the only thing the config loader returns, so downstream consumers are untouched.

- **Repo configuration (new)** — a second, independent config type and file living in each member repo, holding only repo-scoped concerns: knowledge sources and the repo's changelog provider. Carries no pointer to any project. Owns its own defaults synthesis and validation, mirroring the project config's style. Loaded on demand by the repo registry whenever a registered repo is touched, never at startup.

- **Config loader / project gate (changed)** — the single seam through which every command obtains project configuration. Missing project config becomes an explicit "no project configured here" error carrying a next-action pointing at init, replacing the current silent fall-back to defaults; a small allowlist of bootstrap commands (init, version, skill) remains runnable without a project. Still strictly current-directory — no parent search.

- **Repo registry & resolver (new)** — the domain component (modeled on the knowledge set) that turns registry entries into usable repos. Owns provider dispatch (fail-fast on unknown providers), the git provider's resolution order (use `local` when present, otherwise clone `address` into the project's working folder only when absent), the best-effort staleness warning, and validation-on-use of each repo's footprint with a structured repair offer instead of a hard failure. Exposes resolved entries — name, resolved local root, metadata, exclusion rules — to the CLI and to changelog derivation. Hands out paths, never file contents.

- **Git executor (new)** — a thin, provider-internal wrapper around the user's git binary owning exactly two operations: clone, and a read-only remote-head query for the staleness check. Owns the "git not installed" error. The only place in the codebase that shells out; nothing outside the repo provider may reach it. Invocation is platform-neutral by construction: the binary is resolved from PATH on every OS (`git.exe` included on Windows), the argument set is identical across platforms, output parsing tolerates Windows line endings, and authentication continues to delegate to whatever credential machinery the platform's git provides (credential manager on Windows, keychain on macOS, agent/helpers on Linux).

- **Ignore matcher & ignore-aware store wrapper (new)** — loads a source-rooted `.spektacular_ignore` (gitignore semantics, via a small pure-Go matcher) and wraps any store with a filter applied only to listing and search; direct read/write/exists/delete pass through untouched, so an explicitly named path is never blocked. Composes with stores rooted anywhere — project storage locations and registered repos alike. Store construction sites opt in by wrapping; the underlying file store stays domain-agnostic.

- **Repo CLI command group (new)** — `repo add` (registers an entry in project config and creates or repairs the target repo's minimal footprint in one idempotent operation — the first CLI-driven config write) and `repo list` (reports registered repos with metadata and resolved paths). Follows the established JSON `--data`/`--schema` command shape. The registry & resolver does the real work; these commands are thin handlers.

- **Changelog store & derived-entry writing (changed)** — every changelog store adopts the project-namespaced layout (a project-named folder containing per-feature files); the shared store-file command factory's ID-prefix validation learns the namespaced path shape, and artifact listing follows the folder level down. Derived per-repo entries are written through a store rooted at each affected repo via that repo's configured changelog provider — reusing the existing foreign-root store construction — scoped to that repo's changes.

- **Front-matter metadata (changed)** — the metadata component gains the structured fields derived entries must carry (project name, project source, spec and plan identifiers) while preserving its existing merge invariants for dates and status.

- **Project init & cascade (changed)** — the single init command becomes explicitly project-level: writes the project identity into config, seeds the colocated repo's own repo configuration (a solo repo is a project of one), ensures the clone working folder is gitignored, and cascades over all registered repos, creating or repairing each one's minimal footprint using the same top-up mechanisms that make init idempotent today. Still the only initialization entry point — no repo-level init exists.

- **Knowledge aggregation (changed)** — the knowledge set aggregates two levels: the sources declared in each registered repo's configuration (each repo's own store plus any additional sources it declares — the repo config is the single knowledge authority for its repo), followed by any project-owned sources declared in project config. Repo-declared sources take precedence (registry order first, project sources last); for a solo project this is its colocated repo plus project sources — behavior identical to today. Consumes resolved repos from the registry rather than raw config.

- **Workflow step templates & skills (changed/new)** — the plan workflow's discovery and architecture templates gain cross-repo requirement attribution (consult the repo registry, search per repo, record attribution in the plan's existing context document); the implement workflow's changelog steps gain the central-then-derived writing procedure; a new library skill documents driving the repo CLI, and the existing workflow skills are updated where they describe configuration or changelog behavior. Agent-facing procedure stays in prose — these templates drive the components above through the CLI.

## Data Structures & Interfaces

**Project configuration (changed type).** The existing config struct gains identity and the registry; knowledge stays as an optional project-owned sources list:

```go
type Config struct {
    Name      string          // required, slug/filesystem-safe — used as the changelog namespace folder
    Source    string          // optional git remote URL for referencing the project externally
    // Command, Agent, SpecTriggerThreshold, Debug, Spec, Plan, Changelog — unchanged
    Knowledge KnowledgeConfig // optional project-owned sources (team/global etc.); empty by default — repo stores live in repo configs
    Repos     []RepoEntry     // the member-repo registry
}

type RepoEntry struct {
    Name         string   // required, slug-safe, unique within the project
    Address      string   // remote URL; provenance metadata when Local is set
    Local        string   // local path; wins over Address when both set
    Description  string   // what the repo does
    Role         string
    Tags         []string
    Dependencies []string
    Deployment   string
    Provider     string        // "git" — the only shipped provider
    Config       GitRepoConfig // provider-specific block, empty in this release; reserved for extension
}
```

At least one of `Address`/`Local` is required per entry. Identity and metadata are deliberately provider-agnostic siblings of the provider block, mirroring how knowledge sources keep `scope` outside their provider config.

**Repo configuration (new type, new file).** An independent type serialized to the repo's own config file, with no project back-pointer:

```go
type RepoConfig struct {
    Knowledge KnowledgeConfig // the repo's own store (default) plus any additional sources this repo declares
    Changelog ChangelogConfig // the repo's changelog provider for derived entries
}
```

It reuses the existing `KnowledgeConfig` and `ChangelogConfig` types unchanged — the split moves sections between files rather than inventing parallel shapes.

**Repo registry & resolution (new contracts).** The registry component consumes `[]RepoEntry` plus the project root and yields resolved repos; this is the sole contract other components (CLI, changelog derivation, knowledge aggregation) depend on:

```go
type ResolvedRepo struct {
    Name         string // registry name
    Root         string // absolute local directory — the only access handle handed out
    Entry        RepoEntry
    Materialized bool   // true when the root is a project-managed clone
    StaleNote    string // non-empty ⇒ warn-only behind-remote notice
}

Resolve(name string) (ResolvedRepo, error)   // validates footprint on use; repair offered via structured error
ResolveAll() ([]ResolvedRepo, error)
```

**Git executor (new, provider-internal interface).** Narrow by design so it can be faked in tests and never grows into a git façade:

```go
type GitRunner interface {
    Clone(url, dir string) error              // plain clone, never submodule
    LocalHead(dir string) (string, error)     // read-only
    RemoteHead(url string) (string, error)    // ls-remote-style, read-only, best-effort
}
```

**Ignore-aware store (new wrapper, unchanged interface).** The `store.Store` interface itself does not change — the wrapper implements it around any inner store:

```go
type IgnoreMatcher interface { Ignored(relPath string, isDir bool) bool }

NewIgnoreStore(inner store.Store, m IgnoreMatcher) store.Store
// List and Search filter through the matcher; Read/Write/Exists/Delete pass through untouched.
```

The matcher is built from the `.spektacular_ignore` file at the wrapped store's root; a missing file yields a pass-everything matcher.

**Front-matter metadata (changed type).** The metadata struct gains the structured provenance fields derived changelog entries carry; existing date/status merge invariants are preserved:

```go
type Metadata struct {
    // CreatedDate, Status, ClosedDate — unchanged
    Project       string // project name (derived entries)
    ProjectSource string // project source URL, when set
    Spec          string // spec identifier
    Plan          string // plan identifier
}
```

**CLI JSON contracts (new/changed).** Following the established `--data`/`--schema` envelope:

- `repo add --data '{"name": "...", "address": "...", "local": "...", "description": "...", "role": "...", "tags": [...], "dependencies": [...], "deployment": "..."}'` → `{registered, footprint: "created"|"repaired"|"unchanged"}`. Idempotent; writes project config and the target repo's footprint in one operation.
- `repo list` → `{repos: [{name, address, local, root, provider, description, role, tags, dependencies, deployment, stale_note?}]}` — the projection agents consume for cross-repo attribution.
- `changelog file write <project-name>/<name>.md --from <path> [--repo <repo-name>]` — without `--repo`, writes the central store; with it, routes through the named repo's configured changelog provider rooted at that repo. Path validation accepts the project-folder prefix and checks the ID on the filename segment.

**Serialization boundaries.** Project config and repo config are two separate YAML files (colocated solo projects hold both in one directory); `.spektacular_ignore` is a plain gitignore-syntax text file rooted at the source it governs; every changelog store organizes entries as `changelog/<project-name>/<id>_<slug>.md`. The workflow engine's runtime config gains no repo-registry fields; the plan workflow's discovery and architecture instructions are rendered with a repo-roster template section — each registered repo's identity metadata (name, description, role, tags, deployment), read from config at render time — while resolved paths, materialization state, and staleness always come from the `repo` CLI commands.

## Implementation Detail

**A new `repo` domain package, shaped like the knowledge set.** The registry & resolver lands as a new internal package that deliberately reads as a sibling of the knowledge set: construct from config + project root, dispatch providers in a literal switch that fails fast on unknown names, expose a small value-struct projection for CLI output. A developer who has read the knowledge set will recognize every move. Provider resolution is the one genuinely new behavior: the git provider materializes (clone-if-absent) *before* applying the existing reachability contract, so "unreachable source ⇒ fail fast, never partial results" still holds once resolution completes. Staleness never fails resolution — it decorates the resolved repo with a warn-only note.

**The first subprocess, quarantined.** The git executor introduces the codebase's first `os/exec` usage, and the structure makes that boundary legible: a narrow three-method interface, implemented in one file, constructed only by the git repo provider, with a fake used everywhere in tests. No other package may import it; the store layer and search remain pure in-process Go. "git is not installed" surfaces as a structured error with a next-action, only ever triggered on the clone/staleness paths.

**Config becomes two types in one package, one loader, one gate.** The config package gains a second top-level type (repo config) beside the project config, each with its own file, defaults synthesis, and validation, deliberately sharing section types (knowledge sources, changelog) rather than duplicating shapes. The command layer's single loader becomes the project gate: it now distinguishes "no project here" (explicit, actionable error) from "config present but invalid" (validation error), and a small named allowlist of bootstrap commands bypasses the gate. This is the plan's one behavioral change that touches every command, implemented in exactly one place; nothing downstream changes type.

**Decorator over the store, not a smarter store.** Exclusion arrives as a classic decorator: an ignore-aware wrapper satisfying the existing store interface, filtering only listing and search. The file store keeps its documented "walks everything, excludes nothing" contract; construction sites choose to wrap. This extends the established layering (the store stays domain-agnostic; policy lives above it) and means repo-rooted stores, knowledge stores, and the central spec/plan/changelog stores all gain exclusion by composition, not by modification.

**Changelog paths gain one folder level; writing gains one routing flag.** The namespaced layout is a path-construction change plus a widened ID-prefix validation (the ID check moves from the leading segment to the filename segment under a project folder) — the flat filename-as-identifier retrieval model is preserved beneath a project namespace. Derived per-repo entries reuse the entire existing staged-write pipeline (stage, `--from`, frontmatter strip, metadata merge, store write); the only new machinery is the `--repo` routing that swaps in a store rooted at the resolved member repo using that repo's own changelog config. Metadata grows four provenance fields with unchanged merge semantics.

**Init cascade reuses init's own idempotency mechanisms.** Project init extends the existing top-up patterns (create-if-missing directories, write-config-only-if-absent, overwrite managed files) rather than inventing a repair framework: the cascade iterates registered repos and applies the same minimal-footprint routine `repo add` uses, so "create" and "repair" are literally the same code path run again. The colocated repo is seeded the same way, making a solo-repo project a true project-of-one rather than a special case. Init remains the only writer of agent guidance; member-repo footprints never receive skills, guidance, or version files.

**Agent-facing behavior ships as prose, not plumbing.** Cross-repo attribution during planning, the central-then-derived changelog procedure during implementation, and repo lifecycle management are all delivered as template and skill text driving the new CLI commands — following the house pattern that workflow steps instruct agents to call commands rather than receiving pre-assembled context. The one deliberate exception is a render-time repo roster: the plan workflow's discovery and architecture instructions are rendered with the registered repos' identity metadata embedded (read from config at render time), so knowing which codebases the project spans never depends on the agent remembering to run a command — `repo list` remains the source for the dynamic half (resolved paths, materialization, staleness). Beyond that one variable there are no workflow-engine changes and no new FSM steps; the plan and implement workflows keep their existing shapes. The instruction-surface regression tests extend naturally to police the new prose.

## Dependencies

**External (new)**

- **`git` binary on PATH (runtime, conditional)** — clone-if-absent materialization and the read-only staleness check. Required only when a registered repo must be resolved from its `address` or checked against its remote; local-only setups never invoke it. Absence surfaces as a structured error with a next-action.
- **A minimal pure-Go gitignore matcher library (e.g. `github.com/sabhiram/go-gitignore`, MIT)** — gitignore-semantics pattern matching for `.spektacular_ignore`. Zero transitive dependencies; exact library to be confirmed at implementation time against negation/anchoring/`**` correctness. No changes to it needed.

**External (existing, unchanged)**

- **`gopkg.in/yaml.v3`** — serializes both new config files; also used by the metadata front-matter extension. No changes.
- **`spf13/cobra`** — hosts the new `repo` command group via the established registration pattern. No changes.
- **`cbroglie/mustache`, `looplab/fsm`, `stretchr/testify`** — template rendering, workflow FSM, and test assertions, all used as-is; the workflow engine itself is deliberately untouched.

**Internal packages this work depends on (and whether they change)**

- **`internal/config`** — changes: gains project identity + repos registry, keeps knowledge as an optional project-owned sources list (default seeding moves to the repo config), gains the repo config type. The center of the breaking change.
- **`internal/store`** — depended on, interface unchanged; gains the ignore-aware wrapper alongside `FileStore`.
- **`internal/knowledge`** — changes: set construction sources from registered repos' configs instead of project config; also the structural template the new repo package copies.
- **`internal/metadata`** — changes: four new provenance fields with existing merge invariants preserved.
- **`internal/project` (init)** — changes: project identity, colocated repo seeding, `.gitignore` coverage of the clone folder, cascade over registered repos.
- **`internal/agent` + `templates/`** — changes: updated workflow skills and step templates, new repo-management library skill; managed-section and skill-install mechanisms reused as-is.
- **`cmd`** — changes: project gate in the single config loader, new `repo` command group, `--repo` routing and namespaced path validation in the store-file factory.

**Planning dependencies (prior work)**

- **Spec 000039_project-level-capabilities** — the upstream spec this plan implements; already completed, locally modified working-tree version is authoritative.
- **000020_context / 000028 knowledge-sources pattern** — already landed; provides the provider-list config shape and set-construction pattern this feature clones. The provider-keyed config decoding those plans deferred "until a second provider lands" is triggered by this work.
- **000033 changelog store** — already landed; provides the changelog store, command factory, and metadata pipeline the namespacing and derivation build on.
- **No unlanded prerequisites** — nothing must land or change before this plan starts; the breaking configuration change means existing setups re-initialize, which the spec explicitly authorizes.

## Testing Approach

**Test kinds.** Four layers, all inside the project's existing conventions (testify `require`, `t.TempDir()`/`t.Chdir()` fixtures, one-behaviour-one-function with shared helpers, `// Criterion N:` traceability comments, cobra flag-reset hygiene in command tests):

- **Unit tests** carry most of the weight: config parsing/validation for both new file shapes (required slug-safe project name, repo-entry uniqueness and at-least-one-of address/local, unknown-provider rejection, project-level knowledge list validated but no longer seeded by default), registry resolution against a fake git runner (local-wins precedence, clone-only-when-absent, warn-only staleness, footprint validation with repair offer), the ignore matcher and store wrapper (filtered listing/search, untouched direct access, missing ignore file passes everything), metadata's new provenance fields with unchanged merge invariants, and namespaced changelog path validation.
- **CLI-level tests** drive commands end-to-end through the root command against temp-dir projects, as the existing command tests do: `repo add` idempotency (fresh, re-run, already-initialized-by-another-project), `repo list` projection, the project gate (explicit error with next-action outside a project; bootstrap commands exempt), init cascade create-and-repair, and `changelog file write` both central-namespaced and `--repo`-routed.
- **Real-git integration tests** for the thin executor only, using local `file://` remotes built in temp dirs — clone materialization into the working folder, gitignore coverage (no tracked or gitlinked entries after materialization), and behind-remote detection. These are the only tests touching a real subprocess; everything above them uses the fake runner.
- **Template/prose regression tests** in the established render-and-assert style: the updated plan-discovery, architecture, and implement-changelog step templates contain the cross-repo instructions (and don't regress existing anchors), the new repo-management skill resolves and lists, and the instruction-surface tests keep policing forbidden command shapes.

**Heaviest coverage and why.** The registry/resolver and the two config types get the most tests — they are the new load-bearing trust boundary (every command flows through the gate; every cross-repo operation flows through resolution). Second is the changelog derivation path, because its correctness is what makes multi-project repos auditable (the namespace collision guarantees). The ignore wrapper is third — small surface, but silent wrongness would invisibly skew planning research.

**Load-bearing assertions in plain language.** A project without config fails loudly and tells you how to init; adding the same repo twice (or from two projects) changes nothing and breaks nothing; a repo materialized by clone can never leak into the project's git history; two projects writing changelogs into one repo cannot collide and every derived entry names its project, source, and spec/plan; an excluded path never appears in listings or search but is always readable when named directly; a broken repo footprint is reported with a repair offer, never a hard failure or silent continuation.

**Success metrics from the spec, each carried here.**

1. *Multi-repo project goes spec→plan→implementation in one Spektacular project with no manual coordination* — **Manual — captured in the implementation test plan** (a live end-to-end agent workflow across two repos; may additionally be exercised by the harbor suite, but the metric itself is a workflow-level observation).
2. *Cross-repo plans attribute requirements to the right repo without user correction* — **Manual — captured in the implementation test plan** (agent-judgement quality; the automatable slice — that the discovery/architecture templates instruct attribution and that `repo list` supplies the metadata — is covered by the template regression tests).
3. *A repo's changelog alone unambiguously identifies project and spec/plan for every entry* — **Behavioural test**: derived entries are asserted to carry project name, project source, and spec/plan identifiers as structured front matter plus the human-readable reference line, under the project-named folder, including the two-projects-one-repo collision case.
4. *Cross-repo searches stay useful — exclusions keep noise out* — **Behavioural test**: ignore-wrapper tests assert excluded paths (e.g. build-artifact and dependency directories) are absent from Spektacular listing/search results while direct access still works.
5. *Setting up a new project including repo registration takes minutes* — **Manual — captured in the implementation test plan** (wall-clock observation; the supporting guarantees — idempotent init, one-command `repo add` — are behaviourally tested).

**Deliberate gaps.** No network-touching tests — all git integration uses local `file://` remotes, so remote-auth behavior (the reason the git binary was chosen) is asserted only indirectly by delegating to the user's git; documented as a known limitation rather than mocked. No new harbor scenario is authored inside this plan — the existing plan-workflow harbor suite must stay green, and a dedicated cross-repo harbor scenario is left to follow-up unless a phase proves cheap to add one. No performance tests — token-efficiency for large multi-repo projects is an explicit spec non-goal.

## Milestones & Phases

### Milestone 1: Projects have an identity and an explicit boundary

**What changes:** Spektacular stops treating "wherever it runs" as an implicit project. Configuration splits into a project-level file — now carrying a required project name and optional source URL — and a separate, minimal repo-level file holding only the repo's own concerns (knowledge storage, changelog provider), with nothing duplicated between them and no pointer from repo back to project. Running Spektacular where no project is configured produces a clear error telling the user to initialize, instead of silently proceeding with defaults; initializing a project sets up its identity, full agent guidance, and storage, and seeds the colocated repo's own config so a solo repo is simply a project of one. This is a breaking change: existing setups re-initialize.

**Validation point:** A freshly initialized project contains both config files with only their own level's settings; every project-operating command run outside a project returns the explicit error with a pointer to init; init re-run produces no changes; the full existing test suite passes against the new configuration shape.

#### - [x] Phase 1.1: Config splits into project and repo types

Split the single configuration into two independent shapes: the project config gains a required, filesystem-safe project name and an optional source URL, and its knowledge section becomes an optional list of project-owned sources (empty by default); a new repo config type holds repo-scoped concerns — the repo's own knowledge store plus any additional sources it declares, and the repo's changelog provider — in its own file, with no pointer back to any project. Both shapes keep the existing defaults-synthesized-at-load philosophy so written files stay minimal.

*Technical detail:* [context.md#phase-11](./context.md#phase-11-config-splits-into-project-and-repo-types)

**Acceptance criteria**:
- [x] A project config file with a missing or non-slug-safe `name` fails validation with an error naming the field
- [x] A repo config file parses and validates independently of any project config, and contains only knowledge and changelog settings
- [x] A project config may declare project-owned knowledge sources, and a repo config declares its repo's sources — each list validated independently at its own level
- [x] Existing spec/plan/changelog configuration behavior is unchanged

#### - [x] Phase 1.2: Running without a project fails explicitly

Replace the silent fall-back to default configuration with an explicit "no project configured here" error that tells the user how to initialize, issued from the single config-loading seam so every command gets the behavior at once. Bootstrap commands (init, version, skill) remain runnable without a project, and there is still no parent-directory search.

*Technical detail:* [context.md#phase-12](./context.md#phase-12-running-without-a-project-fails-explicitly)

**Acceptance criteria**:
- [x] Any project-operating command run in a directory with no project config returns an explicit error naming the init command as the next action
- [x] `init`, `version`, and `skill` still work in an uninitialized directory
- [x] Running from a subdirectory of a project does not silently create or assume a new project there

#### - [x] Phase 1.3: Init establishes project identity and the colocated repo

Project initialization writes the project's name (derived from the directory name unless given explicitly), creates the separate repo config for the colocated repo so a solo repo is a project of one, and knowledge commands aggregate the colocated repo's declared sources with any project-owned sources from the project config. Re-running init remains a no-change operation on an up-to-date project.

*Technical detail:* [context.md#phase-13](./context.md#phase-13-init-establishes-project-identity-and-the-colocated-repo)

**Acceptance criteria**:
- [x] After init, the project directory holds both config files, each containing only its own level's settings
- [x] Knowledge search/read/write behave exactly as before the split for a solo-repo project
- [x] A second init run produces no changes
- [x] The recorded stale-version remediation still composes the correct init command

### Milestone 2: A project can register, materialize, and manage its member repos

**What changes:** The project's configuration gains a registry of member repos — each with a name, an address and/or local path, and descriptive metadata — managed by a new `repo add` command that registers the repo and creates its minimal footprint in one idempotent operation, and a `repo list` command that reports the registry with resolved locations. A repo registered only by address is materialized automatically by cloning into a working folder inside the project, protected from ever entering the project's git history; an already-cloned repo that has fallen behind its remote produces a warning, and Spektacular never fetches or pulls on its own. Touching a registered repo whose footprint is missing or broken surfaces the problem with an offer to repair rather than a hard failure, and re-running project init repairs all registered repos' footprints. Knowledge aggregation now flows through the registry, so a repo's knowledge travels with the repo.

**Validation point:** A repo registered by local path and one registered by address alone both resolve to usable directories; the clone lands in the project's working folder with a clean `git status` in the project; re-running `repo add` (including from a second project) changes nothing; deleting a repo's config file and touching it produces the repair offer; `repo list` reflects the registry.

#### - [x] Phase 2.1: The project config gains a repo registry

Add the repos list to project configuration: each entry carries a required slug-safe unique name, an address and/or local path, descriptive metadata (description, role, tags, dependencies, deployment), and a provider block modeled on the knowledge-sources pattern. Project init seeds the colocated repo as the first registry entry so every project is uniformly a collection of repos.

*Technical detail:* [context.md#phase-21](./context.md#phase-21-the-project-config-gains-a-repo-registry)

**Acceptance criteria**:
- [x] A registry entry with a duplicate or non-slug-safe name, or with neither address nor local, fails validation with an error naming the entry
- [x] An unknown repo provider is rejected at load time
- [x] A freshly initialized project's config registers its colocated repo

#### - [x] Phase 2.2: Repos resolve to local directories, cloning when absent

Introduce the repo registry component and the git provider: a registered repo resolves to its local path when present, and is otherwise materialized by cloning its address into the project's working folder — only when absent, never fetched or pulled. A previously cloned repo that is behind its remote yields a warn-only notice. The working folder is covered by the project's gitignore so clones can never enter the project's history, and touching a repo whose footprint is missing or broken produces a repair offer rather than a hard failure. Git is invoked through a narrow, quarantined executor — the codebase's first and only subprocess — with a clear error when git is not installed.

*Technical detail:* [context.md#phase-22](./context.md#phase-22-repos-resolve-to-local-directories-cloning-when-absent)

**Acceptance criteria**:
- [x] A repo with a usable local path resolves to it without touching git, even when an address is also set
- [x] A repo with only an address is cloned into the project's working folder on first use and reused without cloning afterwards
- [x] After materialization, the project's git status shows no new tracked or gitlinked entries
- [x] A clone behind its remote produces a warning while resolution still succeeds; an unreachable remote degrades to a notice
- [x] Resolving a repo with a missing or invalid repo config reports the problem with an offer to repair

#### - [x] Phase 2.3: The CLI manages the repo lifecycle

Add the `repo add` and `repo list` commands. `repo add` registers a repo in the project config and creates its minimal footprint (repo config plus knowledge storage — nothing else) in one idempotent operation, performing the codebase's first CLI-driven config write; adding a repo already initialized by another project registers it without disturbing its footprint. `repo list` reports the registry with metadata and resolved locations. Removal stays a manual config edit by design.

*Technical detail:* [context.md#phase-23](./context.md#phase-23-the-cli-manages-the-repo-lifecycle)

**Acceptance criteria**:
- [x] After `repo add`, the project config reflects the entry's name, location, and metadata, and the target repo contains exactly its config file and knowledge storage
- [x] Re-running `repo add` for the same repo — from this or another project — changes nothing in the target repo
- [x] `repo list` reports every registered repo with its metadata and resolved path

#### - [x] Phase 2.4: Init cascades over members, and knowledge flows through the registry

Re-running project init now also walks the registered repos, creating or repairing each one's minimal footprint with the same routine `repo add` uses. Knowledge aggregation switches from the colocated shortcut to the registry: every registered repo's declared sources participate in search, listing, and always-applied loading — followed by any project-owned sources from the project config — so a repo's knowledge travels with the repo into every project that registers it, while project-level sources stay declared once at project level.

*Technical detail:* [context.md#phase-24](./context.md#phase-24-init-cascades-over-members-and-knowledge-flows-through-the-registry)

**Acceptance criteria**:
- [x] Breaking a member repo's footprint and re-running init repairs it; a healthy project re-init remains a no-change operation
- [x] Knowledge search results include entries from every registered repo's sources and from project-owned sources declared in the project config, each attributed to its source
- [x] Solo-repo projects see identical knowledge behavior before and after this phase

### Milestone 3: Changelogs identify their project, and work leaves a record in every repo it touches

**What changes:** Every changelog store — the project's central one and each repo's own — organizes entries in a folder named after the project, so multiple projects writing to the same repo can never collide and a reader of any repo's changelog can tell exactly which project and which spec/plan produced each entry. When implemented work touches registered repos, the implement workflow writes the central record as before and then derives one entry per affected repo — scoped to that repo's changes, carrying the project's name, source, and the spec/plan identifiers as structured data plus a readable reference line — through that repo's own configured changelog provider.

**Validation point:** After an implement run touching two repos, the central entry sits under the project's folder and each repo holds a derived entry under the same project-named folder listing only its own changes with the full provenance front matter; a second project writing to one of those repos lands in its own folder with no collision.

#### - [x] Phase 3.1: Changelog stores namespace by project

Every changelog store organizes entries under a folder named after the project, so a repo touched by multiple projects accumulates entries in separate per-project folders with no filename collision, and the project's own central store gains a single folder bearing its name. Front-matter metadata learns the provenance fields derived entries carry: project name, project source, and spec/plan identifiers. Retrieval stays filename-addressed.

*Technical detail:* [context.md#phase-31](./context.md#phase-31-changelog-stores-namespace-by-project)

**Acceptance criteria**:
- [x] A changelog entry written after an implement run lands under the project-named folder in the central store, and reading it back by name works unchanged
- [x] Changelog artifacts remain visible to artifact listing under the new layout
- [x] Front matter round-trips the new provenance fields while preserving existing date/status merge behavior

#### - [x] Phase 3.2: Implemented work derives per-repo changelog entries

Extend the changelog write command with repo routing so an entry can be written through a registered repo's own configured changelog provider, stamped automatically with the project's name, source, and the spec/plan identifiers plus a human-readable reference line. The implement workflow's changelog steps gain the derivation procedure: after the central record is written, one derived entry per affected repo, scoped to only that repo's changes.

*Technical detail:* [context.md#phase-32](./context.md#phase-32-implemented-work-derives-per-repo-changelog-entries)

**Acceptance criteria**:
- [x] Writing a changelog entry with repo routing places it under the project-named folder in that repo's changelog store, carrying the full provenance front matter and reference line
- [x] Two projects writing to the same repo produce entries in separate project-named folders with no collision
- [x] The implement workflow's instructions produce a central entry plus one derived entry per affected repo, each listing only that repo's changes

### Milestone 4: Agents plan and carry out work across repos, and searches stay clean

**What changes:** The planning workflow learns to look across all registered repos — their metadata and contents — to work out which repo (and which files) each requirement belongs to, and the implementation workflow carries each part of the work out in the repo it belongs to, so a feature spanning code in one repo and documentation in another proceeds as one coherent unit. Any repo (and the project's own storage) can declare exclusions in a gitignore-style `.spektacular_ignore` file that keeps noise like build artifacts and dependency directories out of Spektacular's listings and searches — without ever blocking direct access to an explicitly named path. A new skill documents driving the repo-management commands, and the existing planning and implementation skills gain their cross-repo behavior.

**Validation point:** In a two-repo project, a documentation-shaped requirement is attributed to the docs repo in the resulting plan and the implement run changes both repos; a path listed in `.spektacular_ignore` is absent from Spektacular search and listing results but reads fine when named directly; the existing harbor workflow suite stays green.

#### - [x] Phase 4.1: Sources can exclude paths from listing and search

Introduce the gitignore-style `.spektacular_ignore` file, rooted at the source it governs, honoured by an ignore-aware wrapper around the file store that filters only listing and search — a directly named path is never blocked, and agents' native tools are unaffected. Project storage locations and registered repos all gain exclusion by the same composition.

*Technical detail:* [context.md#phase-41](./context.md#phase-41-sources-can-exclude-paths-from-listing-and-search)

**Acceptance criteria**:
- [x] A path matching an exclusion pattern is absent from Spektacular's listings and search results for that source
- [x] The same path is still readable and writable when named directly by its exact path
- [x] A source with no ignore file behaves exactly as before
- [x] Standard gitignore semantics (negation, directory patterns, anchoring) behave as a git user expects

#### - [x] Phase 4.2: Workflows and skills go cross-repo

Teach the planning workflow to attribute requirements across registered repos — consulting the registry's metadata, searching within each repo, and recording the repo-and-files resolution in the plan's context document — and the implementation workflow to carry out and record each part of the work in the repo it belongs to. The discovery and architecture instructions open with a preloaded roster of the registered repos (name, description, role — rendered from config when the instruction is produced), so the agent starts every plan already knowing the codebases it is working across, and consults `repo list` for resolved paths and staleness. A new skill documents driving the repo-management commands, existing workflow skills gain their cross-repo behavior, and the README and changelog document the breaking configuration change.

*Technical detail:* [context.md#phase-42](./context.md#phase-42-workflows-and-skills-go-cross-repo)

**Acceptance criteria**:
- [x] The rendered discovery and architecture instructions include each registered repo's name, description, and role without the agent running any command
- [x] The plan workflow's discovery and architecture instructions direct the agent to attribute each requirement to a repo and files, using `repo list` for resolved locations
- [x] The implement workflow's instructions direct changes and derived changelog entries to each requirement's attributed repo
- [x] A repo-management skill is listed and retrievable alongside the existing skills
- [x] The README describes the project/repo configuration split and the repo registry; the changelog records the breaking change
- [x] The existing agent-workflow test suite passes against the updated templates

## Open Questions

- **Whether the chosen gitignore matcher library reproduces full gitignore(5) semantics** (negation re-inclusion under excluded parents, trailing-slash directory patterns, `**` anchoring). Depends on exercising the library against the Phase 4.1 semantics test matrix — its documentation is not proof. If `sabhiram/go-gitignore` fails a case, the implementer should swap to `go-git/v5/plumbing/format/gitignore` (accepting the module-graph growth) rather than patching around it; only if both fail should they STOP and ask the user about relaxing the semantics.

- **Whether `git ls-remote`/`git clone` can prompt interactively in agent-driven sessions** (credential prompts on private remotes would hang a non-interactive workflow). Depends on the user's git credential configuration — only observable by exercising real remotes. The implementer should set `GIT_TERMINAL_PROMPT=0` (and equivalent SSH batch-mode guards) in the git executor and treat a resulting auth failure as the normal structured error; if that proves too blunt for real setups, STOP and ask the user how clones should authenticate.

- **Whether the git executor's behavior is verified on Windows and macOS, not just Linux** — depends on what CI matrix the project actually runs, only discoverable at implementation time. The executor's platform-sensitive surface is deliberately small (PATH-based binary lookup, CRLF-tolerant output parsing, `filepath`-built clone targets, plain-path test remotes instead of `file://` URLs); the implementer should keep it that small, and if no Windows/macOS runner exists, flag cross-platform verification as a manual test-plan item rather than blocking on it.

- **How much of the command test suite silently depended on the default-config fall-back** removed in Phase 1.2. The research identified the fixture pattern but not every dependent test; the true blast radius surfaces when the suite first runs against the gate. The implementer should mechanically add minimal project fixtures where tests fail — no user input needed — but STOP and ask if any test's *intent* was specifically to exercise no-project behavior that the gate now changes.

- **Whether the harbor plan-workflow suite's oracle assertions embed the pre-split config shape or init output** (the suite renders and inspects real workflow instructions). Only the full harbor run against the updated binary reveals this. Oracle edits that merely track renamed sections are expected and fine; if an assertion encodes single-repo assumptions that the cross-repo templates genuinely break, STOP and surface the divergence to the user before editing the oracle.

- **Whether any consumer outside the identified construction sites builds a `FileStore` and would miss exclusion wiring** — a hidden assumption that only surfaces when Phase 4.1's wrapper lands and an unwrapped walker returns ignored paths. The implementer should grep for `NewFileStore` at that point and wire stragglers through the shared helper; no user input needed.

## Out of Scope

From the spec's Non-Goals, carried forward unchanged:

- **The plan/spec viewer tool** (rich rendering, inline review comments, collaborative server mode) — to be captured as a separate spec.
- **Machine-resolvable cross-install spec/plan links and reference-following tooling** — derived entries record project name, source, and spec/plan identifiers as structured data precisely so this can be built later without migrating entries; nothing in this plan follows those references.
- **Recursive projects** (a project as a member of another project) — explicitly excluded; registering a member repo never exposes that repo's own project-level world.
- **Automatic cleanup of derived changelog entries on deregistration** — removal is a manual config edit with no accompanying automation.
- **Automatic synchronization of cloned repos** — materialization is clone-if-absent plus a staleness warning; keeping clones current is the user's responsibility.
- **Local/private vs shared plans, and sharing plans via pull request** — deferred to follow-up work.
- **Detecting that completed work invalidated dependent plans** — deferred to follow-up work.
- **Token-efficiency optimization of planning context for large multi-repo projects** — a known concern deliberately parked.
- **A home-directory-level registry of projects** — explicitly excluded, despite what readers of the originating issue (#25) might expect.
- **Fully autonomous cross-repo execution** (dependency ordering, unsupervised parallelization across repos) — out of scope.

Left out by the chosen design, beyond the spec's list:

- **Backward compatibility and config migration** — the spec authorizes a breaking change; no migration tooling for pre-split `config.yaml` files is built, and existing setups re-initialize.
- **Comment-preserving config rewriting** — `repo add` writes config via the existing full-marshal round trip, dropping user comments; a yaml.Node-based editor is a possible follow-up if this proves painful in practice.
- **A `repo remove` command** — per the spec, removal is a manual project-config edit; no command ships.
- **A second repo provider** — the provider mechanism ships with only `git`; the typed per-provider config decoding it establishes is the extension point for future providers.
- **A dedicated cross-repo harbor scenario** — the existing harbor workflow suites must stay green, but authoring a new end-to-end multi-repo agent scenario is left to follow-up unless a phase finds it cheap; the corresponding success metrics are carried as manual test-plan items instead.
- **Exclusion support in agents' native file tools** — by spec constraint, `.spektacular_ignore` binds only Spektacular's own listing and search; agent tooling behavior is untouched.
- **Walk-time exclusion performance** — the ignore wrapper post-filters results; pushing the predicate into the directory walk for speed on huge ignored trees (e.g. node_modules) is deferred until profiling shows need, consistent with the parked token-efficiency concern.

## Changelog

### FINAL SUMMARY

All 11 phases across 4 milestones landed on 2026-08-13. Spektacular is now project-aware: configuration split into project-level config.yaml (required slug-safe `name`, optional `source`, member-repo registry) and per-repo repo.yaml (knowledge + changelog provider); running without a project fails explicitly with an init pointer (bootstrap commands exempt); init establishes identity, seeds the colocated repo, and cascades footprint repair over members. A new `internal/repo` package resolves registered repos (local-wins, clone-if-absent into gitignored `.spektacular/repos/`, warn-only staleness, footprint validation with repair offers) through the codebase's only subprocess, a quarantined three-method git executor; `repo add`/`repo list` manage the lifecycle. Changelog stores namespace entries by project name, and `changelog file write --repo` derives provenance-stamped per-repo entries through each repo's own store. `.spektacular_ignore` (gitignore semantics via sabhiram/go-gitignore) filters listing/search through a store decorator without ever blocking direct access. The plan workflow renders a preloaded repo roster into discovery/architecture instructions and attributes requirements to repos; skills, README, and CHANGELOG document the cross-repo behavior and the breaking configuration change. Deviations were minor and are recorded per phase; the full test suite (18+ packages, including real-git integration tests) is green.


### 2026-08-13 — Phase 1.1: Config splits into project and repo types

**What was done**: Project configuration gained a required slug-safe `name` and optional `source` field, validated at load with a new `validateSlug` helper (plus an exported `SlugifyName`); `NewDefault()` no longer seeds knowledge sources, making the project-level knowledge list an empty-by-default list of project-owned sources. A new independent `RepoConfig` type (`internal/config/repo.go`, file `repo.yaml`) holds repo-scoped concerns — the repo's own knowledge store (seeded by default) and its changelog provider — with load/validate/defaults/write mirroring the project config and no pointer back to any project.

**Deviations**: `KnowledgeConfig.WithDefaults`'s project-scope synthesis was kept functional rather than dropped (context.md suggested retiring it here) because its two callers — `internal/project/init.go:44` and `internal/knowledge/set.go:71` — are only rewired in Phases 1.3/2.4; dropping it now would break solo-project knowledge behavior mid-milestone. A small slice of Phase 1.3 was pulled forward: `project.Init` sets `cfg.Name` from the slugified directory basename for fresh configs, because `cmd/init.go` reloads and validates the config it just wrote — without a default name, init itself would fail this phase (Phase 1.3 still owns the `--name` flag and name backfill for existing configs).

**Files changed**:
- `internal/config/config.go`
- `internal/config/repo.go` (new)
- `internal/config/config_test.go`
- `internal/config/repo_test.go` (new)
- `internal/project/init.go`
- `internal/project/init_test.go`
- `cmd/spec_test.go`
- `cmd/knowledge_test.go`
- `cmd/init_test.go`
- `.spektacular/config.yaml` (this repo's own config gained `name: spektacular`)

**Discoveries**: `cmd/init.go` immediately reloads and re-validates the config that `project.Init` just wrote — any future required `Config` field must be given a default inside `project.Init` in the same change, or init breaks. The repo's own `.spektacular/config.yaml` is loaded by the workflow CLI driving this implementation, so breaking config changes must update it in the same commit to keep `go run .` working. `cmd/spec_test.go`'s `writeSpecCommandConfig` helper is shared across spec/root/artifacts/storefile/file/plan/changelog-file cmd tests — one edit there fixed all their fixtures; `cmd/storefile_metadata_test.go` needed no direct edit for the same reason.

### 2026-08-13 — Phase 1.2: Running without a project fails explicitly

**What was done**: The single config-loading seam (`loadConfig` in cmd/root.go) now returns a structured `no_project` error — naming the working directory, the missing config path, and `spektacular init <agent>` as the next action — when no `.spektacular/config.yaml` exists, instead of silently falling back to defaults. A new `loadConfigLenient` preserves the old fall-back for exactly three bootstrap paths: `init`, `version`'s stale-advice composition, and the root command's debug probe. There is still no parent-directory search: a project subdirectory is not the project.

**Deviations**: None.

**Files changed**:
- `cmd/root.go`
- `cmd/init.go`
- `cmd/version.go`
- `cmd/no_project_test.go` (new)
- `cmd/spec_test.go`
- `cmd/implement_test.go`
- `cmd/cross_kind_test.go`
- `cmd/plan_test.go`
- `cmd/error_response_test.go`
- `cmd/root_test.go`

**Discoveries**: No existing test deliberately exercised the old silent-default behavior — every gate-broken test was a fixture that predated the gate, so the plan's open question about intent-level blast radius resolved to "purely mechanical" (the fixture sweep reused `writeSpecCommandConfig`). Two session-log tests in `cmd/root_test.go` encoded "no config file" as their off-state; their intent was "debug defaults to off", so they now use a config without a `debug:` section, keeping the compared runs identical except for the debug toggle. `cmd/skill.go` genuinely never loads config, so the skill command needed no exemption.

### 2026-08-13 — Phase 1.3: Init establishes project identity and the colocated repo

**What was done**: `project.Init` now establishes the project's identity — name resolved as explicit `--name` flag, then the name already stored in config.yaml, then the slugified directory basename — writing config.yaml when absent or when the resolved name differs (backfill/override). Init also seeds the colocated repo's own `.spektacular/repo.yaml` (only-if-absent) and drives knowledge scaffolding from the repo config's sources instead of the project config's. Knowledge commands now aggregate the colocated repo's declared sources (repo.yaml, defaults when missing) followed by project-owned sources from config.yaml.

**Deviations**: A `config.ParseYAMLFile` (parse + defaults, no validation) was extracted from `FromYAMLFile` so init can read a nameless pre-split config and backfill it — the plan described load-modify-write but not that the validating loader would reject the load outright. `Config.Knowledge` and `KnowledgeConfig.Sources` gained `omitempty` YAML tags so a fresh config.yaml omits the empty project-level knowledge stub, honouring "written files stay minimal".

**Files changed**:
- `internal/config/config.go`
- `internal/project/init.go`
- `cmd/init.go`
- `cmd/knowledge.go`
- `internal/project/init_test.go`
- `cmd/init_test.go`
- `cmd/knowledge_test.go`
- `cmd/version_test.go`
- `internal/config/config_test.go`
- `.spektacular/repo.yaml` (new — this repo dogfooded via `go run . init claude`)
- `.spektacular/config.yaml` (this repo's now-repo-owned knowledge section removed)

**Discoveries**: `knowledge.NewSet`'s only production caller is `newKnowledgeSet` in cmd/knowledge.go, so aggregation-by-caller-assembly required no signature change; NewSet's internal `WithDefaults` call is now effectively dead (the repo config always supplies at least one source) but harmless, and Phase 2.4 should decide whether to remove it when aggregation goes registry-wide. Test fixtures that declared a "project"-scope source in config.yaml now duplicate the repo-default project source under aggregation — fixtures were migrated to declare only genuinely project-owned scopes (e.g. team) at project level, the shape real users should follow after the split.

### 2026-08-13 — Phase 2.1: The project config gains a repo registry

**What was done**: Project configuration gained the `repos` registry: each `RepoEntry` carries a required slug-safe unique name, address and/or local path, descriptive metadata (description, role, tags, dependencies, deployment), and a provider block (`git` only, empty defaults to git via `RepoEntry.WithDefaults`), validated from `Config.Validate` in the knowledge-sources style. Project init seeds the colocated repo (`{name: <project-name>, local: "."}`) into an empty registry — including backfilling older configs on re-init — so every project is uniformly a collection of repos.

**Deviations**: None.

**Files changed**:
- `internal/config/config.go`
- `internal/project/init.go`
- `internal/config/config_test.go`
- `internal/project/init_test.go`
- `.spektacular/config.yaml` (this repo dogfooded: colocated repo seeded by re-running init)

**Discoveries**: Config validation errors surface through the generic `internal_error` envelope code (only the missing-config gate has a dedicated `no_project` code) — existing behavior, but worth knowing when Phase 2.2 adds structured repair errors that need their own codes.

### 2026-08-13 — Phase 2.2: Repos resolve to local directories, cloning when absent

**What was done**: The new `internal/repo` package resolves registered repos to local directories: `Set` (modeled on the knowledge set) dispatches the `git` provider fail-fast, uses a repo's local path when present (git never invoked), otherwise materializes it by cloning its address into `.spektacular/repos/<name>/` — only when absent, never fetching or pulling. Staleness is a warn-only note (unreachable remotes degrade to a could-not-check notice), and a missing or invalid `.spektacular/repo.yaml` in the resolved repo surfaces as a `*FootprintError` carrying the repo name and root alongside the still-populated resolution, so the CLI layer can offer repair. The quarantined `GitRunner` executor in `internal/repo/git.go` is the codebase's first and only subprocess site (`GIT_TERMINAL_PROMPT=0`, batch-mode SSH guard only when the user hasn't set their own, CRLF-tolerant output parsing, PATH-resolved binary). The embedded `.spektacular/.gitignore` template gained the `repos/` entry.

**Deviations**: `Set` also exposes an `Entries()` accessor (registry order, defaults applied, no resolution) — a small forward-slice for Phase 2.3's `repo list`, which must report without cloning.

**Files changed**:
- `internal/repo/git.go` (new)
- `internal/repo/set.go` (new)
- `internal/repo/set_test.go` (new)
- `internal/repo/git_integration_test.go` (new)
- `templates/.spektacular/.gitignore`
- `.spektacular/.gitignore` (this repo re-inited to pick up the `repos/` entry)

**Discoveries**: `Resolve` deliberately returns the populated `ResolvedRepo` together with a `*FootprintError` (rather than a zero value) so the repair flow knows the resolved root — callers must check the error before trusting the footprint but may use `Root` for repair. Integration tests mask the user's git config via `GIT_CONFIG_GLOBAL`/`GIT_CONFIG_SYSTEM=os.DevNull` (needs git ≥ 2.32) and set identity through `GIT_AUTHOR_*`/`GIT_COMMITTER_*` env — the pattern future git-touching tests should copy. Plain-path local remotes (not `file://` URLs) keep the tests Windows-portable.

### 2026-08-13 — Phase 2.3: The CLI manages the repo lifecycle

**What was done**: Added the `repo` command group: `repo add` registers a repo (idempotent upsert by name — identical entries skip the config write, the codebase's first CLI-driven config mutation otherwise) and creates or repairs the target's minimal footprint via the new shared `repo.EnsureFootprint` routine, reporting `{registered, footprint: created|repaired|unchanged}`; `repo list` reports the registry with metadata and resolved roots, staying strictly side-effect-free — repos not yet on disk (checked via the new `Set.Present`) are reported unmaterialized rather than cloned. Category README rendering moved onto `knowledge.Category.README()` so init and footprint creation share one renderer.

**Deviations**: Same-name re-adds with different metadata update the entry in place (upsert) rather than erroring — the plan only specified identical-no-op and new-append; an update was chosen so agents can correct metadata without hand-editing config. `EnsureFootprint` creates only missing pieces and never overwrites existing knowledge files (a healthy existing repo.yaml is kept as the authority for its own scaffolding; a broken one is rewritten from defaults) — slightly stricter than init's overwrite-READMEs behaviour, in service of the "changes nothing in the target repo" criterion.

**Files changed**:
- `cmd/repo.go` (new)
- `cmd/repo_test.go` (new)
- `internal/repo/footprint.go` (new)
- `internal/repo/footprint_test.go` (new)
- `internal/repo/set.go` (added `Set.Present`)
- `internal/knowledge/category.go` (added `Category.README()`)
- `internal/project/init.go` (uses the shared README renderer)
- `cmd/root.go` (registers `repoCmd`)

**Discoveries**: `repo add` resolves the target before creating the footprint, so an address-only repo IS cloned at add time (the list/init never-clone rule deliberately does not extend to add — the footprint has to be written somewhere). The `repoGit` package variable in cmd is the seam tests use to fake git; save/restore it with t.Cleanup. `repo add` for a local-path target requires the directory to exist — it registers and footprints repos, it does not create their working trees.

### 2026-08-13 — Phase 2.4: Init cascades over members, and knowledge flows through the registry

**What was done**: Re-running project init now cascades over the registered repos, creating or repairing each one's minimal footprint with the same `repo.EnsureFootprint` routine `repo add` uses — never cloning: unmaterialized address-only repos are skipped with a printed notice. Knowledge aggregation switched from the colocated shortcut to the registry: every registered repo's repo.yaml sources participate in registry order (colocated first, relative locations resolved against each repo's own root), followed by project-owned sources, with every source stamped for attribution (`knowledge sources` now reports which repo declared it, via the new `SourceInfo.Repo` field). A member repo with a missing or invalid repo.yaml surfaces a structured `repo_footprint` error whose next action is the `repo add` repair; the colocated repo degrades to defaults for pre-split compatibility. Duplicate scopes across repos resolve to the first match in registry order.

**Deviations**: `project.Init` now returns `([]string, error)` — the notices channel the plan implied but didn't specify a shape for; cmd/init prints them. An empty registry is treated as an implicitly registered colocated repo so minimal hand-written configs (and the many test fixtures shaped that way) keep their knowledge behavior. `knowledge.NewSet`'s internal `WithDefaults` call was left in place (still harmless; the aggregator always supplies sources).

**Files changed**:
- `internal/project/init.go`
- `cmd/init.go`
- `cmd/knowledge.go`
- `internal/knowledge/set.go`
- `internal/config/config.go` (SourceConfig.Repo, programmatic-only)
- `internal/repo/set.go` (LocalRoot; Present delegates)
- `internal/project/init_test.go`
- `cmd/init_test.go`
- `cmd/knowledge_test.go`

**Discoveries**: Relative knowledge locations in a member's repo.yaml must be absolutized against the member's root before reaching `knowledge.NewSet`, which joins relative paths to the *project* root — the aggregator owns that translation. `SourceConfig.Repo` is `yaml:"-"`: attribution is derived from which file declared the source, never declarable in config. The repair-offer error (`repo_footprint` + next-action) established here is the shape Phase 3.2's repo-routed changelog writes should reuse when they hit broken footprints.

### 2026-08-13 — Phase 3.1: Changelog stores namespace by project

**What was done**: Every changelog store now organizes entries under a folder named after the project: the `changelog file` command group injects `<project-name>` below the CLI surface (agent-facing paths stay `<id>_<slug>.md`, and `validateIDPrefix` needed no change), the implement path strategy renders the true nested `changelog_path` via `ChangelogFilePath(dir, projectName, name)` fed from the new `workflow.Config.ProjectName` scalar (populated by the spec, plan, and implement commands), and artifact listing follows the namespace folder down, tolerating its absence. Front-matter metadata gained the four provenance fields (`project`, `project_source`, `spec`, `plan`, all omitempty) with incoming-wins-when-set merge semantics and unchanged date/status invariants.

**Deviations**: None beyond scope-neutral detail: the sub-agent's test pass exposed that `finished()` in the implement steps still closed the changelog at the flat path — fixed in the same phase to use the namespaced path, or completed runs would have left entries `in-progress` forever.

**Files changed**:
- `cmd/changelog_file.go`
- `cmd/artifacts.go`
- `internal/workflow/workflow.go`
- `cmd/implement.go`, `cmd/plan.go`, `cmd/spec.go`
- `internal/steps/implement/strategy.go`
- `internal/steps/implement/steps.go` (finished() close path)
- `internal/metadata/metadata.go`, `internal/metadata/merge.go`
- Tests: `cmd/artifacts_test.go`, `cmd/changelog_file_test.go`, `cmd/storefile_metadata_test.go`, `cmd/storefile_list_filter_test.go`, `internal/metadata/metadata_test.go`, `internal/steps/implement/strategy_test.go` (new), `internal/steps/implement/steps_test.go`

**Discoveries**: The namespace injection has a second consumer beyond the CLI dir func: any Go code that composes changelog store paths directly (the `finished()` close path was one) must use `ChangelogFilePath` with the project name — grep for `ChangelogDir` joins when touching this area. Legacy flat entries are invisible to `changelog file list` and artifact listing (accepted breaking change; they stay readable by direct path only, and `changelog file list` errors not-found until the namespace folder exists — pre-existing behavior for empty stores).

### 2026-08-13 — Phase 3.2: Implemented work derives per-repo changelog entries

**What was done**: The changelog `file` command group gained `--repo <name>` routing on write, read, and list (spec and plan groups deliberately did not): with the flag, the command operates on the named member repo's own configured changelog store, rooted at the resolved repo and namespaced by the project's name, and writes are auto-stamped with structured provenance — project name, project source when set, and the spec/plan identifiers derived from the filename. A missing or broken member footprint surfaces the standard `repo_footprint` repair offer. The implement templates gained the derivation procedure: phase entries' Files-changed lists prefix member-repo paths with `<repo-name>: `, the feature-changelog step derives one filtered entry (with a human-readable reference line) per affected repo and writes it with `--repo`, and the release-note step now updates each affected repo's root CHANGELOG.md.

**Deviations**: None.

**Files changed**:
- `cmd/storefile.go`
- `cmd/changelog_file.go`, `cmd/file.go`, `cmd/plan_file.go` (repoRouted parameter)
- `templates/steps/implement/10-update_feature_changelog.md`
- `templates/steps/implement/07-update_changelog.md`
- `templates/steps/implement/08-update_repo_changelog.md`
- `cmd/changelog_file_test.go`
- `internal/steps/implement/steps_test.go`

**Discoveries**: The `newStoreFileCmd` closure flags are package-lifetime state — the new `--repo` flag needed its own reset helper in tests (`resetChangelogRepoFlags`), same footgun as every other cobra flag here. Provenance stamping is CLI-owned and only fires on `--repo` writes; central entries stay provenance-free by design (they live inside the project). The spec's two-projects-one-repo no-collision criterion is now covered end-to-end by both a cmd test and live verification.

### 2026-08-13 — Phase 4.1: Sources can exclude paths from listing and search

**What was done**: Added `.spektacular_ignore` support: a gitignore-syntax exclusion file rooted at the source it governs, honoured by a new ignore-aware `store.Store` decorator (`NewIgnoreStore`) that filters only `List` and `Search` — `Read`/`Write`/`Exists`/`Delete` delegate untouched, so a directly named path is never blocked and agents' native tools are unaffected. The `NewSourceStore(root, scope)` helper (FileStore + the root's own ignore rules) was adopted at every store construction site: knowledge sources, the central and repo-routed changelog stores, artifact scanning, and the spec/plan/implement workflow stores. `github.com/sabhiram/go-gitignore` provides the matching; a missing ignore file yields a pass-everything matcher, keeping existing sources byte-identical in behavior.

**Deviations**: None — the plan's open question about the matcher library resolved favorably: the semantics matrix confirmed sabhiram/go-gitignore handles negation, anchoring, `**`, directory patterns, globs, and comments correctly, so the go-git fallback was not needed.

**Files changed**:
- `go.mod`, `go.sum` (new dependency: github.com/sabhiram/go-gitignore)
- `internal/store/ignore.go` (new)
- `internal/store/ignore_test.go` (new)
- `internal/store/search.go` (doc comment)
- `internal/knowledge/set.go`, `cmd/storefile.go`, `cmd/artifacts.go`, `cmd/spec.go`, `cmd/plan.go`, `cmd/implement.go` (NewSourceStore adoption)
- `cmd/knowledge_ignore_test.go` (new)

**Discoveries**: One documented divergence from git, pinned in a test: with `dir/` excluded and `!dir/file.md` negated, real git keeps the file ignored (it never descends into excluded directories) while the library re-includes it per-path — in practice `List` still hides it (the parent vanishes from its parent's listing) but `Search` hits in such a file will surface. The `.spektacular_ignore` file itself appears in listings unless a source self-excludes it with its own line.

### 2026-08-13 — Phase 4.2: Workflows and skills go cross-repo

**What was done**: The plan workflow's discovery and architecture instructions now open with a preloaded roster of the registered repos — identity metadata (name, description, role, tags, deployment) projected from config by `repoRoster` in cmd/plan.go, refreshed into workflow data on every `plan new`/`plan goto` invocation and rendered via the Extra map, with a clean fallback for empty registries — and direct the agent to attribute every requirement to a repo and files, using `repo list` for resolved paths and recording the resolution in the plan's context document. The phases template carries attribution into phase summaries and `<repo>:path:line` file lists. The spawn-planning and spawn-implementation skills gained per-repo fan-out and attributed-repo guidance, a new `manage-repos` library skill documents the repo CLI, the spek-plan and spek-implement workflow skills gained cross-repo notes, the README's Configuration section was rewritten for the project/repo split (fixing the pre-existing missing-changelog-section drift), and CHANGELOG.md records the breaking change.

**Deviations**: The roster reaches templates through workflow data (`SetData("repos", ...)` refreshed each invocation) rather than a `workflow.Config` field — the plan named the Extra-map route but left the data channel unspecified; this keeps `workflow.Config` and stepkit untouched as intended while staying fresh-from-config at render time. The drift noted at read_plan (no `cmd/skill_test.go`) was resolved by creating it as a new topic-grouped test file.

**Files changed**:
- `cmd/plan.go`
- `internal/steps/plan/steps.go`
- `templates/steps/plan/02-discovery.md`, `03-architecture.md`, `10-phases.md`
- `templates/skills/skill_spawn-planning-agents.md`, `skill_spawn-implementation-agents.md`
- `templates/skills/skill_manage-repos.md` (new)
- `templates/skills/workflows/spek-plan/SKILL.md`, `spek-implement/SKILL.md`
- `README.md`, `CHANGELOG.md`
- Tests: `internal/steps/plan/steps_test.go`, `cmd/skill_test.go` (new), `cmd/docs_test.go` (new), `internal/agent/instruction_surface_test.go`

**Discoveries**: Workflow data survives the state.json JSON round-trip as `[]any` of `map[string]any` — mustache handles both that and the fresh `[]map[string]any` shape, and the roster tests pin the round-trip shape. Library skills are served raw (no `{{command}}` substitution), so `skill_manage-repos.md` writes commands bare; only workflow SKILL.md files may use placeholders. Pre-existing gofmt drift exists in `internal/runner`, `internal/stepkit` (test), and `internal/steps/spec` — untouched by this plan, left alone.
