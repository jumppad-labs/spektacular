# Test Plan: spec-workflow-output-changelog

Both of this feature's success metrics require a real changelog record to exist and a human or independent-agent judgment on its quality — neither is expressible as an automated assertion. Both procedures below use the same real record: run the implement workflow against a real feature (spec 000032, `spec-workflow-pair-programming-enhancements`, is the intended first dogfood subject per this plan's sequencing note) through to completion, which produces `.spektacular/changelog/<plan-name>.md` via the `update_feature_changelog` step and `changelog file write`.

## Metric 1: The changelog record is sufficient on its own to write documentation or a blog post about the feature, without needing to re-read the spec, plan, or conversation

**What to measure**: Whether a reader who has only the changelog record (not the spec, plan, or original conversation) can accurately describe what was built, why it matters, and what it enables.

**How**:
1. Run the implement workflow to completion against a real feature: `go run . implement new --data '{"name":"<plan-name>"}'`, then step through to `finished`.
2. Retrieve the produced record: `go run . changelog file read <plan-name>.md`.
3. Hand the record's content (and nothing else — no spec, no plan, no conversation history) to a person or a fresh LLM agent with no other context, and ask them to write a 3-5 sentence user-facing summary of the feature from the record alone.
4. Compare that summary against the feature's actual spec `## Overview`/`## Requirements` sections (`go run . spec file read <plan-name>.md`).

**Expected result**: The summary produced from the record alone should accurately match the feature's actual scope and purpose — no material claim in the summary should be wrong, missing, or contradicted by the real spec. If the reader has to ask "wait, what does this actually do?" or "why was this built?", the record has failed this metric.

**Who / when**: The plan's author (or a designated reviewer), performed once after this plan's own implementation completes and its own changelog record exists — dogfooding the feature on itself is the first available real test case, before 000032 is planned and implemented.

## Metric 2: Users rarely need to supplement a changelog record with outside context when writing docs or announcements from it

**What to measure**: Over repeated real usage (multiple features implemented and their records produced), how often a person writing docs or an announcement from a changelog record has to go back to the spec, plan, or conversation to fill a gap.

**How**:
1. Over the next several features implemented after this plan ships, each time a changelog record is produced (`.spektacular/changelog/<name>.md`), track whether the person writing downstream documentation or a release announcement from it needed to open the spec, plan, or conversation to answer a question the record didn't cover.
2. Keep a simple tally: record used alone (no supplementation needed) vs. record insufficient (had to go back to source material).

**Expected result**: "Rarely" — the record-alone case should be the large majority. If insufficiency is common (roughly more than 1 in 5 uses), the record's authoring instructions (`templates/steps/implement/10-update_feature_changelog.md`) likely need revision to capture more of what's typically missing.

**Who / when**: Whoever writes documentation or release announcements, observed over the first handful of features implemented after this plan ships — this is inherently a trend assessed over time, not a single pass/fail check.
