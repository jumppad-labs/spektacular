### Milestone 1: Version Check Detects Old Config Format

When users run `spektacular version check` after upgrading to a version with multi-repo support, the command now detects when their project uses the old single-file config.yaml format (no repo.yaml present). The output includes a structured migration prompt explaining that config.yaml will be backed up and repo.yaml will be created with scanned metadata. Users see this prompt automatically during their normal post-upgrade version check, without needing to remember a separate migration command.

**Validation**: Run version check in a project with old-format config.yaml. The output includes `migration_prompt.detected: true` and a message explaining the split.

### Milestone 2: Migration Scans Project and Writes Split Config

When users confirm the migration prompt, the system scans their project to suggest repository metadata (description from README.md, role from go.mod/package.json, tags from directory structure), backs up config.yaml to config.yaml.old, and writes the new split configuration: project settings remain in config.yaml while repository metadata goes into the new repo.yaml. If scanning fails to determine values, sensible defaults are used. If any step fails, the backup is restored and both new files are removed, ensuring users never end up in a broken state.

**Validation**: Confirm migration in a test project. Verify config.yaml.old exists and matches original, repo.yaml exists with scanned metadata, and config.yaml is updated. Inject a failure mid-migration and verify rollback restores original state.

### Milestone 3: Migration Logic is Comprehensively Tested

The migration feature has full test coverage: unit tests verify detection logic, scanner heuristics, and rollback behavior; integration tests exercise the end-to-end flow from old config to split config with backup. All acceptance criteria from the spec are covered by automated tests, and the test suite passes completely before the feature is considered done.

**Validation**: Run `go test ./...` and confirm all migration-related tests pass, including detection, scanning, execution, and rollback scenarios.