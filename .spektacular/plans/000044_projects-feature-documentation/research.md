---
created_date: "2026-08-14"
status: completed
closed_date: "2026-08-14"
---

# Research: 000044_projects-feature-documentation

## Alternatives considered and rejected

**New Astro content collection for "reference"/"concepts" content vs. flat `src/pages/*.mdx` page.**
Rejected the content-collection route. `docs:src/content.config.ts:1-14` defines exactly one collection, `tutorials`, with a schema (`title`, `summary`, `order`) built for a listed, ordered walkthrough index. Every non-tutorial reference page (`configuration.mdx`, `repo-configuration.mdx`, `how-it-works.mdx`, `debugging.mdx`, `extending.mdx`, `knowledge-base.mdx`, `plugins.mdx`, `install.mdx`) is instead a flat, individually-routed `src/pages/*.mdx` file using `layout: ../layouts/Shell.astro`. Introducing a second collection for one new page would be inconsistent with every existing reference page and adds schema/loader machinery this page doesn't need (no listing/index page exists for reference content the way `/tutorials/` has one). Chosen: a new flat page, `src/pages/projects.mdx`, following the `configuration.mdx`/`repo-configuration.mdx` pattern exactly.

**Folding the config-split explanation into the existing `configuration.mdx` / `repo-configuration.mdx` pages vs. a new unified page.**
Rejected extending the existing two pages further. They already each document one *half* of the config split (project-side `repos:` registry fields in `configuration.mdx:139-158`; repo-side descriptive/knowledge/changelog fields in `repo-configuration.mdx`) and cross-link to each other for the other half (`configuration.mdx:156` "See [Repository configuration](/repo-configuration/)"; `repo-configuration.mdx:27-29` "See [Configuration](/configuration/)"). The spec's acceptance criterion explicitly requires a reader not have to leave the page to understand the whole config-split picture: splitting it further across those two pages would be the opposite of that. Chosen: the new `projects.mdx` page explains the config split as one topic (why it's split, which fields live where, why a repo can join >1 project) and links out to `configuration.mdx`/`repo-configuration.mdx` only for full per-key reference detail, not for the concept itself.

**Rewriting `getting-started.mdx`'s Step 4 vs. adding a pointer sentence.**
Rejected rewriting. The spec's constraint is explicit: "Must not rewrite or restructure the existing getting-started tutorial's single-repo walkthrough; the tutorial's existing flow stays intact, at most gaining a pointer to the new page." Chosen: a single added sentence/link after the existing Step 4 explanation (`docs:src/content/tutorials/getting-started.mdx:203-206`), not a structural change.

**Full README audit vs. targeted fix.**
Rejected a full audit. The spec's constraint caps this at "verifying the existing project/configuration section against current behavior and adding a cross-link," not a full rewrite. Discovery found one concrete inaccuracy (below) inside that section; only that gets fixed, plus the cross-link addition.

## Chosen approach — evidence

**New page location, layout, frontmatter shape**, modeled directly on `repo-configuration.mdx`:
- Flat page, `docs:src/pages/projects.mdx` (route `/projects/`), frontmatter `layout: ../layouts/Shell.astro`, `title: "... - Spektacular"`, `description: "..."`: exact shape at `docs:src/pages/repo-configuration.mdx:1-5`.
- No `slug`/`order` field; Astro file-based routing derives the route from the filename (confirmed: no such fields anywhere in `repo-configuration.mdx`/`configuration.mdx` frontmatter).

**Components to build the page from** (all already used by `configuration.mdx`/`repo-configuration.mdx`, so no new components needed):
- `Hero.astro`: page heading/sub, e.g. `docs:src/pages/repo-configuration.mdx:13-16`.
- `Section.astro`: general prose sections, `surface` boolean prop defaults `false` (`docs:src/components/sections/Section.astro:12`); alternate explicitly per `alternate-section-background` convention.
- `ConfigurationKeys.astro` (`surface` defaults `true`, `docs:src/components/sections/ConfigurationKeys.astro:8`) wrapping `ConfigKey.astro` (`docs:src/components/sections/ConfigKey.astro`, 24 lines; props `name`, `type?`, `defaultValue?`): only needed if the new page documents config keys directly; likely used sparingly since the config *keys* are already documented on `configuration.mdx`/`repo-configuration.mdx` and this page should link out rather than duplicate.
- `CtaBanner.astro` + `Button.astro`: closing CTA, e.g. `docs:src/pages/repo-configuration.mdx:120-125` (links to `/plugins/`).
- No callout/admonition/tabs component exists in the codebase (verified: zero matches for Callout/Note/Warning/Admonition/Tabs under `docs:src/components`); notes are plain prose sentences, e.g. `docs:src/pages/how-it-works.mdx:85`.
- Code blocks are plain fenced markdown (astro-expressive-code handles chrome automatically per `mdx-authoring.md` convention Rule 4), e.g. YAML samples at `docs:src/pages/configuration.mdx:171-205`.

