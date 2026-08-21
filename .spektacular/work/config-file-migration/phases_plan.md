## Milestone 1: Version Check Detects Old Config Format

#### - [ ] Phase 1.1: Add migration detection to version check

**Repo:** spektacular

The version check command gains logic to detect old-format configurations by checking whether `config.yaml` exists without a corresponding `repo.yaml` in the `.spektacular/` directory. When detected, the command returns a new `MigrationPrompt` field in its existing `VersionCheckResult` output, containing a boolean flag and an explanatory message. This detection runs before the version check returns its result, adding the migration branch to the existing flow without changing the command's core behavior for already-migrated projects.

*Technical detail:* [context.md#phase-11](./context.md#phase-11-add-migration-detection-to-version-check)

**Acceptance criteria**:
- [ ] Version check with old config.yaml and no repo.yaml returns output containing `migration_prompt.detected: true`
- [ ] Version check with both config.yaml and repo.yaml present returns output with no migration prompt
- [ ] Migration prompt message explains that config.yaml will be backed up and repo.yaml will be created

#### - [ ] Phase 1.2: Add detection tests

**Repo:** spektacular

Unit tests verify the detection logic in isolation: a test with old-format config returns a migration prompt, a test with both files present returns no prompt, and the prompt message contains the expected explanation. Tests use temporary directories and do not require a real Spektacular project, keeping them fast and deterministic.

*Technical detail:* [context.md#phase-12](./context.md#phase-12-add-detection-tests)

**Acceptance criteria**:
- [ ] Test suite includes unit tests for old-format detection returning migration prompt
- [ ] Test suite includes unit tests for already-migrated projects returning no prompt
- [ ] All detection tests pass

## Milestone 2: Migration Scans Project and Writes Split Config

#### - [ ] Phase 2.1: Implement project scanner

**Repo:** spektacular

A new project scanner examines the repository to suggest metadata values for the new repo.yaml. It reads README.md and extracts the first paragraph as the description, checks for go.mod or package.json to infer role (tool vs application), inspects directory structure for tags, and determines deployment method. When files are missing or unparseable, the scanner returns default values (description: "Spektacular project", role: "application", tags: empty, deployment: "manual"). The scanner is a pure function that returns a result struct and never writes files.

*Technical detail:* [context.md#phase-21](./context.md#phase-21-implement-project-scanner)

**Acceptance criteria**:
- [ ] Scanner extracts description from README.md first paragraph when present
- [ ] Scanner infers role from go.mod/package.json presence
- [ ] Scanner returns default values when project files are missing
- [ ] Scanner never writes files or modifies project state

#### - [ ] Phase 2.2: Implement migration executor

**Repo:** spektacular

A new migration executor performs the atomic migration operation: it renames config.yaml to config.yaml.old as a backup, writes the new repo.yaml with scanned metadata using `RepoConfig.ToYAMLFile()`, and updates config.yaml using `Config.ToYAMLFile()`. If any step fails after the backup is created, the executor removes both new files and restores the backup, ensuring users never end up in a partially-migrated state. The executor is called from the version check command when the user confirms the migration prompt.

*Technical detail:* [context.md#phase-22](./context.md#phase-22-implement-migration-executor)

**Acceptance criteria**:
- [ ] Migration creates config.yaml.old backup that matches original config.yaml
- [ ] Migration writes repo.yaml with scanned metadata
- [ ] Migration updates config.yaml to remove repo-specific fields
- [ ] Migration rolls back on failure by removing new files and restoring backup

#### - [ ] Phase 2.3: Wire migration into version check

**Repo:** spektacular

The version check command is extended to call the migration executor when detection occurs and the user confirms. The command passes the scanned metadata to the executor and returns the migration result in its output. This completes the end-to-end flow from detection through execution.

*Technical detail:* [context.md#phase-23](./context.md#phase-23-wire-migration-into-version-check)

**Acceptance criteria**:
- [ ] Version check calls migration executor when old format is detected and user confirms
- [ ] Version check output includes migration result (success or failure)
- [ ] End-to-end flow from detection to split config works without manual intervention

#### - [ ] Phase 2.4: Add scanner and executor tests

**Repo:** spektacular

Unit tests verify scanner heuristics (README parsing, role inference, defaults) and executor behavior (backup creation, file writing, rollback on failure). Integration tests exercise the end-to-end flow: a test project with old config runs version check, confirms migration, and ends up with the correct split configuration and backup.

*Technical detail:* [context.md#phase-24](./context.md#phase-24-add-scanner-and-executor-tests)

**Acceptance criteria**:
- [ ] Scanner tests verify README parsing, role inference, and default fallbacks
- [ ] Executor tests verify backup creation, split config writing, and rollback
- [ ] Integration tests verify end-to-end migration flow produces correct files
- [ ] All scanner and executor tests pass

## Milestone 3: Migration Logic is Comprehensively Tested

#### - [ ] Phase 3.1: Add acceptance criteria tests

**Repo:** spektacular

Additional tests verify every acceptance criterion from the spec: detection triggers prompt, prompt explains changes, scanned metadata appears in repo.yaml, backup file is created, migration completes on confirmation, already-migrated projects are skipped, defaults are used when scanning fails, and rollback works on error. These tests ensure the feature meets all requirements before it's considered complete.

*Technical detail:* [context.md#phase-31](./context.md#phase-31-add-acceptance-criteria-tests)

**Acceptance criteria**:
- [ ] Test suite includes tests for all eight acceptance criteria from the spec
- [ ] All acceptance criteria tests pass
- [ ] Running `go test ./...` shows no failures in migration-related tests