# Plan: 000020_context

<!-- Metadata -->
<!-- Created: 2026-05-15T11:49:44Z -->
<!-- Commit: b564e9fc5a693c3409abf70fa426e2d5eb45d5a0 -->
<!-- Branch: main -->
<!-- Repository: jumppad-labs/spektacular -->

## Overview

This plan adds a multi-source knowledge layer to Spektacular so a project can draw on architectural decisions, team conventions, and past learnings that live outside its codebase. Today that knowledge must sit in a single hardcoded directory beside each project, forcing copies between projects and making sharing awkward. After this work a project configures multiple knowledge sources at distinct scopes — project, team, global — each independently located and keyword-searchable, so engineering teams can collaborate on shared knowledge when it helps and keep it private when it doesn't. Delivering this also makes Spektacular's configuration provider-based: `spec`, `plan`, and `knowledge` each name a provider — today only `file` — and a provider-specific `config` block. That refactor is the vehicle the knowledge sources ride on, and it lifts the spec and plan output directories, until now hardcoded string literals, into configurable `config.directory` values.

## Architecture & Design Decisions

The feature adds a multi-source knowledge layer built on the existing `store.Store` abstraction. `Store` gains a new method, `Search`, making keyword search a first-class capability every backend serves itself — there is no central search layer — and its `List` method is widened to return typed entries that distinguish a file from a directory, so a caller can recursively enumerate a store's whole tree (knowledge directories are trees: `architecture/`, `learnings/`, `gotchas/` sit under the root). A new `internal/knowledge` package holds a `Set`: an ordered collection of configured stores, each labelled with a scope (e.g. `project`, `team`, `global`). The `Set` fans `Read`, `List`, and `Search` out across every member store and concatenates the results, tagging each by its originating scope. A new `cmd/knowledge.go` command group (`search`, `read`, `list`, `write`, `sources`) exposes the `Set` to agents, who already drive Spektacular by calling the CLI and parsing its JSON.

This direction was chosen over threading a multi-store through the workflow `StepCallback` signature (Option B). The codebase never assembles context in Go — steps emit instruction text and the agent reads files with its own tools. Putting knowledge behind a CLI command group preserves that grain: the single-store `StepCallback` signature and every existing call site stay untouched, and agents gain search/read/write through the same JSON-over-CLI channel they already use. A templates-only approach (Option C) was rejected because it reduces "cross-source search" to the agent's ad-hoc grepping, which cannot satisfy the per-store-tagged-results or unreachable-source acceptance criteria.

Three key trade-offs are locked in. **Search is per-store and un-ranked globally**: each store returns generic hits (scope, locator/path, ~250-character excerpt, optional cheap relevance score) and the `Set` concatenates them in configured order — the agent scans excerpts and decides what to `Read` in full. **`FileStore.Search` prefers ripgrep with a native Go fallback**: when `rg` is on `PATH` the backend shells out and parses `rg --json`, otherwise it walks the directory and scans lines in pure Go — confirmed during discovery as the only robust option, since no maintained Go *library* is a ripgrep equivalent. **Write confirmation stays instruction-gated**, consistent with every existing approval point in the codebase: `knowledge write` persists when called, and the discovery template prose instructs the agent to propose a scope and obtain explicit user confirmation before invoking it. Knowledge-entry structure (frontmatter, titles) remains a file-format convention parsed above `Store`, never leaking into the interface.

**Configuration becomes provider-based.** Rather than each domain inventing its own config shape, `spec`, `plan`, and `knowledge` follow one pattern — a `provider` field names a backend (`file` is the only one this feature ships) and a `config` block carries that provider's settings. This aligns the configuration grain with the storage grain: the same `file`-vs-future-remote distinction the `Store` interface draws. As a direct consequence the spec and plan output directories move out of hardcoded string literals into `spec.config.directory` and `plan.config.directory`. `knowledge` carries the pattern per source: each entry in `knowledge.sources` names its own `provider`, `scope`, and `config`, so scopes can use different backends independently once a second provider exists. Today `config` decodes straight into the file provider's typed struct; provider-keyed decoding is deferred until a second provider lands.

Rejected options and the evidence behind them are recorded in `research.md#alternatives-considered-and-rejected`.

## Component Breakdown

- **`Store` interface (changed)** — gains a `Search` method and widens `List` to return typed entries that mark each child as a file or a directory. Search becomes part of the storage contract every backend must satisfy, and the typed `List` lets a caller recursively walk a store's whole tree — both without knowing the backend type.

- **`FileStore` (changed)** — implements the new `Search` over the local filesystem. It prefers the `ripgrep` binary when available (parsing its structured JSON output) and falls back to a native Go directory walk and line scan that produces equivalent hits. It also carries its configured scope label so the hits it returns are self-identifying. Existing `FileStore` behaviour is otherwise unchanged.

- **`knowledge.Set` (new)** — an ordered collection of configured stores, each paired with a scope label. It owns the multi-source orchestration: fanning `Read`, `List`, and `Search` out to every member store, concatenating the results in configured order, and surfacing a clear error that names any source it cannot reach. It is the single thing the CLI talks to and has no opinion on backend type.

- **`knowledge.Set` constructor (new)** — builds a `Set` from the project configuration: it reads the list of configured sources, resolves each into a `Store` of the declared type at the declared location, and validates that each source is reachable. This is the one place that maps config entries to live stores.

- **Provider-based configuration (changed `Config`)** — `spec`, `plan`, and `knowledge` each become a `provider` plus a provider-specific `config` block. `SpecConfig` is restructured (its `id_method` moves under `config`), a new `PlanConfig` is added, and `knowledge` holds an ordered list of sources, each naming its own `provider`, `scope`, and `config`. Each section synthesises a default when absent — the `file` provider with `specs`/`plans` directories, and a single `project` knowledge source at `.spektacular/knowledge` — so existing projects keep working. Team and global knowledge sources are opt-in additions the user configures by hand.

