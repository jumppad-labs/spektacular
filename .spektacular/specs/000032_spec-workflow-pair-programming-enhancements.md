# Feature: 000032_spec-workflow-pair-programming-enhancements

## Overview

Right now, Spektacular only captures a specification when a user deliberately starts the spec workflow already knowing they need one. But real feature discussions often start as open-ended exploration — diagnosing a problem, kicking around ideas — and only partway through does it become clear the conversation has produced something substantial enough to deserve a spec. In that pattern, nothing currently prompts the assistant to notice the shift and capture the conversation before it's lost.

This change gives the assistant standing instructions to recognize when an in-progress discussion has crossed the threshold that warrants a specification, and to proactively offer to capture it — carrying forward what's already been decided so the user isn't forced to re-explain themselves from scratch. Where that threshold sits is not fixed: some organizations are comfortable vibe-coding small bug fixes with no formal process, while others want even minor fixes to go through a lightweight spec. The point at which the assistant should propose "let's capture this as a spec" is therefore a configurable policy, not a hardcoded judgment call, so different teams can dial the strictness of their process up or down to match how they work.

## Requirements

- [x] **Assistant recognizes when a discussion warrants a spec**
  During an open-ended conversation (diagnostics, brainstorming, exploratory discussion), the assistant must be able to recognize when the conversation has produced a decision or scope substantial enough to be worth capturing as a specification, rather than only doing so when the user explicitly invokes the spec workflow.

- [x] **Assistant proactively offers to capture the conversation as a spec**
  When the assistant recognizes that threshold has been crossed, it must proactively offer to start the spec process, rather than silently proceeding to implementation or waiting to be asked.

- [x] **Threshold for triggering a spec is configurable, not fixed**
  Users/organizations can configure how readily the assistant proposes creating a spec — ranging from "only for substantial new features" to "even small bug fixes should go through a lightweight spec" — and the assistant must honor that configured strictness when deciding whether to offer.

- [x] **A sensible default applies when no configuration is present**
  If no explicit configuration exists, the assistant must still apply a reasonable default threshold rather than either never triggering or always triggering.

- [x] **Starting the spec from a recognized moment carries forward already-established context**
  When the user accepts the offer to create a spec, the assistant must carry forward the relevant decisions, constraints, and information already established in the conversation, so the user is not forced to re-answer questions (e.g. overview) that the conversation has already effectively answered.

- [x] **The user can defer the offer while investigation continues**
  If the user responds that they're not ready yet (still investigating, not done exploring), the assistant must continue the conversation without a spec and may raise the offer again later in the same conversation as the discussion develops further, rather than treating a "not yet" as a permanent decline.

- [x] **The user can decline the offer outright**
  If the user declines spec creation for this discussion entirely, the assistant must drop the offer for the remainder of that discussion and not repeatedly re-prompt for the same topic.

## Constraints

- The trigger threshold must be user-configurable — the assistant cannot hardcode a single fixed sensitivity for offering a spec.
- A sensible default threshold must apply when the user has not configured one — the feature cannot ship with no behavior at all in the unconfigured case.
- The threshold setting must be stored in the project's existing configuration mechanism — it must not introduce a new, separate configuration file or system alongside it.
- Declining the offer must not be repeated for the same discussion — the assistant cannot keep re-prompting after an outright decline.
- Must not disrupt or replace the existing explicit spec-workflow entry point — the recognize-and-offer behavior is additive, alongside deliberate invocation, not a replacement for it.

## Acceptance Criteria

- [x] **Recognition of spec-worthy discussion**
  Given a conversation that has produced a scoped decision or feature description substantial enough to meet the configured threshold, the assistant's next response includes an offer to capture it as a spec, without the user having invoked the spec workflow themselves.

- [x] **Proactive offer, not silent progression**
  In that situation, the assistant does not proceed straight into implementation or further open-ended work without first surfacing the offer to the user.

- [x] **Threshold is configurable**
  A user can set the trigger sensitivity to a stricter or looser value than the default, and the assistant's offer behavior changes accordingly for the same conversation shape (e.g. a small bug-fix-sized discussion triggers an offer under a strict setting but not under a lenient one).

- [x] **Default threshold applies out of the box**
  With no configuration present, a conversation that reaches a substantial, multi-decision scope still results in an offer; a trivial one-line fix does not.

- [x] **Context carries into the spec**
  When the user accepts, the resulting spec's Overview (and other sections, where the conversation already established them) is pre-populated from the conversation, and the assistant does not re-ask questions the conversation already answered without first proposing its own draft based on that context.

- [x] **Deferral keeps the conversation open and allows re-offering**
  When the user defers, no spec workflow is started, the conversation continues normally, and the assistant may raise the offer again later in the same conversation if the discussion grows further.

- [x] **Decline suppresses further offers for that discussion**
  When the user declines, no spec workflow is started, and the assistant does not raise the offer again for the same discussion topic within that conversation.

## Technical Approach

- No technical direction has been decided beyond the captured constraints; the detailed design (heuristics for judging when a discussion is "substantial enough," how the offer is worded, how carried-forward context is structured for the spec workflow to consume) is left for the plan workflow to propose.

## Success Metrics

- Substantial discussions consistently get offered a spec at a point that feels natural rather than premature or too late.
- Users rarely feel surprised by the offer behavior — neither annoyed by over-triggering on trivial work, nor missing an offer they expected on substantial work.
- Users rarely need to adjust the default threshold, indicating the out-of-the-box default is well-calibrated.

## Non-Goals

- Polishing the requirements-gathering step's interview UX (propose-then-confirm drafting, surfacing rephrasing out loud, prompting for completeness gaps) — deferred to a separate future spec.
- Adding a user-acceptance walkthrough to the plan workflow's verification step — a separately identified gap, not addressed here.
- A durable changelog/context artifact for downstream doc/blog generation — deferred to its own separate spec, to be sequenced before this one's implementation.
- Recognizing spec-worthy discussion across multiple separate conversations or sessions — this feature operates only within a single, ongoing conversation.
- Automatically creating a spec without the user's explicit acceptance of the offer — the assistant always offers and waits for a decision; it never starts a spec workflow unilaterally.
