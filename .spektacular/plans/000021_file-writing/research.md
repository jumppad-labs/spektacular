# Research: 000021_file-writing

## Alternatives considered and rejected

### Option A: Two positional args (`cp src dst` style)

Pass the source file as a second positional argument: `spec file write <source-path> <store-path>` (or reversed). Terser at the call site than `--from` and matches the `cp` mental model.

**Rejected**: position-order swaps are easy to make and silent — the command would happily read the wrong file or write to the wrong destination. The existing first positional (`<path>`) already means "destination in the store", and reusing that slot for a different role would change the meaning of an existing argument and silently break any existing caller habit. `--from` keeps the existing positional contract intact and makes the new role self-documenting in skill prose. Decision recorded in the architecture step of this plan.

### Option B: Source positional, destination via `--to` flag

Invert the shape — make the source the positional and put the destination behind `--to <name>`. Symmetric with Option A but inverts which arg is the flag.

**Rejected**: same swap-by-mistake risk as Option A, plus a much larger churn against the existing test suite and every existing skill template (every invocation has to be rewritten with the args reordered and the destination behind a new flag). No benefit over `--from`. Decision recorded in the architecture step.

### Option C: Extend `Store` with `WriteFrom(dst, srcPath string) error`

Push the file-reading concern into the storage interface so the store backend is responsible for ingesting from a source path.

**Rejected**: the storage layer is currently backend-agnostic — `Store.Write([]byte) error` works for any backend, including a future remote/network store. Adding `WriteFrom(srcPath)` would force every implementation to either handle local-filesystem semantics they may not have (a remote backend wouldn't have access to the CLI caller's local filesystem) or fall back to opening and reading the file themselves — which is exactly what the CLI handler can do once, in one place. No second consumer of `WriteFrom` exists or is foreseen. Evidence: `internal/store/store.go:32-50` (Store interface) and `internal/store/store.go:101-110` (FileStore.Write) — both stay clean under the chosen CLI-layer approach. Decision recorded in the architecture step.

### Option D: Backward-compatible `--from -` to read stdin

Keep the stdin path as a special case: `--from -` reads from stdin, anything else is a file path. Lets old callers keep working.

**Rejected**: the spec explicitly requires removing the stdin interface (`.spektacular/specs/000021_file-writing.md:30` — "Remove inline/stdin input for file writes") and that the old form exit non-zero (`.spektacular/specs/000021_file-writing.md:64` — "Stdin input is no longer accepted"). A compatibility shim violates both requirements. Decision recorded in the architecture step.

### Option E: Auto-detect stdin (no `--from` flag)

When stdin is a pipe, read from it; otherwise require a positional source. Avoids introducing a new flag.

**Rejected**: same spec-violation problem as Option D, plus auto-detection has its own footgun — tests and scripts can have stdin set in non-obvious ways, leading to silent reads or hangs. The explicit required flag is unambiguous. Decision recorded in the architecture step.

## Chosen approach — evidence

The chosen approach is: add a required `--from <path>` flag to the existing shared `write` subcommand, swap the stdin read for `os.ReadFile`, wrap the error to name the path, keep `Store.Write([]byte)` unchanged.

- `cmd/storefile.go:38-116` — `newStoreFileCmd` factory. Single change site; both spec and plan inherit the new behaviour.
- `cmd/storefile.go:41-56` — the `write` subcommand block where the flag is declared and the read swap happens.
- `cmd/storefile.go:50-52` — the lines being replaced (`io.ReadAll(cmd.InOrStdin())` plus its error wrap).
- `internal/store/store.go:101-110` — `FileStore.Write(path, []byte)`. Stays exactly as is, confirming the change does not need to descend into the storage layer.
- `cmd/file_test.go:15-30` and `cmd/plan_file_test.go:14-29` — existing tests show the testing convention (`t.TempDir`, `t.Chdir`, `writeSpecCommandConfig`, `setupImplementCmd`, `rootCmd.SetArgs`); the new tests slot in alongside.
- `internal/agent/skills.go:39-64` — the `installWorkflowSkills` install path that re-renders `.claude/skills/*` from the updated templates with `{{command}}` substitution. The plan exercises this existing pipeline rather than maintaining the rendered output by hand.

## Files examined

