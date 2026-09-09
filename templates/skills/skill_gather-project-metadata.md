# Gather Project Metadata

Collect project metadata for inclusion in plan documents.

## Instructions

Gather the following metadata:

1. **Timestamp**: Current date/time in ISO 8601 format (e.g., `2024-01-15T14:30:00Z`)
2. **Git commit**: Current HEAD commit hash (short form)
3. **Git branch**: Current branch name
4. **Repository**: Repository name or remote URL

## Commands

Run these in the code of the repo the plan targets (the `repo list` command reports where it lives as `root`), not in whatever directory you happen to be in.

```bash
# ISO timestamp
date -u +"%Y-%m-%dT%H:%M:%SZ"

# Git commit (short hash)
git rev-parse --short HEAD

# Git branch
git branch --show-current

# Repository
git remote get-url origin 2>/dev/null || basename "$(pwd)"
```

## Output Format

Include this metadata in the plan document's header or frontmatter:

```
Created: <ISO timestamp>
Commit: <short hash>
Branch: <branch name>
Repository: <repo name>
```
