# Plan: 000005_spek-knowledge-skill

<!-- Metadata -->
<!-- Created: 2026-05-26T16:39:39Z -->
<!-- Commit: 40dcb62 -->
<!-- Branch: f-rationalise-content -->
<!-- Repository: jumppad-labs/spektacular-website -->

## Overview

Add a fourth Spektacular skill, `spek-knowledge`, that lets the user
search, contribute to, and update the project's knowledge base at any
point in a session without starting a formal spec or plan workflow.
The skill is a thin natural-language playbook that orchestrates the
existing `spektacular knowledge` CRUD commands; the user benefits
during ad-hoc and exploratory sessions where valuable conventions,
gotchas, or architecture notes emerge but there is no plan in flight
to capture them.

## Architecture & Design Decisions

The capability ships as a single agent skill — `spek-knowledge` —
installed into every supported agent's skills directory by
`spektacular init`. Its `SKILL.md` body tells the driving agent how
to recognise knowledge-base intent in the user's natural-language
input and to call the existing `spektacular knowledge` CRUD commands
(`search`, `read`, `sources`, `write`) to satisfy that intent. There
is no new CLI subcommand, no FSM, no state file, and no new package
under `internal/steps/`.

One skill handles all three intents (lookup, contribute, update). The
driving agent discriminates at invocation time based on what the user
actually said. This matches the spec wording "invocation by natural
intent" and keeps the slash-command surface flat: one trigger, three
behaviours.

Propose-then-confirm lives in the SKILL.md prose, not in code. Before
any `spektacular knowledge write`, the skill instructs the agent to
show the user the proposed scope, path, and body, and wait for
explicit approval. This reuses the trust model already present in the
plan workflow's discovery-step knowledge writes — a well-trodden
pattern, not a new one.

This direction supersedes the spec's stated Technical Approach, which
proposed a CLI-driven workflow mirroring spec/plan/implement. The
existing `spektacular knowledge` CRUD already exposes every primitive
such a workflow would have wrapped, and persisting state across user
turns adds no value when the agent's conversation loop already
preserves context for a single propose-confirm cycle. See
`research.md#alternatives-considered-and-rejected` for the full
rationale and the trade-offs of the rejected workflow approach.

## Component Breakdown

- **spek-knowledge skill template** *(new)* — owns the
  natural-language guidance that tells the driving agent how to
  recognise knowledge-base intent, which `spektacular knowledge` CRUD
  command to invoke per intent, how to present results with scope
  labels, and how to propose-then-confirm before any write. This is
  the only new artefact in the system.

- **Workflow skill registry** *(changed)* — the central list of
  workflow skills every supported agent installs. Extended with one
  entry so `spek-knowledge` ships to claude, codex, and bob through
  the existing distribution path. The registry is the single hook
  point that keeps the skill agent-agnostic.

- **Workflow description registry** *(changed)* — the map of skill
  name → human-readable description used by the bob command-wrapper
  renderer. Extended with one entry so bob's slash-command menu shows
  meaningful text for the new skill.

- **Existing knowledge CRUD command surface** *(reused, no change)* —
  `spektacular knowledge search / read / list / write / sources`.
  Provides every primitive the skill orchestrates: lookup, scope
  enumeration, persistence. The skill's natural-language body is the
  only thing that changes about how this surface is used.

- **Existing knowledge persistence layer** *(reused, no change)* —
  `knowledge.Set` and its scoped file stores. Already returns
  scope-tagged results from search and exposes configured scopes via
  `sources`, which is exactly what the spec's "search results
  identify their scope" and "scope selection is explicit"
  requirements need.

- **Existing agent installers** *(reused, no change)* — the per-agent
  `Install` implementations iterate the workflow skill registry, so
  they pick up `spek-knowledge` automatically once the registry entry
  is added. No per-agent code change.

## Data Structures & Interfaces

No new types, interfaces, or serialization formats are introduced.

The change extends two existing in-memory data structures by one
entry each:

- The `workflowSkills` slice (entries of the existing `workflowSkill`
  struct: `Name string`, `TemplatePath string`) gains one element
  naming the new skill and its template path.
