# Plan: 000029_readme-refresh

<!-- Metadata -->
<!-- Created: 2026-06-29T11:18:25Z -->
<!-- Commit: 50c13368beeda4a99e71250587ac56602e59b838 -->
<!-- Branch: main -->
<!-- Repository: git@github.com:jumppad-labs/spektacular.git -->

## Overview

Spektacular's root `README.md` has drifted out of step with the tool: it still describes a terminal UI, complexity scoring, and model-routing that never shipped, lists coding agents that aren't supported, and points at a stale repository path and version. This plan refreshes the README into an accurate, concise technical front door — sourced entirely from the documentation site (`spektacular.dev`) and the tool's current behaviour — so a newcomer can correctly understand what Spektacular is, which agents it supports (claude, bob, codex), and the spec → plan → implement workflow, while linking to the docs for depth. No change to the documentation site or the tool's code is required.

## Conventions

No project conventions apply to this feature. The project's always-applied knowledge contains only a placeholder `conventions/README.md` with no populated conventions, so there are no standing rules to reflect in this documentation refresh.

## Architecture & Design Decisions

The work refreshes the single root `README.md` (GitHub-flavoured Markdown) so that every claim agrees with the documentation site at `https://spektacular.dev` and the tool's current behaviour, without changing the site or the code. The chosen direction is a **section-driven rewrite**: the README is reorganised so its headings map one-to-one onto the required topics — introduction/main description, how it works, install & getting started (combined), supported agents, knowledge base, configuration, project structure, testing, and building/contributing/license — rather than patching the existing layout line by line. Accurate prose that already exists (the testing and building sections) is revised in place rather than rewritten, per the spec's technical approach; stale sections (knowledge, agents, install) are rewritten from the docs site and code; and invented sections are deleted outright.

The README is positioned as a **concise technical front door, not a manual**. It may describe the mechanism at a technical level — the self-contained Go binary plus the agent skills it installs, and the state-machine-driven workflow that drives the coding agent step by step through per-step prompts — but for in-depth topics (installation, getting started, full configuration, the command reference) it orients the reader briefly and links to the corresponding `spektacular.dev` page rather than reproducing that content. The supported-agents and knowledge-base sections deliberately go one level deeper (how the capability works and how to extend it) before linking out, as the acceptance criteria require.

Three load-bearing design decisions: (1) **Invented capabilities are removed, not corrected** — the terminal UI section, complexity scoring, and model-routing-by-complexity have no backing in `cmd/` or the docs site and are deleted; the supported-agent list is fixed to exactly claude, bob, and codex (`internal/agent/agent.go:28-56`), dropping Aider/Cursor. (2) **Examples are sourced from code, not the docs site, where the two disagree** — the search-result example uses the current 7-field `Hit` shape (`internal/store/store.go:24-32`), not the stale 4-field example on the site's extending page; the spec forbids fixing the site (a non-goal) and requires examples to match current behaviour, so this divergence is intentional and expected. (3) **Identity and version are made drift-resistant** — every repository/module reference uses the canonical `github.com/jumppad-labs/spektacular` (`go.mod:1`, replacing the stale `nicholasjackson` URLs at `README.md:47,55`), and rather than re-pinning a fast-moving version number (how `v0.1.0` went stale) the README shows an "early development" status and links to the releases page, satisfying the "no version older than the current release" criterion by stating none.

This direction beats the minimal surgical patch because the spec's acceptance criteria are framed as a checklist of required sections with specific depth (e.g. the agents and knowledge sections must go beyond a one-line mention); a line-by-line patch of the current heading layout would leave structural gaps and is more likely to miss a required topic. It beats a from-scratch rewrite by preserving the already-accurate testing and building content. See `research.md#alternatives-considered-and-rejected` for the full evidence behind each rejected option.

## Component Breakdown

The single deliverable is the root `README.md`. Its "components" are the document's sections; each owns one required topic and, for in-depth topics, hands off to a specific `spektacular.dev` page rather than reproducing it.

