---
created_date: "2026-08-14"
status: completed
closed_date: "2026-08-14"
---

# Feature: 000043_flipped-interaction-spec-interview

## Overview

Spektacular's spec workflow currently gathers requirements through a series of scripted questions, one small fixed set per section. This works well when the person writing the spec already knows exactly what they want, but breaks down when they're still working out the shape of the feature. This change lets the coding agent drive the conversation instead — asking open, adaptive questions about what's being built, who it's for, and what constraints apply, stopping once it has a solid enough picture to draft a first pass, rather than working through a fixed script. Each section of the spec is then drafted from that understanding and presented back for confirmation, and if the person writing the spec pushes back on a draft, the agent asks why instead of just taking the stated fix at face value — since a single correction can reveal a whole cluster of related needs the person hadn't thought to mention. People writing specs for features they haven't fully thought through yet get a spec that reflects what they actually need, not just what they thought to say when first asked.

## Requirements

- [x] **New interview phase before section drafting**
  Creating a spec begins with an open-ended conversation where the agent asks questions to understand the feature, rather than the first section immediately asking its own fixed questions.

- [x] **Interview asks adaptive, non-scripted questions**
  The questions the agent asks during the interview are not a fixed, predetermined list — they adapt to what the user has already said and pursue what is still unclear.

- [x] **Interview stops once there is no material ambiguity, not once every detail is known**
  The agent ends the interview when it judges it has enough understanding to draft a credible first pass, not when every possible requirement has been individually enumerated.

- [x] **Interview is aware of the project's registered repositories**
  The interview has access to the project's full set of registered repos and what each one is, not just the repo the conversation is currently focused on.

- [x] **Interview asks about cross-cutting impact on other repos**
  When a feature appears to be primarily about one repo, the interview asks whether it also requires changes in the project's other registered repos, with the question shaped by what each other repo is for (for example, asking whether documentation needs updating, or whether a CLI needs corresponding changes).

- [x] **Sections are drafted from the interview, then presented for confirmation**
  Each section of the spec is drafted from what the interview and any prior corrections established, and the user is asked to confirm or correct it rather than being asked to author it from scratch.

- [x] **Rejecting a drafted section triggers a follow-up conversation, not a blind edit**
  When a user indicates a drafted section is wrong, the agent asks questions to understand why before changing anything, rather than applying the user's stated correction verbatim without further inquiry.

- [x] **A single correction can produce more than one resulting change**
  The follow-up conversation triggered by a rejection can surface more than one change to the spec, not just a single one-for-one edit to the item originally mentioned.

- [x] **Corrections can amend sections other than the one being reviewed**
  Information surfaced while reviewing one section can update the content of a different section already agreed earlier, without requiring the user to separately revisit and re-approve that earlier section in the moment.

- [x] **Documentation explains the Flipped Interaction pattern by name**
  The documentation site describes what the Flipped Interaction pattern is and names it, rather than only describing Spektacular's implementation of it without attribution.

- [x] **Documentation walks through the new interview phase**
  The documentation describes what happens when a user starts a new spec, including that the agent asks adaptive questions before any section is drafted.

- [x] **Documentation is positioned as a differentiator**
  The documentation presents this behavior as a distinguishing capability of Spektacular, not merely as an internal implementation detail buried in a reference page.

- [x] **Documentation demonstrates the cross-repo awareness behavior with a concrete example**
  The documentation includes a worked example of the interview asking a cross-cutting question in a multi-repo project, illustrating the behavior rather than only describing it in the abstract.

## Constraints

- Must be scoped to the spec workflow only — the plan workflow's discovery/architecture steps are not changed by this feature.
- The existing per-section step structure and count must remain — this feature changes what happens within each step, not the sequence of steps itself.
- On a rejected section, the agent must not restart or re-open the initial interview.
- A section amended as a side effect of a correction elsewhere must not require its own separate re-confirmation step from the user in the moment — it relies on the existing end-of-workflow walkthrough as the review point.
- The interview must not be unbounded — it must reach a stopping point within a small number of exchanges rather than continuing indefinitely.

