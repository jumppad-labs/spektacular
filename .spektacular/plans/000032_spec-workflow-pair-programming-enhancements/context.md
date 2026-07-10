# Context: 000032_spec-workflow-pair-programming-enhancements

## Current State Analysis

Spektacular currently captures specifications only when explicitly invoked via `spektacular spec new`. There is no mechanism for agents to recognize when an open-ended discussion has produced something substantial enough to warrant a specification, and no way for the agent to be instructed to carry conversation context into the spec workflow it starts. The spec workflow's step templates always render the same "ask the user..." prompts regardless of whether the agent already knows the answer from the ongoing conversation; nothing in the CLI or template-rendering pipeline (`internal/stepkit/stepkit.go`) merges arbitrary conversation content into what gets rendered, and nothing needs to — carry-forward is a matter of agent behavior, not CLI plumbing.

An earlier attempt at this plan proposed a CLI-data-passing mechanism for carry-forward (`spec new --data '{"overview":...}'` plus mustache conditionals in step templates). That attempt was abandoned: tracing `stepkit.WriteStepResult` showed it never merges `workflow.Data` into the templates it renders, so the design would not have worked as specified, and more fundamentally the mechanism was solving a problem that doesn't exist (see research.md § Alternative 3 for the full account). This plan replaces that milestone entirely with agent-instruction prose.

## Per-Phase Technical Notes

### Phase 1.1: Add the `spec_trigger_threshold` config field

**File changes**:
- `internal/config/config.go:105-113` — Add `SpecTriggerThreshold string` field to the `Config` struct (yaml tag `spec_trigger_threshold`), alongside `Command`/`Agent`.
- `internal/config/config.go:14-18` (near the existing `SpecIDMethod*` const block) — Add three new string constants for the supported values, e.g. `SpecTriggerThresholdStrict = "strict"`, `SpecTriggerThresholdModerate = "moderate"`, `SpecTriggerThresholdLenient = "lenient"`.
- `internal/config/config.go:116-156` (`NewDefault()`) — Set `SpecTriggerThreshold: "moderate"` (or the new moderate constant) in the returned struct literal, alongside `Command: "spektacular"` at line 118.
- `internal/config/config.go:178-192` (top-level `Config.Validate()`) — Add a validation branch for the new field, copying the exact `switch`-with-empty-case-and-`default:`-error style used by `SpecConfig.Validate()` at lines 203-207 (`c.IDMethod` check): accept `""`, `"strict"`, `"moderate"`, `"lenient"`; otherwise return `fmt.Errorf("spec_trigger_threshold must be one of %q, %q, or %q", ...)`.
- `internal/config/config_test.go` — Add a `TestNewDefault_HasExpectedDefaults` assertion for the new field (extend the existing test, following the `require.Equal(t, "timestamp", cfg.Spec.IDMethod)` pattern), and a new `TestFromYAMLFile_UnknownSpecTriggerThresholdReturnsError` test mirroring `TestFromYAMLFile_UnknownSpecIDMethodReturnsError` (lines 47-61): write YAML with an invalid value, assert `require.Error` and `require.Contains(err.Error(), "spec_trigger_threshold")`.

**Complexity**: Low

**Token estimate**: ~2k tokens (single struct field, three constants, one default, one validation branch, two tests)

**Agent strategy**: Single agent, sequential execution.

### Phase 1.2: Add the AGENTS.md managed instruction section (installer plumbing)

**File changes**:
- `templates/agents/spec-trigger.md` (new) — Embedded mustache template, structured like `templates/agents/memory-context.md` (15 lines, static prose, single heading, one `{{command}}` variable). Phase 1.2 creates this file with the heading `## Spec-Worthy Discussion Recognition` and placeholder body text; Phase 1.3 replaces the body.
- `internal/agent/spec_trigger.go` (new) — `installSpecTriggerSection(projectPath string, cfg config.Config, out io.Writer) error`, a structural clone of `installMemoryContextSection` (`internal/agent/memory_context.go:25-56`): read the embedded template via `fs.ReadFile(sourceFS, "agents/spec-trigger.md")`, render with `mustache.Render(string(tmplBytes), map[string]string{"command": cfg.Command})`, ensure trailing newline, read `AGENTS.md`, and follow the same create/append/replace branching as lines 39-55 of `memory_context.go` — reusing (or cloning as needed) the `locate`/`append`/`replace` helpers at lines 58-116 and the atomic-write helper at lines 120-144, retargeted at the new heading constant (`## Spec-Worthy Discussion Recognition` instead of `## Memory & Context`).
- `internal/agent/claude.go:25` — Add a call to `installSpecTriggerSection(projectPath, cfg, out)` immediately after the existing `installMemoryContextSection` call.
- `internal/agent/codex.go:17` — Add the same call after the existing `installMemoryContextSection` tail call (or convert to sequential calls with error handling, matching the surrounding style).
- `internal/agent/bob.go:20` — Add the same call after the existing `installMemoryContextSection` tail call.
- `internal/agent/spec_trigger_test.go` (new) — Mirror all six test cases from `memory_context_test.go` (lines 30, 42, 62, 81, 117, 130): create-from-missing, append-after-existing-content-without-heading, idempotent-rerun, preserve-surrounding-content, template-change-picked-up, cross-agent-idempotency (claude→codex→bob against one shared `AGENTS.md`, asserting exactly one `## Spec-Worthy Discussion Recognition` heading).

