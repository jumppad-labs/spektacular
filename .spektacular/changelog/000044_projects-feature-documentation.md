---
created_date: "2026-08-14"
status: completed
closed_date: "2026-08-14"
---

# Changelog: 000044_projects-feature-documentation

## What was built

The documentation site now reflects Spektacular's multi-repo "projects" capability (shipped in specs 000039 and 000042). Before this change, a reader following the published tutorial only encountered the older single-repo mental model and had no path to the newer feature. Three edits together close that gap:

- A new flat reference page (`docs: src/pages/projects.mdx`, route `/projects/`) explains the capability end-to-end: what a project is, why the multi-repo shape is useful, how a repository is registered, how project-level (`config.yaml`) and repo-level (`repo.yaml`) configuration relate to each other as one coherent topic, how planning and implementation work is attributed across every repo it touches, how a repository becomes locally available, and how paths can be excluded from a project's own search and listing. Built entirely from existing site components (`Hero`, `Section`, `ConfigurationKeys`, `ConfigKey`, `CtaBanner`, `Button`) with alternating section background shading. A "Projects" entry was added to the site's "Resources" navigation dropdown (`docs: src/components/Nav.astro`) so the page is reachable without needing a prior link.
- The getting-started tutorial (`docs: src/content/tutorials/getting-started.mdx`) gained one sentence, immediately after Step 4's closing paragraph, telling a reader following the single-repo walkthrough that a project can register additional repositories, and linking to `/projects/` for that case.
- The Spektacular repository's own `README.md` was corrected and cross-linked in its `## Configuration` section: the stale `description`/`role` lines were removed from the `docs` repo's entry inside the `config.yaml` example's `repos:` list (that metadata now lives exclusively in `repo.yaml` since spec 000042), the four descriptive fields (`description`, `role`, `tags`, `deployment`) were added to the `repo.yaml` example alongside its existing `knowledge`/`changelog` keys, and a cross-link sentence to `/projects/` was appended after the existing full-reference link.

## Why it matters / what it enables

The documentation-site gap meant every new user encountered the tool as a single-repo tool by default — the tutorial did not mention the multi-repo capability, no reference page described it, and the README's example silently disagreed with what the code actually accepted. Users hit the capability only by discovering it through configuration errors or by reading the source.

With a dedicated concept page reachable from navigation, a tutorial pointer that surfaces the option in-flow rather than after the fact, and a README example that matches the shipped configuration split, both new and returning readers can find and understand the multi-repo model as a first-class feature rather than as tribal knowledge. The README fix also removes a small but real source of drift: readers copying its example into a new project were carrying stale schema forward, which the code silently tolerated (unknown YAML keys are ignored) but which any future strict-mode validation would surface as errors.

## Deviations from the plan

None material. The Phase 1.1 closing `CtaBanner` links to `/configuration/` rather than the plan's illustrative `/install/`/`/plugins/` — the plan phrased those with "e.g.", and `/configuration/` follows more directly from the page's immediately preceding config-split section. An in-spirit deviation from an illustrative example, not a plan violation. Both YAML samples in the README fix were validated with `gopkg.in/yaml.v3` before commit.
