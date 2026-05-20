# Create a Feature Specification using Spektacular

You are testing the `spektacular` CLI tool by creating a complete feature specification.
The binary is already installed at `/usr/local/bin/spektacular`.

## Setup

First initialize the project:

```bash
spektacular init {{agent}}
```

## Task

Create a specification for a **user authentication feature using JWT tokens** by
using the `spek-new` skill that was installed during init.

Run the skill:

```
{{spek_new}}
```

The skill will guide you through the full spec workflow. Follow each instruction
it gives you.

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
- The created spec file under `.spektacular/specs/` contains content
- Each spec section has meaningful, non-placeholder text
- The spec file is committed through `spektacular spec file write` — never written or edited directly with built-in file tools
