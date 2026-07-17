# Context: 000036_spec_plan_historical_artifacts

## Current State Analysis

The repository already establishes a durable, session-global agent-instruction surface via `AGENTS.md` at the repo root, imported into Claude Code's own `CLAUDE.md` by an `@AGENTS.md` line (`CLAUDE.md:1`, `internal/agent/claude.go:38`). Two `spektacular init`-managed sections already live inside that file: `## Memory & Context` and `## Spec-Worthy Discussion Recognition`. Each managed section is a triad:

- A Markdown template under `templates/agents/` (`memory-context.md:1-14`, `spec-trigger.md:1-31`), each carrying a heading, a "Managed by … in the Spektacular source" disclaimer blockquote, and prose body rendered with `{{command}}`.
- A Go installer under `internal/agent/` (`memory_context.go:16-144`, `spec_trigger.go:16-117`) with a top-level `install<Name>Section` function plus `locate<Name>Section` / `append<Name>Section` / `replace<Name>Section` helpers. Idempotency and cross-agent convergence are guaranteed by heading-delimited range detection: locate finds the `## <Heading>` line and the next `# ` or `## ` heading, then either appends after existing content or replaces the delimited range in place.
- One-line wire-in inside each registered agent's `Install()` method (`claude.go:18-32`, `bob.go:13-24`, `codex.go:13-25`). The wiring order is fixed: skills first (Claude uses `.claude/skills`; Bob and Codex use their own subdirs), then wrappers if applicable, then each managed-section installer, and for Claude finally `ensureClaudeImportsAGENTS` (`claude.go:45-78`).

Existing tests over this pattern live in `memory_context_test.go:1-153` and `spec_trigger_test.go:1-157` — the same six-case shape covers create-from-missing, append-after-tessl, idempotency, preservation of surrounding content, template-change pickup, and cross-agent convergence. Both files use `fstest.MapFS` fixtures via the package-level `withSourceFS(t, ...)` helper, and the cross-agent test uses `Lookup(<agent-name>)` to exercise the real registered agents. The repo also carries an instruction-surface guard (`instruction_surface_test.go:1-109`) walking the embedded templates FS to reject forbidden CLI patterns; this guard applies to any new template automatically.

The spec/plan/implement workflows already access their artifacts through dedicated CLI commands (`spec file read`, `plan file read`, etc.), and step templates consistently warn against using the `Read` tool directly on those paths. Two step-template locations, however, currently read specs or plans in wording that could look like it contradicts the new "specs and plans are historical" rule: `templates/steps/plan/02-discovery.md:18` (the plan-workflow's discovery step lists and reads *prior* plans and specs to inform new-plan research) and `templates/steps/implement/01-read_plan.md:7-12` (the implement-workflow's first step reads the plan and spec it will drive). Both are covered by the spec's exceptions (archaeology for the former, owning-workflow for the latter), but a reader parsing the templates cold would benefit from that being framed explicitly.

The changelog surface consists of the existing `templates/steps/implement/10-update_feature_changelog.md` step and the resulting per-feature entries under `.spektacular/changelog/` (`000032`–`000035` today). Prior entries follow a stable "What was built / Why it matters / Deviations from the plan" shape (`.spektacular/changelog/000035_plan-walkthrough-conversation.md:1-32`). The changelog note this feature must ship — flagging the "process document, not product document" concept for future website-documentation work — is prose content the existing step will author; no template change is required.

The embedded FS carrying all templates (`templates/templates.go`) uses `//go:embed all:*`, so any new file dropped under `templates/agents/` is auto-included; there is no manifest to update. The `mustache` library (`github.com/cbroglie/mustache`, imported at `internal/agent/memory_context.go:12`) is the only rendering dependency and already handles the `{{command}}` placeholder used across existing templates. The spec being planned is `.spektacular/specs/000036_spec_plan_historical_artifacts.md` — eight requirements, one Constraint, nine acceptance criteria, one Non-Goal (website-docs update deferred), and three runtime-behavioural Success Metrics.

## Per-Phase Technical Notes

### Phase 1.1: Author the Historical Artifacts template and its installer

**File changes**:

