---
created_date: "2026-09-03"
status: completed
closed_date: "2026-09-03"
---

# Context: 000047_repo-scoped-knowledge-addressing

## Current State Analysis

### The defect, reproduced

The knowledge layer resolves a request to a store by scanning its configured sources for the
first whose scope label matches, and returning it: `internal/knowledge/set.go:288`
(`byScope`). Both `Set.Read` (`set.go:170`) and `Set.Write` (`set.go:180`) route through it.
This project registers two repositories and both declare a source with scope `project`, so
every `project`-scoped read and write resolves to whichever repository the registry lists
first, and the second repository's store is unreachable.

Reproduced on branch `f-project-repos` at `0c0cb4f`:

```
$ go run . knowledge read --data '{"scope":"project","path":"conventions/mdx-authoring.md"}'
{"error": true, "code": "internal_error", "message": "not found", "next_action": ""}
```

That entry exists in the `docs` repository's store. The same command for
`conventions/tests-must-pass-for-done.md`, which exists only in the `spektacular` store,
succeeds. The failure also violates `conventions/error-messages-must-suggest-remediation.md`:
it surfaces as `internal_error` with an empty `next_action`.

`cmd/knowledge_test.go:637` asserts this behaviour as intended
(`TestKnowledgeReadWrite_DuplicateScopeResolvesToFirstRegistryRepo`).

### What already exists to build on

- `scopedStore` (`internal/knowledge/set.go:20-26`) already carries `repo`, and `SourceInfo`
  (`:62-67`) already exposes it, so attribution data is present but unused for resolution.
- `AlwaysAppliedEntries` (`:217`) already narrows by a set of repository names, and it works:
  `go run . knowledge always-applied --repo docs` returns only the docs conventions. But every
  entry comes back tagged `scope: project` with no store identity, and `readCategories`
  (`:248-257`) exempts any source with no repository attribution from the filter at line 255.
- `aggregateKnowledgeSources` (`cmd/knowledge.go:180-212`) stamps `src.Repo = e.Name` on each
  repository-declared source at line 203. It is the natural place to stamp tier and name.
- `Set.Search` (`:107-147`) already stamps `hit.Category` after merging, at line 136, which is
  the seam tier and name follow.
- `rejectLegacyRepoAddress` (`internal/config/config.go:275-300`) is the established pattern
  for refusing a removed configuration key: raw re-parse, then an `output.NewError` carrying
  the file, what was found, and what to write instead. `RepoSourceConfig.UnmarshalYAML`
  (`internal/config/repo.go:162-177`) is a second, shape-level precedent.
- `ChangelogConfig` (`internal/config/config.go:98-101`) is the single provider block shape a
  repository's knowledge declaration is being collapsed onto; it already sits beside it in the
  same file (`internal/config/repo.go:37-38`).

### The word "tier" is already taken

`internal/knowledge/category.go:13` declares `type Tier string` with `TierAlwaysApplied` and
`TierLookedUp`, introduced by plan `000028_knowledge-base-categories-tiers-and-dedup`. It
means a category's **retrieval** tier and is published by `knowledge categories`, printed into
every scaffolded category README, and described in `README.md:105-108,128` and on the
documentation site. The addressing tier this spec introduces is a different axis. The two
`tier` JSON fields sit on different objects and never share an envelope, so neither published
field changes; the Go type is renamed `CategoryTier` and prose qualifies the older meaning as
"retrieval tier". This is the single most likely thing for an implementer to trip over.

### Live configuration state

Both registered repositories' footprints are in the superseded form:

- `.spektacular/repo.yaml:11-16` and `spektacular-website/.spektacular/repo.yaml:11-16` each
  declare `knowledge.sources[0]` with `scope: project`.
- `.spektacular/config.yaml` declares no `knowledge:` block at all, so this project has zero
  project-tier stores and `--tier project` will legitimately return nothing here.

From the moment the rejection in Phase 2.3 lands, every knowledge command in this working tree
fails until Phase 2.4 corrects both files.

### Baseline

`go test ./...` passes in full across all 20 packages on this branch before any change. The
documentation site's dependencies are installed locally (node v22.18.0); its build is
`npm run build` and its typecheck is `npx astro check`, the latter not wired into CI.

## Per-Phase Technical Notes

### Phase 1.1: Addressing vocabulary and single-store resolution

**File changes**

- `internal/knowledge/category.go:13,20,24,43,57-92,115` — rename the existing `Tier` type to
  `CategoryTier` and its constants to `CategoryTierAlwaysApplied` / `CategoryTierLookedUp`. The
  `Category.Tier` **field** and its `tier` JSON key are unchanged, so `knowledge categories`
  output stays byte-identical; assert that in `internal/knowledge/category_test.go` and
  `cmd/knowledge_test.go:279-320`. `internal/knowledge/set.go:161-167` (`alwaysAppliedSet`) and
  `set.go:385` (`TestRetier_...`) follow the rename. Do this rename first, in its own commit, so
  the new type below lands on a free name.
- `internal/knowledge/address.go` (new) — `type Tier string` with `TierProject`, `TierRepo`,
  `TierAll`; `Address{Tier, Name}` and `Selector{Tier, Filter}`; `Address.Validate()` and
  `Selector.Validate()`. Validation errors are `output.NewError(...)` values so they survive the
  `%w` wrap and reach `cmd/root.go:252`'s `errors.As`. New codes: `knowledge_tier_required`,
  `knowledge_tier_invalid`, `knowledge_name_required`, `knowledge_store_unknown`. Each
  `WithNextAction` embeds the comma-separated names available in the tier concerned and a
  corrected `--data` example, per `conventions/error-messages-must-suggest-remediation.md`.
- `internal/knowledge/set.go:20-26` — `scopedStore` drops `scope` and `repo`, gains
  `tier Tier` and `name string`.
