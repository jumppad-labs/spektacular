# Context: present-drafts-as-text-not-dialog

## Problem identified and why it needs solving

The plan workflow's step prompts (and, on inspection, the spec and implement
workflow prompts too) frequently instruct the agent to "present" drafted
content and "get the user's agreement" or "confirm" it, without saying *how*
to present it. In practice this sometimes leads the agent to reach for a
structured decision/dialog UI element (e.g. a multiple-choice or yes/no
prompt component) and put the drafted content itself inside it. Dialog/option
UI truncates or compresses long text, making it hard for the user to actually
read the full drafted content (an architecture write-up, a set of milestones,
a drafted spec section, etc.).

## User's exact phrasing for the key requirement

"I recommend we add something like 'Present the drafted content directly as
normal chat text (not via a decision-style prompt/dialog) so the user can
read it in full. Only use a structured question/dialog to ask a yes/no or
multiple-choice confirmation afterward (e.g. "does this look right, or
should anything change?") — never to contain the content itself, since
option/dialog UI truncates or compresses long text and makes it hard to
read.'"

## Requirements discussed

1. Drafted content (architecture options, milestones, plan sections, spec
   sections, changelog summaries, etc.) must always be presented as normal
   chat/text output — never embedded inside a structured decision-dialog
   component.
2. A structured question/dialog UI is fine, but only for the *confirmation*
   itself — a short yes/no or multiple-choice question asked *after* the
   content has already been shown as plain text (e.g. "does this look right,
   or should anything change?").
3. This should not repeat/duplicate itself in every one of the ~20 affected
   step template files (spec, plan, and implement workflows all have this
   "draft then confirm" pattern) — that would be a large, easy-to-drift-out-
   of-sync diff.

## Scope decision (resolved during this conversation, before spec drafting)

Discussed and explicitly chose: add this as a **third managed AGENTS.md
section** (same durable, per-project-installed pattern as the existing
"Memory & Context" and "Spec-Worthy Discussion Recognition" sections),
rather than:
- editing all ~20 affected step templates directly (rejected: large,
  repetitive, drifts easily), or
- a plan-workflow-only fix (rejected: same problem exists in spec and
  implement workflow templates too, not just plan).

Confirmed via direct code reading during this conversation:
- `AGENTS.md`'s existing two managed sections ("Memory & Context",
  "Spec-Worthy Discussion Recognition") are each installed/kept-in-sync by
  `spektacular init`, backed by `internal/agent/memory_context.go` and
  `internal/agent/spec_trigger.go` respectively, and rendered from
  `templates/agents/memory-context.md` / `templates/agents/spec-trigger.md`.
  Both installer functions are called from `internal/agent/claude.go`,
  `internal/agent/codex.go`, and `internal/agent/bob.go`.
- `memory_context.go` and `spec_trigger.go` are structurally near-identical:
  each defines a template path + heading constant, an `installXSection`
  function (read embedded template → mustache-render with `{"command":
  cfg.Command}` → find-or-append-or-replace the managed section in
  `AGENTS.md`, byte-range-located between its heading and the next `# `/`## `
  heading), plus `locateXSection`/`appendXSection`/`replaceXSection` helpers.
  This is ~120 lines duplicated near-verbatim between the two files.

**Decision**: rather than copy this pattern a third time (near-verbatim
`review-presentation.go`), refactor `memory_context.go` + `spec_trigger.go`
into one shared generic helper (parameterized by heading + template path),
then add the new "present drafts as text, not dialog" section as the third
caller of that shared helper. This was an explicit user choice made via
AskUserQuestion between three options (refactor-first / copy-as-is /
templates-only-no-Go-change) — refactor-first was chosen.

## Alternatives considered and rejected

- **Editing every affected step template directly** (~20 files across
  spec/plan/implement) — rejected: repetitive, large diff, drifts out of
  sync as new steps are added later.
- **Plan-workflow-only fix** — rejected: the same "draft then confirm without
  saying how" pattern exists in spec and implement workflow templates too,
  not unique to plan.
- **Copying the existing spec_trigger.go pattern a third time as-is** —
  considered, but explicitly rejected in favor of refactoring first, since
  memory_context.go and spec_trigger.go are already near-duplicates and a
  third copy would make the duplication worse rather than consolidating it.

## Overview step: user confirmed

User confirmed the drafted overview as-is ("ok"), no edits requested. Saved
to `.spektacular/work/000036_present-drafts-as-text-not-dialog/overview.md`.
Proceeding to requirements step.

## Requirements step: user confirmed

User confirmed all 3 requirements as-is ("ok"): plain-text draft
presentation, confirmation asked separately afterward, applies across all
workflows. Saved to
`.spektacular/work/000036_present-drafts-as-text-not-dialog/requirements.md`.
Proceeding to acceptance_criteria step.

## Acceptance criteria step: user confirmed

User confirmed all 3 acceptance criteria as-is ("ok"), one per requirement.
Saved to
`.spektacular/work/000036_present-drafts-as-text-not-dialog/acceptance_criteria.md`.
Proceeding to constraints step.

## Constraints step: user confirmed

User confirmed all 3 constraints as-is ("ok"): must use managed AGENTS.md
section pattern (not scattered template edits); refactor of shared install
code must be behavior-preserving for the two existing sections; must apply
to all three agent surfaces (Claude, Codex, Bob). Saved to
`.spektacular/work/000036_present-drafts-as-text-not-dialog/constraints.md`.
Proceeding to technical_approach step.

## Technical approach step: user confirmed

User confirmed the one technical-direction bullet as-is ("ok"): refactor the
two existing near-duplicate section installers into one shared generic
installer before adding the third section as a caller of it. Saved to
`.spektacular/work/000036_present-drafts-as-text-not-dialog/technical_approach.md`.
Proceeding to success_metrics step.

## Success metrics step: user confirmed

No metrics defined, matching plan 000035's precedent (prose-only feature,
acceptance criteria serve as verification targets instead). User confirmed.
Saved to
`.spektacular/work/000036_present-drafts-as-text-not-dialog/success_metrics.md`.
Proceeding to non_goals step.

