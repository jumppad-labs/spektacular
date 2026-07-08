# spec-workflow-output-changelog

## What was built

Every feature Spektacular's implement workflow finishes now gets a durable, self-contained changelog record automatically, with no user action required. This is a new capability, built in two layers:

1. **A new changelog store.** Spektacular gained a third document store — alongside the existing spec and plan stores — configured in `config.yaml` under a `changelog` section (provider plus directory, defaulting to `.spektacular/changelog`), and a `changelog file read/write/list` command family identical in shape to the existing `spec file`/`plan file` commands. Records are stored one flat Markdown file per feature (`.spektacular/changelog/<feature-name>.md`), so a specific feature's record can be retrieved by name without scanning the whole history, and records accumulate over time rather than overwriting each other.

2. **A new, mandatory step in the implement workflow.** A step called `update_feature_changelog` now runs automatically at the very end of every implement workflow, immediately before the workflow reports completion. It instructs the assistant to read the feature's original spec (for the "why it matters" framing) and the plan's already-accumulated phase-by-phase implementation log (for what was actually built, including any recorded deviations from the original plan), then author and commit a self-contained record covering what was built, why it matters, what it enables, and how the real outcome differed from what was planned. The workflow's final completion summary now also tells the user exactly where the new record lives.

## Why it matters / what it enables

Previously, once a feature shipped, the only records of what happened were a terse, deliberately implementation-detail-free release note in the repo's `CHANGELOG.md`, and a developer-facing audit log embedded inside the plan document itself (which assumes the reader already has the rest of the plan for context). Neither was dense or self-contained enough to hand to a later documentation or announcement process without re-reading the original spec, plan, and conversation.

This feature closes that gap: anyone — or any downstream process — writing user-facing documentation or a release announcement about a completed feature can now read a single, purpose-built record and get an accurate, complete picture without reconstructing context from scratch. The record is grounded in what was *actually* built (via the plan's real implementation history) rather than what was originally intended, so it stays accurate even when implementation diverged from the initial plan.

## Deviations from the plan

None functionally. One cosmetic, explicitly-anticipated choice was made: the implement workflow's terminal `finished` step template was renamed from `10-finished.md` to `11-finished.md` (preserving git history via `git mv`) so templates stay numbered sequentially now that `10-update_feature_changelog.md` occupies that slot. The plan itself flagged this numbering as a free implementer choice with no behavioral effect, since templates are referenced by explicit path rather than directory sort order — not a deviation from intent.

All four planned phases shipped as scoped: changelog configuration, `changelog file` commands, the new FSM step itself (config plumbing, path-variable computation, FSM wiring, and a new template), and the finished-step summary update. No scope was added or cut.
