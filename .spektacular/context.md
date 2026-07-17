# Implement: 000036_spec_plan_historical_artifacts — working context

## Read-plan gate passed

- **Structural validation**: all 10 required plan.md `##` sections present. Every `#### - [ ] Phase N.M:` heading has a `*Technical detail:*` link, and every link target resolves to a matching `### Phase N.M:` heading in context.md.
- **Drift check**: every named file, symbol, template path, and CLI command in plan.md/context.md exists in the current tree. The three new targets (`templates/agents/historical-artifacts.md`, `internal/agent/historical_artifacts.go`, `internal/agent/historical_artifacts_test.go`) are correctly labelled as new files.
- **Spec coverage**: all 8 requirements and all 9 acceptance criteria have coverage in plan.md's Milestones & Phases — reqs 1–7 / ACs 1–8 via the AGENTS.md rule (Phase 1.1), the workflow-live-artifact exception via the same rule + Phase 2.1 clarifiers, and req 8 / AC 9 via Phase 3.1 (changelog note).
- **Changelog mode**: first-phase invocation. plan.md has no `## Changelog` section yet; the `update_changelog` step will create it on first use.

## Plan summary (for downstream steps)

**Chosen approach**: two-surface fix. Primary surface is a new `spektacular init`-managed section in `AGENTS.md` mirroring the existing `Memory & Context` and `Spec-Worthy Discussion Recognition` triad (template + Go installer + one-line wire-in per agent). Secondary surface is prose-only clarifiers in `templates/steps/plan/02-discovery.md` and `templates/steps/implement/01-read_plan.md`.

**Phases in order**:

- **1.1** — Author `templates/agents/historical-artifacts.md`, `internal/agent/historical_artifacts.go`, and `internal/agent/historical_artifacts_test.go`. Mirror `internal/agent/memory_context.go` and `internal/agent/memory_context_test.go` case-for-case.
- **1.2** — Wire the new installer into `internal/agent/{claude,bob,codex}.go` Install() methods.
- **2.1** — Prose tweaks: frame `plan/02-discovery.md:18`'s prior-plan/spec reads as historical archaeology; frame `implement/01-read_plan.md:7-12`'s reads as the owning workflow's own live artifact.
- **3.1** — At `update_feature_changelog` step, include a sentence in the shipped changelog entry flagging the "process document, not product document" concept for future website-docs work.

## Key exemplar files (open when working the phases)

- `templates/agents/memory-context.md` — template shape reference.
- `templates/agents/spec-trigger.md` — same shape, with `{{command}}` mustache variable.
- `internal/agent/memory_context.go` — installer + locate/append/replace helpers to mirror.
- `internal/agent/memory_context_test.go` — six-case test shape to mirror.
- `internal/agent/{claude,bob,codex}.go` — wire-in points.

## Assumptions carried from the plan (STOP if any turn out wrong)

- The archaeology exception can be expressed in prose without any code-level enforcement.
- The owning-workflow exception is self-evident because only the owning workflow uses the `spec file` / `plan file` CLI during its run.
- The changelog note requirement is satisfied by prose in the per-feature changelog entry; no schema/CLI change needed.
- Two soft step-template edits are clarifying rather than noisy; fallback if they turn out noisy is to drop them.

## Auto-Mode call: Phase 1.1 and 1.2 co-landed

The plan splits Phase 1.1 (author template/installer/tests) from Phase 1.2 (wire the installer into the three agents). Phase 1.1's own acceptance criterion — "the six-case unit test set is present and green" — includes the cross-agent idempotency test, which cannot pass until Phase 1.2's wiring exists. The two phases are naturally atomic; they were split only for narrative clarity in the plan.

Under Auto Mode I landed both together in a single implement iteration:

- Phase 1.1 files: `templates/agents/historical-artifacts.md`, `internal/agent/historical_artifacts.go`, `internal/agent/historical_artifacts_test.go` (all six tests green).
- Phase 1.2 wire-in: one call added inside each of `internal/agent/{claude,bob,codex}.go`'s `Install()` method.

`make test` and `make lint` (`go vet ./...`) both pass. When update_plan runs for the current phase (1.1), I will tick both 1.1 and 1.2's checkboxes in the same edit, and the next analyze iteration will pick up Phase 2.1 as the first unchecked phase.
