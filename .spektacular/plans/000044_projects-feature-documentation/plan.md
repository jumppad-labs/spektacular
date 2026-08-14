---
created_date: "2026-08-14"
status: completed
closed_date: "2026-08-14"
---

# Plan: 000044_projects-feature-documentation

<!-- Metadata -->
<!-- Created: 2026-08-14T11:06:39Z -->
<!-- Commit: f7b613c -->
<!-- Branch: main -->
<!-- Repository: git@github.com:jumppad-labs/spektacular.git -->

## Overview

Spektacular's multi-repo "projects" capability (shipped in specs 000039 and 000042) lets a project register more than one repository, split configuration between project-level membership and each repository's own self-description, attribute planning and implementation work back to every repo it touches, and exclude paths from its own search and listing. None of this is yet reflected on the documentation site, so a reader following the published tutorial only learns the older single-repo mental model. This plan adds a new reference page explaining the capability, makes it discoverable from navigation, points the existing tutorial and the Spektacular repository's own README at it, and fixes one confirmed inaccuracy the README's configuration example had drifted into.

## Conventions

- **MDX authoring conventions (ConfigKey pattern, no layout HTML in page bodies, blank lines around slot content, fenced code blocks)**: the new `projects.mdx` page is genuinely new page content on the docs site, and both edited docs-site files (`getting-started.mdx`'s added pointer, and the eventual new page) must follow the established component/slot pattern rather than ad hoc markup.
- **Alternate section background shading**: `projects.mdx` will use multiple `Section`/`ConfigurationKeys`-style top-level sections; each must alternate its `surface` prop against the section immediately before it so the page reads as alternating bands, matching every existing reference page.
- **Plans must sketch content structure, not just summarize it**: this plan introduces one genuinely new page (`projects.mdx`) and a smaller copy addition (the tutorial pointer, the README fix); the phase(s) covering these must include a concrete Content outline/Content example with headings and illustrative text, not just a prose summary.
- **No em dashes**: applies to all authored prose in this plan's own documents (plan.md, context.md, research.md) and to every piece of new or edited copy in both repos (the new docs page, the tutorial pointer, and the README edits).
- **Write unit tests for all new functionality** (project knowledge base convention): does not apply. This plan adds no executable code, only documentation content. No tests are introduced; the Testing Approach step instead defines manual/visual verification checks appropriate to a docs-only change.
- **Update README.md for user-facing changes and document API changes in CHANGELOG.md** (project knowledge base convention): applies narrowly. This plan's Technical Approach already scopes a README pass (fixing the two staleness points and adding the cross-link) as one of its explicit deliverables, so this convention is satisfied by design rather than requiring extra scope.

## Architecture & Design Decisions

Three options were weighed for how to organize the new content, since this feature's only real design axis is content structure rather than code (there is no new runtime behavior, only documentation of what already ships).

**Option A: single unified concepts page (chosen).** One new page, `docs:src/pages/projects.mdx` (route `/projects/`), covers all five spec requirements as sections of one page: what a multi-repo project is and why it's useful, how the config split works (project `config.yaml` vs. repo `repo.yaml`) as one coherent explanation, cross-repo planning/changelog attribution, acquisition/cloning behavior, and the `.spektacular_ignore` mechanism. *Pros*: directly satisfies the spec's explicit acceptance criterion that a reader "does not need to leave the page... to understand the full picture" of the config split, a constraint the user flagged twice (spec Requirements, and again in `.spektacular/context.md`'s carried-forward decision). It matches the site's existing structural precedent: `configuration.mdx` and `repo-configuration.mdx` already each explain one config file plus how it relates to the other, and this page does the same at a higher, cross-cutting level. It requires no new Astro content collection, since `docs:src/content.config.ts:1-14` defines only `tutorials` and every reference page, including the two just named, is a flat routed `.mdx` file (research.md § Chosen approach). *Effort*: Medium (one new page, one nav entry, two small cross-link edits).

**Option B: split across two pages** (a "multi-repo projects" concept page plus a separate "cross-repo workflows" page). *Pros*: keeps each page shorter and would separate "static" concerns (registration, config, cloning, ignore) from "dynamic" ones (planning/changelog attribution across repos). *Cons*: directly contradicts the spec's config-split-as-one-topic requirement if config explanation and the mechanism it enables (cross-repo attribution) are separated. There's no existing site precedent for splitting one concept across two reference pages either: `configuration.mdx`/`repo-configuration.mdx` split by *file*, not by *concept*, and even so cross-link heavily to compensate. Two pages also means two nav entries and two places for a reader arriving from the tutorial or README to land, weakening discoverability. Rejected as unnecessary fragmentation for a concept that reads coherently as one page at the length the five requirements produce. *Effort*: Medium-High (extra page, extra nav entry, cross-linking overhead between the two new pages).

**Option C: extend `configuration.mdx` and `repo-configuration.mdx` in place** rather than adding a new page. *Pros*: no new nav entry needed; reuses pages readers may already know. *Cons*: those two pages are per-key configuration *reference* (the `ConfigKey` pattern, one entry per YAML field), not concept/narrative material. Cramming "why multi-repo projects are useful," a worked example, and behavioral explanations (cloning, ignore, attribution) into key-by-key reference pages would blur their purpose and still leave the "one coherent topic" criterion unmet, since a reader would need to read both pages in full plus infer the connecting narrative themselves. This also contradicts the plan's own Technical Approach constraint (spec: "New content lives in a new reference/concepts section, separate from the existing `tutorials/` content"), which implies new, purpose-built content rather than retrofitting reference pages. Rejected. *Effort*: Low nominal effort but high risk of muddling two already-working pages.

**Chosen direction: Option A.** A single new page is both the simplest structure available and the one the spec's own acceptance criteria most directly require. Cross-repo workflow behavior, acquisition, and the ignore mechanism are folded into the same page as short focused sections rather than spun out, since none of them individually carry enough content to justify a standalone page, and keeping them together lets the config-split section immediately motivate *why* cross-repo attribution and per-repo `.spektacular_ignore` files are possible (a repo owning its own config is what lets it be excluded from search or claim its own changelog entry independent of any one project).

**Key design decisions:**

- **Page structure follows `repo-configuration.mdx`'s component pattern exactly** (`Hero` → `Section`(s) → optional `ConfigurationKeys`/`ConfigKey` → `CtaBanner`), per the MDX authoring conventions cited above: this is the first-choice pattern for every content page on the site and there is no reason to deviate. `ConfigKey` blocks are used sparingly, only where the page needs to name a specific config field (e.g. pointing out that `repos[].local`/`repos[].address` live in `config.yaml` while `description`/`role`/`tags`/`deployment` live in `repo.yaml`); full per-key reference stays on `configuration.mdx`/`repo-configuration.mdx`, which the new page links to rather than duplicates (per the DRY-refactor preference and the spec's own Technical Approach: "link to the new page rather than duplicating its content inline," applied here in the reverse direction).
- **The worked example is this repo's own live registration** (spektacular plus docs, per `go run . repo list`), per the spec's Technical Approach and confirmed live in research.md: a real, running instance beats a synthetic example and needs no invention.
- **Nav entry goes in the existing "Resources" dropdown** (`docs:src/components/Nav.astro:12-20`), alongside "Configuration" and "Repository Configuration," rather than a new top-level nav item: it's the same category of reference material as its two siblings and the dropdown already groups exactly this kind of content.
- **The tutorial gets a pointer, not a rewrite**: one added sentence/link at the end of Step 4 in `docs:src/content/tutorials/getting-started.mdx` (after line 206), per the spec's explicit constraint against restructuring the tutorial.
- **The README fix is scoped to the two staleness points discovery found**, not a general audit: `README.md:180-181`'s `config.yaml` sample still nests `description`/`role` inside a `repos:` entry, which contradicts current `RepoEntry` (membership-only since spec 000042) and must be corrected; `README.md:196-201`'s `repo.yaml` sample is incomplete and gains the four descriptive fields alongside its existing `knowledge`/`changelog` keys. A cross-link to the new `/projects/` page is added to the `## Configuration` section. This matches the spec's constraint ("Must not attempt a full audit or rewrite... beyond verifying the existing project/configuration section... and adding a cross-link"), and the no-em-dashes convention applies to all edited/new prose in both repos.

Full alternatives evidence: research.md#alternatives-considered-and-rejected.

## Component Breakdown

**Multi-repo projects concept page (new)**: owns the full explanation of the multi-repo "projects" capability, what it is and why it's useful, how a repo is registered, the project/repo config split as one coherent topic, cross-repo planning and changelog attribution, automatic-clone-on-first-use and no-auto-fetch behavior, and the `.spektacular_ignore` exclusion mechanism. The single source of truth for these concepts on the documentation site; every other touched surface (nav, tutorial, README) points to it rather than re-explaining it. Built from the site's existing reference-page components (Hero, Section, sparse ConfigKey usage, CtaBanner); no new components introduced.

**Site navigation (changed)**: owns making the concept page reachable without a prior link. Gains one entry pointing at the new page, placed alongside its closest siblings (the two existing configuration reference pages) in the existing "Resources" grouping. No structural change to the navigation component itself.

**Getting-started tutorial (changed)**: owns the hands-on single-repo walkthrough; unchanged in structure and scope. Gains one pointer at the step where a reader would otherwise reasonably conclude a project is limited to one repository, directing them to the concept page for the multi-repo case. Does not duplicate or summarize the concept page's content.

**Spektacular repository README (changed)**: owns the primary repository's own user-facing description of its configuration model. Its existing `## Configuration` section is corrected where it has drifted from current behavior (the project/repo config split's descriptive-field placement) and gains one cross-link to the concept page for readers who want the fuller explanation. No other section of the README changes.