## Non-goals step: user confirmed

User confirmed all 3 proposed non-goals as-is ("ok"): no per-template edits;
no change to the two existing sections' wording/behavior; no retroactive
fix of past conversations. Saved to
`.spektacular/work/000036_present-drafts-as-text-not-dialog/non_goals.md`.
Proceeding to verification step.

## Verification step complete

Assembled spec from all 7 working files, spawned an independent fresh-eyes
reviewer subagent (no conversation context, read only the staged file). It
found 2 real duplication issues (Non-Goals restating Constraints bullets in
reverse — both dropped) and 1 wording nit in Technical Approach ("rather than
copying the pattern a third time" reworded to "rather than adding a third
near-identical implementation" to read as adaptable direction, not a
prohibition). User confirmed all 3 fixes. Final Non-Goals has 1 bullet
(retroactive-fix exclusion); the other 2 were genuine duplicates of
Constraints, correctly dropped per the review brief's guidance that a
constraint-in-reverse is duplication, not a deliberate keep. Spec committed
to `.spektacular/specs/000036_present-drafts-as-text-not-dialog.md`, working
directory removed. Proceeding to finished step.

## Plan workflow: overview step complete

Started `go run . plan new` against this spec. Read the full spec file. Scope
for the plan: refactor `memory_context.go` + `spec_trigger.go` into one
shared generic installer, then add a new "present drafts as text, not
dialog" managed AGENTS.md section as the third caller of that helper.
Behavior-preserving for the two existing sections; must apply to Claude,
Codex, and Bob surfaces. The codebase details already captured above (file
names, structure, ~120 lines of near-duplicate code) carry forward directly
into the plan's discovery step — no need to re-derive them. Proceeding to
discovery step.

## Plan workflow: discovery step complete

Spawned an Explore agent to research `internal/agent/memory_context.go`,
`spec_trigger.go`, their three call sites (claude.go/codex.go/bob.go), their
test files, and the two templates. Verified key claims directly (read
memory_context.go in full, read live repo-root AGENTS.md in full) — findings
confirmed accurate. Distilled into
`.spektacular/work/000036_present-drafts-as-text-not-dialog/research.md`
(7 required sections: alternatives, chosen-approach evidence, files examined,
external references, prior plans/specs, open assumptions, rehydration cues).

Key facts carried forward:
- The two installers differ only in: template path const, heading const,
  and 2 action-noun log strings. Everything else (byte-range heading scan,
  append/replace logic) is 100% identical code.
- `writeAGENTSAtomic`/`writeFileAtomic` already live in memory_context.go
  and are shared (writeFileAtomic also used by claude.go for CLAUDE.md) —
  must stay package-visible after refactor regardless of which file houses
  the new generic installer.
- All 3 agent surfaces (claude.go:25,28 / codex.go:17,20 / bob.go:20,23)
  call installMemoryContextSection then installSpecTriggerSection, same
  order, same AGENTS.md path, same cfg — no per-surface variation.
- Deliberate pre-existing quirk: memory-context.md template hardcodes
  `spektacular init` (no {{command}}), spec-trigger.md uses {{command}}
  (rendered). Must NOT be "fixed" — this asymmetry is intentional per prior
  plan 000023.
- Existing tests call installMemoryContextSection/installSpecTriggerSection
  by *name* with hand-maintained literal expected strings (not
  renderer-computed) — lowest-risk refactor keeps these as thin wrappers
  around one generic installer so no existing test needs to change.
- Open assumption (flagged in research.md, not yet resolved): exact heading
  text + template filename for the new third section — spec only describes
  it conversationally. Deferred to architecture step, not blocking
  discovery.
- No relevant knowledge-base entries exist yet (searched "AGENTS.md
  installer", "managed section" — only 2 unrelated architecture-doc hits
  about the spec/plan/implement FSM, not this pattern). Nothing written to
  knowledge base this step (nothing user-confirmed as durable beyond what's
  already in research.md).

Proceeding to architecture step.

## Plan workflow: architecture step complete

Presented 3 options (A: generic installer + thin per-section wrappers /
B: generic installer called directly, no wrappers, requires rewriting all
10 existing tests / C: copy the pattern a third time, violates spec's
refactor-first constraint). User chose **Option A** (recommended).

Resolved two remaining design decisions via AskUserQuestion:
- New section heading: `## Presenting Drafts and Confirmations`
- New template filename: `templates/agents/draft-presentation.md`
- New shared-code file: `internal/agent/managed_section.go` (houses generic
  installManagedSection/locateManagedSection/appendManagedSection/
  replaceManagedSection, plus relocated writeAGENTSAtomic/writeFileAtomic).
  memory_context.go and spec_trigger.go shrink to constants + thin wrapper
  each; new draft_presentation.go gets the same treatment as a third file.

Key locked decision: installMemoryContextSection/installSpecTriggerSection
keep their exact current names/signatures as thin wrappers around
installManagedSection — zero changes to any of the 10 existing tests, zero
changes to claude.go/codex.go/bob.go call sites for the two existing
sections. installDraftPresentationSection called last in each agent's
Install (after installSpecTriggerSection), so the new section appends after
"Spec-Worthy Discussion Recognition" in AGENTS.md.