## Acceptance Criteria

- [x] **Interview runs before the first section is drafted**
  Starting a new spec produces an open-ended question from the agent about the feature, not a request to author the Overview section directly.

- [ ] **Interview questions vary with the answers given**
  Two spec-creation sessions describing different features produce different interview questions from the agent, rather than the same fixed question sequence both times.

- [ ] **Interview ends without exhausting every possible question**
  The agent moves to drafting after a small number of exchanges once the user's answers stop introducing new information, rather than continuing to ask about every conceivable aspect of the feature.

- [x] **Interview surfaces the project's repo roster**
  In a project with more than one registered repo, the interview's questions or the agent's stated reasoning reference at least one repo other than the one the feature is primarily about.

- [ ] **A single-repo-focused feature prompts a cross-repo question**
  In a project with more than one registered repo, when the described feature is framed around one repo, the interview asks at least one question about impact on another registered repo before concluding.

- [x] **Each section is presented as a draft awaiting confirmation**
  When a section step is reached, the user sees drafted content and a request to confirm or correct it, not a blank prompt asking them to write the section.

- [x] **A rejected section produces a follow-up question before any content changes**
  When the user rejects a drafted section, the agent's next message asks a clarifying question rather than immediately presenting revised content.

- [x] **A rejection can result in edits beyond the single item the user mentioned**
  At least one rejection-triggered conversation in a session results in more than one line of the spec changing as a result.

- [x] **A correction made while reviewing one section is reflected in a different, already-confirmed section without the user re-opening it**
  After a correction surfaces new information relevant to an earlier section, that earlier section's saved content includes the new information without the user being asked to revisit and re-confirm that section in the moment.

- [x] **Pattern name and source are documented**
  A documentation page names the Flipped Interaction pattern and states it is drawn from prior prompt-engineering research.

- [x] **Interview phase is documented with an example**
  A documentation page describes the interview phase and includes at least one example question or exchange.

- [x] **Documentation includes a multi-repo example**
  A documentation page includes an example showing the interview asking a cross-repo question in a multi-repo project (e.g. a CLI-and-docs project), not only a single-repo example.

- [x] **The capability is featured, not buried**
  The behavior is described somewhere a prospective user would encounter early — such as the homepage, README, or how-it-works page — not only in a deep reference page.

## Technical Approach

- The Flipped Interaction pattern is drawn from "A Prompt Pattern Catalog to Enhance Prompt Engineering with ChatGPT" (White et al., arXiv:2302.11382) — implementation should follow that pattern's structure (a stated goal, adaptive questioning toward it, an explicit stopping condition) rather than inventing a different interview shape.
- Cross-repo awareness should draw on the project's existing registered-repo information (name, role, description) rather than introducing a separate mechanism for the interview to learn what other repos exist.
- Prefer capturing the interview's findings in a git-tracked working file, consistent with how other spec-workflow sections already persist their draft content between steps.

## Success Metrics

- A user starting a spec for a feature they haven't fully thought through still ends up with a complete, well-formed spec, without the workflow producing a shallow draft based on a vague initial description.
- The number of back-and-forth corrections needed during section review decreases over time, because the interview surfaced most of what mattered upfront.
- A correction to one section catching a related gap in another section is visible in practice — evidence that the cross-section amendment behavior is actually firing, not just theoretically possible.
- Documentation of the pattern is something a prospective user would point to as a reason to choose Spektacular over a plainer spec-authoring tool.

## Non-Goals

- Applying the interview/Flipped Interaction behavior to the plan workflow's discovery or architecture steps — explicitly out of scope; the plan workflow's autonomous-drafting-plus-walkthrough model is unaffected.
- Documenting the rejection-follow-up behavior on the documentation site — the interview phase and cross-repo example are documented, but this specific behavior is not.