- `internal/knowledge/set.go:288-295` — delete `byScope`. Replace with
  `func (s *Set) resolve(addr Address) (scopedStore, error)`, plus
  `func (s *Set) namesIn(tier Tier) []string` used to build every refusal's next action.
- `internal/knowledge/set.go:170-186` — `Read`/`Write` take an `Address`; both call `resolve`.
- `internal/knowledge/set.go:74-100` — `NewSet` reads `src.Tier`/`src.Name` (stamped by the
  aggregator in Phase 1.3) instead of `src.Scope`/`src.Repo`; the unreachable-source error at
  line 86 names tier and store rather than scope. `store.NewSourceStore(location, ...)` keeps a
  label parameter; pass `string(tier)+":"+name` so existing store-level diagnostics stay useful.
- `internal/knowledge/set_test.go` — `twoScopeSet` (line 25) becomes a two-tier fixture; the
  criterion comments are rewritten against the new acceptance criteria. Add cases for each
  refusal, for the single-store project, and for two repos holding the same path.

**Complexity**: Medium
**Token estimate**: ~35k tokens
**Agent strategy**: Single agent, sequential execution. The new file and the reshaped methods
are one tightly coupled edit; splitting it produces a package that does not compile between
agents.

### Phase 1.2: Narrowing honoured uniformly across every retrieval path

**File changes**

- `internal/knowledge/address.go` — add `func (sel Selector) covers(src scopedStore) bool`: the
  store's tier must match unless the selector's tier is `TierAll`, and when `Filter` is non-empty
  the store's name must appear in it. No branch on whether a store is repo-declared.
- `internal/knowledge/set.go:248-257` — `readCategories` drops its `repos []string` parameter and
  the exemption at line 255 (`src.repo != ""`), taking a `Selector` and calling `covers`.
- `internal/knowledge/set.go:217-219` — `AlwaysAppliedEntries(repos ...string)` becomes
  `AlwaysAppliedEntries(sel Selector)`.
- `internal/knowledge/set.go:228-238` — `Conventions()` gains the same `Selector` parameter.
- `internal/knowledge/set.go:107-121` — `Search` gains a `Selector` and skips non-covered sources
  before querying them, so an excluded store is never read at all.
- `internal/knowledge/set.go:191-203` — `List` gains the same.
- `internal/knowledge/address.go` — `Selector.Validate` cross-checks `Filter` against
  `namesIn(tier)`, so an unreachable name errors rather than returning nothing; this needs the
  set, so it is `func (s *Set) validateSelector(sel Selector) error` on the set rather than on
  the value.
- `internal/knowledge/set_test.go:494-517` — the two existing repo-filter tests are rewritten as
  the tier-by-filter matrix described in the testing approach, with a hand-written expected set
  per cell.

**Complexity**: Medium
**Token estimate**: ~30k tokens
**Agent strategy**: Single agent, sequential. Every retrieval path shares one predicate; the
value of the phase is that they change together.

### Phase 1.3: Every result reports the tier and store it came from

**File changes**

- `internal/store/store.go:25-33` — `Hit.Scope` is replaced by `Tier` and `Name`, both documented
  as left empty by the store and populated by the knowledge layer, exactly as `Category` on line 32
  already is.
- `internal/store/search.go:90` — stop setting `Scope` from the file store's label.
- `internal/store/store.go:61-77` — `FileStore.scope` and the `Scope()` accessor become `label`;
  `NewFileStore(root, label)` and `NewSourceStore(root, label)`
  (`internal/store/ignore.go:76`) keep their signatures, so `cmd/storefile.go:95,133` is untouched.
- `internal/knowledge/set.go:134-145` — `Search` stamps `hit.Tier` and `hit.Name` alongside the
  existing `hit.Category` stamp on line 136.
- `internal/knowledge/set.go:34-67` — `Entry`, `Convention`, `AlwaysAppliedEntry` each swap
  `Scope` for `Tier` + `Name`; `SourceInfo` swaps `Scope` and drops `Repo` for `Tier` + `Name`.
- `internal/knowledge/set.go:199,235,271,282` — the four construction sites updated.
- `cmd/knowledge.go:180-212` — `aggregateKnowledgeSources` stamps `src.Tier = TierRepo` and
  `src.Name = e.Name` for each repo-declared source (replacing `src.Repo = e.Name` on line 203),
  and `src.Tier = TierProject` with `src.Name` already carrying the declared name for the project's
  own sources appended on line 211. Add a per-tier uniqueness check here, where both tiers are
  visible; repo names are already unique via `validateRepos` (`internal/config/config.go:346`), so
  in practice this catches duplicate shared-store names and is the second line of defence for them.
- `internal/config/config.go:115-123` — `SourceConfig` gains `Tier Tier \`yaml:"-"\`` beside the
  existing non-serialized `Repo`, which is removed once nothing reads it.
- `cmd/knowledge_test.go:16-56` — the five mirror structs swap `Scope` for `Tier` + `Name`;
  `knowledgeSource` drops `Repo`. These are hand-maintained independent oracles and must be edited
  by hand, never generated from the production types.
- `cmd/knowledge_test.go:616-635` — `TestKnowledgeSources_AttributesEachSourceToItsRepo` is
  rewritten as a tier-and-name assertion.

**Complexity**: Medium
**Token estimate**: ~30k tokens
**Agent strategy**: 2 parallel agents: one on `internal/store` + `internal/knowledge` envelopes,
one on the `cmd` aggregator and its test mirrors. Integrate sequentially; the package will not
compile until both land.

### Phase 1.4: Command surface and published interface

**File changes**

- `cmd/spec.go:43-46` — `commandSchema` gains `Flags map[string]*schemaProp \`json:"flags,omitempty"\``.
  Nothing else changes, so every other family's `--schema` output stays byte-identical; assert that
  explicitly in `cmd/spec_test.go` or the nearest schema test.
