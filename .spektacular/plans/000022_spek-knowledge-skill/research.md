# Research: 000005_spek-knowledge-skill

## Alternatives considered and rejected

### Option A: CLI-driven workflow mirroring spec/plan/implement

A new `spektacular knowledge-flow new / goto` subcommand tree with an
FSM, step templates under `templates/steps/knowledge/`, a separate
state file at `.spektacular/knowledge-state.json`, and a single
workflow that branches via callback-returned next-step (lookup vs
compose vs select-existing).

**Rejected**: too heavy for the orchestration shape the spec actually
needs. The existing `spektacular knowledge` CRUD commands
(`cmd/knowledge.go:106-191`) already expose every primitive the
workflow would have wrapped. State only has to survive one user turn
(propose → confirm), which the agent's own conversation loop
already manages — there is nothing for `state.json` to add. See
also the user-supplied design preference recorded in memory
[[thin-skill-over-workflow]]. The workflow approach was also the
spec's stated Technical Approach
(`.spektacular/specs/000005_spek-knowledge-skill.md` §Technical
Approach), which this plan supersedes.

### Option B: Three separate skills (lookup / contribute / update)

Ship `spek-knowledge-lookup`, `spek-knowledge-write`, and
`spek-knowledge-update` as three distinct workflow skills, each with
its own slash command.

**Rejected**: spec wording "invocation by natural intent"
(`.spektacular/specs/000005_spek-knowledge-skill.md` §Requirements,
line 45-49) calls for the user to express intent in one natural
sentence and the agent to figure it out. Three slash commands push
disambiguation onto the user. A single skill whose body branches in
prose is friction-free for the user and trivially distributed by the
existing `workflowSkills` slice (`internal/agent/skills.go:25`).

### Option C: Embed the playbook in CLAUDE.md / AGENTS.md

Put the natural-language workflow guidance directly in the
agent-specific instruction files (`CLAUDE.md`, `AGENTS.md`) rather
than as a separate SKILL.md.

**Rejected**: violates the spec's "Must support all
Spektacular-supported agents" constraint
(`.spektacular/specs/000005_spek-knowledge-skill.md` §Constraints,
line 59-62) — claude reads CLAUDE.md, codex reads AGENTS.md, bob
reads neither directly. The skill distribution system at
`internal/agent/skills.go:39` already routes one template to all
three agents, which is the right channel.

### Option D: Share `.spektacular/state.json` with spec/plan/implement

Reuse the existing shared state file for the knowledge workflow.

**Rejected** (not directly relevant since Option A is itself
rejected, but recorded as evidence in case the workflow approach is
ever revisited): spec/plan/implement currently all overwrite the
shared state file at `.spektacular/state.json` —
`cmd/spec.go:212` and `cmd/plan.go:116` each call
`_ = os.Remove(statePath)` on `new`. A knowledge workflow sharing
the file would corrupt an in-progress spec/plan/implement, violating
the constraint "Must not disrupt an in-progress workflow"
(`.spektacular/specs/000005_spek-knowledge-skill.md` §Constraints,
line 64-67).

### Option E: A `--confirmed` CLI guard on `knowledge write`

Add a `--confirmed` flag (or environment variable) that
`spektacular knowledge write` requires; the skill prose would set it
only after explicit user confirmation. Hard-code the
propose-then-confirm contract.

**Rejected**: re-introduces CLI surface and complexity (a new flag,
error handling, schema update at `cmd/knowledge.go:70-76`) for a
guarantee that the existing trust model already delivers via prose
in the plan workflow's discovery step. The user's explicit preference
recorded in [[thin-skill-over-workflow]] is to keep the design as
small as possible; adding a CLI guard fights that preference.

## Chosen approach — evidence

A single thin SKILL.md installed alongside `spek-new`, `spek-plan`,
`spek-implement`. Body orchestrates existing
`spektacular knowledge` CRUD in natural language. No new CLI
commands, no FSM, no state file, no Go packages added.

**Skill template + install slot already exist** — one new entry in
the workflow-skills registry distributes the skill to every supported
agent:

- `internal/agent/skills.go:25-29` — `workflowSkills` slice;
  adding `{Name: "spek-knowledge", TemplatePath:
  "skills/workflows/spek-knowledge/SKILL.md"}` is the install-side
  change in its entirety.
