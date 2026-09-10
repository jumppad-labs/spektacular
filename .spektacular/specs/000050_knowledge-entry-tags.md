---
created_date: "2026-09-10"
status: completed
closed_date: "2026-09-10"
---

# Feature: 000050_knowledge-entry-tags

<!--
  OVERVIEW
  A concise 2-3 sentence summary of the feature. Answer three questions:
    1. What is being built?
    2. What problem does it solve?
    3. Who benefits and why does it matter?
  Avoid implementation details — this should be readable by any stakeholder.
-->
## Overview

Knowledge entries can carry tags that say what an entry is about, independently of the words its prose happens to use. Today an entry is found only if it shares vocabulary with whatever search term the coding agent invents at the time, and that term varies between runs of the same work — so an entry that is exactly on topic can be missed entirely, and the agent plans without knowledge the team has already written down. Tags are proposed by the agent at the moment an entry is captured, drawn from tags already in use, so the knowledge base stays findable as it grows without anyone maintaining a taxonomy by hand.


<!--
  REQUIREMENTS
  Specific, testable behaviours the feature must deliver.
  Format: bold title on the checkbox line, detail indented below.
  Rules:
    - Use active voice: "Users can...", "The system must..."
    - Each requirement should be independently verifiable
    - Focus on WHAT, not HOW — avoid prescribing implementation
    - Keep each item atomic — one behaviour per line
-->
## Requirements

- [x] **Entries carry optional tags**
  A knowledge entry can declare tags describing what it is about. An entry without tags remains valid and is unaffected by this feature.

- [x] **Tags retrieve an entry on their own**
  An entry tagged `go, http` is returned for the query `go http router` on the strength of those tags, whether or not those words appear in its body.

- [x] **A tag match outweighs a body occurrence**
  Given two entries of comparable length that each match the same single query term once, the entry matching it by tag ranks above the entry matching it only in prose.

- [x] **Search returns entries matching any query term, not only all of them**
  An entry carrying evidence for some query terms is returned and ranked, rather than excluded for missing the others.

- [x] **Score rises with the share of query terms matched**
  An entry matching one term of three ranks below an otherwise comparable entry matching all three.

- [x] **Weak matches fall away relative to strong ones**
  An entry matching one of three query terms is returned when it is the only match, and is not returned when an entry matching all three is also present.

- [x] **Repeated occurrences of a term give diminishing returns**
  A long entry mentioning a term many times does not outrank a short entry that is about that term.

- [x] **Users can narrow a search to tagged entries**
  A search can be restricted to entries carrying given tags; an entry without those tags is never returned, however well it would otherwise score.

- [x] **The capture flow proposes tags**
  When an entry is captured, tags are proposed alongside the tier, store, path and content, and are shown for confirmation or editing before anything is written.

- [x] **Proposed tags prefer the vocabulary already in use**
  Tag proposals draw on tags already present in the stores in scope, and propose a new tag only where no existing one fits.

- [x] **Retrieval behaviour is documented for users**
  Published user documentation explains how results are ranked, what tags contribute to ranking, and how to narrow a search by tag.

- [x] **Users can audit the tags on existing entries**
  The knowledge skill can review entries in scope, report tags that an entry's content does not support, and propose tags that are missing. Every change is proposed for confirmation and never written automatically.

- [x] **The ranking design is recorded in the project's knowledge base**
  How ranking works is captured as a knowledge entry, so a later planning run retrieves it rather than re-deriving it from the code.


<!--
  CONSTRAINTS
  Hard boundaries the solution must operate within. These are non-negotiable.
  Format: one bullet point per constraint.
  Examples:
    - Must integrate with the existing authentication system
    - Cannot introduce breaking changes to the public API
    - Must support the current minimum supported runtime versions
  Leave blank if there are no constraints.
-->
## Constraints

- Untagged entries must remain valid and searchable, and upgrading must require no migration step, no backfill, and no command to run.
- Nothing may be written to a knowledge store without explicit user confirmation; tags are proposed and confirmed like every other field, never persisted silently.
- Conventions and glossary must remain excluded from search results, so always-applied knowledge is never surfaced twice.
- The existing knowledge entry format must remain readable by anything already consuming it; adding tags must not break entries or callers that predate them.


<!--
  ACCEPTANCE CRITERIA
  The specific, binary conditions that define "done".
  Format: bold title on the checkbox line, verifiable detail indented below.
  Each criterion must be:
    - Independently verifiable (pass/fail, not subjective)
    - Traceable back to a requirement above
    - Testable by someone who didn't write the code
-->
## Acceptance Criteria

- [x] **An untagged knowledge base needs no migration**
  Given a knowledge base whose entries carry no tags, every knowledge command runs unchanged, no entry is rejected as invalid, and no migration step is required to upgrade.

- [x] **Tags alone retrieve an entry**
  Given an entry tagged `go, http` whose body contains neither word, the search `go http router` returns it.

- [x] **A tagged entry outranks a prose mention**
  Given two entries of comparable length, one tagged `http` and one merely mentioning "http" in its body, the search `http` returns the tagged entry first.

- [x] **A partial term match is returned, not excluded**
  Given an entry with evidence for one query term and none for the other two, the search returns it rather than omitting it.

- [x] **Fuller coverage ranks higher**
  Given two comparable entries, one with evidence for all three query terms and one with evidence for a single term, the search returns the three-term entry first.

