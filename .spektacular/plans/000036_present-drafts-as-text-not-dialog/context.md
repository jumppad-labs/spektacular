# Context: 000036_present-drafts-as-text-not-dialog

## Current State Analysis

`internal/agent/memory_context.go` (145 lines) and `internal/agent/spec_trigger.go` (117 lines) are near-byte-identical: `installMemoryContextSection` (memory_context.go:25-56) and `installSpecTriggerSection` (spec_trigger.go:26-57) share identical control flow, differing only in the template-path constant, heading constant, and two hand-written action-noun log strings. `locateMemoryContextSection`/`locateSpecTriggerSection` (memory_context.go:63-81 / spec_trigger.go:64-82) share a 100% identical byte-range heading-scan algorithm, differing only in which heading constant is referenced. `appendMemoryContextSection`/`appendSpecTriggerSection` (memory_context.go:86-96 / spec_trigger.go:87-97) and `replaceMemoryContextSection`/`replaceSpecTriggerSection` (memory_context.go:101-116 / spec_trigger.go:102-117) are 100% byte-identical bodies with zero differing tokens.

`writeAGENTSAtomic` (memory_context.go:120-130) and `writeFileAtomic` (memory_context.go:132-144) are already shared, unexported helpers defined once in `memory_context.go`, used by both files plus `claude.go:52,73` (`ensureClaudeImportsAGENTS`, for `CLAUDE.md`, unrelated to the two managed sections). Any refactor that removes these from `memory_context.go` must relocate them somewhere all three call sites can still reach — trivial since all are in package `agent`.

All three agent surfaces call the two installers in the same order against the same `AGENTS.md` path, with no per-surface variation:
- `claude.go:18-32` (`claudeAgent.Install`) — skills → `installMemoryContextSection` (line 25) → `installSpecTriggerSection` (line 28) → `ensureClaudeImportsAGENTS` (line 31, Claude-only).
- `codex.go:13-21` (`codexAgent.Install`) — skills → `installMemoryContextSection` (line 17) → `installSpecTriggerSection` (line 20, returned directly).
- `bob.go:13-24` (`bobAgent.Install`) — skills → command wrappers (Bob-only) → `installMemoryContextSection` (line 20) → `installSpecTriggerSection` (line 23, returned directly).

`templates/agents/memory-context.md` (14 lines) and `templates/agents/spec-trigger.md` (32 lines) share a structural contract: H2 heading, blank line, 3-line "Managed by ... — edit `templates/agents/<file>.md` ... Hand edits will not survive the next init." blockquote, blank line, body prose. Load-bearing quirk: `memory-context.md:3` hardcodes the literal `` `spektacular init` `` (no `{{command}}` placeholder), while `spec-trigger.md:3` uses `` `{{command}} init` `` (rendered) — a deliberate, pre-existing asymmetry from plan 000023 (`.spektacular/plans/000023_context-over-memory/plan.md:58`) that this refactor must not "fix". Confirmed live in this repo's own root `AGENTS.md:9` (hardcoded `spektacular init`) vs `AGENTS.md:24` (rendered `go run . init`, since this repo's `cfg.Command = "go run ."`).

Existing tests (`memory_context_test.go`, 154 lines; `spec_trigger_test.go`, 157 lines) call `installMemoryContextSection`/`installSpecTriggerSection` by name (test function names at lines 30, 42, 62, 81, 117, 130 in both files — `CreatesFromMissing`, `AppendsAfterTesslBlock`, `IsIdempotent`, `PreservesSurroundingContent`, `PicksUpTemplateChange`, `CrossAgentIdempotency`), with hand-maintained literal expected-output strings (never renderer-computed, per `.spektacular/plans/000023_context-over-memory/plan.md:152`). The `locate`/`append`/`replace` trio in each file is not referenced by name anywhere outside its own file. `agent_test.go:30-35` defines `withSourceFS`, the fixture-swap helper both test files use, and needs no change. The heading-boundary scan in `locate*Section` correctly stops at the tessl-managed block's own `# Agent Rules <!-- tessl-managed -->` H1 (`AGENTS.md:3`) — both test files seed fixtures with this literal tessl block as prior content (`memory_context_test.go:47`, `spec_trigger_test.go:47`) to exercise this.

