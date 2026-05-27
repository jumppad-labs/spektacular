# Context: 000021_file-writing

## Current State Analysis

The file-write commands today are built by a single shared cobra factory and read all input from stdin via `io.ReadAll`. The wiring is:

- `cmd/storefile.go:38-116` — `newStoreFileCmd(short, dir)` builds a `file` command group with `write`, `read`, `delete`, `list` subcommands. The `write` subcommand at `cmd/storefile.go:41-56` calls `io.ReadAll(cmd.InOrStdin())` (line 50) and forwards the resulting `[]byte` to `st.Write(filepath.Join(storeDir, args[0]), content)` (line 54).
- `cmd/file.go:7-12` — wires `newStoreFileCmd("Manage spec files in the spec store", specFileDir)` under `specCmd`.
- `cmd/plan_file.go:8-12` — wires `newStoreFileCmd("Manage plan documents in the plan store", planFileDir)` under `planCmd`.
- `cmd/storefile.go:3-12` — imports include `io` (used only by `io.ReadAll` in `write`) and `path/filepath`. No `os` import yet.
- `internal/store/store.go:101-110` — `FileStore.Write(path string, content []byte) error` is the persistence sink. It is unchanged by this plan.

Existing CLI tests feed bytes through `rootCmd.SetIn`:

- `cmd/file_test.go:15-30` — `TestSpecFileWrite_ResolvesConfiguredDirectory`. Calls `rootCmd.SetIn(strings.NewReader("spec body"))`, then `rootCmd.SetArgs([]string{"spec", "file", "write", "feature.md"})`, asserts the file lands under the configured directory.
- `cmd/file_test.go:34-48` — `TestSpecFileRead_ResolvesConfiguredDirectory`. Reads back. Unaffected by this plan.
- `cmd/plan_file_test.go:14-29` / `33-47` — same shape for plan write/read.
- Shared helpers: `setupImplementCmd(t)` (`cmd/implement_test.go:39`) resets command state and captures stdout/stderr; `writeSpecCommandConfig(t, dir, yaml)` writes a `.spektacular/config.yml` for the test.

Agent-facing instruction surface that mentions stdin/heredoc:

- Skill templates (source of truth, rendered via `installWorkflowSkills`):
  - `templates/skills/workflows/spek-new/SKILL.md:24`
  - `templates/skills/workflows/spek-plan/SKILL.md:24`
  - `templates/skills/workflows/spek-implement/SKILL.md:24`
- Step templates with active `cat … | {{config.command}} … file write …` invocations:
  - `templates/steps/spec/08-verification.md:39-45`
  - `templates/steps/spec/09-finished.md:7-12`
  - `templates/steps/plan/13-verification.md:84-89`
  - `templates/steps/plan/14-write_plan.md:5-9`
  - `templates/steps/plan/15-write_context.md:5-18`
  - `templates/steps/plan/16-write_research.md:5-9`
  - `templates/steps/plan/17-finished.md:5-11`
  - `templates/steps/implement/06-update_plan.md:18-22`
  - `templates/steps/implement/07-update_changelog.md:35-40`
- Rendered active skills (dogfooded; regenerated from templates):
  - `.claude/skills/spek-new/SKILL.md:24`
  - `.claude/skills/spek-plan/SKILL.md:24`
  - `.claude/skills/spek-implement/SKILL.md:24`
- Inline Go comments to audit:
  - `internal/steps/spec/steps.go:142, 161`
  - `internal/steps/plan/steps.go:192, 201, 216, 231, 242, 253`

Install path used to regenerate rendered skills:

- `internal/agent/skills.go:39-64` — `installWorkflowSkills(projectPath, targetSkillsDir, cfg, out)`. Reads each template from `templates.FS`, renders `{{command}}` from `cfg.Command`, writes to `<projectPath>/<targetSkillsDir>/<skill-name>/SKILL.md`.
- `internal/agent/claude.go:17` — `installWorkflowSkills(projectPath, ".claude/skills", cfg, out)` is the call site for the Claude target.

Scratch convention: `.spektacular/tmp/` exists and is git-ignored (`.gitignore:45`). It currently contains `spec_template.md`, `plan_template.md`, `context_template.md`, `research_template.md` from prior workflow runs — these names are reused by the step templates as the canonical scratch filenames.

