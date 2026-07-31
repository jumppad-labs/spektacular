---
created_date: "2026-07-31"
status: completed
closed_date: "2026-07-31"
---

# Research: 000039_version-check

## Alternatives considered and rejected

1. **Record the version as a field in `.spektacular/config.yaml`** — rejected. The
   spec's constraints require "a dedicated version file written at init time".
   Also conflates user-editable config with tool-managed metadata, and config
   is written twice on fresh init (`internal/project/init.go:84` then
   `cmd/init.go:43`), making ownership murky.
2. **Record the version in `.spektacular/state.json`** — rejected. `state.json`
   is workflow progress state (`internal/workflow/state.go:14-21`), created
   lazily by workflow commands, not by init; wrong lifecycle and it is
   deliberately read with error-swallowing snapshot semantics
   (`cmd/root.go:107-145`).
3. **Piggyback on cobra's built-in `--version`** for the agent-side comparison —
   rejected. `--version` prints plain text (`spektacular version 0.1.0`), not
   the JSON envelope every other command emits (`internal/output/writer.go`),
   and would force each skill to parse text and implement comparison logic —
   exactly what the spec's Technical Approach says to avoid.
4. **Runtime hook in the binary (check on every CLI command)** — rejected by the
   spec (Non-Goals: "Checking on every CLI command run is out of scope"); the
   check is agent-driven at skill invocation.
5. **Auto-refresh installed files on mismatch** — rejected by spec constraint:
   "The check must not modify or re-install any files itself".
6. **Mustache partial for the shared version-check instruction block** —
   possible but not required: `internal/agent/skills.go:47` calls
   `mustache.Render` with no `PartialProvider`, so `{{> partial}}` does not
   resolve today. Adding partial machinery for a ~5-line block duplicated
   across 4 templates is below the DRY threshold; plain per-template insertion
   is simpler. (Revisit if the shared block grows.)
7. **Template content hash as complementary staleness signal for dev builds** —
   descope recommended. Nothing hashes the embedded templates today (only
   hashing in repo is knowledge-search checksums, `internal/store/search.go:177`);
   `//go:embed all:*` (`templates/templates.go:6-7`) embeds test files too, so
   a hash needs a scoped walk. Dev builds (`go run .`) all report `0.1.0`, so
   in a dev-dogfood repo recorded and current version always match → silent,
   which satisfies "no false alarms"; the cost is that dev-build template
   drift goes undetected, which the spec explicitly allows descoping.

## Chosen approach — evidence

(Decision itself belongs to the architecture step; this is the supporting
evidence.)

- **Version source**: `cmd/root.go:18-23` — unexported `version = "0.1.0"`,
  `sha = ""`, set via ldflags `-X github.com/jumppad-labs/spektacular/cmd.version=...`
  (`Makefile:9-10`, `dagger/main.go:149-153`). `versionString()`
  (`cmd/root.go:37-42`) formats `"<version> (<sha>)"`. Commit de84ae6 ("Fix
  version number") fixed ldflags pointing at the wrong package (`main.` →
  `cmd.`) — linker fails silently on wrong `-X` paths; keep the var in
  package `cmd` (or add a guard test if moved).
- **Where to write at init**: `cmd/init.go:21-51` (`runInit`) is the single
  choke point; `cwd` and `cfg` in scope at `cmd/init.go:41-45`, and `version`
  is directly accessible there (same package). Precedent for writing a small
  embedded/static file into `.spektacular/` at init:
  `internal/project/init.go:89-96` (`.gitignore` write).
- **New CLI command shape**: model the parent group on `cmd/changelog.go:5-8`
  (8-line bare parent) and the subcommand on `cmd/skill.go:20-64` (result
  struct with JSON tags → `output.New(cmd.OutOrStdout(), globalFields).WriteResult(...)`);
  register in `cmd/root.go:238-243`. `--schema` branch pattern:
  `cmd/skill.go:28-47`.
