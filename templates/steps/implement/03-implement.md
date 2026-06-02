## Step {{step}}: {{title}}

Write the code for the current phase. You have the analysis summaries from the previous step — use them as your map, not as a narrative to follow verbatim.

### Rules

- **Follow existing patterns.** Match the shape of nearby code. If the phase's integration points use a particular idiom, use the same idiom for consistency.
- **No tests in this step.** Test authoring is the next step and runs in a dedicated sub-agent context. Do not write `*_test.go` files yet.
- **No speculative abstractions.** Implement what the phase says, not what you think the phase should have said.
- **Small, readable diffs.** If a single phase balloons beyond the scope described in `{{context_path}}`, STOP and ask the user whether to split it.

### STOP-on-mismatch

If the code you need to modify has drifted materially from what the plan describes (file renamed, function signature changed, type removed, import path moved), STOP before making any change. Report the mismatch to the user with a concrete description of what the plan expected and what you found. Ask the user to pick one of three options:

1. **Fix the plan first.** Update `{{plan_path}}` and/or `{{context_path}}`, then restart this step.
2. **Proceed with an agreed-upon substitution.** The user tells you how to map stale names to current names.
3. **Skip this phase.** Return to the phase-selection step and pick the next unchecked phase.

Do not silently adapt — visible drift must be acknowledged.

### Advance

Once the code for the current phase compiles:

```
{{config.command}} implement goto --data '{"step":"{{next_step}}"}'
```

---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