- [x] **A weak match appears alone and disappears in company**
  Given an entry matching one of three query terms: when it is the only match, the search returns it; when an entry matching all three is also present, the search does not return it.

- [x] **Volume does not beat relevance**
  Given a long entry mentioning a term many times, and a short entry that is about that term, the search returns the short entry first.

- [x] **The tag filter excludes untagged and differently-tagged entries**
  Given a search narrowed to the tag `http`, every returned entry carries that tag and no entry without it is returned, regardless of score.

- [x] **Capture shows proposed tags before writing**
  When an entry is captured, the proposed tags are displayed with the tier, store and path, and no entry is written to a store until the user confirms.

- [x] **A proposal reuses an existing tag rather than a near-duplicate**
  Given a store already using the tag `http`, capturing an entry about HTTP proposes `http` rather than a variant such as `HTTP` or `http-api`.

- [x] **The documentation describes ranking, tags and the tag filter**
  Published user documentation states how results are ranked, how a tag affects ranking, and how to narrow a search by tag.

- [x] **An unsupported tag is flagged**
  Given an entry carrying a tag its content does not bear out, the audit reports that tag as unsupported.

- [x] **A missing tag is proposed**
  Given an entry whose content is clearly about a subject it carries no tag for, the audit proposes that tag, preferring one already in use over a new one.

- [x] **An audit writes nothing without confirmation**
  No entry is modified until the user confirms the proposed changes.

- [x] **A knowledge entry describes how ranking works**
  After delivery, the project's knowledge base contains an entry describing the ranking behaviour, and a search for that subject returns it.


<!--
  TECHNICAL APPROACH
  High-level technical direction to guide the planning agent. Include:
    - Key architectural decisions already made
    - Preferred patterns or technologies if known
    - Integration points with existing systems
    - Known risks or areas of uncertainty
  Format: one bullet point per direction/steer.
  Leave blank if you want the planner to propose the approach.
-->
## Technical Approach

- Express the weak-match cutoff relative to the strongest hit in the result set, rather than as an absolute score threshold, so it does not need retuning as a knowledge base grows.
- Weight a tag match well above a prose occurrence — enough to decide between entries that are otherwise comparable.
- Follow the existing `--tier` / `--filter` narrowing shape for the tag filter rather than inventing a new one, so narrowing a search stays one idea.
- Expect the weights, damping curve and cutoff fraction to be tuned during implementation rather than fixed by this spec.
- The ranking entry is a natural fit for the `architecture` category, written once the design settles. The rationale behind it is already recorded at `decisions/ranked-or-over-boolean-and.md`.
- The user-facing documentation lands on the project's documentation site, which is a registered repo in this project.


<!--
  SUCCESS METRICS
  How you will know the feature is working well after delivery. Be specific:
    - Quantitative: "p99 latency < 200ms", "error rate < 0.1%"
    - Behavioural: "users complete the flow without support intervention"
  Format: one bullet point per metric.
  Leave blank if not applicable.
-->
## Success Metrics

- **The `api` case reverses.** Searching a plausible surface name that appears nowhere in an entry's prose returns that entry on tags alone. Today `api` returns nothing for the routing entry; after this, it returns it.
- **The observed-query benchmark goes from 0 to 4.** Against this repo's knowledge base, the entry actually about a topic ranks first for all four queries agents have been seen to use for it: `http endpoints go`, `http handler`, `http endpoints handler routing`, and `api`. Measured before this change, the on-topic entry ranked first for none of them.
- **Newly captured entries carry tags.** Entries written through the capture flow after this ships have at least one tag, without anyone having been asked to design a taxonomy first.
- **The tag vocabulary converges rather than fragments.** As entries accumulate, most new entries reuse existing tags; the number of distinct tags grows visibly slower than the number of entries, and near-duplicates such as `http` / `HTTP` / `http-api` do not accumulate.


<!--
  NON-GOALS
  Explicitly state what this spec does NOT cover. This is as important as
  the requirements — it prevents scope creep and sets clear expectations.
  Format: one bullet point per exclusion.
  Examples:
    - "Mobile support is out of scope (tracked in #456)"
    - "Internationalisation will be addressed in a follow-up spec"
  Leave blank if there are no explicit exclusions to call out.
-->
## Non-Goals

- Full BM25-style scoring over the corpus is out of scope; this spec introduces diminishing returns on repeated occurrences, not a general relevance model (remainder of [#35](https://github.com/jumppad-labs/spektacular/issues/35)).
- Weighting title and heading matches above body matches as a general ranking signal is out of scope, beyond the tag weighting introduced here.
- Stemming and synonym expansion are not addressed. A word is not reduced to a root and no thesaurus is consulted, so `route` and `routing` remain distinct terms.
- **Amended during planning (2026-09-10):** the original wording of this exclusion also ruled out plural handling and stated that `router` and `routers` remain distinct. Positional prefix matching with partial credit was added to the feature during the plan walkthrough, which relates a query term and a tag when one opens the other, scoring the match by the length of the shorter over the longer. `router` and `routers` therefore now match, at reduced strength. Prefix matching is the only relaxation of exact tag matching in scope; it is positional, not morphological. See the plan for the algorithm and its rationale.
- Bulk rename or merge of a tag across a store is out of scope. The audit proposes changes per entry, keeping a human in the loop on each one.
- No change to how always-applied knowledge is loaded.