- **Agent-actionable output**: `output.ErrorResponse`
  (`internal/output/writer.go:45-52`) with builders `NewError/WithResource/
  WithNextAction` (`writer.go:60-85`); `"error": false` discriminant is
  auto-injected for success results (`writer.go:104-114`) — never add an
  `Error` field to the result struct. Canonical next_action phrasing:
  `cmd/storefile.go:55-79`, `internal/workflow/workflow.go:205-233`.
- **Skill instruction insertion**: the 4 installed skills are
  `templates/skills/workflows/{spek-new,spek-plan,spek-implement,spek-knowledge}/SKILL.md`
  (list source of truth: `internal/agent/skills.go:24-30`), rendered
  byte-identically for all 3 agents with only `{"command": cfg.Command}`
  (`internal/agent/skills.go:38-65`). Natural insertion point: line 5 of each
  template, immediately after frontmatter (frontmatter must stay at byte 0),
  before the first CLI call. Rendered examples: `.claude/skills/*/SKILL.md`.
- **{{command}} convention guard tests**: `cmd/init_test.go:39,70,85,113,136-162`
  and `internal/agent/instruction_surface_test.go:21-48` (forbidden-substring
  guard walks skills/workflows + steps + rendered `.claude/skills/`).
- **Idempotency contract**: `cmd/init_test.go:164-184` (re-init leaves
  siblings untouched); re-init must overwrite the version file with the
  current binary's version (that is the fix path for a mismatch).
- **Gitignore state**: repo `.gitignore:43-44` ignores only
  `.spektacular/debug/`; `templates/.spektacular/.gitignore` (installed
  unconditionally, `internal/project/init.go:94`) ignores `*.tmp`, `*.log`,
  `tmp/`, etc. A new `.spektacular/version` file is committed by default —
  team-shared staleness signal (teammates with older binaries get prompted).

## Files examined

- `cmd/root.go:18-23,31,37-42` — version/sha vars, cobra `Version:`, `versionString()`.
- `cmd/root.go:57-102` — `runRoot()`: single envelope/exit-code choke point; `toErrorResponse` at 179-184.
- `cmd/root.go:196-227` — `loadConfig` (defaults when config absent), `dataDir()` = `<cwd>/.spektacular`, `projectRoot()` = `os.Getwd()`.
- `cmd/root.go:232-243` — command registration in `init()`; `--fields` persistent flag.
- `cmd/init.go:14-51` — full init flow; plain-text output (not JSON envelope); always `force=true`.
- `internal/project/init.go:17-117` — scaffold: dirs, config-if-absent, `.gitignore` from templates.FS (read directly, not via swappable `sourceFS`), knowledge READMEs.
- `internal/agent/skills.go:24-65` — `workflowSkills` list; `installWorkflowSkills`; `sourceFS` test seam at 32-34; plain `os.WriteFile` (non-atomic).
- `internal/agent/managed_section.go:21-140` — AGENTS.md managed-section machinery, atomic writes.
- `internal/agent/{claude,bob,codex}.go` — 3 agents; skills dirs `.claude/skills`, `.bob/skills`, `.agents/skills`; bob-only command wrappers; claude's `@AGENTS.md` import.
- `internal/agent/historical_artifacts.go:26-118` — duplicates managed-section machinery verbatim (known cleanup candidate, out of scope).
- `internal/output/writer.go:19-148` — `Write`/`WriteResult`/`WriteFailure`, `ErrorResponse`, discriminant injection, `--fields` filtering.
- `internal/config/config.go:110-164,167-183,292-301` — Config struct (no version field), defaults (`Command: "spektacular"`), YAML round-trip; unknown YAML keys silently ignored (old binaries tolerate new fields).
- `internal/workflow/state.go:14-61` — state.json shape and lifecycle (not init-owned).
- `cmd/skill.go:13-108` — model small command; `--schema` short-circuit; note `skill list` hand-rolls JSON (outlier, do not copy).
- `cmd/changelog.go:5-8` — minimal bare parent command group.
- `cmd/storefile.go:55-79,172-173,296-297` — canonical ErrorResponse usage.
- `Makefile:1-10,33-37` — `VERSION := 0.15.1`, ldflags only on `build` target; `cross` and harbor targets pass no ldflags (report 0.1.0).
- `dagger/main.go:56-71,137-153,407,501-545` — CI version from PR label; ldflags injection; `0.0.0` when no token; release refuses `0.0.0`.
- `templates/templates.go:6-7` — `//go:embed all:*` single FS (includes test files).
- `templates/skills/workflows/*/SKILL.md` — 4 skill templates (89/66/62/77 lines); `{{command}}` counts 14/12/12/9; STOP-block duplicated in 3 of 4.
- `cmd/init_test.go:14-184` — per-agent init tests, custom-command test, idempotency test; all `t.TempDir()` + `t.Chdir`.
- `cmd/root_test.go:68-88,92,219,332` — `runRootCmd` harness; envelope contract tables (stdout-only, discriminant, printed-once); plain-text exceptions documented for `init` and `file read` (root_test.go:214-218) — a new JSON command must be added to these tables.
- `cmd/implement_test.go:51-63` — `setupImplementCmd`: generic root buffer harness with cleanup.
- `cmd/error_response_test.go:30-87` — model JSON-error test walkthrough.
- `internal/store/search.go:6,177` — only sha256 usage in repo (knowledge search), cited as the local hashing idiom if ever needed.

