# Feature: 20260509154708-notion-artifact-sync

<!--
  OVERVIEW
  A concise 2-3 sentence summary of the feature. Answer three questions:
    1. What is being built?
    2. What problem does it solve?
    3. Who benefits and why does it matter?
  Avoid implementation details — this should be readable by any stakeholder.
-->
## Overview

Spektacular will let teams keep specifications and implementation plans in Notion while still working from a local cache when agents are creating, reviewing, or implementing work. This solves the coordination problem where shared product and planning artifacts live in Notion, but coding agents need local, repeatable files and conflict checks before they edit or implement them. Product, engineering, and agent users benefit because the shared source of truth stays visible in Notion while local work remains fast, recoverable, and safe from accidental overwrites.



<!--
  REQUIREMENTS
  Specific, testable behaviours the feature must deliver.
  Format: bold title on the checkbox line, detail indented below.
  Rules:
    - Use active voice: "Users can...", "The system must..."
    - Each requirement should be independently verifiable
    - Focus on WHAT, not HOW — avoid prescribing implementation
    - Keep each item atomic — one behaviour per line
-->
## Requirements

- [ ] **Users can connect Spektacular to existing Notion workspaces**
      Users can link Spektacular to existing Notion locations for specifications and plans and receive immediate validation that those locations can support the workflow.
- [ ] **Users can initialize missing Notion workspace structure**
      Users can ask Spektacular to create the required shared locations when they do not already exist, then use those locations through the same linked workflow.
- [ ] **The system identifies incompatible Notion setup before work starts**
      The system reports missing, fixable, and blocking setup issues before creating, syncing, planning, or implementing shared artifacts.
- [ ] **Notion setup requires stable Notion identifiers**
      Spektacular-compatible Notion Specs and Plans locations include required auto-increment identifier fields so Notion-backed artifacts have stable external identifiers.
- [ ] **Users can maintain local working copies of shared artifacts**
      Users and agents can pull shared specifications and plans into a local cache, work from that cache, and keep those cached files out of committed source changes.
- [ ] **Users can pull existing Notion artifacts by shared identity**
      Users and agents can fetch an existing Notion-backed Spek using a Notion URL, Notion page ID, or Spektacular external identifier when another human or agent is picking up work.
- [ ] **Users can sync local changes back to Notion deliberately**
      Users and agents can push local artifact changes to Notion only through an explicit sync action or workflow step that confirms the remote artifact is still safe to update.
- [ ] **The system prevents accidental overwrite of another user's Notion edits**
      The system detects when a shared artifact changed after the local cache was last refreshed and blocks a normal push until the conflict is resolved.
- [ ] **The system guides users and agents through merge conflicts**
      When local and Notion versions diverge, Spektacular presents a merge flow with the local copy, remote copy, and cache baseline so the user or agent can resolve the conflict before retrying sync.
- [ ] **Workflow steps sync changed artifact cache**
      After each Notion-backed workflow step changes a specification, plan, context, or research artifact, the system updates the corresponding local cache and sync metadata before returning the next instruction.
- [ ] **Workflow steps surface conflicts immediately**
      When a step cannot safely sync because a shared Notion artifact changed remotely, the system returns a clear resolution need to the agent or user instead of continuing as though the step succeeded.
- [ ] **Implementation workflows can start from Notion-backed plans**
      When a plan is stored in Notion, the implementation workflow can ensure the local cache is available and current enough before it begins work.
- [ ] **Shared artifact relationships stay visible in Notion**
      Specifications and plans created or synced through Spektacular remain linked in Notion so humans can navigate between the related artifacts.
- [ ] **Notion-backed specifications use external identifiers**
      When Notion artifacts are enabled, specification creation uses an externally supplied identifier from the Notion artifact rather than generating a timestamp or counter identifier locally.
- [ ] **Agent skills explain the Notion artifact flow**
      Because most users run Spektacular through agent skills, installed and embedded skills explain how to use Notion MCP, how to sync cache changes, and how to stop for merge resolution when Notion mode is enabled.


<!--
  CONSTRAINTS
  Hard boundaries the solution must operate within. These are non-negotiable.
  Examples:
    - Must integrate with the existing authentication system
    - Cannot introduce breaking changes to the public API
    - Must support the current minimum supported runtime versions
  Leave blank if there are no constraints.