- `internal/agent/skills.go:39-64` — `installWorkflowSkills`
  iterates the slice unchanged.
- `internal/agent/commands.go:18-22` — `workflowDescriptions`
  map; one new entry feeds bob's command-wrapper renderer.
- `internal/agent/claude.go:13`, `internal/agent/codex.go:13`,
  `internal/agent/bob.go:13` — confirm all three agents install via
  the same `installWorkflowSkills` call.
- `templates/skills/workflows/spek-plan/SKILL.md` — body shape to
  mirror.

**CRUD already covers every behaviour requirement** — mapping spec
requirements to existing CLI calls:

- Look up entries on demand → `cmd/knowledge.go:106`
  (`runKnowledgeSearch`) and `cmd/knowledge.go:125`
  (`runKnowledgeRead`).
- Contribute new entries → `cmd/knowledge.go:191`
  (`runKnowledgeSources`) to enumerate scopes, then
  `cmd/knowledge.go:168` (`runKnowledgeWrite`).
- Update existing entries → `read` + `write` to the same scope/path.
- Search results identify their scope → results are already
  scope-tagged at `internal/knowledge/set.go:77` (`Set.Search`).
- Scope selection is explicit → `internal/knowledge/set.go:126`
  (`Set.Sources`) returns the configured scopes.

**Propose-then-confirm lives in SKILL.md prose, not in code** — the
same model the plan workflow's discovery-step knowledge-write
guidance uses. No new enforcement primitive needed.

**Agent-agnostic distribution is free** — all three supported agents
(claude, codex, bob) consume byte-identical SKILL.md via
`installWorkflowSkills`. One template file installs everywhere; no
per-agent code changes.

## Files examined

- `internal/workflow/workflow.go:65-203` — FSM construction and
  callback-driven branching; reviewed to confirm the existing FSM
  could support the rejected workflow option, before opting for the
  thin-skill direction.
- `internal/workflow/state.go:14-54` — `State` shape and
  load/save; relevant only for understanding the non-disruption
  constraint and the rejected shared-state-file option.
- `internal/workflow/data.go:6-40` — `Data` interface; not
  consumed by the chosen design but examined for completeness.
- `internal/stepkit/stepkit.go:55-94` — generic step renderer;
  relevant only to the rejected workflow option.
- `internal/steps/spec/steps.go:21-34`,
  `internal/steps/plan/steps.go:28-49` — sample workflow shapes
  not adopted.
- `internal/agent/agent.go:20-22` — `Agent` interface (Name +
  Install) confirming the registration surface.
- `internal/agent/skills.go:25-29` — `workflowSkills` slice; the
  single line of source code to extend.
- `internal/agent/skills.go:39-64` — `installWorkflowSkills`
  iteration confirms a new entry is automatically rendered and
  written for every supported agent.
- `internal/agent/commands.go:18-22` — `workflowDescriptions`
  map; the second line of source code to extend.
- `internal/agent/commands.go:28-58` — `installCommandWrappers`
  iteration confirms bob's wrapper is automatically rendered for the
  new skill once the description is in the map.
- `internal/agent/claude.go:13-18`,
  `internal/agent/codex.go:13-15`,
  `internal/agent/bob.go:13-22` — per-agent `Install` methods, all
  three calling `installWorkflowSkills` (bob additionally calls
  `installCommandWrappers`).
- `internal/agent/claude_test.go:25-37`,
  `internal/agent/codex_test.go:25-37`,
  `internal/agent/bob_test.go:25-65` — install tests; each
  `skillAssertions` map needs a fourth entry, bob also needs a
  fourth `commandAssertions` entry and an addition to the
  forbidden-stripped-filenames list at `bob_test.go:58`.
- `internal/agent/instruction_surface_test.go:19-68` —
  forbidden-substring list and walk-based template-surface
  assertions; relevant for ensuring the new template body avoids
  legacy stdin/heredoc patterns.
- `cmd/knowledge.go:106-191` — the four CRUD entry points the
  skill body orchestrates.
- `cmd/knowledge.go:244-251` — confirms the existing `knowledge`
  command tree has no `new/goto/status/steps` subcommands, so adding
  them later (if the workflow option were revived) would not collide.