## Per-Phase Technical Notes

### Phase 1.1: Add `--from` flag and read source file in the write handler

**File changes:**

- `cmd/storefile.go:41-56` — `write` subcommand. Declare a string variable `fromPath` adjacent to the command construction (or use `cmd.Flags().GetString("from")` inside `RunE`). Add `cmd.Flags().StringVar(&fromPath, "from", "", "Path to the source file whose contents will be written")` after the `&cobra.Command{...}` literal, then `_ = write.MarkFlagRequired("from")`. Replace the `io.ReadAll(cmd.InOrStdin())` call (line 50) with `os.ReadFile(fromPath)`. Wrap the error as `fmt.Errorf("reading source file %q: %w", fromPath, err)`. Update the `Short` description from `"Write stdin to a file in the store"` to `"Write the contents of a source file into the store"`.
- `cmd/storefile.go:3-12` — imports. Remove `"io"` (no longer used). Add `"os"`. `path/filepath` and `fmt` remain.

Concrete final shape (illustrative — adjust to match the surrounding style):

```go
var fromPath string
write := &cobra.Command{
    Use:   "write <path>",
    Short: "Write the contents of a source file into the store",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        st, storeDir, err := storeFileStore(dir)
        if err != nil {
            return err
        }
        content, err := os.ReadFile(fromPath)
        if err != nil {
            return fmt.Errorf("reading source file %q: %w", fromPath, err)
        }
        return st.Write(filepath.Join(storeDir, args[0]), content)
    },
}
write.Flags().StringVar(&fromPath, "from", "", "Path to the source file whose contents will be written into the store")
_ = write.MarkFlagRequired("from")
```

**Complexity**: Low
**Token estimate**: ~3k
**Agent strategy**: Single agent, sequential. The whole change lives in one file.

### Phase 1.2: Update and extend CLI tests for both write surfaces

**File changes:**

- `cmd/file_test.go:15-30` — `TestSpecFileWrite_ResolvesConfiguredDirectory`. Remove the `rootCmd.SetIn(strings.NewReader("spec body"))` / `t.Cleanup(func() { rootCmd.SetIn(nil) })` pair. Write the source file to a temp path: `srcPath := filepath.Join(t.TempDir(), "src.md"); require.NoError(t, os.WriteFile(srcPath, []byte("spec body"), 0o644))`. Change `SetArgs` to `[]string{"spec", "file", "write", "feature.md", "--from", srcPath}`. Existing destination-content assertion stays.
- `cmd/file_test.go` (new tests, appended):
  - `TestSpecFileWrite_PreservesProblematicCharacters` — body containing backticks, `$`, `'`, `"`, and embedded `\n`. Assert destination bytes equal source bytes via `require.Equal(t, srcBytes, dstBytes)`.
  - `TestSpecFileWrite_MissingSourceErrors` — point `--from` at a path that doesn't exist; assert `rootCmd.Execute()` returns a non-nil error whose `.Error()` contains the offending path. Assert the configured destination file does not exist.
  - `TestSpecFileWrite_PreservesSourceFile` — successful write; assert `os.ReadFile(srcPath)` after the call returns the original bytes.
  - `TestSpecFileWrite_PipedStdinWithoutFromFails` — `rootCmd.SetIn(strings.NewReader("ignored"))`; args `{"spec", "file", "write", "feature.md"}` (no `--from`). Assert non-nil error and destination file does not exist.
- `cmd/plan_file_test.go:14-29` — mirror the pivot for `TestPlanFileWrite_ResolvesConfiguredDirectory`.
- `cmd/plan_file_test.go` (new tests, appended): mirror the four new spec tests for plan write.

Notes:

- Tests using `rootCmd.SetIn` previously relied on the `t.Cleanup` to clear stdin between tests; the new tests that don't set stdin don't need it, but the negative-stdin test still does.
- The `setupImplementCmd(t)` helper (`cmd/implement_test.go:39`) continues to handle command-state reset; no change to it.
- `os.WriteFile` mode `0o644` matches the existing style in `cmd/file_test.go:41`.

