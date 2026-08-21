The migration logic receives comprehensive unit test coverage across three layers: detection (version check identifies old format), scanning (project scanner suggests correct metadata), and execution (migration creates backup, writes split configs, rolls back on failure). Unit tests use temporary directories to exercise the full file-writing flow without side effects, following the existing test patterns in `cmd/version_test.go` and `internal/project/init_test.go`.

Integration tests verify the end-to-end flow: a test project with old-format config.yaml runs version check, receives the migration prompt, confirms migration, and ends up with config.yaml.old backup plus correctly-populated config.yaml and repo.yaml. These tests assert file presence, content correctness, and that the backup matches the original. Rollback behavior is tested by injecting failures at each step (backup succeeds but repo.yaml write fails, both writes succeed but config.yaml update fails) and verifying the filesystem returns to its pre-migration state.

The project scanner receives focused unit tests for each heuristic: README.md parsing extracts the first paragraph, go.mod/package.json presence correctly infers role, and missing files trigger defaults. Scanner tests use fixture files rather than real projects, keeping them fast and deterministic.

### Success Metrics Verification

The spec defines no quantitative success metrics. Migration success is verified through the following behavioural tests:

- **Detection Triggers Prompt** — Unit test: version check with old config.yaml and no repo.yaml returns VersionCheckResult with MigrationPrompt.Detected=true
- **Prompt Explains Changes** — Unit test: MigrationPrompt.Message contains "backed up" and "repo.yaml will be created"
- **Scanned Metadata Appears** — Integration test: after migration, repo.yaml exists and contains non-empty description, role, tags, deployment fields
- **Backup File Created** — Integration test: config.yaml.old exists and byte-for-byte matches original config.yaml
- **Migration Completes on Confirmation** — Integration test: single migration call produces both repo.yaml and config.yaml.old without additional steps
- **Already-Migrated Skip** — Unit test: version check with both config.yaml and repo.yaml present returns no migration prompt
- **Defaults When Scan Fails** — Unit test: scanner with no README/build files returns ScanResult with default values
- **Rollback on Error** — Unit test: injected failure during migration leaves config.yaml unchanged, no repo.yaml or config.yaml.old exist