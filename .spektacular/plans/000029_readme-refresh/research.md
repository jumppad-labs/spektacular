# Research: 000029_readme-refresh

## Alternatives considered and rejected

- **Rewrite the README from scratch.** Rejected — the spec's Technical Approach explicitly says to revise in place where already accurate (testing, building/contributing) so existing good content is preserved. Full rewrite risks losing accurate prose and re-introducing drift. (spec `.spektacular/specs/000029_readme-refresh.md:57`)
- **Reproduce install/getting-started/config/command-reference content in the README.** Rejected — spec requires the README stay a concise front door that orients briefly and links to the docs site for depth (`...:14,23`). The site already covers these in full (`spektacular-website/src/pages/install.mdx`, `configuration.mdx`, `content/tutorials/getting-started.mdx`).
- **Hardcode the version `v0.11.1` as a prominent badge.** Rejected as the primary mechanism — a pinned number re-drifts (that is exactly how the current `v0.1.0` got stale, `README.md:5`). Prefer an "early development" status note plus a link to releases/site; if a number is shown, it must be ≥ current release `v0.11.1` (spec AC `...:49`). (`spektacular-website/src/pages/index.mdx:22`)
- **Copy the `Hit` example from the docs site's extending page.** Rejected — `extending.mdx` shows a STALE 4-field struct (`Scope, Path, Excerpt, Score`); the spec requires examples match current tool behaviour and forbids fixing the site (non-goal). Source the example from code instead (7 fields). (`spektacular-website/src/pages/extending.mdx` vs `internal/store/store.go:24-32`)
- **Keep the standalone "Roadmap" section.** Rejected — not one of the spec's required sections; "early development" status belongs in the intro/version note, and planned backends (obsidian/notion/jira) fold into the extensibility note. (current `README.md:259-265`; `spektacular-website/src/pages/plugins.mdx` planned plugins)

## Chosen approach — evidence

Refresh the single root `README.md` in place: keep accurate sections (testing, building), fix stale ones, add missing ones, delete invented ones (TUI, complexity scoring, model routing). Source every claim from the docs site (authoritative, base URL `https://spektacular.dev`) and the code.

- Command surface (what actually ships): `cmd/root.go:76-81` (registers spec/plan/implement/knowledge/skill/init), `cmd/spec.go:70-97`, `cmd/plan.go:38-65`, `cmd/implement.go:43-70`, `cmd/knowledge.go:15-67` (8 subcommands incl. `categories`, `always-applied`), `cmd/init.go:14-19`, `cmd/skill.go:20-93`, `cmd/storefile.go:42-115` (file read/write/delete/list).
- Supported agents = exactly claude, bob, codex: `internal/agent/agent.go:28-56`, `internal/agent/claude.go:77-79`, `internal/agent/bob.go:27-29`, `internal/agent/codex.go:20-22`. Docs corroborate: `spektacular-website/src/pages/index.mdx:65`, `plugins.mdx`.
- Knowledge model — 6 categories + 2 tiers: `internal/knowledge/category.go:47-90` (conventions, glossary, architecture, gotchas, learnings, decisions); tiers `category.go:10-20` (`TierAlwaysApplied`, `TierLookedUp`); `AlwaysApplied()` single source of truth `category.go:96-104`. Docs: `spektacular-website/src/pages/knowledge-base.mdx`.
- Search result `Hit` shape (7 fields): `internal/store/store.go:24-32` — scope, path, title, excerpts, score, category, checksum. Schema confirmed `cmd/knowledge.go:69-88`.
- Dedup = exact SHA-256 byte-identity (`store.go:31`, checksum computed `internal/store/search.go:172-218`); scope precedence / merge / always-applied exclusion `internal/knowledge/set.go:102-141` (rank: score desc → source order → path; most-specific scope wins).
- Init on-disk effects: `internal/project/init.go:15-117` creates `.spektacular/{specs,plans,knowledge/<category>/README.md}`, `config.yaml`, `.gitignore`; per-agent install `cmd/init.go:21-51`. Claude ensures `CLAUDE.md` imports `@AGENTS.md`: `internal/agent/claude.go:37-75` (idempotent).
- Config (six sections): `internal/config/config.go:88-133` — command, agent, debug, spec (provider/id_method/directory), plan (provider/directory), knowledge.sources[] (scope/provider/location); `${VAR}` expansion `config.go:252-258`. Docs: `configuration.mdx`.
- Module path canonical = `github.com/jumppad-labs/spektacular` (`go.mod:1`); stale `nicholasjackson` refs at `README.md:47,55`.
- Testing: unit `go test ./...` (`Makefile:12`); Harbor e2e harness, suites spec-workflow + plan-workflow with claude/codex agents (`Makefile:43-72`). Test files: `cmd/implement_test.go:50-62`, `cmd/spec_test.go`, `cmd/plan_test.go`.
- Building: `make build` → `go build -ldflags "-X .../cmd.version=$(VERSION)"` (`Makefile:9-10`); cross-compile `Makefile:36-41`; Dagger module `Makefile:24-32`. Entry `main.go:1-7`.
- Plugin/extensibility interfaces: Store + Agent pluggable behind Go interfaces — Store at `internal/store` (Read/Write/Delete/List/Exists/Search), Agent at `internal/agent` (Name/Install). Docs: `extending.mdx`, `plugins.mdx`.
- Public docs URLs (base `https://spektacular.dev`, `spektacular-website/astro.config.mjs:7`): `/how-it-works/`, `/knowledge-base/`, `/plugins/`, `/extending/`, `/configuration/`, `/install/`, `/tutorials/getting-started`.

