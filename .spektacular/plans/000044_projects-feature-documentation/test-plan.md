---
created_date: "2026-08-14"
status: completed
closed_date: "2026-08-14"
---

# Test Plan: 000044_projects-feature-documentation

## Manual: New-user comprehension check

**Metric**: "A new user reading the documentation site can correctly explain, without consulting the source code, that a project can span multiple repositories, how configuration is split between project and repo, and how work gets attributed across repos." (spec Success Metrics)

**How**: A reader with no prior knowledge of Spektacular's source code visits `https://spektacular.dev/projects/` (or the built local page at `docs/dist/projects/index.html` after `npm run build`) and reads it top to bottom, without opening any file in the `spektacular` or `docs` repos. After reading, they should be able to state, in their own words:

1. That a Spektacular project can register more than one repository (not just the one it's colocated with).
2. That project-level configuration (`config.yaml`) holds only membership (which repos, where to find them), while each repository's own configuration (`repo.yaml`) holds its description, role, tags, deployment, knowledge sources, and changelog settings, and that this split exists because a repository's own file carries no reference back to any project, letting one repository belong to more than one project.
3. That planning and implementation work can span multiple registered repositories in one piece of work, and that every affected repository ends up with its own changelog entry naming the project and spec/plan that produced it.
4. That a repository registered by remote address is cloned automatically the first time it's needed, and that Spektacular never fetches or pulls an already-present repository on its own.
5. That `.spektacular_ignore` excludes paths from search and listing only, and that a directly named file stays accessible even if it matches an ignore pattern.

**Expected result**: The reader can state all five points above in their own words, using only the `/projects/` page (cross-links to `/configuration/`/`/repo-configuration/` may be followed for deeper per-key reference, but the five points themselves should not require leaving `/projects/`).

**Who / when**: A team member unfamiliar with the multi-repo capability's implementation, run once after the docs site is deployed (or against a local `npm run build` + `npm run preview` of the `docs` repo) and before considering this plan's documentation work final.
