# Plan: 000023_context-over-memory

<!-- Metadata -->
<!-- Created: 2026-05-27T15:31:49Z -->
<!-- Commit: 64bba575e939e3cd177e4cb355e8fc11207d7681 -->
<!-- Branch: f-context-not-memory -->
<!-- Repository: git@github.com:jumppad-labs/spektacular.git -->

## Overview

Add a single project-level instruction to this repository so that any supported coding agent (Claude, Codex, Bob) redirects what it would normally save into its per-user, per-machine memory into the project's shared knowledge store instead. Today, learned conventions, feedback, and project facts get trapped on one developer's machine, so collaborators never benefit from them. After this lands, captured knowledge becomes shared, reviewable in git, and travels with the project — without any agent runtime change.

The instruction is installed and maintained by `spektacular init`. Re-running init for the same agent, or initialising a different supported agent in the same repo, updates the managed section in place — it never appends a duplicate. The section is delimited in `AGENTS.md` by its `## Memory & Context` markdown heading, so detection works without injecting machine-only markers into a human-readable file.

## Architecture & Design Decisions

The redirect lives as a single new "Memory & Context" section in `AGENTS.md`. The section is owned by `spektacular init` — it is written from an embedded template, not hand-edited — and is identified inside `AGENTS.md` by its `## Memory & Context` markdown heading. On every init, the writer locates that heading and replaces the span from the heading line up to (but not including) the next `^# ` / `^## ` heading or EOF; if no such heading is present, the section is appended to the end of the file (or the file is created if absent). This is the idempotency contract the spec requires.

This checkout currently uses tessl, which manages the `# Agent Rules <!-- tessl-managed -->` block at the top of `AGENTS.md`. The managed Memory & Context section sits at a different heading level (`##`) under its own H2 heading and is located by exact heading text, so tessl's block and the Spektacular-managed section coexist without collision regardless of which is added first. If tessl is not in use, the same writer still produces the right outcome — it simply finds no tessl block, locates (or creates) `## Memory & Context`, and writes its content.

All three supported agents already load `AGENTS.md` at session start — Claude inherits it through the existing `@AGENTS.md` import in `CLAUDE.md`; Codex loads it natively from the repository root; Bob loads it natively. One file, one section, three agents.

The writer is a single shared helper in `internal/agent/` invoked from each agent's `Install()` method (alongside `installWorkflowSkills` / `installCommandWrappers`). The action is identical for every agent, but routing it through `Install()` rather than `project.Init` keeps the agent surface — skills, command wrappers, AGENTS.md section — owned by the per-agent path and consistent with how Spektacular already factors per-agent installation. Because the operation is idempotent, calling it from each agent's `Install()` is safe even when a project later switches agents or re-runs init.

The destination for redirected writes is the project knowledge store at `.spektacular/knowledge/`, accessed through the `spek-knowledge` skill (or `{{command}} knowledge write` directly, where `{{command}}` is the per-project CLI invocation already used by the skill templates). That destination, its CLI surface, and its propose-then-confirm guard already exist (shipped under specs 000020 and 000022) — this plan only points agents at them.

The redirect text is intentionally plain prose with no agent-specific syntax, so a single block serves all three agents without duplication. It carries only what `AGENTS.md` uniquely needs: the trigger (any persistence the agent would otherwise route into its per-user memory) and the destination (the `spek-knowledge` skill), plus explicit in-repo scoping so behavior in other repositories is unaffected, and a short note that the section is managed by `spektacular init`. All of the operational mechanics — propose-then-confirm, search-first, scope/path discovery, scratch-file staging — already live in the `spek-knowledge` skill (shipped under spec 000022) and are deliberately *not* duplicated here. The section's only job is the redirect; the skill carries the how.

Rejected alternatives — including a new shared instruction file referenced from each agent's entrypoint, inlining the same text in `CLAUDE.md` / `AGENTS.md` / `.bob/rules/`, placing the text in `.tessl/RULES.md` (which `tessl install` overwrites), HTML-comment delimiters around the managed section, and a project-level (rather than per-agent) install path — are documented with citations in [research.md § Alternatives considered and rejected](./research.md#alternatives-considered-and-rejected).

