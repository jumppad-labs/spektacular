# Research: 000023_context-over-memory

## Alternatives considered and rejected

### Option A: New dedicated file (e.g. `.agents/redirect-memory.md`) referenced from each agent's entrypoint

A new shared instruction file at a neutral path, with thin pointers added to `CLAUDE.md`, `AGENTS.md`, and `.bob/rules/rules.md` so each agent picks it up.

**Rejected**: Adds a new instruction surface for every agent to discover when `AGENTS.md` already exists, is already loaded by all three agents, and already serves as the convention-bearing instruction file in the repo (see `AGENTS.md:1-5` and `CLAUDE.md:3`). Three touch points instead of one with no benefit; violates the spec requirement "applies to all coding agents… without per-agent duplication" in spirit by adding per-agent pointer ceremony.

### Option B: Inline the same redirect text in `CLAUDE.md`, `AGENTS.md`, and `.bob/rules/rules.md`

Repeat the same redirect text in each agent's primary instruction file directly, with no shared canonical location.

**Rejected**: Directly violates spec § Requirements: "The instruction must apply to all coding agents the team uses (Claude, Bob, Codex, and any future ones), without per-agent duplication." Also creates wording-drift risk: editing one copy and forgetting the others would produce silently inconsistent behavior between agents.

### Option C: Put the text in `.tessl/RULES.md` (the file `AGENTS.md` already imports)

`AGENTS.md` already does `@.tessl/RULES.md follow the [instructions](.tessl/RULES.md)`, so editing `.tessl/RULES.md` would propagate to every agent that loads `AGENTS.md`.

**Rejected**: `.tessl/RULES.md` says in its own body (`.tessl/RULES.md:3`) "This file is updated when running `tessl install`. If a linked file is missing, make sure to run the command to download any missing tiles from the registry." Any hand-added redirect text would be overwritten on the next `tessl install`, which directly violates the spec constraint "Must not break tessl integration with this repo."

### Option D: Add a thin pointer in `.bob/rules/rules.md` to `AGENTS.md`

Add a one-line "also follow ../../AGENTS.md" reference in `.bob/rules/rules.md` to be safe in case Bob does not natively load `AGENTS.md`.

**Rejected**: User confirmed during the discovery step ("I am pretty sure bob reads Agents.md") that Bob loads `AGENTS.md` natively, so the pointer is unnecessary. The implement workflow has a documented STOP-and-ask point if this turns out to be wrong (see § Open assumptions below and plan.md § Open Questions).

## Chosen approach — evidence

The chosen design — a single new section appended to `AGENTS.md` below the existing tessl-managed block — holds up because:

- `AGENTS.md:3` carries the `<!-- tessl-managed -->` marker, and `AGENTS.md:5` is the last managed line. Text appended after line 5 is outside the tessl-managed block, so `tessl install` will not overwrite it.
- `CLAUDE.md:3` is `@AGENTS.md`, which Claude Code resolves natively. So Claude inherits anything we put in `AGENTS.md` with no Claude-side change.
- `AGENTS.md` is the de-facto convention Codex reads from a repository root (the empty `.codex` marker file at the repo root signals Codex is configured for this project; Codex's instruction surface is `AGENTS.md` itself).
- Per user confirmation during discovery, Bob loads `AGENTS.md` natively in addition to `.bob/rules/`. This is the load-bearing assumption; see § Open assumptions.
- The destination (`spek-knowledge` skill / `go run . knowledge` CLI) is already live — confirmed by the available-skills list including `spek-knowledge` and by `go run . knowledge sources` returning the configured project store at `/home/nicj/code/github.com/jumppad-labs/spektacular/.spektacular/knowledge`.
- The Spektacular CLI's agent-install code (`internal/agent/claude.go:13-18`, `internal/agent/codex.go:13-15`, `internal/agent/bob.go:13-18`) only installs skills and command wrappers into `.claude/skills`, `.agents/skills`, `.bob/skills`, and `.bob/commands`. None of these paths overlap with `AGENTS.md`, `CLAUDE.md`, or `.bob/rules/`, so the edit cannot conflict with `go run . project init` or any reinstall.

## Files examined

- `AGENTS.md:1-5` — current contents: title plus tessl-managed `@.tessl/RULES.md` line; only 5 lines total.
- `CLAUDE.md:1-3` — three lines; just `@AGENTS.md` import. Confirms Claude picks up `AGENTS.md` automatically.
- `.codex` — zero-byte marker file at repo root. Confirms Codex is configured for the repo; Codex reads `AGENTS.md` natively, not this file.
- `.bob/rules/rules.md:1-19` — hand-written Bob guidance about mandatory documentation queries. Independent of `AGENTS.md`; unaffected by this plan.
- `.bob/settings.json` — only `{"tools": {"exclude": []}}`. No instruction-surface configuration.
- `.tessl/RULES.md:1-3` — declares itself overwritten by `tessl install`. Confirms it's unsafe to hand-edit.
- `tessl.json` — `{"name":"my-project","mode":"vendored","dependencies":{...}}`. No instruction-surface configuration.
- `internal/agent/claude.go:13-18` — installs skills under `.claude/skills`; does not touch `CLAUDE.md` or `AGENTS.md`.
- `internal/agent/codex.go:13-15` — installs skills under `.agents/skills`; does not touch `AGENTS.md`.
- `internal/agent/bob.go:13-18` — installs skills and commands under `.bob/skills` and `.bob/commands`; does not touch `.bob/rules/` or `AGENTS.md`.
- `internal/agent/` listing — no other Go file in the package references `AGENTS.md`, `CLAUDE.md`, or `.bob/rules`.

## External references

None. The change is entirely a repo-local instruction edit; no RFCs, library docs, or third-party references are relevant.

## Prior plans / specs consulted

- **`.spektacular/specs/000020_context.md`** — establishes the multi-source knowledge store and the local `FileStore` backend behind `.spektacular/knowledge/`. Provides the destination this redirect targets.
- **`.spektacular/specs/000022_spek-knowledge-skill.md`** — establishes the `spek-knowledge` skill, the agent-facing wrapper around the knowledge CLI, with the propose-then-confirm guard reused by this plan.
- **`.spektacular/specs/000023_context-over-memory.md`** — this plan's source spec.
- `go run . plan file list` returned 22 prior plans; none address per-user-memory redirect, so no prior plan informs the design.

## Open assumptions

- **Bob loads `AGENTS.md` natively.** User stated "I am pretty sure bob reads Agents.md" during discovery. The plan assumes this. **If wrong**, the implement workflow MUST STOP at the post-edit Bob smoke test and consult the user before adding a pointer in `.bob/rules/rules.md` or inlining the redirect text in a `.bob/rules/` file.

## Rehydration cues

If context is lost mid-implementation, rebuild it cold by:

- Reading the source spec: `go run . spec file read 000023_context-over-memory.md`.
- Reading the plan: `go run . plan file read 000023_context-over-memory/plan.md` and `go run . plan file read 000023_context-over-memory/context.md`.
- Re-reading the four instruction-surface files with the `Read` tool: `AGENTS.md`, `CLAUDE.md`, `.bob/rules/rules.md`, `.tessl/RULES.md`.
- Confirming the knowledge-store destination is still live: `go run . knowledge sources`.
- Confirming the `spek-knowledge` skill is still installed by checking the available-skills surface in a fresh agent session.
