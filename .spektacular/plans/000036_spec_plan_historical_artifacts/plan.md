# Plan: 000036_spec_plan_historical_artifacts

<!-- Metadata -->
<!-- Created: 2026-07-17T09:42:58Z -->
<!-- Commit: 5d15377 -->
<!-- Branch: f-artifacts -->
<!-- Repository: git@github.com:jumppad-labs/spektacular.git -->

## Overview

Establish a durable rule, delivered as instruction-surface prose, so every agent working in a Spektacular-initialized repository automatically treats specs and plans as historical archaeology explaining *why* past changes were made, not as current-state documentation of what the codebase does today. The rule ships as a new `spektacular init`-managed section in `AGENTS.md` — reaching every session including ad-hoc questions and unrelated skills — paired with two surgical clarifiers in existing step templates whose current wording could otherwise look contradictory. Users benefit because they no longer need to restate the rule in each session, and agents' current-state answers stay grounded in code, tests, and configuration rather than drifting into stale intent that may have shifted during implementation.

## Conventions

No project conventions apply to this feature. The project's always-applied knowledge base currently contains only category-README stubs for `conventions/` and `glossary/` (verified via `go run . knowledge always-applied` during discovery — no concrete convention entries returned). The feature is instruction-surface prose that follows established in-repo patterns rather than any documented coding standard, and those patterns are already cited inline in the Architecture & Design Decisions section as file:line references rather than as conventions.

## Architecture & Design Decisions

The feature ships as agent-instruction prose in two coordinated surfaces. The primary surface is a new **`spektacular init`-managed section in `AGENTS.md`** — the always-loaded, session-global instruction file that every supported agent picks up automatically. This section states the rule in full: specs and plans are historical archaeology explaining *why* past changes were made, not current-state documentation; agents ground current-state answers in code, tests, and configuration; specs and plans are read only when the user is investigating past intent; and the owning workflow may read and update its own live artifact while that workflow is actively running. The section mirrors the pattern already established for the `Memory & Context` (`internal/agent/memory_context.go:16-56`, `templates/agents/memory-context.md:1-14`) and `Spec-Worthy Discussion Recognition` (`internal/agent/spec_trigger.go:16-57`, `templates/agents/spec-trigger.md:1-31`) managed sections — a template file under `templates/agents/`, a Go install function in `internal/agent/` with locate/append/replace helpers, and one-line wire-in in each of `internal/agent/{claude,bob,codex}.go`'s `Install()` method. Because `templates/templates.go` embeds `all:*`, the new template is auto-included in the embed FS with no manifest change.

The secondary surface is a small pair of **surgical clarifiers in existing step templates** whose current wording could look like it contradicts the new rule. `templates/steps/plan/02-discovery.md:18` currently instructs the plan-workflow's discovery step to list and read prior plans and specs via `plan file list` / `spec file list` — this is genuinely historical archaeology in service of new-plan research, and gets a brief framing tweak so it reads that way rather than as current-state discovery. `templates/steps/implement/01-read_plan.md:7-12` reads the plan and spec that the running implement workflow owns — a brief acknowledgment that this is the owning workflow's own live artifact prevents the read from looking like a rule violation. The other implement-workflow reads (`02-analyze`, `06-update_plan`, `07-update_changelog`, `09-test_plan`, `10-update_feature_changelog`, `11-reconcile_spec`, `12-finished`) inherit that framing without needing individual clarifiers, keeping the maintenance surface small.

