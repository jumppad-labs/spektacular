# Spektacular

Agent-agnostic CLI for spec-driven development. Write a markdown spec; Spektacular plans and implements it with the coding agent of your choice.

> **Status:** early development — see the [releases page](https://github.com/jumppad-labs/spektacular/releases) for the latest version.

## What is Spektacular?

Spektacular is a self-contained Go binary that brings spec-driven development to AI coding agents. You write a markdown specification; Spektacular turns it into a reviewed implementation plan and then drives a coding agent to implement it — keeping your intent reviewable at every stage.

Its core competencies:

- **Self-contained binary plus installed agent skills.** A single binary that, on `init`, installs the skills (and commands) your coding agent needs to run the Spektacular workflows.
- **State-machine-driven workflow.** Spec, plan, and implement each run as a stepwise state machine. Spektacular hands the agent one per-step prompt at a time (`new` / `goto` / `steps`), so every stage is resumable — stop, inspect, edit, and resume without losing work.
- **Agent-agnostic, multi-agent support.** Works with claude, bob, and codex; pick the one your team already uses, or register your own.
- **Project knowledge base.** A searchable, layered store of conventions, architecture, gotchas, and learnings that feeds context into planning.

## How It Works

Spektacular follows a three-stage workflow — **spec → plan → implement** — each driven step by step by a state machine:

1. **Spec.** You write a markdown spec (requirements, constraints, acceptance criteria); `spec new` scaffolds one from a template.
2. **Plan.** `plan new` explores your codebase, asks clarifying questions, and writes a detailed implementation plan — `plan.md`, `research.md`, and `context.md`.
3. **Implement.** `implement new` drives the coding agent through each phase of the plan and validates the result against your acceptance criteria.

For the full pipeline, see the [how-it-works documentation](https://spektacular.dev/how-it-works/).

## Install & getting started

Spektacular is a single self-contained Go binary.

```bash
# Homebrew
brew install jumppad-labs/homebrew-repo/spektacular

# Go 1.21+
go install github.com/jumppad-labs/spektacular@latest
```

Or download a pre-built binary from the [releases page](https://github.com/jumppad-labs/spektacular/releases). See the [install docs](https://spektacular.dev/install/) for apt and other methods. You also need a supported coding agent CLI (claude, bob, or codex) installed and configured.

Once installed, the minimal path is initialise → spec → plan → implement:

```bash
# 1. Initialise your project for a coding agent (claude, bob, or codex)
spektacular init claude

# 2. Scaffold a spec, then fill in your requirements
spektacular spec new --data '{"name":"auth-feature"}'
$EDITOR .spektacular/specs/<returned-spec-name>.md

# 3. Generate an implementation plan
spektacular plan new --data '{"name":"<returned-spec-name>"}'

# 4. Implement the plan
spektacular implement new --data '{"name":"<plan-name>"}'
```

Spec names are normalised and prefixed by the CLI, so use the returned `spec_name` and `spec_path` for follow-up commands rather than the name you passed.

Specs are plain markdown with a small set of structured sections (overview, requirements, constraints, acceptance criteria, and so on), and `spec new` scaffolds the template for you. For the full walkthrough and spec format, see the [getting-started tutorial](https://spektacular.dev/tutorials/getting-started) and the [how-it-works documentation](https://spektacular.dev/how-it-works/).

## Supported agents

Spektacular ships with three coding-agent integrations. `spektacular init <agent>` runs the chosen agent's install step, writing its workflow skills (and, where the agent has no skill mechanism, command wrappers) into your project:

- **claude** — installs the workflow skills under `.claude/skills/` and ensures the project's `CLAUDE.md` imports `@AGENTS.md`, so the Spektacular agent rules take effect.
- **bob** — installs skills under `.bob/skills/` and command wrappers under `.bob/commands/`.
- **codex** — installs skills under `.agents/skills/`.

Each integration is deliberately small: an agent implements a narrow `Agent` interface — `Name()` (its CLI identifier) and `Install()` (which writes its workflow artefacts) — and registers itself with the agent package from an `init()` function. Adding a new agent means implementing those two methods and registering the type.

Both the coding agent and the storage layer are pluggable behind defined Go interfaces — the `Agent` interface in `internal/agent` and the `Store` interface in `internal/store` (the read/write/search surface backing the spec, plan, and knowledge stores). Only the `file` store ships today. For the full interface signatures and how to add your own backend or agent, see the [extending documentation](https://spektacular.dev/extending/) and the [plugins overview](https://spektacular.dev/plugins/).

## Project Structure

Running `spektacular init <agent>` creates:

```
.spektacular/
├── config.yaml              # agent, command, debug, and store settings
├── specs/                   # your specification files
├── plans/                   # generated plans (plan.md, research.md, context.md)
└── knowledge/               # default project knowledge source
    ├── conventions/         # always-applied: standing rules, one per file
    ├── glossary/            # always-applied: shared domain/project terms
    ├── architecture/        # looked-up: how the system is built
    ├── gotchas/             # looked-up: sharp edges and traps
    ├── learnings/           # looked-up: empirical findings from past work
    └── decisions/           # looked-up: the reasoning behind choices
```

Each knowledge category directory is scaffolded with a `README.md` describing what belongs in it. By default Spektacular reads `.spektacular/knowledge/` as the `project` knowledge source; additional sources at other scopes — for example a shared `team` directory or a machine-wide `global` one — can be configured under `knowledge.sources` (see [Configuration](#configuration)). See [Knowledge](#knowledge) for how it is organised and consumed.

## Knowledge

Knowledge is the accumulated know-how a project draws on when planning — conventions, glossary terms, architecture notes, gotchas, learnings, and decisions. It is strictly a **planning-time input**: the planning agent reads it while producing a plan, and the relevant parts are written into the plan itself. The implement workflow then consumes only the plan documents — the plan is the contract.

### Six categories, two tiers

Every entry belongs to exactly one of six categories, fixed by the first segment of its path. Each category has a **retrieval tier** that decides when its entries are loaded:

- **Always-applied** — `conventions` (standing rules to follow) and `glossary` (shared domain and project terms). Loaded in full on every planning task, and deliberately excluded from search results so they are never surfaced twice.
- **Looked-up** — `architecture`, `gotchas`, `learnings`, and `decisions`. The larger reference body, fetched only when a search matches, so it can grow without weighing down every task.

The category model — names, tiers, and per-category boundaries — is declared once in code, so it stays consistent across directory scaffolding, search labelling, and retrieval.

### Scopes, search, and de-duplication

A knowledge source has a **scope** label. The default project ships one scope, `project`, backed by `.spektacular/knowledge/`; you can configure additional scopes — a shared `team` directory or a machine-wide `global` one — under `knowledge.sources` (see [Configuration](#configuration)). Every read, search, and convention load fans across all configured scopes in order, and each result is tagged with the scope and category it came from.

Lookups are **consolidated and de-duplicated** across scopes: each entry carries a SHA-256 checksum over its exact bytes, and byte-identical entries appearing in more than one scope collapse to a single result. A search result looks like:

```
Hit {
  scope     // scope label of the originating store (e.g. project, team)
  path      // locator relative to the store root (e.g. gotchas/db-timeouts.md)
  title     // the document's first heading, or the locator when it has none
  excerpts  // compact matched excerpts
  score     // sum of query-term occurrences (ranking)
  category  // category derived from the path (e.g. gotchas, architecture)
  checksum  // SHA-256 over the entry's raw bytes; the byte-identity de-dup key
}
```

For the full model — every category definition, the retrieval tiers, scope precedence, and the de-duplication rationale — see the [knowledge-base documentation](https://spektacular.dev/knowledge-base/).

### CLI

Agents (and you) reach knowledge through the `spektacular knowledge` commands rather than reading the files directly, so access stays consistent across scopes. The main subcommands:

- `knowledge search <query>` — keyword-search every scope (excluding `conventions/`), returning scope- and category-tagged hits
- `knowledge conventions` / `knowledge always-applied` — read the always-applied entries in full
- `knowledge categories` — list the categories and their tiers
- `knowledge read` / `knowledge list` / `knowledge write` — read, list, and write individual entries
- `knowledge sources` — list the configured scopes and their locations

Every subcommand accepts `--schema` to print its input/output JSON schema and exit.

### Capturing knowledge

When research surfaces a durable learning, gotcha, or convention worth keeping, the agent **proposes** the target scope and exact content and waits for your explicit confirmation before writing — it never persists to a knowledge source unprompted. In a Spektacular-initialised repo, the `spek-knowledge` skill is the entry point for reading, contributing to, and updating the knowledge base in any session, and coding agents route what they would otherwise save to their own per-user memory into the project knowledge base instead, so captured knowledge lands in git and travels with the project.

## Configuration

`.spektacular/config.yaml` controls which coding agent Spektacular drives and the provider-based `spec`, `plan`, and `knowledge` stores. Each of `spec`, `plan`, and `knowledge` names a `provider` (only `file` ships today) and carries a provider-specific `config` block:

```yaml
command: spektacular
agent: claude
debug:
  enabled: false
spec:
  provider: file
  id_method: timestamp              # how new spec identifiers are generated
  config:
    directory: .spektacular/specs   # project-root-relative directory for spec files
plan:
  provider: file
  config:
    directory: .spektacular/plans   # project-root-relative directory for plan files
knowledge:
  sources:
    - scope: project        # written by init; synthesised if removed
      provider: file
      config:
        location: .spektacular/knowledge
    - scope: team           # optional: a shared, hand-configured source
      provider: file
      config:
        location: /shared/team-kb
```

The six top-level sections are `command`, `agent`, `debug`, `spec`, `plan`, and `knowledge`. `spec.id_method` chooses how new spec filenames are prefixed (`timestamp` by default, or `counter` / `external`); `spec.config.directory`, `plan.config.directory`, and each `knowledge` source `location` resolve relative to the project root, and omitting a section falls back to the defaults shown above.

For the full reference — every key, the id-method semantics, name-normalisation rules, and `${VAR}` expansion — see the [configuration documentation](https://spektacular.dev/configuration/).

## Testing

Spektacular has two layers of tests: a fast Go unit suite, and an end-to-end [Harbor](https://harborframework.com/) harness that runs the workflows against real AI coding agents inside sandboxed Docker containers.

### Unit tests

```bash
go test ./...   # or: make test
```

### End-to-end (Harbor)

#### Prerequisites

- Docker
- [uv](https://docs.astral.sh/uv/) (Python package manager)

#### Install Harbor

```bash
uv tool install harbor
```

#### Run the oracle (scripted) tests

The oracle agent runs a scripted solution to validate the test harness itself —
no AI tokens required:

```bash
harbor run -p tests/harbor/spec-workflow -a oracle -o tests/harbor/jobs
```

#### Run with a real agent

Harbor needs an auth token to run Claude Code inside the container. If you use
Claude Max (OAuth), export the token from your local credentials:

```bash
export ANTHROPIC_AUTH_TOKEN=$(python3 -c "import json; print(json.load(open('$HOME/.claude/.credentials.json'))['claudeAiOauth']['accessToken'])")
```

If you use an API key instead, export that:

```bash
export ANTHROPIC_API_KEY=sk-ant-...
```

Then run:

```bash
harbor run -p tests/harbor/spec-workflow -a claude-code -m claude-sonnet-4-6 -o tests/harbor/jobs
```

Makefile wrappers run the suites for you, building the binary and wiring up the agent-specific placeholders:

```bash
make harbor-test-spec          # spec workflow (claude)
make harbor-test-spec-codex    # spec workflow (codex)
make harbor-test-plan          # plan workflow (claude)
```

#### Test results

Results are written to `tests/harbor/jobs/` (gitignored). Each run produces:

```
tests/harbor/jobs/<timestamp>/
├── result.json                    # Overall pass/fail and metrics
└── spec-workflow__<id>/
    ├── agent/                     # Agent output log
    ├── verifier/
    │   ├── test-stdout.txt        # pytest output
    │   └── reward.txt             # 1 = pass, 0 = fail
    └── trial.log                  # Full trial log
```

#### Available test tasks

| Task | Description |
|---|---|
| `tests/harbor/spec-workflow` | Full spec creation workflow, end to end |
| `tests/harbor/plan-workflow` | Full plan generation workflow, end to end |

## Building from Source

```bash
# build binary
make build

# run tests
make test

# cross-compile for all platforms
make cross
```

`make build` produces the binary at `./bin/spektacular`. The `Makefile` targets:

| Target | Description |
|---|---|
| `make build` | Build the `./bin/spektacular` binary |
| `make test` | Run `go test ./...` |
| `make lint` | Run `go vet ./...` |
| `make clean` | Remove build artefacts |
| `make install-local` | Build and copy the binary to `/usr/local/bin` |
| `make cross` | Cross-compile for darwin/linux/windows (amd64 + arm64) |

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b my-feature`)
3. Make your changes
4. Run the tests and vet checks (`make test`, `make lint`)
5. Submit a pull request

## License

[Apache 2.0](LICENSE)