- `cmd/knowledge.go:357-378` — `knowledgeScopePathInput` becomes `knowledgeAddressInput{Tier, Name,
  Path}`; `knowledgeScopePathData` validates all three non-empty and rejects `all`, returning the
  same structured errors Phase 1.1 defines. The `--data` example strings on lines 368, 375, 401, 402
  are updated.
- `cmd/knowledge.go:68` — `knowledgeAlwaysAppliedRepos` is replaced by package-level `knowledgeTier`
  and `knowledgeFilter` vars shared by search, list, conventions, and always-applied.
- `cmd/knowledge.go:398-406` — register `--tier` (default `all`) and repeatable `--filter` on those
  four subcommands; delete the `--repo` flag on line 404.
- `cmd/knowledge.go:76-146` — all eight schemas updated: hit and entry properties gain `tier`/`name`
  and lose `scope`; the read/write input schema requires `tier`, `name`, `path`; search, list,
  conventions and always-applied gain a `Flags` block describing `tier` (with an enum of the three
  values) and `filter` (an array of strings).
- `cmd/knowledge.go:214-355` — the eight run functions build an `Address` or a `Selector` and pass
  it through; `runKnowledgeRead`/`runKnowledgeWrite` echo `tier` and `name` in their result maps
  (lines 250-254, 296).
- `cmd/knowledge_test.go:60-71` — `resetKnowledgeFlags` clears the two new flags and the two new
  package vars; a missed entry here leaks state between subtests.
- `cmd/knowledge_test.go:637-663` — `TestKnowledgeReadWrite_DuplicateScopeResolvesToFirstRegistryRepo`
  is renamed and inverted: the same ambiguous request must now be refused and record nothing, and
  the addressed forms must reach each repo's own store.
- `cmd/knowledge_test.go:321-364,784-800` — the three schema tests updated for the new fields and
  the flags block.
- `cmd/root_test.go` — the cross-command tables that invoke `knowledge` are checked so the new
  refusal shape stays consistent with every other family's error envelope.

**Complexity**: High
**Token estimate**: ~45k tokens
**Agent strategy**: Parallel analysis, sequential integration. One agent maps every schema and flag
site, one maps every test assertion that names `scope`; a single agent then applies the edits, since
`cmd/knowledge.go` is one file and concurrent edits to it will conflict.

### Phase 2.1: A repository declares exactly one knowledge store

**File changes**

- `internal/config/config.go:108-111` — introduce
  `type RepoKnowledgeConfig struct { Provider string \`yaml:"provider"\`; Config FileKnowledgeConfig \`yaml:"config"\` }`
  beside the existing `KnowledgeConfig`, deliberately mirroring `ChangelogConfig` on lines 98-101.
  Add `RepoKnowledgeConfig.Validate()` requiring a supported provider and a non-empty location.
- `internal/config/repo.go:37` — `RepoConfig.Knowledge` changes type to `RepoKnowledgeConfig`.
- `internal/config/repo.go:43-63` — `NewDefaultRepoConfig` emits the single block with
  `DefaultRepoKnowledgeLocation`; `DefaultKnowledgeScope` (`config.go:58-59`) loses its only remaining
  use here and is removed once Phase 2.2 and 2.3 no longer reference it.
- `internal/config/repo.go:99-114` — `WithDefaults(repoRoot)` fills the single block when the provider
  is empty, joining `repoRoot` to the default location as it does today.
- `internal/config/repo.go:87` — `RepoConfig.Validate` calls the new validator.
- `internal/project/init.go:91-101` — the loop and the `src.Scope != config.DefaultKnowledgeScope`
  filter collapse to reading `repoCfg.WithDefaults(spektacularDir).Knowledge.Config.Location`, still
  joined to `projectPath` when relative. `knowledgeRoots` becomes a single value.
- `internal/repo/footprint.go:56-63` — the same collapse for `EnsureFootprint`; the category-directory
  and README scaffolding beneath it (lines 64-85) is unchanged.
- `internal/config/repo_test.go:100-134,210-235` — the struct and raw-YAML round-trip fixtures move to
  the new shape.
- `cmd/repo_test.go:156-179` — the scaffolded footprint file listing is a directory-layout assertion
  and should still pass unchanged; confirm rather than edit.

**Complexity**: Medium
**Token estimate**: ~30k tokens
**Agent strategy**: 2 parallel agents: one on `internal/config`, one on the two scaffolding call
sites. Integrate sequentially.

### Phase 2.2: Shared stores are declared by the project, each under its own name

**File changes**

- `internal/config/config.go:115-123` — `SourceConfig.Scope` becomes `Name string \`yaml:"name"\``.
- `internal/config/config.go:456-473` — `KnowledgeConfig.Validate` checks non-empty `Name` and
  uniqueness of `Name` within the list; its messages name `knowledge.sources[i].name`.
- `internal/config/config.go:480-495` — `KnowledgeConfig.WithDefaults` is deleted. It exists to
  synthesise a default project-owned source at `.spektacular/knowledge`, which is now a repo-tier
  concern owned by `RepoConfig.WithDefaults`; leaving it would silently manufacture an unnamed shared
  store. `internal/knowledge/set.go:75` drops the `WithDefaults` call with it.
- `internal/config/config.go:60-62` — `DefaultKnowledgeLocation` loses its last consumer and is
  removed.
- `internal/config/config_test.go:204-205,317,331` — the knowledge validation assertions move to the
  new key and the new uniqueness rule.
- `cmd/knowledge_test.go:73-116` — `twoScopeProject` becomes a two-tier fixture: the colocated repo's
  own store plus a project-declared shared store named `team`, with the `config.yaml` still written as
  a raw YAML string so the wire format stays pinned.
- `internal/knowledge/set_test.go:258-269` — `TestNewSet_SynthesisesDefaultProjectSource` asserts the
  behaviour being deleted: that a bare default config yields one synthesised `project` source. It is
  removed with the function, and replaced by a test asserting that a project declaring no shared
  stores produces an empty project tier and that requests restricted to that tier return nothing
  rather than failing.

