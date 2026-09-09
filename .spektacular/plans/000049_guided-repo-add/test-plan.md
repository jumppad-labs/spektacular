---
created_date: "2026-09-07"
status: completed
closed_date: "2026-09-07"
---

# Test Plan: 000049_guided-repo-add

Three of this feature's five success metrics cannot be settled by an automated test, because
each is an observation about people rather than a property of the code. They are written up here
as procedures someone else can run without reading the source.

The other two are covered automatically and are **not** repeated here:

- *A typical add takes one value in the user's own words* is asserted by the transcript suite
  (`tests/harbor/repo-workflow/`).
- *An add no longer forces a choice against an in-progress spec or plan* is asserted twice over,
  by `TestRepoGuidedAdd_RunsToCompletionBesideAnInProgressSpec` in `cmd/cross_kind_test.go` and
  by the transcript suite's absence-of-conflict check.

**Before any of the procedures below, note one outstanding item.** The transcript suite has been
written and its rules proven against synthetic transcripts, but it has never been executed
against a live agent. Run it once before treating the two automated metrics as met:

```bash
make harbor-test-repo            # answers each question in turn
make harbor-test-repo-delegated  # hands the whole set over at the first proposal
```

Both need container tooling and agent credentials, and each drives a real agent for up to ten
minutes. Results land in `tests/harbor/jobs/`.

---

## 1. Proposed values are accepted far more often than corrected

**What to measure.** Across real adds, the proportion of proposed values the user accepts
unchanged, per field. The metric is met when acceptance clearly exceeds correction for every one
of name, description, role and tags. A single field corrected more often than accepted is the
signal the specification names: the examination is drawing on the wrong thing for that field.

**How.** This is a tally over real use, not a single run. For each add performed by a real person
against a real repository, record one row: the repository, and for each of the four fields
whether the user accepted the proposal, edited it, or replaced it outright. Ten adds across at
least four genuinely different repositories is enough to see a pattern; fewer than five tells you
nothing. Deliberately include at least one repository with no README and one whose folder name
differs from what its manifest calls it.

A cheap way to gather the raw material without interrupting anyone: after each add, run

```bash
go run . repo list
```

and compare the recorded `description`, `role` and `tags` for that repository against what the
agent proposed in the conversation.

**Expected result.** For every field, accepted outnumbers corrected. Record the per-field counts
rather than only the total, because the total can look healthy while one field is consistently
wrong.

**If it fails.** Do not widen what the examination reads. Change which of the already-permitted
sources that field prefers and in what order, for instance preferring a manifest's declared
description over the README's opening prose, or the reverse. The permitted sources are the
README's opening prose, the identity and description a manifest declares, the top-level entries,
and the languages present. If no ordering of those produces usable suggestions for a field, stop
and raise it: the remedy would be a change to the specification's boundary on the examination,
which is not an implementation decision.

**Who and when.** The maintainers, over the first two weeks of real use after release.

---

## 2. An add takes under a minute of the user's attention

**What to measure.** Elapsed attention, from the first question to the confirmation, for a user
who accepts the proposals. Under one minute. This is the user's attention, not the wall clock:
time spent waiting on the agent's own latency does not count against it, and neither does time
spent reading this documentation.

**How.** Sit with someone adding a repository they know, and time from the moment the agent asks
which repository to add until they answer the confirmation. Use a stopwatch, not timestamps in a
log, because what is being measured is how long they were engaged. Do it for three different
people and three different repositories.

Do not measure this from an automated run. The container suite's elapsed time measures the agent
and the container, not a person's attention, and would report a number that means nothing here.

**Expected result.** Median under sixty seconds, with no run over ninety.

**If it fails.** Look first at how many questions were actually asked. The flow is designed to ask
one thing cold and propose the rest; a run that takes appreciably longer usually means either the
examination found nothing and every question was asked from scratch, or the agent batched or
re-asked questions the instructions told it to ask once. Check the second case against the
instructions in `templates/steps/repo/` before changing anything.

**Who and when.** Whoever is preparing the release, before it ships.

---

## 3. A second or third add needs no documentation

**What to measure.** Whether a user who has added one repository can add another without opening
the documentation and without asking what a question means. This is the clearest single signal
that the conversation explains itself.

**How.** Recruit two or three people who have used Spektacular but have not added a repository
before. Ask each to add a repository, then to add a second one. Watch, and record only two
things: every time they open documentation, and every time they ask what something means. Do not
prompt, do not explain, and do not answer their questions until the run is over.

Pay particular attention to the placement question, which most users will never see, and to the
tags question, which is the one most likely to read as jargon.

**Expected result.** On the second add, zero documentation lookups and zero questions about what
is being asked. On the first add, one or two is acceptable.

**If it fails.** The remedy is the wording of the specific instruction under
`templates/steps/repo/`, not the shape of the flow. Note the exact question that prompted the
lookup, since the instruction that produced it is the thing to change.

**Who and when.** The maintainers, once, before release, and again after any change to the
wording of the questions.
