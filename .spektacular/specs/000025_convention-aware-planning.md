# Feature: 000025_convention-aware-planning

## Overview

When the team plans a new piece of work, the project's accumulated know-how should shape the plan. That know-how comes in two forms: standards that always apply (for example, how the team prefers to structure code) and lessons learned about particular areas (for example, how a specific kind of data should be handled). Today this know-how is easy to overlook while planning, so plans can quietly drift away from how the team has agreed to build things.

This feature makes planning actively draw on that knowledge. The always-apply standards are reviewed every time a plan is created. The lessons relevant to what is being built are pulled in only when that topic actually comes up. Most importantly, the specific standards that bear on this piece of work are written into the plan itself, so whoever builds it is following a plan that already states which standards apply.

The people who benefit are the planners (their plans reflect established practice without having to remember every rule), the people implementing the work (they follow one plan that already carries the relevant standards), and reviewers (they can see at a glance which standards governed the work — and an empty or generic list is a visible signal the knowledge was not really consulted).

## Requirements

- When creating a plan, the system must review all of the project's always-apply conventions.
- When creating a plan, the system must consult topic-specific knowledge only for the topics the work actually touches, rather than reviewing the entire body of topic-specific knowledge.
- The conventions relevant to the work being planned must be recorded in the resulting plan, each accompanied by a stated reason it applies to this work.
- Conventions that are not relevant to the work being planned must not be recorded in the plan.
- Always-apply conventions must be excluded from topic-specific knowledge lookups, so the same convention is not surfaced a second time during planning.
- A user must be able to add or change a single always-apply convention as an independent item, without editing unrelated conventions.
- When implementing a plan, the implementer must rely solely on the plan for the conventions that apply, and must not separately consult the knowledge base.
- When no conventions are relevant to the work (or none exist), the plan must state this plainly, so a reader can tell the knowledge was considered rather than skipped.
- The end-to-end plan-workflow test suite must verify that always-apply conventions are read during planning and that the relevant ones are reflected in the resulting plan.

## Constraints

- Existing projects must keep working: introducing the always-apply convention layout must not orphan, hide, or silently drop conventions a project has already recorded.
- Topic-specific knowledge lookups must return the same results in every environment, whether or not an optional accelerated search tool is available on the machine — the presence or absence of that tool must not change which entries are surfaced or excluded.

## Acceptance Criteria

- Given a project with several always-apply conventions, when a plan is created for a feature that relates to some of them, the finished plan contains a dedicated section listing exactly those relevant conventions, and each listed convention is accompanied by a reason it applies to this feature.
- Given the same plan, conventions that have no bearing on the feature do not appear in that section.
- Given a project with no always-apply conventions, or a feature unrelated to any of them, the finished plan's conventions section is present and plainly states that none apply.
- When a topic-specific knowledge lookup is performed during planning, its results never include entries drawn from the always-apply conventions.
- A user can add a new always-apply convention by introducing a single new item, and a subsequent plan created for a related feature reflects that new convention in its conventions section, without any other convention being modified.
- A plan can be implemented end to end using only the plan documents; at no point does implementing the plan require reading the knowledge base, and the conventions applied during implementation are exactly those recorded in the plan.
- The end-to-end plan-workflow test seeds a distinctive always-apply convention in its project environment; the test suite fails if a full plan run does not read the conventions during its research phase, and fails if the resulting plan's conventions section does not reflect the seeded convention.

## Technical Approach

The knowledge base is treated as a **planning-time input only**. All consultation happens while a plan is being authored; the plan documents are the distilled output that implementation consumes. The implement workflow gains no new knowledge-access behavior — it continues to follow the plan.

Knowledge is split into two tiers by access pattern, not by storage scope:

- **Tier 1 — conventions (always-apply).** Stored as individual files under a reserved `conventions/` directory within each knowledge scope. One rule per file purely for authoring ergonomics; files are concatenated when read. A new read-only CLI subcommand (sketch: `knowledge conventions`) fans across configured scopes — mirroring how the existing `knowledge search`/`list` fan out — and returns every convention body, scope-tagged. A dedicated command is required because the workflow skills forbid reading knowledge entries with the raw `Read` tool; all knowledge access is routed through the CLI. The existing flat `conventions.md` is migrated into this directory, and project scaffolding/init creates the directory for new projects.