- **Spec/plan directory wiring (changed steps)** — the spec and plan workflows stop hardcoding the `specs/` and `plans/` path segments and instead read them from `spec.config.directory` and `plan.config.directory`. `SpecFilePath`/`PlanFilePath` (and the `implement` copy), the spec enumeration in `identifier.go`, the spec-path resolution in `plan/strategy.go`, and the directory creation at `init` all take the configured directory. The directory remains a subpath of the `.spektacular` store root — the single-store architecture is unchanged.

- **`knowledge` CLI command group (new)** — a top-level command exposing `search`, `read`, `list`, `write`, and `sources` subcommands. Each constructs the `knowledge.Set` from config and emits JSON: `search` returns scope-tagged hits with locators and excerpts, `read` returns a full entry from a named scope, `list` enumerates entries across all scopes, `sources` lists the configured scopes and locations, and `write` persists a new entry to one chosen scope. It is the agent's entry point to the knowledge layer, mirroring the existing `spec`/`plan`/`implement` command pattern.

- **Discovery step template (changed)** — the planning workflow's discovery instruction stops hardcoding the `.spektacular/knowledge/` path and instead directs the agent to use the `knowledge` search/read commands, and to propose a scope and obtain explicit user confirmation before any `knowledge write`.

- **README (changed)** — gains a section documenting the `Store` interface contract (including `Search`) and walking through `FileStore` as the worked example for implementing and registering a new backend.

## Data Structures & Interfaces

**`Store` interface — changed.** `Store` gains `Search`, and `List` is widened to return typed entries instead of bare name strings so callers can tell a file from a directory and recurse:

```go
// Search returns hits for a free-form keyword query, scanning only this store.
// Hits carry the store's own scope so callers can attribute them.
Search(query string) ([]Hit, error)

// List returns the direct children of the directory at path. Each entry
// reports whether it is a directory, so a caller can recurse the tree.
List(path string) ([]DirEntry, error)
```

**`store.DirEntry` (new).** A typed directory child — replaces the bare `[]string` the old `List` returned:

```go
type DirEntry struct {
    Name  string // child name, not a full path
    IsDir bool   // true for a subdirectory — recurse into it via List
}
```

**`store.Hit` (new).** The generic search result every backend produces — never the full body, never a structured title/abstract:

```go
type Hit struct {
    Scope   string  // scope label of the originating store
    Path    string  // locator, relative to the store root — pass to Read
    Excerpt string  // compact excerpt, capped at the ~250-char budget
    Score   float64 // optional cheap relevance score; 0 when the backend has none
}
```

**`FileStore` — new field.** `FileStore` carries the scope label it was constructed with so its hits are self-identifying; `NewFileStore` gains a scope parameter (existing workflow call sites pass a fixed internal scope, `"project"`).

**`knowledge.Set` (new).** The multi-source orchestrator. It holds an ordered list of scoped stores and fans operations across them:

```go
type Set struct { /* ordered, scoped stores */ }

func NewSet(cfg config.Config) (*Set, error)   // resolves config sources into live stores

func (s *Set) Search(query string) ([]store.Hit, error)   // concatenated, scope-tagged
func (s *Set) Read(scope, path string) ([]byte, error)     // full entry from one scope
func (s *Set) List() ([]Entry, error)                      // all entries, every scope, recursive
func (s *Set) Write(scope, path string, content []byte) error
func (s *Set) Sources() []SourceInfo                       // configured scopes + locations
```

`knowledge.Entry` is `{Scope, Path string}`; `knowledge.SourceInfo` is `{Scope, Provider, Location string}`. Any operation that touches an unreachable source returns an error naming that source rather than partial results.

**Provider-based `Config` (changed).** `spec`, `plan`, and `knowledge` each name a `provider` and carry a provider-specific `config` block. `file` is the only provider this feature ships:

```go
type Config struct {
    Command   string          `yaml:"command"`
    Agent     string          `yaml:"agent"`
    Debug     DebugConfig     `yaml:"debug"`
    Spec      SpecConfig      `yaml:"spec"`
    Plan      PlanConfig      `yaml:"plan"`
    Knowledge KnowledgeConfig `yaml:"knowledge"`
}

const ProviderFile = "file" // the only provider this feature ships

// SpecConfig — restructured: id_method moves under the provider config.
type SpecConfig struct {
    Provider string         `yaml:"provider"` // "file"
    Config   FileSpecConfig `yaml:"config"`
}
type FileSpecConfig struct {
    Directory string `yaml:"directory"`  // spec directory, default "specs"
    IDMethod  string `yaml:"id_method"`  // timestamp | counter | external
}

// PlanConfig — new.
type PlanConfig struct {
    Provider string         `yaml:"provider"` // "file"
    Config   FilePlanConfig `yaml:"config"`
}
type FilePlanConfig struct {
    Directory string `yaml:"directory"`  // plan directory, default "plans"
}

// KnowledgeConfig — new. Each source names its own provider, so scopes
// can use different backends independently once a second provider exists.
type KnowledgeConfig struct {
    Sources []SourceConfig `yaml:"sources"`
}
type SourceConfig struct {
    Scope    string              `yaml:"scope"`     // free-form label: project/team/global
    Provider string              `yaml:"provider"`  // "file"
    Config   FileKnowledgeConfig `yaml:"config"`
}
type FileKnowledgeConfig struct {
    Location string `yaml:"location"`  // directory path for a file source
}
```

`config` decodes straight into the typed file-provider struct because only one provider exists; provider-keyed decoding is deferred (see Out of Scope). The configured YAML looks like:

```yaml
spec:
  provider: file
  config:
    directory: specs
    id_method: timestamp
plan:
  provider: file
  config:
    directory: plans
knowledge:
  sources:
    - scope: project
      provider: file
      config:
        location: .spektacular/knowledge
    - scope: team
      provider: file
      config:
        location: /shared/team-kb
```

**CLI JSON output (new serialization boundary).** The `knowledge` command group emits, per subcommand: `search` → `{"hits": [Hit, ...]}`; `list` → `{"entries": [Entry, ...]}`; `read` → `{"scope", "path", "content"}`; `write` → `{"scope", "path"}`; `sources` → `{"sources": [SourceInfo, ...]}`. Errors use the existing `{"error": ...}` envelope from `internal/output`.

## Implementation Detail

**A new module boundary, not a new architecture.** The `knowledge` package sits as a thin orchestration layer above `store`, and the `knowledge` command group sits beside the existing `spec`/`plan`/`implement` commands. Both follow patterns the codebase already establishes — a cobra command group with `--data`/`--schema`/JSON-output conventions, and a package that depends only on `store` and `config`. No workflow state machine, no callback-signature change, no new agent-orchestration shape. A developer reading the change sees one new leaf package and one new command file, each shaped like its neighbours.

**Search dispatch inside `FileStore`.** `Search` introduces one genuinely new pattern: a capability-detected execution path. At call time `FileStore` probes for the `ripgrep` binary on `PATH`; if present it shells out and decodes the stream of `rg --json` event objects into `Hit`s, if absent it runs a native Go directory walk that opens each file, scans lines for the query, and builds equivalent `Hit`s. This is the codebase's *first* use of subprocess execution — previously the integration model only ran in the other direction (agents calling this binary). The two paths sit behind the single `Search` method so no caller can observe which ran; both produce hits whose excerpts respect the same compact budget and whose scoring is best-effort (rg can supply line context cheaply; the fallback may leave `Score` zero). The shared excerpt-trimming logic is one helper both paths call, so the budget is enforced in exactly one place.

**Fan-out and fail-fast in `Set`.** `Set` operations iterate the configured stores in order and concatenate. The deliberate design choice is fail-fast over partial results: the moment any store reports its location is unreachable, the whole operation returns an error naming that source — search never silently returns a subset. `Set` performs no ranking, no dedup, and no precedence resolution across scopes; overlapping entries from two scopes both appear, each tagged, exactly as the spec requires.

**Config resolution and defaults.** Every domain switches on its `provider` field — today only `file` resolves, and an unknown provider is a clear configuration error rather than a silent skip. `knowledge.NewSet` is the sole place that maps `SourceConfig` entries to live `Store` instances. When a section is absent the system synthesises a default: `spec` and `plan` default to the `file` provider with `specs`/`plans` directories, and `knowledge` to a single `project` source pointing at the `init`-created `.spektacular/knowledge` directory — so existing projects keep working with zero config. Relative knowledge source locations resolve against the project root, which keeps the default portable while letting team and global sources use absolute paths; spec and plan directories resolve as subpaths of the `.spektacular` store root, so the single-store architecture is untouched.

**Write stays instruction-gated.** `knowledge write` is an ordinary persisting command — it does not implement a confirmation prompt, because the CLI is non-interactive and every approval point in the codebase is gated by agent instruction text instead. The new behaviour lives entirely in the updated discovery template prose: the agent must propose a target scope and obtain explicit user confirmation before it is allowed to invoke `knowledge write`. A developer looking for "the confirmation logic" finds it in the template, consistent with the spec-verify step.

## Dependencies

**Internal packages**

- **`internal/store`** — provides the `Store` interface and `FileStore`. *Changed by this plan*: gains the `Search` method and the `Hit` type, widens `List` to return typed `DirEntry` values, and adds a scope label on `FileStore`.
- **`internal/config`** — provides the project `Config` and YAML load/save. *Changed by this plan*: configuration becomes provider-based — `SpecConfig` is restructured, `PlanConfig` and `KnowledgeConfig`/`SourceConfig` are added, each with provider validation.
- **`internal/steps/spec`, `internal/steps/plan`, `internal/steps/implement`** — own the spec/plan workflows and the `SpecFilePath`/`PlanFilePath` helpers. *Changed by this plan*: the hardcoded `specs/`/`plans/` path segments are replaced with the configured directories.
- **`internal/output`** — provides the JSON result/error writer used by every command. *Used as-is*; the new `knowledge` command emits through it unchanged.
- **`cmd` (cobra root)** — the new `knowledge` command group registers on the existing root command alongside `spec`/`plan`/`implement`. *Changed*: one new command file plus a root registration line.
- **`internal/project`** — creates the `.spektacular` directory tree at `init`. *Changed by this plan*: `init` creates the spec and plan directories named in config (defaulting to `specs`/`plans`); the `knowledge` directory it creates becomes the default `project` source.
- **`templates`** — embedded step templates. *Changed*: the discovery step template is rewritten to use the `knowledge` commands instead of a hardcoded path.

**External libraries**

- **`gopkg.in/yaml.v3`** — already a dependency; serializes the new `knowledge` config section. No version change.
- **`github.com/spf13/cobra`** — already a dependency; hosts the new command group. No change.
- **`github.com/stretchr/testify`** — already the test dependency; covers the new tests. No change.
- **No new Go module dependency.** Discovery confirmed there is no maintained Go *library* equivalent to ripgrep, so search uses the standard library (`os/exec`, `os`, `bufio`) for both the `rg` shell-out and the native fallback.

**External runtime tool**

- **`ripgrep` (`rg`) binary** — an *optional* runtime dependency, not a build or module dependency. When present on `PATH` it is the fast search path; when absent the native Go fallback keeps the feature fully functional. Nothing needs to install it.

**Planning dependencies**

- **Prior plan `000014_spektacular_store`** — established the `Store` interface and `FileStore`; already landed. This plan extends it. No ordering constraint — nothing else must land first.