**Complexity**: Low
**Token estimate**: ~20k tokens
**Agent strategy**: Single agent, sequential. Small and localized; the only subtlety is that deleting
`KnowledgeConfig.WithDefaults` is deliberate, not an oversight.

### Phase 2.3: Configuration in the superseded form is rejected with an actionable error

**File changes**

- `internal/config/config.go:255-267` — `ParseYAMLFile` gains a call to a new
  `rejectLegacyKnowledgeScope(expanded, path)` beside the existing `rejectLegacyRepoAddress` call on
  line 261. It re-parses into `struct{ Knowledge struct{ Sources []map[string]any } }` and fails when
  any entry carries a `scope` key, returning `output.NewError("config_invalid", ...)` naming the file,
  the index, the value found, and the required `name:` key, `WithResource(path)` and a `WithNextAction`
  showing the corrected entry. Modelled directly on `rejectLegacyRepoAddress` (lines 275-300).
- `internal/config/repo.go:65-85` — `RepoConfigFromYAMLFile` gains the mirror guard,
  `rejectLegacySourcesBlock(expanded, path)`: it re-parses into
  `struct{ Knowledge map[string]any }` and fails when the knowledge mapping carries a `sources` key or
  a `scope` key, naming the file and printing the required single provider block inline in the next
  action. The `%w` wrap on line 80 preserves the typed error for `errors.As`.
- `internal/config/repo.go:162-177` — no change; `RepoSourceConfig.UnmarshalYAML` remains the
  shape-rejection precedent this follows.
- `cmd/version.go:182-236` — `scanProjectMetadata` builds on `config.NewDefaultRepoConfig()`, so it
  emits the new shape automatically; confirm and update `cmd/version_test.go`'s
  `TestExecuteMigration`/`TestScanProjectMetadata` assertions on the generated `repo.yaml`.
- `cmd/version.go:151-177` — `detectMigrationNeeded` is a pure file-presence check and needs no
  change; note in the phase that a legacy single-file project is diverted to the migration prompt
  before any knowledge operation parses its configuration.
- New tests in `internal/config` for both guards: the error names the file and the required shape; a
  second load fails identically; the file on disk is byte-identical afterwards; and the corrected form
  loads cleanly.
- New test in `cmd` asserting that a project whose registered repository is in the superseded form
  fails every knowledge subcommand, not just one.

**Complexity**: Medium
**Token estimate**: ~30k tokens
**Agent strategy**: 2 parallel agents, one per guard, since the two files and their tests are
independent. Integrate sequentially.

### Phase 2.4: This project's own configuration is corrected

**File changes**

- `.spektacular/repo.yaml:11-16` — replace the `knowledge.sources` list with:
  ```yaml
  knowledge:
      provider: file
      config:
          location: knowledge
  ```
- `spektacular-website/.spektacular/repo.yaml:11-16` — the identical replacement. Note this file is in
  the `docs` repo's footprint, so the edit lands in that repository even though the phase is otherwise
  about this one.
- `.spektacular/config.yaml` — no change needed; the project declares no `knowledge:` block today, so
  the project tier is legitimately empty here.
- Verification, not file changes: `go run . knowledge sources` reports two repo-tier stores named
  `spektacular` and `docs`; `go run . knowledge read --data '{"tier":"repo","name":"docs","path":"conventions/mdx-authoring.md"}'`
  returns content where the pre-change equivalent returned `not found`.

**Complexity**: Low
**Token estimate**: ~8k tokens
**Agent strategy**: Single agent, sequential. Two small edits, but they gate every subsequent
command in the working tree, so run the verification immediately after.

### Phase 3.1: The knowledge skill works in tiers and shows where an entry will land

**File changes**

- `templates/skills/workflows/spek-knowledge/SKILL.md:31` — the search result field list gains `tier`
  and `name` and loses `scope`.
- `:37` — the read invocation becomes
  `{{command}} knowledge read --data '{"tier":"<tier>","name":"<name>","path":"<path>"}'`.
- `:39` — delete the `project → team → global` specificity chain; replace with a sentence stating that
  a repository's store answers what is true of that repository's code and the project's shared stores
  answer what spans repositories, with no precedence between them.
- `:44` — the citation shape becomes tier, store name, and path.
- `:52` — `knowledge sources` is described as enumerating the configured stores by tier and name, and
  as the authoritative list of writable destinations and of valid narrowing names.
- `:55-59` — the contribute flow chooses a tier and a store name; the propose-then-confirm checkpoint
  at step 5 must show tier, store name, and path before asking for approval; the write invocation
  becomes `--data '{"tier":"<tier>","name":"<name>","path":"<category>/<slug>.md"}'`.
- `:65-76` — the update flow gains the same three-part address, and the line "The scope and path must
  match the original" becomes tier, name, and path.
- `:80` — the decline path is unchanged apart from vocabulary.
- `.claude/skills/spek-knowledge/SKILL.md` and `.bob/skills/spek-knowledge/SKILL.md` — regenerated,
  not hand-edited. Both are tracked in git and are produced by
  `internal/agent/skills.go:40-65` rendering `{{command}}`; running `go run . init claude` and
  `go run . init bob`, or an equivalent regeneration, is what keeps them in step. Verify by diffing
  each against the template with the placeholder substituted.

**Complexity**: Medium
**Token estimate**: ~25k tokens
**Agent strategy**: Single agent, sequential. One document plus a mechanical regeneration; the risk is
hand-editing a generated copy, which a diff against the template catches.

### Phase 3.2: Workflow steps and agent guidance use the new vocabulary

**File changes**

- `templates/steps/plan/02-discovery.md:14` — `knowledge always-applied --repo <name>` becomes
  `knowledge always-applied --tier repo --filter <name>` (repeat `--filter`), and the sentence
  "project-owned sources always load regardless of `--repo`" is deleted, since it is now false. Add
  that a plan needing shared knowledge asks for `--tier all` and names the shared stores it wants.
