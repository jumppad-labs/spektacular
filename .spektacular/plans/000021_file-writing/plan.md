# Plan: 000021_file-writing

<!-- Metadata -->
<!-- Created: 2026-05-26T09:53:05Z -->
<!-- Commit: 28155410d21aa2867a832c720722b9fc7becaeb7 -->
<!-- Branch: f-file-writing -->
<!-- Repository: jumppad-labs/spektacular -->

## Overview

Spektacular's `spec file write` and `plan file write` commands currently read the file body from stdin, forcing the calling agent to use shell heredocs and escape any tricky characters in the content. This plan replaces that with a `--from <path>` flag that points the command at a source file on disk, so the contents are forwarded into the spec/plan store verbatim and skill/step instructions stop relying on fragile shell quoting. The change benefits the agent producing the content and any human or agent later reading the stored files, because the output is reliably well-formed.

## Architecture & Design Decisions

The file-write CLI handler is the only behavioural change site. Both `spec file write` and `plan file write` are built from a single shared cobra factory (`newStoreFileCmd`), so a one-line flag declaration and a single swapped read produce the new behaviour for both surfaces. The `Store` interface and the `FileStore` implementation stay untouched — the file-system read happens inside the CLI handler, with the resulting byte slice forwarded into the existing `Store.Write([]byte)` contract. This keeps the storage layer backend-agnostic so future store implementations (remote, etc.) aren't forced to grow local-filesystem semantics they may not need.

The new flag (`--from <path>`) is declared as required via cobra's `MarkFlagRequired`, which gives the "stdin path is gone" acceptance criterion for free: any invocation without `--from` exits non-zero before any store write happens — including the previous `cat … | … file write …` shell pattern. No bespoke stdin-detection code is needed, and no `--from -` compatibility shim is added (the spec explicitly removes the stdin interface). The source file is read with `os.ReadFile` and any read error is wrapped to name the offending path; the source file is never modified, moved, or deleted by Spektacular — cleanup is an agent-prose obligation.

The second arm of the change is the agent-facing instruction sweep. The source of truth for the workflow skills is `templates/skills/workflows/spek-{new,plan,implement}/SKILL.md`, and the active `.claude/skills/spek-*/SKILL.md` files are rendered output produced by `installWorkflowSkills` with `{{command}}` substitution. The plan edits the templates and the step prompts under `templates/steps/...`, then re-runs the install path to regenerate the dogfooded copies — the templates remain the single source of truth, with the rendered output committed alongside. A regression test scans both the templates and the rendered skill tree for the literal old patterns (`cat … | … file write` and "reads stdin") so a future template edit cannot silently reintroduce the old interface.

Rejected alternatives — most notably extending the `Store` interface with a `WriteFrom(dst, srcPath)` method — are recorded in `research.md § Alternatives considered and rejected`.

## Component Breakdown

- **`file write` command handler (changed)** — the single shared cobra handler that backs both `spec file write` and `plan file write`. Loses its stdin read, gains a required `--from <path>` flag, reads the source file from disk, and forwards the bytes to the unchanged store. Returns a path-naming error when the source file is missing or unreadable. The source file is never touched after reading.

- **`newStoreFileCmd` factory (changed)** — the cobra-command constructor that builds the `file` subcommand group. Declares the new `--from` flag on the `write` subcommand and marks it required, so missing-arg exits naturally cover the "stdin path is gone" acceptance criterion without bespoke stdin-detection code.

- **`Store` interface and `FileStore` (unchanged)** — explicitly called out as unchanged. The store still takes `(path, []byte)`; the source-file concern is a CLI-layer concern, not a storage-layer concern. Future store backends are unaffected.

- **Skill templates (changed)** — the three workflow skills in `templates/skills/workflows/spek-{new,plan,implement}/`. The "(reads stdin)" wording is replaced with the `--from` form, and the prose gains an explicit reminder that the scratch source file should be deleted after a successful write.

