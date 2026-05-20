# Feature: 20_context

<!--
  OVERVIEW
  A concise 2-3 sentence summary of the feature. Answer three questions:
    1. What is being built?
    2. What problem does it solve?
    3. Who benefits and why does it matter?
  Avoid implementation details — this should be readable by any stakeholder.
-->
## Overview

Spektacular produces better plans when it can draw on the architectural decisions, team conventions, and hard-won learnings that live outside the codebase. Today this knowledge must sit alongside each project on disk, which forces copies between projects and makes sharing across engineers awkward. This feature lets each project choose where its knowledge is stored and searched — a local folder, a shared team source, or somewhere else entirely — so teams can collaborate on a single body of knowledge when it helps and keep it private when it doesn't.


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

- [ ] **A project can configure multiple knowledge sources across distinct scopes**
  Scopes such as global, team, and project can coexist for a single project, and each scope can be backed by a different source type.
- [ ] **Spektacular reads knowledge from every configured source**
  When a workflow assembles context, it draws on all sources configured for the project regardless of scope.
- [ ] **Spektacular supports keyword search across all configured sources**
  A free-form query returns excerpts from every configured source, each tagged with its originating store/scope, which an agent can use to decide which entries to read in full.
- [ ] **Search results are optimised for token efficiency**
  Excerpts are compact enough that an agent can scan many results before fetching full content.
- [ ] **Spektacular can write new knowledge to a chosen source**
  The agent proposes a target scope based on the content, and the user must confirm the target and the content before anything is persisted.
- [ ] **Each source is configured independently**
  The type, location, and any credentials of one source do not constrain the configuration of another.
- [ ] **Overlapping entries across sources remain discoverable**
  If two scopes hold entries on the same topic, both remain reachable via reads and search.
- [ ] **The store extension pattern is documented in the README**
  The README explains the `Store` interface contract and walks through the local filesystem backend (`FileStore`) as a worked example of how to implement and register a new store backend.


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

None. Breaking changes to the existing knowledge layout are acceptable, and a network dependency is acceptable.

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

- [ ] **Multiple sources at distinct scopes**
  Given a project configured with at least two scopes (each backed by its own configured source), listing the configured sources returns an entry for each scope with its location reported.
- [ ] **Reads span every configured source**
  Given entries exist in two or more configured scopes, a context-assembly operation for a workflow returns entries from every configured scope in a single result set.
- [ ] **Cross-source search returns per-store results**
  Given entries in multiple sources, a free-form query returns results from each configured store, every result tagged with the store/scope it came from and carrying a locator and a short excerpt — and does not include the full entry body.
- [ ] **Token-efficient excerpts**
  Every excerpt returned by search stays within a fixed compact budget (exact size set during planning) so an agent can review many results before fetching a full entry.
- [ ] **Confirmed write**
  An agent-proposed write only persists after an explicit user confirmation; a declined confirmation leaves every source unchanged.
- [ ] **Write lands in the chosen scope only**
  After a confirmed write, a subsequent read or search returns the new entry from the confirmed scope, and no other scope is modified.
- [ ] **Independent source configuration**
  Changing the location of one configured source has no effect on reads, writes, or searches against any other configured source.
- [ ] **Overlapping entries both surface**
  Given two sources contain entries on the same topic, a search for that topic returns both hits, each tagged with its originating scope.
- [ ] **Unreachable sources surface clearly**
  When a configured source cannot be reached at the time of a read, search, or write, the operation fails with a message identifying the unreachable source rather than silently returning partial results.
- [ ] **Store extension pattern is documented**
  The README contains a section presenting the `Store` interface (including `Search`) and walking through `FileStore` as the worked example, such that a developer can implement and register a new backend by following it without reading an existing backend's source separately.


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

- **Reuse the existing `store.Store` interface — do not introduce a separate knowledge interface.** A store is just a store; the implementation behind it (local filesystem, remote, etc.) is irrelevant to consumers. Add a `Search` operation to `Store` so search is a first-class capability every backend can serve in whatever way suits it (local: keyword scan; a future remote backend: API query).
- **`Search` is local to each store and returns generic hits.** Each store searches only itself; there is no central search layer. A hit carries the originating store/scope, a locator (path), and a short excerpt — never the full body, and never a structured "title" or "abstract". A backend may attach a relevance score if it can produce one cheaply, but ranking is per-store, not global.
- **The agent consumes search hits and decides what to do.** A multi-store search runs each configured store's own search and concatenates the per-store results, each tagged with its store/scope. An LLM scans the excerpts and chooses which entries to read in full via `Store.Read`. There is no separate ranking or abstract-generation layer; the excerpt is whatever the backend can cheaply produce.
- **A project configures multiple stores, one per scope** (e.g. global, team, project). Scopes are labels on configured stores; backend type is orthogonal.
- **Ship one backend in this feature: local filesystem.** `FileStore` already implements `Store`; it gains `Search`. Multiple scopes are served by pointing separate `FileStore`s at different directories. A remote backend is deferred — but because it is just another `Store` implementation, adding it later also makes the whole project directory remotable, not only knowledge.
- **`FileStore.Search` prefers ripgrep, with a native Go fallback.** ripgrep (`rg`) is the fastest practical way to scan files and there is no Go-native equivalent, so when `rg` is on `PATH` the backend shells out to it and parses its structured `--json` output. When `rg` is absent, `FileStore` falls back to a pure-Go directory walk and line scan that returns equivalent hits. The fallback keeps the feature self-contained with no hard runtime dependency on an external binary; the planning pass should confirm whether a maintained Go search library could replace the shell-out before settling on it.
- **Knowledge-entry structure is a file-format convention, not part of the interface.** Any richer shape (frontmatter title/tags, a semantic summary) lives in the knowledge files themselves and is parsed by a thin layer on top of `Store` when needed — it does not leak into the generic store interface.
- **Search excerpts stay within a compact token budget** so the agent can scan many hits before fetching a full entry.

<!--
  SUCCESS METRICS
  How you will know the feature is working well after delivery. Be specific:
    - Quantitative: "p99 latency < 200ms", "error rate < 0.1%"
    - Behavioural: "users complete the flow without support intervention"
  Leave blank if not applicable.
-->
## Success Metrics

- **Planning quality** — When knowledge relevant to a feature exists in a configured source, plans generated for that feature cite or incorporate it rather than rediscovering or ignoring it. Measured by spot-checking plans produced after delivery against the knowledge that was available when they were generated.
- **Search efficiency** — An agent can locate a relevant knowledge entry by searching and reviewing excerpts without loading the full body of unrelated entries. Measured by the total tokens consumed to answer a typical knowledge lookup staying within a fixed budget set during planning.

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

- **Non-filesystem backends** (e.g. GitHub-hosted knowledge) — the interface is designed so they can be added, but only the local filesystem backend ships in this feature. Other backends are tracked for a later spec.
- **Offline operation and caching for remote sources** — moot until a remote backend exists; revisit when one is added.
- **Formal precedence rule for overlapping entries across scopes** — both entries surface from search, but a "project overrides global" ruleset is deferred to a later spec.
- **Migration of pre-existing `.spektacular/knowledge/` layouts** — breaking changes to the on-disk layout are acceptable, so no migration tooling or compatibility shim is in scope.
- **Automatic knowledge capture** — writes are agent-proposed and user-confirmed only in this feature; harvesting learnings without user confirmation is out of scope.
