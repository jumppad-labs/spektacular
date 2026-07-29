# Plan: 000036_present-drafts-as-text-not-dialog

<!-- Metadata -->
<!-- Created: 2026-07-15T07:02:15Z -->
<!-- Commit: e401357 -->
<!-- Branch: main -->
<!-- Repository: git@github.com:jumppad-labs/spektacular.git -->

## Overview

Today, nothing tells Spektacular's assistant surfaces *how* to present a drafted piece of content — an architecture write-up, a set of options, a plan section — before asking the user to confirm it, so the assistant sometimes embeds the draft itself inside a structured yes/no or multiple-choice dialog, which truncates or compresses long text and makes it hard to read. This plan adds a third managed AGENTS.md section, alongside the existing "Memory & Context" and "Spec-Worthy Discussion Recognition" sections, that instructs every agent surface (Claude, Codex, Bob) to always show a draft as plain, readable chat text first and ask any structured confirmation only afterward, as a short, separate question. Because the two existing sections are near-duplicate implementations, this plan first consolidates them into one shared, generic installer before adding the new section as a third caller of it — landing the refactor and the new instruction as two independent, revertible milestones. Users of every Spektacular-managed project benefit from being able to read the assistant's full drafts clearly, regardless of which workflow step produced them.

## Conventions

No project conventions apply to this feature. `go run . knowledge always-applied` returned only the conventions and glossary category README entries (no populated convention entries currently exist in the knowledge base to select from).

## Architecture & Design Decisions

The solution has two parts. First, consolidate `internal/agent/memory_context.go` and `internal/agent/spec_trigger.go` — currently ~120 lines of near-byte-identical code (research.md § Chosen approach — evidence) — into one generic, unexported installer in a new file, `internal/agent/managed_section.go`: `installManagedSection(projectPath string, cfg config.Config, out io.Writer, templatePath, heading, actionNoun string) error`, backed by generic `locateManagedSection(body []byte, heading string) (startIdx, endIdx int, found bool)`, `appendManagedSection(body []byte, rendered string) []byte`, and `replaceManagedSection(body []byte, startIdx, endIdx int, rendered string) []byte` helpers. The two already-shared helpers `writeAGENTSAtomic` and `writeFileAtomic` (today defined once in `memory_context.go:120-144` and also used by `claude.go` for `CLAUDE.md`) relocate to this same new file, since it becomes the natural home for all managed-section machinery regardless of which section calls it.

`memory_context.go` and `spec_trigger.go` shrink to just their existing constants (`memoryContextTemplatePath`/`memoryContextHeading`, `specTriggerTemplatePath`/`specTriggerHeading`) plus a thin wrapper function each — `installMemoryContextSection` and `installSpecTriggerSection` keep their exact current names and signatures, and simply forward to `installManagedSection` with their constants and existing action-noun strings. This is the load-bearing design decision: callers in `claude.go`, `codex.go`, and `bob.go` do not change at all, and — critically — none of the 10 existing tests in `memory_context_test.go`/`spec_trigger_test.go` need to change, since they exercise `installMemoryContextSection`/`installSpecTriggerSection` by name with hand-maintained literal expected-output strings (research.md § Files examined). This was chosen over calling the generic function directly from each agent's `Install` (rejected Option B — see research.md § Alternatives considered and rejected) specifically because that alternative would force rewriting all 10 existing tests for no functional gain, which cuts against the spec's explicit "must not change the observable behavior... of the existing sections" constraint (spec 000036, Constraints).

Second, add the new section as a third, symmetric caller of the same machinery: a new file `internal/agent/draft_presentation.go` defines `draftPresentationTemplatePath = "agents/draft-presentation.md"` and `draftPresentationHeading = "## Presenting Drafts and Confirmations"`, plus `installDraftPresentationSection`, following the exact same thin-wrapper shape as the other two. A new template, `templates/agents/draft-presentation.md`, replicates the established structure verbatim — H2 heading, blank line, the 3-line "Managed by `{{command}} init` — edit `templates/agents/draft-presentation.md` in the Spektacular source..." blockquote (using the `{{command}}` mustache form, matching `spec-trigger.md`'s style rather than `memory-context.md`'s hardcoded-string quirk, since the quirk is a one-off historical artifact and not a convention to replicate — research.md § Chosen approach), blank line, then body prose stating the spec's core rule: show drafted content as plain conversational text first, then ask any yes/no or multiple-choice confirmation as a separate, short follow-up question — never embedding the draft itself inside a structured choice/dialog element. `installDraftPresentationSection` is called as the last of the three section installs in each agent's `Install` method (`claude.go`, `codex.go`, `bob.go`), immediately after `installSpecTriggerSection`, so the new section is appended after "Spec-Worthy Discussion Recognition" in `AGENTS.md` — consistent with the existing append-in-call-order behavior the `locate*Section` boundary scan already relies on (research.md § Chosen approach, tessl-block boundary note).