**Navigation entry**: `docs:src/components/Nav.astro:12-20`, the "Resources" dropdown's `children` array. Add `{ label: "Projects", href: "/projects/" }` alongside the existing four entries (Configuration, Repository Configuration, Extending, Debugging). Active-state highlighting is automatic via `pathname.startsWith(href)` (`docs:src/components/Nav.astro:24-30`); no extra wiring.

**Tutorial pointer**: `docs:src/content/tutorials/getting-started.mdx`, Step 4 "Initialise your project" (lines 162-207). The single-repo-implying framing is at lines 164-166 ("Every repository you want to use Spektacular with needs to be initialised...") and 203-206 (init "creates the `.spektacular/` folder, which contains the default config and the project-level knowledge base"). Add one sentence/link after line 206 pointing to `/projects/` for the multi-repo case, without touching the existing walkthrough steps or code blocks.

**Multi-repo capability facts to document** (all from `spektacular:` repo code, current HEAD):

1. *Registration*: `repo add`/`repo list` CLI (`spektacular:cmd/repo.go:15-30`). Registry entry (`config.RepoEntry`, `spektacular:internal/config/config.go:129-136`) is membership-only: `Name`, `Address`, `Local`, `Dependencies`, `Provider` (defaults `git`), `Config` (provider-specific, empty today). At least one of `Address`/`Local` required, `Local` wins when both set (`spektacular:internal/config/config.go:296-298`, `spektacular:internal/repo/set.go:161-171`).

2. *Config split*: project `.spektacular/config.yaml` → `config.Config` (`spektacular:internal/config/config.go:147-159`): identity (`Name`, `Source`), agent/command settings, central spec/plan/changelog stores, `Repos []RepoEntry` registry, optional project-owned `Knowledge`. Repo `.spektacular/repo.yaml` → `config.RepoConfig` (`spektacular:internal/config/repo.go:20-27`): `Description`, `Role`, `Tags`, `Deployment` (self-describing metadata, moved here by spec 000042) plus repo-scoped `Knowledge`/`Changelog`. `RepoConfig` "carries no pointer back to any project... a repo can belong to multiple projects" (`spektacular:internal/config/repo.go:17-19`): this is *why* the split exists and is the core concept-page fact. Merge is read-only at list/plan time via `Set.DescriptiveMetadata` (`spektacular:internal/repo/set.go:112-130`), never cross-referenced on disk.

3. *Acquisition/cloning*: `Set.resolve` (`spektacular:internal/repo/set.go:161-199`). Local path used directly if present (git never invoked); otherwise clone into `.spektacular/repos/<repo-name>/` **only if the directory doesn't already exist** (`os.Stat` + `os.IsNotExist` gate, `spektacular:internal/repo/set.go:185-189`). An existing clone is reused as-is. **No fetch/pull exists anywhere**: `GitRunner` interface exposes only `Clone`, `LocalHead`, `RemoteHead` (`spektacular:internal/repo/git.go:20-29`); staleness is a warn-only comparison via `git ls-remote`, never `git fetch` (`spektacular:internal/repo/set.go:201-218`, explicit statement at line 215). `Materialized` is `true` only for the clone-path branch (`spektacular:internal/repo/set.go:195`), confirmed live: `repo list` in this repo shows `"materialized": false` for both entries (both resolve via `local`). Listing never triggers cloning: `Present`/`LocalRoot` never invoke git (`spektacular:internal/repo/set.go:74-110`).

