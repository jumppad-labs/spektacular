### Phase 1.1: Add migration detection to version check

**File changes:**
- `cmd/version.go:46-51` — Extend `VersionCheckResult` struct to add optional `MigrationPrompt *MigrationPrompt` field
- `cmd/version.go:67-75` — Add detection logic in `runVersionCheck()` before returning result: check if `config.yaml` exists and `repo.yaml` does not using `os.Stat()` on both paths
- `cmd/version.go:new` — Add `MigrationPrompt` struct with `Detected bool` and `Message string` fields
- `cmd/version.go:new` — Add `detectOldConfigFormat(dataDir string) (*MigrationPrompt, error)` helper function that checks file presence and returns prompt when detected

**Complexity:** Low

**Token estimate:** ~2k tokens

**Agent strategy:** Single agent, sequential execution. Detection is a pure function with no external dependencies.

### Phase 1.2: Add detection tests

**File changes:**
- `cmd/version_test.go:new` — Add `TestVersionCheck_DetectsOldFormat()` that creates temp dir with config.yaml only, runs version check, asserts `MigrationPrompt.Detected == true`
- `cmd/version_test.go:new` — Add `TestVersionCheck_SkipsAlreadyMigrated()` that creates temp dir with both config.yaml and repo.yaml, runs version check, asserts no migration prompt
- `cmd/version_test.go:new` — Add `TestVersionCheck_PromptMessage()` that verifies prompt message contains "backed up" and "repo.yaml"

**Complexity:** Low

**Token estimate:** ~1k tokens

**Agent strategy:** Single agent, sequential execution. Standard Go table-driven tests following existing patterns in version_test.go.

### Phase 2.1: Implement project scanner

**File changes:**
- `internal/project/scanner.go:new` — Create new file with `ScanResult` struct (Description, Role, Tags, Deployment fields)
- `internal/project/scanner.go:new` — Add `ScanProject(projectRoot string) ScanResult` function that walks project, reads README.md, checks for go.mod/package.json, returns result
- `internal/project/scanner.go:new` — Add `extractDescription(readmePath string) string` helper that reads README.md and extracts first paragraph
- `internal/project/scanner.go:new` — Add `inferRole(projectRoot string) string` helper that checks for go.mod (returns "tool") or package.json (returns "application"), defaults to "application"
- `internal/project/scanner.go:new` — Add `defaultScanResult() ScanResult` helper that returns defaults: description "Spektacular project", role "application", tags empty, deployment "manual"

**Complexity:** Medium

**Token estimate:** ~3k tokens

**Agent strategy:** Single agent, sequential execution. Scanner is self-contained with no dependencies on other new components.

### Phase 2.2: Implement migration executor

**File changes:**
- `internal/project/migrate.go:new` — Create new file with `Migrate(projectRoot string, metadata ScanResult) error` function
- `internal/project/migrate.go:new` — Implement atomic migration: rename config.yaml to config.yaml.old, write repo.yaml with metadata using `config.RepoConfig.ToYAMLFile()`, update config.yaml using `config.Config.ToYAMLFile()`
- `internal/project/migrate.go:new` — Add rollback logic: defer function that checks for error, removes repo.yaml and config.yaml if they exist, restores config.yaml from backup
- `internal/project/migrate.go:new` — Add `backupConfig(configPath string) error` helper that renames config.yaml to config.yaml.old
- `internal/project/migrate.go:new` — Add `writeRepoConfig(projectRoot string, metadata ScanResult) error` helper that creates RepoConfig from ScanResult and writes repo.yaml

**Complexity:** Medium

**Token estimate:** ~4k tokens

**Agent strategy:** Single agent, sequential execution. Migration logic is linear with clear error handling and rollback.

### Phase 2.3: Wire migration into version check

**File changes:**
- `cmd/version.go:67-75` — Extend `runVersionCheck()` to call scanner and executor when migration prompt is detected and user confirms
- `cmd/version.go:new` — Add migration confirmation logic (check for confirmation flag or interactive prompt)
- `cmd/version.go:new` — Call `project.ScanProject()` to get metadata, then `project.Migrate()` to execute migration
- `cmd/version.go:new` — Update `VersionCheckResult` to include migration status (success/failure) in output

**Complexity:** Low

**Token estimate:** ~2k tokens

**Agent strategy:** Single agent, sequential execution. Wiring is straightforward integration of existing components.

### Phase 2.4: Add scanner and executor tests

**File changes:**
- `internal/project/scanner_test.go:new` — Add `TestScanProject_ExtractsDescription()` with fixture README.md
- `internal/project/scanner_test.go:new` — Add `TestScanProject_InfersRole()` with fixture go.mod and package.json
- `internal/project/scanner_test.go:new` — Add `TestScanProject_ReturnsDefaults()` with empty project directory
- `internal/project/migrate_test.go:new` — Add `TestMigrate_CreatesBackup()` that verifies config.yaml.old matches original
- `internal/project/migrate_test.go:new` — Add `TestMigrate_WritesSplitConfig()` that verifies repo.yaml and updated config.yaml
- `internal/project/migrate_test.go:new` — Add `TestMigrate_RollsBackOnFailure()` that injects failure and verifies rollback
- `cmd/version_test.go:new` — Add `TestVersionCheck_EndToEndMigration()` integration test with full flow

**Complexity:** Medium

**Token estimate:** ~5k tokens

**Agent strategy:** 2 parallel agents: one for scanner tests, one for executor tests. Integration test is sequential after both complete.

### Phase 3.1: Add acceptance criteria tests

**File changes:**
- `cmd/version_test.go:new` — Add `TestAcceptance_DetectionTriggersPrompt()` verifying old config triggers prompt
- `cmd/version_test.go:new` — Add `TestAcceptance_PromptExplainsChanges()` verifying prompt message content
- `cmd/version_test.go:new` — Add `TestAcceptance_ScannedMetadataAppears()` verifying repo.yaml contains scanned values
- `cmd/version_test.go:new` — Add `TestAcceptance_BackupFileCreated()` verifying config.yaml.old exists and matches
- `cmd/version_test.go:new` — Add `TestAcceptance_MigrationCompletesOnConfirmation()` verifying single-step migration
- `cmd/version_test.go:new` — Add `TestAcceptance_AlreadyMigratedSkip()` verifying no prompt with both files
- `cmd/version_test.go:new` — Add `TestAcceptance_DefaultsWhenScanFails()` verifying defaults used
- `cmd/version_test.go:new` — Add `TestAcceptance_RollbackOnError()` verifying rollback behavior

**Complexity:** Low

**Token estimate:** ~4k tokens

**Agent strategy:** Single agent, sequential execution. Acceptance tests follow standard patterns and build on existing test infrastructure.