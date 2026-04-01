# Discover Project Commands

Discover and document all available project commands (make targets, npm scripts, go tasks, etc.) and write them to `thoughts/notes/commands.md`.

## Instructions

1. Search the project root for build/task files:
   - `Makefile` / `GNUmakefile` — extract targets with descriptions
   - `package.json` — extract `scripts` section
   - `Taskfile.yml` / `Taskfile.yaml` — extract tasks
   - `justfile` — extract recipes
   - `Cargo.toml` — look for workspace commands
   - Any other build system files

2. For each command found, document:
   - **Command**: The exact command to run (e.g., `make test`, `npm run build`)
   - **Description**: What it does
   - **Category**: build, test, lint, deploy, dev, etc.

3. Create `thoughts/notes/commands.md` with the following structure:

```markdown
# Project Commands

## Build
- `<command>` — <description>

## Test
- `<command>` — <description>

## Lint
- `<command>` — <description>

## Dev
- `<command>` — <description>

## Other
- `<command>` — <description>
```

4. If `thoughts/notes/` directory doesn't exist, create it.
