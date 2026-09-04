# Context: 000047_repo-scoped-knowledge-addressing (IMPLEMENT workflow)

Working context for the **implement** workflow. The plan is the source of
truth for what gets built; this file holds cross-cutting learnings,
decisions, and the user's answers as implementation proceeds. Everything
below the "Plan shape" heading was carried over from the plan workflow and
is still accurate.

## Registered repos (from `go run . repo list`)

| name | root | role |
|---|---|---|
| `spektacular` | `/home/nicj/code/github.com/jumppad-labs/spektacular` | Go CLI (the tool itself) |
| `docs` | `/home/nicj/code/github.com/jumppad-labs/spektacular-website` | Astro 5 + Tailwind v4 documentation site |

All code work happens in those roots, never in a guessed directory.

## What the spec asks for (one-line recap)

Replace implicit first-match knowledge-source resolution with explicit
addressing: a **tier** (`project` | `repo` | `all`), a **`name`** identifying
one store within a tier, and a flat **`filter`** list of names narrowing
search / list / always-applied retrieval. Writes must state tier + name or be
refused with the available names. Every result reports its tier and store.
Old-form config is rejected loudly, never migrated.

## Carried forward from the spec session (still true)

- The defect lives in `internal/knowledge/set.go`: `byScope` matches on scope
  name alone and returns the **first** match; `Set.Read` and `Set.Write` both
  route through it, so in a two-repo project every `project`-scoped read and
  write silently resolves to whichever repo the registry lists first.
- `scopedStore` already carries `repo`, `SourceInfo` already exposes it, and
  `AlwaysAppliedEntries` already filters by a set of repo names. The change
  is a narrowing of existing structure, not a new concept.
- `cmd/knowledge.go` is the CLI surface; each subcommand publishes an
  input/output schema via `--schema` that must reflect the new fields.
- Dogfood acceptance: two drafted entries (`conventions/site-layout.md` and
  `decisions/frame-width-flow.md`) must be written into the **`docs`** repo's
  own store through the delivered CLI, not by editing files.

## Superseded

The spec session's "unresolved nuance" (whether scope alone may suffice for
project-owned sources) is **resolved by the spec**: the tier/name model makes
both tiers address the same way, and a write missing either is always
refused, "whether or not what was given happens to identify a single store".
Plan to the spec, not to that earlier note.

## Decisions made during planning

Recorded in full in `.spektacular/work/000047_repo-scoped-knowledge-addressing/assumptions.md`.
The load-bearing ones:

- `tier: all` is valid for search / list / always-applied only; read and write demand
  `project` or `repo` plus a name.
- The `project > team > global` precedence chain is prose-only (no code implements it)
  and is deleted rather than restated in tier terms.
- `--repo` on `always-applied` is replaced outright by `--tier` + repeatable `--filter`.
- Names are unique within a tier, not globally.
- `tier` / `filter` stay CLI flags; `commandSchema` gains an `omitempty` `flags` block so
  they are still published.
- Tier and name are stamped onto hits by the knowledge layer, as `Category` already is,
  leaving `store.NewSourceStore` and the changelog stores untouched.
- **Chosen direction**: the addressing vocabulary (`Address{Tier,Name}`,
  `Selector{Tier,Filter}`) lives in `internal/knowledge`; `byScope` is deleted. Beaten
  alternatives: packed labels resolved in the CLI, and pushing tiers into `store`.
- `repo.yaml` knowledge collapses to a single provider block mirroring `changelog:`;
  `config.yaml` keeps `sources[]` with `scope:` renamed to `name:`. Both superseded
  forms are rejected on load, never migrated.
- Conventions applied: remediation-bearing errors and passing tests (`spektacular`);
  content outlines in plans, no em dashes, MDX authoring, alternating section
  backgrounds, label-before-filename headings (`docs`). The em-dash rule is treated as
  binding on `docs`-repo prose only, not on the CLI repo's README/CHANGELOG.

## Learnings worth carrying forward

- The defect is reproducible in one command: reading `conventions/mdx-authoring.md` at
  `scope: project` returns `not found` because the `docs` store is unreachable.
- That failure also violates this repo's own error convention: it surfaces as
  `internal_error` with an empty `next_action`.
- `cmd/knowledge_test.go:637` (`TestKnowledgeReadWrite_DuplicateScopeResolvesToFirstRegistryRepo`)
  asserts the defect as intended behaviour and must be inverted.
- `readCategories` (`internal/knowledge/set.go:255`) exempts project-owned sources from
  repo narrowing. The spec's "no store is implicitly included or excluded" kills that
  exemption.
- `rejectLegacyRepoAddress` (`internal/config/config.go:275`) is the exact precedent for
  the old-config rejection: raw-YAML re-parse, then an `output.NewError` carrying the
  file, what was found, and what to write instead.
- The config migrator is a file-presence sniff that never parses `knowledge:`, so
  reject-never-migrate does not collide with it.
- The skill has three checked-in copies: the `{{command}}` template plus generated
  `.claude/` and `.bob/` copies that go stale until `init` re-runs.
- Both live `repo.yaml` files are in the superseded form, so every knowledge command in
  this project fails until they are hand-corrected. The project declares no shared
  stores, so the `project` tier is empty here.
- Documentation reaches further than the spec assumed: five docs-site pages plus
  `README.md` and `docs/knowledge-base.md` carry the old shape.

## Plan shape

Four milestones, fourteen phases:

1. **Addressing** (spektacular, phases 1.1-1.4) - vocabulary and single-store resolution,
   uniform narrowing, results reporting tier and store, command surface and published
   interface. Deliberately lands against the *existing* config shape so it is
   independently deliverable and breaks nothing.
2. **Configuration** (spektacular, phases 2.1-2.4) - repo declares one provider block,
   project names its shared stores, both superseded forms rejected, this project's own two
   `repo.yaml` files corrected.
3. **Guidance and documentation** (both repos, phases 3.1-3.5) - skill, workflow steps and
   agent guidance, the docs site's knowledge-base page, the rest of the docs site, and the
   CLI repo's README / in-repo doc / CHANGELOG.
4. **Dogfood** (docs, phase 4.1) - the two supplied layout entries recorded through the
   skill into the `docs` repo's store.

## Assembly note (for a resumed session)

