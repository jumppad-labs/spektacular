## Step {{step}}: {{title}}

Draft the **Testing Approach** section of `plan.md`.

### This section is high-level only

Describe the overall testing strategy and test types. This section is **high-level only**. Per-phase testing detail — which specific tests live in which specific files — stays in `context.md`.

If you find yourself writing "a test in file X asserts Y on line Z", stop and move that content to context.md.

### What to include

- The kinds of tests being added (unit, integration, contract, regression, end-to-end)
- Which components get the most coverage and why
- The load-bearing assertions — what, in plain language, the tests guarantee
- Where tests slot into existing test conventions in the project
- Any deliberate gaps (e.g. "not adding integration tests because the contract is exercised by unit tests")

### Account for the spec's success metrics

Walk every metric in the spec's **Success Metrics** section and make each one verifiable — do not let any metric drop. For each, state in this section how it will be checked:

- **Behavioural test** — when the metric can be asserted automatically (e.g. "responds within 100ms" → a latency assertion), say so and describe what the test guarantees. The implementer writes the actual test.
- **Manual — captured in the implementation test plan** — when the metric cannot be expressed as an automated behavioural test (load under real infrastructure, manual observation, production telemetry), flag it with exactly that phrase. The implement workflow produces a concrete test-plan artifact for these once the code exists, so do **not** write the procedure here — just classify the metric.

If the spec has no success metrics, note that there are none to verify. This metric→verification mapping is the handoff the implementer relies on: the implement workflow consumes the plan, not the spec, so a metric not carried here is invisible downstream.

### What NOT to include

- Specific test file paths
- Per-phase test lists
- Shell commands to run the tests

### What to produce

A draft Testing Approach section ready to drop into plan.md at verification time. Present it to the user for review.

Before advancing, save this section to its working file. Using your own `Write` tool, write the agreed **Testing Approach** content (body only — no `## ` heading line) to `.spektacular/work/{{plan_name}}/testing_approach.md`. This working file is git-tracked and is read back on resume and when the plan documents are assembled, so it must hold the final content. It is **not** a plan store document — write it directly with your file tools and do **not** route it through `{{config.command}} plan file write` (that command is only for the final plan documents).

Once the user is happy with the testing strategy, advance:

{{config.command}} plan goto --data '{"step":"{{next_step}}"}'

---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
