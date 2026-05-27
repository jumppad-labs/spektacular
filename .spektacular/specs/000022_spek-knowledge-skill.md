# Feature: 000005_spek-knowledge-skill

## Overview

Enable the user to read from and contribute to the project's persistent
knowledge base at any time during a session, without needing to start a
formal spec or plan workflow. Today, knowledge-base interaction only
happens as a side-effect of the planning workflow, so insights gained
during ad-hoc work go uncaptured and prior learnings aren't surfaced
when they would be useful. This benefits the user during exploratory or
vibe-coding sessions where valuable conventions, gotchas, or
architecture notes emerge but there is no plan in flight to capture
them.

## Requirements

- [ ] **Look up knowledge base entries on demand**
  The user can ask to search, read, or browse knowledge base entries
  at any point in a session, independent of whether a planning
  workflow is active.

- [ ] **Contribute new knowledge entries on demand**
  The user can ask to record a new entry — a learning, gotcha,
  convention, or architecture note — at any point in a session,
  independent of whether a planning workflow is active.

- [ ] **Update existing knowledge entries**
  The user can revise the contents of an entry that already exists.
  This is distinct from creating a new entry: the user can ask to
  amend, append to, or replace prior content.

- [ ] **Review before persistence**
  Any proposed write or update is shown to the user for explicit
  approval before it is committed. No entry is created or modified
  without confirmation.

- [ ] **Scope selection is explicit**
  When contributing, the user is informed of the available scopes
  and chooses which scope the entry belongs to.

- [ ] **Search results identify their scope**
  When entries are surfaced from a lookup, each result indicates
  which scope it came from so the user understands its provenance.

- [ ] **Invocation by natural intent**
  The user can trigger the workflow by expressing their intent in
  natural language about the knowledge base, without needing to
  recall a specific command or syntax. The trigger works the same
  way regardless of which supported agent is driving the session.

- [ ] **Available immediately after project initialization**
  When a Spektacular project is initialized for a supported agent,
  the knowledge workflow is available without further setup. The
  user does not need to install anything manually after project
  initialization.

## Constraints

- [ ] **Must support all Spektacular-supported agents**
  The solution cannot be coupled to a single agent's
  instruction-loading mechanism. It must work the same for any
  agent Spektacular supports (currently claude, codex, bob).

- [ ] **Must not disrupt an in-progress workflow**
  When a spec, plan, or implement workflow is already active in the
  session, invoking this feature must not corrupt or advance that
  workflow's state.

## Acceptance Criteria

- [ ] **Lookup returns matching entries**
  After the user makes a lookup request with a query, entries from
  the knowledge store matching that query are returned to the user
  in the same session, with no spec or plan workflow active.

- [ ] **Accepted contribution is persisted**
  After the user accepts a proposed contribution, an entry exists
  in the configured knowledge store containing the title and body
  the user approved.

- [ ] **Accepted update replaces prior content**
  After the user accepts a proposed update to an existing entry,
  reading that same entry returns the new content rather than the
  prior content.

- [ ] **Declined or modified proposals do not persist**
  If the user declines a proposed contribution or update, or asks
  for changes before approving, the knowledge store is unchanged
  from its state before the proposal.

- [ ] **Scope is chosen before write**
  Before any contribution or update is written, the user is shown
  the list of available scopes; the entry that ends up persisted is
  located in the scope the user selected.

- [ ] **Lookup results are scope-labeled**
  Every entry returned to the user from a lookup is accompanied by
  the name of the scope it came from.

- [ ] **Same user intent produces same outcome across agents**
  When the user expresses the same knowledge-base intent in a
  session driven by any supported agent, the resulting effect on
  the knowledge store (entry written, entry read, results returned)
  is the same.

- [ ] **Initialization installs the workflow surface**
  After a Spektacular project is initialized for any supported
  agent, the user can invoke the knowledge workflow in a new
  session with that agent without performing any additional
  installation step.

## Technical Approach

The workflow lives inside the Spektacular CLI itself rather than in
any single agent's instruction-loading mechanism. This is what makes
the agent-agnostic constraint achievable: the same proven pattern
already used by spec, plan, and implement — an agent reads an
instruction from the CLI, executes it, and advances state — drives
this workflow too, so it works the same for any agent Spektacular
supports.

Each agent surface (Claude Code skills, equivalent for codex/bob)
contains only a thin shell that hands control to the CLI and loops
on its instructions, never the workflow logic itself.

Persistence reuses the existing `spektacular knowledge` storage
already configured for the project. No parallel store, no new config
keys.

The propose-then-confirm pattern documented in the plan workflow's
discovery step — never write to the knowledge store without explicit
user confirmation — applies here too.

Distribution piggybacks on the existing project-init step that
already installs the agent surfaces for spec, plan, and implement.
The knowledge workflow's surface is installed alongside them, so no
separate user action is required after initialization.

## Success Metrics

- The knowledge store grows in use: entries written through the
  workflow accumulate over time rather than the store staying empty.

- The user invokes the workflow during ad-hoc sessions (not only
  during plan flows), confirming the gap this feature targets is
  real.

- Subsequent plan workflows surface prior entries that materially
  shorten research — i.e. captured knowledge actually pays off
  downstream.

- The user does not bypass the workflow by editing knowledge files
  directly with the editor.

## Non-Goals

- Does not replace the knowledge-base interactions already embedded
  in the plan workflow. This is additive for ad-hoc use.

- Does not silently capture knowledge from conversation context.
  Every write requires explicit user confirmation — no autonomous
  extraction.