- **Header & status** — Owns project identity: name, one-line tagline, an "early development" status, license, and the canonical repository. Links to the releases page for the current version (no hardcoded number). Sets the front-door tone the rest of the document follows.
- **What is Spektacular (introduction)** — Owns the technical overview of core competencies: the self-contained binary plus the agent skills it installs, the state-machine-driven workflow that drives the agent step by step through per-step prompts, agent-agnostic multi-agent support, and the knowledge base. The orienting section every later section elaborates.
- **How it works** — Owns the workflow narrative: spec → plan → implement, driven step by step by the state machine, with implementation a shipped capability validated against acceptance criteria. Expands the introduction's workflow claim and links to the how-it-works page for the full pipeline detail.
- **Install & getting started** — Owns first-run orientation (it's a Go binary; brew / `go install` / releases) and the minimal init→spec→plan→implement path. Deliberately brief: defers to the install and getting-started pages for full instructions. Combined into one section per the spec.
- **Supported agents** — Owns the agent story one level deep: the three agents (claude, bob, codex), what initialising a project sets up for each (including that Claude init ensures the project's agent rules are loaded), how the agent integration works, and how to add/extend an agent. Links to the plugins/extending pages. Shares the extensibility note with the knowledge section.
- **Knowledge base** — Owns the KB explanation one level deep: the six-category model, the two retrieval tiers (always-applied vs looked-up), category-labelled search, and consolidated/de-duplicated lookups across scopes. Where it shows a search result it uses the current `Hit` shape. Links to the knowledge-base page for full detail.
- **Configuration** — Owns what is configurable: the agent and the spec/plan/knowledge stores and related settings. Orients only, then links to the configuration reference. Cross-references the project-structure section (config lives in `.spektacular/`).
- **Extensibility note** — Owns the statement that storage backends and agents are pluggable behind defined interfaces, pointing to the extending/plugins docs. Folded into the agents and knowledge sections rather than standing alone, per the spec.
- **Project structure** — Owns the description of what initialising a project creates on disk (the `.spektacular/` layout). Grounds the configuration and getting-started sections.
- **Testing** — Owns how the project is tested: the unit suite and the end-to-end harness and its modes. Revised in place from the current (accurate) content.
- **Building, contributing & license** — Owns how to build from source, how to contribute, and the license statement. Revised in place from the current (largely accurate) content.

## Data Structures & Interfaces

This feature introduces no new data structures or interfaces and changes none — it edits a single Markdown document and touches no code. The constraint is the reverse of a normal plan: the README must *depict* existing contracts exactly as the tool emits them today, since any example shown must match current behaviour.

One existing contract is reproduced as an example and must match the code verbatim: the knowledge **search-result (`Hit`)** shape. The README's knowledge-base example must show the current seven-field form, not the older four-field form documented on the site's extending page:

```
Hit {
  scope     // scope label of the originating store (e.g. project, team)
  path      // locator relative to the store root (e.g. gotchas/db-timeouts.md)
  title     // document's first heading, or the locator when it has none
  excerpts  // list of compact matched excerpts
  score     // sum of query-term occurrences (ranking)
  category  // category derived from the path (e.g. gotchas, architecture)
  checksum  // SHA-256 over the entry's raw bytes; the byte-identity de-dup key
}
```

Two other surfaces the README references are existing, unchanged interfaces — named only so the extensibility note describes them correctly, not redefined here: the **Store** interface (the read/write/search surface backing the spec, plan, and knowledge stores) and the **Agent** interface (the narrow contract each coding agent implements to declare its name and install its workflow artefacts). The README points to the extending documentation for their full signatures rather than reproducing them.

## Implementation Detail

The change is confined to one Markdown file, so there are no code patterns to introduce. The "patterns" here are editing disciplines that keep the result accurate and maintainable.

**Source-of-truth verification.** Every factual claim in the refreshed README must trace to one of two authorities: a page on the documentation site (`spektacular.dev`) or the tool's current behaviour (command help, schema output, or source). Nothing is asserted from memory or from the stale prior README. Where the two authorities disagree — notably the search-result example, which the site documents in an older form — the tool's current behaviour wins, because the README must match what actually ships and fixing the site is out of scope.

**Triage each existing section into revise / rewrite / delete.** Accurate prose (testing, building/contributing) is revised in place to preserve good content; stale-but-required sections (knowledge, supported agents, install) are rewritten from the authorities; and sections describing things that never shipped or no longer exist (the terminal UI, complexity scoring, model-routing-by-complexity) are deleted outright rather than softened or corrected. The supported-agent list is reduced to exactly the registered agents.

**Front-door-and-link pattern.** Each in-depth topic is handled at two altitudes: the README gives a self-contained technical orientation, then links to the single canonical documentation page for the exhaustive detail. The supported-agents and knowledge-base sections go one level deeper than a one-line mention (how the capability works, and how to extend it) before linking out; install, getting-started, full configuration, and the command reference stay brief and defer. Links use absolute `https://spektacular.dev/...` URLs and the canonical `github.com/jumppad-labs/spektacular` repository path throughout; no `nicholasjackson` reference survives.

**Code-structure UX.** A newcomer reading the file top to bottom should be able to state what Spektacular is, which agents it supports, and the spec → plan → implement workflow without leaving the README, while always having a clear link to follow for depth. A contributor cross-checking the README against the docs site and `--help` output should find no contradiction and no reference to a feature, command, or agent that does not exist.

## Dependencies

This work adds no code and therefore no runtime or library dependencies. Its dependencies are the authoritative sources that must be read to verify each claim, plus the existing behaviours the README describes. None of them need to change before this plan starts — in fact the spec requires the README be made correct *without* changing the documentation site or the tool's behaviour/code.

- **Documentation site (`spektacular.dev`, source in the `spektacular-website` repo)** — Authoritative for capability descriptions, the workflow narrative, install methods, the configuration reference, and the current published version. The README links to its pages. No change required (and changing it is a non-goal); the README must agree with it as it stands today.
- **Tool command surface (`cmd/` — spec, plan, implement, knowledge, init, skill, file subcommands)** — Authoritative for which commands, subcommands, and flags exist. Used to reconcile every command the README names. No change required.
- **Knowledge subsystem (categories, two tiers, search/dedup, `Hit` shape)** — Authoritative for the knowledge-base section and the search-result example. No change required; the README depicts it as-is, including where that diverges from the site's older example.
- **Agent registry (claude, bob, codex) and init behaviour** — Authoritative for the supported-agents list and the description of what initialising a project creates on disk (including Claude's agent-rules import). No change required.
- **Config schema and Makefile (build, cross-compile, Harbor e2e)** — Authoritative for the configuration, testing, and building sections. No change required.

Planning dependencies: none must land first. The knowledge categories/tiers/dedup feature (spec 000028) is already shipped and is the behaviour this README is being brought into line with, not a prerequisite to complete. There is no dependency on the prior README plan (000003) beyond using it as a record of what stale content to remove.

## Testing Approach

The deliverable is prose, not code, so no unit, integration, or end-to-end tests are added — there is no behaviour to assert programmatically, and the existing Go test suite and Harbor e2e harness are unaffected by a README edit. Verification is by review: a reviewer compares the refreshed README against the two authorities (the `spektacular.dev` documentation site and the tool's current command help / behaviour) and confirms that no statement contradicts either and that no named command, agent, or capability is absent from the tool. This matches the spec, which states its success metrics are verified manually.

The load-bearing checks, in plain language: the README names exactly the agents the tool registers (claude, bob, codex) and no others; it contains no terminal-UI, complexity-scoring, or model-routing claims; every repository/module reference uses the canonical `github.com/jumppad-labs/spektacular` path; it states no version older than the current release; any shown example (notably the search-result shape) matches current behaviour; and each in-depth topic links to its documentation page rather than reproducing it.

Deliberate gap: no automated link-checker or string-grep test is added as a project test fixture. Such mechanical aids (e.g. grepping for residual `nicholasjackson`, `Aider`, `Cursor`, `TUI`, or `complexity` strings, and confirming every command shown appears in `--help`) are useful one-off checks during implementation but are not part of the review contract and are not committed as tests.

Success-metric verification mapping (all three are manual, per the spec, which states they "will be verified manually by review against the documentation site and the tool's current behaviour"):

- **Zero contradictions and zero references to features/commands/agents that do not exist, when reviewed against the docs site and tool help** — Manual — captured in the implementation test plan.
- **Every command, agent, and capability named in the README maps to one that exists in the current tool** — Manual — captured in the implementation test plan.
- **A newcomer can read the README alone and correctly describe what Spektacular is, which coding agents it supports, and the spec → plan → implement workflow, without consulting other sources** — Manual — captured in the implementation test plan.

## Milestones & Phases

### Milestone 1: Remove false claims and fix the project's identity

**What changes**: A reader can no longer come away with a wrong picture from the parts of the README that actively misrepresent the tool. The terminal-UI section, the complexity-scoring claim, and the model-routing-by-complexity claim are removed; the supported-agent list is corrected to exactly claude, bob, and codex (Aider and Cursor are gone); every clone/release/module reference uses the canonical `github.com/jumppad-labs/spektacular` path; the stale `v0.1.0` is replaced by an "early development" status that links to the releases page; and the Python-era install instructions are corrected to reflect a Go binary. This milestone is the accuracy floor: even before the deeper sections are rewritten, nothing in the README contradicts the tool or names something that does not exist.

*Validation point:* A grep of the README finds no `nicholasjackson`, `Aider`, `Cursor`, `TUI`, `complexity`, or `v0.1.0`; the Spektacular install instructions no longer mention `uv`/`pip` (the `uv` reference in the testing section stays — it installs the Harbor test harness); the agent list reads exactly claude/bob/codex; and a reviewer confirms no removed section described a real feature.

#### - [x] Phase 1.1: Remove invented and never-shipped content

Delete the parts of the README that describe things the tool does not do: the standalone terminal-UI section and its screenshot, the complexity-scoring and model-routing-by-complexity claims in the introduction and the how-it-works diagram, and the speculative Roadmap section. These are removed outright rather than corrected, because the features never shipped or no longer exist.

*Technical detail:* [context.md#phase-11](./context.md#phase-11-remove-invented-and-never-shipped-content)

**Acceptance criteria**:
- [x] The README has no terminal-UI / "TUI" section and no reference to an interactive terminal UI or its screenshot.
- [x] The README makes no claim that work is scored for complexity or routed to different models by complexity.
- [x] The speculative roadmap content is gone (planned backends, if mentioned at all, appear only as a brief note in the extensibility material).
- [x] No content describing a real, shipped feature was lost in the removal.

#### - [x] Phase 1.2: Correct identity, version, and install basics

Fix the factual identity errors: the supported-agent list becomes exactly claude, bob, and codex; every repository, clone, and release reference uses the canonical `github.com/jumppad-labs/spektacular` path; the stale pinned version is replaced by an "early development" status that links to the releases page; and the install instructions reflect that Spektacular is a Go binary (Homebrew / `go install` / releases) rather than the previous build-only / Python-era steps.

*Technical detail:* [context.md#phase-12](./context.md#phase-12-correct-identity-version-and-install-basics)

**Acceptance criteria**:
- [x] The README names exactly claude, bob, and codex as supported agents, and never mentions Aider or Cursor.
- [x] No `nicholasjackson` reference remains; every repository/module path is `github.com/jumppad-labs/spektacular`.
- [x] The README states no version older than the current release; version is conveyed as an "early development" status with a link to releases rather than a hardcoded older number.
- [x] The install instructions describe obtaining the Go binary and do not reference `uv`/`pip` for installing Spektacular itself.

### Milestone 2: Rewrite the capability sections to current behaviour

**What changes**: The README gains an accurate technical front door to everything Spektacular does today. The introduction conveys the core competencies (self-contained binary plus installed agent skills, the state-machine-driven workflow with per-step prompts, agent-agnostic multi-agent support, and the knowledge base); the how-it-works section presents spec → plan → implement with implementation as a shipped, acceptance-criteria-validated capability; the supported-agents section explains what initialising a project sets up per agent (including Claude's agent-rules import) and how to add an agent; the knowledge-base section describes the six-category model, the two retrieval tiers, category-labelled search, and consolidated/de-duplicated lookups, showing a search-result example that matches current behaviour; and configuration and project-structure sections describe what is configurable and what init creates on disk. Each in-depth topic links to its documentation page rather than reproducing it.

*Validation point:* Every required section is present and each claim is traceable to the docs site or the tool's behaviour; the agents and knowledge sections go beyond a one-line mention and link out; the search-result example matches the current seven-field shape; and an install/getting-started orientation defers to the docs rather than reproducing them.

#### - [x] Phase 2.1: Introduction and how it works

Rewrite the introduction to convey Spektacular's core competencies for a technical reader: a self-contained binary plus the agent skills it installs, a state-machine-driven workflow that drives the coding agent step by step through per-step prompts, agent-agnostic multi-agent support, and a project knowledge base. Replace the analyse/score diagram with the real spec → plan → implement flow, presenting implementation as a shipped capability validated against acceptance criteria, and link to the how-it-works documentation for the full pipeline.

*Technical detail:* [context.md#phase-21](./context.md#phase-21-introduction-and-how-it-works)

**Acceptance criteria**:
- [x] The introduction names the self-contained binary plus installed skills, the state-machine workflow with per-step prompts, agent-agnostic multi-agent support, and the knowledge base.
- [x] The workflow is presented as spec → plan → implement, with implementation described as shipped and validated against acceptance criteria.
- [x] The how-it-works section links to the corresponding documentation page rather than reproducing the full pipeline.

#### - [x] Phase 2.2: Install and getting started

Replace the long Quick Start with a single combined install-and-getting-started section that orients the reader — it is a Go binary; the minimal path is init an agent, create a spec, generate a plan, implement it — and then defers to the install and getting-started documentation for full instructions rather than reproducing them.

*Technical detail:* [context.md#phase-22](./context.md#phase-22-install-and-getting-started)

**Acceptance criteria**:
- [x] A reader sees how to install the binary and the minimal first-run path, briefly.
- [x] The section links to the installation and getting-started documentation rather than reproducing those instructions in full.

#### - [x] Phase 2.3: Supported agents and extensibility

Add a supported-agents section that goes a level deeper: the three agents supported today, what initialising a project sets up for each (including that, for Claude, initialisation ensures the project's agent rules are loaded), how the agent integration works, and how to add or extend an agent. Fold the storage-backend extensibility material into a concise note here and in the knowledge section — storage backends and agents are pluggable behind defined interfaces — pointing to the plugins and extending documentation rather than reproducing the interface signatures.

*Technical detail:* [context.md#phase-23](./context.md#phase-23-supported-agents-and-extensibility)

**Acceptance criteria**:
- [x] The agents section names exactly claude, bob, and codex and describes what init sets up per agent, including the Claude agent-rules import.
- [x] The section explains how the agent integration works and how to add or extend an agent, and links to the plugins/extending documentation.
- [x] The README states that storage backends and agents are pluggable behind defined interfaces, with a pointer to the extending documentation, without reproducing the full interface signatures.

#### - [x] Phase 2.4: Knowledge base and project structure

Rewrite the knowledge-base section for a technical reader: the six-category model, the two retrieval tiers (always-applied conventions and glossary vs looked-up architecture, gotchas, learnings, decisions), category-labelled search, and consolidated/de-duplicated lookups across scopes — then link to the knowledge-base documentation for full detail. Update the knowledge command list and the search-result example so they match current behaviour, and correct the project-structure tree to show what init actually creates on disk (all six knowledge categories).

*Technical detail:* [context.md#phase-24](./context.md#phase-24-knowledge-base-and-project-structure)

**Acceptance criteria**:
- [x] The knowledge-base section names the six categories, the two retrieval tiers, category-labelled search, and consolidated/de-duplicated lookups, and links to the documentation for full detail.
- [x] Any knowledge command list and any search-result example shown match the tool's current behaviour (including the current search-result fields).
- [x] The project-structure section reflects what init creates on disk, including all six knowledge categories.

#### - [x] Phase 2.5: Configuration

Refresh the configuration section so it describes what is actually configurable — the agent and the spec, plan, and knowledge stores and related settings — at front-door depth, with a pointer to the full configuration reference rather than reproducing every key.

*Technical detail:* [context.md#phase-25](./context.md#phase-25-configuration)

**Acceptance criteria**:
- [x] The configuration section describes the configurable agent and the spec/plan/knowledge stores and related settings.
- [x] It links to the full configuration reference rather than reproducing the entire key set, and any example shown matches current behaviour.

### Milestone 3: Revise the retained sections and verify the whole document

**What changes**: The sections that were already accurate — testing (the unit suite and the end-to-end harness and its modes) and building-from-source/contributing/license — are revised in place so they remain correct and consistent with the refreshed surrounding content, and the document is read end to end as a coherent front door. This milestone closes the loop: a newcomer can read the README alone and correctly describe what Spektacular is, which agents it supports, and the spec → plan → implement workflow, and a contributor cross-checking it against the docs site and `--help` finds no contradiction.

*Validation point:* The full manual review passes — zero contradictions against the docs site and tool help, zero references to non-existent features/commands/agents, and the newcomer-comprehension check — i.e. all three success metrics from the spec are satisfied.

#### - [x] Phase 3.1: Testing section

Revise the testing section in place so it accurately describes how the project is tested: the unit suite and the end-to-end Harbor harness and its workflow suites (spec and plan) and modes. Correct the available-test-tasks listing to match the suites that actually exist.

*Technical detail:* [context.md#phase-31](./context.md#phase-31-testing-section)

**Acceptance criteria**:
- [x] The testing section describes the unit suite and the end-to-end harness and its modes, and the listed test suites match those that actually exist.
- [x] Every command or target shown in the testing section exists in the project.

#### - [x] Phase 3.2: Building, contributing, and license

Revise the building-from-source section so the build and developer targets shown match the actual Makefile (correct the binary output path and the install target name), and confirm the contributing steps and the license statement are accurate.

*Technical detail:* [context.md#phase-32](./context.md#phase-32-building-contributing-and-license)

**Acceptance criteria**:
- [x] Every build/developer target shown matches a real Makefile target, and the stated binary output path is correct.
- [x] The contributing steps are accurate and a correct license statement is present.

#### - [x] Phase 3.3: Whole-document verification

Read the assembled README end to end as a single front door and cross-check every claim against the documentation site and the tool's command help. Resolve any remaining contradiction, dangling internal link, or reference to something that does not exist. This phase carries the spec's three manual success metrics.

*Technical detail:* [context.md#phase-33](./context.md#phase-33-whole-document-verification)

**Acceptance criteria**:
- [x] A reviewer comparing the README against the documentation site and the tool's command help finds zero contradictions. *(Manual — captured in the implementation test plan.)*
- [x] Every command, agent, and capability named in the README maps to one that exists in the current tool. *(Manual — captured in the implementation test plan.)*
- [x] A newcomer can read the README alone and correctly describe what Spektacular is, which agents it supports, and the spec → plan → implement workflow. *(Manual — captured in the implementation test plan.)*
- [x] All internal links and documentation links resolve, and every required section is present.

## Open Questions

None. After a genuine pass, there are no uncertainties that can only be resolved once implementation begins. The structure, the section list, the link targets, the version-handling approach, and every factual correction were decided during planning against the documentation site and the tool's code, with citations recorded in `research.md`.

Two items were considered and deliberately resolved now rather than parked here:

- **Docs-site public URLs** — the base URL `https://spektacular.dev` and each page route were verified from `../spektacular-website/astro.config.mjs` and the page sources, so link targets are known, not open. (Recorded as an assumption in `research.md#open-assumptions` only as a safety net: if a link 404s at finalize time, the implementer should confirm the correct route with the user.)
- **Current release version** — handled by design (status note + releases link) precisely so it cannot become an open question; no number needs to be looked up at implement time.

## Out of Scope

- **Changing the documentation site or the tool's behaviour/code.** The README is brought into alignment with the site and the tool as they stand today; nothing in the site, the commands, or the code is modified by this work. (Spec constraint and non-goal.)
- **Fixing staleness in the documentation site itself.** Where a site page is also out of date — notably the extending page's older search-result example — it is left as-is; the README matches current tool behaviour instead of the stale site example. Correcting the site is a separate effort. (Spec non-goal.)
- **Translation or internationalisation of the README.** English only; no localized variants. (Spec non-goal.)
- **Visual or branding redesign.** No new logos, badge sets, screenshots, or layout/branding rework beyond what accuracy requires; the now-unused TUI screenshot reference is removed, but no new visuals are introduced and the orphaned image asset is not otherwise reworked. (Spec non-goal.)
- **Pruning orphaned repository assets.** Deleting the now-unreferenced `images/tui.png` file (and any other orphaned asset) is left out; only the README's reference to it is removed. A later housekeeping change can remove the file if desired.
- **An exhaustive command/configuration reference in the README.** The README intentionally stays a concise front door; the full command surface and full configuration key reference live on the documentation site and are linked, not reproduced. (Chosen design.)

## Changelog

### 2026-06-29 — Phase 1.1: Remove invented and never-shipped content

**What was done**: Removed the parts of `README.md` that describe behaviour the tool does not have. Deleted the standalone TUI section and its `images/tui.png` screenshot reference, stripped the complexity-scoring and model-routing-by-complexity claims from the introduction and the How It Works section (including the complexity ascii diagram and the "interactive TUI" mention), and deleted the speculative Roadmap section along with its stale `initial-idea.md` link.

**Deviations**: None.

**Files changed**:
- `README.md`

**Discoveries**: This plan is documentation-only — no Go code is touched, so the `test` step writes no `*_test.go` files and `verify` is grep + manual review per the plan's Testing Approach (which deliberately does not commit grep checks as fixtures). Phase 1.1 intentionally left `Aider`/`Cursor`, the `nicholasjackson` URLs, and the `v0.1.0` status in place — those are corrected in Phase 1.2. The orphaned `images/tui.png` asset is left on disk (pruning assets is out of scope); only the README reference was removed.

### 2026-06-29 — Phase 1.2: Correct identity, version, and install basics

**What was done**: Fixed the README's factual identity errors. The inline supported-agent list now reads claude, bob, and codex (Aider/Cursor removed); the stale `nicholasjackson` clone and releases URLs are now the canonical `github.com/jumppad-labs/spektacular`; the pinned `v0.1.0` status became an "early development" status linking to the releases page (no hardcoded number); and the Install block now describes obtaining the Go binary via Homebrew, `go install`, and the releases page, with a link to the install docs.

**Deviations**: None. The full Quick Start / getting-started rework remains deferred to Phase 2.2 as planned; this phase only corrected the install basics.

**Files changed**:
- `README.md`

**Discoveries**: Install methods confirmed against the docs site (`../spektacular-website/src/pages/install.mdx`): brew tap `jumppad-labs/homebrew-repo/spektacular`, `go install github.com/jumppad-labs/spektacular@latest` (Go 1.21+), apt, and releases. The `uv` references survive only in the Testing section (they install the Harbor test harness) — correct and intentional; no `pip` remains anywhere.

### 2026-06-29 — Phase 2.1: Introduction and how it works

**What was done**: Rewrote the README's tagline, the "What is Spektacular?" introduction, and the "How It Works" section. The intro now presents the four core competencies (self-contained binary plus installed agent skills, the state-machine-driven workflow with per-step `new`/`goto`/`steps` prompts, agent-agnostic multi-agent support, and the project knowledge base). How It Works presents the real spec → plan → implement flow with implementation described as a shipped capability that validates against acceptance criteria, and links to the how-it-works documentation instead of reproducing the full pipeline.

**Deviations**: None. Dropped the prior numbered "planning agent" framing in favour of the three-stage spec → plan → implement narrative, as the plan directed.

**Files changed**:
- `README.md`

**Discoveries**: Language sourced from `../spektacular-website/src/pages/index.mdx` (core-competency framing, "resumable workflows", multi-agent) and `how-it-works.mdx` (5-step quick start, spec→plan→implement stages). Confirmed the `new`/`goto`/`steps`/`status` subcommands exist on all three workflow commands (`cmd/{spec,plan,implement}.go`) before naming the per-step prompt model.

### 2026-06-29 — Phase 2.2: Install and getting started

**What was done**: Collapsed the long "Quick Start" and standalone "Spec Format" sections into a single concise "Install & getting started" section. It states Spektacular is a Go binary, shows the Homebrew / `go install` / releases install methods, notes a supported coding agent CLI is required, and lays out the minimal init → spec → plan → implement path. The full spec-format example was replaced by a one-line description plus links to the getting-started tutorial and how-it-works docs.

**Deviations**: None. The minimal path now includes the `implement` step (the prior Quick Start stopped at `plan`), reflecting that implementation is shipped.

**Files changed**:
- `README.md`

**Discoveries**: Getting-started tutorial route is `https://spektacular.dev/tutorials/getting-started` (source `../spektacular-website/src/content/tutorials/getting-started.mdx`). Dropped the verbose `id_method`/`external` id prose and the `EXT-123` example — that detail belongs to the configuration section and the docs site, keeping the front door concise.

### 2026-06-29 — Phase 2.3: Supported agents and extensibility

**What was done**: Added a "Supported agents" section naming claude, bob, and codex and describing what `init` installs for each — claude → `.claude/skills/` plus ensuring `CLAUDE.md` imports `@AGENTS.md`; bob → `.bob/skills/` + `.bob/commands/`; codex → `.agents/skills/`. It explains the integration (each agent implements the narrow `Agent` interface's `Name()` and `Install()` and registers via `init()`), how to add one, and folds in a concise pluggability note: both the `Agent` and `Store` interfaces are swappable, only the `file` store ships today, with links to the extending and plugins docs. Removed the standalone "Extending Storage" section, including its full `Store` interface, `DirEntry`, and `Hit` code blocks.

**Deviations**: None. The extensibility material was folded into the agents section per the plan; the corrected 7-field `Hit` search-result example is intentionally deferred to Phase 2.4 (knowledge section).

**Files changed**:
- `README.md`

**Discoveries**: Verified install layouts directly from source — `internal/agent/claude.go` (`.claude/skills`, `ensureClaudeImportsAGENTS`), `bob.go` (`.bob/skills` + `.bob/commands`), `codex.go` (`.agents/skills`), and the `Agent` interface (`Name`/`Install`) plus `register()`/`init()` registry in `agent.go`. No drift.

### 2026-06-29 — Phase 2.4: Knowledge base and project structure

**What was done**: Rewrote the Knowledge section to the canonical model — the six categories grouped into two retrieval tiers (always-applied `conventions`/`glossary`; looked-up `architecture`/`gotchas`/`learnings`/`decisions`), scope- and category-tagged search, and consolidated/de-duplicated lookups keyed on a SHA-256 byte-identity checksum. Replaced the old five-field search-result example with the current seven-field `Hit` (adds `category` and `checksum`), trimmed the exhaustive CLI table to a representative bullet list that now includes `knowledge categories` and `knowledge always-applied`, and pointed the "full model" link at `https://spektacular.dev/knowledge-base/` instead of the in-repo `docs/knowledge-base.md`. Corrected the Project Structure tree to show all six knowledge categories with their tiers and noted that each category directory is scaffolded with a `README.md`.

**Deviations**: None. Dropped the standalone "How planning uses it" subsection to keep the section a concise front door; its essence (conventions selected into the plan) is covered by the "planning-time input" framing and the linked docs.

**Files changed**:
- `README.md`

**Discoveries**: Tiers confirmed from `internal/knowledge/category.go` (`TierAlwaysApplied` = conventions + glossary; `TierLookedUp` = the other four) and the model language from `../spektacular-website/src/pages/knowledge-base.mdx`. The 7-field `Hit` and the SHA-256 byte-identity de-dup key were taken from `internal/store/store.go:24-32` and `internal/store/search.go` — intentionally diverging from the docs site's stale four-field example, as the plan requires. All eight `knowledge` subcommands referenced exist in `cmd/knowledge.go`.

### 2026-06-29 — Phase 2.5: Configuration

**What was done**: Refreshed the Configuration section to front-door depth. The intro now states `config.yaml` controls which coding agent Spektacular drives plus the provider-based spec/plan/knowledge stores. The YAML example was kept (it matches `config.NewDefault()`), and the verbose `id_method` bullet list and the name-normalisation paragraph were collapsed into a single sentence that names the six top-level sections (`command`, `agent`, `debug`, `spec`, `plan`, `knowledge`) and summarises id-method and directory resolution, with a pointer to the configuration documentation for the full reference.

**Deviations**: None.

**Files changed**:
- `README.md`

**Discoveries**: The six top-level config sections were verified against `internal/config/config.go:88-133` (`Config` struct + `NewDefault`); the example's defaults (`command: spektacular`, `spec.id_method: timestamp`, `provider: file`, default directories/location) all match `NewDefault()`. `${VAR}` expansion exists (`config.go` `FromYAMLFile`) and is now mentioned only as a pointer to the docs rather than reproduced.

### 2026-06-29 — Phase 3.1: Testing section

**What was done**: Revised the Testing section to cover both test layers. Reframed the intro around a fast Go unit suite plus the end-to-end Harbor harness, added a "Unit tests" subsection (`go test ./...` / `make test`), added a block showing the Makefile wrappers (`make harbor-test-spec`, `make harbor-test-spec-codex`, `make harbor-test-plan`) alongside the raw `harbor run` invocation, and expanded the "Available test tasks" table to list both the spec-workflow and plan-workflow suites.

**Deviations**: None. Changed the spec-workflow table description from "through all 10 steps" to "end to end" to avoid pinning a step count that could drift.

**Files changed**:
- `README.md`

**Discoveries**: Confirmed both Harbor task directories exist (`tests/harbor/spec-workflow`, `tests/harbor/plan-workflow`) and every Makefile target named in the section is real (`test`, `harbor-test-spec`, `harbor-test-spec-codex`, `harbor-test-plan` — `Makefile:12,43-78`). The `uv`/Harbor install and the claude-code/oracle/codex Harbor agents were kept as-is (legitimate).

### 2026-06-29 — Phase 3.2: Building, contributing, and license

**What was done**: Corrected the "Building from Source" section. The binary output path is now `./bin/spektacular` (was `./spektacular`), the `make install` row was replaced with the real `make install-local` target (copies to `/usr/local/bin`), and a `make clean` row was added — so the table lists exactly the real targets (build, test, lint, clean, install-local, cross). The Contributing steps now cite `make test` and `make lint`, and the Apache 2.0 license statement with its `LICENSE` link was confirmed accurate.

**Deviations**: None. The plan anticipated a stray `go build -o spektacular .` snippet to fix, but the current section already used `make build`, so no such snippet existed.

**Files changed**:
- `README.md`

**Discoveries**: Verified against `Makefile` — `build` emits `./bin/$(BINARY)` where `BINARY := spektacular` (`Makefile:1,9-10`), `install-local` copies `./bin/spektacular` to `/usr/local/bin` (`Makefile:21-22`), and `clean`/`cross` exist (`Makefile:18,36`). `LICENSE` is present and is the Apache License 2.0.

### 2026-06-29 — Phase 3.3: Whole-document verification

**What was done**: Read the assembled `README.md` end to end and cross-checked every claim against the documentation site and the tool's command help. All checks passed: no forbidden terms (Aider/Cursor/Copilot/Gemini/TUI/complexity/model-routing/nicholasjackson/v0.1.0/roadmap); the only agents named are claude/bob/codex; every CLI command shown (`init`, `spec/plan/implement new`, the eight `knowledge` subcommands) exists; all seven `https://spektacular.dev/...` links map to real page sources; the `LICENSE` link and both internal anchors (`#configuration`, `#knowledge`) resolve; and all eleven required sections are present. Fixed one readability regression introduced in Phase 3.1 by demoting the Harbor sub-subsections (Prerequisites, Install Harbor, Run the oracle, Run with a real agent, Test results, Available test tasks) to `####` so they nest under the new `### End-to-end (Harbor)` heading.

**Deviations**: None. No content contradictions or dangling links were found, so no claim-level fixes were needed beyond the heading-hierarchy cleanup.

**Files changed**:
- `README.md`

**Discoveries**: The spec's three success metrics are all manual and are satisfied by this review: (1) zero contradictions against the docs site and tool help; (2) every named command/agent/capability exists in the current tool; (3) a newcomer can read the README alone and describe what Spektacular is, the supported agents (claude/bob/codex), and the spec → plan → implement workflow. Docs-site routes were confirmed against `../spektacular-website/src/pages/*.mdx` and `src/content/tutorials/getting-started.mdx`.