- `templates/agents/historical-artifacts.md` (new file, ~30-50 lines): New Markdown template. Structural shape mirrors `templates/agents/memory-context.md:1-14` and `templates/agents/spec-trigger.md:1-31`:
  - Line 1: `## Historical Artifacts: Specs and Plans as Archaeology` (H2 heading — the locate function keys off this exact string).
  - Lines 2-5 (blank + blockquote): `> Managed by \`{{command}} init\` — edit \`templates/agents/historical-artifacts.md\` in the Spektacular source, not this section in place. Hand edits will not survive the next init.`
  - Remaining body: the rule prose. Cover, in order: (a) the historicity framing (specs/plans describe past intent, not current state); (b) no discovery-time reads (agents ground current-state answers in code/tests/config); (c) the archaeology exception (allowed when the user is investigating past intent); (d) the owning-workflow exception (a running spec/plan/implement workflow reads and updates its own live artifact); (e) that the rule applies everywhere, not only inside workflow steps.
  - `{{command}}` mustache variable used wherever an example CLI invocation appears (e.g. `{{command}} spec file read <name>.md` when describing the CLI channel that owning workflows use).

- `internal/agent/historical_artifacts.go` (new file, ~120-130 lines): New Go source. Directly mirrors `internal/agent/memory_context.go:16-144`:
  - Constants at top: `historicalArtifactsTemplatePath = "agents/historical-artifacts.md"` and `historicalArtifactsHeading = "## Historical Artifacts: Specs and Plans as Archaeology"` (must match template line 1 exactly).
  - `installHistoricalArtifactsSection(projectPath string, cfg config.Config, out io.Writer) error` — reads the template from `sourceFS` (the package-level embedded FS the two prior installers use), renders it with `mustache.Render` against `map[string]string{"command": cfg.Command}`, then locates/appends/replaces the section body inside `AGENTS.md`. On missing file, calls `writeAGENTSAtomic(..., "created")`. On append, uses the "appended Historical Artifacts section to" action string. On replace, uses "updated Historical Artifacts section in".
  - `locateHistoricalArtifactsSection(body []byte) (startIdx, endIdx int, found bool)` — line-scan implementation identical in shape to `locateMemoryContextSection` at `internal/agent/memory_context.go:63-81`.
  - `appendHistoricalArtifactsSection` and `replaceHistoricalArtifactsSection` — verbatim shape from `internal/agent/memory_context.go:86-116`.
  - Package-level `import` block matches `internal/agent/memory_context.go:3-14` (bytes, fmt, io, io/fs, os, path/filepath, strings, `github.com/cbroglie/mustache`, `github.com/jumppad-labs/spektacular/internal/config`).

- `internal/agent/historical_artifacts_test.go` (new file, ~150-160 lines): New test file. Directly mirrors `internal/agent/memory_context_test.go:1-153`. Six test cases:
  1. `TestInstallHistoricalArtifactsSection_CreatesFromMissing` — installs against empty tempdir; asserts AGENTS.md now contains only the rendered section.
  2. `TestInstallHistoricalArtifactsSection_AppendsAfterTesslBlock` — pre-writes the tessl-managed header + `@.tessl/RULES.md` line; asserts append preserves the header and adds exactly one blank-line separator.
  3. `TestInstallHistoricalArtifactsSection_IsIdempotent` — two consecutive installs produce byte-identical output.
  4. `TestInstallHistoricalArtifactsSection_PreservesSurroundingContent` — pre-writes AGENTS.md with the header, an existing STALE Historical Artifacts body, and a downstream `## Other Section`; asserts the stale body is replaced and the downstream section is preserved verbatim.
  5. `TestInstallHistoricalArtifactsSection_PicksUpTemplateChange` — installs with `cfg.Command = "go run ."` then with `cfg.Command = "spektacular"`; asserts the second install re-renders the template with the new command.
  6. `TestInstallHistoricalArtifactsSection_CrossAgentIdempotency` — uses the real embedded templates FS (`config.NewDefault()`), invokes each of `Lookup("claude")`, `Lookup("codex")`, `Lookup("bob")` in sequence against the same tempdir, asserts exactly one instance of the new heading in `AGENTS.md` and each agent's own skills directory is preserved.
  - Uses the same `fixtureHistoricalArtifactsFS()` in-memory fs.FS helper with `fstest.MapFS`, embedding a `{{command}}` placeholder in the fixture body so the picks-up-template-change test can observe rendered difference. Matches the pattern at `internal/agent/memory_context_test.go:20-28`.
  - Uses `withSourceFS(t, ...)` to swap the embed FS with the fixture, and reuses `Lookup` for the cross-agent test — both symbols already exist in the package.