- The `workflowDescriptions` map (`map[string]string`, skill name →
  description) gains one key/value pair so bob's command-wrapper
  renderer has text to render.

Both shapes are unchanged; only their contents grow. No CLI
input/output JSON schema changes — the new skill consumes the
existing `spektacular knowledge` command schemas, which remain
as-is. No new entries in the persisted state file (the skill writes
none). No new config keys.

## Implementation Detail

No new patterns are introduced in the Go code. The skill's
installation hooks into the existing workflow-skill registry; the
agent installers already iterate it and write each skill into the
agent-specific directory. Adding `spek-knowledge` is a
registry-extension, not a new code path.

The one meaningful new shape lives in the skill template itself.
Existing workflow skills (`spek-new`, `spek-plan`, `spek-implement`)
instruct the driving agent to loop on a CLI state machine — read
instruction, execute, call `goto`, repeat. The new skill body does
**not** loop; it is a static playbook that tells the agent to read
user intent, pick one of three branches (lookup / contribute /
update), and call the corresponding `spektacular knowledge` CRUD
command(s) directly. The propose-then-confirm step before any write
is encoded as prose, identical in spirit to the plan workflow's
discovery-step knowledge-write guidance.

Code-structure UX for a developer touching this area: the diff is
small and additive. One new template file slots into the existing
skills template tree. Two existing data structures each gain one
entry. No new packages, no new files in `internal/`, no new CLI
subcommand wiring in `cmd/`. A reviewer can trace the whole change
from the workflow-skills registry entry to the rendered SKILL.md on
disk in one step.

The skill's natural-language body becomes the source of truth for
the workflow logic. This is a deliberate trade — verifiable in
production by reading one file rather than tracing a state machine,
at the cost of relying on the agent to follow prose instructions
rather than on FSM-enforced transitions. The existing plan-workflow
discovery step uses the same trust model.

## Dependencies

- **`internal/agent` package** *(existing, no change required)* —
  owns `workflowSkills`, `workflowDescriptions`, and the per-agent
  `Install` methods. Provides the entire distribution mechanism the
  new skill needs. No code change beyond adding one entry to each of
  the two existing data structures.

- **`templates` embedded filesystem** *(existing, no change
  required)* — already serves workflow skill templates to the agent
  installers via `templates.FS`. Adding the new SKILL.md file under
  `templates/skills/workflows/spek-knowledge/` is automatically
  picked up.

- **`spektacular knowledge` CRUD command surface** *(existing, no
  change required)* — provides search, read, list, write, sources.
  Every behaviour the new skill orchestrates is already implemented
  and reachable from the user shell. The skill assumes its current
  invocation shape: positional query for search;
  `--data '{"scope","path"}'` JSON for read/write; `--file` for write
  content.

- **`internal/knowledge.Set` and its scoped file stores** *(existing,
  no change required)* — handles scope routing and tags results with
  scope, satisfying "search results identify their scope" and "scope
  selection is explicit" without any new code.

- **Existing `mustache` template rendering and
  `installWorkflowSkills` helper** *(existing, no change required)*
  — already render workflow skill templates with the `{{command}}`
  variable. The new skill follows the same pattern.

- **No new external libraries or upstream specs.** This plan does not
  block on any other in-flight work and introduces no new
  third-party dependencies.

## Testing Approach

Coverage is narrow because the change is narrow: distribute one new
skill template to three agent surfaces.

**Unit tests against the install path.** The new skill is asserted
present in the workflow-skill registry and rendered into each
supported agent's skills directory. This mirrors the existing
per-agent install tests and the cross-agent instruction-surface test
that already asserts every registered workflow skill installs for
every agent — both should pick up `spek-knowledge` automatically
once it is added to the registry. The load-bearing assertion: after
`init <agent>`, a file named `SKILL.md` exists at the agent's
expected `spek-knowledge` directory and contains the rendered body.

**Template rendering test.** The new SKILL.md template is rendered
through the existing `mustache` pipeline with a sample `{{command}}`
value, and the result is checked for presence of the documented
`spektacular knowledge search / read / sources / write` invocations.
This guards against future template edits accidentally dropping the
load-bearing CLI invocations the skill body relies on.

