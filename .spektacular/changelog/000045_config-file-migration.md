---
created_date: "2026-08-21"
status: completed
closed_date: "2026-08-21"
---

# Config File Migration

## What Was Built

Spektacular now automatically detects and migrates legacy single-file config.yaml to the new split format (config.yaml + repo.yaml) when users upgrade to versions with multi-repo project support. The migration is integrated into the `version check` command: when it detects a legacy configuration (config.yaml present, repo.yaml absent), it returns a structured prompt explaining the migration and what will happen. The migration creates a backup (config.yaml.old) and generates repo.yaml with metadata scanned from project files (README.md, go.mod, package.json, Makefile, Dockerfile), using best-effort heuristics with graceful fallback to defaults.

The implementation includes:
- **Migration detection** (`detectMigrationNeeded`) that checks for legacy configs before version comparison
- **Project metadata scanner** (`scanProjectMetadata`) that reads project files to infer description, role, tags, and deployment method
- **Atomic migration executor** (`executeMigration`) with deferred rollback that restores original state on any error
- **Version check integration** that returns "migration_needed" status with explanatory prompt when legacy config detected
- **Comprehensive test coverage** including detection, scanning, execution, rollback, and end-to-end integration tests

## Why It Matters

This feature ensures existing Spektacular users can upgrade smoothly to versions with multi-repo project support without manual configuration changes or errors. The automatic detection and guided migration removes friction from the upgrade path, while the backup and rollback mechanisms provide safety. Users get appropriate metadata values suggested automatically through project scanning, reducing manual configuration work.

## Deviations from Plan

None. All phases were implemented as planned with no deviations from the original design.
