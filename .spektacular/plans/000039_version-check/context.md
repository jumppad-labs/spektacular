---
created_date: "2026-07-31"
status: completed
closed_date: "2026-07-31"
---

# Context: 000039_version-check

## Current State Analysis

- No version is recorded anywhere at install time. `internal/project/init.go:17-117` scaffolds `.spektacular/` (dirs, `config.yaml` if absent, `.gitignore` from the embedded FS, knowledge READMEs) and `cmd/init.go:21-51` (`runInit`) rewrites `config.yaml` with the agent set (`:41-45`) then calls the agent's `Install`. Nothing version-stamped; `config.Config` (`internal/config/config.go:110-120`) has no version field.
- The binary's version lives in package `cmd`: `cmd/root.go:18-23` — `var version = "0.1.0"; sha = ""`, injected via ldflags `-X github.com/jumppad-labs/spektacular/cmd.version=...` from `Makefile:9-10` (`VERSION := 0.15.1`, `build` target only) and `dagger/main.go:149-153` (CI; semver from PR label, `dagger/main.go:501-545`). `versionString()` (`cmd/root.go:37-42`) appends ` (sha)` when set. Cobra's built-in `--version` prints plain text — there is no `version` subcommand.
- Dev builds (`go run .`, `make cross`, harbor targets) carry the `0.1.0` default. Commit de84ae6 ("Fix version number") fixed ldflags previously pointing at `main.version` (nonexistent — linker silently no-ops), so every earlier release reported `0.1.0`.
- Skill installation: `internal/agent/skills.go:24-65` — the four workflow skills (`workflowSkills`, `:25-30`) render byte-identically for all three agents (claude `.claude/skills`, bob `.bob/skills`, codex `.agents/skills`) via `mustache.Render` with the single key `{"command": cfg.Command}` (`:47`). No partials wired. `cfg.Command` defaults to `"spektacular"`; this repo's config sets `go run .`.
- Output conventions: `internal/output/writer.go` — `Write`/`WriteResult` inject the `"error": false` discriminant for object results (`:104-114`); `ErrorResponse` (`:45-52`) with `NewError/WithResource/WithNextAction` builders is the agent-actionable error shape; `runRoot` (`cmd/root.go:57-82`) is the single envelope/exit-code choke point.
- Envelope contract tests: `cmd/root_test.go:92` (stdout-only), `:219` (discriminant + exit code), `:332` (printed exactly once); `:214-218` documents plain-text exceptions (`init`, `file read`).
- Gitignore state: repo `.gitignore:43-44` ignores only `.spektacular/debug/`; `templates/.spektacular/.gitignore` ignores `*.tmp`, `*.log`, `tmp/` etc. A new `.spektacular/version` file is therefore committed by default — the agreed, team-visible behavior.

## Per-Phase Technical Notes

### Phase 1.1: Record the installing version at init

**File changes:**

- `cmd/version.go` (new, shared with Phase 2.1) — add `versionFilePath(dataDir string) string` returning `filepath.Join(dataDir, "version")` (i.e. `.spektacular/version`), and `writeVersionFile(path, v string) error` writing `v + "\n"` with mode 0644. Kept in package `cmd` so the ldflags-injected `version` var (`cmd/root.go:18-23`) is directly accessible — do NOT move the var out of package `cmd`; commit de84ae6 showed the linker silently no-ops on a wrong `-X` path.
- `cmd/init.go:41-50` — in `runInit`, after the config rewrite (`cfg.ToYAMLFile`, lines 43-45), write the version file at `filepath.Join(cwd, ".spektacular", "version")` (cwd is already in scope; mirrors `dataDir()` at `cmd/root.go:209-215`) using the package `version` var. Add one plain-text output line in the existing style of `cmd/init.go:47-48` (e.g. `  Version:  .spektacular/version (0.1.0)`). Init deliberately keeps plain-text output — it is a documented envelope exception (`cmd/root_test.go:214-218`).
- `cmd/init_test.go` — extend `TestInit_Claude` (`:14-45`) to assert `.spektacular/version` exists with expected content; add `TestInit_RewritesStaleVersionFile`: pre-write a bogus value (e.g. `9.9.9`), run init, assert the file now holds the current binary's version. Extend `TestInit_Idempotent` (`:164-184`) to cover the version file. Keep the oracle independent: the dev default is the literal `"0.1.0"` — assert against the literal where practical rather than re-deriving through `versionString()`.