This direction beats copying the pattern a third time (rejected Option C) because the spec's Technical Approach section explicitly mandates the refactor-first order, and beats the direct-call alternative (rejected Option B) because it is strictly lower-risk: every currently-passing test keeps passing unmodified, `writeFileAtomic`'s existing dependents (`claude.go:52,73`) are unaffected since the function keeps its same unexported name and signature in the same package, and the only new test surface needed is for the new generic helpers plus the new third section's install/idempotency/cross-agent behavior — additive, not a rewrite. Full alternatives and evidence: research.md § Alternatives considered and rejected, § Chosen approach — evidence.

## Component Breakdown

- **Generic managed-section installer** (new): owns the section-agnostic logic for writing or updating a single managed heading block inside a project's `AGENTS.md` — rendering a template against the agent config, locating an existing section by heading text, and either appending a new section or replacing an existing one in place while preserving all surrounding content untouched. Every other component in this feature depends on it; it has no dependencies of its own beyond the existing atomic-file-write helpers it now owns.

- **Atomic AGENTS.md/file writers** (relocated, not new): the low-level "write to a temp path then rename into place, logging one line describing the action" helpers. Already shared before this change; they move to live alongside the generic installer since that is now the natural place for all managed-section machinery to live. Their behavior and call signature are unchanged — the "ensure CLAUDE.md imports AGENTS.md" component (below) keeps depending on the file-writer directly, independent of the section-installer logic.

- **Memory & Context section** (changed): keeps its existing identity (heading text, template content, install entry point) but its installation logic now delegates to the generic managed-section installer instead of owning a private copy of the byte-manipulation logic. No change to what gets written to `AGENTS.md` or when.

- **Spec-Worthy Discussion Recognition section** (changed): same relationship to the generic installer as Memory & Context — keeps its identity, delegates its mechanics.

- **Presenting Drafts and Confirmations section** (new): the third managed section introduced by this feature. Owns one template describing the standing rule — show drafted content as normal readable text first, then ask any structured yes/no or multiple-choice confirmation as a short, separate follow-up, never embedding the draft itself inside the confirmation UI. Installed the same way as the other two sections, via the generic installer, and appended after Spec-Worthy Discussion Recognition in `AGENTS.md`.

- **Ensure CLAUDE.md imports AGENTS.md** (unchanged): the Claude-specific step that makes sure `CLAUDE.md` references `AGENTS.md`. Not part of the managed-section pattern and not modified by this feature, but noted because it depends on the atomic file-writer component staying available under the same name after the relocation.

- **Per-agent install orchestration** (changed, Claude/Codex/Bob): each agent surface's top-level install routine already orchestrates a sequence of steps (skills, command wrappers where applicable, then the managed AGENTS.md sections). Each gains exactly one more step in that sequence — installing the new Presenting Drafts and Confirmations section — positioned immediately after the existing Spec-Worthy Discussion Recognition step. No other step in any agent's sequence changes.

## Data Structures & Interfaces

This feature introduces one new internal function signature and reuses the existing `config.Config` type unchanged. No new exported types, no new serialization formats, no new persisted state.

**`installManagedSection`** — the generic entry point every section's thin wrapper calls:

```go
func installManagedSection(
    projectPath string,
    cfg config.Config,
    out io.Writer,
    templatePath, heading, actionNoun string,
) error
```

Role: given a template to render and a heading to locate, ensure `AGENTS.md` in `projectPath` contains exactly one up-to-date copy of that section, creating the file if absent, appending if the heading isn't present, or replacing in place if it is. `actionNoun` supplies the human-readable phrase logged to `out` (e.g. "Memory & Context section", "Presenting Drafts and Confirmations section") — mirrors the string each existing wrapper already hard-codes today.

