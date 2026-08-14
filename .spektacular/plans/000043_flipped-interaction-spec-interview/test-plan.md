---
created_date: "2026-08-14"
status: completed
closed_date: "2026-08-14"
---

## Manual verification procedures

Three of the spec's success metrics are qualitative or longitudinal and cannot be asserted by a single automated run. The behavioural halves of these metrics (well-formed sections from a vague prompt; cross-section amendment firing) are already covered by the harbor end-to-end suite (`tests/harbor/spec-workflow/`), extended in Phases 2.3/3.1 of this plan — the procedures below cover only the parts that need a human judgement call or observation over time.

### 1. A genuinely vague starting point still produces a complete, well-formed spec

**What to measure**: whether a real spec-creation session, started from a vague one-line prompt (not the harbor harness's semi-detailed JWT scenario), produces a spec whose sections read as specific and complete rather than generic filler.

**How**: run `spektacular init claude` (or your preferred agent) in a scratch project, invoke the `/spek-new` skill, and describe the feature in one deliberately vague sentence — for example "we should make the CLI easier to use for new people." Let the interview run to its natural stopping point without volunteering extra detail beyond what a real user might say unprompted. Follow the workflow through to `finished`.

**Expected result**: the resulting `.spektacular/specs/<name>.md` has every section (`Overview` through `Non-Goals`) filled with content specific enough to plan from — no section reads as a restatement of the vague prompt or a placeholder. Each Requirements/Acceptance Criteria item is independently actionable. If any section reads as shallow or generic, that is a signal the interview stopped too early or asked the wrong questions, not a pass.

**Who / when**: the maintainer implementing this plan, immediately after this implementation is merged, using a real fresh project (not the harbor container). This is also the natural first real-usage session for the feature.

### 2. Back-and-forth corrections during section review decrease over time as the interview improves

**What to measure**: whether the number of rejection-repair exchanges per spec-creation session trends down across repeated real usage, as evidence the interview is front-loading the right questions rather than section review repeatedly catching gaps.

**How**: track, informally, across the next 5-10 real spec-creation sessions (not harbor test runs) how many times a drafted section is rejected and needs a follow-up conversation before being confirmed. A rough tally per session is sufficient — this is a trend signal, not a metric with a hard threshold.

**Expected result**: no single-session pass/fail; success is a downward trend in average corrections-per-session observed informally over the first month or so of real usage after this ships. If the count stays flat or rises, that's a signal the interview template's questions need revisiting, not that this implementation is broken.

**Who / when**: the maintainer, informally, across real usage in the weeks following release. Not a release gate.

### 3. The documentation reads as a genuine differentiator, not implementation-detail trivia

**What to measure**: whether a prospective user reading `how-it-works.mdx`'s Stage 1 section and the homepage's new "Flipped Interaction interview" card would point to the interview behavior as a reason to choose Spektacular over a plainer spec-authoring tool.

**How**: the mechanical half is already verified — `cd ../spektacular-website && npx astro check && npm run build` both pass cleanly against the new content (confirmed during Phase 3.1/3.2 of this plan). For the qualitative half: have at least one person who did not write this content read `https://spektacular.dev/how-it-works/` (or the local `npm run preview` build) cold, without prior context on this plan, and ask them to describe in their own words what happens when they start a new spec.

**Expected result**: the reader's own description should surface the interview, the fact that it adapts to what they say, and roughly when it stops, without prompting — evidence the page actually communicates the differentiator rather than only technically containing the right words. If the reader misses the interview entirely or describes it as a minor implementation detail, the positioning needs another pass.

**Who / when**: the maintainer plus at least one other reviewer, before or shortly after the docs site deploys this content to production.
