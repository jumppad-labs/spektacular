---
description: Create a new Specification
argument-hint: <spec-name>
---

Create a new spec using the go run . CLI. The user must provide a spec name as the argument.

Spec name: $ARGUMENTS

If no spec name was provided, ask the user for one before proceeding.

Start the spec workflow by running:

```
go run . spec --new --data '{"name": "<spec_name>"}'
```

The command creates the spec file and state file automatically, then returns JSON with an `instruction` field. Follow that instruction exactly.

Each instruction tells you to call `go run . spec --next` (with `--data` if needed) when the step is complete.