# Test Plan: 000029_readme-refresh

The deliverable is the root `README.md`; there is no runtime behaviour to assert
programmatically. All three of the spec's success metrics are verified manually
by review against the two authorities: the documentation site (`https://spektacular.dev`,
source in `../spektacular-website`) and the tool's current command help / source.

Run all three before merging the README change. The reviewer is anyone other than
the author; the point at which they run is pre-merge review of the branch.

---

## Metric 1 — Zero contradictions and no references to non-existent features/commands/agents

**What to measure:** Every factual claim in `README.md` agrees with the docs site
and the tool, and nothing names a feature, command, or agent that does not exist.

**How:**
1. Mechanical sweep — from the repo root, confirm none of the removed/forbidden
   terms remain:
   ```
   grep -niE 'aider|cursor|copilot|gemini|\bTUI\b|bubble ?tea|complexity|model.routing|nicholasjackson|v0\.1\.0|roadmap' README.md
   ```
   (The `uv` reference is legitimate and appears only in the Testing section — it
   installs the Harbor harness, not Spektacular.)
2. Confirm the supported agents read exactly claude/bob/codex and match the
   registry: compare against `go run . init --help` / `internal/agent/*.go`.
3. For every `spektacular <cmd>` shown, confirm it exists: `go run . <cmd> --help`
   (init, spec, plan, implement, knowledge and their subcommands).
4. For every `make` target shown, confirm it exists in the `Makefile`.
5. Read each rewritten section against its docs-site page
   (`../spektacular-website/src/pages/*.mdx`); confirm no statement contradicts it.
   Note: the search-result (`Hit`) example intentionally diverges from the site's
   stale four-field example and matches the current seven-field code shape — this
   is expected and correct.

**Expected result:** The grep returns no matches outside the legitimate Testing
`uv` line; every command, agent, and make target shown resolves; no section
contradicts its docs-site page.

**Who / when:** Reviewer, at pre-merge review.

---

## Metric 2 — Every command, agent, and capability named maps to one that exists

**What to measure:** 100% of the commands, agents, and capabilities the README
names exist in the current tool.

**How:** Enumerate every `spektacular`/`make`/`harbor` command and every named
agent in `README.md`, then resolve each one:
- commands → `go run . <cmd> --help` (and `cmd/*.go`);
- agents → `internal/agent/agent.go` registry (claude, bob, codex);
- make/harbor targets → `Makefile`;
- the `Hit` fields → `internal/store/store.go` (scope, path, title, excerpts,
  score, category, checksum);
- the six knowledge categories and two tiers → `internal/knowledge/category.go`.

**Expected result:** Every enumerated item resolves to a real command, agent,
target, field, or category. Count of unresolved items = 0.

**Who / when:** Reviewer, at pre-merge review.

---

## Metric 3 — A newcomer can describe Spektacular from the README alone

**What to measure:** After reading only `README.md` (no other sources), a reader
new to the project can correctly state (a) what Spektacular is, (b) which coding
agents it supports, and (c) the spec → plan → implement workflow.

**How:** Give the README to someone unfamiliar with the project. Without letting
them consult the docs site, the code, or the maintainers, ask them to answer in
their own words:
1. In one or two sentences, what is Spektacular?
2. Which coding agents does it support?
3. What are the stages of its workflow, in order?

**Expected result:** (1) names a spec-driven CLI that plans and implements a
markdown spec with an AI coding agent; (2) names claude, bob, and codex; (3) names
spec → plan → implement in order. All three answered correctly from the README alone.

**Who / when:** A reviewer (or any colleague) unfamiliar with Spektacular, at
pre-merge review.
