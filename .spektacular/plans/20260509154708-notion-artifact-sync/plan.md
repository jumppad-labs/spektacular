# Plan: 20260509154708-notion-artifact-sync

<!-- Metadata -->
<!-- Created: 2026-05-09T16:23:52Z -->
<!-- Commit: b7f6640 -->
<!-- Branch: notion-artifact-sync-spec -->
<!-- Repository: https://github.com/gregoryhunt/spektacular.git -->
<!-- Slug: 0020-notion-artifact-sync -->

## Overview

Spektacular will add a Notion-backed artifact mode for specifications, plans, context, and research while preserving the existing local-only workflow. The shared Notion pages remain the team-visible source of truth, and the local cache remains the agent-friendly working copy with conflict metadata. This gives humans and agents a safer way to collaborate on the same planning artifacts without committing synced cache files to source control.

## Architecture & Design Decisions

The chosen architecture is an MCP-orchestrated Notion backend with a local cache and manifest. The Spektacular binary owns configuration, workflow state, schema rules, cache paths, local checksums, identity metadata, and conflict decisions; the agent performs Notion MCP operations only when the CLI requests external reads, writes, database creation, schema snapshots, or approved repairs. This keeps Notion authentication out of the binary while still making the binary the deterministic source of workflow behavior.

Notion setup is always linked. Existing databases and databases created during setup both flow through the same validator, so there is no separate managed mode with surprising ownership semantics. The setup commands provide init, link, and doctor flows: init helps create missing databases through MCP, link validates existing databases before config is written, and doctor reports safe additive fixes. Applying those fixes follows the standard migration pattern: show the planned changes, ask the user or agent for explicit approval, then perform only the approved MCP updates.

The local cache is a first-class artifact layer rather than a replacement for the current filesystem store. Local mode continues to use the existing specs and plans layout, while Notion mode uses a separate ignored cache namespace plus a manifest recording remote URLs, Notion page IDs, data source IDs, Spektacular external IDs, remote edit versions, and local checksums. Workflow steps keep the same user-visible cadence as the filesystem workflow: when a step creates, updates, or finalizes an artifact locally, Notion mode syncs the cache and manifest before returning the next instruction. If the remote artifact changed since the cache baseline, the step enters a merge flow with the baseline, local copy, and remote copy instead of advancing.

