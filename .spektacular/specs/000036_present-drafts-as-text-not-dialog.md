# Feature: 000036_present-drafts-as-text-not-dialog

## Overview

Spektacular's spec, plan, and implement workflows regularly ask the assistant to draft content (design options, a plan section, a changelog summary) and then get the user's confirmation on it. Today's instructions don't say how to present that draft, and the assistant sometimes puts the drafted text itself inside a structured yes/no or multiple-choice UI element — which visually truncates or compresses long text, making it hard for the user to actually read what's being proposed. This feature adds a standing instruction so the assistant always shows drafted content as normal readable text first, reserving the structured question UI purely for the short "does this look right?" confirmation that follows. Users benefit by always being able to read full drafts clearly, regardless of which workflow step produced them.

## Requirements

- [x] **Drafted content is shown as plain, readable text**
  When the assistant produces a draft of substantial content for the user to review (e.g. a set of options, a written section, a summary), it presents that draft as normal conversational text, not compressed or truncated inside a structured choice/dialog element.

- [x] **Confirmation is asked separately, after the draft is shown**
  Once a draft has been shown in full, the assistant may then ask the user a short, direct confirmation question (e.g. yes/no or a small set of choices) — but only after the content itself has already been presented as readable text, never as a substitute for showing it.

- [x] **This applies consistently across every workflow that follows a draft-then-confirm pattern**
  The behavior isn't limited to one workflow — any step in any Spektacular-driven workflow that asks the assistant to draft something and get the user's agreement follows this same presentation rule.

## Constraints

- Must be delivered as a managed AGENTS.md section (the same durable, per-project-installed pattern already used for "Memory & Context" and "Spec-Worthy Discussion Recognition"), not as edits scattered across individual workflow step templates.
- Must not change the observable behavior or output of the existing "Memory & Context" and "Spec-Worthy Discussion Recognition" sections — any refactor of their shared installation code must be behavior-preserving for those two.
- Must apply to every agent surface that already receives AGENTS.md (Claude, Codex, Bob) — not just one.

## Acceptance Criteria

- [x] **A draft appears as ordinary chat text, not inside a choice UI**
  When the assistant has substantial content to propose (e.g. more than a one-line answer), that content appears in the assistant's regular message text, in full, before any structured choice/confirmation element is shown.

- [x] **A confirmation question follows the draft, not instead of it**
  After a draft is shown as text, if the assistant asks for confirmation, that confirmation is a separate, short question (e.g. "does this look right?") — the draft's content itself never appears truncated or summarized inside the confirmation element.

- [x] **The instruction is discoverable from a project's agent-facing guidance regardless of which workflow step is running**
  A person reading the project's standing agent instructions (not a single workflow's step-by-step prompts) can find this presentation rule stated once, and it isn't contradicted or silently overridden by any individual spec, plan, or implement step's own wording.

## Technical Approach

- Refactor the existing "Memory & Context" and "Spec-Worthy Discussion Recognition" section installers, which are structurally near-duplicates of each other, into one shared, generic installer before adding this feature's new section as a third caller of it — rather than adding a third near-identical implementation.

## Success Metrics

No success metrics have been defined for this feature.

## Non-Goals

- Retroactively fixing any past conversation or transcript where content was already shown inside a dialog is out of scope — this is a forward-looking instruction, not a corrective action on history.