-->
## Constraints

- Notion-backed artifact support must not remove or break the existing local filesystem artifact workflow.
- Local cache files for Notion-backed specs and plans must not be committed to source control by default.
- Linking existing Notion databases must not silently rename, delete, or change incompatible existing properties.
- Notion schema repair may only apply safe additive changes automatically, such as adding missing compatible properties or missing select options.
- Notion schema repair must present the proposed safe additive changes to the user or agent before applying them, with an explicit approval path for non-interactive agent runs.
- Specs and Plans databases must each include a required Notion auto-increment identifier field.
- Notion-backed spec creation must use an externally supplied Notion-derived identifier and must not fall back to local timestamp or counter generation.
- Workflow commands must not overwrite a shared Notion artifact when the remote artifact has changed since the local cache was last refreshed.
- Notion-backed workflow steps that modify artifacts must either complete local cache sync and metadata updates or stop with an explicit resolution requirement.
- When a project is initialized directly for Notion artifacts, Spektacular must not create the local-only `.spektacular/specs` or `.spektacular/plans` artifact folders; it should create only the project metadata, state/config, knowledge content, and ignored Notion cache paths needed for that mode.
- Knowledge-base content and implementation artifacts are outside the first Notion-backed artifact scope.


<!--
  ACCEPTANCE CRITERIA
  The specific, binary conditions that define "done".
  Format: bold title on the checkbox line, verifiable detail indented below.
  Each criterion must be:
    - Independently verifiable (pass/fail, not subjective)
    - Traceable back to a requirement above
    - Testable by someone who didn't write the code
-->
## Acceptance Criteria

- [ ] **Link validates existing Notion locations**
      Given existing compatible Notion specification and plan locations, when the user links them to Spektacular, the command succeeds, writes the linked configuration, and reports that validation passed.
- [ ] **Link refuses incompatible locations**
      Given linked Notion locations with missing required fields or incompatible field types, when the user links them to Spektacular, the command reports each issue and does not leave the project configured as ready unless validation passes.
- [ ] **Doctor reports fixable and blocking setup issues**
      Given linked Notion locations with a mix of missing safe-to-add fields and incompatible fields, when the user runs the validation command, the output separates fixable issues from blocking issues.
- [ ] **Doctor apply adds safe missing setup**
      Given linked Notion locations missing only safe-to-add fields or options, when the user runs the apply command, the shared Notion locations are updated and a subsequent validation command passes.
- [ ] **Doctor apply prompts before repair**
      Given the validation command identifies safe additive repairs, when the user or agent asks to apply them, the command reports the proposed changes and requires explicit approval before changing Notion.
- [ ] **Init creates missing shared locations**
      Given a Notion parent page without Spektacular-compatible locations, when the user runs initialization for Notion artifacts, Spektacular creates the required specification and plan locations, validates them, and records their identifiers in the project configuration.
- [ ] **Notion schema requires auto-increment IDs**
      Given linked Notion locations do not include the required Spec ID or Plan ID auto-increment fields, when validation runs, the command reports the missing identifier fields as setup issues.
- [ ] **Notion mode omits local artifact folders**
      Given a project is initialized directly for Notion artifacts, when initialization completes, local-only spec and plan artifact folders are not created and the ignored Notion cache location is ready for use.
- [ ] **Notion mode requires external spec identifiers**
      Given Notion artifacts are configured, when a new specification is created, the resulting Spektacular spec name is based on the external Notion identifier and not on the timestamp or counter methods.
- [ ] **Spec creation produces a Notion artifact and local cache**
      Given Notion artifacts are configured, when the user starts a new specification, a new shared specification appears in Notion and the local cache contains the matching editable copy.
- [ ] **Plan creation links back to the specification**
      Given a Notion-backed specification, when the user creates a plan for it, a shared plan appears in Notion, it is linked to the specification, and the local cache contains the plan, context, and research working copies.
- [ ] **Pull refreshes local cache**
      Given a shared artifact changed in Notion, when the user pulls it locally, the cached file reflects the latest Notion content and the sync metadata records the remote version that was pulled.
- [ ] **Pull can locate existing work**
      Given a user or agent has a Notion URL, Notion page ID, or Spektacular external identifier for an existing Notion-backed Spek, when they pull it, the command locates the artifact, records the Notion identity in local state or manifest metadata, and writes the cache.
