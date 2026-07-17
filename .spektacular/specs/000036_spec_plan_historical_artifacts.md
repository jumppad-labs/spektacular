# Feature: 000036_spec_plan_historical_artifacts

<!--
  OVERVIEW
  A concise 2-3 sentence summary of the feature. Answer three questions:
    1. What is being built?
    2. What problem does it solve?
    3. Who benefits and why does it matter?
  Avoid implementation details — this should be readable by any stakeholder.
-->
## Overview

Establish a clear rule for agents working in this repository: historical specifications and plans are archaeological records that explain *why* a past change was made, not a description of what the codebase currently does. When exploring the codebase or answering questions about current behavior, agents must ground their understanding in the code itself and avoid reading these documents from disk or through tools, reaching for them only when a user is genuinely investigating the history behind a past decision. This keeps agent responses aligned with reality — the shipped code — instead of drifting into stale intent that may have shifted during implementation or been abandoned entirely.

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

- [x] **Specs and plans are treated as historical, not current**
  Agents recognize that files in the specs and plans stores describe past intent behind changes, not current-state behavior of the codebase, and never cite them as authoritative for what the codebase does today.

- [x] **Agents do not read specs or plans during general codebase discovery**
  When exploring the codebase, searching for features, or answering questions about current behavior, agents do not open spec or plan files through any available channel.

- [x] **Historical archaeology is the only allowed reason to read a spec or plan**
  Agents may read a historical spec or plan only when the user is investigating why a past change was made the way it was, and never in service of describing what the codebase currently does.

- [x] **The rule applies everywhere agents operate, not just inside specific workflows**
  The guidance binds every agent operating in the repository — during ad-hoc questions, unrelated skills, and general exploration — not only when a workflow step happens to touch on spec/plan handling.

- [x] **Agents ground current-state answers in code**
  When asked what the codebase does today, agents derive the answer from source files, tests, and configuration — not from spec or plan documents that may describe features that were reshaped or never shipped.

- [x] **The rule is durably communicated so agents encounter it without being prompted**
  The rule is captured somewhere every agent working in this repository picks up automatically, so users do not have to restate it in each session.

- [x] **Specs and plans are operational only within their active workflow**
  A spec or plan is a live, authoritative artifact only while the workflow that owns it is actively driving it. During that time, the workflow itself must read and update the artifact; outside of that window — after completion, or by any agent that isn't the owning workflow — the artifact is historical and subject to the same rules as any other historical spec/plan.

- [x] **The changelog entry flags the concept for future website documentation**
  The changelog note that ships with this change explicitly calls out that the "process document, not product document" concept should also be added to the user-facing website documentation in a future update, so the follow-up is not lost when this spec closes.

<!--
  CONSTRAINTS
  Hard boundaries the solution must operate within. These are non-negotiable.
  Format: one bullet point per constraint.
  Examples:
    - Must integrate with the existing authentication system
    - Cannot introduce breaking changes to the public API
    - Must support the current minimum supported runtime versions
  Leave blank if there are no constraints.
-->
## Constraints

- The rule must be enforced at the agent-instruction level, not by filesystem or tool-configuration access blocks. Hard access blocks would break the archaeology exception, which requires that agents remain able to read historical specs and plans when a user is genuinely investigating past intent.

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

- [x] **Agent describes current-state features by citing code**
  In a fresh session, when asked "how does feature X work?", the agent's response cites paths under source, test, or configuration directories — not paths under `.spektacular/specs/` or `.spektacular/plans/`.

- [x] **No spec/plan reads during discovery-flavored tasks**
  During a session where the user asks the agent to explore, summarize, or explain the codebase with no historical intent expressed, the agent's tool-call log contains no read of any file under `.spektacular/specs/` or `.spektacular/plans/`, nor any `spec file read` / `plan file read` invocation.

- [x] **Historical questions unlock spec/plan access**
  When the user explicitly asks "why was X built this way?" or a similar historical/intent question, the agent then reads the relevant spec or plan and cites it as historical context for the past decision — not as a description of current behavior.

- [x] **The rule persists outside workflow steps**
  When the agent is not inside a spec/plan/implement workflow step (e.g., ad-hoc questions, unrelated skills, general exploration), the same restriction on spec/plan reading still applies, verifiable by prompting the agent in a session where no such workflow is active.

- [x] **Current-state answers cite code artifacts**
  When asked "what does the codebase do today?", the agent's response cites paths under source, test, or configuration directories, and does not cite any file under `.spektacular/specs/` or `.spektacular/plans/`.

- [x] **A fresh agent session already knows the rule**
  In a fresh agent session started in the repository, the agent already knows the rule without any user prompt — verifiable by asking the agent to summarize the rule and receiving an accurate answer.

- [x] **Workflow may access its own live artifact**
  During an actively running spec, plan, or implement workflow, the owning workflow successfully reads and updates the artifact it owns without violating the rule.

- [x] **Completed workflow artifacts do not appear in current-state answers**
  After a workflow closes, any agent session asked "what does this feature do?" answers from code and does not cite the completed spec or plan.

- [x] **Changelog entry contains the documentation follow-up note**
  The changelog entry that ships with this change contains an explicit note that the "process document, not product document" concept should be added to the user-facing website documentation in a future update.

<!--
  TECHNICAL APPROACH
  High-level technical direction to guide the planning agent. Include:
    - Key architectural decisions already made
    - Preferred patterns or technologies if known
    - Integration points with existing systems
    - Known risks or areas of uncertainty
  Format: one bullet point per direction/steer.
  Leave blank if you want the planner to propose the approach.
-->
## Technical Approach

- No technical direction has been decided; the detailed design is left for the plan workflow to propose.

<!--
  SUCCESS METRICS
  How you will know the feature is working well after delivery. Be specific:
    - Quantitative: "p99 latency < 200ms", "error rate < 0.1%"
    - Behavioural: "users complete the flow without support intervention"
  Format: one bullet point per metric.
  Leave blank if not applicable.
-->
## Success Metrics

- Users stop needing to remind agents mid-session that specs and plans are not current-state documentation.
- When agents describe existing features, their citations point to source code, tests, or configuration — not to files under `.spektacular/specs/` or `.spektacular/plans/`.
- When users ask historical or intent questions ("why was X built this way?"), agents correctly reach for the relevant spec or plan and cite it as historical context.

<!--
  NON-GOALS
  Explicitly state what this spec does NOT cover. This is as important as
  the requirements — it prevents scope creep and sets clear expectations.
  Format: one bullet point per exclusion.
  Examples:
    - "Mobile support is out of scope (tracked in #456)"
    - "Internationalisation will be addressed in a follow-up spec"
  Leave blank if there are no explicit exclusions to call out.
-->
## Non-Goals

- Updating the user-facing website documentation to introduce the "process document, not product document" concept is deferred to a follow-up; only the changelog note flagging the concept lands in this spec.
