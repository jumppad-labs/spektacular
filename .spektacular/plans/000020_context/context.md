# Context: 000020_context

## Current State Analysis

The starting point for this feature:

- **`Store` interface** — `internal/store/store.go:16-30` declares `Root/Read/Write/Delete/List/Exists`. No `Search`. `FileStore` struct at `store.go:34-36` has a single `root string` field; `NewFileStore(root string) *FileStore` at `store.go:39-41`. `FileStore.abs` (`store.go:49-56`) resolves relative paths and rejects traversal. `ErrNotFound` at `store.go:12`. `List` (`store.go:26-27` decl, `store.go:93-110` impl) is non-recursive — it returns the bare `[]string` names of a directory's direct children via `os.ReadDir`, with no way to tell a file from a subdirectory; this is why `Set.List` cannot just call `store.List("")` and why `List` is widened to typed entries.
- **Single-store assumption everywhere** — every workflow threads exactly one `Store`, always rooted at `<cwd>/.spektacular`. `NewFileStore` is called at `cmd/plan.go:118,173`, `cmd/spec.go:186,269`, `cmd/implement.go:129,184`, and `cmd/file.go:54,62,75,87`. All pass `dataDir` from `cmd/root.go:54-60`. No multi-store or scope concept exists.
- **`StepCallback`** — `internal/workflow/workflow.go:29`: `func(data Data, out ResultWriter, st store.Store, cfg Config) (string, error)`. The FSM passes the single `w.store` (`workflow.go:103`). This signature is **not changed** by this plan.
- **Knowledge directory is never read by Go** — `.spektacular/knowledge/` appears only as instruction prose in `templates/steps/plan/02-discovery.md:7,16` (and the `spawn-planning-agents` skill at `templates/skills/skill_spawn-planning-agents.md:18`). The agent explores it with its own tools. The `internal/runner` package mentions knowledge in prompt constants but is dead code (no non-test importer).
- **Config** — `internal/config/config.go:30-35` `Config` struct (`Command/Agent/Debug/Spec`). `NewDefault()` at `:38-48`; `FromYAMLFile` at `:51-67`; `Config.Validate` at `:70-75`; `SpecConfig.Validate` at `:78-85` is the pattern to mirror. `SpecConfig` today is a flat `{ IDMethod string }` (`config.go:24-27`) — this plan restructures it to `{ Provider string; Config FileSpecConfig }`. Loaded only from `<cwd>/.spektacular/config.yaml` via `cmd/root.go:30-36` (`configFilePath`) and `cmd/root.go:41-50` (`loadConfig`, returns `NewDefault()` if absent). No global/user-level config.
- **Hardcoded spec/plan directories** — the `specs/` and `plans/` path segments are string literals: `internal/steps/spec/steps.go:13` (`SpecFilePath` → `"specs/" + name + ".md"`), `internal/steps/plan/steps.go:13,18,23` (`PlanFilePath`/`ContextFilePath`/`ResearchFilePath` → `"plans/" + name + ...`), `internal/steps/implement/strategy.go:13,18,23` (a deliberate copy of the plan helpers, kept to avoid a cross-package import cycle), `internal/steps/plan/strategy.go:18` (`"specs"` joined in `PathVars`), `internal/steps/spec/identifier.go:177` (`st.List("specs")` for spec enumeration). `internal/project/init.go:25-26` creates the `plans`/`specs` directories. All are subpaths of the `.spektacular` store root; each workflow's store is `store.NewFileStore(dataDir)` rooted at `.spektacular` (`cmd/plan.go:118,173`, `cmd/spec.go:186,269`, `cmd/implement.go:129,184`).
- **Config reaches steps via `workflow.Config`** — `internal/workflow/workflow.go:29` `StepCallback` carries a `workflow.Config` (distinct from `config.Config`); `cmd/plan.go:117` builds it as `workflow.Config{Command: cfg.Command, DryRun: dryRun}`. `SpecFilePath`/`PlanFilePath` are also called directly from the `cmd` layer (`cmd/spec.go:303`, `cmd/plan.go:215`, `cmd/implement.go:114,226`), which has `loadConfig()` in scope. `stepkit.PathStrategy.PathVars(instanceName, storeRoot string)` is implemented by the per-workflow `strategy` struct (`internal/steps/spec/strategy.go`, `internal/steps/plan/strategy.go`).
- **`init`** — `internal/project/init.go:23-31` creates `.spektacular/knowledge/` plus `knowledge/{learnings,architecture,gotchas}`. This directory becomes the default `project` source; init needs no change.
- **CLI command pattern** — cobra; root in `cmd/root.go`, top-level commands registered in `cmd/root.go:62-69` `init()`. Each command is a package-level `var xCmd = &cobra.Command{...}` with a file-local `init()` adding flags/subcommands; see `cmd/plan.go:39-66` + `init()` at `plan.go:254-266`. Conventions: `RunE`, a `--schema` short-circuit emitting `commandSchema{Input,Output}` (`spec.go:27-44`), JSON `--data` input, output via `output.New(cmd.OutOrStdout(), globalFields)`, errors via `output.WriteError`. `cmd/file.go:12-43` shows 2-level subcommand nesting.
- **Output** — `internal/output/writer.go:12-26` `output.Writer`; `WriteResult` marshals indented JSON; `WriteError` emits `{"error": ...}` (`writer.go:29-31`).
- **No subprocess execution** — `os/exec` is used nowhere in the repo. The `Search` ripgrep path is the first use of `exec.Command`.
- **Test conventions** — `testify/require`; `t.TempDir()` wrapped in a `newTestStore(t)` helper (`store_test.go:11-14`); `TestSubject_Behaviour` naming; `errors.Is` + `require.True` for sentinels (`store_test.go:37`). Plan step tests use a `renderStep(t, cb)` harness and `require.Contains` on rendered template text (`internal/steps/plan/steps_test.go:13-49`).

