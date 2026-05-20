# Research: 000020_context

## Alternatives considered and rejected

### Option B: Thread a multi-store into workflow steps

Change the `StepCallback` signature to carry a multi-store `KnowledgeSet`, and have discovery-type steps assemble knowledge context in Go before emitting their instruction.

**Rejected**: The codebase never assembles context in Go. Steps only emit instruction text; the agent reads files itself with its own tools — `templates/steps/plan/02-discovery.md:7,16` is the entire "context assembly" mechanism. The `StepCallback` signature `func(data Data, out ResultWriter, st store.Store, cfg Config) (string, error)` at `internal/workflow/workflow.go:29` is consumed by all three workflows (`internal/steps/{plan,spec,implement}/steps.go`); changing it is high churn across ~38 step callbacks for no gain, since agents still need search/read access to decide what to load. Fights the grain of the inverted integration model.

### Option C: Templates-only (config + Search, no command)

Add `Search` to `Store` and the `knowledge` config section, but expose nothing — just point the discovery template prose at the multiple configured directories and let the agent grep them itself.

**Rejected**: Reduces "Spektacular supports keyword search across all configured sources" and "cross-source search returns per-store results" (spec § Requirements, § Acceptance Criteria) to the agent's ad-hoc grepping. It cannot satisfy the per-store-tagged-result criterion or the "unreachable source surfaces clearly" criterion (spec acceptance criteria) because there is no Spektacular code in the path to tag results or detect unreachable sources. Smallest change but fails the spec.

### A maintained Go library to replace the ripgrep shell-out

The spec explicitly asked the planning pass to confirm whether a maintained Go search library could replace shelling out to `rg`.

**Rejected**: No such library exists. `sift` is a Go-written *CLI tool*, not a consumable library API, and is effectively unmaintained. `google/codesearch` is an index-based, long-dormant project, not a drop-in recursive grep. Go's standard library has no recursive content-search primitive. The spec's prescribed design — shell out to `rg --json` when available, native Go directory-walk fallback otherwise — therefore stands. See § External references.

### Baking the default knowledge source into `config.NewDefault()`

Considered making `NewDefault()` (`internal/config/config.go:38-48`) return a `Config` whose `Knowledge.Sources` already contains the default `project` source.

**Rejected**: `ToYAMLFile` (`config.go:88-97`) marshals whatever `NewDefault` produces, so every freshly-`init`-ed `config.yaml` would carry a verbose `knowledge:` block the user never asked for. Instead the default `project` source is synthesised at `knowledge.NewSet` time via a `WithDefaults` helper, keeping written config files minimal while existing projects still work with zero config.

### Recursive entry discovery: a separate `Walk()` method, or untyped-`List` error-probing

The knowledge directory is a tree (`.spektacular/knowledge/{architecture,learnings,gotchas}/` plus `conventions.md`), but `Store.List` (`internal/store/store.go:93-110`) is non-recursive and returns bare `[]string` names — `Set.List` calling `store.List("")` would surface only top-level files and the names of subdirectories, never the nested entries. Two fixes were considered and rejected. **(a) Add a separate recursive `Walk()` method to `Store`** — rejected because it adds a second enumeration method when `List` already exists; the user's call is to keep `List(path string)` as the single discovery primitive. **(b) Leave `List` returning `[]string` and have `Set.List` probe directory-ness by calling `List` on each name and treating an error as "it's a file"** — rejected as fragile: it depends on `os.ReadDir`'s ENOTDIR error shape leaking through `FileStore`, conflates "is a file" with "is broken", and bakes a `FileStore`-specific behaviour into the backend-agnostic knowledge layer.

**Chosen**: widen `List` to `List(path string) ([]DirEntry, error)`, where `DirEntry` carries `Name` and `IsDir`. One method, still keyed by a path string, but each child is now typed — so `Set.List` recurses cleanly (descend where `IsDir`) with no error-probing and no backend-specific assumptions. `FileStore.List` already calls `os.ReadDir`, whose `os.DirEntry` exposes `IsDir()`, so the implementation cost is near zero; the cost is a mechanical update of the existing `List` callers (`internal/steps/spec/identifier.go`, `cmd/file.go`) to the typed return.

## Chosen approach — evidence

- The inverted integration model — external agents call this binary and parse its JSON — is established in `internal/agent/{claude,codex,bob}.go` and `cmd/*.go`; the CLI never spawns a subprocess (no `os/exec` anywhere in the repo). A `knowledge` CLI command group fits this model directly.
- The single-store `Store` abstraction is already the project storage contract — `internal/store/store.go:16-30`, constructed at `cmd/plan.go:118`, `cmd/spec.go:186`, `cmd/implement.go:129`, `cmd/file.go:54`. Adding `Search` to it and a scope label to `FileStore` extends the existing contract rather than introducing a parallel one.
- The cobra command pattern for a new command group is well-worn: `cmd/plan.go:39-66` + `init()` at `plan.go:254-266`, registered in `cmd/root.go:62-69`. `cmd/file.go:12-43` shows multi-level subcommand nesting.
- Instruction-gated confirmation is the codebase's universal approval mechanism — `templates/steps/spec/08-verification.md:29-35` ("Propose, then wait for confirmation") and `templates/steps/plan/03-architecture.md:15-19`. `knowledge write` follows it: persist on call, gate in template prose.
- The config-validation pattern to mirror is `SpecConfig.Validate` (`internal/config/config.go:78-85`) called from `Config.Validate` (`:70-75`).
- Test conventions: `internal/store/store_test.go` (testify, `t.TempDir()`, `newTestStore` helper), `internal/config/config_test.go` (YAML-file round-trip), `internal/steps/plan/steps_test.go:45-49` (rendered-template assertion).