**Three thin wrapper functions**, one per section, each with a fixed signature `func(projectPath string, cfg config.Config, out io.Writer) error` — this is the existing signature `installMemoryContextSection` and `installSpecTriggerSection` already have today, unchanged, plus a new `installDraftPresentationSection` matching it. Each wrapper's body is a single call to `installManagedSection` supplying that section's own template path, heading, and action noun constants. This is the contract every agent's install orchestration (Claude/Codex/Bob) depends on — from their perspective nothing changes: they call three functions with this exact signature, in a fixed order, and get back an error or nil.

**Three internal byte-manipulation helpers**, generalized from their per-section counterparts, used only by `installManagedSection` — not called directly by anything outside this component:

```go
func locateManagedSection(body []byte, heading string) (startIdx, endIdx int, found bool)
func appendManagedSection(body []byte, rendered string) []byte
func replaceManagedSection(body []byte, startIdx, endIdx int, rendered string) []byte
```

`locateManagedSection` takes the heading as a parameter (previously baked into each per-section variant as a package constant); `appendManagedSection` and `replaceManagedSection` are unchanged in shape since their existing per-section bodies never referenced anything section-specific.

No changes to `config.Config` — the only field consumed is the existing `Command string`. No new on-disk file format — `AGENTS.md` remains plain Markdown, and the new section follows the same heading/blockquote/body shape the other two already establish.

## Implementation Detail

This is a consolidation refactor followed by a straightforward extension, not a new architectural pattern. The codebase already has an established "managed section in a shared Markdown file" shape (heading + provenance blockquote + rendered body, byte-range located and replaced in place); this plan generalizes an implementation that currently exists twice into one implementation that exists once, then reuses it a third time. No new abstraction layer, state machine, or orchestration shape is introduced beyond parameterizing what was previously hard-coded as package constants into function arguments.

The extract-and-parameterize move follows a standard "template method via parameters, not interfaces" shape: rather than introducing a `SectionInstaller` interface or a struct-with-methods, the generic function takes its section-specific values (template path, heading, action noun) as plain arguments. This matches the existing code style in the package — `installWorkflowSkills` and `installCommandWrappers` already take their varying inputs as parameters rather than through an interface — so no new pattern is introduced relative to what the package already does elsewhere; the managed-section code is simply being brought in line with that existing convention instead of continuing to duplicate itself.

Developer experience after this change: a reader who wants to understand "how does Spektacular manage a section of AGENTS.md" now has exactly one place to look — the generic installer and its three helpers — instead of needing to read two files and diff them by eye to find the (currently undocumented) three points of variation. Adding a fourth managed section in the future becomes: write a template, define two constants, add a five-line wrapper function, add one call in each agent's install sequence. That is a smaller unit of work and a smaller diff than today's path of copying an entire file and renaming four functions.

The three per-agent `Install` methods (Claude, Codex, Bob) keep their existing shape — a short sequence of `if err := installX(...); return err` / final-return calls — and simply grow by one more line each, in the same style already used for the first two sections. No change to how those methods are structured or tested beyond that additive line.

The new section's template content itself is new prose, not new code shape: it follows the exact structural contract the other two templates already establish (H2 heading, "Managed by..." provenance blockquote, body prose), so a developer editing `templates/agents/draft-presentation.md` later finds nothing surprising relative to editing the other two.

## Dependencies

