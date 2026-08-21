---
created_date: "2026-08-21"
status: completed
closed_date: "2026-08-21"
---

# Context: 000045_config-file-migration

## Current State Analysis

The current Spektacular codebase has a version check command (`cmd/version.go`) that compares the installed version file against the running binary version. The command returns structured JSON with status values "match", "mismatch", or "missing" and an action field for stale installation messages.

The config system (`internal/config/`) supports both project-level config.yaml (project identity, agent settings, spec/plan/changelog storage) and repo-level repo.yaml (repo metadata: description, role, tags, deployment). The split was introduced in spec 000039_project-level-capabilities to support multi-repo projects.

Legacy installations have only config.yaml without repo.yaml. These predate the multi-repo support and need migration to the new format.

## Per-Phase Technical Notes

### Phase 1.1: Add migration detection function

**File changes:**
- spektacular:cmd/version.go:~130 — Add `detectMigrationNeeded()` function that checks for config.yaml and repo.yaml existence in .spektacular directory
- spektacular:cmd/version_test.go:~150 — Add unit tests for detection logic covering legacy config, already-migrated, and missing directory cases

**Complexity:** Low

**Token estimate:** ~2k tokens

**Agent strategy:** Single agent, sequential execution. Simple file existence checks with straightforward test cases.

### Phase 1.2: Implement project metadata scanner

**File changes:**
- spektacular:cmd/version.go:~160 — Add `scanProjectMetadata(projectRoot string) (config.RepoConfig, error)` function
- spektacular:cmd/version.go:~165 — Implement README.md parsing for description (first paragraph or H1)
- spektacular:cmd/version.go:~180 — Implement go.mod detection and role inference
- spektacular:cmd/version.go:~190 — Implement package.json detection and role inference
- spektacular:cmd/version.go:~200 — Implement Makefile/Dockerfile detection for deployment method
- spektacular:cmd/version.go:~210 — Implement fallback to defaults when all heuristics fail
- spektacular:cmd/version_test.go:~200 — Add unit tests for each heuristic and fallback behavior

**Complexity:** Medium

**Token estimate:** ~5k tokens

**Agent strategy:** Single agent, sequential execution. Each heuristic is independent but builds on the previous, so sequential implementation is clearest.

### Phase 2.1: Implement migration executor with atomic operations

**File changes:**
- spektacular:cmd/version.go:~250 — Add `executeMigration(projectRoot string, repoCfg config.RepoConfig) error` function
- spektacular:cmd/version.go:~255 — Implement backup creation (config.yaml → config.yaml.old)
- spektacular:cmd/version.go:~265 — Implement repo.yaml write using repoCfg.ToYAMLFile
- spektacular:cmd/version.go:~275 — Implement verification (both files exist)
- spektacular:cmd/version.go:~285 — Implement deferred rollback function (restore backup, remove repo.yaml on error)
- spektacular:cmd/version_test.go:~300 — Add unit tests for successful migration
- spektacular:cmd/version_test.go:~350 — Add unit tests for rollback at each error point (backup fails, write fails, verify fails)

**Complexity:** Medium

**Token estimate:** ~6k tokens

**Agent strategy:** Single agent, sequential execution. Atomic operations and error handling require careful sequencing, best done by one agent.

### Phase 3.1: Extend VersionCheckResult for migration status

**File changes:**
- spektacular:cmd/version.go:13-22 — Extend VersionCheckResult struct documentation to mention "migration_needed" status
- spektacular:cmd/version.go:~50 — Update schema output to include "migration_needed" in status enum
- spektacular:cmd/version_test.go:~400 — Add unit test verifying new status can be returned and serialized

**Complexity:** Low

**Token estimate:** ~1k tokens

**Agent strategy:** Single agent, sequential execution. Simple struct extension with schema update.

### Phase 3.2: Integrate migration detection into version check flow

**File changes:**
- spektacular:cmd/version.go:33-77 — Modify runVersionCheck to call detectMigrationNeeded() before version file comparison
- spektacular:cmd/version.go:~40 — Add early return when migration needed, composing prompt text and returning VersionCheckResult with status "migration_needed"
- spektacular:cmd/version.go:~120 — Add helper function to compose migration prompt text
- spektacular:cmd/version_test.go:~450 — Add integration tests for version check with migration detection
- spektacular:cmd/version_test.go:~500 — Add integration tests for version check with already-migrated project

**Complexity:** Medium

**Token estimate:** ~4k tokens

**Agent strategy:** Single agent, sequential execution. Integration requires understanding the existing version check flow, best done by one agent.

### Phase 4.1: Add comprehensive unit and integration tests

**File changes:**
- spektacular:cmd/version_test.go:~550 — Add tests mapping each spec acceptance criterion to a behavioral test
- spektacular:cmd/version_test.go:~600 — Add test: detection triggers prompt
- spektacular:cmd/version_test.go:~620 — Add test: prompt explains changes
- spektacular:cmd/version_test.go:~640 — Add test: scanned metadata appears
- spektacular:cmd/version_test.go:~660 — Add test: backup file created
- spektacular:cmd/version_test.go:~680 — Add test: migration completes on confirmation
- spektacular:cmd/version_test.go:~700 — Add test: already-migrated skip
- spektacular:cmd/version_test.go:~720 — Add test: defaults when scan fails
- spektacular:cmd/version_test.go:~740 — Add test: rollback on error

**Complexity:** Medium

**Token estimate:** ~8k tokens

**Agent strategy:** 2-3 parallel agents for independent test cases. Each acceptance criterion test can be written independently, then integrated into the test file.

## Testing Strategy

Unit tests cover each component in isolation: detector (file existence checks), scanner (each heuristic plus fallback), executor (atomic sequence and rollback). Integration tests verify the full version check flow with migration detection. Error handling tests simulate filesystem failures to exercise rollback paths.

Tests follow existing project patterns: table-driven tests for multiple scenarios, t.TempDir() for isolated filesystem operations, testify/require for assertions. Test files colocated with implementation (cmd/version_test.go).

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|-------------|----------------|
| Low | ~10k | Single agent, sequential |
| Medium | ~25k | 2-3 parallel agents |
| High | ~50k+ | Parallel analysis, sequential integration |

Total estimated tokens across all phases: ~26k (Medium tier). Phases 1.1, 3.1 are Low complexity. Phases 1.2, 2.1, 3.2, 4.1 are Medium complexity. No High complexity phases.

## Migration Notes

This feature itself is a migration tool. No migration steps needed for the feature itself.

## Performance Considerations

Migration detection adds minimal overhead to version check: two file existence checks (config.yaml, repo.yaml). Project scanning reads at most 5 files (README.md, go.mod, package.json, Makefile, Dockerfile), all small text files. The migration itself (backup, write, verify) is a one-time operation per project.

## Project References

- Spec 000039_project-level-capabilities: Introduced the config.yaml/repo.yaml split
- cmd/version.go: Version check command implementation
- internal/config/: Config and RepoConfig types with YAML serialization
- internal/project/init.go: Project initialization patterns for config creation