## Files examined

- `internal/store/store.go:16-30` — `Store` interface; no `Search`; `FileStore` single `root` field.
- `internal/store/store_test.go:11-14` — `newTestStore` helper pattern; `t.TempDir()`, `errors.Is` sentinel checks.
- `internal/config/config.go:30-85` — `Config` struct, `NewDefault`, `FromYAMLFile`, `Validate`; `SpecConfig.Validate` is the validation pattern.
- `internal/config/config_test.go:19-26` — YAML-file-in-TempDir test pattern.
- `internal/workflow/workflow.go:29,61,103` — `StepCallback` signature; single `Store` threaded through the FSM.
- `cmd/root.go:30-69` — `configFilePath`, `loadConfig`, `dataDir`, top-level command registration in `init()`.
- `cmd/plan.go:39-66,118,173,254-266` — cobra command-group pattern; `NewFileStore` call sites.
- `cmd/spec.go:27-44,186,269` — `commandSchema` `--schema` pattern; `NewFileStore` call sites.
- `cmd/file.go:12-43,54-87` — multi-level subcommand nesting; ad-hoc `NewFileStore` calls.
- `internal/output/writer.go:12-31` — `output.Writer`, `WriteResult`, `WriteError` `{"error": ...}` envelope.
- `internal/project/init.go:23-31` — `init` creates `.spektacular/knowledge/{learnings,architecture,gotchas}`.
- `templates/steps/plan/02-discovery.md:7,16,21` — hardcoded `.spektacular/knowledge/` exploration prose; `spawn-planning-agents` skill reference.
- `templates/steps/spec/08-verification.md:29-35` — the "propose, then wait for confirmation" gate pattern.
- `templates/skills/skill_spawn-planning-agents.md:18` — "Agent 2: Prior Research" references the knowledge directory.
- `internal/steps/plan/steps_test.go:13-49` — `renderStep` harness; rendered-template content assertions.
- `internal/runner/runner.go:268,278` — dead-code prompt constants mentioning knowledge; package has no live importer.
- `README.md:118-156` — Project Structure, knowledge directory, Configuration sections to update.

## External references

- ripgrep — https://github.com/BurntSushi/ripgrep — the chosen fast search backend; provides structured `--json` output that the `FileStore` rg path decodes. Why it mattered: confirms `--json` is a stable, documented contract.
- "Beating grep with Go" — https://healeycodes.com/beating-grep-with-go — shows that a native Go recursive content search is a hand-rolled walk-and-scan, not a library import. Why it mattered: confirms the native fallback must be written, not pulled from a dependency.
- Web search (May 2026) for a maintained Go library equivalent to ripgrep returned only `sift` (a Go CLI tool, not a library; unmaintained) and `google/codesearch` (index-based, dormant). Why it mattered: satisfies the spec's explicit ask to confirm no Go library can replace the shell-out before settling on it.

## Prior plans / specs consulted

- `.spektacular/plans/000014_spektacular_store/plan.md` — established the `Store` interface and `FileStore`. Explicitly listed "adding search" as out of scope and future work; this plan picks that up. Confirms the path-traversal guard and the `StepCallback`-carries-store design.
- `.spektacular/specs/000020_context.md` — the source spec. Technical Approach mandates: reuse `store.Store`, add `Search`; per-store generic hits; agent consumes hits; one store per scope; ship only the filesystem backend; `FileStore.Search` prefers ripgrep with a Go fallback; entry structure is a file-format convention.

## Open assumptions

- **`rg --json` event schema is stable** — the `FileStore` rg path decodes `match`-type events with `data.path.text`, `data.lines.text`, `data.line_number`. Assumed stable across ripgrep versions on users' machines. If a future `rg` changes the schema, the rg path breaks but the native fallback still works. If implementation finds the schema differs from expectation, adjust the decoder — no need to STOP.
- **Knowledge directories are small** — the native fallback walk-and-scan is assumed acceptable because knowledge stores hold human-written notes, not large codebases. If a configured source is unexpectedly huge, search may be slow on machines without `rg`; not a correctness issue.
- **Relative source `Location` resolves against the project root** — assumed; the default `project` source uses a relative path, team/global use absolute. If this turns out ambiguous in practice (e.g. CWD vs project root differ), STOP and ask the user.
- **One store per scope** — the spec says scopes are labels on configured stores and scope uniqueness is validated. If a future need arises for multiple stores at one scope, the `Set` design would need revisiting; out of scope here.

## Rehydration cues

- Re-read the spec: `.spektacular/specs/000020_context.md`.
- Re-read this plan: `.spektacular/plans/000020_context/plan.md` and `context.md`.
- Re-read the `Store` contract: `internal/store/store.go` (and `store_test.go` for conventions).
- Re-read the config pattern: `internal/config/config.go` (`SpecConfig.Validate` is the validation model).
- Re-read the cobra command pattern: `cmd/plan.go` + `cmd/root.go` `init()`.
- Re-read the confirmation-gate pattern: `templates/steps/spec/08-verification.md`.
- Skill guidance: `go run . skill spawn-implementation-agents` for per-phase agent orchestration.
- Build/test: `make build`, `make test`, `go test ./...`.