## Per-Phase Technical Notes

### Phase 1.1: Extend the Store contract with search

- `internal/store/store.go:16-30` — add `Search(query string) ([]Hit, error)` to the `Store` interface.
- `internal/store/store.go` — add the `Hit` struct: `Scope, Path, Excerpt string; Score float64`.
- `internal/store/store.go:26-27` — widen `List` from `List(path string) ([]string, error)` to `List(path string) ([]DirEntry, error)`; add the `DirEntry` struct: `Name string; IsDir bool`. `FileStore.List` (`store.go:93-110`) already calls `os.ReadDir`, whose `os.DirEntry` values expose `IsDir()` — map them straight to `store.DirEntry` instead of taking `.Name()` alone. `List` stays non-recursive (one directory level); recursion is the caller's job, now possible because each entry is typed.
- Update every existing `List` caller to the typed return — both consume bare name strings today: `internal/steps/spec/identifier.go` (spec enumeration via `List`/`Exists`) and the `cmd/file.go` file-`list` subcommand (`file.go:87`). The change is mechanical — read `e.Name` instead of `e`.
- `internal/store/store.go:34-41` — add a `scope string` field to `FileStore`; change `NewFileStore` to `NewFileStore(root, scope string) *FileStore`. Add a `Scope()` accessor for symmetry with `Root()`.
- Update all `NewFileStore` call sites to pass scope `"project"`: `cmd/plan.go:118,173`, `cmd/spec.go:186,269`, `cmd/implement.go:129,184`, `cmd/file.go:54,62,75,87`.
- `internal/store/store_test.go:11-14` — update `newTestStore` helper to pass a scope; `TestList_ReturnsEntryNames:53-60` updated to assert on `DirEntry.Name`/`IsDir`; `TestRoot_ReturnsAbsolutePath:79-83` unaffected.
- This phase may add a stub `Search` returning `nil, nil` only if needed to keep the tree compiling between phases; prefer landing 1.1 + 1.2 together so `Search` is never a stub on `main`.

**Complexity**: Low
**Token estimate**: ~12k
**Agent strategy**: Single agent, sequential. Mechanical signature change across 8 call sites plus the interface/struct edit.

### Phase 1.2: Implement FileStore.Search (ripgrep + native fallback)

