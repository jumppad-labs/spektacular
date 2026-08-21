**ScanResult** — Returned by the project scanner, carries suggested repository metadata values:

```go
type ScanResult struct {
    Description string
    Role        string
    Tags        []string
    Deployment  string
}
```

Each field holds either a scanned value or a default when scanning fails. The scanner returns this structure to the migration executor, which writes it into the new `repo.yaml`.

**MigrationPrompt** — Added to `VersionCheckResult` to carry migration-specific information in the version check output:

```go
type VersionCheckResult struct {
    Status           string
    InstalledVersion string
    CurrentVersion   string
    Action           string
    MigrationPrompt  *MigrationPrompt `json:"migration_prompt,omitempty"`
}

type MigrationPrompt struct {
    Detected bool   `json:"detected"`
    Message  string `json:"message"`
}
```

When old-format config is detected, `MigrationPrompt.Detected` is true and `Message` explains the split that will occur. Agents consume this structured output to present the prompt to users.

Existing `Config` and `RepoConfig` structures from `internal/config` are reused without modification. The migration executor uses their existing `ToYAMLFile()` methods to write the split configuration.