- **Tier 2 — reference knowledge (on-demand).** The existing folders (architecture, gotchas, learnings, …) reached only via targeted `knowledge search <surface>`, keyed on the design surfaces a feature introduces (e.g. a file→database migration triggers a search for database conventions). This replaces the current broad "search for anything about this area" pass in the discovery step.

`knowledge search` is amended to **exclude the `conventions/` directory** in every scope, since conventions are already digested in full at discovery and would otherwise be surfaced twice. The exclusion must be applied identically in both the ripgrep-backed search path and the native directory-walk fallback so results stay environment-independent (an existing test asserts the two paths are equivalent — `internal/store/search_test.go`).

The plan workflow changes:

- The **discovery step** (`templates/steps/plan/02-discovery.md`) loads all conventions in full via the new command and runs surface-targeted Tier-2 searches instead of the broad area search.
- A **`## Conventions` section is added to `plan.md`**, populated during planning with only the conventions judged relevant to the feature, each annotated with a one-line rationale and cited inline in the design section it drives. An explicit "none apply" state is written when nothing is relevant — this doubles as a visible knowledge-consultation check.

End-to-end coverage extends the existing `tests/harbor/plan-workflow` harbor task:

- The task **environment** seeds a distinctive always-apply convention (relevant to the seeded JWT auth spec, e.g. an auth/middleware-structure rule) under the conventions directory, so a correct plan run has something concrete to read and reflect.
- The **verifier** (`tests/harbor/plan-workflow/tests/test_plan_workflow.py`) gains: (a) a new hand-maintained per-step oracle asserting the discovery step reads conventions through the CLI (parallel to the existing `EXPECTED_SKILLS_PER_STEP` / per-step-window pattern); (b) `Conventions` added to `EXPECTED_PLAN_SECTIONS`; and (c) a content assertion that the conventions section reflects the distinctive seeded convention — the strong check that knowledge was digested, not merely that a command ran.
- Because that verifier uses **hand-maintained oracles as the independent behavioural check**, the new step order / section / skill expectations are updated in the same change as the template and state-machine changes — never derived at runtime (the test's own docstring requires this).

Known risks / uncertainty:

- Deciding *where* in the plan step sequence the `## Conventions` section is authored and assembled (a new dedicated step vs. folding capture into existing design steps) — to be resolved during planning. Note this also determines whether `EXPECTED_STEP_ORDER` in the harbor verifier changes.
- Migration ergonomics for projects whose conventions currently live in a flat `conventions.md`.

Relevant code: `internal/knowledge/set.go` (scope fan-out), `internal/store/search.go` (search impl), `cmd/knowledge.go` (subcommands), `templates/steps/plan/` (plan steps, embedded via `templates/templates.go`), `internal/project/init.go` (scaffold), `tests/harbor/plan-workflow/` (e2e environment + verifier).

## Success Metrics

- Plans created after this feature ships consistently carry a conventions section that reflects the project's established standards, rather than omitting them — the section is populated (or explicitly marked "none apply") on essentially every new plan.
- The conventions listed in a plan are judged relevant by a reviewer — the section reads as a tailored, justified shortlist, not a copy of the whole conventions folder.
- Implementations stay aligned with project conventions without the implementer consulting the knowledge base, because the relevant rules travel with the plan.
- Planning does not get slower or noisier as the topic-specific knowledge base grows, because only surface-relevant entries are pulled in.

## Non-Goals

- Automatically deciding *which* conventions are relevant without human review is out of scope — relevance is proposed during planning and confirmed by the user, not fully automated.
- Authoring or seeding the conventions themselves is out of scope — this feature governs how conventions are consulted and carried into plans, not what conventions a project should adopt.
- Changing how topic-specific knowledge is searched beyond making it surface-targeted and excluding the conventions area is out of scope — no new ranking, embeddings, or semantic search.
- Giving the implementation workflow any new knowledge-base access is explicitly out of scope; implementation continues to consume only the plan documents.
- Conventions that vary by workflow mode (e.g. separate plan-only vs. implement-only convention sets) are out of scope for this iteration.
