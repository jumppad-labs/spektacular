### Extend version check command vs separate migration command (architecture)
- **Decision**: Extend the existing `version check` command to detect and prompt for migration
- **Rationale**: Users already run version check after upgrades, so integrating migration here provides automatic detection without requiring users to remember a separate command. The version check already has structured output format and prompting patterns that can be reused.
- **Rejected**: Separate `spektacular migrate` command would add friction and require users to know about and remember to run it. Automatic migration without prompt would violate user agency by modifying config files without consent.

### Project scanning heuristics (architecture)
- **Decision**: Scan README.md first paragraph for description, check go.mod/package.json for role inference, use defaults when scanning fails
- **Rationale**: These files are commonly present and contain the most reliable metadata. Defaults ensure migration never fails due to missing files.
- **Rejected**: Interactive prompting for each metadata field would slow down migration and contradict the spec's requirement for immediate execution after confirmation.

### Colocated repo only migration (architecture)
- **Decision**: Only create repo.yaml for the current repository containing .spektacular/, not for registered remote repos
- **Rationale**: Remote repos may not be materialized locally, and attempting to modify them could fail. The spec focuses on migrating "the project" which is the current repository.
- **Rejected**: Migrating all registered repos would require cloning remote repos and could fail if they're not accessible, blocking the entire migration.