**Complexity**: Low.

**Token estimate**: ~10k tokens (three new files, mostly-mirrored patterns, no cross-file reasoning beyond following the two existing exemplars).

**Agent strategy**: Low — single agent, sequential execution. The three files are small, tightly coupled, and mirror an established pattern; there is no independent-changes surface to parallelize.

### Phase 1.2: Wire the installer into all three registered agents

**File changes**:

- `internal/agent/claude.go:28` (edit, ~1 line added): After the existing `if err := installSpecTriggerSection(projectPath, cfg, out); err != nil { return err }` block and before `return ensureClaudeImportsAGENTS(projectPath, out)`, add:
  ```go
  if err := installHistoricalArtifactsSection(projectPath, cfg, out); err != nil {
      return err
  }
  ```

- `internal/agent/bob.go:23` (edit, ~1 line reshape): Change the terminal `return installSpecTriggerSection(...)` to a guarded call followed by `return installHistoricalArtifactsSection(projectPath, cfg, out)` — matches the existing pattern at `internal/agent/claude.go:22-31` where each installer runs as a guarded call and the final one is the return statement.

- `internal/agent/codex.go:20` (edit, ~1 line reshape): Same reshape as Bob — guard the `installSpecTriggerSection` call and return the new installer.

**Complexity**: Low.

**Token estimate**: ~2k tokens (three tiny edits, one line each in effect).

**Agent strategy**: Low — single agent, sequential execution. Trivial mechanical change.

### Phase 2.1: Reframe the read instructions in the two affected step templates

**File changes**:

- `templates/steps/plan/02-discovery.md:18` (edit, prose-only): The current bullet reads:
  > **Prior research** — Existing plans, research, or tickets: search the knowledge sources with `{{config.command}} knowledge search <query>`, list prior plans with `{{config.command}} plan file list` and prior specs with `{{config.command}} spec file list`, then read any that look relevant with `{{config.command}} plan file read <name>/plan.md` or `{{config.command}} spec file read <name>.md`. Always reach plans and specs through these CLI commands — never with the `Read` tool.

  Edit adds a framing sentence at the front of the bullet so the read is described as historical archaeology in service of new-plan research, not as current-state discovery. Example rewritten opener: `**Prior research** — Prior plans and specs are historical archaeology explaining *why* past decisions were made; consult them here to inform this new plan, not to describe what the codebase does today.` The list-and-read CLI commands then follow unchanged. Wording should be tightened at implement time to fit the surrounding paragraph flow; the intent is clarification of purpose, not additional instruction.

- `templates/steps/implement/01-read_plan.md:7-12` (edit, prose-only): The current wording reads:
  > Read the three plan documents **in full** through the plan store. The plan documents are owned by spektacular — always read them with `{{config.command}} plan file read`, never with the `Read` tool, which bypasses the CLI

  Edit adds a short acknowledgment sentence that these documents are the *owning workflow's own live artifact* — the exception the AGENTS.md rule explicitly permits — so the read does not appear to contradict the newly-stated global rule. Placement: as a brief opener before the "Read the three plan documents in full" sentence, or as a trailing clarifier. Wording tightened at implement time.

**Complexity**: Low.

**Token estimate**: ~2k tokens (two small prose edits, no cross-file reasoning).

**Agent strategy**: Low — single agent, sequential execution.

### Phase 3.1: Ship the website-documentation follow-up note in the feature's changelog entry

**File changes**:

- `.spektacular/changelog/<n>_<name>.md` (new artifact, authored at implement time by the existing `templates/steps/implement/10-update_feature_changelog.md` step): The changelog entry that ships with this feature's implementation must contain a clearly-identifiable sentence flagging that the "process document, not product document" concept should be added to the user-facing website documentation in a future update. The sentence belongs in the "What was built" or "Why it matters" section — somewhere a reader scanning the entry lands on it — rather than in a footnote or code block.

