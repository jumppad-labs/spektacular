---
created_date: "2026-08-14"
status: completed
closed_date: "2026-08-14"
---

# Plan: 000043_flipped-interaction-spec-interview

<!-- Metadata -->
<!-- Created: 2026-08-14T08:55:47Z -->
<!-- Commit: c853af8 -->
<!-- Branch: main -->
<!-- Repository: git@github.com:jumppad-labs/spektacular.git -->

## Overview

Spektacular's spec workflow gathers requirements through a fixed script: each of its seven section steps asks its own small set of scripted questions, one section at a time. This plan replaces that with a single, adaptive interview that runs before any section is drafted, following the Flipped Interaction pattern: the agent asks open questions toward understanding the feature, adapting each one to what has already been said, and stops once no further question would change what gets drafted. Every section step then drafts its content from the interview and presents it for the user to confirm or correct, rather than asking from a blank prompt. Rejecting a draft opens a short follow-up conversation to understand why before anything changes, and a correction can update a different, already-confirmed section without requiring the user to separately revisit it — the existing end-of-workflow review remains the single point everything is checked together. Documentation on the Spektacular site names the pattern, walks through the new interview phase, and shows a concrete multi-repo example, positioned as a differentiator rather than a buried implementation detail. People writing specs for features they haven't fully thought through yet get a spec that reflects what they actually need.

## Conventions

- **No em dashes** — applies to all authored prose this plan produces (plan.md, context.md, research.md) and to any new step-template instruction text or documentation copy drafted for the docs site pages.
- **Plans must sketch content structure, not just summarize it** — the phase(s) covering the new `how-it-works.mdx` interview description and the `index.mdx` differentiator card must include a concrete Content outline/example (headings in order, an illustrative example per section) rather than only a prose summary, since this plan introduces genuinely new page content in the docs repo.
- **Prefer DRY refactors over accepting duplication** (feedback memory, not a knowledge-base convention, but load-bearing) — directly drove the architecture's decision to relocate `repoRoster`/`repoRosterExtra` to shared homes (`internal/repo`, `internal/stepkit`) rather than call across from `cmd/spec.go` into a still plan-named function in `cmd/plan.go`.
- **Alternate section background shading** — does not apply: the docs work here lands inside the existing `PipelineStage`/`Fragment slot="body"` content of `how-it-works.mdx` and inside the existing `FeaturesGrid` on `index.mdx`; no new top-level `Section`-style component with a `surface` prop is introduced.
- **MDX authoring conventions (ConfigKey pattern, no layout HTML in page bodies, blank lines around slot content, fenced code blocks)** — applies loosely but is not the primary pattern here: this plan's docs changes are prose edits inside existing `Fragment slot="body"`/`slot="example"` blocks and a new `FeatureCard` inside the existing `FeaturesGrid`, not new `ConfigKey`-style reference content, so the relevant sub-rules are blank-line-around-slot-content and fenced-code-for-any-mock-exchange, not the full ConfigKey pattern.

## Architecture & Design Decisions

The spec workflow's interaction model changes from "ask a scripted question per section" to "interview once, up front, then draft every section from what the interview and any later corrections establish" — implemented, like the plan workflow's own recent 000040_plan-draft-then-walkthrough rewrite, almost entirely as instruction-prose edits to existing step templates plus one new FSM step. A new `interview` step is inserted into `internal/steps/spec/steps.go`'s `Steps()` between `new` and `overview` (`new`'s destination/next_step retargets from `overview` to `interview`; `interview`'s destination/next_step is `overview`; every step from `overview` through `finished` keeps its existing Src/Dst and relative order unchanged). The step's callback is a one-line `writeStep("interview", "overview", "steps/spec/00b-interview.md", ...)` matching the shape every existing gathering-step callback already uses (research.md § Chosen approach — evidence, `internal/steps/spec/steps.go:101-141`). The interview's instructions implement the Flipped Interaction pattern (White et al., arXiv:2302.11382, cited directly in the spec's Technical Approach): the agent asks open, adaptive questions toward the stated goal of "enough understanding to draft a credible first pass," decides for itself each turn whether material ambiguity remains, and stops once it does not, never enumerating every possible requirement. Findings are synthesized (not transcribed verbatim) into a new working file, `.spektacular/work/{{spec_name}}/interview.md`, following the exact per-section working-file convention every gathering step already uses (`templates/work_files_test.go:23-36`), and it is included in the verification step's existing `rm -rf .spektacular/work/{{spec_name}}` cleanup as one more file in that directory, with no new cleanup mechanism.

The seven section-gathering templates (01-overview.md through 07-non_goals.md) each lose only their opening "Ask the user: ..." scripted question block and gain a "draft this section's content from interview.md, plus any prior corrections recorded against it, and present it to the user to confirm or correct" instruction in its place. Every other rule in these templates is untouched: the WHAT-not-HOW test, the stakeholder-readability rule, the checklist-format guidance, the working-file write instruction, and the context-directive footer all stay exactly as they are today, because none of them are about how the content is elicited, only the opening beat changes, mirroring how 000040 rewrote the plan workflow's gathering-step closing beat as a targeted edit rather than a restructure. Rejection handling and cross-section amendment are new shared instruction prose added to the same closing beat of all eight steps (interview plus the seven section steps, since a rejection can happen against interview-derived content on the very first section too): when the user indicates a drafted section is wrong, the agent asks a follow-up question to understand why, holding open both hypotheses (an undersurfaced need, or a genuine misunderstanding), before changing anything, and any resulting edits are applied directly to the owning working file(s), which may include a working file for a section other than the one currently under review, with no fresh reconfirmation gate for that other section. This is not a new mechanism invented for this feature; it is the existing verification-step step-3/4 triage loop (`templates/steps/spec/08-verification.md:75-93`) lifted out of being a one-time, end-of-workflow-only pattern and generalized into prose every section step carries throughout the workflow. The existing verification step itself is unchanged in mechanism: it remains the single safety net that presents everything, including anything amended mid-session underneath the user, satisfying the spec's constraint that an amended earlier section needs no separate in-the-moment reconfirmation.

Cross-repo awareness reuses the plan workflow's existing `repoRoster` mechanism (research.md § Chosen approach — evidence, full call chain `cmd/plan.go:133,234` to `internal/steps/plan/steps.go:100-106` `repoRosterExtra` to `templates/steps/plan/02-discovery.md:15-24`'s `{{#repos}}` block), rather than building a second way for a step to learn what repos a project spans. Rather than reaching from `cmd/spec.go` across into a function still named and reasoned about as plan-specific in `cmd/plan.go`, which would work today but leaves `repoRoster`'s real home misattributed to one caller, `repoRoster` moves to `internal/repo` (as `repo.Roster(cfg, root, git) []map[string]any`), the package that already owns `Set` and `DescriptiveMetadata` and already imports `internal/config`; the six-line pull-back helper `repoRosterExtra` moves to `internal/stepkit`, shared by every step package, no new dependency, matching where `GetString`/`RenderTemplate`/`WriteStepResult` already live. `cmd/plan.go` and the new `cmd/spec.go` wiring both then call the same `repo.Roster(...)` and `stepkit.RepoRosterExtra(data)` symbols with no behavior change to the plan workflow. This directly satisfies **Prefer DRY refactors over accepting duplication**: the alternative of leaving `repoRoster` in `cmd/plan.go` and calling it from `cmd/spec.go` unchanged was rejected because, while it would work, it is the kind of cross-package reach-through that reads as duplication-in-waiting the moment a third workflow needs the same roster, and the relocation costs only two mechanical moves plus updated call sites, not a redesign. The new `interview` step calls `stepkit.RepoRosterExtra(data)` as its `writeStep` `Extra` argument, exactly as `discovery()`/`architecture()` already do in the plan workflow, and the interview template renders the identical `{{#repos}}...{{/repos}}` block (with the same `{{^repos}}` empty-registry fallback sentence) so a project with more than one registered repo sees its full roster, by name, role, and description, before the interview asks its first question.

