# Feature: 000026_ripgrep-replace

<!--
  OVERVIEW
  A concise 2-3 sentence summary of the feature. Answer three questions:
    1. What is being built?
    2. What problem does it solve?
    3. Who benefits and why does it matter?
  Avoid implementation details — this should be readable by any stakeholder.
-->
## Overview

Spektacular's file store search currently depends on an external command-line
tool being installed on the user's machine. This feature replaces that
dependency with a built-in search capability so the same functionality is
provided entirely within Spektacular itself.

This removes the need to install any external tool to use Spektacular. Users get
a self-contained product that works out of the box on any machine, with no
separate setup and no failures when an external tool is missing or the wrong
version.


<!--
  REQUIREMENTS
  Specific, testable behaviours the feature must deliver.
  Format: bold title on the checkbox line, detail indented below.
  Rules:
    - Use active voice: &#34;Users can...&#34;, &#34;The system must...&#34;
    - Each requirement should be independently verifiable
    - Focus on WHAT, not HOW — avoid prescribing implementation
    - Keep each item atomic — one behaviour per line
-->
## Requirements

- Users can perform file store search on a machine with no external tools
  installed.
- The system must perform all search work in-process, without invoking any
  external program.
- Search results must be functionally equivalent to the current behaviour —
  matching the same content with the same case-insensitivity and the same
  excluded areas — so users notice no meaningful change in what they find.


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

- Search must not require, invoke, or depend on any external tool or process. It
  must be self-contained within Spektacular's own binary.

<!--
  ACCEPTANCE CRITERIA
  The specific, binary conditions that define &#34;done&#34;.
  Format: bold title on the checkbox line, verifiable detail indented below.
  Each criterion must be:
    - Independently verifiable (pass/fail, not subjective)
    - Traceable back to a requirement above
    - Testable by someone who didn&#39;t write the code
-->
## Acceptance Criteria

- On a machine with no external search tool installed, running a search over a
  populated store returns matching results — proving search requires no external
  process.
- For a given query and store contents, the search returns the same set of
  matches as the current behaviour (same files, same case-insensitive matching),
  and content that is currently excluded from search (such as convention files)
  remains excluded.


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

Replace the external search tool with an equivalent code-based (in-language)
search implementation.

Prefer reusing an existing search library if one of sufficient quality exists.
If no suitable library is found, building a similar implementation modelled on
the ripgrep Rust source is acceptable.

The detailed design is left for the plan workflow to propose.


<!--
  SUCCESS METRICS
  How you will know the feature is working well after delivery. Be specific:
    - Quantitative: &#34;p99 latency < 200ms&#34;, &#34;error rate < 0.1%&#34;
    - Behavioural: &#34;users complete the flow without support intervention&#34;
  Leave blank if not applicable.
-->
## Success Metrics

- Spektacular has zero external runtime dependencies for search: a fresh install
  on a clean machine performs search successfully with nothing else installed.
- No regression in search results: the existing search-related tests pass with
  the built-in implementation.


<!--
  NON-GOALS
  Explicitly state what this spec does NOT cover. This is as important as
  the requirements — it prevents scope creep and sets clear expectations.
  Examples:
    - &#34;Mobile support is out of scope (tracked in #456)&#34;
    - &#34;Internationalisation will be addressed in a follow-up spec&#34;
  Leave blank if there are no explicit exclusions to call out.
-->
## Non-Goals

- No new search capabilities. This is a like-for-like replacement, not an
  opportunity to add features such as regex search, fuzzy matching, or relevance
  ranking. Behaviour stays equivalent to today.
- Performance improvement is not a goal. Matching current behaviour is the bar;
  making search faster, or guaranteeing it is as fast as the external tool, is
  out of scope.
