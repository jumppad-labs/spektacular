

# Agent Rules <!-- tessl-managed -->

@.tessl/RULES.md follow the [instructions](.tessl/RULES.md)

## Memory & Context

> Managed by `spektacular init` — edit `templates/agents/memory-context.md`
> in the Spektacular source, not this section in place. Hand edits will not
> survive the next init.

In this repository, do not persist anything to your per-user, per-machine
memory store. When you would normally write to it — a learning, convention,
gotcha, project fact, user preference, or anything else worth remembering
between sessions — route the write through the `spek-knowledge` skill
instead. The skill handles scope selection, search-before-write, and
propose-then-confirm.

Outside this repository, continue using your per-user memory store as normal.

## Spec-Worthy Discussion Recognition

> Managed by `go run . init` — edit `templates/agents/spec-trigger.md`
> in the Spektacular source, not this section in place. Hand edits will not
> survive the next init.

During an open-ended discussion (diagnostics, brainstorming, exploratory back-and-forth), watch for the moment it produces something substantial enough to be worth capturing as a specification — multiple requirements mentioned, a scoped decision reached, or a feature described in enough detail that it could be built from. Don't wait to be asked; recognizing this moment and offering is your job, not the user's.

Before deciding whether a discussion has crossed that line, read `spec_trigger_threshold` from `.spektacular/config.yaml` — check it at the moment you are deciding, not once at the start of the session, since the user may change it mid-conversation and expects that to take effect immediately. Treat a missing or absent value as `"moderate"`. Use the configured value to calibrate how readily you offer:

- `"strict"` — only offer for substantial, multi-requirement features. Small fixes and minor tweaks should not trigger an offer.
- `"moderate"` — offer once a discussion reaches a clear, scoped decision or a feature description with more than one requirement. The default balance.
- `"lenient"` — offer readily, including for small bug fixes or narrowly scoped changes, whenever there's a decision worth recording.

When you recognize the moment, always offer — never start the spec workflow unilaterally. Propose capturing the discussion as a spec, briefly say why (e.g. "this sounds like it's grown into a few concrete requirements — want me to capture it as a spec?"), and wait for the user's decision before doing anything else.

The user's response falls into one of three outcomes:

- **Accept** — proceed to carry-forward below.
- **Defer** ("not yet", "still investigating", "let me think") — do not start the spec workflow. Continue the conversation normally, and treat this as temporary: if the discussion keeps developing, you may raise the offer again later in the same conversation.
- **Decline** ("no", "I don't want a spec for this") — do not start the spec workflow, and do not raise the offer again for this discussion topic for the remainder of the conversation. A decline is final for that topic, not a "not now."

If the user accepts, start the spec workflow:

```
go run . spec new --data '{"name":"..."}'
```

Then drive its existing steps — the "ask the user..." prompts in `templates/steps/spec/*.md` are unchanged and still apply — but answer each one from the conversation you already had instead of asking cold. For every step where the discussion already established an answer (e.g. the overview step's "describe this feature in 2-3 sentences"), propose a draft based on what was said and ask the user to confirm or refine it, rather than posing the question as if from scratch. Only ask a step's question directly when the conversation genuinely didn't cover it. The user's confirmation or correction is still the final word — never silently record your own draft as accepted without it.

This behavior is scoped to the current, single conversation: it does not persist across sessions, and it does not apply outside a Spektacular-initialized repository.

## Historical Artifacts: Specs and Plans as Archaeology

> Managed by `go run . init` — edit `templates/agents/historical-artifacts.md`
> in the Spektacular source, not this section in place. Hand edits will not
> survive the next init.

In this repository, treat every file under `.spektacular/specs/` and
`.spektacular/plans/` as a historical, archaeological record. Each one
describes the intent behind a past change — *why* something was proposed,
what was in scope at that moment, and how the author framed the problem.
None of them are a description of what the codebase does today. Intent
recorded in a spec or plan may have been reshaped during implementation,
descoped, or abandoned entirely; only the shipped code, its tests, and its
configuration authoritatively describe current behavior.

Because of that, when you are exploring the codebase, summarizing a
feature, tracing how something works, or answering any question about
current-state behavior, do not read files under `.spektacular/specs/` or
`.spektacular/plans/` — through the `Read` tool, through `go run . spec
file read`, through `go run . plan file read`, or through any other
channel. Ground your answer in source files, tests, and configuration
instead, and cite paths under those directories rather than under the
spec or plan stores.

You may read a historical spec or plan only when the user is genuinely
investigating past intent — questions like "why was X built this way?",
"what was the original plan for Y?", or "which spec introduced Z?". In
that case, read the relevant document, and cite it explicitly as
historical context for a past decision rather than as a description of
current behavior. Archaeology is the only allowed reason to open these
files outside an active workflow.

There is one further exception: while a spec, plan, or implement
workflow is actively running, the workflow that owns its artifact may
read and update that artifact freely. That is what the workflow is for,
and it uses the dedicated CLI (`go run . spec file read/write`,
`go run . plan file read/write`) to do so. Once the workflow closes —
or for any agent that is not the workflow currently driving the
artifact — the artifact is historical again and subject to the same
rules as every other spec or plan on disk.

This rule applies everywhere you operate in the repository, not only
inside spec, plan, or implement workflow steps. It binds ad-hoc
questions, unrelated skills, and general exploration alike. Users
should not have to restate it in each session.

More broadly, the rest of `.spektacular/` — knowledge entries,
`context.md`, changelog records — is generated output *about* the
codebase, not the codebase itself. A broad grep or file scan run to
understand current-state behavior should treat all of `.spektacular/`
as out of scope, the same way it treats `.spektacular/specs/` and
`.spektacular/plans/` above, unless the task explicitly concerns specs,
plans, or knowledge.

## Presenting Drafts and Confirmations

> Managed by `go run . init` — edit `templates/agents/draft-presentation.md`
> in the Spektacular source, not this section in place. Hand edits will not
> survive the next init.

When you have drafted substantial content for the user to review — an architecture write-up, a set of options, a written section, a summary, a plan or spec excerpt — always present that draft as normal, readable chat text first, in full. Never embed the draft itself inside a structured yes/no or multiple-choice dialog element: that kind of UI truncates or compresses long text, making it hard for the user to actually read what you're proposing.

Once the draft has been shown in full as plain text, you may then ask a short, direct confirmation question — e.g. "does this look right, or should anything change?" — using a structured yes/no or multiple-choice element if one is available. That confirmation step comes strictly after the content has already been presented as text, and is never a substitute for showing it.

This rule applies across every workflow step that follows a draft-then-confirm pattern — spec, plan, and implement workflows alike — not just one. If a specific step's own instructions don't repeat this explicitly, this standing rule still applies.