New template uses the {{command}} mustache form (spec-trigger.md style),
NOT the hardcoded-string quirk from memory-context.md (that quirk is
historical/one-off, not a convention to replicate).

Saved to .spektacular/work/000036_present-drafts-as-text-not-dialog/
architecture.md.

## Plan workflow: conventions step complete

Checked go run . knowledge always-applied again at this step (per
instructions) — still only the 2 category READMEs (conventions/glossary),
no populated convention entries exist. Proposed "no conventions apply" to
user, confirmed. Saved to .spektacular/work/000036_present-drafts-as-text-not-dialog/conventions.md.

Proceeding to components step.

## Plan workflow: components step complete

User confirmed the component breakdown as-is ("ok"): generic managed-section
installer, relocated atomic file writers, Memory & Context (changed to
delegate), Spec-Worthy Discussion Recognition (changed to delegate),
Presenting Drafts and Confirmations (new), CLAUDE.md-import step
(unchanged, noted for dependency only), per-agent install orchestration
(each gains one more step). Saved to
.spektacular/work/000036_present-drafts-as-text-not-dialog/components.md.
Proceeding to data_structures step.

## Plan workflow: data_structures step complete

User confirmed as-is ("ok"): installManagedSection generic signature, three
thin wrapper functions keeping existing signature `func(projectPath string,
cfg config.Config, out io.Writer) error` (installMemoryContextSection/
installSpecTriggerSection unchanged, installDraftPresentationSection new but
matching), three internal helpers (locateManagedSection/appendManagedSection/
replaceManagedSection) generalized from per-section variants. No new types,
no new serialization format. Saved to
.spektacular/work/000036_present-drafts-as-text-not-dialog/data_structures.md.
Proceeding to implementation_detail step.

## Plan workflow: implementation_detail step complete

User confirmed as-is ("ok"): framed as consolidation refactor + straightforward
extension, not a new pattern; generic function uses plain parameters (not an
interface), matching installWorkflowSkills/installCommandWrappers convention
already in the package; per-agent Install methods keep existing shape, grow
by one line each. Saved to
.spektacular/work/000036_present-drafts-as-text-not-dialog/implementation_detail.md.
Proceeding to dependencies step.

## Plan workflow: dependencies step complete

User confirmed as-is ("ok"): mustache (unchanged), config.Config (unchanged),
templates embed (no directive change needed), internal/agent package
internals (all in-package), existing test suite (relied on as
behavior-preservation evidence, not itself changed), spec 000036 (already
committed, no blockers), no external services/new libraries. Saved to
.spektacular/work/000036_present-drafts-as-text-not-dialog/dependencies.md.
Proceeding to testing_approach step.

## Plan workflow: testing_approach step complete

User confirmed as-is ("ok"), plus one addition mid-step: user asked to make
sure this doesn't break "the harbour tests" — investigated, found
`tests/harbor/` (correct spelling "harbor", not "harbour") is a separate
Docker-based end-to-end harness that runs a live agent session through the
plan/spec workflow state machine (internal/steps) and asserts on transcript
+ artefacts (test_plan_workflow.py, test_spec_workflow.py). Confirmed it does
NOT exercise internal/agent's AGENTS.md-installer code — that package is only
invoked by `spektacular init`, a different command from `plan new`/`plan
goto` which harbor drives. No code-path overlap. Added a note to
testing_approach.md recommending the implementer still run harbor suites (or
at minimum `go build ./...` / `go vet ./...`) as cheap deliberate insurance
against incidental breakage, even without direct overlap. Saved to
.spektacular/work/000036_present-drafts-as-text-not-dialog/testing_approach.md.
Proceeding to milestones step.

## Plan workflow: milestones step complete

User confirmed as-is ("ok"): 2 milestones. M1 = consolidate managed-section
machinery (internal cleanup, no user-visible effect, justified as spec's
explicit prerequisite + independent revertibility). M2 = add "Presenting
Drafts and Confirmations" section to all 3 agent surfaces (the user-visible
feature). Saved to
.spektacular/work/000036_present-drafts-as-text-not-dialog/milestones.md.
Proceeding to phases step.

## Plan workflow: phases step complete

User confirmed as-is ("ok"): 4 phases across 2 milestones.
- Phase 1.1: extract generic installer into new managed_section.go, old
  files untouched (compiles/passes throughout).
- Phase 1.2: re-point memory_context.go/spec_trigger.go to thin wrappers,
  delete now-redundant locate/append/replace trios (confirmed nothing
  external calls them by name).
- Phase 2.1: new draft-presentation.md template + draft_presentation.go
  (constants: draftPresentationTemplatePath = "agents/draft-presentation.md",
  draftPresentationHeading = "## Presenting Drafts and Confirmations").
- Phase 2.2: wire installDraftPresentationSection into claude.go/codex.go/
  bob.go Install methods (after installSpecTriggerSection call in each),
  extend CrossAgentIdempotency test coverage to 3 headings.
Exact current file:line references verified via grep -n against live source
before writing (memory_context.go 144 lines, claude.go Install 18-32,
codex.go Install 13-21, bob.go Install 13-24, test func names at lines
30/42/62/81/117/130 in both existing test files).
Saved to .spektacular/work/000036_present-drafts-as-text-not-dialog/
phases_plan.md and phases_context.md.
Proceeding to open_questions step.

## Plan workflow: open_questions step complete