No step-template change is required — the `10-update_feature_changelog.md` step already produces this artifact. The phase's role is to make the note an explicit acceptance check on the shipped artifact rather than an implicit expectation that could be lost.

**Complexity**: Low.

**Token estimate**: ~1k tokens (short prose insertion into the changelog artifact at implement time).

**Agent strategy**: Low — single agent, sequential execution. Happens naturally inside the implement workflow's existing `update_feature_changelog` step.

## Testing Strategy

Testing is concentrated at the installer layer in the `internal/agent` package. The bulk of test coverage lives in the new `historical_artifacts_test.go` file, mirroring `memory_context_test.go:1-153` case-for-case. Each of the six tests exercises a distinct property of the installer contract: creation semantics, append semantics after the tessl-managed header, byte-idempotency of consecutive installs, preservation of unrelated sections during in-place replacement, template-change pickup on re-render, and convergence across all three registered agents to a single instance of the new heading.

The `fixtureHistoricalArtifactsFS()` helper is added in-file and returns an `fstest.MapFS` seeded with a fixture template whose body embeds the `{{command}}` placeholder — this is what lets the picks-up-template-change test observe rendered difference between two installer invocations against differing `cfg.Command` values. The fixture is swapped into the package-level `sourceFS` via `withSourceFS(t, ...)` (already exported by the package for use by `memory_context_test.go` and `spec_trigger_test.go`).

Cross-agent idempotency (`TestInstallHistoricalArtifactsSection_CrossAgentIdempotency`) is the load-bearing regression guard: it uses `config.NewDefault()` and the real embedded templates FS, invokes each of `Lookup("claude")`, `Lookup("codex")`, `Lookup("bob")` in sequence against the same tempdir, and asserts (a) exactly one instance of `## Historical Artifacts: …` heading in `AGENTS.md`, (b) each agent's own skills directory is preserved — proving the shared AGENTS.md write did not displace any earlier per-agent install, and (c) crucially, the *existing* `Memory & Context` and `Spec-Worthy Discussion Recognition` heading counts remain exactly one each, proving the new call did not disturb prior sections.

No step-template test is added for Phase 2.1's clarifiers. The two edits are prose-only, do not add template variables or CLI patterns, and inherit the existing `TestEmbeddedTemplatesAvoidStdinInstructionSurface` guard at `internal/agent/instruction_surface_test.go:33-48` — which walks `templates/steps/` and rejects forbidden stdin/heredoc patterns — as their regression cover. If a future edit were to reintroduce a forbidden pattern, that guard fires without any additional test being needed.

Phase 3.1's changelog-note requirement is verified at implement time, not by an automated test: the implementer reads the written entry under `.spektacular/changelog/` and confirms the "process document, not product document" flag sentence is present and prominently placed. This is captured as an acceptance criterion in plan.md rather than as a test case here, because the artifact does not exist until the implement workflow's `update_feature_changelog` step has run.

The three Success Metrics from the spec are all runtime-behavioural observations of agent citation habits across real user sessions and cannot be asserted from within Go tests. They are classified in plan.md's Testing Approach as *Manual — captured in the implementation test plan*, which the implement workflow's `test_plan` step (see `templates/steps/implement/09-test_plan.md`) will materialize into a concrete manual-verification artifact once the code exists.

## Project References

Load-bearing references the implementer will want open while working through the phases. All paths are relative to the repo root.

- **Exemplar templates to model against**
  - `templates/agents/memory-context.md` — one of two shape references for the new `templates/agents/historical-artifacts.md`. Shows the heading + managed-by disclaimer blockquote + prose body pattern in its minimal form (`{{command}}` unused here, but the shape is identical).
  - `templates/agents/spec-trigger.md` — the other shape reference. Shows the same pattern with the `{{command}}` mustache variable, which the new template also uses when referencing example CLI invocations.