Rejected options and evidence are captured in [research.md#alternatives-considered-and-rejected](./research.md#alternatives-considered-and-rejected). The main trade-off is that first-version Notion support depends on capable agents with MCP access, but it avoids embedding a second Notion auth stack into the binary and matches the user's requested operating model.

## Component Breakdown

- Configuration owns the artifact backend selection, cache directory, linked Notion locations, and required external ID mode. It also preserves existing local defaults when no artifact block is present.
- Notion setup owns init, link, doctor, schema validation, required auto-increment identifier fields, and user-approved safe additive repairs. It produces structured instructions for MCP work, validates schema snapshots supplied back by the agent, and writes config only after validation passes.
- Artifact cache owns path selection, manifest load/save, dirty detection, identity lookup, and conflict checks. It is deliberately separate from the existing file store so Notion metadata does not leak into generic filesystem reads and writes.
- Merge flow owns stale-cache resolution. It presents the cache baseline, local content, and remote Notion content, then records the resolved content only after the user or agent explicitly chooses the merge outcome.
- Workflow integration owns the points where spec, plan, and implement commands select local artifacts versus Notion cache artifacts. It ensures artifact-changing steps sync at the same cadence as the filesystem workflow and ensures implementation startup has a current local plan cache.
- Skill and template updates own the agent-facing protocol. They teach agents when Notion mode is enabled, when to call Notion MCP, when to submit schema snapshots, when to sync changed cache files, how to pull existing work by identity, and how to surface merge conflicts to users.
- Documentation and validation tests own the user-facing contract. They make the local workflow, Notion setup workflow, and conflict behavior understandable and regression-tested.

## Data Structures & Interfaces

The config contract adds an artifact block and keeps spec identifier behavior explicit:

```go
type ArtifactsConfig struct {
    Backend  string
    CacheDir string
    Notion   NotionConfig
}

type NotionConfig struct {
    BasePageURL     string
    SpecsDataSource string
    PlansDataSource string
    SpecIDProperty  string
    PlanIDProperty  string
}
```

Schema validation uses a stable CLI-facing snapshot rather than parsing raw MCP markdown. Agents can build the snapshot from MCP fetch results, and the binary validates the contract:

```go
type DataSourceSnapshot struct {
    URL        string
    Properties map[string]PropertySnapshot
}

type SchemaReport struct {
    Status   string
    Fixable  []SetupIssue
    Blocking []SetupIssue
}
```

The cache manifest records local and remote baselines for each synced artifact:

```go
type ArtifactRef struct {
    Kind             string
    Name             string
    LocalPath        string
    RemoteURL        string
    RemotePageID     string
    DataSourceURL    string
    ExternalID       string
    RemoteLastEdited string
    LocalChecksum    string
}

type Manifest struct {
    Version   int
    Artifacts []ArtifactRef
}
```

Workflow and sync commands exchange structured outcomes so agents can react without scraping text:

```go
type SyncResult struct {
    Status            string
    ChangedArtifacts  []ArtifactRef
    ResolutionNeeded  []ResolutionNeed
}
```

Status values should stay small and stable: local, synced, stale, conflict, resolution_required, and not_configured.

The merge contract should expose enough context for agent skills to guide the user:

```go
type MergeRequest struct {
    Artifact ArtifactRef
    Base     string
    Local    string
    Remote   string
}
```

## Implementation Detail

The implementation introduces an artifact service boundary above the current store. Local mode keeps using the existing file store directly, while Notion mode resolves cache paths, manifest entries, and sync requirements before the workflow code reads or writes artifacts. This keeps most existing workflow logic intact while giving Notion mode a place to enforce remote metadata and conflicts.

Setup commands should be modeled as deterministic CLI workflows rather than hidden API clients. When the binary needs Notion information, it emits structured instructions asking the agent to call MCP and return a normalized snapshot. When the agent returns that snapshot, the binary validates it and writes config or emits a report. The same validator powers init, link, and doctor, and doctor apply should follow an industry-standard migration posture: report planned changes first, require explicit approval, apply only safe additive changes, then validate again.

The Notion content layout follows the example workspace. The Spec database page body stores the spec markdown and exposes a required Spec ID auto-increment property. The Plan database page body stores plan.md, exposes a required Plan ID auto-increment property, relates back to the Spec, and owns Context and Research child pages.

Step completion needs special care because the workflow currently persists state as it enters a step. Notion conflict checks that can block advancement should run before the workflow transitions or through a command wrapper that validates sync readiness before calling the FSM event. The implementation should avoid a state file that says a step advanced when the remote sync was rejected, and state or manifest metadata should retain active Notion page IDs and external IDs so other agents can pull existing work by shared identity.

The skill changes are part of the feature, not documentation polish. In Notion mode, the skills must tell agents to create or fetch Notion pages through MCP, submit schema snapshots to the binary, submit changed artifact content through the cache/sync commands, pull existing work by URL/page ID/external ID, and stop on doctor approval or merge-resolution prompts.

## Dependencies

- Existing spec identifier work: provides timestamp, counter, external ID behavior, normalization, and external ID enforcement; this plan depends on that branch landing first.
- Existing config package: provides YAML load, defaults, env expansion, and validation; it needs new artifact configuration and Notion-specific validation.
- Existing workflow package: provides state transitions and command result flow; it needs a pre-transition or wrapper path for sync blocking.
- Existing store package: provides safe local filesystem reads and writes; it remains the local backend and underpins the Notion cache.
- Existing spec, plan, and implement commands: provide the workflow entry points that need artifact backend resolution and cache checks.
- Notion MCP: provides actual Notion database and page operations; the binary should orchestrate it through agent instructions instead of importing a Notion client.
- Notion auto-increment properties: provide the required Spec ID and Plan ID values used as stable external identifiers and lookup keys.
- Harbor and Go tests: provide current regression coverage for workflows and CLI behavior; they need Notion-mode unit and scenario coverage.

## Testing Approach

Unit tests should cover config defaults and validation, required auto-increment schema validation, safe additive doctor behavior with approval, manifest read/write, identity lookup, checksum changes, stale remote detection, merge request generation, and Notion-required external spec IDs. Command tests should cover init, link, doctor, pull, push, merge completion, and the workflow entry points in both local and Notion modes.

Workflow tests should focus on the load-bearing behavior: local mode remains unchanged, Notion mode writes to cache paths, changed artifacts update the manifest, conflicts produce resolution-required output, and implementation startup refuses to proceed without a usable plan cache. Skill/template tests should assert that installed agent skills include the Notion MCP and sync instructions when relevant.

Harbor coverage should stay local-first for the existing scenarios and add at least one Notion-mode dry or fake-MCP scenario that verifies the agent follows the new sync protocol. Direct live Notion integration should remain manual or fixture-backed until the MCP contract can be exercised deterministically in CI.

## Milestones & Phases

### Milestone 1: Projects can be configured for linked Notion artifacts

**What changes**: Users can tell Spektacular that artifacts live in Notion, validate existing databases, and initialize missing shared structure through a linked setup flow. Existing local projects continue to behave as they do today.

#### - [x] Phase 1.1: Configure Notion artifacts

Add the artifact backend configuration and make Notion mode require external spec identifiers. Local mode remains the default, and projects that do not opt into Notion see no behavior change.

*Technical detail:* [context.md#phase-11-configure-notion-artifacts](./context.md#phase-11-configure-notion-artifacts)

**Acceptance criteria**:

- [x] A new default config keeps local artifacts and timestamp spec IDs.
- [x] A Notion artifact config validates only when linked Notion locations and required Spec ID/Plan ID properties are present.
- [x] A Notion artifact config fails validation unless spec IDs use the external method.
- [x] Existing config files without an artifact block continue to load.
- [x] Notion-mode initialization avoids creating local-only specs and plans artifact folders.

#### - [x] Phase 1.2: Validate and prepare linked Notion databases

Add the Notion setup command group and shared schema validator. Users can validate existing databases, receive fixable versus blocking issue reports, and initialize missing databases through MCP-guided steps.

*Technical detail:* [context.md#phase-12-validate-and-prepare-linked-notion-databases](./context.md#phase-12-validate-and-prepare-linked-notion-databases)

**Acceptance criteria**:

- [x] Linking compatible Specs and Plans databases with required auto-increment ID fields writes project config after validation passes.
- [x] Linking incompatible databases reports every blocking issue, including missing required ID fields, and leaves the project unconfigured.
- [x] Doctor reports fixable and blocking issues separately.
- [x] Doctor apply presents safe additive changes for approval, applies only approved changes, and a follow-up validation can pass.

### Milestone 2: Notion artifacts have a safe local working cache

**What changes**: Agents can pull Notion artifacts into an ignored local cache, edit the cache, and push changes only when the remote artifact has not changed since the local baseline. Conflicts are explicit and do not overwrite another user's work.

#### - [x] Phase 2.1: Add artifact cache and manifest

Introduce cache paths and manifest metadata for specs, plans, context, and research. The cache tracks local content checksums and remote versions so commands can distinguish clean, dirty, stale, and conflicting artifacts.

*Technical detail:* [context.md#phase-21-add-artifact-cache-and-manifest](./context.md#phase-21-add-artifact-cache-and-manifest)

**Acceptance criteria**:

- [x] Notion-backed specs and plans resolve to cache paths outside the local artifact directories.
- [x] Pulling an artifact records its remote URL, Notion page ID, data source URL, external ID, remote edit version, local path, and checksum.
- [x] Pulling can locate existing work by Notion URL, Notion page ID, or Spektacular external ID.
- [x] Editing a cached artifact changes its dirty status without modifying the remote baseline.
- [x] The cache path is ignored by default.

#### - [x] Phase 2.2: Add pull, push, and conflict contracts

Add CLI contracts that let agents update cache files from MCP reads, prepare safe MCP writes, and commit returned remote metadata after a successful write. Conflict responses include the exact artifact and required resolution action.

*Technical detail:* [context.md#phase-22-add-pull-push-and-conflict-contracts](./context.md#phase-22-add-pull-push-and-conflict-contracts)

**Acceptance criteria**:

- [x] Pull updates a cache file and manifest from agent-supplied Notion content and metadata.
- [x] Push preparation returns a merge request when the supplied remote version does not match the manifest baseline.
- [x] Merge completion records the resolved content and only then allows retrying the Notion update.
- [x] A successful push commit updates the manifest to the new remote version.
- [x] Conflict output is structured enough for an agent to present the resolution need without guessing.

### Milestone 3: Workflows and skills use Notion artifacts end to end

**What changes**: Spec, plan, and implement workflows can operate against Notion-backed artifacts through the cache and sync layer. The installed skills tell agents how to use Notion MCP and when to stop for conflicts.

#### - [x] Phase 3.1: Route workflow artifact changes through sync

Update spec, plan, and implement workflow commands so Notion mode creates or refreshes cache entries, syncs changed artifacts before returning the next instruction, and blocks step advancement on conflicts.

*Technical detail:* [context.md#phase-31-route-workflow-artifact-changes-through-sync](./context.md#phase-31-route-workflow-artifact-changes-through-sync)

**Acceptance criteria**:

- [x] Notion-backed spec creation uses the required Notion Spec ID auto-increment value as the external identifier and creates a matching cache entry.
- [x] Notion-backed plan creation links to the spec and creates plan, context, and research cache entries.
- [x] Each successful artifact-changing step syncs at the same user-visible point as the filesystem workflow and leaves cache content and manifest metadata aligned with the latest remote version.
- [x] Implementation startup prepares or rejects the Notion-backed plan cache before the implementation workflow reads it.

#### - [x] Phase 3.2: Update skills, docs, and regression coverage

Update the embedded and repo-local skills, README guidance, and regression tests so agents and users understand the Notion setup and sync protocol. Existing local workflow coverage remains in place.

*Technical detail:* [context.md#phase-32-update-skills-docs-and-regression-coverage](./context.md#phase-32-update-skills-docs-and-regression-coverage)

**Acceptance criteria**:

- [x] Installed skills describe Notion MCP setup, enabled-mode detection, pull-by-identity, external IDs, per-step sync responsibilities, doctor approval, and merge resolution.
- [x] README documents local mode, Notion mode, link, init, doctor, cache, and conflict behavior.
- [x] Existing local workflow tests still pass.
- [x] New Notion-mode tests cover config, setup validation, cache sync, conflict output, and workflow startup behavior.

## Open Questions

None are intentionally carried into implementation. If the observed Notion MCP fetch or update payload shape differs from the spike enough that it cannot be normalized into the planned snapshot contracts, implementation should stop and adjust the contract before changing workflow behavior.

## Out of Scope

- Syncing knowledge-base content to or from Notion is out of scope for this plan.
- Syncing implementation artifacts, generated code, test output, or build artifacts to Notion is out of scope.
- Embedding direct Notion API authentication or an HTTP Notion client in the binary is out of scope for the first version.
- Automatically merging conflicting local and remote edits is out of scope; this plan detects and blocks conflicts.
- Deleting, renaming, or changing incompatible existing Notion database properties automatically is out of scope.
- Replacing the current local-only filesystem workflow is out of scope.
- Building a general-purpose Notion database migration framework is out of scope.

## Changelog

### 2026-05-09 — Phase 1.1: Configure Notion artifacts

**What was done**: Added artifact backend configuration with local defaults, Notion cache settings, linked data source fields, required Spec ID/Plan ID property names, and validation that Notion mode requires external spec IDs. Added a mode-aware project initializer that future Notion setup commands can use to create the project metadata and Notion cache without creating local-only spec and plan artifact folders.

**Deviations**: None. The regular `init <agent>` command remains local-only by design; Notion setup uses the dedicated Notion command flow.

**Files changed**:
- `internal/config/config.go`
- `internal/config/config_test.go`
- `internal/project/init.go`
- `internal/project/init_test.go`

**Discoveries**: `InitWithOptions` needs a fully valid Notion config when writing Notion mode, because normal config loading now validates required data source and identifier-property fields. That fits the planned `notion init/link` flow, which should write Notion config only after schema validation has returned the linked data source IDs.

### 2026-05-09 — Phase 1.2: Validate and prepare linked Notion databases

**What was done**: Added a `notion` command group with `init`, `link`, `doctor`, and `status` subcommands. Added Notion data source snapshot parsing, linked Specs/Plans schema validation, blocking versus fixable issue reports, safe additive repair preparation, and config writing only after compatible snapshots validate. The commands accept JSON snapshots from Notion MCP directly or through snapshot files, and Notion mode initializes the project cache without creating local specs/plans artifact folders.

**Deviations**: None.

**Files changed**:
- `cmd/notion.go`
- `cmd/notion_test.go`
- `cmd/root.go`
- `internal/notion/schema/parse.go`
- `internal/notion/schema/schema.go`
- `internal/notion/schema/schema_test.go`
- `internal/notion/setup/instructions.go`

**Discoveries**: Notion MCP fetch output can be passed through as a wrapped JSON/text payload containing `<data-source-state>`, so the validator supports both direct snapshots and MCP fetch wrappers. The example databases use `auto_increment_id` in fetched schema output for `Spec ID` and `Plan ID`, while MCP create DDL represents those fields as `UNIQUE_ID`. Because the project uses Notion MCP as the integration boundary, `doctor --apply` prepares approved MCP update instructions and patched local snapshots rather than mutating Notion directly.

### 2026-05-09 — Phase 2.1: Add artifact cache and manifest

**What was done**: Added the `internal/artifact` package with artifact kind constants, local versus Notion cache path resolution, checksum calculation, manifest load/save, pull recording, identity lookup, and cache status checks for dirty and stale artifacts. Updated the embedded `.spektacular/.gitignore` template so `cache/notion/` and transient sync files are ignored by default.

**Deviations**: None. Workflow path routing was intentionally deferred to Phase 3.1 after the cache and manifest primitives existed.

**Files changed**:
- `internal/artifact/artifact.go`
- `internal/artifact/artifact_test.go`
- `internal/project/init_test.go`
- `templates/.spektacular/.gitignore`

**Discoveries**: Keeping the manifest store-relative lets the existing `FileStore` handle all path safety checks, while artifact names still need explicit separator validation before cache path construction.

### 2026-05-09 — Phase 2.2: Add pull, push, and conflict contracts

**What was done**: Added an `internal/artifact/sync` package and `notion cache` subcommands for `pull`, `status`, `prepare-push`, `commit-push`, and `resolve-merge`. Pull records Notion MCP content and metadata in the cache and manifest, prepare-push returns either clean/ready status or a structured merge request, resolve-merge records remote baseline plus resolved local content, and commit-push records the returned remote version after a successful Notion update.

**Deviations**: None.

**Files changed**:
- `cmd/notion_cache.go`
- `cmd/notion_cache_test.go`
- `internal/artifact/artifact.go`
- `internal/artifact/artifact_test.go`
- `internal/artifact/sync/sync.go`
- `internal/artifact/sync/sync_test.go`

**Discoveries**: Merge requests need a baseline content copy, not only a checksum. The manifest now tracks `baseline_path`, and pull/commit/merge resolution keep that baseline separate from the editable local cache file. The commands remain binary-local contracts around agent-supplied MCP content and metadata, so the binary does not call Notion directly.

### 2026-05-09 — Phase 3.1: Route workflow artifact changes through sync

**What was done**: Extended workflow runtime config with the loaded project config and added config-aware path strategies so spec, plan, and implement workflows emit cache paths in Notion mode while local mode remains unchanged. Spec creation now uses the Notion-provided external ID, writes the scaffold to cache, and records a manifest entry; plan write steps write plan/context/research cache entries when remote metadata is supplied; implement startup checks the Notion plan cache path instead of the local plan path.

**Deviations**: None.

**Files changed**:
- `cmd/implement.go`
- `cmd/notion_workflow_test.go`
- `cmd/plan.go`
- `cmd/spec.go`
- `internal/artifact/artifact.go`
- `internal/artifact/sync/sync.go`
- `internal/stepkit/stepkit.go`
- `internal/steps/implement/strategy.go`
- `internal/steps/plan/steps.go`
- `internal/steps/plan/strategy.go`
- `internal/steps/spec/steps.go`
- `internal/steps/spec/strategy.go`
- `internal/workflow/workflow.go`

**Discoveries**: The least invasive path integration is an optional config-aware strategy interface in `stepkit`. It keeps existing local strategy behavior intact while allowing Notion mode to resolve the same artifact names to cache paths. Notion-backed artifact-changing steps require the agent to supply returned Notion metadata so the binary can record cache and manifest baselines while keeping MCP as the Notion integration boundary.

### 2026-05-09 — Phase 3.2: Update skills, docs, and regression coverage

**What was done**: Updated the embedded workflow skills, repo-local installed skills, final spec/plan verification templates, and README with Notion mode setup and sync guidance. Added regression coverage that installed skills include Notion mode instructions, while the earlier Notion tests cover config validation, schema validation, cache sync, conflict output, and workflow startup behavior.

**Deviations**: Harbor remains covered by the existing local workflow scenarios; Notion-mode coverage is command/unit level because live MCP interaction is fixture/manual until deterministic MCP harness support exists.

**Files changed**:
- `.agents/skills/spek-implement/SKILL.md`
- `.agents/skills/spek-new/SKILL.md`
- `.agents/skills/spek-plan/SKILL.md`
- `README.md`
- `cmd/init_test.go`
- `templates/skills/workflows/spek-implement/SKILL.md`
- `templates/skills/workflows/spek-new/SKILL.md`
- `templates/skills/workflows/spek-plan/SKILL.md`
- `templates/steps/plan/13-verification.md`
- `templates/steps/spec/08-verification.md`

**Discoveries**: The installed repo-local skills are rendered copies, not symlinks to the templates, so both the embedded templates and `.agents/skills` copies must be updated when command flow changes.
