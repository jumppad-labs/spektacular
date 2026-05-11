# Context: 20260509154708-notion-artifact-sync

## Current State Analysis

The current configuration is intentionally small. `internal/config/config.go:13` defines the existing spec ID methods, `internal/config/config.go:24` defines `SpecConfig`, `internal/config/config.go:30` defines top-level `Config`, and `internal/config/config.go:38` sets defaults to timestamp IDs. Validation currently only checks `spec.id_method` at `internal/config/config.go:71`, so artifact backend validation has a clear home.

Project initialization always creates the local artifact folders. `internal/project/init.go:23` builds `.spektacular`, `plans`, `specs`, and `knowledge` directories on every init, and `internal/project/init.go:38` writes default config only if missing. The Notion mode needs to avoid creating local artifact folders when the project is initialized directly into Notion artifact mode, while preserving existing local behavior.

The current store abstraction is filesystem-shaped. `internal/store/store.go:14` defines read, write, delete, list, and exists operations over relative paths, and `internal/store/store.go:32` implements them with `FileStore`. This is a good local cache primitive, but it is not a good direct Notion abstraction because Notion has database schemas, required auto-increment identifiers, relations, remote versions, and conflict metadata.

Spec creation already has the identifier behavior Notion mode needs. `internal/steps/spec/identifier.go:39` resolves canonical names, `internal/steps/spec/identifier.go:53` lets an explicit ID override generated methods, `internal/steps/spec/identifier.go:65` requires an ID for external mode, `internal/steps/spec/identifier.go:77` rejects trimming while normalizing punctuation and symbols, and `internal/steps/spec/identifier.go:129` and `internal/steps/spec/identifier.go:149` implement timestamp and counter generation.

The CLI commands currently create file stores directly. `cmd/spec.go:186` creates a `FileStore` for spec workflows and `cmd/spec.go:192` resolves the spec identifier against that store. `cmd/plan.go:118` and `cmd/plan.go:173` create `FileStore` instances for plan workflows. `cmd/implement.go:112` requires a local plan file before implementation starts, which will fail for a Notion-backed plan until cache preparation happens first.

Workflow state persistence needs careful handling. `internal/workflow/workflow.go:102` invokes step callbacks after transitions, while `internal/workflow/workflow.go:125` persists state on `enter_state`. If a Notion sync conflict is discovered too late, the state may already show the next step. The safest implementation is to perform sync validation before firing the transition or wrap workflow advancement with a Notion preflight that can block cleanly.

Spec and plan step callbacks write local files at their terminal write points. `internal/steps/spec/steps.go:63` creates the initial spec scaffold, and `internal/steps/spec/steps.go:138` writes the final spec template. `internal/steps/plan/steps.go:178`, `internal/steps/plan/steps.go:195`, and `internal/steps/plan/steps.go:212` write plan, context, and research content from workflow data. Notion mode should preserve the same user-visible workflow cadence, using cache files where the filesystem workflow uses artifact files and syncing Notion at the same artifact-changing points.

Path strategies currently assume local filesystem artifact paths. `internal/steps/spec/strategy.go:14`, `internal/steps/plan/strategy.go:14`, and `internal/steps/implement/strategy.go:31` build absolute paths from store root and local path conventions. Notion mode should keep returning local cache paths for agent editing and add remote metadata in result fields rather than replacing paths with Notion URLs.

Agent skills currently describe local artifact workflows. `templates/skills/workflows/spek-new/SKILL.md:25` starts spec creation, `templates/skills/workflows/spek-plan/SKILL.md:25` starts plan creation, and `templates/skills/workflows/spek-implement/SKILL.md:25` requires the plan file to exist locally. These templates need Notion-aware instructions for enabled-mode detection, MCP calls, external IDs, pull-by-identity, cache sync, doctor approval, and merge resolution; the repo-local `.agents/skills` copies need to be regenerated or updated.

The README documents local project layout and spec ID methods at `README.md:120` through `README.md:156`. That section needs to grow to describe artifact backends, Notion setup commands, the ignored cache, and conflict behavior.

## Per-Phase Technical Notes

### Phase 1.1: Configure Notion Artifacts

- `internal/config/config.go:13` - add artifact backend constants, likely `local` and `notion`, next to the existing config constants.
- `internal/config/config.go:24` - keep `SpecConfig` as-is but validate external ID when Notion is selected.
- `internal/config/config.go:30` - add `Artifacts ArtifactsConfig` to top-level `Config`.
- `internal/config/config.go:38` - default artifacts to local backend and a cache directory under `.spektacular/cache/notion`.
- `internal/config/config.go:71` - extend validation to reject unsupported artifact backends, missing Notion data source URLs in Notion mode, missing required Spec ID/Plan ID property configuration, and Notion mode without `spec.id_method: external`.
- `internal/config/config_test.go:12` - extend default, load, missing-field, unknown-value, and round-trip tests for artifact config.
- `internal/project/init.go:23` - add an options struct or mode-aware path so direct Notion initialization can create the project root and cache path without generating local `specs` and `plans` folders.
- `cmd/init.go:14` - add an artifact/backend flag only if regular `init` needs to initialize directly into Notion mode; otherwise keep regular init local and let the Notion command update config.

