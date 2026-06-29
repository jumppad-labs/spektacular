# Working context — IMPLEMENT for 000029_readme-refresh

(Now in the IMPLEMENT workflow. Executing the approved plan: revise the repo
root `README.md` so it accurately reflects current Spektacular behaviour.
Documentation-only change — no Go code is edited.)

## Goal
Rewrite/revise root `README.md` into a concise, accurate front door + links to
the docs site for depth. Authoritative sources: docs site at
`../spektacular-website` + the codebase / tool help. README must be correct
WITHOUT changing site or code.

## read_plan validation result (this session)
- Structure OK: all 10 required plan sections present; phases 1.1–3.3 each have
  a resolving `context.md#phase-NM` link.
- Drift check: NO drift. Every file, symbol, category, Makefile target,
  command, and docs page named in plan/context exists.
- Changelog: ABSENT in plan.md → first-phase invocation. Analyze starts at
  Phase 1.1. update_changelog will create the `## Changelog` section on first use.

## Phase order (single file, SEQUENTIAL — never parallel; concurrent edits to one file conflict)
1.1 remove invented/never-shipped · 1.2 fix identity/version/install ·
2.1 intro+how-it-works · 2.2 install+getting-started · 2.3 agents+extensibility ·
2.4 knowledge+project-structure · 2.5 configuration · 3.1 testing ·
3.2 building/contributing/license · 3.3 whole-doc verification.

## Load-bearing decisions (user-confirmed during plan)
- STRUCTURE = section-driven rewrite; revise accurate prose (testing/building)
  in place, rewrite stale, delete invented (TUI/complexity/model-routing).
- VERSIONING = "early development" status + releases link (NO hardcoded number;
  avoids re-drift). README must state no version older than current release v0.11.1.
- Search-result example sourced from CODE (7-field Hit), NOT the site's stale
  4-field extending.mdx example. Intentional divergence; site fix is a non-goal.
- Drop standalone Roadmap; fold planned backends into extensibility note.
- Conventions: NONE apply (project conventions are placeholder-only).

## Verified facts (citable; full detail in research.md)
- Docs base URL = https://spektacular.dev. Pages: /how-it-works/ /knowledge-base/
  /plugins/ /extending/ /configuration/ /install/ /tutorials/getting-started.