## Testing Approach

**Unit tests carry the load.** Every new component is exercised by `go test` unit tests using the project's established conventions — `testify/require`, `t.TempDir()` for filesystem isolation, `TestSubject_Behaviour` naming — slotting beside the existing `store_test.go` and `config_test.go`. No new test framework or harness is introduced.

**`FileStore.Search` gets the deepest coverage**, because it has two execution paths that must produce equivalent results. The load-bearing guarantee is path equivalence: the same query against the same fixture directory yields the same hits whether the `ripgrep` path or the native Go fallback ran. To make this deterministic the search path is testable via a seam that forces the fallback regardless of whether `rg` is installed; the `rg` path is covered when the binary is available and skipped cleanly when it is not, so the suite passes on any machine. Other Search assertions: excerpts never exceed the compact budget, hits carry the correct scope and a locator that round-trips through `Read`, and a query with no matches returns an empty result rather than an error.

**`knowledge.Set` is tested for fan-out and fail-fast.** Tests stand up multiple `FileStore`s at distinct temp directories and assert that reads, lists, and searches return entries from every configured scope; that overlapping entries on the same topic surface from both scopes, each correctly tagged; and — the critical negative case — that pointing a source at an unreachable location makes the operation fail with an error naming that source, never a silent partial result. A confirmed-write test asserts the entry lands only in the chosen scope and no other scope changes.

**Config, directory wiring, and CLI get contract-level tests.** Config tests cover round-tripping the provider-based `spec`, `plan`, and `knowledge` sections through YAML, default synthesis when a section is absent, and provider validation (unknown provider, missing `config` field, duplicate scope). A directory-wiring test points `spec.config.directory` and `plan.config.directory` at non-default names and asserts specs and plans are written, enumerated, and found under those directories, and that `init` creates them. The `knowledge` command tests assert the JSON output shape of each subcommand against the documented envelope, following the existing command-test style.

**Deliberate gaps.** No new end-to-end Harbor test is added — the agent-facing behaviour (proposing a scope, gating `write` on user confirmation) is template prose, verified by asserting the discovery template's rendered content contains the confirmation beat, the same way the plan-workflow step tests already assert template content. Confirmation itself is an agent behaviour outside the binary's testable surface. Cross-source search against a *remote* backend is untestable here because no remote backend ships.

## Milestones & Phases

### Milestone 1: Knowledge stores can be searched

**What changes**: Search becomes a first-class capability of every storage backend. A `Store` can answer a free-form keyword query and return compact, scope-tagged hits — each a locator plus a short excerpt — without exposing full file bodies. The local filesystem backend serves search fast via `ripgrep` when it is installed and via an equivalent built-in scan when it is not, so the capability works on any machine with no setup. Nothing user-facing ships in this milestone; it is the foundation the multi-source layer builds on.

#### - [x] Phase 1.1: Extend the `Store` contract with search

Adds the `Search` method to the `Store` interface, the `Hit` result type, a scope label on `FileStore`, and widens `List` to return typed `DirEntry` values that distinguish files from directories. This phase changes only the contract and constructor — no search behaviour yet — so the codebase still builds, with every existing store call site updated to pass a scope and to consume the new `List` return type. It is split out so the behaviour-bearing phase that follows starts from a compiling tree.