- `:16` — the search result description gains tier and store; the read example takes the three-part
  address.
- `:29` — unchanged apart from vocabulary.
- `:58` — "run `knowledge sources` to see the available scopes, then propose a target scope" becomes
  tier and store name, matching the skill's checkpoint.
- `templates/steps/plan/03-architecture.md:45` — the always-applied reference gains the narrowing.
- `templates/steps/plan/18-walkthrough.md:22` and
  `templates/steps/implement/07-update_changelog.md:51` — "owns scope selection" becomes "owns tier
  and store selection".
- `templates/skills/skill_spawn-planning-agents.md:22` — "tagged by scope (`project`, `team`,
  `global`)" becomes tagged by tier and store name.
- `templates/agents/memory-context.md:10-12` and `templates/agents/knowledge-trigger.md:24-25` —
  "handles scope selection" becomes "handles tier and store selection". No new managed section is
  added, per the spec's constraint.
- `templates/*_test.go` — add a content-assertion test in the style of
  `templates/repo_source_test.go:98` (`TestRepoListConsumersSendTheAgentToRepoList`) asserting that no
  template still instructs an agent to address knowledge by scope, and that the discovery step passes
  a tier and filter.

**Complexity**: Low
**Token estimate**: ~20k tokens
**Agent strategy**: Single agent, sequential. Eight small text edits plus one new test; parallelizing
risks inconsistent wording across files that must agree.

### Phase 3.3: The documentation site explains the two tiers

**File changes**

- `docs:src/pages/knowledge-base.mdx:141-202` — the `Configuration` `Section` is rewritten to the
  content outline in plan.md. It keeps `heading="Configuration"` and its bare `surface` prop, so the
  page's false/true/false/true/false alternation (`conventions/alternate-section-background.md`) is
  preserved; the subtitle stays in its `<Fragment slot="sub">` and the body stays inside
  `<Prose nested>`. YAML and bash samples are fenced markdown blocks, never JSX with a code prop
  (`conventions/mdx-authoring.md` Rule 4).
- `docs:src/pages/knowledge-base.mdx:99-107` — the `Creating` write example becomes the fully
  addressed form.
- `docs:src/pages/knowledge-base.mdx:249-254` — the `Sources are layered, most-specific first` bullet
  is replaced by one explaining that the tiers answer different questions.
- `docs:src/pages/knowledge-base.mdx:117-125` — the searching paragraph gains a sentence on `--tier`
  and `--filter`.
- `docs:src/pages/knowledge-base.mdx:44-86` — the `The six categories` section already uses the word
  *tier* for a category's retrieval tier. Wherever that meaning appears on this page it becomes
  **retrieval tier**, explicitly, so the two axes are not conflated now that the Configuration
  section introduces addressing tiers. Add one short sentence in the Configuration section naming
  the distinction: a category's retrieval tier says when an entry is loaded, a store's tier says
  which knowledge it belongs to.
- Verification: `npm run build` and `npx astro check` from
  `/home/nicj/code/github.com/jumppad-labs/spektacular-website`, both clean, plus
  `grep -nE "<div|<section|class=" src/pages/*.mdx` returning nothing
  (`conventions/mdx-authoring.md` Rule 1). No em dashes in any authored line
  (`conventions/no-em-dashes.md`).

**Complexity**: Medium
**Token estimate**: ~25k tokens
**Agent strategy**: Single agent, sequential. One page, and the prose must read as one voice.

### Phase 3.4: The rest of the documentation site is corrected

**File changes**

- `docs:src/pages/configuration.mdx:57-62` — the `config.yaml` example's knowledge block uses `name:`.
- `docs:src/pages/configuration.mdx:165-179` — the project-side `ConfigKey name="knowledge"` body is
  replaced by the content example in plan.md; `defaultValue` stays "none".
- `docs:src/pages/configuration.mdx:233-255` — the `repo.yaml` example's knowledge block becomes the
  single provider block.
- `docs:src/pages/configuration.mdx:310-316` — the repository-side `ConfigKey name="knowledge"` body
  is replaced by the second content example; `defaultValue` becomes a file store at `knowledge`.
- `docs:src/pages/configuration.mdx:28` — the intro sentence describing "two knowledge sources" is
  reworded for the two tiers.
- `docs:src/pages/projects.mdx:67-72` — the `repo.yaml` knowledge block becomes the single provider
  block. The directory trees at 129 and 148 need no change.
- `docs:src/content/tutorials/getting-started.mdx:746-772` — the `Multi-source knowledge` section is
  rewritten for the two tiers; its example must stop declaring a `project` scope in `config.yaml`,
  which contradicts `configuration.mdx` today.
- `docs:src/content/tutorials/getting-started.mdx:606-613` — the "several places to put context" list
  is reworded from project/team/global levels to the two tiers.
- `docs:src/pages/index.mdx:65-68` — the feature card's "tags each result by scope" becomes tier and
  store.
- `docs:src/pages/extending.mdx:59-60,78-79` — the `Hit` struct listing swaps `Scope` for `Tier` and
  `Name`, and the surrounding sentence is reworded.
- `docs:src/pages/debugging.mdx:43` — the sample log line uses `knowledge sources`, which is
  unchanged; confirm rather than edit.
- Same three verifications as Phase 3.3.

**Complexity**: Medium
**Token estimate**: ~30k tokens
**Agent strategy**: 2-3 parallel agents, split by page since the files are independent: one on
`configuration.mdx`, one on `getting-started.mdx` + `projects.mdx`, one on `index.mdx` +
`extending.mdx`. Run the build and typecheck once after integration.

### Phase 3.5: The command repository's own documentation is corrected

**File changes**

- `README.md:110-128` — the `Scopes, search, and de-duplication` section is retitled and rewritten per
  the content example in plan.md, including the `Hit` field listing.