## Files examined

- `README.md:1-367` — current stale README: claims Aider/Cursor (`:11`), complexity routing (`:11`), TUI section (`:28-34`), `nicholasjackson` URLs (`:47,55`), v0.1.0 (`:5`), 6-of-8 knowledge cmds (`:165-170`), uv/pip-era install. Accurate-ish: project structure, building.
- `cmd/root.go:12,76-81` — version var `0.1.0` (overridden by ldflags at build); top-level command registration; `--fields` global flag.
- `cmd/{spec,plan,implement}.go` — workflow command groups: new/goto/status/steps + `--schema`, `--dry-run/-n`, `-d/--data`, `--force`, `--stdin`, `--file`.
- `cmd/knowledge.go:15-88` — 8 subcommands; search `--schema` output fields.
- `cmd/init.go:14-51` — `init <agent>` flow: project.Init then agent.Install.
- `cmd/storefile.go:42-115` — `<workflow> file` read/write/delete/list, `--from`.
- `internal/agent/{agent,claude,bob,codex}.go` — registry + the 3 agents and their install layouts (.claude/skills, .bob/skills+commands, .agents/skills).
- `internal/knowledge/{category,set}.go` — category registry, tiers, search merge/exclusion/precedence.
- `internal/store/{store,search}.go` — Hit struct, checksum/dedup.
- `internal/config/config.go:88-133` — config schema + defaults.
- `internal/project/init.go:15-117` — on-disk scaffolding.
- `Makefile:9-72` — build, cross, Dagger, Harbor e2e.
- `go.mod:1` — canonical module path.

## External references

- `spektacular-website/src/pages/index.mdx:22-65` — tagline "Write the spec. Ship the software.", version `v0.11.1 · early development`, 3 agents, brew install. Authoritative landing copy + current version.
- `.../how-it-works.mdx` — spec→plan→implement pipeline, 5-step quick start, 7 spec sections. Source for "how it works" summary.
- `.../knowledge-base.mdx` — definitive KB description (6 categories, 2 tiers, dedup, layered scopes). Mirror this language.
- `.../plugins.mdx` — shipping (file/claude/bob/codex) vs planned (obsidian/notion/jira) plugins.
- `.../extending.mdx` — Store/Agent interfaces; NOTE: stale Hit example (do not copy).
- `.../configuration.mdx` — 6 config keys reference.
- `.../install.mdx` — brew / go install / apt / releases; macOS+Linux.
- `.../content/tutorials/getting-started.mdx` — end-to-end SDD walkthrough; per-agent init detail incl. CLAUDE.md.
- `astro.config.mjs:7` — site base URL `https://spektacular.dev`.

## Prior plans / specs consulted

- `000003_readme` plan — produced the CURRENT (now-stale) README: TUI section, uv/pip install, complexity/model-tier config, Roadmap, `nicholasjackson` paths. Confirms exactly what to remove/replace.
- `000028_knowledge-base-categories-tiers-and-dedup` spec — the feature the current README predates; informs the knowledge section content.
- Searchable KB (`architecture/initial-idea.md`, example-output JSONs) — STALE/non-authoritative (mentions Copilot/Gemini, old `.spectacular/` path). Do not source README claims from it.

## Open assumptions

- Current published release is `v0.11.1` per `index.mdx:22`; if a newer release exists at implement time, the README must not state anything older. Mitigation: prefer status-note + releases link over a hardcoded number.
- The docs site domain `https://spektacular.dev` (from astro config) is the live public URL. If links should instead be relative or point elsewhere, implement must confirm before finalizing link targets.
- Site `extending.mdx` Hit example is stale relative to code; README intentionally diverges from the site here to satisfy "examples match current behaviour." If a reviewer flags README-vs-site mismatch on the Hit shape, that is expected and correct.

## Rehydration cues

- Re-read spec: `go run . spec file read 000029_readme-refresh.md` (CLI, not Read tool).
- Re-verify command surface: `go run . spec/plan/implement/knowledge --help`; `go run . knowledge categories`; `go run . knowledge search <q> --schema`.
- Re-read code anchors: `internal/knowledge/category.go`, `internal/store/store.go`, `internal/project/init.go`, `internal/agent/*.go`, `internal/config/config.go`, `cmd/*.go`, `Makefile`, `go.mod`.
- Docs site: `../spektacular-website/src/pages/*.mdx` + `src/content/tutorials/getting-started.mdx`; base URL in `astro.config.mjs`.
- Current README to revise: root `README.md` (367 lines).
