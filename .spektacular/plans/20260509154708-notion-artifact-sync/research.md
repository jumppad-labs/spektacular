# Research: 20260509154708-notion-artifact-sync

## Alternatives considered and rejected

### Option A: Implement a direct Notion API client inside the binary

The binary could authenticate to Notion, create databases, read pages, write pages, and manage conflicts without agent assistance.

**Rejected**: The user specifically asked to use Notion MCP, and the first-version scope explicitly keeps Notion authentication and workspace access in the MCP layer. The existing binary has no external service client pattern for this feature, and adding one would expand scope beyond artifact orchestration.

### Option B: Treat Notion as another `store.Store`

The existing `Store` interface could get a Notion implementation, letting current spec and plan steps keep calling read and write methods.

**Rejected**: `internal/store/store.go:14` is intentionally filesystem-shaped: it reads, writes, deletes, lists, and checks relative paths. Notion artifacts need database schema validation, required auto-increment identifiers, relations, remote page versions, child pages, merge requests, and conflict metadata. Forcing that into `Store` would hide the behavior this feature is meant to make explicit.

### Option C: Managed versus linked Notion modes

Spektacular could have one mode where it owns databases it creates and another mode where it only links to existing databases.

**Rejected**: The user rejected this model for teams and clarified that linked is how Spektacular should always work. Init and doctor can still create or repair missing structure, but all databases are treated as linked databases after setup.

### Option D: Commit Notion cache files in the existing local artifact folders

Notion-backed specs and plans could use `.spektacular/specs` and `.spektacular/plans` directly so existing code paths need fewer changes.

**Rejected**: The user wants local files to act as a cache and not be committed to git. Reusing the existing artifact folders would blur local-only artifacts with synced Notion cache content and make accidental commits more likely.

## Chosen approach - evidence

Config has a clear extension point. `internal/config/config.go:30` owns the top-level config shape, `internal/config/config.go:38` owns defaults, and `internal/config/config.go:71` owns validation. Adding an artifact backend block there matches current patterns.

The spec ID prerequisite is already implemented. `internal/steps/spec/identifier.go:53` accepts explicit external IDs even when another generated method is configured, and `internal/steps/spec/identifier.go:65` requires an ID in external mode. That supports the requirement that Notion-backed specs use a Notion-derived external ID.

The Notion MCP spike showed that the example database already provides the stable identity shape this feature should require. Specs expose a `Spec ID` auto-increment property and Plans expose a `Plan ID` auto-increment property, so those properties should be required rather than optional for Spektacular-compatible Notion schemas.

The existing local workflow is file-store oriented. `cmd/spec.go:186`, `cmd/plan.go:118`, and `cmd/plan.go:173` construct `FileStore` instances directly, while `internal/steps/spec/steps.go:63` and `internal/steps/plan/steps.go:178` write artifacts through that store. This points to an artifact layer above the store rather than replacing the store itself.

Workflow persistence can be a conflict risk. `internal/workflow/workflow.go:102` runs step callbacks after events, while `internal/workflow/workflow.go:125` persists state on entry. A Notion sync conflict should be detected before a transition is persisted or through a wrapper that prevents the transition from firing.

The Notion MCP spike showed the target workspace is workable. The base page exposed Specs and Plans databases, the Specs database had Status, Approval, Tags, relation, and auto-increment fields, and the Plans database had reciprocal Spec relation, compatible status fields, and auto-increment identity. Creating a Plan with the Spec relation set to the Spec page URL populated the reverse relation automatically. Spec content can live in the Spec page body. Plan content can live in the Plan page body, with Context and Research represented as child pages under the Plan row.

The current skill templates are local-only. `templates/skills/workflows/spek-new/SKILL.md:25`, `templates/skills/workflows/spek-plan/SKILL.md:25`, and `templates/skills/workflows/spek-implement/SKILL.md:25` all assume local artifact files. Updating skills is required because most users exercise Spektacular through agent skills; agents need enabled-mode detection, MCP operation instructions, external ID handling, pull-by-identity guidance, doctor approval prompts, cache sync instructions, and merge-resolution behavior.

## Files examined