- `README.md:132-138` — the CLI subcommand list describes the tier and narrowing options.
- `README.md:180-190` — the `config.yaml` example's knowledge block uses `name:`.
- `README.md:200-212` — the `repo.yaml` example's knowledge block becomes the single provider block.
- `README.md:95,112,218` — the remaining sentences describing scopes and aggregation order.
- `docs/knowledge-base.md:84,96,141-157,167,173-185` — the same corrections, including deleting the
  `Layered source precedence` section at 143-157 and rewriting every `--data '{"scope":...}'` row of
  the command reference table at 176-185. The document's existing use of *tier* for a category's
  retrieval tier is qualified as **retrieval tier** throughout, for the same reason as the site.
- `README.md:100-108,128` — the same qualification: `README.md:105-108` describes the always-applied
  and looked-up tiers, and line 128 refers to "the retrieval tiers"; both stay, written so they
  cannot be read as the addressing tier introduced above them.
- `CHANGELOG.md` — a new top entry for `000047_repo-scoped-knowledge-addressing` opening with the
  breaking-change text in plan.md's content example, placed above the `000045` entry.
- `cmd/docs_test.go:29-70` — the readme and changelog assertions gain checks for the tier vocabulary
  and the breaking-change note, following the existing `require.Contains` style.

**Complexity**: Medium
**Token estimate**: ~25k tokens
**Agent strategy**: 2 parallel agents: one on `README.md` + `CHANGELOG.md` + the docs test, one on
`docs/knowledge-base.md`. Integrate sequentially.

### Phase 4.1: The site's layout knowledge is recorded through the delivered behaviour

**File changes**

- No source file is edited by hand. Two entries are created through the `spek-knowledge` skill, which
  stages each body under `.spektacular/tmp/<slug>.md` and writes it with
  `go run . knowledge write --data '{"tier":"repo","name":"docs","path":"<category>/<slug>.md"}' --file <staged>`:
  - `conventions/site-layout.md` — the five rules of the site's layout system: one frame, one flow
    width, one heading scale, one body rhythm, one component per job.
  - `decisions/frame-width-flow.md` — why body text is not capped at a reading measure: a measure needs
    a left nav and a right table of contents to box it in, the site has neither, and a capped column
    misaligned against the full-width code-and-keys grid in the spec format section of
    `/how-it-works/`.
- The resulting files land under `spektacular-website/.spektacular/knowledge/`, which is the `docs`
  repo's footprint, and are committed there.
- Verification: reading each back at `{"tier":"repo","name":"docs",...}` returns the content;
  reading either at `{"tier":"repo","name":"spektacular",...}` finds nothing; `knowledge search` for a
  distinctive phrase from each reports tier `repo` and name `docs`.
- The skill's propose-then-confirm checkpoint must show the tier and store before each write, which is
  also the acceptance evidence for the spec's approval requirement.

**Complexity**: Low
**Token estimate**: ~15k tokens
**Agent strategy**: Single agent, sequential, driven through the skill rather than by raw commands, as
this repository's own rules require.

## Testing Strategy

### Shape of the suite

The work is tested at three existing levels, all inside the command repository's Go test
suite, following the conventions already in place there: table-free tests using the
project's assertion library, temporary project trees built per test, and expectations
written out by hand rather than derived from the code under test.

- **Unit tests on the knowledge layer** carry the most coverage, because the addressing
  vocabulary and the resolution rule are where the defect lives and where every caller's
  behaviour is decided. They build small multi-store sets directly and assert resolution,
  refusal, and narrowing without going near the command line.
- **Command-surface tests** exercise the real command tree end to end against a temporary
  project on disk, so the JSON envelope a caller actually receives is what is asserted,
  including the shape of a refusal. These are where the acceptance criteria about writes
  landing in the right store, results reporting their origin, and a search hit reading back
  exactly are pinned.
- **Configuration tests** cover the two new shapes and the two rejections, including that a
  rejected file is left byte-identical on disk and fails the same way on a second run.
- **Template content tests** extend an existing suite that asserts what the embedded
  workflow steps and skills say, pinning that the knowledge skill and the planning steps
  describe the new addressing rather than the old vocabulary.
- **Documentation tests** already assert the command repository's readme and changelog
  content; they gain assertions for the addressing model and the breaking-change note.

### Load-bearing assertions, in plain language

- A write that names neither a tier nor a store, or only one of them, records nothing, and a
  read for that location afterwards finds nothing. This holds in a project with several
  stores **and** in a project with exactly one, which is the case most likely to tempt a
  convenience fallback.
- A refusal names the stores available in the tier concerned, and those names are the same
  ones the store-enumeration command reports.
- A write addressed to a repository's own store is readable back at that address and is not
  readable under any other name or in the other tier. Two repositories holding entries at
  the same location stay distinct in both directions.
- A repository whose configuration declares more than one store, or names the one it
  declares, is rejected, and no knowledge operation against that project succeeds until it
  is corrected.
- Every listing entry, every search hit, and every read result reports a tier and a store
  name. Taking any single hit and issuing a read using only what that hit carries returns
  the entry the hit excerpted, including when an identically-named entry exists elsewhere.
- Choosing a tier returns results only from that tier; narrowing to a name returns results
  only from that store; omitting the narrowing covers every store the tier reaches. The
  always-applied load obeys the same rule, with no store implicitly included.
- A stale configuration fails on every knowledge operation with an error naming the file,
  what was found, and what is required; correcting the file resolves it; and a repository
  scaffolded by the tool is accepted with no correction at all.
- A caller reading only the published machine-readable interface can issue a fully addressed
  write successfully, and that interface advertises the tier, name, and narrowing fields.

### Regression the change must invert

One existing command-surface test asserts today's defective behaviour directly: that a read
against an ambiguous scope resolves to the first registered repository's copy, and that a
write lands in that repository's store rather than the intended one. It is rewritten to
assert the opposite, that the same under-specified request is now refused and records
nothing, rather than deleted, so the suite keeps a test standing at the exact point the
defect lived.