- **Step templates (changed)** — every step template that currently shows a `cat <scratch> | … file write …` invocation. Each becomes `… file write … --from <scratch>` followed by a cleanup line that removes the scratch file. Spans nine step templates across the spec, plan, and implement workflows; the `.spektacular/tmp/` scratch convention itself is unchanged.

- **Active rendered skills (regenerated)** — `.claude/skills/spek-{new,plan,implement}/SKILL.md` are byte-rendered output from the skill templates. After the templates change, the install path (`installWorkflowSkills`) re-renders them with the project's `{{command}}` substitution, and the regenerated files are committed alongside the template change.

- **Inline Go comments referencing the old interface (audited)** — comments in `internal/steps/spec/steps.go` and `internal/steps/plan/steps.go` that mention the old stdin/heredoc usage are updated where the wording is now wrong. No behaviour change.

- **CLI tests (changed)** — `cmd/file_test.go` and `cmd/plan_file_test.go` pivot from `rootCmd.SetIn(strings.NewReader(...))` to writing a temp source file with `t.TempDir()` and passing `--from <path>`. Each gains a verbatim-write test for shell-problematic characters, a missing-source error test, a source-preservation test, and a negative test for piping content without `--from`.

## Data Structures & Interfaces

This feature introduces no new types and changes no existing interfaces. The `Store` interface, `FileStore` struct, `config.Config`, and the `storeDirFunc` callback are all unchanged. The only contract that shifts is the **CLI surface** of the `write` subcommand, which is the binding contract between Spektacular and its callers (skills, agents):

```
Before:
  spec file write <path>        # body read from stdin
  plan file write <path>        # body read from stdin

After:
  spec file write <path> --from <source-path>   # --from required
  plan file write <path> --from <source-path>   # --from required
```

- `<path>` — store-relative destination (unchanged in meaning and resolution).
- `--from <source-path>` — required string flag. Filesystem path the CLI will `os.ReadFile`. Resolved relative to the caller's current working directory. The source file is read in full, its bytes forwarded verbatim to `Store.Write`, and the file is left in place on disk.

Per-field implementation detail (exact flag wiring, error-wrapping format, path-resolution rules) lives in `context.md`.

## Implementation Detail

This plan introduces no new code-shape patterns. It exercises three patterns the codebase already has, and introduces one small new convention in agent-facing prose.

- **CLI factory parametrization (existing pattern, reused)** — the cobra `write` subcommand is built once in the shared `newStoreFileCmd` factory and parametrized for both `spec file` and `plan file`. The new `--from` flag is declared in that one place, so both surfaces gain it from a single edit. No factory restructuring.
- **CLI test style (existing pattern, reused)** — the existing `t.Chdir`, `t.TempDir`, `writeSpecCommandConfig`, `setupImplementCmd`, `rootCmd.SetArgs` flow is unchanged. The pivot is only that tests now write a temp source file with `os.WriteFile` and pass `--from <path>` instead of feeding bytes through `rootCmd.SetIn`. New tests follow the same shape.
- **Template → rendered-skill install (existing pattern, reused)** — the project's source of truth for the workflow skills is `templates/skills/workflows/spek-{new,plan,implement}/SKILL.md`, and the rendered copies under `.claude/skills/*` are produced by `installWorkflowSkills` with `{{command}}` substitution. The plan changes the templates and runs the install path; the dogfooded rendered copies are an output of that step, not an additional source to maintain.
- **Scratch-file cleanup convention (new, in prose only)** — every step template that currently writes a scratch file under `.spektacular/tmp/` and pipes it gains a paired cleanup line that removes the scratch file after a successful write. This is a small new motif repeated across ~nine step templates so an agent reading any of them learns the same lifecycle: stage → commit via `--from` → remove. The scratch directory itself is unchanged (already git-ignored).