- `cmd/storefile.go:1-116` — shared cobra factory for `spec file` and `plan file` subcommands; the only behavioural change site for this plan.
- `cmd/file.go:7-12` — spec-side wiring of the factory.
- `cmd/plan_file.go:8-12` — plan-side wiring of the factory.
- `cmd/file_test.go:1-49` — spec write/read tests; current stdin-feeding style via `rootCmd.SetIn`.
- `cmd/plan_file_test.go:1-49` — plan write/read tests; identical structure to file_test.go.
- `cmd/implement_test.go:39-…` — `setupImplementCmd` helper used by the file tests; resets command state and captures stdout/stderr.
- `internal/store/store.go:32-110` — `Store` interface and `FileStore` implementation; confirms `Write([]byte)` is the persistence sink and is not touched by this plan.
- `internal/store/store_test.go:16-107` — store-layer tests; informational, no change.
- `internal/agent/skills.go:1-65` — `installWorkflowSkills` install path; the regeneration mechanism for `.claude/skills/*`.
- `internal/agent/claude.go:17` — `installWorkflowSkills(projectPath, ".claude/skills", cfg, out)` call site for Claude.
- `internal/agent/claude_test.go:24-65` and `internal/agent/agent_test.go:88-130` — skill-install tests; structural checks that survive the byte-level rendered-output change.
- `templates/skills/workflows/spek-new/SKILL.md:24`, `spek-plan/SKILL.md:24`, `spek-implement/SKILL.md:24` — three skill templates with the "(reads stdin)" wording to be replaced.
- `templates/steps/spec/08-verification.md:39-47`, `09-finished.md:7-12` — spec workflow step templates with `cat … | … file write` invocations.
- `templates/steps/plan/13-verification.md:84-89`, `14-write_plan.md:5-9`, `15-write_context.md:5-18`, `16-write_research.md:5-9`, `17-finished.md:5-11` — plan workflow step templates with `cat … | … file write` invocations.
- `templates/steps/implement/06-update_plan.md:18-22`, `07-update_changelog.md:35-40` — implement workflow step templates with `cat … | … file write` invocations.
- `.claude/skills/spek-{new,plan,implement}/SKILL.md:24` — rendered active skills (regenerated from templates).
- `internal/steps/spec/steps.go:142, 161` and `internal/steps/plan/steps.go:192, 201, 216, 231, 242, 253` — inline Go comments referencing `spec file write` / `plan file write`; audited and updated where wording is now stale.
- `.gitignore:45` — confirms `.spektacular/tmp/` is git-ignored, validating its use as the scratch convention.

## External references

- Cobra documentation — `(*Command).MarkFlagRequired` for required flag enforcement. <https://pkg.go.dev/github.com/spf13/cobra#Command.MarkFlagRequired> — why a one-line call is sufficient to satisfy the "stdin path is gone" acceptance criterion without bespoke detection logic.
- Go standard library `os.ReadFile` — <https://pkg.go.dev/os#ReadFile> — returns the file contents as `[]byte`, with `*os.PathError` (which `%w`-wraps cleanly into the error chain) on read failure. Confirms the error already names the path when wrapped.

## Prior plans / specs consulted

- `.spektacular/specs/000021_file-writing.md` — the spec driving this plan. Source of every acceptance criterion.
- `.spektacular/plans/000020_context/plan.md` — closest prior plan touching the workflow templates and CLI surface (knowledge layer + provider-based config). Confirms the `templates/` → `.claude/skills/` install pattern is already in place and shouldn't be reshaped.

## Open assumptions

- `installWorkflowSkills` (`internal/agent/skills.go:39-64`) re-renders `.claude/skills/*` deterministically with `{{command}}` substituted to `cfg.Command`; for this project `cfg.Command` resolves to `go run .`. If the rendered output differs from the committed `.claude/skills/*` because of an out-of-band template-rendering change, the implement workflow must STOP and reconcile before committing the regenerated files.
- The existing rendered active skills (`.claude/skills/spek-*/SKILL.md`) were last regenerated from the current templates and are byte-equivalent to what the install path would produce today. If `diff` against a fresh re-render shows pre-existing drift, STOP and reconcile before applying this plan's changes.
- The `.spektacular/tmp/` scratch convention is the only place agents stage working files; no other scratch location is referenced by any skill or step template. If a hidden second scratch location exists, the regression test in Phase 2.2 must be widened to cover it.
- `internal/agent/instruction_surface_test.go` does not yet exist and its filename is illustrative; the implementer is free to pick a more idiomatic location (e.g. `templates/instruction_surface_test.go` if a `templates` package exists, or sitting alongside `skills_test.go` if one is added).

## Rehydration cues

To rebuild context from cold:

- Read the spec: `go run . spec file read 000021_file-writing.md`.
- Read this plan: `go run . plan file read 000021_file-writing/plan.md`, `go run . plan file read 000021_file-writing/context.md`, `go run . plan file read 000021_file-writing/research.md`.
- Reread the CLI handler: `cmd/storefile.go:38-116`.
- Reread the install path: `internal/agent/skills.go:39-64`.
- List every file containing the old invocation pattern: `grep -RIn --include="*.md" "cat .spektacular/tmp/" templates/ .claude/`.
- List every file containing the old phrase: `grep -RIn --include="*.md" "reads stdin" templates/ .claude/`.
- Run the existing CLI tests to confirm the baseline: `go test ./cmd/...`.