### Success metrics, and how each is verified

- **No knowledge entry lands in the wrong store.** *Behavioural test.* The guarantee is
  two-sided and both sides are asserted: a fully addressed write is readable at that address
  and at no other, and an under-specified write fails without recording anything anywhere.
  The second half is what makes the count of silently misplaced entries zero rather than
  merely low, so it is asserted in the single-store case as well as the multi-store case.
- **The dogfood write succeeds first time.** *Manual, captured in the implementation test
  plan.* Whether recording the two supplied entries needed a second attempt, or a fallback
  to editing files directly, is an observation about the act of delivery rather than a
  property of the code. The durable half of it is covered behaviourally by the assertion
  above; that the two entries are readable from the documentation repository's store, absent
  from every other, and reported with that store as their source is asserted as an
  acceptance check.
- **Planning loads less irrelevant knowledge.** *Behavioural test.* In a fixture with more
  than one store holding always-applied entries, the load narrowed to a single store returns
  strictly fewer entries than the same load with no narrowing, and returns exactly the
  narrowed store's entries. That is the reduction the metric describes, measured the way it
  describes it.
- **A caller can predict what a request covers.** *Behavioural test.* A matrix over the
  three tiers crossed with no narrowing, a narrowing naming one store, and a narrowing
  naming several, asserting the exact set of stores covered in each cell against a
  hand-written expectation. The point of the matrix is that no cell is a special case; a
  narrowing that does not name a store excludes it, in every tier, for every retrieval path.

### Deliberate gaps

- **The documentation site gets no new automated tests.** Its correctness gates are its
  existing build and typecheck, plus the manual markup guard its authoring conventions
  define. The spec constrains this work to introduce no new site tooling, and asserting page
  prose in a test suite would be a new mechanism for no additional confidence.
- **No test derives an expected value from the production types.** Result envelopes are
  decoded into independently-declared mirror structures and compared against hand-written
  expectations, which is the existing convention at the command surface and is what keeps a
  renamed field from silently passing.
- **The store layer gains no new tests of its own.** Its two new fields are left empty by the
  store and populated by the knowledge layer, so the knowledge layer's assertions already
  cover the only behaviour that exists; a store-level test would assert that an empty field
  is empty.
- **No test asserts the same guarantee twice through different mechanisms.** Where a
  command-surface test already pins a behaviour end to end, the corresponding unit test
  covers the resolution rule and its error text rather than repeating the outcome.


### Where each phase's tests live

- **Phases 1.1 and 1.2** — `internal/knowledge/set_test.go`, extending the existing
  `writeFile` / `twoScopeSet` helpers (`:12-53`) into a two-tier fixture, and following the
  `// Criterion N:` comment convention that ties each test to an acceptance criterion. Phase
  1.2 rewrites the two repository-filter tests at `:494-517` as the tier-by-filter matrix.
- **Phase 1.3** — split across `internal/knowledge/set_test.go` for the envelope fields and
  `cmd/knowledge_test.go` for the wire shape, where the five mirror structs at `:16-56` are
  hand-edited. They are independent oracles and must never be generated from the production
  types.
- **Phase 1.4** — `cmd/knowledge_test.go`, driven through `runKnowledge` (`:118-135`) so the
  real JSON envelope is asserted, including `resetKnowledgeFlags` (`:60-71`) gaining the two
  new flags. `TestKnowledgeReadWrite_DuplicateScopeResolvesToFirstRegistryRepo` (`:637`) is
  renamed and inverted here. The schema tests at `:321-364` and `:784-800` cover the published
  interface, and `cmd/root_test.go`'s cross-command tables are checked for envelope
  consistency.
- **Phases 2.1 and 2.2** — `internal/config/repo_test.go` (`:100-134,210-235`) and
  `internal/config/config_test.go` (`:204-205,317,331`) for the two shapes, plus
  `internal/knowledge/set_test.go:258-269` where `TestNewSet_SynthesisesDefaultProjectSource`
  is replaced by an empty-project-tier test.
- **Phase 2.3** — new tests in `internal/config` for both guards, asserting the error names the
  file and the required shape, that a second load fails identically, and that the file is
  byte-identical afterwards; plus a `cmd` test that every knowledge subcommand fails against a
  project whose repository is in the superseded form, and updates to
  `cmd/version_test.go`'s migration assertions.
- **Phase 2.4** — no test file; verified by running the commands in this working tree.
- **Phase 3.2** — a new content-assertion test in `templates`, in the style of
  `templates/repo_source_test.go:98`, asserting no template addresses knowledge by scope and
  that the discovery step passes a tier and a filter.
- **Phases 3.3 and 3.4** — no automated tests, by design. Verified by the site's build, its
  typecheck, and the manual markup guard.
- **Phase 3.5** — `cmd/docs_test.go:29-70`, extending the existing `require.Contains`
  assertions over `README.md` and `CHANGELOG.md`.
- **Phase 4.1** — no test file; verified by reading each entry back at its address, failing to
  read it elsewhere, and searching for it.

## Project References

### Repositories

`go run . repo list` is the only authority on where code lives. As registered today:

| name | root | role |
|---|---|---|
| `spektacular` | `/home/nicj/code/github.com/jumppad-labs/spektacular` | the Go CLI itself |
| `docs` | `/home/nicj/code/github.com/jumppad-labs/spektacular-website` | Astro 5 + Tailwind v4 documentation site |

Every path below without a repository prefix is in `spektacular`. Paths in the documentation
site are prefixed `docs:`.

### Requirement to repository and files

