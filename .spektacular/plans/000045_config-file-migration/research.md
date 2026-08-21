---
created_date: "2026-08-21"
status: completed
closed_date: "2026-08-21"
---

# Research: 000045_config-file-migration

## Alternatives considered and rejected

### Option 1: Separate migration command
Create a standalone `spektacular migrate` command that users run manually.
**Rejected:** Violates the constraint "Must integrate with the existing `version check` command without requiring a separate migration command". Users would need to discover and remember to run it.

### Option 2: Automatic silent migration
Detect and migrate automatically without prompting the user.
**Rejected:** Too aggressive for a structural change. Users should be aware their configuration is being modified, even with a backup. Silent changes to project files violate user expectations.

### Option 3: Preview-then-confirm flow
Show the user what will change, wait for confirmation, then execute.
**Rejected:** The spec explicitly excludes this in Non-Goals: "Providing a preview of changes before migration - migration executes immediately after user confirmation". The prompt itself explains what will happen.

## Chosen approach — evidence

**Integration point:** cmd/version.go:runVersionCheck (lines 33-77)
- Already detects config state via classifyVersion function
- Returns structured JSON with status and action fields
- Natural place to add migration detection and prompt

**Config loading:** internal/config/config.go:FromYAMLFile (lines 158-168)
- Loads and validates config.yaml
- Can be used to detect legacy format (no repos, single-file structure)

**Repo config:** internal/config/repo.go:RepoConfigFromYAMLFile (lines 48-63)
- Loads repo.yaml when it exists
- NewDefaultRepoConfig provides fallback defaults

**Project scanning for metadata:**
- README.md: First paragraph or title for description
- go.mod: Indicates Go project, role="application" or "tool"
- package.json: Indicates Node.js project, role="application"
- Makefile/Dockerfile: Suggests deployment method
- Directory structure: Can infer tags (e.g., "cli", "api", "web")

**File operations:**
- os.ReadFile, os.WriteFile for backup creation
- config.Config.ToYAMLFile and config.RepoConfig.ToYAMLFile for writing new files

## Files examined

spektacular:cmd/version.go:13-77 — Version check command structure, status classification, action message generation
spektacular:cmd/version.go:88-103 — classifyVersion function for detecting config state
spektacular:cmd/version.go:106-119 — staleAction message composition with agent-aware init command
spektacular:internal/config/config.go:158-168 — Config loading and validation entry point
spektacular:internal/config/config.go:170-183 — ParseYAMLFile for lenient loading (used by init)
spektacular:internal/config/config.go:1-157 — Config structure definitions and defaults
spektacular:internal/config/repo.go:1-93 — RepoConfig structure and loading functions
spektacular:internal/config/repo.go:28-42 — NewDefaultRepoConfig with default knowledge and changelog
spektacular:internal/project/init.go:1-189 — Project initialization logic, config creation patterns
spektacular:cmd/init.go:1-60 — Init command that writes config.yaml and version file
spektacular:cmd/repo.go:1-250 — Repo add command showing metadata handling (description, role, tags, deployment)
spektacular:internal/repo/footprint.go:1-93 — EnsureFootprint showing repo.yaml creation and repair patterns

## External references

None — this is an internal refactoring of existing configuration structure.

## Prior plans / specs consulted

spektacular:.spektacular/specs/000039_project-level-capabilities.md — Introduced the config.yaml/repo.yaml split and multi-repo project support. Provides context for why migration is needed: legacy single-file configs predate the split.

## Open assumptions

1. **Metadata scanning heuristics:** Assume README.md first paragraph is suitable for description, go.mod presence indicates Go project. If these heuristics fail, defaults are used.
2. **Backup file naming:** Assume config.yaml.old is an acceptable backup name and won't conflict with user files.
3. **Migration timing:** Assume version check is run frequently enough that users will encounter the prompt soon after upgrading.
4. **Atomic operations:** Assume filesystem operations (backup, write repo.yaml) can be made atomic enough that partial failure is detectable and rollback-able.
5. **Single repo assumption:** Assume legacy configs only have the colocated repo (no registered repos list), so only one repo.yaml needs creation.

## Drafting assumptions

### Integration into version check command (architecture)
- **Decision**: Integrate migration detection and prompt directly into `runVersionCheck` function, executing before version file comparison
- **Rationale**: Version check is already the natural entry point for detecting config state changes. Users run it frequently (via agent skills), ensuring timely discovery of migration needs. Placing migration before version comparison prevents confusing "version mismatch" errors when the migration itself would update tracked files.
- **Rejected**: Separate `spektacular migrate` command would require users to discover and remember to run it, violating the constraint to integrate with existing version check. Automatic silent migration was rejected as too aggressive for structural config changes.

### Project metadata scanning heuristics (architecture)
- **Decision**: Use best-effort heuristics (README.md first paragraph, go.mod/package.json detection, Makefile/Dockerfile presence) with graceful fallback to defaults
- **Rationale**: Most projects follow common patterns (README describes the project, go.mod indicates Go). Heuristics provide better-than-default values in typical cases while never failing — defaults ensure migration always succeeds even when heuristics can't determine appropriate values.
- **Rejected**: Interactive metadata editing during migration (excluded in spec Non-Goals). Requiring all metadata fields to be determinable (would block migration for atypical projects).

### Atomic migration with rollback (architecture)
- **Decision**: Execute migration as atomic sequence (backup → write repo.yaml → verify), with deferred rollback function that restores on any error
- **Rationale**: Ensures project is never left in inconsistent state. Backup-first means original config is always recoverable. Deferred rollback follows Go's standard error-handling pattern and guarantees cleanup even on panic.
- **Rejected**: Multi-step confirmation flow (excluded in spec Non-Goals). Partial migration with manual recovery (violates rollback requirement).

### Convention selection (architecture)
- **Decision**: Apply "Passing tests are required before calling work done" convention
- **Rationale**: Migration logic involves file operations, error handling, and state transitions — all areas where comprehensive testing is critical. The convention directly applies since this feature adds new code that must be tested before completion.
- **Rejected**: No other project conventions apply to this feature. The migration is internal infrastructure work, not user-facing functionality that would trigger UI/UX conventions.

## Rehydration cues

```bash
# Re-examine version check integration point
go run . version check --schema

# Review config loading patterns
rg "FromYAMLFile|ParseYAMLFile" internal/config/

# Check existing error handling patterns
rg "output.NewError" cmd/version.go

# Review init's config creation
rg "ToYAMLFile" cmd/init.go internal/project/init.go

# Examine repo metadata handling
rg "Description|Role|Tags|Deployment" cmd/repo.go internal/config/repo.go
```
