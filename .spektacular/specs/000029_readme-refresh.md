# Feature: 000029_readme-refresh

## Overview

Spektacular's README has drifted out of step with what the tool actually does — it describes an older feature set and omits capabilities that now exist, so a newcomer reading it can come away with a wrong or incomplete picture. This work brings the README back into alignment with the current product, so anyone evaluating or adopting Spektacular can read it and correctly understand what the tool is, what it can do today, and how to get started. The project's own documentation site and the tool's current behaviour are the authoritative sources the refreshed README must agree with.

## Requirements

**Cross-cutting accuracy requirements**

- Every capability, command, interface, and workflow the README describes must be verifiable against the documentation site or the tool's current behaviour.
- The README must not describe anything that no longer exists or never shipped — including the standalone terminal UI, complexity scoring, and routing work to different models by complexity.
- The README is written for a technical reader: it may describe the tool's mechanism — the self-contained binary, the state-machine-driven workflow and its per-step prompts — at a technical level, while still leaving exhaustive detail to the documentation site.
- The README must remain a concise front door: it must cover the breadth of current capabilities, but for in-depth topics — such as installation, getting started, configuration, and the full command reference — it must orient the reader briefly and link to the documentation site rather than reproducing that content.
- The README must state the project's correct identity: its current version, canonical repository and module path, and license.
- Any code or interface example the README shows must match the tool's current behaviour.
- The README must list only the coding agents Spektacular actually supports, and must not name unsupported ones.

**Sections the README must include** (each must accurately cover the stated content)

- An introduction / main description that gives a technical overview of Spektacular's core competencies: that it is a single self-contained binary plus a set of agent skills it installs; that each workflow is a state machine which drives the coding agent step by step through per-step prompts to control the flow; that it is agent-agnostic and supports multiple coding agents; and that it has a project knowledge base.
- A "how it works" section expanding the workflow: the spec → plan → implement flow, driven step by step by the state machine, with implementation a shipped capability that is validated against acceptance criteria.
- An installation and getting-started section that orients the reader and points to the documentation site for the full instructions rather than reproducing them; installation and getting started may be a single combined section.
- A supported-agents section that goes a level deeper: the agents supported today, what initialising a project sets up for each (including that, for Claude, initialisation ensures the project's agent rules are loaded), how the agent integration works, and how to add or extend an agent — linking to the documentation for full detail.
- A knowledge-base section that goes a level deeper: the category model and the two retrieval tiers, category-labelled search, and consolidated, de-duplicated lookups across scopes — explained for a technical reader and linking to the documentation for full detail.
- A configuration section describing what is configurable (the agent, the spec/plan/knowledge stores, and related settings), with a pointer to the full configuration reference.
- An extensibility note that storage backends and agents are pluggable behind defined interfaces, with a pointer to the extending/plugins documentation; this may be folded into the agents and knowledge sections rather than standing alone.
- A project-structure section describing what initialising a project creates on disk.
- A testing section describing how the project is tested (the end-to-end harness and its modes).
- A building-from-source and contributing section describing how to build the tool and how to contribute, and a license statement.

## Constraints

- The work must be delivered as the repository's single root `README.md`, written in GitHub-flavoured Markdown, since it is the repository's landing page.
- Every claim in the README must be sourced from the documentation site or the tool's current behaviour; the README must not introduce capabilities that are not backed by those sources.
- The README must be correct without requiring any change to the documentation site or to the tool's behaviour or code.

## Acceptance Criteria

- The README contains a section for each required topic: introduction/main description, how it works, installation and getting started, supported agents, knowledge base, configuration, project structure, testing, and building/contributing/license.
- The introduction conveys Spektacular's core competencies — the self-contained binary and the agent skills it installs, the state-machine-driven workflow with its per-step prompts, agent-agnostic multi-agent support, and the knowledge base.
- The supported-agents and knowledge-base sections each go beyond a one-line mention — describing how the capability works, and how to extend it where applicable — and link to the corresponding documentation page rather than reproducing it in full.
- The knowledge-base section names the category model and the two retrieval tiers and describes consolidated, de-duplicated lookups, then links to the documentation for full detail.
- The README contains no terminal-UI ("TUI") section, no complexity-scoring claim, and no claim that work is routed to different models by complexity.
- The README names the supported agents as exactly claude, bob, and codex, and contains no reference to unsupported agents (e.g. Aider or Cursor).
- The README presents the workflow as spec → plan → implement, and describes implementation as a shipped capability that is driven through the plan and validated against acceptance criteria.
- The README describes what initialising a project for an agent creates on disk and sets up, including that for Claude it ensures the project's agent rules are loaded.
- Every repository or module reference in the README uses the canonical path github.com/jumppad-labs/spektacular, and no reference to the previous path remains.
- The README states no version older than the current release, and any code or interface example it shows (such as a search result's fields) matches the tool's current behaviour.
- For each in-depth topic the README defers to the documentation site, it links to the corresponding page rather than reproducing it in full.
- A reviewer comparing the README against the documentation site and the tool's command help finds no statement in the README that contradicts either.

## Technical Approach

Treat the documentation site pages and the in-repo knowledge-base document as the primary sources, and reconcile the command surface against the tool's own help and schema output so the commands and behaviours described match what actually ships.

Where the current README is already accurate (for example the testing and building sections), prefer revising it in place rather than rewriting from scratch, so existing good content is preserved while the stale and missing parts are fixed.

Beyond this, no further technical direction has been decided; the detailed structure and wording are left for the plan workflow to propose.

## Success Metrics

- A contributor reviewing the refreshed README against the documentation site and the tool's command help finds zero contradictions and zero references to features, commands, or agents that do not exist.
- Every command, agent, and capability named in the README maps to one that exists in the current tool.
- A newcomer can read the README alone and correctly describe what Spektacular is, which coding agents it supports, and the spec → plan → implement workflow, without consulting other sources.

These will be verified manually by review against the documentation site and the tool's current behaviour.

## Non-Goals

- Improving or correcting the documentation site itself is out of scope, even where it is also stale (for example an outdated code example). This work refreshes only the repository README; site fixes are a separate effort.
- Translating or internationalising the README is out of scope.
- Redesigning the README's visuals or branding (logos, badge sets, new screenshots) is out of scope beyond what accuracy requires.