- New file `internal/store/search.go` — `FileStore.Search`:
  - Probe `exec.LookPath("rg")`. If found, run the ripgrep path; else the native fallback.
  - **ripgrep path**: `exec.Command("rg", "--json", "--no-heading", query, f.root)`; decode stdout line-by-line as JSON objects; keep events of `type == "match"`; from each, take `data.path.text` (make relative to `f.root`), `data.lines.text` for the excerpt, `data.line_number`. `rg` exit code 1 means "no matches" — treat as empty, not error; exit code 2 is a real error.
  - **native fallback**: `filepath.WalkDir(f.root, ...)`; skip directories; for each file open and `bufio.Scanner` line scan; case-insensitive substring match of `query`; on match build a `Hit` with the line as excerpt.
  - Both paths: relative `Path`, `Scope: f.scope`, excerpt passed through one shared `trimExcerpt(s string) string` helper that caps at the ~250-char budget (`maxExcerptBytes = 256`), collapsing whitespace and trimming on a rune boundary. `Score` left 0 in the fallback; the rg path may set a simple score (e.g. match count) if cheap.
- Test seam: an unexported package var `lookPath = exec.LookPath` (or a `forceFallback bool` field on `FileStore` used only in tests) so `search_test.go` can exercise the fallback regardless of the host.
- New `internal/store/search_test.go`: fixture dir via `t.TempDir()` with a few files; assert excerpt budget, scope tagging, locator round-trips through `Read`, empty-result-not-error on no match; an rg-path test guarded by `if _, err := exec.LookPath("rg"); err != nil { t.Skip(...) }`; a path-equivalence test comparing forced-fallback hits to rg hits when rg is present.

**Complexity**: High
**Token estimate**: ~30k
**Agent strategy**: Parallel analysis (one agent confirms the `rg --json` event schema and exit-code semantics, one drafts the fallback walk), sequential integration behind the single `Search` method, then verification.

### Phase 2.1: Make configuration provider-based

- `internal/config/config.go:30-35` — restructure `Config`: keep `Command/Agent/Debug`, change `Spec` to the new `SpecConfig`, add `Plan PlanConfig \`yaml:"plan"\`` and `Knowledge KnowledgeConfig \`yaml:"knowledge"\``.
- Restructure `SpecConfig` (`config.go:24-27`) to `{ Provider string \`yaml:"provider"\`; Config FileSpecConfig \`yaml:"config"\` }`; add `FileSpecConfig{ Directory, IDMethod string }`. The existing `IDMethod` field moves from `SpecConfig` into `FileSpecConfig`.
- Add `PlanConfig{ Provider string; Config FilePlanConfig }` and `FilePlanConfig{ Directory string }`.
- Add `KnowledgeConfig{ Sources []SourceConfig }`, `SourceConfig{ Scope, Provider string; Config FileKnowledgeConfig }`, `FileKnowledgeConfig{ Location string }` — all with yaml tags.
- Add `const ProviderFile = "file"` (this replaces the `SourceTypeFile` constant the original plan named).
- `config.go:38-48` `NewDefault()` — set `Spec` to `{Provider: ProviderFile, Config: {Directory: "specs", IDMethod: SpecIDMethodTimestamp}}` and `Plan` to `{Provider: ProviderFile, Config: {Directory: "plans"}}`. Leave `Knowledge.Sources` empty (zero value) so `ToYAMLFile` output stays clean; default knowledge-source synthesis lives in a helper `KnowledgeConfig.WithDefaults(projectRoot string) KnowledgeConfig` consumed by `knowledge.NewSet` (Phase 2.3).
- Validation — extend `Config.Validate` (`config.go:70-75`) to call `SpecConfig.Validate`, `PlanConfig.Validate`, and `KnowledgeConfig.Validate`. Each checks `Provider == ProviderFile` (else a clear error) and that required `Config` fields are non-empty (`Directory` for spec/plan, `Location`/`Scope` per source); `KnowledgeConfig.Validate` also checks scopes are unique. The `id_method` value check from the current `SpecConfig.Validate` (`config.go:78-85`) moves into `FileSpecConfig` validation. `config.go:80` `switch c.IDMethod` becomes `switch c.Config.IDMethod`.
- Only one provider exists, so the `Config` field is a plain typed struct decoded directly by `yaml.v3` — no `provider`-keyed decoding (out of scope).
- `internal/config/config_test.go` — round-trip tests for the provider-based `spec`/`plan`/`knowledge` sections, default synthesis when a section is absent, and validation-failure tests (unknown provider, empty `directory`, duplicate scope); follow the `t.TempDir()` + YAML-file pattern at `config_test.go:19-26`. Existing `SpecConfig` tests update to the nested shape.