When re-assembling `research.md` from the working files, `assumptions.md` becomes
`## Drafting assumptions` and must be inserted **before** `## Rehydration cues`, not appended
at the end. The verification step checks section order.

Section order in `context.md` is: Current State Analysis, Per-Phase Technical Notes, Testing
Strategy, Project References, Token Management Strategy, Migration Notes, Performance
Considerations.

## User answers during planning

- Asked whether documentation updates were included; confirmed they are (phases 3.3, 3.4, 3.5)
  and signed off on the plan without requesting any change. No assumption was challenged, so
  every drafting assumption stands as recorded in the plan's `research.md`.
- Walkthrough closed with explicit approval on 2026-09-03.


## Implement workflow: validation gate (read_plan)

Ran 2026-09-03 at branch `f-project-repos`, commit `0c0cb4f`.

- **Structure**: all ten required `## ` sections present; fourteen
  `#### - [ ] Phase N.M:` headings; every `*Technical detail:*` link resolves to a
  matching `### Phase N.M:` heading in `context.md`. Pass.
- **Drift**: zero mismatches. Every file, package, Go symbol, template path and CLI
  command named in `plan.md` / `context.md` was verified present in the repo it is
  attributed to. Line numbers drift by a few lines in places (`validateRepos` is at
  `internal/config/config.go:364`, not `:346`), which is cosmetic; symbol names are all
  exact. `byScope` is at `internal/knowledge/set.go:288` as stated.
- **Spec coverage**: all 16 requirements and all 19 acceptance criteria in
  `000047_repo-scoped-knowledge-addressing.md` map to a phase. No gaps, nothing descoped.
- **Changelog mode**: no `## Changelog` section in `plan.md`, so this is a
  **first-phase** invocation. Work starts at Phase 1.1.
- **Baseline**: `go build ./...` clean and `go test ./...` green across all 20 packages
  before any change. The docs site's `node_modules` is present.

## Implement decisions and answers

- User selected plan `000047_repo-scoped-knowledge-addressing` to implement.

### Phase 1.1 analysis (current phase)

Complexity Medium, but the plan's own agent strategy is **single agent, sequential**: the new
`address.go` and the reshaped `Set` methods are one tightly coupled edit and the package does
not compile between them. Research was done in the main context rather than delegated.

Confirmed touchpoints, all present:

- `internal/knowledge/set.go` — `scopedStore:20`, `Read:170`, `Write:180`, `byScope:288`,
  `NewSet:74`, `alwaysAppliedSet:161`.
- `internal/knowledge/category.go` — `type Tier:13`, `TierAlwaysApplied:20`, `TierLookedUp:24`,
  `Category.Tier` field `:43` (JSON key `tier`, must stay), `README():100`, `AlwaysApplied():112`.
- `internal/output/writer.go` — `NewError:60`, `WithResource:69`, `WithNextAction:81`;
  `cmd/root.go:252` `toErrorResponse` uses `errors.As`, so a typed error survives `%w` wrapping.
- Every caller of `Set.Read`/`Set.Write` outside the package is in `cmd/knowledge.go`
  (`:245`, `:292`), so the signature change has exactly two external call sites.
- `internal/knowledge/set_test.go` — `writeFile:13`, `twoScopeSet:25` (builds
  `config.SourceConfig` with `Scope` directly, no aggregator involved).

**Sequencing decision (substitution).** Context's Phase 1.1 note says `NewSet` should read
`src.Tier`/`src.Name`, but those are stamped by the aggregator in Phase 1.3, so reading them in
1.1 would leave the tree red between phases and break `internal/knowledge`'s own tests, which
construct `config.Config` directly and never touch the aggregator. Phase 1.1 therefore derives
tier and name inside `NewSet` from the fields that exist today: a source with `Repo` set becomes
tier `repo` named by `Repo`, any other source becomes tier `project` named by its `Scope`. That
is exactly the behaviour Milestone 1 describes ("a repository's store is addressed by its
registered name and a shared store by the name it is already declared under") and it keeps
`go test ./...` green at every phase boundary. Phase 1.3 replaces the derivation with the
aggregator-stamped fields, as planned.

### Phase 1.1 implementation notes

Written (code only; tests are the next step):

- `internal/knowledge/category.go` — `Tier` renamed `CategoryTier`, constants renamed to
  `CategoryTierAlwaysApplied` / `CategoryTierLookedUp`. `Category.Tier` field and its `tier`
  JSON key are deliberately unchanged, so `knowledge categories` output is byte-identical.
- `internal/knowledge/address.go` (new) — `Tier` (`project` / `repo` / `all`), `Address`,
  `Selector`, their `Validate()` methods, the four error codes, and `availableNames`.
  `Selector` is declared here now but only wired into the retrieval paths in Phase 1.2.
- `internal/knowledge/set.go` — `scopedStore` now carries `tier` + `name`; `byScope` deleted
  and replaced by `resolve(Address)` plus `namesIn(Tier)`; `Read`/`Write` take an `Address`;
  `NewSet` derives the address through the new `addressOf` helper; the store label passed to
  `store.NewSourceStore` is now `"<tier>:<name>"` so store-level diagnostics stay useful.

**Scope calls made because Go will not compile a half-changed package.** Two things the plan
files under later phases had to move forward into 1.1:

- `SourceInfo` drops `Scope`/`Repo` for `Tier`/`Name` now (plan puts it in 1.3). Phase 1.1's
  own acceptance criterion "a refusal names the stores available in the tier concerned, and
  those names match the ones the store-enumeration command reports" is untestable otherwise.
- `cmd/knowledge.go`'s read and write take the three-part address in `--data` now (plan puts
  the command surface in 1.4), because `Set.Read`/`Set.Write` no longer accept a scope string
  and there is no honest interim mapping from one. `knowledgeScopePathInput` became
  `knowledgeAddressInput`, and the read/write input and output schemas moved with it. Left for
  1.4: the `--tier` / `--filter` flags, the `Flags` block on `commandSchema`, the remaining six
  schemas, and inverting the defect-asserting test.

`Entry`, `Convention` and `AlwaysAppliedEntry` still carry a single `Scope` field, now fed from
the store's name. Phase 1.3 splits them into `Tier` + `Name`. In the `cmd` fixtures the visible
consequence is that a repo-declared store reports its repo's registry name (`testproj`) where it
used to report the label `project`, which is the intended behaviour change.

