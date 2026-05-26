---
name: spek-knowledge
description: Search, contribute to, or update the project's knowledge base.
---

# What this skill does

This skill orchestrates the existing `{{command}} knowledge` CRUD surface for ad-hoc read, contribute, and update operations on the project's knowledge store, without starting a spec/plan/implement flow. Unlike `spek-new`, `spek-plan`, and `spek-implement`, it does not drive an interactive CLI state machine — it is a static playbook. The agent recognises the user's natural-language intent, picks one of three branches (lookup / contribute / update), and calls the matching `{{command}} knowledge` command directly.

# When to invoke

Invoke this skill any time the user references the knowledge base, an entry, a convention worth remembering, or asks a question that the knowledge store might already answer. Typical natural-language triggers include:

- "What do we know about X?"
- "Search the knowledge base for Y."
- "Remember that Z is the case." / "Add a note about Z."
- "Update what we have on W."
- "Recall the convention for V."

One skill handles all three intents. Discriminate by what the user actually said — do not ask the user to pick a slash command per intent.

# Intent: lookup

Triggered when the user wants to read or search existing entries.

1. Run `{{command}} knowledge search <query>` with a concise query derived from the user's question. The output is a list of hits, each tagged with its `scope` field (e.g. `project`, `global`).
2. Present results to the user with the **scope label visible** on every hit — never strip it. The user needs to know which configured store the content came from.
3. If the user asks for the full content of a particular hit, run `{{command}} knowledge read --data '{"scope":"<scope>","path":"<path>"}'` for that entry and present the body.
4. If the query returns no hits, say so plainly. Do not fall back to a write unless the user explicitly asks to add a new entry.

# Intent: contribute

Triggered when the user wants to record something new.

1. Run `{{command}} knowledge sources` to enumerate configured scopes. The output is the authoritative list of writable destinations.
2. Decide (or ask the user) which **scope**, **path**, and **body** to write. Path conventions follow the existing store layout — slug-style filenames under the scope's root.
3. Stage the body on disk under `.spektacular/tmp/<slug>.md` using the `Write` tool. Do not pipe the body via stdin; the only supported invocation is `--file <staged>`.
4. **Show the user the proposed scope, path, and body before writing.** Wait for **explicit confirmation** ("yes", "go ahead", or equivalent). If the user asks for changes, revise the staged body and re-show — never write on an implicit signal.
5. Only after explicit confirmation, run:
   ```
   {{command}} knowledge write --data '{"scope":"<scope>","path":"<path>"}' --file .spektacular/tmp/<slug>.md
   ```
6. Remove the scratch file after a successful write: `rm .spektacular/tmp/<slug>.md`.

# Intent: update

Triggered when the user wants to revise an existing entry.

1. Identify the target entry. Run `{{command}} knowledge search <query>` (or read the user-supplied path directly) to locate it. Confirm the scope and path with the user if there is any ambiguity.
2. Read the current body with `{{command}} knowledge read --data '{"scope":"<scope>","path":"<path>"}'`.
3. Apply the user's revision intent to produce new content. Stage the revised body under `.spektacular/tmp/<slug>.md` using the `Write` tool.
4. **Show the user the scope, path, and proposed new body (or a diff against the current body) before writing.** Wait for **explicit confirmation**.
5. Only after explicit confirmation, run:
   ```
   {{command}} knowledge write --data '{"scope":"<scope>","path":"<path>"}' --file .spektacular/tmp/<slug>.md
   ```
   The scope and path must match the original — that is what makes this an update rather than a new entry.
6. Remove the scratch file after a successful write: `rm .spektacular/tmp/<slug>.md`.

# Decline handling

If the user declines, asks for changes, or expresses uncertainty at any propose-then-confirm checkpoint, **do not invoke `{{command}} knowledge write`**. Either loop back to refine the proposal — adjust scope, path, or body and re-show — or stop and leave the knowledge store untouched. Removing the staged scratch file at `.spektacular/tmp/<slug>.md` is fine either way; a half-finished proposal should not linger on disk.

The propose-then-confirm contract is enforced by this prose, not by a CLI guard. Treat it as load-bearing: a write without explicit user approval is a bug in the skill's execution, not an acceptable shortcut.
