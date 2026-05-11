# Create a Feature Specification using Spektacular

You are testing the `spektacular` CLI tool by creating a complete feature specification.
The binary is already installed at `/usr/local/bin/spektacular`.

## Setup

First initialize the project:

```bash
spektacular init codex
```

## Task

Create a specification for a **user authentication feature using JWT tokens** by
driving the native `spektacular spec` workflow.

Start the workflow:

```bash
spektacular spec new --data '{"name":"user-auth"}'
```

The CLI returns JSON with a normalized, prefixed `spec_name` and `spec_path`.
Use those returned values as the source of truth. Follow each instruction the
CLI gives you, and advance with `spektacular spec goto --data '{"step":"..."}'`.

When writing content for each section, use these details about the feature:
- **What**: Stateless user authentication using JWT access and refresh tokens
- **Problem**: The current session-based auth doesn't scale across multiple services
- **Users**: Backend developers consuming the auth API, and end users who log in

Write meaningful, non-placeholder content for every section.

## After completion

Copy the `.spektacular` directory to `/logs/artifacts/` so results are collected:

```bash
cp -r /app/.spektacular /logs/artifacts/spektacular
```

### Success criteria

- The workflow reaches the `finished` or `done` state
- All steps appear in the completed_steps list
- The spec file at the returned `spec_path` contains content
- Each spec section has meaningful, non-placeholder text