## Component Breakdown

- **`AGENTS.md` (managed)** — Project-level instruction file already loaded by Claude (via `@AGENTS.md` in `CLAUDE.md`), Codex (natively at repo root), and Bob (natively). Gains (or, on re-init, has updated in place) a `## Memory & Context` section written from the embedded template. Not hand-edited any more.
- **`templates/agents/memory-context.md` (new)** — Embedded markdown template for the managed section, rendered through `mustache` against `cfg.Command` (matching the existing skill templates that use the `{{command}}` placeholder). Source of truth for the section's body.
- **`internal/agent/memory_context.go` (new)** — Shared helper that reads `AGENTS.md` if present, locates the `## Memory & Context` heading, replaces from that heading up to the next `^# ` / `^## ` heading or EOF, or appends if absent, or creates the file if missing. Idempotent. Exposed as a single function (e.g. `installMemoryContextSection(projectPath, cfg, out)`) that each agent's `Install()` calls.
- **`internal/agent/claude.go`, `codex.go`, `bob.go` (changed)** — Each `Install()` method gains a call to `installMemoryContextSection(...)` alongside the existing `installWorkflowSkills` / `installCommandWrappers` calls. No other change to the per-agent files.
- **`internal/agent/memory_context_test.go` (new)** — Unit tests for the idempotency contract: first-run create, first-run append into an existing file, second-run update-in-place with no duplication, preservation of a synthetic tessl-managed top block, and the create-if-missing path.
- **Project knowledge store (unchanged, referenced)** — The `.spektacular/knowledge/` store shipped under spec 000020 and exposed to agents through the `spek-knowledge` skill shipped under spec 000022. This plan does not modify it; it only directs agents to write into it.
- **tessl-managed `AGENTS.md` block (untouched when present)** — If tessl is in use in this checkout, the `# Agent Rules <!-- tessl-managed -->` block continues to be owned by `tessl install`. The Spektacular-managed section is located by its own H2 heading and never overlaps with the tessl-managed H1 block, so `tessl install`'s regeneration cycle is unaffected.

## Data Structures & Interfaces

One new internal function in `internal/agent/`:

```go
// installMemoryContextSection writes (or updates in place) the managed
// "Memory & Context" section in <projectPath>/AGENTS.md, rendering the
// embedded template against cfg.Command. Idempotent: re-running for the
// same projectPath leaves a single section and does not duplicate.
func installMemoryContextSection(projectPath string, cfg config.Config, out io.Writer) error
```

No exported types or new package boundaries. The existing `Agent` interface is unchanged. The behavior the plan delivers is still mediated by existing CLI surfaces (`{{command}} knowledge {search,read,write}`) and the existing `spek-knowledge` skill; their contracts are unchanged.

## Implementation Detail

**Section template (`templates/agents/memory-context.md`)** — an embedded markdown file holding the body of the managed section. It does not currently use any mustache placeholders, but it is still rendered through `mustache.Render` against `map[string]string{"command": cfg.Command}` to keep the install path uniform with `installWorkflowSkills` and so future edits can add `{{command}}` references without touching the writer. The template begins at the `## Memory & Context` heading and ends with a trailing newline; it carries (1) a one-line note that the section is managed by `spektacular init`, (2) the trigger phrase, (3) the destination (the `spek-knowledge` skill), and (4) the in-repo scoping. It deliberately does *not* carry propose-then-confirm wording, search-first wording, or any CLI invocations — those live in the skill, which is the single source of truth for how to write into the knowledge store. Plain prose, no agent-specific syntax.

**Find-and-replace algorithm (`internal/agent/memory_context.go`)**:

1. Read `<projectPath>/AGENTS.md`. If the file does not exist, write a new file whose only content is the rendered section (with a single trailing newline) and return.
2. Split the file into lines. Locate the first line that, after trimming trailing whitespace, equals `## Memory & Context`.
3. If found, locate the next line at or after that index whose trimmed form begins with `# ` or `## ` (any other H1/H2 heading), or use EOF if there is none. Replace the half-open span `[heading, next_heading)` with the rendered section's lines. Preserve a single blank line of separation before and after the section if the surrounding content does not already supply one.
4. If not found, ensure the existing file ends with a single blank line, then append the rendered section.
5. Write the result back atomically (write to `AGENTS.md.tmp`, then rename) so a crash mid-write cannot corrupt an existing tessl-managed block.
6. Emit one line to `out` describing the action (`"  AGENTS.md: created"`, `"  AGENTS.md: updated Memory & Context section"`, or `"  AGENTS.md: appended Memory & Context section"`), matching the existing one-line-per-artefact convention in `installWorkflowSkills`.