4. *Cross-repo workflow attribution*: plan discovery/architecture step templates render a repo roster (`spektacular:internal/repo/roster.go:18-40`) and require the drafted architecture to name which repo (and files) each requirement is carried out against, recorded in the plan's context document (`spektacular:templates/steps/plan/03-architecture.md:14`). Implement workflow step 10 (`spektacular:templates/steps/implement/10-update_feature_changelog.md`) writes one central changelog entry, then derives one entry per affected repo via `changelog file write <plan>.md --repo <name> --from ...` (line 48), each carrying a human-readable reference line naming the originating project and spec/plan (line 44). Underlying routing: `--repo` flag on `changelog file write/read/list` only (`spektacular:cmd/changelog_file.go:15-20`), resolved via `repoRoutedStore` (`spektacular:cmd/storefile.go:102-134`) into the target repo's own changelog directory, namespaced by the *calling* project's name (`<directory>/<project-name>/...`, line 132) so multiple projects sharing a repo never collide. Provenance fields (`Project`, `ProjectSource`, `Spec`, `Plan`) are stamped mechanically, not agent-authored (`spektacular:internal/metadata/metadata.go:46-49,59-62`; `spektacular:cmd/storefile.go:142-150,219-225`).

5. *Ignore mechanism*: `.spektacular_ignore` (`IgnoreFileName`, `spektacular:internal/store/ignore.go:15`), gitignore syntax via `github.com/sabhiram/go-gitignore`. `ignoreStore` wraps any store (`spektacular:internal/store/ignore.go:61-119`), applied at every store construction site. `List`/`Search` filter through the matcher (lines 90-119); `Read`/`Write`/`Delete`/`Exists` delegate straight through untouched (lines 82-88). The doc comment states plainly: "a directly named path is never blocked by an exclusion rule" (line 59-60). Applies to any source root (a repo, or the project's own storage).

**README accuracy check**: `spektacular:README.md:144-216`, `## Configuration` section.
- Lines 146-149 (the two-file split explanation) are accurate but line 149 says `repo.yaml` covers only "its knowledge sources and its changelog provider." This **omits** the four descriptive fields (`description`, `role`, `tags`, `deployment`) that spec 000042 moved there. Minor omission, not a contradiction; can be tightened when the cross-link is added.
- **Concrete inaccuracy**: the `config.yaml` example at `README.md:175-183` shows a `repos:` entry with `description:`/`role:` nested inside it (lines 180-181). This was true pre-000042 but contradicts current `RepoEntry` (membership-only, no descriptive fields: `spektacular:internal/config/config.go:129-136`) and contradicts 000042's own acceptance criterion ("Project entry is membership-only... no description, role, tags, or deployment"). Verified directly by reading `README.md:175-183` in this session. **Must be fixed**: remove `description`/`role` from that `repos:` entry's YAML sample.
- The `repo.yaml` example at `README.md:196-201` shows only `knowledge:`/`changelog:` keys, omitting the four descriptive fields it should now show alongside them (incomplete, not contradictory). Should be added for accuracy per the same spec.
- The prose paragraph after the `config.yaml` example (`README.md:192`, "description, role, tags, dependencies, and deployment are optional metadata...") is itself accurate but doesn't state which file they belong in, and sits right after a YAML sample that shows them in the wrong file, reinforcing the need to fix the sample.
- Materialization sentence (`README.md:192`, "Cloned repos are never fetched or pulled automatically") and the `.spektacular_ignore` section (`README.md:211-213`) are both accurate against current code; no changes needed there.
- No existing cross-link from README's Configuration section to the docs site's new `/projects/` page; add one, per spec's Technical Approach ("link to the new page rather than duplicating its content inline").

**Worked example for the new page**: this repo's own live registration, confirmed via `go run . repo list`: `spektacular` (local `.`, role `tool`) and `docs` (address `git@github.com:jumppad-labs/spektacular-website.git`, local `../spektacular-website`, role `documentation`), both currently `materialized: false` since both resolve via `local`. Use this pair as the illustrative example per the spec's Technical Approach.

## Files examined

- `spektacular:.spektacular/specs/000044_projects-feature-documentation.md` — the spec being planned; full requirements/acceptance criteria/constraints read.
- `spektacular:.spektacular/context.md` — prior session's spec-phase decisions: docs site is primary deliverable, README gets light pass only, config split must be one coherent topic (explicit user constraint).
- `spektacular:.spektacular/plans/000039_project-level-capabilities/plan.md` — architecture/history of the config split, repo registry, git provider, ignore wrapper, changelog namespacing design. Confirms Option B (git binary + pure-Go gitignore matcher) was chosen for authentication reasons.
- `spektacular:.spektacular/plans/000042_repo-self-describing-metadata/plan.md` — moved descriptive fields (`description`/`role`/`tags`/`deployment`) from `RepoEntry` to `RepoConfig`; identified the exact docs components (`ConfigKey.astro`, nav location) and confirmed no repo-level config reference page existed *before* that plan (it created `repo-configuration.mdx`).
- `spektacular:internal/config/config.go:124-159,194-223,286-315` — `RepoEntry`/`Config` struct definitions, validation, loader.
- `spektacular:internal/config/repo.go:13-71` — `RepoConfig` struct, defaults, loader; doc comments explaining no-back-pointer design.
- `spektacular:internal/repo/set.go:14-218` — resolution order, clone gating (`os.Stat`/`IsNotExist`), staleness warn-only check, `Materialized` field, `Present`/`LocalRoot` side-effect-free listing.
- `spektacular:internal/repo/git.go:20-91` — `GitRunner` interface (`Clone`/`LocalHead`/`RemoteHead` only, no fetch/pull), subprocess execution details.
- `spektacular:internal/repo/roster.go:18-40` — repo roster construction used by plan templates.
- `spektacular:internal/store/ignore.go:15-119` — `.spektacular_ignore` constant, `ignoreStore` wrapper, List/Search filtered vs. Read/Write/Delete/Exists passthrough.
- `spektacular:internal/metadata/metadata.go:46-62` — provenance fields (`Project`/`ProjectSource`/`Spec`/`Plan`) on changelog frontmatter.
- `spektacular:cmd/repo.go:15-316` — `repo add`/`repo list` command definitions, input schema, split-write logic between `RepoEntry` and `RepoConfig`.
- `spektacular:cmd/storefile.go:102-150,171-349` — `--repo` flag routing, `repoRoutedStore`, provenance stamping.
- `spektacular:cmd/changelog_file.go:15-20` — confirms `--repo` flag is opt-in only for changelog commands.
- `spektacular:templates/steps/plan/03-architecture.md:5-14` — per-repo attribution requirement in plan architecture step.
- `spektacular:templates/steps/implement/10-update_feature_changelog.md:32-58` — central-then-derived changelog writing procedure.
- `spektacular:README.md:144-216` — `## Configuration` section; found stale `repos:` YAML sample (lines 180-181) and incomplete `repo.yaml` sample (lines 196-201).
- `docs:src/content.config.ts:1-14` — only one content collection (`tutorials`) exists; reference pages are flat `src/pages/*.mdx`.
- `docs:src/pages/configuration.mdx:139-158` — existing `repos:` key documentation (project-side), already cross-links to `repo-configuration.mdx`.
- `docs:src/pages/repo-configuration.mdx:1-33` — frontmatter shape, `Hero`/`Section` usage pattern, existing cross-link back to `configuration.mdx`; created by plan 000042.
- `docs:src/pages/how-it-works.mdx:250-261` — existing narrative example referencing a "docs" repo during cross-repo interview; already links to `/repo-configuration/`.
- `docs:src/content/tutorials/getting-started.mdx:162-207` — Step 4 "Initialise your project," the single-repo-implying passage that needs the new pointer.
- `docs:src/components/Nav.astro:1-21` — nav item types and the "Resources" dropdown array; exact insertion point for the new page.
- `docs:src/components/sections/ConfigKey.astro`, `ConfigurationKeys.astro`, `Section.astro`, `Hero.astro`, `CtaBanner.astro` — reusable content components available for the new page, all already in use on sibling reference pages.

## External references

None. This is internal documentation work with no external library/API integration; all facts are sourced from this project's own code and prior plans.

## Prior plans / specs consulted

- `000039_project-level-capabilities` — introduced the config split, repo registry, git provider, ignore wrapper, and changelog namespacing being documented. Read for architecture rationale (why git binary was chosen for auth reasons) and original data structure shapes (later partially superseded by 000042).
- `000042_repo-self-describing-metadata` — moved descriptive fields to `repo.yaml`, created `repo-configuration.mdx` and its nav entry, established the `ConfigKey`/`ConfigurationKeys` docs pattern this plan will reuse. Directly informs the README staleness finding (the README was never updated after this plan shipped).

## Open assumptions

- **Page URL/filename**: assumed `src/pages/projects.mdx` → `/projects/` as the new page's route. Not verified against any naming convention document beyond pattern-matching `configuration.mdx`→`/configuration/`. If the implementer or user prefers a different slug (e.g. `/multi-repo/` or `/concepts/projects/`), this is a cheap rename with no structural impact; flag but do not block on it.
- **Extent of config-key duplication on the new page**: assumed the new page explains the config split *conceptually* (why two files, what each owns, why a repo can join >1 project) and links out to `configuration.mdx`/`repo-configuration.mdx` for exhaustive per-key reference, rather than re-documenting every key. This satisfies the "one coherent topic, no page-hopping to understand the whole picture" acceptance criterion without duplicating the existing `ConfigKey` reference tables. If this reads as insufficient during the walkthrough, the fix is additive (more `ConfigKey` blocks on the new page), not structural.
- **README fix scope**: assumed "fix" means correcting the two identified staleness points (stale `repos:` sample, incomplete `repo.yaml` sample) plus adding the cross-link, not a broader rewrite, consistent with the spec's explicit constraint against a full README audit.

## Drafting assumptions

### Single unified concepts page over split pages or extending existing reference pages (architecture)
- **Decision**: Add one new page, `docs:src/pages/projects.mdx` (route `/projects/`), covering all five documentation requirements (concept/registration, config split, cross-repo attribution, acquisition/cloning, ignore mechanism) as sections of one page, rather than splitting into multiple new pages or folding content into `configuration.mdx`/`repo-configuration.mdx`.
- **Rationale**: The spec explicitly requires the config-split explanation to read as one coherent topic without page-hopping, a constraint the user raised twice (spec Requirements, and the spec-phase decision carried into `.spektacular/context.md`). A single page is the only option that satisfies this directly, matches the site's own structural precedent for reference content, and needs no new Astro content collection.
- **Rejected**: Splitting into a separate "concepts" page and "cross-repo workflows" page (fragments a single requirement across two pages, contradicting the explicit constraint); extending `configuration.mdx`/`repo-configuration.mdx` in place (those are per-key reference pages, not concept/narrative pages, and cramming narrative content into them blurs their purpose and still doesn't satisfy the one-page requirement without inferred cross-reading).

### Config-key duplication kept minimal, page links out for full reference (architecture)
- **Decision**: `projects.mdx` explains the config split conceptually (why two files, what each owns, why a repo can join more than one project) and uses `ConfigKey` blocks sparingly only to name the specific fields relevant to the narrative; it links to `configuration.mdx`/`repo-configuration.mdx` for exhaustive per-key reference rather than re-documenting every key.
- **Rationale**: Avoids duplicating the existing, already-accurate `ConfigKey` reference tables (DRY), while still letting the reader understand the whole config-split picture without leaving the page, since the *concept* (not the exhaustive key list) is what the acceptance criterion asks for.
- **Rejected**: Fully re-documenting every `repos[]`/`repo.yaml` key on the new page as well (duplication risk: two sources of truth for the same key list would drift, exactly the failure mode spec 000042 eliminated for repo descriptive metadata).

### README fix scoped to two identified staleness points plus cross-link (architecture)
- **Decision**: Fix `README.md:180-181` (stale `description`/`role` nested under a `config.yaml` `repos:` entry) and `README.md:196-201` (incomplete `repo.yaml` sample missing the four descriptive fields), and add one cross-link to the new `/projects/` page in the `## Configuration` section. No broader README changes.
- **Rationale**: Matches the spec's explicit constraint against a full README audit or rewrite, and these are the only two inaccuracies discovery surfaced by direct inspection of the current file.
- **Rejected**: A general pass over the whole README (out of scope per spec constraint); leaving the stale sample unfixed and only adding the cross-link (would fail the spec's "no inaccuracies found (or any found are fixed)" acceptance criterion).

## Rehydration cues

- Re-read `spektacular:.spektacular/specs/000044_projects-feature-documentation.md` for full requirements/acceptance criteria/constraints text.
- Re-read `spektacular:.spektacular/context.md` for the spec-phase decisions (docs site primary, README light-touch, config-split-as-one-topic user constraint).
- `go run . repo list` in this repo reproduces the worked example (spektacular + docs, both `materialized: false`, both resolved via `local`).
- `go run . knowledge always-applied` reloads the conventions used throughout this research: `mdx-authoring.md`, `plan-content-pages.md`, `no-em-dashes.md`, `alternate-section-background.md`.
- Sibling page `docs:src/pages/repo-configuration.mdx` is the closest structural template for the new page (same layout, same component imports, same cross-link pattern); read it in full before drafting.
