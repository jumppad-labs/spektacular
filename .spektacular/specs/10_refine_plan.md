# Feature: 14 Refine Plan

<!--
  OVERVIEW
  A concise 2-3 sentence summary of the feature. Answer three questions:
    1. What is being built?
    2. What problem does it solve?
    3. Who benefits and why does it matter?
  Avoid implementation details — this should be readable by any stakeholder.
-->
## Overview

Plan mode should have an interactive feedback loop that allows users to refine generated plans. Currently, once a plan is generated, there is no way for the user to request targeted changes — they must regenerate the entire plan from scratch. This feature introduces a conversational flow where the LLM asks clarifying questions if needed and then updates specific parts of the plan based on user feedback. Users can also annotate or comment directly on the plan itself, which the agent reads and acts on. This benefits developers using the planning workflow by giving them iterative control over plan output without losing prior work.


<!--
  REQUIREMENTS
  Specific, testable behaviours the feature must deliver.
  Format: bold title on the checkbox line, detail indented below.
  Rules:
    - Use active voice: "Users can...", "The system must..."
    - Each requirement should be independently verifiable
    - Focus on WHAT, not HOW — avoid prescribing implementation
    - Keep each item atomic — one behaviour per line
-->
## Requirements

- [ ] **Conversational feedback after plan generation** — Users can provide feedback to the agent via a text interface in the CLI after a plan has been produced.
- [ ] **Clarification requests** — The agent must request clarification from the user when feedback is ambiguous or not well understood before making changes.
- [ ] **Plan regeneration from feedback** — The agent must regenerate the plan incorporating the user's feedback once sufficient information has been collected.
- [ ] **Iterative feedback cycles** — Users can provide multiple rounds of feedback until they are satisfied the plan is correct; the process is not limited to a single revision.
- [ ] **Direct plan file editing** — Users can edit the plan files directly to provide feedback or make changes, as an alternative to the text interface.
- [ ] **Re-read plan files before acting** — The agent must always re-read the plan files from disk before executing on feedback, as the user may have edited them without notifying the agent.

<!--
  CONSTRAINTS
  Hard boundaries the solution must operate within. These are non-negotiable.
  Examples:
    - Must integrate with the existing authentication system
    - Cannot introduce breaking changes to the public API
    - Must support the current minimum supported runtime versions
  Leave blank if there are no constraints.
-->
## Constraints

- Must integrate into the existing plan workflow.

<!--
  ACCEPTANCE CRITERIA
  The specific, binary conditions that define "done".
  Format: bold title on the checkbox line, verifiable detail indented below.
  Each criterion must be:
    - Independently verifiable (pass/fail, not subjective)
    - Traceable back to a requirement above
    - Testable by someone who didn't write the code
-->
## Acceptance Criteria

- [ ] **Feedback prompt after plan completion** — When a plan has completed, the user is presented with the message "Please read the plan and provide me any feedback".
- [ ] **Text-based feedback entry** — The user can provide feedback via a text entry in the CLI.
- [ ] **Clarification for unclear feedback** — When the user's feedback is ambiguous, the agent asks the user to clarify before making changes.
- [ ] **No unnecessary clarification** — When the user's feedback is clear, the agent does not ask for clarification and proceeds directly.
- [ ] **Plan files modified after feedback** — After collecting feedback, the agent modifies the plan files to incorporate the requested changes.
- [ ] **No changes without feedback** — If the user provides no feedback, the agent does not modify the plan files.
- [ ] **Post-regeneration prompt** — After modifying the plan, the agent displays "I have modified the plan based on your feedback, please take a look and let me know of any changes".
- [ ] **Further feedback accepted** — After plan regeneration, the user can provide additional rounds of feedback without restarting the process.
- [ ] **Direct file editing supported** — In addition to textual feedback, the user can edit plan files directly to provide feedback or make changes.
- [ ] **Plan files always re-read from disk** — The agent always reads plan files from disk before acting on feedback; it never assumes in-memory plan content is current.
- [ ] **No assumption of user notification** — The agent never assumes the user will tell it that plan files have been changed externally.

<!--
  TECHNICAL APPROACH
  High-level technical direction to guide the planning agent. Include:
    - Key architectural decisions already made
    - Preferred patterns or technologies if known
    - Integration points with existing systems
    - Known risks or areas of uncertainty
  Leave blank if you want the planner to propose the approach.
-->
## Technical Approach

None.

<!--
  SUCCESS METRICS
  How you will know the feature is working well after delivery. Be specific:
    - Quantitative: "p99 latency < 200ms", "error rate < 0.1%"
    - Behavioural: "users complete the flow without support intervention"
  Leave blank if not applicable.
-->
## Success Metrics

None.

<!--
  NON-GOALS
  Explicitly state what this spec does NOT cover. This is as important as
  the requirements — it prevents scope creep and sets clear expectations.
  Examples:
    - "Mobile support is out of scope (tracked in #456)"
    - "Internationalisation will be addressed in a follow-up spec"
  Leave blank if there are no explicit exclusions to call out.
-->
## Non-Goals

None.
