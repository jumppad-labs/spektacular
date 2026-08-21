# Cross-Cutting Context

## Key Architectural Decision: Agent-Driven Metadata Inference

**Decision Made:** 2026-08-21

The metadata inference for repo.yaml will be **agent-driven**, not code-based. The flow is:

1. `version check` detects legacy config.yaml (no repo.yaml) → returns JSON with `status: "migration_needed"` and migration prompt
2. User confirms migration in conversation
3. **Agent** reads project files (README.md, go.mod, package.json, Makefile, Dockerfile, etc.) and infers appropriate metadata values
4. Agent calls migration command with inferred metadata: `spektacular migrate --data '{"description":"...", "role":"...", "tags":[], "deployment":"..."}'`

**Rationale:**
- Leverages LLM's natural language understanding for better metadata extraction (e.g., parsing README descriptions, inferring project purpose)
- Keeps Go code simple - migration command just receives metadata and writes files
- Agent can handle edge cases and ambiguity better than heuristic code
- User can see and correct inferred values in conversation before migration executes

**Implementation Impact:**
- Migration command needs `--data` flag to accept metadata JSON
- No `scanProjectMetadata()` function in Go code
- Agent instructions must include: read project files, infer metadata, call migrate with --data
- Version check only detects and prompts, doesn't scan or execute migration

**Previous Approach (Rejected):**
- Code-based scanning with heuristics (README first paragraph, go.mod detection, etc.)
- Automatic fallback to defaults
- All logic in Go functions

## Architecture Step Completed

**Chosen Direction:** Integration into version check with agent-driven inference and atomic migration command

**Key Design Decisions:**
- Early detection guard in `runVersionCheck()` before version file comparison
- New "migration_needed" status in VersionCheckResult (extends existing status enum)
- Agent performs metadata inference by reading project files
- New `spektacular migrate` command accepts `--data` JSON with metadata
- Atomic operations with deferred rollback (backup → write → verify, restore on error)

**Conventions Applied:**
- Passing tests required (applies to all features)
- Error messages must suggest remediation (migration command returns actionable errors)

**Repo Attribution:**
- All work lands in **spektacular** repo (CLI tool)
- No changes to **docs** repo for this feature