`templates/templates.go`'s `//go:embed all:*` directive already picks up any new file placed under `templates/agents/` with no directive change needed.

## Per-Phase Technical Notes

### Phase 1.1: Extract the generic managed-section installer

**File changes:**

- New file `internal/agent/managed_section.go` (package `agent`). Add:
  - `func installManagedSection(projectPath string, cfg config.Config, out io.Writer, templatePath, heading, actionNoun string) error` — body ports the control flow currently in `internal/agent/memory_context.go:25-56` (`installMemoryContextSection`) verbatim, generalized: read `sourceFS` at `templatePath` (not the `memoryContextTemplatePath` constant), render with `mustache.Render(..., map[string]string{"command": cfg.Command})`, ensure trailing `\n`, compute `agentsPath := filepath.Join(projectPath, "AGENTS.md")`, on missing file call `writeAGENTSAtomic(agentsPath, []byte(rendered), out, "created")`, otherwise call `locateManagedSection(existing, heading)` and either `appendManagedSection`+`writeAGENTSAtomic(..., "appended "+actionNoun+" to")` or `replaceManagedSection`+`writeAGENTSAtomic(..., "updated "+actionNoun+" in")`. Note: today's action strings are `"appended Memory & Context section to"` / `"updated Memory & Context section in"` (memory_context.go:51,55) — `actionNoun` should be passed as `"Memory & Context section"` by the Phase 1.2 wrapper so the generic function's own string concatenation reproduces this exactly; verify the concatenated string is byte-identical to today's hard-coded ones before proceeding.
  - `func locateManagedSection(body []byte, heading string) (startIdx, endIdx int, found bool)` — port of `memory_context.go:63-81` (`locateMemoryContextSection`), with `heading` as a parameter instead of referencing the `memoryContextHeading` constant. Algorithm must stay byte-identical: `strings.SplitAfter(string(body), "\n")` walk, exact-match on right-trimmed (`\n\r \t`) line vs `heading`, boundary on next line prefixed `"# "` or `"## "`.
  - `func appendManagedSection(body []byte, rendered string) []byte` — direct port of `memory_context.go:86-96`, no parameterization needed (body never referenced anything section-specific).
  - `func replaceManagedSection(body []byte, startIdx, endIdx int, rendered string) []byte` — direct port of `memory_context.go:101-116`, same note.
  - Relocate `writeAGENTSAtomic` (currently `memory_context.go:120-130`) and `writeFileAtomic` (currently `memory_context.go:132-144`) into this new file unchanged. `writeFileAtomic` is also called from `internal/agent/claude.go:52,73` (`ensureClaudeImportsAGENTS`) — since both live in package `agent`, the move requires no import or call-site change in `claude.go`.
- `internal/agent/memory_context.go` (144 lines currently): after the move, this file retains only `memoryContextTemplatePath`/`memoryContextHeading` constants (currently lines 16-19) — the `install`/`locate`/`append`/`replace` function bodies and the two relocated helpers are deleted from this file in this phase (their replacement thin wrapper is added in Phase 1.2, not this phase — Phase 1.1 only adds the new generic file; do not delete the old functions from `memory_context.go`/`spec_trigger.go` yet if that would leave `claude.go`/`codex.go`/`bob.go` uncompilable mid-phase — sequence Phase 1.1 as "add the new file, leave the old files untouched and still fully functional" so the tree compiles and all tests pass at the end of 1.1 exactly as they did before it).
- No template file changes in this phase — `templates/agents/memory-context.md` and `templates/agents/spec-trigger.md` are untouched.
- No test file changes in this phase. New tests for `installManagedSection`/`locateManagedSection`/`appendManagedSection`/`replaceManagedSection` can be added in this phase (a new `managed_section_test.go`, `package agent`, `testify/require`, `t.TempDir()`, `withSourceFS`-style fixture swap against a synthetic template path/heading rather than a real section) to exercise the generalized logic directly, mirroring the create/append/idempotent/preserve-surrounding-content/template-change-pickup shape already established in `memory_context_test.go:30-129` — but these are additive, not replacements.