*Technical detail:* [context.md#phase-11](./context.md#phase-11-extend-the-store-contract-with-search)

**Acceptance criteria**:

- [x] The `Store` interface declares `Search` and the `Hit` type exists, carrying scope, locator, excerpt, and optional score.
- [x] `Store.List` returns typed entries that distinguish files from directories, and every existing `List` caller is updated to the new return type.
- [x] `FileStore` records the scope it was constructed with, and every existing workflow call site passes a fixed `project` scope.
- [x] The project builds and all existing tests pass.

#### - [x] Phase 1.2: Implement `FileStore.Search` (ripgrep + native fallback)

Implements search over the local filesystem: when `ripgrep` is on `PATH` it shells out and decodes `rg --json`, otherwise it walks the directory and scans lines in pure Go. Both paths share one excerpt-trimming helper so the compact budget is enforced in a single place, and both produce equivalent scope-tagged hits. A test seam forces the fallback path so both can be tested deterministically on any machine.

*Technical detail:* [context.md#phase-12](./context.md#phase-12-implement-filestore-search-ripgrep--native-fallback)

**Acceptance criteria**:

- [x] A keyword query against a directory returns hits whose excerpts stay within the compact budget.
- [x] The ripgrep path and the native fallback return equivalent hits for the same query and fixture.
- [x] Each hit carries the store's scope and a locator that round-trips through `Read`; a no-match query returns an empty result, not an error.

### Milestone 2: Configuration becomes provider-based and a project can query multiple knowledge sources

**What changes**: Configuration becomes provider-based — `spec`, `plan`, and `knowledge` each name a provider and a `config` block — which makes the spec and plan output directories configurable for the first time, lifting them out of hardcoded string literals. On that foundation, a project can declare multiple knowledge sources, each with its own scope, provider, and config, configured independently — and Spektacular reads, lists, and searches across all of them at once. A new `knowledge` command lets an agent search every source with one query (results tagged by the scope they came from), read a full entry from a named scope, list everything available, and write a new entry into a chosen scope. Overlapping entries from different scopes both surface; an unreachable source fails loudly with a message naming it rather than silently returning a partial answer. Projects with no `knowledge` config keep working unchanged via a synthesised default source.

#### - [x] Phase 2.1: Make configuration provider-based

Restructures `Config` so `spec`, `plan`, and `knowledge` each name a `provider` and carry a provider-specific `config` block. Restructures the existing `SpecConfig` (its `id_method` moves under `config`), adds a new `PlanConfig`, and adds `KnowledgeConfig`/`SourceConfig`. Adds validation — `provider` is a known value, required `config` fields are non-empty, knowledge scopes are unique — and synthesises a default provider/config for each section when it is absent, so existing projects keep working. This phase touches only the config package; wiring the new values into the workflows is Phase 2.2.

*Technical detail:* [context.md#phase-21](./context.md#phase-21-make-configuration-provider-based)

**Acceptance criteria**:

- [x] `spec`, `plan`, and `knowledge` each round-trip a `provider` plus `config` block through YAML, with `knowledge` carrying multiple independently-configured sources.
- [x] A config with a section absent yields the documented default — `file` provider with `specs`/`plans` directories, and a single default `project` knowledge source.
- [x] An unknown provider, a missing required `config` field, or a duplicate knowledge scope is rejected with a clear validation error.

#### - [x] Phase 2.2: Wire the configured spec and plan directories into the workflows

Replaces the hardcoded `specs/` and `plans/` path segments with the directories from `spec.config.directory` and `plan.config.directory`. Threads the configured directories through `SpecFilePath`/`PlanFilePath` (and the `implement` copy), the `"specs"` literal in `plan/strategy.go`, the spec enumeration in `identifier.go`, and the directory creation at `init`. The directory stays a subpath of the `.spektacular` store root — the single-store architecture is unchanged.

*Technical detail:* [context.md#phase-22](./context.md#phase-22-wire-the-configured-spec-and-plan-directories-into-the-workflows)

**Acceptance criteria**:

- [x] With a non-default `spec.config.directory`, a created spec is written under that directory, and spec enumeration and lookup find it there.
- [x] With a non-default `plan.config.directory`, a plan and its context/research files land under that directory.
- [x] `init` creates the configured spec and plan directories; with default config the directories remain `specs`/`plans` and existing tests pass.

#### - [x] Phase 2.3: Build the `knowledge.Set` orchestrator

Adds the `internal/knowledge` package with `Set`: it resolves each config source into a live store — switching on the source's `provider` field — then fans `Read`, `List`, `Search`, and `Write` across all of them, concatenating results in configured order and tagging each by scope. `List` walks each store's tree recursively via the typed `Store.List`, so entries nested in subdirectories are discovered — not just top-level files. Any operation touching an unreachable source fails fast with an error naming that source — never a partial result. There is no ranking, dedup, or precedence; overlapping entries both surface.

*Technical detail:* [context.md#phase-23](./context.md#phase-23-build-the-knowledge-set-orchestrator)

**Acceptance criteria**:

- [x] Reads, lists, and searches return entries from every configured scope — including entries nested in subdirectories — in one result set.
- [x] Overlapping entries on the same topic surface from both scopes, each correctly tagged.
- [x] A source pointing at an unreachable location makes the operation fail with an error naming that source.
- [x] A write persists into exactly the chosen scope and leaves all other scopes unchanged.

#### - [x] Phase 2.4: Add the `knowledge` CLI command group

Adds a top-level `knowledge` command with `search`, `read`, `list`, `write`, and `sources` subcommands, registered alongside `spec`/`plan`/`implement`. Each builds a `Set` from config and emits JSON through the existing output writer, following the established `--data`/`--schema` command conventions.

*Technical detail:* [context.md#phase-24](./context.md#phase-24-add-the-knowledge-cli-command-group)

**Acceptance criteria**:

- [x] `knowledge search` returns scope-tagged hits with locators and excerpts; `read` returns a full entry; `list` enumerates all scopes; `sources` lists configured scopes and locations; `write` persists an entry.
- [x] Each subcommand emits the documented JSON envelope and uses the standard error format on failure.

### Milestone 3: The planning workflow draws on configured knowledge, and the extension pattern is documented

**What changes**: The planning workflow stops looking only in a hardcoded directory and instead draws on every configured knowledge source through the `knowledge` command, so team and global knowledge inform plans automatically. When an agent proposes capturing a new learning, it must propose a target scope and get explicit user confirmation before anything is written. The README gains a section explaining the `Store` interface contract — including `Search` — and walks through `FileStore` as a worked example, so a developer can implement and register a new backend by following the docs alone.

#### - [x] Phase 3.1: Wire the planning workflow to the `knowledge` commands

Rewrites the discovery step template so the agent searches and reads configured knowledge sources via the `knowledge` commands instead of a hardcoded directory path, and adds the confirmation beat: before writing a learning the agent must propose a target scope and get explicit user confirmation. The `spawn-planning-agents` skill instruction is updated to match.

*Technical detail:* [context.md#phase-31](./context.md#phase-31-wire-the-planning-workflow-to-the-knowledge-commands)

**Acceptance criteria**:

- [x] The rendered discovery step instructs the agent to use the `knowledge` commands and no longer hardcodes the knowledge directory path.
- [x] The rendered discovery step instructs the agent to propose a scope and obtain explicit user confirmation before any knowledge write.
- [x] Existing planning-workflow tests still pass.

#### - [x] Phase 3.2: Document the `Store` extension pattern in the README

Adds a README section presenting the `Store` interface contract — including `Search` — and walking through `FileStore` as the worked example, so a developer can implement and register a new backend by following the docs without reading an existing backend's source. Updates the existing configuration section to document the provider-based `spec`/`plan`/`knowledge` shape, including the configurable spec/plan directories and multiple knowledge sources.

*Technical detail:* [context.md#phase-32](./context.md#phase-32-document-the-store-extension-pattern-in-the-readme)

**Acceptance criteria**:

- [x] The README contains a section presenting the `Store` interface (including `Search`) and `FileStore` as a worked extension example.
- [x] The README's configuration section documents the provider-based `spec`/`plan`/`knowledge` shape, the configurable spec/plan directories, and configuring multiple scoped knowledge sources.

## Open Questions

None. Every design decision was resolved during planning: the integration approach (a `knowledge` CLI command group), the ~250-character excerpt budget, the provider-based config shape (`provider` + `config` per domain, per-source for `knowledge`), the configurable spec/plan directories, the fail-fast-on-unreachable-source behaviour, instruction-gated write confirmation, and the ripgrep-with-native-fallback search backend (discovery confirmed no maintained Go library equivalent exists). The `rg --json` event format is a published, stable contract readable during implementation — not an implementation-time-only uncertainty.

## Out of Scope

- **Non-filesystem backends.** Only the local filesystem backend (`FileStore`) ships. The `Store` interface is designed so a remote/GitHub-hosted backend can be added later — and the README documents how — but no such backend is built here. Tracked for a later spec (spec § Non-Goals).
- **Non-`file` providers and provider-keyed config decoding.** Only the `file` provider ships for `spec`, `plan`, and `knowledge`. The `provider` field and `config` block are designed so a second provider can be added later, but no such provider — and no `provider`-keyed `config` decoding (today `config` decodes straight into the file provider's struct) — is built here.
- **Offline operation and caching for remote sources.** Moot until a remote backend exists; revisit when one is added (spec § Non-Goals).
- **A precedence rule for overlapping entries across scopes.** Both entries surface from search and reads, each tagged with its scope; a "project overrides global" ruleset is deferred to a later spec (spec § Non-Goals).
- **Migration of pre-existing `.spektacular/knowledge/` layouts.** Breaking changes to the on-disk layout are acceptable; no migration tooling or compatibility shim is built. The existing directory is simply adopted as the default `project` source (spec § Non-Goals, § Constraints).
- **Automatic knowledge capture.** Writes are agent-proposed and user-confirmed only. Harvesting learnings without explicit user confirmation is not built (spec § Non-Goals).
- **Threading a multi-store through the workflow `StepCallback` signature.** The rejected Option B. The single-store callback signature is left untouched; agents reach knowledge through the new `knowledge` CLI command group instead. Evidence in `research.md#alternatives-considered-and-rejected`.
- **Global ranking or relevance scoring across stores.** Each store may attach a cheap per-store score, but `knowledge.Set` does no cross-store ranking — it concatenates in configured order and the agent decides. A global ranking layer is not planned.
- **A central search index.** Search is per-store and computed on demand (ripgrep or directory walk); no trigram or inverted index is built or maintained.

## Changelog

### 2026-05-18 — Phase 1.1: Extend the `Store` contract with search

**What was done**: Extended the `store` package's contract: the `Store` interface gained a `Search(query string) ([]Hit, error)` method, `List` was widened from `[]string` to `[]DirEntry` so callers can distinguish files from subdirectories and recurse the tree, and the new `Hit` and `DirEntry` types were added. `FileStore` gained a `scope` field, a `Scope()` accessor, and a changed `NewFileStore(root, scope string)` constructor; `FileStore.Search` is a `nil, nil` stub pending Phase 1.2. Every call site was updated — all `cmd/` workflow stores pass a fixed `"project"` scope, and the `List` consumers in `identifier.go` and `cmd/file.go` consume the typed return.

**Deviations**: None.

**Files changed**:
- `internal/store/store.go`
- `internal/store/store_test.go`
- `internal/steps/spec/identifier.go`
- `internal/steps/spec/identifier_test.go`
- `internal/steps/spec/steps_test.go`
- `internal/steps/plan/steps_test.go`
- `internal/steps/implement/steps_test.go`
- `internal/stepkit/stepkit_test.go`
- `cmd/plan.go`
- `cmd/spec.go`
- `cmd/implement.go`
- `cmd/file.go`

**Discoveries**: `cmd/file.go`'s `file list` subcommand emits a JSON array of bare name strings; to preserve that output contract the new `[]DirEntry` return is projected back to `[]string` before serialization rather than leaking `DirEntry` objects into the CLI envelope. `FileStore.Search` lands as a deliberate `nil, nil` stub so the tree compiles between phases — Phase 1.2 replaces it with the ripgrep + native-fallback implementation; do not treat the stub as final behaviour. No non-`FileStore` implementers of `Store` and no interface mocks exist, so the interface widening was fully contained.

### 2026-05-18 — Phase 1.2: Implement `FileStore.Search` (ripgrep + native fallback)

**What was done**: Added `internal/store/search.go` implementing `FileStore.Search`. It probes for the `rg` binary on `PATH`: when present it shells out to `rg --json` and decodes the event stream, keeping `match` events; when absent it runs a native `filepath.WalkDir` + `bufio.Scanner` line scan. Both paths build scope-tagged `Hit`s with store-relative locators and run every excerpt through one shared `trimExcerpt` helper (whitespace-collapsed, capped at `maxExcerptBytes = 256` on a rune boundary). The stub `Search` was removed from `store.go`.

**Deviations**: The context.md sketch showed `exec.Command("rg", "--json", "--no-heading", query, root)`. The implementation additionally passes `--fixed-strings --ignore-case` so ripgrep's matching is literal and case-insensitive — without these, rg would treat the query as a regex while the native fallback does a literal substring scan, and the two paths would not be equivalent. This realises the plan's stated path-equivalence guarantee rather than departing from it.

**Files changed**:
- `internal/store/search.go` (new)
- `internal/store/store.go`
- `internal/store/search_test.go` (new)

**Discoveries**: The test seam is an unexported `forceFallback bool` field on `FileStore` (per-instance, no global state) rather than a package-level `lookPath` var — tests in `package store` set it directly to exercise the native path on any host. The two search paths are *equivalent on `{Scope, Path, Excerpt}` but not on `Score`*: the rg path sets `Score` to the per-line submatch count, the native fallback leaves it 0 — the plan frames scoring as best-effort, so the equivalence test compares the non-`Score` fields. `rg`'s `match` events emit `data.lines.text` only for UTF-8 lines (non-UTF-8 lines use `bytes` and yield an empty excerpt) — acceptable for human-written knowledge files. ripgrep exit code 1 means "no matches" and is mapped to an empty result, not an error; exit code 2 is a real failure.

### 2026-05-18 — Phase 2.1: Make configuration provider-based

**What was done**: Restructured `internal/config/config.go` so `spec`, `plan`, and `knowledge` each name a `provider` and carry a provider-specific `config` block. `SpecConfig` became `{Provider, Config FileSpecConfig}` with `id_method` moving under `FileSpecConfig` alongside a new `Directory` field; added `PlanConfig`/`FilePlanConfig` and `KnowledgeConfig`/`SourceConfig`/`FileKnowledgeConfig`; added the `ProviderFile` constant and `Default*` constants. `NewDefault` populates the `file` provider with `specs`/`plans` directories and leaves `Knowledge.Sources` empty. `Validate` now checks each section's provider, required `config` fields, and knowledge-scope uniqueness; `KnowledgeConfig.WithDefaults(projectRoot)` synthesises the default `project` source on demand.

**Deviations**: The config package itself stays within Phase 2.1 scope, but the field move (`spec.id_method` → `spec.config.id_method`) forced a one-line compile fix in `cmd/spec.go` (`cfg.Spec.IDMethod` → `cfg.Spec.Config.IDMethod`). Workflow *directory wiring* remains Phase 2.2. `WithDefaults` and the `Default*` constants are defined here but consumed in Phases 2.2/2.3.

**Files changed**:
- `internal/config/config.go`
- `cmd/spec.go`
- `internal/config/config_test.go`
- `internal/project/init_test.go`
- `cmd/init_test.go`
- `cmd/spec_test.go`

**Discoveries**: `FromYAMLFile` unmarshals onto a `NewDefault()` base, so a YAML section that is present-but-partial (e.g. `spec.config.directory` only) keeps the default `provider`/`id_method` via field merge — only a fully-absent section, or an explicitly bad value, exercises the defaults/validation paths. `Validate` runs against the raw config (empty `Knowledge.Sources` is valid); the default `project` source is synthesised later by `WithDefaults`, never written to `config.yaml`. The `id_method` field move broke four test files, not the two anticipated in context.md — `cmd/init_test.go` and `cmd/spec_test.go` also carried `spec.id_method` references and YAML fixtures that needed the nested shape.

### 2026-05-18 — Phase 2.2: Wire the configured spec and plan directories into the workflows

**What was done**: Replaced the hardcoded `specs/`/`plans/` path segments with directories sourced from config. `workflow.Config` gained `SpecDir`/`PlanDir` fields; `SpecFilePath`, `PlanFilePath`, `ContextFilePath`, and `ResearchFilePath` (plus the `implement` copies) gained a leading `dir` parameter; the three workflow `strategy` structs carry the configured directories as fields, set from `cfg` in each `writeStep`. `spec.IdentifierRequest` gained a `SpecDir` field threaded through identifier resolution and spec enumeration (`st.List(specDir)`). `project.Init` now loads config before creating directories and creates the configured spec/plan directories. The `cmd` layer populates `wfCfg` with the directories and resolves them in the three status commands and `implement new`.

**Deviations**: None of substance. context.md anticipated wiring `SpecDir`/`PlanDir` via `workflow.Config` and the strategy structs — done as described. The three `run*Status` commands did not previously call `loadConfig()`; each gained a `loadConfig()` call so it can resolve the configured directory for the path it reports.

**Files changed**:
- `internal/workflow/workflow.go`
- `internal/steps/spec/steps.go`, `internal/steps/spec/strategy.go`, `internal/steps/spec/identifier.go`
- `internal/steps/plan/steps.go`, `internal/steps/plan/strategy.go`
- `internal/steps/implement/steps.go`, `internal/steps/implement/strategy.go`
- `internal/project/init.go`
- `cmd/plan.go`, `cmd/spec.go`, `cmd/implement.go`
- `internal/steps/spec/steps_test.go`, `internal/steps/spec/identifier_test.go`, `internal/steps/plan/steps_test.go`, `internal/steps/implement/steps_test.go`, `internal/project/init_test.go`

**Discoveries**: `ResolveIdentifier` defaults an empty `IdentifierRequest.SpecDir` to `config.DefaultSpecDir` so callers and tests that omit it still resolve against `specs/` — the directory is threaded explicitly through `resolveTimestamp`/`resolveCounter`/`resolveWithPrefix`/`nextCounterFromStore`/`specExists` rather than read from `req` in each. The path helpers stay pure (`dir + "/" + name + ...`, no defaulting) — every real call site passes a non-empty directory because the `cmd` layer always builds `wfCfg` from a loaded config; the schema/status `workflow.New(..., workflow.Config{}, nil, nil)` calls never run step callbacks so their empty `SpecDir`/`PlanDir` is never observed. `Init` loading config before creating directories means an invalid existing `config.yaml` now fails `init` (via `FromYAMLFile`'s `Validate`) — a deliberate, acceptable consequence.

### 2026-05-18 — Phase 2.3: Build the `knowledge.Set` orchestrator

**What was done**: Added the `internal/knowledge` package with `set.go`. `Set` is an ordered collection of scoped stores; `NewSet` applies `KnowledgeConfig.WithDefaults`, resolves each source's `provider` to a `store.FileStore` (relative locations join the project root), and fails fast with an error naming any source whose location is unreachable or whose provider is unknown. `Search`/`Read`/`List`/`Write`/`Sources` fan across the member stores: `Search` concatenates scope-tagged hits in configured order with no ranking or dedup, `List` recursively walks each store's tree (the recursion lives in the knowledge layer; `Store.List` stays one level deep), and `Read`/`Write` delegate to the single store matching the named scope.

**Deviations**: `NewSet` takes a second parameter — `NewSet(cfg config.Config, projectRoot string)` — rather than plan.md § Data Structures' `NewSet(cfg config.Config)`. The plan was internally inconsistent: `NewSet` must supply a project root to `KnowledgeConfig.WithDefaults(projectRoot)` (named in context.md) and to resolve relative source locations. An explicit parameter is the testable, deterministic choice over an internal `os.Getwd()`; Phase 2.4's `cmd/knowledge.go` will pass the working directory.

**Files changed**:
- `internal/knowledge/set.go` (new)
- `internal/knowledge/set_test.go` (new)

**Discoveries**: `WithDefaults` already returns an absolute location for the synthesised default source (it joins `projectRoot`), so `NewSet`'s relative→absolute resolution is a no-op for the default and only affects explicitly-configured relative locations — both paths converge cleanly. The unreachable-source check is `os.Stat` + `IsDir`; Go's `||` short-circuit makes `err != nil || !info.IsDir()` safe when `info` is nil. `Set` holds `provider` and `location` on each `scopedStore` (not just the `store.Store`) so `Sources()` can report them without re-reading config.

### 2026-05-18 — Phase 2.4: Add the `knowledge` CLI command group

**What was done**: Added `cmd/knowledge.go` — a top-level `knowledge` cobra command with `search`, `read`, `list`, `write`, and `sources` subcommands, registered in `cmd/root.go`. Each builds a `knowledge.Set` from the loaded config and the working directory, emits its result as JSON through `output.New`/`WriteResult`, honours a `--schema` persistent flag, and reports failures via the standard `output.WriteError` `{"error":...}` envelope. `read`/`write` take a `{"scope","path"}` `--data` payload; `write` reads the entry body from `--file` or stdin.

**Deviations**: Added lowercase `json:` struct tags to `store.Hit`, `knowledge.Entry`, and `knowledge.SourceInfo`. plan.md's Data Structures showed those structs untagged, but Phase 2.4 is the serialization boundary that first marshals them to agent-facing JSON — tagging the source types yields clean `{"scope":...}` output and avoids duplicating parallel DTO structs in the `cmd` layer. No existing code marshalled these types, so the change is safe.

**Files changed**:
- `cmd/knowledge.go` (new)
- `cmd/root.go`
- `internal/store/store.go` (json tags on `Hit`)
- `internal/knowledge/set.go` (json tags on `Entry`, `SourceInfo`)
- `cmd/knowledge_test.go` (new)

**Discoveries**: `search`/`list` coerce a `nil` result slice to an empty slice before marshalling so the JSON envelope carries `[]` rather than `null` — agents can iterate without a nil guard. The `schemaProp` type has no `Properties` field, so array-of-object output schemas can only declare `{Type:"array"}` (no per-item detail) — consistent with the existing `statusOutputSchema`'s `"steps"` entry. `knowledge` command tests must `t.Chdir` into a temp project because the set resolves `.spektacular/` from `os.Getwd()`; this matches the existing `cmd/*_test.go` pattern and means those tests are not parallel-safe.

### 2026-05-18 — Phase 3.1: Wire the planning workflow to the `knowledge` commands

**What was done**: Rewrote the planning workflow's discovery step template (`templates/steps/plan/02-discovery.md`) so the agent searches and reads configured knowledge sources via `{{config.command}} knowledge search`/`knowledge read` instead of grepping a hardcoded `.spektacular/knowledge/` directory, and added a "Step 5: Capturing a learning" section that gates any `knowledge write` behind running `knowledge sources` and obtaining explicit user confirmation of the target scope and content. Updated the `spawn-planning-agents` skill's "Agent 2: Prior Research" to use `knowledge search`.

**Deviations**: None. The skill file is served raw by the `skill` CLI command (no mustache rendering), so its `knowledge search` reference is written as plain prose without a `{{config.command}}` placeholder — consistent with that file's existing placeholder-free style.

**Files changed**:
- `templates/steps/plan/02-discovery.md`
- `templates/skills/skill_spawn-planning-agents.md`
- `internal/steps/plan/steps_test.go`

**Discoveries**: Skill templates have two consumption paths with different rendering: `internal/agent/skills.go` mustache-renders them with a `{{command}}` placeholder at install time, but `cmd/skill.go`'s `skill` subcommand serves the file raw. `skill_spawn-planning-agents.md` currently carries no placeholders, so command references in it must stay plain text to render correctly on the raw-serve path. The confirmation beat is template prose only — there is no binary-side enforcement of the `knowledge write` gate, mirroring every other approval point in the workflow (verified by a rendered-template `require.Contains` assertion, not an integration test).

### 2026-05-18 — Phase 3.2: Document the `Store` extension pattern in the README

**What was done**: Added an "Extending Storage" section to `README.md` presenting the seven-method `Store` interface (including `Search`), the `DirEntry`/`Hit` types, and `FileStore` as the worked example for implementing and registering a new backend. Rewrote the Configuration section's YAML and prose for the provider-based shape — `spec`/`plan` with `provider` + `config.directory`, and a `knowledge.sources` list showing multiple scoped sources — and updated the Project Structure description to explain that `.spektacular/knowledge/` is the default `project` source among potentially several.

**Deviations**: None against the plan. While rewriting the Configuration section the README's pre-existing `spec.counter: 0` field and the claim that `counter` id_method "persists the latest value in `spec.counter`" were corrected — that field never existed in the `config` struct; counter resolution derives the next number by scanning existing spec files. The correction is incidental to the provider-based rewrite, not separate scope.

**Files changed**:
- `README.md`

**Discoveries**: The README's old configuration docs were already stale before this plan — they documented a `spec.counter` config field that the code never had. Documentation-only phases have no unit-testable surface; the two acceptance criteria are documentation-content checks, verified by inspecting the rendered README rather than by a Go test (no README-content test was invented, per the project's test-pattern guidance).