- `AGENTS.md` - delegates repo-specific instructions to `.tessl/RULES.md`.
- `.tessl/RULES.md` - contains no extra operational rules beyond the managed include note.
- `.spektacular/specs/20260509154708-notion-artifact-sync.md` - source specification for this plan.
- `internal/config/config.go:13` - existing spec ID constants and config defaults.
- `internal/config/config.go:71` - current validation entry point.
- `internal/config/config_test.go:12` - current default and round-trip config tests to extend.
- `internal/project/init.go:23` - current init always creates local artifact directories.
- `internal/project/init_test.go:12` - current directory structure expectations to update or preserve per mode.
- `cmd/root.go:62` - root command registration point for a new Notion command group.
- `cmd/init.go:14` - existing init command and agent installation flow.
- `cmd/spec.go:147` - spec new command parses ID, loads config, resolves identifier, and creates workflow.
- `cmd/spec.go:186` - spec workflow currently uses `FileStore` directly.
- `cmd/spec_test.go:67` - tests document the spec schema and identifier behavior.
- `internal/steps/spec/identifier.go:39` - canonical name resolution with timestamp, counter, and external methods.
- `internal/steps/spec/identifier_test.go:23` - table coverage for timestamp, counter, external, normalization, and collisions.
- `internal/steps/spec/steps.go:63` - spec scaffold write point.
- `internal/steps/spec/steps.go:138` - final spec write point.
- `cmd/plan.go:68` - plan new command creates workflow state without creating documents.
- `cmd/plan.go:131` - plan goto command accepts workflow data and file uploads.
- `internal/steps/plan/steps.go:178` - plan.md write point.
- `internal/steps/plan/steps.go:195` - context.md write point.
- `internal/steps/plan/steps.go:212` - research.md write point.
- `internal/steps/plan/steps_test.go:67` - plan workflow order and write-step expectations.
- `cmd/implement.go:112` - implementation startup currently requires a local plan file before workflow creation.
- `internal/workflow/workflow.go:102` - step callbacks run after FSM events.
- `internal/workflow/workflow.go:125` - state persists on enter_state.
- `internal/store/store.go:14` - filesystem store contract.
- `internal/steps/spec/strategy.go:14` - spec instruction path variables.
- `internal/steps/plan/strategy.go:14` - plan instruction path variables.
- `internal/steps/implement/strategy.go:31` - implement instruction path variables.
- `templates/steps/spec/08-verification.md:1` - final spec submission instruction.
- `templates/steps/plan/13-verification.md:1` - final plan/context/research submission instruction.
- `templates/skills/workflows/spek-new/SKILL.md:25` - local spec workflow start instructions.
- `templates/skills/workflows/spek-plan/SKILL.md:25` - local plan workflow start instructions.
- `templates/skills/workflows/spek-implement/SKILL.md:25` - local implement workflow start instructions.
- `README.md:120` - local project layout docs.
- `README.md:138` - config and spec ID method docs.
- `README.md:166` - current testing documentation.
- `tests/harbor/spec-workflow/tests/test_spec_workflow.py` - Harbor coverage for spec workflow behavior.
- `tests/harbor/plan-workflow/tests/test_plan_workflow.py` - Harbor coverage for plan workflow behavior.

## External references

- Notion MCP enhanced markdown spec resource, `notion://docs/enhanced-markdown-spec` - used to understand the page content format returned by Notion MCP fetches.
- Product Engineering Home, `https://www.notion.so/onboard-ai-developer-user-guide/Product-Engineering-Home-359b746f14ca80749eebd9c4400ffb4d` - base page supplied by the user for testing and schema discovery.
- Specs data source, `collection://6fa340fe-3aeb-4822-8772-19745d300027` - existing compatible Notion Specs database from the spike.
- Plans data source, `collection://66115e12-ca84-4fb9-afa3-08fb5f74cbf5` - existing compatible Notion Plans database from the spike.
- Required Notion identity properties from the example workspace - `Spec ID` and `Plan ID` auto-increment fields, used as stable external identifiers.
- Spike spec page, `https://www.notion.so/35bb746f14ca81ada4b3e4ac56401b63` - throwaway MCP-created spec row, later marked Superseded.
- Spike plan page, `https://www.notion.so/35bb746f14ca811d9b10f0ab4ab0213c` - throwaway MCP-created plan row, later marked Superseded.

## Prior plans / specs consulted

- `.spektacular/specs/20260509154708-notion-artifact-sync.md` - primary source of requirements, constraints, acceptance criteria, and non-goals.
- Prior spec ID prefix work on branch `spec-id-prefix-method` and PR `https://github.com/jumppad-labs/spektacular/pull/4` - provides external ID behavior, normalization, counter state, and timestamp collision behavior that Notion mode must build on.

## Open assumptions

- Agents implementing this feature will have Notion MCP access when running Notion setup and sync flows.
- The binary can rely on agents to normalize MCP fetch results into stable schema and page-content snapshots before passing them back to CLI commands.
- Notion `last_edited_time` or equivalent metadata is available through MCP fetch/update results and is sufficient as the first-version remote baseline for conflict detection.
- The first version can represent spec content on the Spec page body, plan content on the Plan page body, and context/research as child pages under that Plan row, matching the successful spike and example database shape.
- Missing safe additive Notion schema pieces can be proposed by doctor, but agents should prompt the user before applying those changes.
- Live Notion integration does not need to run in default CI until a deterministic MCP fixture or fake can be introduced.

## Rehydration cues

To rehydrate this plan, read `.spektacular/specs/20260509154708-notion-artifact-sync.md`, then read config, init, spec command, plan command, implement command, workflow, store, and step strategy files listed above. Re-run the Notion MCP fetch against the Product Engineering Home page if schema details need to be refreshed. Re-check the prior spec ID prefix PR before changing identifier behavior.
