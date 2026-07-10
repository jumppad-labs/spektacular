# Feature: 000035_plan-walkthrough-conversation

## Overview

After Spektacular finishes generating an implementation plan, the user is currently just handed a set of documents to read on their own. This feature lets the assistant instead offer to walk the user through the finished plan as a live conversation — presenting the approach, how the work is broken into stages, and what's intentionally left out — the way one colleague would explain a plan to another face to face. The user can interrupt at any point to ask questions or request changes, which the assistant applies on the spot before continuing, and the conversation ends only once the user explicitly agrees the plan is right. This makes it possible for the user to fully understand and shape a plan through conversation, rather than needing to read dense documents alone to catch something they'd want changed.

## Requirements

- [x] **The assistant offers a walkthrough after a plan is complete**
  Once a plan is fully generated, the assistant offers the user a choice between reading it directly or having the assistant walk through it conversationally. The offer is made once and is not repeated if declined.

- [x] **The walkthrough presents the plan as a structured narrative, not a document recitation**
  When the user accepts, the assistant explains the plan the way a colleague would describe it aloud — covering the chosen approach and reasoning, how the work is broken into stages, and what has intentionally been left out — rather than reading section headings or content verbatim.

- [x] **The user can interrupt at any point to ask questions or request changes**
  During the walkthrough, the user can pause the assistant at any point to ask questions or request a change. The assistant is not required to complete an uninterrupted monologue before the user can respond.

- [x] **Requested changes are applied and reflected immediately in the plan**
  When the user requests a change during the walkthrough, the assistant updates the plan to reflect it and confirms the update before continuing, so the plan document and the conversation never fall out of sync.

- [x] **The walkthrough ends with explicit user agreement**
  The assistant does not consider the plan settled until the user explicitly confirms it is correct and they are ready to proceed. The conversation does not end implicitly.

## Constraints

- The walkthrough must be optional: declining the offer must not block or degrade completion of the plan workflow. Reading the plan document directly must remain a fully valid path with no missing step.
- Changes made during the walkthrough must be written to the same plan document the rest of the workflow reads and writes — not a separate copy or transient state.

## Acceptance Criteria

- [x] **Offer appears exactly once per completed plan**
  After a plan finishes generating, the assistant's next message to the user includes an offer to walk through the plan versus reading it directly. If the user declines, no further offer for the same plan appears later in the conversation.

- [x] **Accepting produces a narrative covering approach, stages, and exclusions**
  When the user accepts the offer, the assistant's walkthrough explicitly addresses three things before reaching a close: the reasoning behind the chosen approach, the ordered breakdown of stages of work, and what has been deliberately excluded — observable as distinct points made in the conversation, not verbatim section text copied from the plan document.

- [x] **A mid-walkthrough question or change request receives a direct response before the walkthrough continues**
  If the user asks a question or requests a change at any point during the walkthrough, the assistant's next message responds to that question or change directly, rather than continuing on to unrelated material first.

- [x] **A requested change is reflected in the plan document immediately, with confirmation**
  When the user requests a change, the plan document on disk reflects that change, and the assistant's response confirms the update, before the walkthrough resumes.

- [x] **The walkthrough only ends after the user gives explicit confirmation**
  The assistant does not state or imply the plan is settled until the user has given an explicit affirmative response (e.g. "yes", "looks good", "proceed") to a direct closing question. Silence, a topic change, or an ambiguous reply does not count as agreement.

## Technical Approach

- The walkthrough should be presented in a small number of natural beats (e.g. approach & reasoning, stage breakdown, scope boundaries) with brief pauses between them, rather than either a rigid per-stage sign-off gate or one uninterrupted monologue.
- When the user requests a change mid-walkthrough, prefer applying it immediately and resuming, rather than deferring all changes to the end — this keeps the conversation and the plan in sync as they discuss it.
- The integration point in the plan workflow — where and how the walkthrough is offered and triggered — is left to the plan workflow's discretion.

## Success Metrics

No success metrics have been defined for this feature.

## Non-Goals

- Reciting the plan document's sections or content verbatim is out of scope — the walkthrough is a narrative explanation, not a read-aloud of the document.
- Deferring all requested changes to the end of the walkthrough is out of scope — changes are handled as they come up, not batched.
- Cross-session or cross-plan memory of prior walkthroughs (e.g. skipping the offer for future plans because the user accepted or declined before) is out of scope — the offer-once behavior applies only within a single plan's completion, not across the project's history.