### Phase 1.1 tests

New `internal/knowledge/address_test.go`; `internal/knowledge/set_test.go`, `cmd/knowledge_test.go`
and `cmd/knowledge_ignore_test.go` updated and extended. All seven Phase 1.1 acceptance criteria
have covering assertions at both the knowledge-layer and command-surface levels. Full suite green,
`go vet` clean, `gofmt -l .` silent.

Two things worth carrying forward:

- `TestKnowledgeReadWrite_DuplicateScopeResolvesToFirstRegistryRepo` is **`t.Skip`ped, body intact**,
  not rewritten. A bare scope is no longer expressible, so "make it pass" would have meant turning it
  into a duplicate of the new same-location-two-repos test under a misleading name. **Phase 1.4 must
  invert it in place** — the plan already assigns that, and the skip message points at it.
- Interim inconsistency this phase introduces, now pinned by tests so a later change to it reads as
  deliberate: `store.Hit.Scope` carries the packed store label `"<tier>:<name>"` (because `NewSet`
  passes that to `store.NewSourceStore`), while `Entry.Scope` / `Convention.Scope` /
  `AlwaysAppliedEntry.Scope` carry the bare store name. Phase 1.3 removes `Hit.Scope` outright in
  favour of `Tier` + `Name` and splits the other three the same way, which resolves it.

### Phase 1.1 verification

`go build`, `go vet`, `go test ./...` and `gofmt -l .` all clean; every one of the seven
acceptance criteria has a named covering test that runs and passes. One skip in the whole
repo, the deliberate Phase 1.4 one.

Verification caught a real gap in criterion 3 that the tests had not: at the command surface a
refusal for an **omitted** tier or name carried only the generic instruction, never the store
names, because `cmd/knowledge.go` validated the address before the `Set` existed, so
`Set.resolve`'s `withAvailable` enrichment never ran. Fixed by making the knowledge set the only
validator of an address: `knowledgeAddressData` now parses and checks the path only, and
`Read`/`Write` refuse through `resolve`, which knows the configured names. `runKnowledgeWrite`
also builds the set before reading the entry body, so a refusal costs nothing. Verified by hand
against a scratch project: all four refusal shapes now name the available stores.

**Learning worth keeping:** validating an address at the command surface is always wrong here.
Only the set knows which stores are configured, so any check that runs before it is built cannot
produce the remediation this repo's error convention requires. Later phases must not reintroduce
a pre-set address check.

`availableNames` renders `TierAll` without naming a tier ("stores available: ..."), since "all"
is not a tier a store lives in.

## User answers during implementation

- **Pacing**: run straight through all remaining phases without pausing between them. Stop only
  for the three Open Questions the plan flags (the two entry bodies in Phase 4.1, the docs-site
  build/typecheck baseline in Phase 3.3, the skill-regeneration diff in Phase 3.1) or a real
  failure. Do not ask to continue after each phase.
- **Knowledge capture**: accepted the Phase 1.1 discovery. Written through the delivered command
  surface to tier `repo`, store `spektacular`, at
  `gotchas/remediation-needs-the-layer-that-holds-the-facts.md`, and verified readable there and
  absent from the `docs` store. Routed to `gotchas` rather than `conventions` because conventions
  are always-applied and paid for on every task.

## Outstanding, to fix in a later phase

- A knowledge read for a missing entry still surfaces as `internal_error` with an empty
  `next_action`, violating `conventions/error-messages-must-suggest-remediation.md`. The plan's
  Conventions section names fixing this as part of the feature. It is not assigned to a phase
  explicitly; **Phase 1.4** owns it, whose criterion reads "a refusal reaches the caller in the
  CLI's standard error envelope, carrying a code, a message naming the problem, and a next action
  giving a runnable correction". Give it a `knowledge_entry_not_found` code and a next action
  pointing at `knowledge list`/`knowledge search` for that store.

### Phase 1.2 implementation notes

Selector-based narrowing now governs every retrieval path, with no exemptions.

- `internal/knowledge/address.go` — `Selector.covers(scopedStore)` is the single membership
  predicate: tier must match unless the selector's tier is `all`, and a non-empty filter must
  name the store. No branch on whether a store is repo-declared.
- `internal/knowledge/set.go` — `readCategories` takes a `Selector` instead of `repos []string`
  and the exemption for stores with no repo attribution is **deleted**, which is the behaviour
  change the spec's predictability requirement demands. `Search`, `List`, `Conventions` and
  `AlwaysAppliedEntries` all take a `Selector`; `Search` and `List` skip a non-covered store
  before querying it, so an excluded store is never opened.
- `internal/knowledge/set.go` — new `(*Set).validateSelector`. It lives on the set, not on the
  value, because a filter naming an unreachable store must be refused with the reachable names,
  and only the set knows them. Same rule as the Phase 1.1 learning above.

**Scope call.** `cmd` keeps the existing `--repo` flag for now and maps it through the new
`knowledgeAlwaysAppliedSelector()` helper: repos named become `{Tier: repo, Filter: repos}`,
none named becomes `{Tier: all}`. Search, list and conventions pass `{Tier: all}`. That delivers
1.2's behaviour change through today's flag surface; **Phase 1.4 replaces `--repo` outright with
`--tier` and repeatable `--filter` across all four fan-out commands** and deletes this helper.

### Phase 1.2 tests

`internal/knowledge/set_test.go` and `cmd/knowledge_test.go` updated and extended; all six
criteria plus both behavioural success metrics covered. The two tests that asserted the deleted
exemption (`TestSet_AlwaysAppliedEntriesFiltersByRepo`,
`TestSet_AlwaysAppliedEntriesUnfilteredWhenNoReposNamed`) were removed and folded into the new
matrix test, as the plan directed. New fixture `narrowingSet` gives two stores per tier, which
`repoScopedSet` cannot (it has one project-tier store, and its exact names are pinned by the
Phase 1.1 refusal test).

The coverage matrix test is the load-bearing one: nine cells (three tiers x none/one/several
names), each asserting the exact covered store set across all four retrieval paths against a
hand-written expectation. The test author mutation-checked it by re-adding the project-tier
exemption to `covers` — eight tests fail, including the cmd-level one — so the matrix genuinely
pins the behaviour rather than passing vacuously.