**Complexity**: Medium

**Token estimate**: ~5k tokens (new template, new installer function cloning a proven pattern, three call-site wiring edits, six mirrored tests)

**Agent strategy**: Single agent, sequential execution. The installer function and its tests should be written directly against the `memory_context.go`/`memory_context_test.go` pair as a structural reference, changing only the heading, template path, and function/test names.

### Phase 1.3: Write the spec-trigger instruction content

**File changes**:
- `templates/agents/spec-trigger.md` — Replace Phase 1.2's placeholder body with the full instruction prose. Content to cover, in order: (1) recognition criteria — what "substantial enough" looks like (multiple requirements mentioned, a scoped decision reached, a feature described in detail); (2) reading the threshold — instruct the agent to read `.spektacular/config.yaml`'s `spec_trigger_threshold` at the moment it is deciding whether to offer (not a value baked into this template — no `{{threshold}}` variable is used), and to treat an absent value as `"moderate"`; (3) the offer itself — always propose-then-confirm, present the option and wait for the user's decision; (4) handling the three outcomes — accept (proceed to carry-forward below), defer (continue the conversation normally, may raise the offer again later in the same conversation as the discussion develops), decline (drop the offer for the remainder of that discussion, do not re-prompt for the same topic); (5) carry-forward on accept — start the spec workflow (`{{command}} spec new --data '{"name":"..."}'`) and, for each step's existing prompt (e.g. `templates/steps/spec/01-overview.md`'s "ask the user to describe this feature"), propose a draft answer drawn from the conversation already had, presenting it to the user for confirmation or refinement rather than asking from scratch.
- No Go code changes in this phase — the installer from Phase 1.2 already renders whatever content lives in the template file.

**Complexity**: Low

**Token estimate**: ~2k tokens (prose-only change to one template file; existing tests from Phase 1.2 continue to pass since they assert structural behavior, not exact body content, except any fixture string constants that embed the placeholder body, which need updating to match)

**Agent strategy**: Single agent, sequential execution. Note: `internal/agent/spec_trigger_test.go`'s expected-content fixtures (the independent-oracle string constants asserting rendered output) must be updated in this phase to reflect the real instruction text, since Phase 1.2's tests were written against the placeholder body.

## Testing Strategy

Testing is split by what each phase actually adds. Phase 1.1's config field gets direct table-driven unit tests on default value and validation acceptance/rejection, following `internal/config/config_test.go`'s existing conventions exactly (`TestNewDefault_HasExpectedDefaults`, `TestFromYAMLFile_Unknown<Field>ReturnsError`). Phase 1.2's installer gets the full six-case coverage mirrored from `memory_context_test.go` (create-from-missing, append, idempotent update, preserve-surrounding-content, template-change-picked-up, cross-agent idempotency), written against placeholder content. Phase 1.3 does not add new test cases — it updates Phase 1.2's expected-content fixture strings to match the real instruction prose, so the same six tests continue to assert the same structural guarantees against the final content.

No integration tests are added anywhere in this plan — the instruction's real-world effectiveness (does an agent actually recognize a spec-worthy discussion, offer correctly, honor the threshold, carry context forward) is not something a Go test can assert; it is verified only by the manual smoke tests recorded in the PR description, following the precedent set by the memory-redirect feature (spec 000023).

## Project References

- Spec 000020 (context): knowledge store
- Spec 000022 (spek-knowledge skill): agent-facing skill for knowledge writes, propose-then-confirm pattern precedent
- Spec 000023 (context-over-memory): managed-section pattern in AGENTS.md (`installMemoryContextSection`)

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|--------------|----------------|
| Low | ~2k | Single agent, sequential |
| Medium | ~5k | Single agent, sequential |

All phases in this plan are Low or Medium complexity and use single-agent sequential execution.

## Migration Notes

No migration required. The feature is additive and backward-compatible. Existing projects pick up the behavior by running `spektacular init <agent>` once.

## Performance Considerations

No performance impact. The configuration field adds negligible overhead to config loading. The instruction is loaded once at agent session start, and the threshold read happens at most once per offer decision (a single small file read).