User confirmed as-is ("ok"): no open questions, all design decisions
resolved during architecture/data_structures/phases steps, codebase fully
read during discovery. Saved to
.spektacular/work/000036_present-drafts-as-text-not-dialog/open_questions.md.
Proceeding to out_of_scope step.

## Plan workflow: out_of_scope step complete

User confirmed as-is ("ok"): 5 exclusions — retroactive transcript fixes
(from spec Non-Goals), per-template editing (rejected per spec Technical
Approach), direct-call-no-wrappers (rejected architecture Option B), any
change to the 2 existing sections' output/asymmetry, automated LLM-behavior
verification (not mechanically testable). Saved to
.spektacular/work/000036_present-drafts-as-text-not-dialog/out_of_scope.md.
Proceeding to assemble step.

## Plan workflow: assemble step complete

Gathered metadata: 2026-07-15T07:02:15Z, commit e401357, branch main,
repo git@github.com:jumppad-labs/spektacular.git. Feature slug already fixed
by the workflow's plan_path (000036_present-drafts-as-text-not-dialog) — no
AskUserQuestion needed, it matches the spec's own numbering already.

Assembled all 3 documents from the 13 working files (overview.md was never
a separate working file — its step only asked to read the spec, so drafted
Overview prose directly from spec + confirmed milestones/architecture
content). Staged to .spektacular/tmp/plan_template.md (181 lines),
context_template.md (126 lines), research_template.md (79 lines).

Notable: rewrote research.md's "Open assumptions" section during assembly —
the 2 items flagged during discovery (heading/filename naming, wrapper
design) were both resolved as explicit decisions during the architecture
step, so the assembled research.md now says "None outstanding" and records
what each was resolved to, rather than leaving stale open items in the
final document.

Proceeding to verification step.

## Plan workflow: verification step complete

Checked all 3 staged documents against required section lists. Found
context.md was missing `## Project References` (between Per-Phase Technical
Notes and Testing Strategy) — not one of the 13 working files gathered
earlier since it wasn't its own explicit step. Backfilled it: wrote
.spektacular/work/000036_present-drafts-as-text-not-dialog/
project_references.md (durable working file) and inserted the matching
section into the staged .spektacular/tmp/context_template.md. Content pulled
from spec 000036, plan 000023 (memory-context precedent), and the harbor
test-suite no-overlap finding.