### Phase 1.3 implementation notes

Every result envelope now reports the tier and store it came from, and identity is established
once, during aggregation.

- `internal/store/store.go` — `Hit.Scope` replaced by `Tier` + `Name`, both documented as left
  empty by the store and stamped by the knowledge layer, exactly as `Category` already was.
  `FileStore.scope` became `label` and `Scope()` became `Label()`: the store keeps a diagnostic
  label but never puts it on a hit, because the generic layer has no notion of a knowledge tier.
  `NewFileStore` / `NewSourceStore` keep their signatures, so the two changelog stores built
  through the same constructor are untouched.
- `internal/store/search.go` — stops setting the hit's attribution from the store's label.
- `internal/knowledge/set.go` — `Entry`, `Convention` and `AlwaysAppliedEntry` each swap `Scope`
  for `Tier` + `Name`; `Search` stamps both onto each hit from the originating source, alongside
  the `Category` stamp already there.
- `cmd/knowledge.go` — the aggregator stamps `Tier`/`Name` (repo tier under the registry name,
  project tier under the declared name) and the new `requireUniqueStoreNames` rejects a duplicate
  name within a tier. The same name in each tier is accepted: the identity is the pair.
- `internal/config/config.go` — `SourceConfig` drops `Repo` for `Tier` + `Name`, both
  `yaml:"-"`. `Tier` is a plain `string`, not `knowledge.Tier`, because `knowledge` imports
  `config` and the reverse would be an import cycle.

**Substitution.** `addressOf` keeps a fallback rather than reading the stamped fields blindly:
an unstamped source (the knowledge package's own tests, and any direct `NewSet` caller that
builds a config by hand) resolves to the project tier under its declared scope. Without it every
`internal/knowledge` test would have to go through the `cmd` aggregator to get a usable set,
which would make the package's unit tests depend on the command layer.

### Phase 1.3 tests

`internal/store/store_test.go`, `internal/store/search_test.go`, `internal/knowledge/set_test.go`,
`cmd/knowledge_test.go` and `cmd/knowledge_ignore_test.go` updated and extended; all five criteria
covered. Criterion 3 (a hit reads back exactly from what it carries) was mutation-checked: stamping
a constant store name in `Search` fails the test.

The test pass caught one production defect: `knowledgeSearchOutputSchema` in `cmd/knowledge.go`
still declared a `scope` property the wire format no longer emits. Fixed to `tier` + `name`.
Lesson, same shape as the Phase 1.1 one: a published schema is not checked against the value the
command actually returns, so a schema can drift silently. Phase 1.4 touches all eight schemas and
should assert each against a real invocation, not just its own literal.

Criterion 5's end-to-end route is not the one the plan assumed: `config.KnowledgeConfig.Validate`
already rejects a duplicate `scope:` in the project config during `loadConfig`, before the
aggregator runs, so two project sources sharing a name never reach `requireUniqueStoreNames`. The
guard is proved wired in through a different real path — one `repo.yaml` declaring two knowledge
sources, which both get stamped with that repo's single registry name and collide in the repo tier.
That path disappears in Phase 2.1, when a repo can declare only one store; re-check this test then.

### Phase 1.3 verification

All four gates clean, all five criteria covered by passing tests. Independently confirmed: the two
changelog stores in `cmd/storefile.go` are byte-identical to HEAD and still pass opaque labels
(`"project"` and `"repo:"+repoName`), so the generic store layer was not forced to invent a tier.
No `scope` key remains in any of the six live knowledge command outputs.

Also fixed here: three stale references the verifier found — two doc comments in `internal/store`
still describing hits as scope-tagged, and `NewSourceStore`'s parameter still named `scope`.

**Carried into Phase 1.4.** The `list`, `sources`, `conventions` and `always-applied` schemas
declare only `{"type":"array"}` with no `items`, so they are silent about item shape rather than
wrong about it. That is pre-existing (byte-identical to HEAD) but Phase 1.4's criterion — "the
published interface for each knowledge command advertises the tier, store name, and narrowing
fields" — is not met until they describe their items. Separately, `knowledge search --schema`
exits 1 without a positional query (`cobra.ExactArgs(1)`), which makes the schema undiscoverable
for the one command whose narrowing matters most; worth fixing in 1.4 while that file is open.

### Phase 1.4 implementation notes

- `cmd/spec.go` — `commandSchema` gains `Flags map[string]*schemaProp` with `omitempty`. Verified
  every other command family still publishes no `flags` key at all.
- `cmd/knowledge.go` — `--repo` deleted; `--tier` (default `all`) and repeatable `--filter`
  registered on search, list, conventions and always-applied, backed by the shared
  `knowledgeTier` / `knowledgeFilter` vars and turned into a `Selector` by `knowledgeSelector()`.
- All eight schemas: the four coarse ones (`list`, `sources`, `conventions`, `always-applied`)
  now describe their item shapes, built from a shared `knowledgeAddressProps` and
  `knowledgeItemSchema` helper so no schema can describe the address differently from its
  neighbours; the four fan-out commands publish `knowledgeNarrowingFlags`.
- `knowledge search --schema` no longer requires a positional query: `Args` relaxed to
  `MaximumNArgs(1)` and the count checked in the run function, which returns
  `knowledge_query_required` with a runnable example. A caller must be able to discover a
  command's interface without already knowing how to invoke it.
- `Set.Read` maps a store-level not-found to `knowledge_entry_not_found`, with the resource and a
  next action naming the exact `knowledge list --tier X --filter Y` that would show what the store
  does hold. This closes the pre-existing violation the plan's Conventions section called out —
  a failed read used to surface as `internal_error` with an empty `next_action`.