- `cmd/root.go:54-60` — `dataDir()` returns `.spektacular/`;
  relevant only to the rejected separate-state-file option.
- `cmd/spec.go:99,212`, `cmd/plan.go:116` — proof that the
  existing spec/plan/implement workflows overwrite the shared state
  file on `new`; evidence behind Option D's rejection.
- `internal/knowledge/set.go:25-142` — `Set` type and `Search`,
  `Read`, `Write`, `List`, `Sources` methods; confirms scope is
  surfaced at `Search` and that no new API is needed.
- `templates/skills/workflows/spek-plan/SKILL.md` — body shape and
  `{{command}}` mustache usage; the model the new template mirrors.
- `templates/commands/wrapper.md` — bob's command wrapper
  template; unchanged.

## External references

None. The implementation is entirely internal to the Spektacular CLI
source tree and reuses existing internal APIs and installation flow.
No external libraries, RFCs, or third-party docs informed the design.

## Prior plans / specs consulted

- `.spektacular/specs/000005_spek-knowledge-skill.md` — the source
  spec. Requirements and Acceptance Criteria are stable and drive
  verification; the spec's §Technical Approach is superseded by the
  chosen thin-skill direction recorded in plan.md.
- `.spektacular/specs/000004_astro-migration.md`,
  `.spektacular/specs/000003_update-content.md`,
  `.spektacular/specs/000002_static-site-generation.md`,
  `.spektacular/specs/000001_install_instructions.md` — unrelated
  (website content / stack migration), no overlap with this work.
- `.spektacular/plans/000002_static-site-generation`,
  `.spektacular/plans/000003_update-content`,
  `.spektacular/plans/1_install_instructions` — website plans,
  consulted only to confirm no conflicting work is in flight.

## Open assumptions

- **Implementation target is the Spektacular CLI repo** at
  `/home/nicj/code/github.com/jumppad-labs/spektacular/` (Go CLI
  source). This website repo only holds the spec and the resulting
  plan documents. Confirmed with the user during discovery. If the
  user later expects the skill to land elsewhere, the implement
  workflow must STOP and ask.

- **Spec's Technical Approach is superseded, not amended.** The plan
  and research record the divergence; the spec file itself is not
  edited. Confirmed with the user during architecture. If the user
  later wants the spec rewritten, the implement workflow does not
  need to coordinate that — it is a separate follow-up.

- **Confirmation is enforced by SKILL.md prose, not code.** No
  programmatic guard prevents an agent from skipping the propose-
  then-confirm step. Same trust model as the plan workflow's
  discovery-step knowledge writes. Confirmed during architecture.

- **Knowledge sources are already configured.** The skill assumes
  `spektacular knowledge sources` returns at least one scope. True
  by default — `internal/knowledge/set.go:47` (`NewSet`) synthesises
  a project source when none is configured.

- **Re-running `spektacular init <agent>` is the migration path.**
  Existing Spektacular projects that want the new skill re-run init
  to pick it up. No automated migration step. Confirmed during
  out-of-scope review.

If any of these turn out wrong during implementation, the implement
workflow must STOP and ask before proceeding.

## Rehydration cues

To rebuild context from cold:

- Read the spec at
  `.spektacular/specs/000005_spek-knowledge-skill.md`. Treat
  §Technical Approach as historical; trust Requirements and
  Acceptance Criteria.
- Read the plan at
  `.spektacular/plans/000005_spek-knowledge-skill/plan.md`,
  starting with §Architecture & Design Decisions.
- Read this research.md (chosen approach + open assumptions).
- In the CLI repo at
  `/home/nicj/code/github.com/jumppad-labs/spektacular/`, re-read:
  - `internal/agent/skills.go` and `commands.go` — the two hook
    points.
  - `templates/skills/workflows/spek-plan/SKILL.md` — the body
    shape to mirror.
  - `internal/agent/claude_test.go`, `codex_test.go`,
    `bob_test.go` — the test maps to extend.
  - `internal/agent/instruction_surface_test.go` — the
    forbidden-pattern list the new body must respect.
- Run `spektacular knowledge --help` and
  `spektacular knowledge sources` to confirm the CRUD surface and
  configured scopes have not shifted.
- Invoke `spektacular skill spawn-implementation-agents` if parallel
  implementation orchestration is needed (not expected — both phases
  are Low complexity).