plan.md (11/11 sections) and research.md (7/7 sections) needed no fixes.
Verified plan.md's 4 Technical detail: links resolve exactly to context.md's
4 Phase N.M headings. Verified phase acceptance-criteria content has no
shell commands or file:line refs (file:line refs that do appear are
correctly confined to the Architecture & Design Decisions section, which
permits them as supporting evidence per the scaffold's own rules).

Proceeding to write_plan step.

## Plan workflow: write_plan step complete

Committed plan.md via `go run . plan file write
000036_present-drafts-as-text-not-dialog/plan.md --from
.spektacular/tmp/plan_template.md`. Removed scratch file. Verified committed
file exists at .spektacular/plans/000036_present-drafts-as-text-not-dialog/
plan.md, 181 lines matching staged version. Proceeding to write_context
step.

## Plan workflow: write_context step complete

Committed context.md via `go run . plan file write
000036_present-drafts-as-text-not-dialog/context.md --from
.spektacular/tmp/context_template.md`. Removed scratch file. Verified
committed file exists, 132 lines. Proceeding to write_research step.

## Plan workflow: write_research step complete

Committed research.md via `go run . plan file write
000036_present-drafts-as-text-not-dialog/research.md --from
.spektacular/tmp/research_template.md`. Removed scratch file. Verified
committed file exists, 79 lines. All 3 plan documents now live in
.spektacular/plans/000036_present-drafts-as-text-not-dialog/ (plan.md 181,
context.md 132, research.md 79 lines). Removed the working directory
.spektacular/work/000036_present-drafts-as-text-not-dialog/ — all content
durably committed to the store. Proceeding to finished step.

## Plan workflow: finished

Plan workflow complete. All 3 documents committed to
.spektacular/plans/000036_present-drafts-as-text-not-dialog/. Offering user
a choice: read directly, or conversational walkthrough. Per instructions,
offer once, do not repeat if declined/not taken up.

## Where this originated

Raised mid-implement-workflow-completion, in the same conversation as (and
immediately after) finishing plan 000035 (plan-walkthrough-conversation) —
unrelated feature, no code overlap. This is a fresh, separate concern about
prompt/instruction UX across the whole spec/plan/implement family of
workflows, not a continuation of 000035's scope.

## Implement workflow: read_plan step complete

Read all 3 plan documents in full via `go run . plan file read`. Structural
validation passed (all 10 required `## ` sections present; 4 phase
checkboxes each with a resolving `context.md#phase-NM` link).

Drift check found one mismatch: plan.md and context.md referenced
`templates/embed.go` as the file holding the `//go:embed all:*` directive;
it actually lives in `templates/templates.go` (`embed.go` doesn't exist).
Directive content/behavior described was accurate — pure filename typo. User
chose "fix the plan first". Fixed both plan.md (1 occurrence, line 90) and
context.md (3 occurrences, lines 18/70/105) via read → sed → `plan file
write` → remove scratch, re-verified no `embed.go` references remain.
research.md still says `embed.go` (historical evidence doc, not corrected —
harmless, not re-validated by this gate).

Everything else in plan.md/context.md was confirmed byte-accurate against
the live tree: `internal/agent/memory_context.go` (144 lines, not 145 as
research.md states — trivial off-by-one, not worth a second fix round) and
`spec_trigger.go` (117 lines) match described line numbers/functions/bodies
exactly; all three agents' `Install` methods match described call order;
both existing templates match described structure; `agent_test.go`'s
`withSourceFS`, `config.Config.Command`, and the live root `AGENTS.md` (52
lines, sections at lines 7 and 22) all match.

Spec coverage check: read spec 000036 in full. All 3 Requirements, all 3
Constraints, and all 3 Acceptance Criteria have coverage in plan.md's
Milestones & Phases (Milestone 1 = behavior-preserving refactor constraint;
Phase 2.1 = the three content requirements + both content-related acceptance
criteria; Phase 2.2 = the "every agent surface" constraint + the
discoverability acceptance criterion). No gaps, nothing descoped.

Changelog mode: no `## Changelog` section exists in plan.md yet — this is a
**first-phase invocation**. `analyze` should pick up at Phase 1.1 (Extract
the generic managed-section installer), the first `#### - [ ]` phase.

Advancing to `analyze` step next.

## Implement workflow: analyze step complete (Phase 1.1)

Current phase: **1.1 — Extract the generic managed-section installer**
(first unchecked phase; changelog section doesn't exist yet, confirming
first-phase invocation). Technical detail at context.md#phase-11.

Low complexity — did analysis directly instead of spawning sub-agents.
Re-confirmed exact const/func line boundaries via grep (matches context.md
exactly): memory_context.go has `const` at :16, `installMemoryContextSection`
at :25, `locateMemoryContextSection` at :63, `appendMemoryContextSection` at
:86, `replaceMemoryContextSection` at :101, `writeAGENTSAtomic` at :120,
`writeFileAtomic` at :134. spec_trigger.go mirrors with `install`/`locate`/
`append`/`replace` at :26/:64/:87/:102. No mismatches — proceeding straight
to implement.

Plan for this phase: create `internal/agent/managed_section.go` with
`installManagedSection`, `locateManagedSection`, `appendManagedSection`,
`replaceManagedSection`, plus relocate `writeAGENTSAtomic`/`writeFileAtomic`
into it unchanged. Per context.md's explicit sequencing note: do NOT delete
anything from memory_context.go/spec_trigger.go in this phase — leave them
fully functional so the tree compiles and all existing tests pass unchanged
at the end of 1.1. The old `writeAGENTSAtomic`/`writeFileAtomic` in
memory_context.go must be removed as part of this phase (since they're being
relocated, and Go disallows duplicate declarations) — everything else
(install/locate/append/replace functions in both files) stays untouched
until Phase 1.2.

Advancing to `implement` step next.

## Implement workflow: implement step complete (Phase 2.2)

Wired `installDraftPresentationSection` into all 3 agents' `Install`
methods, immediately after `installSpecTriggerSection` and before
`ensureClaudeImportsAGENTS` (claude.go) / as the final returned call
(codex.go, bob.go) — matching context.md's exact prescribed insertion
point and code shape.

`go build`/`go vet` clean. `installDraftPresentationSection` no longer
flagged unused (now called from all 3 Install methods). `go test
./internal/agent/...` — all pass.

Advancing to `test` step next — this is where the cross-agent test
extension happens (asserting all 3 headings coexist, count 1 each, via
real Claude+Codex+Bob `.Install()` in sequence).

## Implement workflow: test step complete (Phase 2.2)

Delegated to sub-agent (again instructed not to touch state.json — no
issue this time). Made a minimal, surgical extension to the existing
`TestInstallSpecTriggerSection_CrossAgentIdempotency` in
`spec_trigger_test.go` (chosen location per context.md's "either placement
is acceptable" note, since it already asserted the first 2 headings):
added a `draftPresentationCount` assertion (exactly 1 occurrence of `##
Presenting Drafts and Confirmations`) plus an explicit order assertion
(`memoryContextIdx < specTriggerIdx < draftPresentationIdx` via
`strings.Index`) confirming the plan's required install order. No other
test, file, or import touched.

Verified independently: `go build`/`go vet` clean, `go test
./internal/agent/... -v` — all tests pass including the extended
cross-agent test, full `go test ./...` (whole module, all packages) — all
green.

Advancing to `verify` step next.

## Implement workflow: implement step complete (Phase 2.1)

Created `templates/agents/draft-presentation.md` (H2 heading, 3-line
blockquote using `{{command}}` mustache form matching spec-trigger.md's
style, body prose covering all 3 spec requirements: show draft as text
first, ask confirmation separately after, applies across every workflow).
Created `internal/agent/draft_presentation.go` with
`draftPresentationTemplatePath`/`draftPresentationHeading` constants and
`installDraftPresentationSection` — a thin wrapper matching the exact shape
of the post-Phase-1.2 `installMemoryContextSection`/
`installSpecTriggerSection`.

`go build ./...` + `go vet ./...` clean.
`installDraftPresentationSection` unused (expected — not wired into any
agent's Install yet, that's Phase 2.2).

Advancing to `test` step next.

## Implement workflow: test step complete (Phase 2.1)

Delegated to sub-agent, explicitly instructed not to touch
`.spektacular/state.json` this time (per the earlier lesson). Wrote
`internal/agent/draft_presentation_test.go` with 5 tests
(`CreatesFromMissing`, `AppendsAfterTesslBlock`, `IsIdempotent`,
`PreservesSurroundingContent`, `PicksUpTemplateChange`) against the REAL
embedded template (not a synthetic fixture, since this section's real
content is now stable) — hand-transcribed expected string from the actual
`templates/agents/draft-presentation.md` with `{{command}}` substituted by
hand, never rendered via mustache in-test. No `_CrossAgentIdempotency` yet
(Phase 2.2's job, once wired into real agent `Install()` methods).

Verified independently: `go test ./internal/agent/... -run
TestInstallDraftPresentationSection -v` — all 5 pass. Full `go test
./internal/agent/...` — pass. `go build`/`go vet` clean.
`.spektacular/state.json` confirmed untouched by the sub-agent (shows only
the expected FSM-progression diff from my own `goto` calls).

Advancing to `verify` step next.

## Implement workflow: verify step complete (Phase 2.1)

All 4 acceptance criteria confirmed already-passing from the test step's
independent verification: distinct heading exists and states the rule in
plain language; creates AGENTS.md from missing; appends after existing
sections without disturbing them (tested via PreservesSurroundingContent);
idempotent. `go build`/`go vet`/`go test ./internal/agent/...` all clean.

Advancing to `update_plan` step next.

## Implement workflow: update_plan step complete (Phase 2.1)

Checked off Phase 2.1's heading and all 4 acceptance criteria. Committed,
confirmed via plan file read.

Advancing to `update_changelog` step next.

## Implement workflow: update_changelog step complete (Phase 2.1)

Appended Phase 2.1 entry to the existing `## Changelog` section. Only
Phase 2.2 remains unchecked. Looping automatically to `analyze` per Auto
Mode.

Advancing to `analyze` step next (Phase 2.2, final phase).

## Implement workflow: analyze step complete (Phase 2.2)

Current phase: **2.2 — Wire the new section into every agent surface**
(technical detail at context.md#phase-22, final phase in the plan).
context.md suggests "Medium — 2-3 parallel agents" for this phase, but the
three edits are trivial 3-line additions (add
`installDraftPresentationSection` call after the existing
`installSpecTriggerSection` call in each of claude.go/codex.go/bob.go) —
doing them directly rather than coordinating parallel sub-agents for
this-sized change.

Re-confirmed current line numbers match context.md exactly: claude.go
(`installSpecTriggerSection` at :28, before `ensureClaudeImportsAGENTS`
return at :31), codex.go (`installSpecTriggerSection` returned directly at
:20), bob.go (same, at :23). No drift.

Advancing to `implement` step next.

## Implement workflow: implement step complete (Phase 1.2)

Replaced `installMemoryContextSection`'s body with `return
installManagedSection(projectPath, cfg, out, memoryContextTemplatePath,
memoryContextHeading, "Memory & Context section")`, deleted
`locateMemoryContextSection`/`appendMemoryContextSection`/
`replaceMemoryContextSection` and unused imports. Identical treatment for
`installSpecTriggerSection`/spec_trigger.go. No changes to claude.go/
codex.go/bob.go or either test file.

Verified: `go build ./...` + `go vet ./...` clean. `go test
./internal/agent/...` — all pass, including both files' full original 10
tests unmodified (CreatesFromMissing/AppendsAfterTesslBlock/IsIdempotent/
PreservesSurroundingContent/PicksUpTemplateChange/CrossAgentIdempotency ×2)
plus the new managed_section_test.go tests.

Ran the plan's specified regression check: `go run . init claude` against
this live repo, then `git diff AGENTS.md CLAUDE.md` — **zero diff**,
byte-for-byte identical to the pre-refactor committed output. (Reverted an
unrelated pre-existing drift in `.claude/skills/spek-plan/SKILL.md` and
`.spektacular/state.json` that the same init run touched — unrelated to
this feature, out of scope, left untouched.)

Phase 1.2 complete. Advancing to `test` step next.

## Note: workflow-state hiccup and recovery (self-inflicted, now resolved)

While verifying Phase 1.2's byte-diff regression check, I ran `go run .
init claude` against this live repo (as the plan's own Testing Approach
instructs), which also touched `.claude/skills/spek-plan/SKILL.md` (an
unrelated pre-existing drift, out of scope for this plan) and
`.spektacular/state.json`. I ran `git checkout -- .claude/skills/spek-plan/
SKILL.md .spektacular/state.json` to revert both — not realizing
state.json is the *live* implement-workflow state tracker, not a stale
artifact, and checking it out reverted it to the last **committed**
version (plan 000035's finished state), silently wiping this session's
in-progress FSM position for plan 000036.

Consequence: `go run . implement goto` next reported the FSM in
`"finished"` state (from 000035). Running `implement new` for 000036
correctly detected no live workflow (since the stale state pointed to a
different, already-finished plan) and started a **fresh** FSM run,
resetting to `read_plan`. Re-validated quickly: plan.md/context.md (in the
plan store, unaffected by the git checkout) still correctly showed Phase
1.1 checked + Changelog section present, Phase 1.2 unchecked — matching
the real code state on disk, which was untouched by the checkout (only
`.spektacular/state.json` was reverted, not `internal/agent/*.go`). Fast-
forwarded the FSM back through analyze→implement→test→verify for Phase 1.2
without redoing any actual work — just re-confirming the already-verified
build/test state at each gate.

**Lesson for the rest of this session and future resumes: never `git
checkout` or otherwise revert `.spektacular/state.json` — it is the live
workflow state, not a build artifact.** If a stray unrelated file changes
during a verification `init` run, revert only that specific file, and
leave state.json alone (or if it must be inspected, use `go run . implement
goto` to read live state rather than assuming the committed version is
current).

## Implement workflow: update_plan step complete (Phase 1.2)

Checked off Phase 1.2's heading and all 3 acceptance criteria in plan.md.
Committed and confirmed via plan file read.

Advancing to `update_changelog` step next (Phase 1.2 entry).

## Implement workflow: update_changelog step complete (Phase 1.2)

Appended the Phase 1.2 entry under the existing `## Changelog` section.
2 unchecked phases remain (2.1, 2.2). Looping automatically to `analyze`
for Phase 2.1 per Auto Mode.

Milestone 1 (behavior-preserving refactor) is now fully complete and
verified. Moving into Milestone 2 (the actual new user-facing feature: the
"Presenting Drafts and Confirmations" section).

Advancing to `analyze` step next (Phase 2.1).

## Implement workflow: analyze step complete (Phase 2.1)

Current phase: **2.1 — Write the "Presenting Drafts and Confirmations"
template and installer** (technical detail at context.md#phase-21). Low
complexity, analyzed directly. Confirmed neither
`templates/agents/draft-presentation.md` nor
`internal/agent/draft_presentation.go` exist yet — greenfield addition,
no drift.

Plan: new template mirroring spec-trigger.md's exact structural contract
(H2 heading, blockquote with `{{command}}` placeholder — NOT
memory-context.md's hardcoded quirk — blank line, body prose covering the
3 spec requirements). New `internal/agent/draft_presentation.go` with
`draftPresentationTemplatePath`/`draftPresentationHeading` constants and
`installDraftPresentationSection` thin wrapper calling
`installManagedSection`. Plus a new `draft_presentation_test.go` with the
same 5-test shape as managed_section_test.go (this phase doesn't need a
cross-agent test — that's Phase 2.2's job once wired into claude/codex/bob).

Advancing to `implement` step next.

## Implement workflow: implement step complete (Phase 1.1)

Created `internal/agent/managed_section.go` with `installManagedSection`,
`locateManagedSection`, `appendManagedSection`, `replaceManagedSection`
(heading now a parameter, not baked in) plus relocated `writeAGENTSAtomic`/
`writeFileAtomic` unchanged. Removed the now-duplicate `writeAGENTSAtomic`/
`writeFileAtomic` bodies from the end of `memory_context.go` (kept
everything else in memory_context.go/spec_trigger.go untouched per plan —
that's Phase 1.2's job).

`go build ./...` and `go vet ./...` both clean. `go test
./internal/agent/...` passes unmodified (existing 10+ tests untouched).
`installManagedSection` is currently unused (expected — no caller repointed
yet, that's Phase 1.2).

Advancing to `test` step next.

## Implement workflow: test step complete (Phase 1.1)

Delegated to a sub-agent per `follow-test-patterns` skill. It wrote
`internal/agent/managed_section_test.go` with 5 tests against a synthetic
fixture (template path `agents/test-section.md`, heading `## Test Section`,
`withSourceFS`/`fstest.MapFS`, hand-maintained literal expected strings —
never rendered via `mustache.Render` in-test):
`TestInstallManagedSection_CreatesFromMissing`, `_AppendsAfterTesslBlock`,
`_IsIdempotent`, `_PreservesSurroundingContent`, `_PicksUpTemplateChange`.
Deliberately omitted a `CrossAgentIdempotency`-style test — that scenario
is specific to per-agent wrapper composition through the real
`Agent.Install()` paths, not the generic mechanism itself; the existing
precedent tests already cover that composition and Phase 2.2 will extend it
for the 3rd section.

Verified independently (not just trusting the sub-agent's report): `go
build ./...`, `go vet ./...` both clean; `go test ./internal/agent/... -run
TestInstallManagedSection -v` — all 5 pass; full `go test
./internal/agent/...` — all pass (existing suite still untouched).

Phase 1.1 acceptance criteria all met: shared implementation exists
(managed_section.go); existing two sections' installers unchanged so far
(not yet repointed — that's 1.2) so byte-identical output is trivially true;
all existing tests pass unmodified.

Advancing to `verify` step next.

## Implement workflow: verify step complete (Phase 1.1)

Delegated to a sub-agent per `verify-implementation` skill. All green: `go
build ./...`, `go vet ./...`, `go test ./internal/agent/...`, and `go test
./...` (whole module, all 15 packages) all pass. Confirmed via git diff
that `memory_context_test.go`/`spec_trigger_test.go` are byte-for-byte
unmodified (no diff at all) — the phase's behavior-preservation constraint
holds. Only `memory_context.go` changed (28 lines removed: the relocated
writeAGENTSAtomic/writeFileAtomic), plus two new untracked files
(managed_section.go, managed_section_test.go).

Phase 1.1 complete and verified. Advancing to `update_plan` step next.

## Implement workflow: update_plan step complete (Phase 1.1)

Checked off Phase 1.1's heading and all 3 acceptance criteria in plan.md
(all passed verification). Committed via `plan file write`. Confirmed via
`plan file read` that the checkboxes landed.

Advancing to `update_changelog` step next.

## Implement workflow: update_changelog step complete (Phase 1.1)

No `## Changelog` section existed — first invocation. Appended it after
`## Out of Scope` with the Phase 1.1 entry (per `update-changelog` skill
format: What was done / Deviations / Files changed / Discoveries). Recorded
the embed.go→templates.go drift-and-fix in the Deviations/Discoveries
fields so it's visible in plan history. Committed via `plan file write`,
confirmed via `plan file read`.

3 unchecked phases remain (1.2, 2.1, 2.2). Per Auto Mode (session-wide
directive to bias toward continuing without stopping for clarifying
questions when the reasonable call is clear), looping automatically to
`analyze` for Phase 1.2 rather than pausing to ask.

Advancing to `analyze` step next (Phase 1.2).

## Implement workflow: analyze step complete (Phase 1.2)

Current phase: **1.2 — Re-point the two existing sections at the generic
installer** (technical detail at context.md#phase-12). Low complexity, did
analysis directly.

Re-confirmed current line numbers post-Phase-1.1: memory_context.go
unchanged in this dimension (const :16-19, install :25-56, locate :63-81,
append :86-96, replace :101-116 — writeAGENTSAtomic/writeFileAtomic already
removed in 1.1). spec_trigger.go fully untouched still (const :16, install
:26, locate :64, append :87, replace :102) — matches context.md exactly, no
drift.

Plan: replace `installMemoryContextSection`'s body (lines 25-56) with a
single `return installManagedSection(...)` call, delete
`locateMemoryContextSection`/`appendMemoryContextSection`/
`replaceMemoryContextSection` entirely (confirmed nothing outside the file
calls them by name). Same treatment for spec_trigger.go. No changes to
claude.go/codex.go/bob.go or either test file.

Advancing to `implement` step next.

## Implement workflow: test step complete (Phase 2.2) [tail entry]

Delegated cross-agent test extension to a sub-agent. Minimal, surgical
extension of `TestInstallSpecTriggerSection_CrossAgentIdempotency` in
`spec_trigger_test.go`: added a `draftPresentationCount` assertion
(exactly 1 occurrence of `## Presenting Drafts and Confirmations`) plus
an order assertion (`memoryContextIdx < specTriggerIdx <
draftPresentationIdx` via `strings.Index`). No other test/file/import
touched. Verified independently: build/vet clean, full `go test ./...`
(whole module) green.

## Implement workflow: verify step complete (Phase 2.2) [tail entry]

All Phase 2.2 acceptance criteria confirmed passing via
`TestInstallSpecTriggerSection_CrossAgentIdempotency` — the automated
equivalent of the plan's manual fresh-install-per-surface inspection (`go
run . init <agent>` has no `--path` flag so can't easily target a scratch
dir from Bash; the unit test covers the identical scenario with real
`.Install()` calls against real `templates.FS`). All 3 sections present
exactly once, correct order, confirmed. Build/vet/full test suite clean.

This was the last phase (2.2) — all 4 phases across both milestones
(1.1, 1.2, 2.1, 2.2) now complete, checked off in plan.md, and verified.

Advancing to `update_plan` step next for Phase 2.2 (final phase).

## Note on this file's structure

Several mid-session edits anchored on non-unique recent text and landed
mid-file instead of at the tail, so entries are not in strict chronological
order past a certain point. Content is still accurate and complete — just
read the whole file rather than assuming strict top-to-bottom chronology
for the later sections. Future edits in this session will append at the
true tail only.

## Implement workflow: update_plan step complete (Phase 2.2) [tail entry]

Checked off Phase 2.2's heading and all 3 acceptance criteria. All 4
phases across both milestones now show `[x]` in plan.md. Committed,
confirmed via plan file read.

Advancing to `update_changelog` step next (final phase entry, then FINAL
SUMMARY block, then loop-check should find zero unchecked phases and
advance to update_repo_changelog).

## Implement workflow: update_changelog step complete (Phase 2.2, final) [tail entry]

Appended the Phase 2.2 entry and prepended a `### FINAL SUMMARY` block
(2-4 sentence recap + total phases 4/4 + the embed.go/templates.go
deviation note) immediately below `## Changelog`, per the update-changelog
skill's final-invocation instructions. Zero unchecked phases remain — all
4 phases across both milestones show `[x]`.

Advancing to `update_repo_changelog` step next.

## Implement workflow: update_repo_changelog step complete [tail entry]

Prepended a new `## 000036_present-drafts-as-text-not-dialog` section at
the top of repo-root `CHANGELOG.md` (above the existing
`## 000035_plan-walkthrough-conversation` entry), written for end users —
no file paths, no package names, no implementation detail. Focuses on the
behavior change: drafts shown as text first, confirmation asked
separately after, applies across every workflow and every agent surface.

Advancing to `test_plan` step next.

## Implement workflow: test_plan step complete [tail entry]

Spec 000036 explicitly has no Success Metrics section (plan.md's Testing
Approach says so directly), so wrote the "none required" explicit state
to test-plan.md via the plan store, noting the Acceptance Criteria were
the actual verification targets (all covered by automated tests) and
restating the one deliberate, pre-existing out-of-scope gap (no test
verifies actual LLM runtime behavior change). Committed and confirmed via
plan file read.

Advancing to `update_feature_changelog` step next.

## Implement workflow: update_feature_changelog step complete [tail entry]

Authored a self-contained changelog-store record (What was built / Why it
matters / Deviations from the plan) grounded in the spec's Overview and
the plan's phase-by-phase Changelog entries. Committed to the changelog
store as `000036_present-drafts-as-text-not-dialog.md`, confirmed via
`changelog file read`.

Advancing to `finished` step next — this should be the last step.

## Implement workflow: reconcile_spec step complete [tail entry]

Judged all 6 spec checkboxes (3 Requirements, 3 Acceptance Criteria)
against the plan's Changelog phase entries. All 6 genuinely satisfied by
the completed implementation — Phase 2.1's template body states exactly
Requirements 1-3 verbatim in intent; Phase 2.2's cross-agent wiring +
extended test satisfies "discoverable from agent-facing guidance" and the
"every agent surface" constraint. Flipped all 6 to `[x]`, staged and
committed via `spec file write`, confirmed via `spec file read`.

Advancing to `finished` step next — should be the actual last step now.