**Code-structure UX:** a developer reading `cmd/storefile.go` after the change sees one extra `StringVar`/`MarkFlagRequired` pair on the write subcommand and an `os.ReadFile` in place of the stdin read — the factory shape, the spec/plan parametrization, and the rest of the file are untouched. An agent reading any updated step or skill sees one consistent invocation shape across the whole workflow surface.

## Dependencies

- **`cmd` package (internal, no change)** — already houses the shared `newStoreFileCmd` factory. The new flag declaration lands here.
- **`internal/store` (internal, no change)** — `Store.Write([]byte)` already accepts the byte slice the CLI hands it. No interface or implementation change needed.
- **`internal/agent` (internal, no change)** — already owns `installWorkflowSkills`, which re-renders `.claude/skills/*` from the updated templates with `{{command}}` substitution. The plan invokes this existing path; no code change in this package.
- **`templates/` (internal, content only)** — the embedded templates filesystem (`templates.FS`) is the source of truth for the skill and step files. This plan edits the markdown content; the Go embedding wiring is unchanged.
- **`github.com/spf13/cobra` (external, no version change)** — provides the `StringVar` flag declaration and `MarkFlagRequired` API used to wire `--from`.
- **`github.com/stretchr/testify/require` (external, no version change)** — existing test assertions.
- **Standard library `os` (no change)** — `os.ReadFile` is the file read; `os.WriteFile` and `t.TempDir()` continue to serve the tests.
- **Prior plans / specs** — no upstream specs or plans need to land first. The `.spektacular/tmp/` scratch convention is already in place (used by step templates from earlier plans and `.gitignore`'d), so no preparatory work is required.

## Testing Approach

Testing for this feature is exercised entirely through the existing CLI integration test pattern in the `cmd` package (`rootCmd.Execute()` with `t.TempDir()`, `writeSpecCommandConfig`, and `setupImplementCmd`). The store layer keeps its existing unit-test coverage and gets nothing new — it is unchanged.

**Coverage focus.** The write subcommand is where every acceptance criterion is verifiable, so both `spec file write` and `plan file write` get parallel test coverage. The two surfaces share a handler, but each gets its own tests to guard against regressions if the factory is ever specialised, and to match how the existing tests are organised.

**Load-bearing assertions, in plain language:**
- *Verbatim write* — given a source file containing characters that would be problematic for shell escaping (backticks, `$`, single and double quotes, embedded newlines), the destination file in the store ends up byte-identical to the source.
- *Source preservation* — after a successful write, the source file at the `--from` path still exists with byte-identical content to what it had before the call.
- *Unreadable source produces a clear error* — pointing `--from` at a non-existent path makes the command exit non-zero with a message that names the offending path, and nothing is written to the spec/plan store.
- *Stdin path is gone* — invoking the command with content piped via stdin and no `--from` flag exits non-zero and leaves the store unchanged.

**Regression guard on agent-facing instructions.** One small unit test in `internal/agent` (or a sibling location) asserts that the workflow skill templates and the rendered active skills no longer contain the literal `cat … | … file write` pattern or the phrase "reads stdin". This is the cheapest possible mechanical enforcement of the "Agent-facing instructions reference only the new input method" acceptance criterion. The step templates under `templates/steps/...` get the same scan in the same test.

**Where tests slot in.** Alongside the existing `cmd/file_test.go` and `cmd/plan_file_test.go` for the CLI behaviour, following the table-driven testify style already in those files. The skill-rendering regression test slots alongside the existing `internal/agent/claude_test.go` and `internal/agent/agent_test.go` skill-install tests.

**Deliberate gaps.** No new tests in `internal/store` — the store contract is unchanged, and its existing unit tests still cover `Write([]byte)` correctness. No end-to-end harness exercise of an agent driving the workflow — the CLI integration tests cover the surface the agent sees.

## Milestones & Phases

### Milestone 1: File-write commands accept a source file on disk

**What changes**: When the user (or an agent) runs `spec file write` or `plan file write`, they point the command at a source file on disk instead of piping the body through the shell. The contents arrive in the spec or plan store byte-for-byte, even when the content contains characters that would have been mangled by shell quoting. If the source file is missing or unreadable, the command exits non-zero with a message that names the offending path. If anyone invokes the command in the old style — piping content with no source file — the command exits non-zero and the store is unchanged. The source file is left in place after a successful write.

#### - [x] Phase 1.1: Add `--from` flag and read source file in the write handler

The shared cobra `write` subcommand gains a required `--from <path>` string flag. The handler swaps its stdin read for an `os.ReadFile` of the flag value, wraps any read error with a message that names the offending path, and forwards the byte slice unchanged to the existing store. Both `spec file write` and `plan file write` inherit the change from the single factory, with no store-interface change.

*Technical detail:* [context.md#phase-11](./context.md#phase-11-add---from-flag-and-read-source-file-in-the-write-handler)

**Acceptance criteria**:

- [x] The `write` subcommand accepts `--from <path>` and refuses to run without it.
- [x] When `--from` points at a readable file, the destination in the store contains exactly the bytes of that file.
- [x] When `--from` points at a missing or unreadable file, the command exits non-zero and the error message includes the offending path.
- [x] After a successful write, the source file at `--from` is unchanged on disk.
- [x] The handler no longer reads from stdin.

#### - [x] Phase 1.2: Update and extend CLI tests for both write surfaces

The existing spec and plan write tests pivot from feeding bytes via `rootCmd.SetIn` to writing a temp source file and passing `--from`. Each test file gains new cases: a verbatim-write test with characters that would be problematic for shell escaping, a missing-source test that asserts the path-naming error, a source-preservation test, and a piped-input-with-no-flag negative test that asserts the store is unchanged.

*Technical detail:* [context.md#phase-12](./context.md#phase-12-update-and-extend-cli-tests-for-both-write-surfaces)

**Acceptance criteria**:

- [x] All existing `cmd` tests pass without behavioural regressions on the read/list/delete subcommands.
- [x] New verbatim-write tests cover content containing backticks, `$`, single and double quotes, and embedded newlines, for both spec and plan write.
- [x] New tests cover the missing-source error, source preservation, and the removed stdin path, for both spec and plan write.

### Milestone 2: Agent-facing instructions describe only the new input method

**What changes**: Every skill and step prompt that previously told an agent to `cat … | … file write …` now tells it to use `--from`, names `.spektacular/tmp/` as the scratch location to stage its working file, and reminds it to remove the working file after a successful write. The dogfooded rendered skills under `.claude/skills/` are regenerated from the updated templates so the active project also runs on the new wording. An agent driving Spektacular reads one consistent invocation shape across the whole workflow surface.

#### - [x] Phase 2.1: Rewrite skill and step templates and regenerate active skills

Every skill and step template that currently mentions stdin/heredoc input — three skill templates and the nine step templates spanning the spec, plan, and implement workflows — is rewritten to use `{{config.command}} … file write … --from .spektacular/tmp/<scratch>.md` and to instruct the agent to remove the scratch file after a successful write. The dogfooded rendered skills under `.claude/skills/` are regenerated from the updated templates via the existing install path so the active project runs on the new prose.

*Technical detail:* [context.md#phase-21](./context.md#phase-21-rewrite-skill-and-step-templates-and-regenerate-active-skills)

**Acceptance criteria**:

- [x] No skill or step template in `templates/` still contains `cat … | … file write …` or the phrase "reads stdin".
- [x] Every step template that writes a scratch file under `.spektacular/tmp/` includes a paired cleanup line that removes that scratch file after the successful write.
- [x] The active `.claude/skills/spek-{new,plan,implement}/SKILL.md` files match the re-rendered output of the updated templates.

#### - [x] Phase 2.2: Add a regression test that scans the instruction surface for the old interface

A small test in the `internal/agent` package walks the embedded `templates/` filesystem and the project's `.claude/skills/` tree, and fails if any file contains a `cat … | … file write` invocation or the phrase "reads stdin". This is the mechanical guard for the "Agent-facing instructions reference only the new input method" acceptance criterion and prevents future template edits from silently reintroducing the old pattern.

*Technical detail:* [context.md#phase-22](./context.md#phase-22-add-a-regression-test-that-scans-the-instruction-surface-for-the-old-interface)

**Acceptance criteria**:

- [x] The regression test fails on any reintroduction of `cat … | … file write` in the template or rendered-skill surface.
- [x] The regression test fails on any reintroduction of the literal phrase "reads stdin" in those files.
- [x] The full `go test ./...` suite passes.

## Open Questions

None. Every decision the implementer needs has been made: the CLI shape (`--from <path>`, required flag), the read mechanism (`os.ReadFile`, with the path named in any error), the boundary (CLI handler only — store interface unchanged), the agent-instruction sweep (templates updated, active skills re-rendered via `installWorkflowSkills`), and the regression guard (scan templates + `.claude/skills/` for `cat … | … file write` and "reads stdin"). Nothing here is gated on something only implementation can reveal.

## Out of Scope

- **A `Store.WriteFrom(dst, srcPath)` interface method.** Considered during architecture and rejected — file reading stays a CLI-layer concern so the storage contract remains backend-agnostic. Recorded in `research.md § Alternatives considered and rejected`.
- **Backward-compatible stdin fallback (`--from -` or auto-detect).** The spec requires the stdin path be removed outright; no compatibility shim is provided. Callers must pass `--from <path>`.
- **Automatic cleanup of `.spektacular/tmp/`.** Spektacular does not delete the agent's scratch files itself. The cleanup obligation lives in the agent-facing instructions (the agent runs `rm` after a successful write).
- **Restructuring or renaming the `.spektacular/tmp/` scratch convention.** The existing scratch directory and file names (`spec_template.md`, `plan_template.md`, `context_template.md`, `research_template.md`, plus the `_update.md` variants for the implement workflow) are kept verbatim; this plan only changes the invocation that reads them.
- **Updating non-skill, non-step documentation** that references the workflow only at an abstract level (README, AGENTS.md, top-level CLAUDE.md). None of these currently show the stdin/heredoc form; if a future audit finds one, it's handled as a follow-up, not here.

## Changelog

### 2026-05-26 — Phase 1.1: Add `--from` flag and read source file in the write handler

**What was done**: `cmd/storefile.go` swaps `io.ReadAll(cmd.InOrStdin())` for `os.ReadFile(fromPath)` on the shared `write` subcommand, declares a required `--from <path>` flag via `MarkFlagRequired`, and wraps any read error as `fmt.Errorf("reading source file %q: %w", fromPath, err)`. `Short` was updated and the `io` import was replaced with `os`. Both `spec file write` and `plan file write` inherit the change.

**Deviations**: None.

**Files changed**:
- `cmd/storefile.go`

**Discoveries**: The `cat ... | go run . plan file write ...` invocation embedded in the implement-workflow step instructions (`update_plan` and `update_changelog`) now fails because `--from` is required, so the dogfooded run must use the new form `go run . plan file write <doc> --from <src>` until Phase 2.1 lands the template/skill rewrites.

### 2026-05-26 — Phase 1.2: Update and extend CLI tests for both write surfaces

**What was done**: `cmd/file_test.go` and `cmd/plan_file_test.go` pivot their existing `*FileWrite_ResolvesConfiguredDirectory` tests from `rootCmd.SetIn(strings.NewReader(...))` to writing a `t.TempDir()`-staged source file and passing `--from <src>`. Each file gains four new tests covering the new contract: `PreservesProblematicCharacters` (backticks, `$`, single/double quotes, embedded newlines — byte-equal destination), `MissingSourceErrors` (non-existent `--from` returns an error whose message contains the offending path and leaves the destination absent), `PreservesSourceFile` (source bytes unchanged after a successful write), and `PipedStdinWithoutFromFails` (stdin piped without `--from` returns an error and writes nothing).

**Deviations**: Done in the same loop iteration as Phase 1.1 because the new tests are exactly what verifies Phase 1.1's acceptance criteria; both phase checkboxes were ticked together.

**Files changed**:
- `cmd/file_test.go`
- `cmd/plan_file_test.go`

**Discoveries**: `setupImplementCmd(t)` resets command state between tests, but `rootCmd.SetIn` is not reset by it — the `PipedStdinWithoutFromFails` tests must still pair `SetIn` with a `t.Cleanup` that nils it, or stdin leaks into subsequent test cases.

### 2026-05-26 — Phase 2.1: Rewrite skill and step templates and regenerate active skills

**What was done**: The three workflow skill templates (`templates/skills/workflows/spek-{new,plan,implement}/SKILL.md`) and nine step templates spanning the spec, plan, and implement workflows now describe the `{{config.command}} … file write … --from .spektacular/tmp/<scratch>.md` invocation form and instruct the agent to `rm` the scratch file after a successful write. The "Large specs/plans exceed the tool-call size limit when inlined as a heredoc" justification is dropped — the new form is just how the command works. Inline Go comments in `internal/steps/{spec,plan}/steps.go` were audited; none referenced the stdin/heredoc mechanism, so no Go changes were needed. The dogfooded `.claude/skills/spek-{new,plan,implement}/SKILL.md` files were regenerated by running `go run . init claude`, which substitutes `{{command}}` → `go run .` via `installWorkflowSkills`.

**Deviations**: None.

**Files changed**:
- `templates/skills/workflows/spek-new/SKILL.md`
- `templates/skills/workflows/spek-plan/SKILL.md`
- `templates/skills/workflows/spek-implement/SKILL.md`
- `templates/steps/spec/08-verification.md`
- `templates/steps/spec/09-finished.md`
- `templates/steps/plan/13-verification.md`
- `templates/steps/plan/14-write_plan.md`
- `templates/steps/plan/15-write_context.md`
- `templates/steps/plan/16-write_research.md`
- `templates/steps/plan/17-finished.md`
- `templates/steps/implement/06-update_plan.md`
- `templates/steps/implement/07-update_changelog.md`
- `.claude/skills/spek-new/SKILL.md` (regenerated)
- `.claude/skills/spek-plan/SKILL.md` (regenerated)
- `.claude/skills/spek-implement/SKILL.md` (regenerated)

**Discoveries**: `go run . init claude` is idempotent and does not perturb `.spektacular/config.yaml` when it's already in canonical form — it can safely be used as the skill-regeneration command during plan implementation. The spec workflow's `09-finished.md` template did not actually contain a `cat … |` invocation (it showed a bare `spec file write` line), so the drift noted at the start of the run was correct: the plan's line-reference range was approximate, not the pattern presence.

### 2026-05-26 — Phase 2.2: Add a regression test that scans the instruction surface for the old interface

**What was done**: Added `internal/agent/instruction_surface_test.go` with two tests. `TestEmbeddedTemplatesAvoidStdinInstructionSurface` walks the embedded `templates.FS` under `skills/workflows/` and `steps/` and asserts no `.md` file contains any of the closed list of forbidden substrings (`cat .spektacular/tmp/`, the four `| … file write` pipe forms, and `reads stdin`). `TestRenderedSkillsAvoidStdinInstructionSurface` does the same scan over the on-disk `../../.claude/skills/` tree, including a clear failure message pointing to `go run . init claude` when the tree is missing.

**Deviations**: None.

**Files changed**:
- `internal/agent/instruction_surface_test.go`

**Discoveries**: The `.claude/skills/` tree scan uses a relative path from the `internal/agent/` test working directory. Tests in `internal/agent` therefore have an implicit dependency on the dogfooded rendered output existing at the project root — the failure message in `os.Stat` surfaces this dependency rather than letting a `WalkDir` traverse over a missing root.
