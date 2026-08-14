---
created_date: "2026-08-14"
status: completed
closed_date: "2026-08-14"
---

# Context: 000044_projects-feature-documentation

## Current State Analysis

The multi-repo "projects" capability is fully shipped in the `spektacular` repo (specs 000039 and 000042, both closed) but has no representation on the documentation site (`docs` repo, spektacular-website). The docs site's only content collection is `tutorials` (`docs:src/content.config.ts:1-14`); every reference page (`configuration.mdx`, `repo-configuration.mdx`, `how-it-works.mdx`, `debugging.mdx`, `extending.mdx`, `knowledge-base.mdx`, `plugins.mdx`, `install.mdx`) is instead a flat, individually-routed `docs:src/pages/*.mdx` file using `layout: ../layouts/Shell.astro`. Two existing pages already document config-*key* shape (`configuration.mdx:139-158` for the project-side `repos:` registry, `repo-configuration.mdx` for the repo-side descriptive/knowledge/changelog fields) and cross-link to each other, but neither explains the *behavioral* mechanics (cloning, no-fetch, cross-repo attribution, `.spektacular_ignore`) requested by this spec: confirmed by a repo-wide grep finding zero existing mentions of "clone," fetch behavior, or `.spektacular_ignore` outside of unrelated `.gitignore` references. The `spektacular` repo's own README (`README.md:144-216`) already has a `## Configuration` section describing the split, but its `config.yaml` example (`README.md:180-181`) is stale relative to spec 000042 (still nests `description`/`role` inside a `repos:` entry) and its `repo.yaml` example (`README.md:196-201`) is incomplete (omits those same fields).

Full fact-by-fact detail on the capability itself (registration, config split, cloning/no-fetch, cross-repo attribution, ignore mechanism), all cited to file:line in the `spektacular` repo, is in research.md. It is not duplicated here to avoid drift between the two documents.

## Per-Phase Technical Notes

### Phase 1.1: Add the multi-repo projects concept page

- `docs:src/pages/projects.mdx` (new file): new flat reference page, frontmatter `layout: ../layouts/Shell.astro`, `title: "Multi-Repo Projects - Spektacular"`, `description: "..."`. Import and use `Hero`, `Section`, `ConfigurationKeys`, `ConfigKey`, `CtaBanner`, `Button` from `docs:src/components/sections/*.astro` and `docs:src/components/Button.astro`, exactly as imported in `docs:src/pages/repo-configuration.mdx:6-11`. Follow the Content outline in plan.md's Phase 1.1 for section order and copy. Alternate `surface` on each `Section`/`ConfigurationKeys` against the section before it (`Section.astro` defaults `surface=false` at `docs:src/components/sections/Section.astro:12`; `ConfigurationKeys.astro` defaults `surface=true` at `docs:src/components/sections/ConfigurationKeys.astro:8`).
- `docs:src/components/Nav.astro:12-20`: append `{ label: "Projects", href: "/projects/" }` to the `Resources` dropdown's `children` array, placed after `"Repository Configuration"` and before `"Extending"`.
- Cross-reference: use the confirmed live worked example: `spektacular` (local `.`, role `tool`) and `docs` (address `git@github.com:jumppad-labs/spektacular-website.git`, local `../spektacular-website`, role `documentation`), both `materialized: false` per `go run . repo list` (research.md § Chosen approach, worked example paragraph).
- Facts to state, each traceable to research.md:
  - Registration/config split: `config.RepoEntry` (`spektacular:internal/config/config.go:129-136`, membership-only) vs. `config.RepoConfig` (`spektacular:internal/config/repo.go:20-27`, descriptive + knowledge/changelog); no-back-pointer design (`spektacular:internal/config/repo.go:17-19`).
  - Cloning/no-fetch: `spektacular:internal/repo/set.go:161-199` (clone-only-if-absent gate at 185-189), no fetch/pull method exists (`spektacular:internal/repo/git.go:20-29`), staleness is warn-only via `ls-remote` (`spektacular:internal/repo/set.go:201-218`).
  - Cross-repo attribution: `spektacular:templates/steps/plan/03-architecture.md:14` (per-repo attribution requirement), `spektacular:templates/steps/implement/10-update_feature_changelog.md:38-58` (derived entries, reference line at line 44), provenance fields `spektacular:internal/metadata/metadata.go:46-49,59-62`.
  - Ignore mechanism: `spektacular:internal/store/ignore.go:15,59-60,82-119` (List/Search filtered, Read/Write/Delete/Exists pass through untouched).
- Reuse existing site conventions: MDX authoring Rules 1-4 (`docs:.spektacular/knowledge/conventions/mdx-authoring.md`), blank lines around slot content, fenced code blocks for CLI/YAML examples (astro-expressive-code handles chrome automatically, no per-block config needed).

**Complexity**: Medium. Genuinely new page content across six sections, but every structural element (layout, components, cross-linking pattern) is copied from `repo-configuration.mdx`'s proven shape; the complexity is in accurately compressing five distinct fact groups into readable prose without duplicating existing per-key reference pages.
**Token estimate**: ~15k tokens (reading `repo-configuration.mdx` and `configuration.mdx` in full as structural templates, drafting ~150-200 lines of new MDX, one small Nav.astro edit).
**Agent strategy**: Low: single agent, sequential. The page is one cohesive piece of prose that needs to read as one coherent narrative; splitting drafting across parallel agents risks inconsistent voice or duplicated explanation across sections. The Nav.astro edit is trivial and can be done by the same agent immediately after the page is drafted.

