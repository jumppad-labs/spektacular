# Spektacular

Agent-agnostic CLI for spec-driven development. Write a markdown spec, get an implementation plan.

> **Status:** v0.1.0 — early development

## What is Spektacular?

Spektacular takes a markdown specification and uses AI coding agents to produce a detailed, actionable implementation plan. Instead of jumping straight into code, it follows a structured pipeline: **spec → analyse → plan → execute → validate**.

It works with any coding agent (Claude Code, Aider, Cursor) and routes work to different models based on complexity — so simple tasks use cheaper models and complex tasks get the heavy hitters.

## How It Works

```
spec.md → [Analyse] → complexity score → [Plan] → plan.md
              ↑                             ↑
         cheap model               scaled by complexity
```

1. You write a spec in markdown (requirements, constraints, acceptance criteria)
2. Spektacular scores the complexity of the task
3. An AI agent researches your codebase and generates a detailed plan
4. You review the plan and implement it

The planning agent explores your codebase, asks clarifying questions through an interactive TUI, and produces structured output: `plan.md`, `research.md`, and `context.md`.

## TUI

![](./images/tui.png)

The plan command launches an interactive terminal UI built with [Bubble Tea](https://github.com/charmbracelet/bubbletea). It streams agent output as markdown, shows tool usage in real time, and presents questions with numbered options you can answer by pressing a key.

Press `t` to cycle through 5 built-in color themes (GitHub Dark, Dracula, Nord, Solarized, Monokai).

## Quick Start

### Prerequisites

- Go 1.21+
- [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) installed and configured with an API key

### Install

```bash
# build from source
git clone https://github.com/nicholasjackson/spektacular.git
cd spektacular
go build -o spektacular .

# copy to PATH
cp spektacular /usr/local/bin/
```

Or download a pre-built binary from the [releases page](https://github.com/nicholasjackson/spektacular/releases).

### Usage

```bash
# 1. Initialize a new project for your agent
spektacular init claude

# 2. Create a spec from the workflow
spektacular spec new --data '{"name":"auth-feature"}'

# 3. Edit the spec_path returned by the command to add your requirements
$EDITOR .spektacular/specs/<returned-spec-name>.md

# 4. Generate an implementation plan using the returned spec_name
spektacular plan new --data '{"name":"<returned-spec-name>"}'
```

Spec names are normalized and prefixed by the CLI. Use the returned `spec_name` and `spec_path` for follow-up workflows instead of assuming the requested `name` is the final filename.

External systems can pass their own identifier as the prefix:

```bash
spektacular spec new --data '{"name":"billing-export","id":"EXT-123"}'
```

Passing `id` is accepted for timestamp and counter projects and is required when `spec.id_method` is `external`.

## Spec Format

Specs are plain markdown files with a simple structure:

```markdown
# Feature: User Authentication

## Overview
Add OAuth2 login with Google and GitHub providers.

## Requirements
- [ ] Users can sign in with Google OAuth2
- [ ] Users can sign in with GitHub OAuth2
- [ ] Session persists across browser refreshes

## Constraints
- Must use existing Express backend
- No new dependencies over 50KB gzipped

## Acceptance Criteria
- [ ] Login redirects to provider, returns with valid session
- [ ] Session cookie is httpOnly, secure, sameSite=strict

## Technical Approach
Use passport.js for OAuth2 strategy management.

## Success Metrics
Login flow completes in under 3 seconds.

## Non-Goals
Social login with Apple or Microsoft.
```

Create a new spec with `spektacular spec new --data '{"name":"auth-feature"}'` to get this template.

## Project Structure

Running `spektacular init <agent>` creates:

```
.spektacular/
├── config.yaml              # CLI command, agent, debug, and provider settings
├── specs/                   # Your specification files
├── plans/                   # Generated plans (plan.md, research.md, context.md)
└── knowledge/               # Default project knowledge source
    ├── conventions/         # Always-apply standards, one rule per file
    ├── architecture/        # System design docs
    ├── learnings/           # Captured corrections from past runs
    └── gotchas/             # Known issues and workarounds
```

Knowledge feeds context to the planning agent. By default Spektacular reads `.spektacular/knowledge/` as the `project` knowledge source; additional sources at other scopes — for example a shared `team` directory or a machine-wide `global` one — can be configured under `knowledge.sources` (see [Configuration](#configuration)). Adding architecture docs and past learnings improves plan quality over time. See [Knowledge](#knowledge) for how it is organised and consumed.

## Knowledge

Knowledge is the accumulated know-how a project draws on when planning — conventions, architecture notes, gotchas, and lessons from past work. It is strictly a **planning-time input**: the planning agent reads it while producing a plan, and the relevant parts are written into the plan itself. The implement workflow then consumes only the plan documents, so it never needs to re-read the knowledge base — the plan is the contract.

> For the full model — the six knowledge categories and their definitions, the two retrieval tiers, the per-result category label, de-duplication and consolidation, the exact-checksum rationale, and layered source precedence — see [docs/knowledge-base.md](docs/knowledge-base.md). The summary below is an overview.

### Two tiers

Knowledge is split into two tiers by *how it is accessed*, not where it is stored:

- **Conventions (always-apply).** The standards a project commits to — every plan must reflect them. They live one-rule-per-file under a reserved `conventions/` directory in each knowledge source, so a single convention can be added or changed independently of the others. They are **read in full** at planning time.
- **Reference knowledge (on-demand).** Architecture notes, gotchas, learnings, and anything else. These live anywhere else under a knowledge source and are reached only through **keyword search**, targeted at the specific surfaces a feature touches.

Conventions are deliberately **excluded from search results** — since they are already read in full, surfacing them again would be redundant noise. This keeps planning focused even as the reference knowledge base grows.

### Scopes

A knowledge source has a **scope** label. The default project ships one scope, `project`, backed by `.spektacular/knowledge/`. You can configure additional scopes — for example a shared `team` directory or a machine-wide `global` one — under `knowledge.sources` (see [Configuration](#configuration)). Every read, search, and convention load fans across all configured scopes in order, and results are tagged with the scope they came from.

### How planning uses it

During the plan workflow's **discovery** step, the agent loads every convention in full and runs surface-targeted searches over the reference knowledge for the design surfaces the feature introduces. As the design is locked in the **architecture** step, the agent selects the subset of conventions that actually bear on the work — each with a one-line reason it applies — and the finished `plan.md` carries them in a dedicated `## Conventions` section. When nothing is relevant (or a project has no conventions), the section says so plainly; an empty or generic section is a visible signal that the knowledge base was not consulted.

### CLI

Agents (and you) reach knowledge through the `spektacular knowledge` commands rather than reading the files directly, so access stays consistent across scopes:

| Command | What it does |
|---|---|
| `spektacular knowledge conventions` | Read every always-apply convention across all scopes, returning each one's full body, scope-tagged |
| `spektacular knowledge search <query>` | Keyword-search every scope (excludes `conventions/`); a document matches when every query word occurs in it, and results come back ranked — one scope-tagged result per matching document, with title, score, and excerpts |
| `spektacular knowledge read --data '{"scope":"project","path":"architecture/x.md"}'` | Read one entry's full body from a named scope |
| `spektacular knowledge list` | List every entry across all scopes |
| `spektacular knowledge write --data '{"scope":"project","path":"gotchas/x.md"}' --file <path>` | Write an entry into a named scope (content from `--file`, or stdin) |
| `spektacular knowledge sources` | List the configured scopes and their locations |

Every subcommand accepts `--schema` to print its input/output JSON schema and exit.

### Capturing knowledge

When research surfaces a durable learning, gotcha, or convention worth keeping, the agent **proposes** the target scope and exact content and waits for your explicit confirmation before writing — it never persists to a knowledge source unprompted. In a Spektacular-initialised repo, the `spek-knowledge` skill is the entry point for reading, contributing to, and updating the knowledge base in any session, and coding agents route what they would otherwise save to their own per-user memory into the project knowledge base instead, so captured knowledge lands in git and travels with the project.

## Extending Storage

Spektacular reads and writes every file — specs, plans, and knowledge entries — through a single `Store` interface, so a new backend can be added without touching the workflows. The interface lives in `internal/store`:

```go
type Store interface {
    Root() string                              // absolute path of the store root
    Read(path string) ([]byte, error)          // file contents at path
    Write(path string, content []byte) error   // create or overwrite, creating parent dirs
    Delete(path string) error                  // remove path; nil if it does not exist
    List(path string) ([]DirEntry, error)      // direct children, each typed file-or-dir
    Exists(path string) bool                   // whether a file or directory exists
    Search(query string) ([]Hit, error)        // keyword search, returning scope-tagged hits
}
```

`List` returns typed entries so a caller can recurse a tree, and `Search` returns compact, scope-tagged results:

```go
type DirEntry struct {
    Name  string // child name, not a full path
    IsDir bool   // true for a subdirectory — recurse into it via List
}

type Hit struct {
    Scope    string   // scope label of the originating store
    Path     string   // locator relative to the store root — pass to Read
    Title    string   // the document's first heading, or the locator when it has none
    Excerpts []string // compact excerpts, each capped at the excerpt budget
    Score    float64  // sum of all query terms' occurrences across the document
}
```

**Worked example: `FileStore`.** `internal/store/store.go` and `internal/store/search.go` implement `Store` over the local filesystem. It is the model for a new backend:

- Construct it with `NewFileStore(root, scope string)` — `root` is the directory it is rooted at, `scope` is the label every `Hit` it produces is tagged with.
- All paths are resolved relative to `root`; the `abs` helper rejects path traversal so a caller cannot escape the root.
- `Search` is built-in and self-contained: an in-process directory walk matching each query word as a case-insensitive literal substring, returning one hit per document that contains every word, excluding `conventions/` directories and skipping binary files — no external tool is needed.

To add a backend (e.g. a remote or GitHub-hosted store), implement the seven `Store` methods on a new type, then register it as a provider: the `knowledge` layer resolves a configured source's `provider` field to a concrete `Store` in `knowledge.NewSet` (`internal/knowledge/set.go`), where today only the `file` provider is wired. Add a new `case` there for the new provider name.

## Configuration

`.spektacular/config.yaml` controls the installed agent command and the provider-based `spec`, `plan`, and `knowledge` settings. Each of `spec`, `plan`, and `knowledge` names a `provider` (only `file` ships today) and carries a provider-specific `config` block:

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

`spec.id_method` controls the prefix used for new spec filenames. It sits beside `provider` rather than inside the provider's `config` block, because identifier generation is independent of the storage backend:

- `timestamp` (default): creates names like `20260509010203-billing-export`; collisions bump by one second until unused.
- `counter`: creates names like `000001_billing-export`, deriving the next number from existing spec files.
- `external`: requires an `id` in `spec new --data`; useful when another system owns the identifier.

`spec.config.directory` and `plan.config.directory` are resolved relative to the project root (like `knowledge` source `location` values); omitting a section falls back to the defaults shown above. `knowledge.sources` is an ordered list of scoped sources. `init` writes the default `project` source at `.spektacular/knowledge` into the config explicitly; if the section is removed entirely, Spektacular falls back to synthesising that same `project` source. Relative source `location` values resolve against the project root, so `team` and `global` sources can point at absolute paths shared across projects.

Names and ids are normalized to lowercase, with accepted separators such as `.`, `@`, `-`, and internal whitespace converted to hyphens. Leading or trailing whitespace, path separators, and control characters are rejected.

## Roadmap

- **v0.2** — Automated execution via coding agent subprocess, validation agent, GitHub Issues integration
- **v0.3** — MCP server integration, multiple agent backends, cost tracking
- **v1.0** — Parallel task execution, plugin system, CI integration

See the [architecture document](.spektacular/knowledge/architecture/initial-idea.md) for the full vision.

## Testing

Spektacular uses [Harbor](https://harborframework.com/) to run end-to-end tests against
real AI coding agents inside sandboxed Docker containers.

### Prerequisites

- Docker
- [uv](https://docs.astral.sh/uv/) (Python package manager)

### Install Harbor

```bash
uv tool install harbor
```

### Run the oracle (scripted) tests

The oracle agent runs a scripted solution to validate the test harness itself —
no AI tokens required:

```bash
harbor run -p tests/harbor/spec-workflow -a oracle -o tests/harbor/jobs
```

### Run with a real agent

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

### Test results

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

### Available test tasks

| Task | Description |
|---|---|
| `tests/harbor/spec-workflow` | Full spec creation workflow through all 10 steps |

## Building from Source

```bash
# build binary
make build

# run tests
make test

# cross-compile for all platforms
make cross
```

The `Makefile` targets:

| Target | Description |
|---|---|
| `make build` | Build `./spektacular` binary |
| `make test` | Run `go test ./...` |
| `make lint` | Run `go vet ./...` |
| `make install` | Build and copy to `$GOPATH/bin` |
| `make cross` | Build for darwin/linux/windows (amd64 + arm64) |

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b my-feature`)
3. Make your changes
4. Run tests (`make test`)
5. Submit a pull request

## License

[Apache 2.0](LICENSE)
