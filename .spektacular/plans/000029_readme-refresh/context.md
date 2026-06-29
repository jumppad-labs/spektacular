# Context: 000029_readme-refresh

## Current State Analysis

The deliverable is the single root `README.md` (367 lines), produced by an earlier plan (`000003_readme`) and since drifted. It is partly stale and partly current:

- **Actively wrong / never shipped:** lists Aider & Cursor as agents (`README.md:11`); claims complexity scoring + model-routing-by-complexity (`README.md:9,11,16-19,22`); a standalone TUI section with a Bubble Tea screenshot (`README.md:28-34`, image `images/tui.png`); a speculative Roadmap (`README.md:259-265`); `nicholasjackson` clone/release URLs (`README.md:47,55`); pinned `v0.1.0` (`README.md:5`); Python-era build-only install (`README.md:43-55`).
- **Stale but on the right track:** the Knowledge section already describes a two-tier model (`README.md:136-176`) but links the in-repo `docs/knowledge-base.md` (we link the docs site instead), its CLI table omits `knowledge categories` and `knowledge always-applied` (`README.md:163-170`), the `Hit` example is the older 5-field form missing `category`+`checksum` (`README.md:202-208`), and the project-structure tree shows only 4 of 6 knowledge categories (`README.md:122-132`, missing `glossary` + `decisions`).
- **Already accurate (revise in place):** Testing (`README.md:267-332`) — though it omits the `plan-workflow` Harbor suite and the make wrappers; Building (`README.md:334-355`) — though `make install` should be `install-local` (`Makefile:21`) and the binary output is `./bin/spektacular` not `./spektacular` (`Makefile:9-10`); Contributing + License (`README.md:357-367`).

Authoritative sources confirmed: docs site `https://spektacular.dev` (source `../spektacular-website`, base URL `astro.config.mjs:7`); canonical module `github.com/jumppad-labs/spektacular` (`go.mod:1`); agents claude/bob/codex (`internal/agent/agent.go:28-56`); knowledge categories/tiers (`internal/knowledge/category.go:47-104`); `Hit` 7-field shape (`internal/store/store.go:24-32`); init scaffolding (`internal/project/init.go:15-117`); config schema (`internal/config/config.go:88-133`); current release v0.11.1 (`../spektacular-website/src/pages/index.mdx:22`).

## Per-Phase Technical Notes

> **Cross-cutting note for all phases:** every phase edits the single file `README.md` at the repo root. Phases must run **sequentially by one agent**, not in parallel, because concurrent edits to one file conflict. Per-phase complexity is Low. The authoritative sources are the docs site (`https://spektacular.dev`, source `../spektacular-website`) and the tool's code/help; never source a claim from the prior stale README. Link targets use absolute `https://spektacular.dev/...` URLs (the published site) and the canonical repo `github.com/jumppad-labs/spektacular`. Knowledge-base links go to `https://spektacular.dev/knowledge-base/` (the in-repo `docs/knowledge-base.md` exists but the spec calls for linking the documentation site).

### Phase 1.1: Remove invented and never-shipped content

File changes (`README.md`):
- Delete the entire **TUI** section `README.md:28-34` (heading, `![](./images/tui.png)` image ref at :30, Bubble Tea description, the 5-themes line). The orphaned asset `images/tui.png` may be left in place (pruning assets is out of scope); only the README reference is removed.
- Intro `README.md:9` — remove the `spec → analyse → plan → execute → validate` pipeline framing (rewritten properly in Phase 2.1; this phase just strips the complexity/analyse language).
- Intro `README.md:11` — remove "routes work to different models based on complexity … simple tasks use cheaper models" claim.
- How-It-Works diagram + steps `README.md:15-19` (the `[Analyse] → complexity score` ascii block) and `README.md:22` ("Spektacular scores the complexity of the task"); also the "interactive TUI" mention at `README.md:26`. (Full rewrite of this section in Phase 2.1 — here, remove the complexity/TUI claims so M1 leaves nothing false.)
- Delete the **Roadmap** section `README.md:259-265`, including the link to `.spektacular/knowledge/architecture/initial-idea.md` (stale, non-authoritative).

Complexity: Low. Token estimate: ~8k. Agent strategy: single agent, sequential (first edit to the file).

### Phase 1.2: Correct identity, version, and install basics