Exercised by hand in this project: narrowed search returns only the named store's hits, `--tier
project` legitimately returns nothing here, a filter naming an unknown store is refused with the
available names, and `--repo` is gone.

**Known rough edge, deliberately not fixed:** a stale caller passing `--repo` gets cobra's own
"unknown flag" error wrapped as `internal_error` with no next action. That is cobra's behaviour
for every command in the CLI, not something this feature introduced, and fixing it properly means
a CLI-wide unknown-flag handler. Out of scope here.

### Phase 1.4 tests

`cmd/knowledge_test.go`, `cmd/root_test.go` and `internal/knowledge/set_test.go` updated and
extended; all five criteria covered. **Skip count is now 0** — the defect-asserting test was
inverted in place rather than deleted, as the plan required:
`TestKnowledgeReadWrite_DuplicateScopeResolvesToFirstRegistryRepo` became
`TestKnowledgeReadWrite_AmbiguousRequestIsRefusedNotResolvedToFirstRepo`, on the same fixture,
now asserting the identical under-specified request is refused and records nothing while both
fully addressed forms reach their own repo's store — including the member repo's copy, which
first-match resolution could never reach.

Criterion 2 is a genuine interface test rather than a restatement: the write payload's field
names are taken from the decoded `write --schema` output and the tier value from its enum, so a
schema that drifted from the command would fail it. That is the direct answer to the Phase 1.3
finding that a schema is never checked against what its command returns.

`resetKnowledgeFlags` now loops a `knowledgeNarrowingCmds` slice, restoring `--tier` to `all` and
clearing `Changed` on both flags for all four commands, plus resetting the two package vars.
Mutation-checked: dropping the package-var reset makes the leak test fail.

**Pre-existing, unrelated:** `cmd` has order-dependent tests that fail under `-shuffle` and
`-count=2` (`TestImplementNew_*`, `TestImplementGoto_RequiresActiveWorkflow`,
`TestGotoStepRequired_*`, `TestWrapper_FailureIsPrintedExactlyOnce`, `TestUnknownSubcommand_*`).
Confirmed to fail in isolation with no knowledge test in the run, so they are not this work's
doing. Every test added by this feature passes under both `-shuffle=on` and `-count=2`.

## Milestone 1 complete

Phases 1.1-1.4 all checked off. Milestone 1's validation point was verified by hand against a
purpose-built two-repo scratch project, every clause passing, including the filesystem check that
a refused write records nothing in either the two-repo or the single-store case. Suite has zero
skips.

### Phase 2.1 implementation notes

A repo's knowledge declaration is now a single provider block, mirroring `changelog:` beside it in
the same file, so a repo can no longer declare several stores nor name the one it has.

- `internal/config/config.go` — new `RepoKnowledgeConfig{Provider, Config}` with `Validate()`,
  deliberately shaped like `ChangelogConfig`. `KnowledgeConfig` stays the project-tier list.
- `internal/config/repo.go` — `RepoConfig.Knowledge` changes type; `NewDefaultRepoConfig` emits the
  single block; `WithDefaults` fills it when the provider is empty (was: when the source list was
  empty).
- `internal/repo/footprint.go` and `internal/project/init.go` — both stop hunting a list for a
  source whose scope label matches `DefaultKnowledgeScope` and simply read the one block. That
  removes the drift class where the scaffolder and the config shape could disagree.
- `cmd/knowledge.go` — the aggregator builds one `SourceConfig` per repo from that block.

Verified by hand: `init` in a fresh directory writes the single-block form, scaffolds all six
category directories, and the resulting project accepts a write and reads it straight back with no
correction. That is the phase's "scaffolded repo needs no correction" criterion holding by
construction rather than by coincidence.

`DefaultKnowledgeScope` still exists — Phase 2.2 and 2.3 reference it, and it is removed once they
no longer do.

### Phase 2.1 tests

`internal/config/repo_test.go`, `internal/project/init_test.go`, `internal/repo/footprint_test.go`,
`cmd/repo_test.go` and `cmd/knowledge_test.go` converted and extended. Criteria 1-3 have new tests;
criterion 4 is already asserted verbatim by `cmd/repo_test.go`'s existing `minimalFootprint` layout
listing, which passes unchanged, so no duplicate was added and that test's doc comment now records
it as the criterion's guard.

Two tests were deleted rather than converted, both because the behaviour they asserted is now
unconstructible: `TestInit_CreatesProjectKnowledgeSourceOnly` (a repo has one unscoped store, so
there is no second source to skip) and the "two stores from one repo claim that repo's name"
subtest (a repo can no longer declare two stores; the refusal is still covered directly against
`requireUniqueStoreNames`). This is the re-pointing the Phase 1.3 changelog entry predicted.

`cmd/version_test.go` needed no change: contrary to the plan's note, neither `TestExecuteMigration`
nor `TestScanProjectMetadata` asserts repo.yaml *contents* — one checks the backup bytes and that
the file exists, the other checks returned struct fields. The literal generated-YAML assertion now
lives in `internal/config/repo_test.go` against `NewDefaultRepoConfig()`, which is what the migrator
builds on, so migration output is still pinned.

**Important finding for Phase 2.3.** An old-form `repo.yaml` is not merely ignored, it is actively
harmful. `RepoConfigFromYAMLFile` seeds from `NewDefaultRepoConfig()` before unmarshalling, and
yaml ignores the now-unknown `sources` key, so the seeded default survives. Because `Provider` is
then already `"file"`, `WithDefaults` sees a populated block and fills nothing, so a repo that
declared a custom location silently starts reading and writing the default `<repoRoot>/knowledge`
instead — an empty store — with no error at all. Phase 2.3 must catch this.
`RepoSourceConfig.UnmarshalYAML` is the exact precedent: a custom unmarshaller that detects the
superseded shape and returns an error printing the replacement block.

### Phase 2.2 implementation notes

The project's shared stores keep their list and are identified by name.

- `internal/config/config.go` — `SourceConfig.Scope` becomes `Name string \`yaml:"name"\`` (merged
  with the non-serialized `Name` added in 1.3, so there is one field, not two).
  `KnowledgeConfig.Validate` checks a non-empty name and uniqueness within the project tier, and
  both refusals are now `output.NewError("config_invalid", ...)` with a next action — they were
  bare `fmt.Errorf`, so a duplicate name surfaced as `internal_error` with no remediation, against
  this repo's own convention.
- `KnowledgeConfig.WithDefaults` is **deleted**, deliberately, not adapted. It existed to synthesise
  a default project-owned source at `.spektacular/knowledge`, which is now a repo-tier concern owned
  by `RepoConfig.WithDefaults`; leaving it would silently manufacture an unnamed shared store.
  `NewSet` drops the call with it, and `DefaultKnowledgeScope` / `DefaultKnowledgeLocation` lose
  their last consumers and are removed.