- Module github.com/jumppad-labs/spektacular (go.mod:1); stale nicholasjackson at README.md:47,55.
- Agents = exactly claude/bob/codex (internal/agent/*). README wrongly lists Aider/Cursor.
- Knowledge: 6 cats / 2 tiers (always-applied: conventions,glossary; looked-up:
  architecture,gotchas,learnings,decisions) — category.go. Dedup = SHA-256 byte-identity.
  Hit struct = 7 fields (scope,path,title,excerpts,score,category,checksum) store.go:24-32.
- Knowledge has 8 subcommands (README lists 6; missing categories + always-applied).
- Init scaffolds .spektacular/{specs,plans,knowledge/<cat>/README.md,config.yaml,.gitignore};
  claude init makes CLAUDE.md import @AGENTS.md.
- Config 6 sections (config.go:88-133). Testing: go test ./... + Harbor e2e
  (Makefile:43-73, suites: harbor-test-spec{,-claude,-codex}, harbor-test-plan).
- Build: make build → ./bin/spektacular (NOT ./spektacular); install target is
  install-local (NOT install). Real targets: build,test,lint,clean,install-local,cross.
- uv reference in Testing section is LEGIT (installs Harbor) — keep; only remove
  uv/pip from the Spektacular-install context.
- images/tui.png & docs/knowledge-base.md EXIST — leave both; only remove/redirect README refs.

## Progress — ALL PHASES DONE (M1+M2+M3). 3.3 whole-doc verification PASSED.
- Phase 3.3: full read-through + cross-checks all green — no forbidden terms;
  agents only claude/bob/codex; all 7 spektacular.dev links map to real page
  sources; LICENSE + #configuration/#knowledge anchors resolve; all commands exist;
  11 required sections present. Fixed Testing heading hierarchy (Harbor subsections
  demoted to #### under "### End-to-end (Harbor)", added #### Prerequisites).
  Three spec success metrics (all manual) satisfied.
- Repo CHANGELOG.md: prepended user-facing "## 000029_readme-refresh" section.
- test_plan: wrote 000029_readme-refresh/test-plan.md with 3 grounded manual
  procedures (contradiction sweep; command/agent/capability existence; newcomer
  comprehension). All three spec metrics are manual (prose deliverable).
- WORKFLOW COMPLETE: all 10 phases implemented + verified, plan checkboxes +
  inline changelog done, repo CHANGELOG.md updated, test-plan written.
- Phase 3.2: fixed Building table — output ./bin/spektacular (was ./spektacular);
  make install → install-local (copies to /usr/local/bin); added clean; targets
  build/test/lint/clean/install-local/cross all verified. Contributing now cites
  make test + make lint. License Apache 2.0 + LICENSE link confirmed (LICENSE exists).
- Phase 3.1: Testing reframed as two layers — added Unit tests (go test ./... /
  make test) + "End-to-end (Harbor)" framing; added make wrappers block
  (harbor-test-spec / -codex / harbor-test-plan); Available test tasks table now
  lists spec-workflow AND plan-workflow. Replaced "all 10 steps" with "end to end"
  (avoid pinning a count). All targets/paths verified to exist.
- Phase 2.5: Configuration refreshed — intro names configurable agent; kept YAML
  example (matches config.NewDefault); trimmed verbose id_method bullets +
  normalization paragraph into one sentence naming the 6 sections; added link to
  spektacular.dev/configuration/.
- Phase 2.4: rewrote Knowledge (6 cats/2 tiers; always-applied=conventions+glossary;
  looked-up=architecture/gotchas/learnings/decisions; scope+category-tagged search;
  SHA-256 byte-identity dedup; 7-field Hit example; trimmed CLI to bullet list incl.
  categories+always-applied; link spektacular.dev/knowledge-base/, dropped in-repo
  docs/knowledge-base.md link). Project tree now all 6 categories + per-cat README note.
- Phase 2.3: added "## Supported agents" (claude=.claude/skills+CLAUDE.md→@AGENTS.md;
  bob=.bob/skills+.bob/commands; codex=.agents/skills; Agent iface Name()+Install();
  register via init(); pluggability note for Agent+Store, file store only, links
  extending+plugins). Removed standalone "## Extending Storage" entirely (full Store
  interface block + DirEntry/Hit blocks gone). 7-field Hit example deferred to 2.4.
- Phase 2.2: collapsed Quick Start + Spec Format into "## Install & getting
  started" — install methods, init→spec→plan→implement minimal path (implement
  now included), links install + getting-started + how-it-works docs. Dropped
  verbose id_method/external-id prose + full example spec. Note getting-started
  URL = https://spektacular.dev/tutorials/getting-started.
- Phase 2.1: rewrote tagline (L3), "What is Spektacular?" (4 core competencies:
  binary+skills, state-machine new/goto/steps, agent-agnostic multi-agent, KB),
  and How It Works (spec→plan→implement numbered, implement shipped + validates
  ACs, links how-it-works docs). Confirmed new/goto/steps/status subcommands exist.
- read_plan validation passed (no drift).
- Phase 1.1 analyze: confirmed README line numbers match plan refs — TUI 28-34,
  intro complexity/model-routing 9 & 11, how-it-works diagram 15-19 + 22 + TUI
  mention 26, Roadmap 259-265. Low complexity, single file, no sub-agents needed.
- Phase 1.1 IMPLEMENTED: removed TUI section + screenshot ref, complexity/model-
  routing claims (intro + how-it-works), the complexity ascii diagram, the
  "interactive TUI" mention, and the Roadmap section (incl. initial-idea.md link).
  grep confirms zero residual tui/complexity/roadmap/bubble-tea. Intro still lists
  Aider/Cursor and nicholasjackson/v0.1.0 remain — those are Phase 1.2's job.
- Phase 1.1 test: NO automated tests — deliverable is prose. Plan's Testing
  Approach forbids committing grep checks as fixtures; all ACs are manual/review.
  This applies to EVERY phase of this plan (doc-only). Verification = grep + the
  whole-doc manual review captured in Phase 3.3 + the test_plan step.
- Phase 1.1 verify: all 4 ACs PASS (grep clean for tui/complexity/roadmap;
  real sections intact; structured-output mention preserved). No code changed.
- Phase 1.1 DONE: checkboxes flipped, changelog entry written (## Changelog
  section created after ## Out of Scope). 9 phases remain.
- LOOP MODE: driving straight through all phases without per-phase confirmation
  (user's standing feedback: skip confirmation on simple features; design is
  fully locked in the plan). Only stop for genuine drift/design decisions.
- NEXT: Phase 1.2 — agent list → claude/bob/codex; nicholasjackson →
  jumppad-labs; v0.1.0 → "early development" + releases link; install → Go binary
  (brew / go install / releases). Keep uv in Testing section only.
- Phase 1.2 analyze: verified install methods from install.mdx — brew install
  jumppad-labs/homebrew-repo/spektacular; go install github.com/jumppad-labs/
  spektacular@latest (Go 1.21+); apt; releases. Affected README lines (post-1.1):
  status L5, agent list L11, Install block L28-40 (clone/releases nicholasjackson).
  No drift.
- Phase 1.2 IMPLEMENTED + verified: status → "early development" + releases link;
  agent list → claude/bob/codex (no Aider/Cursor); Install → brew + go install +
  releases + install-docs link; nicholasjackson gone. uv only in Testing; no pip.
  All 4 ACs pass via grep.