**Wiring (`internal/agent/{claude,codex,bob}.go`)** — each `Install()` method gains a single call to `installMemoryContextSection(projectPath, cfg, out)` after its existing skill/command-wrapper calls. The call order is not load-bearing because the operation is idempotent, but placing it last matches the "skills first, then shared instructions" reading order.

**Why each load-bearing element of the section body is present**:

- The trigger phrase ("when you would persist something to your per-user, per-machine memory store") is intentionally agent-neutral — it matches the framing Claude's auto-memory system uses and is general enough for Codex's and Bob's equivalents.
- The destination names the skill (`spek-knowledge`); the skill itself owns the propose-then-confirm guard, scope/path discovery, and the actual `{{command}} knowledge write` invocation, so naming any of those here would duplicate the skill.
- The in-repo scoping satisfies the spec's "Scoped to this repo" acceptance criterion.
- The "managed by `spektacular init`" note tells humans why hand-edits will not survive.

No agent-specific syntax is used. Plain prose only, so the same text works identically for Claude, Codex, and Bob without branching or per-agent variants.

## Dependencies

- **Spec 000020 (context)** — already shipped. Provides the project knowledge store at `.spektacular/knowledge/` that this redirect targets. No changes required.
- **Spec 000022 (spek-knowledge skill)** — already shipped. Provides the agent-facing skill and `go run . knowledge {search,read,write}` CLI used as the redirect destination, including the propose-then-confirm guard. No changes required.
- **`tessl install`** — must continue to own only its `<!-- tessl-managed -->` block of `AGENTS.md`. No change requested; the Spektacular-managed section is located by its own H2 heading and does not overlap with tessl's H1 block.

## Testing Approach

Two distinct testable surfaces, with different test strategies:

**1. Idempotent install (unit-testable).** The `installMemoryContextSection` function is fully testable with table-driven Go tests against a `t.TempDir()`. The test set must cover, at minimum:

- **Create from missing.** `AGENTS.md` absent; after one call, file exists and contains exactly the rendered section.
- **First-time append.** `AGENTS.md` exists with unrelated content (e.g. a simulated tessl block); after one call, the unrelated content is byte-for-byte unchanged and the section is appended below it.
- **Update in place (idempotency).** Call twice on the same `AGENTS.md`; after the second call, the file is byte-for-byte identical to the result of the first call. Exactly one `## Memory & Context` heading is present.
- **Update preserves surrounding content.** Pre-seed `AGENTS.md` with a tessl-style block above the section and an unrelated H2 below it; after re-running, content above and below the section is unchanged and the section is updated in place.
- **Template change is picked up.** Re-render with a different `cfg.Command` value and confirm the section body reflects the new value.
- **Cross-agent idempotency.** Simulating "init claude" then "init codex" then "init bob" against the same temp directory leaves exactly one section. (This guards the spec's "Idempotent install" acceptance criterion directly.)

Tests live in `internal/agent/memory_context_test.go` and follow the patterns in the existing `agent_test.go` / `claude_test.go` / `bob_test.go` / `codex_test.go`. Per memory `feedback_tests_own_their_filesystem`, all filesystem interaction is rooted at `t.TempDir()` and never reads beyond the test's scratch dir. Per memory `feedback_tests_need_independent_oracle`, expected file contents are hand-maintained (a `wantAgentsMD := "<verbatim string>"`) rather than derived at runtime from the production renderer.

**2. Agent adherence (not unit-testable).** Whether a given agent actually honours the redirect at session start is best-effort, per spec § Non-Goals ("Enforcing adherence. We are not adding any verification, hook, or guardrail that detects an agent writing to per-user memory"). This stays a manual sanity check, recorded in the PR description rather than committed:

- **Tessl-safe edit (when applicable).** If tessl is in use in this checkout, run `tessl install` after `spektacular init` and confirm the managed Memory & Context section is preserved while the tessl-managed block round-trips cleanly.
- **Cross-agent discovery smoke test.** Start a fresh session in this repo with each of Claude, Codex, and Bob and ask a trivial "remember this" prompt; observe that the agent proposes a `spek-knowledge` write instead of a per-user memory write.
- **Out-of-repo control.** Start a session in any unrelated repo with the same agents and confirm they still write to per-user memory.

## Milestones & Phases

### Milestone 1: Agent-written learnings land in the project knowledge store

**What changes**: When a developer works in this repo with any of the supported agents (Claude, Codex, Bob), and the agent identifies something worth persisting (a learning, convention, gotcha, project fact, or user preference), the agent proposes a write to the project's shared knowledge store via the `spek-knowledge` skill instead of writing silently to its per-user memory store. The user reviews the proposed entry, picks the scope, and confirms before anything is persisted. The captured knowledge is then visible to every collaborator on the repo through git, and stays absent from any one developer's per-user memory. Outside this repo, the same agents continue using per-user memory as normal. `spektacular init` is the single owner of the Memory & Context section — re-running init (for the same agent or a different one) updates the section in place without duplicating it.

#### - [x] Phase 1.1: Add the embedded section template

Add `templates/agents/memory-context.md` containing the body of the managed section. The body carries only what `AGENTS.md` uniquely needs: the trigger phrase, the destination (route through the `spek-knowledge` skill), the in-repo scoping, and a one-line note that the section is managed by `spektacular init`. Propose-then-confirm, search-first, scope/path discovery, and CLI invocations are deliberately left to the skill — naming them here would duplicate the skill's prose. The template is still routed through `mustache.Render` (against `cfg.Command`) on install for consistency with the skill templates, even though the current body has no placeholders.

*Technical detail:* [context.md#phase-11](./context.md#phase-11-add-the-embedded-section-template)

**Acceptance criteria**:

- [x] `templates/agents/memory-context.md` exists, begins with a `## Memory & Context` heading, and ends with a single trailing newline.
- [x] The file is reachable through `templates.FS` (it is automatically because of `//go:embed all:*`).
- [x] The body covers exactly four elements — the trigger, the destination (route through the `spek-knowledge` skill), the in-repo scoping, and the "managed by `spektacular init`" note — and does *not* duplicate the skill's propose-then-confirm or search-first wording.
- [x] The body contains no hard-coded CLI invocations (e.g. no inline `{{command}} knowledge ...` references); operational mechanics are the skill's responsibility.

#### - [x] Phase 1.2: Implement the idempotent AGENTS.md writer

Add `internal/agent/memory_context.go` with `installMemoryContextSection(projectPath, cfg, out)`. Implement the find-and-replace algorithm described in *Implementation Detail*: locate `## Memory & Context` by exact heading text, replace from that line up to the next `# `/`## ` heading or EOF, append if absent, create the file if missing. Write atomically via temp file + rename. Emit a single line per action to `out`, matching the format used by `installWorkflowSkills`.

*Technical detail:* [context.md#phase-12](./context.md#phase-12-implement-the-idempotent-agentsmd-writer)

**Acceptance criteria**:

- [x] `installMemoryContextSection` exists in `internal/agent/memory_context.go` with the signature documented in *Data Structures & Interfaces*.
- [x] First invocation against an empty `t.TempDir()` creates `AGENTS.md` with exactly the rendered section.
- [x] Invocation against an `AGENTS.md` containing only a synthetic tessl-managed block leaves that block byte-for-byte unchanged and appends the section after it.
- [x] A second invocation on the result of any of the above produces a byte-for-byte identical file (idempotency).
- [x] Pre-seeding the file with the section and additional unrelated content above and below it leaves all unrelated content byte-for-byte unchanged after re-invocation; the section itself is updated to match the current template.
- [x] Re-running with a different `cfg.Command` rewrites the in-section command references to match.
- [x] Writes go through a temp file + rename so a mid-write failure cannot truncate an existing file.

#### - [x] Phase 1.3: Wire the writer into each agent's Install

Each of `internal/agent/claude.go`, `codex.go`, and `bob.go` gains a single call to `installMemoryContextSection(projectPath, cfg, out)` inside `Install()`, after its existing skill/command-wrapper calls. No other change to those files.

*Technical detail:* [context.md#phase-13](./context.md#phase-13-wire-the-writer-into-each-agents-install)

**Acceptance criteria**:

- [x] `claudeAgent.Install`, `codexAgent.Install`, and `bobAgent.Install` all call `installMemoryContextSection` exactly once.
- [x] Running `spektacular init claude` then `spektacular init codex` then `spektacular init bob` against the same temp project directory leaves exactly one `## Memory & Context` section in `AGENTS.md`, and all three agents' skills are installed alongside it.
- [x] Re-running `spektacular init <agent>` for any agent against an already-initialised project leaves `AGENTS.md` byte-for-byte unchanged when the template has not changed.
- [ ] In a fresh session inside this repo, each supported agent proposes a write to the project knowledge store rather than its per-user memory store at the first persistence opportunity.
- [ ] In a fresh session in a different repository, the same agents continue to write to their per-user memory store as normal.

#### - [x] Phase 1.4: Add the idempotency unit tests

Add `internal/agent/memory_context_test.go` covering the cases enumerated in *Testing Approach* §1. Per memory `feedback_tests_own_their_filesystem`, all I/O is rooted in `t.TempDir()`. Per memory `feedback_tests_need_independent_oracle`, expected file contents are hand-maintained string constants in the test, not derived from the production renderer.

*Technical detail:* [context.md#phase-14](./context.md#phase-14-add-the-idempotency-unit-tests)

**Acceptance criteria**:

- [x] Tests cover: create-from-missing, append-after-tessl-block, update-in-place idempotency, preserve-surrounding-content, template-change-picked-up, cross-agent idempotency (claude→codex→bob).
- [x] `go test ./internal/agent/...` passes.
- [x] Tests do not read or write outside `t.TempDir()`.

## Open Questions

- **Does Bob load `AGENTS.md` natively?** — The user is "pretty sure" yes, and the plan assumes it. This can only be verified by starting a Bob session in this repo after init and observing whether Bob picks up the new section. **What the implementer should do when they hit it**: after `spektacular init bob` writes the section, run a smoke session in Bob and check that the redirect text is visible to it. If Bob does not appear to load `AGENTS.md`, STOP and ask the user before deciding between (a) adding a one-line pointer in `.bob/rules/rules.md` from `bobAgent.Install`, or (b) inlining the redirect text in a `.bob/rules/` file (and accepting the small duplication).

## Out of Scope

- **Migrating existing per-user memory entries into the shared knowledge store.** Only persistence triggered after the redirect lands is affected. Existing per-developer memory stays where it is. (Source: spec § Non-Goals.)
- **Building or modifying the shared knowledge store itself.** The store and its CLI already exist (spec 000020 shipped) and the agent-facing wrapper already exists (spec 000022 shipped). This plan only adds the instruction that points agents at them. (Source: spec § Non-Goals.)
- **Enforcing adherence.** No hook, lint, pre-commit check, or runtime guardrail that detects an agent writing to per-user memory is added. Adherence is best-effort per agent. (Source: spec § Non-Goals.)
- **Defining a canonical destination across team members.** Each collaborator may have a different per-user memory store configured by their own agent; the redirect targets the in-repo knowledge store, which is the same for everyone, but the per-user side is left alone. (Source: spec § Non-Goals.)
- **Managing `CLAUDE.md` directly.** Claude already inherits `AGENTS.md` via the existing `@AGENTS.md` import; the writer touches only `AGENTS.md`. If that import is ever removed by hand, this plan does not restore it.
- **Hand-edited Memory & Context section content.** The section is managed by `spektacular init`; hand-edits are overwritten on the next init. Users who want different wording change `templates/agents/memory-context.md` in the Spektacular source, not the rendered output.
- **A new dedicated instruction file** (e.g. `.agents/redirect-memory.md`). Rejected at the architecture step in favor of managing `AGENTS.md`, which all three supported agents already load. (See [research.md § Alternatives considered and rejected](./research.md#alternatives-considered-and-rejected).)
- **HTML-comment delimiters around the managed section.** Rejected in favor of heading-based detection so `AGENTS.md` stays clean human-readable markdown. The trade-off — that hand-edits between the heading and the next H2 are silently overwritten — is documented in the section body itself.
- **Per-agent duplication of the redirect text.** Rejected at the architecture step in favor of a single canonical section in `AGENTS.md`. If the Bob assumption in Open Questions fails at implementation time, the implementer must STOP and consult the user rather than silently introducing duplication.

## Changelog

### 2026-05-27 — Phase 1.1: Add the embedded section template

**What was done**: Created `templates/agents/memory-context.md` containing the body of the managed Memory & Context section — the trigger phrase, the destination (route through the `spek-knowledge` skill), the in-repo scoping, and the "managed by `spektacular init`" callout. The file is automatically embedded via the existing `//go:embed all:*` directive in `templates/templates.go`.

**Deviations**: None on substance. Note: the plan/context refer to `templates/embed.go` as the file holding the embed directive; the actual file is `templates/templates.go`. The `//go:embed all:*` directive and the `templates.FS` export are unchanged, so the implementation strategy holds verbatim — this is documentation drift only.

**Files changed**:
- `templates/agents/memory-context.md` (new)

**Discoveries**:
- The embed directive lives in `templates/templates.go`, not `templates/embed.go` as the plan/context reference. Future phases that need to reference where files become available through `templates.FS` should point at `templates/templates.go`.
- The new file is reachable via `fs.ReadFile(templates.FS, "agents/memory-context.md")` — verified with a one-off harness program. Phase 1.2 should use that same path key when reading the template through the package-level `sourceFS`.

### 2026-05-27 — Phase 1.2: Implement the idempotent AGENTS.md writer

**What was done**: Added `internal/agent/memory_context.go` exposing `installMemoryContextSection(projectPath, cfg, out)` with the signature documented in *Data Structures & Interfaces*. The implementation reads the embedded template through the package-level `sourceFS`, renders it with `mustache` against `cfg.Command` (matching `installWorkflowSkills`), then runs the find-and-replace algorithm from the plan: locate `## Memory & Context` by exact-match heading text, replace from that line up to the next H1/H2 heading or EOF, append after existing content if absent, or create the file if missing. Writes go through a sibling `AGENTS.md.tmp` + `os.Rename` so a mid-write failure cannot truncate the existing file. Emits one line per action to `out`, matching the leading-spaces convention used by `installWorkflowSkills`.

**Deviations**:
- Phase 1.2's behavioural acceptance criteria (idempotency, append-after-tessl-block, etc.) are *covered* by the unit tests scheduled for Phase 1.4. They are intentionally left unchecked on Phase 1.2's criteria list in this update and will be flipped at the end of Phase 1.4 once `go test ./internal/agent/...` exercises them. Only the signature criterion is checked off now; the phase heading is checked off because the code deliverable is complete.
- Blank-line normalisation happens at the join boundaries inside `replaceMemoryContextSection` / `appendMemoryContextSection` (trim-then-re-add) rather than as a separate "preserve a single blank line of separation" pre-pass. The behaviour is the same; the implementation just folds the rule into the join, which keeps idempotency trivially provable.

**Files changed**:
- `internal/agent/memory_context.go` (new)

**Discoveries**:
- The package-level `sourceFS` defined in `internal/agent/skills.go` (`var sourceFS fs.FS = templates.FS`) is the right hook for both the production embed and the fixture-FS substitution Phase 1.4 will use. No new package-level variable is required.
- `installWorkflowSkills` formats its per-artefact log line as `"  Skill:    %s\n"` (two leading spaces, label, padding for column alignment). I mirrored the leading-spaces convention with `"  AGENTS.md: %s %s\n"`. Phase 1.4 tests that inspect `out` should match against this exact format.
- Only H1 (`# foo`) and H2 (`## foo`) terminate the managed span — H3+ (`### foo`) is *not* a terminator. The plan calls this out by listing only `# ` and `## `; the implementation enforces it via `strings.HasPrefix` against those exact two prefixes.

### 2026-05-27 — Phase 1.3: Wire the writer into each agent's Install

**What was done**: Each of `internal/agent/claude.go`, `codex.go`, and `bob.go` now calls `installMemoryContextSection(projectPath, cfg, out)` from inside `Install()`, after its existing skill (and, for Bob, command-wrapper) installs. Each `Install` returns early on error from any prior step so the new call only runs when its prerequisites succeeded; the AGENTS.md write itself is idempotent, so running init for one agent and then another against the same project leaves a single section.

**Deviations**:
- The behavioural acceptance criteria for this phase (cross-agent idempotency, no-op re-init, in-repo vs out-of-repo agent behaviour) are scheduled to be verified by the cross-agent-idempotency unit test in Phase 1.4 and by the manual smoke checks recorded in the PR description, per *Testing Approach §2*. They are intentionally left unchecked in plan.md and will be flipped at the end of Phase 1.4. Only the "exactly one call per agent" criterion is checked off now; the phase heading is checked off because the wiring change is complete.

**Files changed**:
- `internal/agent/claude.go`
- `internal/agent/codex.go`
- `internal/agent/bob.go`

**Discoveries**:
- All three agents' existing `Install` test suites (`claude_test.go`, `codex_test.go`, `bob_test.go`, `agent_test.go`, `instruction_surface_test.go`) still pass after the new call is wired in. Those tests treat `Install` as opaque (they assert on installed skill files, not on AGENTS.md), so they did not need to be updated — the new test surface lands in `memory_context_test.go` (Phase 1.4) and the existing tests remain authoritative for the skills/commands part of `Install`.
- Bob's `Install` now has three sequential steps (skills → commands → AGENTS.md). The pattern (`if err != nil { return err }` after each) is preserved for readability; no helper was introduced to chain them, because the surface is small and the explicit form keeps each integration point easy to find when adding a future step.

### 2026-05-27 — Phase 1.4: Add the idempotency unit tests

**What was done**: Added `internal/agent/memory_context_test.go` with six tests covering the contract enumerated in *Testing Approach* §1: create-from-missing, append-after-tessl-block, byte-equality idempotency, preserve-surrounding-content (a tessl-style block above and an unrelated `## Other Section` below), template-change-picked-up (re-rendering with a different `cfg.Command` rewrites the in-section command references), and cross-agent idempotency (`claude` → `codex` → `bob` against one temp project leaves exactly one `## Memory & Context` heading and all three agents' skill trees in place). The first five tests use a fixture `fstest.MapFS` substituted via the existing `withSourceFS` helper so the expected file bodies are tiny hand-maintained string constants per memory `feedback_tests_need_independent_oracle`. The cross-agent test exercises the real `templates.FS` because it needs the full `Install` paths to run end to end. Flipping these tests on also lets us check off the deferred behavioural acceptance criteria for Phase 1.2 (idempotency, append, template-change pickup, atomic write) and the verifiable cross-agent criteria for Phase 1.3 (single-section after three agents, no-op re-init).

**Deviations**: None on substance. The two remaining unchecked criteria on Phase 1.3 ("agent proposes a write to the project knowledge store" / "out-of-repo control") are *not* unit-testable per plan.md § Testing Approach §2 — they are the live-agent smoke checks recorded in the PR description rather than in `*_test.go`.

**Files changed**:
- `internal/agent/memory_context_test.go` (new)

**Discoveries**:
- `withSourceFS` in `agent_test.go` is the established hook for swapping the package-level `sourceFS` for the duration of a test (it saves+restores via `t.Cleanup`). Reusing it gave us a small fixture FS for the unit tests and a real-templates path for the cross-agent test in the same file, with no new test helper required.
- The cross-agent test must use `cfg := config.NewDefault()` so `cfg.Command` is populated; an empty `config.Config{}` would render an empty `{{command}}` placeholder. The default value (`spektacular`) is harmless because the template body has no command references, but the broader skills install paths under the same `Install` calls do need a non-empty command to render correctly.
- `strings.Count(body, "## Memory & Context")` is the cleanest single-section assertion because the template body uses "memory store" / "per-user memory" in prose — only the H2 heading is the exact `## Memory & Context` literal, so the count is unambiguous.