## External references

- `github.com/spf13/cobra v1.8.1` — CLI framework; built-in `Version:` field supplies `--version` (no subcommand needed for human output).
- `github.com/cbroglie/mustache` — template rendering; supports `PartialProvider` but none is wired in (relevant to the rejected partial alternative).
- `github.com/stretchr/testify` (require) — test assertions.

## Prior plans / specs consulted

- Plan `000018_release_workflow` — confirmed version is PR-label-driven in CI and injected via ldflags by the Dagger `Build` function; explains why dev builds carry the `0.1.0` default.
- Spec `000039_version-check` (source spec) — constraints: dedicated version file, binary's own build version, check never modifies files, must not break pre-feature repos.
- Commit `de84ae6` ("Fix version number", merged via PR #24) — ldflags previously targeted `main.version` (nonexistent) and every released binary reported `0.1.0`; linker is silent on bad `-X` paths. Cautionary tale for moving the version var out of package `cmd`.
- Knowledge base: no entries relevant to init/versioning (searches for "init install", "version", "cli skill" surfaced only architecture docs about agent output formats).

## Open assumptions

1. **Comparison is exact string equality** on the recorded vs current version
   string (after trimming whitespace). "Cannot be parsed" in the spec reduces
   to "empty after trim" — no semver parsing needed. If implement finds a
   need for semver ordering (e.g. downgrade detection), STOP and ask.
2. **What to record**: the bare `version` var (e.g. `0.15.1`), not
   `versionString()` (which appends the sha). Recording version+sha would
   make two same-version builds mismatch. Architecture step must confirm.
3. **Dev builds (`go run .` → `0.1.0`) are simply exact-matched** like any
   other version; the template-hash complementary signal is descoped. In the
   dogfood repo this means version checks are always silent — acceptable.
4. **The version file is committed** (not gitignored) so staleness is
   team-visible. If the user wants it per-checkout, add it to
   `templates/.spektacular/.gitignore` instead.
5. **A mismatch result is not a CLI error** — the command succeeds (exit 0)
   and reports match/mismatch in the success envelope, OR it returns an
   agent-actionable ErrorResponse with exit 1. Architecture step decides;
   evidence supports either (workflow.go:205-233 shows error-with-next_action;
   skill.go:59-64 shows plain result).
6. **`spek-knowledge` also gets the check** — spec says "every time a skill is
   invoked"; all 4 workflow skills get the same instruction block.

## Rehydration cues

- `go run . spec file read 000039_version-check.md` — the source spec.
- `go run . knowledge always-applied` — conventions + glossary (currently just READMEs).
- Read `cmd/init.go`, `cmd/root.go:18-42,196-243`, `internal/agent/skills.go`,
  `internal/output/writer.go`, `templates/skills/workflows/spek-plan/SKILL.md`
  — the five files that anchor every design decision above.
- `go run . skill spawn-planning-agents` — the parallel-research guidance used.
- Working files: `.spektacular/work/000039_version-check/`; cross-cutting
  context in `.spektacular/context.md`.
