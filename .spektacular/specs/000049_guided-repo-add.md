---
created_date: "2026-09-07"
status: completed
closed_date: "2026-09-07"
---

# Feature: 000049_guided-repo-add

## Overview

Adding a repository to a project becomes a guided conversation rather than a form to fill in. The user is asked one thing — which repository they want to add — and every remaining detail is proposed for them, drawn from the repository itself, so they confirm or correct a suggestion instead of supplying facts their own code already states. This fixes the two faults in today's experience: being interrogated for information the tool could simply read, and having Spektacular's internal vocabulary surface in what it says to them.

## Requirements

- [x] **Only the repository itself is asked for cold**
  The workflow asks the user to identify the repository being added — the folder its code lives in — and asks for nothing else without first proposing an answer. When the user has already named the repository in their request, the workflow does not ask again.

- [x] **Every remaining detail arrives as a proposed value**
  Before asking about the repository's name, description, role, or tags, the system examines the repository's own contents and states a concrete proposed value inside the question itself, such that a user who replies only with agreement has that value recorded unchanged.

- [x] **Questions are asked one at a time, in a fixed order**
  The workflow asks about name, then description, then role, then tags, as separate exchanges. It never presents them as one batched set of questions.

- [x] **The user can delegate the whole set**
  A user who declines to review the proposals individually has the system's proposed values recorded without further questions about them. The confirmation before writing still applies.

- [x] **Spektacular's internal vocabulary never reaches the user**
  Nothing the workflow says to the user names Spektacular's internal data-model terms, names the configuration files it writes, quotes the arguments or flags it passes, or justifies a choice by what one of its commands prints. Naming the user's own repository, the folder that will be created inside it, and where its code lives is expected and is not an exception to this — those are facts about their repository, not Spektacular's internals.

- [x] **Where Spektacular's files go is decided silently by default**
  The workflow places its files inside the repository being added, without asking, whenever that repository's folder can be written to and the user has not said otherwise. It asks whether a folder may be added only when the folder cannot be written to, or when the user has stated during the flow that the repository should receive nothing but code. If the user declines, the files are placed in a folder under the project instead and the repository is left unmodified.

- [x] **The user confirms in their own terms before anything is written**
  Before writing to disk, the workflow states what is about to happen — which repository is being registered, which folder will be created, and where its code lives — and waits for explicit confirmation.

- [x] **A guided add always produces descriptive metadata**
  A repository registered through the guided workflow has a description, a role, and tags recorded. The warning that the system emits for repositories lacking descriptive metadata does not occur for one added this way.

- [x] **An interrupted add resumes where it stopped**
  An add interrupted partway through resumes at the point it stopped, retaining every answer already gathered, without re-asking questions the user has already answered.

- [x] **An add can run alongside an in-progress spec or plan**
  Users can start and finish adding a repository while a spec, plan, or implement workflow is in progress. Neither disturbs the other, and the interrupted workflow resumes normally afterwards.

- [x] **The direct, non-interactive add is unaffected**
  A caller that already knows every detail can still register a repository in a single command, without entering the guided flow and without any change to how that command behaves today.

- [x] **The published documentation describes the guided add**
  The project documentation covering multi-repo projects and repository registration describes adding a repository as a guided flow that asks one question at a time and proposes an answer to each, and states that the single-command form remains available for callers that already know every detail.

## Constraints

- **Must follow the convention established by the spec and plan workflows.** The guided add must be a workflow owned by the CLI, not a static playbook the agent improvises from: the CLI holds the state and returns one instruction per step, the agent performs that step and advances the workflow, and the loop runs until the workflow reports itself finished. This is a mandated mechanism, chosen explicitly over adopting the interaction style alone.

- **Must not compete with spec, plan, or implement for workflow state.** The project's existing single workflow slot is shared by those three. A guided add must not occupy it, must not be blocked by a workflow already in it, and must not disturb one that is in progress.

- **Must not change the existing non-interactive add.** Registering a repository in a single command, supplying every detail at once, must keep working exactly as it does today — same behaviour, same output. Scripts, tests, and other callers depend on it.

- **Must not require a repository to be readable to be added.** Proposing a name, description, role, and tags depends on examining the repository, but a repository that offers nothing to read must still be addable; in that case the questions are asked without proposals rather than the add failing.

- **Must not perform deep code analysis to produce its proposals.** The examination is limited to what a repository states about itself — a README, the manifest its language uses, the top-level layout, and the languages present — and must stay fast enough to sit between two questions in a live conversation.

- **Must confirm before writing to a repository.** Registration creates a folder inside a repository the user owns. The user's explicit confirmation is required before anything is written, and declining must leave both that repository and the project registry untouched.

- **Must keep the existing registry, source resolution, and repair behaviour unchanged.** How sources are declared, resolved, cloned, and reported, and how a broken registration is repaired, are outside this change and must continue to behave as they do now.

- **Documentation must ship with the behaviour.** The published documentation covering multi-repo projects and repository registration must describe the guided add, and must not be left describing a flow that no longer exists.

## Acceptance Criteria

- [x] **The first question names nothing but the repository**
  Starting a guided add with no repository named produces exactly one question, asking which repository to add. No question about name, description, role, tags, or file placement appears before that question is answered.

- [x] **A repository named up front is not asked for again**
  Starting a guided add from a request that already names the repository's folder produces no question about which repository to add; the flow opens on the name proposal instead.