The chosen direction beats an AGENTS.md-only fix because the plan-discovery step's prior-plan/spec read would otherwise appear to violate the newly-stated rule (an agent trying to reconcile the two surfaces would be forced to guess), and it beats a broader belt-and-braces version that also touches every `spek-*/SKILL.md` because AGENTS.md is already loaded session-wide by every agent — additional skill-level notes would duplicate the rule without extending its reach. Two alternatives were rejected outright: filesystem-level access blocks on `.spektacular/specs/` and `.spektacular/plans/` (rejected by the spec's Constraint at `.spektacular/specs/000036_spec_plan_historical_artifacts.md:63` because they would break the archaeology exception) and storing the rule in `.spektacular/config.yaml` (rejected because config is not surfaced to fresh agent sessions the way `AGENTS.md` is via `CLAUDE.md`'s `@AGENTS.md` import at `internal/agent/claude.go:38`). See `research.md#alternatives-considered-and-rejected` for the full list with citations.

The changelog note that ships when this feature is implemented — required by spec requirement #8 and acceptance criterion #9 — is authored in the per-feature changelog entry written by the existing `templates/steps/implement/10-update_feature_changelog.md` step (`.spektacular/changelog/<n>_<name>.md`), following the shape of `.spektacular/changelog/000035_plan-walkthrough-conversation.md`. The entry must explicitly call out that the "process document, not product document" concept should be added to the user-facing website documentation in a future update, so the follow-up is not lost when this spec closes. No new schema, CLI flag, or step template is required to carry the note; the requirement is prose content in the changelog entry, verified as an acceptance check on the shipped artifact.

## Component Breakdown

- **Historical Artifacts template** *(new)* — an embedded template file under `templates/agents/` that carries the rule's prose (heading, managed-by disclaimer, rule body including the archaeology and owning-workflow exceptions). Rendered against `cfg.Command` and installed into the project's `AGENTS.md`. Structurally a sibling of the existing `memory-context.md` and `spec-trigger.md` templates and treated identically by the embed FS.

- **Historical Artifacts installer** *(new)* — a Go install function in the `internal/agent` package that reads the new template from the embedded FS, renders it, and writes it into `AGENTS.md` idempotently: creates the file if missing, appends the section if the file exists but the heading is absent, and replaces the section body in place if the heading is already present. Reuses the existing atomic-write and locate/append/replace helper shape established by the two prior section installers; no new plumbing is introduced.

- **Agent Install wiring** *(changed)* — each of the three registered agents' `Install()` methods (Claude, Bob, Codex) gains a single call to the new installer, adjacent to the existing `installMemoryContextSection` / `installSpecTriggerSection` calls. The wiring order preserves existing behavior: the new call slots in as one more managed-section installer, and Claude's `ensureClaudeImportsAGENTS` continues to run last so `CLAUDE.md`'s `@AGENTS.md` import remains intact.

- **Plan-discovery step template clarifier** *(changed)* — the plan workflow's discovery step template gets a small framing tweak so its instruction to read prior plans and specs via the CLI reads as *historical archaeology in service of new-plan research*, not as current-state discovery. The change is prose-only, respects the CLI-only access rule that already exists there, and does not alter the step's structural behavior.

- **Implement-workflow read-plan step clarifier** *(changed)* — the implement workflow's first step template gets a brief acknowledgment that the plan and spec it reads are the owning workflow's own live artifact, covered by the exception stated in `AGENTS.md`. The change is prose-only; it does not alter which files are read, in what order, or through which CLI commands. The remaining implement-workflow read sites inherit this framing without individual clarifiers.

- **Historical Artifacts installer tests** *(new)* — a test file in the `internal/agent` package that mirrors the shape of the existing `memory_context_test.go` and `spec_trigger_test.go` files: create-from-missing, append-after-tessl-block, idempotent, preserves surrounding content, picks up template change, and cross-agent idempotency. Also asserts that all three agents converge on a single instance of the new section when installed in sequence.

- **Per-feature changelog entry** *(new artifact, produced at implement time)* — the entry that ships when the implement workflow completes this feature must explicitly flag the "process document, not product document" concept for future website-documentation follow-up. Authored by the existing `templates/steps/implement/10-update_feature_changelog.md` step; no changes to that step template are required — the requirement is prose content in the written entry, verified as an acceptance check on the shipped artifact.

## Data Structures & Interfaces

This feature introduces no new data structures, interfaces, or serialization boundaries. The change is instruction-surface prose in Markdown templates and a Go install function that mirrors the existing shape used by the two prior `AGENTS.md`-managed-section installers.

The single reused Go function signature is:

```go
func installHistoricalArtifactsSection(
    projectPath string,
    cfg config.Config,
    out io.Writer,
) error
```

This signature is identical to the existing `installMemoryContextSection` and `installSpecTriggerSection` functions in the `internal/agent` package and is called from the same wire-in points inside each agent's `Install()` method. It is not a new contract — it is a repetition of an established one — and it introduces no new type.

No new template variables are introduced beyond the existing `{{command}}` mustache variable already used by other agent-instruction templates. No CLI flag, JSON payload, config-file key, or workflow-state field is added or changed. The changelog entry produced at implement time is a Markdown file authored by the existing `templates/steps/implement/10-update_feature_changelog.md` step and follows the same free-prose shape as prior entries under `.spektacular/changelog/`; it introduces no schema.

## Implementation Detail

No new patterns are introduced. The implementation deliberately re-uses the shape already established by the `Memory & Context` and `Spec-Worthy Discussion Recognition` managed sections: a Markdown template under `templates/agents/` with a managed-by disclaimer blockquote, a Go install function paired with locate/append/replace helpers that treat the section as a heading-delimited range within `AGENTS.md`, and one-line wire-in in each supported agent's `Install()` method. A developer reading the resulting Go and template files should recognise them instantly as the third instance of the same triad.

The Go install function follows the exact idempotency contract of its two siblings — create-from-missing, append-after-existing-content, replace-in-place-if-heading-present — so re-running `spektacular init` any number of times converges on a single copy of the new section without accumulating whitespace, duplicating headings, or disturbing unrelated content. Existing helper functions (`writeAGENTSAtomic`, `writeFileAtomic`) are reused directly rather than re-implemented; the new installer is structurally shallow.

The section body is prose describing the rule in the same voice as the other managed sections: a plain statement of the rule, followed by the two exceptions (archaeology-on-user-intent and owning-workflow-live-artifact) stated explicitly so a reader lands on all three ideas without inference. The section carries the same managed-by disclaimer blockquote pattern the other templates use, so hand edits at the destination are visibly discouraged.

The two step-template clarifiers are prose-only edits to existing files. They do not introduce a new template variable, do not add a new step, do not change the CLI commands the steps invoke, and do not alter the step-to-step transitions the FSM drives. Their purpose is exclusively local reconciliation with the new AGENTS.md rule: the plan-discovery clarifier reframes the existing "read prior plans/specs" instruction as historical archaeology in service of new-plan research, and the implement-read-plan clarifier reframes its existing read as the owning workflow's own live artifact.

The code-structure UX for a developer maintaining this area of the codebase is a small, uniform surface: three `AGENTS.md`-managed sections, each with the same triad, tested with the same six-case shape, wired in the same order across the three agents. Adding a fourth managed section in the future — should one ever be needed — would repeat this pattern once more with no additional infrastructure required.

The per-feature changelog note that ships with the implementation is content, not code — its authoring is already handled by the existing `update_feature_changelog` step. No changes to that step template are needed; the requirement is verified as an acceptance check on the written changelog artifact.

## Dependencies

- **`internal/agent` package (existing, unchanged)** — provides the established shape the new installer mirrors: `installMemoryContextSection`, `installSpecTriggerSection`, their locate/append/replace helpers, and the shared `writeAGENTSAtomic` / `writeFileAtomic` writers. The new installer imports nothing new; it depends on these siblings only as pattern references, not as callable APIs.

- **`templates/templates.go` embed FS (existing, unchanged)** — carries the `//go:embed all:*` directive that automatically pulls the new template file into the embedded FS. No manifest edit is required; adding the file under `templates/agents/` is enough for `fs.ReadFile(sourceFS, "agents/historical-artifacts.md")` to succeed.

- **`github.com/cbroglie/mustache` (existing external, unchanged)** — renders `{{command}}` in the new template, exactly as it already renders the two prior managed-section templates. No new template variable is introduced.

- **`AGENTS.md` at the repo root of any consuming project (runtime, existing)** — the write target. The installer's idempotency contract expects an existing `AGENTS.md` may be missing, present-without-the-heading, or present-with-the-heading; each case is handled by the shared pattern. No pre-existing content in `AGENTS.md` needs to be reshaped before this feature lands.

- **`templates/steps/plan/02-discovery.md` and `templates/steps/implement/01-read_plan.md` (existing, will be edited)** — the two step templates that receive the surgical clarifiers. Both are already loaded by their respective workflows via the existing step-template loader; the edits are prose-only and change no template variables or transitions.

- **`templates/steps/implement/10-update_feature_changelog.md` (existing, unchanged)** — the step that will author this feature's own changelog entry at implement time. No changes to the step itself are required; the "process document, not product document" website-docs follow-up note is prose written into the changelog artifact by that step at implement time.

- **Prior specs and plans** — none. Spec `000036_spec_plan_historical_artifacts` is stand-alone; no upstream spec must land first, and no prior plan gates this work. Spec `000037_artifact_metadata` touches an adjacent concern (metadata on historical artifacts) but is deliberately independent; the two can ship in either order.

- **`spektacular init` re-run in consuming repos (operational)** — after this feature ships, existing repos need `spektacular init` re-run once to inject the new AGENTS.md section. This is the same operational expectation created by every prior managed-section addition; it is not a code dependency but is called out here so it is not forgotten during rollout.

## Testing Approach

The change is instruction-surface prose, so the automated test surface is bounded and mirrors what the two prior managed-section installers already carry. New tests are added at the same layer as the existing agent-package tests, using the same in-memory template-FS fixture pattern and the `stretchr/testify/require` assertions the rest of the package uses. No new test framework, harness, or fixture shape is introduced.

**Unit tests over the installer contract.** The new install function is covered by the same six-case shape that the `Memory & Context` and `Spec-Worthy Discussion Recognition` installers use: creates the section when `AGENTS.md` is missing, appends after an existing tessl-managed block when the file exists but the heading is absent, is byte-idempotent on repeated invocations, preserves surrounding sections unchanged when it replaces its own body in place, picks up template body changes on re-install, and converges on a single instance of the new section when Claude, Bob, and Codex are installed in sequence into the same directory. These are the load-bearing assertions: the installer never duplicates its own section, never corrupts unrelated content, and never disagrees across the three registered agents.

**Regression guard on the embedded template.** The new template file is added under `templates/agents/`, and the existing `instruction_surface_test.go` guard walks the embedded FS asserting no forbidden instruction-surface patterns leak in. The new template inherits that guard for free; no dedicated FS-presence test is required, because the installer test that creates the section from a missing `AGENTS.md` will fail loudly if the embed FS cannot locate the template.

**No changes to the step-template test surface.** The two prose-only clarifiers in `templates/steps/plan/02-discovery.md` and `templates/steps/implement/01-read_plan.md` do not add or remove template variables, CLI commands, or step transitions, so no additional step-template test is added. The existing repo-level guards over the templates FS already keep any regression in prose from silently reintroducing forbidden patterns.

### Success metrics coverage

The spec lists three Success Metrics; all three are runtime-behavioural observations of how agents cite sources during real user sessions, and none can be asserted from inside this Go test suite. They are classified as follows:

- **"Users stop needing to remind agents mid-session that specs and plans are not current-state documentation."** — Manual — captured in the implementation test plan.
- **"When agents describe existing features, their citations point to source code, tests, or configuration — not to files under `.spektacular/specs/` or `.spektacular/plans/`."** — Manual — captured in the implementation test plan.
- **"When users ask historical or intent questions ('why was X built this way?'), agents correctly reach for the relevant spec or plan and cite it as historical context."** — Manual — captured in the implementation test plan.

### Acceptance-check on the changelog artifact

Spec requirement #8 and acceptance criterion #9 require the per-feature changelog entry that ships with this feature to explicitly flag the "process document, not product document" concept for future website-documentation follow-up. This is verified at implement time as a content check on the written changelog artifact (present / absent), not as an automated test. It is called out here as a required acceptance so the implementer does not lose sight of it once the changelog step runs.

### Deliberate gaps

No integration or end-to-end tests are added. The unit tests over the installer already exercise the AGENTS.md write path end-to-end within a temp directory, and the cross-agent idempotency test exercises all three registered agents' real `Install()` paths against the real embedded templates FS. There is no further integration surface to cover with a separate test type; adding one would duplicate coverage without exercising a new contract.

## Milestones & Phases

### Milestone 1 — Every agent session in a Spektacular repo learns, without being asked, that specs and plans are historical artifacts

**What changes.** After this milestone, any agent starting a fresh session in a repo that has re-run `spektacular init` automatically picks up a durable, single-source-of-truth rule: specs and plans are archaeological records that explain *why* past changes were made, not current-state documentation of what the codebase does today. Agents ground current-state answers in source code, tests, and configuration; they read specs and plans only when the user is investigating past intent; and the workflow that owns a spec, plan, or implement artifact may read and update it freely while its workflow is actively running. Users no longer need to restate the rule in each session, and the rule reaches every agent context — ad-hoc questions, unrelated skills, and general exploration — not only workflow steps. `spektacular init` is idempotent for this section just like the two managed sections that already exist, so re-running it any number of times never duplicates or corrupts the rule.

**Validation point.** Running `spektacular init` in a fresh temp directory writes an `AGENTS.md` that contains exactly one instance of the new managed section, with the rule prose rendered against the configured command name. Running it again in the same directory produces byte-identical output. Running each of the three registered agents' full `Install()` paths in sequence into the same directory converges on a single instance of the section. `make test` and `make lint` pass, with the new installer's six-case unit-test set green alongside the existing two installers' test suites.

#### - [x] Phase 1.1: Author the Historical Artifacts template and its installer

Add the third `AGENTS.md`-managed section to the codebase: a Markdown template file carrying the rule prose (heading, managed-by disclaimer, rule body with archaeology and owning-workflow exceptions), a Go install function that reads the template from the embedded FS and writes it into `AGENTS.md` idempotently, and a test file that mirrors the six-case shape used by the two prior installers. The template and installer are added together because neither is meaningful without the other, and the tests exercise the installer against the real embedded template so the full write path is covered end-to-end within the phase.

*Technical detail:* [context.md#phase-11](./context.md#phase-11-author-the-historical-artifacts-template-and-its-installer)

**Acceptance criteria**:
- [x] The template file exists under `templates/agents/` and carries the same structural shape as the other two managed-section templates (heading, managed-by disclaimer blockquote, rule body).
- [x] The installer function exists in the `internal/agent` package and reuses the shared `writeAGENTSAtomic` / `writeFileAtomic` helpers rather than re-implementing them.
- [x] The six-case unit test set for the new installer (create-from-missing, append-after-tessl-block, idempotent, preserves-surrounding-content, picks-up-template-change, cross-agent-idempotency) is present and green.
- [x] `make test` and `make lint` pass.

#### - [x] Phase 1.2: Wire the installer into all three registered agents

Add a single call to the new installer inside each of the Claude, Bob, and Codex agent `Install()` methods, slotted in alongside the existing calls to `installMemoryContextSection` and `installSpecTriggerSection`. The wiring order preserves existing behaviour: the new call runs as one more managed-section installer, and Claude's `ensureClaudeImportsAGENTS` continues to run last so `CLAUDE.md`'s `@AGENTS.md` import remains intact. After this phase, running `spektacular init` in a fresh repo lands the new section automatically; re-running it converges without duplication.

*Technical detail:* [context.md#phase-12](./context.md#phase-12-wire-the-installer-into-all-three-registered-agents)

**Acceptance criteria**:
- [x] Each of the three agent `Install()` methods calls the new installer exactly once.
- [x] Running each agent's full `Install()` path in sequence against the same temp directory produces exactly one instance of the new section in `AGENTS.md`.
- [x] The existing cross-agent idempotency tests for `Memory & Context` and `Spec-Worthy Discussion Recognition` remain green (proving the new call did not disturb prior sections).
- [x] `make test` and `make lint` pass.

### Milestone 2 — The two step templates whose existing wording could look contradictory read consistently with the new rule

**What changes.** After this milestone, the plan workflow's discovery step and the implement workflow's read-plan step frame their reads of specs and plans in language that fits inside the new rule rather than appearing to break it. The plan-discovery step's instruction to read prior plans and specs is described as historical archaeology in service of new-plan research — the same activity a user does when asking "why was X built this way?" — rather than as current-state discovery. The implement-workflow read-plan step's initial read of its plan documents and spec is described as the owning workflow reading its own live artifact, which the AGENTS.md rule explicitly permits. The remaining implement-workflow read sites inherit this framing without individual clarifiers, so the maintenance surface stays small. No CLI commands, template variables, or workflow transitions are altered — the change is prose only.

**Validation point.** Both edited step templates render cleanly under the existing template loader (no mustache errors, no forbidden instruction-surface patterns introduced), and reading each template top-to-bottom the treatment of specs and plans is internally consistent with the AGENTS.md rule shipped in Milestone 1. `make test` and `make lint` pass, and the existing `TestEmbeddedTemplatesAvoidStdinInstructionSurface` guard remains green.

#### - [x] Phase 2.1: Reframe the read instructions in the two affected step templates

Update the plan workflow's discovery step so its instruction to read prior plans and specs is framed as *historical archaeology in service of new-plan research*, not as current-state discovery. Update the implement workflow's read-plan step so its initial read of the plan documents and spec is framed as the *owning workflow reading its own live artifact* — the exception the AGENTS.md rule explicitly permits. Both are prose-only edits; no CLI commands, template variables, or workflow transitions change.

*Technical detail:* [context.md#phase-21](./context.md#phase-21-reframe-the-read-instructions-in-the-two-affected-step-templates)

**Acceptance criteria**:
- [x] Reading `templates/steps/plan/02-discovery.md` top-to-bottom, the prior-plans/specs read reads as historical archaeology in service of new-plan research and does not appear to contradict the AGENTS.md rule.
- [x] Reading `templates/steps/implement/01-read_plan.md` top-to-bottom, the initial plan/spec read reads as the owning workflow's own live artifact and does not appear to contradict the AGENTS.md rule.
- [x] No new template variables, CLI commands, or workflow transitions are introduced by either edit.
- [x] The existing `TestEmbeddedTemplatesAvoidStdinInstructionSurface` guard remains green.
- [x] `make test` and `make lint` pass.

### Milestone 3 — The changelog entry that ships with this feature flags the website-documentation follow-up so it is not lost

**What changes.** When the implement workflow runs and its `update_feature_changelog` step writes this feature's per-feature changelog entry under `.spektacular/changelog/`, that entry explicitly states that the "process document, not product document" concept should be added to the user-facing website documentation in a future update. The entry follows the same shape as prior changelog entries (What was built / Why it matters / Deviations from the plan) and requires no new schema, CLI flag, or step-template change — the note is prose content authored inside the existing changelog workflow. This milestone is deliberately its own delivery so the follow-up is neither buried inside the code-changes milestone nor lost when the implement workflow closes the spec's tracker checkboxes.

**Validation point.** The written changelog artifact under `.spektacular/changelog/` for this feature contains a clearly-identifiable sentence flagging the "process document, not product document" concept for future website-documentation work, verifiable by reading the file directly.

#### - [x] Phase 3.1: Ship the website-documentation follow-up note in the feature's changelog entry

When the implement workflow reaches its `update_feature_changelog` step for this feature, the per-feature changelog entry it writes under `.spektacular/changelog/` must explicitly state that the "process document, not product document" concept should be added to the user-facing website documentation in a future update. The step template itself is unchanged; the requirement is prose content in the written entry, called out here so the note is neither buried in the code-changes milestones nor lost when the spec's checkboxes close.

*Technical detail:* [context.md#phase-31](./context.md#phase-31-ship-the-website-documentation-follow-up-note-in-the-features-changelog-entry)

**Acceptance criteria**:
- [ ] The changelog entry under `.spektacular/changelog/` for this feature contains a clearly-identifiable sentence flagging the "process document, not product document" concept for future website-documentation work.
- [ ] The sentence is placed where a reader scanning the entry will encounter it — not buried in a code block or footnote.
- [ ] The changelog entry otherwise follows the same shape as prior entries (What was built / Why it matters / Deviations from the plan).

## Open Questions

There are no impl-time-only uncertainties. Every design question this plan raises was resolvable now — by reading existing exemplar files (`internal/agent/memory_context.go`, `templates/agents/memory-context.md`), by decision (choosing Option B — AGENTS.md + surgical clarifiers — over the two rejected alternatives), or by direct enumeration of the affected step templates. The implementer has a complete instruction set: mirror the two existing installer patterns, wire into all three agents, edit two step templates for prose framing, and include one sentence in the shipped changelog entry.

## Out of Scope

- **User-facing website documentation update.** Introducing the "process document, not product document" concept into the user-facing website documentation is deliberately deferred to a follow-up. This plan only ensures the concept is flagged in the shipped changelog entry so the follow-up is not lost when the spec closes. Tracked via the changelog note authored in Phase 3.1.

- **Filesystem-level or tool-level access blocks on `.spektacular/specs/` and `.spektacular/plans/`.** Rejected during architecture: hard access blocks would break the archaeology exception, which requires that agents remain able to read historical specs and plans when a user is investigating past intent. Enforcement stays at the agent-instruction level, per the spec's Constraint.

- **Retroactive markers on the ~66 historical specs and plans.** No metadata, marker, or annotation is added to the artifacts already in `.spektacular/specs/` or `.spektacular/plans/`. The archaeology framing works on the current file layout because agents distinguish current-state from historical questions via user intent, not via per-file markers. Related work on artifact metadata is tracked separately in spec `000037_artifact_metadata`; the two ship independently and do not gate each other.

- **Skill-level notes in each `spek-*/SKILL.md`.** Rejected during architecture as duplicative once `AGENTS.md` is session-wide-loaded. The skill files describe workflow orchestration; the rule they need to obey is already stated globally in `AGENTS.md`.

- **Changes to the remaining implement-workflow read sites.** Only `templates/steps/implement/01-read_plan.md` receives a framing clarifier. The other implement-workflow steps that read specs or plans (`02-analyze`, `06-update_plan`, `07-update_changelog`, `09-test_plan`, `10-update_feature_changelog`, `11-reconcile_spec`, `12-finished`) are deliberately left untouched — they inherit the owning-workflow framing from the first step and from `AGENTS.md`, and per-step clarifiers would add maintenance surface without extending coverage.

- **A new "who owns this artifact right now" runtime signal.** No code-level ownership tracking, workflow-state field, or CLI flag is introduced to enforce the owning-workflow exception. The exception is behaviourally self-evident because only the owning workflow reaches for `spec file` / `plan file` CLI commands during its run; that is enough for the prose enforcement model chosen here.

## Changelog

### 2026-07-17 — Phase 1.1: Author the Historical Artifacts template and its installer

**What was done**: Added a third `AGENTS.md`-managed section covering "specs and plans are historical archaeology, not current-state docs" with the archaeology-on-user-intent and owning-workflow-live-artifact exceptions. New Markdown template at `templates/agents/historical-artifacts.md`, new Go installer at `internal/agent/historical_artifacts.go` mirroring `internal/agent/memory_context.go` case-for-case (constants + install + locate/append/replace helpers, all reusing the shared `writeAGENTSAtomic` and `writeFileAtomic` writers), and new test file at `internal/agent/historical_artifacts_test.go` covering the same six cases as the two prior installers.

**Deviations**: None from the phase's own scope. See Phase 1.2's entry for the co-landing note.

**Files changed**:

- `templates/agents/historical-artifacts.md` (new)
- `internal/agent/historical_artifacts.go` (new)
- `internal/agent/historical_artifacts_test.go` (new)

**Discoveries**: The six-case shape borrowed from `memory_context_test.go` includes a cross-agent idempotency test that exercises all three registered agents' real `Install()` paths. That test cannot go green until the installer is wired into `claude.go`, `bob.go`, and `codex.go` — i.e. until Phase 1.2. Since Phase 1.1's own acceptance criterion requires "the six-case unit test set is present and **green**", Phase 1.1 cannot honestly close without Phase 1.2 landing in the same iteration. See Phase 1.2's entry for how this was resolved.

### 2026-07-17 — Phase 1.2: Wire the installer into all three registered agents

**What was done**: Added a single call to `installHistoricalArtifactsSection` inside each of the Claude, Bob, and Codex agent `Install()` methods, slotted in after the existing `installSpecTriggerSection` call. For Claude, the new call sits before `ensureClaudeImportsAGENTS` so `CLAUDE.md`'s `@AGENTS.md` import continues to run last. For Bob and Codex, the terminal `return installSpecTriggerSection(...)` was reshaped into a guarded call followed by `return installHistoricalArtifactsSection(...)`, matching Claude's existing per-installer-guarded pattern.

**Deviations**: Landed together with Phase 1.1 in a single implement iteration under Auto Mode. Phase 1.1's cross-agent idempotency test requires the wire-in to be present to go green, so splitting the two phases into separate iterations would leave Phase 1.1's own acceptance criterion visibly unmet at close. Both checkboxes were ticked in the same `update_plan` write.

**Files changed**:

- `internal/agent/claude.go`
- `internal/agent/bob.go`
- `internal/agent/codex.go`

**Discoveries**: None. `make test` and `make lint` both green after the co-landing; all six new tests pass alongside every existing agent-package test.

### 2026-07-17 — Phase 2.1: Reframe the read instructions in the two affected step templates

**What was done**: Prose-only reframing of two existing step-template read instructions so they read consistently with the new `## Historical Artifacts` rule shipped by Phase 1.1. In `templates/steps/plan/02-discovery.md`, the "Prior research" bullet under Step 2 now opens with an explicit framing sentence: prior plans and specs are historical archaeology explaining *why* past decisions were made, not descriptions of what the codebase does today, and are consulted here to inform the new plan — not to derive current-state behaviour. In `templates/steps/implement/01-read_plan.md`, the Step-1 read instruction now leads with a one-sentence acknowledgment that the plan documents are the implement workflow's own live artifact — the owning-workflow exception the AGENTS.md rule explicitly permits — before the existing "read them through the plan store" instruction.

**Deviations**: None.

**Files changed**:

- `templates/steps/plan/02-discovery.md`
- `templates/steps/implement/01-read_plan.md`

**Discoveries**: The `TestEmbeddedTemplatesAvoidStdinInstructionSurface` guard in `internal/agent/instruction_surface_test.go:33-48` walks `templates/steps/` and rejects a fixed set of forbidden stdin/heredoc CLI patterns; both new sentences respect that surface (no new CLI patterns, no stdin references), so the guard stays green with no test authoring. No template variables, CLI commands, or workflow transitions were added or removed by either edit — the changes are purely narrative.

### 2026-07-17 — Phase 3.1: Ship the website-documentation follow-up note in the feature's changelog entry

**What was done**: Zero code, template, or configuration changes were made in this phase — by design. Phase 3.1 exists to make it an explicit, tracked acceptance requirement that the per-feature changelog entry authored by the later `update_feature_changelog` step (step 10 of the implement workflow) contains a clearly-identifiable sentence flagging the "process document, not product document" concept for future website-documentation work. The phase's actual output ships when `update_feature_changelog` writes the entry under `.spektacular/changelog/`; the sentence will be placed in the "What was built" or "Why it matters" section — somewhere a reader scanning the entry lands on it — and verified as content on the shipped artifact.

**Deviations**: The phase's three acceptance criteria all reference the shipped changelog artifact, which does not yet exist at this update_plan invocation. Only the phase's own `#### - [ ] Phase 3.1:` heading was ticked to `- [x]`; the three acceptance-criterion checkboxes were left `- [ ]` and will be ticked by hand once `update_feature_changelog` writes the entry with the note in place. This preserves the "only tick if verified" invariant while still advancing the workflow past the phase-loop counter, which counts `#### - [ ] Phase` headings (not per-AC checkboxes).

**Files changed**: None in this phase. The actual file that will change — `.spektacular/changelog/<n>_<name>.md` — is authored later by `update_feature_changelog`.

**Discoveries**: The reconciliation step (`reconcile_spec`, step 11) reads the plan's accumulated Changelog record and matches against spec-level requirements/ACs, so the spec's own AC #9 ("Changelog entry contains the documentation follow-up note") will be ticked based on the entry produced by `update_feature_changelog`, not based on Phase 3.1's per-AC checkboxes. Leaving those unchecked here does not block spec reconciliation.
