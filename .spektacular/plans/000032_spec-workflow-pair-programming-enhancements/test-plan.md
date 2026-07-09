# Test Plan: 000032_spec-workflow-pair-programming-enhancements

The unit test suite (`internal/config/config_test.go`, `internal/agent/spec_trigger_test.go`) fully covers the config field's validation/default behavior and the installer's idempotency contract. The procedures below cover what those tests cannot: whether an agent actually reads and follows the instruction prose in `templates/agents/spec-trigger.md` once it's installed into a real project's `AGENTS.md`.

## Setup common to all procedures

1. In a scratch directory, run `go run . init claude` (or `codex` / `bob`) to install the managed `## Spec-Worthy Discussion Recognition` section into `AGENTS.md`.
2. Confirm the section is present: `grep -A5 "Spec-Worthy Discussion Recognition" AGENTS.md`.
3. Start a fresh session with the target agent (Claude Code, Codex, or Bob) rooted in that directory.

## Procedure 1 — Recognition and offer timing (covers success metric 1: "offered at a point that feels natural")

**What to measure**: Whether the agent offers to capture a discussion as a spec once it reaches a scoped, multi-requirement description, and not noticeably before or after that point.

**How**: With `spec_trigger_threshold` left unset (default `"moderate"`), open a conversation and incrementally describe a feature across several turns — start vague ("we should improve error handling somewhere"), then add 2-3 concrete requirements over subsequent turns (e.g. "actually, every command should return errors on the same channel", "and the exit code should always agree with it", "and failures should say what to do next"). Note which turn the agent first offers to capture a spec.

**Expected result**: The offer appears only after the discussion has accumulated multiple concrete requirements (not on the first vague turn), and no later than the turn where the description would already be spec-worthy to a human reading the transcript. Record the actual turn number and a short judgment: too early / about right / too late.

**Who / when**: A developer, once per supported agent (Claude, Codex, Bob), before merging or as part of periodic dogfooding.

## Procedure 2 — Threshold configurability changes offer sensitivity (covers acceptance criterion: "threshold is configurable")

**What to measure**: The same conversation shape triggers an offer under `"lenient"` but not under `"strict"`.

**How**: Set `spec_trigger_threshold: strict` in `.spektacular/config.yaml`. Start a new session and describe a small, single-requirement bug fix (e.g. "the timestamp parser breaks on single-digit days, can you fix that"). Note whether an offer appears. Then edit `config.yaml` to `spec_trigger_threshold: lenient` **without restarting the session or re-running `init`**, and repeat an equivalent small-fix description in a fresh conversation (or continue the same one if the agent re-reads config per-turn).

**Expected result**: Under `"strict"`, no offer for the small fix. Under `"lenient"`, an offer appears for the same shape of request. The change in behavior must be observed **without re-running `spektacular init`** — this specifically verifies the threshold is read live from `config.yaml` at decision time, not baked in at install time (per the instruction's explicit statement to that effect).

**Who / when**: A developer, once per supported agent, before merging.

## Procedure 3 — Carry-forward drafts from conversation instead of asking cold

**What to measure**: Whether accepting the offer produces a spec workflow where the agent proposes draft answers from the conversation rather than re-asking the same questions.

**How**: Have a conversation that already establishes an overview-level description of a feature (what's being built, what problem it solves, who benefits). When the agent offers and you accept, observe what happens at the spec workflow's `overview` step (`go run . spec goto --data '{"step":"overview"}'` is what the agent should run internally, per `templates/steps/spec/01-overview.md`'s existing "ask the user to describe this feature" prompt).

**Expected result**: The agent presents a drafted overview drawn from the conversation and asks you to confirm or refine it, rather than asking "what is being built, what problem does it solve, who benefits" as if starting cold. Confirm the final `.spektacular/work/<name>/overview.md` reflects the conversation's actual content once you either confirm or lightly edit the draft.

**Who / when**: A developer, once per supported agent, before merging.

## Procedure 4 — Defer vs. decline are tracked distinctly within one conversation

**What to measure**: A "not yet" allows a later re-offer in the same conversation; an explicit decline suppresses further offers for that same topic for the rest of the conversation.

**How**: Trigger an offer (per Procedure 1's setup) and respond "not ready yet, still thinking it through." Continue the same discussion, adding another requirement or two, and confirm the agent may offer again. In a separate, fresh conversation, trigger an offer and respond with an explicit decline ("no, I don't want a spec for this"). Continue discussing the same topic further and confirm the agent does not raise the offer again for the rest of that conversation.

**Expected result**: Deferral: offer may reappear later in the same conversation as the discussion develops. Decline: offer does not reappear for that topic, for the remainder of that conversation.

**Who / when**: A developer, once per supported agent, before merging.

## Procedure 5 — Behavior is scoped to Spektacular-initialized repositories

**What to measure**: The recognize-and-offer behavior does not appear in a repository that has never run `spektacular init`.

**How**: In a scratch directory with no `.spektacular/` directory and no `AGENTS.md` (or an `AGENTS.md` without the managed section), start a session with the target agent and have the same kind of multi-requirement discussion used in Procedure 1.

**Expected result**: No spec-creation offer appears, since the instruction was never installed.

**Who / when**: A developer, once per supported agent, before merging (out-of-repo control referenced by the plan's manual smoke test list).

## Ongoing dogfooding signal (covers success metrics 2 and 3: "rarely surprised", "rarely need to adjust the default threshold")

These two metrics are properties of usage over time, not a one-time procedure. During normal day-to-day use of Spektacular-initialized repositories (by the maintainers, post-merge):

- Note any instance where an offer felt surprising — either an offer appeared for work that clearly didn't warrant it (over-triggering), or a discussion that clearly should have prompted an offer didn't (under-triggering).
- Note any instance where a user manually changed `spec_trigger_threshold` away from the default `"moderate"` — and if so, why (too strict, too lenient, or a one-off preference for a specific repository's process).

If either kind of note accumulates noticeably during the weeks after merge, that's a signal the instruction prose in `templates/agents/spec-trigger.md` needs sharper recognition-criteria language, or that the default threshold value itself should change — not a signal to add code-side heuristics, per the plan's design decision to keep threshold interpretation as natural-language guidance.