- `addressOf`'s fallback now only defaults the tier; the name comes from the declaration.

Verified by hand: a project declaring two shared stores addresses each by name, a write to one
reads back from it, a duplicate name is refused with an actionable `config_invalid`, and a project
declaring no shared stores is valid with `--tier project` returning empty rather than failing.

### Phase 2.2 tests

`internal/config/config_test.go`, `internal/config/repo_test.go`, `internal/knowledge/set_test.go`
and `cmd/knowledge_test.go` converted and extended; all four criteria covered. Three tests deleted
rather than converted, all because their subject is gone: the two
`TestKnowledgeConfig_WithDefaults*` tests (the repo-tier equivalents still cover the behaviour that
survives) and `TestKnowledgeConfig_ValidateRejectsDuplicateScope`, replaced rather than renamed
because the refusal is now a structured error rather than a bare string.
`TestNewSet_SynthesisesDefaultProjectSource` was replaced in place by
`TestNewSet_NoDeclaredSharedStoresYieldsAnEmptyProjectTier`, which asserts the inverse.

**Must be fixed in Phase 2.4, or the e2e environment breaks the moment 2.3 lands.** Two live harbor
fixtures still declare the superseded shape and are not exercised by `go test ./...`, so nothing has
caught them:

- `tests/harbor/implement-workflow/environment/repo.yaml`
- `tests/harbor/implement-workflow/environment/docs-repo.yaml`

Both use a `knowledge.sources` list with `scope: project`. The `tests/harbor/jobs/<date>__*/` copies
are historical job artifacts and must be left alone. `README.md` also still shows the old shape;
that is Phase 3.5's.

### Phase 2.3 implementation notes (and 2.4's live corrections, pulled forward)

Two load-time guards, modelled directly on `rejectLegacyRepoAddress`:

- `internal/config/config.go` — `rejectLegacyKnowledgeScope`, called from `ParseYAMLFile`, refuses a
  project config whose `knowledge.sources[i]` still carries `scope:`.
- `internal/config/repo.go` — `rejectLegacyRepoKnowledgeBlock`, called from `RepoConfigFromYAMLFile`,
  refuses a repo config whose `knowledge:` mapping carries `sources:` or `scope:`.

Both name the file, what was found, and print the block now required. Nothing is rewritten on disk.

**A fix the guards forced, worth keeping in mind.** The refusal was initially swallowed:
`aggregateKnowledgeSources` wraps any footprint failure in `repo_footprint` with the next action
"run `repo add` to repair the repo's footprint" — advice that cannot fix a superseded knowledge
block and sends the caller somewhere useless. The aggregator now passes a typed
`*output.ErrorResponse` through untouched and only wraps genuinely unreadable footprints. Same
lesson as Phase 1.1's, from the other direction: there, remediation could not be built too high up;
here, remediation built low down was being discarded higher up.

**Phase 2.4's corrections were made here rather than after**, because the plan's own ordering
constraint says the correction must accompany the rejection: from the moment the guard lands every
knowledge command in this working tree fails. Corrected to the single-block form:
`.spektacular/repo.yaml`, `spektacular-website/.spektacular/repo.yaml`, and the two live harbor
fixtures `tests/harbor/implement-workflow/environment/{repo,docs-repo}.yaml`. No `scope:` remains
in any live config in either repo. Phase 2.4 now verifies rather than edits.

Hand-verified: a stale repo.yaml is refused on every knowledge subcommand with a byte-identical
error on the second run and the file untouched (same md5); correcting it makes the same command
succeed; a stale project config.yaml is refused the same way; and a repo that names its store is
refused too.

### Phase 2.4 — verification only

Its edits landed with Phase 2.3, under the plan's ordering constraint. All four criteria verified
by running the commands in this working tree:

- Both registered repos declare the single-block form.
- Every knowledge subcommand runs successfully again.
- `knowledge sources` reports one store per registered repo, each under its registry name, in the
  repo tier.
- **The spec's original defect is fixed and confirmed against the real project.** Reading
  `conventions/mdx-authoring.md` addressed to `docs` returns the entry; the same path addressed to
  `spektacular` returns `knowledge_entry_not_found`. Before this work the first form returned
  "not found" because the `docs` store was unreachable behind `spektacular` in a first-match scan.

The plan states this phase has no test file, and none was added.

### Phase 3.1 implementation notes

`templates/skills/workflows/spek-knowledge/SKILL.md` rewritten: lookup, contribute and update all
state a tier and a store name; the search-result field list is tier/name; `knowledge sources` is
described as the authoritative set of writable destinations *and* of valid `--filter` names, to be
chosen from rather than inferred; the contribute checkpoint now shows tier, store name and path
before asking for approval, so approval is for a destination rather than a filename; and the update
flow notes that a hit already carries everything a read needs.

**The layered precedence chain is deleted, not restated.** The skill described
`project → team → global` most-specific-wins, which nothing in the tool has ever implemented. It is
replaced by an explicit statement that there is no precedence between stores — a repo's store
answers what is true of that repo's code, the project's shared stores answer what spans repos — and
that a candidate must never be dropped because of where it lives.

**Open question resolved (skill regeneration side effects).** Regenerating both agents' copies
touched three files beyond the two intended:

- `.spektacular/config.yaml` had `agent:` flipped to `bob`, purely because regenerating both means
  running `init bob` last. Restored to `claude` after each regeneration. **Anyone regenerating more
  than one agent's copies must restore this afterwards.**
- `.bob/skills/spek-implement/SKILL.md` and `.bob/skills/spek-plan/SKILL.md` were stale: the
  `.claude` copies were regenerated when the repo-sources work landed at `0c0cb4f`, the `.bob` ones
  were missed. **User asked to keep the refresh** — it is catch-up of prior drift, not this
  feature's doing, and leaving it means Bob keeps receiving outdated `repo list` instructions.
- The catch-up also propagated a stray-period typo present in the source template. **User asked to
  fix it**, so `templates/skills/workflows/spek-implement/SKILL.md:58` was corrected and everything
  regenerated; the typo is now absent from the template and both agents' copies.

Verified: each generated copy is byte-identical to the template with `{{command}}` substituted, and
neither carries any `scope` vocabulary.

### Phase 3.1 tests