**Complexity**: Low
**Token estimate**: ~14k
**Agent strategy**: Single agent, sequential.

### Phase 2.2: Wire the configured spec and plan directories into the workflows

- `internal/steps/spec/steps.go:11-13` — `SpecFilePath` takes the spec directory: `SpecFilePath(dir, name string) string` → `dir + "/" + name + ".md"`. Update all callers (`steps.go:76,141`, `cmd/spec.go:303`).
- `internal/steps/plan/steps.go:11-23` — `PlanFilePath`/`ContextFilePath`/`ResearchFilePath` take the plan directory. Update callers (`steps.go:187,235`, `cmd/plan.go:215`).
- `internal/steps/implement/strategy.go:9-23` — the copied `PlanFilePath`/`ContextFilePath`/`ResearchFilePath` take the plan directory; keep the deliberate copy, just thread the arg. Update callers (`strategy.go:32`, `cmd/implement.go:114,226`).
- `internal/steps/plan/strategy.go:18` — replace the `"specs"` literal in `PathVars` with the configured spec directory; `internal/steps/spec/identifier.go:177` — `st.List("specs")` uses the configured spec directory.
- Threading the directories to the call sites: add `SpecDir` and `PlanDir` fields to `workflow.Config` (`workflow.go`) beside `Command`, populated from `cfg.Spec.Config.Directory`/`cfg.Plan.Config.Directory` where `wfCfg` is built (`cmd/plan.go:117`, `cmd/spec.go`, `cmd/implement.go`); step callbacks already receive `workflow.Config`. The per-workflow `strategy` struct carries the configured dirs as fields so `PathVars` can use them — set when the workflow is constructed. Direct `cmd`-layer calls read the dirs from `loadConfig()`, already in scope.
- `internal/project/init.go:23-31` — create the configured spec and plan directories (defaulting to `specs`/`plans`) instead of the literals; `init` resolves them from config (or `NewDefault` when no config file exists).
- The workflow store stays `store.NewFileStore(dataDir)` rooted at `.spektacular`; only the subpath segment is configurable.
- Tests — `internal/steps/spec` and `internal/steps/plan` step tests render with a non-default directory and assert the resulting paths; a `config`-driven test asserts `init` creates the configured directories. Default config keeps `specs`/`plans` so existing tests pass with a mechanical signature update.

**Complexity**: Medium
**Token estimate**: ~18k
**Agent strategy**: Single agent, sequential — a mechanical-but-wide signature change across the spec/plan/implement step packages, the strategies, `workflow.Config`, the `cmd` layer, and `init`.

### Phase 2.3: Build the knowledge.Set orchestrator

- New package `internal/knowledge`, file `set.go`:
  - `type scopedStore struct { scope string; store store.Store }`; `type Set struct { sources []scopedStore }`.
  - `NewSet(cfg config.Config) (*Set, error)` — apply `KnowledgeConfig.WithDefaults`, then for each `SourceConfig` switch on `Provider`: `ProviderFile` → resolve `Config.Location` (relative paths against project root) and `store.NewFileStore(location, scope)`; unknown provider → error. Validate reachability: the directory exists and is a directory (`os.Stat`); otherwise return an error naming the source.
  - `Search(query string) ([]store.Hit, error)` — iterate sources in order, call `store.Search`, concatenate; any store error → wrap with the scope name and return.
  - `Read(scope, path string) ([]byte, error)` / `Write(scope, path string, content []byte) error` — find the `scopedStore` by scope (error if no such scope) and delegate.
  - `List() ([]Entry, error)` — recursively walk each store: call `store.List("")`, and for every `DirEntry` with `IsDir` true descend with a further `store.List` of that subpath, collecting only file locators; concatenate across sources. `Entry{ Scope, Path string }` where `Path` is the file locator relative to the store root (e.g. `architecture/initial-idea.md`). The recursion lives in the knowledge layer; `Store.List` itself stays one level deep.
  - `Sources() []SourceInfo` — `SourceInfo{ Scope, Provider, Location string }`.