This direction beats two rejected alternatives (research.md § Alternatives considered and rejected has the full evidence): folding the interview into the existing `new` step was rejected because `new` already has one well-defined job (scaffold plus free-form context capture, with its own documented exemption from the generic context-directive test) and because a dedicated FSM step is what makes the interview resumable as its own state, exactly as 000040's `walkthrough` step gained resumability by being its own step rather than folded into an existing one; and adding a fresh per-section reconfirmation gate for every cross-section amendment was rejected per the spec's own explicit constraint, in favor of relying on the existing verification-step walkthrough as the single safety net, the same trade-off 000040 already made and validated for the plan workflow's `walkthrough` step.

## Component Breakdown

**`interview` step (new)** — the spec workflow's first user-facing step (immediately after `new`), owning the entire open-ended Flipped Interaction conversation: asking adaptive questions toward "enough understanding to draft a credible first pass," judging each turn whether material ambiguity remains, and stopping once it does not. Also owns surfacing the project's registered-repo roster (via the shared repo-roster mechanism below) and asking a cross-cutting-impact question when a feature reads as primarily about one repo in a multi-repo project. Synthesizes its findings into a new working file, `interview.md`, which becomes the primary source material every downstream section-drafting step reads from. Relates to the section-drafting steps as their upstream input; relates to the rejection-repair behavior (below) as one of the eight steps that shares it.

**Section-drafting steps (changed)** — the spec workflow's seven existing gathering steps (overview, requirements, acceptance_criteria, constraints, technical_approach, success_metrics, non_goals). Each stops asking its own scripted question and instead drafts its section's content from `interview.md` (and any prior corrections layered onto its own working file by the rejection-repair behavior), presents the draft to the user, and asks for confirmation or correction. Each still owns its existing section-specific rules (WHAT-not-HOW test, checklist formatting, stakeholder-readability) and its own working file, unchanged. Depends on the interview step's output as its primary source; participates in rejection-repair and cross-section amendment as both the section under review and, when amended as a side effect of a different section's correction, the target of an update it did not initiate.

**Rejection-repair behavior (new, shared instruction pattern)** — not a new step or new Go mechanism, but a shared closing-beat instruction added to the interview step and all seven section-drafting steps: when the user indicates a drafted section is wrong, ask a follow-up question to understand why before changing anything, holding open both the "undersurfaced need" and "genuine misunderstanding" hypotheses. Owns turning a single rejection into however many resulting changes are actually warranted (zero, one, or a cluster), across however many sections those changes belong to. Generalizes the existing verification-step triage loop (unchanged below) rather than introducing a new mechanism; every step that carries it writes its edits directly to the owning section's working file, including sections other than the one currently under review.

**Verification step (unchanged in mechanism, now one instance of a pattern used throughout)** — the existing fresh-eyes-review-then-triage step remains exactly as it is today: assemble from working files, spawn an independent reviewer, triage findings with the user, apply confirmed fixes to owning working files, commit. It continues to be the single end-of-workflow safety net that presents everything, including any section amended mid-session by rejection-repair on a later step, satisfying the "no separate reconfirmation gate for an amended earlier section" constraint without needing to know anything changed underneath it.