**Complexity**: Low
**Token estimate**: ~5k
**Agent strategy**: Single agent, sequential — or two parallel agents splitting `file_test.go` and `plan_file_test.go` if the implementer prefers, since the changes are independent and identical in shape.

### Phase 2.1: Rewrite skill and step templates and regenerate active skills

**File changes (templates — source of truth):**

- `templates/skills/workflows/spek-new/SKILL.md:24` — replace `- \`{{command}} spec file write <name>.md\` — write a spec file into the spec store (reads stdin).` with `- \`{{command}} spec file write <name>.md --from <source-path>\` — write a spec file into the spec store from a source file on disk.` Add (or extend an adjacent paragraph) a reminder that the source file should be staged under `.spektacular/tmp/` and removed after a successful write.
- `templates/skills/workflows/spek-plan/SKILL.md:24` — same shape for `plan file write` (with `<name>/<doc>.md`).
- `templates/skills/workflows/spek-implement/SKILL.md:24` — same shape for `plan file write`.
- `templates/steps/spec/08-verification.md:39-47` — replace the heredoc/`cat` block. New form:
  ```
  {{config.command}} spec file write {{spec_name}}.md --from .spektacular/tmp/spec_template.md
  rm .spektacular/tmp/spec_template.md
  ```
  Adjust the preceding prose ("Large specs exceed the tool-call size limit when inlined as a heredoc …") to drop the heredoc justification — now it's simply how the command works.
- `templates/steps/spec/09-finished.md:7-12` — replace any remaining `cat … |` line with the `--from` form plus cleanup `rm`.
- `templates/steps/plan/13-verification.md:84-89` — replace the `cat … | {{config.command}} plan file write {{plan_name}}/plan.md` block with `{{config.command}} plan file write {{plan_name}}/plan.md --from .spektacular/tmp/plan_template.md` followed by `rm .spektacular/tmp/plan_template.md`. Update the surrounding prose the same way.
- `templates/steps/plan/14-write_plan.md:5-9` — same pattern.
- `templates/steps/plan/15-write_context.md:5-18` — two invocations (context.md, research.md). Update both to `--from` plus cleanup.
- `templates/steps/plan/16-write_research.md:5-9` — same.
- `templates/steps/plan/17-finished.md:5-11` — three invocations (plan.md, context.md, research.md). Convert all three; cleanup line per scratch file.
- `templates/steps/implement/06-update_plan.md:18-22` — convert the `cat .spektacular/tmp/plan_update.md | …` invocation to `--from`; add `rm .spektacular/tmp/plan_update.md`.
- `templates/steps/implement/07-update_changelog.md:35-40` — same.
- `internal/steps/spec/steps.go:142, 161` — audit comments referencing `spec file write` and update wording only where it mentions the stdin/heredoc mechanism.
- `internal/steps/plan/steps.go:192, 201, 216, 231, 242, 253` — same audit.

**File changes (rendered output, regenerated):**

- `.claude/skills/spek-new/SKILL.md`
- `.claude/skills/spek-plan/SKILL.md`
- `.claude/skills/spek-implement/SKILL.md`

Regenerated by running the install path — easiest is to write a small one-shot Go test helper or a `go run` driver that calls `installWorkflowSkills(<projectPath>, ".claude/skills", config.Default(), os.Stdout)`. Alternatively, a tiny test-only invocation in `internal/agent/claude.go`'s flow can be exercised manually. The implementer picks whichever path is least friction — the goal is byte-identical output to what `installWorkflowSkills` would produce. The regenerated files are committed.

**Complexity**: Medium — many files, but each edit is mechanical.
**Token estimate**: ~15k across the whole phase.
**Agent strategy**: 2-3 parallel agents. Split by template area:
- Agent A: `templates/skills/workflows/spek-*/SKILL.md` (three files).
- Agent B: `templates/steps/spec/*.md` and `templates/steps/implement/*.md`.
- Agent C: `templates/steps/plan/*.md` (five files).
After all three finish, sequentially: audit the Go comments in `internal/steps/*/steps.go`, then regenerate the three `.claude/skills/spek-*/SKILL.md` files via the install path.

### Phase 2.2: Add a regression test that scans the instruction surface for the old interface

**File changes:**