File changes (`README.md`):
- `README.md:11` — agent list `(Claude Code, Aider, Cursor)` → exactly claude, bob, codex (verified `internal/agent/agent.go:28-56`). The tagline `README.md:3` "Agent-agnostic CLI…" stays.
- `README.md:47` — `git clone https://github.com/nicholasjackson/spektacular.git` → `github.com/jumppad-labs/spektacular` (`go.mod:1`).
- `README.md:55` — releases URL `nicholasjackson` → `jumppad-labs`.
- `README.md:5` — `> **Status:** v0.1.0 — early development` → an "early development" status that links to `https://github.com/jumppad-labs/spektacular/releases` (no hardcoded older number; current release is v0.11.1 per `../spektacular-website/src/pages/index.mdx:22`, but prefer not to pin).
- Install `README.md:43-55` — replace build-from-source-only steps with Go-binary acquisition: `brew install jumppad-labs/homebrew-repo/spektacular`, `go install github.com/jumppad-labs/spektacular@latest` (Go 1.21+), and the releases page (per `../spektacular-website/src/pages/install.mdx`). Full getting-started rework happens in Phase 2.2; this phase ensures install basics are correct. Do NOT touch the `uv` reference in the Testing section (`README.md:275,280` — legitimately installs Harbor).

Complexity: Low. Token estimate: ~8k. Agent strategy: single agent, sequential.

### Phase 2.1: Introduction and how it works

File changes (`README.md`):
- Rewrite intro `README.md:3,7-11`: core competencies — self-contained Go binary + the agent skills it installs (`internal/agent/*` Install); state-machine-driven workflow driving the agent step by step via per-step prompts (the `spec/plan/implement` `new`/`goto`/`steps` model, `cmd/spec.go:70-97`, `cmd/plan.go`, `cmd/implement.go`); agent-agnostic multi-agent support; project knowledge base. Source language from `../spektacular-website/src/pages/index.mdx` and `how-it-works.mdx`.
- Rewrite How-It-Works `README.md:13-26`: spec → plan → implement, each step state-machine-driven; implement is shipped and validates against acceptance criteria (`cmd/implement.go:43-70`; `how-it-works.mdx` stages). Keep an accurate ascii/flow if useful; outputs are `plan.md`, `research.md`, `context.md`. Link `https://spektacular.dev/how-it-works/`.

Complexity: Low. Token estimate: ~10k. Agent strategy: single agent, sequential.

### Phase 2.2: Install and getting started

File changes (`README.md`):
- Collapse Quick Start `README.md:36-81` into a concise combined **Install & getting started** section. Keep: it's a Go binary (install methods from Phase 1.2); the minimal path `spektacular init <agent>` → `spec new` → `plan new` → `implement new` (`how-it-works.mdx` 5-step; `cmd/init.go:14-19`). Drop the verbose id_method/external-id prose (lives in configuration + docs). Link `https://spektacular.dev/install/` and `https://spektacular.dev/tutorials/getting-started`.
- Remove the standalone **Spec Format** block `README.md:83-116` from a front-door standpoint, or compress to a one-line pointer to the getting-started/how-it-works docs (the 7 spec sections are documented on the site `how-it-works.mdx`). Decision: replace the full example spec with a brief mention + link, keeping the README concise.

Complexity: Low. Token estimate: ~10k. Agent strategy: single agent, sequential.

### Phase 2.3: Supported agents and extensibility

File changes (`README.md`):
- Add a new **Supported agents** section. Content: claude / bob / codex (`internal/agent/agent.go:48-55`); per-agent install layout — claude → `.claude/skills/` and ensures `CLAUDE.md` imports `@AGENTS.md` (`internal/agent/claude.go:37-79`), bob → `.bob/skills/` + `.bob/commands/` (`bob.go`), codex → `.agents/skills/` (`codex.go`); how integration works (each agent implements the narrow `Agent` interface: `Name()`, `Install()`); how to add an agent (register via `init()` in the agent package). Link `https://spektacular.dev/plugins/` and `https://spektacular.dev/extending/`.
- Replace the long **Extending Storage** section `README.md:178-217` with a concise extensibility note: storage backends and coding agents are both pluggable behind defined Go interfaces (`Store` in `internal/store`, `Agent` in `internal/agent`); only the `file` store ships today; point to the extending docs for full signatures. Do NOT reproduce the full `Store` interface block. (The corrected search-result/`Hit` example moves to the knowledge section, Phase 2.4.)