**Complexity**: Medium
**Token estimate**: ~18k
**Agent strategy**: Single agent for config and init changes, with tests immediately after because these files are shared by every command.

### Phase 1.2: Validate and Prepare Linked Notion Databases

- `cmd/root.go:62` - register a new Notion command group.
- New `cmd/notion.go` - add init, link, doctor, and status subcommands. Keep command output structured using existing schema/result patterns from `cmd/plan.go:16` and `cmd/spec.go:46`.
- New `internal/notion/schema` package - define data source snapshots, required property rules, required auto-increment identifier fields, fixable issues, blocking issues, and report formatting.
- New `internal/notion/setup` package or equivalent - convert schema reports into agent instructions for MCP create/update/fetch operations, including explicit approval before applying safe additive repairs.
- `cmd/spec.go:108` - reuse `readInputIntoWorkflow` patterns or factor a common file/stdin reader for schema snapshots returned from MCP.
- `internal/config/config.go:89` - write updated config only after link/init validation passes.
- `internal/config/config_test.go:50` - add invalid Notion config tests and valid Notion config tests.
- New command tests - cover compatible snapshots, missing required ID fields, missing fixable properties, incompatible property types, config write after success, explicit repair approval, and no config write after failure.

**Complexity**: High
**Token estimate**: ~35k
**Agent strategy**: Parallel analysis is useful: one agent can define schema/report contracts while another sketches CLI command result shapes. Integration should be sequential to keep result formats consistent.

### Phase 2.1: Add Artifact Cache and Manifest

- New `internal/artifact` package - define artifact kind constants for spec, plan, context, and research, cache path resolution, checksum calculation, identity lookup, manifest loading, and manifest saving.
- `internal/store/store.go:14` - continue using `FileStore` for actual local cache writes instead of making Notion implement this interface.
- `internal/steps/spec/steps.go:11` and `internal/steps/plan/steps.go:11` - preserve existing local paths and add artifact-layer helpers that return cache paths in Notion mode.
- `internal/steps/spec/strategy.go:14`, `internal/steps/plan/strategy.go:14`, and `internal/steps/implement/strategy.go:31` - route path variables through an artifact path resolver so instruction output points agents at the right local working copy.
- `templates/.spektacular/.gitignore` - ensure the Notion cache directory and transient sync files are ignored.
- New manifest tests - cover empty manifest creation, artifact upsert, Notion URL/page ID/external ID lookup, checksum changes, dirty status, stale remote status, and path traversal rejection.

**Complexity**: Medium
**Token estimate**: ~28k
**Agent strategy**: Single agent, sequential execution. This package should be small and well-covered before workflow integration starts.

### Phase 2.2: Add Pull, Push, and Conflict Contracts

- New `cmd/artifact.go` or Notion subcommands - add cache update, prepare push, commit push, status, merge completion, and pull-result commands. Names can live under `notion cache` if that reads better in CLI help.
- New `internal/artifact/sync` package - compare manifest remote baselines with agent-supplied remote metadata and return structured sync results or merge requests containing baseline, local, and remote content.
- `cmd/spec.go:46`, `cmd/plan.go:16`, and `cmd/implement.go:20` - extend result schemas only if workflow outputs need sync status fields; keep backward compatibility for local mode.
- New result structs - include status, changed artifacts, resolution needs, and merge request payloads.
- Command tests - cover successful pull, pull by Notion URL/page ID/external ID, dirty local detection, safe push preparation, push commit, stale remote merge request, merge completion, missing manifest entry, and malformed remote metadata.
- Notion MCP instructions - define the normalized fields the agent must return after MCP fetch/update: page URL, page ID, data source URL, external ID, last edited timestamp, title, relation URLs, and markdown content.

**Complexity**: High
**Token estimate**: ~40k
**Agent strategy**: 2 agents can work in parallel if write scopes are split between command wiring and internal sync logic. Integration and error text should be reviewed sequentially.

### Phase 3.1: Route Workflow Artifact Changes Through Sync

- `cmd/spec.go:177` - load artifact config before store selection and select the right artifact service.
- `cmd/spec.go:192` - in Notion mode, create the Notion Spec page first, read its required Spec ID auto-increment value, then pass that value through existing external identifier resolution.
- `internal/steps/spec/steps.go:63` - in Notion mode, create or cache the Notion-backed scaffold in the Spec page body and manifest entry before returning overview.
- `internal/steps/spec/steps.go:138` - sync final spec content through the artifact service before emitting finished output.
- `cmd/plan.go:99` and `cmd/plan.go:161` - resolve Notion-backed spec and plan cache state before workflow creation and before each artifact-changing step, including lookup by existing external ID when another agent picks up work.
- `internal/steps/plan/steps.go:178`, `internal/steps/plan/steps.go:195`, and `internal/steps/plan/steps.go:212` - route plan, context, and research writes through the artifact service in Notion mode; plan.md maps to the Plan page body, while context.md and research.md map to child pages under the Plan page.
- `cmd/implement.go:112` - replace the raw local file precondition with artifact-cache preparation for Notion-backed plans.
- `internal/workflow/workflow.go:102` and `internal/workflow/workflow.go:125` - avoid discovering conflicts after state persistence. Prefer a pre-transition sync guard in command code or a workflow callback extension that runs before state is persisted.
- Workflow tests - cover unchanged local mode, Notion cache creation, sync success at the filesystem-equivalent step cadence, merge requests that block advancement, and implementation startup cache checks.