**Repo-roster projection (relocated, shared)** — the existing `repoRoster`/`repoRosterExtra` mechanism (today living in `cmd/plan.go` and `internal/steps/plan/steps.go`, used by the plan workflow's discovery/architecture steps) relocates to shared homes it does not yet have: the roster-building function moves into `internal/repo` (alongside `Set`/`DescriptiveMetadata`, which it already depends on), and the workflow-data pull-back helper moves into `internal/stepkit` (alongside the other shared step-callback helpers). Both the plan workflow and the new spec-workflow `interview` step call the same relocated symbols; plan-workflow behavior and output are unchanged. Owns projecting each registered repo's name, role, and description into a step's rendered instruction, refreshed from config on every workflow invocation.

**`spec goto`/`spec new` command handlers (changed)** — `cmd/spec.go`'s `runSpecNew` and `runSpecGoto` gain one call each, `wf.SetData("repos", repo.Roster(cfg, root, git))`, mirroring the existing call sites in `cmd/plan.go` exactly. No other change to command-layer behavior, flags, or output shape.

**`internal/steps/spec` step table (changed)** — the ordered step definitions gain the new `interview` entry and retarget `new`'s destination. A mechanical, precedented insertion of the same shape 000040 already used for `walkthrough` in the plan workflow's own step table; the workflow engine itself (`internal/workflow`, `looplab/fsm`-backed) is untouched.

**`spek-new` skill instructions (changed)** — the agent-facing driver of the spec-workflow step loop. Reframed to describe the interview as the first step after `new`, explains the interview's stopping condition and repo-roster awareness, and covers the rejection-repair/cross-section-amendment behavior as a standing instruction that applies from the interview step onward, not only at verification.

**Documentation pages (changed, docs repo)** — `how-it-works.mdx`'s Stage 1 body gains a description of the interview phase happening before any section is drafted, with an example question, and names the Flipped Interaction pattern with its source attribution. `index.mdx`'s existing `FeaturesGrid` gains a card (or a reframed existing card) positioning this behavior as a differentiator. A new worked example, reusing the existing `SpecFormat`-style `slot="example"` fenced-block pattern, demonstrates the interview asking a cross-repo question in a multi-repo project. No new Astro components are introduced; both pages reuse components and slot patterns already in use elsewhere on the site.

## Data Structures & Interfaces

No new Go types, interfaces, or serialization formats are introduced. The feature reuses every existing contract unchanged and adds one lightweight markdown working-file convention plus two relocated (not redesigned) function signatures.

**FSM step registration (changed contract instance, existing type)** — the spec workflow's step table gains one entry of the existing `workflow.StepConfig` shape, and `new`'s destination retargets:

```go
{Name: "new",       Src: []string{"start"}, Dst: "new",       Callback: new()},
{Name: "interview", Src: []string{"new"},   Dst: "interview", Callback: interview()},
{Name: "overview",  Src: []string{"interview"}, Dst: "overview", Callback: overview()},
// ...unchanged from here through "finished"
```

`interview()` follows the same one-line `writeStep(...)` callback shape as every existing gathering step, with `stepkit.RepoRosterExtra(data)` passed as its `Extra` argument (see below). No change to `StepConfig`, the FSM engine, `Data`, or the render pipeline.

**Relocated repo-roster functions (existing signatures, new package homes)** — these are moves, not redesigns; call signatures and return shapes are unchanged from their current form in `cmd/plan.go` / `internal/steps/plan/steps.go`:

```go
// internal/repo (new home; was cmd/plan.go's unexported repoRoster)
func Roster(cfg config.Config, root string, git GitRunner) []map[string]any

// internal/stepkit (new home; was internal/steps/plan's unexported repoRosterExtra)
func RepoRosterExtra(data workflow.Data) map[string]any
```

`Roster`'s returned `[]map[string]any` shape is unchanged: one map per registered repo with `name`, `description`, `role`, `tags` (comma-joined string), `deployment` keys. `RepoRosterExtra`'s behavior is unchanged: reads the `"repos"` key out of workflow `Data` (set by the command handler on every invocation) and wraps it as `{"repos": repos}` for use as a step's `Extra` map, returning `nil` when the key is absent so a step's template can render its own empty-registry fallback via `{{^repos}}`.

**Interview working file (new markdown convention, not a schema)** — `.spektacular/work/{{spec_name}}/interview.md` holds the interview's synthesized findings in prose, in the same category as every existing per-section working file (`overview.md`, `requirements.md`, etc.). No Go code parses it; downstream section-drafting steps read it with the agent's own file tools, exactly as they already read `.spektacular/context.md` and their own prior working files today. No fixed internal schema is imposed beyond being readable prose a drafting step can work from, unlike the plan workflow's `assumptions.md` convention (which does impose a fixed per-entry shape), the interview file has no equivalent structural contract because its consumers are drafting prompts, not a fold-into-a-document assembly step reading discrete entries.

**Existing contracts relied on, unchanged**: the `spec file read/write <name>.md [--from]` CLI surface (verification-step commit path, unaffected), the persisted workflow state shape and its resume semantics (an interrupted interview resumes on the `interview` step exactly as any other step resumes today), the mustache template variables already supplied to every spec step (`spec_path`, `spec_name`, `step`, `title`, `next_step`, `config.command`), and the `workflow.StepCallback`/`workflow.Data`/`workflow.ResultWriter`/`store.Store` signatures every step callback already uses. The final spec document's shape, name, and location are unchanged.

## Implementation Detail

This feature extends the codebase's established split — the Go layer owns step sequencing, state persistence, and document assembly; instruction prose owns judgement, conversation, and content — without introducing any new mechanism on either side. Everything user-visible about the change (the adaptive interview, draft-and-confirm sections, rejection-repair, cross-section amendment) is expressed as template prose; the only Go changes are one mechanical step insertion of the same shape every existing step already has, plus two function relocations that change no behavior.

**The interview is a rewrite of the workflow's opening beat, not a new interaction primitive.** The spec workflow already asks the user things and already writes what it learns to working files; what changes is when the asking happens (once, up front, adaptively) and what the section steps do with it (draft-and-confirm instead of ask-from-scratch). A developer reading the new `interview` step template sees the same three-part shape every gathering-step template already has, an instruction body, a working-file write instruction, a context-directive footer, with the instruction body implementing the Flipped Interaction pattern's three moves (state the goal, ask adaptively, stop on an explicit condition) instead of a scripted question list. No new template-rendering capability, mustache helper, or step-callback pattern is needed.

**De-scripting each section step is a rewrite of its opening beat, not a restructure.** Each gathering template today opens with "Ask the user: [scripted question]." Under the new model that opening becomes "Draft this section from `interview.md` (and any working file amended by an earlier correction), present it, and ask the user to confirm or correct it," a rewrite of the same paragraph position, not a reshuffling of the template's later sections (working-file rules, format guidance, working-file write instruction, context-directive footer all keep their current position and wording). A developer diffing these seven templates sees exactly one paragraph change per file and nothing else.

**Rejection-repair and cross-section amendment are a second, identically-shaped rewrite applied to the closing beat of every gathering step, generalizing an existing pattern rather than inventing one.** The verification step's step-3/4 triage loop already does this, propose a fix, wait for confirmation, write it to the owning section's working file, no fresh gate for that section, for findings surfaced by its fresh-eyes review. The same shape is added, near-verbatim, to the closing beat of the interview step and all seven section steps, so it fires whenever the user (not just the automated reviewer) rejects a draft, at any point in the workflow, not only at the very end. This is the one place where "propagates to another section's working file" needs explicit statement in the prose: a correction surfaced while reviewing section N can write to section M's `.spektacular/work/{{spec_name}}/<M>.md` directly, with no FSM transition back to M's step and no new confirmation gate; M's already-passed state is simply superseded in its working file, and the existing end-of-workflow verification step is what eventually surfaces the change for review.

**The repo-roster relocation is a pure move, verified by keeping call-site behavior identical.** `repoRoster`'s body and `repoRosterExtra`'s body do not change; only their package and (for `repoRoster`) exported name change, and their call sites in `cmd/plan.go` and `internal/steps/plan/steps.go` update to the new import paths. This is the same category of change as any Go refactor that moves a function without touching its logic, and the plan workflow's existing tests for repo-roster rendering (`internal/steps/plan/steps_test.go`'s `TestDiscovery...RendersRegisteredRepos` family) must continue passing unmodified in behavior, only updated for the new call path, which is itself a strong regression guard that the move introduced no behavior change.