- [ ] **Push blocks stale local changes**
      Given the local cache was based on an older remote version and the Notion artifact has changed since then, when the user tries to push local changes, the command fails with a conflict message and leaves the Notion artifact unchanged.
- [ ] **Merge flow resolves stale pushes**
      Given a push is blocked by a stale remote version, when the user or agent completes the merge flow and explicitly retries sync, the command updates Notion only with the resolved content and records the new remote version.
- [ ] **Step completion syncs changed artifacts**
      Given a Notion-backed workflow step writes or updates a specification, plan, context, or research artifact, when the step completes successfully, the local cache file and sync manifest reflect the same content and remote version returned by Notion.
- [ ] **Step conflict pauses the workflow**
      Given a Notion-backed workflow step detects that a remote artifact changed after the local cache was pulled, when the step attempts to sync, the step returns a conflict or resolution-required response and does not advance to the next workflow instruction.
- [ ] **Implementation startup refreshes Notion-backed plans**
      Given implementation starts from a Notion-backed plan, when the user starts the implementation workflow, Spektacular ensures the plan, context, and research cache files exist and either confirms they are current or reports the action needed before implementation can continue.
- [ ] **Agent skills teach the enabled Notion flow**
      Given Spektacular is initialized for an agent and Notion artifacts are configured, when the agent uses the installed Spek skills, the instructions explain Notion MCP setup, pull, sync, merge resolution, and the external ID requirement.


<!--
  TECHNICAL APPROACH
  High-level technical direction to guide the planning agent. Include:
    - Key architectural decisions already made
    - Preferred patterns or technologies if known
    - Integration points with existing systems
    - Known risks or areas of uncertainty
  Leave blank if you want the planner to propose the approach.
-->
## Technical Approach

Use Notion as the shared source of truth and a local cache as the working copy. Keep `.spektacular/config.yaml` and `.spektacular/state.json` local, but store Notion-backed markdown snapshots under an ignored cache path such as `.spektacular/cache/notion/specs/<spec-name>.md` and `.spektacular/cache/notion/plans/<plan-name>/{plan.md,context.md,research.md}`. This avoids mixing synced cache files with the existing local-only `.spektacular/specs` and `.spektacular/plans` directories.

Do not add direct Notion API calls to the binary in the first version. The binary should own configuration, workflow state, cache metadata, validation rules, and CLI instructions; agents should perform Notion MCP page/database operations when the CLI tells them to. Commands such as `notion init`, `notion link`, and `notion doctor --apply` should emit structured MCP work requests, validate the returned snapshots, and write local config or state only after the requested MCP work is complete. This keeps authentication and workspace access in the MCP layer while still allowing the CLI to enforce deterministic workflow and conflict behavior.

Add Notion artifact configuration under `artifacts`, with `backend: local|notion`, `cache_dir`, and a `notion` block containing the base page URL plus Specs and Plans data source URLs. There is no managed/linked mode distinction: Spektacular always operates against linked Notion databases, whether those databases already existed or were created by `spektacular notion init`.

Introduce Notion setup commands around one shared schema validator. `spektacular notion init --parent <page-url>` guides the agent through creating missing Specs and Plans databases, validates them, applies safe additive setup to databases it just created, and writes config only after validation passes. `spektacular notion link --specs <data-source-url> --plans <data-source-url>` validates existing databases before writing config. `spektacular notion doctor` reports schema status, and `spektacular notion doctor --apply` proposes safe additive fixes, prompts the user or agent for approval, then guides the approved MCP updates.

The required Notion schema should match the example database shape used in the MCP spike. Specs need Name, Status, Approval, Spec ID as a Notion auto-increment identifier, and a relation to Plans. Plans need Name, Status, Approval, Plan ID as a Notion auto-increment identifier, and a relation to Specs. Tags, Author, Reviewers, Linear Issues, and related product fields are useful when present but should not be required for the first version unless needed by the example database relationship model.

Spec content should be stored in the Notion Spec database page body, matching the page-body model available in the example workspace. Plan content should be stored in the Notion Plan database page body, and Context and Research should be child pages under the Plan row. The MCP spike confirmed that creating a Plan with its Spec relation set to the Spec page URL automatically populated the reverse Spec-to-Plan relation.