### Phase 2.1: Point the getting-started tutorial at the new page

- `docs:src/content/tutorials/getting-started.mdx`: insert one sentence with a link to `/projects/` immediately after the existing text ending at line 206 ("...creates the `.spektacular/` folder, which contains the default config and the project-level knowledge base."). Exact insertion text is in plan.md's Phase 2.1 Content example. No other line in this file changes.

**Complexity**: Low. A single-sentence, single-location insertion into an existing paragraph.
**Token estimate**: ~3k tokens (read the surrounding step for phrasing consistency, make the one insertion).
**Agent strategy**: Low: single agent, sequential.

### Phase 2.2: Fix and cross-link the README's configuration section

- `spektacular:README.md:180-181`: remove the `description: the documentation repo` and `role: documentation` lines from the `docs` entry inside the `config.yaml` example's `repos:` list (currently at lines 180-181, inside the block spanning `README.md:175-183`), matching plan.md's Phase 2.2 Content example.
- `spektacular:README.md:196-201`: the `repo.yaml` example currently shows only `knowledge:`/`changelog:` keys; add `description`, `role`, `tags`, `deployment` above them, matching plan.md's Phase 2.2 Content example. Keep the existing `knowledge:`/`changelog:` block content unchanged.
- `spektacular:README.md:192` (the prose sentence following the `config.yaml` example): leave as-is; it is accurate and does not need a `repo.yaml` pointer added, since the two YAML samples themselves now correctly show where each field lives.
- Add one link to `https://spektacular.dev/projects/` in the `## Configuration` section, placed after the existing full-reference sentence at `README.md:203` ("For the full reference... see the [configuration documentation](https://spektacular.dev/configuration/)."), per plan.md's Phase 2.2 Content example.
- No em dashes in any inserted text (`spektacular:.spektacular/knowledge/conventions/no-em-dashes.md`).

**Complexity**: Low. Two small, precisely located YAML-sample edits plus one added sentence; no prose restructuring.
**Token estimate**: ~3k tokens (read `README.md:144-216` in full for context, make the three edits).
**Agent strategy**: Low: single agent, sequential.

## Testing Strategy

No unit/integration tests are added (docs-only change). Per phase:

- **Phase 1.1**: `npm run build` and `npx astro check` in the `docs` repo must both report zero errors after the new page and nav entry are added; guard grep for raw layout HTML (`grep -nE "<div|<section|class=" src/pages/*.mdx`) must show no new matches; visual check that the page renders with alternating section backgrounds and that the CtaBanner links resolve; manual click-through from the "Resources" nav to confirm the entry appears and highlights as active on `/projects/`.
- **Phase 2.1**: read the edited paragraph in context to confirm it flows naturally; confirm the `/projects/` link resolves once Phase 1.1 has shipped.
- **Phase 2.2**: confirm both YAML samples parse as valid YAML after editing; confirm the added link resolves to the live docs site path once Phase 1.1 has shipped; grep the diff for em dashes.
- **Cross-phase**: grep every new/edited file in both repos for the em dash character as a final check before considering the plan done.
- **Manual, deferred to implement workflow's test plan**: the spec's Success Metric (a new user can explain the multi-repo concept, config split, and cross-repo attribution without consulting source code) is a comprehension check, captured as a manual read-through against the five underlying facts once the page is live.

## Project References

- `spektacular:.spektacular/specs/000044_projects-feature-documentation.md` — this plan's spec.
- `spektacular:.spektacular/plans/000039_project-level-capabilities/plan.md` — architecture history for the config split, repo registry, git provider, and changelog namespacing being documented.
- `spektacular:.spektacular/plans/000042_repo-self-describing-metadata/plan.md` — moved descriptive fields to `repo.yaml`, created `repo-configuration.mdx` and its nav entry, established the docs pattern this plan reuses; directly explains why the README is now stale.
- `docs:src/pages/repo-configuration.mdx` — the closest structural template for the new page.
- `docs:src/pages/configuration.mdx` — sibling reference page; source of the existing `repos:` key documentation the new page cross-links to rather than duplicates.
- `docs:src/pages/how-it-works.mdx:250-261` — existing narrative example referencing a "docs" repo during cross-repo interview; already links to `/repo-configuration/`, unaffected by this plan but worth being aware of for tone consistency.

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|-------------|----------------|
| Low | ~10k | Single agent, sequential |
| Medium | ~25k | 2-3 parallel agents |
| High | ~50k+ | Parallel analysis, sequential integration |

All three phases in this plan are Low-to-Medium complexity and are best executed sequentially by a single agent per phase (see Per-Phase Technical Notes above); no phase benefits from parallel sub-agents since each is either one cohesive piece of prose or a small, precisely located edit.

## Migration Notes

None. This plan adds and edits static content only; no data migration, no config schema change, and no backward-compatibility concern.

## Performance Considerations

None. A new static MDX page and small text edits have no measurable effect on site build time or runtime performance beyond the negligible cost of one additional pre-rendered route.