**Complexity**: Low
**Token estimate**: ~15k
**Agent strategy**: Single agent, sequential execution.

### Phase 2.1: Add the version check command

**File changes:**

- `cmd/version.go` (extends Phase 1.1 file):
  - `versionCmd` bare parent group modeled on `cmd/changelog.go:5-8` (`Use: "version"`, no RunE).
  - `VersionCheckResult` struct: `Status`, `InstalledVersion` (omitempty), `CurrentVersion`, `Action` (omitempty) with JSON tags per plan.md § Data Structures & Interfaces. No `Error` field — the discriminant is injected by `internal/output/writer.go:104-114`.
  - Pure classification function `classifyVersion(recorded, current string) (status, installed string)`: trim whitespace from `recorded`; empty → `missing`; equal to `current` → `match`; else `mismatch`. No package-var reads inside — both sides are parameters, keeping unit-test oracles independent.
  - `versionCheckCmd` `RunE`: `--schema` short-circuit first, modeled on `cmd/skill.go:28-47` using `schemaProp`/`schemaObj`/`commandSchema` (`cmd/spec.go:28-46`); then `dataDir()` (`cmd/root.go:209-215`) → `os.ReadFile`; `os.IsNotExist` → treat as missing; any other read error → `output.NewError("version_file_unreadable", ...).WithResource(path).WithNextAction("Check file permissions on .spektacular/version.")` (genuine fault, exit 1 via `runRoot`). Success: `output.New(cmd.OutOrStdout(), globalFields).WriteResult(result)` per `cmd/skill.go:59-64`. Action text on mismatch/missing: the installed Spektacular files are out of date (or have no recorded version) — ask the user to re-run init; include the configured agent from `loadConfig()` (`cmd/root.go:196-205`) in the suggested command when available.
  - `init()` registers `versionCmd.AddCommand(versionCheckCmd)` and the `--schema` flag (pattern: `cmd/skill.go:105-108`).
- `cmd/root.go:238-243` — `rootCmd.AddCommand(versionCmd)`.
- `cmd/version_test.go` (new) — unit tests for `classifyVersion` (exact match, whitespace-padded match, differing, empty, whitespace-only); command tests via `runRootCmd` (`cmd/root_test.go:68-74`) + `t.TempDir()` + `t.Chdir` for all three states asserting exit 0, `"error": false`, correct `status`/versions, `action` present only on stale states; a no-mutation test capturing file bytes before/after a mismatch invocation; a `--schema` smoke test. Models: `cmd/error_response_test.go:30-87`, `cmd/artifacts_test.go:53-70`.
- `cmd/root_test.go:92,219,332` — add the `version check` family to the three envelope contract tables (stdout-only/stderr-empty, discriminant + exit code, printed exactly once). Watch-point: if the table harness does not accept a two-word command family without restructuring, STOP and ask.

**Complexity**: Low–Medium
**Token estimate**: ~25k
**Agent strategy**: Single agent, sequential execution (command, registration, and tests are one coherent unit).

### Phase 3.1: Add the version-check preamble to workflow skills

**File changes:**