| Requirement | Repo | Principal files |
|---|---|---|
| Knowledge is addressed by tier | `spektacular` | `internal/knowledge/address.go` (new), `internal/knowledge/set.go` |
| A store is identified by name within its tier | `spektacular` | `internal/knowledge/set.go`, `cmd/knowledge.go:180-212` |
| A repository has exactly one knowledge store | `spektacular` | `internal/config/repo.go:32-114`, `internal/config/config.go:108-131` |
| Shared knowledge is declared by the project | `spektacular` | `internal/config/config.go:108-123,456-473` |
| Writing states the tier and the store | `spektacular` | `internal/knowledge/set.go:180`, `cmd/knowledge.go:276-297,357-378` |
| A refused write explains how to retry | `spektacular` | `internal/knowledge/address.go`, `internal/output/writer.go:60-84` |
| Reading addresses a single store | `spektacular` | `internal/knowledge/set.go:170`, `cmd/knowledge.go:233-255` |
| Every result reports its tier and store | `spektacular` | `internal/store/store.go:25-33`, `internal/knowledge/set.go:34-67,134-145` |
| A search result can be retrieved without disambiguation | `spektacular` | `internal/knowledge/set.go:107-147` |
| Retrieval can be narrowed to chosen stores | `spektacular` | `internal/knowledge/set.go:107,191,217,228`, `cmd/knowledge.go:398-406` |
| Narrowing is honoured uniformly | `spektacular` | `internal/knowledge/set.go:248-257` (the exemption at 255) |
| An outdated configuration fails loudly | `spektacular` | `internal/config/config.go:255-300`, `internal/config/repo.go:65-85` |
| Recording on the user's behalf shows where it will land | `spektacular` | `templates/skills/workflows/spek-knowledge/SKILL.md:50-76` |
| The published interface describes the addressing fields | `spektacular` | `cmd/spec.go:43-46`, `cmd/knowledge.go:76-146` |
| Knowledge supplied for this feature is recorded through it | `docs` | `spektacular-website/.spektacular/knowledge/` |
| The documentation explains the addressing model | `docs`, `spektacular` | `docs:src/pages/knowledge-base.mdx`, `docs:src/pages/configuration.mdx`, `README.md`, `docs/knowledge-base.md` |

### Commands used repeatedly

- `go run . repo list` — where each repository's code is.
- `go run . knowledge sources` — the configured stores; after this work, by tier and name.
- `go test ./...` from the `spektacular` root.
- `npm run build` and `npx astro check` from the documentation site root, plus
  `grep -nE "<div|<section|class=" src/pages/*.mdx` returning nothing.

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|-------------|----------------|
| Low | ~10k | Single agent, sequential |
| Medium | ~25k | 2-3 parallel agents |
| High | ~50k+ | Parallel analysis, sequential integration |

Per-phase estimates and strategies are recorded with each phase above. The total across all
fourteen phases is roughly 370k tokens, so the plan is not executable in one context window;
phases are the unit of work and each is sized to complete within one.

Two phases dominate and deserve care with context. Phase 1.4 (~45k) touches one large file
plus its test file and every published schema, so the recommended shape is parallel analysis
followed by a single writer, never concurrent edits to `cmd/knowledge.go`. Phase 3.4 (~30k)
splits cleanly by page across two or three agents because the documentation files are
independent of one another.

Three phases are deliberately small and sequential because they gate everything after them:
Phase 2.4 (~8k) restores this working tree's ability to run its own knowledge commands, and
Phase 4.1 (~15k) is driven through the knowledge skill rather than by raw commands, as this
repository's rules require.

## Migration Notes

**There is no automatic migration, by design.** The spec forbids migrating, rewriting, or
aliasing configuration written in the superseded form, and forbids an indefinite period in
which both forms are accepted. A stale file is rejected and corrected by hand.

**What an existing project must change.** Two edits, both one-time:

1. In each registered repository's `repo.yaml`, replace the `knowledge.sources` list with a
   single provider block:

   ```yaml
   knowledge:
     provider: file
     config:
       location: knowledge
   ```

2. In the project's `config.yaml`, if it declares shared stores, rename each entry's `scope:`
   key to `name:`. A project that declares no `knowledge:` block needs no change here.

Until both are done, every knowledge operation fails with an error naming the file, what was
found, and what is required. The file on disk is never touched by the tool.

**Relationship to the existing config migrator.** `detectMigrationNeeded`
(`cmd/version.go:151-177`) handles a different, older migration: the legacy single-file layout
to the split `config.yaml` + `repo.yaml` layout. It is a pure file-presence check and never
parses a `knowledge:` block, and `executeMigration` (`:242-270`) byte-copies the old file and
writes a fresh `repo.yaml` from `config.NewDefaultRepoConfig()`. Once that default emits the
new shape, the migrator produces a correct file for free. Its tests
(`TestExecuteMigration`, `TestScanProjectMetadata` in `cmd/version_test.go`) assert the
generated file's contents and must be updated in Phase 2.3.

**This project's own migration** is Phase 2.4, and it is a prerequisite for Phase 4.1.

**No knowledge entry is migrated.** Entries already written keep their location and content,
including any misplaced by the defect. Nothing relocates, re-attributes, or deletes an entry.

## Performance Considerations

Performance is not a driver of this work, and no phase is expected to change the cost of a
knowledge operation measurably. Two effects are worth recording.

**Narrowing makes fan-out cheaper, which is the point.** Today every retrieval path reads every
configured store. With a selector applied before the store is queried
(`internal/knowledge/set.go:107,191`), an excluded store is never opened at all, so a narrowed
search or always-applied load does strictly less filesystem work than an unnarrowed one. The
spec counts this as a success metric in tokens rather than milliseconds: planning scoped to one
repository stops loading every other repository's conventions and glossary in full, which is
the largest single contributor to a planning task's context.

**Resolution changes from a scan to a scan.** `byScope` walked the source list to the first
match; the replacement walks it looking for an exact tier and name pair. Both are linear in the
number of configured stores, which is bounded by the number of registered repositories plus the
project's declared shared stores, realistically single digits. Building the refusal message
walks the list once more, but only on the error path.

No caching, indexing, or concurrency is introduced, and none is warranted at this scale.