Notion-backed specs must use the existing external ID path from the spec identifier work. On creation, the Notion Spec page should be created first so its required Spec ID auto-increment value can be used as the canonical external identifier, then `spec new` should receive that identifier through the `id` input and run with external ID behavior. This avoids local timestamp or counter IDs diverging from the shared Notion artifact identity.

Maintain a cache manifest with each artifact's remote URL, Notion page ID, Notion data source ID, Spektacular external ID, last remote edit timestamp, local checksum, and local path. Workflow state may also record the active Notion page IDs and external IDs so another user or agent can pull existing work by URL, page ID, or external ID. Pull reads the Notion page content into the cache and records the remote version. Push fetches the remote version first; if the remote timestamp differs from the manifest value, the push enters a merge flow instead of overwriting Notion.

Every Notion-backed workflow step should update and sync artifacts using the same user-visible cadence as the existing filesystem workflow. Agents should keep using local markdown cache files and the same Spek state-machine flow; when the filesystem workflow would create, update, or finalize an artifact, Notion mode should update the cache and sync the corresponding Notion page before returning the next instruction. If the sync layer detects that the remote artifact changed since the cache baseline, the workflow step should not advance; it should engage the user or agent in a merge flow that presents the baseline, local version, and remote version, then retries sync only after explicit resolution.

Implementation startup should call the same sync layer before the workflow begins. For Notion-backed plans, `implement new` should verify the plan, context, and research cache entries exist and are not stale, pulling or reporting the required pull action before the implementation agent reads local files.

Because most users interact with Spektacular through agent skills, Notion mode must be explicit in the embedded and installed skills. Skills should tell agents when Notion mode is enabled, what MCP calls are expected, how to pull existing work, how to keep cache files in sync, how to use the required external identifiers, and when to stop and ask the user during doctor repair or merge resolution.


<!--
  SUCCESS METRICS
  How you will know the feature is working well after delivery. Be specific:
    - Quantitative: "p99 latency < 200ms", "error rate < 0.1%"
    - Behavioural: "users complete the flow without support intervention"
  Leave blank if not applicable.
-->
## Success Metrics

- A compatible existing Notion workspace can be linked, validated, and used to create a spec and plan without manual database edits.
- A missing Notion workspace structure can be initialized from a parent page and pass validation immediately after setup.
- Compatible Notion Specs and Plans databases include required Spec ID and Plan ID auto-increment identifier fields.
- Normal sync operations never overwrite a remote Notion edit that happened after the local cache was last pulled.
- A stale local cache reliably enters a merge flow rather than offering only a generic failure.
- Each successful Notion-backed workflow step leaves the local cache and manifest aligned with the latest remote version of every artifact changed by that step.
- Conflict or resolution-required responses appear at the step where the unsafe sync is detected, not later during planning or implementation.
- Notion-backed implementation startup either prepares a complete local plan/context/research cache or exits with a clear corrective action.
- The existing local filesystem workflow continues to pass its current CLI and Harbor validation after Notion support is added.
- Agent skills are sufficient for an agent to run the Notion-backed flow without reading source code.
- Setup validation errors are actionable enough that a user can distinguish safe automatic fixes from blocking schema incompatibilities without reading source code.


<!--
  NON-GOALS
  Explicitly state what this spec does NOT cover. This is as important as
  the requirements — it prevents scope creep and sets clear expectations.
  Examples:
    - "Mobile support is out of scope (tracked in #456)"
    - "Internationalisation will be addressed in a follow-up spec"
  Leave blank if there are no explicit exclusions to call out.
-->
## Non-Goals

- Syncing `.spektacular/knowledge` content to or from Notion is out of scope.
- Syncing implementation artifacts, generated code, test output, or build artifacts to Notion is out of scope.
- Adding direct Notion API authentication and HTTP clients inside the binary is out of scope for the first version.
- Automatically merging conflicting local and remote edits is out of scope; the first version should detect and block conflicts.
- Deleting, renaming, or changing incompatible existing Notion database properties automatically is out of scope.
- Replacing the current local-only filesystem workflow is out of scope.
- Building a general-purpose Notion database migration framework is out of scope.
- Supporting arbitrary Notion workspace shapes beyond the required Specs and Plans databases is out of scope.