- New test file, e.g. `internal/agent/instruction_surface_test.go`. The test:
  1. Walks `templates.FS` (the embedded fs declared in `templates/embed.go` or equivalent — check actual path) under `skills/workflows/` and `steps/`, reads every `.md` file.
  2. Walks `.claude/skills/` on disk under the project root, reads every `SKILL.md`.
  3. For each file body, asserts it does NOT contain any of the following forbidden substrings:
     - `cat .spektacular/tmp/` (the canonical scratch-pipe pattern across all current usages)
     - `| {{config.command}} spec file write`
     - `| {{config.command}} plan file write`
     - `| go run . spec file write` (the rendered form)
     - `| go run . plan file write`
     - `reads stdin`
  4. Failure message names the offending file and the offending substring.

Use the existing `templates.FS` and `fs.WalkDir`. Use `os.Root` or `filepath.Walk` for the `.claude/skills/` scan. The `.claude/skills/` scan uses the project root resolver if available; otherwise hardcode the relative path from the test working directory (`../../.claude/skills`).

**Complexity**: Low
**Token estimate**: ~2k
**Agent strategy**: Single agent, sequential.

## Testing Strategy

All implementation testing slots into the existing `cmd/` integration-test style:

1. **`cmd/file_test.go`** — extended with four new tests for spec file write (verbatim, missing source, source preservation, no-`--from` stdin negative).
2. **`cmd/plan_file_test.go`** — mirror set of four new tests for plan file write.
3. **`internal/agent/instruction_surface_test.go`** (new) — regression test scanning templates + rendered skills for forbidden patterns.

Existing tests that are touched only by the test-style pivot (not by behaviour change):

- `cmd/file_test.go:15-30` (`TestSpecFileWrite_ResolvesConfiguredDirectory`) — switches input mechanism.
- `cmd/plan_file_test.go:14-29` (`TestPlanFileWrite_ResolvesConfiguredDirectory`) — switches input mechanism.

Existing tests that should pass unchanged:

- `cmd/file_test.go:34-48` (`TestSpecFileRead_ResolvesConfiguredDirectory`)
- `cmd/plan_file_test.go:33-47` (`TestPlanFileRead_ResolvesConfiguredDirectory`)
- `internal/store/store_test.go:16-32`, `:53-65`, `:97-107` — store-layer tests, unaffected.
- `internal/agent/claude_test.go`, `internal/agent/agent_test.go`, `internal/agent/bob_test.go`, `internal/agent/codex_test.go` — skill-install tests; the rendered output changes byte-for-byte, but the structural checks (frontmatter present, files exist) continue to hold.

## Project References

- Specification: `.spektacular/specs/000021_file-writing.md`.
- Closest prior plan touching the workflow templates and CLI: `.spektacular/plans/000020_context/plan.md` (knowledge layer + provider-based config).
- Rendered active skills the dogfooded project relies on: `.claude/skills/spek-{new,plan,implement}/SKILL.md`.
- The `.spektacular/tmp/` scratch convention is listed in `.gitignore:45`.

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|-------------|----------------|
| Low | ~10k | Single agent, sequential |
| Medium | ~25k | 2-3 parallel agents |
| High | ~50k+ | Parallel analysis, sequential integration |

Per-phase application:

- Phase 1.1 — Low (~3k) — single agent.
- Phase 1.2 — Low (~5k) — single agent; optionally two parallel for spec/plan test files.
- Phase 2.1 — Medium (~15k) — three parallel agents on the template directories, then sequential audit of Go comments and skill regeneration.
- Phase 2.2 — Low (~2k) — single agent.

## Migration Notes

There is no in-flight state to migrate. The previous stdin interface is removed outright: no compatibility shim, no transitional flag. The first commit that lands Milestone 1 breaks any out-of-tree caller that pipes content; this is acceptable because the only known callers are the skill/step instructions Spektacular itself owns, and Milestone 2 updates them in lockstep. The `.spektacular/tmp/` scratch convention is preserved as-is, so no agent prose breaks during the staged rollout if Milestone 1 lands before Milestone 2 — only the invocation form changes.

## Performance Considerations

None. `os.ReadFile` of a small markdown file replaces `io.ReadAll(stdin)` of the same content. Both are O(n) in file size with comparable constant factors, and the spec/plan files are small (kilobytes). The change has no measurable performance impact.
