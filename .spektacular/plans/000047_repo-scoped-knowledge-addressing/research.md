---
created_date: "2026-09-03"
status: completed
closed_date: "2026-09-03"
---

# Research: 000047_repo-scoped-knowledge-addressing

## Alternatives considered and rejected

**Packed single-string address (`"repo:docs"`, `"project:team"`) instead of separate
`tier` + `name` fields.** One field, no pairing rules. Rejected: the spec fixes the
vocabulary as tier + name + a flat `filter` list, and a packed string needs parsing
and escaping rules that a flat repeatable CLI flag avoids. The packed form already
exists internally as a *display* label only (`spektacular:cmd/storefile.go:133` builds
`"repo:"+repoName` for a changelog store's label), which is evidence it works as a
label and not as an input contract. It also cannot express "tier `all`, no name".

**Keep `scope` as the store name and add an optional `repo` qualifier (non-breaking).**
Every current invocation keeps working; ambiguity errors only when a bare scope matches
more than one store. Rejected by the spec on three counts: a write missing either field
must be refused "whether or not what was given happens to identify a single store"; a
configuration in the superseded form must be rejected rather than aliased; and a
repository "must not be able to ... name or label the one [store] it has", which a
retained `scope` key would still allow.

**Keep `sources:` as a list in `repo.yaml` with a validated maximum of one entry.**
Smallest config diff, and `config.RepoConfig.WithDefaults`
(`spektacular:internal/config/repo.go:99`) already synthesises exactly one. Rejected by
the spec's technical approach: it "removes the vestigial per-source label rather than
leaving a one-element list that must always say the same thing". The single provider
block also matches the sibling `changelog:` key in the same file
(`spektacular:internal/config/config.go:98`), so the two sections read alike.

**Push `tier` and `name` down into `store.Hit` and `store.NewSourceStore`.** The store
would tag its own hits, as it tags `Scope` today (`spektacular:internal/store/search.go:90`).
Rejected: `NewSourceStore` is also used for the changelog stores
(`spektacular:cmd/storefile.go:95,133`), which have no knowledge tier, and the knowledge
layer already demonstrates the cleaner seam by stamping `hit.Category` after search
(`spektacular:internal/knowledge/set.go:136`). Tier and name belong on the same seam.

**Move `tier`/`filter` into `--data` for `search`, `list` and `always-applied` so the
existing `input` schema covers them.** Rejected as the primary route: `knowledge search`
takes its query as a positional argument and every step template and the skill invoke it
that way; converting it to `--data` breaks all of them for a schema-shape reason. The
narrowing stays on repeatable flags (following the existing `--repo` flag at
`spektacular:cmd/knowledge.go:404` and the `--status`/`--created-after` filters at
`spektacular:cmd/storefile.go:338`), and the published schema is extended to describe
flags. See the open assumption on `commandSchema`.

**Redefine the layered "most-specific scope wins" precedence chain under the new tiers.**
Rejected: no code implements it. `internal/knowledge/set.go` has no precedence logic at
all; the chain exists only as prose in
`spektacular:templates/skills/workflows/spek-knowledge/SKILL.md:39`,
`spektacular:docs/knowledge-base.md:151-153` and `spektacular:README.md:128`. The spec
does not ask for precedence, so the chain is deleted from the prose rather than
reinterpreted.

## Chosen approach — evidence

**The defect and its single point of resolution.**
- `spektacular:internal/knowledge/set.go:288` — `byScope` scans `s.sources` for the
  first entry whose `scope` matches and returns it; it errors only when none matches.
- `spektacular:internal/knowledge/set.go:170,180` — `Set.Read` and `Set.Write` both
  route through `byScope`, so both inherit first-match resolution.
- Reproduced live in this project (two repos, both declaring `scope: project`):
  `go run . knowledge read --data '{"scope":"project","path":"conventions/mdx-authoring.md"}'`
  returns `{"error":true,"code":"internal_error","message":"not found","next_action":""}`
  even though that entry exists in the `docs` store, while
  `conventions/tests-must-pass-for-done.md` (only in the `spektacular` store) reads back
  fine. The `docs` store is unreachable for read and write today.
- The same run shows a second defect: the failure surfaces as `internal_error` with an
  empty `next_action`, against the project convention at
  `conventions/error-messages-must-suggest-remediation.md`.
- `spektacular:cmd/knowledge_test.go:637` — `TestKnowledgeReadWrite_DuplicateScopeResolvesToFirstRegistryRepo`
  asserts the defect as intended behaviour ("Read resolves to the colocated repo's copy",
  "Write lands in the colocated repo's store"). It must be inverted, not extended.

**The structure already present to narrow.**
- `spektacular:internal/knowledge/set.go:20-26` — `scopedStore` already carries `repo`.
- `spektacular:internal/knowledge/set.go:62-67` — `SourceInfo` already exposes `Repo`.
- `spektacular:internal/knowledge/set.go:217,248-257` — `AlwaysAppliedEntries` already
  filters by a set of repo names. Note the exemption at line 255: a source with no repo
  attribution is never filtered out. The spec's "Narrowing is honoured uniformly" and
  "No store is implicitly included or excluded" require that exemption to be removed.
- `spektacular:cmd/knowledge.go:180-212` — `aggregateKnowledgeSources` is where every
  repo-declared source is stamped with `src.Repo = e.Name`; it is the natural place to
  stamp tier and name.
- Verified empirically that repo narrowing already works end to end:
  `go run . knowledge always-applied --repo docs` returns only the docs conventions and
  `--repo spektacular` only the CLI's, but every entry comes back tagged `scope: project`
  with no store identity, which is the reporting gap the spec closes.

**Where tier/name must be stamped and reported.**
- `spektacular:internal/store/store.go:25-33` — `store.Hit` carries `Scope`; the only
  consumer is the knowledge layer, and `Category` is already left empty by the store and
  populated by `Set.Search` (`spektacular:internal/knowledge/set.go:136`).
- `spektacular:internal/knowledge/set.go:34,42,52` — `Entry`, `Convention` and
  `AlwaysAppliedEntry` each carry `Scope` and need the tier/name pair.
- `spektacular:cmd/knowledge.go:76-146` — the eight published schemas, all of which
  describe `scope`.

**Config: the shape change and the rejection.**
- `spektacular:internal/config/repo.go:36` — `RepoConfig.Knowledge KnowledgeConfig`
  (the `sources:` list). Target shape is a single provider block mirroring
  `Changelog ChangelogConfig` on line 38 and `ChangelogConfig` at
  `spektacular:internal/config/config.go:98-101`.
- `spektacular:internal/config/repo.go:43-63` — `NewDefaultRepoConfig` is the single
  function producing every scaffolded `repo.yaml`'s knowledge section.
- `spektacular:internal/config/config.go:115-123` — `SourceConfig.Scope` becomes `Name`
  for project-owned shared stores; `Repo` stays a programmatic, never-serialised field.
- `spektacular:internal/config/config.go:456-473` — `KnowledgeConfig.Validate` enforces
  non-empty and unique `Scope`; it becomes the per-tier name uniqueness check.
- **Rejection precedent, to copy directly:**
  `spektacular:internal/config/config.go:275-300` — `rejectLegacyRepoAddress` re-parses
  the raw YAML into a loose `map[string]any` shape (because the typed struct no longer
  has a field the removed key could land in), then returns
  `output.NewError("config_invalid", ...).WithResource(path).WithNextAction(...)` naming
  the file, what was found, and exactly what to write instead. Called from
  `FromYAMLFile` at line 261, after the typed unmarshal.
- **Second precedent** (shape-level, less good because it emits a bare `fmt.Errorf` with
  no `next_action`): `spektacular:internal/config/repo.go:162-177` —
  `RepoSourceConfig.UnmarshalYAML` rejects the superseded scalar `source:` form and
  prints the replacement block inline in the message.
- `spektacular:internal/config/repo.go:78-85` — `RepoConfigFromYAMLFile` wraps validation
  errors with `%w`, and `spektacular:cmd/root.go:252` uses `errors.As`, so a typed
  `*output.ErrorResponse` survives the wrap and reaches the JSON envelope intact.

**Scaffolding and other Go consumers of the scope literal.**
- `spektacular:internal/project/init.go:91-97` — filters the repo's sources by
  `src.Scope != config.DefaultKnowledgeScope` to find "the project source" to scaffold.
  Collapses to the single provider block.
- `spektacular:internal/repo/footprint.go:56-63` — the identical loop and literal-string
  filter inside `EnsureFootprint`, the shared write path for `init` and `repo add`.
- `spektacular:cmd/repo.go:198-200` — `repo add` builds `NewDefaultRepoConfig()` and
  calls `EnsureFootprint`.
- `spektacular:internal/repo/set.go:80,186,215` — `Entries()`, `Resolve(name)` and
  `Footprint(name)` are the registry API supplying repo-tier store names.

**Migration does not collide.**
- `spektacular:cmd/version.go:151-177` — `detectMigrationNeeded` is a pure file-presence
  sniff (`config.yaml` present, `repo.yaml` absent). It never parses `config.yaml`.
- `spektacular:cmd/version.go:242-270` — `executeMigration` byte-copies `config.yaml` to
  `config.yaml.old` and writes a *fresh* `repo.yaml` from
  `config.NewDefaultRepoConfig()` via `scanProjectMetadata`
  (`spektacular:cmd/version.go:182-236`), which only fills description/role/tags. It
  never reads an existing `knowledge:` block, so it synthesises the new shape for free
  once `NewDefaultRepoConfig` changes, and the spec's "reject, never migrate" constraint
  is not violated by it.

**Skill and template surface (all generated from one source).**
- `spektacular:internal/agent/skills.go:25-30,40-65` — `workflowSkills` maps
  `spek-knowledge` to `skills/workflows/spek-knowledge/SKILL.md`;
  `installWorkflowSkills` renders `{{command}}` via mustache and writes into the agent's
  skills dir. `spektacular:internal/agent/claude.go:22` passes `.claude/skills`.
- Source of truth: `spektacular:templates/skills/workflows/spek-knowledge/SKILL.md`
  (uses `{{command}}`). Generated, checked-in copies that go stale until `init` re-runs:
  `spektacular:.claude/skills/spek-knowledge/SKILL.md` and
  `spektacular:.bob/skills/spek-knowledge/SKILL.md` (byte-identical to each other).
- Scope-bearing lines in the skill: 31 (search result fields), 37 (read `--data`),
  39 (the layered precedence chain), 44 (citation shape "scope + path"), 52 (`knowledge
  sources` "to enumerate configured scopes"), 55-59 (contribute: choose scope, propose,
  write), 67-75 (update: locate, read, rewrite to same scope+path), 80 (decline path).
  Lines 50-61 and 65-76 hold the propose-then-confirm flow the spec's
  "Recording an entry on the user's behalf shows where it will land" attaches to.
- Step templates: `spektacular:templates/steps/plan/02-discovery.md:14` (the
  `--repo <name>` always-applied load, and the sentence "project-owned sources always
  load regardless of `--repo`" which the uniform-narrowing requirement contradicts),
  `:16` (search + read `--data '{"scope":...}'`), `:29` (search), `:58` (sources + write,
  "propose a target scope"); `spektacular:templates/steps/plan/03-architecture.md:45`
  (consumes always-applied output); `spektacular:templates/steps/plan/18-walkthrough.md:22`
  and `spektacular:templates/steps/implement/07-update_changelog.md:51` (both say the
  skill "owns scope selection"); `spektacular:templates/skills/skill_spawn-planning-agents.md:22`
  ("tagged by scope (`project`, `team`, `global`)").
- Agent-rules templates: `spektacular:templates/agents/memory-context.md:10-12` and
  `spektacular:templates/agents/knowledge-trigger.md:24-25` each say the skill "handles
  scope selection". Wording fix only; the spec's constraint forbids adding a managed
  section for the knowledge model.
- `spektacular:templates/*_test.go` — the `templates` package already holds
  content-assertion tests over the template corpus (e.g.
  `TestRepoListConsumersSendTheAgentToRepoList` in `repo_source_test.go:98`), which is
  the established way to pin template wording.

**Documentation surface (wider than the spec's technical approach anticipated).**
- `docs:src/pages/knowledge-base.mdx` — the reference page named by the spec. Five
  `Section`s alternating `surface` false/true/false/true/false: "What it is" (17),
  "The six categories" (44, `surface`), "The lifecycle of an entry" (87),
  "Configuration" (141, `surface`), "Why it works this way" (204), then a `CtaBanner`
  (260). Bodies are wrapped in `<Prose nested>`; no `ConfigKey` on this page.
  Configuration section (141-202) describes sources as scope+provider+location and shows
  both the `repo.yaml` and `config.yaml` blocks. Lifecycle section shows the write
  invocation at 99-107. Line 249-254 states the precedence chain.
- `docs:src/pages/configuration.mdx:28,57-62,165-179,233-255,310-316` — the `config.yaml`
  example, the `ConfigKey name="knowledge"` block documenting
  `knowledge.sources[].scope`, the full `repo.yaml` example, and the repo-side
  `ConfigKey`. These become factually wrong the moment the config shape changes.
- `docs:src/pages/projects.mdx:67-72` — a `repo.yaml` knowledge block in the multi-repo
  walkthrough.
- `docs:src/content/tutorials/getting-started.mdx:606-613,619-648,746-772` — the
  "Multi-source knowledge" section. Its example puts a `project` scope directly in
  `config.yaml`, contradicting `configuration.mdx`; the tier model resolves that.
- `docs:src/pages/index.mdx:65-68` ("tags each result by scope"),
  `docs:src/pages/extending.mdx:59-60,78-79` (documents `Hit.Scope`).
- `spektacular:README.md:112-128,132-138,184,204-209,218` — the "Scopes, search, and
  de-duplication" section, the CLI subcommand list, the `Hit` struct listing, and both
  config examples.
- `spektacular:docs/knowledge-base.md:84,96,141-157,167,173-185` — a second, in-repo copy
  of the same model including the precedence section and a full command-reference table
  with `--data '{"scope":...}'` shapes.
- Docs component API: `Section.astro` (props `heading` required, `sub?`, `surface?`
  default false, `maxWidth?` default `max-w-[1100px]`; named `sub` slot wins over the
  prop; default body slot), `ConfigurationKeys.astro` (`surface?` defaults **true**),
  `ConfigKey.astro` (`name` required, `type?`, `defaultValue?` rendered with `set:html`;
  default slot only), `Prose.astro` (`nested?`).

**Verification toolchain.**
- `spektacular`: `go test ./...` — confirmed green on this branch before any change
  (all 20 packages `ok`).
- `docs`: `npm run build` (= `astro build`) and `npx astro check` (not a package.json
  script; the `Makefile` `check` target runs it). CI (`.github/workflows/deploy.yml`)
  runs `npm ci && npm run build` only. `node_modules` is present locally, node v22.18.0.
- The MDX Rule 1 guard is manual, not wired into CI:
  `grep -nE "<div|<section|class=" src/pages/*.mdx` must return zero matches.

**Test conventions to model on.**
- `spektacular:internal/knowledge/set_test.go:12-53` — `writeFile` and `twoScopeSet`
  helpers build sources over `t.TempDir()`; assertions use `testify/require` with
  hand-written literal expectations, and each test carries a `// Criterion N:` comment
  tying it to an acceptance criterion.
- `spektacular:cmd/knowledge_test.go:16-56` — package-local mirror structs
  (`knowledgeHit`, `knowledgeEntry`, `knowledgeSource`, `alwaysAppliedEntry`,
  `knowledgeCategory`) decode the JSON envelope independently of the production types.
  They are hand-maintained oracles and all carry `Scope` today.
- `spektacular:cmd/knowledge_test.go:60-71` — `resetKnowledgeFlags` clears every cobra
  flag plus the `knowledgeAlwaysAppliedRepos` package var between runs; new flags must be
  added here or they leak across subtests.
- `spektacular:cmd/knowledge_test.go:73-116` — `twoScopeProject` writes `config.yaml` as
  a raw YAML string (pinning the wire format) and the repo config via
  `config.NewDefaultRepoConfig().ToYAMLFile`.
- `spektacular:cmd/knowledge_test.go:118-135` — `runKnowledge` drives the real
  `rootCmd`/`runRoot()` and returns a typed `*output.ErrorResponse` on non-zero exit.
- `spektacular:cmd/root_test.go` — cross-command tables that invoke
  `runRootCmd(t, "knowledge", "sources")` among every other command family, asserting a
  uniform JSON error envelope; a new tier/name error must stay consistent with them.
- `spektacular:cmd/docs_test.go:31-66` — asserts `README.md` and `CHANGELOG.md` content
  at the repo root, so both are under test and a breaking change is expected to land an
  entry in each.

## Files examined

- `spektacular:internal/knowledge/set.go:20-26` — `scopedStore` already carries `repo`.
- `spektacular:internal/knowledge/set.go:34-67` — `Entry`, `Convention`, `AlwaysAppliedEntry`, `SourceInfo`, all keyed on `Scope`.
- `spektacular:internal/knowledge/set.go:74-100` — `NewSet` resolves sources into stores; errors name the scope.
- `spektacular:internal/knowledge/set.go:107-147` — `Search` merges, ranks, stamps `Category`, drops always-applied categories.
- `spektacular:internal/knowledge/set.go:170-203` — `Read`, `Write`, `List`.
- `spektacular:internal/knowledge/set.go:217-276` — `AlwaysAppliedEntries`, `Conventions`, `readCategories` and its project-owned filter exemption at 255.
- `spektacular:internal/knowledge/set.go:288-295` — `byScope`, the defect.
- `spektacular:internal/knowledge/set_test.go:12-53,637` — helper shapes and the criterion-comment convention.
- `spektacular:internal/knowledge/category.go` — the category registry driving scaffolding and the always-applied set.
- `spektacular:cmd/knowledge.go:19-146` — command tree and all eight published schemas.
- `spektacular:cmd/knowledge.go:153-212` — `newKnowledgeSet` and `aggregateKnowledgeSources`, where repo attribution is stamped.
- `spektacular:cmd/knowledge.go:214-355` — the eight run functions.
- `spektacular:cmd/knowledge.go:357-407` — `knowledgeScopePathInput`, its validation, and flag registration.
- `spektacular:cmd/knowledge_test.go:16-135,637-700` — mirror structs, helpers, and the defect-enshrining test.
- `spektacular:cmd/storefile.go:95-133,338-348` — the `--repo` routing precedent and the flag-based filter precedent.
- `spektacular:cmd/spec.go:28-46` — `schemaProp`, `schemaObj`, `commandSchema`, shared by every command.
- `spektacular:cmd/root.go:117,247-256,333` — error rendering; `errors.As` recovers a wrapped `*output.ErrorResponse`.
- `spektacular:cmd/version.go:61-73,105-111,151-177,182-270` — the migration sniff, prompt, and execution.
- `spektacular:cmd/docs_test.go:12-66` — README/CHANGELOG assertions.
- `spektacular:internal/config/config.go:98-131` — `ChangelogConfig` (the target shape), `KnowledgeConfig`, `SourceConfig`, `FileKnowledgeConfig`.
- `spektacular:internal/config/config.go:255-300` — `FromYAMLFile` and `rejectLegacyRepoAddress`, the rejection precedent.
- `spektacular:internal/config/config.go:456-495` — `KnowledgeConfig.Validate` and `WithDefaults`.
- `spektacular:internal/config/config.go:51-62` — `DefaultRepoKnowledgeLocation`, `DefaultKnowledgeScope`, `DefaultKnowledgeLocation`.
- `spektacular:internal/config/repo.go:32-63` — `RepoConfig` and `NewDefaultRepoConfig`.
- `spektacular:internal/config/repo.go:78-114` — `RepoConfigFromYAMLFile`, `Validate`, `WithDefaults`.
- `spektacular:internal/config/repo.go:162-177` — `RepoSourceConfig.UnmarshalYAML`, the shape-rejection precedent.
- `spektacular:internal/store/store.go:25-33,61-77` — `Hit` and the `FileStore` scope label.
- `spektacular:internal/store/search.go:90` — where `Hit.Scope` is populated.
- `spektacular:internal/store/ignore.go:76-78` — `NewSourceStore(root, scope)`.
- `spektacular:internal/project/init.go:60-130` — knowledge-root resolution and scaffolding.
- `spektacular:internal/repo/footprint.go:29-85` — `EnsureFootprint`.
- `spektacular:internal/repo/set.go:57-233` — registry API (`Entries`, `LocalRoot`, `Resolve`, `Footprint`).
- `spektacular:internal/agent/skills.go:25-65` — how skills are rendered and installed.
- `spektacular:templates/skills/workflows/spek-knowledge/SKILL.md:31-82` — the skill's scope-bearing flow.
- `spektacular:templates/steps/plan/02-discovery.md:14,16,29,58` — the heaviest knowledge consumer.
- `spektacular:templates/steps/plan/03-architecture.md:45`, `18-walkthrough.md:22`, `templates/steps/implement/07-update_changelog.md:51`, `templates/skills/skill_spawn-planning-agents.md:22` — the remaining scope-bearing template lines.
- `spektacular:templates/agents/memory-context.md:10-12`, `templates/agents/knowledge-trigger.md:24-25` — "scope selection" wording.
- `spektacular:templates/repo_source_test.go:98` — template content-assertion precedent.
- `spektacular:README.md:112-138,184,204-218` and `spektacular:docs/knowledge-base.md:84-185` — in-repo documentation of the scope model.
- `spektacular:.spektacular/repo.yaml`, `spektacular-website/.spektacular/repo.yaml`, `spektacular:.spektacular/config.yaml` — the live configs that must be hand-corrected; the project declares no shared stores today, so the `project` tier is currently empty here.
- `docs:src/pages/knowledge-base.mdx:17-260` — the reference page's full structure.
- `docs:src/pages/configuration.mdx:28,57-62,165-179,233-255,310-316` — config examples and `ConfigKey` blocks.
- `docs:src/pages/projects.mdx:67-72,129,148`, `docs:src/content/tutorials/getting-started.mdx:606-772`, `docs:src/pages/index.mdx:65-68`, `docs:src/pages/extending.mdx:59-79`, `docs:src/pages/debugging.mdx:43` — the secondary documentation surface.
- `docs:src/components/sections/{Section,ConfigurationKeys,ConfigKey,SectionHeader,Prose}.astro` — component props and slots.
- `docs:package.json`, `docs:Makefile`, `docs:.github/workflows/deploy.yml` — build and check commands.

## External references

None. The change is entirely internal to the two repos: no new library, protocol, or
external standard is involved. Astro 5 and Tailwind v4 are already in use on the docs
site and their behaviour is unchanged by this work.

## Prior plans / specs consulted

`go run . plan file list` returns 47 prior plans (its envelope key is `files`, not `plans`).
The five that bear on this work are below.

- `000028_knowledge-base-categories-tiers-and-dedup/plan.md` — **the most important one to
  read.** It introduced the category registry and, with it, the word *tier* in this codebase:
  `Category{Name, Purpose, Boundary, Tier, EntryShape}` where `Tier` is `always-applied` or
  `looked-up`. That is a **different axis** from the tier this spec introduces, and the Go type
  `knowledge.Tier` (`internal/knowledge/category.go:13`) is already taken by it. The same plan
  also introduced the consolidated, de-duplicated lookup with "the most specific source taking
  precedence", which is where the layered precedence prose this work deletes originates. Note
  the precedence was specified as assistant behaviour during consolidation, not as store
  resolution, which is why no code implements it.
- `000047_repo-scoped-knowledge-addressing.md` — the spec this plan is built from.
  Fixes the vocabulary (tier / name / flat `filter`), forbids migrating or aliasing the
  superseded config form, forbids resolving an under-specified write on the caller's
  behalf, forbids a new managed agent-rules section for the knowledge model, and requires
  two supplied layout entries be recorded into the `docs` repo's own store through the
  delivered behaviour.
- `000046_relocatable-repo-footprint.md` — the immediately preceding work on this branch;
  established that a repo's Spektacular files may sit outside its code and that
  `repo list` reports a `root` per repo. Explains why `aggregateKnowledgeSources`
  resolves relative locations against each repo's root rather than the project root.
- `000042_repo-self-describing-metadata.md` — moved a repo's description/role/tags out of
  the project registry into its own `repo.yaml`, and removed the `address` key. That
  removal is the direct precedent for this work's config rejection: it is the change
  `rejectLegacyRepoAddress` was written for.
- `000045_config-file-migration.md` — the single-file to split-file migration. Confirmed
  it is a file-presence sniff that never parses the knowledge section, so it does not
  collide with the reject-never-migrate constraint.
- `000039_project-level-capabilities` (per `CHANGELOG.md`) — introduced multi-repo
  projects, which is what created the ambiguity this work fixes.

## Open assumptions

If any of these turn out wrong, the implement workflow must STOP and ask.

1. **`tier: all` is rejected for `read` and `write`.** Both address exactly one store,
   and a name could exist in both tiers, so `all` is only meaningful for `search`,
   `list` and `always-applied`. The spec says reading states "its tier, its store name,
   and its location" but does not explicitly forbid `all`.
2. **The layered precedence chain is deleted, not reinterpreted.** No code implements
   it; it exists only in skill and documentation prose. Nothing in the spec asks for
   precedence between tiers.
3. **`--repo` on `always-applied` is removed outright, not aliased to `--filter`.**
   Consistent with the spec's refusal to alias superseded forms, but the spec's
   constraint is written about *configuration*, not flags. This breaks
   `templates/steps/plan/02-discovery.md:14`, which is updated in the same change.
4. **Store names are unique per tier only.** A repo named `team` and a shared store
   named `team` may coexist; tier disambiguates. This is what "unique within its tier"
   in the spec means.
5. **The published `commandSchema` gains a `flags` block** (`omitempty`, so every other
   command's schema output is byte-identical) so `search`, `list` and `always-applied`
   can advertise `tier` and `filter` without moving them into `--data`. This is a change
   to a shared type used by every command family.
6. **Documentation beyond the knowledge-base reference page must be corrected.** The
   spec's technical approach says the documentation "fits on the existing knowledge base
   reference page", but `configuration.mdx`, `projects.mdx` and the getting-started
   tutorial carry `knowledge.sources[].scope` examples that become factually wrong, as do
   `README.md` and `docs/knowledge-base.md` in the CLI repo. Those are correctness fixes,
   not new explanatory prose, and are assumed to be in scope.
7. **The checked-in generated skill copies are updated by hand in the same change.**
   `.claude/skills/spek-knowledge/SKILL.md` and `.bob/skills/spek-knowledge/SKILL.md` are
   tracked in git and rendered from the template only when `init` runs; leaving them
   stale would ship a skill that still says `scope`.
8. **This project's own two `repo.yaml` files and any `config.yaml` knowledge block are
   hand-corrected as part of delivery.** The spec requires old-form configs be rejected,
   and both live `repo.yaml` files are in the old form, so every knowledge command in
   this repo fails until they are edited. That includes the dogfood write.
9. **The word "tier" carrying two meanings is acceptable.** The spec fixes `tier` as the
   addressing field name, and `knowledge categories` already publishes a `tier` field meaning
   something else. Assumed that keeping both wire names, renaming the Go type behind the older
   one, and qualifying the older one as "retrieval tier" in prose is preferable to renaming
   either field. If the user would rather rename one of them, that changes published output and
   must be settled before Phase 1.1.
10. **`store.Hit` keeps a single generic label seam.** Tier and name are stamped by the
   knowledge layer after search, as `Category` already is, so the changelog stores built
   through `NewSourceStore` are untouched.

## Drafting assumptions

### `tier: all` rejected on read and write (discovery)
- **Decision**: `all` is accepted only by `search`, `list` and `always-applied`. `read` and `write` require `project` or `repo`.
- **Rationale**: both address exactly one store, and a name may legitimately exist in both tiers, so `all` would reintroduce the ambiguity the spec exists to remove.
- **Rejected**: allowing `all` and erroring only when the name matches in both tiers, which is the same "resolve if unambiguous" behaviour the spec forbids on the write path.

### The layered scope precedence chain is deleted, not reinterpreted (discovery)
- **Decision**: drop the `project > team > global` "most-specific wins" model from the skill, `README.md`, `docs/knowledge-base.md` and the docs site rather than restating it in tier terms.
- **Rationale**: no code implements it. `internal/knowledge/set.go` has no precedence logic; the chain is prose only, and the spec asks for no precedence between tiers.
- **Rejected**: defining repo tier as more specific than project tier, which would be a new behavioural claim the spec never made and nothing enforces.

### `--repo` is removed rather than aliased to `--filter` (discovery)
- **Decision**: `knowledge always-applied --repo <name>` is replaced by `--tier` plus repeatable `--filter`, with no alias or deprecation period.
- **Rationale**: matches the spec's stance on superseded forms, and keeps one narrowing vocabulary across search, list and always-applied. The only in-tree caller is a template updated in the same change.
- **Rejected**: keeping `--repo` as a hidden alias, which would leave two ways to say the same thing and a flag whose name implies the repo tier is special.

### Store names are unique per tier, not globally (discovery)
- **Decision**: uniqueness is enforced within a tier; a repo and a shared store may share a name.
- **Rationale**: the spec says "a name unique within its tier", and tier is always supplied alongside the name.
- **Rejected**: global uniqueness, which would let a project's choice of shared-store name collide with a repo name it does not control.

### Narrowing on search, list and always-applied stays on flags (discovery)
- **Decision**: `tier` and `filter` are repeatable CLI flags, and `commandSchema` gains an `omitempty` `flags` block so they are still published in the machine-readable interface.
- **Rationale**: `knowledge search` takes its query positionally and is invoked that way by the skill and four template lines; converting it to `--data` for a schema-shape reason would break all of them. Flag-based filters already exist at `cmd/storefile.go:338` and `cmd/knowledge.go:404`.
- **Rejected**: moving `tier`/`filter` into `--data` (breaks every existing search invocation); leaving them undocumented in the schema (fails the spec's published-interface requirement).

### Documentation outside the knowledge-base page is in scope (discovery)
- **Decision**: correct `configuration.mdx`, `projects.mdx`, `getting-started.mdx`, `index.mdx` and `extending.mdx` on the docs site, plus `README.md` and `docs/knowledge-base.md` in the CLI repo, alongside the explanatory rewrite of `knowledge-base.mdx`.
- **Rationale**: those pages carry `knowledge.sources[].scope` examples and `Hit.Scope` field listings that become factually wrong once the config and result shapes change. Leaving them is shipping documentation that contradicts the tool.
- **Rejected**: restricting the change to `knowledge-base.mdx` as the spec's technical approach literally says, which satisfies the wording but leaves four other pages telling readers to write a configuration the tool now rejects.

### Tier and name are stamped by the knowledge layer, not the store (discovery)
- **Decision**: `store.Hit` gains `Tier` and `Name` fields that the store leaves empty and `Set.Search` populates, exactly as `Category` is handled today.
- **Rationale**: `NewSourceStore` also backs the changelog stores, which have no knowledge tier; keeping the seam in the knowledge layer leaves them untouched.
- **Rejected**: threading tier and name through `NewSourceStore` and `FileStore`, which forces two unrelated call sites in `cmd/storefile.go` to invent values they have no use for.

### Chosen direction: addressing vocabulary owned by the knowledge layer (architecture)
- **Decision**: put `Address{Tier,Name}` and `Selector{Tier,Filter}` in `internal/knowledge`, delete `byScope`, and stamp tier/name during aggregation in `cmd/knowledge.go`. The store below and the command surface above stay generic.
- **Rationale**: the defect lives in resolution, and the spec says concentrating the change there closes it for every caller at once. `scopedStore` already carries the repo attribution the tier model needs, and `Set.Search` already demonstrates the post-search stamping seam with `Category`.
- **Rejected**: resolving addressing in the CLI by pre-computing packed labels like `repo:docs`, which leaves first-match resolution intact one layer down and forces callers to unpack a string to learn the tier; and pushing tier and name into `store.NewSourceStore`/`store.Hit`, which drags the two changelog stores into a knowledge concept they have no use for.

### Repo knowledge collapses to a single provider block (architecture)
- **Decision**: `repo.yaml`'s `knowledge:` becomes `provider:` + `config:`, mirroring the `changelog:` key beside it. The project's `config.yaml` keeps `knowledge.sources[]` and renames `scope:` to `name:`.
- **Rationale**: the spec requires a repo be unable to declare several stores or label the one it has, and a fixed one-element list would still permit both in the file even if validation rejected them. The sibling `changelog:` key gives the shape a precedent in the same file.
- **Rejected**: a `sources:` list validated to exactly one entry, which keeps a vestigial label the spec asks to remove.

### Conventions applied, and the scope of the em-dash rule (architecture)
- **Decision**: apply the two `spektacular` conventions (remediation-bearing errors, passing tests) and five `docs` conventions (content outlines in plans, no em dashes, MDX authoring, alternating section backgrounds, label-before-filename headings). Treat **no em dashes** as binding on prose authored into the `docs` repo only.
- **Rationale**: that convention lives in the `docs` repo's own store and describes that site's authoring style; the CLI repo's `README.md` and `CHANGELOG.md` use em dashes throughout and restyling them is outside this spec. The spec's own constraint phrases the rule as the documentation repository's convention.
- **Rejected**: applying it repo-wide, which would turn a knowledge-addressing change into an unrelated rewrite of the CLI repo's documentation; and dropping it for docs-site prose, which the spec explicitly forbids.

### `Tier` is a named string type, not a bare string (data structures)
- **Decision**: define `type Tier string` with three constants rather than passing tiers as plain strings.
- **Rationale**: the tier is a closed set of three values checked at every entry point; a named type makes an invalid value hard to construct accidentally and gives the validation one obvious home. It costs a conversion at the JSON and YAML boundary only.
- **Rejected**: bare strings compared against constants, which is what `scope` does today and is how the current literal-string comparisons in scaffolding drifted from the config shape.

### `SourceInfo` drops its separate `Repo` field (data structures)
- **Decision**: report tier and name only; a repo-tier store's name is its registry name.
- **Rationale**: keeping `Repo` alongside `Name` would carry the same fact twice and invite the two to disagree. The spec asks every result to report its tier and store, not its tier, store, and originating repo.
- **Rejected**: keeping `Repo` for backward compatibility, which contradicts the spec's refusal to preserve superseded forms and would leave callers unsure which field addresses a read.

### A filter naming an unreachable store is an error, not an empty result (data structures)
- **Decision**: `Selector` validation fails when a filter names a store the chosen tier does not contain.
- **Rationale**: the spec requires a caller be able to predict exactly which stores a request covers. Silently returning nothing for a typo'd or wrong-tier name is the same class of silent misdirection this work exists to remove.
- **Rejected**: ignoring unknown filter names, which is more forgiving but hides a mistake behind a plausible-looking empty result.

### `Conventions` gains a selector it did not have (data structures)
- **Decision**: give the conventions-only view the same `Selector` parameter as every other fan-out operation.
- **Rationale**: "narrowing is honoured uniformly" admits no exempt retrieval path, and leaving one command unable to narrow would be exactly the implicit inclusion the spec forbids.
- **Rejected**: leaving `Conventions` unfiltered as a backward-compatible view, which was its original justification but is now the one hole in a rule the spec states without exception.

### Correcting the live configs is part of the configuration change, not a later step (dependencies)
- **Decision**: treat editing this project's two `repo.yaml` files as belonging to the same unit of work as the rejection that makes them invalid.
- **Rationale**: the moment the rejection lands, every knowledge command in this repository fails, which would leave the working tree unable to run its own tooling between phases. The spec's own dogfood requirement depends on the corrected files.
- **Rejected**: correcting them at the end alongside the dogfood write, which leaves an intermediate state where the tool cannot read its own knowledge base.

### Metric 2 classified as manual, with its durable half asserted (testing approach)
- **Decision**: classify "the dogfood write succeeds first time" as manual, captured in the implementation test plan, while asserting behaviourally that the two entries are readable from the documentation repository's store, absent elsewhere, and reported with that store as their source.
- **Rationale**: "needed no second attempt" is a property of the delivery act and cannot be re-observed by a test run later; the entries' final location can, and is an acceptance criterion in its own right.
- **Rejected**: classifying the whole metric as behavioural, which would quietly redefine it as something weaker than what the spec asks; and classifying it as wholly manual, which would drop a checkable acceptance criterion.

### The defect-asserting test is inverted, not deleted (testing approach)
- **Decision**: rewrite the existing test that asserts first-match resolution so it asserts the refusal, keeping a test at that exact point.
- **Rationale**: deleting it leaves the codebase with no standing test at the site of the bug, which is where regression cover is most valuable.
- **Rejected**: deleting it and relying on the new resolution tests, which cover the rule but not the specific ambiguous-project scenario that produced the defect.

### Addressing lands before the configuration shape change (milestones)
- **Decision**: deliver the tier/name/filter vocabulary first against the existing configuration shape, then change the configuration shape and add its rejections as a second milestone.
- **Rationale**: it makes the first milestone independently deliverable and separately testable without breaking any existing project, and it isolates the breaking change into a milestone whose whole subject is that break. A repository's store can be named by its registry entry and a shared store by its existing key without touching either file's shape.
- **Rejected**: one combined milestone, which would make the largest and the most disruptive change land together with no working intermediate state; and doing configuration first, which would break every project before the addressing that justifies the new shape exists.

### The dogfood write is its own milestone, placed last (milestones)
- **Decision**: record the two supplied entries as a fourth milestone, after the skill has been updated.
- **Rationale**: this repository's own rules require knowledge writes to be routed through the knowledge skill rather than issued as raw commands, so the skill must already speak the new addressing. Keeping it separate also keeps its success metric, that it needed no second attempt, observable rather than buried inside a larger milestone.
- **Rejected**: folding it into the documentation milestone, which would obscure whether the write itself succeeded first time; and doing it immediately after the CLI change, which would mean bypassing the skill the project's rules mandate.

### `KnowledgeConfig.WithDefaults` is deleted rather than adapted (phases)
- **Decision**: remove the function that synthesises a default project-owned knowledge source when none is configured, and drop its call site in `NewSet`.
- **Rationale**: it exists to manufacture a `project`-scoped store at `.spektacular/knowledge` when the project declares none. Under the tier model that location is a repository's own store, owned by `RepoConfig.WithDefaults`, and synthesising an unnamed shared store would create exactly the implicitly-included store the spec forbids. A project declaring no shared stores should have an empty project tier.
- **Rejected**: keeping it and giving the synthesised source a default name, which invents a store nobody declared and makes the project tier non-empty by surprise.

### The store keeps a generic label parameter (phases)
- **Decision**: rename `FileStore.scope` to `label` and keep `NewFileStore`/`NewSourceStore` taking it, passing `"<tier>:<name>"` from the knowledge layer.
- **Rationale**: the two changelog stores in `cmd/storefile.go` construct source stores too and already pass labels of exactly that form; removing the parameter would force them to change for no benefit, and keeping a label preserves store-level diagnostics.
- **Rejected**: removing the parameter entirely, which touches two unrelated call sites; and giving the store real tier and name fields, which was rejected at the architecture step for the same reason.

### Selector validation against available names lives on the set (phases)
- **Decision**: `Selector.Validate()` checks the tier only; the cross-check that every filter name exists in the chosen tier is a method on the set.
- **Rationale**: the value type cannot know which stores are configured, and the spec's requirement that a caller can predict what a request covers is only enforceable where the store list is visible.
- **Rejected**: passing the available names into the value type's validator, which gives the type a dependency on its caller's state for no gain.

### The docs repo's `repo.yaml` correction is listed under a `spektacular` phase (phases)
- **Decision**: Phase 2.4 is labelled against the `spektacular` repo while editing one file inside the `docs` repo's footprint, with the cross-repo edit called out explicitly in the technical detail.
- **Rationale**: the phase's subject is this project's own configuration becoming valid again, which is one unit of work; splitting a two-line edit across two phases so each has a single repo label would obscure that both files must change together or the tool stays broken.
- **Rejected**: two separate phases, one per repo, which reads as two independent changes when neither is useful alone.

### The existing `Tier` type is renamed `CategoryTier` (data structures)
- **Decision**: rename `knowledge.Tier` (a category's retrieval tier, `always-applied` / `looked-up`) to `CategoryTier`, freeing the unqualified name for the addressing tier this spec introduces. The `Category.Tier` field and its `tier` JSON key are untouched, so `knowledge categories` publishes exactly what it publishes today.
- **Rationale**: the spec fixes `tier` as the name of the addressing field, so the collision has to be resolved somewhere, and it is better resolved in Go identifiers than in published output. The addressing tier is the concept every knowledge command now carries, which makes it the one that should own the short name.
- **Rejected**: naming the new type `StoreTier`, which leaves every call site reading `StoreTier` while the command line and the spec both say `tier`; and renaming either JSON field, which changes published output the spec never asked to change.

### Documentation qualifies the older meaning as "retrieval tier" (data structures)
- **Decision**: wherever a document discusses both axes, a category's retrieval tier is written as "retrieval tier" in full and "tier" unqualified means the addressing tier.
- **Rationale**: the knowledge base reference page introduces both on one page, and the spec's vocabulary is fixed, so the older meaning is the one that takes the qualifier. Prose can disambiguate without any wire change.
- **Rejected**: leaving both unqualified, which puts two meanings of one word on a single page; and renaming the category concept, which is beyond this spec and would ripple into the category README text every scaffolded repository already carries on disk.

## Rehydration cues

- `go run . repo list` — the only authority on where each repo's code lives:
  `spektacular` at `/home/nicj/code/github.com/jumppad-labs/spektacular`, `docs` at
  `/home/nicj/code/github.com/jumppad-labs/spektacular-website`.
- `go run . knowledge sources` — shows both stores reporting `scope: project`, which is
  the collision in one line.
- Reproduce the defect:
  `go run . knowledge read --data '{"scope":"project","path":"conventions/mdx-authoring.md"}'`
  fails with `not found` while the same path exists in the `docs` store.
- `go run . knowledge always-applied --repo docs` vs `--repo spektacular` — shows repo
  narrowing working but results carrying no store identity.
- Re-read in this order: `internal/knowledge/set.go` (whole file, 322 lines) →
  `cmd/knowledge.go` (whole file, 407 lines) → `internal/config/config.go:98-131,255-300,456-495`
  → `internal/config/repo.go:32-114,162-177`.
- `go run . knowledge always-applied --repo spektacular --repo docs` — loads both repos'
  conventions, including `error-messages-must-suggest-remediation` and
  `tests-must-pass-for-done` (CLI repo) and `no-em-dashes`, `mdx-authoring`,
  `plan-content-pages` (docs repo), all of which bind this work.
- Baseline: `go test ./...` in the `spektacular` root passes on branch
  `f-project-repos` before any change.

