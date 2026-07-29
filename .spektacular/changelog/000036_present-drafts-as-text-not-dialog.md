# Present Drafts as Text, Not Dialog

## What was built

Coding agents working in a Spektacular-initialized project (Claude, Codex, or Bob) now follow a standing rule: whenever they draft substantial content for a user to review — an architecture write-up, a set of options, a written section, a summary — they show that draft as normal, readable chat text first, in full. Only after the draft has been shown do they ask a short, direct confirmation question (e.g. "does this look right?"), and only then via a structured yes/no or multiple-choice element if one is used. The draft itself is never embedded inside that structured element, where long text would be truncated or compressed and hard to read.

This was delivered as a third managed `AGENTS.md` section — "Presenting Drafts and Confirmations" — installed alongside the two that already existed ("Memory & Context" and "Spec-Worthy Discussion Recognition"), so every agent surface picks it up automatically and it's discoverable in one place rather than scattered across the ~20 individual workflow step templates.

Before adding the new section, the two existing managed-section installers — which were near-byte-identical, duplicated implementations — were consolidated into a single generic, shared installer (`installManagedSection`, in a new `internal/agent/managed_section.go`). Both existing sections were re-pointed to thin wrapper calls into this shared logic, confirmed behavior-preserving by diffing a fresh install run against this repository's own live `AGENTS.md`/`CLAUDE.md` (byte-identical) and by every one of the 10 pre-existing tests passing completely unmodified. The new third section then became a third, symmetric caller of the same shared machinery.

## Why it matters

Previously, nothing told an agent *how* to present a draft before asking for confirmation, so an agent would sometimes put the drafted content itself inside a structured decision/dialog UI element — which visually truncates or compresses long text, making it hard to actually read the full proposal (an architecture write-up, a set of milestones, a drafted spec section, etc.). This closes that gap: users of every Spektacular-managed project can now always read an assistant's full drafts clearly, regardless of which workflow step (spec, plan, or implement) produced them, and regardless of which of the three supported coding agents they're using.

## Deviations from the plan

None in scope or approach — all four phases (1.1: extract the generic installer, 1.2: re-point the two existing sections, 2.1: write the new template and installer, 2.2: wire it into all three agent surfaces) landed exactly as planned, with every phase's acceptance criteria met and verified.

One documentation-only drift was caught and fixed before any code was written: the plan referenced `templates/embed.go` as the file containing the `//go:embed all:*` directive that auto-picks-up new template files; the directive actually lives in `templates/templates.go` (`embed.go` doesn't exist as a file). The directive's described content and behavior were accurate — this was a filename typo in the plan, corrected via the plan store during the `read_plan` validation gate, with no effect on the implementation itself.

No requirements were descoped. The one deliberate, pre-existing scope boundary — noted in the spec and plan from the outset, not a deviation — is that no automated or manual test verifies an LLM's actual runtime behavior change (i.e., that a live agent session literally stops using dialog UI for drafts after reading the installed instruction); this is not mechanically testable from this repository, matching the existing test suite's scope for the other two managed sections.