Complexity: Low. Token estimate: ~10k. Agent strategy: single agent, sequential.

### Phase 2.4: Knowledge base and project structure

File changes (`README.md`):
- Rewrite **Knowledge** section `README.md:136-176`. Current text already has the two-tier idea but is incomplete. Update to the canonical model (`internal/knowledge/category.go:47-104`, `../spektacular-website/src/pages/knowledge-base.mdx`): six categories — always-applied: conventions, glossary; looked-up: architecture, gotchas, learnings, decisions; category-labelled search; consolidated/de-duplicated lookups across scopes (SHA-256 byte-identity, `internal/store/store.go:31`, `internal/knowledge/set.go:102-141`). Link `https://spektacular.dev/knowledge-base/`.
- Fix the knowledge CLI table `README.md:163-170`: it lists 6 commands; add `knowledge categories` and `knowledge always-applied` (`cmd/knowledge.go:57-67`). Decision: trim to a brief representative list + link rather than an exhaustive table, to stay front-door.
- Add the corrected **search-result example** here using the current 7-field `Hit` (`internal/store/store.go:24-32`): scope, path, title, excerpts, score, **category**, **checksum** (the old example at `README.md:202-208` had only 5 fields, missing category + checksum).
- Fix **Project Structure** tree `README.md:122-132`: currently lists only `conventions/architecture/learnings/gotchas` — add `glossary/` and `decisions/` and note each category dir gets a `README.md` (`internal/project/init.go:69-113`). Reflect `config.yaml`, `specs/`, `plans/` as created by init.

Complexity: Low-Medium. Token estimate: ~12k. Agent strategy: single agent, sequential.

### Phase 2.5: Configuration

File changes (`README.md`):
- Refresh **Configuration** `README.md:219-257`. Keep it front-door: describe the configurable agent and the spec/plan/knowledge stores and related settings (the six sections: command, agent, debug, spec, plan, knowledge — `internal/config/config.go:88-133`). Trim the long id_method / normalization prose; keep a short example if helpful and ensure it matches current defaults. Link `https://spektacular.dev/configuration/` for the full reference.

Complexity: Low. Token estimate: ~8k. Agent strategy: single agent, sequential.

### Phase 3.1: Testing section

File changes (`README.md`):
- Revise **Testing** `README.md:267-332`. Mostly accurate. Add the unit suite (`go test ./...`, `make test`). Reflect both Harbor workflow suites — spec-workflow AND plan-workflow (`Makefile:43-73`: targets `harbor-test-spec`, `harbor-test-spec-claude`, `harbor-test-spec-codex`, `harbor-test-plan`). Update the "Available test tasks" table `README.md:330-332` (only lists spec-workflow) to include the plan workflow. Mention the make wrappers exist alongside raw `harbor run`. Keep the `uv`/Harbor install (`README.md:275,280`) — legitimate. Keep the agents shown as claude-code/oracle/codex (real Harbor agents, `Makefile:45-48`).

Complexity: Low. Token estimate: ~8k. Agent strategy: single agent, sequential.

### Phase 3.2: Building, contributing, and license

File changes (`README.md`):
- **Building from Source** `README.md:334-355`: build output path — `make build` produces `./bin/spektacular` (`Makefile:9-10`), not `./spektacular` as `README.md:351` states. The Makefile-targets table `README.md:349-355` lists `make install` which does NOT exist — actual target is `install-local` (`Makefile:21`); correct it. Verified targets: build, test, lint, clean, install-local, cross (`Makefile:9-36`). Drop or fix the standalone `go build -o spektacular .` snippet to match.
- **Contributing** `README.md:357-363`: steps are generic and accurate; `make test` exists. Keep, lightly aligned to `make test`/`make lint`.
- **License** `README.md:365-367`: `Apache 2.0` + `LICENSE` link — verify `LICENSE` file exists and license matches; keep.

Complexity: Low. Token estimate: ~6k. Agent strategy: single agent, sequential.

### Phase 3.3: Whole-document verification