New `templates/knowledge_addressing_test.go`, in the style of `templates/repo_source_test.go`.
Criteria 1, 3, 4, 5 covered; criterion 2 is behavioural and already covered by
`cmd/knowledge_test.go`'s round-trip tests, so it was not duplicated. All five tests were
mutation-checked.

**The test pass caught a real defect in my rewrite.** The skill's lookup intro still said
"duplicates removed and *the most specific source winning*" — a precedence claim contradicting the
"no precedence between stores" sentence ten lines below. Deleting the explicit
`project → team → global` chain was not enough; the idea survived in a summary line. Corrected in
the template and both generated copies.

**Phase 3.2 is now forced by a self-cleaning test.** `TestNoTemplateAddressesKnowledgeByScope`
sweeps the whole corpus and carries a documented `pendingScopeVocabularyTemplates` exemption list
naming exactly the six templates 3.2 must rewrite (`agents/knowledge-trigger.md`,
`agents/memory-context.md`, `skills/skill_spawn-planning-agents.md`,
`steps/implement/07-update_changelog.md`, `steps/plan/02-discovery.md`,
`steps/plan/18-walkthrough.md`). The test **also fails if an exempted file stops offending**, so
Phase 3.2 cannot finish without emptying and deleting that list.

Also new: `TestGeneratedSkillCopiesMatchTheirTemplates` asserts the checked-in `.claude/` and
`.bob/` copies are byte-identical to their templates with `{{command}}` substituted, across all
four skills and both agents. No such test existed, so a hand-edit of a generated copy was
previously undetectable.

The "no scope vocabulary" sweep deliberately matches CLI-punctuated forms (`"scope":`, `--scope`,
backticked `scope`) and full clauses, never the bare word, which appears legitimately in ~20
templates ("out of scope", "a scoped decision").

### Phase 3.2 implementation notes

Ten edits across seven templates. The load-bearing one is
`templates/steps/plan/02-discovery.md`: the always-applied load moved to
`knowledge always-applied --tier repo --filter <name>`, and the sentence promising that
"project-owned sources always load regardless of `--repo`" was **deleted rather than reworded**,
because it is now false — narrowing is honoured with no exceptions. The step also gained guidance
that a plan needing shared knowledge asks for `--tier all` and names the stores it wants.

The scope-vocabulary sweep's `pendingScopeVocabularyTemplates` exemption list is **gone**: every
template it named has been rewritten, so `TestNoTemplateAddressesKnowledgeByScope` is now a
corpus-wide sweep with no exemptions. Its doc comment was re-pointed from Phase 3.1 to 3.2.

`AGENTS.md` regenerated: exactly two lines changed, both vocabulary. No new managed section was
added, per the spec's constraint.

### Phase 3.3 open question resolved: the docs site baseline is clean

Run before editing anything, from `/home/nicj/code/github.com/jumppad-labs/spektacular-website`
(node v22.18.0):

- `npm run build` — exit 0, 12 pages built.
- `npx astro check` — **0 errors, 0 warnings**, 3 hints.

The three hints are pre-existing `ts(6387)` deprecation notices about `document.execCommand` in
`public/js/main.js`, its minified twin, and `src/layouts/Shell.astro` — none of which this work
touches. The site's own convention asks for zero errors and zero warnings before merge, and that
holds today. So there is nothing pre-existing to report to the user and nothing for this change to
silently absorb: any error or warning appearing after the doc edits is this work's doing.

### Phase 3.3 implementation notes (docs repo)

`docs:src/pages/knowledge-base.mdx` Configuration section rewritten to the plan's content outline,
in order: the two tiers; a repository declares one store (yaml); the project declares its shared
stores (yaml); addressing a store (bash); narrowing what a request covers (bash). The section keeps
its position, its `surface` prop and its subtitle slot, so the page's false/true/false/true/false
shading alternation is untouched. Also updated: the lifecycle's `Creating` example to the fully
addressed write, its `Searching and retrieving` paragraph to say every hit carries what a read
needs plus `--tier`/`--filter`, its `Keeping it up to date` line to tier/name/path, and the
`Sources are layered, most-specific first` bullet in `Why it works this way`, replaced by one
explaining the tiers answer different questions with no precedence.

The retrieval-tier ambiguity needed less work than the plan expected: `The six categories` already
said "**retrieval tier**" in the checked-in page, so only the new Configuration prose had to state
the distinction, which it does explicitly in its second paragraph.

Conventions verified after the edit: zero em dashes on the page, zero layout markup in any page
body, YAML and bash samples are fenced markdown blocks rather than JSX code props, and content sits
in slots. **Build and typecheck identical to the baseline: 0 errors, 0 warnings, the same 3
pre-existing hints.** Nothing was absorbed.

**The `docs` repo's working tree was already dirty before this work started.** About twenty files
are modified there that have nothing to do with knowledge addressing: most of
`src/components/sections/*.astro`, `src/styles/global.css`, `src/layouts/TutorialLayout.astro`,
`src/pages/tutorials/index.astro`, a deleted `src/components/tutorial/TutorialStep.astro`, and a new
untracked `src/components/sections/SectionHeader.astro`. Some in-progress component refactor.

This feature has touched exactly three files in that repo so far:
`.spektacular/repo.yaml` (Phase 2.3), `src/pages/knowledge-base.mdx` (Phase 3.3), and whatever
Phase 3.4 adds. **Anyone committing this work must stage those specific paths, not `git add -A`.**
It also means the build and typecheck baseline recorded above was taken with that refactor in the
tree, which is the right comparison for "did this change break anything" but is not a clean-tree
baseline.

### Phase 3.2 tests

All in `templates/knowledge_addressing_test.go`. Criteria 2, 3, 4 covered there; criterion 1 is
behavioural and already covered by `internal/knowledge/set_test.go`'s
`TestSet_NarrowedAlwaysAppliedLoadIsStrictlySmaller`, so it was not duplicated. All three new tests
were mutation-verified.

Two pieces of test precision worth keeping:

- `--repo` is still legitimate on the changelog commands, so the sweep does not ban it file-wide. A
  `knowledgeInvocations` helper extracts each `{{command}} knowledge ...` span and asserts only
  those are free of it.
