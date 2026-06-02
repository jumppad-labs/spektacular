## Step {{step}}: {{title}}

Write tests for the code you just implemented. This step runs in a **sub-agent** so the test-authoring context doesn't pollute the main implementation context. Do not write tests in the main context.

### Delegate to the follow-test-patterns skill

Launch a sub-agent with the instructions from:

```
{{config.command}} skill follow-test-patterns
```

The sub-agent should:

1. Read the project's test conventions (typically documented in `thoughts/notes/testing.md` or discoverable via the existing plan/spec step tests).
2. Identify the package or packages the new code lives in.
3. Write `*_test.go` files that match the conventions — `stretchr/testify/require` assertions, `t.TempDir()` for fixtures, co-located with the package under test.
4. Cover the phase's acceptance criteria from `{{plan_path}}` — each criterion should have a corresponding passing test assertion.
5. Return a concise summary: which files were written/modified and what each test asserts.

### STOP-on-mismatch

If the sub-agent reports that the test conventions it discovered don't match what the plan assumes, or that required test infrastructure (fixtures, helpers) is missing, STOP. Report to the user and ask for guidance before continuing.

### Advance

Once tests are written and the sub-agent has returned its summary:

```
{{config.command}} implement goto --data '{"step":"{{next_step}}"}'
```

---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