No new edits beyond fixes surfaced here. Read the assembled `README.md` top to bottom and cross-check:
- Every command shown appears in `go run . <cmd> --help` / the command surface (`cmd/*.go`).
- Agents are exactly claude/bob/codex; no Aider/Cursor/Copilot/Gemini/TUI/complexity anywhere (`grep -niE 'aider|cursor|copilot|gemini|tui|complexity|nicholasjackson|v0\.1\.0'` returns nothing outside legitimate contexts).
- All `https://spektacular.dev/...` links target real pages (`../spektacular-website/src/pages/*`); all internal links (`LICENSE`, anchors) resolve.
- The three spec success metrics (all Manual — captured in the implementation test plan): zero contradictions vs docs site + tool help; every named command/agent/capability exists; newcomer-comprehension.

Complexity: Low. Token estimate: ~10k. Agent strategy: single agent, sequential; this is the integration/review pass.



## Testing Strategy

No automated tests are added — the deliverable is prose and the existing Go suite (`go test ./...`) and Harbor e2e harness are unaffected. Verification is per-phase manual review against the two authorities:

- **Phases 1.1–1.2:** mechanical greps confirm removal — no `nicholasjackson`, `Aider`, `Cursor`, `TUI`, `complexity`, or `v0.1.0`; agent list reads exactly claude/bob/codex; `uv` survives only in the Testing section (Harbor install), not the Spektacular-install section.
- **Phases 2.1–2.5:** each rewritten section is cross-checked against its docs-site page (`../spektacular-website/src/pages/*.mdx`) and the relevant code; the `Hit` example is diffed against `internal/store/store.go:24-32` (7 fields); every command shown is confirmed in `cmd/*.go` / `--help`.
- **Phases 3.1–3.2:** every make target / command shown is confirmed against the `Makefile`.
- **Phase 3.3:** the whole-document review carries the spec's three success metrics, all **Manual — captured in the implementation test plan**: (1) zero contradictions vs docs site + tool help; (2) every named command/agent/capability exists; (3) newcomer can describe the tool from the README alone.

## Project References

The README's claims map to these authoritative sources:

- **Docs site (authoritative for capability descriptions + links):** `../spektacular-website/src/pages/{index,how-it-works,knowledge-base,plugins,configuration,extending,install}.mdx`, `src/content/tutorials/getting-started.mdx`; base URL `astro.config.mjs:7` → `https://spektacular.dev`; current release `index.mdx:22` (v0.11.1).
- **Command surface:** `cmd/root.go:76-81` (registration); `cmd/{spec,plan,implement}.go` (new/goto/status/steps); `cmd/knowledge.go:15-67` (8 subcommands); `cmd/init.go:14-19`; `cmd/skill.go:20-93`; `cmd/storefile.go:42-115`.
- **Agents:** `internal/agent/agent.go:28-56`; `internal/agent/claude.go:37-79` (CLAUDE.md→@AGENTS.md import); `bob.go:9-29`; `codex.go:9-22`.
- **Knowledge model:** `internal/knowledge/category.go:47-104` (6 categories, 2 tiers); `internal/store/store.go:24-32` (7-field `Hit`); `internal/store/search.go:172-218` (SHA-256 checksum); `internal/knowledge/set.go:102-141` (merge/precedence/exclusion).
- **Init / on-disk:** `internal/project/init.go:15-117`.
- **Config:** `internal/config/config.go:88-133`.
- **Build / test:** `Makefile:9-41` (build/test/lint/clean/install-local/cross), `Makefile:43-73` (Harbor spec + plan suites); `main.go:1-7`; `go.mod:1` (canonical module path).
- **Current README to revise:** root `README.md` (367 lines); orphaned asset `images/tui.png`; in-repo `docs/knowledge-base.md` (exists, but link the docs site instead).
- Full evidence and rejected alternatives: `research.md`.

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|-------------|----------------|
| Low | ~10k | Single agent, sequential |
| Medium | ~25k | 2-3 parallel agents |
| High | ~50k+ | Parallel analysis, sequential integration |

All ten phases edit the single `README.md` file and therefore run **sequentially with a single agent** (Low tier) regardless of individual phase size — parallel agents would conflict on one file. The parallelism tiers above are recorded for completeness but do not apply to this single-file change.

## Migration Notes

N/A — documentation-only change; no data, schema, or config migration.

## Performance Considerations

N/A — no runtime code is changed.