**A developer reading the changed code sees familiar shapes everywhere.** The step table gains one entry identical in form to its neighbors (following the exact precedent 000040 set inserting `walkthrough` into the plan workflow's own step table); the new template sits in the existing per-step template directory, numbered into the sequence; the interview's working file is one more file in an existing directory contract; and the skill file's loop framing is extended rather than reshaped, describing one more step in the same fixed loop it already describes. New pattern introduced: only the interview working file's "synthesized findings, not transcript" convention and the "rejection-repair prose fires from any gathering step, not only verification" framing, both are prose conventions interpreted by agents, not new Go mechanisms, matching how 000040 introduced its own assumption-log convention as prose-only.

## Dependencies

- **`internal/steps/spec` (existing, changed)** — the spec workflow's step table and callbacks; gains the `interview` step entry and callback, retargets `new`'s destination. The primary Go package modified.
- **`internal/repo` (existing, changed)** — gains the relocated `Roster` function (moved from `cmd/plan.go`'s unexported `repoRoster`), reusing the package's existing `Set`/`DescriptiveMetadata` machinery unchanged.
- **`internal/stepkit` (existing, changed)** — gains the relocated `RepoRosterExtra` helper (moved from `internal/steps/plan`'s unexported `repoRosterExtra`), alongside its existing `GetString`/`RenderTemplate`/`WriteStepResult` helpers.
- **`cmd` — `spec.go` and `plan.go` (existing, changed)** — `runSpecNew`/`runSpecGoto` gain one `wf.SetData("repos", repo.Roster(cfg, root, git))` call each, mirroring the existing call sites; `plan.go`'s call sites update their import path to the relocated `repo.Roster` but are otherwise unchanged.
- **`internal/steps/plan` (existing, changed only for the relocation)** — `discovery()`/`architecture()` update their call from the local `repoRosterExtra` to the relocated `stepkit.RepoRosterExtra`; no behavior change.
- **`internal/workflow` (existing, unchanged, relied upon)** — provides the FSM engine, state persistence, and resume semantics; the design depends on all of this staying exactly as-is, exactly as 000040 relied on it for the plan workflow's own step insertion.
- **`internal/metadata` (existing, unchanged)** — frontmatter lifecycle for the spec document; untouched by this feature.
- **`cmd` — `spec file` (existing, unchanged, reused)** — the store write path used for the verification step's commit and for the rejection-repair working-file updates; no new write mechanism.
- **`templates/steps/spec/` (existing, changed)** — `00-new.md` retargets its `next` frontmatter and `spec goto` call from `overview` to `interview`; a new `00b-interview.md` (or equivalently numbered) template is added; templates `01-overview.md` through `07-non_goals.md` each get their opening scripted-question paragraph rewritten and gain the shared rejection-repair/cross-section-amendment closing-beat prose; `08-verification.md`'s existing blanket working-directory cleanup already covers the new file, confirmed rather than changed.
- **`templates/steps/plan/02-discovery.md`, `03-architecture.md` (existing, unchanged in rendered output)** — continue rendering the same `{{#repos}}` block; only the Go-side symbol they're fed by changes name/package, not behavior.
- **`templates/skills/workflows/spek-new/SKILL.md` (existing, changed)** — the agent-facing driver of the spec-workflow step loop, updated to describe the interview step, its stopping condition, and the rejection-repair/cross-section-amendment behavior as standing instructions that apply from the interview step onward. Installed copies (e.g. `.claude/skills/spek-new/`) refresh via `init` as usual.
- **Template and step test suites (existing, changed)** — `internal/steps/spec/steps_test.go` (step order, FSM walk), `templates/context_directive_test.go` (new step's directive membership), `templates/work_files_test.go` (interview working file joins the working-file contract), `internal/steps/plan/steps_test.go` (repo-roster tests updated for the relocated call path, no behavior change expected), `cmd/spec_test.go` and `cmd/plan_test.go` (repo-roster wiring at the command layer), `tests/harbor/spec-workflow/` (canonical step order, `instruction.md`/`solve.sh` updated for the new step).
- **External libraries: none added or changed** — `looplab/fsm` and `cbroglie/mustache` are existing dependencies used as-is.
- **Prior plans/specs: none must land first** — the two structural precedents this plan follows (000040's step-insertion-plus-walkthrough recipe, and 000042's repo-descriptive-metadata split that `DescriptiveMetadata`/`repoRoster` already depend on) are already shipped; this plan builds on the code they left behind, not on pending work.
- **Docs repo (spektacular-website, changed)** — `src/pages/how-it-works.mdx` and `src/pages/index.mdx`; no new Astro components, no new dependencies; existing `PipelineStage`, `Fragment slot="body"`/`slot="example"`, `FeaturesGrid`/`FeatureCard` components are reused as-is.

## Testing Approach

The feature splits into a deterministic layer (FSM wiring, template content, the repo-roster relocation) covered by Go unit tests, and an agent-behavior layer (does the interview actually adapt and stop, does rejection-repair actually ask why, does a cross-section amendment actually land) covered by the existing end-to-end harness plus manual checks, mirroring how 000040_plan-draft-then-walkthrough verified the plan workflow's own interaction-model rewrite in this same codebase.

**Unit tests (Go, extending the existing step and template suites).** `internal/steps/spec/steps_test.go`'s `TestStepsOrderMatchesExpected` and `TestFSMWalkFromNewToFinished` are updated for the new canonical order, with `interview` reachable only from `new` and `overview` reachable only from `interview`. A new `TestInterviewStepRendersInstruction` mirrors the existing `TestOverviewStepRendersInstruction`. Template-content assertions, this codebase's established way of testing prose-driven behavior, are extended: `templates/context_directive_test.go` gains the interview template to its directive-bearing count (the default: the interview step carries the standard directive, is not exempt); `templates/work_files_test.go` gains an assertion that the interview template references `.spektacular/work/{{spec_name}}/interview.md`. A new assertion confirms every one of the eight steps that carry rejection-repair prose (interview plus the seven section steps) contains the shared marker text for that behavior, the same style of enforcement `context_directive_test.go` already uses for its own shared marker.

**Relocation regression tests (Go, existing suite, path updated not behavior).** `internal/steps/plan/steps_test.go`'s existing repo-roster tests continue to pass unmodified in assertion content after `repoRoster`/`repoRosterExtra` move to `internal/repo`/`internal/stepkit`, this is the direct regression guard that the relocation introduced no behavior change to the plan workflow. New unit tests for the relocated symbols themselves live in their new packages, and a parallel set of tests in `internal/steps/spec/steps_test.go` confirms the new `interview` step's repo-roster rendering and empty-registry fallback mirror the plan workflow's own tests for the same behavior.

**End-to-end (harbor spec-workflow suite, updated).** `tests/harbor/spec-workflow/tests/test_spec_workflow.py`'s `EXPECTED_STEP_ORDER` gains `"interview"` after `"new"`; a new `TestInterviewStep` class mirrors the existing per-step test shape but adapts its content check since the interview produces no scaffold section, asserting instead that the completed spec's Overview section reflects interview-derived content rather than a placeholder. `tests/harbor/spec-workflow/solution/solve.sh`'s scripted golden path gains a `spec goto interview` call to keep the FSM walk valid.

**Load-bearing guarantees, in plain language:** the interview step cannot be skipped on the way to overview (FSM wiring); a spec workflow run still produces all eight sections with meaningful content (existing per-section content-length assertions, unchanged); the repo-roster relocation changes no rendered template output for the plan workflow (existing plan-workflow repo-roster tests, relied on rather than re-proven); and every gathering step's template literally contains the rejection-repair marker text (template-content test, catching a silent drop of the shared prose during implementation).

**Deliberate gaps.** No unit test attempts to verify conversational quality, whether the interview's questions are actually well-chosen, whether it genuinely stops at the right moment, whether a rejection-repair conversation actually surfaces the right cluster of changes. This is not unit-testable prose judgement, consistent with 000040's own stated gap for its walkthrough step, and is covered by the E2E run's presence checks plus manual review of the actual first real usage. No new tests for the store write path, resume machinery, or the verification step's existing triage loop, all unchanged and already covered.

**Success metrics from the spec, each made verifiable:**

- *A user starting a spec for a feature they haven't fully thought through still ends up with a complete, well-formed spec, without the workflow producing a shallow draft based on a vague initial description* — **Behavioural test**: the end-to-end harness run asserts every section meets the existing minimum-content-length and no-placeholder-text checks, now produced from interview-derived drafts rather than scripted answers, on a task prompt that gives only a vague initial description. **Manual — captured in the implementation test plan**: whether the resulting spec is genuinely well-formed for a real vague-starting-point session, observed on the documentation spec this feature is built to be immediately exercised against.
- *The number of back-and-forth corrections needed during section review decreases over time, because the interview surfaced most of what mattered upfront* — **Manual — captured in the implementation test plan**: a longitudinal, cross-session observation not assertable from any single workflow run.
- *A correction to one section catching a related gap in another section is visible in practice, evidence that the cross-section amendment behavior is actually firing, not just theoretically possible* — **Behavioural test**: an end-to-end scenario deliberately scripts a rejection during a later section that should surface a needed addition to an earlier, already-drafted section, and asserts the earlier section's working file changed as a result, without a repeated confirmation step for that earlier section appearing in the transcript. **Manual**: a real-use instance of this actually happening unprompted is the qualitative confirmation the spec asks for.
- *Documentation of the pattern is something a prospective user would point to as a reason to choose Spektacular over a plainer spec-authoring tool* — **Manual — captured in the implementation test plan**: a documentation-quality and positioning judgement, verified by the docs-repo build/type checks confirming the new content builds correctly, plus human review of the finished page copy.

## Milestones & Phases

### Milestone 1: Every spec starts with an open-ended interview instead of a scripted first question

**What changes**: Starting a new spec no longer goes straight into a fixed set of scripted questions about the Overview section. Instead, the agent opens with an adaptive, open-ended conversation about the feature being built, who it is for, and what constraints apply, asking follow-up questions shaped by what the user has already said rather than working through a fixed list. In a project with more than one registered repository, the interview also surfaces the project's other repos and asks whether the feature has impact beyond the one it is primarily about. The interview ends, and drafting begins, once no further question would materially change what gets drafted, not once every conceivable detail has been asked about. This milestone changes only the very first exchange of a spec-creation session; what happens in the section steps that follow is unaffected until Milestone 2.

#### - [x] Phase 1.1: Insert the interview step into the spec workflow

**Repo:** spektacular

Add a new `interview` step to the spec workflow's step sequence, immediately after `new` and before `overview`, so every spec-creation session opens with the interview rather than the Overview section's scripted question. The `new` step's destination retargets to `interview`; every step from `overview` onward keeps its existing position and order unchanged. The new step's template carries the Flipped Interaction pattern's structure: state the goal (enough understanding to draft a credible first pass), ask adaptive questions toward it, and stop once no question remains that would materially change the draft. Findings are saved to a new per-workflow working file that later steps will read from.

*Technical detail:* [context.md#phase-11](./context.md#phase-11-insert-the-interview-step-into-the-spec-workflow)

**Acceptance criteria**:

- [x] The spec workflow's step sequence includes an interview step immediately after `new` and before `overview`, and `overview` can only be reached from it
- [x] The interview template asks open, adaptive questions rather than a fixed script, states an explicit stopping condition, and instructs saving synthesized findings to a working file
- [x] A spec-creation session interrupted mid-interview resumes on the interview step with its findings intact
- [ ] The step-order and full-workflow-walk unit tests pass with the new step in place — deliberately deferred to Phase 1.3, which owns updating these hardcoded step-order expectations (see context.md Phase 1.3 notes); both tests currently fail as expected and will be fixed when that phase lands

#### - [x] Phase 1.2: Give the interview access to the project's repo roster and a cross-repo question

**Repo:** spektacular

Relocate the plan workflow's existing repo-roster mechanism to shared locations both workflows can use, then wire it into the spec workflow's new interview step exactly as the plan workflow already wires it into its own discovery and architecture steps. The interview template renders the full roster of the project's registered repositories, and instructs asking whether a feature that reads as focused on one repo also needs changes in the others, shaped by what each other repo actually is.

*Technical detail:* [context.md#phase-12](./context.md#phase-12-give-the-interview-access-to-the-projects-repo-roster-and-a-cross-repo-question)

**Acceptance criteria**:

- [x] The interview step's rendered instruction lists every registered repo's name, role, and description in a project with more than one repo, and falls back gracefully in a single-repo project
- [x] The interview's instructions direct asking a cross-repo impact question, shaped by each other repo's stated role, whenever the feature reads as primarily about one repo
- [x] The plan workflow's existing repo-roster rendering is unchanged in output after the relocation, confirmed by its existing tests continuing to pass

#### - [x] Phase 1.3: Align the driving skill and end-to-end harness with the new step

**Repo:** spektacular

Update the agent-facing skill instructions so the spec-creation loop covers the interview step, including how a resumed session continues an open interview, and update the end-to-end harness's canonical step order and success criteria so the suite validates the new sequence.

*Technical detail:* [context.md#phase-13](./context.md#phase-13-align-the-driving-skill-and-end-to-end-harness-with-the-new-step)

**Acceptance criteria**:

- [x] The skill instructions describe the interview as the first step after spec creation, including its stopping condition and repo-awareness behavior
- [x] The end-to-end harness expects the new canonical step order and its scripted golden-path solution walks through the interview step
- [x] The template contract tests account for the new step's working-file and context-directive obligations

### Milestone 2: Sections are drafted and confirmed, and rejecting a draft opens a conversation instead of a blind edit

**What changes**: Each section of the spec is now drafted from the interview and presented to the user to confirm or correct, rather than the user being asked to author it from a blank prompt. When the user indicates a drafted section is wrong, the agent asks a follow-up question to understand why before changing anything, since a single correction can turn out to need more than one change, or to belong to a different section than the one currently under review. A correction that surfaces new information relevant to an already-confirmed earlier section updates that section's content directly, without asking the user to stop and re-review it in the moment; the existing end-of-workflow review remains the point where everything, including anything changed this way, gets one final look together. For a typical spec, the user experiences a colleague proposing a draft and asking what's wrong when something doesn't fit, rather than being interviewed section by section from scratch.

#### - [x] Phase 2.1: Make section steps draft from the interview and present for confirmation

**Repo:** spektacular

Rewrite the opening instruction of every section-gathering step (Overview, Requirements, Acceptance Criteria, Constraints, Technical Approach, Success Metrics, Non-Goals) so each drafts its section from the interview's findings and any prior corrections, presents the draft, and asks the user to confirm or correct it, rather than asking its own scripted question from a blank slate. Every other rule in these templates, format, altitude, the working-file write instruction, stays exactly as it is today.

*Technical detail:* [context.md#phase-21](./context.md#phase-21-make-section-steps-draft-from-the-interview-and-present-for-confirmation)

**Acceptance criteria**:

- [x] No section-gathering step template asks its own scripted question from a blank prompt; each drafts from the interview's working file and presents the draft for confirmation
- [x] Section-specific rules (format, altitude, working-file conventions) are unchanged in every template
- [x] Template unit tests confirm the scripted-question phrasing is gone and draft-and-confirm phrasing is present

#### - [x] Phase 2.2: Add rejection-repair and cross-section amendment to every gathering step

**Repo:** spektacular

Generalize the existing verification-step triage pattern (ask why before changing, update the owning working file, no fresh confirmation gate for an already-passed section) into shared closing-beat instructions carried by the interview step and all seven section steps. When the user rejects a drafted section, the agent asks a follow-up question before making any change, and any resulting edits land in the working file(s) they actually belong to, which may include a section other than the one under review, without requiring the user to separately revisit and re-confirm that other section in the moment.

*Technical detail:* [context.md#phase-22](./context.md#phase-22-add-rejection-repair-and-cross-section-amendment-to-every-gathering-step)

**Acceptance criteria**:

- [x] Every step from interview through the last section step carries the same rejection-repair instruction: ask why before changing, don't assume the stated fix is complete
- [x] The instructions explicitly cover editing a different section's working file than the one currently under review, with no new confirmation gate for that other section
- [x] A rejection can result in edits to more than one section, or none, depending on what the follow-up conversation surfaces, the instructions do not bound it to exactly one change
- [x] Template unit tests confirm the shared rejection-repair marker text is present on every step that must carry it

#### - [x] Phase 2.3: Verify the new interaction model end to end

**Repo:** spektacular

Update the end-to-end harness to exercise the new interaction model: a scenario where a rejection during a later section produces a change to an earlier, already-drafted section's working file, with no repeated confirmation step appearing for that earlier section. This is the automated proof that cross-section amendment actually fires, not just that it is theoretically described in the templates.

*Technical detail:* [context.md#phase-23](./context.md#phase-23-verify-the-new-interaction-model-end-to-end)

**Acceptance criteria**:

- [x] An end-to-end run opens with an adaptive interview question rather than the Overview section's old scripted prompt
- [x] A scripted rejection during a later section produces an observable change to an earlier section's working file in the same run
- [x] The harness's per-section content checks still pass, confirming drafted-and-confirmed sections are as complete as scripted-and-answered ones were

### Milestone 3: The Flipped Interaction pattern is documented as a Spektacular differentiator

**What changes**: The documentation site names and explains the Flipped Interaction pattern, attributing it to the prior prompt-engineering research it is drawn from, and walks through what a user experiences starting a new spec now, the interview phase, with an example question. A concrete multi-repo example shows the interview asking a cross-repo question in a project with more than one registered repository. This capability is positioned where a prospective user would see it early, the homepage or how-it-works page, rather than only in a deep reference page, so it reads as a reason to choose Spektacular rather than an implementation detail. This milestone is independently deliverable from Milestones 1 and 2 in the sense that it lands in a different repository, but it depends on their behavior actually existing so the documented example reflects reality rather than aspiration.

#### - [x] Phase 3.1: Document the Flipped Interaction pattern and the interview phase on the docs site

**Repo:** docs

In the `docs` repo, describe what happens when a user starts a new spec now: the interview phase, its adaptive question style, and an example exchange, and name the Flipped Interaction pattern with attribution to the prior prompt-engineering research it is drawn from. Position this as a differentiator visible early on the site rather than buried in a deep reference page, and include a concrete worked example of the interview asking a cross-repo question in a multi-repo project.

*Technical detail:* [context.md#phase-31](./context.md#phase-31-document-the-flipped-interaction-pattern-and-the-interview-phase-on-the-docs-site)

**Content outline**:

`how-it-works.mdx`, Stage 1 ("Specification") body, revised structure:

1. **Opening line naming the pattern** — states the interview happens first, names "Flipped Interaction," attributes it to prior prompt-engineering research.

   *Illustrative copy:* "Before Spektacular drafts a single section, it interviews you. Rather than working through a fixed script, the agent asks open questions about what you're building, adapting each one to what you've already said, a technique called Flipped Interaction, drawn from prompt-engineering research on shifting conversational control to the agent."

2. **What the interview asks, with one example exchange** — a short worked example in the existing `slot="example"` fenced-block pattern already used elsewhere on this page.

   *Illustrative example:*
   ```
   Agent: What are you building, and who runs into the problem it solves?
   You:   A CLI users can point at a local repo to check for
          outdated dependencies.
   Agent: Should it just report outdated packages, or offer to
          upgrade them too?
   You:   Report only, for now, upgrading is a separate feature.
   Agent: Got it, I have enough to draft an Overview. One more
          thing: does this touch the docs site too, or is it
          CLI-only for now?
   ```

3. **When the interview stops** — one sentence: it stops once no further question would change the draft, not once every detail is enumerated.

4. **Cross-repo worked example** — a second fenced example, reusing config-key terminology already documented on `configuration.mdx`/`repo-configuration.mdx`, showing the interview asking about a second registered repo.

   *Illustrative example:*
   ```
   Agent: This project also has a "docs" repo registered
          (role: documentation). Does exposing this as a new
          CLI flag mean the docs site needs a new page or
          section too?
   You:   Yes, it should get its own reference page.
   ```

5. **Transition to section drafting** — one or two sentences bridging into the existing "then guides you through filling in requirements, constraints, and acceptance criteria" language, revised to say sections are now drafted and presented for confirmation rather than asked from scratch.

**Acceptance criteria**:

- [x] The page names the Flipped Interaction pattern and states it is drawn from prior prompt-engineering research
- [x] The page describes the interview phase with at least one example question or exchange
- [x] The page includes a worked example of the interview asking a cross-repo question in a multi-repo project

#### - [x] Phase 3.2: Feature the capability on the homepage

**Repo:** docs

Add or reframe a card in the homepage's existing features grid so the Flipped Interaction interview is visible to a prospective user early, rather than only on the deeper how-it-works page.

*Technical detail:* [context.md#phase-32](./context.md#phase-32-feature-the-capability-on-the-homepage)

**Content outline**:

`index.mdx`, `FeaturesGrid`, one new (or reframed) `FeatureCard`:

- **Icon**: a conversation/question-mark-style icon consistent with the grid's existing icon set.
- **Title**: short noun phrase naming the capability, e.g. "Flipped Interaction interview."
- **Body** (1-2 sentences, illustrative): "Spektacular interviews you before drafting a spec, asking adaptive questions instead of a fixed script, so you get a complete spec even when you haven't fully thought the feature through yet."

**Acceptance criteria**:

- [x] The homepage features this capability as one of its featured cards, not only referenced on a deeper page
- [x] The docs site builds and type-checks cleanly with the new content

## Open Questions

- **Whether the spec-workflow step templates' numeric filename prefixes are load-bearing anywhere beyond human readability.** Phase 1.1 assumes `00b-interview.md` is safe (no renumbering of `01-overview.md` through `09-finished.md`), based on the FSM's Src/Dst wiring being the actual source of step order, not filename sort order, but this was not exhaustively verified against every place a template path might be enumerated, sorted, or pattern-matched by prefix. Depends on: reading `internal/stepkit`'s template-loading code and any test that walks `templates/steps/spec/` expecting a specific naming scheme. What to do: if the implementer finds any such dependency, use the renumbering fallback already named in context.md's Phase 1.1 notes instead of stopping to ask.
- **The harbor `spec-workflow` suite's `solution/solve.sh` script's actual role in the test suite.** Phase 2.3 depends on knowing whether `solve.sh` is exercised by the same test assertions as the agent-driven run, or is a separate golden-path baseline with its own, looser bar, this determines whether it needs to also demonstrate the cross-section-amendment scenario or can note the scenario as agent-only. Depends on: reading `tests/harbor/spec-workflow/tests/test.sh` and any harbor-runner documentation describing how `solution/` is invoked relative to the agent transcript tests. What to do: STOP and ask the user only if reading `test.sh` and the harbor tooling's own docs leaves genuine ambiguity after a direct look; otherwise resolve it silently from what's found.

## Out of Scope

- **Applying the interview/Flipped Interaction behavior to the plan workflow's discovery or architecture steps.** Explicit spec Non-Goal, the plan workflow's autonomous-drafting-plus-walkthrough model (delivered by 000040_plan-draft-then-walkthrough) is already the right shape for plan work, which is agent-grounded technical synthesis from readable code rather than user-only tacit-knowledge extraction. Not tracked anywhere as future work; this plan treats the boundary as permanent, not deferred.
- **Documenting the rejection-repair behavior on the documentation site.** Explicit spec Non-Goal, the interview phase and the cross-repo example are documented (Milestone 3), but the rejection-follow-up behavior itself is deliberately left undocumented, matching the spec's own scoping decision made during spec creation.
- **Any change to the plan workflow's own behavior beyond the repo-roster relocation.** The relocation of `repoRoster`/`repoRosterExtra` (Phase 1.2) touches files the plan workflow depends on, but is designed and tested to be behavior-neutral for that workflow, this plan is not a vehicle for any other plan-workflow change, and any such change belongs to its own future plan.
- **A distinct interview mechanism or working-file schema beyond the single `interview.md` convention.** The architecture step considered and rejected a more structured, plan-workflow-`assumptions.md`-style fixed-entry-shape file for interview findings; this plan deliberately keeps the interview file as unstructured synthesized prose, rather than building the more elaborate structured-log mechanism plan's own `assumptions.md` convention uses, since interview findings are read by drafting prompts, not folded into a document by a parsing step.
- **A new confirmation gate for sections amended as a side effect of a rejection elsewhere.** Explicit spec constraint, reaffirmed in this plan's architecture: relies entirely on the existing end-of-workflow verification step's fresh-eyes-review-and-triage loop as the single safety net. No new gate is designed or planned.

## Changelog

### FINAL SUMMARY

All 8 phases delivered as planned. The spec workflow now opens with an adaptive `interview` step (Flipped Interaction pattern) before any section is drafted, aware of the project's full registered-repo roster and able to ask cross-repo impact questions. Every section step (Overview through Non-Goals) now drafts its content from the interview and presents it for confirmation rather than asking its own scripted question, and rejecting a draft opens a follow-up conversation that can amend a different, already-confirmed section without a fresh confirmation gate — the existing end-of-workflow review remains the single point everything is checked together. The behavior is documented on the Spektacular site as a named, attributed pattern, with a worked cross-repo example, and featured on the homepage rather than buried in a reference page.

**Total phases**: 8/8 completed

**Notable deviations from the plan**: Two test files not anticipated in context.md's original file lists needed fixing as side effects of the FSM change — `cmd/root_test.go` (Phase 1.1, two tests hardcoding a direct `new`→`overview` transition) and the rejection-repair paragraph wording needed strengthening once during Phase 2.2 to fully satisfy its own acceptance criteria. Phase 3.1's content used plain fenced code blocks instead of a new component slot, since the assumed `PipelineStage` `slot="example"` support doesn't exist on that component. None of these affected scope or required plan changes beyond noting them in the per-phase entries below.

### 2026-08-14 — Phase 1.1: Insert the interview step into the spec workflow

**What was done**: Added a new `interview` step to the spec workflow's FSM, sitting between `new` and `overview`, so every spec-creation session opens with an adaptive, Flipped-Interaction-style interview before any section is drafted. `new` now advances to `interview` instead of directly to `overview`. A new step template (`00b-interview.md`) states the interview's goal, instructs adaptive (non-scripted) questioning, defines an explicit stopping condition, and instructs saving synthesized findings to a new per-workflow working file (`.spektacular/work/{{spec_name}}/interview.md`).

**Deviations**: During verification, found that `cmd/root_test.go` had two tests (`TestSessionLog_AdvancedTrueWhenStateChanges`, `TestSessionLog_SameSessionIDAcrossFoundingAndResumedCalls`) hardcoding `spec new` → `spec goto {"step":"overview"}` as a directly-reachable transition; these broke once `interview` was inserted between them. This file was not listed in context.md's Phase 1.3 file list (which anticipated `internal/steps/spec/steps_test.go` and the harbor Python suite, not `cmd/root_test.go`). User chose to fix these two tests now rather than deferring to Phase 1.3 — see Files changed. The 4th acceptance criterion ("step-order and full-workflow-walk unit tests pass with the new step in place") was left unchecked in plan.md, since fixing `internal/steps/spec/steps_test.go`'s two FSM-order tests is explicitly Phase 1.3's job and they still fail by design at this point.

**Files changed**:
- `internal/steps/spec/steps.go`
- `templates/steps/spec/00-new.md`
- `templates/steps/spec/00b-interview.md`
- `cmd/root_test.go`

**Discoveries**: No test in the codebase enforces sequential filename numbering on `templates/steps/spec/*.md` — templates are referenced by explicit path in `steps.go` and by explicit lists in `templates/work_files_test.go`/`templates/context_directive_test.go`, so `00b-interview.md` did not require renumbering `01-overview.md` through `09-finished.md` (resolves the plan's Open Question on this point in Phase 1.1's favor — no renumbering needed). Also: `templates/context_directive_test.go`'s `TestContextDirectivePresent` walks `steps/spec` via `fs.WalkDir`, so new step templates are automatically covered by its context-directive assertion with no test-file edit required, as long as the template carries the standard directive footer.

### 2026-08-14 — Phase 1.2: Give the interview access to the project's repo roster and a cross-repo question

**What was done**: Relocated the plan workflow's repo-roster mechanism (`repoRoster`, `repoRosterExtra`) from `cmd/plan.go`/`internal/steps/plan/steps.go` to shared locations — `internal/repo.Roster` and `internal/stepkit.RepoRosterExtra` — so both the plan and spec workflows can use it. Wired the spec workflow's `interview` step to render the full repo roster and instruct asking a cross-repo impact question (shaped by each other repo's own role/description) whenever the feature reads as focused on one repo in a multi-repo project.

**Deviations**: None beyond the drift already accepted at the read_plan gate (`cmd/spec.go`'s missing `internal/repo` import and missing `repoGit`-in-scope) — both resolved cleanly: `repoGit` is a package-level var in `package cmd` (`cmd/repo.go`), so it was already reachable from `cmd/spec.go` with no new wiring beyond the import. One test bug was introduced and fixed during the relocation itself (see Discoveries) — not a deviation from the plan, a self-corrected implementation mistake.

**Files changed**:
- `internal/repo/roster.go` (new)
- `internal/repo/roster_test.go` (new — relocated + fixed from `cmd/plan_test.go`)
- `cmd/plan.go`
- `cmd/plan_test.go`
- `internal/stepkit/stepkit.go`
- `internal/steps/plan/steps.go`
- `internal/steps/spec/steps.go`
- `internal/steps/spec/steps_test.go`
- `cmd/spec.go`
- `templates/steps/spec/00b-interview.md`

**Discoveries**: When relocating the repo-roster unit tests from `cmd/plan_test.go` to `internal/repo/roster_test.go`, an initial version using a bare `config.RepoConfig{...}` struct literal (rather than starting from `config.NewDefaultRepoConfig()` and overriding only descriptive fields) silently failed `DescriptiveMetadata` — `config.RepoConfigFromYAMLFile`'s `cfg.Validate()` rejects a `RepoConfig` whose `Knowledge`/`Changelog` sub-configs are left zero-value. This is a non-obvious contract: any test or caller constructing a `config.RepoConfig` from scratch (not read from an existing YAML file) must go through `NewDefaultRepoConfig()` first, not build the struct directly, even when only the descriptive fields (`Description`/`Role`/`Tags`/`Deployment`) are what the test cares about. Caught immediately by the relocated test failing before this changelog entry was written; fixed and confirmed passing.

### 2026-08-14 — Phase 1.3: Align the driving skill and end-to-end harness with the new step

**What was done**: Updated the spec-creation skill's driving instructions and the end-to-end test harness so both reflect the new `interview` step. The skill now describes the interview as the first step after spec creation — its stopping condition, repo-roster awareness, and mid-interview resume behavior. The harbor end-to-end suite's canonical step order, its scripted golden-path solution, and the template contract tests (`work_files_test.go`) all now account for the new step. Fixed the two FSM-order unit tests (`internal/steps/spec/steps_test.go`) that were deliberately left failing since Phase 1.1.

**Deviations**: None. `cmd/root_test.go` (a related fix, but not originally listed for this phase) was already handled during Phase 1.1's verify step per user decision — not duplicated here.

**Files changed**:
- `internal/steps/spec/steps_test.go`
- `tests/harbor/spec-workflow/tests/test_spec_workflow.py`
- `tests/harbor/spec-workflow/solution/solve.sh`
- `templates/skills/workflows/spek-new/SKILL.md`
- `templates/work_files_test.go`

**Discoveries**: The installed skill copies (`.claude/skills/spek-new/SKILL.md`, `.bob/skills/spek-new/SKILL.md`) are git-tracked but generated output of `go run . init <agent>`, rendered from the template this phase edited (`{{command}}` → `go run .`). They are deliberately NOT regenerated by this change — AGENTS.md documents that hand edits (or, by extension, edits that bypass `init`) to these installed copies "will not survive the next init," so they intentionally lag until a user explicitly re-runs `go run . init` in this project. No Makefile target regenerates them automatically. Confirmed with `go test ./...`: this is the first phase where the entire suite is fully green with zero failures anywhere in the repo.

### 2026-08-14 — Phase 2.1: Make section steps draft from the interview and present for confirmation

**What was done**: Rewrote the opening instruction of all 7 section-gathering step templates (Overview, Requirements, Acceptance Criteria, Constraints, Technical Approach, Success Metrics, Non-Goals) so each now drafts its section from the interview's findings (and any prior working-file content) and presents the draft for the user to confirm or correct, rather than asking its own scripted question from a blank prompt. Every section-specific rule (format, altitude boundaries, working-file conventions) was left unchanged.

**Deviations**: None. `07-non_goals.md` already had a partial draft-propose-confirm pattern for inferred non-goals before this phase; adapted rather than replaced it, since it mapped directly onto the new model.

**Files changed**:
- `templates/steps/spec/01-overview.md`
- `templates/steps/spec/02-requirements.md`
- `templates/steps/spec/03-acceptance_criteria.md`
- `templates/steps/spec/04-constraints.md`
- `templates/steps/spec/05-technical_approach.md`
- `templates/steps/spec/06-success_metrics.md`
- `templates/steps/spec/07-non_goals.md`
- `templates/section_drafting_test.go` (new)

**Discoveries**: None new beyond what was already noted in Phase 1.3's entry.

### 2026-08-14 — Phase 2.2: Add rejection-repair and cross-section amendment to every gathering step

**What was done**: Generalized the verification step's existing triage pattern (ask why before changing, update the owning working file, no fresh confirmation gate) into a shared closing-beat paragraph carried by all 8 gathering-step templates (the interview step plus all seven section-drafting steps). When the user rejects a drafted section, the agent now asks a follow-up question before making any change; resulting edits land in whichever working file(s) they actually belong to, which may be a different section than the one under review, without requiring a fresh confirmation gate for that other section; the edit count is explicitly unbounded (could be zero, one, or several sections).

**Deviations**: The initial paragraph wording (copied closely from context.md's specified text) didn't fully satisfy acceptance criterion 3 ("edits to more than one section, or none, not bound to exactly one change") on inspection — strengthened with one additional sentence across all 8 templates before checking that criterion off.

**Files changed**:
- `templates/steps/spec/00b-interview.md`
- `templates/steps/spec/01-overview.md`
- `templates/steps/spec/02-requirements.md`
- `templates/steps/spec/03-acceptance_criteria.md`
- `templates/steps/spec/04-constraints.md`
- `templates/steps/spec/05-technical_approach.md`
- `templates/steps/spec/06-success_metrics.md`
- `templates/steps/spec/07-non_goals.md`
- `templates/rejection_repair_directive_test.go` (new)

**Discoveries**: None new.

### 2026-08-14 — Phase 2.3: Verify the new interaction model end to end

**What was done**: Extended the harbor end-to-end test suite with a scripted cross-section-amendment scenario. `instruction.md` now scripts the driving agent to reject the Constraints draft, explaining (when the agent asks why) that token revocation must be supported — a Requirements-level behavior surfaced only while reviewing a later section. `test_spec_workflow.py` gained a new `TestCrossSectionAmendment` class asserting the final spec's Requirements section reflects the correction, and that Requirements was not re-confirmed a second time in the transcript. Resolved the plan's Open Question about `solution/solve.sh`'s role: it is a Harbor-framework standalone reference script never invoked by the pytest-driven `test.sh`, so it does not need to reproduce the scenario — noted as an explicit comment rather than left ambiguous.

**Deviations**: None from the plan's stated approach; the Open Question resolution matches the fallback the plan itself had already named ("note the scenario as agent-only").

**Files changed**:
- `tests/harbor/spec-workflow/instruction.md`
- `tests/harbor/spec-workflow/tests/test_spec_workflow.py`
- `tests/harbor/spec-workflow/solution/solve.sh` (comment only)

**Discoveries**: The per-section working files under `.spektacular/work/{{spec_name}}/` are deleted by `08-verification.md`'s cleanup step before the harbor harness ever inspects the container's filesystem — so an end-to-end assertion of cross-section amendment can only check the *persisted, final assembled spec* for the correction's effect, never the intermediate working file directly. This is a structural constraint on what any future e2e scenario in this harness can assert, not specific to this one test.

### 2026-08-14 — Phase 3.1: Document the Flipped Interaction pattern and the interview phase on the docs site

**What was done**: Rewrote the how-it-works.mdx page's Stage 1 "Specification" pipeline description in the `docs` repo to describe the new interview-first workflow. Named the Flipped Interaction pattern with attribution to prior prompt-engineering research, added a worked interview example exchange, stated the stopping condition, added a cross-repo worked example (the interview asking about a registered "docs" repo), and revised the transition language to describe draft-and-confirm section drafting instead of the old scripted-question model. Also updated the "New" pipeline node's sub-label and the Quick-start step 3 copy to reflect the interview happening first.

**Deviations**: context.md's Phase 3.1 notes assumed a new `Fragment slot="example"` block on `PipelineStage` (following `SpecFormat`'s pattern), but `PipelineStage.astro` only supports a default slot and `slot="body"` — no `slot="example"`. Rather than extend the Astro component (out of scope for a content-only plan), both example exchanges were embedded as plain fenced code blocks directly within the existing `Fragment slot="body"`, which the site's existing prose conventions already support.

**Files changed**:
- `docs: src/pages/how-it-works.mdx`

**Discoveries**: None new.

### 2026-08-14 — Phase 3.2: Feature the capability on the homepage

**What was done**: Added a new `FeatureCard` to the homepage's `FeaturesGrid` in the `docs` repo, positioned right after the existing "Spec-driven workflow" card: icon 💬, title "Flipped Interaction interview", body copy describing the adaptive-questions-instead-of-a-fixed-script behavior. This surfaces the capability as a homepage differentiator rather than only on the deeper how-it-works page.

**Deviations**: None.

**Files changed**:
- `docs: src/pages/index.mdx`

**Discoveries**: None new.