- New `internal/knowledge/set_test.go` — stand up 2+ `FileStore`s at distinct `t.TempDir()`s; assert fan-out across scopes, overlapping-entry tagging, unreachable-source error names the source, scoped write isolation.

**Complexity**: Medium
**Token estimate**: ~22k
**Agent strategy**: 2 parallel agents — one implements `set.go`, one writes `set_test.go` against the agreed signatures from plan.md § Data Structures.

### Phase 2.4: Add the knowledge CLI command group

- New file `cmd/knowledge.go`:
  - `knowledgeCmd` + subcommands `search`, `read`, `list`, `write`, `sources`, following the cobra pattern of `cmd/plan.go:39-66` and its `init()` at `plan.go:254-266`.
  - Each `RunE`: honour `--schema` (emit `commandSchema`), `loadConfig` (`cmd/root.go:41-50`), `knowledge.NewSet`, run the operation, write via `output.New(cmd.OutOrStdout(), globalFields)`; errors via `output.WriteError`.
  - `search <query>` → `{"hits": [...]}`; `read --data '{"scope","path"}'` → `{"scope","path","content"}`; `list` → `{"entries": [...]}`; `sources` → `{"sources": [...]}`; `write --data '{"scope","path"}' --file <content>` → `{"scope","path"}`. `write` reads body content via `--file`/`--stdin` like existing commands (`cmd/spec.go` input helpers).
- Register `knowledgeCmd` in `cmd/root.go:62-69` `init()`.
- New `cmd/knowledge_test.go` — assert each subcommand's JSON envelope, following existing command-test style.

**Complexity**: Medium
**Token estimate**: ~20k
**Agent strategy**: 2 parallel agents — one builds the command file, one writes command tests.

### Phase 3.1: Wire the planning workflow to the knowledge commands

- `templates/steps/plan/02-discovery.md:7,16` — replace the hardcoded `.spektacular/knowledge/` exploration prose with instructions to run `{{config.command}} knowledge search <query>` and `{{config.command}} knowledge read --data '{"scope":...,"path":...}'`; mention that results are scope-tagged. `:21` already references the `spawn-planning-agents` skill — keep.
- Add a confirmation beat to the discovery template: before any `{{config.command}} knowledge write`, the agent must propose a target scope (from `knowledge sources`) and the content, and obtain explicit user confirmation — phrased like the spec-verify gate in `templates/steps/spec/08-verification.md:29-35`.
- `templates/skills/skill_spawn-planning-agents.md:18` — update "Agent 2: Prior Research" to use `knowledge search` instead of the hardcoded directory.
- `internal/steps/plan/steps_test.go` — add a rendered-template assertion (model on `TestArchitectureStepContainsOptionsAndAgreementBeat` at `steps_test.go:45-49`) checking the discovery instruction contains `knowledge` commands and the confirmation beat.

**Complexity**: Low
**Token estimate**: ~12k
**Agent strategy**: Single agent, sequential.

### Phase 3.2: Document the Store extension pattern in the README

- `README.md` — add a new section (after "Project Structure", around `:118-134`) titled e.g. "Extending Storage", presenting the `Store` interface (all methods including `Search`) and `FileStore` as the worked example: how to implement the interface, how `NewSet` resolves a `type` to a backend, and what a new backend must do to be registered.
- `README.md:134` — update the knowledge-directory description to explain multiple configured sources rather than one directory.
- `README.md:138-156` — rewrite the Configuration section's YAML for the provider-based shape: `spec`/`plan` with `provider` + `config.directory`, and a `knowledge.sources` example showing project/team/global scopes each with `provider` + `config.location`.

