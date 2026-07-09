# Context: 000032_spec-workflow-pair-programming-enhancements

## Current State Analysis

Spektacular currently captures specifications only when explicitly invoked via `spektacular spec new`. There is no mechanism for agents to recognize when an open-ended discussion has produced something substantial enough to warrant a specification, and no way to carry forward context from the discussion into the spec workflow. The spec workflow accepts arbitrary data via `--data`, `--stdin`, and `--file` flags, but step templates always prompt from scratch regardless of whether context was pre-populated.

## Per-Phase Technical Notes

### Phase 1.1: Add configuration field for spec-trigger threshold

**File changes**:
- `internal/config/config.go:1-280` — Add `SpecTriggerThreshold string` field to the `Config` struct (after existing fields, before closing brace). Update `NewDefault()` to set `SpecTriggerThreshold: "moderate"`. Update `Validate()` to check that the value, if present, is one of `"strict"`, `"moderate"`, or `"lenient"` and return an error otherwise (following the pattern used for `Spec.IDMethod` validation at lines 180-186).
### Phase 2.1: Modify new step to clear context.md and return write instruction

**File changes**:
- `internal/steps/spec/steps.go:63-82` — In the `new()` step callback, after creating the spec scaffold file with `st.Write(SpecFilePath(...))`, add code to clear `.spektacular/context.md` by writing an empty file: `os.WriteFile(filepath.Join(projectRoot, ".spektacular/context.md"), []byte(""), 0644)` where projectRoot is derived from the spec file path.
- `templates/steps/spec/00-new.md` (new) — Create a new step template for the `new` step that returns an instruction to the agent: "Write the current conversation context to `.spektacular/context.md` if meaningful context exists. Capture the full discussion in detail: what problem was identified and why it needs solving, all requirements and constraints discussed, alternatives considered and why rejected, and the user's exact phrasing for key requirements. If no meaningful context exists, leave context.md empty. Then proceed to the overview step."
- `internal/steps/spec/steps.go:63-82` — Modify the `new()` step to use `writeStep` instead of returning "overview" directly, so it renders the new template and produces output for the agent.

**Complexity**: Low

**Token estimate**: ~2k tokens (file write operation + template creation + callback modification)

**Agent strategy**: Single agent, sequential execution. Simple file operation and template addition.

## Project References

- Spec 000020 (context): knowledge store
- Spec 000022 (spek-knowledge skill): agent-facing skill for knowledge writes
- Spec 000023 (context-over-memory): managed-section pattern in AGENTS.md

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|--------------|----------------|
| Low | ~2k | Single agent, sequential |
| Medium | ~5k | Single agent, sequential |

All phases in this plan are Low or Medium complexity and use single-agent sequential execution.

## Migration Notes

No migration required. The feature is additive and backward-compatible. Existing projects pick up the behavior by running `spektacular init <agent>` once.

## Performance Considerations

No performance impact. The configuration field adds negligible overhead to config loading, and the instruction is loaded once at agent session start.