- `templates/skills/workflows/spek-new/SKILL.md:5`, `templates/skills/workflows/spek-plan/SKILL.md:5`, `templates/skills/workflows/spek-implement/SKILL.md:5`, `templates/skills/workflows/spek-knowledge/SKILL.md:5` — insert the identical ~5-line preamble block immediately after the closing `---` of the frontmatter (frontmatter must remain at byte 0) and before the first content. Block content: run `{{command}} version check` before any other command; on `status: "match"` continue with no version-related output; on `"mismatch"`/`"missing"` relay the `action` message to the user and wait for their decision (re-run init) before continuing; never modify installed files yourself. Use `{{command}}` only — never a rendered command (guarded by `cmd/init_test.go:39,70,85,113` and `internal/agent/instruction_surface_test.go:21-48`).
- `internal/agent/claude_test.go` (pattern at `:38`) / `bob_test.go` / `codex_test.go` — add an assertion that each installed `SKILL.md` contains the substituted version-check instruction (e.g. contains `version check` after rendering), so dropping the preamble from any template fails the suite. One assertion mechanism; don't duplicate beyond the existing per-agent test structure.
- Dogfood refresh: run `go run . init claude` in this repository so `.claude/skills/*/SKILL.md` pick up the preamble, AGENTS.md sections re-render, and `.spektacular/version` is written (records the dev default `0.1.0`, matching the `go run .` binary — silent thereafter). Review the resulting diff before committing.

**Complexity**: Low
**Token estimate**: ~15k
**Agent strategy**: Single agent, sequential execution (four identical template edits plus one test touch; parallelism buys nothing).

## Testing Strategy

- **Phase 1.1** — extend the existing init suite (`cmd/init_test.go`, all `t.TempDir()` + `t.Chdir`): version file present with exact expected content after fresh init; stale value overwritten on re-init; idempotency contract extended to the new file; pre-feature repo (no file) initializes cleanly.
- **Phase 2.1** — two layers. Unit: `classifyVersion` table tests with hand-written oracles (match, whitespace-padded match, mismatch, empty, whitespace-only). Command: through `runRootCmd` so the real envelope and exit code are observed — three states × (exit 0, discriminant, status, versions, action presence rules), the read-only guarantee (byte-compare the version file and a sampled installed file before/after a mismatch invocation), the unreadable-file fault path returning `ErrorResponse` with exit 1, and `--schema`. Plus the three project-wide envelope contract tables in `cmd/root_test.go`.
- **Phase 3.1** — template-level: existing `{{command}}` and instruction-surface guard tests extend automatically; new assertion that every rendered skill contains the version-check instruction. No end-to-end agent-behavior test (deliberate gap — the preamble is instruction text; CI cannot assert what a live agent does with it).
- **Success metrics**: "informed on next skill invocation" → behavioural at the CLI boundary (mismatch → status + action) with the agent-relay portion **manual — captured in the implementation test plan**; "no false alarms" → behavioural (match → no action text); "stale-install bug reports drop to zero" → **manual — captured in the implementation test plan** (field observation).

## Project References

- Source spec: `000039_version-check` (read via `go run . spec file read 000039_version-check.md`).
- Prior plan `000018_release_workflow` — how CI derives and injects the release version (PR labels → ldflags).
- Commit `de84ae6` / PR #24 — the ldflags-path bug; rationale for keeping the version var in package `cmd`.
- Working files from planning: `.spektacular/work/000039_version-check/` (removed after the plan documents are committed).

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|-------------|----------------|
| Low | ~10k | Single agent, sequential |
| Medium | ~25k | 2-3 parallel agents |
| High | ~50k+ | Parallel analysis, sequential integration |

All three phases are Low to Low–Medium; every phase uses a single sequential agent. Total estimate ~55k tokens.

## Migration Notes

None required by design. Repositories initialized before this feature have no version file; `version check` reports `missing` with the re-run-init instruction, and a single re-init brings them fully current. No data transforms, no config migrations. Old binaries reading a repo that has the new file are unaffected (they never look at it).

## Performance Considerations

Negligible. The check is one file read plus a string comparison per skill invocation, and init gains one small file write. No caching, no network, no hashing.