**Bob command-wrapper test.** The bob installer additionally renders
a command wrapper per workflow skill; the existing bob test is
extended to assert the new wrapper exists and references the new
description string. The load-bearing assertion: the wrapper's
"description" line is non-empty and matches the registered
description.

**Deliberate gap: behaviour-level tests.** The actual workflow logic
— does the agent correctly recognise intent, propose before writing,
label results with scope — lives in natural-language SKILL.md prose,
not in Go. It cannot be exercised by Go unit tests. The acceptance
criteria are verified instead by running the skill end-to-end against
a driving agent (claude in dev) and observing whether the documented
behaviours hold. This matches how the existing workflow skills are
themselves verified.

**Test conventions followed.** All new tests live alongside the
existing tests in `internal/agent/` and reuse the same fixture-FS
substitution pattern. No new test infrastructure.

## Milestones & Phases

### Milestone 1: `spek-knowledge` ships with `spektacular init`

**What changes**: After re-running `spektacular init <agent>`, every
supported agent (claude, codex, bob) gains a new `spek-knowledge`
skill in its skills directory. Users see a new entry in the
slash-command menu (`/spek-knowledge` for claude, equivalents for
codex and bob). The skill is present but functionally minimal — its
body at this point is a placeholder that documents the intent and
the available CRUD commands without yet carrying the full
intent-routing and propose-then-confirm prose.

#### - [x] Phase 1.1: Register `spek-knowledge` as a fourth workflow skill

The agent installer learns about a new skill. After this phase,
running `spektacular init <agent>` on any supported agent writes a
`SKILL.md` for `spek-knowledge` into that agent's skills directory,
and bob additionally gets a matching command wrapper. The skill body
at this point is a deliberately minimal placeholder that names the
available knowledge CRUD commands but does not yet carry the full
intent-routing playbook — that lands in Phase 2.1. The split lets
the distribution mechanism land with cheap unit-test verification
ahead of the content, which can only be verified end-to-end.