- [x] **Every question after the first states a proposed value**
  In a completed transcript, each question about name, description, role, and tags contains a specific proposed value, and replying with agreement alone records that value unchanged.

- [x] **Proposals are drawn from the repository, not the folder name**
  For a repository whose README and manifest describe it, the proposed description and tags contain terms that appear in those files and not merely a restatement of the folder name.

- [ ] **Questions arrive one per exchange**
  A completed transcript shows name, description, role, and tags each asked and answered in separate exchanges, in that order. No single message asks for more than one of them.

- [x] **Delegation records the proposals**
  Answering the first proposal with "just use what you think" produces no further questions about name, description, role, or tags, and the repository registered afterwards carries the values the system had proposed.

- [ ] **No internal vocabulary appears in the transcript**
  In the messages shown to the user during a completed guided add, none of the following appears: the words footprint, source, provider, colocated, or separate used as terms of art; the name of any configuration file the system writes; any command name, argument, or flag; or a justification of the form "that is what <command> reports".

- [x] **The default placement is taken without a question**
  Adding a writable repository produces no question about where Spektacular's files go, and afterwards a folder for them exists inside that repository.

- [x] **An unsuitable placement is raised in the user's terms**
  Adding a repository the user has said should receive nothing but code produces a question asking whether a folder may be added to that repository, phrased without the placement terms listed above. Answering no results in a registration whose files live in a folder under the project, and the repository itself is left with no new folder.

- [x] **Nothing is written before confirmation**
  At the point the confirmation is presented, the target repository has no new folder and the project's registry is unchanged. Declining at that point leaves both untouched.

- [x] **The confirmation states repository, folder, and code location**
  The confirmation message names the repository being registered, the folder that will be created, and where the code lives, and contains no command text or argument list.

- [x] **A guided add records complete metadata**
  After a guided add completes, listing the project's repositories shows the new entry with a non-empty description, role, and tags, and emits no missing-metadata warning for it.

- [x] **An interrupted add resumes without repeating itself**
  An add interrupted after the description is agreed and then resumed continues at the next unanswered question, and the values already agreed are present in the final registration without having been asked for a second time.

- [x] **An add completes while a spec is in progress**
  With a spec workflow in progress, a guided add runs to completion without reporting a workflow conflict and without requiring the spec to be discarded. Afterwards the spec resumes at the step it was on, with its gathered content intact.

- [x] **The direct add is unchanged**
  Registering a repository by supplying every detail in a single command succeeds without entering the guided flow, and produces the same registration and the same output as it does today.

- [x] **Documentation describes the guided flow**
  The published documentation page covering multi-repo projects states that a repository is added through a guided flow that asks one question at a time and proposes an answer to each, states that the single-command form remains available, and contains no instruction to supply name, description, role, or tags as command arguments during an interactive add.

## Technical Approach

- The step templates belonging to the spec and plan workflows are the closest available reference for how each step of a guided add should read: an opening step that gathers, later steps that draft from what was gathered and present the draft back for confirmation, a standing behaviour for what to do when the user rejects a draft, and a terminal step that reports the work finished. Those step templates are the natural home for this behaviour.

- The interaction style those workflows use is the Flipped Interaction pattern (White et al., "A Prompt Pattern Catalog to Enhance Prompt Engineering with ChatGPT", arXiv:2302.11382): the agent drives toward a stated goal with adaptive questions and an explicit stopping condition, rather than asking the user to author each answer from a blank prompt. A guided add is a narrower instance of the same idea — the goal is a complete, accurate registration, and the questions are drawn from what the repository already says about itself.

- Keeping the examination fast matters as much as keeping it accurate, because the user is waiting on it between the first question and the second.

- Prefer ordering the questions so each proposal can build on the answers before it, which is why name comes first: a confirmed name gives the description something to be about.

- The existing skill has already been edited in this direction — a standing rule about which vocabulary belongs to the agent rather than the user, and an eight-step add flow. It is an illustration of the intended behaviour rather than a design to preserve.

- Known risk worth attention during planning: giving the add its own workflow state is the part of this change most likely to have consequences elsewhere, since the existing state is a single per-project slot that three workflows already share, and the resume and discard paths are built around that assumption.

## Success Metrics

- In a typical add, the user supplies one value in their own words — which repository to add — and answers every other question by agreeing with what was proposed.
- Proposed values are accepted without correction far more often than they are corrected; a field corrected more often than it is accepted indicates the examination is looking at the wrong things and needs revisiting.
- Adding a repository takes under a minute of the user's attention, from the first question to the confirmation.
- Users adding a second or third repository do so without consulting documentation or asking what a question means.
- Adding a repository partway through a spec or plan no longer forces a choice between the two.

## Non-Goals

- Inspecting the registry and repairing a broken registration keep their current shape. This change covers the add flow only; neither of those is being made guided.
- Repository removal remains a manual edit of the project's configuration. No command is being added for it.
- The existing behaviour that a stale clone produces a warning and nothing more is unchanged; no automatic fetching or pulling is being introduced.
- Characterising a repository by analysing its code is explicitly not in scope. The examination behind the proposals reads what a repository says about itself and no more.
- Spec, plan, and implement continue to share a single workflow slot among themselves and to collide with one another exactly as they do today. Only the add is being taken out of that contention.
- Other skills keep whatever vocabulary they use today. The rule that Spektacular's internals belong to the agent rather than the user is being applied to the add flow, not swept across the rest of the skill surface.
