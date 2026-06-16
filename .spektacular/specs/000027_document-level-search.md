# Feature: 000027_document-level-search

<!--
  OVERVIEW
  A concise 2-3 sentence summary of the feature. Answer three questions:
    1. What is being built?
    2. What problem does it solve?
    3. Who benefits and why does it matter?
  Avoid implementation details — this should be readable by any stakeholder.
-->
## Overview

Knowledge search today returns one result per matching line and only finds an entry when
the query words appear together, in order, on a single line — so a search like "workflow
steps" comes back empty even though an article about workflow steps exists, while a
broad one-word search buries the right article under dozens of repetitive line-level
results. This feature makes search answer at the level searchers actually act on — the
document: a query matches a document when all of its words appear somewhere in it, and
each matching document is returned once, with its title, how strongly it matched, and
its most relevant excerpts. Anyone searching a project's knowledge base — a person at
the CLI or an AI agent deciding what to read during research — can tell at a glance
which entries are worth opening, instead of missing them entirely or wading through
noise.

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

- **Multi-word queries match at document level.** A query of several words matches a
  knowledge entry when every word appears somewhere in that entry, in any order and at
  any distance — the words do not have to be adjacent or on the same line.
- **Matching stays literal and case-insensitive.** Each query word is matched as a
  literal, case-insensitive substring of the entry's text, exactly as single-word
  matching behaves today.
- **One result per matching entry.** Each knowledge entry that matches appears exactly
  once in the results, no matter how many of its lines contain query words.
- **Each result identifies and previews its entry.** A result must carry enough for a
  reader to decide whether to open the entry and to open it directly: which knowledge
  source it came from, the entry's locator (usable with the existing read command),
  the entry's title, an indication of how strongly it matched, and at least one compact
  excerpt showing matched content in context.
- **Results are ranked.** Entries that match more strongly appear before entries that
  match more weakly, so the most relevant entry is visible first rather than buried.
- **Single-word discovery does not regress.** Any entry a single-word query finds today
  is still found by the same query after the change.
- **Empty and unmatched queries stay clean.** An empty query, and a query no entry
  matches, both return an empty result without error.
- **Existing exclusions and resilience are preserved.** Convention files never appear in
  results; binary files are skipped; an entry containing a pathologically long line
  neither fails the search nor blocks results from other entries.

<!--
  CONSTRAINTS
  Hard boundaries the solution must operate within. These are non-negotiable.
  Examples:
    - Must integrate with the existing authentication system
    - Cannot introduce breaking changes to the public API
    - Must support the current minimum supported runtime versions
  Leave blank if there are no constraints.
-->
## Constraints

- **Search must remain fully self-contained.** No external tool, no separate process,
  and no new runtime dependency may be introduced — the property delivered by
  000026_ripgrep-replace (a fresh install on a clean machine searches successfully with
  nothing else installed) must be preserved.
- **No index or background machinery.** Every search must remain a fresh scan of the
  knowledge sources at query time; the solution must not require building, persisting,
  or maintaining an index, cache, or daemon.
- **The read contract is untouched.** Results must remain consumable by the existing
  entry-reading command exactly as it is today; this feature must not require changes
  to how an entry is fetched once chosen.
- **All shipped consumers move in lockstep.** The result shape may change (that is the
  point), but every surface shipped with the project that consumes or documents search
  output — command output contracts, agent instructions, templates — must be updated in
  the same change, so nothing shipped still describes the old per-line results.

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