*Technical detail:* [context.md#phase-11](./context.md#phase-11-register-spek-knowledge-as-a-fourth-workflow-skill)

**Acceptance criteria**:

- [x] After running `spektacular init claude`, `init codex`, and
  `init bob` in a clean tree, each agent's skills directory contains
  a `spek-knowledge` SKILL.md alongside the existing three.
- [x] The bob installer writes a `spek-knowledge` command wrapper
  with a non-empty description.
- [x] The per-agent install tests cover the new skill — the existing
  three assertions are joined by a fourth.
- [x] No existing skill's installed location, contents, or test
  coverage regresses.

### Milestone 2: Knowledge intent works end-to-end

**What changes**: The `spek-knowledge` SKILL.md body carries the
full natural-language playbook: how to recognise lookup, contribute,
and update intent in user input; which `spektacular knowledge` CRUD
command to call for each; how to label search results with their
scope; how to enumerate scopes before any contribution; and how to
propose-then-confirm before any write. After this milestone, a user
can ask their agent in natural language to read, add, or revise
knowledge entries during any session and observe correct behaviour.

#### - [x] Phase 2.1: Author the `spek-knowledge` skill body

The placeholder body from Phase 1.1 is replaced with the full
natural-language playbook: how the agent recognises lookup,
contribute, and update intent in user input; which `spektacular
knowledge` command to invoke per intent; how to label results with
their scope; how to enumerate scopes from `knowledge sources` before
any contribution; and how to propose-then-confirm before any write.
After this phase, a real Claude Code session can drive all three
documented behaviours end-to-end against the configured knowledge
store.

*Technical detail:* [context.md#phase-21](./context.md#phase-21-author-the-spek-knowledge-skill-body)

**Acceptance criteria**:

- [ ] In a real Claude Code session, asking "what do we know about
  X" returns scope-tagged matches from the configured knowledge
  store.
- [ ] Expressing "remember Y" causes the agent to show the proposed
  title, scope, and body, and wait for explicit confirmation; the
  entry appears in the configured scope only after approval.
- [ ] Asking to update an existing entry causes the agent to read
  current content, propose the change, wait for approval, and only
  then write — the entry afterward reflects the approved content.
- [ ] Declining a proposal leaves the knowledge store unchanged from
  its prior state.
- [ ] The same three scripts run in at least one additional
  supported agent surface (codex or bob) produce equivalent
  outcomes.

## Open Questions

None. Every design decision has been resolved during planning:
approach (thin skill vs CLI workflow), skill name (`spek-knowledge`),
granularity (one skill handling all three intents), confirmation
enforcement (SKILL.md prose, not code), target repository
(`/home/nicj/code/github.com/jumppad-labs/spektacular/`), skill body
trigger surface, CRUD-command mapping, and test-file extension sites
are all pinned down. There are no behaviours that can only be
discovered by exercising downstream code, no untested code paths
likely to surface hidden assumptions, and no library or interface
choices deferred to runtime. An empty Open Questions section is the
expected healthy outcome here.

## Out of Scope

- **Replacing the knowledge-base interactions already embedded in
  the plan workflow.** The plan workflow's discovery step continues
  to call `spektacular knowledge` directly during planning. This
  plan is purely additive — it provides an ad-hoc surface for the
  same primitives. (Spec § Non-Goals.)

- **Autonomous extraction of knowledge from conversation context.**
  Every write is explicit and requires user confirmation; the skill
  does not summarise the session and propose a write on its own
  initiative. (Spec § Non-Goals.)

- **A CLI-driven workflow with FSM, state file, and step
  templates.** Considered and explicitly rejected during architecture
  as too heavy for the orchestration shape the spec actually needs.
  The existing `spektacular knowledge` CRUD surface is sufficient.
  Captured for future reference in
  `research.md § Alternatives considered and rejected`.

- **A `--confirmed` flag on `spektacular knowledge write`.**
  Considered as a programmatic guard against unconfirmed writes;
  rejected in favour of prose enforcement to match the existing
  trust model.

- **Splitting into three separate skills
  (`spek-knowledge-lookup`, `-write`, `-update`).** Rejected — one
  skill matches the spec's "invocation by natural intent" wording.

- **Amending the spec file's Technical Approach text.** The plan
  notes that the original CLI-workflow Technical Approach is
  superseded by the chosen thin-skill direction; the spec file is
  left as the historical record. If the user wants the spec
  rewritten, that is a separate, trivial follow-up.

- **A migration step for existing Spektacular projects.** Users who
  already have a project initialised will re-run `spektacular init
  <agent>` to pick up the new skill, matching how earlier workflow
  skills rolled out historically. No upgrade tooling.

- **Behaviour-level Go tests for the SKILL.md prose.** The skill's
  natural-language body cannot be unit-tested in Go; behavioural
  verification happens end-to-end at the Milestone 2 acceptance
  check.

- **Telemetry for the success metrics in the spec.** The spec lists
  growth and usage metrics. The plan ships the capability but does
  not instrument it; metric collection is left to follow-up work.

## Changelog

### FINAL SUMMARY

Shipped the `spek-knowledge` workflow skill as a fourth installable skill for every supported agent (claude, codex, bob). Phase 1.1 registered the skill in the workflow-skill registry, workflow-description map, and per-agent install tests, alongside a placeholder template body. Phase 2.1 replaced the placeholder with the full natural-language playbook for lookup, contribute, and update intents with propose-then-confirm enforcement. Implementation matches the thin-skill architecture chosen during planning — no new CLI subcommand, no FSM, no state file; the existing `{{command}} knowledge` CRUD surface and skill-distribution mechanism do all the heavy lifting.

**Total phases**: 2/2 completed

**Notable deviations from the plan**: Phase 1.1's context.md notes missed two additional tests in `internal/agent/agent_test.go` (`TestInstallWorkflowSkills_WritesThreeSkillFiles` and `TestInstallCommandWrappers_UsesFilenameFunc`) that iterate the real `workflowSkills` slice with fixture FSes; both were updated in lockstep with the registry growth. Phase 2.1's behavioural acceptance criteria (real-session lookup/contribute/update with explicit confirmation and decline handling) were not exercised by this implement run — the plan explicitly classes them as out of scope for Go unit tests, so they remain unchecked pending manual end-to-end verification.

### 2026-05-26 — Phase 1.1: Register `spek-knowledge` as a fourth workflow skill

**What was done**: Added `spek-knowledge` as a fourth entry in the workflow-skill registry and the workflow-description map, and shipped a placeholder `SKILL.md` template that names the four `{{command}} knowledge` CRUD commands and marks the full intent-routing playbook as TBD for Phase 2.1. After `spektacular init <agent>`, every supported agent (claude, codex, bob) now installs a fourth skill alongside the existing three, and bob additionally gets a `spek-knowledge.md` command wrapper. Per-agent install tests were extended in lockstep so the new skill is asserted present, contains the load-bearing `knowledge` substring, and renders the `{{command}}` mustache placeholder cleanly.

**Deviations**: The context.md phase notes named four test files (`claude_test.go`, `codex_test.go`, `bob_test.go`, `instruction_surface_test.go`) but missed two additional tests in `internal/agent/agent_test.go` that iterate the real `workflowSkills` slice with fixture FSes: `TestInstallWorkflowSkills_WritesThreeSkillFiles` (renamed to `...WritesFourSkillFiles`, extended fixture with a fourth template, bumped count from 3 to 4) and `TestInstallCommandWrappers_UsesFilenameFunc` (added a fourth expected wrapper and bumped count from 3 to 4). Both updates were necessary to keep the package green and follow directly from the registry growing by one entry.

**Files changed**:
- `internal/agent/skills.go`
- `internal/agent/commands.go`
- `templates/skills/workflows/spek-knowledge/SKILL.md`
- `internal/agent/claude_test.go`
- `internal/agent/codex_test.go`
- `internal/agent/bob_test.go`
- `internal/agent/agent_test.go`

**Discoveries**: Anything that iterates the package-level `workflowSkills` slice with a substituted fixture FS (the `withSourceFS` helper in `agent_test.go`) needs its fixture grown whenever the slice grows. Future skill additions should expect to touch both `claude_test.go`/`codex_test.go`/`bob_test.go` (per-agent install assertions) and `agent_test.go` (registry-iteration helpers) together. The instruction-surface test required no edits — its walk-based shape automatically picks up new templates and only asserts forbidden-substring absence.

### 2026-05-26 — Phase 2.1: Author the `spek-knowledge` skill body

**What was done**: Replaced the Phase 1.1 placeholder body of `templates/skills/workflows/spek-knowledge/SKILL.md` with the full natural-language playbook: a "When to invoke" trigger surface naming the user phrases that should activate the skill; three intent sections (lookup, contribute, update) each mapping to the matching `{{command}} knowledge` command sequence; explicit propose-then-confirm with staged scratch files under `.spektacular/tmp/` and `--file <staged>` writes (no stdin); and decline-handling prose that makes the write-without-confirmation case a documented bug rather than an acceptable shortcut. Regenerated the dogfooded `.claude/skills/spek-knowledge/SKILL.md` so the project consumes its own latest template, and added a regression test `TestRenderedSpekKnowledgeBodyContainsCRUDInvocations` in `internal/agent/instruction_surface_test.go` asserting the four load-bearing CRUD substrings (`knowledge search/read/sources/write`) appear in the rendered output.

**Deviations**: Phase 2.1's five acceptance criteria are all behavioural ("In a real Claude Code session...") and the plan explicitly classes them as out of scope for Go unit tests. The unit-testable parts (template rendering, CRUD-substring presence) all pass; the behavioural criteria remain unchecked in plan.md pending end-to-end manual verification against a real driving agent. This deferral matches the plan's "Deliberate gap: behaviour-level tests" note in §Testing Approach.

**Files changed**:
- `templates/skills/workflows/spek-knowledge/SKILL.md`
- `.claude/skills/spek-knowledge/SKILL.md` (regenerated via `go run . init claude`)
- `internal/agent/instruction_surface_test.go`

**Discoveries**: The new template's `Write`-tool staging step under `.spektacular/tmp/<slug>.md` is intentionally compatible with the existing forbidden-substring list — using `--file <staged>` exclusively avoids the `cat ... | knowledge write` patterns the walk-based instruction-surface test rejects. Any future skill that orchestrates a knowledge-store write should adopt the same staging idiom rather than reintroducing stdin pipes.
