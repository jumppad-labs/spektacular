# Research: Config File Migration

## Alternatives considered and rejected

### Option 1: Separate migration command
**Rejected**: Adds friction - users must remember to run an extra command after upgrade. The spec explicitly requires integration with version check for automatic detection.

### Option 2: Automatic migration without prompt
**Rejected**: Violates user agency - modifying config files without explicit consent could surprise users mid-workflow. The spec requires a prompt with user confirmation.

### Option 3: Preview-then-confirm flow
**Rejected**: Adds unnecessary complexity. The spec explicitly states "Non-Goals: Providing a preview of changes before migration - migration executes immediately after user confirmation."

## Chosen approach — evidence

**Integrate migration into version check command** (`cmd/version.go:runVersionCheck`)
- Version check already detects version mismatches and prompts for action (lines 67-75)
- Returns structured output with `status`, `action` fields that agents consume (lines 46-51)
- Already has access to `.spektacular` directory via `dataDir()` helper (line 56)

**Detect old format by checking for missing repo.yaml** (`internal/config/repo.go`)
- `RepoConfigFileName = "repo.yaml"` constant defines the filename (line 12)
- `RepoConfigFromYAMLFile()` loads repo config, returns error if missing (lines 47-62)
- Old projects will have `config.yaml` but no `repo.yaml` in `.spektacular/`

**Scan project files for metadata suggestions**
- No existing scanning logic found - needs implementation
- Should check: README.md (description), go.mod/package.json (language/role), directory structure
- Reference: `internal/project/init.go` shows how to walk project structure (lines 85-110)

**Preserve original config as backup**
- Use standard Go `os.Rename()` to move `config.yaml` to `config.yaml.old`
- Reference: `cmd/version.go:writeVersionFile()` shows file writing pattern (lines 127-129)

**Write new split configs**
- `Config.ToYAMLFile()` writes project config (internal/config/config.go:332-340)
- `RepoConfig.ToYAMLFile()` writes repo config (internal/config/repo.go:77-85)
- Reference: `cmd/init.go:runInit()` shows the write sequence (lines 36-40)

## Files examined

- `cmd/version.go:1-129` — Version check command structure, output format, file path helpers
- `cmd/version.go:46-75` — VersionCheckResult struct and runVersionCheck implementation
- `internal/config/config.go:1-340` — Config struct, validation, YAML serialization
- `internal/config/repo.go:1-85` — RepoConfig struct, separate from project config
- `cmd/init.go:1-40` — How new projects write both config files
- `internal/project/init.go:1-195` — Project initialization, directory walking patterns
- `cmd/root.go:200-240` — loadConfig() and dataDir() helpers used by all commands
- `cmd/repo.go:1-350` — How repo metadata is read/written to repo.yaml

## External references

None - all implementation patterns exist in codebase.

## Prior plans / specs consulted

- `.spektacular/specs/000039_project-level-capabilities.md` — Introduced the config/repo.yaml split (completed 2026-07-30)
- `.spektacular/specs/000042_repo-self-describing-metadata.md` — Moved descriptive metadata to repo.yaml (completed 2026-08-13)

## Open assumptions

1. **Scanning heuristics**: Assume README.md first paragraph is description, go.mod/package.json presence determines role (tool vs application). If scanning fails, use defaults from spec.
2. **Colocated repo only**: Migration only creates repo.yaml for the current repository (the one containing .spektacular/). Registered remote repos are not migrated.
3. **Single backup**: Only one config.yaml.old is kept. If migration runs twice, the second run overwrites the first backup.
4. **No rollback on partial success**: If repo.yaml write succeeds but config.yaml.old creation fails, the migration is considered failed and both files are removed.

## Rehydration cues

```bash
# Load version check implementation
go run . --help | grep version
cat cmd/version.go

# Check config structures
cat internal/config/config.go | grep "type Config struct" -A 20
cat internal/config/repo.go | grep "type RepoConfig struct" -A 10

# Review init process for reference
cat cmd/init.go
cat internal/project/init.go | grep -A 30 "func Init"
```