**Complexity**: Low — this is a mechanical extraction of already-understood, already-tested logic (research.md confirms the two source functions are byte-identical except for 3 substitution points); no new algorithm is designed here.
**Token estimate**: ~15k
**Agent strategy**: Low — single agent, sequential execution. The extraction has one clear source of truth (the existing byte-identical pair) and one clear output file; no benefit to parallelizing.

### Phase 1.2: Re-point the two existing sections at the generic installer

**File changes:**

- `internal/agent/memory_context.go` — replace the file's body (currently `install`/`locate`/`append`/`replace` functions at lines 25-117, plus the now-relocated `writeAGENTSAtomic`/`writeFileAtomic` at 120-144) with: keep the two constants (`memoryContextTemplatePath`, `memoryContextHeading`, currently lines 16-19), and replace `installMemoryContextSection`'s body (currently lines 25-56) with a single call: `return installManagedSection(projectPath, cfg, out, memoryContextTemplatePath, memoryContextHeading, "Memory & Context section")`. Delete `locateMemoryContextSection`, `appendMemoryContextSection`, `replaceMemoryContextSection` entirely — confirmed in research.md that nothing outside this file calls them by name (only `installMemoryContextSection` is called externally, from `claude.go:25`, `codex.go:17`, `bob.go:20`, and only `installMemoryContextSection` is called by name from `memory_context_test.go`).
- `internal/agent/spec_trigger.go` — identical treatment: keep `specTriggerTemplatePath`/`specTriggerHeading` (currently lines 16-19; re-check with `grep -n` before editing since line numbers may have shifted after Phase 1.1's additions elsewhere), replace `installSpecTriggerSection`'s body (currently lines 26-57) with `return installManagedSection(projectPath, cfg, out, specTriggerTemplatePath, specTriggerHeading, "Spec-Worthy Discussion Recognition section")`. Delete `locateSpecTriggerSection`, `appendSpecTriggerSection`, `replaceSpecTriggerSection` entirely (same "nothing external calls them by name" justification, confirmed by research.md).
- No changes to `internal/agent/claude.go`, `codex.go`, `bob.go` in this phase — `installMemoryContextSection(projectPath, cfg, out)` and `installSpecTriggerSection(projectPath, cfg, out)` keep their exact existing signatures, so `claude.go:25,28`, `codex.go:17,20`, `bob.go:20,23` need zero edits.
- No changes to `memory_context_test.go` or `spec_trigger_test.go` — both call `installMemoryContextSection`/`installSpecTriggerSection` by name only (confirmed research.md § Files examined); this phase's exit criterion is that both test files pass unmodified.
- Verification for this phase: run the full `internal/agent` package test suite; additionally, diff the output of a fresh install run against this repository's own live root `AGENTS.md` (52 lines, already read in full during discovery) — must be byte-identical, confirming the "created"/"appended ... to"/"updated ... in" log-string reconstruction from Phase 1.1 is exactly right.

**Complexity**: Low — deleting now-redundant code and swapping two function bodies for one-line delegating calls.
**Token estimate**: ~8k
**Agent strategy**: Low — single agent, sequential. Must run strictly after Phase 1.1 (depends on `installManagedSection` existing) — no parallelization opportunity between 1.1 and 1.2, and the two file edits within 1.2 are small enough not to warrant splitting across agents either.

### Phase 2.1: Write the "Presenting Drafts and Confirmations" template and installer

**File changes:**

- New file `templates/agents/draft-presentation.md`. Structure mirrors `templates/agents/spec-trigger.md` (32 lines) exactly: line 1 `## Presenting Drafts and Confirmations`, blank line, 3-line blockquote `> Managed by \`{{command}} init\` — edit \`templates/agents/draft-presentation.md\`\n> in the Spektacular source, not this section in place. Hand edits will not\n> survive the next init.`, blank line, then body prose. Body content covers the three spec requirements verbatim in intent (spec 000036 Requirements): (1) drafted content is shown as plain, readable text, not compressed inside a structured choice/dialog element; (2) confirmation is asked separately, after the draft is shown, never as a substitute for showing it; (3) this applies across every workflow step that follows a draft-then-confirm pattern (spec, plan, and implement workflows alike), not just one. Use the `{{command}}` mustache placeholder wherever a CLI invocation is referenced (matching `spec-trigger.md`'s style — confirmed in research.md this is the correct style to follow, not `memory-context.md`'s hardcoded-string quirk).
- New file `internal/agent/draft_presentation.go` (package `agent`). Same shape as the post-Phase-1.2 `memory_context.go`/`spec_trigger.go`:
  ```go
  const (
      draftPresentationTemplatePath = "agents/draft-presentation.md"
      draftPresentationHeading      = "## Presenting Drafts and Confirmations"
  )

  func installDraftPresentationSection(projectPath string, cfg config.Config, out io.Writer) error {
      return installManagedSection(projectPath, cfg, out, draftPresentationTemplatePath, draftPresentationHeading, "Presenting Drafts and Confirmations section")
  }
  ```
- No embed-directive change needed — `templates/templates.go`'s `//go:embed all:*` already picks up any new file under `templates/agents/` (confirmed research.md § Chosen approach — evidence).
- New file `internal/agent/draft_presentation_test.go` — mirror the 6-test shape of `memory_context_test.go`/`spec_trigger_test.go` (both 130-157 lines): `TestInstallDraftPresentationSection_CreatesFromMissing`, `_AppendsAfterTesslBlock`, `_IsIdempotent`, `_PreservesSurroundingContent`, `_PicksUpTemplateChange`, `_CrossAgentIdempotency`. Use `withSourceFS` (agent_test.go:30-35) with a fixture `fstest.MapFS` keyed at `draftPresentationTemplatePath`, and hand-maintain the expected rendered-output literal string per the established convention (do not call `mustache.Render` inside the test — research.md § Files examined, citing `.spektacular/plans/000023_context-over-memory/plan.md:152`).

**Complexity**: Low — this is a template-content authoring task plus a 3-constant/1-function file, following an established, now-singular pattern from Phase 1.2. The only genuine judgment call (exact heading/filename) was already resolved during the architecture step.
**Token estimate**: ~12k
**Agent strategy**: Low — single agent, sequential. The template prose and the Go wrapper are small and interdependent (the heading string must match exactly between them); no parallelization benefit.

### Phase 2.2: Wire the new section into every agent surface

**File changes:**

- `internal/agent/claude.go` — inside `claudeAgent.Install` (currently lines 18-32), insert a new step immediately after the existing `installSpecTriggerSection` call (currently line 28) and before the `ensureClaudeImportsAGENTS` return (currently line 31):
  ```go
  if err := installDraftPresentationSection(projectPath, cfg, out); err != nil {
      return err
  }
  ```
- `internal/agent/codex.go` — inside `codexAgent.Install` (currently lines 13-21), the current final line `return installSpecTriggerSection(projectPath, cfg, out)` (line 20) changes to an error-checked call followed by the new section's call:
  ```go
  if err := installSpecTriggerSection(projectPath, cfg, out); err != nil {
      return err
  }
  return installDraftPresentationSection(projectPath, cfg, out)
  ```
- `internal/agent/bob.go` — inside `bobAgent.Install` (currently lines 13-24), same treatment as codex.go: line 23's `return installSpecTriggerSection(projectPath, cfg, out)` becomes an error-checked call followed by `return installDraftPresentationSection(projectPath, cfg, out)`.
- Extend the `CrossAgentIdempotency` test pattern to cover all three sections: either add a third assertion (heading-count == 1 for `## Presenting Drafts and Confirmations`) to the existing `TestInstallMemoryContextSection_CrossAgentIdempotency` (memory_context_test.go:130-153) and/or `TestInstallSpecTriggerSection_CrossAgentIdempotency` (spec_trigger_test.go:130-156) — following the precedent that spec_trigger_test.go's version already cross-checks both prior headings (spec_trigger_test.go:148-149) — or add the equivalent `TestInstallDraftPresentationSection_CrossAgentIdempotency` in the new `draft_presentation_test.go` from Phase 2.1 with all three heading-count assertions. Either placement is acceptable; the load-bearing requirement is that at least one test, run against the real `templates.FS` via all three agents' actual `.Install()` methods, asserts all three headings each appear exactly once after Claude+Codex+Bob install in sequence into the same directory.
- Manual/final verification for this phase: run installation for each of the three agent surfaces against a fresh temp directory and inspect the resulting `AGENTS.md` for correct section count, order, and idempotency on a second run — this is the concrete check for Milestone 2's validation point.

**Complexity**: Low — three small, symmetric one-line-per-file additions plus one test extension; no new logic, pure wiring.
**Token estimate**: ~10k
**Agent strategy**: Medium — 2-3 parallel agents for the three independent per-file edits (claude.go, codex.go, bob.go have no dependency on each other, only on Phase 2.1's `installDraftPresentationSection` existing), followed by sequential integration to extend/add the cross-agent test and run the full suite once all three files are wired.

## Project References

- Spec `000036_present-drafts-as-text-not-dialog.md` (`.spektacular/specs/`) — the source specification this plan implements; read in full during the overview step. Its Technical Approach section mandates the refactor-first order this plan follows.
- Plan `000023_context-over-memory` (`.spektacular/plans/000023_context-over-memory/`) — established the "Memory & Context" section originally. `plan.md:152` is the source of the hand-maintain-expected-output-string testing convention followed by this plan's new tests; `plan.md:58` documents the deliberate no-`{{command}}`-placeholder decision for `memory-context.md` that this refactor must not alter. `context.md` confirms Codex and Bob both read `AGENTS.md` natively and that `templates/templates.go`'s `//go:embed all:*` auto-picks-up new template files.
- `tests/harbor/plan-workflow/tests/test_plan_workflow.py` and the sibling `spec-workflow` suite — a separate, Docker-based end-to-end test harness for the `plan`/`spec` workflow state machine. Confirmed no code-path overlap with this feature (it never exercises `internal/agent`), but the implementer should still run it or `go build ./...`/`go vet ./...` as a cheap sanity check.

## Testing Strategy

Per-phase testing lands additively, never as a rewrite of the existing 10 tests:

- **Phase 1.1** — additive only: a new `managed_section_test.go` exercises the extracted generic logic directly via a synthetic heading/template, mirroring the create/append/idempotent/preserve/template-change shape already in `memory_context_test.go:30-129`. No existing test file touched.
- **Phase 1.2** — no new tests written; this phase's own test is the unmodified existing suite passing, plus a manual byte-diff of a fresh install run against this repo's live root `AGENTS.md`.
- **Phase 2.1** — new `draft_presentation_test.go` with the same 6-test shape as the two existing section test files (`CreatesFromMissing`, `AppendsAfterTesslBlock`, `IsIdempotent`, `PreservesSurroundingContent`, `PicksUpTemplateChange`, `CrossAgentIdempotency`), hand-maintained literal expected strings, `withSourceFS` fixture swap.
- **Phase 2.2** — extends an existing `CrossAgentIdempotency` test (or the new one from 2.1) to assert all three headings coexist with count 1 each after Claude+Codex+Bob install in sequence; plus a manual fresh-install-per-surface inspection as the concrete Milestone 2 validation point.
- **Whole-plan gate** — the `internal/agent` package's full test suite must pass with zero existing test files modified; the harbor end-to-end suite (`tests/harbor/`) has no code-path overlap with this feature but should still be run (or at minimum `go build ./...` / `go vet ./...`) as cheap insurance against incidental breakage.

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|-------------|----------------|
| Low | ~10k | Single agent, sequential |
| Medium | ~25k | 2-3 parallel agents |
| High | ~50k+ | Parallel analysis, sequential integration |

## Migration Notes

None. This is a same-repository, same-package internal refactor plus an additive template/install-step change. No data migration, no config migration, no versioning concern — every existing project that runs `spektacular init` again simply gains one more managed section in its `AGENTS.md`, following the same append/idempotency behavior the existing two sections already exhibit.

## Performance Considerations

None. This is a filesystem read/render/write operation already bounded by the size of a single `AGENTS.md` file and a handful of small embedded templates (tens of lines each); adding one more section call per agent's `Install` sequence adds a constant, negligible amount of work (one more embedded-file read, one more mustache render, one more locate/append-or-replace pass over the same small file) with no algorithmic complexity change.