- **`github.com/cbroglie/mustache`** — already an existing dependency, used unchanged to render the new template the same way the two existing templates are rendered (`map[string]string{"command": cfg.Command}`). No version change or new usage pattern needed.
- **`internal/config` (`config.Config`)** — already depended on by both existing installers; only the existing `Command` field is consumed. No change to this package required.
- **`templates` embedded filesystem (`templates/templates.go`)** — the `//go:embed all:*` directive already picks up any new file placed under `templates/agents/`, so adding `templates/agents/draft-presentation.md` requires no embed-directive change. Confirmed during discovery (research.md § Chosen approach — evidence).
- **`internal/agent` package internals** — `claude.go`, `codex.go`, `bob.go` (the three `Install` methods) and the existing `memory_context.go`/`spec_trigger.go` are all in-package dependents of the refactor; all live in the same package (`agent`), so no cross-package import changes are needed anywhere in this plan.
- **Existing test suite** (`memory_context_test.go`, `spec_trigger_test.go`, `agent_test.go`'s `withSourceFS` helper) — not a dependency to change, but a constraint this plan depends on continuing to pass unmodified as the primary evidence of behavior-preservation (spec 000036 Constraints). `withSourceFS` itself needs no change — it already generically swaps the package-level `sourceFS` regardless of which template path is read.
- **Spec `000036_present-drafts-as-text-not-dialog`** — the upstream spec this plan implements; already written and committed to `.spektacular/specs/`. No other spec or plan must land first — this work has no planning-level blockers and can start immediately.
- **No external services, no new third-party libraries, no database or network dependency** — this is a pure in-repo, filesystem-only change (reading embedded templates, writing `AGENTS.md` in a target project directory).

## Testing Approach

Spec 000036 defines no Success Metrics — there is nothing to carry into a metric→verification mapping. The spec's Acceptance Criteria instead serve as the verification targets, and this plan's testing strategy is built to cover them directly.

This is primarily a **unit-test-level regression and extension** effort, following the existing table-test conventions already established in `internal/agent`'s test suite (`package agent` white-box tests, `testify/require`, `t.TempDir()` per test, hand-maintained literal expected-output strings rather than renderer-computed ones, and the `withSourceFS` fixture-swap helper for substituting the embedded template filesystem).

**Behavior-preservation coverage (highest priority)** — the existing 10 tests across the two current section test files exercise create-from-missing, append-after-tessl-block, idempotency, surrounding-content preservation, template-change pickup, and cross-agent idempotency for both existing sections. These tests are not rewritten; they continue to call the unchanged `installMemoryContextSection`/`installSpecTriggerSection` entry points by name, and their continuing to pass unmodified is itself the primary evidence that the refactor is behavior-preserving, per the spec's explicit constraint. This is a deliberate choice: rather than writing new assertions to re-prove behavior-preservation, the plan relies on the existing suite as a regression gate.

**New generic-installer coverage** — the extracted `installManagedSection` and its three helpers get their own focused unit tests, parameterized similarly to the existing ones but exercised through a synthetic heading/template rather than a real section, to confirm the generalized locate/append/replace logic behaves identically to what the two per-section variants did before extraction (same boundary-detection rules, same whitespace-normalization rules, same idempotency guarantee).

**New third-section coverage** — the new "Presenting Drafts and Confirmations" section gets the same test shape already proven out for the other two: creates from missing, appends after the tessl block, is idempotent, preserves surrounding content, and picks up template/command changes. The existing `CrossAgentIdempotency` tests in both current test files get extended (or a new one is added) to assert all three headings coexist in `AGENTS.md` with exactly one occurrence each after all three agent surfaces (Claude, Codex, Bob) install in sequence — this is the load-bearing assertion that the feature actually reaches every agent surface required by the spec's constraint ("must apply to every agent surface that already receives AGENTS.md").

**Deliberate gaps** — no end-to-end or integration test spins up a real Claude/Codex/Bob session to observe the installed instruction actually changing runtime draft-presentation behavior; that is not mechanically testable from this repository (it depends on an LLM's behavior when reading the installed instruction) and is out of scope for automated testing. This mirrors the existing test suite's own scope, which stops at "the correct Markdown was written to AGENTS.md" and does not attempt to verify an agent's downstream behavior change.

**Harbor end-to-end suite (`tests/harbor/`)** — a separate, Docker-based harness that runs a full live agent session through the `plan`/`spec` workflow state machine (`internal/steps`) and asserts on the resulting transcript and artefacts. It does not exercise `internal/agent`'s AGENTS.md-installer code at all — that package is only invoked by `spektacular init`, a different command from the `plan new`/`plan goto` flow harbor drives — so this refactor has no code-path overlap with it. As a deliberate confirmation step regardless, the implementer should run the harbor plan-workflow and spec-workflow suites (or at minimum `go build ./...` and `go vet ./...` across the whole module) before considering this feature done, to catch any incidental breakage — e.g. a package-level compile error introduced by the refactor — even though no harbor assertion targets this feature's code directly.

## Milestones & Phases

### Milestone 1: Consolidate the managed-section machinery

**What changes**: This is a purely internal cleanup with no user-visible effect — it is worth its own milestone because it is the spec's explicit prerequisite ("refactor... before adding this feature's new section as a third caller of it") and because landing it separately makes it trivially reviewable and revertible on its own: a reviewer (or CI) can confirm nothing about the installed `AGENTS.md` output changed for any existing project before any new behavior is introduced at all. After this milestone, the two existing managed sections ("Memory & Context" and "Spec-Worthy Discussion Recognition") continue to install into `AGENTS.md` exactly as before, for every agent surface (Claude, Codex, Bob), but the code that does it is now a single shared implementation instead of two duplicated ones.

#### - [x] Phase 1.1: Extract the generic managed-section installer

The two existing section installers do the same job — write or update a named block of text inside a project's `AGENTS.md` — with almost identical code, differing only in which template and heading each one hard-codes. This phase pulls that shared logic out into a single implementation that any section can use, without changing what actually gets written to any project's `AGENTS.md` today.

*Technical detail:* [context.md#phase-11](./context.md#phase-11-extract-the-generic-managed-section-installer)

**Acceptance criteria**:
- [x] A single shared implementation exists that can write or update any named section of `AGENTS.md`, given a template and a heading.
- [x] The existing "Memory & Context" and "Spec-Worthy Discussion Recognition" sections continue to install with byte-for-byte identical output to before this phase, for every agent surface.
- [x] Every test that existed for these two sections before this phase still passes without modification.

#### - [x] Phase 1.2: Re-point the two existing sections at the generic installer

With the shared implementation in place, the two existing sections' own code shrinks down to just their identifying details (which template, which heading, which log message) plus a thin call into the shared logic. This completes the consolidation the spec requires before any new section can be added.

*Technical detail:* [context.md#phase-12](./context.md#phase-12-re-point-the-two-existing-sections-at-the-generic-installer)

**Acceptance criteria**:
- [x] Both existing sections are implemented as thin callers of the shared installer, with no duplicated byte-manipulation logic remaining between them.
- [x] Re-running installation against a project that already has both sections produces no changes (idempotent), exactly as before.
- [x] The full existing test suite for both sections passes without any test file being edited.

**Validation point:** The full existing test suite for both sections (`memory_context_test.go`, `spec_trigger_test.go`) passes unmodified — no test file needs an edit. Re-running install against this repository's own live `AGENTS.md` produces a byte-identical file (no diff).

### Milestone 2: Add the "present drafts as text, not dialog" instruction to every agent surface

**What changes**: Every Spektacular-managed project now gets a third standing instruction installed into its `AGENTS.md`, alongside the existing two: the assistant is told to always show a drafted piece of content (an architecture write-up, a set of options, a written section, a summary) as plain, readable chat text first, and only afterward — as a separate step — ask a short structured yes/no or multiple-choice confirmation. This closes the gap the spec identifies: today nothing tells the assistant *how* to present a draft, so it sometimes embeds the draft itself inside a compressed dialog/choice element where long text is hard to read. This lands for every agent surface that already receives the other two managed sections — Claude, Codex, and Bob — with no surface left out.

#### - [x] Phase 2.1: Write the "Presenting Drafts and Confirmations" template and installer

This phase introduces the actual new user-facing instruction: a template stating that drafted content (an architecture write-up, a set of options, a written section, a summary) must be shown as plain, readable chat text first, with any structured yes/no or multiple-choice confirmation asked only afterward, as a separate short question — never as a substitute for showing the draft. The template follows the same structure as the two existing ones, and its installer is a thin caller of the same shared logic introduced in Milestone 1.

*Technical detail:* [context.md#phase-21](./context.md#phase-21-write-the-presenting-drafts-and-confirmations-template-and-installer)

**Acceptance criteria**:
- [x] A new managed section, distinct in heading from the existing two, states the draft-then-confirm presentation rule in plain language a person can read and act on.
- [x] Installing this section into a project with no existing `AGENTS.md` creates the file with just this section.
- [x] Installing this section into a project that already has the other two sections appends it after them without disturbing either.
- [x] Re-running installation is a no-op (idempotent).

#### - [x] Phase 2.2: Wire the new section into every agent surface

The new section is connected to Claude's, Codex's, and Bob's install sequences, immediately after the existing "Spec-Worthy Discussion Recognition" step, so every agent surface that already receives the first two managed sections also receives this one. This is the phase that makes the feature actually reach users — nothing before this point changes what any real project sees when it runs installation.

*Technical detail:* [context.md#phase-22](./context.md#phase-22-wire-the-new-section-into-every-agent-surface)

**Acceptance criteria**:
- [x] Running installation for Claude, for Codex, and for Bob each produces an `AGENTS.md` containing all three managed sections, each appearing exactly once, in the order Memory & Context → Spec-Worthy Discussion Recognition → Presenting Drafts and Confirmations.
- [x] Running installation twice in a row for any of the three agent surfaces produces no further changes.
- [x] No other step in any agent surface's install sequence (skills, command wrappers, the Claude-specific `AGENTS.md` import check) changed behavior.

**Validation point:** Running `spektacular init` (or the equivalent per-surface install) against a fresh project produces an `AGENTS.md` containing all three managed sections, each appearing exactly once, in the order Memory & Context → Spec-Worthy Discussion Recognition → Presenting Drafts and Confirmations. Running it again is a no-op (idempotent). This holds for all three agent surfaces.

## Open Questions

None. Every design decision this plan depends on — the refactor shape, the new section's heading and template filename, the shared code's file layout, the exact call-site insertion point in each agent surface, and the test strategy — was resolved during the architecture, data structures, and phases steps rather than deferred. The codebase itself is small and fully read during discovery (both source files, both test files, all three call sites, both templates, and the live `AGENTS.md` were read in full), so there is no unread code path likely to surface a hidden assumption during implementation.

## Out of Scope

- **Retroactively fixing any past conversation or transcript** where content was already shown inside a dialog. This is a forward-looking instruction only, not a corrective action on history. (Spec 000036 § Non-Goals.)
- **Editing the ~20 individual step templates** across the spec/plan/implement workflows (`templates/steps/*`) to state the draft-then-confirm rule locally in each one. The spec's Technical Approach explicitly requires delivering this as a single managed AGENTS.md section instead, precisely to avoid that repetitive, drift-prone editing. (Spec 000036 § Constraints; this plan's architecture step, rejected Option C.)
- **Calling the generic installer directly from each agent's `Install` method, with no per-section wrapper functions.** Considered and rejected during this plan's architecture step (Option B) because it would require rewriting all 10 existing tests for `installMemoryContextSection`/`installSpecTriggerSection` with no functional benefit — a higher-risk path than keeping thin wrappers. Not pursued.
- **Any change to what the "Memory & Context" or "Spec-Worthy Discussion Recognition" sections say or how they render.** This plan's refactor is required to be strictly behavior-preserving for both — including the deliberate, pre-existing asymmetry where `memory-context.md` hardcodes `spektacular init` while `spec-trigger.md` uses the `{{command}}` placeholder. Neither is "fixed" or normalized by this work. (Spec 000036 § Constraints.)
- **An automated, end-to-end test that verifies an LLM's actual runtime behavior changes** after reading the new installed instruction (i.e. that the assistant literally stops using dialog UI for drafts in a live session). Not mechanically testable from this repository; this plan verifies only that the correct instruction text is installed to `AGENTS.md`, matching the existing test suite's own scope for the other two sections. (This plan's testing_approach step.)

## Changelog

### FINAL SUMMARY

This plan delivered a two-milestone feature. Milestone 1 consolidated the near-duplicate "Memory & Context" and "Spec-Worthy Discussion Recognition" section installers into a single generic, unexported `installManagedSection` (backed by `locateManagedSection`/`appendManagedSection`/`replaceManagedSection` helpers) in a new `internal/agent/managed_section.go`, with both existing sections re-pointed to thin wrapper calls — confirmed behavior-preserving via a byte-identical diff against this repository's own live `AGENTS.md`/`CLAUDE.md` and all 10 pre-existing tests passing completely unmodified. Milestone 2 added the new third managed section, "Presenting Drafts and Confirmations" (new template + thin installer following the identical pattern), and wired it into all three agent surfaces' `Install` methods (Claude, Codex, Bob), immediately after the existing Spec-Worthy Discussion Recognition step — confirmed via an extended cross-agent test asserting all three sections coexist, each exactly once, in the correct order, after all three agents install in sequence into a shared directory.

**Total phases**: 4/4 completed (1.1, 1.2, 2.1, 2.2)

**Notable deviations from the plan**: One plan-documentation drift was found and fixed during the `read_plan` validation gate, before any code was written: `plan.md` and `context.md` referenced `templates/embed.go` as the file holding the `//go:embed all:*` directive; it actually lives in `templates/templates.go` (`embed.go` doesn't exist). The directive's described content and behavior were accurate — a filename typo only. Both plan documents were corrected via the plan store. No other deviations — every phase's acceptance criteria were met exactly as specified, with no scope changes, descoped requirements, or substitutions.

### 2026-07-15 — Phase 1.1: Extract the generic managed-section installer

**What was done**: Extracted the near-duplicate byte-manipulation logic shared by the "Memory & Context" and "Spec-Worthy Discussion Recognition" section installers into a single generic implementation in a new file, `internal/agent/managed_section.go` (`installManagedSection`, `locateManagedSection`, `appendManagedSection`, `replaceManagedSection`), and relocated the already-shared `writeAGENTSAtomic`/`writeFileAtomic` helpers into the same file. The two existing sections were not yet repointed to call the new function (that's Phase 1.2) — this phase only adds the new shared implementation without changing any existing behavior.

**Deviations**: None in code. One plan-documentation drift was found and fixed before implementation began: `plan.md` and `context.md` referenced `templates/embed.go` as the file containing the `//go:embed all:*` directive; it actually lives in `templates/templates.go` (`embed.go` does not exist). The directive's content and behavior were accurately described — this was a filename typo only. Both documents were corrected via the plan store before proceeding.

**Files changed**:
- `internal/agent/managed_section.go`
- `internal/agent/managed_section_test.go`
- `internal/agent/memory_context.go`

**Discoveries**: None beyond the embed.go/templates.go drift noted above (also recorded in this plan's own working `context.md`, and in the session's cross-cutting `.spektacular/context.md`). `research.md` still references `templates/embed.go` in a couple of places; this was left uncorrected since research.md is a historical evidence document not re-validated by the read_plan gate, and the actual directive's location and behavior are unaffected.

### 2026-07-15 — Phase 1.2: Re-point the two existing sections at the generic installer

**What was done**: Replaced the bodies of `installMemoryContextSection` and `installSpecTriggerSection` with single delegating calls to `installManagedSection`, passing each section's own template path, heading, and action-noun constants. Deleted the now-redundant `locate`/`append`/`replace` trio and unused imports from both `memory_context.go` and `spec_trigger.go`. No changes to `claude.go`, `codex.go`, `bob.go`, or either test file.

**Deviations**: None.

**Files changed**:
- `internal/agent/memory_context.go`
- `internal/agent/spec_trigger.go`

**Discoveries**: Confirmed the behavior-preservation constraint directly: ran `go run . init claude` against this live repository and diffed the resulting `AGENTS.md`/`CLAUDE.md` against the committed versions — zero diff, byte-for-byte identical to pre-refactor output. All 10 pre-existing tests across both section test files pass completely unmodified.

### 2026-07-15 — Phase 2.1: Write the "Presenting Drafts and Confirmations" template and installer

**What was done**: Added the new third managed section: `templates/agents/draft-presentation.md` (H2 heading, provenance blockquote using the `{{command}}` mustache form, and body prose covering the spec's three requirements — draft shown as plain text first, confirmation asked separately afterward, applies across every workflow) and `internal/agent/draft_presentation.go` (`draftPresentationTemplatePath`/`draftPresentationHeading` constants plus `installDraftPresentationSection`, a thin wrapper delegating to `installManagedSection`).

**Deviations**: None.

**Files changed**:
- `templates/agents/draft-presentation.md`
- `internal/agent/draft_presentation.go`
- `internal/agent/draft_presentation_test.go`

**Discoveries**: None. The new section is not yet wired into any agent's `Install` method — `installDraftPresentationSection` is currently unused, by design; Phase 2.2 wires it in.

### 2026-07-15 — Phase 2.2: Wire the new section into every agent surface

**What was done**: Added a call to `installDraftPresentationSection` in each of the three agents' `Install` methods (`claude.go`, `codex.go`, `bob.go`), immediately after the existing `installSpecTriggerSection` call and before `claude.go`'s `ensureClaudeImportsAGENTS` return. Extended the existing `TestInstallSpecTriggerSection_CrossAgentIdempotency` test to additionally assert the third heading appears exactly once and that all three headings appear in the correct order (Memory & Context → Spec-Worthy Discussion Recognition → Presenting Drafts and Confirmations) after Claude, Codex, and Bob install in sequence into a shared directory.

**Deviations**: None.

**Files changed**:
- `internal/agent/claude.go`
- `internal/agent/codex.go`
- `internal/agent/bob.go`
- `internal/agent/spec_trigger_test.go`

**Discoveries**: None. This was the final phase; all 4 phases across both milestones are now complete. `go build ./...`, `go vet ./...`, and `go test ./...` (whole module) all pass with no regressions.