- The criterion-3 sweep does not ban the bare word "regardless", which a future template may use
  legitimately; it flags it only when a narrowing term appears on the same line. It also positively
  asserts the replacement sentences are present, so the false claim had to be removed rather than
  reworded.

`TestManagedAgentGuidanceSectionsAreUnchanged` pins the six managed sections by hand and asserts
`AGENTS.md` carries exactly those headings and no others, which is what makes "no new managed
section was added" a property the suite holds rather than a claim. No test enumerated them before.

### Phase 3.4 implementation notes (docs repo)

Five pages corrected: `configuration.mdx` (both YAML examples, both `ConfigKey` bodies rewritten to
the plan's content examples, the repo-side `defaultValue` now "a file store at `knowledge`", and
the intro's "two knowledge sources" reworded), `projects.mdx` (repo.yaml block collapsed),
`index.mdx` (the feature card's "tags each result by scope" became tier and store),
`extending.mdx` (the `Hit` listing swaps `Scope` for `Tier` + `Name`, and the surrounding contract
now says a store leaves them empty for the knowledge layer to stamp), and
`content/tutorials/getting-started.mdx`.

The tutorial carried the contradiction the plan flagged: its multi-source example declared a
`project`-scoped store in the project's own `config.yaml`, which `configuration.mdx` says belongs in
`repo.yaml`. Resolved by splitting the example in two, shared stores in `config.yaml` under names
and the repository's own store in `repo.yaml`, and the section retitled "Knowledge that spans
repositories". Its earlier "several places to put that context" list was reworded to the two tiers.

`debugging.mdx` confirmed rather than edited: its sample log line uses `knowledge sources`, which is
unchanged.

Verified after the edits: zero em dashes in all six changed pages, zero layout markup in any page
body, build exits 0, typecheck 0 errors / 0 warnings / the same 3 pre-existing hints.

### Phase 3.5 implementation notes

`README.md`: the `Scopes, search, and de-duplication` section retitled `Tiers, search, and
de-duplication` and rewritten to the plan's content example, with the `Hit` listing swapping `scope`
for `tier` + `name`; the CLI subcommand list now describes the narrowing options and the addressed
`--data` shape; both configuration examples corrected; and the surrounding prose about the default
store and about proposing a destination before writing. The categories heading became "Six
categories, two **retrieval** tiers" and a sentence now states the two axes explicitly, so the
addressing tier introduced below cannot be read as the same thing.

`docs/knowledge-base.md`: the `Layered source precedence` section is **deleted** and replaced by
`Two tiers, and how a store is addressed`, which ends by stating plainly that there is no precedence
between stores. Every `--data '{"scope":...}'` row of the command reference table is rewritten to
the three-part address, and the rows for the four fan-out commands now show `--tier` and `--filter`.
The search-result field list leads with tier and store name.

`CHANGELOG.md`: new top entry above `000045`, opening with the plan's breaking-change text and
spelling out the two one-time edits an existing project must make.

No `scope` vocabulary remains in either document. Full Go suite green.

### Phase 3.5 tests

`cmd/docs_test.go` extended with five tests, one per criterion plus a repo-wide guard that no
markdown doc shows an unaddressed `--data '{"scope"` example. Criterion 3 was mutation-checked.

Two things worth keeping:

- **The test caught a real leftover of my edit.** `docs/knowledge-base.md:110` still said a
  refinement is "resolved by layered precedence (below)", a dangling cross-reference to the section
  I had deleted, which both asserted precedence and pointed at nothing. Rewritten to the new model.
  Deleting a section is not enough; its inbound references have to go too. This is the third time
  this feature has produced that same shape of miss.
- `knowledge.sources` could not simply be banned: the string is still legitimate for the *project*
  config and in both docs' prose. The test isolates the `### Repo configuration (repo.yaml)` section
  and asserts no `sources:` appears in its fenced YAML, while symmetrically asserting the project
  example *keeps* `sources:` and `- name: team`, so over-eager removal of the legitimate shape also
  fails.

### Phase 4.1 — the dogfood, and the plan's third open question resolved

The user did not have the original drafts. Asked what the entries should cover, they said: the
layout details in the documentation, column widths, heading styles, for consistency going forward.
Both bodies were therefore drafted and **presented in full as chat text before any write**, then
approved.

They were grounded in the site's actual layout system rather than written from the plan's summary:
`src/styles/global.css` (whose own comments already carried the frame/flow rationale and the
`--spek-centered-measure: 52ch` cap), `Section.astro` (`max-w-[1100px]` as the single frame), and
the shared heading scale `clamp(1.625rem, 3vw, 2.25rem)` repeated across `.spek-prose h2`,
`SectionHeader.astro` and `CtaBanner.astro`.

Recorded through the delivered command surface, not by editing files:

- `conventions/site-layout.md` — one frame, one flow width, one heading scale, one body rhythm,
  one component per job.
- `decisions/frame-width-flow.md` — why body text is not capped at a reading measure, and what
  would change the decision.

All five criteria verified: both readable at `{"tier":"repo","name":"docs",...}`, both
`knowledge_entry_not_found` in the `spektacular` store, search reports tier `repo` / name `docs` as
their source, the files landed under the `docs` repo's own footprint, and **neither write needed a
second attempt or a fallback**. This is precisely the write that could not be expressed before this
feature: there was no way to say which repo's store was meant.

### Test plan

Written to `000047_repo-scoped-knowledge-addressing/test-plan.md`. Three of the four success
metrics are covered by automated behavioural tests and are named there rather than restated as
procedures. The fourth, "the dogfood write succeeds first time", is manual by nature because it
measures the act of delivery rather than a property of the code, so it carries a full procedure plus
the recorded result of the run performed for this change: **pass, both entries, first attempt, no
fallback**.

### Feature changelog records

Three written: the project-level record (`000047_repo-scoped-knowledge-addressing.md`) and one per
affected repo, `--repo spektacular` and `--repo docs`, each opening with a user-facing summary and
carrying the readable reference line plus stamped provenance front matter. Both repos were affected,
so both got a record; there is no "the project-level covers it" carve-out for the colocated repo.

### Spec reconciliation

All 35 spec checkboxes (16 requirements, 19 acceptance criteria) flipped to satisfied. Each was
verified against the plan's phase changelog entries and the per-phase verification runs; none was
descoped or deferred, and nothing was marked on a partial match.