- **Exemplar installers to mirror**
  - `internal/agent/memory_context.go` — full triad reference: constants at top, `installMemoryContextSection`, `locateMemoryContextSection`, `appendMemoryContextSection`, `replaceMemoryContextSection`, plus the shared `writeAGENTSAtomic` and `writeFileAtomic` helpers the new installer must reuse rather than duplicate.
  - `internal/agent/spec_trigger.go` — parallel installer; useful as a second confirmation the pattern is stable.

- **Wire-in points for Phase 1.2**
  - `internal/agent/claude.go` — `Install()` at lines 18-32; the new installer call goes after `installSpecTriggerSection` and before `ensureClaudeImportsAGENTS`.
  - `internal/agent/bob.go` — `Install()` at lines 13-24; the terminal `return installSpecTriggerSection(...)` is reshaped so the new installer becomes the terminal return.
  - `internal/agent/codex.go` — `Install()` at lines 13-25; same reshape as Bob.

- **Test-file exemplars to mirror**
  - `internal/agent/memory_context_test.go` — the six-case test shape the new `historical_artifacts_test.go` mirrors. `fixtureMemoryFS()` (lines 20-28) is the shape for `fixtureHistoricalArtifactsFS()`; `withSourceFS(t, ...)` is a package-level helper reused directly; `Lookup(<agent>)` is used by the cross-agent test.
  - `internal/agent/spec_trigger_test.go` — parallel reference.
  - `internal/agent/instruction_surface_test.go` — repo-level guard that walks `templates/steps/` and `skills/workflows/` and rejects forbidden CLI patterns; the new template inherits this guard automatically.

- **Step templates edited in Phase 2.1**
  - `templates/steps/plan/02-discovery.md` — the plan-workflow discovery step; line 18 is the target bullet whose framing is tweaked.
  - `templates/steps/implement/01-read_plan.md` — the implement-workflow's first read; lines 7-12 gain a brief owning-workflow acknowledgment.

- **Changelog step and shape reference for Phase 3.1**
  - `templates/steps/implement/10-update_feature_changelog.md` — the step that authors the per-feature changelog entry; unchanged, but the phase's acceptance check runs against the artifact it writes.
  - `.spektacular/changelog/000035_plan-walkthrough-conversation.md` — canonical shape (What was built / Why it matters / Deviations from the plan) the shipped entry follows.

- **Embed FS surface**
  - `templates/templates.go` — carries `//go:embed all:*`; the new template under `templates/agents/` is auto-included without any manifest edit. Worth opening once to confirm the pattern before Phase 1.1.

- **Spec being planned**
  - `.spektacular/specs/000036_spec_plan_historical_artifacts.md` — the source of truth for requirements, acceptance criteria, constraints, non-goals, and success metrics. The reconciliation and changelog steps at implement-time both read this file directly.

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|-------------|----------------|
| Low | ~10k | Single agent, sequential |
| Medium | ~25k | 2-3 parallel agents |
| High | ~50k+ | Parallel analysis, sequential integration |

Every phase in this plan is Low complexity and fits well inside the ~10k budget. Phase 1.1 (~10k) is the largest — three new files, but each is small and mirrors an existing exemplar the implementer can read once and re-apply. The remaining phases (1.2, 2.1, 3.1) sit under ~2k each. No phase needs parallel agents; there is no independent-changes surface where parallelism would help.

## Migration Notes

There is no data migration. There is one operational rollout step: after this feature ships, `spektacular init` should be re-run once in every consuming repository so that the new `## Historical Artifacts: Specs and Plans as Archaeology` section is injected into that repo's `AGENTS.md`. The installer is idempotent, so re-running in a repo that has already been re-initialized (or in this repo, which is dogfooded) is safe.

Existing `AGENTS.md` files that do not yet carry the new section are handled by the installer's append path (append-after-tessl-managed block, exactly as the two prior installers do); no manual pre-shaping of any existing `AGENTS.md` is required. Existing repos with prior versions of the two current managed sections continue to work — the new installer does not touch those sections' locate/append/replace logic.

## Performance Considerations

None. The change is a small amount of prose in a Markdown template and a Go install function that runs at `spektacular init` time (offline, one-shot, on a file of a few kilobytes). There is no hot path, no request path, and no measurable performance impact. The idempotent atomic-write helper reused by the new installer is the same one all `AGENTS.md` mutations already use, so no new I/O pattern is introduced.