**Complexity**: Low
**Token estimate**: ~10k
**Agent strategy**: Single agent, sequential.

## Testing Strategy

Unit tests using `testify/require`, `t.TempDir()`, and `TestSubject_Behaviour` naming, placed beside the code under test (`store/search_test.go`, `knowledge/set_test.go`, `config/config_test.go`, `cmd/knowledge_test.go`).

Load-bearing assertions:
- **Search path equivalence** — forced-fallback hits equal ripgrep hits for the same query/fixture; the rg-path test skips cleanly when `rg` is absent so the suite passes on any machine.
- **Excerpt budget** — every excerpt is within `maxExcerptBytes`.
- **Fan-out** — `Set` reads/lists/searches return entries from every configured scope, including entries nested in subdirectories (`Set.List` recurses the tree); overlapping entries surface from both scopes, each tagged.
- **Fail-fast** — an unreachable source produces an error naming that source, never a partial result.
- **Scoped write isolation** — a write lands only in the chosen scope.
- **Config round-trip** — the provider-based `spec`/`plan`/`knowledge` sections survive YAML marshal/unmarshal; default synthesis fires when absent; invalid configs (unknown provider, empty directory, duplicate scope) are rejected.
- **Spec/plan directory wiring** — a non-default `spec.config.directory`/`plan.config.directory` makes specs and plans write, enumerate, and resolve under that directory, and `init` creates it.
- **CLI envelopes** — each `knowledge` subcommand emits the documented JSON shape.
- **Template content** — the rendered discovery step contains the `knowledge` commands and the confirmation beat.

No new Harbor end-to-end test; the agent-facing confirmation behaviour is template prose, verified by the rendered-template assertion.

## Project References

- Build & test: `make build`, `make test`, `go test ./...`, `make lint` (`go vet ./...`).
- Spec: `.spektacular/specs/000020_context.md`.
- Prior plan: `.spektacular/plans/000014_spektacular_store/plan.md` — established `Store`/`FileStore`.
- Key files: `internal/store/store.go`, `internal/config/config.go`, `cmd/root.go`, `cmd/plan.go`, `internal/output/writer.go`, `internal/project/init.go`, `templates/steps/plan/02-discovery.md`, `README.md`.

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|-------------|----------------|
| Low | ~10k | Single agent, sequential |
| Medium | ~25k | 2-3 parallel agents |
| High | ~50k+ | Parallel analysis, sequential integration |

Phase tiers: 1.1 Low, 1.2 High, 2.1 Low, 2.2 Medium, 2.3 Medium, 2.4 Medium, 3.1 Low, 3.2 Low. The only High phase is 1.2 (two execution paths with an equivalence contract); land Phases 1.1 and 1.2 together so `Search` is never a stub on `main`.

## Migration Notes

No data migration. Breaking changes to the on-disk knowledge layout are explicitly acceptable (spec § Constraints). Existing projects with no `knowledge` config section keep working unchanged: `knowledge.NewSet` synthesises a single default `project` source pointing at the `init`-created `.spektacular/knowledge` directory. No compatibility shim, no migration tooling.

The `config.yaml` format also changes: `spec.id_method` moves under `spec.config.id_method`, and `spec`/`plan`/`knowledge` gain the `provider`/`config` structure. This is a breaking change to the config file. Consistent with the plan's stance on breaking changes, no migration shim is provided — a project either updates `config.yaml` by hand or removes the stale section and relies on the synthesised defaults.

## Performance Considerations

Search is computed on demand with no index. `ripgrep` is the fast path and is used whenever `rg` is on `PATH`; the native Go fallback (directory walk + line scan) is slower on large trees but acceptable for knowledge directories, which are small by nature. Excerpts are capped at ~250 characters so an agent can scan many hits cheaply — the token-efficiency success metric. `knowledge.Set` fans out sequentially across sources; the number of configured sources is small (typically 1–3 scopes), so sequential iteration is not a bottleneck.
