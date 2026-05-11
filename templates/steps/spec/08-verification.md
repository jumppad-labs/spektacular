## Step {{step}}: {{title}}

Complete the following template with all of the information you have gathered so far

```markdown
{{spec_template}}
```

Review all the information gathered across every step and validate the complete spec for:
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

Once the user is happy, produce the final complete spec. Use the `Write` tool to stage it at `.spektacular/tmp/spec_template.md`, then submit it:

```
{{config.command}} spec goto --data '{"step":"{{next_step}}"}' --file .spektacular/tmp/spec_template.md
```

Spektacular reads the file and writes the final spec to the store. The `--file` flag is required here (not `--stdin`) because large specs exceed the tool-call size limit when inlined as a heredoc.

If `{{config.command}} notion status` reports Notion mode as configured, update the Notion Spec page through Notion MCP before submitting this file. Include the returned remote metadata in the `spec goto` `--data` payload so Spektacular records the cache baseline. If `notion cache prepare-push` returns `merge_required`, stop and resolve that merge before advancing.
