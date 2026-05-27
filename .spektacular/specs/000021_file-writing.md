# Feature: 000021_file-writing

<!--
  OVERVIEW
  A concise 2-3 sentence summary of the feature. Answer three questions:
    1. What is being built?
    2. What problem does it solve?
    3. Who benefits and why does it matter?
  Avoid implementation details — this should be readable by any stakeholder.
-->
## Overview

Spektacular's file-writing commands currently require the file body to be passed inline through the shell, which forces agents to escape special characters and risks producing malformed files. This feature lets the agent point Spektacular at a source file on disk instead, so the contents are written through verbatim. Both the agent producing the content and the humans (or agents) later reading those files benefit, because the output is reliably well-formed.

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

- [ ] **Accept a source file on disk as the input for spec and plan file writes**
    The caller points the file-writing command at a path on disk and the source file's contents are written through verbatim into the spec/plan store.
- [ ] **Remove inline/stdin input for file writes**
    The previous stdin-based input path is no longer accepted by the file-writing commands.
- [ ] **Report a clear error when the source file cannot be read**
    Missing files, unreadable files, or permission errors surface as actionable errors instead of silently writing empty or partial content.
- [ ] **Leave the caller's source file in place after a successful write**
    Spektacular does not delete, move, or otherwise modify the source file the caller pointed it at.
- [ ] **Update all agent-facing instructions to describe the new input method**
    Skills, prompts, and documentation that previously showed the stdin/heredoc usage are updated to show the file-path-based usage, instruct the agent to place its working file under a designated scratch location inside the project, and remind the agent to remove the working file after a successful write.

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

- [ ] **Source-file input writes content verbatim**
    Given a source file containing arbitrary content (including characters that would be problematic for shell escaping — backticks, `$`, single and double quotes, embedded newlines), invoking the spec or plan file write command pointed at that path produces a file in the spec/plan store whose bytes are identical to the source file.
- [ ] **Stdin input is no longer accepted**
    Invoking the spec or plan file write command with content piped to stdin (the previous interface) exits non-zero with an error and leaves the spec/plan store unchanged.
- [ ] **Unreadable source files produce a clear error**
    When the command is pointed at a path that does not exist or cannot be read, it exits non-zero with an error message that names the offending path and the reason, and nothing is written to the spec/plan store.
- [ ] **Source file is preserved**
    After a successful write, the source file the caller pointed at still exists on disk with byte-identical content to what it had before the call.
- [ ] **Agent-facing instructions reference only the new input method**
    No skill, prompt, or documentation file in the project still contains an example of the stdin/heredoc usage for spec or plan writes; every such file shows the file-path-based usage, names the scratch location agents should use, and tells the agent to remove the working file after a successful write.

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

The existing `spec file write` (and equivalent plan-writing) command reads the file body from stdin, which forces the calling agent to use shell heredocs and to escape any tricky characters in the content. Replace that with a source-file argument: the caller writes its content to a path on disk (by convention under a scratch directory inside the project such as `.spektacular/tmp`) and Spektacular copies that content through verbatim, bypassing shell quoting entirely. Agent-facing instructions (skills, prompts) need to be updated in lockstep so that they direct agents at the new input mechanism and remind them to clean up their working files.

<!--
  SUCCESS METRICS
  How you will know the feature is working well after delivery. Be specific:
    - Quantitative: "p99 latency < 200ms", "error rate < 0.1%"
    - Behavioural: "users complete the flow without support intervention"
  Leave blank if not applicable.
-->
## Success Metrics


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