- **Scattered words still find the entry.** Given an entry in which every query word
  appears but never together on one line (e.g. an entry titled "Workflow Step
  Architecture" whose body also contains "steps"), searching for "workflow steps"
  returns that entry. Today the same search returns nothing.
- **A word missing from an entry excludes it.** Given a multi-word query where one word
  appears in an entry and another does not, that entry is not returned.
- **Exactly one result per entry.** Given an entry containing query words on many
  different lines, the search output lists that entry exactly once.
- **Results are self-describing and actionable.** Every result shows the knowledge
  source it came from, the entry's locator, the entry's title, a match-strength value,
  and at least one excerpt containing a query word; passing the result's source and
  locator to the existing read command returns that entry's full content.
- **Stronger matches rank first.** Given one entry where the query words occur many
  times and another where they occur once, the first entry appears before the second in
  the results.
- **No single-word regression.** For a fixed fixture of entries, the set of entries
  found by a single-word query is identical before and after the change.
- **Clean empty results.** An empty query and a query matching no entry both complete
  successfully and return an empty result list, not an error.
- **Exclusions and resilience hold.** A convention file containing every query word is
  absent from results; a binary file containing the query words is absent from results;
  when one entry contains a pathologically long line, the search still succeeds and
  other matching entries are returned.

<!--
  TECHNICAL APPROACH
  High-level technical direction to guide the planning agent. Include:
    - Key architectural decisions already made
    - Preferred patterns or technologies if known
    - Integration points with existing systems
    - Known risks or areas of uncertainty
  Leave blank if you want the planner to propose the approach.
-->
## Technical Approach

Direction discussed when the feature was identified (non-binding; the plan workflow
should validate or replace it):

- Evolve the existing in-process per-file scanner rather than rebuilding it: it already
  visits every file and collects per-line matches, so document-level matching is
  expected to be an aggregation over what one file's scan produces (which terms were
  seen, where, how often), not a new search engine.
- A result's title is expected to come from the entry's first heading, falling back to
  its locator when an entry has none.
- Match strength is expected to derive from term occurrence counts across the document;
  lines containing more distinct query terms are the natural candidates for the
  result's excerpts.
- Approximate matching was considered and deliberately excluded — see Non-Goals.

<!--
  SUCCESS METRICS
  How you will know the feature is working well after delivery. Be specific:
    - Quantitative: "p99 latency < 200ms", "error rate < 0.1%"
    - Behavioural: "users complete the flow without support intervention"
  Leave blank if not applicable.
-->
## Success Metrics

Measured against this project's own knowledge base, where the motivating failures were
observed:

1. **The motivating queries now succeed.** "workflow steps", "workflow state", and
   "interrupted workflow" — all of which return zero results today — each return the
   relevant architecture entry, with it ranked first.
2. **Broad queries become triageable.** A search for "workflow" returns one ranked
   result per matching entry (single digits on today's knowledge base) instead of the
   current 60 line-level results, and the entry with the most matches is listed first.
3. **No discovery regression.** For every single-word query, the set of entries
   returned is identical to the set of entries the per-line results referenced before
   the change.

<!--
  NON-GOALS
  Explicitly state what this spec does NOT cover. This is as important as
  the requirements — it prevents scope creep and sets clear expectations.
  Examples:
    - "Mobile support is out of scope (tracked in #456)"
    - "Internationalisation will be addressed in a follow-up spec"
  Leave blank if there are no explicit exclusions to call out.
-->
## Non-Goals

All four exclusions below were weighed against the success metrics and confirmed
unnecessary for this feature to succeed; each is a candidate follow-up, not a hidden
dependency.

- **No approximate matching** — no typo tolerance, fuzzy matching, stemming, or
  synonyms. Note the deliberate asymmetry that makes this acceptable: substring
  matching means a shorter query form matches longer words ("step" finds "steps",
  "stepping"), so stem-form query words are the recommended idiom; only the reverse
  ("steps" against an entry that only ever says "step") misses, and full stemming is
  deferred until that proves a problem in practice.
- **No query syntax** — no quoted exact-phrase operator, no OR/NOT booleans. A query is
  simply words that must all be present in an entry.
- **No filename matching** — matching is over entry content (including its title line)
  only; a query word that appears solely in an entry's file name or locator does not
  make the entry match. Entries' first headings mirror their filenames closely enough
  in practice; if real misses surface after delivery, this is the cheapest follow-up.
- **No smarter relevance, limits, or pagination** — match strength stays based on
  simple occurrence counts; no semantic or embedding-based ranking, no TF-IDF, no
  result caps or paging. Knowledge bases are small (an assumption standing since the
  search feature was first built) and occurrence counts already separate strong matches
  from incidental ones by a wide margin.
