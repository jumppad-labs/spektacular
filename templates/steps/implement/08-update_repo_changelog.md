## Step {{step}}: {{title}}

This is the user-facing release-note step, just before the test-plan step. Append a short, user-facing summary of the overall change to the repo-level `CHANGELOG.md` so downstream users and reviewers can see what shipped in a release note without reading the plan.

### What to write

Create or update `CHANGELOG.md` at the **repo root** (not inside `.spektacular/` or the plan directory). If the file does not exist, create it.

Prepend a new section above any existing sections, using this shape:

```
## {{plan_name}}

<2-4 sentence user-facing summary of what this plan delivered, written for a reader who has never seen the plan. No file paths, no internal package names, no implementation detail. Focus on the behavior change users will experience.>
```

Key rules:

- **Use `{{plan_name}}` as the section header.** It is the plan's slug and serves as the unique anchor for this release.
- **Prepend, don't append.** New entries go at the top of the file, above any existing sections, so the most recent change is always visible first.
- **Write for users, not implementers.** The inline `{{changelog_section_name}}` section of `{{plan_path}}` already has the phase-by-phase implementation audit log. The repo `CHANGELOG.md` is for humans scanning release notes.
- **One section per plan.** Do not add per-phase subsections here. If you notice a `## {{plan_name}}` section already exists at the top (perhaps from a partial previous run), replace it rather than creating a duplicate.

### STOP-on-mismatch

If `{{plan_name}}` is empty or looks like a placeholder, STOP and ask the user for the release-note header. Do not write a section with a broken or empty header.

### Advance

Once `CHANGELOG.md` has been updated:

```
{{config.command}} implement goto --data '{"step":"{{next_step}}"}'
```

---

**Before you advance:** refresh `.spektacular/context.md` with your cross-cutting working context only — the key decisions and substitutions made, the answers the user gave to your questions, and learnings worth carrying forward. Keep it to learnings and decisions, not a transcript and not a copy of content already captured elsewhere (such as a section's own working file). Use your own file tools. This file is git-tracked, and a resumed session reads it back to pick up where you left off, so keep it current every time before running the `goto` command above.