## Data Structures & Interfaces

No new data structures or interfaces are introduced. This plan adds and edits documentation content only (MDX prose, an Astro navigation array entry, and Markdown text); it changes no Go types, no config schemas, and no runtime contracts. The one array literal touched (`docs:src/components/Nav.astro`'s `items` list) is a static content list, not a type or interface boundary, and its shape is unchanged: a new element matching the existing `{ label: string; href: string }` entry shape is appended to an existing array.

## Implementation Detail

No new patterns are introduced; every changed surface follows a pattern already established elsewhere in its own repo. The new concept page follows the same component-composition shape as the site's existing reference pages: a Hero, one or more Section/ConfigurationKeys blocks with alternating surface shading, and a closing CtaBanner, assembled entirely from existing components. Nothing new is added to that component library, and no new Astro content collection or page-loading mechanism is introduced.

The tutorial and README edits are additive, not restructural: each gains a short pointer sentence at an identified location, with no reflow of surrounding content, no new section headings, and no change to either document's existing information architecture. A developer reading either diff will see a small, localized insertion rather than a rewrite.

The only piece of non-prose content is a single new entry appended to an existing static array in the site's navigation component, matching the shape and grouping of its four existing siblings.

## Dependencies

- **Specs 000039 (project-level capabilities) and 000042 (repo self-describing metadata)**: already shipped and closed; the multi-repo capability this plan documents is fully implemented by them. No further code changes to either are required; this plan only needs their current, as-shipped behavior to remain stable while the documentation work lands.
- **spektacular-website's existing Astro/MDX toolchain (Astro 5, `@astrojs/mdx`, `astro-expressive-code`, Tailwind v4)**: already configured; the new page and edits use it as-is with no version or config changes.
- **spektacular-website's existing reference-page components** (`Hero`, `Section`, `ConfigurationKeys`, `ConfigKey`, `CtaBanner`, `Button`, `Nav`): already built and in use by `configuration.mdx`/`repo-configuration.mdx`; reused without modification.
- **The `docs` repo being registered and resolvable from this project**: confirmed already registered (`go run . repo list`) and materialized locally at `../spektacular-website`, so no repo-registration or cloning step is needed before this plan's phases can touch it.
- No external libraries, services, or as-yet-unmerged prior plans block the start of this work.

## Testing Approach

This plan adds no executable code, so there are no unit, integration, or contract tests in the conventional sense. Verification instead relies on build/lint checks the docs site already enforces plus structured manual review checks, consistent with how prior docs-only work in this project (spec 000042's `repo-configuration.mdx` addition) was verified.

Automated/build-level checks against the docs repo: `npm run build` succeeds with the new page and edited files in place; `npx astro check` reports zero errors and zero warnings; a guard grep for raw layout HTML in MDX bodies returns no new matches; a grep for the em dash character across every new or edited file in both repos returns no matches.

Manual checks captured in the implementation test plan: visually confirming the new page renders correctly in a browser (section backgrounds alternate correctly, ConfigKey blocks render as expected, the CTA banner links work); clicking through the new "Resources" nav entry to confirm it resolves and highlights as active; reading the edited tutorial step and README section end to end to confirm the added pointers read naturally.

Acceptance-criteria-level checks (one per spec requirement, verified by direct inspection of the shipped page/edits against the source-of-truth facts already gathered in research.md): the new page states repo registration, why multi-repo projects are useful, and uses the live spektacular-plus-docs example; the config-split explanation covers both project-level and repo-level concerns on one page with no required page-hop; the page states cross-repo planning/changelog attribution; the page states clone-on-first-use and explicitly states no automatic fetch/pull; the page describes `.spektacular_ignore` and explicitly states a directly-named file stays accessible; the nav entry is present and the page reachable without a prior link; the tutorial links to the new page; the README's `## Configuration` section has no remaining inaccuracies and links to the new page.

Spec Success Metrics mapping: "A new user reading the documentation site can correctly explain, without consulting the source code, that a project can span multiple repositories, how configuration is split between project and repo, and how work gets attributed across repos" is classified as **Manual (captured in the implementation test plan)**, since this is a comprehension outcome, not something a build check or grep can assert; the implement workflow's test plan should capture it as a manual read-through against the five underlying facts, confirming the page states each in plain language without requiring source-code knowledge. The spec's second stated point, that correctness/completeness is already covered by the Acceptance Criteria, is satisfied by the per-requirement checks above; no metric is dropped.

## Milestones & Phases

### Milestone 1: The multi-repo projects concept is documented and discoverable on the site

**What changes**: A reader browsing the documentation site can find and read a new page that explains, in one place, what a multi-repo Spektacular project is, why it's useful, how a repository gets registered into one, how configuration is split between the project and each repo (and why), how planning and implementation work is attributed back to every repo it touches, how a registered repository becomes available locally, and how paths can be excluded from search and listing. The page is reachable directly from the site's navigation, not only by following a link from elsewhere, using this project's own live spektacular-plus-docs registration as its worked example. This milestone delivers the spec's core documentation content and is independently useful the moment it ships, even before the cross-linking in Milestone 2 lands.

#### - [x] Phase 1.1: Add the multi-repo projects concept page

**Repo:** docs

**Summary:** Adds a new reference page to the documentation site explaining Spektacular's multi-repo "projects" capability: what it is, why it helps, how a repository is registered, how project-level and repo-level configuration relate to each other as one topic, how planning and implementation work is attributed across repos, how a repository becomes available locally, and how paths can be excluded from search. The page is built entirely from components already used on the site's other reference pages and is added to the primary navigation so it's reachable without a prior link.

*Technical detail:* [context.md#phase-11](./context.md#phase-11-add-the-multi-repo-projects-concept-page)

**Content outline**

- **Hero**: "Multi-Repo Projects" / "A Spektacular project can span more than one repository, tracked together as one coherent unit of work."
- **Section: "What a multi-repo project is"**: explains a project can register more than one repository (its own colocated repo plus any number of others), motivates it with a code-repo-plus-docs-repo example, and states that this very project is itself a live instance of it.
  - *Illustrative example*: a short prose sentence plus the output shape of `spektacular repo list`, using this project's own two entries (`spektacular`, local; `docs`, registered by remote address, resolved to a local clone) as the worked example.
- **Section: "Registering a repository"**: explains `repo add` registers an entry (name, and an address or a local path) into the project, and that `repo list` shows the registry with resolved locations.
  - *Illustrative example*: a fenced `bash` block showing `spektacular repo add --data '{"name":"docs","address":"git@example.com:org/docs.git"}'` in the same style as existing CLI examples on the site.
- **ConfigurationKeys: "How configuration is split"** (the single coherent explanation the config-split acceptance criterion requires, in one section, no page-hop needed): explains the project's `config.yaml` holds membership only (which repos belong to this project and how to find them) while each repository's own `repo.yaml` holds everything that describes the repository itself (its description, role, tags, deployment, knowledge sources, and changelog settings), and explicitly states why: a repository's own file carries no reference back to any project, so the same repository can be registered into more than one project without duplicating or re-entering its description each time.
  - Two sparse `ConfigKey` blocks naming only the fields relevant to this narrative (`repos[].name`/`address`/`local` in `config.yaml`; `description`/`role`/`tags`/`deployment` in `repo.yaml`), each closing with a link to `/configuration/` or `/repo-configuration/` for the full per-key reference rather than repeating it.
- **Section: "Work that spans repositories"**: explains that planning and implementation can address requirements that belong to more than one registered repository in a single piece of work, and that every affected repository ends up with its own changelog entry naming the project and the spec/plan that produced it, so the history in each repo is self-contained.
  - *Illustrative example*: a short mock excerpt of a derived changelog entry's reference line, e.g. `> Derived from project spektacular (git@github.com:jumppad-labs/spektacular.git), spec/plan 000044_projects-feature-documentation.`
- **Section: "How a registered repository becomes available"**: states plainly that a repository registered by remote address is cloned automatically the first time it's needed, that an already-cloned repository is reused as-is, and explicitly that Spektacular never fetches or pulls a repository that's already present without the user doing so themselves.
- **Section: "Excluding paths from search"**: describes `.spektacular_ignore` (gitignore-style patterns) for keeping build artifacts and similar paths out of search and listing results, and explicitly states that a file named directly is still accessible even if it matches an ignore pattern.
- **CtaBanner**: closing call to action pointing at `/install/` or `/plugins/`, matching the pattern on `configuration.mdx`/`repo-configuration.mdx`.

**Acceptance criteria**:
- [x] A reader can navigate to the new page from the site's "Resources" navigation menu without already knowing its URL.
- [x] The page explains what a multi-repo project is, why it's useful, and how a repository is registered, using this project's own repositories as the example.
- [x] The page explains the full project/repo configuration split in one place, including why a repository's descriptive metadata lives in its own file rather than the project's.
- [x] The page states that planning and implementation can span multiple repositories and that each affected repository gets its own attributed changelog entry.
- [x] The page states that a remotely-registered repository is cloned automatically on first use and is never fetched or pulled automatically afterward.
- [x] The page states that `.spektacular_ignore` excludes paths from search and listing only, and that a directly named file remains accessible.
- [x] The docs site builds and type-checks cleanly with the new page in place.

### Milestone 2: Existing documentation points readers to the new page instead of leaving them with an outdated impression

**What changes**: The existing getting-started tutorial no longer leaves a reader assuming a Spektacular project can only ever contain one repository; at the point in the walkthrough where that impression would otherwise form, a link now points to the new concept page. The Spektacular repository's own README is checked against current shipped behavior, its one identified inaccuracy (a stale configuration example that still shows repository description fields in the wrong file) is corrected, and it gains a link to the new concept page for readers who want the fuller explanation. This milestone closes the loop opened by Milestone 1: a reader arriving through either existing entry point (the tutorial or the README) now has a path to the new material, and no existing document contradicts it.

#### - [x] Phase 2.1: Point the getting-started tutorial at the new page

**Repo:** docs

**Summary:** Adds a short pointer in the getting-started tutorial's project-initialization step so a reader following the single-repo walkthrough learns that a project can register additional repositories beyond the one they just initialized, and is directed to the new concept page for that case. The existing walkthrough steps, code blocks, and structure are left untouched.

*Technical detail:* [context.md#phase-21](./context.md#phase-21-point-the-getting-started-tutorial-at-the-new-page)

**Content example**

Appended immediately after the existing closing sentence of Step 4 ("This command installs the skills for your agent and creates the `.spektacular/` folder, which contains the default config and the project-level knowledge base."):

> A project isn't limited to the repository you just initialized: it can register other repositories too, such as a companion docs repo, so that planning and implementation work can span all of them together. See [Multi-Repo Projects](/projects/) to learn how.

**Acceptance criteria**:
- [x] The getting-started tutorial contains a link to the new concept page placed at the end of the project-initialization step.
- [x] No other part of the tutorial's existing text, structure, or code blocks changed.
- [x] The added sentence reads naturally as a continuation of the surrounding paragraph, not an inserted aside.

#### - [x] Phase 2.2: Fix and cross-link the README's configuration section

**Repo:** spektacular

**Summary:** Corrects the one inaccuracy discovery found in the README's `## Configuration` section, where the `config.yaml` example still showed a repository's description and role nested inside the project's own registry entry, even though that metadata now lives exclusively in the repository's own `repo.yaml`. The `repo.yaml` example is completed to show those descriptive fields where they now actually belong, and the section gains a link to the new documentation site page for readers who want the fuller explanation. No other part of the README changes.

*Technical detail:* [context.md#phase-22](./context.md#phase-22-fix-and-cross-link-the-readmes-configuration-section)

**Content example**

The `config.yaml` example's `docs` repo entry changes from:

```yaml
  - name: docs                      # a member repo by local path
    local: ../docs
    description: the documentation repo
    role: documentation
```

to:

```yaml
  - name: docs                      # a member repo by local path
    local: ../docs
```

The `repo.yaml` example gains the descriptive fields alongside its existing keys:

```yaml
description: the documentation repo
role: documentation
tags: [docs]
deployment: static-site
knowledge:
  sources:
    - scope: project
      provider: file
      config:
        location: .spektacular/knowledge
changelog:
  provider: file
  config:
    directory: .spektacular/changelog
```

And the section gains a closing pointer, e.g. appended after the existing full-reference link: "For the concept of multi-repo projects, why the configuration is split this way, and how work is attributed across repos, see [Multi-Repo Projects](https://spektacular.dev/projects/)."

**Acceptance criteria**:
- [x] The `config.yaml` example no longer shows `description`/`role` nested inside a `repos:` entry.
- [x] The `repo.yaml` example shows the four descriptive fields alongside its existing `knowledge`/`changelog` keys.
- [x] The `## Configuration` section links to the new documentation site page.
- [x] No other section of the README changed.

## Open Questions

None. Every fact this plan's content depends on (registration fields, config split, cloning/no-fetch behavior, cross-repo changelog attribution, and the ignore mechanism) was confirmed directly against current shipped code and cited with file:line in research.md, and both target repos (the docs site's structure and the README's current content) were read in full during discovery rather than assumed. There is no downstream system to exercise, no ambiguous API response to observe, and no untested code path this plan touches: it is a documentation change against already-stable, already-tested behavior. The three judgement calls made during drafting (page URL, extent of config-key duplication, README fix scope) each have a low-cost, easily-corrected default and are recorded in research.md's Drafting assumptions rather than parked here, since none of them require exercising the system to resolve.

## Out of Scope

- **A hands-on, step-by-step tutorial for setting up a multi-repo project**: the spec's Non-Goals section excludes this explicitly; this plan delivers reference/concept documentation only. A future tutorial, if wanted, would be a separate spec.
- **Translating or localizing the new documentation**: excluded explicitly by the spec's Non-Goals section; the site has no localization infrastructure today and none is introduced here.
- **A full audit or rewrite of the Spektacular repository's README beyond its `## Configuration` section**: excluded by the spec's Constraints; only the two staleness points discovery identified within that section are fixed, plus the one added cross-link. Any other drift elsewhere in the README is not this plan's concern.
- **Restructuring or rewriting the existing getting-started tutorial's single-repo walkthrough**: excluded by the spec's Constraints; the tutorial gains one pointer sentence only, with its existing flow, steps, and code blocks left untouched.
- **Any change to the multi-repo capability's actual behavior** (registration, config split, cloning, changelog attribution, or the ignore mechanism): this plan documents already-shipped behavior from specs 000039 and 000042; it makes no code changes to either mechanism. Any behavior change belongs in its own future spec against the `spektacular` repo, not here.
- **A new Astro content collection or site-wide navigation restructuring**: the architecture step deliberately chose a single flat page reusing the existing "Resources" navigation grouping; broader navigation or content-collection redesign is not part of this plan.

## Changelog

### 2026-08-14 — Phase 1.1: Add the multi-repo projects concept page

**What was done**: Added a new flat reference page (`src/pages/projects.mdx`, route `/projects/`) to the documentation site explaining Spektacular's multi-repo "projects" capability: what it is, why it's useful, how a repository is registered, how project-level and repo-level configuration relate to each other as one coherent topic, how planning and implementation work is attributed across repos, how a repository becomes locally available, and how paths can be excluded from search. Built entirely from existing site components (`Hero`, `Section`, `ConfigurationKeys`, `ConfigKey`, `CtaBanner`, `Button`) with alternating section background shading. Added a "Projects" entry to the site's "Resources" navigation dropdown so the page is reachable without a prior link.

**Deviations**: The closing `CtaBanner` links to `/configuration/` rather than the plan's suggested `/install/`/`/plugins/`. The plan phrased this as an illustrative example ("e.g."), and `/configuration/` is more directly relevant immediately after the page's config-split section — an in-spirit deviation, not a plan violation.

**Files changed**:
- `docs: src/pages/projects.mdx`
- `docs: src/components/Nav.astro`

**Discoveries**: None durable beyond this change — the docs site's build/check tooling, component contracts, and CLI invocation conventions all matched the plan's expectations exactly.

### 2026-08-14 — Phase 2.1: Point the getting-started tutorial at the new page

**What was done**: Added one sentence, with a link to `/projects/`, immediately after the getting-started tutorial's Step 4 closing paragraph, so a reader following the single-repo walkthrough learns a project can register additional repositories and is pointed to the new concept page for that case.

**Deviations**: None. The inserted text matches the plan's Content example verbatim.

**Files changed**:
- `docs: src/content/tutorials/getting-started.mdx`

**Discoveries**: None.

### 2026-08-14 — Phase 2.2: Fix and cross-link the README's configuration section

**What was done**: Corrected the README's `## Configuration` section: removed the stale `description`/`role` lines that were still nested under the `docs` repo's entry in the `config.yaml` example's `repos:` list (that metadata now lives exclusively in `repo.yaml` since spec 000042), added the four descriptive fields (`description`, `role`, `tags`, `deployment`) to the `repo.yaml` example alongside its existing `knowledge`/`changelog` keys, and appended a cross-link sentence to the new `/projects/` documentation page after the existing full-reference link.

**Deviations**: None. Both YAML samples validated with `gopkg.in/yaml.v3`; the diff touches only the three described locations (5 insertions, 3 deletions).

**Files changed**:
- `README.md`

**Discoveries**: None.
