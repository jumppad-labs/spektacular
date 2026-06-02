## Step {{step}}: {{title}}

Run the verification commands for the current phase and report the result. This step runs in a **sub-agent** so the full command output stays out of the main context.

### Delegate to the verify-implementation skill

Launch a sub-agent with the instructions from:

```
{{config.command}} skill verify-implementation
```

The sub-agent should:

1. Read the current phase's acceptance criteria from `{{plan_path}}`.
2. Map each criterion to a concrete verification command (typically `make test`, `make lint`, or a phase-specific command listed in `{{context_path}}` or `thoughts/notes/commands.md`).
3. Run each command, capture exit codes and a short excerpt of any failures.
4. Return a **concise pass/fail summary** — one line per command, no full test output. If everything passes, a single "all green" line is enough.

### STOP-on-mismatch

If any verification command fails, STOP. Report the failures to the user as the sub-agent returned them. Do not advance to the next step — the user must decide whether to:

1. **Fix the code and re-run verification.** Return to the `implement` step with the fixes in mind.
2. **Accept the partial failure and proceed.** Only if the user explicitly chooses this.
3. **Abandon this phase.** Skip to the next unchecked phase.

### Advance

Once verification is green (or the user has explicitly authorized proceeding past a failure):

```
{{config.command}} implement goto --data '{"step":"{{next_step}}"}'
```

---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
