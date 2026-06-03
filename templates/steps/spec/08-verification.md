## Step {{step}}: {{title}}

Assemble the final spec from the per-section working files you wrote during the earlier steps. Read each section's working file under `.spektacular/work/{{spec_name}}/` and drop its content under the matching `## ` heading in the scaffold below. The working files hold body content only — this scaffold owns the headings and their order:

- `.spektacular/work/{{spec_name}}/overview.md` → `## Overview`
- `.spektacular/work/{{spec_name}}/requirements.md` → `## Requirements`
- `.spektacular/work/{{spec_name}}/constraints.md` → `## Constraints`
- `.spektacular/work/{{spec_name}}/acceptance_criteria.md` → `## Acceptance Criteria`
- `.spektacular/work/{{spec_name}}/technical_approach.md` → `## Technical Approach`
- `.spektacular/work/{{spec_name}}/success_metrics.md` → `## Success Metrics`
- `.spektacular/work/{{spec_name}}/non_goals.md` → `## Non-Goals`

If a working file is missing, that section was never completed — STOP and gather it (re-run the matching step) before assembling.

```markdown
{{spec_template}}
```

Review the assembled spec and validate it for:
• Completeness — all sections are filled
• Clarity — requirements are specific and testable
• Consistency — sections reference each other appropriately
• Section hygiene — each section stays within its brief, with no implementation leaking into sections that should be implementation-free

### Section hygiene check

Implementation detail tends to drift upward from Technical Approach into Overview, Requirements, Constraints, and Acceptance Criteria. Run each section against its own test and flag any leaks to the user as a proposed rewrite — with the leaked detail moved to Technical Approach, not deleted.

**Overview** — no file paths, section names, step names, framework/library names, or code identifiers. A non-engineer stakeholder should be able to read it and understand the value.

**Requirements** — apply the test: *if this were implemented with a completely different mechanism, would the requirement still be true?* If no, rephrase at the behavior level and move the mechanism to Technical Approach. Warning signs: specific file paths, section or heading names, step or state names, numeric step positions.

**Constraints** — apply the test: *if this constraint were removed, would the feature become impossible, or just implemented differently?* Only the first belongs here. "Must use X" items are almost always design decisions, not constraints — move them to Technical Approach.

**Acceptance Criteria** — each criterion must be observable from outside the implementation. "File X exists after user accepts" is good. "FSM rejects step Y from source state Z" is a unit test of internal plumbing — rephrase as an observable outcome.

**Technical Approach** — this is where everything the other sections rejected should live. It's the one section where file paths, state names, library choices, and concrete mechanisms are welcome.

For every leak you find, report it to the user in this shape:

> *"In <section>, item N names <specific implementation detail>. I'd like to rephrase it as <behavior-level version> and move the implementation detail to Technical Approach. OK?"*

Do not silently rewrite — the user may have intentionally locked in a design decision. Propose, then wait for confirmation.

Report any issues to the user and ask for clarification until you are confident the spec is correct, complete, and free of section-hygiene leaks.

Once the user is happy, produce the final complete spec and commit it to the spec store.

**Never edit the spec file with the `Write` or `Edit` tools.** `{{config.command}} spec file write` is the only supported way to write the spec — it routes the write through Spektacular into the configured spec directory.

Stage the content through a scratch file first. Use the `Write` tool to write the completed spec to the scratch path `.spektacular/tmp/spec_template.md`, point `spec file write` at it with `--from`, then remove the scratch file:

```
{{config.command}} spec file write {{spec_name}}.md --from .spektacular/tmp/spec_template.md
rm .spektacular/tmp/spec_template.md
```

That writes the final spec into the configured spec directory through Spektacular. The path argument is the spec file name only — `spec file write` resolves it against the configured spec directory for you.

After `spec file write` succeeds, verify the stored spec with `{{config.command}} spec file read {{spec_name}}.md`. Once you have confirmed it is correct, the per-section working files have served their purpose — remove the working directory with your own file tools:

```
rm -rf .spektacular/work/{{spec_name}}
```

Then advance:

```
{{config.command}} spec goto --data '{"step":"{{next_step}}"}'
```


---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
