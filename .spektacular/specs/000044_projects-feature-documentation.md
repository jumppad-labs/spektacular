---
created_date: "2026-08-14"
status: completed
closed_date: "2026-08-14"
---

# Feature: 000044_projects-feature-documentation

## Overview

Spektacular recently gained the ability to work across multiple repositories as a single project — registering a code repo alongside a docs repo, planning and implementing work that spans both, and recording attributed changelog entries in each. This capability isn't yet reflected in Spektacular's own documentation site, so new users following the published tutorial still learn only the older, single-repo mental model and have no way to discover or learn how multi-repo projects work. This work adds documentation covering the concept — how repos are registered, how configuration is organized between the project and each repo, and how work is tracked across repos — giving users a way to learn and adopt the capability that already exists in the product.

## Requirements

- [x] **Concept documentation for multi-repo projects**
  New documentation on the documentation site explains, at a concept level, that a Spektacular project can span more than one repository, why that's useful (e.g. code and docs tracked together), and how repositories are registered into a project.

- [x] **Configuration model explained as one coherent topic**
  The documentation explains how configuration is divided between project-level settings and settings owned by each individual repository, and why that division exists (a repository can belong to more than one project). This explanation is presented as a single, unified topic rather than split across separate, disconnected pages, so a reader can understand the whole configuration model in one place.

- [x] **Cross-repo workflow behavior explained**
  The documentation explains that planning and implementation work can span multiple repositories at once, and that each affected repository receives its own record of what changed and why, attributable back to the project and originating spec.

- [x] **Repository acquisition behavior explained**
  The documentation explains how a repository registered by remote address becomes available locally (cloned automatically on first use) and clarifies that Spektacular does not fetch or pull an already-present repository without the user initiating it.

- [x] **Search/listing exclusion mechanism explained**
  The documentation explains that a project or repository can exclude paths (e.g. build artifacts) from Spektacular's own search and listing behavior, while still allowing direct access to a specifically named file.

- [x] **New documentation is discoverable**
  The new documentation is reachable from the documentation site's navigation, not just linked from other pages, so a user browsing the site (not only following the existing tutorial) can find it.

- [x] **Existing tutorial does not contradict the new documentation**
  The documentation site's existing getting-started tutorial is not left implying that a project can only ever contain a single repository; at minimum it points readers to the new documentation for the multi-repo case.

- [x] **Primary repository's documentation stays accurate and cross-linked**
  The Spektacular repository's own README continues to accurately describe the multi-repo project capability and points readers to the new documentation site content for a fuller explanation.

## Constraints

- Must be built using the documentation site's existing Astro/content-collection framework and conventions — no new site tooling or a separate publishing mechanism.
- Must not rewrite or restructure the existing getting-started tutorial's single-repo walkthrough; the tutorial's existing flow stays intact, at most gaining a pointer to the new page.
- Must not attempt a full audit or rewrite of the Spektacular repository's README beyond verifying the existing project/configuration section against current behavior and adding a cross-link.
- Must accurately reflect the multi-repo capability as it is currently shipped (registration, config split, cross-repo attribution, ignore files) — not aspirational or planned behavior.

## Acceptance Criteria

- [x] **Concept documentation for multi-repo projects**
  The documentation site contains a page that explains what a multi-repo project is, why it's useful, and how a repository is registered into one, using this repository's own multi-repo setup (or an equivalent example) to illustrate it.

- [x] **Configuration model explained as one coherent topic**
  The same page explains both the project-level and per-repo configuration concerns, and a reader does not need to leave the page (or jump to an unrelated page) to understand the full picture of how the two relate.

- [x] **Cross-repo workflow behavior explained**
  The page states that planning/implementation can span multiple registered repositories in one piece of work, and that each affected repository ends up with its own changelog entry naming the project and spec that produced it.

- [x] **Repository acquisition behavior explained**
  The page states that a repository registered by remote address is cloned automatically the first time it's needed, and explicitly states that Spektacular never fetches or pulls a repository that's already present without the user initiating it.

- [x] **Search/listing exclusion mechanism explained**
  The page describes the ignore-file mechanism for excluding paths from search/listing, and states that a directly named file is still accessible even if it matches an ignore pattern.

- [x] **New documentation is discoverable**
  A reader can navigate to the new page from the documentation site's primary navigation (e.g. a sidebar or menu entry) without already knowing its URL or having followed a link from another page.

- [x] **Existing tutorial does not contradict the new documentation**
  The getting-started tutorial contains a link or reference pointing to the new page at the point where a reader might otherwise assume a project is limited to one repository.

- [x] **Primary repository's documentation stays accurate and cross-linked**
  The Spektacular repository's README's project/configuration section is checked against current shipped behavior with no inaccuracies found (or any found are fixed), and it contains a link to the new documentation site page.

## Technical Approach

- New content lives in a new reference/concepts section, separate from the existing `tutorials/` content, since this is concept/reference material rather than a hands-on walkthrough.
- Use this repository's own `.spektacular/config.yaml` and `.spektacular/repo.yaml` (which already register a `docs` repo alongside itself) as the worked example in the documentation, since it's a real, live instance of the capability being documented.
- The README cross-link and the tutorial pointer should link to the new page rather than duplicating its content inline.

## Success Metrics

- A new user reading the documentation site can correctly explain, without consulting the source code, that a project can span multiple repositories, how configuration is split between project and repo, and how work gets attributed across repos.
- No metrics beyond the above are defined for this change — this is documentation content, not a product feature with usage or performance to measure, and correctness/completeness is already covered by the Acceptance Criteria.

## Non-Goals

- A hands-on, step-by-step tutorial for setting up a multi-repo project is not included — this spec delivers reference/concept documentation only.
- Translating or localizing the new documentation is out of scope.