**Complexity**: High
**Token estimate**: ~50k
**Agent strategy**: Parallel analysis for spec/plan/implement command surfaces, then sequential integration. This phase touches shared workflow behavior and should be kept behind artifact backend branches.

### Phase 3.2: Update Skills, Docs, and Regression Coverage

- `templates/skills/workflows/spek-new/SKILL.md:25` - add Notion mode behavior: detect enabled mode, use Notion MCP to create/fetch a spec artifact, pass the required Spec ID external identifier, and sync changed cache before each applicable step advancement.
- `templates/skills/workflows/spek-plan/SKILL.md:25` - add Notion mode behavior: ensure spec cache exists, create/link a plan row with Plan ID, create context and research child pages, and sync changed artifacts.
- `templates/skills/workflows/spek-implement/SKILL.md:25` - update the local plan precondition to allow Notion cache preparation or pull-by-identity before implementation starts.
- `.agents/skills/spek-new/SKILL.md`, `.agents/skills/spek-plan/SKILL.md`, and `.agents/skills/spek-implement/SKILL.md` - update repo-local installed skills so this checkout uses the new instructions during implementation.
- `templates/steps/spec/08-verification.md:1` and `templates/steps/plan/13-verification.md:1` - add Notion-specific sync reminders around final artifact submission where needed.
- `README.md:120` - document local layout and Notion cache layout.
- `README.md:138` - document artifact config and external ID requirement.
- `README.md:166` - document which validation suites cover local mode versus Notion mode.
- Harbor tests - keep existing local workflow tests passing and add a fake or dry Notion scenario for the agent sync protocol.

**Complexity**: Medium
**Token estimate**: ~30k
**Agent strategy**: 2 agents can work in parallel if one owns docs/skills and one owns tests. Final verification should be sequential because templates and tests must agree on command names.

## Testing Strategy

Config tests should prove local defaults remain unchanged and Notion config is strictly validated. Schema validation tests should be table-driven with compatible, fixable, and blocking database snapshots, including missing required auto-increment identifier properties. Artifact cache tests should use temporary directories and deterministic remote metadata so dirty, stale, lookup, merge, and conflict states are easy to assert.

Command tests should follow existing patterns in `cmd/spec_test.go` and `cmd/init_test.go`, using test helpers that reset Cobra flags between cases. Workflow tests should exercise both local mode and Notion mode so the current local path behavior remains protected.

Harbor should continue validating the existing local spec and plan workflows. Add Notion protocol coverage only where it can be deterministic without a live Notion account; live MCP checks can remain manual until the MCP interaction can be fixture-backed.

## Project References

- Spec: `.spektacular/specs/20260509154708-notion-artifact-sync.md`
- Current branch: `notion-artifact-sync-spec`
- Prior PR this builds on: `https://github.com/jumppad-labs/spektacular/pull/4`
- Notion base page used for spike: `https://www.notion.so/onboard-ai-developer-user-guide/Product-Engineering-Home-359b746f14ca80749eebd9c4400ffb4d`
- Existing Specs data source from spike: `collection://6fa340fe-3aeb-4822-8772-19745d300027`
- Existing Plans data source from spike: `collection://66115e12-ca84-4fb9-afa3-08fb5f74cbf5`
- Required identity properties from example schema: `Spec ID` and `Plan ID`

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|-------------|----------------|
| Low | ~10k | Single agent, sequential |
| Medium | ~25k | 2-3 parallel agents with disjoint write scopes |
| High | ~50k+ | Parallel analysis, sequential integration |

Phase 1.1 and Phase 2.1 are good single-agent phases. Phase 1.2, Phase 2.2, and Phase 3.1 are high risk because they introduce new command contracts and workflow behavior. Phase 3.2 can be split if command names and result shapes are already stable.

## Migration Notes

Existing projects without an `artifacts` config block should load as local mode. Existing `.spektacular/specs` and `.spektacular/plans` content should not move automatically. Opting into Notion should be explicit through link or init, and any cache files created during Notion mode should remain ignored.

Projects that already have a config file should preserve command, agent, debug, and spec fields when Notion setup writes artifact fields. Notion mode should set or require `spec.id_method: external`; it should not silently change the user's configured ID method outside the setup flow.

## Performance Considerations

Local mode should have no measurable performance change beyond config parsing a larger struct. Notion mode performance is dominated by MCP calls, so the binary should minimize redundant remote fetches by using manifest baselines and only requesting remote metadata for artifacts that are about to sync.

Manifest operations should be small JSON reads and writes. Cache checksum calculation is proportional to artifact size, which is acceptable for markdown spec and plan documents.
